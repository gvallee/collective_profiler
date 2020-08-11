//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package bins

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/go_util/pkg/util"
)

type Data struct {
	Min  int
	Max  int
	Size int
}

func getOutputFile(dir string, jobid, rank int, b Data) string {
	outputFile := fmt.Sprintf("bin.job%d.rank%d_%d-%d.txt", jobid, rank, b.Min, b.Max)
	if b.Max == -1 {
		outputFile = fmt.Sprintf("bin.job%d.rank%d_%d+.txt", jobid, rank, b.Min)
	}
	if dir != "" {
		outputFile = filepath.Join(dir, outputFile)
	}

	return outputFile
}

func FilesExist(outputDir string, jobid int, rank int, listBins []int) bool {
	bins := Create(listBins) // Create is a cheap operation
	for _, b := range bins {
		if !util.PathExists(getOutputFile(outputDir, jobid, rank, b)) {
			return false
		}
	}
	return true
}

// GetBinsFromInputDescr parses the string describing a series of threshold to use
// for the organization of data into bins and returns a slice of int with each
// element being a threshold
func GetFromInputDescr(binStr string) []int {
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

func Create(listBins []int) []Data {
	var bins []Data

	start := 0
	end := listBins[0]
	for i := 0; i < len(listBins)+1; i++ {
		var b Data
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

func GetFromCounts(counts []string, bins []Data, numCalls int, datatypeSize int) ([]Data, error) {
	for _, c := range counts {
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
	return bins, nil
}

// Save writes the data of all the bins into output file. The output files
// are created in a target output directory.
func Save(dir string, jobid, rank int, bins []Data) error {
	for _, b := range bins {
		outputFile := getOutputFile(dir, jobid, rank, b)
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("unable to create file %s: %s", outputFile, err)
		}
		defer f.Close()

		_, err = f.WriteString(fmt.Sprintf("%d\n", b.Size))
		if err != nil {
			return fmt.Errorf("unable to write bin to file: %s", err)
		}
	}
	return nil
}

// GetFromReader parses a count file using a provided reader and classify all counts
// into bins based on the threshold specified through a slice of integers.
func GetFromReader(reader *bufio.Reader, listBins []int) ([]Data, error) {
	bins := Create(listBins)
	log.Printf("Successfully initialized %d bins\n", len(bins))

	for {
		countsHeader, readerr := counts.GetHeader(reader)
		if readerr == io.EOF {
			break
		}
		if readerr != nil {
			return bins, readerr
		}

		counters, err := counts.GetCounters(reader)
		if err != nil {
			return bins, err
		}

		bins, err := GetFromCounts(counters, bins, len(countsHeader.CallIDs), countsHeader.DatatypeSize)
		if err != nil {
			return bins, err
		}
	}
	return bins, nil
}

// GetFromFile opens a count file and classify all counts into bins
// based on a list of threshold sizes
func GetFromFile(filePath string, listBins []int) ([]Data, error) {
	log.Printf("Creating bins out of values from %s\n", filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	return GetFromReader(reader, listBins)
}
