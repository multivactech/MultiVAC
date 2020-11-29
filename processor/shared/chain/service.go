package chain

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// Service is a struct which implements iblockchain.BlockChain interface.
type Service struct {
	actor    message.Actor
	actorCtx *message.ActorContext
}

// Is unused
// func (s *Service) sendAndWait(e *message.Event) interface{} {
// 	c := make(chan interface{}, 1)
// 	s.actorCtx.Send(s.actor, e, func(r interface{}) { c <- r })
// 	return <-c
// }

// ReceiveBlock sends the ReceiveBlock Event to the service's actor and wait until getting response.
func (s *Service) ReceiveBlock(block *wire.MsgBlock) bool {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtReceiveBlock, block)).(bool)
}

// ReceiveHeader sends the ReceiveHeader Event to the service's actor and wait until getting response.
func (s *Service) ReceiveHeader(header *wire.BlockHeader) bool {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtReceiveHeader, header)).(bool)
}

// GetShardsHeight sends the ShardHeight Event to the service's actor and wait until getting a BlockHeight.
func (s *Service) GetShardsHeight(shardIndex shard.Index) wire.BlockHeight {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtShardHeight, shardIndex)).(wire.BlockHeight)
}

// GetShardsBlockByHeight sends the BlockByShardAndHeight Event to the service's actor and wait until getting a MsgBlock.
func (s *Service) GetShardsBlockByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.MsgBlock {
	return s.actorCtx.SendAndWait(s.actor,
		message.NewEvent(evtBlockByShardAndHeight,
			shard.NewShardIndexAndHeight(shardIndex, int64(hgt)))).(*wire.MsgBlock)
}

// GetShardsBlockByHash sends the BlockByHash Event to the service's actor and wait until getting a MsgBlock.
func (s *Service) GetShardsBlockByHash(headerHash chainhash.Hash) *wire.MsgBlock {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtBlockByHash, headerHash)).(*wire.MsgBlock)
}

// GetShardsHeaderByHeight sends the HeaderByShardAndHeight Event to the service's actor and wait util getting a block header.
func (s *Service) GetShardsHeaderByHeight(shardIndex shard.Index, hgt wire.BlockHeight) *wire.BlockHeader {
	return s.actorCtx.SendAndWait(s.actor,
		message.NewEvent(evtHeaderByShardAndHeight,
			shard.NewShardIndexAndHeight(shardIndex, int64(hgt)))).(*wire.BlockHeader)
}

// GetShardsHeaderByHash sends the HeaderByHash Event to the service's actor and wait util getting a block header.
func (s *Service) GetShardsHeaderByHash(headerHash chainhash.Hash) *wire.BlockHeader {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtHeaderByHash, headerHash)).(*wire.BlockHeader)
}

// SetSyncTrigger sends SetTrigger Event to the service's actor.
func (s *Service) SetSyncTrigger(shard shard.Index, trigger SyncTrigger) {
	s.actorCtx.Send(s.actor, message.NewEvent(evtSetTrigger, &triggerRequest{shard, trigger}), nil)
}

// GetShardsHeaderHashes sends a ShardHeaderHashes Event to the service's actor and wait util getting headerHashes.
func (s *Service) GetShardsHeaderHashes(
	shardIndex shard.Index, fromHgt wire.BlockHeight, toHgt wire.BlockHeight) []chainhash.Hash {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(
		evtShardHeaderHashes,
		wire.NewBlockLocator(shardIndex, int64(fromHgt), int64(toHgt)))).([]chainhash.Hash)
}

// GetSlimBlock sends a SlimBlockMsgByShardAndHeight Event to the service's actor and wait util getting a SlimBlock.
func (s *Service) GetSlimBlock(toshard shard.Index, shardIndex shard.Index, hgt wire.BlockHeight) *wire.SlimBlock {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtSlimBlockMsgByShardAndHeight, saveSlimBlockRequest{toshard: toshard, shard: shardIndex, height: hgt})).(*wire.SlimBlock)
}

// ReceiveSlimBlock sends a ReceiveSlimBlock Event to the service's actor and wait util getting a response.
func (s *Service) ReceiveSlimBlock(msg *wire.SlimBlock) bool {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtReceiveSlimBlock, msg)).(bool)
}

// GetSmartContract sends a SmartContractByAddress Event to the service's actor and wait util getting a SmartContract.
func (s *Service) GetSmartContract(addr multivacaddress.Address) *wire.SmartContract {
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtSmartContractByAddress, addr)).(*wire.SmartContract)
}

// GetSmartContractCodeOut sends a SmartContractCodeOut Event to the service's actor and wait util getting a OutState.
func (s *Service) GetSmartContractCodeOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	addrShard := []interface{}{contractAddr, shardIdx}
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtSmartContractCodeOut, addrShard)).(*wire.OutState)
}

// GetSmartContractShardInitOut sends a SmartContractShardInitOut Event to the service's actor and wait util getting a OutState.
func (s *Service) GetSmartContractShardInitOut(contractAddr multivacaddress.Address, shardIdx shard.Index) *wire.OutState {
	addrShard := []interface{}{contractAddr, shardIdx}
	return s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtSmartContractShardInitOut, addrShard)).(*wire.OutState)
}

// ReceiveSmartContractShardInitOut sends a ReceiveSmartContractShardInitOut Event to the service's actor and
// wait util getting a response.
func (s *Service) ReceiveSmartContractShardInitOut(out *wire.OutState) error {
	err := s.actorCtx.SendAndWait(s.actor, message.NewEvent(evtReceiveSmartContractShardInitOut, out))
	if err == nil {
		return nil
	}
	return err.(error)
}
