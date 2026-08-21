// Package generator turns a load profile into a stream of transactions.
//
// # Build order
//
// NewGenerator runs three steps once, at startup:
//
//  1. createScenarios — one scenario instance per config entry, each bound to an
//     account pool (its own, or the shared top-level pool).
//  2. deployAll — deploy the contract each instance needs, in sequence.
//  3. build — expand the instances by weight and shuffle them into the
//     round-robin the run draws from.
//
// An error in any step fails the run. A generator that cannot deploy has nothing
// valid to generate, so a failed deployment surfaces as a startup error rather
// than as a run that sends transactions to an address holding no contract.
//
// # The deployer is received, not minted
//
// deployAll signs its deployments with the account NewGenerator is handed.
// Paying for a deployment is a funding concern, and the funder package owns the
// run's funded identity, so funder.Deployer names the account and this package
// spends it. Minting a key here cannot work: no account pool holds it, so
// funding never reaches it, and a chain that charges for gas rejects every
// deployment it signs.
//
// # Deployment nonces
//
// A deployment leaves its nonce unset, so go-ethereum reads the deployer's
// pending nonce from the chain, and deployAll waits for the receipt before it
// sends the next one. This is what makes a deployer with on-chain history safe:
// the funding root has spent nonces before the run, and spends more right after
// these deployments when it funds the pool. A nonce derived from the instance
// index is correct only for a key that starts at zero. Deploying concurrently
// reintroduces the collision the sequence prevents; the funder package doc makes
// the same argument for the same key.
//
// # Mock deploy
//
// Under config.MockDeploy no deployment reaches a chain. Each instance attaches
// its binding at a random address, which is enough to shape calldata, and the
// deployer goes unused. This is the path --dry-run and the unit tests take.
package generator
