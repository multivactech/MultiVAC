/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	assert.NotNil(sn, "Faild to create storagenode")
	defer clear()

	// test func-getShardHeight
	assert.Equal(sn.shardHeight, sn.ShardHeight(), "Invalid height of sn")

	// test func-getShardHeights
	heights := sn.getShardHeights()
	for _, shardIndex := range shard.ShardList {
		index := heights[shardIndex].Index
		assert.EqualValues(shardIndex, index, "Invalid index")

		height := heights[shardIndex].Height
		assert.EqualValues(int64(1), height, "Invalid height")
	}
}

func TestOnConfirmationReceived(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// test onConfirmationReceived
	sBlocks := createTestSlimBlocks(t, 3)
	cfms := createTestConfirmations(t, sBlocks)

	// H2-confirmation
	sn.onConfirmationReceived(cfms[0])

	assert.True(sn.confirmedHeaders[sBlocks[0].Header.BlockHeaderHash()], "Failed to cache confirmationH02")
	assert.Equal(cfms[0], sn.lastConfirmation[testShard], "Faild to save confirmation in lastConfirmation")

	// receive slimBlockH2
	sn.onSlimBlockReceived(sBlocks[0])

	assert.False(sn.confirmedHeaders[sBlocks[0].Header.BlockHeaderHash()], "Failed to delete confirmationH02")
	verifySlimBlockInfo(t, sBlocks[:1], sn.getSlimBlockInfo(testShard, 2))

	// receive slimBlockH4 & slimBlockH4 without confirmation
	sn.onSlimBlockReceived(sBlocks[1])
	sn.onSlimBlockReceived(sBlocks[2])

	verifySlimBlockInfo(t, sBlocks[:1], sn.getSlimBlockInfo(testShard, 2))
	assert.Equal(sBlocks[1], sn.receivedBlocks[sBlocks[1].Header.BlockHeaderHash()], "Failed to cache slimBlockH03")
	assert.Equal(sBlocks[2], sn.receivedBlocks[sBlocks[2].Header.BlockHeaderHash()], "Failed to cache slimBlockH04")

	// H4-confirmation
	sn.onConfirmationReceived(cfms[2])

	assert.Nil(sn.receivedBlocks[sBlocks[1].Header.BlockHeaderHash()], "Faild to delete confirmationH03")
	assert.Nil(sn.receivedBlocks[sBlocks[2].Header.BlockHeaderHash()], "Faild to delete confirmationH04")
	verifySlimBlockInfo(t, sBlocks, sn.getSlimBlockInfo(testShard, 2))
}

