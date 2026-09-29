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
| A1 | **Done 2026-09-28.** **Linux build and test** (notes R3), in WSL Ubuntu 24.04 (`wsl -d Ubuntu`; Go 1.23.2 in `~/.local/go`, clang 18 from apt); rerun with `internal/wsl-test.sh` | `go test ./...` green on Linux x86-64: the runtime compiles (incl. the never-compiled `posix_spawnp` branch of `os.run`), every example and doc block passes, the thread stress (`VELES_THREADS=1/2/4/8`) passes; differences fixed, not skipped. A `bench/results.md` run recorded from that machine |
| A2 | **Done 2026-09-28.** **Sanitizers over the examples** (`--sanitize`; `go test ./examples -sanitize`, `go test ./driver -sanitize` on Linux) | the runtime built with `-fsanitize=address,undefined` and a tiny `VELES_GC_THRESHOLD`, run over every example (a test flag or a script under `internal/`); clean, or each finding fixed with a test |
| A3 | **Done 2026-09-28.** **Verify D21 ships**: `+` checked in debug, wrapping in release, `+%` always; `as` between widths truncates only where written | a codegen golden per profile; checklist §1.4/§2 items ticked or turned into bugs |
| A4 | **Done 2026-09-28** (every diagnostic has a case or the reason no program reaches it). **Conformance suite** (the old P6 item) | `sema/testdata/conform/*.vs` with `// error: text` / `// warning: text` expectations, grouped by D-number, one negative case per diagnostic the checker emits; a test runs them all and lists diagnostics without a case |
| A5 | **Done 2026-09-28.** **Socket close vs a read in flight** (§11) | a per-socket in-use count (Go's fdMutex idea) so a close on one thread waits for, or fails, I/O in progress on another; a threaded test that closes during reads |
| A6 | **Done 2026-09-28.** **Fuzz the HTTP request parser** (checklist §2) | an in-process fuzz target over the request parsing `http.call` shares with `serve`; corpus kept; no panic, limits hold |
| A7 | macOS (later; the user runs it on an M4) | as A1, on macOS arm64 — the aarch64 register capture is so far only cross-compiled |

## B — Daily developer experience (decision-free unless marked)

| # | Task | Acceptance |
|---|---|---|
| B1 | **Done 2026-09-28.** **Every diagnostic names its fix and links its docs** (checklist §6; D79 named families + `veles explain`) | audit of every `errorf`/`warnf` site; each message says what to write; an anchor per error family in `reference/errors.md`, printed as `see: <link>` and sent as LSP `codeDescription` |
| B2 | **Done 2026-09-28.** `veles build --timings` (checklist §3.2) | wall time per phase and per module; docs 01 |
| B3 | **Done 2026-09-28** (D80). `veles test`: parallel tests (`--jobs`); helper tracing through `async helper()` (§11) | tests run as tasks across threads, reports stay in order; a test pins the order |
| B4 | **Done 2026-09-28** (D81). **Panic stack traces with symbol names** (checklist §4, D49) | a panic in a task prints the Veles call chain, `at file:line` per frame; the crash-report format documented |
| B5 | **Done 2026-09-28** (D82). `os.run` with stdin and a separately captured stderr (checklist §5.10) — **public API, ask first** | both platforms; docs 15 |
| B6 | **Done 2026-09-28.** LSP leftovers | code action "implement missing trait members"; whatever the B1 audit shows missing |
| B7 | **Done 2026-09-28.** `veles new --template server` (checklist §6) | logging, `/healthz`, graceful shutdown wired; runs and tests first try |
| B8 | **Blocked 2026-09-28: not by speed (with `reserve` the prelude form is as fast at -O2) but by effects — the lowered adapters let a lambda throw or suspend (`xs.map(a => try eval(a))` is a `Result`); a prelude function would need to be generic over its argument's effects. See the progress log.** Move the Go-lowered eager adapters into the prelude (notes #14) | benchmark first (`veles-bench`); move only what is not slower at -O2 |

| B9 | **Done 2026-09-29.** **D86: numeric conversions are methods; `as` only renames** (decided 2026-09-29) | `toT()` / `wrapT()` on all ten numeric types and `p.cast<*raw U>()`; `x as T` an error with a fix and a lint migrating std, examples, docs and tests; literal overflow is a compile error; formatter, LSP hover, docs (tutorial, cheat sheet, errors), a codegen golden and sema conformance cases; Windows and WSL green |

## C — Standard library breadth (each public API asked first)

In this order, because each unblocks the next real program:

1. **Done 2026-09-29.** **`std/log`** — levels, structured fields, request-scoped through
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
| S1 | **Done 2026-09-29.** **Refresh `veles-selfhost-frontend-plan.md`** against D60–D78 | its §4 "not gaps" list re-checked (a module-level `var` is now an error unless `Mutex`/`Atomic` — the keyword table and any interning move to a `val` or into a struct; `this`; `init(params)`; D78 tests for the harness; the UAX #31 table generator); stale claims struck with the date |
| S2 | **Done 2026-09-29.** **Compiler-shaped benchmarks** in `bench/` | Veles vs Go on: tokenising a large file, building a sealed-tree AST and walking it with `when`, a string-interning map, emitting text through `StringBuilder`; recorded, and each gap over 3× Go gets a checklist item |
| S3 | **Done 2026-09-29.** **Capability audit for a whole compiler**, not only the front end | a section in the self-host plan: what sema/codegen in Veles would lean on (large pointer graphs under the GC, maps keyed by structs, deterministic iteration order, sorting, `os.run` of clang, file I/O, deep recursion) — each marked has / gap, gaps become checklist items |
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

**A1 done: Linux x86-64.** Go 1.23.2 installed in WSL user space
(`~/.local/go`, no sudo; clang 18 was already there);
`internal/wsl-test.sh` syncs the Windows working tree into a clone at
`~/veles` and runs the suite there. Green: `go build/vet/test ./...`,
every example under `VELES_THREADS=1/2/4/8` ×5, `VELES_GC_THRESHOLD=256`,
`VELES_GC_POISON`, the driver ×3; the `posix_spawnp` branch of `os.run`
compiled and its test passes. Found and fixed: (1) the runtime did not
compile — glibc hides `pthread_getattr_np` without `_GNU_SOURCE`, now
defined at the top of every runtime file on Linux; (2) **Windows wrote
stdin/stdout/stderr in C text mode**: every `\n` printed became `\r\n`
and stdin ended silently at the first 0x1A byte — binary mode now, so a
program's output is the same bytes on every platform (driver
`TestStandardStreamsAreBytes`, red before); every `expected.txt` is now
plain LF; (3) two doc samples were platform-dependent: glibc's `cbrt(27)`
is 3.0000000000000004 (the FFI sample now calls `ldexp`, exact by
definition), and the cancellation sample raced a 1 ms sleep against a
child's start (it now waits on a channel; docs 12 says a task cancelled
before it starts never runs). Bench recorded from Linux: single-threaded
work matches Windows; `spawn` is 3.9× Go against 1.1× on Windows —
checklist §3.1 item.

