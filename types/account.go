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
	Nonce  uint64
	Txs []*LoadTx
}

// NewAccount generates new account.
func NewAccount() *Account {
	privateKey := utils.OrPanic1(crypto.GenerateKey())
	return &Account{
		Address: crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivKey: privateKey,
	}
}

func (s *Account) PushTx(tx *LoadTx) {
	if tx.EthTx.Nonce()!=s.Nonce {
		return
	}
	s.Nonce += 1
	s.Txs = append(s.Txs,tx)
}

// GenerateAccounts generates random accounts.
func GenerateAccounts(n int) []*Account {
	result := make([]*Account, n)
	for i := range result {
		result[i] = NewAccount()
	}
	return result
}
