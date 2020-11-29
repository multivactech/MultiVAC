package mocks

import (
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

// MockBroadCaster is a mock broadcaster.
type MockBroadCaster struct {
	Count int
}

// NewMockBroadCaster mocks a broadcaster.
func NewMockBroadCaster() *MockBroadCaster {
	return &MockBroadCaster{Count: 0}
}

// BroadcastMessage mock a broadcast action.
func (mbc *MockBroadCaster) BroadcastMessage(msg wire.Message, params *connection.BroadcastParams) {
	mbc.Count++
}

// RegisterChannels is a mock method.
func (mbc *MockBroadCaster) RegisterChannels(dispatch *connection.MessagesAndReceiver) {

}

// HandleMessage is a mock method.
func (mbc *MockBroadCaster) HandleMessage(message wire.Message) {
}
