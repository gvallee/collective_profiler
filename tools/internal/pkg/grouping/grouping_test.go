//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package grouping

import (
	"strings"
	"testing"
)

func TestSplitting(t *testing.T) {
	tests := []struct {
		elts       []int
		splitIndex int
		result     [][]int
	}{

		{
			elts:       []int{0, 1, 2},
			splitIndex: 1,
			result: [][]int{
				[]int{0},
				[]int{1, 2},
			},
		},
		{
			elts:       []int{1, 2, 3},
			splitIndex: 2,
			result: [][]int{
				[]int{1, 2},
				[]int{3},
			},
		},
	}

	for _, tt := range tests {
		e := Init()
		for j := 0; j < len(tt.elts); j++ {
			err := e.AddDatapoint(j, tt.elts)
			if err != nil {
				t.Fatalf("unable to add point: %s", err)
			}
		}

		t.Logf("Group ready, splitting at %d", tt.splitIndex)
		ng, err := e.splitGroup(e.Groups[0], tt.splitIndex, tt.elts)
		if err != nil {
			t.Fatalf("unable to split group: %s", err)
		}
		if len(ng.Elts) != len(tt.result[1]) {
			str := groupToString(ng.Elts)
			t.Fatalf("result group is of length %d instead of the expected %d (%s)", len(ng.Elts), len(tt.result[1]), str)
		}

		// We check the content of the first group
		if len(e.Groups[0].Elts) != len(tt.result[0]) {
			t.Fatalf("invalid first group after split: %d elements vs. %d expected", len(e.Groups[0].Elts), len(tt.result[0]))
		}

		for g := 0; g < 2; g++ {
			for n := 0; n < len(e.Groups[g].Elts); n++ {
				if tt.elts[e.Groups[g].Elts[n]] != tt.result[g][n] {
					t.Fatalf("Element %d of result group #%d is %d instead of expected %d (group is:%s)", n, g, e.Groups[g].Elts[n], tt.result[g][n], groupToString(e.Groups[g].Elts))
				}
			}
		}
	}
}

func TestGrouping(t *testing.T) {
	tests := []struct {
		points       []int
		groupsResult [][]int
	}{
		{
			points: []int{1, 2, 3, 3, 3},
			groupsResult: [][]int{
				[]int{1, 2},
				[]int{3, 3, 3},
			},
		},
		{
			points: []int{1, 2, 3},
			groupsResult: [][]int{
				[]int{1, 2, 3},
			},
		},
		{
			points: []int{1, 2, 3, 5},
			groupsResult: [][]int{
				[]int{1, 2, 3, 5},
			},
		},
		{
			points: []int{1, 2, 3, 10, 11, 12},
			groupsResult: [][]int{
				[]int{1, 2, 3},
				[]int{10, 11, 12},
			},
		},
		{
			points: []int{0, 1, 2, 5, 6, 7, 20, 30, 25},
			groupsResult: [][]int{
				[]int{0, 1, 2},
				[]int{5, 6, 7},
				[]int{20, 25, 30},
			},
		},
		{
			points: []int{100, 0, 1, 5, 6, 7, 20, 2, 30, 25},
			groupsResult: [][]int{
				[]int{0, 1, 2},
				[]int{5, 6, 7},
				[]int{20, 25, 30},
				[]int{100},
			},
		},
	}

	num := 1
	for _, tt := range tests {
		e := Init()
		t.Logf("Running test %d", num)
		for j := 0; j < len(tt.points); j++ {
			t.Logf("-> Adding %d\n", tt.points[j])
			err := e.AddDatapoint(j, tt.points)
			if err != nil {
				t.Fatalf("unable to add point: %s", err)
			}
		}

		// Compare the resulting groups with what we expect
		gps, err := e.GetGroups()
		if err != nil {
			t.Logf("unable to get groups: %s", err)
		}

		var groupsStr []string
		for _, g := range gps {
			groupsStr = append(groupsStr, groupToString(g.Elts))
		}
		if len(gps) != len(tt.groupsResult) {
			t.Fatalf("test %d reports %d groups instead of %d (groups: %s)\n", num, len(gps), len(tt.groupsResult), strings.Join(groupsStr, "; "))
		}
		for k := 0; k < len(tt.groupsResult); k++ {
			for l := 0; l < len(gps[k].Elts); l++ {
				if len(gps[k].Elts) != len(tt.groupsResult[k]) {
					t.Fatalf("returned group #%d has %d elements while expecting %d\n", k, len(gps[k].Elts), len(tt.groupsResult[k]))
				}
				if tt.groupsResult[k][l] != tt.points[gps[k].Elts[l]] {
					t.Fatalf("element %d of group %d is %d instead of %d\n", l, k, tt.points[gps[k].Elts[l]], tt.groupsResult[k][l])
				}
			}
		}
		num++
	}
}
