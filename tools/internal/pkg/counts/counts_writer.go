//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"fmt"
	"os"
	"sort"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
)

// WriteDatatypeToFile save datatype stats to a file
func WriteDatatypeToFile(fd *os.File, numCalls int, datatypesSend map[int]int, datatypesRecv map[int]int) error {
	// Sort by datatype size first so we have a predictable output
	sKeys := make([]int, len(datatypesSend))
	i := 0
	for key := range datatypesSend {
		sKeys[i] = key
		i++
	}
	sort.Ints(sKeys)

	rKeys := make([]int, len(datatypesRecv))
	i = 0
	for key := range datatypesRecv {
		rKeys[i] = key
		i++
	}
	sort.Ints(rKeys)

	_, err := fd.WriteString("# Datatypes\n\n")
	if err != nil {
		return err
	}
	//for datatypeSize, n := range datatypesSend {
	for _, size := range sKeys {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while sending data\n", datatypesSend[size], numCalls, size))
		if err != nil {
			return err
		}
	}
	//for datatypeSize, n := range datatypesRecv {
	for _, size := range rKeys {
		_, err := fd.WriteString(fmt.Sprintf("%d/%d calls use a datatype of size %d while receiving data\n", datatypesRecv[size], numCalls, size))
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

// WriteCommunicatorSizesToFile save to a file the data about communicator sizes
func WriteCommunicatorSizesToFile(fd *os.File, numCalls int, commSizes map[int]int) error {
	_, err := fd.WriteString("# Communicator size(s)\n\n")
	if err != nil {
		return err
	}

	kvList := format.ConvertIntMapToOrderedArrayByValue(commSizes)
	for _, elt := range kvList {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls use a communicator size of %d\n", elt.Val, numCalls, elt.Key))
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

// WriteCountStatsToFile writes all the stats to a file
func WriteCountStatsToFile(fd *os.File, numCalls int, sizeThreshold int, cs SendRecvStats) error {
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
	sSparsityKV := format.ConvertIntMapToOrderedArrayByValue(cs.CallSendSparsity)
	for _, keyval := range sSparsityKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d send counts equals to zero\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}
	rSparsityKV := format.ConvertIntMapToOrderedArrayByValue(cs.CallRecvSparsity)
	for _, keyval := range rSparsityKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d of all calls have %d recv counts equals to zero\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n# Min/max\n")
	if err != nil {
		return err
	}
	sMinsKV := format.ConvertIntMapToOrderedArrayByValue(cs.SendMins)
	for _, keyval := range sMinsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	rMinsKV := format.ConvertIntMapToOrderedArrayByValue(cs.RecvMins)
	for _, keyval := range rMinsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	sendNotZeroMinsKV := format.ConvertIntMapToOrderedArrayByValue(cs.SendNotZeroMins)
	for _, keyval := range sendNotZeroMinsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count min of %d (excluding zero)\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	recvNotZeroMinsKV := format.ConvertIntMapToOrderedArrayByValue(cs.RecvNotZeroMins)
	for _, keyval := range recvNotZeroMinsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count min of %d (excluding zero)\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	sendMaxsKV := format.ConvertIntMapToOrderedArrayByValue(cs.SendMaxs)
	for _, keyval := range sendMaxsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a send count max of %d\n", keyval.Val, numCalls, keyval.Key))
		if err != nil {
			return err
		}
	}

	_, err = fd.WriteString("\n")
	if err != nil {
		return err
	}

	recvMaxsKV := format.ConvertIntMapToOrderedArrayByValue(cs.RecvMaxs)
	for _, keyval := range recvMaxsKV {
		_, err = fd.WriteString(fmt.Sprintf("%d/%d calls have a recv count max of %d\n", keyval.Val, numCalls, keyval.Key))
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
