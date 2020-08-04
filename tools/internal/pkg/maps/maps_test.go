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

func TestCreateMapFromCounts(t *testing.T) {
	tests := []struct {
		name            string
		datatypeSize    int
		counts          []string
		locations       []string
		expectedHeatMap []int
	}{
		{
			name:            "oneComm",
			datatypeSize:    1,
			counts:          []string{"Rank(s) 0: 1 2", "Rank(s) 1: 3 4"},
			locations:       []string{"COMMWORLD rank: 0 - COMM rank: 0 - PID: 2 - Hostname: node1", "COMMWORLD rank: 1 - COMM rank: 1 - PID: 2 - Hostname: node2"},
			expectedHeatMap: []int{4, 6},
		},
	}

	for _, tt := range tests {
		globalHeatMap := make(map[int]int)
		l, err := getLocationsFromStrings(tt.locations)
		if err != nil {
			t.Fatalf("getLocationFromString() failed: %s", err)
		}

		ranksMap := getRanksMapFromLocations(l)
		callHeatMap, err := createMapFromCounts(tt.counts, tt.datatypeSize, ranksMap, globalHeatMap)
		if err != nil {
			t.Fatalf("createMapFromCounts() failed: %s", err)
		}

		if len(callHeatMap) != len(tt.expectedHeatMap) {
			t.Fatalf("call heat map has an invalid size: %d instead of %d", len(callHeatMap), len(tt.expectedHeatMap))
		}
		if len(globalHeatMap) != len(tt.expectedHeatMap) {
			t.Fatalf("global heat map has an invalid size: %d instead of %d", len(globalHeatMap), len(tt.expectedHeatMap))
		}

		for i := 0; i < len(tt.expectedHeatMap); i++ {
			if callHeatMap[i] != tt.expectedHeatMap[i] {
				t.Fatalf("Value for rank %d in call heat is invalid: %d instead of %d", i, callHeatMap[i], tt.expectedHeatMap[i])
			}
		}

		for i := 0; i < len(tt.expectedHeatMap); i++ {
			if globalHeatMap[i] != tt.expectedHeatMap[i] {
				t.Fatalf("Value for rank %d in call heat is invalid: %d instead of %d", i, globalHeatMap[i], tt.expectedHeatMap[i])
			}
		}

	}
}
