package internal

import (
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/model/shard"
)

var (
	// DefaultShard is the default shard when no sharding is enabled.
	DefaultShard = shard.IDToShardIndex(0)
)

// ValidationInfo wraps the identity validation relevant contributors.
type ValidationInfo struct {
	Selector iconsensus.Sortitionor
	Pk       []byte
	Sk       []byte
	Vrf      vrf.VRF
}
