/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package signature

import (
	"io"

	"golang.org/x/crypto/ed25519"
)

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
	// SignatureSize is the size, in bytes, of signatures generated and verified by this package.
	SignatureSize = 64
)

// PrivateKey is a byte array, []byte.
type PrivateKey ed25519.PrivateKey

// PublicKey is a byte array, []byte.
type PublicKey ed25519.PublicKey

// Public returns the public key fo the private key.
func (priv PrivateKey) Public() PublicKey {
	privateKey := ed25519.PrivateKey(priv)
	publicKey := privateKey.Public()
	return PublicKey(publicKey.(ed25519.PublicKey))
}

// Sign is used to signature the transaction.
func Sign(privateKey PrivateKey, message []byte) []byte {
	return ed25519.Sign(ed25519.PrivateKey(privateKey), message)
}

// Verify is used to confirm whether the signature is right.
func Verify(publicKey PublicKey, message, sig []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, sig)
}

// GenerateKey generates a public/private key pair using entropy from rand.
// If rand is nil, crypto/rand.Reader will be used.
func GenerateKey(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	pubkey, privkey, err := ed25519.GenerateKey(rand)
	if err == nil {
		return PublicKey(pubkey), PrivateKey(privkey), nil
	}
	return nil, nil, err

}
