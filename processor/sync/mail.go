/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	evtMsg message.EventTopic = iota
	evtAddPeer
	evtDonePeer
	evtSyncReq
)

type peerMail struct {
	connPeer *connection.ConnPeer
	nodeType config.NodeType
}

type storageMissInfo struct {
}

type syncReq struct{}
