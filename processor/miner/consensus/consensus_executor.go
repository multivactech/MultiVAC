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
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/iabci"
	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	// futureRoundThreshold means the threshold of sync condition,
	// if msg.round > consensus.round + futureRoundThreshold, it may be out of date, and need sync.
	futureRoundThreshold = 5
	// bbaMaxStep is the max step of bba, if step of bba more than the bbaMaxStep, it will force reach consensus, and
	// generate empty block.
	bbaMaxStep = 50
	// Proportion is the probability rate to calculate threshold when the number of miners is known.
	proportion float64 = 0.666666667
)

var emptyBlockHash = chainhash.Hash{}

// executor is the consensus's dominant executor,
// in which the iconsensus interface is implemented to implement the Byzantine algorithm
type executor struct {
	shardIndex                shard.Index
	state                     *consensusState
	vrf                       vrf.VRF
	sortitionor               iconsensus.Sortitionor
	depositFetcher            iconsensus.DepositFetcher
	gcPreVoter                *pbftVoter
	gcCommitVoter             *pbftVoter
	bbaVoter                  *binaryBAVoter
	blockchain                iabci.AppBlockChain
	pk                        []byte
	sk                        []byte
	futurePeerNumberThreshold int
	futureRoundThreshold      int
	gcTimeOutInSec            int
	timer                     *consensusTimer
	gcTimer                   *consensusTimer
	bbaTimer                  *consensusTimer

	// Message
	gossipNode connection.GossipNode // register message handle channels and send broadcast messages
	// blockReporter is used for testing only
	blockReporter chan *SimpleBlockInfo
	msgQueue      chan *connection.MessageAndReply

	threadHeartBeatManager iheartbeat.HeartBeat
	logger                 btclog.Logger

	recvData *receivedData
	pendData *pendingData

	event   *message.PubSubManager
	metrics *metrics.Metrics
}

// NewExecutor return consensus interface and register message handle channel.
func NewExecutor(abci iabci.AppBlockChain, selector iconsensus.Sortitionor, p *Params) iconsensus.Consensus {
	ce := newExecutor(abci, selector, p)
	// register message handle channel.
	if ce.gossipNode != nil {
		ce.gossipNode.RegisterChannels(&connection.MessagesAndReceiver{
			Tags: []connection.Tag{
				{Msg: wire.CmdBinaryBA, Shard: ce.shardIndex},
				{Msg: wire.CmdLeaderVote, Shard: ce.shardIndex},
				{Msg: wire.CmdGC, Shard: ce.shardIndex},
				{Msg: wire.CmdLeaderProposal, Shard: ce.shardIndex},
				{Msg: wire.CmdSeed, Shard: ce.shardIndex},
				{Msg: wire.CmdMsgBlockConfirmation, Shard: ce.shardIndex},
			},
			Channels: ce.msgQueue,
		})
	}
	ce.traceBlockForTesting()
	return ce
}

// NewExecutor function will return a point to executor, it will finish some initialization process.
func newExecutor(abci iabci.AppBlockChain, selector iconsensus.Sortitionor, p *Params) *executor {
	ce := &executor{}
	ce.shardIndex = p.shardIndex
	ce.logger = logBackend.Logger(fmt.Sprintf("CONS%d", ce.shardIndex.GetID()))
	ce.logger.SetLevel(logger.ConsLogLevel)
	ce.blockchain = abci
	ce.vrf = &vrf.Ed25519VRF{}
	ce.sortitionor = selector
	// TODO(ze): replace FakeDepositFetcher by DepositFetcher safely.
	ce.depositFetcher = &FakeDepositFetcher{}
	ce.threadHeartBeatManager = p.heartbeatManager
	ce.gcPreVoter = newPbftVoter(1, leaderProposalVoteType, ce.vrf, p.voteThreshold)
	ce.gcCommitVoter = newPbftVoter(1, leaderCommitVoteType, ce.vrf, p.voteThreshold)
	ce.gcTimeOutInSec = defaultGCTimeoutInSec
	ce.bbaVoter = newBinaryBAVoter(1, p.voteThreshold, ce.logger)
	ce.recvData = newReceivedData()
	ce.pendData = newPendingData()
	ce.state = newConsensusState()
	// TODO(liang): tune these thresholds
	ce.futureRoundThreshold = futureRoundThreshold
	ce.futurePeerNumberThreshold = p.voteThreshold
	ce.metrics = p.metrics
	ce.gossipNode = connection.GlobalConnServer

	ce.pk = p.pk
	ce.sk = p.sk
	ce.state.setDepositCache(
		ce.depositFetcher.GetDepositUnit(p.pk, 0, ce.shardIndex),
	)

	ce.blockReporter = make(chan *SimpleBlockInfo, 1)

	ce.msgQueue = make(chan *connection.MessageAndReply, 10000)
	ce.event = p.pubSubMgr
	// Listen abci initialization status
	ce.event.Sub(newInitSubscriber(ce))

	return ce
}

