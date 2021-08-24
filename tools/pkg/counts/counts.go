//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
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
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/collective_profiler/tools/pkg/errors"
)

const (
	/* All the constants related to the compact format */

	// CompactHeader is the string used as a prefix to indicate raw counters in the count files
	CompactHeader               = "# Raw counters"
	CompactCountsFileHeader     = "# Raw counters\n"
	CompactCountMarker          = "Count: "
	NumberOfRanksMarker         = "Number of ranks: "
	DatatypeSizeMarker          = "Datatype size: "
	AlltoallvCallNumbersMarker  = "Alltoallv calls "
	BeginningCompactDataMarker  = "BEGINNING DATA"
	EndCompactDataMarker        = "END DATA"
	RankListPrefix              = "Rank(s) "
	CompactFormatLeadRankMarker = ".rank"
	CompactFormatJobIDMarker    = "job"

	/* All the constants related to the standard format */
	StandardFormatSendDatatypeMarker = "Send datatype size: "
	StandardFormatRecvDatatypeMarker = "Recv datatype size: "
	StandardFormatCommSizeMarker     = "Comm size: "
	StandardFormatSendCountsMarker   = "Send counts"
	StandardFormatRecvCountsMarker   = "Recv counts"

	// SendCountersFilePrefix is the prefix used for all send counts files
	SendCountersFilePrefix = "send-counters."
	// RecvCountersFilePrefix is the prefix used for all receive counts files
	RecvCountersFilePrefix = "recv-counters."
	// RawCountersFilePrefix is the prefix used for all raw counts files (one file per call; no compact format)
	RawCountersFilePrefix = "counts.rank"
)

const (
	FormatUnknown = iota
	FormatCompact = iota
	FormatPerCall = iota
)

type CompactFormatHeader struct {
	// DatatypeSize is the size of the datatype associated to the counts
	DatatypeSize int
}

type StandardFormatHeader struct {
	SendDatatypeSize int
	RecvDatatypeSize int
}

// HeaderT is the data extracted from the counts headr from a count profile file in the compact format
type HeaderT struct {
	// TotalNumCalls is the overall total number of alltoallv calls
	TotalNumCalls int

	// CallIDs is the list of alltoallv calls associated to the counts
	CallIDs []int

	// CallIDsStr is the list in string and compact format of all alltoallv calls associated to the counts
	CallIDsStr string

	// NumRanks is the number of ranks involved in the alltoallv call (i.e., the communicator size)
	NumRanks int

	DatatypeInfo struct {
		CompactFormatDatatypeInfo  CompactFormatHeader
		StandardFormatDatatypeInfo StandardFormatHeader
	}
}

type rawCountsT struct {
	SendDatatypeSize int
	RecvDatatypeSize int
	CommSize         int
	SendCounts       []string // One line per rank in order, based on the rank number of the communicator used
	RecvCounts       []string // One line per rank in order, based on the rank number of the communicator used
}

type RawCountsCallsT struct {
	LeadRank int
	Calls    []int
	Counts   *rawCountsT
}

// CallData gathers all the data related to one and only one alltoallv call
type CallData struct {
	// CommSize is the communicator size used for the call
	CommSize int

	// MsgSizeThreshold is the size value that differentiate small and large messages.
	MsgSizeThreshold int

	// SendData is all the data from the send counts
	SendData Data

	// RecvData is all the data from the receive counts
	RecvData Data
}

// Stats represent the stats related to counts of a specific collective operation
type Stats struct {
	// DatatypeSize is the size of the datatype
	DatatypeSize int

	// MsgSizeThreshold is the message size used to differentiate small messages from larrge messages while parsing the counts
	MsgSizeThreshold int

	// TotalNumCalls is the number of alltoallv calls covered by the statistics
	TotalNumCalls int

	// Sum is the total count for all ranks data is sent to or received from
	Sum int

	// MinWithoutZero from the entire counts not including zero
	MinWithoutZero int

	// Min from the entire counts, including zero
	Min int

	// NotZeroMin is the minimum but not equal to zero count
	NotZeroMin int

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

	// TotalNonZeroCounts is the total number of non-zero counts from counters
	TotalNonZeroCounts int

	// ZerosPerRankPatterns gathers the number of 0-counts on a per-rank basis ('X ranks have Y 0-counts')
	ZerosPerRankPatterns map[int]int

	// NoZerosPerRankPatterns gathers the number of non-0-counts on a per-rank bases ('X ranks have Y non-0-counts)
	NoZerosPerRankPatterns map[int]int

	// Patterns gathers the number of peers involved in actual communication, i.e., non-zero ('<value> ranks are sending to <key> ranks')
	Patterns map[int]int
}

