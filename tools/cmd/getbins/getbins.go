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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	file := flag.String("file", "", "Input file with all the counts")
	binThresholds := flag.String("bins", "200", "Comma-separated list of thresholds to use for the creation of bins")
	dir := flag.String("dir", "", "Output directory")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyzes a given count file and classifying all the counts into bins", cmdName)
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

	jobids, err := datafilereader.GetJobIDsFromFileNames([]string{*file})
	if err != nil || len(jobids) != 1 {
		fmt.Println("[ERROR] unable to get job ID from filename")
		os.Exit(1)
	}

	ranks, err := datafilereader.GetRanksFromFileNames([]string{*file})
	if err != nil || len(ranks) != 1 {
		fmt.Println("[ERROR] unable to get rank from filename")
		os.Exit(1)
	}

	listBins := bins.GetFromInputDescr(*binThresholds)
	log.Printf("Ready to create %d bins\n", len(listBins))

	b, err := bins.GetFromFile(*file, listBins)
	if err != nil {
		fmt.Printf("[ERROR] Unable to get bins: %s", err)
		os.Exit(1)
	}

	err = bins.Save(*dir, jobids[0], ranks[0], b)
	if err != nil {
		fmt.Printf("[ERROR] Unable to save data in %s: %s\n", *dir, err)
		os.Exit(1)
	}
}
