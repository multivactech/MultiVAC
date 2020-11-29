// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package storagenode

import (
	"fmt"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"

	di "github.com/google/wire"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/shard"
)

// StorageDir is defined as string type
type StorageDir string

// NewStorageProcessor returns a new instance of StorageProcessor.
func NewStorageProcessor(shard2 shard.Index, dataDir StorageDir, blockChain iblockchain.BlockChain,
	dPool idepositpool.DepositPool, heartBeatManager iheartbeat.HeartBeat) *Processor {
	panic(di.Build(di.NewSet(
		provideProcessor,
		newStorageNode,
		newThreadedStorageNode,
		provideLogger,
	)))
}

func provideProcessor(logger btclog.Logger, sn *threadedStorageNode) *Processor {
	return &Processor{sn: sn, logger: logger}
}

func provideLogger(shard shard.Index) btclog.Logger {
	return logBackend.Logger(
		fmt.Sprintf("%s-%d", logger.StorageLoggerTag, shard.GetID()))
}
