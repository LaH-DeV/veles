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
- `lexer`, `ast`, `intern`, `emit` (2026-09-29, plan S2) are the shapes a
  compiler written in Veles would lean on: a byte-scanning tokeniser with a
  keyword map, a sealed-family tree built and walked with `when`, a
  string-interning map, and text emitted through one `StringBuilder`. The
  input text of `lexer` and `intern` is built before the clock starts, and
  the Go reference leaves the same work out (`goSetup`). On a quiet machine
  they were 1.5× (`lexer`), 0.6× (`ast`), 2.0× (`intern`) and 1.2× (`emit`)
  of Go; on a loaded one the Go side swung 2–3× between runs while Veles
  stayed put, so read a ratio only from a run where `maps` and `trees` are
  near their usual numbers. None is over 3×, so none opens a checklist item.
- `gzip`, `gunzip` (2026-10-04, D124): std/compress against Go's `compress/gzip` on 2 MiB of
  word-salad text (level 6; `gunzip` is eight decodes, CRC and length checked). The
  compressed bytes differ from Go's, so the checksum is of what comes back out. On this
  data Veles is 1.1× (compress) and 1.5× (decompress) of Go; on the prose of the docs
  (0.8 MB of Markdown) the first measurement was 2.3× / 4.6×, and after the CRC (slicing by
  eight) and `List.addAll`/`slice` (one `memcpy`) work 2.3× / ~3×: text with long matches
  is the friendly case, short ones cost a push each. Ratios: Veles's output is 0.7–1.7%
  smaller than Go's at level 6 (it compares stored, fixed and dynamic per block).
- `preempt`, `backedge` (2026-10-27, plan F21). `preempt` runs one busy
  loop per worker beside a task that sleeps 1 ms at a time; its table of
  worst lateness is the point (D145's ~10 ms slice against Go's
  asynchronous preemption), its time only says the loops ran. Both sides
  start one loop per worker, so `VELES_THREADS` and `GOMAXPROCS` must agree
  (both unset, or set alike) or the checksums differ. `veles run
  bench/preempt -- all` prints every case F21 measured. `backedge` is four
  tight loops: Go has neither a back-edge poll nor a vectorizer, so the
  ratio is not the poll's price; that one, against C, is in the 2026-10-27
  section.
- One machine, one run; differences under ~10% are noise.

## 2026-09-25 23:50 — go1.23.2, windows/amd64 (HEAD 77c6e95, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   2.46ms |  27.72ms |       0.1× |
| json      |    40000 | 314.07ms |  112.2ms |       2.8× |
| maps      |  2000000 |  89.32ms | 188.53ms |       0.5× |
| sha256    |       16 | 301.17ms |   13.4ms |      22.5× |
| sort      |   900000 | 295.31ms |    109ms |       2.7× |
| strings   |  1000000 | 295.33ms |   97.8ms |       3.0× |
| trees     | 14592688 | 191.08ms |  724.7ms |       0.3× |

Before the first recorded run the same day, the allocator scanned every
slot of every full span on each allocation between collections: `json`
took **376 s** (now 0.31 s) and `strings` did not finish in ten minutes
(now 0.3 s). `veles_gc.c` now skips a full span in O(1) and resumes each
size class where the last allocation succeeded.

## 2026-09-27 02:27 — go1.23.2, windows/amd64 (HEAD b9b7ef0, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   7.04ms |   10.4ms |       0.7× |
| json      |    40000 | 136.73ms |  47.27ms |       2.9× |
| maps      |  2000000 |  50.33ms |  86.08ms |       0.6× |
| parallel  |       64 |  16.73ms |  17.06ms |       1.0× |
| sha256    |       16 |  80.36ms |   7.65ms |      10.5× |
| sort      |   900000 | 117.31ms |  53.34ms |       2.2× |
| spawn     |   100000 | 443.99ms |  32.37ms |      13.7× |
| strings   |  1000000 | 185.31ms |  39.78ms |       4.7× |
| trees     | 14592688 | 146.47ms | 318.18ms |       0.5× |

## 2026-09-27 09:06 — go1.23.2, windows/amd64 (HEAD b9b7ef0, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   7.21ms |  10.62ms |       0.7× |
| json      |    40000 | 126.98ms |  41.79ms |       3.0× |
| maps      |  2000000 |   46.6ms |  68.27ms |       0.7× |
| parallel  |       64 |  16.32ms |  13.52ms |       1.2× |
| sha256    |       16 |  70.61ms |    7.2ms |       9.8× |
| sort      |   900000 | 106.85ms |  49.25ms |       2.2× |
| spawn     |   100000 |  40.08ms |  31.43ms |       1.3× |
| strings   |  1000000 |  160.8ms |  33.99ms |       4.7× |
| trees     | 14592688 | 130.94ms | 268.13ms |       0.5× |

## 2026-09-27 10:24 — go1.23.2, windows/amd64 (HEAD 544f12e, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   4.56ms |   9.79ms |       0.5× |
| json      |    40000 | 144.74ms |  46.21ms |       3.1× |
| maps      |  2000000 |  43.93ms |  71.14ms |       0.6× |
| parallel  |       64 |  15.66ms |  16.56ms |       0.9× |
| pipes     |  1600000 |  13.48ms |   62.5ms |       0.2× |
| sha256    |       16 |  78.88ms |   8.83ms |       8.9× |
| sort      |   900000 |  114.4ms |  51.38ms |       2.2× |
| spawn     |   100000 |   42.3ms |  28.07ms |       1.5× |
| strings   |  1000000 | 171.37ms |  34.86ms |       4.9× |
| trees     | 14592688 | 139.69ms | 292.42ms |       0.5× |

## 2026-09-27 12:36 — go1.23.2, windows/amd64 (HEAD b3651be, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   4.54ms |  10.42ms |       0.4× |
| json      |    40000 |  95.26ms |  48.36ms |       2.0× |
| maps      |  2000000 |  44.83ms |   71.9ms |       0.6× |
| parallel  |       64 |  17.02ms |  17.88ms |       1.0× |
| pipes     |  1600000 |  15.79ms |  71.83ms |       0.2× |
| sha256    |       16 |  83.24ms |   8.64ms |       9.6× |
| sort      |   900000 | 112.85ms |  50.42ms |       2.2× |
| spawn     |   100000 |  40.81ms |  29.33ms |       1.4× |
| strings   |  1000000 |  63.88ms |  36.05ms |       1.8× |
| trees     | 14592688 | 144.82ms | 299.07ms |       0.5× |

