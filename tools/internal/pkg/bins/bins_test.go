//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package bins

import (
	"testing"
)

func TestBins(t *testing.T) {
	tests := []struct {
		name           string
		binsThresholds []int
		counts         []string
		datatypeSize   int
		binSizes       []int
	}{
		{
			name:           "3Bins3EltsInEachBin",
			binsThresholds: []int{10, 20},
			counts:         []string{"Rank(s) 0: 0 4 11 15 1 21 20 100 12"},
			datatypeSize:   1,
			binSizes:       []int{3, 3, 3},
		},
		{
			name:           "2bins1Elt5Elts",
			binsThresholds: []int{10},
			counts:         []string{"Rank(s) 0: 9 10 21 43 34 65"},
			datatypeSize:   1,
			binSizes:       []int{1, 5},
		},
	}

	for _, tt := range tests {
		bins := Create(tt.binsThresholds)
		bins, err := GetFromCounts(tt.counts, bins, 1, tt.datatypeSize)
		if err != nil {
			t.Fatalf("GetFromCounts() failed: %s", err)
		}

		for i := 0; i < len(bins); i++ {
			if bins[i].Size != tt.binSizes[i] {
				t.Fatalf("bin %d is of size %d instead of %d", i, bins[i].Size, tt.binSizes[i])
			}
		}
	}
}
