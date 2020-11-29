// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
)

func fakeMsgReturnTxs() *MsgReturnTxs {
	tx := FakeTxMsgWithProof()
	sc := newSmartContractInfoForTest()
	idx := shard.Index(3)
	return &MsgReturnTxs{
		ShardIndex:         idx,
		Txs:                []*MsgTxWithProofs{tx},
		SmartContractInfos: []*SmartContractInfo{sc},
	}
}

func TestReturnTxsEncodeDecode(t *testing.T) {
	msg := fakeMsgReturnTxs()
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgReturnTxs encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgReturnTxs decoding failure, err: %v", err)
	}
}

func TestReturnTxsCommand(t *testing.T) {
	msg := fakeMsgReturnTxs()
	expect := CmdReturnTxs
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestReturnTxsMaxPayloadLength(t *testing.T) {
	msg := fakeMsgReturnTxs()
	expect := uint32(MaxReturnedMsgsPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
