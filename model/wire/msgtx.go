// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/reducekey"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/prometheus/common/log"
)

// OutType is the type of tx out.
type OutType byte

const (
	// TxVersion is the current latest supported transaction version.
	TxVersion = 1

	// it seems unused.
	//maxTxInPerMessage  = 1000
	//maxTxOutPerMessage = 1000

	// StateUnused is state of an out point. Not using boolean because there will be special types
	// of transactions and outs which will have more than 2 states.
	StateUnused byte = 0
	// StateUsed is state of an out point.
	StateUsed byte = 1

	// LockDuration the time in second for locking withdraw output.
	// it seems unused.
	//LockDuration = 24 * 3600
	//precision    = 18

	// DepositOut is type of deposit
	DepositOut OutType = iota
	// WithdrawOut is type of withdraw
	WithdrawOut
	// Binding is type of binding
	Binding
	// Transfer is type of transfer out
	Transfer

	// WithdrawHeight is deposit withdraw lock height
	WithdrawHeight = 200
	// BindingLockHeight is binding tx locking duration
	BindingLockHeight = 20
)

var (
	// MinerReward is the amount of MultiVac coins given to miner as reward for creating a block.
	MinerReward = new(big.Int).SetUint64(10e18)
	// DepositTxFee is the transaction fee for a deposit transaction
	// It is set to motivate a miner stay a minimal time as a miner
	DepositTxFee = big.NewInt(10)
	// WithdrawTxFee is need to calculate the accurate deposit amount
	WithdrawTxFee = big.NewInt(10)

	// MinimalDepositAmount defines the minimal deposit count.
	MinimalDepositAmount = big.NewInt(10)
)

// OutPoint defines a data type that is used to track previous transaction outputs.
type OutPoint struct {
	Shard           shard.Index             `json:"Shard"`
	TxHash          chainhash.Hash          `json:"TxHash"`
	Index           int                     `json:"Index"`
	UserAddress     multivacaddress.Address `json:"UserAddress"`
	Type            OutType                 `json:"OutType"`
	Data            []byte                  `json:"Data"`
	ContractAddress multivacaddress.Address `json:"ContractAddress"`
}

//TODO: Add pkhash and value into string.
//String returns the OutPoint in the human-readable form "hash:index".
func (o *OutPoint) String() string {
	// Allocate enough for hash string, colon, and 10 digits.  Although
	// at the time of writing, the number of digits can be no greater than
	// the length of the decimal representation of maxTxOutPerMessage, the
	// maximum message payload may increase in the future and this
	// optimization may go unnoticed, so allocate space for 10 decimal
	// digits, which will fit any uint32.
	buf := make([]byte, 2*chainhash.HashSize+1, 2*chainhash.HashSize+1+10)
	copy(buf, o.TxHash.String())
	buf[2*chainhash.HashSize] = ':'
	buf = strconv.AppendUint(buf, uint64(o.Index), 10)
	return string(buf)
}

// UnspentOutHash returns the unspent out hash.
func UnspentOutHash(o *OutPoint) *merkle.MerkleHash {
	return merkle.ComputeMerkleHash(o.ToUnspentOutState().ToBytesArray())
}

// IsLock returns if the withdraw out is in locking.
func (o *OutPoint) IsLock(height int64) bool {
	if o.IsWithdrawOut() {
		data, err := OutToWithdrawData(o)
		if err != nil {
			return true
		}
		if height <= data.WithdrawHeight {
			return true
		}

		if height-data.WithdrawHeight < WithdrawHeight {
			return true
		}
	}
	return false
}

// IsBindingLock check the binding tx is locked.
func (o *OutPoint) IsBindingLock(height int64) bool {
	if o.IsDepositOut() {
		data, err := OutToDepositData(o)
		if err != nil {
			return true
		}
		if height-data.Height < BindingLockHeight {
			return true
		}
	}
	return false
}

