// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers

package connection

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multivactech/MultiVAC/configs/params"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/addrmgr"
	"github.com/multivactech/MultiVAC/p2p/connmgr"
	"github.com/multivactech/MultiVAC/p2p/peer"
)

// ConnPeer extends the peer to maintain state shared by the Server.
type ConnPeer struct {
	// The following variables must only be used atomically
	CpFeeFilter int64

	*peer.Peer

	connReq            *connmgr.ConnReq
	connServer         *ConnServer
	persistent         bool
	relayMtx           sync.Mutex
	disableRelayTx     bool
	sentAddrs          bool
	isWhitelisted      bool
	shards             []shard.Index
	isStorageNode      bool
	isDNSSeed          bool
	listenerAddress    string
	rpcListenerAddress string
	//filter         *bloom.Filter
	knownAddresses map[string]struct{}
	banScore       connmgr.DynamicBanScore
	sendHashChan   chan *msgHash
	quit           chan struct{}
	// The following chans are used to sync blockmanager and Server.
	ServerPeerHandle func(p peer.Reply, msg wire.Message, buf []byte)
}

type msgHash struct {
	requestHash []string
	relayHash   []string
}

const (
	sendBufferSize    = 4
	sendBufferTimeOut = 200 * time.Millisecond
)

// newConnPeer returns a new ConnPeer instance. The peer needs to be set by
// the caller.
func newConnPeer(s *ConnServer, isPersistent bool) *ConnPeer {
	connPeer := &ConnPeer{
		connServer: s,
		persistent: isPersistent,
		//filter:         bloom.LoadFilter(nil),
		shards:         make([]shard.Index, 0, 1024),
		knownAddresses: make(map[string]struct{}),
		sendHashChan:   make(chan *msgHash, 1024),
		quit:           make(chan struct{}),
	}
	go connPeer.SendHash()
	return connPeer
}

// SendHash 当新的 connpeer 启动时会启动 SendHash 协程, 负责消息发送缓冲的作用: 当缓冲区消息数大于设定缓冲数或发送超时时，节点会向相应节点发送消息.
func (cp *ConnPeer) SendHash() {
	sendMsg := &wire.RelayHashMsg{}
out:
	for {
		select {
		case msg := <-cp.sendHashChan:
			sendMsg.Relay = append(sendMsg.Relay, msg.relayHash...)
			sendMsg.Request = append(sendMsg.Request, msg.requestHash...)
			if len(sendMsg.Relay)+len(sendMsg.Request) >= sendBufferSize {
				go cp.QueueMessage(sendMsg, nil)
				sendMsg = &wire.RelayHashMsg{}
			}
		case <-time.After(sendBufferTimeOut):
			if len(sendMsg.Request)+len(sendMsg.Relay) > 0 {
				go cp.QueueMessage(sendMsg, nil)
				sendMsg = &wire.RelayHashMsg{}
			}

		case <-cp.quit:
			break out
		}
	}

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-cp.sendHashChan:
		default:
			break cleanup
		}
	}
}

// addKnownAddresses adds the given addresses to the set of known addresses to
// the peer to prevent sending duplicate addresses.
func (cp *ConnPeer) addKnownAddresses(addresses []*wire.NetAddress) {
	for _, na := range addresses {
		cp.knownAddresses[addrmgr.NetAddressKey(na)] = struct{}{}
	}
}

// addressKnown true if the given address is already known to the peer.
func (cp *ConnPeer) addressKnown(na *wire.NetAddress) bool {
	_, exists := cp.knownAddresses[addrmgr.NetAddressKey(na)]
	return exists
}

// setDisableRelayTx toggles relaying of transactions for the given peer.
// It is safe for concurrent access.
func (cp *ConnPeer) setDisableRelayTx(disable bool) {
	cp.relayMtx.Lock()
	cp.disableRelayTx = disable
	cp.relayMtx.Unlock()
}

// relayTxDisabled returns whether or not relaying of transactions for the given
// peer is disabled.
// It is safe for concurrent access.
// todo(ymh):seems to unuse from golangci-lint.
//func (cp *ConnPeer) relayTxDisabled() bool {
//	cp.relayMtx.Lock()
//	isDisabled := cp.disableRelayTx
//	cp.relayMtx.Unlock()
//
//	return isDisabled
//}

