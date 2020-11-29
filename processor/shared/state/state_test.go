/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package state_test

import (
	"fmt"
	"testing"

	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/processor/shared/state"
	"github.com/multivactech/MultiVAC/processor/shared/state/testingutils"
)

var (
	shard0 = shard.ShardList[0]
	// todo:it seems unused.
	//shard1 = shard.ShardList[1]
)

func TestAddHeaderToNewUpdate(t *testing.T) {
	fc := testingutils.NewFakeChains()
	block := fc.GetNewBlock(shard0, fc.GetNewTx(shard0))
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)

	update, err := m.NewUpdate()
	if err != nil {
		t.Error(err)
	}
	err = update.AddHeaders(&block.Header)
	if err != nil {
		fmt.Printf("failed to add headers,err:%v", err)
	}

	// Generate a valid tx
	tx := fc.GetNewTx(shard0)
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs

	err = update.AddHeaders(&fc.GetNewBlock(shard0).Header) // TODO: add tx
	if err != nil {
		fmt.Printf("failed to add headers,err:%v", err)
	}
	if update.GetShardHeight(shard0) != 3 {
		t.Errorf(
			"(expected) Wrong new shard height for shard0: want %v but got %v",
			3, update.GetShardHeight(shard0),
		)
	}
	if *update.DeltaTree.GetMerkleRoot() != *fc.Chains[shard0].ShardLedgerTree.MerkleRoot {
		t.Errorf("Wrong merkle root: want %v but get %v",
			*update.DeltaTree.GetMerkleRoot(),
			*fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		)
	}
}

func TestApplyUpdate(t *testing.T) {
	fc := testingutils.NewFakeChains()
	block := fc.GetNewBlock(shard0, fc.GetNewTx(shard0))
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)

	update, err := m.NewUpdate()
	if err != nil {
		t.Error(err)
	}

	err = update.AddHeaders(
		&block.Header,
		// &fc.GetNewBlock(shard0).Header, // TODO: add tx.
		&fc.GetNewBlock(shard0).Header,
	)
	if err != nil {
		t.Error(err)
	}
	err = m.ApplyUpdate(update)
	if err != nil {
		t.Error(err)
	}
	if m.LedgerInfo().GetShardHeight(shard0) != 3 {
		t.Errorf(
			"(expected) Wrong new shard height for shard0: want %v but got %v",
			3, update.GetShardHeight(shard0),
		)
	}
	if m.Tree.Root != *fc.Chains[shard0].ShardLedgerTree.MerkleRoot {
		t.Errorf("Wrong merkle root: want %v but get %v",
			m.Tree.Root, *fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		)
	}

	// Generate a valid tx
	tx := fc.GetNewTx(shard0)
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs

	err = update.AddHeaders(&fc.GetNewBlock(shard0).Header)
	if err != nil {
		t.Error(err)
	}
	err = m.ApplyUpdate(update)
	if err != nil {
		t.Error(err)
	}
	if m.LedgerInfo().GetShardHeight(shard0) != 4 {
		t.Errorf(
			"(expected) Wrong new shard height for shard0: want %v but got %v",
			4, update.GetShardHeight(shard0),
		)
	}
	if m.Tree.Root != *fc.Chains[shard0].ShardLedgerTree.MerkleRoot {
		t.Error("Wrong merkle root")
	}
}

