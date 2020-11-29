/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"time"
)

const (
	// defaultGCTimeoutInSec means the time interval of GC timeout. the unit is second.
	defaultGCTimeoutInSec = 10
	maxStep               = 3
)

// TODO(libo): consider move it /base folder.
// mockable ticker for easier testing.
type ticker interface {
	Stop()
	Chan() <-chan time.Time
}

type realTicker struct{ *time.Ticker }

func (rt *realTicker) Chan() <-chan time.Time {
	return rt.C
}

type consensusTimer struct {
	ticker       ticker
	tickerGen    func(int) ticker
	currentRound int
	currentStep  int
	stop         chan bool
}

func newConsensusTimer() *consensusTimer {
	return &consensusTimer{
		tickerGen: func(timeOutInSec int) ticker {
			return &realTicker{
				time.NewTicker(time.Duration(timeOutInSec) * time.Second),
			}
		},
	}
}

func (ct *consensusTimer) startNewRound(round int, onTimeOut timerCallback, timeOutInSec int) {
	if ct == nil {
		ct = newConsensusTimer()
	}
	ct.stopTicker()
	ct.ticker = ct.tickerGen(timeOutInSec)
	if ct.stop == nil {
		ct.stop = make(chan bool, 1)
	}
	ct.currentRound = round
	ct.currentStep = 1
	go ct.timeoutRoutine(onTimeOut)
}

func (ct *consensusTimer) stopTicker() {
	if ct.ticker != nil {
		ct.ticker.Stop()
	}
	if ct.stop != nil {
		ct.stop <- true
	}
}

func (ct *consensusTimer) timeoutRoutine(onTimeOut func(round int, step int)) {
	for {
		select {
		case t := <-ct.ticker.Chan():
			log.Debug("time out", t)
			onTimeOut(ct.currentRound, ct.currentStep)
			ct.currentStep++
			if ct.currentStep >= maxStep {
				ct.stopTicker()
			}
		case <-ct.stop:
			return
		}
	}
}

func (ct *consensusTimer) onTimeOut(timeOutFn func(), timeOutInSec int) {
	if ct == nil {
		ct = newConsensusTimer()
	}
	ticker := ct.tickerGen(timeOutInSec)
	ct.ticker = ticker
	<-ticker.Chan()
	ticker.Stop()
	timeOutFn()
}

type timerCallback func(round int, step int)
