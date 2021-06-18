//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/comm"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/maps"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/plot"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/go_util/pkg/util"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

type callsPageData struct {
	PageTitle string
	Calls     []counts.CommDataT
}

type callPageData struct {
	LeadRank  int
	CallID    int
	CallsData []counts.CommDataT
	PlotPath  string
}

type jsonPageData struct {
	LeadRank  int
	CallID    int
	Kth       int
	Dis       string
	HeatMap   map[int][][]int
	CallsData []counts.CommDataT
}

type patternsSummaryData struct {
	Content string
}

type HeapMapData struct {
	Content string
	Calls   []counts.CommDataT
}

type heapPageData struct {
	LeadRank  int
	CallID    int
	CallsData []counts.CommDataT
	PlotPath  string
}

type HeatMapData struct {
	PlotPath string
}

type server struct {
	mux              *http.ServeMux
	cfg              *Config
	indexTemplate    *template.Template
	callsTemplate    *template.Template
	callTemplate     *template.Template
	heatmapTemplate  *template.Template
	heatTemplate     *template.Template
	patternsTemplate *template.Template
	stopTemplate     *template.Template
}

// Config represents the configuration of a webUI
type Config struct {
	wg          *sync.WaitGroup
	Port        int
	codeBaseDir string
	DatasetDir  string
	Name        string
	mux         *http.ServeMux
	srv         *http.Server

	// The webUI is designed at the moment to support only alltoallv over a single communicator
	// so we hardcode corresponding data
	collectiveName string
	commID         int
	numCalls       int
	stats          map[int]counts.SendRecvStats
	allPatterns    map[int]patterns.Data
	allCallsData   []counts.CommDataT
	rankFileData   map[int]*location.RankFileData
	callMaps       map[int]maps.CallsDataT

	// callsSendHeatMap represents the heat on a per-call basis.
	// The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value a map with the key being a rank and the value its ordered counts
	callsSendHeatMap map[int]map[int]map[int]int

	// callsRecvHeatMap represents the heat on a per-call basis. The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value to amount of data received
	// The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value a map with the key being a rank and the value its ordered counts
	callsRecvHeatMap map[int]map[int]map[int]int

	// save the heatmap matrix ["callnum"][i][j] = rank i--send-->rank j / rank i--recv-->rank j
	jsonSendHeetMap map[int][][]int
	jsonRecvHeetMap map[int][][]int

	globalSendHeatMap     map[int]int
	globalRecvHeatMap     map[int]int
	rankNumCallsMap       map[int]int
	operationsTimings     map[string]*timings.CollectiveTimings
	totalExecutionTimes   map[int]float64
	totalLateArrivalTimes map[int]float64

	mainData callsPageData
	heapData HeatMapData
	cpd      callPageData
	jpd      jsonPageData
	psd      patternsSummaryData
	hsd      heapPageData

	indexTemplatePath    string
	callsTemplatePath    string
	patternsTemplatePath string
	callTemplatePath     string
	stopTemplatePath     string
	heatmapTemplatePath  string
	heatTemplatePath     string
}

const (
	sizeThreshold = 200
	binThresholds = "200,1024,2048,4096"

	// DefaultPort is the default port used to start the webui
	DefaultPort = 8080
)

func allDataAvailable(collectiveName string, dir string, leadRank int, commID int, jobID int, callID int) bool {
	callSendHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-send.call%d.txt", maps.CallHeatMapPrefix, leadRank, callID))
	callRecvHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-recv.call%d.txt", maps.CallHeatMapPrefix, leadRank, callID))

	if !util.PathExists(callSendHeatMapFilePath) {
		log.Printf("%s is missing!\n", callSendHeatMapFilePath)
		return false
	}

	if !util.PathExists(callRecvHeatMapFilePath) {
		log.Printf("%s is missing!\n", callRecvHeatMapFilePath)
		return false
	}

	lateArrivalTimingFilePath := filepath.Join(dir, timings.GetExecTimingFilename(collectiveName, leadRank, commID, jobID))
	execTimingFilePath := filepath.Join(dir, timings.GetLateArrivalTimingFilename(collectiveName, leadRank, commID, jobID))

	if !util.PathExists(execTimingFilePath) {
		log.Printf("%s is missing!\n", execTimingFilePath)
		return false
	}

	if !util.PathExists(lateArrivalTimingFilePath) {
		log.Printf("%s is missing!\n", lateArrivalTimingFilePath)
		return false
	}

	hostMapFilePath := filepath.Join(dir, maps.RankFilename)

	if !util.PathExists(hostMapFilePath) {
		log.Printf("%s is missing!\n", hostMapFilePath)
		return false
	}

	log.Printf("All files for call %d.%d are present!!!", leadRank, callID)

	return true
}

