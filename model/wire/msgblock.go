// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"

	"github.com/prometheus/common/log"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// BlockVersion defines the version of block.
const BlockVersion = 1

// MaxBlocksPerMsg is the maximum number of blocks allowed per message.
const MaxBlocksPerMsg = 500

// MaxBlockPayload is the maximum bytes a block message can be in bytes.
// After Segregated Witness, the max block payload has been raised to 4MB.
const MaxBlockPayload = 4000000

// BlockHeight defines the data structure of block height.
type BlockHeight int64

// LedgerInfo defines the data structure of ledger info.
type LedgerInfo struct {
	// How many blocks has been included in the ledger.
	Size int64

	ShardsHeights []*shard.IndexAndHeight
}

// GetShardHeight returns the height for a certain shard. If there's no info for this shard, then return 0.
func (ledger *LedgerInfo) GetShardHeight(shardIndex shard.Index) int64 {
	shardIndexAndHeight := ledger.findShardIndexAndHeight(shardIndex)
	if shardIndexAndHeight == nil {
		return 0
	}
	return shardIndexAndHeight.Height
}

// SetShardHeight set the shardheight.
func (ledger *LedgerInfo) SetShardHeight(shardIndex shard.Index, height int64) {
	shardIndexAndHeight := ledger.findShardIndexAndHeight(shardIndex)
	if shardIndexAndHeight != nil {
		shardIndexAndHeight.Height = height
	} else {
		newShardIndexAndHeight := &shard.IndexAndHeight{
			Index:  shardIndex,
			Height: height,
		}
		ledger.ShardsHeights = append(ledger.ShardsHeights, newShardIndexAndHeight)
		sort.SliceStable(ledger.ShardsHeights, func(i, j int) bool {
			return ledger.ShardsHeights[i].Index < ledger.ShardsHeights[j].Index
		})
	}
}

func (ledger *LedgerInfo) findShardIndexAndHeight(index shard.Index) *shard.IndexAndHeight {
	if ledger.ShardsHeights == nil {
		ledger.ShardsHeights = []*shard.IndexAndHeight{}
	}
	for _, shardIndexAndHeight := range ledger.ShardsHeights {
		if shardIndexAndHeight.Index == index {
			return shardIndexAndHeight
		}
	}
	return nil
}

// BlockBody defines the data structure of block body.
type BlockBody struct {
	Transactions   []*MsgTxWithProofs
	LedgerInfo     LedgerInfo
	SmartContracts []*SmartContract
	Outs           []*OutState
	UpdateActions  []*UpdateAction
}

// ToBytesArray will serialize the block body to bytes array.
func (body *BlockBody) ToBytesArray() []byte {
	ret, err := rlp.EncodeToBytes(body)
	if err != nil {
		return nil
	}
	return ret
}

// BlockBodyHash returns the block body hash.
func (body *BlockBody) BlockBodyHash() chainhash.Hash {
	return chainhash.HashH(body.ToBytesArray())
}

// MsgBlock implements the Message interface and represents a bitcoin
// block message.  It is used to deliver block and transaction information in
// response to a getdata message (MsgGetData) for a given block hash.
type MsgBlock struct {
	Header BlockHeader
	Body   *BlockBody
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
// See Deserialize for decoding blocks stored to disk, such as in a database, as
// opposed to decoding blocks from the wire.
func (m *MsgBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, m)
}

// Deserialize decodes a block from r into the receiver using a format that is
// suitable for long-term storage such as a database while respecting the
// Version field in the block.  This function differs from BtcDecode in that
// BtcDecode decodes from the bitcoin wire protocol as it was sent across the
// network.  The wire encoding can technically differ depending on the protocol
// version and doesn't even really need to match the format of a stored block at
// all.  As of the time this comment was written, the encoded block is the same
// in both instances, but there is a distinct difference and separating the two
// allows the API to be flexible enough to deal with changes.
func (m *MsgBlock) Deserialize(r io.Reader) error {
	// At the current time, there is no difference between the wire encoding
	// at protocol version 0 and the stable long-term storage format.  As
	// a result, make use of BtcDecode.
	//
	// Passing an encoding type of WitnessEncoding to BtcEncode for the
	// MessageEncoding parameter indicates that the transactions within the
	// block are expected to be serialized according to the new
	// serialization structure defined in BIP0141.
	return m.BtcDecode(r, 0, WitnessEncoding)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
// See Serialize for encoding blocks to be stored to disk, such as in a
// database, as opposed to encoding blocks for the wire.
func (m *MsgBlock) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	newMsg := *m
	return rlp.Encode(w, newMsg)
}

