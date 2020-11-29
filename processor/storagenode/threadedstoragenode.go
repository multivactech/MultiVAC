/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"github.com/multivactech/MultiVAC/interface/isync"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/peer"

	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

type threadedStorageNode struct {
	shardIndex shard.Index
	sn         *storageNode
	actorCtx   *message.ActorContext
}

func newThreadedStorageNode(shardIndex shard.Index, sn *storageNode) *threadedStorageNode {
	tsn := &threadedStorageNode{
		shardIndex: shardIndex,
		actorCtx:   message.NewActorContext(),
		sn:         sn,
	}
	return tsn
}

func (tsn *threadedStorageNode) Start() {
	tsn.sn.init()
	tsn.actorCtx.StartActor(tsn.sn)
}

func (tsn *threadedStorageNode) OnTransactionReceived(msg *wire.MsgTx) {
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtMsgTx, msg), nil)
}

func (tsn *threadedStorageNode) OnBlockReceived(msg *wire.MsgBlock) {
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtMsgBlock, msg), nil)
}

func (tsn *threadedStorageNode) OnSlimBlockReceived(msg *wire.SlimBlock) {
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtSlimBlock, msg), nil)
}

func (tsn *threadedStorageNode) OnFetchMessage(msg wire.Message, pr peer.Reply) {
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtFetchMsg, fetchRequest{msg: msg, peerReply: pr}), nil)
}

func (tsn *threadedStorageNode) GetOutState(outPoint *wire.OutPoint) (*wire.OutState, error) {
	rsp := tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetOutStateReq, outPoint)).(*getOutStateResponse)
	return rsp.outState, rsp.err
}

func (tsn *threadedStorageNode) PendingTransactionNumber() int {
	if tsn.sn != nil {
		return tsn.sn.pendingTransactionNumber()
	}
	return 0
}

func (tsn *threadedStorageNode) ShardHeight() int64 {
	if tsn.sn != nil {
		return tsn.sn.ShardHeight()
	}
	return int64(0)
}

// TODO(huangsz): This looks very likely to have concurrency issue, fix it.
func (tsn *threadedStorageNode) GetBlockInfo(height int64) []*wire.MsgBlock {
	return tsn.sn.getBlockInfo(height)
}

func (tsn *threadedStorageNode) GetSlimBlockInfo(toShard shard.Index, height int64) []*wire.SlimBlock {
	return tsn.sn.getSlimBlockInfo(toShard, height)
}

func (tsn *threadedStorageNode) GetBlockSliceInfoSince(startHeight int64) []*wire.MsgBlock {
	return tsn.sn.getBlockSliceInfoSince(startHeight)
}

func (tsn *threadedStorageNode) GetBlockSliceInfoBetween(startHeight, endHeight int64) []*wire.MsgBlock {
	return tsn.sn.getBlockSliceInfoBetween(startHeight, endHeight)
}

func (tsn *threadedStorageNode) GetSlimBlockSliceInfoSince(to shard.Index, startHeight int64) []*wire.SlimBlock {
	return tsn.sn.getSlimBlockSliceInfoSince(to, startHeight)
}

func (tsn *threadedStorageNode) GetSlimBlockSliceInfoBetween(to shard.Index, startHeight, endHeight int64) []*wire.SlimBlock {
	return tsn.sn.getSlimBlockSliceInfoBetween(to, startHeight, endHeight)
}

func (tsn *threadedStorageNode) GetAllShardHeights() []shard.IndexAndHeight {
	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetAllShardHgts, nil)).([]shard.IndexAndHeight)
}

func (tsn *threadedStorageNode) GetHeaderHashes(locators []*wire.BlockLocator) []*wire.InvGroup {
	rsp := tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetHeaderHashes, &reqHeaderHashes{locators: locators})).(resHeaderHashes)
	return rsp.headerGroups
}

//func (tsn *threadedStorageNode) getBlockInShard(s shard.Index, header chainhash.Hash) *wire.MsgBlock {
//	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetBlockReq, &getBlockReq{shardIdx: s, headerHash: header})).(*wire.MsgBlock)
//}

func (tsn *threadedStorageNode) getLastConfirmation(s shard.Index) *wire.MsgBlockConfirmation {
	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetLastConfirmation, s)).(*wire.MsgBlockConfirmation)
}

func (tsn *threadedStorageNode) getSlimBlockInShard(s shard.Index, header chainhash.Hash) *wire.SlimBlock {
	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetSlimBlockReq, &getBlockReq{shardIdx: s, headerHash: header})).(*wire.SlimBlock)
}

func (tsn *threadedStorageNode) setSyncManager(mgr isync.SyncManager) {
	tsn.sn.setSyncMgr(mgr)
}

func (tsn *threadedStorageNode) GetSmartContractByAddress(addr chainhash.Hash) *wire.SmartContract {
	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetSmartContract, addr)).(*wire.SmartContract)
}

func (tsn *threadedStorageNode) GetSmartContractInfo(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.SmartContractInfo {
	scAddrShard := []interface{}{contractAddr, shardIdx}
	return tsn.actorCtx.SendAndWait(tsn.sn, message.NewEvent(evtGetSmartContractInfo, scAddrShard)).(*wire.SmartContractInfo)
}

func (tsn *threadedStorageNode) OnConfirmationReceived(msg *wire.MsgBlockConfirmation) {
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtMsgBlockConfirmation, msg), nil)
}

/*func (tsn *threadedStorageNode) addSmartContractInfos(infos []*wire.SmartContractInfo, root *merkle.MerkleHash) {
	infoAndRoot := []interface{}{infos, root}
	tsn.actorCtx.Send(tsn.sn, message.NewEvent(evtAddSmartContractInfosToDataStore, infoAndRoot), nil)
}*/
