package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	loadtypes "github.com/sei-protocol/sei-load/types"
)

// CreateTransactOpts creates transaction options for contract deployment or interaction
func createTransactOpts(chainID *big.Int, account loadtypes.Account, gasLimit uint64, nonce uint64, noSend bool) (*bind.TransactOpts, error) {
	// Create transactor
	auth, err := bind.NewKeyedTransactorWithChainID(account.PrivKey, chainID)
	if err != nil {
		return nil, err
	}

	// Set transaction parameters
	auth.Nonce = big.NewInt(int64(nonce))
	auth.NoSend = noSend

	auth.GasLimit = gasLimit
	auth.GasTipCap = big.NewInt(2000000000)  // 2 gwei tip (priority fee)
	auth.GasFeeCap = big.NewInt(20000000000) // 20 gwei max fee (base + priority)

	return auth, nil
}

// CreateDeploymentOpts creates transaction options specifically for contract deployment
func CreateDeploymentOpts(chainID *big.Int, client *ethclient.Client, account loadtypes.Account, nonce uint64) (*bind.TransactOpts, error) {
	// For deployment, use the account's current nonce (don't fetch from blockchain)
	// This allows sequential deployments with incrementing nonces
	return createTransactOpts(chainID, account, 3000000, nonce, false) // 3M gas limit for deployment
}

// CreateTransactionOpts creates transaction options for regular contract interactions
func CreateTransactionOpts(chainID *big.Int, scenario *loadtypes.TxScenario) *bind.TransactOpts {
	opts, err := createTransactOpts(chainID, scenario.Sender, 200000, scenario.Nonce, true) // 200k gas limit for transactions
	if err != nil {
		panic("Failed to create transaction options: " + err.Error())
	}
	return opts
}
