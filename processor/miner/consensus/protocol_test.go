/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/miner/consensus/mock"
)

// sub-function to test fully connected network with given rounds, peers.
func testFullyConnected(t *testing.T, stopRound, numPeers int) {
	config.SetDefaultConfig()
	// generate peers
	var ces []*executor
	var queue []chan *connection.MessageAndReply
	var bcProducers []*mock.BcProducer
	id2pk := make(map[int]string)

	v := &vrf.Ed25519VRF{}

	output := make([][]*SimpleBlockInfo, numPeers)
	for i := 0; i < numPeers; i++ {
		output[i] = make([]*SimpleBlockInfo, stopRound+1)
	}

	for i := 0; i < numPeers; i++ {
		testPk, testSk, err := v.GenerateKey(nil)
		if err != nil {
			t.Fatalf("Failed to generate pk/sk pair, err: %v", err)
		}
		// TODO(liang): fix the unstability when leader rate is 4
		ce := newConsenesusExecutorForTest(testPk, testSk, 1, 1, numPeers)
		id2pk[i] = hex.EncodeToString(testPk)
		fmt.Printf("PeerId: %v, PublicKey: %v\n", i, id2pk[i])
		ce.logger = btclog.NewBackend(os.Stdout).Logger("TEST_" + strconv.Itoa(i))
		ce.logger.SetLevel(btclog.LevelError)
		ce.timer = nil
		ce.bbaVoter.logger = ce.logger
		ces = append(ces, ce)
		queue = append(queue, ce.msgQueue)
		ce.state.syncStatus = upToDate
		ce.Init()
		ce.state.updateShardStopRound(stopRound)
		bcProducers = append(bcProducers, mock.NewMockBcProducer(testShard))
	}

	// build simulated broadcast network
	var wg sync.WaitGroup

	var broadcastLen = make([]int32, numPeers)
	var broadcastMsgMap = make([]map[string]*int32, numPeers)
	wg.Add(numPeers)
	for i := 0; i < numPeers; i++ {
		go func(i int) {
			time.Sleep(time.Second)
			round := 0
			broadcastMsgMap[i] = make(map[string]*int32)
			initMap(broadcastMsgMap[i])

			for {
				select {
				case msg := <-ces[i].gossipNode.(fakeGossipNode):
					var msgLen int
					var bw bytes.Buffer
					if msg.BtcEncode(&bw, 0, 0) == nil {
						msgLen = bw.Len()
					}
					fmt.Printf("peer %v trying to broadcast %s: %v, msg lenth: %v\n", i, msg.Command(), msg, msgLen)
					atomic.AddInt32(&broadcastLen[i], int32(msgLen))
					atomic.AddInt32(broadcastMsgMap[i][msg.Command()], 1)

					switch m := msg.(type) {
					case *wire.BlockHeader:
						bcProducers[i].SaveHeader(m)
					case *wire.MsgBinaryBAFin:
						bc := bcProducers[i].GenerateBlockConfirmation(m)
						queue[i] <- &connection.MessageAndReply{Msg: bc}
					default:
						for j := 0; j < numPeers; j++ {
							queue[j] <- &connection.MessageAndReply{Msg: msg}
						}
					}
				case blk := <-ces[i].blockReporter:
					fmt.Printf("new block generated for peer %v, round %v, block: %v\n", i, blk.round, blk)
					output[i][round] = blk
					round++
					if blk.round == int32(stopRound-1) {
						wg.Done()
						return
					}
				}
			}
		}(i)
	}
	for i := 0; i < numPeers; i++ {
		ces[i].Start()
	}
	wg.Wait()

	// check output blocks
	for i := 0; i < numPeers; i++ {
		t.Logf("output block for node [%v]:", i)
		for r := 0; r < stopRound; r++ {
			t.Logf("+++ round[%v]: %v", r, output[i][r])
		}
	}
	for i := 1; i < numPeers; i++ {
		for r := 0; r < stopRound; r++ {
			if output[i][r].leader != output[0][r].leader || output[i][r].curHash != output[0][r].curHash {
				t.Errorf("Output blocks not totally identical")
			}
		}
	}

	for i := 0; i < numPeers; i++ {
		t.Logf("============")
		t.Logf("peer number %v", i)
		t.Logf("Total broadcasted length %v", broadcastLen[i])
		for k, v := range broadcastMsgMap[i] {
			t.Logf("broadcasted %v number %v", k, *v)

		}
	}

	// the first byte of fake block hash of peer i is i+100
	/*
		for r := 1; r <= stopRound; r++ {
			leaderId := output[0][r].curHash[0] - 100
			if id2pk[int(leaderId)] != output[0][r].leader {
				t.Errorf("Output block is incorrect")
			}
		}
	*/
}

func TestConsensusProtocolFullConnected(t *testing.T) {
	testFullyConnected(t, 5, 5)
}

// helper
func initMap(mp map[string]*int32) {
	mp[wire.CmdLeaderProposal] = new(int32)
	mp[wire.CmdMsgBlockConfirmation] = new(int32)
	mp[wire.CmdSeed] = new(int32)
	mp[wire.CmdGC] = new(int32)
	mp[wire.CmdBinaryBA] = new(int32)
	mp[wire.CmdLeaderVote] = new(int32)
	mp[wire.CmdBlockHeader] = new(int32)
	mp[wire.CmdBinaryBAFin] = new(int32)
}
