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
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/miner/consensus/mock"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/stretchr/testify/assert"
)

var (
	testLeaderPublicKeyA, _  = hex.DecodeString("885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
	testLeaderPrivateKeyA, _ = hex.DecodeString("1fcce948db9fc312902d49745249cfd287de1a764fd48afb3cd0bdd0a8d74674885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
	testLeaderPublicKeyB, _  = hex.DecodeString("6c7b10564cfe8e2e379470644dd04e39ea08a8241ff07e19d45836191ffaae97")
	testLeaderPrivateKeyB, _ = hex.DecodeString("bb29f145eba51d934f8566c6128adebc1ad3fc58205da78f29ad9a5f49eede226c7b10564cfe8e2e379470644dd04e39ea08a8241ff07e19d45836191ffaae97")
	testLeaderPublicKeyC, _  = hex.DecodeString("5cd8903c34e2da7efbfd7bc320594184461d406c9baaa22b33370ead8a767ab0")
	testLeaderPrivateKeyC, _ = hex.DecodeString("bf1a84322ccf4cbabf69e5d32e17b02c36fdba2461f1f11ae53b7884d92913555cd8903c34e2da7efbfd7bc320594184461d406c9baaa22b33370ead8a767ab0")
)

var (
	testByzAgreementValue1     = &wire.ByzAgreementValue{BlockHash: chainhash.HashH([]byte{1}), Leader: "a"}
	testByzAgreementValue2     = &wire.ByzAgreementValue{BlockHash: chainhash.HashH([]byte{2}), Leader: "b"}
	testByzAgreementValueEmpty = &wire.ByzAgreementValue{BlockHash: emptyBlockHash}
)

var tickerGenFn = func(timeOutInSec int) ticker {
	testTicker := mock.FakeTicker{
		C:      make(chan time.Time),
		Period: time.Duration(timeOutInSec) * time.Second,
	}
	return &testTicker
}

// init
func newConsenesusExecutorForTest(pk, sk []byte, startHeight int64, leaderRate uint32, testNodeNum int) *executor {
	shardIndex := shard.IDToShardIndex(0)
	config.SetDefaultConfig()
	config.GlobalConfig().TimeForPeerDiscovery = 0
	thb := &mock.HeartBeat{
		TestNodeNum: testNodeNum,
	}
	pubSubMgr := message.NewPubSubManager()
	p := &Params{
		pk:               pk,
		sk:               sk,
		heartbeatManager: thb,
		voteThreshold:    int(math.Ceil(float64(testNodeNum) * proportion)),
		pubSubMgr:        pubSubMgr,
	}

	// 避免当 testNodeNum 等于 3 时，voteThreshold 向上取整与 testNodeNum 相同
	if testNodeNum == 3 {
		p.voteThreshold = 2
	}

	mockABCI := mock.NewMockABCI(shardIndex, startHeight, pubSubMgr)
	ce := newExecutor(mockABCI, &Sortitionor{}, p)
	ce.gossipNode = newBroadcaster()
	fakeReShardProofs, _ := ce.vrf.Generate(pk, sk, []byte("FakeReshard"))
	ce.state.setInShardProof(fakeReShardProofs)
	ce.sortitionor = NewSortitionor(uint32(params.GenesisNumberOfShards), leaderRate)
	return ce
}

// test
func TestExecutor_startNewRoundAsNotPotentialLeader(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 0, -1)

	ce.Start()

	assert.Equal(t, 2, ce.state.getStep())
	assert.Empty(t, ce.gossipNode.(fakeGossipNode))
}

func TestExecutor_startNewRoundAsPotentialLeader(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 1)

	ce.Start()

	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	msgNum := len(ce.gossipNode.(fakeGossipNode))
	assert.Equal(t, 2, ce.state.getStep())
	assert.Equal(t, 2, msgNum)
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgSeed{}, msg)
	seedMsg := msg.(*wire.MsgSeed)
	// is Potential Leader
	seed := calcRoundSeed(seedMsg.LastSeedSig, seedMsg.GetRound())
	assert.Equal(t, true, ce.isPotentialLeader(seed[:]))
	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.LeaderProposalMessage{}, msg)
	leaderProposalMsg := msg.(*wire.LeaderProposalMessage)
	assert.Equal(t, seed, leaderProposalMsg.Block.Header.Seed)
}