// UpdateInShardProof will update executor.inshardProof, it will be safe in concurrent.
func (ce *executor) UpdateInShardProof(inShardProof []byte) {
	ce.state.setInShardProof(inShardProof)
	ce.logger.Debugf("update the inshard proof: %v", inShardProof)
}

// Start function include consensus initiation process, which, when executed,
// will implement the initialization logic for consensus_executor
func (ce *executor) Start() {
	ce.updateShardStatus(shardEnabled)
	curRound := ce.state.getRound()
	if curRound == initialRound {
		ce.logger.Debugf("executor start from beginning")
		ce.startAtRound(startRound)
	} else {
		// the state of sync was update in sync_listener
		ce.logger.Debugf("executor WaitForBlockConfirm, with round %v", curRound)
	}
}

// StopConsensusAtRound means when shard needs to be disabled, stop the consensus firstly
func (ce *executor) StopConsensusAtRound(round int) {
	ce.state.updateShardStopRound(round)
}

// StartConsensusAtRound means set the consensus start at the round
func (ce *executor) StartConsensusAtRound(round int) {
	ce.state.updateShardStartRound(round)
}

func (ce *executor) onTimeout(round int, step int) {
	curRound := ce.state.getRound()
	curStep := ce.state.getStep()
	ce.logger.Debugf("timeout round %d, step %d, curRound:%d ,curStep:%d", round, step, curRound, curStep)

	if round != curRound {
		ce.logger.Debug("timeout.round != curRound or sync status != upToDate")
		return
	}
	switch step {
	case 1:
		vote := ce.gcPreVoter.generateVoteResult()
		var val *wire.ByzAgreementValue
		if vote != nil {
			val = &wire.ByzAgreementValue{
				Leader:    hex.EncodeToString(vote.votePublicKey),
				BlockHash: vote.blockHash,
			}
			ce.logger.Debugf("round %v: vote for proposal from %v, with hash %v", round, val.Leader, val.BlockHash)
		} else {
			ce.logger.Debugf("round %v: Didn't get any valid proposal for first step of GC, vote for empty block", round)
			val = &wire.ByzAgreementValue{}
		}
		cred := wire.NewCredential(int32(round), 2)
		msg := &wire.LeaderVoteMessage{
			ShardIndex:   ce.shardIndex,
			InShardProof: ce.state.getInShardProof(),
			SignedCredential: &wire.SignedMsg{
				Message: wire.ConsensusMsg{Credential: cred},
				Pk:      ce.pk,
			},
			SignedVal: &wire.SignedMsg{
				Message: wire.ConsensusMsg{ByzAgreementValue: val},
				Pk:      ce.pk,
			},
		}
		if err := msg.Sign(ce.sk); err == nil {
			ce.broadcastMessage(msg, nil)
		}
	case 2:
		if curStep > 3 {
			ce.logger.Debugf("Second step of GC timeout, but step wrong, so skip.")
			return
		}
		ce.logger.Debugf("Second step of GC timeout, generate empty block")
		val := wire.ByzAgreementValue{} // empty block
		cred := wire.NewCredential(int32(round), 3)
		msg := &wire.GCMessage{
			ShardIndex: ce.shardIndex,
			SignedCredential: &wire.SignedMsg{
				Message: wire.ConsensusMsg{Credential: cred},
				Pk:      ce.pk,
			},
			SignedVal: &wire.SignedMsg{
				Message: wire.ConsensusMsg{ByzAgreementValue: &val},
				Pk:      ce.pk,
			},
			InShardProof: ce.state.getInShardProof(),
		}
		if err := msg.Sign(ce.sk); err == nil {
			ce.broadcastMessage(msg, nil)
		}
		ce.state.setStepInRound(round, 4)
		go ce.gcTimer.onTimeOut(ce.onGCLastStepTimeout(round, 4), 30)
	}
}

func (ce *executor) shouldSkipProposal(seed chainhash.Hash) bool {
	smallestCred := ce.state.getSmallestCred()
	if !bytes.Equal(smallestCred[0:], maxCred[0:]) && bytes.Compare(seed[0:], smallestCred[0:]) > 0 {
		ce.logger.Debugf("Got a larger credential than known, skip propagation")
		return true
	}
	// TODO(liang): we should check if we get more than one proposal from same peer,
	// (which means the peer is malicious), we should ignore all the proposals except the first one.
	return false
}

// generateNewRoundSeed returns the seed of last round and current round
func (ce *executor) generateNewRoundSeed(round int) ([]byte, chainhash.Hash, error) {
	//Generate seed
	prevSeed := ce.state.getRoundSeed(round - 1)
	if prevSeed == nil {
		return nil, chainhash.Hash{}, errors.New("cant get seed")
	}
	lastSeedSig, err := ce.vrf.Generate(ce.pk, ce.sk, prevSeed[0:])
	if err != nil {
		ce.logger.Debugf("Failed to sign %v", err)
		return nil, chainhash.Hash{}, err
	}

	// TODO:(guotao) use a more graceful way to convert object to bytes
	curSeed := calcRoundSeed(lastSeedSig, round)
	return lastSeedSig, curSeed, nil
}

