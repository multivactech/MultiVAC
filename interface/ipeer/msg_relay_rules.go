// Copyright (c) 2018-present, MultiVAC Foundation.
//This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ipeer

import (
	"github.com/multivactech/MultiVAC/model/wire"
)

// MsgRelayRule is used to deal with relay message.
type MsgRelayRule interface {
	// Filter will filter the message, it should follow the rules below:
	// 1.If an error occurs, return the original message.
	// 2.If the message does not need to be relayed, return nil.
	Filter(msg wire.Message) (wire.Message, error)
}
