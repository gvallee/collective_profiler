//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"
	"github.com/gvallee/go_util/pkg/util"
)

func getIDsFromFileNames(files []string, id string) ([]int, error) {
	var ids []int
	for _, file := range files {
		tokens := strings.Split(filepath.Base(file), ".")
		for _, t := range tokens {
			if strings.Contains(t, id) {
				t = strings.ReplaceAll(t, id, "")
				id, err := strconv.Atoi(t)
				if err != nil {
					return ids, err
				}
				found := false
				for _, i := range ids {
					if i == id {
						found = true
					}
				}
				if !found {
					ids = append(ids, id)
				}
				break
			}
		}
	}

	return ids, nil
}

func getRanksFromFileNames(files []string) ([]int, error) {
	return getIDsFromFileNames(files, "rank")
}

func getJobIDsFromFileNames(files []string) ([]int, error) {
	return getIDsFromFileNames(files, "job")
}

func analyzeJobRankCounts(basedir string, jobid int, rank int, sizeThreshold int) (profiler.CountStats, error) {
	var cs profiler.CountStats

	sendCountFile, recvCountFile := datafilereader.GetCountsFiles(jobid, rank)
	sendCountFile = filepath.Join(basedir, sendCountFile)
	recvCountFile = filepath.Join(basedir, recvCountFile)

	numCalls, err := datafilereader.GetNumCalls(sendCountFile)
	if err != nil {
		return cs, fmt.Errorf("unable to get the number of alltoallv calls: %s", err)
	}

	cs, err = profiler.ParseCountFiles(sendCountFile, recvCountFile, numCalls, sizeThreshold)
	if err != nil {
		return cs, fmt.Errorf("unable to parse count file %s", sendCountFile)
	}

	outputFilesInfo, err := profiler.GetCountProfilerFileDesc(basedir, jobid, rank)
	if err != nil {
		return cs, fmt.Errorf("unable to open output files: %s", err)
	}
	defer outputFilesInfo.Cleanup()

	err = profiler.SaveCounterStats(outputFilesInfo, cs, numCalls, sizeThreshold)
	if err != nil {
		return cs, fmt.Errorf("unable to save counters' stats: %s", err)
	}

	return cs, nil
}

func analyzeCountFiles(basedir string, sendCountFiles []string, recvCountFiles []string, sizeThreshold int) (map[int]profiler.CountStats, error) {
	// Find all the files based on the rank who created the file.
	// Remember that we have more than one rank creating files, it means that different communicators were
	// used to run the alltoallv operations
	sendRanks, err := getRanksFromFileNames(sendCountFiles)
	if err != nil {
		return nil, err
	}
	sort.Ints(sendRanks)

	recvRanks, err := getRanksFromFileNames(recvCountFiles)
	if err != nil {
		return nil, err
	}
	sort.Ints(recvRanks)

	if !reflect.DeepEqual(sendRanks, recvRanks) {
		return nil, fmt.Errorf("list of ranks logging send and receive counts differ, data likely to be corrupted")
	}

	sendJobids, err := getJobIDsFromFileNames(sendCountFiles)
	if err != nil {
		return nil, err
	}

	if len(sendJobids) != 1 {
		return nil, fmt.Errorf("more than one job detected through send counts files; inconsistent data? (len: %d)", len(sendJobids))
	}

	recvJobids, err := getJobIDsFromFileNames(recvCountFiles)
	if err != nil {
		return nil, err
	}

	if len(recvJobids) != 1 {
		return nil, fmt.Errorf("more than one job detected through recv counts files; inconsistent data?")
	}

	if sendJobids[0] != recvJobids[0] {
		return nil, fmt.Errorf("results seem to be from different jobs, we strongly encourage users to get their counts data though a single run")
	}

	jobid := sendJobids[0]
	allStats := make(map[int]profiler.CountStats)
	for _, rank := range sendRanks {
		cs, err := analyzeJobRankCounts(basedir, jobid, rank, sizeThreshold)
		if err != nil {
			return nil, err
		}
		allStats[rank] = cs
	}

	return allStats, nil
}

func handleCountsFiles(dir string, sizeThreshold int) (map[int]profiler.CountStats, error) {
	// Figure out all the send/recv counts files
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var profileFiles []string
	var sendCountsFiles []string
	var recvCountsFiles []string
	for _, file := range f {
		if strings.HasPrefix(file.Name(), datafilereader.ProfileSummaryFilePrefix) {
			profileFiles = append(profileFiles, filepath.Join(dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), datafilereader.SendCountersFilePrefix) {
			sendCountsFiles = append(sendCountsFiles, filepath.Join(dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), datafilereader.RecvCountersFilePrefix) {
			recvCountsFiles = append(recvCountsFiles, filepath.Join(dir, file.Name()))
		}
	}

	// Analyze all the files we found
	stats, err := analyzeCountFiles(dir, sendCountsFiles, recvCountsFiles, sizeThreshold)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func analyzeTimingsFiles(dir string, files []string) error {
	for _, file := range files {
		// The output directory is where the data is, this tool keeps all the data together
		err := profiler.ParseTimingsFile(file, dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleTimingFiles(dir string) error {
	// Figure out all the send/recv counts files
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var timingsFiles []string
	for _, file := range f {
		if strings.HasPrefix(file.Name(), datafilereader.TimingsFilePrefix) {
			timingsFiles = append(timingsFiles, filepath.Join(dir, file.Name()))
		}
	}

	// Analyze all the files we found
	err = analyzeTimingsFiles(dir, timingsFiles)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	help := flag.Bool("h", false, "Help message")
	sizeThreshold := flag.Int("size-threshold", 200, "Size to differentiate small and big messages")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyzes all the data gathered while running an application with our shared library", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	stats, err := handleCountsFiles(*dir, *sizeThreshold)
	if err != nil {
		fmt.Printf("ERROR: unable to analyze counts: %s", err)
		os.Exit(1)
	}

	err = handleTimingFiles(*dir)
	if err != nil {
		fmt.Printf("ERROR: unable to analyze timings: %s", err)
		os.Exit(1)
	}

	err = profiler.AnalyzeSubCommsResults(*dir, stats)
	if err != nil {
		fmt.Printf("ERROR: unable to analyze sub-communicators results: %s", err)
		os.Exit(1)
	}
}
