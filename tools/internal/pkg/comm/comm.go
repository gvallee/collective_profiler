//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package comm

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/format"
)

const (
	// CommsFileToken is the token used to identify files with communicators' data
	CommsFileToken = "_comm_data_rank"
)

// Info represents the information about a single communicator
type Info struct {
	ID       int
	LeadRank int
}

// CommsInfo represents the data about all the communicators from the profile data
type CommsInfo struct {
	// Comms is an array of communicator information
	Comms []*Info

	// LeadMap is a map where the key is the world rank of the rank 0 on the communicator and the
	// value an array of communicator IDs
	LeadMap map[int][]int
}

func parseCommFile(codeBaseDir string, path string, leadRank int) (*CommsInfo, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Populate the rank map
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("comm file %s does not have a proper header", path)
	}

	formatMatch, err := format.CheckDataFormatLineFromProfileFile(lines[0], codeBaseDir)
	if err != nil {
		return nil, fmt.Errorf("unable to parse format version: %s", err)
	}
	if !formatMatch {
		return nil, fmt.Errorf("data format does not match")
	}

	// Followed by an empty line
	if lines[1] != "" {
		return nil, fmt.Errorf("invalid data format, second line is '%s' instead of an empty line", lines[1])
	}

	comms := new(CommsInfo)
	comms.LeadMap = make(map[int][]int)
	nLine := 2
	for nLine < len(lines) {
		if lines[nLine] == "" {
			nLine++
			continue
		}

		line := strings.TrimRight(lines[nLine], "\n")

		tokens := strings.Split(line, "; ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("parseCommFile() - invalid format: %s", line)
		}
		commIDStr := strings.TrimLeft(tokens[0], "ID: ")
		commID, err := strconv.Atoi(commIDStr)
		if err != nil {
			return nil, err
		}
		rankStr := strings.TrimLeft(tokens[1], "world rank: ")
		rank, err := strconv.Atoi(rankStr)
		if err != nil {
			return nil, err
		}
		if leadRank != rank {
			return nil, fmt.Errorf("lead rank on line %d in %s is %d instead of expected %d", nLine, path, rank, leadRank)
		}

		info := new(Info)
		info.ID = commID
		info.LeadRank = leadRank
		comms.Comms = append(comms.Comms, info)
		comms.LeadMap[leadRank] = append(comms.LeadMap[leadRank], commID)

		nLine++
	}

	return comms, nil
}

func parseFileName(filename string) (string, int, error) {
	tokens := strings.Split(filename, "_")
	if len(tokens) != 4 {
		return "", -1, fmt.Errorf("%s is not a valid filename", filename)
	}
	rankStr := strings.TrimRight(tokens[3], ".md")
	rankStr = strings.TrimLeft(rankStr, "rank")
	leadRank, err := strconv.Atoi(rankStr)
	if err != nil {
		return "", -1, err
	}

	return tokens[0], leadRank, nil
}

// GetData parses all the communcator file in a directory
func GetData(codeBaseDir string, dir string) (*CommsInfo, error) {
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	allComms := new(CommsInfo)
	allComms.LeadMap = make(map[int][]int)
	for _, file := range f {
		filename := file.Name()
		if strings.Contains(filename, CommsFileToken) {
			_, leadRank, err := parseFileName(filename)
			if err != nil {
				continue
			}
			comms, err := parseCommFile(codeBaseDir, filepath.Join(dir, filename), leadRank)
			if err != nil {
				return nil, err
			}
			// Merge results with the ones we already have
			for _, comm := range comms.Comms {
				allComms.Comms = append(allComms.Comms, comm)
			}

			for rank, listComms := range comms.LeadMap {
				allComms.LeadMap[rank] = append(allComms.LeadMap[rank], listComms...)
			}
		}
	}

	return allComms, nil
}
