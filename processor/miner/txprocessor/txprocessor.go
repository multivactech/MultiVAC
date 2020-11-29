/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txprocessor

import (
	"bytes"
	"sort"

	"github.com/multivactech/MultiVAC/interface/itxprocessor"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/smartcontractdatastore"
)

// NewTxProcessor returns a new instance of TxProcessor.
func NewTxProcessor(shard shard.Index, sCDS smartcontractdatastore.SmartContractDataStore) itxprocessor.TxProcessor {
	var txs []*wire.MsgTx
	return &txProcessor{
		shard:                  shard,
		txs:                    txs,
		smartContractDataStore: sCDS,
	}
}

type txProcessor struct {
	shard                  shard.Index
	txs                    []*wire.MsgTx
	smartContractDataStore smartcontractdatastore.SmartContractDataStore
}

type result struct {
	tree           *merkle.FullMerkleTree
	successfulTxs  []*wire.MsgTx
	outStates      []*wire.OutState
	smartContracts []*wire.SmartContract
	updateActions  []*wire.UpdateAction
	reduceActions  []*wire.ReduceAction
}

// AddTx adds a tx to the txs.
func (tp *txProcessor) AddTx(tx *wire.MsgTx) {
	tp.txs = append(tp.txs, tx)
}

// Execute transactions given the blockHeight.
// Returns a new result that contains new sorted outStates, merkleTree...
func (tp *txProcessor) Execute(proposeBlockHeight int64) itxprocessor.Result {
	if len(tp.txs) == 0 {
		return &result{}
	}

	// Space for updateActions must be created ahead of time because DeepEqual(emptyArray, nil) == false
	r := &result{
		updateActions: make([]*wire.UpdateAction, 0),
	}
	sysCE := NewSysCE(tp.shard)

	var outHashes []*merkle.MerkleHash
	var outStates []*wire.OutState
	var scs []*wire.SmartContract
	exeSmCMap := make(map[multivacaddress.PublicKeyHash][]*wire.MsgTx) // 存储执行智能合约的交易

	for _, tx := range tp.txs {
		if tx.IsSmartContractTx() {
			pkHash, err := tx.ContractAddress.GetPublicKeyHash(multivacaddress.SmartContractAddress)
			if err != nil {
				// Treat the tx as a failure
				continue
			}
			smartCTxs := exeSmCMap[*pkHash]
			smartCTxs = append(smartCTxs, tx)
			exeSmCMap[*pkHash] = smartCTxs
			continue
		}

		ops, sc, err := sysCE.Execute(tx, proposeBlockHeight)
		if err == nil {
			r.successfulTxs = append(r.successfulTxs, tx)
			if sc != nil {
				scs = append(scs, sc)
			}
			for _, op := range ops {
				outStates = append(outStates, op.ToUnspentOutState())
			}
		} else {
			log.Errorf("failed to execute tx %v, err msg: %v", tx.API, err)
		}
	}

	smartCDataStore := tp.smartContractDataStore
	for pKHash, txs := range exeSmCMap {
		sCAddr := multivacaddress.GenerateAddressByPublicKeyHash(pKHash, multivacaddress.SmartContractAddress)
		smartCE := newSmartCE(tp.shard, sCAddr)

		sCInfo := smartCDataStore.GetSmartContractInfo(pKHash)
		sCResult, err := smartCE.Execute(txs, sCInfo)
		if err != nil {
			log.Errorf("fail to execute smart contract, contract address: %v, shard: %v, err: %v",
				sCAddr, tp.shard, err)
			continue
		}
		r.successfulTxs = append(r.successfulTxs, sCResult.successfulTxs...)
		r.updateActions = append(r.updateActions, sCResult.updateShardInitOut)
		r.reduceActions = append(r.reduceActions, sCResult.reduceActions...)
		outStates = append(outStates, sCResult.newOuts...)
	}

	sort.SliceStable(scs, func(i, j int) bool {
		return bytes.Compare(scs[i].ContractAddr, scs[j].ContractAddr) < 0
	})

	r.smartContracts = scs

	// Sort out states by shard, tx hash, and index.
	sort.SliceStable(outStates, func(i, j int) bool {
		if outStates[i].OutPoint.Shard != outStates[j].OutPoint.Shard {
			return outStates[i].OutPoint.Shard < outStates[j].OutPoint.Shard
		} else if outStates[i].OutPoint.TxHash != outStates[j].OutPoint.TxHash {
			return bytes.Compare(outStates[i].OutPoint.TxHash[:], outStates[j].OutPoint.TxHash[:]) < 0
		} else {
			return outStates[i].OutPoint.Index < outStates[j].OutPoint.Index
		}
	})

	for _, outState := range outStates {
		outStateHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
		outHashes = append(outHashes, outStateHash)
	}
	r.tree = merkle.NewFullMerkleTree(outHashes...)
	r.outStates = outStates
	return r
}

// GetOutsMerkleTree returns the tree of result.
func (r *result) GetOutsMerkleTree() *merkle.FullMerkleTree {
	return r.tree
}

// GetMerkleRoot returns the root of the tree of the result.
func (r *result) GetMerkleRoot() *merkle.MerkleHash {
	if r.tree == nil || r.tree.MerkleRoot == nil {
		return &merkle.EmptyHash
	}
	return r.tree.MerkleRoot

}

// GetSuccessfulTxs returns the successfulTxs of the result.
func (r *result) GetSuccessfulTxs() []*wire.MsgTx {
	return r.successfulTxs
}

// GetOutStates returns outStates of the result.
func (r *result) GetOutStates() []*wire.OutState {
	return r.outStates
}

// GetSmartContracts returns smartContracts of the result.
func (r *result) GetSmartContracts() []*wire.SmartContract {
	return r.smartContracts
}

// GetUpdateActions returns updateActions of the result.
func (r *result) GetUpdateActions() []*wire.UpdateAction {
	return r.updateActions
}

// GetReduceActions returns reduceActions of the result.
func (r *result) GetReduceActions() []*wire.ReduceAction {
	return r.reduceActions
}
