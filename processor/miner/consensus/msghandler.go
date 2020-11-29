package consensus

import (
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
)

// consensusMsg is a new interface that can obtain all the information needed to determine the legitimacy of a message,
// without needing to know the specific message type
type consensusMsg interface {
	Command() string
	GetInShardProofs() [][]byte
	GetRound() int
	GetStep() int
	// todo(MH): handle panic
	IsValidated() error
	GetSignedCredential() *wire.SignedMsg
}

const handleMsgThreads = 4

// Init create handle Message Threads.
func (ce *executor) Init() {
	for i := 0; i < handleMsgThreads; i++ {
		go ce.messageHandlerThread()
	}
}

func (ce *executor) messageHandlerThread() {
	for msg := range ce.msgQueue {
		ce.dispatchHandleMessage(msg.Msg)
	}
}

// handleConsensusMsg will write to msgQueue the new Message, and it will be handled in dispatch goroutine.
func (ce *executor) handleConsensusMsg(msg wire.Message) {
	ce.msgQueue <- &connection.MessageAndReply{Msg: msg}
}

// dispatchHandleMessage will check if the message should be processed and handle the message in dispatch function.
func (ce *executor) dispatchHandleMessage(msg wire.Message) {
	cmsg := msg.(consensusMsg)
	curRound := ce.state.getRound()
	ce.logger.Debugf("Receive msg %s ,msg round %d, current round %d", msg.Command(), cmsg.GetRound(), curRound)
	switch ce.msgFilter(curRound, cmsg) {
	case ignore:
		return
	case handleMsg:
		ce.handleNewMessage(curRound, cmsg)
	case handleOtherStateMsg:
		ce.handleOtherStateMessage(curRound, cmsg)
	}
}

// handleNewMessage will dispatch msg to each corresponding solving function. when calling this function.
func (ce *executor) handleNewMessage(curRound int, msg consensusMsg) {
	switch msg := msg.(type) {
	case *wire.LeaderProposalMessage:
		ce.onReceiveLeaderProposal(curRound, msg)
	case *wire.LeaderVoteMessage:
		ce.onReceiveLeaderVote(curRound, msg)
	case *wire.GCMessage:
		ce.onGCMessage(curRound, msg)
	case *wire.MsgBinaryBA:
		ce.onBinaryBA(curRound, msg)
	case *wire.MsgSeed:
		ce.onSeed(curRound, msg)
	}
}

// handleOtherStateMessage will handle other state message.
// There are two situations:
// 1. ce state is in shardPreparation
// 2. message from future round
func (ce *executor) handleOtherStateMessage(curRound int, msg consensusMsg) {
	switch msg := msg.(type) {
	case *wire.MsgBlockConfirmation:
		ce.onMsgBlockConfirmationInShardPreparation(curRound, msg)
	case *wire.MsgSeed:
		ce.cacheSeed(msg)
	case *wire.LeaderProposalMessage:
		ce.cacheLeaderProposal(msg)
	default:
		ce.pendData.saveMsg(msg.GetRound(), msg.GetStep(), string(msg.GetSignedCredential().Pk), msg.(wire.Message))
	}
}
