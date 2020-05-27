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
	"reflect"
	"strconv"
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

func CompareCallPatterns(p1 map[int]int, p2 map[int]int) bool {
	if len(p1) != len(p2) {
		return false
	}

	return reflect.DeepEqual(p1, p2)
}

func getNumberOfRanksFromCompressedNotation(str string) (int, error) {
	num := 0
	t1 := strings.Split(str, ", ")
	for _, t := range t1 {
		t2 := strings.Split(t, "-")
		if len(t2) == 2 {
			val1, err := strconv.Atoi(t2[0])
			if err != nil {
				return 0, err
			}
			val2, err := strconv.Atoi(t2[1])
			if err != nil {
				return 0, err
			}
			num += val2 - val1 + 1
		} else {
			num++
		}
	}
	return num, nil
}

func parseCounts(counts []string, msgSizeThreshold int, datatypeSize int) (CountsStats, error) { // (int, int, int, int, int, int, map[int]int, map[int]int, map[int]int, error) {
	var stats CountsStats
	stats.Min = -1
	stats.Max = -1
	stats.MinWithoutZero = -1
	stats.Patterns = make(map[int]int)
	stats.NoZerosPerRankPatterns = make(map[int]int)
	stats.ZerosPerRankPatterns = make(map[int]int)

	zeros := 0
	nonZeros := 0
	//smallMsgs := 0
	smallNotZeroMsgs := 0
	//largeMsgs := 0

	for _, line := range counts {
		tokens := strings.Split(line, ": ")
		c := tokens[0]
		c = strings.ReplaceAll(c, "Rank(s) ", "")
		numberOfRanks, err := getNumberOfRanksFromCompressedNotation(c)
		if err != nil {
			return stats, err
		}

		zeros = 0
		nonZeros = 0
		smallNotZeroMsgs = 0
		//smallMsgs = 0
		//largeMsgs = 0

		words := strings.Split(strings.ReplaceAll(tokens[1], "\n", ""), " ")
		for _, w := range words {
			if w == "" {
				continue
			}
			count, err := strconv.Atoi(w)
			if err != nil {
				log.Printf("unable to parse %s (%s): %s", w, tokens[1], err)
				return stats, err
			}
			if count == 0 {
				zeros++
				stats.TotalZeroCounts += numberOfRanks
			} else {
				nonZeros++
			}

			if count*datatypeSize <= msgSizeThreshold {
				stats.SmallMsgs += numberOfRanks
				if count > 0 {
					stats.SmallNotZeroMsgs += numberOfRanks
				}
			}
			if count*datatypeSize > msgSizeThreshold {
				stats.LargeMsgs += numberOfRanks
			}

			if stats.Max < count {
				stats.Max = count
			}

			if stats.Min == -1 || (stats.Min != -1 && stats.Min > count) {
				stats.Min = count
			}

			if stats.MinWithoutZero == -1 && count >= 0 {
				stats.MinWithoutZero = count
			}

			if stats.MinWithoutZero != -1 && count < stats.MinWithoutZero {
				stats.MinWithoutZero = count
			}
		}

		if nonZeros > 0 {
			if _, ok := stats.Patterns[nonZeros]; ok {
				stats.Patterns[nonZeros] += numberOfRanks
			} else {
				stats.Patterns[nonZeros] = numberOfRanks
			}
		}
		if zeros > 0 {
			if _, ok := stats.ZerosPerRankPatterns[zeros]; ok {
				stats.ZerosPerRankPatterns[zeros] += numberOfRanks
			} else {
				stats.ZerosPerRankPatterns[zeros] = numberOfRanks
			}
		}

		if stats.SmallNotZeroMsgs > 0 {
			if _, ok := stats.NoZerosPerRankPatterns[smallNotZeroMsgs]; ok {
				stats.NoZerosPerRankPatterns[smallNotZeroMsgs] += numberOfRanks
			} else {
				stats.NoZerosPerRankPatterns[smallNotZeroMsgs] = numberOfRanks
			}
		}
	}

	return stats, nil
}

func convertCompressedCallListtoIntSlice(str string) ([]int, error) {
	var callIDs []int

	tokens := strings.Split(str, ", ")
	for _, t := range tokens {
		tokens2 := strings.Split(t, "-")
		if len(tokens2) == 2 {
			val1, err := strconv.Atoi(tokens2[0])
			if err != nil {
				return callIDs, err
			}
			val2, err := strconv.Atoi(tokens2[1])
			if err != nil {
				return callIDs, err
			}
			for i := val1; i <= val2; i++ {
				callIDs = append(callIDs, i)
			}
		} else {
			for _, t2 := range tokens2 {
				n, err := strconv.Atoi(t2)
				if err != nil {
					return callIDs, fmt.Errorf("unable to parse %s", str)
				}
				callIDs = append(callIDs, n)
			}
		}
	}

	return callIDs, nil
}

