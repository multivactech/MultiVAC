/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/multivactech/MultiVAC/base/db"
	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/interface/istate"
	"github.com/multivactech/MultiVAC/interface/isync"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/state"
	"github.com/multivactech/MultiVAC/processor/shared/txpool"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	updateLedgerReportThreshold = 15
	debug                       = true
)

var (
	emptyMerkleHash = merkle.MerkleHash{}
)

type storageNode struct {
	// If use disk merkle tree, this path should be provided.
	dataDir string

	// A full merkle tree for storage node to store the ledger info.
	ledgerTree *merkle.FullMerkleTree

	// Which shard is this storage node working for.
	// TODO: Support shard list, rather than only one shard.
	shardIndex shard.Index

	// The height of the wire in this shard that has been confirmed and stored in this node.
	shardHeight int64

	// It records all shards' height and total blocks.
	ledgerInfo wire.LedgerInfo

	// Storage node maintain a txpool to store txs.
	txpool txpool.TxPool

	state istate.LedgerStateManager

	// A blockchain instance to store ledger info.
	blockChain iblockchain.BlockChain

	// dPool used to store all deposit outs that belong to this shard.
	dPool idepositpool.DepositPool

	// heartBeatManager used to managed the heartbeat
	heartBeatManager iheartbeat.HeartBeat

	// Can be set lazily.
	syncMgr isync.SyncManager

	db db.DB

	updateFailCnt int

	logger btclog.Logger

	mutex sync.RWMutex

	// Cache the confirmation message.
	confirmedHeaders map[chainhash.Hash]bool

	// Cache the last confirmation message
	lastConfirmation map[shard.Index]*wire.MsgBlockConfirmation

	// Cache the blocks whose confirmaton message has not reached.
	receivedBlocks map[chainhash.Hash]*wire.SlimBlock

	fetcher *storageNodeFetcher

	depositManager *depositManager

	//smartContractDataStore smartcontractdatastore.SmartContractDataStore // smartContractDataPool that storage node maintained
}

func newStorageNode(shardIndex shard.Index, dataDir StorageDir, blockChain iblockchain.BlockChain,
	dPool idepositpool.DepositPool, heartBeatManager iheartbeat.HeartBeat) *storageNode {
	return &storageNode{
		shardIndex:       shardIndex,
		dataDir:          string(dataDir),
		updateFailCnt:    0,
		blockChain:       blockChain,
		confirmedHeaders: make(map[chainhash.Hash]bool),
		lastConfirmation: make(map[shard.Index]*wire.MsgBlockConfirmation),
		receivedBlocks:   make(map[chainhash.Hash]*wire.SlimBlock),
		dPool:            dPool,
		heartBeatManager: heartBeatManager,
		depositManager:   newDepositManager(),
	}
}

func (node *storageNode) init() {
	// log config
	node.logger = logBackend.Logger(fmt.Sprintf("STRG%d", node.shardIndex.GetID()))
	node.logger.SetLevel(logger.StrgLogLevel)

	// initialization ledger and txpool
	node.initFromDisk()
	node.initTxPool()

	// Initialize fetcher for receiving requested data
	node.fetcher = newStorageNodeFetcher()
	node.fetcher.start()

}

// Todo: unused now
/*func (node *storageNode) stop(){
	node.snf.stop()
	close(node.fetchTxReq)
	close(node.fetchSCReq)
	close(node.fetchInitReq)
}*/

// checkClearReceivedBlock根据被共识的slimBlock来检验其preBlock并执行指定操作
func (node *storageNode) checkClearReceivedBlocks(preHeaderHash chainhash.Hash) {

	node.confirmedHeaders[preHeaderHash] = true

	slimBlock, ok := node.receivedBlocks[preHeaderHash]
	for ok {
		node.blockChain.ReceiveSlimBlock(slimBlock)
		delete(node.receivedBlocks, preHeaderHash)
		delete(node.confirmedHeaders, preHeaderHash)

		preHeaderHash = slimBlock.Header.PrevBlockHeader
		node.confirmedHeaders[preHeaderHash] = true
		slimBlock, ok = node.receivedBlocks[preHeaderHash]
	}
}

