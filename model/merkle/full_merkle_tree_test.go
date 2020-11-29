/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/multivactech/MultiVAC/base/db"
)

var (
	n01, n02, n03, n04, n05, n06, n07, n08, n09, n10, n11 MerkleHash
	n0102, n01to04, n05to08, n09to11, n01to08, n01to11    *MerkleHash
)

func TestNewFullMerkleTree(t *testing.T) {
	createFullMerkleTree()
	merkleTree := NewFullMerkleTree(&n01, &n02, &n03, &n04, &n05, &n06, &n07, &n08, &n09, &n10, &n11)
	if *merkleTree.MerkleRoot != *n01to11 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
}

func TestFullMerkleTreeAppend(t *testing.T) {
	createFullMerkleTree()
	// Create an empty merkle tree.
	merkleTree := NewFullMerkleTree()
	err := merkleTree.Append(&n01, &n02, &n03, &n04, &n05, &n06)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.Append(&n07, &n08, &n09, &n10, &n11)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.MerkleRoot != *n01to11 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
}

func TestOnDiskFullMerkleTree_Append(t *testing.T) {
	createFullMerkleTree()
	// Create an empty merkle tree.
	dir := "testondiskfullmerkletree_append"
	err := os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
	tdb, _ := db.OpenDB(dir, "test")
	merkleTree := NewOnDiskFullMerkleTree(tdb)

	if err := merkleTree.Append(&n01); err != nil {
		t.Error(err)
	}
	if *merkleTree.MerkleRoot != n01 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
	if err := merkleTree.Append(&n02); err != nil {
		t.Error(err)
	}

	if *merkleTree.MerkleRoot != *n0102 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
	if err := merkleTree.Append(&n03, &n04); err != nil {
		t.Error(err)
	}
	if *merkleTree.MerkleRoot != *n01to04 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
	if err := merkleTree.Append(&n05, &n06, &n07, &n08); err != nil {
		t.Error(err)
	}
	if *merkleTree.MerkleRoot != *n01to08 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
	if err := merkleTree.Append(&n09, &n10, &n11); err != nil {
		t.Error(err)
	}
	if *merkleTree.MerkleRoot != *n01to11 {
		t.Errorf(
			"Full merkle tree is built wrongly, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}
	err = os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
}

func TestFullMerkleTreeUpdateLeaf(t *testing.T) {
	createFullMerkleTree()
	// Create an empty merkle tree.
	merkleTree := NewFullMerkleTree()
	err := merkleTree.Append(&n01, &n02, &n03, &n04)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n01, &n05)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n02, &n06)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n03, &n07)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n04, &n08)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.MerkleRoot != *n05to08 {
		t.Errorf(
			"Update leaf failed, expected merkle root %x, actual merkle root %x.",
			*n05to08,
			*merkleTree.MerkleRoot)
	}
}

func TestOnDiskFullMerkleTree_UpdateLeaf(t *testing.T) {
	createFullMerkleTree()
	// Create an empty merkle tree.
	dir := "testondiskfullmerkletree_updateleaf"
	err := os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
	tdb, _ := db.OpenDB(dir, "test")
	merkleTree := NewOnDiskFullMerkleTree(tdb)
	err = merkleTree.Append(&n01, &n02, &n03, &n04)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n01, &n05)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n02, &n06)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n03, &n07)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(&n04, &n08)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.MerkleRoot != *n05to08 {
		t.Errorf(
			"Update leaf failed, expected merkle root %x, actual merkle root %x.",
			*n05to08,
			*merkleTree.MerkleRoot)
	}
	err = os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
}

func TestGetMerklePath(t *testing.T) {
	createFullMerkleTree()
	merkleTree := NewFullMerkleTree(&n01, &n02, &n03, &n04, &n05, &n06, &n07, &n08, &n09, &n10, &n11)
	merklePath, _ := merkleTree.GetMerklePath(&n03)
	if *merklePath.Hashes[0] != n03 {
		t.Errorf("Wrong merkle path, the first hash isn't n3")
	}
	if err := merklePath.Verify(n01to11); err != nil {
		t.Errorf("Wrong merkle path, error message %s", err.Error())
	}

	n12 := MerkleHash(sha256.Sum256([]byte("n12")))
	if _, err := merkleTree.GetMerklePath(&n12); err == nil {
		t.Errorf("Wrong merkle path, n12 doesn't exist in merkle tree")
	}

	err := merkleTree.UpdateLeaf(&n03, &n09)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if _, err := merkleTree.GetMerklePath(&n03); err == nil {
		t.Errorf("Wrong merkle path, n03 has been replaced by 09")
	}
	if _, err := merkleTree.GetMerklePath(&n09); err != nil {
		t.Errorf("Failed to get merkle path for n09, err %v", err)
	}
}

