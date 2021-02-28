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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/errors"
)

const (
	rawCountsSendDatatypePrefix = "Send datatype size: "
	rawCountsRecvDatatypePrefix = "Recv datatype size: "
	rawCountsCommSizePrefix     = "Comm size: "
	rawCountsSendCountsPrefix   = "Send counts"
	rawCountsRecvCountsPrefix   = "Recv counts"
)

// AnalyzeCounts analyses the count from a count file
func AnalyzeCounts(counts []string, msgSizeThreshold int, datatypeSize int) (Stats, map[int][]int, error) {
	var stats Stats
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
		c = strings.ReplaceAll(c, "Rank(s) ", "")
		numberOfRanks, err := notation.GetNumberOfRanksFromCompressedNotation(c)
		if err != nil {
			return stats, nil, err
		}
		ranks, err := notation.ConvertCompressedCallListToIntSlice(c)

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

	return stats, data, nil
}

// GetHeader reads and parses a specific header from a send or receive count profile in the compact format
func GetHeader(reader *bufio.Reader) (HeaderT, error) {
	var header HeaderT
	var err error

	header.CallIDsStr = ""
	header.TotalNumCalls = 0
	header.NumRanks = 0
	header.DatatypeSize = 0

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
	if !strings.HasPrefix(line, "# Raw") {
		return header, fmt.Errorf("[ERROR] not a header")
	}

	for {
		line, readerErr = reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return header, readerErr
		}

		if strings.HasPrefix(line, numberOfRanksMarker) {
			line = strings.ReplaceAll(line, numberOfRanksMarker, "")
			line = strings.ReplaceAll(line, "\n", "")
			header.NumRanks, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse number of ranks")
				return header, readerErr
			}
		}

		if strings.HasPrefix(line, datatypeSizeMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			line = strings.ReplaceAll(line, datatypeSizeMarker, "")
			header.DatatypeSize, err = strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse the datatype size")
				return header, readerErr
			}
		}

		if strings.HasPrefix(line, alltoallvCallNumbersMarker) {
			line = strings.ReplaceAll(line, "\n", "")
			callRange := strings.ReplaceAll(line, alltoallvCallNumbersMarker, "")
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

		if strings.HasPrefix(line, marker) {
			line = strings.ReplaceAll(line, "\n", "")
			strParsing := line
			tokens := strings.Split(line, " - ")
			if len(tokens) > 1 {
				strParsing = tokens[0]
				header.CallIDsStr = tokens[1]
				tokens2 := strings.Split(header.CallIDsStr, " (")
				if len(tokens2) > 1 {
					header.CallIDsStr = tokens2[0]
				}
			}

			strParsing = strings.ReplaceAll(strParsing, marker, "")
			strParsing = strings.ReplaceAll(strParsing, " calls", "")

			if header.CallIDsStr != "" {
				header.CallIDs, err = notation.ConvertCompressedCallListToIntSlice(header.CallIDsStr)
				if err != nil {
					log.Printf("[ERROR] unable to parse calls IDs: %s", err)
					return header, err
				}
			}
		}

		// We check for the beginning of the actual data
		if strings.HasPrefix(line, beginningDataMarker) {
			break
		}

		if readerErr == io.EOF {
			return header, readerErr
		}
	}

	return header, nil
}

// GetCounters returns the counts using the provided reader
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
		header, err = GetHeader(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return header, nil, errors.New(errors.ErrFatal, fmt.Errorf("unable to read header: %s", err))
		}
		for _, i := range callIDs {
			if i == numCall {
				counts, err = GetCounters(reader)
				if err == nil {
					// We found the call's data
					return header, counts, errors.New(errors.ErrNone, nil)
				}
				return header, nil, errors.New(errors.ErrFatal, err)
			}
		}

		// We do not need these counts but we still read them to find the next header
		_, err = GetCounters(reader)
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
			header, readerErr1 := GetHeader(reader)
			datatypeSize = header.DatatypeSize

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