// Data gathers all the data from a count file (send or receive) for a given alltoallv call.
// This is used to store data when parsing a given count file.
type Data struct {
	// File is the path to the associated counts files
	File string

	// CountsMetadata is the metadata associated to the counts
	CountsMetadata HeaderT

	// RawCounts is the string representing all the send counts. Used for instance by the webui
	RawCounts []string

	// Counts are the counts for all ranks involved in the operation
	// For the outer map: The key is the call ID
	// For the inner map: The key is the rank sending/receiving the data and the value an array of integers representing counts for each destination/source
	Counts map[int]map[int][]int

	// Statistics is all the statistics we could gather while parsing the count file
	Statistics Stats

	// BinThresholds is the list of thresholds used to create bins
	BinThresholds []int

	/* Cannot be here, bins are above counts package
	// Bins is the list of bins of counts
	Bins []bins.Data
	*/

	// MsgSizeThreshold is the message size used to differentiate small messages from larrge messages while parsing the counts
	MsgSizeThreshold int
}

// Stats gathers all the data related to send and receive counts for one or more alltoallv call(s)
// This is used when combining data from send and receive counts for specific alltoallv calls.
type SendRecvStats struct {
	NumSendSmallMsgs        int
	NumSendLargeMsgs        int
	SizeThreshold           int
	NumSendSmallNotZeroMsgs int

	/*
		// SendSums is the sum of all the send counts. It can be used to calculate how much data is sent during the alltoallv call.
		SendSums map[int]int

		// RecvSums is the sum of all the receive counts. It can be used to calculate how much data is received during the alltoallv call.
		RecvSums map[int]int
	*/

	// TotalNumCalls is the number of alltoallv calls covered by the statistics
	TotalNumCalls int

	// TotalSendZeroCounts is the total number of send count equal to zero
	TotalSendZeroCounts int

	// TotalSendNonZeroCounts is the total number of send count not equal to zero
	TotalSendNonZeroCounts int

	// TotalRecvZeroCounts is the total number of receive count equal to zero
	TotalRecvZeroCounts int

	// TotalRecvNonZeroCounts is the total number of receive count not equal to zero
	TotalRecvNonZeroCounts int

	// CommSizes is the distribution of communicator size across all alltoallv calls. The key is the size of the communicator; the value is the number of alltoallv calls having that size
	CommSizes map[int]int

	// DatatypesSend stores statistics related to MPI datatypes that are used to send data. The key is the size of the datatype, the value hte number of time the datatype is used
	DatatypesSend map[int]int

	// DatatypesRecv stores statistics related to MPI datatypes that are used to receive data. The key is the size of the datatype, the value hte number of time the datatype is used
	DatatypesRecv map[int]int

	// CallSendSparisty is the distribution of zero send counts across alltoallv calls. The key is the number of zero counts and the value is the number of calls that has so many zero send counts
	CallSendSparsity map[int]int

	// CallRecvSparisty is the distribution of zero receive counts across alltoallv calls. The key is the number of zero counts and the value is the number of calls that has so many zero receive counts
	CallRecvSparsity map[int]int

	// SendMins is the send min distribution across alltoallv calls. The key is the value of the min for a given alltoallv call, the value the number of calls having that min
	SendMins map[int]int

	// RecvMins is the receive min distribution across alltoallv calls. The key is the value of the min for a given alltoallv call, the value the number of calls having that min
	RecvMins map[int]int

	// SendMaxs is the send max distribution across alltoallv calls. The key is the value of the max for a given alltoallv call, the value the number of calls having that max
	SendMaxs map[int]int

	// RecvMaxs is the recv max distribution across alltoallv calls. The key is the value of the max for a given alltoallv call, the value the number of calls having that max
	RecvMaxs map[int]int

	SendNotZeroMins map[int]int
	RecvNotZeroMins map[int]int

	// SendNotZeroCounts is the distribution of non-zero counts across alltoallv calls. Counter-part of CallSendSparsity
	SendNotZeroCounts map[int]int

	// RecvNotZeroCounts is the distribution of non-zero counts across alltoallv calls. Counter-part of CallRecvSparsity
	RecvNotZeroCounts map[int]int

	// SendSums is the distribution of the amount of data sent during alltoallv calls. The key is the total amount of data sent during a call; the value is the number of calls sending that amount of data
	SendSums map[int]int

	// RecvSums is the distribution of the amount of data received during alltoallv calls. The key is the total amount of data received during a call; the value is the number of calls receiving that amount of data
	RecvSums map[int]int

	/*
		SendPatterns      map[int]int
		RecvPatterns      map[int]int
	*/
}

