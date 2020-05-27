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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
)

const (
	sendCountersFilePrefix = "send-counters."
	recvCountersFilePrefix = "recv-counters."
)

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

func findSendCountersFiles(basedir string, jobid int, pid int) ([]string, error) {
	pidStr := strconv.Itoa(pid)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, pidStr, sendCountersFilePrefix)
}

func findRecvCountersFiles(basedir string, jobid int, pid int) ([]string, error) {
	pidStr := strconv.Itoa(pid)
	jobIDStr := strconv.Itoa(jobid)
	return findCountersFilesWithPrefix(basedir, jobIDStr, pidStr, recvCountersFilePrefix)
}

func GetCountsFiles(jobid int, pid int) (string, string) {
	suffix := "job" + strconv.Itoa(jobid) + ".pid" + strconv.Itoa(pid) + ".txt"
	return sendCountersFilePrefix + suffix, recvCountersFilePrefix + suffix
}

func containsCall(callNum int, calls []int) bool {
	for i := 0; i < len(calls); i++ {
		if calls[i] == callNum {
			return true
		}
	}
	return false
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

func findCallRankCounters(files []string, rank int, callNum int) (string, bool, error) {
	counters := ""
	found := false

	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			return "", found, fmt.Errorf("unable to open %s: %w", f, err)
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		for {
			_, _, callIDs, _, _, _, readerErr1 := datafilereader.GetHeader(reader)

			if readerErr1 != nil && readerErr1 != io.EOF {
				fmt.Printf("ERROR: %s", readerErr1)
				return counters, found, fmt.Errorf("unable to read header from %s: %w", f, readerErr1)
			}

			targetCall := false
			for i := 0; i < len(callIDs); i++ {
				if callIDs[i] == callNum {
					targetCall = true
					break
				}
			}

			var readerErr2 error
			var callCounters []string
			if targetCall == true {
				callCounters, readerErr2 = datafilereader.GetCounters(reader)
				if readerErr2 != nil && readerErr2 != io.EOF {
					return counters, found, readerErr2
				}

				counters, err = extractRankCounters(callCounters, rank)
				if err != nil {
					return counters, found, err
				}
				found = true

				return counters, found, nil
			} else {
				// The current counters are not about the call we care about, skipping...
				_, err := datafilereader.GetCounters(reader)
				if err != nil {
					return counters, found, err
				}
			}

			if readerErr1 == io.EOF || readerErr2 == io.EOF {
				break
			}
		}
	}

	return counters, found, fmt.Errorf("unable to find data for rank %d in call %d", rank, callNum)
}

func findCallRankSendCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, error) {
	files, err := findSendCountersFiles(basedir, jobid, pid)
	if err != nil {
		return "", err
	}
	counters, _, err := findCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("* unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func findCallRankRecvCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, error) {
	files, err := findRecvCountersFiles(basedir, jobid, pid)
	if err != nil {
		return "", err
	}
	counters, _, err := findCallRankCounters(files, rank, callNum)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("unable to find counters for rank %d in call %d: %s", rank, callNum, err)
	}

	return counters, nil
}

func FindCallRankCounters(basedir string, jobid int, pid int, rank int, callNum int) (string, string, error) {
	sendCounters, err := findCallRankSendCounters(basedir, jobid, pid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	recvCounters, err := findCallRankRecvCounters(basedir, jobid, pid, rank, callNum)
	if err != nil {
		return "", "", err
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")
	sendCounters = strings.TrimRight(sendCounters, " ")
	recvCounters = strings.TrimRight(recvCounters, " ")

	return sendCounters, recvCounters, nil
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
		pid, rank, call, err := getInfoFromFilename(f)
		if err != nil {
			return err
		}

		log.Printf("Looking up counters for rank %d during call %d\n", rank, call)
		sendCounters1, recvCounters1, err := getCountersFromValidationFile(f)
		if err != nil {
			fmt.Printf("unable to get counters from validation data: %s", err)
			return err
		}

		sendCounters2, recvCounters2, err := FindCallRankCounters(dir, jobid, pid, rank, call)
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
