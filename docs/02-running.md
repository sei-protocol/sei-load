# Running sei-load

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it: how to build and invoke the `seiload`
> binary, every CLI flag and its effect, settings precedence, the metrics
> endpoint, a ladder of copy-pasteable invocations, and the run lifecycle
> (prewarm → run → run-summary → flush). Read this when you are about to start,
> stop, or reproduce a run. For the meaning of config-file fields, see
> [03-config-reference.md](03-config-reference.md).

## Build and run

```bash
make build                       # produces ./build/seiload
./build/seiload --config <path>  # --config is REQUIRED; run fails fast without it
```

The binary reads one JSON config file (`--config`/`-c`), resolves settings (CLI >
config > defaults), validates them, then runs until its duration elapses or it
receives `SIGTERM`/`SIGINT`.

A run with no endpoints or no scenarios in the config is rejected at load time
(`no endpoints specified` / `no scenarios specified`).

### Metrics endpoint

Prometheus metrics are served at `http://<metricsListenAddr>/metrics` (default
`0.0.0.0:9090`). OpenMetrics is enabled so exemplars survive scraping. Point your
scraper here; the run holds the process open for a scrape window after it
finishes (see [Lifecycle](#lifecycle)). To export traces/OTLP, set
`OTEL_EXPORTER_OTLP_ENDPOINT` in the environment.

## CLI flags

Every flag below maps 1:1 to a `settings` field (same default) except `--config`,
`--nodes`, and `--metricsListenAddr`, which are CLI-only. Flag defaults come from
`DefaultSettings()`; config-file values override these defaults, and CLI flags
override the config file.

| Flag | Short | Default | Meaning / effect |
|------|-------|---------|------------------|
| `--config` | `-c` | (required) | Path to the JSON config file. No default; run aborts if unset. |
| `--workers` | `-w` | `1` | Tasks (workers) **per endpoint**. Total senders = workers × endpoints. |
| `--tps` | `-t` | `0` | Target transactions/sec, shared across all workers (single rate limiter). `0` = no limit. Required (>0) for open-loop unless `--ramp-up`. |
| `--arrival-model` | | `closed_loop` | `open_loop` schedules tx *i* at t₀+i/λ and drops overdue txs; `closed_loop` is the legacy generate-then-send lockstep. See [03](03-config-reference.md#arrivalmodel). |
| `--max-in-flight` | | `10000` | **Open-loop only.** Max concurrent in-flight sends; txs that would exceed this at their scheduled instant are dropped and counted (the clock is never throttled). Ignored in closed-loop. |
| `--stats-interval` | `-s` | `10s` | Interval for logging throughput/latency stats and the user-latency tracker tick. |
| `--buffer-size` | `-b` | `1000` | Channel buffer size per worker. Larger = more in-memory queueing; reduce under memory pressure. |
| `--dry-run` | | `false` | Simulate generation/sending without hitting the chain. Forces `mockDeploy`. Disables the inclusion tracker (simulated sends never land, would all reap as expired). |
| `--debug` | | `false` | Log each transaction. High-volume; for small/diagnostic runs only. |
| `--track-receipts` | | `false` | Enable the block-indexed tx→inclusion tracker (stamps inclusion time; reports included/expired/dropped-at-cap/inflight-at-shutdown). No-op under `--dry-run` or with zero endpoints. |
| `--inclusion-reap-after` | | `30s` | How long an un-included tx stays in the inclusion registry before being reaped as **expired**. Tune to expected inclusion time on congested chains. Only meaningful with `--track-receipts`. |
| `--track-blocks` | | `false` | Collect block statistics (block time, gas) from `endpoints[0]`. |
| `--track-user-latency` | | `false` | Track per-user latency from `endpoints[0]`, sampled at `--stats-interval`. |
| `--prewarm` | | `false` | Prewarm accounts with self-transactions before the main run (warms nonces/state; excluded from main stats). |
| `--ramp-up` | | `false` | Drive load with a built-in ramp curve instead of a fixed rate. Provides a finite λ for open-loop without a fixed `--tps`. Curve is fixed in code: start 100 TPS, +100 per step, 120s load interval, 30s recovery interval. |
| `--report-path` | | `""` | Write a **formatted text** report to this path (`/dev/stdout` is valid). Empty = no report file. Note: a text dump today, **not** JSON — schema-versioned run-summary JSON is future work (PLT-467). See [06 §4.2](06-measurement-metrics.md#42---report-path-file--stdout-final-stats). |
| `--txs-dir` | | `""` | Write generated transactions to this dir instead of sending them (offline tx-writer mode). Forces closed-loop; open-loop is ignored with a logged downgrade. |
| `--target-gas` | | `10000000` | Target gas per block (tx-writer mode). |
| `--num-blocks-to-write` | | `100` | Number of blocks to write (tx-writer mode). |
| `--duration` | | `0` | Run duration. `0` = run until `SIGTERM`/`SIGINT`. |
| `--post-summary-flush-delay` | | `25s` | In-process sleep AFTER the run-summary metrics are recorded, so Prometheus can scrape final values before exit. Set `0` to exit immediately (you lose the final scrape). |
| `--nodes` | `-n` | `0` | Limit to the first N endpoints from the config. `0` = use all. |
| `--metricsListenAddr` | | `0.0.0.0:9090` | `ip:port` for the Prometheus `/metrics` endpoint. |

> Trackers that read chain state (`--track-blocks`, `--track-user-latency`,
> `--track-receipts`, and the ramper's block collector) all read from
> `endpoints[0]` only. Put a representative/stable RPC first.

> **No `--seed` flag.** The seed is **config-file-only** (top-level `seed`,
> `LoadConfig.Seed *uint64`). To pin or replay a workload, set `seed` in the config
> file — there is no CLI override. See
> [05-reproducibility.md](05-reproducibility.md#setting-the-seed).

> **`seiChainID` casing is cosmetic only.** The struct tag is `seiChainID` (capital
> `ID`), and several shipped profiles write `seiChainId` (lowercase `d`). Go's
> `encoding/json` matches tags **case-insensitively**, so `seiChainId` binds to the
> same field — `chain_id` is populated and `chain_id`-keyed PromQL works either way.
> Prefer `seiChainID` for style consistency, but it does **not** affect binding or
> queries. See [03 gotchas](03-config-reference.md#gotchas).

## Settings precedence

```
CLI flag  >  config-file "settings"  >  built-in default
```

Resolution is via viper: defaults are seeded from `DefaultSettings()`, the config
file's `settings` block is merged, then bound CLI flags override. A field absent
everywhere falls back to its default. After resolution the settings are validated
(`Settings.Validate`) and the run aborts on an invalid combination — notably
`arrival-model open_loop` with no finite rate (`--tps<=0` and not `--ramp-up`).

## Common invocations

Minimal → realistic.

**1. Validate a config without touching the chain (dry-run):**
```bash
./build/seiload --config profiles/local.json --dry-run --debug
```
Generates and logs transactions; deploys are mocked; no sends. Use this to
confirm scenarios, accounts, and weights resolve before a real run.

**2. Closed-loop, fixed TPS (legacy baseline):**
```bash
./build/seiload --config profiles/local.json --workers 50 --tps 100
```
Workers generate then send in lockstep; the shared limiter caps aggregate rate at
100 TPS. Susceptible to coordinated omission — prefer open-loop for latency
claims.

**3. Open-loop, fixed λ (coordinated-omission-correct):**
```bash
./build/seiload --config profiles/local.json \
  --arrival-model open_loop --tps 100 --max-in-flight 5000
```
Arrivals are scheduled at t₀+i/λ independent of sender availability; if in-flight
hits `--max-in-flight` the overdue tx is dropped and counted (reported as
`Open-loop dropped N txs` at exit) rather than slowing the clock.

**4. Ramped run (open-loop, no fixed TPS):**
```bash
./build/seiload --config profiles/local.json --arrival-model open_loop --ramp-up
```
The ramper supplies a finite, increasing λ to the shared limiter — this satisfies
open-loop's "finite positive rate" requirement without `--tps`. Final ramp stats
are logged at exit.

**5. Run with inclusion + block tracking:**
```bash
./build/seiload --config profiles/arctic-1.json \
  --track-receipts --track-blocks --inclusion-reap-after 45s
```
Stamps each sent tx and matches it against on-chain blocks from `endpoints[0]`;
at exit reports `included / expired / dropped_at_cap / inflight_at_shutdown`. On
a congested chain raise `--inclusion-reap-after` so slow-but-real inclusions are
not miscounted as expired.

**6. Limit endpoints with `--nodes`:**
```bash
./build/seiload --config profiles/local_docker.json --nodes 2
```
Uses only the first 2 of the config's endpoints. Useful to A/B fan-out without
editing the config.

**7. Bounded duration vs. signal-driven:**
```bash
./build/seiload --config profiles/local.json --tps 100 --duration 5m   # stops after 5m
./build/seiload --config profiles/local.json --tps 100                  # runs until Ctrl-C / SIGTERM
```

## Lifecycle

A run proceeds in this order:

1. **Load + resolve + validate** config and settings; abort fast on bad combos.
2. **Setup**: start the metrics server, observability, block/user-latency/inclusion
   trackers (per flags), and connect the sharded sender.
3. **Fund** the account pool (only if `funding` is set and not `--dry-run`).
4. **Prewarm** (if `--prewarm`): self-transactions warm accounts; excluded from
   main stats (the stats logger starts *after* prewarm).
5. **Run**: dispatcher drives the workload (open- or closed-loop) under the shared
   rate limiter; stats logged every `--stats-interval`.
6. **End**: the run stops when `--duration` elapses (context timeout) or a
   `SIGTERM`/`SIGINT` arrives. Workers and trackers drain and join.
7. **Run summary**: final stats are logged, the inclusion conservation identity is
   read after join (so `inflight_at_shutdown` is final), and a run-summary metric
   is emitted (`arrival_model`, `dropped`, `failed`, inclusion counts).
8. **Flush window**: the process sleeps `--post-summary-flush-delay` (default
   `25s`) so Prometheus can scrape the final summary, then exits cleanly. A
   `context.Canceled`/`DeadlineExceeded` from a clean duration/signal stop is
   treated as success (exit 0).

> If you scrape final summary metrics, the scrape interval must be shorter than
> `--post-summary-flush-delay`, or set the delay higher. Setting it to `0` exits
> immediately and the last scrape is lost.

## See also

- [01-mental-model.md](01-mental-model.md) — what sei-load is and how its pieces fit.
- [03-config-reference.md](03-config-reference.md) — the full config schema these flags mirror.
- [04-workload-model.md](04-workload-model.md) — scenarios, distributions, accounts.
- [06-measurement-metrics.md](06-measurement-metrics.md) — what the metrics/summary mean.
- [07-experiment-playbook.md](07-experiment-playbook.md) — recipes for reproducible experiments.
