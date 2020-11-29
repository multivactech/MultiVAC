// /**
//  * Copyright (c) 2018-present, MultiVAC Foundation.
//  *
//  * This source code is licensed under the MIT license found in the
//  * LICENSE file in the root directory of this source tree.
//  */

// package storagenode

// import (
// 	"crypto/sha256"
// 	"fmt"
// 	"os"
// 	"testing"

// 	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
// 	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
// 	"github.com/multivactech/MultiVAC/merkle"
// 	"github.com/multivactech/MultiVAC/wire"
// )

// func createLedgerInfo() wire.LedgerInfo {
// 	// TODO: This is incorrect currently, only to be kept consistent with txpool UpdateWithFullBlock logic.
// 	// LedgerInfo in the block doesn't count the block itself.
// 	ledgerInfo := wire.LedgerInfo{}
// 	for _, shard := range genesis.ShardList {
// 		// Include all genesis blocks.
// 		ledgerInfo.SetShardHeight(shard, 1)
// 	}
// 	ledgerInfo.Size = int64(len(genesis.ShardList))
// 	return ledgerInfo
// }

// func createFakeTxWithProof() *wire.MsgTxWithProofs {
// 	tx = wire.NewMsgTx(wire.TxVersion)
// 	out := wire.TxOut{
// 		Value:  10,
// 		PkHash: chainhash.Hash{},
// 	}
// 	tx.AddTxOut(&out)
// 	txp := wire.MsgTxWithProofs{
// 		Tx: *tx,
// 	}
// 	return &txp
// }

// func setUp2(dataDir string) {
// 	fmt.Println("start setting up")
// 	shard0 = genesis.ShardList[0]

// 	ledgerInfo := createLedgerInfo()
// 	ledgerInfo.SetShardHeight(shard0, 2)
// 	var blockS0H2Body = wire.BlockBody{
// 		Transactions: []*wire.MsgTxWithProofs{createFakeTxWithProof()},
// 		LedgerInfo:   ledgerInfo,
// 	}

// 	var ledgerS0MerkleTree = merkle.NewFullMerkleTree()
// 	for _, shard := range genesis.ShardList {
// 		ledgerS0MerkleTree.Append(&genesis.GenesisBlocks[shard].Header.OutsMerkleRoot)
// 		if shard == shard0 {
// 			ledgerS0MerkleTree.Append(wire.CreateOutsMerkleTree(&blockS0H2Body).MerkleRoot)
// 		}
// 	}

// 	blockS0H2 = wire.MsgBlock{
// 		Header: wire.BlockHeader{
// 			Version:         1,
// 			ShardIndex:      genesis.ShardList[0],
// 			Height:          2,
// 			PrevBlockHeader: genesis.GenesisBlocks[shard0].Header.BlockHeaderHash(),
// 			// Empty block, no transactions.
// 			OutsMerkleRoot:        *wire.CreateOutsMerkleTree(&blockS0H2Body).MerkleRoot,
// 			ShardLedgerMerkleRoot: *ledgerS0MerkleTree.MerkleRoot,
// 			BlockBodyHash:         blockS0H2Body.BlockBodyHash(),
// 		},
// 		Body: &blockS0H2Body,
// 	}
// 	sn = newThreadedStorageNode(shard0, dataDir)
// 	sn.Start()
// 	for _, shard := range genesis.ShardList {
// 		if shard != shard0 {
// 			sn.OnBlockReceived(genesis.GenesisBlocks[shard])
// 		}
// 	}
// 	sn.OnBlockReceived(&blockS0H2)
// }

// func TestReceiveEmptyBlock(t *testing.T) {
// 	dataDir := "TestReceiveEmptyBlock"
// 	setUp(dataDir)
// 	merkleRoot := blockS0H2.Header.ShardLedgerMerkleRoot

// 	previousBlockHeaderHash := blockS0H2.Header.BlockHeaderHash()
// 	emptyBlock := wire.NewBlock(
// 		&genesis.ShardList[0],
// 		3,
// 		&previousBlockHeaderHash,
// 		merkle.MerkleHash{}, nil, nil, true)

// 	// After receiving empty block, the merkle tree of the ledger should stay same.
// 	sn.OnBlockReceived(emptyBlock)

// 	outState := genesis.CreateGenesisOutState(&genesis.ShardList[1], 3)
// 	merkleHash := merkle.MerkleHash(sha256.Sum256(outState.ToBytesArray()))
// 	merklePath, err := sn.sn.ledgerTree.GetMerklePath(&merkleHash)

// 	if err != nil {
// 		t.Errorf(err.Error())
// 	} else {
// 		err = merklePath.Verify(&merkleRoot)
// 		if err != nil {
// 			t.Errorf(err.Error())
// 		}
// 	}
// 	os.RemoveAll(dataDir)
// }
package storagenode
