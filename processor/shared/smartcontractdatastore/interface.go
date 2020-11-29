/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package smartcontractdatastore

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// SmartContractDataStore is a interface that contains some methods used for getting or updating the smart contract.
type SmartContractDataStore interface {
	AddSmartContractInfo(scInfo *wire.SmartContractInfo, root *merkle.MerkleHash) error
	GetSmartContractInfo(addr multivacaddress.PublicKeyHash) *wire.SmartContractInfo
	RefreshDataStore(update *state.Update, actions []*wire.UpdateAction)
	Reset()
}

// NewSmartContractDataStore returns a new instance of smartContractDataStore.
func NewSmartContractDataStore(shardIdx shard.Index) *SContractDataStore {
	return &SContractDataStore{
		shardIdx:  shardIdx,
		dataStore: make(map[multivacaddress.PublicKeyHash]*wire.SmartContractInfo),
	}
}
