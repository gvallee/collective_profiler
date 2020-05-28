//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package analyzer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
)

type analyzer struct {
	realEndpoints map[int]int
	ranksComm     map[int]string
	InputFile     string
}

type srCountAnalyzer struct {
	RecvCountsFile     string
	SendCountsFile     string
	RecvCountsAnalyzer *analyzer
	SendCountsAnalyzer *analyzer
}

type rankData struct {
	numZeroMsgs   int
	ranksRealComm int
	totalMsgs     int
}

type OpTimeStat struct {
	CallNum     int
	MinTime     float64
	RankMinTime int
	MaxTime     float64
	RankMaxTime int
}

func CreateAnalyzer() *analyzer {
	a := new(analyzer)
	a.realEndpoints = make(map[int]int)
	a.ranksComm = make(map[int]string)
	return a
}

func CreateSRCountsAnalyzer(sendCountsFile string, recvCountsFile string) *srCountAnalyzer {
	a := new(srCountAnalyzer)
	a.SendCountsFile = sendCountsFile
	a.RecvCountsFile = recvCountsFile
	a.RecvCountsAnalyzer = CreateAnalyzer()
	a.SendCountsAnalyzer = CreateAnalyzer()
	a.RecvCountsAnalyzer.InputFile = recvCountsFile
	a.SendCountsAnalyzer.InputFile = sendCountsFile
	return a
}

func (a *analyzer) Finalize() error {
	return nil
}

func (a *analyzer) resetStats() {
	a.realEndpoints = make(map[int]int)
	a.ranksComm = make(map[int]string)

	if len(a.realEndpoints) > 0 || len(a.ranksComm) > 0 {
		log.Fatalf("Map is not reset")
	}
}

// parseForRanges parses the string and check for either a single value (a rank) or a range of ranks
// The new rank is added appropriately
func checkForRanges(str string, rankString string) (string, error) {
	rank, err := strconv.Atoi(rankString)
	if err != nil {
		return "", err
	}

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

func (a *analyzer) handleRankCounters(rankString string, d rankData) error {
	finalStr := ""
	if a.ranksComm[d.ranksRealComm] == "" {
		finalStr = rankString
	} else {
		tokens := strings.Split(a.ranksComm[d.ranksRealComm], ", ")
		if len(tokens) > 1 {
			s, err := checkForRanges(tokens[len(tokens)-1], rankString)
			if err != nil {
				return err
			}
			tokens[len(tokens)-1] = s
			finalStr = strings.Join(tokens, ", ")
		} else {
			var err error
			finalStr, err = checkForRanges(a.ranksComm[d.ranksRealComm], rankString)
			if err != nil {
				return err
			}
		}
	}
	a.ranksComm[d.ranksRealComm] = finalStr

	if _, ok := a.realEndpoints[d.ranksRealComm]; ok {
		a.realEndpoints[d.ranksRealComm]++
		//fmt.Printf("Record for communications with %d ranks is now: %d\n", d.ranksRealComm, a.realDests[d.ranksRealComm])
	} else {
		//fmt.Printf("Rank communicating with %d other ranks and adding a new record about communications with %d ranks\n", d.ranksRealComm, d.ranksRealComm)
		a.realEndpoints[d.ranksRealComm] = 1
	}
	return nil
}

func (a *analyzer) parseLine(lineNum int, line string) (bool, error) {
	var d rankData
	var ranks []string

	// Each line is of the form 'Rank(s) <list of ranks, space separated>: <counters>
	entries := strings.Split(line, ": ")
	if len(entries) > 1 {
		line = entries[1]
		ranks = strings.Split(strings.ReplaceAll(entries[0], "Rank(s) ", ""), " ")
	}

	tokens := strings.Split(line, " ")
	if len(tokens) <= 1 {
		// Not an actual line with counters
		return false, nil
	}

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

	for _, rankStr := range ranks {
		err := a.handleRankCounters(rankStr, d)
		if err != nil {
			return false, err
		}
	}

	//fmt.Printf("rank %d communicates with %d other ranks and has %d zero counts for a total of %d msgs\n", rank, d.ranksRealComm, d.numZeroMsgs, d.totalMsgs)

	return true, nil
}

func (a *analyzer) Parse() error {
	//parsing := true // used to track if we are actively parsing data or just skipping text
	rank := 0
	alltoallvCallNumber := 0
	alltoallvCallStart := 0
	alltoallvCallEnd := -1
	lineNumber := 1
	numCalls := 0
	//callIDs := ""

	file, err := os.Open(a.InputFile)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", a.InputFile, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		log.Println("Getting header...")
		_, numCalls, _, callIDsStr, _, _, readerErr := datafilereader.GetHeader(reader)
		if readerErr != nil && readerErr != io.EOF {
			log.Printf("[ERROR] unable to read header: %s", readerErr)
			return readerErr
		}
		if readerErr == io.EOF {
			break
		}
		log.Printf("-> Number of alltoallv calls: %d\n", numCalls)

		// After successfully reading a new header, we know we are about to read
		// a bunch of new data so we re-init a few things
		rank = 0
		alltoallvCallNumber++
		a.resetStats()

		// Read all the data for the call(s)
		log.Println("Reading counters...")
		line := ""
		for {
			line, readerErr = reader.ReadString('\n')
			if readerErr != nil && readerErr != io.EOF {
				return readerErr
			}

			if strings.HasPrefix(line, "END DATA") {
				for key, _ := range a.realEndpoints {
					fmt.Printf("alltoallv call(s) #%s (%d calls): %d ranks (%s) communicate with %d other ranks \n", callIDsStr, numCalls, a.realEndpoints[key], a.ranksComm[key], key)
				}
				break
			}

			res, err := a.parseLine(lineNumber, line)
			if err != nil {
				return err
			}
			if res {
				rank++
			}

			if readerErr == io.EOF {
				break
			}
		}
		if readerErr == io.EOF {
			break
		}
	}

	/*
		if alltoallvCallEnd == -1 {
			log.Println("[WARN] Metadata is incomplete, unable to validate consistency of counters")
		} else {
	*/
	expectedNumCalls := alltoallvCallEnd - alltoallvCallStart + 1 // 0 indexed so we need to add 1
	if numCalls != expectedNumCalls {
		log.Printf("[ERROR] Metadata specifies %d calls but we extracted %d calls", expectedNumCalls, numCalls)
	}
	//}

	return nil
}

