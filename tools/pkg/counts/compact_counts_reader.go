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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/collective_profiler/tools/internal/pkg/notation"
	"github.com/gvallee/collective_profiler/tools/internal/pkg/progress"
	"github.com/gvallee/collective_profiler/tools/pkg/errors"
)

const (
	CompactFormatSendContext    = iota
	CompactFormatRecvContext    = iota
	CompactFormatUnknownContext = iota
)

// AnalyzeCounts analyses the count from a count file
func AnalyzeCounts(counts []string, msgSizeThreshold int, datatypeSize int) (Stats, map[int][]int, error) {
	var stats Stats

	if datatypeSize == 0 {
		return stats, nil, fmt.Errorf("invalid datatype size (%d)", datatypeSize)
	}

	stats.Min = -1
	stats.Max = -1
	stats.MinWithoutZero = -1
	stats.Patterns = make(map[int]int)
	stats.NoZerosPerRankPatterns = make(map[int]int)
	stats.ZerosPerRankPatterns = make(map[int]int)
	stats.Sum = 0
	stats.MsgSizeThreshold = msgSizeThreshold
	stats.TotalZeroCounts = 0
	stats.TotalNonZeroCounts = 0

	data := make(map[int][]int)

	zeros := 0
	nonZeros := 0
	//smallMsgs := 0
	smallNotZeroMsgs := 0
	//largeMsgs := 0

	for _, line := range counts {
		tokens := strings.Split(line, ": ")
		c := tokens[0]
		c = strings.TrimPrefix(c, RankListPrefix)
		numberOfRanks, err := notation.GetNumberOfRanksFromCompressedNotation(c)
		if err != nil {
			return stats, nil, fmt.Errorf("notation.GetNumberOfRanksFromCompressedNotation() failed: %w", err)
		}
		ranks, err := notation.ConvertCompressedCallListToIntSlice(c)
		if err != nil {
			return stats, nil, fmt.Errorf("notation.ConvertCompressedCallListToIntSlice() failed: %w", err)
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
				return stats, nil, err
			}
			for _, rank := range ranks {
				data[rank] = append(data[rank], count)
			}
			stats.Sum += count

			if count == 0 {
				zeros++
				stats.TotalZeroCounts += numberOfRanks
			} else {
				nonZeros++
				stats.TotalNonZeroCounts += numberOfRanks
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
			stats.Patterns[nonZeros] += numberOfRanks
		}

		if zeros > 0 {
			stats.ZerosPerRankPatterns[zeros] += numberOfRanks
		}

		if stats.SmallNotZeroMsgs > 0 {
			stats.NoZerosPerRankPatterns[smallNotZeroMsgs] += numberOfRanks
		}
	}

	return stats, data, nil
}

