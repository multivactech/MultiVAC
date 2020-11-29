package connection

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/multivactech/MultiVAC/configs/params"

	"github.com/multivactech/MultiVAC/model/chaincfg"
	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/addrmgr"
	"github.com/multivactech/MultiVAC/p2p/connmgr"
	"github.com/multivactech/MultiVAC/p2p/peer"
)

const (
	// defaultServices describes the default services that are supported by
	// the connServer.
	defaultServices = wire.SFNodeNetwork | wire.SFNodeBloom |
		wire.SFNodeWitness | wire.SFNodeCF

	// defaultRequiredServices describes the default services that are
	// required to be supported by outbound peers.
	defaultRequiredServices = wire.SFNodeNetwork

	// defaultTargetOutbound is the default number of outbound peers to target.
	defaultTargetOutbound = 8

	// connectionRetryInterval is the base amount of time to wait in between
	// retries when connecting to persistent peers.  It is adjusted by the
	// number of retries such that there is a retry backoff.
	connectionRetryInterval = time.Second * 5

	// checkConnectedShardTickInterval is the time that server check whether it connecting to enough different shards.
	checkConnectedShardTickInterval = time.Second * 10
)

// GossipNode defines an abstract node and its given method.
type GossipNode interface {
	BroadcastMessage(msg wire.Message, par *BroadcastParams)
	RegisterChannels(dispatch *MessagesAndReceiver)
	HandleMessage(msg wire.Message)
}

// GlobalConnServer is used to register handler.
var GlobalConnServer GossipNode

var (
	// userAgentName is the user agent name and is used to help identify
	// ourselves to other bitcoin peers.
	userAgentName = "btcd"

	// userAgentVersion is the user agent version and is used to help
	// identify ourselves to other bitcoin peers.
	userAgentVersion = fmt.Sprintf("%d.%d.%d", config.AppMajor, config.AppMinor, config.AppPatch)

	// default shard when no sharding is enabled.
	// todo(ymh):check if it is useful?
	//defaultShard = shard.IDToShardIndex(0)
)

type toNodeType byte

const (
	// ToStorageNode defines the role that message should be sent to storage nodes.
	ToStorageNode toNodeType = iota
	// ToMinerNode defines the role that message should be sent to miner nodes.
	ToMinerNode
	// ToAllNode defines the role that message should be sent to all nodes.
	ToAllNode
)

// ToAllShard defines that the message should go to all shard.
const ToAllShard shard.Index = params.GenesisNumberOfShards

// onionAddr implements the net.Addr interface and represents a tor address.
type onionAddr struct {
	addr string
}

// String returns the onion address.
//
// This is part of the net.Addr interface.
func (oa *onionAddr) String() string {
	return oa.addr
}

// Network returns "onion".
//
// This is part of the net.Addr interface.
func (oa *onionAddr) Network() string {
	return "onion"
}

// Ensure onionAddr implements the net.Addr interface.
var _ net.Addr = (*onionAddr)(nil)

// onionAddr implements the net.Addr interface with two struct fields
type simpleAddr struct {
	net, addr string
}

// String returns the address.
//
// This is part of the net.Addr interface.
func (a simpleAddr) String() string {
	return a.addr
}

// Network returns the network.
//
// This is part of the net.Addr interface.
func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}

// broadcastMsg provides the ability to house a message to be broadcast
// to all connected peers except specified excluded peers. If shard is is not empty,
// broadcast to peers in specific shard.
type broadcastMsg struct {
	hash         string
	message      wire.Message
	shard        shard.Index
	excludePeers []*ConnPeer
	toNode       toNodeType
}

type peerState struct {
	currentStorageNode sync.Map
	currentMinerNode   sync.Map
	shardNumber        []int32
	persistentPeers    map[int32]*ConnPeer
	banned             map[string]time.Time
	count              int32
}

// Count returns the count of all known peers.
func (ps *peerState) Count() int {
	if ps == nil {
		return 0
	}
	return int(atomic.LoadInt32(&ps.count))
}

// forAllPeers is a helper function that runs closure on all peers known to
// peerState.
func (ps *peerState) forAllPeers(closure func(cp *ConnPeer)) {
	ps.currentStorageNode.Range(func(_, v interface{}) bool {
		closure(v.(*ConnPeer))
		return true
	})

	ps.currentMinerNode.Range(func(_, v interface{}) bool {
		closure(v.(*ConnPeer))
		return true
	})
}

// forAllPeers is a helper function that runs closure on all storage node peers known to
// peerState.
func (ps *peerState) forAllStoragePeers(closure func(cp *ConnPeer)) {
	ps.currentStorageNode.Range(func(_, v interface{}) bool {
		closure(v.(*ConnPeer))
		return true
	})
}

// forAllMinerPeers is a helper function that runs closure on all miner node peers known to
// peerState.
func (ps *peerState) forAllMinerPeers(closure func(cp *ConnPeer)) {
	ps.currentMinerNode.Range(func(_, v interface{}) bool {
		closure(v.(*ConnPeer))
		return true
	})
}

// SelfPeer means local peer.
var SelfPeer = &ConnPeer{}

// ConnServer defines the data structure of connect server.
type ConnServer struct {
	bytesReceived uint64 // Total bytes received from all peers since start.
	bytesSent     uint64 // Total bytes sent by all peers since start.

	// A list of enabled shards of this server.
	enabledShards []shard.Index

	isStorageNode bool

	shutdown int32

	chainParams *chaincfg.Params
	addrManager *addrmgr.AddrManager
	connManager *connmgr.ConnManager

	newPeers             chan *ConnPeer
	donePeers            chan *ConnPeer
	newConnPeers         chan *ConnPeer
	newPeersServer       chan *ConnPeer
	donePeersServer      chan *ConnPeer
	banPeers             chan *ConnPeer
	query                chan interface{}
	wg                   sync.WaitGroup
	broadcast            chan *broadcastMsg
	peerState            *peerState
	nat                  NAT
	services             wire.ServiceFlag
	storageNodeAddr      []string
	ConnectToEnoughPeers chan struct{}
	messagesSent         *lru.Cache
	requestMsg           sync.Map
	quit                 chan struct{}
	handlers             *Multiplexer
}

// GetPeerShardCountMsg defines a type of query request to show how many shards miner has join.
type GetPeerShardCountMsg struct {
	Reply chan int32
}

