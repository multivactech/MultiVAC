// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package controller

import (
	"encoding/hex"

	di "github.com/google/wire"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/iheartbeat"
	"github.com/multivactech/MultiVAC/metrics"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/p2p/connection"
	"github.com/multivactech/MultiVAC/pos/depositpool"
	"github.com/multivactech/MultiVAC/pos/heartbeat"
	"github.com/multivactech/MultiVAC/processor/controller/internal"
	"github.com/multivactech/MultiVAC/processor/miner/consensus"
	"github.com/multivactech/MultiVAC/processor/shared/chain"
)

// NewRootController is responsible for creating RootController and
// processor for each shards.
// Global objects should be passed to each shardProcessor here
func NewRootController(cfg *config.Config, connector connection.Connector) *RootController {
	panic(di.Build(di.NewSet(
		provideRootController,
		provideValidationInfo,
		metrics.ProvideMonitorMetrics,
		internal.ProvideShardManager,

		// heartbeat manager
		provideHeartBeatManager,
		di.Bind(new(heartbeatEnabler), new(iheartbeat.HeartBeat)),

		// block chain
		chain.NewService,
		di.Bind(new(iblockchain.BlockChain), new(*chain.Service)),

		// deposit pool
		di.Value(depositpool.MemoryMode),
		depositpool.NewThreadedDepositPool,
	)))
}

func provideRootController(cfg *config.Config, shardMgr *internal.ThreadedShardManager,
	dPool idepositpool.DepositPool, heartbeatMgr heartbeatEnabler) *RootController {
	ctl := &RootController{
		reshardHandler:   shardMgr,
		shardRouter:      shardMgr,
		cfg:              cfg,
		depositPool:      dPool,
		heartBeatManager: heartbeatMgr,
	}

	// Set the default state of shard and start then enabled shard.
	shardMgr.Init()
	return ctl
}

func provideHeartBeatManager(cfg *config.Config, info *internal.ValidationInfo,
	dPool idepositpool.DepositPool, metrics *metrics.Metrics) iheartbeat.HeartBeat {
	return heartbeat.NewThreadHeartBeatManager(info.Pk, info.Sk, dPool, metrics)
}

func provideValidationInfo(cfg *config.Config) *internal.ValidationInfo {
	info := &internal.ValidationInfo{
		Selector: consensus.NewSortitionor(uint32(cfg.ExpNumShard), uint32(cfg.LeaderRate)),
	}

	info.Vrf = &vrf.Ed25519VRF{}
	var err error
	//TODO(MH): this is only for miner client.
	if cfg.Sk != "" {
		if info.Pk, info.Sk, err = vrf.GeneratePkWithSk(cfg.Sk); err != nil {
			rootCtrlLogger.Errorf("fail to generate key pair")
			return nil
		}
		cfg.Pk = hex.EncodeToString(info.Pk)

		if cfg.RewardAddr == "" {
			cfg.RewardAddr = cfg.Pk
		}
		addr, err := multivacaddress.StringToUserAddress(cfg.Pk)
		if err != nil {
			rootCtrlLogger.Errorf("invalid reward address")
			return nil
		}
		cfg.RewardAddr = addr.String()
	} else {
		rootCtrlLogger.Errorf("please import private key")
		return nil
	}
	return info
}
