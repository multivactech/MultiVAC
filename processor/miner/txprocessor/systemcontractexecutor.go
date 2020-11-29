/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txprocessor

import (
	"fmt"
	"math/big"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/configs/params"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/mvvm"
)

// NewSysCE returns a instance of SysContractExecutor.
// Visible for testing only, as in for txgenerator. Don't use it outside of this package
// for other reason.
func NewSysCE(shard shard.Index) *SysContractExecutor {
	return &SysContractExecutor{
		shard: shard,
	}
}

// SysContractExecutor is an executor to execute system contract.
type SysContractExecutor struct {
	shard         shard.Index
	rewardTxCount int
}

// Execute system contract using given tx and propose block height.
func (sce *SysContractExecutor) Execute(tx *wire.MsgTx, proposeBlockHeight int64) ([]*wire.OutPoint, *wire.SmartContract, error) {
	var ops []*wire.OutPoint
	var err error
	if len(tx.API) == 0 || len(tx.Params) == 0 {
		return nil, nil, fmt.Errorf("the API or Params is empty, tx %v", tx)
	}
	switch tx.API {
	case isysapi.SysAPITransfer:
		var tp isysapi.TransferParams
		err = rlp.DecodeBytes(tx.Params, &tp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}

		outData := wire.MtvValueToData(tp.Amount)
		ops, err = executeTransaction(sce.shard, tx, outData, tp, tx.API)
		if err != nil {
			return nil, nil, err
		}

		return ops, nil, nil
	case isysapi.SysAPIDeposit:
		var dp isysapi.DepositParams
		err = rlp.DecodeBytes(tx.Params, &dp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}
		// Verify deoposit out, deposit amount can not less than specified value.
		// TODO: Should it be handled here?
		if wire.MinimalDepositAmount.Cmp(dp.Amount) > 0 {
			return nil, nil, fmt.Errorf("deposit amount is too little")
		}

		// Generate a deposit out data.
		depositData := &wire.MTVDepositData{
			Value:          dp.Amount,
			Height:         proposeBlockHeight,
			BindingAddress: dp.BindingAddress,
		}

		ops, err = executeTransaction(sce.shard, tx, depositData.ToData(), dp, tx.API)
		if err != nil {
			return nil, nil, err
		}

		return ops, nil, nil
	case isysapi.SysAPIWithdraw:
		var wp isysapi.WithdrawParams
		err = rlp.DecodeBytes(tx.Params, &wp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}
		// Generate a withdraw out data.
		withdrawData := &wire.MTVWithdrawData{
			Value:          wp.Amount,
			WithdrawHeight: proposeBlockHeight,
		}
		ops, err = executeTransaction(sce.shard, tx, withdrawData.ToData(), wp, tx.API)
		if err != nil {
			return nil, nil, err
		}
		return ops, nil, nil
	case isysapi.SysAPIBinding:
		var bp isysapi.BindingParams
		err = rlp.DecodeBytes(tx.Params, &bp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}
		// Generate binding out data
		bindingData := &wire.MTVDepositData{
			Value:          bp.Amount,
			Height:         proposeBlockHeight,
			BindingAddress: bp.BindingAddress,
		}
		ops, err = executeTransaction(sce.shard, tx, bindingData.ToData(), bp, tx.API)
		if err != nil {
			return nil, nil, err
		}
		return ops, nil, nil
	case isysapi.SysAPIReward:
		if !params.EnableStorageReward && sce.rewardTxCount > 0 {
			return nil, nil, fmt.Errorf("Only 1 reward tx allowed. tx %v", tx)
		}
		var rp isysapi.RewardParams
		err = rlp.DecodeBytes(tx.Params, &rp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}
		if rp.Amount == nil {
			return nil, nil, fmt.Errorf("Wrong reward amount, amount %v, tx %v", rp.Amount, tx)
		}
		defaultShard := multivacaddress.GetDefaultAddrShard(rp.To.String())
		// Reward tx always go to same shard as the tx
		ops = append(ops, &wire.OutPoint{
			Shard:           shard.Index(defaultShard),
			TxHash:          tx.TxHash(),
			Index:           0,
			UserAddress:     rp.To,
			Data:            wire.MtvValueToData(rp.Amount),
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.Transfer,
		})
		sce.rewardTxCount++
		return ops, nil, nil
	case isysapi.SysAPIDeploy:
		// resolve params
		var dp isysapi.DeployParams
		err = rlp.DecodeBytes(tx.Params, &dp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal params, err msg %v, tx %v", err, tx)
		}
		if len(dp.APIList) == 0 || len(dp.Code) == 0 || len(dp.Init) == 0 {
			return nil, nil, fmt.Errorf("params error, len(APIList): %d,len(Code): %d,len(Init): %d", len(dp.APIList), len(dp.Code), len(dp.Init))
		}

		txHash := tx.TxHash()
		sCAddr := txHash.FormatSmartContractAddress()

		// Initialize SmartContract according to the tx.Params
		sc := &wire.SmartContract{ContractAddr: sCAddr, APIList: dp.APIList, Code: dp.Code}

		// To build the data of outPoint at index==0
		scHash := sc.SmartContractHash()
		scBytes := scHash.CloneBytes()

		vm, err := mvvm.NewDeployMvvm(dp.Init)
		if err != nil {
			return nil, nil, err
		}

		for _, shardIdx := range shard.ShardList {
			out1 := &wire.OutPoint{
				Shard:           shardIdx,
				TxHash:          txHash,
				Index:           0,
				Data:            scBytes,
				ContractAddress: sc.ContractAddr,
				Type:            wire.Transfer,
			}

			out2 := &wire.OutPoint{
				Shard:           shardIdx,
				TxHash:          txHash,
				Index:           1,
				Data:            vm.Initialize(shardIdx),
				ContractAddress: sc.ContractAddr,
				Type:            wire.Transfer,
			}
			ops = append(ops, out1, out2)
		}

		// TODO(#issue 203): the logic of gas should be defined when the VM has been implemented
		gas := big.NewInt(0)

		totalInputs := big.NewInt(0)
		for _, txIn := range tx.TxIn {
			value := wire.GetMtvValueFromOut(&txIn.PreviousOutPoint)
			totalInputs = new(big.Int).Add(totalInputs, value)
		}

		if totalInputs.Cmp(gas) < 0 {
			return nil, nil, fmt.Errorf("totalInpus is smaller than gas")
		}

		// Refund data
		value := new(big.Int).Sub(totalInputs, gas)
		if params.EnableStorageReward {
			value = new(big.Int).Sub(value, isysapi.StorageRewardAmount)

			if value.Cmp(big.NewInt(0)) < 0 {
				return nil, nil, fmt.Errorf("the depoly tx is not enough for storage reward")
			}
		}
		ops = append(ops, &wire.OutPoint{
			Shard:           sce.shard,
			TxHash:          tx.TxHash(),
			Index:           2,
			UserAddress:     tx.TxIn[0].PreviousOutPoint.UserAddress,
			Data:            wire.MtvValueToData(value),
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.Transfer,
		})

		return ops, sc, nil
	default:
		return nil, nil, fmt.Errorf("unknown API %v, tx %v", tx.API, tx)
	}
}

