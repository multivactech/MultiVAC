/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package multivacaddress

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"golang.org/x/crypto/ripemd160"
)

// AddressType define the multivac address type.
type AddressType byte

const (
	// UserAddress defines the address type for user.Base58Check encoding, it is prefixed with 1
	UserAddress AddressType = 0x00
	// SystemContractAddress defines the address type for system contract address.
	SystemContractAddress AddressType = 0x01
	// SmartContractAddress defines the address type for smart contract address.
	// Base58Check encoding, it is prefixed with 3.
	SmartContractAddress AddressType = 0x05
)

const (
	addrTypeSize      int = 1  // The address type of multiVAC.
	publicKeyHashSize int = 20 // Byte size after double Hash.
	checkSumSize      int = 4  // The byte size of the checksum
)

// Address defines the data structure of multivac address.
type Address []byte

// PublicKeyHash is an array used to store the hasn of public key.
type PublicKeyHash [publicKeyHashSize]byte

// GenerateAddressByPublicKeyHash accept after sha256 and ripemd hash length of 20 byte array,
// and the address type as a parameter to generate the MTV address.
func GenerateAddressByPublicKeyHash(pubKey PublicKeyHash, addrType AddressType) Address {
	versionPayload := append([]byte{byte(addrType)}, pubKey[:]...)

	// Calculate checkSum.
	firstSHA := sha256.Sum256(versionPayload)
	secondSHA := sha256.Sum256(firstSHA[:])

	// CheckSum is the first four bytes.
	checkSum := secondSHA[:4]

	// Base58 encode.
	fullPayload := append(versionPayload, checkSum...)
	address := base58.Encode(fullPayload)
	address = fmt.Sprintf("MTV%s", address)

	return Address(address)

}

// GenerateAddress generates the MTV address based on the public key and address type
func GenerateAddress(pk signature.PublicKey, addrType AddressType) Address {
	publicRIPEMD160 := make([]byte, publicKeyHashSize)
	if addrType == UserAddress {
		// Calculate publicKeyHash.
		pkHash256 := sha256.Sum256(pk)

		RIPEMD160Hasher := ripemd160.New()

		n, err := RIPEMD160Hasher.Write(pkHash256[:])

		if err != nil || n != len(pkHash256) {
			err := fmt.Errorf("fail to write data by RIPEMD160Hasher")
			panic(err)
		}

		publicRIPEMD160 = RIPEMD160Hasher.Sum(nil)

	} else {
		copy(publicRIPEMD160, pk)
	}
	pubKey := [publicKeyHashSize]byte{}
	copy(pubKey[:], publicRIPEMD160)

	return GenerateAddressByPublicKeyHash(pubKey, addrType)
}

// VerifyAddressType determines whether it is legal by address type.
// This method only verifies that the address is formatted correctly.
func (addr Address) VerifyAddressType(addrType AddressType) error {
	var err error

	if !bytes.Equal(addr[:3], []byte("MTV")) {
		err = fmt.Errorf("address prefix error, want:%s, get:%s", "MTV", addr[:3])
		return err
	}

	fullPayload := base58.Decode(string(addr[3:]))

	if len(fullPayload) == 0 {
		err = fmt.Errorf("fail to base58.Decode address, decode results in length 0")
		return err
	}

	if fullPayload[0] != byte(addrType) {
		err = fmt.Errorf("addressType error, want:%d, get:%d", addrType, fullPayload[0])
		return err
	}

	versionAndPkHash := fullPayload[:addrTypeSize+publicKeyHashSize]
	checkSum := fullPayload[addrTypeSize+publicKeyHashSize : addrTypeSize+publicKeyHashSize+checkSumSize]

	firstSHA := sha256.Sum256(versionAndPkHash)
	secondSHA := sha256.Sum256(firstSHA[:])

	if !bytes.Equal(secondSHA[:4], checkSum) {
		err = fmt.Errorf("fail to check checkSum")
		return err
	}
	return nil
}

// VerifyPublicKey verifies that the address format is valid before verifying that the address and public key match
func (addr Address) VerifyPublicKey(pk signature.PublicKey, addrType AddressType) error {
	var err error

	if !bytes.Equal(addr[:3], []byte("MTV")) {
		err = fmt.Errorf("address prefix error, want:%s, get:%s", "MTV", addr[:3])
		return err
	}

	fullPayload := base58.Decode(string(addr[3:]))

	if len(fullPayload) == 0 {
		err = fmt.Errorf("fail to base58.Decode address, decode results in length 0")
		return err
	}

	if fullPayload[0] != byte(addrType) {
		err = fmt.Errorf("addressType error, want:%d, get:%d", addrType, fullPayload[0])
		return err
	}

	pkHash := fullPayload[addrTypeSize : addrTypeSize+publicKeyHashSize] // Addr's public key hash section.
	versionAndPkHash := fullPayload[:addrTypeSize+publicKeyHashSize]
	checkSum := fullPayload[addrTypeSize+publicKeyHashSize : addrTypeSize+publicKeyHashSize+checkSumSize]

	firstSHA := sha256.Sum256(versionAndPkHash)
	secondSHA := sha256.Sum256(firstSHA[:])

	if !bytes.Equal(secondSHA[:4], checkSum) {
		err = fmt.Errorf("fail to check checkSum")
		return err
	}

	var publicRIPEMD160 []byte

	// According to the parameter public key, the double hash result is calculated.
	if addrType == UserAddress {
		pkHash256 := sha256.Sum256(pk)

		RIPEMD160Hasher := ripemd160.New()

		n, err := RIPEMD160Hasher.Write(pkHash256[:])

		if err != nil || n != len(pkHash256) {
			err := fmt.Errorf("fail to write data by RIPEMD160Hasher")
			panic(err)
		}

		publicRIPEMD160 = RIPEMD160Hasher.Sum(nil)

	} else {
		publicRIPEMD160 = make([]byte, publicKeyHashSize)
		copy(publicRIPEMD160, pk)

	}

	if !bytes.Equal(pkHash, publicRIPEMD160) {
		err := fmt.Errorf("the public key does not match the address")
		return err
	}

	return nil

}

// IsEqual returns true if target is the same as hash.
func (addr Address) IsEqual(target Address) bool {
	if addr == nil && target == nil {
		return true
	}
	if addr == nil || target == nil {
		return false
	}
	return bytes.Equal(addr, target)
}

// GetPublicKeyHash returns the hash array of public key.
func (addr Address) GetPublicKeyHash(addrType AddressType) (*PublicKeyHash, error) {

	rtn := &PublicKeyHash{}
	if err := addr.VerifyAddressType(addrType); err != nil {
		return rtn, err
	}
	fullPayload := base58.Decode(string(addr[3:]))
	pkHash := fullPayload[addrTypeSize : len(fullPayload)-checkSumSize]

	copy(rtn[:], pkHash)
	return rtn, nil

}

func (addr Address) String() string {
	return string(addr)
}

// StringToUserAddress converts pk string to multivac user address
func StringToUserAddress(str string) (Address, error) {
	if len(str) == 64 {
		pkBytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		address := GenerateAddress(signature.PublicKey(pkBytes), UserAddress)
		return address, nil
	}
	addr := Address(str)
	if err := addr.VerifyAddressType(UserAddress); err != nil {
		return nil, err
	}
	return addr, nil
}

// GetDefaultAddrShard returns the default shard of address.
func GetDefaultAddrShard(mtvAddr string) int {
	sum := 0
	for _, x := range mtvAddr {
		sum += int(x)
	}
	x := sum % 4
	return x
}