// Serialize encodes the block to w using a format that suitable for long-term
// storage such as a database while respecting the Version field in the block.
// This function differs from BtcEncode in that BtcEncode encodes the block to
// the bitcoin wire protocol in order to be sent across the network.  The wire
// encoding can technically differ depending on the protocol version and doesn't
// even really need to match the format of a stored block at all.  As of the
// time this comment was written, the encoded block is the same in both
// instances, but there is a distinct difference and separating the two allows
// the API to be flexible enough to deal with changes.
func (m *MsgBlock) Serialize(w io.Writer) error {
	// At the current time, there is no difference between the wire encoding
	// at protocol version 0 and the stable long-term storage format.  As
	// a result, make use of BtcEncode.
	//
	// Passing WitnessEncoding as the encoding type here indicates that
	// each of the transactions should be serialized using the witness
	// serialization structure defined in BIP0141.
	return m.BtcEncode(w, 0, WitnessEncoding)
}

// SerializeNoWitness encodes a block to w using an identical format to
// Serialize, with all (if any) witness data stripped from all transactions.
// This method is provided in additon to the regular Serialize, in order to
// allow one to selectively encode transaction witness data to non-upgraded
// peers which are unaware of the new encoding.
func (m *MsgBlock) SerializeNoWitness(w io.Writer) error {
	return m.BtcEncode(w, 0, BaseEncoding)
}

// SerializeSize returns the number of bytes it would take to serialize the block.
func (m *MsgBlock) SerializeSize() int {
	var buf bytes.Buffer
	err := m.BtcEncode(&buf, 0, BaseEncoding)
	if err != nil {
		log.Errorf("failed to encode message,err:%v", err)
		return 0
	}
	return buf.Len()
}

// SerializeSizeStripped returns the number of bytes it would take to serialize
// the block, excluding any witness data (if any).
func (m *MsgBlock) SerializeSizeStripped() int {
	return m.SerializeSize()
}

// Command returns the protocol command string for the message.
func (m *MsgBlock) Command() string {
	return CmdBlock
}

// GetShardIndex returns the shardIndex.
func (m *MsgBlock) GetShardIndex() shard.Index {
	return m.Header.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

func (m *MsgBlock) String() string {
	return fmt.Sprintf("{TransNum: %d, Header: %v, Ledger %v}",
		len(m.Body.Transactions),
		m.Header.String(),
		m.Body.LedgerInfo)
}

// NewBlock creates valid new block base given data.
// Note. This function should be updated when block struct changes.
func NewBlock(
	shardIndex shard.Index,
	height int64,
	prevBlockHeader chainhash.Hash,
	shardLedgerRoot merkle.MerkleHash,
	outsMerkleRoot merkle.MerkleHash,
	txs []*MsgTxWithProofs,
	ledgerInfo *LedgerInfo,
	isEmptyBlock bool,
	timeStamp int64,
	reshardSeed []byte,
	sc []*SmartContract,
	outs []*OutState,
	updateActions []*UpdateAction,
	reduceActions []*ReduceAction) *MsgBlock {
	body := BlockBody{
		Transactions:   txs,
		SmartContracts: sc,
		Outs:           outs,
		UpdateActions:  updateActions,
	}
	if ledgerInfo != nil {
		body.LedgerInfo = *ledgerInfo
	}

	block := &MsgBlock{
		Header: BlockHeader{
			Version:               BlockVersion,
			ShardIndex:            shardIndex,
			Height:                height,
			PrevBlockHeader:       prevBlockHeader,
			OutsMerkleRoot:        outsMerkleRoot,
			ShardLedgerMerkleRoot: shardLedgerRoot,
			BlockBodyHash:         chainhash.Hash(sha256.Sum256(body.ToBytesArray())),
			IsEmptyBlock:          isEmptyBlock,
			TimeStamp:             timeStamp,
			ReshardSeed:           reshardSeed,
			ReduceActions:         reduceActions,
		},
		Body: &body,
	}
	return block
}