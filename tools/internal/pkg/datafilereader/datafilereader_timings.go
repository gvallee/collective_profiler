//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package datafilereader

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TimingsStats struct {
	Timings string
	Max     float64
	Min     float64
	Mean    float64
}

type CallTimings struct {
	ExecutionTimings    TimingsStats
	LateArrivalsTimings TimingsStats
}

func extractTimingData(reader *bufio.Reader, laf *os.File, a2af *os.File) error {
	extractingLateArrivalTimings := false
	extractingAlltoallvExecutionTimings := false
	currentCall := ""

	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return readerErr
		}

		if strings.HasPrefix(line, timingsCallDelimiter) {
			currentCall = strings.TrimRight(line, "\n")
			currentCall = strings.TrimLeft(currentCall, timingsCallDelimiter)
			continue
		}

		if strings.HasPrefix(line, lateArrivalTimingDelimiter) {
			extractingLateArrivalTimings = true
			extractingAlltoallvExecutionTimings = false
			continue
		}

		if strings.HasPrefix(line, executionTimeDelimiter) {
			extractingLateArrivalTimings = false
			extractingAlltoallvExecutionTimings = true
			continue
		}

		if extractingAlltoallvExecutionTimings {
			err := saveLine(a2af, currentCall, line)
			if err != nil {
				return err
			}
		}

		if extractingLateArrivalTimings {
			err := saveLine(laf, currentCall, line)
			if err != nil {
				return err
			}
		}

		if readerErr == io.EOF {
			break
		}
	}
	return nil
}

func ExtractTimings(inputFile string, lateArrivalFilename string, a2aFilename string) error {
	inputf, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer inputf.Close()
	reader := bufio.NewReader(inputf)

	laf, err := os.OpenFile(lateArrivalFilename, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer laf.Close()

	a2af, err := os.OpenFile(a2aFilename, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer a2af.Close()

	return extractTimingData(reader, laf, a2af)
}

func getAlltoallvTimingsFilePath(dir string, jobid int, rank int) string {
	return filepath.Join(dir, fmt.Sprintf("alltoallv_timings.job%d.rank%d.dat", jobid, rank))
}

func getLateArrivalTimingsFilePath(dir string, jobid int, rank int) string {
	return filepath.Join(dir, fmt.Sprintf("late_arrival_timings.job%d.rank%d.dat", jobid, rank))
}

func getCallDataFromTimingsFile(path string, numCall int) (TimingsStats, error) {
	var t TimingsStats
	t.Max = 0.0
	t.Min = -1.0
	t.Timings = ""
	sum := 0.0
	num := 0.0

	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		line, readerr := reader.ReadString('\n')
		if readerr != nil && readerr != io.EOF {
			return t, readerr
		}
		if line == "" && readerr == io.EOF {
			break
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			continue
		}

		// We split the line, the first element is the call number and the second element the actual timing
		tokens := strings.Split(line, "\t")
		if len(tokens) != 2 {
			return t, fmt.Errorf("invalid format: %s", line)
		}
		callID, err := strconv.Atoi(tokens[0])
		if err != nil {
			return t, err
		}

		if callID < numCall {
			continue
		}

		if callID == numCall {
			timing, err := strconv.ParseFloat(strings.TrimRight(tokens[1], "\n"), 64)
			if err != nil {
				return t, err
			}
			sum += timing

			if t.Min == -1.0 || t.Min > timing {
				t.Min = timing
			}
			if t.Max < timing {
				t.Max = timing
			}

			t.Timings = t.Timings + tokens[1] + "\n"
			num++
		}

		if callID > numCall {
			break
		}
	}

	t.Mean = sum / num
	return t, nil
}

func getCallTimings(dir string, jobid int, rank int, numCall int) (CallTimings, error) {
	var t CallTimings
	var err error

	a2aTimingsFile := getAlltoallvTimingsFilePath(dir, jobid, rank)
	lateArrivalTimingsFile := getLateArrivalTimingsFilePath(dir, jobid, rank)

	log.Printf("-> Getting execution timings from %s\n", a2aTimingsFile)
	t.ExecutionTimings, err = getCallDataFromTimingsFile(a2aTimingsFile, numCall)
	if err != nil {
		return t, err
	}
	log.Printf("-> Getting late arrival timings from %s\n", lateArrivalTimingsFile)
	t.LateArrivalsTimings, err = getCallDataFromTimingsFile(lateArrivalTimingsFile, numCall)
	if err != nil {
		return t, err
	}

	return t, nil
}
