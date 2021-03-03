package webui

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"

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
)

type CallsPageData struct {
	PageTitle string
	Calls     []counts.CommDataT
}

type CallPageData struct {
	LeadRank  int
	CallID    int
	CallsData []counts.CommDataT
	PlotPath  string
}

type PatternsSummaryData struct {
	Content string
}

// Config represents the configuration of a webUI
type Config struct {
	wg          *sync.WaitGroup
	Port        int
	codeBaseDir string
	DatasetDir  string
	Name        string
	srv         *http.Server
}

const (
	sizeThreshold = 200
	binThresholds = "200,1024,2048,4096"
	DefaultPort   = 8080
)

// The webUI is designed at the moment to support only alltoallv over a single communicator
// so we hardcode corresponding data
var collectiveName = "alltoallv"
var commID = 0

// A bunch of global variable to avoiding loading data all the time and make everything super slow
// when dealing with big datasets
var numCalls int
var stats map[int]counts.SendRecvStats
var allPatterns map[int]patterns.Data
var allCallsData []counts.CommDataT
var rankFileData map[int]*location.RankFileData
var callMaps map[int]maps.CallsDataT

// callsSendHeatMap represents the heat on a per-call basis.
// The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value a map with the key being a rank and the value its ordered counts
var callsSendHeatMap map[int]map[int]map[int]int

// callsRecvHeatMap represents the heat on a per-call basis. The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value to amount of data received
// The first key is the lead rank to identify the communicator and the value a map where the key is a callID and the value a map with the key being a rank and the value its ordered counts
var callsRecvHeatMap map[int]map[int]map[int]int

var globalSendHeatMap map[int]int
var globalRecvHeatMap map[int]int
var rankNumCallsMap map[int]int
var operationsTimings map[string]*timings.CollectiveTimings
var totalExecutionTimes map[int]float64
var totalLateArrivalTimes map[int]float64

var codeBaseDir string
var datasetBasedir string
var datasetName string
var mainData CallsPageData

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