// LoadCallsData parses the count files and load all the data about all the calls.
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
		cd.SendData.CountsMetadata, readerErr = GetHeader(reader)
		if readerErr == io.EOF || len(cd.SendData.CountsMetadata.CallIDs) == 0 {
			break
		}
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", sendCountsFile, readerErr)
		}
		cd.CommSize = cd.SendData.CountsMetadata.NumRanks
		cd.MsgSizeThreshold = msgSizeThreshold
		cd.SendData.RawCounts, readerErr = GetCounters(reader)
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", sendCountsFile, readerErr)
		}
		cd.SendData.File = sendCountsFile

		var sendCounts map[int][]int
		cd.SendData.Statistics, sendCounts, err = AnalyzeCounts(cd.SendData.RawCounts, msgSizeThreshold, cd.SendData.Statistics.DatatypeSize)
		if err != nil {
			return nil, err
		}
		cd.SendData.Statistics.DatatypeSize = cd.SendData.CountsMetadata.DatatypeSize

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
		header, readerErr := GetHeader(reader)
		if readerErr == io.EOF {
			break
		}
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unable to read header from %s: %w", recvCountsFile, readerErr)
		}

		counts, readerErr := GetCounters(reader)
		if readerErr != nil && readerErr != io.EOF {
			return nil, fmt.Errorf("unavle to reader counts from %s: %w", recvCountsFile, readerErr)
		}

		stats, data, err := AnalyzeCounts(counts, msgSizeThreshold, header.DatatypeSize)
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
			cb.RecvData.Statistics.DatatypeSize = header.DatatypeSize
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
	header, err := GetHeader(reader)
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

func rawSendCountsAlreadyExists(rc rawCountsT, list []rawCountsCallsT) int {
	idx := 0
	for _, d := range list {
		if rc.sendDatatypeSize == d.counts.sendDatatypeSize && rc.commSize == d.counts.commSize && sameRawCounts(rc.sendCounts, d.counts.sendCounts) {
			return idx
		}
		idx++
	}

	return -1
}

func rawRecvCountsAlreadyExists(rc rawCountsT, list []rawCountsCallsT) int {
	idx := 0
	for _, d := range list {
		if rc.recvDatatypeSize == d.counts.recvDatatypeSize && rc.commSize == d.counts.commSize && sameRawCounts(rc.recvCounts, d.counts.recvCounts) {
			return idx
		}
		idx++
	}

	return -1
}

func getInfoFromRawCountsFileName(filename string) (int, int, error) {
	tokens := strings.Split(filepath.Base(filename), "_")
	if len(tokens) != 2 {
		return -1, -1, fmt.Errorf("%s is of wrong format", filename)
	}

	callStr := strings.TrimLeft(tokens[1], "call")
	callStr = strings.TrimRight(callStr, ".md")

	rankStr := strings.TrimLeft(tokens[0], "counts.rank")

	leadRank, err := strconv.Atoi(rankStr)
	if err != nil {
		return -1, -1, err
	}

	callID, err := strconv.Atoi(callStr)
	if err != nil {
		return -1, -1, err
	}

	return leadRank, callID, nil
}

func countsSeriesExists(c string, list []compressedRanksCountsT) int {
	idx := 0
	for _, i := range list {
		if c == i.counts {
			return idx
		}
		idx++
	}
	return -1
}

func compressCounts(counts []string) []string {
	var uniqueCountsSeries []compressedRanksCountsT
	var compressedCounts []string
	curRank := 0
	for _, s := range counts {
		idx := countsSeriesExists(s, uniqueCountsSeries)
		if idx == -1 {
			newCountsSeries := compressedRanksCountsT{
				counts: s,
				ranks:  []int{curRank},
			}
			uniqueCountsSeries = append(uniqueCountsSeries, newCountsSeries)
		} else {
			uniqueCountsSeries[idx].ranks = append(uniqueCountsSeries[idx].ranks, curRank)
		}
		curRank++
	}

	for _, s := range uniqueCountsSeries {
		ranksStr := notation.CompressIntArray(s.ranks)
		compressedCounts = append(compressedCounts, fmt.Sprintf("Rank(s) %s: %s", ranksStr, strings.TrimRight(s.counts, "\n")))
	}

	return compressedCounts
}

