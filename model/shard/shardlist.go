// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package shard

import (
	"github.com/multivactech/MultiVAC/configs/params"
)

func generateShardList() []Index {
	shardList := []Index{}
	for i := 0; i < (1 << params.GenesisShardPrefixLength); i++ {
		shardList = append(shardList, IDToShardIndex(uint32(i)))
	}
	return shardList
}

// ShardList is the list of shard index.
var ShardList = generateShardList()
