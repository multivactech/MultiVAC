/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txprocessor

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestRewardTx(t *testing.T) {
	toAddr := multivacaddress.Address([]byte("toHash"))
	testShard := shard.Index(3)
	rp := isysapi.RewardParams{
		To:     toAddr,
		Amount: isysapi.MinerRewardAmount,
	}
	params, err := rlp.EncodeToBytes(rp)
	if err != nil {
		t.Errorf("failed to marshal reward params %v, err msg %v", rp, err)
	}

	tx := &wire.MsgTx{
		Shard:           testShard,
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIReward,
		Params:          params,
	}

	sce := NewSysCE(testShard)
	ops, scs, err := sce.Execute(tx, 0)
	if err != nil {
		t.Errorf("Failed to execute reward tx %v, err msg %v", tx, err)
	}
	if len(ops) != 1 {
		t.Errorf("Unexpected length of ops %v", len(ops))
	}
	if scs != nil {
		t.Errorf("Unexpected variable %v", scs)
	}
	op := ops[0]
	value := wire.GetMtvValueFromOut(op)
	if op.Shard != testShard || !op.UserAddress.IsEqual(toAddr) ||
		value.Cmp(isysapi.MinerRewardAmount) != 0 {
		t.Errorf("wrong reward out point %v", op)
	}
}

// Todo：not used now
/*func testRewardTxWrongValue(t *testing.T) {
	rp := isysapi.RewardParams{
		To:     multivacaddress.Address{},
		Amount: new(big.Int).Add(isysapi.MinerRewardAmount, new(big.Int).SetUint64(1e18)),
	}
	params, _ := rlp.EncodeToBytes(rp)
	tx := &wire.MsgTx{
		Shard:           shard.Index(0),
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIReward,
		Params:          params,
	}

	sce := NewSysCE(shard.Index(0))
	_, _, err := sce.Execute(tx, 0)
	if err == nil {
		t.Errorf("executor didn't through error for wrong value")
	}
}*/
func TestDeploySmartContract(t *testing.T) {
	dp := isysapi.DeployParams{
		APIList: []string{"deployContract"},
		// char* init(int n) {
		//     char* arr = new char[6];
		//     arr = "0code";
		//     arr[0] = 0x84;   // rlp encoding for "code"
		//     return arr;
		// }
		Init: []byte{
			0, 97, 115, 109, 1, 0, 0, 0, 1, 134, 128, 128, 128, 0, 1, 96, 1, 127, 1, 127, 3, 130, 128, 128, 128, 0, 1,
			0, 4, 132, 128, 128, 128, 0, 1, 112, 0, 0, 5, 131, 128, 128, 128, 0, 1, 0, 1, 6, 129, 128, 128, 128, 0, 0,
			7, 149, 128, 128, 128, 0, 2, 6, 109, 101, 109, 111, 114, 121, 2, 0, 8, 95, 90, 52, 105, 110, 105, 116, 105,
			0, 0, 10, 146, 128, 128, 128, 0, 1, 140, 128, 128, 128, 0, 0, 65, 0, 65, 132, 1, 58, 0, 16, 65, 16, 11, 11,
			140, 128, 128, 128, 0, 1, 0, 65, 16, 11, 6, 48, 99, 111, 100, 101, 0,
		},
		Code: []byte{105, 110, 105, 116},
	}
	params, _ := rlp.EncodeToBytes(dp)
	out := wire.OutPoint{Shard: shard.Index(1), Data: wire.MtvValueToData(big.NewInt(1))}
	tx := &wire.MsgTx{
		Shard:           shard.Index(1),
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIDeploy,
		Params:          params,
		TxIn:            []*wire.TxIn{wire.NewTxIn(&out)},
	}
	if !tx.IsEnoughForReward() {
		return
	}
	sce := NewSysCE(shard.Index(1))
	outs, sc, err := sce.Execute(tx, 0)
	if err != nil {
		t.Errorf("failed to execute deploy tx %v, err msg %v", tx, err)
	}
	if len(outs) != 2*len(shard.ShardList)+1 {
		t.Errorf("unexpected length of ops want: %v\nget: %v", 2*len(shard.ShardList)+1, len(outs))
	}
	if sc == nil {
		t.Errorf("Smart contracts are not generated,sc is %v", sc)
	}
	txHash := tx.TxHash()

	smartContract := &wire.SmartContract{
		ContractAddr: txHash.FormatSmartContractAddress(),
		APIList:      dp.APIList,
		Code:         dp.Code,
	}

	scHash := smartContract.SmartContractHash()
	scBytes := scHash.CloneBytes()

	gas := big.NewInt(0)
	totalInputs := big.NewInt(1)
	value := new(big.Int).Sub(totalInputs, gas)

	txHashValue := tx.TxHash()
	sCAddr := txHashValue.FormatSmartContractAddress()
	for _, out := range outs {

		if !out.TxHash.IsEqual(&txHashValue) {
			t.Errorf("the TxHash of out is wrong,\nwant: %v,\nget: %v", tx.TxHash(), out.TxHash)
		}

		if out.Index != 2 && !out.ContractAddress.IsEqual(sCAddr) {
			t.Errorf("the ContractAddress is wrong,\nwant: %v,\nget: %v", sCAddr, out.ContractAddress)
		}

		if !shard.IsValidShardID(out.Shard.GetID()) {
			t.Errorf("out is not in the legal shard, the shardId of out is %d, it should be less than %d", out.Shard.GetID(), len(shard.ShardList))
		}

		if out.Index == 0 && !bytes.Equal(out.Data, scBytes) {

			t.Errorf("data with index of 0 is compiled code. It is wrong.\nwant: %v,\nget: %v", scBytes, out.Data)

		} else if retval := []byte{0x84, 99, 111, 100, 101}; out.Index == 1 && !bytes.Equal(out.Data, retval) {
			t.Errorf("data with index of 1 is Initialization data. It is wrong.\nwant: %v,\nget: %v", retval, out.Data)
		} else if out.Index == 2 && !bytes.Equal(out.Data, wire.MtvValueToData(value)) {

			t.Errorf("data with index of 2 is Refund data. It is wrong.\nwant: %v,\nget: %v", []byte{}, out.Data)

		}

		if out.Index < 0 || out.Index > 2 {
			t.Errorf("the index:%d of out is invalid, It should be between 0 and 2", out.Index)
		}

	}
}

