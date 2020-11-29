// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/prometheus/common/log"
	"math/big"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
)

const defaultShardIDForTest = shard.Index(0)

var privateKeyForTestingTx = signature.PrivateKey{
	244, 138, 151, 117, 41, 188, 240, 122, 9, 91, 196, 16, 0, 218, 181, 139, 191, 239, 56, 161, 117, 54, 234, 61, 117, 159, 75, 32, 8, 250, 197, 124, 194, 112, 47, 169, 227, 191, 100, 160, 190, 42, 73, 45, 224, 181, 65, 162, 68, 159, 26, 160, 8, 109, 133, 247, 223, 40, 100, 226, 107, 234, 19, 104,
}

func newMsgTxForTest() *MsgTx {
	return NewMsgTx(TxVersion, defaultShardIDForTest)
}

// TestTx tests the MsgTx API.
func TestTx(t *testing.T) {
	pver := ProtocolVersion

	// Block 100000 hash.
	hashStr := "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Ensure the command is expected value.
	wantCmd := "tx"
	msg := newMsgTxForTest()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAddr: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	wantPayload := uint32(1000 * 4000)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure we get the same transaction output point data back out.
	// NOTE: This is a block hash and made up index, but we're only
	// testing package functionality.
	prevOutIndex := 1
	prevOut := newOutPoint(hash, prevOutIndex, chainhash.Hash{}, big.NewInt(10))
	if !prevOut.TxHash.IsEqual(hash) {
		t.Errorf("newOutPoint: wrong hash - got %v, want %v",
			spew.Sprint(&prevOut.TxHash), spew.Sprint(hash))
	}
	if prevOut.Index != prevOutIndex {
		t.Errorf("newOutPoint: wrong index - got %v, want %v",
			prevOut.Index, prevOutIndex)
	}
	prevOutStr := fmt.Sprintf("%s:%d", hash.String(), prevOutIndex)
	if s := prevOut.String(); s != prevOutStr {
		t.Errorf("OutPoint.String: unexpected result - got %v, "+
			"want %v", s, prevOutStr)
	}

	txIn := NewTxIn(prevOut)
	if !reflect.DeepEqual(&txIn.PreviousOutPoint, prevOut) {
		t.Errorf("NewTxIn: wrong prev outpoint - got %v, want %v",
			spew.Sprint(&txIn.PreviousOutPoint),
			spew.Sprint(prevOut))
	}

	// Ensure we get the same transaction output back out.

	// Ensure transaction inputs are added properly.
	msg.AddTxIn(txIn)
	if !reflect.DeepEqual(msg.TxIn[0], txIn) {
		t.Errorf("AddTxIn: wrong transaction input added - got %v, want %v",
			spew.Sprint(msg.TxIn[0]), spew.Sprint(txIn))
	}

	// Ensure transaction outputs are added properly.

}

// TestTxHash tests the ability to generate the hash of a transaction accurately.
func TestTxHash(t *testing.T) {
	// From block 23157 in a past version of segnet.
	msgTx := newMsgTxForTest()
	txIn := TxIn{
		PreviousOutPoint: OutPoint{
			TxHash: chainhash.Hash{
				0xa5, 0x33, 0x52, 0xd5, 0x13, 0x57, 0x66, 0xf0,
				0x30, 0x76, 0x59, 0x74, 0x18, 0x26, 0x3d, 0xa2,
				0xd9, 0xc9, 0x58, 0x31, 0x59, 0x68, 0xfe, 0xa8,
				0x23, 0x52, 0x94, 0x67, 0x48, 0x1f, 0xf9, 0xcd,
			},
			Index: 19,
			Data:  []byte{},
		},
	}

	msgTx.AddTxIn(&txIn)

	txHash := msgTx.TxHash()
	var buf bytes.Buffer
	err := msgTx.BtcEncode(&buf, 0, BaseEncoding)
	if err != nil {
		log.Errorf("failed to encode message,err:%v", err)
	}
	wantHash := chainhash.HashH(buf.Bytes())

	if !txHash.IsEqual(&wantHash) {
		t.Errorf("TxSha: wrong hash - got %v, want %v",
			spew.Sprint(txHash), spew.Sprint(wantHash))
	}
}