func (ce *executor) signBlock(block *wire.MsgBlock) error {
	// Sign block header
	block.Header.Pk = ce.pk
	blockHeaderHash := block.Header.BlockHeaderHash()
	headerSig, err := ce.vrf.Generate(ce.pk, ce.sk, blockHeaderHash[0:])
	if err != nil {
		ce.logger.Debugf("Failed to sign blockheaderhash %v ", err)
		return err
	}
	block.Header.HeaderSig = headerSig

	return nil
}

func (ce *executor) setCurrentRound(round int) {
	ce.gcPreVoter.setCurrentRound(round)
	ce.gcCommitVoter.setCurrentRound(round)
	ce.bbaVoter.setCurrentRound(round)
	ce.state.setRound(round)
	// Reset threshold
	ce.setThreshold()
}

func (ce *executor) setThreshold() {
	// TODO(#157)
	threshold := int(math.Ceil(float64(ce.threadHeartBeatManager.PerceivedCount()) * proportion))
	if threshold <= 1 {
		threshold = int(^uint(0) >> 1)
	}
	fakeDepositUnit := float64(1.0)
	// TODO: just for testnet-3.0
	threshold = int(float64(threshold) * (1 - math.Pow((1 - params.SecurityLevel), fakeDepositUnit)))

	ce.sortitionor.SetLeaderRate(ce.threadHeartBeatManager.PerceivedCount(), ce.shardIndex)
	ce.gcPreVoter.setThreshold(threshold)
	ce.gcCommitVoter.setThreshold(threshold)
	ce.bbaVoter.setThreshold(threshold)
	ce.logger.Debugf("Reset a new round's threshold, current threshold is %d", threshold)
}

// Note: parameter tx is only used by test running. In real production,
// we pass nil here, and transactions are collected by other ways.
func (ce *executor) startAtRound(round int) {
	defer func() {
		go ce.replayPendingMessage(round)
		ce.cleanUnusedData(round)
	}()
	if !ce.state.isShardEnable() {
		// Update round so that the previous block won't be handled over and over again.
		ce.setCurrentRound(round)
		return
	}

	// If the round equals to the round which be setted to stop
	if round == ce.state.getShardStopRound() {
		ce.setCurrentRound(round)
		// Notify shardManager to stop this shard
		ce.state.setShardAndListenerDisabled()
		ce.logger.Debugf("ce want to startnewround: %v, but consensus touch the stop round. can't start new round,", round)
		return
	}

	// If it is not reshard done, it will not start the next round until the ledger state initialization is complete.
	if ce.state.getShardStatus() != shardEnabled {
		ce.logger.Debugf("It's not reshard complete, consensus is stoped")
		// Update round so that the previous block won't be handled over and over again.
		// Just like ce.stopping
		ce.setCurrentRound(round)
		return
	}

	ce.logger.Infof("Shard-%d start new round: %d", ce.shardIndex, round)

	ce.logger.Debugf("startAtRound %d, ce.shard state:%v, ce.sync state: %v", round,
		ce.state.getShardStatusString(ce.state.getShardStatus()),
		ce.state.getSyncStatusString(ce.state.getSyncStatus()))

	ce.timer.startNewRound(round, ce.onTimeout, ce.gcTimeOutInSec)

	ce.setCurrentRound(round)

	// Generate seed and lastSeedSig by last round seed
	lastSeedSig, seed, err := ce.generateNewRoundSeed(round)
	if err != nil {
		ce.logger.Debugf("generateNewRoundSeed error %v", err)
		return
	}

	if ce.isPotentialLeader(seed[0:]) {
		ce.logger.Infof("I'm potential leader for round %v", round)
		block, err := ce.blockchain.ProposeBlock(ce.pk, ce.sk)

		if err != nil || block == nil {
			ce.logger.Debugf("Cur round should init txpool, skip.")
			// If it's a potential leader but err when propose block, don't propose the block and entry next step.
			ce.state.setNextStepInRound(round, 2)
			return
		}

		block.Header.LastSeedSig = lastSeedSig
		block.Header.Seed = seed
		if err := ce.signBlock(block); err != nil {
			ce.logger.Debugf("SignBlock error round %v, %v", round, err)
		} else {
			ce.logger.Debugf("SignBlock block is %v", block)
		}

		if block.Header.Height != int64(round)+2 {
			ce.logger.Debugf("ERROR! Proposed block height %v should be equal to current round %v + 2.",
				block.Header.Height, int64(round))
			// If it's a potential leader but err when propose block, don't propose the block and entry next step.
			ce.state.setNextStepInRound(round, 2)
			return
		}
		ce.logger.Debugf("CORRECT! Proposed block height %v should be equal to current round %v + 2.",
			block.Header.Height, int64(round))
		cred := wire.NewCredential(int32(round), 1)

		// Sign block hash
		blockHash := block.Header.BlockHeaderHash()
		signedHash, err := ce.vrf.Generate(ce.pk, ce.sk, blockHash[:])
		if err != nil {
			ce.logger.Debugf("Can't sign block hash, %v", err)
			return
		}

		// If this seed is not smallest, skip
		if !ce.shouldSkipProposal(seed) {
			ce.logger.Debugf("block proposal for round %v: %v", round, block)
			ce.state.setSmallestCredInRound(round, seed)
			seedMsg := &wire.MsgSeed{
				ShardIndex: ce.shardIndex,
				SignedCredential: &wire.SignedMsg{
					Message: wire.ConsensusMsg{Credential: cred},
					Pk:      ce.pk,
				},
				LastSeedSig:  lastSeedSig,
				InShardProof: ce.state.getInShardProof(),
			}
			if err := seedMsg.Sign(ce.sk); err == nil {
				ce.broadcastMessage(seedMsg, nil)
			}
			proposalMsg := &wire.LeaderProposalMessage{
				ShardIndex: ce.shardIndex,
				SignedCredential: &wire.SignedMsg{
					Message: wire.ConsensusMsg{Credential: cred},
					Pk:      ce.pk,
				},
				LastSeedSig:  lastSeedSig,
				Block:        block,
				BlockHash:    blockHash,
				SignedHash:   signedHash,
				InShardProof: ce.state.getInShardProof(),
			}
			if err := proposalMsg.Sign(ce.sk); err == nil {
				ce.broadcastMessage(proposalMsg, nil)
			}
		} else {
			ce.logger.Debugf("Skip block proposal for round %v because already got credential with smaller hash", round)
		}
	} else {
		ce.logger.Debugf("I'm not potential leader for round %v", round)
	}
	ce.state.setNextStepInRound(round, 2)
}

