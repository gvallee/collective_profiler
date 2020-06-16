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
	inputdir := flag.String("input-dir", "", "Where all the data is")
	outputdir := flag.String("output-dir", "", "Where the result files will be stored")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s analyses the data gathered while executing the application with liballtoallv_backtrace.so (or equivalent shared library gathering backtrace data).", cmdName)
		fmt.Println("The lirbary saves all the backtraces in a  'backtraces' directory and within it, files are named 'backtrace_rank<RANK>_call<CALLID>.md'.")
		fmt.Println("The files are on a rank basis because any rank can be rank 0 on a sub-communicator used for the alltoallv operation.")
		fmt.Println("The command parses all these files and report unique backtrace, i.e., unique contexts in which alltoallv is called.")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	callers, err := analyzer.GetCallersFromBacktraces(*inputdir)
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
			resultFilename := filepath.Join(*outputdir, fmt.Sprintf("alltoallv_caller_%d.txt", numCaller))
			fmt.Printf("Saving results in %s (%d elements)\n", resultFilename, len(codeInfo))
			ioutil.WriteFile(resultFilename, []byte(header+str), 0755)
		} else {
			log.Printf("unable to find information about caller %d\n", numCaller)
		}
		numCaller++
	}
}