// TestTxWire tests the MsgTx wire encode and decode for various numbers
// of transaction inputs and outputs and protocol versions.
func TestTxWire(t *testing.T) {
	// Empty tx message.
	noTx := newMsgTxForTest()
	noTx.Version = 1
	noTxOut := &MsgTx{
		Version:            1,
		TxIn:               []*TxIn{},
		SignatureScript:    []byte{},
		Params:             []byte{},
		ContractAddress:    isysapi.SysAPIAddress,
		PublicKey:          []byte{},
		StorageNodeAddress: multivacaddress.Address{},
	}

	tests := []struct {
		in   *MsgTx          // Message to encode
		out  *MsgTx          // Expected decoded message
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding format
	}{
		// Latest protocol version with no transactions.
		{
			noTx,
			noTxOut,
			ProtocolVersion,
			BaseEncoding,
		},

		// Latest protocol version with multiple transactions.
		{
			multiTx,
			multiTx,
			ProtocolVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		// Decode the message from wire format.
		var msg MsgTx
		rbuf := bytes.NewReader(buf.Bytes())
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&msg), spew.Sdump(test.out))
			continue
		}
	}
}

func TestTxSignature(t *testing.T) {
	pubKey := privateKeyForTestingTx.Public()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))
	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))
	data := tx.doubleSha256WithoutSignatureAndPubKey()
	sig := signature.Sign(privateKeyForTestingTx, data[:])
	tx.SetSignatureScriptAndPubKey(pubKey, sig)

	if !tx.VerifySignature() {
		t.Error("Signature verification failed")
	}
}

func TestTxSignature_wrongSignature(t *testing.T) {
	randPubKeyBytes := sha256.Sum256([]byte("blahblah"))
	randPubKey := signature.PublicKey(randPubKeyBytes[:])

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(randPubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))
	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(randPubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))
	data := tx.doubleSha256WithoutSignatureAndPubKey()
	sig := signature.Sign(privateKeyForTestingTx, data[:])
	tx.SetSignatureScriptAndPubKey(randPubKey, sig)

	if tx.VerifySignature() {
		t.Error("Signature verification failed")
	}
}

func TestVerifyTx_txinIsFromDifferentShard(t *testing.T) {
	pubKey := privateKeyForTestingTx.Public()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(&OutPoint{
		Shard:       defaultShardIDForTest + 1,
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))

	tx.Sign(&privateKeyForTestingTx)

	if err := tx.VerifyTransaction(); err == nil {
		t.Error("Expecting error, Txin is from a different shard")
	}
}

func TestVerifyTx_sameInputUsedTwice(t *testing.T) {
	pubKey := privateKeyForTestingTx.Public()

	tx := newMsgTxForTest()
	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))

	tx.AddTxIn(NewTxIn(&OutPoint{
		TxHash:      chainhash.Hash{},
		Index:       1,
		UserAddress: multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress),
		Data:        MtvValueToData(big.NewInt(100)),
	}))

	tx.Sign(&privateKeyForTestingTx)

	if err := tx.VerifyTransaction(); err == nil {
		t.Error("Expecting error, same input is used twice")
	}
}

// it seems unused.
//var multiTxData = struct{ Value *big.Int }{Value: big.NewInt(100000)}

// it seems unused.
//var multiTxDataBytes, _ = rlp.EncodeToBytes(multiTxData)

// multiTx is a MsgTx with an input and output and used in various tests.
var multiTx = &MsgTx{
	Version: 1,
	TxIn: []*TxIn{
		{
			PreviousOutPoint: OutPoint{
				TxHash:          chainhash.Hash{},
				Index:           0xffffffff,
				UserAddress:     multivacaddress.Address{},
				Data:            MtvValueToData(big.NewInt(100000)),
				ContractAddress: isysapi.SysAPIAddress,
			},
		},
	},
	ContractAddress:    isysapi.SysAPIAddress,
	SignatureScript:    []byte{},
	Params:             []byte{},
	PublicKey:          []byte{},
	StorageNodeAddress: multivacaddress.Address{},
}

