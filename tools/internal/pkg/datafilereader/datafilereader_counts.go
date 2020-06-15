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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

func parseCounts(counts []string, msgSizeThreshold int, datatypeSize int) (CountsStats, error) { // (int, int, int, int, int, int, map[int]int, map[int]int, map[int]int, error) {
	var stats CountsStats
	stats.Min = -1
	stats.Max = -1
	stats.MinWithoutZero = -1
	stats.Patterns = make(map[int]int)
	stats.NoZerosPerRankPatterns = make(map[int]int)
	stats.ZerosPerRankPatterns = make(map[int]int)
	stats.Sum = 0
	stats.MsgSizeThreshold = msgSizeThreshold

	zeros := 0
	nonZeros := 0
	//smallMsgs := 0
	smallNotZeroMsgs := 0
	//largeMsgs := 0

	for _, line := range counts {
		tokens := strings.Split(line, ": ")
		c := tokens[0]
		c = strings.ReplaceAll(c, "Rank(s) ", "")
		numberOfRanks, err := notation.GetNumberOfRanksFromCompressedNotation(c)
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
			stats.Sum += count

			if count == 0 {
				zeros++
				stats.TotalZeroCounts += numberOfRanks
			} else {
				nonZeros++
			}

			if msgSizeThreshold != -1 && count*datatypeSize <= msgSizeThreshold {
				stats.SmallMsgs += numberOfRanks
				if count > 0 {
					stats.SmallNotZeroMsgs += numberOfRanks
				}
			}
			if msgSizeThreshold != -1 && count*datatypeSize > msgSizeThreshold {
				stats.LargeMsgs += numberOfRanks
			}

			if stats.Max < count {
				stats.Max = count
			}

			if stats.Min == -1 || (stats.Min != -1 && stats.Min > count) {
				stats.Min = count
			}

			if stats.MinWithoutZero == -1 && count > 0 {
				stats.MinWithoutZero = count
			}

			if stats.MinWithoutZero != -1 && count > 0 && count < stats.MinWithoutZero {
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

	// Get the first line of the header skipping potential empty lines that
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

				callIDs, err = notation.ConvertCompressedCallListToIntSlice(callIDsStr)
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

func (info *CallInfo) getCallStatsFromCounts(msgSizeThreshold int) error {
	sendStats, err := parseCounts(info.SendCounts, msgSizeThreshold, info.SendDatatypeSize)
	if err != nil {
		return err
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
	info.SendSum = sendStats.Sum

	recvStats, err := parseCounts(info.RecvCounts, msgSizeThreshold, info.RecvDatatypeSize)
	if err != nil {
		return err
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
	info.RecvSum = sendStats.Sum

	return nil
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
	sendNumRanks := 0
	sendNumRanks, info.SendDatatypeSize, info.SendCounts, err = lookupCallfromCountsFile(sendCountsReader, numCall)
	if err != nil {
		return info, err
	}

	// find the call's data from the recv counts file then
	recvNumRanks := 0
	recvNumRanks, info.RecvDatatypeSize, info.RecvCounts, err = lookupCallfromCountsFile(recvCountsReader, numCall)
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

	err = info.getCallStatsFromCounts(msgSizeThreshold)
	if err != nil {
		return info, err
	}

	return info, nil
}

func findCountersFilesWithPrefix(basedir string, jobid string, pid string, prefix string) ([]string, error) {
	var files []string

	f, err := ioutil.ReadDir(basedir)
	if err != nil {
		return files, fmt.Errorf("[ERROR] unable to read %s: %w", basedir, err)
	}

	log.Printf("Looking for files from job %s and PID %s\n", jobid, pid)

	for _, file := range f {
		log.Printf("Checking file: %s\n", file.Name())

		if strings.HasPrefix(file.Name(), prefix) && strings.Contains(file.Name(), "pid"+pid) && strings.Contains(file.Name(), "job"+jobid) {
			log.Printf("-> Found a match: %s\n", file.Name())
			path := filepath.Join(basedir, file.Name())
			files = append(files, path)
		}
	}

	return files, nil
}

func extractRankCounters(callCounters []string, rank int) (string, error) {
	//log.Printf("call counters: %s\n", strings.Join(callCounters, "\n"))
	for i := 0; i < len(callCounters); i++ {
		ts := strings.Split(callCounters[i], ": ")
		ranks := ts[0]
		counters := ts[1]
		ranksListStr := strings.Split(strings.ReplaceAll(ranks, "Rank(s) ", ""), " ")
		for j := 0; j < len(ranksListStr); j++ {
			// We may have a list that includes ranges
			tokens := strings.Split(ranksListStr[j], ",")
			for _, t := range tokens {
				tokens2 := strings.Split(t, "-")
				if len(tokens2) == 2 {
					startRank, _ := strconv.Atoi(tokens2[0])
					endRank, _ := strconv.Atoi(tokens2[1])
					if startRank <= rank && rank <= endRank {
						return counters, nil
					}
				} else if len(tokens) == 1 {
					rankID, _ := strconv.Atoi(tokens2[0])
					if rankID == rank {
						return counters, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("unable to find counters for rank %d", rank)
}

func ReadCallRankCounters(files []string, rank int, callNum int) (string, int, bool, error) {
	counters := ""
	found := false
	datatypeSize := 0

	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			return "", datatypeSize, found, fmt.Errorf("unable to open %s: %w", f, err)
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		for {
			_, _, callIDs, _, _, dtSize, readerErr1 := GetHeader(reader)
			datatypeSize = dtSize

			if readerErr1 != nil && readerErr1 != io.EOF {
				fmt.Printf("ERROR: %s", readerErr1)
				return counters, datatypeSize, found, fmt.Errorf("unable to read header from %s: %w", f, readerErr1)
			}

			targetCall := false
			for i := 0; i < len(callIDs); i++ {
				if callIDs[i] == callNum {
					targetCall = true
					break
				}
			}

			var readerErr2 error
			var callCounters []string
			if targetCall == true {
				callCounters, readerErr2 = GetCounters(reader)
				if readerErr2 != nil && readerErr2 != io.EOF {
					return counters, datatypeSize, found, readerErr2
				}

				counters, err = extractRankCounters(callCounters, rank)
				if err != nil {
					return counters, datatypeSize, found, err
				}
				found = true

				return counters, datatypeSize, found, nil
			} else {
				// The current counters are not about the call we care about, skipping...
				_, err := GetCounters(reader)
				if err != nil {
					return counters, datatypeSize, found, err
				}
			}

			if readerErr1 == io.EOF || readerErr2 == io.EOF {
				break
			}
		}
	}

	return counters, datatypeSize, found, fmt.Errorf("unable to find data for rank %d in call %d", rank, callNum)
}

func findSendCountersFiles(basedir string, jobid int, pid int) ([]string, error) {
	pidStr := strconv.Itoa(pid)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, pidStr, sendCountersFilePrefix)
}

func findRecvCountersFiles(basedir string, jobid int, pid int) ([]string, error) {
	pidStr := strconv.Itoa(pid)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, pidStr, recvCountersFilePrefix)
}

func GetCountsFiles(jobid int, pid int) (string, string) {
	suffix := "job" + strconv.Itoa(jobid) + ".pid" + strconv.Itoa(pid) + ".txt"
	return sendCountersFilePrefix + suffix, recvCountersFilePrefix + suffix
}

func findCallRankSendCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, error) {
	files, err := findSendCountersFiles(basedir, jobid, pid)
	if err != nil {
		return "", err
	}
	counters, _, _, err := ReadCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("* unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func findCallRankRecvCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, error) {
	files, err := findRecvCountersFiles(basedir, jobid, pid)
	if err != nil {
		return "", err
	}
	counters, _, _, err := ReadCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func FindCallRankCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, string, error) {
	sendCounters, err := findCallRankSendCounters(basedir, jobid, pid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	recvCounters, err := findCallRankRecvCounters(basedir, jobid, pid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")
	sendCounters = strings.TrimRight(sendCounters, " ")
	recvCounters = strings.TrimRight(recvCounters, " ")

	return sendCounters, recvCounters, nil
}
