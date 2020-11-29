/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"crypto/sha256"
	"errors"
	"os"
	"testing"
)

var (
	a, b, c, d, e, f, g, h                     MerkleHash
	ab, cd, abcd, ae, aecd, fd, aefd, gd, aegd *MerkleHash
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	createSimpleMerkleTree()
}

func TestUpdateLeaf(t *testing.T) {
	merklePathForB := MerklePath{
		Hashes:    []*MerkleHash{&b, &a, cd},
		ProofPath: []byte{Left, Right},
	}

	merklePathForC := MerklePath{
		Hashes:    []*MerkleHash{&c, &d, ab},
		ProofPath: []byte{Right, Left},
	}

	merkleTree := NewDeltaMerkleTree(abcd)
	// Replace b with e.
	err := merkleTree.UpdateLeaf(&merklePathForB, &e)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *aecd {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", aecd, merkleTree.GetMerkleRoot())
	}
	// Then replace c with f. The merkle path is old merkle path before b replaced with e.
	err = merkleTree.UpdateLeaf(&merklePathForC, &f)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *aefd {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", aefd, merkleTree.GetMerkleRoot())
	}

	// Replace c with g again. Merkle tree itself doesn't forbidden update a node twice. If the caller doesn't want
	// a node be updated more than once, then it's the caller's job to check that.
	err = merkleTree.UpdateLeaf(&merklePathForC, &g)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *aegd {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", aefd, merkleTree.GetMerkleRoot())
	}
}

func TestUpdateInvalidLeafRejected(t *testing.T) {
	// Invalid merle path, Hashes should be &b, &a, cd.
	merklePathForB := MerklePath{
		Hashes:    []*MerkleHash{&b, &c, cd},
		ProofPath: []byte{Left, Right},
	}

	// Invalid merle path, ProofPath should be Right, Left.
	merklePathForC := MerklePath{
		Hashes:    []*MerkleHash{&c, &d, ab},
		ProofPath: []byte{Left, Left},
	}

	merkleTree := NewDeltaMerkleTree(abcd)
	// Replace b with e. Should be rejected.
	err := merkleTree.UpdateLeaf(&merklePathForB, &e)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcd {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", aecd, merkleTree.GetMerkleRoot())
	}
	// Replace c with f. Should be rejected.
	err = merkleTree.UpdateLeaf(&merklePathForC, &f)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcd {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", aefd, merkleTree.GetMerkleRoot())
	}
}

func TestRefreshMerklePath(t *testing.T) {
	merklePathForB := MerklePath{
		Hashes:    []*MerkleHash{&b, &a, cd},
		ProofPath: []byte{Left, Right},
	}
	merkleTree := NewDeltaMerkleTree(abcd)
	// Replace b with e.
	err := merkleTree.UpdateLeaf(&merklePathForB, &e)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	// Test refresh merkle path for a. a is in b's merkle proof so it's in the merkle tree.
	merklePathForA := MerklePath{
		Hashes:    []*MerkleHash{&a, &b, cd},
		ProofPath: []byte{Right, Right},
	}
	expectedMerklePathForAAfterUpdate := MerklePath{
		Hashes:    []*MerkleHash{&a, &e, cd},
		ProofPath: []byte{Right, Right},
	}
	newMerklePathForA, _ := merkleTree.RefreshMerklePath(&merklePathForA)
	err = checkMerklePathEquals(&expectedMerklePathForAAfterUpdate, newMerklePathForA)
	if err != nil {
		t.Errorf("Failed to refresh merkle path for a, error msg: %s", err.Error())
	}

	merklePathForC := MerklePath{
		Hashes:    []*MerkleHash{&c, &d, ab},
		ProofPath: []byte{Right, Left},
	}
	expectedMerklePathForCAfterUpdate := MerklePath{
		Hashes:    []*MerkleHash{&c, &d, ae},
		ProofPath: []byte{Right, Left},
	}
	newMerklePathForC, _ := merkleTree.RefreshMerklePath(&merklePathForC)
	err = checkMerklePathEquals(&expectedMerklePathForCAfterUpdate, newMerklePathForC)
	if err != nil {
		t.Errorf("Failed to refresh merkle path for c, error msg: %s", err.Error())
	}
}