// GetShardPeersCountMsg defines a type of query request to show how many miners in every shard.
type GetShardPeersCountMsg struct {
	Reply chan []int32
}

// GetShardPeersAddrMsg defines a type of query request to show the list of miners' listen address.
type GetShardPeersAddrMsg struct {
	Shard []shard.Index
	Reply chan []string
}

// GetConnCountMsg defines a type of query request to return the number of connected peer.
type GetConnCountMsg struct {
	Reply chan int32
}

// GetPeersMsg defines a type of query request to return all the connected peer.
type GetPeersMsg struct {
	Reply chan []*ConnPeer
}

// GetAddedNodesMsg defines a type of query request to return a slice of the relevant peers.
type GetAddedNodesMsg struct {
	Reply chan []*ConnPeer
}

// DisconnectNodeMsg defines a type of query request to Check inbound peers
type DisconnectNodeMsg struct {
	Cmp   func(*ConnPeer) bool
	Reply chan error
}

// ConnectNodeMsg defines a type of query request to return connected node message.
type ConnectNodeMsg struct {
	Addr      string
	Permanent bool
	Reply     chan error
}

// RemoveNodeMsg defines a type of query request to remove the connection if it's suitable for cmp.
type RemoveNodeMsg struct {
	Cmp   func(*ConnPeer) bool
	Reply chan error
}

// QueryChannel returns the channel that query message.
func (s *ConnServer) QueryChannel() chan interface{} {
	return s.query
}

// handleQuery is the central handler for all queries and commands from other
// goroutines related to peer state.
func (s *ConnServer) handleQuery(state *peerState, querymsg interface{}) {
	switch msg := querymsg.(type) {
	case GetPeerShardCountMsg:
		shardSet := make(map[uint32]struct{})
		state.forAllPeers(func(cp *ConnPeer) {
			if !cp.Connected() {
				return
			}
			for _, shardIndex := range cp.shards {
				shardSet[shardIndex.GetID()] = struct{}{}
			}
		})
		msg.Reply <- int32(len(shardSet))

	case GetShardPeersCountMsg:
		peerNum := make([]int32, params.GenesisNumberOfShards)
		state.forAllMinerPeers(func(cp *ConnPeer) {
			if !cp.Connected() {
				return
			}
			for _, shardIndex := range cp.shards {
				peerNum[shardIndex.GetID()]++
			}
		})
		msg.Reply <- peerNum
	case GetShardPeersAddrMsg:
		shardNum := make(map[uint32]int32)
		for _, shard := range msg.Shard {
			shardNum[shard.GetID()] = 0
		}
		var reply []string

		s.peerState.forAllMinerPeers(func(cp *ConnPeer) {
			needReply := false
			if cp.listenerAddress == "" {
				return
			}
			for _, shardI := range cp.shards {
				if _, ok := shardNum[shardI.GetID()]; ok {
					shardNum[shardI.GetID()]++
					needReply = true
					break
				}
			}
			if needReply {
				reply = append(reply, cp.listenerAddress)
			}
		})
		msg.Reply <- reply

	case GetConnCountMsg:
		nconnected := int32(0)
		state.forAllPeers(func(cp *ConnPeer) {
			if cp.Connected() {
				nconnected++
			}
		})
		msg.Reply <- nconnected

	case GetPeersMsg:
		peers := make([]*ConnPeer, 0, state.Count())
		state.forAllPeers(func(cp *ConnPeer) {
			if !cp.Connected() {
				return
			}
			peers = append(peers, cp)
		})
		msg.Reply <- peers

	case ConnectNodeMsg:
		// TODO: duplicate oneshots?
		// Limit max number of total peers.
		if state.Count() >= config.GlobalConfig().MaxPeers {
			msg.Reply <- errors.New("max peers reached")
			return
		}

		for _, peer := range state.persistentPeers {
			if peer.Addr() == msg.Addr {
				if msg.Permanent {
					msg.Reply <- errors.New("peer already connected")
				} else {
					msg.Reply <- errors.New("peer exists as a permanent peer")
				}
				return
			}
		}

		netAddr, err := addrStringToNetAddr(msg.Addr)
		if err != nil {
			msg.Reply <- err
			return
		}

		// TODO: if too many, nuke a non-perm peer.
		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: msg.Permanent,
		})
		msg.Reply <- nil
	case RemoveNodeMsg:
		found := disconnectPeer(state, msg.Cmp)
		if found {
			msg.Reply <- nil
		} else {
			msg.Reply <- errors.New("peer not found")
		}
	case GetAddedNodesMsg:
		// Respond with a slice of the relevant peers.
		peers := make([]*ConnPeer, 0, len(state.persistentPeers))
		for _, cp := range state.persistentPeers {
			peers = append(peers, cp)
		}
		msg.Reply <- peers
	case DisconnectNodeMsg:
		// Check inbound peers. We pass a nil callback since we don't
		// require any additional actions on disconnect for inbound peers.
		found := disconnectPeer(state, msg.Cmp)
		if found {
			msg.Reply <- nil
			return
		}
		if found {
			// If there are multiple outbound connections to the same
			// ip:port, continue disconnecting them all until no such
			// peers are found.
			for found {
				found = disconnectPeer(state, msg.Cmp)
			}
			msg.Reply <- nil
			return
		}

		msg.Reply <- errors.New("peer not found")
	}
}

// inboundPeerConnected is invoked by the connection manager when a new inbound
// connection is established.  It initializes a new inbound Server peer
// instance, associates it with the connection, and starts a goroutine to wait
// for disconnection.
func (s *ConnServer) inboundPeerConnected(conn net.Conn) {
	cp := newConnPeer(s, false)
	cfg := config.GlobalConfig()
	cp.isWhitelisted = cfg.IsIPWhitelisted(conn.RemoteAddr())
	cp.Peer = peer.NewInboundPeer(newPeerConfig(cp))
	logger.ConnServerLogger().Debugf("New Inbound Peer %s", cp)
	cp.AssociateConnection(conn,
		wire.UpdatePeerInfoMessage{
			Shards:               s.getEnabledShards(),
			IsDNS:                config.GlobalConfig().RunAsDNSSeed,
			IsStorageNode:        s.isStorageNode,
			StorageNode:          s.getStorageNodeAddr(),
			RPCListenerAddress:   cfg.RPCListeners[0],
			LocalListenerAddress: cfg.Listeners[0],
		},
	)
	logger.ConnServerLogger().Debugf("New Inbound Peer %s", cp)
	go s.peerDoneHandler(cp)
}

