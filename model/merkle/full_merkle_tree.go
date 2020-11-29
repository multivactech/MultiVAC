// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package merkle

import (
	"errors"
	"fmt"

	"github.com/multivactech/MultiVAC/base/db"
)

// FullMerkleTree is the data structure of full merkle tree.
type FullMerkleTree struct {
	MerkleTree
}

// NewFullMerkleTree create a full merkle tree given all the leaves. This will be used to build merkle tree
// for outs in a block.
func NewFullMerkleTree(leaves ...*MerkleHash) *FullMerkleTree {
	merkleTree := &FullMerkleTree{*newEmptyRAMMerkleTree()}
	merkleTree.init(leaves...)
	return merkleTree
}

// NewOnDiskFullMerkleTree create a full merkle tree given all the leaves. Data in this merkle tree is saved on disk.
func NewOnDiskFullMerkleTree(db db.DB) *FullMerkleTree {
	merkleTree := &FullMerkleTree{*newDiskMerkleTree(db)}
	merkleTree.init()
	return merkleTree
}

func (merkleTree *FullMerkleTree) init(leaves ...*MerkleHash) {
	merkleTree.nodeHashChangedListener = merkleTree
	if len(leaves) == 0 {
		return
	}
	merkleTree.Size = int64(len(leaves))
	var node iMerkleNode = nil
	numberOfNodes := merkleTree.Size
	for numberOfNodes > 0 {
		lsb := numberOfNodes & ^(numberOfNodes - 1)
		// Only in the first round the lsb could be 1. And when lsb is 1, node must be nil.
		if lsb == 1 {
			node = merkleTree.createNode(leaves[numberOfNodes-1])
		} else {
			subTreeRoot := merkleTree.buildPerfectSubtree(leaves[numberOfNodes-lsb : numberOfNodes]...)
			if node == nil {
				node = subTreeRoot
			} else {
				newRoot := merkleTree.createNode(Hash(subTreeRoot.getHash(), node.getHash(), Right))
				newRoot.setLeftChild(subTreeRoot)
				newRoot.setRightChild(node)
				subTreeRoot.setParent(newRoot)
				node.setParent(newRoot)
				node = newRoot
			}
		}
		numberOfNodes = numberOfNodes - lsb
	}
	merkleTree.MerkleRoot = node.getHash()
	merkleTree.lastNodeHash = leaves[merkleTree.Size-1]
}

// Build a subtree whose size is a power of 2 and equal or larger than 2, e.g. 2, 4, or 16.
func (merkleTree *FullMerkleTree) buildPerfectSubtree(leaves ...*MerkleHash) iMerkleNode {
	size := len(leaves)
	var leftNode iMerkleNode
	var rightNode iMerkleNode
	var node iMerkleNode
	if size < 2 {
		// This method is recursively called. If the size is smaller than 2, that means the
		// size of original leaves isn't a power of 2.
		panic("Size of leaves isn't power of 2")
	} else if size == 2 {
		leftNode = merkleTree.createNode(leaves[0])
		rightNode = merkleTree.createNode(leaves[1])
	} else {
		leftNode = merkleTree.buildPerfectSubtree(leaves[0:(size >> 1)]...)
		rightNode = merkleTree.buildPerfectSubtree(leaves[(size >> 1):]...)
	}
	node = merkleTree.createNode(Hash(leftNode.getHash(), rightNode.getHash(), Right))
	node.setLeftChild(leftNode)
	node.setRightChild(rightNode)
	leftNode.setParent(node)
	rightNode.setParent(node)
	return node
}

func (merkleTree *FullMerkleTree) createNode(hash *MerkleHash) iMerkleNode {
	node := merkleTree.MerkleTree.newNode()
	node.setHash(hash)
	merkleTree.nodeMap.set(*hash, node)
	return node
}

// UpdateLeaf update the leaf and merkle root.
func (merkleTree *FullMerkleTree) UpdateLeaf(originalHash *MerkleHash, newHash *MerkleHash) error {
	merkleTree.MerkleTree.mtx.Lock()
	defer merkleTree.MerkleTree.mtx.Unlock()
	node, ok := merkleTree.nodeMap.get(*originalHash)
	if !ok {
		return fmt.Errorf("can't find hash %x in merkle tree", *originalHash)
	}
	merkleTree.updateNode(node, newHash)

	merkleTree.updateMerkleRoot()
	return nil
}

