/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/istate"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/reducekey"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/processor/miner/txprocessor"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/smartcontractdatastore"
	"github.com/multivactech/MultiVAC/processor/shared/state"
	"github.com/multivactech/MultiVAC/processor/shared/txpool"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	maxTransactionsInBlock = 5000

	// If the number of transactions in tx pool is smaller than this number, it will
	// trys to fetch more transactions.
	// TODO: Before launching, change this to be around 1x number of transactions in
	// a block.
	minTransactionsInTxPool = 5000

	// When fetching transactions, it will try to reach this max number of transactions.
	// TODO: Before launching, change this to be around 2x number of transactions in
	// a block.
	maxTransactionsInTxPool = 10000
)

type appBlockChain struct {
	shardIndex             shard.Index
	shardHeight            int64
	proposeBlockHgt        int64                                         // the height of propose block
	txPool                 txpool.TxPool                                 // txpool used to manage transactions
	state                  istate.LedgerStateManager                     // state used to maintain ledger
	blockChain             iblockchain.BlockChain                        // blockchain used to save block and header
	dPool                  idepositpool.DepositPool                      // dPool used to manage deposti transaction
	smartContractDataStore smartcontractdatastore.SmartContractDataStore // smartContractDataPool that miner maintained
	fetcher                *abciFetcher                                  // fetcher used to fetch message from storagenode

	// Message
	gossipNode connection.GossipNode  // register message handle channels and send broadcast messages
	pubSubMgr  *message.PubSubManager // Inter-module communication

	// Cache
	prevReshardHeader wire.BlockHeader // Last resharded header
	prevHeader        wire.BlockHeader
	prevBlock         *wire.MsgBlock

	txFetchTicker *time.Ticker
	isFetchingTx  bool
	log           btclog.Logger
}

func newApplicationBlockChain(shardIdx shard.Index, blockchain iblockchain.BlockChain,
	dPool idepositpool.DepositPool, pubSubMgr *message.PubSubManager) *appBlockChain {
	abci := &appBlockChain{
		shardIndex:    shardIdx,
		blockChain:    blockchain,
		gossipNode:    connection.GlobalConnServer,
		dPool:         dPool,
		pubSubMgr:     pubSubMgr,
		txFetchTicker: time.NewTicker(fetchTransactionsInterval),
		fetcher:       newABCIFetcher(),
		log:           logBackend.Logger(fmt.Sprintf("ABCI%d", shardIdx.GetID())),
	}

	// Subscribe to related events
	pubSubMgr.Sub(abci)
	abci.fetcher.start()

	// log config
	abci.log.SetLevel(logger.AbciLogLevel)

	return abci
}

func (abc *appBlockChain) setPrevReshardHeader(header *wire.BlockHeader) {
	abc.log.Debugf("update reshard header at height of %d", header.Height)
	abc.prevReshardHeader = *header
}

func (abc *appBlockChain) broadcastMsg(msg wire.Message) {
	params := new(connection.BroadcastParams)
	params.ToNode = connection.ToStorageNode

	switch message := msg.(type) {
	case *wire.SlimBlock:
		params.ToShard = message.ToShard
	case *wire.MsgFetchInit, *wire.MsgFetchSmartContractInfo, *wire.MsgFetchTxs:
		params.ToShard = abc.shardIndex
	}
	abc.gossipNode.BroadcastMessage(msg, params)
}

func (abc *appBlockChain) getAddress() multivacaddress.Address {
	pk := config.GlobalConfig().Pk
	address, _ := multivacaddress.StringToUserAddress(pk)
	return address
}

