---
name: veles-bench
description: Measure Veles performance against Go references, add a benchmark, record results, and guard against regressions. Use before and after any runtime, GC, scheduler, codegen or hot-std change, or when asked to optimise something.
---

# Benchmarking Veles

`bench/` pairs each Veles workload with a Go program doing the same work;
the runner refuses to compare them unless both print the same checksum.
Results read as Veles time / Go time. The header of `bench/results.md`
explains the caveats of each workload — read it before interpreting.

## Run

```bash
go run ./bench                # all, printed
go run ./bench -run spawn     # names containing "spawn"
go run ./bench -record        # also append a dated table to bench/results.md
```

Veles is built `--release`. One machine, one run: differences under ~10%
are noise — rerun two or three times before claiming a change, and report
the spread when it matters.

## When

- **Before** optimising: measure, find the workload that shows the problem.
  If none does, add one first. Past wins came from measuring first
  (a quadratic allocator at 376 s, a global lock that made 8 threads 20×
  slower than 1).
- **After** any change to `runtime/c/`, `codegen/llvm/`, or a hot std path:
  run the affected benchmarks; `-record` when the change lands so the
  history in `results.md` stays continuous.
- Concurrency changes: also run with `VELES_THREADS=1` and `8` — scaling,
  not just the default.

## Add a benchmark

1. `bench/<name>/main.vs`: time only the workload with
   `time.Stopwatch.start()`, fold results into a checksum so nothing is
   optimised away, print exactly
   `BENCH <name> <ops> <nanoseconds> <checksum>` (see `bench/maps/main.vs`).
2. The Go reference in `references` in `bench/main.go`: the same
   algorithm and data, not Go's idiomatic best (the question is the
   compiler and runtime, not the library) — unless the point is the
   library, and then say so in the comment.
3. One line in the `results.md` caveats if the comparison is not
   apples to apples (e.g. Go's SHA uses CPU instructions).
4. Run it; the checksums must match.

## Report

Before/after table for the affected rows, thread counts used, run-to-run
spread, and the cause of the change in one sentence. A slowdown elsewhere is
reported, not hidden. Do not tune to a benchmark in a way that hurts a real
program's shape; the checklist §3 lists the performance work that matters.
