/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package smartcontractdatastore

import (
	"crypto/sha256"
	"math/big"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func generateOutsMerkleTree(shard shard.Index) *merkle.FullMerkleTree {
	var outsHashes []*merkle.MerkleHash

	tx := GenerateGenesisTx(shard)

	txHash := tx.TxHash()

	outStates := CreateDeploySmartContractGenesisOutState(shard, &txHash)

	for _, out := range outStates {
		outsHash := merkle.MerkleHash(sha256.Sum256(out.ToBytesArray()))
		outsHashes = append(outsHashes, &outsHash)
	}

	fullMerkleTree := merkle.NewFullMerkleTree(outsHashes...)

	return fullMerkleTree
}

func setTotalAmountOfEachShard() {
	var success bool
	TotalAmountOfEachShard, success = new(big.Int).SetString("100000000000000000000000000000", 10)
	if !success {
		panic("generate TotalAmountOfEachShard err!")
	}
}

// GenerateGenesisTx returns a new instance of MsgTx using the given shardIndex.
func GenerateGenesisTx(shard shard.Index) *wire.MsgTx {
	setTotalAmountOfEachShard()
	return NewMsgTx(wire.TxVersion, shard)
}

// NewMsgTx returns a new instance of MsgTx using the given version and shardIndex.
func NewMsgTx(version int32, shard shard.Index) *wire.MsgTx {
	return &wire.MsgTx{
		Shard:           shard,
		Version:         version,
		TxIn:            []*wire.TxIn{},
		ContractAddress: isysapi.SysAPIAddress,
		API:             "deploy",
		Params:          []byte{},
		SignatureScript: []byte{},
	}
}

// CreateDeploySmartContractGenesisOutState creates some OutStates using shard and txHash.
func CreateDeploySmartContractGenesisOutState(shard shard.Index, txHash *chainhash.Hash) []*wire.OutState {
	setTotalAmountOfEachShard()

	genesisAddr := multivacaddress.GenerateAddress(GenesisPublicKeys[shard], multivacaddress.UserAddress)
	data := wire.MTVCoinData{
		Value: new(big.Int).Div(TotalAmountOfEachShard, big.NewInt(int64(NumberOfOutsInEachGenesisBlock))),
	}
	dataBytes, _ := rlp.EncodeToBytes(data)
	var outStates []*wire.OutState
	var outState *wire.OutState
	sc := wire.SmartContract{}

	sc.ContractAddr = txHash.FormatSmartContractAddress()
	sc.APIList = []string{}
	sc.Code = []byte{}
	scHash := sc.SmartContractHash()
	scBytes := scHash.CloneBytes()

	outState = &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          *txHash,
			Index:           0,
			Shard:           shard,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shard], multivacaddress.UserAddress),
			ContractAddress: sc.ContractAddr,
			Data:            scBytes,
		},
		State: wire.StateUnused,
	}
	outStates = append(outStates, outState)

	outState = &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          *txHash,
			Index:           1,
			Shard:           shard,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shard], multivacaddress.UserAddress),
			ContractAddress: sc.ContractAddr,
			Data:            []byte{},
		},
		State: wire.StateUnused,
	}
	outStates = append(outStates, outState)

	outState = &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:      *txHash,
			Index:       2,
			Shard:       shard,
			UserAddress: genesisAddr,
			Data:        dataBytes,
		},
		State: wire.StateUnused,
	}
	outStates = append(outStates, outState)

	return outStates
}

// func convertToPkHash(pk signature.PublicKey) chainhash.Hash {
// 	tpkBytes := []byte(pk)
// 	var pkBytes [chainhash.HashSize]byte
// 	copy(pkBytes[:], tpkBytes[:])
// 	return chainhash.Hash(pkBytes)
// }

// TotalAmountOfEachShard is defined as *big.Int.
var TotalAmountOfEachShard *big.Int

// GenesisPublicKeys is a map whose key is Index and value is pk.
var GenesisPublicKeys = publicKeys()

func privateKeys() map[shard.Index]signature.PrivateKey {
	keys := make(map[shard.Index]signature.PrivateKey)
	keys[shard.ShardList[0]] = signature.PrivateKey{
		158, 139, 132, 23, 249, 119, 67, 251,
		173, 194, 184, 163, 121, 5, 133, 141,
		123, 103, 187, 55, 104, 147, 54, 44,
		20, 47, 78, 40, 15, 112, 88, 125,
		1, 97, 158, 26, 138, 75, 21, 208,
		187, 249, 216, 252, 33, 7, 115, 33,
		146, 121, 3, 59, 119, 31, 54, 227,
		216, 51, 81, 75, 46, 177, 200, 223,
	}
	keys[shard.ShardList[1]] = signature.PrivateKey{
		79, 199, 93, 238, 92, 239, 238, 93,
		189, 17, 6, 86, 44, 100, 239, 137,
		12, 210, 39, 106, 254, 184, 123, 14,
		253, 75, 77, 223, 61, 236, 146, 101,
		69, 31, 137, 15, 136, 151, 178, 134,
		25, 164, 33, 136, 246, 52, 191, 231,
		192, 210, 201, 97, 23, 50, 73, 138,
		54, 152, 25, 5, 71, 44, 206, 24,
	}
	keys[shard.ShardList[2]] = signature.PrivateKey{
		75, 76, 92, 202, 188, 6, 151, 111,
		164, 231, 68, 83, 108, 243, 96, 41,
		109, 248, 165, 12, 190, 108, 215, 246,
		85, 162, 70, 121, 3, 22, 41, 127,
		186, 158, 175, 142, 225, 84, 115, 68,
		141, 76, 235, 233, 142, 237, 65, 166,
		191, 13, 108, 45, 94, 156, 165, 9,
		3, 4, 2, 251, 148, 150, 238, 115,
	}
	keys[shard.ShardList[3]] = signature.PrivateKey{
		245, 123, 199, 133, 167, 253, 6, 166,
		198, 210, 103, 235, 91, 123, 225, 219,
		196, 4, 157, 150, 41, 40, 239, 133,
		219, 102, 182, 149, 250, 33, 146, 158,
		232, 20, 163, 58, 76, 34, 65, 118,
		143, 106, 162, 95, 177, 216, 248, 226,
		52, 179, 123, 245, 187, 44, 110, 219,
		140, 213, 112, 90, 105, 208, 101, 116,
	}

	return keys
}

// NumberOfOutsInEachGenesisBlock is a int value for test.
var NumberOfOutsInEachGenesisBlock = 3

// GenesisPrivateKeys is a map whose key is Index and value is privateKey.
var GenesisPrivateKeys = privateKeys()

func publicKeys() map[shard.Index]signature.PublicKey {
	keys := make(map[shard.Index]signature.PublicKey)
	for _, shard := range shard.ShardList {
		keys[shard] = GenesisPrivateKeys[shard].Public()
	}
	return keys
}