func TestOnSlimBlockReceived(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// test onSlimBlockReceived
	slimBlocks := createTestSlimBlocks(t, 3)
	cfms := createTestConfirmations(t, slimBlocks)

	// without confirmation & from testShard
	sn.onSlimBlockReceived(slimBlocks[0])

	assert.Equal(slimBlocks[0], sn.receivedBlocks[slimBlocks[0].Header.BlockHeaderHash()], "Invalid slimBlock")
	assert.Empty(sn.getSlimBlockInfo(testShard, 2), "Invalid slimBlock")

	// send 1st confirmation
	// reveive 1st slimBlock
	sn.onConfirmationReceived(cfms[0])

	assert.False(sn.confirmedHeaders[slimBlocks[0].Header.BlockHeaderHash()], "Faild to delete confirmation")
	assert.Empty(sn.receivedBlocks[slimBlocks[0].Header.BlockHeaderHash()], "Faild to delete block cache")
	assert.Equal(int64(2), sn.ShardHeight(), "Invalid height of shard")

	// verify if correct
	rsb := sn.getSlimBlock(testShard, slimBlocks[0].Header.BlockHeaderHash())
	verifySlimBlockInfo(t, []*wire.SlimBlock{slimBlocks[0]}, []*wire.SlimBlock{rsb})

	// fake slimBlock with wrong toShard
	slimBlocks[0].ToShard = shard.ShardList[2]
	sn.onSlimBlockReceived(slimBlocks[0])
	assert.Empty(sn.receivedBlocks[slimBlocks[0].Header.BlockHeaderHash()], "Invalid cache of slimBlock")
	// correct shardIndex
	slimBlocks[0].ToShard = testShard

	// send 2nd slimBlock
	// fake ledgerInfo
	sn.onConfirmationReceived(cfms[1])
	slimBlocks[1].LedgerInfo.SetShardHeight(shard.ShardList[2], int64(2))
	sn.updateFailCnt = updateLedgerReportThreshold
	sn.onSlimBlockReceived(slimBlocks[1])

	// verify if correct
	rsb = sn.getSlimBlockInfo(testShard, 2)[1]
	assert.Equal(slimBlocks[1].Header.BlockHeaderHash(), rsb.Header.BlockHeaderHash(), "Invalid header hash")
	assert.NotEqual(slimBlocks[1].LedgerInfo, sn.ledgerInfo, "Invalid ledgerInfo")
	assert.Equal(0, sn.updateFailCnt, "Invalid updateFailCnt")
	assert.Equal(int64(2), sn.ShardHeight(), "Invalid height of shard")

	// correct ledger
	slimBlocks[1].LedgerInfo.SetShardHeight(shard.ShardList[2], int64(1))
	sn.onSlimBlockReceived(slimBlocks[1])

	// send 3rd slimBlock
	// empty block
	slimBlocks[2].Header.IsEmptyBlock = true
	sn.onConfirmationReceived(cfms[2])
	sn.onSlimBlockReceived(slimBlocks[2])

	// verify if correct
	rsb = sn.getSlimBlockInfo(testShard, 2)[1]
	assert.Equal(slimBlocks[1].Header.BlockHeaderHash(), rsb.Header.BlockHeaderHash(), "Invalid header hash")
	assert.Equal(int64(4), sn.ShardHeight(), "Invalid height of shardIndex")
	assert.Equal(slimBlocks[1].LedgerInfo, sn.ledgerInfo, "Invalid ledgerInfo")
}

func TestGetSlimBlockInfo(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// send slimBlock
	slimBlocks := createTestSlimBlocks(t, 5)
	cfms := createTestConfirmations(t, slimBlocks)
	for i, sBlock := range slimBlocks {
		sn.onConfirmationReceived(cfms[i])
		sn.onSlimBlockReceived(sBlock)
	}

	// verify blocks from H02 to H06
	verifySlimBlockInfo(t, slimBlocks, sn.getSlimBlockInfo(testShard, 2))
	// verify blocks from H03 to H04
	verifySlimBlockInfo(t, slimBlocks[1:3], sn.getSlimBlockInfo(testShard, 3)[:2])
	// verify ledgerInfo
	assert.Equal(slimBlocks[4].LedgerInfo, sn.ledgerInfo, "Invalid ledgerInfo")
}

func TestGetSlimBlockSliceInfoSince(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// send slimBlock
	slimBlocks := createTestSlimBlocks(t, 5)
	cfms := createTestConfirmations(t, slimBlocks)
	for i, sBlock := range slimBlocks {
		sn.onConfirmationReceived(cfms[i])
		sn.onSlimBlockReceived(sBlock)
	}

	// verify blocks from H02 to H06
	verifySlimBlockInfo(t, slimBlocks, sn.getSlimBlockSliceInfoSince(testShard, 2))
	// verify blocks from H03 to H04
	verifySlimBlockInfo(t, slimBlocks[1:3], sn.getSlimBlockSliceInfoSince(testShard, 3)[:2])
	// verify ledgerInfo
	assert.Equal(slimBlocks[4].LedgerInfo, sn.ledgerInfo, "Invalid ledgerInfo")
}

