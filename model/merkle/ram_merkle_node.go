// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package merkle

func createRAMMerkleNode() iMerkleNode {
	return &ramMerkleNode{}
}

type ramMerkleNode struct {
	parent     iMerkleNode
	leftChild  iMerkleNode
	rightChild iMerkleNode
	hash       *MerkleHash
}

func (mn *ramMerkleNode) setParent(node iMerkleNode) {
	mn.parent = node
}

func (mn *ramMerkleNode) getParent() iMerkleNode {
	return mn.parent
}

func (mn *ramMerkleNode) setLeftChild(node iMerkleNode) {
	mn.leftChild = node
}

func (mn *ramMerkleNode) getLeftChild() iMerkleNode {
	return mn.leftChild
}

func (mn *ramMerkleNode) setRightChild(node iMerkleNode) {
	mn.rightChild = node
}

func (mn *ramMerkleNode) getRightChild() iMerkleNode {
	return mn.rightChild
}

func (mn *ramMerkleNode) setHash(hash *MerkleHash) {
	mn.hash = hash
}

func (mn *ramMerkleNode) getHash() *MerkleHash {
	return mn.hash
}

func (mn *ramMerkleNode) getID() uint64 {
	return 0
}

func (mn *ramMerkleNode) isEqualTo(node iMerkleNode) bool {
	if n, ok := node.(*ramMerkleNode); ok {
		return mn == n
	}
	return false

}
