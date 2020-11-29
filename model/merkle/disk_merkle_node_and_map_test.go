/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

import (
	"os"
	"testing"

	"github.com/multivactech/MultiVAC/base/db"
)

var dataDir = "./test"
var namespace = "my"

func TestCreateNewMerkleNode(t *testing.T) {
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)
	db, err := db.OpenDB(dataDir, namespace)
	if err != nil {
		t.Error("Faild to create db")
	}
	factory := newDiskMerkleFactory(db)
	nodes := make([]iMerkleNode, 10)
	for i := 0; i < 10; i++ {
		node := factory.newNode()
		if node == nil {
			t.Errorf("faild to create new merkle node.")
		}
		nodes[i] = node
	}
	// Test id
	for i := 1; i >= 10; i++ {
		nodeID := nodes[i-1].getID()
		if int(nodeID) != i {
			t.Errorf("Node's id is wrong, want: %d, actual: %d", i, nodeID)
		}
	}
}

func TestBytesConvert(t *testing.T) {

	fakeHash := convertBytesToMerkleHash([]byte("fake"))
	pid, lid, rid := uint64(1), uint64(2), uint64(3)
	dbvalue := getDbValue(fakeHash[:], pid, lid, rid)
	v := bytesToValueInDB(dbvalue)
	if *v.hash != *fakeHash {
		t.Error("Hash match error")
	}
	if v.pid != pid {
		t.Error("Pid match error")
	}
	if v.lid != lid {
		t.Error("Lid match error")
	}
	if v.rid != rid {
		t.Error("Rid match error")
	}
}

func TestGetHashByBytesValue(t *testing.T) {
	fakeHash := convertBytesToMerkleHash([]byte("fake"))
	pid, lid, rid := uint64(1), uint64(2), uint64(3)
	dbvalue := getDbValue(fakeHash[:], pid, lid, rid)

	v := bytesToValueInDB(dbvalue)
	if *v.hash != *fakeHash {
		t.Error("Hash match error")
	}
}

func TestGetAndSet(t *testing.T) {
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)
	db, err := db.OpenDB(dataDir, namespace)
	if err != nil {
		t.Error("Faild to create db")
	}
	factory := newDiskMerkleFactory(db)
	fakeHash := convertBytesToMerkleHash([]byte("fake"))
	fakeHashParent := convertBytesToMerkleHash([]byte("parent"))
	fakeHashLeft := convertBytesToMerkleHash([]byte("left"))
	fakeHashRight := convertBytesToMerkleHash([]byte("right"))

	nodeInMem := factory.newNode()
	parent := factory.newNode()
	left := factory.newNode()
	right := factory.newNode()
	nodeInMem.setHash(fakeHash)
	parent.setHash(fakeHashParent)
	left.setHash(fakeHashLeft)
	right.setHash(fakeHashRight)

	nodeInMem.setParent(parent)
	nodeInMem.setLeftChild(left)
	nodeInMem.setRightChild(right)

	nodeFromDB, ok := factory.getNodeMap().get(*fakeHash)
	if !ok {
		t.Error("faild to get merkle node from db")
	}
	if !nodeFromDB.isEqualTo(nodeInMem) {
		t.Error("Node match error")
	}
	if nodeFromDB.getParent() != nil && !nodeFromDB.getParent().isEqualTo(parent) {
		t.Error("Node's parent not match")
	}
	if nodeFromDB.getLeftChild() != nil && !nodeFromDB.getLeftChild().isEqualTo(left) {
		t.Error("Node's left child not match")
	}
	if nodeFromDB.getRightChild() != nil && !nodeFromDB.getRightChild().isEqualTo(right) {
		t.Error("Node's right child not match")
	}

	left.setParent(nodeInMem)
	if !left.getParent().isEqualTo(nodeFromDB) {
		t.Error("left -> parent: not match")
	}
}
