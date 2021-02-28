//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package maps

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
)

func getRanksMapFromLocations(locations []*location.RankLocation) map[int]int {
	ranksMap := make(map[int]int)
	for _, l := range locations {
		ranksMap[l.CommRank] = l.CommWorldRank
	}
	return ranksMap
}

func getRankFileDataFromLocations(locations []*location.RankLocation) location.RankFileData {
	var data location.RankFileData
	data.RankMap = make(map[int]string)
	data.HostMap = make(map[string][]int)

	for _, l := range locations {
		if _, ok := data.RankMap[l.CommWorldRank]; !ok {
			data.RankMap[l.CommWorldRank] = l.Hostname
			data.HostMap[l.Hostname] = append(data.HostMap[l.Hostname], l.CommWorldRank)
		}
	}
	return data
}

func TestCreateMapFromCounts(t *testing.T) {
	tests := []struct {
		name                string
		datatypeSize        int
		commSize            int
		countsStr           []string
		countsInt           map[int][]int
		locations           []string
		expectedCallHeatMap []int
		expectedHostHeatMap map[string]int
	}{
		{
			name:         "oneComm",
			datatypeSize: 1,
			commSize:     2,
			countsStr:    []string{"Rank(s) 0: 1 2", "Rank(s) 1: 3 4"},
			countsInt: map[int][]int{
				0: {1, 2},
				1: {3, 4},
			},
			locations:           []string{"Communicator ID: 0\n", "Calls: 0-1\n", "COMM_WORLD ranks: 0-1\n", "PIDs: 1041208-1041209\n", "Hostnames:\n", "\tRank 0: node1\n", "\tRank 1: node2"},
			expectedCallHeatMap: []int{3, 7}, // Rank 0 sends a total of 3 bytes; rank 1 a total of 7 bytes
			expectedHostHeatMap: map[string]int{
				"node1": 3, // 3 bytes are sent from node1
				"node2": 7, // 7 bytes are sent from node2
			},
		},
	}

	for _, tt := range tests {
		globalHeatMap := make(map[int]int)
		locationData, err := location.GetLocationDataFromStrings(tt.locations, 0)
		if err != nil {
			t.Fatalf("getLocationFromString() failed: %s", err)
		}

		ranksMap := getRanksMapFromLocations(locationData.RankLocations)
		rankFileData := getRankFileDataFromLocations(locationData.RankLocations)
		var data counts.Data
		data.RawCounts = tt.countsStr
		data.Counts = make(map[int]map[int][]int)
		data.Counts[0] = tt.countsInt
		data.CountsMetadata.DatatypeSize = tt.datatypeSize
		data.CountsMetadata.CallIDs = []int{0}
		data.CountsMetadata.NumRanks = tt.commSize
		rankNumCallsMap := make(map[int]int)
		var hostHeatMap map[string]int
		var callHeatMap map[int]int
		callHeatMap, hostHeatMap, err = createCallsMapsFromCounts(0, data, tt.datatypeSize, &rankFileData, ranksMap, hostHeatMap, globalHeatMap, rankNumCallsMap)
		if err != nil {
			t.Fatalf("createMapFromCounts() failed: %s", err)
		}

		if len(rankNumCallsMap) != tt.commSize {
			t.Fatalf("number of calls per rank is invalid: %d instead of %d", len(rankNumCallsMap), tt.commSize)
		}
		for rank, numCalls := range rankNumCallsMap {
			// fixme: do not hardcode this, in the context of multicommunicators, that would not be right
			if numCalls != 1 {
				t.Fatalf("number of calls for rank %d is %d instead of 1", rank, numCalls)
			}
		}

		if len(hostHeatMap) != len(tt.expectedHostHeatMap) {
			t.Fatalf("host heat map has an invalid size: %d instead of %d", len(hostHeatMap), len(tt.expectedHostHeatMap))
		}

		if len(callHeatMap) != len(tt.expectedCallHeatMap) {
			t.Fatalf("call heat map has an invalid size: %d instead of %d", len(callHeatMap), len(tt.expectedCallHeatMap))
		}
		if len(globalHeatMap) != len(tt.expectedCallHeatMap) {
			t.Fatalf("global heat map has an invalid size: %d instead of %d", len(globalHeatMap), len(tt.expectedCallHeatMap))
		}

		for i := 0; i < len(tt.expectedCallHeatMap); i++ {
			if callHeatMap[i] != tt.expectedCallHeatMap[i] {
				t.Fatalf("Value for rank %d in call heat is invalid: %d instead of %d", i, callHeatMap[i], tt.expectedCallHeatMap[i])
			}
		}

		for i := 0; i < len(tt.expectedCallHeatMap); i++ {
			if globalHeatMap[i] != tt.expectedCallHeatMap[i] {
				t.Fatalf("Value for rank %d in call heat is invalid: %d instead of %d", i, globalHeatMap[i], tt.expectedCallHeatMap[i])
			}
		}

		for host, value := range tt.expectedHostHeatMap {
			if hostHeatMap[host] != value {
				t.Fatalf("Host heat map is invalid, value for host %s is %d instead of %d", host, hostHeatMap[host], value)
			}
		}

	}
}

func TestLoadHostMap(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(filename)

	tests := []struct {
		inputFile   string
		expectedMap map[string][]int
	}{
		{
			inputFile: filepath.Join(basedir, "testData", "set1", "input", "rankfile.txt"),
			expectedMap: map[string][]int{
				"node-031": {960, 961, 962, 963, 964, 965, 966, 967, 968, 969, 970},
				"node-002": {32},
				"node-012": {352, 353, 354},
				"node-017": {512, 513, 514, 515, 516, 517, 518, 520},
				"node-026": {800, 801, 802},
				"node-029": {900, 901, 902, 903, 904, 905, 906, 907, 908, 909, 910, 911, 912},
			},
		},
	}

	for _, tt := range tests {
		m, err := LoadHostMap(tt.inputFile)
		if err != nil {
			t.Fatalf("LoadHostMap() failed: %s", err)
		}
		if len(m) != len(tt.expectedMap) {
			t.Fatalf("LoadHostMap() returned %d ranks instead of %d", len(m), len(tt.expectedMap))
		}
		for k, v := range tt.expectedMap {
			if len(v) != len(m[k]) {
				t.Fatalf("Host %s is reported as having %d ranks instead of %d", k, len(m[k]), len(v))
			}
			for i := 0; i < len(v); i++ {
				if tt.expectedMap[k][i] != m[k][i] {
					t.Fatalf("Rank %d for host %s is reported as %d instead of %d", i, k, m[k][i], tt.expectedMap[k][i])
				}
			}
		}
	}
}
