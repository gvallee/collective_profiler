//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/hash"
	"github.com/gvallee/go_util/pkg/util"
)

const (
	sharedLibCounts      = "liballtoallv_counts.so"
	sharedLibBacktrace   = "liballtoallv_backtrace.so"
	sharedLibLocation    = "liballtoallv_location.so"
	sharedLibLateArrival = "liballtoallv_late_arrival.so"
	sharedLibA2ATime     = "liballtoallv_exec_timings.so"

	exampleFileC          = "alltoallv.c"
	exampleFileF          = "alltoallv.f90"
	exampleFileMulticommC = "alltoallv_multicomms.c"
	exampleFileBigCountsC = "alltoallv_bigcounts.c"

	exampleBinaryC          = "alltoallv_c"
	exampleBinaryF          = "alltoallv_f"
	exampleBinaryMulticommC = "alltoallv_multicomms_c"
	exampleBinaryBigCountsC = "alltoallv_bigcounts_c"
)

// Test gathers all the information required to run a specific test
type Test struct {
	np                             int
	source                         string
	binary                         string
	expectedSendCompactCountsFiles []string
	expectedRecvCompactCountsFiles []string
	expectedCountsFiles            []string
	expectedLocationFiles          []string
	expectedA2ATimeFiles           []string
	expectedLateArrivalFiles       []string
}

func validateCountProfiles(dir string, jobid int, id int) error {
	err := counts.Validate(jobid, id, dir)
	if err != nil {
		return err
	}

	return nil
}

func checkOutputFiles(expectedOutputDir string, tempDir string, expectedFiles []string) error {
	for _, expectedOutputFile := range expectedFiles {
		referenceFile := filepath.Join(expectedOutputDir, expectedOutputFile)
		resultFile := filepath.Join(tempDir, expectedOutputFile)
		fmt.Printf("- Comparing %s and %s...", referenceFile, resultFile)
		hashResultFile, err := hash.File(resultFile)
		if err != nil {
			fmt.Println(" failed")
			return err
		}
		hashRefFile, err := hash.File(referenceFile)
		if err != nil {
			fmt.Println(" failed")
			return err
		}
		if hashRefFile != hashResultFile {
			fmt.Println(" failed")
			return fmt.Errorf("Invalid output, send counters do not match (%s vs. %s)", resultFile, referenceFile)
		}
		fmt.Println(" ok")
	}

	return nil
}

func checkOutput(basedir string, tempDir string, tt Test) error {
	expectedOutputDir := filepath.Join(basedir, "tests", tt.binary, "expectedOutput")

	fmt.Printf("Checking if %s exists...\n", tt.expectedSendCompactCountsFiles)
	err := checkOutputFiles(expectedOutputDir, tempDir, tt.expectedSendCompactCountsFiles)
	if err != nil {
		return err
	}

	fmt.Printf("Checking if %s exists...\n", tt.expectedRecvCompactCountsFiles)
	err = checkOutputFiles(expectedOutputDir, tempDir, tt.expectedRecvCompactCountsFiles)
	if err != nil {
		return err
	}

	// For the other files, we do not at the moment check the content (we could check its
	// format), we just check if the files were correctly generated
	fmt.Printf("Checking if %s exists...\n", tt.expectedA2ATimeFiles)
	for _, file := range tt.expectedA2ATimeFiles {
		execTimingFile := filepath.Join(tempDir, file)
		if !util.FileExists(execTimingFile) {
			return fmt.Errorf("%s is missing", execTimingFile)
		}
	}

	fmt.Printf("Checking if %s exists...\n", tt.expectedLateArrivalFiles)
	for _, file := range tt.expectedLateArrivalFiles {
		lateArrivalFile := filepath.Join(tempDir, file)
		if !util.FileExists(lateArrivalFile) {
			return fmt.Errorf("%s is missing", lateArrivalFile)
		}
	}

	/* todo
	fmt.Printf("Checking if %s exists...\n", tt.expectedLocationFiles[0])
	locationFile := filepath.Join(tempDir, tt.expectedLocationFiles[0])
	if !util.FileExists(locationFile) {
		return fmt.Errorf("%s is missing", locationFile)
	}
	*/

	return nil
}

