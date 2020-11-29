// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

const (
	// HashSize means the length of hash string.
	HashSize = 32
)

type dbImpl struct {
	db            *leveldb.DB // LevelDB instance
	fn            string      // filename
	blockWriteSet map[string][]byte
	imem          map[string][]byte
	batch         Batch
}

// Get retrieves the given key if it's present in the key-value store.
func (db *dbImpl) Get(key []byte) ([]byte, error) {

	//Search from memory
	if db.blockWriteSet != nil {
		v, ok := db.blockWriteSet[string(key)]
		if ok {
			return v, nil
		}
	}
	// Search from imem
	if db.imem != nil {
		iv, iok := db.imem[string(key)]
		if iok {
			return iv, nil
		}
	}
	return db.db.Get(key, nil)
}

// Put inserts the given value into the key-value store.
func (db *dbImpl) Put(key []byte, value []byte) error {
	// In the blockWriteSet there are two kinds of k-v:
	// 1: id -> {hash,pid,lid,rid}
	// 2: hash -> id
	// If the key is hash, it means this record is type of 2,
	// so get the id from the and get old hash from map by this id
	// then delete the key of old hash and insert the new record.
	if db.blockWriteSet != nil {
		// TODO:(zhaozheng)
		// Warning: can't use this condition to verify the key
		if len(key) == HashSize {
			key := string(value)
			value, ok := db.blockWriteSet[key]
			if ok {
				delete(db.blockWriteSet, string(value[0:HashSize]))
			}
		}
		db.blockWriteSet[string(key)] = value
		return nil
	}

	return db.db.Put(key, value, nil)
}

// Has retrieves if a key is present in the key-value store.
func (db *dbImpl) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Delete removes the key from the key-value store.
func (db *dbImpl) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

// AtomicWrite applies the given batch to the transaction.
func (db *dbImpl) atomicWrite(b Batch) error {
	tx, err := db.db.OpenTransaction()
	defer tx.Commit()
	if err != nil {
		return err
	}
	if bt, ok := b.(*batch); ok {
		return tx.Write(bt.b, nil)
	}
	return errors.New("can't write batch: unknown type")
}

// StartTransactionRecord creates a temporary map to store the data that will be written to the DB next.
func (db *dbImpl) StartTransactionRecord() {
	db.blockWriteSet = make(map[string][]byte)
}

// CommitTransaction packages the data in the temporary map into a transaction to write to the database then clear the map.
func (db *dbImpl) CommitTransaction() {
	db.imem = db.blockWriteSet
	db.blockWriteSet = nil

	for k, v := range db.imem {
		db.batch.Put([]byte(k), v)
	}
	err := db.atomicWrite(db.batch)
	if err != nil {
		panic(err)
	}

	db.imem = nil
	db.batch.Reset()
}

// GetIter return the iterator of the db
func (db *dbImpl) GetIter() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}

// Close will close the database connection.
func (db *dbImpl) Close() error {
	return db.db.Close()
}

type batch struct {
	b    *leveldb.Batch
	size int
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size++
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.b.Reset()
	b.size = 0
}
