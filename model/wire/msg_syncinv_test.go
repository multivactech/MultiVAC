// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"testing"
)

const (
	TestSyncInvShardIdx = 3
)

func fakeMsgSyncInv(t SyncInvType) *MsgSyncInv {
	msg := NewMsgSyncInv(shard.Index(TestSyncInvShardIdx), t)
	msg.AddInvGroup(shard.Index(TestSyncInvShardIdx),
		chainhash.HashH([]byte("test1")), chainhash.HashH([]byte("test2")))
	return msg
}

func TestSyncInvEncodeDecode(t *testing.T) {
	msg := fakeMsgSyncInv(SyncInvTypeFullList)

	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncInv encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncInv decoding failure, err: %v", err)
	}
	if id := msg.ReqShardIdx.GetID(); id != TestSyncInvShardIdx {
		t.Errorf("RequestShardIndex, expected: %v, got: %v", TestSyncInvShardIdx, id)
	}
	groups := msg.InvGroups
	if size := len(groups); size != 1 {
		t.Errorf("InvGroups len, expected: %d, got: %d", 1, size)
	}
	if size := len(groups[0].HeaderHashes); size != 2 {
		t.Errorf("InvGroup[0] HeaderHashes len, expected: %d, got: %d", 2, size)
	}
}

func TestSyncInvCommand(t *testing.T) {
	msg := fakeMsgSyncInv(SyncInvTypeFullList)
	expect := CmdSyncInv
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestSyncInvMaxPayloadLength(t *testing.T) {
	msg := fakeMsgSyncInv(SyncInvTypeFullList)
	expect := uint32(MsgSyncInvMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
