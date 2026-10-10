# Veles — Production Checklist

The working tracker for everything the language and its standard library
still need, can do better, safer or faster, before Veles can carry a
production server (and, more generally, the kind of program Go, Rust or C
are trusted with — with the best developer experience we can manage).

`veles-spec.md` records *decisions*; this file records *work*. An item is
ticked when it is implemented, tested (an example with `expected.txt` or a
`sema_test.go` case) and documented under `docs/`. Open design notes that
are taste rather than work stay in `notes_to_change.txt`.

Legend: `[ ]` open · `[~]` in progress · `[x]` done · `[?]` blocked on a
decision (see §9) · `→ Dn` the spec entry that settles it.

---

## 1. Language

### 1.1 Derivation — the current focus

Everything under §5 that maps a struct to the outside world (JSON, rows,
config, CLI args) needs the compiler to synthesize code from a type's
shape. The precedent exists: `==`/hashing are structural, `Sendable` is
answered from the shape, tuples get `Comparable`, enums get
`toString`/`parse`/`values` — all overridable by writing the `implement`.

- [x] **Mechanism** decided: an empty `implement Trait for T { }` asks the compiler to
      synthesize the body; written methods are kept; foreign types allowed (§10)
- [x] **Target** decided: generic `Encodable`/`Decodable` (`Codable`) with
      `Encoder`/`Decoder` traits; JSON, rows, config reuse one derive; the
      `Json` tree is itself `Codable` for dynamic use (§10)
- [x] Field-level attributes with arguments the derive can read (`@key`, `@key(json: ..)`, `@skip`, `@tag`, `@required`) → D51 revisit
- [x] Sealed traits: internally tagged, `@tag`, `content:` for the adjacent layout
- [ ] Sealed decode fast path: dispatch in-stream when the tag is the first key (today every sealed value buffers through `Value`)
- [x] Enums: by name, number via encoder option
- [x] Nullable fields: missing → `null`, defaults on missing, `@required`
- [x] Generic structs: bounds inferred for an empty implement; implement bounds are now checked at every method lookup and in `implements`
- [x] Recursive types: `json.Options(maxDepth: 64)` in the decoder
- [x] Derived code reaches prelude helpers (`styleKey`, `childPath`, `joinPath`, `panic`) through `ast.PreludeName`, so a module declaring a function of the same name no longer breaks `implement Codable` (2026-09-26; was §11)
- [x] Fields that cannot be derived (functions, `Mutex`, raw pointers, handles):
      error at the use site naming the field, like the `Equatable`-key check
- [x] Private fields / constructor rule (M5): decoding is construction (an inline implement may set them; a foreign implement may not)
- [x] Partial override: hand-write `encode`, keep the synthesized `decode`
- [x] Foreign types: `implement Codable for pkg.T` at top level (private fields of a foreign type refuse construction, as for any caller)
- [x] Error model: every problem with its path; a decoder records mismatches and reads on, a nested value that cannot be built is caught by its parent, which finishes its own checks and fails once
      (`DecodeError { problems }`, `Problem { path, message }`, capped)
- [x] Performance: encoding streams straight to text, decoding reads the bytes;
      no tree except for sealed values (see the fast-path item) — not yet benchmarked (§3.3)
- [x] LSP: hover on a derived implement shows what was synthesized — the implements produced, the bounds inferred for a generic target, the signatures, and the wire shape (keys, optional, skipped); the bodies are behind `veles explain <path> --derive [Type]`, which prints them as Veles (`sema/derive_print.go`)
- [x] Derivable set: `Codable` (`Encodable`, `Decodable`) and `Comparable`
      (`==`, hashing, printing already structural); `Default` added 2026-10-02 (D119)
- [x] Supertraits: `trait A : B + C`, transitive bounds, super check on impls
- [x] Trait objects of a trait with supertraits: the table composes the supers' (`objectSlots` in `sema/supers.go`), inherited methods and their default bodies included; a combination trait is an object built from its parts; two supers declaring one name is an ambiguity and object safety is asked of the supers too
- [x] Parser: braceless empty `implement Trait` (body and top level); field attributes kept; formatter drops empty braces
- [x] Spec entry written (D58) with the rejected alternatives

### 1.2 Foreign function interface

- [x] C ABI FFI design → D67 (2026-09-26): extern blocks in any package, native libraries in the manifest; marshaling of strings/buffers and callbacks → D69
- [~] `extern "C"` blocks: calling convention and callbacks into Veles done (D69, 2026-09-26: `extern "C" fun` + `&name` : `extern fun(...)`, called through inside `unsafe`); variadic calls (D123) built 2026-10-03: `...` in an `extern "C"` block, C's promotions, TestCVariadicCalls (`snprintf`; `open`/`fcntl` on POSIX, `_open` on Windows)
- [x] **Bug (found 2026-10-01): a struct passed or returned by value across
      the C boundary had no C ABI lowering** — `lldiv` (16-byte return)
      segfaulted on Windows, `div` was right by luck. Fixed the same day
      (plan A8): a classifier per target in `codegen/llvm/cabi.go`; driver
      `TestCStructsByValue` (17 shapes × 3 directions + register
      exhaustion, debug and release) on Windows and Linux, codegen
      `TestCStructClassification` against clang for Win64, SysV and
      AAPCS64 (ARM64 not run on hardware until A7)
- [x] C layout (D120) — built 2026-10-03: `@packed`, `@align(n)` (struct and
      extern field), `extern union`, `@transparent`; one layout for checker
      and codegen (`types/layout.go`), padded LLVM types, aligned allocas,
      globals and collector bodies; TestCLayoutAgreesWithC (C fills and reads
      each shape, sizeof/offsetof/stride agree, alignment on stack and heap)
- [x] `Array<T, N>` with const generics (D121) — built 2026-10-03: inline
      arrays, `<const N: i64>` on functions, structs and impls (inferred or
      written, per-`N` instances), constants as type arguments, read-only
      const tables, the memory class for values of 128 bytes and more
      (`memmove` copies, address arguments, `sret` results — a 1 MiB local
      compiles in a second), `Default`/`Codable`/`withRaw`, C arrays in
      extern structs (TestArraysAgreeWithC), TestArrays, conformance,
      golden `arrays`; SHA-256 moved to arrays (113–135 ms → 87–91 ms).
      Not offered yet: `ref(i)`/`sort` on arrays, arithmetic on `N`, a
      native version of every read-only list method (all but seven copy once)
- [x] Ownership at the boundary (D69): C keeps only copies (`ffi.CString`, `ffi.alloc`/`free`), a list is lent for a closure (`withRaw`, `CLayout` elements), a value C hands back travels as an `ffi.handle` (a scanned table index, never a GC address)
- [x] Panics never cross into C: a panic inside an `extern "C" fun` ends the process with its location (runtime `veles_ffi_enter`/`leave` around the body); errors cannot cross either (an exported fun may not throw)
- [x] Linking: `[native]` in `veles.toml` — `libs`, `static-libs` (archive resolved by name), `lib-paths`, `pkg-config`, file entries; dependencies' tables link too (2026-09-26, `driver/native.go`, TestNativeLinking)
- [x] ~~Declaration files (`.d.vs`)~~ — dropped 2026-10-01 (D122): a binding is an ordinary package
- [x] A foreign call cannot stall the collector: every call to an extern
      outside std (and through an `extern fun` pointer) runs in a safe
      region; a callback from C leaves it for the Veles code, and a thread
      Veles did not start is registered on its first callback (2026-09-27,
      TestForeignCallDoesNotStallCollection)
- [x] A blocking C call no longer holds up other tasks (2026-09-27): its
      thread's run queue moves to a spare thread (D66 addendum)
- [x] Raw pointer arithmetic as D50 decided it (2026-09-28; decided and
      documented long before, never built): `p + n`, `p - n`, `p += n`
      element-scaled, `p - q` an element count, `<` `<=` `>` `>=` on
      addresses, all inside `unsafe`; `*raw ()` refuses to step (cast to
      `*raw u8`), `n + p` says to write the pointer first. A plain GEP, not
      `inbounds` (sema/rawptr.go, golden `rawptr`, docs 13, examples/ffi
      turns bsearch's result into an index)
- [x] `--sanitize` for C code the program links (see §2)

### 1.3 Concurrency

- [x] Multi-threaded executor → D66 stage 1 (2026-09-27): N workers
      (`VELES_THREADS`, default one per core) on one run queue; stop-the-world
      collection at safepoints (allocation, suspension, loop back-edges);
      per-thread span ownership, so allocation takes no lock; lock-free safe
      regions around blocking runtime calls (stdin, `os.run`, file reads) and
      foreign calls; `println` writes a line whole
- [x] Work stealing (D66 stage 2, 2026-09-27): a ring per worker plus a shared
      queue, runnext, stealing half a ring; a task's scheduling state is one CAS'd
      word, so taking, running and requeueing a task take no runtime lock, and
      neither do spawn, `await` of a finished task or a successful finish.
      `bench/spawn` (100k tasks) 443 → 39 ms at 8 threads (Go 32 ms)
- [x] The collector records a safe thread's registers in assembly at the
      entry of `veles_enter_safe`/`veles_blocking_enter`: a C helper had lost
      a register it reused (a just-spawned task was swept; TestSpawnDuringCollections)
- [x] `Mutex<T>` is a real lock (2026-09-27): a heap word, one CAS each way
      uncontended, spin then park on striped condvars in a safe region;
      re-locking inside its own `withLock` panics; released when `f` panics
- [x] `Atomic<T>` operations are indivisible (same lock word); `update(f)`
      for read-modify-write (2026-09-27)
- [x] `Atomic` of a machine word without the lock (D66 addendum, 2026-09-27):
      integers, floats and `bool` use seq_cst load/store/xchg/cmpxchg; `update`
      is a CAS loop. 8 threads × 3 atomics × 200k updates: 148 ms locked → 99 ms;
      1 thread 6 ns/op (TestAtomicWordsUnderThreads)
- [x] A blocking call hands its run queue to a spare thread (D66 addendum,
      2026-09-27): a monitor thread (1 ms looks, asleep when nothing blocks)
      hands off a queue whose thread stays blocked while work waits; with
      `VELES_THREADS=1` three 600 ms C calls overlap and timers/computation run
      meanwhile (TestBlockingCallHandsOffItsThread)
- [x] A module-level `var` is an error unless it is a `Mutex`/`Atomic`
      (D66, 2026-09-27); std's UUID v7 clock and random's generator moved
      behind a `Mutex`
- [x] Cancellation documented as a surface (chapter 12 "Cancellation",
      stdlib "Concurrency primitives"): the three causes, where it is seen,
      unwinding order, shielded cleanup, `task.cancel()`, `withTimeout`, and
      `await sleep(Duration.zero)` as the check in a long computation
- [x] Task-local values (D72, 2026-09-27): `taskLocal(fallback)`, scoped
      immutable `withValue(v, f)`, inherited by tasks started inside;
      TestTaskLocalsUnderThreads, sema TestTaskLocal, chapter 12
- [x] `race` send arms (`ch.send(v) => ...`, D108, 2026-10-02): a send waiter
      that claims the race before its value moves; a send arm meeting another
      race's receive arm takes both claims or neither (`race_pair`); a closed
      channel panics. TestRaceSendArms (1/2/4/8 threads ×5), conformance
      D108-race-send, chapter 12. Clean under `--sanitize` on Linux, 1/2/4/8
      threads ×5, with the D107 and D110 programs (2026-10-02)
- [x] `with p = m.lock()` (D107), `with expr` without a name (D109), `retry` /
      `Semaphore` / channel drains / `time.ticker` (D110) — built 2026-10-02:
      TestLockGuard, TestConcurrencyHelpers, std/time/ticker.test.vs,
      conformance D107-lock-guard, D107-lock-call, D109-with-unnamed, LSP
      TestLockInTheEditor; chapters 12, 13, 20
- [x] Which `race` arm wins when several are ready at once (2026-10-02):
      the first ready in written order when the race starts, else the first
      to become ready — deterministic, biased to earlier arms, not Go's
      random pick; chapter 12 and the D38 addendum
- [x] Bounded channels with backpressure (`capacity: n`, blocked senders
      served in order); `race` is the `select` over receives, sleeps and
      tasks (D38). `Channel<T>()` is a true rendezvous (2026-09-27)
- [x] Each channel has its own lock; `race` claims its winner by CAS
      (2026-09-27): `bench/pipes` (8 independent pairs) 1209 → 13 ms at 8
      threads, `channels` 7.2 → 4.6 ms (TestChannelHandoffUnderThreads,
      TestRaceOverChannelsUnderThreads — the latter caught a race re-listing
      its nodes after a wake for another reason, cutting other waiters off)
- [x] Many tasks parked on one channel (concurrency review B1, 2026-10-09):
      appending a waiter walked the list and unlinking scanned it, under a
      spinlock that never yielded — 100k parked receivers kept all 32 cores at
      100 % for seconds (62.8 s of CPU in 2 s). Waiter lists are O(1) doubly
      linked with a tail; the spinlock yields its time slice after 128 spins.
      Same program: 47 ms of CPU in 2 s (TestManyWaitersOnOneChannel, 1/8
      threads; 20 s timeout before, 1.7 s after)
- [x] A suspending call runs in its caller's task (concurrency review B2/F3, 2026-10-09): the
      callee's frame comes from a per-task arena and is resumed inline; a call that does not wait
      is ~4.4 ns (was ~70 ns: a task, a frame, an argument block, a result cell and a `setjmp`
      each); one that waits parks the task in its own frame and hands it back when it ends.
      `bench/suscall`; TestSuspendingCallsInOneTask (debug/release, 1/2/8 threads, GC pressure).
      Found on the way: a `suspends` method-table slot over a plain method forwarded no arguments
      (worked only while they stayed in their registers)
- [x] Loops are cancellation points; suspending loops yield (D145, plan F4, 2026-10-09): a back edge
      polls one cache-line-aligned word; a cancellation unwinds there (a plain function from where it
      is, the task then waiting as T_ENDING for the children of the scopes it left — a panic too
      now); a scope body notices a failed child; a suspending loop that ran ~10 ms while work waits
      yields; lock regions and `with` closes are shielded; `yieldNow()`, `checkCancelled()`; the
      monitor moves a long-running task's runnext to the shared queue. TestLoopsAreCancellationPoints
      (debug/release, 1/2/4/8 threads, GC pressure); docs reference/concurrency, chapters 12, 14, 16,
      concurrency-explained, stdlib
- [x] Atomics (D144, plan F5, 2026-10-09): `compareAndSet`, `compareExchange`, integer `add`/`sub`/
      `fetchAnd`/`fetchOr`/`fetchXor`, `order: MemoryOrder` on every operation (an invalid order
      written as a case is a compile error), lock-free `Atomic` of a pointer, nullable pointer or
      enum. sema TestAtomicMemoryOrders; driver TestAtomicOperations (a lock-free stack, debug/release,
      1/2/8 threads, GC pressure); docs chapters 12 and 13, reference/concurrency, stdlib, cheatsheet
- [x] Synchronisation types (D146, plan F6, 2026-10-09): `RwLock<T>` (writer-preferring, re-lock
      panics, `with c = l.read()` / `l.write()` under the D107 rules), `Event`, `Lazy<T>`,
      `Broadcast<T>` + `Subscription<T>` (`Lagged`), `Watch<T>`; their waits are awaited and are
      `race` arms. sema TestSyncTypes; driver TestSyncTypes (debug/release, 1/2/8 threads, GC
      pressure); docs chapter 12, reference/concurrency, stdlib, cheatsheet, errors
- [x] A `close()` may suspend (D147, plan F7, 2026-10-10): `fun close() suspends`; a `with` on it waits
      on every way out, shielded; a panic's unwinding hands the close and the rest of the cleanups to
      a closer task; lock regions, non-suspending lambdas and Closeable objects refuse it. std: `Tx`
      rolls back at close, `otel.start` flushes, TLS sends close_notify at close as Go does
      (`net.Conn.tryWrite`). sema TestSuspendingClose; driver TestSuspendingClose (debug/release,
      1/2/8 threads, GC pressure), TestTLSClient (the alert reaches a Go server), the db suite against
      PostgreSQL (the connection is kept); std/otel test; docs chapters 13, 16, 23, 24, references
- [x] Executors and threads (D143, plan F8, 2026-10-10): `[runtime] threads` in the program's
      manifest; `Executor.pool`/`Executor.thread` (static functions that throw `ThreadError`),
      `scope(on:)`/`gather(on:)` (a task stays on its executor), `e.run(f)`, `blocking(f)` (a growing pool
      of at most 128 threads), `Thread.start(…, f:)` joined by its `with` (blocking; a panic is raised
      there); thread names, priority and CPUs on Windows and Linux. sema TestExecutorPlacement,
      TestManifestRuntime; format TestScopeOnRoundTrips; driver TestExecutors (debug/release, 1/2/8
      threads, GC pressure), TestRuntimeThreadsManifest; selfhost parses `scope(on:)`; docs chapters
      11 and 12, concurrency-explained, reference/concurrency, stdlib, cheatsheet, errors
- [x] Deadlock detection: "deadlock: every task is blocked" when no task
      can run and no timer, socket or blocking call can wake one
- [x] Blocking-call detection → superseded: a blocking call hands its
      run queue to a spare thread (below), so it no longer stalls other tasks
- [ ] Under `VELES_THREADS=1` a computation in a function that does not suspend still keeps the
      only thread (D145: it cannot yield), so a `withTimeout` around it fires only when it ends — the
      timer's task has no thread. A plain function cannot be resumed; making it fair needs it
      compiled suspending (or a second thread). Recorded 2026-10-09 (F4)
- [ ] A side effect inside `Atomic.update`'s lambda (found 2026-10-02 in
      chapter 12's Semaphore sample: a counter bumped in the lambda counted
      every retry under contention) is only warned about in the docs. Refusing
      it at compile time needs a rule for what the lambda may do — a decision

### 1.4 Type system and syntax

- [ ] Attributes with typed arguments → D51 (today the derive attributes
      check their own arguments; a general typed form is not designed)
- [x] Coherence/orphan rules for `implement`: none beyond D17 — any implement
      anywhere, one per (trait, type) pair program-wide (§10, derivation batch)
- [x] `Default` (D119) — built 2026-10-02 (plan B15): prelude trait with
      the listed implements (tuples to eight), derived by an empty
      `implement Default`, `T.default()`; conformance `D119-default`,
      `examples/generics`, chapter 8, stdlib reference
- [x] Suspension follows the argument (D116) and `is Trait` at run time and
      `T implements X` at compile time (D117) — built 2026-10-02 (plan B15):
      plain and coroutine instances per call (`TestConditionalSuspension`,
      `D116-conditional-suspension`, hover "suspends if `f` does"); a type id
      in every method table, per-trait tables, narrowing (`D135-is-on-trait-
      objects`, `examples/traitobjects`); `T implements X` decided per
      instance, only the taken branch checked (`D117-implements`). B8 moved
      the eager adapters into the prelude with it
- [x] `x is Display` on a non-trait-object said "'i64' can never be
      'Display'", and `p is Frag` on a trait object that `Frag` implements
      said "can never be 'Frag'" — replaced 2026-10-02 by D117's
      always-true/false warnings (with a fix on `is` expressions) and D135's
      downcast
- [x] A call through a suspending function value evaluated its arguments
      twice (found 2026-10-02 building D116): fixed, pinned in
      `TestConditionalSuspension`
- [x] One `try` over a chain (D134) — built 2026-10-01 (plan B11):
      `tryChain` unwraps every failing link of the receiver chain; `try (try
      f()).g()` warns that the inner `try` is redundant, with a fix; the `(try f()).g()` sites in std, examples and docs that
      end an expression written as one `try`.
      Tests: conformance `D134-try-chain`, sema `TestTryChain`
