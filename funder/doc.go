// Package funder owns seiload's funded identity: it resolves the root key and
// spends it, both to fund the generated account pool and to name the account
// that signs the run's contract deployments, so a load run can execute against
// a real chain.
//
// # Why
//
// seiload generates random EVM accounts and never funds them. That works
// against mock chains, where mock_balances auto-tops every EVM account's
// balance at execution, or a fresh genesis that pre-funds them — but against a
// real, long-running chain (e.g. arctic-1) the accounts start at zero and every
// transfer reverts for lack of gas. This package gives the pool a balance
// before the run starts. When cfg.Funding is nil the package is inert.
//
// # Flow
//
// FundAccounts runs once at startup, after the generator is built and before
// prewarm and dispatch (both spend gas the accounts don't have until funded):
//
//  1. Resolve the root key (rootKeyFile, preferred; or rootKeyEnv).
//  2. Dial the EVM RPC and enumerate every account across the pools.
//  3. Skip accounts already at/above the target (a cheap, bounded, cancellable
//     concurrent balance check).
//  4. Deploy a fresh Disperse contract.
//  5. disperseEther the per-account amount to the underfunded set, in batches.
//
// # The contract deployer
//
// Deployer names the account that signs the run's contract deployments, and the
// generator receives it (see the generator package doc). With funding configured
// that account is the root: paying for a deployment is a funding concern, and
// the root is the one key a run knows to hold a balance. Without funding it is a
// fresh random account, payable only by a chain that credits unknown senders.
// Scenario contracts are deployer-neutral — none of them gates a load
// transaction on who deployed it — so the choice costs the workload nothing.
//
// # One nonce stream
//
// When the root is also the deployer, the scenario deployments and this
// package's Disperse deployment plus disperseEther batches are one EVM nonce
// stream on one key. Both phases run on the startup goroutine, in sequence, and
// each awaits its receipt before it sends the next tx, so every tx reads a
// pending nonce that already counts the one before it. Overlapping the phases,
// or pinning auth.Nonce in either, collides them.
//
// # Cosmos to EVM association
//
// The root is a single secp256k1 key with both a cosmos (sei1) and an EVM (0x)
// representation. Its usei must be EVM-spendable, which on Sei requires the
// account to be associated. The Sei ante handler auto-associates the sender on
// its first EVM tx — pulling the cosmos balance to the EVM side within that tx
// — and the root's first EVM tx of a run is a deploy: a scenario contract when
// the profile has one, else Disperse. So no explicit association step is
// needed, provided the root is funded at its EVM (cast) address or is already
// associated. Recipients receive native value via the Disperse contract, which
// credits their EVM balance directly; each self-associates on its own first
// load tx.
//
// # Self-deploy, not a configured address
//
// The Disperse contract is always deployed fresh rather than bound from a
// configured address. A configurable contract address would be a value-bearing
// call from the highest-value key in the system to an unverified target; always
// deploying — and asserting the deployed code is non-empty — removes that
// footgun.
//
// # Sequential funding and nonce ordering
//
// The batch loop is deliberately sequential: auth.Nonce stays nil so go-ethereum's
// bind fetches PendingNonceAt per tx, and each batch is awaited (WaitMined plus a
// receipt-status assertion) before the next is sent, so the prior nonce is mined
// and visible first. Parallelizing batches or setting auth.Nonce would
// reintroduce nonce-collision races.
//
// # Idempotency and restarts
//
// Funding targets the current pool. seiload generates a fresh random pool on
// every start, so a pod restart funds a new set of accounts; the prior set's
// balances are stranded. That is acceptable on a funny-money devnet and bounded
// by the root balance. The already-funded skip guards against double-funding
// within a single run, not across restarts.
package funder
