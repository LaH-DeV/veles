# Veles — Execution Plan (reset 2026-09-28)

The order of work from here. Everything before this date — Phases 0–5 of
the previous plan (harness, codegen goldens, decision-free debt, three
batches of decisions, D65–D78, the threaded executor, FFI, the test
framework, the 2026-09-27 security pass) and its progress log — is done
and kept verbatim in `archive/progress-log-2026-09.md`.

How the files relate:

| File | Holds |
|---|---|
| `veles-spec.md` | language decisions (D/M numbers) — what a program means |
| `veles-checklist.md` | production readiness per area; §9 the open decisions (Q-labels), §10 the decision log, §11 known limits |
| this file | the **order** of work, with acceptance; a progress log at the bottom |
| `notes_to_change.txt` | the user's open design notes and standing directions |
| `veles-selfhost-frontend-plan.md` | the Veles rewrite of the lexer/parser (track E4) |
| `archive/` | finished plans, decided design notes, old logs — read-only |

A task is done when its acceptance holds, `go build ./... && go vet ./...
&& go test ./...` is green, docs say what changed, and the checklist is
ticked. A task that changes what a program means or a public std API waits
for its §9 decision (`veles-decide`).

## Principles (user, 2026-09-28)

1. **Every feature is complete**: every layer it touches (syntax, checker,
   codegen, runtime, std, formatter, LSP, docs, tests) and every platform,
   or it is not done.
2. **Performant**: measured against the Go reference when it touches the
   runtime, codegen or hot std code.
3. **Two levels**: manageable at the lower level (raw pointers, capacity,
   explicit resources, `unsafe` where it must be), while the high-level
   path is the shortest, safest spelling with the best developer experience.
4. **Towards self-hosting, slowly but surely**: not writing the Veles
   compiler yet, but every language and std decision is checked against
   "could the compiler be written with this?" (track S).

Order rule: **A** (the compiler is right on every platform) before **B**
(what every user touches daily) before **C** (std breadth, each API asked
first) and **E** (large items). **D** is asked in batches whenever a batch
is ready, so the user's answers are never the bottleneck.

---

## A — Correct everywhere (next)

