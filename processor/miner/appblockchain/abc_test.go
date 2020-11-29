/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/pos/depositpool"
	"github.com/multivactech/MultiVAC/processor/miner/txprocessor"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/smartcontractdatastore"
	"github.com/multivactech/MultiVAC/processor/shared/txpool"
	"github.com/multivactech/MultiVAC/testutil"
	"github.com/stretchr/testify/assert"
)

var (
	shard0   = shard.ShardList[0]
	TestDir  = "testdata"
	pk, sk   []byte
	pks, sks string
)

type fakeGossipNode struct {
	messageCallBack func(msg wire.Message)
}

func (fb *fakeGossipNode) BroadcastMessage(msg wire.Message, params *connection.BroadcastParams) {
	go fb.messageCallBack(msg)
}

func (fb *fakeGossipNode) RegisterChannels(dispatch *connection.MessagesAndReceiver) {
}

func (fb *fakeGossipNode) HandleMessage(message wire.Message) {
}

// createFakeLedgerInfo creates a fake ledger for abci initialization
// ledger: [ shard0:1 ]
func createFakeLedgerInfo(assert *assert.Assertions) *wire.MsgReturnInit {
	// create a fullmerkletree
	var tree = merkle.NewFullMerkleTree()
	err := tree.Append(&genesis.GenesisBlocks[shard0].Header.OutsMerkleRoot)
	assert.Nil(err)
	err = tree.HookSecondLayerTree(genesis.GenesisBlocksOutsMerkleTrees[shard0])
	assert.Nil(err)

	// get lastnode's merkle path
	lastPath, err := tree.GetLastNodeMerklePath()
	assert.Nil(err)

	// ledger: [0:1,1:0,2:0,3:0]
	ledgerInfo := wire.LedgerInfo{
		Size: 1,
	}
	ledgerInfo.SetShardHeight(shard0, 1)

	address, err := multivacaddress.StringToUserAddress(pks)
	assert.Nil(err)

	deposits, err := getDepositsFromGenesisBlock(shard0, address, lastPath)
	assert.Nil(err)
	ledger := &wire.MsgReturnInit{
		Ledger:       ledgerInfo,
		RightPath:    *lastPath,
		ShardHeight:  1,
		LatestHeader: genesis.GenesisBlocks[shard0].Header,
		TreeSize:     1,
		ShardIndex:   shard0,
		Deposits:     deposits,
	}
	return ledger
}

