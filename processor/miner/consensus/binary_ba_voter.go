/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"sync"

	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/wire"
)

type voteKey struct {
	round, step int
	b           byte
}

// voteValue for same binary value b, there may be different ByzAgreementValue vote
type voteValue struct {
	cnt    int                            // total count for a given b
	cntMap map[wire.ByzAgreementValue]int // vote count for each particular byz value v
}

type (
	resultType int
	pkKey      [32]byte
)

const (
	// totalAgreeOnZero consensus all agree on 0.
	totalAgreeOnZero resultType = iota
	// partialAgreeOnZero consensus partial agree on 0, others are unknown.
	partialAgreeOnZero
	// agreeOnOne number of votes meeting the threshold reached consensus on 1.
	agreeOnOne
	// notAgree failure to reach consensus.
	notAgree
)

type binaryBAVoter struct {
	mu          *sync.RWMutex
	result      map[voteKey]*voteValue
	resultValue *wire.ByzAgreementValue
	threshold   int
	round       int
	logger      btclog.Logger
	bbaHistory  map[voteKey]map[pkKey]*wire.MsgBinaryBA
	gcHistory   map[pkKey]bool
	resRound    int
	// set threshold to make sure there is overwhelmingly probability such that honest voters are more than 2/3
}

func pkToPKKey(pk []byte) pkKey {
	var key [32]byte
	copy(key[:], pk[:32])
	return pkKey(key)
}

func newBinaryBAVoter(round, threshold int, logger btclog.Logger) *binaryBAVoter {

	return &binaryBAVoter{
		mu:          new(sync.RWMutex),
		threshold:   threshold,
		round:       round,
		logger:      logger,
		result:      make(map[voteKey]*voteValue),
		resultValue: &wire.ByzAgreementValue{},
		bbaHistory:  make(map[voteKey]map[pkKey]*wire.MsgBinaryBA),
		gcHistory:   make(map[pkKey]bool),
	}
}

func (bv *binaryBAVoter) setThreshold(threshold int) {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	bv.threshold = threshold
}

func (bv *binaryBAVoter) setCurrentRound(round int) {
	bv.mu.Lock()
	defer bv.mu.Unlock()
	bv.round = round

	// clear used vote
	for key := range bv.bbaHistory {
		if key.round < round-1 {
			delete(bv.bbaHistory, key)
		}
	}
	for key := range bv.result {
		if key.round < round-1 {
			delete(bv.result, key)
		}
	}
	bv.gcHistory = make(map[pkKey]bool)
}

/*
	if step < 0 , it means node reach BBAFin at -step
	TODO(guotao):save with sig ,do not trust vote info and BBAFin info

*/

func (bv *binaryBAVoter) numOfFinNodesBeforeStep(round, step int, b byte, vote *wire.ByzAgreementValue) int {
	sum := 0
	for i := 4; i < step; i++ {
		k := voteKey{round, -i, b}
		if exist := bv.result[k]; exist != nil {
			if vote == nil {
				sum += bv.result[k].cnt
			} else {
				sum += bv.result[k].cntMap[*vote]
			}
		}
	}
	if bv.logger != nil && sum > 0 {
		bv.logger.Debugf("numOfFinNodesBeforeStep round %d, step %d,b %d ,sum %d ", round, step, b, sum)
	}
	return sum
}

func (bv *binaryBAVoter) getBBAHistory(round, step int, b byte) []*wire.MsgBinaryBA {
	bv.mu.RLock()
	defer bv.mu.RUnlock()

	k := voteKey{round, step, b}
	//merge with FIN bba
	res := []*wire.MsgBinaryBA{}
	for _, v := range bv.bbaHistory[k] {
		res = append(res, v)
	}

	if bv.logger != nil && len(res) == 0 {
		bv.logger.Warnf("getBBAHistory round %d, step %d,b %d ,length is 0,history is %v ", round, step, b, bv.bbaHistory[k])
	}
	return res
}

