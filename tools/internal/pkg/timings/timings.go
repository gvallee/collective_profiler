//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package timings

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/grouping"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/go_util/pkg/util"
)

const (
	// separateTimingsFiles represents the case where the profiler generates two file: one for all a2a execution time and one for all late arrival timings
	separateTimingsFiles = iota

	// singleProfileFile identifies the old format where the profiler generated a single file with all the timings in it
	singleProfileFile

	// execTimingsFilenameIdentifier is the string to use to identify files that have timing data
	execTimingsFilenameIdentifier = "_execution_times.rank"

	lateArrivalTimingsFilenameIdentifier = "_late_arrival_times.rank"

	callIDtoken = "# Call "
)

// Stats is the structure used to save all stats related to timings
type Stats struct {
	Timings string
	Data    []float64
	Max     float64
	Min     float64

	Mean     float64
	Grouping *grouping.Engine
}

// CallTimings is the structure used to store all timing data related to a specific collective operation.
type CallTimings struct {
	ExecutionTimings    *Stats
	LateArrivalsTimings *Stats
}

// Metadata is the metadata from a timing file
type Metadata struct {
	formatVersion int
	NumCalls      int
	NumRanks      int
}

// CommT is the data required to identify a communicator in a unique manner
type CommT struct {
	LeadRank int
	CommID   int
}

// CollectiveTimings is the data structure used to store all the timing data for a specific collective (e.g., alltoallv, alltoall)
type CollectiveTimings struct {
	execFiles           []string
	lateArrivalFiles    []string
	execTimesMetadata   *Metadata
	lateArrivalMetadata *Metadata
	// ExecTimes is a hash of a hash of a hash: leadRank/commID -> callID -> rank
	ExecTimes        map[CommT]map[int]map[int]float64
	LateArrivalTimes map[CommT]map[int]map[int]float64
	execStats        Stats
	lateArrivalStats Stats
}

func getExecTimingsFilePath(collective string, dir string, jobid int, commID int, rank int) string {
	filename := collective + execTimingsFilenameIdentifier + strconv.Itoa(rank) + "_comm" + strconv.Itoa(commID) + "_job_" + strconv.Itoa(jobid)
	return filepath.Join(dir, filename)
}

func getLateArrivalTimingsFilePath(collective string, dir string, jobid int, commID int, rank int) string {
	filename := collective + lateArrivalTimingsFilenameIdentifier + strconv.Itoa(rank) + "_comm" + strconv.Itoa(commID) + "_job_" + strconv.Itoa(jobid)
	return filepath.Join(dir, filename)
}

func (s *Stats) groupTimings() error {
	if s.Grouping == nil {
		s.Grouping = grouping.Init()
	}

	var ints []int
	for _, d := range s.Data {
		ints = append(ints, int(d))
	}

	for i := 0; i < len(ints); i++ {
		err := s.Grouping.AddDatapoint(i, ints)
		if err != nil {
			return err
		}
	}

	gps, err := s.Grouping.GetGroups()
	if err != nil {
		return err
	}
	fmt.Printf("Number of groups: %d", len(gps))

	return nil
}

// getMetadataFromFilename returns the metadata contained in the name of a timing file, i.e., the lead rank, communicator ID, job ID
func getMetadataFromFilename(filename string) (int, int, int, error) {
	idx := strings.LastIndex(filename, "rank")
	rankStr := filename[idx+4:]
	rankStr = strings.TrimRight(rankStr, ".md")
	rankStr = strings.TrimRight(rankStr, ".dat")
	tokens := strings.Split(rankStr, "_")
	if len(tokens) != 3 {
		return -1, -1, -1, fmt.Errorf("unable to parse filename %s", filename)
	}
	rankStr = tokens[0]
	rank, err := strconv.Atoi(rankStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("unable to parse filename %s: %s", filename, err)
	}
	commIDstr := strings.TrimLeft(tokens[1], "comm")
	commid, err := strconv.Atoi(commIDstr)
	if err != nil {
		return -1, -1, -1, err
	}
	jobIDstr := strings.TrimLeft(tokens[2], "job")
	jobid, err := strconv.Atoi(jobIDstr)
	if err != nil {
		return -1, -1, -1, err
	}
	return rank, commid, jobid, nil
}