func TestExecutor_SaveHighPrioritySeedMsg(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 2)
	seedMsg1, seed1 := generateSeedMsg(ce)
	ce.pk, ce.sk = testLeaderPublicKeyB, testLeaderPrivateKeyB
	seedMsg2, seed2 := generateSeedMsg(ce)
	var expectSeed = seed2
	if comparePriority(seed1, seed2) {
		expectSeed = seed1
	}

	ce.onSeed(curRound, seedMsg1)
	ce.onSeed(curRound, seedMsg2)

	assert.Equal(t, expectSeed, ce.state.getSmallestCred())
}

func TestExecutor_GenerateVoteForHighestPriorityLeaderProposal(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 2)
	leaderProposalMsg, seed1 := generateLeaderProposalMsg(ce, testLeaderPublicKeyA, testLeaderPrivateKeyA)
	var highPriorityMsg = leaderProposalMsg
	var highPrioritySeed = seed1
	leaderProposalMsg2, seed2 := generateLeaderProposalMsg(ce, testLeaderPublicKeyB, testLeaderPrivateKeyB)
	if !comparePriority(highPrioritySeed, seed2) {
		highPriorityMsg = leaderProposalMsg2
		highPrioritySeed = seed2
	}
	leaderProposalMsg3, seed3 := generateLeaderProposalMsg(ce, testLeaderPublicKeyC, testLeaderPrivateKeyC)
	if !comparePriority(highPrioritySeed, seed3) {
		highPriorityMsg = leaderProposalMsg3
		highPrioritySeed = seed3
	}

	ce.onReceiveLeaderProposal(curRound, leaderProposalMsg)
	ce.onReceiveLeaderProposal(curRound, leaderProposalMsg2)
	ce.onReceiveLeaderProposal(curRound, leaderProposalMsg3)

	assert.Equal(t, highPrioritySeed, ce.state.getSmallestCred())
	block, _ := ce.recvData.getBlock(ce.state.getRound(), highPriorityMsg.BlockHash)
	assert.Equal(t, block, highPriorityMsg.Block)
	vote := ce.gcPreVoter.generateVoteResult()
	assert.Equal(t, highPriorityMsg.BlockHash, vote.blockHash)
	assert.Equal(t, highPrioritySeed, vote.seed)
}

func TestExecutor_BroadcastGcMsgWhenGetEnoughVotes(t *testing.T) {
	// threshold = 3
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 2)
	leaderVoteMsgA1 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyA, testByzAgreementValue1)
	leaderVoteMsgB2 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyB, testByzAgreementValue2)
	leaderVoteMsgB1 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyB, testByzAgreementValue1)
	leaderVoteMsgC1 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyC, testByzAgreementValue1)

	ce.onReceiveLeaderVote(curRound, leaderVoteMsgA1)
	ce.onReceiveLeaderVote(curRound, leaderVoteMsgB2)
	ce.onReceiveLeaderVote(curRound, leaderVoteMsgB1)
	ce.onReceiveLeaderVote(curRound, leaderVoteMsgC1)

	assert.NotNil(t, ce.gcCommitVoter.generateVoteResult())
	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	assert.Equal(t, 1, len(ce.gossipNode.(fakeGossipNode)))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.GCMessage{}, msg)
	assert.Equal(t, testByzAgreementValue1, msg.(*wire.GCMessage).SignedVal.Message.ByzAgreementValue)
	assert.Equal(t, 4, ce.state.getStep())
}

func TestExecutor_BroadcastBBAMsgWhenGetEnoughGCVotes(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 4)
	expectedCredentialWithBA := wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(4), 0, *testByzAgreementValue1)
	gcMsgA1 := generateGCMsg(curRound, testLeaderPublicKeyA, testByzAgreementValue1)
	gcMsgA2 := generateGCMsg(curRound, testLeaderPublicKeyA, testByzAgreementValue2)
	gcMsgB1 := generateGCMsg(curRound, testLeaderPublicKeyB, testByzAgreementValue1)
	gcMsgC1 := generateGCMsg(curRound, testLeaderPublicKeyC, testByzAgreementValue1)

	ce.onGCMessage(curRound, gcMsgA1)
	ce.onGCMessage(curRound, gcMsgA2)
	ce.onGCMessage(curRound, gcMsgB1)
	ce.onGCMessage(curRound, gcMsgC1)

	// 达到阈值,进入下一步
	assert.Equal(t, 5, ce.state.getStep())
	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	assert.Equal(t, 1, len(ce.gossipNode.(fakeGossipNode)))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBA{}, msg)
	assert.Equal(t, expectedCredentialWithBA, msg.(*wire.MsgBinaryBA).SignedCredentialWithBA.Message.CredentialWithBA)
}

