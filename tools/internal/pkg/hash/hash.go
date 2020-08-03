//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func File(path string) (string, error) {
	fileFd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fileFd.Close()
	hasher := sha256.New()
	_, err = io.Copy(hasher, fileFd)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