// pushAddrMsg sends an addr message to the connected peer using the provided
// addresses.
func (cp *ConnPeer) pushAddrMsg(addresses []*wire.NetAddress) {
	// Filter addresses already known to the peer.
	addrs := make([]*wire.NetAddress, 0, len(addresses))
	for _, addr := range addresses {
		if !cp.addressKnown(addr) {
			addrs = append(addrs, addr)
		}
	}
	known, err := cp.PushAddrMsg(addrs)
	if err != nil {
		logger.PeerLogger().Errorf("Can't push address message to %s: %v", cp.Peer, err)
		cp.Disconnect()
		return
	}
	cp.addKnownAddresses(known)
}

// tryBroadcastMsg tries to broadcast the message.
// Returns true if succeed in queueing this message for broadcast, false otherwise.
func (cp *ConnPeer) tryBroadcastMsg(bmsg *broadcastMsg) bool {
	if !cp.Connected() {
		return false
	}

	if cp.isDNSSeed {
		return false
	}

	for _, ep := range bmsg.excludePeers {
		if cp == ep {
			return false
		}
	}

	// Only send msg to peer within named shard.
	if bmsg.shard < params.GenesisNumberOfShards {
		isTarget := false
		for _, shard := range cp.shards {
			if bmsg.shard == shard {
				isTarget = true
			}
		}
		if !isTarget {
			return false
		}
	}
	if bmsg.message != nil {
		cp.QueueMessage(bmsg.message, nil)
	} else {
		cp.sendHashChan <- &msgHash{
			relayHash: []string{bmsg.hash},
		}
	}
	return true
}

// addBanScore increases the persistent and decaying ban score fields by the
// values passed as parameters. If the resulting score exceeds half of the ban
// threshold, a warning is logged including the reason provided. Further, if
// the score is above the ban threshold, the peer will be banned and
// disconnected.
// todo(ymh):seems to unuse from golangci-lint.
//func (cp *ConnPeer) addBanScore(persistent, transient uint32, reason string) {
//	cfg := config.GlobalConfig()
//	// No warning is logged and no score is calculated if banning is disabled.
//	if cfg.DisableBanning {
//		return
//	}
//	if cp.isWhitelisted {
//		logger.PeerLogger().Debugf("Misbehaving whitelisted peer %s: %s", cp, reason)
//		return
//	}
//
//	warnThreshold := cfg.BanThreshold >> 1
//	if transient == 0 && persistent == 0 {
//		// The score is not being increased, but a warning message is still
//		// logged if the score is above the warn threshold.
//		score := cp.banScore.Int()
//		if score > warnThreshold {
//			logger.PeerLogger().Warnf("Misbehaving peer %s: %s -- ban score is %d, "+
//				"it was not increased this time", cp, reason, score)
//		}
//		return
//	}
//	score := cp.banScore.Increase(persistent, transient)
//	if score > warnThreshold {
//		logger.PeerLogger().Warnf("Misbehaving peer %s: %s -- ban score increased to %d",
//			cp, reason, score)
//		if score > cfg.BanThreshold {
//			logger.PeerLogger().Warnf("Misbehaving peer %s -- banning and disconnecting",
//				cp)
//			cp.connServer.BanPeer(cp)
//			cp.Disconnect()
//		}
//	}
//}

// OnVersion is invoked when a peer receives a version bitcoin message
// and is used to negotiate the protocol version details as well as kick start
// the communications.
func (cp *ConnPeer) OnVersion(_ *peer.Peer, msg *wire.MsgVersion) {
	// Add the remote peer time as a sample for creating an offset against
	// the local clock to keep the network time in sync.
	//mtvac cp.Server.timeSource.AddTimeSample(cp.Addr(), msg.Timestamp)

	// Choose whether or not to relay transactions before a filter command
	// is received.
	cp.setDisableRelayTx(msg.DisableRelayTx)

	// Update the address manager and request known addresses from the
	// remote peer for outbound connections.  This is skipped when running
	// on the simulation test network since it is only intended to connect
	// to specified peers and actively avoids advertising and connecting to
	// discovered peers.
	cfg := config.GlobalConfig()
	if !cfg.SimNet {
		addrManager := cp.connServer.addrManager

		// Outbound connections.
		if !cp.Inbound() {
			// After soft-fork activation, only make outbound
			// connection to peers if they flag that they're segwit
			// enabled.
			/**mtvac
			chain := cp.connServer.chain
			segwitActive, err := chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
			if err != nil {
				logger.PeerLogger().Errorf("Unable to query for segwit "+
					"soft-fork state: %v", err)
				return
			}

			if segwitActive && !cp.IsWitnessEnabled() {
				logger.PeerLogger().Infof("Disconnecting non-segwit "+
					"peer %v, isn't segwit enabled and "+
					"we need more segwit enabled peers", cp)
				cp.Disconnect()
				return
			}
			*/

			// TODO(davec): Only do this if not doing the initial block
			// download and the local address is routable.
			if !cfg.DisableListen /* && isCurrent? */ {
				// Get address that best matches.
				lna := addrManager.GetBestLocalAddress(cp.NA())
				if addrmgr.IsRoutable(lna) {
					// Filter addresses the peer already knows about.
					addresses := []*wire.NetAddress{lna}
					cp.pushAddrMsg(addresses)
				}
			}

			// Request known addresses if the Server address manager needs
			// more and the peer has a protocol version new enough to
			// include a timestamp with addresses.
			hasTimestamp := cp.ProtocolVersion() >=
				wire.NetAddressTimeVersion
			if addrManager.NeedMoreAddresses() && hasTimestamp {
				cp.QueueMessage(wire.NewMsgGetAddr(), nil)
			}

			// Mark the address as a known good address.
			addrManager.Good(cp.NA())
		}
	}

	// Add valid peer to the Server.
	cp.connServer.AddPeer(cp)
}

