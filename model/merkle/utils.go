/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

func convertBytesToMerkleHash(b []byte) *MerkleHash {
	var bytes [HashSize]byte
	copy(bytes[:], b)
	hash := MerkleHash(bytes)
	return &hash
}

// ComputeMerkleHash returns its merkle hash.
func ComputeMerkleHash(b []byte) *MerkleHash {
	hash := MerkleHash(sha256.Sum256(b))
	return &hash
}

// Hash returns the left(or right) child and it sibling's merkle hash.
func Hash(hash *MerkleHash, siblingHash *MerkleHash, pos byte) *MerkleHash {
	hashInput := make([]byte, HashSize*2)
	if pos == Left {
		hashInput = append(hashInput, hash[:]...)
		hashInput = append(hashInput, siblingHash[:]...)
	} else {
		hashInput = append(hashInput, siblingHash[:]...)
		hashInput = append(hashInput, hash[:]...)
	}
	newHash := MerkleHash(sha256.Sum256(hashInput))
	return &newHash
}

// VerifyLastNodeMerklePath verify a merkle path is the merkle path for the last leaf node in the merkle tree before updating.
// This method has the assumption that the size is trustworthy.
func VerifyLastNodeMerklePath(merkleRoot *MerkleHash, size int64, lastNodeMerklePath *MerklePath) error {
	err := lastNodeMerklePath.Verify(merkleRoot)
	if err != nil {
		return err
	}
	lengthOfProof := countOne(size - 1)
	if len(lastNodeMerklePath.ProofPath) != lengthOfProof || len(lastNodeMerklePath.Hashes) != lengthOfProof+1 {
		return fmt.Errorf("length of merkle path is wrong, size %v, expected length %v, actual length %v",
			size, lengthOfProof+1, len(lastNodeMerklePath.Hashes))
	}
	for _, pos := range lastNodeMerklePath.ProofPath {
		if pos != Left {
			return errors.New("wrong merkle path, not the one for last node")
		}
	}
	return nil
}

func bytesToUint64(bytes []byte) uint64 {
	return uint64(binary.BigEndian.Uint64(bytes))
}
