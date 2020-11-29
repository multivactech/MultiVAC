// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// MsgBinaryBA indicates binary Byzantine agreement message
type MsgBinaryBA struct {
	InShardProof           []byte
	SignedCredentialWithBA *SignedMsg `rlp:"nil"`
}

// BtcDecode decode the message.
func (m *MsgBinaryBA) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgBinaryBA) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgBinaryBA) Command() string {
	return CmdBinaryBA
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgBinaryBA) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetRawB get B of credentialwithBA.
func (m *MsgBinaryBA) GetRawB() []byte {
	// TODO: decode m.B.Message to get a boolean value
	return []byte{m.SignedCredentialWithBA.Message.CredentialWithBA.B}
}

// GetRawV get V of credentialwithBA.
func (m *MsgBinaryBA) GetRawV() []byte {
	return (interface{})(m.SignedCredentialWithBA.Message.CredentialWithBA.V).([]byte)
}

// GetBValue returns the B value from credentialWithBA
func (m *MsgBinaryBA) GetBValue() byte {
	return m.SignedCredentialWithBA.Message.CredentialWithBA.B
}

// GetByzAgreementValue get the agreement value from credential.
func (m *MsgBinaryBA) GetByzAgreementValue() ByzAgreementValue {
	return m.SignedCredentialWithBA.Message.CredentialWithBA.V
}

// GetStep returns step.
func (m *MsgBinaryBA) GetStep() int {
	return int(m.SignedCredentialWithBA.Message.CredentialWithBA.Step)
}

// GetRound to return msg.round.
func (m *MsgBinaryBA) GetRound() int {
	return int(m.SignedCredentialWithBA.Message.CredentialWithBA.Round)
}

// GetShardIndex returns the shardIndex.
func (m *MsgBinaryBA) GetShardIndex() shard.Index {
	return m.SignedCredentialWithBA.Message.CredentialWithBA.ShardIndex
}

// String returns the format string.
func (m MsgBinaryBA) String() string {
	return fmt.Sprintf("MsgBinaryBA {shardIndex:%v, Round:%v, Step:%v, B:%v, v:%v, SignedCredentialWithBA:%v}",
		m.GetShardIndex(), m.GetRound(), m.GetStep(), m.GetBValue(), m.GetByzAgreementValue(), m.SignedCredentialWithBA.Message)
}

// NewMessageBinaryBA create a MsgBinaryBA with given params.
func NewMessageBinaryBA(inshardProof []byte, signedMsg *SignedMsg) *MsgBinaryBA {
	return &MsgBinaryBA{
		InShardProof:           inshardProof,
		SignedCredentialWithBA: signedMsg,
	}
}

// GetSignedCredential returns the SignedCredential.
func (m *MsgBinaryBA) GetSignedCredential() *SignedMsg {
	return m.SignedCredentialWithBA
}

// GetInShardProofs returns message's InShardProof.
func (m *MsgBinaryBA) GetInShardProofs() [][]byte {
	return [][]byte{m.InShardProof}
}

// IsValidated Verify the signature of ByzantineValue.
func (m *MsgBinaryBA) IsValidated() error {
	return isValidSignedCredential(m.SignedCredentialWithBA.Message.CredentialWithBA, m.SignedCredentialWithBA)
}

// Sign Message.
func (m *MsgBinaryBA) Sign(sk []byte) (err error) {
	return m.SignedCredentialWithBA.sign(m.SignedCredentialWithBA.Message.CredentialWithBA, sk)
}

// MsgBinaryBAFin indicates last step binary Byzantine agreement message.
type MsgBinaryBAFin struct {
	*MsgBinaryBA
}

// BtcDecode decode the message.
func (m *MsgBinaryBAFin) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgBinaryBAFin) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgBinaryBAFin) Command() string {
	return CmdBinaryBAFin
}

// NewMessageBinaryBAFin create a MsgBinaryBAFin with given MsgBinaryBA with steps less than 0.
func NewMessageBinaryBAFin(message *MsgBinaryBA) *MsgBinaryBAFin {
	return &MsgBinaryBAFin{MsgBinaryBA: message}
}
