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
	"github.com/multivactech/MultiVAC/p2p/peer"
)

type storageWorker struct {
	*baseWorker
	sn storageSyncContract
}

// The contract of features required by sync in order to deal with storage node syncing problems.
type storageSyncContract interface {
	// Handles storage node relevant logic.
	// p indicates the source of the message, can be nil (for example, msg from rpc not from server)
	HandleMsg(msg wire.Message, p peer.Reply)

	// GetAllShardHeights returns the current height of all shards
	GetAllShardHeights() []shard.IndexAndHeight
}

func newStorageWorker(shardIdx shard.Index, sn storageSyncContract) *storageWorker {
	w := &storageWorker{
		baseWorker: newBaseWorker(shardIdx),
		sn:         sn,
	}
	w.baseWorker.diffWorker = w
	return w
}

func (w *storageWorker) OnFreqLedgerUpdateFail() {
	w.mailbox <- &storageMissInfo{}
}

func (w *storageWorker) handleCustomMsg(msg wire.Message) {
	switch m := msg.(type) {
	case *wire.MsgSyncBlock:
		w.handleSyncBlock(m.Block)
	case *wire.MsgSyncSlimBlock:
		w.handleSyncSlimBlock(m.SlimBlock)
	}
}

func (w *storageWorker) handleSyncBlock(block *wire.MsgBlock) {
	if w.status != RUNNING {
		return
	}
	log.Infof("%v handleSyncBlock:  fromShard: %v  height: %d", w, block.Header.ShardIndex, block.Header.Height)
	w.sn.HandleMsg(block, nil)
	w.recHeader(block.Header.ShardIndex, block.Header.BlockHeaderHash())
}

func (w *storageWorker) handleSyncSlimBlock(block *wire.SlimBlock) {
	if w.status != RUNNING {
		return
	}
	log.Infof("%v handleSyncSlimBlock:  fromShard: %v  height: %d", w, block.Header.ShardIndex, block.Header.Height)
	w.sn.HandleMsg(block, nil)
	w.recHeader(block.Header.ShardIndex, block.Header.BlockHeaderHash())
}

func (w *storageWorker) makeSyncReq() {
	w.sendMsg(w.createSyncReq())
}

func (w *storageWorker) createSyncReq() *wire.MsgSyncReq {
	msg := wire.NewMsgSyncReq(w.shardIdx)
	for _, idxAndHgt := range w.sn.GetAllShardHeights() {
		msg.AddBlockLocatorToRecent(idxAndHgt.Index, idxAndHgt.Height+1)
	}
	return msg
}

// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (w *storageWorker) isSyncCandidate(cp *connection.ConnPeer, t config.NodeType) bool {
	return !cp.Peer.Inbound() && t == config.StorageNode
}
