//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package maps

import (
	"testing"
)

func TestMapCreate(t *testing.T) {
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
		t.Fatalf("Not running test %s", tt.name)
	}
}
