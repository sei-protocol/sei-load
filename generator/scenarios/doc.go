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
// StorageRW exercises the SLOAD + SSTORE storage path under load. Each
// transaction draws three things independently from the scenario config: the
// storage slot from KeyDistribution over a RecordCount-wide keyspace, the
// calldata pad length from SizeDistribution over the SizeBuckets histogram, and
// the method — read, write, or rmw — from the Operations mix.
//
// Those three axes are what make contention a continuum rather than a binary. A
// wide keyspace drawn uniformly approaches zero conflict; a single slot is total
// conflict; a zipfian draw sits anywhere between.
//
// Every axis is optional and every default is the pre-distribution behavior: one
// fixed slot, an empty pad, and rmw. An unconfigured scenario draws no randomness
// at all, so adding these fields to a profile that does not use them cannot
// perturb its workload. The draws run in a fixed order — slot, pad, operation —
// because they share one RNG, which makes the order part of the reproducibility
// contract.
//
// Gas sizing. The rmw is an SLOAD + SSTORE on a single slot: ~26k gas warm, but
// ~44k on a cold first touch, where the cold SLOAD and the zero-to-nonzero SSTORE
// both charge their higher rates. The base limit is 50k, covering the
// cold-first-touch case with headroom, and the pad's intrinsic calldata cost is
// added on top per transaction so a large pad cannot underprovision. That packs
// roughly 4x denser than the 200k default in CreateTransactionOpts. Density
// matters on a gas-limit-admission chain, where a block admits transactions up to
// its gas limit regardless of gas actually used — an oversized limit reserves
// block space the transaction never spends and throttles achievable throughput.
package scenarios