**Typo suggestions (B1, first part).** `sema/suggest.go`: every unknown
name, function, method, field and module member says what was probably
meant — edit distance with transpositions, ignoring case, long prefixes
(`toUppercase`→`toUpper`), ties to the longest shared subsequence
(`printn`→`println`), and other languages' names (`size`→`len`,
`append`→`push`, `get` on a List→`at`, `has`→`containsKey`,
`reduce`→`fold`, `lenght`→`len`) offered only when the type has the
target; private members and statics are never suggested. A field called
as a method, or a method read as a field, is told which it is. The guess
is a `source.Fix` with `Guess` set: an LSP quick fix that is not
`isPreferred`, never applied by `check --fix`. Two old annoyances fixed
alongside: a local named only through a typo was "never used", and
`check --fix` renamed it to `_`; a failed call's lambda arguments added
"cannot infer the type of lambda parameter" errors. Tests:
`sema/suggest_test.go`, driver `TestCheckFixLeavesGuesses`, lsp
`TestCodeActionTypoGuess`; docs 01.

**A2 done: sanitizers.** `veles build/run/test --sanitize` compiles the
C runtime with `-fsanitize=address,undefined -fno-sanitize-recover=undefined`
at -O1 (its own object cache) and links the sanitizer runtimes; the Veles
code is not instrumented (bounds-checked already; its heap is the
collector's). Two runtime changes make it sound: the collector's
`scan_range` is `no_sanitize("address")` (it reads redzones on purpose),
and `__asan_default_options` turns off use-after-return fake frames, which
would keep C locals off the stack the collector scans. Test flags:
`go test ./examples -sanitize` (collector every 4 KiB; a sanitizer report
fails the example) and `go test ./driver -sanitize` (the runtime stress and
`veles test` programs). On Linux: every example at threads 1/2/8 ×2–3 and
the driver ×3 clean; one finding, fixed — `buf_append` in the test
runner's capture did `NULL + 0` on an empty print (UB; red under
`go test ./driver -sanitize` without the fix). A deliberate `memset` past a
`malloc` block is reported with the Veles frame in the stack (driver
`TestSanitizeCatchesOverflow`, skipped where clang lacks compiler-rt, as in
this MSYS2). Docs 01 and 13 (a "When C goes wrong" section). Found on the
way: D50's raw-pointer arithmetic (`p + n`, `p - q`, ordering) is decided
and documented but not built — next.

**D50 raw pointer arithmetic built.** Decided and documented (docs 13's
table said "`unsafe` only, C-style") but `p + 4` was "operator '+' is not
defined" — found while writing the A2 probe. Now `p + n` / `p - n` /
`p += n` step by elements (a plain `getelementptr`, not `inbounds`: D50
promises no checks and no assumptions), `p - q` counts elements (byte
difference over the element size), and the orderings compare addresses
(`icmp ult`…), all inside `unsafe`. Errors name the rule: outside
`unsafe`, `n + p`, a non-integer step, `*raw ()` (no size — cast to
`*raw u8`), mixed pointer types. sema/rawptr.go; tests
`sema/rawptr_test.go`, golden `codegen/llvm/golden/rawptr` (IR and run);
docs 13 has a runnable sample; `examples/ffi` now turns `bsearch`'s
result pointer into an index with `p - q`.

