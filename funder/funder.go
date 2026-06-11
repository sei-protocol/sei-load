// Package funder funds a generated account pool from a single root key so
// seiload can run against a real chain (one where accounts are not auto-funded
// by mock_balances or genesis). It fans out native value to every account via
// the Disperse contract before the load run starts.
package funder

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/sync/errgroup"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/types"
)

const balanceCheckConcurrency = 16

// FundAccounts funds every account across the given pools to at least the
// configured per-account amount, drawing from cfg.Funding's root key. The
// root's first EVM tx (the Disperse deploy) auto-associates its cosmos balance
// to the EVM side (Sei ante handler), so no explicit association step is
// required — the root must be funded at its EVM (cast) address or be already
// associated.
//
// Funding targets the current pool. seiload generates a fresh random pool each
// start, so a restart funds a new set of accounts (the prior set's balances are
// stranded — acceptable on a funny-money devnet, bounded by the root balance).
// The already-funded skip below guards against double-funding within a run.
func FundAccounts(ctx context.Context, cfg *config.LoadConfig, pools []types.AccountPool) error {
	fc := cfg.Funding
	if fc == nil {
		return nil
	}
	rootKeyHex, err := resolveRootKey(fc)
	if err != nil {
		return err
	}
	// TrimSpace is load-bearing: a SOPS-mounted key file commonly carries a
	// trailing newline.
	rootKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(rootKeyHex), "0x"))
	if err != nil {
		return fmt.Errorf("funder: parse root key: %w", err)
	}
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("funder: no endpoints configured")
	}

	client, err := ethclient.Dial(cfg.Endpoints[0])
	if err != nil {
		return fmt.Errorf("funder: dial %s: %w", cfg.Endpoints[0], err)
	}
	defer client.Close()

	recipients := uniqueAddresses(pools)
	if len(recipients) == 0 {
		log.Printf("💰 funder: no accounts to fund")
		return nil
	}
	amount := fc.FundAmount()
	log.Printf("💰 funder: %d accounts, target %s wei each, from %s",
		len(recipients), amount.String(), crypto.PubkeyToAddress(rootKey.PublicKey).Hex())

	underfunded, err := filterUnderfunded(ctx, client, recipients, amount)
	if err != nil {
		return err
	}
	if len(underfunded) == 0 {
		log.Printf("💰 funder: all accounts already funded — nothing to do")
		return nil
	}
	log.Printf("💰 funder: %d of %d need funding", len(underfunded), len(recipients))

	chainID := cfg.GetChainID()
	auth, err := bind.NewKeyedTransactorWithChainID(rootKey, chainID)
	if err != nil {
		return fmt.Errorf("funder: transactor: %w", err)
	}
	auth.Context = ctx
	auth.GasTipCap = big.NewInt(1_000_000_000)   // 1 gwei (chain min fee)
	auth.GasFeeCap = big.NewInt(100_000_000_000) // 100 gwei cap

	disperse, err := deployDisperse(ctx, client, auth)
	if err != nil {
		return err
	}

	// Sequential by design: auth.Nonce stays nil so bind fetches PendingNonceAt
	// per tx, and WaitMined gates each batch — the prior nonce is mined and
	// visible before the next send. Do not parallelize batches or set
	// auth.Nonce without reworking this.
	batch := fc.Batch()
	for start := 0; start < len(underfunded); start += batch {
		end := start + batch
		if end > len(underfunded) {
			end = len(underfunded)
		}
		chunk := underfunded[start:end]
		values := make([]*big.Int, len(chunk))
		total := new(big.Int)
		for i := range chunk {
			values[i] = amount // read-only by the contract call; safe to share
			total.Add(total, amount)
		}
		auth.Value = total
		tx, err := disperse.DisperseEther(auth, chunk, values)
		if err != nil {
			return fmt.Errorf("funder: disperseEther [%d:%d]: %w", start, end, err)
		}
		if err := waitSuccess(ctx, client, tx, "disperseEther"); err != nil {
			return err
		}
		log.Printf("💰 funder: funded %d/%d (tx %s)", end, len(underfunded), tx.Hash().Hex())
	}
	auth.Value = nil
	log.Printf("✅ funder: funding complete")
	return nil
}

func resolveRootKey(fc *config.FundingConfig) (string, error) {
	if fc.RootKeyFile != "" {
		b, err := os.ReadFile(fc.RootKeyFile)
		if err != nil {
			return "", fmt.Errorf("funder: read rootKeyFile: %w", err)
		}
		if len(strings.TrimSpace(string(b))) == 0 {
			return "", fmt.Errorf("funder: rootKeyFile %s is empty", fc.RootKeyFile)
		}
		return string(b), nil
	}
	if fc.RootKeyEnv != "" {
		v := os.Getenv(fc.RootKeyEnv)
		if v == "" {
			return "", fmt.Errorf("funder: env %s is empty", fc.RootKeyEnv)
		}
		return v, nil
	}
	return "", fmt.Errorf("funder: no root key (set funding.rootKeyFile or funding.rootKeyEnv)")
}

func uniqueAddresses(pools []types.AccountPool) []common.Address {
	seen := make(map[common.Address]struct{})
	var out []common.Address
	for _, p := range pools {
		for _, a := range p.GetAccounts() {
			if _, ok := seen[a.Address]; ok {
				continue
			}
			seen[a.Address] = struct{}{}
			out = append(out, a.Address)
		}
	}
	return out
}

// filterUnderfunded returns the subset of addresses whose balance is below
// amount, querying balances concurrently. The errgroup bounds concurrency and
// cancels all in-flight queries on the first error or on ctx cancellation, with
// no goroutine leak.
func filterUnderfunded(ctx context.Context, client *ethclient.Client, addrs []common.Address, amount *big.Int) ([]common.Address, error) {
	var (
		mu          sync.Mutex
		underfunded []common.Address
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(balanceCheckConcurrency)
	for _, a := range addrs {
		g.Go(func() error {
			bal, err := client.BalanceAt(gctx, a, nil)
			if err != nil {
				return fmt.Errorf("funder: balance %s: %w", a.Hex(), err)
			}
			if bal.Cmp(amount) < 0 {
				mu.Lock()
				underfunded = append(underfunded, a)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return underfunded, nil
}

// deployDisperse deploys a fresh Disperse contract. This is the root's first
// EVM tx, which also auto-associates the root. Deploying fresh (rather than
// trusting a configured address) avoids sending the root's value to an
// unverified contract.
func deployDisperse(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*bindings.Disperse, error) {
	addr, tx, d, err := bindings.DeployDisperse(auth, client, big.NewInt(0), big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("funder: deploy Disperse: %w", err)
	}
	if err := waitSuccess(ctx, client, tx, "deploy Disperse"); err != nil {
		return nil, err
	}
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("funder: CodeAt %s: %w", addr.Hex(), err)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("funder: deployed Disperse at %s has no code", addr.Hex())
	}
	log.Printf("💰 funder: deployed Disperse at %s", addr.Hex())
	return d, nil
}

// waitSuccess blocks until tx is mined and asserts it did not revert.
func waitSuccess(ctx context.Context, client *ethclient.Client, tx *ethtypes.Transaction, what string) error {
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("funder: wait %s (%s): %w", what, tx.Hash().Hex(), err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("funder: %s reverted (tx %s)", what, tx.Hash().Hex())
	}
	return nil
}
