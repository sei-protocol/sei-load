# Limits & Accepted Boundaries

> [← AGENTS.md index](../AGENTS.md)

> What this covers: the known, accepted measurement boundaries in sei-load's send and inclusion paths — what each is, when it bites, why it's accepted, and the counter to check before trusting a run. When an agent needs it: interpreting results, especially deciding whether a non-zero counter invalidates a conclusion or is benign.

Every boundary below is **accepted by design** and bounded. The contract is conservative: where the tooling can be wrong, it is wrong in a known direction (almost always *undercounting* inclusions, never inventing them). An agent reading a run should treat a non-zero boundary counter as a *confidence discount in a known direction*, not as silent corruption. Grounded in `sender/doc.go`.

> **Metric names here are conceptual.** Names like `block_gaps`, `dropped_at_cap`, `dropped`, `failed` are the *concepts* to check; the exact queryable series carry the `seiload_` prefix + Prometheus suffixes (e.g. `seiload_block_gaps_total`, `seiload_inclusion_outcome_total{outcome="dropped_at_cap"}`). See [Measurement & Metrics §2](06-measurement-metrics.md) for the authoritative catalog before writing a query.

## Send path

### Open-loop shutdown boundary

- **What:** On a clean drain (generator exhaustion), `admitted == succeeded + failed` holds exactly. On `ctx` cancel (SIGTERM or `--duration` expiry), txs already admitted and buffered for a worker can exit **uncounted** (`sender/doc.go:72-75`).
- **When it bites:** Only on cancellation-terminated runs — duration-bounded or interrupted. Never on a run that ends because the workload drained.
- **Why accepted:** The undercount is bounded by the worker channel backlog (a small fixed buffer), and the conservation identity is exact on clean completion.
- **How to interpret:** If the run ended by duration/SIGTERM and `admitted ≠ succeeded + failed`, the gap is shutdown buffer, not lost load — bounded by backlog. For exact conservation, end runs by generator drain (finite workload) rather than by duration. Check the `dropped` and `failed` gauges in the run summary.

