//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	file := flag.String("file", "", "Timing file that we need to parse to extract timings from")
	sendCountsFile := flag.String("send-counts", "", "File where all the send counts for the alltoallv calls are stored")
	recvCountsFile := flag.String("recv-counts", "", "File where all the recv counts for the alltoallv calls are stored")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "extracttimings")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	if !util.PathExists(*sendCountsFile) {
		log.Fatalf("%s does not exists", *sendCountsFile)
	}
	if !util.PathExists(*recvCountsFile) {
		log.Fatalf("%s does not exists", *recvCountsFile)
	}

	inputf, err := os.Open(*file)
	if err != nil {
		log.Fatalf("unable to open %s: %s", *file, err)
	}
	defer inputf.Close()
	reader := bufio.NewReader(inputf)

	timings, err := analyzer.GetCallsTimings(reader)
	if err != nil {
		log.Fatalf("unable to get timings for alltoallv calls: %s", err)
	}

	for _, t := range timings {
		tWait := t.MaxTime - t.MinTime
		minDataSentSize, minDataRecvSize, err := profiler.GetCallRankData(*sendCountsFile, *recvCountsFile, t.CallNum, t.RankMinTime)
		if err != nil {
			log.Fatalf("unable to look up data for call %d, rank %d: %s", t.CallNum, t.RankMinTime, err)
		}
		maxDataSentSize, maxDataRecvSize, err := profiler.GetCallRankData(*sendCountsFile, *recvCountsFile, t.CallNum, t.RankMaxTime)
		if err != nil {
			log.Fatalf("unable to look up data for call %d, rank %d: %s", t.CallNum, t.RankMaxTime, err)
		}
		fmt.Printf("Call #%d: execution time = %fs; wait time = %fs; faster rank: %d (send: %d bytes; recv: %d bytes); slower rank: %d (send: %d bytes; recv: %d bytes)\n",
			t.CallNum, t.MinTime, tWait, t.RankMinTime, minDataSentSize, minDataRecvSize, t.RankMaxTime, maxDataSentSize, maxDataRecvSize)
	}
}
