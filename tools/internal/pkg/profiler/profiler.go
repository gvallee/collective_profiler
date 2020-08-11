//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package profiler

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/backtraces"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/bins"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/timings"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/errors"
)

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

// CallInfo gathers all the data extracted about a specific alltoallv call
type CallInfo struct {
	// ID is the call number (zero-indexed)
	ID int

	// CountsData is the data gathered after parsing the send and receive counts files
	CountsData counts.CallData

	// Stats gathers all the communication patterns associated to the alltoallv call
	Patterns patterns.Data

	// PatternStr is the string version of the communication patterns
	PatternStr string

	// Timings represent all the timings associated to the alltoallv call (e.g., late arrival and execution timings)
	Timings timings.CallTimings

	// Backtrace is the string version of the alltoallv call's backtrace
	Backtrace string

	// SendStats gives all the statistics and data gathered while parsing the count file of the alltoallv call
	SendStats counts.Stats

	RecvStats counts.Stats
}

func LookupCall(sendCountsFile string, recvCountsFile string, numCall int, msgSizeThreshold int) (CallInfo, error) {
	var info CallInfo
	var profilerErr *errors.ProfilerError

	info.CountsData, profilerErr = counts.LookupCall(sendCountsFile, recvCountsFile, numCall)
	if !profilerErr.Is(errors.ErrNone) {
		return info, profilerErr.GetInternal()
	}
	//info.CommSize = info.CountsStats.CommSize

	// todo: get the patterns here. Call counts.AnalyzeCounts?

	return info, nil
}

func containsCall(callNum int, calls []int) bool {
	for i := 0; i < len(calls); i++ {
		if calls[i] == callNum {
			return true
		}
	}
	return false
}

