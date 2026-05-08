package types

import (
	"fmt"
	"math/rand"
	"sync"
)

// AccountPool returns a next account for load generation.
type AccountPool interface {
	NextAccount() *Account
}

// AccountConfig stores the configuration for account generation.
type AccountConfig struct {
	Accounts       []*Account
	NewAccountRate float64
	// SingleUseSenders requires NewAccountRate == 0 (enforced by NewAccountPool).
	// Incompatible with settings.prewarm (enforced at seiload startup in config.ValidatePrewarmAccountPools).
	SingleUseSenders bool
}

type accountPool struct {
	Accounts []*Account
	cfg      *AccountConfig

	mx  sync.Mutex
	idx int
}

func (a *accountPool) nextIndex() int {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.idx++
	a.idx %= len(a.Accounts)
	return a.idx
}

// NextAccount returns the next account.
func (a *accountPool) NextAccount() *Account {
	if a.cfg.NewAccountRate > 0 {
		randomNumber := rand.Float64()
		if randomNumber <= a.cfg.NewAccountRate {
			return GenerateAccounts(1)[0]
		}
	}
	if a.cfg.SingleUseSenders {
		a.mx.Lock()
		defer a.mx.Unlock()
		if a.idx >= len(a.Accounts) {
			return nil
		}
		acc := a.Accounts[a.idx]
		a.idx++
		return acc
	}
	return a.Accounts[a.nextIndex()]
}

// NewAccountPool creates a new account generator from a config.
func NewAccountPool(cfg *AccountConfig) (AccountPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("account pool config is nil")
	}
	if cfg.SingleUseSenders && cfg.NewAccountRate > 0 {
		return nil, fmt.Errorf(
			"account pool: singleUseSenders is incompatible with newAccountRate > 0 (got newAccountRate=%g)",
			cfg.NewAccountRate,
		)
	}
	return &accountPool{
		Accounts: cfg.Accounts,
		cfg:      cfg,
	}, nil
}
