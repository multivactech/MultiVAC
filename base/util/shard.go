// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os/exec"
)

// Int64ToBytes transform int64 to byte array.
func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// BytesToInt64 transform the byte array to int64.
func BytesToInt64(buf []byte) int64 {
	if len(buf) < 1 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(buf))
}

func RedirectLog(content, fileAbsPath string) (string, error) {
	str := fmt.Sprintf("echo \"%s\">>%s", content, fileAbsPath)
	cmd := exec.Command("sh", "-c", str)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stderr.String() != "" {
		return "", fmt.Errorf("err: %v %v", err, stderr.String())
	}

	return out.String(), nil
}
