// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MsgFetchSmartContractInfo is a message to request smart contract info from storage node.
type MsgFetchSmartContractInfo struct {
	MsgID        uint32
	ContractAddr multivacaddress.Address
	ShardIndex   shard.Index
}

// BtcDecode decode the message.
func (msg *MsgFetchSmartContractInfo) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgFetchSmartContractInfo) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgFetchSmartContractInfo) Command() string {
	return CmdFetchSmartContractInfo
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgFetchSmartContractInfo) MaxPayloadLength(pver uint32) uint32 {
	// 10k. In theory this message is very small.
	return 10240
}

// GetShardIndex returns the shardIndex.
func (msg *MsgFetchSmartContractInfo) GetShardIndex() shard.Index {
	return msg.ShardIndex
}