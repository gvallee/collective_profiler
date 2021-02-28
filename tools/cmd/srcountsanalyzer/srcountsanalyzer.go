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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"

	"github.com/gvallee/go_util/pkg/util"
)

func displayCallPatterns(p patterns.CallData) {
	for numPeers, numRanks := range p.Send {
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
		os.Exit(0)
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
	defer outputFileInfo.Cleanup()
	if err != nil {
		log.Fatalf("unable to open output files: %s", err)
	}

	sendCountsFile, recvCountsFile := counts.GetFiles(*jobid, *rank)
	sendCountsFile = filepath.Join(*dir, sendCountsFile)
	recvCountsFile = filepath.Join(*dir, recvCountsFile)

	if !util.PathExists(sendCountsFile) || !util.PathExists(recvCountsFile) {
		fmt.Printf("[ERROR] unable to locate send or recv counts file(s) (%s, %s) in %s", sendCountsFile, recvCountsFile, *dir)
		os.Exit(1)
	}

	log.Printf("Send counts file: %s\n", sendCountsFile)
	log.Printf("Recv counts file: %s\n", recvCountsFile)

	numCalls, err := counts.GetNumCalls(sendCountsFile)
	if err != nil {
		fmt.Printf("[ERROR] unable to get the number of alltoallv calls: %s", err)
		os.Exit(1)
	}

	cs, p, err := patterns.ParseFiles(sendCountsFile, recvCountsFile, numCalls, *rank, *sizeThreshold)
	if err != nil {
		fmt.Printf("[ERROR] unable to parse count file %s", sendCountsFile)
		os.Exit(1)
	}

	sendRecvStats, err := counts.GatherStatsFromCallData(cs, *sizeThreshold)
	if err != nil {
		fmt.Printf("[ERROR] unable to gather statistics from alltoallv calls' data")
		os.Exit(1)
	}

	err = profiler.SaveStats(outputFileInfo, sendRecvStats, p, numCalls, *sizeThreshold)
	if err != nil {
		fmt.Printf("[ERROR] unable to save counters' stats: %s", err)
		os.Exit(1)
	}
}
