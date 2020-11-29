package controller

// TODO(huangsz): re-enable / re-write the tests
//import (
//	"reflect"
//	"testing"
//
//	"github.com/multivactech/MultiVAC/configs/config"
//	"github.com/multivactech/MultiVAC/model/shard"
//	"github.com/multivactech/MultiVAC/processor/shared/message"
//	"github.com/multivactech/MultiVAC/rpc/btcjson"
//	"github.com/stretchr/testify/assert"
//)
//
//func TestController_HandleRpcReq_NonSharded(t *testing.T) {
//	_, _ = config.LoadConfig()
//	tests := []struct {
//		desc        string
//		req         *message.RPCReq
//		expected    *message.RPCResp
//		resultEqual func(a, o interface{}) bool
//	}{
//		{
//			desc: "RPCGetOutState",
//			req:  message.NewNonShardedRPCReq(message.RPCGetAllShardsInfo),
//			expected: message.NewSuccessRPCResp(&btcjson.ShardsInfo{
//				NumShards: len(shardListForTest),
//				Shards: []btcjson.ShardInfo{
//					{
//						Index:   shard.IDToShardIndex(shardListForTest[0]),
//						Enabled: true,
//					},
//					{
//						Index:   shard.IDToShardIndex(shardListForTest[1]),
//						Enabled: true,
//					},
//					{
//						Index:   shard.IDToShardIndex(shardListForTest[2]),
//						Enabled: true,
//					},
//					{
//						Index:   shard.IDToShardIndex(shardListForTest[3]),
//						Enabled: true,
//					},
//				},
//			}),
//			resultEqual: func(a, o interface{}) bool {
//				return reflect.DeepEqual(a, o)
//			},
//		},
//	}
//
//	ctrl := creatFakeStorageController(&config.Config{
//		StorageNode:        true,
//		IsOneOfFirstMiners: false,
//		IsSharded:          false,
//		Shards:             shardListForTest,
//	})
//	a := assert.New(t)
//	for _, test := range tests {
//		r := ctrl.HandleRPCReq(test.req)
//
//		a.Equalf(r.Status, test.expected.Status, "HandleRPCReq %v", test.desc)
//		a.Condition(
//			func() bool { return test.resultEqual(test.expected.Result, r.Result) },
//			"HandleRPCReq %v, Result expected: %v, got: %v",
//			test.desc, test.expected.Result, r.Result)
//	}
//}
