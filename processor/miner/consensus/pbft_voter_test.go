/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

/**
 * Copyright (c) 2018-present, MultiVAC dev team.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/multivactech/MultiVAC/base/vrf"
	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/stretchr/testify/assert"
)

var (
	testVRF           = vrf.Ed25519VRF{}
	testVoteThreshold = 2
)

func (v *vote) String() string {
	return fmt.Sprintf("{round: %d; signature: %s; validatorPeerId: %d; blockHash: %v}",
		v.round, v.voteSignature, v.votePublicKey, v.blockHash)
}

// Equal is used for judge whether the both Vote is equal to each other.
func (v *vote) Equal(vb *vote) bool {
	return bytes.Equal(v.voteSignature, vb.voteSignature) &&
		bytes.Equal(v.votePublicKey, vb.votePublicKey) &&
		v.blockHash == vb.blockHash &&
		v.seed == vb.seed &&
		v.round == vb.round
}

func TestPBFTVoter_GenerateVoteResult_PreVote_emptyVote(t *testing.T) {

	voter := newPbftVoter(0, leaderProposalVoteType, testVRF, testVoteThreshold)

	voteResult := voter.generateVoteResult()
	assert.Nilf(t, voteResult, "Expected nil vote result for empty vote set")
}

func TestPBFTVoter_GenerateVoteResult_PreVote(t *testing.T) {
	voter := newPbftVoter(0, leaderProposalVoteType, testVRF, testVoteThreshold)
	seed1, _ := chainhash.NewHashFromStr("a")
	seed2, _ := chainhash.NewHashFromStr("b")
	expectedVote := &vote{round: 0, votePublicKey: []byte("3"), voteSignature: []byte("def"), seed: *seed1}

	voter.addVote(&vote{round: 0, votePublicKey: []byte("1"), voteSignature: []byte("abc"), seed: *seed2})
	voter.addVote(&vote{round: 0, votePublicKey: []byte("2"), voteSignature: []byte("bcd"), seed: *seed2})
	voter.addVote(&vote{round: 0, votePublicKey: []byte("3"), voteSignature: []byte("def"), seed: *seed1})
	voter.addVote(&vote{round: 1, votePublicKey: []byte("4"), voteSignature: []byte("fgh"), seed: *seed1})

	voteResult := voter.generateVoteResult()
	assert.Truef(t, voteResult.Equal(expectedVote),
		"expected vote result %s, actual vote result %s", expectedVote.String(), voteResult.String())
}

func TestPBFTVoter_GenerateVoteResult_PreCommit(t *testing.T) {
	voter := newPbftVoter(0, leaderCommitVoteType, testVRF, testVoteThreshold)
	blockHash1, _ := chainhash.NewHashFromStr("a")
	blockHash2, _ := chainhash.NewHashFromStr("b")
	expectedVoteResult := &vote{round: 0, blockHash: *blockHash1}

	voter.addVote(&vote{round: 0, votePublicKey: []byte("1"), voteSignature: []byte("abc"), blockHash: *blockHash1})
	voter.addVote(&vote{round: 0, votePublicKey: []byte("2"), voteSignature: []byte("bcd"), blockHash: *blockHash2})

	voteResult := voter.generateVoteResult()
	assert.Nil(t, voteResult)

	voter.addVote(&vote{round: 0, votePublicKey: []byte("3"), voteSignature: []byte("bcd"), blockHash: *blockHash1})
	voteResult = voter.generateVoteResult()
	assert.Truef(t, voteResult.Equal(expectedVoteResult),
		"expected vote result %s, actual vote result %s", expectedVoteResult.String(), voteResult.String())
}
