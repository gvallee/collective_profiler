//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package scale

import (
	"fmt"
	"sort"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/unit"
)

func intsScaleDown(unitType int, unitScale int, values []int) (int, int, []int) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale - 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = intsCompute(DOWN, values)

	return unitType, newUnitScale, values
}

func intsScaleUp(unitType int, unitScale int, values []int) (int, int, []int) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale + 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = intsCompute(UP, values)

	return unitType, newUnitScale, values
}

func intsCompute(op int, values []int) []int {
	var newValues []int
	switch op {
	case DOWN:
		for _, val := range values {
			newValues = append(newValues, val*1000)
		}
	case UP:
		for _, val := range values {
			newValues = append(newValues, val/1000)
		}
	}
	return newValues
}

// Ints scales an array of Int
func Ints(unitID string, values []int) (string, []int, error) {
	var sortedValues []int

	if len(values) == 0 {
		return "", nil, fmt.Errorf("map is empty")
	}

	// Copy and sort the values to figure out what can be done
	for _, v := range values {
		sortedValues = append(sortedValues, v)
	}
	sort.Ints(sortedValues)

	// If all values are 0 nothing can be done
	if allZerosInts(sortedValues) {
		return unitID, values, nil
	}

	/* We deal with integers so this does not make much sense i think
	if sortedValues[0] >= 0 && sortedValues[len(values)-1] <= 1 {
		// We scale down all the values if possible

		// Translate the human reading unit into something we can inteprete
		unitType, unitScale := unit.FromString(unitID)

		unitType, unitScale, newValues := intsScaleDown(unitType, unitScale, values)
		newUnitID := unit.ToString(unitType, unitScale)
		return Ints(newUnitID, newValues)
	}
	*/

	if sortedValues[0] >= 1000 {
		// We scale up the value if possible

		// Translate the human reading unit into something we can inteprete
		unitType, unitScale := unit.FromString(unitID)

		unitType, unitScale, newValues := intsScaleUp(unitType, unitScale, values)
		newUnitID := unit.ToString(unitType, unitScale)
		if unit.IsMax(unitType, unitScale) {
			return newUnitID, newValues, nil
		}

		return Ints(newUnitID, newValues)
	}

	// Nothing to do, just return the same
	return unitID, values, nil
}
