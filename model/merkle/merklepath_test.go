/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"crypto/sha256"
	"encoding/gob"
	"os"
	"reflect"

	//"strconv"
	"testing"
)

func TestEncodeAndDecodeMerklePath(t *testing.T) {
	a := MerkleHash(sha256.Sum256([]byte("a")))
	b := MerkleHash(sha256.Sum256([]byte("b")))
	c := MerkleHash(sha256.Sum256([]byte("c")))
	d := MerkleHash(sha256.Sum256([]byte("d")))
	ab := Hash(&a, &b, Right)
	cd := Hash(&c, &d, Right)
	merklePathForB := MerklePath{
		Hashes:    []*MerkleHash{&b, &a, cd},
		ProofPath: []byte{Left, Right},
	}
	merklePathForC := MerklePath{
		Hashes:    []*MerkleHash{&c, &d, ab},
		ProofPath: []byte{Right, Left},
	}
	merklePath := []MerklePath{
		merklePathForB,
		merklePathForC,
	}
	filename := "testencodeanddecodemerklepath.testdata"
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("failed to create file,err:%v", err)
	}
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(merklePath)
	if err != nil {
		log.Errorf("failed to encode merkle path,err:%v", err)
	}
	file.Close()

	newMerklePath := []MerklePath{}
	file, err = os.Open(filename)
	if err != nil {
		log.Errorf("failed to open file,err:%v", err)
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&newMerklePath)
	file.Close()

	if err != nil {
		t.Errorf(err.Error())
	}

	if !reflect.DeepEqual(merklePath, newMerklePath) {
		t.Errorf("The decoded merkle path are different from the encoded merkle path")
	}
}

/*
// Add a test to test the size of merkle path.
func TestMerklePathSize(t *testing.T) {
	merkleTree := NewFullMerkleTree()
	for i := 0; i < 1000000; i++ {
		hash := MerkleHash(sha256.Sum256([]byte(strconv.Itoa(i))))
		merkleTree.Append(&hash)
	}
	merklePaths := []MerklePath{}
	for i := 0; i < 10000; i++ {
		j := i * 100
		hash := MerkleHash(sha256.Sum256([]byte(strconv.Itoa(j))))
		merklePath, err := merkleTree.GetMerklePath(&hash)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		merklePaths = append(merklePaths, *merklePath)
	}
	filename := "merklepathsizetest.testdata"
	file, err := os.Create(filename)
	encoder := gob.NewEncoder(file)
	encoder.Encode(merklePaths)
	file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
}
*/
