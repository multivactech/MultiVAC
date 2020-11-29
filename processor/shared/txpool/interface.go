/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txpool

import (
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// TxPool is a interface that contains some methods used to add, verify and get transactions with proofs.
type TxPool interface {
	AddNewTransaction(tx *wire.MsgTxWithProofs, root *merkle.MerkleHash) (*TxDesc, error)
	VerifyTransaction(tx *wire.MsgTxWithProofs, root *merkle.MerkleHash) error
	GetTopTransactions(number int, update *state.Update, height int64) []*wire.MsgTxWithProofs
	NumberOfTransactions() int
	RefreshPool(update *state.Update)
}

// NewTxPool returns a new instance of txPool.
func NewTxPool() TxPool {
	return &txPool{
		pool:           make(map[merkle.MerkleHash]*TxDesc),
		usedOutsInPool: make(map[merkle.MerkleHash]bool),
	}
}
