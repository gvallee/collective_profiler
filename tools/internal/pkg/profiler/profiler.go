//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package profiler

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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
)

type Bin struct {
	Min  int
	Max  int
	Size int
}

type CallPattern struct {
	Send  map[int]int
	Recv  map[int]int
	Count int
	Calls []int
}

type GlobalPatterns struct {
	AllPatterns []*CallPattern
	OneToN      []*CallPattern
}

type CountStats struct {
	NumSendSmallMsgs        int
	NumSendLargeMsgs        int
	SizeThreshold           int
	NumSendSmallNotZeroMsgs int
	CommSizes               map[int]int
	DatatypesSend           map[int]int
	DatatypesRecv           map[int]int
	CallSendSparsity        map[int]int
	CallRecvSparsity        map[int]int
	SendMins                map[int]int
	RecvMins                map[int]int
	SendMaxs                map[int]int
	RecvMaxs                map[int]int
	SendNotZeroMins         map[int]int
	RecvNotZeroMins         map[int]int
	Patterns                GlobalPatterns
}

func containsCall(callNum int, calls []int) bool {
	for i := 0; i < len(calls); i++ {
		if calls[i] == callNum {
			return true
		}
	}
	return false
}

func HandleCounters(input string) error {
	a := analyzer.CreateAnalyzer()
	a.InputFile = input

	err := a.Parse()
	if err != nil {
		return err
	}

	a.Finalize()

	return nil
}

func getValidationFiles(basedir string, id string) ([]string, error) {
	var files []string

	f, err := ioutil.ReadDir(basedir)
	if err != nil {
		return files, fmt.Errorf("[ERROR] unable to read %s: %w", basedir, err)
	}

	for _, file := range f {
		if strings.HasPrefix(file.Name(), "validation_data-pid"+id) {
			path := filepath.Join(basedir, file.Name())
			files = append(files, path)
		}
	}

	return files, nil
}

func getInfoFromFilename(path string) (int, int, int, error) {
	filename := filepath.Base(path)
	filename = strings.ReplaceAll(filename, "validation_data-", "")
	filename = strings.ReplaceAll(filename, ".txt", "")
	tokens := strings.Split(filename, "-")
	if len(tokens) != 3 {
		return -1, -1, -1, fmt.Errorf("filename has the wrong format")
	}
	idStr := tokens[0]
	rankStr := tokens[1]
	callStr := tokens[2]

	idStr = strings.ReplaceAll(idStr, "pid", "")
	rankStr = strings.ReplaceAll(rankStr, "rank", "")
	callStr = strings.ReplaceAll(callStr, "call", "")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("unable to convert %s: %w", idStr, err)
	}

	rank, err := strconv.Atoi(rankStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("unable to convert %s: %w", rankStr, err)
	}

	call, err := strconv.Atoi(callStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("unable to convert %s: %w", callStr, err)
	}

	return id, rank, call, nil
}

func getCountersFromValidationFile(path string) (string, string, error) {

	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("unable to open %s: %w", path, err)
	}
	defer file.Close()

	sendCounters := ""
	recvCounters := ""

	reader := bufio.NewReader(file)
	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			fmt.Printf("ERROR: %s", readerErr)
			return "", "", fmt.Errorf("unable to read header from %s: %w", path, readerErr)
		}

		if line != "" && line != "\n" {
			if sendCounters == "" {
				sendCounters = line
			} else if recvCounters == "" {
				recvCounters = line
			} else {
				return "", "", fmt.Errorf("invalid file format")
			}
		}

		if readerErr == io.EOF {
			break
		}
	}

	if sendCounters == "" || recvCounters == "" {
		return "", "", fmt.Errorf("unable to load send and receive counters from %s", path)
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")
	sendCounters = strings.TrimRight(sendCounters, " ")
	recvCounters = strings.TrimRight(recvCounters, " ")

	return sendCounters, recvCounters, nil
}

func Validate(jobid int, pid int, dir string) error {
	// Find all the data randomly generated during the execution of the app
	idStr := strconv.Itoa(pid)
	files, err := getValidationFiles(dir, idStr)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d files with data for validation\n", len(files))

	// For each file, load the counters with our framework and compare with the data we got directly from the app
	for _, f := range files {
		_, rank, call, err := getInfoFromFilename(f)
		if err != nil {
			return err
		}

		log.Printf("Looking up counters for rank %d during call %d\n", rank, call)
		sendCounters1, recvCounters1, err := getCountersFromValidationFile(f)
		if err != nil {
			fmt.Printf("unable to get counters from validation data: %s", err)
			return err
		}

		sendCounters2, recvCounters2, err := datafilereader.FindCallRankCounters(dir, jobid, rank, call)
		if err != nil {
			fmt.Printf("unable to get counters: %s", err)
			return err
		}

		if sendCounters1 != sendCounters2 {
			return fmt.Errorf("Send counters do not match with %s: expected '%s' but got '%s'\nReceive counts are: %s vs. %s", filepath.Base(f), sendCounters1, sendCounters2, recvCounters1, recvCounters2)
		}

		if recvCounters1 != recvCounters2 {
			return fmt.Errorf("Receive counters do not match %s: expected '%s' but got '%s'\nSend counts are: %s vs. %s", filepath.Base(f), recvCounters1, recvCounters2, sendCounters1, sendCounters2)
		}

		fmt.Printf("File %s validated\n", filepath.Base(f))
	}

	return nil
}

