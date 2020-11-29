//  Copyright (c) 2018-present, MultiVAC Foundation.
//  This source code is licensed under the MIT license found in the
//  LICENSE file in the root directory of this source tree.
//  Copyright 2017 Yahoo Inc. All rights reserved.

package vrf

// Life of a verification:
//
// pk, sk, _ = GenerateKey(nil)
// proof, _ = Generate(pk, sk, msg)
//   Make sure the proof is indeed the verification:
// verified bool = Verify(pk, proof, msg)
//   To generate the hash:
// hash = ProofToHash(proof)
//
// Please notice that ProofToHash only uses proof. To make sure the hash is for the proof, just check
// ProofToHash(proof) == hash.
//
// Play with ed25519_vrf_test.go for a more concrete example.

// This VRF is implemented according to IETF ECVRF:
// https://tools.ietf.org/id/draft-irtf-cfrg-vrf-02.html
//
// Package vrf implements a verifiable random function using the Edwards form
// of Curve25519, SHA512 and the Elligator map.
//
// A few fixed terms:
// F - finite field
// 2n - length, in octets, or a field element in F
// E - elliptic curve(EC) defined over F
// m - length, in octets, of an EC point encoded as an octet string
// G - subgroup of E of large prime order
// q - prime order of group G; must be less than 2^(2n)
// cofactor - number of points on E divided by q
// g - generator of group G
// Hash - cryptographic hash function
// hLen - output length in octets of Hash; must be at least 2n
// suite - a single nonzero octet specifying the ECVRF ciphersuite, which determines
// the above options

// A few other terms:
// p^k - When p is an EC point: point multiplication, i.e. k repetition of the EC group operation
// applied to the EC point p. When p is an integer: exponention
// || - octet string concatenation
// I2OSP - conversion of nonnegative integer conversion to octet string
//         I = integer 2 = to OS = octet string
// EC2OSP - conversion of EC point to an m-octet string
// OS2ECP - OS -> EC
// RS2ECP - a random 2n-octet string -> EC point

// Parameters used:
// sk = Private key
// x: VRF secret scalar, an integer
//   TODO(jj5): secret key does not have to be the same as this scalar. It  can be just a derivation.
//              The question is, which do we want?
// y = g^x, an EC point. Also used as pk.

import (
	"encoding/hex"
	"io"

	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	ed "github.com/multivactech/MultiVAC/base/vrf/edwards25519"
	"math/big"
)

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
)

const (
	// N2 is a necessary parameter for ed25519 algorithm
	N2 = 32 // ceil(log2(qs) / 8)
	// N is a necessary parameter for ed25519 algorithm
	N        = N2 / 2
	qs       = "1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed" // 2^252 + 27742317777372353535851937790883648493
	cofactor = 8
	NOSIGN   = 3
)

var (
	// ErrMalformedInput returns the error of malformed input.
	ErrMalformedInput = errors.New("ECVRF: malformed input")
	// ErrDecodeError returns the error of failed to decode proof.
	ErrDecodeError = errors.New("ECVRF: decode error")
	q, _           = new(big.Int).SetString(qs, 16)
	g              = G()
)

type Ed25519VRF struct{}

// GenerateKey creates a public/private key pair. rnd is used for randomness.
// If it is nil, `crypto/rand` is used.
func (vrf Ed25519VRF) GenerateKey(rnd io.Reader) (pk PublicKey, sk PrivateKey, err error) {
	if rnd == nil {
		rnd = rand.Reader
	}
	sk = make([]byte, PrivateKeySize)
	_, err = io.ReadFull(rnd, sk[:32])
	if err != nil {
		return nil, nil, err
	}
	x := expandPrivateKey(sk)

	var pkP ed.ExtendedGroupElement
	ed.GeScalarMultBase(&pkP, x)
	var pkBytes [PublicKeySize]byte
	pkP.ToBytes(&pkBytes)

	copy(sk[32:], pkBytes[:])
	return pkBytes[:], sk, err
}

