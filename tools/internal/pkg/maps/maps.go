//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package maps

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/errors"
)

const (
	// Heat is used as an identifier to refer to heat map
	Heat = 1

	heatFilePrefix     = "heat-map."
	locationFilePrefix = "locations_"
	commWorldID        = "COMMWORLD rank: "
	commID             = "COMM rank: "
	pidID              = "PID: "
	hostnameID         = "Hostname: "

	commHeatMapPrefix   = "heat-map-subcomm"
	globalHeatMapPrefix = "heat-map.txt"
)

type Location struct {
	CommWorldRank int
	CommRank      int
	PID           int
	Hostname      string
}

func getCallidRankFromLocationFile(path string) (int, int, error) {
	log.Printf("Parsing %s...", path)
	str := filepath.Base(path)
	str = strings.ReplaceAll(str, locationFilePrefix, "")
	str = strings.ReplaceAll(str, ".md", "")
	tokens := strings.Split(str, "_")
	if len(tokens) != 2 {
		return -1, -1, fmt.Errorf("invalid filename format: %s", path)
	}

	rankStr := strings.ReplaceAll(tokens[0], "rank", "")
	callidStr := strings.ReplaceAll(tokens[1], "call", "")
	rank, err := strconv.Atoi(rankStr)
	if err != nil {
		return -1, -1, err
	}
	callid, err := strconv.Atoi(callidStr)
	if err != nil {
		return -1, -1, err
	}

	return callid, rank, nil
}

func parseLocationFile(path string) ([]Location, error) {
	var locations []Location

	// Get the data from the file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Populate the rank map
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var l Location
		tokens := strings.Split(line, " - ")
		if len(tokens) != 4 {
			return nil, fmt.Errorf("invalid file content: %s", line)
		}

		commWorldRankStr := strings.TrimLeft(tokens[0], commWorldID)
		l.CommWorldRank, err = strconv.Atoi(commWorldRankStr)
		if err != nil {
			return nil, err
		}

		commRankStr := strings.TrimLeft(tokens[1], commID)
		l.CommRank, err = strconv.Atoi(commRankStr)
		if err != nil {
			return nil, err
		}

		pidStr := strings.TrimLeft(tokens[2], pidID)
		l.PID, err = strconv.Atoi(pidStr)
		if err != nil {
			return nil, err
		}

		l.Hostname = strings.TrimLeft(tokens[3], hostnameID)

		locations = append(locations, l)
	}

	return locations, nil
}

func findCountFile(countsFiles []string, rankId int) string {
	for _, file := range countsFiles {
		if strings.Contains(file, ".rank"+strconv.Itoa(rankId)+".") {
			return file
		}
	}
	return ""
}

func getRankMapFromLocations(locations []Location) map[int]int {
	m := make(map[int]int)
	for _, l := range locations {
		m[l.CommRank] = l.CommWorldRank
	}
	return m
}

func saveHeatMap(heatmap map[int]int, filepath string) error {
	fd, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()
	for key, value := range heatmap {
		_, err := fd.WriteString(fmt.Sprintf("Rank %d: %d bytes\n", key, value))
		if err != nil {
			return err
		}
	}
	return nil
}

func getCallInfo(countFile string, callID int) (int, []string, error) {
	f, err := os.Open(countFile)
	if err != nil {
		return -1, nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	_, datatypeSize, callCounts, profilerErr := counts.LookupCallFromFile(reader, callID)
	if !profilerErr.Is(errors.ErrNone) {
		return -1, nil, profilerErr.GetInternal()
	}

	return datatypeSize, callCounts, nil
}

func createHeatMap(dir string) error {
	// Find all the files that have location data
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var locationFiles []string
	var countsFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), locationFilePrefix) {
			locationFiles = append(locationFiles, filepath.Join(dir, file.Name()))
		}
		if strings.HasPrefix(file.Name(), counts.SendCountersFilePrefix) {
			countsFiles = append(countsFiles, filepath.Join(dir, file.Name()))
		}
	}

	heatMap := make(map[int]int) // The comm world rank is the key, the value amount of data sent to it
	bar := progress.NewBar(len(locationFiles), "Rank location files")
	defer progress.EndBar(bar)
	for _, file := range locationFiles {
		bar.Increment(1)
		// This is not correct, commHeatMap will give the heat map for the last call that will be analyzed, it does not make sense.
		commHeatMap := make(map[int]int)

		callID, rankID, err := getCallidRankFromLocationFile(file)
		if err != nil {
			return err
		}

		countFile := findCountFile(countsFiles, rankID)

		l, err := parseLocationFile(file)
		if err != nil {
			return err
		}

		// Get call info
		datatypeSize, callCounts, err := getCallInfo(countFile, callID)
		if err != nil {
			return err
		}

		ranksMap := getRankMapFromLocations(l)

		// Now we can have send counts for all the ranks on the communicator as well as th translation comm rank to COMMWORLD rank
		// We can populate the heat map
		for _, counts := range callCounts {
			counts = strings.TrimRight(counts, "\n")

			// We need to clean up the string callCounts since it also has the list of sending ranks,
			// which we do not care about here
			tokens := strings.Split(counts, ": ")
			if len(tokens) != 2 {
				return fmt.Errorf("wrong counts format: %s", counts)
			}
			counts = tokens[1]

			tokens = strings.Split(counts, " ")
			curRank := 0 // curRank is also the rank of the communicator
			for _, countStr := range tokens {
				if countStr == "" {
					continue
				}
				worldRank := ranksMap[curRank]
				count, err := strconv.Atoi(countStr)
				if err != nil {
					return err
				}
				heatMap[worldRank] += count * datatypeSize
				commHeatMap[worldRank] += count * datatypeSize
				curRank++
			}
		}

		// Save the communicator-based heat map
		commHeatMapFilePath := filepath.Join(dir, commHeatMapPrefix+".rank"+strconv.Itoa(rankID)+".txt")
		err = saveHeatMap(commHeatMap, commHeatMapFilePath)
		if err != nil {
			return err
		}
	}

	// Save the heat map for the entire execution
	globalHeatMapFilePath := filepath.Join(dir, globalHeatMapPrefix)
	err = saveHeatMap(heatMap, globalHeatMapFilePath)
	if err != nil {
		return err
	}

	return nil
}

func Create(id int, dir string) error {
	switch id {
	case Heat:
		return createHeatMap(dir)
	}

	return fmt.Errorf("unknown map type: %d", id)
}
