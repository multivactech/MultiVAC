/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/interface/isync"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/p2p/peer"
	"github.com/multivactech/MultiVAC/processor/miner"
	"github.com/multivactech/MultiVAC/processor/miner/consensus"
	bcp "github.com/multivactech/MultiVAC/processor/shared/blockconfirmproductor"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/storagenode"
	mtvsync "github.com/multivactech/MultiVAC/processor/sync"
)

// shardController represents a shard that is responsible for scheduling the business logic components
// in the shard. If the node is a storage node, then it will manage storagenode, otherwise manage abci.
//
// shardController created by RootController and it's state is managed by shardManager.
type shardController struct {
	shardIndex shard.Index
	// TODO(huangsz): Can be referred by cfg
	isStandaloneStorageNode bool // Used to identify whether it is storagenode
	enabled                 bool
	mtvSyncMgr              isync.SyncManager
	storagePcr              *storagenode.Processor
	minerPcr                *miner.Processor
	blockChain              iblockchain.BlockChain
	depositPool             idepositpool.DepositPool
	pubSubMgr               *message.PubSubManager
	bcProducer              bcp.BcProducerEnabler
	logger                  btclog.Logger
	cfg                     *config.Config

	mu sync.RWMutex
}

// Init performs initialization operations for logical components. It does not
// put the component in operation.
// enabled indicates if the shardController is in enabled status, which in practice means it will receive message for
// this shard.
func (sp *shardController) Init(enabled bool) {
	if !sp.isStandaloneStorageNode {
		sp.minerPcr.Init()
		sp.minerPcr.RegisterCEListener(&consensusStateListener{sp})
	}
	sp.mtvSyncMgr.Start()
	sp.setShardEnableStatus(enabled)
}

// Start makes all components in working.
func (sp *shardController) Start(isIntegrationTest bool) {
	if sp.isStandaloneStorageNode {
		sp.storagePcr.Start()
	} else {
		// If it is the first miner, set the prevReshardHeader GenesisBlocks header when start
		prevReshardHeader := &genesis.GenesisBlocks[sp.shardIndex].Header
		sp.minerPcr.StartABCI(prevReshardHeader)
	}
}

// ShardIndex returns the index of the shardController
func (sp *shardController) ShardIndex() shard.Index {
	return sp.shardIndex
}

// StartService makes the shardController start servicing in its own shard, often called when re-sharding selects this
// instance to be in the given shard. Called in miner instance only.
// consensusStartRound the round to start consensus on
// curHeader the header of the current round to start service from, aka "当前re-shard轮公示出来的block header"
// enabledShards all enabled shards in the current network
func (sp *shardController) StartService(
	consensusStartRound int, curHeader *wire.BlockHeader, enabledShards []shard.Index) {
	sp.minerPcr.SetStartConsensusAtRound(consensusStartRound)
	sp.setShardEnableStatus(true)
	sp.minerPcr.StartABCI(curHeader)
}

// StopService makes the shardController stop servicing in its own shard, often called when re-sharding selects this
// instance out of the given shard. After stopped, this shard will not receive any further message until it is started
// again. Called in miner instance only.
// enabledShards all enabled shards in the current network
func (sp *shardController) StopService() {
	sp.minerPcr.StopABCI()
	sp.setShardEnableStatus(false)
}

// SuspendService will suspend the instance, making it not to contributing to the mining behavior.
// This will not disable the shard, so that the shard keeps receiving incoming messages, but it will stop contributing
// to consensus (hence stop contributing to mining). Called in miner instance only.
func (sp *shardController) SuspendService(consensusStopRound int) {
	sp.minerPcr.StopConsensusAtRound(consensusStopRound)
}

// IsEnabled returns if the processor is in working.
func (sp *shardController) IsEnabled() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.enabled
}

// UpdateProof is used to update the proof for the node instance. Currently it only applies to miner
func (sp *shardController) UpdateProof(proof []byte) {
	if !sp.isStandaloneStorageNode {
		sp.minerPcr.UpdateMinerProof(proof)
	}
}

func (sp *shardController) setShardEnableStatus(status bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.enabled = status
}

// handleStorageNodeMsg handles storage node relevant logic.
// p indicates the source of the message, can be nil (for example, msg from rpc not from server)
func (sp *shardController) handleStorageNodeMsg(msg wire.Message, p peer.Reply) {
	sp.storagePcr.HandleMsg(msg, p)
}

