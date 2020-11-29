// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iblockchain

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/chain"
)

// BlockChain is used to connect and operate levelDB.
type BlockChain interface {
	// Receives a new block, no-op if it is not in the same shard.
	// Returns true if the block is added to the BlockChain
	ReceiveBlock(block *wire.MsgBlock) bool

	// ReceiveBlock will receives blocks from all shards
	// Returns true if the block is added to the BlockChain
	ReceiveHeader(header *wire.BlockHeader) bool

	// ReceiveSlimBlock receives the concise information of the block and returns whether it is ok
	ReceiveSlimBlock(msg *wire.SlimBlock) bool

	// GetShardsHeight returns the height of specified shard.
	GetShardsHeight(shardIndex shard.Index) wire.BlockHeight

	// GetShardsBlockByHeight returns block of specified by height and shard, if not exit return nil.
	GetShardsBlockByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.MsgBlock

	// GetShardsBlockByHash returns block of specified by hash, if not exit return nil.
	GetShardsBlockByHash(headerHash chainhash.Hash) *wire.MsgBlock

	// GetShardsHeaderByHeight returns block header of specified shard by height, if not exit return nil.
	GetShardsHeaderByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.BlockHeader

	// GetShardsHeaderByHash returns block header of specified shard by hash, if not exit return nil.
	GetShardsHeaderByHash(headerHash chainhash.Hash) *wire.BlockHeader

	// GetShardsHeaderHashes returns the hashes of the specified shard and height range.
	GetShardsHeaderHashes(shardIndex shard.Index, fromHgt wire.BlockHeight, toHgt wire.BlockHeight) []chainhash.Hash

	// Register a trigger
	SetSyncTrigger(shard shard.Index, trigger chain.SyncTrigger)

	// GetSlimBlock returns the necessary information to build a clip tree for storage node.
	GetSlimBlock(toshard shard.Index, shardIndex shard.Index, hgt wire.BlockHeight) *wire.SlimBlock

	// GetSmartContract根据合约地址返回smart contract结构体
	GetSmartContract(addr multivacaddress.Address) *wire.SmartContract

	// GetSmartContractCodeOut根据合约地址以及分片编号返回代码Out
	GetSmartContractCodeOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState

	// GetSmartContractShardInitOut根据合约地址以及分片编号返回分片初始化out
	GetSmartContractShardInitOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState

	// ReceiveSmartContractShardInitOut将收到的outState作为分片数据持久化.
	ReceiveSmartContractShardInitOut(out *wire.OutState) error
}
