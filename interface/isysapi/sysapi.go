// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package isysapi

import (
	"math/big"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
)

const (
	// SysAPITransfer : deal with transfer type transaction.
	SysAPITransfer = "transfer"
	// SysAPIDeposit : deal with deposit type transaction.
	SysAPIDeposit = "deposit"
	// SysAPIWithdraw : deal with withdraw type transaction.
	SysAPIWithdraw = "withdraw"
	// SysAPIReward : deal with reward type transaction.
	SysAPIReward = "reward"
	// SysAPIDeploy : deal with deploy type transaction.
	SysAPIDeploy = "deploy"
	// SysAPIBinding : deal with binding type transaction.
	SysAPIBinding = "binding"
)

var (
	// MinerRewardAmount proposes block reward.
	MinerRewardAmount = new(big.Int).SetUint64(5e18)

	// StorageRewardAmount node service reward.
	StorageRewardAmount = new(big.Int).SetUint64(3e18)

	// SysAPIAddress is should be immutable.
	SysAPIAddress = multivacaddress.GenerateAddress(signature.PublicKey([]byte{}),
		multivacaddress.SystemContractAddress)
)

// TxParameter specifies the parameter of a transaction.
type TxParameter interface {
	GetTo() multivacaddress.Address
	GetShard() shard.Index
	GetAmount() *big.Int
}

// TransferParams is the parameter of normal transaction.
type TransferParams struct {
	To     multivacaddress.Address
	Shard  shard.Index
	Amount *big.Int
}

// GetTo get the To address.
func (tp TransferParams) GetTo() multivacaddress.Address {
	return tp.To
}

// GetShard get the shard of tp.
func (tp TransferParams) GetShard() shard.Index {
	return tp.Shard
}

// GetAmount get tp's amount.
func (tp TransferParams) GetAmount() *big.Int {
	return tp.Amount
}

// DepositParams is the parameter of deposit transaction.
type DepositParams struct {
	*TransferParams
	BindingAddress multivacaddress.Address
}

// WithdrawParams is the parameter of withdraw transaction.
type WithdrawParams struct {
	*TransferParams
}

// RewardParams is the parameter of reward transaction.
type RewardParams struct {
	To     multivacaddress.Address
	Amount *big.Int
}

// DeployParams is the parameter of a smart contract deployment.
type DeployParams struct {
	APIList []string
	// Store the  init code , it will be builded by VM and create shard data outputs which index is 1
	Init []byte
	// Store the  smart contract code, it will be builded by VM
	Code []byte
}

// BindingParams is the params of binding trasaction
// See docs.md to get more details of binding transaction
type BindingParams struct {
	*TransferParams
	BindingAddress multivacaddress.Address
}
