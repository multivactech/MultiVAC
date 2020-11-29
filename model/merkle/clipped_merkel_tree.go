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

// clippedMerkleTreeBuilder builds the clipped tree.
// These fields are intermediaries not used by the tree.
type clippedMerkleTreeBuilder struct {
	tree            *FullMerkleTree
	Size            int64
	StartIndex      int64
	NumLeaves       int64
	Leaves          []*MerkleHash
	LeftMerklePath  *MerklePath
	RightMerklePath *MerklePath
	leftDepth       int64
	rightDepth      int64
	useLeft         bool
	useRight        bool
}

// NewClippedMerkleTree create a clipped merkle tree given:
//   * size of total leaves.
//   * a list of leaves
//   * merkle path of the first element of the give leaves.
//   * merkle path of the last element of the given leaves.
func NewClippedMerkleTree(
	size int64, startIndex int64, numLeaves int64, leaves []*MerkleHash,
	leftMerklePath *MerklePath, rightMerklePath *MerklePath) (*FullMerkleTree, error) {
	if numLeaves <= 0 || len(leaves) == 0 {
		return nil, fmt.Errorf("At least one leaf is needed")
	}
	if numLeaves != int64(len(leaves)) {
		return nil, fmt.Errorf(
			"NumLeaves (%d) and length of Leaves (%d) doesn't match", numLeaves, len(leaves))
	}
	builder := &clippedMerkleTreeBuilder{
		Size:            size,
		StartIndex:      startIndex,
		NumLeaves:       numLeaves,
		Leaves:          leaves,
		LeftMerklePath:  leftMerklePath,
		RightMerklePath: rightMerklePath,
		leftDepth:       0,
		rightDepth:      0,
		useLeft:         false,
		useRight:        false,
	}
	return builder.build()
}

func (builder *clippedMerkleTreeBuilder) build() (*FullMerkleTree, error) {
	tree := &FullMerkleTree{*newEmptyRAMMerkleTree()}
	tree.Size = builder.Size
	tree.nodeHashChangedListener = tree
	builder.tree = tree

	var rightChild iMerkleNode
	numberOfNodes := tree.Size
	useLeft := false
	useRight := false
	// If the right most subtree has been built
	rightMostSubtreeBuilt := false
	for numberOfNodes > 0 {
		lsb := numberOfNodes & ^(numberOfNodes - 1)
		leftChild, childUseLeft, childUseRight := builder.buildPerfectSubtree(numberOfNodes-lsb, numberOfNodes)
		useLeft = useLeft || childUseLeft
		useRight = useRight || childUseRight
		if !rightMostSubtreeBuilt {
			// rigthchild must be nil
			if rightChild != nil {
				panic("First right tree shouldn't have been set.")
			}
			rightChild = leftChild
			rightMostSubtreeBuilt = true
		} else {
			rightChild = builder.createParentNode(leftChild, rightChild, useLeft, useRight)
		}
		numberOfNodes = numberOfNodes - lsb
	}
	if !(useLeft && useRight) {
		return nil, errors.New("Left merkle path and right merkle path must be used. There is a bug in the code")
	}
	tree.MerkleRoot = rightChild.getHash()
	tree.lastNodeHash = builder.Leaves[builder.NumLeaves-1]
	// TODO: verify left and right merkle path is right.
	return tree, nil
}

func (builder *clippedMerkleTreeBuilder) getLeave(index int64) *MerkleHash {
	index -= builder.StartIndex
	if index >= 0 && index < builder.NumLeaves {
		return builder.Leaves[index]
	}
	return nil
}

func (builder *clippedMerkleTreeBuilder) createParentNode(
	leftNode iMerkleNode, rightNode iMerkleNode, useLeft bool, useRight bool) iMerkleNode {
	if useLeft {
		builder.leftDepth++
	}
	if useRight {
		builder.rightDepth++
	}
	tree := builder.tree
	if leftNode == nil && rightNode == nil {
		return nil
	} else if leftNode == nil && rightNode != nil {
		// useLeft must be true
		leftHash := builder.LeftMerklePath.Hashes[builder.leftDepth]
		leftNode = tree.createNode(leftHash)
	} else if leftNode != nil && rightNode == nil {
		// useRight must be true
		rightHash := builder.RightMerklePath.Hashes[builder.rightDepth]
		rightNode = tree.createNode(rightHash)
	}
	node := tree.createNode(Hash(leftNode.getHash(), rightNode.getHash(), Right))
	node.setLeftChild(leftNode)
	node.setRightChild(rightNode)
	leftNode.setParent(node)
	rightNode.setParent(node)
	return node
}

// buildPerfectSubtree try to build the perfect sub tree for start and end.
// Returns the built node and if left most or right most clipped leaf has been seen.
func (builder *clippedMerkleTreeBuilder) buildPerfectSubtree(start int64, end int64) (iMerkleNode, bool, bool) {
	if end <= builder.StartIndex || start >= builder.StartIndex+builder.NumLeaves {
		return nil, false, false
	}
	size := end - start
	var leftNode iMerkleNode
	var rightNode iMerkleNode
	var node iMerkleNode = nil
	useLeft := false
	useRight := false
	if size == 1 {
		if start == builder.StartIndex {
			useLeft = true
		}
		if builder.StartIndex+builder.NumLeaves == end {
			useRight = true
		}
		leafHash := builder.getLeave(start)
		if leafHash != nil {
			node = builder.tree.createNode(leafHash)
		}
		return node, useLeft, useRight
	}
	leftNode, leftUseLeft, leftUseRight := builder.buildPerfectSubtree(start, start+(size>>1))
	rightNode, rightUseLeft, rightUseRight := builder.buildPerfectSubtree(start+size>>1, end)
	useLeft = leftUseLeft || rightUseLeft
	useRight = leftUseRight || rightUseRight
	return builder.createParentNode(leftNode, rightNode, useLeft, useRight), useLeft, useRight
}
