// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/base/go-socks/socks"
	"github.com/multivactech/MultiVAC/model/chaincfg"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

const (
	testRPCListenerAddr = "127.0.0.1:8334"
	testListenerAddr    = "127.0.0.1:8333"
)

// conn mocks a network connection by implementing the net.Conn interface.  It
// is used to test peer connection without actually opening a network
// connection.
type conn struct {
	io.Reader
	io.Writer
	io.Closer

	// local network, address for the connection.
	lnet, laddr string

	// remote network, address for the connection.
	rnet, raddr string

	// mocks socks proxy if true
	proxy bool
}

// LocalAddr returns the local address for the connection.
func (c conn) LocalAddr() net.Addr {
	return &addr{c.lnet, c.laddr}
}

// Remote returns the remote address for the connection.
func (c conn) RemoteAddr() net.Addr {
	if !c.proxy {
		return &addr{c.rnet, c.raddr}
	}
	host, strPort, _ := net.SplitHostPort(c.raddr)
	port, _ := strconv.Atoi(strPort)
	return &socks.ProxiedAddr{
		Net:  c.rnet,
		Host: host,
		Port: port,
	}
}

// Close handles closing the connection.
func (c conn) Close() error {
	if c.Closer == nil {
		return nil
	}
	return c.Closer.Close()
}

func (c conn) SetDeadline(t time.Time) error      { return nil }
func (c conn) SetReadDeadline(t time.Time) error  { return nil }
func (c conn) SetWriteDeadline(t time.Time) error { return nil }

// addr mocks a network address
type addr struct {
	net, address string
}

func (m addr) Network() string { return m.net }
func (m addr) String() string  { return m.address }

// pipe turns two mock connections into a full-duplex connection similar to
// net.Pipe to allow pipe's with (fake) addresses.
func pipe(c1, c2 *conn) (*conn, *conn) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	c1.Writer = w1
	c1.Closer = w1
	c2.Reader = r1
	c1.Reader = r2
	c2.Writer = w2
	c2.Closer = w2

	return c1, c2
}

func newUpdatePeerInfoMessage() wire.UpdatePeerInfoMessage {
	return wire.UpdatePeerInfoMessage{
		Shards:               shard.ShardList,
		IsStorageNode:        false,
		RPCListenerAddress:   testRPCListenerAddr,
		LocalListenerAddress: testListenerAddr,
	}
}

// peerStats holds the expected peer stats used for testing peer.
type peerStats struct {
	wantUserAgent       string
	wantServices        wire.ServiceFlag
	wantProtocolVersion uint32
	wantConnected       bool
	wantVersionKnown    bool
	wantVerAckReceived  bool
	wantLastBlock       int32
	wantStartingHeight  int32
	wantLastPingTime    time.Time
	wantLastPingNonce   uint64
	wantLastPingMicros  int64
	wantTimeOffset      int64
	wantBytesSent       uint64
	wantBytesReceived   uint64
	wantWitnessEnabled  bool
}

