// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

const (
	// MsgSyncInvMaxPayload is a very arbitrary number
	MsgSyncInvMaxPayload = 4000000
)

const (
	// SyncInvTypeFullList is to sync inventory full list.
	SyncInvTypeFullList = iota
	// SyncInvTypeGetData is to sync inventory data.
	SyncInvTypeGetData
)

// SyncInvType defines sync type.
type SyncInvType int32

// MsgSyncInv is the message to send an inventory of header hashes for sync.
// Depending on different type specified in this struct, it represents:
// 1. A full list of all header hashes that are available for sync
// 2. A list of header hashes for a single fetch data request to get full content of a block or header.
type MsgSyncInv struct {
	Type        SyncInvType
	ReqShardIdx shard.Index
	InvGroups   []*InvGroup
}

// InvGroup is group of inventory.
type InvGroup struct {
	Shard        shard.Index
	HeaderHashes []chainhash.Hash
}

// NewMsgSyncInv create a sync inventory message.
func NewMsgSyncInv(s shard.Index, t SyncInvType) *MsgSyncInv {
	msg := MsgSyncInv{}
	msg.ReqShardIdx = s
	msg.Type = t
	msg.InvGroups = make([]*InvGroup, 0, DefaultShardCapacity)
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgSyncInv) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgSyncInv) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgSyncInv) Command() string {
	return CmdSyncInv
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgSyncInv) MaxPayloadLength(uint32) uint32 {
	return MsgSyncInvMaxPayload
}

// AddInvGroup add a group inventory.
func (msg *MsgSyncInv) AddInvGroup(shardIdx shard.Index, hashes ...chainhash.Hash) {
	msg.InvGroups = append(msg.InvGroups, NewInvGroup(shardIdx, hashes...))
}

// NewInvGroup create a inventory group.
func NewInvGroup(shardIdx shard.Index, hashes ...chainhash.Hash) *InvGroup {
	g := InvGroup{Shard: shardIdx}
	l := len(hashes)
	g.HeaderHashes = make([]chainhash.Hash, l)
	for i := 0; i < l; i++ {
		g.HeaderHashes[i] = hashes[i]
	}
	return &g
}

func (msg *MsgSyncInv) String() string {
	return fmt.Sprintf("MsgSyncInv {Shard: %v, Type: %v, Groups: [%v]}", msg.ReqShardIdx, msg.Type, msg.InvGroups)
}

func (g *InvGroup) String() string {
	return fmt.Sprintf("Group {Shard: %v, Headers: %v}", g.Shard.GetID(), g.HeaderHashes)
}

// GetShardIndex returns the shardIndex.
func (msg *MsgSyncInv) GetShardIndex() shard.Index {
	return msg.ReqShardIdx
}