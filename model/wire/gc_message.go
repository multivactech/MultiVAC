// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2018-present, MultiVAC dev team.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// GCMessage defines a type of message GC.
type GCMessage struct {
	ShardIndex       shard.Index
	Seed             []byte
	SignedCredential *SignedMsg `rlp:"nil"`
	SignedVal        *SignedMsg `rlp:"nil"`
	Block            *MsgBlock  `rlp:"nil"`
	InShardProof     []byte
}

// BtcDecode decode the message.
func (m *GCMessage) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *GCMessage) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (GCMessage) Command() string {
	return CmdGC
}

// GetShardIndex returns the shardIndex.
func (m *GCMessage) GetShardIndex() shard.Index {
	return m.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (GCMessage) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetRound returns the message's round.
func (m *GCMessage) GetRound() int {
	return int(m.SignedCredential.Message.Credential.Round)
}

// GetStep returns the message's step.
func (m *GCMessage) GetStep() int {
	return int(m.SignedCredential.Message.Credential.Step)
}

// GetByzAgreementValue returns the ByzAgreementValue of the message.
func (m *GCMessage) GetByzAgreementValue() *ByzAgreementValue {
	return m.SignedVal.Message.ByzAgreementValue
}

// String returns the format string for GCMessage.
func (m *GCMessage) String() string {
	return fmt.Sprintf("GCMessage {shardIndex {%v}, round %v, seed %v, Val %v}", m.ShardIndex, m.GetRound(), hex.Dump(m.Seed), m.GetByzAgreementValue())
}

// GetInShardProofs returns the message's InShardProof
func (m *GCMessage) GetInShardProofs() [][]byte {
	return [][]byte{m.InShardProof}
}

// GetSignedCredential returns the message's SignedCredential.
func (m *GCMessage) GetSignedCredential() *SignedMsg {
	return m.SignedCredential
}

// IsValidated returns whether the message is legal.
func (m *GCMessage) IsValidated() (err error) {
	err = isValidSignedCredential(m.SignedCredential.Message.Credential, m.SignedCredential)
	if err != nil {
		return
	}

	err = isValidSignedCredential(m.SignedVal.Message.ByzAgreementValue, m.SignedVal)
	return
}

// Sign Message.
func (m *GCMessage) Sign(sk []byte) (err error) {
	err = m.SignedCredential.sign(m.SignedCredential.Message.Credential, sk)
	if err != nil {
		return
	}
	return m.SignedVal.sign(m.SignedVal.Message.ByzAgreementValue, sk)
}
