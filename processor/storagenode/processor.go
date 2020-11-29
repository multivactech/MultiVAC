/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"github.com/multivactech/MultiVAC/interface/isync"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/p2p/peer"
)

// Processor manages a storage node.
type Processor struct {
	sn       *threadedStorageNode
	logger   btclog.Logger
	syncMgr  isync.SyncManager
	msgQueue chan *connection.MessageAndReply
}

// Start a storage node.
func (pcr *Processor) Start() {
	// init msgQueue
	pcr.msgQueue = make(chan *connection.MessageAndReply, 1000)
	// register channel
	tag := []connection.Tag{
		{Msg: wire.CmdTx, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdBlock, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdTxBatch, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdFetchTxs, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdInitAbciData, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdFetchSmartContractInfo, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdSyncReq, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdSyncInv, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdSlimBlock, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdSyncBlock, Shard: pcr.sn.shardIndex},
		{Msg: wire.CmdSyncSlimBlock, Shard: pcr.sn.shardIndex},
	}
	for _, shardIndex := range shard.ShardList {
		tag = append(tag, connection.Tag{Msg: wire.CmdMsgBlockConfirmation, Shard: shardIndex})
	}
	connection.GlobalConnServer.RegisterChannels(&connection.MessagesAndReceiver{
		Tags:     tag,
		Channels: pcr.msgQueue,
	})
	// handle msg
	go pcr.msgHandler()

	pcr.sn.Start()
}

func (pcr *Processor) msgHandler() {
	for msg := range pcr.msgQueue {
		pcr.HandleMsg(msg.Msg, msg.Reply)
	}
}

// HandleMsg handles messages from peers.
func (pcr *Processor) HandleMsg(msg wire.Message, pr peer.Reply) {
	switch msg := msg.(type) {
	case *wire.MsgTx:
		pcr.sn.OnTransactionReceived(msg)
	case *wire.MsgBlock:
		pcr.sn.OnBlockReceived(msg)
	case *wire.MsgTxBatch:
		for _, tx := range msg.Txs {
			pcr.sn.OnTransactionReceived(&tx)
		}
	case *wire.MsgFetchTxs, *wire.MsgFetchInit, *wire.MsgFetchSmartContractInfo:
		pcr.handleFetchMsg(msg, pr)
	case *wire.MsgSyncReq:
		pcr.handleSyncReq(msg, pr)
	case *wire.MsgSyncInv:
		pcr.handleSyncInv(msg, pr)
	case *wire.SlimBlock:
		pcr.sn.OnSlimBlockReceived(msg)
	case *wire.MsgBlockConfirmation:
		pcr.sn.OnConfirmationReceived(msg)
	case *wire.MsgSyncBlock:
		pcr.syncMgr.AcceptMsg(msg)
	case *wire.MsgSyncSlimBlock:
		pcr.syncMgr.AcceptMsg(msg)
	}
}

// PendingTransactionNumber returns the number of transactions waiting to be packaged in the current trading pool.
func (pcr *Processor) PendingTransactionNumber() int {
	return pcr.sn.PendingTransactionNumber()
}

func (pcr *Processor) handleFetchMsg(msg wire.Message, pr peer.Reply) {
	pcr.sn.OnFetchMessage(msg, pr)
}

func (pcr *Processor) sendSyncData(msg *wire.MsgSyncInv, pr peer.Reply) {
	// 不处理非本分片消息
	if pcr.sn.shardIndex != msg.ReqShardIdx {
		return
	}

	toShard := msg.ReqShardIdx

	for _, group := range msg.InvGroups {
		fromShard := group.Shard

		for _, hash := range group.HeaderHashes {
			slimBlock := pcr.sn.getSlimBlockInShard(fromShard, hash)
			if slimBlock == nil {
				pcr.logger.Errorf("Cannot find slim block for shard: %v with header hash: %v", fromShard, hash)
				continue
			}
			pcr.logger.Infof("同步请求发送slimBlock(shard: %d,hgt: %d)",
				fromShard, slimBlock.Header.Height)
			pr(wire.NewMsgSyncSlimBlock(toShard, slimBlock), nil)
		}

		lastConfirmation := pcr.sn.getLastConfirmation(fromShard)
		if lastConfirmation == nil {
			pcr.logger.Errorf("failed to getLastConfirmation, fromShard: %d,toShard: %d", fromShard, toShard)
			continue
		}
		pr(lastConfirmation, nil)

	}
}

