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
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

func TestBaseWorker_ShouldStartSync(t *testing.T) {
	tests := []struct {
		desc            string
		shouldStartSync bool
		status          syncStatus
		p               *syncPeer
		isStaleSync     bool
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
			desc:            "IDLE status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			shouldStartSync: true,
		},
		{
			desc:            "RUNNING status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			status:          RUNNING,
			shouldStartSync: false,
			isStaleSync:     false,
		},
		{
			desc:            "RUNNING status with outbound storage peer but sync is stale",
			p:               createPeerForTest(true, config.StorageNode),
			status:          RUNNING,
			shouldStartSync: true,
			isStaleSync:     true,
		},
	}

	for _, tt := range tests {
		w := newBaseWorker(shard.IDToShardIndex(TestCurrentShard))
		w.status = tt.status
		if tt.isStaleSync {
			w.syncStartTime = time.Now().Add(time.Duration(-(syncStaleMin + 1) * time.Minute))
		} else {
			w.syncStartTime = time.Now()
		}

		if tt.p != nil {
			w.addSyncPeer(tt.p.cp, tt.p.nodeType)
		}
		if s := w.shouldStartSync(); s != tt.shouldStartSync {
			t.Errorf("BaseWorker added peer when [%s], expected: %v, got: %v", tt.desc, tt.shouldStartSync, s)
		}
	}
}

func TestBaseWorker_HandleAddPeer(t *testing.T) {
	tests := []struct {
		desc            string
		status          syncStatus
		syncStartTime   time.Time
		p               *syncPeer
		sn              storageSyncContract
		abc             minerSyncContract
		shouldStartSync bool
	}{
		{
			desc:            "RUNNING status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			sn:              createFakeStorageContract(),
			abc:             nil,
			status:          RUNNING,
			syncStartTime:   time.Now().Add(-time.Minute * 1),
			shouldStartSync: false,
		},
		{
			desc:            "RUNNING status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			sn:              createFakeStorageContract(),
			abc:             nil,
			status:          RUNNING,
			syncStartTime:   time.Now().Add(-time.Minute * 10),
			shouldStartSync: true,
		},
		{
			desc:            "IDLE status with outbound storage peer",
			p:               createPeerForTest(true, config.StorageNode),
			sn:              createFakeStorageContract(),
			abc:             nil,
			status:          IDLE,
			shouldStartSync: true,
		},
		{
			desc:            "RUNNING status with inbound storage peer",
			p:               createPeerForTest(false, config.StorageNode),
			sn:              createFakeStorageContract(),
			abc:             nil,
			status:          RUNNING,
			syncStartTime:   time.Now().Add(-time.Minute * 10),
			shouldStartSync: false,
		},

		{
			desc:            "RUNNING status with outbound miner peer",
			p:               createPeerForTest(true, config.MinerNode),
			sn:              nil,
			abc:             &syncContract{},
			status:          RUNNING,
			syncStartTime:   time.Now().Add(-time.Minute * 10),
			shouldStartSync: true,
		},
		{
			desc:            "IDLE status with outbound miner peer",
			p:               createPeerForTest(true, config.MinerNode),
			sn:              nil,
			abc:             &syncContract{},
			status:          IDLE,
			shouldStartSync: true,
		},
	}
	for _, tt := range tests {
		mgr := createFakeMgr()
		if tt.p.nodeType == config.StorageNode {
			w := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
			w.setManager(mgr)
			if tt.status == RUNNING {
				if tt.syncStartTime.IsZero() {
					w.syncStartTime = time.Now()
				} else {
					w.syncStartTime = tt.syncStartTime
				}
			}
			w.status = tt.status
			if tt.p != nil {
				w.handleAddPeer(tt.p.cp, tt.p.nodeType)
				w.handleSyncReq()
			}
			if s := mgr.syncing; s != tt.shouldStartSync {
				t.Errorf("StorageWorker is syncing [%s], expected: %v, got: %v", tt.desc, tt.shouldStartSync, s)
			}
		} else {
			w := newMinerWorker(shard.IDToShardIndex(TestCurrentShard), tt.abc)
			w.setManager(mgr)
			if tt.status == RUNNING {
				if tt.syncStartTime.IsZero() {
					w.syncStartTime = time.Now()
				} else {
					w.syncStartTime = tt.syncStartTime
				}
			}
			w.status = tt.status
			if tt.p != nil {
				w.handleAddPeer(tt.p.cp, tt.p.nodeType)
				w.handleSyncReq()
			}
			if s := mgr.syncing; s != tt.shouldStartSync {
				t.Errorf("MinerWorker is syncing [%s], expected: %v, got: %v", tt.desc, tt.shouldStartSync, s)
			}
		}
	}
}

