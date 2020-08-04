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
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/hash"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/errors"
)

const (
	// Heat is used as an identifier to refer to heat map
	Heat = 1

	heatFilePrefix     = "heat-map."
	commRankMapPrefix  = "comm-rank-map."
	locationFilePrefix = "locations_"
	commWorldID        = "COMMWORLD rank: "
	commID             = "COMM rank: "
	pidID              = "PID: "
	hostnameID         = "Hostname: "

	commHeatMapPrefix   = "heat-map-subcomm"
	callHeatMapPrefix   = "heat-map.rank"
	globalHeatMapPrefix = "heat-map.txt"
	rankFilename        = "rankfile.txt"
)

type rankFileData struct {
	hostMap map[string][]int // The key is the hostname, the value an array of COMMWORLD rank that are on that host
	rankMap map[int]string   // The key is the rank on COMMWORLD, the value the hostname
}

type Location struct {
	CommWorldRank int
	CommRank      int
	PID           int
	Hostname      string
}

type commFiles struct {
	calls []int
	files []string
}

type locationFileInfo struct {
	callID int
	file   string
}

type rankMapInfo struct {
	calls []int
	file  string
}

type CommDataT struct {
	// LeadRank is the rank on COMMWORLD that is rank 0 on the communicator used for the alltoallv operation
	LeadRand int

	// Maps gathers the rank map and the host map
	Maps rankFileData
}

func getCallidRankFromLocationFile(path string) (int, int, error) {
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

func getLocationsFromStrings(lines []string) ([]Location, error) {
	var locations []Location
	var err error
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

func parseLocationFile(path string) ([]Location, error) {

	// Get the data from the file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Populate the rank map
	lines := strings.Split(string(content), "\n")
	return getLocationsFromStrings(lines)
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

func createMapFromCounts(callCounts []string, datatypeSize int, ranksMap map[int]int /*ranksLocation []Location*/, globalHeatMap map[int]int) (map[int]int, error) {
	// Now we can have send counts for all the ranks on the communicator as well as th translation comm rank to COMMWORLD rank
	// We can populate the heat map
	callHeatMap := make(map[int]int)

	for _, counts := range callCounts {
		counts = strings.TrimRight(counts, "\n")

		// We need to clean up the string callCounts since it also has the list of sending ranks,
		// which we do not care about here
		tokens := strings.Split(counts, ": ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("wrong counts format: %s", counts)
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
				return nil, err
			}
			globalHeatMap[worldRank] += count * datatypeSize
			callHeatMap[worldRank] += count * datatypeSize
			curRank++
		}
	}
	return callHeatMap, nil
}

func createHeatMap(dir string, rankMap rankFileData, allCallsData []counts.CommDataT, callMap map[int][]Location, callsRanksMap map[int]map[int]int) error {
	// Find all the files that have location data
	/* GV HERE
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	*/

	/*
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
	*/

	// Create a list of all the communicators and all the alltoallv calls on these

	globalHeatMap := make(map[int]int) // The comm world rank is the key, the value amount of data sent to it
	bar := progress.NewBar(len(allCallsData), "Rank location files")
	defer progress.EndBar(bar)

	for leadRank, v := range allCallsData {
		bar.Increment(1)
		for callID, cd := range v.CallData {
			callHeatMap, err := createMapFromCounts(cd.SendData.Counts, cd.SendData.Statistics.DatatypeSize, callsRanksMap[callID], globalHeatMap)
			if err != nil {
				return err
			}

			// Save the call-based heat map
			callHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d.call%d.txt", callHeatMapPrefix, leadRank, callID))
			err = saveHeatMap(callHeatMap, callHeatMapFilePath)
			if err != nil {
				return err
			}
		}
	}
	/*
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

							err = createMapFromCounts(heatMap, commHeatMap)
							if err != nil {
								return err
							}

				// Save the communicator-based heat map
				commHeatMapFilePath := filepath.Join(dir, commHeatMapPrefix+".rank"+strconv.Itoa(rankID)+".txt")
				err = saveHeatMap(commHeatMap, commHeatMapFilePath)
				if err != nil {
					return err
				}
			}
	*/

	// Save the heat map for the entire execution
	globalHeatMapFilePath := filepath.Join(dir, globalHeatMapPrefix)
	err := saveHeatMap(globalHeatMap, globalHeatMapFilePath)
	if err != nil {
		return err
	}

	return nil
}

// Create is the main function to create heat maps. The id identifies what type of maps
// need to be created.
func Create(id int, dir string, allCallsData []counts.CommDataT) error {
	switch id {
	case Heat:
		rankMap, callsMap, callsRanksMap, err := prepareRanksMap(dir)
		if err != nil {
			return err
		}
		return createHeatMap(dir, rankMap, allCallsData, callsMap, callsRanksMap)
	}

	return fmt.Errorf("unknown map type: %d", id)
}

// getCommMap lloks at the list of files generated during profiling. The file is created by the
// lead rank of the communicator (rank 0 on the communicator) and stores the following data for
// all ranks on the communicator: rank on communicator, rank on COMMWORLD, hostname and PID.
// The function extracts the alltoallv call associated to a specific file.
// The function returns a map where: the key is the lead rank; the value is the structure storing the call ID and file path.
func getCommMap(dir string) (map[int][]locationFileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	commMap := make(map[int][]locationFileInfo)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), locationFilePrefix+"rank") {
			str := strings.TrimLeft(file.Name(), locationFilePrefix+"rank")
			str = strings.TrimRight(str, ".md")
			tokens := strings.Split(str, "_call")
			if len(tokens) != 2 {
				continue
			}
			rank, err := strconv.Atoi(tokens[0])
			if err != nil {
				return nil, err
			}
			call, err := strconv.Atoi(tokens[0])
			if err != nil {
				return nil, err
			}

			l := locationFileInfo{
				callID: call,
				file:   filepath.Join(dir, file.Name()),
			}
			if _, ok := commMap[rank]; ok {
				cf := commMap[rank]
				cf = append(cf, l)
				commMap[rank] = cf
			} else {
				commMap[rank] = []locationFileInfo{l}
			}
		}
	}
	return commMap, nil
}

