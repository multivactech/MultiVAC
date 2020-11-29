/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockconfirmproductor

import (
	"fmt"
	"math"

	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

// BcProducerEnabler Define the bcProducer startup interface.
type BcProducerEnabler interface {
	Start()
}

type heartBeat interface {
	Has(pk []byte) bool
	PerceivedCount() int
}

const (
	minerNodeThreshold = 0.6667
	initialRound       = 0
	fakeDepositUnit    = 1
)

var emptyBlockHash = chainhash.Hash{}

type bcProducer struct {
	shardIndex             shard.Index
	threadHeartBeatManager heartBeat

	// register message handle channels and send broadcast messages
	gossipNode connection.GossipNode
	msgQueue   chan *connection.MessageAndReply

	// block header cache
	receivedBlockHeader map[int]map[chainhash.Hash]*wire.BlockHeader
	// pendingConfirmation save MsgBlockConfirmation when reach threshold but have not received header.
	pendingConfirmation map[int]*wire.MsgBlockConfirmation
	selector            iconsensus.Sortitionor
	curRound            int
	bbaFinVoter         *voter
	logger              btclog.Logger
}

// NewBcProducer return a block confirmation message producer interface.
// bcProducer receives enough bba final messages and corresponding headers to generate corresponding confirmation messages
func NewBcProducer(shard shard.Index, heartbeatManager heartBeat, selector iconsensus.Sortitionor) BcProducerEnabler {
	bcp := newBcProducer(shard, heartbeatManager, selector)
	bcp.logger.SetLevel(logger.BCPLogLevel)
	bcp.bbaFinVoter.logger = bcp.logger
	if bcp.gossipNode != nil {
		bcp.gossipNode.RegisterChannels(&connection.MessagesAndReceiver{
			Tags: []connection.Tag{
				{Msg: wire.CmdBinaryBAFin, Shard: shard},
				{Msg: wire.CmdBlockHeader, Shard: shard},
			},
			Channels: bcp.msgQueue,
		})
	}
	return bcp
}

func newBcProducer(shard shard.Index, heartbeatManager heartBeat, selector iconsensus.Sortitionor) *bcProducer {
	log := logBackend.Logger(fmt.Sprintf("BCP%d", shard.GetID()))
	return &bcProducer{
		shardIndex:             shard,
		gossipNode:             connection.GlobalConnServer,
		threadHeartBeatManager: heartbeatManager,
		receivedBlockHeader:    make(map[int]map[chainhash.Hash]*wire.BlockHeader),
		pendingConfirmation:    make(map[int]*wire.MsgBlockConfirmation),
		msgQueue:               make(chan *connection.MessageAndReply, 10000),
		selector:               selector,
		bbaFinVoter:            &voter{bbaHistory: make(map[voteKey]map[pkKey]*wire.MsgBinaryBAFin), logger: log},
		logger:                 log,
	}
}

// Start makes bcProducer in working. note: bcProducer are best started after receiving enough heartbeats.
func (bcp *bcProducer) Start() {
	bcp.setNewRound(initialRound)
	go bcp.handleMessage()
}

func (bcp *bcProducer) handleMessage() {
	for msg := range bcp.msgQueue {
		bcp.onMessage(msg.Msg)
	}
}

func (bcp *bcProducer) onMessage(msg wire.Message) {
	switch m := msg.(type) {
	case *wire.MsgBinaryBAFin:
		if err := bcp.verifyBBAFinMsg(m); err != nil {
			bcp.logger.Debugf("receive bba final message is not validated: %v", err)
			return
		}
		bcp.onBBAFinMessage(m)
	case *wire.BlockHeader:
		bcp.onBlockHeader(m)
	}
}

func (bcp *bcProducer) setNewRound(round int) {
	minConfirmationCount := float64(bcp.threadHeartBeatManager.PerceivedCount()) * params.SecurityLevel * minerNodeThreshold
	threshold := int(math.Ceil(minConfirmationCount)) - 1
	bcp.curRound = round
	bcp.bbaFinVoter.setCurrentRound(round, threshold)
	bcp.cleanOutdatedHeader(round)
}

func (bcp *bcProducer) onBBAFinMessage(message *wire.MsgBinaryBAFin) {
	bcp.logger.Debugf("receive bbaFin message: %v", message)
	round := message.GetRound()
	step := message.GetStep()
	v := message.GetByzAgreementValue()
	b := message.GetBValue()
	if bcp.bbaFinVoter.addVoteWithBBAFinMsg(int(round), int(step), b, &v, message) {
		bcMsg := wire.NewMessageBlockConfirmation(
			bcp.shardIndex,
			int32(round),
			int32(step),
			&v,
			nil,
			bcp.bbaFinVoter.getBBAFinHistory(round, step, v.BlockHash),
		)
		if blockHeader, ok := bcp.getBlockHeader(round, v.BlockHash); ok {
			//broadcast blockconfirmation
			bcMsg.Header = blockHeader
			bcp.logger.Debugf("generate new blockconfirmation: %v", bcMsg)
			bcp.gossipNode.HandleMessage(bcMsg)
			bcp.setNewRound(round + 1)
		} else {
			bcp.logger.Debugf("receive enough BBAFinMsg but have not received block header, "+
				"round %d blockhash:%v", round, v.BlockHash)
			bcp.pendingConfirmation[round] = bcMsg
		}
	}
}

func (bcp *bcProducer) onBlockHeader(message *wire.BlockHeader) {
	round := int(message.Height - 2)
	bcp.logger.Debugf("receive block header of round %v : %v", round, message)
	if bcp.curRound > round {
		bcp.logger.Debugf("receive outdated block header")
		return
	}
	blockHash := message.BlockHeaderHash()
	if message.IsEmptyBlock {
		blockHash = emptyBlockHash
	}
	if _, exist := bcp.receivedBlockHeader[round]; !exist {
		bcp.receivedBlockHeader[round] = make(map[chainhash.Hash]*wire.BlockHeader)
	}
	if _, ok := bcp.receivedBlockHeader[round][blockHash]; !ok {
		if bc, ok := bcp.pendingConfirmation[round]; ok {
			bc.Header = message
			bcp.logger.Debugf("generate new blockconfirmation: %v", bc)
			bcp.gossipNode.HandleMessage(bc)
			bcp.setNewRound(round + 1)
		} else {
			bcp.receivedBlockHeader[round][blockHash] = message
		}
	}
}

func (bcp *bcProducer) getBlockHeader(round int, blockHash chainhash.Hash) (*wire.BlockHeader, bool) {
	if _, exist := bcp.receivedBlockHeader[round]; exist {
		if header, ok := bcp.receivedBlockHeader[round][blockHash]; ok {
			return header, ok
		}
	}
	return nil, false
}

func (bcp *bcProducer) cleanOutdatedHeader(round int) {
	for headerRound := range bcp.receivedBlockHeader {
		if headerRound < round {
			delete(bcp.receivedBlockHeader, headerRound)
		}
	}

	for r := range bcp.pendingConfirmation {
		if r < round {
			delete(bcp.pendingConfirmation, r)
		}
	}
}

// verifyBBAFinMsg determine whether the bba final message the sender belongs to the heartbeat list
// and the signature is legal.
func (bcp *bcProducer) verifyBBAFinMsg(message *wire.MsgBinaryBAFin) error {
	round := message.GetRound()
	if bcp.curRound > round {
		return fmt.Errorf("receive outdated bbaFin message")
	}
	pk := message.GetSignedCredential().Pk
	if !bcp.threadHeartBeatManager.Has(pk) {
		return fmt.Errorf("message sender %v is not in heartbeat list", pk)
	}

	if err := message.IsValidated(); err != nil {
		return fmt.Errorf("receive bba final message %d is not valid, error : %v", message.GetRound(), err)
	}

	if !bcp.selector.IsInShard(message.InShardProof, fakeDepositUnit, bcp.shardIndex) {
		return fmt.Errorf("can't verify message sender's inshard proof")
	}
	return nil
}
