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
- [x] Field-level attributes with arguments the derive can read (`@key`, `@key(json: ..)`, `@skip`, `@tag`, `@required`)
      (`@key`, `@key(json: ..)`, `@skip`, `@tag`, `@required`) → D51 revisit
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

- [x] C ABI FFI design → D67 (2026-09-26): extern blocks in any package, native libraries in the manifest; marshaling of strings/buffers and callbacks still to decide
- [~] `extern "C"` blocks: calling convention and callbacks into Veles done (D69, 2026-09-26: `extern "C" fun` + `&name` : `extern fun(...)`, called through inside `unsafe`); varargs (`printf`) still open
- [ ] `extern struct` layout: packed, explicit alignment, transparent wrappers
      (the "noted pressure point" under D51)
- [x] Ownership at the boundary (D69): C keeps only copies (`ffi.CString`, `ffi.alloc`/`free`), a list is lent for a closure (`withRaw`, `CLayout` elements), a value C hands back travels as an `ffi.handle` (a scanned table index, never a GC address)
- [x] Panics never cross into C: a panic inside an `extern "C" fun` ends the process with its location (runtime `veles_ffi_enter`/`leave` around the body); errors cannot cross either (an exported fun may not throw)
- [x] Linking: `[native]` in `veles.toml` — `libs`, `static-libs` (archive resolved by name), `lib-paths`, `pkg-config`, file entries; dependencies' tables link too (2026-09-26, `driver/native.go`, TestNativeLinking)
- [ ] Declaration files (`.d.vs`) so bindings are typed and shareable
- [x] A foreign call cannot stall the collector: every call to an extern
      outside std (and through an `extern fun` pointer) runs in a safe
      region; a callback from C leaves it for the Veles code, and a thread
      Veles did not start is registered on its first callback (2026-09-27,
      TestForeignCallDoesNotStallCollection)
- [ ] Suspension-aware bindings: a blocking C call runs on a helper thread
      and parks the task (today it occupies one worker; the others go on)

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
- [ ] Cancellation propagation API documented as a surface, not just D3's discipline
- [ ] Task-local values (request id, trace context) that follow `async`
- [ ] Bounded channels with backpressure; `select` over several waits
- [ ] Deadlock/lost-wakeup detection in debug builds
- [ ] Blocking-call detection: a syscall on the executor thread is a warning

### 1.4 Type system and syntax

- [ ] Attributes with typed arguments (needed by 1.1) → D51
- [ ] Coherence/orphan rules for `implement` (spec: "TBD")
- [ ] `Default` values for generics without a hand-written implement
- [ ] Integer overflow policy per build profile (checked in debug, wrapping
      explicit with `+%`) — verify it is actually what ships
- [ ] `const` evaluation: what may appear in a `const` (today: literals)
- [ ] Compile-time assertions (`static assert`) for wire-format invariants
- [ ] Better inference for empty collection literals (`notes_to_change` §8)
- [ ] Stable ABI story for `.vs` packages: none needed while source-only,
      but say so

---

## 2. Safety

- [x] `readRequest` limits: `http.Limits` — request line, header line, header
      count (lines, not map entries), header bytes, body bytes, and three clocks
      (header, body, idle). A byte ceiling answers 414/431/413 and closes, a time
      ceiling 408; nothing reaches a handler, and a body is never assembled to
      discover it was too big
- [x] Every read from the network is bounded by a caller-given max: `readLine(max:)`
      has no default and throws `net.TooLong`, so the unbounded call does not compile;
      `read(max)` and `readExact(n)` were already caller-bounded, and http checks
      `Content-Length` against `bodyBytes` before the read
- [ ] Panic-freedom analysis on leaf functions (also a perf win)
- [ ] `unsafe` blocks audited: each std use has a comment saying why it is sound
- [ ] Bounds checks stay on in release; a profile that removes them is opt-in
      and loud
