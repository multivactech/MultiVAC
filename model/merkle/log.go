/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package merkle

import (
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

var log btclog.Logger

func init() {
	var exists bool
	log, exists = logger.GetLogger(logger.AddrmgrLoggerTag)
	if !exists {
		panic("Fail to get logger for merkle package")
	}
}
