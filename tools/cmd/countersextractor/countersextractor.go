//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func saveSendCounterLine() {

}

func saveRecvCounterLine() {

}

func lineContextFromLine(line string) string {
	if strings.HasPrefix(line, "## Data sent per rank") {
		return "SEND"
	}
	if strings.HasPrefix(line, "## Data received per rank") {
		return "RECV"
	}
	return ""
}

func readAndParseFileLinebyLine(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("unable to open %f: %w", filepath, err)
	}
	defer file.Close()

	var ctxt string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		potentialCtxt := getContextFromLine(line)
		if potentialCtxt != "" {
			ctxt = potentialCtxt
		}
		if lineIsCounterData(line) {
			if ctxt == "SEND" {
				err := saveSendCounterLine(line)
			} else {
				err := saveRecvCounterLine(line)
			}
			if err != nil {
				return fmt.Errorf("unable to save line: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner failed while reading the file: %w", err)
	}
}

func main() {
	file := flag.String("file", "", "Path to the file from which we want to extract counters")
	outputDir := flag.String("output-dir", "", "Where the output files will be stored")

	flag.Parse()

	if file == "" || outputDir == "" {
		log.Fatalf("undefined input file or output directory")
	}

	readAndParseFileLinebyLine(file, outputDir)
}
