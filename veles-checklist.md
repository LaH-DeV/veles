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
      (`==`, hashing, printing already structural; `Default` deferred)
- [x] Supertraits: `trait A : B + C`, transitive bounds, super check on impls
- [x] Trait objects of a trait with supertraits: the table composes the supers' (`objectSlots` in `sema/supers.go`), inherited methods and their default bodies included; a combination trait is an object built from its parts; two supers declaring one name is an ambiguity and object safety is asked of the supers too
- [x] Parser: braceless empty `implement Trait` (body and top level); field attributes kept; formatter drops empty braces
- [x] Spec entry written (D58) with the rejected alternatives

### 1.2 Foreign function interface

- [x] C ABI FFI design → D67 (2026-09-26): extern blocks in any package, native libraries in the manifest; marshaling of strings/buffers and callbacks → D69
- [~] `extern "C"` blocks: calling convention and callbacks into Veles done (D69, 2026-09-26: `extern "C" fun` + `&name` : `extern fun(...)`, called through inside `unsafe`); varargs (`printf`) open [? §9 Q7]
- [?] `extern struct` layout: packed, explicit alignment, transparent wrappers
      (the "noted pressure point" under D51) — §9 Q7
- [x] Ownership at the boundary (D69): C keeps only copies (`ffi.CString`, `ffi.alloc`/`free`), a list is lent for a closure (`withRaw`, `CLayout` elements), a value C hands back travels as an `ffi.handle` (a scanned table index, never a GC address)
- [x] Panics never cross into C: a panic inside an `extern "C" fun` ends the process with its location (runtime `veles_ffi_enter`/`leave` around the body); errors cannot cross either (an exported fun may not throw)
- [x] Linking: `[native]` in `veles.toml` — `libs`, `static-libs` (archive resolved by name), `lib-paths`, `pkg-config`, file entries; dependencies' tables link too (2026-09-26, `driver/native.go`, TestNativeLinking)
- [?] Declaration files (`.d.vs`) so bindings are typed and shareable — §9 Q7
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
- [?] `race` send arms (`ch.send(v) => ...`) — asked 2026-09-27, undecided (§9 Q1)
- [x] Bounded channels with backpressure (`capacity: n`, blocked senders
      served in order); `race` is the `select` over receives, sleeps and
      tasks (D38). `Channel<T>()` is a true rendezvous (2026-09-27)
- [x] Each channel has its own lock; `race` claims its winner by CAS
      (2026-09-27): `bench/pipes` (8 independent pairs) 1209 → 13 ms at 8
      threads, `channels` 7.2 → 4.6 ms (TestChannelHandoffUnderThreads,
      TestRaceOverChannelsUnderThreads — the latter caught a race re-listing
      its nodes after a wake for another reason, cutting other waiters off)
- [x] Deadlock detection: "deadlock: every task is blocked" when no task
      can run and no timer, socket or blocking call can wake one
- [x] Blocking-call detection → superseded: a blocking call hands its
      run queue to a spare thread (below), so it no longer stalls other tasks

### 1.4 Type system and syntax

- [ ] Attributes with typed arguments → D51 (today the derive attributes
      check their own arguments; a general typed form is not designed)
- [x] Coherence/orphan rules for `implement`: none beyond D17 — any implement
      anywhere, one per (trait, type) pair program-wide (§10, derivation batch)
- [ ] `Default` values for generics without a hand-written implement
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
      to a method the element type lacks (`this.holdsNothing()` inside a
      `List<T>` extend) compiles until something uses it. Checking bodies
      against their bounds at definition — which the self-hosted compiler
      will want too
- [?] `const` evaluation beyond literals; `static assert` — §9 Q10
- [ ] Better inference for empty collection literals (`val xs = []` typed
      from later use; the typed-context half landed with the old note #8)
- [ ] Stable ABI story for `.vs` packages: none needed while source-only,
      but say so

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
      installed here; the build says so)
- [x] `readRequest` limits: `http.Limits` — request line, header line, header
      count (lines, not map entries), header bytes, body bytes, and three clocks
      (header, body, idle). A byte ceiling answers 414/431/413 and closes, a time
      ceiling 408; nothing reaches a handler, and a body is never assembled to
      discover it was too big
