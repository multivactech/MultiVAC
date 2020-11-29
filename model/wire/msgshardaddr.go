// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// MsgGetShardAddr defines the data structure of massage that get shard address.
type MsgGetShardAddr struct {
	ShardIndex []shard.Index
}

// BtcDecode decode the message.
func (m *MsgGetShardAddr) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgGetShardAddr) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgGetShardAddr) Command() string {
	return CmdGetShardAddr
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgGetShardAddr) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// MsgReturnShardAddr defines the data structure of message return shard address.
type MsgReturnShardAddr struct {
	Address []string
}

// BtcDecode decode the message.
func (m *MsgReturnShardAddr) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgReturnShardAddr) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgReturnShardAddr) Command() string {
	return CmdReturnAddr
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgReturnShardAddr) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}
