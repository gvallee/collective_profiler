//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package timings

import (
	"path/filepath"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
)

func ParseFile(filePath string, outputDir string) error {
	lateArrivalFilename := strings.ReplaceAll(filepath.Base(filePath), "timings", "late_arrival_timings")
	lateArrivalFilename = strings.ReplaceAll(lateArrivalFilename, ".md", ".dat")
	a2aFilename := strings.ReplaceAll(filepath.Base(filePath), "timings", "alltoallv_timings")
	a2aFilename = strings.ReplaceAll(a2aFilename, ".md", ".dat")
	if outputDir != "" {
		lateArrivalFilename = filepath.Join(outputDir, lateArrivalFilename)
		a2aFilename = filepath.Join(outputDir, a2aFilename)
	}

	err := datafilereader.ExtractTimings(filePath, lateArrivalFilename, a2aFilename)
	if err != nil {
		return err
	}

	return nil
}