## 2026-09-27 12:43 — go1.23.2, windows/amd64 (HEAD b3651be, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |  12.79ms |  11.59ms |       1.1× |
| json      |    40000 |  95.32ms | 137.07ms |       0.7× |
| maps      |  2000000 |  46.66ms | 144.92ms |       0.3× |
| parallel  |       64 |   11.5ms |  24.18ms |       0.5× |
| pipes     |  1600000 |  17.71ms |  74.85ms |       0.2× |
| sha256    |       16 |  84.44ms |   9.25ms |       9.1× |
| sort      |   900000 |  34.66ms |  78.01ms |       0.4× |
| spawn     |   100000 |  36.64ms |  33.44ms |       1.1× |
| strings   |  1000000 |  65.94ms |  37.85ms |       1.7× |
| trees     | 14592688 | 155.12ms | 328.79ms |       0.5× |

## 2026-09-27 15:54 — go1.23.2, windows/amd64 (HEAD 807eeb0, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   4.59ms |  10.65ms |       0.4× |
| json      |    40000 |  44.62ms |  48.33ms |       0.9× |
| maps      |  2000000 |  45.63ms |  77.43ms |       0.6× |
| parallel  |       64 |   9.46ms |  13.97ms |       0.7× |
| pipes     |  1600000 |  14.96ms |  69.23ms |       0.2× |
| sha256    |       16 |  83.88ms |   8.65ms |       9.7× |
| sort      |   900000 |  36.12ms |  54.78ms |       0.7× |
| spawn     |   100000 |  33.69ms |  30.65ms |       1.1× |
| strings   |  1000000 |  51.56ms |  35.82ms |       1.4× |
| trees     | 14592688 | 149.38ms | 302.21ms |       0.5× |