func (abc *appBlockChain) verifyBlockHeader(header *wire.BlockHeader) error {
	if header.Version != wire.BlockVersion {
		return invalidBlockError("Header.Version", header.Version, wire.BlockVersion)
	}
	if header.ShardIndex != abc.shardIndex {
		return invalidBlockError("Header.Index", header.ShardIndex, abc.shardIndex)
	}
	if header.Height != abc.proposeBlockHgt {
		return invalidBlockError("Header.Height", header.Height, abc.proposeBlockHgt)
	}
	prevHash := abc.prevHeader.BlockHeaderHash()
	if !header.PrevBlockHeader.IsEqual(&prevHash) {
		return invalidBlockError(
			"Header.PrevBlockHeader",
			header.PrevBlockHeader,
			prevHash)
	}
	if !header.IsEmptyBlock {
		// verify ReshardSeed
		err := abc.verifyReshardSeed(header)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO:(tianhongce) Verify that the new ReshardSeed is legal
func (abc *appBlockChain) verifyReshardSeed(header *wire.BlockHeader) error {
	if !bytes.Equal(header.ReshardSeed, abc.prevReshardHeader.ReshardSeed) {
		// If the seed is updated, verify if it should be updated.
		if (header.Height - abc.prevReshardHeader.Height) < params.ReshardRoundThreshold {
			abc.log.Debugf("header.height %v, prevReshardHeader.Height %v", header.Height, abc.prevReshardHeader.Height)
			return errors.New("verifyblockheader: should not update reshard seed, but update")
		}
		reshardHash := calcReshardSeedHash(abc.prevReshardHeader.ReshardSeed, header.Height)

		// If the seed is updated, verify that it is updated correctly.
		res, err := vrf.Ed25519VRF{}.VerifyProof(header.Pk, header.ReshardSeed, reshardHash[:])
		if err != nil {
			return err
		}
		if !res {
			return errors.New("verifyblockheader: can't verify the new reshard seed")
		}
	} else {
		// If the seed is not updated, verify that it should not be updated.
		if ((header.Height - abc.prevReshardHeader.Height) >= params.ReshardRoundThreshold) && !header.IsEmptyBlock {
			return errors.New("verifyblockheader: should update reshard seed, but not")
		}
	}
	return nil
}

// Verifies the block body is valid.
// 1. All transactions with proofs in this block body is valid, including reward tx.
// 2. No duplicated TxIn across all transactions.
// 3. Outs merkle root is correct.
//
// Note that txprocessor is responsible for verifying the internal logic of the tx.
// TODO(issues#168): Revisit this, maybe move more verification logic into txprocessor.
func (abc *appBlockChain) verifyBlockBodyInternal(body *wire.BlockBody, ledgerMerkleRoot *merkle.MerkleHash,
	outsMerkleRoot *merkle.MerkleHash,
	shardIndex shard.Index) error {

	if len(body.Transactions) > 0 {
		var normalTxs []*wire.MsgTxWithProofs

		// If enable coinbase, the first index of tx is coinbase tx
		if params.EnableCoinbaseTx {
			normalTxs = body.Transactions[1:]
		} else {
			normalTxs = body.Transactions
		}

		// Verify all normal transactions are valid.
		for index, tx := range normalTxs {
			if !tx.Tx.IsTxInShard(shardIndex) {
				return fmt.Errorf("the %d th transaction not in shard %v", index, shardIndex)
			}

			// Reduce tx and storage reward tx don't need verify proof.
			if (!params.EnableStorageReward || tx.Tx.API != isysapi.SysAPIReward) && !tx.Tx.IsReduceTx() {
				if err := tx.VerifyTxWithProof(ledgerMerkleRoot); err != nil {
					return err
				}
			}

			// Verify tx interval
			err := abc.verifyTx(tx)
			if err != nil {
				return err
			}
		}

		// Verify there's no outs used more than once in all transactions in this block body.
		outsMap := make(map[merkle.MerkleHash]bool)
		for _, tx := range normalTxs {
			for _, txIn := range tx.Tx.TxIn {
				key := merkle.ComputeMerkleHash(
					txIn.PreviousOutPoint.ToUnspentOutState().ToBytesArray())
				_, ok := outsMap[*key]
				if ok {
					return fmt.Errorf("verify block body: Transactions have duplicate txin")
				}
				outsMap[*key] = true
			}
		}
	}

	// Generate txOut.
	tp := txprocessor.NewTxProcessor(abc.shardIndex, abc.smartContractDataStore)
	for _, tx := range body.Transactions {
		tp.AddTx(&tx.Tx)
	}
	r := tp.Execute(abc.proposeBlockHgt)
	expectedMerkleRoot := r.GetMerkleRoot()

	// Check the root of block-merkle-tree is correct.
	if *outsMerkleRoot != *expectedMerkleRoot {
		return fmt.Errorf(
			"wrong merkle root, expecting %v, actual %v",
			*expectedMerkleRoot,
			*outsMerkleRoot)
	}

	// Verify the results of executing smart contracts.
	updateActions := r.GetUpdateActions()
	if len(updateActions) != len(body.UpdateActions) {
		return fmt.Errorf("the length of the update area is incorrect, want: %v, get: %v", len(updateActions), len(body.UpdateActions))
	}
	if !reflect.DeepEqual(updateActions, body.UpdateActions) {
		return fmt.Errorf("the generated update area is incorrect, want: %v, get: %v", updateActions, body.UpdateActions)
	}

	// Check that the successful number of transactions matches the number of records in the block.
	if len(r.GetSuccessfulTxs()) != len(body.Transactions) {
		return fmt.Errorf("there are failed transactions in block body")
	}

	// Verify smartContract.
	if err := abc.verifySmartContract(body); err != nil {
		return err
	}

	return nil
}

func (abc *appBlockChain) verifyTx(tx *wire.MsgTxWithProofs) error {
	// Transaction
	//
	// Nout -> Normal Outpoint
	// Dout -> Deposit Outpoint
	// Wout -> Withdraw Outpoint
	//
	// -------------------------------------------------------------
	// Type      | Transfer   | Deposit    | WithDraw | Binding    |
	// ------------------------------------------------------------|
	// Input     | Wout       | Wout       | Dout     | Dout       |
	//           | Nout       | Nout       |          |            |
	// ------------------------------------------------------------|
	// Output    | Nout       | Dout       | Wout     | Dout       |
	// ------------------------------------------------------------|
	// Ipunt     | Wout not   | Wout not   | No       | Dout not   |
	// Condition | in locking | in locking |          | in locking |
	// -------------------------------------------------------------
	switch tx.Tx.API {
	// Transfer transaction & Deposit Transaction
	// If input is wout, check lock height
	// If input is in locking, check failed
	case isysapi.SysAPITransfer, isysapi.SysAPIDeposit:
		for _, txIn := range tx.Tx.TxIn {
			if txIn.PreviousOutPoint.IsLock(abc.proposeBlockHgt) {
				return fmt.Errorf("withdraw out is in locking")
			}
			if txIn.PreviousOutPoint.IsDepositOut() {
				return fmt.Errorf("transfer tx's input should not be dout")
			}
		}
	// Withdraw transaction
	// Tx's input must be deposit out
	case isysapi.SysAPIWithdraw:
		for _, txIn := range tx.Tx.TxIn {
			if !txIn.PreviousOutPoint.IsDepositOut() {
				return fmt.Errorf("withdraw tx's input must be deposit out")
			}
		}
	// Binding transaction
	// Check all deposit out's lock height
	case isysapi.SysAPIBinding:
		for _, txIn := range tx.Tx.TxIn {
			if !txIn.PreviousOutPoint.IsDepositOut() {
				return fmt.Errorf("binding tx's input must be deposit out")
			}
			if txIn.PreviousOutPoint.IsBindingLock(abc.proposeBlockHgt) {
				return fmt.Errorf("binding tx's input is in locking")
			}
		}
	}

	return nil
}

// Verify smartContracts
// TODO(#issue 178): verify the out created by smartcontract, But now storage nodes cannot index smart contract data
func (abc *appBlockChain) verifySmartContract(body *wire.BlockBody) error {
	tp := txprocessor.NewTxProcessor(abc.shardIndex, abc.smartContractDataStore)

	for _, tx := range body.Transactions {
		tp.AddTx(&tx.Tx)
	}
	blockHgt := body.LedgerInfo.GetShardHeight(abc.shardIndex) + 1
	r := tp.Execute(blockHgt)

	scs := r.GetSmartContracts()
	scsInBody := body.SmartContracts

	if len(scs) != len(scsInBody) {
		return fmt.Errorf("verify failed: the length of smart contract is Wrong, %d != %d", len(scs), len(body.SmartContracts))
	}

	for i := 0; i < len(scs); i++ {
		if !scs[i].Equal(scsInBody[i]) {
			return fmt.Errorf("verify failed: the content of smartContract is different")
		}
	}

	return nil
}

// verifyBlockBody checks if the block body is correct.
func (abc *appBlockChain) verifyBlockBody(block *wire.MsgBlock) error {
	body := block.Body
	hash := &block.Header.BlockBodyHash
	actualHash := body.BlockBodyHash()

	// Verify the hash of block
	if !hash.IsEqual(&actualHash) {
		return invalidBlockError("Header.BlockBodyHash", block.Header.BlockBodyHash, actualHash)
	}

	// Verify block body.
	ledgerRoot := abc.state.GetLedgerRoot()
	if err := abc.verifyBlockBodyInternal(
		body,
		&ledgerRoot,
		&block.Header.OutsMerkleRoot,
		abc.shardIndex); err != nil {

		return err
	}

	// Verify clock vector.
	var size int64
	for _, index := range shard.ShardList {
		// TODO: consider rejecting LedgerInfo including the block headers which are not heard yet.
		if body.LedgerInfo.GetShardHeight(index) < abc.state.GetShardHeight(index) {
			return invalidBlockError(
				fmt.Sprintf("Body.LedgerInfo.GetShardHeight(%v)", index),
				body.LedgerInfo.GetShardHeight(index),
				abc.state.GetShardHeight(index))
		}
		size += body.LedgerInfo.GetShardHeight(index)
	}
	if body.LedgerInfo.Size != size {
		return invalidBlockError("Body.LedgerInfo.Size", body.LedgerInfo, size)
	}

	return nil
}

// verifyReduceActionsInBlock verifies whether reduce tx generated by header and reduce tx in block are identical.
func (abc *appBlockChain) verifyReduceActionsInBlock(block *wire.MsgBlock) error {

	// Add headers that from other shards to ledger tree.
	ledgerInfoProposal := block.Body.LedgerInfo
	headers, err := abc.state.GetBlockHeadersToAppend(&ledgerInfoProposal)
	if err != nil {
		return err
	}

	var reduceTxs []*wire.MsgTxWithProofs

	// Build reduce tx based on reduce action and add it to TxProcessor
	for _, header := range headers {
		reduceTxsInHeader, err := abc.generateReduceTxsByHeader(header)
		if err != nil {
			return err
		}
		if reduceTxsInHeader != nil {
			reduceTxs = append(reduceTxs, reduceTxsInHeader...)
		}
	}

	var reduceTxsInBlock []*wire.MsgTxWithProofs
	txs := block.Body.Transactions
	for _, tx := range txs {
		if tx.Tx.IsReduceTx() {
			reduceTxsInBlock = append(reduceTxsInBlock, tx)
		}
	}
	if len(reduceTxs) != len(reduceTxsInBlock) {
		return fmt.Errorf("the number of reduce tx generated is inconsistent, lenInBlock: %d, lenByGenerated: %d", len(reduceTxsInBlock), len(reduceTxs))
	}
	if !reflect.DeepEqual(reduceTxs, reduceTxsInBlock) {
		return fmt.Errorf("the reduce txs generated through header is inconsistent with the reduce txs in the block")
	}

	return nil
}

// VerifyBlock checks block header and block body.
func (abc *appBlockChain) VerifyBlock(block *wire.MsgBlock) error {
	if err := abc.verifyBlockHeader(&block.Header); err != nil {
		return err
	}

	if err := abc.verifyBlockBody(block); err != nil {
		return err
	}

	err := abc.verifyReduceActionsInBlock(block)
	if err != nil {
		return err
	}

	// Verify the new ledger root is correct.
	update, err := abc.state.NewUpdate()
	if err != nil {
		return err
	}
	// Block body has been verified, so it's guaranteed that each transaction has same amount
	// txin and proofs.
	for _, tx := range block.Body.Transactions {
		for index := range tx.Tx.TxIn {
			if !tx.Tx.IsReduceTx() {
				err := update.MarkSpentOuts(&tx.Tx.TxIn[index].PreviousOutPoint, &tx.Proofs[index])
				if err != nil {
					abc.log.Errorf("failed to mark spent out, %v", err)
				}
			}
		}
	}
	var headers []*wire.BlockHeader
	headers, err = abc.state.GetBlockHeadersToAppend(&block.Body.LedgerInfo)
	if err != nil {
		return err
	}

	// verify headers that can be mounted
	for _, header := range headers {
		// generateReduceTxsByHeader is used solely to identify whether the header can be mounted.
		_, err := abc.generateReduceTxsByHeader(header)
		if err != nil {
			return err
		}
	}
	if err := update.AddHeaders(headers...); err != nil {
		return err
	}
	newShardLedgerRoot := update.DeltaTree.GetMerkleRoot()
	if block.Header.ShardLedgerMerkleRoot != *newShardLedgerRoot {
		return fmt.Errorf("Wrong ledger merkle root")
	}
	return nil
}

// TODO: Restrict the size of block.
// ProposeBlock returns a valid block with transactions as a new block proposal.
// When ProposeBlock generates a new reshard seed using the pk and sk, if needed.
func (abc *appBlockChain) ProposeBlock(pk []byte, sk []byte) (*wire.MsgBlock, error) {
	// When proposing a new block, the ledger of the miner should be updated.
	// Update instance holds the old ledger tree that can be changed by
	// transactions and block-tree-root, returns the new ledger after updating.fmt.
	update, err := abc.state.NewUpdate()
	if err != nil {
		return nil, err
	}

	// Extract transactions from the transaction pool.
	allTxs := abc.txPool.GetTopTransactions(maxTransactionsInBlock, update, abc.shardHeight)

	// it used to store transactions that filter out reduce tx
	var txs []*wire.MsgTxWithProofs
	// Reduce actions all of mountHeader can build reduce tx
	var mountHeader []*wire.BlockHeader
	// errShard is used to store the sharding heights that go wrong when building reduce tx based on headers
	var errShard map[shard.Index]int64

	// Checking the correctness of all transactions
	for _, tx := range allTxs {
		// Filter reduce transactions to prevent evil
		if tx.Tx.IsReduceTx() {
			continue
		}

		// Verify tx interval and ignore invalid tx
		err := abc.verifyTx(tx)
		if err != nil {
			abc.log.Errorf("Failed to verify tx, err: %v", err)
			continue
		}

		// Transactions from tx pool are supposed to be valid and have same
		// amount of proof and out points.
		for index := range tx.Tx.TxIn {
			out := &tx.Tx.TxIn[index].PreviousOutPoint
			err := update.MarkSpentOuts(out, &tx.Proofs[index])
			if err != nil {
				panic(err)
			}
		}
		txs = append(txs, tx)
	}

	// Add headers that from other shards to ledger tree.
	ledgerInfoProposal := abc.state.LocalLedgerInfo()
	headers, err := abc.state.GetBlockHeadersToAppend(ledgerInfoProposal)
	if err != nil {
		abc.log.Errorf("Propose block error: %v", err)
		return nil, err
	}

	// Build reduce tx based on reduce action and add it to TxProcessor
	for _, header := range headers {
		if _, ok := errShard[header.ShardIndex]; ok {
			continue
		}
		reduceTxsInHeader, err := abc.generateReduceTxsByHeader(header)
		if err != nil {
			abc.log.Debugf("fail to generateReduceTxsByHeader, err: %v", err)
			if errShard == nil {
				errShard = make(map[shard.Index]int64)
			}
			errShard[header.ShardIndex] = header.Height
			continue
		}
		if reduceTxsInHeader != nil {
			txs = append(txs, reduceTxsInHeader...)
		}
		mountHeader = append(mountHeader, header)
	}

	err = update.AddHeaders(mountHeader...)
	if err != nil {
		panic(err)
	}

	// update ledgerInfoProposal
	for _, shardHeight := range ledgerInfoProposal.ShardsHeights {
		shardIdx := shardHeight.Index
		if hgt, ok := errShard[shardIdx]; ok {
			ledgerInfoProposal.SetShardHeight(shardIdx, hgt)
		}
	}

	// Generate the new root of ledger.
	newShardLedgerRoot := update.DeltaTree.GetMerkleRoot()

	if params.EnableStorageReward {
		storageRewardTx := getStorageReward(txs, abc.shardHeight, abc.shardIndex)
		txs = append(txs, storageRewardTx...)
	}

	// Put miner reward tx at the first place, and only one miner reward, storage reward can be more than one.
	// TODO:(zz) using binding address
	pkAddr, err := abc.getRewardAddress()
	if err != nil {
		abc.log.Errorf("Failed to get reward address for this miner: %v", err)
		return nil, err
	}
	if params.EnableCoinbaseTx {
		rewardTx := createRewardTx(abc.shardIndex, abc.shardHeight, pkAddr)
		txs = append([]*wire.MsgTxWithProofs{rewardTx}, txs...)
	}

	// If enable resharding, update next seed of resharding.
	reshardSeed := abc.prevReshardHeader.ReshardSeed

	//Update ReshardSeed if need to resharding, ReshardSeed -> previous ReshardSeed
	if (abc.proposeBlockHgt - abc.prevReshardHeader.Height) >= params.ReshardRoundThreshold {

		reshardSeed, err = genNextReshardSeed(pk, sk, reshardSeed, abc.proposeBlockHgt)
		if err != nil {
			return nil, errors.New("when propose block, can't update reshardseed")
		}
	}

	// Processing all transactions and generate the root of outs-tree.
	var tp = txprocessor.NewTxProcessor(abc.shardIndex, abc.smartContractDataStore)

	for _, tx := range txs {
		tp.AddTx(&tx.Tx)
	}

	// Executing trades and getting Merkel trees
	r := tp.Execute(abc.proposeBlockHgt)
	outsMerkleRoot := r.GetMerkleRoot()

	// Package all correct tx to the new block.
	txsMap := make(map[chainhash.Hash]*wire.MsgTxWithProofs)
	for _, tx := range txs {
		txsMap[tx.Tx.TxHash()] = tx
	}

	var txsInBlock []*wire.MsgTxWithProofs
	for _, tx := range r.GetSuccessfulTxs() {
		txsInBlock = append(txsInBlock, txsMap[tx.TxHash()])
	}

	prevHash := abc.prevHeader.BlockHeaderHash()

	newBlock := wire.NewBlock(abc.shardIndex,
		abc.proposeBlockHgt,
		prevHash,
		*newShardLedgerRoot,
		*outsMerkleRoot,
		txsInBlock,
		ledgerInfoProposal,
		false,
		time.Now().Unix(),
		reshardSeed,
		r.GetSmartContracts(),
		r.GetOutStates(),
		r.GetUpdateActions(),
		r.GetReduceActions())
	newBlock.Header.Pk = pk
	return newBlock, nil
}

func (abc *appBlockChain) ProposeAndAcceptEmptyBlock(height int64) (*wire.BlockHeader, error) {
	prevHeader := abc.prevHeader
	if prevHeader.Height+1 != abc.proposeBlockHgt || abc.proposeBlockHgt != height {
		//abc.log.Errorf("Faild to propose empty emptyBlock at height %d", abc.proposeBlockHgt)
		return nil, fmt.Errorf("Propose emptyBlock error, height not match")
	}

	// TODO(zz): If not used these code, remove

	preHeaderHash := prevHeader.BlockHeaderHash()
	abc.log.Debugf("Previous emptyBlock header, height: %v, root: %v", prevHeader.Height, prevHeader.ShardLedgerMerkleRoot)
	prevLedgerRoot := prevHeader.ShardLedgerMerkleRoot

	// Empty emptyBlock's timeStamp is last emptyBlock's timeStamp plus ten seconds
	timeStamp := int64(prevHeader.TimeStamp + 10)
	reshardSeed := prevHeader.ReshardSeed

	// Empty emptyBlock's ledgerroot is the ledgerroot of previous emptyBlock's
	abc.log.Debugf("Propose empty emptyBlock height %v, previousBlockHeaderHash: %v, prevLedgerRoot: %v, Seed: %v",
		abc.proposeBlockHgt, preHeaderHash, prevLedgerRoot, reshardSeed)

	emptyBlock := wire.NewBlock(abc.shardIndex,
		abc.proposeBlockHgt,
		preHeaderHash,
		prevLedgerRoot, merkle.EmptyHash, nil, nil, true, timeStamp, reshardSeed, nil, nil, nil, nil)
	if err := abc.blockChain.ReceiveHeader(&emptyBlock.Header); !err {
		abc.log.Debugf("Failed to call blockChain.ReceiveHeader on emptyBlock %v", emptyBlock.Header.Height)
	}
	abc.blockAccepted(emptyBlock)
	return &emptyBlock.Header, nil
}

// AcceptBlock Handles insertion of new blocks into the block chain and merkle tree.
// It will verify the block before actually inserting, if the block is invalid an error will be returned.
// For now, it's the caller's responsibility to make sure given block is under consensus.
// If VerifyBlock returns err, but block shard and version is right, it will cache the block, and try to catch the latest block
// It is used to facilitate the reconstruction of consensus
func (abc *appBlockChain) AcceptBlock(block *wire.MsgBlock) error {
	if block == nil {
		abc.log.Errorf("can't receive nil")
		return fmt.Errorf("block is nil")
	}

	if abc.shardHeight+1 > block.Header.Height {
		return fmt.Errorf("the block may have been accepted")
	}

	abc.log.Debugf("Before accept block, height: %d, prevHeader: %v,prevHeader.Height: %v, reshardSeed: %v",
		block.Header.Height, abc.prevHeader.BlockHeaderHash(), abc.prevHeader.Height, abc.prevReshardHeader.ReshardSeed)

	if err := abc.blockChain.ReceiveHeader(&block.Header); !err {
		abc.log.Debugf("Failed to call blockChain.ReceiveHeader on block %v", block)
	}

	// Verify if the block is correct
	if err := abc.VerifyBlock(block); err != nil {
		abc.log.Debugf("Cant't verify block: %v, abci will fetch new ledger at height %d", err, block.Header.Height)

		// Try to fetch new ledger
		abc.fetchNewLedger()
		//abc.log.Errorf("Failed to accept block, fetch ledger continue,err: %v", err)
		if !bytes.Equal(block.Header.ReshardSeed, abc.prevReshardHeader.ReshardSeed) {
			abc.log.Debugf("Update reshard seed at height %d", block.Header.Height)
			abc.prevReshardHeader = block.Header
		}
		return err
	}
	update, err := abc.state.GetUpdateWithFullBlock(block)
	if err != nil {
		panic(err)
	}
	// Update deposit pool. It is same to update tx that in txPool.
	err = abc.dPool.Update(update, abc.shardIndex, wire.BlockHeight(block.Header.Height))
	if err != nil {
		abc.log.Errorf("Update deposit pool error: %v", err)
		panic(err)
	}

	// Update txpool
	abc.txPool.RefreshPool(update)

	// Update Datastore
	abc.smartContractDataStore.RefreshDataStore(update, block.Body.UpdateActions)

	// Update abci's ledger
	err = abc.state.ApplyUpdate(update)
	if err != nil {
		panic("failed to apply update on abci")
	}

	preBlock := abc.prevBlock
	// Try to add deposit txs witch in last block to deposit pool
	if preBlock != nil {
		err := abc.updateLastBlockDeposit(update, preBlock)
		if err != nil {
			panic(err)
		}
	}
	abc.blockAccepted(block)
	return nil
}

// updateLastBlockDeposit updates deposit information for the previous block.
// Note: dout is deposit out
// When a dout is packaged in a block, it will not be immediately hung on the
// merkel tree. Because the current block's clock vector does not contain itself,
// so, we can only get the merkle path to dout in the next round.
func (abc *appBlockChain) updateLastBlockDeposit(update *state.Update, block *wire.MsgBlock) error {
	fTree := getFullMerkleTreeByBlock(block)
	// Traverse all outs, add dout to the deposit pool
	for _, out := range block.Body.Outs {
		myAddress := abc.getAddress()
		if out.OutPoint.IsDepositOut() {
			outData, err := wire.OutToDepositData(&out.OutPoint)
			if err != nil {
				abc.log.Errorf("Failed to get data out deposit out: %v", out.OutPoint)
				return err
			}
			// The miner only cares about his own deposit
			if outData.BindingAddress.IsEqual(myAddress) {
				proof, err := getOutProofByUpdate(update, fTree, out)
				if err != nil {
					abc.log.Errorf("Failed to get proof of deposit out,%v", err)
					return err
				}
				err = abc.dPool.Add(&out.OutPoint, wire.BlockHeight(block.Header.Height), proof)
				if err != nil {
					abc.log.Errorf("Failed to add dout to pool: %s", out.OutPoint.String())
					return err
				}
			}
		}
	}

	// Traverse all transactions, If a dout was used, lock or remove
	for _, tx := range block.Body.Transactions {
		switch tx.Tx.API {
		case isysapi.SysAPIWithdraw:
			for _, txIn := range tx.Tx.TxIn {
				if txIn.PreviousOutPoint.IsDepositOut() {
					err := abc.dPool.Lock(&txIn.PreviousOutPoint)
					if err != nil {
						abc.log.Errorf("failed to lock previous outpoint,%v", err)
						return err
					}
				}
			}
		case isysapi.SysAPIBinding:
			for _, txIn := range tx.Tx.TxIn {
				if txIn.PreviousOutPoint.IsDepositOut() {
					err := abc.dPool.Remove(&txIn.PreviousOutPoint)
					if err != nil {
						abc.log.Errorf("failed to remove previous outpoint,%v", err)
						return err
					}
				}
			}
		case isysapi.SysAPIDeposit, isysapi.SysAPITransfer:
			for _, txIn := range tx.Tx.TxIn {
				if txIn.PreviousOutPoint.IsWithdrawOut() {
					err := abc.dPool.Remove(&txIn.PreviousOutPoint)
					if err != nil {
						abc.log.Errorf("failed to remove previous outpoint,%v", err)
						return err
					}
				}
			}
		}
	}
	return nil
}

func (abc *appBlockChain) blockAccepted(block *wire.MsgBlock) {
	// TODO: testnet-3.0
	if !block.Header.IsEmptyBlock && !bytes.Equal(block.Header.ReshardSeed, abc.prevReshardHeader.ReshardSeed) {
		abc.log.Debugf("Update reshard seed at height %d", block.Header.Height)
		abc.prevReshardHeader = block.Header
	}

	// Update height of shard
	abc.shardHeight++
	abc.proposeBlockHgt++
	abc.prevHeader = block.Header
	abc.prevBlock = block

	// broadcast block to the own shard
	//abc.broadcast <- block

	// Broadcast slimblock to the corresponding shard
	slimBlocks, err := wire.GetSlimBlocksFromBlock(block)
	if err != nil {
		abc.log.Debugf("get slimBlocks error : %v", err)
		return
	}

	for _, slimBlock := range slimBlocks {
		abc.log.Debugf("abc broadcast slimBlock shard : %v, out number : %v\n", slimBlock.ToShard, len(slimBlock.ClipTreeData.Outs))
		abc.broadcastMsg(slimBlock)
	}
	abc.log.Debugf("Accept block successfully, height: %d, ledger: %v, state: %v, preblock: %v",
		block.Header.Height, block.Header.ShardLedgerMerkleRoot, abc.state.GetLedgerRoot(), abc.prevHeader.BlockHeaderHash())

	// print info
	abc.printMinerInfo(block)

	abc.monitoringShardHeight()

	abc.monitoringTxNum()
}

func (abc *appBlockChain) generateReduceTxsByHeader(header *wire.BlockHeader) ([]*wire.MsgTxWithProofs, error) {
	// reduceSk is used to sign reduce tx
	reduceSk := reducekey.ReducePrivateKey

	var reduceTxs []*wire.MsgTxWithProofs

	for _, reduceAction := range header.ReduceActions {
		reduceOut := reduceAction.ToBeReducedOut
		if reduceOut.Shard != abc.shardIndex {
			continue
		}
		contractAddrHash, err := reduceOut.ContractAddress.GetPublicKeyHash(multivacaddress.SmartContractAddress)
		if err != nil {
			return nil, err
		}
		sCInfo := abc.smartContractDataStore.GetSmartContractInfo(*contractAddrHash)
		if sCInfo == nil {
			abc.fetchSmartContract(reduceOut.ContractAddress, reduceOut.Shard)
		}
		tx := &wire.MsgTx{
			Version: wire.TxVersion,
			Shard:   abc.shardIndex,
			TxIn: []*wire.TxIn{
				&wire.TxIn{
					PreviousOutPoint: *reduceOut,
				},
			},
			ContractAddress: reduceOut.ContractAddress,
			API:             reduceAction.API,
		}

		tx.Sign(&reduceSk)

		txWithProof := &wire.MsgTxWithProofs{
			Tx:     *tx,
			Proofs: nil,
		}

		reduceTxs = append(reduceTxs, txWithProof)
	}
	return reduceTxs, nil
}

func (abc *appBlockChain) printMinerInfo(block *wire.MsgBlock) {
	pkBytes, _ := hex.DecodeString(config.GlobalConfig().Pk)
	if bytes.Equal(pkBytes, block.Header.Pk) {
		abc.log.Infof("====================================================")
		abc.log.Infof("| 💰 💰 Mined a new block, shard: %d, height: %d  |", abc.shardIndex, block.Header.Height)
		abc.log.Infof("====================================================")
		content := fmt.Sprintf("| 💰 💰 Mined a new block, shard: %d, height: %d  | %v",
			abc.shardIndex, block.Header.Height, time.Now())
		_, err := util.RedirectLog(content, "~/summary.txt")
		if err != nil {
			abc.log.Errorf("failed to RedirectLog: %v", err)
		}
	} else {
		abc.log.Infof("Accept a new block, shard: %d, height: %d", abc.shardIndex, block.Header.Height)
	}
}

func invalidBlockError(field string, value, expected interface{}) error {
	return fmt.Errorf("invalid block field: %s %+v (actual) vs %+v (expected)", field, value, expected)
}

// monitoringTxNumData 监控采集矿工交易池中交易总数
func (abc *appBlockChain) monitoringTxNum() {
	metric := metrics.Metric
	if abc.txPool != nil {
		metric.PendingTxNum.With(
			prometheus.Labels{"shard_id": fmt.Sprint(abc.shardIndex)},
		).Set(float64(abc.txPool.NumberOfTransactions()))
	}
}

// monitoringShardHeight 监控采集miner分片高度
func (abc *appBlockChain) monitoringShardHeight() {
	metric := metrics.Metric
	metric.ShardHeight.With(
		prometheus.Labels{"shard_id": fmt.Sprint(abc.shardIndex)},
	).Set(float64(abc.shardHeight))
}