func GetHeader(reader *bufio.Reader) (int, int, []int, string, int, int, error) {
	var callIDs []int
	numCalls := 0
	callIDsStr := ""
	alltoallvCallStart := -1
	alltoallvCallEnd := -1
	totalNumCalls := 0
	//alltoallvCallNumber := 0
	//alltoallvCallStart := 0
	//alltoallvCallEnd := -1
	line := ""
	var err error
	numRanks := 0
	datatypeSize := 0

	// Get the forst line of the header skipping potential empty lines that
	// can be in front of header
	var readerErr error
	for line == "" || line == "\n" {
		line, readerErr = reader.ReadString('\n')
		if readerErr == io.EOF {
			return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
		if readerErr != nil {
			return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
	}

	// Are we at the beginning of a metadata block?
	if !strings.HasPrefix(line, "# Raw") {
		return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, fmt.Errorf("[ERROR] not a header")
	}

	for {
		line, readerErr = reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}

		if strings.HasPrefix(line, numberOfRanksMarker) {
			line = strings.ReplaceAll(line, numberOfRanksMarker, "")
			line = strings.ReplaceAll(line, "\n", "")
			numRanks, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse number of ranks")
				return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
			}
		}

		if strings.HasPrefix(line, datatypeSizeMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			line = strings.ReplaceAll(line, datatypeSizeMarker, "")
			datatypeSize, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse the datatype size")
				return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
			}
		}

		if strings.HasPrefix(line, alltoallvCallNumbersMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			callRange := strings.ReplaceAll(line, alltoallvCallNumbersMarker, "")
			tokens := strings.Split(callRange, "-")
			if len(tokens) == 2 {
				alltoallvCallStart, err = strconv.Atoi(tokens[0])
				if err != nil {
					log.Println("[ERROR] unable to parse line to get first alltoallv call number")
					return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}
				alltoallvCallEnd, err = strconv.Atoi(tokens[1])
				if err != nil {
					log.Printf("[ERROR] unable to convert %s to interger: %s", tokens[1], err)
					return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}
				totalNumCalls = alltoallvCallEnd - alltoallvCallStart + 1 // Add 1 because we are 0-indexed
			}
		}

		if strings.HasPrefix(line, countMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			strParsing := line
			tokens := strings.Split(line, " - ")
			if len(tokens) > 1 {
				strParsing = tokens[0]
				callIDsStr = tokens[1]
				tokens2 := strings.Split(callIDsStr, " (")
				if len(tokens2) > 1 {
					callIDsStr = tokens2[0]
				}
			}

			strParsing = strings.ReplaceAll(strParsing, countMarker, "")
			strParsing = strings.ReplaceAll(strParsing, " calls", "")
			numCalls, err = strconv.Atoi(strParsing)
			if err != nil {
				log.Println("[ERROR] unable to parse line to get #s of alltoallv calls")
				return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
			}

			if callIDsStr != "" {
				/*
					tokens := strings.Split(callIDsStr, " ")
					for _, t := range tokens {
						if t == "..." {
							incompleteData = true
						}
						if t != "" && t != "..." { // '...' means that we have a few more calls that are involved but we do not know what they are
							n, err := strconv.Atoi(t)
							if err != nil {
								log.Fatalf("unable to parse '%s' - '%s'\n", callIDsStr, t)
								return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, incompleteData, err
							}
							callIDs = append(callIDs, n)
						}
					}
				*/

				callIDs, err = convertCompressedCallListtoIntSlice(callIDsStr)
				if err != nil {
					log.Printf("[ERROR] unable to parse calls IDs: %s", err)
					return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}

			}
		}

		// We check for the beginning of the actual data
		if strings.HasPrefix(line, beginningDataMarker) {
			break
		}

		if readerErr == io.EOF {
			return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
	}

	/*
		if numCalls != alltoallvCallNumber {
			return numCalls, callIDs, callIDsStr, fmt.Errorf("[ERROR] Inconsistent metadata, number of calls differs (%d vs. %d)", numCalls, alltoallvCallNumber)
		}
	*/

	return totalNumCalls, numCalls, callIDs, callIDsStr, numRanks, datatypeSize, nil
}

func GetCounters(reader *bufio.Reader) ([]string, error) {
	var callCounters []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return callCounters, err
		}

		if strings.Contains(line, "END DATA") {
			break
		}

		callCounters = append(callCounters, line)
	}

	return callCounters, nil
}

