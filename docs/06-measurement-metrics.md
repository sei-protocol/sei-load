# 06 — Measurement & Metrics

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it: the exact signals sei-load emits, what
> they mean, and the conservation model that ties them together. Read this before
> writing any query against a run, before computing a rate/percentile/verdict, and
> before trusting a number. **The tool emits raw signals only; every rate,
> percentile, and pass/fail verdict is computed by you, the agent, via queries.**

---

## 1. The conservation model

sei-load supports an **open-loop** arrival model, but it is **opt-in**:
**closed-loop is the default** (see [01-mental-model](01-mental-model.md)), and open-loop
is selected with `--arrival-model open_loop` (see [04-workload-model](04-workload-model.md)
for arrival mechanics). The inclusion identities and inclusion-latency series below are
valid **only for open-loop runs**; in closed-loop `IntendedSendTime` is enqueue time, so
the latency sample is omitted (counts still tracked). Every transaction the scheduler
creates flows through two accounting stages whose terms must balance. These identities are
the foundation of run validity: if the terms don't add up, the run is suspect.

### Stage 1 — Send accounting (always tracked)

```
scheduled == dropped + admitted
admitted  == succeeded + failed
```

| Term | Meaning |
|------|---------|
| `scheduled` | Every arrival tick the open-loop scheduler reaches at instant `t₀ + i/λ`. Not directly emitted; it is the sum of the right-hand side. |
| `dropped` | Ticks shed because true in-flight was saturated (`maxInFlight` reached) at the scheduled instant. **Genuine load shed**, not buffer geometry. A dropped tick draws no generator and signs no tx. → `seiload_run_txs_dropped_total`. |
| `admitted` | Ticks that acquired an in-flight permit and were generated + signed + enqueued. |
| `succeeded` | Admitted txs whose synchronous RPC send returned nil error (accepted by the endpoint). → `seiload_txs_accepted_total`. |
| `failed` | Admitted txs whose send returned an error. Counted, never lost. → `seiload_run_txs_failed_total` (and per-error `seiload_txs_rejected_total`). |

**Shutdown boundary:** `admitted == succeeded + failed` holds exactly only on a clean
drain (generator exhaustion). On `ctx` cancel (SIGTERM / `--duration` expiry), some
admitted txs may still be buffered for a worker and exit uncounted — bounded by channel
backlog. For latency/goodput claims prefer runs that drain cleanly or are long enough
that the boundary undercount is negligible.

### Stage 2 — Inclusion accounting (only with `--track-receipts`)

```
registered == included + expired + inflight_at_shutdown
registered ⊆ succeeded        (only successful sends are registered)
```

> Note: `dropped_at_cap` is **not** a term in this identity — it is excluded.
> `registered = succeeded − dropped_at_cap` (sends rejected at the registry cap were
> never registered, so they appear in neither side of the conservation balance).

| Term | Meaning |
|------|---------|
| `registered` | Successful sends handed to the inclusion tracker. **Not its own series** — by design the denominator for inclusion rate is `succeeded` (`seiload_txs_accepted_total`), never a minted `registered` series. |
| `included` | Txs observed on-chain (matched in an arriving block, `InclusionTime` stamped). → the `_count` of `seiload_inclusion_latency_seconds` **in open-loop only**; otherwise read from the run-summary log line / `seiload_inclusion_outcome_total` is *not* it. See §3.1. |
| `expired` | Registered txs reaped un-included after `reapAfter` (default 30s, `--inclusion-reap-after`). → `seiload_inclusion_outcome_total{outcome="expired"}`. |
| `dropped_at_cap` | Successful sends rejected at the inclusion-registry cap (registry full). **Excluded from the inclusion denominator** — they were never registered. → `seiload_inclusion_outcome_total{outcome="dropped_at_cap"}`. |
| `inflight_at_shutdown` | Registry size at run end, read after workers + tracker join. → `seiload_run_inflight_at_shutdown`. |

**Conservative degradation (undercounts only, never miscounts):** WS head gaps
(`seiload_block_gaps_total`), block-body fetch failures (`seiload_block_fetch_errors_total`),
and late registrations all cause affected txs to reap as `expired` rather than be
miscounted as included. A nonzero `seiload_block_gaps_total` or
`seiload_block_fetch_errors_total` means your `included` is an **under**count — factor that
into inclusion-rate claims.

---

## 2. The emitted-metric catalog

