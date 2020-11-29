/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package testingutils

import (
	"fmt"
	"log"
	"time"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/miner/txprocessor"
)

var fakeReshardSeed = []byte{}

type fakeChain struct {
	ShardIndex      shard.Index
	ShardLedgerTree *merkle.FullMerkleTree
	Outs            []*wire.OutPoint
	FakeChains      *FakeChains
	LedgerInfo      *wire.LedgerInfo

	lastBlockHeaderHash *chainhash.Hash
}

// FakeChains is a fake struct for test.
type FakeChains struct {
	ShardHeight map[shard.Index]int64
	Chains      map[shard.Index]*fakeChain

	shardIndexs      []shard.Index
	blockHeaders     map[shard.IndexAndHeight]*wire.BlockHeader
	privateKeys      map[multivacaddress.PublicKeyHash]*signature.PrivateKey
	addresses        []*multivacaddress.Address
	pkHashToShardMap map[multivacaddress.PublicKeyHash]shard.Index
}

// NewFakeChains generates a fake chains for test.
func NewFakeChains() *FakeChains {
	fakeChains := FakeChains{
		shardIndexs:      shard.ShardList,
		ShardHeight:      make(map[shard.Index]int64),
		Chains:           make(map[shard.Index]*fakeChain),
		blockHeaders:     make(map[shard.IndexAndHeight]*wire.BlockHeader),
		privateKeys:      make(map[multivacaddress.PublicKeyHash]*signature.PrivateKey),
		addresses:        []*multivacaddress.Address{},
		pkHashToShardMap: make(map[multivacaddress.PublicKeyHash]shard.Index),
	}

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

	for _, tag := range shard.ShardList {
		fakeChains.ShardHeight[tag] = 1
		chain := &fakeChain{
			ShardIndex:      tag,
			ShardLedgerTree: merkle.NewFullMerkleTree(),
			FakeChains:      &fakeChains,
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

		chain.ShardLedgerTree = merkle.NewFullMerkleTree(&block.Header.OutsMerkleRoot)
		err := chain.ShardLedgerTree.HookSecondLayerTree(genesis.GenesisBlocksOutsMerkleTrees[tag])
		if err != nil {
			log.Printf("failed to hook second layer tree,err:%v", err)
		}
	}
	return &fakeChains
}

// GetNewBlock returns a new MsgBlock given a shardIndex and some MsgTxWithProofs.
func (fc *FakeChains) GetNewBlock(shardIndex shard.Index, txps ...*wire.MsgTxWithProofs) *wire.MsgBlock {
	return fc.Chains[shardIndex].GetNewBlock(txps...)
}

func (fc *fakeChain) GetNewBlock(txps ...*wire.MsgTxWithProofs) *wire.MsgBlock {
	shardIndex := fc.ShardIndex
	height := fc.FakeChains.ShardHeight[shardIndex] + 1
	for i, txp := range txps {
		if i == 0 && len(txp.Tx.TxIn) == 0 {
			// Skip reward tx which has no TxIn
			continue
		}
		err := fc.ShardLedgerTree.UpdateLeaf(
			makeUnusedOutStateHash(&txp.Tx.TxIn[0].PreviousOutPoint),
			makeUsedOutStateHash(&txp.Tx.TxIn[0].PreviousOutPoint),
		)
		if err != nil {
			fmt.Printf("failed to update leaf,err:%v", err)
		}
	}

	size := int64(0)
	ledgerInfo := wire.LedgerInfo{}

	for _, tag := range fc.FakeChains.shardIndexs {
		for i := fc.LedgerInfo.GetShardHeight(tag) + 1; i <= fc.FakeChains.ShardHeight[tag]; i++ {
			shardIndexAndHeight := shard.IndexAndHeight{
				Index:  tag,
				Height: i,
			}
			header, ok := fc.FakeChains.blockHeaders[shardIndexAndHeight]
			if !ok {
				panic(fmt.Sprintf("Can't find block header for index %v, height %d", tag, i))
			}
			emptyMerkleHash := merkle.MerkleHash{}
			if header.OutsMerkleRoot != emptyMerkleHash {
				err := fc.ShardLedgerTree.Append(&header.OutsMerkleRoot)
				if err != nil {
					log.Printf("failed to append header,err:%v", err)
				}
			}
		}
		ledgerInfo.SetShardHeight(tag, fc.FakeChains.ShardHeight[tag])
		size += fc.FakeChains.ShardHeight[tag]
	}
	ledgerInfo.Size = size

	shardLedgerMerkleRoot := *fc.ShardLedgerTree.MerkleRoot

	var tp = txprocessor.NewTxProcessor(shardIndex, nil)
	for _, tx := range txps {
		tp.AddTx(&tx.Tx)
	}
	r := tp.Execute(fc.FakeChains.ShardHeight[shardIndex] + 1)
	outsMerkleRoot := r.GetMerkleRoot()

	block := wire.NewBlock(shardIndex,
		height,
		*fc.lastBlockHeaderHash,
		shardLedgerMerkleRoot,
		*outsMerkleRoot,
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

	for _, outState := range r.GetOutStates() {
		fc.Outs = append(fc.Outs, &outState.OutPoint)
	}
	return block
}

func makeUnusedOutStateHash(outPoint *wire.OutPoint) *merkle.MerkleHash {
	outState := wire.OutState{
		OutPoint: *outPoint,
		State:    wire.StateUnused,
	}
	return merkle.ComputeMerkleHash(outState.ToBytesArray())
}

func makeUsedOutStateHash(outPoint *wire.OutPoint) *merkle.MerkleHash {
	outState := wire.OutState{
		OutPoint: *outPoint,
		State:    wire.StateUsed,
	}
	return merkle.ComputeMerkleHash(outState.ToBytesArray())
}

// GetNewTx Generate a Tx with proofs. The inputs will be deleted immediately form the fake chains.
func (fc *FakeChains) GetNewTx(shardIndex shard.Index) *wire.MsgTxWithProofs {
	return fc.Chains[shardIndex].getNewTx()
}

func (fc *fakeChain) getNewTx() *wire.MsgTxWithProofs {
	addr := fc.FakeChains.addresses[0]
	if len(fc.Outs) <= 0 {
		panic("retrieve too many transactions")
	}
	pkHash, err := addr.GetPublicKeyHash(multivacaddress.UserAddress)
	if err != nil {
		panic(err)
	}
	outToSpend := fc.Outs[0]
	fc.Outs = fc.Outs[1:len(fc.Outs)]
	path, err := fc.ShardLedgerTree.GetMerklePath(
		makeUnusedOutStateHash(outToSpend))
	if err != nil {
		panic(err)
	}
	if err := path.Verify(fc.ShardLedgerTree.MerkleRoot); err != nil {
		panic(err)
	}
	value := wire.GetMtvValueFromOut(outToSpend)
	params, _ := rlp.EncodeToBytes(isysapi.TransferParams{
		To:     *addr,
		Shard:  fc.FakeChains.pkHashToShardMap[*pkHash],
		Amount: value,
	})
	outToSpendPKHash, err := outToSpend.UserAddress.GetPublicKeyHash(multivacaddress.UserAddress)
	if err != nil {
		panic(err)
	}
	tx := wire.MsgTx{
		Shard: fc.FakeChains.pkHashToShardMap[*outToSpendPKHash],
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: *outToSpend,
			},
		},
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPITransfer,
		Params:          params,
	}
	privateKey, ok := fc.FakeChains.privateKeys[*outToSpendPKHash]
	if !ok {
		panic("Can't find private key for out")
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