func lookupCallfromCountsFile(reader *bufio.Reader, numCall int) (int, int, []string, error) {
	var counts []string
	var err error
	var callIDs []int

	numRanks := 0
	datatypeSize := 0

	for {
		_, _, callIDs, _, numRanks, datatypeSize, err = GetHeader(reader)
		if err != nil {
			return numRanks, datatypeSize, counts, fmt.Errorf("unable to read header: %s", err)
		}
		for _, i := range callIDs {
			if i == numCall {
				counts, err = GetCounters(reader)
				return numRanks, datatypeSize, counts, nil
			}
		}

		// We do not need these counts but we still read them to find the next header
		_, err := GetCounters(reader)
		if err != nil {
			return numRanks, datatypeSize, counts, fmt.Errorf("unable to parse file: %s", err)
		}
	}
}

func LookupCall(sendCountsFile string, recvCountsFile string, numCall int, msgSizeThreshold int) (CallInfo, error) {
	var info CallInfo

	fSendCounts, err := os.Open(sendCountsFile)
	if err != nil {
		return info, fmt.Errorf("unable to open %s: %s", sendCountsFile, err)
	}
	defer fSendCounts.Close()
	fRecvCounts, err := os.Open(recvCountsFile)
	if err != nil {
		return info, fmt.Errorf("unable to open %s: %s", recvCountsFile, err)
	}
	defer fRecvCounts.Close()

	sendCountsReader := bufio.NewReader(fSendCounts)
	recvCountsReader := bufio.NewReader(fRecvCounts)

	// find the call's data from the send counts file first
	var sendCounts []string
	sendNumRanks := 0
	sendNumRanks, info.SendDatatypeSize, sendCounts, err = lookupCallfromCountsFile(sendCountsReader, numCall)
	if err != nil {
		return info, err
	}

	// find the call's data from the recv counts file then
	var recvCounts []string
	recvNumRanks := 0
	recvNumRanks, info.RecvDatatypeSize, recvCounts, err = lookupCallfromCountsFile(recvCountsReader, numCall)
	if err != nil {
		return info, err
	}

	if sendNumRanks != recvNumRanks {
		return info, fmt.Errorf("different communicator sizes for send and recv data")
	}
	info.CommSize = sendNumRanks

	var cp CallPattern
	cp.SendZeroCounts = make(map[int]int)
	cp.RecvZeroCounts = make(map[int]int)
	//info.SendNotZeroMin, info.SendMin, info.SendMax, info.SendSmallMsgs, info.SendSmallNotZeroMsgs, info.SendLargeMsgs, info.Patterns.SendZeroCounts, info.Patterns.SendNotZeroCounts, info.Patterns.SendPatterns, err = parseCounts(sendCounts, msgSizeThreshold, info.SendDatatypeSize)
	sendStats, err := parseCounts(sendCounts, msgSizeThreshold, info.SendDatatypeSize)
	if err != nil {
		return info, err
	}

	info.Patterns.SendPatterns = sendStats.Patterns
	info.Patterns.SendZeroCounts = sendStats.ZerosPerRankPatterns
	info.Patterns.SendNotZeroCounts = sendStats.NoZerosPerRankPatterns
	info.SendLargeMsgs = sendStats.LargeMsgs
	info.SendMax = sendStats.Max
	info.SendMin = sendStats.Min
	info.SendNotZeroMin = sendStats.MinWithoutZero
	info.SendSmallMsgs = sendStats.SmallMsgs
	info.SendSmallNotZeroMsgs = sendStats.SmallNotZeroMsgs
	info.TotalSendZeroCounts = sendStats.TotalZeroCounts

	//info.RecvNotZeroMin, info.RecvMin, info.RecvMax, info.RecvSmallMsgs, info.RecvSmallNotZeroMsgs, info.RecvLargeMsgs, info.Patterns.RecvZeroCounts, info.Patterns.RecvNotZeroCounts, info.Patterns.RecvPatterns, err = parseCounts(recvCounts, msgSizeThreshold, info.RecvDatatypeSize)
	recvStats, err := parseCounts(recvCounts, msgSizeThreshold, info.RecvDatatypeSize)
	if err != nil {
		return info, err
	}

	info.Patterns.RecvPatterns = recvStats.Patterns
	info.Patterns.RecvZeroCounts = recvStats.ZerosPerRankPatterns
	info.Patterns.RecvNotZeroCounts = recvStats.NoZerosPerRankPatterns
	info.RecvNotZeroMin = recvStats.MinWithoutZero
	info.RecvMin = recvStats.Min
	info.RecvMax = recvStats.Max
	info.RecvSmallMsgs = recvStats.SmallMsgs
	info.RecvSmallNotZeroMsgs = recvStats.SmallNotZeroMsgs
	info.RecvLargeMsgs = recvStats.LargeMsgs
	info.TotalRecvZeroCounts = recvStats.TotalZeroCounts

	return info, nil
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
