/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package merkle

func createRAMNodeMap() iNodeMap {
	nm := &ramNodeMap{}
	nm.nodeMap = make(map[MerkleHash]iMerkleNode)
	return nm
}

type ramNodeMap struct {
	nodeMap map[MerkleHash]iMerkleNode
}

func (nm *ramNodeMap) set(key MerkleHash, value iMerkleNode) {
	nm.nodeMap[key] = value
}

func (nm *ramNodeMap) get(key MerkleHash) (iMerkleNode, bool) {
	v, e := nm.nodeMap[key]
	return v, e
}

func (nm *ramNodeMap) del(key MerkleHash) {
	delete(nm.nodeMap, key)
}