// getDepositsFromGenesisBlock returns all deposits of the specific address
func getDepositsFromGenesisBlock(shard shard.Index, address multivacaddress.Address, path *merkle.MerklePath) ([]*wire.OutWithProof, error) {
	block := genesis.GenesisBlocks[shard]
	dPool, err := depositpool.NewDepositPool(testutil.NewMockChain(), false)
	if err != nil {
		return nil, err
	}

	BlockTree := getFullMerkleTreeByBlock(block)
	for _, out := range block.Body.Outs {
		if out.IsDepositOut() {
			outData, err := wire.OutToDepositData(&out.OutPoint)
			if err != nil {
				return nil, err
			}
			outHash := merkle.ComputeMerkleHash(out.ToBytesArray())

			// Get path from blockTree
			proof, err := BlockTree.GetMerklePath(outHash)
			if err != nil {
				return nil, err
			}

			// The miner only cares about his own deposit
			if outData.BindingAddress.IsEqual(address) {
				err = dPool.Add(&out.OutPoint, wire.BlockHeight(block.Header.Height), proof)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return dPool.GetAll(shard, address)
}

// clear clears the testdata
func clear() {
	os.RemoveAll(TestDir)
}

func loadConfig(assert *assert.Assertions) {
	_, err := config.LoadConfig()
	assert.Nil(err)

	// Randomly take a pair of public and private keys from the genesis block
	pool := genesis.NewPool()
	pks, sks, err = pool.Pop()
	assert.Nil(err)

	pk, err = hex.DecodeString(pks)
	assert.Nil(err)
	sk, err = hex.DecodeString(sks)
	assert.Nil(err)

	config.GlobalConfig().Pk = pks
	config.GlobalConfig().Sk = sks
	config.GlobalConfig().DataDir = TestDir
}

func createTestAbci(t *testing.T) *appBlockChain {
	// Remove all test dir firstly.
	err := os.RemoveAll(TestDir)
	if err != nil {
		t.Error(err)
	}

	assert := assert.New(t)

	// Load configuration parameters.
	loadConfig(assert)

	// Create fake blockchain, depositpool, pubsubManager and so on.
	b := testutil.NewMockChain()
	dp := depositpool.NewThreadedDepositPool(b, true)
	pubSubMgr := message.NewPubSubManager()
	fb := new(fakeGossipNode)
	fb.messageCallBack = func(msg wire.Message) {}

	// Create abci for testing
	abc := newApplicationBlockChain(shard0, b, dp, pubSubMgr)
	abc.gossipNode = fb
	abc.setPrevReshardHeader(&genesis.GenesisBlocks[shard0].Header)

	err = abc.handleInit(createFakeLedgerInfo(assert))
	if err != nil {
		t.Error(err)
	}
	return abc
}

func TestStorageReward(t *testing.T) {
	if !params.EnableStorageReward {
		return
	}
	up := func(mp map[string]*big.Int, key string, add *big.Int) {
		tmp, ok := mp[key]
		if !ok {
			tmp = big.NewInt(0)
		}
		tmp = new(big.Int).Add(tmp, add)
		mp[key] = tmp
	}

	o1 := make(map[string]*big.Int)
	o2 := make(map[string]*big.Int)
	storageAddress := []string{"storage1", "storage2", "storage3", "storage4"}
	userAddress := []string{"user1", "user2", "user3", "user4"}
	txs := make([]*wire.MsgTxWithProofs, 0)
	for i := 1; i < 200; i++ {
		val := big.NewInt(int64(i))
		txin := make([]*wire.TxIn, 0)
		in1 := &wire.TxIn{PreviousOutPoint: wire.OutPoint{
			Data: wire.MtvValueToData(val),
		}}
		in2 := &wire.TxIn{PreviousOutPoint: wire.OutPoint{
			Data: wire.MtvValueToData(val),
		}}
		txin = append(txin, in1)
		txin = append(txin, in2)
		in := new(big.Int).Add(val, val)
		in = new(big.Int).Sub(in, isysapi.StorageRewardAmount)
		tp := isysapi.TransferParams{
			Shard:  shard.Index(1),
			To:     []byte(userAddress[i%4]),
			Amount: in,
		}
		if in.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		params, _ := rlp.EncodeToBytes(tp)
		tx := &wire.MsgTxWithProofs{
			Tx: wire.MsgTx{
				Shard:              shard.Index(1),
				TxIn:               txin,
				ContractAddress:    isysapi.SysAPIAddress,
				API:                isysapi.SysAPITransfer,
				Params:             params,
				StorageNodeAddress: []byte(storageAddress[i%4]),
			},
		}
		if tx.Tx.IsEnoughForReward() {
			up(o1, storageAddress[i%4], isysapi.StorageRewardAmount)
			up(o1, userAddress[i%4], in)
			txs = append(txs, tx)
		}
	}
	for i := 11; i < 40; i++ {
		val := big.NewInt(int64(i))
		txin := make([]*wire.TxIn, 0)
		in1 := &wire.TxIn{PreviousOutPoint: wire.OutPoint{
			Data: wire.MtvValueToData(val),
		}}
		in2 := &wire.TxIn{PreviousOutPoint: wire.OutPoint{
			Data: wire.MtvValueToData(val),
		}}
		txin = append(txin, in1)
		txin = append(txin, in2)
		in := new(big.Int).Add(val, val)
		in = new(big.Int).Sub(in, isysapi.StorageRewardAmount)
		tp := isysapi.DepositParams{
			TransferParams: &isysapi.TransferParams{
				Shard:  shard.Index(1),
				To:     []byte(userAddress[i%4]),
				Amount: in,
			},
			BindingAddress: nil,
		}
		params, _ := rlp.EncodeToBytes(tp)
		tx := &wire.MsgTxWithProofs{
			Tx: wire.MsgTx{
				Shard:              shard.Index(1),
				TxIn:               txin,
				ContractAddress:    isysapi.SysAPIAddress,
				API:                isysapi.SysAPIDeposit,
				Params:             params,
				StorageNodeAddress: []byte(storageAddress[i%4]),
			},
		}
		if tx.Tx.IsEnoughForReward() {
			up(o1, storageAddress[i%4], isysapi.StorageRewardAmount)
			up(o1, userAddress[i%4], in)
			txs = append(txs, tx)
		}
	}
	storageRewardTx := getStorageReward(txs, 1, shard.Index(1))
	txs = append(txs, storageRewardTx...)
	var smart smartcontractdatastore.SmartContractDataStore
	var tp = txprocessor.NewTxProcessor(shard.Index(1), smart)
	for _, tx := range txs {
		tp.AddTx(&tx.Tx)
	}
	r := tp.Execute(1)
	outs := r.GetOutStates()
	for _, out := range outs {
		if out.IsDepositOut() {
			depositData, _ := wire.OutToDepositData(&out.OutPoint)
			up(o2, string(out.UserAddress), depositData.Value)
		} else {
			up(o2, string(out.UserAddress), wire.GetMtvValueFromOut(&out.OutPoint))
		}
	}
	if len(o1) != len(o2) {
		t.Errorf("reward fail")
	}
	for key := range o1 {
		if o1[key].Cmp(o2[key]) != 0 {
			t.Errorf("reward error")
		}
	}
}

func TestAcceptBlock(t *testing.T) {
	defer clear()
	addr := multivacaddress.GenerateAddress(genesis.GenesisPublicKeys[shard0], multivacaddress.UserAddress)
	value := new(big.Int).SetInt64(10)
	abc := createTestAbci(t)
	fc := testutil.NewFakeChains(t, abc.blockChain)
	block := fc.Next(shard0, addr, value)
	err := abc.AcceptBlock(block)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), abc.shardHeight)
	block2 := fc.Next(shard0, addr, value)
	err = abc.AcceptBlock(block2)
	assert.Nil(t, err)
	assert.Equal(t, int64(3), abc.shardHeight)

	err = abc.AcceptBlock(nil)
	assert.Equal(t, fmt.Errorf("block is nil"), err)
	assert.Equal(t, int64(3), abc.shardHeight)

	// change the reshard seed to make it error
	block.Header.ReshardSeed = make([]byte, 7)
	err = abc.AcceptBlock(block)
	assert.Equal(t, fmt.Errorf("the block may have been accepted"), err)
	assert.Equal(t, abc.shardHeight, int64(3))
}
func TestAcceptEmptyBlock(t *testing.T) {
	defer clear()
	abc := createTestAbci(t)
	_, err := abc.ProposeAndAcceptEmptyBlock(2)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), abc.shardHeight)
	_, err = abc.ProposeAndAcceptEmptyBlock(3)
	assert.Nil(t, err)
	assert.Equal(t, int64(3), abc.shardHeight)
}

