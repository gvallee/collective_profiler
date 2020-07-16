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
	"sort"
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

// CountStats gathers all the data related to send and receive counts for one or more alltoallv call(s)
type CountStats struct {
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

// getBins parses a count file using a provided reader and classify all counts
// into bins based on the threshold specified through a slice of integers.
func getBins(reader *bufio.Reader, listBins []int) ([]Bin, error) {
	bins := createBins(listBins)
	log.Printf("Successfully initialized %d bins\n", len(bins))

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

// GetBinsFromCountFile opens a count file and classify all counts into bins
// based on a list of threshold sizes
func GetBinsFromCountFile(countFilePath string, listBins []int) ([]Bin, error) {
	log.Printf("Creating bins out of values from %s\n", countFilePath)

	f, err := os.Open(countFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	return getBins(reader, listBins)
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

// ParseCountFiles parses both send and receive counts files
func ParseCountFiles(sendCountsFile string, recvCountsFile string, numCalls int, sizeThreshold int) (CountStats, error) {
	cs := newCountStats()
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

func writeDataPatternToFile(fd *os.File, cp *CallPattern) error {
	for sendTo, n := range cp.Send {
		_, err := fd.WriteString(fmt.Sprintf("%d ranks sent to %d other ranks\n", n, sendTo))
		if err != nil {
			return err
		}
	}
	for recvFrom, n := range cp.Recv {
		_, err := fd.WriteString(fmt.Sprintf("%d ranks recv'd from %d other ranks\n", n, recvFrom))
		if err != nil {
			return err
		}
	}
	return nil
}

func writePatternsToFile(fd *os.File, num int, totalNumCalls int, cp *CallPattern) error {
	_, err := fd.WriteString(fmt.Sprintf("## Pattern #%d (%d/%d alltoallv calls)\n\n", num, cp.Count, totalNumCalls))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("Alltoallv calls: %s\n", notation.CompressIntArray(cp.Calls)))
	if err != nil {
		return err
	}

	err = writeDataPatternToFile(fd, cp)
	if err != nil {
		return err
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func writeDatatypeToFile(fd *os.File, numCalls int, datatypesSend map[int]int, datatypesRecv map[int]int) error {
	_, err := fd.WriteString("# Datatypes\n\n")
	if err != nil {
		return err
	}
	for datatypeSize, n := range datatypesSend {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while sending data\n", n, numCalls, datatypeSize))
		if err != nil {
			return err
		}
	}
	for datatypeSize, n := range datatypesRecv {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while receiving data\n", n, numCalls, datatypeSize))
		if err != nil {
			return err
		}
	}
	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func writeCommunicatorSizesToFile(fd *os.File, numCalls int, commSizes map[int]int) error {
	_, err := fd.WriteString("# Communicator size(s)\n\n")
	if err != nil {
		return err
	}
	for commSize, n := range commSizes {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls use a communicator size of %d\n", n, numCalls, commSize))
		if err != nil {
			return err
		}
	}
	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

func writeCountStatsToFile(fd *os.File, numCalls int, sizeThreshold int, cs CountStats) error {
	_, err := fd.WriteString("# Message sizes\n\n")
	if err != nil {
		return err
	}
	totalSendMsgs := cs.NumSendSmallMsgs + cs.NumSendLargeMsgs
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are large (threshold = %d)\n", cs.NumSendLargeMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small (threshold = %d)\n", cs.NumSendSmallMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small, but not 0-size (threshold = %d)\n", cs.NumSendSmallNotZeroMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}

	_, err = fd.WriteString("\n# Sparsity\n\n")
	if err != nil {
		return err
	}
	for numZeros, nCalls := range cs.CallSendSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d send counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}
	for numZeros, nCalls := range cs.CallRecvSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d recv counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n# Min/max\n")
	if err != nil {
		return err
	}
	for mins, n := range cs.SendMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.RecvMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for mins, n := range cs.SendNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.RecvNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for maxs, n := range cs.SendMaxs {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count max of %d\n", n, numCalls, maxs))
		if err != nil {
			return err
		}
	}
	for maxs, n := range cs.RecvMaxs {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count max of %d\n", n, numCalls, maxs))
		if err != nil {
			return err
		}
	}

	return nil
}

func noPatternsSummary(cs CountStats) bool {
	if len(cs.Patterns.OneToN) != 0 {
		return false
	}

	if len(cs.Patterns.NToOne) != 0 {
		return false
	}

	if len(cs.Patterns.NToN) != 0 {
		return false
	}

	return true
}

func SaveCounterStats(info OutputFileInfo, cs CountStats, numCalls int, sizeThreshold int) error {
	_, err := info.defaultFd.WriteString(fmt.Sprintf("Total number of alltoallv calls: %d\n\n", numCalls))
	if err != nil {
		return err
	}

	err = writeDatatypeToFile(info.defaultFd, numCalls, cs.DatatypesSend, cs.DatatypesRecv)
	if err != nil {
		return err
	}

	err = writeCommunicatorSizesToFile(info.defaultFd, numCalls, cs.CommSizes)
	if err != nil {
		return err
	}

	err = writeCountStatsToFile(info.defaultFd, numCalls, sizeThreshold, cs)
	if err != nil {
		return err
	}

	_, err = info.patternsFd.WriteString("# Patterns\n")
	if err != nil {
		return err
	}
	num := 0
	for _, cp := range cs.Patterns.AllPatterns {
		err = writePatternsToFile(info.patternsFd, num, numCalls, cp)
		if err != nil {
			return err
		}
		num++
	}

	if !noPatternsSummary(cs) {
		if len(cs.Patterns.OneToN) != 0 {
			_, err := info.patternsSummaryFd.WriteString("# 1 to N patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.OneToN {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
				num++
			}
		}

		if len(cs.Patterns.NToOne) != 0 {
			_, err := info.patternsSummaryFd.WriteString("\n# N to 1 patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.NToOne {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
			}
		}

		if len(cs.Patterns.NToN) != 0 {
			_, err := info.patternsSummaryFd.WriteString("\n# N to n patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.NToN {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
			}
		}
	} else {
		_, err = info.patternsSummaryFd.WriteString("Nothing special detected; no summary")
		if err != nil {
			return err
		}
	}

	return nil
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

func ParseTimingsFile(filePath string, outputDir string) error {
	lateArrivalFilename := strings.ReplaceAll(filepath.Base(filePath), "timings", "late_arrival_timings")
	lateArrivalFilename = strings.ReplaceAll(lateArrivalFilename, ".md", ".dat")
	a2aFilename := strings.ReplaceAll(filepath.Base(filePath), "timings", "alltoallv_timings")
	a2aFilename = strings.ReplaceAll(a2aFilename, ".md", ".dat")
	if outputDir != "" {
		lateArrivalFilename = filepath.Join(outputDir, lateArrivalFilename)
		a2aFilename = filepath.Join(outputDir, a2aFilename)
	}

	err := datafilereader.ExtractTimings(filePath, lateArrivalFilename, a2aFilename)
	if err != nil {
		return err
	}

	return nil
}

/*
func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
*/

// patternIsInList checks whether a given pattern is in a list of patterns. If so, it returns the
// number of alltoallv calls that have the pattern, otherwise it returns 0
func patternIsInList(numPeers int, numRanks int, ctx string, patterns []*CallPattern) int {
	for _, p := range patterns {
		if ctx == "SEND" {
			for numP, numR := range p.Send {
				if numP == numP && numRanks == numR {
					return p.Count
				}
			}
		} else {
			for numP, numR := range p.Recv {
				if numP == numP && numRanks == numR {
					return p.Count
				}
			}
		}
	}
	return 0
}

func sameListOfPatterns(patterns1, patterns2 []*CallPattern) bool {
	// reflect.DeepEqual cannot be used here

	// Compare send counts
	for _, p1 := range patterns1 {
		for numPeers, numRanks := range p1.Send {
			count := patternIsInList(numPeers, numRanks, "SEND", patterns2)
			if count == 0 {
				return false
			}
			if p1.Count != count {
				log.Printf("Send counts differ: %d vs. %d", p1.Count, count)
			}
		}
	}

	// Compare recv counts
	for _, p1 := range patterns1 {
		for numPeers, numRanks := range p1.Recv {
			count := patternIsInList(numPeers, numRanks, "RECV", patterns2)
			if count == 0 {
				return false
			}
			if p1.Count != count {
				log.Printf("Recv counts differ: %d vs. %d", p1.Count, count)
			}
		}
	}

	return true
}

func samePatterns(patterns1, patterns2 GlobalPatterns) bool {
	return sameListOfPatterns(patterns1.AllPatterns, patterns2.AllPatterns)
}

func displayPatterns(pattern []*CallPattern) {
	for _, p := range pattern {
		for numPeers, numRanks := range p.Send {
			fmt.Printf("%d ranks are sending to %d other ranks\n", numRanks, numPeers)
		}
		for numPeers, numRanks := range p.Recv {
			fmt.Printf("%d ranks are receiving from %d other ranks\n", numRanks, numPeers)
		}
	}
}

func writeSubcommNtoNPatterns(fd *os.File, ranks []int, stats map[int]CountStats) error {
	_, err := fd.WriteString("## N to n patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.NToN {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.NToN {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

func writeSubcomm1toNPatterns(fd *os.File, ranks []int, stats map[int]CountStats) error {
	_, err := fd.WriteString("## 1 to n patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.OneToN {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.OneToN {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

func writeSubcommNto1Patterns(fd *os.File, ranks []int, stats map[int]CountStats) error {
	_, err := fd.WriteString("## N to 1 patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.NToOne {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.NToOne {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

// AnalyzeSubCommsResults go through the results and analyzes results specific
// to sub-communicators cases
func AnalyzeSubCommsResults(dir string, stats map[int]CountStats) error {
	numPatterns := -1
	numNtoNPatterns := -1
	num1toNPatterns := -1
	numNto1Patterns := -1
	var referencePatterns GlobalPatterns

	// At the moment, we do a very basic analysis: are the patterns the same on all sub-communicators?
	for _, rankStats := range stats {
		if numPatterns == -1 {
			numPatterns = len(rankStats.Patterns.AllPatterns)
			numNto1Patterns = len(rankStats.Patterns.NToOne)
			numNtoNPatterns = len(rankStats.Patterns.NToN)
			num1toNPatterns = len(rankStats.Patterns.OneToN)
			referencePatterns = rankStats.Patterns
			continue
		}

		if numPatterns != len(rankStats.Patterns.AllPatterns) ||
			numNto1Patterns != len(rankStats.Patterns.NToOne) ||
			numNtoNPatterns != len(rankStats.Patterns.NToN) ||
			num1toNPatterns != len(rankStats.Patterns.OneToN) {
			return nil
		}

		if !samePatterns(referencePatterns, rankStats.Patterns) {
			/*
				fmt.Println("Patterns differ:")
				displayPatterns(referencePatterns.AllPatterns)
				fmt.Printf("\n")
				displayPatterns(rankStats.Patterns.AllPatterns)
			*/
			return nil
		}
	}

	// If we get there it means all ranks, i.e., sub-communicators have the same amount of patterns
	log.Println("All patterns on all sub-communicators are similar")
	multicommHighlightFile := filepath.Join(dir, datafilereader.MulticommHighlightFilePrefix+".md")
	fd, err := os.OpenFile(multicommHighlightFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.WriteString("Alltoallv on sub-communicators detected.\n\n# Patterns summary\n\n")
	if err != nil {
		return err
	}

	var ranks []int
	for r := range stats {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)

	if len(stats[ranks[0]].Patterns.NToN) > 0 {
		err := writeSubcommNtoNPatterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	if len(stats[ranks[0]].Patterns.OneToN) > 0 {
		err := writeSubcomm1toNPatterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	if len(stats[ranks[0]].Patterns.NToOne) > 0 {
		err := writeSubcommNto1Patterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n## All 0 counts pattern; no data exchanged\n\n")
	if err != nil {
		return err
	}
	for _, rank := range ranks {
		if len(stats[rank].Patterns.Empty) > 0 {
			_, err = fd.WriteString(fmt.Sprintf("-> Sub-communicator led by rank %d: %d/%d alltoallv calls\n", rank, len(stats[rank].Patterns.Empty), stats[rank].TotalNumCalls))
			if err != nil {
				return err
			}
		}
	}

	// For now we save the bins' data separately because we do not have a good way at the moment
	// to mix bins and patterns (bins are specific to a count file, not a call; we could change that
	// but it would take time).
	_, err = fd.WriteString("\n# Counts analysis\n\n")
	if err != nil {
		return err
	}
	for _, rank := range ranks {
		_, err := fd.WriteString(fmt.Sprintf("-> Sub-communicator led by rank %d:\n", rank))
		if err != nil {
			return err
		}
		for _, b := range stats[rank].Bins {
			if b.Max != -1 {
				_, err := fd.WriteString(fmt.Sprintf("\t%d of the messages are of size between %d and %d bytes\n", b.Size, b.Min, b.Max-1))
				if err != nil {
					return err
				}
			} else {
				_, err := fd.WriteString(fmt.Sprintf("\t%d of messages are larger or equal of %d bytes\n", b.Size, b.Min))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getBinOutputFile(dir string, jobid, rank int, b Bin) string {
	outputFile := fmt.Sprintf("bin.job%d.rank%d_%d-%d.txt", jobid, rank, b.Min, b.Max)
	if b.Max == -1 {
		outputFile = fmt.Sprintf("bin.job%d.rank%d_%d+.txt", jobid, rank, b.Min)
	}
	if dir != "" {
		outputFile = filepath.Join(dir, outputFile)
	}

	return outputFile
}

// SaveBins writes the data of all the bins into output file. The output files
// are created in a target output directory.
func SaveBins(dir string, jobid, rank int, bins []Bin) error {
	for _, b := range bins {
		outputFile := getBinOutputFile(dir, jobid, rank, b)
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("unable to create file %s: %s", outputFile, err)
		}

		_, err = f.WriteString(fmt.Sprintf("%d\n", b.Size))
		if err != nil {
			return fmt.Errorf("unable to write bin to file: %s", err)
		}
	}
	return nil
}

// GetBinsFromInputDescr parses the string describing a series of threshold to use
// for the organization of data into bins and returns a slice of int with each
// element being a threshold
func GetBinsFromInputDescr(binStr string) []int {
	listBinsStr := strings.Split(binStr, ",")
	var listBins []int
	for _, s := range listBinsStr {
		n, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("unable to get array of thresholds for bins: %s", err)
		}
		listBins = append(listBins, n)
	}
	return listBins
}
