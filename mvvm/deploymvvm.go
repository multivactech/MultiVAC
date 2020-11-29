/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"github.com/perlin-network/life/exec"

	"github.com/multivactech/MultiVAC/interface/imvvm"
	"github.com/multivactech/MultiVAC/model/shard"
)

// deployMvvm is a metaphor of the MultiVAC Virtual Machine (MVVM), which is used in deploying the smart contract.
//
// For now the virtual machine is a expand of Life (https://github.com/perlin-network/life).
type deployMvvm struct {
	vm exec.VirtualMachine
}

// NewDeployMvvm returns a new deployMVVM instance.
func NewDeployMvvm(code []byte) (imvvm.DeployInitializer, error) {
	log.Debugf("New deployMvvm: code: %s", code)
	vm, err := exec.NewVirtualMachine(code, exec.VMConfig{
		DefaultTableSize:   defaultTableSize,
		DefaultMemoryPages: defaultMemoryPages,
	}, newDefaultResolver(resolverSize), mvvmGasPolicy)
	if err != nil {
		return nil, err
	}
	return &deployMvvm{vm: *vm}, nil
}

// Initialize implements imvvm.DeployInitializer interface.
func (dm *deployMvvm) Initialize(index shard.Index) []byte {
	// For developers, they implement the "init" method use C/C++/Rust..., and they compile the method into .wasm.
	// For virtual machine, in order to invoke the "init" method from the given .wasm code, the VM should get the
	// function ID of the "init" method, and invoke VM.RunWithGasLimit(id, params...).
	log.Debugf("Initialize Index %s", index)
	id, ok := dm.vm.GetFunctionExport(getWasmFuncName("init", integerParam))
	if !ok {
		log.Errorf("Initialize function not found")
		panic("initialize function not found")
	}
	res, err := dm.vm.RunWithGasLimit(id, int(mvvmDefaultGasLimit), int64(index))
	if err != nil {
		dm.vm.PrintStackTrace()
		log.Errorf("Error caught when Run the Initialize code: %s", err)
		panic(err)
	}
	if mem, err := validMemory(dm.vm.Memory, int(res)); err != nil {
		log.Errorf("Error caught when getting data from memory: %s", err)
		panic(err)
	} else {
		return mem
	}
}
