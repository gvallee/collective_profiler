//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	file := flag.String("file", "", "Timing file that we need to parse to extract timings from")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "extracttimings")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	lateArrivalFilename := strings.ReplaceAll(filepath.Base(*file), "timings", "late_arrival_timings")
	lateArrivalFilename = strings.ReplaceAll(lateArrivalFilename, ".md", ".dat")
	a2aFilename := strings.ReplaceAll(filepath.Base(*file), "timings", "alltoallv_timings")
	a2aFilename = strings.ReplaceAll(a2aFilename, ".md", ".dat")

	err := datafilereader.ExtractTimings(*file, lateArrivalFilename, a2aFilename)
	if err != nil {
		log.Fatalf("unable to extract data: %s", err)
	}
}
