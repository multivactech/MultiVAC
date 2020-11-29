/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package internal

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/prometheus/client_golang/prometheus"
)

// reshardState is the state of re-sharding information.
type reshardState struct {
	height  int64
	enabled bool
	proof   []byte
}

// preInfo is the last information of resharding.
type preInfo struct {
	seed   []byte
	height int64
}

// ShardManager is used to manage the state of each shard, it detects the
// incoming confirmation message and processes the message if it match
// re-shard condition.
//
// note, it doesn't support concurrency processing, that means when
// a shard begins re-shard others should wait for a moment. Considering
// that it only modifies the state of shard without causing any blocking
// and usually the reshard period is longer, so we have serialized it to
// avoid concurrency security problems.
type ShardManager struct {
	// All shard's processor
	shardProcessors map[shard.Index]*shardController
	// Matain the state of all shards
	shardStatus map[shard.Index]*reshardState
	// lastMessage used to filter new message
	lastMessage map[shard.Index]*wire.MsgBlockConfirmation
	// preInfo is the information of last reshard information
	preInfo map[shard.Index]*preInfo
	// executeable message can be put into the queue
	msgQueue       chan *wire.MsgBlockConfirmation
	validationInfo *ValidationInfo
	cfg            *config.Config
	logger         btclog.Logger
}

// newShardManagerImpl returns the new instance of ShardManager.
func newShardManagerImpl(cfg *config.Config, info *ValidationInfo) *ShardManager {
	sm := &ShardManager{
		shardProcessors: make(map[shard.Index]*shardController),
		shardStatus:     make(map[shard.Index]*reshardState),
		lastMessage:     make(map[shard.Index]*wire.MsgBlockConfirmation),
		preInfo:         make(map[shard.Index]*preInfo),
		msgQueue:        make(chan *wire.MsgBlockConfirmation),
		cfg:             cfg,
		validationInfo:  info,
	}
	sm.setLogger()
	return sm
}

// handleConfirmationMsg used to change the state of re-sharding.
//
// The re-sharding cycle mainly has the following steps:
// 1. Listening to confirmation messages and the same message only to be resolved once
// 2. Once the reshard seed change is detected, enter the preparation round
// 3. If the shard is in preparation round and re-shard height satisfied, reshard
func (sm *ShardManager) handleConfirmationMsg(msg *wire.MsgBlockConfirmation) {
	if sm.cfg.StorageNode {
		return
	}
	sm.logger.Debugf("Shard-%d receive a confirmation message height is %d", msg.ShardIndex, msg.Header.Height)
	// Check if is in re-shard round
	if msg.Header.Height <= params.ReshardRoundThreshold {
		return
	}

	shardIndex := msg.ShardIndex
	preSeed := sm.preInfo[shardIndex].seed
	preHeight := sm.preInfo[shardIndex].height

	// For new node, cache the current seed and wait next reshard round
	if !sm.cfg.IsOneOfFirstMiners && len(preSeed) == 0 {
		sm.logger.Infof("New node, waiting for the next re-sharding round, please wait some minutes...")
		sm.preInfo[shardIndex].seed = msg.Header.ReshardSeed
		sm.preInfo[shardIndex].height = msg.Header.Height
		return
	}
	// Enter the ready state when a change in seed and height is detected
	if !bytes.Equal(preSeed, msg.Header.ReshardSeed) && msg.Header.Height > preHeight {
		sm.prepare(msg)
		sm.preInfo[shardIndex].seed = msg.Header.ReshardSeed
		sm.preInfo[shardIndex].height = msg.Header.Height
	} else {
		sm.logger.Debugf("Cant't reshard preSeed: %v, curSeed: %v, preHeight: %d, curHeight: %d", preSeed, msg.Header.ReshardSeed, preHeight, msg.Header.Height)
	}

	// When the current state is in the preparation round and the condition of
	// the re-shard is highly satisfied, start to reshard.
	prepRound := int64(params.ReshardPrepareRoundThreshold)
	if msg.Header.Height == (sm.shardStatus[shardIndex].height + prepRound) {
		sm.reshard(msg.Header.ShardIndex)
	}

}