Related send-path lenses (not boundaries):
- `schedule_lag` (`AttemptedSendTime − IntendedSendTime`) — ⚠️ **a concept, NOT an emitted metric on main**: there is no `schedule_lag` series to query (emitter punted as PLT-463); judge it externally via the [06 §3.4](06-measurement-metrics.md#34-detecting-a-generator-bound-invalid-run--schedule_lag) heuristics. Conceptually it is the primary coordinated-omission gate: non-zero/growing lag means sends are falling behind the open-loop arrival schedule *before* any tx is shed, and latency conclusions are suspect once it's large (`sender/doc.go:119-124`).
- `dropped` — genuine load shed once `maxInFlight` saturates (drop-and-count). This is real backpressure, not buffer geometry (`sender/doc.go:36-48`).
- `failed` — sends that returned a non-nil error; counted, never lost (`sender/doc.go:62-70`).

Conservation to assert per run: `scheduled == dropped + admitted` and `admitted == succeeded + failed` (the latter exact only on clean drain).

## Inclusion tracking (`--track-receipts`)

Inclusion is observed block-by-block by the `InclusionTracker`, not by per-tx receipt polling: it subscribes to new heads, fetches each arriving block body **once**, and stamps `InclusionTime` on matched in-flight txs (`sender/doc.go:88-97`). Conservation: `registered == included + expired + inflight_at_shutdown`, and `registered ⊆ succeeded` — only successful sends are registered, and the inclusion denominator is `succeeded` (`txs_accepted`), never a minted series (`sender/doc.go:99-103`).

The six accepted boundaries (`sender/doc.go:105-117`):

### 1. WebSocket head gaps

- **What:** A missed new-head subscription event is counted (`block_gaps`) but **never backfilled**. Txs in the missed block are not matched and eventually reap as `expired`.
- **When it bites:** Flaky WS connection, or head-arrival faster than the subscriber drains.
- **Why accepted:** Degrades conservatively — an *undercount of inclusions*, never a miscount.
- **Interpret:** Non-zero `block_gaps` ⇒ reported inclusion rate is a **lower bound**; true inclusion is ≥ reported. Don't read an inclusion shortfall as chain-side drops without first checking `block_gaps`.

### 2. Reorg first-observation-wins

- **What:** On a reorg the tracker uses first-observation-wins (stamp `InclusionTime` + delete from in-flight); there is no canonical-chain reconciliation.
- **When it bites:** Chain reorgs during the run.
- **Why accepted:** Inclusion-time error is bounded by `reorg_depth × block_time`.
- **Interpret:** If the SUT reorged, inclusion-latency samples carry up to `reorg_depth × block_time` of error. On a stable chain this is zero. Treat inclusion *latency* (not the count) as the affected metric.

### 3. Single fetch endpoint

- **What:** Block bodies are fetched from one endpoint only — `Endpoints[0]`, shared with the block collector.
- **When it bites:** Always present; it adds a small read load to that one node and ties inclusion observation to that node's view.
- **Why accepted:** Small added load; single consistent view.
- **Interpret:** `Endpoints[0]` is the inclusion oracle. If you multi-target sends across endpoints, inclusion is still judged from `Endpoints[0]`'s chain view. Note that contract scenarios also deploy/bind against `Endpoints[0]` (`generator/scenarios/*.go` `Attach`).

### 4. Header-arrival clock

- **What:** `InclusionTime` is the **header-arrival wall-clock** at the tracker — not fetch-completion time, and not `header.Time` (the block's own timestamp).
- **When it bites:** Always; it's the definition of the inclusion timestamp.
- **Why accepted:** It's the measurable instant closest to "the tracker learned this block exists."
- **Interpret:** `inclusion_latency = InclusionTime − IntendedSendTime` includes network propagation to the tracker. It is **open-loop-only**: in closed-loop, `IntendedSendTime` is enqueue time, so the latency sample is omitted (counts still tracked) (`sender/doc.go:94-97`). Do not compare inclusion-latency across arrival models, and do not equate it with on-chain block timestamp deltas.

### 5. Failed block fetch

- **What:** A failed block-body fetch is counted (`block_fetch_errors`) and **not retried**; that block's txs reap as `expired`.
- **When it bites:** Transient RPC errors fetching a body from `Endpoints[0]`.
- **Why accepted:** Same conservative undercount as a WS gap (boundary 1).
- **Interpret:** Non-zero `block_fetch_errors` ⇒ inclusion is again a lower bound. Sum it with `block_gaps` when judging how much of an inclusion shortfall is observational vs. real.

### 6. Late register / dropped-at-cap

- **What (late register):** A tx registered *after* its including block was already scanned is missed and reaps as `expired` — bounded by the microsecond register window vs. block time (a rare conservative undercount, same direction as a WS gap) (`sender/doc.go:115-117`).
- **What (dropped-at-cap):** When the inclusion registry hits its cap, registrations are dropped and counted (`dropped_at_cap`); these txs are **excluded from the inclusion denominator** (`sender/doc.go:101-103`).
- **When it bites:** Late-register is rare (register window ≪ block time). `dropped_at_cap` bites under sustained inclusion backlog (registry can't keep up).
- **Why accepted:** Late-register undercount is microsecond-window-bounded; cap-drops are excluded from the denominator so they can't inflate or deflate the inclusion rate.
- **Interpret:** Non-zero `dropped_at_cap` ⇒ the inclusion rate is computed over fewer txs than `succeeded`; it's still correct *for the registered subset* but doesn't cover the whole run. If `dropped_at_cap` is large, raise the registry cap or lower the rate before trusting inclusion as run-wide.

### Inclusion summary

- `inclusion_latency` is **open-loop-only** (omitted, not zero, in closed-loop).
- `inflight_at_shutdown` is read only after both workers and tracker have joined (`sender/doc.go:103`), so it is a true terminal residual, not a race artifact.
- Master identity to assert: `registered == included + expired + inflight_at_shutdown`, with `registered ⊆ succeeded`.
- **Direction rule:** boundaries 1, 5, 6(late) all push inclusion *down*. If your run shows fewer inclusions than expected, check `block_gaps + block_fetch_errors + dropped_at_cap` **first** — that sum caps how much of the shortfall is observational before you attribute any of it to the SUT.

## See also

- [03-config-reference](03-config-reference.md) — `--track-receipts`, endpoints, registry cap settings.
- [06-measurement-metrics](06-measurement-metrics.md) — the counter series named above.
- [07-experiment-playbook](07-experiment-playbook.md) — how to design runs that keep these boundaries at zero.