// outboundPeerConnected is invoked by the connection manager when a new
// outbound connection is established.  It initializes a new outbound Server
// peer instance, associates it with the relevant state such as the connection
// request instance and the connection itself, and finally notifies the address
// manager of the attempt.
func (s *ConnServer) outboundPeerConnected(c *connmgr.ConnReq, conn net.Conn) {
	cp := newConnPeer(s, c.Permanent)
	p, err := peer.NewOutboundPeer(newPeerConfig(cp), c.Addr.String())
	if err != nil {
		logger.ConnServerLogger().Debugf("Cannot create outbound peer %s: %v", c.Addr, err)
		s.connManager.Disconnect(c.ID())
	}
	cp.Peer = p
	cp.connReq = c
	cfg := config.GlobalConfig()
	cp.isWhitelisted = cfg.IsIPWhitelisted(conn.RemoteAddr())
	cp.AssociateConnection(conn, wire.UpdatePeerInfoMessage{
		Shards:               s.getEnabledShards(),
		IsDNS:                config.GlobalConfig().RunAsDNSSeed,
		IsStorageNode:        s.isStorageNode,
		StorageNode:          s.getStorageNodeAddr(),
		RPCListenerAddress:   cfg.RPCListeners[0],
		LocalListenerAddress: cfg.Listeners[0],
	})
	logger.ConnServerLogger().Debugf("New Outbound Peer %s", cp)
	go s.peerDoneHandler(cp)
	s.addrManager.Attempt(cp.NA())
}

// UpdatePeerInfo broadcast UpdatePeerInfoMessage when resharding
func (s *ConnServer) UpdatePeerInfo(shards []shard.Index) {
	if !s.SetEnabledShards(shards) {
		return
	}
	cfg := config.GlobalConfig()
	s.peerState.forAllPeers(func(cp *ConnPeer) {
		cp.QueueMessage(&wire.UpdatePeerInfoMessage{
			Shards:               s.getEnabledShards(),
			IsDNS:                config.GlobalConfig().RunAsDNSSeed,
			IsStorageNode:        s.isStorageNode,
			StorageNode:          s.getStorageNodeAddr(),
			RPCListenerAddress:   cfg.RPCListeners[0],
			LocalListenerAddress: cfg.Listeners[0],
		}, nil)
	})
}

// Maintain enough connected shards. Periodically send possible connecting the peer request.
func (s *ConnServer) connectedShardsMaintainer() {
	for s.needNewPeer() {
		time.Sleep(time.Second)
	}
	s.ConnectToEnoughPeers <- struct{}{}

	ticker := time.NewTicker(checkConnectedShardTickInterval)
	defer ticker.Stop()

out:
	for {
		select {
		case <-ticker.C:
			s.needNewPeer()

		case <-s.quit:
			break out
		}
	}
}

func (s *ConnServer) needNewPeer() bool {
	if len(s.getStorageNodeAddr()) < 1 {
		logger.ConnServerLogger().Debugf("don't connect to enough storagenode")
		s.connManager.AddNewConnections()
		return true
	}
	var requestShard []shard.Index
	shardPeersCount := s.QueryShardPeersCount()
	for index, count := range shardPeersCount {
		if count < config.GlobalConfig().MinShardPeers {
			requestShard = append(requestShard, shard.Index(index))
		}
	}
	if len(requestShard) > 0 {
		logger.ConnServerLogger().Debugf("don't connect to enough peer, except shard %v ==== %v", requestShard, shardPeersCount)
		msg := &wire.MsgGetShardAddr{ShardIndex: requestShard}
		s.BroadcastMessage(msg, &BroadcastParams{
			ToNode:  msgToNode(msg),
			ToShard: getMsgShard(msg),
		})
		return true
	}
	return false
}

func (s *ConnServer) connectToAddr(addrs []string) {
	for _, addr := range addrs {
		if _, ok := s.peerState.currentMinerNode.Load(addr); !ok {
			netAddr, err := addrStringToNetAddr(addr)
			if err == nil {
				logger.ConnServerLogger().Debugf("connect to addr %v", addr)
				s.connManager.Connect(&connmgr.ConnReq{
					Addr:      netAddr,
					Permanent: false,
				})
			}
		}
	}
}

// directionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

// handleBanPeerMsg deals with banning peers.  It is invoked from the
// peerHandler goroutine.
func (s *ConnServer) handleBanPeerMsg(state *peerState, cp *ConnPeer) {
	host, _, err := net.SplitHostPort(cp.Addr())
	if err != nil {
		logger.ConnServerLogger().Debugf("can't split ban peer %s %v", cp.Addr(), err)
		return
	}
	direction := directionString(cp.Inbound())
	cfg := config.GlobalConfig()
	logger.ConnServerLogger().Infof("Banned peer %s (%s) for %v", host, direction,
		cfg.BanDuration)
	state.banned[host] = time.Now().Add(cfg.BanDuration)
}

// handleBroadcastMsg deals with broadcasting messages to peers.  It is invoked
// from the peerHandler goroutine.
func (s *ConnServer) handleBroadcastMsg(state *peerState, bmsg *broadcastMsg) {
	//logger.ConnServerLogger().Infof("Going to broadcast message: %v", bmsg)
	broadcastFunc := func(cp *ConnPeer) {
		if cp.tryBroadcastMsg(bmsg) {
			s.recordBroadcastMsgForMonitor(bmsg)
		}
	}
	switch bmsg.toNode {
	case ToMinerNode:
		state.forAllMinerPeers(broadcastFunc)
	case ToStorageNode:
		state.forAllStoragePeers(broadcastFunc)
	case ToAllNode:
		state.forAllPeers(broadcastFunc)
	}
}

func (s *ConnServer) recordBroadcastMsgForMonitor(bmsg *broadcastMsg) {
	//TODO(nanlin)
}