// prepare 处理重分片的准备轮。
// 重分片具体逻辑见：https://docs.google.com/document/d/1z3OjIi60wWc7juuCaFKlqsJ8brWctesjpxbWXnrGPK0/edit#heading=h.wg9hqaeq50tu
func (sm *ShardManager) prepare(msg *wire.MsgBlockConfirmation) {
	sm.logger.Debugf("Shard-%d start to prepare for re-shard", msg.ShardIndex.GetID())
	sp := sm.shardProcessors[msg.ShardIndex]
	isInShard, proof := sm.isInShard(msg.Header.ShardIndex, msg.Header.ReshardSeed)
	sm.shardStatus[msg.ShardIndex] = &reshardState{
		height:  msg.Header.Height,
		enabled: isInShard,
		proof:   proof,
	}

	// Don't service for this shard
	if !isInShard {
		nextStopRound := int(msg.Header.Height + params.ReshardPrepareRoundThreshold - 1)
		sp.SuspendService(nextStopRound)
		sm.logger.Infof("miner will not serve for shard:%d, stop consensus at round: %d", msg.ShardIndex.GetID(), nextStopRound)
	}

	// Will service for this shard
	if isInShard {
		if !sp.IsEnabled() {
			startRound := int(sm.shardStatus[msg.ShardIndex].height + params.ReshardPrepareRoundThreshold - 1)
			sm.logger.Infof("Miner will serve for this shard %d %v", msg.ShardIndex.GetID(), startRound)
			sp.StartService(startRound, msg.Header, sm.getEnableShards())
		} else {
			sm.logger.Debugf("miner is already serve for this shard")
		}
	}
}

// reshard 处理重分片时真正起作用时的逻辑(开始轮)
func (sm *ShardManager) reshard(shardIndex shard.Index) {
	sm.logger.Debugf("Shard-%v start to reshard", shardIndex)
	if sm.shardStatus[shardIndex].enabled {
		sm.logger.Debugf("Shard-%v enable", shardIndex)
		sp := sm.shardProcessors[shardIndex]
		sp.UpdateProof(sm.shardStatus[shardIndex].proof)
	}

	sp := sm.shardProcessors[shardIndex]
	if !sm.shardStatus[shardIndex].enabled {
		sm.logger.Debugf("Shard-%v disable", shardIndex)
		time.AfterFunc(5*time.Second, func() {
			sp.StopService()
		})
	}
}

func (sm *ShardManager) getEnableShards() []shard.Index {
	var result []shard.Index
	for shard, sp := range sm.shardProcessors {
		if sp.IsEnabled() {
			result = append(result, shard)
		}
	}
	return result
}

func (sm *ShardManager) isInShard(shardIndex shard.Index, seed []byte) (bool, []byte) {
	info := sm.validationInfo
	proof, err := info.Vrf.Generate(info.Pk, info.Sk, seed)
	if err != nil {
		return false, nil
	}

	// TODO: use real deposit
	fakeDepositUnit := uint32(1)
	isInShard := info.Selector.IsInShard(proof, fakeDepositUnit, shardIndex)
	return isInShard, proof
}

// TODO:
// msgValidate processes incoming confirmation message, if the message is legal
// and has never been received, put it into msgQueue and wait for processing.
// Validating message can take a lot of time.
//
// note, this is concurrency safe and be care of goroutine overflow.
func (sm *ShardManager) msgValidate(msg *wire.MsgBlockConfirmation) {
	old, ok := sm.getLastMessage(msg.ShardIndex)
	if ok {
		if old.Header.IsEqualTo(msg.Header) || msg.Header.Height <= old.Header.Height {
			return
		}
	}
	if err := msg.IsValidated(); err != nil {
		sm.logger.Debugf("ShardManager receive a invalid confirmation: %v", err)
		return
	}
	sm.setLastMessage(msg)
	sm.handleConfirmationMsg(msg)
}

// Add adds a new shardController to it's processor map, if
// the processor already exists, return error.
func (sm *ShardManager) Add(sp *shardController) {
	// Check if sp exists
	shardIndex := sp.ShardIndex()
	if _, ok := sm.shardProcessors[shardIndex]; ok {
		return
	}
	// Set default status for new shard
	state := &reshardState{
		height:  0,
		enabled: false,
		proof:   []byte{},
	}
	sm.shardStatus[shardIndex] = state
	sm.shardProcessors[shardIndex] = sp
	sm.logger.Debugf("Add shardController to ShardManager, shard is %d", sp.ShardIndex())
}

func (sm *ShardManager) getProcessorByIndex(shardIndex shard.Index) *shardController {
	if sp, ok := sm.shardProcessors[shardIndex]; ok {
		return sp
	}
	return nil
}

func (sm *ShardManager) getAllProcessors() []*shardController {
	var sps []*shardController
	for _, sp := range sm.shardProcessors {
		sps = append(sps, sp)
	}
	return sps
}

