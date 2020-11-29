// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"math/big"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
)

var (
	out1, out2, out3                               *OutPoint
	out1MerkleHash, out2MerkleHash, out3MerkleHash *merkle.MerkleHash
	path1, path2, path3                            *merkle.MerklePath
	outsTree                                       *merkle.FullMerkleTree
	privateKeyForTestingTWP                        = privateKeyForTestingTx
	pkHash                                         chainhash.Hash
)

func FakeTxMsgWithProof() *MsgTxWithProofs {
	createSimpleOutsTree()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(out1))
	tx.AddTxIn(NewTxIn(out2))
	tx.AddTxIn(NewTxIn(out3))

	tx.Sign(&privateKeyForTestingTWP)

	proofs := []merkle.MerklePath{
		*path1, *path2, *path3,
	}

	twp := &MsgTxWithProofs{
		Tx:     *tx,
		Proofs: proofs,
	}

	return twp
}

func TestVerifyTxWithProof(t *testing.T) {
	twp := FakeTxMsgWithProof()

	if err := twp.VerifyTxWithProof(outsTree.MerkleRoot); err != nil && twp.Tx.IsEnoughForReward() {
		t.Error(err.Error())
	}
}

func TestVerifyTxWithProof_lengthOfProofMismatch(t *testing.T) {
	createSimpleOutsTree()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(out1))
	tx.AddTxIn(NewTxIn(out2))
	tx.AddTxIn(NewTxIn(out3))
	tx.Sign(&privateKeyForTestingTWP)

	// Three txin but only two proof
	proofs := []merkle.MerklePath{
		*path1, *path2,
	}

	twp := &MsgTxWithProofs{
		Tx:     *tx,
		Proofs: proofs,
	}

	if err := twp.VerifyTxWithProof(outsTree.MerkleRoot); err == nil {
		t.Error("Expecting error: There are less proof than txin")
	}
}

func TestVerifyTxWithProof_wrongProof(t *testing.T) {
	createSimpleOutsTree()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(out1))
	tx.AddTxIn(NewTxIn(out2))
	tx.AddTxIn(NewTxIn(out3))
	tx.Sign(&privateKeyForTestingTWP)

	proofs := []merkle.MerklePath{
		*path1, *path2, *path3,
	}

	twp := &MsgTxWithProofs{
		Tx:     *tx,
		Proofs: proofs,
	}

	anotherRoot := merkle.ComputeMerkleHash([]byte("another root"))

	if err := twp.VerifyTxWithProof(anotherRoot); err == nil {
		t.Error("Expecting error: merkle proofs are not valid")
	}
}

func createSimpleOutsTree() {
	pubKey := privateKeyForTestingTWP.Public()
	tpkBytes := []byte(pubKey)
	var pubKeyBytes [32]byte
	copy(pubKeyBytes[:], tpkBytes[:])
	pkHash = chainhash.Hash(pubKeyBytes)

	txHash := chainhash.HashH([]byte("testtx"))
	out1 = newOutPoint(&txHash, 1, pkHash, big.NewInt(100))
	out2 = newOutPoint(&txHash, 2, pkHash, big.NewInt(100))
	out3 = newOutPoint(&txHash, 3, pkHash, big.NewInt(100))

	out1MerkleHash = merkle.ComputeMerkleHash(out1.ToUnspentOutState().ToBytesArray())
	out2MerkleHash = merkle.ComputeMerkleHash(out2.ToUnspentOutState().ToBytesArray())
	out3MerkleHash = merkle.ComputeMerkleHash(out3.ToUnspentOutState().ToBytesArray())

	outsTree = merkle.NewFullMerkleTree(out1MerkleHash, out2MerkleHash, out3MerkleHash)
	path1, _ = outsTree.GetMerklePath(out1MerkleHash)
	path2, _ = outsTree.GetMerklePath(out2MerkleHash)
	path3, _ = outsTree.GetMerklePath(out3MerkleHash)
}
