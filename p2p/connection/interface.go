package connection

import "github.com/multivactech/MultiVAC/model/shard"

// Connector is used for inject.
type Connector interface {
	UpdatePeerInfo([]shard.Index)
}

// BroadcastParams defines the broadcast params.
type BroadcastParams struct {
	ToNode      toNodeType
	ToShard     shard.Index
	IncludeSelf bool
	//exclPeers []*ConnPeer
	// todo(ymh):seems to unuse from golangci-lint.
	// targetShards []shard.Index
}
