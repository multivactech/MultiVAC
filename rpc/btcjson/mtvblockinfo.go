package btcjson

import "github.com/multivactech/MultiVAC/model/wire"

// BlockInfo contains complete blocks from the startHeight.
// TODO: Maybe move this to another MTV specific package
type BlockInfo struct {
	Block       []*wire.MsgBlock
	StartHeight int64
}

// SlimBlockInfo contains the SlimBlocks from  startHeight.
type SlimBlockInfo struct {
	SlimBlock []*wire.SlimBlock
	Height    int64
}

// BlockSliceInfo contains blocks from startHeight to endHeight.
type BlockSliceInfo struct {
	Block       []*wire.MsgBlock
	StartHeight int64
	EndHeight   int64
}

// SlimBlockSliceInfo contains slimBlocks from startHeight to endHeight.
type SlimBlockSliceInfo struct {
	SlimBlock   []*wire.SlimBlock
	StartHeight int64
	EndHeight   int64
}
