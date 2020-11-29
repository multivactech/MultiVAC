// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MaxReturnedMsgsPayload is the max payload for returned msgs from storagenode is set to be at the maximum of 20 blocks' size.
const MaxReturnedMsgsPayload = MaxBlockPayload * 20

// MsgReturnTxs is a message to return a list of transactions (with proofs) and smart contracts from storage node to
// miner node.
//
// For each transaction, if a smart contract is included, MsgReturnTxs will return the smart contract as well.
type MsgReturnTxs struct {
	MsgID              uint32
	ShardIndex         shard.Index
	Txs                []*MsgTxWithProofs
	SmartContractInfos []*SmartContractInfo
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgReturnTxs) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgReturnTxs) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgReturnTxs) Command() string {
	return CmdReturnTxs
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgReturnTxs) MaxPayloadLength(pver uint32) uint32 {
	return MaxReturnedMsgsPayload
}

// GetShardIndex returns the shardIndex.
func (msg *MsgReturnTxs) GetShardIndex() shard.Index {
	return msg.ShardIndex
}
