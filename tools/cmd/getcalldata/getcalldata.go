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
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func saveCountsSummary(f *os.File, callInfo profiler.CallInfo) error {
	_, err := f.WriteString(fmt.Sprintf("Communicator size: %d\n", callInfo.CountsData.CommSize))
	if err != nil {
		return err
	}

	_, err = f.WriteString("\n### Send counts summary\n\n")
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Send data type size: %d\n", callInfo.CountsData.SendData.Statistics.DatatypeSize))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Total amount of data sent: %d\n", callInfo.CountsData.SendData.Statistics.Sum*callInfo.CountsData.SendData.Statistics.DatatypeSize))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Max send count: %d\n", callInfo.CountsData.SendData.Statistics.Max))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Min send count: %d\n", callInfo.CountsData.SendData.Statistics.Min))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Min non-zero send count: %d\n", callInfo.CountsData.SendData.Statistics.NotZeroMin))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Number of small non-zero messages: %d\n", callInfo.CountsData.SendData.Statistics.SmallNotZeroMsgs))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Number of large messages: %d\n", callInfo.CountsData.SendData.Statistics.LargeMsgs))
	if err != nil {
		return err
	}
	_, err = f.WriteString("\n### Receive counts summary\n\n")
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Recv data type size: %d\n", callInfo.CountsData.RecvData.Statistics.DatatypeSize))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Total amount of data received: %d\n", callInfo.CountsData.RecvData.Statistics.Sum*callInfo.CountsData.RecvData.Statistics.DatatypeSize))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Max recv count: %d\n", callInfo.CountsData.RecvData.Statistics.Max))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Min recv count: %d\n", callInfo.CountsData.RecvData.Statistics.Min))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Min non-zero recv count: %d\n", callInfo.CountsData.RecvData.Statistics.NotZeroMin))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Number of small non-zero messages: %d\n", callInfo.CountsData.RecvData.Statistics.SmallNotZeroMsgs))
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("Number of large messages: %d\n", callInfo.CountsData.RecvData.Statistics.LargeMsgs))
	if err != nil {
		return err
	}

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	calls := flag.String("calls", "", "Calls for which we want to extract data. It can be a comma-separated list of call number as well as ranges in the format X-Y.")
	dir := flag.String("dir", "", "Where the data files are stored")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")
	// todo: clarify lead rank vs communicator ID to handle data
	rank := flag.Int("rank", 0, "Rank for which we want to analyse the counters. When using multiple communicators for alltoallv operations, results for multiple ranks are reported.")
	commid := flag.Int("comm", 0, "Communicator ID that identifies from which communicator we want the data")
	collectiveName := flag.String("collective", "alltoallv", "Name of the collective operation from which the call data is requested (alltoallv by default)")
	msgSizeThreshold := flag.Int("msg-size-threshold", format.DefaultMsgSizeThreshold, "Message size threshold to differentiate small messages from large messages.")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s extracts the data related to one or more alltoallv call.", cmdName)
		fmt.Println("Note that it will overwrite any previous result files since the command gathers statistics based on a run-time parameters.")
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

	// Convert the list of calls into something that can actually be used
	var listCalls []int
	tokens := strings.Split(*calls, ",")
	for _, t := range tokens {
		tokens2 := strings.Split(t, "-")
		if len(tokens2) == 2 {
			startVal, err := strconv.Atoi(tokens2[0])
			if err != nil {
				log.Fatalf("unable to parse %s: %s", tokens2[0], err)
			}
			endVal, err := strconv.Atoi(tokens2[1])
			if err != nil {
				log.Fatalf("unable to parse %s: %s", tokens2[1], err)
			}
			for i := startVal; i <= endVal; i++ {
				listCalls = append(listCalls, i)
			}
		} else {
			val, err := strconv.Atoi(t)
			if err != nil {
				log.Fatalf("unable to parse %s: %s", t, err)
			}
			listCalls = append(listCalls, val)
		}
	}

	// Getting the actual data for each call
	if *verbose {
		log.Printf("Getting data for call(s):")
		for _, val := range listCalls {
			fmt.Printf(" %d", val)
		}
		fmt.Printf("\n")
	}

	_, filename, _, _ := runtime.Caller(0)
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	var callsInfo []profiler.CallInfo
	for _, callNum := range listCalls {
		callInfo, err := profiler.GetCallData(codeBaseDir, *collectiveName, *dir, *commid, *jobid, *rank, callNum, *msgSizeThreshold)
		if err != nil {
			log.Fatalf("unable to get data of call #%d: %s", callNum, err)
		}
		callsInfo = append(callsInfo, callInfo)

		callFilePath := filepath.Join(*dir, fmt.Sprintf("call%d-job%d-rank%d.md", callNum, *jobid, *rank))
		newFile, err := os.OpenFile(callFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Fatalf("unable to create %s: %s", callFilePath, err)
		}
		defer newFile.Close()

		_, err = newFile.WriteString("\n# Bracktrace\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if callInfo.Backtrace != "" {
			_, err = newFile.WriteString(callInfo.Backtrace)
		} else {
			_, err = newFile.WriteString("No data\n")
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}

		_, err = newFile.WriteString("\n# Patterns\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if callInfo.PatternStr != "" {
			_, err = newFile.WriteString(callInfo.PatternStr)
		} else {
			_, err = newFile.WriteString("No data\n")
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}

		_, err = newFile.WriteString("\n# Timing summary (data further below)\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString("## Late arrivals timings\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString(fmt.Sprintf("Min: %f; Max: %f; Mean: %f\n", callInfo.Timings.LateArrivalsTimings.Min, callInfo.Timings.LateArrivalsTimings.Max, callInfo.Timings.LateArrivalsTimings.Mean))
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString("\n## Execution timings\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString(fmt.Sprintf("Min: %f; Max: %f; Mean: %f\n", callInfo.Timings.ExecutionTimings.Min, callInfo.Timings.ExecutionTimings.Max, callInfo.Timings.ExecutionTimings.Mean))
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}

		_, err = newFile.WriteString("\n# Counts\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}

		err = saveCountsSummary(newFile, callInfo)
		if err != nil {
			log.Fatalf("unable to save counts summary: %s", err)
		}

		_, err = newFile.WriteString("\n## Send counts\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if len(callInfo.CountsData.SendData.RawCounts) != 0 {
			_, err = newFile.WriteString(strings.Join(callInfo.CountsData.SendData.RawCounts, "\n"))
		} else {
			_, err = newFile.WriteString("No data\n")
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString("\n# Receive counts\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if len(callInfo.CountsData.RecvData.RawCounts) == 0 {
			_, err = newFile.WriteString("No data\n")
		} else {
			_, err = newFile.WriteString(strings.Join(callInfo.CountsData.RecvData.RawCounts, "\n"))
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}

		_, err = newFile.WriteString("\n# Timings\n\n")
		_, err = newFile.WriteString("\n## Late arrival timings\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if callInfo.Timings.LateArrivalsTimings.Timings != "" {
			_, err = newFile.WriteString(callInfo.Timings.LateArrivalsTimings.Timings)
		} else {
			_, err = newFile.WriteString("No data\n")
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = newFile.WriteString("\n##Alltoallv execution time\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if callInfo.Timings.ExecutionTimings.Timings != "" {
			_, err = newFile.WriteString(callInfo.Timings.ExecutionTimings.Timings)
		} else {
			_, err = newFile.WriteString("No data\n")
		}
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		fmt.Printf("Data for call #%d saved in %s\n", callNum, callFilePath)
	}

	summaryFilePath := filepath.Join(*dir, fmt.Sprintf("calls-%s.md", *calls))
	summaryFile, err := os.OpenFile(summaryFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatalf("unable to open file %s: %s", summaryFilePath, err)
	}
	defer summaryFile.Close()

	var uniqueBacktraces []string
	// Find unique backtraces
	for _, info := range callsInfo {
		backtraceExists := false
		for _, bt := range uniqueBacktraces {
			if info.Backtrace == bt {
				backtraceExists = true
				break
			}
		}

		if !backtraceExists {
			uniqueBacktraces = append(uniqueBacktraces, info.Backtrace)
		}
	}

	num := 1
	for _, ubt := range uniqueBacktraces {
		_, err = summaryFile.WriteString("# Backtraces\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = summaryFile.WriteString(fmt.Sprintf("## Backtrace %d\n\n", num))
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		_, err = summaryFile.WriteString(fmt.Sprintf("%s\n", ubt))
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		num++
	}

	// Find unique min/max counts
	var uniqueMinMaxCallsInfo []profiler.CallInfo
	callIDs := make([][]int, len(callsInfo))
	num = 0
	for _, info := range callsInfo {
		dataExists := false
		for _, uniqueMinMaxInfo := range uniqueMinMaxCallsInfo {
			if info.CountsData.CommSize == uniqueMinMaxInfo.CountsData.CommSize &&
				info.RecvStats.DatatypeSize == uniqueMinMaxInfo.RecvStats.DatatypeSize &&
				info.SendStats.DatatypeSize == uniqueMinMaxInfo.SendStats.DatatypeSize &&
				reflect.DeepEqual(info.CountsData.RecvData.RawCounts, uniqueMinMaxInfo.CountsData.RecvData.RawCounts) &&
				reflect.DeepEqual(info.CountsData.SendData.RawCounts, uniqueMinMaxInfo.CountsData.SendData.RawCounts) {
				dataExists = true
				callIDs[num] = append(callIDs[num], info.ID)
				break
			}
		}

		if !dataExists {
			uniqueMinMaxCallsInfo = append(uniqueMinMaxCallsInfo, info)
			callIDs[len(uniqueMinMaxCallsInfo)-1] = append(callIDs[len(uniqueMinMaxCallsInfo)-1], info.ID)
		}
		num++
	}

	num = 0
	_, err = summaryFile.WriteString("\n## Summary\n\n")
	if err != nil {
		log.Fatalf("unable to write to file: %s", err)
	}
	for _, uniqueInfo := range uniqueMinMaxCallsInfo {
		_, err = summaryFile.WriteString(fmt.Sprintf("Call(s): %s\n", notation.CompressIntArray(callIDs[num])))
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		err = saveCountsSummary(summaryFile, uniqueInfo)
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
	}
	fmt.Printf("Summary is saved in %s\n", summaryFilePath)
}
