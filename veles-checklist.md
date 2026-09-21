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
- [x] Fields that cannot be derived (functions, `Mutex`, raw pointers, handles):
      error at the use site naming the field, like the `Equatable`-key check
- [x] Private fields / constructor rule (M5): decoding is construction (an inline implement may set them; a foreign implement may not)
- [x] Partial override: hand-write `encode`, keep the synthesized `decode`
- [x] Foreign types: `implement Codable for pkg.T` at top level (private fields of a foreign type refuse construction, as for any caller)
- [x] Error model: every problem with its path; a decoder records mismatches and reads on, a nested value that cannot be built is caught by its parent, which finishes its own checks and fails once
      (`DecodeError { problems }`, `Problem { path, message }`, capped)
- [x] Performance: encoding streams straight to text, decoding reads the bytes;
      no tree except for sealed values (see the fast-path item) — not yet benchmarked (§3.3)
- [ ] LSP: hover on a derived implement shows what was synthesized (`implement.Derived` and `sema.DumpDerived` exist; the hover is not wired)
- [x] Derivable set: `Codable` (`Encodable`, `Decodable`) and `Comparable`
      (`==`, hashing, printing already structural; `Default` deferred)
- [x] Supertraits: `trait A : B + C`, transitive bounds, super check on impls
- [ ] Trait objects of a trait with supertraits (vtable composition) — refused today with a message
- [x] Parser: braceless empty `implement Trait` (body and top level); field attributes kept; formatter drops empty braces
- [x] Spec entry written (D58) with the rejected alternatives

### 1.2 Foreign function interface

- [?] C ABI FFI design (spec §7 lists it as not designed; §8 scopes it)
- [ ] `extern "C"` blocks: calling convention, varargs, callbacks into Veles
- [ ] `extern struct` layout: packed, explicit alignment, transparent wrappers
      (the "noted pressure point" under D51)
- [ ] Ownership at the boundary: who frees, GC pinning for buffers handed to C
- [ ] Panics/unwinds never cross into C; errors come back as codes
- [ ] Linking: `veles.toml` declares native libs, pkg-config on POSIX,
      the `.lib` equivalent on Windows
- [ ] Declaration files (`.d.vs`) so bindings are typed and shareable
- [ ] Suspension-aware bindings: a blocking C call runs on a helper thread
      and parks the task (needs §1.3)

### 1.3 Concurrency

- [ ] Multi-threaded executor → D35 (single-threaded today)
- [ ] Work stealing or per-thread queues — measure before choosing
- [ ] `Mutex<T>` becomes a real lock once threads exist (today a cell)
- [ ] `Atomic<T>` with real atomics and memory order
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

- [ ] `readRequest` limits: max header bytes, max header count, max body
      bytes, request-header timeout separate from idle timeout
- [ ] Every `List<u8>` read from the network is bounded by a caller-given max
- [ ] Panic-freedom analysis on leaf functions (also a perf win)
- [ ] `unsafe` blocks audited: each std use has a comment saying why it is sound
- [ ] Bounds checks stay on in release; a profile that removes them is opt-in
      and loud
- [ ] Integer conversions (`as`) between widths: truncation is explicit
- [ ] Constant-time comparison primitive in `std/crypto`
- [ ] Secrets: a `Secret<T>` wrapper that does not `Display`, does not derive,
      and zeroes on collection
- [ ] Path traversal: `http.files` already refuses `..` — make the check a
      `path.within(root, p)` primitive others can reuse
- [ ] Fuzz corpus for the HTTP parser, the JSON parser and `percentDecode`
- [ ] Resource leaks: a `Closeable` dropped without `with` is a warning
- [ ] Stack depth: recursion limit in decoders and the router

---

## 3. Performance

### 3.1 Runtime

- [ ] Threads (see 1.3) — the single largest throughput multiplier
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

- [ ] `bench/` with an HTTP hello, a JSON round-trip, a map-heavy workload
- [ ] Numbers recorded in-repo per commit that touches the runtime

---

## 4. Runtime and operations

- [ ] Signals: `os.onSignal`, `os.shutdownSignal()` (SIGTERM/SIGINT/Ctrl+C)
- [ ] Graceful shutdown in `http.serve`: stop accepting, drain with deadline
- [ ] Panic in a task: stack trace with symbol names (DWARF unwinding is on
      the remaining list)
