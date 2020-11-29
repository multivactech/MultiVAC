// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/base/rlp"
)

const (
	// MsgSyncBlockMaxPayload is a very arbitrary number
	MsgSyncBlockMaxPayload = MaxBlockPayload
)

// MsgSyncBlock is a type of message sync block.
type MsgSyncBlock struct {
	ReqShardIdx shard.Index
	Block       *MsgBlock
}

// NewMsgSyncBlock creates a new messagesyncblock.
func NewMsgSyncBlock(s shard.Index, block *MsgBlock) *MsgSyncBlock {
	msg := MsgSyncBlock{}
	msg.ReqShardIdx = s
	msg.Block = block
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgSyncBlock) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgSyncBlock) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgSyncBlock) Command() string {
	return CmdSyncBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgSyncBlock) MaxPayloadLength(uint32) uint32 {
	return MsgSyncBlockMaxPayload
}

// GetShardIndex returns the shardIndex.
func (msg *MsgSyncBlock) GetShardIndex() shard.Index {
	return msg.ReqShardIdx
}