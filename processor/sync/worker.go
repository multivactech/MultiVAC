/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"fmt"
	"time"

	"github.com/multivactech/MultiVAC/logger"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	maxMailBuffer = 10000
	// After this time if the sync hasn't completed, it will be considered as stale.
	syncStaleMin = 2
)

// syncWorker is the interface of a SyncWorker, which does the actual work of syncing with instructions from SyncManager.
type syncWorker interface {
	// syncWorker implements message.Actor
	message.Actor

	// setManager takes registers the manager of this worker for callback.
	setManager(manager reportAPI)
}

// diffWork represented the concrete work that differentiate between storage and miner.
type diffWork interface {
	handleCustomMsg(msg wire.Message)
	makeSyncReq()
	isSyncCandidate(cp *connection.ConnPeer, t config.NodeType) bool
}

type baseWorker struct {
	// mailbox is the channel to receive instructions from SyncManager for sequence execution.
	mailbox    chan interface{}
	peerMgr    *syncPeerManager
	manager    reportAPI
	shardIdx   shard.Index
	diffWorker diffWork

	// Variables with changing content, maybe create a separate struct for them (work cache or something)
	status        syncStatus
	curInvs       map[shard.Index]*hashSet // Contains header hashes that we haven't synced
	syncStartTime time.Time
	syncTimer     *syncTimer
}

type hashSet struct {
	hashes map[chainhash.Hash]bool
}

func (s *hashSet) add(h chainhash.Hash) {
	s.hashes[h] = true
}

func (s *hashSet) remove(h chainhash.Hash) {
	delete(s.hashes, h)
}

func (s *hashSet) contains(h chainhash.Hash) bool {
	return s.hashes[h]
}

func (s *hashSet) size() int {
	return len(s.hashes)
}

func newBaseWorker(shardIdx shard.Index) *baseWorker {
	return &baseWorker{
		mailbox:   make(chan interface{}, maxMailBuffer),
		status:    IDLE,
		peerMgr:   newSyncPeerManager(shardIdx),
		shardIdx:  shardIdx,
		syncTimer: &syncTimer{},
	}
}

func (w *baseWorker) setManager(manager reportAPI) {
	w.manager = manager
}

// Act is executed when the given Event is catched, then the callback func will be acted.
func (w *baseWorker) Act(e *message.Event, _ func(m interface{})) {
	switch e.Topic {
	case evtAddPeer:
		mail := e.Extra.(*peerMail)
		w.handleAddPeer(mail.connPeer, mail.nodeType)
	case evtDonePeer:
		mail := e.Extra.(*peerMail)
		w.handlePeerDone(mail.connPeer, mail.nodeType)
	case evtMsg:
		mail := e.Extra.(wire.Message)
		w.handleMsg(mail)
	case evtSyncReq:
		w.handleSyncReq()
	default:
		log.Debugf("%v received unknown mail: %v", w, e.Topic)
	}
}

func (w *baseWorker) handleMsg(msg wire.Message) {
	switch m := msg.(type) {
	case *wire.MsgSyncInv:
		w.handleSyncInv(m)
	default:
		w.diffWorker.handleCustomMsg(m)
	}
}

// Todo(DAN):should be remove?
/*func (w *baseWorker) handleReSendSyncReq(msg *wire.MsgReSendSyncReq) {
	if w.status != RUNNING {
		return
	}
	if syncPeer := w.peerMgr.curSyncPeer; syncPeer != nil {
		log.Infof("%v 当前SyncPeer的数据不满足同步要求，需要更换SyncPeer节点", w)
		if w.peerMgr.removePeerCandidate(syncPeer.cp, syncPeer.nodeType) {
			log.Infof("%v 已将不满足同步要求的存储节点从同步节点列表中删除", w)
		}
	}
	log.Infof("%v 正在重新发起同步数据请求", w)
	w.diffWorker.makeSyncReq()
}*/

func (w *baseWorker) handleSyncInv(msg *wire.MsgSyncInv) {
	if w.status != RUNNING {
		return
	}
	if msg.Type == wire.SyncInvTypeFullList {
		w.recInvFullList(msg.InvGroups)
	} else if msg.Type == wire.SyncInvTypeGetData {
		log.Errorf("%v MsgSyncInv [SyncInvTypeGetData] shouldn't be handled in SyncManager.", w)
	}
}

// Receives the full list of header hashes to sync.
// Needs to be triggered in RUNNING status
func (w *baseWorker) recInvFullList(groups []*wire.InvGroup) {
	w.curInvs = make(map[shard.Index]*hashSet, len(groups))
	hasData := false

	for _, g := range groups {
		if len(g.HeaderHashes) == 0 {
			continue
		}
		hasData = true
		w.curInvs[g.Shard] = &hashSet{hashes: make(map[chainhash.Hash]bool, len(g.HeaderHashes))}
		for _, hash := range g.HeaderHashes {
			w.curInvs[g.Shard].add(hash)
		}
	}
	if hasData {
		w.reqFetchData()
	} else {
		log.Infof("%v stop sync as there is no data to catch up.", w)
		w.markSyncStop(false)
	}
}

