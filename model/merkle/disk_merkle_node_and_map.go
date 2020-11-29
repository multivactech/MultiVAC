// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package merkle

import (
	"encoding/binary"

	"github.com/multivactech/MultiVAC/base/db"
	"github.com/multivactech/MultiVAC/base/util"
)

const (
	// NilKey means there is no merkle node the same to this.
	NilKey = 0
	// HashOffset is the offset of hash，used to analyze the hash for given byte slice.
	HashOffset = 32
	// PidOffset is the offset of parent id ，used to analyze the pid for given byte slice.
	PidOffset = HashOffset + 8
	// LidOffset is the offset of left id ,used to analyze the lid for given byte slice.
	LidOffset = PidOffset + 8
	// RidOffset is the offset of right id ,used to analyze the rid for given byte slice.
	RidOffset = LidOffset + 8
	// ValueInDbLength is the length for one merkle node data in level db.
	ValueInDbLength = RidOffset
)

type valueInDB struct {
	hash *MerkleHash
	pid  uint64
	lid  uint64
	rid  uint64
}

// TODO(zhongdong): close db when MultiVAC shuts down.
type diskMerkleFactory struct {
	db     db.DB
	nm     *diskNodeMap
	cursor uint64
}

// TODO(zhongdong): cache the node instances in memory after they are loaded.
type diskNodeMap struct {
	db db.DB
}

type diskMerkleNode struct {
	nm *diskNodeMap
	v  *valueInDB
	id uint64
}

func newDiskMerkleFactory(db db.DB) *diskMerkleFactory {
	nm := &diskNodeMap{}
	factory := &diskMerkleFactory{}
	factory.db = db
	factory.nm = nm
	factory.nm.db = db
	// For restart: read cursor from db
	res, err := factory.db.Get([]byte(util.KeyNodeCursor))
	if err != nil {
		log.Errorf("Merkle: Get factory cursor err, %v", err)
	}
	cursor := util.BytesToInt64(res)
	log.Infof("get cursor from db: %d", cursor)
	factory.cursor = uint64(cursor)
	return factory
}

func (factory *diskMerkleFactory) getNodeMap() iNodeMap {
	return factory.nm
}

func (factory *diskMerkleFactory) newNode() iMerkleNode {
	node := &diskMerkleNode{}
	node.nm = factory.nm
	node.v = &valueInDB{}
	factory.cursor++
	factory.putCursorToDisk()
	node.id = factory.cursor
	return node
}

func (factory *diskMerkleFactory) putCursorToDisk() {
	err := factory.db.Put([]byte(util.KeyNodeCursor), util.Int64ToBytes(int64(factory.cursor)))
	if err != nil {
		log.Errorf("Merkle: Can't write cursor to db, %v", err)
	}
	//return
}

func (nm *diskNodeMap) set(key MerkleHash, value iMerkleNode) {
	dbkey := getDbKey(value.getID())
	err := nm.db.Put(key[:], dbkey)
	if err != nil {
		log.Errorf("failed to put to database,err:%v", err)
	}
}

func (nm *diskNodeMap) getByID(id uint64) iMerkleNode {
	if id == NilKey {
		return nil
	}
	dbValue, err := nm.db.Get(getDbKey(id))
	if dbValue == nil || err != nil {
		return nil
	}

	v := bytesToValueInDB(dbValue)
	if v == nil {
		return nil
	}
	node := &diskMerkleNode{}
	node.nm = nm
	node.v = v
	node.id = id
	return node
}

func (nm *diskNodeMap) get(key MerkleHash) (iMerkleNode, bool) {
	dbkey, err := nm.db.Get(key[:])
	if err != nil {
		return nil, false
	}
	node := nm.getByID(bytesToUint64(dbkey))
	return node, node != nil
}

func (nm *diskNodeMap) del(key MerkleHash) {
	err := nm.db.Delete(key[:])
	if err != nil {
		log.Errorf("failed to delete,err:%v,key:%v", err, key)
	}
}

func (mn *diskMerkleNode) getID() uint64 {
	return mn.id
}

func (mn *diskMerkleNode) setParent(node iMerkleNode) {
	pid := mn.getNodeID(node)
	if mn.v.pid != pid {
		mn.v.pid = pid
		mn.save()
	}
}

func (mn *diskMerkleNode) getParent() iMerkleNode {
	return mn.nm.getByID(mn.v.pid)
}

func (mn *diskMerkleNode) setLeftChild(node iMerkleNode) {
	lid := mn.getNodeID(node)
	if mn.v.lid != lid {
		mn.v.lid = lid
		mn.save()
	}
}

func (mn *diskMerkleNode) getLeftChild() iMerkleNode {
	return mn.nm.getByID(mn.v.lid)
}

func (mn *diskMerkleNode) setRightChild(node iMerkleNode) {
	rid := mn.getNodeID(node)
	if mn.v.rid != rid {
		mn.v.rid = rid
		mn.save()
	}
}

func (mn *diskMerkleNode) getRightChild() iMerkleNode {
	return mn.nm.getByID(mn.v.rid)
}

func (mn *diskMerkleNode) getNodeID(node iMerkleNode) uint64 {
	if n, ok := node.(*diskMerkleNode); ok {
		return n.id
	}
	return NilKey

}

func (mn *diskMerkleNode) setHash(hash *MerkleHash) {
	mn.v.hash = hash
	mn.nm.set(*hash, mn)
	mn.save()
}

func (mn *diskMerkleNode) getHash() *MerkleHash {
	return mn.v.hash
}

func (mn *diskMerkleNode) isEqualTo(node iMerkleNode) bool {
	if n, ok := node.(*diskMerkleNode); ok {
		return *n.getHash() == *mn.getHash()
	}
	return false

}

// save saves the node info to db.
func (mn *diskMerkleNode) save() {
	if mn.v.hash == nil {
		return
	}
	dbkey := getDbKey(mn.getID())
	dbValue := getDbValue(mn.v.hash[:], mn.v.pid, mn.v.lid, mn.v.rid)
	err := mn.nm.db.Put(dbkey, dbValue)
	if err != nil {
		log.Errorf("failed to put to database,err:%v,key:%v", err, dbkey)
	}
}

func getDbKey(id uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)
	return b[:]
}

func getDbValue(hash []byte, pid uint64, lid uint64, rid uint64) []byte {
	pidToDSave := getDbKey(pid)
	lidToSave := getDbKey(lid)
	ridToSave := getDbKey(rid)
	value := append(append(append(hash, pidToDSave...), lidToSave...), ridToSave...)
	return value
}

func bytesToValueInDB(bytes []byte) *valueInDB {
	if len(bytes) != ValueInDbLength {
		return nil
	}

	hash := convertBytesToMerkleHash(bytes[0:HashOffset])
	pid := bytesToUint64(bytes[HashOffset:PidOffset])
	lid := bytesToUint64(bytes[PidOffset:LidOffset])
	rid := bytesToUint64(bytes[LidOffset:RidOffset])

	v := &valueInDB{
		hash: hash,
		pid:  pid,
		lid:  lid,
		rid:  rid,
	}
	return v
}

// it seems uused.
//func _panicIfError(err error) {
//	if err != nil {
//		panic(err)
//	}
//}
