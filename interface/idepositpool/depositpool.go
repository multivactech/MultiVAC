// Copyright (c) 2018-present, MultiVAC Foundation
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package idepositpool

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// DepositPool used to manage deposit outs.
type DepositPool interface {
	Add(out *wire.OutPoint, height wire.BlockHeight, proof *merkle.MerklePath) error
	Remove(out *wire.OutPoint) error
	Update(update *state.Update, shard shard.Index, height wire.BlockHeight) error
	GetBiggest(address multivacaddress.Address) ([]byte, error)
	GetAll(shard shard.Index, address multivacaddress.Address) ([]*wire.OutWithProof, error)
	Verify(address multivacaddress.Address, proof []byte) bool
	Lock(out *wire.OutPoint) error
}