// IsDepositOut returns if it is a deposit out.
func (o *OutPoint) IsDepositOut() bool {
	return o.Type == DepositOut
}

// IsWithdrawOut returns if it is a withdraw out.
func (o *OutPoint) IsWithdrawOut() bool {
	return o.Type == WithdrawOut
}

// ToUnspentOutState marks a OutPoint to unspent state.
func (o *OutPoint) ToUnspentOutState() *OutState {
	return &OutState{
		OutPoint: *o,
		State:    StateUnused,
	}
}

// ToSpentOutState marks a OutPoint to spent state.
func (o *OutPoint) ToSpentOutState() *OutState {
	return &OutState{
		OutPoint: *o,
		State:    StateUsed,
	}
}

// BtcDecode decode the message.
func (o *OutPoint) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, o)
}

// Deserialize deserialize the message.
func (o *OutPoint) Deserialize(r io.Reader) error {
	return o.BtcDecode(r, 0, BaseEncoding)
}

// BtcEncode encode the message.
func (o *OutPoint) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, o)
}

// Serialize serialize the message.
func (o *OutPoint) Serialize(w io.Writer) error {
	return o.BtcEncode(w, 0, BaseEncoding)
}

// IsSmartContractCode check if it is smart contract code.
func (o *OutPoint) IsSmartContractCode() bool {
	contractAddr := o.TxHash.FormatSmartContractAddress()
	return contractAddr.IsEqual(o.ContractAddress) && o.Index == 0
}

// IsSmartContractShardInitOut check if the out from smart contract.
func (o *OutPoint) IsSmartContractShardInitOut() bool {
	contractAddr := o.TxHash.FormatSmartContractAddress()
	return contractAddr.IsEqual(o.ContractAddress) && o.Index == 1
}

// MTVCoinData is the OutPoint data for transfer tx.
type MTVCoinData struct {
	Value *big.Int
}

// MtvValueToData writes the out Value into OutPoint.Data
func MtvValueToData(value *big.Int) []byte {
	if value == nil {
		panic(fmt.Errorf("value is nil"))
	}
	if value.Sign() <= 0 {
		panic(fmt.Errorf("value must be positive, %s", value.String()))
	}
	data := MTVCoinData{
		Value: value,
	}
	dataBytes, err := rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}
	return dataBytes
}

// GetMtvValueFromOut fetches the out Value from the given OutPoint.Data
func GetMtvValueFromOut(op *OutPoint) *big.Int {
	if op == nil {
		panic(fmt.Errorf("outPoint is nil"))
	}
	var data MTVCoinData
	if err := rlp.DecodeBytes(op.Data, &data); err != nil {
		panic(err)
	}
	return data.Value
}

// MTVDepositData is the OutPoint data for deposit tx
type MTVDepositData struct {
	Value *big.Int
	// Height is the block height at which out is effective.
	Height int64
	// BindingAddress is used to grant mining auth to specified user.
	BindingAddress multivacaddress.Address
}

// ToData serializes MTVDepositData to bytes by using rlp.
func (mdd *MTVDepositData) ToData() []byte {
	if mdd.Value == nil {
		panic(fmt.Errorf("value is nil"))
	}
	if mdd.Value.Sign() <= 0 {
		panic(fmt.Errorf("value must be positive, %s", mdd.Value.String()))
	}
	if mdd.Height <= 0 {
		panic(fmt.Errorf("invalid height of %d", mdd.Height))
	}

	dataBytes, err := rlp.EncodeToBytes(*mdd)
	if err != nil {
		panic(err)
	}
	return dataBytes
}