func hashToCurve(m []byte, pk []byte) *ed.ExtendedGroupElement {
	hash := sha256.New()
	for i := int64(0); i < 100; i++ {
		ctr := I2OSP(big.NewInt(i), 4)
		hash.Write(m)
		hash.Write(pk)
		hash.Write(ctr)
		h := hash.Sum(nil)
		hash.Reset()
		if P := OS2ECP(h, NOSIGN); P != nil {
			// assume cofactor is 2^n
			for j := 1; j < cofactor; j *= 2 {
				P = GeDouble(P)
			}
			return P
		}
	}
	panic("hashToCurve: couldn't make a point on curve")
}

// Generate proof of given msg.
//
// PublicKey and PrivateKey must be generated from EC25519, i.e. ed25519.GenerateKey()
//
// Steps:
// 1. h = hash_to_curve(suite, y, seed)
// 2. h1 = EC2OSP(h)
// 3. gamma = h^x
// 4. k = nonce_generation(sk, h1)
// 5. c = hash_points(h, gamma, g^k, h^k)
// 6. s = (k + c * x) mod q (* = integer multiplication)
// 7. output = EC2OSP(gamma) || I2OSP(c, n) || I2OSP(s, 2n)
func (vrf Ed25519VRF) Generate(pk PublicKey, sk PrivateKey, msg []byte) (proof []byte, err error) {
	x := expandPrivateKey(sk)
	h := hashToCurve(msg, pk)
	r := ECP2OS(GeScalarMult(h, x))

	kp, ks, err := vrf.GenerateKey(nil) // use GenerateKey to generate a random
	if err != nil {
		return nil, err
	}
	k := expandPrivateKey(ks)

	// ECVRF_hash_points(g, h, g^x, h^x, g^k, h^k)
	c := hashPoints(ECP2OS(g), ECP2OS(h), S2OS(pk), r, S2OS(kp), ECP2OS(GeScalarMult(h, k)))

	// s = k - c*x mod q
	var z big.Int
	s := z.Mod(z.Sub(F2IP(k), z.Mul(c, F2IP(x))), q)

	// pi = gamma || I2OSP(c, N) || I2OSP(s, 2N)
	var buf bytes.Buffer
	buf.Write(r) // 2N
	buf.Write(I2OSP(c, N))
	buf.Write(I2OSP(s, N2))
	return buf.Bytes(), nil
}

//GeneratePkWithSk generates KeyPair With sk(string)
func GeneratePkWithSk(sk string) (PublicKey, PrivateKey, error) {
	ks, err := hex.DecodeString(sk)
	if err != nil {
		return nil, nil, err
	}
	x := expandPrivateKey(ks)

	var pkP ed.ExtendedGroupElement
	ed.GeScalarMultBase(&pkP, x)
	var pkBytes [PublicKeySize]byte
	pkP.ToBytes(&pkBytes)

	copy(ks[32:], pkBytes[:])
	return PublicKey(pkBytes[:]), PrivateKey(ks), nil
}

func hashPoints(ps ...[]byte) *big.Int {
	h := sha256.New()
	for _, p := range ps {
		h.Write(p)
	}
	v := h.Sum(nil)
	return OS2IP(v[:N])
}

// ProofToHash generates hash from given proof.
//
// Steps:
// 1. (gamma, c, s) = decode_proof(pi)
// 2. three = 0x03 = I2OSP(3, 1)
// 3. preBeta = Hash(suite || three || EC2OSP(gamma ^cofactor))
// 4. output = first 2n octets of preBeta
func (vrf Ed25519VRF) ProofToHash(proof []byte) []byte {
	return proof[1 : N2+1]
}