- [x] Integer overflow policy per build profile (D21: checked in debug,
      wrapping in release, `+%` always) — verified 2026-09-28 (plan A3):
      golden `overflow` is lowered and run in both profiles (`main.ll` /
      `main.release.ll`): `+ - *`, unary `-`, `/` and `%` at MIN/-1, `+=`,
      panic in debug and wrap in release; `+%` and the conversions (`wrapT()`, D86) behave the same in
      both. Found: an inclusive range ending at its type's maximum
      (`loop (i in 250..255)` over u8) looped for ever in both profiles —
      the loop and `RangeIter` now stop by a flag, measured free at -O2;
      `(-128..127).len()` and `.step()`/`.reversed()` over more than half a
      type overflowed or silently yielded one element — wrapping distances
      now; `MIN.abs()` returned MIN silently in debug — now an overflow
      panic like `-MIN`. `pow` panics on overflow in both profiles, as its
      doc says
- [ ] Generic prelude bodies are checked only when instantiated: a call
      to a method the element type lacks (`this.isEmpty()` inside a
      `List<T>` extend) compiles until something uses it. Checking bodies
      against their bounds at definition — which the self-hosted compiler
      will want too
- [x] Compile-time evaluation: constant expressions, constant tables,
      `const fun`, `static assert` (D113, decided 2026-10-01) — plan B14.
      Parts 1–3 built 2026-10-03 (`sema/consteval.go`, `codegen/llvm/consts.go`,
      `static assert`); part 4, `const fun`, built 2026-10-05 (`sema/constfun.go`,
      `constnum.go`; the differential test `driver/constfun_test.go`; chapter 2,
      `examples/constfun`) — what is left is under §11 "Deferred from `const fun`"
- [x] Module-level values are initialized in dependency order (through the
      functions their initializers call); a cycle is an error — built
      2026-10-03. Before, `val a = f()` with `f` reading a later `val` saw
      its zero bits, and a cycle between globals overflowed the compiler's
      stack
- [ ] Better inference for empty collection literals (`val xs = []` typed
      from later use; the typed-context half landed with the old note #8)
- [ ] Stable ABI story for `.vs` packages: none needed while source-only,
      but say so
- [x] D100 `with x = e` as a statement and `with t = async f()` — built
      2026-10-01 (plan B10): `ast.WithStmt` read by the checker as the
      block form over the rest of the block; with-tasks lower to a scope
      cancelled at the end of its body; refusals and the "closed as soon as
      it is opened" warning; hover on `with` and the close-order inlay hint;
      the tree migrated. Tests: conformance `D100-with-statement`,
      `D100-with-syntax`, driver `TestWithStatementExits` (every exit path,
      both profiles), lsp `TestWithInTheEditor`, parser
      `TestWithStatement`, format "with statement", `examples/with`
- [x] D101–D106 — built 2026-10-01 (plan B11): `if (m = …)` one error with
      the `val` fix (D101, conformance `D101-heads`); loop mutation refused at
      compile time and caught at run time by a modification count on lists
      and maps (D102, `D102-loop-changes`, driver
      `TestLoopOverChangedCollection`); a one-task `gather` is its Result and
      `async` runs `sendable fun` values (D103, `D103-gather-and-values`);
      rotate/swap/reverse/copySign/isSignNegative/nextUp/nextDown (D104,
      `examples/bits`, golden `bits`); `reserve` on StringBuilder, maps, sets,
      deques (D105, `examples/reserve`); empty-literal message and fix,
      `Range.isEmpty`, lambda arms parenthesized (D106, `D106-empty-literals`,
      format "lambda arm")
- [x] A generic call's type argument inferred from the expected type
      (`val small: i8 = id(12)`) — built 2026-10-01 (plan B11): a literal
      argument takes the type the expected result binds; conformance
      `D25-type-arguments`

---

## 2. Safety

- [x] Sanitizers (plan A2, 2026-09-28): `veles build/run/test --sanitize`
      builds the C runtime under ASan + UBSan and links their runtimes (the
      collector's stack scan is exempt; use-after-return fake frames are
      turned off by the runtime, since they would hide roots). On Linux
      every example (`go test ./examples -sanitize`, the collector every
      4 KiB, threads 1/2/8) and the driver's runtime and test-runner
      programs (`go test ./driver -sanitize`, ×3) are clean. One finding,
      fixed: the test runner's capture buffer did `NULL + 0` on an empty
      print (UB). Docs 13. Windows: needs MSYS2's compiler-rt (not
      installed here; the build says so). 2026-10-02: every sanitized
      program had come to abort as its first thread ended — the runtime's
      malloc'd alternate signal stack was still installed, and ASan unmaps
      the installed one; the runtime now puts the previous one back
      (`veles_stack_thread_done`, now called by the main thread too).
      TestSanitizedProgramEndsCleanly runs on Linux in every `go test`
- [x] `readRequest` limits: `http.Limits` — request line, header line, header
      count (lines, not map entries), header bytes, body bytes, and three clocks
      (header, body, idle). A byte ceiling answers 414/431/413 and closes, a time
      ceiling 408; nothing reaches a handler, and a body is never assembled to
      discover it was too big
- [x] Every read from the network is bounded by a caller-given max: `readLine(max:)`
      has no default and throws `io.TooLong` (moved from `net` with D128), so the unbounded call does not compile;
      `read(max)` and `readExact(n)` were already caller-bounded, and http checks
      `Content-Length` against `bodyBytes` before the read
- [x] `unsafe` blocks audited: each std use has a comment saying why it is sound.
      The `// SAFETY:` lint (D44 addendum) warns on a block without a reason,
      and sema `TestStdIsWarningFree` holds std to zero warnings. The audit
      found a **real hole**: `net.Conn`/`Listener.close()` closed the socket
      number again on a second call, and systems reuse a freed number at once —
      a second `close()` of one connection closed another (reproduced), and a
      read after close could have read another peer's bytes. The number is now
      a shared atomic cell that `close` swaps out (driver
      `TestSocketClosedTwice`) (2026-09-27)
