# 07 — Experiment Playbook

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it: the reasoning layer on top of the metrics.
> Given an objective, which knobs to turn, which signals to read, and what they mean for
> your next move. Read this when you are about to *design* a run, not just interpret one.
> Metric names and PromQL referenced here are defined in
> [06-measurement-metrics](06-measurement-metrics.md) — do not re-derive them.

---

## 0. The autonomous run loop

Every experiment is one turn of this loop. Run it deliberately; don't fire-and-forget.

```
1. OBJECTIVE   → what question am I answering? (capacity? tail latency? contention?)
2. KNOBS       → set exactly the variables under test; FREEZE everything else (seed!).
3. VALIDITY    → is this run a fair measurement? (§5 — check BEFORE trusting numbers)
4. READ        → pull the specific signals the objective needs.
5. INTERPRET   → what do they mean? does conservation balance?
6. NEXT MOVE   → adjust one knob, or conclude. Record the seed + config for A/B.
```

**Cardinal rule for comparability:** change **one** independent variable per run, hold a
**fixed seed** (set the top-level `seed` in the config file — there is **no `--seed` CLI
flag**; see [05-reproducibility](05-reproducibility.md)), and use **open-loop**
(`--arrival-model open_loop`; note closed-loop is the *default*) for any latency or
capacity claim.

> ⚠️ **StorageRW axes require PLT-465 (#54, unmerged as of writing).** Recipes 2 and 3
> below sweep `keyDistribution`/`zipfian-θ`/`recordCount`/`sizeDistribution`/`sizeBuckets`/
> `operations`. **On main these fields parse but do not affect generated transactions** —
> StorageRW emits a fixed scaffold (slot 0, empty pad, all-`rmw`). Treat the contention and
> size sweeps as runnable only once PLT-465 lands; see
> [04-workload-model](04-workload-model.md).

---

## 1. Decision framework — objective → knobs

| Objective | Primary knob(s) | Hold fixed | Read |
|-----------|-----------------|-----------|------|
| Key/state contention | scenario `StorageRW`, zipfian-θ over `recordCount` **(PLT-465 — no effect on main)** | seed, λ, tx mix, endpoints | `seiload_tps_achieved_per_second`, `seiload_inclusion_latency_seconds` p99, SUT block-stm abort rate (external) |
| Tx-size scaling | size-distribution / `sizeBuckets` **(PLT-465 — no effect on main)** | seed, λ | `seiload_gas_used` per block, `seiload_send_latency_seconds`, `seiload_inclusion_latency_seconds` |
| Trustworthy tail latency | fixed λ at/above suspected capacity, open-loop | seed, mix | `seiload_inclusion_latency_seconds` p99 via `histogram_quantile`; validity (§5) |
| Throughput knee | λ sweep or `--ramp-up` | seed, mix | `seiload_run_txs_dropped_total`, inclusion rate, `seiload_inclusion_latency_seconds`, `seiload_block_time_seconds` |

---

## 2. Recipe: probe key/state contention

**Goal:** find how concurrent reads/writes to a hot key-set degrade throughput — i.e.
expose Sei's parallel-execution (block-stm) conflict/abort behavior (see
[04-workload-model](04-workload-model.md) for the Sei mechanism).

**Design:** `StorageRW` scenario, sweep zipfian skew **θ** while sweeping `recordCount`
(smaller `recordCount` + higher θ = hotter contention). Hold seed, λ, endpoints, and tx
mix constant across the sweep.

**Read & interpret:**
- Throughput vs θ: as θ rises, committed throughput (`seiload_inclusion_latency_seconds_count /
  seiload_run_duration_seconds`) should fall if the SUT serializes on conflicts. A flat curve means you
  haven't reached the contention regime — raise θ / shrink `recordCount`.
- Pair with the **SUT's** block-stm conflict/abort rate (a Sei **node-side** signal,
  not emitted by sei-load — on Sei this typically surfaces as `sei_occ_*`, but confirm
  the exact series name exists on your SUT before relying on it; the node version under
  test may not export it at all. File `/issue` if that signal isn't exposed where you
  can query it).
- `seiload_block_time_seconds` widening while `seiload_gas_used` holds steady ⇒ execution
  is the bottleneck, not block fullness — a contention signature.

**Next move:** binary-search θ for the knee where throughput drops sharply; that θ is the
contention threshold for this `recordCount`.

---

## 3. Recipe: probe tx-size scaling

**Goal:** how does per-tx size/gas affect block packing and latency.

**Design:** sweep the size distribution / `sizeBuckets`. Hold seed and λ fixed.

**Read & interpret:**
- `seiload_gas_used` histogram (per block) — does the SUT hit the gas ceiling
  (`--target-gas`, default 10M)? `histogram_quantile(0.99, …seiload_gas_used_bucket…)` near
  the ceiling ⇒ blocks are gas-bound.
- `seiload_send_latency_seconds` and `seiload_inclusion_latency_seconds` — larger txs raise
  both if execution/propagation cost scales with size.
- `seiload_block_time_seconds` — rising with size ⇒ production cost is size-sensitive.

**Next move:** if blocks are gas-bound before λ saturates, you are measuring block-packing,
not throughput — lower per-tx gas or raise `--target-gas` to isolate the variable.

---

## 4. Recipe: measure trustworthy tail latency

**Goal:** a defensible p99 time-to-chain.