// NotifyNewPeerAdded sends new peer added information to mtvSyncMgr
func (sp *shardController) NotifyNewPeerAdded(cp *connection.ConnPeer, t config.NodeType) {
	// TODO(issue #126, tianhongce): UpdatePeerInfo will trigger NotifyNewPeerAdded
	if !cp.Peer.Inbound() {
		// 延时触发OnNewPeer的目的是让新启动的存储节点收取20s消息后，再触发OnNewPeer，进而触发同步，否则会出现数据断层的问题
		if sp.isStandaloneStorageNode {
			go func() {
				time.Sleep(20 * time.Second)
				sp.mtvSyncMgr.OnNewPeer(cp, t)
			}()
			return
		}
		// sp.mtvSyncMgr.OnNewPeer(cp, t)
	}
}

// NotifyPeerDone sends the new-done information to syncManager.
func (sp *shardController) NotifyPeerDone(cp *connection.ConnPeer, t config.NodeType) {
	sp.mtvSyncMgr.OnPeerDone(cp, t)
}

func (sp *shardController) String() string {
	return fmt.Sprintf("ShardProcess [%v]", sp.shardIndex.GetID())
}

// GetOutState returns *wire.OutState if the node is storagenode
func (sp *shardController) getOutState(outPoint *wire.OutPoint) *wire.OutState {
	if sp.isStandaloneStorageNode {
		r, err := sp.storagePcr.GetOutState(outPoint)
		if err != nil {
			sp.logger.Errorf("GetOutState failed for shard: %v, err: %v", sp.shardIndex, err)
		} else {
			return r
		}
	}
	return nil
}

// getBlockInfo returns the block information by the specified height.
func (sp *shardController) getBlockInfo(height int64) []*wire.MsgBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetBlockInfo(height)
	}
	return nil
}

// getSlimBlockInfo returns the slimblock information by the specified toshard and height.
func (sp *shardController) getSlimBlockInfo(toShard shard.Index, height int64) []*wire.SlimBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetSlimBlockInfo(toShard, height)
	}
	return nil
}

// getBlockSliceInfoSince returns the slice of blocks information by the specified start height.
func (sp *shardController) getBlockSliceInfoSince(startHeight int64) []*wire.MsgBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetBlockSliceInfoSince(startHeight)
	}
	return nil
}

// getBlockSliceInfoBetween returns the block information of the specified interval.
func (sp *shardController) getBlockSliceInfoBetween(startHeight, endHeight int64) []*wire.MsgBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetBlockSliceInfoBetween(startHeight, endHeight)
	}
	return nil
}

// getSlimBlockSliceInfoSince returns the slice of slimblocks information by the specified start height and toShardIndex.
func (sp *shardController) getSlimBlockSliceInfoSince(toShardIndex shard.Index, startHeight int64) []*wire.SlimBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetSlimBlockSliceInfoSince(toShardIndex, startHeight)
	}
	return nil
}

// getSlimBlockSliceInfoBetween returns the slimblock information of the specified interval and toShardIndex.
func (sp *shardController) getSlimBlockSliceInfoBetween(toShardIndex shard.Index, startHeight, endHeight int64) []*wire.SlimBlock {
	if sp.isStandaloneStorageNode {
		return sp.storagePcr.GetSlimBlockSliceInfoBetween(toShardIndex, startHeight, endHeight)
	}
	return nil
}

// GetShardHeight returns the height of this shard
func (sp *shardController) GetShardHeight() int64 {
	return int64(sp.blockChain.GetShardsHeight(sp.shardIndex))
}

// ReceiveSyncHeader used to receive header from sync.
func (sp *shardController) ReceiveSyncHeader(header *wire.BlockHeader) {
	sp.minerPcr.ReceiveSyncHeader(header)
}

// getSmartContractInfo returns smartContractInfo for contract address in current shard.
func (sp *shardController) getSmartContractInfo(contractAddr multivacaddress.Address) *wire.SmartContractInfo {
	return sp.storagePcr.GetSmartContractInfo(contractAddr, sp.shardIndex)
}

