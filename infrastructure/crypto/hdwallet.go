package crypto

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"

	bip39 "github.com/tyler-smith/go-bip39"
)

const (
	hardenedOffset = 0x80000000
	bitcoinSeed    = "Bitcoin seed"
)

func DeriveKeyFromMnemonic(mnemonic string, index uint32) (privateKeyHex string, addressHex string, err error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return "", "", fmt.Errorf("invalid mnemonic")
	}

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return "", "", fmt.Errorf("generate seed: %w", err)
	}

	masterKey, masterChainCode, err := deriveMasterKey(seed)
	if err != nil {
		return "", "", fmt.Errorf("derive master key: %w", err)
	}

	key, chainCode := masterKey, masterChainCode

	key, chainCode, err = deriveChildKey(key, chainCode, 44+hardenedOffset)
	if err != nil {
		return "", "", fmt.Errorf("derive m/44': %w", err)
	}
	key, chainCode, err = deriveChildKey(key, chainCode, 60+hardenedOffset)
	if err != nil {
		return "", "", fmt.Errorf("derive m/44'/60': %w", err)
	}
	key, chainCode, err = deriveChildKey(key, chainCode, 0+hardenedOffset)
	if err != nil {
		return "", "", fmt.Errorf("derive m/44'/60'/0': %w", err)
	}
	key, chainCode, err = deriveChildKey(key, chainCode, 0)
	if err != nil {
		return "", "", fmt.Errorf("derive m/44'/60'/0'/0: %w", err)
	}
	key, _, err = deriveChildKey(key, chainCode, index)
	if err != nil {
		return "", "", fmt.Errorf("derive m/44'/60'/0'/0/%d: %w", index, err)
	}

	privateKeyHex = hex.EncodeToString(crypto.FromECDSA(key))
	addressHex = crypto.PubkeyToAddress(key.PublicKey).Hex()

	return privateKeyHex, addressHex, nil
}

func deriveMasterKey(seed []byte) (*ecdsa.PrivateKey, []byte, error) {
	mac := hmac.New(sha512.New, []byte(bitcoinSeed))
	mac.Write(seed)
	I := mac.Sum(nil)

	IL := I[:32]
	IR := I[32:]

	key, err := crypto.ToECDSA(IL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid master key: %w", err)
	}

	return key, IR, nil
}

func deriveChildKey(parentKey *ecdsa.PrivateKey, parentChainCode []byte, index uint32) (*ecdsa.PrivateKey, []byte, error) {
	isHardened := index >= hardenedOffset

	var data []byte
	if isHardened {
		data = append([]byte{0x00}, crypto.FromECDSA(parentKey)...)
	} else {
		data = compressPubkey(&parentKey.PublicKey)
	}

	idxBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idxBytes, index)
	data = append(data, idxBytes...)

	mac := hmac.New(sha512.New, parentChainCode)
	mac.Write(data)
	I := mac.Sum(nil)

	IL := new(big.Int).SetBytes(I[:32])
	IR := I[32:]

	curveOrder := crypto.S256().Params().N
	IL.Add(IL, parentKey.D)
	IL.Mod(IL, curveOrder)

	if IL.Sign() == 0 {
		return nil, nil, fmt.Errorf("derived child key is zero (index %d)", index)
	}

	childKey, err := crypto.ToECDSA(padTo32(IL.Bytes()))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid child key at index %d: %w", index, err)
	}

	return childKey, IR, nil
}

func compressPubkey(pub *ecdsa.PublicKey) []byte {
	b := make([]byte, 33)
	if pub.Y.Bit(0) == 1 {
		b[0] = 0x03
	} else {
		b[0] = 0x02
	}
	copy(b[33-len(pub.X.Bytes()):], pub.X.Bytes())
	return b
}

func padTo32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}