// handleAddPeerMsg deals with adding new peers.  It is invoked from the
// peerHandler goroutine.
func (s *ConnServer) handleAddPeerMsg(state *peerState, cp *ConnPeer) bool {
	if cp == nil {
		return false
	}

	// Ignore new peers if we're shutting down.
	if atomic.LoadInt32(&s.shutdown) != 0 {
		logger.ConnServerLogger().Infof("New peer %s ignored - server is shutting down", cp)
		cp.Disconnect()
		return false
	}

	// Disconnect banned peers.
	host, _, err := net.SplitHostPort(cp.Addr())
	if err != nil {
		logger.ConnServerLogger().Debugf("can't split hostport %v", err)
		cp.Disconnect()
		return false
	}
	if banEnd, ok := state.banned[host]; ok {
		if time.Now().Before(banEnd) {
			logger.ConnServerLogger().Debugf("Peer %s is banned for another %v - disconnecting",
				host, time.Until(banEnd))
			cp.Disconnect()
			return false
		}

		logger.ConnServerLogger().Infof("Peer %s is no longer banned", host)
		delete(state.banned, host)
	}

	// TODO: Check for max peers from a single IP.

	// Limit max number of total peers.
	cfg := config.GlobalConfig()
	if state.Count() >= cfg.MaxPeers {
		logger.ConnServerLogger().Infof("Max peers reached [%d] - disconnecting peer %s",
			cfg.MaxPeers, cp)
		cp.Disconnect()
		// TODO: how to handle permanent peers here?
		// they should be rescheduled.
		return false
	}

	// Add the new peer and start it.
	logger.ConnServerLogger().Debugf("New peer %s", cp)
	return true
}

func (s *ConnServer) handleAddConnPeerMsg(state *peerState, cp *ConnPeer) {
	// todo(ymh)
	//if cp.persistent {
	//	//state.persistentPeers[cp.ID()] = cp
	//}
	if cp.isStorageNode {
		s.setStorageNodeAddr(cp.listenerAddress)
		state.currentStorageNode.Store(cp.listenerAddress, cp)
	} else {
		state.currentMinerNode.Store(cp.listenerAddress, cp)
	}
	s.newPeersServer <- cp
	atomic.AddInt32(&state.count, 1)
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It is
// invoked from the peerHandler goroutine.
func (s *ConnServer) handleDonePeerMsg(state *peerState, cp *ConnPeer) {
	s.donePeersServer <- cp
	atomic.AddInt32(&state.count, -1)
	for _, shardIndex := range cp.shards {
		atomic.AddInt32(&cp.connServer.peerState.shardNumber[shardIndex.GetID()], -1)
	}
	state.currentMinerNode.Delete(cp.listenerAddress)
	state.currentStorageNode.Delete(cp.listenerAddress)
	logger.ConnServerLogger().Debugf("Removed peer %s", cp)

	if cp.connReq != nil {
		s.connManager.Disconnect(cp.connReq.ID())
	}

	// Update the address' last seen time if the peer has acknowledged
	// our version and has sent us its version as well.
	if cp.VerAckReceived() && cp.VersionKnown() && cp.NA() != nil {
		s.addrManager.Connected(cp.NA())
	}

	// If we get here it means that either we didn't know about the peer
	// or we purposefully deleted it.
}

// BroadcastMessage sends msg to all peers currently connected to the Server
// except those in the passed peers to exclude.
func (s *ConnServer) BroadcastMessage(msg wire.Message, par *BroadcastParams) {
	// XXX: Need to determine if this is an alert that has already been
	// broadcast and refrain from broadcasting again.
	if par == nil {
		//par = &BroadcastParams{} // For convenience, avoiding nil check
		panic("param must not be nil")
	}
	logger.ConnServerLogger().Debugf("broadcast:%v,toNode:%v,toShard:%v", msg.Command(), par.ToNode, par.ToShard)
	bmsg := &broadcastMsg{
		//excludePeers: par.exclPeers,
		shard:  par.ToShard,
		toNode: par.ToNode,
	}

	if config.GlobalConfig().EnableRelayMessage && wire.MsgShouldRelay(msg) {
		hash := getMsgHash(msg)
		s.messagesSent.Add(hash, msg)
		bmsg.hash = hash
	} else {
		bmsg.message = msg
	}
	if par.IncludeSelf {
		s.HandleMessage(msg)
	}
	s.broadcast <- bmsg
}

// HandleMessage handle messages sent to myself.
func (s *ConnServer) HandleMessage(msg wire.Message) {
	s.handlers.Handle(msg, nil)
}

func msgToNode(msg wire.Message) toNodeType {
	switch msg.(type) {
	case *wire.MsgFetchTxs, *wire.MsgFetchInit, *wire.MsgBlock, *wire.SlimBlock, *wire.MsgGetShardAddr:
		return ToStorageNode
	case *wire.MsgBlockConfirmation, *wire.MsgStartNet:
		return ToAllNode
	}
	return ToMinerNode
}

func (s *ConnServer) broadcastMessageHash(hash string, exclPeers []*ConnPeer, toShard shard.Index) {
	s.broadcast <- &broadcastMsg{
		hash:         hash,
		excludePeers: exclPeers,
		shard:        toShard,
		toNode:       ToMinerNode,
	}
}

// AddPeer adds a new peer that has already been connected to the Server.
func (s *ConnServer) AddPeer(cp *ConnPeer) {
	logger.ConnServerLogger().Debugf("add new peer %s", cp.String())
	s.newPeers <- cp
}

// BanPeer bans a peer that has already been connected to the Server by ip.
func (s *ConnServer) BanPeer(cp *ConnPeer) {
	s.banPeers <- cp
}

// AddBytesSent adds the passed number of bytes to the total bytes sent counter
// for the Server.  It is safe for concurrent access.
func (s *ConnServer) AddBytesSent(bytesSent uint64) {
	atomic.AddUint64(&s.bytesSent, bytesSent)
}

// AddBytesReceived adds the passed number of bytes to the total bytes received
// counter for the ConnServer.  It is safe for concurrent access.
func (s *ConnServer) AddBytesReceived(bytesReceived uint64) {
	atomic.AddUint64(&s.bytesReceived, bytesReceived)
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.  It is safe for concurrent access.
func (s *ConnServer) NetTotals() (uint64, uint64) {
	return atomic.LoadUint64(&s.bytesReceived),
		atomic.LoadUint64(&s.bytesSent)
}

// NewConnServer creates a connect server.
func NewConnServer(listenAddrs []string, chainParams *chaincfg.Params,
	newPeersServer chan *ConnPeer, donePeersServer chan *ConnPeer, isStorageNode bool) (*ConnServer, error) {
	cfg := config.GlobalConfig()

	services := defaultServices
	if cfg.NoPeerBloomFilters {
		services &^= wire.SFNodeBloom
	}
	if cfg.NoCFilters {
		services &^= wire.SFNodeCF
	}

	amgr := addrmgr.New(cfg.DataDir, !cfg.RunAsDNSSeed, btcdLookup)

	var listeners []net.Listener
	var nat NAT
	if !cfg.DisableListen {
		var err error
		listeners, nat, err = initListeners(amgr, listenAddrs, services)
		if err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, errors.New("no valid listen address")
		}
	}

	s := ConnServer{
		chainParams:          chainParams,
		addrManager:          amgr,
		newPeers:             make(chan *ConnPeer),
		donePeers:            make(chan *ConnPeer),
		newPeersServer:       newPeersServer,
		donePeersServer:      donePeersServer,
		banPeers:             make(chan *ConnPeer),
		query:                make(chan interface{}),
		newConnPeers:         make(chan *ConnPeer),
		broadcast:            make(chan *broadcastMsg),
		nat:                  nat,
		services:             services,
		ConnectToEnoughPeers: make(chan struct{}),
		quit:                 make(chan struct{}),
		// enabledShards is shard.ShardList when initializing connserver
		isStorageNode: isStorageNode,
	}

	s.peerState = &peerState{
		shardNumber:     make([]int32, params.GenesisNumberOfShards),
		persistentPeers: make(map[int32]*ConnPeer),
		banned:          make(map[string]time.Time),
	}

	// Only setup a function to return new addresses to connect to when
	// not running in connect-only mode.  The simulation network is always
	// in connect-only mode since it is only intended to connect to
	// specified peers and actively avoid advertising and connecting to
	// discovered peers in order to prevent it from becoming a public test
	// network.
	var newAddressFunc func() (net.Addr, error)
	if !cfg.SimNet && !cfg.RunAsDNSSeed && len(cfg.ConnectPeers) == 0 {
		newAddressFunc = func() (net.Addr, error) {
			for tries := 0; tries < 100; tries++ {
				addr := s.addrManager.GetAddress()
				if addr == nil {
					break
				}

				addrString := addrmgr.NetAddressKey(addr.NetAddress())
				returnString, err := addrStringToNetAddr(addrString)

				// TODO(nanlin): May add it back.
				// Address will not be invalid, local or unroutable
				// because addrmanager rejects those on addition.
				// Just check that we don't already have an address
				// in the same group so that we are not connecting
				// to the same network segment at the expense of
				// others.
				//key := addrmgr.GroupKey(addr.NetAddress())
				//if s.OutboundGroupCount(key) != 0 {
				//	continue
				//}

				// only allow recent nodes (10mins) after we failed 30
				// times
				if tries < 30 && time.Since(addr.LastAttempt()) < 10*time.Minute {
					continue
				}

				// TODO(nanlin): May add it back.
				// allow nondefault ports after 50 failed tries.
				//if tries < 50 && fmt.Sprintf("%d", addr.NetAddress().Port) !=
				//	config.ActiveNetParams().DefaultPort {
				//	continue
				//}

				logger.ConnServerLogger().Debugf("AddrManager.GetAddress(): %s", returnString.String())

				return returnString, err
			}

			return nil, errors.New("no valid connect address")
		}
	}
	// Create a connection manager.
	targetOutbound := defaultTargetOutbound
	if cfg.MaxPeers < targetOutbound {
		targetOutbound = cfg.MaxPeers
	}
	cmgr, err := connmgr.New(&connmgr.Config{
		Listeners:      listeners,
		OnAccept:       s.inboundPeerConnected,
		RetryDuration:  connectionRetryInterval,
		TargetOutbound: uint32(targetOutbound),
		Dial:           btcdDial,
		OnConnection:   s.outboundPeerConnected,
		GetNewAddress:  newAddressFunc,
	})
	if err != nil {
		return nil, err
	}
	s.connManager = cmgr

	// Start up persistent peers.
	permanentPeers := cfg.ConnectPeers
	if len(permanentPeers) == 0 {
		permanentPeers = cfg.AddPeers
	}
	for _, addr := range permanentPeers {
		netAddr, err := addrStringToNetAddr(addr)
		if err != nil {
			return nil, err
		}

		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: true,
		})
	}

	s.messagesSent, err = lru.New(params.RelayCache)
	if err != nil {
		logger.ConnServerLogger().Errorf("Can't create LRU cache, %v", err)
		return nil, err
	}
	SelfPeer.connServer = &s
	s.handlers = MakeMultiplexer()
	GlobalConnServer = &s
	return &s, nil
}