func TestOnDiskMerkleTree_GetMerklePath(t *testing.T) {
	createFullMerkleTree()

	dir := "testondiskfullmerkletree_getmerklepath"
	err := os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
	tdb, _ := db.OpenDB(dir, "test")
	merkleTree := NewOnDiskFullMerkleTree(tdb)
	err = merkleTree.Append(&n01, &n02, &n03, &n04, &n05, &n06, &n07, &n08)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	merklePath, _ := merkleTree.GetMerklePath(&n03)
	if *merklePath.Hashes[0] != n03 {
		t.Errorf("Wrong merkle path, the first hash isn't n3")
	}
	if err := merklePath.Verify(n01to08); err != nil {
		t.Errorf("Wrong merkle path, error message %s", err.Error())
	}

	n12 := MerkleHash(sha256.Sum256([]byte("n12")))
	if _, err := merkleTree.GetMerklePath(&n12); err == nil {
		t.Errorf("Wrong merkle path, n12 doesn't exist in merkle tree")
	}

	err = merkleTree.UpdateLeaf(&n03, &n09)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if _, err := merkleTree.GetMerklePath(&n03); err == nil {
		t.Errorf("Wrong merkle path, n03 has been replaced by 09")
	}
	if _, err := merkleTree.GetMerklePath(&n09); err != nil {
		t.Errorf("Failed to get merkle path for n09, err %v", err)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
}

func TestHookSecondLayerTree(t *testing.T) {
	createFullMerkleTree()
	merkleTree := NewFullMerkleTree(n01to04, n05to08)
	subtree := NewFullMerkleTree(&n05, &n06, &n07, &n08)
	err := merkleTree.HookSecondLayerTree(subtree)
	if err != nil {
		log.Errorf("failed to hook second layer tree,err:%v", err)
	}

	// Verify the new merkle tree can still provide merkle path correctly
	merklePath, _ := merkleTree.GetMerklePath(&n06)
	if *merklePath.Hashes[0] != n06 {
		t.Errorf("Wrong merkle path, the first hash isn't n6")
	}
	if err := merklePath.Verify(n01to08); err != nil {
		t.Errorf("Wrong merkle path, error message %s", err.Error())
	}

	// Verify the new merkle tree can still append node correctly.
	err = merkleTree.Append(n09to11)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.MerkleRoot != *n01to11 {
		t.Errorf(
			"Failed to append node, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}

	n12 := MerkleHash(sha256.Sum256([]byte("n12")))
	err = merkleTree.UpdateLeaf(&n05, &n12)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if _, err := merkleTree.GetMerklePath(&n05); err == nil {
		t.Errorf("Wrong merkle path, n05 has been replaced by 12")
	}
	if _, err := merkleTree.GetMerklePath(&n12); err != nil {
		t.Errorf("Failed to get merkle path for n12, err %v", err)
	}
}

func TestOnDiskMerkleTree_HookSecondLayerTree(t *testing.T) {
	createFullMerkleTree()

	dir := "testondiskfullmerkletree_HookSecondLayerTree"
	err := os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
	tdb, _ := db.OpenDB(dir, "test")
	merkleTree := NewOnDiskFullMerkleTree(tdb)
	err = merkleTree.Append(n01to04, n05to08)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	subtree := NewFullMerkleTree()
	err = subtree.Append(&n05, &n06, &n07, &n08)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.HookSecondLayerTree(subtree)
	if err != nil {
		log.Errorf("failed to hook second layer tree,err:%v", err)
	}

	// Verify the new merkle tree can still provide merkle path correctly
	merklePath, _ := merkleTree.GetMerklePath(&n06)
	if *merklePath.Hashes[0] != n06 {
		t.Errorf("Wrong merkle path, the first hash isn't n6")
	}
	if err := merklePath.Verify(n01to08); err != nil {
		t.Errorf("Wrong merkle path, error message %s", err.Error())
	}

	// Verify the new merkle tree can still append node correctly.
	err = merkleTree.Append(n09to11)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.MerkleRoot != *n01to11 {
		t.Errorf(
			"Failed to append node, expected merkle root %x, actual merkle root %x.",
			*n01to11,
			*merkleTree.MerkleRoot)
	}

	n12 := MerkleHash(sha256.Sum256([]byte("n12")))

	err = merkleTree.UpdateLeaf(&n05, &n12)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if _, err := merkleTree.GetMerklePath(&n05); err == nil {
		t.Errorf("Wrong merkle path, n05 has been replaced by 12")
	}
	if _, err := merkleTree.GetMerklePath(&n12); err != nil {
		t.Errorf("Failed to get merkle path for n12, err %v", err)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		log.Errorf("failed to remove file,err:%v", err)
	}
}

func createFullMerkleTree() {
	n01 = MerkleHash(sha256.Sum256([]byte("n01")))
	n02 = MerkleHash(sha256.Sum256([]byte("n02")))
	n03 = MerkleHash(sha256.Sum256([]byte("n03")))
	n04 = MerkleHash(sha256.Sum256([]byte("n04")))
	n05 = MerkleHash(sha256.Sum256([]byte("n05")))
	n06 = MerkleHash(sha256.Sum256([]byte("n06")))
	n07 = MerkleHash(sha256.Sum256([]byte("n07")))
	n08 = MerkleHash(sha256.Sum256([]byte("n08")))
	n09 = MerkleHash(sha256.Sum256([]byte("n09")))
	n10 = MerkleHash(sha256.Sum256([]byte("n10")))
	n11 = MerkleHash(sha256.Sum256([]byte("n11")))

	n0102 = Hash(&n01, &n02, Right)
	n0304 := Hash(&n03, &n04, Right)
	n0506 := Hash(&n05, &n06, Right)
	n0708 := Hash(&n07, &n08, Right)
	n0910 := Hash(&n09, &n10, Right)

	n01to04 = Hash(n0102, n0304, Right)
	n05to08 = Hash(n0506, n0708, Right)
	n09to11 = Hash(n0910, &n11, Right)

	n01to08 = Hash(n01to04, n05to08, Right)

	n01to11 = Hash(n01to08, n09to11, Right)
}