// CommDataT is a structure that allows us to store alltoallv calls' data on a per-communicator basis.
// The communicator is identified by the leadRank, i.e., the comm world rank of the communicator's rank 0.
// Note that for any leadRank, it is possible to have multiple commDataT since alltoallv calls can be
// invoked on different communicators with the same leadRank and we are currently limited to compare
// communicators to identify these that are identical.
type CommDataT struct {
	// LeadRank is the rank on COMMWORLD that is rank 0 on the communicator used for the alltoallv operation
	LeadRank int

	// CallData is the data for all the alltoallv calls performed on the communicator(s) led by leadRank;
	// rhw key is the call number and the value a pointer to the call's data (several calls can share the same data)
	CallData map[int]*CallData

	RawCounts []*RawCountsCallsT
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

		sendCounters2, recvCounters2, err := FindCallRankCounters(dir, jobid, rank, call)
		if err != nil {
			fmt.Printf("unable to get counters: %s", err)
			return err
		}

		if sendCounters1 != sendCounters2 {
			return fmt.Errorf("send counters do not match with %s: expected '%s' but got '%s'\nReceive counts are: %s vs. %s", filepath.Base(f), sendCounters1, sendCounters2, recvCounters1, recvCounters2)
		}

		if recvCounters1 != recvCounters2 {
			return fmt.Errorf("receive counters do not match %s: expected '%s' but got '%s'\nSend counts are: %s vs. %s", filepath.Base(f), recvCounters1, recvCounters2, sendCounters1, sendCounters2)
		}

		fmt.Printf("File %s validated\n", filepath.Base(f))
	}

	return nil
}

// NewStats returns a fully initialized Stats structure
func NewSendRecvStats(sizeThreshold int) SendRecvStats {
	cs := SendRecvStats{
		NumSendSmallMsgs:        0,
		NumSendLargeMsgs:        0,
		SizeThreshold:           sizeThreshold,
		NumSendSmallNotZeroMsgs: 0,
		TotalNumCalls:           0,
		TotalSendZeroCounts:     0,
		TotalSendNonZeroCounts:  0,
		TotalRecvZeroCounts:     0,
		TotalRecvNonZeroCounts:  0,
		SendSums:                make(map[int]int),
		RecvSums:                make(map[int]int),
		CommSizes:               make(map[int]int),
		DatatypesSend:           make(map[int]int),
		DatatypesRecv:           make(map[int]int),
		CallSendSparsity:        make(map[int]int),
		CallRecvSparsity:        make(map[int]int),
		SendMins:                make(map[int]int),
		RecvMins:                make(map[int]int),
		SendMaxs:                make(map[int]int),
		RecvMaxs:                make(map[int]int),
		SendNotZeroMins:         make(map[int]int),
		RecvNotZeroMins:         make(map[int]int),
		SendNotZeroCounts:       make(map[int]int),
		RecvNotZeroCounts:       make(map[int]int),
		/*
			todo: rethink send and receive patterns
				SendPatterns:            make(map[int]int),
				RecvPatterns:            make(map[int]int),
		*/
	}
	return cs
}

