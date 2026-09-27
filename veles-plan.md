# Veles — Execution Plan (2026-09-25)

The order of work from here, with a spec and acceptance criteria for each
task. The inputs are `veles-review-fixes.md` (engineering findings),
`veles-checklist.md` (production readiness), `notes_to_change.txt` (open
design notes) and `veles-selfhost-frontend-plan.md`.

How this file relates to the others:

- `veles-spec.md` records **language decisions** (D-numbers). A task here
  that changes what a program means waits for a decision (Phase 3) before
  it is built.
- `veles-checklist.md` records **production readiness**; a task here that
  closes one of its items ticks it there too.
- This file records **order and acceptance**. A task is done when its
  acceptance list holds, `go build ./... && go vet ./... && go test ./...`
  is green, and the docs say what changed.

The rule for ordering: first make the harness trustworthy (a green test
run must mean the compiler works), then pay debt that needs no decision,
then batch the decisions and ask, then the large items.

---

## Phase 0 — A green run means a working compiler

From `veles-review-fixes.md`. Nothing here changes the language.

### 0.1 End-to-end suites build a fresh compiler (review §1)

**Spec.** `examples/examples_test.go` and `docs/docs_test.go` build the
compiler into `t.TempDir()` on every run, never reuse a binary from the
tree, and never skip the build. A shared helper would need a new package
imported only by tests; two copies of eight lines are simpler, so each test
keeps its own.

A fresh build is half the fix. `go test` caches a package's result keyed on
the test binary plus the files and env vars the test reads *through the
`os` package*; `go build` runs in a child process, so its inputs are
invisible to the cache and an edited compiler would still get `(cached)`.
So each suite also **opens every compiler source file** (`.go` files of the
module outside `examples/`, `docs/`, `testdata/`, and the embedded `std/`
and `runtime/` trees, which the binary also carries) before building. The
cache then records them, and any edit invalidates the result. Cost: a walk
over a few hundred files, milliseconds. The walk lives in one small
test-only package, `internal/buildtest`, shared by both suites along with
the build itself.

**Acceptance.**
- Append a comment to `codegen/llvm/types.go`: `go test ./examples ./docs`
  runs both suites instead of printing `(cached)`.
- Touch a file in `std/` or `runtime/c/`: same.
- A build failure is `t.Fatalf` with the combined output.
- The tests no longer write `veles.exe` into the tree.

### 0.2 An internal compiler error is a report, not a Go stack trace (review §3)

**Spec.** `main.go` (the one entry to every subcommand) recovers a panic
and prints

```
internal compiler error: <message>
  while <phase> <file>:<line>:<col>   (when a position is known)
This is a bug in the Veles compiler. Please report it with the file above.
Set VELES_DEBUG=1 to print the Go stack trace.
```

and exits with status 3 (1 is a user error, 2 is usage). The position is
kept in a tiny `source`-level "current span" that codegen and sema update
as they lower a function (one assignment per function, not per node — the
function is the useful unit and it costs nothing). `VELES_DEBUG=1`
re-panics after printing so the full trace appears.

**Acceptance.** A test in `driver` calls the recovery wrapper around a
function that panics with a known position and checks the text and exit
code; with `VELES_DEBUG=1` the wrapper re-panics.

### 0.3 `veles fmt` writes atomically and reports walk errors (review §4, §5)

**Spec.** Writing a formatted file goes: temp file in the same directory
(`.<name>.veles-fmt-*`), write, `Sync`, `Close`, chmod to the original
file's mode, `os.Rename` over the original; on any failure the temp file is
removed and the original is untouched. A directory walk reports every error
it meets (root or entry) on stderr and makes the exit code 1; nothing is
skipped silently.

**Acceptance.** `driver` tests: formatting preserves the mode bits
(POSIX-only assertion), leaves no temp file behind, and `Format` on a
missing path or unreadable entry exits 1 with the error on stderr.

### 0.4 Hygiene (review §6)

