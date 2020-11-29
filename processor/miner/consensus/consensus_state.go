/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"fmt"
	"sync"

	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

// Initialization constant for executor
const (
	// initialRound is uesd for Consensus_Executor init.
	initialRound = -1
	initialStep  = 0
	startRound   = 0
	// startingStepForRound is the first step in every round of consensus.
	startingStepForRound = 1
	// init the shardStopRound so that consensus will not stop when beginning
	initShardStopRound  = -2
	initShardStartRound = 0
)

var (
	emptySeedBuf = [32]byte{}
	emptySeed, _ = chainhash.NewHash(emptySeedBuf[0:]) //empty at first
)

// CEStatus is a type (int8) for mark status.
type ceStatus int8

// Sync status constants
// Allowed state machine changes:
//   unSynced -> WaitForBlockConfirm
//   WaitForBlockConfirm -> unSynced
//   WaitForBlockConfirm -> upToDate
//   upToDate -> unSynced
const (
	// unSynced	means the consensus need stop for sync.
	unSynced ceStatus = iota
	// upToDate means the consensus is the lastest data.
	upToDate
	// WaitForBlockConfirm means the consensus have recv initData from storage node,
	// and need to recv blockconfirmation in cur shard.
	// If recv bc, the consensus will begin again.
	waitForBlockConfirm
)

// Shard status for Reshard.
// Allowed state machine changes:
//   ShardDisabled -> ShardPreparation : ledger init done
//   ShardDisabled -> ShardEnabled : first miners init done
//   ShardPreparation -> ShardEnabled : really resharding
//   ShardEnabled -> ShardDisabled : need to reshard
const (
	// at height H, judging whether to consensus in the shard next time
	// at height H+K, consensus or not in the shard
	// if there is no consensus in the shard before height H+K, consensus after H+K,
	// it need to get the latest ledger between height K. (shardState: ShardPerparation)

	// ShardDisabled means do not start new round
	shardDisabled ceStatus = iota
	// ShardPreparation means cache the proposal block and receive the block after confirmation
	shardPreparation
	// ShardEnabled means normal consensus
	shardEnabled
)

var (
	maxCred, _ = chainhash.NewHash([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
)

// consensusState struct for document the status of the current consensus,
// it contains round, step, status of sync, status of reshard and a general state of consensus.
type consensusState struct {
	mu    *sync.RWMutex
	round int
	step  int

	// running means whether the consensus is running, or stop.
	running bool

	// The key of smallestCred is round number, the value is the smallest (hashed?) credential
	// of block proposal messages that has seen so far for this round.
	// This is an optimization: if we get a larger credential later, don't need to propagate.
	// change type to sync.Map, for map concurrent
	smallestCred chainhash.Hash

	prevRoundSeed chainhash.Hash
	preRoundS     [10]*pre
	//Need to update when resharding
	inShardProof   []byte
	myDepositCache uint32

	// is starting new round, if it is true, any other msg should not be dispatched.
	isStartNewRound        bool
	waitGroupProcessingMsg *sync.WaitGroup
	// shardStopRound: When reaching the stopped round, it need to change
	// the state immediately to stop entering the next round.
	shardStartRound int
	shardStopRound  int

	// they shouldn't export to other package.
	syncStatus  ceStatus
	shardStatus ceStatus

	shardStatusListeners []iconsensus.CEShardStateListener
}

type pre struct {
	round int
	seed  *chainhash.Hash
}

// newConsensusState return a point to consensusState struct.
func newConsensusState() *consensusState {
	return &consensusState{
		mu:                     new(sync.RWMutex),
		round:                  initialRound,
		step:                   initialStep,
		running:                true,
		smallestCred:           *maxCred,
		prevRoundSeed:          *emptySeed,
		inShardProof:           make([]byte, 32),
		isStartNewRound:        false,
		waitGroupProcessingMsg: new(sync.WaitGroup),
		shardStopRound:         initShardStopRound,
		shardStartRound:        initShardStartRound,
		syncStatus:             unSynced,
		shardStatus:            shardDisabled,
		shardStatusListeners:   make([]iconsensus.CEShardStateListener, 0),
	}
}

func (cs *consensusState) getStep() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.step
}

func (cs *consensusState) getRound() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.round
}

func (cs *consensusState) setStepInRound(round, newStep int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if newStep > cs.step && cs.round == round {
		cs.step = newStep
	}
}

