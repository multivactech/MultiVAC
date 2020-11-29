// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"fmt"
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
)

// ConsensusMsg defines the data structure of consensus message.
type ConsensusMsg struct {
	Credential        *Credential        `rlp:"nil"`
	CredentialWithBA  *CredentialWithBA  `rlp:"nil"`
	ByzAgreementValue *ByzAgreementValue `rlp:"nil"`
}

var ed25519VRF = &vrf.Ed25519VRF{}

// SignedMsg represents a signed message by a user
type SignedMsg struct {
	Pk        []byte // pk is used for identify the signer
	Signature []byte
	Message   ConsensusMsg
}

// Serializable defines the interface for serialization.
type Serializable interface {
	Serialize() ([]byte, error)
}

// Credential defines the data structure of credential.
type Credential struct {
	Round int32
	Step  int32
}

// CredentialWithBA defines the data structure of credential with BA.
type CredentialWithBA struct {
	ShardIndex shard.Index
	Round      int32
	Step       int32
	B          byte
	V          ByzAgreementValue
}

// Serialize serialize the message.
func (c ConsensusMsg) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(c)
}

// NewCredential returns a credential .
func NewCredential(round, step int32) *Credential {
	return &Credential{round, step}
}

// NewCredentialWithBA returns CredentialWithBA with given params.
func NewCredentialWithBA(shardIndex shard.Index, round, step int32, b byte, v ByzAgreementValue) *CredentialWithBA {
	return &CredentialWithBA{
		shardIndex, round, step, b, v}
}

// ByzAgreementValue defines the data structure of Byz Agreement Value.
type ByzAgreementValue struct {
	BlockHash chainhash.Hash
	// leader indicates the public key of corresponding peer
	// TODO(liang): use byte array with const length, instead of string
	Leader string
}

// Serialize serialize the credential.
func (c Credential) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(c)
}

// Serialize serialize the credential with Byz agreement.,
func (c CredentialWithBA) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(c)
}

// Serialize the byz agreement value.
func (b ByzAgreementValue) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(b)
}

// String returns the format string of ByzAgreementValue.
func (b ByzAgreementValue) String() string {
	return fmt.Sprintf("{blockHash:%v, leader:%v}", b.BlockHash.String(), b.Leader)
}

func isValidSignedCredential(cred Serializable, sc *SignedMsg) error {
	s, err := cred.Serialize()
	if err != nil {
		return fmt.Errorf("cant't serialize ByzAgreementValue message, error: %v", err)
	}
	if ok, err := ed25519VRF.VerifyProof(sc.Pk, sc.Signature, s); !ok {
		return fmt.Errorf("invalid ByzAgreementValue message, error: %v, pk : %v", err, sc.Pk)
	}
	return nil
}

func (msg *SignedMsg) sign(cred Serializable, sk []byte) error {
	buf, err := cred.Serialize()
	if err != nil {
		return err
	}

	sig, err := ed25519VRF.Generate(msg.Pk, sk, buf)
	if err != nil {
		return err
	}
	msg.Signature = sig
	return nil
}
