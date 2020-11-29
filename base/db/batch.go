// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type batchImpl struct {
	batch *leveldb.Batch
}

func (bt *batchImpl) Put(key []byte, value []byte) {
	bt.batch.Put(key, value)
}

func (bt *batchImpl) Len() int {
	return bt.batch.Len()
}
