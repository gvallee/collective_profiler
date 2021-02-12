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
		expectedOutput [3]int
	}{
		{
			input:          "alltoallv_execution_times.rank0_comm0_job0.md",
			expectedOutput: [3]int{0, 0, 0},
		},
		{
			input:          "alltoallv_execution_times.rank2453463_comm52542_job542.md",
			expectedOutput: [3]int{2453463, 52542, 542},
		},
		{
			input:          "alltoall_execution_times.rank0_comm0_job0.md",
			expectedOutput: [3]int{0, 0, 0},
		},
		{
			input:          "alltoall_execution_times.rank3587_comm2452_job5384.md",
			expectedOutput: [3]int{3587, 2452, 5384},
		},
		{
			input:          "alltoallv_late_arrival_times.rank0_comm0_job0.md",
			expectedOutput: [3]int{0, 0, 0},
		},
		{
			input:          "alltoallv_late_arrival_times.rank1234_comm5423_job57645.md",
			expectedOutput: [3]int{1234, 5423, 57645},
		},
		{
			input:          "alltoall_late_arrival_times.rank0_comm0_job0.md",
			expectedOutput: [3]int{0, 0, 0},
		},
		{
			input:          "alltoall_late_arrival_times.rank1234_comm5423_job57645.md",
			expectedOutput: [3]int{1234, 5423, 57645},
		},
	}

	for _, tt := range tests {
		leadRank, commID, jobID, err := getMetadataFromFilename(tt.input)
		if err != nil {
			t.Fatalf("getMetadataFromFilename() failed: %s", err)
		}
		if leadRank != tt.expectedOutput[0] {
			t.Fatalf("getMetadataFromFilename() returned %d instead of %d", leadRank, tt.expectedOutput[0])
		}
		if commID != tt.expectedOutput[1] {
			t.Fatalf("getMetadataFromFilename() returned %d instead of %d", commID, tt.expectedOutput[1])
		}
		if jobID != tt.expectedOutput[2] {
			t.Fatalf("getMetadataFromFilename() returned %d instead of %d", jobID, tt.expectedOutput[2])
		}
	}
}