func TestBaseWorker_HandlePeerDone(t *testing.T) {
	tests := []struct {
		desc           string
		status         syncStatus
		p              *syncPeer
		sn             storageSyncContract
		abc            minerSyncContract
		shouldStopSync bool
	}{
		{
			desc:           "RUNNING status with outbound storage peer",
			p:              createPeerForTest(true, config.StorageNode),
			sn:             createFakeStorageContract(),
			abc:            nil,
			status:         RUNNING,
			shouldStopSync: true,
		},
		{
			desc:           "IDLE status with outbound storage peer",
			p:              createPeerForTest(true, config.StorageNode),
			sn:             createFakeStorageContract(),
			abc:            nil,
			status:         IDLE,
			shouldStopSync: false,
		},
		{
			desc:           "RUNNING status with inbound storage peer",
			p:              createPeerForTest(false, config.StorageNode),
			sn:             createFakeStorageContract(),
			abc:            nil,
			status:         RUNNING,
			shouldStopSync: false,
		},

		{
			desc:           "RUNNING status with outbound miner peer",
			p:              createPeerForTest(true, config.MinerNode),
			sn:             nil,
			abc:            &syncContract{},
			status:         RUNNING,
			shouldStopSync: true,
		},
		{
			desc:           "IDLE status with outbound miner peer",
			p:              createPeerForTest(true, config.MinerNode),
			sn:             nil,
			abc:            &syncContract{},
			status:         IDLE,
			shouldStopSync: false,
		},
	}
	for _, tt := range tests {
		mgr := createFakeMgr()
		if tt.p.nodeType == config.StorageNode {
			w := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
			w.setManager(mgr)
			w.status = tt.status
			w.peerMgr.addSyncPeerCandidate(tt.p.cp, tt.p.nodeType)
			w.peerMgr.getSyncPeer()
			w.handlePeerDone(tt.p.cp, tt.p.nodeType)
			if s := mgr.stop; s != tt.shouldStopSync {
				t.Errorf("StorageWorker stop [%s], expected: %v, got: %v", tt.desc, tt.shouldStopSync, s)
			}
		} else {
			w := newMinerWorker(shard.IDToShardIndex(TestCurrentShard), tt.abc)
			w.setManager(mgr)
			w.status = tt.status
			w.peerMgr.addSyncPeerCandidate(tt.p.cp, tt.p.nodeType)
			w.peerMgr.getSyncPeer()
			w.handlePeerDone(tt.p.cp, tt.p.nodeType)
			if s := mgr.stop; s != tt.shouldStopSync {
				t.Errorf("MinerWorker stop [%s], expected: %v, got: %v", tt.desc, tt.shouldStopSync, s)
			}
		}
	}
}

func TestBaseWorker_IsSyncStale(t *testing.T) {
	bw := newBaseWorker(shard.IDToShardIndex(TestCurrentShard))
	tests := []struct {
		time time.Time
		res  bool
	}{
		{
			time: time.Now().Add(-time.Minute * 1),
			res:  false,
		},
		{
			time: time.Now().Add(-time.Minute * 5),
			res:  true,
		},
		{
			time: time.Now().Add(-time.Minute * 4),
			res:  true,
		},
		{
			time: time.Now().Add(-time.Minute * 10),
			res:  true,
		},
	}
	for _, tt := range tests {
		bw.syncStartTime = tt.time
		if bw.isSyncStale() != tt.res {
			t.Errorf("Last sync time is:%v, now time is:%v ,the IsSyncStale result should be:%v,but get:%v", tt.time, time.Now(), tt.res, bw.isSyncStale())
		}
	}
}

func TestBaseWork_ProcessMail(t *testing.T) {
	mgr := createFakeMgr()
	sp := createPeerForTest(true, config.StorageNode)
	sw := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
	sw.setManager(mgr)

	msgAddPeer := &peerMail{connPeer: sp.cp, nodeType: sp.nodeType}
	sw.status = IDLE
	sw.Act(message.NewEvent(evtAddPeer, msgAddPeer), nil)
	sw.handleSyncReq()
	if s := mgr.syncing; s != true {
		t.Errorf("Process add peer mail,the peer's sync status, expected: %v, got: %v", true, s)
	}

	msgPeerDone := &peerMail{connPeer: sp.cp, nodeType: sp.nodeType}
	sw.status = RUNNING
	sw.Act(message.NewEvent(evtDonePeer, msgPeerDone), nil)
	if s := mgr.stop; s != true {
		t.Errorf("Process peer done mail,the peer's stop status, expected: %v, got: %v", true, s)
	}

	//process message type e.g. msgSyncInv, msgSyncReq was tested in handleMag test
	msgSyncInv := wire.NewMsgSyncInv(shard.IDToShardIndex(TestCurrentShard), wire.SyncInvTypeGetData)
	sw.Act(message.NewEvent(evtMsg, msgSyncInv), nil)
	msgSyncReq := wire.NewMsgSyncReq(shard.IDToShardIndex(TestCurrentShard))
	sw.Act(message.NewEvent(evtMsg, msgSyncReq), nil)

	//process default mail e.g. storageMissInfo was tested in handleCustomReq test
	msgDefault := &storageMissInfo{}
	sw.Act(message.NewEvent(-1, msgDefault), nil)
}