// replayPendingMessage divided msg with step to 4 type.
// step 1: LeaderProposalMessage
// step 2: LeaderVoteMessage
// step 3: GCMessage
// step 4: BinaryBAMsg
func (ce *executor) replayPendingMessage(round int) {
	if pendingMsg, maxStep, exist := ce.pendData.getPendingMsg(round); exist {
		for step := 1; step <= maxStep; step++ {
			if _, exist := pendingMsg[step]; !exist {
				continue
			}
			for _, msg := range pendingMsg[step] {
				ce.logger.Debugf("Replay message %v for step %v", msg.Command(), step)
				ce.handleConsensusMsg(msg)
			}
		}
	}
}

func (ce *executor) cleanUnusedData(round int) {
	ce.pendData.deletePendingMsg(round)
	ce.recvData.cleanUnusedData(round)
}

func (ce *executor) onReceiveLeaderProposal(curRound int, message *wire.LeaderProposalMessage) {
	ce.logger.Debugf("reveive onReceiveLeaderProposal msg %v", message)
	curStep := ce.state.getStep()
	if curStep > 2 {
		ce.logger.Debugf("Should not process the message of this step, current step is %d", curStep)
		return
	}

	blockVerified := ce.blockchain.VerifyBlock(message.Block)
	if blockVerified != nil {
		if message.Block.Header.Height < int64(curRound+2) {
			ce.logger.Debugf("invalid block", blockVerified)
			return
		} //Ignore for I have missied some old blockes
	}

	ce.recvData.saveBlock(message.GetRound(), message.BlockHash, message.Block)

	seed := calcRoundSeed(message.LastSeedSig, message.GetRound())
	if ce.shouldSkipProposal(seed) {
		ce.logger.Debugf("CurRound should skipProposal.")
		return
	}

	ce.state.setSmallestCredInRound(curRound, seed)

	if ce.isCommitteeVerifier(ce.state.inShardProof, ce.state.getDepositCache(), ce.shardIndex) {
		ce.logger.Debugf("consensus_executor.go onReceiverLeader. Adding vote: %v", *message)
		ce.gcPreVoter.addVote(&vote{
			round:         message.GetRound(),
			voteSignature: message.SignedCredential.Signature,
			votePublicKey: message.SignedCredential.Pk,
			blockHash:     message.BlockHash,
			seed:          seed,
		})
	}
}

func (ce *executor) cacheLeaderProposal(message *wire.LeaderProposalMessage) {
	ce.logger.Debugf("ce resharding is in preparation status or receive future message, just cache the block received: %v", message.Block)
	ce.recvData.cacheBlock(message.BlockHash, message.Block)
	ce.recvData.saveBlock(message.GetRound(), message.BlockHash, message.Block)
}

