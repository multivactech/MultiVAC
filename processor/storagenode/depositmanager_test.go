/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"testing"

	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

func TestRemoveOuts(t *testing.T) {
	shard0 := shard.IDToShardIndex(0)
	dm := newDepositManager()
	dm.removeOutsBeforeHeight(shard0, wire.BlockHeight(2))

	out1 := &wire.OutState{
		OutPoint: wire.OutPoint{
			Shard: shard0,
		},
	}

	out2 := &wire.OutState{
		OutPoint: wire.OutPoint{
			Shard: shard0,
		},
	}

	dm.add(out1, 2)
	dm.add(out2, 2)
	dm.removeOutsBeforeHeight(shard0, 2)
}
