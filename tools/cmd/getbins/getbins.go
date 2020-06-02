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
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	file := flag.String("file", "", "Input file with all the counts")
	binThresholds := flag.String("bins", "200", "Comma-separated list of thresholds to use for the creation of bins")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "getbins")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	listBinsStr := strings.Split(*binThresholds, ",")
	var listBins []int
	for _, s := range listBinsStr {
		n, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("unable to get array of thresholds for bins: %s", err)
		}
		listBins = append(listBins, n)
	}
	log.Printf("Ready to create %d bins\n", len(listBins))

	bins, err := profiler.GetBins(*file, listBins)
	if err != nil {
		log.Fatalf("unable to get bins: %s", err)
	}

	for _, b := range bins {
		outputFile := fmt.Sprintf("bin_%d-%d.txt", b.Min, b.Max)
		if b.Max == -1 {
			outputFile = fmt.Sprintf("bin_%d+.txt", b.Min)
		}
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("unable to create file %s: %s", outputFile, err)
		}

		_, err = f.WriteString(fmt.Sprintf("%d\n", b.Size))
		if err != nil {
			log.Fatalf("unable to write bin to file: %s", err)
		}
	}
}
