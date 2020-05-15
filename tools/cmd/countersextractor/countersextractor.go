//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type extractor struct {
	numSendCountersBlocks int
	numRecvCountersBlocks int
	sendCountersFile      *os.File
	recvCountersFile      *os.File
}

func createExtractor(outputDir string) *extractor {
	e := new(extractor)

	e.numRecvCountersBlocks = 0
	e.numSendCountersBlocks = 0

	pathToSendCountersFile := filepath.Join(outputDir, "send_counters.txt")
	pathToRecvCountersFile := filepath.Join(outputDir, "recv_counters.txt")

	var err error
	fmt.Printf("Writing to %s and %s\n", pathToRecvCountersFile, pathToSendCountersFile)
	e.sendCountersFile, err = os.OpenFile(pathToSendCountersFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("[ERROR] unable to open %s: %s", pathToSendCountersFile, err)
		return nil
	}
	e.recvCountersFile, err = os.OpenFile(pathToRecvCountersFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("[ERROR] unable to open %s: %s", pathToRecvCountersFile, err)
		return nil
	}

	return e
}

func (e *extractor) finalize() error {
	e.recvCountersFile.Sync()
	e.sendCountersFile.Sync()

	e.recvCountersFile.Close()
	e.sendCountersFile.Close()

	return nil
}

func (e *extractor) saveSendCounterLine(line string) error {
	if e == nil || e.sendCountersFile == nil {
		return fmt.Errorf("invalid extractor")
	}
	_, err := e.sendCountersFile.WriteString(line + "\n")
	if err != nil {
		log.Printf("[ERROR] unable to write to file: %s", err)
		return err
	}
	return nil
}

func (e *extractor) saveRecvCounterLine(line string) error {
	if e == nil || e.recvCountersFile == nil {
		return fmt.Errorf("invalid extractor")
	}
	_, err := e.recvCountersFile.WriteString(line + "\n")
	if err != nil {
		log.Printf("[ERROR] unable to write to file: %s", err)
		return err
	}
	return nil
}

func lineIsCounterData(line string) bool {
	if line == "" {
		return false
	}

	if strings.HasPrefix(line, "#") {
		return false
	}

	if strings.HasPrefix(line, "Total") {
		return false
	}

	if strings.HasPrefix(line, "Rank") {
		return false
	}

	return true
}

func (e *extractor) getContextFromLine(line string) string {
	if strings.HasPrefix(line, "## Data sent per rank") {
		e.numSendCountersBlocks++
		e.sendCountersFile.WriteString("\n")
		return "SEND"
	}
	if strings.HasPrefix(line, "## Data received per rank") {
		e.numRecvCountersBlocks++
		e.recvCountersFile.WriteString("\n")
		return "RECV"
	}
	return ""
}

func readAndParseFileLinebyLine(filepath string, outputDir string) error {
	e := createExtractor(outputDir)

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", filepath, err)
	}
	defer file.Close()

	var ctxt string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		potentialCtxt := e.getContextFromLine(line)
		if potentialCtxt != "" {
			ctxt = potentialCtxt
		}
		if lineIsCounterData(line) {
			if ctxt == "SEND" {
				err := e.saveSendCounterLine(line)
				if err != nil {
					return err
				}
			} else {
				err := e.saveRecvCounterLine(line)
				if err != nil {
					return err
				}
			}
			if err != nil {
				return fmt.Errorf("unable to save line: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner failed while reading the file: %w", err)
	}

	e.finalize()

	fmt.Printf("Number of send counters blocks read: %d\n", e.numSendCountersBlocks)
	fmt.Printf("Number of recv counters blocks read: %d\n", e.numRecvCountersBlocks)

	return nil
}

func main() {
	file := flag.String("file", "", "Path to the file from which we want to extract counters")
	outputDir := flag.String("output-dir", "", "Where the output files will be stored")

	flag.Parse()

	if *file == "" || *outputDir == "" {
		log.Fatalf("undefined input file or output directory")
	}

	err := readAndParseFileLinebyLine(*file, *outputDir)
	if err != nil {
		log.Fatalf("unable to parse input file: %s", err)
	}
}