func readMetaData(reader *bufio.Reader, codeBaseDir string, skipCheckMetadata bool) (*Metadata, error) {
	// First line must be the format version and it should matches the one used by the
	// postmortem tool
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var md *Metadata
	md = nil

	if !skipCheckMetadata {
		line = strings.TrimRight(line, "\n")

		if !strings.HasPrefix(line, format.DataFormatHeader) {
			return nil, fmt.Errorf("Invalid data format, format version missing")
		}

		dataFormatVersion, err := strconv.Atoi(strings.TrimLeft(line, format.DataFormatHeader))
		if err != nil {
			return nil, err
		}
		err = format.CheckDataFormat(dataFormatVersion, codeBaseDir)
		if err != nil {
			return nil, err
		}
		md = new(Metadata)
		md.formatVersion = dataFormatVersion
	}

	// Second line must be empty
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("data format impose an empty line but we have instead %s", line)
	}

	return md, nil
}

func getMetaData(reader *bufio.Reader, codeBaseDir string) (*Metadata, error) {
	return readMetaData(reader, codeBaseDir, false)
}

func skipMetaData(reader *bufio.Reader) error {
	_, err := readMetaData(reader, "", true)
	return err
}

func getCallID(reader *bufio.Reader) (int, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return -1, err
	}
	line = strings.TrimRight(line, "\n")

	if !strings.HasPrefix(line, callIDtoken) {
		return -1, fmt.Errorf("invalid format %s does not start with %s", line, callIDtoken)
	}

	callID, err := strconv.Atoi(strings.TrimLeft(line, callIDtoken))
	if err != nil {
		return -1, err
	}
	return callID, nil
}

// getCallTimings returns the timings for a specific call in the form of a map where the key is the rank and the value the timing
func getCallTimings(reader *bufio.Reader) (map[int]float64, error) {
	rankData := make(map[int]float64)
	rank := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			// real error
			return nil, err
		}
		if line == "\n" || (err != nil && err == io.EOF) {
			// end of the call's data
			return rankData, nil
		}

		line = strings.TrimRight(line, "\n")
		timing, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return nil, err
		}
		rankData[rank] = timing
		rank++
	}
}

func getCallData(reader *bufio.Reader) (*Stats, error) {
	stats := new(Stats)
	stats.Max = 0.0
	stats.Min = -1.0
	stats.Timings = ""
	sum := 0.0
	num := 0.0 // we use a float here to avoid having golang complain about dividing a float by an int

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			// real error
			return nil, err
		}
		if line == "\n" || (err != nil && err == io.EOF) {
			// end of the call's data, before we exit, we calculate the mean
			stats.Mean = sum / num
			return stats, nil
		}

		stats.Timings += line
		line = strings.TrimRight(line, "\n")

		timing, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return nil, err
		}
		stats.Data = append(stats.Data, timing)

		sum += timing

		if stats.Min == -1.0 || stats.Min > timing {
			stats.Min = timing
		}
		if stats.Max < timing {
			stats.Max = timing
		}
		num++
	}
}

// ParseTimingFile returns the timings data from a given timing file.
// The function returns some metadata (e.g., number of calls, number of ranks) as well as the timings for each call,
// as well as the total time per rank (time map).
func ParseTimingFile(filePath string, codeBaseDir string) (*Metadata, map[int]map[int]float64, map[int]float64, error) {
	timingFile, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer timingFile.Close()
	r := bufio.NewReader(timingFile)

	md, err := getMetaData(r, codeBaseDir)
	if err != nil {
		return nil, nil, nil, err
	}
	if md == nil {
		return nil, nil, nil, fmt.Errorf("metadata for %s is missing", filePath)
	}

	// map of map: calls -> ranks -> rank's timing
	callsData := make(map[int]map[int]float64)

	totalTimesPerRank := make(map[int]float64)

	// Now we can read the timing data for all the calls in the file
	for {
		callID, err := getCallID(r)
		if err != nil && err == io.EOF {
			// we reached the end of the file, we exit with everything we read so far
			return md, callsData, totalTimesPerRank, nil
		}
		if err != nil && err != io.EOF {
			// this is an actual error
			return nil, nil, nil, err
		}

		// We have the header for a new call, we update some metadata
		md.NumCalls++

		ranksTimings, err := getCallTimings(r)
		if err != nil {
			// We should never get an error here since we successfully called getCallID()
			return nil, nil, nil, err
		}

		if md.NumRanks == 0 {
			// First call we are parsing, we save some metadata
			md.NumRanks = len(ranksTimings)
		}

		if md.NumRanks != len(ranksTimings) {
			return nil, nil, nil, fmt.Errorf("inconsistent timing file, call %d has %d ranks instead of %d", callID, len(ranksTimings), md.NumRanks)
		}

		// Update the total time per rank map
		for rank, time := range ranksTimings {
			totalTimesPerRank[rank] += time
		}

		callsData[callID] = ranksTimings
	}
}

