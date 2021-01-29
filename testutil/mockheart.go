/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package testutil

import "github.com/multivactech/MultiVAC/model/wire"

// MockHeart is heartbeat just used for testing
type MockHeart struct {
	listsize int
}

// NewMockHeart returns a new instance of MockHeart
func NewMockHeart(n int) *MockHeart {
	return &MockHeart{
		listsize: n,
	}
}

// Receive used to receive heartbeat message from other nodes
func (mh *MockHeart) Receive(msg *wire.HeartBeatMsg) {}

// Has verfity if the pk is in whitelist
func (mh *MockHeart) Has(pk []byte) bool {
	return true
}

// PerceivedCount returns the number of full network miners currently listening
func (mh *MockHeart) PerceivedCount() int {
	return mh.listsize
}

// Start this module
func (mh *MockHeart) Start() {}

// Stop this module
func (mh *MockHeart) Stop() {}

// SetNumberOfList just used for testing
func (mh *MockHeart) SetNumberOfList(n int) {
	mh.listsize = n
}

// CheckConfirmation implements HeatBeat interface.
func (mh *MockHeart) CheckConfirmation(msg *wire.MsgBlockConfirmation) bool {
	return true
}