var enableShardLock sync.RWMutex

// SetEnabledShards set the enabled shards and return whether the enabled shards is set.
func (s *ConnServer) SetEnabledShards(shards []shard.Index) bool {
	enableShardLock.Lock()
	defer enableShardLock.Unlock()
	if len(s.enabledShards) != len(shards) {
		s.enabledShards = shards
		return true
	}
	mp := make(map[shard.Index]struct{})
	for _, s := range shards {
		mp[s] = struct{}{}
	}
	replace := false
	for _, v := range s.enabledShards {
		if _, ok := mp[v]; !ok {
			replace = true
			break
		}
	}
	if replace {
		s.enabledShards = shards
	}
	return replace
}
func (s *ConnServer) getEnabledShards() []shard.Index {
	enableShardLock.RLock()
	defer enableShardLock.RUnlock()
	return s.enabledShards
}

// Start begins accepting connections from peers.
func (s *ConnServer) Start() {
	logger.ConnServerLogger().Trace("Starting Connection Server")

	// Start the peer handler which in turn starts the address and block
	// managers.
	s.wg.Add(1)
	go s.peerHandler()

	if s.nat != nil {
		s.wg.Add(1)
		go s.upnpUpdateThread()
	}
}

// Stop gracefully shuts down the Server by stopping and disconnecting all
// peers and the main listener.
func (s *ConnServer) Stop() error {
	// Make sure this only happens once.
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		logger.ConnServerLogger().Infof("Conn Server is already in the process of shutting down")
		return nil
	}

	logger.ConnServerLogger().Warnf("Conn Server shutting down")

	// Signal the remaining goroutines to quit.
	close(s.quit)
	return nil
}

