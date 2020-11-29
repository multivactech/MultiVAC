package appblockchain

import (
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

// Recv implements the message.Recv interface to subscribe events.
func (abci *appBlockChain) Recv(e message.Event) {
	switch e.Topic {
	case message.EvtLedgerFallBehind:
		abci.fetchNewLedger()
	}
}
