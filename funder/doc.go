// Package funder funds seiload's generated account pool from a single root key
// so a load run can execute against a real chain.
//
// # Why
//
// seiload generates random EVM accounts and never funds them. That works
// against mock chains, where mock_balances auto-tops every EVM account's
// balance at execution, or a fresh genesis that pre-funds them — but against a
// real, long-running chain (e.g. arctic-1) the accounts start at zero and every
// transfer reverts for lack of gas. This package gives the pool a balance
// before the run starts. When cfg.Funding is nil the package is inert and
// seiload's mock/genesis behavior is unchanged.
//
// # Flow
//
// FundAccounts runs once at startup, after the generator and sender are built
// and before prewarm and dispatch (both spend gas the accounts don't have until
// funded):
//
//  1. Resolve the root key (rootKeyFile, preferred; or rootKeyEnv).
//  2. Dial the EVM RPC and enumerate every account across the pools.
//  3. Skip accounts already at/above the target (a cheap, bounded, cancellable
//     concurrent balance check).
//  4. Deploy a fresh Disperse contract.
//  5. disperseEther the per-account amount to the underfunded set, in batches.
//
// # Cosmos to EVM association
//
// The root is a single secp256k1 key with both a cosmos (sei1) and an EVM (0x)
// representation. Its usei must be EVM-spendable, which on Sei requires the
// account to be associated. The Disperse deploy is the root's first EVM tx, and
// the Sei ante handler auto-associates the sender on its first EVM tx — pulling
// the cosmos balance to the EVM side within that tx. So no explicit association
// step is needed, provided the root is funded at its EVM (cast) address or is
// already associated. Recipients receive native value via the Disperse
// contract, which credits their EVM balance directly; each self-associates on
// its own first load tx.
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
