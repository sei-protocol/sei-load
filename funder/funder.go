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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/types"
)

const balanceCheckConcurrency = 16

// FundAccounts funds every account across the given pools to at least the
// configured per-account amount, drawing from cfg.Funding's root key. It is
// idempotent: accounts already at/above the target are skipped, so a pod
// restart re-funds only what was spent. The first EVM tx the root sends
// auto-associates its cosmos balance to the EVM side (Sei ante handler), so no
// explicit association step is required.
func FundAccounts(ctx context.Context, cfg *config.LoadConfig, pools []types.AccountPool) error {
	fc := cfg.Funding
	if fc == nil {
		return nil
	}
	rootKeyHex, err := resolveRootKey(fc)
	if err != nil {
		return err
	}
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

	disperse, err := getDisperse(ctx, client, auth, fc)
	if err != nil {
		return err
	}

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
			values[i] = new(big.Int).Set(amount)
			total.Add(total, amount)
		}
		auth.Value = total
		tx, err := disperse.DisperseEther(auth, chunk, values)
		if err != nil {
			return fmt.Errorf("funder: disperseEther [%d:%d]: %w", start, end, err)
		}
		if _, err := bind.WaitMined(ctx, client, tx); err != nil {
			return fmt.Errorf("funder: wait disperse [%d:%d]: %w", start, end, err)
		}
		log.Printf("💰 funder: funded %d/%d (tx %s)", end, len(underfunded), tx.Hash().Hex())
	}
	auth.Value = nil
	log.Printf("✅ funder: funding complete")
	return nil
}

func resolveRootKey(fc *config.FundingConfig) (string, error) {
	if fc.RootKeyEnv != "" {
		v := os.Getenv(fc.RootKeyEnv)
		if v == "" {
			return "", fmt.Errorf("funder: env %s is empty", fc.RootKeyEnv)
		}
		return v, nil
	}
	if fc.RootKey != "" {
		return fc.RootKey, nil
	}
	return "", fmt.Errorf("funder: no root key (set funding.rootKeyEnv or funding.rootKey)")
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
// amount, querying balances concurrently.
func filterUnderfunded(ctx context.Context, client *ethclient.Client, addrs []common.Address, amount *big.Int) ([]common.Address, error) {
	type res struct {
		addr common.Address
		low  bool
		err  error
	}
	in := make(chan common.Address)
	out := make(chan res)
	var wg sync.WaitGroup
	for i := 0; i < balanceCheckConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range in {
				bal, err := client.BalanceAt(ctx, a, nil)
				if err != nil {
					out <- res{addr: a, err: err}
					continue
				}
				out <- res{addr: a, low: bal.Cmp(amount) < 0}
			}
		}()
	}
	go func() {
		for _, a := range addrs {
			in <- a
		}
		close(in)
	}()
	go func() { wg.Wait(); close(out) }()

	var underfunded []common.Address
	for r := range out {
		if r.err != nil {
			return nil, fmt.Errorf("funder: balance check %s: %w", r.addr.Hex(), r.err)
		}
		if r.low {
			underfunded = append(underfunded, r.addr)
		}
	}
	return underfunded, nil
}

// getDisperse reuses a pre-deployed Disperse contract when an address is
// configured, otherwise deploys one (the root's first EVM tx, which also
// auto-associates the root).
func getDisperse(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, fc *config.FundingConfig) (*bindings.Disperse, error) {
	if fc.DisperseAddress != "" {
		addr := common.HexToAddress(fc.DisperseAddress)
		d, err := bindings.NewDisperse(addr, client)
		if err != nil {
			return nil, fmt.Errorf("funder: bind Disperse at %s: %w", addr.Hex(), err)
		}
		log.Printf("💰 funder: using pre-deployed Disperse at %s", addr.Hex())
		return d, nil
	}
	addr, tx, d, err := bindings.DeployDisperse(auth, client, big.NewInt(0), big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("funder: deploy Disperse: %w", err)
	}
	if _, err := bind.WaitMined(ctx, client, tx); err != nil {
		return nil, fmt.Errorf("funder: wait Disperse deploy: %w", err)
	}
	log.Printf("💰 funder: deployed Disperse at %s", addr.Hex())
	return d, nil
}
