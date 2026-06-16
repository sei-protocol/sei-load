# Workload Model

> [← AGENTS.md index](../AGENTS.md)

> What this covers: the scenario set sei-load can generate, what each one stresses, and the StorageRW contention/size/op-mix knobs that let an agent dial conflict and tx size. When an agent needs it: designing an experiment — choosing a scenario and configuring the axes that produce the load shape under test.

> ⚠️ **Requires PLT-465 (#54, unmerged as of writing).** The StorageRW per-tx axes described below — `keyDistribution`, `sizeDistribution`, `sizeBuckets`, `recordCount`, and `operations` op-mix sampling — **parse but do not affect generated transactions on main**. They only shape txs once PLT-465 lands. On main today StorageRW emits a fixed scaffold (single slot 0, empty pad, all-`rmw`). Treat the axis sections as the PLT-465 interface, not current behavior.

Each scenario is a `TxGenerator` resolved by lowercase `name` through the factory (`generator/scenarios/factory.go`). The `name` you put in a scenario config is one of the registered keys below. Unknown names panic at startup.

## Scenario set

Registered names (from `scenarioFactories`, `generator/scenarios/factory.go:13`):

| `name` | Contract | Per-tx action | What it stresses |
|--------|----------|---------------|------------------|
| `evmtransfer` | none | Native value transfer, `value = now().Unix()`, gas 21000 | Baseline native-path throughput; signature recovery + balance update. Value varies per tx (non-zero). |
| `evmtransferfast` | none | Native transfer, fixed `value = 1e12`, **zero tip** | Same as above with constant value and no priority fee — cheapest native baseline. (Registered name is also `evmtransfer` via `Name()`; distinct factory key `evmtransferfast`.) |
| `evmtransfernoop` | none | Self-transfer, `value = 0`, gas 21000 | Native path with no balance delta — isolates execution overhead from state change. |
| `erc20` | ERC20 | `transfer(to, 1)`, gas 72156 | Real ERC20 SSTORE path: two balance-slot writes per transfer. Distinct sender/receiver slots → low cross-tx conflict. |
| `erc20noop` | ERC20Noop | `transfer(to, 1)`, gas 22460 | ERC20 ABI surface with a no-op body — measures dispatch/calldata cost without the storage writes. |
| `erc20conflict` | ERC20Conflict | `transfer(to, 1)`, gas 22460 | ERC20 variant engineered so transfers contend on shared state → drives the parallel executor's conflict path via an ERC20 shape. |
| `erc721` | ERC721 | `mint(to, id)`, monotonic `id` (atomic counter), gas 22460 | NFT mint path: fresh-slot SSTORE per tx plus a contended counter. |
| `disperse` | Disperse | `disperseEtherFixed(targets)` to 100 fresh accounts/tx | Fan-out: one tx touching 100 distinct recipient accounts. Account-creation heavy. |
| `storagerw` | StorageRWv1 | `read`/`write`/`rmw` against a caller-chosen slot, with calldata pad | **The tunable axis scenario.** SLOAD+SSTORE storage path with configurable key contention, tx size, and op mix. See below. |

Gas limits above are the per-tx `GasLimit` each scenario pins (`CreateContractTransaction` / `CreateTransaction`). On a gas-limit-admission chain the limit — not gas used — reserves block space, so these are sized tight. A `gasPicker` in config overrides the native-transfer gas; contract scenarios pin their own limit.

All contracts compile to the **`paris`** EVM target (solc 0.8.19, `Makefile:39`). `paris ⊂ Sei`'s active fork, so bytecode is unconditionally safe to deploy; runtime gas is set by the chain's live fork regardless of compile target (`Makefile:31-38`).

## StorageRW: the two axes

StorageRW (`generator/scenarios/StorageRW.go`) is the scenario built for parametric conflict/size experiments. Per tx it makes three **independent** draws — slot (key contention), pad length (tx size), and operation (op mix) — each on its own seeded RNG sub-stream, then builds a `read`/`write`/`rmw` call against `StorageRWv1`.

> ⚠️ **Requires PLT-465 (#54, unmerged as of writing) — on main these fields parse but do not affect generated transactions.** The per-tx slot, op, and pad axes are delivered by PLT-465 (not yet on main). On main, StorageRW is a scaffold: every tx is a fixed-slot-0, empty-pad `rmw` (`generator/scenarios/doc.go` "StorageRW scaffold"). What follows is the PLT-465 interface.

The contract `StorageRWv1` (`generator/contracts/StorageRWv1.sol`) is mapping-backed (`mapping(uint256 => uint256) store`) with no fixed keyspace — the slot index is caller-chosen, so the keyspace resizes with config and never needs a redeploy. `read` folds the load into `readAccumulator` so the SLOAD is non-elidable; `rmw` does `store[slot] += 1`; `write` sets `store[slot] = 1`. All use `unchecked` arithmetic so no tx ever reverts on overflow.

**Defaults (nil-guarded, the 100%-conflict baseline):** with no `keyDistribution`/`recordCount`, every tx hits fixed slot 0 (`pickSlot` — PLT-465 branch, not on main). With no `sizeDistribution`/`sizeBuckets`, the pad is empty (`pickPad` — PLT-465 branch, not on main). With no `operations`, every tx is `rmw` (`pickOp` — PLT-465 branch, not on main). So bare `{"name":"storagerw"}` = single-slot, empty-pad, all-rmw = maximum contention. (On main this is the *only* behavior — see the banner; the scaffold is unconditionally fixed-slot-0, empty-pad, `rmw`.)

### Axis 1 — KEY CONTENTION

The slot each tx touches is `keyDistribution.SampleIndex(recordCount)` — a draw in `[0, recordCount)` (PLT-465 branch, not on main). Contention is the probability that two txs in the same block draw the same slot.

- **Keyspace size** = `recordCount`. Larger → lower collision probability at fixed distribution.
- **Distribution** = `keyDistribution`: `uniform` (flat) or `zipfian` with `theta` in `[0, 1)`.
  - `theta → 0`: approaches uniform. Over a large keyspace, collision ≈ 0% (`config/doc.go:28-36`).
  - `theta → 1`: draws concentrate on low indices → a hotspot. `theta` is validated to `[0, 1)`; `alpha = 1/(1-theta)` diverges at 1 (`distribution.go:163`).
  - `recordCount = 0` (or no `keyDistribution`): single slot 0 = **100% conflict**.

To set X contention, configure:

```jsonc
// ~0% conflict: uniform over a large keyspace
{ "name": "storagerw",
  "keyDistribution": {"Name": "uniform"},
  "recordCount": 1000000 }

// moderate hotspot: zipfian, low indices favored
{ "name": "storagerw",
  "keyDistribution": {"Name": "zipfian", "theta": 0.9},
  "recordCount": 1000000 }

// 100% conflict: single slot (omit key config)
{ "name": "storagerw" }
```

Verified on the PLT-465 branch: `TestStorageRWContentionSweep` (not on main) pins both ends — uniform over 1e6 with 2000 draws is >99% distinct slots; default config is always slot 0.

Note `recordCount` is the keyspace the distribution **indexes into**, not a count of distinct slots that will be touched in a run. Actual collision in a single block is a function of `recordCount`, distribution shape, and how many StorageRW txs land in that block (i.e. your rate ÷ block production).

### Axis 2 — TX SIZE

Each tx carries a zero-filled calldata pad whose length is `sizeBuckets[sizeDistribution.SampleIndex(len(sizeBuckets))]` (`pickPad` — PLT-465 branch, not on main). The pad is an ignored `bytes _pad` argument on every method — it varies tx size without touching the storage logic.

- `sizeBuckets`: the histogram of candidate pad lengths in bytes, e.g. `[0, 64, 256, 1024]`. Each entry capped at 1 MiB (`config.go`).
- `sizeDistribution`: `uniform` or `zipfian`, selects which bucket index per tx.
- **Gas:** the pad's intrinsic cost is `4 gas per zero byte` (the base calldata gas schedule for zero bytes — this rate predates and is unchanged by EIP-2028, which only lowered the *non-zero* byte cost from 68→16) added on top of the 50k base: `GasLimit = 50000 + len(pad)*4` (PLT-465 branch, not on main). A larger pad → larger tx → more calldata gas, scaling block-space consumption per tx.

```jsonc
{ "name": "storagerw",
  "keyDistribution": {"Name": "uniform"}, "recordCount": 1000000,
  "sizeDistribution": {"Name": "uniform"},
  "sizeBuckets": [0, 64, 256, 1024] }
```

**Independence (load-bearing):** the size draw rides sub-stream `dist:%d:size`, distinct from the key sub-stream `dist:%d:key` (`utils/rng/streams.go` — both stream IDs are frozen and present on main). Changing the size config never perturbs the key sequence — verified on the PLT-465 branch by `TestStorageRWKeySizeIndependence` (not on main): same seed + same key config yields an identical slot sequence with and without a size distribution. This lets an agent sweep one axis while holding the other's draw multiset fixed.

### Axis 3 — OP MIX

`operations` weights the read/write/rmw selection (`config/operation.go` — PLT-465 branch, not on main). Weights are relative; a per-tx draw picks in proportion to weight over total. Nil or all-zero → all `rmw` (the default, since `OpRmw` is the zero value).

```jsonc
{ "name": "storagerw",
  "operations": {"read": 1, "write": 1, "rmw": 2} }
```

What each op does to conflict: `read` is an SLOAD (folded into `readAccumulator`); `write` and `rmw` are SSTOREs. Two reads of the same slot do **not** conflict under OCC (no write); a read+write or write+write on the same slot **does**. So op mix and key contention compose: a high-`theta` keyspace with all-`read` exhibits far less executor conflict than the same keyspace with all-`rmw`. The op draw rides its own sub-stream `dist:%d:op` — **a PLT-465-future stream ID, NOT one of the streams frozen on main** (main's frozen set is the 8 IDs in [05-reproducibility §Stream IDs that exist](05-reproducibility.md#stream-ids-that-exist); `dist:%d:op` is added only by PLT-465). Verified independent of the key sequence on the PLT-465 branch by `TestStorageRWOpIndependence` (not on main).

## What these axes actually probe on Sei

> This section is domain reasoning about Sei's execution model layered on top of what the code generates. Where a claim is about sei-load code it is cited; where it is about Sei node behavior it is flagged as REASONED — confidence noted. Validate node-side claims against the SUT's own metrics.

**Sei is a parallel-EVM chain with optimistic concurrency control (Block-STM-style).** Transactions in a block are executed speculatively in parallel; a read-set/write-set validation pass detects when one tx read a slot another tx wrote, and re-executes the loser serially. (REASONED — this is the documented Sei/Block-STM design; confidence: high on the mechanism class, medium on exact scheduler details which vary by sei-chain version.)

**Key contention exercises the conflict/abort-and-re-execute path.** When many txs in one block draw the same `store[slot]` and at least one writes it, the optimistic schedule's validation fails for the conflicting txs and they re-execute. As contention rises (smaller `recordCount`, higher `theta`, or single-slot default), the hot slot's throughput degrades toward **serial** as the conflicting write-set fraction → 1 (for that hot slot) — the parallel executor cannot retire conflicting writers concurrently. Throughput for the hot slot is bounded by serialized re-execution, not by parallel width. (REASONED; confidence: high — this is the defining behavior of OCC under write contention.)

**Contrast with a DynamoDB-style hot shard — different mechanism, same observable.** A DynamoDB hot partition degrades because the partition has a fixed WCU/RCU budget and excess requests are **throttled** (a storage-capacity/rate limit). Sei has **no per-key throughput cap**. The limit on a hot slot is *execution-conflict serialization*: the slot can be written as fast as the executor can run the conflicting txs back-to-back, but those txs cannot run *in parallel*. Same surface symptom (hot key → throughput plateaus), fundamentally different cause (OCC re-execution vs. provisioned-capacity throttling). An agent must not interpret a StorageRW hot-slot plateau as a storage-rate limit — there is no quota to raise; the cure is reducing conflict (spread the keyspace) or accepting serial throughput for that slot. (REASONED; confidence: high.)

**Node-side signal to watch:** Block-STM conflict / abort / re-execution rate. On Sei this surfaces (when exposed by the SUT) as `sei_occ_*` metrics. (REASONED — the metric family name is the expected Sei convention; confidence: medium. Confirm the exact series exposed by the node version under test before relying on them; the SUT may not export them at all.) The generator-side signal is unambiguous: you control conflict probability via `recordCount` + `theta` + op mix, and those draws are deterministic for a given seed.

**Gas-model interplay.** The calldata pad (Axis 2) adds `4 gas per zero byte` (the base calldata gas schedule; PLT-465 branch, not on main), so larger txs consume proportionally more block gas and admit fewer txs/block on a gas-limit-admission chain. Size and contention are orthogonal stressors: size limits *how many* txs fit a block; contention limits *how many of those can execute in parallel*. Sweeping both maps the throughput surface. (Code-grounded for the gas formula; the admission behavior is REASONED, confidence: high — consistent with the package doc's "gas-limit-admission" rationale, `generator/scenarios/doc.go`.)

**EVM version.** Contracts target `paris` (solc 0.8.19), a strict subset of Sei's active fork — safe on Sei, and compile target does not distort runtime gas (`Makefile:31-39`). VERIFIED.

## See also

- [03-config-reference](03-config-reference.md) — full Scenario/Distribution JSON schema.
- [06-measurement-metrics](06-measurement-metrics.md) — the counters to read when interpreting a run.
- [07-experiment-playbook](07-experiment-playbook.md) — putting axes together into a sweep.
- [08-limits-boundaries](08-limits-boundaries.md) — measurement boundaries that bound how to read results.
