package appblockchain

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/pos/depositpool"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	// waitingTime is the wait time for asynchronous program execution
	// Note that this parameter should not be set too small
	waitingTime = time.Millisecond * 500
)

var (
	currentBlock *wire.MsgBlock
)

func wait() {
	time.Sleep(waitingTime)
}

// ABCI module testing
func TestABCI(t *testing.T) {
	fmt.Println("Start module test......")
	assert := assert.New(t)
	// load config and create instance
	loadConfig(assert)
	var txTo, _ = multivacaddress.StringToUserAddress(config.GlobalConfig().Pk)
	var value = new(big.Int).SetInt64(10)
	pubSuber := message.NewPubSubManager()
	blockchain := testutil.NewMockChain()
	depositPool := depositpool.NewThreadedDepositPool(blockchain, false)
	txGen := newTxGen(shard0)
	fb := new(fakeGossipNode)
	abci := newApplicationBlockChain(shard0, blockchain, depositPool, pubSuber)
	abci.gossipNode = fb

	cfg := &toolsConfig{
		to:      txTo,
		binding: txTo,
		value:   value,
	}
	tools := newTools(blockchain, abci, depositPool, txGen, t, cfg)
	fb.messageCallBack = tools.handleMessage

	// abci initializes the ledger by requesting data from the storage node
	abci.fetchNewLedger()
	wait()
	assert.Equal(abci.shardHeight, int64(1), "Abci maybe not init successful")
	assert.Equal(abci.state.GetLedgerRoot().String(), tools.tree.MerkleRoot.String(), "Abci's root is not match tools root")

	// 1.Test request transaction
	// fetch normal transaction
	tools.setTxGenType("transfer")
	abci.fetchTransactions()
	wait()
	assert.Equal(abci.txPool.NumberOfTransactions(), 1, "Wrong number of tx in pool")

	// fetch deposit transaction
	tools.setTxGenType("deposit")
	abci.fetchTransactions()
	wait()
	assert.Equal(abci.txPool.NumberOfTransactions(), 2, "Wrong number of tx in pool")

	// 2.Test propose and accept block
	// propose block with height 2
	block1, err := abci.ProposeBlock(pk, sk)
	assert.Nil(err)
	assert.Nil(tools.receiveBlock(block1))
	assert.Equal(tools.height, int64(2), "Invalid height of tools")

	// accept block1
	assert.Nil(abci.AcceptBlock(block1))
	if tools.tree.MerkleRoot.String() != abci.state.GetLedgerRoot().String() {
		t.Error("Root not match at height 2")
	}

	// 3.Test add deposit tx to pool and update proof of deposit
	// propose block2 with height 3
	// Note: When accepting this block, deposit tx will add to deposit pool
	block2, err := abci.ProposeBlock(pk, sk)
	assert.Nil(err)
	assert.Nil(tools.receiveBlock(block2))
	assert.Equal(tools.height, int64(3), "Invalid height of tools")

	// deposit will be added to the pool
	assert.Nil(abci.AcceptBlock(block2))
	dtxs, err := depositPool.GetAll(abci.shardIndex, abci.getAddress())
	assert.Nil(err)
	assert.Equal(len(dtxs), 2, "Invalid number of deposit in deposit pool")

	// deposit tx will be updated when accepting a new block
	block3, err := abci.ProposeBlock(pk, sk)
	assert.Nil(err)
	assert.Nil(abci.VerifyBlock(block3))
	assert.Nil(abci.AcceptBlock(block3))

	dtxs, err = depositPool.GetAll(abci.shardIndex, abci.getAddress())
	assert.Nil(err)
	assert.Nil(dtxs[0].Proof.Verify(&block3.Header.ShardLedgerMerkleRoot))

	// 4.Test new peer add
	assert.Nil(tools.receiveBlock(block3))
	doolForNew := depositpool.NewThreadedDepositPool(blockchain, false)
	abciForNew := newApplicationBlockChain(shard0, blockchain, doolForNew, pubSuber)
	abciForNew.gossipNode = fb
	toolsForNew := newTools(blockchain, abciForNew, doolForNew, txGen, t, cfg)
	fb.messageCallBack = toolsForNew.handleMessage

	assert.Nil(toolsForNew.receiveBlock(block1))
	assert.Nil(toolsForNew.receiveBlock(block2))
	assert.Nil(toolsForNew.receiveBlock(block3))
	assert.Equal(toolsForNew.height, int64(4), fmt.Sprintf("Invalid height of tools: %d", tools.height))

	abciForNew.setPrevReshardHeader(&block3.Header)
	abciForNew.fetchNewLedger()
	wait()

	assert.Equal(abciForNew.shardHeight, int64(4), fmt.Sprintf("Invalid height of new node: %d", abciForNew.shardHeight))
	assert.Equal(abciForNew.state.GetLedgerRoot().String(), abci.state.GetLedgerRoot().String(), "New node's root is not match old node's")
	// test new peer's deposit proof
	newDtxs, err := depositPool.GetAll(abciForNew.shardIndex, abciForNew.getAddress())
	assert.Nil(err)
	assert.Equal(len(newDtxs), 2, "Invalid length of deposit proof in deposit pool")

	block4, err := abci.ProposeBlock(pk, sk)
	assert.Nil(err)
	assert.Nil(abciForNew.AcceptBlock(block4), fmt.Sprintf("New node can't accept old node's block: %v", err))

	// 5.Test reshard
	currentBlock = block3
	currentReshardSeed := abci.prevReshardHeader.ReshardSeed
	for i := currentBlock.Header.Height + 1; i <= params.ReshardRoundThreshold+1; i++ {
		b, err := abci.ProposeBlock(pk, sk)
		assert.Nil(err)

		err = abci.AcceptBlock(b)
		assert.Nil(err)
	}
	assert.NotEqual(currentReshardSeed, abci.prevReshardHeader.ReshardSeed, "Invalid seed")

	// Pass all test cases
	fmt.Println("Module test case pass!")
}
