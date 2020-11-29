/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcserver

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/p2p/peer"
	"sync/atomic"
)

// rpcPeer provides a peer for use with the RPC server and implements the
// RPCServerPeer interface.
type rpcPeer connection.ConnPeer

// Ensure rpcPeer implements the RPCServerPeer interface.
var _ Peer = (*rpcPeer)(nil)

// ToPeer returns the underlying peer instance.
//
// This function is safe for concurrent access and is part of the RPCServerPeer
// interface implementation.
func (p *rpcPeer) ToPeer() *peer.Peer {
	if p == nil {
		return nil
	}
	return (*connection.ConnPeer)(p).Peer
}

// IsTxRelayDisabled returns whether or not the peer has disabled transaction
// relay.
//
// This function is safe for concurrent access and is part of the RPCServerPeer
// interface implementation.
func (p *rpcPeer) IsTxRelayDisabled() bool {
	return (*connection.ConnPeer)(p).GetDisableRelayTx()
}

// BanScore returns the current integer value that represents how close the peer
// is to being banned.
//
// This function is safe for concurrent access and is part of the RPCServerPeer
// interface implementation.
func (p *rpcPeer) BanScore() uint32 {
	return (*connection.ConnPeer)(p).GetBanScore()
}

// FeeFilter returns the requested current minimum fee rate for which
// transactions should be announced.
//
// This function is safe for concurrent access and is part of the RPCServerPeer
// interface implementation.
func (p *rpcPeer) FeeFilter() int64 {
	return atomic.LoadInt64(&(*connection.ConnPeer)(p).CpFeeFilter)
}

// rpcConnManager provides a connection manager for use with the RPC server and
// implements the RPCServerConnManager interface.
type rpcConnManager struct {
	server *connection.ConnServer
}

// Ensure rpcConnManager implements the RPCServerConnManager interface.
var _ Manager = &rpcConnManager{}

// Connect adds the provided address as a new outbound peer.  The permanent flag
// indicates whether or not to make the peer persistent and reconnect if the
// connection is lost.  Attempting to connect to an already existing peer will
// return an error.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) Connect(addr string, permanent bool) error {
	replyChan := make(chan error)
	cm.server.QueryChannel() <- connection.ConnectNodeMsg{
		Addr:      addr,
		Permanent: permanent,
		Reply:     replyChan,
	}
	return <-replyChan
}

// RemoveByID removes the peer associated with the provided id from the list of
// persistent peers.  Attempting to remove an id that does not exist will return
// an error.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) RemoveByID(id int32) error {
	replyChan := make(chan error)
	cm.server.QueryChannel() <- connection.RemoveNodeMsg{
		Cmp:   func(sp *connection.ConnPeer) bool { return sp.ID() == id },
		Reply: replyChan,
	}
	return <-replyChan
}

// RemoveByAddr removes the peer associated with the provided address from the
// list of persistent peers.  Attempting to remove an address that does not
// exist will return an error.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) RemoveByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.QueryChannel() <- connection.RemoveNodeMsg{
		Cmp:   func(sp *connection.ConnPeer) bool { return sp.Addr() == addr },
		Reply: replyChan,
	}
	return <-replyChan
}

// DisconnectByID disconnects the peer associated with the provided id.  This
// applies to both inbound and outbound peers.  Attempting to remove an id that
// does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) DisconnectByID(id int32) error {
	replyChan := make(chan error)
	cm.server.QueryChannel() <- connection.DisconnectNodeMsg{
		Cmp:   func(sp *connection.ConnPeer) bool { return sp.ID() == id },
		Reply: replyChan,
	}
	return <-replyChan
}

// DisconnectByAddr disconnects the peer associated with the provided address.
// This applies to both inbound and outbound peers.  Attempting to remove an
// address that does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) DisconnectByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.QueryChannel() <- connection.DisconnectNodeMsg{
		Cmp:   func(sp *connection.ConnPeer) bool { return sp.Addr() == addr },
		Reply: replyChan,
	}
	return <-replyChan
}

// QueryPeerShardCount returns the number of shards the current Server has.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) QueryPeerShardCount() int32 {
	return cm.server.QueryPeerShardCount()
}

// ConnectedCount returns the number of currently connected peers.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) ConnectedCount() int32 {
	return cm.server.ConnectedCount()
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) NetTotals() (uint64, uint64) {
	return cm.server.NetTotals()
}

// ConnectedPeers returns an array consisting of all connected peers.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) ConnectedPeers() []Peer {
	replyChan := make(chan []*connection.ConnPeer)
	cm.server.QueryChannel() <- connection.GetPeersMsg{Reply: replyChan}
	serverPeers := <-replyChan

	// Convert to RPC server peers.
	peers := make([]Peer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

// PersistentPeers returns an array consisting of all the added persistent
// peers.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) PersistentPeers() []Peer {
	replyChan := make(chan []*connection.ConnPeer)
	cm.server.QueryChannel() <- connection.GetAddedNodesMsg{Reply: replyChan}
	serverPeers := <-replyChan

	// Convert to generic peers.
	peers := make([]Peer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

// BroadcastMessage sends the provided message to all currently connected peers.
//
// This function is safe for concurrent access and is part of the
// RPCServerConnManager interface implementation.
func (cm *rpcConnManager) BroadcastMessage(msg wire.Message) {
	switch msg.(type) {
	case *wire.MsgFetchTxs, *wire.MsgFetchInit, *wire.MsgBlock, *wire.SlimBlock, *wire.MsgGetShardAddr:
		connection.GlobalConnServer.BroadcastMessage(msg, &connection.BroadcastParams{
			ToNode:  connection.ToStorageNode,
			ToShard: getMsgShard(msg),
		})
	case *wire.MsgBlockConfirmation, *wire.MsgStartNet:
		connection.GlobalConnServer.BroadcastMessage(msg, &connection.BroadcastParams{
			ToNode:  connection.ToAllNode,
			ToShard: getMsgShard(msg),
		})
	default:
		connection.GlobalConnServer.BroadcastMessage(msg, &connection.BroadcastParams{
			ToNode:  connection.ToMinerNode,
			ToShard: getMsgShard(msg),
		})
	}
}

// AddRebroadcastInventory adds the provided inventory to the list of
// inventories to be rebroadcast at random intervals until they show up in a
// block.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) AddRebroadcastInventory(iv *wire.InvVect, data interface{}) {
	// cm.AddRebroadcastInventory(iv, data)
}

func getMsgShard(msg wire.Message) shard.Index {
	switch msg := msg.(type) {
	case wire.ShardMessage:
		return msg.GetShardIndex()
	}
	return connection.ToAllShard
}
