// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// ClipTreeData 矿工将重建剪枝树所需要的必要信息打包到SlimBlock数据结构发送给指定的存储节点.
type ClipTreeData struct {
	Size            int64
	StartIndex      int64
	IsLeftOutNil    bool
	IsRightOutNil   bool
	Outs            []*OutState
	LeftMerklePath  *merkle.MerklePath
	RightMerklePath *merkle.MerklePath
}

// SlimBlock defines the data structure of slimblock.
type SlimBlock struct {
	ToShard        shard.Index
	Header         BlockHeader
	ClipTreeData   *ClipTreeData
	SmartContracts []*SmartContract
	UpdateActions  []*UpdateAction
	Transactions   []*MsgTxWithProofs
	LedgerInfo     LedgerInfo
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
// See Deserialize for decoding blocks stored to disk, such as in a database, as
// opposed to decoding blocks from the wire.
func (m *SlimBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
// See Serialize for encoding blocks to be stored to disk, such as in a
// database, as opposed to encoding blocks for the wire.
func (m *SlimBlock) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	newMsg := *m
	return rlp.Encode(w, newMsg)
}

// Command returns the protocol command string for the message.
func (m *SlimBlock) Command() string {
	return CmdSlimBlock
}

// GetShardIndex returns the shardIndex.
func (m *SlimBlock) GetShardIndex() shard.Index {
	return m.ToShard
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *SlimBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

func getFromEmptyBlock(block *MsgBlock) ([]*SlimBlock, error) {
	slimBlocks := make([]*SlimBlock, 0)
	for _, shard := range shard.ShardList {
		s := &SlimBlock{
			ToShard: shard,
			Header:  block.Header,
			ClipTreeData: &ClipTreeData{
				Size:            0,
				StartIndex:      -1,
				Outs:            []*OutState{},
				IsLeftOutNil:    true,
				IsRightOutNil:   true,
				LeftMerklePath:  &merkle.MerklePath{},
				RightMerklePath: &merkle.MerklePath{},
			},
			SmartContracts: block.Body.SmartContracts,
			UpdateActions:  []*UpdateAction{},
		}
		if s.ToShard == s.Header.ShardIndex {
			s.UpdateActions = block.Body.UpdateActions
		}
		slimBlocks = append(slimBlocks, s)
	}
	return slimBlocks, nil
}

// GetSlimBlocksFromBlock 矿工将需要打包到SlimBlock的信息从一个块中提取出来.
func GetSlimBlocksFromBlock(block *MsgBlock) ([]*SlimBlock, error) {
	if block.Header.IsEmptyBlock {
		return getFromEmptyBlock(block)
	}
	slimBlocks := make([]*SlimBlock, 0)
	var leftMerklePath *merkle.MerklePath
	var rightMerklePath *merkle.MerklePath
	var startIndex int64
	var outs []*OutState
	var err error
	size := int64(len(block.Body.Outs))
	var outHashs []*merkle.MerkleHash
	for _, outState := range block.Body.Outs {
		outStateHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
		outHashs = append(outHashs, outStateHash)
	}
	fullMerkleTree := merkle.NewFullMerkleTree(outHashs...)

	var isLeftNil bool
	var isRightNil bool
	var p int
	for _, shard := range shard.ShardList {
		outs = make([]*OutState, 0)
		for p < len(block.Body.Outs) && block.Body.Outs[p].Shard < shard {
			p++
		}
		if p != 0 {
			startIndex = int64(p - 1)
			isLeftNil = false
			outs = append(outs, block.Body.Outs[p-1])
		} else {
			startIndex = int64(p)
			isLeftNil = true
		}
		for p < len(block.Body.Outs) && block.Body.Outs[p].Shard == shard {
			outs = append(outs, block.Body.Outs[p])
			p++
		}
		if p != len(block.Body.Outs) {
			isRightNil = false
			outs = append(outs, block.Body.Outs[p])
		} else {
			isRightNil = true
		}
		leftMerklePath, err = fullMerkleTree.GetMerklePath(merkle.ComputeMerkleHash(outs[0].ToBytesArray()))
		if err != nil {
			return nil, err
		}
		rightMerklePath, err = fullMerkleTree.GetMerklePath(merkle.ComputeMerkleHash(outs[len(outs)-1].ToBytesArray()))
		if err != nil {
			return nil, err
		}
		s := &SlimBlock{
			ToShard: shard,
			Header:  block.Header,
			ClipTreeData: &ClipTreeData{
				Size:            size,
				StartIndex:      startIndex,
				Outs:            outs,
				IsLeftOutNil:    isLeftNil,
				IsRightOutNil:   isRightNil,
				LeftMerklePath:  leftMerklePath,
				RightMerklePath: rightMerklePath,
			},
			SmartContracts: block.Body.SmartContracts,
			UpdateActions:  []*UpdateAction{},
			Transactions:   []*MsgTxWithProofs{},
			LedgerInfo:     LedgerInfo{},
		}

		if s.ToShard == s.Header.ShardIndex {
			s.UpdateActions = block.Body.UpdateActions
			s.Transactions = block.Body.Transactions
			s.LedgerInfo = block.Body.LedgerInfo
		}
		slimBlocks = append(slimBlocks, s)
	}

	return slimBlocks, nil
}

// GetSlimBlockFromBlockByShard 根据分片以及block返回剪枝后的SlimBlock
func GetSlimBlockFromBlockByShard(block *MsgBlock, shardIdx shard.Index) (*SlimBlock, error) {
	slimBlocks, err := GetSlimBlocksFromBlock(block)
	if err != nil {
		return nil, err
	}

	if int(shardIdx) >= len(slimBlocks) {
		return nil, fmt.Errorf("illegal shard index ")
	}

	slimBlock := slimBlocks[shardIdx]
	if slimBlock.ToShard != shardIdx {
		return nil, fmt.Errorf("failed to GetSlimBlockFromBlockByShard, want shardIdx: %d,got shardIdx: %d",
			shardIdx, slimBlock.ToShard)
	}
	return slimBlock, nil
}

func (clipTreeData *ClipTreeData) verifyIntegrity(shard shard.Index) error {
	for index, out := range clipTreeData.Outs {
		if (index == 0 && !clipTreeData.IsLeftOutNil && out.Shard >= shard) || (index == len(clipTreeData.Outs)-1 && !clipTreeData.IsRightOutNil && out.Shard <= shard) {
			return fmt.Errorf("clip tree data has wrong outs")
		}
		if out.Shard != shard {
			if !(index == 0 && !clipTreeData.IsLeftOutNil) && !(index == len(clipTreeData.Outs)-1 && !clipTreeData.IsRightOutNil) {
				return fmt.Errorf("clip tree data has wrong outs")
			}
		}
		if index != 0 && index != len(clipTreeData.Outs)-1 && out.Shard != shard {
			return fmt.Errorf(". clip tree data has wrong outs")
		}
	}
	return nil
}

// Rebuild storage node rebuilds the merkle tree by the given clipTreeData.
func (clipTreeData *ClipTreeData) Rebuild(root merkle.MerkleHash, shard shard.Index) (*merkle.FullMerkleTree, error) {
	err := clipTreeData.verifyIntegrity(shard)
	if err != nil {
		return nil, err
	}
	var fullMerkleTree *merkle.FullMerkleTree
	leaves := make([]*merkle.MerkleHash, 0)
	for i := 0; i < len(clipTreeData.Outs); i++ {
		leafHash := merkle.ComputeMerkleHash(clipTreeData.Outs[i].ToBytesArray())
		leaves = append(leaves, leafHash)
	}
	fullMerkleTree, err = merkle.NewClippedMerkleTree(clipTreeData.Size, clipTreeData.StartIndex, int64(len(leaves)), leaves, clipTreeData.LeftMerklePath, clipTreeData.RightMerklePath)
	if err != nil {
		return nil, fmt.Errorf("rebuild clip tree error : %v", err)
	}
	if *fullMerkleTree.MerkleRoot != root {
		return nil, fmt.Errorf("rebuilt clip tree root is not equal to the root in header")
	}
	return fullMerkleTree, nil
}
