/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

import (
	"fmt"

	"github.com/multivactech/MultiVAC/base/db"
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/base/util"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	headerKeyTemplate                 = "header_%s"
	blockKeyTemplate                  = "block_%s"
	indexAndHeightKeyTemplate         = "%d_%d"
	smartContractKeyTemplate          = "smartContract_%s"
	slimBlockKeyTemplate              = "toShard_%d_shard_%d_height_%d"
	smartContractShardInitOutTemplate = "smartContractInitOut_%s_%d"
	smartContractCodeOutTemplate      = "smartContractCodeOut_%s_%d"
	dbNameSpace                       = "chainData"
)

type diskBlockChain struct {
	// The current max continuous height of the block chain.
	curShardsHeight map[shard.Index]wire.BlockHeight

	// Database for disk store
	chainDB db.DB

	// used to trigger synchronization
	syncTriggers map[shard.Index]SyncTrigger
	logger       btclog.Logger
}

type headerNode struct {
	headerHash chainhash.Hash
	header     *wire.BlockHeader
}

// newBlockChain returns the blockChain instance
func newDiskBlockChain() *diskBlockChain {
	chain := diskBlockChain{}
	chain.curShardsHeight = make(map[shard.Index]wire.BlockHeight)
	chain.syncTriggers = make(map[shard.Index]SyncTrigger)

	// TODO: Temporary soultion for disk store.
	// Miner node doesn't need to store all data.
	db, err := db.OpenDB(config.GlobalConfig().DataDir, dbNameSpace)
	if err != nil {
		panic(err)
	}
	chain.chainDB = db
	// log config
	chain.logger = logBackend.Logger("BlockChain-Disk")
	chain.logger.SetLevel(logger.ChainLogLevel)

	//read curHeight from db
	for _, shardIndex := range shard.ShardList {
		rawData, err := chain.chainDB.Get(util.GetShardHeightKey(shardIndex.GetID()))
		if err != nil {
			chain.logger.Debugf("there's no curHeight in db : %v. If it's first start, can ingore this error info.", err)
		} else {
			height := util.BytesToInt64(rawData)
			chain.logger.Debugf("shard:%d height %d from db", shardIndex, height)
			chain.curShardsHeight[shardIndex] = wire.BlockHeight(height)
		}
	}

	chain.init()
	return &chain
}

func (chain *diskBlockChain) init() {
	// Load all shards' genesis blocks
	for _, shardIndex := range shard.ShardList {
		//read curShardHeight from db
		if _, hasKey := chain.curShardsHeight[shardIndex]; !hasKey {
			ok := chain.ReceiveBlock(genesis.GenesisBlocks[shardIndex])
			if ok {
				chain.curShardsHeight[shardIndex] = 1
			} else {
				panic("can't load genesis block!")
			}
		}
	}
}

func (chain *diskBlockChain) SetSyncTrigger(shard shard.Index, trigger SyncTrigger) {
	chain.syncTriggers[shard] = trigger
}

// ReceiveHeader will receives headers from all shards
func (chain *diskBlockChain) ReceiveHeader(header *wire.BlockHeader) bool {
	chain.logger.Debugf("receiving block header %v", header)
	return chain.receiveAllHeader(header) != nil
}