All instruments are OTel, exported on the Prometheus `/metrics` endpoint
(`--metricsListenAddr`, default `0.0.0.0:9090`; OpenMetrics enabled so exemplars
survive).

> **Wire names differ from the instrument base names.** The Prometheus exporter is
> configured with `WithNamespace("seiload")` (configurable; `observability/setup.go`),
> so every exported series is prefixed **`seiload_`**. OTel also appends **unit
> suffixes** on scrape: a `s`-unit histogram becomes `…_seconds`, etc. Combined with
> Prometheus's own suffixing — histograms expose `_bucket`/`_sum`/`_count`, counters end
> `_total` — the wire name can differ substantially from the base name an instrument is
> declared with. The catalog and every PromQL below use the **real wire names**. (The
> `{gas}`, `{height}`, `{transactions}`, `{count}` "annotation" units are dropped, not
> suffixed; only real units like `s` / `/s` produce a suffix.)

### 2.1 Block & gas signals (require `--track-blocks`)

Emitted by the block collector from new-head subscriptions (`stats/block_collector.go`).
`seiload_block_time_seconds` is **header-arrival-to-arrival wall clock**, not `header.Time`.

| Metric | Type | Unit | Attributes | Meaning |
|--------|------|------|-----------|---------|
| `seiload_gas_used` | histogram | `{gas}` (dropped) | `chain_id` | Gas used per block (`_bucket`/`_sum`/`_count`). Buckets: 1, 1k, 10k, 50k, 100k, 200k, 300k, 400k, 500k, 600k, 700k, 800k, 1M. |
| `seiload_block_time_seconds` | histogram | `s` | `chain_id` | Wall-clock interval between observed block headers (`_bucket`/`_sum`/`_count`). Buckets: 0.1…1.0 (0.1 step), 2, 5, 10, 20. |
| `seiload_block_number` | gauge | `{height}` (dropped) | `chain_id` | Highest block height observed (monotonic). |

### 2.2 Send-path signals (always on)

Emitted from the worker send loop (`sender/worker.go`, `sender/metrics.go`).

| Metric | Type | Unit | Attributes | Meaning |
|--------|------|------|-----------|---------|
| `seiload_send_latency_seconds` | histogram | `s` | `scenario`, `endpoint`, `chain_id`, `status` (`success`/`failure`) | RPC send round-trip latency (`_bucket`/`_sum`/`_count`). **NOT inclusion latency** — this is enqueue→RPC-return, the SUT-admission cost, not time-to-chain. Buckets: 0.1, 0.2, 0.3, 0.5, 1, 2, 3, 5, 10, 20. Carries trace exemplars. |
| `seiload_txs_accepted_total` | counter | `{transactions}` (dropped) | `endpoint`, `scenario` | Sends accepted by the endpoint (`succeeded`). The inclusion-rate denominator. |
| `seiload_txs_rejected_total` | counter | `{transactions}` (dropped) | `endpoint`, `scenario`, `reason` (currently only `rpc`) | Sends the target/client rejected (`failed`). |
| `seiload_worker_queue_length` | observable gauge | `{count}` (dropped) | `endpoint`, `worker_id`, `chain_id` | Current depth of a worker's send channel. Saturation/backpressure signal. |
| `seiload_tps_achieved_per_second` | observable gauge | `{transactions}/s` | `endpoint`, `chain_id`, `scenario` | Most recent sender-sampled TPS per endpoint/scenario. |

### 2.3 Inclusion signals (require `--track-receipts`, not under `--dry-run`)

Emitted by `stats.InclusionTracker` (`stats/inclusion_tracker.go`).

| Metric | Type | Unit | Attributes | Meaning |
|--------|------|------|-----------|---------|
| `seiload_inclusion_latency_seconds` | histogram | `s` | `chain_id` | `InclusionTime − IntendedSendTime` (`_bucket`/`_sum`/`_count`). **Open-loop only** (in closed-loop `IntendedSendTime` is enqueue time, so the sample is omitted; counts still tracked). Its `_count` is the open-loop `included` total. Buckets: 0.5, 1, 2, 5, 10, 30, 60, 120. |
| `seiload_inclusion_outcome_total` | counter | `{transactions}` (dropped) | `chain_id`, `outcome` (`expired` \| `dropped_at_cap`) | In-flight txs that left the registry un-included. |
| `seiload_block_gaps_total` | counter | `{blocks}` (dropped) | `chain_id` | Missed head heights (no backfill). Nonzero ⇒ `included` is an undercount. |
| `seiload_block_fetch_errors_total` | counter | `{blocks}` (dropped) | `chain_id` | Block-body fetches that failed (no retry); those txs reap as `expired`. Nonzero ⇒ `included` undercount. |
| `seiload_inclusion_inflight` | observable gauge | `{transactions}` (dropped) | `chain_id` | Live size of the in-flight inclusion registry. |

