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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"

	"github.com/gvallee/go_util/pkg/util"
)

func displayCallPatterns(info datafilereader.CallInfo) {
	for numPeers, numRanks := range info.Patterns.SendPatterns {
		fmt.Printf("%d ranks are sending non-zero data to %d other ranks\n", numRanks, numPeers)
	}
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	outputDir := flag.String("output-dir", "", "Where all the output files will be created")
	rank := flag.Int("rank", 0, "Rank for which we want to analyse the counters. When using multiple communicators for alltoallv operations, results for multiple ranks are reported.")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")
	sizeThreshold := flag.Int("size-threshold", 200, "Threshold to differentiate size and large messages")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s extracts data from send and receive counts and gathers statistics about them.", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
	}

	if !util.PathExists(*outputDir) {
		fmt.Printf("Output directory '%s' does not exist", *outputDir)
		flag.PrintDefaults()
		os.Exit(1)
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	outputFileInfo, err := profiler.GetCountProfilerFileDesc(*outputDir, *jobid, *rank)
	if err != nil {
		log.Fatalf("unable to open output files: %s", err)
	}
	defer outputFileInfo.Cleanup()

	sendCountsFile, recvCountsFile := datafilereader.GetCountsFiles(*jobid, *rank)
	sendCountsFile = filepath.Join(*dir, sendCountsFile)
	recvCountsFile = filepath.Join(*dir, recvCountsFile)

	if !util.PathExists(sendCountsFile) || !util.PathExists(recvCountsFile) {
		log.Fatalf("unable to locate send or recv counts file(s) (%s, %s) in %s", sendCountsFile, recvCountsFile, *dir)
	}

	log.Printf("Send counts file: %s\n", sendCountsFile)
	log.Printf("Recv counts file: %s\n", recvCountsFile)

	numCalls, err := datafilereader.GetNumCalls(sendCountsFile)
	if err != nil {
		log.Fatalf("unable to get the number of alltoallv calls: %s", err)
	}

	cs, err := profiler.ParseCountFiles(sendCountsFile, recvCountsFile, numCalls, *sizeThreshold)
	if err != nil {
		log.Fatalf("unable to parse count file %s", sendCountsFile)
	}

	err = profiler.SaveCounterStats(outputFileInfo, cs, numCalls, *sizeThreshold)
	if err != nil {
		log.Fatalf("unable to save counters' stats: %s", err)
	}
}
