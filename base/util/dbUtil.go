// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package util

import (
	"fmt"
	"github.com/multivactech/MultiVAC/base/rlp"
)

const (
	// KeyLedgertreeRoot is root of Ledgertree.
	KeyLedgertreeRoot = "LEDGERTREE_ROOT"
	// KeyLedgertreeLastnodehash is the Last node hash of Ledgertree.
	KeyLedgertreeLastnodehash = "LEDGERTREE_LASTNODEHASH"
	// KeyLedgertreeSize is the size of Ledgertree.
	KeyLedgertreeSize = "LEDGERTREE_SIZE"
	// KeyShardHeight is shard height.
	KeyShardHeight = "SHARD_HEIGHT_%d"
	// KeyStorageShardheight is shard height of storage node.
	KeyStorageShardheight = "STORAGE_SHARDHEIGHT"
	// KeyNodeCursor is node cursor.
	KeyNodeCursor = "NODE_CURSOR"
)

// GetShardHeightKey returns the shard key by given shard.
func GetShardHeightKey(shard uint32) []byte {
	res := fmt.Sprintf(KeyShardHeight, shard)
	return []byte(res)
}

// ValToByte transform val to byte array.
func ValToByte(val interface{}) ([]byte, error) {
	res, err := rlp.EncodeToBytes(val)
	if err != nil {
		err := fmt.Errorf("EncodeToBytes err:%v", err)
		return nil, err
	}
	return res, nil
}