func LookupCall(sendCountsFile, recvCountsFile string, numCall int) (CallData, *errors.ProfilerError) {
	var data CallData
	var profilerErr *errors.ProfilerError

	fSendCounts, err := os.Open(sendCountsFile)
	if err != nil {
		return data, errors.New(errors.ErrFatal, fmt.Errorf("unable to open %s: %s", sendCountsFile, err))
	}
	defer fSendCounts.Close()
	fRecvCounts, err := os.Open(recvCountsFile)
	if err != nil {
		return data, errors.New(errors.ErrFatal, fmt.Errorf("unable to open %s: %s", recvCountsFile, err))
	}
	defer fRecvCounts.Close()

	sendCountsReader := bufio.NewReader(fSendCounts)
	recvCountsReader := bufio.NewReader(fRecvCounts)

	// find the call's data from the send counts file first
	var sendCountsHeader HeaderT
	sendCountsHeader, data.SendData.RawCounts, profilerErr = LookupCallFromFile(sendCountsReader, numCall)
	if !profilerErr.Is(errors.ErrNone) {
		return data, profilerErr
	}
	data.SendData.Statistics.DatatypeSize = sendCountsHeader.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize

	// find the call's data from the recv counts file then
	var recvCountsHeader HeaderT
	recvCountsHeader, data.RecvData.RawCounts, profilerErr = LookupCallFromFile(recvCountsReader, numCall)
	if err != nil {
		return data, profilerErr
	}
	data.RecvData.Statistics.DatatypeSize = recvCountsHeader.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize

	if sendCountsHeader.NumRanks != recvCountsHeader.NumRanks {
		return data, errors.New(errors.ErrFatal, fmt.Errorf("differ number of ranks from send and recv counts files"))
	}

	return data, errors.New(errors.ErrNone, nil)
}

// ParseFiles parses both send and receive counts files
func ParseFiles(sendCountsFile string, recvCountsFile string, numCalls int, sizeThreshold int) (SendRecvStats, error) {
	cs := NewSendRecvStats(sizeThreshold)
	cs.TotalNumCalls = numCalls

	for i := 0; i < numCalls; i++ {
		log.Printf("Analyzing call #%d\n", i)
		callData, profilerErr := LookupCall(sendCountsFile, recvCountsFile, i)
		if profilerErr.Is(errors.ErrNotFound) {
			log.Printf("Call %d could not be find in files, it may have happened on a different communicator", i)
			continue
		}

		if !profilerErr.Is(errors.ErrNone) {
			return cs, profilerErr.GetInternal()
		}

		cs.NumSendSmallMsgs += callData.SendData.Statistics.SmallMsgs
		cs.NumSendSmallNotZeroMsgs += callData.SendData.Statistics.SmallNotZeroMsgs
		cs.NumSendLargeMsgs += callData.SendData.Statistics.LargeMsgs

		cs.DatatypesSend[callData.SendData.Statistics.DatatypeSize]++
		cs.DatatypesRecv[callData.RecvData.Statistics.DatatypeSize]++
		cs.CommSizes[callData.CommSize]++
		cs.SendMins[callData.SendData.Statistics.Min]++
		cs.RecvMins[callData.RecvData.Statistics.Min]++
		cs.SendMaxs[callData.SendData.Statistics.Max]++
		cs.RecvMaxs[callData.RecvData.Statistics.Max]++
		cs.SendMins[callData.SendData.Statistics.NotZeroMin]++
		cs.RecvMins[callData.RecvData.Statistics.NotZeroMin]++
		cs.CallSendSparsity[callData.SendData.Statistics.TotalZeroCounts]++
		cs.CallRecvSparsity[callData.RecvData.Statistics.TotalZeroCounts]++
	}

	return cs, nil
}

