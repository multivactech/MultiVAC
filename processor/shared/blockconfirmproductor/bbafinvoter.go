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
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/wire"
)

type voter struct {
	logger     btclog.Logger
	threshold  int
	round      int
	bbaHistory map[voteKey]map[pkKey]*wire.MsgBinaryBAFin
}

type voteKey struct {
	round, step int
	blockHash   chainhash.Hash
}

type pkKey [32]byte

func (v *voter) setCurrentRound(round int, threshold int) {
	v.round = round
	v.threshold = threshold
	// clean outdated vote
	for key := range v.bbaHistory {
		if key.round < round {
			delete(v.bbaHistory, key)
		}
	}
}

func (v *voter) addVoteWithBBAFinMsg(round, step int, b byte, value *wire.ByzAgreementValue,
	bbaFin *wire.MsgBinaryBAFin) bool {
	k := voteKey{round, step, value.BlockHash}
	pk := pkToPKKey(bbaFin.SignedCredentialWithBA.Pk)
	if historyListByKey, find := v.bbaHistory[k]; find {
		if _, find2 := historyListByKey[pk]; find2 {
			v.logger.Debugf("there is same PK of the BBAFinMsg, so ignore. %v", pk)
			return false // receive same bba info
		}
		historyListByKey[pk] = bbaFin
	} else {
		v.bbaHistory[k] = map[pkKey]*wire.MsgBinaryBAFin{
			pk: bbaFin,
		}
	}
	v.logger.Debugf("receive bba result: %v, threshold: %v", len(v.bbaHistory[k]), v.threshold)
	return len(v.bbaHistory[k]) >= v.threshold
}

func pkToPKKey(pk []byte) pkKey {
	var key [32]byte
	copy(key[:], pk[:32])
	return pkKey(key)
}

func (v *voter) getBBAFinHistory(round, step int, b chainhash.Hash) []*wire.MsgBinaryBAFin {
	k := voteKey{round, step, b}
	//merge with FIN bba
	var res []*wire.MsgBinaryBAFin
	for _, v := range v.bbaHistory[k] {
		res = append(res, v)
	}
	return res
}