- [ ] Integer conversions (`as`) between widths: truncation is explicit
- [x] Constant-time comparison primitive in `std/crypto`: `Digest`'s `==`, and
      `crypto.equalBytes` for raw bytes. Structural `==` on `List<u8>`
      short-circuits, so a MAC is compared as a `Digest`, never as bytes (D59)
- [ ] Secrets: a `Secret<T>` wrapper that does not `Display`, does not derive,
      and zeroes on collection
- [x] Path traversal: `path.within(root, p)` over a lexical `path.clean`,
      both separators on every platform, a `\\server\share` root kept.
      Writing it found a **real hole**: `http.files` split the request on
      `/` only, so on Windows `GET /static/..\main.vs` passed the `..` check
      and served a file from outside the root (reproduced: 200 with the
      file's contents). `files` now checks `within` *and* refuses any `..`
      segment on either separator; `examples/httpd` probes `..`, `..\` and
      `%2e%2e` (2026-09-25)
- [~] Fuzzing: `examples/fuzz` (seeded, `fuzz [iterations] [seed]`) checks
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
      the printer already checks against (2026-09-25). Still open: the HTTP
      request parser (needs an in-process connection)
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
- [ ] `StringBuilder` growth policy and a `reserve`
- [ ] `List<u8>` ↔ socket: writev/readv, no intermediate copies
- [ ] I/O: `poll` → `epoll`/`kqueue`/IOCP when connection counts justify it

### 3.2 Compiler

- [ ] Release profile: `-O2`, LTO, no frame pointers unless profiling
- [ ] Panic-freedom analysis (also §2)
- [ ] Devirtualisation of trait objects with one implement in the program
- [ ] Incremental compilation or at least per-module caching of IR
- [ ] Compile-time budget: `veles build --timings`

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

- [x] Signals: `os.shutdownSignal(): os.Signal` (D68, 2026-09-26; `os.onSignal` rejected) + `os.raiseSignal` for tests. Real console/POSIX delivery not exercised by the suite (no console in the test harness); the path from the recorded signal on is (`examples/shutdown`)
- [x] Graceful shutdown in `http.serve(..., stop:, grace:)`: stop accepting, close idle keep-alive, drain with `connection: close`, cancel after `grace` (2026-09-26)
- [ ] Panic in a task: stack trace with symbol names (DWARF unwinding is on
      the remaining list)
- [ ] Crash report: what a production panic prints and where
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

- [ ] Levels, structured fields, JSON or key=value output
- [ ] Request-scoped logger (task-local, 1.3)
- [ ] No interpolation cost when the level is off

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

- [ ] `std/config`: typed env parsing, all missing keys reported at once
- [ ] `std/compress`: gzip/deflate via zlib binding
- [~] `std/os`: `hostname`, `pid`, `tempDir` done (2026-09-25); `shutdownSignal`/`raiseSignal` done (D68)
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
- [ ] Collections: queue/deque, priority queue, `Set` ops complete
      (`notes_to_change` §9)
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
- [ ] `veles test`: coverage, `--filter`, parallel, test timeouts
- [ ] `veles bench`
- [ ] LSP: rename, find references, code actions ("add `@json(skip)`",
      "write `implement Json`"), inlay hints for inferred `suspends`/`throws`
- [ ] Diagnostics: every error names the fix, with a `docs/` link
- [ ] `veles doc`: rendered API docs from `///`
- [ ] Package registry / MVS (on the remaining list)
- [ ] Lockfile and reproducible builds
- [ ] `veles new server` template with logging, health, graceful shutdown wired

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

Each is answered in a conversation with options, examples, edge cases and
pros/cons before anything is built; the answer becomes a spec entry.

1. ~~Derivation mechanism~~ — decided, see §10.
2. ~~Derivation target~~ — decided, see §10.
3. ~~Field attributes~~ — decided, see §10.
4. ~~Sealed-variant discriminator~~ — decided, see §10.
5. ~~Nullable fields~~ — decided, see §10.
6. ~~Enum representation~~ — decided, see §10.
7. ~~Encoder/Decoder surface~~ — decided, see §10.
8. ~~Decode error model~~ — decided, see §10.
9. ~~Generic structs / braceless empty implement~~ — decided, see §10.
10. ~~First derivable set~~ — decided, see §10. (Orphan rule: none needed — D17 already allows any implement anywhere, one per pair program-wide.)
11. ~~`Codable` via supertraits~~ — decided, see §10.
 12. ~~Function type parameters after the name~~ — decided, see §10.
13. ~~`implement` → `implement`~~ — decided, see §10.
14. **A check that a value implements a trait** (`x is Display`): trait-object RTTI (Go-style, a per-type trait table in every box) vs a compile-time `T implements X` in generics (per stencil, no runtime cost) vs neither. Deferred; to be designed as its own decision with the costs. *Open.*
8. ~~FFI design~~ — decided 2026-09-26 (D67), see §10.
9. ~~Executor threading model~~ — decided 2026-09-26 (D66), see §10.
10. User-definable derivation (phase 2, once the compiler-known set is proven). *Not yet asked.*
11. ~~Arithmetic operator traits~~ — decided 2026-09-26 (D71), see §10. Was: (`Addable`/`Subtractable`/… so `a + b` works on a `Duration`, a `Timestamp`, a vector, a money amount). D60 deliberately did not take it: the five operator traits today are about *comparison and text*, and adding arithmetic ones raises overflow, mixed operand types (`Timestamp + Duration` is not `Timestamp + Timestamp`) and whether `+=` follows. `Duration.plus`/`minus`/`times`/`dividedBy` are named so that such a trait could adopt them. *Not yet asked.*
12. ~~How a `Duration` goes on the wire~~ — decided 2026-09-25, see §10.

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
| 2026-09-25 | A `Duration` on the wire (§9.12) | **`"90.5s"` by default, any of five on request** (user: "90.5s, but we need to be able to convert to different"). `DurationStyle { Seconds, Iso8601, Text, Nanos, Millis }` is the format's policy, like `EnumStyle`. Spec D60 addendum. Rejected: ISO 8601 by default (Go/Python stdlib do not read it), seconds as a number (precision). |
| 2026-09-25 | Identifiers and invisible characters (notes #21) | **UAX #31** (user's choice over the recommended fixed list): identifiers are `XID_Start XID_Continue*` plus `_`, so a bidi control, a zero-width character or a no-break space is no longer an identifier byte — the Trojan-source case (CVE-2021-42574) becomes a diagnostic. Costs Unicode tables in the lexer, and in the self-hosted one. |
| 2026-09-25 | Changing a by-value struct parameter (R20 follow-up 1) | **A warning**: a function that assigns a `var` field of a value-struct parameter, directly or through a method that writes `self`, is told the caller never sees it, with `*T` or returning the value as the fixes. R20's rule (`val`/`var` govern rebinding only) is unchanged. Rejected: Swift's immutable parameters (changes R20). |
| 2026-09-25 | Guard binding (I1) | **Let-else plus `??` on Result** (user, after `veles-guard-design.md`): `val x = r else { e => ... }` binds or leaves (Result, nullable, variant pattern; one-statement `else return` allowed); `r ?? fallback` / `r ?? { e => ... }` is `?:` for a Result, and each operator on the other kind is an error with a fix that swaps it. Spec D61. Rejected: widening `?:` to Result (the recommendation), a `Fallible` trait (its own decision, later). |
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

## 11. Known limitations to revisit

- A panic std raises for a caller's misuse (`xs.swap(0, 7)`, `chunked(0)`)
  reports the std line (`at std/prelude/list.vs:408:27`), not the caller's
  (D64). Fix if it matters: a `#[track_caller]`-style attribute that passes
  the call site's location down.
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