func saveCountsInCompactFormat(fd *os.File, data []rawCountsCallsT, numCalls int, context string) error {
	for _, c := range data {
		_, err := fd.WriteString(compactCountsFileHeader)
		if err != nil {
			return err
		}

		_, err = fd.WriteString(fmt.Sprintf("%s%d\n", numberOfRanksMarker, c.counts.commSize))
		if err != nil {
			return err
		}

		if context == "S" {
			_, err = fd.WriteString(fmt.Sprintf("%s%d\n", datatypeSizeMarker, c.counts.sendDatatypeSize))
			if err != nil {
				return err
			}
		} else {
			_, err = fd.WriteString(fmt.Sprintf("%s%d\n", datatypeSizeMarker, c.counts.recvDatatypeSize))
			if err != nil {
				return err
			}

		}

		_, err = fd.WriteString(fmt.Sprintf("%s 0-%d\n", alltoallvCallNumbersMarker, numCalls-1))
		if err != nil {
			return err
		}

		compressedListCalls := notation.CompressIntArray(c.calls)
		_, err = fd.WriteString(fmt.Sprintf("%s%d calls - %s\n", marker, len(c.calls), compressedListCalls))
		if err != nil {
			return err
		}

		_, err = fd.WriteString(fmt.Sprintf("\n\n%s\n", beginningDataMarker))
		if err != nil {
			return err
		}

		if context == "S" {
			sendCompressedCounts := compressCounts(c.counts.sendCounts)
			_, err = fd.WriteString(strings.Join(sendCompressedCounts, "\n"))
			if err != nil {
				return err
			}
		} else {
			recvCompressedCounts := compressCounts(c.counts.recvCounts)
			_, err = fd.WriteString(strings.Join(recvCompressedCounts, "\n"))
			if err != nil {
				return err
			}
		}

		_, err = fd.WriteString(fmt.Sprintf("\n%s\n", endDataMarker))
		if err != nil {
			return err
		}
	}
	return nil
}

