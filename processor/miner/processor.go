/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package miner

import (
	"fmt"

	"github.com/multivactech/MultiVAC/interface/iabci"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/iconsensus"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// Processor is used to manage the miner node.
type Processor struct {
	shard             shard.Index
	consensusExecutor iconsensus.Consensus
	abci              iabci.AppBlockChain
	blockChain        iblockchain.BlockChain
}

// StopConsensusAtRound means stop the consensus at the round.
func (p *Processor) StopConsensusAtRound(round int) {
	p.consensusExecutor.StopConsensusAtRound(round)
}

// SetStartConsensusAtRound means set the consensus start at the round.
func (p *Processor) SetStartConsensusAtRound(round int) {
	p.consensusExecutor.StartConsensusAtRound(round)
}

// Init the miner node processor.
func (p *Processor) Init() {
	p.consensusExecutor.Init()
}

// Start integration test mode.
func (p *Processor) Start(isIntegrationTest bool) {
	if isIntegrationTest {
		p.startIntegrationTestMode()
	}
}

// StartABCI starts abci and records the preReshardHeader.
func (p *Processor) StartABCI(prevReshardheader *wire.BlockHeader) {
	p.abci.Start(prevReshardheader)
}

func (p *Processor) PreStartABCI() {
	p.abci.PreStart()
}

// StopABCI stop to handleAbciMsg and fetch tx, change the stoping state of abci to true.
func (p *Processor) StopABCI() {
	p.abci.Stop()
}

// UpdateMinerProof is used to update miner's proof.
func (p *Processor) UpdateMinerProof(proof []byte) {
	p.consensusExecutor.UpdateInShardProof(proof)
}

// RegisterCEListener adds a listener to shardStateListeners.
func (p *Processor) RegisterCEListener(listener iconsensus.CEShardStateListener) {
	p.consensusExecutor.RegisterCEShardStateListener(listener)
}

// ReceiveSyncHeader receives the block head and use abci to handle it.
func (p *Processor) ReceiveSyncHeader(header *wire.BlockHeader) {
	// TODO(zz): move it to shardController
	// p.abci.onMessage(header)
}

// String returns the processor's id information as string.
func (p *Processor) String() string {
	return fmt.Sprintf("Miner/Processor [%v]", p.shard.GetID())
}

func (p *Processor) startIntegrationTestMode() {
	go p.startConsensusForIntegrationTest()
}

func (p *Processor) startConsensusForIntegrationTest() {
	p.consensusExecutor.Start()
}