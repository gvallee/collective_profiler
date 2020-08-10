//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/hash"
)

func TestConvertRawToCompactFormat(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(filename)

	tests := []struct {
		dirs                   []string
		expectedSendCountsFile string
		expectedRecvCountsFile string
	}{
		{
			dirs:                   []string{filepath.Join(basedir, "testData", "set1", "input")},
			expectedSendCountsFile: filepath.Join(basedir, "testData", "set1", "expectedOutput", "send-counters.job0.rank0.txt"),
			expectedRecvCountsFile: filepath.Join(basedir, "testData", "set1", "expectedOutput", "recv-counters.job0.rank0.txt"),
		},
	}

	for _, tt := range tests {
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("unable to create temporary directory")
		}
		defer os.RemoveAll(tempDir)

		err = LoadRawCountsFromDirs(tt.dirs, tempDir)
		if err != nil {
			t.Fatalf("LoadRawCountsFromDirs() failed: %s", err)
		}

		resultSendFile := filepath.Join(tempDir, filepath.Base(tt.expectedSendCountsFile))
		resultRecvFile := filepath.Join(tempDir, filepath.Base(tt.expectedRecvCountsFile))

		hashS1, err := hash.File(resultSendFile)
		if err != nil {
			t.Fatalf("hash.File() failed: %s", err)
		}
		hashS2, err := hash.File(tt.expectedSendCountsFile)
		if err != nil {
			t.Fatalf("hash.File() failed: %s", err)
		}
		hashR1, err := hash.File(resultRecvFile)
		if err != nil {
			t.Fatalf("hash.File() failed: %s", err)
		}
		hashR2, err := hash.File(tt.expectedRecvCountsFile)
		if err != nil {
			t.Fatalf("hash.File() failed: %s", err)
		}

		if hashS1 != hashS2 {
			t.Fatalf("%s and %s differ", resultSendFile, tt.expectedSendCountsFile)
		}

		if hashR1 != hashR2 {
			t.Fatalf("%s and %s differ", resultRecvFile, tt.expectedRecvCountsFile)
		}
	}
}

func TestLoadCallsData(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(filename)

	tests := []struct {
		inputSendCountsFile  string
		inputRecvCountsFile  string
		expectedSendCounts   map[int][]string
		expectedRecvCounts   map[int][]string
		expectedSendPatterns map[int][]int
	}{
		{
			inputSendCountsFile: filepath.Join(basedir, "testData", "set2", "input", "send-counters.job0.rank0.txt"),
			inputRecvCountsFile: filepath.Join(basedir, "testData", "set2", "input", "recv-counters.job0.rank0.txt"),
			expectedSendCounts: map[int][]string{
				0: []string{"Rank(s) 0-3: 0 0 0 0"},
				1: []string{"Rank(s) 0,2-3: 0 0 0 0", "Rank(s) 1: 1 1 1 1"},
				2: []string{"Rank(s) 0,2-3: 0 0 0 0", "Rank(s) 1: 1 1 1 1"},
			},
			expectedRecvCounts: map[int][]string{
				0: []string{"Rank(s) 0-3: 0 0 0 0"},
				1: []string{"Rank(s) 0,2-3: 0 0 0 0", "Rank(s) 1: 1 1 1 1"},
				2: []string{"Rank(s) 0,2-3: 0 0 0 0", "Rank(s) 1: 1 1 1 1"},
			},
			expectedSendPatterns: map[int][]int{
				1: []int{4, 1}, // Call #1: 1 rank sends to 4 others
				2: []int{4, 1}, // Call #1: 1 rank sends to 4 others
			},
		},
	}

	for _, tt := range tests {
		data, err := LoadCallsData(tt.inputSendCountsFile, tt.inputRecvCountsFile, 0, 0)
		if err != nil {
			t.Fatalf("LoadCallsData() failed: %s", err)
		}

		if len(data) != 3 {
			t.Fatalf("Number of call mismatch: %d vs. 3", len(data))
		}

		for callID, callData := range data {
			if callData.CommSize != 4 {
				t.Fatalf("Comm size mismatch: %d vs. 4", callData.CommSize)
			}

			if callData.MsgSizeThreshold != 0 {
				t.Fatalf("Message size threshold mismatch: %d vs. 0", callData.MsgSizeThreshold)
			}

			if callData.SendData.File != tt.inputSendCountsFile {
				t.Fatalf("Send count file mismatch: %s vs. %s", callData.SendData.File, tt.inputSendCountsFile)
			}

			if callData.RecvData.File != tt.inputRecvCountsFile {
				t.Fatalf("Send count file mismatch: %s vs. %s", callData.RecvData.File, tt.inputRecvCountsFile)
			}

			if !reflect.DeepEqual(callData.SendData.RawCounts, tt.expectedSendCounts[callID]) {
				t.Fatalf("Wrong counts for call %d: .%s. vs. .%s.", callID, strings.Join(callData.SendData.RawCounts, "|"), strings.Join(tt.expectedSendCounts[callID], "|"))
			}

			if !reflect.DeepEqual(callData.RecvData.RawCounts, tt.expectedRecvCounts[callID]) {
				t.Fatalf("Wrong counts for call %d: .%s. vs. .%s.", callID, strings.Join(callData.RecvData.RawCounts, "|"), strings.Join(tt.expectedRecvCounts[callID], "|"))
			}

			// todo: deal with callData.SendData.Stats / callData.RecvData.Stats
			if len(callData.SendData.Statistics.Patterns) != len(tt.expectedSendPatterns[callID])/2 {
				t.Fatalf("# of patterns mismatch for call %d: %d vs. %d patterns detected", callID, len(callData.SendData.Statistics.Patterns), len(tt.expectedSendPatterns[callID])/2)
			}
			idx := 0
			for key, val := range callData.SendData.Statistics.Patterns {
				if tt.expectedSendPatterns[callID][idx*2] != key || tt.expectedSendPatterns[callID][idx*2+1] != val {
					t.Fatalf("Pattern mismatch %d/%d vs %d/%d", key, val, tt.expectedSendPatterns[callID][idx*2], tt.expectedSendPatterns[callID][idx*2+1])
				}
				idx++
			}

			// todo: deal with callData.SendData.BinThresholds / callData.RecvData.BinThresholds
			// todo: deal with callData.SendData.MsgSizeThreshold / callData.RecvData.MsgSizeThreshold
		}
	}

}
