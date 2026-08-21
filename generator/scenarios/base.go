package scenarios

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	mrand "math/rand/v2"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/utils"
	"github.com/sei-protocol/sei-load/types"
	loadutils "github.com/sei-protocol/sei-load/utils"
)

// bigOne is 1 in big.Int.
var bigOne = big.NewInt(1)

// TxGenerator defines the interface for generating transactions.
type TxGenerator interface {
	Name() string
	Generate(rng *mrand.Rand, scenario *types.TxScenario) (*ethtypes.Transaction, error)
	Attach(config *config.LoadConfig, address common.Address) error
	Deploy(ctx context.Context, config *config.LoadConfig, deployer types.Account) (common.Address, error)
}

// ScenarioDeployer defines the interface for scenario-specific deployment logic
// This can be implemented by both contract and non-contract scenarios
type ScenarioDeployer interface {
	// DeployScenario handles any setup required for the scenario.
	// For contracts: deploys the contract from deployer's account and returns its
	// address, erroring if the deployment does not mine successfully.
	// For non-contracts: performs any initialization and returns zero address.
	DeployScenario(ctx context.Context, config *config.LoadConfig, deployer types.Account) (common.Address, error)

	// AttachScenario connects to an existing contract.
	AttachScenario(config *config.LoadConfig, address common.Address) common.Address

	// CreateTransaction creates a transaction for this scenario
	CreateTransaction(rng *mrand.Rand, config *config.LoadConfig, scenario *types.TxScenario) (*ethtypes.Transaction, error)
}

// ContractBindFunc defines a function that creates a contract instance from an address
type ContractBindFunc[T any] func(address common.Address, backend bind.ContractBackend) (*T, error)

// ContractDeployer defines the interface for contract-specific deployment logic
// This extends ScenarioDeployer with contract-specific methods
type ContractDeployer[T any] interface {
	ScenarioDeployer

	// DeployContract deploys the contract with specific constructor arguments
	DeployContract(opts *bind.TransactOpts, client *ethclient.Client) (common.Address, *ethtypes.Transaction, error)

	// GetBindFunc returns the function to bind the contract instance
	GetBindFunc() ContractBindFunc[T]

	// SetContract stores the bound contract instance
	SetContract(contract *T)

	// CreateContractTransaction creates a contract interaction transaction
	CreateContractTransaction(rng *mrand.Rand, auth *bind.TransactOpts, scenario *types.TxScenario) (*ethtypes.Transaction, error)
}

// ScenarioBase provides common functionality for all scenarios
type ScenarioBase struct {
	config   *config.LoadConfig
	deployed bool
	address  common.Address
	deployer ScenarioDeployer

	scenarioConfig config.Scenario
}

// NewScenarioBase creates a new base scenario with the given deployer
func NewScenarioBase(deployer ScenarioDeployer, cfg config.Scenario) *ScenarioBase {
	return &ScenarioBase{
		deployer:       deployer,
		scenarioConfig: cfg,
	}
}

// Deploy handles the common deployment flow. A scenario whose deployment fails
// stays undeployed, so Generate reports it rather than shaping transactions
// against an address that holds no contract.
func (s *ScenarioBase) Deploy(ctx context.Context, config *config.LoadConfig, deployer types.Account) (common.Address, error) {
	s.config = config
	address, err := s.deployer.DeployScenario(ctx, config, deployer)
	if err != nil {
		return common.Address{}, err
	}
	s.address = address
	s.deployed = true
	return s.address, nil
}

// Attach connects to an existing contract.
func (s *ScenarioBase) Attach(config *config.LoadConfig, address common.Address) error {
	s.config = config
	s.address = s.deployer.AttachScenario(config, address)
	s.deployed = true
	return nil
}

// Generate handles the common transaction generation flow
func (s *ScenarioBase) Generate(rng *mrand.Rand, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	if !s.deployed {
		return nil, fmt.Errorf("scenario not deployed/initialized")
	}
	// Create transaction using scenario-specific logic
	return s.deployer.CreateTransaction(rng, s.config, scenario)
}

// GetConfig returns the configuration
func (s *ScenarioBase) GetConfig() *config.LoadConfig {
	return s.config
}

// GetAddress returns the deployed contract address (zero address for non-contract scenarios)
func (s *ScenarioBase) GetAddress() common.Address {
	return s.address
}

// ContractScenarioBase provides common functionality for contract scenarios
type ContractScenarioBase[T any] struct {
	*ScenarioBase
	deployer ContractDeployer[T]
}

// NewContractScenarioBase creates a new base scenario with the given contract deployer
func NewContractScenarioBase[T any](deployer ContractDeployer[T], cfg config.Scenario) *ContractScenarioBase[T] {
	base := &ContractScenarioBase[T]{deployer: deployer}
	base.ScenarioBase = NewScenarioBase(base, cfg)
	return base
}

func dial(config *config.LoadConfig) (*ethclient.Client, error) {
	if len(config.Endpoints) == 0 {
		return ethclient.NewClient(nil), nil
	}
	return ethclient.Dial(config.Endpoints[0])
}

// AttachScenario implements AttachScenario interface for contract scenarios
func (c *ContractScenarioBase[T]) AttachScenario(config *config.LoadConfig, address common.Address) common.Address {
	client, err := dial(config)
	if err != nil {
		panic("Failed to connect to Ethereum client: " + err.Error())
	}

	// Bind contract instance using the provided bind function
	bindFunc := c.deployer.GetBindFunc()
	contract, err := bindFunc(address, client)
	if err != nil {
		panic("Failed to bind contract: " + err.Error())
	}

	// Store the contract instance
	c.deployer.SetContract(contract)
	return address
}

