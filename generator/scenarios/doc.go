// Package scenarios defines the load scenarios seiload can generate, and the
// shared scaffolding that lets a contract-backed scenario describe only what is
// unique to its contract.
//
// # The contract-scenario pattern
//
// Every scenario satisfies TxGenerator (Name/Generate/Attach/Deploy). Non-contract
// scenarios (the EVMTransfer family) implement it directly; contract scenarios
// compose ContractScenarioBase[T], which factors out the deploy-wait-bind flow
// and the per-tx auth construction so the concrete scenario only supplies its
// contract specifics.
//
// A contract scenario embeds *ContractScenarioBase[T] (T being the generated
// binding) and implements ContractDeployer[T]:
//
//   - DeployContract — deploy the contract for this run.
//   - GetBindFunc    — return the binding's constructor so the base can bind the
//     deployed (or attached) address.
//   - SetContract    — receive the bound instance for later CreateContractTransaction
//     calls.
//   - CreateContractTransaction — build one load transaction against the contract.
//
// The base owns the rest: DeployScenario deploys, waits for the receipt, asserts
// success, then binds and hands back the instance via SetContract; AttachScenario
// binds an already-deployed address the same way; CreateTransaction builds the
// per-tx auth and delegates to CreateContractTransaction.
//
// # MockDeploy attach
//
// Under config.MockDeploy a scenario attaches to a known address without a live
// endpoint, so the bind backend is nil. This is the path the tests and
// generator.mockDeployAll exercise: bind at an address, produce calldata, but
// never send. CreateContractTransaction must therefore stay pure (it shapes a
// transaction; it does not touch the chain).
//
// # Factory registration
//
// scenarioFactories maps a lowercase scenario name to its constructor, and
// CreateScenario resolves a config.Scenario by name. Non-contract entries are
// hand-written; contract entries below the AUTO-GENERATED marker in factory.go
// are emitted by `make generate` from the contract bindings — do not edit that
// block by hand.
//
// # StorageRW scaffold
//
// StorageRW issues a read-modify-write against StorageRWv1 to exercise the SLOAD
// + SSTORE storage path under load. PLT-461 lands it as a scaffold: every
// transaction targets one fixed slot with an empty calldata pad, which is enough
// to prove the deploy/send path. The per-tx slot/value/pad distribution arrives
// in PLT-465.
//
// Gas sizing. The rmw is an SLOAD + SSTORE on a single slot: ~26k gas warm, but
// ~44k on a cold first touch (the cold-SLOAD and the zero-to-nonzero SSTORE both
// charge their higher rates). The scaffold pins GasLimit to 50k: it covers the
// cold-first-touch case with headroom for the (currently empty) pad, and packs
// roughly 4x denser than the 200k default in CreateTransactionOpts. Density
// matters on a gas-limit-admission chain, where a block admits transactions up
// to its gas limit regardless of gas actually used — an oversized limit reserves
// block space the rmw never spends and throttles achievable throughput. PLT-465
// revisits the limit once the calldata pad is distribution-driven, since pad size
// changes calldata gas.
package scenarios
