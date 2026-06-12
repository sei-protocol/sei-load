// Package sender drives transactions from generation to the chain.
//
// The send path is a pipeline: a [generator.Generator] produces transactions;
// the [Dispatcher] times their arrival and hands each off to a [TxSender]; the
// [ShardedSender] routes each tx to one of N per-endpoint [Worker]s by shard;
// the worker's send loop stamps the attempt and calls the go-ethereum client
// (eth_sendRawTransaction). Receipts, when tracked, are
// polled on a separate worker loop. A shared [golang.org/x/time/rate.Limiter] is
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
// counting semaphore bounds true in-flight sends to maxInFlight. At each tx's
// scheduled instant the scheduler tries to acquire a permit without blocking: if
// the senders are saturated the tx is dropped and counted, and the clock moves
// on. The permit is not released at enqueue. The scheduler installs a release
// callback on tx.OnComplete, and the worker invokes it only after the
// synchronous send returns — note the two phases of the worker path: the enqueue
// into the worker's channel ([TxSender.Send]) is asynchronous and returns at
// once, but the RPC send itself is synchronous. So the permit is held for the
// full unacked-in-flight window (enqueue plus RPC round-trip), and maxInFlight
// bounds real in-flight work while the drop count measures genuine load shed,
// not buffer geometry.
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
// Detection and baseline. schedule_lag (AttemptedSendTime minus IntendedSendTime)
// is the primary coordinated-omission gate: it shows when sends fall behind the
// arrival schedule even before any tx is shed. The drop count measures only
// genuine shedding once in-flight saturates. A send error does not tear down the
// campaign — errors and drops are surfaced through counters, not by aborting the
// run. The closed-loop model is retained as the regression baseline.
package sender