// OutToDepositData converts OutPoint to MTVDepositData.
func OutToDepositData(op *OutPoint) (*MTVDepositData, error) {
	if op == nil {
		return nil, fmt.Errorf("output is nil")
	}
	var data *MTVDepositData
	if err := rlp.DecodeBytes(op.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// MTVWithdrawData is the OutPoint data for withdraw tx.
type MTVWithdrawData struct {
	Value          *big.Int
	WithdrawHeight int64
}

// ToData converts MTVWithdrawData to bytes by using rlp.
func (mwd *MTVWithdrawData) ToData() []byte {
	if mwd.Value == nil {
		panic(fmt.Errorf("value is nil"))
	}
	if mwd.Value.Sign() <= 0 {
		panic(fmt.Errorf("value must be positive"))
	}
	if mwd.WithdrawHeight <= 0 {
		panic(fmt.Errorf("invalid height for withdrawHeight"))
	}

	dataBytes, err := rlp.EncodeToBytes(*mwd)
	if err != nil {
		panic(err)
	}
	return dataBytes
}

// OutToWithdrawData converts OutPoint to MTVDepositData.
func OutToWithdrawData(op *OutPoint) (*MTVWithdrawData, error) {
	if op == nil {
		return nil, fmt.Errorf("OutPoint is nil")
	}
	var data *MTVWithdrawData
	if err := rlp.DecodeBytes(op.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// OutState is the state of an out point, usually either unused or used except
// for special out.
type OutState struct {
	OutPoint
	State byte
}

// ToBytesArray converts OutState to bytes by using rlp.
func (out *OutState) ToBytesArray() []byte {
	ret, err := rlp.EncodeToBytes(out)
	if err != nil {
		return nil
	}
	return ret
}

// TxIn defines a bitcoin transaction input.
type TxIn struct {
	PreviousOutPoint OutPoint `json:"PreviousOutPoint"`
}

// NewTxIn returns a new bitcoin transaction input with the provided
// previous outpoint point and signature script.
func NewTxIn(prevOut *OutPoint) *TxIn {
	return &TxIn{
		PreviousOutPoint: *prevOut,
	}
}

// MsgTx represents a tx message.
// Use the AddTxIn and AddTxOut functions to build up the list of transaction
// inputs and outputs.
type MsgTx struct {
	Version            int32                   `json:"Version"`
	Shard              shard.Index             `json:"Shard"`
	TxIn               []*TxIn                 `json:"TxIn"`
	ContractAddress    multivacaddress.Address `json:"ContractAddress"`
	API                string                  `json:"API"`
	Params             []byte                  `json:"Params"`
	Nonce              uint64                  `json:"Nonce"`
	PublicKey          []byte                  `json:"PublicKey"`
	SignatureScript    []byte                  `json:"SignatureScript"`
	StorageNodeAddress multivacaddress.Address `json:"StorageNodeAddress"`
	Memo               string                  `json:"Memo"`
}

func (tx *MsgTx) doubleSha256WithoutSignatureAndPubKey() [32]byte {
	data := tx.SerializeWithoutSignatureAndPubKey()
	dataByHash := sha256.Sum256(data)
	return sha256.Sum256(dataByHash[:])
}

// SerializeWithoutSignatureAndPubKey serialize the message without
func (tx MsgTx) SerializeWithoutSignatureAndPubKey() []byte {
	tx.SignatureScript = []byte{}
	tx.PublicKey = []byte{}
	ret, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil
	}
	return ret
}

// TotalMtvValue returns the total MTV value of the given tx.
func (tx MsgTx) TotalMtvValue() *big.Int {
	txIns := tx.TxIn
	var value = big.NewInt(0)
	for _, in := range txIns {
		if in.PreviousOutPoint.ContractAddress.IsEqual(isysapi.SysAPIAddress) {
			value.Add(value, GetMtvValueFromOut(&in.PreviousOutPoint))
		}
	}
	return value
}

// SetSignatureScriptAndPubKey When signing a transaction, use your private key to sign tx.DoubleSha256WithoutSignatureAndPubKey(),
// then pass publicKey and the signature into this method.
// See msgtx.go#Sign for example.
func (tx *MsgTx) SetSignatureScriptAndPubKey(publicKey signature.PublicKey, sig []byte) {
	tx.PublicKey = publicKey
	tx.SignatureScript = sig
}

// Sign the transaction with a private key.
//
// SetSignatureScript is more recommended, and it doesn't need private key. This method
// is to make testing easier.
func (tx *MsgTx) Sign(privateKey *signature.PrivateKey) {
	data := tx.doubleSha256WithoutSignatureAndPubKey()
	sig := signature.Sign(*privateKey, data[:])
	tx.SetSignatureScriptAndPubKey(privateKey.Public(), sig)
}

// VerifySignature verify the signature is valid.
func (tx *MsgTx) VerifySignature() bool {
	if len(tx.SignatureScript) == 0 {
		return false
	}

	publicKey := tx.PublicKey

	// Verify that the address of outPoint in PreviousOutPoint matches the publicKey in Tx.
	if tx.IsReduceTx() {
		if len(tx.TxIn) != 1 {
			return false
		}

		txIn := tx.TxIn[0]
		if err := txIn.PreviousOutPoint.UserAddress.VerifyPublicKey(publicKey,
			multivacaddress.SmartContractAddress); err != nil {
			logger.WireLogger().Warn("MsgTx#VerifySignature wrong pub key for reduce out")
			return false
		}
	} else {
		for _, txIn := range tx.TxIn {
			if err := txIn.PreviousOutPoint.UserAddress.VerifyPublicKey(publicKey,
				multivacaddress.UserAddress); err != nil {
				logger.WireLogger().Warn("MsgTx#VerifySignature wrong pub key for outs")
				return false
			}
		}
	}
	sig := tx.SignatureScript
	data := tx.doubleSha256WithoutSignatureAndPubKey()
	v := signature.Verify(publicKey, data[:], sig)
	if !v {
		logger.WireLogger().Warn("MsgTx#VerifySignature Signature is wrong")
	}
	return v
}

// VerifyTransaction verifies whether the MsgTx is valid.
// 1. Its Signatures is correct and signed by same key as all outpoints.
// 2. All Txin are in same shard as the transation.
// 3. No duplicate Txins in the transaction.
// 4. All values are positive.
// 5. Verify whether the transaction is enough for reward storage node.
func (tx *MsgTx) VerifyTransaction() error {
	if !tx.VerifySignature() {
		return errors.New("Invalid Signature")
	}

	// Verify all txin are in same shard as the transaction.
	for _, txIn := range tx.TxIn {
		if txIn.PreviousOutPoint.Shard != tx.Shard {
			return fmt.Errorf("invalid TxIn shard %d, expect %d", txIn.PreviousOutPoint.Shard, tx.Shard)
		}
	}

	// Verify no duplicate txin.
	outsMap := make(map[merkle.MerkleHash]bool)
	for _, txIn := range tx.TxIn {
		hash := merkle.ComputeMerkleHash(
			txIn.PreviousOutPoint.ToUnspentOutState().ToBytesArray())
		_, ok := outsMap[*hash]
		if ok {
			return fmt.Errorf("Verify block body: Transactions have duplicate txin")
		}
		outsMap[*hash] = true
	}

	if len(tx.TxIn) == 0 {
		return fmt.Errorf("there is no txin")
	}

	if params.EnableStorageReward {
		// Verify the tx is enough for reward storage node.
		if !tx.IsEnoughForReward() {
			return fmt.Errorf("the tx value is not enough for storage reward")
		}
	}

	return nil
}

// CreateRewardTx creates a standard reward transaction.
func CreateRewardTx(shardIdx shard.Index, height int64, pk chainhash.Hash) *MsgTx {
	// We use public key's byte array directly for "hash".
	var nonce = uint64(shardIdx)
	nonce = nonce << 32
	nonce += uint64(height)

	//var pk [chainhash.HashSize]byte
	//copy(pk[:], PublicKey)
	return &MsgTx{
		Shard: shardIdx,
		Nonce: nonce,
	}
}

// GetFee Verifies the input and output values and calculates the transaction
// fee. Returns errors if the input output value is invalid, or fee is negative.
func (tx *MsgTx) GetFee() (*big.Int, error) {
	inputSum, outputSum, fee := big.NewInt(0), big.NewInt(0), big.NewInt(0)
	// TODO: Currently this has the risk of int64 overflow in extreme cases.
	// We could set a upper bond for values, and check it here.
	for _, txIn := range tx.TxIn {
		var prevOutPointData struct{ Value *big.Int }
		if err := rlp.DecodeBytes(txIn.PreviousOutPoint.Data, &prevOutPointData); err != nil {
			return nil, err
		}
		if prevOutPointData.Value.Sign() <= 0 {
			return nil, errors.New("Negative input value found")
		}
		inputSum.Add(inputSum, prevOutPointData.Value)
	}

	fee.Sub(inputSum, outputSum)
	if fee.Sign() < 0 {
		return fee, fmt.Errorf("Negative tx fee: %v", fee)
	}
	return fee, nil
}

// VerifyWithdrawTx verifies all withdraw TxIn are DepositOut.
func (tx *MsgTx) VerifyWithdrawTx(blockHeight int64) error {
	// For withdraw transactions, all txIn should be depositOut.
	for _, txIn := range tx.TxIn {
		if !txIn.PreviousOutPoint.IsDepositOut() {
			return fmt.Errorf("out's type is not DepositOut")
		}
	}
	return nil
}

// AddTxIn adds a transaction input to the message.
func (tx *MsgTx) AddTxIn(ti *TxIn) {
	if tx.TxIn == nil {
		tx.TxIn = []*TxIn{}
	}
	tx.TxIn = append(tx.TxIn, ti)
}

// TxHash generates the Hash for the transaction.
func (tx *MsgTx) TxHash() chainhash.Hash {
	// Encode the transaction and calculate double sha256 on the result.
	// Ignore the error returns since the only way the encode could fail
	// is being out of memory or due to nil pointers, both of which would
	// cause a run-time panic.
	buf := bytes.NewBuffer([]byte{})
	_ = tx.Serialize(buf)
	return chainhash.HashH(buf.Bytes())
}

// BtcDecode decode the message.
func (tx *MsgTx) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, tx)
}

// Deserialize deserialize the message.
func (tx *MsgTx) Deserialize(r io.Reader) error {
	return tx.BtcDecode(r, 0, BaseEncoding)
}

// BtcEncode encode the message.
func (tx *MsgTx) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, tx)
}

