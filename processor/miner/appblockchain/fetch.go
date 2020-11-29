/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"time"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
)

func (abc *appBlockChain) fetchTxWorker() {
	abc.txFetchTicker = time.NewTicker(fetchTransactionsInterval)
	abc.isFetchingTx = true
	for range abc.txFetchTicker.C {
		abc.fetchTransactions()
	}
}

func (abc *appBlockChain) fetchNewLedger() {
	abc.fetcher.inject(newFetchInitDataAnnounce(abc.shardIndex, abc.shardHeight, abc.getAddress(), abc.handleInit, abc.broadcastMsg))
}

func (abc *appBlockChain) fetchTxs(num int, hashList []uint32) {
	abc.fetcher.inject(newFetchTxAnnounce(num, hashList, abc.shardIndex, abc.getAddress(), abc.handleMsgReturnTxs, abc.broadcastMsg))
}

func (abc *appBlockChain) fetchSmartContract(contractAddr multivacaddress.Address, shardIdx shard.Index) {
	abc.fetcher.inject(newFetchSmartContractInfoAnnounce(contractAddr, shardIdx, abc.getAddress(), abc.onSmartContractInfo, abc.broadcastMsg))
}

// fetchTransactions fetch transactions from storage node
func (abc *appBlockChain) fetchTransactions() {
	// If number of txs is smaller than min value, then request new txs
	numberOfTxs := abc.txPool.NumberOfTransactions()
	// todo(MH): remove temporarily, avoiding Concurrency Issues
	//update, err := abc.state.NewUpdate()
	//if err != nil {
	//	panic(err)
	//}
	if numberOfTxs < minTransactionsInTxPool {
		// Get all tx from txpool then calculate their hash
		//hashList := make([]uint32, numberOfTxs)
		//fnvInst := fnv.New32()
		//txWithProof := abc.txPool.GetTopTransactions(numberOfTxs, update, abc.shardHeight)
		//for i, twp := range txWithProof {
		//	_, err := fnvInst.Write(twp.Tx.SerializeWithoutSignatureAndPubKey())
		//	if err != nil {
		//		abc.log.Errorf("failed to write tx, %v", err)
		//	}
		//	hashList[i] = fnvInst.Sum32()
		//	fnvInst.Reset()
		//}
		abc.fetchTxs(maxTransactionsInTxPool-numberOfTxs, nil)
	}
}
