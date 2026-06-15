// Package sender drives transactions from generation to the chain.
//
// The send path is a pipeline: a [generator.Generator] produces transactions;
// the [Dispatcher] times their arrival and hands each off to a [TxSender]; the
// [ShardedSender] routes each tx to one of N per-endpoint [Worker]s by shard;
// the worker's send loop stamps the attempt and calls the go-ethereum client
// (eth_sendRawTransaction). Inclusion, when tracked, is observed by the
// block-indexed [stats.InclusionTracker] (see Inclusion stage below), not by
// per-tx receipt polling. A shared [golang.org/x/time/rate.Limiter] is
// the single rate authority for the whole pipeline; the [Ramper] drives its
// limit up or down via SetLimit.
//
// # Open-loop arrival model
//
// The dispatcher supports two arrival models (see [ArrivalModel]). The open-loop
// model exists to eliminate coordinated omission from the latency measurement.
//
// Coordinated omission. In the legacy closed-loop model the dispatcher generates
// the next tx only once a sender is free, so the dequeue clock is the SUT's
// clock: when the system under test slows, the generator slows with it and
// simply stops issuing the requests that would have observed the slowdown. The
// latency histogram then under-reports, because the worst-affected requests were
// never sent. This is coordinated omission. The closed-loop model lies about
// latency precisely when the answer matters most.
//
// Open-loop fixes this by decoupling the arrival clock from sender availability.
// Transaction i is scheduled at a fixed instant t₀ + i/λ, where t₀ is the run
// start and λ is the target rate, regardless of whether any sender is free. The
// scheduler sleeps until each absolute instant rather than for a relative gap
// ("sleep 1/λ from now"), so per-tx scheduling slop cannot accumulate into clock
// drift over a long run. λ is sampled from the shared limiter on each step, so a
// ramping rate is honored; at a fixed λ the running sum telescopes to exactly
// t₀ + i/λ. The limiter is read here as a clock source, not as a permit gate —
// the schedule advances whether or not the SUT keeps up.
//
// Bounded in-flight and drop-and-count. The arrival clock is never throttled by
// backpressure; throttling it would reintroduce coordinated omission. Instead a
// counting semaphore bounds true in-flight sends to maxInFlight. At each
// scheduled instant the scheduler tries to acquire a permit without blocking: if
// the senders are saturated the tick is dropped and counted, and the clock moves
// on. The permit is not released at enqueue. The scheduler installs a release
// callback on tx.OnComplete, and the worker invokes it only after the
// synchronous send returns — note the two phases of the worker path: the enqueue
// into the worker's channel ([TxSender.Send]) is asynchronous and returns at
// once, but the RPC send itself is synchronous. So the permit is held for the
// full unacked-in-flight window (enqueue plus RPC round-trip), and maxInFlight
// bounds real in-flight work while the drop count measures genuine load shed,
// not buffer geometry.
//
// Admit before generate (determinism). The permit is acquired BEFORE the
// generator is drawn, not after. A dropped tick therefore calls neither
// Generate (no seeded-stream draw, no per-tx state advance) nor the signer (no
// wasted CPU on shed load). This makes the admitted transactions a deterministic
// PREFIX of the seeded generator sequence: the same seed yields the same
// admitted multiset regardless of how many ticks the SUT speed forced to drop —
// the per-stream reproducibility contract holds under saturation, where it
// otherwise would not. SequenceIndex is the arrival-tick index i (so
// IntendedSendTime = t₀ + i/λ holds); under drops it is monotonic but
// non-contiguous across admitted txs, because dropped ticks still advance i and
// the clock while consuming no draw.
//
// Conservation. Every scheduled tick reaches exactly one terminal state:
// dropped (not admitted) or admitted. Every admitted tx in turn completes as
// succeeded (nil send error) or failed (non-nil send error) — a failed send is
// counted, never lost. So scheduled == dropped + admitted and
// admitted == succeeded + failed. The dispatcher folds these into the run
// summary (dropped and failed gauges) for goodput accounting. A finite workload
// ends when the generator drains; the terminal tick that discovers this is a
// probe, not an arrival — it advances neither the clock nor the counters, so it
// is excluded from the conservation totals.
//
// Shutdown boundary (accepted, not drift). admitted == succeeded + failed holds
// on a clean drain (generator exhaustion). On ctx cancel (SIGTERM/duration),
// admitted txs still buffered for a worker exit uncounted; the undercount is
// bounded by the channel backlog and never affects a cleanly completed run.
//
// LoadTx lifecycle and scheduling. The scheduling-relevant fields of [types.LoadTx]
// follow its single-writer concurrency contract: each is written once by the
// goroutine that solely owns the tx at that stage, then is immutable as ownership
// transfers with the pointer across channels. The scheduler stamps IntendedSendTime
// (the true scheduled instant t₀ + i/λ) and SequenceIndex (the arrival index i)
// before hand-off; the worker stamps AttemptedSendTime at the real send. A tx
// cannot self-describe which model produced it — an open-loop and a closed-loop
// tx are byte-identical — so coordinated-omission safety is a property of the
// run's arrival model, not of any per-tx field. Latency and schedule-lag consumers
// must gate on the run-level arrival model.
//
// # Inclusion stage
//
// When enabled (--track-receipts), the worker hands each successful send to the
// [stats.InclusionTracker] at send-completion (after OnComplete, only on a nil
// send error). The tracker subscribes to new heads, fetches each arriving
// block's body once (O(blocks), not O(txs)), and stamps InclusionTime on every
// matched in-flight tx with the block's header-ARRIVAL time. Un-included txs are
// reaped as expired after reapAfter. inclusion_latency (arrival minus
// IntendedSendTime) is an open-loop-only measure; in closed-loop IntendedSendTime
// is enqueue time, so the latency sample is omitted (counts are tracked in both).
//
// Conservation. registered == included + expired + inflight_at_shutdown, and
// registered ⊆ succeeded (only successful sends are registered). The inclusion
// denominator is succeeded (txs_accepted), never a minted "registered" series;
// dropped_at_cap txs are excluded from it. inflight_at_shutdown is read only
// after both the workers and the tracker have joined.
//
// Accepted boundaries. (1) WS gaps degrade conservatively: a missed head is
// counted (block_gaps) but never backfilled, so its txs reap as expired —
// an undercount of inclusions, never a miscount. (2) Reorgs use
// first-observation-wins (stamp + delete); the inclusion-time error is bounded
// by reorg_depth × block_time, with no canonical reconciliation. (3) A single
// fetch endpoint (Endpoints[0], shared with the block collector) adds a small
// read load. (4) InclusionTime is the header-arrival wall clock, not fetch
// completion and not header.Time. (5) A failed block-body fetch is counted
// (block_fetch_errors) and not retried — that block's txs reap as expired, the
// same conservative undercount as a WS gap. (6) A tx registered after its
// including block was already scanned is missed and reaps as expired — bounded
// by the microsecond register window versus block time, a rare conservative
// undercount, the same direction as a WS gap.
//
// Detection and baseline. schedule_lag (AttemptedSendTime minus IntendedSendTime)
// is the primary coordinated-omission gate: it shows when sends fall behind the
// arrival schedule even before any tx is shed. The drop count measures only
// genuine shedding once in-flight saturates. A send error does not tear down the
// campaign — errors and drops are surfaced through counters, not by aborting the
// run. The closed-loop model is retained as the regression baseline.
package sender
