/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package storagenode

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

type outAndHeight struct {
	out    *wire.OutState
	height wire.BlockHeight
}

// depositManager is a helper object, it is used to record the update information
// of the deposit proof.
type depositManager struct {
	outs           map[shard.Index][]*outAndHeight // all deposit outs of a shard
	pendingHeights []wire.BlockHeight              // Height queue of blocks waiting to be processed
}

func newDepositManager() *depositManager {
	dm := &depositManager{
		outs:           make(map[shard.Index][]*outAndHeight),
		pendingHeights: make([]wire.BlockHeight, 0, 0),
	}
	return dm
}

// shouldProcess returns whether the current deposit processing on the block is possible.
func (dm *depositManager) shouldProcess() bool {
	return len(dm.pendingHeights) >= 1
}

// head returns the head height of the pending queue.
func (dm *depositManager) head() wire.BlockHeight {
	return dm.pendingHeights[0]
}

// deQueue removes the head height of the pending queue.
func (dm *depositManager) deQueue() {
	dm.pendingHeights = dm.pendingHeights[1:]
}

// enQueue adds a new pending height to queue.
func (dm *depositManager) enQueue(height wire.BlockHeight) {
	dm.pendingHeights = append(dm.pendingHeights, height)
}

// add adds a deposit outs to manager.
func (dm *depositManager) add(out *wire.OutState, height wire.BlockHeight) {
	oah := &outAndHeight{
		out:    out,
		height: height,
	}
	dm.outs[out.Shard] = append(dm.outs[out.Shard], oah)
}

// removeOutsBeforeHeight remove out before specified height.
func (dm *depositManager) removeOutsBeforeHeight(shard shard.Index, height wire.BlockHeight) []*wire.OutState {
	var results []*wire.OutState
	for i, outWithHeight := range dm.outs[shard] {
		if outWithHeight.height <= height {
			results = append(results, outWithHeight.out)
			if i < len(dm.outs[shard])-1 {
				dm.outs[shard] = append(dm.outs[shard][:i], dm.outs[shard][i+1:]...)
			} else {
				dm.outs[shard] = dm.outs[shard][:i]
			}
		}
	}
	return results
}