// testPeer tests the given peer's flags and stats
func testPeer(t *testing.T, p *Peer, s peerStats) {
	if p.UserAgent() != s.wantUserAgent {
		t.Errorf("testPeer: wrong UserAgent - got %v, want %v", p.UserAgent(), s.wantUserAgent)
		return
	}

	if p.Services() != s.wantServices {
		t.Errorf("testPeer: wrong Services - got %v, want %v", p.Services(), s.wantServices)
		return
	}

	if !p.LastPingTime().Equal(s.wantLastPingTime) {
		t.Errorf("testPeer: wrong LastPingTime - got %v, want %v", p.LastPingTime(), s.wantLastPingTime)
		return
	}

	if p.LastPingNonce() != s.wantLastPingNonce {
		t.Errorf("testPeer: wrong LastPingNonce - got %v, want %v", p.LastPingNonce(), s.wantLastPingNonce)
		return
	}

	if p.LastPingMicros() != s.wantLastPingMicros {
		t.Errorf("testPeer: wrong LastPingMicros - got %v, want %v", p.LastPingMicros(), s.wantLastPingMicros)
		return
	}

	if p.VerAckReceived() != s.wantVerAckReceived {
		t.Errorf("testPeer: wrong VerAckReceived - got %v, want %v", p.VerAckReceived(), s.wantVerAckReceived)
		return
	}

	if p.VersionKnown() != s.wantVersionKnown {
		t.Errorf("testPeer: wrong VersionKnown - got %v, want %v", p.VersionKnown(), s.wantVersionKnown)
		return
	}

	if p.ProtocolVersion() != s.wantProtocolVersion {
		t.Errorf("testPeer: wrong ProtocolVersion - got %v, want %v", p.ProtocolVersion(), s.wantProtocolVersion)
		return
	}

	if p.LastBlock() != s.wantLastBlock {
		t.Errorf("testPeer: wrong LastBlock - got %v, want %v", p.LastBlock(), s.wantLastBlock)
		return
	}

	// Allow for a deviation of 1s, as the second may tick when the message is
	// in transit and the protocol doesn't support any further precision.
	if p.TimeOffset() != s.wantTimeOffset && p.TimeOffset() != s.wantTimeOffset-1 {
		t.Errorf("testPeer: wrong TimeOffset - got %v, want %v or %v", p.TimeOffset(),
			s.wantTimeOffset, s.wantTimeOffset-1)
		return
	}

	//if p.BytesSent() != s.wantBytesSent {
	//	t.Errorf("testPeer: wrong BytesSent - got %v, want %v", p.BytesSent(), s.wantBytesSent)
	//	return
	//}
	//
	//if p.BytesReceived() != s.wantBytesReceived {
	//	t.Errorf("testPeer: wrong BytesReceived - got %v, want %v", p.BytesReceived(), s.wantBytesReceived)
	//	return
	//}

	if p.StartingHeight() != s.wantStartingHeight {
		t.Errorf("testPeer: wrong StartingHeight - got %v, want %v", p.StartingHeight(), s.wantStartingHeight)
		return
	}

	if p.Connected() != s.wantConnected {
		t.Errorf("testPeer: wrong Connected - got %v, want %v", p.Connected(), s.wantConnected)
		return
	}

	if p.IsWitnessEnabled() != s.wantWitnessEnabled {
		t.Errorf("testPeer: wrong WitnessEnabled - got %v, want %v",
			p.IsWitnessEnabled(), s.wantWitnessEnabled)
		return
	}

	stats := p.StatsSnapshot()

	if p.ID() != stats.ID {
		t.Errorf("testPeer: wrong ID - got %v, want %v", p.ID(), stats.ID)
		return
	}

	if p.Addr() != stats.Addr {
		t.Errorf("testPeer: wrong Addr - got %v, want %v", p.Addr(), stats.Addr)
		return
	}

	if p.LastSend() != stats.LastSend {
		t.Errorf("testPeer: wrong LastSend - got %v, want %v", p.LastSend(), stats.LastSend)
		return
	}

	if p.LastRecv() != stats.LastRecv {
		t.Errorf("testPeer: wrong LastRecv - got %v, want %v", p.LastRecv(), stats.LastRecv)
		return
	}
}

