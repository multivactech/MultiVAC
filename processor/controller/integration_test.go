package controller

import (
	"encoding/hex"
	"sort"
	"testing"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var shardListForTest = []uint32{0, 1, 2, 3}
var shardIndexListForTest = []shard.Index{
	shard.IDToShardIndex(0),
	shard.IDToShardIndex(1),
	shard.IDToShardIndex(2),
	shard.IDToShardIndex(3),
}

func TestController_initEnabledShards_miner(t *testing.T) {
	_, _ = config.LoadConfig()
	vrf := &vrf.Ed25519VRF{}
	pk, sk, _ := vrf.GenerateKey(nil)

	testcases := []struct {
		desc              string
		cfg               *config.Config
		controller        *RootController
		resultReshardings []shard.Index //ENABLE_RESHARDING = true
	}{
		{
			desc: "MinerNode IsOneOfFirstMiners:true IsSharded:false",
			controller: NewRootController(&config.Config{
				StorageNode:        false,
				IsOneOfFirstMiners: true,
				IsSharded:          true,
				Shards:             shardListForTest,
				ExpNumShard:        len(shardListForTest),
				LeaderRate:         3,
				Sk:                 hex.EncodeToString(sk),
				Pk:                 hex.EncodeToString(pk),
			}, new(mockedConnector)),
			resultReshardings: []shard.Index{
				shard.IDToShardIndex(3),
				shard.IDToShardIndex(0),
				shard.IDToShardIndex(1),
				shard.IDToShardIndex(2),
			},
		},
	}

	a := assert.New(t)
	for _, test := range testcases {
		a.Truef(isEqual(test.controller.GetEnabledShards(), test.resultReshardings),
			"Enabled resharding, when initEnabledShards, %s want: %v, get: %v",
			test.desc, test.resultReshardings, test.controller.GetEnabledShards())
	}
}

type mockedConnector struct {
	mock.Mock
}

func (c *mockedConnector) UpdatePeerInfo([]shard.Index) {

}

func isEqual(a, b []shard.Index) bool {
	sort.SliceStable(a, func(i, j int) bool {
		return a[i] < a[j]
	})
	sort.SliceStable(b, func(i, j int) bool {
		return b[i] < b[j]
	})
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