**A3 done: D21 verified, four edge bugs fixed.** The golden test now runs
a fixture in a second profile when it has `main.release.ll`: golden
`overflow` is one source that prints "integer overflow" (a panic caught by
`gather`) for `127 + 1` (i8), `0 - 1` (u8), `MIN * -1`, `-MIN`,
`MIN / -1`, `MIN.abs()` and `x += 1` in debug, and the wrapped values in
release; `+%`, `as` truncation and the range cases print the same in
both. Bugs found probing type edges: (1) **an inclusive range ending at
its type's maximum looped for ever** in every profile (`loop (i in
250..255)` over u8: the index wrapped to 0 and `i <= hi` held) — the loop
lowering and the prelude's `RangeIter` now stop on a flag set at the last
value, Rust's `RangeInclusive` approach; bench before/after (3 runs each,
all workloads) shows no change, LLVM folds it; (2) `Range.len()`,
`step()` and `RangeStepIter.reversed()` measured spans in `T`:
`(-128..127).len()` panicked, `(-128..127).step(100)` silently yielded
just `[-128]`, and an exclusive range ending at MIN counted as nearly
full — spans are wrapping distances now (negative only past half the
type, where no step is too long); (3) `MIN.abs()` returned MIN silently in
debug while `-MIN` panicked — now an overflow like it (catalogue updated).
Three ergonomic questions went to §9 Q17 (a one-task `gather` is a
1-tuple; `async f()` on a function value; `Range.isEmpty`), and two gaps
to the checklist (generic prelude bodies unchecked until used; no
inference of a type argument from the expected type).

**A4 done: the conformance suite, and the cascades it exposed.**
`sema/conform_test.go`: case files under `sema/testdata/conform/` (20,
by D-number) end each offending line with `// error: text` or
`// warning: text` (several per line allowed); a case must provoke
exactly what it lists. Coverage is by diagnostic format: the test parses
the checker's source for every `errorf`/`warnf`/`errorFix`/`warnFix`
format (string constants resolved), records what a full `go test ./sema`
reported (a hook in the four Checker functions, set only by the test's
`TestMain`), and fails for a format nothing provoked that
`uncovered.txt` does not list — so a new diagnostic comes with a case,
and a covered one must leave the list (`-conform-update` rewrites it,
keeping `# why` notes). 267 uncovered at the start → 65: 33 with a
reason (parser-guarded, std-internal intrinsics, two-module M5 rules,
defensive AST cases), 32 still to write. `.gitignore` ignored every
`testdata/*`; the conform directory is excepted (it would have been left
out of the commit). Writing the cases found **13 checker defects**, all
fixed and pinned by the cases: cascading second errors after a generic
method's type parameters differ from the trait's, after a value-`if`
without `else`, after a generic argument that cannot fit, after `when`
arms of incompatible types, after a lambda parameter declared with the
wrong type, after every `race` arm was refused; names bound by a refused
`when` pattern, `is` pattern or let-else destructuring reported as
unknown or never used; `Some`/`None` on a non-nullable subject reported
as "'Some' is not a type"; a function or lambda body ending in a call
with no value reported "expected 'i64', found '()'" instead of "missing
return"; `try`/`throw` in a global initializer blamed an "enclosing
function"; the needless-`throws` lint firing on a body that failed to
check; and `sealed trait X : Named` without methods silently dropping its
supertraits.

**A5 done: closing a socket while another task uses it.** `std/net`
wraps the number in a `Socket`: every `accept`/`read`/`write`/
`shutdownWrite`/`port` holds it for the call (`with (held =
fd.using())`, a count in one atomic with a closing bit); `close()` sets
the bit, and the runtime's new closing set (`veles_io_closing` /
`veles_io_closed` in `veles_task.c`) wakes every task parked on that fd
and refuses new waits, so they fail with an `IoError` at once. The
number is released by whichever of `close` and the last in-flight call
finishes second — a call can no longer reach a number the system handed
to a newer socket. Red before: driver `TestSocketCloseDuringRead` hung
(a read parked on a closed socket never woke, verified with HEAD's
`net.vs`); green after, 200 rounds at `VELES_THREADS` 1 and 8, on Windows
and Linux, and ×3 under `--sanitize`; `httpd`/`shutdown`/`cancel` ×5 at
2 and 8 threads. §11 limit removed, chapter 16 says what `close()` from
another task does.

