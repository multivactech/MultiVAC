/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import "testing"

func fakeConfig(storageNode bool) *Config {
	return &Config{StorageNode: storageNode}
}

func TestGetNodeType(t *testing.T) {
	tests := []struct {
		desc             string
		storageNode      bool
		expectedNodeType NodeType
	}{
		{
			desc:             "Storage node",
			storageNode:      true,
			expectedNodeType: StorageNode,
		},
		{
			desc:             "Miner node",
			storageNode:      false,
			expectedNodeType: MinerNode,
		},
	}
	for _, test := range tests {
		globalCfg = fakeConfig(test.storageNode)
		if tt := GetNodeType(); tt != test.expectedNodeType {
			t.Errorf("Running test %v failed, expected: %v, get: %v", test.desc, test.expectedNodeType, tt)
		}
	}
}
