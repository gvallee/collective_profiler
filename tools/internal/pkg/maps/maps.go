//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package maps

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/alltoallv_profiling/tools/pkg/errors"
)

const (
	// Heat is used as an identifier to refer to heat map
	Heat = 1

	heatFilePrefix    = "heat-map."
	commRankMapPrefix = "comm-rank-map."

	// CommHeatMapPrefix is the prefix used for the file storing the communicator heat map
	CommHeatMapPrefix = "heat-map-subcomm"

	// HostHeatMapPrefix is the prefix used for the file storing the host heat map
	HostHeatMapPrefix = "hosts-heat-map.rank"

	// CallHeatMapPrefix is the prefix used for the file storing the call heat map
	CallHeatMapPrefix = "heat-map.rank"

	// GlobalHeatMapPrefix is the prefix used for the file storing the global heat map
	GlobalHeatMapPrefix = "heat-map"

	// RankFilename is the name of the generated rank file
	RankFilename = "rankfile.txt"
)

type commFiles struct {
	calls []int
	files []string
}

type rankMapInfo struct {
	calls []int
	file  string
}

// CallsDataT stores mapping of ranks, send heat map and recv heat map of all the alltoallv calls
type CallsDataT struct {
	// RanksMap for all the alltoallv calls
	RanksMap map[int]map[int]int

	// SendHeatMap for all the alltoallv calls
	SendHeatMap map[int]map[int]int

	// RecvHeatMMap for all the alltoallv calls
	RecvHeatMap map[int]map[int]int
}

func findCountFile(countsFiles []string, rankID int) string {
	for _, file := range countsFiles {
		if strings.Contains(file, ".rank"+strconv.Itoa(rankID)+".") {
			return file
		}
	}
	return ""
}

func getRankMapFromLocations(locations []location.Info) map[int]int {
	m := make(map[int]int)
	for _, l := range locations {
		m[l.CommRank] = l.CommWorldRank
	}
	return m
}

func getDataFromHeatMapFilename(filename string) (int, int, error) {
	filename = strings.TrimLeft(filename, CallHeatMapPrefix)
	tokens := strings.Split(filename, "-")
	if len(tokens) != 2 {
		return -1, -1, fmt.Errorf("unabel to parse filename: %s", filename)
	}
	leadRankStr := tokens[0]
	leadRank, err := strconv.Atoi(leadRankStr)
	if err != nil {
		return -1, -1, err
	}
	callIDStr := tokens[1]
	callIDStr = strings.TrimRight(callIDStr, "-send.call")
	callIDStr = strings.TrimRight(callIDStr, "-recv.call")
	callIDStr = strings.TrimLeft(callIDStr, ".txt")
	callID, err := strconv.Atoi(callIDStr)
	if err != nil {
		return -1, -1, err
	}

	return leadRank, callID, err
}

// LoadCallFileHeatMap loads data from a call heat map. It returns a map where the key is a collective number and the value the heat map value
func LoadCallFileHeatMap(filePath string) (map[int]int, error) {
	m := make(map[int]int)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return nil, readerErr
		}
		if readerErr != nil && readerErr == io.EOF {
			break // end of dataset
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ": ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("%s is not in a valid format", line)
		}
		rank, err := strconv.Atoi(strings.TrimLeft(tokens[0], "Rank "))
		if err != nil {
			return nil, err
		}

		size, err := strconv.Atoi(strings.TrimRight(tokens[1], " bytes"))
		if err != nil {
			return nil, err
		}
		m[rank] = size
	}

	return m, nil
}

