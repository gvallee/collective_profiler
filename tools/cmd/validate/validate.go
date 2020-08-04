//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/hash"
	"github.com/gvallee/go_util/pkg/util"
)

func validateCountProfiles(dir string, jobid int, id int) error {
	err := counts.Validate(jobid, id, dir)
	if err != nil {
		return err
	}

	return nil
}

func validateProfiler() error {
	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	// Find MPI
	mpiBin, err := exec.LookPath("mpirun")
	if err != nil {
		return err
	}

	// Find make
	makeBin, err := exec.LookPath("make")
	if err != nil {
		return err
	}

	// Compile both the profiler libraries and the example
	log.Println("Building libraries and tests...")
	cmd := exec.Command(makeBin, "clean", "all")
	cmd.Dir = filepath.Join(basedir, "alltoallv")
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command(makeBin, "clean", "all")
	cmd.Dir = filepath.Join(basedir, "examples")
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Create a temporary directory where to store the results
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	// Run the profiler
	pathToLib := filepath.Join(basedir, "alltoallv", "liballtoallv_counts.so")
	fmt.Printf("Running MPI application and gathering profiles with %s...\n", pathToLib)
	cmd = exec.Command(mpiBin, "-np", "3", "--oversubscribe", filepath.Join(basedir, "examples", "alltoallv_f"))
	cmd.Env = append(os.Environ(),
		"LD_PRELOAD="+pathToLib,
		"A2A_PROFILING_OUTPUT_DIR="+tempDir)
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Check the results
	expectedOutputDir := filepath.Join(basedir, "tests", "alltoallv_f", "expectedOutput")

	referenceSendProfileFile := filepath.Join(expectedOutputDir, "send-counters.job0.rank0.txt")
	resultSendProfileFile := filepath.Join(tempDir, "send-counters.job0.rank0.txt")
	fmt.Printf("Comparing %s and %s...", referenceSendProfileFile, resultSendProfileFile)
	hashSendProfile, err := hash.File(resultSendProfileFile)
	if err != nil {
		fmt.Println(" failed")
		return err
	}
	hashRefSendProfile, err := hash.File(referenceSendProfileFile)
	if err != nil {
		fmt.Println(" failed")
		return err
	}
	if hashRefSendProfile != hashSendProfile {
		fmt.Println(" failed")
		return fmt.Errorf("Invalid output, send counters do not match")
	}
	fmt.Println(" ok")

	resultRecvProfileFile := filepath.Join(tempDir, "recv-counters.job0.rank0.txt")
	referenceRecvProfileFile := filepath.Join(expectedOutputDir, "recv-counters.job0.rank0.txt")
	fmt.Printf("Comparing %s and %s...", referenceRecvProfileFile, resultRecvProfileFile)
	hashRecvProfile, err := hash.File(resultRecvProfileFile)
	if err != nil {
		fmt.Println(" failed")
		return err
	}
	hashRefRecvProfile, err := hash.File(referenceRecvProfileFile)
	if err != nil {
		fmt.Println(" failed")
		return err
	}
	if hashRefRecvProfile != hashRecvProfile {
		fmt.Println(" failed")
		return fmt.Errorf("Invalid output, recv counters do not match")
	}
	fmt.Println(" ok")

	return nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	counts := flag.Bool("counts", false, "Validate the count data generated during the validation run of the profiler with an MPI application. Requires the following additional options: -dir, -job, -id.")
	profiler := flag.Bool("profiler", false, "Perform a validation of the profiler itself running various tests. Requires MPI. Does not require any additional option.")
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

	if !*counts && !*profiler {
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

	if *profiler {
		err := validateProfiler()
		if err != nil {
			fmt.Printf("Validation of the infrastructure failed: %s\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Successful validation")
}
