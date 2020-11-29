// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MsgReturnInit defines a type of message return init.
type MsgReturnInit struct {
	ShardIndex   shard.Index
	MsgID        uint32
	Ledger       LedgerInfo
	RightPath    merkle.MerklePath
	LatestHeader BlockHeader
	ShardHeight  int64
	TreeSize     int64
	Deposits     []*OutWithProof
}

// BtcDecode decode the message.
func (msg *MsgReturnInit) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgReturnInit) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
// TODO(zz)
func (msg *MsgReturnInit) Command() string {
	return CmdReturnInit
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
// TODO?
func (msg *MsgReturnInit) MaxPayloadLength(pver uint32) uint32 {
	return MaxReturnedMsgsPayload
}

// GetShardIndex returns the shardIndex.
func (msg *MsgReturnInit) GetShardIndex() shard.Index {
	return msg.ShardIndex
}
