// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package miner

import (
	di "github.com/google/wire"
	"github.com/multivactech/MultiVAC/interface/iabci"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/processor/miner/appblockchain"
	"github.com/multivactech/MultiVAC/processor/miner/consensus"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// NewProcessor creates a miner.Processor instance which responsible for handling miner
// relevant business logic.
func NewProcessor(blockChain iblockchain.BlockChain, params *consensus.Params, shardIndex shard.Index,
	selector iconsensus.Sortitionor, dPool idepositpool.DepositPool, pubSubMgr *message.PubSubManager) *Processor {
	panic(di.Build(di.NewSet(
		provideProcessor,
		appblockchain.NewAppBlockChain,
		consensus.NewExecutor)))
}

func provideProcessor(shard shard.Index, abc iabci.AppBlockChain,
	blockChain iblockchain.BlockChain, c iconsensus.Consensus) *Processor {
	pcr := &Processor{shard: shard, blockChain: blockChain}
	pcr.abci = abc
	pcr.consensusExecutor = c
	return pcr
}