// ReceiveBlock will receives blocks from all shards
func (chain *diskBlockChain) ReceiveBlock(block *wire.MsgBlock) bool {
	chain.logger.Debugf("receiving block %v", block)
	h := chain.receiveAllHeader(&block.Header)
	if h == nil {
		return false
	}
	// encoding block
	dataToStore, err := rlp.EncodeToBytes(block)
	if err != nil {
		return false
	}
	// save block to db
	key := getBlockKey(block.Header.BlockHeaderHash())
	err = chain.chainDB.Put(key, dataToStore)
	if err != nil {
		chain.logger.Error(err)
		return false
	}

	// save smart contract to db
	if scs := block.Body.SmartContracts; len(scs) > 0 {
		chain.receiveSmartContracts(scs)
	}

	// save smart contract outs to db
	for _, out := range block.Body.Outs {
		if out.IsSmartContractCode() {
			err = chain.saveSmartContractCodeOut(out.ContractAddress, out.Shard, out)
			if err != nil {
				chain.logger.Errorf("fail to saveSmartContractCodeOut,contractAddr: %v,"+
					"shard: %v,out: %v", out.ContractAddress, out.Shard, out)
			}
		} else if out.IsSmartContractShardInitOut() {
			err = chain.saveSmartContractShardInitOut(out.ContractAddress, out.Shard, out)
			if err != nil {
				chain.logger.Errorf("fail to saveSmartContractShardInitOut,contractAddr: %v,"+
					"shard: %v,out: %v", out.ContractAddress, out.Shard, out)
			}
		}
	}

	return true
}

// receiveAllHeader will cache all non-existent headers
func (chain *diskBlockChain) receiveAllHeader(h *wire.BlockHeader) *headerNode {
	header := &headerNode{headerHash: h.BlockHeaderHash(), header: h}
	shardIndex := header.header.ShardIndex
	hgt := wire.BlockHeight(header.header.Height)
	if chain.containsShardsBlock(shardIndex, hgt) {
		// We have received this block already, ignore it.
		chain.logger.Debugf("Inside receiveHeader, rejected because already exists: %v", h)
		return header
	}

	shardIndexAndHeight := shard.IndexAndHeight{Index: shardIndex, Height: int64(hgt)}
	// Temporory sulotion for disk store
	dataToSave, err := rlp.EncodeToBytes(header.header)
	if err != nil {
		chain.logger.Error(err)
		return nil
	}
	// get key
	headerKey := getHeaderKey(header.headerHash)
	indexAndHeightKey := getIndexAndHeightKey(shardIndexAndHeight)
	// save to db
	err = chain.chainDB.Put(headerKey, dataToSave)
	if err != nil {
		chain.logger.Debugf("DBSave key: headerKey error:", err)
		return nil
	}
	err = chain.chainDB.Put(indexAndHeightKey, dataToSave)
	if err != nil {
		chain.logger.Debugf("DBSave key: indexAndeightKey error:", err)
		return nil
	}

	// try to update shards height
	for nextHgt := chain.curShardsHeight[shardIndex] + 1; chain.containsShardsBlock(shardIndex, nextHgt); nextHgt++ {
		chain.curShardsHeight[shardIndex] = nextHgt
	}
	if err = chain.chainDB.Put(util.GetShardHeightKey(shardIndex.GetID()),
		util.Int64ToBytes(int64(chain.curShardsHeight[shardIndex]))); err != nil {
		chain.logger.Debugf("DBSave GetShardHeightKey to curShardsHeight failed, detail: %v", err)
		return nil
	}

	// start sync when receive a higher header
	// condition: currentHeight+1 < receiveHeight
	if chain.curShardsHeight[shardIndex]+1 < hgt {
		chain.logger.Debugf("Recevie a higher header(%d), current header hight is %d, start to sync", hgt, chain.curShardsHeight[shardIndex])
		// if trigger, ok := chain.syncTriggers[shardIndex]; ok {
		// 	trigger.MaybeSync()
		// }
	}
	return header
}

// GetShardsHeight returns the height of the specified shard.
func (chain *diskBlockChain) GetShardsHeight(shardIndex shard.Index) wire.BlockHeight {
	return chain.curShardsHeight[shardIndex]
}

// containsShardsBlock determines whether a specified block exists in the cache by a given height and slice ID.
func (chain *diskBlockChain) containsShardsBlock(shardIndex shard.Index, hgt wire.BlockHeight) bool {
	shardIndexAndHeight := shard.IndexAndHeight{Index: shardIndex, Height: int64(hgt)}
	key := getIndexAndHeightKey(shardIndexAndHeight)
	header, err := chain.getHeaderByIndexAndHeight(key)
	if err != nil || header == nil {
		return false
	}
	return true
}

