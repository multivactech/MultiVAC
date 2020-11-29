// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// MsgStartNet ???
type MsgStartNet struct {
}

// BtcDecode decode the message.
func (m *MsgStartNet) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encode the message.
func (m *MsgStartNet) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the command string.
func (m *MsgStartNet) Command() string {
	return CmdStartNet
}

// MaxPayloadLength return the mac playload length of MsgStartNet.
func (m *MsgStartNet) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}
