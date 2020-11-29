/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package state

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// MinimumMerkleTree storages a merkleTree's root,size and last path.
type MinimumMerkleTree struct {
	Root     merkle.MerkleHash
	Size     int64
	LastPath merkle.MerklePath
}

// Copy the MinimumMerkleTree and return a new tree.
func (tree MinimumMerkleTree) Copy() MinimumMerkleTree {
	return MinimumMerkleTree{
		Root: tree.Root,
		Size: tree.Size,
		LastPath: merkle.MerklePath{
			// This assume the underlying merkle path do not modify hash values.
			Hashes:    append([]*merkle.MerkleHash{}, tree.LastPath.Hashes...),
			ProofPath: append([]byte{}, tree.LastPath.ProofPath...),
		},
	}
}

// Update Represents an mutation of the Shard Ledger.
type Update struct {
	DeltaTree     *merkle.DeltaMerkleTree
	newLedgerInfo wire.LedgerInfo
}

// GetShardHeight returns the shard's height given shardIndex.
func (update *Update) GetShardHeight(shardIdx shard.Index) int64 {
	return update.newLedgerInfo.GetShardHeight(shardIdx)
}

// AddHeaders Adds new headers in given order into this update.
// Delta Ledger Tree will be appended with outs tree root in each header.
// Ledger Info will be update accordingly for each shard.
func (update *Update) AddHeaders(headers ...*wire.BlockHeader) error {
	var outsMerkleRoots []*merkle.MerkleHash
	emptyMerkleHash := merkle.MerkleHash{}
	newLedgerInfo := update.newLedgerInfo
	for _, header := range headers {
		index := header.ShardIndex
		if header.OutsMerkleRoot != emptyMerkleHash {
			outsMerkleRoots = append(outsMerkleRoots,
				&header.OutsMerkleRoot)
		}
		if newLedgerInfo.GetShardHeight(index)+1 != header.Height {
			return fmt.Errorf("Wrong header height: expect %v but get %v",
				newLedgerInfo.GetShardHeight(index)+1, header.Height)
		}
		newLedgerInfo.SetShardHeight(index, newLedgerInfo.GetShardHeight(index)+1)
	}
	update.newLedgerInfo = newLedgerInfo
	if len(outsMerkleRoots) > 0 {
		err := update.DeltaTree.Append(outsMerkleRoots...)
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkSpentOuts Add a modification of spent Out state to this mutation.
func (update *Update) MarkSpentOuts(outPoint *wire.OutPoint, proof *merkle.MerklePath) error {
	newOutState := wire.OutState{
		OutPoint: *outPoint,
		State:    wire.StateUsed,
	}
	err := update.DeltaTree.UpdateLeaf(proof, makeMerkleHash(newOutState.ToBytesArray()))
	if err != nil {
		panic(fmt.Errorf("failed to mark spent outs, err: %v", err))
		//return err
	}
	return nil
}

// UpdateProofs trys to update a slice of input proofs (merkle path) based on this mutation.
func (update *Update) UpdateProofs(proofs []merkle.MerklePath) ([]merkle.MerklePath, error) {
	newProofs := make([]merkle.MerklePath, len(proofs))
	for i, proof := range proofs {
		newProof, err := update.DeltaTree.RefreshMerklePath(&proof)
		if err != nil {
			return nil, err
		}
		if err := newProof.Verify(update.DeltaTree.GetMerkleRoot()); err != nil {
			return nil, errors.New("Leaf node is changed")
		}
		newProofs[i] = *newProof
	}
	return newProofs, nil
}

// GetShardDataOutProof returns the proof of shard init data.
func (update *Update) GetShardDataOutProof(oldOut, newOut *wire.OutState) (*merkle.MerklePath, error) {
	return update.DeltaTree.GetShardDataMerklePath(makeMerkleHash(oldOut.ToBytesArray()), makeMerkleHash(newOut.ToBytesArray()))
}

// ShardStateManager keeps state of the shard ledger.
type ShardStateManager struct {
	// A delta merkle tree of the shard ledger, for updating all pending transaction proofs.
	Tree MinimumMerkleTree
	// The height of each shard.
	ShardHeights map[shard.Index]int64
	// The instance of blockchain, for getting all available headers.
	chain iblockchain.BlockChain
}

// NewShardStateManager initialize new ShardStateManager object with necessary info to build a delta merkle tree.
func NewShardStateManager(root *merkle.MerkleHash,
	size int64,
	latestPath *merkle.MerklePath,
	ledgerInfo *wire.LedgerInfo,
	chain iblockchain.BlockChain) *ShardStateManager {

	shardHeights := make(map[shard.Index]int64)
	logger.TxpoolLogger().Debugf("NewShardStateManager, size: %v, root %v, last node path %v", size, root, latestPath)
	for _, tag := range shard.ShardList {
		shardHeights[tag] = ledgerInfo.GetShardHeight(tag)
	}
	return &ShardStateManager{
		Tree: MinimumMerkleTree{
			Root:     *root,
			Size:     size,
			LastPath: *latestPath,
		},
		ShardHeights: shardHeights,
		chain:        chain,
	}
}

// GetShardHeight returns the height of Specified shard
func (state *ShardStateManager) GetShardHeight(shardIndex shard.Index) int64 {
	return state.ShardHeights[shardIndex]
}

// NewUpdate Creates a new empty Update (mutation) based on current state.
func (state *ShardStateManager) NewUpdate() (*Update, error) {
	deltaTree := merkle.NewDeltaMerkleTree(&state.Tree.Root)
	err := deltaTree.SetSizeAndLastNodeMerklePath(state.Tree.Size, &state.Tree.LastPath)
	if err != nil {
		panic(err)
	}
	update := Update{
		DeltaTree:     deltaTree,
		newLedgerInfo: *state.LedgerInfo(),
	}
	return &update, nil
}

// GetUpdateWithSlimBlock returns a new update instance which update with the given block.
// 1. marks spent outs in the merkle tree and updates the merkle root;
// 2. conditionally appends the merkle root of new outs from cached block headers, according to given full block header;
// 3. update merkle tree according to update action
func (state *ShardStateManager) GetUpdateWithSlimBlock(block *wire.SlimBlock) (*Update, error) {
	update, _ := state.NewUpdate()
	for _, tx := range block.Transactions {
		if tx.Tx.IsReduceTx() {
			continue
		}
		for index := range tx.Tx.TxIn {
			// Transactions from tx pool are supposed to be valid and have same amount of
			// proof and out points.
			err := update.MarkSpentOuts(&tx.Tx.TxIn[index].PreviousOutPoint, &tx.Proofs[index])
			if err != nil {
				return nil, err
			}
		}
	}
	// Update the ledger according to UpdateActions。
	for _, action := range block.UpdateActions {
		err := update.DeltaTree.UpdateLeaf(action.OriginOutProof, makeMerkleHash(action.NewOut.ToBytesArray()))
		if err != nil {
			// todo(MH): just for bug fix
			panic(fmt.Errorf("failed to update account according to updateaction, err: %v", err))
			//return nil, err
		}
	}

	headers, err := state.GetBlockHeadersToAppend(&block.LedgerInfo)
	if err != nil {
		return nil, err
	}
	if err := update.AddHeaders(headers...); err != nil {
		return nil, err
	}
	return update, nil
}

// GetUpdateWithFullBlock returns a new update instance which update with the given block.
// 1. marks spent outs in the merkle tree and updates the merkle root;
// 2. conditionally appends the merkle root of new outs from cached block headers, according to given full block header;
// 3. update merkle tree according to update action
func (state *ShardStateManager) GetUpdateWithFullBlock(block *wire.MsgBlock) (*Update, error) {
	update, err := state.NewUpdate()
	if err != nil {
		return nil, err
	}
	for _, tx := range block.Body.Transactions {
		if tx.Tx.IsReduceTx() {
			continue
		}
		for index := range tx.Tx.TxIn {
			// Transactions from tx pool are supposed to be valid and have same amount of
			// proof and out points.
			err := update.MarkSpentOuts(&tx.Tx.TxIn[index].PreviousOutPoint, &tx.Proofs[index])
			if err != nil {
				return nil, err
			}
		}
	}
	// Update the ledger according to UpdateActions。
	for _, action := range block.Body.UpdateActions {
		err := update.DeltaTree.UpdateLeaf(action.OriginOutProof, makeMerkleHash(action.NewOut.ToBytesArray()))
		if err != nil {
			// todo(MH): just for bug fix
			panic(fmt.Errorf("failed to update account according to updateaction, err: %v", err))
			//return nil, err
		}
	}
	headers, err := state.GetBlockHeadersToAppend(&block.Body.LedgerInfo)
	if err != nil {
		return nil, err
	}
	if err := update.AddHeaders(headers...); err != nil {
		return nil, err
	}
	return update, nil
}

// GetBlockHeadersToAppend retrieve continous block headers in each shard for the range from current ledger info to the given ledger info.
func (state *ShardStateManager) GetBlockHeadersToAppend(ledgerInfo *wire.LedgerInfo) ([]*wire.BlockHeader, error) {
	headers := []*wire.BlockHeader{}
	for _, shardIdx := range shard.ShardList {
		oldHeight := int64(state.GetShardHeight(shardIdx))
		newHeight := ledgerInfo.GetShardHeight(shardIdx)
		if newHeight < oldHeight {
			return nil, fmt.Errorf("Shard height in ledgerInfo is smaller than old height,"+
				" shard %v, old height %v, new height %v", shardIdx, oldHeight, newHeight)
		}
		if newHeight == oldHeight {
			continue
		}
		for i := oldHeight + 1; i <= newHeight; i++ {
			shardHeight := shard.IndexAndHeight{
				Index:  shardIdx,
				Height: i,
			}

			header := state.chain.GetShardsHeaderByHeight(shardIdx, wire.BlockHeight(i))
			if header == nil {
				return nil, fmt.Errorf("Can't find header for height %v", shardHeight)
			}
			headers = append(headers, header)
		}
	}
	return headers, nil
}

// ApplyUpdate Applies a given StateUpdate (mutation).
func (state *ShardStateManager) ApplyUpdate(update *Update) error {
	root := update.DeltaTree.GetMerkleRoot()
	size := update.DeltaTree.Size
	latestPath, err := update.DeltaTree.GetLastNodeMerklePath()
	if err != nil {
		panic(fmt.Errorf("Error applying update, err %v", err))
	}
	err = merkle.VerifyLastNodeMerklePath(root, size, latestPath)
	if err != nil {
		panic(fmt.Errorf("Last node path after applying update is wrong, err %v", err))
	}
	state.Tree = MinimumMerkleTree{
		Root:     *root,
		Size:     size,
		LastPath: *latestPath,
	}
	for _, shard := range shard.ShardList {
		newHeight := update.newLedgerInfo.GetShardHeight(shard)
		state.ShardHeights[shard] = newHeight
	}
	return nil
}

// LedgerInfo returns all shards' ledgerInfo
func (state *ShardStateManager) LedgerInfo() *wire.LedgerInfo {
	var size int64
	var ledgerInfo wire.LedgerInfo
	for tag, height := range state.ShardHeights {
		ledgerInfo.SetShardHeight(tag, height)
		size += height
	}
	ledgerInfo.Size = size
	return &ledgerInfo
}

// LocalLedgerInfo returns the ledgerInfo which blocks in state's chache may not update
func (state *ShardStateManager) LocalLedgerInfo() *wire.LedgerInfo {
	ledgerInfo := state.LedgerInfo()
	for _, tag := range shard.ShardList {
		height := ledgerInfo.GetShardHeight(tag)
		for {
			ok := state.chain.GetShardsHeaderByHeight(tag, wire.BlockHeight(height+1))
			if ok == nil {
				break
			}
			height++
			ledgerInfo.Size++
		}
		ledgerInfo.SetShardHeight(tag, height)
	}
	return ledgerInfo
}

// GetLedgerRoot returns the last updated ledger
func (state *ShardStateManager) GetLedgerRoot() merkle.MerkleHash {
	return state.Tree.Root
}

func makeMerkleHash(content []byte) *merkle.MerkleHash {
	hash := merkle.MerkleHash(sha256.Sum256(content))
	return &hash
}
