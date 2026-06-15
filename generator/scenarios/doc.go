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
// # StorageRW
//
// StorageRW exercises the SLOAD + SSTORE storage path under load against
// StorageRWv1. PLT-465 turns it into the two customer-named axes: key contention
// and tx size. Per tx the scenario draws a slot from the key distribution over
// the configured RecordCount keyspace, an operation (read/write/rmw) from the
// configured mix, and a calldata-pad length from the size distribution over the
// configured SizeBuckets histogram. The three draws ride independent rng
// sub-streams (dist:i:key, dist:i:op, dist:i:size) so tuning any one axis leaves
// the others' sequences identical. Each field is nil-guarded exactly like the
// gas pickers: with no distribution config the scenario reproduces the PLT-461
// scaffold byte-for-byte — single slot 0, empty pad, rmw.
//
// Gas sizing. The rmw is an SLOAD + SSTORE on a single slot: ~26k gas warm, but
// ~44k on a cold first touch (the cold-SLOAD and the zero-to-nonzero SSTORE both
// charge their higher rates). The base GasLimit is pinned to 50k, which covers
// the cold-first-touch case with headroom for the fixed calldata head and packs
// roughly 4x denser than the 200k default in CreateTransactionOpts. Density
// matters on a gas-limit-admission chain, where a block admits transactions up
// to its gas limit regardless of gas actually used — an oversized limit reserves
// block space the rmw never spends and throttles achievable throughput. The
// distribution-driven pad adds its own intrinsic calldata cost (4 gas per
// zero pad byte, EIP-2028) on top of the base so a large pad cannot
// underprovision the tx, without inflating the limit when the pad is empty.
package scenarios