// peerDoneHandler handles peer disconnects by notifying the Server that it's
// done along with other performing other desirable cleanup.
func (s *ConnServer) peerDoneHandler(cp *ConnPeer) {
	cp.WaitForDisconnect()
	s.donePeers <- cp

	// Only tell sync manager we are gone if we ever told it we existed.
	if cp.VersionKnown() {
		logger.ConnServerLogger().Debugf("peer done %s", cp.Peer.String())
		// TODO(liang): dispatch it
		// s.syncManager.DonePeer(cp.Peer)

		/**mtvac
		// Evict any remaining orphans that were sent by the peer.
		numEvicted := s.txMemPool.RemoveOrphansByTag(mempool.Tag(cp.ID()))
		if numEvicted > 0 {
			txmpLog.Debugf("Evicted %d %s from peer %v (id %d)",
				numEvicted, pickNoun(numEvicted, "orphan",
					"orphans"), cp, cp.ID())
		}
		*/
	}
	close(cp.quit)
}

// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the Server, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
func (s *ConnServer) peerHandler() {
	// Start the address manager and sync manager, both of which are needed
	// by peers.  This is done here since their lifecycle is closely tied
	// to this handler and rather than adding more channels to sychronize
	// things, it's easier and slightly faster to simply start and stop them
	// in this handler.
	s.addrManager.Start()

	logger.ConnServerLogger().Tracef("Starting peer handler")

	if !config.GlobalConfig().DisableDNSSeed && !config.GlobalConfig().RunAsDNSSeed {
		// Add peers discovered through DNS to the address manager.
		connmgr.SeedFromDNS(config.ActiveNetParams().Params, defaultRequiredServices,
			btcdLookup, func(addrs []*wire.NetAddress) {
				// Bitcoind uses a lookup of the dns seeder here. This
				// is rather strange since the values looked up by the
				// DNS seed lookups will vary quite a lot.
				// to replicate this behaviour we put all addresses as
				// having come from the first one.
				s.addrManager.AddAddresses(addrs, addrs[0])
			})
	}
	go s.connManager.Start()

	if !config.GlobalConfig().DisableDNSSeed && !config.GlobalConfig().RunAsDNSSeed && !config.GlobalConfig().StorageNode {
		// Maintain enough peer in every shard.
		go s.connectedShardsMaintainer()
	}

out:
	for {
		select {
		// New peers connected to the Server.
		case p := <-s.newPeers:
			s.handleAddPeerMsg(s.peerState, p)

			// New Connpeers connected to the Server.
		case p := <-s.newConnPeers:
			s.handleAddConnPeerMsg(s.peerState, p)

			// Disconnected peers.
		case p := <-s.donePeers:
			s.handleDonePeerMsg(s.peerState, p)

			// Peer to ban.
		case p := <-s.banPeers:
			s.handleBanPeerMsg(s.peerState, p)

			// Message to broadcast to all connected peers except those
			// which are excluded by the message.
		case bmsg := <-s.broadcast:
			s.handleBroadcastMsg(s.peerState, bmsg)

		case qmsg := <-s.query:
			s.handleQuery(s.peerState, qmsg)

		case <-s.quit:
			// Disconnect all peers on Server shutdown.
			s.peerState.forAllPeers(func(cp *ConnPeer) {
				logger.ConnServerLogger().Tracef("Shutdown peer %s", cp)
				cp.Disconnect()
			})
			break out
		}
	}

	s.connManager.Stop()
	//s.syncManager.Stop()
	err := s.addrManager.Stop()
	logger.ConnServerLogger().Debugf("failed to stop address manager,err:%v", err)

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-s.newPeers:
		case <-s.donePeers:
		case <-s.broadcast:
		default:
			break cleanup
		}
	}
	s.wg.Done()
	logger.ConnServerLogger().Tracef("Peer handler done")
}