func (collectiveInfo *CollectiveTimings) analyzeTimingsFiles(codeBaseDir string, dir string, collectiveName string, totalExecutionTimes map[int]float64, totalLateArrivalTimes map[int]float64) error {
	// comm-centric maps where the keys are: CommT, callID and world rank; the value an array of time
	collectiveInfo.ExecTimes = make(map[CommT]map[int]map[int]float64)
	collectiveInfo.LateArrivalTimes = make(map[CommT]map[int]map[int]float64)

	numFilesToParse := 0
	for _, execFile := range collectiveInfo.execFiles {
		if util.FileExists(execFile) {
			numFilesToParse++
		}
	}

	for _, lateArrivalFile := range collectiveInfo.lateArrivalFiles {
		if util.FileExists(lateArrivalFile) {
			numFilesToParse++
		}
	}

	bar := progress.NewBar(numFilesToParse, "Analyzing timings files for "+collectiveName)
	defer progress.EndBar(bar)

	for _, execFile := range collectiveInfo.execFiles {
		if util.FileExists(execFile) {
			bar.Increment(1)
			leadRank, commid, _, err := getMetadataFromFilename(execFile)
			if err != nil {
				return err
			}
			execTimesMetadata, execTimes, execRankTimeMap, err := ParseTimingFile(execFile, codeBaseDir)
			if err != nil {
				return err
			}
			collectiveInfo.execTimesMetadata = execTimesMetadata
			commIdentifier := CommT{
				CommID:   commid,
				LeadRank: leadRank,
			}
			collectiveInfo.ExecTimes[commIdentifier] = execTimes

			for rank, time := range execRankTimeMap {
				totalExecutionTimes[rank] += time
			}
		}
	}

	for _, lateArrivalFile := range collectiveInfo.lateArrivalFiles {
		if util.FileExists(lateArrivalFile) {
			bar.Increment(1)
			leadRank, commid, _, err := getMetadataFromFilename(lateArrivalFile)
			if err != nil {
				return err
			}
			lateArrivalMetadata, lateTimes, lateArrivalRankTimeMap, err := ParseTimingFile(lateArrivalFile, codeBaseDir)
			if err != nil {
				return err
			}
			collectiveInfo.lateArrivalMetadata = lateArrivalMetadata
			commIdentifier := CommT{
				CommID:   commid,
				LeadRank: leadRank,
			}
			collectiveInfo.LateArrivalTimes[commIdentifier] = lateTimes

			for rank, time := range lateArrivalRankTimeMap {
				totalLateArrivalTimes[rank] += time
			}
		}
	}

	return nil
}

