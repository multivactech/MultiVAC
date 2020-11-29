/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

import (
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// NewService returns a new instance of Service.
func NewService() *Service {
	s := &Service{actorCtx: message.NewActorContext()}
	s.actor = newDiskBlockChain()
	s.actorCtx.StartActor(s.actor)
	return s
}
