// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"compress/bzip2"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network, regression test network, and test network (version 3).
var genesisCoinbaseTx = MsgTx{
	Version: 1,
	TxIn: []*TxIn{
		{
			PreviousOutPoint: OutPoint{
				TxHash: chainhash.Hash{},
				Index:  0xffffffff,
			},
		},
	},
}

// BenchmarkWriteVarInt1 performs a benchmark on how long it takes to write
// a single byte variable length integer.
func BenchmarkWriteVarInt1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarInt(ioutil.Discard, 0, 1)
		if err != nil {
			log.Errorf("failed to write var int,err:%v", err)
		}
	}
}

// BenchmarkWriteVarInt3 performs a benchmark on how long it takes to write
// a three byte variable length integer.
func BenchmarkWriteVarInt3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarInt(ioutil.Discard, 0, 65535)
		if err != nil {
			log.Errorf("failed to write var int,err:%v", err)
		}
	}
}

// BenchmarkWriteVarInt5 performs a benchmark on how long it takes to write
// a five byte variable length integer.
func BenchmarkWriteVarInt5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarInt(ioutil.Discard, 0, 4294967295)
		if err != nil {
			log.Errorf("failed to write var int,err:%v", err)
		}
	}
}

// BenchmarkWriteVarInt9 performs a benchmark on how long it takes to write
// a nine byte variable length integer.
func BenchmarkWriteVarInt9(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarInt(ioutil.Discard, 0, 18446744073709551615)
		if err != nil {
			log.Errorf("failed to write var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarInt1 performs a benchmark on how long it takes to read
// a single byte variable length integer.
func BenchmarkReadVarInt1(b *testing.B) {
	buf := []byte{0x01}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarInt3 performs a benchmark on how long it takes to read
// a three byte variable length integer.
func BenchmarkReadVarInt3(b *testing.B) {
	buf := []byte{0x0fd, 0xff, 0xff}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarInt5 performs a benchmark on how long it takes to read
// a five byte variable length integer.
func BenchmarkReadVarInt5(b *testing.B) {
	buf := []byte{0xfe, 0xff, 0xff, 0xff, 0xff}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarInt9 performs a benchmark on how long it takes to read
// a nine byte variable length integer.
func BenchmarkReadVarInt9(b *testing.B) {
	buf := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarStr4 performs a benchmark on how long it takes to read a
// four byte variable length string.
func BenchmarkReadVarStr4(b *testing.B) {
	buf := []byte{0x04, 't', 'e', 's', 't'}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkReadVarStr10 performs a benchmark on how long it takes to read a
// ten byte variable length string.
func BenchmarkReadVarStr10(b *testing.B) {
	buf := []byte{0x0a, 't', 'e', 's', 't', '0', '1', '2', '3', '4', '5'}
	r := bytes.NewReader(buf)
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		_, err = ReadVarInt(r, 0)
		if err != nil {
			log.Errorf("failed to read var int,err:%v", err)
		}
	}
}

// BenchmarkWriteVarStr4 performs a benchmark on how long it takes to write a
// four byte variable length string.
func BenchmarkWriteVarStr4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarString(ioutil.Discard, 0, "test")
		if err != nil {
			log.Errorf("failed to write string,err:%v", err)
		}
	}
}

// BenchmarkWriteVarStr10 performs a benchmark on how long it takes to write a
// ten byte variable length string.
func BenchmarkWriteVarStr10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := WriteVarString(ioutil.Discard, 0, "test012345")
		if err != nil {
			log.Errorf("failed to write string,err:%v", err)
		}
	}
}

// BenchmarkDeserializeTx performs a benchmark on how long it takes to
// deserialize a small transaction.
func BenchmarkDeserializeTxSmall(b *testing.B) {
	buf := []byte{
		0x01, 0x00, 0x00, 0x00, // Version
		0x01, // Varint for number of input transactions
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //  // Previous output hash
		0xff, 0xff, 0xff, 0xff, // Prevous output index
		0x07,                                     // Varint for length of signature script
		0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, // Signature script
		0xff, 0xff, 0xff, 0xff, // Sequence
		0x01,                                           // Varint for number of output transactions
		0x00, 0xf2, 0x05, 0x2a, 0x01, 0x00, 0x00, 0x00, // Transaction amount
		0x43, // Varint for length of pk script
		0x41, // OP_DATA_65
		0x04, 0x96, 0xb5, 0x38, 0xe8, 0x53, 0x51, 0x9c,
		0x72, 0x6a, 0x2c, 0x91, 0xe6, 0x1e, 0xc1, 0x16,
		0x00, 0xae, 0x13, 0x90, 0x81, 0x3a, 0x62, 0x7c,
		0x66, 0xfb, 0x8b, 0xe7, 0x94, 0x7b, 0xe6, 0x3c,
		0x52, 0xda, 0x75, 0x89, 0x37, 0x95, 0x15, 0xd4,
		0xe0, 0xa6, 0x04, 0xf8, 0x14, 0x17, 0x81, 0xe6,
		0x22, 0x94, 0x72, 0x11, 0x66, 0xbf, 0x62, 0x1e,
		0x73, 0xa8, 0x2c, 0xbf, 0x23, 0x42, 0xc8, 0x58,
		0xee,                   // 65-byte signature
		0xac,                   // OP_CHECKSIG
		0x00, 0x00, 0x00, 0x00, // Lock time
	}

	r := bytes.NewReader(buf)
	var tx MsgTx
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		err = tx.Deserialize(r)
		if err != nil {
			log.Errorf("failed to deserialize data,err:%v", err)
		}
	}
}

// BenchmarkDeserializeTxLarge performs a benchmark on how long it takes to
// deserialize a very large transaction.
func BenchmarkDeserializeTxLarge(b *testing.B) {
	// tx bb41a757f405890fb0f5856228e23b715702d714d59bf2b1feb70d8b2b4e3e08
	// from the main block chain.
	fi, err := os.Open("testdata/megatx.bin.bz2")
	if err != nil {
		if fi != nil {
			err = fi.Close()
			if err != nil {
				log.Errorf("failed to close file,err:%v", err)
			}
		}

		b.Fatalf("Failed to read transaction data: %v", err)
	}
	defer fi.Close()
	buf, err := ioutil.ReadAll(bzip2.NewReader(fi))
	if err != nil {
		b.Fatalf("Failed to read transaction data: %v", err)
	}

	r := bytes.NewReader(buf)
	var tx MsgTx
	for i := 0; i < b.N; i++ {
		_, err = r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		err = tx.Deserialize(r)
		if err != nil {
			log.Errorf("failed to deserialize data,err:%v", err)
		}
	}
}

// BenchmarkSerializeTx performs a benchmark on how long it takes to serialize
// a transaction.
func BenchmarkSerializeTx(b *testing.B) {
	tx := blockOne.Body.Transactions[0]
	for i := 0; i < b.N; i++ {
		err := tx.Serialize(ioutil.Discard)
		if err != nil {
			log.Errorf("failed to serialize data,err:%v", err)
		}
	}
}

// BenchmarkReadBlockHeader performs a benchmark on how long it takes to
// deserialize a block header.
func BenchmarkReadBlockHeader(b *testing.B) {
	buf := []byte{
		0x01, 0x00, 0x00, 0x00, // Version 1
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
		0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, // PrevBlock
		0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
		0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
		0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
		0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a, // MerkleRoot
		0x29, 0xab, 0x5f, 0x49, // Timestamp
		0xff, 0xff, 0x00, 0x1d, // Bits
		0xf3, 0xe0, 0x01, 0x00, // Nonce
		0x00, // TxnCount Varint
	}
	r := bytes.NewReader(buf)
	var header BlockHeader
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seek,err:%v", err)
		}
		err = readBlockHeader(r, 0, &header)
		if err != nil {
			log.Errorf("failed to read block header,err:%v", err)
		}
	}
}

