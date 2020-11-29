// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// MsgFetchTxs is a message to request transactions from storage node.
type MsgFetchTxs struct {
	MsgID       uint32
	NumberOfTxs int
	ShardIndex  shard.Index
	ExistTx     []uint32
}

// BtcDecode decode the message.
func (m *MsgFetchTxs) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgFetchTxs) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgFetchTxs) Command() string {
	return CmdFetchTxs
}

// GetShardIndex returns the shardIndex.
func (m *MsgFetchTxs) GetShardIndex() shard.Index {
	return m.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgFetchTxs) MaxPayloadLength(pver uint32) uint32 {
	// 10k. In theory this message is very small.
	return 10240
}
