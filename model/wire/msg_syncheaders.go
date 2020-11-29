// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/shard"
	"io"
)

const (
	// MsgSyncHeadersMaxPayload is a very arbitrary number
	MsgSyncHeadersMaxPayload = MaxBlockPayload
)

// MsgSyncHeaders is used for passing back the requested header content during sync for a given shard.
type MsgSyncHeaders struct {
	ReqShardIdx shard.Index
	Headers     []*BlockHeader
}

// NewMsgSyncHeaders creates a new MsgSyncHeaders.
func NewMsgSyncHeaders(s shard.Index, headers []*BlockHeader) *MsgSyncHeaders {
	msg := MsgSyncHeaders{}
	msg.ReqShardIdx = s
	msg.Headers = headers
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgSyncHeaders) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgSyncHeaders) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgSyncHeaders) Command() string {
	return CmdSyncHeaders
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgSyncHeaders) MaxPayloadLength(uint32) uint32 {
	return MsgSyncHeadersMaxPayload
}

// GetShardIndex returns the shardIndex.
func (msg *MsgSyncHeaders) GetShardIndex() shard.Index {
	return msg.ReqShardIdx
}