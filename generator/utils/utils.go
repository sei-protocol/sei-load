// Package utils builds the go-ethereum transact options behind seiload's two
// transaction paths: a contract deployment, which is signed and sent live at
// startup, and a load transaction, which is shaped offline and sent by the
// sender package.
//
// # Nonce sourcing
//
// The two paths source their nonce differently, and the difference is the point.
// A load transaction pins the nonce the generator assigned it: the sender owns
// that per-account sequence, and no RPC round-trip belongs on the hot path. A
// deployment leaves the nonce unset, so go-ethereum reads the deployer's pending
// nonce from the chain. That is what makes a deployer with on-chain history safe
// — the funding root spends nonces of its own, before and after these
// deployments.
package utils

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	loadtypes "github.com/sei-protocol/sei-load/types"
)

const (
	// deployGasLimit is the gas limit every contract creation is sent with.
	deployGasLimit = 3_000_000
	// txGasLimit is the default per-transaction limit; a scenario that knows its
	// own cost overrides it.
	txGasLimit = 200_000
	// gasTipCapWei is the priority fee (2 gwei).
	gasTipCapWei = 2_000_000_000
	// gasFeeCapWei is the max fee, base plus priority (20 gwei).
	gasFeeCapWei = 20_000_000_000
	// deployGasFeeCapWei is the max fee for a contract creation (100 gwei),
	// matching the funding path because both sign from the same key.
	deployGasFeeCapWei = 100_000_000_000
)

// CreateDeploymentOpts returns the options for a contract deployment signed by
// account. The transaction is sent live, so ctx bounds the send and the nonce
// fetch behind it.
func CreateDeploymentOpts(ctx context.Context, chainID *big.Int, account loadtypes.Account) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(account.PrivKey, chainID)
	if err != nil {
		return nil, err
	}
	auth.Context = ctx
	auth.GasLimit = deployGasLimit
	auth.GasTipCap = big.NewInt(gasTipCapWei)
	// A deploy is the first transaction on the deployer's nonce stream, and when
	// funding is configured that stream belongs to the root key. Pricing it at
	// the load-transaction cap would put the stream's weakest-priced transaction
	// at its head, so a base fee above that cap blocks every later root
	// transaction until someone replaces the nonce by hand. Match the funding
	// cap instead — the same key, the same exposure, one number.
	auth.GasFeeCap = big.NewInt(deployGasFeeCapWei)
	return auth, nil
}

// CreateTransactionOpts returns the options for one load transaction against a
// contract. NoSend keeps the transaction in hand for the sender, and the signer
// hands it back unsigned: the sender signs it at send time.
func CreateTransactionOpts(chainID *big.Int, scenario *loadtypes.TxScenario) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(scenario.Sender.PrivKey, chainID)
	if err != nil {
		panic("Failed to create transaction options: " + err.Error())
	}
	auth.Nonce = new(big.Int).SetUint64(scenario.Nonce)
	auth.NoSend = true
	auth.GasLimit = txGasLimit
	auth.GasTipCap = big.NewInt(gasTipCapWei)
	auth.GasFeeCap = big.NewInt(gasFeeCapWei)
	auth.Signer = func(address common.Address, tx *ethtypes.Transaction) (*ethtypes.Transaction, error) {
		if address != scenario.Sender.Address {
			return nil, bind.ErrNotAuthorized
		}
		return tx, nil
	}
	return auth
}
