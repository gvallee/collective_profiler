//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"fmt"
	"os"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

// SaveBins writes the data of all the bins into output file. The output files
// are created in a target output directory.
func SaveBins(dir string, jobid, rank int, bins []Bin) error {
	for _, b := range bins {
		outputFile := getBinOutputFile(dir, jobid, rank, b)
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("unable to create file %s: %s", outputFile, err)
		}

		_, err = f.WriteString(fmt.Sprintf("%d\n", b.Size))
		if err != nil {
			return fmt.Errorf("unable to write bin to file: %s", err)
		}
	}
	return nil
}

func WriteSubcommNtoNPatterns(fd *os.File, ranks []int, stats map[int]Stats) error {
	_, err := fd.WriteString("## N to n patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.NToN {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.NToN {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

func WriteSubcomm1toNPatterns(fd *os.File, ranks []int, stats map[int]Stats) error {
	_, err := fd.WriteString("## 1 to n patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.OneToN {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.OneToN {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

func WriteSubcommNto1Patterns(fd *os.File, ranks []int, stats map[int]Stats) error {
	_, err := fd.WriteString("## N to 1 patterns\n\n")
	if err != nil {
		return err
	}

	// Print the pattern, which is the same for all ranks if we reach this function
	_, err = fd.WriteString("\n### Pattern(s) description\n\n")
	if err != nil {
		return err
	}
	for _, p := range stats[ranks[0]].Patterns.NToOne {
		err := writeDataPatternToFile(fd, p)
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n\n### Sub-communicator(s) information\n\n")
	if err != nil {
		return err
	}
	for _, r := range ranks {
		// Print metadata for the subcomm
		_, err := fd.WriteString(fmt.Sprintf("-> Subcommunicator led by rank %d:\n", r))
		if err != nil {
			return err
		}
		num := 0
		for _, p := range stats[r].Patterns.NToOne {
			_, err := fd.WriteString(fmt.Sprintf("\tpattern #%d: %d/%d alltoallv calls\n", num, p.Count, stats[r].TotalNumCalls))
			if err != nil {
				return err
			}
			num++
		}
	}

	return nil
}

func SaveStats(info OutputFileInfo, cs Stats, numCalls int, sizeThreshold int) error {
	_, err := info.defaultFd.WriteString(fmt.Sprintf("Total number of alltoallv calls: %d\n\n", numCalls))
	if err != nil {
		return err
	}

	err = writeDatatypeToFile(info.defaultFd, numCalls, cs.DatatypesSend, cs.DatatypesRecv)
	if err != nil {
		return err
	}

	err = writeCommunicatorSizesToFile(info.defaultFd, numCalls, cs.CommSizes)
	if err != nil {
		return err
	}

	err = writeCountStatsToFile(info.defaultFd, numCalls, sizeThreshold, cs)
	if err != nil {
		return err
	}

	_, err = info.patternsFd.WriteString("# Patterns\n")
	if err != nil {
		return err
	}
	num := 0
	for _, cp := range cs.Patterns.AllPatterns {
		err = writePatternsToFile(info.patternsFd, num, numCalls, cp)
		if err != nil {
			return err
		}
		num++
	}

	if !noPatternsSummary(cs) {
		if len(cs.Patterns.OneToN) != 0 {
			_, err := info.patternsSummaryFd.WriteString("# 1 to N patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.OneToN {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
				num++
			}
		}

		if len(cs.Patterns.NToOne) != 0 {
			_, err := info.patternsSummaryFd.WriteString("\n# N to 1 patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.NToOne {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
			}
		}

		if len(cs.Patterns.NToN) != 0 {
			_, err := info.patternsSummaryFd.WriteString("\n# N to n patterns\n\n")
			if err != nil {
				return err
			}
			num = 0
			for _, cp := range cs.Patterns.NToN {
				err = writePatternsToFile(info.patternsSummaryFd, num, numCalls, cp)
				if err != nil {
					return err
				}
			}
		}
	} else {
		_, err = info.patternsSummaryFd.WriteString("Nothing special detected; no summary")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDataPatternToFile(fd *os.File, cp *CallPattern) error {
	for sendTo, n := range cp.Send {
		_, err := fd.WriteString(fmt.Sprintf("%d ranks sent to %d other ranks\n", n, sendTo))
		if err != nil {
			return err
		}
	}
	for recvFrom, n := range cp.Recv {
		_, err := fd.WriteString(fmt.Sprintf("%d ranks recv'd from %d other ranks\n", n, recvFrom))
		if err != nil {
			return err
		}
	}
	return nil
}

func writePatternsToFile(fd *os.File, num int, totalNumCalls int, cp *CallPattern) error {
	_, err := fd.WriteString(fmt.Sprintf("## Pattern #%d (%d/%d alltoallv calls)\n\n", num, cp.Count, totalNumCalls))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("Alltoallv calls: %s\n", notation.CompressIntArray(cp.Calls)))
	if err != nil {
		return err
	}

	err = writeDataPatternToFile(fd, cp)
	if err != nil {
		return err
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func writeDatatypeToFile(fd *os.File, numCalls int, datatypesSend map[int]int, datatypesRecv map[int]int) error {
	_, err := fd.WriteString("# Datatypes\n\n")
	if err != nil {
		return err
	}
	for datatypeSize, n := range datatypesSend {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while sending data\n", n, numCalls, datatypeSize))
		if err != nil {
			return err
		}
	}
	for datatypeSize, n := range datatypesRecv {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while receiving data\n", n, numCalls, datatypeSize))
		if err != nil {
			return err
		}
	}
	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func writeCommunicatorSizesToFile(fd *os.File, numCalls int, commSizes map[int]int) error {
	_, err := fd.WriteString("# Communicator size(s)\n\n")
	if err != nil {
		return err
	}
	for commSize, n := range commSizes {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls use a communicator size of %d\n", n, numCalls, commSize))
		if err != nil {
			return err
		}
	}
	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

func writeCountStatsToFile(fd *os.File, numCalls int, sizeThreshold int, cs Stats) error {
	_, err := fd.WriteString("# Message sizes\n\n")
	if err != nil {
		return err
	}
	totalSendMsgs := cs.NumSendSmallMsgs + cs.NumSendLargeMsgs
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are large (threshold = %d)\n", cs.NumSendLargeMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small (threshold = %d)\n", cs.NumSendSmallMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small, but not 0-size (threshold = %d)\n", cs.NumSendSmallNotZeroMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}

	_, err = fd.WriteString("\n# Sparsity\n\n")
	if err != nil {
		return err
	}
	for numZeros, nCalls := range cs.CallSendSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d send counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}
	for numZeros, nCalls := range cs.CallRecvSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d recv counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n# Min/max\n")
	if err != nil {
		return err
	}
	for mins, n := range cs.SendMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.RecvMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for mins, n := range cs.SendNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.RecvNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for maxs, n := range cs.SendMaxs {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count max of %d\n", n, numCalls, maxs))
		if err != nil {
			return err
		}
	}
	for maxs, n := range cs.RecvMaxs {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count max of %d\n", n, numCalls, maxs))
		if err != nil {
			return err
		}
	}

	return nil
}
