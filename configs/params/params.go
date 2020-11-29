// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
package params

const (
	// LeveldbCacheMb defines the write buffer size of leveldb.
	LeveldbCacheMb = 64

	// LeveldbBlockCacheMb defines the capacity of the leveldb's 'sorted table' block caching.
	LeveldbBlockCacheMb = 128

	// LeveldbFileNumber defines the capacity of the leveldb's open files caching.
	LeveldbFileNumber = 64

	// SecurityLevel Determines whether the probability of being selected in the shard.
	// rand < 1 - (1 - securityLevel)^depositUnit, as mtv yellow paper.
	SecurityLevel = 0.8

	// ReshardRoundThreshold is the number of rounds that trigger resharding.
	// TODO(issue #127, tianhongce): Set a more reasonable threshold.
	ReshardRoundThreshold = 20
	// ReshardPrepareRoundThreshold is the number of perparation rounds before resharding.
	// More details about the preparation of the resharding, please see the controller/README.md.
	ReshardPrepareRoundThreshold = 5
	// RelayCache define the size of lru map.
	RelayCache = 1000000
	// DepositValue is defined that how much  mtv that user has deposited can have qualifications to mine.
	DepositValue = 1000000
	// LeaderRateBase defines the number of miner who may mine block.
	LeaderRateBase = 20
)

var (
	// OnlineTimeS defines that the duration of a miner must be online.
	OnlineTimeS int64 = 180

	// TimeDeviationS defines the acceptable deviation of time for heartbeat.
	TimeDeviationS int64 = 100

	// SendHeartbeatIntervalS defines the interval to send heartbeat.
	SendHeartbeatIntervalS int64 = 60
)
