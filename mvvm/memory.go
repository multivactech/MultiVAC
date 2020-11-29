/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"fmt"

	"github.com/multivactech/MultiVAC/base/rlp"
)

const (
	// defaultMemoryPages shows the default count of memory page for a new MVVM.
	// Citing from wasm document, "The length of the vector always is a multiple of the WebAssembly page size, which is
	// defined to be the constant 65536 – abbreviated 64 Ki."
	// For now defaultMemoryPages is set as 128, same as in the test cases in life.
	defaultMemoryPages = 1 << 7

	// defaultMemoryOffset shows the default offset of memory for storing the given MVVM input data.
	// TODO (wangruichao@mtv.ac): consider the correct offset for user code.
	//   For now defaultMemoryOffset is set as 3000.
	defaultMemoryOffset = 3000
)

// validMemory gets the shard data in vm.Memory and returns it.
func validMemory(memory []byte, offset int) ([]byte, error) {
	// For now, consider the situation that the developer returns a char* in C/Cpp.
	// So the return value begins from the offset, and ends at the first '\0' char.
	// TODO (wangruichao@mtv.ac): return the shard data in other situations.
	if offset < 0 || offset >= len(memory) {
		return []byte{}, fmt.Errorf("invalid offset in vm.Memory: %d", offset)
	}
	for end := offset; ; end++ {
		if end >= len(memory) || memory[end] == 0 {
			return memory[offset:end], nil
		}
	}
}

// storeIntoMemory encodes the given struct, and stores the encoding result into memory.
func storeIntoMemory(memory []byte, offset int, s interface{}) (int, error) {
	bytes, err := rlp.EncodeToBytes(s)
	if err != nil {
		log.Errorf("Error got when encoding struct %v to rlp: %v\n", s, err)
		return 0, err
	}
	length := len(bytes)
	copy(memory[offset:offset+length], bytes)
	return length, nil
}

// loadFromMemory loads data from the given memory, and decodes it into a struct.
// func loadFromMemory(memory []byte, offset int, s interface{}) error {
// 	err := rlp.DecodeBytes(memory[offset:], s)
// 	if err != nil {
// 		log.Errorf("Error got when decoding data in memory: %v\n", err)
// 		return err
// 	}
// 	return nil
// }

// getMemUint32 returns the uint32 value from the position in memory.
//
// Because that the data is stored little Endian, data is taken out one by one
// from the bigger position to smaller one, and the length of the stored data is 4.
func getMemUint32(mem []byte, ptr int) uint32 {
	var res uint32
	for i := 3; i >= 0; i-- {
		res = (res << 8) + uint32(mem[ptr+i])
	}
	return res
}
