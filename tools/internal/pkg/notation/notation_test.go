//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package notation

import (
	"testing"
)

func TestCompressIntArray(t *testing.T) {
	tests := []struct {
		array           []int
		expectedResults string
	}{
		{
			array:           []int{0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 42},
			expectedResults: "0-6,8-10,42",
		},
	}

	for _, tt := range tests {
		str := CompressIntArray(tt.array)
		if tt.expectedResults != str {
			t.Fatalf("Test failed: got %s instead of %s", str, tt.expectedResults)
		}
	}
}

func TestGetNumberOfRanksFromCompressedNotation(t *testing.T) {
	tests := []struct {
		input          string
		expectedOutput int
	}{
		{
			input:          "1, 2",
			expectedOutput: 2,
		},
		{
			input:          "1,2",
			expectedOutput: 2,
		},
		{
			input:          "1-5",
			expectedOutput: 5,
		},
		{
			input:          "0,1-5",
			expectedOutput: 6,
		},
		{
			input:          "0,1-5,6",
			expectedOutput: 7,
		},
	}
	for _, tt := range tests {
		n, err := GetNumberOfRanksFromCompressedNotation(tt.input)
		if err != nil {
			t.Fatalf("GetNumberOfRanksFromCompressedNotation() failed: %s", err)
		}
		if n != tt.expectedOutput {
			t.Fatalf("GetNumberOfRanksFromCompressedNotation() returned %d instead of %d for %s", n, tt.expectedOutput, tt.input)
		}
	}
}
