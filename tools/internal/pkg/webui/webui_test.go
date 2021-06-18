package webui

import (
	"sort"
	"strconv"
	"testing"
)

func in(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func TestFindCountRankFileList(t *testing.T) {
	pwd := "/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432"
	var filename []string
	for i :=0; i<963;i++ {
		string := strconv.Itoa(i)
		filename=append(filename,pwd+"/counts.rank0_call"+string+".md")
	}
	tests := []struct {
		unit           string
		expectedUnit []string
	}{
		{
			unit:           pwd,
			expectedUnit: filename,
		},
	}

	for _, tt := range tests {
		scaledUnit,err := findCountRankFileList(tt.unit)
		if err != nil {
			t.Fatalf("Ints() failed: %s", err)
		}
		for index, item := range tt.expectedUnit {
			if in(item,scaledUnit) {
				t.Fatalf("Resulting unit is %s instead of %s", scaledUnit[index], item)
			}
		}

	}
}