// Needs to be triggered in RUNNING status
func (w *baseWorker) reqFetchData() {
	// TODO(huangsz): Set a limit for max number at one time.
	msg := wire.NewMsgSyncInv(w.shardIdx, wire.SyncInvTypeGetData)
	l := len(w.curInvs)
	groups := make([]*wire.InvGroup, 0, l)
	for shardIdx, set := range w.curInvs {
		hashes := make([]chainhash.Hash, 0, set.size())
		for hash := range set.hashes {
			hashes = append(hashes, hash)
		}
		groups = append(groups, wire.NewInvGroup(shardIdx, hashes...))
	}
	msg.InvGroups = groups
	w.sendMsg(msg)
}

// Receives the header hash.
// Needs to be triggered in RUNNING status
func (w *baseWorker) recHeader(s shard.Index, headerHash chainhash.Hash) {
	set := w.curInvs[s]
	if set == nil || !set.contains(headerHash) {
		log.Warnf("%v didn't request or has already received header [shard: %v, header hash: %v]", w, s, headerHash)
		return
	}
	set.remove(headerHash)
	if set.size() == 0 {
		delete(w.curInvs, s)
	}
	if len(w.curInvs) == 0 {
		w.markSyncStop(true)
	}
}

func (w *baseWorker) markSyncStart() {
	log.Infof("%v start syncing", w)
	w.syncTimer.start(w.syncTimeout, syncStaleMin*60)
	w.status = RUNNING
	w.syncStartTime = time.Now()
	w.manager.notifySyncStart()
}

func (w *baseWorker) markSyncStop(dataFetched bool) {
	log.Infof("%v stop syncing, dataFetched: %v", w, dataFetched)
	w.syncTimer.stop()
	w.reset()
	w.manager.notifySyncComplete(dataFetched)
}

func (w *baseWorker) markSyncAborted() {
	log.Infof("%v sync aborted", w)
	w.syncTimer.stop()
	w.reset()
	w.manager.notifySyncAborted()
}

func (w *baseWorker) reset() {
	w.status = IDLE
	w.curInvs = nil
	// The curSyncPeer is set to be nil after sync completed, and it is reselected next time.
	w.peerMgr.curSyncPeer = nil
}

func (w *baseWorker) sendMsg(msg wire.Message) {
	sp := w.peerMgr.getSyncPeer()
	if sp == nil {
		panic("There is no peer to sync from")
	}
	logger.ServerLogger().Debugf("Send sync message")
	sp.cp.Peer.QueueMessage(msg, nil)
}

func (w *baseWorker) addSyncPeer(cp *connection.ConnPeer, t config.NodeType) {
	w.peerMgr.addSyncPeerCandidate(cp, t)
}

func (w *baseWorker) handleAddPeer(cp *connection.ConnPeer, t config.NodeType) {
	// Initialize the peer state
	if w.diffWorker.isSyncCandidate(cp, t) {
		w.addSyncPeer(cp, t)
	}
	//if !params.EnableResharding {
	//	// If disable resharding and the peer is outbound, handle sync.
	//	if !cp.Peer.Inbound() {
	//		w.handleSyncReq()
	//	}
	//}

	if !cp.Peer.Inbound() {
		w.handleSyncReq()
	}
}

func (w *baseWorker) handleSyncReq() {
	log.Debugf("%v received sync request, shouldStartSync %v", w, w.shouldStartSync())
	if w.shouldStartSync() {
		w.startSync()
	}
}

func (w *baseWorker) shouldStartSync() bool {
	return (w.status == IDLE || w.isSyncStale()) && w.peerMgr.getSyncPeer() != nil
}

func (w *baseWorker) startSync() {
	log.Infof("%v Start syncing.", w)
	w.markSyncStart()
	w.diffWorker.makeSyncReq()
}

// Returns whether or not current sync peer has been removed.
func (w *baseWorker) handlePeerDone(cp *connection.ConnPeer, t config.NodeType) {
	// A peer not a sync candidate won't have effect on the worker.
	if !w.diffWorker.isSyncCandidate(cp, t) {
		return
	}
	if w.peerMgr.removePeerCandidate(cp, t) {
		w.syncPeerRemoved()
	}
}

func (w *baseWorker) syncPeerRemoved() {
	if w.status != RUNNING {
		// If syncing is not running, then there is no effect removing one candidate.
		return
	}
	if newSyncPeer := w.peerMgr.getSyncPeer(); newSyncPeer != nil {
		if len(w.curInvs) != 0 {
			// If inv has been received from previous sync peer, request sync data from new sync peer for remaining inv.
			w.reqFetchData()
		} else {
			// If there is no inv list received from previous sync peer, make a new sync request to new sync peer.
			w.diffWorker.makeSyncReq()
		}
	} else {
		// No new sync peer available, aborting the current sync effort.
		w.markSyncAborted()
	}
}

func (w *baseWorker) isSyncStale() bool {
	return time.Since(w.syncStartTime) > time.Duration(syncStaleMin*time.Minute)
}

func (w *baseWorker) String() string {
	return fmt.Sprintf("baseworker {shard: %v}", w.shardIdx.GetID())
}

func (w *baseWorker) syncTimeout() {
	log.Errorf("%v sync time out, abort sync!", w)
	w.markSyncAborted()
}
