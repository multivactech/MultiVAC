/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/pos/depositpool"
	"github.com/multivactech/MultiVAC/processor/shared/chain"
	"github.com/multivactech/MultiVAC/testutil"
	"github.com/stretchr/testify/assert"
)

var (
	cfg *config.Config
	pk  vrf.PublicKey
	sk  vrf.PrivateKey

	testShard = shard.IDToShardIndex(0)
	vrfEd     = &vrf.Ed25519VRF{}
	//numberOfHeart = 5
)

func createTestStorageNode(t *testing.T) *storageNode {
	loadConfig(t)
	bc := chain.NewService()                         // blockchain
	hb := testutil.NewMockHeart(5)                   // heartbeat
	dp, err := depositpool.NewDepositPool(bc, false) // depositpool
	if err != nil {
		t.Error(err)
	}
	sn := newStorageNode(testShard, StorageDir(cfg.DataDir), bc, dp, hb)
	sn.init()
	return sn
}

func loadConfig(t *testing.T) {
	_, err := config.LoadConfig()
	if err != nil {
		t.Error(err)
	}
	cfg = config.GlobalConfig()
	cfg.DataDir = "test"
	cfg.Sk = hex.EncodeToString(genesis.GenesisPrivateKeys[testShard])
	pk, sk, err = vrf.GeneratePkWithSk(cfg.Sk)
	if err != nil {
		t.Error(err)
	}
}

func clear() {
	os.RemoveAll("test")
}

func createTestSlimBlocks(t *testing.T, num int) []*wire.SlimBlock {
	// create fake chain
	chain := testutil.NewMockChain()
	fc := testutil.NewFakeChains(t, chain)
	addr := multivacaddress.GenerateAddress(genesis.GenesisPublicKeys[testShard], multivacaddress.UserAddress)
	prevHash := genesis.GenesisBlocks[testShard].Header.BlockHeaderHash()
	slimBlocks := make([]*wire.SlimBlock, 0)
	// get SlimBlocks
	for i := 0; i < num; i++ {
		val := new(big.Int).SetInt64(int64(i + 54))
		testMsgBlock := fc.Next(testShard, addr, val)
		testSlimBlock, err := wire.GetSlimBlockFromBlockByShard(testMsgBlock, testShard)
		if err != nil {
			t.Error(err)
		}
		// header hash
		testSlimBlock.Header.PrevBlockHeader = prevHash
		// set seed
		testSlimBlock.Header.Seed = calcSeed(testSlimBlock.Header.LastSeedSig, testSlimBlock.Header.Height-2)
		// signBlock
		if err = signBlock(testSlimBlock); err != nil {
			t.Error(err)
		}
		// record prevHash
		prevHash = testSlimBlock.Header.BlockHeaderHash()

		slimBlocks = append(slimBlocks, testSlimBlock)
	}
	return slimBlocks
}

func createTestConfirmations(t *testing.T, slimBlocks []*wire.SlimBlock) []*wire.MsgBlockConfirmation {
	confirmations := make([]*wire.MsgBlockConfirmation, 0)
	for _, slimBlock := range slimBlocks {
		cfm := &wire.MsgBlockConfirmation{Header: &slimBlock.Header}
		cfm.V.Leader = hex.EncodeToString(slimBlock.Header.Pk)
		confirmations = append(confirmations, cfm)
	}
	return confirmations
}

func createTestBlocks(t *testing.T, num int) []*wire.MsgBlock {
	// create fake chain
	chain := testutil.NewMockChain()
	fc := testutil.NewFakeChains(t, chain)
	addr := multivacaddress.GenerateAddress(genesis.GenesisPublicKeys[testShard], multivacaddress.UserAddress)
	blocks := make([]*wire.MsgBlock, 0)
	// get SlimBlocks
	for i := 0; i < num; i++ {
		val := new(big.Int).SetInt64(int64(i + 54))
		testMsgBlock := fc.Next(testShard, addr, val)
		blocks = append(blocks, testMsgBlock)
	}
	return blocks
}

func createTestTransactions(t *testing.T, num int) []*wire.MsgTx {
	// num should under 1001
	txs := make([]*wire.MsgTx, 0)
	outs := genesis.GenesisBlocks[testShard].Body.Outs
	key := genesis.GenesisPrivateKeys[testShard]
	for i := 0; i < num; i++ {
		tx := &wire.MsgTx{
			Shard:           testShard,
			TxIn:            []*wire.TxIn{&wire.TxIn{PreviousOutPoint: outs[i].OutPoint}},
			ContractAddress: isysapi.SysAPIAddress,
		}
		tx.Sign(&key)
		txs = append(txs, tx)
	}
	return txs
}

func verifySlimBlockInfo(t *testing.T, exp []*wire.SlimBlock, act []*wire.SlimBlock) {
	assert := assert.New(t)
	expLength := len(exp)
	assert.Equal(expLength, len(act), "Invalid length of slimBlockList")
	for i := 0; i < expLength; i++ {
		assert.Equal(exp[i].Header.Height, act[i].Header.Height, "Invalid height")
		assert.Equal(exp[i].Header.ShardIndex, act[i].Header.ShardIndex, "Invalid fromShard")
		assert.Equal(exp[i].ToShard, act[i].ToShard, "Invalid toShard")
	}
}

func verifyBlockInfo(t *testing.T, exp []*wire.MsgBlock, act []*wire.MsgBlock) {
	assert := assert.New(t)
	expLength := len(exp)
	assert.Equal(expLength, len(act), "Invalid length of blockList")
	for i := 0; i < expLength; i++ {
		assert.Equal(exp[i].Body.BlockBodyHash(), act[i].Body.BlockBodyHash(), "Invalid bodyHash")
		assert.Equal(exp[i].Header.BlockHeaderHash(), act[i].Header.BlockHeaderHash(), "Invalid headerHash")
		assert.Equal(exp[i].Header.Height, act[i].Header.Height, "Invalid height")
		assert.Equal(exp[i].Header.ShardIndex, act[i].Header.ShardIndex, "Invalid shardIndex")
	}
}

func signBlock(testSlimBlock *wire.SlimBlock) error {
	// PK
	testSlimBlock.Header.Pk = genesis.GenesisPublicKeys[testShard]
	// sig
	headerHash := testSlimBlock.Header.BlockHeaderHash()
	headerSig, err := vrfEd.Generate(pk, sk, headerHash[0:])
	if err != nil {
		return err
	}
	testSlimBlock.Header.HeaderSig = headerSig
	return nil
}

func calcSeed(lastSeedSig []byte, round int64) chainhash.Hash {
	int64Size := 8
	seedBuf := make([]byte, len(lastSeedSig)+int64Size)
	copy(seedBuf, lastSeedSig[0:])
	binary.PutVarint(seedBuf[len(lastSeedSig):], round)
	curSeed := sha256.Sum256(seedBuf)
	return curSeed
}
