//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package patterns

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/hash"
)

func compareFiles(t *testing.T, file1 string, file2 string) {
	hashFile1, err := hash.File(file1)
	if err != nil {
		t.Fatalf("hash.File() failed: %s", err)
	}
	hashFile2, err := hash.File(file2)
	if err != nil {
		t.Fatalf("hash.File() failed: %s", err)
	}

	if hashFile1 != hashFile2 {
		t.Fatalf("resulting file is not as expected: hashes are %s and %s (%s %s)", hashFile1, hashFile2, file1, file2)
	}
}

func generateCounts(rankStart int, rankEnd int, numZeros int, numNonZeros int, value int) []string {
	str := "Rank(s) " + strconv.Itoa(rankStart) + "-" + strconv.Itoa(rankEnd) + ":"
	if rankStart == rankEnd {
		str = "Rank(s) " + strconv.Itoa(rankStart) + ":"
	}

	for i := 0; i < numZeros; i++ {
		str += " 0"
	}

	for i := 0; i < numNonZeros; i++ {
		str += " " + strconv.Itoa(value)
	}

	return []string{str}
}

func TestParsingCounts(t *testing.T) {
	tests := []struct {
		name       string
		sendCounts []string
		recvCounts []string
	}{
		{
			name: "oneRankSend1To10Recv0",
			sendCounts: []string{"Rank(s) 0: 1 2 3 4 5 6 7 8 9 10",
				"Rank(s) 1-9: 0 0 0 0 0 0 0 0 0 0"},
			recvCounts: []string{"Rank(s) 0: 1 0 0 0 0 0 0 0 0 0",
				"Rank(s) 1: 2 0 0 0 0 0 0 0 0 0",
				"Rank(s) 2: 3 0 0 0 0 0 0 0 0 0",
				"Rank(s) 3: 4 0 0 0 0 0 0 0 0 0",
				"Rank(s) 4: 5 0 0 0 0 0 0 0 0 0",
				"Rank(s) 5: 6 0 0 0 0 0 0 0 0 0",
				"Rank(s) 6: 7 0 0 0 0 0 0 0 0 0",
				"Rank(s) 7: 8 0 0 0 0 0 0 0 0 0",
				"Rank(s) 8: 9 0 0 0 0 0 0 0 0 0",
				"Ranks(s) 9: 10 0 0 0 0 0 0 0 0 0"},
		},
		{
			name:       "threeRanksSendNToNRecvNToN",
			sendCounts: []string{"Rank(s) 0-2: 3 3 3"},
			recvCounts: []string{"Rank(s) 0-2: 1 1 1"},
		},
		{
			name:       "threeRanksNto1",
			sendCounts: generateCounts(0, 109, 109, 1, 3),                                              // Ranks 0-109: 109 zero counts and one count equal to 3
			recvCounts: append(generateCounts(0, 0, 1, 110, 42), generateCounts(1, 109, 110, 0, 0)...), // Rank 0: 1 count equal to 0 and 110 counts equal to 42; ranks 1-109:  110 zero counts, non counts with non-zero vaue
		},
	}

	for _, tt := range tests {
		var patterns Data
		sendStats, _, err := counts.AnalyzeCounts(tt.sendCounts, 200, 8)
		if err != nil {
			t.Fatalf("AnalyzeCounts() failed: %s", err)
		}
		recvStats, _, err := counts.AnalyzeCounts(tt.recvCounts, 200, 8)
		if err != nil {
			t.Fatalf("AnalyzeCounts() failed: %s", err)
		}
		err = patterns.addPattern(0, sendStats.Patterns, recvStats.Patterns)
		if err != nil {
			t.Fatalf("addPattern() failed: %s\n", err)
		}

		patternsFd, err := ioutil.TempFile("", "patterns-"+tt.name)
		if err != nil {
			t.Fatalf("ioutil.TempFile() failed: %s\n", err)
		}
		defer os.Remove(patternsFd.Name())
		patternsSummaryFd, err := ioutil.TempFile("", "summary-"+tt.name)
		if err != nil {
			t.Fatalf("ioutil.TempFile() failed: %s\n", err)
		}
		defer os.Remove(patternsSummaryFd.Name())
		err = WriteData(patternsFd, patternsSummaryFd, patterns, 1)
		if err != nil {
			t.Fatalf("writePatternsData() failed: %s\n", err)
		}

		// We close and reopen the file to make sure everything is synced when we get the hash
		patternsFilepath := patternsFd.Name()
		err = patternsFd.Close()
		if err != nil {
			t.Fatalf("Sync() failed: %s", err)
		}

		summaryFilePath := patternsSummaryFd.Name()
		err = patternsSummaryFd.Close()
		if err != nil {
			t.Fatalf("Sync() failed: %s", err)
		}

		_, filename, _, _ := runtime.Caller(0)
		refPatternsFilepath := filepath.Join(filepath.Dir(filename), "test_expectedOutputFiles", "patterns-"+tt.name+".txt")
		refSummaryFilepath := filepath.Join(filepath.Dir(filename), "test_expectedOutputFiles", "summary-"+tt.name+".txt")

		compareFiles(t, patternsFilepath, refPatternsFilepath)
		compareFiles(t, summaryFilePath, refSummaryFilepath)
	}
}

