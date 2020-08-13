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

func TestConvertCompressedCallListToIntSlice(t *testing.T) {
	tests := []struct {
		input         string
		expectedArray []int
	}{
		{
			input:         "32",
			expectedArray: []int{32},
		},
		{
			input:         "32-35,40",
			expectedArray: []int{32, 33, 34, 35, 40},
		},
	}

	for _, tt := range tests {
		array, err := ConvertCompressedCallListToIntSlice(tt.input)
		if err != nil {
			t.Fatalf("ConvertCompressedCallListToIntSlice() failed: %s", err)
		}
		if len(array) != len(tt.expectedArray) {
			t.Fatalf("ConvertCompressedCallListToIntSlice() returned %d elements instead of %d", len(array), len(tt.expectedArray))
		}
		for i := 0; i < len(tt.expectedArray); i++ {
			if array[i] != tt.expectedArray[i] {
				t.Fatalf("element %d of the array is %d instead of %d", i, array[i], tt.expectedArray[i])
			}
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
		{
			input:          "32",
			expectedOutput: 1,
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

func TestIntSliceToString(t *testing.T) {
	tests := []struct {
		s              []int
		expectedResult string
	}{
		{
			s:              []int{1, 2, 3, 4, 5, 6},
			expectedResult: "1,2,3,4,5,6",
		},
	}

	for _, tt := range tests {
		str := IntSliceToString(tt.s)
		if str != tt.expectedResult {
			t.Fatalf("Test returned %s instead of %s", str, tt.expectedResult)
		}
	}
}
