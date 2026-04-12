package chain

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// PackAndSign generates a deterministic keccak256 hash matching Solidity's abi.encodePacked
// and signs it with the provided ECDSA private key.
func PackAndSign(privateKey *ecdsa.PrivateKey, args ...[]byte) ([]byte, error) {
	// Solidity abi.encodePacked simply concatenates bytes
	var packed []byte
	for _, arg := range args {
		packed = append(packed, arg...)
	}

	// 1. Hash the packed arguments
	digest := crypto.Keccak256(packed)
	// Alternatively crypto.Keccak256Hash(packed).Bytes()

	// 2. Wrap in Ethereum Signed Message Prefix
	prefixedHash := accounts.TextHash(digest)

	// 3. Sign the prefixed hash
	signature, err := crypto.Sign(prefixedHash, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payload: %v", err)
	}

	// Adjust V value to Ethereum standard (27/28)
	if len(signature) == 65 {
		signature[64] += 27
	}

	return signature, nil
}

// EncodeAddress encodes an address to 20 bytes
func EncodeAddress(addr string) []byte {
	return common.HexToAddress(addr).Bytes()
}

// EncodeString encodes a string into its byte representation
func EncodeString(str string) []byte {
	return []byte(str)
}

// EncodeUint256 encodes a uint256 string to its 32-byte representation.
// Returns an error if the string is not a valid number.
func EncodeUint256(val string) ([]byte, error) {
	parsed, ok := math.ParseBig256(val)
	if !ok || parsed == nil {
		return nil, fmt.Errorf("invalid uint256 string: %s", val)
	}
	return common.LeftPadBytes(parsed.Bytes(), 32), nil
}
