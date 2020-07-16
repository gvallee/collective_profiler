//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

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

// GlobalPatterns holds the data all the patterns the infrastructure was able to detect
type GlobalPatterns struct {
	// AllPatterns is the data for all the patterns that have been detected
	AllPatterns []*CallPattern

	// OneToN is the data of all the patterns that fits with a 1 -> N scheme
	OneToN []*CallPattern

	// NToN is the data of all the patterns where N ranks exchange data between all of them
	NToN []*CallPattern

	// NoToOne is the data of all the patterns that fits with a N -> 1 scheme
	NToOne []*CallPattern

	// Empty is the data of all the patterns that do not exchange any data (all counts are equal to 0)
	Empty []*CallPattern
}

// Stats gathers all the data related to send and receive counts for one or more alltoallv call(s)
type Stats struct {
	NumSendSmallMsgs        int
	NumSendLargeMsgs        int
	SizeThreshold           int
	NumSendSmallNotZeroMsgs int

	// BinThresholds is the list of thresholds used to create bins
	BinThresholds []int
	// Bins is the list of bins of counts
	Bins []Bin

	// TotalNumCalls is the number of alltoallv calls covered by the statistics
	TotalNumCalls    int
	CommSizes        map[int]int
	DatatypesSend    map[int]int
	DatatypesRecv    map[int]int
	CallSendSparsity map[int]int
	CallRecvSparsity map[int]int
	SendMins         map[int]int
	RecvMins         map[int]int
	SendMaxs         map[int]int
	RecvMaxs         map[int]int
	SendNotZeroMins  map[int]int
	RecvNotZeroMins  map[int]int
	Patterns         GlobalPatterns
}

// OutputFileInfo gathers all the data for the handling of output files while analysis counts
type OutputFileInfo struct {
	// defaultFd is the file descriptor for the creation of the default output file while analyzing counts
	defaultFd *os.File

	// patternsFd is the file descriptor for the creation of the output files to store patterns discovered during the analysis of the counts
	patternsFd *os.File

	// patternsSummaryFd is the file descriptor for the creation of the summary output file for the patterns discovered during the analysis of the counts
	patternsSummaryFd *os.File

	// defaultOutputFile is the path of the file associated to DefaultFd
	defaultOutputFile string

	// patternsOutputFile is the path of the file associated to PatternsFd
	patternsOutputFile string

	// patternsSummaryOutputFile is the path of the file associated to SummaryPatternsFd
	patternsSummaryOutputFile string

	// Cleanup is the function to call after being done with all the files
	Cleanup func()
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

func getValidationFiles(basedir string, id string) ([]string, error) {
	var files []string

	f, err := ioutil.ReadDir(basedir)
	if err != nil {
		return files, fmt.Errorf("unable to read %s: %w", basedir, err)
	}

	for _, file := range f {
		if strings.HasPrefix(file.Name(), "validation_data-pid"+id) {
			path := filepath.Join(basedir, file.Name())
			files = append(files, path)
		}
	}

	return files, nil
}

func Handle(input string) error {
	a := analyzer.CreateAnalyzer()
	a.InputFile = input

	err := a.Parse()
	if err != nil {
		return err
	}

	a.Finalize()

	return nil
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

func newStats() Stats {
	cs := Stats{
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
		TotalNumCalls:           0,
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

	// Detect specific patterns using the send counts only, e.g., 1->n, n->1 and n->n
	// Note: we do not need to check the receive side because if n ranks are sending to n other ranks,
	// we know that n ranks are receiving from n other ranks with equivalent counts. Send/receive symmetry.
	for sendTo, n := range sendPatterns {
		// Detect 1->n patterns
		if sendTo > n*100 {
			globalPatterns.OneToN = append(globalPatterns.OneToN, new_cp)
			continue
		}

		// Detect n->n patterns
		if sendTo == n {
			globalPatterns.NToN = append(globalPatterns.NToN, new_cp)
			continue
		}

		// Detect n->1 patterns
		if sendTo*100 < n {
			globalPatterns.NToOne = append(globalPatterns.NToOne, new_cp)
			continue
		}
	}

	return nil
}

// ParseFiles parses both send and receive counts files
func ParseFiles(sendCountsFile string, recvCountsFile string, numCalls int, sizeThreshold int) (Stats, error) {
	cs := newStats()
	cs.TotalNumCalls = numCalls

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

		// We need to track calls that act like a barrier (no data exchanged)
		if callInfo.TotalSendNonZeroCounts == 0 && callInfo.TotalRecvNonZeroCounts == 0 {
			emptyPattern := new(CallPattern)
			emptyPattern.Count = 1
			emptyPattern.Calls = []int{i}
			cs.Patterns.Empty = append(cs.Patterns.Empty, emptyPattern)
		}
	}

	return cs, nil
}

func GetCountProfilerFileDesc(basedir string, jobid int, rank int) (OutputFileInfo, error) {
	var info OutputFileInfo
	var err error

	info.defaultOutputFile = datafilereader.GetStatsFilePath(basedir, jobid, rank)
	info.patternsOutputFile = datafilereader.GetPatternFilePath(basedir, jobid, rank)
	info.patternsSummaryOutputFile = datafilereader.GetPatternSummaryFilePath(basedir, jobid, rank)
	info.defaultFd, err = os.OpenFile(info.defaultOutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return info, fmt.Errorf("unable to create %s: %s", info.defaultOutputFile, err)
	}

	info.patternsFd, err = os.OpenFile(info.patternsOutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return info, fmt.Errorf("unable to create %s: %s", info.patternsOutputFile, err)
	}

	info.patternsSummaryFd, err = os.OpenFile(info.patternsSummaryOutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return info, fmt.Errorf("unable to create %s: %s", info.patternsSummaryOutputFile, err)
	}

	info.Cleanup = func() {
		info.defaultFd.Close()
		info.patternsFd.Close()
		info.patternsSummaryFd.Close()
	}

	fmt.Println("Results are saved in:")
	fmt.Printf("-> %s\n", info.defaultOutputFile)
	fmt.Printf("-> %s\n", info.patternsOutputFile)
	fmt.Printf("Patterns summary: %s\n", info.patternsSummaryOutputFile)

	return info, nil
}