func GetCallRankData(sendCountersFile string, recvCountersFile string, callNum int, rank int) (int, int, error) {
	sendCounters, sendDatatypeSize, _, err := datafilereader.ReadCallRankCounters([]string{sendCountersFile}, rank, callNum)
	if err != nil {
		return 0, 0, err
	}
	recvCounters, recvDatatypeSize, _, err := datafilereader.ReadCallRankCounters([]string{recvCountersFile}, rank, callNum)
	if err != nil {
		return 0, 0, err
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")

	// We parse the send counters to know how much data is being sent
	sendSum := 0
	tokens := strings.Split(sendCounters, " ")
	for _, t := range tokens {
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			return 0, 0, err
		}
		sendSum += n
	}
	sendSum = sendSum * sendDatatypeSize

	// We parse the recv counters to know how much data is being received
	recvSum := 0
	tokens = strings.Split(recvCounters, " ")
	for _, t := range tokens {
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			return 0, 0, err
		}
		recvSum += n
	}
	recvSum = recvSum * recvDatatypeSize

	return sendSum, recvSum, nil
}

func createBins(listBins []int) []Bin {
	var bins []Bin

	start := 0
	end := listBins[0]
	for i := 0; i < len(listBins)+1; i++ {
		var b Bin
		b.Min = start
		b.Max = end
		b.Size = 0

		start = end
		if i+1 < len(listBins) {
			end = listBins[i+1]
		} else {
			end = -1 // Means no max
		}

		bins = append(bins, b)
	}

	return bins
}

func GetBins(countFilePath string, listBins []int) ([]Bin, error) {
	log.Printf("Creating bins out of values from %s\n", countFilePath)

	bins := createBins(listBins)
	log.Printf("Successfully initialized %d bins\n", len(bins))

	f, err := os.Open(countFilePath)
	if err != nil {
		return bins, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		_, numCalls, _, _, _, datatypeSize, readerr := datafilereader.GetHeader(reader)
		if readerr == io.EOF {
			break
		}
		if readerr != nil {
			return bins, readerr
		}

		counters, err := datafilereader.GetCounters(reader)
		if err != nil {
			return bins, err
		}
		for _, c := range counters {
			tokens := strings.Split(c, ": ")
			ranks := tokens[0]
			counts := strings.TrimRight(tokens[1], "\n")
			ranks = strings.TrimLeft(ranks, "Rank(s) ")
			listRanks, err := notation.ConvertCompressedCallListToIntSlice(ranks)
			if err != nil {
				return bins, err
			}
			nRanks := len(listRanks)

			// Now we parse the counts one by one
			for _, oneCount := range strings.Split(counts, " ") {
				if oneCount == "" {
					continue
				}

				countVal, err := strconv.Atoi(oneCount)
				if err != nil {
					return bins, err
				}

				val := countVal * datatypeSize
				for i := 0; i < len(bins); i++ {
					if (bins[i].Max != -1 && bins[i].Min <= val && val < bins[i].Max) || (bins[i].Max == -1 && val >= bins[i].Min) {
						bins[i].Size += numCalls * nRanks
						break
					}
				}
			}
		}
	}

	return bins, nil
}

func newCountStats() CountStats {
	cs := CountStats{
		CommSizes:               make(map[int]int),
		DatatypesSend:           make(map[int]int),
		DatatypesRecv:           make(map[int]int),
		SendMins:                make(map[int]int),
		RecvMins:                make(map[int]int),
		SendMaxs:                make(map[int]int),
		RecvMaxs:                make(map[int]int),
		RecvNotZeroMins:         make(map[int]int),
		SendNotZeroMins:         make(map[int]int),
		CallSendSparsity:        make(map[int]int),
		CallRecvSparsity:        make(map[int]int),
		NumSendSmallMsgs:        0,
		NumSendSmallNotZeroMsgs: 0,
		NumSendLargeMsgs:        0,
	}
	return cs
}