// Append is to add new leaves and update merkle tree's root.
func (merkleTree *FullMerkleTree) Append(leaves ...*MerkleHash) error {
	merkleTree.MerkleTree.mtx.Lock()
	defer merkleTree.MerkleTree.mtx.Unlock()

	err := merkleTree.MerkleTree.append(leaves...)
	if err != nil {
		return err
	}
	merkleTree.updateMerkleRoot()
	return nil
}

func (merkleTree *FullMerkleTree) onNodeHashChanged(node iMerkleNode, originalHash *MerkleHash, newHash *MerkleHash) {
	if *originalHash == *newHash {
		return
	}
	merkleTree.nodeMap.set(*newHash, node)
	merkleTree.nodeMap.del(*originalHash)

	if merkleTree.lastNodeHash != nil && *merkleTree.lastNodeHash == *originalHash {
		merkleTree.lastNodeHash = newHash
	}
}

func (merkleTree *FullMerkleTree) updateMerkleRoot() {
	node, exist := merkleTree.nodeMap.get(*merkleTree.lastNodeHash)
	if !exist {
		log.Infof("updating root, last node hash %v\n", merkleTree.lastNodeHash)
	}
	for node.getParent() != nil {
		node = node.getParent()
	}
	merkleTree.MerkleRoot = node.getHash()
}

// HookSecondLayerTree is to build a two layer merkle tree. A leaf node of top layer tree is root
// node of a second layer tree.
func (merkleTree *FullMerkleTree) HookSecondLayerTree(subtree *FullMerkleTree) error {
	merkleTree.MerkleTree.mtx.Lock()
	defer merkleTree.MerkleTree.mtx.Unlock()

	if subtree == nil {
		return errors.New("Subtree is nil")
	}
	var node iMerkleNode
	var ok bool
	var subtreeRoot iMerkleNode
	node, ok = merkleTree.nodeMap.get(*subtree.MerkleRoot)
	if !ok {
		return errors.New("Can't find subtree's merkle root in top layer tree")
	}
	subtreeRoot, ok = subtree.nodeMap.get(*subtree.MerkleRoot)
	if !ok {
		return errors.New("Can't find subtree's merkle root in sub tree")
	}
	if subtreeRoot.getLeftChild() != nil && subtreeRoot.getRightChild() != nil {
		leftNode := merkleTree.deepcopyNode(subtreeRoot.getLeftChild())
		rightNode := merkleTree.deepcopyNode(subtreeRoot.getRightChild())
		node.setLeftChild(leftNode)
		node.setRightChild(rightNode)
		leftNode.setParent(node)
		rightNode.setParent(node)
	}
	return nil
}

func (merkleTree *FullMerkleTree) deepcopyNode(node iMerkleNode) iMerkleNode {
	newNode := merkleTree.MerkleTree.newNode()
	newNode.setHash(node.getHash())
	if node.getLeftChild() != nil && node.getRightChild() != nil {
		leftNode := merkleTree.deepcopyNode(node.getLeftChild())
		rightNode := merkleTree.deepcopyNode(node.getRightChild())
		newNode.setLeftChild(leftNode)
		newNode.setRightChild(rightNode)
		leftNode.setParent(newNode)
		rightNode.setParent(newNode)
	} else if node.getLeftChild() != nil || node.getRightChild() != nil {
		panic("If any child is nil, the other child should be nil too")
	}
	merkleTree.nodeMap.set(*node.getHash(), newNode)
	return newNode
}

// GetMerklePath returns the merkle path.
func (merkleTree *FullMerkleTree) GetMerklePath(hash *MerkleHash) (*MerklePath, error) {
	merkleTree.MerkleTree.mtx.Lock()
	node, ok := merkleTree.nodeMap.get(*hash)
	merkleTree.MerkleTree.mtx.Unlock()

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

// HasNode checks whether there is the node in full merkle tree.
func (merkleTree *FullMerkleTree) HasNode(hash *MerkleHash) bool {
	merkleTree.MerkleTree.mtx.Lock()
	defer merkleTree.MerkleTree.mtx.Unlock()

	_, ok := merkleTree.nodeMap.get(*hash)
	return ok
}
