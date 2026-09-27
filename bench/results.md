# Benchmark results

`go run ./bench -record` appends a table here. Each row is a Veles program
in `bench/<name>/main.vs`, built with `--release`, next to a Go program
doing the same work (the runner checks both computed the same checksum).
Read the last column: Veles's time as a multiple of Go's. Record a run
whenever a commit touches the runtime or code generation.

Caveats worth keeping in mind when reading them:

- Since 2026-09-27 the executor runs tasks on one thread per core (D66
  stage 1): `parallel` spreads CPU work over cores; `channels` keeps a
  producer and consumer on one worker (runnext), as Go's scheduler does.
  `spawn` measures the per-task cost: 100k tasks of ~100 ns each from one
  producer, so more threads mostly add handoffs; it is within ~1.2× of
  Go since the per-worker run queues (D66 stage 2). `pipes` runs eight
  independent channel pairs at once: it scales only if channels that
  share nothing share no lock (each has its own since 2026-09-27; under
  the one runtime lock it took 1.2 s at 8 threads).
- `sha256`: Go's uses the CPU's SHA instructions; Veles's is plain Veles
  (checklist §11).
- One machine, one run; differences under ~10% are noise.

## 2026-09-25 23:50 — go1.23.2, windows/amd64 (HEAD 77c6e95, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| channels | 200000 | 2.46ms | 27.72ms | 0.1× |
| json | 40000 | 314.07ms | 112.2ms | 2.8× |
| maps | 2000000 | 89.32ms | 188.53ms | 0.5× |
| sha256 | 16 | 301.17ms | 13.4ms | 22.5× |
| sort | 900000 | 295.31ms | 109ms | 2.7× |
| strings | 1000000 | 295.33ms | 97.8ms | 3.0× |
| trees | 14592688 | 191.08ms | 724.7ms | 0.3× |

Before the first recorded run the same day, the allocator scanned every
slot of every full span on each allocation between collections: `json`
took **376 s** (now 0.31 s) and `strings` did not finish in ten minutes
(now 0.3 s). `veles_gc.c` now skips a full span in O(1) and resumes each
size class where the last allocation succeeded.

## 2026-09-27 02:27 — go1.23.2, windows/amd64 (HEAD b9b7ef0, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| channels | 200000 | 7.04ms | 10.4ms | 0.7× |
| json | 40000 | 136.73ms | 47.27ms | 2.9× |
| maps | 2000000 | 50.33ms | 86.08ms | 0.6× |
| parallel | 64 | 16.73ms | 17.06ms | 1.0× |
| sha256 | 16 | 80.36ms | 7.65ms | 10.5× |
| sort | 900000 | 117.31ms | 53.34ms | 2.2× |
| spawn | 100000 | 443.99ms | 32.37ms | 13.7× |
| strings | 1000000 | 185.31ms | 39.78ms | 4.7× |
| trees | 14592688 | 146.47ms | 318.18ms | 0.5× |

## 2026-09-27 09:06 — go1.23.2, windows/amd64 (HEAD b9b7ef0, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| channels | 200000 | 7.21ms | 10.62ms | 0.7× |
| json | 40000 | 126.98ms | 41.79ms | 3.0× |
| maps | 2000000 | 46.6ms | 68.27ms | 0.7× |
| parallel | 64 | 16.32ms | 13.52ms | 1.2× |
| sha256 | 16 | 70.61ms | 7.2ms | 9.8× |
| sort | 900000 | 106.85ms | 49.25ms | 2.2× |
| spawn | 100000 | 40.08ms | 31.43ms | 1.3× |
| strings | 1000000 | 160.8ms | 33.99ms | 4.7× |
| trees | 14592688 | 130.94ms | 268.13ms | 0.5× |

## 2026-09-27 10:24 — go1.23.2, windows/amd64 (HEAD 544f12e, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| channels | 200000 | 4.56ms | 9.79ms | 0.5× |
| json | 40000 | 144.74ms | 46.21ms | 3.1× |
| maps | 2000000 | 43.93ms | 71.14ms | 0.6× |
| parallel | 64 | 15.66ms | 16.56ms | 0.9× |
| pipes | 1600000 | 13.48ms | 62.5ms | 0.2× |
| sha256 | 16 | 78.88ms | 8.83ms | 8.9× |
| sort | 900000 | 114.4ms | 51.38ms | 2.2× |
| spawn | 100000 | 42.3ms | 28.07ms | 1.5× |
| strings | 1000000 | 171.37ms | 34.86ms | 4.9× |
| trees | 14592688 | 139.69ms | 292.42ms | 0.5× |
