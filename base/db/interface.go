// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"os"
	"path/filepath"

	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	cacheSize   = 64
	fileHandles = 64
)

// Batch interface is defined to deal for a series of data.
type Batch interface {
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	Reset()
}

// DB interface is used for database.
type DB interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	StartTransactionRecord()
	CommitTransaction()
	Close() error
	GetIter() iterator.Iterator
}

// Cache interface is used for cache data from database.
type Cache interface {
	Put(key []byte, value []byte)
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
}

// OpenDB returns a wrapped leveldb instance(currently).
func OpenDB(dataDir string, namespace string) (DB, error) {
	dir := filepath.Join(dataDir, "leveldb", namespace)
	os.MkdirAll(dir, os.ModePerm)

	// Open db and recover any potential corruptions
	options := opt.Options{
		WriteBuffer:            params.LeveldbCacheMb * opt.MiB,
		BlockCacheCapacity:     params.LeveldbBlockCacheMb * opt.MiB,
		OpenFilesCacheCapacity: params.LeveldbFileNumber,
		Filter:                 filter.NewBloomFilter(19),
	}
	ldb, err := leveldb.OpenFile(dir, &options)
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		ldb, err = leveldb.RecoverFile(dir, nil)
	}
	if err != nil {
		return nil, err
	}
	return &dbImpl{
		db:    ldb,
		fn:    dir,
		batch: NewBatch(),
	}, nil
}

// NewBatch creates a batch instance.
func NewBatch() Batch {
	return &batch{
		b: &leveldb.Batch{},
	}
}