// HandleNewMsg handles the new incoming msg from other peers, dispatches it to appropriate logic or layer.
func (cp *ConnPeer) HandleNewMsg(pr peer.Reply, rmsg wire.Message, buf []byte) {
	switch msg := rmsg.(type) {
	case *wire.MsgGetAddr:
		cp.OnGetAddr(msg)
	case *wire.MsgAddr:
		cp.OnAddr(msg)
	case *wire.MsgGetShardAddr:
		addresses := cp.connServer.QueryShardPeersAddr(msg.ShardIndex)
		if len(addresses) > 0 {
			cp.QueueMessage(&wire.MsgReturnShardAddr{Address: addresses}, nil)
		}
	case *wire.MsgReturnShardAddr:
		go cp.connServer.connectToAddr(msg.Address)
	case *wire.MsgStartNet:
		atomic.CompareAndSwapInt32(&config.StartNet, 0, 1)
	case *wire.MsgAlert:
	// Note: The reference client currently bans peers that send alerts
	// not signed with its key.  We could verify against their key, but
	// since the reserserference client is currently unwilling to support
	// other implementations' alert messages, we will not relay theirs.
	case *wire.UpdatePeerInfoMessage:
		cp.OnUpdatePeerInfo(msg)
	default:
		// When a new ConnPeer instance has been created and The ServerPeer has not been
		// created yet, the ServerPeerHandle will be nil; the Upcoming messages cann't
		// be handled until ServerPeerHandle is not nil; It must be run as a goroutine.
		if config.GlobalConfig().EnableRelayMessage && wire.MsgShouldRelay(rmsg) {
			// apply the rely rules to message
			//rmsg, err := p.applyRelayRules(rmsg)
			//if err != nil {
			//	log.Debugf("Wrong to filter the rely message, %v", err)
			//}
			// if result is equals to nil, will not rely it
			go cp.connServer.belongToRelayMsg(cp, msg)
		}
		if msg.Command() != wire.CmdRelayHash {
			if cp.connServer.handlers.Handle(msg, pr) {
				logger.ConnServerLogger().Debugf("message %v already send to other group by channel", rmsg.Command())
			}
		}
	}
}

// enforceNodeBloomFlag disconnects the peer if the Server is not configured to
// allow bloom filters.  Additionally, if the peer has negotiated to a protocol
// version  that is high enough to observe the bloom filter service support bit,
// it will be banned since it is intentionally violating the protocol.
// todo(ymh):seems to unuse from golangci-lint.
//func (cp *ConnPeer) enforceNodeBloomFlag(cmd string) bool {
//	return true
//}

// OnGetAddr is invoked when a peer receives a getaddr bitcoin message
// and is used to provide the peer with known addresses from the address
// manager.
func (cp *ConnPeer) OnGetAddr(msg *wire.MsgGetAddr) {
	// Don't return any addresses when running on the simulation test
	// network.  This helps prevent the network from becoming another
	// public test network since it will not be able to learn about other
	// peers that have not specifically been provided.
	if config.GlobalConfig().SimNet {
		return
	}

	// Do not accept getaddr requests from outbound peers.  This reduces
	// fingerprinting attacks.
	if !cp.Inbound() {
		logger.PeerLogger().Debugf("Ignoring getaddr request from outbound peer ",
			"%v", cp)
		return
	}

	// Only allow one getaddr request per connection to discourage
	// address stamping of inv announcements.
	if cp.sentAddrs {
		logger.PeerLogger().Debugf("Ignoring repeated getaddr request from peer ",
			"%v", cp)
		return
	}
	cp.sentAddrs = true

	// Get the current known addresses from the address manager.
	addrCache := cp.connServer.addrManager.AddressCache()

	// Push the addresses.
	cp.pushAddrMsg(addrCache)
}

