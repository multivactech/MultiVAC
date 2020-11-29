// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

// ReduceAction is used to deal with the issue of cross-shard-update
type ReduceAction struct {
	ToBeReducedOut *OutPoint // OutState that needs to be merged
	API            string    // ReduceAction needs to call the smart contract API
}