func (ce *executor) onReceiveLeaderVote(curRound int, message *wire.LeaderVoteMessage) {
	inShardProof := ce.state.getInShardProof()
	curStep := ce.state.getStep()
	ce.logger.Debugf("receive leader vote on round: %v", curRound)

	if curStep > 3 {
		ce.logger.Debugf("recv outdated LeaderVote message")
		return
	}

	if ce.isCommitteeVerifier(inShardProof, ce.state.getDepositCache(), ce.shardIndex) {
		ce.logger.Debugf("recv round %v vote for proposal from leader %v, with block hash %v.",
			message.GetRound(), message.GetByzAgreementValue().Leader, message.GetByzAgreementValue().BlockHash)
		ce.gcCommitVoter.addVote(&vote{
			round:         message.GetRound(),
			voteSignature: message.SignedCredential.Signature,
			votePublicKey: message.SignedCredential.Pk,
			blockHash:     message.GetByzAgreementValue().BlockHash,
			leader:        message.GetByzAgreementValue().Leader,
		})

		if ce.state.getShardStatus() == shardPreparation {
			ce.logger.Debugf("Receive onReceiveLeaderVote, ce resharding is in ShardPrparation status, return")
			return
		}

		vote := ce.gcCommitVoter.generateVoteResult()
		if vote != nil {
			val := wire.ByzAgreementValue{
				Leader:    vote.leader,
				BlockHash: vote.blockHash,
			}

			cred := wire.NewCredential(int32(curRound), 3)
			gcMsg := &wire.GCMessage{
				ShardIndex:   ce.shardIndex,
				InShardProof: inShardProof,
				SignedCredential: &wire.SignedMsg{
					Message: wire.ConsensusMsg{Credential: cred},
					Pk:      ce.pk,
				},
				SignedVal: &wire.SignedMsg{
					Message: wire.ConsensusMsg{ByzAgreementValue: &val},
					Pk:      ce.pk,
				},
			}
			if err := gcMsg.Sign(ce.sk); err == nil {
				ce.logger.Debugf("got enough vote, broadcast gc message %v", gcMsg)
				ce.broadcastMessage(gcMsg, nil)
			}
			ce.state.setStepInRound(curRound, 4)
			go ce.gcTimer.onTimeOut(ce.onGCLastStepTimeout(curRound, 4), 30)
		} else {
			ce.logger.Debugf("no enough valid vote yet")
		}
	} else {
		ce.logger.Debugf("I am not verifier")
	}
}

func (ce *executor) onGCMessage(curRound int, message *wire.GCMessage) {
	msgRound := message.GetRound()
	msgStep := message.GetStep()
	v := message.GetByzAgreementValue()
	ce.logger.Debugf("got gc message for round %v step %v, message: %v", msgRound, msgStep, v)
	curStep := ce.state.getStep()
	if curStep > 4 {
		ce.logger.Debugf("got out-dated GC message, ignored")
		return
	}
	// here we don't care the binary value `b` yet
	if voteKey, agreeType, haveBroadcast := ce.bbaVoter.addGCVote(int(msgRound),
		msgStep, 0, v, message.SignedCredential.Pk); agreeType == totalAgreeOnZero {
		if haveBroadcast {
			// ce.logger.Infof("although GC got enough voteKey, but have broadcast, so ignore."+
			// 	"round %v, val: %v, voteKey %v, step %v", msgRound, v, voteKey, curStep)
			return
		}
		ce.logger.Debugf("Output of GC: got enough voteKey for round %v, val: %v, voteKey %v, step %v", msgRound, v, voteKey, curStep)
		var nextB byte
		if v.BlockHash != emptyBlockHash {
			nextB = 0
		} else {
			nextB = 1
		}
		ce.bbaVoter.setResultValue(msgRound, v)
		// here we don't care the curStep, because the gc have reach consensus, so we set step to 4.
		ce.state.setStepInRound(curRound, 4)

		finStep := 4
		msg := ce.checkVerifierAndGenerateSignedMessage(nextB, v, curRound, finStep)
		if ok := ce.state.setNextStepInRound(curRound, curStep+1); !ok {
			ce.logger.Debugf("failed to setNextStep, curStep: %v, targetStep: %v", curStep, curStep+1)
		}

		ce.broadcastMessage(msg, nil)

		// after broadcasting, conceptually we have already got to next step,
		// so register timeout callback for next step
		go ce.bbaTimer.onTimeOut(ce.onBBATimeoutForNextStep(curRound, curStep+1), 30)
	}
}

// startNewRound start new round with given block header.
func (ce *executor) startNewRound(header *wire.BlockHeader) {
	round := int(header.Height - 2)
	var leader string
	if !header.IsEmptyBlock {
		leader = chainhash.HashH(header.Pk).String()
	}
	blockInfo := &SimpleBlockInfo{round: int32(round), curHash: header.BlockHeaderHash(),
		prevHash: header.PrevBlockHeader, leader: leader}
	ce.logger.Debugf("NewExecutor block generated for shard %v, blockInfo: %v, block.hash: %v",
		ce.shardIndex, blockInfo, header.BlockBodyHash)

	if ce.blockReporter != nil {
		ce.blockReporter <- blockInfo
	}
	ce.state.setRoundSeed(round, header.Seed)
	ce.startAtRound(round + 1)
}

