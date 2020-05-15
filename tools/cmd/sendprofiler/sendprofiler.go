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
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type analyzer struct {
	realDests map[int]int
	ranksComm map[int]string
}

type rankData struct {
	numZeroMsgs   int
	ranksRealComm int
	totalMsgs     int
}

func createAnalyzer() *analyzer {
	a := new(analyzer)
	a.realDests = make(map[int]int)
	a.ranksComm = make(map[int]string)
	return a
}

func (a *analyzer) finalize() error {
	return nil
}

func (a *analyzer) resetStats() {
	a.realDests = make(map[int]int)
	a.ranksComm = make(map[int]string)

	if len(a.realDests) > 0 || len(a.ranksComm) > 0 {
		log.Fatalf("Map is not reset")
	}
}

// parseForRanges parses the string and check for either a single value (a rank) or a range of ranks
// The new rank is added appropriately
func checkForRanges(str string, rank int, rankString string) (string, error) {
	tokens := strings.Split(str, "-")
	if len(tokens) > 1 {
		// We have a range
		lastRank, err := strconv.Atoi(tokens[1])
		if err != nil {
			return "", err
		}
		if lastRank+1 == rank {
			tokens[1] = rankString
			return strings.Join(tokens, "-"), nil
		} else {
			return str + ", " + rankString, nil
		}
	} else {
		// no range of ranks at the end of the string
		n, err := strconv.Atoi(str)
		if err != nil {
			return "", err
		}
		if n+1 == rank {
			return str + "-" + rankString, nil
		} else {
			return str + ", " + rankString, nil
		}
	}
}

func (a *analyzer) parseLine(rank int, lineNum int, line string) (bool, error) {
	var d rankData

	tokens := strings.Split(line, " ")
	if len(tokens) <= 1 {
		// Not an actual line with counters
		return false, nil
	}

	/*
		if rank < 5 || rank >= 1024 {
			fmt.Printf("I am rank %d at line %d and my counters are %s\n", rank, lineNum+1, line)
		}
	*/
	for _, word := range tokens {
		d.totalMsgs++
		num, err := strconv.Atoi(word)
		if err == nil {
			if num == 0 {
				d.numZeroMsgs++
			} else {
				//fmt.Printf("Rank %d has a non-zero count on line %d: count is at %d\n", rank, lineNum, num)
				d.ranksRealComm++
			}
		}
	}

	//fmt.Printf("rank %d communicates with %d other ranks and has %d zero counts for a total of %d msgs\n", rank, d.ranksRealComm, d.numZeroMsgs, d.totalMsgs)

	rankString := strconv.Itoa(rank)

	finalStr := ""
	if a.ranksComm[d.ranksRealComm] == "" {
		finalStr = rankString
	} else {
		tokens := strings.Split(a.ranksComm[d.ranksRealComm], ", ")
		if len(tokens) > 1 {
			s, err := checkForRanges(tokens[len(tokens)-1], rank, rankString)
			if err != nil {
				return false, err
			}
			tokens[len(tokens)-1] = s
			finalStr = strings.Join(tokens, ", ")
		} else {
			var err error
			finalStr, err = checkForRanges(a.ranksComm[d.ranksRealComm], rank, rankString)
			if err != nil {
				return false, err
			}
		}
	}
	a.ranksComm[d.ranksRealComm] = finalStr

	if _, ok := a.realDests[d.ranksRealComm]; ok {
		a.realDests[d.ranksRealComm]++
		//fmt.Printf("Record for communications with %d ranks is now: %d\n", d.ranksRealComm, a.realDests[d.ranksRealComm])
	} else {
		//fmt.Printf("Rank communicating with %d other ranks and adding a new record about communications with %d ranks\n", d.ranksRealComm, d.ranksRealComm)
		a.realDests[d.ranksRealComm] = 1
	}

	return true, nil
}

func analyzeSendCounts(input string, outputDir string) error {
	parsing := true // used to track if we are actively parsing data or just skipping text
	a := createAnalyzer()
	rank := 0
	alltoallvCallNumber := 0
	alltoallvCallStart := 0
	alltoallvCallEnd := -1
	lineNumber := 1
	numCalls := 0
	callIDs := ""

	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", input, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasPrefix(line, "# Raw") {
			parsing = false
			numCalls = 0
			callIDs = ""
		}

		if strings.HasPrefix(line, "Alltoallv calls") {
			line = strings.ReplaceAll(line, "\n", "")
			callRange := strings.ReplaceAll(line, "Alltoallv calls ", "")
			tokens := strings.Split(callRange, "-")
			if len(tokens) == 2 {
				alltoallvCallStart, err = strconv.Atoi(tokens[0])
				if err != nil {
					log.Println("[ERROR] unable to parse line to get first alltoallv call number")
					return err
				}
				alltoallvCallEnd, err = strconv.Atoi(tokens[1])
				if err != nil {
					return err
				}
			}
		}

		if strings.HasPrefix(line, "Count: ") {
			line = strings.ReplaceAll(line, "\n", "")
			strParsing := line
			tokens := strings.Split(line, " - ")
			if len(tokens) > 1 {
				strParsing = tokens[0]
				callIDs = tokens[1]
				tokens2 := strings.Split(callIDs, " (")
				if len(tokens2) > 1 {
					callIDs = tokens2[0]
				}
			}
			strParsing = strings.ReplaceAll(strParsing, "Count: ", "")
			strParsing = strings.ReplaceAll(strParsing, " calls", "")
			numCalls, err = strconv.Atoi(strParsing)
			if err != nil {
				log.Println("[ERROR] unable to parse line to get #s of alltoallv calls")
				return err
			}
		}

		if strings.HasPrefix(line, "END DATA") {
			for key, _ := range a.realDests {
				fmt.Printf("alltoallv call(s) #%s (%d calls): %d ranks (%s) communicate with %d other ranks \n", callIDs, numCalls, a.realDests[key], a.ranksComm[key], key)
			}
			parsing = false
		}

		if line != "" && parsing == true {
			res, err := a.parseLine(rank, lineNumber, line)
			if err != nil {
				return err
			}
			if res {
				rank++
			}
		}

		if strings.HasPrefix(line, "BEGINNING") {
			parsing = true
			rank = 0
			alltoallvCallNumber++
			a.resetStats()
		}

		lineNumber++
	}

	if alltoallvCallEnd == -1 {
		log.Println("[WARN] Metadata is incomplete, unable to validate consistency of counters")
	} else {
		expectedNumCalls := alltoallvCallEnd - alltoallvCallStart + 1 // 0 indexed so we need to add 1
		if numCalls != expectedNumCalls {
			log.Printf("[ERROR] Metadata specifies %d calls but we extracted %d calls", expectedNumCalls, numCalls)
		}
	}

	a.finalize()

	return nil
}

func main() {
	file := flag.String("file", "", "Path to the file from which we want to extract counters")
	outputDir := flag.String("output-dir", "", "Where the output files will be stored")

	flag.Parse()

	if *file == "" || *outputDir == "" {
		log.Fatalf("undefined input file or output directory")
	}

	err := analyzeSendCounts(*file, *outputDir)
	if err != nil {
		log.Fatalf("[ERROR] Impossible to analyze send counters: %s", err)
	}
}
