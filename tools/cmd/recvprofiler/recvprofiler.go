//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"log"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
)

func main() {
	file := flag.String("file", "", "Path to the file from which we want to extract counters")

	flag.Parse()

	if *file == "" {
		log.Fatalf("undefined input file or output directory")
	}

	err := profiler.HandleCounts(*file)
	if err != nil {
		log.Fatalf("[ERROR] Impossible to analyze recv counters: %s", err)
	}
}
