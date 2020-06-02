//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package datafilereader

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	header1                    = "# Raw counters"
	numberOfRanksMarker        = "Number of ranks: "
	datatypeSizeMarker         = "Datatype size: "
	alltoallvCallNumbersMarker = "Alltoallv calls "
	countMarker                = "Count: "
	beginningDataMarker        = "BEGINNING DATA"
	endDataMarker              = "END DATA"

	timingsCallDelimiter       = "Alltoallv call #"
	lateArrivalTimingDelimiter = "# Late arrival timings"
	executionTimeDelimiter     = "# Execution times of Alltoallv function"

	sendCountersFilePrefix = "send-counters."
	recvCountersFilePrefix = "recv-counters."
)

type CallPattern struct {
	SendZeroCounts    map[int]int // Number of zeros in send counts (on a per-rank basis)
	RecvZeroCounts    map[int]int
	SendNotZeroCounts map[int]int
	RecvNotZeroCounts map[int]int
	SendPatterns      map[int]int
	RecvPatterns      map[int]int
}

type CallInfo struct {
	Patterns             CallPattern
	PatternStr           string
	SendCounts           []string
	RecvCounts           []string
	Timings              CallTimings
	Backtrace            string
	SendMin              int
	RecvMin              int
	SendNotZeroMin       int
	RecvNotZeroMin       int
	SendMax              int
	RecvMax              int
	SendDatatypeSize     int
	RecvDatatypeSize     int
	CommSize             int
	SendSmallMsgs        int
	SendSmallNotZeroMsgs int
	SendLargeMsgs        int
	RecvSmallMsgs        int
	RecvSmallNotZeroMsgs int
	RecvLargeMsgs        int
	TotalSendZeroCounts  int
	TotalRecvZeroCounts  int
}

// CountsStats gathers all the stats from counts (send or receive) for a given alltoallv call
type CountsStats struct {
	MinWithoutZero         int         // Min from the entire counts not including zero
	Min                    int         // Min from the entire counts, including zero
	Max                    int         // Max from the entire counts
	SmallMsgs              int         // Number of small message from counts, including 0-size count
	SmallNotZeroMsgs       int         // Number of small message from counts, not including 0-size counts
	LargeMsgs              int         // Number of large messages from counts
	TotalZeroCounts        int         // Total number of zero counts from counters
	ZerosPerRankPatterns   map[int]int // Number of 0-counts on a per-rank basis ('X ranks have Y 0-counts')
	NoZerosPerRankPatterns map[int]int // Number of non-0-counts on a per-rank bases ('X ranks have Y non-0-counts)
	Patterns               map[int]int // Number of peers involved in actual communication, i.e., non-zeroa ('X ranks are sendinng to Y ranks')
}

func GetNumCalls(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	totalNumCalls, _, _, _, _, _, err := GetHeader(reader)
	if err != nil {
		return 0, err
	}

	return totalNumCalls, nil
}

func saveLine(file *os.File, callNum string, line string) error {
	if strings.HasPrefix(line, "Rank") {
		tokens := strings.Split(line, ": ")
		line = callNum + "\t" + tokens[1]
	}
	_, err := file.WriteString(line)
	return err
}

func GetStatsFilePath(basedir string, jobid int, pid int) string {
	return filepath.Join(basedir, fmt.Sprintf("stats-job%d-pid%d.md", jobid, pid))
}

func GetCallData(dir string, jobid int, pid int, callNum int) (CallInfo, error) {
	var info CallInfo

	// Load the counts from raw data
	log.Printf("Extracting send/receive counts for call #%d\n", callNum)
	sendCountsFile, recvCountsFile := GetCountsFiles(jobid, pid)
	sendCountsFile = filepath.Join(dir, sendCountsFile)
	recvCountsFile = filepath.Join(dir, recvCountsFile)

	sendCountsFd, err := os.Open(sendCountsFile)
	if err != nil {
		return info, nil
	}
	defer sendCountsFd.Close()
	sendCountsFileReader := bufio.NewReader(sendCountsFd)

	recvCountsFd, err := os.Open(recvCountsFile)
	if err != nil {
		return info, nil
	}
	defer recvCountsFd.Close()
	recvCountsFileReader := bufio.NewReader(recvCountsFd)

	_, _, info.SendCounts, err = lookupCallfromCountsFile(sendCountsFileReader, callNum)
	if err != nil {
		return info, nil
	}
	_, _, info.RecvCounts, err = lookupCallfromCountsFile(recvCountsFileReader, callNum)
	if err != nil {
		return info, nil
	}

	// Get timings from formatted timing file
	// todo: if the files do not exist, we should get the data from scratch

	log.Printf("Extracting timings for call #%d\n", callNum)
	info.Timings, err = getCallTimings(dir, jobid, pid, callNum)
	if err != nil {
		return info, err
	}
	//info.AlltoallvTimings, info.LateArrivalTiming

	// Load patterns from result file.
	// todo: if the file does not exists, we should get the data from scratch
	log.Printf("Extracting patterns for call #%d\n", callNum)
	info.PatternStr, err = getCallPatterns(dir, jobid, pid, callNum)
	if err != nil {
		return info, err
	}

	// Load the backtrace
	log.Printf("Extracting backtrace for call #%d\n", callNum)
	info.Backtrace, err = getCallBacktrace(dir, callNum)
	if err != nil {
		return info, err
	}

	return info, nil
}
