/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

type triggerRequest struct {
	shard   shard.Index
	trigger SyncTrigger
}

type saveSlimBlockRequest struct {
	toshard shard.Index
	shard   shard.Index
	height  wire.BlockHeight
}
