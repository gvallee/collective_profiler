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
)

func getDataUnits() map[int]string {
	return map[int]string{
		0: "B",
		1: "KB",
		2: "MB",
		3: "GB",
		4: "TB"}
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
	}

	return false
}

/*
func getBandwidthUnits() map[int]string {
	return map[int]string{
		0: "B/s",
	}
}
*/
