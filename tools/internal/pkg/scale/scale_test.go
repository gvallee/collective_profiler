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
		{
			unit:           "MB/s",
			values:         []int{0, 0, 0, 0},
			expectedUnit:   "MB/s",
			expectedValues: []int{0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues, err := Ints(tt.unit, tt.values)
		if err != nil {
			t.Fatalf("Ints() failed: %s", err)
		}
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
		{
			unit:           "MB/s",
			values:         []float64{0, 0, 0, 0},
			expectedUnit:   "MB/s",
			expectedValues: []float64{0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues, err := Float64s(tt.unit, tt.values)
		if err != nil {
			t.Fatalf("Float64s() failed: %s", err)
		}
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

func TestMapInts(t *testing.T) {
	tests := []struct {
		unit           string
		values         map[int]int
		expectedUnit   string
		expectedValues map[int]int
	}{
		{
			unit:           "B",
			values:         map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 13, 7: 24, 8: 664, 9: 534, 10: 23},
			expectedUnit:   "B",
			expectedValues: map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 13, 7: 24, 8: 664, 9: 534, 10: 23},
		},
		{
			unit:           "MB",
			values:         map[int]int{0: 1, 1: 2, 2: 3, 3: 4, 4: 43, 5: 54, 6: 56, 7: 65, 8: 6, 10: 5},
			expectedUnit:   "MB",
			expectedValues: map[int]int{0: 1, 1: 2, 2: 3, 3: 4, 4: 43, 5: 54, 6: 56, 7: 65, 8: 6, 10: 5},
		},
		{
			unit:           "B",
			values:         map[int]int{1: 1000, 3: 1100, 4: 10001, 5: 10002, 6: 22222, 8: 2222, 10: 244242},
			expectedUnit:   "KB",
			expectedValues: map[int]int{1: 1, 3: 1, 4: 10, 5: 10, 6: 22, 8: 2, 10: 244},
		},
		{
			unit:           "MB/s",
			values:         map[int]int{0: 0, 1: 0, 2: 0, 3: 0},
			expectedUnit:   "MB/s",
			expectedValues: map[int]int{0: 0, 1: 0, 2: 0, 3: 0},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues, err := MapInts(tt.unit, tt.values)
		if err != nil {
			t.Fatalf("MapInts failed: %s", err)
		}
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

func TestMapFloat64s(t *testing.T) {
	tests := []struct {
		unit           string
		values         map[int]float64
		expectedUnit   string
		expectedValues map[int]float64
	}{
		{
			unit:           "seconds",
			values:         map[int]float64{2: 0, 0: 1, 3: 2, 4: 3, 5: 4, 6: 5, 7: 13, 8: 24, 9: 664, 10: 534, 12: 23},
			expectedUnit:   "seconds",
			expectedValues: map[int]float64{2: 0, 0: 1, 3: 2, 4: 3, 5: 4, 6: 5, 7: 13, 8: 24, 9: 664, 10: 534, 12: 23},
		},
		{
			unit:           "seconds",
			values:         map[int]float64{1: 1, 2: 2, 3: 3, 4: 4, 6: 43, 5: 54, 8: 56, 9: 65, 10: 6, 12: 5},
			expectedUnit:   "seconds",
			expectedValues: map[int]float64{1: 1, 2: 2, 3: 3, 4: 4, 6: 43, 5: 54, 8: 56, 9: 65, 10: 6, 12: 5},
		},
		{
			unit:           "nanoseconds",
			values:         map[int]float64{1: 1000, 2: 1100, 3: 10001, 4: 10002, 5: 22222, 6: 2222, 7: 244242},
			expectedUnit:   "microseconds",
			expectedValues: map[int]float64{1: 1, 2: 1.1, 3: 10.001, 4: 10.002, 5: 22.222, 6: 2.222, 7: 244.242},
		},
		{
			unit:           "milliseconds",
			values:         map[int]float64{1: 0.1, 2: 0.2, 3: 0.01, 4: 0.002, 6: 0.3, 5: 0.11},
			expectedUnit:   "microseconds",
			expectedValues: map[int]float64{1: 100, 2: 200, 3: 10, 4: 2, 6: 300, 5: 110},
		},
		{
			unit:           "MB/s",
			values:         map[int]float64{0: 0.1, 2: 0.2, 1: 0.01, 3: 0.002, 4: 0.3, 5: 0.11},
			expectedUnit:   "KB/s",
			expectedValues: map[int]float64{0: 100, 2: 200, 1: 10, 3: 2, 4: 300, 5: 110},
		},
		{
			unit:           "GB/s",
			values:         map[int]float64{0: 1000, 1: 1100, 2: 10001, 3: 10002, 4: 22222, 5: 2222, 10: 244242},
			expectedUnit:   "TB/s",
			expectedValues: map[int]float64{0: 1, 1: 1.1, 2: 10.001, 3: 10.002, 4: 22.222, 5: 2.222, 10: 244.242},
		},
		{
			unit:           "MB/s",
			values:         map[int]float64{0: 0, 1: 0, 2: 0, 3: 0},
			expectedUnit:   "MB/s",
			expectedValues: map[int]float64{0: 0, 1: 0, 2: 0, 3: 0},
		},
	}

	for _, tt := range tests {
		scaledUnit, scaledValues, err := MapFloat64s(tt.unit, tt.values)
		if err != nil {
			t.Fatalf("MapFloat64s() failed: %s", err)
		}
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
