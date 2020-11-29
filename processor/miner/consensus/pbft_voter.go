/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"bytes"
	"sync"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/wire"
)

// voteType uint8
type voteType uint8

const (
	// leaderProposalVoteType means the vote is for LeaderProposal.
	leaderProposalVoteType = voteType(0x01)
	// leaderCommitVoteType means the vote is for LeaderCommit.
	leaderCommitVoteType = voteType(0x02)
	// minSeedKey is the minest seed key in map.
	minSeedKey = "MinSeed"
)

// vote contains round, VoteSig, VotePub, blockHash, seed, and leader. It's a description
// of the Byzantine fault-tolerant algorithm's ticket.
type vote struct {
	round         int
	voteSignature []byte
	votePublicKey []byte
	blockHash     chainhash.Hash
	seed          chainhash.Hash
	leader        string
}

// pbftVoter is a voter for Practical Byzantine Fault Tolerance.
type pbftVoter struct {
	mu            *sync.RWMutex
	round         int
	voteThreshold int
	voteType      voteType
	verifier      vrf.VRF
	preVotes      map[string]*vote
	preCommits    map[string]*vote
}

func newPbftVoter(round int, voteType voteType, verifier vrf.VRF, voteThreshold int) *pbftVoter {
	return &pbftVoter{
		mu:            new(sync.RWMutex),
		round:         round,
		voteType:      voteType,
		verifier:      verifier,
		preVotes:      make(map[string]*vote),
		preCommits:    make(map[string]*vote),
		voteThreshold: voteThreshold,
	}
}

func (voter *pbftVoter) setThreshold(threshold int) {
	voter.mu.Lock()
	defer voter.mu.Unlock()

	voter.voteThreshold = threshold
}

func (voter *pbftVoter) addVote(vote *vote) {
	if voter.voteType == leaderProposalVoteType {
		voter.addPreVote(vote)
	} else {
		voter.addPreCommit(vote)
	}
}

// This function should not be called externally, but only by AddVote
func (voter *pbftVoter) addPreVote(vote *vote) {
	voter.mu.Lock()
	defer voter.mu.Unlock()
	if _, existing := voter.preVotes[string(vote.votePublicKey)]; !existing {
		voter.preVotes[string(vote.votePublicKey)] = vote
		// save the smallest seed
		if minSeed, ok := voter.preVotes[minSeedKey]; ok {
			if bytes.Compare(vote.seed[0:], minSeed.seed[0:]) == -1 {
				voter.preVotes[minSeedKey] = vote
			}
		} else {
			voter.preVotes[minSeedKey] = vote
		}
	}
}

// This function should not be called externally, but only by AddVote
func (voter *pbftVoter) addPreCommit(vote *vote) {
	voter.mu.Lock()
	defer voter.mu.Unlock()
	if _, existing := voter.preCommits[string(vote.votePublicKey)]; !existing {
		voter.preCommits[string(vote.votePublicKey)] = vote
	}
}

func (voter *pbftVoter) generateVoteResult() *vote {
	if voter.voteType == leaderProposalVoteType {
		return voter.generatePreVoteResult()
	} else if voter.voteType == leaderCommitVoteType {
		return voter.generatePreCommitResult()
	} else {
		log.Infof("invalid vote type %d\n", voter.voteType)
		return nil
	}
}

// This function should not be called externally, but only by AddVote
func (voter *pbftVoter) generatePreVoteResult() *vote {
	voter.mu.RLock()
	defer voter.mu.RUnlock()
	var selectedCandidate *vote
	if minSeed, ok := voter.preVotes[minSeedKey]; ok {
		// log.Infof("Show received pre-votes %v", voter.preVotes)
		return minSeed
	}
	return selectedCandidate
}

// This function should not be called externally, but only by AddVote
func (voter *pbftVoter) generatePreCommitResult() *vote {
	voter.mu.RLock()
	defer voter.mu.RUnlock()
	counts := make(map[wire.ByzAgreementValue]int)
	var selectedVote *vote

	for _, v := range voter.preCommits {
		key := wire.ByzAgreementValue{
			BlockHash: v.blockHash,
			Leader:    v.leader,
		}
		counts[key]++
		if counts[key] >= voter.voteThreshold {
			selectedVote = &vote{round: voter.round, /* voteSignature: []byte(signature), */
				blockHash: v.blockHash, leader: v.leader}
			break
		}
	}
	log.Debugf("Show vote result count: %v", counts)
	return selectedVote
}

func (voter *pbftVoter) setCurrentRound(round int) {
	voter.mu.Lock()
	defer voter.mu.Unlock()
	voter.round = round
	voter.preCommits = make(map[string]*vote)
	voter.preVotes = make(map[string]*vote)
}
