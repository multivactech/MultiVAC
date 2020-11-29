/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package iconsensus

// Consensus is the consensus engine
type Consensus interface {
	// StartConsensusAtRound means set the consensus start at the round;
	StartConsensusAtRound(round int)
	// StopConsensusAtRound means when shard needs to be disabled, stop the consensus firstly
	StopConsensusAtRound(round int)
	// todo(MH): maybe remove
	// UpdateInShardProof will update consensusExecutor.inshardProof, it will be safe in concurrent.
	UpdateInShardProof(inShardProof []byte)
	// RegisterCEListener adds a listener to shardStateListeners
	RegisterCEShardStateListener(listener CEShardStateListener)

	Init()
	Start()
}