// GetShardsHeaderHashes returns the hashes of the specified shard and height range.
func (chain *diskBlockChain) GetShardsHeaderHashes(shardIndex shard.Index, fromHgt wire.BlockHeight, toHgt wire.BlockHeight) []chainhash.Hash {
	if chain.curShardsHeight[shardIndex] == 0 {
		return nil
	}
	if toHgt == wire.ReqSyncToLatest || toHgt > chain.GetShardsHeight(shardIndex) {
		toHgt = chain.GetShardsHeight(shardIndex)
	}
	chain.logger.Debugf("==== toHgt: %v, fromHgt: %v", toHgt, fromHgt)
	if toHgt < fromHgt {
		return nil
	}
	hashes := make([]chainhash.Hash, 0, toHgt-fromHgt)
	for i := fromHgt; i <= toHgt; i++ {
		shardIndexAndHeight := shard.IndexAndHeight{Index: shardIndex, Height: int64(i)}
		key := getIndexAndHeightKey(shardIndexAndHeight)
		header, err := chain.getHeaderByIndexAndHeight(key)
		if err != nil || header == nil {
			continue
		}
		hashes = append(hashes, header.BlockHeaderHash())
	}
	return hashes
}

// GetShardsBlockByHeight returns the height of the specified shard and height
func (chain *diskBlockChain) GetShardsBlockByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.MsgBlock {
	shardIndexAndHeight := shard.IndexAndHeight{Index: shardIndex, Height: int64(hgt)}
	key := getIndexAndHeightKey(shardIndexAndHeight)
	header, err := chain.getHeaderByIndexAndHeight(key)
	if err != nil || header == nil {
		return nil
	}
	// Query from db
	key = getBlockKey(header.BlockHeaderHash())
	block, err := chain.getBlock(key)
	if err != nil {
		return nil
	}
	return block
}

// GetSlimBlock returns the SlimBlock
func (chain *diskBlockChain) GetSlimBlock(toshard shard.Index, shardIndex shard.Index, hgt wire.BlockHeight) *wire.SlimBlock {
	// Query from db
	key := getSlimBlockKey(int(toshard), int(shardIndex), int(hgt))
	slimBlock, err := chain.getSlimBlock(key)
	if err != nil {
		return nil
	}
	return slimBlock
}

func (chain *diskBlockChain) getSlimBlock(key []byte) (*wire.SlimBlock, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, nil
	}

	slimBlock := &wire.SlimBlock{}
	err = rlp.DecodeBytes(rawData, slimBlock)

	if err != nil {
		return nil, err
	}
	return slimBlock, nil
}

func (chain *diskBlockChain) ReceiveSlimBlock(msg *wire.SlimBlock) bool {
	var err error
	chain.logger.Debugf("receiving slim block shardIndex : %v, height : %v", msg.Header.ShardIndex, msg.Header.Height)

	h := chain.receiveAllHeader(&msg.Header)
	if h == nil {
		chain.logger.Errorf("fail to receiveallheader in receive slimblock")
		return false
	}

	// save slimBlock message to db
	err = chain.saveSlimBlock(msg)
	if err != nil {
		chain.logger.Errorf("fail to saveSlimBlock, shard : %v, height : %v, err: %v", msg.ToShard, msg.Header.Height, err)
		return false
	}

	// save smart contract to db
	if len(msg.SmartContracts) > 0 {
		flag := chain.receiveSmartContracts(msg.SmartContracts)
		if !flag {
			chain.logger.Errorf("fail to receiveSmartContracts, smartContracts: %v", msg.SmartContracts)
			return false
		}
	}

	// save smart contract outs to db
	for _, out := range msg.ClipTreeData.Outs {
		if out.IsSmartContractCode() {
			err = chain.saveSmartContractCodeOut(out.ContractAddress, out.Shard, out)
			if err != nil {
				chain.logger.Errorf("fail to saveSmartContractCodeOut,contractAddr: %v,"+
					"shard: %v,out: %v, err: %v", out.ContractAddress, out.Shard, out, err)
				return false
			}
		} else if out.IsSmartContractShardInitOut() {
			err = chain.saveSmartContractShardInitOut(out.ContractAddress, out.Shard, out)
			if err != nil {
				chain.logger.Errorf("fail to saveSmartContractShardInitOut,contractAddr: %v,"+
					"shard: %v,out: %v, err: %v", out.ContractAddress, out.Shard, out, err)
				return false
			}
		}
	}

	return true
}

