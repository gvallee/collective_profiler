package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/go_util/pkg/util"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	calls := flag.String("calls", "", "Calls for which we want to extract data. It can be a comma-separated list of call number as well as ranges in the format X-Y.")
	dir := flag.String("dir", "", "Where the data files are stored")
	pid := flag.Int("pid", 0, "Identifier of the experiment, e.g., X from <pidX> in the profile file name")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "extracttimings")
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

	for _, callNum := range listCalls {
		callInfo, err := datafilereader.GetCallData(*dir, *jobid, *pid, callNum)
		if err != nil {
			log.Fatalf("unable to get data of call #%d: %s", callNum, err)
		}

		callFilePath := filepath.Join(*dir, fmt.Sprintf("call%d-job%d-pid%d.md", callNum, *jobid, *pid))
		newFile, err := os.OpenFile(callFilePath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("unable to create %s: %s", callFilePath, err)
		}

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
		_, err = newFile.WriteString("## Send counts\n\n")
		if err != nil {
			log.Fatalf("unable to write to file: %s", err)
		}
		if len(callInfo.SendCounts) != 0 {
			_, err = newFile.WriteString(strings.Join(callInfo.SendCounts, "\n"))
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
		if len(callInfo.RecvCounts) == 0 {
			_, err = newFile.WriteString("No data\n")
		} else {
			_, err = newFile.WriteString(strings.Join(callInfo.RecvCounts, "\n"))
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
}
