package autoupdater

import (
	"bytes"
	"log"
	"net/url"
	"time"
)

// poller polls the given URL periodically and sends messages
// to the `changes` channel when the content of the URL changes.
type poller struct {
	url      url.URL
	interval time.Duration
	ticker   *time.Ticker
	changes  chan []byte
	quit     chan struct{}

	lastResult []byte
}

func newPoller(url url.URL, interval time.Duration) *poller {
	return &poller{
		url:      url,
		interval: interval,
		changes:  make(chan []byte),
		quit:     make(chan struct{}),
	}
}

func (p *poller) Start() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	p.ticker = time.NewTicker(p.interval)
	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.Poll()
			case <-p.quit:
				return
			}
		}
	}()
}

func (p *poller) Poll() {
	res, err := download(p.url.String())
	if err != nil {
		log.Println(err)
		return
	}
	if !bytes.Equal(res, p.lastResult) {
		p.changes <- res
		p.lastResult = res
	}
}

func (p *poller) Stop() {
	p.quit <- struct{}{}
	p.ticker.Stop()
}