func (chain *diskBlockChain) saveSlimBlock(slimBlock *wire.SlimBlock) error {
	// encoding
	dataToStore, err := rlp.EncodeToBytes(slimBlock)
	if err != nil {
		return err
	}

	// save to db
	key := getSlimBlockKey(int(slimBlock.ToShard), int(slimBlock.Header.ShardIndex), int(slimBlock.Header.Height))
	return chain.chainDB.Put(key, dataToStore)
}

// GetShardsBlockByHash returns the block of the specified shard and hash
func (chain *diskBlockChain) GetShardsBlockByHash(headerHash chainhash.Hash) *wire.MsgBlock {
	key := getBlockKey(headerHash)
	block, err := chain.getBlock(key)
	if err != nil {
		return nil
	}
	return block
}

// GetShardsHeaderByHeight returns the block header of the specified shard and height
func (chain *diskBlockChain) GetShardsHeaderByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.BlockHeader {
	shardIndexAndHeight := &shard.IndexAndHeight{Index: shardIndex, Height: int64(hgt)}
	key := getIndexAndHeightKey(*shardIndexAndHeight)
	header, err := chain.getHeaderByIndexAndHeight(key)
	if err != nil || header == nil {
		return nil
	}
	return header
}

// GetShardsHeaderByHash returns the block header of the specified shard and hash
func (chain *diskBlockChain) GetShardsHeaderByHash(headerHash chainhash.Hash) *wire.BlockHeader {
	key := getHeaderKey(headerHash)
	if header, err := chain.getHeader(key); err == nil {
		return header
	}
	return nil
}

// getHeaderKey returns the key of header for disk store.
func getHeaderKey(hash chainhash.Hash) []byte {
	s := fmt.Sprintf(headerKeyTemplate, hash)
	return []byte(s)
}

// getBlockKey returns the key of block for disk store.
func getBlockKey(hash chainhash.Hash) []byte {
	s := fmt.Sprintf(blockKeyTemplate, hash)
	return []byte(s)
}

func getSlimBlockKey(toShard, shard, height int) []byte {
	s := fmt.Sprintf(slimBlockKeyTemplate, toShard, shard, height)
	return []byte(s)
}

// getIndexAndHeightKey returns the key of header for disk store,
// this key is consist of shard index and height.
func getIndexAndHeightKey(shardIndexAndHeight shard.IndexAndHeight) []byte {
	s := fmt.Sprintf(indexAndHeightKeyTemplate, shardIndexAndHeight.Index.GetID(), shardIndexAndHeight.Height)
	return []byte(s)
}

// getHeader get header from db by header's key
func (chain *diskBlockChain) getHeader(key []byte) (*wire.BlockHeader, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, nil
	}
	header := &wire.BlockHeader{}
	err = rlp.DecodeBytes(rawData, header)
	if err != nil {
		chain.logger.Error("RLP decoding error:", err)
		return nil, err
	}

	return header, nil
}

// getHeaderByIndexAndHeight get header from db by header's index and height key
func (chain *diskBlockChain) getHeaderByIndexAndHeight(key []byte) (*wire.BlockHeader, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, nil
	}
	header := &wire.BlockHeader{}
	err = rlp.DecodeBytes(rawData, header)
	if err != nil {
		chain.logger.Error("RLP decoding error:", err)
		return nil, err
	}

	return header, nil
}

