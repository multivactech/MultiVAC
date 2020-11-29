// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

func newSmartContractInfoForTest() *SmartContractInfo {
	// TODO (wangruichao@mtv.ac): empty for now, add smart contract here
	sc := SmartContractInfo{
		ShardIdx: shard.ShardList[0],

		SmartContract: &SmartContract{
			ContractAddr: isysapi.SysAPIAddress,
			Code:         []byte{},
			APIList:      []string{},
		},

		CodeOut:           &OutState{},
		CodeOutProof:      &merkle.MerklePath{},
		ShardInitOut:      &OutState{},
		ShardInitOutProof: &merkle.MerklePath{},
	}
	return &sc
}
