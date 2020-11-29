/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package controller

import (
	"fmt"
	"time"

	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/prometheus/common/log"
)

type heartbeatEnabler interface {
	Start()
}

// RootController is at the upper level of the business logic and
// is used to schedule work for all shards.
type RootController struct {
	shardRouter      iShardRouter
	reshardHandler   iReshardHandler
	cfg              *config.Config
	heartBeatManager heartbeatEnabler
	depositPool      idepositpool.DepositPool
}

// GetEnabledShards returns all enabled shard indexes.
func (ctrl *RootController) GetEnabledShards() []shard.Index {
	return ctrl.reshardHandler.GetEnableShards()
}

// NotifyPeerDone notifies all shards a particular peer is done (disconnected or not needed)
func (ctrl *RootController) NotifyPeerDone(cp *connection.ConnPeer) {
	for _, sp := range ctrl.shardRouter.GetAllShardControllers() {
		sp.NotifyPeerDone(cp, config.ParseNodeType(cp.IsStorageNode()))
	}
}

// NotifyPeerAdd notifies all shards a particular peer is connected
func (ctrl *RootController) NotifyPeerAdd(cp *connection.ConnPeer) {
	for _, sp := range ctrl.shardRouter.GetAllShardControllers() {
		sp.NotifyNewPeerAdded(cp, config.ParseNodeType(cp.IsStorageNode()))
	}
}

// Start is used to start controller
func (ctrl *RootController) Start() {
	content := fmt.Sprintf("%v Start joining the network", time.Now())
	_, err := util.RedirectLog(content, "~/summary.txt")
	if err != nil {
		log.Errorf("failed to RedirectLog: %v", err)
	}
	if !ctrl.cfg.StorageNode {
		ctrl.heartBeatManager.Start()
	}
	ctrl.reshardHandler.StartEnabledShardControllers()
}