// OnAddr is invoked when a peer receives an addr bitcoin message and is
// used to notify the Server about advertised addresses.
func (cp *ConnPeer) OnAddr(msg *wire.MsgAddr) {
	// Ignore addresses when running on the simulation test network.  This
	// helps prevent the network from becoming another public test network
	// since it will not be able to learn about other peers that have not
	// specifically been provided.
	if config.GlobalConfig().SimNet {
		return
	}

	// Ignore old style addresses which don't include a timestamp.
	if cp.ProtocolVersion() < wire.NetAddressTimeVersion {
		return
	}

	// A message that has no addresses is invalid.
	if len(msg.AddrList) == 0 {
		logger.PeerLogger().Errorf("Command [%s] from %s does not contain any addresses",
			msg.Command(), cp.Peer)
		cp.Disconnect()
		return
	}

	for _, na := range msg.AddrList {
		// Don't add more address if we're disconnecting.
		if !cp.Connected() {
			return
		}

		// Set the timestamp to 5 days ago if it's more than 24 hours
		// in the future so this address is one of the first to be
		// removed when space is needed.
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

		// Add address to known addresses for this peer.
		cp.addKnownAddresses([]*wire.NetAddress{na})
	}

	// Add addresses to Server address manager.  The address manager handles
	// the details of things such as preventing duplicate addresses, max
	// addresses, and last seen updates.
	// XXX bitcoind gives a 2 hour time penalty here, do we want to do the
	// same?
	cp.connServer.addrManager.AddAddresses(msg.AddrList, cp.NA())
}

// OnUpdatePeerInfo update the connected peer info.
func (cp *ConnPeer) OnUpdatePeerInfo(msg *wire.UpdatePeerInfoMessage) {
	ip, _, _ := net.SplitHostPort(cp.Addr())
	_, portStr, _ := net.SplitHostPort(msg.LocalListenerAddress)
	cp.listenerAddress = fmt.Sprintf("%s:%s", ip, portStr)

	for _, shardIndex := range cp.shards {
		atomic.AddInt32(&cp.connServer.peerState.shardNumber[shardIndex.GetID()], -1)
	}

	for _, shardIndex := range msg.GetShards() {
		atomic.AddInt32(&cp.connServer.peerState.shardNumber[shardIndex.GetID()], 1)
	}

	cp.shards = msg.GetShards()
	cp.isStorageNode = msg.IsStorageNode
	cp.isDNSSeed = msg.IsDNS
	cp.rpcListenerAddress = msg.RPCListenerAddress

	cfg := config.GlobalConfig()
	if _, ok := cp.connServer.peerState.currentMinerNode.Load(cp.listenerAddress); ok {
		logger.ConnServerLogger().Debugf("peer has existed, update peer info")
	} else {
		if cp.isStorageNode || cp.isDNSSeed || cfg.StorageNode || cfg.RunAsDNSSeed || len(cp.shards) == 0 {
			cp.connServer.newConnPeers <- cp
		} else {
			connect := false
			for _, shardIndex := range cp.shards {
				if atomic.LoadInt32(&cp.connServer.peerState.shardNumber[shardIndex.GetID()]) <= params.GenesisNumberOfShards*int32(cfg.MinShardPeers) {
					connect = true
				}
			}
			if !connect {
				logger.ConnServerLogger().Debugf("enough peer, disconnect %v === %v", cp.shards, cp.listenerAddress)
				cp.Disconnect()
			} else {
				cp.connServer.newConnPeers <- cp
			}
		}
	}

	if !cfg.StorageNode && !cfg.RunAsDNSSeed && len(cp.connServer.storageNodeAddr) == 0 && len(msg.StorageNode) != 0 {
		for _, storageNode := range msg.StorageNode {
			netAddr, err := addrStringToNetAddr(storageNode)
			if err == nil {
				logger.ConnServerLogger().Debugf("connect to storagenode  %v", msg.StorageNode)
				go cp.connServer.connManager.Connect(&connmgr.ConnReq{
					Addr:      netAddr,
					Permanent: true,
				})
			}
		}
	}

	if config.GlobalConfig().RunAsDNSSeed {
		pna := cp.Peer.NA()
		port, _ := strconv.Atoi(portStr)
		na := wire.NewNetAddressTimestamp(pna.Timestamp, pna.Services, pna.IP, uint16(port))
		cp.connServer.addrManager.AddAddress(na, na)
	}
	if cfg.StorageNode && len(msg.StorageNode) > 0 {
		for _, storagenode := range msg.StorageNode {
			netAddr, err := addrStringToNetAddr(storagenode)
			if err != nil {
				logger.ConnServerLogger().Debugf("failed to convert address string to net address,err:%v", err)
				continue
			}
			logger.ConnServerLogger().Infof("connect to %v", storagenode)
			//cp.connServer.addrManager.AddAddress()
			go cp.connServer.connManager.Connect(&connmgr.ConnReq{
				Addr:      netAddr,
				Permanent: true,
			})
			pna := cp.Peer.NA()
			addr, err := cp.connServer.addrManager.DeserializeNetAddress(storagenode)
			if err != nil {
				logger.ConnServerLogger().Infof("failed to deserialize address,err:%v,address string;%v", err, storagenode)
				continue
			}
			logger.ConnServerLogger().Infof("address manager add  address:%v", addr)
			cp.connServer.addrManager.AddAddress(addr, pna)
		}
	}
}

