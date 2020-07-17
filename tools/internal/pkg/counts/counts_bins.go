//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

func getBinOutputFile(dir string, jobid, rank int, b Bin) string {
	outputFile := fmt.Sprintf("bin.job%d.rank%d_%d-%d.txt", jobid, rank, b.Min, b.Max)
	if b.Max == -1 {
		outputFile = fmt.Sprintf("bin.job%d.rank%d_%d+.txt", jobid, rank, b.Min)
	}
	if dir != "" {
		outputFile = filepath.Join(dir, outputFile)
	}

	return outputFile
}

// GetBinsFromInputDescr parses the string describing a series of threshold to use
// for the organization of data into bins and returns a slice of int with each
// element being a threshold
func GetBinsFromInputDescr(binStr string) []int {
	listBinsStr := strings.Split(binStr, ",")
	var listBins []int
	for _, s := range listBinsStr {
		n, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("unable to get array of thresholds for bins: %s", err)
		}
		listBins = append(listBins, n)
	}
	return listBins
}

func createBins(listBins []int) []Bin {
	var bins []Bin

	start := 0
	end := listBins[0]
	for i := 0; i < len(listBins)+1; i++ {
		var b Bin
		b.Min = start
		b.Max = end
		b.Size = 0

		start = end
		if i+1 < len(listBins) {
			end = listBins[i+1]
		} else {
			end = -1 // Means no max
		}

		bins = append(bins, b)
	}

	return bins
}

// getBins parses a count file using a provided reader and classify all counts
// into bins based on the threshold specified through a slice of integers.
func getBins(reader *bufio.Reader, listBins []int) ([]Bin, error) {
	bins := createBins(listBins)
	log.Printf("Successfully initialized %d bins\n", len(bins))

	for {
		_, numCalls, _, _, _, datatypeSize, readerr := GetHeader(reader)
		if readerr == io.EOF {
			break
		}
		if readerr != nil {
			return bins, readerr
		}

		counters, err := GetCounters(reader)
		if err != nil {
			return bins, err
		}
		for _, c := range counters {
			tokens := strings.Split(c, ": ")
			ranks := tokens[0]
			counts := strings.TrimRight(tokens[1], "\n")
			ranks = strings.TrimLeft(ranks, "Rank(s) ")
			listRanks, err := notation.ConvertCompressedCallListToIntSlice(ranks)
			if err != nil {
				return bins, err
			}
			nRanks := len(listRanks)

			// Now we parse the counts one by one
			for _, oneCount := range strings.Split(counts, " ") {
				if oneCount == "" {
					continue
				}

				countVal, err := strconv.Atoi(oneCount)
				if err != nil {
					return bins, err
				}

				val := countVal * datatypeSize
				for i := 0; i < len(bins); i++ {
					if (bins[i].Max != -1 && bins[i].Min <= val && val < bins[i].Max) || (bins[i].Max == -1 && val >= bins[i].Min) {
						bins[i].Size += numCalls * nRanks
						break
					}
				}
			}
		}
	}
	return bins, nil
}

// GetBinsFromFile opens a count file and classify all counts into bins
// based on a list of threshold sizes
func GetBinsFromFile(filePath string, listBins []int) ([]Bin, error) {
	log.Printf("Creating bins out of values from %s\n", filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	return getBins(reader, listBins)
}