func TestMarkSpentOuts(t *testing.T) {
	fc := testingutils.NewFakeChains()
	// Add to initial blocks to shard 0
	block := fc.GetNewBlock(shard0, fc.GetNewTx(shard0))
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)
	// m.CacheBlockHeader(&block.Header)
	update, _ := m.NewUpdate()

	// Generate a valid tx
	tx := fc.GetNewTx(shard0)
	// An update for its proofs should succeed, and be a noop.
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs

	err = update.AddHeaders(&block.Header)
	if err != nil {
		t.Error(err)
	}
	// the recently out-of-date tx proofs should be able to be update.
	newProofs, err = update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}

	// Have to call GetNewBlock to update the fake chain ledger tree
	fc.GetNewBlock(shard0)
	// All new proofs are valid within the new tree.
	for _, proof := range newProofs {
		err = proof.Verify(fc.Chains[shard0].ShardLedgerTree.MerkleRoot)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestUpdateProofs(t *testing.T) {
	fc := testingutils.NewFakeChains()
	// Add to initial blocks to shard 0
	block := fc.GetNewBlock(shard0, fc.GetNewTx(shard0))
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)
	update, _ := m.NewUpdate()

	// Generate a valid tx
	tx := fc.GetNewTx(shard0)
	// An update for its proofs should succeed, and be a noop.
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs
	err = update.AddHeaders(&block.Header)
	if err != nil {
		t.Error(err)
	}
	// Try to modify the hash of the DeltaTree
	t.Log("Real root hash:", update.DeltaTree.MerkleRoot)
	var fakeHashByte [merkle.HashSize]byte
	fakeRootHash := merkle.MerkleHash(fakeHashByte)
	update.DeltaTree.MerkleRoot = &fakeRootHash
	t.Log("Fake root hash:", update.DeltaTree.MerkleRoot)

	_, err = update.UpdateProofs(tx.Proofs)
	if err == nil {
		t.Error("UpdateProofs should throw exception: incorrect DeltaTree hash.")
	}
}

func TestNewUpdate(t *testing.T) {
	fc := testingutils.NewFakeChains()
	// Add to initial blocks to shard 0
	// block := fc.GetNewBlock(shard0, fc.GetNewTx(shard0))
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)
	_, err := m.NewUpdate()
	if err != nil {
		t.Error(err)
	}

	// Try to modify the size of the MinimumMerkleTree, to make the verification fail.
	// Recover the panic from NewUpdate.
	exceptedTreeSize := m.Tree.Size
	m.Tree.Size = 10
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
			t.Logf("Current Tree.Size = %v, excepted Tree.Size = %v",
				m.Tree.Size, exceptedTreeSize)
		}
	}()

	_, err = m.NewUpdate()
	if err == nil {
		t.Error("Should throw exception.")
	}
}

func TestGetUpdateWithFullBlock(t *testing.T) {
	fc := testingutils.NewFakeChains()
	block := fc.GetNewBlock(shard0)
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)

	// Check whether a void block can get update.
	update, err := m.GetUpdateWithFullBlock(block)
	if err != nil {
		t.Error(err)
	}
	err = m.ApplyUpdate(update)
	if err != nil {
		t.Error(err)
	}
	// Compare the block header's Merkle root with the DeltaTree Merkle root
	if block.Header.ShardLedgerMerkleRoot != *update.DeltaTree.MerkleRoot {
		t.Errorf("Dismatch Merkle root of block-header %v and DeltaTree-header %v",
			block.Header.ShardLedgerMerkleRoot, update.DeltaTree.MerkleRoot)
	}

	// Generate a valid tx, update it into the block header.
	tx := fc.GetNewTx(shard0)
	update, err = m.NewUpdate()
	if err != nil {
		t.Error(err)
	}
	newProofs, err := update.UpdateProofs(tx.Proofs)
	if err != nil {
		t.Error(err)
	}
	tx.Proofs = newProofs
	err = update.AddHeaders(&block.Header)
	if err != nil {
		t.Error(err)
	}
	update, err = m.GetUpdateWithFullBlock(block)
	if err != nil {
		t.Error(err)
	}
	err = m.ApplyUpdate(update)
	if err != nil {
		t.Error(err)
	}
	if block.Header.ShardLedgerMerkleRoot != *update.DeltaTree.MerkleRoot {
		t.Errorf("Dismatch Merkle root of block-header %v and DeltaTree-header %v",
			block.Header.ShardLedgerMerkleRoot, update.DeltaTree.MerkleRoot)
	}
}

func TestGetBlockHeadersToAppend(t *testing.T) {
	fc := testingutils.NewFakeChains()
	block := fc.GetNewBlock(shard0)
	m := state.NewShardStateManager(fc.Chains[shard0].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard0].ShardLedgerTree.Size,
		fc.GetLastPath(shard0), fc.GetLedgerInfo(shard0), nil)

	_, err := m.GetBlockHeadersToAppend(&block.Body.LedgerInfo)
	if err != nil {
		t.Error(err)
	}

	block.Body.LedgerInfo.SetShardHeight(shard0, 0)
	_, err = m.GetBlockHeadersToAppend(&block.Body.LedgerInfo)
	if err == nil {
		t.Error(err)
	}
}
