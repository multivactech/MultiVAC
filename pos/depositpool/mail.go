package depositpool

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

const (
	evtAddReq message.EventTopic = iota
	evtRemoveReq
	evtVerifyReq
	evtTopNReq
	evtLockReq
	evtUpdateReq
	evtDepositWithProofReq
)

type reqGetAll struct {
	shard   shard.Index
	address multivacaddress.Address
}

type addToPoolRequest struct {
	out    *wire.OutPoint
	height wire.BlockHeight
	proof  *merkle.MerklePath
}

type removeFromPoolRequest struct {
	o *wire.OutPoint
}

type verifyRequest struct {
	address multivacaddress.Address
	deposit []byte
}

type verifyResponse struct {
	ok bool
}

type refreshRequest struct {
	shard  shard.Index
	update *state.Update
	height wire.BlockHeight
}

type topDepositRequest struct {
	address multivacaddress.Address
}

type topDepositResponse struct {
	deposit []byte
	err     error
}

type getAllResponse struct {
	OutsWithProof []*wire.OutWithProof
	err           error
}

type lockRequest struct {
	o *wire.OutPoint
}
