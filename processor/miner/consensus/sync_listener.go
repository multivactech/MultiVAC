/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// abciTxPoolInitSubscriber is used to listen to abci's behavior of ledger update.
type abciTxPoolInitSubscriber struct {
	ce *executor
}

func (s *abciTxPoolInitSubscriber) Recv(e message.Event) {
	switch e.Topic {
	case message.EvtTxPoolInitFinished:
		s.onInitFinished(e.Extra.(int64))
	default:
		log.Debugf("%v received unknown mail: %v", s, e.Topic)
	}
}

func (s *abciTxPoolInitSubscriber) onInitFinished(shardHeight int64) {
	// Get the current state(round&step) of consensus
	ce := s.ce
	curRound := ce.state.getRound()
	nextRnd := int(shardHeight - 1)

	ce.logger.Debugf("%v receives AppBlockChain txpool init complete signal, current height in block chain: %v, round: %v",
		ce, shardHeight, curRound)

	// change the shardstate
	if ce.state.getShardStatus() == shardDisabled {
		// If it is the first miners and inits firstly, set the shard enabled
		if config.GlobalConfig().IsOneOfFirstMiners && shardHeight == 1 {
			ce.updateShardStatus(shardEnabled)
		} else {
			// If it inits complete, set the shard preparation
			if curRound > nextRnd {
				ce.logger.Debugf("Ce is in outdated round, ignore it")
				return
			}
			ce.logger.Debugf("reshard txpool init complete, set round: %v, set the reshard state to ReshardPreparation", nextRnd)
			ce.updateShardStatus(shardPreparation)
			ce.setCurrentRound(nextRnd)
		}
	}
}

func newInitSubscriber(ce *executor) *abciTxPoolInitSubscriber {
	return &abciTxPoolInitSubscriber{ce}
}
