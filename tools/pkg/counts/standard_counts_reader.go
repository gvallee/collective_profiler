//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package counts

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

type StandardCounts struct {
	SendCounts []string
	RecvCounts []string
}

func GetStandardCounts(reader *bufio.Reader) (*StandardCounts, error) {
	callCounters := new(StandardCounts)

	// First line is the send counts header
	line, err := reader.ReadString('\n')
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(line, StandardFormatSendCountsMarker) {
		return nil, fmt.Errorf("invalid header: %s", line)
	}

	// Send counts
	for {
		line, err = reader.ReadString('\n')
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}
		callCounters.SendCounts = append(callCounters.SendCounts, line)
	}

	//  Recv counts header
	line, err = reader.ReadString('\n')
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(line, StandardFormatRecvCountsMarker) {
		return nil, fmt.Errorf("invalid header: %s", line)
	}

	// Recv counts
	for {
		line, err = reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		callCounters.SendCounts = append(callCounters.RecvCounts, line)
	}

	return callCounters, nil
}

func GetStandardHeader(reader *bufio.Reader) (HeaderT, error) {
	var header HeaderT

	// The first line is the send datatype size
	line, err := reader.ReadString('\n')
	if err == io.EOF {
		return header, err
	}
	if err != nil {
		return header, err
	}
	if !strings.HasSuffix(line, StandardFormatSendDatatypeMarker) {
		return header, fmt.Errorf("invalid header: %s", line)
	}
	header.DatatypeInfo.StandardFormatDatatypeInfo.SendDatatypeSize, err = strconv.Atoi(strings.TrimPrefix(line, StandardFormatSendDatatypeMarker))
	if err != nil {
		return header, err
	}

	// The second line is the recv datatype size
	line, err = reader.ReadString('\n')
	if err == io.EOF {
		return header, err
	}
	if err != nil {
		return header, err
	}
	if !strings.HasSuffix(line, StandardFormatRecvDatatypeMarker) {
		return header, fmt.Errorf("invalid header: %s", line)
	}
	header.DatatypeInfo.StandardFormatDatatypeInfo.RecvDatatypeSize, err = strconv.Atoi(strings.TrimPrefix(line, StandardFormatRecvDatatypeMarker))
	if err != nil {
		return header, err
	}

	// The third line is the communicator size
	line, err = reader.ReadString('\n')
	if err == io.EOF {
		return header, err
	}
	if err != nil {
		return header, err
	}
	if !strings.HasSuffix(line, StandardFormatRecvDatatypeMarker) {
		return header, fmt.Errorf("invalid header: %s", line)
	}
	if !strings.HasSuffix(line, StandardFormatCommSizeMarker) {
		return header, fmt.Errorf("invalid header: %s", line)
	}
	header.NumRanks, err = strconv.Atoi(strings.TrimPrefix(line, StandardFormatCommSizeMarker))
	if err != nil {
		return header, err
	}

	// Finally the next line is empty
	line, err = reader.ReadString('\n')
	if err == io.EOF {
		return header, err
	}
	if err != nil {
		return header, err
	}
	if line != "" {
		return header, fmt.Errorf("invalid header: %s", line)
	}

	return header, nil
}

func getCallIDFromFileName(filepath string) (int, error) {
	filename := path.Base(filepath)
	str := strings.TrimPrefix(filename, RawCountersFilePrefix)
	str = strings.TrimSuffix(str, ".md")
	tokens := strings.Split(str, "_call")
	if len(tokens) != 2 {
		return -1, fmt.Errorf("unable to parse %s", filename)
	}
	callID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return -1, err
	}
	return callID, nil
}

// ParsePerCallFileCount loads the counts from a non-compact count file.
// With that format, details about each call (both send and receive counts)
// are saved in separate files.
func ParsePerCallFileCount(path string) ([]RawCountsCallsT, error) {
	var rawCounts []RawCountsCallsT

	// Get the call ID from the file name
	callID, err := getCallIDFromFileName(path)
	if err != nil {
		return nil, err
	}

	// Parse the content of the file
	countFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", path, err)
	}
	defer countFile.Close()
	reader := bufio.NewReader(countFile)
	header, err := GetStandardHeader(reader)
	if err != nil {
		return nil, err
	}
	counts, err := GetStandardCounts(reader)
	if err != nil {
		return nil, err
	}
	rc := new(rawCountsT)
	rc.commSize = header.NumRanks
	rc.recvCounts = counts.RecvCounts
	rc.sendCounts = counts.SendCounts
	rc.recvDatatypeSize = header.DatatypeInfo.StandardFormatDatatypeInfo.RecvDatatypeSize
	rc.sendDatatypeSize = header.DatatypeInfo.StandardFormatDatatypeInfo.SendDatatypeSize

	newData := RawCountsCallsT{
		calls:  []int{callID},
		counts: rc,
	}
	rawCounts = append(rawCounts, newData)

	return rawCounts, nil
}
