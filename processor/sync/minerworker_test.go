/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestMinerWorker_HandleAddPeer(t *testing.T) {
	tests := []struct {
		desc            string
		shouldStartSync bool
		status          syncStatus
		p               *syncPeer
	}{
		{
			desc:            "IDLE status without peer",
			status:          IDLE,
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status without peer",
			shouldStartSync: false,
			status:          RUNNING,
		},
		{
			desc:            "IDLE status with inbound miner peer",
			p:               createPeerForTest(false, config.MinerNode),
			shouldStartSync: false,
		},
		{
			desc:            "IDLE status with outbound miner peer",
			p:               createPeerForTest(true, config.MinerNode),
			shouldStartSync: true,
		},
		{
			desc:            "IDLE status with inbound storage peer",
			p:               createPeerForTest(false, config.StorageNode),
			shouldStartSync: false,
		},
		{
			desc:            "IDLE status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status with inbound miner peer",
			p:               createPeerForTest(false, config.MinerNode),
			status:          RUNNING,
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status with outbound miner peer",
			p:               createPeerForTest(true, config.MinerNode),
			status:          RUNNING,
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status with inbound storage peer",
			p:               createPeerForTest(false, config.StorageNode),
			status:          RUNNING,
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			status:          RUNNING,
			shouldStartSync: false,
		},
	}

	for _, tt := range tests {
		mgr := createFakeMgr()
		mw := newMinerWorker(shard.IDToShardIndex(TestCurrentShard), &syncContract{})
		mw.setManager(mgr)

		if tt.status == RUNNING {
			mw.syncStartTime = time.Now()
		}
		mw.status = tt.status
		if tt.p != nil {
			mw.handleAddPeer(tt.p.cp, tt.p.nodeType)
			mw.handleSyncReq()
		}
		if s := mgr.syncing; s != tt.shouldStartSync {
			t.Errorf("MinerWorker is syncing [%s], expected: %v, got: %v", tt.desc, tt.shouldStartSync, s)
		}
	}
}
func TestMinerWorker_CreateSyncReq(t *testing.T) {
	mw := newMinerWorker(shard.IDToShardIndex(TestCurrentShard), &syncContract{})
	msg := mw.createSyncReq()
	if msg.ReqShardIdx.GetID() != TestCurrentShard {
		t.Errorf("MsgSyncReq shard expected: %v, got: %v", TestCurrentShard, msg.ReqShardIdx.GetID())
	}
	for _, locator := range msg.Locators {
		if locator.ToHeight != wire.ReqSyncToLatest {
			t.Errorf("MsgSyncReq locator [shard: %v] toHeight expected: %v, got: %v",
				locator.ShardIdx.GetID(), wire.ReqSyncToLatest, locator.ToHeight)
		}
		if e := TestShardHeight + 1; e != locator.FromHeight {
			t.Errorf("MsgSyncReq locators [shard: %v] fromHeight expected: %v, got: %v",
				locator.ShardIdx.GetID(), e, locator.FromHeight)
		}
	}
}

var TestShardHeight int64 = 6

type syncContract struct {
}

func (c *syncContract) GetShardHeight() int64 {
	return TestShardHeight
}

func (c *syncContract) ReceiveSyncHeader(header *wire.BlockHeader) {}
