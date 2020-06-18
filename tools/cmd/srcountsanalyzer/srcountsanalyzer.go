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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/datafilereader"

	"github.com/gvallee/go_util/pkg/util"
)

type callPattern struct {
	send  map[int]int
	recv  map[int]int
	count int
	calls []int
}

type GlobalPatterns struct {
	cp     []*callPattern
	oneToN []*callPattern
}

type countStats struct {
	numSendSmallMsgs        int
	numSendLargeMsgs        int
	sizeThreshold           int
	numSendSmallNotZeroMsgs int
	callSendSparsity        map[int]int
	callRecvSparsity        map[int]int
	sendMins                map[int]int
	recvMins                map[int]int
	sendMaxs                map[int]int
	recvMaxs                map[int]int
	sendNotZeroMins         map[int]int
	recvNotZeroMins         map[int]int
}

func (globalPatterns *GlobalPatterns) addPattern(callNum int, sendPatterns map[int]int, recvPatterns map[int]int) error {
	for idx, x := range globalPatterns.cp {
		if datafilereader.CompareCallPatterns(x.send, sendPatterns) && datafilereader.CompareCallPatterns(x.recv, recvPatterns) {
			// Increment count for pattern
			log.Printf("-> Alltoallv call #%d - Adding alltoallv to pattern %d...\n", callNum, idx)
			x.count++
			x.calls = append(x.calls, callNum)

			// todo: We may want to track 1 -> N more independently but right now, we handle pointers
			// so the details are only about the main list.
			/*
				if sentTo > n*100 {
					// This is also a 1->n pattern and we need to update the list of such patterns
					for _, candidatePattern := range globalPatterns.oneToN {
						if datafilereader.CompareCallPatterns(candidatePattern.send, sendPatterns) && datafilereader.CompareCallPatterns(candidatePattern.recv, recvPatterns) {
							candidatePattern.count ++
						}
					}
				}
			*/
			return nil
		}
	}

	// If we get here, it means that we did not find a similar pattern
	log.Printf("-> Alltoallv call %d - Adding new pattern...\n", callNum)
	new_cp := new(callPattern)
	new_cp.send = sendPatterns
	new_cp.recv = recvPatterns
	new_cp.count = 1
	new_cp.calls = append(new_cp.calls, callNum)
	globalPatterns.cp = append(globalPatterns.cp, new_cp)

	// Detect 1 -> n patterns using the send counts only
	for sendTo, n := range sendPatterns {
		if sendTo > n*100 {
			globalPatterns.oneToN = append(globalPatterns.oneToN, new_cp)
		}
	}

	return nil
}

