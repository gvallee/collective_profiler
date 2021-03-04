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
	"os"
	"path/filepath"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/webui"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	datasetDir := flag.String("dataset", "", "Base directory of the dataset")
	name := flag.String("name", "example", "Name of the dataset to display")
	help := flag.Bool("h", false, "Help message")
	port := flag.Int("port", webui.DefaultPort, "Port on which to start the WebUI")

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

	fmt.Printf("Starting WebUI for dataset in %s...\n", *datasetDir)

	cfg := webui.Init()

	if *datasetDir == "" || !util.PathExists(*datasetDir) {
		fmt.Printf("%s is an invalid dataset, please make sure to use the '-dataset' parameter to point to the profiling data\n", *datasetDir)
		os.Exit(1)
	}
	cfg.DatasetDir = *datasetDir
	cfg.Name = *name
	cfg.Port = *port

	err := cfg.Start()
	if err != nil {
		fmt.Printf("WebUI faced an internal error: %s\n", err)
		os.Exit(1)
	}
	cfg.Wait()
}
