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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/progress"
	"github.com/gvallee/go_util/pkg/util"
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

func getRankMapFromLocations(locations []location.RankLocation) map[int]int {
	m := make(map[int]int)
	for _, l := range locations {
		m[l.CommRank] = l.CommWorldRank
	}
	return m
}

func saveGlobalHeatMap(codeBaseDir string, heatmap map[int]int, filepath string) error {
	var fd *os.File
	var err error
	if !util.FileExists(filepath) {
		fd, err = os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		err = format.WriteDataFormat(codeBaseDir, fd)
		if err != nil {
			return err
		}
	} else {
		fd, err = os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND, 0755)
		if err != nil {
			return err
		}
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

func saveCallsHeatMap(codeBaseDir string, heatmap map[int]map[int]int, filepath string) error {
	var fd *os.File
	var err error
	if !util.FileExists(filepath) {
		fd, err = os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		err = format.WriteDataFormat(codeBaseDir, fd)
		if err != nil {
			return err
		}
	} else {
		fd, err = os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND, 0755)
		if err != nil {
			return err
		}
	}
	defer fd.Close()

	var sortedCallList []int
	for k := range heatmap {
		sortedCallList = append(sortedCallList, k)
	}
	sort.Ints(sortedCallList)

	for _, callID := range sortedCallList {
		fd.WriteString(fmt.Sprintf("# Call %d:\n", callID))
		var sortedRankList []int
		for k := range heatmap[callID] {
			sortedRankList = append(sortedRankList, k)
		}
		sort.Ints(sortedRankList)
		for _, rank := range sortedRankList {
			_, err := fd.WriteString(fmt.Sprintf("Rank %d: %d bytes\n", rank, heatmap[callID][rank]))
			if err != nil {
				return err
			}
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func saveHostHeatMap(codeBaseDir string, heatMap map[string]int, filepath string) error {
	fd, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	err = format.WriteDataFormat(codeBaseDir, fd)
	if err != nil {
		return err
	}

	for key, value := range heatMap {
		_, err := fd.WriteString(fmt.Sprintf("Host %s: %d bytes\n", key, value))
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadCallsFileHeatMap parses a heat map file and return the content
func LoadCallsFileHeatMap(codeBaseDir string, path string) (map[int]map[int]int, error) {
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
		return nil, fmt.Errorf("unable to parse format version from %s: %s", path, err)
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

	data := make(map[int]map[int]int)
	callID := -1
	for {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err != nil && err == io.EOF {
			return data, nil
		}

		line = strings.TrimRight(line, "\n")
		if strings.HasPrefix(line, "# Call ") {
			line = strings.TrimLeft(line, "# Call ")
			line = strings.TrimRight(line, ":")
			callID, err = strconv.Atoi(line)
			if err != nil {
				return nil, err
			}
			data[callID] = make(map[int]int)
		}

		if strings.HasPrefix(line, "Rank ") {
			line = strings.TrimLeft(line, "Rank ")
			line = strings.TrimRight(line, " bytes")
			tokens := strings.Split(line, ": ")
			if len(tokens) != 2 {
				return nil, fmt.Errorf("%s is not a valid format", line)
			}
			rank, err := strconv.Atoi(tokens[0])
			if err != nil {
				return nil, err
			}
			size, err := strconv.Atoi(tokens[1])
			if err != nil {
				return nil, err
			}
			data[callID][rank] = size
		}
	}
}

func createCallsMapsFromCounts(callID int, callCounts counts.Data, datatypeSize int, rankMap *location.RankFileData, ranksMap map[int]int, hostHeatMap map[string]int, globalHeatMap map[int]int, rankNumCallsMap map[int]int) (map[int]int, map[string]int, error) {
	// Now we can have send counts for all the ranks on the communicator as well as th translation comm rank to COMMWORLD rank
	// We can populate the heat map
	callHeatMap := make(map[int]int)
	if hostHeatMap == nil {
		hostHeatMap = make(map[string]int)
	}

	for curRank, counts := range callCounts.Counts[callID] {
		worldRank := ranksMap[curRank]
		curRankHost := rankMap.RankMap[curRank]
		countSum := 0
		for _, count := range counts {
			countSum += count
		}
		hostHeatMap[curRankHost] += countSum * datatypeSize
		globalHeatMap[worldRank] += countSum * datatypeSize
		callHeatMap[worldRank] += countSum * datatypeSize
		rankNumCallsMap[worldRank] += len(callCounts.CountsMetadata.CallIDs)
	}

	return callHeatMap, hostHeatMap, nil
}

// GetSendCallsHeatMapFilename returns the name of the file that stores the send heat map
func GetSendCallsHeatMapFilename(dir string, collectiveName string, leadRank int) string {
	return filepath.Join(dir, fmt.Sprintf("%s_%s%d-send.md", collectiveName, CallHeatMapPrefix, leadRank))
}

// GetRecvCallsHeatMapFilename returns the name of the file that stores the recv heat map
func GetRecvCallsHeatMapFilename(dir string, collectiveName string, leadRank int) string {
	return filepath.Join(dir, fmt.Sprintf("%s_%s%d-recv.md", collectiveName, CallHeatMapPrefix, leadRank))
}

func createHeatMap(codeBaseDir string, collectiveName string, dir string, leadRank int, rankMap *location.RankFileData, allCallsData map[int]*counts.CallData, callsData *CallsDataT, globalSendHeatMap map[int]int, globalRecvHeatMap map[int]int, rankNumCallsMap map[int]int) error {
	bar := progress.NewBar(len(allCallsData), "Gathering map data")
	defer progress.EndBar(bar)

	var err error
	var hostSendHeatMap map[string]int
	var hostRecvHeatMap map[string]int
	for callID, cd := range allCallsData {
		bar.Increment(1)

		callsData.SendHeatMap[callID], hostSendHeatMap, err = createCallsMapsFromCounts(callID, cd.SendData, cd.SendData.Statistics.DatatypeSize, rankMap, callsData.RanksMap[callID], hostSendHeatMap, globalSendHeatMap, rankNumCallsMap)
		if err != nil {
			return err
		}

		callsData.RecvHeatMap[callID], hostRecvHeatMap, err = createCallsMapsFromCounts(callID, cd.RecvData, cd.RecvData.Statistics.DatatypeSize, rankMap, callsData.RanksMap[callID], hostRecvHeatMap, globalRecvHeatMap, rankNumCallsMap)
		if err != nil {
			return err
		}
	}

	fmt.Println("\nSaving maps...")
	callSendHeatMapFilePath := GetSendCallsHeatMapFilename(dir, collectiveName, leadRank)
	err = saveCallsHeatMap(codeBaseDir, callsData.SendHeatMap, callSendHeatMapFilePath)
	if err != nil {
		return err
	}

	hostSendHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s_%s%d-send.md", collectiveName, HostHeatMapPrefix, leadRank))
	err = saveHostHeatMap(codeBaseDir, hostSendHeatMap, hostSendHeatMapFilePath)
	if err != nil {
		return err
	}

	callRecvHeatMapFilePath := GetRecvCallsHeatMapFilename(dir, collectiveName, leadRank)
	err = saveCallsHeatMap(codeBaseDir, callsData.RecvHeatMap, callRecvHeatMapFilePath)
	if err != nil {
		return err
	}
	hostRecvHeatMapFilePath := filepath.Join(dir, fmt.Sprintf("%s_%s%d-recv.md", collectiveName, HostHeatMapPrefix, leadRank))
	err = saveHostHeatMap(codeBaseDir, hostRecvHeatMap, hostRecvHeatMapFilePath)
	if err != nil {
		return err
	}

	return nil
}

func commCreate(codeBaseDir string, collectiveName string, dir string, leadRank int, allCallsData map[int]*counts.CallData, globalSendHeatMap map[int]int, globalRecvHeatMap map[int]int, rankNumCallsMap map[int]int) (*location.RankFileData, CallsDataT, error) {
	commMaps := CallsDataT{
		SendHeatMap: map[int]map[int]int{},
		RecvHeatMap: map[int]map[int]int{},
	}
	var rankFileData *location.RankFileData
	var err error
	rankFileData, _, commMaps.RanksMap, err = prepareRanksMap(codeBaseDir, dir)
	if err != nil {
		return nil, commMaps, err
	}

	err = createHeatMap(codeBaseDir, collectiveName, dir, leadRank, rankFileData, allCallsData, &commMaps, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap)
	if err != nil {
		return rankFileData, commMaps, err
	}

	// Save the heat maps for the entire execution
	globalSendHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-send.md")
	err = saveGlobalHeatMap(codeBaseDir, globalSendHeatMap, globalSendHeatMapFilePath)
	if err != nil {
		return rankFileData, commMaps, err
	}

	globalRecvHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-recv.md")
	err = saveGlobalHeatMap(codeBaseDir, globalRecvHeatMap, globalRecvHeatMapFilePath)
	if err != nil {
		return rankFileData, commMaps, err
	}

	return rankFileData, commMaps, nil
}

// Create is the main function to create heat maps. The id identifies what type of maps
// need to be created.
func Create(codeBaseDir string, collectiveName string, id int, dir string, allCallsData []counts.CommDataT) (map[int]*location.RankFileData, map[int]CallsDataT, map[int]int, map[int]int, map[int]int, error) {
	switch id {
	case Heat:
		var err error

		rankNumCallsMap := make(map[int]int)
		globalCallsData := make(map[int]CallsDataT)
		// fixme: RankFileData is supposed to be static and dealing with ranks on comm world, no need to track per lead rank
		globalCommRankFileData := make(map[int]*location.RankFileData)
		globalSendHeatMap := make(map[int]int) // The comm world rank is the key, the value amount of data sent to it
		globalRecvHeatMap := make(map[int]int)

		for _, commData := range allCallsData {
			globalCommRankFileData[commData.LeadRank], globalCallsData[commData.LeadRank], err = commCreate(codeBaseDir, collectiveName, dir, commData.LeadRank, commData.CallData, globalSendHeatMap, globalRecvHeatMap, rankNumCallsMap)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}

		// Save the heat maps for the entire execution
		globalSendHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-send.md")
		err = saveGlobalHeatMap(codeBaseDir, globalSendHeatMap, globalSendHeatMapFilePath)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		globalRecvHeatMapFilePath := filepath.Join(dir, GlobalHeatMapPrefix+"-recv.md")
		err = saveGlobalHeatMap(codeBaseDir, globalRecvHeatMap, globalRecvHeatMapFilePath)
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
func getRankMapFromFile(codeBaseDir string, info map[string]*rankMapInfo, hm location.RankFileData, callMap map[int][]*location.RankLocation) (map[int]int, map[int][]*location.RankLocation, error) {
	m := make(map[int]int) // The key is the rank on the communicator; the value is the rank on COMMWORLD

	for _, data := range info {
		locations, locationsData, err := location.ParseLocationFile(codeBaseDir, data.file)
		if err != nil {
			return nil, nil, err
		}

		// Merge the call data with what we already have
		for c, callData := range locations {
			if _, ok := callMap[c]; ok {
				fmt.Printf("[WARN] Location data for call %d already present", c)
			}
			callMap[c] = callData
		}

		for _, l := range locationsData.RankLocations {
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

	return m, callMap, nil
}

func createRankFile(dir string, hm *location.RankFileData) error {
	rankFilePath := filepath.Join(dir, RankFilename)
	fd, err := os.OpenFile(rankFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

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

func prepareRanksMap(codeBaseDir string, dir string) (*location.RankFileData, map[int][]*location.RankLocation, map[int]map[int]int, error) {
	callMap := make(map[int][]*location.RankLocation)
	callsRanksMap := make(map[int]map[int]int)
	// This is to track the files for a specific communicator
	hm := new(location.RankFileData)
	hm.HostMap = make(map[string][]int)
	hm.RankMap = make(map[int]string)

	// Find all the location files
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	var locationFiles []string
	for _, file := range f {
		filename := file.Name()

		if strings.Contains(filename, location.LocationFileToken) {
			locationFiles = append(locationFiles, filepath.Join(dir, filename))
		}
	}

	// Parse each file and aggregate the results from each file.
	for _, locationFile := range locationFiles {
		callsData, locationsData, err := location.ParseLocationFile(codeBaseDir, locationFile)
		if err != nil {
			return nil, nil, nil, err
		}
		for callID := range callsData {
			if _, ok := callsRanksMap[callID]; !ok {
				// Transform the array of locations into a map
				listCallRanks := callsData[callID]
				callLocationMap := make(map[int]int)
				for _, rankLocation := range listCallRanks {
					callLocationMap[rankLocation.CommRank] = rankLocation.CommWorldRank
				}
				callsRanksMap[callID] = callLocationMap
			}

			if _, ok := callMap[callID]; !ok {
				callMap[callID] = locationsData.RankLocations
			}
		}
		for _, l := range locationsData.RankLocations {
			if _, ok := hm.HostMap[l.Hostname]; !ok {
				hm.HostMap[l.Hostname] = append(hm.HostMap[l.Hostname], l.CommWorldRank)
			}
			if _, ok := hm.RankMap[l.CommWorldRank]; !ok {
				hm.RankMap[l.CommWorldRank] = l.Hostname
			}
		}
	}

	err = createRankFile(dir, hm)
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
