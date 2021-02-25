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

func float64sScaleDown(unitType int, unitScale int, values []float64) (int, int, []float64) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale - 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = float64sCompute(DOWN, values)

	return unitType, newUnitScale, values
}

func float64sScaleUp(unitType int, unitScale int, values []float64) (int, int, []float64) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale + 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = float64sCompute(UP, values)

	return unitType, newUnitScale, values
}

func float64sCompute(op int, values []float64) []float64 {
	var newValues []float64
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

// Float64s scales an array of float64
func Float64s(unitID string, values []float64) (string, []float64, error) {
	var sortedValues []float64

	if len(values) == 0 {
		return "", nil, fmt.Errorf("map is empty")
	}

	// Copy and sort the values to figure out what can be done
	for _, v := range values {
		sortedValues = append(sortedValues, v)
	}
	sort.Float64s(sortedValues)

	// If all values are 0 nothing can be done
	if allZerosFloat64s(sortedValues) {
		return unitID, values, nil
	}

	if sortedValues[0] >= 0 && sortedValues[len(values)-1] <= 1 {
		// We scale down all the values if possible

		// Translate the human reading unit into something we can inteprete
		unitType, unitScale := unit.FromString(unitID)

		unitType, unitScale, newValues := float64sScaleDown(unitType, unitScale, values)
		newUnitID := unit.ToString(unitType, unitScale)
		if unit.IsMin(unitType, unitScale) {
			return newUnitID, newValues, nil
		}

		return Float64s(newUnitID, newValues)
	}

	if sortedValues[0] >= 1000 {
		// We scale up the value if possible

		// Translate the human reading unit into something we can inteprete
		unitType, unitScale := unit.FromString(unitID)
		unitType, unitScale, newValues := float64sScaleUp(unitType, unitScale, values)
		newUnitID := unit.ToString(unitType, unitScale)
		if unit.IsMax(unitType, unitScale) {
			return newUnitID, newValues, nil
		}

		return Float64s(newUnitID, newValues)
	}

	// Nothing to do, just return the same
	return unitID, values, nil
}
