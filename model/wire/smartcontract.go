// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
)

// SmartContract defines the data structure of smart contract.
type SmartContract struct {
	ContractAddr multivacaddress.Address
	APIList      []string
	// Store a piece of code which is builded by VM
	Code []byte
}

// SmartContractInfo defines the data structure of smart contract info.
type SmartContractInfo struct {
	MsgID             uint32
	SmartContract     *SmartContract     // 智能合约数据
	ShardIdx          shard.Index        // 分片编号
	CodeOut           *OutState          // 智能合约代码Out
	CodeOutProof      *merkle.MerklePath // 智能合约代码Out的merkle path
	ShardInitOut      *OutState          // 智能合约分片初始化数据
	ShardInitOutProof *merkle.MerklePath // 智能合约分片初始化数据的merkle path
}

// BtcDecode use rlp serialization to decode message.
func (msg *SmartContractInfo) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode use rlp serialization to encode message.
func (msg *SmartContractInfo) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the command string.
func (msg *SmartContractInfo) Command() string {
	return CmdSmartContractInfo
}

// MaxPayloadLength returns the max playload of block.
func (msg *SmartContractInfo) MaxPayloadLength(uint32) uint32 {
	return MaxBlockPayload
}

// Verify is to varify the smart contract info is right.
func (msg *SmartContractInfo) Verify(root *merkle.MerkleHash) error {
	if msg.CodeOut == nil || msg.CodeOutProof == nil {
		return fmt.Errorf("invalid data for CodeOut and CodeOutProof, "+
			"CodeOut: %v, CodeOutProof: %v", msg.CodeOut, msg.CodeOutProof)
	}

	if msg.ShardInitOut == nil || msg.ShardInitOutProof == nil {
		return fmt.Errorf("invalid data for ShardInitOut and ShardInitOutProof, "+
			"ShardInitOut: %v, ShardInitOutProof: %v", msg.ShardInitOut, msg.ShardInitOutProof)
	}

	if msg.SmartContract == nil {
		return fmt.Errorf("invalid data for msg.SmartContract, SmartContract: %v", msg.SmartContract)
	}

	sc := SmartContract{msg.CodeOut.ContractAddress,
		msg.SmartContract.APIList, msg.SmartContract.Code}
	scHash := sc.SmartContractHash()
	scBytes := scHash.CloneBytes()

	if !bytes.Equal(scBytes, msg.CodeOut.Data) {
		return fmt.Errorf("the data of code out not equal to code hash\ncode hash:%v\ncode out data:%v",
			scBytes, msg.CodeOut.Data)
	}

	if err := msg.CodeOut.verifyOutState(msg.CodeOutProof, root); err != nil {
		return err
	}

	if err := msg.ShardInitOut.verifyOutState(msg.ShardInitOutProof, root); err != nil {
		return err
	}

	return nil
}

// SmartContractHash generates the Hash for the SmartContract.
func (sc *SmartContract) SmartContractHash() chainhash.Hash {
	return chainhash.HashH(sc.toBytesArray())
}

func (sc *SmartContract) toBytesArray() []byte {
	rtn, err := rlp.EncodeToBytes(sc)
	if err != nil {
		return nil
	}
	return rtn
}

// Equal check whether the two SmartContracts is equal.
func (sc *SmartContract) Equal(s *SmartContract) bool {
	return reflect.DeepEqual(sc, s)
}