- [x] Every read from the network is bounded by a caller-given max: `readLine(max:)`
      has no default and throws `net.TooLong`, so the unbounded call does not compile;
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
- [?] Bounds checks stay on in release; a profile that removes them is opt-in
      and loud — §9 Q11
- [x] Integer conversions between widths: `toT()` is checked (`T?`), `wrapT()` is the explicit
      truncation, a literal that cannot fit is a compile error (D86, 2026-09-29;
      `examples/conversions`, conform `D86-conversions`, golden `arith`)
- [x] Constant-time comparison primitive in `std/crypto`: `Digest`'s `==`, and
      `crypto.equalBytes` for raw bytes. Structural `==` on `List<u8>`
      short-circuits, so a MAC is compared as a `Digest`, never as bytes (D59)
- [?] Secrets: a `Secret<T>` wrapper that does not `Display`, does not derive,
      and zeroes on collection — §9 Q9
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
- [ ] Resource leaks: a `Closeable` dropped without `with` is a warning
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
- [ ] **Stack exhaustion is silent and the stack is the OS default** (found by
      plan S3, 2026-09-29). A recursion that is not a loop overflows at about
      30 000–50 000 frames on Windows (1 MB stack) and 200 000–1 000 000 on
      Linux (8 MB); the process dies with exit code 127 (Windows) or a bare
      SIGSEGV (Linux) and prints nothing — no task, no function, no line.
      Tasks are stackless but run on a worker thread's stack, so the same
      limit holds inside one. Wanted: (a) a guard-page / vectored-exception
      handler that prints `stack overflow in <function> at <file:line>` and
      exits non-zero, in the sentence style of the other panics; (b) a stack
      size a program can rely on (main and worker threads created with a
      larger reserved stack, committed lazily). Measure with a recursion over
      a deep sealed tree. The conventional answer for input-driven walks
      stays `recursion.Depth`

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
      doubling storage (2026-09-27); `reserve` is a public API — §9 Q12
- [ ] `List<u8>` ↔ socket: writev/readv, no intermediate copies
- [ ] I/O: `poll` → `epoll`/`kqueue`/IOCP when connection counts justify it
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
      lever; measure `--timings`' clang line before and after

### 3.3 Benchmarks (so regressions are seen)

- [x] `bench/`: seven workloads (sha256, maps, sort, json, strings, GC-heavy
      trees, channels), each next to a Go program doing the same work, the
      checksums compared, and the result read as a multiple of Go —
      `go run ./bench` (2026-09-25). Still to add: an HTTP hello (needs a load
      generator), and parsing a large file once the self-hosted parser exists
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
- [ ] Chunked transfer-encoding: requests (501 today) and responses
- [ ] Streaming bodies: `Request.body`/`Response.body` as reader/writer,
      not `List<u8>`; `http.files` streams
- [x] Middleware: `type Middleware = sendable fun(Handler): Handler`; `router.wrap(m)`
      (`use` is the import keyword, so the verb is `wrap`) — outermost first, around
      the router's own 404s and 405s too
- [~] Standard middleware: `requestId()`, `logging()`, `timeout(d: Duration)` done; recovery
      needs nothing (a panic is already caught at the request boundary, D56) and
      body-limit is `Limits.bodyBytes`. Still open: CORS, auth hook, compression
- [x] Graceful shutdown (§4, D68)
- [ ] Max concurrent connections with backpressure at `accept`
- [ ] `Expect: 100-continue`
- [x] `HEAD`/`OPTIONS` defaults; `405` with `Allow` (2026-09-26: HEAD runs the GET route and the writer drops the body, keeping its `content-length`; OPTIONS answers 204 + `Allow`; a 1xx/204/304 is written without body or length)
- [ ] Cookies: parse and set, with `SameSite`/`Secure`/`HttpOnly`
- [ ] Forms: `x-www-form-urlencoded` (have `parseQuery`), multipart streamed to disk
- [ ] Static files: `ETag`, `Last-Modified`, `Range`, `Cache-Control`, index files
- [~] Router: method-not-allowed vs not-found distinction (done: 405 + `Allow`), route groups,
      typed path params (`{id: i64}`)
- [x] In-process test client: `http.call(handler, method, target, body:, headers:)` (2026-09-26; same target parsing and panic boundary as `serve`; docs 17 "Testing a handler")
- [~] Access log format: one line with peer, method, path, status, latency (a `Duration`, so the unit is in the line) and the
      request id (`logging()`); not structured (JSON/key=value) and no byte count yet
