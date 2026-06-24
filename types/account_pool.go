package types

import (
	"math/rand/v2"
	"sync"

	"github.com/sei-protocol/sei-load/utils/rng"
)

// AccountPool returns a next account for load generation.
type AccountPool struct {
	Accounts []*Account
	cfg      *AccountConfig

	mx  sync.Mutex
	idx int
}

// AccountConfig stores the configuration for account generation.
type AccountConfig struct {
	InitialSize    int
	NewAccountRate float64
	// Stream, when non-nil, makes the new-account roll deterministic. A nil
	// Stream leaves the pool on the unseeded global RNG.
	Stream *rng.Stream
}

func (a *AccountPool) nextIndex() int {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.idx++
	a.idx %= len(a.Accounts)
	return a.idx
}

// NextAccount returns the next account.
func (a *AccountPool) NextAccount() *Account {
	if a.cfg.NewAccountRate > 0 {
		var randomNumber float64
		if a.cfg.Stream != nil {
			randomNumber = a.cfg.Stream.Float64()
		} else {
			randomNumber = rand.Float64()
		}
		if randomNumber <= a.cfg.NewAccountRate {
			return GenerateAccounts(1)[0]
		}
	}
	return a.Accounts[a.nextIndex()]
}

// GetAccounts returns the fixed accounts backing the pool.
func (a *AccountPool) GetAccounts() []*Account {
	return a.Accounts
}

// NewAccountPool creates a new account generator from a config.
func NewAccountPool(cfg *AccountConfig) *AccountPool {
	return &AccountPool{
		Accounts: GenerateAccounts(cfg.InitialSize),
		cfg:      cfg,
	}
}