// canUpdateLedger calculates whether the current ledger can be updated.
func (node *storageNode) canUpdateLedger() bool {
	// Check if the next block has been received
	hgt := wire.BlockHeight(node.shardHeight + 1)
	slimblock := node.blockChain.GetSlimBlock(node.shardIndex, node.shardIndex, hgt)
	if slimblock == nil {
		node.logger.Errorf("Shard %d canUpdateLedger false. Missing slimblock with height %d in this shard.",
			node.shardIndex.GetID(), node.shardHeight+1)
		return false
	}

	// Try to check clock vector
	ledgerInfo := slimblock.LedgerInfo
	for _, shardIndex := range shard.ShardList {
		newHeight := ledgerInfo.GetShardHeight(shardIndex)
		previousHeight := node.ledgerInfo.GetShardHeight(shardIndex)
		for height := previousHeight + 1; height <= newHeight; height++ {
			// Skip genesis block
			if height == 1 {
				continue
			}
			if b := node.blockChain.GetSlimBlock(node.shardIndex, shardIndex, wire.BlockHeight(height)); b == nil {
				node.logger.Warnf("Shard %d canUpdateLedger false. Missing slimblock shard: %d, height: %d, prevHeight: %d, newHeight: %d",
					node.shardIndex.GetID(), shardIndex.GetID(), height, previousHeight, newHeight)
				return false
			}
		}
	}
	return true
}

