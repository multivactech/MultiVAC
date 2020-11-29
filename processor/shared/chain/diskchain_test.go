/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

const (
	TestDir = "test"
)

var (
	chain     *diskBlockChain
	TestShard = shard.IDToShardIndex(1)
)

func fakeBlock(shard shard.Index, hgt wire.BlockHeight) *wire.MsgBlock {
	header := wire.BlockHeader{ShardIndex: shard, Height: int64(hgt)}
	return &wire.MsgBlock{
		Header: header,
		Body:   &wire.BlockBody{},
	}
}

// Set up to do some initialization work
func setup() {
	_, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	config.GlobalConfig().DataDir = TestDir
	chain = newDiskBlockChain()
}

// For disk chain, remove the test folder
func clear() {
	os.RemoveAll(TestDir)
}

func TestReceiveBlock(t *testing.T) {
	setup()
	defer clear()

	// Make a fake block
	fBlock := fakeBlock(TestShard, 2)
	ok := chain.ReceiveBlock(fBlock)
	if !ok {
		t.Errorf("Chain faild to received block: %v", fBlock)
	}
	// Check if the block is received successfully.
	if !chain.containsShardsBlock(TestShard, wire.BlockHeight(2)) {
		t.Errorf("Wrong")
	}
	// Check receive the same block.
	ok = chain.ReceiveBlock(fBlock)
	if !ok {
		t.Errorf("Chain faild to received the same block: %v", fBlock)
	}
}

func TestGetInfo(t *testing.T) {
	setup()
	defer clear()

	// Make a fake block.
	fBlock := fakeBlock(TestShard, 2)
	ok := chain.ReceiveBlock(fBlock)
	if !ok {
		t.Errorf("Chain faild to received block: %v", fBlock)
	}
	// Test get block by hash.
	b := chain.GetShardsBlockByHash(fBlock.Header.BlockHeaderHash())
	if b == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if b.Header.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Block match faild (by hash), (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), b.Header.BlockHeaderHash())
	}

	// Test get block by height.
	b = chain.GetShardsBlockByHeight(TestShard, 2)
	if b == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if b.Header.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Block match faild (by height), (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), b.Header.BlockHeaderHash())
	}

	// Test get header by hash.
	h := chain.GetShardsHeaderByHash(fBlock.Header.BlockHeaderHash())
	if h == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if h.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Header match faild (by hash), (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), h.BlockHeaderHash())
	}

	// Test get header by hash.
	h = chain.GetShardsHeaderByHeight(TestShard, 2)
	if h == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if h.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Header match faild (by height), (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), h.BlockHeaderHash())
	}

	// Test get header hashes.
	from := wire.BlockHeight(1)
	end := wire.BlockHeight(2)
	hashes := chain.GetShardsHeaderHashes(TestShard, from, end)
	if len(hashes) != int(end-from+1) {
		t.Errorf("Wrong lenght of result, (expect): %d, (actual): %d", end-from+1, len(hashes))
	}

	// Test get shard height
	height := chain.GetShardsHeight(TestShard)
	if height != 2 {
		t.Errorf("Height match error, (expect): 2, (actual): %d", height)
	}
}

func TestReceiveHeader(t *testing.T) {
	setup()
	defer clear()

	// Make a fake block.
	fBlock := fakeBlock(TestShard, 2)
	ok := chain.ReceiveHeader(&fBlock.Header)
	if !ok {
		t.Errorf("Chain faild to received header: %v", fBlock.Header)
	}

	// Test get header by height
	h := chain.GetShardsHeaderByHeight(TestShard, 2)
	if h == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if h.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Header match faild (by height) (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), h.BlockHeaderHash())
	}

	// Test get header by hash.
	h = chain.GetShardsHeaderByHash(fBlock.Header.BlockHeaderHash())
	if h == nil {
		t.Errorf("Faild find block in chain.")
		return
	}
	if h.BlockHeaderHash() != fBlock.Header.BlockHeaderHash() {
		t.Errorf("Header match faild (by hash), (expect): %v, (actual): %v", fBlock.Header.BlockHeaderHash(), h.BlockHeaderHash())
	}

	// TODO(zz): If receive header first, then can't received the corresponding block.
	// Is this a bug or our design?
	chain.ReceiveBlock(fBlock)
}

// TODO: complete testSyncTrigger when test sync
// Used for function(TestSync)
// type testSyncTrigger struct {
// 	count int
// }

// func (st *testSyncTrigger) MaybeSync() {
// 	st.count++
// }

// TODO: fix for testnet-3.0
// func TestSync(t *testing.T) {
// 	setup()
// 	defer clear()

// 	// Make a fake syncTrigger
// 	trigger := new(testSyncTrigger)
// 	// Register syncTrigger
// 	chain.SetSyncTrigger(TestShard, trigger)

