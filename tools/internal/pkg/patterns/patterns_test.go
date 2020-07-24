//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package patterns

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
)

func compareFiles(t *testing.T, file1 string, file2 string) {
	file1Fd, err := os.Open(file1)
	if err != nil {
		t.Fatalf("os.Open() failed: %s", err)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, file1Fd)
	if err != nil {
		t.Fatalf("io.Copy() failed: %s", err)
	}
	hashFile1 := hex.EncodeToString(hasher.Sum(nil))

	file2Fd, err := os.Open(file2)
	if err != nil {
		t.Fatalf("os.Open() failed: %s", err)
	}
	hasher = sha256.New()
	_, err = io.Copy(hasher, file2Fd)
	hashFile2 := hex.EncodeToString(hasher.Sum(nil))

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
			name:       "oneRankSend1To10Recv0",
			sendCounts: []string{"Rank(s) 0: 1 2 3 4 5 6 7 8 9 10"},
			recvCounts: []string{"Rank(s) 0: 1 0 0 0 0 0 0 0 0 0"},
		},
		{
			name:       "threeRanksSendNToNRecvNToN",
			sendCounts: []string{"Rank(s) 0-2: 3 3 3"},
			recvCounts: []string{"Rank(s) 0-2: 1 1 1"},
		},
		{
			name:       "threeRanksNto1",
			sendCounts: generateCounts(0, 109, 109, 1, 3),                                              // Ranks 0-109: 109 zero counts and one count equal to 3
			recvCounts: append(generateCounts(0, 0, 1, 110, 42), generateCounts(1, 109, 110, 0, 0)...), //Rank 0: 1 count equal to 0 and 110 counts equal to 42; ranks 1-109:  110 zero counts, non counts with non-zero vaue
		},
	}

	for _, tt := range tests {
		var patterns Data
		sendStats, err := counts.AnalyzeCounts(tt.sendCounts, 200, 8)
		if err != nil {
			t.Fatalf("AnalyzeCounts() failed: %s", err)
		}
		recvStats, err := counts.AnalyzeCounts(tt.recvCounts, 200, 8)
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
