/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package depositpool

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/testutil"
)

func setUp() {
	_, _ = config.LoadConfig()
	config.GlobalConfig().DataDir = "test"
	testcase()
}
func clear() {
	os.RemoveAll("test")
}

var (
	addOutPointWithProof []*DepositInfo
	TxHash               [6]chainhash.Hash
	dataByte             [6][]byte
	depositTx            [6]wire.OutPoint
	value                = [6]int64{100, 200, 50, 250, 300, 20}
	PKHash               [6][chainhash.HashSize]byte
	testShard            = shard.IDToShardIndex(0)
	removeHashs          []chainhash.Hash
	removeOuts           []*wire.OutPoint
	removeIndex          = [6]int{0, 2, 4}
)

func testcase() {
	//         root
	//       /     \
	//      ab     cd
	//     /  \   /  \
	//    a    b c    d
	s := "885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd"
	a := merkle.MerkleHash(sha256.Sum256([]byte("a")))
	b := merkle.MerkleHash(sha256.Sum256([]byte("b")))
	c := merkle.MerkleHash(sha256.Sum256([]byte("c")))
	d := merkle.MerkleHash(sha256.Sum256([]byte("d")))
	ab := merkle.Hash(&a, &b, merkle.Right)
	cd := merkle.Hash(&c, &d, merkle.Right)
	root := merkle.Hash(ab, cd, merkle.Right)

	var merklePath [6]merkle.MerklePath
	// a's merkle path
	merklePath[0] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&a, &b, cd, root},
		ProofPath: []byte{merkle.Left, merkle.Left},
	}

	// b's merkle path
	merklePath[1] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&b, &a, cd, root},
		ProofPath: []byte{merkle.Left, merkle.Right},
	}
	// c's merkle path
	merklePath[2] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&c, &d, ab, root},
		ProofPath: []byte{merkle.Right, merkle.Left},
	}
	// d's merkle path
	merklePath[3] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&d, &c, ab, root},
		ProofPath: []byte{merkle.Right, merkle.Right},
	}
	merklePath[4] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&a, &b, cd, root},
		ProofPath: []byte{merkle.Left, merkle.Left},
	}
	merklePath[5] = merkle.MerklePath{
		Hashes:    []*merkle.MerkleHash{&a, &b, cd, root},
		ProofPath: []byte{merkle.Left, merkle.Left},
	}

	for i := 0; i < 6; i++ {
		TxHashSlice, _ := hex.DecodeString(s + strconv.Itoa(i))
		copy(TxHash[i][:], TxHashSlice[:chainhash.HashSize])

		PkHashSlice, _ := hex.DecodeString(s + strconv.Itoa(i+3))
		copy(PKHash[i][:], PkHashSlice[:chainhash.HashSize])

		mtvDepositData := wire.MTVDepositData{
			Value:          big.NewInt(value[i]),
			Height:         1,
			BindingAddress: toAddress(PKHash[i][:]),
		}

		dataByte[i] = mtvDepositData.ToData()
		var data *wire.MTVDepositData
		if err := rlp.DecodeBytes(dataByte[i], &data); err != nil || data == nil {
			fmt.Println("decode fail")
		}
		depositTx[i] = wire.OutPoint{
			Shard:           testShard,
			UserAddress:     toAddress(PKHash[i][:]),
			TxHash:          TxHash[i],
			Data:            dataByte[i],
			ContractAddress: []byte{'a'},
		}
		addOutPointWithProof = append(addOutPointWithProof, &DepositInfo{OutPoint: &depositTx[i], Proof: &merklePath[i], BlockHeight: wire.BlockHeight(i), Locked: false})
		if IsIn(i, removeIndex[:]) {
			removeHashs = append(removeHashs, TxHash[i])
			removeOuts = append(removeOuts, &depositTx[i])
		}
	}
}

func IsIn(i int, list []int) bool {
	for _, v := range list {
		if v == i {
			return true
		}
	}
	return false
}

