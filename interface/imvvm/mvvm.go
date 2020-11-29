// Copyright (c) 2018-present, MultiVAC Foundation
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package imvvm

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// DeployInitializer will initialize the initial data of each shard when a smart contract is deployed.
type DeployInitializer interface {
	// Initialize is called when a smart contract is deployed and initialized.
	Initialize(shard shard.Index) []byte
}

// SmartContractMvvmExecutor will execute the given code when the smart contract is applied, and return a bunch of
// OutState and Action to change the status of each shard.
type SmartContractMvvmExecutor interface {
	// Execute is called when a smart contract is applied and executed.
	Execute(tx *wire.MsgTx, shardState *wire.OutState) (*wire.OutState, []*wire.OutState, []*wire.ReduceAction, error)
}
