# Benchmark results

`go run ./bench -record` appends a table here. Each row is a Veles program
in `bench/<name>/main.vs`, built with `--release`, next to a Go program
doing the same work (the runner checks both computed the same checksum).
Read the last column: Veles's time as a multiple of Go's. Record a run
whenever a commit touches the runtime or code generation.

Caveats worth keeping in mind when reading them:

- `channels` is not like for like: Veles's executor is single-threaded
  coroutines, Go's channel hands values between OS threads. It will move
  when the executor gets threads (checklist §1.3).
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
