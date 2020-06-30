//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/profiler"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"

	"github.com/gvallee/go_util/pkg/util"
)

func writePatternsToFile(fd *os.File, num int, cp *profiler.CallPattern) error {
	_, err := fd.WriteString(fmt.Sprintf("## Pattern #%d (%d alltoallv calls)\n", num, cp.Count))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("Alltoallv calls: %s\n", notation.CompressIntArray(cp.Calls)))
	if err != nil {
		return err
	}

	for sendTo, n := range cp.Send {
		_, err = fd.WriteString(fmt.Sprintf("%d ranks sent to %d other ranks\n", n, sendTo))
		if err != nil {
			return err
		}
	}
	for recvFrom, n := range cp.Recv {
		_, err = fd.WriteString(fmt.Sprintf("%d ranks recv'd from %d other ranks\n", n, recvFrom))
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

func writeCountStatsToFile(fd *os.File, numCalls int, sizeThreshold int, cs profiler.CountStats) error {
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

func displayPattern(p map[int]int, ctx string) {
	for numPeers, numRanks := range p {
		fmt.Printf("%d ranks are %s non-zero data to %d other ranks\n", numRanks, ctx, numPeers)
	}
}

func displayCallPatterns(info datafilereader.CallInfo) {
	for numPeers, numRanks := range info.Patterns.SendPatterns {
		fmt.Printf("%d ranks are sending non-zero data to %d other ranks\n", numRanks, numPeers)
	}
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	dir := flag.String("dir", "", "Where all the data is")
	outputDir := flag.String("output-dir", "", "Where all the output files will be created")
	rank := flag.Int("rank", 0, "Rank for which we want to analyse the counters. When using multiple communicators for alltoallv operations, results for multiple ranks are reported.")
	jobid := flag.Int("jobid", 0, "Job ID associated to the count files")
	sizeThreshold := flag.Int("size-threshold", 200, "Threshold to differentiate size and large messages")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s extracts data from send and receive counts and gathers statistics about them.", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
	}

	if !util.PathExists(*outputDir) {
		fmt.Printf("Output directory '%s' does not exist", *outputDir)
		flag.PrintDefaults()
		os.Exit(1)
	}

	logFile := util.OpenLogFile("alltoallv", cmdName)
	defer logFile.Close()
	if *verbose {
		nultiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(nultiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	defaultOutputFile := datafilereader.GetStatsFilePath(*outputDir, *jobid, *rank)
	patternsOutputFile := datafilereader.GetPatternFilePath(*outputDir, *jobid, *rank)
	patternsSummaryOutputFile := datafilereader.GetPatternSummaryFilePath(*outputDir, *jobid, *rank)
	defaultFd, err := os.OpenFile(defaultOutputFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("unable to create %s: %s", defaultOutputFile, err)
	}
	defer defaultFd.Close()
	patternsFd, err := os.OpenFile(patternsOutputFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("unable to create %s: %s", patternsOutputFile, err)
	}
	defer patternsFd.Close()
	patternsSummaryFd, err := os.OpenFile(patternsSummaryOutputFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("unable to create %s: %s", patternsSummaryOutputFile, err)
	}
	patternsSummaryFd.Close()

	sendCountsFile, recvCountsFile := datafilereader.GetCountsFiles(*jobid, *rank)
	sendCountsFile = filepath.Join(*dir, sendCountsFile)
	recvCountsFile = filepath.Join(*dir, recvCountsFile)

	if !util.PathExists(sendCountsFile) || !util.PathExists(recvCountsFile) {
		log.Fatalf("unable to locate send or recv counts file(s) (%s, %s) in %s", sendCountsFile, recvCountsFile, *dir)
	}

	log.Printf("Send counts file: %s\n", sendCountsFile)
	log.Printf("Recv counts file: %s\n", recvCountsFile)

	numCalls, err := datafilereader.GetNumCalls(sendCountsFile)
	if err != nil {
		log.Fatalf("unable to get the number of alltoallv calls: %s", err)
	}

	cs, err := profiler.ParseCountFiles(sendCountsFile, recvCountsFile, numCalls, *sizeThreshold)
	if err != nil {
		log.Fatalf("unable to parse count file %s", sendCountsFile)
	}

	_, err = defaultFd.WriteString(fmt.Sprintf("Total number of alltoallv calls: %d\n\n", numCalls))
	if err != nil {
		log.Fatalf("unable to write data: %s", err)
	}

	err = writeDatatypeToFile(defaultFd, numCalls, cs.DatatypesSend, cs.DatatypesRecv)
	if err != nil {
		log.Fatalf("unable to write datatype to file: %s", err)
	}

	err = writeCommunicatorSizesToFile(defaultFd, numCalls, cs.CommSizes)
	if err != nil {
		log.Fatalf("unable to write communicator sizes to file: %s", err)
	}

	err = writeCountStatsToFile(defaultFd, numCalls, *sizeThreshold, cs)
	if err != nil {
		log.Fatalf("unable to write communicator sizes to file: %s", err)
	}

	_, err = patternsFd.WriteString("# Patterns\n")
	if err != nil {
		log.Fatalf("unable to write data: %s", err)
	}
	num := 0
	for _, cp := range cs.Patterns.AllPatterns {
		err = writePatternsToFile(patternsFd, num, cp)
		if err != nil {
			log.Fatalf("unable to write patterns to file: %s", err)
		}
		num++
	}

	_, err = patternsFd.WriteString("# Patterns summary")
	num = 0
	for _, cp := range cs.Patterns.OneToN {
		err = writePatternsToFile(patternsSummaryFd, num, cp)
		if err != nil {
			log.Fatalf("unable to write patterns summary to file: %s", err)
		}
		num++
	}

	fmt.Println("Results are saved in:")
	fmt.Printf("-> %s\n", defaultOutputFile)
	fmt.Printf("-> %s\n", patternsOutputFile)
	fmt.Printf("Patterns summary: %s\n", patternsSummaryOutputFile)
}
