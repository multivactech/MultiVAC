/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

import "github.com/multivactech/MultiVAC/processor/shared/message"

const (
	evtReceiveBlock message.EventTopic = iota
	evtReceiveHeader
	evtShardHeight
	evtBlockByShardAndHeight
	evtHeaderByShardAndHeight
	evtBlockByHash
	evtHeaderByHash
	evtShardHeaderHashes
	evtSetTrigger
	evtReceiveSlimBlock
	evtSlimBlockMsgByShardAndHeight
	evtSmartContractByAddress
	evtSmartContractCodeOut
	evtSmartContractShardInitOut
	evtReceiveSmartContractShardInitOut
)
