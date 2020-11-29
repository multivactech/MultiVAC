/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package internal

import (
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/icontroller"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// ThreadedShardManager is used to manage the state of each shard, it detects the
// incoming confirmation message and processes the message if it match
// re-shard condition.
type ThreadedShardManager struct {
	worker   *ShardManager
	actorCtx *message.ActorContext
}

// add adds a new shardController to the shard manager.
func (tsm *ThreadedShardManager) add(sp *shardController) {
	tsm.actorCtx.Send(tsm.worker, message.NewEvent(evtAdd, sp), nil)
}

// StartEnabledShardControllers triggers the start of all enabled shards controllers.
func (tsm *ThreadedShardManager) StartEnabledShardControllers() {
	tsm.actorCtx.Send(tsm.worker, message.NewEvent(evtStartProcessor, nil), nil)
}

// Init initializes the shard manager.
func (tsm *ThreadedShardManager) Init() {
	tsm.actorCtx.Send(tsm.worker, message.NewEvent(evtInit, nil), nil)
}

// GetEnableShards returns the enabled shard index array.
func (tsm *ThreadedShardManager) GetEnableShards() []shard.Index {
	resp := tsm.actorCtx.SendAndWait(tsm.worker, message.NewEvent(evtGetEnableShards, nil)).(*enableShardsResp)
	return resp.enableShards
}

// GetShardController returns a relevant shard controller with given shard index.
func (tsm *ThreadedShardManager) GetShardController(shardIndex shard.Index) icontroller.IShardController {
	resp := tsm.actorCtx.SendAndWait(tsm.worker, message.NewEvent(evtGetProcessor, shardIndex)).(*shardController)
	return resp
}

// GetAllShardControllers returns all shard controllers.
func (tsm *ThreadedShardManager) GetAllShardControllers() []icontroller.IShardController {
	resp := tsm.actorCtx.SendAndWait(tsm.worker, message.NewEvent(evtGetAllProcessor, nil)).([]*shardController)
	r := make([]icontroller.IShardController, len(resp))
	for i := range resp {
		r[i] = resp[i]
	}
	return r
}

// ProvideShardManager creates an instance of ThreadedShardManager
func ProvideShardManager(cfg *config.Config, info *ValidationInfo, metrics *metrics.Metrics,
	blockChain iblockchain.BlockChain, depositPool idepositpool.DepositPool, heartbeatMgr iheartbeat.HeartBeat,
	connector connection.Connector) *ThreadedShardManager {
	tsm := newShardManager(cfg, info)

	// For every shard, create shard processor
	for _, shardIdx := range shard.ShardList {
		if !cfg.IsSharded && shardIdx != DefaultShard {
			continue
		}
		rootCtrlLogger.Infof("Create shard %v", shardIdx)
		tmpShardProcessor, err := newShardController(
			info,
			shardIdx,
			blockChain,
			depositPool,
			cfg,
			heartbeatMgr,
			metrics,
		)
		if err != nil {
			rootCtrlLogger.Errorf("Error in creating shardController for shard: %v", shardIdx)
			return nil
		}
		tsm.add(tmpShardProcessor)
	}

	// todo(MH): hot fix bug for new peer
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			connector.UpdatePeerInfo(tsm.GetEnableShards())
		}
	}()

	return tsm
}

// handleConfirmationMessage handles MsgBlockConfirmation from other peers.
func (tsm *ThreadedShardManager) handleConfirmationMessage(msg *wire.MsgBlockConfirmation) {
	tsm.actorCtx.Send(tsm.worker, message.NewEvent(evtHandleMessage, msg), nil)
}

// TODO(huangsz): plug in DI for ShardManager
func newShardManager(cfg *config.Config, info *ValidationInfo) *ThreadedShardManager {
	tsm := new(ThreadedShardManager)
	tsm.worker = newShardManagerImpl(cfg, info)
	tsm.actorCtx = message.NewActorContext()
	tsm.actorCtx.StartActor(tsm.worker)
	if connection.GlobalConnServer != nil {
		msgChannel := make(chan *connection.MessageAndReply)
		var tag []connection.Tag
		for _, shardIndex := range shard.ShardList {
			tag = append(tag, connection.Tag{Msg: wire.CmdMsgBlockConfirmation, Shard: shardIndex})
		}
		connection.GlobalConnServer.RegisterChannels(&connection.MessagesAndReceiver{
			Tags:     tag,
			Channels: msgChannel,
		})

		go func() {
			for msg := range msgChannel {
				tsm.handleConfirmationMessage(msg.Msg.(*wire.MsgBlockConfirmation))
			}
		}()
	}
	return tsm
}