// maybeUpdateLedger try to update the status of storage node's ledgerTree.
// It will take three steps:
// 1. Mark all txIn.previousOutState to spend state.
// 2. Update sharding data
// 3. Get all the blocks recorded in xx, add out tree to the ledger.
func (node *storageNode) maybeUpdateLedger() {
	// Check if it can be updated.
	if !node.canUpdateLedger() {
		return
	}
	node.shardHeight++

	// Get the block of the current shard to update leger.
	slimblock := node.blockChain.GetSlimBlock(node.shardIndex, node.shardIndex, wire.BlockHeight(node.shardHeight))
	if slimblock == nil {
		panic("Slimblock is nil but update ledger")
	}
	node.logger.Debugf("Storage node update ledger, shard: %v, height: %v, block header height: %v, empty block: %v",
		node.shardIndex.GetID(), node.shardHeight, slimblock.Header.Height, slimblock.Header.IsEmptyBlock)

	// If the block is an empty block created by consensus, don't need to update merkle tree and merkle proofs.
	if slimblock.Header.IsEmptyBlock {
		node.logger.Debugf("Shard %d received empty block, state will not update the height: %d",
			node.shardIndex.GetID(), slimblock.Header.Height)
		// We need to persist the current state to the database to support the restart.
		err := node.persistentInfo()
		if err != nil {
			node.logger.Errorf("Persistent info to db failed after update ledger: %v.")
		}
		return
	}

	// Modify txouts to un-spend state to already consumed state.
	for _, tx := range slimblock.Transactions {
		if tx.Tx.IsReduceTx() {
			continue
		}
		for _, txIn := range tx.Tx.TxIn {
			previousOutState := wire.OutState{
				OutPoint: txIn.PreviousOutPoint,
				State:    wire.StateUnused,
			}
			previousHash := merkle.MerkleHash(sha256.Sum256(previousOutState.ToBytesArray()))
			newOutState := wire.OutState{
				OutPoint: txIn.PreviousOutPoint,
				State:    wire.StateUsed,
			}
			newHash := merkle.MerkleHash(sha256.Sum256(newOutState.ToBytesArray()))
			err := node.ledgerTree.UpdateLeaf(&previousHash, &newHash)
			if err != nil && debug {
				panic(fmt.Sprintf("Error updating merkle tree, err msg: %s", err.Error()))
			}
			if tx.Tx.API == isysapi.SysAPIWithdraw {
				if txIn.PreviousOutPoint.IsDepositOut() {
					_ = node.dPool.Lock(&txIn.PreviousOutPoint)
				}
			}
		}
	}

	// Try to add deposit proof to pool according to the past blocks.
	node.tryToSaveDeposit()

	// Update out's proof in txpool and deposit pool
	//
	// If current ledger height is H, when a new block is comming,
	// the ledger root is updated to lastest, for example H'. Before
	// applying the update, the out in the pool needs to be updated.
	err := node.updateStateBySlimblock(slimblock)
	if err != nil {
		node.logger.Error(err)
		panic(err)
	}

	// After updating the state, record that this block has not been processed
	// for deposit. Because this block has not been updated to the ledger tree.
	node.depositManager.enQueue(wire.BlockHeight(slimblock.Header.Height))

	// Update full merkle tree according to the new block and the ledgerInfo in it.
	ledgerInfo := slimblock.LedgerInfo
	node.ledgerInfo.Size = slimblock.LedgerInfo.Size
	for _, shardIndex := range shard.ShardList {
		// LedgerInfo records the height of all shards, it looks like this:
		// | 1: 10 |
		// | 2: 11 |
		// | 3: 12 |
		// | 4: 10 |
		// When the current shard updates the ledger, the corresponding block
		// information needs to be obtained according to the height of other
		// shards recorded in the ledgerinfo.
		newHeight := ledgerInfo.GetShardHeight(shardIndex)
		preHeight := node.ledgerInfo.GetShardHeight(shardIndex)

		// Storage node locally records the current height of other shards, when
		// it updates the ledger, it needs to compare the new height(records in new block)
		// with the old height and update all the block information between the heights in turn.
		//
		// For example:
		// Local ledgerInfo: [ 1:10 ]  New ledgerInfo: [ 1:13 ]
		// The blocks need to update: block 12 and block 13.
		// Note: storage node should update all shards' block information.
		for height := preHeight + 1; height <= newHeight; height++ {
			// For genesis block, we should only append the block merkle root(BMT) to ledger tree.
			if height == 1 {
				err := node.ledgerTree.Append(&genesis.GenesisBlocks[shardIndex].Header.OutsMerkleRoot)
				if err != nil {
					panic(err)
				}
				err = node.ledgerTree.HookSecondLayerTree(genesis.GenesisBlocksOutsMerkleTrees[shardIndex])
				if err != nil {
					panic(err)
				}
				continue
			}
			var header wire.BlockHeader
			var updateActions []*wire.UpdateAction
			var err error
			if shardIndex.GetID() == node.shardIndex.GetID() {
				pastBlock := node.blockChain.GetSlimBlock(shardIndex, shardIndex, wire.BlockHeight(height))
				if pastBlock == nil && debug {
					panic(fmt.Sprintf(
						"Error updating merkle tree in shard %d, pastBlock (Shard: %d, Height: %d) not received yet. Previous height: %d\n",
						shard.IndexToID(node.shardIndex),
						shard.IndexToID(shardIndex),
						height, preHeight))
				}
				updateActions = pastBlock.UpdateActions
			}

			slimBlock := node.blockChain.GetSlimBlock(node.shardIndex, shardIndex, wire.BlockHeight(height))
			if slimBlock == nil && debug {
				panic(fmt.Sprintf(
					"Error updating merkle tree in shard %d, pastMsg (Shard: %d, Height: %d) not received yet. Previous height: %d\n",
					shard.IndexToID(node.shardIndex),
					shard.IndexToID(shardIndex),
					height, preHeight))
			}

			if shardIndex.GetID() == node.shardIndex.GetID() {
				updateActions = slimBlock.UpdateActions
			}

			header = slimBlock.Header
			if header.IsEmptyBlock {
				node.logger.Debugf("%v node receive empty slimblock shard:%v, height : %v\n", node.shardIndex, header.ShardIndex, header.Height)
				continue
			}

			clipMsg := slimBlock.ClipTreeData
			if node.shardIndex != slimBlock.ToShard {
				node.logger.Errorf("%v node receive wrong slimblock shouldtoshard : %v\n", node.shardIndex, slimBlock.ToShard)
				return
			}

			fullMerkleTree, err := clipMsg.Rebuild(header.OutsMerkleRoot, slimBlock.ToShard)
			if err != nil {
				node.logger.Errorf("clip tree rebuild error : %v\n", err)
				return
			}
			merkleRoot := fullMerkleTree.MerkleRoot

			// Compare the root that in block and local execut result.
			// For the correct root, append it to the fullMerkleTree.
			if header.OutsMerkleRoot != emptyMerkleHash {
				if *merkleRoot != header.OutsMerkleRoot {
					panic(fmt.Sprintf("Merkle root in header is different from computed merkle root, "+
						"merkle root in header %v, computed merkle root %v, the message received is block %v is msgHeaderAndOuts %v",
						header.OutsMerkleRoot, *merkleRoot, shardIndex.GetID() == node.shardIndex.GetID(), !(shardIndex.GetID() == node.shardIndex.GetID())))
				}
				err := node.ledgerTree.Append(merkleRoot)
				if err != nil {
					panic(err)
				}
				err = node.ledgerTree.HookSecondLayerTree(fullMerkleTree)
				if err != nil {
					panic(err)
				}
			}
			if len(updateActions) > 0 {
				for _, action := range updateActions {
					originOut := action.OriginOut
					newOut := action.NewOut
					if err := node.ledgerTree.UpdateLeaf(merkle.ComputeMerkleHash(originOut.ToBytesArray()),
						merkle.ComputeMerkleHash(newOut.ToBytesArray())); err != nil {
						panic(err)
					}

					err = node.blockChain.ReceiveSmartContractShardInitOut(newOut)
					if err != nil {
						node.logger.Errorf("failed to update blockchain's shard init "+
							"out according to update action, err: %v", err)
					}
				}
			}
		}

		node.ledgerInfo.SetShardHeight(shardIndex, newHeight)
		// Saver persistent info to db for restart.
		if node.persistentInfo() != nil {
			node.logger.Errorf("Persistent info to db failed after update ledger: %v.")
		}
	}
}

