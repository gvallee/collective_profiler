//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package scale

import (
	"sort"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/unit"
)

func mapIntsScaleDown(unitType int, unitScale int, values map[int]int) (int, int, map[int]int) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale - 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = mapIntsCompute(DOWN, values)

	return unitType, newUnitScale, values
}

func mapIntsScaleUp(unitType int, unitScale int, values map[int]int) (int, int, map[int]int) {
	if unitScale == -1 {
		// Unit not recognized, nothing we can do
		return unitType, unitScale, values
	}

	newUnitScale := unitScale + 1
	if !unit.IsValidScale(unitType, newUnitScale) {
		// nothing we can do
		return unitType, unitScale, values
	}

	values = mapIntsCompute(UP, values)

	return unitType, newUnitScale, values
}

func mapIntsCompute(op int, values map[int]int) map[int]int {
	newMap := make(map[int]int)
	switch op {
	case DOWN:
		for key, val := range values {
			newMap[key] = val * 1000
		}
	case UP:
		for key, val := range values {
			newMap[key] = val / 1000
		}
	}
	return newMap
}

func MapInts(unitID string, m map[int]int) (string, map[int]int) {
	var sortedValues []int

	for _, v := range m {
		sortedValues = append(sortedValues, v)
	}
	sort.Ints(sortedValues)

	if sortedValues[0] >= 1000 {
		// We scale up the value if possible

		// Translate the human reading unit into something we can inteprete
		unitType, unitScale := unit.FromString(unitID)

		unitType, unitScale, newMap := mapIntsScaleUp(unitType, unitScale, m)
		newUnitID := unit.ToString(unitType, unitScale)
		return MapInts(newUnitID, newMap)
	}

	// Nothing to do, just return the same
	return unitID, m

}
