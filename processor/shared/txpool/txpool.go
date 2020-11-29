/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txpool

import (
	"crypto/sha256"
	"fmt"

	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// TxDesc contains a Tx and the Tx's priority based on fee.
type TxDesc struct {
	Tx       wire.MsgTxWithProofs
	Priority float64
}

type txPool struct {
	// TODO: Change pool to be a queue instead of map.
	pool           map[merkle.MerkleHash]*TxDesc
	usedOutsInPool map[merkle.MerkleHash]bool
	counter        int
}

// VerifyTransaction verifies that the transaction is legal based on the given root and tx,
// and returns error if it is not legal.
func (tp *txPool) VerifyTransaction(tx *wire.MsgTxWithProofs, root *merkle.MerkleHash) error {
	return tx.VerifyTxWithProof(root)
}

// addTransaction adds a transaction to the transaction pool, which verifies all inputs of the transaction.
func (tp *txPool) addTransaction(tx *wire.MsgTxWithProofs) (*TxDesc, error) {
	txD := &TxDesc{
		Tx:       *tx,
		Priority: 1.0, // TODO: calculate priority based on fee
	}
	usedOutsHash := []*merkle.MerkleHash{}
	for index, txin := range tx.Tx.TxIn {
		hash := merkle.ComputeMerkleHash(
			txin.PreviousOutPoint.ToUnspentOutState().ToBytesArray())
		_, exist := tp.usedOutsInPool[*hash]
		if exist {
			return nil, fmt.Errorf("The %d th txin has been used in other txs in txpool", index)
		}
		usedOutsHash = append(usedOutsHash, hash)
	}
	for _, hash := range usedOutsHash {
		tp.usedOutsInPool[*hash] = true
	}
	hash := makeMerkleHash(tx.ToBytesArray())
	tp.pool[*hash] = txD
	tp.counter++
	return txD, nil
}

// AddNewTransaction is a public method to add tx to txpool.
func (tp *txPool) AddNewTransaction(tx *wire.MsgTxWithProofs, root *merkle.MerkleHash) (*TxDesc, error) {
	if err := tx.VerifyTxWithProof(root); err != nil {
		return nil, fmt.Errorf("Failed to add new transactions, error msg: %s", err)
	}
	return tp.addTransaction(tx)
}

// NumberOfTransactions returns the current number of transactions in txpool.
func (tp *txPool) NumberOfTransactions() int {
	return tp.counter
}

// func getUnspentOutStateHash(outPoint *wire.OutPoint) *merkle.MerkleHash {
// 	unspentOutState := wire.OutState{
// 		OutPoint: *outPoint,
// 		State:    wire.StateUnused,
// 	}
// 	return makeMerkleHash(unspentOutState.ToBytesArray())
// }

// RefreshPool updates the txs in txpool based on the given update instance.
func (tp *txPool) RefreshPool(update *state.Update) {
	for hash, txD := range tp.pool {
		newProofs, err := update.UpdateProofs(txD.Tx.Proofs)
		if err != nil {
			for _, txin := range txD.Tx.Tx.TxIn {
				hash := merkle.ComputeMerkleHash(
					txin.PreviousOutPoint.ToUnspentOutState().ToBytesArray())
				delete(tp.usedOutsInPool, *hash)
			}
			delete(tp.pool, hash)
			tp.counter--
			continue
		}
		txD.Tx.Proofs = newProofs
	}
}

// GetTopTransactions returns the specified number of transactions based on number.
func (tp *txPool) GetTopTransactions(number int, update *state.Update, height int64) []*wire.MsgTxWithProofs {
	result := []*wire.MsgTxWithProofs{}
Txd:
	for _, txd := range tp.pool {
		for i, txIn := range txd.Tx.Tx.TxIn {
			out := txIn.PreviousOutPoint
			proof := txd.Tx.Proofs[i]
			// TODO: add a test case for outs been used already.
			if err := update.MarkSpentOuts(&out, &proof); err != nil {
				continue Txd
			}
			// Withdraw tx can be spent after locked.
			if out.IsLock(height) {
				continue Txd
			}
		}
		result = append(result, &txd.Tx)
		if len(result) >= number {
			break
		}
	}
	return result
}

func makeMerkleHash(content []byte) *merkle.MerkleHash {
	hash := merkle.MerkleHash(sha256.Sum256(content))
	return &hash
}
