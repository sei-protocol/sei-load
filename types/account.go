package types

import (
	"crypto/ecdsa"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sei-protocol/sei-load/utils"
)

// Account wraps address and private key.
type Account struct {
	Address common.Address
	PrivKey *ecdsa.PrivateKey
	Nonce   atomic.Uint64
}

// NewAccount generates new account.
func NewAccount() *Account {
	privateKey := utils.OrPanic1(crypto.GenerateKey())
	return &Account{
		Address: crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivKey: privateKey,
	}
}

// GetAndIncrementNonce increments the nonce.
func (s *Account) GetAndIncrementNonce() uint64 {
	return s.Nonce.Add(1) - 1
}

// GenerateAccounts generates random accounts.
func GenerateAccounts(n int) []*Account {
	result := make([]*Account, n)
	for i := range result {
		result[i] = NewAccount()
	}
	return result
}
