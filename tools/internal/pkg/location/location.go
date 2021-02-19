//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package location

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

const (
	communicatorIDToken = "Communicator ID: "
	callsToken          = "Calls: "
	commWorldRanksToken = "COMM_WORLD ranks: "
	pidsToken           = "PIDs: "
	hostnameToken       = "Hostnames:\n"
	rankToken           = "\tRank "
)

// RankFileData gathers data about the hostname and what COMMWORLD ranks are on each node
// This is static information for the entire duration of the application's execution
type RankFileData struct {
	HostMap map[string][]int // The key is the hostname, the value an array of COMMWORLD rank that are on that host
	RankMap map[int]string   // The key is the rank on COMMWORLD, the value the hostname
}

// Info hosts all the data related to a rank location (i.e., rank on COMM_WORLD, the current communicator, the PID and the hostname)
type Info struct {
	CommID        int
	CommWorldRank int
	CommRank      int
	PID           int
	Hostname      string
}

// GetLocationDataFromStrings specifically reads the core of the location data after reading all the content from a location file
// and converting the content to an array of strings.
func GetLocationDataFromStrings(lines []string, readIndex int, locationData *RankFileData) ([]int, []*Info, *RankFileData, error) {
	if locationData == nil {
		locationData = new(RankFileData)
		locationData.HostMap = make(map[string][]int)
		locationData.RankMap = make(map[int]string)
	}

	// Read the communicator ID
	if !strings.HasPrefix(lines[readIndex], communicatorIDToken) {
		return nil, nil, nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], communicatorIDToken)
	}
	commIDStr := strings.TrimRight(lines[readIndex], "\n")
	commIDStr = strings.TrimLeft(commIDStr, communicatorIDToken)
	commID, err := strconv.Atoi(commIDStr)
	if err != nil {
		return nil, nil, nil, err
	}
	readIndex++

	// Read the list of calls
	if !strings.HasPrefix(lines[readIndex], callsToken) {
		return nil, nil, nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], callsToken)
	}
	callsListStr := strings.TrimRight(lines[readIndex], "\n")
	callsListStr = strings.TrimLeft(callsListStr, callsToken)
	calls, err := notation.ConvertCompressedCallListToIntSlice(callsListStr)
	if err != nil {
		return nil, nil, nil, err
	}
	readIndex++

	// Read the list of associated COMM_WORLD ranks
	if !strings.HasPrefix(lines[readIndex], commWorldRanksToken) {
		return nil, nil, nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], commWorldRanksToken)
	}
	commWorldRanksStr := strings.TrimRight(lines[readIndex], "\n")
	commWorldRanksStr = strings.TrimLeft(commWorldRanksStr, commWorldRanksToken)
	commWorldRanks, err := notation.ConvertCompressedCallListToIntSlice(commWorldRanksStr)
	if err != nil {
		return nil, nil, nil, err
	}
	readIndex++

	// Read the PIDs
	if !strings.HasPrefix(lines[readIndex], pidsToken) {
		return nil, nil, nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], pidsToken)
	}
	pidsListStr := strings.TrimRight(lines[readIndex], "\n")
	pidsListStr = strings.TrimLeft(pidsListStr, pidsToken)
	pids, err := notation.ConvertCompressedCallListToIntSlice(pidsListStr)
	if err != nil {
		return nil, nil, nil, err
	}
	readIndex++

	// Read hostnames header
	if strings.TrimRight(lines[readIndex], "\n") != strings.TrimRight(hostnameToken, "\n") {
		return nil, nil, nil, fmt.Errorf("invalid format, %s is not expected %s", lines[readIndex], hostnameToken)
	}
	readIndex++
	// We do not need it right now, skip

	// Read all the ranks' location
	index := 0
	var l []*Info
	for readIndex < len(lines) {
		if lines[readIndex] == "" {
			readIndex++
			continue
		}

		if !strings.HasPrefix(lines[readIndex], rankToken) {
			return nil, nil, nil, fmt.Errorf("invalid format, %s is not expected %s (l.%d/%d)", lines[readIndex], rankToken, readIndex, len(lines))
		}
		line := strings.TrimRight(lines[readIndex], "\n")
		line = strings.TrimLeft(line, rankToken)
		tokens := strings.Split(line, ": ")
		if len(tokens) != 2 {
			return nil, nil, nil, fmt.Errorf("invalid rank location format %s", line)
		}
		rank, err := strconv.Atoi(tokens[0])
		if err != nil {
			return nil, nil, nil, err
		}

		i := new(Info)
		i.CommRank = rank
		i.CommWorldRank = commWorldRanks[index]
		i.Hostname = tokens[1]
		i.PID = pids[index]
		i.CommID = commID
		l = append(l, i)

		// We update the host map is necessary
		found := false
		if val, ok := locationData.HostMap[i.Hostname]; ok {
			// Check if the rank is already in the list
			for _, knownRank := range val {
				if knownRank == i.CommWorldRank {
					found = true
					break
				}
			}
			if !found {
				locationData.HostMap[i.Hostname] = append(locationData.HostMap[i.Hostname], i.CommWorldRank)
			}
		}

		// We update the rank map if necessary
		if _, ok := locationData.RankMap[i.CommWorldRank]; !ok {
			locationData.RankMap[i.CommWorldRank] = i.Hostname
		}

		index++
		readIndex++
	}

	return calls, l, locationData, nil
}

// getLocations parses the core of a location file through a reader.
// In other terms, it assumes the file's header already has been parsed.
// It returns a map where the key is the absolute callID and the value the locations' data.
func getLocations(lines []string, readIndex int, locationData *RankFileData) (map[int][]*Info, *RankFileData, error) {
	locations := make(map[int][]*Info)

	calls, l, locationData, err := GetLocationDataFromStrings(lines, readIndex, locationData)
	if err != nil {
		return nil, nil, err
	}

	for _, callID := range calls {
		locations[callID] = l
	}

	return locations, locationData, nil
}

// ParseLocationFile parses a location file and return the corresponding location of all ranks of the communicator
func ParseLocationFile(codeBaseDir string, path string, locationData *RankFileData) (map[int][]*Info, *RankFileData, error) {
	readIndex := 0
	// Get the data from the file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	// Populate the rank map
	lines := strings.Split(string(content), "\n")

	// First line must be the data format
	formatMatch, err := format.CheckDataFormatLineFromProfileFile(lines[readIndex], codeBaseDir)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse format version: %s", err)
	}
	if !formatMatch {
		return nil, nil, fmt.Errorf("data format does not match")
	}
	readIndex++

	// Followed by an empty line
	readIndex++

	// Populate the rank map
	return getLocations(lines, readIndex, locationData)
}
