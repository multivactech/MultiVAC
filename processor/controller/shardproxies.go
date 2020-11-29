/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// This file defines the relevant interfaces and logic that proxies routing and dispatching work to shards.

package controller

import (
	"github.com/multivactech/MultiVAC/interface/icontroller"
	"github.com/multivactech/MultiVAC/model/shard"
)

// iReshardHandler defines the local interface for handling re-sharding relevant logic.
type iReshardHandler interface {
	// StartEnabledShardControllers triggers the start of all enabled shards controllers.
	StartEnabledShardControllers()
	// GetEnableShards returns the enabled shard index array.
	GetEnableShards() []shard.Index
}

// iShardRouter defines the local instance for routing the shards.
type iShardRouter interface {
	// GetShardController returns the shard controller for given shard
	GetShardController(shardIndex shard.Index) icontroller.IShardController
	// GetAllShardControllers returns all shard controllers
	GetAllShardControllers() []icontroller.IShardController
}
