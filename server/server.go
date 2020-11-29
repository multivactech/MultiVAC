/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers

package server

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/controller"
	RpcServer "github.com/multivactech/MultiVAC/rpc/rpcserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

// Server provides a bitcoin server for handling communications to and from
// bitcoin peers.
type Server struct {
	started       int32
	shutdown      int32
	shutdownSched int32
	startupTime   int64

	peerSlice   map[string]bool
	chainParams *chaincfg.Params
	rpcServer   *RpcServer.RPCServer
	connServer  *connection.ConnServer

	newPeers  chan *connection.ConnPeer
	donePeers chan *connection.ConnPeer
	query     chan interface{}
	wg        sync.WaitGroup
	quit      chan struct{}

	// Processor for each shard.
	controller mtvService
	//shardProcessors map[shard.Index]*controller.ShardProcessor
	// A list of enabled shards of this server.
	enabledShards []shard.Index

	isStorageNode bool
}

type mtvService interface {
	// Start starts the service
	Start()
	// NotifyPeerAdd notifies all shards a particular peer is connected
	NotifyPeerAdd(cp *connection.ConnPeer)
	// NotifyPeerDone notifies all shards a particular peer is done (disconnected or not needed)
	NotifyPeerDone(cp *connection.ConnPeer)
}

type getOutboundGroup struct {
	key   string
	reply chan int
}

// OutboundGroupCount returns the number of peers connected to the given
// outbound group key.
func (s *Server) OutboundGroupCount(key string) int {
	replyChan := make(chan int)
	s.query <- getOutboundGroup{key: key, reply: replyChan}
	return <-replyChan
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.  It is safe for concurrent access.
func (s *Server) NetTotals() (uint64, uint64) {
	return s.connServer.NetTotals()
}

// Start begins accepting connections from peers.
func (s *Server) Start() {
	// Already started?
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	cfg := config.GlobalConfig()

	// Server startup time. Used for the uptime command for uptime calculation.
	s.startupTime = time.Now().Unix()
	// Start the connServer which in turn starts the address and block
	// managers.
	s.wg.Add(1)
	s.connServer.Start()

	if !cfg.DisableRPC {
		s.rpcServer.Start()
	}

	go s.peerHandler()

	if cfg.RunAsDNSSeed {
		return
	}
	if !config.GlobalConfig().DisableDNSSeed && !config.GlobalConfig().StorageNode {
		<-s.connServer.ConnectToEnoughPeers
	}
	logger.ServerLogger().Infof("Starting Server")
	s.controller.Start()
	// Start the CPU miner if generation is enabled.
	/*if cfg.Generate {
		s.cpuMiner.Start()
	}*/
}

// Stop gracefully shuts down the Server by stopping and disconnecting all
// peers and the main listener.
func (s *Server) Stop() error {
	// Make sure this only happens once.
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		logger.ServerLogger().Infof("Server is already in the process of shutting down")
		return nil
	}

	logger.ServerLogger().Warnf("Server shutting down")

	// Shutdown the RPC Server if it's not disabled.
	/*if !config.GlobalConfig().DisableRPC {
		s.RPCServer.Stop()
	}*/

	_ = s.connServer.Stop()

	// Signal the remaining goroutines to quit.
	close(s.quit)
	return nil
}

// WaitForShutdown blocks until the main listener and peer handlers are stopped.
func (s *Server) WaitForShutdown() {
	s.wg.Wait()
}

// ScheduleShutdown schedules a Server shutdown after the specified duration.
// It also dynamically adjusts how often to warn the Server is going down based
// on remaining duration.
func (s *Server) ScheduleShutdown(duration time.Duration) {
	// Don't schedule shutdown more than once.
	if atomic.AddInt32(&s.shutdownSched, 1) != 1 {
		return
	}
	logger.ServerLogger().Warnf("Server shutdown in %v", duration)
	go func() {
		remaining := duration
		tickDuration := dynamicTickDuration(remaining)
		done := time.After(remaining)
		ticker := time.NewTicker(tickDuration)
	out:
		for {
			select {
			case <-done:
				ticker.Stop()
				_ = s.Stop()
				break out
			case <-ticker.C:
				remaining = remaining - tickDuration
				if remaining < time.Second {
					continue
				}

				// Change tick duration dynamically based on remaining time.
				newDuration := dynamicTickDuration(remaining)
				if tickDuration != newDuration {
					tickDuration = newDuration
					ticker.Stop()
					ticker = time.NewTicker(tickDuration)
				}
				logger.ServerLogger().Warnf("Server shutdown in %v", remaining)
			}
		}
	}()
}

