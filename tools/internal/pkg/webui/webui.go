//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package webui

import (
	"context"
	"fmt"
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

	"github.com/gomarkdown/markdown"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/comm"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/maps"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
	internalplotter "github.com/gvallee/alltoallv_profiling/tools/internal/pkg/plot"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/plugins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/counts"
	"github.com/gvallee/go_util/pkg/util"
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

type patternsSummaryData struct {
	Content string
}

type imbalanceData struct {
	Enabled  bool
	CommData map[int]map[int][]string
}

type server struct {
	mux               *http.ServeMux
	cfg               *Config
	indexTemplate     *template.Template
	callsTemplate     *template.Template
	callTemplate      *template.Template
	patternsTemplate  *template.Template
	stopTemplate      *template.Template
	imbalanceTemplate *template.Template
}

// Config represents the configuration of a webUI
type Config struct {
	wg          *sync.WaitGroup
	Port        int
	codeBaseDir string
	DatasetDir  string
	Name        string
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

	globalSendHeatMap     map[int]int
	globalRecvHeatMap     map[int]int
	rankNumCallsMap       map[int]int
	operationsTimings     map[string]*timings.CollectiveTimings
	totalExecutionTimes   map[int]float64
	totalLateArrivalTimes map[int]float64

	mainData callsPageData
	cpd      callPageData
	psd      patternsSummaryData
	imbdata  imbalanceData

	indexTemplatePath     string
	callsTemplatePath     string
	patternsTemplatePath  string
	callTemplatePath      string
	stopTemplatePath      string
	imbalanceTemplatePath string

	Plugins plugins.Plugins
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

	lateArrivalTimingFilePath := timings.GetExecTimingFilename(collectiveName, leadRank, commID, jobID)
	execTimingFilePath := timings.GetLateArrivalTimingFilename(collectiveName, leadRank, commID, jobID)

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
				return
			}
		}

		if k == "callID" {
			callID, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if k == "jobID" {
			jobID, err = strconv.Atoi(v[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
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
	if !internalplotter.CallFilesExist(c.DatasetDir, leadRank, callID) {
		if allDataAvailable(c.collectiveName, c.DatasetDir, leadRank, c.commID, jobID, callID) {
			if c.callsSendHeatMap[leadRank] == nil {
				sendHeatMapFilename := maps.GetSendCallsHeatMapFilename(c.DatasetDir, c.collectiveName, leadRank)
				sendHeatMap, err := maps.LoadCallsFileHeatMap(c.codeBaseDir, sendHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				c.callsSendHeatMap[leadRank] = sendHeatMap
			}

			if c.callsRecvHeatMap[leadRank] == nil {
				recvHeatMapFilename := maps.GetRecvCallsHeatMapFilename(c.DatasetDir, c.collectiveName, leadRank)
				recvHeatMap, err := maps.LoadCallsFileHeatMap(c.codeBaseDir, recvHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				c.callsRecvHeatMap[leadRank] = recvHeatMap
			}

			execTimingsFile := timings.GetExecTimingFilename(c.collectiveName, leadRank, c.commID, jobID)
			_, execTimings, _, err := timings.ParseTimingFile(execTimingsFile, c.codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			callExecTimings := execTimings[callID]

			lateArrivalFile := timings.GetLateArrivalTimingFilename(c.collectiveName, leadRank, c.commID, jobID)
			_, lateArrivalTimings, _, err := timings.ParseTimingFile(lateArrivalFile, c.codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			callLateArrivalTimings := lateArrivalTimings[callID]

			hostMap, err := maps.LoadHostMap(filepath.Join(c.DatasetDir, maps.RankFilename))
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			pngFile, err := internalplotter.CallData(c.DatasetDir, c.DatasetDir, leadRank, callID, hostMap, c.callsSendHeatMap[leadRank][callID], c.callsRecvHeatMap[leadRank][callID], callExecTimings, callLateArrivalTimings)
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if pngFile == "" {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, "plot generation failed", http.StatusInternalServerError)
				return
			}
		} else {
			if c.callMaps == nil {
				c.rankFileData, c.callMaps, c.globalSendHeatMap, c.globalRecvHeatMap, c.rankNumCallsMap, err = maps.Create(c.codeBaseDir, c.collectiveName, maps.Heat, c.DatasetDir, c.allCallsData)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			if c.operationsTimings == nil {
				log.Println("Loading timing data...")
				c.operationsTimings, c.totalExecutionTimes, c.totalLateArrivalTimes, err = timings.HandleTimingFiles(c.codeBaseDir, c.DatasetDir, c.numCalls)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			comms, err := comm.GetData(c.codeBaseDir, c.DatasetDir)
			if err != nil {
				log.Printf("comm.GetData() failed: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if comms == nil {
				err = fmt.Errorf("undefined list of communicators")
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if comms.LeadMap == nil {
				err = fmt.Errorf("undefined LeadMap")
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for leadRank, listComms := range comms.LeadMap {
				if listComms == nil {
					err := fmt.Errorf("listComms is nil")
					log.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				for _, commID := range listComms {
					id := timings.CommT{
						CommID:   commID,
						LeadRank: leadRank,
					}
					// The call we are looking for may not be on that communicator
					if c.operationsTimings[c.collectiveName].ExecTimes[id][callID] != nil {
						_, err = internalplotter.CallData(c.DatasetDir, c.DatasetDir, leadRank, callID, c.rankFileData[leadRank].HostMap, c.callMaps[leadRank].SendHeatMap[callID], c.callMaps[leadRank].RecvHeatMap[callID], c.operationsTimings[c.collectiveName].ExecTimes[id][callID], c.operationsTimings[c.collectiveName].LateArrivalTimes[id][callID])
						if err != nil {
							err = fmt.Errorf("plot.CallData() failed for call %d on comm %d: %s", callID, leadRank, err)
							log.Println(err)
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					}
				}
			}
		}
	}

	c.cpd = callPageData{
		LeadRank:  leadRank,
		CallID:    callID,
		CallsData: c.mainData.Calls,
	}
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
			return fmt.Errorf("unable to find send count file(s) in %s", c.DatasetDir)
		}

		listBins := bins.GetFromInputDescr(binThresholds)
		data, err := profiler.HandleCountsFiles(c.DatasetDir, sizeThreshold, listBins)
		if err != nil {
			return err
		}

		c.numCalls = data.TotalNumCalls
		c.stats = data.AllStats
		c.allPatterns = data.AllPatterns
		c.allCallsData = data.AllCallsData
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

func findPatternsSummaryFile(c *Config) (string, error) {
	files, err := ioutil.ReadDir(c.DatasetDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), patterns.SummaryFilePrefix) {
			return filepath.Join(c.DatasetDir, file.Name()), nil
		}
	}

	return "", nil
}

func findImbalanceFiles(c *Config) ([]string, error) {
	files, err := ioutil.ReadDir(c.DatasetDir)
	if err != nil {
		return nil, err
	}

	var imbalanceFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "imbalance-") {
			imbalanceFiles = append(imbalanceFiles, file.Name())
		}
	}

	return imbalanceFiles, nil
}

func parseImbalanceEntry(line string) (int, []string, error) {
	tokens := strings.Split(line, ": ")
	if len(tokens) != 2 {
		return -1, nil, fmt.Errorf("unable to parse line")
	}
	ratioStr := tokens[0]
	listCallsStr := tokens[1]

	ratioStr = strings.TrimPrefix(ratioStr, "ratio ")
	ratio, err := strconv.Atoi(ratioStr)
	if err != nil {
		return -1, nil, err
	}

	listCallsStr = strings.TrimPrefix(listCallsStr, "[")
	listCallsStr = strings.TrimSuffix(listCallsStr, "]")
	calls := strings.Split(listCallsStr, " ")

	return ratio, calls, nil
}

func (c *Config) serviceImbalanceRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var imbData map[int]map[int][]string
	enabled := false
	if c.Plugins.ImbalanceDetect != nil {
		enabled = true
		imbalanceFiles, err := findImbalanceFiles(c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		imbData = make(map[int]map[int][]string)

		for _, f := range imbalanceFiles {
			comm := strings.TrimPrefix(f, "imbalance-comm")
			comm = strings.TrimSuffix(comm, ".txt")
			commID, err := strconv.Atoi(comm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			content, err := ioutil.ReadFile(filepath.Join(c.DatasetDir, f))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}

				ratio, calls, err := parseImbalanceEntry(line)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				if imbData[commID] == nil {
					imbData[commID] = make(map[int][]string)
				}
				imbData[commID][ratio] = calls
				/*
					if len(calls) == 1 {
						filename := fmt.Sprintf("imbalance-comm%d-call%s", commID, calls[0])
						label := "Arrival timings call #%d" + calls[0]
						var x []float64
						var y []float64

						c.PlotImbalanceCallGraph(filename, label, x, y)
					}
				*/
			}
		}
	}

	c.imbdata = imbalanceData{
		Enabled:  enabled,
		CommData: imbData,
	}
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

/*
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
*/

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
	cfg.stopTemplatePath = cfg.getTemplateFilePath("bye")
	cfg.patternsTemplatePath = cfg.getTemplateFilePath("patterns")
	cfg.imbalanceTemplatePath = cfg.getTemplateFilePath("imbalance")

	plugins.Load(cfg.codeBaseDir, &cfg.Plugins)

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

func (s *server) imbalance(w http.ResponseWriter, r *http.Request) {
	s.cfg.serviceImbalanceRequest(w, r)
	s.imbalanceTemplate.Execute(w, s.cfg.imbdata)
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
	s.mux.HandleFunc("/imbalance", s.imbalance)
	s.mux.HandleFunc("/stop", s.stop)
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
	s.patternsTemplate = template.Must(template.ParseFiles(c.patternsTemplatePath))
	s.stopTemplate = template.Must(template.ParseFiles(c.stopTemplatePath))
	s.imbalanceTemplate = template.Must(template.New("imbalance.html").Funcs(template.FuncMap{
		"displayImbalance": func(enabled bool, data map[int]map[int][]string) string {
			if !enabled {
				return "Imbalance plugin not found"
			}

			if data == nil {
				return "Data is undefined"
			}

			content := "<h1>Imbalance detection based on late arrival timings</h1><p>The imbalance ratio is defined as the ratio between the time of the first rank entering the collective operation and the time of the last rank entering the collective operation.<p>"
			var comms []int
			for commID := range data {
				comms = append(comms, commID)
			}
			sort.Ints(comms)
			for _, comm := range comms {
				content += fmt.Sprintf("<h2>Communicator %d</h2>", comm)
				commData := data[comm]
				var allRatio []int
				for ratio := range commData {
					allRatio = append(allRatio, ratio)
				}
				sort.Ints(allRatio)
				maxRatio := allRatio[len(allRatio)-1]
				content += fmt.Sprintf("<p> Strongest imbalance (ratio %d) for calls %s</p></br>", maxRatio, data[comm][maxRatio])
			}
			return content
		}}).ParseFiles(c.imbalanceTemplatePath))

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
