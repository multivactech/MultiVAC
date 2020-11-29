package params

// All temporary feature flags.
const (
	// StorageNodeShardConfig If enabled, people can configure supported shards in storage node, otherwise it is all-shards.
	StorageNodeShardConfig = false
	//EnableCoinbaseTx allows to hava a transaction from coinbase.
	EnableCoinbaseTx = true

	// EnableStorageReward enable means that storage node has reward.
	EnableStorageReward = false

	// EnableTestnetThreshold require users to mortgage to qualify for mining.
	EnableTestnetThreshold = false
)
