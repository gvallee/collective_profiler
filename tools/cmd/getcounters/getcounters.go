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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	rank := flag.Int("rank", 0, "Rank for which we want to extract counters")
	call := flag.Int("call", 0, "Number of the alltoallv call for which we want to extract counters")
	dir := flag.String("dir", "", "Where the data files are stored")
	pid := flag.Int("pid", 0, "Identifier of the experiment, e.g., X from <pidX> in the profile file name")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "getcounters")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	sendCounters, recvCounters, err := datafilereader.FindCallRankCounters(*dir, *jobid, *pid, *rank, *call)
	if err != nil {
		log.Fatalf("unable to find counters: %s", err)
	}

	fmt.Println("Send counters:")
	fmt.Printf("%s\n", sendCounters)
	fmt.Println("Receive counters")
	fmt.Printf("%s\n", recvCounters)
}
