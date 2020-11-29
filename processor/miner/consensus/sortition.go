/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/configs/params"
)

// Sortitionor The sortitionor interface is implemented, to determine whether the miner can propose potential
// block and belong to a shard.
type Sortitionor struct {
	// Expected number of shards a user shall represent.
	// This is for simple sharding without deposit only.
	expNumShard uint32

	// Of leaderRates number of Leaders, only one is expected to be leader.
	leaderRates []uint32
}

// NewSortitionor will return a pointer to DummySortitionor.
func NewSortitionor(expNumShard uint32, leaderRate uint32) *Sortitionor {
	if expNumShard < 1 || expNumShard > params.GenesisNumberOfShards {
		panic(fmt.Sprintf("Expected number of shards a node can hold must be within [1, %d]",
			params.GenesisNumberOfShards))
	}

	s := &Sortitionor{
		expNumShard: expNumShard,
		leaderRates: make([]uint32, params.GenesisNumberOfShards),
	}
	for i := range s.leaderRates {
		s.leaderRates[i] = leaderRate
	}
	return s
}

// SetLeaderRate is just for testnet2.0
// n is the whole number of miners
func (ds *Sortitionor) SetLeaderRate(n int, index shard.Index) {
	n = int(math.Ceil(float64(n) * params.SecurityLevel))
	if n <= params.LeaderRateBase {
		ds.leaderRates[index] = uint32(n)
	} else {
		ds.leaderRates[index] = uint32(math.Ceil(float64(n) / float64(params.LeaderRateBase)))
	}
}

// IsLeader return whether the miner is leader.
// TODO(#155): Current algorithm is not correct
func (ds *Sortitionor) IsLeader(seed []byte, index shard.Index) bool { //Verify
	num := int(ds.leaderRates[index]) // jylu: ds.leaderRates is 0????
	log.Debugf("leader rate is %d", num)
	//num = max(num, 4)         // 4 is used for testing only
	h := fnv.New32()
	_, err := h.Write(seed)
	if err != nil {
		return false
	}
	rand := h.Sum32()
	return rand < math.MaxUint32/uint32(num)
}

// IsInShard return whether the miner belongs to the shard.
func (ds *Sortitionor) IsInShard(proof []byte, depositUnit uint32, index shard.Index) bool {
	h := fnv.New32()
	_, err := h.Write(proof)
	if err != nil {
		return false
	}
	rand := float64(h.Sum32()) / (1 << 32)
	securityLevel := params.SecurityLevel

	// rand < 1 - (1 - securityLevel)^depositUnit, as mtv yellow paper
	return rand < 1-math.Pow((1 - securityLevel), float64(depositUnit))
}
