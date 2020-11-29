package consensus

import (
	"encoding/hex"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/stretchr/testify/assert"
)

var (
	testNumShard   uint32 = 4
	testLeaderRate uint32 = 4
	// IsLeader is always equal to true when leaderRates is equal to one.
	testLeaderRateIsOne  = 1
	testShard            = shard.Index(0)
	testLeaderSeed, _    = hex.DecodeString("205f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
	testNotLeaderSeed, _ = hex.DecodeString("885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
)

func TestSortitionor_IsLeader(t *testing.T) {
	// init new Sortitionor
	ds := NewSortitionor(testNumShard, testLeaderRate)

	assert.Equal(t, ds.IsLeader(testLeaderSeed, testShard), true)
	assert.Equal(t, ds.IsLeader(testNotLeaderSeed, testShard), false)

	ds.SetLeaderRate(testLeaderRateIsOne, testShard)
	assert.Equal(t, ds.IsLeader(testNotLeaderSeed, testShard), true)
}
