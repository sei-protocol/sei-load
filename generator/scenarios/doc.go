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
// StorageRWv1. Per tx it draws three things: a slot from the key distribution
// over the RecordCount keyspace, a pad length from the size distribution over
// the SizeBuckets histogram, and an operation — read, write, or rmw — from the
// Operations mix. Slot and pad are the two customer-named axes, key contention
// and tx size; the operation mix shapes what each drawn slot is used for.
//
// Slot and keyspace are what make contention a continuum rather than a binary.
// A wide keyspace drawn uniformly approaches zero conflict; a single slot is
// total conflict; a zipfian draw sits anywhere between. Throughout this section
// record, key, and slot all name the same thing: an index in [0, RecordCount).
//
// Every axis is optional. Omit them all and StorageRW behaves exactly as it did
// before they existed: slot 0, an empty pad, rmw — the 100%-conflict baseline —
// and zero draws from the RNG. So adding these fields to a profile that does not
// set them cannot perturb its workload.
//
// Whichever draws a config does enable run in a fixed order: slot, then pad,
// then operation. That order must stay stable. The draws share the run's single
// PRNG, so reordering them shifts every subsequent draw and diverges any saved
// workload replayed at the same seed. Conversely, enabling or reweighting an
// axis changes how many draws each transaction takes, which shifts the other
// axes — see the config package doc on what the seed does and does not promise.
//
// Gas sizing. All three operations share one base GasLimit of 50k. The rmw is
// the costliest: an SLOAD plus an SSTORE on a single slot, ~26k warm and ~44k on
// a cold first touch, where the cold SLOAD and the zero-to-nonzero SSTORE both
// charge their higher rates. write is an SSTORE without the load, and read folds
// its load into readAccumulator, so both sit under the rmw ceiling — 50k covers
// the worst of the three with headroom for the fixed calldata head.
//
// One limit for all three trades a little slack on read and write for a single
// number to reason about. Density is why the number is tight at all: it packs
// roughly 4x denser than the 200k default in CreateTransactionOpts, and on a
// gas-limit-admission chain a block admits transactions up to its gas limit
// regardless of gas actually used, so an oversized limit reserves block space
// the transaction never spends and throttles achievable throughput.
//
// The drawn pad adds its own intrinsic calldata cost — 4 gas per zero pad byte,
// EIP-2028 — on top of the base, so a large pad cannot underprovision the tx
// while an empty pad leaves the limit at exactly 50k.
package scenarios