func compareArrayCallData(t *testing.T, data1 []*CallData, data2 []*CallData) {
	if len(data1) != len(data2) {
		t.Fatalf("sizes differ: length is %d instead of %d", len(data1), len(data2))
	}
	for i := 0; i < len(data1); i++ {
		if len(data1[i].Send) != len(data2[i].Send) {
			t.Fatalf("Send patterns are of length %d instead of %d", len(data1[i].Send), len(data2[i].Send))
		}
		if len(data1[i].Recv) != len(data2[i].Recv) {
			t.Fatalf("Recv patterns are of length %d instead of %d", len(data1[i].Recv), len(data2[i].Recv))
		}
		if len(data1[i].Calls) != len(data2[i].Calls) {
			t.Fatalf("Send patterns are of length %d instead of %d", len(data1[i].Calls), len(data2[i].Calls))
		}
		for j := 0; j < len(data1[i].Calls); j++ {
			if data1[i].Calls[j] != data2[i].Calls[j] {
				t.Fatalf("Pattern call #%d is %d instead of %d", j, data1[i].Calls[j], data2[i].Calls[j])
			}
		}
		for k, v := range data1[i].Send {
			if data2[i].Send[k] != v {
				t.Fatalf("Value for key %d is %d instead of %d", k, data2[i].Send[k], v)
			}
		}
		for k, v := range data1[i].Recv {
			if data2[i].Recv[k] != v {
				t.Fatalf("Value for key %d is %d instead of %d", k, data2[i].Recv[k], v)
			}
		}
	}
}

func TestParseFile(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(filename)

	tests := []struct {
		sendCountsFile string
		recvCountsFile string
		numCalls       int
		leadRank       int
		sizeThreshold  int
		expectedOutput Data
	}{
		{
			sendCountsFile: filepath.Join(basedir, "testData", "set1", "input", "send-counters.job0.rank0.txt"),
			recvCountsFile: filepath.Join(basedir, "testData", "set1", "input", "recv-counters.job0.rank0.txt"),
			numCalls:       3,
			leadRank:       0,
			sizeThreshold:  200,
			expectedOutput: Data{
				AllPatterns: []*CallData{
					{
						Send: map[int]int{
							1023: 1024,
						},
						Recv: map[int]int{
							1023: 1024,
						},
						Count: 1,
						Calls: []int{0, 1, 2},
					},
				},
				NToN: []*CallData{
					{
						Send: map[int]int{
							1023: 1024,
						},
						Recv: map[int]int{
							1023: 1024,
						},
						Count: 1,
						Calls: []int{0, 1, 2},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		_, patterns, err := ParseFiles(tt.sendCountsFile, tt.recvCountsFile, tt.numCalls, tt.leadRank, tt.sizeThreshold)
		if err != nil {
			t.Fatalf("ParseFile() failed: %s", err)
		}
		t.Log("Comparing AllPatterns list...")
		compareArrayCallData(t, tt.expectedOutput.AllPatterns, patterns.AllPatterns)
		t.Log("Comparing OneToN list...")
		compareArrayCallData(t, tt.expectedOutput.OneToN, patterns.OneToN)
		t.Log("Comparing NToN list...")
		compareArrayCallData(t, tt.expectedOutput.NToN, patterns.NToN)
		t.Log("Comparing NToOne list...")
		compareArrayCallData(t, tt.expectedOutput.NToOne, patterns.NToOne)
		t.Log("Comparing Empty patterns...")
		compareArrayCallData(t, tt.expectedOutput.Empty, patterns.Empty)
		t.Log("All done for test.")
	}
}
