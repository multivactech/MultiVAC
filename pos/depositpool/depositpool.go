/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package depositpool

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/multivactech/MultiVAC/base/db"
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/message"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

const (
	dbNameSpace = "deposit_data"
)

// DepositInfo represents the information
// that is used for vetifying the deposit tx.
type DepositInfo struct {
	OutPoint    *wire.OutPoint
	Proof       *merkle.MerklePath
	BlockHeight wire.BlockHeight
	Locked      bool
}

// Return 1: i > j.
// Return 0: i = j.
// Return -1 i < j.
func compare(i, j *wire.OutPoint) int {
	valuei, _ := wire.OutToDepositData(i)
	valuej, _ := wire.OutToDepositData(j)
	return valuei.Value.Cmp(valuej.Value)
}

// DepositPool is to store the information about
// the deposits and their owners.
type DepositPool struct {
	pool       map[string]*DepositInfo
	isUsedDB   bool
	depositDb  db.DB
	blockChain iblockchain.BlockChain
}

// NewDepositPool returns a depositpool need blockchain and isuseddb to init.
func NewDepositPool(blockChain iblockchain.BlockChain, isUsedDB bool) (*DepositPool, error) {
	dPool := &DepositPool{
		pool:       make(map[string]*DepositInfo),
		blockChain: blockChain,
		isUsedDB:   isUsedDB,
	}
	dPool.depositDb = nil
	if isUsedDB {
		depositDb, err := db.OpenDB(config.GlobalConfig().DataDir, dbNameSpace)
		if err != nil {
			return dPool, err
		}
		dPool.depositDb = depositDb
	}
	return dPool, nil
}

func getOutKey(out *wire.OutPoint) string {
	hash := wire.UnspentOutHash(out)
	val := hex.EncodeToString(hash[:])
	return val
}

// GetAll returns all deposit proofs of the given address.
func (dPool *DepositPool) GetAll(shardIndex shard.Index, address multivacaddress.Address) ([]*wire.OutWithProof, error) {
	outsWithProof := make([]*wire.OutWithProof, 0)
	if dPool.isUsedDB {
		deposits, err := dPool.getAllDeposit()
		if err != nil {
			return nil, err
		}
		for _, deposit := range deposits {
			data, err := wire.OutToDepositData(deposit.OutPoint)
			if err != nil {
				continue
			}
			if data.BindingAddress.IsEqual(address) && deposit.OutPoint.Shard == shardIndex {
				outsWithProof = append(outsWithProof, &wire.OutWithProof{
					Out:    deposit.OutPoint,
					Proof:  deposit.Proof,
					Height: deposit.BlockHeight,
				})
			}
		}
	} else {
		for _, deposit := range dPool.pool {
			data, err := wire.OutToDepositData(deposit.OutPoint)
			if err != nil {
				continue
			}
			if data.BindingAddress.IsEqual(address) && deposit.OutPoint.Shard == shardIndex {
				outsWithProof = append(outsWithProof, &wire.OutWithProof{
					Out:    deposit.OutPoint,
					Proof:  deposit.Proof,
					Height: deposit.BlockHeight,
				})
			}
		}
	}
	return outsWithProof, nil
}

