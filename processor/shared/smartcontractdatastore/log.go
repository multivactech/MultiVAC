/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package smartcontractdatastore

import (
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

var log btclog.Logger

func init() {
	var exists bool
	log, exists = logger.GetLogger(logger.SmartContractDataStoreTag)
	if !exists {
		panic("Fail to get logger for smartContractDataStore package")
	}
}
