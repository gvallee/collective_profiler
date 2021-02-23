//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	libraries := []string{"liballtoallv_counts.so", "liballtoallv_backtrace.so", "liballtoallv_exec_timings.so", "liballtoallv_late_arrival.so", "liballtoallv_location.so"}
	for _, lib := range libraries {
		cmdArgs := os.Args[1:]
		libPath := filepath.Join(codeBaseDir, "src", "alltoallv", lib)
		cmdArgs = append([]string{"-x", "LD_PRELOAD=" + libPath}, cmdArgs...)
		var cmd exec.Cmd
		mpirunPath, err := exec.LookPath("mpirun")
		if err != nil {
			fmt.Printf("unable to find mpirun: %s\n", err)
			os.Exit(1)
		}
		cmd.Path = mpirunPath
		cmd.Args = cmdArgs
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()
		fmt.Printf("Executing: %s %s\n", cmd.Path, cmd.Args)
		err = cmd.Run()
		if err != nil {
			fmt.Printf("command failed: %s\n", err)
			// DO NOT EXIT HERE. We have application that do not terminate cleanly
			// If we have an error we report it and move to the next run
		}
	}
}
