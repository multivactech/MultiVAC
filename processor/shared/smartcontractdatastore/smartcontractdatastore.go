package smartcontractdatastore

import (
	"fmt"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/processor/shared/state"
)

// SContractDataStore used to store smart-contract data
type SContractDataStore struct {
	shardIdx shard.Index
	// key is the public key hash of contract address, value is a map of shard smartContractInfo
	dataStore map[multivacaddress.PublicKeyHash]*wire.SmartContractInfo
}

// AddSmartContractInfo adds infos to data store
func (sCDS *SContractDataStore) AddSmartContractInfo(sCInfo *wire.SmartContractInfo, root *merkle.MerkleHash) error {
	if err := sCInfo.Verify(root); err != nil {
		return err
	}

	contractAddr := sCInfo.SmartContract.ContractAddr
	shardIdx := sCInfo.ShardIdx
	if shardIdx != sCDS.shardIdx {
		return fmt.Errorf("the shard index of smartContractInfo is wrong, want: %v, get: %v",
			sCDS.shardIdx, sCInfo.ShardIdx)
	}
	pKHash, err := contractAddr.GetPublicKeyHash(multivacaddress.SmartContractAddress)
	if err != nil {
		return err
	}
	sCDS.dataStore[*pKHash] = sCInfo

	return nil
}

// GetSmartContractInfo 返回指定智能合约指定分片编号的SmartContractInfo
func (sCDS *SContractDataStore) GetSmartContractInfo(pKHash multivacaddress.PublicKeyHash) *wire.SmartContractInfo {
	return sCDS.dataStore[pKHash]
}

// RefreshDataStore updates the smartContractDataStore according to the given StateUpdate and Actions.
func (sCDS *SContractDataStore) RefreshDataStore(update *state.Update, actions []*wire.UpdateAction) {
	// Update proof without update action
	if len(actions) == 0 {

		for addrHash, sCInfo := range sCDS.dataStore {
			proofs := []merkle.MerklePath{*sCInfo.CodeOutProof, *sCInfo.ShardInitOutProof}
			newProofs, err := update.UpdateProofs(proofs)
			if err != nil {
				log.Errorf("failed to update the proof of codeOut and shardInitOut, err: %v", err)
				delete(sCDS.dataStore, addrHash)
				continue
			}

			sCInfo.CodeOutProof = &newProofs[0]
			sCInfo.ShardInitOutProof = &newProofs[1]
			sCDS.dataStore[addrHash] = sCInfo

		}
		return
	}
	// Update proof with update action
	for _, action := range actions {
		pKHash, err := action.OriginOut.ContractAddress.GetPublicKeyHash(multivacaddress.SmartContractAddress)
		if err != nil {
			log.Errorf("smartContractDataStore failed to GetPublicKeyHash, err: %v", err)
			continue
		}
		if action.OriginOut.Shard != sCDS.shardIdx {
			log.Errorf("the shardIdx of smartContractDataStore and action.OriginOut.Shard are not equal")
			continue
		}

		if !action.OriginOut.IsSmartContractShardInitOut() {
			log.Errorf("the action.OriginOut is not smartContractShardInitOut")
			continue
		}

		sCInfo, ok := sCDS.dataStore[*pKHash]
		if ok {
			sCInfo.ShardInitOut = action.NewOut
		}

		newCodeProof, err := update.UpdateProofs([]merkle.MerklePath{*sCInfo.CodeOutProof})
		if err != nil {
			log.Errorf("fail to update proofs of code out, err: %v", err)
			delete(sCDS.dataStore, *pKHash)
			continue
		}
		sCInfo.CodeOutProof = &newCodeProof[0]
		newShardOutProof, err := update.GetShardDataOutProof(action.OriginOut, action.NewOut)
		if err != nil {
			log.Errorf("fail to update proofs of shard out, err: %v", err)
			delete(sCDS.dataStore, *pKHash)
			continue
		}
		sCInfo.ShardInitOutProof = newShardOutProof
		sCDS.dataStore[*pKHash] = sCInfo
	}
}

// Reset sets all smartContractInfo's shard data and code data in the smartContractDataStore into null info.
func (sCDS *SContractDataStore) Reset() {
	sCDS.dataStore = make(map[multivacaddress.PublicKeyHash]*wire.SmartContractInfo)
}