// onMsgBlockConfirmationInShardPreparation handle the block confirmation message in shard preparation state.
func (ce *executor) onMsgBlockConfirmationInShardPreparation(curRound int, message *wire.MsgBlockConfirmation) {
	ce.logger.Debugf("Receive MsgBlockConfirmation shard %s,round %d, Info %v,cur shard state: %v cur syncstatus: %v",
		message.ShardIndex.String(), message.Round, message, ce.state.getShardStatus(), ce.state.getSyncStatus())

	if curRound == int(message.Header.Height-1) {
		ce.logger.Debugf("On preparation round, receive duplicate message at height %d", message.Header.Height)
		return
	}

	var err error
	// If in resharding preparation round
	if message.Header.IsEmptyBlock {
		ce.logger.Debugf("on shard preparation state, other miners consensus a empty block")
		_, err = ce.blockchain.ProposeAndAcceptEmptyBlock(message.Header.Height)
	} else {
		if block, ok := ce.recvData.reloadBlock(message.Header.BlockHeaderHash()); ok {
			err = ce.blockchain.AcceptBlock(block)
		} else {
			err = fmt.Errorf("block which height = %v, hash = %v ,maybe have accepted or no time to receive the block",
				message.Header.Height, message.Header.BlockHeaderHash())
		}
	}

	nextRound := int(message.Header.Height - 1)
	ce.logger.Debugf("set the ce round: %v", nextRound)
	ce.setCurrentRound(nextRound - 1)

	if err != nil {
		//ce.logger.Errorf("accept block err: %v", err)
		return
	}

	if nextRound >= ce.state.getShardStartRound() {
		ce.logger.Debugf("ShardStartRound is %d, next round is %d", ce.state.getShardStartRound(), nextRound)
		ce.logger.Debugf("Set startAtRound %d, ce.shard state:%v, ce.sync state: %v", nextRound, ce.state.getShardStatus(), ce.state.getSyncStatus())
		ce.updateShardStatus(shardEnabled)
		//ce.updateSyncStatus(upToDate)
		ce.recvData.cleanCacheBlock()

		ce.startNewRound(message.Header)
	}
}

func (ce *executor) onSeed(curRound int, message *wire.MsgSeed) {
	seed := ce.cacheSeed(message)
	//TODO:validate sigSeed proofs
	if ce.shouldSkipProposal(seed) {
		return
	}
	ce.state.setSmallestCredInRound(curRound, seed)

	ce.logger.Debugf("Received seed for round %v,  seed: %v", curRound, seed.String())
}

func (ce *executor) cacheSeed(message *wire.MsgSeed) chainhash.Hash {
	recvRound := message.GetRound()
	seed := calcRoundSeed(message.LastSeedSig, recvRound)
	sender := hex.EncodeToString(message.GetSender())
	ce.recvData.saveSeed(recvRound, sender, seed)
	return seed
}

// This function should be registered at the second step of GC
// It will be called if we can't get enough votes when time out
func (ce *executor) onGCLastStepTimeout(round, step int) func() {
	return func() {
		ce.logger.Debugf("GC last step time out at round %d.", round)
		curRound := ce.state.getRound()
		curStep := ce.state.getStep()
		if curRound == round && curStep == step {
			var nextB byte = 1
			voteKey, voteValue := ce.bbaVoter.getVoteKeyWithHalfThreshold()
			if voteKey == nil {
				voteValue = &wire.ByzAgreementValue{BlockHash: emptyBlockHash, Leader: ""}
			}
			ce.bbaVoter.setResultValue(round, voteValue)
			finStep := step
			msg := ce.checkVerifierAndGenerateSignedMessage(nextB, voteValue, round, finStep)
			if ok := ce.state.setNextStepInRound(round, step+1); !ok {
				ce.logger.Debugf("failed to setNextStep, step: %v, targetStep: %v", step, step+1)
			}
			ce.broadcastMessage(msg, nil)
			// after broadcasting, conceptually we have already got to next step,
			// so register timeout callback for next step
			go ce.bbaTimer.onTimeOut(ce.onBBATimeoutForNextStep(round, step+1), 30)
		}
	}
}

func (ce *executor) onBBATimeoutForNextStep(round, step int) func() {
	return func() {
		ce.logger.Debugf("BBA time out at round %d step %d.", round, step)
		curRound := ce.state.getRound()
		curStep := ce.state.getStep()
		if curRound == round && curStep == step {
			var fixValue byte
			if (step-1)%3 == 0 {
				fixValue = 0
			} else if (step-1)%3 == 1 {
				fixValue = 1
			} else {
				// TODO: set fix value as lsb(min(recv_crediential))
				fixValue = byte(rand.Int() % 2)
			}
			bbaValue := ce.bbaVoter.getResultValue(round)
			if bbaValue != nil {
				finStep := step
				msg := ce.checkVerifierAndGenerateSignedMessage(fixValue, bbaValue, round, finStep)

				if ok := ce.state.setNextStepInRound(round, step+1); !ok {
					ce.logger.Debugf("failed to setNextStep, step: %v, targetStep: %v", step, step+1)
				}
				ce.broadcastMessage(msg, nil)
				// after broadcasting, conceptually we have already got to next step,
				// so register timeout callback for next step
				go ce.bbaTimer.onTimeOut(ce.onBBATimeoutForNextStep(round, step+1), 30)
			}
			return
		}
	}
}

