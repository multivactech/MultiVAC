/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"math"

	"github.com/perlin-network/life/compiler"
)

var (
	// mvvmGasPolicy is the gas policy for MVVM.
	//
	// In default cases, assume that each single step will consume the unit gas fee.
	mvvmGasPolicy = &compiler.SimpleGasPolicy{GasPerInstruction: 1}

	// mvvmDefaultGasLimit is the default gas limit for MVVM.
	//
	// Mostly, it is assumed that the developer will give a user-defined gas limit. But if the limit is not assigned,
	// mvvmDefaultGasLimit will be used as the limit value.
	// For now the value is set as 65535. Deeper discussion is needed.
	// Note that mvvmDefaultGasLimit > 0. 'cause if mvvmDefaultGasLimit <= 0, gas fee will never exceed.
	mvvmDefaultGasLimit int64 = 1<<16 - 1

	// mvvmMaxGasLimit is the maximum gas limit for MVVM.
	//
	// The MTV Coin value cost for gas is a big.Int, which is sometimes much bigger than the gas limit for life.
	// Hence it is assumed that if the coin value is bigger than mvvmDefaultGasLimit, the gas limit is the maximum limit.
	// For now this value is set as the maximum value of int64.
	mvvmMaxGasLimit int64 = math.MaxInt64
)