func TestFetcherResponseNormal(t *testing.T) {
	defer clear()
	abc := createTestAbci(t)
	abc.fetcher.fetching[8] = newFetchInitDataAnnounce(abc.shardIndex, abc.shardHeight, abc.getAddress(), abc.handleInit, abc.broadcastMsg)
	abc.fetcher.fetching[8].ID = 8
	abc.fetcher.fetching[8].Time = time.Now()
	abc.onNewMessage(&wire.MsgReturnInit{
		MsgID: 8,
	})
	time.Sleep(time.Millisecond * 100)
	assert.NotContains(t, abc.fetcher.fetching, uint32(8))
}

func TestFetcherResponseTimeOut(t *testing.T) {
	defer clear()
	abc := createTestAbci(t)
	abc.fetcher.fetching[8] = &announce{
		ID:          8,
		Time:        time.Now().AddDate(0, 0, -1),
		ResendRound: 0,
	}
	abc.onNewMessage(&wire.MsgReturnInit{
		MsgID: 9,
	})
	time.Sleep(time.Millisecond * 100)
	assert.Contains(t, abc.fetcher.fetching, uint32(8))
	abc.onNewMessage(&wire.MsgReturnInit{
		MsgID: 8,
	})
	time.Sleep(time.Millisecond * 100)
	assert.NotContains(t, abc.fetcher.fetching, uint32(8))
}

func TestFetcherInject(t *testing.T) {
	defer clear()
	abc := createTestAbci(t)
	abc.fetchNewLedger()
	time.Sleep(time.Millisecond * 110)
	assert.Equal(t, 1, len(abc.fetcher.fetching))
	time.Sleep(time.Millisecond * 5000)
	abc.fetchNewLedger()
	abc.fetchNewLedger()
	assert.Equal(t, 1, len(abc.fetcher.announced))
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, 1, len(abc.fetcher.fetching))
}