- [x] Closing a socket while another thread reads it: each `net` call holds
      the socket for its duration (a count, Go's fdMutex idea); `close()`
      marks it closing, wakes every task waiting on it (they fail with an
      `IoError`), and the number is freed by whichever of `close` and the
      last call finishes second, so no call can land on a reused number.
      Before, a read parked on the socket never woke (driver
      `TestSocketCloseDuringRead` hung; 200 rounds at 1 and 8 threads now
      pass on Windows and Linux, and under `--sanitize`) (2026-09-28)
- [x] Bounds checks stay on in every profile; `atUnchecked` /
      `setUnchecked` / `byteAtUnchecked` in `unsafe` (D114) — built
      2026-10-02: refused outside `unsafe` with the checked spelling named,
      a debug panic "unchecked index … out of bounds" at the caller, a plain
      load/store in release (golden `unchecked` in both profiles, driver
      `TestUncheckedAccess`, conform `D114-unchecked-access`)
- [x] Integer conversions between widths: `toT()` is checked (`T?`), `wrapT()` is the explicit
      truncation, a literal that cannot fit is a compile error (D86, 2026-09-29;
      `examples/conversions`, conform `D86-conversions`, golden `arith`)
- [x] Constant-time comparison primitive in `std/crypto`: `Digest`'s `==`, and
      `crypto.equalBytes` for raw bytes. Structural `==` on `List<u8>`
      short-circuits, so a MAC is compared as a `Digest`, never as bytes (D59)
- [x] Secrets: `Secret<T>` (D112) — built 2026-10-02: prelude
      `std/prelude/secret.vs`, `[redacted]` everywhere text is made, not
      Encodable (derive asks for `@skip`), Decodable, constant-time `==`,
      never hashed or ordered, wiped by `close()` and by the sweep (a
      `DESC_WIPE` descriptor bit; runtime probe `veles_gc_test_wipe`); jwt and
      `crypto.hmac…()` keys take it (driver `TestSecret`, conform `D112-*`)
- [x] Path traversal: `path.within(root, p)` over a lexical `path.clean`,
      both separators on every platform, a `\\server\share` root kept.
      Writing it found a **real hole**: `http.files` split the request on
      `/` only, so on Windows `GET /static/..\main.vs` passed the `..` check
      and served a file from outside the root (reproduced: 200 with the
      file's contents). `files` now checks `within` *and* refuses any `..`
      segment on either separator; `examples/httpd` probes `..`, `..\` and
      `%2e%2e` (2026-09-25)
- [x] Fuzzing: `examples/fuzz` (seeded, `fuzz [iterations] [seed]`) checks
      properties, not examples — base64/hex round trips and canonical
      re-encoding, JSON text refused-or-round-trips, generated JSON values
      round-trip, `percentDecode` never panics, `std/utf8` and the runtime's
      `decodeUtf8` agree. Its first minute found three bugs, all fixed with
      the inputs kept as a corpus it checks every run: `toF64` **panicked** on
      `1e99999999999999999999` (the exponent overflowed); `json.parse`
      accepted `1e400` as infinity, which `json.encode` then refused; and
      `toF64` was **not correctly rounded** (digits summed in floating point),
      so a float's shortest text read back as a different float and drifted
      on every JSON round trip — it now takes the value from `strtod`, which
      the printer already checks against (2026-09-25). The HTTP request
      parser is fuzzed over a real connection: `http.serve` in the same
      process on loopback, one to three requests per connection, the last
      with one of 21 faults (each with the status RFC 9112 asks for) or the
      whole write mutated; every answer must be a well-formed response,
      none a 500 or a timeout. Its first run found **response splitting**
      (a path decoding to `\r\n`, echoed into a header, let the client
      write headers into the response — values are now sent with line
      breaks and NUL as spaces, and the server owns `content-length`/
      `transfer-encoding`/`connection`), a line that is not UTF-8 closing
      the connection without an answer (now 400), HTTP/1.0 kept alive by
      default, and a lenient head a proxy could read differently: space
      before a colon, folded lines, non-token names, CR/NUL in values,
      `+5` or two differing `Content-Length`s, a missing or doubled `Host`,
      `HTTP/1.1x` — all 400 now. The ten inputs are a corpus it checks
      every run (2026-09-28)
- [x] Command injection: `os.run(program, args)` joined its arguments into a
      shell command line (`popen`), quoting only those with a space or a
      quote, so `x;echo pwned` ran a second command on POSIX and `%VAR%`,
      `&`, `|` were live on Windows. It now spawns with no shell —
      `posix_spawnp` with an argv, `CreateProcessW` with each argument
      quoted by the MSVC rules — `mergeStderr` is a dup of the pipe, not
      `2>&1`; `.bat`/`.cmd` refused on Windows (BatBadBut), a NUL byte
      refused everywhere. driver `TestRunPassesArgumentsVerbatim` (2026-09-27;
      the POSIX branch compiled and passing on Linux since 2026-09-28)
- [x] NUL bytes at the C boundary: a Veles string may hold `\0`, a C string
      ends there. `fs` refuses a path holding one (`InvalidInput`), since
      `dir/..\0/x` passed a `..` check as one odd segment and then opened
      `dir/..`; `net.listen`/`connect` refuse such a host
      (`evil.example\0.trusted.example` passes an `endsWith` allow-list and
      resolved `evil.example`); `os.env` answers null; `os.run` refuses it.
      driver `TestPathsWithNulRefused` (2026-09-27)
- [x] Resource leaks: a `Closeable` never closed is a warning (D115) —
      built 2026-10-02 (`sema/lint_closeable.go`, fixes `with x =` and
      `with _ =`, `val _ =` discards); no hits over std, examples, bench
      and std's tests (conform `D115-never-closed`, driver
      `TestCheckFixNeverClosed`)
- [x] A value holding a `Task` is received with `with` (D111) — built
      2026-10-02: tasks launch into the receiving block's scope, carried by
      the running task while the value is computed; joined before
      `close()`; a failing later field starts nothing (`sema/taskheld.go`,
      `codegen/llvm/held.go`, driver `TestHeldTasks` on 1/2/4/8 threads,
      conform `D111-task-holding`, lsp `TestHeldInTheEditor`); std/http's
      tests use a `TestServer` value instead of a lending function
- [x] D100: a `with`-bound value (either form) cannot be returned, yielded,
      stored outside its block or captured by a returned/stored lambda —
      built 2026-10-01 (`sema/resources.go`, error family `resources`); the
      check is local (names, `val` aliases, literals and lambdas around the
      value; stores into mutable collections, `Deque`, `PriorityQueue`,
      channels), so a callee that keeps what it is lent is not caught (§11)
- [x] Stack depth: one limit, in the prelude — `maxRecursionDepth` (1000),
      `tooDeepMessage(limit)` for the one sentence every caller reports, and
      `Depth` (`enter`/`leave`/`deepest`) for a walk whose depth is only its
      call frames. Every walk in the library over input someone else wrote is
      bounded by it: the JSON decoder as before, and now the JSON **encoder**,
      `ValueEncoder` and `ValueDecoder` — a tree built in memory nests as deep
      as whoever built it wanted, so refusing a document on the way in while
      walking one on the way out was a hole. `json.Options.maxDepth` (64) is
      JSON's own policy against that default. The router does not recurse (it
      matches segments in a loop) and so has nothing to bound; the front-end
      rewrite takes `Depth` as it stands (`veles-selfhost-frontend-plan.md` §4.3),
      and `examples/recursion` is that parser in miniature — a recursive descent
      that answers a diagnostic rather than a segmentation fault
- [x] Stack exhaustion is a panic and the stack is big (D92, found by plan
      S3, 2026-09-29; built 2026-09-29). Before: a recursion that is not a
      loop died silently at 30 000–50 000 frames on Windows (1 MB, exit 127)
      and 200 000–1 000 000 on Linux (8 MB, SIGSEGV). Now every thread that
      runs Veles code has 256 MB reserved and committed lazily
      (`VELES_STACK=<MB>`, 1–4096, a refused reservation retried at half), the
      program starts on such a thread (`main` is the runtime's,
      `runtime/c/veles_stack.c`; the generated one is `veles_main`), and
      running out prints `panic: stack overflow` with the size and, in a debug
      build, the innermost calls of the D81 chain, exit code 101. Tests:
      driver `TestStackOverflowIsReported` (a million frames fit in main and in
      a task; overflow in both profiles; `VELES_STACK=1`; a bad value), on
      Windows and Linux. Measured: `spawn`, `parallel`, `channels`, `pipes`
      unchanged. Open: a single frame larger than the guard region can jump
      it (clang stack probes), macOS untested (A7)

---

## 3. Performance

### 3.1 Runtime

- [x] Threads (see 1.3) — the single largest throughput multiplier (D66 stage 1)
- [ ] GC: pause-time metric exported; heap ceiling; allocation-rate counter
- [ ] GC: generational or incremental marking once a server-shaped heap is
      measured (mark-sweep stop-the-world today)
- [ ] Escape analysis: short-lived request objects on the stack
- [ ] Coroutine frames: size report per function; pool frames of hot shapes
- [ ] `string` representation: check that slicing and `substring` do not copy
- [~] `StringBuilder` growth policy and a `reserve`: append is one copy into
      doubling storage (2026-09-27); `reserve` decided 2026-09-30 (D105,
      plan B11)
- [ ] `List<u8>` ↔ socket: writev/readv, no intermediate copies
- [x] I/O: `poll` → a reactor (plan E3, 2026-10-08): epoll on Linux, AFD poll
      requests on an I/O completion port on Windows, `poll()` elsewhere
      (`runtime/c/veles_poll.c`); kqueue for macOS comes with A7 (§11).
      `bench/httphello`: 20.6× Go → ~1.2× at 8 threads (Windows)
- [ ] Task handoff on Linux: `spawn` is 3.9× Go on Linux against 1.1× on
      Windows, `parallel` 1.2× against 0.7× (bench/results.md, 2026-09-28) —
      profile the wake-up path (futex condvars under the runtime lock)

### 3.2 Compiler

- [ ] Release profile: `-O2`, LTO, no frame pointers unless profiling
- [ ] Panic-freedom analysis on leaf functions (a safety and a perf win)
- [ ] Devirtualisation of trait objects with one implement in the program
- [ ] Incremental compilation or at least per-module caching of IR
- [x] Compile-time budget: `veles build --timings` (also run/test/check)
      prints each phase — load, check, codegen, runtime objects, clang —
      and the ten modules that cost the most to parse + check (2026-09-28).
      First reading, `examples/httpd`, debug build: 939ms, of which clang
      853ms on 2.7 MB of IR, the front end 27ms — the build is bound by the
      size of the IR handed to clang, so the per-module IR cache above is
      where build time is won
- [ ] IR size: 2.7 MB for `examples/httpd` (32 source files, 266 KB).
      Measured 2026-09-28: no one culprit — 630 std functions make 2.3 MB,
      and single bodies are large (`http.readRequest`, ~100 source lines,
      is 109 KB of IR; `withTimeout` 50 KB over 3 instances): lowering
      writes list adapters, checks and interpolation out inline in every
      body. Plan B8 (the eager adapters as prelude functions) is the first
      lever; measure `--timings`' clang line before and after. B8 done
      2026-10-02: 4.4 MB now (4 607 016 → 4 594 632 bytes at HEAD → after;
      the IR had grown with std since 09-28) — a call instead of an inlined
      loop per adapter, but each element type still gets its own instance.
      Interpolation and checks written out inline remain the bulk

### 3.3 Benchmarks (so regressions are seen)

- [x] `bench/`: seven workloads (sha256, maps, sort, json, strings, GC-heavy
      trees, channels), each next to a Go program doing the same work, the
      checksums compared, and the result read as a multiple of Go —
      `go run ./bench` (2026-09-25). `httphello` added 2026-10-07 (64 keep-alive
      clients × 250 requests against an in-process server, the Go reference the
      same with `net/http`). Still to add: parsing a large file once the
      self-hosted parser exists
- [x] Numbers recorded in-repo: `go run ./bench -record` appends to
      `bench/results.md`; the first run is there
- [x] The allocator was quadratic between collections (every slot of every
      full span scanned per allocation): `json` took 376 s, now 0.31 s. Fixed in
      `veles_gc.c` (O(1) full-span skip, a per-class cursor); `examples/gc`
      guards it (2026-09-25)

---

## 4. Runtime and operations

- [x] Linux x86-64 (WSL2 Ubuntu 24.04, clang 18): `go test ./...` green,
      every example under `VELES_THREADS=1/2/4/8` ×5, a tiny
      `VELES_GC_THRESHOLD` and `VELES_GC_POISON`; a bench run recorded
      (2026-09-28). Found on the way: the runtime needed `_GNU_SOURCE`
      (`pthread_getattr_np`); Windows wrote the standard streams in C text
      mode (`\n` → `\r\n`, stdin ended at a 0x1A byte) — binary now, so
      output is the same bytes everywhere; two doc samples depended on the
      platform (glibc's `cbrt` rounding, a 1 ms race). `internal/wsl-test.sh`
      reruns it from the Windows tree
- [ ] macOS arm64 (plan A7)
- [x] Signals: `os.shutdownSignal(): os.Signal` (D68, 2026-09-26; `os.onSignal` rejected) + `os.raiseSignal` for tests. Real console/POSIX delivery not exercised by the suite (no console in the test harness); the path from the recorded signal on is (`examples/shutdown`)
- [x] Graceful shutdown in `http.serve(..., stop:, grace:)`: stop accepting, close idle keep-alive, drain with `connection: close`, cancel after `grace` (2026-09-26)
- [x] Panic in a task: the call chain with function names (D81, 2026-09-28): debug builds keep a per-task shadow stack; release builds print the location and say a debug build shows the chain. DWARF unwinding for release traces stays possible later (rejected for now, D81)
- [x] Crash report: what a panic prints and where — message, `at file:line:col in fn`, `called from` lines, on standard error, exit 101 (D64, D81; chapter 7, errors reference)
- [ ] Static binaries; cross-compile Windows → Linux (LLVM target triple)
- [ ] Docker base image and a one-line `veles build --release --target linux`
- [x] Environment: `os.env`, `os.hostname()`, `os.pid()`, `os.tempDir()`; the working
      directory is `fs.cwd()` (2026-09-25)
- [ ] File descriptors: `ulimit` awareness, accept-loop behaviour at the limit
      (today: log and sleep 100 ms — keep, but count it)

---

## 5. Standard library

### 5.1 `std/json`

- [x] `Value` tree type in the prelude (`VNull` … `VObject`), `Codable` itself
- [x] Strict RFC 8259 parser with a depth limit; byte position in malformed-document errors (no size limit yet: the caller bounds the body, §2)
- [~] Compact and pretty printers; control characters escaped (U+2028/2029 pass through — fine for JSON, audit for HTML embedding)
- [x] Derived encode/decode
- [x] Numbers: `i64` exact, `f64`; an integer that does not fit is a problem, not rounded
- [~] Streaming: encode builds a `StringBuilder`, decode reads bytes; writing straight to a socket writer is still to do
- [ ] Validation layer: constraints → one 400/422 with a field-path list

### 5.2 `std/http` — server

- [x] Request limits (§2)
- [x] Chunked transfer-encoding: requests (decoded; extensions/trailers handled, framing conflicts 400/501) and
      responses (`Response.stream`), D97, 2026-09-29; `examples/fuzz` generates chunked requests and faults
- [x] Streaming bodies (D97): lazy request body (`req.bytes/text/stream/multipart`, `Expect: 100-continue`,
      unread bodies drained), `Response.stream` (+ `length:`), `http.files` streams from disk,
      `fs.open` → `File`, `MediaType`. Tests: `std/http/body.test.vs`, `stream.test.vs`, `multipart.test.vs`,
      `std/fs/fs.test.vs`; docs 15, 17. Request-body decompression: `http.decompressRequests` (D124, 2026-10-04). Open: a bound on a producer from
      `http.timeout`, typed multipart forms, an SSE helper, zero-copy `sendfile`, `fs.File` off the event loop
- [~] Multipart: streamed parts to disk done (D97); typed `req.form<T>()` with a file field open
- [x] Middleware: `type Middleware = sendable fun(Handler): Handler`; `router.wrap(m)`
      (`use` is the import keyword, so the verb is `wrap`) — outermost first, around
      the router's own 404s and 405s too
- [~] Standard middleware: `requestId()`, `logging()`, `timeout(d: Duration)`, `cors(...)`, `guard`,
      `basicAuth`, `bearer` done (D99, 2026-09-30: `std/http/cors.vs`, `auth.vs`, tests `cors.test.vs`,
      `auth.test.vs`; docs 17 "Other origins, and who may call"); recovery needs nothing (a panic
      is already caught at the request boundary, D56) and body-limit is `Limits.bodyBytes`. Still
      open: per-route guards (route groups). Compression: `http.compress()` (D124, 2026-10-04)
- [x] Graceful shutdown (§4, D68)
- [x] Max concurrent connections with backpressure at `accept` (D99, 2026-09-30: `Limits.connections`, default 10000, 0 = none; a permit channel taken before `accept`, so a full server stops accepting; `std/http/limits.test.vs`; docs 17)
- [x] `Expect: 100-continue` (D97: sent on the handler's first body read)
- [x] `HEAD`/`OPTIONS` defaults; `405` with `Allow` (2026-09-26: HEAD runs the GET route and the writer drops the body, keeping its `content-length`; OPTIONS answers 204 + `Allow`; a 1xx/204/304 is written without body or length)
- [x] Cookies (D94, 2026-09-29): `req.cookie(name)`/`cookies()`; `resp.withCookie(Cookie(...))`
      and `withoutCookie` — `Response.cookies` written as one `Set-Cookie` line each;
      safe defaults (`Path=/`, `HttpOnly`, `SameSite=Lax`, `secure` opt-in), value
      percent-encoded, a bad name/path/domain, `SameSite=None` without `secure`, a
      broken `__Host-`/`__Secure-` name a panic at the caller's line. Tests:
      `std/http/cookie.test.vs` (run by driver `TestStdHttpUnitTests`),
      `examples/session` over a socket. Open: signed/encrypted session cookies
- [~] Forms (D94, 2026-09-29): `x-www-form-urlencoded` done — `req.form<T>()`,
      `req.query<T>()` through the codec layer (`http.FormDecoder`, format `form`;
      400 with every problem at its field, 415 for another type), `formFields`,
      `formValue(s)`, `queryFields` (`http.Fields` keeps repeats), `Request.rawQuery`;
      `std/http/form.test.vs`, `examples/session`. Multipart: streamed to disk
      since D97 (typed `req.form<T>()` with a file field open)
- [x] Static files (D96, 2026-09-29): `fs.stat`; `http.files` sends `last-modified` + weak `etag`, answers
      `If-None-Match`/`If-Modified-Since` 304 and `If-Match`/`If-Unmodified-Since` 412 from the stat alone, one
      `Range` (206/416, `If-Range`), `cache-control` (`no-cache`, or `maxAge:`/`immutable:`), `index:` list,
      308 to the slash (`redirect: false` off), dotfiles 404 (`dotfiles: true` on), GET/HEAD only.
      `std/http/files.test.vs`; docs 15, 17. Streams from disk since D97. Precompressed `name.gz` siblings since D124 (2026-10-04; `.br` needs a Brotli codec, not offered). Open: directory listing (not offered)
- [~] Router: method-not-allowed vs not-found distinction (done: 405 + `Allow`), route groups,
      typed path params (`{id: i64}`)
- [x] In-process test client: `http.call(handler, method, target, body:, headers:)` (2026-09-26; same target parsing and panic boundary as `serve`; docs 17 "Testing a handler")
- [~] Access log format: one line with peer, method, path, status, latency (a `Duration`, so the unit is in the line) and the
      request id (`logging()`); not structured (JSON/key=value) and no byte count yet
- [ ] HTTP/2 (later; needed for gRPC-style internal traffic)
- [ ] WebSocket (later)

### 5.3 `std/http` — client

- [x] `http.get/post/...` and a `Client` with pooling, timeouts, redirects —
      `http.fetch` + the Go spellings (D127), built 2026-10-04 (plan C7):
      keep-alive pool with lazy expiry, one deadline over the whole request
      and its body, redirects for GET/HEAD with credentials dropped across
      hosts, bounded bodies, strict response parsing; the body is a
      `Payload` (D127 addendum)
- [x] TLS (5.4) — `https://` URLs (2026-10-05, E1); `http.Client(tlsOptions:)` for a private authority
- [x] Retries with backoff and idempotency awareness (`retry:`; idempotent
      methods only; 502/503/504 and connection failures)
- [ ] Proxy environment variables

### 5.4 `std/tls`

- [x] Client TLS via a system binding (SChannel on Windows, OpenSSL elsewhere, loaded at run time) — 2026-10-05, E1
- [x] Server TLS with certificate reload (`tls.listen`, `Certificate`, `tls.reloading`) — 2026-10-05, E1
- [x] Certificate verification on by default; opt-out is loud
- [x] `http.serve(tls: cert)` (2026-10-05; the user chose a `tls:` parameter over an `io.Listener` trait)

### 5.5 `std/crypto`, `std/hex`, `std/base64`, `std/jwt` → D59

Four modules, not two: `hex` and `base64` are encodings and do not live
behind a name that says "crypto" (§10, 2026-09-23).

- [x] base64 (std + url) and hex, both strict on the way in: the other
      alphabet, whitespace, a misplaced `=`, an impossible length and a
      **non-canonical** last character are refused with the byte offset
      (`base64.Invalid`/`hex.Invalid`)
- [x] SHA-256/384/512 and SHA-1 (`sha1Legacy`), one-shot and streaming
      (`Hasher`: `start`/`update`/`finish`, finished once); HMAC written once
      over `H: Hasher` (`Hmac<Sha256>`); constant-time compare is what
      `Digest`'s `==` *is*, plus `equalBytes` for raw bytes
- [x] CSPRNG: `crypto.randomBytes(n)`/`randomU64()` over `BCryptGenRandom` /
      `getrandom(2)` / `arc4random_buf` (`veles_random_bytes` in `veles_os.c`,
      `-lbcrypt`); panics rather than throws, and `std/random`'s doc now
      points here
- [x] UUID v4 and v7; v7 carries a counter in `rand_a`, so successive ids are
      **strictly increasing** (5000 checked in `examples/crypto`)
- [x] JWT sign/verify (HS256/HS384/HS512) with the algorithm taken from the
      caller, `exp` required by default, `crit` rejected, a key shorter than
      the digest refused, and `Reason` so `Expired` can be told from the rest;
      checked against the RFC 7515 A.1 token
- [x] Password hashing: `crypto.hashPassword` / `verifyPassword`, Argon2id in PHC form, OWASP parameters, ~60 ms at -O2 (2026-10-05, E2) —
      written in Veles instead of the libargon2 binding (no native library to install); checked against the three
      vectors of RFC 9106 and a hash made by argon2-cffi
- [ ] RS256/ES256 — needs bignum or a binding (blocked on 1.2)
- [ ] A `Hasher` that is not a SHA: BLAKE3 or SHA-3 when something asks
- [x] Zeroing: std's HMAC/JWT keys are `Secret<List<u8>>` (D112, 2026-10-02);
      `Hmac.start` keeps raw bytes as the low level, and its pads are
      ordinary memory (§11)
- [ ] Not benchmarked; the compression functions allocate nothing per block
      but are plain Veles, so they are far from a hand-tuned C hash (§3.3)

### 5.6 `std/time` → D60

- [x] `Duration` type: a prelude struct over an `i64` of nanoseconds
      (±292 years), `Duration.seconds(5)` and siblings, `Comparable`,
      `Display` (`0s`, `250ms`, `1.5s`, `2m30s`, `1d1h`) and `Parsable`
      that reads back exactly what `Display` writes — the fraction is
      carried in whole nanoseconds, never a float, so
      `Duration.parse("$d") == d` for every `d`. Arithmetic is methods
      (`plus`/`minus`/`times`/`dividedBy`), named so an `Arithmetic` trait
      could later adopt them without a second spelling
- [x] RFC 3339 parse and format, in one parser with three named leniencies
      (lower-case `t`/`z`, a space separator, ISO 8601 expanded years so the
      round trip reads back every instant but the last second at each end of
      the range); the leap second `23:59:60` reads as the last
      microsecond of its minute. HTTP-date moved out of `std/http`:
      `formatHttp` writes IMF-fixdate, `parseHttp` reads all three forms
      RFC 9110 §5.6.7 requires a recipient to accept
- [x] Zone offsets: `Offset` is minutes east of UTC (±18:00), `DateTime`
      carries one, `Offset.local(at:)` asks the host for its offset at a
      given instant. The calendar is pure Veles (Hinnant's
      `days_from_civil` and its inverse); C answers only "what time is it"
      and "what is this host's offset". tz database still later
- [x] Monotonic deadlines: `Deadline.after(d)` / `remaining()` / `expired()`
      / `extend` / `earlier`, and `Stopwatch.elapsed(): Duration`. No raw
      monotonic reading is handed out any more, so it cannot be compared
      with a wall-clock one; `withTimeout` and `sleep` take a `Duration`
      (the `sleep` builtin unwraps it in `sema/check_task.go`, rounding
      **up** to the executor's millisecond)
- [x] `Timestamp`: microseconds since the epoch, so RFC 3339's six digits
      and a PostgreSQL `timestamptz` round-trip; `Display`/`Parsable`/
      `Codable` are all RFC 3339 text
- [x] `Duration` is `Codable`: `"90.5s"` by default, and `DurationStyle` (`Seconds`,
      `Iso8601`, `Text`, `Nanos`, `Millis`) as a format option —
      `json.Options(durations:)` — the `EnumStyle` precedent (D60 addendum,
      docs 18 "Durations", `examples/codable`) (2026-09-25)

### 5.7 `std/log`

- [x] Levels, structured fields, JSON or key=value output (2026-09-29, D91: text on a terminal,
      JSON otherwise; `VELES_LOG`; `examples`/docs chapter 21)
- [x] Request-scoped logger (task-local, 1.3): `log.withFields`; `http.requestId()` binds `id`
- [~] No interpolation cost when the level is off: the message is `lazy` (D90) and is not built;
      the call still makes a closure (~30 ns a disabled `log.debug` at -O2, measured 2026-09-29;
      the guarded form is ~0.3 ns) and `field(...)` arguments are evaluated — hoisting the level
      check before the closure (inlining a lazy callee, or escape analysis) would close it

### 5.8 `std/db`

- [x] PostgreSQL over the wire protocol, written in Veles (2026-10-05, E2): the extended query protocol, SCRAM-SHA-256,
      TLS through `std/tls`; no libpq. Tested against PostgreSQL 16 on Windows and Linux (`driver/db_test.go`)
- [x] Connection pool with health checks (a task checks idle connections; idle ones expire)
- [x] Parameterized queries only; no string concatenation path (`sql"…"`, `Sql` has no constructor from a `string` but `dangerouslyRaw`)
- [x] Transactions as a `with` resource (a block that ends without `commit` abandons the transaction, see §11)
- [x] Row → struct mapping through the derive (`@key(db:)`, snake_case columns, NULL, arrays, bytea, timestamps, uuid)
- [x] Statement and connection timeouts
- [x] A span per statement (D126), with the text and never the values
- [ ] Migrations helper (later)

### 5.9 Observability

- [x] `/healthz`, `/readyz` helpers — `http.Health` (D133), built 2026-10-04:
      checks run at once under their timeouts, details logged not returned,
      `serve(health:)` flips readiness at the stop, probes kept out of the
      request log; the server template uses it
- [ ] Metrics, traces and logs as OpenTelemetry over OTLP/protobuf, with
      W3C `traceparent` in and out — decided 2026-10-01 (D126), not built
      (plan C4); a Prometheus pull exporter not decided
- [ ] Runtime metrics: GC pauses, heap, tasks, open connections

### 5.10 Misc

- [x] Float bits without `unsafe` (D93, found by plan S3, built
      2026-09-29): `x.toBits()` on `f64`/`f32` gives the IEEE 754 pattern as
      `u64`/`u32`, `f64.fromBits(bits)` / `f32.fromBits(bits)` invert it;
      nothing is rounded, a NaN keeps its payload, `-0.0` differs from `0.0`.
      In `std/prelude/number.vs` over the same `unsafe` cast (LLVM folds it to
      a `bitcast` at -O2). `examples/floatbits` (LLVM-style hex constants,
      round trips, NaN payload, `-0.0`, sign/exponent/fraction), docs chapter 2
      and the stdlib reference. A compiler builtin can replace it later
- [x] `std/config` (D125, 2026-10-04): `config.load<T>(files:, prefix:)` reads a derived struct from
      the environment, dotenv and JSON files beneath it, then field defaults; every problem at once
      by variable name with its source, a `Secret` never echoed; `config.describe<T>()` lists the
      variables. Built on `Decodable.schema()`, which the derive writes beside `decode` (enum members,
      per-format `@key`, defaults as text) and `KeyStyle.UpperSnake`. Tests `std/config/*.test.vs`
      (29), `sema/derive_print_test.go`, conformance D04; example `examples/config`; docs chapter 22,
      stdlib reference; `veles new --template server` reads `HOST`/`PORT` through it. A module may now
      declare its own `Error` (the `error` sugar names the prelude's trait). Not offered: a `Map` or a
      list of structs from variables (a panic naming the field), `${VAR}` interpolation, a way to set
      the environment from a program (tests pass a lookup to the private `loadWith`)
- [x] `std/compress` (D124, 2026-10-04): gzip/deflate/inflate written in Veles, 64 MiB default
      ceiling on the output (`TooLarge`), `GzipWriter`/`GzipReader` over `io.Stream`, `GzipEncoder`,
      `http.compress()`, `http.decompressRequests()`, `.gz` siblings in `http.files`. Tests:
      `std/compress/compress.test.vs`, `std/http/compress.test.vs`; Go-made vectors, every level,
      a decoder fed one byte at a time, the ceiling; benchmarked against Go's `compress/gzip`
      (`bench/gzip`, `bench/gunzip`: 1.1× and 1.5× on text, up to 2–3× on short-match prose);
      docs 17 and the reference. Open: `deflate` at levels 7–9 is slow on long inputs
      (zlib's own limits would help), `inflate` could write straight into a sized buffer
- [x] `io.Stream` (D128, 2026-10-04): `net.Conn`, `fs.File`; `http` over it; `tls.Conn` joins with E1
- [x] endian bytes and the `fs` additions (D130, 2026-10-04): `x.toBeBytes()`/`toLeBytes()`, `T.fromBeBytes`/`fromLeBytes` on
      every integer type, `readU16Be` … `readI64Le` on `List<u8>` and `pushU16Be` … on `MutableList<u8>`
      (`std/prelude/endian.vs`, generated once); `fs.writeAtomic`, `fs.copy`, `fs.lines(path, max:)`
      (Result items, Closeable), `file.seek`, `file.sync`, `file.lock`/`tryLock` (`std/fs`, runtime
      `veles_os.c`). Tests `std/prelude/endian.test.vs` (9), `std/fs/fs.test.vs`; example
      `examples/binfile`; docs chapter 15, stdlib reference. Windows: a lock is one byte far past the
      end (so it stays advisory), an appended file writes at the end by offset, a file open elsewhere cannot
      be replaced by `writeAtomic`. Not offered: byte-range locks, shared (read) locks
- [x] template literals (D129 part 1, 2026-10-04): `tag"text ${x}"` calls a `@template fun tag(parts: List<string>, values: List<V>)`;
      `ast.TemplateExpr`, parser (adjacent only, a space is an error that says so), checker lowering to the call
      (`sema/template.go`), `@template` validated at the declaration, formatter, `Dump`, VS Code grammar. Tests:
      parser expression cases + `TestTemplateLiteralNeedsNoSpace`, `TestTemplateLiteralRoundTrips`,
      conformance D129-template-signature / D129-template-use, `TestTemplateTagQualifiedByModule`; example
      `examples/templates`; docs chapter 2 and 14, cheat sheet. `std/db` (E2) is the first user
- [~] `std/os`: `hostname`, `pid`, `tempDir` done (2026-09-25); `shutdownSignal`/`raiseSignal` done (D68); `run` without a shell (2026-09-27, §2); `run(..., input:, stderr: os.Stderr)` with `Output.stderr` (D82, 2026-09-28)
- [~] `std/fs`: `walk` done (2026-09-25: depth-first, name order, links to
      directories not followed, its own stack); streaming reads/writes, atomic
      rename, file locks open
- [x] `std/utf8`: one code point at a time — `decode`/`decodeLast` over a
      `string`, `decodeBytes` over a buffer that carries no promise,
      `encodeTo`/`encode`/`char` back, `combineSurrogates` for the formats
      that quote characters the UTF-16 way, `isValid`/`count` over a buffer.
      Strict in (shortest form only, no surrogates, nothing above U+10FFFF —
      Table 3-7) and lossy out (U+FFFD for a value that is not a code point,
      as Go and Rust do). `std/json` now decodes its `\u` escapes through it
      instead of a private copy; `examples/utf8` is the Table 3-7 and Kuhn
      stress vectors
- [x] Collections: `Deque<T>` (ring buffer) and `PriorityQueue<T>` (binary
      heap) in the prelude; `Set` has `union`/`intersect`/`difference`/`isSubsetOf`
- [x] Binary search in the prelude: `binarySearch`, `binarySearchWith`,
      `binarySearchBy`, `lowerBound`, `upperBound` and the `partitionPoint`
      they are all written in terms of. -1 for absent, as `indexOf` answers,
      and always the *first* of several equal elements

---

## 6. Tooling and developer experience

- [x] Bug (found 2026-10-02): `veles test <dir>` also ran the embedded
      std test files (`*.test.vs`) of every std module the code imports —
      seen with `fs` and `ticker`. Fixed 2026-10-02: test mode drops the test
      files of std and dependency modules other than the one given
      (`veles test std/time` still runs its own); driver
      `TestTestSkipsImportedStdTests`
- [x] Bug (found 2026-10-02): a mismatched comparison in `expect`
      (`expect(1 == "1")`) was reported at `<builtin>`, with no file or
      line: the names standing for the captured sides had no span. Fixed
      2026-10-02; they carry the side's span (conformance
      `D78-test-vocabulary`)

- [x] The needless-`throws` warning (D45) alone printed `see: veles explain
      type-mismatch` (found 2026-09-30 by the spec review): reworded
      ("declares 'throws', but …") so `results-and-errors` claims it; pinned
      in `source/TestFamilyOfRealMessages` (2026-10-01, plan B11)
- [x] `veles fmt` stable on every example: `format/TestCorpus` now fails on any
      file under `examples/` or `std/` that `veles fmt` would change (2026-09-25).
      (docs blocks are not held to it: they align comments by hand)
      A 2026-09-24 review found it swept a function body's comments into a
      parameter list the author had wrapped: fixed, with two cases in
      `format/format_test.go`. That it took reading a formatted file to
      notice is the argument for the CI check
- [x] D140: expression bodies are `fun f() => expr`; `= expr` errors with a
      fix; the tree, the docs and the Veles inside Go tests migrated (2026-10-08)
- [x] Imports one per line by default, `[format] imports = "merged"` for the
      comma list; the tree and the docs' programs reformatted (2026-10-08)
- [x] `[lint]` in the manifest: lints of a package's own code;
      `implicit_return = "full" | "lambda" | "expr"` (where a `{ }` body may
      end in its value; otherwise an error with a fix that writes `return`)
      (2026-10-08)
- [~] `veles test`: `--filter` (no match is an error), per-test `--timeout`
      (default 10m; a watchdog thread reports the test and ends the run),
      a summary line naming the failures — done 2026-09-27. Tests are
      `test "sentence" { }` with `expect`/`require`/`expectThrows`/
      `expectPanics`/`fail` (expression capture, both sides, location; soft
      failures reported together), `test fun` helpers, `*.test.vs` files,
      `assert(cond, "why")` anywhere (reason required); `@test fun` errors with a fix (D78) — done
      2026-09-27. A test's output is captured and shown only under its failure (2026-09-27). Tests run at once (D80, 2026-09-28): each its own record (failures, output) bound in its task's locals and inherited by what it starts, up to `--jobs` running (default: one per runtime thread), reports in declaration order, a watchdog per running test; a helper launched with `async` and a panic inside a helper both name the test's calling line. Open: coverage
- [ ] `veles bench`
- [~] LSP: find references, document highlights and rename done
      (2026-09-27): across modules, into interpolations and named arguments,
      a trait method with its implementations; the rename re-checks the
      package and refuses if any name would resolve differently or an error
      appears, spells out D28 puns, and refuses wire names of a derived
      `Codable` (fields, variants). Inlay hints for inferred binding types,
      expression-body return types, `suspends` and bare-`throws` sets done
      the same day. Quick fix "add `implement Encodable`/`Decodable`/`Comparable`"
      (derived) on every "does not implement" and "cannot order by" error,
      explained through generic impls (`List<Tag>` → `Tag`) — done
      2026-09-27. "Mark 'f' @skip" on a coding derive stopped by a field
      with no wire form (a function, a channel, a Mutex), when the field
      has the default a skipped field needs — done 2026-09-27. Signature help and workspace symbols done
      (works mid-typing and inside `${}` interpolations). Go to type
      definition (through `?`, `*`, lists and maps to the declared type),
      go to implementation (a trait → its implements, a sealed trait → its
      variants, a trait method → its implementations) and folding ranges
      (bodies, argument lists, comment runs, `use` runs; token-based, so
      they work while the file does not parse) — done 2026-09-27. "Add the missing methods" (stubs that panic until written) on an implement lacking trait methods and on a sealed variant with no implement — a guess: offered, not preferred, never applied by `--fix` (2026-09-28)
- [x] Conformance suite (plan A4, 2026-09-28): `sema/testdata/conform/`,
      30 case files by D-number, each line's `// error:`/`// warning:`
      checked strictly both ways; the checker's every diagnostic format is
      read from its source, and a full `go test ./sema` fails for one no
      test provokes unless `uncovered.txt` lists it with the reason no
      program reaches it (parser-guarded, std-internal, defensive, …) —
      a listed entry without a reason fails too. 267 → 46, every one with
      its reason; the two-module rules are covered by sema package tests
- [x] Diagnostics: every error names the fix, with a `docs/` link. Done
      2026-09-28: "did you mean" on every unknown name, function, method,
      field and module member (edit distance with transpositions, case,
      long prefixes, and other languages' names — `size`→`len`,
      `append`→`push`, `toUpperCase`→`toUpper`, `has`→`containsKey` — only
      when the type has the target); a field called as a method and a method
      read as a field are told which they are; the guess is an editor quick
      fix (not preferred) that `check --fix` never applies; a local named by
      a typo is not "never used". The docs link, D79 (2026-09-28): every
      diagnostic belongs to one of 34 named families (`source/family.go`,
      matched on the message); the CLI prints `see: veles explain
      <family>` once per family, `veles explain <family>` prints its
      section of `reference/errors.md` (embedded; offline, version-matched,
      "did you mean" on a wrong name), the LSP sends `code` +
      `codeDescription`; sema `TestEveryDiagnosticHasAFamily` (all 612
      formats of sema/parser/lexer, filled with sample arguments) and docs
      `TestEveryFamilyIsExplained` hold both ends. The audit (plan B1):
      every message names what to write, unless its fix is simply undoing
      the one fact it states (`duplicate field 'x'`, `'None' takes no
      arguments`) or it is a compiler-internal check
- [x] `veles doc`: rendered API docs from `///` — Markdown per reachable
      module (root + what it re-exports, D89), declarations as hover shows them from
      outside, member docs; `-o dir` for files (2026-09-27)
- [x] Packages (D138, D139; plan E7, built 2026-10-05..07): `veles.toml` with path, git and registry
      dependencies, one instance per (source, major), minimal version selection,
      `veles.sum` (byte-exact hashes, verified on every use), module cache and `vendor/`,
      `veles add/update/remove/deps/vendor/fetch/audit/publish/yank/attest`, workspaces,
      capabilities computed from source with a `[policy]`, the registry protocol with
      tiers, yanks and signed reviews, a reference server
- [ ] `package.vs` (D131): the manifest as a typed constant — decided, unbuilt; `package`
      is a reserved word, so D131's `const package = …` does not parse (see §11)
- [x] ~~Lockfile~~ — none, by decision (D132): MVS + `veles.sum` give
      reproducible builds
- [x] `veles new <dir>`: a package that runs and tests first try (manifest,
      `main.vs` with a test, `.gitignore`); `veles build` in a package names
      the binary after it — done 2026-09-27. `--template server`: an HTTP
      service with `/healthz`, request logging, a request id and a time
      limit per request, HOST/PORT from the environment, a graceful stop on
      Ctrl+C/SIGTERM, and in-memory tests of its handlers; driver
      `TestNewServer` builds it, tests it, starts it on PORT=0 and asks
      `/healthz` (2026-09-28)

---

## 7. Documentation

- [x] `docs/18-codable-and-json.md` (chapter, attributes table, stdlib reference)
- [x] `docs/19-secrets-and-crypto.md` (hashes, MACs, randomness, UUIDs, the two
      encodings, JWT and what each refuses)
- [x] `docs/20-time.md` (why there are three types, `Duration`, `Timestamp`,
      the calendar and offsets, RFC 3339, HTTP dates, `Stopwatch`/`Deadline`,
      and where a `Duration` turns up); the stdlib reference and the cheat
      sheet moved with it
- [ ] A "deploying Veles" page: binaries, signals, Docker, health checks
- [ ] A production-servers chapter (was pencilled in as chapter 20; the
      number is taken, so it is 21)
- [x] Derivation chapter with the rules from 1.1 in plain language
- [~] Security guidance: request limits are in chapter 17 and constant-time
      comparison, canonical decoding and the JWT refusals in chapter 19; a page
      that collects them, with secrets and TLS defaults, is still to write

---

## 8. Examples (each with `expected.txt`)

- [~] `examples/httpd` upgraded: `std/json` (done: derived `Note`, a `PUT` with a 400 that lists the problems); middleware, graceful shutdown,
      limits — the reference server
- [x] `examples/codable`: every derivation rule in one program
- [x] `examples/crypto`: every published test vector — FIPS 180-4, RFC 4231,
      RFC 2202, RFC 4648 and the RFC 7515 A.1 token — plus the decoder
      refusals and 5000 monotonic v7 ids
- [x] `examples/time`: the RFC 3339 grammar and its leniencies, the leap
      second, the calendar across 1970 and the year 2000, the three HTTP-date
      forms and the refusals, and the round trips for `Duration` and
      `Timestamp`. Only the last section reads the host clock, and only for
      what holds of every reading of one
- [ ] `examples/apiclient`: calls a JSON API over TLS
- [ ] `examples/pgnotes`: the notes API on PostgreSQL (needs a server: an example whose expected output the suite can compare must start one; see §11)

---

## 8b. Self-hosting the front end

The lexer and parser rewritten in Veles, planned in
`veles-selfhost-frontend-plan.md`: six phases, each gated on byte-identical
output against the Go front end over the repository's own `.vs` files, ending
with the Veles parser parsing itself. Four decisions to make before it starts
(a rune API, `unicode.IsPrint`, the recursion limit, `Span`'s representation)
are listed in §4 of that file.

- [x] P0 `source` — `selfhost/source/`, gate P0 over the corpus (2026-10-08)
- [x] P1 the lexer — `selfhost/lexer/`, gate G1 (tokens, docs, comments,
      diagnostics) over the corpus and hand-written broken files (2026-10-08)
- [x] P2/P3 the AST and its printer — `selfhost/ast/`, Go's `%q` checked on
      every corpus line (2026-10-08)
- [x] P4/P5 the parser — `selfhost/parser/`, gates G2 (the tree) and G3 (every
      diagnostic) over the corpus and broken files for each recovery path; the
      Veles front end parses its own sources and agrees (G4) (2026-10-08)
- [x] The self-hosted front end reads the current language only: removed
      spellings are syntax errors there, files using one are outside the gates
      and `TestRemovedSpellings` covers them; the port rewritten as idiomatic
      Veles (self-host plan §11) (2026-10-08)
- [ ] P6 `Layout` (comments and parenthesised spans, for the formatter) and a
      dump that prints every span (`ast.Dump` prints none, so G2 does not check
      spans), files that are not valid UTF-8 in the corpus (§11)

---

## 9. Decisions pending

Only what is still open. Each is asked through `veles-decide` (options,
examples, edge cases, pros/cons, a recommendation); the answer becomes a
spec entry and a row in §10, and the item leaves this list. The labels are
stable; §10 rows written before 2026-09-27 cite the old numbering (the
list as it was is in `archive/progress-log-2026-09.md` and git history).

(Q15 named imports was decided 2026-09-29: D85; Q16 `as` conversions the same day: D86; Q19 the same; Q4 D87, Q8 D88, Q3 D89 — decided 2026-09-29, being built in that order.)

**Open:**

- **Q29 — non-Sendable state on an `Executor.thread`** (D143 left it to be
  decided once executors were built; they are, 2026-10-10). Every task
  placed on one executor of one thread runs on the same OS thread, so such
  tasks could share a `MutableList` without a lock, as Swift's
  `@MainActor` code does — if the compiler can tell that a value never
  leaves that executor. Today they must share through a `Mutex` like any
  tasks.

(2026-10-09: Q24–Q28, raised by the concurrency review, were decided the same day: D143–D147; Q22 and Q23 on 2026-10-08 into D142; Q21, packages, 2026-10-05: D138.)

(Q12, Q13, Q17 and Q20 were decided 2026-09-30: D101–D106. Q1 was decided 2026-10-01: D108; Q2, Q9, Q10, Q11 the same day: D111–D114; Q5, Q6 the same day: D117, D118; Q7 the same day: D120–D123; Q14 the same day: D131, D132.)

Every new public std API (http cookies/forms/client, `std/log`,
`std/config`, metrics, ...) is its own decision when its turn comes in
`veles-plan.md`; they are not pre-listed here.

## 10. Decision log

| Date | Decision | Result |
|---|---|---|
| 2026-09-21 | Derivation mechanism | **Explicit opt-in by empty `implement`**: `implement Codable for User` synthesizes the body; hand-written methods are kept, the rest synthesized; foreign types allowed under the orphan rule. Principle: *properties of a value* (`==`, hash, `Sendable`) are implicit; *contracts with the outside world* (serialization) are declared. Rejected: implicit-from-shape (silent renames, leaked fields), visibility-gated implicit (couples API visibility to wire shape), `@derive` attribute (a second spelling, no partial override), user-definable derivation (phase 2). |
| 2026-09-21 | Derivation target | **Generic `Encodable`/`Decodable`** over `Encoder`/`Decoder` traits (Swift Codable / serde): one derive feeds JSON, DB rows, config; encoding streams to bytes, no tree. The `Json` tree type is itself `Codable`. Rejected: JSON-specific `toJson()/fromJson()` (a derive per format, a tree allocation per encode). |
| 2026-09-21 | Field attributes | **`@key` with per-format overrides**: `@key("user_id")` for every format, `@key(json: "userId", db: "user_id")` per format by string name (the `Encoder`/`Decoder` report `format()`), `@skip` / `@skip(json)`. Casing policy is an encoder *option* (`json.Options(keys: KeyStyle.SnakeCase)`), never an attribute. Errors: two fields on one key; `@skip` on a field without a default. Rejected: one shared key only (cannot express camelCase JSON + snake_case column); per-format attributes `@json`/`@db` (each format needs compiler support; library formats impossible). |
| 2026-09-21 | Sealed discriminator | **Internally tagged**: `{"type": "Circle", "r": 1.0}`; `implement Codable for Shape { }` on the sealed trait covers every variant; `@tag("kind")` on the trait renames the key, `@tag("type", content: "value")` opts into the adjacent layout; a variant's tag value is its declared name, renamed by `@key`. A field named like the tag is a compile error. Formats without variants (rows) refuse sealed types with a runtime `DecodeError` naming the path (the encoder is a runtime value, so the derive cannot know the format statically). Rejected: externally tagged (rare in APIs), untagged (fragile, slow, bad errors). |
| 2026-09-21 | Nullable / missing | **Missing `T?` → `null`** (Swift/serde). Required = no default and not nullable; a field default is the value when the key is missing (no `@default` attribute); `@required` marks a nullable field whose key must be present; absent-vs-null when it matters: `T??` or `json.Patch<T>`. Encoding writes `null` by default; `json.Options(omitNulls: true)` drops them. Rejected: Kotlin's strict rule (every optional field needs `= null`). |
| 2026-09-21 | Enums (part 1) | **Automatic, by name** — no `implement` needed (enums are compiler-owned, D57); `"Active"` on the wire, exact-match decode, `@key("active")` renames a member, unknown name → error listing the valid names. Rejected: by number (unreadable, renumbering breaks clients); a declared `implement` (relaxes D57, a line per enum). *How a number representation is requested: see part 2.* |
| 2026-09-21 | Enums (part 2) | **Representation is the format's policy, not the type's**: `json.Options(enums: EnumStyle.Number)`; `json` defaults to `Name`, `db` to `Number`. No attribute. Decoding is strict per style (a number under `Name` is an error). A per-type exception is written by hand through partial override. Rejected: `@key(number)` on the enum (second meaning for `@key`, bare-identifier argument); a new `@codable(...)` attribute (a new word for a rare case). |
| 2026-09-21 | Encoder/Decoder surface | **Flat event stream**: `Encoder { format, beginObject, key, endObject, beginList, endList, i64/u64/f64/bool/string/bytes/null }`, `Decoder { format, beginObject, nextKey, endObject, beginList, hasNext, endList, primitives, isNull, skip, path }`; values encode themselves (`self.id.encode(to)`), so no generics in the trait and no boxing. `Encodable { encode(to: Encoder) throws EncodeError }`, `Decodable { static decode(from: Decoder): Self throws DecodeError }`; `Codable` = both. Fixed error types (D40 object-safety). **The encoder is a trait object**, not a generic: object-safe `Encodable`, simpler derive; devirtualise later. Sealed decode: fast path when the tag is the first key, else buffer through the `Codable` `Json` tree. Prelude implements primitives, `List`, `Map<string, _>`, `T?`, tuples by hand. Rejected: Swift-style containers (generic `field<T>` impossible on a trait object; the non-generic form boxes every value); a neutral `Value` tree (allocation per field); generic `<E: Encoder>` (loses object safety, more code per format). |
| 2026-09-21 | Decode errors | **Collect all problems**: `DecodeError { problems: List<Problem> }`, `Problem { path, message }` (later `expected`/`found`), capped (`maxProblems`); the derive decodes each field into a temporary, records instead of throwing, `skip()`s what it cannot decode, throws once at the end; nested structs prefix the path. A malformed document is one problem. Path spelling `user.address[2].zip`, `Problem.pointer()` gives RFC 6901; dotted keys quoted. Validation (§5.1) reuses `Problem`. Rejected: fail-fast (one field per round trip); fail-fast types + collected validation (split responses). |
| 2026-09-21 | Derived impls on generics | **Bounds inferred for an empty implement**: `implement Codable` inside `struct Page<T>` is read as `implement<T: Codable> Codable for Page<T>`; only parameters used by non-skipped fields get the bound; hover shows the header. **Spelling**: an empty implement may omit the braces — `implement Codable` in a struct body (and `implement Codable for geo.Point` at top level); `implement Codable { }` stays legal, the formatter drops empty braces. Rejected: requiring the explicit top-level form (ceremony, first error a newcomer hits); bounding every parameter (wrong for phantom/`@skip`-only parameters). |
| 2026-09-21 | First derivable set | **`Codable` (`Encodable`, `Decodable`) and `Comparable`** (lexicographic by field order, reusing the tuple comparison). `==`, hashing and printing are already structural. `Default` deferred (field defaults cover it). Traits and error types live in the prelude; the JSON encoder/decoder in `std/json`. Spec entry: D58. |
| 2026-09-21 | `Codable` and supertraits | **Supertraits implemented**: `trait Codable : Encodable + Decodable { }` (supers spelled like a bound) is an ordinary prelude trait; a bound `T: Codable` expands transitively; `implement Codable for X { ... }` needs the supers implemented, and an *empty* implement derives every super implement the type lacks. Bootstrap limit: a trait with supertraits cannot yet be a trait object (error names the super to use). Rejected: compiler-known sugar (a name that is not a trait); `implement A + B for X` (new syntax, two names per DTO). |
| 2026-09-21 | Function type parameters | **After the name**: `fun encode<T: Encodable>(value: T)`, as on a struct and at the call site. The Kotlin form `fun <T> encode` is an error with the fix; 17 sites rewritten. Rejected: keeping both (two spellings). |
| 2026-09-21 | The keyword | **`implement`** replaces `implement` everywhere — lexer, parser (the old word still parses, with an error naming the new one), formatter, grammar, LSP, spec, docs, std, examples. Rejected: keeping `implement` (the Rust abbreviation the notes dislike); folding trait impls into `extend … with …` (a change to D23, not a rename). |
| 2026-09-22 | Bounding a network read (std) | **`readLine` takes a required `max`.** The peer decides where the newline goes, so a line is only as long as the reader allows; a default would make the unbounded call the easy one and the ceiling invisible. Reaching it throws `net.TooLong`, distinct from `IoError` because the caller has to answer differently (431/414, not 500) — which costs every caller a wider `throws`, and that is D45 doing its job. `readExact(n)` keeps its shape: `n` is the ceiling, and http checks `Content-Length` against `Limits.bodyBytes` before calling. Rejected: a generous default (the unbounded intent stops being expressible, and safety by forgetfulness); a budget on the `Conn` (invisible at the call site). |
| 2026-09-22 | Hover on a derived implement (tooling) | **The summary in the hover, the code behind a command**: hovering the trait name shows the implements produced, the bounds inferred for a generic target, the synthesized signatures and the wire shape (keys, optional, skipped) — the question a derive actually raises; `veles explain <path> --derive [Type]` prints the bodies as Veles. Needed a printer for synthesized nodes (`sema/derive_print.go`): the formatter is source-guided (it copies literals verbatim and reads the author's line breaks out of the text) and every derived node carries the implement's span, so it could not be reused. Rejected: the full bodies in the hover (~80 lines for a Codable pair); the s-expression debug dump (syntax no Veles programmer writes). |
| 2026-09-22 | `internal` in hover (tooling) | **The unwritten visibility level has no word.** It is the common case, and `internal` on every line made hovers harder to read; `public`, `private`, `protected`, `static`, `val` are still spelled out. `visPrefix` in `sema/index.go` is the one place that decides. (A half-done version of this shipped in 5d050f5 and left a stray leading space and 8 red LSP tests.) |
| 2026-09-21 | `is Trait` check | **Deferred** to its own decision (§9 item 14): a runtime check needs RTTI in trait objects, a compile-time one belongs to generics, and for a concrete type the answer is static. |
| 2026-09-23 | Crypto module layout | **Four modules**: `crypto` (digests, HMAC, constant-time compare, CSPRNG, UUID), `hex`, `base64`, `jwt`. base64 is an encoding, not encryption, and a name that implies otherwise is how `base64.encode(password)` gets written; `encoding.Base64.encodeUrl(...)` would also be three hops for a web server's commonest call. Rejected: one `encoding` module with the codecs as statics; everything under `crypto`. |
| 2026-09-23 | What a hash returns | **A `Digest`**, not `List<u8>`: `Display` prints lower-case hex, `Equatable` compares in **constant time**, so `mac == expected` is both the natural and the safe spelling; `bytes()` is for *sending* a digest and compares with the ordinary short-circuiting `==`; `Digest.of(bytes)` wraps a signature from outside so it can be compared safely, and `equalBytes` covers raw bytes. Rejected: `List<u8>` (a timing oracle that compiles silently, defended only by a doc comment). |
| 2026-09-23 | base64 surface | **Four functions** — `encode`/`decode` (standard, padded) and `encodeUrl`/`decodeUrl` (RFC 4648 §5, unpadded). Two axes with two useful combinations do not earn an options struct, and `encodeUrl` says at the call site what it does. Both decoders take padded or unpadded input; everything else is refused with the byte offset, including a **non-canonical** last character (two texts, one signature). Rejected: an `Options(alphabet:, pad:)` struct; Go-style codec values (`base64.Std.encode`). |
| 2026-09-23 | `randomBytes` on failure | **Panics, does not throw.** A kernel that will not produce randomness is not a condition a caller can answer — there is no weaker source and nothing to decide — and `throws IoError` would spread to `uuidV4()` and every session-token line. A panic is still caught at a request boundary (D56). The one place in std where an external failure is a panic, because it is unrecoverable rather than rare; Go 1.24 made the same move. Rejected: `throws IoError`; a silent fall back to `std/random`. |
| 2026-09-23 | UUID v7 ordering | **Strictly increasing**: the twelve `rand_a` bits hold a counter (RFC 9562 §6.2), started at a random point in the lower half of its range each millisecond, borrowing the next millisecond on overflow and ignoring a clock that goes backwards. Monotonicity is the whole reason to prefer v7 over v4 — it appends to an index instead of scattering writes — and milliseconds alone do not give it. Cost: module-level mutable state, a lock once the executor has threads (D35). Rejected: 74 random bits (unordered within a millisecond, which the example caught). |
| 2026-09-23 | JWT scope and strictness | **HMAC only, strict by default**: the expected algorithm comes from `Options`, never from the token's `alg` (that is `alg: none` and RS256-as-HS256); `exp` required unless waived; `crit` rejected; the signature compared through `Digest`; a key shorter than the digest refused with a panic (RFC 7518 §3.2). `Invalid` carries a `Reason` enum because `Expired` means "refresh" and the rest mean "sign in again". RS256/ES256 wait for a bignum or a binding. |
| 2026-09-23 | Static call on a qualified generic (compiler) | **`mod.Type<T>.f(...)` parses.** The speculative type-argument path only allowed a bare name before, so `crypto.Hmac<Sha256>.start(key)` — the natural spelling from outside the module — was a syntax error. `ast.MemberExpr.TypeArgs` carries them, `moduleTypeNamed` resolves the instance, and a qualified generic named without arguments gets the same error as an unqualified one. |
| 2026-09-23 | Formatter: a grid stays a grid | **A broken list keeps the author's grouping.** `brokenList` put one element per line, which turned SHA-256's 64 round constants into 64 lines. A newline now goes only where the author had one, so a table written as a grid stays a grid — the same "read the line breaks out of the text" rule the rest of the printer follows. |
| 2026-09-24 | Time: three types, not one number | **`Duration` (prelude, i64 nanoseconds), `Timestamp` (wall clock, i64 microseconds) and `Deadline`/`Stopwatch` (monotonic).** No raw monotonic reading is handed out, so it cannot be compared with a wall-clock one — the confusion is unrepresentable rather than documented. Microseconds for the wall clock because RFC 3339's six digits and a PostgreSQL `timestamptz` must survive a round trip; `toSeconds`/`toMillis` round **down** on a point and truncate toward zero on a length. Arithmetic is methods (`a.plus(b)`), accepted rather than paid for with an `Arithmetic` trait, which stays its own decision; `plus`/`minus`/`times`/`dividedBy` are named so it could adopt them. Rejected: bare `i64` with named constructors (the unit stays optional); Go's `time.Time` carrying both readings; keeping `sleep` on `i64`; migrating std later (two spellings). Spec entry: D60. |
| 2026-09-24 | What RFC 3339 accepts | **One parser, three named leniencies** — lower-case `t`/`z` (the RFC's NOTE), a space separator (§5.6, PostgreSQL) and ISO 8601 expanded years, so `parse(t.toString())` reads back every instant but the last second at each end of the microsecond range (the arithmetic to buy those two back is not checkable by eye). A year beyond the ±292,277 a `Timestamp` holds is *refused*, in every parser, because it is a number the sender chose and the multiplication overflows. `23:59:60` is accepted and read as the last microsecond of the minute: it is legal RFC 3339 with no POSIX instant, and a conforming producer's timestamp failing to parse (Go's answer) is worse than the microsecond. A fraction longer than six digits truncates, never rounds, so the order of two texts is the order of their instants. |
| 2026-09-24 | Where HTTP-date lives | **`std/time`, not `std/http`.** `Last-Modified` and `If-Modified-Since` are two ends of one conversation; `formatHttp` writes IMF-fixdate only, `parseHttp` also reads RFC 850 and asctime, which RFC 9110 §5.6.7 requires of a recipient and which is exactly what an old client sends. `http.httpDate` is a one-line name for the same function. |
| 2026-09-24 | A date from outside is refused, not overflowed (review) | **One checked conversion.** A review of D60 found that `+999999-01-01T00:00:00Z` and `Sun, 06 Nov 99999999999999 08:49:37 GMT` both reached `days * 86400 * 1000000` and panicked — a parser fed untrusted text may not. Every field-to-instant conversion now goes through `instantOf`, which bounds the year before `daysFromCivil` can overflow and the seconds before the multiplication can. Formatting was made total the same way: `Timestamp.at` shifts the day and the microsecond within it instead of the microsecond count, and `floorMod` is `((a % b) + b) % b` — both of the old forms overflowed within a day of the ends of the i64 range. Left as a documented gap: the last second at each end prints but does not parse back. |
| 2026-09-24 | A local offset the host cannot give (review) | **Ask again for a date it can.** `Offset.local(at:)` reads the zone through the C library, and the Microsoft CRT refuses a negative `time_t` and anything past the year 3000; a date of birth silently read back as `Z`, correct only in the UK. The C side now reports "cannot answer" distinguishably from "UTC" (zero being a real offset), and `std/time` retries with the same month, day and time of day in a year the host can convert, keeping leap-year parity, so the daylight-saving half is right too. Rejected: falling back to UTC (wrong for almost every zone), and carrying a tz database (§5.6, later). |
| 2026-09-24 | Formatter: a broken parameter list is not a comment sink | **`flushComments` stopped at the end of the *function*.** So when an author wrapped a signature over two lines, every comment in the body was swept up into the parameter list and the statements they documented were left bare — no text lost, all meaning lost, and idempotent in the wrong state. The limit is now the parameter list's own `)`, found by a scan that skips the only things that can stand there (whitespace, a trailing comma, comments). Found by reviewing D60's own source after `veles fmt` rearranged it. |
| 2026-09-25 | `trySend` on a closed channel (std) | **Panics, as `send` does.** Sending into a closed channel is a bug in the program, not a state to poll for, and one rule for both forms is easier to hold. `trySend` returns `false` only for "full"; `tryRecv` returns `null` for "nothing buffered", which covers both empty-for-now and closed-and-drained — `len()` or a waiting `recv` tells them apart. |
| 2026-09-25 | A sleeper woken early (runtime) | **It goes back to sleep.** `veles_task_sleep` returned 0 on an early wake without re-blocking the task, and `fire_timers` only wakes a blocked task, so a task sleeping when its scope's child finished was lost and the executor reported a deadlock — any `scope` whose body sleeps while a producer ends. And `sleep(Duration.zero)` returned at once instead of yielding, as the compiler, the cheat sheet and the stdlib page all said it would, so a polling loop starved the task it polled for. Both fixed in `veles_task.c`; `codegen/llvm/golden/tasks` runs both shapes. |
| 2026-09-25 | What `fs.walk` returns (std) | **Every file, as a list, links to directories not followed.** Eager like `listDir` (one call, a stable order, sortable, `mapConcurrent`-able), files only (what every caller in the tree wanted), iterative (a deep tree cannot overflow the stack), and a symbolic link or junction to a directory is neither listed nor entered, so a cycle cannot loop the walk — `examples/dedup`'s own recursive walker would have. `root` itself may be a link. Rejected: a callback walker (no caller needs to prune yet); following links (cycles). |
| 2026-09-25 | `os.hostname` can fail (std) | **`throws IoError`**, like Go's `os.Hostname`: rare, but a host can refuse, and `std/os` reports failures as `IoError`. `pid()` and `tempDir()` cannot fail and do not throw; `tempDir()` has no trailing separator so `path.join` reads right. |
| 2026-09-27 | Task-local values (§9.13) | **Scoped, immutable bindings** (user, recommended of three): `val id = taskLocal(fallback)`; `id.withValue(v, f)` binds for `f` and every task started inside, which keep it; `id.get()`; `T: Sendable`. Spec D72. Rejected: a mutable ThreadLocal-style `set`/`get` (hidden mutation, values that outlive their step); an explicit context parameter (every signature carries it). |
| 2026-09-25 | A `Duration` on the wire (§9.12) | **`"90.5s"` by default, any of five on request** (user: "90.5s, but we need to be able to convert to different"). `DurationStyle { Seconds, Iso8601, Text, Nanos, Millis }` is the format's policy, like `EnumStyle`. Spec D60 addendum. Rejected: ISO 8601 by default (Go/Python stdlib do not read it), seconds as a number (precision). |
| 2026-09-25 | Identifiers and invisible characters (notes #21) | **UAX #31** (user's choice over the recommended fixed list): identifiers are `XID_Start XID_Continue*` plus `_`, so a bidi control, a zero-width character or a no-break space is no longer an identifier byte — the Trojan-source case (CVE-2021-42574) becomes a diagnostic. Costs Unicode tables in the lexer, and in the self-hosted one. |
| 2026-09-25 | Changing a by-value struct parameter (R20 follow-up 1) | **A warning**: a function that assigns a `var` field of a value-struct parameter, directly or through a method that writes `self`, is told the caller never sees it, with `*T` or returning the value as the fixes. R20's rule (`val`/`var` govern rebinding only) is unchanged. Rejected: Swift's immutable parameters (changes R20). |
| 2026-09-25 | Guard binding (I1) | **Let-else plus `??` on Result** (user, after `archive/veles-guard-design.md`): `val x = r else { e => ... }` binds or leaves (Result, nullable, variant pattern; one-statement `else return` allowed); `r ?? fallback` / `r ?? { e => ... }` is `?:` for a Result, and each operator on the other kind is an error with a fix that swaps it. Spec D61. Rejected: widening `?:` to Result (the recommendation), a `Fallible` trait (its own decision, later). |
| 2026-09-26 | Fewer panics: prove the index (user: "atOrPanic is kind of a cheatcode") | **All four parts, in order A → B → C, D alongside.** A: list patterns in `when` and let-else (`val [h, p, s] = xs else ...`, `[x, ..rest]`). B: bounds facts — `xs.at(i)` is `T` where the checker knows the index is in range (index loops, `i < len` guards, constant indexes after a length check; a `MutableList`'s facts end at any call that could reach it). C: `atOrPanic`/`getOrPanic`/`refOrPanic` removed; the only way left is `?: panic("why")` (user's choice over `.expect("why")` and keeping the method with a lint). D: `indices()`, `splitOnce`, `enumerate`/`zip` on List. Spec D62. Built: **A** (2026-09-26: parser, checker, `list.slice` runtime op for `..rest`, examples/listpatterns, docs 04, std/jwt + std/http request line migrated). **B** (2026-09-26: sema/bounds.go — facts in the smart-cast map; `indices()`; `?:` after a proven read is a warning with a fix; ~20 std sites now plain `at`, the rest are index arithmetic or `self` with calls; found D63 on the way). **C** (2026-09-26: the three names are errors whose fix writes `(x.at(i) ?: panic("TODO: say why this cannot fail"))`; std, examples, bench, docs and tests migrated by hand — list patterns, iteration and `?: return` where they fit, a stated invariant elsewhere (one helper per data structure); codegen now inlines list `len` and the bounds-checked element read: sha256 301→72 ms, json 314→135 ms, sort 295→108 ms). **D** (2026-09-26): `indices()`, `s.splitOnce(sep): (string, string)?`, eager `List.enumerate(): List<(i64, T)>` (`zip` was already on List); splitOnce replaced the hand-split header, target, query, ISO-duration and HTTP-date code in std. |
| 2026-09-26 | A `catch` block for panics | **Not added; D52 stands** (user: "we use gather"). Panics are the bug channel, not exceptions: they stay unrecoverable inside a task and are observed only where a task ends (`gather`). Presented: function-level `fun f() { … } catch { p => … }` (recommended), a catch after any block, or none. Should it be revisited, the panic is bound with the D61 handler form `catch { p => … }`, not `this`, since `this` would hide a method's receiver. |
| 2026-09-26 | `self` becomes `this` (user) | **Built 2026-09-26.** The receiver is spelled `this` across the language, std, docs and tooling (spec D65, v0.40). `self` still parses as the receiver with an error whose fix writes `this` (`veles check --fix`, LSP quick fix, inside interpolations too); 1,520 sites migrated by a token-level rewrite (code only, never comments or string text), prose by hand; `Self` the type is unchanged; the VS Code grammar highlights `this`, flags `self`, and no longer marks `enum`/`for` illegal. |
| 2026-09-26 | A MutableList where a List is expected (found by D62) | **Moved when unescaped, otherwise `.toList()`** (user, recommended of four): the checker passed the same handle, contradicting D35 (a List could shrink during a call, or be shared with a task while written). Now a fresh local that never escaped converts at its last use with no copy; anything else is an error with a `.toList()`/`.toMap()`/`.toSet()` fix. Rejected: implicit copy, always explicit, keeping the view. Spec D63. Built 2026-09-26 (sema/move.go; `==` and compiler-written code compare without converting; std: 4 `.toList()` copies, the sha leftover buffer and crypto padding). |
| 2026-09-26 | A Kotlin-style `x!!` to panic on null (user: "controversial, so I'm not sure") | **Not added; every panic shows its location instead** (user, recommended of four): `!!` would be D62's cheatcode at two characters, for every `T?`, with no reason recorded. What it offered was the line, so panics now print `at file:line:col` (relative to the package root) under the message — `panic(...)`, overflow, division, `pow`, non-exhaustive match, list index; kept through `scope`'s re-raise, `Panic.location` at `gather`, printed by `veles test`. Rejected: `!!` everywhere, `!!` with a warning + fix, `!!` only in tests. Spec D64. Built 2026-09-26 (runtime `veles_panic_at`, `veles_list_index_panic`; codegen `g.where`). Known gap: a std panic for caller misuse (`swap`) reports the std line. |
| 2026-09-26 | Executor threading model (§9.9) | **Work-stealing M:N over one shared heap** (user, recommended of three): tasks may resume on any worker; GC stops workers at safepoints (allocation, suspension); per-worker allocation buffers; real `Mutex`/`Atomic`; thread-safe channels/timers/poller. Staged: global queue first, stealing second, each measured in `bench/`. A module-level `var` that is not `Mutex`/`Atomic` becomes an error. Spec D66. Rejected: thread-per-core (imbalance, still needs safepoints), single thread + offload pool (breaks D35). *Both stages built 2026-09-27.* |
| 2026-09-26 | FFI shape (§9.8) | **Hand-written `extern` blocks in any package + a `[native]` table in `veles.toml`** (`libs`, `pkg-config`, Windows lib paths), `extern struct`, C function-pointer types for callbacks; "maybe" a `veles bindgen` later that writes the same form (user). User asked that every extern be unsafe — it already is: a call needs `unsafe`, and an extern cannot be taken as a value. Spec D67. Rejected: `@cImport`-style header import. *Not built.* |
| 2026-09-26 | Signals and graceful shutdown | **A signal is awaited** (user, recommended of three): `os.shutdownSignal(): os.Signal` suspends until SIGINT/SIGTERM (Ctrl+C/Break/close on Windows), installs its handlers only when first asked, a second signal has the default effect; `http.serve(..., stop:, grace:)` stops accepting, drains, cancels after `grace`. Spec D68. Rejected: `os.onSignal` callbacks, signals handled inside `serve` by default. |
| 2026-09-26 | A std panic caused by the caller (D64's gap) | **Left as is** (user, over the recommended `@callerLocation` attribute): the std line is reported until stack traces exist. |
| 2026-09-26 | FFI: strings and buffers | **Explicit copies plus a scoped borrow** (user, recommended of three): `std/ffi` — `CString.of(s)` (Closeable), `ffi.string(p)`, `ffi.bytes(p, n)`, `alloc`/`free` — and `xs.withRaw(p => ...)` lending a `List<u8>`'s storage for a closure. Spec D69. Rejected: explicit only (a copy per big buffer), implicit `string`→`char*` bridging. Built 2026-09-26: `std/ffi` (`CString.of` refuses NUL bytes with `ffi.NulByte`, `readString`/`readBytes`, `alloc`/`free`, `handle`/`Handle<T>.from`), prelude `CLayout` + `withRaw`, raw-pointer casts in `unsafe`, runtime `veles_ffi.c`; examples/ffi. |
| 2026-09-26 | FFI: callbacks | **`extern "C" fun name(...) { body }`** (user, recommended of three): C calling convention, `&name` is an `extern fun(...)` pointer, no suspend/throw, a panic ends the process. Spec D69. Rejected: `@cabi` attribute, no callbacks. Built 2026-09-26 (parser `parseExportedFun`, `types.Func.C`, codegen `exportWrapper`; C can also call it by symbol — TestNativeLinking). |
| 2026-09-26 | `?.` through a chain (R8) | **Swift's rule** (user, recommended): a null after `?.` skips the rest of the postfix chain; `a?.b.c()` is `R?`. Spec D70. Rejected: Kotlin's one-step rule. Built 2026-09-26 (`safeBelow`/`safeChain` in sema/check_safe.go; `Grouped` on MemberExpr/CallExpr marks parentheses; a nullable place is used in place, so a mutating call lands). |
| 2026-09-26 | Arithmetic operator traits (§9.11) | **The full set** (user, recommended of three): `+ - * /` and unary `-` on user types call `plus`/`minus`/`times`/`dividedBy`/`negate` through prelude traits with a right-hand and an output type; `+=` follows; `Duration`/`Timestamp` adopt them. Spec D71. Rejected: `+`/`-` only; methods only. Built 2026-09-26 (sema/operators.go; associated types inferred from the written method, a general rule; `Duration`/`Timestamp` adopted; `negated()` renamed `negate()`). |
| 2026-09-27 | Constructors that do work (user note: a factory "should not be the required way") | **`init` takes parameters** (user, recommended of three; spec D73): `init(value: T) { ... }` parameters are constructor parameters — `Mutex(value: 0)`, `Atomic(value: 0)`, `TaskLocal(fallback: "-")`, `PriorityQueue<Job>(compare: c)`; the natural-order queue is `PriorityQueue<T>.natural()`. Before it, and needing no decision: M5 v0.30's private-field rule now holds across modules, so `StringBuilder()`, `Deque<T>()`, `http.Router()`, `Depth(limit: n)`, `json.JsonEncoder()` are constructors and their factories are removed (errors with fixes). Rejected: a `static fun of` convention; keeping the factories. |
| 2026-09-27 | HTTP statuses, methods, headers (user note #5) | **Open value types with named constants** (user, recommended of three; spec D74): `http.Status.notFound`, `http.Status(code: 418)`, `http.Method.post`, `http.Header.cacheControl`. Rejected: closed enums (unknown codes unrepresentable); integer/string constants only. |
| 2026-09-27 | What is global (user note #4) | **A slim prelude** (user, recommended of three; spec D75): codec machinery → `use codec`, `CLayout` → `use ffi`, `Depth`/`maxRecursionDepth`/`tooDeepMessage` → `use recursion`; the core types, traits, collections, `StringBuilder`, `Duration`, sync types and `Codable` stay global. Rejected: Go-minimal; as is. |
| 2026-09-27 | Which I/O failure (user note #5) | **`IoError.kind: IoKind`** (user, recommended of four; spec D76): a prelude enum the runtime maps each platform code to; `code` stays. Rejected: separate error types, predicates, as is. |
| 2026-09-27 | `DateTime.weekday()` (user note #5) | **`Weekday` enum, ISO Monday = 1** (user, recommended of three; spec D77); `month` stays `i64`. Rejected: a `Month` enum too; as is. |
| 2026-09-27 | Testing (§9 item 15) | **`test "sentence" { }` + a test-only vocabulary** (user, the recommended combination of `archive/veles-testing-design.md`; spec D78): `expect`/`require`/`expectThrows`/`expectPanics`/`fail` with expression capture, `test fun` helpers, `*.test.vs` files, `check(cond)` for invariants — amended the same day to `assert(cond, "why")` with the reason required (user: "should be called 'assert' ... require string explanation like panic"); `@test fun` errors with a fix. Rejected: `test fun` as the test form, an importable `testing` module, a global `assert` as the *test* vocabulary (it stops at the first failure). |
| 2026-09-27 | Test suites (D78 amendment) | **`+bt+`suite "name" { }`+bt+` blocks and `+bt+`*.test.vs`+bt+` files as suites; the report grouped and indented** (user: "I think both is the answer"; report: grouped, over the recommended qualified-name lines). Suites hold tests, suites and `+bt+`test fun`+bt+` helpers scoped to the suite; names are qualified `+bt+`a / b / test`+bt+` for the summary and `+bt+`--filter`+bt+`. Setup/teardown: the lexical proposal rejected ("the setup for before and after isn't good... no new keyword for them either"); open as §9 item 16. |
| 2026-09-28 | How a diagnostic links its docs (plan B1) | **Named families** (user, recommended of 4; spec D79): every diagnostic belongs to a family with a readable name that is its anchor in `reference/errors.md`; the CLI prints `see: veles explain <family>` once per family after the errors, `veles explain` prints the section from the copy compiled into the binary (offline, version-matched), the LSP sends the family as `code` and the GitHub anchor as `codeDescription`; a test makes every format belong to a family. Rejected: a GitHub URL per error (network, follows `main`), numbered codes `error[V0105]`, no link. |
| 2026-09-28 | Parallel tests (plan B3) | **Parallel by default** (user, recommended of 4; spec D80): tests run as tasks across threads, `--jobs N` bounds them, `--jobs 1` is sequential; the report stays in declaration order, each test's lines together. Safe because D66/D35 already rule out data races between tests. Rejected: sequential by default with `--jobs` to opt in, a `@serial` attribute, leaving it sequential. |
| 2026-09-28 | Panic call chains (plan B4) | **A shadow stack in debug builds** (user, recommended of 3, after DWARF and the shadow stack were explained; spec D81): each call records callee and call site on the task's own stack; a panic prints it under `at`; release prints the panic's line and a note. Rejected: DWARF unwinding (three binary formats, a coroutine walker, debug info in release, no wasm — possible later for release), a shadow stack in release too. |
| 2026-09-28 | `os.run` input and stderr (plan B5) | **`input:` + `stderr: os.Stderr` (Capture/Inherit/Merge, default Capture); `Output.stderr`; `mergeStderr` removed with a fix** (user, recommended of 3; spec D82). Rejected: two more booleans, a `Command` builder. |
| 2026-09-28 | List capacity | **`xs.reserve(n)`** — room for at least n in total (user, recommended of 4; spec D83). Rejected: `withCapacity(n)`, both, neither. |
| 2026-09-28 | Q18: two bounds declaring one associated type | **Refuse `T.Item` and name it `T.Trait.Item`** (user, recommended of 3; spec D84). Rejected: refusing with no qualified form, keeping the first bound. |
| 2026-09-29 | Q15: named imports | **`use m { f, T as U }`, one statement, `as` renames, `m.` stays** (user, recommended of 4, and braces add names on top of the qualifier; spec D85). Rejected: `from m import { … }`, `::`/`:` renames, the dotted `use m.{ }`. Q16 (`as` for conversions) stays open.
| 2026-09-29 | Q16: numeric conversions | **Methods split by risk, `as` only renames** (user, recommended of 4; spec D86): `toT()` total where lossless else `T?`, `wrapT()` explicit truncation, `p.cast<*raw U>()` in `unsafe`. Naming: `toU8()` + `wrapU8()` (user, recommended of 3). Rejected: keep `as`, `as` lossless-only, call style `i64(x)`. |
| 2026-09-29 | Q19: converting to a type parameter | **`x.wrapTo<T>()`** (user, recommended of 4; spec D86 addendum). Rejected: an `Integer` trait with `T.wrap(from:)`, keeping the std-only `as` exemption, rewriting `Range.reversed()`. |
| 2026-09-29 | Q4: protected mutable-collection field | **`protected` = look, don't take** (user, recommended of 3; spec D87). Rejected: leave it, read-only view types. |
| 2026-09-29 | Q8: where a misuse panic points | **`@caller_location`, std only for now** (user, recommended of 3; spec D88). Rejected: leave it, automatic for every std panic. |
| 2026-09-29 | The handler's spelling (D98, revised the same day) | **`catch (e) { ... }` — one spelling wherever an error is bound: a tight postfix on a `Result`, on `try chain`, and on a `do` block; `?? { e => }`, `else { e => }` and `catch { e => }` removed outright; no patterns yet** (user, "probably C" of 4; "we should make `catch (e) {` like we have `when (v) {`"; "we do not need to keep old syntax with fix... because it is really not used by anyone else yet"; tight postfix of 2; `try chain catch` of 3 — "`(parse(s) catch (e) { 0 }).len()` is not looking good"; none-now of 2). Rejected: keeping `{ e => }`, changing only the `do` block, typed clauses now, a per-call postfix marker for Results (`?.` is nullable chaining, `!.` echoes the rejected `!!`), parentheses only. |
| 2026-09-29 | Handling several `try`s in one place (raised while testing D97) | **`do { ... } catch (e) { ... }`: an expression, `try` kept inside, a failed `try` or a `throw` in the block goes to the handler, `e` is the union of the block's errors, the block is not a lambda** (user, recommended of 4 and 2; then the spelling revisited — "other languages have 'success story' in a try, and catch block is where the error happens" — and changed to `do { } catch { e => }`, the recommended of 4, "because it reads good, understandable"; spec D98). The 2026-09-26 rejection of a *panic* `catch` (D52) stands. Rejected: `catch { } else { e => }` (chosen first, withdrawn), `attempt { } else { e => }`, `try { } catch`, `on { e => }`, no `try` inside the block, a function-level `catch` suffix, leaving the closure idiom. |
| 2026-09-29 | Body model (plan C2, third task) | **A lazy request body (`req.bytes/text/stream/multipart`, `body` field removed), `Response.stream` with `fs.File` and a streaming `http.files`, streamed multipart, and a `MediaType` value type** (user, recommended of 3, 3, 3 and 3; on the last, the user's note: "shouldn't things like `".html", ".htm" => "text/html; charset=utf-8"` sit in the 'values' file in the http?"; spec D97). Rejected: buffered bodies with chunked support only; a per-route stream mode; `Response.stream` without files, or none; buffered or deferred multipart; moving `contentTypeOf` unchanged, leaving it in `files.vs`. |
| 2026-09-29 | Static files (plan C2, second task) | **`fs.stat` + weak ETag/Last-Modified with 304/412 from the stat, one byte range (206/416), `no-cache` by default with `maxAge:`/`immutable:` parameters, redirect to the slash + dotfiles 404 + `index:` list, all switchable** (user, recommended of 4, 3, 3 and 3; on directories "the code must be configurable like web frameworks, so probably no slash pathing should be also an option" — hence `redirect: false`; spec D96). Rejected: content-hash strong ETag; Last-Modified only; `multipart/byteranges`; no ranges; a general `CacheControl` value now; no cache header; `index:` only; leaving directories and dotfiles as they were. |
| 2026-09-29 | Q3: package surface in source | **`public use m` and `public use m { a, T as U }`; manifest `exports` goes away** (user, recommended of 3; spec D89). Rejected: keep manifest exports, whole modules only. |
| 2026-09-29 | Binding a nullable in a condition (found writing `setCookieLine`) | **`if (val x = e && ...)`, bindings chaining with `&&`, and no `loop` form** (user: "we need something like `?.let` in Kotlin ... execute some code without letting nullable"; recommended A of 3, chains of 2; `loop` "No"; spec D95). Rejected: `x?.let(v => ...)` (a lambda: no `return`/`break`/`continue` out of it); nothing new; a single binding per condition; `loop (val x = e)`. |
| 2026-09-29 | std/http cookies and forms (plan C2, first task) | **Order: cookies+forms, static-file caching, body model, limits/middleware; `Response.cookies` list with `withCookie`; safe defaults, percent-encoded values, footguns panic at the caller; typed `form<T>()`/`query<T>()` plus `formValue(s)`** (user, all four recommended; spec D94). Rejected: the body model first; multi-map headers now (breaks every `headers:` literal — decided with the body model); Go-style raw cookies with everything off; strict cookies with no encoding; untyped-only and untyped multi-valued forms. |
| 2026-09-30 | Connection limit, CORS, auth hook, compression source (plan C2) | **`Limits.connections` with backpressure at `accept`; `http.cors` with an explicit origin list and no predicate; `guard` + `basicAuth` + `bearer` (the brief wrongly said std lacked a constant-time compare; `crypto.equalBytes` is it, nothing added); `std/compress` written in Veles** (user, recommended of 3, 3, 4 and 4; spec D99). Rejected: accept-then-503; a CORS predicate; guard-only or helpers-only; miniz in the runtime (fallback if the speed is poor); system zlib. |
| 2026-09-29 | Float bit patterns (found by plan S3) | **`x.toBits()` / `f64.fromBits(bits)` (and `f32` with `u32`), in std over `unsafe`, only those** (user, recommended of 3, 2 and "only `toBits`/`fromBits`, but note the rest to be made with another decision later" — the rest is Q20; spec D93). Rejected: instance methods both ways in the D86 style (float-only methods on every integer); leaving it to `unsafe`; a compiler builtin for now. |
| 2026-09-29 | Stack overflow (found by plan S3) | **A fault handler that prints a named panic, 256 MB reserved stacks on every thread, and the program started on a big-stack thread on every platform** (user, recommended of 3, 3 and 3; spec D92). Rejected: compiler-inserted stack probes (cost on every call); leaving it silent; 64 MB; the OS defaults with a diagnostic only; raising `RLIMIT_STACK` at startup; leaving the Linux main thread at 8 MB. |
| 2026-09-29 | std/log design (plan C1) | **`lazy` modifier on a `fun(): T` parameter (std-only), `field(key, value)` tail, text on a terminal / JSON otherwise with `VELES_LOG`, `withFields` on a task-local** (user, recommended of 3, 3, 3 and 2; specs D90, D91). Rejected: `@lazy` attribute, a compiler special case, a lambda-only call, a manual guard, a field map, a struct per message, text only, a sink trait, an explicit Logger value. |
| 2026-09-30 | Less nesting (user: "I do not want to have to create 'towers of terrors' when programming safely and 'properly'"; batch 1 of `archive/veles-spec-prep.md`) | **`with x = e` as a statement — the rest of the enclosing block is its body — and `with t = async f()`, a fail-fast background child cancelled then joined when the block ends; a with-bound value may not outlive its block; the whole tree migrated, no warning kept for the old shape** (user, recommended of 3 for resources, of 4 for tasks, of 3 for the spelling; migration: "just migrate all of our codebase, not needed support for later"; spec D100). Rejected: `defer`, blocks only, a bare `scope` statement, implicit scopes in every block, `with val f =` / `val f = with`, a warning plus fix. |
| 2026-10-01 | `Mutex` as a `with` guard (spec §7, open since D43) | **`with p = m.lock()` binds `*T`, the held region refuses suspension, `withLock` kept for one expression** (user, recommended of 3; spec D107). Rejected: replacing `withLock`, `withLock` only. |
| 2026-10-01 | Q1: send arms in `race` | **`ch.send(v) => body` arms; a losing send arm never sent; operands evaluated once at the start** (user, recommended of 3, after the example of a logger dropping a line when its queue stays full; spec D108). Rejected: leaving it, keeping it open. |
| 2026-10-01 | `with expr` without a name | **Allowed in both forms, items mix in the block form** (user, recommended of 2; spec D109). Rejected: requiring `with _ =`. |
| 2026-10-01 | Concurrency helpers (notes P11) | **Pre-approved: `retry(times, f, delay:)`, `Semaphore(permits:)` with a `Closeable` `Permit`, `ch.forEach` / `ch.toList`, `ticker(every:)`** (user, all four ticked; spec D110). Asked later: `filterConcurrent`, `firstConcurrent`, stages, `awaitAll`. |
| 2026-10-01 | Q2: setup/teardown, asked again with D100 | **Still open** (first round). Presented: suite-level `with` lines around each test, lending functions (`with fun … yield v`), both, neither. User: "I'm not sure yet, I still kinda don't understand what this should do in normal language... Are lending functions like function generators in js". |
| 2026-10-01 | Q2: setup/teardown, the user's own idea | **Go-style values: a value holding a `Task` must be received with `with`; its tasks are fail-fast children of that block, cancelled and joined before its `close()`** (user: "maybe returned reference to something should work like in golang? and that would work with `with` no?"; then recommended of 4; spec D111). Rejected: lending functions ("not convinced"), suite-level setup, leaving it open. |
| 2026-10-01 | Q9: `Secret<T>` | **Prelude `Secret<string>` / `Secret<List<u8>>`: `[redacted]` when printed, not Encodable, Decodable, `expose()`, constant-time `==`, zeroed on `close()` and by the collector when freed** (user, "also zero when collected" over the recommended close-only; spec D112). Rejected: leaving it. |
| 2026-10-01 | Q10: compile-time evaluation | **Constant expressions over consts, constant tables read-only in the binary, `const fun` evaluated by the compiler, `static assert(cond, "why")`** (user: "I think expressions and tables and comptime functions", over the recommended expressions + tables; `const fun` recommended of 3; spec D113). Rejected: `comptime fun`, unmarked inference, expressions only. |
| 2026-10-01 | Q11: bounds checks | **No global switch; `atUnchecked` / `setUnchecked` / `byteAtUnchecked` in `unsafe`, still checked in debug** (user, recommended; spec D114). Rejected: `--release-unchecked`, no unchecked access. D21's overflow policy not reopened. |
| 2026-10-01 | One `try` over a chain (left open by D98; found in the consistency pass) | **Swift's rule: one `try` unwraps every failing link of its receiver chain; arguments not covered; the parentheses warning removed, an inner `try` redundant** (user, recommended of 2; spec D134). Rejected: one `try` per failing call. |
| 2026-10-01 | Downcast on a trait object (found in the consistency pass: D129 needs it) | **`x is T` for a concrete `T` on an open trait object, by type id, narrowing** (user, recommended of 2; spec D135). Rejected: an `asSql()` method instead. |
| 2026-10-01 | Health endpoints (left out of D126) | **`http.Health` builder: `/healthz` runs no checks, `/readyz` runs them all with timeouts and answers 503 once a graceful stop begins; details logged, not returned** (user: "yes, add health endpoints"; spec D133). |
| 2026-10-01 | Q14: the manifest | **`package.vs`: one typed constant `const package = Package(…)`, evaluated alone under D113; the same declaration in `.vss` scripts; conditions on target facts only (`target.os/arch/release`; spelled `build.os` when asked), no environment** (user, after seeing TOML, `veles.mod` and this in full: "having 'Package' struct, might tell us, that in .vss files we could have similar thing"; the recommendation then moved to it; spec D131). Rejected: `veles.mod` (recommended first), flattened TOML, environment conditions, no conditions. Notes #3 (dependencies) and #17 close. |
| 2026-10-01 | Where packages come from | **Decentralized: git repositories with `github:`-style shorthands, the major version in the version not the path, tags only (explicit `commit(…)` pins, no pseudo-versions), MVS per (repository, major), mandatory `veles.sum`, optional proxy, no lockfile, a search index later** (user: "decentralized but … go works fine but look awful"; all four refinements; no lockfile recommended; spec D132). Rejected: a central registry, Go's `/v2` paths and pseudo-versions, a lockfile. |
| 2026-10-01 | `std/compress` ceiling | **A 64 MiB default, `max:` to change it** (user, over the recommended required `max:`; spec D124). Rejected: required `max:`, no limit. |
| 2026-10-01 | `std/config` | **A struct decoded from the environment, dotenv/JSON files beneath it, every problem at once** (user, recommended of 3; spec D125). Rejected: env only, getters. |
| 2026-10-04 | `std/config` mechanism | **`Decodable.schema()` written by the derive; an empty variable is a value; dotenv has no interpolation** (user, all recommended; spec D125 addendum). Rejected: environment-scan decoder, opt-in `Configurable`, empty = unset, `${VAR}` expansion. |
| 2026-10-01 | Observability | **OpenTelemetry: metrics, traces and logs, OTLP over HTTP with protobuf** (user: "OpenTelemetry" over the recommended Prometheus-shaped metrics; then "All signals + protobuf" over metrics + traces with OTLP/JSON; spec D126). Rejected: Prometheus-shaped metrics, counters/gauges only, metrics first. Health endpoints left to confirm. |
| 2026-10-01 | HTTP client | **Both: `http.fetch(url, …)` and `http.get/post/…` + `http.Client`** (user, over the recommended `fetch` only; spec D127). |
| 2026-10-01 | Streams and TLS | **`io.Stream` implemented by `net.Conn`, `tls.Conn`, `fs.File`; HTTP over any stream; TLS verification on, `dangerouslyAcceptAnyCertificate` the only opt-out** (user, recommended of 2; spec D128). |
| 2026-10-01 | SQL injection | **Template literals (`@template`, `tag"…"`) and `std/db` taking only `db.Sql` from `sql"…"`** (user, after asking "how tagged literal would look and work, will it be safe for injections?" and the example; recommended of 3; spec D129). Rejected: constant SQL + arguments, plain strings. |
| 2026-10-04 | `fs.lines` | **An iterator of `Result<string, IoError>` that is also `Closeable`; `max:` required** (user, both recommended; spec D130 addendum). Rejected: a reader with a throwing `next()`, an iterator that ends silently with `error()` afterwards, a default `max`. |
| 2026-10-04 | HTTP client request body | **A `Payload` value (`Payload.text/json/form/bytes`) in one `body:` parameter** (user, recommended of 3; spec D127 addendum). The original `json: T? = null` cannot be inferred. Rejected: `postJson`-style functions, explicit type arguments. |
| 2026-10-04 | Health: readiness at the stop | **`http.serve(…, health: health)` flips `/readyz` to 503 when the stop begins** (user, recommended of 3; spec D133 addendum). Rejected: a hand-called `health.drain()`, a stop hook returned by `endpoints()`. |
| 2026-10-05 | otel ↔ http cycle | **An explicit exporter: `otel.start(service:, exporter: http.otlp(endpoint:))`; `std/otel` imports nothing from `http`** (user, over the recommended `http.Observer` hook, after asking for the breakdown of what otel is; spec D126 addendum). Rejected: a hook slot in `http`, a mini HTTP client inside `otel`. |
| 2026-10-05 | Binding a span | **`TaskLocal.bind` (a public `Closeable`) plus `otel.inSpan`** (user, recommended of 3; spec D126 addendum). Rejected: closure form only, `bind` alone. |
| 2026-10-01 | Small std additions | **`http.testServer`, endian bytes, `fs.writeAtomic`/locks/`lines`/…, argon2id password hashing** (user, all ticked; spec D130). |
| 2026-10-01 | Q7: C layout | **Compiler-known `@packed`, `@align(n)` (any struct), `@transparent` (one-field struct), and `extern union` (fields only in `unsafe`)** (user, recommended of 3; spec D120). Rejected: a layout clause, leaving it. |
| 2026-10-01 | Fixed-size arrays | **`Array<T, N>` everywhere, inline value type, with `<const N: i64>` parameters** (user, recommended of 4; spec D121). Rejected: `[T; N]`, extern-only arrays, leaving it. |
| 2026-10-01 | Q7: `.d.vs` | **Dropped: a binding is an ordinary package** (user, recommended of 3; spec D122; notes #16 closed). Rejected: a declaration-only file kind, later. |
| 2026-10-01 | Q7: variadic C functions | **Calls only, with C's default promotions; never defined in Veles** (user, recommended of 2; spec D123, superseding §4b). New evidence: POSIX `open`/`fcntl`; LLVM does the per-platform call. Rejected: keeping them excluded. |
| 2026-10-01 | Suspension through a function parameter (plan B8's blocker) | **Follows the argument: a `suspends` parameter means "may"; the function is compiled plain and as a coroutine, each call picks; the eager adapters move into the prelude** (user, recommended of 3; spec D116). Rejected: an explicit marker, leaving it. |
| 2026-10-01 | Q5: `is Trait` | **Both: `x is Trait` at run time on trait objects (a program-wide type→method-table lookup, narrowing), and `T implements Trait` at compile time in generics** (user, "Both" over the recommended run-time form; spec D117). Rejected: one form only, neither. |
| 2026-10-01 | Q6: user-defined derivation | **Later, through compile-time reflection on `const fun`** (user, recommended of 3; spec D118). Rejected: macros, designing it now. |
| 2026-10-01 | `Default` | **Prelude trait, opt-in by empty `implement Default`, `T.default()`** (user, recommended of 3; spec D119). Rejected: automatic, leaving it. |
| 2026-10-01 | A `Closeable` never closed | **A warning with the fix `val` → `with`; passing counts as a hand-off** (user, recommended; spec D115). Rejected: an error, nothing. |
| 2026-09-30 | Q13 (I2): which heads write `val` | **A head that holds an expression (`if`, `when`) writes `val`; one that can only bind (`with`, `loop`, `catch`) does not; `when (m = …)` is one error with a fix** (user, recommended of 3; spec D101). Rejected: no `val` in `if`/`when`, `val` in every head. |
| 2026-09-30 | Q13 (I3): changing a collection while a loop walks it | **Refused: a length/order change on the looped path is a compile error; a change through another path panics at the loop's line (a modification count in the header); element replacement allowed** (user, recommended of 4; spec D102). Rejected: compile time only, snapshot, documenting the live-index behaviour. |
| 2026-09-30 | Q17 (a, b): one-task `gather`; `async` on a function value | **One launch yields its `Result` itself; `async f()` allowed on a `sendable fun` value, refused with a fix on a plain `fun`** (user, recommended of 3 and 3; spec D103). Rejected: the 1-tuple, an `attempt(f)` function, any function value, keeping the refusal. |
| 2026-09-30 | Q20: bit operations | **Rust's names camel-cased: `rotateLeft/rotateRight(n)` (mod width, negative reverses), `swapBytes`, `reverseBits` on every integer type; `copySign`, `isSignNegative`, `nextUp`, `nextDown` on floats** (user, recommended of 3; spec D104). Rejected: a Go-style `bits` module, Java's names. |
| 2026-09-30 | Q12: capacity hints | **`reserve(n)` on `StringBuilder`, `MutableMap`, `MutableSet`, `Deque`** with D83's meaning (user, recommended of 3; spec D105). Rejected: `StringBuilder` only, leaving it. |
| 2026-09-30 | Q17c, §6.1, empty literals | **`var xs = []` keeps needing its type and the tool writes it from the first deciding use; `Range.isEmpty()`; `veles fmt` prints `cond => (x => …)`** (user, recommended of 3; both small ones accepted; spec D106). Rejected: Rust-style inference from later use, the message only; a lint for the lambda arm. |
| 2026-10-02 | A static of a generic type without type arguments (found building D112: `Secret.of(v)` did not compile) | **Inferred from the arguments and the expected type, as a constructor's are, for every generic struct and the prelude's collection statics** (user, recommended of 3; spec D137). Rejected: inferring for `Secret.of` alone, respelling D112 as `Secret<string>.of(v)`. |
| 2026-10-02 | D111's hand-off, found unsafe while building (a finishing task can race the move between scopes; a cancelled caller can orphan staged tasks) | **The running task carries the receiving `with`'s scope while the value is computed; held tasks launch straight into it** (user, recommended of 2; spec D111 note). Rejected: adopting after the return with the races patched. |
| 2026-10-01 | Closing a `with` value by hand (found building B10: the close hint showed `close()` running twice) | **Refused: `close()` on a `with`-bound value or its alias is an error with a fix that removes it; close earlier with the block form** (user, recommended of 3; spec D136). Rejected: `with` noticing the hand close and skipping its own; leaving it and requiring every `close()` to tolerate a second call. |
| 2026-10-04 | How a trait says its objects are `Sendable` (found moving `http` onto `io.Stream`: its connection crosses into `withTimeout`/`async`) | **A trait may require `Sendable` as a supertrait** (`trait Stream : Closeable + Sendable`): its objects are Sendable, each implementor must be Sendable (error at its `implement`), and boxing one that is not is an error (user, recommended of 3; spec D35 addendum). Rejected: a `sendable trait` modifier (new syntax for the same meaning); objects never Sendable with http generic over the stream (a type parameter on every http type). |
| 2026-10-05 | E7 planning: what a "major" is below 1.0 | **Cargo's rule: the major is the first non-zero component (`1.4.2` → 1, `0.3.1` → 0.3, `0.0.2` → 0.0.2)** (user, recommended of 3). Rejected: Go's v0-is-v1, refusing 0.x. The registry's listed tier takes 1.0.0 or later; 0.x lives in the unreviewed tier (D138). |
| 2026-10-05 | E7 planning: how packages are fetched | **git and a proxy, with a built-in HTTPS client (no `git` needed for the proxy path)** (user, free-text over the options). The proxy and the registry are the same service (D138). |
| 2026-10-05 | E7 planning: monorepos | **Must be supported and easy/comfortable** (user); **`[workspace]` builds only, publishing members later** (user, over the recommended per-package versions with prefixed tags; spec D138). |
| 2026-10-05 | Q21: manifest format | **`veles.toml` now, schema ready for `package.vs`** (user, recommended of 3; D138). Rejected: TOML only; `package.vs` now. |
| 2026-10-05 | Q21: where packages come from | **Registry + git hybrid** (`owner/name` registry as index, immutable mirror and proxy; git URLs the unofficial path) (user, recommended of 3; D138). Rejected: decentralized only; registry only. |
| 2026-10-05 | Q21: review and safety | **Automated gates + capability policy + opt-in signed reviews; tiers unreviewed / listed (>= 1.0.0)** (user, recommended of 3; D138). Rejected: mandatory manual approval; checksums only. |
| 2026-10-07 | E7 stage g: how a package gets into the registry | **Both: the service pulls from git, and `veles publish` uploads** (user, over the recommended pull-from-git only; D139). Rejected: upload only. |
| 2026-10-07 | E7 stage g: tier and yank | **`unlisted` as a policy capability; yanked versions refused for new resolutions, a warning for a build whose `veles.sum` holds them** (user, recommended of 3; D139). Rejected: a separate `tier` key; tier shown only. |
| 2026-10-07 | E7 stage g: reviews | **Portable ed25519-signed statements, trust by key in `[policy] trust`, `require = { reviewed = N }`; served by the registry or committed under `attestations/`** (user, recommended of 3; D139). Rejected: registry-hosted only; cargo-vet-style imports by URL. |
| 2026-10-07 | E7 stage g: registry integrity | **Signed `.info` with a key pinned in `[registry] key`** (user, recommended of 3; D139). Rejected: trust on first use alone; a Merkle transparency log now. |
| 2026-10-08 | Expression bodies | **`fun f(): T => expr`** (D140): `=` only binds, `=>` only yields; `= expr` is a removed spelling with a fix (user, recommended of 3, prompted by the `implicit_return` levels). Rejected: keep `= expr` (§4); decide later. |
| 2026-10-08 | Task handles (user: "can we await tasks outside of the scope? … what for is the scope really? what about unawaited tasks?") | **A handle stays in the `scope`/`gather` that started it — returned, stored in anything declared before the block, or captured by a lambda stored there is an error — and in a fail-fast block `await` on a throwing child gives the value (`Task<R>`, no `try`); `gather` and D111 fields keep `Task<Result<R, E>>`** (user, recommended of 3 for each; D141). Rejected: an `await` after the scope that throws `Cancelled`; Swift-style delivery of an awaited child's error to the `await`; leaving either as it was (a run-time panic; an `Err` arm that never ran). |
| 2026-10-08 | Visibility levels (user: "let's think how could we do less `public` painting but have the default internal/private things") | **`private` the type, `internal` the module, unmarked the package, `public` other packages, exported at its module path; `public use` stays as a facade; a member another package cannot see is supplied to the implicit constructor when it has no default and defaulted otherwise ("if it is needed for creation, it should be needed for creation"); top-level `private` an error (fix `internal`); `public` in a program a compiler warning with `--fix`** (user's own proposal, over the recommended "members default to their type's level"; D142, amends M5 and D89; build: plan B17). Rejected: members inheriting their type's level; a `data struct` modifier; a `public { }` block; leaving M5. Opened Q22, Q23. |
| 2026-10-08 | Q22: facade re-export of unmarked items | **Allowed: `public use` in a root may export an unmarked item, reachable outside only under the facade's name; `internal`/`private` refused** (user, over the recommended "only `public` items"; D142). Rejected: only `public` items re-exportable. |
| 2026-10-08 | Q23: a `public` signature naming an unexported type | **Error with fixes (make the type `public`, or drop `public`); a type exported by a facade counts as visible** (user, recommended of 3; D142). Rejected: a warning; allowing it. |
| 2026-10-09 | Q24: OS-thread control (concurrency review; user: "possibility to precisely manage the native OS threads") | **Executors as values and structured threads**: `[runtime] threads` in the manifest, `Executor(threads:, name:)`, `Executor.thread(name:)`, `scope(on:)`/`gather(on:)` placement (a task stays on its executor), `e.run(f)`, `blocking(f)` on a bounded pool, `Thread.start(name:, priority:, cpus:, stackSize:)` joined by its `with`; a refused priority/affinity throws `ThreadError` (user, recommended of 4; D143). Open after building: non-Sendable state on a single-thread executor. Rejected: a minimal set; raw threads only; `VELES_THREADS` only. |
| 2026-10-09 | Q25: atomics | **`compareAndSet`, `compareExchange`, integer `add`/`sub` (new value), `fetchAnd/Or/Xor` (old value), an optional `order: MemoryOrder` (default `SeqCst`; invalid for the operation = compile error), lock-free atomic pointers** (user, recommended of 3; D144). Rejected: SeqCst only (Go); leaving it. |
| 2026-10-09 | Q26: loops that never suspend | **A pending cancellation unwinds at any loop back-edge (not in a lock region or `close()`); suspending functions also yield there after ~10 ms; `yieldNow()`, `checkCancelled()`** (user, recommended of 4; D145). Rejected: explicit points only; oversubscribing threads; leaving it. |
| 2026-10-09 | Q27: synchronisation types | **`RwLock<T>`, `Event`, `Lazy<T>`, `Broadcast<T>` + `Watch<T>`** (user, all four offered; D146). Not offered: `Barrier`/`Latch`. |
| 2026-10-09 | Q28: a `close()` that suspends | **An `implement Closeable` may declare `close()` `suspends`; a `with` on that type is a shielded suspension point; generic code gets two instances (D116); such a type cannot be a `Closeable` trait object** (user, recommended of 3; D147). Rejected: a second trait; explicit `shutdown()`/`rollback()` only. |
| 2026-10-10 | D143: making an `Executor` that the OS may refuse (D73: an `init` cannot throw) | **Static functions that throw: `try Executor.pool(threads:, name:, priority:, cpus:)`, `try Executor.thread(name:, …)`** (user, recommended of 3; D143 addendum). Rejected: an `init` that may throw; `Executor(...)` that panics plus a throwing `Executor.pinned`. |
| 2026-10-10 | D143: `Thread.start`'s function (no trailing lambdas) | **The argument `f:`** — `Thread.start(name: "audio", f: () => audioLoop())` (user, over the recommended `run:`; `body:` also offered). |
| 2026-10-10 | D143: joining a `Thread` | **Its close blocks the closing OS thread (a blocking call, D66), as Rust's `join`** (user, recommended of 2). Rejected: a close that suspends (D147) — a `Thread`'s plain function could not join one. |
| 2026-10-09 | `MutableList<i64>` indexed increment | **`incrementAt(i)` increments in place, returns unit and panics out of range** (user, recommended of 2; D148). Rejected: return `bool` and leave invalid-index handling to the caller. |
| 2026-10-09 | Transform one mutable-list element | **`MutableList<T>.updateAt(i, transform): T` calls a synchronous, non-throwing transform once, stores and returns the replacement; negative indexes count from the end and out-of-range panics** (user, recommended of 3; D149). Rejected: `mapAt`, unit return, or no helper. |

## 11. Known limitations to revisit

- **Executors (D143, built 2026-10-10), not done yet:** on Windows a thread can only be kept
  to CPUs 0–63 (other processor groups throw `ThreadError`; `SetThreadGroupAffinity` would lift it);
  macOS (plan A7) names threads but refuses any priority or CPU list; a blocking call on a pool's
  thread does not hand its queue to a spare thread as the default pool's does (a pool keeps the
  threads it was made with); a `Thread`'s plain function cannot run tasks on an executor (`run` and
  `scope(on:)` suspend; D143 said it "may create and run an executor of its own" — a blocking form
  of `run` would need a decision); timers and sockets are served by the default pool only, so a
  default pool whose every thread is in a long plain loop delays a pool task's `sleep` too;
  `async cpu.run(f)` is refused (a method's receiver is a pointer, not Sendable — wrap the call in
  a function taking the `Executor`); with CPU work on a pool the default pool's request latency
  keeps its median but its tail grows — p99 +20–40 % on Windows, +35–100 % on Linux (WSL), the
  worst request 5–6 ms against ~1.4 ms idle (not explained: SMT siblings and turbo clocks shared with
  the spinning threads are the likely cause; the monitor's look at the pools under the runtime
  lock is the other suspect).

- **Found by the concurrency review (2026-10-09, `veles-concurrency-review.md`), not fixed yet**
  (B1 and B2 are fixed; B3 is decided as D145 and planned as F4): (B4, medium) `veles_task` is ~430 bytes (15 flag
  words, panic and debug data, one field set per wait kind); ~560–670 bytes per parked task, target
  ≤ 250 — F2 (2026-10-09) took the task to 248 bytes and a parked task to 400. Missing benchmark:
  `bench/idle` (peak memory per parked task). On Linux a just-launched child takes up to
  ~7 ms to start at 8 threads (Windows ≤ 0.3 ms): the wake-up path (plan F9). Many tasks
  parking on one channel at 32 threads still spend ~1.5 s of CPU per 100k parks on its lock (bounded
  now; a per-channel waiter queue without the lock would remove it). Plan track F has the order.

- Visibility (D142, decided 2026-10-08, not built: plan B17). Kept for later:
  a file-scoped `private` on top-level declarations (Kotlin's rule) — refused
  as an error today so the word stays free; take it up if large modules ask
  for per-file helpers. A semver check at `veles publish` that names a field
  without a default added to a `public` type (it breaks other packages'
  construction calls) — not built.
- Arrays (D121): `ref(i)` and in-place `sort`/`swap` are not offered (`set` and
  `loop (&x in a)` are); a type argument may not calculate from a constant
  parameter (`Array<u8, N + 1>`), and a constant parameter is an `i64`; every
  read-only `List` method except `map`/`filter`/`forEach`/`fold`/`any`/`all`/
  `find` copies the array into a list per call; a value of 128 bytes or more
  that holds an array is copied wherever it is passed or assigned (share one
  with a pointer); a local array is on the stack, so one of hundreds of
  megabytes may overflow it (the type allows a gigabyte); a pointer array's
  collector descriptor lists every element's pointer, so `Array<string, N>`
  with a huge `N` has a large descriptor.
- A `with` resource may not leave its block (D100 part 3), but the check is
  local: it sees the value returned, assigned, stored in a collection or
  captured, directly or through a `val` alias or a literal around it — not
  what a function it is lent to does with it (by the decision), nor a value
  derived from it by a call (`conn.reader()`).
- A task handle may not leave its scope (D141) by the same local check: a
  function the handle is passed to may keep it (in a global, say). Awaiting
  it after its scope then panics at run time ("awaited task was cancelled",
  or "awaited task failed; its error left with its scope" when the scope
  failed). An `await` on a failed child inside its scope yields until the
  scope's failure abandons or cancels the awaiter — a short spin on the run
  queue, not a parked wait.
- Run-time concurrency gotchas, documented in
  `docs/documentation/reference/concurrency.md` (2026-10-08), that the
  compiler could catch — each a decision (`veles-decide`) before building:
  `withTimeout` (or a `race` against a task) given a function the compiler
  knows never suspends cannot be stopped, and reports `Timeout` only after
  the function finished (D116 tells which lambdas suspend: an error or a
  warning is possible); `await t` after `t.cancel()` in the same block
  panics unless the task had finished (a local check could refuse it);
  `send` on a closed channel panics (not statically knowable in general).
  A partial deadlock (some tasks stuck while a timer or socket wait exists
  elsewhere) is not detected; only a whole-program one is.
- A child launched with `async` that *returns* a `Result` without `throws`
  is not a failing child in a `scope` (D141, pinned by
  `driver/taskhandle_test.go`); how `gather` and a D111 field treat one was
  not reviewed in that change.
- A test's output is captured through `io` only: what C code writes itself
  (`printf` through FFI) goes straight to the terminal.
- `expectPanics(body)` runs `body` in a task of its own, so `body` must be
  a sendable function: it cannot capture a `MutableList` of the test's.
- A panic std raises for a caller's misuse points at the caller only for the
  functions marked `@caller_location` (D88: `swap`, `insert`, `removeAt`,
  `chunked`, `windowed`, `step`, `toString(radix:)`, `randomBytes`, `Uuid.of`,
  `random.range`); other std panics (`Hmac.update` after finish, the JWT key
  length check, `mapConcurrent` and `retry(0, …)` — which suspend) still
  report the std line. `Semaphore(permits:)` and `time.ticker` are marked.
- `T?.decode(from)` written by hand parses as a safe call on `T`; use a
  generic (`fun decodeIt<T: Decodable>(...)`) or a field. Derived code uses a
  resolved-type receiver and is unaffected.
- `crypto.randomBytes` returns the bytes through a `string`-shaped runtime
  buffer and `.bytes()`, the convention `fs.readBytes` already uses. It
  works (the length, not a NUL, delimits the buffer) but a `List<u8>` out
  parameter would be one copy cheaper.
- A `Secret`'s bytes are wiped (D112), but what `expose()` returns, the
  text a `Secret` was made from, and HMAC's pads are ordinary memory.
- A function value that returns a value holding a task (D111), passed
  through a generic std function (`withTimeout(d, () => serving())`), is
  refused — but the error points into the std function's body, where the
  value is kept in a variable, not at the call that passed it.
- Conditional suspension (D116) covers first-order `suspends` parameters
  only: one whose own parameters are suspending functions (a callback that
  takes a callback) makes its function suspend whenever called. So does a
  parameter the function keeps — stores, returns, or hands to a function
  that is not itself conditional. Trait methods keep their declared
  effects.
- `T implements X` (D117): each instance checks only the branch it takes,
  so a mistake in a branch no instance takes is not reported until one
  does. `static assert(T implements X, "…")` states the requirement per
  instance (B14).
- Constants (D113): a constant can call only a `const fun`; a constant
  `Map`/`Set` whose key type has its own
  `hash`/`equals` is refused (the compiler lays the table out with the
  structural hash); a sealed variant is not a constant value; a table's
  element is a `when` pattern only through a named `const`
  (`const FIRST = T.at(0)`), patterns being names and literals.
- **Deferred from E7 planning (2026-10-05, D138), to build later:** publishing the members of a workspace (per-package versions
  with prefixed tags `pkg/a/v1.2.0`, optional lockstep; path dependencies turned into version dependencies on publish); a local
  override while developing (Go's `replace`) — decide with publishing; the hosted registry service (the protocol and a reference
  server come first); `package.vs` (D131) and its `package` keyword clash; a transparency log for the registry (D139 chose a signed `.info`); replay protection of a signed `.info`; ed25519 in `std/crypto` for a Veles-written client.
- **Deferred from E7 stage (g), the registry (2026-10-07, D139), to build later:** the hosted registry service itself, with
  registration of `owner/name` → repository, the service-side pull from git, the real listing gates (builds on Windows and
  Linux, tests, API diff) and owner identity — the `registry` package is an in-memory reference only (no persistence, no
  `veles registry serve` command, one lock); a transparency log, replay protection of a signed `.info` (an old one with a
  valid signature can be served again), registry key rotation without editing every manifest, more than one registry
  at once (`[registry]` is one address; a dependency cannot choose); `veles publish` from a git tag's exact bytes (it
  zips the directory, so a package with hidden files hashes differently from its git tag); no `veles unpublish`
  (versions are immutable by design); yank state of a cached registry package is refreshed only for a new resolution,
  `fetch` and `audit`, not on every build; reviews: no revocation of a statement, no expiry, no claim vocabulary beyond the
  word (`reviewed` is the only one policy knows), no `attest` for path packages, `import` of a URL has no pinned
  host, keys are plain files with no passphrase; ed25519 is not yet in `std/crypto` (a Veles-written client needs it);
  `veles audit` does not list which statements it ignored (untrusted key, other bytes).
- **Deferred from E7 stage (f), capabilities (2026-10-07), to build later:** `veles deps` does not show capabilities (only `audit` and `add` do); the capability table is by module purpose and is a judgement (`log` reaches `os` and `otel` inside and counts as none; `config` counts as fs and os) — revisit when std grows, and when a std module can be audited by what it calls; a package that spawns a process only through a C library shows as `native`/`extern`, not `os`; no capability for `random`, `time` or `crypto`; no licence field check (`license` is read, nothing uses it); the analysis is by `use` and syntax, so a dependency cannot be caught reaching a capability it never imports (it can only reach through packages the report also lists); policy is per project, with no user-wide default (`~/.veles/policy.toml`); no machine-readable `audit --json` for CI.
- **Deferred from E7 stage (e), the commands (2026-10-07), to build later:** `veles add` does not check the new name against std
  modules and the package's directories (the loader reports it at the next build); `update` has no `--major`, no dry run
  (`--check`) and does not look at yanked versions; `add` accepts one spec at a time; `vendor` copies every version the
  resolution mentions (a smaller vendor would need the graph stored); `veles deps` prints no capabilities or licences yet
  (stage f); `remove`/`update` do not prune `veles.sum` in a workspace; the manifest editor does not touch
  `[dependencies.name]` tables (it says so); `veles fetch`/`deps`/`vendor` take one package, not a whole workspace at once.
- **Deferred from E7 stage (d), fetching (2026-10-05), to build later:** `[dev-dependencies]` are not resolved;
  the proxy protocol is the archive endpoint only — no version list, no checksum log, no yank (stage g), and
  `VELES_PROXY` is the only registry (no default URL until the service exists); a git repository is fetched whole-tag
  (`--depth 1`), a package inside a repository subdirectory is not supported (publishing workspace members); no
  authentication beyond what git does (SSH keys and credential helpers work, prompts are off); fetches run one after
  another, not in parallel; a resolution walks every version mentioned and downloads each (Go's graph pruning would skip
  some); the cache is never cleaned (`veles cache clean`); a workspace resolves each member on its own, not the
  whole workspace at once, so two members may select different versions of one package; the library warning fires for any
  fetched package, not only one with no `main`; a repository that is also reachable as another URL (https vs ssh) is
  two packages; no mirror/proxy fallback for git sources; the module cache is not safe against two builds fetching at
  once beyond the rename (a lost race is accepted).
- **Deferred from E7 stage (c), workspaces (2026-10-05), to build later:** `veles doc` at a workspace root documents the root
  as one package, not each member (`fmt` and the language server likewise treat a member as its own package, with no view of
  the whole root); `veles new --template workspace`; `veles test` at the root does not merge the members' reports into one
  count; members run one after another (no `--jobs` over members, no dependency order).
- **Deferred from E7 stage (a), the manifest (2026-10-05), to build later:** `veles = "…"` (minimum compiler) is read and
  checked but not enforced — the compiler has no version constant yet (add one with the release process, then refuse an older
  compiler with the manifest's line); `[workspace] members` takes directories only, not patterns such as `libs/*`;
  `[dev-dependencies]` are read but nothing loads them yet (`veles test` and `*.test.vs` in stage b/c); the policy table
  (capabilities, tiers: stage f) is not in the schema yet; `[dependencies]` accept an exact version only — no ranges, by D132's MVS;
  a package `version` that is not `major.minor.patch` (the old fixtures used any string) is now an error.
- **Deferred from E3, the reactor (2026-10-08), to build later:** kqueue for macOS (A7; until then
  macOS uses the `poll()` backend, which rebuilds its descriptor array per wait and is interrupted by
  every arm — tested on Linux by `driver/TestReactorPollFallback`); `writev`/`readv` and receiving into a
  list without the copy (`List<u8>` ↔ socket, §3.1); the runtime lock is still taken by `race_wait`
  (24 times per HTTP request — every `withTimeout`), `scope_cancel`, socket waits and the timer heap:
  `bench/httphello` at 32 threads (this machine's default) is about twice its 8-thread time, and making
  `race_wait` lock-free needs a design for cancellation, which today detaches a running race under that
  lock (a node left on a channel would swallow a later send); idle workers still sleep on one condition
  variable under the runtime lock (Go parks each thread on its own note), with the spinning added here in
  front of it; the Windows backend watches sockets only (an `ioWait` on another handle is "ready" at
  once and its call retried); `sleep` and race timeouts take whole milliseconds, rounded up, so a
  `Duration` of 50 µs sleeps a millisecond (the deadline itself is kept in nanoseconds); timers are one
  heap under the runtime lock (Go keeps one per P).
- **Deferred from `const fun` (2026-10-05), to build later:** `throws` and
  `try`/`catch` inside a `const fun` (a thrown error would be a compile error
  at the constant; today the function is refused); trait objects (a call through
  one is refused; lambdas, closures over `var`s and named `const fun`s as values
  work since 2026-10-08, with `map`/`filter`/`fold`/`sortedBy`… marked); the functions the compiler derives
  for a user type (`equals`, `hash`, `toString`, `compareTo`, the codec) are not
  `const fun`s, so `==` on a struct with a user `equals` and a key with its own
  `hash` in a constant `Map` are refused, though structural `==` and
  interpolation work; marked since 2026-10-08 (and listed in
  `std/testdata/const-functions.txt`): the function-free `List`/`MutableList` and
  `Range` helpers, `Duration`, the `hex`/`base64` encoders and `std/utf8` — still
  not: the lazy iterators (`iter()`, their adapters), the `Result` helpers
  (`Ok`/`Err` and other sealed variants are not evaluated as values), the decoders
  (they throw), `Duration.zero` (a `static val`), the number formatting
  (`toFixed`), `toF64()` (it calls `strtod`) and the other std modules; the float functions
  of the C library (`sin`, `cos`, `tan`, `exp`, `log`, `log2`, `log10`, `pow`,
  `atan2`, `hypot`) are not evaluated because two libraries differ in the last
  bit — a correctly rounded software implementation in the evaluator (or a
  decision to let them differ) would lift it; generic `const fun` bodies are
  validated per instance, as every generic body is (§11 above); a `const fun`
  body that reads a constant calling the same function reports the
  initialization cycle, not a `const fun` cycle; evaluation speed is about
  6 million steps a second (a tree walk over the HIR with a map per call frame),
  enough for tables of thousands of entries — a compiled evaluator or slots
  instead of maps would matter for a self-hosted lexer's Unicode tables;
  `tryConst`, which reads a constant index or a range bound while a body is
  being checked, runs no function (so `TABLE.at(f())` is not read at compile
  time); a long failure chain keeps its five outermost and innermost calls
  only.
- An initialization cycle is found through direct calls and lambdas; a
  call through a trait object or a function value read from elsewhere is
  not followed.
- The built-in types print, compare and hash without the prelude traits,
  so `i64 implements Display` is false although `"$n"` prints it.
- Hashes are plain Veles. They allocate nothing per block, but expect a
  multiple of a hand-tuned C implementation's time; there is no benchmark
  yet to say which multiple (§3.3).
- `time.now()` is the only clock `uuidV7` has, so two processes on one host
  can produce the same (millisecond, counter) pair. Uniqueness comes from
  the 62 random bits, as in v4; only the ordering is per-process.
- `Offset.local(at:)` goes through the C library's `localtime_r`, so it
  reads the host's zone, honours `TZ`, and is as correct as the platform's
  tz data. A fixed offset still cannot answer a *local* time on a DST
  boundary; that needs the IANA database (§5.6).
- `Timestamp` outside the years 0000–9999 prints ISO 8601's expanded year
  (`+011476-08-15T05:20:00Z`). That text is not RFC 3339, and the parser
  accepts it only so that `parse(t.toString())` reads back; a peer that is
  strict about RFC 3339 will refuse it.
- The last second at each end of the microsecond range prints and does not
  parse back: the parser stops where `secs * 1000000 + micros` would leave
  the i64, and recovering those two seconds needs arithmetic split across
  the boundary that no reader could check. Everything between them round
  trips (`examples/time` asserts both ends print).
- `Offset.local(at:)` outside what the host's C library will convert — a
  negative `time_t` or past the year 3000 on the Microsoft CRT — is the
  offset for the *same date in a year the host can do*, not the offset that
  zone actually had then. Without a tz database "the rules did not change"
  is the only assumption available; with one (§5.6) this stops being an
  approximation.
- `Duration.nanos(n).toString()` for the single most negative `n` prints one
  nanosecond short, because `abs()` saturates there rather than overflowing.
  It is the one `Duration` whose text does not read back, and it cannot be
  written as a literal.
- `sleep` rounds a `Duration` **up** to the executor's millisecond, so
  `Duration.micros(1)` sleeps for one millisecond rather than a
  microsecond. Sub-millisecond waiting needs a finer timer wheel in
  `veles_task.c`, which is a runtime change, not a library one.
- **Deferred from C2/C3/C5/C6 and the test-file rule (2026-10-04), to build later:**
  - `std/compress`: `inflate` could write into a sized buffer (the safepoint poll in each inner loop and
    the oversized-shift selects show in the IR); levels 7–9 on large noisy inputs are slow (no zlib-style
    give-up on a long chain); a Brotli codec, so `http.files` could offer `.br` siblings; `bench/results.md`
    holds two separately recorded gzip/gunzip tables.
  - `tls.Conn` implements `io.Stream` with E1 (TLS), so `http` serves HTTPS through the same code.
  - Codegen: dead-function elimination (every module's unused functions still reach the IR).
  - Unused-import lint: a file with a generic never instantiated skips the warning (its body is not
    checked); scan the generic's body syntactically instead.
  - `std/config`: a `Map` or a list of structs has no variable form and panics at the first call — make it a
    compile-time error; `os.setEnv` (tests give a lookup to the private `loadWith`); `.env` search upward from
    the working directory; `${VAR}` interpolation in dotenv (decided against for now); a JSON file's keys are
    the declared field names, `@key(json: …)` is not consulted; a hand-written `decode` has an opaque schema,
    so such a type is one variable; a generic sealed trait has no derive.
  - `std/fs`: byte-range and shared (read) locks; a copy that keeps permissions and times, and
    an atomic `copy`; on Windows `writeAtomic` cannot replace a file another handle has open (POSIX
    disposition semantics would); `fs.lines` yields `IoError` for a too-long line (an `io.TooLong` would need
    a union as a `Result` error type).
  - Endian bytes: `isize`/`usize` are fixed at 8 bytes (a 32-bit target would change it); no `Array`
    writers (`pushU32Be` is on `MutableList<u8>` only).
  - Tests: **exporting test code from a package** so people can write reusable test libraries (the user's
    "later", D78 amendment). (`TestStdFsUnitTests` failed under a full run because two `fs` tests used the same
    scratch file name while tests run in parallel; fixed by naming. A scratch helper that makes the name
    unique per test by construction would keep it from coming back.)
- **Deferred from C8 template literals (2026-10-04), to build later:** `std/db` and its `sql"…"` (plan E2);
  std tags such as `html"…"` that escapes and `regex"…"` (the literal is the mechanism, the tags are std
  work and public API); embedded-language highlighting inside a tag's string (SQL, HTML) in the editor
  grammar; LSP semantic tokens for the tag (the TextMate grammar colours it, the language server does
  not); a tag with explicit type arguments (`tag<T>"…"` is not parsed); multi-line and raw literals
  (Veles has neither yet) tagged; when the Veles lexer/parser is written (`veles-selfhost-frontend-plan.md`) it
  needs the `TemplateExpr` node and its adjacency rule, and the self-host oracle compares `(template …)` in `Dump`.
- **Deferred from E2 `std/db` (2026-10-05), to build later:** other databases (MySQL, SQLite) and the pooling of
  prepared statements — every statement is parsed afresh (an unnamed statement), a named-statement cache per
  connection would save the parse and is the first performance step (`veles-bench` it against a tight query loop);
  streaming a large result row by row (a result is read whole into memory; a `maxRows` ceiling and a cursor
  API are the way); `COPY`, `LISTEN`/`NOTIFY`, cancel requests (a statement past its time is cancelled by
  dropping the connection); a retry helper for `isRetryable()` transactions; reading `interval`, `numeric` with a
  fraction, `json`/`jsonb` and multi-dimensional arrays into typed fields (they read as text); `Duration`
  decoding (the decoder's duration style is seconds, the server's text is `HH:MM:SS`); SASLprep of passwords with
  non-ASCII characters (the password is used as written, which only differs for characters that normalise);
  unix-socket and `PGPASSFILE`/`PG*` environment connections; `target_session_attrs` and several hosts;
  `Pool.close()` only closes idle connections (a `Tx` left open now sends `ROLLBACK` at its close, D147, 2026-10-10); `examples/pgnotes` and a `veles new --template` with a database need a server in the example
  harness; `driver/db_test.go` skips when no PostgreSQL binaries are given (`VELES_PG_BIN`), so a machine without them
  runs only the tests that need no server; on Windows PostgreSQL sometimes resets the connection after refusing a login,
  which then shows as `ErrorKind.Connection` rather than `Auth` (the reset discards the unread error); `needsRehash`
  for stored password hashes whose parameters fell behind; `Uuid` is now `Codable` (text form) — a std addition made for
  `std/db`.
- **Deferred from E1 TLS (2026-10-05):** the server span's `url.scheme` is `http` even under `serve(tls:)` (a `Request` does not know it came over TLS); macOS (SecureTransport/Network.framework or the system's OpenSSL —
  the dlopen list names Homebrew's, untested; A7); client certificates (mutual TLS) and the server asking for
  them; revocation checking (CRL/OCSP) and OCSP stapling; session resumption tickets kept across connections;
  SNI-based certificate choice on one listener; custom cipher or version options (TLS 1.2 is the floor, the
  system picks the rest); a distinct `IoKind` for certificate failures (they are `Other`, `detail` starts `tls:`);
  `readLine`/`readExact` buffering is now copied in `net.Conn`, `fs.File` and `tls.Conn` (a shared `io` helper
  would remove it); `close()` sends `close_notify` without waiting only over a `net.Conn` (D147 addendum: over
  another `io.Stream` it is not sent; `shutdownWrite()` sends it and waits); a stored-key leak on Windows only
  if the process is killed (swept by the next TLS server); the encrypted PEM key; ALPN-driven HTTP/2.
- **Deferred from C7 the HTTP client (2026-10-04), to build later:** `https://` (done 2026-10-05) and `HTTPS_PROXY`;
  `HTTP_PROXY` / `NO_PROXY` and the `proxy:` option of D127 (an absolute-form request to the proxy; `CONNECT`
  for https); transparent `Accept-Encoding: gzip` + decoding (today a compressed answer's bytes are returned
  as they came; `std/compress` has the decoder, a streaming `GzipReader` exists); per-phase limits (connect,
  headers, between body reads) beside the one total `timeout:`; a streaming request body (`Payload` is in
  memory; an upload from a file needs a chunked writer); `Expect: 100-continue` on large uploads; a response
  that is never read or closed leaks its socket until exit (a finalizer, or `Closeable` enforcement beyond
  the D115 warning); a background reaper for idle connections (they are only dropped when the pool is next
  asked); HTTP/2; cookie jar and `Retry-After`; the client span and `traceparent` header (D126, with
  `std/otel`); charsets other than UTF-8 in `text()`; a public `http.Url` type (the parser is private); a
  `redirect` callback; `retry:` on a POST with an idempotency key.
- **Deferred from C4 health endpoints (2026-10-04), to build later:** the readiness `503` during a stop has
  almost no window today — `serve` stops reading requests on its connections as soon as the stop begins, so
  a probe sees a closed or refused connection rather than the 503; a delay between flipping readiness and
  closing (Kubernetes' `preStop` shape, e.g. `drainDelay:`) would make it observable, and is a design choice
  for the user; no `/startupz`; checks cannot report `degraded` (a pass/fail pair only); no spans exclusion
  yet because `std/otel` (D126) is not built — it must skip `quiet` responses; check results are not cached,
  so a probe every second runs every check every second (a `cacheFor:` option).
- **Deferred from `std/otel` (2026-10-05), to build later:** runtime metrics (GC pauses and heap need E5's counters;
  tasks, threads, open connections need runtime accessors) registered automatically as D126 says; a span per
  database query and `db.system` attributes (with `std/db`, E2); `https` export (E1) and the `https` client span
  attributes (`with otel.start(...)` flushes at its close since D147, 2026-10-10; `GzipWriter` and `Pool` could
  use a suspending close too); exponential histograms, exemplars, span
  links, `tracestate`, baggage and the W3C `baggage` header; limits on attributes per span and events per span
  (OTel's defaults are 128); delta temporality; `Retry-After` honoured by the exporter; the span of a retried
  `http.fetch` is one span for all tries (OTel makes one per try); `http.fetch` spans end at the answer's head, not
  when the body is read; HTTP semantic-convention attributes beyond the common ones (`http.request.body.size`,
  `network.protocol.version`, `url.scheme` on the client side); a Prometheus pull endpoint (not decided);
  `Span` is not Sendable (it owns the task-local binding) — a handle is passed instead; id generation is one
  `crypto.randomBytes` call per id (a per-thread buffer would be faster); sampling is by trace-id ratio plus the
  parent's decision only (no rate-limited or tail sampling); the otel module's tests share one process-wide
  pipeline and take turns (a pipeline value instead of a global would let them run at once).
- **Fixed on the way (2026-10-05): a runtime heap overflow in `veles_list_append_list`** (`addAll`, `concat`): the grown
  buffer was allocated in elements instead of bytes, so appending to a list of anything wider than a byte past its
  capacity wrote beyond the buffer and corrupted the next object — seen as a crash in the collector far from the
  append. `std/prelude/list.test.vs` pins it (it fails on the old runtime). Worth an audit: the other runtime list
  and map growth paths were read for the same mistake and have it right; a debug build that checks every slot header
  at each allocation (as the investigation did by hand) would find this class at once — not built.
