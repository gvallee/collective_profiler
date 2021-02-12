//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package format

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	// ProfileSummaryFilePrefix is the prefix used for all generated profile summary files
	ProfileSummaryFilePrefix = "profile_alltoallv_rank"

	// MulticommHighlightFilePrefix is the prefix of the file used to store the highlights when data has multi-communicators patterns
	MulticommHighlightFilePrefix = "multicomm-highlights"

	// DefaultMsgSizeThreshold is the default threshold to differentiate message and large messages.
	DefaultMsgSizeThreshold = 200

	// DataFormatVersionFilename is the name of the file with the version of the data format
	DataFormatVersionFilename = "FORMAT_VERSION"

	// DataFormatHeader is the string used at the begining of data file to specify which format version is used
	DataFormatHeader = "FORMAT_VERSION: "
)

// KV is a structure used to transform a map[int]int into
// an ordered array.
type KV struct {
	Key int
	Val int
}

// KVList is a type representing a slice of KVs.
type KVList []KV

// Len returns the length of a KVList; part of the API used for value-based sorting.
func (x KVList) Len() int { return len(x) }

// Less compares two KV elewment and returns true when the value of the first
// element is lesser than the one of the second element. Part of the API used
// for value-based sorting.
func (x KVList) Less(i, j int) bool { return x[i].Val < x[j].Val }

// Swap swaps two KV element in a list. Part of the API used for value-based sorting.
func (x KVList) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ConvertIntMapToOrderedArrayByValue converts a map[int]int into
// a ordered array based on the values of the map. This is mainly
// used to have predictable output since a map is not by nature
// ordered.
func ConvertIntMapToOrderedArrayByValue(m map[int]int) KVList {
	var sortedArray KVList
	for k, v := range m {
		sortedArray = append(sortedArray, KV{Key: k, Val: v})
	}
	sort.Sort(sortedArray)
	return sortedArray
}

// GetDataFormatVersion returns the version of the data format of the repostiory
func GetDataFormatVersion(codeBaseDir string) (int, error) {
	content, err := ioutil.ReadFile(filepath.Join(codeBaseDir, DataFormatVersionFilename))
	if err != nil {
		return -1, err
	}
	str := string(content)
	str = strings.TrimRight(str, "\n")
	version, err := strconv.Atoi(str)
	if err != nil {
		return -1, err
	}
	return version, nil
}

func CheckDataFormat(version int, codeBaseDir string) error {
	myVersion, err := GetDataFormatVersion(codeBaseDir)
	if err != nil {
		return err
	}

	if myVersion != version {
		return fmt.Errorf("unmatching data format: %d vs. %d", myVersion, version)
	}

	return nil
}
