/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package genesis

import (
	"encoding/hex"
	"fmt"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
)

const (
// it seems unused.
//pkToGen = 10
)

// Pool is the pk pool for the first batch of miners.
// All of the pks in pool will store in all miners' genesis block.
//
// Notion: Pool should be used by a scheduler, the scheduler should
// ensure that each miner gets a different pk.
type Pool struct {
	pks     []string // all pks from config file
	sks     []string // all sks corresponding to pk
	address map[string]string
	cur     int // index of current pk
}

// NewPool returns a new instance of Pool, it will read config file from
// the given path.
// TODO: should not hard coee.
func NewPool() *Pool {
	pkPool := new(Pool)
	pkPool.pks = make([]string, 0)
	pkPool.sks = make([]string, 0)
	pkPool.address = make(map[string]string)
	var pks []string
	var sks []string

	//pks = []string{
	//	"cb86132e3b230be51b6ffa52e9dfa1c66c9482353487b7cb6f24daaffa3ab618",
	//	"dceadf79ac2d6d120d8f67ef5a280a86bff8f69098932b61280b1957f969df1e",
	//	"292d57b6dbdcadb95eee26bd9beebe023d2ccb9126e40f73cc229e5ebde7a0dd",
	//	"a34b09d95b8d5693ba0e57be837a1045aa7c652b7be2f3457b99dcfac6e56955",
	//	"e55ec0980afae528a4f75c707b303a39edc3bdf5d77017fd91ab1844205e4188",
	//	"3ffaf068f40b504d7e9fd3d78115ce55ea36d190ab5c4bb4c8ca1bfa03208e56",
	//	"a52d8632ee6a43959eb278a012c1e4a458ab3086b18bc09df35d5692902209a2",
	//	"1349e012a7d58030c24c649e4dd339739d9ba8b4b6d9bc73007f3771b8e45700",
	//}

	//sks = []string{
	//	"ddb5e865ca8f6742dfa280c3741c9b1a6b58682e2cc6cdf729216de9890d71c0cb86132e3b230be51b6ffa52e9dfa1c66c9482353487b7cb6f24daaffa3ab618",
	//	"83a9f4d24189afee5215bcedc2b9818b06a1dd070d7dfd8d23e2009617faa508dceadf79ac2d6d120d8f67ef5a280a86bff8f69098932b61280b1957f969df1e",
	//	"d3d725b61ffe2a25d48761a1d413ea0415288530fcc2d8dd44b40075967e7ed1292d57b6dbdcadb95eee26bd9beebe023d2ccb9126e40f73cc229e5ebde7a0dd",
	//	"e1a20fc33783111f7b95b6afb54d3c6c4b642c812f31d1bc650d8bedfb09dd5ea34b09d95b8d5693ba0e57be837a1045aa7c652b7be2f3457b99dcfac6e56955",
	//	"f62a16429e3d19701fee058c8ac921f822f8ee7870f25f6e3ca4014fcdd7334ce55ec0980afae528a4f75c707b303a39edc3bdf5d77017fd91ab1844205e4188",
	//	"b1e06383717555b1d8fb9761617fc2918b649ab62b327c26824fc9900fe3c6293ffaf068f40b504d7e9fd3d78115ce55ea36d190ab5c4bb4c8ca1bfa03208e56",
	//	"93e605d0ae8920804da21d591c2f148c9588f8738e9a0b6976edf1624e8eca12a52d8632ee6a43959eb278a012c1e4a458ab3086b18bc09df35d5692902209a2",
	//	"7a16ded8b8f37faa69d6f655eefdd813b8a60464dd662d892747acda99aa284f1349e012a7d58030c24c649e4dd339739d9ba8b4b6d9bc73007f3771b8e45700",
	//}
	pks = TestNetToolsPubs
	sks = TestNetToolsPrvs

	pkPool.pks = append(pkPool.pks, pks...)
	pkPool.sks = append(pkPool.sks, sks...)

	for _, shard := range shard.ShardList {
		genesisPK := GenesisPublicKeys[shard]
		genesisSK := GenesisPrivateKeys[shard]
		genesisPKS := hex.EncodeToString(genesisPK)
		genesisSKS := hex.EncodeToString(genesisSK)
		pkPool.pks = append(pkPool.pks, genesisPKS)
		pkPool.sks = append(pkPool.sks, genesisSKS)
	}

	for i, pk := range pkPool.pks {
		address, err := multivacaddress.StringToUserAddress(pk)
		if err != nil {
			return nil
		}
		pkPool.address[address.String()] = pkPool.sks[i]
	}
	return pkPool
}

// Pop returns a new pk from the pool.
func (pool *Pool) Pop() (string, string, error) {
	if pool.IsEmpty() {
		return "", "", fmt.Errorf("no enough pk")
	}
	pool.cur++
	return pool.pks[pool.cur-1], pool.sks[pool.cur-1], nil
}

// Pks returns all publicKeys from the pool.
func (pool *Pool) Pks() []string {
	return pool.pks
}

// GetSkFromAddress returns the pk of the given address
func (pool *Pool) GetSkFromAddress(address multivacaddress.Address) ([]byte, error) {
	sks, ok := pool.address[address.String()]
	if !ok {
		return nil, fmt.Errorf("no pk in pool")
	}
	sk, err := hex.DecodeString(sks)
	if err != nil {
		return nil, err
	}
	return sk, nil
}

// IsEmpty retunrs if has pks to provide.
func (pool *Pool) IsEmpty() bool {
	return pool.cur > len(pool.pks)-1
}

// Reset set the cur pointer to the initial position.
func (pool *Pool) Reset() {
	pool.cur = 0
}
