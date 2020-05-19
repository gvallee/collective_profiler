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
	sendCountersFilePrefix = "send-counters.pid"
	recvCountersFilePrefix = "recv-counters.pid"
)

/*
func convertCompressedCallListtoIntSlice(str string) ([]int, error) {
	var callIDs []int

	fmt.Printf("Found range: %s\n", str)
	tokens := strings.Split(str, ", ")
	for _, t := range tokens {
		tokens2 := strings.Split(t, "-")
		for _, t2 := range tokens2 {
			n, err := strconv.Atoi(t2)
			if err != nil {
				return callIDs, fmt.Errorf("unable to parse %s", str)
			}
			callIDs = append(callIDs, n)
		}
	}

	return callIDs, nil
}
*/

func findCountersFilesWithPrefix(basedir string, id string, prefix string) ([]string, error) {
	var files []string

	f, err := ioutil.ReadDir(basedir)
	if err != nil {
		return files, fmt.Errorf("[ERROR] unable to read %s: %w", basedir, err)
	}

	for _, file := range f {
		if strings.HasPrefix(file.Name(), prefix) && strings.Contains(file.Name(), "pid"+id) {
			path := filepath.Join(basedir, file.Name())
			files = append(files, path)
		}
	}

	return files, nil
}

func findSendCountersFiles(basedir string, id int) ([]string, error) {
	idStr := strconv.Itoa(id)
	return findCountersFilesWithPrefix(basedir, idStr, sendCountersFilePrefix)
}

func findRecvCountersFiles(basedir string, id int) ([]string, error) {
	idStr := strconv.Itoa(id)
	return findCountersFilesWithPrefix(basedir, idStr, recvCountersFilePrefix)
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
	rankStr := strconv.Itoa(rank)

	for i := 0; i < len(callCounters); i++ {
		ranksListStr := strings.Split(strings.ReplaceAll(strings.Split(callCounters[i], ": ")[0], "Rank(s) ", ""), " ")
		for j := 0; j < len(ranksListStr); j++ {
			if ranksListStr[j] == rankStr {
				return strings.Split(callCounters[i], ": ")[1], nil
			}
		}
	}

	return "", fmt.Errorf("unable to find counters for rank %d", rank)
}

func findCounters(files []string, rank int, callNum int) (string, bool, error) {
	counters := ""

	for _, f := range files {
		log.Printf("Scanning %s...\n", f)
		file, err := os.Open(f)
		if err != nil {
			return "", false, fmt.Errorf("unable to open %s: %w", f, err)
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		for {
			log.Println("-> Reading header...")
			_, callIDs, _, readerErr1 := datafilereader.GetHeader(reader)
			if readerErr1 != nil && readerErr1 != io.EOF {
				fmt.Printf("ERROR: %s", readerErr1)
				return counters, false, fmt.Errorf("unable to read header from %s: %w", f, readerErr1)
			}

			var readerErr2 error
			var callCounters []string
			if containsCall(callNum, callIDs) {
				log.Println("Reading counters...")
				callCounters, readerErr2 = datafilereader.GetCounters(reader)
				if readerErr2 != nil && readerErr2 != io.EOF {
					return counters, false, readerErr2
				}
				counters, err := extractRankCounters(callCounters, rank)
				if err != nil {
					return counters, false, err
				}

				if readerErr2 == io.EOF {
					break
				}

				return counters, true, nil
			} else {
				// The current counters are not about the call we care about, skipping...
				_, err := datafilereader.GetCounters(reader)
				if err != nil {
					return counters, false, err
				}
			}

			if readerErr1 == io.EOF || readerErr2 == io.EOF {
				break
			}
		}
	}

	log.Printf("unable to find data for rank %d in call %d", rank, callNum)
	return counters, false, nil
}

func findSendCounters(basedir string, id int, rank int, callNum int) (string, error) {
	files, err := findSendCountersFiles(basedir, id)
	if err != nil {
		return "", err
	}
	counters, found, err := findCounters(files, rank, callNum)
	if !found {
		return "", fmt.Errorf("unable to find counters for rank %d in call %d", rank, callNum)
	}

	return counters, nil
}

func findRecvCounters(basedir string, id int, rank int, callNum int) (string, error) {
	files, err := findRecvCountersFiles(basedir, id)
	if err != nil {
		return "", err
	}
	counters, found, err := findCounters(files, rank, callNum)
	if !found {
		return "", fmt.Errorf("unable to find counters for rank %d in call %d", rank, callNum)
	}

	return counters, nil
}

func FindCounters(basedir string, id int, rank int, callNum int) (string, string, error) {
	sendCounters, err := findSendCounters(basedir, id, rank, callNum)
	if err != nil {
		return "", "", err
	}

	recvCounters, err := findRecvCounters(basedir, id, rank, callNum)
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

func Validate(id int, dir string) error {
	// Find all the data randomly generated during the execution of the app
	idStr := strconv.Itoa(id)
	files, err := getValidationFiles(dir, idStr)
	if err != nil {
		return err
	}

	// For each file, load the counters with our framework and compare with the data we got directly from the app
	for _, f := range files {
		id, rank, call, err := getInfoFromFilename(f)
		if err != nil {
			return err
		}

		sendCounters1, recvCounters1, err := getCountersFromValidationFile(f)
		if err != nil {
			return err
		}

		sendCounters2, recvCounters2, err := FindCounters(dir, id, rank, call)
		if err != nil {
			return err
		}

		if sendCounters1 != sendCounters2 {
			return fmt.Errorf("Send counters do not match: '%s' vs. '%s'", sendCounters1, sendCounters2)
		}

		if recvCounters1 != recvCounters2 {
			return fmt.Errorf("Receive counters do not match: '%s' vs. '%s'", recvCounters1, recvCounters2)
		}
	}

	return nil
}
