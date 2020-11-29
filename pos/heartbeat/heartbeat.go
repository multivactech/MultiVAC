/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package heartbeat

import (
	"bytes"
	"encoding/hex"
	"os"
	"sync"
	"time"

	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	minerNodeThreshold = 0.6667
	initialListCap     = 10000
	startStatus        = iota
	stopStatus
)

var maxNoDeposit = 200

type ticker interface {
	Stop()
	Chan() <-chan time.Time
}

type realTicker struct{ *time.Ticker }

func (rt *realTicker) Chan() <-chan time.Time {
	return rt.C
}

// recorder is used to manage the online time of miners whose online time are not enough.
type recorder struct {
	// firstReceiveTime used to record the timestamp when the node
	// receives a heartbeat message that has never receives, node
	// needs track it until it meets on-line time condition.
	firstReceiveTime wire.Timestamp

	// lastReceiveTime used to record the last timestamp of a heartbeat
	// message that the node has been received but not meets on-line rules.
	lastReceiveTime wire.Timestamp
}

// Manager is used to manage nodes to send and receive heartbeat messages.
type Manager struct {
	// whiteList is the list of all konwn and legal miners.
	whiteList map[string]wire.Timestamp

	// cacheList is used to manage the heartbeat that online time in it are not enough.
	cacheList map[string]*recorder

	// validator is used to check if the miner is legal.
	dPool idepositpool.DepositPool

	// register message handle channels and send broadcast messages
	gossipNode connection.GossipNode
	startCh    chan struct{}
	quitCh     chan struct{}

	vrf vrf.VRF
	pk  []byte
	sk  []byte

	status     int
	sendTicker ticker
	logger     btclog.Logger
	metrics    *metrics.Metrics

	// TODO: testnet-3.0
	maxNoDeposit int

	// TODO: temporary fix for testnet-3.0
	mu sync.RWMutex
}

// newHeartBeatManager returns the instance of HearBeatManager.
func newHeartBeatManager(pk, sk []byte, dPool idepositpool.DepositPool, metrics *metrics.Metrics, gossipNode connection.GossipNode) (*Manager, error) {
	manager := &Manager{
		whiteList:  make(map[string]wire.Timestamp, initialListCap),
		cacheList:  make(map[string]*recorder, initialListCap),
		vrf:        &vrf.Ed25519VRF{},
		gossipNode: gossipNode,
		status:     stopStatus,
		pk:         pk,
		sk:         sk,
		startCh:    make(chan struct{}),
		quitCh:     make(chan struct{}),
		metrics:    metrics,
	}
	manager.dPool = dPool
	manager.configLogger()

	go manager.loop()
	return manager, nil
}

func (manager *Manager) loop() {
	for range manager.startCh {
		manager.send()
	}
}

func (manager *Manager) start() {
	if manager.status != stopStatus {
		return
	}
	manager.sendTicker = &realTicker{time.NewTicker(time.Second * time.Duration(params.SendHeartbeatIntervalS))}
	manager.startCh <- struct{}{}
}

func (manager *Manager) stop() {
	if manager.status != startStatus {
		return
	}
	manager.sendTicker.Stop()
	manager.quitCh <- struct{}{}
}

// whitelistSize returns the size of whitelist, it means the number of online miner.
func (manager *Manager) whitelistSize() int {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	manager.logger.Debugf("Current count of online miner is %d", len(manager.whiteList))
	return len(manager.whiteList)
}

// IsInList returns if the given pk is in the whitelist, this must meet certain rules.
func (manager *Manager) isInList(pk []byte) bool {
	pkKey := hex.EncodeToString(pk)
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	_, ok := manager.whiteList[pkKey]
	return ok
}

