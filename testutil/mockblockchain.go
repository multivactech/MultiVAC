/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package testutil

import (
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/chain"
)

// MockChain used to facility deposit test
type MockChain struct {
	chasche map[shard.Index]map[wire.BlockHeight]*wire.MsgBlock
}

// NewMockChain returns the new instance of MockChain
func NewMockChain() iblockchain.BlockChain {
	chain := new(MockChain)
	chain.chasche = make(map[shard.Index]map[wire.BlockHeight]*wire.MsgBlock)
	for _, shard := range shard.ShardList {
		chain.chasche[shard] = make(map[wire.BlockHeight]*wire.MsgBlock)
		genesisBlock := genesis.GenesisBlocks[shard]
		chain.ReceiveBlock(genesisBlock)
	}
	return chain
}

// ReceiveBlock receives a new block, no-op if it is not in the same shard
func (chain *MockChain) ReceiveBlock(block *wire.MsgBlock) bool {
	shard := block.Header.ShardIndex
	height := wire.BlockHeight(block.Header.Height)
	if _, ok := chain.chasche[shard]; !ok {
		chain.chasche[shard] = make(map[wire.BlockHeight]*wire.MsgBlock)
	}
	chain.chasche[shard][height] = block
	return true
}

// ReceiveHeader receives header and puts it into db.
func (chain *MockChain) ReceiveHeader(header *wire.BlockHeader) bool {
	return true
}

// ReceiveSlimBlock receives slimBlock and puts it into db.
func (chain *MockChain) ReceiveSlimBlock(msg *wire.SlimBlock) bool {
	return true
}

// GetShardsHeight returns the height of the shard.
func (chain *MockChain) GetShardsHeight(shardIndex shard.Index) wire.BlockHeight {
	return wire.BlockHeight(1)
}

// GetShardsBlockByHeight returns a block according to the shard and height.
func (chain *MockChain) GetShardsBlockByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.MsgBlock {
	if h, ok := chain.chasche[shardIndex][hgt]; ok {
		return h
	}
	return nil
}

// GetShardsBlockByHash returns a block according to the header hash.
func (chain *MockChain) GetShardsBlockByHash(headerHash chainhash.Hash) *wire.MsgBlock {
	return nil
}

// GetShardsHeaderByHeight returns a header according to the shard and height.
func (chain *MockChain) GetShardsHeaderByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.BlockHeader {
	if h, ok := chain.chasche[shardIndex][hgt]; ok {
		return &h.Header
	}
	return nil
}

// GetShardsHeaderByHash returns a header accroding to header hash.
func (chain *MockChain) GetShardsHeaderByHash(headerHash chainhash.Hash) *wire.BlockHeader {
	return nil
}

// GetShardsHeaderHashes returns the header hash according to shard index.
func (chain *MockChain) GetShardsHeaderHashes(shardIndex shard.Index, fromHgt wire.BlockHeight, toHgt wire.BlockHeight) []chainhash.Hash {
	return nil
}

// SetSyncTrigger maybe unused now.
func (chain *MockChain) SetSyncTrigger(shard shard.Index, trigger chain.SyncTrigger) {

}

// GetSlimBlock returns slimblock by shard index and height.
func (chain *MockChain) GetSlimBlock(toshard shard.Index, shardIndex shard.Index, hgt wire.BlockHeight) *wire.SlimBlock {
	return nil
}

// GetSmartContract returns smart-contract data from db.
func (chain *MockChain) GetSmartContract(addr multivacaddress.Address) *wire.SmartContract {
	return nil
}

// GetSmartContractCodeOut returns smart-contract out from db.
func (chain *MockChain) GetSmartContractCodeOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	return nil
}

// GetSmartContractShardInitOut returns data of smart-contract.
func (chain *MockChain) GetSmartContractShardInitOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	return nil
}

// ReceiveSmartContractShardInitOut puts smart-contract init data to db.
func (chain *MockChain) ReceiveSmartContractShardInitOut(out *wire.OutState) error {
	return nil
}