// BenchmarkWriteBlockHeader performs a benchmark on how long it takes to
// serialize a block header.
func BenchmarkWriteBlockHeader(b *testing.B) {
	header := blockOne.Header
	for i := 0; i < b.N; i++ {
		err := writeBlockHeader(ioutil.Discard, 0, &header)
		if err != nil {
			log.Errorf("failed to write block header,err:%v", err)
		}
	}
}

// BenchmarkDecodeAddr performs a benchmark on how long it takes to decode an
// addr message with the maximum number of addresses.
func BenchmarkDecodeAddr(b *testing.B) {
	// Create a message with the maximum number of addresses.
	pver := ProtocolVersion
	ip := net.ParseIP("127.0.0.1")
	ma := NewMsgAddr()
	for port := uint16(0); port < MaxAddrPerMsg; port++ {
		err := ma.AddAddress(NewNetAddressIPPort(ip, port, SFNodeNetwork))
		if err != nil {
			log.Errorf("failed to add address,err:%v", err)
		}
	}

	// Serialize it so the bytes are available to test the decode below.
	var bb bytes.Buffer
	if err := ma.BtcEncode(&bb, pver, LatestEncoding); err != nil {
		b.Fatalf("MsgAddr.BtcEncode: unexpected error: %v", err)
	}
	buf := bb.Bytes()

	r := bytes.NewReader(buf)
	var msg MsgAddr
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Seek(0, 0)
		if err != nil {
			log.Errorf("failed to seed,err:%v", err)
		}
		err = msg.BtcDecode(r, pver, LatestEncoding)
		if err != nil {
			log.Errorf("failed to decode message,err:%v", err)
		}
	}
}

// BenchmarkTxHash performs a benchmark on how long it takes to hash a
// transaction.
func BenchmarkTxHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		genesisCoinbaseTx.TxHash()
	}
}

// BenchmarkDoubleHashB performs a benchmark on how long it takes to perform a
// double hash returning a byte slice.
func BenchmarkDoubleHashB(b *testing.B) {
	var buf bytes.Buffer
	if err := genesisCoinbaseTx.Serialize(&buf); err != nil {
		b.Errorf("Serialize: unexpected error: %v", err)
		return
	}
	txBytes := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chainhash.DoubleHashB(txBytes)
	}
}

// BenchmarkDoubleHashH performs a benchmark on how long it takes to perform
// a double hash returning a chainhash.Hash.
func BenchmarkDoubleHashH(b *testing.B) {
	var buf bytes.Buffer
	if err := genesisCoinbaseTx.Serialize(&buf); err != nil {
		b.Errorf("Serialize: unexpected error: %v", err)
		return
	}
	txBytes := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chainhash.DoubleHashH(txBytes)
	}
}