func TestGetSlimBlockSliceInfoBetween(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// send slimBlock
	slimBlocks := createTestSlimBlocks(t, 5)
	cfms := createTestConfirmations(t, slimBlocks)
	for i, sBlock := range slimBlocks {
		sn.onConfirmationReceived(cfms[i])
		sn.onSlimBlockReceived(sBlock)
	}

	// verify blocks from H02 to H06
	verifySlimBlockInfo(t, slimBlocks, sn.getSlimBlockSliceInfoBetween(testShard, 2, 7))
	// verify blocks from H03 to H04
	verifySlimBlockInfo(t, slimBlocks[1:3], sn.getSlimBlockSliceInfoBetween(testShard, 3, 5))
	// verify ledgerInfo
	assert.Equal(slimBlocks[4].LedgerInfo, sn.ledgerInfo, "Invalid ledgerInfo")
}

func TestOnBlockReceived(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// save block
	blockList := createTestBlocks(t, 10)
	for _, block := range blockList {
		sn.onBlockReceived(block)
	}

	assert.Empty(sn.getSlimBlockInfo(testShard, 2), "Should not saved slimBlock in blockChain")
	assert.Equal(10, len(sn.getBlockSliceInfoSince(2)), "Invalid number of blocks saved in blockChain")

	for _, block := range blockList {
		actBlock := sn.getBlock(testShard, block.Header.BlockHeaderHash())
		assert.Equal(block.Body.BlockBodyHash(), actBlock.Body.BlockBodyHash(), "Invalid bodyHash")
		assert.Equal(block.Header.BlockHeaderHash(), actBlock.Header.BlockHeaderHash(), "Invalid headerHash")
		assert.Equal(block.Header.Height, actBlock.Header.Height, "Invalid height")

		slimBlock, _ := wire.GetSlimBlockFromBlockByShard(block, testShard)
		assert.Equal(slimBlock, sn.receivedBlocks[block.Header.BlockHeaderHash()], "Invalid cache of slimBlock")
	}
}

func TestGetBlockInfo(t *testing.T) {
	sn := createTestStorageNode(t)
	defer clear()

	// send block
	blockList := createTestBlocks(t, 10)
	for _, block := range blockList {
		sn.onBlockReceived(block)
	}

	// verify blocks H01
	verifyBlockInfo(t, []*wire.MsgBlock{genesis.GenesisBlocks[testShard]}, sn.getBlockInfo(1)[:1])
	// verify blocks from H06 to H11
	verifyBlockInfo(t, blockList[4:], sn.getBlockInfo(6))
	// verify blocks from H03 to H04
	verifyBlockInfo(t, blockList[1:3], sn.getBlockInfo(3)[:2])
}

func TestGetBlockSliceInfoSince(t *testing.T) {
	sn := createTestStorageNode(t)
	defer clear()

	// send block
	blockList := createTestBlocks(t, 10)
	for _, block := range blockList {
		sn.onBlockReceived(block)
	}

	// verify blocks H01
	verifyBlockInfo(t, []*wire.MsgBlock{genesis.GenesisBlocks[testShard]}, sn.getBlockSliceInfoSince(1)[:1])
	// verify blocks from H06 to H11
	verifyBlockInfo(t, blockList[4:], sn.getBlockSliceInfoSince(6))
	// verify blocks from H03 to H04
	verifyBlockInfo(t, blockList[1:3], sn.getBlockSliceInfoSince(3)[:2])
}

func TestGetBlockSliceInfoBetween(t *testing.T) {
	sn := createTestStorageNode(t)
	defer clear()

	// send block
	blockList := createTestBlocks(t, 10)
	for _, block := range blockList {
		sn.onBlockReceived(block)
	}

	// verify blocks H01
	verifyBlockInfo(t, []*wire.MsgBlock{genesis.GenesisBlocks[testShard]}, sn.getBlockSliceInfoBetween(1, 2))
	// verify blocks from H06 to H11
	verifyBlockInfo(t, blockList[4:], sn.getBlockSliceInfoBetween(6, 12))
	// verify blocks from H03 to H04
	verifyBlockInfo(t, blockList[1:3], sn.getBlockSliceInfoBetween(3, 5))
}

