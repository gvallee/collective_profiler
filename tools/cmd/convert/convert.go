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
	files := flag.String("files", "", "Comma-separated list of files to convert")
	outputDir := flag.String("output-dir", "", "Where the resulting files will be saved")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s converts various files from the profiler into the format used for post-portem analysis", cmdName)
		fmt.Printf("\nUsage: %s [-dirs <list directories | -files <list files]\n", cmdName)
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

	if *dirs != "" && *files != "" {
		fmt.Printf("[ERROR] Both 'dirs' and 'files' options are set; only one can be used at a time")
		os.Exit(1)
	}

	if *dirs != "" {
		listDirs := strings.Split(*dirs, ",")
		err := counts.LoadRawCountsFromDirs(listDirs, *outputDir)
		if err != nil {
			fmt.Printf("[ERROR] unable to load counters from directories: %s\n", err)
			os.Exit(1)
		}
	}

	if *files != "" {
		listFiles := strings.Split(*files, ",")
		err := counts.LoadRawCountsFromFiles(listFiles, *outputDir)
		if err != nil {
			fmt.Printf("[ERROR] unable to load counters from files: %s\n", err)
		}
	}
}
