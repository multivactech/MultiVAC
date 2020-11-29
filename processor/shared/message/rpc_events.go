/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package message

import "github.com/multivactech/MultiVAC/model/shard"

// EventType represents types for an event, which is used for internal communication within the application.
type EventType int

const (
	// RPCGetAllShardsInfo nonsharded request
	// args: []
	// return: *btcjson.ShardsInfo
	RPCGetAllShardsInfo EventType = iota
	// RPCGetOutState sharded request (storage node only)
	// args: [*wire.Outpoint]
	// return: *wire.OutSrate
	RPCGetOutState
	// RPCGetBlockInfo sharded request
	// args: [startHeight int64]
	// return: []*wire.MsgBlock
	RPCGetBlockInfo
	// RPCGetSlimBlockInfo sharded request (storage node only)
	// args: [toShard shard.Index, height int64]
	// return: []*wire.SlimBlock
	RPCGetSlimBlockInfo
	// RPCGetBlockSliceInfo sharded request (storage node only)
	// args: [startHeight int64, optional<endHeight int64>]
	// return: []*wire.MsgBlock
	RPCGetBlockSliceInfo
	// RPCGetSlimBlockSliceInfo sharded request (storage node only)
	// args: [toShard shard.Index, startHeight int64, optional<endHeight int64>]
	// return: []*wire.SlimBlock
	RPCGetSlimBlockSliceInfo
	// RPCGetSCInfo sharded request, get smart contract info (storage node only)
	// args: [contractAddr multivacaddress.Address]
	// return: *wire.SmartContractInfo
	RPCGetSCInfo
	// RPCSendRawTx sharded request, send raw transaction (storage node only)
	// args [*wire.MsgTx]
	// return: nil
	RPCSendRawTx
)

// RPCReq indicates request from RPCServer to application business logic.
type RPCReq struct {
	sharded     bool
	Event       EventType
	TargetShard shard.Index
	Args        []interface{}
}

// NewNonShardedRPCReq creates a new RPCReq.
func NewNonShardedRPCReq(t EventType, args ...interface{}) *RPCReq {
	return &RPCReq{
		Event:   t,
		Args:    args,
		sharded: false,
	}
}

// NewShardedRPCReq creates a new RPCReq which targets for a particular shard.
func NewShardedRPCReq(t EventType, s shard.Index, args ...interface{}) *RPCReq {
	return &RPCReq{
		Event:       t,
		TargetShard: s,
		Args:        args,
		sharded:     true,
	}
}

// IsSharded returns if the relevant RPCReq targets for any shard or not.
func (r *RPCReq) IsSharded() bool {
	return r.sharded
}

// RPCStatus indicates if a RPCReq to business logic succeeds or not.
type RPCStatus int

const (
	// RPCOk indicates RPCStatus is OK
	RPCOk RPCStatus = iota
	// RPCFail indicates RPCStatus is a failure.
	RPCFail
)

// RPCResp indicates a response from application business logic for the relevant RPCReq
type RPCResp struct {
	Status RPCStatus
	Result interface{}
}

// NewFailedRPCResp creates a new RPCResp with status RPCFail
func NewFailedRPCResp() *RPCResp {
	return &RPCResp{Status: RPCFail}
}

// NewSuccessRPCResp creates a new RPCResp with status RPCOk
func NewSuccessRPCResp(r interface{}) *RPCResp {
	return &RPCResp{Status: RPCOk, Result: r}
}