func TestAppend(t *testing.T) {
	merkleTree := NewDeltaMerkleTree(abcd)
	// d is the last node in the tree.
	merklePathForD := &MerklePath{
		Hashes:    []*MerkleHash{&d, &c, ab},
		ProofPath: []byte{Left, Left},
	}
	// Expected hash value after append e into merkle tree.
	abcde := Hash(abcd, &e, Right)
	err := merkleTree.SetSizeAndLastNodeMerklePath(4, merklePathForD)
	if err != nil {
		log.Errorf("failed to set size and last node merkle path,err:%v", err)
	}
	err = merkleTree.Append(&e)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcde {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcde, *merkleTree.GetMerkleRoot())
	}

	ef := Hash(&e, &f, Right)
	abcdef := Hash(abcd, ef, Right)

	// Size and merkle path for last node is only useful the first time. They are ignored afterwards.
	err = merkleTree.Append(&f)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcdef {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcdef, *merkleTree.GetMerkleRoot())
	}

	efg := Hash(ef, &g, Right)
	abcdefg := Hash(abcd, efg, Right)
	err = merkleTree.Append(&g)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcdefg {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcdefg, *merkleTree.GetMerkleRoot())
	}

	gh := Hash(&g, &h, Right)
	efgh := Hash(ef, gh, Right)
	abcdefgh := Hash(abcd, efgh, Right)
	err = merkleTree.Append(&h)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcdefgh {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcdefgh, *merkleTree.GetMerkleRoot())
	}
}

func TestAppendMoreThanOneNodeTogether(t *testing.T) {
	merkleTree := NewDeltaMerkleTree(abcd)
	// d is the last node in the tree.
	merklePathForD := &MerklePath{
		Hashes:    []*MerkleHash{&d, &c, ab},
		ProofPath: []byte{Left, Left},
	}
	ef := Hash(&e, &f, Right)
	efg := Hash(ef, &g, Right)
	abcdefg := Hash(abcd, efg, Right)
	err := merkleTree.SetSizeAndLastNodeMerklePath(4, merklePathForD)
	if err != nil {
		log.Errorf("failed to set size and last node merkle path,err:%v", err)
	}
	err = merkleTree.Append(&e, &f, &g)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcdefg {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcdefg, *merkleTree.GetMerkleRoot())
	}
}

func TestAppendAfterUpdateLastNode(t *testing.T) {
	abcde := Hash(abcd, &e, Right)
	merkleTree := NewDeltaMerkleTree(abcde)
	// d is the last node in the tree.
	merklePathForE := &MerklePath{
		Hashes:    []*MerkleHash{&e, abcd},
		ProofPath: []byte{Left},
	}
	r := MerkleHash(sha256.Sum256([]byte("r")))
	// Update the merkle tree to abcdr
	err := merkleTree.UpdateLeaf(merklePathForE, &r)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	i := MerkleHash(sha256.Sum256([]byte("i")))
	rf := Hash(&r, &f, Right)
	gh := Hash(&g, &h, Right)
	rfgh := Hash(rf, gh, Right)
	abcdrfgh := Hash(abcd, rfgh, Right)
	abcdrfghi := Hash(abcdrfgh, &i, Right)

	err = merkleTree.SetSizeAndLastNodeMerklePath(5, merklePathForE)
	if err != nil {
		log.Errorf("failed to set size and lost node merkle path,err:%v", err)
	}
	err = merkleTree.Append(&f)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.Append(&g, &h)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.Append(&i)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	if *merkleTree.GetMerkleRoot() != *abcdrfghi {
		t.Errorf("The new merkle root is wrong, expected root %x, actual root %x", *abcdrfghi, *merkleTree.GetMerkleRoot())
	}
}

func TestRefreshMerklePathAfterAppend(t *testing.T) {
	merkleTree := NewDeltaMerkleTree(abcd)
	// d is the last node in the tree.
	merklePathForD := &MerklePath{
		Hashes:    []*MerkleHash{&d, &c, ab},
		ProofPath: []byte{Left, Left},
	}
	// Update the merkle tree to abch
	err := merkleTree.UpdateLeaf(merklePathForD, &h)
	if err != nil {
		log.Errorf("failed to update leaf,err:%v", err)
	}
	ch := Hash(&c, &h, Right)
	ef := Hash(&e, &f, Right)
	efg := Hash(ef, &g, Right)
	err = merkleTree.SetSizeAndLastNodeMerklePath(4, merklePathForD)
	if err != nil {
		log.Errorf("failed to set size and lost node merkle path,err:%v", err)
	}
	err = merkleTree.Append(&e)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	err = merkleTree.Append(&f, &g)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	originalMerklePathForB := &MerklePath{
		Hashes:    []*MerkleHash{&b, &a, cd},
		ProofPath: []byte{Left, Right},
	}
	expectedMerklePathForB := &MerklePath{
		Hashes:    []*MerkleHash{&b, &a, ch, efg},
		ProofPath: []byte{Left, Right, Right},
	}
	newMerklePathForB, _ := merkleTree.RefreshMerklePath(originalMerklePathForB)

	err = checkMerklePathEquals(expectedMerklePathForB, newMerklePathForB)
	if err != nil {
		t.Errorf("Failed to refresh merkle path for b, error msg: %s", err.Error())
	}
}

