/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package testutil

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/reducekey"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/miner/txprocessor"
	"github.com/multivactech/MultiVAC/processor/shared/smartcontractdatastore"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

var fakeReshardSeed = []byte{}

type fakeChain struct {
	ShardIndex             shard.Index
	ShardLedgerTree        *merkle.FullMerkleTree
	Outs                   []*wire.OutPoint
	FakeChains             *FakeChains
	LedgerInfo             *wire.LedgerInfo
	lastBlockHeaderHash    *chainhash.Hash
	smartContractDataStore smartcontractdatastore.SmartContractDataStore
	state                  *state.ShardStateManager
	t                      *testing.T
}

// FakeChains is a fake abci used for testing
type FakeChains struct {
	ShardHeight      map[shard.Index]int64
	Chains           map[shard.Index]*fakeChain
	blockHeaders     map[shard.IndexAndHeight]*wire.BlockHeader
	privateKeys      map[multivacaddress.PublicKeyHash]*signature.PrivateKey
	pkHashToShardMap map[multivacaddress.PublicKeyHash]shard.Index
	addresses        []*multivacaddress.Address
	shardIndexs      []shard.Index
	blockchain       iblockchain.BlockChain
	t                *testing.T
}

// NewFakeChains returns the new instance of fake abci
func NewFakeChains(t *testing.T, blockchain iblockchain.BlockChain) *FakeChains {
	fakeChains := FakeChains{
		shardIndexs:      shard.ShardList,
		ShardHeight:      make(map[shard.Index]int64),
		Chains:           make(map[shard.Index]*fakeChain),
		blockHeaders:     make(map[shard.IndexAndHeight]*wire.BlockHeader),
		privateKeys:      make(map[multivacaddress.PublicKeyHash]*signature.PrivateKey),
		addresses:        []*multivacaddress.Address{},
		pkHashToShardMap: make(map[multivacaddress.PublicKeyHash]shard.Index),
		blockchain:       blockchain,
		t:                t,
	}
	fakeChains.blockchain.ReceiveBlock(genesis.GenesisBlocks[0])
	fakeChains.blockchain.ReceiveBlock(genesis.GenesisBlocks[1])
	fakeChains.blockchain.ReceiveBlock(genesis.GenesisBlocks[2])
	fakeChains.blockchain.ReceiveBlock(genesis.GenesisBlocks[3])

	// Init address and it's key
	for i := 0; i < params.GenesisNumberOfShards; i++ {
		shard := shard.ShardList[i]
		pubKey := genesis.GenesisPublicKeys[shard]
		gAddr := multivacaddress.GenerateAddress(pubKey, multivacaddress.UserAddress)
		gPKHash, err := gAddr.GetPublicKeyHash(multivacaddress.UserAddress)
		if err != nil {
			panic(err)
		}
		fakeChains.addresses = append(fakeChains.addresses, &gAddr)
		fakeChains.pkHashToShardMap[*gPKHash] = shard
		privKey := genesis.GenesisPrivateKeys[shard]
		fakeChains.privateKeys[*gPKHash] = &privKey
	}

	// init subchain
	for _, tag := range shard.ShardList {
		fakeChains.ShardHeight[tag] = 1
		chain := &fakeChain{
			ShardIndex:             tag,
			ShardLedgerTree:        merkle.NewFullMerkleTree(),
			FakeChains:             &fakeChains,
			smartContractDataStore: smartcontractdatastore.NewSmartContractDataStore(tag),
			t:                      t,
		}

		fakeChains.Chains[tag] = chain
		block := genesis.GenesisBlocks[tag]
		chain.LedgerInfo = &block.Body.LedgerInfo
		shardIndexAndHeight := shard.IndexAndHeight{
			Index:  tag,
			Height: 1,
		}
		fakeChains.blockHeaders[shardIndexAndHeight] = &block.Header

		chain.Outs = []*wire.OutPoint{}
		txHash := genesis.GenerateGenesisTx(tag).TxHash()
		for i := 1; i < genesis.NumberOfOutsInEachGenesisBlock; i++ {
			outState := genesis.CreateGenesisOutState(tag, i, &txHash)
			chain.Outs = append(chain.Outs, &outState.OutPoint)
		}

		lastBlockHeaderHash := block.Header.BlockHeaderHash()
		chain.lastBlockHeaderHash = &lastBlockHeaderHash

		// init shard ledger tree
		chain.ShardLedgerTree = merkle.NewFullMerkleTree(&block.Header.OutsMerkleRoot)
		err := chain.ShardLedgerTree.HookSecondLayerTree(genesis.GenesisBlocksOutsMerkleTrees[tag])
		if err != nil {
			log.Printf("failed to hook second layer tree,err:%v", err)
		}
	}

	// Append other shard's block merkle tree to the ledger tree
	for _, v := range fakeChains.Chains {
		lastPath, _ := v.ShardLedgerTree.GetLastNodeMerklePath()
		state := state.NewShardStateManager(v.ShardLedgerTree.MerkleRoot,
			v.ShardLedgerTree.Size,
			lastPath, v.LedgerInfo, blockchain)

		v.state = state
	}

	return &fakeChains
}

