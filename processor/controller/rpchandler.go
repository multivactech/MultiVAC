/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package controller

import (
	"sort"

	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/rpc/btcjson"
)

// HandleRPCReq should be the root entry for all RPC requests to get some internal state or trigger internal operation.
func (ctrl *RootController) HandleRPCReq(req *message.RPCReq) *message.RPCResp {
	if req.IsSharded() {
		if sp := ctrl.shardRouter.GetShardController(req.TargetShard); sp != nil {
			return sp.HandleRPCReq(req)
		}
	} else {
		return ctrl.handleRPCInternal(req)
	}
	return message.NewFailedRPCResp()
}

func (ctrl *RootController) handleRPCInternal(req *message.RPCReq) *message.RPCResp {
	switch req.Event {
	case message.RPCGetAllShardsInfo:
		info := &btcjson.ShardsInfo{
			NumShards: len(ctrl.shardRouter.GetAllShardControllers()),
		}
		for _, sp := range ctrl.shardRouter.GetAllShardControllers() {
			info.Shards = append(info.Shards, btcjson.ShardInfo{Index: sp.ShardIndex(), Enabled: sp.IsEnabled()})
		}
		sort.Sort(info)
		return message.NewSuccessRPCResp(info)
	}
	return message.NewFailedRPCResp()
}
