/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package internal

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// TODO(huangsz): the concurrency concern here is only needed for handling MsgBlockConfirmation, actor doesn't seem
// needed here. Actually, looks a bit strange.
const (
	evtAdd message.EventTopic = iota
	evtHandleMessage
	evtStartProcessor
	evtInit
	evtGetEnableShards
	evtGetProcessor
	evtGetAllProcessor
)

type enableShardsResp struct {
	enableShards []shard.Index
}