// HandleTimingFiles is a high-level function that will find all the timings related files
// generated by the profiler and analyze them
func HandleTimingFiles(codeBaseDir string, dir string, totalNumCalls int) (map[string]*CollectiveTimings, map[int]float64, map[int]float64, error) {
	// Detect all timings file, regardless of their format
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	totalExecutionTimes := make(map[int]float64)
	totalLateArrivalTimes := make(map[int]float64)

	data := make(map[string]*CollectiveTimings)
	for _, file := range f {
		filename := file.Name()
		if !strings.Contains(filename, execTimingsFilenameIdentifier) && !strings.Contains(filename, lateArrivalTimingsFilenameIdentifier) {
			// Not a timing profile file
			continue
		}

		// Get collective name from filename
		indexFirstDelimiter := strings.Index(filename, "_")
		if indexFirstDelimiter == -1 {
			return nil, nil, nil, fmt.Errorf("Timing data file does not follow format: %s", filename)
		}
		collectiveName := filename[0:indexFirstDelimiter]

		if strings.Contains(filename, execTimingsFilenameIdentifier) {
			if _, ok := data[collectiveName]; !ok {
				collectiveData := new(CollectiveTimings)
				data[collectiveName] = collectiveData
			}
			collectiveData := data[collectiveName]
			collectiveData.execFiles = append(collectiveData.execFiles, filepath.Join(dir, filename))
			data[collectiveName] = collectiveData
		}

		if strings.Contains(filename, lateArrivalTimingsFilenameIdentifier) {
			if _, ok := data[collectiveName]; !ok {
				collectiveData := new(CollectiveTimings)
				data[collectiveName] = collectiveData
			}
			collectiveData := data[collectiveName]
			collectiveData.lateArrivalFiles = append(collectiveData.lateArrivalFiles, filepath.Join(dir, filename))
			data[collectiveName] = collectiveData
		}
	}

	// Analyze all the files we found
	for collectiveName, collectiveData := range data {
		err := collectiveData.analyzeTimingsFiles(codeBaseDir, dir, collectiveName, totalExecutionTimes, totalLateArrivalTimes)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return data, totalExecutionTimes, totalLateArrivalTimes, nil
}

func getCallDataFromFile(filePath string, numCall int) (*Stats, error) {
	timingFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer timingFile.Close()
	r := bufio.NewReader(timingFile)

	err = skipMetaData(r)
	if err != nil {
		return nil, err
	}

	// Now we can read the timing data for all the calls in the file
	for {
		callID, err := getCallID(r)
		if err != nil && err == io.EOF {
			// we reached the end of the file, we exit with everything we read so far
			return nil, fmt.Errorf("unable to find call %d from %s", numCall, filePath)
		}
		if err != nil && err != io.EOF {
			// this is an actual error
			return nil, err
		}

		if callID < numCall {
			continue
		}

		if callID == numCall {
			stats, err := getCallData(r)
			if err != nil {
				// We should never get an error here since we successfully called getCallID()
				return nil, err
			}

			return stats, nil
		}
	}
}

// GetCallData returns all the data for a specific call, including raw text
func GetCallData(collectiveName string, dir string, commID int, jobID int, leadRank int, callID int) (*CallTimings, error) {
	var err error
	callData := new(CallTimings)

	execTimingsFilepath := getExecTimingsFilePath(collectiveName, dir, jobID, commID, leadRank)
	callData.ExecutionTimings, err = getCallDataFromFile(execTimingsFilepath, callID)
	if err != nil {
		return nil, err
	}

	lateArrivalTimingsFilepath := getLateArrivalTimingsFilePath(collectiveName, dir, jobID, commID, leadRank)
	callData.LateArrivalsTimings, err = getCallDataFromFile(lateArrivalTimingsFilepath, callID)
	if err != nil {
		return nil, err
	}

	return callData, nil
}

func readBlockFromSingleTimingFile(reader *bufio.Reader) []string {
	var lines []string
	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return nil
		}
		if readerErr != nil && readerErr == io.EOF {
			break
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			// This is the end of the block
			break
		}
		lines = append(lines, line)
	}
	return lines
}

// GetExecTimingFilename returns the expected execution time file name for a given configuration
func GetExecTimingFilename(collectiveName string, leadRank int, commID int, jobID int) string {
	return collectiveName + execTimingsFilenameIdentifier + strconv.Itoa(leadRank) + "_comm" + strconv.Itoa(commID) + "_job" + strconv.Itoa(jobID) + ".md"
}

// GetLateArrivalTimingFilename returns the expected late arrival time file name for a given configuration
func GetLateArrivalTimingFilename(collectiveName string, leadRank int, commID int, jobID int) string {
	return collectiveName + lateArrivalTimingsFilenameIdentifier + strconv.Itoa(leadRank) + "_comm" + strconv.Itoa(commID) + "_job" + strconv.Itoa(jobID) + ".md"
}