// callHandler is the http handler invoked when details about a specific call are requested
func callHandler(w http.ResponseWriter, r *http.Request) {
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

	if callsSendHeatMap == nil {
		callsSendHeatMap = make(map[int]map[int]map[int]int)
	}
	if callsRecvHeatMap == nil {
		callsRecvHeatMap = make(map[int]map[int]map[int]int)
	}

	// Make sure the graph is ready
	if !plot.CallFilesExist(datasetBasedir, leadRank, callID) {
		if allDataAvailable(collectiveName, datasetBasedir, leadRank, commID, jobID, callID) {
			if callsSendHeatMap[leadRank] == nil {
				sendHeatMapFilename := maps.GetSendCallsHeatMapFilename(datasetBasedir, collectiveName, leadRank)
				sendHeatMap, err := maps.LoadCallsFileHeatMap(codeBaseDir, sendHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				callsSendHeatMap[leadRank] = sendHeatMap
			}

			if callsRecvHeatMap[leadRank] == nil {
				recvHeatMapFilename := maps.GetRecvCallsHeatMapFilename(datasetBasedir, collectiveName, leadRank)
				recvHeatMap, err := maps.LoadCallsFileHeatMap(codeBaseDir, recvHeatMapFilename)
				if err != nil {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				callsRecvHeatMap[leadRank] = recvHeatMap
			}

			execTimingsFile := timings.GetExecTimingFilename(collectiveName, leadRank, commID, jobID)
			_, execTimings, _, err := timings.ParseTimingFile(execTimingsFile, codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
			}
			callExecTimings := execTimings[callID]

			lateArrivalFile := timings.GetLateArrivalTimingFilename(collectiveName, leadRank, commID, jobID)
			_, lateArrivalTimings, _, err := timings.ParseTimingFile(lateArrivalFile, codeBaseDir)
			if err != nil {
				log.Printf("unable to parse %s: %s", execTimingsFile, err)
			}
			callLateArrivalTimings := lateArrivalTimings[callID]

			hostMap, err := maps.LoadHostMap(filepath.Join(datasetBasedir, maps.RankFilename))
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			pngFile, err := plot.CallData(datasetBasedir, datasetBasedir, leadRank, callID, hostMap, callsSendHeatMap[leadRank][callID], callsRecvHeatMap[leadRank][callID], callExecTimings, callLateArrivalTimings)
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if pngFile == "" {
				log.Printf("ERROR: %s\n", err)
				http.Error(w, "plot generation failed", http.StatusInternalServerError)
			}
		} else {
			if callMaps == nil {
				rankFileData, callMaps, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap, err = maps.Create(codeBaseDir, collectiveName, maps.Heat, datasetBasedir, allCallsData)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}

			if operationsTimings == nil {
				operationsTimings, totalExecutionTimes, totalLateArrivalTimes, err = timings.HandleTimingFiles(codeBaseDir, datasetBasedir, numCalls)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}

			comms, err := comm.GetData(codeBaseDir, datasetBasedir)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			for i := 0; i < len(allCallsData); i++ {
				if allCallsData[i].LeadRank == leadRank {
					opTimings := operationsTimings[collectiveName]
					opExecTimings := opTimings.ExecTimes
					opLateArrivalTimings := opTimings.LateArrivalTimes

					for leadRank, listComms := range comms.LeadMap {
						for _, c := range listComms {
							id := timings.CommT{
								CommID:   c,
								LeadRank: leadRank,
							}

							_, err = plot.CallData(datasetBasedir, datasetBasedir, leadRank, callID, rankFileData[leadRank].HostMap, callMaps[leadRank].SendHeatMap[i], callMaps[leadRank].RecvHeatMap[i], opExecTimings[id][i], opLateArrivalTimings[id][i])
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
							}
						}
					}
					break
				}
			}
		}
	}

	cpd := CallPageData{
		LeadRank:  leadRank,
		CallID:    callID,
		CallsData: mainData.Calls,
	}

	callTemplate, err := template.New("callDetails.html").Funcs(template.FuncMap{
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
		},
	}).ParseFiles(filepath.Join(codeBaseDir, "tools", "cmd", "webui", "templates", "callDetails.html"))

	err = callTemplate.Execute(w, cpd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func loadData() error {
	if stats == nil {
		_, sendCountsFiles, _, err := profiler.FindCompactFormatCountsFiles(datasetBasedir)
		if err != nil {
			return err
		}
		if len(sendCountsFiles) == 0 {
			// We do not have the files in the right format: try to convert raw counts files
			fileInfo := profiler.FindRawCountFiles(datasetBasedir)
			err := counts.LoadRawCountsFromDirs(fileInfo.Dirs, datasetBasedir)
			if err != nil {
				return err
			}
		}

		listBins := bins.GetFromInputDescr(binThresholds)
		numCalls, stats, allPatterns, allCallsData, err = profiler.HandleCountsFiles(datasetBasedir, sizeThreshold, listBins)
		if err != nil {
			return err
		}
	}

	return nil
}

func callsLayoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	loadData()

	mainData = CallsPageData{
		PageTitle: datasetName,
		Calls:     allCallsData,
	}

	callsLayoutTemplate, err := template.New("callsLayout.html").ParseFiles(filepath.Join(codeBaseDir, "tools", "cmd", "webui", "templates", "callsLayout.html"))
	err = callsLayoutTemplate.Execute(w, mainData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func findPatternsSummaryFile() (string, error) {
	files, err := ioutil.ReadDir(datasetBasedir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), patterns.SummaryFilePrefix) {
			return filepath.Join(datasetBasedir, file.Name()), nil
		}
	}

	return "", nil
}

func patternsHandler(w http.ResponseWriter, r *http.Request) {
	// check if the summary file is already there; if not, generate it.

	patternsFilePath, err := findPatternsSummaryFile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if patternsFilePath == "" {
		// The summary pattern file does not exist
		loadData()
		err = profiler.AnalyzeSubCommsResults(datasetBasedir, stats, allPatterns)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	patternsFilePath, err = findPatternsSummaryFile()
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

	patternsSummaryData := PatternsSummaryData{
		Content: htmlContent,
	}

	patternsTemplate, err := template.New("patterns.html").ParseFiles(filepath.Join(codeBaseDir, "tools", "cmd", "webui", "templates", "patterns.html"))
	err = patternsTemplate.Execute(w, patternsSummaryData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexTemplate, err := template.New("index.html").ParseFiles(filepath.Join(codeBaseDir, "tools", "cmd", "webui", "templates", "index.html"))
	err = indexTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	indexTemplate, err := template.New("bye.html").ParseFiles(filepath.Join(codeBaseDir, "tools", "cmd", "webui", "templates", "bye.html"))
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
	cfg.wg = &sync.WaitGroup{}
	cfg.wg.Add(1)
	cfg.Port = DefaultPort
	_, filename, _, _ := runtime.Caller(0)
	cfg.codeBaseDir = filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return cfg
}

// Start instantiates a HTTP server and start the webUI. This is a non-blocking function,
// meaning the function returns after successfully initiating the WebUI. To wait for the
// termination of the webUI, please use Wait()
func (c *Config) Start() error {
	datasetBasedir = c.DatasetDir
	datasetName = c.Name

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/calls", callsLayoutHandler)
	mux.HandleFunc("/patterns", patternsHandler)
	mux.HandleFunc("/call", callHandler)
	mux.HandleFunc("/stop", c.stopHandler)
	mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(datasetBasedir))))

	c.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: mux,
	}

	go func() {
		defer c.wg.Done()
		c.srv.ListenAndServe()
		fmt.Println("HTTP server is now terminated")
	}()

	return nil
}

// Wait makes the current process wait for the termination of the webUI
func (c *Config) Wait() {
	c.wg.Wait()
}