// multiTxEncoded is the wire encoded bytes for multiTx using protocol version
// 60002 and is used in the various tests.
// it seems unused.
//var multiTxEncoded = []byte{
//	0x01, 0x00, 0x00, 0x00, // Version
//	0x01, // Varint for number of input transactions
//	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Previous output hash
//	0xff, 0xff, 0xff, 0xff, // Prevous output index
//	0x07,                                     // Varint for length of signature script
//	0x04, 0x31, 0xdc, 0x00, 0x1b, 0x01, 0x62, // Signature script
//	0xff, 0xff, 0xff, 0xff, // Sequence
//	0x02,                                           // Varint for number of output transactions
//	0x00, 0xf2, 0x05, 0x2a, 0x01, 0x00, 0x00, 0x00, // Transaction amount
//	0x43, // Varint for length of pk script
//	0x41, // OP_DATA_65
//	0x04, 0xd6, 0x4b, 0xdf, 0xd0, 0x9e, 0xb1, 0xc5,
//	0xfe, 0x29, 0x5a, 0xbd, 0xeb, 0x1d, 0xca, 0x42,
//	0x81, 0xbe, 0x98, 0x8e, 0x2d, 0xa0, 0xb6, 0xc1,
//	0xc6, 0xa5, 0x9d, 0xc2, 0x26, 0xc2, 0x86, 0x24,
//	0xe1, 0x81, 0x75, 0xe8, 0x51, 0xc9, 0x6b, 0x97,
//	0x3d, 0x81, 0xb0, 0x1c, 0xc3, 0x1f, 0x04, 0x78,
//	0x34, 0xbc, 0x06, 0xd6, 0xd6, 0xed, 0xf6, 0x20,
//	0xd1, 0x84, 0x24, 0x1a, 0x6a, 0xed, 0x8b, 0x63,
//	0xa6,                                           // 65-byte signature
//	0xac,                                           // OP_CHECKSIG
//	0x00, 0xe1, 0xf5, 0x05, 0x00, 0x00, 0x00, 0x00, // Transaction amount
//	0x43, // Varint for length of pk script
//	0x41, // OP_DATA_65
//	0x04, 0xd6, 0x4b, 0xdf, 0xd0, 0x9e, 0xb1, 0xc5,
//	0xfe, 0x29, 0x5a, 0xbd, 0xeb, 0x1d, 0xca, 0x42,
//	0x81, 0xbe, 0x98, 0x8e, 0x2d, 0xa0, 0xb6, 0xc1,
//	0xc6, 0xa5, 0x9d, 0xc2, 0x26, 0xc2, 0x86, 0x24,
//	0xe1, 0x81, 0x75, 0xe8, 0x51, 0xc9, 0x6b, 0x97,
//	0x3d, 0x81, 0xb0, 0x1c, 0xc3, 0x1f, 0x04, 0x78,
//	0x34, 0xbc, 0x06, 0xd6, 0xd6, 0xed, 0xf6, 0x20,
//	0xd1, 0x84, 0x24, 0x1a, 0x6a, 0xed, 0x8b, 0x63,
//	0xa6,                   // 65-byte signature
//	0xac,                   // OP_CHECKSIG
//	0x00, 0x00, 0x00, 0x00, // Lock time
//}

// Returns a new bitcoin transaction outpoint point with the provided hash and index.
func newOutPoint(txHash *chainhash.Hash, index int, pkHash chainhash.Hash, value *big.Int) *OutPoint {
	data := struct{ Value *big.Int }{Value: value}
	dataBytes, _ := rlp.EncodeToBytes(data) // Ignore error
	publicKey := signature.PublicKey(pkHash.CloneBytes())
	return &OutPoint{
		TxHash:      *txHash,
		Index:       index,
		Shard:       shard.Index(0),
		UserAddress: multivacaddress.GenerateAddress(publicKey, multivacaddress.UserAddress),
		Data:        dataBytes,
	}
}
