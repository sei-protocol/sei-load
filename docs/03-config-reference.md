# Config reference

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it: the complete JSON config schema for
> sei-load — top-level `LoadConfig`, the `settings` block (every field, type,
> default, and run effect), `scenarios`, `accounts`, and `funding` — with an
> annotated example and the field interactions that change a run's behavior. Read
> this when authoring or editing a config. For how to invoke the binary and the
> CLI-flag equivalents, see [02-running.md](02-running.md).

The config is a single JSON object parsed into `config.LoadConfig`. Every field is
optional except `endpoints` and `scenarios` (the loader rejects a config missing
either). Unknown fields are ignored, and **a config that uses no new fields runs
the legacy closed-loop path unchanged** — the schema is additive by construction.

## Top-level `LoadConfig`

| Field | JSON key | Type | Default | Meaning |
|-------|----------|------|---------|---------|
| ChainID | `chainId` | int64 | `0` | EVM chain ID used to sign transactions. Must match the target chain. |
| SeiChainID | `seiChainID` | string | `""` | Textual chain ID used to tag metrics and block/inclusion collectors. Key casing is cosmetic — `seiChainId` (lowercase `d`) also binds (see [gotcha](#gotchas)). |
| Endpoints | `endpoints` | []string | (required) | RPC endpoints. Workers shard across all of them; trackers read only `endpoints[0]`. |
| Accounts | `accounts` | object | none | Shared account pool (see [Accounts](#accounts)). |
| Scenarios | `scenarios` | []object | (required) | Weighted workload mix (see [Scenarios](#scenarios)). |
| Settings | `settings` | object | `DefaultSettings()` | Run knobs; CLI flags override (see [Settings](#settings)). |
| Funding | `funding` | object | none | Root-key funding of the account pool (see [Funding](#funding)). |
| Seed | `seed` | uint64 | random | Roots the deterministic PRNG. Same seed + config = same workload draw multiset (see [Seed](#seed-reproducibility)). |
| MockDeploy | `mockDeploy` | bool | `false` | Mock contract deploys. Auto-forced on under `--dry-run`; rarely set by hand. |
| ReportPath | `reportPath` | string | `""` | Alias also accepted at top level; `settings.reportPath` is the normal place. |

### Annotated example

```jsonc
{
  "chainId": 713715,                 // EVM chain id; must match the chain
  "seiChainID": "arctic-1",          // metric/collector tag (casing cosmetic; seiChainId also binds)
  "endpoints": [                     // workers shard across these; trackers use [0]
    "http://rpc-a:8545",
    "http://rpc-b:8545"
  ],
  "seed": 42,                        // optional; omit for a random (recorded) seed
  "accounts": {                      // shared pool unless a scenario overrides
    "count": 500,
    "newAccountRate": 0.0
  },
  "funding": {                       // optional; fund the pool from a root key
    "rootKeyFile": "/etc/seiload-key/root-key.hex",
    "fundAmountWei": "1000000000000000000",  // 1 SEI; decimal STRING (precision)
    "batchSize": 200
  },
  "scenarios": [
    { "name": "EVMTransfer", "weight": 7 },
    { "name": "ERC20",       "weight": 3 }
  ],
  "settings": {
    "workers": 50,
    "tps": 100,
    "arrivalModel": "open_loop",
    "maxInFlight": 5000,
    "statsInterval": "10s",
    "bufferSize": 1000,
    "trackReceipts": true,
    "inclusionReapAfter": "45s",
    "trackBlocks": true,
    "prewarm": true,
    "postSummaryFlushDelay": "25s",
    "reportPath": "/dev/stdout"
  }
}
```

## Settings

Every field below has a CLI-flag twin with the same default (see
[02-running.md](02-running.md#cli-flags)); CLI overrides config overrides default.
Duration fields are JSON strings parsed by Go's `time.ParseDuration` (e.g.
`"10s"`, `"45s"`, `"5m"`).

| Field (JSON key) | Type | Default | Effect on the run |
|------------------|------|---------|-------------------|
| `workers` | int | `1` | Tasks per endpoint. Total senders = workers × endpoints. (Struct field is `TasksPerEndpoint`.) |
| `tps` | float64 | `0` | Aggregate target rate via one shared limiter. `0` = unbounded. Required (>0) for open-loop unless `rampUp`. |
| `arrivalModel` | string | `"closed_loop"` | `"open_loop"` vs `"closed_loop"` — see [arrivalModel](#arrivalmodel). |
| `maxInFlight` | int | `10000` | **Open-loop only.** Cap on concurrent in-flight sends; overdue txs past the cap are dropped+counted, the arrival clock is never throttled. Ignored in closed-loop. |
| `statsInterval` | duration | `"10s"` | Stats-logging cadence; also the user-latency tracker tick. |
| `inclusionReapAfter` | duration | `"30s"` | Time an un-included tx waits before being reaped as **expired**. Only used when `trackReceipts` is on. Too short → real-but-slow inclusions counted expired; too long → inflated in-flight map. Also sizes the inclusion registry cap (≈ tps × reapAfter × 1.5, floored at maxInFlight × 4). |
| `bufferSize` | int | `1000` | Per-worker channel buffer. Larger = more in-memory queueing; lower under memory pressure. |
| `dryRun` | bool | `false` | Simulate without sending; forces `mockDeploy`; disables the inclusion tracker. |
| `debug` | bool | `false` | Log every transaction. Diagnostic/small runs only. |
| `trackReceipts` | bool | `false` | Enable the block-indexed inclusion tracker (included/expired/dropped-at-cap/inflight-at-shutdown). No-op under `dryRun` or with zero endpoints. Reads `endpoints[0]`. |
| `trackBlocks` | bool | `false` | Collect block time/gas stats from `endpoints[0]`. |
| `trackUserLatency` | bool | `false` | Per-user latency sampled at `statsInterval` from `endpoints[0]`. |
| `prewarm` | bool | `false` | Self-transaction prewarm before the main run; excluded from main stats. |
| `rampUp` | bool | `false` | Drive load with the built-in ramp curve. Supplies a finite λ to satisfy open-loop without a fixed `tps`. |
| `reportPath` | string | `""` | Write a **formatted text** report to this path (`/dev/stdout` valid); empty = none. Text dump today, not JSON — schema-versioned JSON is future work (PLT-467). |
| `txsDir` | string | `""` | Offline tx-writer mode: write generated txs to this dir instead of sending. Forces closed-loop (open-loop logged as ignored). |
| `targetGas` | uint64 | `10000000` | Target gas/block in tx-writer mode. |
| `numBlocksToWrite` | int | `100` | Blocks to write in tx-writer mode. |
| `postSummaryFlushDelay` | duration | `"25s"` | Post-summary sleep so Prometheus scrapes final metrics before exit. `0` = exit immediately (last scrape lost). |

> CLI-only (not in `settings`): `--config`, `--nodes`, `--metricsListenAddr`.

### `arrivalModel`

The single field that most changes a run's semantics.

- **`closed_loop`** (default) — legacy generate-then-send lockstep. Each worker
  generates a tx, sends it, then generates the next; throughput is bounded by
  sender latency. Susceptible to **coordinated omission** (slow sends suppress
  arrivals, hiding tail latency). `maxInFlight` is ignored. Keep as the
  regression baseline.
- **`open_loop`** — schedules tx *i* at t₀ + i/λ **independent of sender
  availability** (the coordinated-omission fix). λ comes from `tps>0` or the ramp
  curve (`rampUp`). When concurrent in-flight sends would exceed `maxInFlight`,
  the overdue tx is **dropped and counted** rather than throttling the clock —
  reported at exit as `Open-loop dropped N txs`. Use this for any latency claim.

Validation (`Settings.Validate`) rejects:
- an `arrivalModel` other than `open_loop`/`closed_loop`;
- `open_loop` with no finite positive rate (`tps<=0` **and** not `rampUp`) — λ
  would be infinite, the inter-arrival gap collapses to 0, and the scheduler spins
  and drops everything.

### Seed (reproducibility)

`seed` roots the deterministic PRNG sub-streams (keys, sizes, gas, accounts). Same
seed + same config reproduces the **draw multiset**, so the workload distribution
is statistically reproducible for fair A/B comparison. Caveats from the code:

- Per-tx emission ordering is reproducible only at a single worker; above one
  worker the multiset matches but ordering does not, and on-chain arrival order is
  concurrent regardless.
- Omitting `seed` means "unseeded": the generator draws a random seed, writes it
  back, and logs it for after-the-fact replay.

## Scenarios

`scenarios` is a weighted mix. Each entry creates one scenario instance; the
dispatcher selects among them by `weight` (relative, integer). The same `name` may
appear multiple times (instances are suffixed `_0`, `_1`, …).

| Field | JSON key | Type | Meaning |
|-------|----------|------|---------|
| Name | `name` | string | Scenario kind (case-insensitive match). See list below. |
| Weight | `weight` | int | Relative selection weight within the mix. |
| Accounts | `accounts` | object | Optional per-scenario account pool; overrides the shared pool for this scenario. |
| GasPicker | `gasPicker` | object | Optional gas-limit picker (`fixed`/`random`). |
| GasFeeCapPicker | `gasFeeCapPicker` | object | Optional `maxFeePerGas` picker. |
| GasTipCapPicker | `gasTipCapPicker` | object | Optional `maxPriorityFeePerGas` picker. |
| KeyDistribution | `keyDistribution` | object | Keyspace index distribution (`uniform`/`zipfian`). ⚠️ **Requires PLT-465 (#54, unmerged) — parses but does not affect generated transactions on main.** See [gap](#schema-vs-implementation-gaps). |
| SizeDistribution | `sizeDistribution` | object | Payload-size distribution. ⚠️ **Same status as `keyDistribution`: requires PLT-465; parses but does not affect generated txs on main.** |

### Scenario names

Matched case-insensitively. Registered on main:

`EVMTransfer`, `EVMTransferFast`, `EVMTransferNoop`, `ERC20`, `ERC20Noop`,
`ERC20Conflict`, `ERC721`, `Disperse`, `StorageRW`.

An unknown name panics at scenario creation — validate with `--dry-run` first.

### Gas pickers

A picker is a tagged object discriminated by `Name`:

```jsonc
"gasPicker": { "Name": "fixed",  "Gas": 21000 }
"gasPicker": { "Name": "random", "Min": 21000, "Max": 100000 }   // inclusive range
```

`random` requires `Min < Max`. With no picker, the scenario uses its built-in
defaults. Pickers are consumed by the EVMTransfer family (`GenerateGas`); the
field keys (`Name`, `Gas`, `Min`, `Max`) are capitalized on the wire.

### Distributions

A distribution is discriminated by `Name`:

```jsonc
"keyDistribution": { "Name": "uniform" }
"keyDistribution": { "Name": "zipfian", "theta": 0.9 }   // theta in [0, 1)
```

`zipfian.theta` must be in `[0, 1)`; `0` is uniform, larger hotspots low indices.
⚠️ These distributions (and the related `recordCount`, `sizeBuckets`, and
`operations` op-mix axes) **require PLT-465 (#54, unmerged as of writing) — on
main they parse but do not affect generated transactions.** See the
[implementation gap](#schema-vs-implementation-gaps) before relying on these for
workload skew.

## Accounts

```jsonc
"accounts": {
  "count": 500,           // pool size
  "newAccountRate": 0.0   // fraction of txs that mint a fresh recipient account
}
```

| Field | JSON key | Type | Default | Effect |
|-------|----------|------|---------|--------|
| Accounts | `count` | int | `0` | Number of pre-generated accounts in the pool. |
| NewAccountRate | `newAccountRate` | float64 | `0.0` | Fraction of transactions that target a newly-minted account instead of a pool member. `0` = fixed pool. |

A top-level `accounts` block is the **shared pool** for all scenarios; a
per-scenario `accounts` block creates a separate pool for that scenario. If
neither exists, scenario creation errors (`no accounts config defined`).

**Funding interaction:** funding requires `newAccountRate == 0` everywhere (both
top-level and per-scenario). On-demand accounts are never funded, so their first
tx would fail for gas — `ValidateFunding` rejects the combo at load.

## Funding

Optional. When set (and not `--dry-run`), the account pool is funded from a root
key at startup so the run works against a real chain.

| Field | JSON key | Type | Default | Meaning |
|-------|----------|------|---------|---------|
| RootKeyFile | `rootKeyFile` | string | `""` | Path to a file holding the root account's hex private key. **Preferred** — not exposed in the process environment. |
| RootKeyEnv | `rootKeyEnv` | string | `""` | Env var name holding the hex key. Fallback when `rootKeyFile` is unset. |
| FundAmountWei | `fundAmountWei` | string | `"1000000000000000000"` (1 SEI) | Per-account funding in wei. **Decimal STRING** (JSON numbers lose precision above 2^53). |
| BatchSize | `batchSize` | int | `200` | Recipients per `disperseEther` call. |

`ValidateFunding` requires exactly one key source (`rootKeyFile` or `rootKeyEnv`)
and `newAccountRate == 0` across all account configs.

## Gotchas

- **`seiChainID` casing is cosmetic.** The struct tag is `seiChainID` (capital `ID`),
  and several shipped profiles write `seiChainId` (lowercase `d`). Go's `encoding/json`
  matches tags **case-insensitively**, so `seiChainId` binds to the same field — the
  value is populated and the `chain_id` metric label and `chain_id`-keyed PromQL work
  regardless of casing. Prefer `seiChainID` for style consistency only; it has **no
  effect on binding or queries**.
- **Durations are strings.** `"10s"`, not `10`. A bare number fails to parse.
- **`fundAmountWei` is a string.** Quoting matters; an unquoted big number loses
  precision or fails.
- **Trackers read `endpoints[0]` only.** Order endpoints so the first is stable.

## Schema vs. implementation gaps

Verified against main at doc time:

- ⚠️ **`keyDistribution` / `sizeDistribution` / `sizeBuckets` / `recordCount` /
  `operations` require PLT-465 (#54, unmerged as of writing) — on main these
  fields parse but do not affect generated transactions.** They parse, validate,
  and bind to deterministic RNG sub-streams in the generator, but **no scenario on
  main calls `SampleIndex` on them** — the only `SampleIndex` call site is inside
  `config/distribution.go` itself. Setting these fields today has no behavioral
  effect on emitted transactions. PLT-465 (#54) is the pending PR that wires
  scenario sampling; once it lands, revisit this note and the StorageRW axes in
  [04-workload-model.md](04-workload-model.md).

## See also

- [01-mental-model.md](01-mental-model.md) — the pieces and how they connect.
- [02-running.md](02-running.md) — invoking the binary; CLI-flag equivalents.
- [04-workload-model.md](04-workload-model.md) — scenarios, distributions, accounts in depth.
- [06-measurement-metrics.md](06-measurement-metrics.md) — interpreting metrics and the run summary.
- [07-experiment-playbook.md](07-experiment-playbook.md) — reproducible experiment recipes.
