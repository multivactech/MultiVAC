package mock

import (
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

// SyncMgr is a fake struct.
type SyncMgr struct{}

// Start is a fake method that does nothing.
func (fsm *SyncMgr) Start() {}

// AcceptMsg is a fake method that does nothing.
func (fsm *SyncMgr) AcceptMsg(msg wire.Message) {}

// MaybeSync is a fake method that does nothing.
func (fsm *SyncMgr) MaybeSync() {}

// OnNewPeer is a fake method that does nothing.
func (fsm *SyncMgr) OnNewPeer(cp *connection.ConnPeer, t config.NodeType) {}

// OnPeerDone is a fake method that does nothing.
func (fsm *SyncMgr) OnPeerDone(cp *connection.ConnPeer, t config.NodeType) {}
