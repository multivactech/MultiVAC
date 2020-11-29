// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// TestBlockHeader tests the BlockHeader API.
func TestBlockHeader(t *testing.T) {
	hash := mainNetGenesisHash
	merkleHash := merkle.MerkleHash{}
	bh := NewBlockHeader(1, shard.Index(1), int64(1), &hash, &hash, &merkleHash, &merkleHash, nil, nil)

	// Ensure we get the same data back out.
	if bh.PrevBlockHeader != hash {
		t.Errorf("NewBlockHeader: wrong prev block hash - got %v, want %v",
			spew.Sprint(bh.PrevBlockHeader), spew.Sprint(hash))
	}

	if bh.BlockBodyHash != hash {
		t.Errorf("NewBlockHeader: wrong block body hash - got %v, want %v",
			spew.Sprint(bh.BlockBodyHash), spew.Sprint(hash))
	}

	if bh.OutsMerkleRoot != merkleHash {
		t.Errorf("NewBlockHeader: wrong outs merkle root hash - got %v, want %v",
			spew.Sprint(bh.OutsMerkleRoot), spew.Sprint(merkleHash))
	}

	if bh.ShardLedgerMerkleRoot != merkleHash {
		t.Errorf("NewBlockHeader: wrong shard ledger merkle root - got %v, want %v",
			spew.Sprint(bh.ShardLedgerMerkleRoot), spew.Sprint(merkleHash))
	}
}

// TestBlockHeaderWire tests the BlockHeader wire encode and decode for various
// protocol versions.
func TestBlockHeaderWire(t *testing.T) {
	// baseBlockHdr is used in the various tests as a baseline BlockHeader.
	baseBlockHdr := &BlockHeader{
		Version:               1,
		ShardIndex:            shard.Index(1),
		Height:                int64(1),
		PrevBlockHeader:       mainNetGenesisHash,
		BlockBodyHash:         mainNetGenesisMerkleRoot,
		OutsMerkleRoot:        merkle.MerkleHash{},
		ShardLedgerMerkleRoot: merkle.MerkleHash{},
		DepositTxsOuts:        []OutPoint{},
		WithdrawTxsOuts:       []OutPoint{},
		TimeStamp:             int64(1),
		ReshardSeed:           []uint8{},
		Pk:                    []uint8{},
		LastSeedSig:           []uint8{},
		HeaderSig:             []uint8{},
		ReduceActions:         []*ReduceAction{},
	}

	tests := []struct {
		in   *BlockHeader    // Data to encode
		out  *BlockHeader    // Expected decoded data
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding variant to use
	}{
		// Latest protocol version.
		{
			baseBlockHdr,
			baseBlockHdr,
			ProtocolVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		err := writeBlockHeader(&buf, test.pver, test.in)
		if err != nil {
			t.Errorf("writeBlockHeader #%d error %v", i, err)
			continue
		}

		// Decode the block header from wire format.
		var bh BlockHeader
		rbuf := bytes.NewReader(buf.Bytes())
		err = readBlockHeader(rbuf, test.pver, &bh)
		if err != nil {
			t.Errorf("readBlockHeader #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&bh, test.out) {
			t.Errorf("readBlockHeader #%d\n got: %s want: %s", i,
				spew.Sdump(&bh), spew.Sdump(test.out))
			continue
		}
	}
}