func (a *srCountAnalyzer) SRCountsParse() error {
	err := a.SendCountsAnalyzer.Parse()
	if err != nil {
		return err
	}

	err = a.RecvCountsAnalyzer.Parse()
	if err != nil {
		return err
	}

	return nil
}

func GetCallsTimings(reader *bufio.Reader) ([]OpTimeStat, error) {
	var stats []OpTimeStat
	curCall := ""
	curRank := 0

	for {
		line, readerErr := reader.ReadString('\n')
		if readerErr != nil && readerErr != io.EOF {
			return stats, readerErr
		}

		if line == "" {
			if readerErr == io.EOF {
				break
			}
			continue
		}

		tokens := strings.Split(line, "\t")
		if len(tokens) != 2 {
			fmt.Printf("Wrong format: '%s'\n", line)
			return stats, fmt.Errorf("unable to parse %s", line)
		}
		timing, err := strconv.ParseFloat(strings.TrimRight(tokens[1], "\n"), 64)
		if err != nil {
			return stats, err
		}

		if curCall != tokens[0] {
			// We start to parse data for a new alltoallv call
			curRank = 0
			var newStats OpTimeStat
			newStats.CallNum, err = strconv.Atoi(tokens[0])
			if err != nil {
				return stats, err
			}
			newStats.MaxTime = timing
			newStats.MinTime = timing
			newStats.RankMaxTime = curRank
			newStats.RankMinTime = curRank
			stats = append(stats, newStats)
			curCall = tokens[0]
		} else {
			curRank++
			if stats[len(stats)-1].MaxTime < timing {
				stats[len(stats)-1].MaxTime = timing
				stats[len(stats)-1].RankMaxTime = curRank
			}

			if stats[len(stats)-1].MinTime > timing {
				stats[len(stats)-1].MinTime = timing
				stats[len(stats)-1].RankMinTime = curRank
			}
		}

		if readerErr == io.EOF {
			break
		}
	}

	return stats, nil
}
