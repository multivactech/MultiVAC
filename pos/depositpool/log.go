/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package depositpool

import (
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log btclog.Logger
var logBackend *btclog.Backend

func init() {
	logBackend = logger.BackendLogger()
	log = logBackend.Logger(logger.DpoolLoggerTag)
	log.SetLevel(logger.DpoolLogLevel)
}
