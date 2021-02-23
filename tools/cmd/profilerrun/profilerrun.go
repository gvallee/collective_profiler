//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	ioTimeout = 1
)

// getStdin gets all available content from stdin.
// Note that golang does not have any API for non-blocking IO operations
// so we need to be clever to be able to read data from stdin when available
// but not block if nothing is available. We assume here the data of interest
// from stdin is because the user specify writing to stdin from the command line
// e.g., profilerrun -np 2 myapp < input.txt.
func getStdin() string {
	stdinText := ""
	ch := make(chan string)
	go func(ch chan string) {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					ch <- s
				}
				close(ch)
				return
			}
			ch <- s
		}
	}(ch)

readfromstdin:
	for {
		// We either read data from stdin or timeout if we did not get any data within 1 second
		select {
		case text, ok := <-ch:
			if !ok {
				break readfromstdin
			} else {
				stdinText += text
			}
		case <-time.After(ioTimeout * time.Second):
			return stdinText
		}
	}
	return stdinText
}

func main() {
	_, filename, _, _ := runtime.Caller(0)
	codeBaseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	// Check if there is anything to read from stdin, if so, save what is there so we
	// can feed it to the various mpirun sub-commands we will execute
	stdinText := getStdin()

	stderr := os.Stderr
	stdout := os.Stdout

	libraries := []string{"liballtoallv_counts.so", "liballtoallv_backtrace.so", "liballtoallv_exec_timings.so", "liballtoallv_late_arrival.so", "liballtoallv_location.so"}

	mpirunPath, err := exec.LookPath("mpirun")
	if err != nil {
		fmt.Printf("unable to find mpirun: %s\n", err)
		os.Exit(1)
	}
	mpiDir := filepath.Base(mpirunPath)
	if strings.HasSuffix(mpiDir, "bin") {
		mpiDir = filepath.Base(mpiDir)
	}

	for _, lib := range libraries {
		cmdArgs := os.Args[1:]
		libPath := filepath.Join(codeBaseDir, "src", "alltoallv", lib)
		cmdArgs = append([]string{"-x", "LD_PRELOAD=" + libPath}, cmdArgs...)
		cmd := exec.Command(mpirunPath, cmdArgs...)
		cmd.Stderr = stderr
		cmd.Stdout = stdout
		if stdinText != "" {
			// We got data from stdin when the wrapper was invoked so we make sure
			// we pass that data in to the mpirun command.
			stdin, err := cmd.StdinPipe()
			if err != nil {
				fmt.Printf("unable to provide data to the mpirun command: %s", err)
				os.Exit(1)
			}
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, stdinText)
			}()
		}
		newPath := filepath.Join(mpiDir, "bin")
		newPath = newPath + ":" + os.Getenv("PATH")
		newLDpath := filepath.Join(mpiDir, "lib")
		newLDpath = newLDpath + ":" + os.Getenv("LD_LIBRARY_PATH")
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, []string{"PATH=" + newPath, "LD_LIBRARY_PATH=" + newLDpath}...)
		fmt.Printf("Executing: %s %s\n", mpirunPath, cmdArgs)
		err = cmd.Run()
		if err != nil {
			fmt.Printf("command failed: %s\n", err)
			// DO NOT EXIT HERE. We have application that do not terminate cleanly
			// If we have an error we report it and move to the next run
		}
	}
}
