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
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/p2p/peer"
)

const (
	TestCurrentShard = 1
)

var (
	TestShardHeights = [4]int64{4, 3, 8, 7}
)

func TestNewStorageWorker(t *testing.T) {
	w := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
	if w.status != IDLE {
		t.Errorf("StorageWorker initial status, expected: %v, got: %v", IDLE, w.status)
	}
	if w.peerMgr.getSyncPeer() != nil {
		t.Errorf("StorageWorker expected 0 sync peers by default, got: %d", len(w.peerMgr.candidates))
	}
}

func TestStorageWorker_HandleAddPeer(t *testing.T) {
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
			shouldStartSync: false,
		},
		{
			desc:            "IDLE status with inbound storage peer",
			p:               createPeerForTest(false, config.StorageNode),
			shouldStartSync: false,
		},
		{
			desc:            "IDLE status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			shouldStartSync: true,
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
		w := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
		w.setManager(mgr)
		if tt.status == RUNNING {
			w.syncStartTime = time.Now()
		}
		w.status = tt.status
		if tt.p != nil {
			w.handleAddPeer(tt.p.cp, tt.p.nodeType)
			w.handleSyncReq()
		}
		if s := mgr.syncing; s != tt.shouldStartSync {
			t.Errorf("StorageWorker is syncing [%s], expected: %v, got: %v", tt.desc, tt.shouldStartSync, s)
		}
	}
}

func TestCreateSyncReq(t *testing.T) {
	w := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
	msg := w.createSyncReq()
	if msg.ReqShardIdx.GetID() != TestCurrentShard {
		t.Errorf("MsgSyncReq shard expected: %v, got: %v", TestCurrentShard, msg.ReqShardIdx.GetID())
	}
	if l := len(msg.Locators); l != len(TestShardHeights) {
		t.Errorf("MsgSyncReq locators length expected: %v, got: %v", len(TestShardHeights), l)
	}
	for _, locator := range msg.Locators {
		if locator.ToHeight != wire.ReqSyncToLatest {
			t.Errorf("MsgSyncReq locator [shard: %v] toHeight expected: %v, got: %v",
				locator.ShardIdx.GetID(), wire.ReqSyncToLatest, locator.ToHeight)
		}
		if e := TestShardHeights[locator.ShardIdx.GetID()] + 1; e != locator.FromHeight {
			t.Errorf("MsgSyncReq locators [shard: %v] fromHeight expected: %v, got: %v",
				locator.ShardIdx.GetID(), e, locator.FromHeight)
		}
	}
}

func createPeerForTest(outbound bool, t config.NodeType) *syncPeer {
	_, _ = config.LoadConfig()
	config.GlobalConfig().RunAsDNSSeed = false
	sp := syncPeer{nodeType: t}
	var isStorageNode bool
	if t == config.StorageNode {
		isStorageNode = true
	} else {
		isStorageNode = false
	}
	allShards := [4]shard.Index{shard.IDToShardIndex(0), shard.IDToShardIndex(1),
		shard.IDToShardIndex(2), shard.IDToShardIndex(3)}
	if outbound {
		p, _ := peer.NewOutboundPeer(&peer.Config{}, "10.0.0.2:8333")
		var shards = allShards[0:4]
		updatePeerInfo := &wire.UpdatePeerInfoMessage{
			Shards:               shards,
			IsStorageNode:        isStorageNode,
			RPCListenerAddress:   "10.0.0.2:8334",
			LocalListenerAddress: "10.0.0.2:8333",
		}
		cp := &connection.ConnPeer{}
		cp.Peer = p
		cp.OnUpdateShardInfo(updatePeerInfo)
		sp.cp = cp
	} else {
		p := peer.NewInboundPeer(&peer.Config{})
		var shards = allShards[0:4]
		shards[0] = shard.IDToShardIndex(0)
		shards[1] = shard.IDToShardIndex(1)
		shards[2] = shard.IDToShardIndex(2)
		shards[3] = shard.IDToShardIndex(3)
		updatePeerInfo := &wire.UpdatePeerInfoMessage{
			Shards:               shards,
			IsStorageNode:        isStorageNode,
			RPCListenerAddress:   "10.0.0.2:8334",
			LocalListenerAddress: "10.0.0.2:8333",
		}
		cp := &connection.ConnPeer{}
		cp.Peer = p
		cp.OnUpdateShardInfo(updatePeerInfo)
		sp.cp = cp
	}
	return &sp
}

type fakeManager struct {
	syncing bool
	stop    bool
}

func createFakeMgr() *fakeManager {
	return &fakeManager{syncing: false, stop: false}
}

func (m *fakeManager) notifySyncStart() {
	m.syncing = true
	m.stop = false
}

func (m *fakeManager) notifySyncComplete(dataFetched bool) {
	m.syncing = false
	m.stop = true
}

func (m *fakeManager) notifySyncAborted() {
	m.syncing = false
	m.stop = true
}

type fakeStorageNode struct {
}

func createFakeStorageContract() *fakeStorageNode {
	return &fakeStorageNode{}
}

func (sn *fakeStorageNode) GetAllShardHeights() []shard.IndexAndHeight {
	return []shard.IndexAndHeight{
		{
			Index:  shard.IDToShardIndex(0),
			Height: TestShardHeights[0],
		},
		{
			Index:  shard.IDToShardIndex(3),
			Height: TestShardHeights[3],
		},
		{
			Index:  shard.IDToShardIndex(2),
			Height: TestShardHeights[2],
		},
		{
			Index:  shard.IDToShardIndex(1),
			Height: TestShardHeights[1],
		},
	}
}

func (sn *fakeStorageNode) HandleMsg(msg wire.Message, p peer.Reply) {}
