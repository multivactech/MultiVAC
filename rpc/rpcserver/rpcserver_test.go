package rpcserver

import (
	"strconv"
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/rpc/btcjson"
	m "github.com/multivactech/MultiVAC/rpc/rpcserver/mocks"

	"github.com/stretchr/testify/assert"
)

const (
	testDataSpan = 10
	defaultNum   = 1
)

func TestGetBlockInfo(t *testing.T) {
	a := assert.New(t)
	req := message.NewShardedRPCReq(message.RPCGetBlockInfo, defaultNum, int64(defaultNum))
	cmd := &btcjson.QueryBlockCmd{
		StartHeight: int64(defaultNum),
		ShardID:     strconv.Itoa(defaultNum),
	}

	tests := buildTestData(cmd, fakeBlockArray(wire.BlockHeight(defaultNum), wire.BlockHeight(defaultNum+testDataSpan), defaultNum), req)

	for _, x := range tests {
		obtainResp, err := handleQueryBlockInfo(x.req.rpcServer, x.req.cmd.(*btcjson.QueryBlockCmd), nil)
		a.Nil(err, "got error when query block")
		obtainBlock := obtainResp.(btcjson.BlockInfo).Block
		a.Equalf(x.expect.Result.([]*wire.MsgBlock), obtainBlock, "handle query block info")
	}
}

func TestGetBlockSliceInfo(t *testing.T) {
	a := assert.New(t)
	req := message.NewShardedRPCReq(message.RPCGetBlockSliceInfo, defaultNum, int64(defaultNum), int64(testDataSpan))
	cmd := &btcjson.QueryBlockSliceCmd{
		StartHeight: defaultNum,
		EndHeight:   testDataSpan,
		ShardID:     strconv.Itoa(defaultNum),
	}

	tests := buildTestData(cmd, fakeBlockArray(wire.BlockHeight(defaultNum), wire.BlockHeight(testDataSpan), defaultNum), req)
	for _, x := range tests {
		obtainResp, err := handleQueryBlockSliceInfo(x.req.rpcServer, x.req.cmd.(*btcjson.QueryBlockSliceCmd), nil)
		a.Nil(err, "got error when query block slice")
		obtainBlockSlice := obtainResp.(btcjson.BlockSliceInfo).Block
		a.Equalf(x.expect.Result.([]*wire.MsgBlock), obtainBlockSlice, "handle query block slice info")
	}
}

func TestGetShardInfo(t *testing.T) {
	a := assert.New(t)
	req := message.NewNonShardedRPCReq(message.RPCGetAllShardsInfo)

	tests := buildTestData(nil, fakeShardsInfo(), req)

	for _, x := range tests {
		shardInfoResp, err := handleQueryShardInfo(x.req.rpcServer, x.req.cmd, nil)
		a.Nil(err, "got error when query shard info")
		obtainShardsInfo := shardInfoResp.(btcjson.ShardsInfo)
		a.Equalf(*x.expect.Result.(*btcjson.ShardsInfo), obtainShardsInfo, "handle query shards info")
	}

}

func TestGetSlimBlockInfo(t *testing.T) {
	a := assert.New(t)
	req := message.NewShardedRPCReq(message.RPCGetSlimBlockInfo, defaultNum, shard.Index(defaultNum), int64(defaultNum))
	cmd := &btcjson.QuerySlimBlockCmd{
		Height:    int64(defaultNum),
		ShardID:   strconv.Itoa(defaultNum),
		ToShardID: strconv.Itoa(defaultNum),
	}
	tests := buildTestData(cmd, fakeSlimBlocks(defaultNum, int64(defaultNum), int64(defaultNum+testDataSpan)), req)

	for _, x := range tests {
		obtainResp, err := handleQuerySlimBlockInfo(x.req.rpcServer, x.req.cmd, nil)
		a.Nil(err, "got error when query slim block info")
		obtainSlimBlock := obtainResp.(btcjson.SlimBlockInfo).SlimBlock
		a.Equalf(x.expect.Result.([]*wire.SlimBlock), obtainSlimBlock, "handle query slim block info")
	}
}

func TestGetSlimBlockSliceInfo(t *testing.T) {
	a := assert.New(t)
	req := message.NewShardedRPCReq(message.RPCGetSlimBlockSliceInfo, defaultNum, shard.Index(defaultNum), int64(defaultNum), int64(defaultNum+testDataSpan))
	cmd := &btcjson.QuerySlimBlockSliceCmd{
		StartHeight: defaultNum,
		EndHeight:   defaultNum + testDataSpan,
		ShardID:     strconv.Itoa(defaultNum),
		ToShardID:   strconv.Itoa(defaultNum),
	}

	tests := buildTestData(cmd, fakeSlimBlocks(defaultNum, defaultNum, defaultNum+testDataSpan), req)

	for _, x := range tests {
		obtainResp, err := handleQuerySlimBlockSliceInfo(x.req.rpcServer, x.req.cmd, nil)
		a.Nil(err, "got error when query slim block info")
		obtainSlimBlock := obtainResp.(btcjson.SlimBlockSliceInfo).SlimBlock
		a.Equalf(x.expect.Result.([]*wire.SlimBlock), obtainSlimBlock, "handle query slim block info")
	}
}

type testTable struct {
	req    request
	expect *message.RPCResp
}

type request struct {
	rpcServer *RPCServer
	cmd       interface{}
}

func fakeBlock(shard shard.Index, hgt wire.BlockHeight) *wire.MsgBlock {
	header := wire.BlockHeader{ShardIndex: shard, Height: int64(hgt)}
	return &wire.MsgBlock{
		Header: header,
		Body:   &wire.BlockBody{},
	}
}
func fakeBlockArray(startHeight wire.BlockHeight, endHeight wire.BlockHeight, defaultShardIndex shard.Index) []*wire.MsgBlock {
	var blockArr []*wire.MsgBlock
	for i := startHeight; i <= endHeight; i++ {
		blockArr = append(blockArr, fakeBlock(defaultShardIndex, i))
	}
	return blockArr
}

func fakeShardsInfo() *btcjson.ShardsInfo {
	return &btcjson.ShardsInfo{
		NumShards: 4,
		Shards: []btcjson.ShardInfo{{
			Index:   0,
			Enabled: true,
		}, {
			Index:   1,
			Enabled: true,
		}, {
			Index:   2,
			Enabled: false,
		}, {
			Index:   3,
			Enabled: true,
		}},
	}
}

func fakeSlimBlocks(shard shard.Index, shgt int64, ehgt int64) []*wire.SlimBlock {
	var slimBlockArr []*wire.SlimBlock

	for i := shgt; i <= ehgt; i++ {
		slimBlockArr = append(slimBlockArr, &wire.SlimBlock{
			ToShard: shard,
			Header: wire.BlockHeader{
				ShardIndex: shard,
				Height:     i,
			},
			LedgerInfo: wire.LedgerInfo{},
		})
	}
	return slimBlockArr
}

func buildTestData(cmd interface{}, expectData interface{}, req *message.RPCReq) (tests []testTable) {
	mockHandler := new(m.MockRpcHandler)
	mockHandler.On("HandleRPCReq", req).Return(message.NewSuccessRPCResp(expectData))

	for i := 0; i < testDataSpan; i++ {
		tests = append(tests, testTable{
			req: request{
				rpcServer: &RPCServer{
					cfg: Config{
						businessController: mockHandler,
					},
				},
				cmd: cmd,
			},
			expect: message.NewSuccessRPCResp(expectData),
		})
	}
	return tests
}
