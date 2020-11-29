/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/peer"
	"github.com/multivactech/MultiVAC/processor/shared/message"
)

const (
	evtMsgTx message.EventTopic = iota
	evtMsgBlock
	evtGetOutStateReq
	evtGetAllShardHgts
	evtGetHeaderHashes
	evtGetBlockReq
	evtGetLastConfirmation
	evtGetSlimBlockReq
	evtSlimBlock
	evtGetSmartContract
	evtGetSmartContractInfo
	evtMsgBlockConfirmation
	evtFetchMsg
	//evtAddSmartContractInfosToDataStore
)

type getOutStateResponse struct {
	outState *wire.OutState
	err      error
}

type fetchRequest struct {
	msg       wire.Message
	peerReply peer.Reply
}

// Request header hashes based on locators
type reqHeaderHashes struct {
	locators []*wire.BlockLocator
}

// Respond header hashes
type resHeaderHashes struct {
	headerGroups []*wire.InvGroup
}

// Request a block message specified by header hash in a given shard.
type getBlockReq struct {
	shardIdx   shard.Index
	headerHash chainhash.Hash
}
