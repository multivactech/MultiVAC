// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package merkle

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/multivactech/MultiVAC/base/db"
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/util"
)

const (
	// HashSize defines the length of hash array.
	HashSize = 32
	// Left points to the left subtree path of the merkle tree.
	Left byte = 0
	// Right points to the right subtree path of the merkle tree.
	Right byte = 1
	// UnknownSize means size of merkle tree is unknown.
	UnknownSize = -1
)

// EmptyHash defines a default merkle hash.
var EmptyHash = MerkleHash{}

// MerkleHash defines the data structure for the leaves and root fo the merkle tree.
// TODO: merge chainhash.Hash and MerkleHash
type MerkleHash [HashSize]byte

// String returns the Hash as the hexadecimal string of the byte-reversed
// hash.
func (hash MerkleHash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

// IsEmptyHash check whether the merkle hash is empty.
func (hash *MerkleHash) IsEmptyHash() bool {
	for _, b := range *hash {
		if b != 0 {
			return false
		}
	}
	return true
}

// Equal returns whether x is length same and value same to hash.
func (hash *MerkleHash) Equal(x *MerkleHash) bool {
	return bytes.Equal(hash[:], x[:])
}

// RootHashMismatchError defines the detail of root hash mismatch error.
type RootHashMismatchError struct {
	ExpectedHash *MerkleHash
	ActualHash   *MerkleHash
}

func (err RootHashMismatchError) Error() string {
	return fmt.Sprintf(
		"RootHashMismatchError, expected hash %v, actual hash %v",
		*err.ExpectedHash,
		*err.ActualHash)
}

// MerklePath is a series of merkle hash from a merkle leaf to merkle root.
type MerklePath struct {
	// The first hash is the leaf hash. Rest of them are merkle proof.
	// Note. This slice should always have at least one item.
	Hashes []*MerkleHash
	// Path is an array of position, either left or right. Left = 0 and Right = 1
	ProofPath []byte
}

type iMerkleNode interface {
	setParent(node iMerkleNode)
	getParent() iMerkleNode
	setLeftChild(node iMerkleNode)
	getLeftChild() iMerkleNode
	setRightChild(node iMerkleNode)
	getRightChild() iMerkleNode
	setHash(hash *MerkleHash)
	getHash() *MerkleHash
	getID() uint64
	isEqualTo(node iMerkleNode) bool
}

type iNodeMap interface {
	set(key MerkleHash, value iMerkleNode)
	get(key MerkleHash) (iMerkleNode, bool)
	del(key MerkleHash)
}

// MerkleTree has an assumption that there's never two nodes have same hash value.
type MerkleTree struct {
	nodeMap                 iNodeMap
	Size                    int64
	MerkleRoot              *MerkleHash
	lastNodeHash            *MerkleHash
	nodeHashChangedListener nodeHashChangedListener
	mtx                     sync.RWMutex
	newNode                 func() iMerkleNode
}

type nodeHashChangedListener interface {
	onNodeHashChanged(node iMerkleNode, originalHash *MerkleHash, hash *MerkleHash)
}

// Verify this merkle path's merkle root is same as expected hash.
func (merklePath *MerklePath) Verify(expectedHash *MerkleHash) error {
	if len(merklePath.ProofPath)+1 != len(merklePath.Hashes) {
		return errors.New("length of path not equal to length of hashes")
	}
	actualHash := merklePath.Hashes[0]
	proof := merklePath.Hashes[1:]
	for index, pos := range merklePath.ProofPath {
		actualHash = Hash(actualHash, proof[index], pos)
	}
	if *actualHash == *expectedHash {
		return nil
	} else {
		return RootHashMismatchError{
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
		}
	}
}

// GetLeaf returns the leaf node of the merkle path.
func (merklePath *MerklePath) GetLeaf() *MerkleHash {
	return merklePath.Hashes[0]
}

// Append leaf nodes into merkle tree. To do this, it needs to know what's the size of current merkle tree and what's the
// last node's merkle path.
func (merkleTree *MerkleTree) append(leafHashes ...*MerkleHash) error {
	if merkleTree.Size == UnknownSize {
		return errors.New("merkle tree's size and last node are unknown")
	}

	size := merkleTree.Size
	var lastNode iMerkleNode
	if size > 0 {
		lastNode, _ = merkleTree.nodeMap.get(*merkleTree.lastNodeHash)
	}

	for _, hashToInsert := range leafHashes {
		nodeToInsert := merkleTree.newNode()
		nodeToInsert.setHash(hashToInsert)

		merkleTree.nodeMap.set(*hashToInsert, nodeToInsert)
		if size == 0 {
			merkleTree.MerkleRoot = hashToInsert
		} else {
			leastSignificantBit := size & ^(size - 1)
			depthToReplace := countOne(leastSignificantBit - 1)

			nodeToReplace := lastNode
			for i := 0; i < depthToReplace; i++ {
				nodeToReplace = nodeToReplace.getParent()
			}

			// Create a new Node, and put it in the position of nodeToReplace
			newJointNodeHash := Hash(hashToInsert, nodeToReplace.getHash(), Left)

			newJointNode := merkleTree.newNode()
			newJointNode.setHash(newJointNodeHash)
			pn := nodeToReplace.getParent()
			newJointNode.setParent(pn)
			newJointNode.setLeftChild(nodeToReplace)
			newJointNode.setRightChild(nodeToInsert)

			merkleTree.nodeMap.set(*newJointNodeHash, newJointNode)

			nodeToInsert.setParent(newJointNode)
			nodeToReplace.setParent(newJointNode)
			if pn != nil {
				pn.setRightChild(newJointNode)
			}

			merkleTree.updateNode(newJointNode, newJointNodeHash)
		}

		size = size + 1
		lastNode = nodeToInsert
	}
	merkleTree.Size = size
	merkleTree.lastNodeHash = lastNode.getHash()
	return nil
}

// GetLastNodeMerklePath returns the merkle path of the last node.
func (merkleTree *MerkleTree) GetLastNodeMerklePath() (*MerklePath, error) {
	merkleTree.mtx.Lock()
	defer merkleTree.mtx.Unlock()

	if merkleTree.lastNodeHash == nil {
		return nil, fmt.Errorf("LastNode unknown")
	}
	lastNode, _ := merkleTree.nodeMap.get(*merkleTree.lastNodeHash)
	hashes := []*MerkleHash{lastNode.getHash()}
	proofHashes, proofPath := createMerkleProof(lastNode)
	return &MerklePath{
		Hashes:    append(hashes, proofHashes...),
		ProofPath: proofPath,
	}, nil
}

// GetLastNodeHash returns the last node hash.
func (merkleTree *MerkleTree) GetLastNodeHash() *MerkleHash {
	return merkleTree.lastNodeHash
}

func newEmptyRAMMerkleTree() *MerkleTree {
	return &MerkleTree{
		nodeMap:    createRAMNodeMap(),
		Size:       0,
		MerkleRoot: nil,
		newNode:    createRAMMerkleNode,
	}
}

func getMerkleTreeInfoFromDB(db db.DB) (int64, *MerkleHash, *MerkleHash, error) {
	var err error
	var size int64

	sizeRawData, err := db.Get([]byte(util.KeyLedgertreeSize))
	if err != nil {
		log.Errorf("get LedgerTreeSize from db err: %v. If it's first start, can ignore this error info ", err)
	}
	size = util.BytesToInt64(sizeRawData)

	rootRawData, err := db.Get([]byte(util.KeyLedgertreeRoot))
	if err != nil {
		log.Errorf("get LedgerTreeRoot from db err: %v. If it's first start, can ignore this error info ", err)
	}

	root := &MerkleHash{}
	err = rlp.DecodeBytes(rootRawData, root)
	if err != nil {
		log.Errorf("DecodeBytes err: %v. If it's first start, can ignore this error info ", err)
	}

	lastNodeHashRawData, err := db.Get([]byte(util.KeyLedgertreeLastnodehash))
	if err != nil {
		log.Errorf("get LedgerTreeLastNodeHash from db err: %v. If it's first start, can ignore this error info ", err)
	}

	lastNodeHash := &MerkleHash{}
	err = rlp.DecodeBytes(lastNodeHashRawData, lastNodeHash)
	if err != nil {
		log.Errorf("DecodeBytes err: %v. If it's first start, can ignore this error info ", err)
	}
	return size, root, lastNodeHash, err
}

// Create a new disk merkle tree.
// It tries to restore merkle tree from db first, and gets merkle tree's size, root, and last node hash
// from db. If there's no data or it fails to read data, then it will create a new empty merkle tree.
func newDiskMerkleTree(db db.DB) *MerkleTree {
	size, root, lastNodeHash, err := getMerkleTreeInfoFromDB(db)
	if err != nil {
		size = 0
		root = nil
		lastNodeHash = nil
	}
	log.Infof("newDiskMerkleTree,size:%d,root:%v,lastnodeHash:%v", size, root, lastNodeHash)
	factory := newDiskMerkleFactory(db)
	return &MerkleTree{
		nodeMap:      factory.getNodeMap(),
		Size:         size,
		MerkleRoot:   root,
		lastNodeHash: lastNodeHash,
		newNode:      factory.newNode,
	}
}

func (merkleTree *MerkleTree) updateNode(node iMerkleNode, newHash *MerkleHash) {
	merkleTree.setNodeHash(node, newHash)
	node = node.getParent()
	for node != nil {
		newHash := Hash(node.getLeftChild().getHash(), node.getRightChild().getHash(), Right)
		merkleTree.setNodeHash(node, newHash)
		node = node.getParent()
	}
}

func (merkleTree *MerkleTree) setNodeHash(node iMerkleNode, newHash *MerkleHash) {
	originalHash := node.getHash()
	pn := node.getParent()
	var isLeftChild bool
	if pn != nil && pn.getLeftChild().isEqualTo(node) {
		isLeftChild = true
	} else {
		isLeftChild = false
	}
	node.setHash(newHash)
	lc := node.getLeftChild()
	if lc != nil {
		lc.setParent(node)
	}
	rc := node.getRightChild()
	if rc != nil {
		rc.setParent(node)
	}
	if pn != nil {
		if isLeftChild {
			pn.setLeftChild(node)
		} else {
			pn.setRightChild(node)
		}
	}

	if merkleTree.nodeHashChangedListener != nil {
		merkleTree.nodeHashChangedListener.onNodeHashChanged(node, originalHash, newHash)
	}
}

func (merkleTree *MerkleTree) String(depth int) string {
	node, _ := merkleTree.nodeMap.get(*merkleTree.lastNodeHash)
	if node == nil {
		return "(empty tree or last node is null)"
	}
	pnode := node.getParent()
	for pnode != nil {
		node = pnode
		pnode = node.getParent()
	}
	var nodeToString func(node iMerkleNode, depth int) string
	nodeToString = func(node iMerkleNode, depth int) string {
		r := "("
		if node.getLeftChild() != nil && depth > 0 {
			r += nodeToString(node.getLeftChild(), depth-1)
		}
		r += fmt.Sprintf(" %v ", node.getHash())
		if node.getRightChild() != nil && depth > 0 {
			r += nodeToString(node.getRightChild(), depth-1)
		}
		r += ")"
		return r
	}
	return nodeToString(node, depth)
}

// Create merkle proof. This merkle proof doesn't include the node itself.
func createMerkleProof(node iMerkleNode) ([]*MerkleHash, []byte) {
	// Precondition: this node isn't nil and exists in this tree.
	hashes := []*MerkleHash{}
	path := []byte{}

	pnode := node.getParent()
	for pnode != nil {
		lnode := pnode.getLeftChild()
		rnode := pnode.getRightChild()
		// If a node exists in delta merkle tree, and it isn't the root node, then its parent node and
		// sibling node must exists in delta merkle tree as well.
		if lnode.isEqualTo(node) {
			// Its sibling is on the right side of the tree.
			path = append(path, Right)
			hashes = append(hashes, rnode.getHash())
		} else {
			// Its sibling is on the left side of the tree.
			path = append(path, Left)
			hashes = append(hashes, lnode.getHash())
		}
		node = pnode
		pnode = node.getParent()
	}
	return hashes, path
}
