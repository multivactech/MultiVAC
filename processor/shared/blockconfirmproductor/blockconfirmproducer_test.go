package blockconfirmproductor

import (
	"encoding/hex"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/stretchr/testify/assert"
)

type fakeGossipNode chan wire.Message
type fakeHeartBeat struct {
	count int
}

var testShard = shard.Index(0)
var (
	testHeader1             = &wire.BlockHeader{Height: 2}
	testHeader2             = &wire.BlockHeader{Height: 3}
	testByzAgreementValue1  = &wire.ByzAgreementValue{BlockHash: testHeader1.BlockHeaderHash(), Leader: "a"}
	credentialWithBA1       = wire.NewCredentialWithBA(testShard, int32(0), int32(-7), 0, *testByzAgreementValue1)
	testLeaderPublicKeyA, _ = hex.DecodeString("885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
	testLeaderPublicKeyB, _ = hex.DecodeString("6c7b10564cfe8e2e379470644dd04e39ea08a8241ff07e19d45836191ffaae97")
)

var (
	testBBAFinA1 = &wire.MsgBinaryBAFin{
		MsgBinaryBA: wire.NewMessageBinaryBA(nil,
			&wire.SignedMsg{
				Pk: testLeaderPublicKeyA,
				Message: wire.ConsensusMsg{
					CredentialWithBA: credentialWithBA1,
				},
			})}
	testBBAFinA2 = &wire.MsgBinaryBAFin{
		MsgBinaryBA: wire.NewMessageBinaryBA(nil,
			&wire.SignedMsg{
				Pk: testLeaderPublicKeyA,
				Message: wire.ConsensusMsg{
					CredentialWithBA: credentialWithBA1,
				},
			})}
	testBBAFinB1 = &wire.MsgBinaryBAFin{
		MsgBinaryBA: wire.NewMessageBinaryBA(nil,
			&wire.SignedMsg{
				Pk: testLeaderPublicKeyB,
				Message: wire.ConsensusMsg{
					CredentialWithBA: credentialWithBA1,
				},
			})}
)

func TestBcProducer_setNewRound(t *testing.T) {
	bcp := newBcProducer(testShard, newHeartBeat(100), nil)
	bcp.receivedBlockHeader = map[int]map[chainhash.Hash]*wire.BlockHeader{
		0: nil,
		1: nil,
		2: nil,
	}
	bcp.bbaFinVoter.bbaHistory = map[voteKey]map[pkKey]*wire.MsgBinaryBAFin{
		{round: 1}: nil,
		{round: 0}: nil,
	}
	expectThreshold := 67

	bcp.setNewRound(1)

	assert.Equal(t, expectThreshold, bcp.bbaFinVoter.threshold)
	_, ok := bcp.receivedBlockHeader[0]
	assert.Equal(t, false, ok)
	assert.Equal(t, 2, len(bcp.receivedBlockHeader))
	_, ok = bcp.bbaFinVoter.bbaHistory[voteKey{round: 0}]
	assert.Equal(t, false, ok)
	assert.Equal(t, 1, len(bcp.bbaFinVoter.bbaHistory))
}

func TestBcProducer_onBlockHeaderSaveAndGet(t *testing.T) {
	tests := []struct {
		round   int
		header  *wire.BlockHeader
		success bool
	}{
		{
			round:   0,
			header:  testHeader1,
			success: true,
		},
		{
			round:   0,
			header:  testHeader2,
			success: true,
		},
		{
			round:   2,
			header:  testHeader1,
			success: false,
		},
	}

	for _, test := range tests {
		bcp := newBcProducer(testShard, newHeartBeat(100), nil)
		bcp.setNewRound(test.round)
		bcp.onBlockHeader(test.header)
		saveHeader, ok := bcp.getBlockHeader(int(test.header.Height-2), test.header.BlockHeaderHash())
		if ok {
			assert.Equal(t, test.header, saveHeader)
		}
		assert.Equal(t, test.success, ok)
	}
}

func TestBcProducer_onBBAFinMessageReachThreshold(t *testing.T) {
	bcp := newBcProducer(testShard, newHeartBeat(3), nil)
	bcp.gossipNode = newBroadcaster()
	bcp.setNewRound(0)
	bcp.bbaFinVoter.threshold = 2
	bcp.onBlockHeader(testHeader1)
	expectBlockconfirmationMsg := wire.NewMessageBlockConfirmation(testShard, 0, -7, testByzAgreementValue1,
		testHeader1, []*wire.MsgBinaryBAFin{testBBAFinA1, testBBAFinB1})

	bcp.onBBAFinMessage(testBBAFinA1)
	bcp.onBBAFinMessage(testBBAFinA2)
	bcp.onBBAFinMessage(testBBAFinB1)

	assert.NotEmpty(t, bcp.gossipNode)
	msg := <-bcp.gossipNode.(fakeGossipNode)
	assert.IsType(t, &wire.MsgBlockConfirmation{}, msg)
	assert.Equal(t, expectBlockconfirmationMsg, msg)
	assert.Equal(t, 1, bcp.bbaFinVoter.round)
	assert.Equal(t, 0, len(bcp.receivedBlockHeader))
	assert.Equal(t, 0, len(bcp.bbaFinVoter.bbaHistory))
}

func newBroadcaster() connection.GossipNode {
	ch := fakeGossipNode(make(chan wire.Message, 1000))
	return ch
}

func (b fakeGossipNode) BroadcastMessage(msg wire.Message, params *connection.BroadcastParams) {
}

func (b fakeGossipNode) RegisterChannels(dispatch *connection.MessagesAndReceiver) {
}

func (b fakeGossipNode) HandleMessage(message wire.Message) {
	b <- message
}

func newHeartBeat(count int) heartBeat {
	return &fakeHeartBeat{count: count}
}

func (h *fakeHeartBeat) PerceivedCount() int {
	return h.count
}

func (h *fakeHeartBeat) Has(pk []byte) bool {
	return true
}