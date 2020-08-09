//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package scale

import (
	"testing"
)

func TestInts(t *testing.T) {
	tests := []struct {
		unit           string
		values         []int
		expectedUnit   string
		expectedValues []int
	}{
		{
			unit:           "B",
			values:         []int{0, 1, 2, 3, 4, 5, 13, 24, 664, 534, 23},
			expectedUnit:   "B",
			expectedValues: []int{0, 1, 2, 3, 4, 5, 13, 24, 664, 534, 23},
		},
		{
			unit:           "MB",
			values:         []int{1, 2, 3, 4, 43, 54, 56, 65, 6, 5},
			expectedUnit:   "MB",
			expectedValues: []int{1, 2, 3, 4, 43, 54, 56, 65, 6, 5},
		},
		{
			unit:           "B",
			values:         []int{1000, 1100, 10001, 10002, 22222, 2222, 244242},
			expectedUnit:   "KB",
			expectedValues: []int{1, 1, 10, 10, 22, 2, 244},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues := Ints(tt.unit, tt.values)
		if scaledUnit != tt.expectedUnit {
			t.Fatalf("Resulting unit is %s instead of %s", scaledUnit, tt.expectedUnit)
		}

		if len(scaledValues) != len(tt.expectedValues) {
			t.Fatalf("Resulting values differ: %d elements vs. %d expected", len(scaledValues), len(tt.expectedValues))
		}

		for i := 0; i < len(scaledValues); i++ {
			if scaledValues[i] != tt.expectedValues[i] {
				t.Fatalf("Value of element %d is %d instead of expected %d", i, scaledValues[i], tt.expectedValues[i])
			}
		}
	}
}

func TestFloat64s(t *testing.T) {
	tests := []struct {
		unit           string
		values         []float64
		expectedUnit   string
		expectedValues []float64
	}{
		{
			unit:           "seconds",
			values:         []float64{0, 1, 2, 3, 4, 5, 13, 24, 664, 534, 23},
			expectedUnit:   "seconds",
			expectedValues: []float64{0, 1, 2, 3, 4, 5, 13, 24, 664, 534, 23},
		},
		{
			unit:           "seconds",
			values:         []float64{1, 2, 3, 4, 43, 54, 56, 65, 6, 5},
			expectedUnit:   "seconds",
			expectedValues: []float64{1, 2, 3, 4, 43, 54, 56, 65, 6, 5},
		},
		{
			unit:           "nanoseconds",
			values:         []float64{1000, 1100, 10001, 10002, 22222, 2222, 244242},
			expectedUnit:   "microseconds",
			expectedValues: []float64{1, 1.1, 10.001, 10.002, 22.222, 2.222, 244.242},
		},
		{
			unit:           "milliseconds",
			values:         []float64{0.1, 0.2, 0.01, 0.002, 0.3, 0.11},
			expectedUnit:   "microseconds",
			expectedValues: []float64{100, 200, 10, 2, 300, 110},
		},
		{
			unit:           "MB/s",
			values:         []float64{0.1, 0.2, 0.01, 0.002, 0.3, 0.11},
			expectedUnit:   "KB/s",
			expectedValues: []float64{100, 200, 10, 2, 300, 110},
		},
		{
			unit:           "GB/s",
			values:         []float64{1000, 1100, 10001, 10002, 22222, 2222, 244242},
			expectedUnit:   "TB/s",
			expectedValues: []float64{1, 1.1, 10.001, 10.002, 22.222, 2.222, 244.242},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues := Float64s(tt.unit, tt.values)
		if scaledUnit != tt.expectedUnit {
			t.Fatalf("Resulting unit is %s instead of %s", scaledUnit, tt.expectedUnit)
		}

		if len(scaledValues) != len(tt.expectedValues) {
			t.Fatalf("Resulting values differ: %d elements vs. %d expected", len(scaledValues), len(tt.expectedValues))
		}

		for i := 0; i < len(scaledValues); i++ {
			if scaledValues[i] != tt.expectedValues[i] {
				t.Fatalf("Value of element %d is %f instead of expected %f", i, scaledValues[i], tt.expectedValues[i])
			}
		}
	}
}