| # | Task | Acceptance |
|---|---|---|
| A1 | **Linux build and test** (notes R3), in WSL Ubuntu 24.04 (installed; not the default distro — `wsl -d Ubuntu`). Setup is the user's: `sudo apt install clang make`, and Go from go.dev (`go.mod` wants 1.23.2; Ubuntu 24.04's apt Go is older) | `go test ./...` green on Linux x86-64: the runtime compiles (incl. the never-compiled `posix_spawnp` branch of `os.run`), every example and doc block passes, the thread stress (`VELES_THREADS=1/2/4/8`) passes; differences fixed, not skipped. A `bench/results.md` run recorded from that machine |
| A2 | **Sanitizers over the examples** | the runtime built with `-fsanitize=address,undefined` and a tiny `VELES_GC_THRESHOLD`, run over every example (a test flag or a script under `internal/`); clean, or each finding fixed with a test |
| A3 | **Verify D21 ships**: `+` checked in debug, wrapping in release, `+%` always; `as` between widths truncates only where written | a codegen golden per profile; checklist §1.4/§2 items ticked or turned into bugs |
| A4 | **Conformance suite** (the old P6 item) | `sema/testdata/conform/*.vs` with `// error: text` / `// warning: text` expectations, grouped by D-number, one negative case per diagnostic the checker emits; a test runs them all and lists diagnostics without a case |
| A5 | **Socket close vs a read in flight** (§11) | a per-socket in-use count (Go's fdMutex idea) so a close on one thread waits for, or fails, I/O in progress on another; a threaded test that closes during reads |
| A6 | **Fuzz the HTTP request parser** (checklist §2) | an in-process fuzz target over the request parsing `http.call` shares with `serve`; corpus kept; no panic, limits hold |
| A7 | macOS (later; the user runs it on an M4) | as A1, on macOS arm64 — the aarch64 register capture is so far only cross-compiled |

## B — Daily developer experience (decision-free unless marked)

| # | Task | Acceptance |
|---|---|---|
| B1 | **Every diagnostic names its fix and links its docs** (checklist §6) | audit of every `errorf`/`warnf` site; each message says what to write; an anchor per error family in `reference/errors.md`, printed as `see: <link>` and sent as LSP `codeDescription` |
| B2 | `veles build --timings` (checklist §3.2) | wall time per phase and per module; docs 01 |
| B3 | `veles test`: parallel tests (`--jobs`); helper tracing through `async helper()` (§11) | tests run as tasks across threads, reports stay in order; a test pins the order |
| B4 | **Panic stack traces with symbol names** (checklist §4, D49) | a panic in a task prints the Veles call chain, `at file:line` per frame; the crash-report format documented |
| B5 | `os.run` with stdin and a separately captured stderr (checklist §5.10) — **public API, ask first** | both platforms; docs 15 |
| B6 | LSP leftovers | code action "implement missing trait members"; whatever the B1 audit shows missing |
| B7 | `veles new --template server` (checklist §6) | logging, `/healthz`, graceful shutdown wired; runs and tests first try |
| B8 | Move the Go-lowered eager adapters into the prelude (notes #14) | benchmark first (`veles-bench`); move only what is not slower at -O2 |

## C — Standard library breadth (each public API asked first)

In this order, because each unblocks the next real program:

1. **`std/log`** — levels, structured fields, request-scoped through
   task-locals, no cost when a level is off (checklist §5.7).
2. **`std/http` server completeness** — cookies, forms (urlencoded,
   multipart to disk), static-file caching (`ETag`, `Range`), max
   connections with backpressure, chunked encoding, streaming bodies, CORS
   (checklist §5.2).
3. **`std/config`** — typed env parsing, every missing key reported at
   once (§5.10).
4. **Observability** — `/healthz` `/readyz`, a metrics registry with
   Prometheus text, `traceparent`, runtime metrics (§5.9; the GC counters
   come from E5).
5. **`std/fs`** — streaming reads/writes, atomic rename, file locks (§5.10).
6. The candidate concurrency helpers, when a program needs one (notes P11).

## D — Decisions to ask (checklist §9)

First, raised by the user: **Q15** named imports and **Q16** `as` no
longer meaning conversion — prepared together (they share `use` and
`as`), and early, because each rewrites much of std, the examples and the
docs, and every week adds code to migrate. Q3 `public use` is a natural
third in the same message (it is also about imports).

Then, in this order: **Q3** `public use` (if not asked with Q15), **Q4**
read-only collection fields, **Q8** caller location for
std's misuse panics, **Q7** FFI varargs / `extern struct` layout / `.d.vs`,
**Q13** the small syntax consistencies. Waiting on the user: **Q1** race
send arms, **Q2** test setup/teardown. Later: Q5, Q6, Q9–Q12, Q14.

## S — Towards self-hosting (ongoing; no compiler rewrite yet)

The front-end rewrite itself is E4. This track keeps the language and std
on course for it, a little at a time, alongside A–C.

| # | Task | Acceptance |
|---|---|---|
| S1 | **Refresh `veles-selfhost-frontend-plan.md`** against D60–D78 | its §4 "not gaps" list re-checked (a module-level `var` is now an error unless `Mutex`/`Atomic` — the keyword table and any interning move to a `val` or into a struct; `this`; `init(params)`; D78 tests for the harness; the UAX #31 table generator); stale claims struck with the date |
| S2 | **Compiler-shaped benchmarks** in `bench/` | Veles vs Go on: tokenising a large file, building a sealed-tree AST and walking it with `when`, a string-interning map, emitting text through `StringBuilder`; recorded, and each gap over 3× Go gets a checklist item |
| S3 | **Capability audit for a whole compiler**, not only the front end | a section in the self-host plan: what sema/codegen in Veles would lean on (large pointer graphs under the GC, maps keyed by structs, deterministic iteration order, sorting, `os.run` of clang, file I/O, deep recursion) — each marked has / gap, gaps become checklist items |
| S4 | **Readiness in every decision** | `veles-decide` prepares a "what this means for writing the compiler in Veles" line when it applies (Q15/Q16 first) |
| S5 | Start E4's P0 (`source` + harness) | once A1 is green and S1 is done |

## E — Large items, in order

1. **TLS** through a system binding (SChannel / OpenSSL) → `std/tls` → the
   HTTP client (§5.3, §5.4); then `examples/apiclient`.
2. **PostgreSQL**: libpq binding, pool, transactions as `with`, rows
   mapped through the derive → `examples/pgnotes` (§5.8).
3. **I/O reactor**: `poll` → epoll/kqueue/IOCP, writev/readv; an HTTP
   hello load test added to `bench/` first, to measure it (§3.1, §3.3).
4. **Self-hosted front end**, P0 → P6 (`veles-selfhost-frontend-plan.md`).
5. **GC**: pause/heap/allocation metrics, then generational or incremental
   marking once a server-shaped heap is measured (§3.1).
6. **Release engineering**: an `-O2` + LTO profile, static binaries,
   Windows → Linux cross-compile, a Docker image, the "deploying Veles"
   page (§3.2, §4, §7).
7. **Packages**: M7 registry/MVS, lockfile, reproducible builds; then the
   manifest questions (Q14, notes #17).
8. **Compiler speed and code quality**: per-module IR caching,
   devirtualisation, escape analysis, panic-freedom analysis (§3.2).

---

## Progress log

Older entries: `archive/progress-log-2026-09.md`.

### 2026-09-28

**Records cleaned up.** Finished plans and decided design notes moved to
`archive/` (`veles-build-plan.md`, `veles-review-fixes.md`,
`veles-guard-design.md`, `veles-testing-design.md`); the previous plan
with its log, and `notes_to_change.txt` as it stood, kept verbatim there.
The notes file now holds only open items (the user's text unchanged;
status lines marked `→`). Checklist §9 rebuilt as the open decisions
Q1–Q14 (the old list appended to the archived log); stale items ticked
(orphan rules — none beyond D17; `Deque`/`PriorityQueue`/`Set` operations)
or marked `[?]` with their Q; the duplicated panic-freedom item merged;
§11 cleared of the limits fixed on 2026-09-27 and given the two found then.

**User directions added (2026-09-28).** Four principles now lead this
file (complete, performant, two levels, towards self-hosting) and are in
CLAUDE.md and the `veles-decide` preparation list. New track S
(self-hosting readiness, S1–S5). Checklist §9 gained Q15 named imports
(reopens R9) and Q16 `as` no longer meaning conversion, first in track D.
A1 runs in WSL Ubuntu 24.04 (present; Go and clang still to install); A7
is the user's M4, later.
