/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package multivacaddress

import (
	"crypto/sha256"
	"fmt"
	"github.com/prometheus/common/log"
	"reflect"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"golang.org/x/crypto/ripemd160"
)

var publicKey = privateKey.Public()
var privateKey = signature.PrivateKey{
	158, 139, 132, 23, 249, 119, 67, 251,
	173, 194, 184, 163, 121, 5, 133, 141,
	123, 103, 187, 55, 104, 147, 54, 44,
	20, 47, 78, 40, 15, 112, 88, 125,
	1, 97, 158, 26, 138, 75, 21, 208,
	187, 249, 216, 252, 33, 7, 115, 33,
	146, 121, 3, 59, 119, 31, 54, 227,
	216, 51, 81, 75, 46, 177, 200, 223,
}

func TestGenerateAddress(t *testing.T) {
	var err error
	// There are normal results tests.
	addr1 := GenerateAddress(publicKey, SystemContractAddress)

	err = addr1.VerifyAddressType(SystemContractAddress)
	if err != nil {
		t.Errorf("verifyAddressType fails, err: %v", err)
	}

	err = addr1.VerifyPublicKey(publicKey, SystemContractAddress)
	if err != nil {
		t.Errorf("verifyPublicKey fails, err: %v", err)
	}

	addr2 := GenerateAddress(publicKey, UserAddress)

	err = addr2.VerifyAddressType(UserAddress)
	if err != nil {
		t.Errorf("verifyAddressType fails, err: %v", err)
	}

	err = addr2.VerifyPublicKey(publicKey, UserAddress)
	if err != nil {
		t.Errorf("verifyPublicKey fails, err: %v", err)
	}

	addr3 := GenerateAddress(publicKey, SmartContractAddress)

	err = addr3.VerifyAddressType(SmartContractAddress)
	if err != nil {
		t.Errorf("verifyAddressType fails, err: %v", err)
	}

	err = addr3.VerifyPublicKey(publicKey, SmartContractAddress)
	if err != nil {
		t.Errorf("verifyPublicKey fails, err: %v", err)
	}

	// There are error result tests.

	addr4 := GenerateAddress(publicKey, SmartContractAddress)

	addr4[0] = 'A'
	err = addr4.VerifyAddressType(SmartContractAddress)
	if err == nil {
		t.Errorf("validation fails, address prefix error should be reported, but not")
	}

	err = addr4.VerifyPublicKey(publicKey, SmartContractAddress)
	if err == nil {
		t.Errorf("validation fails, address prefix error should be reported, but not")
	}

	addr4[0] = 'M'

	fullPayload := base58.Decode(string(addr4[3:]))
	fullPayload[0] = byte(SystemContractAddress)

	copy(addr4[3:], fullPayload)

	err = addr4.VerifyAddressType(SmartContractAddress)
	if err == nil {
		t.Errorf("validation fails, fail to base58.Decode, decode results in length 0")
	}

	err = addr4.VerifyPublicKey(publicKey, SmartContractAddress)
	if err == nil {
		t.Errorf("validation fails, fail to base58.Decode, decode results in length 0")
	}

	addr5 := GenerateAddress(publicKey, SmartContractAddress)

	err = addr5.VerifyAddressType(SystemContractAddress)
	if err == nil {
		t.Errorf("validation fails, address prefix error should be reported, but not")
	}

	err = addr5.VerifyPublicKey(publicKey, SystemContractAddress)
	if err == nil {
		t.Errorf("validation fails, address prefix error should be reported, but not")
	}
}

func TestGetPublicKeyHash(t *testing.T) {
	addr1 := GenerateAddress(publicKey, SystemContractAddress)
	pkHash1, err := addr1.GetPublicKeyHash(SystemContractAddress)
	if err != nil {
		t.Errorf("fail to GetPublicKeyHash, err: %v", err)
	}
	publicRIPEMD160 := make([]byte, publicKeyHashSize)
	copy(publicRIPEMD160, publicKey)

	if !reflect.DeepEqual(pkHash1[:], publicRIPEMD160[:]) {
		t.Errorf("fail to GetPublicKeyHash, want: %v\n, get: %v", pkHash1, publicRIPEMD160)
	}

	pkHash1, err = addr1.GetPublicKeyHash(UserAddress)
	if err == nil {
		t.Errorf("getPublicKeyHash should have reported an error, but it didn't")
	}

	addr2 := GenerateAddress(publicKey, UserAddress)
	pkHash2, err := addr2.GetPublicKeyHash(UserAddress)
	if err != nil {
		log.Errorf("failed to get public key hash,err:%v", err)
	}

	pkHash256 := sha256.Sum256(publicKey)

	RIPEMD160Hasher := ripemd160.New()

	n, err := RIPEMD160Hasher.Write(pkHash256[:])

	if err != nil || n != len(pkHash256) {
		err := fmt.Errorf("fail to write data by RIPEMD160Hasher")
		panic(err)
	}
	publicRIPEMD160 = RIPEMD160Hasher.Sum(nil)

	if !reflect.DeepEqual(pkHash2[:], publicRIPEMD160[:]) {
		t.Errorf("fail to GetPublicKeyHash, want: %v\n, get: %v", pkHash1, publicRIPEMD160)
	}

	_, err = addr2.GetPublicKeyHash(SmartContractAddress)
	if err == nil {
		t.Errorf("getPublicKeyHash should have reported an error, but it didn't")
	}
}
