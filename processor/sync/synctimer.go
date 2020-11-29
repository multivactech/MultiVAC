package sync

import (
	"time"
)

// syncTimer is a timer for sync, the timer will start when start to sync.
// If sync isn't completed within a certain period, the timeour func will be triggered.
type syncTimer struct {
	ticker *time.Ticker
	isStop chan bool
}

// syncTimer start, onTimeOut is the timeout func.
func (st *syncTimer) start(onTimeOut timerCallback, timeOutInSec int) {
	st.ticker = time.NewTicker(time.Duration(timeOutInSec) * time.Second)
	if st.isStop == nil {
		st.isStop = make(chan bool, 1)
	}
	go st.timeoutRoutine(onTimeOut)
}

// syncTimer stop.
func (st *syncTimer) stop() {
	st.stopTicker()
}

func (st *syncTimer) stopTicker() {
	if st.ticker != nil {
		st.ticker.Stop()
	}
	if st.isStop != nil {
		st.isStop <- true
	}
}
func (st *syncTimer) timeoutRoutine(onTimeOut func()) {
	for {
		select {
		case <-st.ticker.C:
			log.Errorf("Sync time out!")
			onTimeOut()
			st.stopTicker()
		case <-st.isStop:
			return
		}
	}
}

type timerCallback func()
