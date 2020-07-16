//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package profiler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/counts"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"
)

func containsCall(callNum int, calls []int) bool {
	for i := 0; i < len(calls); i++ {
		if calls[i] == callNum {
			return true
		}
	}
	return false
}

func GetCallRankData(sendCountersFile string, recvCountersFile string, callNum int, rank int) (int, int, error) {
	sendCounters, sendDatatypeSize, _, err := datafilereader.ReadCallRankCounters([]string{sendCountersFile}, rank, callNum)
	if err != nil {
		return 0, 0, err
	}
	recvCounters, recvDatatypeSize, _, err := datafilereader.ReadCallRankCounters([]string{recvCountersFile}, rank, callNum)
	if err != nil {
		return 0, 0, err
	}

	sendCounters = strings.TrimRight(sendCounters, "\n")
	recvCounters = strings.TrimRight(recvCounters, "\n")

	// We parse the send counters to know how much data is being sent
	sendSum := 0
	tokens := strings.Split(sendCounters, " ")
	for _, t := range tokens {
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			return 0, 0, err
		}
		sendSum += n
	}
	sendSum = sendSum * sendDatatypeSize

	// We parse the recv counters to know how much data is being received
	recvSum := 0
	tokens = strings.Split(recvCounters, " ")
	for _, t := range tokens {
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			return 0, 0, err
		}
		recvSum += n
	}
	recvSum = recvSum * recvDatatypeSize

	return sendSum, recvSum, nil
}

// AnalyzeSubCommsResults go through the results and analyzes results specific
// to sub-communicators cases
func AnalyzeSubCommsResults(dir string, stats map[int]counts.Stats) error {
	numPatterns := -1
	numNtoNPatterns := -1
	num1toNPatterns := -1
	numNto1Patterns := -1
	var referencePatterns counts.GlobalPatterns

	// At the moment, we do a very basic analysis: are the patterns the same on all sub-communicators?
	for _, rankStats := range stats {
		if numPatterns == -1 {
			numPatterns = len(rankStats.Patterns.AllPatterns)
			numNto1Patterns = len(rankStats.Patterns.NToOne)
			numNtoNPatterns = len(rankStats.Patterns.NToN)
			num1toNPatterns = len(rankStats.Patterns.OneToN)
			referencePatterns = rankStats.Patterns
			continue
		}

		if numPatterns != len(rankStats.Patterns.AllPatterns) ||
			numNto1Patterns != len(rankStats.Patterns.NToOne) ||
			numNtoNPatterns != len(rankStats.Patterns.NToN) ||
			num1toNPatterns != len(rankStats.Patterns.OneToN) {
			return nil
		}

		if !counts.SamePatterns(referencePatterns, rankStats.Patterns) {
			/*
				fmt.Println("Patterns differ:")
				displayPatterns(referencePatterns.AllPatterns)
				fmt.Printf("\n")
				displayPatterns(rankStats.Patterns.AllPatterns)
			*/
			return nil
		}
	}

	// If we get there it means all ranks, i.e., sub-communicators have the same amount of patterns
	log.Println("All patterns on all sub-communicators are similar")
	multicommHighlightFile := filepath.Join(dir, datafilereader.MulticommHighlightFilePrefix+".md")
	fd, err := os.OpenFile(multicommHighlightFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.WriteString("Alltoallv on sub-communicators detected.\n\n# Patterns summary\n\n")
	if err != nil {
		return err
	}

	var ranks []int
	for r := range stats {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)

	if len(stats[ranks[0]].Patterns.NToN) > 0 {
		err := counts.WriteSubcommNtoNPatterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	if len(stats[ranks[0]].Patterns.OneToN) > 0 {
		err := counts.WriteSubcomm1toNPatterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	if len(stats[ranks[0]].Patterns.NToOne) > 0 {
		err := counts.WriteSubcommNto1Patterns(fd, ranks, stats)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n## All 0 counts pattern; no data exchanged\n\n")
	if err != nil {
		return err
	}
	for _, rank := range ranks {
		if len(stats[rank].Patterns.Empty) > 0 {
			_, err = fd.WriteString(fmt.Sprintf("-> Sub-communicator led by rank %d: %d/%d alltoallv calls\n", rank, len(stats[rank].Patterns.Empty), stats[rank].TotalNumCalls))
			if err != nil {
				return err
			}
		}
	}

	// For now we save the bins' data separately because we do not have a good way at the moment
	// to mix bins and patterns (bins are specific to a count file, not a call; we could change that
	// but it would take time).
	_, err = fd.WriteString("\n# Counts analysis\n\n")
	if err != nil {
		return err
	}
	for _, rank := range ranks {
		_, err := fd.WriteString(fmt.Sprintf("-> Sub-communicator led by rank %d:\n", rank))
		if err != nil {
			return err
		}
		for _, b := range stats[rank].Bins {
			if b.Max != -1 {
				_, err := fd.WriteString(fmt.Sprintf("\t%d of the messages are of size between %d and %d bytes\n", b.Size, b.Min, b.Max-1))
				if err != nil {
					return err
				}
			} else {
				_, err := fd.WriteString(fmt.Sprintf("\t%d of messages are larger or equal of %d bytes\n", b.Size, b.Min))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
