/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package genesis

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

var genesisDepositValue = new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1000000000000000000))

// NumberOfOutsInEachGenesisBlock indicats how many outs are there in each genesis block.
var NumberOfOutsInEachGenesisBlock = 1000

// TotalAmountOfEachShard represents number of crypto coins are there in each genesis block.
var TotalAmountOfEachShard *big.Int

// it seems unused.
//func convertToPkHash(pk signature.PublicKey) chainhash.Hash {
//	tpkBytes := []byte(pk)
//	var pkBytes [chainhash.HashSize]byte
//	copy(pkBytes[:], tpkBytes[:])
//	return chainhash.Hash(pkBytes)
//}

// ConvertByteToPkHash coverts pk bytes to chainhash.Hash
func ConvertByteToPkHash(pk []byte) chainhash.Hash {
	var pkBytes [chainhash.HashSize]byte
	copy(pkBytes[:], pk[:])
	return chainhash.Hash(pkBytes)
}

// ConvertStringToPkHash coverts pk string to chainhash.Hash
func ConvertStringToPkHash(pkString string) (chainhash.Hash, error) {
	pkbytes, err := hex.DecodeString(pkString)
	if err != nil {
		return chainhash.Hash{}, err
	}
	to := ConvertByteToPkHash(pkbytes)
	return to, nil
}

func setTotalAmountOfEachShard() {
	var success bool
	TotalAmountOfEachShard, success = new(big.Int).SetString("100000000000000000000000000000", 10)
	if !success {
		panic("generate TotalAmountOfEachShard err!")
	}
}

// CreateGenesisOutState makes it easier to create an out state corresponding to a outpoint
// in genesis block.
func CreateGenesisOutState(shard shard.Index, index int, txHash *chainhash.Hash) *wire.OutState {
	setTotalAmountOfEachShard()
	data := wire.MTVCoinData{
		Value: new(big.Int).Div(TotalAmountOfEachShard, big.NewInt(int64(NumberOfOutsInEachGenesisBlock))),
	}
	dataBytes, _ := rlp.EncodeToBytes(data)

	outState := &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          *txHash,
			Index:           index,
			Shard:           shard,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shard], multivacaddress.UserAddress),
			Data:            dataBytes,
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.Transfer,
		},
		State: wire.StateUnused,
	}
	return outState
}

// createGenesisDeposit generates deposit out for the given pk. This is for the first miner to start up.
// Every pk has 10,0000 MTV in each shard.
func createGenesisDeposit(shard shard.Index, index int, txHash *chainhash.Hash, to multivacaddress.Address) *wire.OutState {
	depositData := wire.MTVDepositData{
		Value:          genesisDepositValue,
		Height:         1,
		BindingAddress: to,
	}
	rawData := depositData.ToData()
	outState := &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          *txHash,
			Index:           index,
			Shard:           shard,
			UserAddress:     to,
			Data:            rawData,
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.DepositOut,
		},
		State: wire.StateUnused,
	}
	return outState
}

// GenerateGenesisTx returns a empty tx with the given shard.
func GenerateGenesisTx(shard shard.Index) *wire.MsgTx {
	setTotalAmountOfEachShard()
	return wire.NewMsgTx(wire.TxVersion, shard)
}

func genDepositTxWithProof(shard shard.Index) *wire.MsgTxWithProofs {
	tx := wire.NewMsgTx(wire.TxVersion, shard)
	tx.API = isysapi.SysAPIDeposit
	// Build tx-in for every deposit tx.
	data := wire.MTVCoinData{
		Value: genesisDepositValue,
	}
	dataBytes, _ := rlp.EncodeToBytes(data)
	in := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Data:            dataBytes,
			Shard:           shard,
			TxHash:          chainhash.Hash{},
			UserAddress:     multivacaddress.Address{},
			ContractAddress: isysapi.SysAPIAddress,
			Index:           0,
			Type:            wire.Transfer,
		},
	}
	tx.AddTxIn(in)
	return &wire.MsgTxWithProofs{
		Tx:     *tx,
		Proofs: []merkle.MerklePath{},
	}
}

