package appblockchain

import (
	"errors"
	"hash/fnv"
	"time"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// announce is used to manage fetch message. it contains message's send time or receive time and resend round
type announce struct {
	ID                 uint32 // announce id is message id without resend round, it's last ResendRoundBitNum bits are 0
	Message            wire.Message
	Time               time.Time
	ResendRound        byte
	HandleFuncCallback func(msg wire.Message) error
	BroadFuncCallback  func(msg wire.Message)
}

func newAnnounce(id uint32, msg wire.Message, resendRound byte, handleFuncCallback func(msg wire.Message) error,
	broadFuncCallback func(msg wire.Message)) *announce {
	return &announce{
		ID:                 id,
		Message:            msg,
		Time:               time.Now(),
		ResendRound:        resendRound,
		HandleFuncCallback: handleFuncCallback,
		BroadFuncCallback:  broadFuncCallback,
	}
}

func encodeAndHashMsg(msg wire.Message, shardHeight int64, address multivacaddress.Address) (uint32, error) {
	fnvInst := fnv.New32()
	ret, err := rlp.EncodeToBytes(wire.MsgFetchEncode{
		Msg:         msg,
		ShardHeight: shardHeight,
		UserAddress: address,
	})
	if err != nil {
		err := errors.New("failed to encode message")
		return 0, err
	}
	_, err = fnvInst.Write(ret)
	if err != nil {
		err := errors.New("failed to calculate hash")
		return 0, err
	}
	return fnvInst.Sum32(), nil
}

// convertMessageHash convert msgHash to a announce id, whose last ResendRoundBitNum bit is 0
func convertMessageHash(msgHash uint32) uint32 {
	return Mask & msgHash
}

func newFetchTxAnnounce(number int, hashList []uint32, shard shard.Index, address multivacaddress.Address,
	handleFuncCallback func(msg wire.Message) error, broadFuncCallback func(msg wire.Message)) *announce {
	msg := &wire.MsgFetchTxs{
		MsgID:       0,
		NumberOfTxs: number,
		ShardIndex:  shard,
		ExistTx:     hashList,
	}
	msgHash, err := encodeAndHashMsg(msg, 0, address)
	if err != nil {
		return nil
	}
	msg.MsgID = convertMessageHash(msgHash)
	return newAnnounce(msg.MsgID, msg, 0, handleFuncCallback, broadFuncCallback)
}

func newFetchInitDataAnnounce(shardIndex shard.Index, shardHeight int64, address multivacaddress.Address,
	handleFuncCallback func(msg wire.Message) error, broadFuncCallback func(msg wire.Message)) *announce {
	msg := &wire.MsgFetchInit{
		MsgID:      0,
		ShardIndex: shardIndex,
		Address:    address,
	}
	msgHash, err := encodeAndHashMsg(msg, shardHeight, nil)
	if err != nil {
		return nil
	}
	msg.MsgID = convertMessageHash(msgHash)
	return newAnnounce(msg.MsgID, msg, 0, handleFuncCallback, broadFuncCallback)
}

func newFetchSmartContractInfoAnnounce(contractAddr multivacaddress.Address, shard shard.Index, userAddress multivacaddress.Address,
	handleFuncCallback func(msg wire.Message) error, broadFuncCallback func(msg wire.Message)) *announce {
	msg := &wire.MsgFetchSmartContractInfo{
		MsgID:        0,
		ContractAddr: contractAddr,
		ShardIndex:   shard,
	}
	msgHash, err := encodeAndHashMsg(msg, 0, userAddress)
	if err != nil {
		return nil
	}
	msg.MsgID = convertMessageHash(msgHash)
	return newAnnounce(msg.MsgID, msg, 0, handleFuncCallback, broadFuncCallback)
}
