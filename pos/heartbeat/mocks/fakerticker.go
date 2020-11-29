package mocks

import (
	"time"
)

// FakeTicker for easier testing.
type FakeTicker struct {
	C       chan time.Time
	Stopped bool
}

// Chan implement ticker interface
func (ft *FakeTicker) Chan() <-chan time.Time {
	return ft.C
}

// Stop implement ticker interface
func (ft *FakeTicker) Stop() {
	ft.Stopped = true
}

// Tick sends the tick time to the ticker channel after every period.
// Tick events are discarded if the underlying ticker channel does
// not have enough capacity.
func (ft *FakeTicker) Tick() {
	if ft.Stopped {
		return
	}
	ft.C <- time.Now()
}