func TestProposeBlock(t *testing.T) {
	defer clear()
	abc := createTestAbci(t)

	pk, sk, err := vrf.Ed25519VRF{}.GenerateKey(nil)
	if err != nil {
		t.Error(err)
	}

	b, err := abc.ProposeBlock(pk, sk)
	if err != nil {
		t.Errorf("Faild to propose block")
	}

	// Check heigth
	if b.Header.Height != 2 {
		t.Errorf("Wrong height of new block")
	}

	// Check pre-header
	if b.Header.PrevBlockHeader != genesis.GenesisBlocks[shard0].Header.BlockHeaderHash() {
		t.Errorf("Wrong preve header hash")
	}

	if len(b.Body.Transactions) != 1 {
		t.Error("Invalid number of tx is block")
	}

	if b.Header.ShardLedgerMerkleRoot == genesis.GenesisBlocks[shard0].Header.ShardLedgerMerkleRoot {
		t.Error("Invalid shard ledger merkle root in block")
	}
	// TODO: check reward address

	// Check all txs
	body := b.Body
	var tp = txprocessor.NewTxProcessor(shard0, nil)
	for _, tx := range body.Transactions {
		tp.AddTx(&tx.Tx)
	}
	r := tp.Execute(1)
	scsInBody := body.SmartContracts
	scs := r.GetSmartContracts()
	if (scs == nil) != (scsInBody == nil) {
		t.Errorf("Verify failed: smartContract should both be nil or neither ")
	}
	if len(scs) != len(scsInBody) {
		t.Errorf("Verify failed: the length of smart contract is Wrong, %d != %d", len(scs), len(body.SmartContracts))
	}
	for i := 0; i < len(scs); i++ {
		if !scs[i].Equal(scsInBody[i]) {
			t.Errorf("Verify failed: the content of smartContract is different ")
		}
	}
}

