// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package controller

import (
	"encoding/hex"
	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/configs/config"
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

// Injectors from injector.go:

func NewRootController(cfg *config.Config, connector connection.Connector) *RootController {
	validationInfo := provideValidationInfo(cfg)
	metricsMetrics := metrics.ProvideMonitorMetrics()
	service := chain.NewService()
	storageMode := _wireStorageModeValue
	depositPool := depositpool.NewThreadedDepositPool(service, storageMode)
	heartBeat := provideHeartBeatManager(cfg, validationInfo, depositPool, metricsMetrics)
	threadedShardManager := internal.ProvideShardManager(cfg, validationInfo, metricsMetrics, service, depositPool, heartBeat, connector)
	rootController := provideRootController(cfg, threadedShardManager, depositPool, heartBeat)
	return rootController
}

var (
	_wireStorageModeValue = depositpool.MemoryMode
)

// injector.go:

func provideRootController(cfg *config.Config, shardMgr *internal.ThreadedShardManager,
	dPool idepositpool.DepositPool, heartbeatMgr heartbeatEnabler) *RootController {
	ctl := &RootController{
		reshardHandler:   shardMgr,
		shardRouter:      shardMgr,
		cfg:              cfg,
		depositPool:      dPool,
		heartBeatManager: heartbeatMgr,
	}

	shardMgr.Init()
	return ctl
}

func provideHeartBeatManager(cfg *config.Config, info *internal.ValidationInfo,
	dPool idepositpool.DepositPool, metrics2 *metrics.Metrics) iheartbeat.HeartBeat {
	return heartbeat.NewThreadHeartBeatManager(info.Pk, info.Sk, dPool, metrics2)
}

func provideValidationInfo(cfg *config.Config) *internal.ValidationInfo {
	info := &internal.ValidationInfo{
		Selector: consensus.NewSortitionor(uint32(cfg.ExpNumShard), uint32(cfg.LeaderRate)),
	}

	info.Vrf = &vrf.Ed25519VRF{}
	var err error

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
