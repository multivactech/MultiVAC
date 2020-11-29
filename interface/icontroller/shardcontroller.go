package icontroller

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// IShardController defines the local interface which serves as the entry for business logic in a particular shard.
type IShardController interface {
	// NotifyNewPeerAdded sends new peer added information to mtvSyncMgr
	NotifyNewPeerAdded(cp *connection.ConnPeer, t config.NodeType)
	// NotifyPeerDone sends the new-done information to syncManager.
	NotifyPeerDone(cp *connection.ConnPeer, t config.NodeType)
	// HandleRPCReq handles a RPCReq for the relevant shard.
	HandleRPCReq(req *message.RPCReq) *message.RPCResp
	// ShardIndex returns the index of the shardController
	ShardIndex() shard.Index
	// IsEnabled returns if the processor is in working.
	IsEnabled() bool
}
