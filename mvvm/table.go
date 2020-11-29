/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

const (
	// defaultTableSize shows the default size of table for a new MVVM.
	// The table is used for storing the "indirect call" function.
	// For now defaultTableSize is set as 65536.
	defaultTableSize = 1 << 16
)
