/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
package smartcontractdatastore

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

var (
	shard0   = shard.IDToShardIndex(0)
	fakeTree = merkle.NewFullMerkleTree()
)

func TestSmartContractDataStore_AddSmartContractInfo(t *testing.T) {
	shardIdx := shard0
	fakeTree = generateOutsMerkleTree(shardIdx)

	scds := NewSmartContractDataStore(shard0)
	var scInfo wire.SmartContractInfo

	tx := &wire.MsgTx{
		Shard:   shardIdx,
		Version: wire.TxVersion,
		TxIn:    []*wire.TxIn{},
		ContractAddress: multivacaddress.GenerateAddress(signature.PublicKey([]byte{}),
			multivacaddress.SystemContractAddress),
		API:             "deploy",
		Params:          []byte{},
		SignatureScript: []byte{},
	}
	txHash := tx.TxHash()

	sc := wire.SmartContract{}

	sCAddr := txHash.FormatSmartContractAddress()
	sc.ContractAddr = sCAddr
	sc.APIList = []string{}
	sc.Code = []byte{}
	scHash := sc.SmartContractHash()
	scBytes := scHash.CloneBytes()

	codeOut := &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          txHash,
			Index:           0,
			Shard:           shardIdx,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shardIdx], multivacaddress.UserAddress),
			Data:            scBytes,
			ContractAddress: sc.ContractAddr,
		},
		State: wire.StateUnused,
	}
	scInfo.SmartContract = &sc
	scInfo.SmartContract.Code = []byte{}

	codeOutHash := merkle.ComputeMerkleHash(codeOut.ToBytesArray())
	codeOutProof, err := fakeTree.GetMerklePath(codeOutHash)
	if err != nil {
		t.Errorf("fail to get the merkle path of code out, err: %v", err)
	}

	scInfo.CodeOut = codeOut
	scInfo.CodeOutProof = codeOutProof

	shardOut := &wire.OutState{
		OutPoint: wire.OutPoint{
			TxHash:          txHash,
			Index:           1,
			Shard:           shardIdx,
			UserAddress:     multivacaddress.GenerateAddress(GenesisPublicKeys[shardIdx], multivacaddress.UserAddress),
			ContractAddress: sc.ContractAddr,
			Data:            []byte{},
		},
		State: wire.StateUnused,
	}
	shardOutHash := merkle.ComputeMerkleHash(shardOut.ToBytesArray())
	shardOutProof, err := fakeTree.GetMerklePath(shardOutHash)

	if err != nil {
		t.Errorf("fail to get the merkle path of shard out,err: %v", err)
	}
	scInfo.SmartContract.ContractAddr = sc.ContractAddr
	scInfo.ShardIdx = shardIdx
	scInfo.ShardInitOut = shardOut
	scInfo.ShardInitOutProof = shardOutProof
	scInfo.SmartContract.APIList = []string{}
	root := fakeTree.MerkleRoot

	if err := scds.AddSmartContractInfo(&scInfo, root); err != nil {
		t.Error("fail to add smart contract info to smart contract data store")
	}

	if len(scds.dataStore) == 0 {
		t.Errorf("fail to get data store")
	}
	publicKeyHash, err := sCAddr.GetPublicKeyHash(multivacaddress.SmartContractAddress)
	if err != nil {
		panic(err)
	}
	getSmartInfo := scds.GetSmartContractInfo(*publicKeyHash)

	if getSmartInfo == nil {
		t.Errorf("fail to get smart contract info from data store")
	}

	if !reflect.DeepEqual(getSmartInfo.SmartContract.APIList, []string{}) {
		t.Errorf("fail to get api list from smart contract date store")
	}

	if !bytes.Equal(getSmartInfo.SmartContract.Code, []byte{}) {
		t.Errorf("fail to get code from smart contract date store")
	}

	if !reflect.DeepEqual(getSmartInfo.CodeOutProof, codeOutProof) {
		t.Errorf("fail to get code out proof from smart contract date store")
	}

	if !reflect.DeepEqual(getSmartInfo.ShardInitOut, shardOut) {
		t.Errorf("fail to get shard data out from smart contract date store\nwant: %v,\nget: %v", shardOut, getSmartInfo.ShardInitOut)
	}

	if !reflect.DeepEqual(getSmartInfo.ShardInitOutProof, shardOutProof) {
		t.Errorf("fail to get shard data out proof from smart contract date store\nwant: %v,\nget: %v", shardOutProof, getSmartInfo.ShardInitOutProof)
	}
}
