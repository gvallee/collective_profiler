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

	// SendCountersFilePrefix is the prefix used for all send counts files
	SendCountersFilePrefix = "send-counters."
	// RecvCountersFilePrefix is the prefix used for all receive counts files
	RecvCountersFilePrefix = "recv-counters."

	// ProfileSummaryFilePrefix is the prefix used for all generated profile summary files
	ProfileSummaryFilePrefix = "profile_alltoallv_rank"

	// DefaultMsgSizeThreshold is the default threshold to differentiate message and large messages.
	DefaultMsgSizeThreshold = 200
)

type CallPattern struct {
	SendZeroCounts    map[int]int // Number of zeros in send counts (on a per-rank basis)
	RecvZeroCounts    map[int]int
	SendNotZeroCounts map[int]int
	RecvNotZeroCounts map[int]int
	SendPatterns      map[int]int
	RecvPatterns      map[int]int
}

// CallInfo gathers all the data extracted about a specific alltoallv call
type CallInfo struct {
	// ID is the call number (zero-indexed)
	ID int

	// Patterns gathers all the communication patterns associated to the alltoallv call
	Patterns CallPattern

	// PatternStr is the string version of the communication patterns
	PatternStr string

	// SendCounts is the string representing all the send counts
	SendCounts []string

	// RecvCounts is the string representing all the receive counts
	RecvCounts []string

	// Timings represent all the timings associated to the alltoallv call (e.g., late arrival and execution timings)
	Timings CallTimings

	// Backtrace is the string version of the alltoallv call's backtrace
	Backtrace string

	// SendSum is the sum of all the send counts. It can be used to calculate how much data is sent during the alltoallv call.
	SendSum int

	// RecvSum is the sum of all the receive counts. It can be used to calculate how much data is received during the alltoallv call.
	RecvSum int

	// SendMin is the minimum send count of the alltoallv call.
	SendMin int

	// RecvMin is the minimum receive count of the alltoallv call.
	RecvMin int

	// SendNotZeroMin is the minimum send count not equal to zero of the alltoallv call.
	SendNotZeroMin int

	// RecvNotZeroMin is the minimum receive count not equal to zero of the alltoallv call.
	RecvNotZeroMin int

	// SencMax is the maximum send count of the alltoallv call.
	SendMax int

	// RecvMax is the maximum receive count of the alltoallv call.
	RecvMax int

	// SendDatatypeSize is the size of the datatype used by the alltoallv call while sending data.
	SendDatatypeSize int

	// RecvDatatypeSize is the size of the datatype used by the alltoallv call while receiving data.
	RecvDatatypeSize int

	// CommSize is the communicator size of the alltoallv call.
	CommSize int

	// MsgSizeThreshold is the size value that differentiate small and large messages.
	MsgSizeThreshold int

	// SendSmallMsgs is the number of small messages sent during the alltoallv call. The size threshold can be adjusted at run time.
	SendSmallMsgs int

	// SendSmallNotZeroMsgs is the number of small messages but not of size 0 that is send during the alltoallv call. The size threshold can be adjusted at run time.
	SendSmallNotZeroMsgs int

	// SendLargeMsgs is the number of large message sent during the alltoallv call. The size threshold can be adjusted at run time.
	SendLargeMsgs int

	// RecvSmallMsgs is the number of small message received during the alltoallv call. The size threshold can be adjusted at run time.
	RecvSmallMsgs int

	// RecvSmallNotZeroMsgs is the number of small message but not of size 0 that are received during the alltoallv call. The size threshold can be adjusted at run time.
	RecvSmallNotZeroMsgs int

	// RecvLargeMsgs is the number of large messages received during the alltoallv call. The size threshold can be adjusted at run time.
	RecvLargeMsgs int

	// TotalSendZeroCounts is the total number of send count equal to zero
	TotalSendZeroCounts int

	// TotalRecvZeroCounts is the total number of receive count equal to zero
	TotalRecvZeroCounts int
}

// CountsStats gathers all the stats from counts (send or receive) for a given alltoallv call
type CountsStats struct {
	// Sum is the total count for all ranks data is sent to or received from
	Sum int

	// MinWithoutZero from the entire counts not including zero
	MinWithoutZero int

	// Min from the entire counts, including zero
	Min int

	// Max from the entire counts
	Max int

	// SmallMsgs is the number of small message from counts, including 0-size count
	SmallMsgs int

	// SmallNotZerroMsgs is the number of small message from counts, not including 0-size counts
	SmallNotZeroMsgs int

	// LargeMsgs is the number of large messages from counts
	LargeMsgs int

	// TotalZeroCounts is the total number of zero counts from counters
	TotalZeroCounts int

	// ZerosPerRankPatterns gathers the number of 0-counts on a per-rank basis ('X ranks have Y 0-counts')
	ZerosPerRankPatterns map[int]int

	// NoZerosPerRankPatterns gathers the number of non-0-counts on a per-rank bases ('X ranks have Y non-0-counts)
	NoZerosPerRankPatterns map[int]int

	// Patterns gathers the number of peers involved in actual communication, i.e., non-zeroa ('X ranks are sendinng to Y ranks')
	Patterns map[int]int

	// MsgSizeThreshold is the message size used to differentiate small messages from larrge messages while parsing the counts
	MsgSizeThreshold int
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

func GetStatsFilePath(basedir string, jobid int, rank int) string {
	return filepath.Join(basedir, fmt.Sprintf("stats-job%d-rank%d.md", jobid, rank))
}

// GetCallData extract all the data related to a specific call.
func GetCallData(dir string, jobid int, rank int, callNum int, msgSizeThreshold int) (CallInfo, error) {
	var info CallInfo
	info.ID = callNum

	// Load the counts from raw data
	log.Printf("Extracting send/receive counts for call #%d\n", callNum)
	sendCountsFile, recvCountsFile := GetCountsFiles(jobid, rank)
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

	info.CommSize, info.SendDatatypeSize, info.SendCounts, err = lookupCallfromCountsFile(sendCountsFileReader, callNum)
	if err != nil {
		return info, nil
	}
	_, info.RecvDatatypeSize, info.RecvCounts, err = lookupCallfromCountsFile(recvCountsFileReader, callNum)
	if err != nil {
		return info, nil
	}
	err = info.getCallStatsFromCounts(msgSizeThreshold)
	if err != nil {
		return info, err
	}

	// Get timings from formatted timing file
	// todo: if the files do not exist, we should get the data from scratch

	log.Printf("Extracting timings for call #%d\n", callNum)
	info.Timings, err = getCallTimings(dir, jobid, rank, callNum)
	if err != nil {
		return info, err
	}
	//info.AlltoallvTimings, info.LateArrivalTiming

	// Load patterns from result file.
	// todo: if the file does not exists, we should get the data from scratch
	log.Printf("Extracting patterns for call #%d\n", callNum)
	info.PatternStr, err = getCallPatterns(dir, jobid, rank, callNum)
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
