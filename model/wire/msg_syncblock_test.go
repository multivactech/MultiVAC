// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
)

const (
	TestSyncBlockShardIdx = 3
)

func fakeMsgSyncBlock() *MsgSyncBlock {
	msg := NewMsgSyncBlock(shard.Index(TestSyncBlockShardIdx), &blockOne)
	return msg
}

func TestSyncBlockEncodeDecode(t *testing.T) {
	msg := fakeMsgSyncBlock()
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncBlock encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncBlock decoding failure, err: %v", err)
	}
	if id := msg.ReqShardIdx.GetID(); id != TestSyncBlockShardIdx {
		t.Errorf("RequestShardIndex, expected: %v, got: %v", TestSyncBlockShardIdx, id)
	}
}

func TestSyncBlockCommand(t *testing.T) {
	msg := fakeMsgSyncBlock()
	expect := CmdSyncBlock
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestSyncBlockMaxPayloadLength(t *testing.T) {
	msg := fakeMsgSyncBlock()
	expect := uint32(MsgSyncBlockMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