func writePatternsToFile(fd *os.File, num int, cp *callPattern) error {
	_, err := fd.WriteString(fmt.Sprintf("## Pattern #%d (%d alltoallv calls)\n", num, cp.count))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("Alltoallv calls: %s\n", notation.CompressIntArray(cp.calls)))
	if err != nil {
		return err
	}

	for sendTo, n := range cp.send {
		_, err = fd.WriteString(fmt.Sprintf("%d ranks sent to %d other ranks\n", n, sendTo))
		if err != nil {
			return err
		}
	}
	for recvFrom, n := range cp.recv {
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

func writeCountStatsToFile(fd *os.File, numCalls int, sizeThreshold int, cs countStats) error { // numSendSmallMsgs int, numSendLargeMsgs int, sizeThreshold int, numSendSmallNotZeroMsgs int, callSendSparsity map[int]int, callRecvSparsity map[int]int, sendMins map[int]int, recvMins map[int]int) error {
	_, err := fd.WriteString("# Message sizes\n\n")
	if err != nil {
		return err
	}
	totalSendMsgs := cs.numSendSmallMsgs + cs.numSendLargeMsgs
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are large (threshold = %d)\n", cs.numSendLargeMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small (threshold = %d)\n", cs.numSendSmallMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("%d/%d of all messages are small, but not 0-size (threshold = %d)\n", cs.numSendSmallNotZeroMsgs, totalSendMsgs, sizeThreshold))
	if err != nil {
		return err
	}

	_, err = fd.WriteString("\n# Sparsity\n\n")
	if err != nil {
		return err
	}
	for numZeros, nCalls := range cs.callSendSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d send counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}
	for numZeros, nCalls := range cs.callRecvSparsity {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d recv counts equals to zero\n", nCalls, numCalls, numZeros))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n# Min/max\n")
	if err != nil {
		return err
	}
	for mins, n := range cs.sendMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.recvMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for mins, n := range cs.sendNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}
	for mins, n := range cs.recvNotZeroMins {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d (excluding zero)\n", n, numCalls, mins))
		if err != nil {
			return err
		}
	}

	for maxs, n := range cs.sendMaxs {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count max of %d\n", n, numCalls, maxs))
		if err != nil {
			return err
		}
	}
	for maxs, n := range cs.recvMaxs {
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

func newCountStats() countStats {
	cs := countStats{
		sendMins:                make(map[int]int),
		recvMins:                make(map[int]int),
		sendMaxs:                make(map[int]int),
		recvMaxs:                make(map[int]int),
		recvNotZeroMins:         make(map[int]int),
		sendNotZeroMins:         make(map[int]int),
		callSendSparsity:        make(map[int]int),
		callRecvSparsity:        make(map[int]int),
		numSendSmallMsgs:        0,
		numSendSmallNotZeroMsgs: 0,
		numSendLargeMsgs:        0,
	}
	return cs
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

	_, err = defaultFd.WriteString(fmt.Sprintf("Total number of alltoallv calls: %d\n\n", numCalls))
	if err != nil {
		log.Fatalf("unable to write data: %s", err)
	}

	//a := analyzer.CreateSRCountsAnalyzer(sendCountsFile, recvCountsFile)

	/*
		fSendCounts, err := os.Open(sendCountsFile)
		if err != nil {
			log.Fatalf("unable to open %s: %s", sendCountsFile, err)
		}
		defer fSendCounts.Close()

		fRecvCounts, err := os.Open(recvCountsFile)
		if err != nil {
			log.Fatalf("unable to open %s: %s", sendCountsFile, err)
		}

		sendCountReader := bufio.NewReader(fSendCounts)
		recvCountReader := bufio.NewReader(fRecvCounts)
	*/

	var globalPatterns GlobalPatterns
	var patterns1ToN []*callPattern
	datatypesSend := make(map[int]int)
	datatypesRecv := make(map[int]int)
	commSizes := make(map[int]int)

	cs := newCountStats()

	for i := 0; i < numCalls; i++ {
		log.Printf("Analyzing call #%d\n", i)
		callInfo, err := datafilereader.LookupCall(sendCountsFile, recvCountsFile, i, *sizeThreshold)
		if err != nil {
			log.Fatalf("unable to lookup call #%d: %s", i, err)
		}

		cs.numSendSmallMsgs += callInfo.SendSmallMsgs
		cs.numSendSmallNotZeroMsgs += callInfo.SendSmallNotZeroMsgs
		cs.numSendLargeMsgs += callInfo.SendLargeMsgs

		if _, ok := datatypesSend[callInfo.SendDatatypeSize]; ok {
			datatypesSend[callInfo.SendDatatypeSize]++
		} else {
			datatypesSend[callInfo.SendDatatypeSize] = 1
		}

		if _, ok := datatypesRecv[callInfo.RecvDatatypeSize]; ok {
			datatypesRecv[callInfo.RecvDatatypeSize]++
		} else {
			datatypesRecv[callInfo.RecvDatatypeSize] = 1
		}

		if _, ok := commSizes[callInfo.CommSize]; ok {
			commSizes[callInfo.CommSize]++
		} else {
			commSizes[callInfo.CommSize] = 1
		}

		if _, ok := cs.sendMins[callInfo.SendMin]; ok {
			cs.sendMins[callInfo.SendMin]++
		} else {
			cs.sendMins[callInfo.SendMin] = 1
		}

		if _, ok := cs.recvMins[callInfo.RecvMin]; ok {
			cs.recvMins[callInfo.RecvMin]++
		} else {
			cs.recvMins[callInfo.RecvMin] = 1
		}

		if _, ok := cs.sendMaxs[callInfo.SendMax]; ok {
			cs.sendMaxs[callInfo.SendMax]++
		} else {
			cs.sendMaxs[callInfo.SendMax] = 1
		}

		if _, ok := cs.recvMaxs[callInfo.RecvMax]; ok {
			cs.recvMaxs[callInfo.RecvMax]++
		} else {
			cs.recvMaxs[callInfo.RecvMax] = 1
		}

		if _, ok := cs.sendNotZeroMins[callInfo.SendNotZeroMin]; ok {
			cs.sendMins[callInfo.SendNotZeroMin]++
		} else {
			cs.sendMins[callInfo.SendNotZeroMin] = 1
		}

		if _, ok := cs.recvNotZeroMins[callInfo.RecvNotZeroMin]; ok {
			cs.recvMins[callInfo.RecvNotZeroMin]++
		} else {
			cs.recvMins[callInfo.RecvNotZeroMin] = 1
		}

		if _, ok := cs.callSendSparsity[callInfo.TotalSendZeroCounts]; ok {
			cs.callSendSparsity[callInfo.TotalSendZeroCounts]++
		} else {
			cs.callSendSparsity[callInfo.TotalSendZeroCounts] = 1
		}

		if _, ok := cs.callRecvSparsity[callInfo.TotalRecvZeroCounts]; ok {
			cs.callRecvSparsity[callInfo.TotalRecvZeroCounts]++
		} else {
			cs.callRecvSparsity[callInfo.TotalRecvZeroCounts] = 1
		}

		//displayCallPatterns(callInfo)
		// Analyze the send/receive pattern from the call
		err = globalPatterns.addPattern(i, callInfo.Patterns.SendPatterns, callInfo.Patterns.RecvPatterns)
		if err != nil {
			log.Fatalf("unabel to add pattern: %s", err)
		}
	}

	err = writeDatatypeToFile(defaultFd, numCalls, datatypesSend, datatypesRecv)
	if err != nil {
		log.Fatalf("unable to write datatype to file: %s", err)
	}

	err = writeCommunicatorSizesToFile(defaultFd, numCalls, commSizes)
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
	for _, cp := range globalPatterns.cp {
		writePatternsToFile(patternsFd, num, cp)
		num++
	}

	_, err = patternsFd.WriteString("# Patterns summary")
	num = 0
	for _, cp := range patterns1ToN {
		writePatternsToFile(patternsSummaryFd, num, cp)
		num++
	}

	fmt.Println("Results are saved in:")
	fmt.Printf("-> %s\n", defaultOutputFile)
	fmt.Printf("-> %s\n", patternsOutputFile)
	fmt.Printf("Patterns summary: %s\n", patternsSummaryOutputFile)
}
