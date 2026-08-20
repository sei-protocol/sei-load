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
// read writes too, and that bounds what the key axis can show. Every read folds
// its load into readAccumulator — one contract-wide slot — so reads conflict with
// each other no matter which key they drew. A read-weighted mix therefore does
// not sweep contention; only rmw and write do. Reads also measure absent slots
// until something has written them, since a fresh deploy starts empty and there
// is no warm-up phase.
//
// Gas sizing. All three operations share one base GasLimit of 50k. The measured
// worst case is a read that first writes readAccumulator, at 46,269 including
// intrinsic cost; rmw and write cold-first-touch sit near 44k. So 50k clears all
// three, but by only ~3.7k — and SSTORE_SET is a Sei governance parameter
// (SeiSstoreSetGasEip2200, default 20,000), so a raise past roughly 23.7k would
// put read out of gas. Widening the keyspace also makes cold first touches the
// normal case rather than the exception, which is the regime this headroom has to
// survive.
//
// One limit for all three trades slack on the cheaper operations for a single
// number to reason about. Density is why the number is tight at all: it packs
// roughly 4x denser than the 200k default in CreateTransactionOpts, and on a
// gas-limit-admission chain a block admits transactions up to their declared
// limit regardless of gas actually used, so an oversized limit reserves block
// space the transaction never spends and throttles achievable throughput.
//
// The drawn pad is charged at 10 gas per on-wire byte on top of the base. That is
// the EIP-7623 floor rate, which is live on Sei, and it is the binding cost above
// roughly 4.6 KiB of pad. Sei's ante checks only the intrinsic cost, so a limit
// sized to the older 4-gas rate is admitted, reserves its full limit, then fails
// in execution with GasUsed equal to the limit — an included failure that
// inflates the gas-used metric the run reports. An empty pad leaves the limit at
// exactly 50k.
package scenarios
