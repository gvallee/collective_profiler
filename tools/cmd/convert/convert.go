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
	"os"
	"path/filepath"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dirs := flag.String("dirs", "", "Comma-separated list of all the directories with raw non-aggregated count files from the profiler")
	outputDir := flag.String("output-dir", "", "Where the resulting files will be saved")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s convert various files from the profiler into the format used for post-portem analysis", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	listDirs := strings.Split(*dirs, ",")
	err := counts.LoadRawCountsFromDirs(listDirs, *outputDir)
	if err != nil {
		fmt.Printf("[ERROR] unable to load counters: %s\n", err)
		os.Exit(1)
	}
}