// GetNewBlock returns a new block of a shard
func (fc *FakeChains) GetNewBlock(shardIndex shard.Index, txps ...*wire.MsgTxWithProofs) *wire.MsgBlock {
	return fc.Chains[shardIndex].GetNewBlock(txps...)
}

// Next used to gererate a new block on the current chain
// It usually used to confirm the last block
func (fc *FakeChains) Next(shard shard.Index, to multivacaddress.Address, amount *big.Int) *wire.MsgBlock {
	tp := &isysapi.TransferParams{
		To:     to,
		Shard:  shard,
		Amount: amount,
	}
	return fc.GetNewBlock(shard, fc.GetNewTx(shard, isysapi.SysAPITransfer, tp))
}

func (fc *fakeChain) GetNewBlock(txps ...*wire.MsgTxWithProofs) *wire.MsgBlock {

	// For testing, block must have tx
	if len(txps) <= 0 {
		fc.t.Error("Must have tx in block")
		return nil
	}
	shardIndex := fc.ShardIndex
	height := fc.FakeChains.ShardHeight[shardIndex] + 1

	// Mark out to spend status
	for i, txp := range txps {
		if i == 0 && len(txp.Tx.TxIn) == 0 {
			// Skip reward tx which has no TxIn.
			continue
		}
		err := fc.ShardLedgerTree.UpdateLeaf(
			makeUnusedOutStateHash(&txp.Tx.TxIn[0].PreviousOutPoint),
			makeUsedOutStateHash(&txp.Tx.TxIn[0].PreviousOutPoint),
		)
		if err != nil {
			fc.t.Error(err)
		}
	}

	size := int64(0)
	ledgerInfo := wire.LedgerInfo{}

	for _, tag := range fc.FakeChains.shardIndexs {
	flag:
		for i := fc.LedgerInfo.GetShardHeight(tag) + 1; i <= fc.FakeChains.ShardHeight[tag]; i++ {
			shardIndexAndHeight := shard.IndexAndHeight{
				Index:  tag,
				Height: i,
			}
			header, ok := fc.FakeChains.blockHeaders[shardIndexAndHeight]
			if !ok {
				panic(fmt.Sprintf("Can't find block header for index %v, height %d", tag, i))
			}

			var reduceTxInHeader []*wire.MsgTxWithProofs
			for _, reduceAction := range header.ReduceActions {
				reduceOut := reduceAction.ToBeReducedOut
				if reduceOut.Shard != tag {
					continue
				}
				contractAddrHash, err := reduceOut.ContractAddress.GetPublicKeyHash(multivacaddress.SmartContractAddress)
				if err != nil {
					continue flag
				}
				sCInfo := fc.smartContractDataStore.GetSmartContractInfo(*contractAddrHash)
				if sCInfo == nil {
					continue flag
				}
				tx := &wire.MsgTx{
					Version:         wire.TxVersion,
					Shard:           tag,
					TxIn:            []*wire.TxIn{{PreviousOutPoint: *reduceOut}},
					ContractAddress: reduceOut.ContractAddress,
					API:             reduceAction.API,
				}
				tx.Sign(&reducekey.ReducePrivateKey)
				txWithProof := &wire.MsgTxWithProofs{
					Tx:     *tx,
					Proofs: []merkle.MerklePath{},
				}
				reduceTxInHeader = append(reduceTxInHeader, txWithProof)
			}

			txps = append(txps, reduceTxInHeader...)
			// tree
			b := fc.FakeChains.blockchain.GetShardsBlockByHeight(shardIndexAndHeight.Index, wire.BlockHeight(shardIndexAndHeight.Height))
			if b == nil {
				panic(fmt.Sprintf("can not get bolck in fakechain, indexAndHeight:%v", shardIndexAndHeight))
			}
			emptyMerkleHash := merkle.MerkleHash{}
			if b.Header.OutsMerkleRoot != emptyMerkleHash {
				err := fc.ShardLedgerTree.Append(&b.Header.OutsMerkleRoot)
				if err != nil {
					fc.t.Error(err)
				}
			}

			var outHashes []*merkle.MerkleHash
			for _, outState := range b.Body.Outs {
				outStateHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
				outHashes = append(outHashes, outStateHash)
			}

			fullMerkleTree := merkle.NewFullMerkleTree(outHashes...)
			err := fc.ShardLedgerTree.HookSecondLayerTree(fullMerkleTree)
			if err != nil {
				fc.t.Error(err)
			}

		}
		ledgerInfo.SetShardHeight(tag, fc.FakeChains.ShardHeight[tag])
		size += fc.FakeChains.ShardHeight[tag]
	}
	ledgerInfo.Size = size
	shardLedgerMerkleRoot := *fc.ShardLedgerTree.MerkleRoot

	// Execute tx
	tp := txprocessor.NewTxProcessor(shardIndex, nil)
	for _, tx := range txps {
		tp.AddTx(&tx.Tx)
	}
	r := tp.Execute(fc.FakeChains.ShardHeight[shardIndex] + 1)
	if len(r.GetOutStates()) == 0 {
		fc.t.Error("no successful tx")
	}

	// Make a new block
	block := wire.NewBlock(shardIndex,
		height,
		*fc.lastBlockHeaderHash,
		shardLedgerMerkleRoot,
		*r.GetMerkleRoot(),
		txps,
		&ledgerInfo,
		false,
		time.Now().Unix(), fakeReshardSeed, r.GetSmartContracts(), r.GetOutStates(), r.GetUpdateActions(),
		r.GetReduceActions(),
	)

	hash := block.Header.BlockHeaderHash()
	fc.lastBlockHeaderHash = &hash
	fc.LedgerInfo = &ledgerInfo
	fc.FakeChains.ShardHeight[shardIndex] = height

	shardIndexAndHeight := shard.IndexAndHeight{
		Index:  shardIndex,
		Height: height,
	}
	fc.FakeChains.blockHeaders[shardIndexAndHeight] = &block.Header

	// update state
	fc.FakeChains.blockchain.ReceiveBlock(block)
	update, err := fc.state.GetUpdateWithFullBlock(block)
	if err != nil {
		fc.t.Error(err)
	}

	err = fc.state.ApplyUpdate(update)
	if err != nil {
		fc.t.Error(err)
	}

	return block
}