// VerifyProof verifies the proof given seed.
//
// Steps:
// 1. (gamma, c, s) = decode(pi)
// 2. u = g^s / y^c (where / denotes EC point subtraction, i.e. the group operation
//    applied to g^s and the inverse of y^c
// 3. h = hash_to_curve(suite, y, seed)
// 4. v = h^s * gamma^c
// 5. c'= hash_points(h, gamma, u, v)
// 6. output = (c == c')
func (vrf Ed25519VRF) VerifyProof(pk PublicKey, proof []byte, msg []byte) (bool, error) {
	gamma, c, s, err := DecodeProof(proof)
	if err != nil {
		return false, err
	}

	// u = (g^x)^c * g^s = P^c * g^s
	var u ed.ProjectiveGroupElement
	P := OS2ECP(pk, pk[31]>>7)
	if P == nil {
		return false, ErrMalformedInput
	}
	ed.GeDoubleScalarMultVartime(&u, c, P, s)

	h := hashToCurve(msg, pk)

	// v = gamma^c * h^s
	v := GeAdd(GeScalarMult(gamma, c), GeScalarMult(h, s))

	// c' = hashPoints(g, h, g^x, gamma, u, v)
	c2 := hashPoints(ECP2OS(g), ECP2OS(h), S2OS(pk), ECP2OS(gamma), ECP2OSProj(&u), ECP2OS(v))

	return c2.Cmp(F2IP(c)) == 0, nil
}

// VRF returns the proof and its hash.
func (vrf Ed25519VRF) VRF(pk PublicKey, sk PrivateKey, msg []byte) (hash []byte, proof []byte, err error) {
	proof, err = vrf.Generate(pk, sk, msg)
	if err != nil {
		return nil, nil, err
	}
	hash = vrf.ProofToHash(proof)
	return
}

// DecodeProof decode from bytes to VRF proof.
func DecodeProof(pi []byte) (r *ed.ExtendedGroupElement, c *[N2]byte, s *[N2]byte, err error) {
	i := 0
	sign := pi[i]
	i++
	if sign != 2 && sign != 3 {
		return nil, nil, nil, ErrDecodeError
	}
	r = OS2ECP(pi[i:i+N2], sign-2)
	i += N2
	if r == nil {
		return nil, nil, nil, ErrDecodeError
	}

	// swap and expand to make it a field
	c = new([N2]byte)
	for j := N - 1; j >= 0; j-- {
		c[j] = pi[i]
		i++
	}

	// swap to make it a field
	s = new([N2]byte)
	for j := N2 - 1; j >= 0; j-- {
		s[j] = pi[i]
		i++
	}
	return
}

func OS2ECP(os []byte, sign byte) *ed.ExtendedGroupElement {
	P := new(ed.ExtendedGroupElement)
	var buf [32]byte
	copy(buf[:], os)
	if sign == 0 || sign == 1 {
		buf[31] = (sign << 7) | (buf[31] & 0x7f)
	}
	if !P.FromBytes(&buf) {
		return nil
	}
	return P
}

// S2OS just prepend the sign octet.
func S2OS(s []byte) []byte {
	sign := s[31] >> 7     // @@ we should clear the sign bit??
	os := []byte{sign + 2} // Y = 0x02 if positive or 0x03 if negative
	os = append([]byte(os), s...)
	return os
}

func ECP2OS(P *ed.ExtendedGroupElement) []byte {
	var s [32]byte
	P.ToBytes(&s)
	return S2OS(s[:])
}

func ECP2OSProj(P *ed.ProjectiveGroupElement) []byte {
	var s [32]byte
	P.ToBytes(&s)
	return S2OS(s[:])
}

func I2OSP(b *big.Int, n int) []byte {
	os := b.Bytes()
	if n > len(os) {
		var buf bytes.Buffer
		buf.Write(make([]byte, n-len(os))) // prepend 0s
		buf.Write(os)
		return buf.Bytes()
	}
	return os[:n]

}

