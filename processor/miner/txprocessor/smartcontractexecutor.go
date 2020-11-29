/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txprocessor

import (
	"fmt"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/mvvm"
)

type smartContractExecutor struct {
	shardIdx        shard.Index
	contractAddress multivacaddress.Address
}

type smartContractExeResult struct {
	successfulTxs      []*wire.MsgTx
	failedTxs          []*wire.MsgTx        // 用来存储执行失败的tx，因为执行失败的tx还是需要付gas fee，所以需要记录留存
	updateShardInitOut *wire.UpdateAction   // 用来更新分片数据
	newOuts            []*wire.OutState     // 新产生的的out
	reduceActions      []*wire.ReduceAction // Store reduce action
}

// newSmartCE used as an instance of an initialization smartContractExecutor
func newSmartCE(idx shard.Index, contractAddr multivacaddress.Address) *smartContractExecutor {
	return &smartContractExecutor{
		shardIdx:        idx,
		contractAddress: contractAddr,
	}
}

// Execute used as execute smart contract
func (sce *smartContractExecutor) Execute(txs []*wire.MsgTx, sCInfo *wire.SmartContractInfo) (*smartContractExeResult, error) {
	var result *smartContractExeResult
	var successfulTxs []*wire.MsgTx
	var failedTxs []*wire.MsgTx
	var updateNewOuts []*wire.OutState
	var reduceActions []*wire.ReduceAction

	if sCInfo == nil {
		return nil, fmt.Errorf("the incoming smartContractInfo parameter is wrong, value: nil")
	}

	if !sCInfo.SmartContract.ContractAddr.IsEqual(sce.contractAddress) {
		return nil, fmt.Errorf("the contract address of smartContractInfo is wrong, want: %v,get: %v",
			sce.contractAddress, sCInfo.SmartContract.ContractAddr)
	}

	if sCInfo.ShardIdx != sce.shardIdx {
		return nil, fmt.Errorf("the shardIdx of smartContractInfo is wrong, want: %v,get: %v",
			sce.shardIdx, sCInfo.ShardIdx)
	}

	code := sCInfo.SmartContract.Code

	originShardInitOut := sCInfo.ShardInitOut
	originShardInitOutProof := sCInfo.ShardInitOutProof
	newShardInitOut := sCInfo.ShardInitOut

	vmExecutor, err := mvvm.NewExecuteMvvm(code)
	if err != nil {
		return nil, fmt.Errorf("fail to init mvvm, err: %v ", err)
	}

	for _, tx := range txs {
		shardInitOut, newOuts, newReduceActions, err := vmExecutor.Execute(tx, newShardInitOut)
		// TODO(issue 278):根据虚拟机返回的err类型断言来进一步操作
		if err != nil {
			failedTxs = append(failedTxs, tx)
			continue
		}
		successfulTxs = append(successfulTxs, tx)
		newShardInitOut = shardInitOut
		updateNewOuts = append(updateNewOuts, newOuts...)
		reduceActions = append(reduceActions, newReduceActions...)

	}
	result.successfulTxs = successfulTxs
	result.failedTxs = failedTxs
	result.updateShardInitOut = &wire.UpdateAction{
		OriginOut:      originShardInitOut,
		OriginOutProof: originShardInitOutProof,
		NewOut:         newShardInitOut,
	}
	result.newOuts = updateNewOuts
	result.reduceActions = reduceActions

	return result, nil
}
