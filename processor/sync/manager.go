/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// The SyncManager implementation.
type syncManager struct {
	shard    shard.Index
	worker   syncWorker
	state    *syncState
	actorCtx *message.ActorContext
}

// Start the process of the worker of the syncManager.
func (mgr *syncManager) Start() {
	mgr.actorCtx.StartActor(mgr.worker)
}

// AcceptMsg sends the message to the worker of the syncManager.
func (mgr *syncManager) AcceptMsg(msg wire.Message) {
	mgr.actorCtx.Send(mgr.worker, message.NewEvent(evtMsg, msg), nil)
}

// OnNewPeer sends the message of new peer added in to the worker of the syncManager.
func (mgr *syncManager) OnNewPeer(cp *connection.ConnPeer, t config.NodeType) {
	mgr.actorCtx.Send(mgr.worker, message.NewEvent(evtAddPeer, &peerMail{connPeer: cp, nodeType: t}), nil)
}

// OnPeerDone sends the message of peer done to the worker of the syncManager.
func (mgr *syncManager) OnPeerDone(cp *connection.ConnPeer, t config.NodeType) {
	mgr.actorCtx.Send(mgr.worker, message.NewEvent(evtDonePeer, &peerMail{connPeer: cp, nodeType: t}), nil)
}

// MaybeSync send a syn request to the worker of the syncManager.
func (mgr *syncManager) MaybeSync() {
	log.Debugf("%v maybe sync", mgr.worker)
	if mgr.state.status != RUNNING {
		mgr.actorCtx.Send(mgr.worker, message.NewEvent(evtSyncReq, &syncReq{}), nil)
	}
}

// callback from worker
type reportAPI interface {
	notifySyncStart()

	// dataFetched indicates if there is any data (block/header) fetched in this sync.
	notifySyncComplete(dataFetched bool)

	notifySyncAborted()
}

func (mgr *syncManager) notifySyncStart() {
	mgr.state.maybeStart()
}

func (mgr *syncManager) notifySyncComplete(dataFetched bool) {
	mgr.state.maybeStop(dataFetched)
}

func (mgr *syncManager) notifySyncAborted() {
	mgr.state.maybeAbort()
}