func OS2IP(os []byte) *big.Int {
	return new(big.Int).SetBytes(os)
}

// F2IP converts a field number (in LittleEndian) to a big int.
func F2IP(f *[32]byte) *big.Int {
	var t [32]byte
	for i := 0; i < 32; i++ {
		t[32-i-1] = f[i]
	}
	return OS2IP(t[:])
}

// IP2F converts a big int to a field number.
func IP2F(b *big.Int) *[32]byte {
	os := b.Bytes()
	r := new([32]byte)
	j := len(os) - 1
	for i := 0; i < 32 && j >= 0; i++ {
		r[i] = os[j]
		j--
	}
	return r
}

func GeScalarMult(h *ed.ExtendedGroupElement, a *[32]byte) *ed.ExtendedGroupElement {
	r := new(ed.ExtendedGroupElement)
	var pg ed.ProjectiveGroupElement
	ed.GeDoubleScalarMultVartime(&pg, a, h, &[32]byte{}) // h^a * g^0
	var t [32]byte
	pg.ToBytes(&t)
	r.FromBytes(&t)
	return r
}

func GeDouble(p *ed.ExtendedGroupElement) *ed.ExtendedGroupElement {
	var q ed.ProjectiveGroupElement
	p.ToProjective(&q)
	var rc ed.CompletedGroupElement
	q.Double(&rc)
	r := new(ed.ExtendedGroupElement)
	rc.ToExtended(r)
	return r
}

func G() *ed.ExtendedGroupElement {
	g := new(ed.ExtendedGroupElement)
	var f ed.FieldElement
	ed.FeOne(&f)
	var s [32]byte
	ed.FeToBytes(&s, &f)
	ed.GeScalarMultBase(g, &s) // g = g^1
	return g
}

func expandPrivateKey(sk []byte) *[32]byte {
	// copied from golang.org/x/crypto/ed25519/ed25519.go -- has to be the same
	digest := sha512.Sum512(sk[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64
	h := new([32]byte)
	copy(h[:], digest[:])
	return h
}

// CachedGroupElement is copied from ed.go and const.go in golang.org/x/crypto/ed25519/internal/ed
type CachedGroupElement struct {
	yPlusX, yMinusX, Z, T2d ed.FieldElement
}

// d2 is 2*d.
var d2 = ed.FieldElement{
	-21827239, -5839606, -30745221, 13898782, 229458, 15978800, -12551817, -6495438, 29715968, 9444199,
}

func ToCached(r *CachedGroupElement, p *ed.ExtendedGroupElement) {
	ed.FeAdd(&r.yPlusX, &p.Y, &p.X)
	ed.FeSub(&r.yMinusX, &p.Y, &p.X)
	ed.FeCopy(&r.Z, &p.Z)
	ed.FeMul(&r.T2d, &p.T, &d2)
}

func GeAdd(p, qe *ed.ExtendedGroupElement) *ed.ExtendedGroupElement {
	var q CachedGroupElement
	var r ed.CompletedGroupElement
	var t0 ed.FieldElement

	ToCached(&q, qe)

	ed.FeAdd(&r.X, &p.Y, &p.X)
	ed.FeSub(&r.Y, &p.Y, &p.X)
	ed.FeMul(&r.Z, &r.X, &q.yPlusX)
	ed.FeMul(&r.Y, &r.Y, &q.yMinusX)
	ed.FeMul(&r.T, &q.T2d, &p.T)
	ed.FeMul(&r.X, &p.Z, &q.Z)
	ed.FeAdd(&t0, &r.X, &r.X)
	ed.FeSub(&r.X, &r.Z, &r.Y)
	ed.FeAdd(&r.Y, &r.Z, &r.Y)
	ed.FeAdd(&r.Z, &t0, &r.T)
	ed.FeSub(&r.T, &t0, &r.T)

	re := new(ed.ExtendedGroupElement)
	r.ToExtended(re)
	return re
}
