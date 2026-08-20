package types

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sei-protocol/sei-load/utils"
)

// Account wraps address and private key.
type Account struct {
	Address common.Address
	PrivKey *ecdsa.PrivateKey
	Tracked bool
}

// NewAccount generates new account.
func NewAccount(tracked bool) Account {
	return AccountFromKey(utils.OrPanic1(crypto.GenerateKey()), tracked)
}

// AccountFromKey wraps an existing private key, deriving its EVM address.
func AccountFromKey(privKey *ecdsa.PrivateKey, tracked bool) Account {
	return Account{
		Address: crypto.PubkeyToAddress(privKey.PublicKey),
		PrivKey: privKey,
		Tracked: tracked,
	}
}

// GenerateAccounts generates random accounts.
func GenerateAccounts(n int, tracked bool) []Account {
	result := make([]Account, n)
	for i := range result {
		result[i] = NewAccount(tracked)
	}
	return result
}
