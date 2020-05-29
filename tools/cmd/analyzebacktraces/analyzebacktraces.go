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
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/analyzer"
	"github.com/gvallee/go_util/pkg/util"
)

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

	// For each element in callers, we try to translate the address to a file/line number
	numCaller := 0
	var codeInfo []string
	for _, c := range callers {
		addr2lineBin, err := exec.LookPath("addr2line")
		if err != nil {
			log.Fatalf("unable to locate addr2line binary: %s", err)
		}

		for _, a := range c.Addresses {
			cmd := exec.Command(addr2lineBin, "-e", c.Binary, a)
			var output bytes.Buffer
			cmd.Stdout = &output
			err := cmd.Run()
			if err != nil {
				log.Fatal("unable to execute command")
			}

			if strings.HasPrefix(output.String(), "??:0") {
				continue
			}
			codeInfo = append(codeInfo, output.String())
		}

		// Save the results
		resultFilename := filepath.Join(*dir, fmt.Sprintf("alltoallv_caller_%d.txt", numCaller))
		str := ""
		for _, ci := range codeInfo {
			str += ci + "\n"
		}
		ioutil.WriteFile(resultFilename, []byte(str), 0755)
		numCaller++
	}
}
