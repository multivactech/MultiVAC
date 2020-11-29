/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/jrick/logrotate/rotator"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	if logRotator != nil {
		logRotator.Write(p)
	}
	return len(p), nil
}

// System tags for loggers
const (
	RootControllerLoggerTag      = "ROOTCTRL"
	ShardprocessorLoggerTag      = "SHPR"
	AddrmgrLoggerTag             = "AMGR"
	ConnmgrLoggerTag             = "CMGR"
	PeerLoggerTag                = "PEER"
	ConsensusLoggerTag           = "CONS"
	BtcdLoggerTag                = "MultiVAC"
	RpcserverLoggerTag           = "RPCS"
	ServerLoggerTag              = "SRVR"
	ConnServerLoggerTag          = "CONNECTION"
	SyncLoggerTag                = "SYNC"
	StorageLoggerTag             = "STRG"
	BlockchainLoggerTag          = "CHAN"
	TxpoolLoggerTag              = "TXPL"
	AppblockchainLoggerTag       = "ABCL"
	WireLoggerTag                = "WIRE"
	StateLoggerTag               = "STATE"
	MinerLoggerTag               = "MINER"
	HeartbeatTag                 = "HEART"
	TxProcessorTag               = "TXPROCR"
	VirtualMachineTag            = "MVVM"
	DpoolLoggerTag               = "DPOOL"
	SmartContractDataStoreTag    = "SmCDS"
	BlockConfirmationProducerTag = "BCP"

	logFileSize   = 30 * 1024
	logFileNumber = 3
)

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// initLogRotator.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = btclog.NewBackend(logWriter{})

	// logRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	logRotator *rotator.Rotator

	rootCtrlLog      = backendLog.Logger(RootControllerLoggerTag)
	amgrLog          = backendLog.Logger(AddrmgrLoggerTag)
	cmgrLog          = backendLog.Logger(ConnmgrLoggerTag)
	btcdLog          = backendLog.Logger(BtcdLoggerTag)
	peerLog          = backendLog.Logger(PeerLoggerTag)
	rpcsLog          = backendLog.Logger(RpcserverLoggerTag)
	srvrLog          = backendLog.Logger(ServerLoggerTag)
	connLog          = backendLog.Logger(ConnServerLoggerTag)
	consLog          = backendLog.Logger(ConsensusLoggerTag)
	shprLog          = backendLog.Logger(ShardprocessorLoggerTag)
	syncLog          = backendLog.Logger(SyncLoggerTag)
	storageLogger    = backendLog.Logger(StorageLoggerTag)
	blockChainLogger = backendLog.Logger(BlockchainLoggerTag)
	txpoolLog        = backendLog.Logger(TxpoolLoggerTag)
	abcLog           = backendLog.Logger(AppblockchainLoggerTag)
	wireLog          = backendLog.Logger(WireLoggerTag)
	stateLog         = backendLog.Logger(StateLoggerTag)
	minerLog         = backendLog.Logger(MinerLoggerTag)
	heartLog         = backendLog.Logger(HeartbeatTag)
	txProcessorLog   = backendLog.Logger(TxProcessorTag)
	mvvmLog          = backendLog.Logger(VirtualMachineTag)
	depositPoolLog   = backendLog.Logger(DpoolLoggerTag)
	smartCDSLog      = backendLog.Logger(SmartContractDataStoreTag)
	bcpLog           = backendLog.Logger(BlockConfirmationProducerTag)
)

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]btclog.Logger{
	RootControllerLoggerTag:      rootCtrlLog,
	AddrmgrLoggerTag:             amgrLog,
	ConnmgrLoggerTag:             cmgrLog,
	PeerLoggerTag:                peerLog,
	ConsensusLoggerTag:           consLog,
	ShardprocessorLoggerTag:      shprLog,
	BtcdLoggerTag:                btcdLog,
	RpcserverLoggerTag:           rpcsLog,
	ServerLoggerTag:              srvrLog,
	ConnServerLoggerTag:          connLog,
	SyncLoggerTag:                syncLog,
	StorageLoggerTag:             storageLogger,
	BlockchainLoggerTag:          blockChainLogger,
	TxpoolLoggerTag:              txpoolLog,
	AppblockchainLoggerTag:       abcLog,
	WireLoggerTag:                wireLog,
	StateLoggerTag:               stateLog,
	MinerLoggerTag:               minerLog,
	HeartbeatTag:                 heartLog,
	TxProcessorTag:               txProcessorLog,
	VirtualMachineTag:            mvvmLog,
	DpoolLoggerTag:               depositPoolLog,
	SmartContractDataStoreTag:    smartCDSLog,
	BlockConfirmationProducerTag: bcpLog,
}