// tryToSaveDeposit looks for unprocessed blocks based on records and attempts to
// add out from the block to the pool.
//
// Due to the special structure of slimblock, it is also necessary to take out and
// process deposit out of other blocks recorded in ledgerInfo.
func (node *storageNode) tryToSaveDeposit() {
	if !node.depositManager.shouldProcess() {
		return
	}

	ledgerHeight := wire.BlockHeight(node.state.GetShardHeight(node.shardIndex))
	blockHeight := node.depositManager.head()

	// To ensure that the proof can be obtained, the height of the block cannot
	// be higher than the height of the ledger.
	if blockHeight <= ledgerHeight {
		pastBlock := node.blockChain.GetSlimBlock(node.shardIndex, node.shardIndex, blockHeight)
		for _, shard := range pastBlock.LedgerInfo.ShardsHeights {
			outs := node.depositManager.removeOutsBeforeHeight(shard.Index, wire.BlockHeight(shard.Height))
			node.processDeposit(outs)
		}
		// This block has already been processed, so remove it from the queue
		node.depositManager.deQueue()
	}

}

func (node *storageNode) processDeposit(outs []*wire.OutState) {
	for _, out := range outs {
		// If deposit out, save it to depositpool
		if out.IsDepositOut() && out.Shard.GetID() == node.shardIndex.GetID() {
			outWithProof, err := node.getOutWithProof(&out.OutPoint)
			if err != nil {
				node.logger.Debugf("Can't get proof of out, to %v", out.UserAddress)
				continue
			}
			node.logger.Debugf("Add new deposit to pool, to %v", out.OutPoint.UserAddress)
			err = node.dPool.Add(outWithProof.Out, outWithProof.Height, outWithProof.Proof)
			if err != nil {
				node.logger.Errorf("Failed to add deposit proof to the pool: %v", outWithProof.Out)
			}
		}
	}
}

func (node *storageNode) updateStateBySlimblock(block *wire.SlimBlock) error {
	update, err := node.state.GetUpdateWithSlimBlock(block)
	if err != nil {
		node.logger.Debugf("Error update txpool with full block,"+
			" node shard %v, height %v, current ledger root %v, txpool root %v, inner err %v, block %v",
			node.shardIndex,
			node.shardHeight+1,
			err,
			block)
		return err
	}
	// Update the merkle path of all txs which are in txpool.
	node.txpool.RefreshPool(update)

	for _, action := range block.UpdateActions {
		err := node.ledgerTree.UpdateLeaf(merkle.ComputeMerkleHash(action.OriginOut.ToBytesArray()),
			merkle.ComputeMerkleHash(action.NewOut.ToBytesArray()))
		if err != nil {
			panic(err)
		}
	}

	// Update deposit's proof
	err = node.dPool.Update(update, node.shardIndex, wire.BlockHeight(block.Header.Height))
	if err != nil {
		return err
	}
	err = node.state.ApplyUpdate(update)
	return err
}

