package mock

import "github.com/multivactech/MultiVAC/model/wire"

// HeartBeat for easier testing.
type HeartBeat struct {
	TestNodeNum int
}

// Receive implement HeartBeat interface
func (thbm *HeartBeat) Receive(msg *wire.HeartBeatMsg) {

}

// Has implement HeartBeat interface
func (thbm *HeartBeat) Has(pk []byte) bool {
	return true
}

// PerceivedCount implement HeartBeat interface
func (thbm *HeartBeat) PerceivedCount() int {
	return thbm.TestNodeNum
}

// Start implement HeartBeat interface
func (thbm *HeartBeat) Start() {
}

// Stop implement HeartBeat interface
func (thbm *HeartBeat) Stop() {
}

// CheckConfirmation implements HeatBeat interface.
func (thbm *HeartBeat) CheckConfirmation(msg *wire.MsgBlockConfirmation) bool {
	return true
}
