/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"errors"
	"fmt"
)

// DeltaMerkleTree is a data structure used by normal node to calculate the delta updates to ledger's
// merkle tree without knowing the tree itself. It's able to update leaves' hash using its merkle path,
// append new node into the tree, and given a merkle path from old merkle tree, create a new merkle path
// corresponding to the new tree.
// One tree instance is corresponding to one block's update to ledger's merkle tree. Create a new one
// when handling a new block.
type DeltaMerkleTree struct {
	MerkleTree
}

// NewDeltaMerkleTree creates a delta merkle tree.
func NewDeltaMerkleTree(merkleRoot *MerkleHash) *DeltaMerkleTree {
	deltaMerkleTree := DeltaMerkleTree{
		MerkleTree: *newEmptyRAMMerkleTree(),
	}
	deltaMerkleTree.MerkleRoot = merkleRoot
	// Size is unknown.
	deltaMerkleTree.Size = UnknownSize
	return &deltaMerkleTree
}

// UpdateLeaf update a leaf node's hash to a new hash.
func (deltaMerkleTree *DeltaMerkleTree) UpdateLeaf(merklePath *MerklePath, newHash *MerkleHash) error {
	deltaMerkleTree.MerkleTree.mtx.Lock()
	defer deltaMerkleTree.MerkleTree.mtx.Unlock()
	err := merklePath.Verify(deltaMerkleTree.MerkleRoot)
	if err != nil {
		return err
	}
	deltaMerkleTree.buildPartialTreeForOldMerklePath(merklePath)
	hash := merklePath.Hashes[0]
	node, _ := deltaMerkleTree.nodeMap.get(*hash)
	deltaMerkleTree.updateNode(node, newHash)
	return nil
}

// GetShardDataMerklePath update a leaf node's hash to a new hash.
func (deltaMerkleTree *DeltaMerkleTree) GetShardDataMerklePath(oldHash, newHash *MerkleHash) (*MerklePath, error) {
	deltaMerkleTree.mtx.Lock()
	node, ok := deltaMerkleTree.nodeMap.get(*oldHash)
	deltaMerkleTree.mtx.Unlock()

	if !ok {
		return nil, fmt.Errorf("can't find hash %x in deltaMerkleTree", *oldHash)
	}

	hashes := []*MerkleHash{newHash}
	proofHashes, proofPath := createMerkleProof(node)

	path := MerklePath{
		Hashes:    append(hashes, proofHashes...),
		ProofPath: proofPath,
	}

	if err := path.Verify(deltaMerkleTree.GetMerkleRoot()); err != nil {
		return nil, err
	}

	return &path, nil
}

// Returns the merkle root after updates.
func (deltaMerkleTree *DeltaMerkleTree) GetMerkleRoot() *MerkleHash {
	deltaMerkleTree.MerkleTree.mtx.Lock()
	defer deltaMerkleTree.MerkleTree.mtx.Unlock()

	if node, ok := deltaMerkleTree.nodeMap.get(*deltaMerkleTree.MerkleRoot); ok {
		for node.getParent() != nil {
			node = node.getParent()
		}
		return node.getHash()
	}
	return deltaMerkleTree.MerkleRoot
}

// RefreshMerklePath refresh an old merkle path which points to the old merkle root to a new merkle path that points to the new merkle root.
func (deltaMerkleTree *DeltaMerkleTree) RefreshMerklePath(merklePath *MerklePath) (*MerklePath, error) {
	deltaMerkleTree.MerkleTree.mtx.Lock()
	defer deltaMerkleTree.MerkleTree.mtx.Unlock()
	err := merklePath.Verify(deltaMerkleTree.MerkleRoot)
	if err != nil {
		return nil, err
	}

	leafHash := merklePath.Hashes[0]
	node, ok := deltaMerkleTree.nodeMap.get(*leafHash)

	hashes := []*MerkleHash{}
	hashes = append(hashes, leafHash)
	path := []byte{}
	nodeHash := leafHash

	if !ok {
		for index, hash := range merklePath.Hashes[1:] {
			hashes = append(hashes, hash)
			path = append(path, merklePath.ProofPath[index])

			parentHash := Hash(nodeHash, hash, merklePath.ProofPath[index])
			node, ok = deltaMerkleTree.nodeMap.get(*parentHash)
			if ok {
				break
			}
			nodeHash = parentHash
		}
	}

	if ok {
		proofHashes, proofPath := createMerkleProof(node)
		hashes = append(hashes, proofHashes...)
		path = append(path, proofPath...)
	}

	return &MerklePath{
		Hashes:    hashes,
		ProofPath: path,
	}, nil
}