// NewServer returns a new btcd Server configured to listen on addr for the
// bitcoin network type specified by chainParams.  Use start to begin accepting
// connections from peers.
// monitorListenAddrs is added for temporary hardcoded metrics exporting.
// TODO: remove monitorListenAddrs when we use metric exporter instead.
func NewServer(listenAddrs []string, chainParams *chaincfg.Params, monitorListenAddrs []string) (*Server, error) {
	cfg := config.GlobalConfig()

	// TODO Temporary solution

	s := Server{
		chainParams:   chainParams,
		query:         make(chan interface{}),
		quit:          make(chan struct{}),
		newPeers:      make(chan *connection.ConnPeer),
		donePeers:     make(chan *connection.ConnPeer),
		peerSlice:     make(map[string]bool),
		isStorageNode: cfg.StorageNode,
	}
	var cerr error
	s.connServer, cerr = connection.NewConnServer(listenAddrs, chainParams, s.newPeers, s.donePeers, s.isStorageNode)
	if cerr != nil {
		return nil, cerr
	}
	logger.ServerLogger().Debugf("is it one of the first miners? %v", cfg.IsOneOfFirstMiners)
	rootCtrl := controller.NewRootController(cfg, s.connServer)
	s.controller = rootCtrl
	s.enabledShards = rootCtrl.GetEnabledShards()
	s.connServer.SetEnabledShards(s.enabledShards)

	if !cfg.DisableRPC {
		s.rpcServer, cerr = RpcServer.NewRPCServer(chainParams, s.connServer, rootCtrl)
		if cerr != nil {
			return nil, cerr
		}
	}

	monitorListens, _ := connection.ParseListeners(monitorListenAddrs)
	if len(monitorListens) != 0 {
		go pollConnectedPeerNum(&s)
		go handleMonitoringRequests(monitorListens[0])
	}

	return &s, nil
}

// pollConnectedPeerNum 轮询的去采集监控数据ConnectedPeerNum
func pollConnectedPeerNum(s *Server) {
	metric := metrics.Metric
	for {
		metric.ConnectedPeerNum.Set(float64(len(s.peerSlice)))
		time.Sleep(5 * time.Second)
	}
}

// handleMonitoringRequests 用于处理prometheus的http请求
func handleMonitoringRequests(exportAddr net.Addr) {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(exportAddr.String(), nil))
}

// dynamicTickDuration is a convenience function used to dynamically choose a
// tick duration based on remaining time.  It is primarily used during
// Server shutdown to make shutdown warnings more frequent as the shutdown time
// approaches.
func dynamicTickDuration(remaining time.Duration) time.Duration {
	switch {
	case remaining <= time.Second*5:
		return time.Second
	case remaining <= time.Second*15:
		return time.Second * 5
	case remaining <= time.Minute:
		return time.Second * 15
	case remaining <= time.Minute*5:
		return time.Minute
	case remaining <= time.Minute*15:
		return time.Minute * 5
	case remaining <= time.Hour:
		return time.Minute * 15
	}
	return time.Hour
}

func (s *Server) handleAddPeerMsg(cp *connection.ConnPeer) {

	s.peerSlice[cp.GetListenAddr()] = true
	s.controller.NotifyPeerAdd(cp)
}

func (s *Server) handleDonePeerMsg(cp *connection.ConnPeer) {
	delete(s.peerSlice, cp.GetListenAddr())
	s.controller.NotifyPeerDone(cp)
}

// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the Server, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
func (s *Server) peerHandler() {
	/** TODO(liang): enable it
	s.syncManager.Start()
	*/

	logger.ServerLogger().Tracef("Starting peer handler")
out:
	for {
		select {
		// New peers connected to the Server.
		case p := <-s.newPeers:
			s.handleAddPeerMsg(p)

			// Disconnected peers.
		case p := <-s.donePeers:
			s.handleDonePeerMsg(p)

		case qmsg := <-s.query:
			s.connServer.QueryChannel() <- qmsg

		case <-s.quit:
			break out
		}
	}

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-s.newPeers:
		case <-s.donePeers:
		case <-s.query:
		default:
			break cleanup
		}
	}
	s.wg.Done()
	logger.ServerLogger().Tracef("Peer handler done")
}