// Got enough value of current step
func TestExecutor_BBAEndingConditionOnZero(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 2, 1, 3)
	ce.state.setRound(1)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 7)
	curStep := ce.state.getStep()
	header := wire.BlockHeader{
		Version:         1,
		BlockBodyHash:   testByzAgreementValue1.BlockHash,
		PrevBlockHeader: chainhash.HashH([]byte{}),
		Height:          3,
		Seed:            chainhash.HashH([]byte{1}),
	}
	bbaMsgA1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyA, testByzAgreementValue1)
	bbaMsgB1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyB, testByzAgreementValue1)
	expectBBAMsg := wire.NewMessageBinaryBA(
		ce.state.getInShardProof(),
		&wire.SignedMsg{
			Pk: ce.pk,
			Message: wire.ConsensusMsg{
				CredentialWithBA: wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(-7), 0, *testByzAgreementValue1),
			},
		},
	)
	// set leaderproposal result
	ce.recvData.saveSeed(curRound, testByzAgreementValue1.Leader, chainhash.HashH([]byte{1}))
	ce.recvData.saveBlock(curRound, testByzAgreementValue1.BlockHash,
		&wire.MsgBlock{
			Header: header,
			Body:   &wire.BlockBody{},
		})
	// set gc result
	ce.bbaVoter.setResultValue(curRound, testByzAgreementValue1)

	ce.onBinaryBA(curRound, bbaMsgA1)
	ce.onBinaryBA(curRound, bbaMsgB1)

	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.BlockHeader{}, msg)
	assert.Equal(t, header, *msg.(*wire.BlockHeader))
	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBAFin{}, msg)
	assert.Equal(t, -7, msg.(*wire.MsgBinaryBAFin).GetStep())
	assert.Equal(t, expectBBAMsg.SignedCredentialWithBA.Message, msg.(*wire.MsgBinaryBAFin).SignedCredentialWithBA.Message)
	assert.Equal(t, ce.state.getRound(), curRound+1)
	assert.Equal(t, &header.Seed, ce.state.getRoundSeed(curRound))

	assert.NotEmpty(t, ce.blockReporter)
	block := <-ce.blockReporter
	assert.Equal(t, chainhash.HashH(header.Pk).String(), block.leader)

}

// Ending condition 1, then force to consensus on empty block
func TestExecutor_BBAEndingConditionOnOne(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 5)
	curStep := ce.state.getStep()
	emptyBlockHeader := &wire.BlockHeader{IsEmptyBlock: true, Version: 1, Height: 2}
	bbaMsgANil := generateBBAMsg(curRound, curStep, testLeaderPublicKeyA, nil)
	bbaMsgBNil := generateBBAMsg(curRound, curStep, testLeaderPublicKeyB, nil)
	expectBBAMsg := wire.NewMessageBinaryBA(
		ce.state.getInShardProof(),
		&wire.SignedMsg{
			Pk: ce.pk,
			Message: wire.ConsensusMsg{
				CredentialWithBA: wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(-5), 1, *testByzAgreementValueEmpty),
			},
		},
	)

	ce.onBinaryBA(curRound, bbaMsgANil)
	ce.onBinaryBA(curRound, bbaMsgBNil)

	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.BlockHeader{}, msg)
	assert.Equal(t, *emptyBlockHeader, *msg.(*wire.BlockHeader))
	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBAFin{}, msg)
	assert.Equal(t, -5, msg.(*wire.MsgBinaryBAFin).GetStep())
	assert.Equal(t, expectBBAMsg.SignedCredentialWithBA.Message, msg.(*wire.MsgBinaryBAFin).SignedCredentialWithBA.Message)
	assert.Equal(t, ce.state.getRound(), curRound+1)
	assert.Equal(t, emptySeed, ce.state.getRoundSeed(curRound))
	assert.NotEmpty(t, ce.blockReporter)
	block := <-ce.blockReporter
	assert.Equal(t, chainhash.HashH(emptyBlockHeader.Pk).String(), block.leader)
}

