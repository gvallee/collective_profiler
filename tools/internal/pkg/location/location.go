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
	// LocationFileToken is the token used to identify location files
	LocationFileToken = "_locations_comm"

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

// RankLocation hosts all the data related to a rank location (i.e., rank on COMM_WORLD, the current communicator, the PID and the hostname)
type RankLocation struct {
	CommID        int
	CommWorldRank int
	CommRank      int
	PID           int
	Hostname      string
}

// Data represents all the data from a location file
type Data struct {
	RankLocations []*RankLocation
	Calls         []int
	RankfileData  RankFileData
	CommData      map[int]int // A map where the key is the rank on the communicator and the value its rank on COMM_WORLD
}

// GetLocationDataFromStrings specifically reads the core of the location data after reading all the content from a location file
// and converting the content to an array of strings.
// The function returns 2 different types of data: the list of calls covered by the file; the data from the file though an array of
// location Info and the rank map per host.
func GetLocationDataFromStrings(lines []string, readIndex int) (*Data, error) {
	info := new(Data)
	info.RankfileData.HostMap = make(map[string][]int)
	info.RankfileData.RankMap = make(map[int]string)

	// Read the communicator ID
	if !strings.HasPrefix(lines[readIndex], communicatorIDToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], communicatorIDToken)
	}
	commIDStr := strings.TrimRight(lines[readIndex], "\n")
	commIDStr = strings.TrimLeft(commIDStr, communicatorIDToken)
	commID, err := strconv.Atoi(commIDStr)
	if err != nil {
		return nil, err
	}
	readIndex++

	// Read the list of calls
	if !strings.HasPrefix(lines[readIndex], callsToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], callsToken)
	}
	callsListStr := strings.TrimRight(lines[readIndex], "\n")
	callsListStr = strings.TrimLeft(callsListStr, callsToken)
	info.Calls, err = notation.ConvertCompressedCallListToIntSlice(callsListStr)
	if err != nil {
		return nil, err
	}
	readIndex++

	// Read the list of associated COMM_WORLD ranks
	if !strings.HasPrefix(lines[readIndex], commWorldRanksToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], commWorldRanksToken)
	}
	commWorldRanksStr := strings.TrimRight(lines[readIndex], "\n")
	commWorldRanksStr = strings.TrimLeft(commWorldRanksStr, commWorldRanksToken)
	commWorldRanks, err := notation.ConvertCompressedCallListToIntSlice(commWorldRanksStr)
	if err != nil {
		return nil, err
	}
	readIndex++

	// Read the PIDs
	if !strings.HasPrefix(lines[readIndex], pidsToken) {
		return nil, fmt.Errorf("invalid format, %s does not start with %s", lines[readIndex], pidsToken)
	}
	pidsListStr := strings.TrimRight(lines[readIndex], "\n")
	pidsListStr = strings.TrimLeft(pidsListStr, pidsToken)
	pids, err := notation.ConvertCompressedCallListToIntSlice(pidsListStr)
	if err != nil {
		return nil, err
	}
	readIndex++

	// Read hostnames header
	if strings.TrimRight(lines[readIndex], "\n") != strings.TrimRight(hostnameToken, "\n") {
		return nil, fmt.Errorf("invalid format, %s is not expected %s", lines[readIndex], hostnameToken)
	}
	readIndex++
	// We do not need it right now, skip

	// Read all the ranks' location
	index := 0
	for readIndex < len(lines) {
		if lines[readIndex] == "" {
			readIndex++
			continue
		}

		if !strings.HasPrefix(lines[readIndex], rankToken) {
			return nil, fmt.Errorf("invalid format, %s is not expected %s (l.%d/%d)", lines[readIndex], rankToken, readIndex, len(lines))
		}
		line := strings.TrimRight(lines[readIndex], "\n")
		line = strings.TrimLeft(line, rankToken)
		tokens := strings.Split(line, ": ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid rank location format %s", line)
		}
		rank, err := strconv.Atoi(tokens[0])
		if err != nil {
			return nil, err
		}

		i := new(RankLocation)
		i.CommRank = rank
		i.CommWorldRank = commWorldRanks[index]
		i.Hostname = tokens[1]
		i.PID = pids[index]
		i.CommID = commID
		info.RankLocations = append(info.RankLocations, i)

		info.RankfileData.HostMap[i.Hostname] = append(info.RankfileData.HostMap[i.Hostname], i.CommWorldRank)

		// We update the rank map if necessary
		if _, ok := info.RankfileData.RankMap[i.CommWorldRank]; !ok {
			info.RankfileData.RankMap[i.CommWorldRank] = i.Hostname
		}

		index++
		readIndex++
	}

	return info, nil
}

// getLocations parses the core of a location file through a reader.
// In other terms, it assumes the file's header already has been parsed.
// It returns a map where the key is the absolute callID and the value the locations' data.
func getLocations(lines []string, readIndex int) (map[int][]*RankLocation, *Data, error) {
	locations := make(map[int][]*RankLocation)

	data, err := GetLocationDataFromStrings(lines, readIndex)
	if err != nil {
		return nil, nil, err
	}

	for _, callID := range data.Calls {
		locations[callID] = append(locations[callID], data.RankLocations...)
	}

	return locations, data, nil
}

// ParseLocationFile parses a location file and return the corresponding location of all ranks of the communicator per call
func ParseLocationFile(codeBaseDir string, path string) (map[int][]*RankLocation, *Data, error) {
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
	return getLocations(lines, readIndex)
}
