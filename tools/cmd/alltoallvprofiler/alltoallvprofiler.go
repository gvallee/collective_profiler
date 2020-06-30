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
				ids = append(ids, id)
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
	return getIDsFromFileNames(files, "jobid")
}

func analyzeCountFiles(basedir string, sendCountFiles []string, recvCountFiles []string, sizeThreshold int) error {
	// Find all the files based on the rank who created the file.
	// Remember that we have more than one rank creating files, it means that different communicators were
	// used to run the alltoallv operations
	sendRanks, err := getRanksFromFileNames(sendCountFiles)
	if err != nil {
		return err
	}
	sort.Ints(sendRanks)

	recvRanks, err := getRanksFromFileNames(recvCountFiles)
	if err != nil {
		return err
	}
	sort.Ints(recvRanks)

	if !reflect.DeepEqual(sendRanks, recvRanks) {
		return fmt.Errorf("list of ranks logging send and receive counts differ, data likely to be corrupted")
	}

	sendJobids, err := getJobIDsFromFileNames(sendCountFiles)
	if err != nil {
		return err
	}

	if len(sendJobids) != 1 {
		return fmt.Errorf("more than one job detected through send counts files; inconsistent data?")
	}

	recvJobids, err := getJobIDsFromFileNames(recvCountFiles)
	if err != nil {
		return err
	}

	if len(recvJobids) != 1 {
		return fmt.Errorf("more than one job detected through recv counts files; inconsistent data?")
	}

	if sendJobids[0] != recvJobids[0] {
		return fmt.Errorf("results seem to be from different jobs, we strongly encourage users to get their counts data though a single run")
	}

	jobid := sendJobids[0]

	for _, rank := range sendRanks {
		sendCountFile, recvCountFile := datafilereader.GetCountsFiles(jobid, rank)
		sendCountFile = filepath.Join(basedir, sendCountFile)
		recvCountFile = filepath.Join(basedir, recvCountFile)

		numCalls, err := datafilereader.GetNumCalls(sendCountFile)
		if err != nil {
			log.Fatalf("unable to get the number of alltoallv calls: %s", err)
		}

		_, err = profiler.ParseCountFiles(sendCountFile, recvCountFile, numCalls, sizeThreshold)
		if err != nil {
			log.Fatalf("unable to parse count file %s", sendCountFile)
		}
	}

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	help := flag.Bool("h", false, "Help message")
	sizeThredshold := flag.Int("size-thredshold", 200, "Size to differentiate small and big messages")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyzes all the data gathered while running an application with our shared library", cmdName)
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	// Figure out all the send/recv counts
	f, err := ioutil.ReadDir(*dir)
	if err != nil {
		fmt.Printf("[ERROR] Unable to read %s: %s", *dir, err)
		os.Exit(1)
	}

	var profileFiles []string
	var sendCountsFiles []string
	var recvCountsFiles []string
	for _, file := range f {
		if strings.HasPrefix(file.Name(), datafilereader.ProfileSummaryFilePrefix) {
			profileFiles = append(profileFiles, filepath.Join(*dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), datafilereader.SendCountersFilePrefix) {
			sendCountsFiles = append(sendCountsFiles, filepath.Join(*dir, file.Name()))
		}

		if strings.HasPrefix(file.Name(), datafilereader.RecvCountersFilePrefix) {
			recvCountsFiles = append(recvCountsFiles, filepath.Join(*dir, file.Name()))
		}
	}

	err = analyzeCountFiles(*dir, sendCountsFiles, recvCountsFiles, *sizeThredshold)
	if err != nil {
		log.Fatalf("unable to analyze send count files: %s", err)
	}
}
