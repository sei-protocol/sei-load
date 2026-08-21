# Mental Model

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it. The conceptual foundation you must
> hold before designing, running, or interpreting a sei-load experiment: the send
> pipeline, the open-loop arrival model and the coordinated-omission problem it
> solves, where verdicts come from (not the tool), and the load-bearing
> vocabulary. Read this first; the config and metric specifics are in the sibling
> docs linked at the end.

## What sei-load is

sei-load drives synthetic transaction load at a Sei EVM endpoint and **emits
measurements** about how the system under test (SUT) responds. It is a load
generator and a measurement instrument. It is **not** a judge: it does not
compute pass/fail verdicts or SLO compliance (see [Measurement philosophy](#measurement-philosophy)).

## The send pipeline

A transaction flows through a fixed pipeline:

```
generator → dispatcher → sharded sender → per-endpoint workers → Sei RPC
```

- **Generator** (`generator.Generator`) produces `*types.LoadTx` values. Each
  `Generate()` call draws from the seeded PRNG sub-streams (accounts, gas, key/
  size distributions) — this is the only place workload randomness is consumed.
- **Dispatcher** (`sender.Dispatcher`) owns the arrival timing. It runs in one of
  two arrival models (below) and hands each tx to the sender.
- **Sharded sender** (`sender.ShardedSender`, satisfies `sender.TxSender`) routes
  each tx to one of N per-endpoint workers by shard. `Send` enqueues into the
  worker's channel and returns immediately — it is asynchronous.
- **Workers** (`sender.Worker`) each own one RPC client to one endpoint and run
  `Tasks` send goroutines over a shared channel. The send goroutine stamps
  `AttemptedSendTime`, then calls go-ethereum `eth_sendRawTransaction`
  **synchronously**.
- **Sei RPC** is the SUT. The send returns nil (accepted) or an error (rejected).

A single shared `golang.org/x/time/rate.Limiter` is the one rate authority for
the whole pipeline. In closed-loop the worker gates on it; in open-loop the
scheduler reads it as a clock source (see below). When ramping is enabled, a
`Ramper` drives the limiter's limit up or down via `SetLimit`.

Optionally, when `--track-receipts` is set, successful sends are handed to a
block-indexed `stats.InclusionTracker` that observes on-chain inclusion by
scanning arriving blocks (O(blocks), not per-tx receipt polling). See
[06-measurement-metrics.md](06-measurement-metrics.md).

## The arrival model: why open-loop exists

The dispatcher supports two arrival models, selected by `arrivalModel`
(`sender.ArrivalModel`, values `"closed_loop"` / `"open_loop"`).

### Coordinated omission (the problem)

In the legacy **closed-loop** model the dispatcher generates the next tx only
once a sender is free (`runClosedLoop`: generate-then-send in lockstep). The
dequeue clock is therefore the SUT's clock: **when the SUT slows, the generator
slows with it and simply stops issuing the requests that would have observed the
slowdown.** The latency histogram under-reports, because the worst-affected
requests were never sent. This is **coordinated omission** — the closed-loop
model lies about latency precisely when the answer matters most (under stress).

### Open-loop (the fix)

The **open-loop** model decouples the arrival clock from sender availability
(`sender.openLoopScheduler`). Transaction `i` is scheduled at a fixed instant
**`t₀ + i/λ`**, where `t₀` is the run start and `λ` is the target rate, regardless
of whether any sender is free.

Properties that make it honest:

- **Absolute-instant scheduling.** The scheduler sleeps until each absolute
  instant (`SleepUntil(nextSend)`), not for a relative gap, so per-tx scheduling
  slop cannot accumulate into clock drift over a long run.
- **λ as a clock, not a gate.** λ is sampled from the shared limiter on each step
  (`limiter.Limit()`), so a ramping rate is honored; at fixed λ the running sum
  telescopes to exactly `t₀ + i/λ`. The limiter is read here as a clock source —
  the schedule advances whether or not the SUT keeps up.
- **Bounded in-flight + drop-and-count.** The arrival clock is **never throttled
  by backpressure** (throttling would reintroduce coordinated omission). Instead
  a counting semaphore bounds true in-flight sends to `maxInFlight`. At each
  scheduled instant the scheduler does a non-blocking `TryAcquire`: if senders are
  saturated the tick is **dropped and counted** and the clock moves on. The permit
  is held across the full unacked-in-flight window (enqueue + RPC round-trip) and
  released only after the synchronous send returns (via `tx.OnComplete`), so
  `maxInFlight` bounds real in-flight work and the drop count measures genuine
  load shed, not buffer geometry.
- **Admit before generate.** The permit is acquired **before** the generator is
  drawn. A dropped tick draws no tx (no seeded-stream consumption, no signer CPU),
  which makes admitted txs a deterministic prefix of the seeded sequence — see
  [05-reproducibility.md](05-reproducibility.md).

Closed-loop is retained only as the **legacy regression baseline**. For any
experiment where tail latency under load matters, use open-loop.

To use open-loop: set `arrivalModel: "open_loop"` and a finite positive rate
(`tps > 0` or `rampUp: true`); validation rejects open-loop with no finite λ.
See [03-config-reference.md](03-config-reference.md).

### Conservation (how counts must add up)

Every scheduled tick reaches exactly one terminal state, and the dispatcher folds
these into the run summary:

```
scheduled = dropped + admitted
admitted  = succeeded + failed
```

- **dropped** — shed because in-flight was saturated at the scheduled instant
  (never admitted, never sent).
- **admitted** — took a permit and drew a tx.
- **succeeded** — admitted, send returned nil (`DispatcherStats.TotalSent`).
- **failed** — admitted, send returned an error. **Counted, never lost**
  (`DispatcherStats.Failed`); a send error does not tear down the run.

In closed-loop, `Failed` and `Dropped` are always 0.

A finite workload ends when the generator drains; the terminal probe that
discovers this advances neither clock, index, nor counters. On a clean drain
`admitted == succeeded + failed` holds exactly. On `ctx` cancel (SIGTERM /
duration limit) some admitted txs may still be buffered for a worker and exit
uncounted — a bounded undercount that never affects a cleanly completed run.

## Measurement philosophy

**The generator emits measurements; it does not pronounce verdicts.** SLO
judgments, A/B comparisons, and pass/fail decisions are computed **externally**
via metric queries against the telemetry the tool emits — they are not owned by
sei-load. This shapes how you consume outputs:

- Treat sei-load output as raw signal (counters, histograms, the run summary),
  not as a graded result.
- Build your verdict logic in your query/analysis layer, gating on the run-level
  arrival model (see next point).
- **A tx cannot self-describe which model produced it.** An open-loop and a
  closed-loop `LoadTx` are byte-identical; coordinated-omission safety is a
  property of the run's arrival model, not of any per-tx field. Latency and
  schedule-lag consumers **must gate on the run-level `arrivalModel`** before
  trusting a latency or schedule-lag sample. In closed-loop, `IntendedSendTime`
  is merely the back-pressured enqueue time, so derived latency is omitted /
  meaningless.

> **`schedule_lag` is a concept, not a metric on main today.** It is the
> coordinated-omission/validity quantity `AttemptedSendTime − IntendedSendTime`,
> computed and judged **externally** — there is no `schedule_lag` series on
> `/metrics` (the emitter was punted as PLT-463). Do not write a query against it;
> see [06-measurement-metrics.md](06-measurement-metrics.md#34-detecting-a-generator-bound-invalid-run--schedule_lag)
> for the external validity heuristics that stand in for it.

## Glossary

| Term | Meaning |
|---|---|
| **λ (lambda)** | Target arrival rate (tx/s). In open-loop, sampled from the shared limiter each step as a clock source; the inter-arrival gap is `1/λ`. |
| **t₀** | Run start instant; the anchor for the open-loop schedule. |
| **intended send time** | `IntendedSendTime` = `t₀ + i/λ`, the true scheduled instant (open-loop). In closed-loop it is the enqueue time instead — not a real schedule. |
| **attempted send time** | `AttemptedSendTime`, the wall clock when a worker actually called the RPC. |
| **inclusion time** | `InclusionTime`, the header-arrival wall clock of the block that included the tx (set only when `--track-receipts`). |
| **schedule_lag** | `AttemptedSendTime − IntendedSendTime`. The primary coordinated-omission gate: it shows sends falling behind the arrival schedule even before any tx is shed. Open-loop only. **A concept, not a metric on main** — computed/judged externally; not a queryable series (emitter punted as PLT-463). |
| **SequenceIndex** | The arrival-tick index `i`. Monotonic; under drops it is non-contiguous across admitted txs (dropped ticks advance `i` and the clock but consume no draw). |
| **admitted** | A tick that took an in-flight permit and drew a tx. |
| **dropped** | A tick shed because in-flight was saturated (drop-and-count). |
| **failed** | An admitted tx whose send returned an error (counted, not lost). |
| **in-flight** | Concurrent unacked sends, bounded by `maxInFlight` via the semaphore; a permit is held enqueue → RPC return. |
| **drop-and-count** | The open-loop overload policy: shed and tally overdue ticks rather than throttle the arrival clock. |

## See also

- [02-running.md](02-running.md) — invoking a run.
- [03-config-reference.md](03-config-reference.md) — every config/CLI setting.
- [04-workload-model.md](04-workload-model.md) — scenarios, distributions, accounts.
- [05-reproducibility.md](05-reproducibility.md) — seeds, sub-streams, A/B.
- [06-measurement-metrics.md](06-measurement-metrics.md) — emitted metrics and the run summary.
- [07-experiment-playbook.md](07-experiment-playbook.md) — recipes for common experiments.
