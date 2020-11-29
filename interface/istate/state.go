// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package istate

import (
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// LedgerStateManager is used to manage ledger statue.
type LedgerStateManager interface {

	// LedgerInfo returns the last updated ledger info.
	LedgerInfo() *wire.LedgerInfo

	// LocalLedgerInfo returns the current local ledger.
	LocalLedgerInfo() *wire.LedgerInfo

	// GetLedgerRoot returns state's merkle tree's root hash.
	GetLedgerRoot() merkle.MerkleHash

	// GetShardHeight returns the height of the specific shrad.
	GetShardHeight(shardIndex shard.Index) int64

	// GetNewUpdate returns a temporary StateUpdate instance which can verify tx or block quickly.
	NewUpdate() (*state.Update, error)

	// ApplyUpdate apply a new update to state.
	ApplyUpdate(update *state.Update) error

	// GetUpdateWithFullBlock is similar to UpdateWithFullBlock, but it will not really update the ledger.
	GetUpdateWithFullBlock(block *wire.MsgBlock) (*state.Update, error)

	// GetUpdateWithSlimBlock is similar to GetUpdateWithFullBlock, but it updates with the slimblock rather than fullblock.
	GetUpdateWithSlimBlock(slimblock *wire.SlimBlock) (*state.Update, error)

	// It returns the available header according to the given ledgerInfo.
	GetBlockHeadersToAppend(ledgerInfo *wire.LedgerInfo) ([]*wire.BlockHeader, error)
}
