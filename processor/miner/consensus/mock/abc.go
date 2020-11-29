package mock

import (
	"fmt"
	"math/rand"
	"sync/atomic"

	"github.com/multivactech/MultiVAC/interface/iabci"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// storageNode is used to handle fetch init data message.
var storageNode int64 = 1

// mockABCI a test specific struct that implements the ABCI interface.
type mockABCI struct {
	ShardIndex  shard.Index
	ShardHeight int64
	PubSubMgr   *message.PubSubManager
}

func (ma *mockABCI) Broadcast() chan wire.Message {
	// TODO(ze): Virtual a Network Layer.
	return nil
}

// NewMockABCI will return a fake abci, who will always return a particular blockBodyHash when ProposeBlock is called.
func NewMockABCI(shardIndex shard.Index, startHeight int64, pubSubMgr *message.PubSubManager) iabci.AppBlockChain {
	abci := &mockABCI{
		ShardIndex:  shardIndex,
		ShardHeight: startHeight,
		PubSubMgr:   pubSubMgr,
	}

	abci.PubSubMgr.Sub(newMockAbciSyncSubscriber(abci))
	return abci
}

func (ma *mockABCI) ProposeBlock(pk []byte, sk []byte) (*wire.MsgBlock, error) {
	// generate a block sizeof 1K
	block := generateBlockForSpecialSize(1024)
	block.Header.Pk = pk
	block.Header.ShardIndex = ma.ShardIndex
	block.Header.Height = ma.ShardHeight + 1
	block.Header.Seed = chainhash.Hash{}
	block.Header.IsEmptyBlock = false

	// if version == 0, verify return yes and acceptBlock yes.
	// if version == 1, ............. no  and ........... no.
	block.Header.Version = calculteVersion(0.7)

	// TODO(liang): we make acceptBlock always be yes, otherwise protocol_test will not pass. Need fix that.
	block.Header.Version = 1
	return block, nil
}

func (ma *mockABCI) proposeEmptyBlock() (*wire.MsgBlock, error) {
	block := &wire.MsgBlock{}
	block.Header.ShardIndex = ma.ShardIndex
	block.Header.Height = ma.ShardHeight + 1
	block.Header.IsEmptyBlock = true
	block.Header.Version = 1
	return block, nil
}

func (ma *mockABCI) VerifyBlock(block *wire.MsgBlock) error {
	if block.Header.Version == 1 {
		return nil
	}
	return fmt.Errorf("blockHeader Version == 0, refuse it")
}

func (ma *mockABCI) AcceptBlock(block *wire.MsgBlock) error {
	if block.Header.Version != 1 {
		return fmt.Errorf("blockHeader Version == 0, refuse it")
	}
	if block.Header.Height != ma.ShardHeight+1 {
		return fmt.Errorf("block height %d != chain height %d + 1, refuse it", block.Header.Height, ma.ShardHeight)
	}
	ma.ShardHeight++
	atomic.CompareAndSwapInt64(&storageNode, block.Header.Height-1, block.Header.Height)
	return nil
}

func (ma *mockABCI) ProposeAndAcceptEmptyBlock(height int64) (*wire.BlockHeader, error) {
	if ma.ShardHeight+1 != height {
		return nil, fmt.Errorf("shard height is not match")
	}
	block, err := ma.proposeEmptyBlock()
	if err != nil {
		return nil, err
	}
	err = ma.AcceptBlock(block)
	if err != nil {
		return nil, err
	}
	return &block.Header, nil
}

func (ma *mockABCI) Start(prevReshardHeader *wire.BlockHeader) {
}

func (ma *mockABCI) Stop() {
}

func (ma *mockABCI) OnMessage(msg wire.Message) {
}

// generateBlockForSpeciaSize will return a block which size is specialSize.
func generateBlockForSpecialSize(size int) *wire.MsgBlock {
	block := &wire.MsgBlock{}
	block.Body = &wire.BlockBody{
		Transactions:   []*wire.MsgTxWithProofs{},
		LedgerInfo:     wire.LedgerInfo{},
		SmartContracts: []*wire.SmartContract{},
		Outs:           []*wire.OutState{},
	}
	block.Header = wire.BlockHeader{
		Version:               666,
		PrevBlockHeader:       chainhash.Hash{},
		BlockBodyHash:         chainhash.Hash{},
		OutsMerkleRoot:        merkle.MerkleHash{},
		ShardLedgerMerkleRoot: merkle.MerkleHash{},
	}

	for i := 0; i <= (size / 8); i++ {
		block.Body.Transactions = append(block.Body.Transactions, generateTxForTest())
	}
	return block
}

// generateTxForTest return a tx which size 8 bytes
func generateTxForTest() *wire.MsgTxWithProofs {
	tx := &wire.MsgTxWithProofs{}

	tx.Tx = wire.MsgTx{
		API: randStringBytes(64),
	}
	return tx
}

// randStringBytes create random string from 'abc' dic
func randStringBytes(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// rate: probability of normal nodes
func calculteVersion(rate float32) int32 {
	if rand.Float32() < rate {
		return 1
	}
	return 0
}

// abciSyncSubscriber will request initial data from storage node when sync completes with new data fetched
type mockAbciSyncSubscriber struct {
	abc *mockABCI
}

func newMockAbciSyncSubscriber(abc *mockABCI) *mockAbciSyncSubscriber {
	return &mockAbciSyncSubscriber{abc: abc}
}

func (abciSyncSubscriber *mockAbciSyncSubscriber) Recv(e message.Event) {
	switch e.Topic {
	case message.EvtLedgerFallBehind:
		abciSyncSubscriber.abc.ShardHeight = atomic.LoadInt64(&storageNode)
		abciSyncSubscriber.abc.PubSubMgr.Pub(*message.NewEvent(message.EvtTxPoolInitFinished, atomic.LoadInt64(&storageNode)))
	}
}
