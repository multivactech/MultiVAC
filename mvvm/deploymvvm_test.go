/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
)

func getTestCode(filename string) []byte {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestNewDeployMvvm(t *testing.T) {
	code := getTestCode("test_init.wasm")
	vm, err := NewDeployMvvm(code)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Error("runtime error when executing init")
		}
	}()
	data := vm.Initialize(shard.Index(0))
	if string(data) != "abcdefghijklmnopqrstuvwxyz" {
		t.Error(fmt.Errorf("incorrect return value: %v", data))
	}
}
