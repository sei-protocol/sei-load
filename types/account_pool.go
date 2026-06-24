package types

import (
	mrand "math/rand/v2"
	"sync"
	"github.com/ethereum/go-ethereum/common"
)

// AccountRegistry owns account pools created for a run.
type AccountRegistry struct {
	accounts map[common.Address]*Account
	newAccountsTxs []*LoadTx
	pools []*AccountPool
}

// NewAccountRegistry creates an empty account registry.
func NewAccountRegistry() *AccountRegistry {
	return &AccountRegistry{
		accounts: map[common.Address]*Account{},
	}
}

func (r *AccountRegistry) ResetNonce(addr common.Address, nonce uint64) {
	if a,ok := r.accounts[addr]; ok {
		a.Txs = nil
		a.Nonce = nonce
	}
}

// Accounts returns a flat copy of all accounts across all pools.
func (r *AccountRegistry) Accounts() []*Account {
	var accounts []*Account
	for _, pool := range r.pools {
		accounts = append(accounts, pool.GetAccounts()...)
	}
	return accounts
}

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
}

func (a *AccountPool) nextIndex() int {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.idx++
	a.idx %= len(a.Accounts)
	return a.idx
}

// NextAccount returns the next account, using rng for the new-account roll when
// NewAccountRate > 0.
func (a *AccountPool) NextAccount(rng *mrand.Rand) *Account {
	if a.cfg.NewAccountRate > 0 {
		if rng.Float64() <= a.cfg.NewAccountRate {
			return GenerateAccounts(1)[0]
		}
	}
	return a.Accounts[a.nextIndex()]
}

// GetAccounts returns the fixed accounts backing the pool.
func (a *AccountPool) GetAccounts() []*Account {
	return a.Accounts
}

// NewPool creates a new account generator from a config, records it, and returns it.
func (r *AccountRegistry) NewPool(cfg *AccountConfig) *AccountPool {
	accounts := GenerateAccounts(cfg.InitialSize)
	pool := &AccountPool{
		Accounts: accounts,
		cfg:      cfg,
	}
	for _,a := range accounts {
		r.accounts[a.Address] = a
	}
	r.pools = append(r.pools, pool)
	return pool
}
