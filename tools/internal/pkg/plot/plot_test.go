//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package plot

import "testing"

func TestSortHostMapKeys(t *testing.T) {
	tests := []struct {
		inputMap       map[string][]int
		expectedOutput []string
	}{
		{
			inputMap: map[string][]int{
				"node1": {1, 3, 5, 7, 9, 11},
				"node0": {0, 2, 4, 6, 8, 10},
			},
			expectedOutput: []string{"node0", "node1"},
		},
	}

	for _, tt := range tests {
		listNodes := sortHostMapKeys(tt.inputMap)
		if len(listNodes) != len(tt.expectedOutput) {
			t.Fatalf("sortHostMapKeys() returned %d elements instead of %d", len(listNodes), len(tt.expectedOutput))
		}
		for i := 0; i < len(tt.expectedOutput); i++ {
			if listNodes[i] != tt.expectedOutput[i] {
				t.Fatalf("element %d is %s instead of %s", i, listNodes[i], tt.expectedOutput[i])
			}
		}
	}
}
