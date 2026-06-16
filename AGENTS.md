# sei-load — agent guide

sei-load drives synthetic transaction load at a Sei EVM endpoint and **emits measurements** (counters, histograms, a run summary) about how the system under test (SUT) responds. It is a load generator and a measurement instrument — **not** a judge: it computes no pass/fail verdicts, percentiles, or SLO compliance; you derive those externally from the signals it emits. These docs are the operating manual for an agent that designs, runs, and interprets sei-load experiments and acts on the results.

## Start here (reading order for a new agent)

Read linearly the first time:

1. [docs/01-mental-model.md](docs/01-mental-model.md) — the pipeline, open- vs closed-loop, coordinated omission, and the measure-not-judge philosophy. **Read this first.**
2. [docs/02-running.md](docs/02-running.md) — build the binary, every CLI flag, the run lifecycle.
3. [docs/03-config-reference.md](docs/03-config-reference.md) — the JSON config schema behind those flags.
4. [docs/04-workload-model.md](docs/04-workload-model.md) — scenarios and the StorageRW contention/size/op axes.
5. [docs/06-measurement-metrics.md](docs/06-measurement-metrics.md) — the authoritative metric catalog and the conservation model; the PromQL you query.
6. [docs/05-reproducibility.md](docs/05-reproducibility.md) — seed → sub-stream determinism and fair A/B setup. **Read before 07:** the playbook's cardinal rule depends on the seed/fair-A/B mechanics defined here.
7. [docs/07-experiment-playbook.md](docs/07-experiment-playbook.md) — objective → knobs → read → interpret recipes.

Keep [docs/08-limits-boundaries.md](docs/08-limits-boundaries.md) as a reference — pull it in when a non-zero boundary counter forces you to discount a result.

## Table of contents

| Doc | Covers / when you need it |
|-----|---------------------------|
| [01-mental-model.md](docs/01-mental-model.md) | The send pipeline, open-loop vs closed-loop arrival, coordinated omission, conservation identities, and why the tool emits signal not verdicts. The conceptual floor — read before anything else. |
| [02-running.md](docs/02-running.md) | Building/invoking `seiload`, every CLI flag, settings precedence, the metrics endpoint, copy-pasteable invocations, and the run lifecycle. Need it when starting/stopping/reproducing a run. |
| [03-config-reference.md](docs/03-config-reference.md) | The complete JSON config schema — `LoadConfig`, `settings`, `scenarios`, `accounts`, `funding`, gotchas. Need it when authoring or editing a config. |
| [04-workload-model.md](docs/04-workload-model.md) | The scenario set, what each stresses, and the StorageRW key-contention / tx-size / op-mix axes plus what they probe on Sei's parallel executor. Need it when choosing a scenario and shaping load. |
| [05-reproducibility.md](docs/05-reproducibility.md) | Seed → sub-stream derivation, the exact determinism guarantee, fair A/B setup, open-loop determinism under drops. Need it before comparing two runs. |
| [06-measurement-metrics.md](docs/06-measurement-metrics.md) | The authoritative 19-instrument catalog, the conservation model, and the PromQL recipes for rates/percentiles/goodput/validity. Need it before writing any query or trusting a number. |
| [07-experiment-playbook.md](docs/07-experiment-playbook.md) | The reasoning layer: objective → knobs → validity → read → interpret → next move, with recipes for contention, size, and tail-latency experiments. Need it when designing a run. |
| [08-limits-boundaries.md](docs/08-limits-boundaries.md) | The accepted measurement boundaries (WS gaps, reorgs, single fetch endpoint, header-arrival clock, cap drops) and the counter to check for each. Need it when deciding whether a non-zero counter invalidates a conclusion. |

## Fastest path to a first experiment

1. Build and validate offline: `make build`, then a `--dry-run` invocation — see [docs/02-running.md](docs/02-running.md#common-invocations).
2. Run an open-loop, fixed-λ measurement with receipt tracking and follow the trustworthy-tail-latency recipe — see [docs/07-experiment-playbook.md](docs/07-experiment-playbook.md) §4, then validity-gate it with §5 before quoting any number.

## Standing caveats (true on `main` today)

- **StorageRW distribution/size/op axes require PLT-465 (#54, unmerged).** `keyDistribution`, `sizeDistribution`, `sizeBuckets`, `recordCount`, and `operations` parse but **do not affect generated transactions** on main — StorageRW emits a fixed scaffold (slot 0, empty pad, all-`rmw`). See [docs/04-workload-model.md](docs/04-workload-model.md).
- **`schedule_lag` is a concept, not a queryable metric** (emitter punted as PLT-463). Judge generator validity externally via the [06 §3.4](docs/06-measurement-metrics.md) heuristics.
- **`--report-path` writes a formatted text dump, not JSON** (schema-versioned JSON is PLT-467). The seed is **config-file-only** (no `--seed` flag).
- **Exported series carry a `seiload_` prefix and unit suffixes.** The Prometheus exporter sets `WithNamespace("seiload")` (configurable), so every series is prefixed `seiload_`, and OTel appends unit suffixes (`s`-unit → `_seconds`, etc.); histograms expose `_bucket`/`_sum`/`_count` and counters end `_total`. The wire names — not the instrument base names — are what you query; [docs/06](docs/06-measurement-metrics.md) §2 lists them.
- **The tool emits signal, not verdicts.** Every rate, percentile, and pass/fail is computed by you.
