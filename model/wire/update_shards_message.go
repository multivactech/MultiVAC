// Copyright (c) 2018-present, MultiVAC dev team.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// UpdatePeerInfoMessage is the message for updating shards info.
type UpdatePeerInfoMessage struct {
	Shards               []shard.Index
	IsDNS                bool
	IsStorageNode        bool
	StorageNode          []string
	RPCListenerAddress   string
	LocalListenerAddress string
}

// BtcDecode decode the message.
func (msg *UpdatePeerInfoMessage) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode serialize the message.
func (msg *UpdatePeerInfoMessage) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the command string.
func (msg *UpdatePeerInfoMessage) Command() string {
	return CmdUpdatePeerInfo
}

// MaxPayloadLength returns the max playlod length.
func (msg *UpdatePeerInfoMessage) MaxPayloadLength(_ uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetShards returns shard index slice.
func (msg *UpdatePeerInfoMessage) GetShards() []shard.Index {
	return msg.Shards
}
