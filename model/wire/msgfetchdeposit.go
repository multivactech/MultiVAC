// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"encoding/gob"
	"io"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MsgFetchDeposit is a message to request deposit info from storage node.
type MsgFetchDeposit struct {
	ShardIndex shard.Index
	Address    multivacaddress.Address
}

// BtcDecode decode the message.
func (msg *MsgFetchDeposit) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgFetchDeposit) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	encoder := gob.NewEncoder(w)
	return encoder.Encode(*msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgFetchDeposit) Command() string {
	return CmdFetchDeposit
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgFetchDeposit) MaxPayloadLength(pver uint32) uint32 {
	// 10k. In theory this message is very small.
	return 10240
}

// GetShardIndex returns the shardIndex.
func (msg *MsgFetchDeposit) GetShardIndex() shard.Index {
	return msg.ShardIndex
}