func (c *Config) getTemplateFilePath(name string) string {
	return filepath.Join(c.codeBaseDir, "tools", "internal", "pkg", "webui", "templates", name+".html")
}

func (c *Config) serviceCallDetailsRequest(w http.ResponseWriter, r *http.Request) {
	var err error

	leadRank := 0
	callID := 0
	jobID := 0
	params := r.URL.Query()
	for k, v := range params {
		if k == "leadRank" {
			leadRank, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		if k == "callID" {
			callID, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		if k == "jobID" {
			jobID, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}

	if c.callsSendHeatMap == nil {
		c.callsSendHeatMap = make(map[int]map[int]map[int]int)
	}
	if c.callsRecvHeatMap == nil {
		c.callsRecvHeatMap = make(map[int]map[int]map[int]int)
	}

	// Make sure the graph is ready
	if !plot.CallFilesExist(c.DatasetDir, leadRank, callID) {
		fmt.Print(c.DatasetDir)
		if allDataAvailable(c.collectiveName, c.DatasetDir, leadRank, c.commID, jobID, callID) {
			if c.callsSendHeatMap[leadRank] == nil {
				sendHeatMapFilename := maps.GetSendCallsHeatMapFilename(c.DatasetDir, c.collectiveName, leadRank)
				sendHeatMap, err := maps.LoadCallsFileHeatMap(c.codeBaseDir, sendHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				c.callsSendHeatMap[leadRank] = sendHeatMap
			}

			if c.callsRecvHeatMap[leadRank] == nil {
				recvHeatMapFilename := maps.GetRecvCallsHeatMapFilename(c.DatasetDir, c.collectiveName, leadRank)
				recvHeatMap, err := maps.LoadCallsFileHeatMap(c.codeBaseDir, recvHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				c.callsRecvHeatMap[leadRank] = recvHeatMap
			}

			execTimingsFile := filepath.Join(c.DatasetDir, timings.GetExecTimingFilename(c.collectiveName, leadRank, c.commID, jobID))
			_, execTimings, _, err := timings.ParseTimingFile(execTimingsFile, c.codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
			}
			callExecTimings := execTimings[callID]

			lateArrivalFile := filepath.Join(c.DatasetDir, timings.GetLateArrivalTimingFilename(c.collectiveName, leadRank, c.commID, jobID))
			_, lateArrivalTimings, _, err := timings.ParseTimingFile(lateArrivalFile, c.codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
			}
			callLateArrivalTimings := lateArrivalTimings[callID]

			hostMap, err := maps.LoadHostMap(filepath.Join(c.DatasetDir, maps.RankFilename))
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			// Debug
			fmt.Print(c.callsSendHeatMap[leadRank][callID], c.callsSendHeatMap[leadRank][callID])

			//pngFile, err := plot.CallData(c.DatasetDir, c.DatasetDir, leadRank, callID, hostMap, c.callsSendHeatMap[leadRank][callID], c.callsSendHeatMap[leadRank][callID], callExecTimings, callLateArrivalTimings)
			pngFile, err := plot.CallData(c.DatasetDir, c.DatasetDir, leadRank, callID, hostMap, c.callsSendHeatMap[leadRank][0], c.callsSendHeatMap[leadRank][0], callExecTimings, callLateArrivalTimings)
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if pngFile == "" {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, "plot generation failed", http.StatusInternalServerError)
			}
		} else {
			if c.callMaps == nil {
				c.rankFileData, c.callMaps, c.globalSendHeatMap, c.globalRecvHeatMap, c.rankNumCallsMap, err = maps.Create(c.codeBaseDir, c.collectiveName, maps.Heat, c.DatasetDir, c.allCallsData)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}

			if c.operationsTimings == nil {
				log.Println("Loading timing data...")
				c.operationsTimings, c.totalExecutionTimes, c.totalLateArrivalTimes, err = timings.HandleTimingFiles(c.codeBaseDir, c.DatasetDir, c.numCalls)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}

			comms, err := comm.GetData(c.codeBaseDir, c.DatasetDir)
			if err != nil {
				log.Printf("comm.GetData() failed: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if comms == nil {
				err = fmt.Errorf("undefined list of communicators")
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			for leadRank, listComms := range comms.LeadMap {
				if listComms == nil {
					err := fmt.Errorf("listComms is nil")
					log.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				for _, commID := range listComms {
					id := timings.CommT{
						CommID:   commID,
						LeadRank: leadRank,
					}
					// The call we are looking for may not be on that communicator
					if c.operationsTimings[c.collectiveName].ExecTimes[id][callID] != nil {
						_, err = plot.CallData(c.DatasetDir, c.DatasetDir, leadRank, callID, c.rankFileData[leadRank].HostMap, c.callMaps[leadRank].SendHeatMap[callID], c.callMaps[leadRank].RecvHeatMap[callID], c.operationsTimings[c.collectiveName].ExecTimes[id][callID], c.operationsTimings[c.collectiveName].LateArrivalTimes[id][callID])
						if err != nil {
							err = fmt.Errorf("plot.CallData() failed for call %d on comm %d: %s", callID, leadRank, err)
							log.Println(err)
							http.Error(w, err.Error(), http.StatusInternalServerError)
						}
					}
				}
			}
		}
	}
	fmt.Print(c.callsSendHeatMap)
	c.cpd = callPageData{
		LeadRank:  leadRank,
		CallID:    callID,
		CallsData: c.mainData.Calls,
	}
}

func (c *Config) serviceHeatDetailsRequest(w http.ResponseWriter, r *http.Request) {
	var err error

	leadRank := 0
	//callID := 0
	//Dis := "shanghaitech"
	//Kth := 0
	params := r.URL.Query()
	for k, v := range params {
		if k == "leadRank" {
			leadRank, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		//
		//if k == "callID" {
		//	callID, err = strconv.Atoi(v[0])
		//	if err != nil {
		//		http.Error(w, err.Error(), http.StatusInternalServerError)
		//	}
		//}
		//if k == "Kth" {
		//	Kth, err = strconv.Atoi(v[0])
		//	if err != nil {
		//		http.Error(w, err.Error(), http.StatusInternalServerError)
		//	}
		//}
	}

	// save the heatmap matrix ["callnum"][i][j] = rank i--send-->rank j / rank i--recv-->rank j

	if c.jsonSendHeetMap == nil {
		c.jsonSendHeetMap = make(map[int][][]int)
	}

	pwd := c.DatasetDir //"examples/result_task2_wrf_run-at-20210608-150432/result_task2_wrf_run-at-20210608-150432"
	listFiles, err := findCountRankFileList(pwd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(listFiles)

	type topMatrix struct {
		CallID int     `json:"call_id"`
		Sum    int     `json:"sum"`
		Matrix [][]int `json:"matrix"`
	}

	topMatrixArray := make([]topMatrix, 0)

	for _, filePath := range listFiles {
		if strings.HasPrefix(filepath.Base(filePath), fmt.Sprintf("counts.rank%d", leadRank)) {
			callIDString := strings.TrimPrefix(filepath.Base(filePath), fmt.Sprintf("counts.rank%d_call", leadRank))
			callIDString = strings.TrimSuffix(callIDString, ".md")
			callIDCurrent, err := strconv.Atoi(callIDString)
			matrixCurrentSum := 0
			if err != nil {
				panic(err)
			}

			rawCountsT, _ := counts.ParseRawFile(filePath)
			// Warning: This is easy to cause crash!!!
			matrixSize := len(strings.Split(rawCountsT.SendCounts[0], " "))
			currentMatrix := make([][]int, matrixSize)
			for i := 0; i < matrixSize; i++ {
				currentMatrix[i] = make([]int, matrixSize)
				currentMatrixStr := strings.Split(rawCountsT.SendCounts[i], " ")
				for j := 0; j < matrixSize; j++ {
					currentMatrix[i][j], err = strconv.Atoi(currentMatrixStr[j])
					if err != nil {
						panic(err)
					}
					matrixCurrentSum += currentMatrix[i][j]
				}
			}
			topMatrixArray = append(topMatrixArray, topMatrix{Matrix: currentMatrix, Sum: matrixCurrentSum, CallID: callIDCurrent})
		}
	}
	sort.SliceStable(topMatrixArray, func(i, j int) bool {
		return topMatrixArray[i].Sum > topMatrixArray[j].Sum
	})

	topMatrixArray = topMatrixArray[:10]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(topMatrixArray)

	//c.jpd = jsonPageData{
	//	LeadRank:  leadRank,
	//	CallID:    callID,
	//	Kth:       Kth,
	//	Dis:	   Dis,
	//	HeatMap:   jsonSendHeetMap,
	//	CallsData: c.mainData.Calls,
	//}
}

func findCountRankFileList(pwd string) ([]string, error) {
	fileInfoList, err := ioutil.ReadDir(pwd)

	if err != nil {
		log.Fatal(err)
	}
	var fileList []string
	for _, file := range fileInfoList {
		if strings.HasPrefix(file.Name(), patterns.CountFilePrefix) {
			fileList = append(fileList, filepath.Join(pwd, file.Name()))
		}
	}
	return fileList, nil
}

func (c *Config) loadData() error {
	if c.stats == nil {
		if c.DatasetDir == "" {
			return fmt.Errorf("c.DatasetDir is undefined")
		}

		if !util.PathExists(c.DatasetDir) {
			return fmt.Errorf("datasetBasedir %s does not exit", c.DatasetDir)
		}

		_, sendCountsFiles, _, err := profiler.FindCompactFormatCountsFiles(c.DatasetDir)
		if err != nil {
			return err
		}
		if len(sendCountsFiles) == 0 {
			// We do not have the files in the right format: try to convert raw counts files
			fileInfo := profiler.FindRawCountFiles(c.DatasetDir)
			err := counts.LoadRawCountsFromDirs(fileInfo.Dirs, c.DatasetDir)
			if err != nil {
				return err
			}
		}

		listBins := bins.GetFromInputDescr(binThresholds)
		c.numCalls, c.stats, c.allPatterns, c.allCallsData, err = profiler.HandleCountsFiles(c.DatasetDir, sizeThreshold, listBins)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) serviceCallsRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	err := c.loadData()
	if err != nil {
		fmt.Printf("unable to load data: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	c.mainData = callsPageData{
		PageTitle: c.Name,
		Calls:     c.allCallsData,
	}
}

func (c *Config) serviceHeatmapRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	err := c.loadData()
	if err != nil {
		fmt.Printf("unable to load data: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	c.mainData = callsPageData{
		PageTitle: c.Name,
		Calls:     c.allCallsData,
	}
}

func findPatternsSummaryFile(c *Config) (string, error) {
	files, err := ioutil.ReadDir(c.DatasetDir)
	if err != nil {
		return "", err
	}
	// where we get all the data we need and put it into c
	for _, file := range files {
		if strings.HasPrefix(file.Name(), patterns.SummaryFilePrefix) {
			return filepath.Join(c.DatasetDir, file.Name()), nil
		}
	}

	return "", nil
}

func (c *Config) servicePatternRequest(w http.ResponseWriter, r *http.Request) {

	patternsFilePath, err := findPatternsSummaryFile(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if patternsFilePath == "" {
		// The summary pattern file does not exist
		err = c.loadData()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		err = profiler.AnalyzeSubCommsResults(c.DatasetDir, c.stats, c.allPatterns)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	patternsFilePath, err = findPatternsSummaryFile(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if patternsFilePath == "" {
		http.Error(w, "unable to load patterns", http.StatusInternalServerError)
	}

	mdContent, err := ioutil.ReadFile(patternsFilePath)
	if err != nil {
		http.Error(w, "unable to load patterns", http.StatusInternalServerError)
	}
	htmlContent := string(markdown.ToHTML(mdContent, nil, nil))

	c.psd = patternsSummaryData{
		Content: htmlContent,
	}
}

// Stop cleanly terminates a running webUI
func (c *Config) Stop() error {
	err := c.srv.Shutdown(context.TODO())
	if err != nil {
		return err
	}
	c.wg.Wait()
	return nil
}

func (c *Config) stopHandler(w http.ResponseWriter, r *http.Request) {
	templatePath := c.getTemplateFilePath("bye")
	indexTemplate, err := template.New("bye.html").ParseFiles(templatePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = indexTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/text")
	err = c.srv.Shutdown(context.TODO())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RemoteStop remotely terminates a WebUI by sending a termination request
func RemoteStop(host string, port string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://"+host+":"+port+"/stop", nil)
	if err != nil {
		return err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/text")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	bs := string(body)
	fmt.Printf("checkme: %s\n", bs)

	return nil
}

// Init creates a configuration for the webui that can then be used to start/stop a webui
func Init() *Config {
	cfg := new(Config)
	cfg.wg = new(sync.WaitGroup)
	cfg.wg.Add(1)
	cfg.Port = DefaultPort
	cfg.collectiveName = "alltoallv"
	cfg.commID = 0
	_, filename, _, _ := runtime.Caller(0)
	cfg.codeBaseDir = filepath.Join(filepath.Dir(filename), "..", "..", "..", "..")

	cfg.indexTemplatePath = cfg.getTemplateFilePath("index")
	cfg.callsTemplatePath = cfg.getTemplateFilePath("callsLayout")
	cfg.callTemplatePath = cfg.getTemplateFilePath("callDetails")
	cfg.heatmapTemplatePath = cfg.getTemplateFilePath("heatmapLayout")
	cfg.heatTemplatePath = cfg.getTemplateFilePath("heatDetails")
	cfg.stopTemplatePath = cfg.getTemplateFilePath("bye")
	cfg.patternsTemplatePath = cfg.getTemplateFilePath("patterns")
	return cfg
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method, r.URL.String())
	s.mux.ServeHTTP(w, r)
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	s.indexTemplate.Execute(w, s.cfg)
}

func (s *server) calls(w http.ResponseWriter, r *http.Request) {
	s.cfg.serviceCallsRequest(w, r)
	s.callsTemplate.Execute(w, s.cfg.mainData /*s.cfg*/)
}

func (s *server) heatmap(w http.ResponseWriter, r *http.Request) {
	s.cfg.serviceHeatmapRequest(w, r)
	s.heatmapTemplate.Execute(w, s.cfg.mainData /*s.cfg*/)
}

func (s *server) heat(w http.ResponseWriter, r *http.Request) {
	s.cfg.serviceCallDetailsRequest(w, r)
	s.callsTemplate.Execute(w, s.cfg.cpd /*s.cfg*/)
}

func (s *server) call(w http.ResponseWriter, r *http.Request) {
	s.cfg.serviceCallDetailsRequest(w, r)
	s.callTemplate.Execute(w, s.cfg.cpd)
}

func (s *server) patterns(w http.ResponseWriter, r *http.Request) {
	s.cfg.servicePatternRequest(w, r)
	s.patternsTemplate.Execute(w, s.cfg.psd /*s.cfg*/)
}

func (s *server) stop(w http.ResponseWriter, r *http.Request) {
	s.stopTemplate.Execute(w, s.cfg)
}

func newServer(cfg *Config) *server {
	s := &server{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
	s.mux.HandleFunc("/", s.index)
	s.mux.HandleFunc("/calls", s.calls)
	s.mux.HandleFunc("/call", s.call)
	s.mux.HandleFunc("/patterns", s.patterns)
	s.mux.HandleFunc("/stop", s.stop)
	s.mux.HandleFunc("/heatmap", s.heatmap)
	s.mux.HandleFunc("/he", cfg.serviceHeatDetailsRequest)
	//serviceHeatDetailsRequest
	//s.mux.HandleFunc("/call_json", s.call_json)
	s.mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(s.cfg.DatasetDir))))
	return s
}

// Start instantiates a HTTP server and start the webUI. This is a non-blocking function,
// meaning the function returns after successfully initiating the WebUI. To wait for the
// termination of the webUI, please use Wait()
func (c *Config) Start() error {
	//c.mux = http.NewServeMux()
	s := newServer(c)
	s.indexTemplate = template.Must(template.ParseFiles(c.indexTemplatePath))
	s.callTemplate = template.Must(template.New("callDetails.html").Funcs(template.FuncMap{
		"displaySendCounts": func(cd []counts.CommDataT, leadRank int, callID int) string {
			for _, data := range cd {
				if data.LeadRank == leadRank {
					return strings.Join(cd[leadRank].CallData[callID].SendData.RawCounts, "<br />")
				}
			}
			return "Call not found"
		},
		"displayRecvCounts": func(cd []counts.CommDataT, leadRank int, callID int) string {
			for _, data := range cd {
				if data.LeadRank == leadRank {
					return strings.Join(cd[leadRank].CallData[callID].RecvData.RawCounts, "<br />")
				}
			}
			return "Call not found"
		},
		"displayCallPlot": func(leadRank int, callID int) string {
			return fmt.Sprintf("profiler_rank%d_call%d.png", leadRank, callID)
		}}).ParseFiles(c.callTemplatePath))
	s.callsTemplate = template.Must(template.ParseFiles(c.callsTemplatePath))
	s.heatmapTemplate = template.Must(template.ParseFiles(c.heatmapTemplatePath))
	s.patternsTemplate = template.Must(template.ParseFiles(c.patternsTemplatePath))
	s.stopTemplate = template.Must(template.ParseFiles(c.stopTemplatePath))

	c.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: s,
	}

	go func(c *Config) {
		defer c.wg.Done()
		c.srv.ListenAndServe()
		fmt.Println("HTTP server is now terminated")
	}(c)

	return nil
}

// Wait makes the current process wait for the termination of the webUI
func (c *Config) Wait() {
	c.wg.Wait()
}