`json` 95 → 45 ms (2.0× → 0.9× Go) and `strings` 60 → 52 ms (1.7× → 1.4×).
Measured by phase first: a float cost **1.2 µs to print** — the shortest
round-trip digits were found by up to 18 `snprintf` and 17 `strtod` calls,
~1 µs on the Windows CRT — and the decoder read its position through a
bounds-checked list cell on every byte, built a string before parsing an
integer, and pushed string bytes one at a time. Now: an exact fast path
for printing (the fewest decimals whose correctly rounded digits read
back, `fma` for the exact product; 1.5M random values agree with the old
search except exact ties, which now round half to even on every
platform), Clinger's fast path for parsing (4M strings agree with
`strtod` bit for bit), the decoder's cursor behind a pointer, integers
accumulated directly, a string without escapes copied once
(`listDecodeUtf8Range`), `writeQuoted` copying an escape-free string
whole, UTF-8 validation eight ASCII bytes at a time, and `split` as one
runtime pass (`veles_string_split`, parts sharing the text's bytes).
`examples/features/text` pins split and float edge cases.

## 2026-09-28 11:48 — go1.23.2, linux/amd64 (HEAD 76b8eb1, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| channels  |   200000 |   7.46ms |   7.35ms |       1.0× |
| json      |    40000 |  42.66ms |  53.29ms |       0.8× |
| maps      |  2000000 |  44.08ms |  84.89ms |       0.5× |
| parallel  |       64 |  24.98ms |  19.98ms |       1.2× |
| pipes     |  1600000 |  17.64ms |   62.9ms |       0.3× |
| sha256    |       16 |   86.2ms |    9.6ms |       9.0× |
| sort      |   900000 |  39.62ms |  53.32ms |       0.7× |
| spawn     |   100000 |  105.1ms |  26.87ms |       3.9× |
| strings   |  1000000 |  52.92ms |  43.94ms |       1.2× |
| trees     | 14592688 | 121.08ms | 291.97ms |       0.4× |

First run on Linux (WSL2 Ubuntu 24.04 on the same 32-thread machine as
the Windows runs, clang 18). Single-threaded work matches Windows within
noise. The executor does not: `spawn` is 3.9× Go here against 1.1× on
Windows the day before, and `parallel` 1.2× against 0.7× — the
per-task handoff and worker wake-ups cost more on Linux (futex-backed
condition variables through the runtime lock, vs SRW locks on Windows).
Checklist §3.1 has the item.

## 2026-09-29 14:37 — go1.23.2, windows/amd64 (HEAD d2dfde7, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| ast       |   861841 |  39.89ms |  91.08ms |       0.4× |
| channels  |   200000 |  12.71ms |  15.34ms |       0.8× |
| emit      |  1000000 |  71.07ms | 174.72ms |       0.4× |
| intern    |  1000000 |     37ms |  48.92ms |       0.8× |
| json      |    40000 |  65.15ms | 106.41ms |       0.6× |
| lexer     |  2100000 |  70.92ms |  184.9ms |       0.4× |
| maps      |  2000000 |  45.51ms | 182.46ms |       0.2× |
| parallel  |       64 |  11.33ms |  57.09ms |       0.2× |
| pipes     |  1600000 |  19.52ms |  69.52ms |       0.3× |
| sha256    |       16 |  87.59ms |  12.71ms |       6.9× |
| sort      |   900000 |  38.73ms |  93.06ms |       0.4× |
| spawn     |   100000 |  46.71ms |  41.02ms |       1.1× |
| strings   |  1000000 |  68.49ms | 145.12ms |       0.5× |
| trees     | 14592688 | 149.39ms |  806.4ms |       0.2× |

## 2026-10-01 — go1.23.2, windows/amd64 (HEAD 627814f, plus the working tree: B10, B11)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| ast       |   861841 |  40.36ms |  42.63ms |       0.9× |
| channels  |   200000 |    4.6ms |   9.69ms |       0.5× |
| emit      |  1000000 |  71.53ms |  56.68ms |       1.3× |
| intern    |  1000000 |  38.29ms |  34.96ms |       1.1× |
| json      |    40000 |  42.05ms |  47.34ms |       0.9× |
| lexer     |  2100000 |  61.55ms |  43.36ms |       1.4× |
| maps      |  2000000 |  47.04ms |  69.56ms |       0.7× |
| parallel  |       64 |  10.07ms |  12.76ms |       0.8× |
| pipes     |  1600000 |  14.55ms |  75.83ms |       0.2× |
| sha256    |       16 |  89.87ms |  10.52ms |       8.5× |
| sort      |   900000 |  34.34ms |  52.72ms |       0.7× |
| spawn     |   100000 |  40.74ms |  31.38ms |       1.3× |
| strings   |  1000000 |  52.86ms |  35.31ms |       1.5× |
| trees     | 14592688 | 160.72ms | 286.78ms |       0.6× |

D102 adds a modification count to every list and map header, bumped by each
change of length or order (the inline `push` included), and a load and compare
per step of a loop over a mutable list or map. Against HEAD on the same machine
(run alternately, twice each for the ones that moved) every benchmark was
within noise: sha256 84–94 ms at HEAD, 84–93 ms after; maps 44–56 vs 47–54;
intern 34–35 vs 34–38; sort 39.8 vs 34.3; json 48.6 vs 42.1. `spawn` swung
30–53 ms on both trees and touches nothing that changed.

## 2026-10-02 15:02 — go1.23.2, windows/amd64 (HEAD 9937139, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| adapters  | 24000000 |  53.82ms |  59.13ms |       0.9× |
| ast       |   861841 |  49.72ms |  132.8ms |       0.4× |
| channels  |   200000 |   9.28ms |  27.08ms |       0.3× |
| emit      |  1000000 |  66.51ms | 129.16ms |       0.5× |
| intern    |  1000000 |  39.89ms |  77.98ms |       0.5× |
| json      |    40000 |  43.63ms |  76.33ms |       0.6× |
| lexer     |  2100000 |  65.81ms | 175.34ms |       0.4× |
| maps      |  2000000 |  57.48ms | 111.31ms |       0.5× |
| parallel  |       64 |  21.63ms |  28.07ms |       0.8× |
| pipes     |  1600000 |  29.01ms |  74.58ms |       0.4× |
| sha256    |       16 |  99.13ms |  12.28ms |       8.1× |
| sort      |   900000 |   35.9ms |  75.64ms |       0.5× |
| spawn     |   100000 |  42.44ms |  37.75ms |       1.1× |
| strings   |  1000000 |  62.36ms | 138.93ms |       0.4× |
| trees     | 14592688 | 181.25ms | 937.03ms |       0.2× |

Plan B8 / D116 (the eager list and map adapters moved from compiler lowering
into the prelude, generic over `throws E` and conditional suspension). The
new `adapters` workload (map, filter, fold, any, all, find over 200k
integers, 20 rounds), three runs each: 64–73 ms lowered, 43–46 ms in the
prelude — `map` reserves its output once, which the lowering never did. The
other rows against HEAD in a worktree: within run-to-run noise (`lexer`
62–78 ms at HEAD, 64–69 ms now). The Go column swung 2× between runs on this
machine today; compare the Veles column.

## 2026-10-03 21:53 — go1.23.2, windows/amd64 (HEAD 3167e87, plus the working tree)

| benchmark | ops |   Veles |      Go | Veles / Go |
| --------- | --: | ------: | ------: | ---------: |
| sha256    |  16 | 86.39ms | 11.31ms |       7.6× |

`sha256` after std's SHA-256 moved its state, block buffer and message
schedule from `MutableList`s to `Array<u32, 8>`, `Array<u8, 64>` and a
`var w: Array<u32, 64>` per block (D121; the round constants are an
`Array<u32, 64>` table the bounds facts index without a check): 86–91 ms in
four runs against 113–135 ms before (four runs). Every other benchmark's IR is
unchanged by the memory class (the goldens did not move), so only this row
is expected to differ.

## 2026-10-04 12:55 — go1.23.2, windows/amd64 (HEAD 1b733d4, plus the working tree)

| benchmark |     ops |    Veles |       Go | Veles / Go |
| --------- | ------: | -------: | -------: | ---------: |
| gzip      | 2097154 | 140.88ms | 130.28ms |       1.1× |

## 2026-10-04 12:55 — go1.23.2, windows/amd64 (HEAD 1b733d4, plus the working tree)

| benchmark |      ops |   Veles |     Go | Veles / Go |
| --------- | -------: | ------: | -----: | ---------: |
| gunzip    | 16777232 | 81.02ms | 51.2ms |       1.6× |

## 2026-10-10 21:34 — go1.23.2, windows/amd64 (HEAD 6b71c14, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| adapters  | 24000000 |  72.12ms |  78.16ms |       0.9× |
| ast       |   861841 |  64.48ms |  95.48ms |       0.7× |
| channels  |   200000 |    9.5ms |  25.74ms |       0.4× |
| emit      |  1000000 | 114.35ms | 152.59ms |       0.7× |
| gunzip    | 16777232 | 128.89ms | 122.52ms |       1.1× |
| gzip      |  2097154 | 250.79ms | 335.77ms |       0.7× |
| httphello |    16000 | 372.33ms |   1.328s |       0.3× |
| idle      |   100000 | 135.97ms | 723.86ms |       0.2× |
| intern    |  1000000 |   64.8ms |  98.75ms |       0.7× |
| json      |    40000 |  74.09ms | 112.61ms |       0.7× |
| lexer     |  2100000 | 102.62ms | 165.79ms |       0.6× |
| maps      |  2000000 |  79.66ms | 174.97ms |       0.5× |
| parallel  |       64 |  13.73ms |  32.11ms |       0.4× |
| pipes     |  1600000 |  41.39ms | 108.15ms |       0.4× |
| sha256    |       16 | 101.38ms |   13.2ms |       7.7× |
| sort      |   900000 |   60.2ms | 112.59ms |       0.5× |
| spawn     |   100000 |  64.21ms |  47.39ms |       1.4× |
| strings   |  1000000 |  84.47ms | 101.54ms |       0.8× |
| suscall   | 10000000 | 210.62ms |  19.74ms |      10.7× |
| trees     | 14592688 | 260.84ms | 723.36ms |       0.4× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
| --------- | -------------: | ----------: | ---------: |
| idle      |            345 |        8721 |      0.04× |

## 2026-10-11 03:03 — go1.23.2, windows/amd64 (HEAD 76afdea, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| adapters  | 24000000 |  42.53ms |  27.15ms |       1.6× |
| ast       |   861841 |  40.11ms |  39.19ms |       1.0× |
| channels  |   200000 |   5.71ms |  11.84ms |       0.5× |
| emit      |  1000000 |  75.76ms |  69.14ms |       1.1× |
| gunzip    | 16777232 |  82.78ms |  57.04ms |       1.5× |
| gzip      |  2097154 | 149.47ms | 138.72ms |       1.1× |
| httphello |    16000 | 207.75ms | 685.25ms |       0.3× |
| idle      |   100000 |  84.55ms | 347.21ms |       0.2× |
| intern    |  1000000 |  37.67ms |  40.46ms |       0.9× |
| json      |    40000 |   45.2ms |  50.62ms |       0.9× |
| lexer     |  2100000 |  82.03ms |  94.05ms |       0.9× |
| maps      |  2000000 |  46.44ms |  71.72ms |       0.6× |
| parallel  |       64 |   7.51ms |  13.17ms |       0.6× |
| pipes     |  1600000 |  27.49ms |   64.3ms |       0.4× |
| sha256    |       16 |  62.33ms |   8.09ms |       7.7× |
| sort      |   900000 |  36.99ms |  56.22ms |       0.7× |
| spawn     |   100000 |  39.64ms |  28.03ms |       1.4× |
| strings   |  1000000 |  50.83ms |  61.75ms |       0.8× |
| suscall   | 10000000 | 127.26ms |   5.74ms |      22.2× |
| trees     | 14592688 | 162.74ms |  327.6ms |       0.5× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
| --------- | -------------: | ----------: | ---------: |
| idle      |            249 |        8723 |      0.03× |

## 2026-10-13 19:01 — go1.23.2, windows/amd64 (HEAD ea45784, plus the working tree)

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| adapters  | 24000000 |  61.52ms |     72ms |       0.9× |
| ast       |   861841 |  46.46ms |  91.41ms |       0.5× |
| channels  |   200000 |   6.08ms |   11.3ms |       0.5× |
| emit      |  1000000 |  92.45ms | 110.07ms |       0.8× |
| gcpause   | 20000000 | 752.93ms | 627.02ms |       1.2× |
| gunzip    | 16777232 | 112.24ms |    101ms |       1.1× |
| gzip      |  2097154 | 155.78ms | 227.32ms |       0.7× |
| httphello |    16000 | 246.61ms | 440.66ms |       0.6× |
| idle      |   100000 |   93.7ms | 544.67ms |       0.2× |
| intern    |  1000000 |  47.18ms |  62.89ms |       0.8× |
| json      |    40000 |  52.12ms |  75.59ms |       0.7× |
| lexer     |  2100000 | 100.02ms | 119.97ms |       0.8× |
| maps      |  2000000 |  82.88ms | 131.69ms |       0.6× |
| parallel  |       64 |   6.56ms |   26.6ms |       0.2× |
| pipes     |  1600000 |   29.9ms |  75.44ms |       0.4× |
| sha256    |       16 |  74.34ms |   8.98ms |       8.3× |
| sort      |   900000 |  44.92ms |  89.91ms |       0.5× |
| spawn     |   100000 |  37.43ms |  34.78ms |       1.1× |
| strings   |  1000000 |  71.18ms |  85.34ms |       0.8× |
| suscall   | 10000000 | 134.75ms |  17.88ms |       7.5× |
| trees     | 14592688 | 180.16ms | 523.47ms |       0.3× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
| --------- | -------------: | ----------: | ---------: |
| idle      |            249 |        8725 |      0.03× |

| benchmark | Veles longest pause | Go longest pause |
| --------- | ------------------: | ---------------: |
| gcpause   |              1.01ms |           4.06ms |

## 2026-10-20 10:15 — go1.23.2, windows/amd64 (HEAD d029143, plus the working tree)

| benchmark |     ops |   Veles |      Go | Veles / Go |
| --------- | ------: | ------: | ------: | ---------: |
| emptycoll | 2000000 | 71.74ms | 18.39ms |       3.9× |

`emptycoll` (D187) is two million records with an empty `MutableList`, an empty `MutableMap` and an
empty `List` field, one in eight used. Before D187 (empty values allocated their storage and every
`[]` its own header) it ran in about 205 ms (10.7–11.3× Go, three runs); after, about 73 ms
(three runs within 4%). The Go column is Go with escape analysis: its record never leaves the
loop, so nothing is heap-allocated. The same Go record stored in a global (heap-allocated, like
every Veles collection) takes about 105 ms, so the allocators are even or Veles is ahead; the gap
to the table row is stack allocation, which Veles did not have at this row. Since D188 the header of a
collection that never leaves its function is a stack slot (`sema/escape_hir.go`): the next row is
that run: 20–28 ms over several runs (1.0× Go in the recorded one).

## 2026-10-20 13:11 — go1.23.2, windows/amd64 (HEAD 6de3a52, plus the working tree)

| benchmark |     ops |   Veles |      Go | Veles / Go |
| --------- | ------: | ------: | ------: | ---------: |
| emptycoll | 2000000 | 20.32ms | 20.22ms |       1.0× |

## 2026-10-21 17:14 — go1.23.2, darwin/arm64 (HEAD cef7a4f, plus the working tree)

macOS A7, Apple clang 21, native kqueue, default worker count. All checksums
match Go. Marking remains in the pause on macOS (15.96 ms longest here).
Before/after reactor comparison with Go 1.27.1: poll 369.11 ms before;
kqueue 392.53, 393.73 and 388.24 ms after. This 5–7% difference is within
the runner's 10% noise threshold, not evidence of a speedup.

| benchmark |      ops |    Veles |       Go | Veles / Go |
| --------- | -------: | -------: | -------: | ---------: |
| adapters  | 24000000 |  57.74ms |  21.81ms |       2.6× |
| ast       |   861841 |   43.8ms |  37.67ms |       1.2× |
| channels  |   200000 |   9.42ms |   8.44ms |       1.1× |
| emit      |  1000000 |   55.2ms |  39.31ms |       1.4× |
| emptycoll |  2000000 |  21.26ms |  19.87ms |       1.1× |
| gcpause   | 20000000 | 355.79ms | 249.06ms |       1.4× |
| gunzip    | 16777232 |  48.36ms |  50.77ms |       1.0× |
| gzip      |  2097154 | 138.47ms |  99.88ms |       1.4× |
| httphello |    16000 | 386.13ms | 452.27ms |       0.9× |
| idle      |   100000 |  97.31ms | 206.72ms |       0.5× |
| intern    |  1000000 |  36.76ms |  32.34ms |       1.1× |
| json      |    40000 |  33.35ms |  33.29ms |       1.0× |
| lexer     |  2100000 |  54.97ms |  95.74ms |       0.6× |
| maps      |  2000000 |  29.64ms |  84.42ms |       0.4× |
| parallel  |       64 |   9.94ms |  34.68ms |       0.3× |
| pipes     |  1600000 |  48.71ms |  69.24ms |       0.7× |
| sha256    |       16 |  60.34ms |   6.48ms |       9.3× |
| sort      |   900000 |  48.33ms |  52.02ms |       0.9× |
| spawn     |   100000 |   54.1ms |  25.71ms |       2.1× |
| strings   |  1000000 |  64.72ms |  57.69ms |       1.1× |
| suscall   | 10000000 | 142.05ms |   4.89ms |      29.0× |
| trees     | 14592688 | 165.46ms | 260.98ms |       0.6× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
| --------- | -------------: | ----------: | ---------: |
| idle      |            248 |        2605 |      0.10× |

| benchmark | Veles longest pause | Go longest pause |
| --------- | ------------------: | ---------------: |
| gcpause   |             15.96ms |            900µs |

## 2026-10-21 17:14 — go1.23.2, darwin/arm64 (HEAD cef7a4f, plus the working tree)

`VELES_THREADS=1`, native kqueue.

| benchmark |   ops |    Veles |       Go | Veles / Go |
| --------- | ----: | -------: | -------: | ---------: |
| httphello | 16000 | 789.63ms | 445.98ms |       1.8× |

## 2026-10-21 17:14 — go1.23.2, darwin/arm64 (HEAD cef7a4f, plus the working tree)

`VELES_THREADS=8`, native kqueue.

| benchmark |   ops |    Veles |       Go | Veles / Go |
| --------- | ----: | -------: | -------: | ---------: |
| httphello | 16000 | 383.81ms | 445.99ms |       0.9× |

## 2026-10-23 00:13 — go1.23.2, windows/amd64 (HEAD c6080ab, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| adapters | 24000000 | 57.29ms | 30.68ms | 1.9× |
| ast | 861841 | 39.77ms | 40.3ms | 1.0× |
| channels | 200000 | 5.1ms | 10.77ms | 0.5× |
| emit | 1000000 | 82.07ms | 57.55ms | 1.4× |
| emptycoll | 2000000 | 19.69ms | 18.76ms | 1.0× |
| gcpause | 20000000 | 448.31ms | 388.57ms | 1.2× |
| gunzip | 16777232 | 90.92ms | 56.33ms | 1.6× |
| gzip | 2097154 | 144.22ms | 131.16ms | 1.1× |
| httphello | 16000 | 196.85ms | 690.2ms | 0.3× |
| idle | 100000 | 89.71ms | 331.46ms | 0.3× |
| intern | 1000000 | 43.34ms | 31.14ms | 1.4× |
| json | 40000 | 42.76ms | 46.81ms | 0.9× |
| lexer | 2100000 | 84.68ms | 95.74ms | 0.9× |
| maps | 2000000 | 69.02ms | 76.92ms | 0.9× |
| parallel | 64 | 11.1ms | 17.07ms | 0.7× |
| pipes | 1600000 | 38.75ms | 78.61ms | 0.5× |
| sha256 | 16 | 145.64ms | 9.9ms | 14.7× |
| sort | 900000 | 60.35ms | 131.8ms | 0.5× |
| spawn | 100000 | 50.06ms | 35.88ms | 1.4× |
| strings | 1000000 | 74.09ms | 58.24ms | 1.3× |
| suscall | 10000000 | 125.55ms | 6.95ms | 18.1× |
| trees | 14592688 | 164.82ms | 335.1ms | 0.5× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
|---|---:|---:|---:|
| idle | 249 | 8710 | 0.03× |

| benchmark | Veles longest pause | Go longest pause |
|---|---:|---:|
| gcpause | 1.36ms | 1.72ms |

Run after codegen started emitting only reachable functions (`codegen/llvm/prune.go`).
Function bodies are unchanged, so run time should be too: the Veles column is within noise
of the 2026-10-13 Windows run except `sha256` (146 ms here). Built with the old and the new
compiler side by side, it was 60–64 ms for both over three runs with the same checksum, so
that row was a noisy run, not a regression.

Build time, `veles build --timings` (`-O0 -g`, runtime objects cached), three runs each:

| program | IR before | IR after | `define`s | clang before | clang after | build before | build after |
| ------- | --------: | -------: | --------: | -----------: | ----------: | -----------: | ----------: |
| hello world (`use io`, one `println`) | 1.3 MB | 28 KB | 540 → 4 | 443–453 ms | 226–258 ms | 513–530 ms | 295–325 ms |
| `examples/httpd` | 9.3 MB | 4.0 MB | 2156 → 822 | 2.86–2.91 s | 1.40–1.42 s | 3.21–3.25 s | 1.75–1.78 s |

Wall time of `go test -count=1 ./docs ./examples`: 59 s before, 32 s after (docs 57 → 29 s,
examples 45 → 25 s). In the full `go test ./...` on Windows, `driver` went from 586 s
(2026-10-22) to 325 s.

## 2026-10-23 02:31 — go1.23.2, windows/amd64 (HEAD f1461d8, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| awaitall | 50000 | 44.77ms | 62.89ms | 0.7× |
| awaitscope | 50000 | 41.66ms | 62.78ms | 0.7× |

New (F19): `awaitall` joins two inline launches with `awaitAll`, `awaitscope` the
same pairs in a `scope` with two handles. Four runs: 40.6–44.8 ms against
41.0–41.7 ms — the same cost; Go's two goroutines and a WaitGroup 63 ms.

## 2026-10-23 03:04 — go1.23.2, windows/amd64 (HEAD f1461d8, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| parse | 155 | 90.94ms | 54.91ms | 1.7× |

New (plan S7): `parse` is the self-hosted lexer and parser (`selfhost --bench-parse std`)
against the Go front end over std's 155 files. It started at 164 ms (3.0×); three changes
brought it to 85–91 ms (1.5–1.7×): `byteAt` compiled inline instead of a runtime call
(150 ms; also `lexer` 85 → 53 ms), the allocator no longer clears memory that is still
zero from the OS (130 ms; most benchmarks gain a little), and the self-hosted `Span`
holds its file behind a pointer instead of a copy per node (84 ms). The rest of the gap
is mostly the size of a sealed value: `ast.Expr` is an inline union of 180 bytes,
`Decl` 348 — a question about D12 put to the user.

## 2026-10-23 03:40 — go1.23.2, windows/amd64 (HEAD f1461d8, plus the working tree)

The whole suite after `byteAt` inline and the allocator change (before the `Span` change, so
`parse` reads 135 ms here). Against the 00:13 run: lexer 85 → 53 ms, adapters 57 → 48, emit
82 → 67, strings 74 → 54, sort 60 → 39, gunzip 91 → 73, spawn 50 → 37; the rest within noise
(`gcpause`'s longest pause read 2.17 ms once, 1.22–1.29 ms on three reruns).

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| adapters | 24000000 | 47.66ms | 31.34ms | 1.5× |
| ast | 861841 | 36.25ms | 38.44ms | 0.9× |
| awaitall | 50000 | 40.3ms | 61.09ms | 0.7× |
| awaitscope | 50000 | 40.78ms | 61.22ms | 0.7× |
| channels | 200000 | 5.13ms | 11.8ms | 0.4× |
| emit | 1000000 | 66.87ms | 54.35ms | 1.2× |
| emptycoll | 2000000 | 20.56ms | 16.68ms | 1.2× |
| gcpause | 20000000 | 412.45ms | 378.69ms | 1.1× |
| gunzip | 16777232 | 73.04ms | 57.84ms | 1.3× |
| gzip | 2097154 | 129.29ms | 116.52ms | 1.1× |
| httphello | 16000 | 191.65ms | 427.8ms | 0.4× |
| idle | 100000 | 86.45ms | 339.93ms | 0.3× |
| intern | 1000000 | 48.77ms | 36ms | 1.4× |
| json | 40000 | 46.5ms | 53.76ms | 0.9× |
| lexer | 2100000 | 52.72ms | 80.92ms | 0.7× |
| maps | 2000000 | 64.18ms | 76.31ms | 0.8× |
| parallel | 64 | 4.46ms | 12.83ms | 0.3× |
| parse | 155 | 134.93ms | 73.8ms | 1.8× |
| pipes | 1600000 | 23.88ms | 71.3ms | 0.3× |
| sha256 | 16 | 57.77ms | 7.4ms | 7.8× |
| sort | 900000 | 38.82ms | 50.45ms | 0.8× |
| spawn | 100000 | 37.04ms | 33.41ms | 1.1× |
| strings | 1000000 | 53.57ms | 49.44ms | 1.1× |
| suscall | 10000000 | 118.57ms | 5.87ms | 20.2× |
| trees | 14592688 | 171.71ms | 305.08ms | 0.6× |

## 2026-10-23 — go1.23.2, windows/amd64 (HEAD de836bb, plus the working tree): S16 / D198

Trees boxed (D198) and concurrent marking from 16 MiB live instead of 4. `parse` 69–74 ms
against Go 61–64 (1.1×; Linux 63–70 against 64–65, parity). The Go column of this run is
noisy (`parse` 112 ms, `trees` 472 ms); `ast` and `json`, A/B against HEAD three times
alternately, within noise (ast 36–42 vs 37–41 ms, json 38–58 vs 40–56 ms). `gcpause`'s longest
pause 1.32–1.45 ms over four runs, as at 4 MiB in the same session.

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| adapters | 24000000 | 52.15ms | 35.36ms | 1.5× |
| ast | 861841 | 44.43ms | 42.51ms | 1.0× |
| awaitall | 50000 | 38.82ms | 60.56ms | 0.6× |
| awaitscope | 50000 | 42.65ms | 62.32ms | 0.7× |
| channels | 200000 | 6.29ms | 10.69ms | 0.6× |
| emit | 1000000 | 70.16ms | 52.9ms | 1.3× |
| emptycoll | 2000000 | 26.68ms | 19.39ms | 1.4× |
| gcpause | 20000000 | 457.86ms | 613.05ms | 0.7× |
| gunzip | 16777232 | 76.08ms | 59.1ms | 1.3× |
| gzip | 2097154 | 143.62ms | 138.6ms | 1.0× |
| httphello | 16000 | 184.16ms | 494.28ms | 0.4× |
| idle | 100000 | 94.73ms | 399.27ms | 0.2× |
| intern | 1000000 | 37.74ms | 39.9ms | 0.9× |
| json | 40000 | 59.61ms | 59.72ms | 1.0× |
| lexer | 2100000 | 58.23ms | 128.01ms | 0.5× |
| maps | 2000000 | 61.58ms | 78.44ms | 0.8× |
| parallel | 64 | 5.07ms | 12.17ms | 0.4× |
| parse | 155 | 70.88ms | 111.7ms | 0.6× |
| pipes | 1600000 | 26.71ms | 69.01ms | 0.4× |
| sha256 | 16 | 63.32ms | 9.28ms | 6.8× |
| sort | 900000 | 38.8ms | 54.56ms | 0.7× |
| spawn | 100000 | 36.56ms | 30.99ms | 1.2× |
| strings | 1000000 | 55.68ms | 51.96ms | 1.1× |
| suscall | 10000000 | 126.02ms | 5.97ms | 21.1× |
| trees | 14592688 | 166.03ms | 472.46ms | 0.4× |
| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
|---|---:|---:|---:|
| idle | 249 | 8709 | 0.03× |
| benchmark | Veles longest pause | Go longest pause |
|---|---:|---:|
| gcpause | 1.37ms | 1.53ms |

## 2026-10-24 02:26 — go1.23.2, windows/amd64 (HEAD de836bb, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| parse | 155 | 52.01ms | 62.46ms | 0.8× |

## 2026-10-26 20:30 — go1.27.1, darwin/arm64 (HEAD d7eecc7, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| adapters | 24000000 | 53.94ms | 25.5ms | 2.1× |
| ast | 861841 | 52.08ms | 24.62ms | 2.1× |
| awaitall | 50000 | 94.23ms | 27.91ms | 3.4× |
| awaitscope | 50000 | 93.13ms | 26.59ms | 3.5× |
| bigargs | 40000000 | 46.55ms | 96.96ms | 0.5× |
| channels | 200000 | 13.93ms | 7.77ms | 1.8× |
| emit | 1000000 | 63.84ms | 37.01ms | 1.7× |
| emptycoll | 2000000 | 22.23ms | 19.32ms | 1.2× |
| gcpause | 20000000 | 364.24ms | 129.24ms | 2.8× |
| gunzip | 16777232 | 44.41ms | 51.13ms | 0.9× |
| gzip | 2097154 | 113.52ms | 15.98ms | 7.1× |
| httphello | 16000 | 389.44ms | 362.57ms | 1.1× |
| idle | 100000 | 97.15ms | 113.11ms | 0.9× |
| intern | 1000000 | 30.46ms | 14.99ms | 2.0× |
| json | 40000 | 52.46ms | 25.99ms | 2.0× |
| lexer | 2100000 | 37.44ms | 74.14ms | 0.5× |
| maps | 2000000 | 53.96ms | 51.18ms | 1.1× |
| parallel | 64 | 6.67ms | 32.21ms | 0.2× |
| parse | 163 | 37.44ms | 62.92ms | 0.6× |
| pipes | 1600000 | 51.37ms | 65.56ms | 0.8× |
| sha256 | 16 | 60.82ms | 6.27ms | 9.7× |
| sort | 900000 | 47.01ms | 49.82ms | 0.9× |
| spawn | 100000 | 64.49ms | 20.04ms | 3.2× |
| strings | 1000000 | 59.71ms | 47.69ms | 1.3× |
| suscall | 10000000 | 136.41ms | 4.71ms | 29.0× |
| trees | 14592688 | 151.61ms | 150.5ms | 1.0× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
|---|---:|---:|---:|
| idle | 248 | 2673 | 0.09× |

| benchmark | Veles longest pause | Go longest pause |
|---|---:|---:|
| gcpause | 14.79ms | 1.42ms |

## 2026-10-27 01:22 — go1.27.1, windows/amd64 (HEAD d06a846, plus the working tree)

| benchmark | ops | Veles | Go | Veles / Go |
|---|---:|---:|---:|---:|
| adapters | 24000000 | 48.63ms | 30.85ms | 1.6× |
| ast | 861841 | 40.11ms | 32.69ms | 1.2× |
| awaitall | 50000 | 40.37ms | 64.34ms | 0.6× |
| awaitscope | 50000 | 41.34ms | 64.52ms | 0.6× |
| bigargs | 40000000 | 35.69ms | 104.07ms | 0.3× |
| channels | 200000 | 10.11ms | 17.07ms | 0.6× |
| emit | 1000000 | 74.08ms | 69.24ms | 1.1× |
| emptycoll | 2000000 | 22.32ms | 19.49ms | 1.1× |
| gcpause | 20000000 | 505.88ms | 226.01ms | 2.2× |
| gunzip | 16777232 | 84.97ms | 62.57ms | 1.4× |
| gzip | 2097154 | 138.92ms | 20.17ms | 6.9× |
| httphello | 16000 | 227.72ms | 441.99ms | 0.5× |
| idle | 100000 | 106.95ms | 413.36ms | 0.3× |
| intern | 1000000 | 32.05ms | 16.83ms | 1.9× |
| json | 40000 | 58.22ms | 42.78ms | 1.4× |
| lexer | 2100000 | 55.69ms | 69.89ms | 0.8× |
| maps | 2000000 | 72.33ms | 58.24ms | 1.2× |
| parallel | 64 | 7.25ms | 26.17ms | 0.3× |
| parse | 163 | 58.34ms | 66.07ms | 0.9× |
| pipes | 1600000 | 24.43ms | 84.77ms | 0.3× |
| sha256 | 16 | 70.34ms | 8.18ms | 8.6× |
| sort | 900000 | 35.34ms | 53.31ms | 0.7× |
| spawn | 100000 | 38.36ms | 25.11ms | 1.5× |
| strings | 1000000 | 60.6ms | 48.19ms | 1.3× |
| suscall | 10000000 | 129.52ms | 6.92ms | 18.7× |
| trees | 14592688 | 165.9ms | 211.64ms | 0.8× |

| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |
|---|---:|---:|---:|
| idle | 249 | 8783 | 0.03× |

| benchmark | Veles longest pause | Go longest pause |
|---|---:|---:|
| gcpause | 1.81ms | 1.7ms |

## 2026-10-27 — D197 `cede` (plan F20) and D145's loop yield (plan F21), windows/amd64 and linux/amd64 (WSL), 32 logical CPUs (HEAD bc96824, plus the working tree)

**`cede` against `yieldNow()`.** One task per worker ceding 2M times
(`yieldNow()` at HEAD, `cede` after), ns per yield per task, three runs:

| workers | before (`yieldNow()`) | after (`cede`) |
| ------: | --------------------: | -------------: |
|       1 |            92, 92, 92 |     93, 89, 88 |
|       8 |      5379, 5209, 5114 | 5227, 5489, 5187 |

The same path in the runtime (a zero sleep); the call of the prelude
function is gone. For scale, Go's `runtime.Gosched()` in the same shape:
85 ns at `GOMAXPROCS=1`, 2195 ns at 8 — both go through a shared queue
under a lock; ours is 2.4× Go's under that contention (checklist §11).

**No regression** in `idle`, `spawn`, `channels`, `suscall`, `httphello`
at 1 and 8 workers (two runs each through the runner, before and after;
`channels` and `spawn` again six times each): the distributions overlap,
e.g. `spawn` at 8 workers 33.4–39.8 ms before, 37.6–43.6 ms after with
the same median band; `channels` at 8 5.3–7.2 ms before, 5.0–6.9 after.

**F21: how late a ticking task wakes beside busy loops** (`veles run
bench/preempt -- all`; one busy task per worker, a ticker sleeping 1 ms;
"lateness" is how much later than 1 ms each sleep returned, a tick in
flight at the end included).

Windows, after the timer-resolution fix below:

| workers | case       |  elapsed | ticks | worst lateness | mean lateness |
| ------: | ---------- | -------: | ----: | -------------: | ------------: |
|       1 | idle       |   301 ms |   197 |         0.8 ms |        0.5 ms |
|       1 | suspending |   367 ms |    23 |        16.5 ms |       14.9 ms |
|       1 | plain      |   370 ms |     1 |   **368.8 ms** |      368.8 ms |
|       1 | cede       |   134 ms |   133 |         0.6 ms |        0.0 ms |
|       1 | cede4096   |   396 ms |   391 |         0.7 ms |        0.0 ms |
|       1 | thread     |   382 ms |    24 |        16.1 ms |       14.9 ms |
|       8 | idle       |   301 ms |   193 |         0.9 ms |        0.6 ms |
|       8 | suspending |   449 ms |    46 |        16.5 ms |        8.8 ms |
|       8 | plain      |   432 ms |    36 |   **378.0 ms** |       11.0 ms |
|       8 | cede       |  5720 ms |  5703 |         0.4 ms |        0.0 ms |
|       8 | cede4096   |   551 ms |   548 |         0.3 ms |        0.0 ms |
|       8 | thread     |   497 ms |    28 |        29.2 ms |       16.7 ms |

Linux (WSL):

| workers | case       |  elapsed | ticks | worst lateness | mean lateness |
| ------: | ---------- | -------: | ----: | -------------: | ------------: |
|       1 | idle       |   301 ms |   269 |         0.2 ms |        0.1 ms |
|       1 | suspending |   346 ms |    23 |        14.5 ms |       14.0 ms |
|       1 | plain      |   316 ms |     1 |   **314.5 ms** |      314.5 ms |
|       1 | cede       |   105 ms |   105 |         0.1 ms |        0.0 ms |
|       1 | cede4096   |   347 ms |   344 |         0.1 ms |        0.0 ms |
|       1 | thread     |   345 ms |    23 |        14.5 ms |       14.0 ms |
|       8 | idle       |   300 ms |   262 |         0.2 ms |        0.1 ms |
|       8 | suspending |   414 ms |    36 |        14.5 ms |       10.5 ms |
|       8 | plain      |   385 ms |    15 |   **368.9 ms** |       24.7 ms |
|       8 | cede       |  4395 ms |  4382 |         1.8 ms |        0.0 ms |
|       8 | cede4096   |   437 ms |   435 |         0.1 ms |        0.0 ms |
|       8 | thread     |   321 ms |    22 |        14.3 ms |       13.5 ms |

`cede` runs a hundredth of the others' work (each cede is a trip through
the queue): ~110 ns a cede on one worker, ~5 µs at eight (contention, above).

Against Go (the runner's `preempt`, Windows; Go preempts a goroutine
asynchronously after 10 ms), worst lateness over three runs:

| workers | Veles            | Go                  |
| ------: | ---------------- | ------------------- |
|       1 | 16.1, 16.4, 16.8 | 28.2, 28.6, 22.1    |
|       8 | 16.6, 16.1, 16.5 | 20.6, 20.6, 20.5    |
|      32 | 15.6, 23.6, 33.1 | 275.9, 448.0, 474.6 |

Found on the way (Windows): seconds into a run Windows 11 stopped
honouring the process's 1 ms timer period (a process without a visible
window is throttled), the monitor's 5 ms looks became 15.6 ms ones, and a
one-thread executor's worst lateness rose to 46 ms (mean 44). Measured by
timing the monitor's waits; the runtime now opts the process out of that
throttling (`veles_timer_period`), and the monitor's waits stay at ~5.5 ms.

