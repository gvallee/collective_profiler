//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"fmt"
	"log"
)

// SamePatterns compare two patterns
func SamePatterns(patterns1, patterns2 GlobalPatterns) bool {
	return sameListOfPatterns(patterns1.AllPatterns, patterns2.AllPatterns)
}

func displayPatterns(pattern []*CallPattern) {
	for _, p := range pattern {
		for numPeers, numRanks := range p.Send {
			fmt.Printf("%d ranks are sending to %d other ranks\n", numRanks, numPeers)
		}
		for numPeers, numRanks := range p.Recv {
			fmt.Printf("%d ranks are receiving from %d other ranks\n", numRanks, numPeers)
		}
	}
}

// patternIsInList checks whether a given pattern is in a list of patterns. If so, it returns the
// number of alltoallv calls that have the pattern, otherwise it returns 0
func patternIsInList(numPeers int, numRanks int, ctx string, patterns []*CallPattern) int {
	for _, p := range patterns {
		if ctx == "SEND" {
			for numP, numR := range p.Send {
				if numP == numP && numRanks == numR {
					return p.Count
				}
			}
		} else {
			for numP, numR := range p.Recv {
				if numP == numP && numRanks == numR {
					return p.Count
				}
			}
		}
	}
	return 0
}

func sameListOfPatterns(patterns1, patterns2 []*CallPattern) bool {
	// reflect.DeepEqual cannot be used here

	// Compare send counts
	for _, p1 := range patterns1 {
		for numPeers, numRanks := range p1.Send {
			count := patternIsInList(numPeers, numRanks, "SEND", patterns2)
			if count == 0 {
				return false
			}
			if p1.Count != count {
				log.Printf("Send counts differ: %d vs. %d", p1.Count, count)
			}
		}
	}

	// Compare recv counts
	for _, p1 := range patterns1 {
		for numPeers, numRanks := range p1.Recv {
			count := patternIsInList(numPeers, numRanks, "RECV", patterns2)
			if count == 0 {
				return false
			}
			if p1.Count != count {
				log.Printf("Recv counts differ: %d vs. %d", p1.Count, count)
			}
		}
	}

	return true
}

func noPatternsSummary(cs Stats) bool {
	if len(cs.Patterns.OneToN) != 0 {
		return false
	}

	if len(cs.Patterns.NToOne) != 0 {
		return false
	}

	if len(cs.Patterns.NToN) != 0 {
		return false
	}

	return true
}
