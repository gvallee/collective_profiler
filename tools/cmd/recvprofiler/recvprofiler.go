//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	file := flag.String("file", "", "Path to the file from which we want to extract counters")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "recvprofiler")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	if *file == "" {
		log.Fatalf("undefined input file or output directory")
	}

	err := profiler.Handle(*file)
	if err != nil {
		log.Fatalf("[ERROR] Impossible to analyze recv counters: %s", err)
	}
}
