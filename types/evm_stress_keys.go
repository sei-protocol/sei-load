package types

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EvmStressPrivateKey derives a deterministic secp256k1 key from a 32-byte seed (idx in high bytes).
func EvmStressPrivateKey(idx uint64) (*ecdsa.PrivateKey, error) {
	seed := make([]byte, 32)
	seed[0] = 0x01
	for i := 0; i < 8; i++ {
		seed[1+i] = byte(idx >> (56 - 8*i))
	}
	return crypto.ToECDSA(seed)
}

// EvmStressRecipientAddress is key index 0 (fixed recipient for stress profiles).
func EvmStressRecipientAddress() common.Address {
	key, err := EvmStressPrivateKey(0)
	if err != nil {
		panic(fmt.Sprintf("evm stress key 0: %v", err))
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}

// GenerateEvmStressSenderAccounts returns accounts for indices 1..n (inclusive).
// Fund the corresponding native-chain accounts in genesis when using this pool.
func GenerateEvmStressSenderAccounts(n int) []*Account {
	if n < 1 {
		return nil
	}
	out := make([]*Account, 0, n)
	for i := uint64(1); i <= uint64(n); i++ {
		priv, err := EvmStressPrivateKey(i)
		if err != nil {
			panic(fmt.Sprintf("evm stress key %d: %v", i, err))
		}
		out = append(out, &Account{
			Address: crypto.PubkeyToAddress(priv.PublicKey),
			PrivKey:  priv,
		})
	}
	return out
}
