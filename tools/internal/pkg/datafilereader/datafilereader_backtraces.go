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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

const (
	backtracesDirName             = "backtraces"
	alltoallvCallerFilenamePrefix = "alltoallv_caller_"
	backtraceFileHeaderPrefix     = "Alltoallv calls: "
)

func readBacktraceFile(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// First line is the list of calls
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(line, backtraceFileHeaderPrefix) {
		return "", "", fmt.Errorf("invalid backtrace file header: %s", line)
	}

	calls := strings.TrimRight(line, "\n")
	calls = strings.TrimLeft(calls, backtraceFileHeaderPrefix)

	trace := ""
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", "", err
		}

		trace += line
	}

	return calls, trace, nil
}

func getCallBacktrace(dir string, callNum int) (string, error) {
	backtracesDir := filepath.Join(dir, backtracesDirName)
	files, err := ioutil.ReadDir(backtracesDir)
	if err != nil {
		return "", err
	}

	var backtracesFiles []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), alltoallvCallerFilenamePrefix) {
			backtracesFiles = append(backtracesFiles, filepath.Join(dir, backtracesDirName, f.Name()))
		}
	}

	for _, f := range backtracesFiles {
		log.Printf("-> Reading %s\n", f)
		callIDsStr, trace, err := readBacktraceFile(f)
		callIDs, err := notation.ConvertCompressedCallListToIntSlice(callIDsStr)
		if err != nil {
			return "", err
		}

		for _, i := range callIDs {
			if i == callNum {
				return trace, nil
			}
		}
	}
	return "", fmt.Errorf("unable to find backtrace for call #%d", callNum)
}