func validateTestPostmortemResults(testName string, dir string) error {
	toolName := "srcountsanalyzer"
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	toolDir := filepath.Join(basedir, "tools", "cmd", toolName)
	toolBin := filepath.Join(toolDir, toolName)
	if !util.FileExists(toolBin) {
		return fmt.Errorf("%s does not exist", toolBin)
	}

	fmt.Printf("Running mostmortem analysis for %s in %s\n", testName, dir)
	cmd := exec.Command(toolBin, "-dir", dir, "-output-dir", dir, "-jobid", "0", "-rank", "0")
	err := cmd.Run()
	if err != nil {
		return err
	}

	expectedFiles := []string{"profile_alltoallv_job0.rank0.md",
		"stats-job0-rank0.md",
		"patterns-job0-rank0.md",
		"patterns-summary-job0-rank0.md"}
	/* todo:
		"a2a-timings.job0.rank0.md",           // We do not have a good way to check the content but the file should be there
		"late-arrivals-timings.job0.rank0.md", // We do not have a good way to check the content but the file should be there
	}
	*/
	expectedOutputDir := filepath.Join(basedir, "tests", testName, "expectedOutput")
	err = checkOutputFiles(expectedOutputDir, dir, expectedFiles)
	if err != nil {
		return err
	}

	return nil
}

func validatePostmortemAnalysisTools(profilerResults map[string]string) error {
	for source, dir := range profilerResults {
		err := validateTestPostmortemResults(source, dir)
		if err != nil {
			fmt.Printf("validation of the postmortem analysis for %s in %s failed\n", source, dir)
			return err
		}
	}

	// If successful, we can then delete all the directory that were created
	for _, dir := range profilerResults {
		os.RemoveAll(dir)
	}

	return nil
}