func TestExecutor_ExitBBAWhenExceedMaxStep(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 51)
	curStep := ce.state.getStep()
	block := &wire.MsgBlock{
		Header: wire.BlockHeader{
			ShardIndex:   0,
			Height:       2,
			IsEmptyBlock: true,
			Version:      1,
		},
	}
	bbaMsgA1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyA, testByzAgreementValue1)
	bbaMsgBEmpty := generateBBAMsg(curRound, curStep, testLeaderPublicKeyB, testByzAgreementValueEmpty)
	bbaMsgC1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyC, testByzAgreementValue1)
	expectBBAMsg := wire.NewMessageBinaryBA(
		ce.state.getInShardProof(),
		&wire.SignedMsg{
			Pk: ce.pk,
			Message: wire.ConsensusMsg{
				CredentialWithBA: wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(-51), 1, *testByzAgreementValueEmpty),
			},
		},
	)

	ce.onBinaryBA(curRound, bbaMsgA1)
	ce.onBinaryBA(curRound, bbaMsgBEmpty)
	ce.onBinaryBA(curRound, bbaMsgC1)

	// curStep have exceed bbaMaxStep, then force to consensus on empty block
	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.BlockHeader{}, msg)
	assert.Equal(t, block.Header, *msg.(*wire.BlockHeader))
	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBAFin{}, msg)
	assert.Equal(t, -51, msg.(*wire.MsgBinaryBAFin).GetStep())
	assert.Equal(t, expectBBAMsg.SignedCredentialWithBA.Message, msg.(*wire.MsgBinaryBAFin).SignedCredentialWithBA.Message)
	assert.Equal(t, ce.state.getRound(), curRound+1)
	assert.Equal(t, emptySeed, ce.state.getRoundSeed(curRound))
	assert.NotEmpty(t, ce.blockReporter)
	blockInfo := <-ce.blockReporter
	assert.Equal(t, chainhash.HashH(block.Header.Pk).String(), blockInfo.leader)
}

func TestExecutor_BBAContinueOnFixFlipped(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	step := rand.Intn(2) + 5
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, step)
	curStep := ce.state.getStep()
	credentialWithBA := wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(step+1), 0, *testByzAgreementValueEmpty)
	expectBBAMsg := wire.NewMessageBinaryBA(
		ce.state.getInShardProof(),
		&wire.SignedMsg{
			Pk: ce.pk,
			Message: wire.ConsensusMsg{
				CredentialWithBA: credentialWithBA,
			},
		},
	)
	bbaMsgA1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyA, testByzAgreementValue1)
	bbaMsgBEmpty := generateBBAMsg(curRound, curStep, testLeaderPublicKeyB, testByzAgreementValueEmpty)
	bbaMsgC2 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyC, testByzAgreementValue2)

	ce.onBinaryBA(curRound, bbaMsgA1)
	ce.onBinaryBA(curRound, bbaMsgBEmpty)
	ce.onBinaryBA(curRound, bbaMsgC2)

	assert.Equal(t, step+1, ce.state.getStep())
	assert.NotEmpty(t, ce.gossipNode.(fakeGossipNode))
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBA{}, msg)
	// ignore signature
	msg.(*wire.MsgBinaryBA).SignedCredentialWithBA.Signature = nil
	assert.Equal(t, expectBBAMsg, msg)
}

func TestExecutor_HandleEmptyBlockBlockConfirmationMsgInShardPreparation(t *testing.T) {
	round := 1
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 2, 1, 3)
	ce.state.setRound(round)
	ce.state.shardStatus = shardPreparation
	curRound := ce.state.getRound()
	header := wire.BlockHeader{
		Version:      1,
		IsEmptyBlock: true,
		Height:       3,
		Seed:         chainhash.HashH([]byte{1}),
	}
	blcokConfirmationMsg := wire.NewMessageBlockConfirmation(
		ce.shardIndex,
		int32(round),
		int32(7),
		testByzAgreementValue1,
		&header,
		nil,
	)

	ce.onMsgBlockConfirmationInShardPreparation(curRound, blcokConfirmationMsg)

	assert.Equal(t, shardEnabled, ce.state.getShardStatus())
	assert.Equal(t, round+1, ce.state.getRound())
	assert.Equal(t, chainhash.HashH([]byte{1}), *ce.state.getRoundSeed(1))
}

