/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sync

import (
	"testing"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/shard"
)

func TestSyncPeerManager_GetSyncPeer(test *testing.T) {
	tests := []struct {
		candidate *syncPeer
		desc      string
	}{
		{
			desc: "GetSyncPeer: No candidate",
		},
		{
			desc:      "GetSyncPeer: With candidate",
			candidate: createPeerForTest(true, config.MinerNode),
		},
	}
	for _, t := range tests {
		mgr := newSyncPeerManager(shard.IDToShardIndex(TestCurrentShard))
		if t.candidate != nil {
			mgr.addSyncPeerCandidate(t.candidate.cp, t.candidate.nodeType)
		}
		p := mgr.getSyncPeer()
		if (p == nil && t.candidate != nil) || !p.equals(t.candidate) {
			test.Errorf("%s expected: %v, got: %v", t.desc, t.candidate, p)
		}
	}
}

func TestSyncPeerManager_IsCurrentSyncPeer(test *testing.T) {
	mgr := newSyncPeerManager(shard.IDToShardIndex(TestCurrentShard))
	p := createPeerForTest(true, config.MinerNode)
	mgr.addSyncPeerCandidate(p.cp, p.nodeType)

	if mgr.isCurSyncPeer(p) {
		test.Errorf("IsCurrentSyncPeer before requesting a sync peer, expected false, got true")
	}
	mgr.getSyncPeer()
	if !mgr.isCurSyncPeer(p) {
		test.Errorf("IsCurrentSyncPeer expected true, got false")
	}
	a := createPeerForTest(true, config.MinerNode)
	if mgr.isCurSyncPeer(a) {
		test.Errorf("IsCurrentSyncPeer expected false, got true")
	}
}

func TestSyncPeerManager_RemovePeerCandidate(test *testing.T) {
	tests := []struct {
		desc              string
		candidate         *syncPeer
		removeCurSyncPeer bool
		expected          bool
	}{
		{
			desc:     "No candidate",
			expected: false,
		},
		{
			desc:              "Remove current sync peer",
			expected:          true,
			removeCurSyncPeer: true,
			candidate:         createPeerForTest(true, config.StorageNode),
		},
		{
			desc:      "Remove non current sync peer",
			expected:  false,
			candidate: createPeerForTest(true, config.StorageNode),
		},
	}
	for _, t := range tests {
		mgr := newSyncPeerManager(shard.IDToShardIndex(TestCurrentShard))
		if t.candidate != nil {
			mgr.addSyncPeerCandidate(t.candidate.cp, t.candidate.nodeType)
		}
		p := mgr.getSyncPeer()
		if p == nil || !t.removeCurSyncPeer {
			p = createPeerForTest(true, config.StorageNode)
		}
		if r := mgr.removePeerCandidate(p.cp, p.nodeType); r != t.expected {
			test.Errorf("RemovePeerCandidate [%v], expected: %v, got: %v", t.desc, t.expected, r)
		}
	}
}
