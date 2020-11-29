/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package chain

// SyncTrigger is a interface that contains a method called MayBeSync
// which notifies SyncManager to request a sync, whether or not a sync will actually happen depending on internal logic.
type SyncTrigger interface {
	MaybeSync()
}
