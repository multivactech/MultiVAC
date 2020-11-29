package mock

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// BcProducer for easier testing.
type BcProducer struct {
	shard       shard.Index
	blockHeader *wire.BlockHeader
}

// NewMockBcProducer return BcProducer.
func NewMockBcProducer(shard shard.Index) *BcProducer {
	return &BcProducer{shard: shard}
}

// GenerateBlockConfirmation generate blockconfirmation message by bba final message.
func (mbcp *BcProducer) GenerateBlockConfirmation(message *wire.MsgBinaryBAFin) *wire.MsgBlockConfirmation {
	v := message.GetByzAgreementValue()
	return wire.NewMessageBlockConfirmation(
		mbcp.shard,
		int32(message.GetRound()),
		int32(message.GetStep()),
		&v,
		mbcp.blockHeader,
		[]*wire.MsgBinaryBAFin{message},
	)
}

// SaveHeader cache block header.
func (mbcp *BcProducer) SaveHeader(header *wire.BlockHeader) {
	mbcp.blockHeader = header
}
