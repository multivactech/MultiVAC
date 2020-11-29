// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package itxprocessor

import (
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/wire"
)

// TxProcessor interface which executes txs for one block.
// Before proposing a block, add all candidate txs into txprocessor. Then run
// TxProcessor#Execute() which will generate a Result object. See interface
// Result for more info about what's in it.
type TxProcessor interface {
	AddTx(tx *wire.MsgTx)
	Execute(proposeHeight int64) Result
}

// Result of executing txs.
// After executing txs, a list of new out states are generated. And a merkle
// tree is build for these out states. Besides, Result interface also stores
// the list of txs which are successfully executed. This list excludes txs
// which encounter any error. So when proposing blocks, use the successful txs
// instead of original candidate txs.
type Result interface {
	GetOutsMerkleTree() *merkle.FullMerkleTree
	GetMerkleRoot() *merkle.MerkleHash
	GetSuccessfulTxs() []*wire.MsgTx
	GetOutStates() []*wire.OutState
	GetSmartContracts() []*wire.SmartContract
	GetUpdateActions() []*wire.UpdateAction
	GetReduceActions() []*wire.ReduceAction
}