// deployTimeout bounds one deployment end to end: the nonce fetch, the send, and
// the wait for the receipt. Nothing else bounds startup, so a chain that never
// mines the transaction would hold the run open.
const deployTimeout = 30 * time.Second

// DeployScenario implements ScenarioDeployer interface for contract scenarios.
// It bounds the deployment at deployTimeout and reports an expiry of that budget
// without a context sentinel in the error chain: main reads those sentinels as a
// clean shutdown, so a deployment that never mined would otherwise be reported
// as a successful run that did nothing. A sentinel from the caller's own context
// is passed through, because that one really is a shutdown.
func (c *ContractScenarioBase[T]) DeployScenario(ctx context.Context, config *config.LoadConfig, deployer types.Account) (common.Address, error) {
	var address common.Address
	err := loadutils.WithinBudget(ctx, deployTimeout, "deployment", func(ctx context.Context) error {
		var err error
		address, err = c.deployWithin(ctx, config, deployer)
		return err
	})
	return address, err
}

func (c *ContractScenarioBase[T]) deployWithin(ctx context.Context, config *config.LoadConfig, deployer types.Account) (common.Address, error) {
	client, err := dial(config)
	if err != nil {
		return common.Address{}, fmt.Errorf("dial: %w", err)
	}

	auth, err := utils.CreateDeploymentOpts(ctx, config.GetChainID(), deployer)
	if err != nil {
		return common.Address{}, fmt.Errorf("deployment options for %s: %w", deployer.Address.Hex(), err)
	}

	address, tx, err := c.deployer.DeployContract(auth, client)
	if err != nil {
		return common.Address{}, fmt.Errorf("send deployment from %s: %w", deployer.Address.Hex(), err)
	}
	log.Printf("📤 Deployment transaction sent: %s", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return common.Address{}, fmt.Errorf("wait for deployment %s: %w", tx.Hash().Hex(), err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return common.Address{}, failedDeployment(ctx, client, tx, receipt)
	}
	log.Printf("✅ Deployment successful at block %d (gas used: %d)", receipt.BlockNumber.Uint64(), receipt.GasUsed)

	contract, err := c.deployer.GetBindFunc()(address, client)
	if err != nil {
		return common.Address{}, fmt.Errorf("bind contract at %s: %w", address.Hex(), err)
	}
	c.deployer.SetContract(contract)
	return address, nil
}

// failedDeployment reports a deployment that mined with a failed status. It
// replays the transaction with eth_call and asks the node for the transaction
// error, so the message carries the revert reason when the node offers one.
func failedDeployment(
	ctx context.Context,
	client *ethclient.Client,
	tx *ethtypes.Transaction,
	receipt *ethtypes.Receipt,
) error {
	msg := fmt.Sprintf(
		"deployment transaction failed with status %d (tx: %s, block: %d, gas used: %d/%d, contract: %s)",
		receipt.Status,
		tx.Hash().Hex(),
		receipt.BlockNumber.Uint64(),
		receipt.GasUsed,
		tx.Gas(),
		receipt.ContractAddress.Hex(),
	)

	if receipt.GasUsed == tx.Gas() {
		msg += " [gas used hit gas limit]"
	}

	callMsg := ethereum.CallMsg{
		From:  deployerAddress(tx),
		To:    tx.To(),
		Gas:   tx.Gas(),
		Value: tx.Value(),
		Data:  tx.Data(),
	}
	if tx.Type() == ethtypes.AccessListTxType {
		callMsg.AccessList = tx.AccessList()
	}
	if tx.Type() == ethtypes.LegacyTxType || tx.Type() == ethtypes.AccessListTxType {
		callMsg.GasPrice = tx.GasPrice()
	} else {
		callMsg.GasFeeCap = tx.GasFeeCap()
		callMsg.GasTipCap = tx.GasTipCap()
	}

	prevBlock := new(big.Int).Sub(receipt.BlockNumber, common.Big1)
	if prevBlock.Sign() < 0 {
		prevBlock = big.NewInt(0)
	}
	if _, err := client.CallContract(ctx, callMsg, prevBlock); err != nil {
		msg += fmt.Sprintf(" [eth_call replay: %v]", err)
	}
	if txErr := fetchTransactionErrorByHash(ctx, client, tx.Hash()); txErr != "" {
		msg += fmt.Sprintf(" [eth_getTransactionErrorByHash: %s]", txErr)
	}

	return errors.New(msg)
}

func deployerAddress(tx *ethtypes.Transaction) common.Address {
	signer := ethtypes.LatestSignerForChainID(tx.ChainId())
	from, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return common.Address{}
	}
	return from
}

func fetchTransactionErrorByHash(ctx context.Context, client *ethclient.Client, hash common.Hash) string {
	rpcClient := client.Client()
	if rpcClient == nil {
		return ""
	}

	var result string
	if err := rpcClient.CallContext(ctx, &result, "eth_getTransactionErrorByHash", hash); err != nil {
		return fmt.Sprintf("rpc error: %v", err)
	}
	return result
}

// CreateTransaction implements ScenarioDeployer interface for contract scenarios
func (c *ContractScenarioBase[T]) CreateTransaction(rng *mrand.Rand, config *config.LoadConfig, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	auth := utils.CreateTransactionOpts(config.GetChainID(), scenario)
	return c.deployer.CreateContractTransaction(rng, auth, scenario)
}