// generateDepositTx gengerates a fake deposit out. If multiIn is true, return the tx with two
// txIn that has the same amount.
func generateDepositTx(shard shard.Index, amount int64, from multivacaddress.Address, to multivacaddress.Address,
	multiIn bool) (*wire.MsgTx, error) {

	dp := isysapi.DepositParams{
		TransferParams: &isysapi.TransferParams{
			To:     to,
			Amount: big.NewInt(amount),
			Shard:  shard,
		},
	}
	params, err := rlp.EncodeToBytes(dp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deposit params %v, err msg %v", dp, err)
	}

	// Construct a normal tx out.
	data := wire.MtvValueToData(big.NewInt(amount))
	out := wire.OutPoint{
		Data:        data,
		UserAddress: from,
	}
	in := []*wire.TxIn{
		{
			PreviousOutPoint: out,
		},
	}
	if multiIn {
		in = append(in, &wire.TxIn{
			PreviousOutPoint: out,
		})
	}

	// Construct a deposit tx.
	tx := &wire.MsgTx{
		Shard:           shard,
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIDeposit,
		Params:          params,
		TxIn:            in,
	}
	return tx, nil
}

func generateWithdrawTx(shard shard.Index, amount int64, from multivacaddress.Address, to multivacaddress.Address,
	multiIn bool, withdrawHeight int64) (*wire.MsgTx, error) {

	wp := isysapi.WithdrawParams{
		TransferParams: &isysapi.TransferParams{
			To:     to,
			Amount: big.NewInt(amount),
			Shard:  shard,
		},
	}
	params, err := rlp.EncodeToBytes(wp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deposit params %v, err msg %v", wp, err)
	}

	// Construct a deposit tx out for tx in.
	data := wire.MTVDepositData{
		Value:  big.NewInt(amount),
		Height: 1,
	}
	out := wire.OutPoint{
		Data:        data.ToData(),
		UserAddress: from,
		Type:        wire.DepositOut,
	}
	in := []*wire.TxIn{
		{
			PreviousOutPoint: out,
		},
	}
	if multiIn {
		in = append(in, &wire.TxIn{
			PreviousOutPoint: out,
		})
	}

	// Construct a withdraw tx.
	tx := &wire.MsgTx{
		Shard:           shard,
		ContractAddress: isysapi.SysAPIAddress,
		API:             isysapi.SysAPIWithdraw,
		Params:          params,
		TxIn:            in,
	}
	return tx, nil
}