func GatherStatsFromCallData(cd map[int]*CallData, sizeThreshold int) (SendRecvStats, error) {
	cs := NewSendRecvStats(sizeThreshold)

	for _, d := range cd {
		if cs.SizeThreshold == 0 {
			cs.SizeThreshold = d.MsgSizeThreshold
		} else {
			if cs.SizeThreshold != d.MsgSizeThreshold {
				return cs, fmt.Errorf("inconsistent data, different message size thresholds: %d vs. %d", cs.SizeThreshold, d.MsgSizeThreshold)
			}
		}

		cs.TotalNumCalls += d.SendData.Statistics.TotalNumCalls

		cs.NumSendSmallMsgs += d.SendData.Statistics.SmallMsgs
		cs.NumSendLargeMsgs += d.SendData.Statistics.LargeMsgs
		cs.NumSendSmallNotZeroMsgs += d.SendData.Statistics.SmallNotZeroMsgs
		cs.TotalSendZeroCounts += d.SendData.Statistics.TotalZeroCounts
		cs.TotalSendNonZeroCounts += d.SendData.Statistics.TotalNonZeroCounts

		cs.TotalRecvNonZeroCounts += d.RecvData.Statistics.TotalNonZeroCounts
		cs.TotalRecvZeroCounts += d.RecvData.Statistics.TotalZeroCounts
		cs.SendMins[d.SendData.Statistics.Min]++
		cs.RecvMins[d.RecvData.Statistics.Min]++
		cs.SendMaxs[d.SendData.Statistics.Max]++
		cs.RecvMaxs[d.RecvData.Statistics.Max]++
		cs.DatatypesSend[d.SendData.Statistics.DatatypeSize]++
		cs.DatatypesRecv[d.RecvData.Statistics.DatatypeSize]++
		cs.SendNotZeroMins[d.SendData.Statistics.NotZeroMin]++
		cs.RecvNotZeroMins[d.RecvData.Statistics.NotZeroMin]++
		cs.CommSizes[d.CommSize]++
		cs.CallSendSparsity[d.SendData.Statistics.TotalZeroCounts]++
		cs.CallRecvSparsity[d.RecvData.Statistics.TotalZeroCounts]++
		cs.SendNotZeroCounts[d.SendData.Statistics.TotalNonZeroCounts]++
		cs.RecvNotZeroCounts[d.RecvData.Statistics.TotalNonZeroCounts]++
		cs.SendSums[d.SendData.Statistics.Sum]++
		cs.RecvSums[d.RecvData.Statistics.Sum]++

		// FIXME: what to do with SendPatterns?
		// FIXME: what to do with RecvPatterns?

	}

	return cs, nil
}

func DetectFileFormat(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err == io.EOF {
		return FormatUnknown, err
	}
	if err != nil {
		return FormatUnknown, err
	}

	fmt.Printf("DBG - %s\n", line)
	if strings.HasPrefix(line, "Send datatype size: ") {
		fmt.Println("DBG - FormatPerCall")
		return FormatPerCall, nil
	}

	if strings.HasPrefix(line, "# Raw counters") {
		return FormatCompact, nil
	}

	return FormatUnknown, nil
}

func ParseCountFile(filePath string) (*RawCountsCallsT, error) {
	format, err := DetectFileFormat(filePath)
	if err != nil {
		return nil, err
	}
	if format == FormatUnknown {
		return nil, fmt.Errorf("unknown count file format (%s)", filePath)
	}

	var countData *RawCountsCallsT
	if format == FormatCompact {
		filename := path.Base(filePath)
		ctxt, _, leadRank, err := GetMetadataFromCompactFormatCountFileName(filename)
		if err != nil {
			return nil, err
		}
		countData, err = LoadCountsFromCompactFormatFile(filePath, ctxt)
		if err != nil {
			return nil, err
		}
		countData.LeadRank = leadRank
	} else {
		fmt.Printf("DBG - calling ParsePerCallFileCount()\n")
		countData, err = ParsePerCallFileCount(filePath)
		if err != nil {
			return nil, err
		}
		countData.LeadRank = -1
	}

	return countData, nil
}
