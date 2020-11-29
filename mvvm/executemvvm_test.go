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
	"reflect"
	"testing"

	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestNewExecuteMvvm(t *testing.T) {
	tests := []struct {
		tx            wire.MsgTx           // Input transaction
		shardInit     wire.OutPoint        // Input shard init data
		shardOut      *wire.OutState       // Output shard data
		userOuts      []*wire.OutState     // Output user data
		reduceActions []*wire.ReduceAction // Output reduce action
		err           error                // Executing error
	}{
		{
			tx: wire.MsgTx{
				Version: wire.TxVersion,
				Shard:   shard.Index(1),
				TxIn: []*wire.TxIn{
					wire.NewTxIn(&wire.OutPoint{
						Shard:           shard.Index(1),
						Data:            wire.MtvValueToData(big.NewInt(4000000)),
						ContractAddress: isysapi.SysAPIAddress,
					}), wire.NewTxIn(&wire.OutPoint{
						Shard: shard.Index(1),
						Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
						ContractAddress: multivacaddress.GenerateAddress(
							signature.PublicKey([]byte{255}),
							multivacaddress.SmartContractAddress,
						),
					}),
				},
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
				API:    "__Z7wrapperPhi",
				Params: []byte("abcdefghijklmnopqrstuvwxyz"),
			},
			shardInit: wire.OutPoint{
				Shard: shard.Index(1),
				Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
			},
			shardOut: &wire.OutState{
				OutPoint: wire.OutPoint{
					Shard:       shard.Index(1),
					TxHash:      chainhash.Hash{},
					Index:       0,
					UserAddress: multivacaddress.Address(nil),
					Data:        []byte("abcdefghijklmnopqrstuvwx"),
					ContractAddress: multivacaddress.Address{
						0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a, 0x50,
						0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66, 0x64, 0x48,
						0x50, 0x32, 0x54, 0x43, 0x56,
					},
				},
				State: wire.StateUnused,
			},
			userOuts: []*wire.OutState{
				{
					OutPoint: wire.OutPoint{
						Shard: shard.Index(1),
						TxHash: chainhash.Hash{
							0x5a, 0x53, 0xa9, 0x37, 0x71, 0x9e, 0xe5, 0xf1, 0x70, 0x38, 0x5b, 0x78, 0x15, 0xab, 0x15,
							0x90, 0x72, 0xa2, 0x89, 0x36, 0x44, 0xfa, 0xf9, 0xed, 0x13, 0xec, 0xda, 0x93, 0x43, 0xcf,
							0xbd, 0x26,
						},
						Index: 0,
						UserAddress: multivacaddress.Address{
							0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a,
							0x50, 0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66,
							0x64, 0x48, 0x50, 0x32, 0x54, 0x43, 0x56,
						},
						Data: []byte("abcdefghijklmnopqrstuvwx"),
						ContractAddress: multivacaddress.Address{
							0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a,
							0x50, 0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66,
							0x64, 0x48, 0x50, 0x32, 0x54, 0x43, 0x56,
						},
					},
					State: wire.StateUnused,
				},
				{
					OutPoint: wire.OutPoint{
						Shard: shard.Index(1),
						TxHash: chainhash.Hash{
							0x5a, 0x53, 0xa9, 0x37, 0x71, 0x9e, 0xe5, 0xf1, 0x70, 0x38, 0x5b, 0x78, 0x15, 0xab, 0x15,
							0x90, 0x72, 0xa2, 0x89, 0x36, 0x44, 0xfa, 0xf9, 0xed, 0x13, 0xec, 0xda, 0x93, 0x43, 0xcf,
							0xbd, 0x26,
						},
						Index: 1,
						UserAddress: multivacaddress.Address{
							0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a,
							0x50, 0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66,
							0x64, 0x48, 0x50, 0x32, 0x54, 0x43, 0x56,
						},
						Data: []byte{0xc3, 0x82, 0x3, 0xe8},
						ContractAddress: multivacaddress.Address{
							0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a,
							0x50, 0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66,
							0x64, 0x48, 0x50, 0x32, 0x54, 0x43, 0x56,
						},
					},
					State: wire.StateUnused,
				},
				{
					OutPoint: wire.OutPoint{
						Shard: shard.Index(1),
						TxHash: chainhash.Hash{
							0x5a, 0x53, 0xa9, 0x37, 0x71, 0x9e, 0xe5, 0xf1, 0x70, 0x38, 0x5b, 0x78, 0x15, 0xab, 0x15,
							0x90, 0x72, 0xa2, 0x89, 0x36, 0x44, 0xfa, 0xf9, 0xed, 0x13, 0xec, 0xda, 0x93, 0x43, 0xcf,
							0xbd, 0x26,
						},
						Index:       2,
						UserAddress: multivacaddress.Address(nil),
						Data:        []byte{0xc4, 0x83, 0x2d, 0xaa, 0x93},
						ContractAddress: multivacaddress.Address{
							0x4d, 0x54, 0x56, 0x51, 0x4c, 0x62, 0x7a, 0x37, 0x4a, 0x48, 0x69, 0x42, 0x54, 0x73, 0x70,
							0x53, 0x39, 0x36, 0x32, 0x52, 0x4c, 0x4b, 0x56, 0x38, 0x47, 0x6e, 0x64, 0x57, 0x46, 0x77,
							0x6a, 0x41, 0x35, 0x4b, 0x36, 0x36,
						},
					},
					State: wire.StateUnused,
				},
			},
			reduceActions: []*wire.ReduceAction(nil),
			err:           nil,
		},
		{
			tx: wire.MsgTx{
				Version: wire.TxVersion,
				Shard:   shard.Index(1),
				TxIn: []*wire.TxIn{
					wire.NewTxIn(&wire.OutPoint{
						Shard:           shard.Index(1),
						Data:            wire.MtvValueToData(big.NewInt(1024)),
						ContractAddress: isysapi.SysAPIAddress,
					}), wire.NewTxIn(&wire.OutPoint{
						Shard: shard.Index(1),
						Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
						ContractAddress: multivacaddress.GenerateAddress(
							signature.PublicKey([]byte{255}),
							multivacaddress.SmartContractAddress,
						),
					}),
				},
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
				API:    "__Z7wrapperPhi",
				Params: []byte("abcdefghijklmnopqrstuvwxyz"),
			},
			shardInit: wire.OutPoint{
				Shard: shard.Index(1),
				Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
			},
			shardOut: &wire.OutState{
				OutPoint: wire.OutPoint{
					Shard: shard.Index(1),
					Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
					ContractAddress: multivacaddress.GenerateAddress(
						signature.PublicKey([]byte{255}),
						multivacaddress.SmartContractAddress,
					),
				},
				State: wire.StateUnused,
			},
			userOuts:      []*wire.OutState(nil),
			reduceActions: []*wire.ReduceAction(nil),
			err:           errors.New("not enough gas fee after mtv cost"),
		},
		{
			tx: wire.MsgTx{
				Version: wire.TxVersion,
				Shard:   shard.Index(1),
				TxIn: []*wire.TxIn{
					wire.NewTxIn(&wire.OutPoint{
						Shard:           shard.Index(1),
						Data:            wire.MtvValueToData(big.NewInt(1024)),
						ContractAddress: isysapi.SysAPIAddress,
					}), wire.NewTxIn(&wire.OutPoint{
						Shard: shard.Index(1),
						Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
						ContractAddress: multivacaddress.GenerateAddress(
							signature.PublicKey([]byte{255}),
							multivacaddress.SmartContractAddress,
						),
					}),
				},
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
				API:    "__Z7wrapperPhi",
				Params: []byte("abcdefghijklmnopqrstuvwxyz"),
			},
			shardInit: wire.OutPoint{
				Shard: shard.Index(1),
				Data:  []byte{},
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
			},
			shardOut: &wire.OutState{
				OutPoint: wire.OutPoint{
					Shard: shard.Index(1),
					Data:  []byte{},
					ContractAddress: multivacaddress.GenerateAddress(
						signature.PublicKey([]byte{255}),
						multivacaddress.SmartContractAddress,
					),
				},
				State: wire.StateUnused,
			},
			userOuts:      []*wire.OutState(nil),
			reduceActions: []*wire.ReduceAction(nil),
			err:           errors.New("runtime error: index out of range"),
		},
		{
			tx: wire.MsgTx{
				Version: wire.TxVersion,
				Shard:   shard.Index(1),
				TxIn: []*wire.TxIn{
					wire.NewTxIn(&wire.OutPoint{
						Shard:           shard.Index(1),
						Data:            wire.MtvValueToData(big.NewInt(4000000)),
						ContractAddress: isysapi.SysAPIAddress,
					}), wire.NewTxIn(&wire.OutPoint{
						Shard: shard.Index(1),
						Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
						ContractAddress: multivacaddress.GenerateAddress(
							signature.PublicKey([]byte{255}),
							multivacaddress.SmartContractAddress,
						),
					}),
				},
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
				API:    "wrapper",
				Params: []byte("abcdefghijklmnopqrstuvwxyz"),
			},
			shardInit: wire.OutPoint{
				Shard: shard.Index(1),
				Data:  []byte("abcdefghijklmnopqrstuvwxyz"),
				ContractAddress: multivacaddress.GenerateAddress(
					signature.PublicKey([]byte{255}),
					multivacaddress.SmartContractAddress,
				),
			},
			shardOut: &wire.OutState{
				OutPoint: wire.OutPoint{
					Shard:       shard.Index(1),
					TxHash:      chainhash.Hash{},
					Index:       0,
					UserAddress: multivacaddress.Address(nil),
					Data:        []byte("abcdefghijklmnopqrstuvwxyz"),
					ContractAddress: multivacaddress.Address{
						0x4d, 0x54, 0x56, 0x33, 0x51, 0x77, 0x4c, 0x45, 0x54, 0x66, 0x51, 0x71, 0x75, 0x59, 0x5a, 0x50,
						0x34, 0x46, 0x48, 0x35, 0x66, 0x48, 0x62, 0x77, 0x54, 0x50, 0x50, 0x4c, 0x6a, 0x66, 0x64, 0x48,
						0x50, 0x32, 0x54, 0x43, 0x56,
					},
				},
				State: wire.StateUnused,
			},
			userOuts:      []*wire.OutState(nil),
			reduceActions: []*wire.ReduceAction(nil),
			err:           errors.New("execute function not found"),
		},
	}
	code := getTestCode("test_exec.wasm")
	vm, err := NewExecuteMvvm(code)
	if err != nil {
		t.Error(err)
	}
	for i, test := range tests {
		shardInit := test.shardInit.ToUnspentOutState()
		defer func() {
			if r := recover(); r != nil {
				t.Error("runtime error when executing code")
			}
		}()
		shardOut, userOuts, reduceActions, err := vm.Execute(&test.tx, shardInit)
		if test.err == nil {
			if err != nil {
				t.Errorf("execute smart contract #%d got error when executing code: %v\n", i, err)
			} else if !reflect.DeepEqual(shardOut, test.shardOut) {
				t.Errorf("execute smart contract #%d got incorrect shard output: got %v, want %v\n",
					i, shardOut, test.shardOut)
			} else if !reflect.DeepEqual(userOuts, test.userOuts) {
				t.Errorf("execute smart contract #%d got incorrect user output: got %v, want %v\n",
					i, userOuts, test.userOuts)
			} else if !reflect.DeepEqual(reduceActions, test.reduceActions) {
				t.Errorf("execute smart contract #%d got incorrect reduce actions: got %v, want %v\n",
					i, reduceActions, test.reduceActions)
			}
		} else {
			if err.Error() != test.err.Error() {
				t.Errorf("execute smart contract #%d got incorrect error when executing code: got %v, want %v\n",
					i, err.Error(), test.err.Error())
			} else if !reflect.DeepEqual(shardOut, test.shardOut) {
				t.Errorf("execute smart contract #%d got incorrect shard data when caught error during code executing: "+
					"got %v, want %v\n", i, shardOut, test.shardOut)
			} else if !reflect.DeepEqual(userOuts, test.userOuts) {
				t.Errorf("execute smart contract #%d got incorrect user data when caught error during code executing: "+
					"got %v, want %v\n", i, userOuts, test.userOuts)
			} else if !reflect.DeepEqual(reduceActions, test.reduceActions) {
				t.Errorf("execute smart contract #%d got incorrect reduce action when caught error during code executing: "+
					"got %v, want %v\n", i, reduceActions, test.reduceActions)
			}
		}
	}
}
