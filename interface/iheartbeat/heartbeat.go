package iheartbeat

import "github.com/multivactech/MultiVAC/model/wire"

// HeartBeat is used to deal with heartbeat message.
type HeartBeat interface {
	Has(pk []byte) bool
	PerceivedCount() int
	Start()
	Stop()
	CheckConfirmation(confirmation *wire.MsgBlockConfirmation) bool
}
