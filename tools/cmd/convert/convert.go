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
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dirs := flag.String("dirs", "", "Comma-separated list of all the directories with files from the profiler (raw non-aggregated count or timing file(s)")
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
		profileFiles, err := profiler.ScanDirectories(listDirs)
		if err != nil {
			fmt.Printf("[ERROR] unable to scan directories: %s", err)
			os.Exit(1)
		}
		if profileFiles.IncludesRawCountsFiles() {
			err := counts.LoadRawCountsFromDirs(listDirs, *outputDir)
			if err != nil {
				fmt.Printf("[ERROR] unable to load counters from directories: %s\n", err)
				os.Exit(1)
			}
		}
		/*
			if profileFiles.IncludeSingleTimingsFile() {
				err := timings.ConvertSingleTimingsFile(profileFiles.SingleTimingFile)
				if err != nil {
					fmt.Printf("[ERROR] unable to convert single timing file: %s\n", err)
					os.Exit(1)
				}
			}
		*/
		if profileFiles.IncludesCoupledTimingFiles() {
			allFiles := profileFiles.CoupledTimingFiles.A2AExecTimingFiles
			allFiles = append(allFiles, profileFiles.CoupledTimingFiles.LateArrivalTimingFiles...)
			err := timings.ConvertCoupledTimingFiles(allFiles, *outputDir)
			if err != nil {
				fmt.Printf("[ERROR] unable to convert coupled timing files: %s\n", err)
				os.Exit(1)
			}
		}
	}

	if *files != "" {
		listFiles := strings.Split(*files, ",")

		var listRawCountsFiles []string
		var listCoupledTimingFiles []string

		for _, file := range listFiles {
			filename := filepath.Base(file)
			fmt.Printf("Checking %s...\n", filename)
			if strings.HasPrefix(filename, counts.RawCountersFilePrefix) {
				listRawCountsFiles = append(listRawCountsFiles, file)
			}
			if strings.HasPrefix(filename, timings.AlltoallExecCoupledFilePrefix) || strings.HasPrefix(filename, timings.LateArrivalFilePrefix) || strings.HasPrefix(filename, "late_arrival_timings.") {
				fmt.Printf("Adding file: %s\n", filename)
				listCoupledTimingFiles = append(listCoupledTimingFiles, file)
			}
		}

		if len(listRawCountsFiles) > 0 {
			err := counts.LoadRawCountsFromFiles(listRawCountsFiles, *outputDir)
			if err != nil {
				fmt.Printf("[ERROR] unable to load counters from files: %s\n", err)
				os.Exit(1)
			}
		}

		if len(listCoupledTimingFiles) > 0 {
			err := timings.ConvertCoupledTimingFiles(listCoupledTimingFiles, *outputDir)
			if err != nil {
				fmt.Printf("[ERROR] unable to convert timing files %s: %s\n", strings.Join(listCoupledTimingFiles, ","), err)
				os.Exit(1)
			}
		}
	}
}