- [ ] HTTP/2 (later; needed for gRPC-style internal traffic)
- [ ] WebSocket (later)

### 5.3 `std/http` — client

- [ ] `http.get/post/...` and a `Client` with pooling, timeouts, redirects
- [ ] TLS (5.4)
- [ ] Retries with backoff and idempotency awareness
- [ ] Proxy environment variables

### 5.4 `std/tls`

- [ ] Client TLS via a system binding (SChannel on Windows, OpenSSL /
      rustls-ffi elsewhere) — blocked on 1.2
- [ ] Server TLS with certificate reload
- [ ] Certificate verification on by default; opt-out is loud

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
- [ ] Password hashing (argon2id) via binding
- [ ] RS256/ES256 — needs bignum or a binding (blocked on 1.2)
- [ ] A `Hasher` that is not a SHA: BLAKE3 or SHA-3 when something asks
- [ ] Zeroing: a key or a pad is left to the GC (see `Secret<T>` in §2)
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

- [ ] PostgreSQL: libpq binding first, pure wire protocol later (needs
      SCRAM-SHA-256 from 5.5)
- [ ] Connection pool with health checks
- [ ] Parameterized queries only; no string concatenation path
- [ ] Transactions as a `with` resource
- [ ] Row → struct mapping through the derive (1.1)
- [ ] Statement and connection timeouts
- [ ] Migrations helper (later)

### 5.9 Observability

- [ ] `/healthz`, `/readyz` helpers
- [ ] Metrics registry: counters, gauges, histograms; Prometheus exposition
- [ ] W3C `traceparent` propagation in the request-id middleware
- [ ] Runtime metrics: GC pauses, heap, tasks, open connections

### 5.10 Misc

