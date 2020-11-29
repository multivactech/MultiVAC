/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
package txprocessor

import (
	"fmt"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

var shardIdx = shard.Index(1)

// TODO(issue 277): 等到虚拟机支持执行智能合约后，完善测试
func TestSmartContractExecutor_Execute(t *testing.T) {
	contractAddrHash := &chainhash.Hash{}
	err := contractAddrHash.SetBytes([]byte("     execute smart contract     "))
	if err != nil {
		t.Errorf("fail to set contract address, err: %v", err)
	}
	contractAddr := contractAddrHash.FormatSmartContractAddress()

	sCInfo := &wire.SmartContractInfo{
		ShardIdx: shardIdx,
		SmartContract: &wire.SmartContract{
			ContractAddr: contractAddr,
			Code: []byte{0, 97, 115, 109, 1, 0, 0, 0, 1, 132, 128, 128, 128, 0, 1, 96, 0, 0, 3, 130, 128,
				128, 128, 0, 1, 0, 4, 132, 128, 128, 128, 0, 1, 112, 0, 0, 5, 131, 128, 128, 128, 0, 1, 0,
				1, 6, 129, 128, 128, 128, 0, 0, 7, 146, 128, 128, 128, 0, 2, 6, 109, 101, 109, 111, 114, 121, 2, 0, 5,
				95, 90, 49, 102, 118, 0, 0, 10, 136, 128, 128, 128, 0, 1, 130, 128, 128, 128, 0, 0, 11},
			APIList: []string{"_Z1fv"},
		},
		CodeOut:           &wire.OutState{},
		CodeOutProof:      nil,
		ShardInitOut:      &wire.OutState{},
		ShardInitOutProof: nil,
	}

	sCE := newSmartCE(shardIdx, contractAddr)
	tx := &wire.MsgTx{
		Shard:           shardIdx,
		ContractAddress: contractAddr,
		API:             "_Z1fv",
		Params:          []byte{},
	}

	fmt.Println(sCInfo, sCE, tx)
	//r, err := sCE.Execute([]*wire.MsgTx{tx}, sCInfo)

}
