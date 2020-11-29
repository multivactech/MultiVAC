// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2018-present, MultiVAC dev team.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"errors"
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
)

// LeaderProposalMessage is a message for broadcasting potential leader in current round.
type LeaderProposalMessage struct {
	ShardIndex       shard.Index
	InShardProof     []byte
	Block            *MsgBlock `rlp:"nil"`
	BlockHash        chainhash.Hash
	SignedHash       []byte
	LastSeedSig      []byte
	SignedCredential *SignedMsg `rlp:"nil"`
}

// BtcDecode decode the message.
func (msg *LeaderProposalMessage) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *LeaderProposalMessage) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	newMsg := *msg
	return rlp.Encode(w, newMsg)
}

// Command returns the protocol command string for the message.
func (msg *LeaderProposalMessage) Command() string {
	return CmdLeaderProposal
}

// GetShardIndex returns the shardIndex.
func (msg *LeaderProposalMessage) GetShardIndex() shard.Index {
	return msg.ShardIndex
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *LeaderProposalMessage) MaxPayloadLength(_ uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

func (msg *LeaderProposalMessage) String() string {
	return fmt.Sprintf("%s Index:%v, Round:%v, BlockHash:%v",
		msg.Command(), msg.ShardIndex, msg.GetRound(),
		msg.BlockHash)
}

// GetRound to return msg.round which in msg.SignedCredentialWithBA.Credential.round.
func (msg *LeaderProposalMessage) GetRound() int {
	return int(msg.SignedCredential.Message.Credential.Round)
}

// GetStep to return message's step.
func (msg *LeaderProposalMessage) GetStep() int {
	return int(msg.SignedCredential.Message.Credential.Step)
}

// GetInShardProofs returns message's InShardProof.
func (msg *LeaderProposalMessage) GetInShardProofs() [][]byte {
	return [][]byte{msg.InShardProof}
}

// GetSignedCredential returns the SignedCredential.
func (msg *LeaderProposalMessage) GetSignedCredential() *SignedMsg {
	return msg.SignedCredential
}

// IsValidated Verify the signature of ByzantineValue.
func (msg *LeaderProposalMessage) IsValidated() error {
	err := isValidSignedCredential(msg.SignedCredential.Message.Credential, msg.SignedCredential)
	if err != nil {
		return err
	}

	// validate hash is correct.
	if msg.BlockHash != msg.Block.Header.BlockHeaderHash() {
		return errors.New("block hash does not match")
	}

	// Validate signedBlockHash is correct.
	if ok, err := ed25519VRF.VerifyProof(msg.SignedCredential.Pk, msg.SignedHash, msg.BlockHash[:]); !ok {
		return fmt.Errorf("can't verify the sign of blockHash, detail: %v", err)

	}
	return nil
}

// Sign Message.
func (msg *LeaderProposalMessage) Sign(sk []byte) (err error) {
	return msg.SignedCredential.sign(msg.SignedCredential.Message.Credential, sk)
}