func (cs *consensusState) setRound(round int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.round = round
	cs.step = startingStepForRound
	cs.smallestCred = *maxCred
}

func (cs *consensusState) setNextStepInRound(round, step int) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.step == step-1 && cs.round == round {
		cs.step++
		return true
	}
	return false
}

func (cs *consensusState) updateShardStatus(shardStatus ceStatus) error {
	logger.ServerLogger().Debugf("Update cons state to %v ", shardStatus)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	//   ShardDisabled -> ShardPreparation : ledger init done
	//   ShardDisabled -> ShardEnabled : first miners init done
	//   ShardPreparation -> ShardEnabled : really resharding
	//   ShardEnabled -> ShardDisabled : need to reshard
	switch shardStatus {
	case shardEnabled:
		cs.shardStatus = shardStatus
		return nil
	case shardDisabled:
		if cs.shardStatus == shardEnabled {
			cs.shardStatus = shardStatus
		} else {
			return fmt.Errorf("Up to ShardDisabled, cur shard state is %v, cur round is %v", cs.getShardStatusString(cs.shardStatus), cs.round)
		}
	case shardPreparation:
		if cs.shardStatus == shardDisabled {
			cs.shardStatus = shardStatus
		} else {
			return fmt.Errorf("Up to ShardPreparation, cur shard state is %v, cur round is %v", cs.getShardStatusString(cs.shardStatus), cs.round)
		}
	}
	return nil
}

func (cs *consensusState) getSyncStatusString(syncStatus ceStatus) string {
	switch syncStatus {
	case unSynced:
		return "unSynced"
	case upToDate:
		return "upToDate"
	case waitForBlockConfirm:
		return "WaitForBlockConfirm"
	default:
		return ""
	}
}

func (cs *consensusState) getShardStatusString(shardStatus ceStatus) string {
	switch shardStatus {
	case shardEnabled:
		return "ShardEnabled"
	case shardDisabled:
		return "ShardDisabled"
	case shardPreparation:
		return "ShardPreparation"
	default:
		return ""
	}
}

func (cs *consensusState) updateShardStartRound(round int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.shardStartRound = round
}

func (cs *consensusState) updateShardStopRound(round int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.shardStopRound = round
}

func (cs *consensusState) getShardStatus() ceStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.shardStatus
}

func (cs *consensusState) getSyncStatus() ceStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.syncStatus
}

func (cs *consensusState) getShardStartRound() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.shardStartRound
}

func (cs *consensusState) getShardStopRound() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.shardStopRound
}

func (cs *consensusState) isShardEnable() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.running
}

func (cs *consensusState) registerCEShardStateListener(listener iconsensus.CEShardStateListener) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.shardStatusListeners = append(cs.shardStatusListeners, listener)
}

func (cs *consensusState) setShardAndListenerDisabled() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.shardStatus = shardDisabled
	logger.ServerLogger().Debugf("CONS start to notify listener")
	for _, listener := range cs.shardStatusListeners {
		logger.ServerLogger().Debugf("Cons notify shardManager to disable shard")
		go listener.OnShardDisabled()
	}
}

// 	smallestCred   chainhash.Hash
// prevRoundSeed  chainhash.Hash
// inShardProof   []byte
// myDepositCache uint32

func (cs *consensusState) setSmallestCredInRound(round int, cred chainhash.Hash) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if round == cs.round {
		cs.smallestCred = cred
	}
}

func (cs *consensusState) getSmallestCred() chainhash.Hash {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.smallestCred
}

func (cs *consensusState) setRoundSeed(round int, seed chainhash.Hash) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.preRoundS[round%10] == nil || cs.preRoundS[round%10].round < round {
		cs.preRoundS[round%10] = &pre{
			round: round,
			seed:  &seed,
		}
	}
}

// TODO: testnet-3.0
func (cs *consensusState) getRoundSeed(round int) *chainhash.Hash {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if round <= 0 {
		return emptySeed
	}
	cd := cs.preRoundS[round%10]
	if cd != nil && cd.round == round {
		return cd.seed
	}
	return emptySeed
}

func (cs *consensusState) setInShardProof(proof []byte) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.inShardProof = proof
}

func (cs *consensusState) getInShardProof() []byte {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.inShardProof
}

func (cs *consensusState) setDepositCache(cache uint32) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.myDepositCache = cache
}

func (cs *consensusState) getDepositCache() uint32 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.myDepositCache
}
