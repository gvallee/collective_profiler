//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package scale

const (
	DOWN = -1
	UP   = 1
)

func allZerosInts(sortedValues []int) bool {
	// If all values are 0 nothing can be done
	if len(sortedValues) >= 2 {
		if sortedValues[0] == 0 && sortedValues[len(sortedValues)-1] == 0 {
			return true
		}
	} else {
		if sortedValues[0] == 0 {
			return true
		}
	}

	return false
}

func allZerosFloat64s(sortedValues []float64) bool {
	// If all values are 0 nothing can be done
	if len(sortedValues) >= 2 && sortedValues[0] == 0 && sortedValues[len(sortedValues)-1] == 0 {
		return true
	}

	if len(sortedValues) == 1 && sortedValues[0] == 0 {
		return true
	}

	if len(sortedValues) == 0 {
		return true
	}

	return false
}