func GetCallRankData(sendCountersFile string, recvCountersFile string, callNum int, rank int) (int, int, error) {
	sendCounters, sendDatatypeSize, _, err := counts.ReadCallRankCounters([]string{sendCountersFile}, rank, callNum)
	if err != nil {
		return 0, 0, err
	}
	recvCounters, recvDatatypeSize, _, err := counts.ReadCallRankCounters([]string{recvCountersFile}, rank, callNum)
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

// AnalyzeSubCommsResults go through the results and analyzes results specific
// to sub-communicators cases
func AnalyzeSubCommsResults(dir string, stats map[int]counts.SendRecvStats, allPatterns map[int]patterns.Data) error {
	numPatterns := -1
	numNtoNPatterns := -1
	num1toNPatterns := -1
	numNto1Patterns := -1
	var referencePatterns patterns.Data

	// At the moment, we do a very basic analysis: are the patterns the same on all sub-communicators?
	for _, p := range allPatterns {
		if numPatterns == -1 {
			numPatterns = len(p.AllPatterns)
			numNto1Patterns = len(p.NToOne)
			numNtoNPatterns = len(p.NToN)
			num1toNPatterns = len(p.OneToN)
			referencePatterns = p
			continue
		}

		if numPatterns != len(p.AllPatterns) ||
			numNto1Patterns != len(p.NToOne) ||
			numNtoNPatterns != len(p.NToN) ||
			num1toNPatterns != len(p.OneToN) {
			return nil
		}

		if !patterns.Same(referencePatterns, p) {
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
	multicommHighlightFile := filepath.Join(dir, format.MulticommHighlightFilePrefix+".md")
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

	if len(allPatterns[ranks[0]].NToN) > 0 {
		err := patterns.WriteSubcommNtoNPatterns(fd, ranks, stats, allPatterns)
		if err != nil {
			return err
		}
	}

	if len(allPatterns[ranks[0]].OneToN) > 0 {
		err := patterns.WriteSubcomm1toNPatterns(fd, ranks, stats, allPatterns)
		if err != nil {
			return err
		}
	}

	if len(allPatterns[ranks[0]].NToOne) > 0 {
		err := patterns.WriteSubcommNto1Patterns(fd, ranks, stats, allPatterns)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n## All 0 counts pattern; no data exchanged\n\n")
	if err != nil {
		return err
	}
	for _, rank := range ranks {
		if len(allPatterns[rank].Empty) > 0 {
			_, err = fd.WriteString(fmt.Sprintf("-> Sub-communicator led by rank %d: %d/%d alltoallv calls\n", rank, len(allPatterns[rank].Empty), stats[rank].TotalNumCalls))
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
		/* FIXME!!!!!!!!!
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
		*/
	}

	return nil
}

// GetCallData extract all the data related to a specific call.
func GetCallData(dir string, jobid int, rank int, callNum int, msgSizeThreshold int) (CallInfo, error) {
	var info CallInfo
	info.ID = callNum

	// Load the counts from raw data
	log.Printf("Extracting send/receive counts for call #%d\n", callNum)
	sendCountsFile, recvCountsFile := counts.GetFiles(jobid, rank)
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

	var profilerErr *errors.ProfilerError
	var countsHeader counts.HeaderT
	countsHeader, info.CountsData.SendData.RawCounts, profilerErr = counts.LookupCallFromFile(sendCountsFileReader, callNum)
	if !profilerErr.Is(errors.ErrNone) {
		return info, profilerErr.GetInternal()
	}
	info.CountsData.CommSize = countsHeader.NumRanks
	info.CountsData.SendData.Statistics.DatatypeSize = countsHeader.DatatypeSize
	countsHeader, info.CountsData.RecvData.RawCounts, profilerErr = counts.LookupCallFromFile(recvCountsFileReader, callNum)
	if !profilerErr.Is(errors.ErrNone) {
		return info, profilerErr.GetInternal()
	}
	if info.CountsData.CommSize != countsHeader.NumRanks {
		return info, fmt.Errorf("Communicator of different size: %d vs. %d", info.CountsData.CommSize, countsHeader.NumRanks)
	}
	info.CountsData.RecvData.Statistics.DatatypeSize = countsHeader.DatatypeSize

	info.SendStats, err = counts.AnalyzeCounts(info.CountsData.SendData.RawCounts, msgSizeThreshold, info.CountsData.SendData.Statistics.DatatypeSize)
	if err != nil {
		return info, err
	}
	info.RecvStats, err = counts.AnalyzeCounts(info.CountsData.RecvData.RawCounts, msgSizeThreshold, info.CountsData.RecvData.Statistics.DatatypeSize)
	if err != nil {
		return info, err
	}

	// Get timings from formatted timing file
	// todo: if the files do not exist, we should get the data from scratch

	log.Printf("Extracting timings for call #%d\n", callNum)
	info.Timings, err = timings.GetCallData(dir, jobid, rank, callNum)
	if err != nil {
		return info, err
	}
	gps, err := info.Timings.LateArrivalsTimings.Grouping.GetGroups()
	if err != nil {
		return info, err
	}
	if len(gps) > 1 {
		fmt.Printf("[WARN] %d groups of late arrival times have been found\n", len(gps))
	} else {
		fmt.Printf("[INFO] No outliers in late arrival times\n")
	}
	gps, err = info.Timings.ExecutionTimings.Grouping.GetGroups()
	if err != nil {
		return info, err
	}
	if len(gps) > 1 {
		fmt.Printf("[WARN] %d groups of execution time have been found\n", len(gps))
	} else {
		fmt.Printf("[INFO] No outliers in execution times\n")
	}

	// Load patterns from result file.
	// todo: if the file does not exists, we should get the data from scratch
	log.Printf("Extracting patterns for call #%d\n", callNum)
	info.PatternStr, err = patterns.GetCall(dir, jobid, rank, callNum)
	if err != nil {
		return info, err
	}

	// Load the backtrace
	log.Printf("Extracting backtrace for call #%d\n", callNum)
	info.Backtrace, err = backtraces.GetCall(dir, callNum)
	if err != nil {
		return info, err
	}

	return info, nil
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

func SaveStats(info OutputFileInfo, cs counts.SendRecvStats, patternsData patterns.Data, numCalls int, sizeThreshold int) error {
	_, err := info.defaultFd.WriteString(fmt.Sprintf("Total number of alltoallv calls: %d\n\n", numCalls))
	if err != nil {
		return err
	}

	err = counts.WriteDatatypeToFile(info.defaultFd, numCalls, cs.DatatypesSend, cs.DatatypesRecv)
	if err != nil {
		return err
	}

	err = counts.WriteCommunicatorSizesToFile(info.defaultFd, numCalls, cs.CommSizes)
	if err != nil {
		return err
	}

	err = counts.WriteCountStatsToFile(info.defaultFd, numCalls, sizeThreshold, cs)
	if err != nil {
		return err
	}

	err = patterns.WriteData(info.patternsFd, info.patternsSummaryFd, patternsData, numCalls)
	if err != nil {
		return err
	}

	return nil
}

func GetCountProfilerFileDesc(basedir string, jobid int, rank int) (OutputFileInfo, error) {
	var info OutputFileInfo
	var err error

	info.defaultOutputFile = datafilereader.GetStatsFilePath(basedir, jobid, rank)
	info.patternsOutputFile = patterns.GetFilePath(basedir, jobid, rank)
	info.patternsSummaryOutputFile = patterns.GetSummaryFilePath(basedir, jobid, rank)
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

	log.Println("Results are saved in:")
	log.Printf("-> %s\n", info.defaultOutputFile)
	log.Printf("-> %s\n", info.patternsOutputFile)
	log.Printf("Patterns summary: %s\n", info.patternsSummaryOutputFile)

	return info, nil
}

func analyzeJobRankCounts(basedir string, jobid int, rank int, sizeThreshold int, listBins []int) (map[int]*counts.CallData, counts.SendRecvStats, patterns.Data, error) {
	var p patterns.Data
	var sendRecvStats counts.SendRecvStats
	var cs map[int]*counts.CallData // The key is the call number and the value a pointer to the call's data (several calls can share the same data)
	sendCountFile, recvCountFile := counts.GetFiles(jobid, rank)
	sendCountFile = filepath.Join(basedir, sendCountFile)
	recvCountFile = filepath.Join(basedir, recvCountFile)

	numCalls, err := counts.GetNumCalls(sendCountFile)
	if err != nil {
		return nil, sendRecvStats, p, fmt.Errorf("unable to get the number of alltoallv calls: %s", err)
	}

	// Note that by extracting the patterns, it will implicitly parses the send/recv counts
	// since it is necessary to figure out patterns.
	cs, p, err = patterns.ParseFiles(sendCountFile, recvCountFile, numCalls, rank, sizeThreshold)
	if err != nil {
		return cs, sendRecvStats, p, fmt.Errorf("unable to parse count file %s: %s", sendCountFile, err)
	}

	b := progress.NewBar(len(cs), "Bin creation")
	defer progress.EndBar(b)
	for _, callData := range cs {
		b.Increment(1)
		callData.SendData.BinThresholds = listBins
		sendBins := bins.Create(listBins)
		sendBins, err = bins.GetFromCounts(callData.SendData.RawCounts, sendBins, callData.SendData.Statistics.TotalNumCalls, callData.SendData.Statistics.DatatypeSize)
		if err != nil {
			return cs, sendRecvStats, p, err
		}
		err = bins.Save(basedir, jobid, rank, sendBins)
		if err != nil {
			return cs, sendRecvStats, p, err
		}
	}

	sendRecvStats, err = counts.GatherStatsFromCallData(cs, sizeThreshold)
	if err != nil {
		return cs, sendRecvStats, p, err
	}

	outputFilesInfo, err := GetCountProfilerFileDesc(basedir, jobid, rank)
	defer outputFilesInfo.Cleanup()

	err = SaveStats(outputFilesInfo, sendRecvStats, p, numCalls, sizeThreshold)
	if err != nil {
		return cs, sendRecvStats, p, fmt.Errorf("unable to save counters' stats: %s", err)
	}

	return cs, sendRecvStats, p, nil
}

func analyzeCountFiles(basedir string, sendCountFiles []string, recvCountFiles []string, sizeThreshold int, listBins []int) (int, map[int]counts.SendRecvStats, map[int]patterns.Data, []counts.CommDataT, error) {
	// Find all the files based on the rank who created the file.
	// Remember that we have more than one rank creating files, it means that different communicators were
	// used to run the alltoallv operations
	sendRanks, err := datafilereader.GetRanksFromFileNames(sendCountFiles)
	if err != nil || len(sendRanks) == 0 {
		return 0, nil, nil, nil, err
	}
	sort.Ints(sendRanks)

	recvRanks, err := datafilereader.GetRanksFromFileNames(recvCountFiles)
	if err != nil || len(recvRanks) == 0 {
		return 0, nil, nil, nil, err
	}
	sort.Ints(recvRanks)

	if !reflect.DeepEqual(sendRanks, recvRanks) {
		return 0, nil, nil, nil, fmt.Errorf("list of ranks logging send and receive counts differ, data likely to be corrupted")
	}

	sendJobids, err := datafilereader.GetJobIDsFromFileNames(sendCountFiles)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	if len(sendJobids) != 1 {
		return 0, nil, nil, nil, fmt.Errorf("more than one job detected through send counts files; inconsistent data? (len: %d)", len(sendJobids))
	}

	recvJobids, err := datafilereader.GetJobIDsFromFileNames(recvCountFiles)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	if len(recvJobids) != 1 {
		return 0, nil, nil, nil, fmt.Errorf("more than one job detected through recv counts files; inconsistent data?")
	}

	if sendJobids[0] != recvJobids[0] {
		return 0, nil, nil, nil, fmt.Errorf("results seem to be from different jobs, we strongly encourage users to get their counts data though a single run")
	}

	jobid := sendJobids[0]
	allStats := make(map[int]counts.SendRecvStats)
	allPatterns := make(map[int]patterns.Data)
	totalNumCalls := 0

	var allCallsData []counts.CommDataT
	for _, rank := range sendRanks {
		callsData, sendRecvStats, p, err := analyzeJobRankCounts(basedir, jobid, rank, sizeThreshold, listBins)
		if err != nil {
			return 0, nil, nil, nil, err
		}
		totalNumCalls += len(callsData)
		allStats[rank] = sendRecvStats
		allPatterns[rank] = p

		d := counts.CommDataT{
			LeadRank: rank,
			CallData: callsData,
		}
		allCallsData = append(allCallsData, d)
	}

	return totalNumCalls, allStats, allPatterns, allCallsData, nil
}

// FindCompactFormatCountsFiles figures out all the send/recv counts files that are in
// the compact format
func FindCompactFormatCountsFiles(dir string) ([]string, []string, []string, error) {
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	var profileFiles []string
	var sendCountsFiles []string
	var recvCountsFiles []string
	for _, file := range f {
		if strings.HasPrefix(file.Name(), format.ProfileSummaryFilePrefix) {
			profileFiles = append(profileFiles, filepath.Join(dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), counts.SendCountersFilePrefix) {
			sendCountsFiles = append(sendCountsFiles, filepath.Join(dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), counts.RecvCountersFilePrefix) {
			recvCountsFiles = append(recvCountsFiles, filepath.Join(dir, file.Name()))
		}
	}

	return profileFiles, sendCountsFiles, recvCountsFiles, nil
}

func HandleCountsFiles(dir string, sizeThreshold int, listBins []int) (int, map[int]counts.SendRecvStats, map[int]patterns.Data, []counts.CommDataT, error) {
	_, sendCountsFiles, recvCountsFiles, err := FindCompactFormatCountsFiles(dir)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	// Analyze all the files we found
	return analyzeCountFiles(dir, sendCountsFiles, recvCountsFiles, sizeThreshold, listBins)
}

// FindRawCountFiles walks all the sub-directories from a given directory and
// create the list of all the raw count files (so not in compact format). This is
// useful when the raw counts files were generated by the profiler and stored in
// various sub-directory for convenience.
func FindRawCountFiles(dir string) ([]string, []string) {
	var rawCountsFiles []string
	rawCountsDirs := make(map[string]bool)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		filename := filepath.Base(path)
		if strings.HasPrefix(filename, counts.RawCountersFilePrefix) {
			dir := filepath.Dir(path)
			rawCountsFiles = append(rawCountsFiles, path)
			if _, ok := rawCountsDirs[dir]; !ok {
				rawCountsDirs[dir] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil
	}

	var dirs []string
	for dir, _ := range rawCountsDirs {
		dirs = append(dirs, dir)
	}

	return rawCountsFiles, dirs
}
