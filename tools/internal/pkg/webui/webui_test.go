package webui

import (
	"testing"
)

func TestFindCountRankFileList(t *testing.T) {
	pwd := "/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432"
	for i :=range {

	}
	tests := []struct {
		unit           string
		expectedUnit []string
	}{
		{
			unit:           pwd,
			expectedUnit: []string{pwd+"counts.rank0_call844","das"},
		},
	}

	for _, tt := range tests {
		scaledUnit,err := findCountRankFileList(tt.unit)
		if err != nil {
			t.Fatalf("Ints() failed: %s", err)
		}
		for index, item := range tt.expectedUnit {
			if item != scaledUnit[index] {
				t.Fatalf("Resulting unit is %s instead of %s", scaledUnit[index], item)
			}
		}

	}
}