func (node *storageNode) persistentInfo() error {
	ledgerinfo := node.ledgerInfo
	//save ledgerinfo to db
	for _, shardIndex := range shard.ShardList {
		if err := node.db.Put(util.GetShardHeightKey(shardIndex.GetID()), util.Int64ToBytes(ledgerinfo.GetShardHeight(shardIndex))); err != nil {
			return err
		}
	}
	//save fullMekleTree info to db
	rootBytes, err := util.ValToByte(node.ledgerTree.MerkleRoot)
	if err != nil {
		return err
	}
	if err := node.db.Put([]byte(util.KeyLedgertreeRoot), rootBytes); err != nil {
		return err
	}
	nodeHashBytes, err := util.ValToByte(node.ledgerTree.GetLastNodeHash())
	if err != nil {
		return err
	}
	if err := node.db.Put([]byte(util.KeyLedgertreeLastnodehash), nodeHashBytes); err != nil {
		return err
	}
	if err := node.db.Put([]byte(util.KeyLedgertreeSize), util.Int64ToBytes(node.ledgerTree.Size)); err != nil {
		return err
	}
	//save storage's shardHeight to db
	if err := node.db.Put([]byte(util.KeyStorageShardheight), util.Int64ToBytes(node.shardHeight)); err != nil {
		return err
	}
	return nil
}

// loadGenesisBlock load the genesis block of the current shard.
func (node *storageNode) loadGenesisBlock() {

	genesisBlock := genesis.GenesisBlocks[node.shardIndex]
	outsMerkleTree := genesis.GenesisBlocksOutsMerkleTrees[node.shardIndex]

	// Build merkle tree by genesis block
	err := node.ledgerTree.Append(&genesisBlock.Header.OutsMerkleRoot)
	if err != nil {
		panic(err)
	}
	err = node.ledgerTree.HookSecondLayerTree(outsMerkleTree)
	if err != nil {
		panic(err)
	}

	node.ledgerInfo.SetShardHeight(node.shardIndex, 1)
	node.ledgerInfo.Size = 1
	node.shardHeight = 1

	// If there are deposit tx in genesis block, save them to pool
	node.processDeposit(genesisBlock.Body.Outs)
}

// initTxPool crate txpool and stateManager.
// TODO(zhaozheng): Remove the logic to create stateManager.
func (node *storageNode) initTxPool() {
	lastMerklePath, err := node.ledgerTree.GetLastNodeMerklePath()
	if err != nil {
		panic(err)
	}
	ledgerInfo := wire.LedgerInfo{}

	// Make a copy of ledger info before passing into transaction pool.
	for _, shard := range shard.ShardList {
		height := node.ledgerInfo.GetShardHeight(shard)
		ledgerInfo.SetShardHeight(shard, height)
	}

	// set state
	ledgerInfo.Size = node.ledgerInfo.Size
	node.txpool = txpool.NewTxPool()
	node.state = state.NewShardStateManager(node.ledgerTree.MerkleRoot, node.ledgerTree.Size, lastMerklePath, &ledgerInfo, node.blockChain)
}

func (node *storageNode) initFromDisk() {
	namespace := "treedata-shard" + shard.IndexToString(node.shardIndex)
	db, err := db.OpenDB(node.dataDir, namespace)
	if err != nil {
		panic(err)
	}
	node.db = db
	node.ledgerTree = merkle.NewOnDiskFullMerkleTree(db)
	node.loadLedgerFromDisk()
	if node.shardHeight < 1 {
		node.loadGenesisBlock()
	}
}

