package merkle

import (
	"crypto/sha256"
	"testing"
)

var (
	leafPool []MerkleHash
)

// Utils for debugging.
// it seems unused.
//func PrintFullMerkleTree(t *testing.T, tree *FullMerkleTree) {
//	root, exists := tree.nodeMap.get(*tree.MerkleRoot)
//	if !exists {
//		t.Error("Root doesn't exists")
//	}
//	PrintMerkleTreeRoot(t, root)
//}

// it seems unused.
//func PrintClippedMerkleTree(t *testing.T, tree *FullMerkleTree) {
//	root, exists := tree.nodeMap.get(*tree.MerkleRoot)
//	if !exists {
//		t.Error("Root doesn't exists")
//	}
//	PrintMerkleTreeRoot(t, root)
//}

// it seems unused.
//func PrintMerkleTreeRoot(t *testing.T, root iMerkleNode) {
//	left := root.getLeftChild()
//	right := root.getRightChild()
//	if left != nil {
//		t.Logf("%s -> %s, %s", root.getHash(), left.getHash(), right.getHash())
//		PrintMerkleTreeRoot(t, left)
//		PrintMerkleTreeRoot(t, right)
//	} else {
//		t.Logf("%s -> nil, nil", root.getHash())
//	}
//}

func buildAndVerifyClippedMerkleTreeForRange(t *testing.T, fullTree *FullMerkleTree, leaves []*MerkleHash, start int64, end int64) {
	leftMerklePath, err := fullTree.GetMerklePath(leaves[start])
	if err != nil {
		t.Error(err)
	}
	rightMerklePath, err := fullTree.GetMerklePath(leaves[end-1])
	if err != nil {
		t.Error(err)
	}
	clippedTree, err := NewClippedMerkleTree(int64(len(leaves)), start, end-start, leaves[start:end], leftMerklePath, rightMerklePath)
	if err != nil {
		t.Errorf("Failed to build clipped merkle tree: %v", err)
	}

	// t.Logf("Leaves: %v", leaves[start:end])
	// t.Logf("Left MH: %v", leftMerklePath.Hashes)
	// t.Logf("Right MH: %v", rightMerklePath.Hashes)
	// t.Log("Full Tree")
	// PrintFullMerkleTree(t, fullTree)
	// t.Log("Clipped Tree")
	// PrintClippedMerkleTree(t, clippedTree)

	if *clippedTree.MerkleRoot != *fullTree.MerkleRoot {
		t.Errorf(
			"Clipped tree root for [%d:%d] of full tree size %d does not match full tree root: %s vs %s",
			start, end, fullTree.Size, clippedTree.MerkleRoot, fullTree.MerkleRoot)
	}
}

func buildAndVerifyAllClippedTreesForShape(t *testing.T, fullTreeSize int) {
	leaves := []*MerkleHash{}
	for i := 0; i < fullTreeSize; i++ {
		leaves = append(leaves, &leafPool[i])
	}

	fullTree := NewFullMerkleTree(leaves...)
	var start int64
	var end int64
	for start = 0; start < int64(fullTreeSize)-1; start++ {
		for end = start + 1; end < int64(fullTreeSize); end++ {
			buildAndVerifyClippedMerkleTreeForRange(t, fullTree, leaves, start, end)
		}
	}
}

// Test all possible clipped trees of all possible trees of max 0xF leaves.
// This test depends on the correctness of FullMerkleTree.
func TestNewClippedMerkleTree(t *testing.T) {
	// Generate 0xF leaves in the pool
	poolSize := 0xF
	for i := 0; i < poolSize; i++ {
		leafPool = append(leafPool, MerkleHash(sha256.Sum256([]byte{byte(i)})))
	}
	for fullTreeSize := 1; fullTreeSize <= poolSize; fullTreeSize++ {
		buildAndVerifyAllClippedTreesForShape(t, fullTreeSize)
	}
}
