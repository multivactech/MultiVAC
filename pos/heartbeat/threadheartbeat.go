/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package heartbeat

import (
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// ThreadHeartBeatManager is the wrapper of HearBeatManager.
// This implements the Actor pattern, it is concurrency safe.
type ThreadHeartBeatManager struct {
	cmdQueue chan *connection.MessageAndReply
	worker   *Manager
	actorCtx *message.ActorContext
}

// NewThreadHeartBeatManager returns the instance of ThreadHeartBeatManager. messageChan parameter
// is the channel for heartBeatManager to broadcast message.
func NewThreadHeartBeatManager(pk, sk []byte, dPool idepositpool.DepositPool, metrics *metrics.Metrics) iheartbeat.HeartBeat {
	thbm := &ThreadHeartBeatManager{
		actorCtx: message.NewActorContext(),
	}
	var err error
	thbm.worker, err = newHeartBeatManager(pk, sk, dPool, metrics, connection.GlobalConnServer)
	if err != nil {
		panic(err)
	}
	thbm.cmdQueue = make(chan *connection.MessageAndReply, 1000)
	thbm.actorCtx.StartActor(thbm.worker)
	thbm.registerMsgChannel()
	return thbm
}

func (thbm *ThreadHeartBeatManager) registerMsgChannel() {
	if connection.GlobalConnServer == nil {
		return
	}
	connection.GlobalConnServer.RegisterChannels(&connection.MessagesAndReceiver{
		Tags: []connection.Tag{
			{
				Msg:   wire.CmdHeartBeat,
				Shard: connection.ToAllShard,
			},
		},
		Channels: thbm.cmdQueue,
	})
	go func() {
		for cmd := range thbm.cmdQueue {
			thbm.actorCtx.Send(thbm.worker, message.NewEvent(evtReceiveReq, cmd.Msg), nil)
		}
	}()
}

// Has is used to determine whether the heartbeat information of a given miner has been received.
func (thbm *ThreadHeartBeatManager) Has(pk []byte) bool {
	resp := thbm.actorCtx.SendAndWait(thbm.worker, message.NewEvent(evtHasReq, pk)).(bool)
	return resp
}

// PerceivedCount returns the counts of legal message of heartbeat.
func (thbm *ThreadHeartBeatManager) PerceivedCount() int {
	resp := thbm.actorCtx.SendAndWait(thbm.worker, message.NewEvent(evtPerceivedCountReq, nil)).(int)
	return resp
}

// CheckConfirmation is used to check whether the msg contains enough confirmations.
func (thbm *ThreadHeartBeatManager) CheckConfirmation(msg *wire.MsgBlockConfirmation) bool {
	resp := thbm.actorCtx.SendAndWait(thbm.worker, message.NewEvent(evtCheckConfirmation, msg)).(bool)
	return resp
}

// Start is used to make heartBeatManager work, it will start to send heartbeat message.
func (thbm *ThreadHeartBeatManager) Start() {
	thbm.actorCtx.Send(thbm.worker, message.NewEvent(evtStartReq, nil), nil)
}

// Stop is used to stop sending heartbeat message.
func (thbm *ThreadHeartBeatManager) Stop() {
	thbm.actorCtx.Send(thbm.worker, message.NewEvent(evtStopReq, nil), nil)
}
