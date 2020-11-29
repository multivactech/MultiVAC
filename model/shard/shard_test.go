/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package shard_test

import (
	"testing"

	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestShardIndexAndIdTransfer(t *testing.T) {
	shardId := uint32(3)
	shardIndex := shard.IDToShardIndex(shardId)
	if res := shard.IndexToID(shardIndex); res != shardId {
		t.Errorf("ShardId to Index transfer not match, expected: %v, got: %v", shardId, res)
	}
}

func TestIsValidShardId(t *testing.T) {
	tests := []struct {
		desc     string
		shardId  uint32
		expected bool
	}{
		{
			desc:     "Valid id",
			shardId:  3,
			expected: true,
		},
		{
			desc:    "Invalid id - edge",
			shardId: params.GenesisNumberOfShards,
		},
		{
			desc:    "Invalid id",
			shardId: params.GenesisNumberOfShards + 10,
		},
	}
	for _, test := range tests {
		if res := shard.IsValidShardID(test.shardId); res != test.expected {
			t.Errorf("Test failed for: %v. Expected %v, got %v", test.desc, test.expected, res)
		}
	}
}

func TestGetShardIdByPublicHash(t *testing.T) {
	type args struct {
		address multivacaddress.Address
	}
	p1ChainHash := chainhash.Hash{0xff, 0xff, 0xff, 0xff}
	p2ChainHash := chainhash.Hash{0x7f, 0xff, 0xff, 0xff}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			name: "Shard 3",
			args: args{
				address: multivacaddress.GenerateAddress(signature.PublicKey(p1ChainHash.CloneBytes()),
					multivacaddress.UserAddress),
			},
			want: 2,
		},
		{
			name: "Shard 1",
			args: args{
				address: multivacaddress.GenerateAddress(signature.PublicKey(p2ChainHash.CloneBytes()),
					multivacaddress.UserAddress),
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkAddr := tt.args.address

			pkHash, err := pkAddr.GetPublicKeyHash(multivacaddress.UserAddress)
			if err != nil {
				panic(err)
			}
			if got := shard.GetShardIDByPublicHash(pkHash); got != tt.want {
				t.Errorf("GetShardIDByPublicHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTxShard(t *testing.T) {
	tests := []struct {
		name       string
		tx         *wire.MsgTx
		shardIndex shard.Index
		isInShard  bool
	}{
		{
			name: "TestTxIsInShard",
			tx: &wire.MsgTx{
				Shard: 3,
			},
			shardIndex: 3,
			isInShard:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.tx.IsTxInShard(test.shardIndex) != test.isInShard {
				t.Errorf("ShardIdx check is broken, tx: %v, shard: %v\n", test.tx, test.shardIndex)
			}
		})
	}
}
