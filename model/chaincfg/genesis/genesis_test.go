/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package genesis

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestGenesisBlock(t *testing.T) {
	shard := shard.ShardList[0]
	tx := GenerateGenesisTx(shard)
	txHash := tx.TxHash()
	setTotalAmountOfEachShard()

	value := new(big.Int).Div(TotalAmountOfEachShard, big.NewInt(int64(NumberOfOutsInEachGenesisBlock)))
	outState := &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          txHash,
			Index:           10,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shard], multivacaddress.UserAddress),
			Data:            wire.MtvValueToData(value),
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.Transfer,
		},
		State: wire.StateUnused,
	}

	outsHash := merkle.MerkleHash(sha256.Sum256(outState.ToBytesArray()))

	tree := GenesisBlocksOutsMerkleTrees[shard]
	outsMerklePath, _ := tree.GetMerklePath(&outsHash)

	if err := outsMerklePath.Verify(tree.MerkleRoot); err != nil {
		t.Errorf(err.Error())
	}
}
