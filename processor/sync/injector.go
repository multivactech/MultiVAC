// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package sync

import (
	di "github.com/google/wire"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/isync"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// NewSyncManager creates a new SyncManager instance.
// shard is the given shard for this SyncManager to work on.
// sn is the StorageNode instance in this shard for storage node, nil for miner node.
// abc is the AppBlockChain instance in this shard for miner node, nil for storage node.
func NewSyncManager(shard shard.Index, sn storageSyncContract, miner minerSyncContract, pubSubMgr *message.PubSubManager) isync.SyncManager {
	if config.GetNodeType() == config.StorageNode {
		return ProvideStorageSyncManager(shard, sn, pubSubMgr)
	}
	return ProvideMinerSyncManager(shard, miner, pubSubMgr)
}

func ProvideStorageSyncManager(shard shard.Index, sn storageSyncContract, pubSubMgr *message.PubSubManager) isync.SyncManager {
	panic(di.Build(di.NewSet(
		provideSyncManager, newStorageWorker,
		di.Bind(new(isync.SyncManager), new(*syncManager)),
		di.Bind(new(syncWorker), new(*storageWorker)))))
}

func ProvideMinerSyncManager(shard shard.Index, miner minerSyncContract, pubSubMgr *message.PubSubManager) isync.SyncManager {
	panic(di.Build(di.NewSet(
		provideSyncManager, newMinerWorker,
		di.Bind(new(isync.SyncManager), new(*syncManager)),
		di.Bind(new(syncWorker), new(*minerWorker)))))
}

func provideSyncManager(shard shard.Index, worker syncWorker, pubSubMgr *message.PubSubManager) *syncManager {
	mgr := &syncManager{
		shard:    shard,
		state:    newSyncState(pubSubMgr),
		worker:   worker,
		actorCtx: message.NewActorContext(),
	}
	worker.setManager(mgr)
	return mgr
}

func provideManager(idx shard.Index, pubSubMgr *message.PubSubManager) *syncManager {
	return &syncManager{
		shard:    idx,
		state:    newSyncState(pubSubMgr),
		actorCtx: message.NewActorContext(),
	}
}