func executeTransaction(secShard shard.Index, tx *wire.MsgTx, outData []byte,
	param isysapi.TxParameter, apiType string) ([]*wire.OutPoint, error) {

	// TODO:(zz) different type for different tx
	// Not sure type: deploy, reward
	var outType wire.OutType
	switch tx.API {
	case isysapi.SysAPITransfer:
		outType = wire.Transfer
	case isysapi.SysAPIDeposit:
		outType = wire.DepositOut
	case isysapi.SysAPIBinding:
		outType = wire.DepositOut
	case isysapi.SysAPIWithdraw:
		outType = wire.WithdrawOut
	case isysapi.SysAPIDeploy:
		outType = wire.Transfer
	case isysapi.SysAPIReward:
		outType = wire.Transfer
	}

	var ops []*wire.OutPoint
	// Check the legality of shardID
	if !shard.IsValidShardID(shard.IndexToID(param.GetShard())) {
		return nil, fmt.Errorf("invalid shard id: %v, tx %v", param.GetShard(), tx)
	}

	// Get total input amount by traversing
	totalInputs := big.NewInt(0)
	for _, txIn := range tx.TxIn {
		var value *big.Int
		// Withdraw out
		if txIn.PreviousOutPoint.IsWithdrawOut() {
			withdrawData, err := wire.OutToWithdrawData(&txIn.PreviousOutPoint)
			if err != nil {
				return nil, err
			}
			value = withdrawData.Value
			totalInputs = new(big.Int).Add(totalInputs, value)
			continue
		}

		// Deposit out
		if txIn.PreviousOutPoint.IsDepositOut() {
			depositData, err := wire.OutToDepositData(&txIn.PreviousOutPoint)
			if err != nil {
				return nil, err
			}
			value = depositData.Value
			totalInputs = new(big.Int).Add(totalInputs, value)
			continue
		}

		// Transfer out
		value = wire.GetMtvValueFromOut(&txIn.PreviousOutPoint)
		totalInputs = new(big.Int).Add(totalInputs, value)
	}

	// Check the integrity of the transaction
	// - Tx should have amount
	// - Total amount of txIn >= total amount of txOut
	// - Signature should't be nil
	if param.GetAmount() == nil || totalInputs.Cmp(param.GetAmount()) < 0 || param.GetAmount().Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount to transfer %v, tx %v", param.GetAmount(), tx)
	}

	// The first index of out
	ops = append(ops, &wire.OutPoint{
		Shard:           param.GetShard(),
		TxHash:          tx.TxHash(),
		Index:           0,
		UserAddress:     param.GetTo(),
		Data:            outData,
		ContractAddress: isysapi.SysAPIAddress,
		Type:            outType,
	})

	// Set value = totalInputs + (-amount)
	// Why not use big.Int.Sub()?
	value := new(big.Int).Add(totalInputs, new(big.Int).Neg(param.GetAmount()))

	// If enabled storagenode reward, totalInput - reward should larger or equal than output amount
	if params.EnableStorageReward {
		value = new(big.Int).Sub(value, isysapi.StorageRewardAmount)
		if value.Cmp(big.NewInt(0)) < 0 {
			return nil, fmt.Errorf("this tx not enough for storage node reward")
		}
	}

	// If there's more inputs than outpus, return extra money to the orignal sender.
	// TODO(issues#167): Implement transaction fee.
	if value.Cmp(big.NewInt(0)) > 0 {
		ops = append(ops, &wire.OutPoint{
			Shard:           secShard,
			TxHash:          tx.TxHash(),
			Index:           1,
			UserAddress:     tx.TxIn[0].PreviousOutPoint.UserAddress,
			Data:            wire.MtvValueToData(value),
			ContractAddress: isysapi.SysAPIAddress,
			Type:            wire.Transfer,
		})
	}
	return ops, nil
}