// getBlock get block from db by block's key
func (chain *diskBlockChain) getBlock(key []byte) (*wire.MsgBlock, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, nil
	}
	block := &wire.MsgBlock{}
	err = rlp.DecodeBytes(rawData, block)
	if err != nil {
		chain.logger.Error("RLP decoding error:", err)
		return nil, err
	}
	return block, nil
}

// getSmartContractKey returns the key of smart contract
func getSmartContractKey(addr multivacaddress.Address) []byte {
	s := fmt.Sprintf(smartContractKeyTemplate, addr)
	return []byte(s)
}

// saveSmartContract将smart contract以contract address作为键持久化至数据库
func (chain *diskBlockChain) saveSmartContract(contractAddr multivacaddress.Address, smartContract *wire.SmartContract) error {
	// encoding
	dataToStore, err := rlp.EncodeToBytes(smartContract)
	if err != nil {
		return err
	}

	key := getSmartContractKey(contractAddr)
	// save to db
	return chain.chainDB.Put(key, dataToStore)
}

// GetSmartContract根据contract address返回smart contract结构体
func (chain *diskBlockChain) GetSmartContract(contractAddr multivacaddress.Address) *wire.SmartContract {
	// Query from db
	key := getSmartContractKey(contractAddr)
	sc, err := chain.getSmartContract(key)
	if err != nil {
		chain.logger.Errorf("failed to GetSmartContract, err: %v", err)
		return nil
	}
	return sc
}

// getSmartContract根据ContractAddress返回smart contract
func (chain *diskBlockChain) getSmartContract(key []byte) (*wire.SmartContract, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, fmt.Errorf("no corresponding value was found based on the keyword: %s", key)
	}
	sc := &wire.SmartContract{}
	err = rlp.DecodeBytes(rawData, sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// receiveSmartContracts会将收到的所有智能合约持久化至数据库
func (chain *diskBlockChain) receiveSmartContracts(scs []*wire.SmartContract) bool {
	for _, sc := range scs {
		err := chain.saveSmartContract(sc.ContractAddr, sc)
		if err != nil {
			chain.logger.Errorf("failed to receiveSmartContracts, err: %v", err)
			return false
		}
	}
	return true
}

// getSmartContractCodeOutKey returns the key of smart contract code out
func getSmartContractCodeOutKey(contractAddr multivacaddress.Address, shardIdx shard.Index) []byte {
	// 格式化中两个输入参数分别代表合约地址、分片编号
	s := fmt.Sprintf(smartContractCodeOutTemplate, contractAddr, shardIdx)
	return []byte(s)
}

func (chain *diskBlockChain) saveSmartContractCodeOut(contractAddr multivacaddress.Address, shardIdx shard.Index, out *wire.OutState) error {
	key := getSmartContractCodeOutKey(contractAddr, shardIdx)
	dataToStore, err := rlp.EncodeToBytes(out)
	if err != nil {
		return err
	}
	return chain.chainDB.Put(key, dataToStore)
}

// getSmartContractCodeOut根据合约地址以及分片编号返回部署合约后产生的代码out
func (chain *diskBlockChain) getSmartContractCodeOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	// Query from db
	key := getSmartContractCodeOutKey(contractAddr, shardIdx)
	out, err := chain.getSmartContractOut(key)
	if err != nil {
		chain.logger.Errorf("failed to getSmartContractCodeOut, err: %v", err)
		return nil
	}
	return out
}

// getSmartContractShardInitOutKey returns the key of smart contract shard init data out
func getSmartContractShardInitOutKey(contractAddr multivacaddress.Address, shardIdx shard.Index) []byte {
	// 格式化中三个输入参数分别代表合约地址、分片编号、OutPoint的index
	s := fmt.Sprintf(smartContractShardInitOutTemplate, contractAddr, shardIdx)
	return []byte(s)
}

func (chain *diskBlockChain) saveSmartContractShardInitOut(contractAddr multivacaddress.Address, shardIdx shard.Index, outs *wire.OutState) error {
	key := getSmartContractShardInitOutKey(contractAddr, shardIdx)
	dataToStore, err := rlp.EncodeToBytes(outs)
	if err != nil {
		return err
	}
	return chain.chainDB.Put(key, dataToStore)
}

