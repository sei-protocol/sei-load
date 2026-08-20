package funder

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/types"
)

// Deployer returns the account that signs the run's contract deployments.
//
// With funding configured that account is the funding root: the one key the run
// knows to hold a balance, and the key this package already deploys Disperse
// from. Without funding — or under mock deploy, where no deployment reaches a
// chain — it is a fresh random account, payable only by a chain that credits
// unknown senders (a mock chain or a pre-funded genesis).
//
// It fails only when funding is configured and its root key cannot be read or
// parsed.
func Deployer(cfg *config.LoadConfig) (types.Account, error) {
	if cfg.Funding == nil || cfg.MockDeploy {
		return types.NewAccount(false), nil
	}
	return rootAccount(cfg.Funding)
}

// rootAccount resolves the configured root key into the account that spends it.
func rootAccount(fc *config.FundingConfig) (types.Account, error) {
	rootKeyHex, err := resolveRootKey(fc)
	if err != nil {
		return types.Account{}, err
	}
	// TrimSpace: a SOPS-mounted key file commonly carries a trailing newline.
	rootKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(rootKeyHex), "0x"))
	if err != nil {
		return types.Account{}, fmt.Errorf("funder: parse root key: %w", err)
	}
	return types.AccountFromKey(rootKey, false), nil
}

// resolveRootKey reads the hex root key from the configured file, else from the
// configured environment variable.
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