// validateProfiler runs the profiler against examples and compare the resuls to the results output.
// If keepResults is set to true, the results are *not* removed after execution. They can then be used
// later on to validate postmortem analysis.
func validateProfiler(keepResults bool) (map[string]string, error) {
	sharedLibraries := []string{sharedLibCounts, sharedLibBacktrace, sharedLibLocation, sharedLibLateArrival, sharedLibA2ATime}
	validationTests := []Test{
		{
			np:                             4,
			source:                         exampleFileC,
			binary:                         exampleBinaryC,
			expectedSendCompactCountsFiles: []string{"send-counters.job0.rank0.txt"},
			expectedRecvCompactCountsFiles: []string{"recv-counters.job0.rank0.txt"},
			// todo: expectedCountsFiles
			expectedLocationFiles:    []string{},
			expectedA2ATimeFiles:     []string{"a2a-timings.job0.rank0.md"},
			expectedLateArrivalFiles: []string{"late-arrivals-timings.job0.rank0.md"},
		},
		{
			np:                             3,
			source:                         exampleFileF,
			binary:                         exampleBinaryF,
			expectedSendCompactCountsFiles: []string{"send-counters.job0.rank0.txt"},
			expectedRecvCompactCountsFiles: []string{"recv-counters.job0.rank0.txt"},
			// todo: expectedCountsFiles
			expectedLocationFiles:    []string{},
			expectedA2ATimeFiles:     []string{"a2a-timings.job0.rank0.md"},
			expectedLateArrivalFiles: []string{"late-arrivals-timings.job0.rank0.md"},
		},
		{
			np:                             4,
			source:                         exampleFileMulticommC,
			binary:                         exampleBinaryMulticommC,
			expectedSendCompactCountsFiles: []string{"send-counters.job0.rank0.txt"},
			expectedRecvCompactCountsFiles: []string{"recv-counters.job0.rank0.txt"},
			// todo: expectedCountsFiles
			expectedLocationFiles:    []string{},
			expectedA2ATimeFiles:     []string{"a2a-timings.job0.rank0.md"},
			expectedLateArrivalFiles: []string{"late-arrivals-timings.job0.rank0.md"},
		},
		{
			np:                             4, // This test runs a large number of interations over a collective with a limited number of ranks
			source:                         exampleFileBigCountsC,
			binary:                         exampleBinaryBigCountsC,
			expectedSendCompactCountsFiles: []string{"send-counters.job0.rank0.txt"},
			expectedRecvCompactCountsFiles: []string{"recv-counters.job0.rank0.txt"},
			// todo: expectedCountsFiles
			expectedLocationFiles:    []string{},
			expectedA2ATimeFiles:     []string{"a2a-timings.job0.rank0.md"},
			expectedLateArrivalFiles: []string{"late-arrivals-timings.job0.rank0.md"},
		},
	}

	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	// Find MPI
	mpiBin, err := exec.LookPath("mpirun")
	if err != nil {
		return nil, err
	}

	// Find make
	makeBin, err := exec.LookPath("make")
	if err != nil {
		return nil, err
	}

	// Compile both the profiler libraries and the example
	log.Println("Building libraries and tests...")
	cmd := exec.Command(makeBin, "clean", "all")
	cmd.Dir = filepath.Join(basedir, "src", "alltoallv")
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	cmd = exec.Command(makeBin, "clean", "all")
	cmd.Dir = filepath.Join(basedir, "examples")
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	// Create a map to store the data about all the directories where
	// results are created when the results need to be kept
	var results map[string]string
	if keepResults {
		results = make(map[string]string)
	}

	for _, tt := range validationTests {
		// Create a temporary directory where to store the results
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}

		if keepResults {
			results[tt.binary] = tempDir
		}

		// Run the profiler
		var stdout, stderr bytes.Buffer
		for _, lib := range sharedLibraries {
			pathToLib := filepath.Join(basedir, "src", "alltoallv", lib)
			fmt.Printf("Running MPI application (%s) and gathering profiles with %s...\n", tt.binary, pathToLib)
			cmd = exec.Command(mpiBin, "-np", strconv.Itoa(tt.np), "--oversubscribe", filepath.Join(basedir, "examples", tt.binary))
			cmd.Env = append(os.Environ(),
				"LD_PRELOAD="+pathToLib,
				"A2A_PROFILING_OUTPUT_DIR="+tempDir)
			cmd.Dir = tempDir
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				return nil, fmt.Errorf("mpirun failed.\n\tstdout: %s\n\tstderr: %s", stdout.String(), stderr.String())
			}
		}

		// Check the results
		err = checkOutput(basedir, tempDir, tt)
		if err != nil {
			return nil, err
		}

		// We clean up *only* when tests are successful and
		// if results do not need to be kept
		if !keepResults {
			os.RemoveAll(tempDir)
		}
	}

	// Return the map describing the data resulting from the tests only
	// when the results need to be kept to later on validate postmortem
	// analysis
	if keepResults {
		return results, nil
	}

	return nil, nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	counts := flag.Bool("counts", false, "Validate the count data generated during the validation run of the profiler with an MPI application. Requires the following additional options: -dir, -job, -id.")
	profiler := flag.Bool("profiler", false, "Perform a validation of the profiler itself running various tests. Requires MPI. Does not require any additional option.")
	postmortem := flag.Bool("postmortem", false, "Perform a validation of the postmortem analysis tools.")
	dir := flag.String("dir", "", "Where all the data is")
	id := flag.Int("id", 0, "Identifier of the experiment, e.g., X from <pidX> in the profile file name")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s validates various aspects of this infrastructure", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	if !*counts && !*profiler && !*postmortem {
		fmt.Println("No validate option select, run '-h' for more details")
		os.Exit(1)
	}

	if *counts {
		err := validateCountProfiles(*dir, *jobid, *id)
		if err != nil {
			fmt.Printf("Validation of the profiler failed: %s\n", err)
			os.Exit(1)
		}
	}

	if *profiler && !*postmortem {
		_, err := validateProfiler(false)
		if err != nil {
			fmt.Printf("Validation of the infrastructure failed: %s\n", err)
			os.Exit(1)
		}
	}

	if *postmortem {
		var err error
		profilerValidationResults := make(map[string]string)

		profilerValidationResults, err = validateProfiler(true)
		if err != nil || profilerValidationResults == nil {
			fmt.Printf("Validation of the infrastructure failed: %s\n", err)
			os.Exit(1)
		}

		err = validatePostmortemAnalysisTools(profilerValidationResults)
		if err != nil {
			fmt.Printf("Validation of the postmortem analysis tools failed: %s\n", err)
			os.Exit(1)
		}
	}

	if *counts {
		err := validateCountProfiles(*dir, *jobid, *id)
		if err != nil {
			fmt.Printf("Validation of the count data failed")
			os.Exit(1)
		}
	}

	fmt.Println("Successful validation")
}