// GetCompactHeader reads and parses a specific header from a send or receive count profile in the compact format
func GetCompactHeader(reader *bufio.Reader) (HeaderT, error) {
	var header HeaderT
	var err error

	header.CallIDsStr = ""
	header.TotalNumCalls = 0
	header.NumRanks = 0
	header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize = 0

	alltoallvCallStart := -1
	alltoallvCallEnd := -1
	line := ""

	// Get the first line of the header skipping potential empty lines that
	// can be in front of header
	var readerErr error
	for line == "" || line == "\n" {
		line, readerErr = reader.ReadString('\n')
		if readerErr == io.EOF {
			return header, readerErr
		}
		if readerErr != nil {
			return header, readerErr
		}
	}

	// Are we at the beginning of a metadata block?
	if !strings.HasPrefix(line, CompactCountsFileHeader) {
		return header, fmt.Errorf("%s is not a header (.%s. vs. .%s.)", line, CompactCountsFileHeader, line)
	}

	for {
		line, readerErr = reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return header, readerErr
		}

		if strings.HasPrefix(line, NumberOfRanksMarker) {
			line = strings.ReplaceAll(line, NumberOfRanksMarker, "")
			line = strings.ReplaceAll(line, "\n", "")
			header.NumRanks, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse number of ranks")
				return header, readerErr
			}
		}

		if strings.HasPrefix(line, DatatypeSizeMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			line = strings.ReplaceAll(line, DatatypeSizeMarker, "")
			header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse the datatype size")
				return header, readerErr
			}
		}

		if strings.HasPrefix(line, AlltoallvCallNumbersMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			callRange := strings.ReplaceAll(line, AlltoallvCallNumbersMarker, "")
			tokens := strings.Split(callRange, "-")
			if len(tokens) == 2 {
				alltoallvCallStart, err = strconv.Atoi(strings.TrimLeft(tokens[0], " "))
				if err != nil {
					log.Printf("[ERROR] unable to parse line to get first alltoallv call number: %s", line)
					return header, err
				}
				alltoallvCallEnd, err = strconv.Atoi(tokens[1])
				if err != nil {
					log.Printf("[ERROR] unable to convert %s to interger: %s", tokens[1], err)
					return header, err
				}
				header.TotalNumCalls = alltoallvCallEnd - alltoallvCallStart + 1 // Add 1 because we are 0-indexed
			}
		}

		if strings.HasPrefix(line, CompactCountMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			//strParsing := line
			tokens := strings.Split(line, " - ")
			if len(tokens) > 1 {
				//strParsing = tokens[0]
				header.CallIDsStr = tokens[1]
				tokens2 := strings.Split(header.CallIDsStr, " (")
				if len(tokens2) > 1 {
					header.CallIDsStr = tokens2[0]
				}
			}

			//strParsing = strings.ReplaceAll(strParsing, marker, "")
			//strParsing = strings.ReplaceAll(strParsing, " calls", "")

			if header.CallIDsStr != "" {
				header.CallIDs, err = notation.ConvertCompressedCallListToIntSlice(header.CallIDsStr)
				if err != nil {
					log.Printf("[ERROR] unable to parse calls IDs: %s", err)
					return header, err
				}
			}
		}

		// We check for the beginning of the actual data
		if strings.HasPrefix(line, BeginningCompactDataMarker) {
			break
		}

		if readerErr == io.EOF {
			return header, readerErr
		}
	}

	return header, nil
}

// GetCompactCounters returns the counts using the provided reader
func GetCompactCounters(reader *bufio.Reader) ([]string, error) {
	var callCounters []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return callCounters, err
		}

		if strings.Contains(line, "END DATA") {
			break
		}

		callCounters = append(callCounters, strings.TrimRight(strings.TrimRight(line, "\n"), " "))
	}

	return callCounters, nil
}