func TestGetLastNodeMerklePathAfterAppend(t *testing.T) {
	var lastNodePath *MerklePath
	var err error
	merkleTree := NewDeltaMerkleTree(&a)
	merklePathForA := &MerklePath{
		Hashes:    []*MerkleHash{&a},
		ProofPath: []byte{},
	}
	err = merkleTree.SetSizeAndLastNodeMerklePath(1, merklePathForA)
	if err != nil {
		log.Errorf("failed to set size and lost node merkle path,err:%v", err)
	}
	err = merkleTree.UpdateLeaf(merklePathForA, &e)
	if err != nil {
		t.Error(err)
	}
	err = merkleTree.Append(&b)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	oldTree := merkleTree
	merkleTree = NewDeltaMerkleTree(oldTree.GetMerkleRoot())
	lastNodePath, err = oldTree.GetLastNodeMerklePath()
	if err != nil {
		t.Error(err)
	}
	err = merkleTree.SetSizeAndLastNodeMerklePath(oldTree.Size, lastNodePath)
	if err != nil {
		t.Error(err)
	}

	merklePathForE := &MerklePath{
		Hashes:    []*MerkleHash{&e, &b},
		ProofPath: []byte{Right},
	}
	err = merkleTree.UpdateLeaf(merklePathForE, &f)
	if err != nil {
		t.Error(err)
	}

	err = merkleTree.Append(&c)
	if err != nil {
		log.Errorf("failed to append leaves,err:%v", err)
	}
	lastNodePath, err = merkleTree.GetLastNodeMerklePath()
	if err != nil {
		t.Error(err)
	}

	fb := Hash(&f, &b, Right)
	merklePathForC := &MerklePath{
		Hashes:    []*MerkleHash{&c, fb},
		ProofPath: []byte{Left},
	}
	err = checkMerklePathEquals(lastNodePath, merklePathForC)
	if err != nil {
		t.Error(err)
	}
}

func checkMerklePathEquals(a *MerklePath, b *MerklePath) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil {
		return errors.New("one of the path is nil while the other isn't")
	}
	if len(a.Hashes) != len(b.Hashes) || len(a.ProofPath) != len(b.ProofPath) {
		return errors.New("length of hashes or proof path isn't equal")
	}
	for index, hash := range a.Hashes {
		if *hash != *b.Hashes[index] {
			return errors.New("hashes of two merkle path aren't equal")
		}
	}
	for index, pos := range a.ProofPath {
		if pos != b.ProofPath[index] {
			return errors.New("proof path of two merkle path aren't equal")
		}
	}
	return nil
}

func createSimpleMerkleTree() {
	a = MerkleHash(sha256.Sum256([]byte("a")))
	b = MerkleHash(sha256.Sum256([]byte("b")))
	c = MerkleHash(sha256.Sum256([]byte("c")))
	d = MerkleHash(sha256.Sum256([]byte("d")))
	e = MerkleHash(sha256.Sum256([]byte("e")))
	f = MerkleHash(sha256.Sum256([]byte("f")))
	g = MerkleHash(sha256.Sum256([]byte("g")))
	h = MerkleHash(sha256.Sum256([]byte("h")))

	ab = Hash(&a, &b, Right)
	cd = Hash(&c, &d, Right)
	abcd = Hash(ab, cd, Right)

	ae = Hash(&a, &e, Right)
	aecd = Hash(ae, cd, Right)

	fd = Hash(&f, &d, Right)
	aefd = Hash(ae, fd, Right)

	gd = Hash(&g, &d, Right)
	aegd = Hash(ae, gd, Right)
}
