// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"math/big"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// it seems unused.
//var blockOneData = struct{ Value *big.Int }{Value: big.NewInt(0x12a05f200)}

// it seems unused.
//var blockOneDataBytes, _ = rlp.EncodeToBytes(blockOneData)

var blockOne = MsgBlock{
	Header: BlockHeader{
		Version: 1,
		PrevBlockHeader: chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
			0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
			0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
			0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
			0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
		}),
		DepositTxsOuts:  []OutPoint{},
		WithdrawTxsOuts: []OutPoint{},
		Pk:              []uint8{},
		LastSeedSig:     []uint8{},
		HeaderSig:       []uint8{},
		ReshardSeed:     []uint8{},
		TimeStamp:       0,
		ReduceActions:   []*ReduceAction{},
	},
	Body: &BlockBody{
		Transactions: []*MsgTxWithProofs{
			{
				Tx: MsgTx{
					Version: 1,
					TxIn: []*TxIn{
						{
							PreviousOutPoint: OutPoint{
								TxHash:          chainhash.Hash{},
								Index:           0xffffffff,
								Data:            MtvValueToData(big.NewInt(0x12a05f200)),
								UserAddress:     multivacaddress.Address{},
								ContractAddress: multivacaddress.Address{},
							},
						},
					},
					Params:             []byte{},
					SignatureScript:    []uint8{},
					ContractAddress:    multivacaddress.Address{},
					PublicKey:          []byte{},
					StorageNodeAddress: multivacaddress.Address{},
				},
				Proofs: []merkle.MerklePath{},
			},
		},
		LedgerInfo: LedgerInfo{
			ShardsHeights: []*shard.IndexAndHeight{},
		},
		SmartContracts: []*SmartContract{},
		Outs:           []*OutState{},
		UpdateActions:  []*UpdateAction{},
	},
}

var blockTwo = SlimBlock{
	Header: BlockHeader{
		Version: 1,
		PrevBlockHeader: chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
			0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
			0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
			0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
			0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
		}),
		DepositTxsOuts:  []OutPoint{},
		WithdrawTxsOuts: []OutPoint{},
		Pk:              []uint8{},
		LastSeedSig:     []uint8{},
		HeaderSig:       []uint8{},
		ReshardSeed:     []uint8{},
		TimeStamp:       0,
		ReduceActions:   []*ReduceAction{},
	},
	ClipTreeData: &ClipTreeData{
		LeftMerklePath:  &merkle.MerklePath{},
		RightMerklePath: &merkle.MerklePath{},
	},
}
