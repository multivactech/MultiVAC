/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

type syncStatus int8

const (
	// IDLE is a kind of status of syncState, which means waiting to sync.
	IDLE syncStatus = iota

	// RUNNING is a kind of status of syncState, which means is doing sync.
	RUNNING
)

// const (
// 	maxListeners = 100
// )

type syncState struct {
	status    syncStatus
	pubSubMgr *message.PubSubManager
}

func newSyncState(pubSubMgr *message.PubSubManager) *syncState {
	state := syncState{
		status:    IDLE,
		pubSubMgr: pubSubMgr,
	}
	return &state
}

func (state *syncState) maybeStart() bool {
	if state.status != IDLE {
		log.Errorf("Attempting to start sync while it is not in IDLE mode.")
		return false
	}
	state.status = RUNNING
	state.pubSubMgr.Pub(*message.NewEvent(message.EvtSyncStart, nil))
	return true
}

func (state *syncState) maybeStop(dataFetched bool) bool {
	if state.status != RUNNING {
		log.Errorf("Attempting to stop sync while it is not in RUNNING mode.")
		return false
	}
	state.status = IDLE
	state.pubSubMgr.Pub(*message.NewEvent(message.EvtSyncComplete, dataFetched))
	return true
}

func (state *syncState) maybeAbort() bool {
	if state.status != RUNNING {
		log.Errorf("Attempting to abort sync while it is not in RUNNING mode.")
		return false
	}
	state.status = IDLE
	state.pubSubMgr.Pub(*message.NewEvent(message.EvtSyncAbort, nil))
	return true
}
