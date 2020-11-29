/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package appblockchain

import (
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	evtMsg message.EventTopic = iota
	//evtGetShardHgtReq
	evtAcceptBlockReq
	evtVrfBlockReq
	evtPropBlockReq
	//evtGetHeaderHashes
	evtPropAndAcceptEmptyBlock
	//evtGetBlockHeader
)

type proposeBlockRequest struct {
	pk []byte
	sk []byte
}

// Todo : not used now
// type request interface{}

type proposeBlockResponse struct {
	block *wire.MsgBlock
	err   error
}

type acceptEmptyBlockResponse struct {
	header *wire.BlockHeader
	err    error
}

// Act is executed when the given Event is catched, then the callback func will be acted.
func (abc *appBlockChain) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtMsg:
		abc.onNewMessage(e.Extra.(wire.Message))
	case evtVrfBlockReq:
		block := e.Extra.(*wire.MsgBlock)
		callback(abc.VerifyBlock(block))
	case evtPropBlockReq:
		msg := e.Extra.(proposeBlockRequest)
		block, err := abc.ProposeBlock(msg.pk, msg.sk)
		callback(proposeBlockResponse{block, err})
	case evtPropAndAcceptEmptyBlock:
		blockHeader, err := abc.ProposeAndAcceptEmptyBlock(e.Extra.(int64))
		callback(acceptEmptyBlockResponse{blockHeader, err})
	case evtAcceptBlockReq:
		msg := e.Extra.(*wire.MsgBlock)
		callback(abc.AcceptBlock(msg))
	default:
		abc.log.Debugf("%v received unknown mail: %v", abc.shardIndex, e.Topic)
	}
}
