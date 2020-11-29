/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"math/rand"
	"time"

	"github.com/multivactech/MultiVAC/interface/iabci"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

var (
	fetchTransactionsInterval = time.Duration(randInt64(5, 10)) * time.Second
)

func randInt64(min, max int64) int64 {
	rand.Seed(time.Now().UnixNano())
	return min + rand.Int63n(max-min+1)
}

type threadedAbci struct {
	shardIndex shard.Index
	worker     *appBlockChain
	actorCtx   *message.ActorContext

	messageCh chan *connection.MessageAndReply // Used to receive message from other peers
}

// NewAppBlockChain creates a new instance of AppBlockChain.
func NewAppBlockChain(id shard.Index, blockChain iblockchain.BlockChain,
	dPool idepositpool.DepositPool,
	pubSubMgr *message.PubSubManager) iabci.AppBlockChain {
	return newThreadedAbci(id, blockChain, dPool, pubSubMgr)
}

func newThreadedAbci(tag shard.Index, blockchain iblockchain.BlockChain,
	dPool idepositpool.DepositPool,
	pubSubMgr *message.PubSubManager) *threadedAbci {
	tsa := &threadedAbci{
		shardIndex: tag,
		actorCtx:   message.NewActorContext(),
		messageCh:  make(chan *connection.MessageAndReply),
	}
	tsa.worker = newApplicationBlockChain(tsa.shardIndex, blockchain, dPool, pubSubMgr)
	tsa.actorCtx.StartActor(tsa.worker)

	// register and handle message that abci care aboout
	tsa.messageRegister()
	go tsa.handleMessage()

	return tsa
}

func (tsa *threadedAbci) handleMessage() {
	for msg := range tsa.messageCh {
		tsa.OnMessage(msg.Msg)
	}
}

func (tsa *threadedAbci) messageRegister() {
	shardID := tsa.shardIndex
	if connection.GlobalConnServer != nil {
		tag := []connection.Tag{
			{Msg: wire.CmdReturnInit, Shard: shardID},
			{Msg: wire.CmdReturnTxs, Shard: shardID},
			{Msg: wire.CmdSmartContractInfo, Shard: shardID},
		}
		for _, shardIndex := range shard.ShardList {
			tag = append(tag, connection.Tag{Msg: wire.CmdBlockHeader, Shard: shardIndex})
		}
		connection.GlobalConnServer.RegisterChannels(&connection.MessagesAndReceiver{
			Tags:     tag,
			Channels: tsa.messageCh,
		})
	}
}

func (tsa *threadedAbci) Stop() {
	tsa.worker.txFetchTicker.Stop()
	tsa.worker.isFetchingTx = false
}

// PreStart contains two steps:
// 1.set ledger status to false
// 2.fetch initdata
func (tsa *threadedAbci) PreStart() {
	address := tsa.worker.getAddress()
	connection.GlobalConnServer.BroadcastMessage(&wire.MsgFetchInit{ShardIndex: tsa.shardIndex, Address: address}, &connection.BroadcastParams{
		ToNode:  connection.ToStorageNode,
		ToShard: tsa.shardIndex,
	})
}

// Start the abci, and set the prevReshardheader
func (tsa *threadedAbci) Start(prevReshardheader *wire.BlockHeader) {
	// set the prevReshardHeader
	tsa.worker.setPrevReshardHeader(prevReshardheader)
	//tsa.worker.setLedgerStatus(false)
	tsa.worker.fetchNewLedger()
}

func (tsa *threadedAbci) OnMessage(msg wire.Message) {
	tsa.actorCtx.Send(tsa.worker, message.NewEvent(evtMsg, msg), nil)
}

func (tsa *threadedAbci) ProposeAndAcceptEmptyBlock(height int64) (*wire.BlockHeader, error) {
	resp := tsa.actorCtx.SendAndWait(tsa.worker, message.NewEvent(evtPropAndAcceptEmptyBlock, height)).(acceptEmptyBlockResponse)
	return resp.header, resp.err
}

func (tsa *threadedAbci) AcceptBlock(block *wire.MsgBlock) error {
	err := tsa.actorCtx.SendAndWait(tsa.worker, message.NewEvent(evtAcceptBlockReq, block))
	if err == nil {
		return nil
	}
	return err.(error)
}

func (tsa *threadedAbci) VerifyBlock(block *wire.MsgBlock) error {
	err := tsa.actorCtx.SendAndWait(tsa.worker, message.NewEvent(evtVrfBlockReq, block))
	if err == nil {
		return nil
	}
	return err.(error)
}

func (tsa *threadedAbci) ProposeBlock(pk, sk []byte) (*wire.MsgBlock, error) {
	resp := tsa.actorCtx.SendAndWait(tsa.worker, message.NewEvent(evtPropBlockReq, proposeBlockRequest{pk: pk, sk: sk})).(proposeBlockResponse)
	return resp.block, resp.err
}
