package storagenode

import (
	"sync"
	"time"

	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

type storageNodeFetcher struct {
	handling map[uint32]*announce // contains fetch requests those are  being handled
	injected chan *announce
	result   chan *announce
	done     chan struct{}
	quit     chan struct{}
	mu       sync.RWMutex
	log      btclog.Logger
}

const (
	maxHandleTime = time.Millisecond * 5000
)

func newStorageNodeFetcher() *storageNodeFetcher {
	snf := &storageNodeFetcher{
		handling: make(map[uint32]*announce),
		injected: make(chan *announce),
		result:   make(chan *announce),
		done:     make(chan struct{}),
		quit:     make(chan struct{}),
	}
	snf.log = logBackend.Logger("SNFT")
	snf.log.SetLevel(logger.StrgLogLevel)
	return snf
}

func (snf *storageNodeFetcher) inject(anno *announce) {
	snf.injected <- anno
}
func (snf *storageNodeFetcher) handle(anno *announce) {
	anno.Callback(anno.Message)
}

func (snf *storageNodeFetcher) onResult(anno *announce) {
	snf.result <- anno
}

// receives fetch request and determines whether to handle them
func (snf *storageNodeFetcher) injectLoop() {
	for {
		select {
		case <-snf.quit:
			return
		case injected := <-snf.injected:
			if handling, ok := snf.getAnnounce(injected.ID); ok { // same request is being handle
				if handling.ResendRound < injected.ResendRound { // new resend round request, replace old by this new request
					snf.setAnnounce(injected.ID, injected)
					snf.handle(injected)
					snf.log.Debugf("begin to handle msg,replace old,new resend round:%v,msg:%v", injected.ResendRound, injected.Message)
				} else {
					snf.log.Debugf("drop old request,handling resend round:%v,old resend round:%v,old request:%v",
						handling.ResendRound, injected.ResendRound, injected.Message)
				}
			} else { // handle new request
				snf.setAnnounce(injected.ID, injected)
				snf.handle(injected)
				snf.log.Debugf("begin to handle msg:%v", injected.Message)
			}
			snf.done <- struct{}{}
		}
	}
}

// receives results and determines whether to send each result
func (snf *storageNodeFetcher) responseLoop() {
	for {
		select {
		case <-snf.quit:
			return
		case result := <-snf.result:
			if handling, ok := snf.getAnnounce(result.ID); ok {
				if handling.ResendRound == result.ResendRound && time.Since(handling.Time) < maxHandleTime {
					if result.Message != nil {
						handling.PeerReply(result.Message, nil)
					}
					snf.removeAnnounce(result.ID)
					snf.log.Debugf("handle msg done,time used:%v,msg:%v", time.Since(handling.Time), handling.Message)
				} else { // new resend round request is being handled, drop this result
					snf.log.Debugf("handle time out,time used:%v,handling resend round:%v,"+
						"result resend round:%v,msg:%v", time.Since(handling.Time), handling.ResendRound, result.ResendRound, handling.Message)
				}
			} else { // result of new same request has been sent
				snf.log.Debugf("handle time out,result of new same request has been sent,msg:%v", result.Message)
			}
		}
	}
}

// thread safely get map item
func (snf *storageNodeFetcher) getAnnounce(id uint32) (*announce, bool) {
	defer snf.mu.RUnlock()
	snf.mu.RLock()
	announce, ok := snf.handling[id]
	return announce, ok
}

// thread safely set map item
func (snf *storageNodeFetcher) setAnnounce(id uint32, anno *announce) {
	defer snf.mu.Unlock()
	snf.mu.Lock()
	snf.handling[id] = anno
}

// thread safely remove map item
func (snf *storageNodeFetcher) removeAnnounce(id uint32) {
	defer snf.mu.Unlock()
	snf.mu.Lock()
	delete(snf.handling, id)
}

func (snf *storageNodeFetcher) start() {
	go snf.injectLoop()
	go snf.responseLoop()
}

// TODO(fhl):unused now
/*func (snf *storageNodeFetcher) stop() {
	snf.quit <- struct{}{}
	snf.quit <- struct{}{}
}*/
