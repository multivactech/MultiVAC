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
	TestSyncHeadersShardIdx = 3
)

func fakeMsgSyncHeaders() *MsgSyncHeaders {
	header := BlockHeader{
		Version: 1,
		PrevBlockHeader: chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
			0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
			0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
			0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
			0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
		}),
		DepositTxsOuts:  []OutPoint{},
		WithdrawTxsOuts: []OutPoint{},
		Pk:              []uint8{},
		LastSeedSig:     []uint8{},
		HeaderSig:       []uint8{},
	}
	headers := []*BlockHeader{&header}
	msg := NewMsgSyncHeaders(shard.Index(TestSyncHeadersShardIdx), headers)
	return msg
}

func TestSyncHeadersEncodeDecode(t *testing.T) {
	msg := fakeMsgSyncHeaders()
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncHeaders encoding failure, err: %v", err)
	}
	rbuf := bytes.NewReader(buf.Bytes())
	if err := msg.BtcDecode(rbuf, ProtocolVersion, BaseEncoding); err != nil {
		t.Errorf("MsgSyncHeaders decoding failure, err: %v", err)
	}
	if id := msg.ReqShardIdx.GetID(); id != TestSyncHeadersShardIdx {
		t.Errorf("RequestShardIndex, expected: %v, got: %v", TestSyncHeadersShardIdx, id)
	}
}

func TestSyncHeadersCommand(t *testing.T) {
	msg := fakeMsgSyncHeaders()
	expect := CmdSyncHeaders
	if cmd := msg.Command(); cmd != expect {
		t.Errorf("Command want: %s, actual: %s", expect, cmd)
	}
}

func TestSyncHeadersMaxPayloadLength(t *testing.T) {
	msg := fakeMsgSyncHeaders()
	expect := uint32(MsgSyncHeadersMaxPayload)
	if len := msg.MaxPayloadLength(ProtocolVersion); len != expect {
		t.Errorf("MaxPayloadLength want: %d, actual: %d", expect, len)
	}
}