func (s *ConnServer) upnpUpdateThread() {
	// Go off immediately to prevent code duplication, thereafter we renew
	// lease every 15 minutes.
	timer := time.NewTimer(0 * time.Second)
	lport, _ := strconv.ParseInt(config.ActiveNetParams().DefaultPort, 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
			// TODO: pick external port  more cleverly
			// TODO: know which ports we are listening to on an external net.
			// TODO: if specific listen port doesn't work then ask for wildcard
			// listen port?
			// XXX this assumes timeout is in seconds.
			listenPort, err := s.nat.AddPortMapping("tcp", int(lport), int(lport),
				"btcd listen port", 20*60)
			if err != nil {
				logger.ConnServerLogger().Warnf("can't add UPnP port mapping: %v", err)
			}
			if first && err == nil {
				// TODO: look this up periodically to see if upnp domain changed
				// and so did ip.
				externalip, err := s.nat.GetExternalAddress()
				if err != nil {
					logger.ConnServerLogger().Warnf("UPnP can't get external address: %v", err)
					continue out
				}
				na := wire.NewNetAddressIPPort(externalip, uint16(listenPort),
					s.services)
				err = s.addrManager.AddLocalAddress(na, addrmgr.UpnpPrio)

				if err != nil {
					// todo(ymh):check if it needs to DeletePortMapping.
					logger.ConnServerLogger().Debugf("failed to add local address,err:%v", err)
					// XXX DeletePortMapping?
				}
				logger.ConnServerLogger().Warnf("Successfully bound via UPnP to %s", addrmgr.NetAddressKey(na))
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("tcp", int(lport), int(lport)); err != nil {
		logger.ConnServerLogger().Warnf("unable to remove UPnP port mapping: %v", err)
	} else {
		logger.ConnServerLogger().Debugf("successfully disestablished UPnP port mapping")
	}

	s.wg.Done()
}

// initListeners initializes the configured net listeners and adds any bound
// addresses to the address manager. Returns the listeners and a NAT interface,
// which is non-nil if UPnP is in use.
func initListeners(amgr *addrmgr.AddrManager, listenAddrs []string, services wire.ServiceFlag) ([]net.Listener, NAT, error) {
	// Listen for TCP connections at the configured addresses
	netAddrs, err := ParseListeners(listenAddrs)
	if err != nil {
		return nil, nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := net.Listen(addr.Network(), addr.String())
		if err != nil {
			logger.ConnServerLogger().Warnf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	cfg := config.GlobalConfig()
	var nat NAT
	if len(cfg.ExternalIPs) != 0 {
		defaultPort, err := strconv.ParseUint(config.ActiveNetParams().DefaultPort, 10, 16)
		if err != nil {
			logger.ConnServerLogger().Errorf("Can not parse default port %s for active chain: %v",
				config.ActiveNetParams().DefaultPort, err)
			return nil, nil, err
		}

		for _, sip := range cfg.ExternalIPs {
			eport := uint16(defaultPort)
			host, portstr, err := net.SplitHostPort(sip)
			if err != nil {
				// no port, use default.
				host = sip
			} else {
				port, err := strconv.ParseUint(portstr, 10, 16)
				if err != nil {
					logger.ConnServerLogger().Warnf("Can not parse port from %s for "+
						"externalip: %v", sip, err)
					continue
				}
				eport = uint16(port)
			}
			na, err := amgr.HostToNetAddress(host, eport, services)
			if err != nil {
				logger.ConnServerLogger().Warnf("Not adding %s as externalip: %v", sip, err)
				continue
			}

			err = amgr.AddLocalAddress(na, addrmgr.ManualPrio)
			if err != nil {
				logger.AddrMgrLogger().Warnf("Skipping specified external IP: %v", err)
			}
		}
	} else {
		if cfg.Upnp {
			var err error
			nat, err = Discover()
			if err != nil {
				logger.ConnServerLogger().Warnf("Can't discover upnp: %v", err)
			}
			// nil nat here is fine, just means no upnp on network.
		}

		// Add bound addresses to address manager to be advertised to peers.
		for _, listener := range listeners {
			addr := listener.Addr().String()
			err := addLocalAddress(amgr, addr, services)
			if err != nil {
				logger.AddrMgrLogger().Warnf("Skipping bound address %s: %v", addr, err)
			}
		}
	}

	return listeners, nat, nil
}

// addLocalAddress adds an address that this node is listening on to the
// address manager so that it may be relayed to peers.
func addLocalAddress(addrMgr *addrmgr.AddrManager, addr string, services wire.ServiceFlag) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		// If bound to unspecified address, advertise all local interfaces
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			ifaceIP, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

			// If bound to 0.0.0.0, do not add IPv6 interfaces and if bound to
			// ::, do not add IPv4 interfaces.
			if (ip.To4() == nil) != (ifaceIP.To4() == nil) {
				continue
			}

			netAddr := wire.NewNetAddressIPPort(ifaceIP, uint16(port), services)
			err = addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
			if err != nil {
				logger.ConnServerLogger().Debugf("failed to add local address,err:%v", err)
			}
		}
	} else {
		netAddr, err := addrMgr.HostToNetAddress(host, uint16(port), services)
		if err != nil {
			return err
		}

		err = addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
		if err != nil {
			logger.ConnServerLogger().Debugf("failed to add local address,err:%v", err)
		}
	}

	return nil
}

// ParseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func ParseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// Shouldn't happen due to already being normalized.
			return nil, err
		}

		// Empty host or host of * on plan9 is both IPv4 and IPv6.
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

		// Strip IPv6 zone id if present since net.ParseIP does not
		// handle it.
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

		// Parse the IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

		// To4 returns nil when the IP is not an IPv4 address, so use
		// this determine the address type.
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

// addrStringToNetAddr takes an address in the form of 'host:port' and returns
// a net.Addr which maps to the original address with any host names resolved
// to IP addresses.  It also handles tor addresses properly by returning a
// net.Addr that encapsulates the address.
func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

	// Skip if host is already an IP address.
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

	// Tor addresses cannot be resolved to an IP, so just return an onion
	// address instead.
	cfg := config.GlobalConfig()
	if strings.HasSuffix(host, ".onion") {
		if cfg.NoOnion {
			return nil, errors.New("tor has been disabled")
		}

		return &onionAddr{addr: addr}, nil
	}

	// Attempt to look up an IP address associated with the parsed host.
	ips, err := btcdLookup(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}

// btcdDial connects to the address on the named network using the appropriate
// dial function depending on the address and configuration options.  For
// example, .onion addresses will be dialed using the onion specific proxy if
// one was specified, but will otherwise use the normal dial function (which
// could itself use a proxy or not).
func btcdDial(addr net.Addr) (net.Conn, error) {
	cfg := config.GlobalConfig()
	if strings.Contains(addr.String(), ".onion:") {
		return cfg.Oniondial(addr.Network(), addr.String(),
			config.DefaultConnectTimeout)
	}
	return cfg.Dial(addr.Network(), addr.String(), config.DefaultConnectTimeout)
}

// btcdLookup resolves the IP of the given host using the correct DNS lookup
// function depending on the configuration options.  For example, addresses will
// be resolved using tor when the --proxy flag was specified unless --noonion
// was also specified in which case the normal system DNS resolver will be used.
//
// Any attempt to resolve a tor address (.onion) will return an error since they
// are not intended to be resolved outside of the tor proxy.
func btcdLookup(host string) ([]net.IP, error) {
	if strings.HasSuffix(host, ".onion") {
		return nil, fmt.Errorf("attempt to resolve tor address %s", host)
	}

	return config.GlobalConfig().Lookup(host)
}

// QueryPeerShardCount returns the number of currently connected peers shards count.
func (s *ConnServer) QueryPeerShardCount() int32 {
	replyChan := make(chan int32)

	s.query <- GetPeerShardCountMsg{Reply: replyChan}

	return <-replyChan
}

// QueryShardPeersCount returns the number of currently connected peers shards count.
func (s *ConnServer) QueryShardPeersCount() []int32 {
	replyChan := make(chan []int32)

	s.query <- GetShardPeersCountMsg{Reply: replyChan}

	return <-replyChan
}

// QueryShardPeersAddr returns connected peers address belong to the shards.
func (s *ConnServer) QueryShardPeersAddr(shard []shard.Index) []string {
	replyChan := make(chan []string)

	s.query <- GetShardPeersAddrMsg{Reply: replyChan, Shard: shard}

	return <-replyChan
}

// ConnectedCount returns the number of currently connected peers.
func (s *ConnServer) ConnectedCount() int32 {
	replyChan := make(chan int32)

	s.query <- GetConnCountMsg{Reply: replyChan}

	return <-replyChan
}

// belongToRelayMsg 用于将需要广播的消息或需要处理 hash 的消息发送到 relayMailbox 中
func (s *ConnServer) belongToRelayMsg(from *ConnPeer, msg wire.Message) {
	switch msg := msg.(type) {
	case *wire.RelayHashMsg:
		s.handleHashMsg(from, msg.Relay, msg.Request)
	default:
		s.handleRelayMsg(from, msg)
	}
}

