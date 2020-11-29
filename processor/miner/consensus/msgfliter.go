/**
* Copyright (c) 2018-present, MultiVAC Foundation.
*
* This source code is licensed under the MIT license found in the
* LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"github.com/multivactech/MultiVAC/model/wire"
)

type handleType int

const (
	ignore handleType = iota
	handleMsg
	handleOtherStateMsg
)

// msgFilter determine legality of the round, inShardProof, signature.
func (ce *executor) msgFilter(curRound int, message consensusMsg) handleType {
	msgType := ce.isStateValid(curRound, message)
	if msgType == ignore {
		return ignore
	}

	if !ce.isInShard(message) {
		ce.logger.Debugf("Receive %s ,inShardProof not valid ,round %d", message.Command(), message.GetRound())
		return ignore
	}

	if !ce.isValidated(message) {
		return ignore
	}
	return msgType
}

// isStateValid returns handleType according to the current ce status and message status.
// ignore : message round is outdated or ceStatus is shardDisabled
// handleOtherStateMsg : receive future message or ceStatus is shardPreparation
// handleMsg : normal message
func (ce *executor) isStateValid(curRound int, message consensusMsg) handleType {
	ceStatus := ce.state.getShardStatus()
	if message.GetRound() < curRound || ceStatus == shardDisabled {
		return ignore
	}
	if message.GetRound() > curRound || ceStatus == shardPreparation {
		return handleOtherStateMsg
	}

	return handleMsg
}

// isValidated determine whether the message the sender belongs to the heartbeat list
// and the signature is legal.
func (ce *executor) isValidated(message consensusMsg) bool {
	// todo(MH): 可能需要判断 confirmation 投票者在心跳名单中
	if message.Command() != wire.CmdMsgBlockConfirmation {
		if !ce.verifyMiner(message.GetSignedCredential().Pk) {
			return false
		}
	}

	if err := message.IsValidated(); err != nil {
		ce.logger.Debugf("Receive %s not valid, round %v, error : %v", message.Command(), message.GetRound(), err)
		return false
	}
	return true
}

func (ce *executor) isInShard(message consensusMsg) bool {
	inShardProofs := message.GetInShardProofs()
	isInShard := true
	for _, inShardProof := range inShardProofs {
		if !ce.sortitionor.IsInShard(inShardProof, ce.state.getDepositCache(), ce.shardIndex) {
			isInShard = false
			break
		}
	}
	return isInShard && len(inShardProofs) > 0
}

func (ce *executor) verifyMiner(pk []byte) bool {
	if ce.threadHeartBeatManager.Has(pk) {
		return true
	}
	ce.logger.Debugf("Failed to verify miner, %v", pk)
	return false
}