/* broadcast reach end cont of bba */
func (ce *executor) broadcastBBAFinMsg(round, step int, b byte, blockHeader *wire.BlockHeader) {
	ce.logger.Debugf("broadcastBBAFinMsg %s round:%d step:%d", ce.shardIndex.String(),
		round, step)

	//for same shard
	//broadcast BinaryBA ,{b,v} step = FinStep
	v := ce.bbaVoter.getResultValue(round)
	if b == 1 {
		v = &wire.ByzAgreementValue{}
	}
	if v == nil {
		return
	}
	ce.logger.Debugf("Propose block header in broadcastBBAFinMsg: %v", *blockHeader)
	// Use -step to indicate that fin at step
	finStep := -step
	msg := ce.checkVerifierAndGenerateSignedMessage(b, v, round, finStep)
	if err := ce.state.setNextStepInRound(round, step+1); !err {
		ce.logger.Debugf("failed to setNextStep, curStep: %v, targetStep: %v", step, step+1)
	}
	var par = &connection.BroadcastParams{
		ToNode:      connection.ToAllNode,
		ToShard:     connection.ToAllShard,
		IncludeSelf: true,
	}
	ce.broadcastMessage(blockHeader, par)
	ce.broadcastMessage(msg, par)
	ce.logger.Debugf("broadcast block header: %v and bba fin message %v by %v", msg, blockHeader, hex.EncodeToString(ce.pk))
}

func (ce *executor) onBinaryBA(curRound int, message *wire.MsgBinaryBA) {
	ce.logger.Debugf("Receive BinaryBAMsg. %v", message)
	prev := ce.state.getRoundSeed(curRound)
	if prev == nil {
		return
	}
	curStep := ce.state.getStep()
	round := message.GetRound()
	step := message.GetStep()
	v := message.GetByzAgreementValue()
	b := message.GetBValue()

	// TODO(liang): better encoding of (b,v)
	if b == 1 {
		// when b == 1, we don't care the value of v, we set it a dummy one
		v = wire.ByzAgreementValue{}
	}

	//TODO(guotao): should validte MsgBinaryBA is from committee, and has never seen before

	if voteKey, agreeType, haveBroadcast := ce.bbaVoter.addVoteWithBBAMsg(int(round), int(step), b, &v, message); voteKey != nil {
		// Got enough value for current step
		if haveBroadcast {
			ce.logger.Debugf("Got enough vote from (%d,%d), but have broadcast, so ignore."+
				" current state (%d,%d), agreeType: %v", voteKey.round, voteKey.step, curRound,
				step, ce.bbaVoter.getAgreementString(agreeType))
			return
		}

		// Got enough value of past or current step
		// Ending Condition 0
		if (voteKey.step-1)%3 == 0 && agreeType == totalAgreeOnZero {
			ce.logger.Debugf("round %v, step %v, total agree on zero, val: %v", round, step, v)
			if _, ok := ce.recvData.getSeed(curRound, v.Leader); ok || v.BlockHash == emptyBlockHash {
				// Need to check when receiving LeaderProposal message
				if block, ok := ce.recvData.getBlock(curRound, v.BlockHash); ok {
					err := ce.blockchain.AcceptBlock(block)
					if err != nil {
						ce.logger.Debugf("Failed to call blockchain.AcceptBlock: %v", err)
						if curRound == ce.state.getRound() && err.Error() != "the block may have been accepted" {
							ce.maybeBehind(curRound)
						}
						return
					}

					// BBA 达成共识
					ce.broadcastBBAFinMsg(curRound, step, 0, &block.Header)
					ce.startNewRound(&block.Header)
				} else {
					// TODO(liang): it's possible that we get consensus on block hash, but haven't received the block content
					ce.logger.Debugf("Reached consensus, but hasn't received corresponding block content")
					if curRound == ce.state.getRound() {
						ce.maybeBehind(curRound)
					}
				}
			} else {
				ce.logger.Debugf("have not received seed for leader %v", v.Leader)
				if curRound == ce.state.getRound() {
					ce.maybeBehind(curRound)
				}
			}
			return
		}

		// Ending condition 1
		// Or curStep have exceed bbaMaxStep, then force to consensus on empty block
		if curStep > bbaMaxStep || ((voteKey.step-1)%3 == 1 && agreeType == agreeOnOne) {
			emptyHeight := int64(curRound + 2)
			emptyBlockHeader, err := ce.blockchain.ProposeAndAcceptEmptyBlock(emptyHeight)
			if err != nil {
				ce.logger.Debugf("Failed to call blockchain.AcceptBlock for empty block: %v", err)
				return
			}
			ce.broadcastBBAFinMsg(curRound, step, 1, emptyBlockHeader)
			ce.startNewRound(emptyBlockHeader)
			return
		}

		ce.logger.Debugf("Got enough vote from (%d,%d), current state (%d,%d), agreeType: %v", voteKey.round,
			voteKey.step, curRound, step, ce.bbaVoter.getAgreementString(agreeType))

		if voteKey.step < 0 {
			ce.logger.Debugf("consensus reaches final step")
			return
		}

		var newB byte
		if agreeType == totalAgreeOnZero || agreeType == partialAgreeOnZero {
			newB = 0
		} else if agreeType == agreeOnOne {
			newB = 1
		} else {
			ce.logger.Debugf("Shouldn't happen!")
		}

		bbaValue := ce.bbaVoter.getResultValue(curRound)
		if bbaValue != nil {
			finStep := voteKey.step + 1
			msg := ce.checkVerifierAndGenerateSignedMessage(newB, bbaValue, curRound, finStep)
			if ok := ce.state.setNextStepInRound(curRound, step+1); !ok {
				ce.logger.Debugf("failed to setNextStep, curStep: %v, targetStep: %v", ce.state.getStep(), step+1)
			}

			ce.broadcastMessage(msg, nil)

			// after broadcasting, conceptually we have already got to next step,
			// so register timeout callback for next step
			go ce.bbaTimer.onTimeOut(ce.onBBATimeoutForNextStep(curRound, step+1), 30)
		}
	}
}

