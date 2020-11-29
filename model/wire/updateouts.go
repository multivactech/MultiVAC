// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/model/merkle"
)

// UpdateAction 是矿工根据计算结果对存储节点发送的修改数据的请求
type UpdateAction struct {
	OriginOut      *OutState
	OriginOutProof *merkle.MerklePath
	NewOut         *OutState
}
