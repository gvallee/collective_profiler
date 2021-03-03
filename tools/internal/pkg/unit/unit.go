//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package unit

const (
	// DATA represents the unit used for data volume (e.g., bytes)
	DATA = iota

	// TIME represents the unit used for time mesurements (e.g., seconds)
	TIME

	// BW represents the unit used for bandwidth mesurements (e.g., B/s)
	BW
)

func getDataUnits() map[int]string {
	return map[int]string{
		0: "B",
		1: "KB",
		2: "MB",
		3: "GB",
		4: "TB",
	}
}

func getBWUnits() map[int]string {
	return map[int]string{
		0: "B/s",
		1: "KB/s",
		2: "MB/s",
		3: "GB/s",
		4: "TB/s",
	}
}

func getTimeUnits() map[int]string {
	return map[int]string{
		3: "seconds",
		2: "milliseconds",
		1: "microseconds",
		0: "nanoseconds",
	}
}

// FromString translates a type identifier that is easy to manipate to a unit dataset
func FromString(unitID string) (int, int) {
	dataTypeData := getDataUnits()
	for lvl, val := range dataTypeData {
		if val == unitID {
			return DATA, lvl
		}
	}

	timeTypeData := getTimeUnits()
	for lvl, val := range timeTypeData {
		if val == unitID {
			return TIME, lvl
		}
	}

	bwTypeData := getBWUnits()
	for lvl, val := range bwTypeData {
		if val == unitID {
			return BW, lvl
		}
	}

	return -1, -1
}

// ToString converts data about a dataset's unit to a string that is readable
func ToString(unitType int, unitScale int) string {
	switch unitType {
	case DATA:
		internalUnitData := getDataUnits()
		return internalUnitData[unitScale]

	case TIME:
		internalUnitData := getTimeUnits()
		return internalUnitData[unitScale]
	case BW:
		internalUnitData := getBWUnits()
		return internalUnitData[unitScale]
	}
	return ""
}

func IsValidScale(unitType int, newUnitScale int) bool {
	switch unitType {
	case DATA:
		internalUnitData := getDataUnits()
		_, ok := internalUnitData[newUnitScale]
		return ok
	case TIME:
		internalUnitData := getTimeUnits()
		_, ok := internalUnitData[newUnitScale]
		return ok
	case BW:
		internalUnitData := getBWUnits()
		_, ok := internalUnitData[newUnitScale]
		return ok
	}

	return false
}

// IsMax checks if a unit can be scaled up further
func IsMax(unitType int, unitScale int) bool {
	switch unitType {
	case DATA:
		internalUnitData := getDataUnits()
		_, ok := internalUnitData[unitScale+1]
		return ok
	case TIME:
		internalUnitData := getTimeUnits()
		_, ok := internalUnitData[unitScale+1]
		return ok
	case BW:
		internalUnitData := getBWUnits()
		_, ok := internalUnitData[unitScale+1]
		return ok
	}

	return false
}

// IsMin checks if a unit can be scaled down further
func IsMin(unitType int, unitScale int) bool {
	switch unitType {
	case DATA:
		internalUnitData := getDataUnits()
		_, ok := internalUnitData[unitScale-1]
		return ok
	case TIME:
		internalUnitData := getTimeUnits()
		_, ok := internalUnitData[unitScale-1]
		return ok
	case BW:
		internalUnitData := getBWUnits()
		_, ok := internalUnitData[unitScale-1]
		return ok
	}

	return false
}
