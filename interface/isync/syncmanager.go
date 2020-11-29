// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package isync defines the interface of sync
package isync

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

const (
	// MaxSyncHeadersPerMsg is the max number of message that a peer chould sync
	MaxSyncHeadersPerMsg = 10
)

// SyncManager used to sync header or block from peer
type SyncManager interface {
	// Start will start the processing of sync management.
	Start()

	// AcceptMsg accepts a new message and maybe act on it.
	AcceptMsg(msg wire.Message)

	// Notifies SyncManager to request a sync, whether or not a sync will actually happen depending on internal logic.
	MaybeSync()

	// TODO(huangsz): Remove these peer relevant interfaces when p2p refactor is done, we shall register a listener
	// to PeerManager instead.
	OnNewPeer(cp *connection.ConnPeer, t config.NodeType)
	OnPeerDone(cp *connection.ConnPeer, t config.NodeType)
}
