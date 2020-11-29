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

// MsgFetchInit is a message to request ledger info from storage node.
type MsgFetchInit struct {
	MsgID      uint32
	ShardIndex shard.Index
	Address    multivacaddress.Address
}

// BtcDecode decode the message.
func (m *MsgFetchInit) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgFetchInit) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	encoder := gob.NewEncoder(w)
	return encoder.Encode(*m)
}

// Command returns the protocol command string for the message.
func (m *MsgFetchInit) Command() string {
	return CmdInitAbciData
}

// GetShardIndex returns the shardIndex.
func (m *MsgFetchInit) GetShardIndex() shard.Index {
	return m.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgFetchInit) MaxPayloadLength(pver uint32) uint32 {
	// 10k. In theory this message is very small.
	return 10240
}
