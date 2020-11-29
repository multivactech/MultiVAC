// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"github.com/multivactech/MultiVAC/model/shard"
	"testing"
)

const (
	TestSyncReqShardIdx = 3
)

func fakeMsgSyncReq() *MsgSyncReq {
	msg := NewMsgSyncReq(shard.Index(TestSyncReqShardIdx))
	msg.AddBlockLocatorToRecent(shard.Index(0), 2)
	msg.AddBlockLocator(shard.Index(1), 4, 8)
	return msg
}

func TestSyncReqEncodeDecode(t *testing.T) {
	msg := fakeMsgSyncReq()
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncReq encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncReq decoding failure, err: %v", err)
	}
	if id := msg.ReqShardIdx.GetID(); id != TestSyncReqShardIdx {
		t.Errorf("RequestShardIndex, expected: %v, got: %v", TestSyncReqShardIdx, id)
	}
	if size := len(msg.Locators); size != 2 {
		t.Errorf("Locator len, expected: %d, got: %d", 2, size)
	}
}

func TestSyncReqCommand(t *testing.T) {
	msg := fakeMsgSyncReq()
	expect := CmdSyncReq
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestSyncReqMaxPayloadLength(t *testing.T) {
	msg := fakeMsgSyncReq()
	expect := uint32(MsgSyncReqMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