- [ ] Float bits without `unsafe` (found by plan S3, 2026-09-29): `f64` ↔
      `u64` and `f32` ↔ `u32` reinterpretation (`toBits()`/`fromBits()` in the
      style of Go's `math.Float64bits`). A code generator needs it to write
      LLVM's hexadecimal float constants; today only
      `unsafe { p.cast<*raw u64>() }` does it. A std decision — `veles-decide`
      first
- [ ] `std/config`: typed env parsing, all missing keys reported at once
- [ ] `std/compress`: gzip/deflate via zlib binding
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

- [x] `veles fmt` stable on every example: `format/TestCorpus` now fails on any
      file under `examples/` or `std/` that `veles fmt` would change (2026-09-25).
      (docs blocks are not held to it: they align comments by hand)
      A 2026-09-24 review found it swept a function body's comments into a
      parameter list the author had wrapped: fixed, with two cases in
      `format/format_test.go`. That it took reading a formatted file to
      notice is the argument for the CI check
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
- [ ] Package registry / MVS (on the remaining list)
- [ ] Lockfile and reproducible builds
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
- [ ] `examples/pgnotes`: the notes API on PostgreSQL

---

## 8b. Self-hosting the front end

The lexer and parser rewritten in Veles, planned in
`veles-selfhost-frontend-plan.md`: six phases, each gated on byte-identical
output against the Go front end over the repository's own `.vs` files, ending
with the Veles parser parsing itself. Four decisions to make before it starts
(a rune API, `unicode.IsPrint`, the recursion limit, `Span`'s representation)
are listed in §4 of that file.

---

## 9. Decisions pending

Only what is still open. Each is asked through `veles-decide` (options,
examples, edge cases, pros/cons, a recommendation); the answer becomes a
spec entry and a row in §10, and the item leaves this list. The labels are
stable; §10 rows written before 2026-09-27 cite the old numbering (the
list as it was is in `archive/progress-log-2026-09.md` and git history).

(Q15 named imports was decided 2026-09-29: D85; Q16 `as` conversions the same day: D86; Q19 the same; Q4 D87, Q8 D88, Q3 D89 — decided 2026-09-29, being built in that order.)

**Asked, awaiting an answer**

- **Q1. Send arms in `race`** (`queue.send(line) => {}` next to a `sleep`
  arm: a send with a deadline, no task per send; a losing send arm never
  sent). Asked 2026-09-27: "idk yet". Today: `withTimeout(d, () =>
  ch.send(v))` or `trySend` polling.
- **Q2. Test setup and teardown per test or per suite** (user, 2026-09-27:
  "the setup for before and after isn't good... (no new keyword for them
  either)" — a suite's own `val`s and `with`s run fresh before each test
  was the rejected proposal). Today: a helper plus `with` in each test.

**Prepared or half-designed, not yet asked**

- **Q5. `is Trait`** (`x is Display`): trait-object RTTI (a per-type table
  in every box) vs a compile-time `T implements X` in generics vs neither.
  `sema/supers.go` refuses converting one trait object to another for the
  same reason.
- **Q6. User-definable derivation** — phase 2 of D58, once the compiler-known
  set is proven.

**Raised by the checklist, not yet designed**

- **Q7. FFI surface still open**: varargs calls (`printf`), `extern struct`
  layout (packed, alignment, transparent wrappers — the pressure point under
  D51), `.d.vs` declaration files (with notes #16).
- **Q9. `Secret<T>`**: no `Display`, no derive, zeroed on collection (§2,
  §5.5).
- **Q10. Compile-time evaluation**: what a `const` may hold beyond literals,
  and `static assert` for wire-format invariants (§1.4).
- **Q11. Build profiles**: bounds checks off only when opted into loudly;
  the release overflow policy (§1.4, §2).
- **Q12. `StringBuilder.reserve`** and other capacity hints as public API
  (§3.1).
- **Q13. Small syntax consistencies** (notes I2, I3): `when (val n = ...)`
  vs `loop (x in c)`; what mutating a `MutableList` while looping over it
  means.
- **Q14. Manifest dependency syntax** (notes #3, #17) — after M7 gives the
  manifest real content.
- **Q17. Small ergonomics found writing the D21 golden (2026-09-28)**:
  (a) `gather` with one task yields a 1-tuple, so `when (gather { async
  f() }) { is Ok ... }` fails with "'Ok' is not a type" — should one task
  yield its `Result` itself?; (b) `async f()` where `f` is a function value
  is refused ("launches a direct call of a named function") — a trampoline
  `fun call(f) = f()` works, so the rule costs code, not safety; (c)
  `Range.isEmpty()` for symmetry with every collection (written internally
  as `holdsNothing` in the prelude until decided).

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
| 2026-09-29 | Q3: package surface in source | **`public use m` and `public use m { a, T as U }`; manifest `exports` goes away** (user, recommended of 3; spec D89). Rejected: keep manifest exports, whole modules only. |
| 2026-09-29 | std/log design (plan C1) | **`lazy` modifier on a `fun(): T` parameter (std-only), `field(key, value)` tail, text on a terminal / JSON otherwise with `VELES_LOG`, `withFields` on a task-local** (user, recommended of 3, 3, 3 and 2; specs D90, D91). Rejected: `@lazy` attribute, a compiler special case, a lambda-only call, a manual guard, a field map, a struct per message, text only, a sink trait, an explicit Logger value. |

## 11. Known limitations to revisit

- A test's output is captured through `io` only: what C code writes itself
  (`printf` through FFI) goes straight to the terminal.
- `expectPanics(body)` runs `body` in a task of its own, so `body` must be
  a sendable function: it cannot capture a `MutableList` of the test's.
- A panic std raises for a caller's misuse points at the caller only for the
  functions marked `@caller_location` (D88: `swap`, `insert`, `removeAt`,
  `chunked`, `windowed`, `step`, `toString(radix:)`, `randomBytes`, `Uuid.of`,
  `random.range`); other std panics (`Hmac.update` after finish, the JWT key
  length check, `mapConcurrent` — which suspends) still report the std line.
- `T?.decode(from)` written by hand parses as a safe call on `T`; use a
  generic (`fun decodeIt<T: Decodable>(...)`) or a field. Derived code uses a
  resolved-type receiver and is unaffected.
- `crypto.randomBytes` returns the bytes through a `string`-shaped runtime
  buffer and `.bytes()`, the convention `fs.readBytes` already uses. It
  works (the length, not a NUL, delimits the buffer) but a `List<u8>` out
  parameter would be one copy cheaper.
- Keys and HMAC pads are ordinary garbage-collected memory: nothing is
  zeroed after use. `Secret<T>` (§2) is where that belongs.
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
- A generic function's type argument is inferred from its arguments only,
  not from the type expected of the call: `val x: i8 = id(127)` infers
  `T = i64` from the literal and then fails, where `id<i8>(127)` works.