// send broadcasts heartbaet message at regular intervals.
func (manager *Manager) send() {
	if config.GlobalConfig().IsOneOfFirstMiners {
		maxNoDeposit = 200
	}
	manager.status = startStatus
	removeFlag := time.Now()
	// Send heartbeat at intervals
	for {
		select {
		case <-manager.sendTicker.Chan():
			now := time.Now().Unix()
			manager.logger.Infof("Send heartbeat, timestamp is %d", now)
			// Add myself to the whitelist.
			pkKey := hex.EncodeToString(manager.pk)
			manager.mu.Lock()
			manager.whiteList[pkKey] = wire.Timestamp(now)
			manager.mu.Unlock()
			// Generate my depositpool proof.
			// TODO: maybe wrong
			address := multivacaddress.GenerateAddress(signature.PublicKey(manager.pk), multivacaddress.UserAddress)
			proof, err := manager.dPool.GetBiggest(address)
			if err != nil {
				manager.logger.Debugf("Failed to get proof of deposit: %v", err)
				manager.maxNoDeposit++
				if manager.maxNoDeposit > maxNoDeposit {
					manager.logger.Errorf("Please deposit MTV before joining the network")
					os.Exit(1)
				}
				continue
			}
			// Sign the proof and timestamp.
			sign, err := manager.signMessage(proof, now)
			if err != nil {
				manager.logger.Errorf("Failed to sign message: %v", err)
				continue
			}
			heartbeat := &wire.HeartBeatMsg{
				Pk:        manager.pk,
				Proof:     proof,
				TimeStamp: wire.Timestamp(now),
				Signature: sign,
			}
			if ok, err := manager.verifyMessage(heartbeat); !ok {
				panic(err)
			}

			manager.gossipNode.BroadcastMessage(heartbeat, &connection.BroadcastParams{
				ToNode:      connection.ToAllNode,
				ToShard:     connection.ToAllShard,
				IncludeSelf: true,
			})

			// Try to remove expire heartbeat.
			if time.Since(removeFlag)*time.Second > time.Second*time.Duration(params.OnlineTimeS) {
				manager.removeExpired()
				removeFlag = time.Now()
			}
			manager.logger.Debugf("Send heartbeat, timestamp is %d", now)
			manager.metrics.OnlineMiner.Set(float64(manager.whitelistSize()))
		case <-manager.quitCh:
			manager.logger.Debugf("Quit send")
			manager.status = stopStatus
			return
		}
	}
}

// handleHeartBeat receives heartbeat message from other nodtrues and maintain a whitelist for the long-term online miners.
func (manager *Manager) handleHeartBeat(msg *wire.HeartBeatMsg) {
	manager.logger.Debugf("Receive a heartbeat Pk is %v, current miner is %d", msg.Pk, manager.whitelistSize())
	// Verify if the miner is legal.
	address := multivacaddress.GenerateAddress(signature.PublicKey(msg.Pk), multivacaddress.UserAddress)
	if !manager.dPool.Verify(address, msg.Proof) {
		manager.logger.Debugf("Illegal proof, pk: %v, proof: %v", msg.Pk, msg.Proof)
		return
	}
	// Verify the signature of timestamp and proof.
	ok, err := manager.verifyMessage(msg)
	if !ok {
		manager.logger.Errorf("Signature verification failed, %v", err)
		return
	}

	// Verify the timestamp is match the rule
	// The timestamp of the heartbeat should be near local time
	now := wire.Timestamp(time.Now().Unix())
	if !isTimestampValid(now, msg.TimeStamp) {
		manager.logger.Debugf("Receive a invalid heartbeat, timestamp is not match the rule, deviation is %ds, actual is %ds", params.OnlineTimeS, now-msg.TimeStamp)
		return
	}

	pkKey := hex.EncodeToString(msg.Pk)
	manager.mu.Lock()
	defer manager.mu.Unlock()
	// If the hearbeat is in whitelist, update it's timestamp
	if _, ok := manager.whiteList[pkKey]; ok {
		manager.whiteList[pkKey] = msg.TimeStamp
		manager.logger.Debugf("Receive a heartbeat message that has been received, (%d)", len(manager.whiteList))
		// If not in whitelist, means the hearbeat is a new one
	} else if v, ok := manager.cacheList[pkKey]; ok {
		// For all the new heartbeats we receive, we have to make sure they
		// meet certain online time before putting them in the whitelist.
		if isLastHeartBeatExpired(msg.TimeStamp, v.lastReceiveTime) {
			manager.cacheList[pkKey] = &recorder{
				firstReceiveTime: msg.TimeStamp,
				lastReceiveTime:  msg.TimeStamp,
			}
			manager.logger.Debugf("Received a heartbeat, but it’s been a long time since received it last time, time interval is %ds",
				msg.TimeStamp-v.lastReceiveTime)
		} else {
			// The new heartbeat's online time is enough
			firstReceiveTime := int64(msg.TimeStamp - manager.cacheList[pkKey].firstReceiveTime)
			if firstReceiveTime >= params.OnlineTimeS {
				manager.logger.Debugf("Received a heartbeat, enough online time, online time is %ds", msg.TimeStamp-manager.cacheList[pkKey].firstReceiveTime)
				delete(manager.cacheList, pkKey)
				manager.whiteList[pkKey] = msg.TimeStamp
			} else {
				manager.logger.Debugf("Received a heartbeat, but not enough online time, it’s online time is %ds", msg.TimeStamp-manager.cacheList[pkKey].firstReceiveTime)
				manager.cacheList[pkKey].lastReceiveTime = msg.TimeStamp
			}
		}
	} else {
		// For all heartbeats that have never been received
		record := &recorder{
			lastReceiveTime:  msg.TimeStamp,
			firstReceiveTime: msg.TimeStamp,
		}
		manager.cacheList[pkKey] = record
		manager.logger.Debugf("Received a heartbeat that has never received before")
	}
}