// TestPeerConnection tests connection between inbound and outbound peers.
func TestPeerConnection(t *testing.T) {
	verack := make(chan struct{})
	peer1Cfg := &Config{
		Listeners: MessageListeners{
			OnVerAck: func(p *Peer, msg *wire.MsgVerAck) {
				verack <- struct{}{}
			},
			OnWrite: func(bytesWritten int, msg wire.Message, err error) {
				if _, ok := msg.(*wire.MsgVerAck); ok {
					verack <- struct{}{}
				}
			},
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		ProtocolVersion:   wire.RejectVersion, // Configure with older version
		Services:          0,
	}
	peer2Cfg := &Config{
		Listeners:         peer1Cfg.Listeners,
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          wire.SFNodeNetwork | wire.SFNodeWitness,
	}

	wantStats1 := peerStats{
		wantUserAgent:       wire.DefaultUserAgent + "peer:1.0(comment)/",
		wantServices:        0,
		wantProtocolVersion: wire.RejectVersion,
		wantConnected:       true,
		wantVersionKnown:    true,
		wantVerAckReceived:  true,
		wantLastPingTime:    time.Time{},
		wantLastPingNonce:   uint64(0),
		wantLastPingMicros:  int64(0),
		wantTimeOffset:      int64(0),
		wantBytesSent:       167, // 143 version + 24 verack
		wantBytesReceived:   167,
		wantWitnessEnabled:  false,
	}
	wantStats2 := peerStats{
		wantUserAgent:       wire.DefaultUserAgent + "peer:1.0(comment)/",
		wantServices:        wire.SFNodeNetwork | wire.SFNodeWitness,
		wantProtocolVersion: wire.RejectVersion,
		wantConnected:       true,
		wantVersionKnown:    true,
		wantVerAckReceived:  true,
		wantLastPingTime:    time.Time{},
		wantLastPingNonce:   uint64(0),
		wantLastPingMicros:  int64(0),
		wantTimeOffset:      int64(0),
		wantBytesSent:       167, // 143 version + 24 verack
		wantBytesReceived:   167,
		wantWitnessEnabled:  true,
	}

	tests := []struct {
		name  string
		setup func() (*Peer, *Peer, error)
	}{
		{
			"basic handshake",
			func() (*Peer, *Peer, error) {
				inConn, outConn := pipe(
					&conn{raddr: "10.0.0.1:8333"},
					&conn{raddr: "10.0.0.2:8333"},
				)
				inPeer := NewInboundPeer(peer1Cfg)
				inPeer.AssociateConnection(inConn, newUpdatePeerInfoMessage())

				outPeer, err := NewOutboundPeer(peer2Cfg, "10.0.0.2:8333")
				if err != nil {
					return nil, nil, err
				}
				outPeer.AssociateConnection(outConn, newUpdatePeerInfoMessage())

				for i := 0; i < 4; i++ {
					select {
					case <-verack:
					case <-time.After(time.Second):
						return nil, nil, errors.New("verack timeout")
					}
				}
				return inPeer, outPeer, nil
			},
		},
		{
			"socks proxy",
			func() (*Peer, *Peer, error) {
				inConn, outConn := pipe(
					&conn{raddr: "10.0.0.1:8333", proxy: true},
					&conn{raddr: "10.0.0.2:8333"},
				)
				inPeer := NewInboundPeer(peer1Cfg)
				inPeer.AssociateConnection(inConn, newUpdatePeerInfoMessage())

				outPeer, err := NewOutboundPeer(peer2Cfg, "10.0.0.2:8333")
				if err != nil {
					return nil, nil, err
				}
				outPeer.AssociateConnection(outConn, newUpdatePeerInfoMessage())

				for i := 0; i < 4; i++ {
					select {
					case <-verack:
					case <-time.After(time.Second):
						return nil, nil, errors.New("verack timeout")
					}
				}
				return inPeer, outPeer, nil
			},
		},
	}
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		inPeer, outPeer, err := test.setup()
		if err != nil {
			t.Errorf("TestPeerConnection setup #%d: unexpected err %v", i, err)
			return
		}
		testPeer(t, inPeer, wantStats2)
		testPeer(t, outPeer, wantStats1)

		inPeer.Disconnect()
		outPeer.Disconnect()
		inPeer.WaitForDisconnect()
		outPeer.WaitForDisconnect()
	}
}

