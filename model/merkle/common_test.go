/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"crypto/sha256"
	"testing"
)

func TestVerifyMerklePath(t *testing.T) {
	// Create a simple merkle tree
	a := MerkleHash(sha256.Sum256([]byte("a")))
	b := MerkleHash(sha256.Sum256([]byte("b")))
	c := MerkleHash(sha256.Sum256([]byte("c")))
	d := MerkleHash(sha256.Sum256([]byte("d")))
	ab := Hash(&a, &b, Right)
	cd := Hash(&c, &d, Right)
	abcd := Hash(ab, cd, Right)

	merklePath := MerklePath{
		Hashes:    []*MerkleHash{&b, &a, cd},
		ProofPath: []byte{Left, Right},
	}

	var err interface{}

	err = merklePath.Verify(abcd)
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}

	err = merklePath.Verify(cd)
	if _, ok := err.(RootHashMismatchError); !ok {
		t.Errorf("Expecting RootHashMismatchError")
	}
}

func TestIsEmptyMerkleHash(t *testing.T) {
	hash := &MerkleHash{}
	if !hash.IsEmptyHash() {
		t.Errorf("Hash is empty, IsEmptyHash should return True")
	}
	bts := [HashSize]byte(*hash)
	hash = ComputeMerkleHash(bts[:])
	if hash.IsEmptyHash() {
		t.Errorf("Hash is not empty, IsEmptyHash should return False")
	}
}
