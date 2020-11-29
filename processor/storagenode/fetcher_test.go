package storagenode

import (
	"github.com/multivactech/MultiVAC/p2p/peer"
	"testing"
	"time"

	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/stretchr/testify/assert"
)

var fakePeerReply peer.Reply = func(msg wire.Message, downChan chan<- struct{}) {}

func TestFetcherNormalResponse(t *testing.T) {
	defer clear()
	sn := createTestStorageNode(t)
	sn.fetcher.handling[8] = &announce{ResendRound: 0, PeerReply: fakePeerReply, Time: time.Now()}
	sn.fetcher.onResult(newResultAnnounce(&wire.MsgReturnTxs{
		MsgID: 8,
	}))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 0, len(sn.fetcher.handling))
}

func TestFetcherHandleTimeOut(t *testing.T) {
	defer clear()
	sn := createTestStorageNode(t)
	sn.fetcher.handling[8] = &announce{
		Time: time.Now().AddDate(0, 0, -1),
	}
	sn.fetcher.onResult(newResultAnnounce(&wire.MsgReturnTxs{ // time out
		MsgID: 8,
	}))
	assert.Equal(t, 1, len(sn.fetcher.handling))
	sn.fetcher.handling[8] = &announce{}
	sn.fetcher.onResult(newResultAnnounce(&wire.MsgReturnTxs{ // new resend round
		MsgID: 9,
	}))
	assert.Equal(t, 1, len(sn.fetcher.handling))
	sn.fetcher.onResult(newResultAnnounce(&wire.MsgReturnTxs{ // not exist
		MsgID: 17,
	}))
	assert.Equal(t, 1, len(sn.fetcher.handling))
}

func TestFetcherInject(t *testing.T) {
	defer clear()
	sn := createTestStorageNode(t)
	sn.fetcher.handling[8] = &announce{ResendRound: 2}
	sn.fetcher.inject(newFetchAnnounce(&wire.MsgFetchInit{MsgID: 17}, fakePeerReply, sn.onfetchMsg)) // new fetch msg
	<-sn.fetcher.done
	assert.Equal(t, 2, len(sn.fetcher.handling))
	time.Sleep(time.Millisecond * 50)                                                               // wait last inject msg handle done
	assert.Equal(t, 1, len(sn.fetcher.handling))                                                    // one left
	sn.fetcher.inject(newFetchAnnounce(&wire.MsgFetchInit{MsgID: 9}, fakePeerReply, sn.onfetchMsg)) // old resend round
	<-sn.fetcher.done
	assert.Equal(t, 1, len(sn.fetcher.handling))                                                     // still one
	sn.fetcher.inject(newFetchAnnounce(&wire.MsgFetchInit{MsgID: 11}, fakePeerReply, sn.onfetchMsg)) // new resend round
	<-sn.fetcher.done
	assert.Equal(t, 1, len(sn.fetcher.handling)) // replace old, still one
	time.Sleep(time.Millisecond * 50)            // wait handle
	assert.Equal(t, 0, len(sn.fetcher.handling)) // left zero
}
