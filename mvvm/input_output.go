/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/shard"
)

// inputUserData is the user data of the input.
//
// TODO (wangruichao@mtv.ac): Add Metadata into inputUserData.
type inputUserData struct {
	Data []byte
}

// outputUserData is the user data of the output.
type outputUserData struct {
	Shard       shard.Index
	UserAddress multivacaddress.Address
	Data        []byte
}

// outputMtvData is the MTV cost data of the output.
type outputMtvData struct {
	Shard       shard.Index
	UserAddress multivacaddress.Address
	Value       uint64
}

// outputReduceData is the data in the reduce actions of the output.
type outputReduceData struct {
	Shard shard.Index
	Data  []byte
	API   []byte
}

// mvvmInput is the input data transfered into MVVM when a smart contract is applied and executed.
type mvvmInput struct {
	Params    []byte                  // Parameter of the input.
	Shard     shard.Index             // Shard index of the input.
	Caller    multivacaddress.Address // Caller hash of the input.
	MtvValue  uint64                  // Total MTV value spent in executing the smart contract.
	ShardData []byte                  // Shard init data of the smart contract.
	UserData  []inputUserData         // User data of the smart contract.
}

// mvvmOutput is the output data returned from MVVM when a smart contract is applied and executed.
// In fact, for now it just returns the modified shard data and the added user data (*wire.OutState).
type mvvmOutput struct {
	ShardData  []byte             // Shard data of the returned value.
	UserData   []outputUserData   // User data of the returned value.
	MtvData    []outputMtvData    // Change for 3rd-party cost of the returned value.
	ReduceData []outputReduceData // Data for ReduceActions of the returned value.
}