// LookupCallFromFile extract counts of a specific call from a count file.
func LookupCallFromFile(reader *bufio.Reader, numCall int) (HeaderT, []string, *errors.ProfilerError) {
	var counts []string
	var err error
	var callIDs []int
	var header HeaderT

	for {
		header, err = GetCompactHeader(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return header, nil, errors.New(errors.ErrFatal, fmt.Errorf("unable to read header: %s", err))
		}
		for _, i := range callIDs {
			if i == numCall {
				counts, err = GetCompactCounters(reader)
				if err == nil {
					// We found the call's data
					return header, counts, errors.New(errors.ErrNone, nil)
				}
				return header, nil, errors.New(errors.ErrFatal, err)
			}
		}

		// We do not need these counts but we still read them to find the next header
		_, err = GetCompactCounters(reader)
		if err != nil {
			return header, nil, errors.New(errors.ErrFatal, fmt.Errorf("unable to parse file: %s", err))
		}
	}

	// We did not find the callID and it might be expected: the call ID is absolute,
	// i.e., reflect all the Alltoallv calls the rank encounters as a lead (rank 0
	// on the communicator) or participants.
	return header, nil, errors.New(errors.ErrNotFound, nil)
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
		ranksListStr := strings.Split(strings.TrimPrefix(ranks, RankListPrefix), " ")
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

func ParseRawCompactFormatFile(f string) ([]string, []int, int, error) {
	var counters []string
	datatypeSize := 0
	file, err := os.Open(f)
	if err != nil {
		return nil, nil, datatypeSize, fmt.Errorf("unable to open %s: %w", f, err)
	}
	defer file.Close()

	var callIDs []int

	reader := bufio.NewReader(file)
	for {
		header, readerErr1 := GetCompactHeader(reader)
		callIDs = append(callIDs, header.CallIDs...)
		datatypeSize = header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize

		if readerErr1 != nil && readerErr1 != io.EOF {
			fmt.Printf("ERROR: %s", readerErr1)
			return nil, nil, 0, fmt.Errorf("unable to read header from %s: %w", f, readerErr1)
		}

		callCounters, readerErr2 := GetCompactCounters(reader)
		if readerErr2 != nil && readerErr2 != io.EOF {
			return nil, nil, 0, readerErr2
		}
		counters = append(counters, strings.Join(callCounters, "\n"))

		if readerErr1 == io.EOF || readerErr2 == io.EOF {
			break
		}
	}
	return counters, callIDs, datatypeSize, nil
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
			header, readerErr1 := GetCompactHeader(reader)
			datatypeSize = header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize

			if readerErr1 != nil && readerErr1 != io.EOF {
				fmt.Printf("ERROR: %s", readerErr1)
				return counters, datatypeSize, found, fmt.Errorf("unable to read header from %s: %w", f, readerErr1)
			}

			targetCall := false
			for i := 0; i < len(header.CallIDs); i++ {
				if header.CallIDs[i] == callNum {
					targetCall = true
					break
				}
			}

			var readerErr2 error
			var callCounters []string
			if targetCall {
				callCounters, readerErr2 = GetCompactCounters(reader)
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
				_, err := GetCompactCounters(reader)
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

// LoadCallsData parses the count files and load all the data about all the calls.
// The returned data is map where the key is the call number and the value the data about the call.
func LoadCallsData(sendCountsFile, recvCountsFile string, rank int, msgSizeThreshold int) (map[int]*CallData, error) {
	var readerErr error

	callData := make(map[int]*CallData) // The key is the call number and the value a pointer to the call's data (several calls can share the same data)

	bar := progress.NewBar(2, "Reading count files")
	defer progress.EndBar(bar)

	bar.Increment(1)
	sendFile, err := os.Open(sendCountsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", sendCountsFile, err)
	}
	defer sendFile.Close()
	reader := bufio.NewReader(sendFile)
	for {
		cd := new(CallData)
		cd.SendData.CountsMetadata, readerErr = GetCompactHeader(reader)
		if readerErr == io.EOF || len(cd.SendData.CountsMetadata.CallIDs) == 0 {
			break
		}
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", sendCountsFile, readerErr)
		}
		cd.CommSize = cd.SendData.CountsMetadata.NumRanks
		cd.MsgSizeThreshold = msgSizeThreshold
		cd.SendData.RawCounts, readerErr = GetCompactCounters(reader)
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", sendCountsFile, readerErr)
		}
		cd.SendData.File = sendCountsFile

		var sendCounts map[int][]int
		cd.SendData.Statistics, sendCounts, err = AnalyzeCounts(cd.SendData.RawCounts, msgSizeThreshold, cd.SendData.CountsMetadata.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize)
		if err != nil {
			return nil, err
		}
		cd.SendData.Statistics.DatatypeSize = cd.SendData.CountsMetadata.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize

		for _, callID := range cd.SendData.CountsMetadata.CallIDs {
			callData[callID] = cd
			if cd.SendData.Counts == nil {
				cd.SendData.Counts = make(map[int]map[int][]int)
			}
			cd.SendData.Counts[callID] = sendCounts
		}

		if readerErr == io.EOF {
			break
		}
	}

	bar.Increment(1)
	recvFile, err := os.Open(recvCountsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", recvCountsFile, err)
	}
	defer recvFile.Close()
	reader = bufio.NewReader(recvFile)
	for {
		header, readerErr := GetCompactHeader(reader)
		if readerErr == io.EOF {
			break
		}
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", recvCountsFile, readerErr)
		}

		counts, readerErr := GetCompactCounters(reader)
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to reader counts from %s: %w", recvCountsFile, readerErr)
		}

		stats, data, err := AnalyzeCounts(counts, msgSizeThreshold, header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize)
		if err != nil {
			return nil, err
		}

		for _, callID := range header.CallIDs {
			if header.NumRanks != callData[callID].CommSize {
				return nil, fmt.Errorf("inconsistent comm size for call %d: %d vs. %d", callID, header.NumRanks, callData[callID].CommSize)
			}
			cb := callData[callID]
			cb.RecvData.CountsMetadata = header
			cb.RecvData.Statistics = stats
			cb.RecvData.RawCounts = counts
			cb.RecvData.File = recvCountsFile
			cb.RecvData.Statistics.DatatypeSize = header.DatatypeInfo.CompactFormatDatatypeInfo.DatatypeSize
			callData[callID] = cb
			if cb.RecvData.Counts == nil {
				cb.RecvData.Counts = make(map[int]map[int][]int)
			}
			cb.RecvData.Counts[callID] = data
		}

		if readerErr == io.EOF {
			break
		}
	}

	return callData, nil
}

