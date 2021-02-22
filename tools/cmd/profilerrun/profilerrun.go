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

	"github.com/gvallee/go_exec/pkg/advexec"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	libraries := []string{"liballtoallv_counts.so", "liballtoallv_backtrace.so", "liballtoallv_exec_timings.so", "liballtoallv_late_arrival.so", "liballtoallv_location.so"}
	for _, lib := range libraries {
		cmdArgs := os.Args[1:]
		libPath := filepath.Join(codeBaseDir, "src", "alltoallv", lib)
		cmdArgs = append([]string{"-x", "LD_PRELOAD=" + libPath}, cmdArgs...)
		var cmd advexec.Advcmd
		mpirunPath, err := exec.LookPath("mpirun")
		if err != nil {
			fmt.Printf("unable to find mpirun: %s\n", err)
			os.Exit(1)
		}
		cmd.BinPath = mpirunPath
		cmd.CmdArgs = cmdArgs
		cmd.Env = os.Environ()
		fmt.Printf("Executing: %s %s\n", cmd.BinPath, cmd.CmdArgs)
		res := cmd.Run()
		if res.Err != nil {
			fmt.Printf("Unable to run the command: %s - stdout: %s - stderr: %s\n", res.Err, res.Stdout, res.Stderr)
			os.Exit(1)
		}
	}
}
