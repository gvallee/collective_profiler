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
	"runtime"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	help := flag.Bool("h", false, "Help message")
	sizeThreshold := flag.Int("size-threshold", profiler.DefaultMsgSizeThreshold, "Size to differentiate small and big messages")
	binThresholds := flag.String("bins", profiler.DefaultBinThreshold, "Comma-separated list of thresholds to use for the creation of bins")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyzes all the data gathered while running an application with our shared library", cmdName)
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
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	err := profiler.AnalyzeDataset(codeBaseDir, *dir, *binThresholds, *sizeThreshold)
	if err != nil {
		fmt.Printf("profiler.AnalyzeDataset() failed: %s\n", err)
		os.Exit(1)
	}
}
