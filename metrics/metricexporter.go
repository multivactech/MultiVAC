/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metric 是全局监控实例
var Metric *Metrics

func init() {
	Metric = &Metrics{
		ShardHeight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multivac_shard_height",
				Help: "Shard Height",
			},
			[]string{"shard_id"},
		),
		StorageShardHeight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multivac_storage_shard_height",
				Help: "Storage Shard Height",
			},
			[]string{"shard_id"},
		),
		PendingTxNum: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multivac_pending_tx_cnt",
				Help: "Pending tx number in Tx Pool",
			},
			[]string{"shard_id"},
		),
		BroadcastedMsgCnt: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "multivac_broadcasted_msg_cnt",
				Help: "How many messages have been broadcasted since start.",
			},
			[]string{"shard_id", "command"},
		),
		ShardEnableMinerNum: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multivac_shard_enabled_miner_num",
				Help: "How many miner are service in each shard.",
			},
			[]string{"shard_id"},
		),
		ConfirmedTxCnt: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "multivac_confirmed_tx_cnt",
				Help: "How many transacations have been confirmed since start.",
			},
			[]string{"shard_id"},
		),
		ShardPresence: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multivac_shard_presence",
				Help: "This indicates the presence of this instance in specific shard. 1 means it's in that shard.",
			},
			[]string{"shard_id"},
		),
		ConnectedPeerNum: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "multivac_connected_peer_num",
				Help: "How many peers are connected from this node currenlty.",
			},
		),
		OnlineMiner: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "multivac_minerNum",
				Help: "Number of miner",
			},
		),
		EnableShardsNum: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "multivac_enabled_shards_num",
				Help: "Number of enabled shard",
			},
		),
	}
	registerMetrics(Metric)
}

// Metrics used for prometheus
type Metrics struct {
	// 各个分片级别的metrics
	ShardHeight         *prometheus.GaugeVec
	StorageShardHeight  *prometheus.GaugeVec
	PendingTxNum        *prometheus.GaugeVec
	ShardPresence       *prometheus.GaugeVec
	ShardEnableMinerNum *prometheus.GaugeVec
	BroadcastedMsgCnt   *prometheus.CounterVec
	ConfirmedTxCnt      *prometheus.CounterVec
	// 全局的metrics 小贴士: prometheus.Gauge是接口类型，如果用指针*prometheus.Gauge的话，无法正常调用方法
	OnlineMiner      prometheus.Gauge
	ConnectedPeerNum prometheus.Gauge
	EnableShardsNum  prometheus.Gauge
}

func registerMetrics(m *Metrics) {
	prometheus.MustRegister(m.ShardHeight, m.StorageShardHeight, m.PendingTxNum,
		m.BroadcastedMsgCnt, m.ConfirmedTxCnt, m.ShardPresence, m.ConnectedPeerNum,
		m.OnlineMiner, m.EnableShardsNum, m.ShardEnableMinerNum)
}

// ProvideMonitorMetrics returns the global variable Metric.
func ProvideMonitorMetrics() *Metrics {
	return Metric
}
