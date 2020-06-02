//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package datafilereader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

func CompareCallPatterns(p1 map[int]int, p2 map[int]int) bool {
	if len(p1) != len(p2) {
		return false
	}

	return reflect.DeepEqual(p1, p2)
}

func GetPatternHeader(reader *bufio.Reader) ([]int, string, error) {
	var callIDs []int
	callIDsStr := ""

	line, readerErr := reader.ReadString('\n')
	if readerErr != nil {
		return callIDs, callIDsStr, readerErr
	}

	// Are we at the beginning of a metadata block?
	if !strings.HasPrefix(line, "## Pattern #") {
		return callIDs, callIDsStr, fmt.Errorf("[ERROR] not a header (line: %s)", line)
	}

	line, readerErr = reader.ReadString('\n')
	if readerErr != nil {
		return callIDs, callIDsStr, readerErr
	}

	if !strings.HasPrefix(line, "Alltoallv calls: ") {
		return callIDs, callIDsStr, fmt.Errorf("[ERROR] not a header (line: %s)", line)
	}

	var err error
	callIDsStr = strings.TrimLeft(line, "Alltoallv calls: ")
	callIDsStr = strings.TrimRight(callIDsStr, "\n")
	callIDs, err = notation.ConvertCompressedCallListToIntSlice(callIDsStr)
	if err != nil {
		return callIDs, callIDsStr, err
	}

	return callIDs, callIDsStr, nil
}

func getPatterns(reader *bufio.Reader) (string, error) {
	patterns := ""

	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return patterns, readerErr
		}
		if readerErr == io.EOF {
			break
		}

		if line == "" || line == "\n" {
			// end of pattern data
			break
		}

		if strings.HasPrefix(line, "Alltoallv calls: ") {
			return patterns, fmt.Errorf("header and patterns parser are compromised: %s", line)
		}

		patterns += line

	}

	return patterns, nil
}

func GetPatternFilePath(basedir string, jobid int, pid int) string {
	return filepath.Join(basedir, fmt.Sprintf("patterns-job%d-pid%d.md", jobid, pid))
}

func getCallPatterns(dir string, jobid int, pid int, callNum int) (string, error) {
	patternsOutputFile := GetPatternFilePath(dir, jobid, pid)
	patternsFd, err := os.Open(patternsOutputFile)
	if err != nil {
		return "", err
	}
	defer patternsFd.Close()
	patternsReader := bufio.NewReader(patternsFd)

	// The very first line should be '#Patterns'
	line, readerErr := patternsReader.ReadString('\n')
	if readerErr != nil {
		return "", readerErr
	}
	if line != "# Patterns\n" {
		return "", fmt.Errorf("wrong file format: %s", line)
	}

	for {
		callIDs, _, err := GetPatternHeader(patternsReader)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("unable to read %s: %w", patternsOutputFile, err)
		}
		if err == io.EOF {
			break
		}

		targetBlock := false
		for _, c := range callIDs {
			if c == callNum {
				targetBlock = true
				break
			}
		}

		if targetBlock {
			patterns, err := getPatterns(patternsReader)
			if err != nil {
				return "", nil
			}
			return patterns, nil
		} else {
			_, err := getPatterns(patternsReader)
			if err != nil {
				return "", nil
			}
		}
	}

	return "", fmt.Errorf("unable to find data for call %d", callNum)
}