// getSmartContractShardInitOut returns smart contract sharding data based on contract address and sharding number
func (chain *diskBlockChain) getSmartContractShardInitOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	// Query from db
	key := getSmartContractShardInitOutKey(contractAddr, shardIdx)
	out, err := chain.getSmartContractOut(key)
	if err != nil {
		chain.logger.Errorf("fail to getSmartContractShardInitOut, err: %v", err)
		return nil
	}
	return out
}

// getSmartContractOut根据key返回一个分片out
func (chain *diskBlockChain) getSmartContractOut(key []byte) (*wire.OutState, error) {
	rawData, err := chain.chainDB.Get(key)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return nil, fmt.Errorf("no corresponding value was found based on the keyword: %s", key)
	}
	var outs wire.OutState
	err = rlp.DecodeBytes(rawData, &outs)
	if err != nil {
		return nil, err
	}
	return &outs, nil
}

func (chain *diskBlockChain) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtReceiveBlock:
		callback(chain.ReceiveBlock(e.Extra.(*wire.MsgBlock)))
	case evtReceiveHeader:
		callback(chain.ReceiveHeader(e.Extra.(*wire.BlockHeader)))
	case evtShardHeight:
		callback(chain.GetShardsHeight(e.Extra.(shard.Index)))
	case evtBlockByShardAndHeight:
		shardAndHgt := e.Extra.(shard.IndexAndHeight)
		callback(chain.GetShardsBlockByHeight(shardAndHgt.Index, wire.BlockHeight(shardAndHgt.Height)))
	case evtHeaderByShardAndHeight:
		shardAndHgt := e.Extra.(shard.IndexAndHeight)
		callback(chain.GetShardsHeaderByHeight(shardAndHgt.Index, wire.BlockHeight(shardAndHgt.Height)))
	case evtBlockByHash:
		callback(chain.GetShardsBlockByHash(e.Extra.(chainhash.Hash)))
	case evtHeaderByHash:
		callback(chain.GetShardsHeaderByHash(e.Extra.(chainhash.Hash)))
	case evtSetTrigger:
		params := e.Extra.(*triggerRequest)
		chain.SetSyncTrigger(params.shard, params.trigger)
	case evtShardHeaderHashes:
		locator := e.Extra.(*wire.BlockLocator)
		callback(chain.GetShardsHeaderHashes(
			locator.ShardIdx, wire.BlockHeight(locator.FromHeight), wire.BlockHeight(locator.ToHeight)))
	case evtSlimBlockMsgByShardAndHeight:
		saveSlimBlockParam := e.Extra.(saveSlimBlockRequest)
		callback(chain.GetSlimBlock(saveSlimBlockParam.toshard, saveSlimBlockParam.shard, saveSlimBlockParam.height))
	case evtSmartContractByAddress:
		callback(chain.GetSmartContract(e.Extra.(multivacaddress.Address)))
	case evtReceiveSlimBlock:
		callback(chain.ReceiveSlimBlock(e.Extra.(*wire.SlimBlock)))
	case evtSmartContractCodeOut:
		msg := e.Extra.([]interface{})
		contractAddr := msg[0].(multivacaddress.Address)
		shardIdx := msg[1].(shard.Index)
		callback(chain.getSmartContractCodeOut(contractAddr, shardIdx))
	case evtSmartContractShardInitOut:
		msg := e.Extra.([]interface{})
		contractAddr := msg[0].(multivacaddress.Address)
		shardIdx := msg[1].(shard.Index)
		callback(chain.getSmartContractShardInitOut(contractAddr, shardIdx))
	case evtReceiveSmartContractShardInitOut:
		msg := e.Extra.(*wire.OutState)
		contractAddr := msg.ContractAddress
		shardIdx := msg.Shard
		callback(chain.saveSmartContractShardInitOut(contractAddr, shardIdx, msg))
	}
}
