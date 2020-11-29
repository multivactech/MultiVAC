// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"encoding/gob"
	"io"

	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// OutWithProof contains outpoint and it's merkle path
type OutWithProof struct {
	Out    *OutPoint
	Proof  *merkle.MerklePath
	Height BlockHeight
}

// MsgReturnDeposit is deposits data come from storagenode
type MsgReturnDeposit struct {
	ShardIndex shard.Index
	Deposits   []*OutWithProof
}

// BtcDecode decode the message.
func (msg *MsgReturnDeposit) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgReturnDeposit) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	encoder := gob.NewEncoder(w)
	return encoder.Encode(*msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgReturnDeposit) Command() string {
	return CmdReturnDeposit
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgReturnDeposit) MaxPayloadLength(pver uint32) uint32 {
	return MaxReturnedMsgsPayload
}