func TestNormalDepositTx(t *testing.T) {
	sender := multivacaddress.Address([]byte("sender"))
	toAddr := multivacaddress.Address([]byte("toHash"))
	testShard := shard.Index(3)
	testAmount := int64(10)

	tx, err := generateDepositTx(testShard, testAmount, sender, toAddr, true)
	if err != nil {
		t.Errorf("faild to generate deposit tx, %v", err)
	}

	sce := NewSysCE(testShard)
	ops, scs, err := sce.Execute(tx, 1)
	if err != nil {
		t.Errorf("failed to execute deposit tx %v, err msg %v", tx, err)
	}
	if len(ops) < 1 {
		t.Errorf("unexpected length of ops %v", len(ops))
	}
	if scs != nil {
		t.Errorf("unexpected variable %v", scs)
	}

	// The out of deposit.
	depositData, err := wire.OutToDepositData(ops[0])
	if err != nil {
		t.Errorf("faild to get data from out, %v", err)
	}
	if ops[0].Shard != testShard || !ops[0].UserAddress.IsEqual(toAddr) ||
		depositData.Value.Cmp(big.NewInt(testAmount)) != 0 {
		t.Errorf("wrong deposit out point %v", ops[0])
	}

	// The out of original sender.
	mtvCoinData := wire.GetMtvValueFromOut(ops[1])
	if ops[1].Shard != testShard || !ops[1].UserAddress.IsEqual(sender) ||
		(mtvCoinData.Cmp(new(big.Int).Sub(new(big.Int).SetInt64(testAmount), isysapi.StorageRewardAmount)) != 0 && params.EnableStorageReward) {
		t.Errorf("wrong deposit out point %v", ops[1])
	}
}

// In current system, We stipulate that the amount of deposit must not be less than 10 MTV.
func TestSmallAmountDepositTx(t *testing.T) {
	sender := multivacaddress.Address([]byte("sender"))
	toAddr := multivacaddress.Address([]byte("toHash"))
	testShard := shard.Index(3)
	// Deposit 8 MTV
	testAmount := int64(8)

	tx, err := generateDepositTx(testShard, testAmount, sender, toAddr, true)
	if err != nil {
		t.Errorf("faild to generate deposit tx, %v", err)
	}

	sce := NewSysCE(testShard)
	_, _, err = sce.Execute(tx, 0)
	if err == nil {
		t.Errorf("execute tx should faild but success")
	}
}

func TestWithdrawTx(t *testing.T) {
	sender := multivacaddress.Address([]byte("sender"))
	toAddr := multivacaddress.Address([]byte("toHash"))
	testShard := shard.Index(3)
	testAmount := int64(8)
	testHeight := int64(20)

	tx, err := generateWithdrawTx(testShard, testAmount, sender, toAddr, true, testHeight)
	if err != nil {
		t.Errorf("faild to generate deposit tx, %v", err)
	}

	sce := NewSysCE(testShard)
	ops, scs, err := sce.Execute(tx, testHeight)
	if err != nil {
		t.Errorf("failed to execute withdraw tx %v, err msg %v", tx, err)
	}
	if len(ops) < 1 {
		t.Errorf("unexpected length of ops %v", len(ops))
	}
	if scs != nil {
		t.Errorf("unexpected variable %v", scs)
	}

	// The out of deposit.
	withdrawData, err := wire.OutToWithdrawData(ops[0])
	if err != nil ||
		ops[0].Shard != testShard ||
		!ops[0].UserAddress.IsEqual(toAddr) ||
		withdrawData.Value.Cmp(big.NewInt(testAmount)) != 0 ||
		withdrawData.WithdrawHeight != testHeight {
		t.Errorf("wrong withdraw out point %v", ops[0])
	}

	// The out of original sender.
	mtvCoinData := wire.GetMtvValueFromOut(ops[1])
	if ops[1].Shard != testShard || !ops[1].UserAddress.IsEqual(sender) ||
		(mtvCoinData.Cmp(new(big.Int).Sub(new(big.Int).SetInt64(testAmount), isysapi.StorageRewardAmount)) != 0 && params.EnableStorageReward) {
		t.Errorf("wrong deposit out point %v", ops[1])
	}
}
