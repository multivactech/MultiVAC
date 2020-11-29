/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"time"

	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/wire"
)

const (
	// ResendRoundBitNum defines the number of bit of resend round in the message id
	// resend round num is stored in the last ResendRoundBitNum bits of the message id
	ResendRoundBitNum = 3
	fetchInterval     = 100 * time.Millisecond
	fetchWaitTime     = 1 * time.Second
	fetchTimeOut      = 5 * time.Second
)

// Mask is a uint32 number whose last ResendRoundBitNum bit is 0 and the rest is 1
var Mask = ^uint32(0) - (1<<ResendRoundBitNum - 1)

// getResendRoundByMsgId returns the last ResendRoundBitNum bit of msgID
func getResendRoundByMsgID(msgID uint32) byte {
	return byte(msgID & (1<<ResendRoundBitNum - 1))
}

// getAnnounceIDByMsgID sets the last ResendRoundBitNum bit of msgID to 0
func getAnnounceIDByMsgID(msgID uint32) uint32 {
	return msgID & Mask
}

type abciFetcher struct {
	announced []*announce          // store fetch messages those are ready to be send
	fetching  map[uint32]*announce // store fetch messages those have been send and not receive response
	injected  chan *announce
	response  chan *announce
	done      chan struct{}
	quit      chan struct{}
	log       btclog.Logger
}

func newABCIFetcher() *abciFetcher {
	ft := &abciFetcher{
		announced: []*announce{},
		fetching:  make(map[uint32]*announce),
		injected:  make(chan *announce),
		response:  make(chan *announce),
		done:      make(chan struct{}),
		quit:      make(chan struct{}),
	}
	ft.log = logBackend.Logger("ABCFT")
	ft.log.SetLevel(logger.AbciLogLevel)
	return ft
}

func (af *abciFetcher) fetchMessage(msg wire.Message, callback func(msg wire.Message)) {
	callback(msg)
}

func (af *abciFetcher) onMessage(msg wire.Message) {
	switch msg := msg.(type) {
	case *wire.MsgReturnInit:
		af.response <- newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	case *wire.MsgReturnTxs:
		af.response <- newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	case *wire.SmartContractInfo:
		af.response <- newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	}
}

func (af *abciFetcher) inject(anno *announce) {
	af.injected <- anno
}

func (af *abciFetcher) handleMessage(msg wire.Message, callback func(msg wire.Message) error) {
	switch msg := msg.(type) {
	case *wire.MsgReturnTxs:
		if err := callback(msg); err != nil {
			af.log.Errorf("failed to handle MsgReturnTxs, err:%v", err)
		}
	case *wire.MsgReturnInit:
		if err := callback(msg); err != nil {
			af.log.Errorf("failed to handle MsgInitABCIData, err:%v", err)
		}
	case *wire.SmartContractInfo:
		if err := callback(msg); err != nil {
			af.log.Errorf("failed to handle SmartContractInfo, err:%v", err)
		}
	}
}

// setResendRound set resend round of the announce and add the resend round number to msg id
func setResendRound(anno *announce, resendRound byte) {
	switch msg := anno.Message.(type) {
	case *wire.MsgFetchTxs:
		msg.MsgID += uint32(resendRound)
		anno.ResendRound = resendRound
	case *wire.MsgFetchInit:
		msg.MsgID += uint32(resendRound)
		anno.ResendRound = resendRound
	case *wire.MsgFetchSmartContractInfo:
		msg.MsgID += uint32(resendRound)
		anno.ResendRound = resendRound
	}
}

func (af *abciFetcher) loop() {
	fetchTimer := time.NewTimer(fetchInterval)
	for {
		select {
		case <-af.quit:
			return
		case inject := <-af.injected:
			if announcing, ok := af.fetching[inject.ID]; ok { // this message has been fetched before and have not receive response
				if time.Since(announcing.Time) > fetchTimeOut { // previous fetching message time out
					if announcing.ResendRound < 1<<ResendRoundBitNum-1 {
						setResendRound(inject, announcing.ResendRound+1) // set resend round in inject
					} else {
						// af.log.Errorf("same msg has been resend more than %v times,msg:%v", announcing.ResendRound, announcing.Message)
					}
					delete(af.fetching, inject.ID)
					af.announced = append(af.announced, inject)
					af.log.Debugf("old fetch request time out,old msg:%v,old resendRound:%v", announcing.Message, announcing.ResendRound)
				} else { // don't need to send this message before last request time out
					af.log.Debugf("fetch request has been send recently,time interval:%v, drop this request:%v", time.Since(announcing.Time), inject.Message)
				}
			} else { // announce is not fetching ，judge if there is a same announce in af.announced
				isInAnnounced := false
				for _, v := range af.announced {
					if v.ID == inject.ID {
						isInAnnounced = true
						break
					}
				}
				if !isInAnnounced {
					af.announced = append(af.announced, inject)
				}
			}
		case <-fetchTimer.C:
			for i := range af.announced {
				anno := af.announced[i]
				if time.Since(anno.Time) > fetchWaitTime { // skip announce that wait time out
					af.log.Debugf("wait in queue time out,drop message:%v,wait time:%v", anno.Message, time.Since(anno.Time))
				} else {
					af.log.Debugf("fetch message:%v,resend round:%v", anno.Message, anno.ResendRound)
					af.fetchMessage(anno.Message, anno.BroadFuncCallback)
					af.announced = af.announced[i+1:]
					af.fetching[anno.ID] = anno
					break
				}
			}
			fetchTimer.Reset(fetchInterval)
		case res := <-af.response:
			if anno, ok := af.fetching[res.ID]; ok {
				if res.ResendRound == anno.ResendRound { // announce is just the fetching request's response
					if time.Since(anno.Time) < fetchTimeOut {
						af.log.Debugf("handle msg,time from fetch to response:%v,msg:%v", time.Since(anno.Time), res.Message)
						af.handleMessage(res.Message, anno.HandleFuncCallback)
					} else {
						af.log.Debugf("response message time out,time from fetch to response:%v,msg:%v", time.Since(anno.Time), res.Message)
					}
					delete(af.fetching, res.ID)
				} else { // this is an old fetch request's response
					af.log.Debugf("response message:%v time out,new request has been send,"+
						"old round:%d new round:%d", res.Message, res.ResendRound, anno.ResendRound)
				}
			} else { // the request of this response has been remove from fetching map
				af.log.Debugf("response message:%v time out,request has been response or "+
					"new request is waiting to be send.resend round:%v", res.Message, res.ResendRound)
			}
			af.done <- struct{}{}
		}

	}

}

func (af *abciFetcher) start() {
	go af.loop()
}

// Todo: is unused now
/*func (af *abciFetcher) stop(){
	af.quit <- struct{}{}
}*/