func TestBaseWork_HandleSyncInv(t *testing.T) {
	msgSyncInvFullList := wire.NewMsgSyncInv(shard.IDToShardIndex(TestCurrentShard), wire.SyncInvTypeFullList)
	msgSyncInvFullList.AddInvGroup(shard.IDToShardIndex(TestCurrentShard), *testHash0)
	sw := newStorageWorker(shard.IDToShardIndex(TestCurrentShard), createFakeStorageContract())
	p := createPeerForTest(true, config.StorageNode)
	sw.peerMgr.addSyncPeerCandidate(p.cp, p.nodeType)
	sw.status = IDLE
	lenCurInvBeforeSync0 := len(sw.curInvs)
	sw.handleSyncInv(msgSyncInvFullList)
	lenCurInvAfterSync0 := len(sw.curInvs)
	if lenCurInvAfterSync0 != lenCurInvBeforeSync0 {
		t.Errorf("If the status of worker is IDLE, the SyncInv, the curInv will not work")
	}
	sw.status = RUNNING
	lenCurInvBeforeSync := len(sw.curInvs)
	sw.handleSyncInv(msgSyncInvFullList)
	lenCurInvAfterSync := len(sw.curInvs)
	if lenCurInvAfterSync != lenCurInvBeforeSync+len(msgSyncInvFullList.InvGroups) {
		t.Errorf("After SyncInv, the curInv length of worker should add to: %v, got: %v ", lenCurInvAfterSync, lenCurInvBeforeSync)
	}
	msgSyncInvGetData := wire.NewMsgSyncInv(shard.IDToShardIndex(TestCurrentShard), wire.SyncInvTypeGetData)
	msgSyncInvGetData.AddInvGroup(shard.IDToShardIndex(TestCurrentShard), *testHash0)
	sw.handleSyncInv(msgSyncInvGetData)
}

var (
	testHashStr0  = "14a0810ac680a3eb3f82edc878cea25ec41d6b790744e5daeef"
	testHash0, _  = chainhash.NewHashFromStr(testHashStr0)
	testkHashStr1 = "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	testHash1, _  = chainhash.NewHashFromStr(testkHashStr1)
)

func TestHashSet_Add(t *testing.T) {
	hs := &hashSet{
		hashes: make(map[chainhash.Hash]bool),
	}
	lenBeforAdd := len(hs.hashes)
	hs.add(*testHash0)
	lenAfterAdd := len(hs.hashes)
	if lenAfterAdd != lenBeforAdd+1 {
		t.Errorf("After add a hash,wo should get: %v, but got: %v", lenBeforAdd+1, lenAfterAdd)
	}
}

func TestHashSet_Remove(t *testing.T) {
	hs := &hashSet{
		hashes: make(map[chainhash.Hash]bool),
	}
	hs.add(*testHash0)
	lenBeforRem0 := len(hs.hashes)
	hs.remove(*testHash1)
	lenAfterRem0 := len(hs.hashes)
	//because there is not testHash1 in hashSet, the length will not change after remove
	if lenAfterRem0 != lenBeforRem0 {
		t.Errorf("After remove a hash which is not in, wo should get: %v, but got: %v", lenBeforRem0, lenAfterRem0)
	}
	lenBeforRem1 := hs.size()
	hs.remove(*testHash0)
	lenAfterRem1 := hs.size()
	if lenAfterRem1 != lenBeforRem1-1 {
		t.Errorf("After remove a hash, wo should get: %v, but got: %v", lenBeforRem1-1, lenAfterRem1)
	}
}

func TestHashSet_Contains(t *testing.T) {
	hs := &hashSet{
		hashes: make(map[chainhash.Hash]bool),
	}
	hs.add(*testHash0)
	if hs.contains(*testHash1) != false {
		t.Errorf("There is not a testHash1, wo should get: %v, but got: %v", false, hs.contains(*testHash1))
	}
	if hs.contains(*testHash0) != true {
		t.Errorf("There is a testHash0, wo should get: %v, but got: %v", true, hs.contains(*testHash0))
	}
}