func makeUnusedOutStateHash(outPoint *wire.OutPoint) *merkle.MerkleHash {
	outState := wire.OutState{
		OutPoint: *outPoint,
		State:    wire.StateUnused,
	}
	return makeMerkleHash(outState.ToBytesArray())
}

func makeUsedOutStateHash(outPoint *wire.OutPoint) *merkle.MerkleHash {
	outState := wire.OutState{
		OutPoint: *outPoint,
		State:    wire.StateUsed,
	}
	return makeMerkleHash(outState.ToBytesArray())
}

// GetNewTx Generate a Tx with proofs. The inputs will be deleted immediately form the fake chains.
func (fc *FakeChains) GetNewTx(shardIndex shard.Index, txType string, params interface{}, out ...*wire.OutPoint) *wire.MsgTxWithProofs {
	return fc.Chains[shardIndex].getNewTx(txType, params, out...)
}

// todo:it seems unused.
//func (fc *FakeChains) getState(shard shard.Index) (state.ShardStateManager, error) {
//	state := state.NewShardStateManager(fc.Chains[shard].ShardLedgerTree.MerkleRoot,
//		fc.Chains[shard].ShardLedgerTree.Size,
//		fc.GetLastPath(shard), fc.GetLedgerInfo(shard), fc.blockchain)
//	return *state, nil
//}

// GetState get update from fakechain
func (fc *FakeChains) GetState(shard shard.Index) (state.ShardStateManager, error) {
	//return fc.Chains[shard].state, nil
	state := state.NewShardStateManager(fc.Chains[shard].ShardLedgerTree.MerkleRoot,
		fc.Chains[shard].ShardLedgerTree.Size,
		fc.GetLastPath(shard), fc.GetLedgerInfo(shard), fc.blockchain)
	return *state, nil
}

// GetOutPath get an out's merkle path
func (fc *FakeChains) GetOutPath(shard shard.Index, out *wire.OutPoint) (*merkle.MerklePath, error) {
	f := fc.Chains[shard]
	return f.ShardLedgerTree.GetMerklePath(makeUnusedOutStateHash(out))
}

