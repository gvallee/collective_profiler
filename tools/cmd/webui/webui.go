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

var datasetBasedir string
var datasetName string
var mainData CallsPageData

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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sizeThreshold := 200
	binThresholds := "200,1024,2048,4096"
	listBins := bins.GetFromInputDescr(binThresholds)
	_, _, _, allCallsData, err := profiler.HandleCountsFiles(datasetBasedir, sizeThreshold, listBins)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	mainData = CallsPageData{
		PageTitle: datasetName,
		Calls:     allCallsData,
	}

	indexTemplate, err := template.New("callsLayout.html").ParseFiles(filepath.Join(basedir, "templates", "callsLayout.html"))
	err = indexTemplate.Execute(w, mainData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func displayUI(dataBasedir string, name string) error {
	datasetBasedir = dataBasedir
	datasetName = name
	_, filename, _, _ := runtime.Caller(0)
	basedir = filepath.Dir(filename)

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/call", CallHandler)
	http.ListenAndServe(":8080", nil)

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
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

	basedir := "/home/gvallee/projects/alltoall_profiling/examples"
	name := "example"
	displayUI(basedir, name)
}
