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
	"runtime"
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
