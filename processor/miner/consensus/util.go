/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/multivactech/MultiVAC/metrics"
	"sync"

	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// used for testing only

// SimpleBlockInfo is used for test, (tools, testctl...), it will record cur round block and write them to local file.
type SimpleBlockInfo struct {
	round    int32
	leader   string
	prevHash chainhash.Hash
	curHash  chainhash.Hash
}

func (sb *SimpleBlockInfo) String() string {
	return fmt.Sprintf("{round %v, leader %v, prevHash %v, curHash %v",
		sb.round, sb.leader, sb.prevHash, sb.curHash)
}

// Params contains some infos of the miner, such as pk, sk, voteThreshold...
type Params struct {
	pk, sk           []byte
	voteThreshold    int
	shardIndex       shard.Index
	integrationTest  bool
	heartbeatManager iheartbeat.HeartBeat
	pubSubMgr        *message.PubSubManager
	metrics          *metrics.Metrics
}

// CreateParams will return a pointer to Params, which initialization data is provided by the incoming parameters.
func CreateParams(pk, sk []byte, voteThreshold int, integrationTest bool, shardIndex shard.Index,
	manager iheartbeat.HeartBeat, pubSubMgr *message.PubSubManager, metrics *metrics.Metrics) *Params {
	return &Params{
		pk: pk, sk: sk,
		voteThreshold:    voteThreshold,
		integrationTest:  integrationTest,
		shardIndex:       shardIndex,
		heartbeatManager: manager,
		pubSubMgr:        pubSubMgr,
		metrics:          metrics,
	}
}

// calcRoundSeed calculates the seeds of this round based on the previous round of seeds.
// Algorithm: currentSeed = sha256(lastSignSeed + round)
func calcRoundSeed(lastSeedSig []byte, round int) chainhash.Hash {
	int64Size := 8
	seedBuf := make([]byte, len(lastSeedSig)+int64Size)
	copy(seedBuf, lastSeedSig[0:])
	binary.PutVarint(seedBuf[len(lastSeedSig):], int64(round))
	curSeed := sha256.Sum256(seedBuf)
	return curSeed
}

// FakeDepositFetcher just for test.
type FakeDepositFetcher struct {
}

// GetDepositUnit return 1. for test.
func (df FakeDepositFetcher) GetDepositUnit(pk []byte, round int32, index shard.Index) uint32 {
	return 1
}

type receivedData struct {
	mu *sync.RWMutex

	receivedSeed  map[int]map[string]chainhash.Hash
	receivedBlock map[int]map[chainhash.Hash]*wire.MsgBlock
	// shardStopRound: When reaching the stopped round, it need to change the state
	// immediately to stop entering the next round.
	cacheBlockForPrep map[chainhash.Hash]*wire.MsgBlock
}

func newReceivedData() *receivedData {
	return &receivedData{
		mu:                new(sync.RWMutex),
		receivedSeed:      make(map[int]map[string]chainhash.Hash),
		receivedBlock:     make(map[int]map[chainhash.Hash]*wire.MsgBlock),
		cacheBlockForPrep: make(map[chainhash.Hash]*wire.MsgBlock),
	}
}

func (rd *receivedData) cleanUnusedData(round int) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	for r := range rd.receivedBlock {
		if r < round {
			delete(rd.receivedBlock, r)
		}
	}
	for r := range rd.receivedSeed {
		if r < round {
			delete(rd.receivedSeed, r)
		}
	}
}

func (rd *receivedData) cleanCacheBlock() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.cacheBlockForPrep = make(map[chainhash.Hash]*wire.MsgBlock)
}

func (rd *receivedData) saveSeed(round int, sender string, seed chainhash.Hash) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if _, exist := rd.receivedSeed[round]; !exist {
		rd.receivedSeed[round] = make(map[string]chainhash.Hash)
	}
	rd.receivedSeed[round][sender] = seed
}

func (rd *receivedData) getSeed(round int, sender string) (chainhash.Hash, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	if _, exist := rd.receivedSeed[round]; exist {
		if seed, ok := rd.receivedSeed[round][sender]; ok {
			return seed, ok
		}
	}
	return chainhash.Hash{}, false
}

func (rd *receivedData) saveBlock(round int, blockHash chainhash.Hash, block *wire.MsgBlock) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if _, exist := rd.receivedBlock[round]; !exist {
		rd.receivedBlock[round] = make(map[chainhash.Hash]*wire.MsgBlock)
	}
	rd.receivedBlock[round][blockHash] = block
}

func (rd *receivedData) getBlock(round int, blockHash chainhash.Hash) (*wire.MsgBlock, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	if _, exist := rd.receivedBlock[round]; exist {
		if block, ok := rd.receivedBlock[round][blockHash]; ok {
			return block, ok
		}
	}
	return nil, false
}

func (rd *receivedData) cacheBlock(blockHash chainhash.Hash, block *wire.MsgBlock) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.cacheBlockForPrep[blockHash] = block
}

func (rd *receivedData) reloadBlock(blockHash chainhash.Hash) (*wire.MsgBlock, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	block, ok := rd.cacheBlockForPrep[blockHash]
	return block, ok
}

type pendingData struct {
	mu *sync.RWMutex
	// pendingMsg stores the messages which are coming from "future" rounds,
	// it will be replayed when current round finishes. The key is round # and step #.
	// And pendingMsgMaxStep (whose key is round #) indicates the max step # in pendingMsg.
	pendingMsg        map[int]map[int][]wire.Message
	pendingMsgMaxStep map[int]int
	// pendingMsgBySender indicates if we got messages from a particular peer in a particular round.
	// Keyed by round # and peer id (pk). If we have received enough peers of a far-enough future round,
	// we think we are falling behind too much, and a catch-up procedure will be triggered.
	pendingMsgBySender map[int]map[string]bool
}

func newPendingData() *pendingData {
	return &pendingData{
		mu:                 new(sync.RWMutex),
		pendingMsg:         make(map[int]map[int][]wire.Message),
		pendingMsgMaxStep:  make(map[int]int),
		pendingMsgBySender: make(map[int]map[string]bool),
	}
}

func (pd *pendingData) saveMsg(round, step int, sender string, msg wire.Message) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if _, exist := pd.pendingMsg[round]; !exist {
		pd.pendingMsg[round] = make(map[int][]wire.Message)
		pd.pendingMsgBySender[round] = make(map[string]bool)
	}
	pd.pendingMsg[round][step] = append(pd.pendingMsg[round][step], msg)
	if step > pd.pendingMsgMaxStep[round] {
		pd.pendingMsgMaxStep[round] = step
	}
	pd.pendingMsgBySender[round][sender] = true
}

func (pd *pendingData) getPendingMsg(round int) (map[int][]wire.Message, int, bool) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	var (
		pendingMsg map[int][]wire.Message
		maxStep    int
		exist      bool
	)
	if pendingMsg, exist = pd.pendingMsg[round]; !exist {
		return nil, 0, false
	}
	if maxStep, exist = pd.pendingMsgMaxStep[round]; !exist {
		return nil, 0, false
	}
	return pendingMsg, maxStep, true
}

func (pd *pendingData) deletePendingMsg(round int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	for r := range pd.pendingMsg {
		if r < round {
			delete(pd.pendingMsg, r)
		}
	}
	for r := range pd.pendingMsgMaxStep {
		if r < round {
			delete(pd.pendingMsgMaxStep, r)
		}
	}
	for r := range pd.pendingMsgBySender {
		if r < round {
			delete(pd.pendingMsgBySender, r)
		}
	}
}
