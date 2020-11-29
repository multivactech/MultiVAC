// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MsgBlockConfirmation indicates binary Byzantine agreement message.
type MsgBlockConfirmation struct {
	ShardIndex shard.Index
	Round      int32
	Step       int32
	V          ByzAgreementValue
	// Header 当前re-shard轮公示出来的block header
	Header   *BlockHeader `rlp:"nil"`
	Confirms []*MsgBinaryBAFin
}

// BtcDecode decode the message.
func (msg *MsgBlockConfirmation) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgBlockConfirmation) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgBlockConfirmation) Command() string {
	return CmdMsgBlockConfirmation
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgBlockConfirmation) MaxPayloadLength(uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetRawV get the V value.
func (msg *MsgBlockConfirmation) GetRawV() []byte {
	return (interface{})(msg.V).([]byte)
}

// GetByzAgreementValue get the byz agreement value.
func (msg *MsgBlockConfirmation) GetByzAgreementValue() ByzAgreementValue {
	return msg.V
}

// GetStep get the step of message.
func (msg *MsgBlockConfirmation) GetStep() int {
	return int(msg.Step)
}

// GetRound get the round of message.
func (msg *MsgBlockConfirmation) GetRound() int {
	return int(msg.Round)
}

func (msg *MsgBlockConfirmation) String() string {
	return fmt.Sprintf("MsgBlockConfirmation {shardIndex:%v, round:%v, Step:%v, V:%v, Length of Confirms :%d}",
		msg.ShardIndex, msg.Round, msg.Step, msg.V, len(msg.Confirms))
}

// IsValidated judged the following conditions:
// 1. Header and Header.Sig
// 2. Cur round seed.
// 3. Confirms:
//    * bbaMsg and cred Signature(pk)
//    * bbaMsg.Index, Round, Step, ByzAgreementValue
// BUT STILL THE FOLLOWING NOT JUDGED:
// 1. Header.PrevBlockHeader ?
// 2. Header.PrevSeedSig ?
// 3. Header.PK is Potential leader and is InShard ?
// 4. len(confirms) >= threshold ?
// 5. every BBAMsg.PK is InShard and alive ?
func (msg *MsgBlockConfirmation) IsValidated() error {
	vrf := &vrf.Ed25519VRF{}

	headerPk := msg.Header.Pk
	round := msg.Header.Height - 2
	emptyBlockHash := chainhash.Hash{}

	// verify blockHeader.
	// but PrevBlockHeader cannot be verified here.
	blockHeaderHash := msg.Header.BlockHeaderHash()
	if msg.Header.IsEmptyBlock {
		return nil
	}

	if res, err := vrf.VerifyProof(headerPk, msg.Header.HeaderSig, blockHeaderHash[0:]); !res {
		return fmt.Errorf("verify header failed, %v, pk: %v, headerSig: %v, headerHash: %v", err,
			hex.EncodeToString(headerPk),
			hex.EncodeToString(msg.Header.HeaderSig), hex.EncodeToString(blockHeaderHash[0:]))
	}

	int64Size := 8
	seedBuf := make([]byte, len(msg.Header.LastSeedSig)+int64Size)
	copy(seedBuf, msg.Header.LastSeedSig[0:])
	binary.PutVarint(seedBuf[len(msg.Header.LastSeedSig):], int64(round))

	// verify seed
	seed := msg.Header.Seed
	curSeed := sha256.Sum256(seedBuf)
	if !bytes.Equal(seed[0:], curSeed[0:]) {
		return fmt.Errorf("verify seed failed. msg.seed: %v, calculateSeed: %v", seed, curSeed)
	}

	// verify Value
	if hex.EncodeToString(msg.Header.Pk) != msg.V.Leader {
		return fmt.Errorf("verify Header.Pk and ByzAgreementValue.Leader failed")
	}

	// It is impossible to judge whether the header.pk is recognized by sortitionor as the leader of this round here,
	// so this item should be judged in the consensus

	// verify confirms
	B := byte(0)
	if msg.Header.BlockHeaderHash() == emptyBlockHash {
		B = 1
	}
	for _, bbaMsg := range msg.Confirms {
		// verify bba sigature
		cred := NewCredentialWithBA(bbaMsg.GetShardIndex(), int32(bbaMsg.GetRound()), int32(bbaMsg.GetStep()), B,
			bbaMsg.GetByzAgreementValue())
		var buf []byte
		var err error
		if buf, err = cred.Serialize(); err != nil {
			return fmt.Errorf("verify confirms failed, cred serialize failed")
		}
		if res, err := vrf.VerifyProof(bbaMsg.SignedCredentialWithBA.Pk, bbaMsg.SignedCredentialWithBA.Signature, buf); !res {
			return fmt.Errorf("verify confirms failed, bba proof wrong, %v %v", err, cred)
		}
		// verify shardIndex
		if bbaMsg.GetShardIndex() != msg.ShardIndex {
			return fmt.Errorf("verify confirms failed, confirmation.Index: %v, bbaMsg.Index: %v",
				msg.ShardIndex, bbaMsg.GetShardIndex())
		}
		// verify round
		if bbaMsg.GetRound() != int(msg.Round) {
			return fmt.Errorf("verify confirms failed, confirmation.round: %v, bbaMsg.round: %v",
				msg.Round, bbaMsg.GetRound())
		}
		// verify step
		if int32(bbaMsg.GetStep()) > msg.Step {
			return fmt.Errorf("verify confirms failed, confirmation.round: %v, bbaMsg.round: %v",
				msg.Step, bbaMsg.GetStep())
		}
		// verify ByzAgreementValue
		if bbaMsg.GetByzAgreementValue() != msg.V {
			return fmt.Errorf("verify confirms failed, bba ByzAgreementValue wrong")
		}
		// need to verify bbaMsg.pk is in the shard, but not in this function.
	}
	return nil
}

// GetInShardProofs returns message's InShardProof.
func (msg *MsgBlockConfirmation) GetInShardProofs() [][]byte {
	var inShardProofs [][]byte
	for _, cf := range msg.Confirms {
		inShardProofs = append(inShardProofs, cf.InShardProof)
	}
	return inShardProofs
}

// GetSignedCredential returns the SignedCredential.
func (msg *MsgBlockConfirmation) GetSignedCredential() *SignedMsg {
	return nil
}

// GetVoters will return all pk of voters.
func (msg *MsgBlockConfirmation) GetVoters() [][]byte {
	var voters [][]byte
	for _, bbaMsg := range msg.Confirms {
		voters = append(voters, bbaMsg.SignedCredentialWithBA.Pk)
	}
	return voters
}

// GetShardIndex returns the shardIndex.
func (msg *MsgBlockConfirmation) GetShardIndex() shard.Index {
	return msg.ShardIndex
}

// NewMessageBlockConfirmation create a new block confirmation message。
func NewMessageBlockConfirmation(shard shard.Index, round, step int32, v *ByzAgreementValue,
	header *BlockHeader, confirms []*MsgBinaryBAFin) *MsgBlockConfirmation {
	return &MsgBlockConfirmation{shard, round, step, *v, header, confirms}
}