func TestAddMoveSort(t *testing.T) {
	// init testcase
	setUp()
	defer clear()

	var blockChain iblockchain.BlockChain
	depositPool, _ := NewDepositPool(blockChain, true)
	for _, deposit := range addOutPointWithProof {
		err := depositPool.Add(deposit.OutPoint, deposit.BlockHeight, deposit.Proof)
		if err != nil {
			t.Error(err)
		}
	}

	// test add and getAll
	for i := 0; i < 6; i++ {
		addr := toAddress(PKHash[i][:])
		allDout, _ := depositPool.GetAll(testShard, addr)
		if len(allDout) != 1 {
			t.Errorf("Invalid length of out")
		}
		for _, out := range allDout {
			if !bytes.Equal(out.Out.UserAddress, addr) {
				t.Errorf("Invalid out to address")
			}
			data, err := wire.OutToDepositData(out.Out)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(data.BindingAddress, addr) {
				t.Errorf("Invalid binding address")
			}
		}
	}

	// test get
	rawData, err := depositPool.GetBiggest(toAddress(PKHash[0][:]))
	if err != nil {
		t.Error(err)
	}
	deposit := &DepositInfo{}
	err = rlp.DecodeBytes(rawData, deposit)
	if err != nil {
		t.Error(err)
	}

	// check value in out
	d, err := wire.OutToDepositData(deposit.OutPoint)
	if err != nil {
		t.Error(err)
	}
	v := new(big.Int).Sub(d.Value, new(big.Int).SetInt64(value[0])).Cmp(new(big.Int).SetInt64(0))
	if v != 0 {
		t.Error("Invalid value in out")
	}

	// test remove
	for _, o := range removeOuts {
		err := depositPool.Remove(o)
		if err != nil {
			t.Error(err)
		}
	}
	for _, v := range removeIndex {
		_, err = depositPool.GetBiggest(toAddress(PKHash[v][:]))
		if err == nil {
			t.Errorf("remove error in %d", v)
		}
	}

	// test lock
	for i := 0; i < 6; i++ {
		if !IsIn(i, removeIndex[:]) {
			err := depositPool.Lock(addOutPointWithProof[i].OutPoint)
			if err != nil {
				t.Error(err)
			}
			rawData, err = depositPool.GetBiggest(toAddress(PKHash[i][:]))
			if err != nil {
				t.Errorf("get error in %d, %v", i, err)
			}
			deposit := &DepositInfo{}
			err = rlp.DecodeBytes(rawData, deposit)
			if err != nil {
				t.Error(err)
			}
			if !deposit.Locked {
				t.Errorf("Locked error")
			}
		}
	}
}

var (
	shard0 = shard.IDToShardIndex(0)
)

func TestUpdateDepositPool(t *testing.T) {
	defer clear()
	var unUsedOuts []*wire.OutPoint
	addr := multivacaddress.GenerateAddress(genesis.GenesisPublicKeys[shard0], multivacaddress.UserAddress)
	value := new(big.Int).SetInt64(10)

	// make a fake chain
	blockChain := testutil.NewMockChain()
	fc := testutil.NewFakeChains(t, blockChain)

	// Generate a new block according to the txs
	block := fc.Next(shard0, addr, value)

	// Get unused out used to build deposit transactions
	for _, out := range block.Body.Outs {
		unUsedOuts = append(unUsedOuts, &out.OutPoint)
	}

	// Make sure the last block has been confirmed, so we
	// can use the out in it
	fc.Next(shard0, addr, value)

	// Generate a new block with deposit out
	dp := &isysapi.DepositParams{
		TransferParams: &isysapi.TransferParams{
			To:     addr,
			Shard:  shard0,
			Amount: value,
		},
		BindingAddress: addr,
	}

	// generate the block with deposit tx
	dTx := fc.GetNewTx(shard0, isysapi.SysAPIDeposit, dp, unUsedOuts...)
	blockWithDout := fc.GetNewBlock(shard0, dTx)

	// Make sure the last block has been confirmed
	fc.Next(shard0, addr, value)

	// make a deposit pool
	depositPool, err := NewDepositPool(blockChain, true)
	if err != nil {
		t.Errorf("new depositpool error")
	}

	// Find all deposit out in block and put them to deposit pool
	for _, outState := range blockWithDout.Body.Outs {
		if outState.IsDepositOut() {
			proof, err := fc.GetOutPath(shard0, &outState.OutPoint)
			if err != nil {
				t.Error(err)
			}
			err = depositPool.Add(&outState.OutPoint, wire.BlockHeight(blockWithDout.Header.Height), proof)
			if err != nil {
				t.Error(err)
			}
		}
	}

	// get state from fakechain and make an update with a block
	state, err := fc.GetState(shard0)
	if err != nil {
		t.Error(err)
	}
	blockToUpdate := fc.Next(shard0, addr, value)
	update, err := state.GetUpdateWithFullBlock(blockToUpdate)
	if err != nil {
		t.Error(err)
	}

	// use update to refresh deposit pool
	err = depositPool.Update(update, shard0, wire.BlockHeight(blockToUpdate.Header.Height))
	if err != nil {
		t.Error(err)
	}

	// check all out's proof height
	all, err := depositPool.getAllDeposit()
	if err != nil {
		t.Error(err)
	}
	for _, v := range all {
		if v.BlockHeight != wire.BlockHeight(blockToUpdate.Header.Height) {
			t.Error("invalid heigh of poorf")
		}
		err = v.Proof.Verify(&blockToUpdate.Header.ShardLedgerMerkleRoot)
		if err != nil {
			t.Error(err)
		}
	}
}

func toAddress(pk []byte) multivacaddress.Address {
	return multivacaddress.GenerateAddress(pk, multivacaddress.UserAddress)
}