// SetSizeAndLastNodeMerklePath is in order to append node into merkle tree, this tree needs to know the size of tree and last node's merkle path.
func (deltaMerkleTree *DeltaMerkleTree) SetSizeAndLastNodeMerklePath(size int64, lastNodeMerklePath *MerklePath) error {
	deltaMerkleTree.MerkleTree.mtx.Lock()
	defer deltaMerkleTree.MerkleTree.mtx.Unlock()
	if deltaMerkleTree.Size == UnknownSize {
		err := VerifyLastNodeMerklePath(deltaMerkleTree.MerkleRoot, size, lastNodeMerklePath)
		if err != nil {
			return err
		}
		deltaMerkleTree.buildPartialTreeForOldMerklePath(lastNodeMerklePath)
		deltaMerkleTree.lastNodeHash = lastNodeMerklePath.Hashes[0]
		deltaMerkleTree.Size = size
		return nil
	}
	return errors.New("size and merkle path for last node have already been set")

}

// Append add leaves to delta merkle tree.
func (deltaMerkleTree *DeltaMerkleTree) Append(leaves ...*MerkleHash) error {
	deltaMerkleTree.MerkleTree.mtx.Lock()
	defer deltaMerkleTree.MerkleTree.mtx.Unlock()
	return deltaMerkleTree.MerkleTree.append(leaves...)
}

// getOrCreateNode gets or creates a node with the specified hash.
// If the node exists, return node and true
// If the node doesn't exist, create and return node and false
func (deltaMerkleTree *DeltaMerkleTree) getOrCreateNode(hash *MerkleHash) (iMerkleNode, bool) {
	node, ok := deltaMerkleTree.nodeMap.get(*hash)
	if ok {
		return node, true
	}
	node = deltaMerkleTree.MerkleTree.newNode()
	node.setHash(hash)
	deltaMerkleTree.nodeMap.set(*hash, node)
	return node, false
}

// buildPartialTreeForOldMerklePath build the parts of the tree provided by the original merkle path.
// If a node already exists, that means it was created by previous calls of this function.
// In this case, we don't recreate the node or touch the hash.
func (deltaMerkleTree *DeltaMerkleTree) buildPartialTreeForOldMerklePath(merklePath *MerklePath) {
	hash := merklePath.Hashes[0]
	curNode, exists := deltaMerkleTree.getOrCreateNode(hash)
	if exists {
		return
	}
	for index, siblingPos := range merklePath.ProofPath {
		siblingHash := merklePath.Hashes[index+1]
		siblingNode, exists := deltaMerkleTree.getOrCreateNode(siblingHash)
		// If sibling already exists, the tree above has been built.
		if exists {
			break
		}
		parentHash := Hash(hash, siblingHash, siblingPos)
		parentNode, _ := deltaMerkleTree.getOrCreateNode(parentHash)

		curNode.setParent(parentNode)
		siblingNode.setParent(parentNode)
		if siblingPos == Left {
			parentNode.setLeftChild(siblingNode)
			parentNode.setRightChild(curNode)
		} else {
			parentNode.setLeftChild(curNode)
			parentNode.setRightChild(siblingNode)
		}
		hash = parentHash
		curNode = parentNode
	}
}

// GetMerklePath returns the merkle path.
func (merkleTree *DeltaMerkleTree) GetMerklePath(hash *MerkleHash) (*MerklePath, error) {
	merkleTree.MerkleTree.mtx.Lock()
	defer merkleTree.MerkleTree.mtx.Unlock()

	node, ok := merkleTree.nodeMap.get(*hash)
	if !ok {
		return nil, fmt.Errorf("can't find hash %x in merkle tree", *hash)
	}
	hashes := []*MerkleHash{hash}
	proofHashes, proofPath := createMerkleProof(node)
	path := MerklePath{
		Hashes:    append(hashes, proofHashes...),
		ProofPath: proofPath,
	}
	if err := path.Verify(merkleTree.MerkleRoot); err != nil {
		return nil, err
		//panic(err)
	}
	return &path, nil
}

// Private function.
func countOne(value int64) int {
	numberOfOne := 0
	for value != 0 {
		numberOfOne++
		value = value & (value - 1)
	}
	return numberOfOne
}