**Design (load-bearing):**
- `--arrival-model open_loop` — **mandatory**. Closed-loop suffers coordinated omission
  and `seiload_inclusion_latency_seconds` is not even recorded there (see [06](06-measurement-metrics.md) §3.2).
- Fixed λ (`--tps`) **at or above** suspected capacity — you want the schedule to expose
  the slowdown, not avoid it.
- `--track-receipts` enabled (inclusion histogram requires it).
- Fixed seed.
- Size `--max-in-flight` and `--inclusion-reap-after` so healthy txs aren't reaped:
  registry cap auto-sizes from `TPS × reapAfter × 1.5`, but verify `dropped_at_cap == 0`.

**Read:**
```promql
histogram_quantile(0.99, sum by (le) (rate(seiload_inclusion_latency_seconds_bucket[1m])))
```

**Interpret / validity gate:** the p99 is only trustworthy if the run wasn't
generator-bound (§5). Confirm `seiload_run_tps_final_per_second ≈ --tps`,
`dropped_at_cap == 0`, and `seiload_block_gaps_total == 0 && seiload_block_fetch_errors_total == 0`
(else `included` is an undercount biasing the tail). If those hold, quote the p99;
otherwise rerun.

---

## 5. Ensuring a run is VALID / comparable

Run this checklist **before** trusting any number. A failing item invalidates the run.

| Check | Query / signal | Pass condition | If it fails |
|-------|----------------|----------------|-------------|
| Fixed seed | config | identical seed across A/B | reseed; reruns aren't comparable |
| Open-loop for latency | `--arrival-model` | `open_loop` | closed-loop → coordinated omission; rerun |
| Generator kept up | `seiload_run_tps_final_per_second` vs `--tps` | within tolerance | under-loaded; raise workers/λ headroom |
| Drops are real shedding | `seiload_run_txs_dropped_total` | tracks SUT saturation, not generator stalls | suspect generator/`maxInFlight`; see [06](06-measurement-metrics.md) §3.4 |
| No registry starvation | `seiload_inclusion_outcome_total{outcome="dropped_at_cap"}` | `== 0` | raise `--inclusion-reap-after` / cap; inclusion undercounted |
| No observer loss | `seiload_block_gaps_total`, `seiload_block_fetch_errors_total` | `== 0` | `included` undercounts; treat inclusion rate as a lower bound |
| Sends not erroring en masse | `seiload_run_txs_failed_total`, `seiload_txs_rejected_total` | low / explained | investigate SUT/client rejection before reading throughput |
| Conservation balances | run-summary log + queries | `registered == included + expired + inflight_at_shutdown` | accounting broken; do not trust derived rates |
| Clean shutdown | drain vs SIGTERM/`--duration` | clean drain preferred for exact accounting | note the shutdown-boundary undercount in your report |

`schedule_lag` is the ideal generator-validity gate but is a **concept, not an emitted
metric on main** (emitter punted as PLT-463) — you cannot query it. Compute validity from
the heuristics above and state the assumption. See
[06-measurement-metrics](06-measurement-metrics.md#34-detecting-a-generator-bound-invalid-run--schedule_lag) §3.4.

For fair A/B methodology see [05-reproducibility](05-reproducibility.md); for failure
modes to rule out (what a bad number *isn't*) see [08-limits-boundaries](08-limits-boundaries.md).

---

## 6. Compact run → check → mean → move loop

A drop-in autonomous sequence for a single run:

| Run output | Metric to check | What it means | Next move |
|------------|-----------------|---------------|-----------|
| Run started | `seiload_tps_achieved_per_second`, `seiload_worker_queue_length` | Is the generator hitting λ? | Queue rising + TPS < λ ⇒ add `--workers` |
| Mid-run | `seiload_block_time_seconds`, `seiload_gas_used` p99 | Is the SUT block-bound or gas-bound? | Gas-bound ⇒ adjust tx size / `--target-gas` |
| Mid-run | `seiload_run_txs_dropped_total` climbing | In-flight saturating | Near/above capacity — good for tail latency; bad if you wanted under-capacity |
| End | run-summary log line | Conservation balances? | If not, discard run |
| End | inclusion rate (§3.1 of [06](06-measurement-metrics.md)) | Fraction reaching chain | < target ⇒ SUT shedding; investigate expired vs dropped_at_cap |
| End | `seiload_inclusion_latency_seconds` p99 | Tail time-to-chain | Validity-gate it (§5), then record with seed + config |
| End | `seiload_block_gaps_total`/`seiload_block_fetch_errors_total` | Observer integrity | Nonzero ⇒ inclusion is a lower bound; note it |

**When a needed signal doesn't exist** (e.g. `schedule_lag`, SUT block-stm aborts where
you can query them), do not paper over it: file an `/issue` naming the exact query you
were trying to write and why, so the gap gets closed rather than guessed around.

---

## See also

- [01-mental-model](01-mental-model.md) — what sei-load is and isn't.
- [04-workload-model](04-workload-model.md) — arrival model, scenarios, the Sei contention mechanism.
- [05-reproducibility](05-reproducibility.md) — fixed seed, open vs closed loop, fair A/B.
- [06-measurement-metrics](06-measurement-metrics.md) — the metric catalog and PromQL.
- [08-limits-boundaries](08-limits-boundaries.md) — what to rule out before trusting a result.
