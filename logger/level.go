// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// This file used to set the log level of subsystem logger

package logger

import "github.com/multivactech/MultiVAC/logger/btclog"

const (
	// RootCtrlLogLevel -> RootController
	RootCtrlLogLevel = btclog.LevelInfo
	// AbciLogLevel -> abci
	AbciLogLevel = btclog.LevelInfo
	// StrgLogLevel -> storagenode
	StrgLogLevel = btclog.LevelInfo
	// SmLogLevel   -> shardManager
	SmLogLevel = btclog.LevelInfo
	// ConsLogLevel -> consensus
	ConsLogLevel = btclog.LevelInfo
	// BCPLogLevel -> blockconfirmproducer
	BCPLogLevel = btclog.LevelInfo
	// ShrpLogLevel -> shardProcessor
	ShrpLogLevel = btclog.LevelInfo
	// ChainLogLevel -> chain
	ChainLogLevel = btclog.LevelInfo
	// HeartLogLevel -> heartbeat
	HeartLogLevel = btclog.LevelInfo
	// DpoolLogLevel -> depositPool
	DpoolLogLevel = btclog.LevelInfo
	// PeerLogLevel -> peer
	PeerLogLevel = btclog.LevelInfo
	// ConnectionLogLevel -> connection
	ConnectionLogLevel = btclog.LevelInfo
	// AmgrLogLevel -> addressManager
	AmgrLogLevel = btclog.LevelInfo
	// CmgrLogLevel -> connectManager
	CmgrLogLevel = btclog.LevelInfo
	// RPCLogLevel -> rpc
	RPCLogLevel = btclog.LevelInfo
	// ServerLogLevel -> server
	ServerLogLevel = btclog.LevelInfo
	// BtcdLogLevel -> btcd
	BtcdLogLevel = btclog.LevelInfo
	// SyncLogLevel -> sync
	SyncLogLevel = btclog.LevelInfo
	// TxpoolLogLevel -> txpool
	TxpoolLogLevel = btclog.LevelInfo
)
