// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iconsensus

import (
	"github.com/multivactech/MultiVAC/model/shard"
)

// DepositFetcher get deposited information.
type DepositFetcher interface {
	GetDepositUnit(pk []byte, round int32, index shard.Index) uint32
}
