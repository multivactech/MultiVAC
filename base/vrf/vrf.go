// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package vrf

import "io"

// PublicKey defines to be public key data structure of MultiVAC account.
type PublicKey []byte

// PrivateKey defines to be private key data structure of MultiVAC account.
type PrivateKey []byte

// VRF is named Verifiable Random Function.
// See https://en.wikipedia.org/wiki//Verifable_random_function for more info.
//
// This class is overall in charge of all functionalities related to VRF, including generating RNG, verification, etc.
//
// VRF is designed to be a stateless oracle(like how it was originally designed). Caller is responsible for
// rendering the private key.
type VRF interface {
	// GenerateKey creates a public/private key pair. rnd is used for randomness.
	//
	// If rnd is nil, 'crypto/rand' is used.
	GenerateKey(rnd io.Reader) (pk PublicKey, sk PrivateKey, err error)

	// Generates a proof with given private key and public key.
	//
	// Note: the public key is not necessary. Just for efficiency.
	Generate(pk PublicKey, sk PrivateKey, msg []byte) (proof []byte, err error)

	// Verifies the proof is indeed for given message.
	//
	// Notice that the PublicKey belongs to the broadcaster of the RNG, not the node itself.
	VerifyProof(publicKey PublicKey, pf []byte, msg []byte) (res bool, err error)

	// Renders hash from proof.
	ProofToHash(pf []byte) []byte

	// TODO Clean up the API and update test for this.
	// Generates hash and its proof
	VRF(pk PublicKey, sk PrivateKey, msg []byte) (hash []byte, proof []byte, err error)
}
