// Copyright (c) 2018-present, MultiVAC Foundation
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package shard

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
)

const (
// it seems unused.
//shuffleShardEveryNRound = 60
)

// IDToShardIndex returns a MultiVAC Index format from a given number in the supported shard range.
func IDToShardIndex(id uint32) Index {
	if !IsValidShardID(id) {
		panic(fmt.Sprintf("Requested id %d out of number of shards %d.", id, params.GenesisNumberOfShards))
	}
	return Index(id)
}

// IndexToID transfers a Index back to id.
func IndexToID(index Index) uint32 {
	id := index.GetID()
	if !IsValidShardID(id) {
		panic(fmt.Sprintf("Requested id %d out of number of shards %d.", id, params.GenesisNumberOfShards))
	}
	return id
}

// IndexToString convert the shard index to string type.
func IndexToString(index Index) string {
	return strconv.Itoa(int(IndexToID(index)))
}

// IsValidShardID returns whether or not the id is within the valid shard id range.
func IsValidShardID(id uint32) bool {
	return id < params.GenesisNumberOfShards
}

// GetShardIDByPublicHash function has been deprecated, so don't use it in the future.
func GetShardIDByPublicHash(pkHash *multivacaddress.PublicKeyHash) uint32 {
	shardIndex := GetShardIndexByPublicHash(pkHash)
	return shardIndex.GetID()
}

// GetShardIndexByPublicHash function has been deprecated, so don't use it in the future.
func GetShardIndexByPublicHash(pkHash *multivacaddress.PublicKeyHash) Index {
	pkPrefixBytes := pkHash[0:4]
	pkPrefix := binary.BigEndian.Uint32(pkPrefixBytes[:])
	return Index(pkPrefix >> (32 - params.GenesisShardPrefixLength))
}

// Index is the index of shard.
type Index uint32

// MarshalJSON serialization the data to byte slice and then keep it to json.
func (s Index) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint32(s))
}

// UnmarshalJSON deserialization the json data to origin data.
func (s *Index) UnmarshalJSON(data []byte) error {
	var id uint32
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	*s = Index(id)
	return nil
}

// GetID returns the shard index.
func (s Index) GetID() uint32 {
	return uint32(s)
}

// String returns the format string like:Id:shardID.
func (s Index) String() string {
	return fmt.Sprintf("Id: %d", s.GetID())
}

// IndexAndHeight combines the index and height.
type IndexAndHeight struct {
	Index  Index
	Height int64
}

// NewShardIndexAndHeight returns a IndexAndHeight.
func NewShardIndexAndHeight(shard Index, hgt int64) IndexAndHeight {
	return IndexAndHeight{Index: shard, Height: hgt}
}

// String returns the format string for IndexAndHeight.
func (s IndexAndHeight) String() string {
	return fmt.Sprintf("(Shard:%v, Height:%v)", s.Index.GetID(), s.Height)
}
