/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"testing"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

func TestProvideManager(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	if mgr.state.status != IDLE {
		t.Errorf("We should get the init status of provideManager: %v, got: %v,", IDLE, mgr.state.status)
	}
}

func TestSyncManager_Start(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	fw := newFakeWorker()
	mgr.worker = fw
	mgr.Start()
	if !mgr.actorCtx.ActorStarted(fw) {
		t.Errorf("After SyncManager start, wo should get the status of fakeManager: %v, got: %v", RUNNING, fw.status)
	}
}

func TestSyncManager_AcceptMsg(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	fw := newFakeWorker()
	mgr.worker = fw
	mgr.actorCtx.StartActor(fw)

	msg := wire.NewMsgSyncInv(shard.IDToShardIndex(TestCurrentShard), wire.SyncInvTypeGetData)
	mgr.AcceptMsg(msg)
	switch m := <-fw.mail; mail := m.(type) {
	case wire.Message:
		if mail != msg {
			t.Errorf("We send msg: %v, but got: %v", msg, mail)
		}
	default:
		t.Errorf("Incorrect type of mail: want wire.Message, get %T", fw.mail)
	}
}

func TestSyncManager_OnNewPeer(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	fw := newFakeWorker()
	mgr.worker = fw
	mgr.actorCtx.StartActor(fw)

	sp := createPeerForTest(true, config.StorageNode)
	mgr.OnNewPeer(sp.cp, sp.nodeType)
	switch m := <-fw.mail; mail := m.(type) {
	case *peerMail:
		if sp.cp != mail.connPeer || sp.nodeType != mail.nodeType {
			t.Errorf("We send OnNewPeer: %v %v, but got: %v %v", sp.cp, sp.nodeType, mail.connPeer, mail.nodeType)
		}
	default:
		t.Errorf("Incorrect type of mail: want *peerMail, get %T", fw.mail)
	}
}

func TestSyncManager_OnPeerDone(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	fw := newFakeWorker()
	mgr.worker = fw
	mgr.actorCtx.StartActor(fw)

	sp := createPeerForTest(true, config.StorageNode)
	mgr.OnPeerDone(sp.cp, sp.nodeType)
	switch m := <-fw.mail; mail := m.(type) {
	case *peerMail:
		if mail.connPeer != sp.cp || mail.nodeType != sp.nodeType {
			t.Errorf("We send OnPeerDone: %v %v, but got: %v %v", sp.cp, sp.nodeType, mail.connPeer, mail.nodeType)
		}
	default:
		t.Errorf("Incorrect type of mail: want *peerMail, get %T", fw.mail)
	}
}

func TestSyncManager_Sub(t *testing.T) {
	mgr := provideManager(shard.IDToShardIndex(TestCurrentShard), message.NewPubSubManager())
	fw := newFakeWorker()
	mgr.worker = fw
	mgr.actorCtx.StartActor(fw)

	ss := newSyncState(message.NewPubSubManager())
	mgr.state = ss
	fs := createFakeSubscriber()
	mgr.state.pubSubMgr.Sub(fs)
	mgr.notifySyncStart()
	if isStart := <-fs.isStart; isStart != true {
		t.Errorf("After SyncMgr notifySyncStart, fakeSubscriber.isStart state should be:%v,but got:%v", true, fs.isStart)
	}
	mgr.notifySyncComplete(true)
	if isComplete := <-fs.isComplete; isComplete != true {
		t.Errorf("After SyncMgr notifySyncComplete, fakeSubscriber.isComplete state should be:%v,but got:%v", true, fs.isComplete)
	}
}

type fakeWorker struct {
	mailbox chan interface{}
	// unused
	// peerMger *syncPeerManager
	status syncStatus
	mail   chan interface{}
}

func newFakeWorker() *fakeWorker {
	return &fakeWorker{
		mailbox: make(chan interface{}, maxMailBuffer),
		status:  IDLE,
		mail:    make(chan interface{}),
	}
}

func (fw *fakeWorker) setManager(mgr reportAPI) {
}

func (fw *fakeWorker) Act(e *message.Event, _ func(m interface{})) {
	fw.mail <- e.Extra
}
