// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// MaxBlockHeaderPayload is set to be 1/2000 of a whole block size at max.
// Note that this wasn't decided with a scientific calculation.
const (
	MaxBlockHeaderPayload = MaxBlockPayload / 2000
)

var (
	emptyHeaderSig = make([]byte, 0)
)

// BlockHeader implements the Message interface and represents a MultiVAC
// block header message.  It is used to broadcast block header when a new
// block is accepted.
type BlockHeader struct {
	Version               int32
	ShardIndex            shard.Index
	Height                int64
	PrevBlockHeader       chainhash.Hash
	BlockBodyHash         chainhash.Hash
	OutsMerkleRoot        merkle.MerkleHash
	ShardLedgerMerkleRoot merkle.MerkleHash
	IsEmptyBlock          bool
	TimeStamp             int64
	// TODO: Add seed into block header.
	// TODO: Add proof that the node which created this block is eligible to create it.
	DepositTxsOuts  []OutPoint
	WithdrawTxsOuts []OutPoint

	// ReshardSeed is used to indicate re-sharding.
	// Update ReshardSeed if relevant shard needs to do re-sharding; otherwise, it remains the same with
	// the prevHeader.
	ReshardSeed   []byte
	Pk            []byte
	LastSeedSig   []byte //Use to verify seed
	HeaderSig     []byte
	Seed          chainhash.Hash //redunt,can calculate from LastSeedSig,Round
	ReduceActions []*ReduceAction
}

func (h *BlockHeader) String() string {
	return fmt.Sprintf("{"+
		"Version:%d, Index:%v, Height:%v, BlockHeaderHash:%v, BlockBodyHash:%v, "+
		"PrevBlockHash:%v, IsEmptyBlock:%v, TimeStamp:%d, ReshardSeed:%v, Seed:%v, MerkleRoot:%v, Pk:%v, HeaderSig:%v }",
		h.Version,
		h.ShardIndex.GetID(),
		h.Height,
		h.BlockHeaderHash(),
		h.BlockBodyHash,
		h.PrevBlockHeader,
		h.IsEmptyBlock,
		h.TimeStamp,
		h.ReshardSeed,
		h.Seed,
		h.ShardLedgerMerkleRoot,
		h.Pk,
		h.HeaderSig)
}

// BlockHeaderHash computes the block identifier hash for the given block header.
// TODO:(guotao) use more grace way to hash without HeaderSig
func (h BlockHeader) BlockHeaderHash() chainhash.Hash {
	// Encode the header and double sha256 everything prior to the number of
	// transactions.  Ignore the error returns since there is no way the
	// encode could fail except being out of memory which would cause a
	// run-time panic.
	//h changed from pointer receiver to value receiver with a copy value
	//does not change the value of the original h
	buf := bytes.NewBuffer([]byte{})
	h.HeaderSig = emptyHeaderSig
	_ = writeBlockHeader(buf, 0, &h)

	return chainhash.HashH(buf.Bytes())
}

// IsEqualTo compares two blockheads to see if they are equal
func (h *BlockHeader) IsEqualTo(header *BlockHeader) bool {
	return reflect.DeepEqual(h, header)
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
// See Deserialize for decoding block headers stored to disk, such as in a
// database, as opposed to decoding block headers from the wire.
func (h *BlockHeader) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readBlockHeader(r, pver, h)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
// See Serialize for encoding block headers to be stored to disk, such as in a
// database, as opposed to encoding block headers for the wire.
func (h *BlockHeader) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeBlockHeader(w, pver, h)
}

// Deserialize decodes a block header from r into the receiver using a format
// that is suitable for long-term storage such as a database while respecting
// the Version field.
func (h *BlockHeader) Deserialize(r io.Reader) error {
	// At the current time, there is no difference between the wire encoding
	// at protocol version 0 and the stable long-term storage format.  As
	// a result, make use of readBlockHeader.
	return readBlockHeader(r, 0, h)
}

// Serialize encodes a block header from r into the receiver using a format
// that is suitable for long-term storage such as a database while respecting
// the Version field.
func (h *BlockHeader) Serialize(w io.Writer) error {
	// At the current time, there is no difference between the wire encoding
	// at protocol version 0 and the stable long-term storage format.  As
	// a result, make use of writeBlockHeader.
	return writeBlockHeader(w, 0, h)
}

// Command returns command block header.
func (h *BlockHeader) Command() string {
	return CmdBlockHeader
}

// GetShardIndex returns the shardIndex.
func (h *BlockHeader) GetShardIndex() shard.Index {
	return h.ShardIndex
}

// MaxPayloadLength returns max playload .
func (h *BlockHeader) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockHeaderPayload
}

// NewBlockHeader returns a new BlockHeader using the provided version, previous
// block hash, merkle root hash, difficulty bits, and nonce used to generate the
// block with defaults for the remaining fields.
func NewBlockHeader(version int32, shardIndex shard.Index, height int64,
	prevHash *chainhash.Hash, blockBodyHash *chainhash.Hash,
	outsMerkleRoot *merkle.MerkleHash, shardLedgerMerkleRoot *merkle.MerkleHash, depositTxsOuts []OutPoint, withdrawTxsOuts []OutPoint) *BlockHeader {

	// Limit the timestamp to one second precision since the protocol
	// doesn't support better.
	return &BlockHeader{
		Version:               version,
		ShardIndex:            shardIndex,
		Height:                height,
		PrevBlockHeader:       *prevHash,
		BlockBodyHash:         *blockBodyHash,
		OutsMerkleRoot:        *outsMerkleRoot,
		ShardLedgerMerkleRoot: *shardLedgerMerkleRoot,
		DepositTxsOuts:        depositTxsOuts,
		WithdrawTxsOuts:       withdrawTxsOuts,
	}
}

// readBlockHeader reads a block header from r.  See Deserialize for
// decoding block headers stored to disk, such as in a database, as opposed to
// decoding from the wire.
func readBlockHeader(r io.Reader, pver uint32, bh *BlockHeader) error {
	return rlp.Decode(r, bh)
}

// writeBlockHeader writes a block header to w.  See Serialize for
// encoding block headers to be stored to disk, such as in a database, as
// opposed to encoding for the wire.
func writeBlockHeader(w io.Writer, pver uint32, bh *BlockHeader) error {
	return rlp.Encode(w, bh)
}

// ToBytesArray serialize the block header to byte array.
func (h *BlockHeader) ToBytesArray() []byte {
	ret, err := rlp.EncodeToBytes(h)
	if err != nil {
		return nil
	}
	return ret
}