func (globalPatterns *GlobalPatterns) addPattern(callNum int, sendPatterns map[int]int, recvPatterns map[int]int) error {
	for idx, x := range globalPatterns.AllPatterns {
		if datafilereader.CompareCallPatterns(x.Send, sendPatterns) && datafilereader.CompareCallPatterns(x.Recv, recvPatterns) {
			// Increment count for pattern
			log.Printf("-> Alltoallv call #%d - Adding alltoallv to pattern %d...\n", callNum, idx)
			x.Count++
			x.Calls = append(x.Calls, callNum)

			// todo: We may want to track 1 -> N more independently but right now, we handle pointers
			// so the details are only about the main list.
			/*
				if sentTo > n*100 {
					// This is also a 1->n pattern and we need to update the list of such patterns
					for _, candidatePattern := range globalPatterns.oneToN {
						if datafilereader.CompareCallPatterns(candidatePattern.send, sendPatterns) && datafilereader.CompareCallPatterns(candidatePattern.recv, recvPatterns) {
							candidatePattern.count ++
						}
					}
				}
			*/
			return nil
		}
	}

	// If we get here, it means that we did not find a similar pattern
	log.Printf("-> Alltoallv call %d - Adding new pattern...\n", callNum)
	new_cp := new(CallPattern)
	new_cp.Send = sendPatterns
	new_cp.Recv = recvPatterns
	new_cp.Count = 1
	new_cp.Calls = append(new_cp.Calls, callNum)
	globalPatterns.AllPatterns = append(globalPatterns.AllPatterns, new_cp)

	// Detect 1 -> n patterns using the send counts only
	for sendTo, n := range sendPatterns {
		if sendTo > n*100 {
			globalPatterns.OneToN = append(globalPatterns.OneToN, new_cp)
		}
	}

	return nil
}

func ParseCountFiles(sendCountsFile string, recvCountsFile string, numCalls int, sizeThreshold int) (CountStats, error) {
	cs := newCountStats()

	for i := 0; i < numCalls; i++ {
		log.Printf("Analyzing call #%d\n", i)
		callInfo, err := datafilereader.LookupCall(sendCountsFile, recvCountsFile, i, sizeThreshold)
		if err != nil {
			return cs, err
		}

		cs.NumSendSmallMsgs += callInfo.SendSmallMsgs
		cs.NumSendSmallNotZeroMsgs += callInfo.SendSmallNotZeroMsgs
		cs.NumSendLargeMsgs += callInfo.SendLargeMsgs

		if _, ok := cs.DatatypesSend[callInfo.SendDatatypeSize]; ok {
			cs.DatatypesSend[callInfo.SendDatatypeSize]++
		} else {
			cs.DatatypesSend[callInfo.SendDatatypeSize] = 1
		}

		if _, ok := cs.DatatypesRecv[callInfo.RecvDatatypeSize]; ok {
			cs.DatatypesRecv[callInfo.RecvDatatypeSize]++
		} else {
			cs.DatatypesRecv[callInfo.RecvDatatypeSize] = 1
		}

		if _, ok := cs.CommSizes[callInfo.CommSize]; ok {
			cs.CommSizes[callInfo.CommSize]++
		} else {
			cs.CommSizes[callInfo.CommSize] = 1
		}

		if _, ok := cs.SendMins[callInfo.SendMin]; ok {
			cs.SendMins[callInfo.SendMin]++
		} else {
			cs.SendMins[callInfo.SendMin] = 1
		}

		if _, ok := cs.RecvMins[callInfo.RecvMin]; ok {
			cs.RecvMins[callInfo.RecvMin]++
		} else {
			cs.RecvMins[callInfo.RecvMin] = 1
		}

		if _, ok := cs.SendMaxs[callInfo.SendMax]; ok {
			cs.SendMaxs[callInfo.SendMax]++
		} else {
			cs.SendMaxs[callInfo.SendMax] = 1
		}

		if _, ok := cs.RecvMaxs[callInfo.RecvMax]; ok {
			cs.RecvMaxs[callInfo.RecvMax]++
		} else {
			cs.RecvMaxs[callInfo.RecvMax] = 1
		}

		if _, ok := cs.SendNotZeroMins[callInfo.SendNotZeroMin]; ok {
			cs.SendMins[callInfo.SendNotZeroMin]++
		} else {
			cs.SendMins[callInfo.SendNotZeroMin] = 1
		}

		if _, ok := cs.RecvNotZeroMins[callInfo.RecvNotZeroMin]; ok {
			cs.RecvMins[callInfo.RecvNotZeroMin]++
		} else {
			cs.RecvMins[callInfo.RecvNotZeroMin] = 1
		}

		if _, ok := cs.CallSendSparsity[callInfo.TotalSendZeroCounts]; ok {
			cs.CallSendSparsity[callInfo.TotalSendZeroCounts]++
		} else {
			cs.CallSendSparsity[callInfo.TotalSendZeroCounts] = 1
		}

		if _, ok := cs.CallRecvSparsity[callInfo.TotalRecvZeroCounts]; ok {
			cs.CallRecvSparsity[callInfo.TotalRecvZeroCounts]++
		} else {
			cs.CallRecvSparsity[callInfo.TotalRecvZeroCounts] = 1
		}

		//displayCallPatterns(callInfo)
		// Analyze the send/receive pattern from the call
		err = cs.Patterns.addPattern(i, callInfo.Patterns.SendPatterns, callInfo.Patterns.RecvPatterns)
		if err != nil {
			return cs, err
		}
	}

	return cs, nil
}