// TestPeerListeners tests that the peer listeners are called as expected.
func TestPeerListeners(t *testing.T) {
	verack := make(chan struct{}, 1)
	ok := make(chan wire.Message, 20)
	peerCfg := &Config{
		Listeners: MessageListeners{
			HandleNewMsg: func(p Reply, msg wire.Message, buf []byte) {
				switch msg.(type) {
				case *wire.MsgGetAddr, *wire.MsgAddr, *wire.MsgAlert, *wire.MsgTx, *wire.MsgBlock,
					*wire.MsgFeeFilter, *wire.MsgFilterAdd,
					*wire.MsgFilterClear, *wire.MsgFilterLoad, *wire.MsgReject:
					ok <- msg
				}
			},
			OnVersion: func(p *Peer, msg *wire.MsgVersion) {
				ok <- msg
			},
			OnVerAck: func(p *Peer, msg *wire.MsgVerAck) {
				verack <- struct{}{}
			},
			OnSendHeaders: func(p *Peer, msg *wire.MsgSendHeaders) {
				ok <- msg
			},
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          wire.SFNodeBloom,
	}
	inConn, outConn := pipe(
		&conn{raddr: "10.0.0.1:8333"},
		&conn{raddr: "10.0.0.2:8333"},
	)
	inPeer := NewInboundPeer(peerCfg)
	inPeer.AssociateConnection(inConn, newUpdatePeerInfoMessage())

	peerCfg.Listeners = MessageListeners{
		OnVerAck: func(p *Peer, msg *wire.MsgVerAck) {
			verack <- struct{}{}
		},
	}
	outPeer, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Errorf("NewOutboundPeer: unexpected err %v\n", err)
		return
	}
	outPeer.AssociateConnection(outConn, newUpdatePeerInfoMessage())

	for i := 0; i < 2; i++ {
		select {
		case <-verack:
		case <-time.After(time.Second * 1):
			t.Errorf("TestPeerListeners: verack timeout\n")
			return
		}
	}

	testShardID := shard.Index(0)

	tests := []struct {
		listener string
		msg      wire.Message
	}{
		{
			"OnGetAddr",
			wire.NewMsgGetAddr(),
		},
		{
			"OnAddr",
			wire.NewMsgAddr(),
		},
		{
			"OnAlert",
			wire.NewMsgAlert([]byte("payload"), []byte("signature")),
		},
		{
			"OnTx",
			wire.NewMsgTx(wire.TxVersion, 0),
		},
		{
			"OnBlock",
			wire.NewBlock(testShardID, 1,
				chainhash.Hash{}, merkle.MerkleHash{}, merkle.EmptyHash, []*wire.MsgTxWithProofs{{Tx: wire.MsgTx{}}}, &wire.LedgerInfo{}, false, time.Now().Unix(), nil, nil, nil, nil, nil),
		},
		{
			"OnFeeFilter",
			wire.NewMsgFeeFilter(15000),
		},
		{
			"OnFilterAdd",
			wire.NewMsgFilterAdd([]byte{0x01}),
		},
		{
			"OnFilterClear",
			wire.NewMsgFilterClear(),
		},
		{
			"OnFilterLoad",
			wire.NewMsgFilterLoad([]byte{0x01}, 10, 0, wire.BloomUpdateNone),
		},
		// only one version message is allowed
		// only one verack message is allowed
		{
			"OnReject",
			wire.NewMsgReject("block", wire.RejectDuplicate, "dupe block"),
		},
		{
			"OnSendHeaders",
			wire.NewMsgSendHeaders(),
		},
		// test Davis
		{
			"msgTx",
			wire.NewMsgTx(5454, 1),
		},
		{
			"msgReject",
			wire.NewMsgReject("Reject", 200, "Dirty data"),
		},
	}
	t.Logf("Running %d tests", len(tests))
	for _, test := range tests {
		// Queue the test message
		outPeer.QueueMessage(test.msg, nil)
		select {
		case <-ok:
		case <-time.After(time.Second * 1):
			t.Errorf("TestPeerListeners: %s timeout", test.listener)
			return
		}
	}
	inPeer.Disconnect()
	outPeer.Disconnect()
}

// TestOutboundPeer tests that the outbound peer works as expected.
func TestOutboundPeer(t *testing.T) {

	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	r, w := io.Pipe()
	c := &conn{raddr: "10.0.0.1:8333", Writer: w, Reader: r}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Errorf("NewOutboundPeer: unexpected err - %v\n", err)
		return
	}

	// Test trying to connect twice.
	p.AssociateConnection(c, newUpdatePeerInfoMessage())
	p.AssociateConnection(c, newUpdatePeerInfoMessage())

	disconnected := make(chan struct{})
	go func() {
		p.WaitForDisconnect()
		disconnected <- struct{}{}
	}()

	select {
	case <-disconnected:
		close(disconnected)
	case <-time.After(time.Second):
		t.Fatal("Peer did not automatically disconnect.")
	}

	if p.Connected() {
		t.Fatalf("Should not be connected as NewestBlock produces error.")
	}

	// Test Queue Inv
	fakeBlockHash := &chainhash.Hash{0: 0x00, 1: 0x01}
	fakeInv := wire.NewInvVect(wire.InvTypeBlock, fakeBlockHash)

	// Should be noops as the peer could not connect.
	p.AddKnownInventory(fakeInv)

	fakeMsg := wire.NewMsgVerAck()
	p.QueueMessage(fakeMsg, nil)
	done := make(chan struct{})
	p.QueueMessage(fakeMsg, done)
	<-done
	p.Disconnect()

	// Test NewestBlock
	var newestBlock = func() (*chainhash.Hash, int32, error) {
		hashStr := "14a0810ac680a3eb3f82edc878cea25ec41d6b790744e5daeef"
		hash, err := chainhash.NewHashFromStr(hashStr)
		if err != nil {
			return nil, 0, err
		}
		return hash, 234439, nil
	}

	peerCfg.NewestBlock = newestBlock
	r1, w1 := io.Pipe()
	c1 := &conn{raddr: "10.0.0.1:8333", Writer: w1, Reader: r1}
	p1, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Errorf("NewOutboundPeer: unexpected err - %v\n", err)
		return
	}
	p1.AssociateConnection(c1, newUpdatePeerInfoMessage())

	// Test update latest block
	latestBlockHash, err := chainhash.NewHashFromStr("1a63f9cdff1752e6375c8c76e543a71d239e1a2e5c6db1aa679")
	if err != nil {
		t.Errorf("NewHashFromStr: unexpected err %v\n", err)
		return
	}
	p1.UpdateLastAnnouncedBlock(latestBlockHash)
	p1.UpdateLastBlockHeight(234440)
	if p1.LastAnnouncedBlock() != latestBlockHash {
		t.Errorf("LastAnnouncedBlock: wrong block - got %v, want %v",
			p1.LastAnnouncedBlock(), latestBlockHash)
		return
	}

	// Test Queue Inv after connection
	p1.Disconnect()

	// Test regression
	peerCfg.ChainParams = &chaincfg.RegressionNetParams
	peerCfg.Services = wire.SFNodeBloom
	r2, w2 := io.Pipe()
	c2 := &conn{raddr: "10.0.0.1:8333", Writer: w2, Reader: r2}
	p2, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Errorf("NewOutboundPeer: unexpected err - %v\n", err)
		return
	}
	p2.AssociateConnection(c2, newUpdatePeerInfoMessage())

	// Test PushXXX
	var addrs []*wire.NetAddress
	for i := 0; i < 5; i++ {
		na := wire.NetAddress{}
		addrs = append(addrs, &na)
	}
	if _, err := p2.PushAddrMsg(addrs); err != nil {
		t.Errorf("PushAddrMsg: unexpected err %v\n", err)
		return
	}

	p2.PushRejectMsg("block", wire.RejectMalformed, "malformed", nil, false)
	p2.PushRejectMsg("block", wire.RejectInvalid, "invalid", nil, false)

	// Test Queue Messages
	p2.QueueMessage(wire.NewMsgGetAddr(), nil)
	p2.QueueMessage(wire.NewMsgPing(1), nil)
	p2.QueueMessage(wire.NewMsgFeeFilter(20000), nil)

	p2.Disconnect()

	// test messageSummary
	//p2.AssociateConnection(c2)
}

