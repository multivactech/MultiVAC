// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"testing"
)

func fakeMsgReSendSyncReq() *MsgReSendSyncReq {

	return NewReSendSyncReq()
}

func TestReSendSyncReqEncodeDecode(t *testing.T) {
	msg := fakeMsgReSendSyncReq()

	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncInv encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncInv decoding failure, err: %v", err)
	}
}

func TestReSendSyncReqCommand(t *testing.T) {
	msg := fakeMsgReSendSyncReq()
	expect := CmdReSendSyncreq
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestReSendSyncReqMaxPayloadLength(t *testing.T) {
	msg := fakeMsgReSendSyncReq()
	expect := uint32(MsgReSendSyncReqMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
