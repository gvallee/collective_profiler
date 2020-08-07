//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package maps

import (
	"testing"
)

func getRanksMapFromLocations(locations []Location) map[int]int {
	ranksMap := make(map[int]int)
	for _, l := range locations {
		ranksMap[l.CommRank] = l.CommWorldRank
	}
	return ranksMap
}

func getRankFileDataFromLocations(locations []Location) RankFileData {
	var data RankFileData
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

func TestGetLocationsFromStrings(t *testing.T) {
	tests := []struct {
		input             []string
		expectedLocations []Location
	}{
		{
			input: []string{"COMMWORLD rank: 0 - COMM rank: 0 - PID: 2 - Hostname: node1", "COMMWORLD rank: 1 - COMM rank: 1 - PID: 3 - Hostname: node2"},
			expectedLocations: []Location{
				{
					CommWorldRank: 0,
					CommRank:      0,
					PID:           2,
					Hostname:      "node1",
				},
				{
					CommWorldRank: 1,
					CommRank:      1,
					PID:           3,
					Hostname:      "node2",
				},
			},
		},
	}

	for _, tt := range tests {
		l, err := getLocationsFromStrings(tt.input)
		if err != nil {
			t.Fatalf("getLocationFromString() failed: %s", err)
		}

		if len(l) != len(tt.expectedLocations) {
			t.Fatalf("getLocationFromString() returned %d locations instead of %d", len(l), len(tt.expectedLocations))
		}

		for i := 0; i < len(l); i++ {
			if l[i].CommWorldRank != tt.expectedLocations[i].CommWorldRank {
				t.Fatalf("COMM WORLD's rank for element %d is %d instead of %d", i, l[i].CommWorldRank, tt.expectedLocations[i].CommWorldRank)
			}
			if l[i].CommRank != tt.expectedLocations[i].CommRank {
				t.Fatalf("comm rank for element %d is %d instead of %d", i, l[i].CommRank, tt.expectedLocations[i].CommRank)
			}
			if l[i].Hostname != tt.expectedLocations[i].Hostname {
				t.Fatalf("Rank location for element %d is %s instead of %s", i, l[i].Hostname, tt.expectedLocations[i].Hostname)
			}
		}
	}
}

func TestCreateMapFromCounts(t *testing.T) {
	tests := []struct {
		name                string
		datatypeSize        int
		counts              []string
		locations           []string
		expectedCallHeatMap []int
		expectedHostHeatMap map[string]int
	}{
		{
			name:                "oneComm",
			datatypeSize:        1,
			counts:              []string{"Rank(s) 0: 1 2", "Rank(s) 1: 3 4"},
			locations:           []string{"COMMWORLD rank: 0 - COMM rank: 0 - PID: 2 - Hostname: node1", "COMMWORLD rank: 1 - COMM rank: 1 - PID: 3 - Hostname: node2"},
			expectedCallHeatMap: []int{4, 6},
			expectedHostHeatMap: map[string]int{
				"node1": 4,
				"node2": 6,
			},
		},
	}

	for _, tt := range tests {
		globalHeatMap := make(map[int]int)
		l, err := getLocationsFromStrings(tt.locations)
		if err != nil {
			t.Fatalf("getLocationFromString() failed: %s", err)
		}

		ranksMap := getRanksMapFromLocations(l)
		rankFileData := getRankFileDataFromLocations(l)
		callHeatMap, hostHeatMap, err := createCallsMapsFromCounts(tt.counts, tt.datatypeSize, &rankFileData, ranksMap, globalHeatMap)
		if err != nil {
			t.Fatalf("createMapFromCounts() failed: %s", err)
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