func (ce *executor) isPotentialLeader(seed []byte) bool {
	return ce.sortitionor.IsLeader(seed, ce.shardIndex)
}

func (ce *executor) isCommitteeVerifier(proofs []byte, depositUnit uint32, index shard.Index) bool {
	return ce.sortitionor.IsInShard(proofs, depositUnit, index)
}

func (ce *executor) checkVerifierAndGenerateSignedMessage(b byte, v *wire.ByzAgreementValue, curRound, finStep int) wire.Message {
	inShardProof := ce.state.getInShardProof()
	if !ce.isCommitteeVerifier(inShardProof, ce.depositFetcher.GetDepositUnit(inShardProof, int32(curRound), ce.shardIndex),
		ce.shardIndex) {
		ce.logger.Debugf("I am not in committee for round %v", curRound)
		return nil
	}
	credentialWithBA := wire.NewCredentialWithBA(ce.shardIndex, int32(curRound), int32(finStep), b, *v)
	bbaMsg := wire.NewMessageBinaryBA(
		inShardProof,
		&wire.SignedMsg{
			Pk: ce.pk,
			Message: wire.ConsensusMsg{
				CredentialWithBA: credentialWithBA,
			},
		})
	if err := bbaMsg.Sign(ce.sk); err != nil {
		return nil
	}
	if finStep < 0 {
		return wire.NewMessageBinaryBAFin(bbaMsg)
	}
	return bbaMsg
}

func (ce *executor) broadcastMessage(msg wire.Message, par *connection.BroadcastParams) {
	if par == nil {
		par = &connection.BroadcastParams{
			ToShard:     ce.shardIndex,
			ToNode:      connection.ToMinerNode,
			IncludeSelf: true,
		}
	}

	ce.gossipNode.BroadcastMessage(msg, par)
}

// UpdateShardStatus will update consensusState.shardStatus, it will be exported because processor will call it.
func (ce *executor) updateShardStatus(shardstate ceStatus) {
	if err := ce.state.updateShardStatus(shardstate); err != nil {
		ce.logger.Debugf("Failed to update shard status. detail: %v", err)
	}
}

// RegisterCEShardStateListener will add ce.shardStatus listener from processor.
func (ce *executor) RegisterCEShardStateListener(listener iconsensus.CEShardStateListener) {
	ce.state.registerCEShardStateListener(listener)
}

func (ce *executor) maybeBehind(msgRound int) {
	ce.logger.Debugf("maybeBehind on round: %v", msgRound)
	if ce.state.getShardStatus() != shardEnabled {
		ce.logger.Debugf("%v the shard state is not enabled, ignoring fall behind msg", ce)
		return
	}

	ce.event.Pub(*message.NewEvent(message.EvtLedgerFallBehind, nil))
}

// This if for testing, trace the information of localtest.
func (ce *executor) traceBlockForTesting() {
	go func(tmpShard shard.Index) {
		filename := config.GlobalConfig().DataDir + "/" + strconv.Itoa(int(tmpShard.GetID()))
		fp, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.ServerLogger().Errorf("Failed to create block info file: %v", err)
			return
		}
		for blockInfo := range ce.blockReporter {
			_, _ = fp.WriteString(blockInfo.String() + "\n")
		}
		_ = fp.Close()
	}(ce.shardIndex)
}
