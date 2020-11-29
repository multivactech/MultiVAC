// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iabci

import (
	"github.com/multivactech/MultiVAC/model/shard"
)

// Fetcher is an interface to dealing with request fetching information from storage node.
type Fetcher interface {
	// Request a given number of transactions from storage node.
	FetchTransactions(number int, hashList []uint32, shard shard.Index)
	// Request data for abci initalization from storage node
	FetchInitData(shard shard.Index)
}
