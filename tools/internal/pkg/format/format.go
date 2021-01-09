//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package format

import "sort"

const (

	// ProfileSummaryFilePrefix is the prefix used for all generated profile summary files
	ProfileSummaryFilePrefix = "profile_alltoallv_rank"

	// MulticommHighlightFilePrefix is the prefix of the file used to store the highlights when data has multi-communicators patterns
	MulticommHighlightFilePrefix = "multicomm-highlights"

	// DefaultMsgSizeThreshold is the default threshold to differentiate message and large messages.
	DefaultMsgSizeThreshold = 200
)

type KV struct {
	Key int
	Val int
}
type KVList []KV

func (x KVList) Len() int           { return len(x) }
func (x KVList) Less(i, j int) bool { return x[i].Val < x[j].Val }
func (x KVList) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func ConvertIntMapToOrderedArrayByValue(m map[int]int) KVList {
	var sortedArray KVList
	for k, v := range m {
		sortedArray = append(sortedArray, KV{Key: k, Val: v})
	}
	sort.Sort(sortedArray)
	return sortedArray
}