// generateOutsMerkleTree generates genesis block merkle tree for the shard.
// It will generates a tx with NumberOfOutsInEachGenesisBlock transfer outs and
// len(pk-pool.pks) deposit outs and returns the out merkle tree and the tx with
// tx-proof.
func generateOutsMerkleTree(shard shard.Index) (*merkle.FullMerkleTree, []*wire.MsgTxWithProofs, []*wire.OutState) {
	var outsHashes []*merkle.MerkleHash
	var txWithProofs []*wire.MsgTxWithProofs
	var dTxsWithProofs []*wire.MsgTxWithProofs
	var outStates []*wire.OutState
	tx := GenerateGenesisTx(shard)
	txWithProof := wire.MsgTxWithProofs{
		Tx:     *tx,
		Proofs: []merkle.MerklePath{},
	}
	txHash := tx.TxHash()

	// Create normal transactions.
	for i := 0; i < NumberOfOutsInEachGenesisBlock; i++ {
		outState := CreateGenesisOutState(shard, i, &txHash)
		outStates = append(outStates, outState)
		outsHash := merkle.MerkleHash(sha256.Sum256(outState.ToBytesArray()))
		outsHashes = append(outsHashes, &outsHash)
	}
	pkPool := NewPool()

	// Create genesis deposit for all pks in the PkPool.
	for i, pk := range pkPool.Pks() {
		// Deposit transaction
		dtxp := genDepositTxWithProof(shard)
		txHash := dtxp.Tx.TxHash()
		to, _ := multivacaddress.StringToUserAddress(pk)
		// Build deposit outs
		outState := createGenesisDeposit(shard, NumberOfOutsInEachGenesisBlock+i, &txHash, to)
		outStates = append(outStates, outState)
		outHash := merkle.MerkleHash(sha256.Sum256(outState.ToBytesArray()))
		outsHashes = append(outsHashes, &outHash)
		// Build deposit tx
		dp := isysapi.DepositParams{
			TransferParams: &isysapi.TransferParams{
				To:     to,
				Amount: new(big.Int).Sub(genesisDepositValue, isysapi.StorageRewardAmount),
				Shard:  shard,
			},
		}
		params, err := rlp.EncodeToBytes(dp)
		if err != nil {
			panic(err)
		}
		dtxp.Tx.Params = params
		dTxsWithProofs = append(dTxsWithProofs, dtxp)
	}
	// Generate merkle tree.
	fullMerkleTree := merkle.NewFullMerkleTree(outsHashes...)

	// Generate transfer tx's proof.
	for index := range outsHashes {
		merklePath, err := fullMerkleTree.GetMerklePath(outsHashes[index])
		if err != nil {
			panic(err)
		}
		// Deposit outs' index start at NumberOfOutsInEachGenesisBlock.
		if index >= NumberOfOutsInEachGenesisBlock {
			dtxp := dTxsWithProofs[index-NumberOfOutsInEachGenesisBlock]
			dtxp.Proofs = append(dtxp.Proofs, *merklePath)
		}
		txWithProof.Proofs = append(txWithProof.Proofs, *merklePath)
	}

	txWithProofs = append(txWithProofs, &txWithProof)
	txWithProofs = append(txWithProofs, dTxsWithProofs...)
	logger.ServerLogger().Debugf("zz Shard %d genesis root is %v", shard, fullMerkleTree.MerkleRoot)
	return fullMerkleTree, txWithProofs, outStates
}

// generateGenesisBlocks generates genesis block for each shard.
func generateGenesisBlocks() (map[shard.Index]*wire.MsgBlock, map[shard.Index]*merkle.FullMerkleTree) {
	blockMap := make(map[shard.Index]*wire.MsgBlock)
	merkleTreeMap := make(map[shard.Index]*merkle.FullMerkleTree)
	// The number of blocks is corresponding to the number of shards.
	for _, shard := range shard.ShardList {
		outsMerkleTree, transactions, outs := generateOutsMerkleTree(shard)
		outsMerkleRoot := outsMerkleTree.MerkleRoot
		// Only one block in this shard, so ledger tree is equal to outs tree.
		shardLedgerMerkleRoot := outsMerkleRoot
		// Only one block
		// ledgerInfo: [ shard: 1 ]
		ledgerInfo := wire.LedgerInfo{}
		ledgerInfo.SetShardHeight(shard, 1)
		// Make block body.
		body := &wire.BlockBody{
			LedgerInfo:   ledgerInfo,
			Transactions: transactions,
			Outs:         outs,
		}
		bodyHash := chainhash.Hash(chainhash.HashH(body.ToBytesArray()))

		// Generate block for this shard.
		genesisBlock := &wire.MsgBlock{
			Header: wire.BlockHeader{
				Version:               wire.BlockVersion,
				ShardIndex:            shard,
				Height:                1,
				PrevBlockHeader:       chainhash.Hash{},
				BlockBodyHash:         bodyHash,
				OutsMerkleRoot:        *outsMerkleRoot,
				ShardLedgerMerkleRoot: *shardLedgerMerkleRoot,
			},
			Body: body,
		}
		blockMap[shard] = genesisBlock
		merkleTreeMap[shard] = outsMerkleTree
	}
	return blockMap, merkleTreeMap
}

// GenesisBlocks contains all shards' genesis blocks
// In each genesis blocks, there is NumberOfOutsInEachGenesisBlock outs. However they
// have none transaction. Outs are directly added into merkle trees, and the roots of
// the merkle trees are added into genesis blocks.
// GenesisBlockOutsMerkleTrees are the merkle trees that contains outs in genesis blocks.
// When building ledger merkle tree, don't directly use outs merkle tree. Outs merkle
// trees are second layer tree. Create a merkle tree with the OutsMerkleRoot in blockheader,
// then hook a outs merkle tree. For example:
//   block := GenesisBlocks[shard]
//   tree := merkle.NewFullMerkleTree(&block.Header.OutsMerkleRoot)
//   outsMerkleTree := GenesisBlocksOutsMerkleTrees[shard]
//   tree.HookSecondLayerTree(outsMerkleTree)
var GenesisBlocks map[shard.Index]*wire.MsgBlock

// GenesisBlocksOutsMerkleTrees contains all shards' out merkle tree
var GenesisBlocksOutsMerkleTrees map[shard.Index]*merkle.FullMerkleTree

func init() {
	GenesisBlocks, GenesisBlocksOutsMerkleTrees = generateGenesisBlocks()
}
