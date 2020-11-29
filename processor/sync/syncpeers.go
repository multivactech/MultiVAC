/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

const (
	maxSyncCandidates = 10
)

// A simple sync peer management class per worker.
// NOTE(huangsz): Better to be handled in a centralized place when p2p refactor is done.
type syncPeerManager struct {
	shardIdx    shard.Index
	curSyncPeer *syncPeer
	candidates  map[*connection.ConnPeer]*syncPeer
}

type syncPeer struct {
	cp       *connection.ConnPeer
	nodeType config.NodeType
}

func (p *syncPeer) equals(a *syncPeer) bool {
	if a == nil {
		return p == a
	}
	return p.cp == a.cp && p.nodeType == a.nodeType
}

func newSyncPeerManager(shardIdx shard.Index) *syncPeerManager {
	return &syncPeerManager{shardIdx: shardIdx, candidates: make(map[*connection.ConnPeer]*syncPeer, maxSyncCandidates)}
}

func (mgr *syncPeerManager) addSyncPeerCandidate(cp *connection.ConnPeer, t config.NodeType) {
	mgr.candidates[cp] = &syncPeer{cp: cp, nodeType: t}
}

// Returns one peer for sync.
// Will choose one when available if there isn't one yet.
// Returns nil if there is no sync peer.
func (mgr *syncPeerManager) getSyncPeer() *syncPeer {
	if mgr.curSyncPeer == nil {
		for _, p := range mgr.candidates {
			shards := p.cp.GetShards()
			for _, shardID := range shards {
				if mgr.shardIdx == shardID {
					mgr.curSyncPeer = p
					break
				}
			}
		}
	}
	return mgr.curSyncPeer
}

// Returns whether or not the removed peer is the current sync peer.
func (mgr *syncPeerManager) removePeerCandidate(cp *connection.ConnPeer, t config.NodeType) bool {
	delete(mgr.candidates, cp)
	m := &syncPeer{cp: cp, nodeType: t}
	if mgr.isCurSyncPeer(m) {
		mgr.curSyncPeer = nil
		return true
	}
	return false
}

func (mgr *syncPeerManager) isCurSyncPeer(sp *syncPeer) bool {
	return mgr.curSyncPeer != nil && mgr.curSyncPeer.cp == sp.cp && mgr.curSyncPeer.nodeType == sp.nodeType
}
