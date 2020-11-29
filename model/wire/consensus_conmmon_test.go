package wire

import (
	"encoding/hex"
	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testLeaderPublicKeyA, _  = hex.DecodeString("885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
	testLeaderPrivateKeyA, _ = hex.DecodeString("1fcce948db9fc312902d49745249cfd287de1a764fd48afb3cd0bdd0a8d74674885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2")
)

func TestConsensusMsg_sign(t *testing.T) {
	var signMsg = &SignedMsg{
		Pk: testLeaderPublicKeyA,
	}
	var cred = &Credential{
		Round: 0,
		Step:  1,
	}

	err := signMsg.sign(cred, testLeaderPrivateKeyA)

	assert.Nil(t, err)
	buf, err := rlp.EncodeToBytes(cred)
	assert.Nil(t, err)
	ok, err := ed25519VRF.VerifyProof(testLeaderPublicKeyA, signMsg.Signature, buf)
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestConsensusMsg_isValidSignedCredential(t *testing.T) {
	var cred = &Credential{
		Round: 0,
		Step:  1,
	}
	var signMsg = &SignedMsg{
		Pk:      testLeaderPublicKeyA,
		Message: ConsensusMsg{Credential: cred},
	}
	err := signMsg.sign(cred, testLeaderPrivateKeyA)
	assert.Nil(t, err)

	err = isValidSignedCredential(signMsg.Message.Credential, signMsg)

	assert.Nil(t, err)
}
