/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package heartbeat

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/pos/heartbeat/mocks"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func makeCorrectMessage(pk, sk []byte, time wire.Timestamp) *wire.HeartBeatMsg {
	var buf bytes.Buffer
	proof := []byte("test")
	VRF := &vrf.Ed25519VRF{}
	buf.Write(proof)
	buf.Write(util.Int64ToBytes(int64(time)))
	sign, _ := VRF.Generate(pk, sk, buf.Bytes())
	return &wire.HeartBeatMsg{
		Pk:        pk,
		Proof:     proof,
		TimeStamp: time,
		Signature: sign,
	}
}

func newTestHeartBeatManager(gossipNode connection.GossipNode) *ThreadHeartBeatManager {
	VRF := &vrf.Ed25519VRF{}
	pk, sk, _ := VRF.GenerateKey(nil)
	_, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	dp := new(mocks.DepositPool)
	dp.On("GetBiggest", multivacaddress.GenerateAddress(signature.PublicKey(pk),
		multivacaddress.UserAddress)).Return([]byte("test"), nil)
	dp.On("Verify", multivacaddress.GenerateAddress(signature.PublicKey(pk),
		multivacaddress.UserAddress), []byte("test")).Return(true)
	metric := &metrics.Metrics{
		OnlineMiner: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "multivac_minerNum",
				Help: "Number of miner",
			},
		),
	}
	thbm := &ThreadHeartBeatManager{
		actorCtx: message.NewActorContext(),
	}
	thbm.worker, err = newHeartBeatManager(pk, sk, dp, metric, gossipNode)
	if err != nil {
		panic(err)
	}
	thbm.actorCtx.StartActor(thbm.worker)
	return thbm
}

func TestHeartBeatReceive(t *testing.T) {
	// For this test we need miner's online time should >= 15s (params.ONLINE_TIME)
	// The acceptable deviation of time for heartbeat is 100s(for testing) (params.TIME_DEVIATION)
	manager := newTestHeartBeatManager(mocks.NewMockBroadCaster())
	VRF := &vrf.Ed25519VRF{}
	pkMsg, skMsg, _ := VRF.GenerateKey(nil)
	dp := manager.worker.dPool.(*mocks.DepositPool)
	dp.On("Verify", multivacaddress.GenerateAddress(signature.PublicKey(pkMsg),
		multivacaddress.UserAddress), []byte("test")).Return(true)
	testTime := wire.Timestamp(time.Now().Unix())
	msg := makeCorrectMessage(pkMsg, skMsg, testTime-36) // first -36 , last -36
	manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	msg = makeCorrectMessage(pkMsg, skMsg, testTime-15) // last time out, first -15 , last -15
	manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	assert.Equal(t, 0, manager.PerceivedCount())

	msg = makeCorrectMessage(pkMsg, skMsg, testTime-10) // first -15 , last -10
	manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	msg = makeCorrectMessage(pkMsg, skMsg, testTime) // first -15 , last 0 , last - first >= 15
	manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	//time.Sleep(time.Millisecond * 10)            // wait goroutine done
	assert.Equal(t, 1, manager.PerceivedCount()) // normal
	assert.True(t, manager.Has(pkMsg))
}

func TestHeartBeatSend(t *testing.T) {
	broadcaster := mocks.NewMockBroadCaster()
	params.SendHeartbeatIntervalS = 1 // send one heartbeat per sec
	manager := newTestHeartBeatManager(broadcaster)
	ticker := &mocks.FakeTicker{
		Stopped: false,
		C:       make(chan time.Time),
	}
	manager.worker.sendTicker = ticker
	manager.worker.startCh <- struct{}{}
	ticker.Tick()
	ticker.Tick()
	time.Sleep(time.Millisecond * 50)
	assert.Equal(t, 2, broadcaster.Count) // count means the number of beats that have been broadcast

	manager.worker.sendTicker.Stop()
	ticker.Tick()
	time.Sleep(time.Millisecond * 50)
	assert.Equal(t, 2, broadcaster.Count) // still 2, because send process is stopped
}

func TestBench(t *testing.T) {
	type pkSk struct {
		P vrf.PublicKey
		S vrf.PrivateKey
	}
	var a []pkSk
	manager := newTestHeartBeatManager(mocks.NewMockBroadCaster())
	testTime := wire.Timestamp(time.Now().Unix())
	var count = 500
	for i := 0; i < count; i++ {
		VRF := &vrf.Ed25519VRF{}
		pkMsg, skMsg, _ := VRF.GenerateKey(nil)
		a = append(a, pkSk{P: pkMsg, S: skMsg})
		dp := manager.worker.dPool.(*mocks.DepositPool)
		dp.On("Verify", multivacaddress.GenerateAddress(signature.PublicKey(pkMsg),
			multivacaddress.UserAddress), []byte("test")).Return(true)
	}
	begin := time.Now()
	for i := 0; i < count; i++ {
		msg := makeCorrectMessage(a[i].P, a[i].S, testTime-15)
		go manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	}
	for i := 0; i < count; i++ {
		msg := makeCorrectMessage(a[i].P, a[i].S, testTime-7)
		go manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	}
	for i := 0; i < count; i++ {
		msg := makeCorrectMessage(a[i].P, a[i].S, testTime)
		go manager.actorCtx.Send(manager.worker, message.NewEvent(evtReceiveReq, msg), nil)
	}
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, count, manager.PerceivedCount())
	used := time.Since(begin)
	fmt.Printf("time used %v\n", used)
}
