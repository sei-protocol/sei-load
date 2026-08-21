# Reproducibility

> [← AGENTS.md index](../AGENTS.md)

> What this covers / when an agent needs it. How to get reproducible workloads for
> fair A/B exploration: the seed → sub-stream derivation, the exact (and honest)
> determinism guarantee, how to set up an A/B run, and the open-loop property that
> keeps the admitted workload stable under SUT-driven drops. Read this before you
> compare two runs and attribute a difference to the change you made.

## The determinism guarantee — read precisely

sei-load gives you **per-stream draw multiset reproducibility**:

> Same seed + same config ⇒ identical per-stream draw multiset.

That is, the *distribution* of keys, sizes, gas values, and accounts is
statistically reproducible — which is exactly what fair A/B comparison requires.

**What is NOT guaranteed:**

- **Ordered, byte-identical replay above 1 worker.** With more than one worker,
  workers interleave their draws into the shared streams non-deterministically, so
  the ordered per-tx sequence differs run to run at the same seed (the multiset
  still matches).
- **On-chain arrival order** is concurrent regardless of worker count, so it is
  never reproducible.

**Ordered replay holds only at a single worker** (`workers: 1`,
`TasksPerEndpoint: 1`). If you need byte-for-byte deterministic emission ordering,
run with one worker. Otherwise, design your analysis around the multiset, not the
sequence. (Contract source: `utils/rng/rng.go` package doc.)

## Seed → sub-stream derivation

A run is rooted at one `seed`. Each logical consumer draws from its own
independent sub-stream, derived by the **FROZEN** formula:

```
substream(seed, streamID) = NewPCG(seed, splitmix64(fnv1a64(streamID)))
```

- `fnv1a64(streamID)` hashes the consumer name to a uint64.
- `splitmix64` diffuses it so near-identical names (e.g. `gas:0:base` /
  `gas:1:base`) seed well-separated PCG states.
- The result seeds a `math/rand/v2.PCG`.

**Worker-count independence.** Sub-streams are keyed by a *logical* stream id (a
string naming the consumer/purpose), never by a live-goroutine counter. So the
per-stream draw multiset a seed yields is invariant to `--workers`: adding workers
does not shift any stream's sequence.

### The FROZEN one-way-door contract

Changing the derivation breaks replay of every previously saved run. **Four
inputs are frozen** (`utils/rng/rng.go`), each a one-way door requiring a
`config_sha256` version bump:

1. The derivation formula (hash, diffusion, PCG argument order).
2. The set of stream-id strings (`utils/rng/streams.go`). The streamID feeds
   `fnv1a64`, so renaming any id reseeds that stream. Additions are append-only
   and do not perturb existing streams.
3. The per-stream draw order (e.g. drawing base before tip before feecap).
4. The per-tx account draw cadence: `sender` then `receiver` `NextAccount()` per
   tx (`generator/scenario.go`), each consuming the account stream.

Replay archives are keyed by `config_sha256`. If you (or a tool) change any frozen
input, do not expect old saved runs to replay — they will silently produce a
different draw sequence for the same `(seed, config)`.

### Stream IDs that exist

Defined in `utils/rng/streams.go`. `%d` is the scenario's config index `i`:

| Stream ID | Consumer |
|---|---|
| `accounts:shared` | shared (top-level) account pool (`StreamAccountsShared`) |
| `accounts:scenario:%d` | scenario `i`'s own account pool (`AccountsScenarioStream`) |
| `weighted:shuffle` | the weighted scenario selector's shuffle (`StreamWeightedShuffle`) |
| `gas:%d:base` | scenario `i`'s base-gas picker (`GasBaseStream`) |
| `gas:%d:tip` | scenario `i`'s tip-cap picker (`GasTipStream`) |
| `gas:%d:feecap` | scenario `i`'s fee-cap picker (`GasFeeCapStream`) |
| `dist:%d:key` | scenario `i`'s key-distribution index sampler (`KeyDistributionStream`) |
| `dist:%d:size` | scenario `i`'s size-distribution index sampler (`SizeDistributionStream`) |

## Setting the seed

The seed lives in the **config file**, not on the CLI. Set the top-level `seed`
field (`config.LoadConfig.Seed`, a `*uint64`):

```json
{
  "chainId": 1329,
  "endpoints": ["http://localhost:8545"],
  "seed": 42,
  "scenarios": [ /* ... */ ],
  "settings": { /* ... */ }
}
```

**Unset seed is randomized and recorded.** With no `seed`, the generator resolves
a cryptographically-random one, writes it back into the config, and logs it:

```
🎲 No seed configured; generated random seed 12345678901234567890 (set "seed" to replay)
```

To replay that run after the fact, copy the logged seed into the `seed` field and
re-run with the same config. (Source: `generator.resolveSeed`,
`rng.NewRandomSource`.) Note: the resolved seed is surfaced via the log line and
written back into the in-memory config — it is **not** a field on the emitted
`stats.RunSummary`, so capture it from the log if you need it.

## Running a reproducible A/B

1. **Pin the seed.** Set `seed` to a fixed value in both arms.
2. **Hold config constant** across the two arms — same scenarios, weights,
   distributions, account config, endpoints set.
3. **Vary exactly one axis** (the thing under test): e.g. `tps`, `maxInFlight`,
   `arrivalModel`, or a SUT-side change.
4. Compare the externally-computed metrics (this tool emits signal, not verdicts —
   see [01-mental-model.md](01-mental-model.md#measurement-philosophy)).

Because the workload is a fixed multiset at a fixed seed, a difference between
arms is attributable to the one axis you varied (plus concurrency noise above 1
worker — keep that in mind for tight comparisons; drop to `workers: 1` if you need
ordered determinism).

Changing scenarios, weights, distribution parameters (e.g. `theta`), account
config, or any frozen input changes the workload itself — that is no longer a fair
A/B of one axis.

## Open-loop determinism under drops

A critical property for stress experiments: in open-loop, **admitted txs are a
deterministic prefix of the seeded sequence**, because a dropped tick draws no tx
(the permit is acquired *before* `Generate()` — see
[01-mental-model.md](01-mental-model.md#open-loop-the-fix)).

Consequence: **the same seed yields the same admitted multiset regardless of how
many ticks SUT slowness forced to drop.** A faster SUT (fewer drops) and a slower
SUT (more drops) admit different *counts*, but the slower run's admitted set is a
prefix of the faster run's — the per-stream reproducibility contract holds under
saturation, where a draw-on-drop scheme would have broken it. `SequenceIndex` is
the arrival-tick index `i`: monotonic but non-contiguous across admitted txs under
drops (dropped ticks advance `i` and the clock while consuming no draw).

In closed-loop there is no such admission gate; the SUT speed governs how many
txs are generated, so the comparison anchor is weaker.

## See also

- [01-mental-model.md](01-mental-model.md) — pipeline, arrival models, glossary.
- [02-running.md](02-running.md) — invoking a run.
- [03-config-reference.md](03-config-reference.md) — every config/CLI setting.
- [04-workload-model.md](04-workload-model.md) — scenarios, distributions, accounts.
- [06-measurement-metrics.md](06-measurement-metrics.md) — emitted metrics and the run summary.
- [07-experiment-playbook.md](07-experiment-playbook.md) — recipes for common experiments.
