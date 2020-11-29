// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"github.com/multivactech/MultiVAC/model/shard"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

const (
	// MsgSyncReqMaxPayload is a very arbitrary number
	MsgSyncReqMaxPayload = 4000000

	// DefaultShardCapacity defines the default shard capacity.
	// TODO(huangsz): Make this to be loaded from genesis
	DefaultShardCapacity = 64

	// ReqSyncToLatest defines whether request to the last.
	ReqSyncToLatest = -1
)

// MsgSyncReq is a message to request sync from another node.
type MsgSyncReq struct {
	// The shard we request sync for.
	ReqShardIdx shard.Index

	// The locator of requested blocks/headers for each shard.
	// Inclusive on both sides, aka: [fromHgt, toHgt]
	Locators []*BlockLocator
}

func (msg *MsgSyncReq) String() string {
	return fmt.Sprintf("MsgSyncReq {ReqShard: %v, Locators: [%v]}", msg.ReqShardIdx.GetID(), msg.Locators)
}

// BlockLocator defines the data structure of block located.
type BlockLocator struct {
	ShardIdx   shard.Index
	FromHeight int64
	ToHeight   int64
}

func (msg *BlockLocator) String() string {
	return fmt.Sprintf("BlockLocator {Shard: %v, fromHgt: %v, toHgt: %v}", msg.ShardIdx.GetID(), msg.FromHeight, msg.ToHeight)
}

// NewMsgSyncReq create a new message MsgSyncReq.
func NewMsgSyncReq(shardIdx shard.Index) *MsgSyncReq {
	msg := MsgSyncReq{}
	msg.ReqShardIdx = shardIdx
	msg.Locators = make([]*BlockLocator, 0, DefaultShardCapacity)
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgSyncReq) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgSyncReq) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgSyncReq) Command() string {
	return CmdSyncReq
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgSyncReq) MaxPayloadLength(uint32) uint32 {
	return MsgSyncReqMaxPayload
}

// AddBlockLocatorToRecent add a recent blocklocator.
func (msg *MsgSyncReq) AddBlockLocatorToRecent(shardIdx shard.Index, fromHeight int64) {
	msg.Locators = append(msg.Locators, NewBlockLocator(shardIdx, fromHeight, ReqSyncToLatest))
}

// AddBlockLocator add a blocklocator with given height.
func (msg *MsgSyncReq) AddBlockLocator(shardIdx shard.Index, fromHeight int64, toHeight int64) {
	msg.Locators = append(msg.Locators, NewBlockLocator(shardIdx, fromHeight, toHeight))
}

// NewBlockLocator creates a blocklocator.
func NewBlockLocator(shardIdx shard.Index, fromHeight int64, toHeight int64) *BlockLocator {
	return &BlockLocator{
		ShardIdx:   shardIdx,
		FromHeight: fromHeight,
		ToHeight:   toHeight,
	}
}

// GetShardIndex returns the shardIndex.
func (msg *MsgSyncReq) GetShardIndex() shard.Index {
	return msg.ReqShardIdx
}