func (fc *fakeChain) getNewTx(txType string, params interface{}, out ...*wire.OutPoint) *wire.MsgTxWithProofs {

	if len(out) != 0 {
		if len(fc.Outs) <= 0 {
			fc.t.Error("retrieve too many transactions")
			return nil
		}
	}

	// Use genesis out or external out
	var outToSpend *wire.OutPoint
	if len(out) <= 0 {
		outToSpend = fc.Outs[0]
		fc.Outs = fc.Outs[1:len(fc.Outs)]
	} else {
		outToSpend = out[0]
	}

	// Get the path of out
	path, err := fc.ShardLedgerTree.GetMerklePath(makeUnusedOutStateHash(outToSpend))
	if err != nil {
		fc.t.Error(err)
		return nil
	}
	if err := path.Verify(fc.ShardLedgerTree.MerkleRoot); err != nil {
		fc.t.Error(err)
		return nil
	}
	outToSpendPKHash, err := outToSpend.UserAddress.GetPublicKeyHash(multivacaddress.UserAddress)
	if err != nil {
		fc.t.Error(err)
		return nil
	}

	// Generate tx
	tx := wire.MsgTx{
		Shard: fc.FakeChains.pkHashToShardMap[*outToSpendPKHash],
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: *outToSpend,
			},
		},
		ContractAddress: isysapi.SysAPIAddress,
	}

	switch txType {
	case isysapi.SysAPITransfer:
		tp, ok := params.(*isysapi.TransferParams)
		if !ok {
			fc.t.Errorf("Type assert error for %v", txType)
			return nil
		}
		data, err := rlp.EncodeToBytes(tp)
		if err != nil {
			fc.t.Error(err)
			return nil
		}
		tx.Params = data
		tx.API = isysapi.SysAPITransfer

	case isysapi.SysAPIDeposit:
		dp, ok := params.(*isysapi.DepositParams)
		if !ok {
			fc.t.Errorf("Type assert error for %v", txType)
			fmt.Println(params)
			return nil
		}
		data, err := rlp.EncodeToBytes(dp)
		if err != nil {
			fc.t.Error(err)
			return nil
		}
		tx.Params = data
		tx.API = isysapi.SysAPIDeposit

	case isysapi.SysAPIWithdraw:
		wp, ok := params.(*isysapi.WithdrawParams)
		if !ok {
			fc.t.Errorf("Type assert error for %v", txType)
			return nil
		}
		data, err := rlp.EncodeToBytes(wp)
		if err != nil {
			fc.t.Error(err)
			return nil
		}
		tx.Params = data
		tx.API = isysapi.SysAPIWithdraw
	case isysapi.SysAPIBinding:
		bp, ok := params.(*isysapi.BindingParams)
		if !ok {
			fc.t.Errorf("Type assert error for %v", txType)
			return nil
		}
		data, err := rlp.EncodeToBytes(bp)
		if err != nil {
			fc.t.Error(err)
			return nil
		}
		tx.Params = data
		tx.API = isysapi.SysAPIWithdraw
	default:
	}

	privateKey, ok := fc.FakeChains.privateKeys[*outToSpendPKHash]
	if !ok {
		fc.t.Error("Can't find private key for out")
		return nil
	}
	tx.Sign(privateKey)
	txWithProofs := wire.MsgTxWithProofs{
		Tx: tx,
		Proofs: []merkle.MerklePath{
			*path,
		},
	}
	return &txWithProofs
}

// GetLastPath Returns the last path of the ledger tree for given shard.
func (fc *FakeChains) GetLastPath(shardIndex shard.Index) *merkle.MerklePath {
	path, err := fc.Chains[shardIndex].ShardLedgerTree.GetLastNodeMerklePath()
	if err != nil {
		panic("FakeChains fails to get last path.")
	}
	return path
}

// GetLedgerInfo Returns a LedgerInfo struct contains the current height of each shard and total size.
func (fc *FakeChains) GetLedgerInfo(index shard.Index) *wire.LedgerInfo {
	return fc.Chains[index].LedgerInfo
}

// todo:it seems unused.
//func convertToPkHash(pk *signature.PublicKey) *chainhash.Hash {
//	pkBytes := []byte(*pk)
//	var pkHashBytes [chainhash.HashSize]byte
//	copy(pkHashBytes[:], pkBytes[:])
//	pkHash := chainhash.Hash(pkHashBytes)
//	return &pkHash
//}

func makeMerkleHash(content []byte) *merkle.MerkleHash {
	hash := merkle.MerkleHash(sha256.Sum256(content))
	return &hash
}