// Serialize serialize the message.
func (tx *MsgTx) Serialize(w io.Writer) error {
	return tx.BtcEncode(w, 0, BaseEncoding)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (tx *MsgTx) Command() string {
	return CmdTx
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (tx *MsgTx) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

// NewMsgTx returns a new bitcoin tx message that conforms to the Message
// interface.  The return instance has a default version of TxVersion and there
// are no transaction inputs or outputs.  Also, the lock time is set to zero
// to indicate the transaction is valid immediately as opposed to some time in
// future.
func NewMsgTx(version int32, shard shard.Index) *MsgTx {
	return &MsgTx{
		Shard:              shard,
		Version:            version,
		TxIn:               []*TxIn{},
		ContractAddress:    isysapi.SysAPIAddress,
		API:                "",
		Params:             []byte{},
		SignatureScript:    []byte{},
		PublicKey:          []byte{},
		StorageNodeAddress: multivacaddress.Address{},
	}
}

// SerializeSize returns the number of bytes it would take to serialize the block.
func (tx *MsgTx) SerializeSize() int {
	var buf bytes.Buffer
	err := tx.BtcEncode(&buf, 0, BaseEncoding)
	if err != nil {
		log.Errorf("failed to encode message,err:%v", err)
		return 0
	}
	return buf.Len()
}

// SerializeSizeStripped returns the number of bytes it would take to serialize
// the block, excluding any witness data (if any).
func (tx *MsgTx) SerializeSizeStripped() int {
	return tx.SerializeSize()
}

// IsTxInShard returns if the tx belongs to the given shard.
func (tx *MsgTx) IsTxInShard(shardIndex shard.Index) bool {
	return shardIndex == tx.GetShardForTx()
}

// GetShardForTx returns the shard of a tx.
func (tx *MsgTx) GetShardForTx() shard.Index {
	return tx.Shard
}

func (out *OutState) verifyOutState(proof *merkle.MerklePath, root *merkle.MerkleHash) error {
	hash := merkle.ComputeMerkleHash(out.ToBytesArray())
	var err error
	// Verify the proof of this out is valid
	if *hash != *proof.GetLeaf() {
		return fmt.Errorf("wrong proof for out,err: %v ", err)
	}
	if err = proof.Verify(root); err != nil {
		err = fmt.Errorf("invalid proof, err msg: %s", err)
		return err
	}
	return nil
}

// IsSmartContractTx determine if tx is a transaction that executes a smart contract
func (tx *MsgTx) IsSmartContractTx() bool {
	return !tx.ContractAddress.IsEqual(isysapi.SysAPIAddress)
}

// IsRewardTx check if it is reward tx.
func (tx *MsgTx) IsRewardTx() bool {
	return tx.API == isysapi.SysAPIReward
}

// IsEnoughForReward judge whether the tx's value is enough for reward storage node.
func (tx *MsgTx) IsEnoughForReward() bool {
	if tx.IsRewardTx() {
		return true
	}
	vinValue := tx.getVinValue()
	if vinValue == nil {
		return false
	}
	voutValue := tx.getVoutValue()
	if voutValue == nil {
		return false
	}
	if new(big.Int).Sub(vinValue, voutValue).Cmp(isysapi.StorageRewardAmount) < 0 {
		return false
	}
	return true
}

func (tx *MsgTx) getVinValue() *big.Int {
	totalInputs := big.NewInt(0)
	for _, txIn := range tx.TxIn {
		var value *big.Int
		if tx.API == isysapi.SysAPIWithdraw {
			depositData, err := OutToDepositData(&txIn.PreviousOutPoint)
			if err != nil {
				return nil
			}
			value = depositData.Value
		} else if txIn.PreviousOutPoint.IsWithdrawOut() {
			withdrawData, err := OutToWithdrawData(&txIn.PreviousOutPoint)
			if err != nil {
				return nil
			}
			value = withdrawData.Value
		} else {
			value = GetMtvValueFromOut(&txIn.PreviousOutPoint)
		}
		totalInputs = new(big.Int).Add(totalInputs, value)
	}
	return totalInputs
}

func (tx *MsgTx) getVoutValue() *big.Int {
	if len(tx.API) == 0 || len(tx.Params) == 0 {
		return nil
	}
	var err error
	switch tx.API {
	case isysapi.SysAPITransfer:
		var tp isysapi.TransferParams
		err = rlp.DecodeBytes(tx.Params, &tp)
		if err != nil {
			return nil
		}
		return tp.Amount
	case isysapi.SysAPIDeposit:
		var dp isysapi.DepositParams
		err = rlp.DecodeBytes(tx.Params, &dp)
		if err != nil {
			return nil
		}
		return dp.Amount
	case isysapi.SysAPIWithdraw:
		var wp isysapi.WithdrawParams
		err = rlp.DecodeBytes(tx.Params, &wp)
		if err != nil {
			return nil
		}
		return wp.Amount
	// TODO(#issue 203): the logic of out should be defined when the VM has been implemented
	case isysapi.SysAPIDeploy:
		return big.NewInt(0)
	default:
		return nil
	}
}

// IsReduceTx determine whether the tx is reduceTx
func (tx *MsgTx) IsReduceTx() bool {
	return bytes.Equal(tx.PublicKey, reducekey.ReducePublicKey)
}