**A6 done: the HTTP request parser, fuzzed over a real connection.**
Not through `http.call` (it never parses bytes) but through `serve`
itself: `examples/fuzz` runs `http.serve` in the same process on a
loopback listener with small `Limits`, and each run writes one to three
requests on a fresh connection and closes its side. The last request
carries one of 21 faults, each with the status RFC 9112 asks for (414,
431 ×3, 413, 400 for framing and syntax, 501, 505), or the whole write is
mutated; the client parses everything that comes back and fails on a
malformed response, a 500, a refusal that keeps the connection, a wrong
echo of method, decoded path or body, or a read that waits (5 s — the
client has closed, so the server has no reason to). Against HEAD's server
the first 500 runs failed 225 times. Found and fixed in `std/http`:
**response splitting** — a path decoding to `\r\n`, echoed into a header
by the handler, let the client write headers into the response; values
are now written with CR/LF/NUL as spaces, names that are not tokens are
dropped, and `content-length`/`transfer-encoding`/`connection` belong to
the server (a handler's `connection: close` is still obeyed). A request
line or header that is not UTF-8 closed the connection without an answer
(now 400). HTTP/1.0 was kept alive by default. And the head was read
leniently where a proxy could read it differently: space before a colon,
folded lines, non-token names and methods, CR or NUL inside a value,
`Content-Length: +5`, two differing lengths, a missing or doubled `Host`,
`HTTP/1.1x` — all 400 now. The ten inputs are a corpus checked every run
(all ten fail at HEAD). 30 seeds × 3000 runs at 1/2/4/8 threads on Linux:
0 failures. On Windows, a long sweep runs out of ephemeral ports (closed
client ports sit in TIME_WAIT for 120 s) — the default 500 runs are well
inside that. Chapter 17 says what the server refuses and why.

**B1 done: every diagnostic names its fix and its family (D79).**
The user chose named families (recommended of 4) for the docs link.
`source/family.go` sorts every message into one of 34 families by its
wording; the CLI prints `see: veles explain <family>` once per family
under the errors, `veles explain <family>` prints that section of
`reference/errors.md` from the copy compiled in (`docs/errors.go`;
offline, version-matched, "did you mean" on a wrong name), and the LSP
sends the family as `code` with the GitHub anchor as `codeDescription`.
`errors.md` is rewritten as one section per family, the old entries
under theirs, and the panics in their own part. Two tests hold it: sema
`TestEveryDiagnosticHasAFamily` fills each of the 612 formats of
sema/parser/lexer with sample arguments and needs a family for it (it
caught the first table matching formats but not real messages — a `%s`
in a pattern now stands for any text); docs `TestEveryFamilyIsExplained`
needs a section per family and no stray one. The audit, rule: a message
names what to write, unless its fix is simply undoing the one fact it
states, or it is compiler-internal. About 45 messages rewritten — a
private name says to declare it `public` there (or that a std name is
not API); a built-in's arity quotes its signature (`'at' takes 1
argument: at(i: i64): string?`); a literal that does not fit names its
type's range; `%` on floats points at `mod`; a bare member inside a
method says `this.twice` (a certain fix, not a guess); `Type.method`
says it is a method; a method implementing a trait method it does not
have gets "did you mean"; a global initializer that suspends or opens a
scope says to compute it in `main`; missing fields, impossible patterns,
unreachable code, missing returns, a value from a unit function,
`break` outside a loop, labels, Sendable, generic functions as values —
each says what to write. Found on the way: `true - false` told the user
to implement `Subtractable` for `bool`, which no program can do — the
"implement it" hint now appears only for the program's own structs and
sealed traits; `atOrPanic()` and a non-comparable constant pattern each
reported one mistake twice. Families in chapter 1, the command table,
the index, and the veles-test / veles-docs skills.

**A4 closed: every diagnostic has a case or a reason.** The 32 left
without a case got one or a written reason: 10 new case files (sealed
variants, trait objects, `when` values, constructors and members, `init`
reads, removed reads and indexes, error unions, the old `@test`, tasks
in globals, operators) and two package tests for the two-module rules
(`TestPrivateAcrossModulesSaysHow`, `TestExternCSymbolIsUnique`); 46
listed, every one with the reason no program reaches it. The test now
also fails for a listed entry without a reason. Writing the cases found
a **missing check**: a variant of a sealed trait with no `implement` of
the trait's methods was reported only when a call dispatched to it (and
then naming whichever variant came first, even one whose method sat one
level too far out) — it is now an error at the variant's declaration,
with the method to add, or "move it inside 'implement Shape { }'".
Hit a wall: `T.Item` with two bounds declaring `Item` silently takes the
first bound's; refusing it needs a way to name the other, which is
language design — checklist §9 Q18.

**B2 done: `--timings`.** `veles build|run|test|check --timings` prints
each phase (load with files/bytes/modules, check, codegen with the IR's
size, the runtime objects, clang with its flags) and the ten modules
that cost the most to parse + check (`sema.LoadPackageTimed`; the loader
times each parse, `drainQueue` each body, charged to the module it is
declared in; nil when not asked). Driver tests pin the report and the
duration format. First reading, `examples/httpd` debug: 939ms total —
clang 853ms on 2.7 MB of IR, the whole front end 27ms. So build time is
IR size: 630 std functions make 2.3 MB, and one body can be large
(`http.readRequest`, ~100 lines, is 109 KB of IR) because lowering
writes adapters, checks and interpolation inline — which makes B8 (the
eager adapters as prelude functions) a build-time item as well as a
runtime one (checklist §3.2).

**B3 done: tests run at once (D80), and helpers are traced through
`async` and panics.** The user chose parallel by default (recommended of
4). The runtime keeps one record per test (failures, captured output)
bound as a task-local value of the test's task, so the tasks a test
starts record into it; the runner queues every test (`veles_test_queue`:
up to `--jobs`, default one per runtime thread; a test that ends starts
the next), then waits for them in declaration order with `veles_run`
and prints each report whole — the output is what it was, test by test.
The watchdog watches every running test; its report says how many others
had not finished and that `--jobs 1` runs them one at a time. Found on
the way: `veles_run` reset the executor's clock base on every call (it
was only ever called once per test, sequentially); the queue and the
records were reachable only from C globals the collector does not scan —
`VELES_GC_POISON` crashed `--jobs 1` until they became roots. Helper call
sites: a test helper launched with `async helper()` is traced (the site
is bound around the launch, so the task inherits it), and a panic inside
a helper lists the test's calling lines under its `at` line — both §11
limits gone. Pinned by `TestTestsRunAtOnceReportInOrder` (four sleeping
tests overlap, `--jobs 1` does not, reports in order with each test's
own output), `TestTestHelperPanicCallSites`, the async case in
`TestTestHelperCallSites`. Windows and Linux; Linux also under
`--sanitize`, `VELES_THREADS` 1/2/4/8 ×3 and GC poison.

**B6 done: "Add the missing methods".** An implement that leaves out
methods of its trait carries, on each of its errors, one fix that adds
them all — each as `fun name(...): T = panic("'name' is not written
yet")`, indented into the block whether it was `{ }` on one line or a
body with methods in it; a sealed variant with no `implement` of its
trait (the check added under A4) gets the whole block the same way.
Scaffolding, so it is a guess: the editor offers it, not preferred, and
`check --fix` never applies it. `TestAddMissingMethodsFix` applies each
fix and checks the result compiles. The B1 audit left nothing else for
the language server. Docs: chapter 8, the VS Code README.

**B7 done: `veles new --template server`, and live stdout.** The
template is an HTTP service that builds, runs and passes its tests as
created: a Router with `/healthz` and a JSON endpoint, `requestId` and a
5-second `timeout` wrapped around every route, request logging (serve's
own), HOST/PORT from the environment, `http.serve(..., stop:
os.shutdownSignal())`, and three tests of the handlers through
`http.call` — no port. `veles new` names the templates (`app`, `server`)
and refuses an unknown one; the old hint named a test (`greetsByName`)
that had been renamed. Driver `TestNewServer` checks, tests and builds
it, starts it with PORT=0, reads the address from its first line and
asks `/healthz`. That first line found a **real bug**: on Windows the
line never arrived — C's runtime ignores `_IOLBF` there and fully
buffered stdout, so a Veles service under Docker or a supervisor would
log nothing until it stopped. Standard output is now written at each
line end unless it is a regular file (measured: a million lines to a
file stay at 0.18s; flushing each would have been 2.1s), on every
platform; Linux had been line-buffering even files. `TestNew` now runs
the app template's test too, not only a check.

**B8 measured, and blocked.** The prelude form of an eager adapter —
`fun mapV<T, U>(xs: List<T>, f: fun(T): U): List<U>` with a push loop,
and the same for `filter` — against the Go-lowered `xs.map(...).filter(...)`,
5M elements ×10, `--release`: 1.15s against 0.78s, **45% slower**, stable
over three runs. The lambda stays an indirect call nothing inlines, and the
result grows by pushing where the lowering allocates it once. So B8's own
rule says: do not move them. What would make the move free is a way to
specialise a generic function for the function value it is called with —
a compiler pass (per-call-site specialisation of higher-order generics,
which LLVM then inlines), or a language form like Kotlin's `inline fun`,
which is a decision. Until one exists the adapters stay lowered (and are
most of the IR size measured under B2). Also found writing the benchmark,
and recorded for their own decisions: `MutableList` has no `reserve(n)` (public API: asked separately). Two
diagnostics were wrong and are fixed, with cases: `List.generate(n, ...)`
was told to write `List<T>.generate`, which does not exist either — it now
says "no static function 'generate' on type 'List'; did you mean
'MutableList<T>.make(...)'?" (the nearest static of the type or of its
mutable counterpart, other languages' builder names leading to `make`);
and a lambda passed after an argument that had failed was reported as
"cannot infer the type of lambda parameter" — the same mistake twice, now
checked loosely.

**B8, measured again with `reserve` (D83): speed is not the blocker.**
With `out.reserve(xs.len())` in the prelude form, `mapV`+`filterV` over
5M ×10 at `--release` ran 0.53s against the lowered adapters' 0.56s (three
runs each) — the 45% earlier was regrowth, and LLVM does inline the lambda
once `mapV` is inlined. The real reason the adapters are lowered in Go is
effects: a lambda given to `map`/`filter`/`fold`... may throw
(`args.map(a => try eval(a))` types as `Result<List<U>, E>`, lower_try.go)
or suspend, and the lowering carries that out of the loop. A prelude
function would have to be generic over its argument's effects —
`fun map<U, E>(f: fun(T): U throws E): List<U> throws E`, with `E = Never`
meaning "does not throw", and the same for `suspends`. Next step when B8's
turn comes back: find whether the checker already instantiates `throws E`
with `Never` to a non-throwing function, and what suspension-generic would
mean; if either is a language change, it goes through `veles-decide`.

**Decisions D81–D84 (user, all recommended), and three of them built.**
Asked together: panic call chains (D81, a shadow stack in debug builds —
after the user asked for DWARF to be explained next to it), `os.run`
input and stderr (D82), list capacity (D83), and Q18 (D84).
- **D83 `xs.reserve(n)`**: a built-in (`veles_list_reserve`), in the
  catalogue and hover, allowed on a local that is later moved (D63's
  method list), refused on a `List`; chapter 4's `build(n)` uses it.
- **D84**: `T.Item` with two bounds declaring `Item` is an error naming
  both and the qualified form; `T.Trait.Item` resolves through that bound
  (a run returns each implement's value). Conformance
  `D84-associated-types.vs`; chapter 8.
- **D82 / B5 done**: `os.run(program, args, input: "", stderr:
  os.Stderr.Capture)` with `Capture | Inherit | Merge` (capitalised like
  every enum's members — the question had them in lower case) and
  `Output.stderr`. Both platforms feed the input and read both outputs at
  once — Windows with a writer and a reader thread, POSIX with one `poll`
  loop (and SIGPIPE ignored, so a child that stops reading is an EPIPE) —
  so a child that writes a megabyte of errors before reading its input
  cannot deadlock (driver `TestRunInputAndStderr`, with a 60s guard; also
  under `--sanitize` on Linux). No input is an empty input now, not this
  process's. `mergeStderr:` still parses, with an error whose fix writes
  `os.Stderr.Merge`, or `Inherit` for `false` (the old default).
  Chapter 15, the stdlib reference, the cheat sheet.

- **D81 / B4 done**: in a debug build every call the programmer wrote is
  bracketed by `veles_call_push("site callee ")` / `veles_call_pop`
  (`codegen/llvm/expr.go`); the stack is the task's own (`veles_shadow`, in
  collected memory), one per thread outside tasks, and a suspending call's
  child task links to its caller (`veles_call_link`) so the chain runs
  through suspension. A panic copies the chain as text (`panic_trace`), a
  scope's re-raise carries the child's chain up, and the test report shows
  it under `at` (a helper call site the chain already has is not printed
  twice; helper lines now indent like the rest). Release builds emit
  nothing and print `(a debug build shows the call chain)`. Deeper than
  4096 frames the chain is dropped rather than shown wrongly. Tests: driver
  `TestPanicPrintsCallChain` (three calls, through a suspension, release
  note), updated test-runner reports; the goldens carry the push/pop.
  Chapter 7, the errors and stdlib references.

- **2026-09-29, D85 — named imports (checklist Q15 decided, built)**:
  `use io { println, eprintln as warn }`, spelled with `as`, braces add bare
  names on top of `io.` (user's choice among `from … import`, `::`, `:`,
  `as`). `ast.UseSpec.Names`; the parser reads the braces, refuses `{ * }` and
  `{ }`, and reads the removed `use m.{ }` with a fix that drops the dot;
  `declareUseNames` binds the module's own symbols into the file's import
  scope (a copy under the alias), refuses private, test-code, missing (with a
  typo guess), prelude-global and prelude-home names, and collisions with
  another import or a module declaration; `lintUnusedNames` warns with a
  fix (the scope records which names a lookup found). Formatter sorts the
  names; completion offers the module's names inside braces and the bound
  names elsewhere; TextMate colours them. Tests: conform `D85-*` (three
  files), `TestNamedImports` (across modules: type, enum, function, alias,
  shadowing, private, test code, unused), format and LSP cases; example
  `examples/named`. Docs: chapter 11, cheat sheet, errors reference. Not
  done: auto-import on completion of a name the file has not imported;
  rename of an aliased name follows the member, not the alias. `as` for
  conversions (Q16) is untouched.
- **2026-09-29, D85 follow-ups** (the two "not done" items above are done,
  and the tree is migrated): auto-import on completion — the standard
  modules' public names are read once from the embedded sources, the loaded
  modules' from the package; the edit goes into the braces, after a bare
  `use m`, or on a new line, and only with a typed prefix. A renamed import
  is a declaration of its own (`finishImportRefs`): uses of the alias
  reference the alias, hover says `alias of io.eprintln`, an unrenamed bare
  use hovers `module io`, and renaming the alias leaves the member alone.
  Migration: `io.println/print/eprintln/eprint/readLine` became bare in std,
  examples, bench and the `veles new` templates (63 files; a small Go
  program did the text change, `veles fmt` the layout; every expected output
  unchanged). Docs and the compiler's own Go tests keep the qualified form.
  Tests: `TestAutoImportOnCompletion`, `TestRenamedImportHoverAndRename`.

- **2026-09-29, B9 / D86: numeric conversions are methods, `as` only
  renames.** `x.toT()` on all twelve numeric types (`i8…usize`, `f32`, `f64`):
  total where nothing can be lost, a `T?` (null when out of range) otherwise;
  a float truncates toward zero and NaN is null. `x.wrapT()` between integers
  keeps the low bits; a literal that cannot fit (`300.toU8()`) is a compile
  error. `p.cast<*raw U>()` replaces `as *raw U` (unsafe). Codegen:
  `num.toChecked` (range test by converting back and comparing, with the sign
  flip check; floats by bounds, NaN false) beside the existing cast. `x as T`
  keeps parsing with an error naming the method and a fix — `veles check --fix`
  migrated std, examples, bench and the goldens (~140 sites; nested casts took
  two passes); docs (chapters 2, 3, 6, 7, 13, 19, cheat sheet, errors
  reference: three new anchors under `numbers`) and the Go tests' embedded
  programs by hand. Two float→int sites became explicit decisions
  (`Duration.ofSeconds` panics when the count does not fit, `jwt` refuses an
  out-of-range NumericDate) and `Value.asI64` reads a float only when
  integral. Hover/completion entries for every method. Tests: conform
  `D86-conversions`, `examples/conversions` (edges of every family), golden
  `arith` regenerated. Q19 (decided the same day): `x.wrapTo<T>()` is
  `wrapT()` with the target as a type argument, for generic code (the range
  iterators use it); the embedded-std `as` exemption is gone. Also: `internal/wsl-test.sh`
  cleaned the clone after, not before, checking out HEAD, so a file the
  commit began to track aborted the checkout and the run used a stale base;
  order fixed.

- **2026-09-29, hover shows an inferred `suspends`** (user's ordering: this,
  then Q3/Q4/Q8, `std/log`). Hovers were rendered from the written signature
  before the suspension pass ran, so only the inlay hint knew. Now
  `showInferredSuspends` (`sema/index.go`) adds ` suspends` to the hover of
  every reference to a function whose effect was inferred, at its declaration
  and at each call, methods included (before ` throws`). Test
  `TestHoverShowsInferredSuspends` (declaration, transitive, call site,
  method, and a pure function that must not say it); docs chapter 12.

- **2026-09-29, Q4 / D87: `protected` on a mutable-collection field is "look,
  don't take".** `sema/protected_contents.go`: a read of such a field from
  outside the type is refused unless it is a method receiver, a value-loop
  head or an interpolated value (`lookOnly` marks the expression checked
  next); a receiver is then refused when the method mutates — a built-in of
  the mutable family (the catalogue lists those apart), or an `extend` block
  naming the mutable type itself. `loop (&x in …)`, binding, passing and
  returning are refused; `.toList()` copies. Nothing in std or examples used
  a protected collection, so no migration. Tests: conform
  `D87-protected-collections` (every refusal and every allowed look), docs
  chapter 5 (a run block), errors reference. Notes R18(b) and R32 leave
  the notes file.

- **2026-09-29, Q8 / D88: `@caller_location`.** A marked function takes the
  site it is called from as a hidden last parameter (`Func.CallerLoc`; the
  checker refuses the attribute outside std, on trait implementations,
  externs, and on anything that suspends); a call passes its own span, or —
  from inside another marked function — the site it was itself given, so
  `n.toString(radix: 1)` through `checkRadix` reports the user's line; a
  `panic(...)` in the body reports that site (its own line when the site is
  empty: a compiler-made call, a function value's thunk). Both profiles. The
  debug chain says the site once: the runtime finds the chain line whose site
  is the panic's, prints `at SITE in CALLER` and drops the std frames above
  it. Marked: `swap insert removeAt chunked windowed step` (both), `toString
  (radix:)` + `checkRadix`, `randomBytes`, `Uuid.of`, `random.range`. Not
  marked: trait implementations (`Hmac.update`), the JWT key check, and
  `mapConcurrent` (suspends). Tests: driver `TestCallerLocationPanics`
  (debug + release, through two marked frames), conform `D88-*`, goldens
  regenerated (an extra argument on every marked call); docs chapter 7 and the
  errors reference.

- **2026-09-29, Q3 / D89: `public use`.** Parser and formatter: a `public use`
  is a declaration of its own (`UseDecl.Pub`; the formatter neither merges it
  into the run of plain imports nor sorts it). Checker (`sema/reexport.go`):
  `public use m` puts the module symbol into the module's scope under its name
  or alias, `public use m { a, T as U }` puts those items there (the module's
  own symbol, so hover and go-to-definition land on the declaration);
  collisions with declarations are errors; std and dependencies cannot be
  re-exported; a re-exported name is never "unused". Loader (`resolveDep`): a
  dependency's surface is read from its root module's `public use` — one per
  path segment, so a module passes on what it re-exports — replacing the
  manifest list; the error names the line to add. The manifest's `exports` is
  an error naming the replacement (`Manifest.Exports` and `exported` are
  gone); `veles doc` lists the root plus what it re-exports. Migrated:
  `examples/packages` (root re-exports `geometry` and flattens
  `support { root as sqrt }`, a module that stays hidden). Tests:
  `TestPublicUse` (facade, alias, hidden module, flattened-only module, chain,
  std/dependency refused, both collisions, manifest error), conform
  `D89-public-use`, format `TestPublicUseIsNotMerged`, LSP
  `TestDefinitionFollowsReexport`, docs chapter 11 ("What a package shows"),
  errors reference, cheat sheet. Notes #19 leaves the notes file.

- **2026-09-29, C1 / D90 + D91: `lazy` parameters and `std/log`.**
  `lazy` (parser, `ast.Param.Lazy`, `types.Param.Lazy`, `sema/lazy.go`): a
  parameter of type `fun(): T` (no arguments or effects, std-only) takes a
  plain expression at the call, wrapped in a lambda by `bindArgs`;
  formatter, hover and signature help spell the modifier. `std/log`: `Level`
  (Debug, Info, Warn, Error, Off), `debug/info/warn/error(lazy msg, Field...)`,
  `field<T: Encodable>`, `setLevel`, `enabled`, `withFields` (a `TaskLocal`,
  D72), `VELES_LOG`; text on a terminal (a runtime `veles_stderr_is_terminal`,
  both platforms), JSON otherwise; one `eprintln` per line, which the runtime
  already writes whole under the stream lock. `http`: `serve`, `logging()`
  and the handler-failure lines go through `log`; `requestId()` binds `id`
  for the request, so `logging()` no longer reads the header itself (it needs
  `requestId()` outside it, as documented). Tests: `std/log/log.test.vs`
  (both line formats, quoting, level names, thresholds — run by
  `TestStdLogUnitTests`), `TestLogInAProgram` (typed JSON, laziness, scope
  through a child task, `VELES_LOG`, 400 lines from 8 tasks all whole),
  conform `D90-lazy`, format round trip; docs chapter 21, the stdlib
  reference, chapter 17. Measured: a disabled `log.debug` is ~30 ns (a
  closure per call), the guarded form ~0.3 ns — checklist §5.7 keeps the
  hoisting as an open item.

- **2026-09-29, S1–S3: self-host readiness.** S1: `veles-selfhost-frontend-plan.md`
  refreshed against v0.49 — line counts (6353 non-test lines, was 5472), corpus
  size (173 files), the "module-level `var` is a real mutable global" claim
  struck (D66), stale `binarySearch` and `partitionPoint` references fixed,
  and a new §8 listing what D60–D91 change for the port (`this`/`init`, D78
  tests, D86 conversion methods, D85 named imports, D89 `public use`, panic
  chains, `lazy`/`std/log`). S2: four compiler-shaped benchmarks, each with a
  Go reference (`bench/compiler_refs.go`; the runner now subtracts the setup
  a reference does before Veles's clock starts, `goSetup`): `lexer` (byte
  scan, keyword map, token structs), `ast` (sealed family built and walked
  with `when`), `intern` (string-keyed symbol table), `emit` (250 000 lines of
  IR through a `StringBuilder`). Ratios to Go 0.4–2.0×, none over 3×, so no
  checklist item; the Go side swung 2–3× between runs on a loaded machine
  (caveat in `bench/results.md`). S3: §9 of the self-host plan audits what a
  whole compiler would lean on, each row probed with a program: large
  cyclic pointer graphs, struct- and sealed-keyed maps, insertion-ordered
  iteration, stable sorting, `os.run` of clang, file I/O, parallel tasks all
  **have** it. Two findings became checklist items: stack exhaustion is
  silent (exit 127 / SIGSEGV at ~30–50k frames on Windows, ~200k–1M on
  Linux) and the stack size is the OS default (§2), and there is no safe
  `f64`↔`u64` bit reinterpretation (§5.10, needs a decision). Not done: the
  compile time and peak memory of a 30 000-line program by a Veles-written
  compiler, which belong to E4.

- **2026-09-29, stack overflow (S3 finding): D92.** `runtime/c/veles_stack.c`
  is new: the process's `main` (the generated one is now `veles_main`) starts
  the program on a thread with a 256 MB reserved stack (`VELES_STACK=<MB>`,
  1–4096; a refused reservation is retried at half down to 8 MB), and
  `veles_thread_spawn` gives every worker the same, while the timer monitor
  and the mutex watchdog use `veles_thread_spawn_small` (OS default). A fault
  handler — SIGSEGV/SIGBUS on a per-thread `sigaltstack` on POSIX, a vectored
  exception handler with `SetThreadStackGuarantee(64 KB)` on Windows — prints
  `panic: stack overflow`, the size and, in a debug build, the innermost
  calls of the D81 chain (`veles_shadow_peek` in `veles_task.c`), using no
  allocation and no formatted output, then exits 101. Found on the way: below
  the 8 MB floor the size loop never tried a thread (fixed), and `_exit` needed
  more stack than a 1 MB stack leaves on Windows (`TerminateProcess`).
  Test: driver `TestStackOverflowIsReported` (a million frames in main and in
  a task; overflow in both profiles; `VELES_STACK=1`; a bad value), green on
  Windows and WSL, as is the full suite; `spawn`/`parallel`/`channels`/`pipes`
  benchmarks unchanged. Docs: chapter 7, chapter 3, the stdlib reference,
  `veles-debug`. Open: a frame bigger than the guard region (clang stack
  probes), macOS (A7).
