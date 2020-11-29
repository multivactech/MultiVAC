/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"errors"
	"math/big"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/interface/imvvm"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/wire"

	"github.com/perlin-network/life/exec"
)

// executeMvvm is a metaphor of the MultiVAC Virtual Machine (MVVM), which is used in executing the smart contract.
//
// For now the virtual machine is a expand of Life (https://github.com/perlin-network/life).
type executeMvvm struct {
	vm exec.VirtualMachine
}

// NewExecuteMvvm returns a new executeMvvm instance.
func NewExecuteMvvm(code []byte) (imvvm.SmartContractMvvmExecutor, error) {

	// TODO (wangruichao@mtv.ac): consider VMConfig here.
	/*
		if len(code) == 0 {
			var err error
			//code, err = ioutil.ReadFile("/Users/aidongning/go/src/github.com/multivactech/MultiVAC/mvvm/main.wasm")
			code, err = ioutil.ReadFile("/Users/aidongning/go/src/github.com/multivactech/MultiVAC/mvvm/reduce.wasm")
			//code, err = ioutil.ReadFile("/Users/rikka/Code/Golang/src/github.com/multivactech/MultiVAC/mvvm/reduce.wasm")
			if err != nil {
				panic(fmt.Sprintf("Got error: %v", err))
			}
		} else {
			panic("!@#$##@#$@#%@")
		}
	*/

	log.Warnf("New deployMvvm: code with length %d", len(code))
	vm, err := exec.NewVirtualMachine(code, exec.VMConfig{
		DefaultTableSize:   defaultTableSize,
		DefaultMemoryPages: defaultMemoryPages,
	}, newDefaultResolver(resolverSize), mvvmGasPolicy)
	if err != nil {
		return nil, err
	}
	return &executeMvvm{vm: *vm}, nil
}

// Execute implements imvvm.SmartContractMvvmExecutor interface.
//
// When given a smart contract, dm.Execute will get the function ID of the API in the code, and run it.
// Execute returns shardOut, data of each shard, and userOuts, the slice of OutState containing user data.
func (dm *executeMvvm) Execute(tx *wire.MsgTx, shardInitOut *wire.OutState) (
	shardOut *wire.OutState, userOuts []*wire.OutState, reduceActions []*wire.ReduceAction, err error) {
	shardOut = shardInitOut
	shardDataOut := *shardInitOut
	totalValue := tx.TotalMtvValue().Int64()

	// If a smart contract is sent to MVVM, execute it.
	if !tx.ContractAddress.IsEqual(isysapi.SysAPIAddress) {
		// Transfer the shard initialization data and user data into memory.
		// TODO (wangruichao@mtv.ac): consider the suitable offset. 0 is not suitable in some cases.
		offset := defaultMemoryOffset

		var userData []inputUserData
		for _, in := range tx.TxIn {
			if !in.PreviousOutPoint.ContractAddress.IsEqual(isysapi.SysAPIAddress) {
				userData = append(userData, inputUserData{Data: in.PreviousOutPoint.Data})
			}
		}

		var length int
		length, err = storeIntoMemory(dm.vm.Memory, offset, mvvmInput{
			Params:    tx.Params,
			Shard:     tx.Shard,
			Caller:    tx.ContractAddress,
			MtvValue:  uint64(totalValue),
			ShardData: shardDataOut.Data,
			UserData:  userData,
		})
		if err != nil {
			log.Errorf("Error got when store input data into memory: %v\n", err)
			return
		}
		offset += length

		log.Debugf("Execute API with parameters: %s", tx.Params)
		id, ok := dm.vm.GetFunctionExport(tx.API)
		if !ok {
			err = errors.New("execute function not found")
			log.Errorf(err.Error())
			return
		}
		// Execute the code using MVVM.
		// Here all unspent OutPoint array are added to get the gas limit.
		// Note that if it is a reduce tx, the gas fee is not considered (or as a huge value)
		gasLimit := totalValue
		if totalValue > mvvmMaxGasLimit || tx.IsReduceTx() {
			gasLimit = mvvmMaxGasLimit
		}
		var res int64
		res, err = dm.vm.RunWithGasLimit(id, int(gasLimit), defaultMemoryOffset, int64(offset-defaultMemoryOffset))
		if err != nil {
			dm.vm.PrintStackTrace()
			log.Errorf("Error caught when running the code: %s", err)
			return
		}
		// Get the returned OutPoint from vm.Memory.
		var output mvvmOutput
		mem := dm.vm.Memory
		if err = rlp.DecodeBytes(mem[getMemUint32(mem, int(res)):], &output); err != nil {
			switch err {
			case rlp.ErrMoreThanOneValue:
				// When getting more than one value during the decode process, ignore it and go on.
				log.Warnf("Warning caught when getting returned value: %s", err)
				err = nil
			default:
				log.Errorf("Error caught when getting returned value: %s", err)
				return
			}
		}
		shardDataOut.Data = output.ShardData

		var newOuts []*wire.OutState
		var index int
		if newOuts, index, err = generateOuts(tx, totalValue, output, int64(dm.vm.Gas)); err != nil {
			return
		}
		userOuts = append(userOuts, newOuts...)

		for _, reduceData := range output.ReduceData {
			newReduceAction := wire.ReduceAction{
				ToBeReducedOut: &wire.OutPoint{
					Shard:           reduceData.Shard,
					TxHash:          tx.TxHash(),
					Index:           index,
					Data:            reduceData.Data,
					ContractAddress: tx.ContractAddress,
				},
				API: string(reduceData.API),
			}
			reduceActions = append(reduceActions, &newReduceAction)
			index++
		}
	}
	shardOut = &shardDataOut
	return
}

