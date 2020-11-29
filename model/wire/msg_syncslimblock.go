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
	// MsgSyncSlimBlockMaxPayload is a very arbitrary number
	MsgSyncSlimBlockMaxPayload = MaxBlockPayload
)

// MsgSyncSlimBlock is a type of message sync block.
type MsgSyncSlimBlock struct {
	ReqShardIdx shard.Index
	SlimBlock   *SlimBlock
}

// NewMsgSyncSlimBlock creates a new messagesyncblock.
func NewMsgSyncSlimBlock(s shard.Index, block *SlimBlock) *MsgSyncSlimBlock {
	msg := MsgSyncSlimBlock{}
	msg.ReqShardIdx = s
	msg.SlimBlock = block
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgSyncSlimBlock) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgSyncSlimBlock) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgSyncSlimBlock) Command() string {
	return CmdSyncSlimBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgSyncSlimBlock) MaxPayloadLength(uint32) uint32 {
	return MsgSyncSlimBlockMaxPayload
}

// GetShardIndex returns the shardIndex.
func (msg *MsgSyncSlimBlock) GetShardIndex() shard.Index {
	return msg.ReqShardIdx
}