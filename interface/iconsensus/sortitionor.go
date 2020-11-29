// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iconsensus

import "github.com/multivactech/MultiVAC/model/shard"

// Sortitionor use to elect potential leaders.
type Sortitionor interface {
	IsLeader(seed []byte, index shard.Index) bool
	IsInShard(proof []byte, depositUnit uint32, index shard.Index) bool
	SetLeaderRate(n int, index shard.Index)
}
