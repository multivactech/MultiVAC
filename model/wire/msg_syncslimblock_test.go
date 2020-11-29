package wire

import (
	"bytes"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
)

// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

const (
	TestSyncSlimBlockShardIdx = 3
)

func fakeMsgSyncSlimBlock() *MsgSyncSlimBlock {
	msg := NewMsgSyncSlimBlock(shard.Index(TestSyncSlimBlockShardIdx), &blockTwo)
	return msg
}

func TestSyncSlimBlockEncodeDecode(t *testing.T) {
	msg := fakeMsgSyncSlimBlock()
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncSlimBlock encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncSlimBlock decoding failure, err: %v", err)
	}
	if id := msg.ReqShardIdx.GetID(); id != TestSyncSlimBlockShardIdx {
		t.Errorf("RequestShardIndex, expected: %v, got: %v", TestSyncSlimBlockShardIdx, id)
	}
}

func TestSyncSlimBlockCommand(t *testing.T) {
	msg := fakeMsgSyncSlimBlock()
	expect := CmdSyncSlimBlock
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestSyncSlimBlockMaxPayloadLength(t *testing.T) {
	msg := fakeMsgSyncSlimBlock()
	expect := uint32(MsgSyncSlimBlockMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