func saveCallHeatMap(heatmap map[int]int, filepath string) error {
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

func saveHostHeatMap(heatMap map[string]int, filepath string) error {
	fd, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()
	for key, value := range heatMap {
		_, err := fd.WriteString(fmt.Sprintf("Host %s: %d bytes\n", key, value))
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
	countsHeader, callCounts, profilerErr := counts.LookupCallFromFile(reader, callID)
	if !profilerErr.Is(errors.ErrNone) {
		return -1, nil, profilerErr.GetInternal()
	}

	return countsHeader.DatatypeSize, callCounts, nil
}

func createCallsMapsFromCounts(callCounts counts.Data, datatypeSize int, rankMap *location.RankFileData, ranksMap map[int]int, globalHeatMap map[int]int, rankNumCallsMap map[int]int) (map[int]int, map[string]int, error) {
	// Now we can have send counts for all the ranks on the communicator as well as th translation comm rank to COMMWORLD rank
	// We can populate the heat map
	callHeatMap := make(map[int]int)
	callHostHeatMap := make(map[string]int)

	for _, counts := range callCounts.RawCounts {
		counts = strings.TrimRight(counts, "\n")

		// We need to clean up the string callCounts since it also has the list of sending ranks,
		// which we do not care about here
		tokens := strings.Split(counts, ": ")
		if len(tokens) != 2 {
			return nil, nil, fmt.Errorf("wrong counts format: %s", counts)
		}
		counts = tokens[1]
		ranks, err := notation.ConvertCompressedCallListToIntSlice(strings.TrimLeft(tokens[0], "Rank(s) "))
		if err != nil {
			return nil, nil, err
		}

		for _, curRank := range ranks {
			tokens = strings.Split(counts, " ")
			worldRank := ranksMap[curRank]
			curRankHost := rankMap.RankMap[curRank]
			countSum := 0
			for _, countStr := range tokens {
				if countStr == "" {
					continue
				}
				count, err := strconv.Atoi(countStr)
				if err != nil {
					return nil, nil, err
				}
				countSum += count
				curRank++
			}
			callHostHeatMap[curRankHost] += countSum * datatypeSize
			globalHeatMap[worldRank] += countSum * datatypeSize
			callHeatMap[worldRank] += countSum * datatypeSize
			rankNumCallsMap[worldRank] += len(callCounts.CountsMetadata.CallIDs)
		}
	}
	return callHeatMap, callHostHeatMap, nil
}

func createHeatMap(dir string, leadRank int, rankMap *location.RankFileData, allCallsData map[int]*counts.CallData, callsData *CallsDataT, globalSendHeatMap map[int]int, globalRecvHeatMap map[int]int, rankNumCallsMap map[int]int) error {
	bar := progress.NewBar(len(allCallsData), "Gathering map data")
	defer progress.EndBar(bar)

	for callID, cd := range allCallsData {
		bar.Increment(1)

		var err error
		var hostSendHeatMap map[string]int
		callsData.SendHeatMap[callID], hostSendHeatMap, err = createCallsMapsFromCounts(cd.SendData, cd.SendData.Statistics.DatatypeSize, rankMap, callsData.RanksMap[callID], globalSendHeatMap, rankNumCallsMap)
		if err != nil {
			return err
		}

		var hostRecvHeatMap map[string]int
		callsData.RecvHeatMap[callID], hostRecvHeatMap, err = createCallsMapsFromCounts(cd.RecvData, cd.RecvData.Statistics.DatatypeSize, rankMap, callsData.RanksMap[callID], globalRecvHeatMap, rankNumCallsMap)
		if err != nil {
			return err
		}

		// Save the call-based heat maps
		callSendHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-send.call%d.txt", CallHeatMapPrefix, leadRank, callID))
		err = saveCallHeatMap(callsData.SendHeatMap[callID], callSendHeatMapFilePath)
		if err != nil {
			return err
		}
		hostSendHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-send.call%d.txt", HostHeatMapPrefix, leadRank, callID))
		err = saveHostHeatMap(hostSendHeatMap, hostSendHeatMapFilePath)
		if err != nil {
			return err
		}

		callRecvHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-recv.call%d.txt", CallHeatMapPrefix, leadRank, callID))
		err = saveCallHeatMap(callsData.RecvHeatMap[callID], callRecvHeatMapFilePath)
		if err != nil {
			return err
		}
		hostRecvHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s%d-recv.call%d.txt", HostHeatMapPrefix, leadRank, callID))
		err = saveHostHeatMap(hostRecvHeatMap, hostRecvHeatMapFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func commCreate(dir string, leadRank int, allCallsData map[int]*counts.CallData, globalSendHeatMap map[int]int, globalRecvHeatMap map[int]int, rankNumCallsMap map[int]int) (location.RankFileData, CallsDataT, error) {
	commMaps := CallsDataT{
		SendHeatMap: map[int]map[int]int{},
		RecvHeatMap: map[int]map[int]int{},
	}
	var rankFileData location.RankFileData
	var err error
	rankFileData, _, commMaps.RanksMap, err = prepareRanksMap(dir)
	if err != nil {
		return rankFileData, commMaps, err
	}

	err = createHeatMap(dir, leadRank, &rankFileData, allCallsData, &commMaps, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap)
	if err != nil {
		return rankFileData, commMaps, err
	}

	// Save the heat maps for the entire execution
	globalSendHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-send.txt")
	err = saveCallHeatMap(globalSendHeatMap, globalSendHeatMapFilePath)
	if err != nil {
		return rankFileData, commMaps, err
	}

	globalRecvHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-recv.txt")
	err = saveCallHeatMap(globalRecvHeatMap, globalRecvHeatMapFilePath)
	if err != nil {
		return rankFileData, commMaps, err
	}

	return rankFileData, commMaps, nil
}

// Create is the main function to create heat maps. The id identifies what type of maps
// need to be created.
func Create(id int, dir string, allCallsData []counts.CommDataT) (map[int]location.RankFileData, map[int]CallsDataT, map[int]int, map[int]int, map[int]int, error) {
	switch id {
	case Heat:
		var err error

		rankNumCallsMap := make(map[int]int)
		globalCallsData := make(map[int]CallsDataT)
		// fixme: RankFileData is supposed to be static and dealing with ranks on comm world, no need to track per lead rank
		globalCommRankFileData := make(map[int]location.RankFileData)
		globalSendHeatMap := make(map[int]int) // The comm world rank is the key, the value amount of data sent to it
		globalRecvHeatMap := make(map[int]int)

		for _, commData := range allCallsData {
			globalCommRankFileData[commData.LeadRank], globalCallsData[commData.LeadRank], err = commCreate(dir, commData.LeadRank, commData.CallData, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}

		// Save the heat maps for the entire execution
		globalSendHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-send.txt")
		err = saveCallHeatMap(globalSendHeatMap, globalSendHeatMapFilePath)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		globalRecvHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-recv.txt")
		err = saveCallHeatMap(globalRecvHeatMap, globalRecvHeatMapFilePath)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		return globalCommRankFileData, globalCallsData, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap, nil
	}

	return nil, nil, nil, nil, nil, fmt.Errorf("unknown map type: %d", id)
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
func getRankMapFromFile(info map[string]*rankMapInfo, hm location.RankFileData, callMap map[int][]location.Info) (map[int]int, map[int][]location.Info, error) {
	m := make(map[int]int) // The key is the rank on the communicator; the value is the rank on COMMWORLD

	/*
		for _, data := range info {
			locations, err := location.ParseLocationFile(data.file)
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
				if _, ok := hm.RankMap[l.CommWorldRank]; !ok {
					// We do not track the host information for that rank yet
					hm.RankMap[l.CommWorldRank] = l.Hostname
					rankList := hm.HostMap[l.Hostname]
					rankList = append(rankList, l.CommWorldRank)
					hm.HostMap[l.Hostname] = rankList
				}
			}
		}
	*/

	return m, callMap, nil
}

func createRankFile(dir string, hm location.RankFileData) error {
	rankFilePath := filepath.Join(dir, RankFilename)
	fd, err := os.OpenFile(rankFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = fd.WriteString(fmt.Sprintf("Total of %d nodes\n", len(hm.HostMap)))
	if err != nil {
		return err
	}

	for host, rankList := range hm.HostMap {
		sort.Ints(rankList)
		_, err = fd.WriteString(fmt.Sprintf("Host %s - %d ranks: %s\n", host, len(rankList), notation.CompressIntArray(rankList)))
		if err != nil {
			return err
		}
	}

	return nil
}

func prepareRanksMap(dir string) (location.RankFileData, map[int][]location.Info, map[int]map[int]int, error) {
	callMap := make(map[int][]location.Info)
	var callsRanksMap = map[int]map[int]int{}

	// This is to track the files for a specific communicator
	hm := location.RankFileData{
		HostMap: make(map[string][]int),
		RankMap: make(map[int]string),
	}

	/*
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
			m, _, err := getRankMapFromFile(commRankMap, hm, callMap)
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
	*/

	err := createRankFile(dir, hm)
	if err != nil {
		return hm, nil, nil, err
	}

	return hm, callMap, callsRanksMap, nil
}

// CreateAvgMaps uses the send and receive counts to create an average heat map of the data that is sent/received
func CreateAvgMaps(totalNumCalls int, globalSendHeatMap map[int]int, globalRecvHeatMap map[int]int) (map[int]int, map[int]int) {
	avgSendMap := make(map[int]int)
	avgRecvMap := make(map[int]int)

	for key, val := range globalSendHeatMap {
		avgSendMap[key] = val / totalNumCalls
	}

	for key, val := range globalRecvHeatMap {
		avgRecvMap[key] = val / totalNumCalls
	}

	return avgSendMap, avgRecvMap
}

// LoadHostMap parses a host map and return a map where the key is the host and the value a list of rank (on COMM_WORLD) on that host
func LoadHostMap(filePath string) (map[string][]int, error) {
	m := make(map[string][]int)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// First line is metadata, we skip it
	_, readerErr := reader.ReadString('\n')
	if readerErr != nil {
		return nil, readerErr
	}

	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return nil, readerErr
		}
		if readerErr != nil && readerErr == io.EOF {
			break // end of dataset
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			continue
		}

		tokens := strings.Split(line, "ranks: ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("%s is not of valid format", line)
		}
		hostname := strings.TrimLeft(tokens[0], "Host ")
		tokens2 := strings.Split(hostname, " - ")
		if len(tokens2) != 2 {
			return nil, fmt.Errorf("%s is of invalid format", line)
		}
		hostname = tokens2[0]
		m[hostname], err = notation.ConvertCompressedCallListToIntSlice(tokens[1])
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}
