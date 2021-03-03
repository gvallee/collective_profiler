//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/maps"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/webui"
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

const (
	sizeThreshold = 200
	binThresholds = "200,1024,2048,4096"
)

var srv *http.Server

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

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	basedir := flag.String("basedir", "", "Base directory of the dataset")
	name := flag.String("name", "example", "Name of the dataset to display")
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

	fmt.Println("Calling displayUI...")

	cfg := webui.Init()
	cfg.DatasetDir = *basedir
	cfg.Name = *name

	err := cfg.Start()
	if err != nil {
		fmt.Printf("WebUI faced an internal error: %s\n", err)
		os.Exit(1)
	}
	cfg.Wait()
}