// analyzeCommFiles avoids parsing location files with same content for a given lead rank, i.e., sub-communicator(s).
// Many of the files may have the same content, we go through them to get to the minimum amount of information required
// so we do not have to parse all the files.
func analyzeCommFiles(leadRank int, commInfo []locationFileInfo) (map[string]rankMapInfo, error) {
	uniqueRankMap := make(map[string]rankMapInfo)
	for _, info := range commInfo {
		h, err := hash.File(info.file)
		if err != nil {
			return nil, err
		}
		if _, ok := uniqueRankMap[h]; ok {
			data := uniqueRankMap[h]
			data.calls = append(data.calls, info.callID)
		} else {
			newData := rankMapInfo{
				calls: []int{info.callID},
				file:  info.file,
			}
			uniqueRankMap[h] = newData
		}
	}
	return uniqueRankMap, nil
}

func saveProcessedLocationData(dir string, leadRank int, info map[int]int) error {
	targetFile := filepath.Join(dir, commRankMapPrefix+strconv.Itoa(leadRank)+".txt")
	fd, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	for commRank, worldRank := range info {
		_, err := fd.WriteString(fmt.Sprintf("COMM rank: %d; WORLD rank: %d\n", commRank, worldRank))
		if err != nil {
			return err
		}
	}

	return nil
}

// getRankMapFromFile parses the unique location files that could be found and returns
// both a rank-based map (map of comm rank/worldcomm rank), as well as a call map (map
// where the key if the call ID and the value a slice of Location)
func getRankMapFromFile(info map[string]rankMapInfo, hm rankFileData, callMap map[int][]Location) (map[int]int, map[int][]Location, error) {
	m := make(map[int]int) // The key is the rank on the communicator; the value is the rank on COMMWORLD

	for _, data := range info {
		locations, err := parseLocationFile(data.file)
		if err != nil {
			return nil, nil, err
		}

		for _, c := range data.calls {
			if _, ok := callMap[c]; ok {
				fmt.Printf("[WARN] Location data for call %d already present", c)
			}
			callMap[c] = locations
		}

		for _, l := range locations {
			m[l.CommRank] = l.CommWorldRank
			if _, ok := hm.rankMap[l.CommWorldRank]; !ok {
				// We do not track the host information for that rank yet
				hm.rankMap[l.CommWorldRank] = l.Hostname
				rankList := hm.hostMap[l.Hostname]
				rankList = append(rankList, l.CommWorldRank)
				hm.hostMap[l.Hostname] = rankList
			}
		}
	}

	return m, callMap, nil
}

func createRankFile(dir string, hm rankFileData) error {
	rankFilePath := filepath.Join(dir, rankFilename)
	fd, err := os.OpenFile(rankFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = fd.WriteString(fmt.Sprintf("Total of %d nodes\n", len(hm.hostMap)))
	if err != nil {
		return err
	}

	for host, rankList := range hm.hostMap {
		sort.Ints(rankList)
		_, err = fd.WriteString(fmt.Sprintf("Host %s - %d ranks: %s\n", host, len(rankList), notation.CompressIntArray(rankList)))
		if err != nil {
			return err
		}
	}

	return nil
}

func prepareRanksMap(dir string) (rankFileData, map[int][]Location, map[int]map[int]int, error) {
	callMap := make(map[int][]Location)
	var callsRanksMap = map[int]map[int]int{}

	// This is to track the files for a specific communicator
	hm := rankFileData{
		hostMap: make(map[string][]int),
		rankMap: make(map[int]string),
	}

	commMap, err := getCommMap(dir)
	if err != nil {
		return hm, nil, nil, err
	}

	// We have a list of all the location files as well as the communicator lead rank and the call ID; based on comm rank lead.
	// Now we will parse that data: first we pre-process the list of files, identifying identical files; then we parse the unique files and gather the data
	// In other words, for each unique rank lead previously identified, we parse all the associated location files.
	for leadRank, commInfo := range commMap {
		// Curate the data to avoid parsing identical location files.
		// The returned data includes the list of all the associated call IDs
		commRankMap, err := analyzeCommFiles(leadRank, commInfo)
		if err != nil {
			return hm, nil, nil, err
		}

		// Now we have a curated data, i.e., unique location files that represent location data for all the alltoallv calls performed on the communicator(s) led by leadRank.
		// So we can efficiently parse the location files.
		// We also slowly build the rank file's data will going through the files.
		m, _ /*callMap*/, err := getRankMapFromFile(commRankMap, hm, callMap)
		if err != nil {
			return hm, nil, nil, err
		}

		// We link rank mapping to actual calls so we can use it later
		for _, rankLocationInfo := range commRankMap {
			for _, c := range rankLocationInfo.calls {
				callsRanksMap[c] = make(map[int]int)
				callsRanksMap[c] = m
			}
		}

		err = saveProcessedLocationData(dir, leadRank, m)
		if err != nil {
			return hm, nil, nil, err
		}
	}

	err = createRankFile(dir, hm)
	if err != nil {
		return hm, nil, nil, err
	}

	return hm, callMap, callsRanksMap, nil
}
