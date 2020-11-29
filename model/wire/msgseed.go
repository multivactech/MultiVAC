// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// MsgSeed is the data structure of seed message.
type MsgSeed struct {
	ShardIndex       shard.Index
	SignedCredential *SignedMsg `rlp:"nil"`
	LastSeedSig      []byte
	InShardProof     []byte
}

// BtcDecode decode the message.
func (m *MsgSeed) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgSeed) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *MsgSeed) Command() string {
	return CmdSeed
}

// GetShardIndex returns the shardIndex.
func (m *MsgSeed) GetShardIndex() shard.Index {
	return m.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *MsgSeed) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetSender returns the sender's pk.
func (m *MsgSeed) GetSender() []byte {
	return m.SignedCredential.Pk
}

// GetRound get the round of message.
func (m *MsgSeed) GetRound() int {
	return int(m.SignedCredential.Message.Credential.Round)
}

// GetStep get the step of message.
func (m *MsgSeed) GetStep() int {
	return int(m.SignedCredential.Message.Credential.Step)
}

// GetSignedCredential returns the SignedCredential.
func (m *MsgSeed) GetSignedCredential() *SignedMsg {
	return m.SignedCredential
}

// GetInShardProofs returns message's InShardProof.
func (m *MsgSeed) GetInShardProofs() [][]byte {
	return [][]byte{m.InShardProof}
}

// IsValidated Verify the signature of ByzantineValue.
func (m *MsgSeed) IsValidated() error {
	return isValidSignedCredential(m.SignedCredential.Message.Credential, m.SignedCredential)
}

// Sign Message.
func (m *MsgSeed) Sign(sk []byte) error {
	return m.SignedCredential.sign(m.SignedCredential.Message.Credential, sk)
}