**F21 (4): the back-edge poll in tight loops** (ps per round, best of
five, 500M rounds; C through the same clang at `-O2`):

| loop                          | Veles plain | Veles suspending | C   | C + the same poll every round | C + the poll every 64 rounds |
| ----------------------------- | ----------: | ---------------: | --: | ----------------------------: | ---------------------------: |
| chain `x = x * a + i`         |         865 |             1723 | 100 |                           884 |                          260 |
| sum `t += i * i`              |         358 |              698 |   0 |                           366 |                          129 |

C with the poll matches Veles: the poll is the whole difference. Its
volatile load in every round stops clang from unrolling and reassociating
the chain and from turning the sum into a closed form or vectorizing it.
The suspending twins pay twice again (values stored to the coroutine
frame every round; checklist §11). Whether to poll every N rounds is Q65.

**Verdict on the slice (F21): keep it.** ~10 ms with a 5 ms monitor gives
a worst ~16 ms on Windows, ~14.5 ms on Linux, the same on a one-thread
executor (D150), better than Go in the same program, and nothing in these
numbers asks for a per-executor slice. What the measurements did find is
not the slice: the Windows throttling (fixed), the poll's price in tight
loops (Q65), the coroutine-frame traffic in suspending loops and a yield's
cost under contention (checklist §11). A plain function's loop still cannot
yield (D145 by design): a ticker beside it waits for the whole loop.