// StartEnabledShardControllers start all enabled shardProcessors
func (sm *ShardManager) StartEnabledShardControllers() {
	for _, sp := range sm.shardProcessors {
		if sp.IsEnabled() {
			sp.Start(true)
		}
		if !sp.isStandaloneStorageNode {
			sp.minerPcr.PreStartABCI()
		}
		go func(s *shardController) {
			//time.Sleep(time.Duration(params.SendHeartbeatIntervalS) * 6 * time.Second)
			for config.GlobalConfig().IsOneOfFirstMiners && atomic.LoadInt32(&config.StartNet) == 0 {
				time.Sleep(time.Second)
			}
			if !s.isStandaloneStorageNode && s.IsEnabled() {
				s.minerPcr.Start(true)
			}
			s.bcProducer.Start()
		}(sp)
	}
}

func (sm *ShardManager) init() {
	for shardIndex := range sm.shardProcessors {
		if sm.cfg.StorageNode {
			sm.initShardAsStorageNode(shardIndex)
		} else {
			sm.initShardAsMinerNode(shardIndex)
		}
	}
	// 开启监控
	go sm.monitorData()
}

func (sm *ShardManager) initShardAsStorageNode(index shard.Index) {
	sm.preInfo[index] = nil

	// If the shard should be enabled by default during init
	enable := false
	// storage node is enabled for all shards by default
	if !sm.cfg.IsSharded || !params.StorageNodeShardConfig {
		enable = true
	} else {
		// If the storage node is configured to serve the specified shard,
		// then these shards will be set to the boot state.
		for _, id := range sm.cfg.Shards {
			if index.GetID() == id {
				enable = true
				break
			}
		}
	}
	sm.shardProcessors[index].Init(enable)
}

func (sm *ShardManager) initShardAsMinerNode(index shard.Index) {
	sm.preInfo[index] = &preInfo{
		seed:   []byte{},
		height: 0,
	}
	sp := sm.shardProcessors[index]

	// If the shard should be enabled by default during init
	enable := false
	// If it is the first batch of nodes, there are no other miners in the system.
	// There is no reshard round in the system, so start directly.
	if sm.cfg.IsOneOfFirstMiners {
		isInshard, proof := sm.isInShard(index, []byte(sp.String()))
		if isInshard {
			sp.UpdateProof(proof)
			enable = true
		}
	}
	sp.Init(enable)
}

func (sm *ShardManager) getLastMessage(shardIndex shard.Index) (*wire.MsgBlockConfirmation, bool) {
	msg, ok := sm.lastMessage[shardIndex]
	return msg, ok
}

func (sm *ShardManager) setLastMessage(msg *wire.MsgBlockConfirmation) {
	sm.lastMessage[msg.ShardIndex] = msg
}

func (sm *ShardManager) setLogger() {
	logBackend := logger.BackendLogger()
	sm.logger = logBackend.Logger("SM")
	sm.logger.SetLevel(logger.SmLogLevel)
}

func (sm *ShardManager) monitorData() {
	metric := metrics.Metric
	for {
		// 监控采集enable状态的分片数目
		metric.EnableShardsNum.Set(float64(len(sm.getEnableShards())))

		for shardIdx, shardCtrl := range sm.shardProcessors {
			if shardCtrl.enabled {
				if !shardCtrl.isStandaloneStorageNode {
					// 监控采集enable状态的矿工个数
					metric.ShardEnableMinerNum.With(
						prometheus.Labels{"shard_id": fmt.Sprint(shardIdx)},
					).Set(float64(1))
				}
				// 监控采集ShardPresence状态
				metric.ShardPresence.With(
					prometheus.Labels{"shard_id": fmt.Sprint(shardIdx)},
				).Set(float64(1))
			} else {
				if !shardCtrl.isStandaloneStorageNode {
					// 监控采集enable状态的矿工个数
					metric.ShardEnableMinerNum.With(
						prometheus.Labels{"shard_id": fmt.Sprint(shardIdx)},
					).Set(float64(0))
				}
				// 监控采集ShardPresence状态
				metric.ShardPresence.With(
					prometheus.Labels{"shard_id": fmt.Sprint(shardIdx)},
				).Set(float64(0))
			}

		}
		time.Sleep(5 * time.Second)
	}

}

// Act is the function for implements the actor-pattern interface.
func (sm *ShardManager) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtAdd:
		sm.Add(e.Extra.(*shardController))
	case evtHandleMessage:
		msg := e.Extra.(*wire.MsgBlockConfirmation)
		sm.msgValidate(msg)
	case evtStartProcessor:
		sm.StartEnabledShardControllers()
	case evtInit:
		sm.init()
	case evtGetEnableShards:
		resp := &enableShardsResp{sm.getEnableShards()}
		callback(resp)
	case evtGetProcessor:
		resp := sm.getProcessorByIndex(e.Extra.(shard.Index))
		callback(resp)
	case evtGetAllProcessor:
		resp := sm.getAllProcessors()
		callback(resp)
	default:
	}
}