func (bv *binaryBAVoter) addVoteWithBBAMsg(round, step int, b byte, value *wire.ByzAgreementValue,
	msgbba *wire.MsgBinaryBA) (*voteKey, resultType, bool) {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	k := voteKey{round, step, b}
	if historyListByKey, find := bv.bbaHistory[k]; find {
		if _, find2 := historyListByKey[pkToPKKey(msgbba.SignedCredentialWithBA.Pk)]; find2 {
			log.Debugf("there is same PK of the BBAMsg, so ignore. %v", msgbba.SignedCredentialWithBA.Pk)
			return nil, notAgree, false // receive same bba info
		}
		historyListByKey[pkToPKKey(msgbba.SignedCredentialWithBA.Pk)] = msgbba
	} else {
		bv.bbaHistory[k] = map[pkKey]*wire.MsgBinaryBA{
			pkToPKKey(msgbba.SignedCredentialWithBA.Pk): msgbba,
		}
	}
	return bv.addVote(round, step, b, value)
}

func (bv *binaryBAVoter) addGCVote(round, step int, b byte, value *wire.ByzAgreementValue, pk []byte) (*voteKey, resultType, bool) {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if _, exist := bv.gcHistory[pkToPKKey(pk)]; exist {
		log.Debugf("there is same PK of the GCMsg, so ignore. %v", pk)
		return nil, notAgree, false
	}
	bv.gcHistory[pkToPKKey(pk)] = true

	return bv.addVote(round, step, b, value)
}

// NOTE: will return a bool value named haveBroadcast, which means the number whether more than threshold.
// if it more than threshold, consensus will not handle and broadcast the reach-consensus-msg,
// because consensus have solve it when the num equal with threshold. avoiding more consumption of network.
func (bv *binaryBAVoter) addVote(round, step int, b byte, value *wire.ByzAgreementValue) (*voteKey, resultType, bool) {
	k := voteKey{round, step, b}
	if exist := bv.result[k]; exist != nil {
		bv.result[k].cnt++
		bv.result[k].cntMap[*value]++
	} else {
		bv.result[k] = &voteValue{
			cnt: 1,
			cntMap: map[wire.ByzAgreementValue]int{
				*value: 1,
			},
		}
	}

	if bv.logger != nil {
		bv.logger.Debugf("BBAVoter: cnt for result[%v]: %v, threshold: %v", k, bv.result[k].cnt, bv.threshold)
	}

	// If receive more than threshold votes that vote for the same block
	if b == 0 {
		num := bv.result[k].cntMap[*value] + bv.numOfFinNodesBeforeStep(round, step, b, value)
		if num == bv.threshold {
			return &k, totalAgreeOnZero, false
		} else if num > bv.threshold {
			return &k, totalAgreeOnZero, true
		}
	}

	// If receive more than threshold votes that form of <round,step,b> (may vote for different vote)
	num := bv.result[k].cnt + bv.numOfFinNodesBeforeStep(round, step, b, nil)
	if num == bv.threshold {
		if b == 0 {
			return &k, partialAgreeOnZero, false
		}
		return &k, agreeOnOne, false
	} else if num > bv.threshold {
		if b == 0 {
			return &k, partialAgreeOnZero, false
		}
		return &k, agreeOnOne, true
	}

	return nil, notAgree, false
}

// Get the byzValue that has halfThreshold votes
func (bv *binaryBAVoter) getVoteKeyWithHalfThreshold() (*voteKey, *wire.ByzAgreementValue) {
	bv.mu.RLock()
	defer bv.mu.RUnlock()

	step := 4
	k := voteKey{bv.round, step, 0}
	if exist := bv.result[k]; exist != nil {
		for byzValue, count := range bv.result[k].cntMap {
			if count >= (bv.threshold+1)/2 {
				return &k, &byzValue
			}
		}
	}
	return nil, nil
}

func (bv *binaryBAVoter) getResultValue(round int) *wire.ByzAgreementValue {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	if round != bv.resRound {
		return nil
	}
	return bv.resultValue
}

func (bv *binaryBAVoter) setResultValue(round int, value *wire.ByzAgreementValue) {
	bv.mu.Lock()
	defer bv.mu.Unlock()
	if round >= bv.resRound {
		bv.resultValue = value
		bv.resRound = round
	}
}

func (bv *binaryBAVoter) getAgreementString(result resultType) string {
	switch result {
	case totalAgreeOnZero:
		return "totalAgreeOnZero"
	case partialAgreeOnZero:
		return "partialAgreeOnZero"
	case agreeOnOne:
		return "agreeOnOne"
	case notAgree:
		return "notAgree"
	}
	return ""
}
