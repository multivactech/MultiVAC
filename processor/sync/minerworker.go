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
)

type minerWorker struct {
	*baseWorker

	minerContract minerSyncContract
}

// The contract of features required by sync in order to deal with miner syncing problems.
type minerSyncContract interface {
	GetShardHeight() int64
	ReceiveSyncHeader(header *wire.BlockHeader)
}

func newMinerWorker(shardIdx shard.Index, minerContract minerSyncContract) *minerWorker {
	w := &minerWorker{
		baseWorker:    newBaseWorker(shardIdx),
		minerContract: minerContract,
	}
	w.baseWorker.diffWorker = w
	return w
}

func (w *minerWorker) handleCustomMsg(msg wire.Message) {
	switch msg := msg.(type) {
	case *wire.MsgSyncHeaders:
		if msg.ReqShardIdx != w.shardIdx {
			log.Errorf("%v cannot deal with MsgSyncHeaders from another shard (%v)", w, msg.ReqShardIdx.GetID())
			return
		}
		for _, header := range msg.Headers {
			w.minerContract.ReceiveSyncHeader(header)
			w.recHeader(header.ShardIndex, header.BlockHeaderHash())
		}
	}
}

func (w *minerWorker) makeSyncReq() {
	w.sendMsg(w.createSyncReq())
}

// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (w *minerWorker) isSyncCandidate(cp *connection.ConnPeer, t config.NodeType) bool {
	return !cp.Peer.Inbound() && t == config.MinerNode
}

func (w *minerWorker) createSyncReq() *wire.MsgSyncReq {
	s := w.shardIdx
	msg := wire.NewMsgSyncReq(s)
	hgt := w.minerContract.GetShardHeight()
	msg.AddBlockLocatorToRecent(s, hgt+1)
	return msg
}
