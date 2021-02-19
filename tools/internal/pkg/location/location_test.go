//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package location

import "testing"

func TestGetLocationsFromStrings(t *testing.T) {
	tests := []struct {
		input             []string
		expectedLocations []RankLocation
	}{
		{
			input: []string{"Communicator ID: 0\n", "Calls: 0-1\n", "COMM_WORLD ranks: 0-1\n", "PIDs: 1041208-1041209\n", "Hostnames:\n", "\tRank 0: node1\n", "\tRank 1: node2\n"},
			expectedLocations: []RankLocation{
				{
					CommWorldRank: 0,
					CommRank:      0,
					PID:           1041208,
					Hostname:      "node1",
				},
				{
					CommWorldRank: 1,
					CommRank:      1,
					PID:           1041209,
					Hostname:      "node2",
				},
			},
		},
	}

	for _, tt := range tests {
		data, err := GetLocationDataFromStrings(tt.input, 0)
		if err != nil {
			t.Fatalf("getLocationFromString() failed: %s", err)
		}

		if len(data.RankLocations) != len(tt.expectedLocations) {
			t.Fatalf("getLocationFromString() returned %d locations instead of %d", len(data.RankLocations), len(tt.expectedLocations))
		}

		for i := 0; i < len(data.RankLocations); i++ {
			if data.RankLocations[i].CommWorldRank != tt.expectedLocations[i].CommWorldRank {
				t.Fatalf("COMM WORLD's rank for element %d is %d instead of %d", i, data.RankLocations[i].CommWorldRank, tt.expectedLocations[i].CommWorldRank)
			}
			if data.RankLocations[i].CommRank != tt.expectedLocations[i].CommRank {
				t.Fatalf("comm rank for element %d is %d instead of %d", i, data.RankLocations[i].CommRank, tt.expectedLocations[i].CommRank)
			}
			if data.RankLocations[i].Hostname != tt.expectedLocations[i].Hostname {
				t.Fatalf("Rank location for element %d is %s instead of %s", i, data.RankLocations[i].Hostname, tt.expectedLocations[i].Hostname)
			}
		}
	}
}