// 	// Receive a fake block
// 	fBlock := fakeBlock(TestShard, 2)
// 	ok := chain.ReceiveBlock(fBlock)
// 	if !ok {
// 		t.Errorf("Faild to receive block, block height is %d", fBlock.Header.Height)
// 	}
// 	// Receive a block with heigh 4, this may trigger sync.
// 	fBlock2 := fakeBlock(TestShard, 4)
// 	ok = chain.ReceiveBlock(fBlock2)
// 	if !ok {
// 		t.Errorf("Faild to receive block, block height is %d", fBlock2.Header.Height)
// 	}

// 	if trigger.count != 1 {
// 		t.Error("Faild to trigger sync.")
// 	}

// 	// Receive a block that maybe from sync, this will not trigger sync.
// 	fBlock3 := fakeBlock(TestShard, 3)
// 	ok = chain.ReceiveBlock(fBlock3)
// 	if !ok {
// 		t.Errorf("Faild to receive block, block height is %d", fBlock3.Header.Height)
// 	}

// 	if trigger.count != 1 {
// 		t.Error("Invalid count for trigger")
// 	}
// }
func TestGetSmartContract(t *testing.T) {
	setup()
	defer clear()

	// Make a fake smart contracts
	txHash := chainhash.Hash{}
	err := txHash.SetBytes([]byte("testContractAddr"))
	if err != nil {
		fmt.Println("Faild to set bytes: " + err.Error())
	}

	scs := []*wire.SmartContract{
		{
			ContractAddr: txHash.FormatSmartContractAddress(),
			APIList:      []string{"exe", "do"},
			Code:         []byte("public static void main"),
		},
	}

	ok := chain.receiveSmartContracts(scs)
	if !ok {
		t.Errorf("Chain faild to received samrt contracts: %v", scs)
	}

	sc := chain.GetSmartContract(txHash.FormatSmartContractAddress())
	// Check if the smartContract is received correct.
	if sc == nil {
		t.Errorf("Chain faild to get smart contract: %v", sc)
	}

	if !reflect.DeepEqual(sc.APIList, []string{"exe", "do"}) {
		t.Errorf("Chain save wrong smart contract APIList, want %v\n get: %v", []string{"exe", "do"}, sc.APIList)
	}

	if !reflect.DeepEqual(sc.Code, []byte("public static void main")) {
		t.Errorf("Chain save wrong smart contract Code, want %v\n get: %v", []byte("public static void main"), sc.Code)
	}
}

func TestGetSmartContractOuts(t *testing.T) {
	setup()
	defer clear()

	txHash := chainhash.Hash{}
	err := txHash.SetBytes([]byte("testContractAddr"))
	if err != nil {
		fmt.Println("Faild to set bytes: " + err.Error())
	}

	codeData := []byte("public static void main")
	initData := []byte("init")

	contractAddr := multivacaddress.GenerateAddress(signature.PublicKey(txHash.CloneBytes()), multivacaddress.SmartContractAddress)
	// Make a fake smart contract outs
	out1 := &wire.OutPoint{
		Shard:           TestShard,
		TxHash:          txHash,
		Index:           0,
		Data:            codeData,
		ContractAddress: contractAddr,
	}

	out2 := &wire.OutPoint{
		Shard:           TestShard,
		TxHash:          txHash,
		Index:           1,
		Data:            initData,
		ContractAddress: contractAddr,
	}

	err = chain.saveSmartContractCodeOut(out1.ContractAddress, out1.Shard, out1.ToUnspentOutState())
	if err != nil {
		t.Errorf("Chain faild to received smart contract code out: %v", out1)
	}

	err = chain.saveSmartContractShardInitOut(out2.ContractAddress, out2.Shard, out2.ToUnspentOutState())
	if err != nil {
		t.Errorf("Chain faild to received smart contract shard init out: %v", out2)
	}

	codeOut := chain.getSmartContractCodeOut(contractAddr, out1.Shard)
	shardDataOut := chain.getSmartContractShardInitOut(contractAddr, out2.Shard)

	// Check if the smartContract outs is received correct.
	if codeOut == nil || shardDataOut == nil {
		t.Errorf("Chain faild to get smart contract out: %v\n%v", codeOut, shardDataOut)
	}

	// 判断存放了合约代码的out数据是否正确
	if codeOut.Index != 0 {
		t.Errorf("Chain faild to get smart contract data out According to the order: %v", codeOut)
	}

	if !reflect.DeepEqual(codeOut.Data, codeData) {
		t.Errorf("Chain get WRONG smart contract data out, want: %v\nget: %v", codeData, codeOut.Data)
	}

	// 判断存放了init的out数据是否正确
	if shardDataOut.Index != 1 {
		t.Errorf("Chain faild to get smart contract data out According to the order: %v", shardDataOut)
	}

	if !reflect.DeepEqual(shardDataOut.Data, initData) {
		t.Errorf("Chain get WRONG smart contract data out, want: %v\nget: %v", initData, shardDataOut.Data)
	}

}