func (pcr *Processor) getHeaderHashes(locators []*wire.BlockLocator) []*wire.InvGroup {
	return pcr.sn.GetHeaderHashes(locators)
}

// ShardHeight returns the current shard height.
func (pcr *Processor) ShardHeight() int64 {
	return pcr.sn.ShardHeight()
}

// GetOutState returns the corresponding outstate according to the specified outpoint.
func (pcr *Processor) GetOutState(outPoint *wire.OutPoint) (*wire.OutState, error) {
	return pcr.sn.GetOutState(outPoint)
}

// GetBlockInfo returns all blocks after the specified height.
func (pcr *Processor) GetBlockInfo(height int64) []*wire.MsgBlock {
	return pcr.sn.GetBlockInfo(height)
}

// GetSlimBlockInfo returns all SlimBlocks after the specified height in the specified shard.
func (pcr *Processor) GetSlimBlockInfo(toShard shard.Index, height int64) []*wire.SlimBlock {
	return pcr.sn.GetSlimBlockInfo(toShard, height)
}

// GetBlockSliceInfoSince returns all blocks after the specified height.
func (pcr *Processor) GetBlockSliceInfoSince(startHeight int64) []*wire.MsgBlock {
	return pcr.sn.GetBlockSliceInfoSince(startHeight)
}

// GetBlockSliceInfoBetween returns all blocks between the specified heights.
func (pcr *Processor) GetBlockSliceInfoBetween(startHeight, endHeight int64) []*wire.MsgBlock {
	return pcr.sn.GetBlockSliceInfoBetween(startHeight, endHeight)
}

// GetSlimBlockSliceInfoSince returns all slimBlocks after the specified height.
func (pcr *Processor) GetSlimBlockSliceInfoSince(toShard shard.Index, startHeight int64) []*wire.SlimBlock {
	return pcr.sn.GetSlimBlockSliceInfoSince(toShard, startHeight)
}

// GetSlimBlockSliceInfoBetween returns all slimblocks between the specified heights.
func (pcr *Processor) GetSlimBlockSliceInfoBetween(toShard shard.Index, startHeight, endHeight int64) []*wire.SlimBlock {
	return pcr.sn.GetSlimBlockSliceInfoBetween(toShard, startHeight, endHeight)
}

// GetAllShardHeights returns the current height of all shards.
func (pcr *Processor) GetAllShardHeights() []shard.IndexAndHeight {
	return pcr.sn.GetAllShardHeights()
}

// SetSyncManager sets the storage node's sync manager.
func (pcr *Processor) SetSyncManager(mgr isync.SyncManager) {
	pcr.syncMgr = mgr
	pcr.sn.setSyncManager(mgr)
}

func (pcr *Processor) handleSyncReq(msg *wire.MsgSyncReq, pr peer.Reply) {
	inv := wire.NewMsgSyncInv(msg.ReqShardIdx, wire.SyncInvTypeFullList)
	inv.InvGroups = pcr.getHeaderHashes(msg.Locators)
	pr(inv, nil)
}

func (pcr *Processor) handleSyncInv(msg *wire.MsgSyncInv, pr peer.Reply) {
	if msg.Type == wire.SyncInvTypeGetData {
		pcr.sendSyncData(msg, pr)
	} else if msg.Type == wire.SyncInvTypeFullList {
		pcr.syncMgr.AcceptMsg(msg)
	}
}

// GetSmartContractInfo returns smartContractInfo for specified shard and contract address.
func (pcr *Processor) GetSmartContractInfo(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.SmartContractInfo {
	return pcr.sn.GetSmartContractInfo(contractAddr, shardIdx)
}
