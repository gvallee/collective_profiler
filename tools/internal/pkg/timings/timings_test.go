//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package timings

import (
	"testing"
)

func TestGetLeadRankFromFilename(t *testing.T) {
	tests := []struct {
		input          string
		expectedOutput int
	}{
		{
			input:          "a2a-timings.job0.rank0.md",
			expectedOutput: 0,
		},
		{
			input:          "a2a-timings.job4235245.rank52454.md",
			expectedOutput: 52454,
		},
		{
			input:          "late-arrivals-timings.job0.rank0.md",
			expectedOutput: 0,
		},
		{
			input:          "late-arrivals-timings.job446546531.rank4434333245.md",
			expectedOutput: 4434333245,
		},
	}

	for _, tt := range tests {
		val, err := getLeadRankFromFilename(tt.input)
		if err != nil {
			t.Fatalf("getLeadRankFromFilename() failed: %s", err)
		}
		if val != tt.expectedOutput {
			t.Fatalf("getLeadRankFromFilename() returned %d instead of %d", val, tt.expectedOutput)
		}
	}
}