// Tests that the node disconnects from peers with an unsupported protocol
// version.
func TestUnsupportedVersionPeer(t *testing.T) {
	peerCfg := &Config{
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	localNA := wire.NewNetAddressIPPort(
		net.ParseIP("10.0.0.1"),
		uint16(8333),
		wire.SFNodeNetwork,
	)
	remoteNA := wire.NewNetAddressIPPort(
		net.ParseIP("10.0.0.2"),
		uint16(8333),
		wire.SFNodeNetwork,
	)
	localConn, remoteConn := pipe(
		&conn{laddr: "10.0.0.1:8333", raddr: "10.0.0.2:8333"},
		&conn{laddr: "10.0.0.2:8333", raddr: "10.0.0.1:8333"},
	)

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Fatalf("NewOutboundPeer: unexpected err - %v\n", err)
	}
	p.AssociateConnection(localConn, newUpdatePeerInfoMessage())

	// Read outbound messages to peer into a channel
	outboundMessages := make(chan wire.Message)
	go func() {
		for {
			_, msg, _, err := wire.ReadMessageN(
				remoteConn,
				p.ProtocolVersion(),
				peerCfg.ChainParams.Net,
			)
			if err == io.EOF {
				close(outboundMessages)
				return
			}
			if err != nil {
				t.Errorf("Error reading message from local node: %v\n", err)
				return
			}

			outboundMessages <- msg
		}
	}()

	// Read version message sent to remote peer
	select {
	case msg := <-outboundMessages:
		if _, ok := msg.(*wire.MsgVersion); !ok {
			t.Fatalf("Expected version message, got [%s]", msg.Command())
		}
	case <-time.After(time.Second):
		t.Fatal("Peer did not send version message")
	}

	// Remote peer writes version message advertising invalid protocol version 1
	invalidVersionMsg := wire.NewMsgVersion(remoteNA, localNA, 0, 0)
	invalidVersionMsg.ProtocolVersion = 1

	_, err = wire.WriteMessageN(
		remoteConn.Writer,
		invalidVersionMsg,
		uint32(invalidVersionMsg.ProtocolVersion),
		peerCfg.ChainParams.Net,
	)
	if err != nil {
		t.Fatalf("wire.WriteMessageN: unexpected err - %v\n", err)
	}

	// Expect peer to disconnect automatically
	disconnected := make(chan struct{})
	go func() {
		p.WaitForDisconnect()
		disconnected <- struct{}{}
	}()

	select {
	case <-disconnected:
		close(disconnected)
	case <-time.After(time.Second):
		t.Fatal("Peer did not automatically disconnect")
	}

	// Expect no further outbound messages from peer
	select {
	case msg, chanOpen := <-outboundMessages:
		if chanOpen {
			t.Fatalf("Expected no further messages, received [%s]", msg.Command())
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for remote reader to close")
	}
}

type TestRule struct {
	Type string
}

func (rule *TestRule) Filter(msg wire.Message) (wire.Message, error) {
	if msg.Command() != rule.Type {
		return nil, errors.New("Message's type not match")
	}
	testMessage := &wire.MsgFetchTxs{}
	return testMessage, nil
}

func TestRelyRules(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		t.Errorf("NewOutboundPeer: unexpected err - %v\n", err)
		return
	}

	testCommand := "initabcdata"
	// test rule
	rule := &TestRule{
		Type: testCommand,
	}
	err = p.RegisterRelayRules(rule.Type, rule)
	if err != nil {
		fmt.Println(err)
	}
	
	// test the message which in the rule
	testMessage := &wire.MsgFetchTxs{}
	m2 := &wire.MsgReturnInit{}
	r2, err := p.applyRelayRules(m2)
	if r2.Command() == m2.Command() || err != nil || r2.Command() != testMessage.Command() {
		fmt.Printf("wrong message type, expcet: %v, actual; %v", testMessage.Command(), r2.Command())
	}

	// test remove rule
	err = p.RemoveRelayRules(testCommand)
	if err != nil {
		fmt.Println(err)
	}
	r2, err = p.applyRelayRules(m2)
	if r2.Command() != m2.Command() || err != nil {
		t.Errorf("Wrong message type, expcet: %v, actual; %v", r2.Command(), m2.Command())
	}
}

