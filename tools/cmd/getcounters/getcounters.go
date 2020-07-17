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
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	rank := flag.Int("rank", 0, "Rank for which we want to extract counters")
	call := flag.Int("call", 0, "Number of the alltoallv call for which we want to extract counters")
	dir := flag.String("dir", "", "Where the data files are stored")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s extracts the send and receive counts from a profile's dataset.", cmdName)
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

	sendCounters, recvCounters, err := counts.FindCallRankCounters(*dir, *jobid, *rank, *call)
	if err != nil {
		log.Fatalf("unable to find counters: %s", err)
	}

	fmt.Println("Send counters:")
	fmt.Printf("%s\n", sendCounters)
	fmt.Println("Receive counters")
	fmt.Printf("%s\n", recvCounters)
}
