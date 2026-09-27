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

## Phase 5 — The user's cleanup notes (2026-09-27)

From the user's side notes of 2026-09-27; decisions D73–D75.

| # | Task | Acceptance |
|---|---|---|
| 5.1 | Same-named types read apart (note 1) | `throws hex.Invalid \| base64.Invalid`, never `Invalid \| Invalid`; mismatch messages likewise; sema test |
| 5.2 | Hover for built-ins (note 3) | primitive types show doc + methods + implements; `Ok`/`Err`/`Some`/`None`, `panic` hover; built-in method signatures substituted for the receiver; lsp test |
| 5.3 | Constructors instead of factories, no decision needed (note 2) | M5 v0.30 across modules; `StringBuilder()`, `Deque<T>()`, `http.Router()`, `Depth(limit:)`, `json.JsonEncoder()`; old calls error with a fix; tree migrated |
| 5.4 | D73 `init(params)` | parser/formatter/AST dump, sema (params in the constructor, scope in the block, name clash with a passable field refused), hover/signature help show them; `Mutex`/`Atomic`/`TaskLocal`/`PriorityQueue` converted, factories removed with fixes, tree migrated, docs 05 + stdlib + cheat sheet |
| 5.5 | D74 http named values | `Status`, `Method`, `Header` in `std/http`; every public status/method typed; examples/httpd, docs 17, stdlib reference |
| 5.6 | D75 slim prelude | `std/codec`, `std/recursion`, `CLayout` in `std/ffi`; derived code needs no import; tree migrated; stdlib reference regrouped |
| 5.7 | CLI: path optional, flags anywhere | `veles run` in a package directory; `veles check --fix <path>` |

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
**LSP references and rename, 2026-09-27.** `textDocument/references`,
`documentHighlight`, `prepareRename`, `rename` (`lsp/references.go`),
built on the checker's index grouped by declaration (`sema.Ref.Key`). The
index gained what rename needs: named-argument labels as uses of the
parameter or field (`scale(factor: 2)`, `Point(x: 1)`), D28 puns marked
(`Point(x)` is spelled out `Point(col: x)` / `Point(x: row)` so the other
name survives), a trait method's declaration and calls through a trait
object, and `Family` tying each implementation to the trait's method. A
rename is proved before it is sent: the package is checked again with the
edits and the rename is refused, naming the place, if any use resolves to
a different declaration (capture, shadowing) or an error appears. Fields
and sealed variants a derived `Codable` writes on the wire (no `@key`,
not `@skip`) are refused with the `@key` hint — the program would compile
and stop reading its own data (`Index.Wire`). Tests: TestReferences,
TestRename (10 cases incl. capture and pun), TestRenameKeepsWireNames,
TestRenameAcrossModules. Not done: enum members are not guarded (every
enum is Codable implicitly, so guarding would refuse every member rename).
**`veles test` runner, 2026-09-27.** `--filter text` (substring of the
test name; matching nothing exits 1 so a mistyped CI filter cannot pass
green), `--timeout d` per test (default 10m, `0` unbounded): the runner
arms `veles_test_watch` (runtime/c/veles_sync.c, a lazily started thread
on a condvar) before each test; at the deadline it prints `FAILED: timed
out after 300ms` and how many tests did not run, and ends the process —
a task busy in a loop cannot be stopped from outside. A summary line
closes every run: `1 passed, 2 failed: failing, panicking; 2 filtered
out`. Pinned by driver TestTestRunner (exact output, filter, empty
filter, timeout). Docs 11/14, cheat sheet, README. Also: three
`httpd.exe` from `examples` runs on 2026-09-26 were still running a day
later — a hung example held the suite until `go test` gave up, which
leaves the child alive. `examples_test.go` now bounds each run (2 min,
killed, the test fails with the output so far). Why those runs hung was
not found (they were left running; the current tree passes).
**LSP inlay hints, 2026-09-27.** `textDocument/inlayHint` (`lsp/inlay.go`):
`: T` after an untyped `val`/`var`/loop/lambda binding (none when the
initializer names the type — `Point(`, `Point.origin()`, `geo.Point(` —
and none when a generic body's instances disagree), the return type of an
expression-bodied function, `suspends` where inference added it (D2), and
the set of a bare `throws` (D45). The checker records the unwritten parts
per declaration after effect inference (`Index.Inferred`, only for
functions with one meaning — no own/owner/impl type parameters, not a
trait default). Tests: TestInlayHints (the rendered source with the hints
spliced in), TestInlayHintsGeneric.
**LSP signature help, 2026-09-27.** `textDocument/signatureHelp`
(`lsp/signature.go`, triggers `(` `,`): the innermost unclosed `(` from
the tokens before the cursor (so a call being typed, which breaks the
parse, still answers from the last good index), inside `${}` too; the
active parameter by position, by `name:` once written, the variadic one
for the rest; constructors list the fields the caller gives (not `init`
ones), defaults shown `= …`. TestSignatureHelp.
Also `workspace/symbol` (Ctrl+T): every declaration and member of the
analysed packages, letters-in-order matching (`nf` → `notFound`), the
standard library left out. TestWorkspaceSymbols.
**Strings performance, 2026-09-27.** Measured first: the `strings`
workload split into its phases (interpolation 95 ms, `StringBuilder.append`
75 ms, split 26 ms for 1M items). Four causes, all fixed: (1) integers
were formatted with `snprintf` (~100 ns in the Windows CRT) — now a
digit-pair loop, and inside an interpolation into a frame buffer that is
copied once (`veles_i64_format`); (2) an interpolation of N parts made
N−1 allocations, each copying the prefix — now one
`veles_string_concat_n`; (3) `MutableList.push` was always an out-of-line
runtime call with a memcpy — the fast path is inline now (every push in
every program); (4) `StringBuilder.append` pushed byte by byte through a
copied `bytes()` list — now one copy (std-only `listAppendText`). Found
on the way: **`List.join` was quadratic** (it appended to an accumulator
it copied whole each step; `replace` is split + join): 80 000 numbers
took 38 s, now 5 ms (`veles_string_join`, one allocation);
`examples/gc` pins it (the examples harness now bounds each run at
2 minutes, so a regression fails instead of hanging).
Measured (`go run ./bench`, --release, default threads, 2 runs each,
spread under 10%): strings 180 → 64–70 ms (4.9× → 1.8× Go), json 143 →
95–98 ms (3.1× → 2.0×); sort, maps, trees, channels unchanged. Recorded in
`bench/results.md`.
**`sorted()` on integers and strings, 2026-09-27.** The prelude merge
sort paid one indirect comparator call (then `compareTo`) per comparison,
17M for the `sort` workload. Two equal integers or byte-equal strings
cannot be told apart, so stability is unobservable there and `sorted()`
now sorts a copy in the runtime (`veles_list_sort_native`: the same
runs-of-32 + bottom-up merge, direct comparisons, a malloc'd scratch
buffer). Floats stay on the comparator path (NaN and ±0 make order among
"equals" visible). sort 113 → 41–46 ms (2.2× → 0.5–0.9× Go; the Go side
is noisy today). `examples/algorithms` cross-checks it against
`sortedWith` for i64/u8/string at 10 sizes around the run boundaries.
An insertion-run pre-pass in the generic `sortedWith` was tried and
measured no gain; not kept.
**Phase 5 (user notes of 2026-09-27) — 5.1–5.4 and 5.7, 2026-09-27.**
(1) Same-named types read apart: `types.Distinct` spells a named type
with its module when another type in the same message shares its name —
`throws base64.Invalid | hex.Invalid` was `Invalid | Invalid` (the user's
"the same error multiple times"); the union's order breaks name ties by
module. (2) Hover for the compiler's own names: primitive types show a
description, every method (catalogue + prelude `extend` blocks) and the
traits they implement; `Ok`/`Err`/`Some`/`None` and `panic` hover; a
built-in method's signature is written for its receiver
(`List<i64>.sorted(): List<i64>`, was `List<T>`). (3) M5 v0.30 (a
`private` field with a default is left to it, one without is given by
the call) held only inside the module; it holds across modules now, so
`StringBuilder()`, `Deque<T>()`, `http.Router()`, `Depth(limit: n)`,
`json.JsonEncoder()` are constructors; the factories are removed and an
old call is an error with a fix (`sema/removed.go`). (4) D73 built:
`init(params)`; `Mutex(value:)`, `Atomic(value:)`, `TaskLocal(fallback:)`,
`PriorityQueue<T>(compare:)`, `PriorityQueue<T>.natural()`; generic
inference and puns see init parameters; a derived `Decodable` is refused
when `init` needs an argument. **Found on the way: a generic struct's or
sealed type's bounds were never checked** — `Box<P>` for `struct Box<T:
Comparable>` compiled; `checkTypeArgBounds` now checks every instantiation
with a source position (deferred past collection, silent inside std,
whose generics only pass on their caller's choice). (5) CLI: the path is
optional for build/run/check/test (the current directory) and flags may
precede it (`veles check --fix x.vs` said "unknown flag x.vs").
Tests: sema TestSameNamedTypesAreQualified, TestRemovedFactories,
TestInitParameters, TestStructTypeArgBounds, TestPrivateFieldsAndTheConstructorAcrossModules;
lsp TestBuiltinNamesHover; parser TestInitParams; a format style case.
**5.5 — D74 built, 2026-09-27.** `std/http/values.vs`: `Status` (47
constants with RFC 9110 names — 413 is "Content Too Large" now —,
`reason()`, `isSuccess()`…, prints `404 Not Found`, a code outside
100..999 panics), `Method` (nine constants), `Header` (lower-case name
constants). Every public status/method in `std/http` is typed; routes keep
`Method?` (null = any), `Router.patch` added; `reasonOf` removed (error
naming `Status(code: n).reason()`). The status line and the log lines
still print the number. DX found on the way: a mismatch into a one-field
value type now says how to build one and lists its constants, and a
literal matching a constant (`status: 201`) gets that constant with a fix
(`http.Status.created`), spelled as the reader's module names it.
examples/httpd migrated with `veles check --fix`; its expected output
changed only in the reason phrases. Test: sema TestHttpNamedValues.
**5.6 — D75 built, 2026-09-27.** The declarations stay in the prelude's
sources (its own `Encodable` impls and the derived code need them in one
unit), and `sema/prelude_home.go` gives 27 names a home module: they go
into that module's scope instead of the universe. `std/codec`,
`std/recursion` are new modules whose files hold only documentation;
`CLayout` joins `std/ffi`. Derived code and the checker's lowering look
names up in the prelude module itself (`preludeSym`), so `implement
Codable` needs no import. An unknown name that has a home says where
(`unknown type 'Value'; did you mean 'codec.Value'? (add 'use codec' ...)`),
and the "parameter has type '<invalid>'" error that used to follow a
hand-written `encode(to: Encoder)` is gone. Completion: `codec.` offers
them, the global list does not. Migrated: std json/jwt/time, examples
codable/crypto/fuzz/recursion, docs 18/19, the stdlib reference (new
`codec` and `recursion` sections). Tests: sema TestSlimPrelude, lsp
TestCompletionOfPreludeHomes; TestDerivation's sources now `use codec`.
**DX follow-ups, 2026-09-27.** (1) `veles check --fix` runs to a
fixpoint (up to 8 passes) and reports what is left once: a fix often
uncovers the next — an error in a declaration stops the checker before
the bodies — so one run now settles what took several (nested index
forms, D75 names). Identical edits from different fixes are applied once.
(2) An unknown name the prelude writes for a module gets a fix that
qualifies it and adds the `use` (above the first declaration's doc
comment, or into the existing `use` block); only for those names — any
other "did you mean" is a guess, and `--fix` applies every fix.
(3) A binding whose value failed to check is still declared (as unknown):
`loop (x in <bad>)` and `val (a, b) = <bad>` no longer report every use
of the names. Tests: driver TestCheckFixQualifiesModuleNames (and the two
fix tests now pin one-run convergence), sema TestNoCascadeFromABadBinding.
**`veles new`, 2026-09-27.** `veles new <dir>` writes `veles.toml`,
`main.vs` (a function, `main`, a passing `@test`) and `.gitignore`; the
name must be an identifier (dependents write it in `use`); a non-empty
directory is refused. `veles build` in a package names the executable
after the package (it was `main` when run as `veles build` from inside).
Docs 01 + cheat sheet; driver TestNew.
**Performance: json and strings, 2026-09-27.** Measured by phase: a
float took 1.2 µs to print (a printf/strtod search) and the JSON decoder
paid a bounds-checked list access per byte. Fast paths in the runtime
(shortest-digits printing, Clinger parsing, 8-byte ASCII UTF-8 check,
one-pass `split`) and in `std/json` (pointer cursor, direct integers,
one-copy strings via the std-only `listDecodeUtf8Range`). json 2.0× →
0.9× Go, strings 1.7× → 1.4×; other rows unchanged. Both float paths
were checked against the old code / `strtod` on millions of values in a
standalone C harness; `examples/text` pins the edge cases. Output change:
an exact tie in the last printed digit now rounds half to even on every
platform (the Microsoft CRT rounded it away from zero).
Also: docs 12's first concurrency example relied on a 10 ms gap between
two sleeps and failed once on a loaded machine; the gap is 90 ms now.
**Testing design — investigated, open, 2026-09-27.** Proposed a built-in
`assert`; the user asked for a deeper look (test-only helpers, whether
`@test` should be a `test` form). `veles-testing-design.md` lays out
today's gaps, seven languages, five questions with options and examples,
and a recommendation. Checklist §9 item 15. Nothing built.
**Diagnostics that fix themselves, 2026-09-27.** (1) "type 'P' does not
implement trait 'Encodable'" (and Decodable, Comparable; also "cannot
order by 'P'") carries a fix inserting `implement Encodable` into the
user's struct — the compiler derives the rest — and the hint says so
instead of `{ ... }`. (2) A generic instance's failure names the element:
"'List<T>' implements 'Encodable' when 'T' does, and 'Tag' does not",
with the fix on `Tag`. (3) `try f() ?? x`: one error, "'??' handles the
error itself; drop the 'try'", with the fix, instead of two unrelated
ones. (4) `time.Duration` / `json.Value`: "'Duration' is global (the
prelude): write 'Duration'" (fix) / "it is 'codec.Value'"; the "unknown
type" that followed a failed qualified path is gone. (5) No "cannot infer
type parameter" after an argument already failed; no unused-name warning
from derived code (`struct E { implement Codable }` warned about `k`).
Pinned: driver TestCheckFixDerivesMissingImplements (a broken program →
one `check --fix` → runs), sema TestUnmetBoundThroughAnImpl,
TestMemberOfTheWrongModule, TestDerivedCodeHasNoWarnings.
**`veles doc`, 2026-09-27.** `veles doc [dir] [-o out]`: the package's
public API as Markdown — each module an outside reader can reach (root and
`exports`; `Package.LoadAll` loads modules nothing imports), each public
declaration in source order as the hover spells it from outside
(`sema.DocPackage`, reusing `shapeFrom`/`funDecl`/`globalDecl`), its `///`
comment and its documented members; private members and unexported
modules left out; a package with errors refused. Docs 11 + 01 + cheat
sheet; driver TestDoc. Checklist §6 ticked.
**Fuzzing, 2026-09-27.** FuzzFormat found `if (c) { !return }` formatted
as `!(return)`, which did not parse: a bare `return` now also ends at `)`,
`]` or `,`, where it stands as an expression. The input is kept in
`format/testdata/fuzz`. FuzzCheckMutated ran 3.5 min (30k inputs) clean.
**LSP navigation, 2026-09-27.** Go to type definition (`sema.TypeDecl`:
through `?`, `*`, tasks and containers to the first declared type; a
call goes to its result's type), go to implementation (`Index.Impls`,
filled after checking even when the package has errors: a trait → the
implements the author wrote, a sealed trait → its variants, a trait
method → each implementing method) and folding ranges (from tokens:
bracket pairs across lines, comment runs, `use` runs). lsp
TestTypeDefinition, TestImplementation, TestFoldingRanges.
**Quick fix: `@skip`, 2026-09-27.** "cannot derive 'Encodable' for 'Job':
field 'onDone' is a function…" carries "Mark 'onDone' @skip" when the
field has a default; without one no fix is offered, since the edit would
only trade the error for "a @skip field needs a default". sema
TestSkipFixForAnUncodableField (applies the fix and re-checks).
**`->` for `=>`, 2026-09-27.** Kotlin's `1 -> "one"` in a `when` or
`race` arm was four errors per arm ("expected '=>'", then "expected
newline between arms" for each arm after it). Now one error per arrow,
"'->' is not an operator; an arm is 'pattern => value' (D33)", with the
fix, parsed as `=>` so the rest checks; the two older `->` errors
(expression, statement end) carry the fix too. parser
TestThinArrowInArms.
**D76 `IoError.kind` + D77 `Weekday`, 2026-09-27** (user decisions, note
#5). `IoKind` (17 members, `Other = 0`) sits in the prelude beside
`IoError`, which gains `public kind: IoKind = IoKind.Other`; the runtime's
`veles_io_kind` (in `veles_net.c`, the one file that sees errno and the
Winsock codes; every errno case `#ifdef`-guarded) maps the number, and
`os.ioError` fills it; `net`'s not-UTF-8 line is `InvalidData`. driver
TestIoErrorKinds builds a program and checks NotFound / AddressInUse /
ConnectionRefused (Windows; Linux not run). `time.Weekday` (ISO, Monday =
1) is what `DateTime.weekday()` returns; `formatHttp` indexes by it.
`examples/files` shows the default-on-missing idiom, `examples/time`
prints day names; both regenerated. Docs 15, 20, stdlib reference.
**D78 tests, 2026-09-27** (user decision: the recommended combination of
`veles-testing-design.md`). Parser: `test "sentence" { }` (contextual;
interpolated or empty names refused) and `test fun`; AST dump, formatter,
grammar. Sema (`testing.go`, `testing_words.go`): a test is a synthesized
function with an inferred `throws`, named by its sentence; duplicate names
per module refused; `@test fun` errors with a fix to `test "words"`
(`parsesURLQuickly` → "parses URL quickly"). The vocabulary lowers to HIR
plus synthesized syntax over hidden locals: `expect`/`check` capture both
sides of a comparison, `require` unwraps `T?`/`Result` or stops,
`expectThrows<E>` (compile-time refusal when the body cannot throw `E`),
`expectPanics` (through `gather` and a private prelude `runTestBody`),
`fail`. Test code (test, `test fun`, `*.test.vs`, lambdas in them) is the
only place the words and test helpers are allowed. Runtime: per-test
failure buffer under a lock; the runner prints each test's recorded
failures and counts a test with any as failed. `*.test.vs` never reaches
`build`/`run`. LSP: hover for the words, tests in the outline. Migrated:
`examples/testing` (six tests, all pass), syntax tour, `veles new`
template, docs 01/11/14 (14 rewritten; its four stray table rows above the
title moved into the table), index, cheat sheet, README. Pinned: driver
TestTestVocabulary (exact report), TestTestRunner (new form), sema
TestTestDeclarations, parser TestTestDecl, format "tests (D78)", lsp
TestTestsInTheEditor. Known limits in checklist §11: test output not
captured; `expectPanics` body must be sendable; a helper's failure reports
the helper's line.
**D78 amended: `assert`, 2026-09-27** (user: "the everywhere available
'check' should be called 'assert' (and it should check condition and
require string explanation like panic)"). `assert(cond, "why")`: the
reason is required (one argument is an error saying so), evaluated only
on failure, and leads the panic — `panic: <why>`, then `assert(cond)` and
both sides of a comparison. `check(...)` with nothing declared under that
name is an error naming `assert`; a user's own `check` is untouched
(httpd and chapter 12 declare one). Example, docs 14, cheat sheet, spec,
checklist; sema TestTestDeclarations.
**D78 amended: suites, 2026-09-27** (user: "both" — blocks and files;
report grouped, indented). `suite "name" { }` (contextual, parser checks
it holds only tests, suites and `test fun`), AST dump, formatter, grammar,
LSP outline (nested). Sema: a suite's helpers live in a scope of its own
between the file's and its tests' (`FuncTemplate.SuiteScope`, unique
mangled names), and "unknown function" names the suite a helper belongs
to; qualified names `a / b / test`, unique per module; a `*.test.vs` file
is the suite of its stem. Runner: headings when a test enters a suite,
tests and recorded failures indented by depth (`veles_test_take` indents),
tests outside suites first. driver TestTestSuites (exact report, filter by
suite), sema TestTestSuites, parser TestSuiteDecl, format "suites",
lsp outline; docs 14 "Suites", cheat sheet, `examples/testing` suite
"ports". Per-test setup/teardown: rejected as proposed, open (§9 item 16).
**`// SAFETY:` lint + std `unsafe` audit, 2026-09-27** (notes #13, decided
2026-09-18 as a lint; checklist §2). `sema/lint_unsafe.go`: an `unsafe`
block outside `unsafe fun` (and not nested in another block) warns unless
a `// SAFETY:` comment sits in the comment run above its line or first
inside the block; the fix inserts the stub, and an empty reason still
warns. All 73 blocks in std got a reason written against what the runtime
function does (the ones that are declaration bodies carry it inside, so
doc comments stay attached); `examples/ffi`, the syntax tour, docs 13
(new paragraph), errors reference and cheat sheet likewise. sema
TestUnsafeNeedsSafetyComment, TestStdIsWarningFree (a program importing
every std module: no warning in std, since each would reach every user).
BUG FOUND BY THE AUDIT: `net` closed a socket number again on a second
`close()` — a closed `Conn` copy, or `close()` inside a `with`, closed
whichever socket the system had given that number next (reproduced on
Windows: `b` died when `a` was closed twice), and a read after close
could reach another peer. `net.Socket` (private) keeps the number in an
`Atomic<i64>` that `close` swaps to -1; driver TestSocketClosedTwice. Not
fixed: a close racing a read on another thread (Go's fd refcount) — the
read may still hit a reused number in that window.
**`await` on a call, 2026-09-27.** `try await net.connect(...)` said
"'await' applies to channels, timers and task handles, not
'Result<Conn, IoError>'" — true, and no help. `await` on a plain call now
says a suspending function is called like any other (D2) and carries a
fix that removes the word; checking continues with the call's type, so
nothing cascades. sema TestAwaitOnACall; errors reference.
**Test output captured, helper call sites, 2026-09-27** (§11 known
limits of D78). (1) While a test runs, `io.print`/`println`/`eprint`/
`eprintln` go to a buffer (`veles_test_capture`, veles_sync.c, one atomic
load when no test runs); `veles_test_take` ends the capture and appends it
under `output:` to the report, so a passing test's output is dropped and
a failing test's shows under its failure; a timed-out test prints what it
had. (2) A call from test code to a `test fun` (or a `*.test.vs`
function) is wrapped in `test.enter`/`test.leave` (sema traceHelperCall),
which bind the call site in the task-local list under a reserved key, so
tasks started inside inherit it; `veles_test_fail` appends `called from
file:line:col` per site, innermost first. driver TestTestOutputCaptured,
TestTestHelperCallSites; TestTestSuites now shows the call site. Docs 14.
**`os.run` without a shell, 2026-09-27** (security; found reading std for
the audit). `run(program, args)` built a command line for `popen` and
quoted an argument only when it held a space or `"`: `x;echo pwned`
ran a second command on POSIX, `$(...)` and backticks expanded even inside
the quotes, and on Windows `%VAR%`, `&` and `|` were cmd.exe's. The
runtime now takes program and arguments NUL-separated and spawns
directly: `posix_spawnp` + a pipe (`mergeStderr` = dup onto fd 2), or
`CreateProcessW` with the MSVC quoting rules (backslashes doubled only
before a quote) and an inheritable pipe. `.bat`/`.cmd` are refused on
Windows (cmd.exe re-parses; the "BatBadBut" family), a NUL byte in an
argument is `InvalidInput`. driver TestRunPassesArgumentsVerbatim runs
the program as its own child with ten hostile arguments. NOT VERIFIED:
the POSIX branch was not compiled (no Linux headers here) — build it on
Linux before relying on it. Docs 15, stdlib reference.
**NUL bytes at the C boundary, 2026-09-27.** A Veles string may hold a
NUL; the runtime hands paths and hosts to C, which stops at it. `fs`
refuses such a path (`checkPath`, `InvalidInput`; `exists`/`isFile`/
`isDir` answer false), `net.listen`/`connect` such a host (an
`endsWith(".trusted.example")` allow-list otherwise resolved the part
before the NUL), `os.env` answers null. driver TestPathsWithNulRefused.
