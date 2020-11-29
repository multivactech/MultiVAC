/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * Copyright 2017 Yahoo Inc. All rights reserved.
 */

package vrf

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	ed "github.com/multivactech/MultiVAC/base/vrf/edwards25519"
)

const message = "message"

func DoTestECVRF(t *testing.T, vrf VRF, pk, sk []byte, msg []byte, verbose bool) {
	pi, err := vrf.Generate(pk, sk, msg[:])
	if err != nil {
		t.Fatal(err)
	}
	res, err := vrf.VerifyProof(pk, pi, msg[:])
	if err != nil {
		t.Fatal(err)
	}
	if !res {
		t.Errorf("VRF failed")
	}

	// when everything get through
	if verbose {
		fmt.Printf("alpha: %s\n", hex.EncodeToString(msg))
		fmt.Printf("sk: %s\n", hex.EncodeToString(sk))
		fmt.Printf("pk: %s\n", hex.EncodeToString(pk))
		fmt.Printf("proof: %s\n", hex.EncodeToString(pi))
		fmt.Printf("hash: %s\n", hex.EncodeToString(vrf.ProofToHash(pi)))
		fmt.Println("the length of proof is of size ", len(pi), " bytes")

		r, c, s, err := DecodeProof(pi)
		if err != nil {
			t.Fatal(err)
		}
		// u = (g^x)^c * g^s = P^c * g^s
		var u ed.ProjectiveGroupElement
		P := OS2ECP(pk, pk[31]>>7)
		ed.GeDoubleScalarMultVartime(&u, c, P, s)
		fmt.Printf("r: %s\n", hex.EncodeToString(ECP2OS(r)))
		fmt.Printf("c: %s\n", hex.EncodeToString(c[:]))
		fmt.Printf("s: %s\n", hex.EncodeToString(s[:]))
		fmt.Printf("u: %s\n", hex.EncodeToString(ECP2OSProj(&u)))
	}
}

const howMany = 1000

func TestECVRF(t *testing.T) {
	vrf := Ed25519VRF{}
	for i := howMany; i > 0; i-- {
		pk, sk, err := vrf.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		var msg [32]byte
		io.ReadFull(rand.Reader, msg[:])
		DoTestECVRF(t, vrf, pk, sk, msg[:], false)
	}
}

const pks = "885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"
const sks = "1fcce948db9fc312902d49745249cfd287de1a764fd48afb3cd0bdd0a8d74674885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"

// old keys -- must fail
//const sks = "c4d50101fc48c65ea496105e5f0a43a67636972d0186edfb9445a2d3377e8b9c8e6fb0fd096402527e7c2375255093975324751f99ef0b7db014eb7e311befb5"
//const pks = "8e6fb0fd096402527e7c2375255093975324751f99ef0b7db014eb7e311befb5"

func TestECVRFOnce(t *testing.T) {
	vrf := Ed25519VRF{}

	pk, _ := hex.DecodeString(pks)
	sk, _ := hex.DecodeString(sks)
	m := []byte(message)
	DoTestECVRF(t, vrf, pk, sk, m, true)

	h := hashToCurve(m, pk)
	fmt.Printf("h: %s\n", hex.EncodeToString(ECP2OS(h)))
}

func TestHashToCurve(t *testing.T) {
	var m [32]byte
	pk, _ := hex.DecodeString(pks)
	for i := 0; i < 1000; i++ {
		io.ReadFull(rand.Reader, m[:])
		P := hashToCurve(m[:], pk)
		// test P on curve by P^order = infinity
		var infs [32]byte
		inf := GeScalarMult(P, IP2F(q))
		inf.ToBytes(&infs)
		if infs != [32]byte{1} {
			t.Fatalf("OS2ECP: not valid curve")
		}
	}
}

//jylu
func TestUsage(t *testing.T) {
	vrf := Ed25519VRF{}
	pk, sk, err := vrf.GenerateKey(bytes.NewReader([]byte("995f699c8390293eb74d08cf38d3300771e9e319cfd12a21429eeff2eddeebd2")))
	if err != nil {
		t.Error("cannot generate pk sk from a string")
	}
	proof, err := vrf.Generate(pk, sk, []byte("Test Message"))
	if err != nil {
		t.Error("cannot generate proof from a msg")
	}
	fmt.Println("pk is", hex.EncodeToString(pk))
	fmt.Println("sk is", hex.EncodeToString(sk))
	fmt.Println("pf is", hex.EncodeToString(proof))
	fmt.Println("the length of proof is of size ", len(proof), " bytes")
	flag, err := vrf.VerifyProof(pk, proof, []byte("Test Message"))
	if err != nil {
		t.Error("cannot verify a proof")
	}
	if flag != true {
		t.Error("cannot verify a proof")
	}
}
