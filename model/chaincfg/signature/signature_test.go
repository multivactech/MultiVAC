/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package signature

import (
	"crypto/sha256"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	message := []byte("Heatwave: Supermarket sleepover for cool customers")
	randPubKeyBytes := sha256.Sum256([]byte("blahblah"))
	randPubKey := PublicKey(randPubKeyBytes[:])
	publicKey, privateKey, err := GenerateKey(nil)
	if err != nil {
		t.Error(err.Error())
	}
	sig := Sign(privateKey, message)

	if !Verify(publicKey, message, sig) {
		t.Error("Fail to verify message and sig")
	}

	if Verify(randPubKey, message, sig) {
		t.Error("Verified messaged by wrong pub key")
	}
}