// HandleRPCReq handles a RPCReq for the relevant shard.
func (sp *shardController) HandleRPCReq(req *message.RPCReq) *message.RPCResp {
	switch req.Event {
	case message.RPCGetOutState:
		o := req.Args[0].(*wire.OutPoint)
		if r := sp.getOutState(o); r != nil {
			return message.NewSuccessRPCResp(r)
		}
	case message.RPCGetBlockInfo:
		height := req.Args[0].(int64)
		return message.NewSuccessRPCResp(sp.getBlockInfo(height))
	case message.RPCGetSlimBlockInfo:
		toShard := req.Args[0].(shard.Index)
		height := req.Args[1].(int64)
		return message.NewSuccessRPCResp(sp.getSlimBlockInfo(toShard, height))
	case message.RPCGetBlockSliceInfo:
		startHeight := req.Args[0].(int64)
		if len(req.Args) == 1 {
			return message.NewSuccessRPCResp(sp.getBlockSliceInfoSince(startHeight))
		}
		endHeight := req.Args[1].(int64)
		return message.NewSuccessRPCResp(sp.getBlockSliceInfoBetween(startHeight, endHeight))
	case message.RPCGetSlimBlockSliceInfo:
		toShard := req.Args[0].(shard.Index)
		startHeight := req.Args[1].(int64)
		if len(req.Args) == 2 {
			return message.NewSuccessRPCResp(sp.getSlimBlockSliceInfoSince(toShard, startHeight))
		}
		endHeight := req.Args[2].(int64)
		return message.NewSuccessRPCResp(sp.getSlimBlockSliceInfoBetween(toShard, startHeight, endHeight))
	case message.RPCGetSCInfo:
		addr := req.Args[0].(multivacaddress.Address)
		return message.NewSuccessRPCResp(sp.getSmartContractInfo(addr))
	case message.RPCSendRawTx:
		sp.handleStorageNodeMsg(req.Args[0].(*wire.MsgTx), nil)
		return message.NewSuccessRPCResp(nil)
	}
	return message.NewFailedRPCResp()
}

// newShardController Creates a new shardController, by default the shard processor will be disabled after creation.
// To use it one must call StartService() first.
func newShardController(
	info *ValidationInfo,
	shardIndex shard.Index,
	blockChain iblockchain.BlockChain,
	depositPool idepositpool.DepositPool,
	cfg *config.Config,
	heartbeatMgr iheartbeat.HeartBeat,
	metrics *metrics.Metrics) (*shardController, error) {

	sp := &shardController{
		shardIndex:              shardIndex,
		isStandaloneStorageNode: cfg.StorageNode,
		blockChain:              blockChain,
		depositPool:             depositPool,
		cfg:                     cfg,
	}
	// pubSubMgr initialize.
	sp.pubSubMgr = message.NewPubSubManager()
	sp.bcProducer = bcp.NewBcProducer(shardIndex, heartbeatMgr, info.Selector)
	// If config for storage node, start storagenode or start miner service.
	if cfg.StorageNode {
		sp.storagePcr = storagenode.NewStorageProcessor(
			shardIndex, storagenode.StorageDir(config.GlobalConfig().DataDir), sp.blockChain, sp.depositPool, heartbeatMgr)
	} else {
		sp.minerPcr = miner.NewProcessor(
			sp.blockChain,
			consensus.CreateParams(info.Pk, info.Sk, cfg.VoteThreshold, cfg.IntegrationTest, sp.shardIndex, heartbeatMgr, sp.pubSubMgr, metrics),
			shardIndex, info.Selector, sp.depositPool, sp.pubSubMgr)
	}
	// Set syncManagerr for storagenode or miner node.
	sp.mtvSyncMgr = mtvsync.NewSyncManager(shardIndex, sp.storagePcr, sp, sp.pubSubMgr)
	if sp.storagePcr != nil {
		sp.storagePcr.SetSyncManager(sp.mtvSyncMgr)
	}
	// set sync trigger to chain
	sp.blockChain.SetSyncTrigger(shardIndex, sp.mtvSyncMgr)
	// log config
	sp.logger = logBackend.Logger(fmt.Sprintf("SHRP%d", sp.shardIndex.GetID()))
	sp.logger.SetLevel(logger.ShrpLogLevel)
	return sp, nil
}

// ConsensusStateListener used to receive messgae from consensus
type consensusStateListener struct {
	sp *shardController
}

// OnShardDisabled sets the shard processor to disabled
func (cal *consensusStateListener) OnShardDisabled() {
	cal.sp.logger.Debugf("receive shard disabled signal, set the shard disabled")
	cal.sp.StopService()
	cal.sp.logger.Debugf("blockchain stopped")
}
