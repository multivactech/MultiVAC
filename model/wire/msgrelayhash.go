// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// RelayHashMsg is a type of relay hash message.
type RelayHashMsg struct {
	Relay   []string
	Request []string
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
// TODO(jylu)
func (msg *RelayHashMsg) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// Deserialize will deserialize the message.
// TODO(jylu)
func (msg *RelayHashMsg) Deserialize(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
//TODO(jylu)
func (msg *RelayHashMsg) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Serialize encodes the block to w.
// TODO(jylu)
func (msg *RelayHashMsg) Serialize(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *RelayHashMsg) Command() string {
	return CmdRelayHash
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *RelayHashMsg) MaxPayloadLength(pver uint32) uint32 {
	return LargeMaxPayloadLength
}