// generateOuts returns the generated OutPoint and change out (gas fee, 3rd-party cost, etc.) in MVVM.
func generateOuts(tx *wire.MsgTx, value int64, output mvvmOutput, gasfee int64) (outs []*wire.OutState, index int, err error) {

	log.Debugf("In generateOuts:\n")
	log.Debugf("tx = %+v\n", tx)
	log.Debugf("gasfee = %v\n", gasfee)
	log.Debugf("output = \n")

	from := tx.TxIn[0].PreviousOutPoint
	txhash := tx.TxHash()

	for i, ud := range output.UserData {
		log.Debugf("userdata[%v] = %+v\n", i, ud)
		newOut := wire.OutPoint{
			Shard:           ud.Shard,
			TxHash:          txhash,
			Index:           index,
			UserAddress:     ud.UserAddress,
			Data:            ud.Data,
			ContractAddress: tx.ContractAddress,
		}
		outs = append(outs, newOut.ToUnspentOutState())
		index++
	}

	for i, cost := range output.MtvData {
		log.Debugf("mtvdata[%v] = %+v\n", i, cost)
		v := int64(cost.Value)
		if v < 0 || v > value {
			// If the change after being spent in MVVM is larger than the total value, return error.
			err = errors.New("invalid change value after executing smart contract")
			return
		}
		out := wire.OutPoint{
			Shard:           cost.Shard,
			TxHash:          txhash,
			Index:           index,
			UserAddress:     cost.UserAddress,
			Data:            wire.MtvValueToData(new(big.Int).SetUint64(cost.Value)),
			ContractAddress: tx.ContractAddress,
		}
		outs = append(outs, out.ToUnspentOutState())
		value -= v
		index++
	}

	// If it is a reduce tx, gas fee is not considered now.
	if !tx.IsReduceTx() {
		if value < gasfee {
			err = errors.New("not enough gas fee after mtv cost")
			return
		}
		change := wire.OutPoint{
			Shard:           tx.Shard,
			TxHash:          txhash,
			Index:           index,
			UserAddress:     from.UserAddress,
			Data:            wire.MtvValueToData(big.NewInt(value - gasfee)),
			ContractAddress: isysapi.SysAPIAddress,
		}
		outs = append(outs, change.ToUnspentOutState())
		index++
	}

	return
}
