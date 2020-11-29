package iconsensus

// CEShardStateListener is used to control the status of consensus.
type CEShardStateListener interface {
	OnShardDisabled()
}
