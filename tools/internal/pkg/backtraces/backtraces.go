//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package backtraces

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

const (
	filenameToken         = "_backtrace_rank"
	communicatorToken     = "Communicator: "
	communicatorRankToken = "Communicator rank: "
	commWorldRankToken    = "COMM_WORLD rank: "
	callsToken            = "Calls: "
)

// ReadBacktraceFile parses a given backtrace file and populate a map where the key is the call number and the value the trace.
// If the map passed in nil, a new map is created.
func ReadBacktraceFile(codeBaseDir string, path string, m map[int]string) (map[int]string, error) {
	if m == nil {
		m = make(map[int]string)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// First line must be the data format
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	formatMatch, err := format.CheckDataFormatLineFromProfileFile(line, codeBaseDir)
	if err != nil {
		return nil, fmt.Errorf("unable to parse format version: %s", err)
	}
	if !formatMatch {
		return nil, fmt.Errorf("data format does not match")
	}

	// Followed by an empty line
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("invalid data format, second line is %s instead of an empty line", line)
	}

	// Followed by a reference to the binary and PID; and another empty line
	// ATM we do not use it, we skip it.
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("invalid data format, second line is %s instead of an empty line", line)
	}

	// Then we have the actual trace
	// Header first, then trace data
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "# Trace\n" {
		return nil, fmt.Errorf("invalid data format, %s was read instead of '# Trace'", line)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("invalid data format, second line is %s instead of an empty line", line)
	}

	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	trace := ""
	for line != "\n" {
		trace += line
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}

	// Finally we have the different context, from which we can get the list of calls
	// Header first, then context's data
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "# Context ") {
		return nil, fmt.Errorf("invalid data format, %s was read instead of '# Context'", line)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("invalid data format, second line is %s instead of an empty line", line)
	}

	// Read the communicator ID
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, communicatorToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", line, communicatorToken)
	}
	// We do not need it right now, skip

	// Read the Communicator rank
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, communicatorRankToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", line, communicatorRankToken)
	}
	// We do not need it right now, skip

	// Read the COMM_WORLD rank
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, commWorldRankToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", line, commWorldRankToken)
	}
	// We do not need it right now, skip

	// Read the list of associated calls
	line, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, callsToken) {
		return nil, fmt.Errorf("invalid format, %s does nto start with %s", line, callsToken)
	}
	line = strings.TrimLeft(line, callsToken)
	line = strings.TrimRight(line, "\n")
	calls, err := notation.ConvertCompressedCallListToIntSlice(line)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %s", line, err)
	}
	for _, callID := range calls {
		if _, ok := m[callID]; ok {
			return nil, fmt.Errorf("backtrace for call %d is defined more than once", callID)
		}
		m[callID] = trace
	}

	return m, nil
}

func findBacktraceFiles(dir string, collectiveName string) ([]string, error) {
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var list []string
	for _, file := range f {
		filename := file.Name()
		if strings.HasPrefix(filename, collectiveName) && strings.Contains(filename, filenameToken) {
			list = append(list, filepath.Join(dir, filename))
		}
	}

	return list, nil
}

// GetCall returns the backtrace data for a specific collective call
func GetCall(codeBaseDir string, dir string, collectiveName string, callNum int) (string, error) {
	backtraceFiles, err := findBacktraceFiles(dir, collectiveName)
	if err != nil {
		return "", err
	}

	for _, backtraceFile := range backtraceFiles {
		callsData, err := ReadBacktraceFile(codeBaseDir, backtraceFile, nil)
		if err != nil {
			return "", err
		}
		if trace, ok := callsData[callNum]; ok {
			return trace, nil
		}
	}
	return "", fmt.Errorf("unable to find backtrace for call #%d", callNum)
}