- [ ] Crash report: what a production panic prints and where
- [ ] Static binaries; cross-compile Windows → Linux (LLVM target triple)
- [ ] Docker base image and a one-line `veles build --release --target linux`
- [ ] Environment: `os.env` exists; `os.hostname`, `os.pid`, `os.cwd`
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

- [ ] Request limits (§2)
- [ ] Chunked transfer-encoding: requests (501 today) and responses
- [ ] Streaming bodies: `Request.body`/`Response.body` as reader/writer,
      not `List<u8>`; `http.files` streams
- [ ] Middleware: `type Middleware = fun(Handler): Handler`; `router.use(...)`
- [ ] Standard middleware: recovery (D56), request-id, access log, CORS,
      timeout, body-limit, auth hook, compression
- [ ] Graceful shutdown (§4)
- [ ] Max concurrent connections with backpressure at `accept`
- [ ] `Expect: 100-continue`
- [ ] `HEAD`/`OPTIONS` defaults; `405` with `Allow` (today `PUT` → check)
- [ ] Cookies: parse and set, with `SameSite`/`Secure`/`HttpOnly`
- [ ] Forms: `x-www-form-urlencoded` (have `parseQuery`), multipart streamed to disk
- [ ] Static files: `ETag`, `Last-Modified`, `Range`, `Cache-Control`, index files
- [ ] Router: method-not-allowed vs not-found distinction, route groups,
      typed path params (`{id: i64}`)
- [ ] In-process test client: `http.call(handler, method, path, body)`
- [ ] Access log format: structured, one line, latency, bytes, request id
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

### 5.5 `std/crypto` and `std/encoding`

- [ ] base64 (std + url), hex
- [ ] SHA-256/512, HMAC, constant-time compare
- [ ] CSPRNG (`std/random` is xoshiro — not for crypto; say so in the API)
- [ ] UUID v4/v7
- [ ] Password hashing (argon2id) via binding
- [ ] JWT sign/verify (HS256, RS256/ES256 via binding)

### 5.6 `std/time`

- [ ] `Duration` type
- [ ] RFC 3339 / ISO-8601 parse and format; HTTP-date
- [ ] Zone offsets (UTC + fixed offset first; tz database later)
- [ ] Monotonic deadlines usable by `withTimeout`

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
- [ ] `std/os`: `onSignal`, `hostname`, `pid`, `tempDir`
- [ ] `std/fs`: streaming reads/writes, `walk`, atomic rename, file locks
- [ ] Collections: queue/deque, priority queue, `Set` ops complete
      (`notes_to_change` §9)

---

## 6. Tooling and developer experience

- [ ] `veles fmt` stable on every example (formatter exists — CI check)
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

- [x] `docs/18-codable-and-json.md` (chapter, attributes table, stdlib reference); `docs/19-production-servers.md` still to write
- [ ] A "deploying Veles" page: binaries, signals, Docker, health checks
- [x] Derivation chapter with the rules from 1.1 in plain language
- [ ] Security guidance: limits, secrets, TLS defaults

---

## 8. Examples (each with `expected.txt`)

- [~] `examples/httpd` upgraded: `std/json` (done: derived `Note`, a `PUT` with a 400 that lists the problems); middleware, graceful shutdown,
      limits — the reference server
- [x] `examples/codable`: every derivation rule in one program
- [ ] `examples/apiclient`: calls a JSON API over TLS
- [ ] `examples/pgnotes`: the notes API on PostgreSQL

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
8. FFI design. *Not yet asked.*
9. Executor threading model. *Not yet asked.*
10. User-definable derivation (phase 2, once the compiler-known set is proven). *Not yet asked.*

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
| 2026-09-21 | `is Trait` check | **Deferred** to its own decision (§9 item 14): a runtime check needs RTTI in trait objects, a compile-time one belongs to generics, and for a concrete type the answer is static. |

## 11. Known limitations to revisit

- Derived code calls the prelude functions `styleKey`, `childPath` and
  `joinPath` by name; a user declaration of the same name in the module
  shadows them and produces a confusing error. Fix: resolve prelude names
  from synthesized code directly (a `ResolvedFunc` node).
- `T?.decode(from)` written by hand parses as a safe call on `T`; use a
  generic (`fun decodeIt<T: Decodable>(...)`) or a field. Derived code uses a
  resolved-type receiver and is unaffected.
- `examples/tutorial/04-functions.expected.txt` is out of date relative to
  its source (three lines printed, one expected) — pre-existing, not touched.