// Add adds a deposit to the deposit pool.
//func (dPool *DepositPool) Add(d *DepositInfo) error {
func (dPool *DepositPool) Add(out *wire.OutPoint, height wire.BlockHeight, proof *merkle.MerklePath) error {
	if out == nil || proof == nil {
		return errors.New("add nil proof or out")
	}
	log.Debugf("Add out to pool, to:  %v", out.UserAddress)
	d := &DepositInfo{
		OutPoint:    out,
		Proof:       proof,
		BlockHeight: height,
		Locked:      false,
	}
	key := getOutKey(d.OutPoint)
	_, ok := dPool.pool[key]
	if !ok {
		dPool.pool[key] = d
	} else {
		d.Locked = dPool.pool[key].Locked
		dPool.pool[key] = d
	}

	if dPool.isUsedDB {
		dkey, err := hex.DecodeString(key)
		if err != nil {
			return fmt.Errorf("failed to decode key")
		}

		has, err := dPool.depositDb.Has(dkey)
		if err != nil {
			return err
		}
		if !has {
			dataToSave, err := rlp.EncodeToBytes(d)
			if err != nil {
				return fmt.Errorf("rlp encodet to bytes err : %v", err)
			}
			err = dPool.depositDb.Put(dkey, dataToSave)
			if err != nil {
				return err
			}
		} else {
			data, err := dPool.depositDb.Get(dkey)
			if err != nil {
				return err
			}
			deposit := &DepositInfo{}
			err = rlp.DecodeBytes(data, deposit)
			if err != nil {
				return err
			}
			d.Locked = deposit.Locked

			dataToSave, err := rlp.EncodeToBytes(d)
			if err != nil {
				return err
			}
			err = dPool.depositDb.Put(dkey, dataToSave)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Remove removes the corresponding deposit from the pool.
func (dPool *DepositPool) Remove(o *wire.OutPoint) error {
	key := getOutKey(o)
	_, ok := dPool.pool[key]
	if ok {
		delete(dPool.pool, key)
	}
	if dPool.isUsedDB {
		dkey, err := hex.DecodeString(key)
		if err != nil {
			return fmt.Errorf("failed to decode key")
		}
		has, err := dPool.depositDb.Has(dkey)
		if err != nil {
			return err
		}
		if has {
			err = dPool.depositDb.Delete(dkey)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Verify with given blockHeight and outpoint proof
// the param []byte is encoded by a DepositInfo pointer.
func (dPool *DepositPool) Verify(address multivacaddress.Address, d []byte) bool {
	// Decode data from []byte
	deposit := &DepositInfo{}
	err := rlp.DecodeBytes(d, deposit)
	if err != nil {
		return false
	}

	data, err := wire.OutToDepositData(deposit.OutPoint)
	if err != nil {
		log.Errorf("Dpool failed to resovle deposit data for address: %v", deposit.OutPoint.UserAddress)
		return false
	}

	// Verify if the deposit info is belongs to the given pk.
	if !address.IsEqual(data.BindingAddress) {
		return false
	}

	// Verify if the deposit value is less than threshold.
	if data.Value.Cmp(big.NewInt(params.DepositValue)) == -1 {
		log.Errorf("This deposit's value is not enough: %v", data.Value.String())
		return false
	}

	if !wire.UnspentOutHash(deposit.OutPoint).Equal(deposit.Proof.Hashes[0]) {
		log.Debugf("Wrong proof of the given outpoint")
		return false
	}

	shardIndex := deposit.OutPoint.Shard
	height := dPool.blockChain.GetShardsHeight(shardIndex)

	// Deposit out has withdraw lock time that allows to be verified during this period.
	// W is withdraw time, WL is withdraw unlock time so M is the lock time.
	//
	//       W-M       W       WL
	//---------------------------------
	//        |____M___|____M___|
	if int(height)-int(deposit.BlockHeight) > wire.WithdrawHeight {
		return false
	}
	blockHeader := dPool.blockChain.GetShardsHeaderByHeight(shardIndex, wire.BlockHeight(deposit.BlockHeight))
	if blockHeader == nil {
		log.Debugf("Can't found header(%d) of shard(%d) from chain, current height is %d", deposit.BlockHeight, shardIndex, dPool.blockChain.GetShardsHeight(shardIndex))
		// todo(MH｜Diorzz|Akun-kf): sync header https://github.com/multivactech/MultiVAC/issues/418
		return true
	}
	err = deposit.Proof.Verify(&blockHeader.ShardLedgerMerkleRoot)
	if err != nil {
		log.Debugf("Proof not match the root")
		return false
	}
	return true
}

// GetBiggest return the biggest deposit of the given node hash.
// The return value is a byte array encoded by a DepositInfo pointer.
func (dPool *DepositPool) GetBiggest(address multivacaddress.Address) ([]byte, error) {
	var d *DepositInfo
	init := false
	for _, deposit := range dPool.pool {
		data, err := wire.OutToDepositData(deposit.OutPoint)
		if err != nil {
			continue
		}
		if !data.BindingAddress.IsEqual(address) {
			continue
		}
		if !init {
			d = deposit
			init = true
		} else {
			if compare(deposit.OutPoint, d.OutPoint) == 1 {
				d = deposit
			}
		}
	}
	if dPool.isUsedDB {
		deposits, err := dPool.getAllDeposit()
		if err != nil {
			return nil, fmt.Errorf("get all object fomr db error : %v", err)
		}
		for _, deposit := range deposits {
			data, err := wire.OutToDepositData(deposit.OutPoint)
			if err != nil {
				continue
			}
			if !data.BindingAddress.IsEqual(address) {
				continue
			}
			if !init {
				d = deposit
				init = true
			} else {
				if compare(deposit.OutPoint, d.OutPoint) == 1 {
					d = deposit
				}
			}
		}
	}
	if !init {
		return nil, fmt.Errorf("this node has no deposit, pk: %v", address)
	}

	// TODO: just for testnet-3.0
	if params.EnableTestnetThreshold {
		data, err := wire.OutToDepositData(d.OutPoint)
		if err != nil {
			return nil, err
		}
		if data.Value.Cmp(big.NewInt(params.DepositValue)) == -1 {
			return nil, fmt.Errorf("No enough deposit, at least 1000000MTV")
		}
	}

	if d.Proof == nil {
		return nil, fmt.Errorf("Invalid proof of deposit")
	}

	ret, err := rlp.EncodeToBytes(d)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func updateProof(oldproof *merkle.MerklePath, update *state.Update) (*merkle.MerklePath, error) {
	log.Debugf("Deposit update proof: %v", oldproof)
	newProof, err := update.DeltaTree.RefreshMerklePath(oldproof)
	if err != nil {
		return nil, err
	}
	if err := newProof.Verify(update.DeltaTree.GetMerkleRoot()); err != nil {
		return nil, err
	}
	return newProof, nil
}

// Update updates the outpoints in pool based on the given update instance.
func (dPool *DepositPool) Update(update *state.Update, shard shard.Index, height wire.BlockHeight) error {
	for hash, deposit := range dPool.pool {
		if deposit.Locked {
			if height-deposit.BlockHeight > wire.WithdrawHeight {
				delete(dPool.pool, hash)
			}
			continue
		}
		if deposit.OutPoint.Shard.GetID() != shard.GetID() {
			continue
		}
		if deposit.Proof == nil {
			delete(dPool.pool, hash)
			continue
		}

		newProof, err := updateProof(deposit.Proof, update)
		if err != nil {
			return err
		}
		deposit.Proof = newProof
		deposit.BlockHeight = height
	}
	if dPool.isUsedDB {
		deposits, err := dPool.getAllDeposit()
		if err != nil {
			return err
		}
		for _, deposit := range deposits {
			if deposit.Locked {
				if height-deposit.BlockHeight > wire.WithdrawHeight {
					// TODO: Delete from db
					//hash := wire.UnspentOutHash(deposit.OutPoint)
					//key := getDepositKey(*hash)
					//dPool.depositDb.Delete(key)

					// 删除该deposit
					key := getOutKey(deposit.OutPoint)
					dkey, err := hex.DecodeString(key)
					if err != nil {
						return fmt.Errorf("failed to decode key")
					}

					has, err := dPool.depositDb.Has(dkey)
					if err != nil {
						return err
					}
					if has {
						err = dPool.depositDb.Delete(dkey)
						if err != nil {
							return err
						}
					}

				}
				continue
			}
			if deposit.OutPoint.Shard != shard {
				continue
			}
			if deposit.Proof == nil {
				// 删除该deposit
				key := getOutKey(deposit.OutPoint)
				dkey, err := hex.DecodeString(key)
				if err != nil {
					return fmt.Errorf("failed to decode key")
				}

				has, err := dPool.depositDb.Has(dkey)
				if err != nil {
					return err
				}
				if has {
					err = dPool.depositDb.Delete(dkey)
					if err != nil {
						return err
					}
				}
				continue
			}
			newProof, err := updateProof(deposit.Proof, update)
			if err != nil {
				return err
			}
			deposit.Proof = newProof
			deposit.BlockHeight = height

			key := getOutKey(deposit.OutPoint)
			dkey, err := hex.DecodeString(key)
			if err != nil {
				return err
			}

			dataToSave, err := rlp.EncodeToBytes(deposit)
			if err != nil {
				return err
			}
			err = dPool.depositDb.Put(dkey, dataToSave)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dPool *DepositPool) getAllDeposit() ([]*DepositInfo, error) {
	var deposits []*DepositInfo
	iter := dPool.depositDb.GetIter()
	for iter.Next() {
		deposit := &DepositInfo{}
		err := rlp.DecodeBytes(iter.Value(), deposit)
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, deposit)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return deposits, err
}

// Lock locks the specific depoist and this deposit will not update anymore.
func (dPool *DepositPool) Lock(o *wire.OutPoint) error {
	key := getOutKey(o)
	_, ok := dPool.pool[key]
	if ok {
		dPool.pool[key].Locked = true
	}
	if dPool.isUsedDB {
		dkey, err := hex.DecodeString(key)
		if err != nil {
			return fmt.Errorf("failed to decode key")
		}
		has, err := dPool.depositDb.Has(dkey)
		if err != nil {
			return err
		}
		if !has {
			return fmt.Errorf("this deposit tx not exist")
		}
		rawData, err := dPool.depositDb.Get(dkey)
		if err != nil {
			return err
		}
		deposit := &DepositInfo{}
		err = rlp.DecodeBytes(rawData, deposit)
		if err != nil {
			return err
		}
		deposit.Locked = true
		rawData, err = rlp.EncodeToBytes(deposit)
		if err != nil {
			return nil
		}
		err = dPool.depositDb.Put(dkey, rawData)
		if err != nil {
			return err
		}
	}
	return nil
}

// Act is the implementation of actor pattern.
func (dPool *DepositPool) Act(e *message.Event, callback func(m interface{})) {
	switch e.Topic {
	case evtAddReq:
		msg := e.Extra.(addToPoolRequest)
		callback(dPool.Add(msg.out, msg.height, msg.proof))
	case evtRemoveReq:
		msg := e.Extra.(removeFromPoolRequest)
		callback(dPool.Remove(msg.o))
	case evtTopNReq:
		msg := e.Extra.(topDepositRequest)
		deposit, err := dPool.GetBiggest(msg.address)
		callback(topDepositResponse{deposit: deposit, err: err})
	case evtDepositWithProofReq:
		req := e.Extra.(reqGetAll)
		outsWithProof, err := dPool.GetAll(req.shard, req.address)
		callback(getAllResponse{OutsWithProof: outsWithProof, err: err})
	case evtUpdateReq:
		msg := e.Extra.(refreshRequest)
		callback(dPool.Update(msg.update, msg.shard, msg.height))
	case evtVerifyReq:
		msg := e.Extra.(verifyRequest)
		ok := dPool.Verify(msg.address, msg.deposit)
		callback(verifyResponse{ok: ok})
	case evtLockReq:
		msg := e.Extra.(lockRequest)
		callback(dPool.Lock(msg.o))
	default:
		log.Debugf("%v received unknown mail: %v", dPool, e.Topic)
	}
}
