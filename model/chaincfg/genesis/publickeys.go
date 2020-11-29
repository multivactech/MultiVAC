/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package genesis

import (
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/shard"
)

func publicKeys() map[shard.Index]signature.PublicKey {
	keys := make(map[shard.Index]signature.PublicKey)
	for _, shard := range shard.ShardList {
		keys[shard] = GenesisPrivateKeys[shard].Public()
	}
	return keys
}

// GenesisPublicKeys contains all genesis public keys
var GenesisPublicKeys = publicKeys()