func TestExecutor_HandleBlockConfirmationMsgInShardPreparation(t *testing.T) {
	round := 2
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 3, 1, 3)
	ce.state.setRound(round)
	ce.state.shardStatus = shardPreparation
	curRound := ce.state.getRound()
	header := wire.BlockHeader{
		Version:         1,
		Height:          4,
		Seed:            chainhash.HashH([]byte{1}),
		BlockBodyHash:   testByzAgreementValue1.BlockHash,
		PrevBlockHeader: chainhash.HashH([]byte{}),
	}
	blockConfirmationMsg := wire.NewMessageBlockConfirmation(
		ce.shardIndex,
		int32(round),
		int32(7),
		testByzAgreementValue1,
		&header,
		nil,
	)
	ce.recvData.cacheBlock(header.BlockHeaderHash(),
		&wire.MsgBlock{
			Header: header,
			Body:   &wire.BlockBody{},
		})

	ce.onMsgBlockConfirmationInShardPreparation(curRound, blockConfirmationMsg)

	assert.Equal(t, shardEnabled, ce.state.getShardStatus())
	assert.Equal(t, ce.state.getRound(), round+1)
	assert.Equal(t, *ce.state.getRoundSeed(round), chainhash.HashH([]byte{1}))
}

func TestExecutor_firstStepOfGCTimeout(t *testing.T) {
	// testNodeNum = -1 表示'永远'不为提块者
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 0, -1)
	ce.timer = &consensusTimer{tickerGen: tickerGenFn}

	// ce.Start() 将会执行 ce.timer.startAtRound(round, ce.onTimeout, ce.gcTimeOutInSec)
	ce.Start()
	// 手动触发超时
	ce.timer.ticker.(*mock.FakeTicker).Tick()

	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.LeaderVoteMessage{}, msg)
	assert.Equal(t, testByzAgreementValueEmpty, msg.(*wire.LeaderVoteMessage).GetByzAgreementValue())
}

func TestExecutor_secondStepOfGCTimeout(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 0, -1)
	ce.timer = &consensusTimer{tickerGen: tickerGenFn}

	// ce.Start() 将会执行 ce.timer.startAtRound(round, ce.onTimeout, ce.gcTimeOutInSec)
	ce.Start()
	ce.timer.currentStep = 2
	// 手动触发超时
	ce.timer.ticker.(*mock.FakeTicker).Tick()

	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.GCMessage{}, msg)
	assert.Equal(t, testByzAgreementValueEmpty, msg.(*wire.GCMessage).GetByzAgreementValue())
	assert.Equal(t, 4, ce.state.getStep())
}

func TestExecutor_onGCLastStepTimeout(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	ce.gcTimer = &consensusTimer{tickerGen: tickerGenFn}
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 2)
	leaderVoteMsgA1 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyA, testByzAgreementValue1)
	leaderVoteMsgB1 := generateLeaderVoteMsg(curRound, testLeaderPublicKeyB, testByzAgreementValue1)
	// 收到足够 LeaderVote， 进入下一步，触发 onGCLastStepTimeout
	ce.onReceiveLeaderVote(curRound, leaderVoteMsgA1)
	ce.onReceiveLeaderVote(curRound, leaderVoteMsgB1)
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.GCMessage{}, msg)

	// 确保 onGCLastStepTimeout 执行
	time.Sleep(time.Second)
	ce.gcTimer.ticker.(*mock.FakeTicker).Tick()

	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBA{}, msg)
	assert.Equal(t, *testByzAgreementValueEmpty, msg.(*wire.MsgBinaryBA).GetByzAgreementValue())
	assert.Equal(t, 5, ce.state.getStep())
}

func TestExecutor_onBBATimeoutForNextStep(t *testing.T) {
	ce := newConsenesusExecutorForTest(testLeaderPublicKeyA, testLeaderPrivateKeyA, 1, 1, 3)
	ce.state.setRound(0)
	ce.bbaTimer = &consensusTimer{tickerGen: tickerGenFn}
	curRound := ce.state.getRound()
	ce.state.setStepInRound(curRound, 6)
	curStep := ce.state.getStep()
	bbaMsgA2 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyA, testByzAgreementValue2)
	bbaMsgB1 := generateBBAMsg(curRound, curStep, testLeaderPublicKeyB, testByzAgreementValue1)
	ce.onBinaryBA(curRound, bbaMsgA2)
	ce.onBinaryBA(curRound, bbaMsgB1)
	msg := <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBA{}, msg)
	assert.Equal(t, 7, ce.state.getStep())

	// 模拟 BBA 始终收不到投票触发超时
	time.Sleep(time.Second)
	ce.bbaTimer.ticker.(*mock.FakeTicker).Tick()

	msg = <-ce.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBinaryBA{}, msg)
	bbaMsg := msg.(*wire.MsgBinaryBA)
	assert.Equal(t, 7, bbaMsg.GetStep())
	assert.Equal(t, uint8(0), bbaMsg.GetBValue())
	assert.Equal(t, *testByzAgreementValueEmpty, bbaMsg.GetByzAgreementValue())
	assert.Equal(t, 8, ce.state.getStep())
}