Delete `veles.exe~`. `notes_to_change.txt` stays where it is (it is the
user's notebook; moving it is their call) — see Phase 3.

---

## Phase 1 — `codegen/llvm` gets direct tests (review §2)

### 1.1 Golden IR

**Spec.** `codegen/llvm/golden_test.go` compiles each `testdata/golden/*.vs`
through the same front end the driver uses (parse → sema → lower) and
compares the emitted module text with `testdata/golden/*.ll`. `-update`
rewrites them. The fixtures are small and each targets one area:
arithmetic and casts (every int/float pair, saturating `f64 as i64`),
builtins (`len`, bitwise, overflow ops), maps (get/set/ref/remove), sealed
dispatch and `when`, closures, a suspending function (coroutine
intrinsics), `with` cleanup, string interpolation.

A golden file is a *diff generator*, not a correctness oracle: the value is
that a codegen change shows its effect as readable text in review, and that
an accidental change fails loudly. Correctness stays with the examples.

**Acceptance.** `go test ./codegen/...` runs the goldens; `go test -cover
./codegen/llvm` reports well above 0%; every fixture also *builds and runs*
through clang in the test when clang is present (skipped otherwise), so a
fixture's IR is known to be valid LLVM, not merely unchanged.

### 1.2 Coverage for the rest

`lexer` and `driver` raised opportunistically as tasks touch them.

---

## Phase 2 — Debt that needs no decision

Each item is a gap a user hits, small, and already agreed in the notes or
checklist. Each lands with a test and a docs line.

| # | Task | Source | Spec in one line |
|---|---|---|---|
| 2.1 | Lint: `throws` on a function that cannot throw | notes "NEW TODO" | a declared `throws X` whose body's inferred error set is `Never` warns, fix removes the clause; not for trait methods, impls of a trait method, `extern`, or functions whose value is passed where a throwing type is expected |
| 2.2 | Lint: `loop (true) { }` → `loop { }` | T4 | warning + fix |
| 2.3 | Lint: counting loop → `.count()` | T4 | `var n = 0; loop (x in c) { n += 1 }` with `x` unused and `n` only incremented → hint with fix `val n = c.iter().count()` when `n` is not reassigned afterwards |
| 2.4 | Display recursion diagnostic | P7 | a `toString` in `implement Display for T` that interpolates `self` (directly) is an error: it recurses forever |
| 2.5 | Missing trait member reported at the implement | #12 | verify; if the error still lands at the use site, move it to the `implement` |
| 2.6 | `Channel.trySend(x): bool`, `tryRecv(): T?` | #14 | non-suspending; runtime functions in `veles_task.c`; `false`/`null` when full/empty or closed |
| 2.7 | `saturatingMul` on integers | #14 | next to the existing saturating ops |
| 2.8 | `os.hostname()`, `os.pid()`, `os.tempDir()` | checklist §4, §5.10 | wide APIs on Windows (R3) |
| 2.9 | `path.within(root, p): bool` | checklist §2 | lexical, after cleaning; `http.files` uses it |
| 2.10 | `fs.walk(root, f)` | checklist §5.10 | depth-first, sorted per directory, does not follow symlinked dirs |
| 2.11 | LSP: statics in `Type.` completion | P3 note | verify (R29 says done) and close the note |

Order: 2.1–2.5 are compiler work in `sema`, 2.6–2.10 are std/runtime.

---

## Phase 3 — Decisions to ask (one message, decision protocol)

Prepared with options, examples, edge cases, pros/cons and a
recommendation; nothing below is built until the user answers.

1. **#21 Trojan source**: refuse a fixed list of invisible/bidi code points
   outside string literals (recommended), vs UAX #31, vs leave as is.
2. **Duration on the wire** (checklist §9.12): ISO 8601 vs seconds vs the
   `Display` text.
3. **Guard binding I1**: `val x = expr else { ... }` vs `?:` on Result.
4. **`?.` through the rest of a chain** (R8 "to decide").
5. **Read-only exposure of a mutable-collection field** (R32).
6. **Arithmetic operator traits** (checklist §9.11).
7. **`is Trait`** (checklist §9.14).
8. **`notes_to_change.txt` layout**: fold the DONE log into the spec/
   checklist history and keep only open items.
9. **Mutating a value-struct parameter** (R20 follow-up 1, hit for real by
   `examples/fuzz`): a warning when a function changes a `var` field of a
   by-value struct parameter (directly or through a method that writes
   `self`), since the caller never sees it — vs making such a parameter
   immutable (Swift's rule) — vs leaving it.

---

## Phase 4 — The large items, in order

1. Multi-threaded executor (D35) — the throughput multiplier; `Mutex`,
   `Atomic`, the v7 counter lock. Needs decision 9 in the checklist.
2. FFI (checklist §1.2) — unblocks TLS, PostgreSQL, zlib, argon2.
3. `os.onSignal` + graceful shutdown in `http.serve`.
4. Benchmarks (`bench/`, checklist §3.3) before any runtime tuning.
5. Self-hosted front end (`veles-selfhost-frontend-plan.md`, P0 → P6).

---

## Progress log

Entries are appended as tasks land.

### 2026-09-25

**Phase 0 — done.** `internal/buildtest` (fresh compiler per run, cache
keyed on every compiler source — verified by touching `codegen/` and
`std/`), `driver.Guard` + `source.SetWhere` (ICE report, exit 3,
`VELES_DEBUG=1` for the trace), atomic `veles fmt` writes with walk errors
reported. Details in `veles-review-fixes.md` → Resolution.

**Phase 1 — done.** `codegen/llvm/golden_test.go`, eight fixtures, each
compared as IR *and* built and run; `codegen/llvm` coverage 0% → 56.8%.

**Phase 2 — done (2.1–2.11).**

- 2.1 `throws` on a body that cannot throw → warning + fix; the callers'
  `try` then errors with its own "Remove 'try'" fix, so `--fix` settles in
  two passes (the cascade is real: a caller that only forwarded becomes
  needless in turn). Public functions, trait methods/impls, functions used
  as values and type-parameter errors are exempt.
- 2.2 `loop (true)` → `loop { }` (warning + fix).
- 2.3 hand-counting loop over List/Set/Map/Range → `val n = xs.len()`.
- 2.4 `"$self"` inside its own `Display.toString` is an error (infinite
  recursion); another value of the type is not flagged.
- 2.5 #12 already held (reported at the `implement`); the message now
  says what to add (`add: fun name(short: bool): string`), and "does not
  implement" — for a bound and for a trait-object value, which used to be a
  bare "type mismatch" — says where (`add 'implement Shape { ... }' inside
  'struct Tri'`).
- 2.6 `Channel.trySend(x): bool`, `tryRecv(): T?`. Writing the test found
  **two executor bugs**, both fixed: a task sleeping when its scope's child
  finished was lost (reported as a deadlock), and `sleep(Duration.zero)`
  did not yield although the compiler and docs said it did.
- 2.7 `saturatingMul` on every integer type (with.overflow + select).
- 2.8 `os.pid()`, `os.hostname()`, `os.tempDir()`.
- 2.9 `path.clean`, `path.within`. Found and fixed a **directory traversal
  in `http.files` on Windows** (`GET /static/..\x` served files outside
  the root).
- 2.10 `fs.walk(root)`; `examples/dedup` uses it (its own walker followed
  directory links and could loop).
- 2.11 statics in `Type.` completion: already done (R29).

Also: the chapter 20 docs block's unused variable, and gofmt over the
tree (pre-existing misalignment in 16 files).

**Phase 2b — found while doing Phase 2 (done).**

- `format/TestCorpus` fails on any `examples/` or `std/` file `veles fmt`
  would change (checklist §6).
- `if (c) null else x` and a `when` with `=> null` arms infer `T?` from the
  other branches (Kotlin's rule); all-`null` is still an error. Hit while
  writing the fuzzer.
- `examples/fuzz`: a property fuzzer for the std decoders. Found and fixed:
  `toF64` panicking on a huge exponent, `json.parse` accepting `1e400`, and
  `toF64` not being correctly rounded (JSON floats drifted on each round
  trip; now `strtod`, the printer's own reference).
- Hit, **not** fixed (needs a decision, Phase 3 item 9): a value-struct
  parameter is a copy, so `fun step(f: Fuzzer) { f.rng.next() }` advances a
  copy and the caller's generator never moves — no warning. The fuzzer fed
  the same input 2000 times before this was noticed.

**Phase 3 — first batch answered and built (2026-09-25).**

- #21 → **UAX #31 identifiers** (`lexer/ident.go`), invisible characters
  named in errors, bidi controls an error in comments and a warning in
  strings, a leading BOM skipped. Spec D18 addendum; the self-host plan
  now needs a generated ID_Start/ID_Continue table (§4.2 note).
- Item 9 → **warning** on a changed by-value struct parameter
  (`sema/lint_paramcopy.go`), spec D22 addendum, docs 05 + errors page.
- Duration on the wire → **`"90.5s"` by default, `DurationStyle` to
  convert** (Seconds, Iso8601, Text, Nanos, Millis); spec D60 addendum,
  docs 18, `examples/codable`.
- I1 guard binding → **not decided**; `veles-guard-design.md` lays out
  let-else, `?:` on Result, `??` (and why every language that has it means
  Veles's `?:`), and a `Fallible` trait for user types. Recommendation:
  let-else + `?:` on Result; the trait later.

Still to ask: `?.` through a chain (R8), read-only collection fields (R32),
arithmetic operator traits (§9.11), `is Trait` (§9.14), the notes layout,
the executor threading model (§9.9), FFI (§9.8).

**Phase 3, I1 — built (2026-09-25).** Let-else + `??` on Result, spec D61:
`val x = r else { e => ... }` / `else return` (Result, nullable, variant
pattern; the else must leave), `r ?? fallback` / `r ?? { e => ... }`, and
`?:`↔`??` misuse as errors with a fix that swaps them. Parser, formatter,
checker, grammar, docs 06/07, cheat sheet, errors page;
`examples/json` uses it. Fuzzed (8.4M parser/formatter executions, clean).

**Phase 4.4 — benchmarks, done (2026-09-25).** `bench/` + `go run ./bench`
(Go reference per workload, checksums compared, ×Go column,
`-record` → `bench/results.md`). Writing it found the allocator
**quadratic between collections** — `json` 376 s → 0.31 s, `strings` from
not finishing to 0.3 s — fixed in `veles_gc.c`, guarded in `examples/gc`.
Baseline: maps 0.5×, trees 0.3×, json 2.8×, sort 2.7×, strings 3.0×,
sha256 22× (no SHA instructions), channels not comparable yet.

### 2026-09-26

D62 (A–D), D63 and D64 landed (see the checklist's decision log).

**`self` → `this`, spec D65 (v0.40).** Lexer maps `this` to the receiver
token; `self` still parses as the receiver with an error whose fix writes
`this` (interpolations included). 1,520 sites migrated by a lexer-driven
rewrite that touches tokens only (std, examples, bench, docs blocks, test
fixtures); prose, messages, hover, completion and the VS Code grammar by
hand. Found on the way: **`veles check --fix` never applied a fix attached
to a parse error** (`::`, `mut fun`, now `self`) because loading stopped
first — fixed in `driver.Run`, pinned by `TestCheckFixParseErrors`.

**Derived code vs the module's names.** `ast.PreludeName`: derived
`Codable` reaches `styleKey`/`childPath`/`joinPath`/`panic` in the prelude
even when the module declares functions of those names (was §11).

**HTTP correctness (checklist §5.2).** A 405 carries `Allow`; `HEAD` runs
the `GET` route and the writer drops the body but keeps its
`content-length`; `OPTIONS` answers 204 + `Allow`; 1xx/204/304 are sent
without body or length. `http.call(handler, method, target, body:,
headers:)` is the in-process test client. The stdlib reference's http
lines still used milliseconds from before D60 — fixed.

**Phase 3 second batch — asked and answered.** Threads: work-stealing M:N
(D66). FFI: extern blocks anywhere + `[native]` in the manifest, bindgen
maybe later (D67; the user's "every extern is unsafe" already holds).
Shutdown: awaited signal + `serve(stop:, grace:)` (D68). The std panic's
caller line: left as is.

**D68 built.** `os.shutdownSignal()`, `os.Signal`, `os.raiseSignal`
(`veles_signal_*` in veles_os.c: a one-shot handler per call, SIGINT/
SIGTERM on POSIX, the console control handler on Windows);
`http.serve(stop:, grace:)`; `examples/shutdown`, `examples/httpd` stops
gracefully, docs 17 "Stopping gracefully". Found on the way:
- **cancellation did not reach through a join**: a task cancelled while
  parked at its `scope`/`gather` join unwound without cancelling its
  children, which were orphaned with their `with` blocks never closed
  (D34/D43 violated). The abandon cleanup now stays active through the
  join (`codegen/llvm/coro.go`); `examples/cancel` pins it.
- `race` arms did not type `=> null` from the others (`if`/`when` did).
- **`veles fmt` changed a program's meaning**: `(fun(): T)?` was printed
  `fun(): T?`; the formatter, `ast.Dump` and type display now keep the
  parentheses (the round-trip test compared dumps, which had the same
  ambiguity, so it could not see it).
- **the codegen golden `.ll` files were never committed** (`*.ll` in
  `.gitignore`), so the golden test failed on a fresh clone; now
  un-ignored.
- Not verified here: a real console Ctrl+C/`kill` reaching the handler —
  the test environment has no console (`AllocConsole` is refused).

**D67 native linking built.** `[native]` in `veles.toml` (`libs`,
`static-libs`, `lib-paths`, `pkg-config`, file entries), collected from the
entry package and its dependencies (`driver/native.go`). `static-libs`
names the archive to the linker, which is what makes a link static on
every platform. `TestNativeLinking` compiles its own C with the same clang.

**Phase 3 third batch — asked and answered, all built.**
- D70 `?.` short-circuits the rest of the chain (Swift): `safeBelow`/
  `safeChain` in sema; parentheses end a chain (`Grouped` on the AST); a
  nullable place is used in place, so `maybe?.address.visit()` mutates the
  stored value.
- D71 arithmetic operators on user types through `Addable`/`Subtractable`/
  `Multipliable`/`Divisible`/`Negatable` (associated `Rhs`/`Out`, inferred
  from the method — a general rule for associated types now); `+=` follows;
  `Duration`/`Timestamp` adopted (`negated()` → `negate()`), docs and
  examples use the operators.
- D69 FFI marshaling and callbacks: `std/ffi` (`CString`, `readString`/
  `readBytes`, `alloc`/`free`, `handle`), prelude `CLayout` + `withRaw`,
  `p as *raw T` in unsafe, `extern "C" fun` + `&name: extern fun(...)`, a
  panic in a callback ends the process instead of unwinding through C;
  `examples/ffi` (qsort/bsearch with Veles comparators, handles,
  withRaw/memset, strtoll). Still open in FFI: varargs, `extern struct`
  layout controls, `.d.vs` declaration files, blocking calls on a helper
  thread (needs D66).

**D66 stage 1 — built 2026-09-27.** `runtime/c/veles_sync.c` (threads,
locks, condvars, the Mutex lock word); GC thread registry, lock-free safe
regions (Dekker-style flags against `veles_stop_requested`), park and
safepoints (allocation, suspension, loop back-edges), stop-the-world,
multi-stack scan, per-thread span ownership (allocation takes no lock);
executor on N workers (`VELES_THREADS`, default one per core) on one run
queue; `async` spawns onto the queue. 64-task allocation benchmark: 72 ms →
15 ms at 8 threads. Foreign calls and blocking runtime calls (stdin,
`os.run`, file reads) run in safe regions (TestForeignCallDoesNotStallCollection
fails without it); callbacks leave the region; `println` writes whole lines.
`Mutex`/`Atomic` are real locks (re-lock panics, released on panic,
`Atomic.update`); `with` cleanup records live in the frame, not the heap
(uncontended lock op 56 → 40 ns). Module-level `var` is an error unless
`Mutex`/`Atomic`; uuid v7 clock, random's generator, examples moved.
Pre-existing bugs threads exposed, all fixed with regression tests:
(1) a race waiting on two channels was linked into both waiter lists
through its one `next` field — a send on one cut the other's list (the
`examples/shutdown` failure; now one waiter node per arm, examples/tasks
"two races"); (2) a scope's rethrow matched the failed child's *runtime*
launch index against static sites, so a task launched in a loop that was
not the first lost its error (`mapConcurrent` with a throwing `f` panicked
"a slot was never filled"; examples/tasks "loop"); (3) cancel/abandon
left stale channel waiter entries. Five docs/examples relied on the
single-thread schedule's order and were rewritten to be deterministic.
Stress: every example × threads 1/2/4/8 × 8 runs, docs × 2/16 × 2 — clean.
Performance pass after: `bench/channels` had gone 2.5 ms → 60 ms with threads.
Three causes, all fixed: (a) MinGW clang supports only *emulated* TLS
(native TLS segfaults even in a hello-world with its GNU ld), so every
`current`/`me` access was a function call — now one per-thread block
(`runtime/c/veles_tls.h`) read from a TEB slot with one `%gs` load
(`__thread` elsewhere); (b) every wake handed the task to another
thread — now Go's *runnext*: a task woken by the running task runs next
on the same worker (fair after 32 turns; flushed to the shared queue
before any blocking call, `veles_blocking_enter`; the GC scans each
thread's block); (c) the runtime was compiled at -O0 in debug builds —
now always -O2. Result: channels 6.4 ms at 1/2/8 threads (Go 12.6 ms),
uncontended Mutex op 40 → 12 ns, 8-thread contended 1.55 s → 0.89 s,
parallel 64-task benchmark still 73 → 16 ms at 8 threads. The -O2
runtime exposed a latent ABI bug: a `bool` passed to C as a bare `i1`
has undefined upper bits and C reads the byte (`random.boolean()`
printed "true\0false\0in") — C-ABI small integers now carry
`zeroext`/`signext` like clang's (TestCABIExtension, golden strings).
**D66 stage 2 — built 2026-09-27.** Measured first: `bench/spawn` (100k
short tasks) and `bench/parallel` added with Go references; spawn got
*slower* with threads (19 → 443 ms at 1 → default threads) because every
task took the runtime lock ~5 times. Now: a 256-slot ring per worker +
shared queue + runnext + steal-half; per-task `sched` word (IDLE/QUEUED/
RUNNING/WOKEN) by CAS; spawn joins the scope under a per-scope spinlock;
`started`, `cancelled`, `await` of a done task and a successful `finish`
are lock-free (DONE published before the waiter is read; `await` and race
registration look again after registering). spawn: 17 ms at 1 thread,
39 ms at 8 (Go 32 ms). Two bugs found on the way: (1) the collector
recorded a safe thread's registers with setjmp in a C helper whose own
frame held a caller register it reused — dead after return, overwritten
while the thread waited — so a task held only in a register between
`async` and its spawn was swept (found with VELES_GC_POISON, a probe of
unmarked pending tasks, and gdb); now a few instructions of assembly at
the entry of `veles_enter_safe`/`veles_blocking_enter` record them exactly
(`veles_tls.h`), callbacks restore the outer record;
TestSpawnDuringCollections fails with the old recording. (2) The
deadlock detector fired while a worker holding a runnext task waited for
the lock, and after the root had just finished — it now checks both.
(aarch64 register capture: built later the same day, below.)
**D66 addendum — built 2026-09-27.** (1) `Atomic<T>` of an integer, float
or bool is lock-free: std-only builtins `atomicLockFree/Load/Store/Swap/CompareAndSwap` (sema/atomic.go, codegen/llvm/atomic.go) resolve per
instantiation — bodies are checked per instance, so the element type is
concrete; a non-word instance compiles the dead branch to `unreachable`.
`update` is a CAS loop (f may rerun). (2) Blocking calls hand off: the
worker (a P) gets `bstate` RUNNING/BLOCKED/HANDED_OFF + `bseq`; a monitor
thread hands a queue blocked across a whole look, while work waits and no
worker is idle, to a spare (`free_workers`, `spare_cv`); the returning
thread loses it by CAS, finishes its task's step and becomes a spare;
`place()` inside a C callback uses the shared queue (the queue may be
changing hands). Tests: TestAtomicWordsUnderThreads,
TestBlockingCallHandsOffItsThread (fails with the monitor disabled).
Docs: `docs/documentation/concurrency-explained.md` — the model for
beginners (threads/tasks/coroutines/green threads, JS comparison).
**Channels, 2026-09-27.** Measured first: 8 independent channel pairs
took 58 ms at 1 thread and 1209 ms at 8 — every channel operation took
the runtime lock. Now: a spinlock per channel; blocked waiters carry
their value slot and the other side copies across (`chan_done` marks a
completed op; `veles_task_cancelled` ignores a cancel until the retry
returns it); wakes run after the channel lock is released (`wakes`),
lock order runtime → channel; `race` winner by CAS, re-entry detaches
first (the old code re-pushed listed nodes: a waiter behind it was cut
off — deadlock, reproduced by TestRaceOverChannelsUnderThreads);
`enqueue` only turns BLOCKED into RUNNABLE (a late wake of a finished
task). `Channel<T>()` is a true rendezvous. pipes 13 ms (Go 62),
channels 4.6 ms (Go 9.8). **AArch64 register capture:** the stub
stores x19–x29, sp, d8–d15 on the stack and `veles_capture_store`
copies them (no TLS access from assembly); checked by cross-compiling
for aarch64-w64-mingw32 and disassembling — not run on hardware.