func findSendCountersFiles(basedir string, jobid int, id int) ([]string, error) {
	idStr := strconv.Itoa(id)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, idStr, SendCountersFilePrefix)
}

func findRecvCountersFiles(basedir string, jobid int, id int) ([]string, error) {
	idStr := strconv.Itoa(id)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, idStr, RecvCountersFilePrefix)
}

// GetFiles returns the full path to the count files for a given rank of a given job
func GetFiles(jobid int, rank int) (string, string) {
	suffix := "job" + strconv.Itoa(jobid) + ".rank" + strconv.Itoa(rank) + ".txt"
	return SendCountersFilePrefix + suffix, RecvCountersFilePrefix + suffix
}

func findCallRankSendCounters(basedir string, jobid int, rank int, callNum int) (string, error) {
	files, err := findSendCountersFiles(basedir, jobid, rank)
	if err != nil {
		return "", err
	}
	counters, _, _, err := ReadCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("* unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func findCallRankRecvCounters(basedir string, jobid int, rank int, callNum int) (string, error) {
	files, err := findRecvCountersFiles(basedir, jobid, rank)
	if err != nil {
		return "", err
	}
	counters, _, _, err := ReadCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func FindCallRankCounters(basedir string, jobid int, rank int, callNum int) (string, string, error) {
	sendCounters, err := findCallRankSendCounters(basedir, jobid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	recvCounters, err := findCallRankRecvCounters(basedir, jobid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")
	sendCounters = strings.TrimRight(sendCounters, " ")
	recvCounters = strings.TrimRight(recvCounters, " ")

	return sendCounters, recvCounters, nil
}

// GetNumCalls returns the total number of calls associated to a specific send/receive count profile file
func GetNumCalls(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	header, err := GetCompactHeader(reader)
	if err != nil {
		return 0, err
	}
	return header.TotalNumCalls, nil
}

func sameRawCounts(counts1 []string, counts2 []string) bool {
	if len(counts1) != len(counts2) {
		return false
	}

	for i := 0; i < len(counts1); i++ {
		if counts1[i] != counts2[i] {
			return false
		}
	}

	return true
}

func rawSendCountsAlreadyExists(rc *RawCountsCallsT, list []*RawCountsCallsT) int {
	idx := 0
	for _, d := range list {
		if rc.Counts.SendDatatypeSize == d.Counts.SendDatatypeSize && rc.Counts.CommSize == d.Counts.CommSize && sameRawCounts(rc.Counts.SendCounts, d.Counts.SendCounts) {
			return idx
		}
		idx++
	}

	return -1
}

/*
func rawRecvCountsAlreadyExists(rc rawCountsT, list []RawCountsCallsT) int {
	idx := 0
	for _, d := range list {
		if rc.recvDatatypeSize == d.counts.recvDatatypeSize && rc.commSize == d.counts.commSize && sameRawCounts(rc.recvCounts, d.counts.recvCounts) {
			return idx
		}
		idx++
	}

	return -1
}
*/

func compactCountFormatToList(rawCounters []string) ([]string, error) {
	mapCounts := make(map[int]string)
	for _, recvCounts := range rawCounters {
		tokens := strings.Split(recvCounts, ": ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("%s is not a valid format for compact counts", recvCounts)
		}
		callIDsStr := strings.TrimPrefix(tokens[0], RankListPrefix)
		callIDs, err := notation.ConvertCompressedCallListToIntSlice(callIDsStr)
		if err != nil {
			return nil, err
		}
		for _, callID := range callIDs {
			mapCounts[callID] = tokens[1]
		}
	}

	var listCounts []string
	for rank := 0; rank < len(mapCounts); rank++ {
		listCounts = append(listCounts, mapCounts[rank])
	}
	return listCounts, nil
}

func LoadCountsFromCompactFormatFile(file string, ctxt int) (*RawCountsCallsT, error) {
	var recvRawCounters []string
	var recvDatatypeSize int
	var sendRawCounters []string
	var sendDatatypeSize int
	var callIDs []int
	var err error

	switch ctxt {
	case CompactFormatRecvContext:
		recvRawCounters, callIDs, recvDatatypeSize, err = ParseRawCompactFormatFile(file)
		if err != nil {
			return nil, fmt.Errorf("parseRawFile() failed (%s): %w", file, err)
		}
	case CompactFormatSendContext:
		sendRawCounters, callIDs, sendDatatypeSize, err = ParseRawCompactFormatFile(file)
		if err != nil {
			return nil, fmt.Errorf("parseRawFile() failed (%s): %w", file, err)
		}
	default:
		return nil, fmt.Errorf("unsupported mode: %d (should be %d or %d)", ctxt, CompactFormatSendContext, CompactFormatRecvContext)
	}

	// From here, we know we have the content of the file but still in the compact format.

	data := new(RawCountsCallsT)
	data.Calls = callIDs
	data.Counts = new(rawCountsT)
	// Convert compact counts so we are independent from the compact format.
	data.Counts.SendCounts, err = compactCountFormatToList(sendRawCounters)
	if err != nil {
		return nil, err
	}
	data.Counts.SendDatatypeSize = sendDatatypeSize
	// Convert compact counts so we are independent from the compact format.
	data.Counts.RecvCounts, err = compactCountFormatToList(recvRawCounters)
	data.Counts.RecvDatatypeSize = recvDatatypeSize
	return data, err
}

func LoadCommunicatorRawCompactFormatCounts(outputDir string, jobId int, commLeadRank int) ([]*RawCountsCallsT, error) {
	var rawCounts []*RawCountsCallsT
	recvCountFile := fmt.Sprintf("recv-counters.job%d.rank%d.txt", jobId, commLeadRank)
	sendCountFile := fmt.Sprintf("recv-counters.job%d.rank%d.txt", jobId, commLeadRank)

	b := progress.NewBar(2, fmt.Sprintf("Load count files for communicator %d", commLeadRank))
	defer progress.EndBar(b)

	b.Increment(1)
	file := filepath.Join(outputDir, sendCountFile)
	rc, err := LoadCountsFromCompactFormatFile(file, CompactFormatSendContext)
	if err != nil {
		return nil, err
	}

	b.Increment(1)
	file = filepath.Join(outputDir, recvCountFile)
	rc2, err := LoadCountsFromCompactFormatFile(file, CompactFormatRecvContext)
	if err != nil {
		return nil, err
	}

	rc.Counts.RecvCounts = rc2.Counts.RecvCounts
	rc.Counts.RecvDatatypeSize = rc2.Counts.RecvDatatypeSize
	idx := rawSendCountsAlreadyExists(rc, rawCounts)
	if idx == -1 {
		rawCounts = append(rawCounts, rc)
	} else {
		rawCounts[idx].Calls = append(rawCounts[idx].Calls, rc.Calls...)
	}

	return rawCounts, nil
}

func GetContextFromFileName(filename string) int {
	if strings.HasPrefix(filename, SendCountersFilePrefix) {
		return CompactFormatSendContext
	}

	if strings.HasPrefix(filename, RecvCountersFilePrefix) {
		return CompactFormatRecvContext
	}

	return CompactFormatUnknownContext
}

func GetMetadataFromCompactFormatCountFileName(filename string) (int, int, int, error) {
	ctxt := CompactFormatUnknownContext
	if strings.HasPrefix(filename, SendCountersFilePrefix) {
		ctxt = CompactFormatSendContext
	}
	if strings.HasPrefix(filename, RecvCountersFilePrefix) {
		ctxt = CompactFormatRecvContext
	}

	str := strings.TrimPrefix(filename, SendCountersFilePrefix)
	str = strings.TrimPrefix(str, RecvCountersFilePrefix)
	str = strings.TrimSuffix(str, ".txt")
	tokens := strings.Split(str, CompactFormatLeadRankMarker)
	if len(tokens) != 2 {
		return -1, -1, -1, fmt.Errorf("unable to parse file name (%s)", filename)
	}
	leadRank, err := strconv.Atoi(tokens[1])
	if err != nil {
		return -1, -1, -1, err
	}
	str = strings.TrimPrefix(tokens[0], CompactFormatJobIDMarker)
	jobID, err := strconv.Atoi(str)
	if err != nil {
		return -1, -1, -1, err
	}
	return ctxt, jobID, leadRank, nil
}
