// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/prometheus/common/log"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/merkle"
)

// MsgTxWithProofs is used for propagating pending transactions between nodes.
type MsgTxWithProofs struct {
	Tx MsgTx
	// Merkle path in the tree of all OutStates in the shard.
	Proofs []merkle.MerklePath
}

// Command returns the command string.
func (msg *MsgTxWithProofs) Command() string {
	return CmdTxWithProofs
}

// BtcDecode decode the message.
func (msg *MsgTxWithProofs) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// Deserialize deserialize the message.
func (msg *MsgTxWithProofs) Deserialize(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

// BtcEncode encode the message.
func (msg *MsgTxWithProofs) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Serialize serialize the message.
func (msg *MsgTxWithProofs) Serialize(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

// MaxPayloadLength returns max block playload.
func (msg *MsgTxWithProofs) MaxPayloadLength(_ uint32) uint32 {
	return MaxBlockPayload
}

// ToBytesArray serialize the message and return byte array.
func (msg *MsgTxWithProofs) ToBytesArray() []byte {
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, 0, BaseEncoding)
	if err != nil {
		log.Errorf("failed to encode message,err:%v", err)
		return nil
	}
	return buf.Bytes()
}

// VerifyTxWithProof verifies whether the msg is valid.
// 1. Verifies whether the transaction itself is valid, see MsgTx#verifyTransaction.
// 2. Verifies all the proofs are valid.
// 3. Verifies for each TxIn, there's a corresponding proof.
func (msg *MsgTxWithProofs) VerifyTxWithProof(ledgerMerkleRoot *merkle.MerkleHash) error {
	if err := msg.Tx.VerifyTransaction(); err != nil {
		return err
	}
	if len(msg.Tx.TxIn) != len(msg.Proofs) {
		return errors.New("length of txin is different from length of proofs")
	}

	// Verify all txin are match the proof.
	for index, txIn := range msg.Tx.TxIn {
		hash := merkle.ComputeMerkleHash(
			txIn.PreviousOutPoint.ToUnspentOutState().ToBytesArray())

		// Verify the proof of this out is valid
		proof := msg.Proofs[index]
		if *hash != *proof.GetLeaf() {
			return fmt.Errorf("wrong proof for %d th txin", index)
		}
		if err := proof.Verify(ledgerMerkleRoot); err != nil {
			return fmt.Errorf("invalid proof, err msg: %s", err)
		}
	}
	return nil
}