func init() {
	// Allow self connection when running the tests.
	TstAllowSelfConns()
}
func TestPeer_NA(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	na := p.NA()
	fmt.Printf("IP:%v,\tport:%v\n", na.IP, na.Port)
}
func TestPeer_Inbound(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Println(p.Inbound())
}

func TestPeer_LocalAddr(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	_, outConn := pipe(
		&conn{raddr: "10.0.0.1:8333"},
		&conn{raddr: "10.0.0.2:8333"},
	)
	p.conn = outConn
	p.connected = 0
	connLocal := p.conn.LocalAddr()
	fmt.Println(connLocal)
	address := p.LocalAddr()
	fmt.Printf("%v\n", address)
	p.connected = 1
	address = p.LocalAddr()
	fmt.Printf("%v\n", address)
}
func TestPeer_TimeConnected(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	p.timeConnected = time.Now()
	fmt.Println(p.timeConnected)
}

func TestPeer_WantsHeaders(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
	}

	p, err := NewOutboundPeer(peerCfg, "10.0.0.1:8333")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Println(p.WantsHeaders())
	fmt.Println(p.String())
}
func TestNewOutboundPeer(t *testing.T) {
	peerCfg := &Config{
		NewestBlock: func() (*chainhash.Hash, int32, error) {
			return nil, 0, errors.New("newest block not found")
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &chaincfg.MainNetParams,
		Services:          0,
		//HostToNetAddress:  h2a,
	}
	ips := []string{
		"128.0.0.1:32332",
		"192.168.23.24:99999",
		"192.168.23.24,99999",
		"128.0.0.1:-1",
	}
	for _, val := range ips {
		_, err := NewOutboundPeer(peerCfg, val)
		if err != nil {
			fmt.Println(err)
		}
	}
	peerCfg.HostToNetAddress = func(host string, port uint16, services wire.ServiceFlag) (address *wire.NetAddress, e error) {
		return &wire.NetAddress{
			Timestamp: time.Now(),
			Services:  1,
			IP:        net.IP{192, 168, 1, 12},
			Port:      2333,
		}, nil
	}
	_, err := NewOutboundPeer(peerCfg, "198.189.198.198:9898")
	if err != nil {
		fmt.Println(err)
	}
}

//func h2a(host string, port uint16, services wire.ServiceFlag) (address *wire.NetAddress, e error) {
//	return &wire.NetAddress{
//		Timestamp: time.Now(),
//		Services:  1,
//		IP:        net.IP{192, 168, 1, 12},
//		Port:      2333,
//	}, nil
//}