// helper
func generateSeedMsg(ce *executor) (*wire.MsgSeed, chainhash.Hash) {
	lastSeedSig, seed, _ := ce.generateNewRoundSeed(ce.state.getRound())
	return &wire.MsgSeed{
		SignedCredential: &wire.SignedMsg{
			Message: wire.ConsensusMsg{Credential: wire.NewCredential(int32(ce.state.getRound()), 1)},
		},
		LastSeedSig: lastSeedSig,
	}, seed
}

func generateLeaderProposalMsg(ce *executor, pk, sk []byte) (*wire.LeaderProposalMessage, chainhash.Hash) {
	lastSeedSig, seed, _ := ce.generateNewRoundSeed(ce.state.getRound())
	block := &wire.MsgBlock{
		Header: wire.BlockHeader{
			ShardIndex: 0,
			Height:     int64(ce.state.getRound() + 2),
			Version:    1,
			Pk:         pk,
		},
		Body: &wire.BlockBody{},
	}

	return &wire.LeaderProposalMessage{
		ShardIndex: ce.shardIndex,
		SignedCredential: &wire.SignedMsg{
			Message: wire.ConsensusMsg{Credential: wire.NewCredential(int32(ce.state.getRound()), 1)},
			Pk:      pk,
		},
		LastSeedSig: lastSeedSig,
		Block:       block,
		BlockHash:   block.Header.BlockHeaderHash(),
	}, seed
}

func generateLeaderVoteMsg(round int, pk []byte, val *wire.ByzAgreementValue) *wire.LeaderVoteMessage {
	cred := wire.NewCredential(int32(round), 2)
	return &wire.LeaderVoteMessage{
		SignedCredential: &wire.SignedMsg{
			Message: wire.ConsensusMsg{Credential: cred},
			Pk:      pk,
		},
		SignedVal: &wire.SignedMsg{
			Message: wire.ConsensusMsg{ByzAgreementValue: val},
			Pk:      pk,
		},
	}
}

func generateGCMsg(round int, pk []byte, val *wire.ByzAgreementValue) *wire.GCMessage {
	cred := wire.NewCredential(int32(round), 3)
	return &wire.GCMessage{
		SignedCredential: &wire.SignedMsg{
			Message: wire.ConsensusMsg{Credential: cred},
			Pk:      pk,
		},
		SignedVal: &wire.SignedMsg{
			Message: wire.ConsensusMsg{ByzAgreementValue: val},
			Pk:      pk,
		},
	}
}

func generateBBAMsg(round, step int, pk []byte, v *wire.ByzAgreementValue) *wire.MsgBinaryBA {
	var b byte
	if v == nil {
		b = 1
		v = &wire.ByzAgreementValue{}
	}
	credentialWithBA := wire.NewCredentialWithBA(0, int32(round), int32(step), b, *v)
	return wire.NewMessageBinaryBA(nil, &wire.SignedMsg{
		Pk: pk,
		Message: wire.ConsensusMsg{
			CredentialWithBA: credentialWithBA,
		},
	})
}

// comparePriority return true when seed1 is high Priority
func comparePriority(seed1, seed2 chainhash.Hash) bool {
	if !bytes.Equal(seed1[0:], seed2[0:]) && bytes.Compare(seed1[0:], seed2[0:]) > 0 {
		return false
	}
	return true
}

type fakeGossipNode chan wire.Message

func newBroadcaster() connection.GossipNode {
	ch := fakeGossipNode(make(chan wire.Message, 1000))
	return ch
}

func (b fakeGossipNode) BroadcastMessage(msg wire.Message, params *connection.BroadcastParams) {
	b <- msg
}

func (b fakeGossipNode) RegisterChannels(dispatch *connection.MessagesAndReceiver) {
}

func (b fakeGossipNode) HandleMessage(message wire.Message) {
}
