/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txpool

import (
	"testing"

	"github.com/multivactech/MultiVAC/configs/params"

	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

var (
	shard0   = shard.IDToShardIndex(0)
	fakeTree = merkle.NewFullMerkleTree()
)

func setFakeTree() {
	genesisBlock := genesis.GenesisBlocks[shard0]
	outsMerkleTree := genesis.GenesisBlocksOutsMerkleTrees[shard0]
	_ = fakeTree.Append(&genesisBlock.Header.OutsMerkleRoot)
	_ = fakeTree.HookSecondLayerTree(outsMerkleTree)
}

func TestVerifyTransaction(t *testing.T) {
	fc := NewFakeChains()
	setFakeTree()
	fc.GetNewBlock(shard0)
	tp := NewTxPool()
	tx := fc.GetNewTx(shard0)

	err := tp.VerifyTransaction(tx, fakeTree.MerkleRoot)
	if err == nil {
		t.Error("Succeed to verify an invalid tx")
	}

	err = tp.VerifyTransaction(tx, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	if err != nil && tx.Tx.IsEnoughForReward() {
		t.Error("Fail to verify tx")
	}
}

func TestAddNewTransactions(t *testing.T) {
	fc := NewFakeChains()
	setFakeTree()
	fc.GetNewBlock(shard0)
	tp := NewTxPool()

	_, err := tp.AddNewTransaction(fc.GetNewTx(shard0), fakeTree.MerkleRoot)
	if err == nil {
		t.Error(err)
	}
	if tp.NumberOfTransactions() != 0 {
		t.Error("Succeed to add an invalid tx")
	}

	// Get a valid tx from genesis block into txpool...
	tx := fc.GetNewTx(shard0)
	block := fc.GetNewBlock(shard0)
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)
	update, err := m.NewUpdate()
	if err != nil {
		t.Error(err)
	}
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs
	_ = update.AddHeaders(&block.Header)
	_, err = m.GetUpdateWithFullBlock(block)
	if err != nil {
		t.Error(err)
	}

	txD, err := tp.AddNewTransaction(tx, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	if err != nil && tx.Tx.IsEnoughForReward() {
		t.Error(txD)
		t.Error(err)
	}
	if tp.NumberOfTransactions() == 0 && tx.Tx.IsEnoughForReward() {
		t.Error("Fail to add tx")
	}
}

func TestRefreshPool(t *testing.T) {
	fc := NewFakeChains()
	setFakeTree()
	fc.GetNewBlock(shard0)
	tp := NewTxPool()

	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)
	update, _ := m.NewUpdate()

	// Add txs into tp...
	numOfTxs := 5
	numOfFaultProofs := 0
	numOfWrongTxs := 0
	txDs := []*TxDesc{}
	for i := 0; i < numOfTxs; i++ {
		tx := fc.GetNewTx(shard0)
		txD, err := tp.AddNewTransaction(tx, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
		txDs = append(txDs, txD)
		if !tx.Tx.IsEnoughForReward() && params.EnableStorageReward {
			numOfWrongTxs++
		}
		if err != nil && params.EnableStorageReward && tx.Tx.IsEnoughForReward() {
			t.Error("Fail to add tx")
		}
	}
	if tp.NumberOfTransactions() != numOfTxs-numOfWrongTxs {
		t.Error("Wrong NumberOfTransactions")
	}

	for i := 0; i < numOfFaultProofs; i++ {
		if txDs[i].Tx.Proofs[0].ProofPath[0] == 0 {
			txDs[i].Tx.Proofs[0].ProofPath[0] = 1
		} else {
			txDs[i].Tx.Proofs[0].ProofPath[0] = 0
		}
	}
	// Refresh txpool based on the update instance...
	tp.RefreshPool(update)
	if tp.NumberOfTransactions() != numOfTxs-numOfFaultProofs-numOfWrongTxs {
		t.Error("Wrong NumberOfTransactions")
	}
}

func TestNumberOfTransactions(t *testing.T) {
	fc := NewFakeChains()
	setFakeTree()
	fc.GetNewBlock(shard0)
	fc.GetNewBlock(shard0)
	fc.GetNewBlock(shard0)
	tp := NewTxPool()

	tx1 := fc.GetNewTx(shard0)
	tx2 := fc.GetNewTx(shard0)
	tx3 := fc.GetNewTx(shard0)

	_, err := tp.AddNewTransaction(tx1, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	if err != nil {
		t.Error(err)
	}
	if tp.NumberOfTransactions() != 1 && tx1.Tx.IsEnoughForReward() {
		t.Error("Wrong NumberOfTransactions")
	}
	_, _ = tp.AddNewTransaction(tx2, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	_, _ = tp.AddNewTransaction(tx2, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	_, _ = tp.AddNewTransaction(tx1, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	if tp.NumberOfTransactions() != 2 && tx2.Tx.IsEnoughForReward() {
		t.Error("Wrong NumberOfTransactions")
	}
	_, _ = tp.AddNewTransaction(tx3, fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
	if tp.NumberOfTransactions() != 3 && tx3.Tx.IsEnoughForReward() {
		t.Error("Wrong NumberOfTransactions")
	}
}