func TestOnHandleConfirmation(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// send slimBlock
	slimBlocks := createTestSlimBlocks(t, 2)
	cfms := createTestConfirmations(t, slimBlocks)
	sn.onSlimBlockReceived(slimBlocks[0])
	sn.onSlimBlockReceived(slimBlocks[1])
	sn.onHandleConfirmation(cfms[1])

	// verify
	sBlocks := sn.getSlimBlockInfo(testShard, 2)
	assert.Equal(2, len(sBlocks), "Invalid length")
	assert.Equal(slimBlocks[0].Header.BlockHeaderHash(), sBlocks[0].Header.BlockHeaderHash(), "Invalid H02-header hash")
	assert.Equal(slimBlocks[1].Header.BlockHeaderHash(), sBlocks[1].Header.BlockHeaderHash(), "Invalid H03-header hash")
}

func TestOnTransactionReceived(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// send transaction
	txs := createTestTransactions(t, 3)
	sn.onTransactionReceived(txs[0])

	// verify
	assert.Equal(1, sn.pendingTransactionNumber(), "Invalid num of trans")
	fTxs := &wire.MsgFetchTxs{ShardIndex: testShard}
	rTxs, err := sn.onRequestingTransactions(fTxs)
	assert.Nil(err, "Faild to get transaction")
	assert.Equal(txs[0], &rTxs.Txs[0].Tx, "Invalid Tx")

	// fake tx from other shard
	txs[1].Shard = shard.ShardList[2]
	sn.onTransactionReceived(txs[1])
	assert.Equal(1, sn.pendingTransactionNumber(), "Invalid num of trans")

	// fake tx used repeatly
	txs[2].TxIn = append(txs[2].TxIn, txs[2].TxIn[0])
	sn.onTransactionReceived(txs[2])
	assert.Equal(1, sn.pendingTransactionNumber(), "Invalid num of trans")
}

func TestGetHeaderHashes(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()

	// save slimBlocks
	sBlocks := createTestSlimBlocks(t, 2)
	cfms := createTestConfirmations(t, sBlocks)
	for i, sBlock := range sBlocks {
		sn.onConfirmationReceived(cfms[i])
		sn.onSlimBlockReceived(sBlock)
	}
	// create blockLocator
	locator := &wire.BlockLocator{
		ShardIdx:   testShard,
		FromHeight: 1,
		ToHeight:   3,
	}

	invGroup := sn.getHeaderHashes([]*wire.BlockLocator{locator})[0]
	assert.Equal(3, len(invGroup.HeaderHashes), "Invalid length of invGroup")
	assert.Equal(testShard, invGroup.Shard, "Invalid shard index")

	hashes := invGroup.HeaderHashes
	assert.Equal(genesis.GenesisBlocks[testShard].Header.BlockHeaderHash(), hashes[0], "Invalid 1st header hash")
	assert.Equal(sBlocks[0].Header.BlockHeaderHash(), hashes[1], "Invalid 2nd header hash")
	assert.Equal(sBlocks[1].Header.BlockHeaderHash(), hashes[2], "Invalid 3rd header hash")
}

func TestGetOutState(t *testing.T) {
	assert := assert.New(t)
	sn := createTestStorageNode(t)
	defer clear()
	outs := genesis.GenesisBlocks[testShard].Body.Outs

	// get transaction
	out, err := sn.getOutState(&outs[0].OutPoint)
	assert.Nil(err, "Faild to get out state")
	assert.Equal(outs[0].State, out.State, "Invalid spent state")

	// unsaved out
	fHash := chainhash.Hash{}
	err = fHash.SetBytes([]byte("It's fake hash but asked for 32."))
	assert.Nil(err, "Faild to set hash into bytes")
	outs[1].TxHash = fHash
	out, err = sn.getOutState(&outs[1].OutPoint)
	assert.Nil(out, "Out shouldn't exsited")
	assert.Equal("GetOutState: not found", err.Error(), "Invalid error message")
}
