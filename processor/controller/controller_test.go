package controller

import (
	"testing"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/icontroller"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/controller/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRootController_GetEnabledShards(t *testing.T) {
	tests := []struct {
		desc       string
		controller *RootController
		expected   []shard.Index
	}{
		{
			desc:       "GetEnabledShards for Storage Node",
			controller: createMockedRootController(true),
			expected:   shardIndexListForTest,
		},
		{
			desc:       "GetEnabledShards for Miner Node",
			controller: createMockedRootController(false),
			expected:   shardIndexListForTest,
		},
	}
	a := assert.New(t)
	for _, test := range tests {
		mockHandler := new(mocks.MockedReshardHandler)
		mockHandler.On("GetEnableShards").Return(test.expected)
		test.controller.reshardHandler = mockHandler
		a.Equalf(test.expected, test.controller.GetEnabledShards(), "Test %v failure", test.desc)
		mockHandler.AssertNumberOfCalls(t, "GetEnableShards", 1)
	}
}

func TestRootController_NotifyPeerAdd(t *testing.T) {
	ctrl := createMockedRootController(false)
	p := &connection.ConnPeer{} // By default it is a miner peer
	mockRouter := new(mocks.MockedShardRouter)
	ctrl.shardRouter = mockRouter
	mockShardCtrl := new(mocks.MockedShardController)
	shardControllers := []icontroller.IShardController{mockShardCtrl, mockShardCtrl, mockShardCtrl}
	mockShardCtrl.On("NotifyNewPeerAdded", p, config.ParseNodeType(false)).Times(len(shardControllers))
	mockRouter.On("GetAllShardControllers").Return(shardControllers)

	ctrl.NotifyPeerAdd(p)

	mockShardCtrl.AssertNumberOfCalls(t, "NotifyNewPeerAdded", len(shardControllers))
}

func TestRootController_NotifyPeerDone(t *testing.T) {
	ctrl := createMockedRootController(true)
	p := &connection.ConnPeer{} // By default it is a miner peer
	mockRouter := new(mocks.MockedShardRouter)
	ctrl.shardRouter = mockRouter
	mockShardCtrl := new(mocks.MockedShardController)
	shardControllers := []icontroller.IShardController{mockShardCtrl, mockShardCtrl, mockShardCtrl}
	mockShardCtrl.On("NotifyPeerDone", p, config.ParseNodeType(false)).Times(len(shardControllers))
	mockRouter.On("GetAllShardControllers").Return(shardControllers)

	ctrl.NotifyPeerDone(p)

	mockShardCtrl.AssertNumberOfCalls(t, "NotifyPeerDone", len(shardControllers))
}

func TestRootController_Start(t *testing.T) {
	tests := []struct {
		desc          string
		isStorageNode bool
	}{
		{
			desc:          "RootController start as storage node",
			isStorageNode: true,
		},
		{
			desc:          "RootController start as miner node",
			isStorageNode: false,
		},
	}

	for _, test := range tests {
		ctrl := createMockedRootController(test.isStorageNode)

		// reshard handler
		mockReshardHandler := new(mocks.MockedReshardHandler)
		mockReshardHandler.On("StartEnabledShardControllers").Times(1)
		ctrl.reshardHandler = mockReshardHandler
		// heartbeat enabler
		mockHeartbeatEnabler := new(mocks.MockedHeartbeatEnabler)
		mockHeartbeatEnabler.On("Start").Times(1)
		ctrl.heartBeatManager = mockHeartbeatEnabler

		ctrl.Start()

		mockReshardHandler.AssertNumberOfCalls(t, "StartEnabledShardControllers", 1)
		if test.isStorageNode {
			mockHeartbeatEnabler.AssertNumberOfCalls(t, "Start", 0)
		} else {
			mockHeartbeatEnabler.AssertNumberOfCalls(t, "Start", 1)
		}
	}
}
