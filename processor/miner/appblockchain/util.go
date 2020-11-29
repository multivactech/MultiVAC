/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/pos/depositpool"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// getFullMerkleTreeByBlock build a full merkle tree base on the info of block.
func getFullMerkleTreeByBlock(block *wire.MsgBlock) *merkle.FullMerkleTree {
	var outHashes []*merkle.MerkleHash
	for _, out := range block.Body.Outs {
		outStateHash := merkle.ComputeMerkleHash(out.ToBytesArray())
		outHashes = append(outHashes, outStateHash)
	}
	fullMerkleTree := merkle.NewFullMerkleTree(outHashes...)
	return fullMerkleTree
}

// getOutProofByUpdate can get a OutPoint's merkle path by update and a block merkle tree.
//
// tree: block merkle tree
// update: an update of a ledger
//
//          |--  root
// topTree  |    /  \
//          |-- b1  b2    --|
//             / \  / \     |  blockTree
//            o1 o2 o3 o4 --|
// For example, if we want to get the full merkle path of o2, we need to get a
// part from blockTree and get another part from topTree, then merger the two parts.
// topTree can get from update and blockTree is parameter.
func getOutProofByUpdate(update *state.Update, btree *merkle.FullMerkleTree, out *wire.OutState) (*merkle.MerklePath, error) {
	outHash := merkle.ComputeMerkleHash(out.ToBytesArray())

	// Get path from blockTree
	proofInBlock, err := btree.GetMerklePath(outHash)
	if err != nil {
		return nil, err
	}

	// Get another path from topTree
	topProof, err := update.DeltaTree.GetMerklePath(btree.MerkleRoot)
	if err != nil {
		return nil, err
	}
	if len(topProof.Hashes) == 0 {
		return nil, fmt.Errorf("deltatree get merkle path error")
	}

	// The first index of topProof is the same as the last index of proofInBlock
	// egg:
	// o2's block path is [o2,o1,b1], o2's top path is [b1,b2,root]
	// so o2's full path is [o2,o1,b1,root]
	//
	// Remove duplicate values
	topProof.Hashes = topProof.Hashes[1:]
	// Merge the two part of the path
	proofInBlock.Hashes = append(proofInBlock.Hashes, topProof.Hashes...)
	proofInBlock.ProofPath = append(proofInBlock.ProofPath, topProof.ProofPath...)

	return proofInBlock, nil
}

func createRewardTx(shard shard.Index, height int64, addr multivacaddress.Address) *wire.MsgTxWithProofs {
	var nonce = uint64(shard)
	nonce = nonce << 32
	nonce += uint64(height)

	rp := isysapi.RewardParams{

		To:     addr,
		Amount: isysapi.MinerRewardAmount,
	}
	params, err := rlp.EncodeToBytes(rp)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal reward params %v, err msg %v", rp, err))
	}

	return &wire.MsgTxWithProofs{
		Tx: wire.MsgTx{
			Shard:           shard,
			ContractAddress: isysapi.SysAPIAddress,
			API:             isysapi.SysAPIReward,
			Params:          params,
			Nonce:           nonce,
		},
	}
}

// Hash the reshardseed and the current height to calculate the seed of the next round of resharding.
func calcReshardSeedHash(reshardSeed []byte, height int64) chainhash.Hash {
	int64Size := 8
	seedBuf := make([]byte, len(reshardSeed)+int64Size)
	copy(seedBuf, reshardSeed[0:])
	binary.PutVarint(seedBuf[len(reshardSeed):], height)
	reshardSeedHash := sha256.Sum256(seedBuf)
	return reshardSeedHash
}

// updateReshardSeed Generate new ReshardSeed using VRF, pk, sk
func genNextReshardSeed(pk []byte, sk []byte, reshardSeed []byte, height int64) ([]byte, error) {
	lastReshardHash := calcReshardSeedHash(reshardSeed, height)
	nextReshardSeed, err := vrf.Ed25519VRF{}.Generate(pk, sk, lastReshardHash[:])
	if err != nil {
		return nil, err
	}
	return nextReshardSeed, nil
}

func getStorageReward(txs []*wire.MsgTxWithProofs, height int64, shardIndex shard.Index) []*wire.MsgTxWithProofs {
	rewardTx := make([]*wire.MsgTxWithProofs, 0)
	reward := make(map[string]*big.Int)

	for _, tx := range txs {
		if tx.Tx.IsRewardTx() || len(tx.Tx.StorageNodeAddress) == 0 {
			continue
		}
		address := hex.EncodeToString(tx.Tx.StorageNodeAddress)
		tmp, ok := reward[address]
		if !ok {
			tmp = big.NewInt(0)
		}
		reward[address] = new(big.Int).Add(tmp, isysapi.StorageRewardAmount)
	}
	for to, val := range reward {
		address, _ := hex.DecodeString(to)
		rp := isysapi.RewardParams{
			To:     address,
			Amount: val,
		}

		params, err := rlp.EncodeToBytes(rp)
		if err == nil {
			tx := &wire.MsgTxWithProofs{
				Tx: wire.MsgTx{
					Shard:           shardIndex,
					ContractAddress: isysapi.SysAPIAddress,
					API:             isysapi.SysAPIReward,
					Params:          params,
				},
			}
			rewardTx = append(rewardTx, tx)
		}
	}
	return rewardTx
}

// getRewardAddress is used to get reward address for this miner
// TODO:(zz) maybe change in the furture
func (abc *appBlockChain) getRewardAddress() (multivacaddress.Address, error) {
	data, err := abc.dPool.GetBiggest(abc.getAddress())
	if err != nil {
		return nil, err
	}

	depositInfo := &depositpool.DepositInfo{}
	err = rlp.DecodeBytes(data, depositInfo)
	if err != nil {
		return nil, err
	}

	return depositInfo.OutPoint.UserAddress, nil
}
