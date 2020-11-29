package sync

import (
	"testing"
	"time"
)

var startTimeout = false
var stopTimeout = false

func TestSyncTimer_Start(t *testing.T) {
	syncTimer := &syncTimer{}
	syncTimer.start(fakeStartTimeOutFun, 2)
	time.Sleep(3 * time.Second)
	if startTimeout != true {
		t.Error("sync time out, but didn't call time out func")
	}
}

func fakeStartTimeOutFun() {
	startTimeout = true
}
func fakeStopTimeOutFun() {
	stopTimeout = true
}

func TestSyncTimer_Stop(t *testing.T) {
	syncTimer := &syncTimer{}
	syncTimer.start(fakeStopTimeOutFun, 2)
	time.Sleep(1 * time.Second)
	syncTimer.stop()
	time.Sleep(3 * time.Second)
	if stopTimeout == true {
		t.Error("syncTimer stopped, but still call time out func")
	}
}
