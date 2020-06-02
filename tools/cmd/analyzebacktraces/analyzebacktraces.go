//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
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
	"sort"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/go_util/pkg/util"
)

func printCallerInfo(info analyzer.CallerInfo) {
	fmt.Printf("Binary: %s\n", info.Binary)
	fmt.Println("Addresses:")
	for _, a := range info.Addresses {
		fmt.Printf("%s\n", a)
	}
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")

	flag.Parse()

	logFile := util.OpenLogFile("alltoallv", "analyzebacktraces")
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	callers, err := analyzer.GetCallersFromBacktraces(*dir)
	if err != nil {
		log.Fatalf("unable to get callers from backtraces: %s", err)
	}

	if len(callers.Callers) == 0 {
		log.Printf("No caller information")
		os.Exit(1)
	} else {
		log.Printf("Found %d distinct caller information", len(callers.Callers))
	}

	// For each element in callers, we try to translate the address to a file/line number
	fmt.Printf("Found %d unique caller\n", len(callers.Callers))
	numCaller := 0
	for _, c := range callers.Callers {
		var codeInfo []string
		addr2lineBin, err := exec.LookPath("addr2line")
		if err != nil {
			log.Fatalf("unable to locate addr2line binary: %s", err)
		}

		if len(c.Addresses) == 0 {
			log.Printf("No address for caller #%d", numCaller)
		} else {
			log.Printf("Looking up %d addresses", len(c.Addresses))
		}
		fmt.Printf("Trying to translate %d addresses\n", len(c.Addresses))
		for _, a := range c.Addresses {
			log.Printf("Executing: %s -e %s %s\n", addr2lineBin, c.Binary, a)
			cmd := exec.Command(addr2lineBin, "-e", c.Binary, a)
			var output bytes.Buffer
			cmd.Stdout = &output
			err := cmd.Run()
			if err != nil {
				log.Fatal("unable to execute command")
			}

			if strings.HasPrefix(output.String(), "??:0") || strings.HasPrefix(output.String(), "??:?") {
				continue
			}
			fmt.Printf("%d - Reference to %s\n", numCaller, output.String())
			codeInfo = append(codeInfo, output.String())
		}

		if len(codeInfo) == 0 {
			fmt.Println("[WARN] unable to find any address from the caller's backtrace that can point to code")
			printCallerInfo(c)
		} else {
			fmt.Println("[INFO] Found caller's code")
		}

		// Save the results
		sort.Ints(c.Calls)
		header := "Alltoallv calls: " + notation.CompressIntArray(c.Calls) + "\n"

		str := ""
		for _, ci := range codeInfo {
			str += ci
		}
		if str != "" {
			//str = notation.CompressIntArray(c.Calls) + "\n" + str
			resultFilename := filepath.Join(*dir, fmt.Sprintf("alltoallv_caller_%d.txt", numCaller))
			log.Printf("Saving results in %s (%d elements)\n", resultFilename, len(codeInfo))
			ioutil.WriteFile(resultFilename, []byte(header+str), 0755)
		} else {
			log.Printf("unable to find information about caller %d\n", numCaller)
		}
		numCaller++
	}
}
