//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
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
	"runtime"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	help := flag.Bool("h", false, "Help message")
	steps := flag.String("steps", profiler.DefaultSteps, "Request specific steps to be executed.\n"+
		"WARNING! The current implementation may generate files for every single collective operation, which can result in a very large amount of files.\n"+
		"To specify steps, it is possible to list specific steps through a comma separated list or a rang of steps (e.g., \"1-3\").\n"+
		"Steps are currently:\n"+
		"\t1 - analyze send/recv counts;\n"+
		"\t2 - detect patterms;\n"+
		"\t3 - create heat maps;\n"+
		"\t4 - analyze timing data;\n"+
		"\t5 - gathering of statistics for every single calls"+
		"\t6 - plot graphs;\n"+
		"\t7 - create bins;\n")
	sizeThreshold := flag.Int("size-threshold", profiler.DefaultMsgSizeThreshold, "Size to differentiate small and big messages")
	graphList := fmt.Sprintf("0-%d", profiler.DefaultNumGeneratedGraphs)
	plotCalls := flag.String("plot", graphList, "Range of calls for which the tool will generate graphs.\n"+
		"To specify calls, it is possible to list specific steps through a comma separated list or a rang of steps (e.g., \"1-3\").\n")
	binThresholds := flag.String("bins", profiler.DefaultBinThreshold, "Comma-separated list of thresholds to use for the creation of bins")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyzes all the data gathered while running an application with our shared library", cmdName)
		fmt.Println("\nUsage:")
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

	_, filename, _, _ := runtime.Caller(0)
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	collectiveName := "alltoallv" // hardcoded for now detection coming soon

	profilerCfg := profiler.PostmortemConfig{
		CodeBaseDir:    codeBaseDir,
		CollectiveName: collectiveName,
		DatasetDir:     *dir,
		BinThresholds:  *binThresholds,
		SizeThreshold:  *sizeThreshold,
		Steps:          *steps,
		CallsToPlot:    *plotCalls,
	}
	err := profilerCfg.Analyze()
	if err != nil {
		fmt.Printf("profiler.AnalyzeDataset() failed: %s\n", err)
		os.Exit(1)
	}
}