func TestVerifyBlock(t *testing.T) {
	defer clear()
	fc := txpool.NewFakeChains()
	tx1 := fc.GetNewTx(shard0)
	tx1.Tx.API = isysapi.SysAPITransfer
	block := fc.GetNewBlock(shard0, tx1, fc.GetNewTx(shard0))

	type args struct {
		block *wire.MsgBlock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{block: block},
			wantErr: true,
		},
		// TODO (wangruichao@mtv.ac): Add tests for invalid blocks.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abc := createTestAbci(t)
			err := abc.VerifyBlock(tt.args.block)
			if ((err == nil) && tt.wantErr) || ((err != nil) && (!tt.wantErr)) {
				t.Errorf("applicationBlockChain.VerifyBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify the part of smartContract
	var testForSC []struct {
		name    string
		args    args
		wantErr bool
	}

	dp := isysapi.DeployParams{
		APIList: []string{"deployContract"},
		Init:    []byte{105, 110, 105, 116},
		Code:    []byte{99, 111, 100, 101},
	}

	params, _ := rlp.EncodeToBytes(dp)
	out := wire.OutPoint{Shard: shard.Index(1), Data: wire.MtvValueToData(big.NewInt(1))}

	tx := &wire.MsgTx{
		Shard:           shard.Index(1),
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIDeploy,
		Params:          params,
		TxIn:            []*wire.TxIn{wire.NewTxIn(&out)},
	}
	txHash := tx.TxHash()
	sCAddr := txHash.FormatSmartContractAddress()
	sc := wire.SmartContract{
		ContractAddr: sCAddr,
		APIList:      dp.APIList,
		Code:         dp.Code,
	}

	body := wire.BlockBody{
		Transactions: []*wire.MsgTxWithProofs{
			{
				Tx: *tx,
			},
		},
		SmartContracts: []*wire.SmartContract{
			&sc,
		},
	}

	// There should be smart contracts, but not in the block
	bCopy1 := body
	bCopy1.SmartContracts = nil
	testForSC = append(testForSC, struct {
		name    string
		args    args
		wantErr bool
	}{
		name:    "shouldHaveSmartContractButBlockDont",
		args:    args{block: &wire.MsgBlock{Body: &bCopy1}},
		wantErr: false,
	})

	// There should be smart contracts, and there are in the block
	bCopy2 := body
	testForSC = append(testForSC, struct {
		name    string
		args    args
		wantErr bool
	}{
		name:    "asExpectedHaveSmartContract",
		args:    args{block: &wire.MsgBlock{Body: &bCopy2}},
		wantErr: true,
	})

	// There should be no smart contract here, but there is one in the block
	bCopy3 := body
	bCopy3.Transactions = nil
	testForSC = append(testForSC, struct {
		name    string
		args    args
		wantErr bool
	}{
		name:    "dontHaveSmartContractButBlockHave",
		args:    args{block: &wire.MsgBlock{Body: &bCopy3}},
		wantErr: true,
	})

	// There should be no smart contracts here, and there are none in the block
	bCopy4 := body
	bCopy4.SmartContracts = nil
	bCopy4.Transactions = nil
	testForSC = append(testForSC, struct {
		name    string
		args    args
		wantErr bool
	}{
		name:    "asExpectedDontHaveSmartContract",
		args:    args{block: &wire.MsgBlock{Body: &bCopy4}},
		wantErr: false,
	})

	for _, tt := range testForSC {
		t.Run(tt.name, func(t *testing.T) {
			abc := appBlockChain{}
			err := abc.verifySmartContract(tt.args.block.Body)
			if ((err == nil) && tt.wantErr) || ((err != nil) && (!tt.wantErr)) {
				t.Errorf("applicationBlockChain.verifySmartContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyTx(t *testing.T) {
	defer clear()
	assert := assert.New(t)
	abci := createTestAbci(t)
	abci.proposeBlockHgt = 250
	tests := []struct {
		tx     wire.MsgTx
		expect interface{}
	}{
		{
			// transfer tx with deposit out
			tx: wire.MsgTx{
				API: isysapi.SysAPITransfer,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.DepositOut,
							Data: makeDepositData(2, t),
						},
					},
				},
			},
			expect: errors.New("transfer tx's input should not be dout"),
		},
		{
			// transfer tx with withdraw out in locking
			tx: wire.MsgTx{
				API: isysapi.SysAPITransfer,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.WithdrawOut,
							Data: makeWithdrawData(100, t),
						},
					},
				},
			},
			expect: errors.New("withdraw out is in locking"),
		},
		{
			// withdraw tx with withdraw tx txin
			tx: wire.MsgTx{
				API: isysapi.SysAPIWithdraw,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.WithdrawOut,
							Data: makeWithdrawData(2, t),
						},
					},
				},
			},
			expect: errors.New("withdraw tx's input must be deposit out"),
		},
		{
			// withdraw tx with withdraw tx txin
			tx: wire.MsgTx{
				API: isysapi.SysAPIWithdraw,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.Transfer,
						},
					},
				},
			},
			expect: errors.New("withdraw tx's input must be deposit out"),
		},
		{
			// withdraw tx with deposit txin
			tx: wire.MsgTx{
				API: isysapi.SysAPIWithdraw,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.DepositOut,
							Data: makeDepositData(2, t),
						},
					},
				},
			},
			expect: nil,
		},
		{
			// binding tx with deposit out in locking
			tx: wire.MsgTx{
				API: isysapi.SysAPIBinding,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.DepositOut,
							Data: makeDepositData(240, t),
						},
					},
				},
			},
			expect: errors.New("binding tx's input is in locking"),
		},
		{
			// binding tx with deposit out
			tx: wire.MsgTx{
				API: isysapi.SysAPIBinding,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.DepositOut,
							Data: makeDepositData(220, t),
						},
					},
				},
			},
			expect: nil,
		},
		{
			// binding tx with withdraw out
			tx: wire.MsgTx{
				API: isysapi.SysAPIBinding,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.WithdrawOut,
							Data: makeWithdrawData(2, t),
						},
					},
				},
			},
			expect: errors.New("binding tx's input must be deposit out"),
		},
		{
			// binding tx with transfer out
			tx: wire.MsgTx{
				API: isysapi.SysAPIBinding,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Type: wire.Transfer,
						},
					},
				},
			},
			expect: errors.New("binding tx's input must be deposit out"),
		},
	}

	for _, v := range tests {
		txp := &wire.MsgTxWithProofs{
			Tx: v.tx,
		}
		actual := abci.verifyTx(txp)
		assert.Equal(actual, v.expect, v.expect)
	}
}

func TestGenNextReshardSeed(t *testing.T) {
	v := &vrf.Ed25519VRF{}
	pk, sk, _ := v.GenerateKey(nil)
	prevReshardSeed, _ := v.Generate(pk, sk, []byte("test"))
	reshardSeed, _ := genNextReshardSeed(pk, sk, prevReshardSeed, 10)
	prevReshardHash := calcReshardSeedHash(prevReshardSeed, 10)
	res, _ := v.VerifyProof(pk, reshardSeed, prevReshardHash[:])
	if res != true {
		t.Errorf("res: %v", res)
	}
}
