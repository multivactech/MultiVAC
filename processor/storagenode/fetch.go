package storagenode

import (
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// getTxWithProofs returns the merkle path of specified tx.
func (node *storageNode) getTxWithProofs(tx *wire.MsgTx) (*wire.MsgTxWithProofs, error) {
	txWithProof := wire.MsgTxWithProofs{
		Tx:     *tx,
		Proofs: []merkle.MerklePath{},
	}
	for _, txIn := range tx.TxIn {
		outState := txIn.PreviousOutPoint.ToUnspentOutState()
		merkleHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
		merklePath, err := node.ledgerTree.GetMerklePath(merkleHash)
		if err != nil {
			return nil, fmt.Errorf("fail to find proof, shard: %v, address: %v, out: %v", txIn.PreviousOutPoint.Index, txIn.PreviousOutPoint.UserAddress, txIn.PreviousOutPoint)
		}
		txWithProof.Proofs = append(txWithProof.Proofs, *merklePath)
	}
	return &txWithProof, nil
}

func (node *storageNode) getOutWithProof(outPoint *wire.OutPoint) (*wire.OutWithProof, error) {
	outState := outPoint.ToUnspentOutState()
	merkleHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
	merklePath, err := node.ledgerTree.GetMerklePath(merkleHash)
	if err != nil {
		return nil, errors.New("Fail to find proof")
	}
	owp := &wire.OutWithProof{
		Out:    outPoint,
		Proof:  merklePath,
		Height: wire.BlockHeight(node.shardHeight),
	}
	return owp, nil
}

// getOutState returns the corresponding outstate according to the specified outpoint.
func (node *storageNode) getOutState(outPoint *wire.OutPoint) (*wire.OutState, error) {
	unspentOutState := outPoint.ToUnspentOutState()
	if _, err := node.ledgerTree.GetMerklePath(
		merkle.ComputeMerkleHash(unspentOutState.ToBytesArray())); err == nil {
		node.logger.Infof("unspent out found: %v", outPoint)
		return unspentOutState, nil
	}
	spentOutState := outPoint.ToSpentOutState()
	if _, err := node.ledgerTree.GetMerklePath(
		merkle.ComputeMerkleHash(spentOutState.ToBytesArray())); err == nil {
		node.logger.Infof("spent out found: %v", outPoint)
		return spentOutState, nil
	}
	return nil, errors.New("GetOutState: not found")
}

// getBlockInfo returns all blocks after the specified height.
func (node *storageNode) getBlockInfo(startHeight int64) []*wire.MsgBlock {
	blockList := []*wire.MsgBlock{}
	hgt := wire.BlockHeight(startHeight)
	for {
		if block := node.blockChain.GetShardsBlockByHeight(node.shardIndex, hgt); block != nil {
			blockList = append(blockList, block)
			hgt++
		} else {
			break
		}
	}
	return blockList
}

// getSlimBlockInfo returns all SlimBlocks after the specified height in the specified shard.
func (node *storageNode) getSlimBlockInfo(toShard shard.Index, startHeight int64) []*wire.SlimBlock {
	blockList := []*wire.SlimBlock{}
	hgt := wire.BlockHeight(startHeight)
	for {
		if slimblock := node.blockChain.GetSlimBlock(toShard, node.shardIndex, hgt); slimblock != nil {
			blockList = append(blockList, slimblock)
			hgt++
		} else {
			break
		}
	}
	return blockList
}

// getBlockSliceInfoSince returns all blocks after the specified height.
func (node *storageNode) getBlockSliceInfoSince(startHeight int64) []*wire.MsgBlock {
	blockList := []*wire.MsgBlock{}
	hgt := wire.BlockHeight(startHeight)
	for {
		if block := node.blockChain.GetShardsBlockByHeight(node.shardIndex, hgt); block != nil {
			blockList = append(blockList, block)
			hgt++
		} else {
			break
		}
	}
	return blockList
}

// getBlockSliceInfoBetween returns all blocks between the specified heights.
func (node *storageNode) getBlockSliceInfoBetween(startHeight, endHeight int64) []*wire.MsgBlock {
	blockList := []*wire.MsgBlock{}
	for hgt := startHeight; hgt < endHeight; hgt++ {
		if block := node.blockChain.GetShardsBlockByHeight(node.shardIndex, wire.BlockHeight(hgt)); block != nil {
			blockList = append(blockList, block)
		} else {
			break
		}
	}
	return blockList
}

// getSlimBlockSliceInfoSince returns all slimblocks after the specified height.
func (node *storageNode) getSlimBlockSliceInfoSince(to shard.Index, startHeight int64) []*wire.SlimBlock {
	blockList := []*wire.SlimBlock{}
	hgt := wire.BlockHeight(startHeight)
	for {
		if block := node.blockChain.GetSlimBlock(to, node.shardIndex, hgt); block != nil {
			blockList = append(blockList, block)
			hgt++
		} else {
			break
		}
	}
	return blockList
}

// getSlimBlockSliceInfoBetween returns all slimblocks between the specified heights.
func (node *storageNode) getSlimBlockSliceInfoBetween(to shard.Index, startHeight, endHeight int64) []*wire.SlimBlock {
	blockList := []*wire.SlimBlock{}
	for hgt := startHeight; hgt < endHeight; hgt++ {
		if block := node.blockChain.GetSlimBlock(to, node.shardIndex, wire.BlockHeight(hgt)); block != nil {
			blockList = append(blockList, block)
		} else {
			break
		}
	}
	return blockList
}

// ShardHeight returns the current shard height.
func (node *storageNode) ShardHeight() int64 {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	return node.shardHeight
}

// LedgerInfo returns the ledgerInfo.
func (node *storageNode) LedgerInfo() *wire.LedgerInfo {
	return &node.ledgerInfo
}

// getShardHeights is a internal method, it retunrs the height of all shards.
func (node *storageNode) getShardHeights() []shard.IndexAndHeight {
	l := len(shard.ShardList)
	heights := make([]shard.IndexAndHeight, 0, l)
	for _, id := range shard.ShardList {
		idxAndHgt := shard.IndexAndHeight{Index: id, Height: int64(node.blockChain.GetShardsHeight(id))}
		heights = append(heights, idxAndHgt)
	}
	return heights
}

// getHeaderHashes servers for synchronization, it returns the block header of the specified range.
func (node *storageNode) getHeaderHashes(locators []*wire.BlockLocator) []*wire.InvGroup {
	groups := make([]*wire.InvGroup, 0, len(locators))
	for _, locator := range locators {
		id := locator.ShardIdx
		hashes := node.blockChain.GetShardsHeaderHashes(id,
			wire.BlockHeight(locator.FromHeight),
			wire.BlockHeight(locator.ToHeight))
		groups = append(groups, wire.NewInvGroup(id, hashes...))
	}
	return groups
}

// getBlock serves for rpc request, it returns block of specified shard and hash.
func (node *storageNode) getBlock(s shard.Index, headerHash chainhash.Hash) *wire.MsgBlock {
	c := node.blockChain
	return c.GetShardsBlockByHash(headerHash)
}

// getSlimBlock serves for rpc request, it returns slim block.
func (node *storageNode) getSlimBlock(s shard.Index, headerHash chainhash.Hash) *wire.SlimBlock {
	c := node.blockChain
	header := c.GetShardsHeaderByHash(headerHash)
	if header == nil {
		node.logger.Errorf("can't find Header: %v", node.shardIndex.GetID(), headerHash)
		return nil
	}
	rtn := c.GetSlimBlock(node.shardIndex, s, wire.BlockHeight(header.Height))
	if rtn == nil {
		node.logger.Errorf("can't find slim block(shard: %d,hgt: %d,headerHash: %v) ",
			s, header.Height, headerHash)
	}
	return rtn
}

func (node *storageNode) getSmartContractInfo(msg *wire.MsgFetchSmartContractInfo) *wire.SmartContractInfo {

	c := node.blockChain
	sc := c.GetSmartContract(msg.ContractAddr)
	if sc == nil {
		node.logger.Errorf("GetSmartContract失败")
		return nil
	}

	codeOut := c.GetSmartContractCodeOut(msg.ContractAddr, msg.ShardIndex)
	if codeOut == nil {
		node.logger.Errorf("GetSmartContractCodeOut失败")
		return nil
	}
	codeOutHash := merkle.ComputeMerkleHash(codeOut.ToBytesArray())
	codeOutProof, err := node.ledgerTree.GetMerklePath(codeOutHash)
	if err != nil {
		node.logger.Errorf("failed to getCodeOutProof, err: %v", err)
		return nil
	}

	shardInitOut := c.GetSmartContractShardInitOut(msg.ContractAddr, msg.ShardIndex)
	if shardInitOut == nil {
		node.logger.Errorf("failed to getSmartContractShardInitOut")
		return nil
	}

	outHash := merkle.ComputeMerkleHash(shardInitOut.ToBytesArray())
	shardInitOutProof, err := node.ledgerTree.GetMerklePath(outHash)
	if err != nil {
		node.logger.Errorf("failed to getShardInitOutProof, err: %v", err)
		return nil
	}

	return &wire.SmartContractInfo{
		MsgID:    msg.MsgID,
		ShardIdx: msg.ShardIndex,
		SmartContract: &wire.SmartContract{
			ContractAddr: msg.ContractAddr,
			APIList:      sc.APIList,
			Code:         sc.Code,
		},

		CodeOut: codeOut, CodeOutProof: codeOutProof,
		ShardInitOut: shardInitOut, ShardInitOutProof: shardInitOutProof}
}

func (node *storageNode) getLastConfirmation(shardIdx shard.Index) *wire.MsgBlockConfirmation {
	return node.lastConfirmation[shardIdx]
}

// onFetchInitData is triggered by the miner node, for those who are behind,
// it will return the status information of the current ledger.
func (node *storageNode) onFetchInitData(msg *wire.MsgFetchInit) *wire.MsgReturnInit {
	ledger := node.LedgerInfo()
	shardHeight := node.ShardHeight()
	tree := node.ledgerTree

	treePath, err := tree.GetLastNodeMerklePath()
	treeSize := tree.Size
	if err != nil {
		node.logger.Errorf("OnRequestingMerklePath: error %s\n", err.Error())
		return nil
	}

	latestHeader := node.blockChain.GetShardsHeaderByHeight(node.shardIndex, wire.BlockHeight(shardHeight))
	result := &wire.MsgReturnInit{
		ShardIndex:   node.shardIndex,
		MsgID:        msg.MsgID,
		Ledger:       *ledger,
		RightPath:    *treePath,
		ShardHeight:  shardHeight,
		LatestHeader: *latestHeader,
		TreeSize:     treeSize,
	}

	// Storagenode should verify the lastnode path first, if error it means the ledger of storagenode is wrong
	err = merkle.VerifyLastNodeMerklePath(&result.LatestHeader.ShardLedgerMerkleRoot, result.TreeSize, &result.RightPath)
	if err != nil {
		panic(node.shardIndex)
	}

	outWithProofs, err := node.dPool.GetAll(node.shardIndex, msg.Address)
	if err != nil {
		node.logger.Debugf("Address has no deposit: %d", msg.Address)
	} else {
		result.Deposits = outWithProofs
	}

	node.logger.Debugf("Length of %v is %d", msg.Address, len(outWithProofs))
	return result
}

// pendingTransactionNumber returns the number of transactions waiting to be packaged in the current trading pool.
func (node *storageNode) pendingTransactionNumber() int {
	if node.txpool != nil {
		return node.txpool.NumberOfTransactions()
	}
	return 0
}

// onFetchTxs returns number transactions that do not exist in the hashlist.
func (node *storageNode) onRequestingTransactions(msg *wire.MsgFetchTxs) (*wire.MsgReturnTxs, error) {
	update, err := node.state.NewUpdate()
	if err != nil {
		panic(err)
	}
	txsWithProof := node.txpool.GetTopTransactions(msg.NumberOfTxs+len(msg.ExistTx), update, node.shardHeight)
	txMap := make(map[uint32]bool)
	for _, hash := range msg.ExistTx {
		txMap[hash] = true
	}
	// Filter out existing tx that recorded in hashList
	fnvInst := fnv.New32()
	txs := make([]*wire.MsgTxWithProofs, 0)
	for _, twp := range txsWithProof {
		_, _ = fnvInst.Write(twp.Tx.SerializeWithoutSignatureAndPubKey())
		txHash := fnvInst.Sum32()
		if _, ok := txMap[txHash]; !ok {
			txs = append(txs, twp)
		}
		fnvInst.Reset()
	}
	if len(txs) > 0 {
		var sCInfos []*wire.SmartContractInfo
		var rtnTxs []*wire.MsgTxWithProofs
		for _, tx := range txs {
			// When dealing with system contract, skip it.
			if tx.Tx.IsSmartContractTx() {
				sCInfo := node.getSmartContractInfo(&wire.MsgFetchSmartContractInfo{ContractAddr: tx.Tx.ContractAddress, ShardIndex: tx.Tx.Shard})
				if sCInfo == nil {
					continue
				}
				sCInfos = append(sCInfos, sCInfo)
			}
			rtnTxs = append(rtnTxs, tx)
		}
		return &wire.MsgReturnTxs{
			MsgID:              msg.MsgID,
			ShardIndex:         msg.ShardIndex,
			Txs:                rtnTxs,
			SmartContractInfos: sCInfos,
		}, nil
	}
	return nil, errors.New("no txs return")
}

func (node *storageNode) onfetchMsg(msg wire.Message) {
	switch msg := msg.(type) {
	case *wire.MsgFetchInit:
		result := node.onFetchInitData(msg)
		node.fetcher.onResult(newResultAnnounce(result))
	case *wire.MsgFetchTxs:
		result, err := node.onRequestingTransactions(msg)
		if err != nil {
			node.fetcher.onResult(newAnnounce(getAnnounceIDByMsgID(msg.MsgID), nil, getResendRoundByMsgID(msg.MsgID), nil, nil))
			return
		}
		node.fetcher.onResult(newResultAnnounce(result))
	case *wire.MsgFetchSmartContractInfo:
		result := node.getSmartContractInfo(msg)
		node.fetcher.onResult(newResultAnnounce(result))
	}
}
