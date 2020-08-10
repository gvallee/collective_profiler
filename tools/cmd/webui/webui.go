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

func displayUI(datasetBasedir string, datasetName string) error {
	sizeThreshold := 200
	binThresholds := "200,1024,2048,4096"
	listBins := bins.GetFromInputDescr(binThresholds)
	_, _, _, allCallsData, err := profiler.HandleCountsFiles(datasetBasedir, sizeThreshold, listBins)
	if err != nil {
		return err
	}

	data := CallsPageData{
		PageTitle: datasetName,
		Calls:     allCallsData,
	}

	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(filename)

	tmpl := template.Must(template.ParseFiles(filepath.Join(basedir, "templates", "callsLayout.html")))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, data)
	})
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
