// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"errors"
	"github.com/multivactech/MultiVAC/model/wire"
)

// ConfirmationMessageRelyRule is the relay rule of MsgBlockConfirmation, The rules are as follows:
// 1. If a peer receives the ConfirmationMessage with the same block, then returns a new message only with votes.
//    The votes in the message are the peer has never seen before.
// 2. If a peer receives other type of message, return it.
type ConfirmationMessageRelyRule struct {
	// todo(zhaozheng):seems unused.
	//message wire.Message
	//cache   []wire.Message
}

// Filter filter the message which is not MsgBlockConfirmation。
func (rule *ConfirmationMessageRelyRule) Filter(message wire.Message) (wire.Message, error) {

	// TODO(#131): complete it
	msg := message.(*wire.MsgBlockConfirmation)
	if msg == nil {
		return message, errors.New("can't convert message to MsgBlockConfirmation")
	}

	return msg, nil
}
