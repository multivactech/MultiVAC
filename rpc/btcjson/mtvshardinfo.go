package btcjson

import "github.com/multivactech/MultiVAC/model/shard"

// ShardInfo defines a struct that contains a shard index and its state.
// TODO Maybe move this to another MTV specific package
type ShardInfo struct {
	Index   shard.Index
	Enabled bool
}

// ShardsInfo contains the a list of shard, it is used as the response of queryshardinfo rpc.
type ShardsInfo struct {
	NumShards int
	Shards    []ShardInfo
}

// Len is the number of elements in the collection. Implements sort.Interface
func (s *ShardsInfo) Len() int {
	return s.NumShards
}

// Less reports whether the element with
// index i should sort before the element with index j. Implements sort.Interface
func (s *ShardsInfo) Less(i, j int) bool {
	return s.Shards[i].Index.GetID() < s.Shards[j].Index.GetID()
}

// Swap swaps the elements with indexes i and j. Implements sort.Interface
func (s *ShardsInfo) Swap(i, j int) {
	tmp := s.Shards[i]
	s.Shards[i] = s.Shards[j]
	s.Shards[j] = tmp
}
