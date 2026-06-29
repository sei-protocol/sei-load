package types

import (
	mrand "math/rand/v2"
	"sync"
)

// AccountPool returns a next account for load generation.
type AccountPool struct {
	newAccountRate float64
	accounts       []Account
	mx             sync.Mutex
	idx            int
}

// AccountConfig stores the configuration for account generation.
type AccountConfig struct {
	InitialSize    int
	NewAccountRate float64
}

func (a *AccountPool) nextIndex() int {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.idx++
	a.idx %= len(a.accounts)
	return a.idx
}

// NextAccount returns the next account, using rng for the new-account roll when
// NewAccountRate > 0.
func (a *AccountPool) NextAccount(rng *mrand.Rand) Account {
	if a.newAccountRate > 0 {
		if rng.Float64() <= a.newAccountRate {
			return NewAccount(false)
		}
	}
	return a.accounts[a.nextIndex()]
}

// GetAccounts returns the fixed accounts backing the pool.
func (a *AccountPool) Accounts() []Account { return a.accounts }

// NewPool creates a new account generator from a config, records it, and returns it.
func NewAccountPool(size int, newAccountRate float64) *AccountPool {
	return &AccountPool{
		accounts:       GenerateAccounts(size, true),
		newAccountRate: newAccountRate,
	}
}
