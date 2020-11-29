package storagenode

import (
	"time"

	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/peer"
	"github.com/multivactech/MultiVAC/processor/miner/appblockchain"
)

// announce is used to manage fetch message. it contains message's send time or receive time and resend round
type announce struct {
	ID          uint32 // announce id is message id without resend round, it's last resendRoundBitNum bits are 0
	Message     wire.Message
	Time        time.Time
	ResendRound byte
	PeerReply   peer.Reply
	Callback    func(message wire.Message)
}

func getAnnounceIDByMsgID(msgID uint32) uint32 {
	return msgID & appblockchain.Mask
}

func getResendRoundByMsgID(msgID uint32) byte {
	return byte(msgID & (1<<appblockchain.ResendRoundBitNum - 1))
}

func newAnnounce(id uint32, msg wire.Message, resendRound byte, pr peer.Reply, callBack func(msg wire.Message)) *announce {
	return &announce{
		ID:          id,
		Message:     msg,
		Time:        time.Now(),
		ResendRound: resendRound,
		PeerReply:   pr,
		Callback:    callBack,
	}
}

func newFetchAnnounce(msg wire.Message, pr peer.Reply, callback func(msg wire.Message)) *announce {
	switch msg := msg.(type) {
	case *wire.MsgFetchInit:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), pr, callback)
	case *wire.MsgFetchTxs:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), pr, callback)
	case *wire.MsgFetchSmartContractInfo:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), pr, callback)
	}
	return nil
}

func newResultAnnounce(msg wire.Message) *announce {
	switch msg := msg.(type) {
	case *wire.MsgReturnInit:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	case *wire.MsgReturnTxs:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	case *wire.SmartContractInfo:
		return newAnnounce(getAnnounceIDByMsgID(msg.MsgID), msg, getResendRoundByMsgID(msg.MsgID), nil, nil)
	}
	return nil
}
