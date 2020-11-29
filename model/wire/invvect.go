// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

const (
	// InvWitnessFlag denotes that the inventory vector type is requesting,
	// or sending a version which includes witness data.
	InvWitnessFlag = 1 << 30
)

// InvType represents the allowed types of inventory vectors.  See InvVect.
type InvType uint32

// These constants define the various supported inventory vector types.
const (
	InvTypeError                InvType = 0
	InvTypeTx                   InvType = 1
	InvTypeBlock                InvType = 2
	InvTypeFilteredBlock        InvType = 3
	InvTypeMultiVac             InvType = 4
	InvTypeWitnessBlock                 = InvTypeBlock | InvWitnessFlag
	InvTypeWitnessTx                    = InvTypeTx | InvWitnessFlag
	InvTypeFilteredWitnessBlock         = InvTypeFilteredBlock | InvWitnessFlag
)

// Map of service flags back to their constant names for pretty printing.
var ivStrings = map[InvType]string{
	InvTypeError:                "ERROR",
	InvTypeTx:                   "MSG_TX",
	InvTypeBlock:                "MSG_BLOCK",
	InvTypeFilteredBlock:        "MSG_FILTERED_BLOCK",
	InvTypeMultiVac:             "MSG_MULTIVAC",
	InvTypeWitnessBlock:         "MSG_WITNESS_BLOCK",
	InvTypeWitnessTx:            "MSG_WITNESS_TX",
	InvTypeFilteredWitnessBlock: "MSG_FILTERED_WITNESS_BLOCK",
}

// String returns the InvType in human-readable form.
func (invtype InvType) String() string {
	if s, ok := ivStrings[invtype]; ok {
		return s
	}

	return fmt.Sprintf("Unknown InvType (%d)", uint32(invtype))
}

// InvVect defines a bitcoin inventory vector which is used to describe data,
// as specified by the Type field, that a peer wants, has, or does not have to
// another peer.
type InvVect struct {
	Type InvType        // Type of data
	Hash chainhash.Hash // Hash of the data
}

// NewInvVect returns a new InvVect using the provided type and hash.
func NewInvVect(typ InvType, hash *chainhash.Hash) *InvVect {
	return &InvVect{
		Type: typ,
		Hash: *hash,
	}
}

// readInvVect reads an encoded InvVect from r depending on the protocol
// version.
func readInvVect(r io.Reader, pver uint32, iv *InvVect) error {
	return readElements(r, &iv.Type, &iv.Hash)
}

// writeInvVect serializes an InvVect to w depending on the protocol version.
func writeInvVect(w io.Writer, pver uint32, iv *InvVect) error {
	return writeElements(w, iv.Type, &iv.Hash)
}