func (node *storageNode) loadLedgerFromDisk() {
	var shardsHeight = make([]*shard.IndexAndHeight, 0)
	var size int64
	rawData, err := node.db.Get([]byte(util.KeyStorageShardheight))
	if err != nil {
		node.logger.Errorf("Get storage's shardHeight from db err when initLedgerFromDisk: %v. If it's first start, can ingore this error info.", err)
		return
	}
	shardHeight := util.BytesToInt64(rawData)
	node.shardHeight = shardHeight
	for _, shardIndex := range shard.ShardList {
		rawData, err := node.db.Get(util.GetShardHeightKey(shardIndex.GetID()))
		if err != nil {
			node.logger.Errorf("Get ledgerInfo from db err when initLedgerFromDisk: %v. If it's first start, can ingore this error info.", err)
			return
		}
		height := util.BytesToInt64(rawData)
		size += height
		shardAndHeight := &shard.IndexAndHeight{Index: shardIndex, Height: height}
		shardsHeight = append(shardsHeight, shardAndHeight)
	}
	ledgerInfo := wire.LedgerInfo{ShardsHeights: shardsHeight, Size: size}
	node.ledgerInfo = ledgerInfo
}

// setSyncMgr sets the storage node's sync manager.
func (node *storageNode) setSyncMgr(mgr isync.SyncManager) {
	node.syncMgr = mgr
}

func (node *storageNode) String() string {
	return fmt.Sprintf("StorageNode {Shard: %v}", node.shardIndex.GetID())
}

// monitoringTxNumData 监控采集存储节点交易池中交易总数
func (node *storageNode) monitoringTxNum() {
	metric := metrics.Metric
	if node.txpool != nil {
		metric.PendingTxNum.With(
			prometheus.Labels{"shard_id": fmt.Sprint(node.shardIndex)},
		).Set(float64(node.txpool.NumberOfTransactions()))
	}
}

// monitoringShardHeight 监控采集存储节点分片高度
func (node *storageNode) monitoringShardHeight() {
	metric := metrics.Metric
	metric.StorageShardHeight.With(
		prometheus.Labels{"shard_id": fmt.Sprint(node.shardIndex)},
	).Set(float64(node.shardHeight))
}

// Act implements message.Actor interface.
//
// Act pass the given Event as parameter into the Callback function, and run the Callback function.
func (node *storageNode) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtMsgTx:
		msg := e.Extra.(*wire.MsgTx)
		node.onTransactionReceived(msg)
	case evtMsgBlock:
		msg := e.Extra.(*wire.MsgBlock)
		node.onBlockReceived(msg)
	case evtGetOutStateReq:
		msg := e.Extra.(*wire.OutPoint)
		outState, err := node.getOutState(msg)
		callback(&getOutStateResponse{outState, err})
	case evtGetAllShardHgts:
		callback(node.getShardHeights())
	case evtGetHeaderHashes:
		msg := e.Extra.(*reqHeaderHashes)
		callback(resHeaderHashes{headerGroups: node.getHeaderHashes(msg.locators)})
	case evtGetBlockReq:
		msg := e.Extra.(*getBlockReq)
		callback(node.getBlock(msg.shardIdx, msg.headerHash))
	case evtGetSlimBlockReq:
		msg := e.Extra.(*getBlockReq)
		callback(node.getSlimBlock(msg.shardIdx, msg.headerHash))
	case evtSlimBlock:
		msg := e.Extra.(*wire.SlimBlock)
		node.onSlimBlockReceived(msg)
	case evtGetSmartContract:
		msg := e.Extra.(multivacaddress.Address)
		callback(node.blockChain.GetSmartContract(msg))
	case evtGetSmartContractInfo:
		msg := e.Extra.([]interface{})
		contractAddr := msg[0].(multivacaddress.Address)
		shardIdx := msg[1].(shard.Index)
		callback(node.getSmartContractInfo(&wire.MsgFetchSmartContractInfo{
			MsgID:        0,
			ContractAddr: contractAddr,
			ShardIndex:   shardIdx,
		}))
	case evtMsgBlockConfirmation:
		msg := e.Extra.(*wire.MsgBlockConfirmation)
		node.onConfirmationReceived(msg)
	case evtFetchMsg:
		msg := e.Extra.(fetchRequest)
		node.fetcher.inject(newFetchAnnounce(msg.msg, msg.peerReply, node.onfetchMsg))
		<-node.fetcher.done
	case evtGetLastConfirmation:
		msg := e.Extra.(shard.Index)
		callback(node.getLastConfirmation(msg))
	default:
		node.logger.Debug("%v received unknown mail: %v", node, e.Topic)
	}
}
