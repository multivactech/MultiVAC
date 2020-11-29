// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2018-present, MultiVAC dev team.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/shard"
)

// LeaderVoteMessage is a type of message LeaderVoteMessage.
type LeaderVoteMessage struct {
	ShardIndex       shard.Index
	InShardProof     []byte
	SignedVal        *SignedMsg `rlp:"nil"`
	Block            *MsgBlock  `rlp:"nil"`
	SignedCredential *SignedMsg `rlp:"nil"`
}

// BtcDecode decode the message.
func (m *LeaderVoteMessage) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *LeaderVoteMessage) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, m)
}

// Command returns the protocol command string for the message.
func (m *LeaderVoteMessage) Command() string {
	return CmdLeaderVote
}

// GetShardIndex returns the shardIndex.
func (m *LeaderVoteMessage) GetShardIndex() shard.Index {
	return m.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (m *LeaderVoteMessage) MaxPayloadLength(_ uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// String returns the format string for LeaderVoteMessage.
func (m *LeaderVoteMessage) String() string {
	return fmt.Sprintf("LeaderVoteMessage {shardIndex:%v, round:%v, InshardProof:%v, Val:%v, Block:%v, SignedCredentialWithBA:%v}",
		m.ShardIndex, m.GetRound(), m.InShardProof, m.GetByzAgreementValue(), m.Block, m.SignedCredential.Message)
}

// GetRound returns the round of message.
func (m *LeaderVoteMessage) GetRound() int {
	return int(m.SignedCredential.Message.Credential.Round)
}

// GetStep returns the step of message.
func (m *LeaderVoteMessage) GetStep() int {
	return int(m.SignedCredential.Message.Credential.Step)
}

// GetByzAgreementValue returns the ByzAgreementValue.
func (m *LeaderVoteMessage) GetByzAgreementValue() *ByzAgreementValue {
	return m.SignedVal.Message.ByzAgreementValue
}

// GetInShardProofs returns message's InShardProof.
func (m *LeaderVoteMessage) GetInShardProofs() [][]byte {
	return [][]byte{m.InShardProof}
}

// GetSignedCredential returns the SignedCredential.
func (m *LeaderVoteMessage) GetSignedCredential() *SignedMsg {
	return m.SignedCredential
}

// IsValidated Verify the signature of ByzantineValue.
func (m *LeaderVoteMessage) IsValidated() (err error) {
	err = isValidSignedCredential(m.SignedCredential.Message.Credential, m.SignedCredential)
	if err != nil {
		return
	}

	err = isValidSignedCredential(m.SignedVal.Message.ByzAgreementValue, m.SignedVal)
	return
}

// Sign Message
func (m *LeaderVoteMessage) Sign(sk []byte) (err error) {
	err = m.SignedCredential.sign(m.SignedCredential.Message.Credential, sk)
	if err != nil {
		return
	}
	return m.SignedVal.sign(m.SignedVal.Message.ByzAgreementValue, sk)
}
