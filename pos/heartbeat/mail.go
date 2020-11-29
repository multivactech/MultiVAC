/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package heartbeat

import "github.com/multivactech/MultiVAC/processor/shared/message"

const (
	evtReceiveReq message.EventTopic = iota
	evtHasReq
	evtPerceivedCountReq
	evtStartReq
	evtStopReq
	evtCheckConfirmation
)