// OnUpdateShardInfo just use for test.
// todo: remove, OnUpdateShardInfo just for test
func (cp *ConnPeer) OnUpdateShardInfo(msg *wire.UpdatePeerInfoMessage) {
	cp.shards = msg.Shards
}

// OnRead is invoked when a peer receives a message and it is used to update
// the bytes received by the Server.
func (cp *ConnPeer) OnRead(bytesRead int, msg wire.Message, err error) {
	cp.connServer.AddBytesReceived(uint64(bytesRead))
}

// OnWrite is invoked when a peer sends a message and it is used to update
// the bytes sent by the Server.
func (cp *ConnPeer) OnWrite(bytesWritten int, msg wire.Message, err error) {
	cp.connServer.AddBytesSent(uint64(bytesWritten))
}

// disconnectPeer attempts to drop the connection of a targeted peer in the
// passed peer list. Targets are identified via usage of the passed
// `compareFunc`, which should return `true` if the passed peer is the target
// peer. This function returns true on success and false if the peer is unable
// to be located. If the peer is found, and the passed callback: `whenFound'
// isn't nil, we call it with the peer as the argument before it is removed
// from the peerList, and is disconnected from the Server.
func disconnectPeer(ps *peerState, compareFunc func(*ConnPeer) bool) bool {
	found := false
	ps.forAllPeers(func(cp *ConnPeer) {
		peer := cp
		if compareFunc(peer) {
			peer.Disconnect()
		}
		found = true
	})
	return found
}

// newPeerConfig returns the configuration for the given ConnPeer.
func newPeerConfig(cp *ConnPeer) *peer.Config {
	cfg := config.GlobalConfig()
	return &peer.Config{
		Listeners: peer.MessageListeners{
			OnVersion:    cp.OnVersion,
			OnRead:       cp.OnRead,
			OnWrite:      cp.OnWrite,
			HandleNewMsg: cp.HandleNewMsg,
		},
		//NewestBlock:       sp.newestBlock,
		HostToNetAddress:   cp.connServer.addrManager.HostToNetAddress,
		Proxy:              cfg.Proxy,
		UserAgentName:      userAgentName,
		UserAgentVersion:   userAgentVersion,
		UserAgentComments:  cfg.UserAgentComments,
		ChainParams:        cp.connServer.chainParams,
		Services:           cp.connServer.services,
		DisableRelayTx:     cfg.BlocksOnly,
		ProtocolVersion:    peer.MaxProtocolVersion,
		EnableRelayMessage: cfg.EnableRelayMessage,
	}
}

// RPCListenerAddress returns the rpc listener address.
func (cp *ConnPeer) RPCListenerAddress() string {
	return cp.rpcListenerAddress
}

// IsStorageNode returns whether connpeer is storagenode.
func (cp *ConnPeer) IsStorageNode() bool {
	return cp.isStorageNode
}

// GetShards gets all shardIndex.
func (cp *ConnPeer) GetShards() []shard.Index {
	return cp.shards
}

// GetBanScore returns the current ban score.
func (cp *ConnPeer) GetBanScore() uint32 {
	return cp.banScore.Int()
}

// GetDisableRelayTx get the disable relay tx.
func (cp *ConnPeer) GetDisableRelayTx() bool {
	return cp.disableRelayTx
}

// GetListenAddr returns the listen address.
func (cp *ConnPeer) GetListenAddr() string {
	return cp.listenerAddress
}
