package mock

import "time"

// FakeTicker for easier testing.
type FakeTicker struct {
	C       chan time.Time
	Period  time.Duration
	stopped bool
	now     time.Time
}

// Chan implement ticker interface
func (ft *FakeTicker) Chan() <-chan time.Time {
	return ft.C
}

// Stop implement ticker interface
func (ft *FakeTicker) Stop() {
	ft.stopped = true
}

// Tick sends the tick time to the ticker channel after every period.
// Tick events are discarded if the underlying ticker channel does
// not have enough capacity.
func (ft *FakeTicker) Tick() {
	if ft.stopped {
		return
	}

	ft.now = ft.now.Add(ft.Period)
	ft.C <- ft.now
}
