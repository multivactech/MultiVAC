/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package consensus

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
	"github.com/multivactech/MultiVAC/model/wire"
)

type nextBAndByzValue struct {
	b byte
	v *wire.ByzAgreementValue
}

// newByzValueForTest generates a random b and a corresponding fake ByzValue.
func newByzValueForTest() (byte, *wire.ByzAgreementValue) {
	emptyValue := &wire.ByzAgreementValue{}

	b := (byte)(rand.Int() % 2)

	if b == 0 {
		hash := *new(chainhash.Hash)

		randBytes := make([]byte, 32)
		for i := range randBytes {
			randBytes[i] = byte(rand.Int() % 10)
		}

		if err := hash.SetBytes(randBytes); err != nil {
			return 0, nil
		}

		value := &wire.ByzAgreementValue{BlockHash: hash, Leader: ""}
		return b, value
	}
	return b, emptyValue
}

// test
func TestBinaryBAVoter_numOfFinNodesBeforeStep(t *testing.T) {
	// count all votes which have finished and step after 4
	voter := newBinaryBAVoter(0, 100, nil)
	voter.setCurrentRound(1)
	bCount := make(map[byte]int)
	for i := 0; i < 50; i++ {
		b, v := newByzValueForTest()
		bCount[b]++
		step := (rand.Int() % 4) + 4
		_, _, _ = voter.addVote(1, -step, b, v)
	}

	numOfFinNodesBeforeStep0 := voter.numOfFinNodesBeforeStep(1, 10, 0, nil)
	numOfFinNodesBeforeStep1 := voter.numOfFinNodesBeforeStep(1, 10, 1, nil)

	assert.Equal(t, numOfFinNodesBeforeStep0, bCount[0])
	assert.Equal(t, numOfFinNodesBeforeStep1, bCount[1])
	assert.Equal(t, numOfFinNodesBeforeStep0+numOfFinNodesBeforeStep1, 50)
}

func TestBinaryBAVoter_getBBAHistory(t *testing.T) {
	round, step, b := 1, 1, 1
	voter := newBinaryBAVoter(0, 100, nil)
	voter.setCurrentRound(round)
	bbaMap := make(map[*wire.MsgBinaryBA]struct{})
	for i := 0; i < 50; i++ {
		bba := &wire.MsgBinaryBA{
			SignedCredentialWithBA: &wire.SignedMsg{Pk: []byte{31: byte(i)}},
		}
		bbaMap[bba] = struct{}{}
		voter.addVoteWithBBAMsg(round, step, byte(b), &wire.ByzAgreementValue{}, bba)
	}

	bbaHistory := voter.getBBAHistory(round, step, byte(b))
	assert.Equal(t, len(bbaMap), len(bbaHistory))
	for _, bba := range bbaHistory {
		assert.Contains(t, bbaMap, bba)
	}
}

func TestBinaryBAVoter_addVoteWithBBAMsg(t *testing.T) {
	var threshold = 100
	round := 1
	step := 5

	voter := newBinaryBAVoter(0, 100, nil)

	for i := 0; i < 2*threshold; i++ {
		b, v := newByzValueForTest()
		msgbba := &wire.MsgBinaryBA{}
		msgbba.SignedCredentialWithBA = &wire.SignedMsg{}
		msgbba.SignedCredentialWithBA.Pk = make([]byte, 32)

		randBytes := make([]byte, 32)
		for i := range randBytes {
			randBytes[i] = byte(rand.Int() % 10)
		}

		copy(msgbba.SignedCredentialWithBA.Pk, randBytes)

		res, agreeType, _ := voter.addVoteWithBBAMsg(round, step, b, v, msgbba)
		if agreeType == notAgree && res != nil {
			t.Error("wrong")
		}
	}
}

func TestBinaryBAVoter_addVote(t *testing.T) {
	var threshold = 100
	voter := newBinaryBAVoter(0, 100, nil)

	v := &wire.ByzAgreementValue{}
	var b byte

	// if round not equal
	res, agreeType, _ := voter.addVote(1, 5, 0, v)
	if res != nil || agreeType != notAgree {
		t.Errorf("when currentRound != vote.round, it work wrong")
	}

	voter.setCurrentRound(1)

	// make a byzValuePool, so we can put it in addVote
	byzValuePool := make([]nextBAndByzValue, 50)
	for i := 0; i < threshold/2; i++ {
		b, v = newByzValueForTest()
		byzValuePool[i] = nextBAndByzValue{b, v}
	}
	for i := 0; i < threshold; i++ {
		b, v = byzValuePool[0].b, byzValuePool[0].v
		res, agreeType, _ := voter.addVote(1, 5, b, v)
		if res != nil {
			if agreeType == notAgree {
				t.Error("vote belong to same point, it wrong", agreeType)
			}
		}
	}
	binaryCount := make(map[byte]int)

	// 1/3 same point
	// 2/3 random point

	for i := 1; i < 2*threshold; i++ {

		idx := (rand.Int() % 49) + 1

		b, v = byzValuePool[idx].b, byzValuePool[idx].v

		binaryCount[b]++
		step := (rand.Int() % 4) + 4
		if res, agreeType, _ := voter.addVote(1, step, b, v); res != nil {
			switch agreeType {
			case notAgree:
				if res != nil {
					t.Error("notAgree, expect nil, recieve value")
				}
			default:
				if res == nil {
					t.Error("Agree, but res is empty")
				}
			}
			//if res.b == 0 {
			//	if !(binaryCount[0] >= threshold && binaryCount[1] < threshold && agreeType == totalAgreeOnZero) {
			//		//t.Errorf("get result b = 0, but count[0] = %v, count[1] = %v, agreeType = %v",
			//		//	binaryCount[0], binaryCount[1], agreeType)
			//	}
			//} else if res.b == 1 {
			//	if !(binaryCount[1] >= threshold && binaryCount[0] < threshold && agreeType == agreeOnOne) {
			//		//t.Errorf("get result b = 1, but count[0] = %v, count[1] = %v, agreeType = %v",
			//		//	binaryCount[0], binaryCount[1], agreeType)
			//	}
			//} else {
			//	t.Errorf("Wrong result b")
			//}
		}
	}
}

func TestBinaryBAVoter_GetVoteKeyWithHalfThreshold(t *testing.T) {
	threshold := 100
	voter := newBinaryBAVoter(1, threshold, nil)

	emptyValue := &wire.ByzAgreementValue{}
	validValue := &wire.ByzAgreementValue{BlockHash: *new(chainhash.Hash), Leader: "42"}

	if voteKey, byzValue := voter.getVoteKeyWithHalfThreshold(); voteKey != nil || byzValue != nil {
		t.Errorf("voteKey is empty")
	}

	binaryCount := make(map[byte]int)
	for {
		b := (byte)(rand.Int() % 2)
		var v *wire.ByzAgreementValue
		if b == 0 {
			v = validValue
		} else {
			v = emptyValue
		}
		binaryCount[b]++
		voter.addVote(1, 4, b, v)
		if voteKey, _ := voter.getVoteKeyWithHalfThreshold(); voteKey != nil {
			if binaryCount[voteKey.b] < (threshold+1)/2 {
				t.Errorf("Votekey didn't reach more than half of threshold")
			}
			break
		}
	}
}
