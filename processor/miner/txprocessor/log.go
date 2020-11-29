/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package txprocessor

import (
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

var log btclog.Logger

func init() {
	var exists bool
	log, exists = logger.GetLogger(logger.TxProcessorTag)
	if !exists {
		panic("Fail to get logger for txprocessor package")
	}
}