// InitLevel initialization those levels.
func InitLevel() {
	rootCtrlLog.SetLevel(RootCtrlLogLevel)
	amgrLog.SetLevel(AmgrLogLevel)
	cmgrLog.SetLevel(CmgrLogLevel)
	peerLog.SetLevel(PeerLogLevel)
	consLog.SetLevel(ConsLogLevel)
	shprLog.SetLevel(ShrpLogLevel)
	btcdLog.SetLevel(BtcdLogLevel)
	rpcsLog.SetLevel(RPCLogLevel)
	srvrLog.SetLevel(ServerLogLevel)
	connLog.SetLevel(ConnectionLogLevel)
	syncLog.SetLevel(SyncLogLevel)
	txpoolLog.SetLevel(TxpoolLogLevel)
}

// LogCleanup does the necessary cleaning before system shuts down.
func LogCleanup() {
	if logRotator != nil {
		logRotator.Close()
	}
}

// InitLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func InitLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, logFileSize, false, logFileNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}
	initPanicFile(logFile)
	logRotator = r
}

// SetLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func SetLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := btclog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// SetLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func SetLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		SetLogLevel(subsystemID, logLevel)
	}
}

// GetLogger returns the logger for relevant subsystem specified by sysId.
// This function returns the logger and a boolean indicating if the logger exists or not.
func GetLogger(sysID string) (btclog.Logger, bool) {
	logger, exists := subsystemLoggers[sysID]
	return logger, exists
}

// BackendLogger returns the btclog.Backend pointer.
func BackendLogger() *btclog.Backend {
	return backendLog
}

// BtcdLogger returns the logger for btcd subsystem.
func BtcdLogger() btclog.Logger {
	return btcdLog
}

// ServerLogger returns the logger for server subsystem.
func ServerLogger() btclog.Logger {
	return srvrLog
}

// ConnServerLogger returns the logger for connection server subsystem.
func ConnServerLogger() btclog.Logger {
	return connLog
}

// RPCSvrLogger returns the logger for rpc server subsystem.
func RPCSvrLogger() btclog.Logger {
	return rpcsLog
}

// AddrMgrLogger returns the logger for address manager subsystem.
func AddrMgrLogger() btclog.Logger {
	return amgrLog
}

// PeerLogger returns the logger for peer subsystem.
func PeerLogger() btclog.Logger {
	return peerLog
}

// TxpoolLogger returns the logger for txPool subsystem.
func TxpoolLogger() btclog.Logger {
	return txpoolLog
}

// AbcLogger returns the logger for AppBlockChain subsystem.
func AbcLogger() btclog.Logger {
	return abcLog
}

// WireLogger returns the logger for Wire subsystem.
func WireLogger() btclog.Logger {
	return wireLog
}

// StateLogger returns the logger for State subsystem.
func StateLogger() btclog.Logger {
	return stateLog
}

// SupportedSubsystems returns a sorted slice of the supported subsystems for
// logging purposes.
func SupportedSubsystems() []string {
	// Convert the subsystemLoggers map keys to a slice.
	subsystems := make([]string, 0, len(subsystemLoggers))
	for subsysID := range subsystemLoggers {
		subsystems = append(subsystems, subsysID)
	}

	// Sort the subsystems for stable display.
	sort.Strings(subsystems)
	return subsystems
}

func initPanicFile(globalFile string) error {
	//log.Println("init panic file in unix mode")
	file, err := os.OpenFile(globalFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		println(err)
		return err
	}
	if err = syscall.Dup2(int(file.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}
	return nil
}
