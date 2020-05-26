//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package datafilereader

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

const (
	header1                    = "# Raw counters"
	numberOfRanksMarker        = "Number of ranks: "
	alltoallvCallNumbersMarker = "Alltoallv calls "
	countMarker                = "Count: "
	beginningDataMarker        = "BEGINNING DATA"
	endDataMarker              = "END DATA"
)

func convertCompressedCallListtoIntSlice(str string) ([]int, error) {
	var callIDs []int

	fmt.Printf("Found range: %s\n", str)
	tokens := strings.Split(str, ", ")
	for _, t := range tokens {
		tokens2 := strings.Split(t, "-")
		for _, t2 := range tokens2 {
			n, err := strconv.Atoi(t2)
			if err != nil {
				return callIDs, fmt.Errorf("unable to parse %s", str)
			}
			callIDs = append(callIDs, n)
		}
	}

	return callIDs, nil
}

func GetHeader(reader *bufio.Reader) (int, []int, string, int, int, error) {
	var callIDs []int
	numCalls := 0
	callIDsStr := ""
	//alltoallvCallNumber := 0
	//alltoallvCallStart := 0
	//alltoallvCallEnd := -1
	line := ""
	var err error
	numRanks := 0
	datatypeSize := 0

	// Get the forst line of the header skipping potential empty lines that
	// can be in front of header
	var readerErr error
	for line == "" || line == "\n" {
		line, readerErr = reader.ReadString('\n')
		if readerErr == io.EOF {
			return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
		if readerErr != nil {
			return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
	}

	// Are we at the beginning of a metadata block?
	if !strings.HasPrefix(line, "# Raw") {
		return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, fmt.Errorf("[ERROR] not a header")
	}

	for {
		line, readerErr = reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}

		if strings.HasPrefix(line, "Number of ranke: ") {
			line = strings.ReplaceAll(line, "Number of ranke: ", "")
			line = strings.ReplaceAll(line, "\n", "")
			numRanks, err := strconv.Atoi(line)
			if err != nil {
				log.Println("[ERROR] unable to parse number of ranks")
				return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
			}
		}

		if strings.HasPrefix(line, "Alltoallv calls") {
			line = strings.ReplaceAll(line, "\n", "")
			callRange := strings.ReplaceAll(line, "Alltoallv calls ", "")
			tokens := strings.Split(callRange, "-")
			if len(tokens) == 2 {
				//alltoallvCallStart, err = strconv.Atoi(tokens[0])
				if err != nil {
					log.Println("[ERROR] unable to parse line to get first alltoallv call number")
					return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}
				//alltoallvCallEnd, err = strconv.Atoi(tokens[1])
				if err != nil {
					log.Printf("[ERROR] unable to convert %s to interger: %s", tokens[1], err)
					return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}
			}
			//alltoallvCallNumber = alltoallvCallEnd - alltoallvCallStart + 1 // zero indexed to add one
		}

		if strings.HasPrefix(line, "Count: ") {
			line = strings.ReplaceAll(line, "\n", "")
			strParsing := line
			tokens := strings.Split(line, " - ")
			if len(tokens) > 1 {
				strParsing = tokens[0]
				callIDsStr = tokens[1]
				tokens2 := strings.Split(callIDsStr, " (")
				if len(tokens2) > 1 {
					callIDsStr = tokens2[0]
				}
			}

			strParsing = strings.ReplaceAll(strParsing, "Count: ", "")
			strParsing = strings.ReplaceAll(strParsing, " calls", "")
			numCalls, err = strconv.Atoi(strParsing)
			if err != nil {
				log.Println("[ERROR] unable to parse line to get #s of alltoallv calls")
				return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
			}

			if callIDsStr != "" {
				/*
					tokens := strings.Split(callIDsStr, " ")
					for _, t := range tokens {
						if t == "..." {
							incompleteData = true
						}
						if t != "" && t != "..." { // '...' means that we have a few more calls that are involved but we do not know what they are
							n, err := strconv.Atoi(t)
							if err != nil {
								log.Fatalf("unable to parse '%s' - '%s'\n", callIDsStr, t)
								return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, incompleteData, err
							}
							callIDs = append(callIDs, n)
						}
					}
				*/

				callIDs, err = convertCompressedCallListtoIntSlice(callIDsStr)
				if err != nil {
					return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, err
				}

			}
		}

		// We check for the beginning of the actual data
		if strings.HasPrefix(line, beginningDataMarker) {
			break
		}

		if readerErr == io.EOF {
			return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, readerErr
		}
	}

	/*
		if numCalls != alltoallvCallNumber {
			return numCalls, callIDs, callIDsStr, fmt.Errorf("[ERROR] Inconsistent metadata, number of calls differs (%d vs. %d)", numCalls, alltoallvCallNumber)
		}
	*/

	return numCalls, callIDs, callIDsStr, numRanks, datatypeSize, nil
}

func GetCounters(reader *bufio.Reader) ([]string, error) {
	var callCounters []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return callCounters, err
		}

		if strings.Contains(line, "END DATA") {
			break
		}

		callCounters = append(callCounters, line)
	}

	return callCounters, nil
}
