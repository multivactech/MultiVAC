/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"errors"
	"fmt"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/smartcontractdatastore"
	"github.com/multivactech/MultiVAC/processor/shared/state"
	"github.com/multivactech/MultiVAC/processor/shared/txpool"
)

// onNewMessage Handles new messages from peer. Currently it handles new transactions and new blocks.
func (abc *appBlockChain) onNewMessage(msg wire.Message) {
	switch msg := msg.(type) {
	case *wire.BlockHeader:
		abc.log.Debugf("block header: shard: %d, height: %d", msg.ShardIndex, msg.Height)
		err := abc.onBlockHeaderReceived(msg)
		if err != nil {
			abc.log.Errorf("failed to handle block header,%v", err)
		}
	case *wire.MsgReturnInit, *wire.MsgReturnTxs, *wire.SmartContractInfo:
		abc.fetcher.onMessage(msg)
		<-abc.fetcher.done
	}
}

// handleInit initializes abci with some ledger data which come from storage node.
func (abc *appBlockChain) handleInit(msg wire.Message) error {
	initData := msg.(*wire.MsgReturnInit)
	if initData.ShardHeight <= abc.shardHeight {
		return fmt.Errorf("Receive the ladger is too old, current height: %d, ledger height: %d",
			abc.shardHeight, initData.ShardHeight)
	}

	// State represents the current ledger state
	// When the init data from storagenode is behind the current ledger data, ignore
	if (abc.shardHeight > 1 && initData.ShardHeight <= abc.shardHeight) || initData.ShardHeight < abc.prevReshardHeader.Height {
		go abc.fetchNewLedger()
		return fmt.Errorf("Shard(%v) rejecting init data due to behind the current ledger data, init data height: %v, reshardHeight: %v",
			abc.shardIndex.GetID(), initData.ShardHeight, abc.prevReshardHeader.Height)
	}

	treeRoot := &initData.LatestHeader.ShardLedgerMerkleRoot
	treeSize := initData.TreeSize
	rightPath := &initData.RightPath
	ledger := &initData.Ledger
	// Verify if the right-path is corresponding to root
	err := merkle.VerifyLastNodeMerklePath(treeRoot, treeSize, rightPath)
	if err != nil {
		abc.log.Debugf("rejecting init data due to wrong last node path, err %v", err)
		return err
	}
	// Initialize the txpool
	txPool := txpool.NewTxPool()
	smartContractDataStore := smartcontractdatastore.NewSmartContractDataStore(abc.shardIndex)
	state := state.NewShardStateManager(
		treeRoot,
		treeSize,
		rightPath,
		ledger,
		abc.blockChain)

	abc.state = state
	abc.txPool = txPool
	abc.smartContractDataStore = smartContractDataStore
	abc.shardHeight = initData.ShardHeight
	abc.proposeBlockHgt = abc.shardHeight + 1
	abc.prevHeader = initData.LatestHeader

	if abc.prevReshardHeader.Height == 0 {
		abc.prevReshardHeader = initData.LatestHeader
	}

	// Cache the receving header to ledger
	abc.blockChain.ReceiveHeader(&initData.LatestHeader)
	// Trigger event and nofity listeners.
	abc.pubSubMgr.Pub(*message.NewEvent(message.EvtTxPoolInitFinished, abc.shardHeight))

	for _, d := range initData.Deposits {
		err := abc.dPool.Add(d.Out, d.Height, d.Proof)
		if err != nil {
			abc.log.Errorf("failed to add deposit,%v", err)
		}
	}

	// Try to fetch tx
	if !abc.isFetchingTx {
		go abc.fetchTxWorker()
	}

	abc.log.Debugf("txPool init done, the current height of abci is %d\n", abc.shardHeight)
	return nil
}

// onSmartContractInfo handles the smartContractInfo sent from the storage node after requesting smartContractInfo
func (abc *appBlockChain) onSmartContractInfo(msg wire.Message) error {
	smartContractInfo := msg.(*wire.SmartContractInfo)
	ledgerRoot := abc.state.GetLedgerRoot()
	pKHash, err := smartContractInfo.SmartContract.ContractAddr.GetPublicKeyHash(multivacaddress.SmartContractAddress)
	if err != nil {
		abc.log.Debugf("fail to GetPublicKeyHash, err: %v", err)
	}
	if sCInfo := abc.smartContractDataStore.GetSmartContractInfo(*pKHash); sCInfo != nil {
		return nil
	}
	if err := abc.smartContractDataStore.AddSmartContractInfo(smartContractInfo, &ledgerRoot); err != nil {
		abc.log.Debugf("fail to AddSmartContractInfo, err: %v", err)
	}
	return nil
}

func (abc *appBlockChain) handleMsgReturnTxs(msg wire.Message) error {
	returnTxs := msg.(*wire.MsgReturnTxs)
	ledgerRoot := abc.state.GetLedgerRoot()

	// adding a new pending transaction to the transaction pool.
	// the transaction will be included in a later proposed block.
	for _, txp := range returnTxs.Txs {
		if _, err := abc.txPool.AddNewTransaction(txp, &ledgerRoot); err != nil {
			abc.log.Errorf("Handling adding transactions, %s", err.Error())
			return err
		}
	}

	// 将SmartContractInfo存储在smartContractDataStore的pool当中
	for _, sci := range returnTxs.SmartContractInfos {
		if err := abc.smartContractDataStore.AddSmartContractInfo(sci, &ledgerRoot); err != nil {
			return err
		}
	}

	abc.monitoringTxNum()

	return nil
}

// onBlockHeaderReceived receive headers sent from other nodes
func (abc *appBlockChain) onBlockHeaderReceived(header *wire.BlockHeader) error {
	if ok := abc.blockChain.ReceiveHeader(header); !ok {
		abc.log.Debugf("Blockchain can't receive header")
		return errors.New("abc.onBlockHeaderReceived: Blockchain can't receive header")
	}
	shardIdx := abc.shardIndex
	isFetched := make(map[multivacaddress.PublicKeyHash]bool)

	// Request smart contract info from the storage node based on reduceAction in the header
	for _, action := range header.ReduceActions {
		if action.ToBeReducedOut.Shard != abc.shardIndex {
			continue
		}
		addr := action.ToBeReducedOut.ContractAddress
		pKHash, err := addr.GetPublicKeyHash(multivacaddress.SmartContractAddress)
		if err != nil {
			abc.log.Debugf("fail to GetPublicKeyHash, err: %v", err)
			continue
		}

		if !isFetched[*pKHash] {
			isFetched[*pKHash] = true
			if sCInfo := abc.smartContractDataStore.GetSmartContractInfo(*pKHash); sCInfo != nil {
				continue
			}
			abc.fetchSmartContract(addr, shardIdx)
		}

	}
	return nil
}