func saveAllCountsInCompactFormat(dir string, jobid int, leadRank int, numCalls int, sc []rawCountsCallsT, rc []rawCountsCallsT) error {
	sendCountFile := filepath.Join(dir, fmt.Sprintf("%sjob%d.rank%d.txt", SendCountersFilePrefix, jobid, leadRank))
	scFd, err := os.OpenFile(sendCountFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer scFd.Close()

	err = saveCountsInCompactFormat(scFd, sc, numCalls, "S")
	if err != nil {
		return err
	}

	recvCountFile := filepath.Join(dir, fmt.Sprintf("%sjob%d.rank%d.txt", RecvCountersFilePrefix, jobid, leadRank))
	rcFd, err := os.OpenFile(recvCountFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer rcFd.Close()

	err = saveCountsInCompactFormat(rcFd, rc, numCalls, "R")
	if err != nil {
		return err
	}

	return nil
}

func loadCommunicatorRawCounts(outputDir string, leadRank int, numCalls int, files []string) error {
	var rawSendCounts []rawCountsCallsT
	var rawRecvCounts []rawCountsCallsT

	b := progress.NewBar(len(files), fmt.Sprintf("Converting count files for communicator %d", leadRank))
	defer progress.EndBar(b)
	for _, file := range files {
		b.Increment(1)
		rc, err := parseRawFile(file)
		if err != nil {
			return err
		}

		_, callId, err := getInfoFromRawCountsFileName(file)
		if err != nil {
			return err
		}

		idx := rawSendCountsAlreadyExists(rc, rawSendCounts)
		if idx == -1 {
			newData := rawCountsCallsT{
				calls:  []int{callId},
				counts: &rc,
			}
			rawSendCounts = append(rawSendCounts, newData)
		} else {
			rawSendCounts[idx].calls = append(rawSendCounts[idx].calls, callId)
		}

		idx = rawRecvCountsAlreadyExists(rc, rawRecvCounts)
		if idx == -1 {
			newData := rawCountsCallsT{
				calls:  []int{callId},
				counts: &rc,
			}
			rawRecvCounts = append(rawRecvCounts, newData)
		} else {
			rawRecvCounts[idx].calls = append(rawRecvCounts[idx].calls, callId)
		}
	}

	err := saveAllCountsInCompactFormat(outputDir, 0, leadRank, numCalls, rawSendCounts, rawRecvCounts)
	if err != nil {
		return err
	}

	return nil
}

func LoadRawCountsFromFiles(listFiles []string, outputDir string) error {
	commMap := make(map[int][]string)
	numCalls := 0

	for _, file := range listFiles {
		leadRank, _, err := getInfoFromRawCountsFileName(file)
		if err != nil {
			return err
		}
		if _, ok := commMap[leadRank]; ok {
			commMap[leadRank] = append(commMap[leadRank], file)
		} else {
			commMap[leadRank] = []string{file}
		}
		numCalls++ // One call per file, we just parsed one file.
	}

	// Then we parse the file based on the leadRank, which ultimately lets us deal with sub-communicators
	for leadRank, files := range commMap {
		err := loadCommunicatorRawCounts(outputDir, leadRank, numCalls, files)
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadRawCountsFromDirs(dirs []string, outputDir string) error {
	commMap := make(map[int][]string)
	numCalls := 0

	// First we group all the file based on the lead rank of the communicator the call was made on
	for _, dir := range dirs {
		f, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range f {
			if !strings.HasPrefix(file.Name(), "counts.rank") {
				continue
			}

			leadRank, _, err := getInfoFromRawCountsFileName(file.Name())
			if err != nil {
				return err
			}
			if _, ok := commMap[leadRank]; ok {
				commMap[leadRank] = append(commMap[leadRank], filepath.Join(dir, file.Name()))
			} else {
				commMap[leadRank] = []string{filepath.Join(dir, file.Name())}
			}
			numCalls++ // One call per file, we just parsed one file.
		}
	}

	// Then we parse the file based on the leadRank, which ultimately lets us deal with sub-communicators
	for leadRank, files := range commMap {
		err := loadCommunicatorRawCounts(outputDir, leadRank, numCalls, files)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseRawFile(file string) (rawCountsT, error) {
	var rc rawCountsT

	fd, err := os.Open(file)
	if err != nil {
		return rc, fmt.Errorf("unable to open %s: %w", file, err)
	}
	defer fd.Close()
	reader := bufio.NewReader(fd)

	// First line is send datatype size
	line, err := reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	line = strings.TrimRight(line, "\n")
	rc.sendDatatypeSize, err = strconv.Atoi(strings.TrimLeft(line, rawCountsSendDatatypePrefix))
	if err != nil {
		return rc, err
	}

	// Second line is recv datatype size
	line, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	line = strings.TrimRight(line, "\n")
	rc.recvDatatypeSize, err = strconv.Atoi(strings.TrimLeft(line, rawCountsRecvDatatypePrefix))
	if err != nil {
		return rc, err
	}

	// Third line is comm size
	line, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	line = strings.TrimRight(line, "\n")
	rc.commSize, err = strconv.Atoi(strings.TrimLeft(line, rawCountsCommSizePrefix))
	if err != nil {
		return rc, err
	}

	// Forth we have an empty line
	_, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}

	// Then we have send counts
	line, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	line = strings.TrimRight(line, "\n")
	if line != rawCountsSendCountsPrefix {
		return rc, fmt.Errorf("Wrong format, we have %s instead of %s", line, rawCountsSendCountsPrefix)
	}
	for i := 0; i < rc.commSize; i++ {
		line, err = reader.ReadString('\n')
		if err != nil {
			return rc, err
		}
		rc.sendCounts = append(rc.sendCounts, strings.TrimRight(strings.TrimRight(line, "\n"), " "))
	}

	// We have two empty lines
	_, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}

	// Finally we have recv counts
	line, err = reader.ReadString('\n')
	if err != nil {
		return rc, err
	}
	line = strings.TrimRight(line, "\n")
	if line != rawCountsRecvCountsPrefix {
		return rc, fmt.Errorf("Wrong format, we have %s instead of %s", line, rawCountsRecvCountsPrefix)
	}
	for i := 0; i < rc.commSize; i++ {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF { // the last line does not include "\n"
			return rc, err
		}
		rc.recvCounts = append(rc.recvCounts, strings.TrimRight(strings.TrimRight(line, "\n"), " "))
	}

	return rc, nil
}
