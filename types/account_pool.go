package types

import (
	"context"
	"time"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	mrand "math/rand/v2"
	"sync"
)

type accQueue struct {
	Txs   []*LoadTx
	Nonce uint64
}

// AccountRegistry owns account pools created for a run.
type TxsQueue struct {
	byAddr    map[common.Address]*accQueue
	untracked []*LoadTx
}

// NewAccountRegistry creates an empty account registry.
func NewTxsQueue() *TxsQueue {
	return &TxsQueue{byAddr: map[common.Address]*accQueue{}}
}

func (q *TxsQueue) Push(ctx context.Context, scenario *TxScenario, tx *ethtypes.Transaction) error {
	// TODO: make it respect Settings.MaxInFlight
	// TODO: make it blocking
	sender := scenario.Sender
	ltx := &LoadTx{EthTx:tx,IntendedSendTime:time.Now(),Scenario:scenario}
	if sender.Tracked {
		aq, ok := q.byAddr[sender.Address]
		if !ok {
			aq = &accQueue{}
			q.byAddr[sender.Address] = aq
		}
		if aq.Nonce != tx.Nonce() {
			return nil
		}
		aq.Nonce += 1
		aq.Txs = append(aq.Txs, ltx)
	} else {
		if tx.Nonce() != 0 {
			return nil
		}
		q.untracked = append(q.untracked, ltx)
	}
	return nil
}

func (q *TxsQueue) WaitUntilEmpty(ctx context.Context) error {
	panic("unimplemented")
}

func (q *TxsQueue) Nonce(addr common.Address) uint64 {
	if aq, ok := q.byAddr[addr]; ok {
		return aq.Nonce
	}
	return 0
}

func (q *TxsQueue) ResetNonce(addr common.Address, nonce uint64) {
	if a, ok := q.byAddr[addr]; ok {
		a.Txs = nil
		a.Nonce = nonce
	}
}

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