### 2.4 Run-summary gauges (emitted once at run end)

Recorded by `Collector.EmitRunSummary` (`stats/run_summary.go`) at shutdown, then held
for `--post-summary-flush-delay` (default 25s) so the final scrape catches them. One
series per run via the OTel Resource (run-scope) join.

| Metric | Type | Unit | Attributes | Meaning |
|--------|------|------|-----------|---------|
| `seiload_run_tps_final_per_second` | gauge | `{transactions}/s` | — | Peak observed overall TPS (10s sliding-window max) for the run. |
| `seiload_run_duration_seconds` | gauge | `s` | — | Wall-clock run duration. |
| `seiload_run_txs_accepted_total` | gauge | `{transactions}` (dropped) | — | Total txs accepted by endpoints over the run (collector's `totalTxs`). Gauge already named `…_total`; no extra suffix. |
| `seiload_run_txs_dropped_total` | gauge | `{transactions}` (dropped) | `arrival_model` | Open-loop txs `dropped` on in-flight saturation. |
| `seiload_run_txs_failed_total` | gauge | `{transactions}` (dropped) | `arrival_model` | Admitted txs whose send `failed`. |
| `seiload_run_inflight_at_shutdown` | gauge | `{transactions}` (dropped) | — | Inclusion registry size at end (only emitted when `--track-receipts`). |

> Note: `seiload_run_tps_final_per_second` is a **peak** (sliding-window max), not a
> mean. For a mean, compute `seiload_run_txs_accepted_total / seiload_run_duration_seconds`.

---

## 3. Verdicts are external — compute them yourself

The tool deliberately emits **counts and histograms**, not rates/percentiles/verdicts.
You derive those. Concrete recipes follow.

Run-scope identity rides on the OTel **Resource**, not on per-sample labels, so it reaches
PromQL via a join target (e.g. `target_info` / `seiload_target_info`) rather than as a label
on each series. The run-scope join keys that *can* exist are
`seiload_run_id`, `seiload_chain_id`, `seiload_commit_id`, `seiload_workload`,
`service_instance_id`, and `service_version` (`observability/setup.go`). Each is
**conditional on its `SEILOAD_*` env var being set** (`service_instance_id` falls back to
hostname; the rest are omitted when empty) — adjust selectors to your environment. See
[../observability/README.md](../observability/README.md) for the cardinality rationale and
how the Resource is exported.

### 3.1 Inclusion rate

`included / succeeded`. In **open-loop**, `included` is the `seiload_inclusion_latency_seconds`
histogram count:

```promql
# open-loop inclusion rate over the run
sum(seiload_inclusion_latency_seconds_count) / sum(seiload_txs_accepted_total)
```

In **closed-loop**, `seiload_inclusion_latency_seconds` is not recorded — read `included` from the
run-summary log line (`📦 Inclusion: included=…`) or compute the complement from outcomes:
`included = registered − expired − dropped_at_cap − inflight_at_shutdown`, where
`registered = succeeded − dropped_at_cap`. For rate claims that need a histogram count,
**use open-loop** (§ [05-reproducibility](05-reproducibility.md)).

Subtract the un-included tail explicitly when you need the loss breakdown:

```promql
sum(seiload_inclusion_outcome_total{outcome="expired"})        # timed out un-included
sum(seiload_inclusion_outcome_total{outcome="dropped_at_cap"}) # registry full (denominator excludes these)
```

### 3.2 Latency percentiles (tail)

Use `histogram_quantile` over the open-loop inclusion histogram for **time-to-chain**:

```promql
# p99 inclusion latency, open-loop only
histogram_quantile(0.99, sum by (le) (rate(seiload_inclusion_latency_seconds_bucket[1m])))
```

For **admission latency** (send round-trip, any model):

```promql
histogram_quantile(0.99, sum by (le) (rate(seiload_send_latency_seconds_bucket[1m])))
```

Do **not** quote `seiload_inclusion_latency_seconds` percentiles from a closed-loop run — the histogram
is empty there, and even where it exists closed-loop suffers coordinated omission
(see [04-workload-model](04-workload-model.md)).

### 3.3 Goodput (committed / offered)

Goodput = on-chain commitments per second relative to what was offered:

```promql
# committed throughput (TPS)
sum(seiload_inclusion_latency_seconds_count) / scalar(seiload_run_duration_seconds)

# goodput ratio: committed / offered
sum(seiload_inclusion_latency_seconds_count) / sum(seiload_run_txs_accepted_total)
```

Drop and failure fractions of offered load:

```promql
sum(seiload_run_txs_dropped_total) / (sum(seiload_run_txs_accepted_total) + sum(seiload_run_txs_dropped_total))
sum(seiload_run_txs_failed_total)  / sum(seiload_run_txs_accepted_total)
```

### 3.4 Detecting a generator-bound (invalid) run — `schedule_lag`

A run is only a valid load measurement if the generator **kept up with its own
schedule**. The canonical gate is `schedule_lag = AttemptedSendTime − IntendedSendTime`
(sends falling behind the arrival schedule even before any tx is shed).

> **`schedule_lag` is a concept, NOT an emitted metric on main today** (the emitter
> was punted as PLT-463). You cannot query it. Compute run validity externally from
> the signals that *are* emitted:
>
> - **High `seiload_run_txs_dropped_total` with low SUT utilization** ⇒ suspect the
>   generator (or `maxInFlight`) shed load before the SUT was saturated. Drops should
>   track SUT saturation, not generator stalls.
> - **`seiload_run_tps_final_per_second` ≪ configured `--tps`** ⇒ the generator never
>   reached target rate; the run under-loaded the SUT and latency/throughput numbers are
>   not at the intended λ.
> - **Rising `seiload_worker_queue_length`** ⇒ workers are backing up; admission is the bottleneck.
>
> If you need a hard generator-validity gate, file an `/issue` requesting a
> `schedule_lag` histogram (the query you want: `histogram_quantile(0.99,
> schedule_lag_bucket)` to assert p99 lag < one inter-arrival gap). Until then, treat the
> heuristics above as the validity check and state the assumption in your report.

---

## 4. Reading the run-summary / final stats output

Two surfaces report end-of-run state. **Both** are worth capturing.

### 4.1 Run-summary gauges + log lines (authoritative for conservation)

At shutdown sei-load logs the conservation tallies and records the §2.4 gauges:

```
⚠️  Open-loop dropped N txs (in-flight saturated; not throttled)
⚠️  Open-loop N txs failed to send (admitted but errored; not lost)
📦 Inclusion: included=… expired=… dropped_at_cap=… inflight_at_shutdown=…
```

This log line is the ground truth for the Stage-2 identity; cross-check it against your
`inclusion_*` queries. The gauges persist on `/metrics` for `--post-summary-flush-delay`
so a final scrape captures them — ensure your scrape interval is shorter than that delay.

### 4.2 `--report-path` file / stdout final stats

`Logger.LogFinalStats` (`stats/logger.go`) prints — and, with `--report-path`, writes —
a **formatted text report** (not JSON, despite the JSON-tagged `FinalStats` struct).
Schema-versioned run-summary JSON is future work (PLT-467); do not write a parser
expecting JSON from `--report-path` today.
It contains: runtime, total txs, avg/max TPS, per-endpoint P50/P99 (in-process
percentiles over a 10k-sample ring buffer — coarse, not the histogram), per-scenario
distribution, and block-time/gas P50/P99/max.

> Caveat: the report's per-endpoint P50/P99 are computed in-process over a bounded
> latency ring (`maxLatencyHistory = 10000`) and are **send latency**, not inclusion
> latency. For trustworthy tail-latency claims use the `seiload_inclusion_latency_seconds` /
> `seiload_send_latency_seconds` histograms via `histogram_quantile` (§3.2), not the report file.

---

## See also

- [01-mental-model](01-mental-model.md) — what sei-load is and isn't.
- [04-workload-model](04-workload-model.md) — open-loop arrival, λ, drops, coordinated omission.
- [05-reproducibility](05-reproducibility.md) — fixed seed, open vs closed loop, fair A/B.
- [07-experiment-playbook](07-experiment-playbook.md) — objective → knobs → interpretation.
- [08-limits-boundaries](08-limits-boundaries.md) — what to rule out before trusting a result.
