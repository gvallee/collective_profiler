//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
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
}

type PatternsSummaryData struct {
	Content string
}

const (
	sizeThreshold = 200
	binThresholds = "200,1024,2048,4096"
)

var datasetBasedir string
var datasetName string
var mainData CallsPageData

var numCalls int
var stats map[int]counts.SendRecvStats
var allPatterns map[int]patterns.Data
var allCallsData []counts.CommDataT

var basedir string

func CallHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	leadRank := 0
	callID := 0
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
	}).ParseFiles(filepath.Join(basedir, "templates", "callDetails.html"))

	err = callTemplate.Execute(w, cpd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func loadData() error {
	var err error

	if stats == nil {
		listBins := bins.GetFromInputDescr(binThresholds)
		numCalls, stats, allPatterns, allCallsData, err = profiler.HandleCountsFiles(datasetBasedir, sizeThreshold, listBins)
		if err != nil {
			return err
		}
	}

	return nil
}

func CallsLayoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	loadData()

	mainData = CallsPageData{
		PageTitle: datasetName,
		Calls:     allCallsData,
	}

	callsLayoutTemplate, err := template.New("callsLayout.html").ParseFiles(filepath.Join(basedir, "templates", "callsLayout.html"))
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

func PatternsHandler(w http.ResponseWriter, r *http.Request) {
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

	patternsTemplate, err := template.New("patterns.html").ParseFiles(filepath.Join(basedir, "templates", "patterns.html"))
	err = patternsTemplate.Execute(w, patternsSummaryData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	indexTemplate, err := template.New("index.html").ParseFiles(filepath.Join(basedir, "templates", "index.html"))
	err = indexTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func displayUI(dataBasedir string, name string) error {
	datasetBasedir = dataBasedir
	datasetName = name

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/calls", CallsLayoutHandler)
	http.HandleFunc("/patterns", PatternsHandler)
	http.HandleFunc("/call", CallHandler)
	http.ListenAndServe(":8080", nil)

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	baseDir := flag.String("basedir", "", "Base directory of the dataset")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s starts a Web-based user interface to explore a dataset", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	_, filename, _, _ := runtime.Caller(0)
	basedir = filepath.Dir(filename)
	name := "example"
	displayUI(*baseDir, name)
}