// removeExpired will remove the miners who are not online for a long time.
func (manager *Manager) removeExpired() {
	manager.logger.Debug("Start to remove expire heartbeat")
	now := wire.Timestamp(time.Now().Unix())
	manager.mu.Lock()
	defer manager.mu.Unlock()
	for k, v := range manager.whiteList {
		// Ignore myself
		if hex.EncodeToString(manager.pk) == k {
			continue
		}
		if isLastHeartBeatExpired(now, v) {
			manager.logger.Debugf("Remove expire heartbeat from whitelist, pk: %v, time interval is %d", k, now-v)
			delete(manager.whiteList, k)
		}
	}
}

// isTimeValid verifies the receive heartbeat's timestamp, make sure it is a legal timestamp.
func isTimestampValid(now, t wire.Timestamp) bool {
	deviation := t - now
	if int64(deviation) > params.TimeDeviationS || int64(deviation) < -params.TimeDeviationS {
		return false
	}
	return true
}

// isSatisfyOnlineTime determines if a heartbeat satisfy the online time requirement.
func isLastHeartBeatExpired(now, lastHeartBeatTs wire.Timestamp) bool {
	return int64(now-lastHeartBeatTs) > params.OnlineTimeS
}

func (manager *Manager) configLogger() {
	// Set logger
	manager.logger = logBackend.Logger("HEART")
	manager.logger.SetLevel(logger.HeartLogLevel)
}

func (manager *Manager) signMessage(proof []byte, now int64) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	_, err = buf.Write(proof)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(util.Int64ToBytes(now))
	if err != nil {
		return nil, err
	}

	sign, err := manager.vrf.Generate(manager.pk, manager.sk, buf.Bytes())
	if err != nil {
		manager.logger.Errorf("Faild to sign message: %v", err)
		return nil, err
	}
	return sign, nil
}

func (manager *Manager) verifyMessage(msg *wire.HeartBeatMsg) (bool, error) {
	var buf bytes.Buffer
	var err error
	_, err = buf.Write(msg.Proof)
	if err != nil {
		return false, err
	}
	_, err = buf.Write(util.Int64ToBytes(int64(msg.TimeStamp)))
	if err != nil {
		return false, err
	}

	ok, err := manager.vrf.VerifyProof(msg.Pk, msg.Signature, buf.Bytes())
	return ok, err
}

func (manager *Manager) checkConfirmation(msg *wire.MsgBlockConfirmation) bool {
	minConfirmationCount := float64(manager.whitelistSize()) * params.SecurityLevel * minerNodeThreshold
	if float64(len(msg.Confirms)) < minConfirmationCount {
		return false
	}
	count := 0
	for _, msgBinaryBA := range msg.Confirms {
		pk := msgBinaryBA.SignedCredentialWithBA.Pk
		if manager.isInList(pk) {
			count++
		}
	}
	return float64(count) >= minConfirmationCount
}

// Act implements the actor interface.
func (manager *Manager) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtReceiveReq:
		msg := e.Extra.(*wire.HeartBeatMsg)
		manager.handleHeartBeat(msg)
	case evtPerceivedCountReq:
		callback(manager.whitelistSize())
	case evtHasReq:
		pk := e.Extra.([]byte)
		callback(manager.isInList(pk))
	case evtCheckConfirmation:
		msg := e.Extra.(*wire.MsgBlockConfirmation)
		callback(manager.checkConfirmation(msg))
	case evtStartReq:
		manager.start()
	case evtStopReq:
		manager.stop()
	default:
		manager.logger.Errorf("Receive a wrong topic %v", e.Topic)
	}
}
