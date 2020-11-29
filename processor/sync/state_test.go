/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"testing"

	"github.com/multivactech/MultiVAC/processor/shared/message"
)

func TestNewSyncState(t *testing.T) {
	state := newSyncState(message.NewPubSubManager())
	if state.status != IDLE {
		t.Errorf("SyncState initial status, shouldStartSync: %v, got: %v", IDLE, state.status)
	}
}

func TestSyncState_MaybeStart(t *testing.T) {
	s := createFakeSubscriber()
	state := newSyncState(message.NewPubSubManager())
	state.pubSubMgr.Sub(s)
	state.status = RUNNING
	if res := state.maybeStart(); res != false {
		t.Errorf("When state.status is IDLE, maybeStart should return: %v, got: %v", false, res)
	}
	state.status = IDLE
	state.maybeStart()
	if isStart := <-s.isStart; isStart != true {
		t.Errorf("maybeStart should change the onStart state: %v, but got: %v", true, isStart)
	}
}

func TestSyncState_MaybeStop(t *testing.T) {
	s := createFakeSubscriber()
	state := newSyncState(message.NewPubSubManager())
	state.pubSubMgr.Sub(s)
	state.status = IDLE
	if res := state.maybeStop(true); res != false {
		t.Errorf("When state.status is RUNNING, maybeStop should return: %v, got: %v",
			false, res)
	}
	state.status = RUNNING
	state.maybeStop(true)
	if isComplete := <-s.isComplete; isComplete != true {
		t.Errorf("maybeStop should change the onComplete state: %v, but got: %v", true, isComplete)
	}
}

// fakeListner is for state_test.go which should implement SyncListener interface
type fakeSubscriber struct {
	isStart    chan bool
	isComplete chan bool
	isAbort    chan bool
}

func createFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{
		isStart:    make(chan bool),
		isComplete: make(chan bool),
		isAbort:    make(chan bool),
	}
}

func (fs *fakeSubscriber) Recv(e message.Event) {
	switch e.Topic {
	case message.EvtSyncStart:
		fs.OnStart()
	case message.EvtSyncComplete:
		fs.OnComplete(e.Extra.(bool))
	case message.EvtSyncAbort:
		fs.OnAbort()
	}
}

func (fs *fakeSubscriber) OnStart() {
	fs.isStart <- true
}

func (fs *fakeSubscriber) OnComplete(dataFetched bool) {
	fs.isComplete <- dataFetched
}

func (fs *fakeSubscriber) OnAbort() {
	fs.isAbort <- true
}
