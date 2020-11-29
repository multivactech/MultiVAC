// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iabci

import (
	"github.com/multivactech/MultiVAC/model/wire"
)

// AppBlockChain is used to manage merkle tree.
type AppBlockChain interface {
	VerifyBlock(block *wire.MsgBlock) error
	ProposeBlock(pk []byte, sk []byte) (*wire.MsgBlock, error)
	ProposeAndAcceptEmptyBlock(height int64) (*wire.BlockHeader, error)
	AcceptBlock(block *wire.MsgBlock) error
	Start(prevReshardheader *wire.BlockHeader)
	Stop()
	PreStart()
}