// handleRelayHashMsg 处理收到的 RelayHashMsg, 分别处理 relay hash 和 request hash.
func (s *ConnServer) handleHashMsg(cp *ConnPeer, relay, request []string) {
	go s.handleRequestHashMsg(cp, request)
	s.handleRelayHashMsg(cp, relay)
}

// handleRelayHashMsg 处理收到来自其他节点的的 relay hash, 判断 hash 所对应的消息是否已经收到、是否正在请求中: 若消息已经收到或
// 正在请求中，则忽略此 hash; 若消息未收到且未请求, 则向该节点发送该 hash 来请求消息.
func (s *ConnServer) handleRelayHashMsg(cp *ConnPeer, relay []string) {
	for i := range relay {
		if s.messagesSent.Contains(relay[i]) {
			continue
		}

		cpChan, ok := s.requestMsg.LoadOrStore(relay[i], make(chan *ConnPeer, config.GlobalConfig().MaxPeers))

		if ok {
			// 若消息正在请求中, 将此节点加入该消息的可用节点 chan 中.
			cpChan.(chan *ConnPeer) <- cp
		} else {
			cp.QueueMessage(&wire.RelayHashMsg{Request: []string{relay[i]}}, nil)
			time.AfterFunc(time.Second, s.handleRequest(cpChan.(chan *ConnPeer), relay[i], nil, []*ConnPeer{cp}))
		}
	}
}

// handleRequest 在发送请求 hash 对应的消息一定时间后以 goroutine 启动: 获取该消息可请求节点. 判断请求的消息是否收到,
// 若已收到消息或无可用待请求节点, 从 requestMsg 中移除该消息; 若不存在且有可请求节点, 向最新收到消息 hash 节点请求消息详情;
// 并在指定时间后启动 handleRequest. 每个可请求节点最多请求两次.
func (s *ConnServer) handleRequest(cpChan <-chan *ConnPeer, hash string, pendingCps, requestedCps []*ConnPeer) func() {
	return func() {
		// 获取该消息所有可用节点.
	out:
		for {
			select {
			case cp := <-cpChan:
				pendingCps = append(pendingCps, cp)
			default:
				break out
			}
		}
		var requestPeer *ConnPeer
		needToRequest := true
		// 首先向最新收到且未请求过的节点请求消息详情, 将待请求加入已请求过节点列表.
		if len(pendingCps) > 0 {
			requestPeer = pendingCps[len(pendingCps)-1]
			pendingCps = pendingCps[:len(pendingCps)-1]
			requestedCps = append(requestedCps, requestPeer)
		} else {
			// 若无未请求过的节点, 倒序从已请求过节点列表请求, 将所选节点从已请求过节点列表中移除.
			if len(requestedCps) > 0 {
				requestPeer = requestedCps[len(requestedCps)-1]
				requestedCps = requestedCps[:len(requestedCps)-1]
			} else {
				needToRequest = false
			}
		}

		// 发送请求前判断消息是否已经收到: 若收到, 则将此消息从待请求消息列表移除.
		if needToRequest && !s.messagesSent.Contains(hash) {
			requestPeer.QueueMessage(&wire.RelayHashMsg{Request: []string{hash}}, nil)
			//requestPeer.sendHashChan <- &msgHash{
			//	requestHash: []string{hash},
			//}
			time.AfterFunc(time.Second, s.handleRequest(cpChan, hash, pendingCps, requestedCps))
		} else {
			s.requestMsg.Delete(hash)
		}
	}
}

// handleRequestHashMsg 处理其他节点发来的 hash 请求消息, 判断 messagesSent 中是否存在该消息: 若存在, 向该节点发送完整的消息;
// 若不存在, 忽略该请求.
func (s *ConnServer) handleRequestHashMsg(cp *ConnPeer, keys []string) {
	for i := range keys {
		msg, ok := s.messagesSent.Get(keys[i])
		if ok {
			cp.QueueMessage(msg.(wire.Message), nil)
			continue
		}
		logger.ConnServerLogger().Debugf("can't find the requested message:%v", keys[i])

	}
}

// handleRelayMsg 处理收到完整的、需要广播的消息类型, 包括自身需要发送的消息与收到其他节点发来的消息, 判断 messagesSent 中是否
// 存在该消息: 若不存在, 记录该消息并将消息的 hash 发送到所有相连的节点; 若存在, 则忽略该消息.
// 判断如果是如下特殊的消息类型，进行传递优化：
// 1.如果是 blockconfirmation 类型的消息，使用 MsgBlockConfirmation.Header.BlockodyHash+分片号+分片高度进行hash
// 作为key，整个消息作为value存入map.
// 2.如果是 commit 类型消息，在发送之前进行过滤只发送优先级最高的(同一分片同一高度Block.Header.Seed最小的).
func (s *ConnServer) handleRelayMsg(cp *ConnPeer, msg wire.Message) {
	msgKey := getMsgHash(msg)
	if ok, _ := s.messagesSent.ContainsOrAdd(msgKey, msg); ok {
		return
	}
	logger.ConnServerLogger().Debugf("Receive new relay message:%s", msg.Command())
	if s.isStorageNode {
		return
	}

	shard := getMsgShard(msg)
	if msg.Command() == wire.CmdBinaryBAFin {
		shard = ToAllShard
	}
	s.broadcastMessageHash(msgKey, []*ConnPeer{cp}, shard)
}

func getMsgHash(msg wire.Message) string {
	// 对其他类型的消息的默认处理方式.
	hash := wire.MakeMessageHash(msg, 0, 0)
	return hex.EncodeToString(hash[:])
}

func getMsgShard(msg wire.Message) shard.Index {
	switch rmsg := msg.(type) {
	case wire.ShardMessage:
		return rmsg.GetShardIndex()
	}
	return ToAllShard

}

var sm sync.RWMutex

func (s *ConnServer) getStorageNodeAddr() []string {
	sm.RLock()
	defer sm.RUnlock()
	return s.storageNodeAddr
}

func (s *ConnServer) setStorageNodeAddr(addr string) {
	sm.Lock()
	defer sm.Unlock()
	s.storageNodeAddr = append(s.storageNodeAddr, addr)
}

// RegisterChannels is used to register handlers.
func (s *ConnServer) RegisterChannels(dispatch *MessagesAndReceiver) {
	s.handlers.RegisterChannels(dispatch)
}
