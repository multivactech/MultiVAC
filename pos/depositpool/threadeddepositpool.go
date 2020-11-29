package depositpool

import (
	"time"

	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// ThreadedDepositPool used to wrap DepositPool
type ThreadedDepositPool struct {
	worker   *DepositPool
	actorCtx *message.ActorContext
}

// StorageMode is used to indicate whether is a storagenode
type StorageMode bool

const (
	// MemoryMode -> use memory deposit pool
	MemoryMode StorageMode = false
	// DatabaseMode -> use db deposit pool
	DatabaseMode StorageMode = true
)

// NewThreadedDepositPool returns the thread-safe instance of depositpool.
func NewThreadedDepositPool(blockChain iblockchain.BlockChain, storageMode StorageMode) idepositpool.DepositPool {
	depositpool, err := NewDepositPool(blockChain, bool(storageMode))
	if err != nil {
		panic(err)
	}
	tdp := &ThreadedDepositPool{
		worker:   depositpool,
		actorCtx: message.NewActorContext(),
	}
	tdp.actorCtx.StartActor(tdp.worker)
	return tdp
}

// Add adds a new proof of deposit in pool.
func (tdp *ThreadedDepositPool) Add(out *wire.OutPoint, height wire.BlockHeight, proof *merkle.MerklePath) error {
	now := time.Now()
	err := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtAddReq, addToPoolRequest{out: out, height: height, proof: proof}))
	if err == nil {
		return nil
	}
	d := time.Since(now)
	log.Debugf("Deposit-Time: Add %v", d)
	return err.(error)
}

// Remove removes a proof in pool.
func (tdp *ThreadedDepositPool) Remove(o *wire.OutPoint) error {
	err := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtRemoveReq, removeFromPoolRequest{o: o}))
	if err == nil {
		return nil
	}
	return err.(error)
}

// GetBiggest returns a biggest proof.
func (tdp *ThreadedDepositPool) GetBiggest(address multivacaddress.Address) ([]byte, error) {
	resp := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtTopNReq, topDepositRequest{address: address})).(topDepositResponse)
	if resp.err != nil {
		return nil, resp.err
	}
	return resp.deposit, nil
}

// GetAll returns all deposits of the given address.
func (tdp *ThreadedDepositPool) GetAll(shardIndex shard.Index, address multivacaddress.Address) ([]*wire.OutWithProof, error) {
	resp := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtDepositWithProofReq, reqGetAll{address: address, shard: shardIndex})).(getAllResponse)
	if resp.err != nil {
		return nil, resp.err
	}
	return resp.OutsWithProof, nil
}

// Update udaptes all proof's path in the pool.
func (tdp *ThreadedDepositPool) Update(update *state.Update, shard shard.Index, height wire.BlockHeight) error {
	err := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtUpdateReq, refreshRequest{update: update, shard: shard, height: height}))
	if err == nil {
		return nil
	}
	return err.(error)
}

// Verify verifies the correction of the given proof.
func (tdp *ThreadedDepositPool) Verify(address multivacaddress.Address, d []byte) bool {
	resp := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtVerifyReq, verifyRequest{deposit: d, address: address})).(verifyResponse)
	return resp.ok
}

// Lock locks a proof, it means the deposit will not update anymore.
func (tdp *ThreadedDepositPool) Lock(o *wire.OutPoint) error {
	err := tdp.actorCtx.SendAndWait(tdp.worker, message.NewEvent(evtLockReq, lockRequest{o: o}))
	if err == nil {
		return nil
	}
	return err.(error)
}
