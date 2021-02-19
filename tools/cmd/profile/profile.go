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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/maps"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/plot"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/go_util/pkg/util"
)

func plotCallsData(dir string, allCallsData []counts.CommDataT, rankFileData map[int]location.RankFileData, callMaps map[int]maps.CallsDataT, a2aExecutionTimes map[int]map[int]map[int]float64, lateArrivalTimes map[int]map[int]map[int]float64) error {
	for i := 0; i < len(allCallsData); i++ {
		b := progress.NewBar(len(allCallsData), "Plotting data for alltoallv calls")
		defer progress.EndBar(b)
		leadRank := allCallsData[i].LeadRank
		for callID := range allCallsData[i].CallData {
			b.Increment(1)

			_, err := plot.CallData(dir, dir, leadRank, callID, rankFileData[leadRank].HostMap, callMaps[leadRank].SendHeatMap[i], callMaps[leadRank].RecvHeatMap[i], a2aExecutionTimes[leadRank][i], lateArrivalTimes[leadRank][i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	help := flag.Bool("h", false, "Help message")
	sizeThreshold := flag.Int("size-threshold", 200, "Size to differentiate small and big messages")
	binThresholds := flag.String("bins", "200,1024,2048,4096", "Comma-separated list of thresholds to use for the creation of bins")

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

	listBins := bins.GetFromInputDescr(*binThresholds)

	totalNumSteps := 5
	currentStep := 1
	fmt.Printf("* Step %d/%d: analyzing counts...\n", currentStep, totalNumSteps)
	t := timer.Start()
	totalNumCalls, stats, allPatterns, allCallsData, err := profiler.HandleCountsFiles(*dir, *sizeThreshold, listBins)
	duration := t.Stop()
	if err != nil {
		fmt.Printf("ERROR: unable to analyze counts: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Step completed in %s\n", duration)
	currentStep++

	fmt.Printf("\n* Step %d/%d: analyzing MPI communicator data...\n", currentStep, totalNumSteps)
	t = timer.Start()
	err = profiler.AnalyzeSubCommsResults(*dir, stats, allPatterns)
	duration = t.Stop()
	if err != nil {
		fmt.Printf("ERROR: unable to analyze sub-communicators results: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Step completed in %s\n", duration)
	currentStep++

	fmt.Printf("\n* Step %d/%d: create maps...\n", currentStep, totalNumSteps)
	t = timer.Start()
	//rankFileData, callMaps, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap, err := maps.Create(maps.Heat, *dir, allCallsData)
	_, callMaps, globalSendHeatMap, globalRecvHeatMap, _, err := maps.Create(maps.Heat, *dir, allCallsData)
	duration = t.Stop()
	if err != nil {
		fmt.Printf("ERROR: unable to create heat map: %s\n", err)
		os.Exit(1)
	}
	// Create maps with averages
	//avgSendHeatMap, avgRecvHeatMap := maps.CreateAvgMaps(totalNumCalls, globalSendHeatMap, globalRecvHeatMap)
	maps.CreateAvgMaps(totalNumCalls, globalSendHeatMap, globalRecvHeatMap)
	fmt.Printf("Step completed in %s\n", duration)
	currentStep++

	fmt.Printf("\n* Step %d/%d: analyzing timing files...\n", currentStep, totalNumSteps)
	t = timer.Start()
	_, _, _, err = timings.HandleTimingFiles(codeBaseDir, *dir, totalNumCalls, callMaps)
	if err != nil {
		fmt.Printf("Unable to parse timing data: %s", err)
		os.Exit(1)
	}
	/*
		duration = t.Stop()
		if err != nil {
			fmt.Printf("ERROR: unable to analyze timings: %s\n", err)
			os.Exit(1)
		}
		avgExecutionTimes := make(map[int]float64)
		for rank, execTime := range totalA2AExecutionTimes {
			rankNumCalls := rankNumCallsMap[rank]
			avgExecutionTimes[rank] = execTime / float64(rankNumCalls)
		}
		avgLateArrivalTimes := make(map[int]float64)
		for rank, lateTime := range totalLateArrivalTimes {
			rankNumCalls := rankNumCallsMap[rank]
			avgExecutionTimes[rank] = lateTime / float64(rankNumCalls)
		}
		fmt.Printf("Step completed in %s\n", duration)
		currentStep++

		fmt.Printf("\n* Step %d/%d: generating plots...\n", currentStep, totalNumSteps)
		t = timer.Start()
		err = plotCallsData(*dir, allCallsData, rankFileData, callMaps, a2aExecutionTimes, lateArrivalTimes)
		duration = t.Stop()
		if err != nil {
			fmt.Printf("ERROR: unable to plot data: %s", err)
			os.Exit(1)
		}
		err = plot.Avgs(*dir, *dir, len(rankFileData[0].RankMap), rankFileData[0].HostMap, avgSendHeatMap, avgRecvHeatMap, avgExecutionTimes, avgLateArrivalTimes)
		if err != nil {
			fmt.Printf("ERROR: unable to plot average data: %s", err)
		}
		fmt.Printf("Step completed in %s\n", duration)
		currentStep++
	*/
}
