package storagenode

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/wire"
)

// onConfirmationReceived receives MsgBlockConfirmation and caches this block's message.
func (node *storageNode) onConfirmationReceived(msg *wire.MsgBlockConfirmation) {
	node.logger.Debugf("receive block confirmation message %v", msg)
	_, ok := node.confirmedHeaders[msg.Header.BlockHeaderHash()]
	if ok {
		node.logger.Warnf("storageNode has received duplicate block confirmation message")
		return
	}

	if err := msg.IsValidated(); err != nil {
		node.logger.Warnf("%v", err)
		return
	}
	node.onHandleConfirmation(msg)
}

// onHandleConfirmation receive slimBlocks in cache
func (node *storageNode) onHandleConfirmation(msg *wire.MsgBlockConfirmation) {
	msgShard := msg.Header.ShardIndex
	node.confirmedHeaders[msg.Header.BlockHeaderHash()] = true

	if node.lastConfirmation[msgShard] == nil ||
		node.lastConfirmation[msgShard].Header.Height < msg.Header.Height {
		node.lastConfirmation[msgShard] = msg
	}

	node.logger.Infof("验证缓存的没有收到confirmation(shard: %d,hgt: %d)的slimBlock",
		msg.Header.ShardIndex, msg.Header.Height)

	// 验证缓存的没有收到confirmation信息的slimBlock
	node.checkClearReceivedBlocks(msg.Header.PrevBlockHeader)

	slimblock, ok := node.receivedBlocks[msg.Header.BlockHeaderHash()]
	if ok {
		node.onSlimBlockReceived(slimblock)
		return
	}
}

// To implement faster, store all outs in all blocks for now.
// TODO(imzeadom): Only get part of outs that belong to the shard this node is working for.
func (node *storageNode) onBlockReceived(block *wire.MsgBlock) {
	// TODO(imzeadom): Verify this block is a legal block first. And ignore the illegal blocks.
	node.logger.Infof("Shard %d storage node received block. height: %v, shard: %d, current height: %d,  Hash: %v",
		node.shardIndex.GetID(), block.Header.Height,
		block.Header.ShardIndex.GetID(), node.shardHeight, block.Header.BlockHeaderHash())

	node.blockChain.ReceiveBlock(block)
	slimBlock, err := wire.GetSlimBlockFromBlockByShard(block, node.shardIndex)
	if err != nil {
		panic(err)
		//node.logger.Error(err)
	}
	node.onSlimBlockReceived(slimBlock)
}

// onSlimBlockReceived cache the received SlimBlock
func (node *storageNode) onSlimBlockReceived(msg *wire.SlimBlock) {
	node.logger.Infof("Shard %d storage node 高度：%v", node.shardIndex, node.shardHeight)
	node.logger.Infof("Shard %d storage node received slimBlock(shard: %d, height: %d)"+
		" shouldToShard: %d HeaderHash: %v\n", node.shardIndex, msg.Header.ShardIndex, msg.Header.Height, msg.ToShard, msg.Header.BlockHeaderHash())
	if node.shardIndex != msg.ToShard {
		node.logger.Errorf("shard %v storage node received wrong slimBlock shouldToShard : %v\n", node.shardIndex, msg.ToShard)
		return
	}
	if node.shardIndex == msg.Header.ShardIndex && node.shardHeight == 1 {
		// 监控采集存储节点数据，如果不加本行代码，对于新启动请求同步的存储节点的监控数据将无法正常采集
		node.monitoringShardHeight()
		node.monitoringTxNum()
	}
	if node.shardIndex == msg.Header.ShardIndex {
		_, ok := node.confirmedHeaders[msg.Header.BlockHeaderHash()]
		if !ok {
			node.logger.Warnf("this slimBlock(shard: %d,Hgt: %d) has not been confirmed",
				msg.Header.ShardIndex, msg.Header.Height)
			node.receivedBlocks[msg.Header.BlockHeaderHash()] = msg
			return
		}
		delete(node.confirmedHeaders, msg.Header.BlockHeaderHash())
		delete(node.receivedBlocks, msg.Header.BlockHeaderHash())
		node.checkClearReceivedBlocks(msg.Header.PrevBlockHeader)
	}

	header := msg.Header
	if !header.IsEmptyBlock {
		clipMsg := msg.ClipTreeData
		_, err := clipMsg.Rebuild(header.OutsMerkleRoot, msg.ToShard)
		if err != nil {
			node.logger.Errorf("clip tree rebuild error : %v\n", err)
			return
		}

		// Find all deposit outs belong to my shard, cache them.
		for _, out := range clipMsg.Outs {
			if out.IsDepositOut() && out.Shard.GetID() == node.shardIndex.GetID() {
				node.depositManager.add(out, wire.BlockHeight(header.Height))
			}
		}
	}
	if header.IsEmptyBlock {
		node.logger.Debugf("Warning: receive an empty block height is %d, shard is %v", header.Height, header.ShardIndex)
	}
	node.blockChain.ReceiveSlimBlock(msg)

	// update the ledger only when storage node receives its own slimblock.
	if msg.Header.ShardIndex != node.shardIndex {
		return
	}

	// try to update ledgerTree
	updated := false
	for node.canUpdateLedger() {
		updated = true
		node.db.StartTransactionRecord()
		node.maybeUpdateLedger()
		node.db.CommitTransaction()
	}

	// Notify listener that this storage node has been sequentially fail to update ledger.
	if !updated {
		node.updateFailCnt++
		if node.updateFailCnt >= updateLedgerReportThreshold {
			node.logger.Warnf("Shard %v sequential ledger update failure, notify listeners.", node.shardIndex.GetID())
			node.updateFailCnt = 0
			if node.syncMgr != nil {
				node.logger.Infof("storageNode 开始准备同步")
				node.syncMgr.MaybeSync()
			}
		}
	} else {
		// 监控采集存储节点数据
		node.monitoringShardHeight()
		node.monitoringTxNum()
	}
}

// onTransactionReceived receives transactions from client by rpc request.
func (node *storageNode) onTransactionReceived(tx *wire.MsgTx) {
	node.logger.Debugf("New tx: %v, tx: %v", tx.TxHash().String(), tx)
	// Storagenode in this shard can only process tx belongs to this shard.
	if !tx.IsTxInShard(node.shardIndex) {
		node.logger.Warnf("Received a tx for other shard.")
		return
	}

	// Verify there's no outs used more than once in this transaction.
	outsMap := make(map[struct {
		chainhash.Hash
		int
	}]bool)
	for _, txIn := range tx.TxIn {
		key := struct {
			chainhash.Hash
			int
		}{
			txIn.PreviousOutPoint.TxHash,
			txIn.PreviousOutPoint.Index,
		}
		_, ok := outsMap[key]
		if ok {
			node.logger.Warn("OnTransactionReceived: Transaction has duplicate txin")
			return
		}
		outsMap[key] = true
	}

	// Generate merkle path for the tx.
	txWithProof, err := node.getTxWithProofs(tx)
	if err != nil {
		node.logger.Warnf("OnTransactionReceived: error %s\n", err.Error())
		// Ignore this transaction if failed to get merkle path for this TxIn.
		return
	}

	// Add the correct tx to pool.
	stateRoot := node.state.GetLedgerRoot()
	_, err = node.txpool.AddNewTransaction(txWithProof, &stateRoot)
	if err != nil {
		// The txWithProof to insert should be correct. No error expected. So panic if any error happens.
		node.logger.Error(err)
		return
	}
	node.logger.Debugf("OnTransactionReceived: Add new transaction to txpool, tx: %v", tx.TxHash().String())
	node.monitoringTxNum()
}
