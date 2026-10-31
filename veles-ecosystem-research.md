# Veles foundations and first-party libraries

Research backlog, 2026-10-22. Not a language specification or an implementation
plan. No names, signatures, release order, or new semantics here are approved.

User direction: "most of this is needed as the integral part of the language."
The user asked to preserve the valuable findings for expansion, then investigate
"not std but a library provided by us" that makes Veles desirable to use.
This endorses the broad direction, not every previous suggestion individually.

## 1. Boundaries and status

- Language: semantics and syntax; changes require a design decision.
- Runtime/compiler/tools: the implementation and everyday development loop.
- Standard library: stable, broadly useful building blocks shipped with Veles.
- First-party packages: optional, independently versioned libraries maintained
  by the Veles project. Framework policy and specialist dependencies belong here.
- Applications/templates: working products that demonstrate and exercise those
  packages; not substitutes for a supported library contract.

"Integral" means a coherent supported experience, not necessarily a keyword,
compiler intrinsic, std module, or one inseparable release artifact.
The current spec and checklist override historical research and README summaries.
This document records repository findings, not fresh compiler or performance tests.

## 2. Foundations to expand

Status: **existing** means already specified, built in part, or tracked elsewhere;
**proposal** means a candidate, not a newly accepted rule. Costs are qualitative.

| ID  | Topic                                    | Status and next investigation                                                                                                                                                                                                        |
| --- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| F01 | Source debugging                         | Existing S8: line tables, locals, stepping and breakpoint behavior on Windows/Linux/macOS; separately design inspection of suspended frames and logical async call chains. High cost.                                                |
| F02 | Module interfaces and incremental builds | Existing S12/E8: materialize inferred errors/effects, deterministic interfaces, cache invalidation, dependency and toolchain fingerprints; measure warm build/editor latency. High cost.                                             |
| F03 | Explain effects and costs                | Proposal: editor/CLI explanations for suspension, allocation, escape, boxing, large copies and specialization, including originating call chains. Distinguish facts from optimization estimates. Medium/high cost.                   |
| F04 | Deterministic concurrency tests          | Existing research I11: virtual time plus quiescence, isolated task trees, seeded schedule variation and fake I/O. Real sockets/FFI cannot automatically be made deterministic. High cost.                                            |
| F05 | Resource and task safety                 | Existing checklist limits: interprocedural escapes, values derived from resources, mutable receiver escaping as immutable List, and cancellation/cleanup ordering. Fix specified guarantees before widening APIs. High cost.         |
| F06 | Higher-order effects                     | Existing D116 with limits: nested callbacks, stored callbacks and trait dispatch; improve precision without requiring novices to learn effect rows. High cost.                                                                       |
| F07 | Built-in operations and traits           | Existing open I6: equality/hash/display bounds should agree with the operations a type actually supports, with explicit exceptions such as Secret. Design required. Medium cost.                                                     |
| F08 | Fallible lazy iteration                  | Existing D42/D46 gaps and streams research: terminal errors, consumer errors, early close, cancellation, source/cursor identity and repeated consumption. Settle before generator spelling. High cost.                               |
| F09 | Generators                               | Existing I1 research: compare pull state machines and internal iteration on recursive walks, tokens, pages and zip; pin frame costs, recursion and cleanup. Not implied by cede. High cost.                                          |
| F10 | Exhaustive patterns                      | Existing limitation: nested usefulness/exhaustiveness, missing-case examples, guards and public sealed-type evolution. Medium/high cost.                                                                                             |
| F11 | Bounds facts and contracts               | Existing D62 plus open I12: first improve range/membership/loop facts and helper summaries; only then compare general preconditions. Never silently remove release safety checks. Medium/high cost.                                  |
| F12 | Compile-time composition                 | Existing D113 limits: errors, sealed values, derived functions, diagnostics, evaluation budget and reproducible floating point; reuse ordinary semantics. Medium/high cost.                                                          |
| F13 | Reflection and derives                   | Existing D118 direction: user-defined derivation, expansion inspection, source spans, deterministic bounded evaluation and invalid-schema errors. High cost.                                                                         |
| F14 | Small consistency repairs                | Existing findings: irrefutable struct destructuring, generic diagnostics, extension imports, annotation arguments and actionable fixes. Resolve conflicts with approved decisions before migration. Small/medium cost.               |
| F15 | Allocation constraints                   | Proposal: allocation reports first; investigate opt-in checked constraints covering callees, boxing, coroutine frames, initialization and FFI. A failed proof is not proof that a call allocates. High cost.                         |
| F16 | Reusable byte storage                    | Existing reactor/I/O limits: read-into, range writes, scatter/gather, buffer reuse and fewer conversions/copies. Design invalidation and ownership explicitly. Medium/high cost.                                                     |
| F17 | Views and retention                      | D196 immutable List sharing is built; string views already exist. Investigate retention diagnostics/detaching vocabulary. Mutable windows remain a separate aliasing design. Medium/high cost.                                       |
| F18 | Arenas and scratch allocation            | Existing open I8: compiler phases and request scratch; prove non-escape or use handles/runtime checks. Raw pointers alone cannot make bulk free safe. High cost.                                                                     |
| F19 | Data-oriented containers                 | Existing open I8: BitSet, interning, ordered maps/sets and small inline collections; choose by real compiler/server workloads. Medium cost.                                                                                          |
| F20 | SIMD                                     | Existing D181 direction: target dispatch, scalar fallbacks, alignment and tails; measure on multiple architectures. Do not describe approved vectors as a newly missing feature. Medium/high cost.                                   |
| F21 | Runtime predictability                   | Existing F18/concurrency limits: GC platform parity, heap ceiling/metrics, retained memory, frame costs, fairness and tail latency. Report throughput and p99 separately. High cost.                                                 |
| F22 | Optimization transparency                | Proposal: inlining/specialization/copy reports; compare idiomatic adapters with loops including build time and binary size, not just runtime. Medium/high cost.                                                                      |
| F23 | Cancellation outcomes                    | Existing Q45: operation-level token completion is open; preserve the distinction from task cancellation. Compare typed error, outcome and unchanged behavior. Design required.                                                       |
| F24 | Concurrency composition                  | Existing limits: first-success racing, async adapters, worker limits including retained ordered results, placement independent of ownership. Public names remain open. Medium/high cost.                                             |
| F25 | Streams/state/events                     | Partially selected direction, not implemented: cold/hot, repeatability, terminal errors, buffering, replay, overflow, scope ownership, sharing/restart and upstream-only recovery. High cost.                                        |
| F26 | Coherent toolchain                       | Existing tooling with gaps: installation, pinned toolchains, offline/reproducible builds, workspaces, dev-dependencies, migrations, watch, local searchable docs, coverage, benchmarks and effect-aware API diffs. Medium/high cost. |
| F27 | Learning and acceptance                  | Existing standing direction: beginner path postpones pointers/executors/advanced traits; maintained CLI, service and binary-tool applications; observe new users and measure confusing diagnostics/time to first result.             |

### 2.1 Standard-library candidates, not new approvals

- Everyday applications: finish CLI, structured child processes, platform
  directories, terminal basics and file watching on the approved OS foundation.
- Data interchange: URL/query handling, MIME, CSV, ZIP/TAR, complete typed TOML;
  streaming, limits and malformed-input diagnostics are part of completeness.
- Testing: fixtures, unique temporary resources, property generators/shrinking,
  snapshots, table cases, virtual time and user-facing benchmarks.
- Demand-led: SQLite integration, WebSocket/SSE, HTTP/2, database cursors,
  decimal/big integers, Unicode normalization/graphemes and IANA zones.
- Security/platform: keychain access, signatures, and approved hardware bindings;
  evaluate maintained native implementations rather than inventing cryptography.
- Keep framework policy outside std even when the underlying protocol or OS API
  belongs in std. A SQLite driver and an application migration framework need not
  have the same ownership or release policy.

### 2.2 Questions to carry forward

1. Which safety gaps contradict the specified contract, and which require a
   stronger contract? Separate repairs from design changes.
2. What is guaranteed allocation-free versus merely optimized that way?
3. Can resource lifetimes be checked without exposing general lifetime syntax?
4. How do fallible iteration, streams and generators share a small protocol?
5. Which effects belong in public API compatibility checks?
6. What is the smallest useful reflection surface that removes compiler magic?
7. What developer-facing profiling/debugging data should every package expose?

### 2.3 Constraints and rejected shortcuts

Keep tracing GC as the default. Do not treat a memory-model rewrite, classes,
unrestricted conversions, broad overloading, general macros, implicit dependency
injection or full algebraic handlers as approved by this research. Existing
rejections remain in force. Low-level escape hatches must not silently weaken
the guarantees of ordinary code. Do not use task-plus-channel per sequential
stream/generator operator. Do not equate a microbenchmark win with predictable
production latency.

## 3. First-party package investigation

The question is not which module names to add. It is which complete workflows
would make a programmer choose Veles despite its young ecosystem.

Candidate product families to evaluate:

| ID  | Family                               | Initial user outcome                                                                                                                     | Main risk                                                                             |
| --- | ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| L01 | Application/backend framework        | Build a typed API with validation, database access, auth integration, jobs and observability without assembling incompatible libraries.  | Becoming an enormous framework before the pieces are reliable.                        |
| L02 | Reactive desktop applications        | Build a native application or a webview shell with typed native calls, managed subscriptions and cancellation-safe background work.      | Three-platform packaging, accessibility and renderer integration.                     |
| L03 | CLI/TUI application toolkit          | Ship useful terminal applications with forms, tables, progress, subprocesses and one executable.                                         | Unicode, terminal differences and packaging are real work.                            |
| L04 | Data and document pipelines          | Stream logs/CSV/JSON/database rows, transform with bounded concurrency, produce reports.                                                 | Hidden materialization, buffer retention and uncontrolled memory.                     |
| L05 | Device and automation toolkit        | Discover/connect devices and compose sensor/event streams, with simulators for tests.                                                    | Platform permissions, hardware variance and cancellation behavior.                    |
| L06 | Workflow engine                      | Persist and resume jobs, retries, timers and human approvals across process restarts.                                                    | Exactly-once claims and incompatibility between persisted state and upgraded code.    |
| L07 | Scientific/image/media compute       | Typed arrays, SIMD-friendly kernels, image operations and native-library bindings.                                                       | Competing with mature Python ecosystems without equivalent library coverage.          |
| L08 | Testing/simulation toolkit           | Test entire applications against fake time, filesystem/network/device backends and replayable failures.                                  | Overpromising determinism across external I/O and native libraries.                   |
| L09 | Game and interactive toolkit         | Input, audio, assets, ECS and a maintained renderer binding.                                                                             | Engine scope can dominate the language project.                                       |
| L10 | Local-first application kit          | Embedded storage, migrations, reactive queries, offline state and explicit conflict resolution.                                          | Sync correctness, schema evolution and multi-device security.                         |
| L11 | Typed web/full-stack integration     | Generate a TypeScript client from a Veles API; validate at boundaries; stream events and errors without a second hand-maintained schema. | No browser/JS target should be implied; code generation is a separate feasible route. |
| L12 | Deployment and operations kit        | Build/package/release native services and CLIs with config, health, telemetry and graceful shutdown.                                     | Supporting every infrastructure provider rather than stable portable outputs.         |
| L13 | Native inference and model workflows | Embed a model or call a hosted model, validate outputs and execute explicitly authorized tools with bounded resources.                   | Model compatibility, GPU packaging, untrusted output and native memory/thread pools.  |
| L14 | Language and developer-tool toolkit  | Build parsers, formatters, language servers and incremental asset/build tools in Veles.                                                  | Exposing unstable compiler internals or building a generic engine without users.      |

### 3.1 Evaluation criteria

- A useful first release with a small, named audience and a working application.
- A concrete advantage from Veles's typed errors, inferred suspension, structured
  concurrency, value types, derives or native interop; not only fewer lines.
- Escape hatches into the underlying libraries without bypassing safety silently.
- Windows, Linux and macOS support where the product promises it; explicit
  feature matrices for unsupported capabilities rather than pretend parity.
- Maintainer cost, security exposure, native dependencies, licensing, binary
  size, build time and release/update burden.
- Optional dependencies and independent versions; importing a parser must not
  bring a UI renderer, database or telemetry exporter into the program.
- Runnable examples, diagnostics, testing fakes, cancellation/cleanup behavior,
  migration policy and performance measurements belong to the product.
- No assumption that all fourteen families should be built by the core team.

### 3.2 Typed services and web integration: L01 + L11

**Audience and attraction:** small teams building an API, background importer
and TypeScript frontend with one authoritative wire contract and a native server.
The product is an integrated development loop, not another routing library.

**First useful release:** routing, typed request/response decoding, validation
with field paths, explicit public error mapping, OpenAPI and a generated TS
client, one supported database with migrations and transaction helpers, request
traces, graceful shutdown and in-process integration tests. Run locally without
a hosted account. Existing std HTTP/DB/OTel are starting points, not evidence
these framework contracts already exist.

**Why Veles:** inferred suspension keeps handlers/helpers ordinary-looking;
typed errors can make boundary mapping auditable; structured tasks can bound
request-owned work. Independent jobs need application ownership. Derives may
eventually remove schema duplication; explicit schema/code generation is a
viable earlier route that need not wait for new language features.

**Controls and traps:** retain raw HTTP/SQL access. Expose body/response limits,
timeouts, bounded queues and pools. Never automatically expose internal errors
or every struct field. Define large integers, nullable values, timestamps and
tagged variants on the JS wire; generated types do not replace runtime checks.
Cancellation cannot undo a committed external effect. Auth, permissions,
CSRF/CORS and rate limits need review; JWT support is not an auth system.

**Acceptance application:** an issue tracker with an import job, transactional
updates, progress events and a generated browser client. Test disconnects and
DB timeouts; measure builds, startup, throughput, p99 and memory against Go.
Encore demonstrates the endpoint/database/test/trace/deploy workflow [S1].
A later server-driven HTML option could help connected admin tools: LiveView
uses server state, events and HTML diffs [S2]. This is neither a Veles browser
compiler nor an offline client. Reconnection and per-session memory add scope.

### 3.3 Local-first desktop applications: L02 + L10

**Audience and attraction:** personal tools, document apps, data browsers and
device dashboards with native background work and local data ownership.

**First useful release:** webview shell around a Veles process, typed IPC,
capability-limited native commands, SQLite-backed storage/migrations, reactive
queries, progress, dialogs and Windows/Linux/macOS packaging. Start offline on
one device; cross-device sync is a separate milestone, not a database feature.

**Why Veles:** UI-owner scopes can own subscriptions and cancel obsolete work;
native tasks can index, parse and access devices without a JS backend. The
selected streams direction is relevant but not built. Use existing task/state
contracts or wait for decisions; do not invent a second stream protocol here.

**Controls and traps:** the frontend remains HTML/JS/TS with its own tooling.
UI-thread affinity, IME, accessibility, focus, menus, webview differences,
signing/notarization and secure updates are release requirements. OS webviews
are runtime dependencies. Untrusted content must not inherit native-command
access. GC does not solve native callback/resource lifetimes.

For sync, evaluate an existing CRDT such as Automerge [S4], not a home-grown
generic merge engine. Its C API makes a binding plausible, not proven. Specify
identity, authorization/revocation, encryption, schema migration and compaction.
Convergent document changes do not automatically preserve booking limits or
account balances. A truly native widget/custom GPU UI is an alternative to
compare, not another initial renderer to promise.

**Acceptance application:** local document search with cancellable indexing;
later, two-device disconnected editing with an explicit conflict policy.
Measure idle RSS, startup, input latency, large results and cleanup. Tauri's
shell/IPC/plugin/bundler separation is a useful reference [S3].

### 3.4 Terminal applications and automation: L03

**Audience and attraction:** replace fragile scripts with distributable native
tools, then grow into interactive applications. This is the smallest plausible
route to useful third-party programs.

**First useful release:** typed command/config/environment integration, prompts,
completion, tables, progress, fuzzy selection, subprocess pipelines and widgets.
Separate command logic from rendering; support redirection and stable machine
output. CLI parsing stays on its approved foundation; interaction policy is
optional. Structured tasks can coordinate processes/output with explicit limits.

**Acceptance application:** parallel test/build runner with an interactive
failure browser and CI JSON output. Verify restoration after failure, resize,
cancellation, Unicode width and non-TTY execution. Ratatui separates rendering
and backends, with friendly application entry points [S5]; its widgets do not
own all input handling. A Veles toolkit still needs that integration.

### 3.5 Data, document and native compute workflows: L04 + L07

**Audience and attraction:** ingest files/APIs, enrich records concurrently,
query locally and produce reports without Python/native glue/separate services.
This is a niche, not a claim to replace Python's scientific ecosystem.

**First useful release:** typed CSV/JSON-lines/row pipelines, compression,
HTTP/file sources, bounded transforms, batched writes, progress and row-level
diagnostics/quarantine. An optional DuckDB C binding can supply analytical SQL
and formats instead of a new dataframe engine [S6]. Document/image extraction,
search and rendering can be separate demand-led adapters with explicit licenses
and malformed-input limits.

**Why Veles and controls:** native custom transforms, typed failures and bounded
I/O in one program. Expose batches, ordering, queue byte budgets, native thread
limits and spill behavior. Lazy-looking joins/sorts may materialize input.
SIMD/buffer reuse can help later, but do not guarantee zero-copy native calls.
Test a log importer on data larger than its working-memory budget; measure RSS
including native allocations. Bind mature numerical/media engines first.

### 3.6 Devices and host automation: L05

**Audience and attraction:** lab equipment, desktop/device utilities, gateways
and personal automation. Compose discovery, reconnect, readings, UI and storage
instead of assembling callback-driven vendor SDKs.

**First useful release:** a narrow device/platform matrix: serial plus one
approved transport, typed commands/readings, reconnect policy, record/replay
and simulators. MQTT, HID, cameras, MIDI and industrial protocols are optional
adapters, not simultaneous promises. Allow raw frames and explicit buffering.

**Acceptance application:** a device-logging dashboard that also runs against
a simulator. Verify permissions, unplug during I/O, backoff and shutdown on each
promised OS. Native gateway support does not imply microcontrollers, bare metal
or hard real-time. Structured tasks help ownership, not timing guarantees.

### 3.7 Testing and simulation: L08

Grow alongside the chosen application family: fake clocks, seeded generators
and shrinking, fake HTTP/filesystem/device adapters, failure injection and
readable traces. Start with cooperative fakes; runtime scheduler controls need
separately designed hooks (F04). This can be optional first-party infrastructure
even when basic test syntax/support belongs to std.

The attraction is a reproducible seed/event trace, not guaranteed replay of
arbitrary native schedules. Redact secrets and personal data in recordings.
Keep real-provider integration tests: fakes and applications can agree and both
be wrong. A large generic simulation platform is not the first milestone.

### 3.8 Durable work: L06

Start with persistent jobs: transactional enqueue/outbox, bounded workers,
retry policies, idempotency keys, leases and inspection. Prove crash recovery,
duplicate delivery and schema upgrades before promising workflows. Do not
automatically retry arbitrary failures or non-idempotent external effects.

A full engine additionally needs durable history/state, replay rules, timers,
versioning and recovery semantics. Temporal shows why this differs from ordinary
coroutine suspension [S7]. Serializing raw Veles frames couples persistence to
compiler layouts and does not solve external effects. Prefer an existing-engine
integration or explicit persisted state machines first. No blanket exactly-once
guarantee. High value, very high implementation and maintenance cost.

### 3.9 Native inference and model workflows: L13

**Audience and attraction:** private desktop inference, classification/search,
document processing and small model-backed services with native compute and
ordinary application code in one language.

Start with one maintained runtime binding, typed tensor metadata, explicit
sessions/resources and a complete local semantic-search application. ONNX
Runtime has a C API with tensor/session execution, providers and thread/allocator
controls [S8]. Models, tokenizers, preprocessing, dynamic shapes, device transfers
and redistribution rights still require integration. Native/GPU memory is outside
the GC heap budget.

A separate hosted-model adapter can provide streams, validated structured
responses, fake providers and explicit tool dispatch with cost/time/concurrency
limits. Output is untrusted: schema checks are not factual correctness or
authorization. Require application capability checks, not unrestricted autonomous
execution as a default. Cancellation may not immediately stop a remote charge
or native kernel. This is more useful than a generic vendor HTTP wrapper.

### 3.10 Other families worth retaining

- **Games (L09):** window/input/audio/renderer bindings, assets and a small 2D
  application first. Compare existing-engine integration before owning an ECS;
  do not promise an AAA engine/editor. Frame pacing and GC tails matter.
- **Deployment (L12):** config, signals, health/readiness, telemetry, portable
  artifacts and CI templates on std OS/log/OTel. Include shared libraries,
  signing and SBOMs; native does not imply one static file. No required cloud
  account or provider lock-in for local use.
- **Developer tools (L14):** parser combinators/generator integration, syntax
  trees, source maps, diagnostics, LSP helpers and incremental file/task graphs.
  A formatter or schema compiler is a useful first application and exercises
  self-hosting. Do not publish unstable bootstrap compiler internals as an API.

### 3.11 Recommended investment, not a new build order

1. **First adoption target: L01 + L11.** Leverage existing HTTP/DB/JSON/OTel;
   compete on the complete typed loop. API plus generated client before a UI
   engine. These foundations still have tracked gaps.
2. **Smallest useful release: L03.** Native developer tools need no new renderer,
   browser target or distributed protocol; they also prove distribution.
3. **Distinctive showcase: L02 + local L10.** Native work plus reactive local
   data and a familiar frontend. Sync and a second renderer remain separate.
4. **Shared support: L08 and narrow L12.** Grow from the first applications,
   strengthening testability and deployment rather than expanding checklists.
5. **Demand-led niche: choose L04, L05 or L13.** Let early users and a working
   prototype choose; do not staff all three before validating demand.
6. **Defer broad engines.** Workflows, generic sync, native UI, comprehensive
   games/numerics need dedicated maintainers or partnerships. Bindings first.

Rankings are qualitative judgments, not measured demand or delivery estimates.
No prototypes or comparative benchmarks were built in this research. The existing
plan remains the execution authority. Interview target users and maintain a real
application before treating ecosystem breadth as a release criterion.

### 3.12 What "provided by us" should guarantee

- Optional packages with owners, support tiers, compatible compiler versions,
  security reporting and upgrade policies. Experimental is not stable.
- Independent versions and tested application templates using ordinary,
  reproducible dependencies; no hidden global plugins. Track F26 package/release
  gaps rather than assuming them solved.
- A friendly umbrella entry point and reusable smaller packages below it.
  Importing service logic must not bring desktop/native inference dependencies.
- Shared configuration, validation, public errors, cleanup, cancellation, time,
  tracing and secret-handling conventions where semantics actually match.
- Native-boundary contracts for buffers, GC roots, callbacks, thread affinity,
  blocking offload, close/retain, cancellation and upstream ABI versions.
  Bindings include binary acquisition, licenses and security updates.
- Plain Veles with explicit dependencies first. Schema tools/compiler hooks need
  separate decisions. Inspectable generation and low-level access remain exits.
- Real acceptance applications and metrics: managed plus native memory, builds,
  startup and latency tails. No performance promises from syntax alone.

### 3.13 Primary-source comparisons

Reviewed 2026-10-22. Sources establish other projects' workflows, not Veles
compatibility, measured performance or approval to implement their APIs.

- S1: [Encore API tutorial](https://encore.dev/docs/ts/tutorials/rest-api):
  endpoint/database/migration/test/trace/local-deploy integration.
- S2: [LiveView introduction](https://phoenix-live-view.hexdocs.pm/welcome.html):
  server-owned state, events, initial HTML and persistent diff updates.
- S3: [Tauri architecture](https://v2.tauri.app/concept/architecture/):
  OS webview, native IPC, separate plugins/tooling/bundling.
- S4: [Automerge introduction](https://automerge.org/docs/hello/):
  CRDT documents, network-independent core and C bindings.
- S5: [Ratatui documentation](https://docs.rs/ratatui/latest/ratatui/):
  terminal lifecycle, buffered rendering and backend separation.
- S6: [DuckDB clients](https://duckdb.org/docs/2.0/clients/overview.html):
  embedded database, C API and explicit client support tiers.
- S7: [Temporal execution](https://docs.temporal.io/workflow-execution):
  durable history, replay and command/awaitable execution.
- S8: [ONNX Runtime C integration](https://onnxruntime.ai/docs/get-started/with-c.html):
  sessions/tensors, providers, thread pools and native allocators.

### 3.14 Focused package opportunities

Follow-up, 2026-10-22: the user asked for narrower programs/functionality rather
than broad framework categories. P01-P12 are local research labels, not approved
package names or public APIs. Each row describes a reusable library plus a
small application that proves it. None of these packages is implemented by this
document; using existing std primitives does not establish package readiness.

| ID  | Specific library                   | What the programmer can build                                                                                                                                                         | First acceptance check                                                                                                                                                     |
| --- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P01 | Resumable transfer manager         | Download large artifacts with bounded parallel ranges, progress, checksums and resume manifests; upload through one explicitly supported multipart protocol.                          | Kill/restart a transfer; change the remote ETag; corrupt a chunk; never silently combine different object versions.                                                        |
| P02 | Webhook inbox/outbox               | Verify one provider's signatures, persist incoming events, deduplicate delivery and process transactional outgoing events.                                                            | Duplicate and reorder events; crash after a side effect but before acknowledgement; show where application idempotency is required.                                        |
| P03 | SQL contract checker               | Check named SQL queries against a migration-built disposable schema and generate typed parameters/results with source-mapped errors.                                                  | A renamed column or nullable result breaks the check before deployment; database-specific behavior is explicit.                                                            |
| P04 | Live local query store             | Subscribe to SQLite queries, invalidate after committed changes and cancel obsolete UI work.                                                                                          | Rollback emits no committed change; a subscribed join reacts to both tables; external writers have a documented strategy.                                                  |
| P05 | Incremental file index             | Maintain a searchable document/catalog index with watching, ignore rules, content fingerprints and resumable scanning.                                                                | Recover after event overflow and process death; handle rename, permission failure and symlink policy without infinite traversal.                                           |
| P06 | Typed API contract exporter        | Export endpoint schemas and generate a TS client with validation/error decoding, including a supported streaming transport.                                                           | Compile the TS consumer and test int64 precision, nested nullability, unknown variants and incompatible contract changes.                                                  |
| P07 | Protocol recorder and simulator    | Frame serial messages, validate checksums, correlate replies and replay captured device sessions with injected faults.                                                                | Fragment packets, lose replies, unplug mid-command and replay without hardware; redact sensitive recordings.                                                               |
| P08 | Subprocess supervisor              | Start pipelines with bounded output, progress parsing, deadlines and explicit process-tree cleanup policy.                                                                            | A child spawns grandchildren, fills stderr or ignores shutdown; verify promised cleanup on every supported OS.                                                             |
| P09 | Safe archive/package extractor     | Stream archive entries through size/path/link policies and publish staged output only after validation.                                                                               | Reject traversal, decompression bombs and unsafe links; prevent filesystem races where the platform contract promises it.                                                  |
| P10 | Image thumbnail pipeline           | Decode, orient, resize, cache and batch-process images through a maintained codec backend.                                                                                            | Bound decoded pixels/native memory; cancel a batch; test corrupt files, color policy and metadata stripping.                                                               |
| P11 | Model session and batch runner     | Run one supported model family with validated tensor metadata, explicit provider selection and bounded concurrent inference.                                                          | Reject wrong shapes; report native memory; test model-load failure and actual cancellation behavior of the selected backend.                                               |
| P12 | Schema evolution and semantic diff | Compare two wire schemas, explain compatibility changes and generate explicit migration scaffolding.                                                                                  | Distinguish adding an optional field from changing units, numeric range, defaults or variant policy; mark unknown compatibility.                                           |
| P13 | Runtime validation library         | Validate decoded external values with composable constraints, cross-field rules and path-aware issue collection; a Zod-like boundary layer, not a replacement for Veles static types. | Validate nested JSON input for a registration/API example; collect all field issues with stable paths, reject bad values without panics/coercion, and measure allocations. |

**Most useful first choices:** P08 for a small native tool, P03 + P06 for backend
adoption, P04 + P05 for desktop/local data, and P07 for a distinctive device niche.
P01 is a good concurrency showcase. P09 needs security review before a safe
extraction claim. P10/P11 should wrap mature engines, not create codecs/kernels.

Useful scope distinctions:

- P01 is not another HTTP client. It owns resume consistency, transfer scheduling,
  atomic destination replacement and progress. Resume is conditional on server
  support; download ranges and upload protocols are different capabilities.
- P02 is not a message broker or full workflow engine. Its value is a small,
  inspectable persistent state machine. Retention, replay authorization and
  signature-secret rotation matter as much as the happy path.
- P03 is not a mandatory ORM or compiler SQL keyword. Start with a separate
  generator/checker, one SQL dialect and a disposable database; prepared queries
  and explicit SQL remain accessible. Schema checks cannot prove query cost or
  that the production schema matches the checked migration set.
- P04 is not general CRDT sync. Start with writes through the store and explicit
  query dependencies; expand to native hooks/external-writer detection only when
  tested. UI delivery and transaction ordering need a real contract.
- P05 is not just file watching. Watches are hints: reconcile against directory
  state, keep a recoverable index and expose freshness/partial failures.
- P12 is not automatic business-data migration. Structural compatibility cannot
  infer meaning; a currency/unit change may preserve the same representation.
- P13 is for values that already crossed a runtime boundary: Veles's type
  checker handles statically known values, while `Codable` handles structural
  decoding and `codec.Schema` describes that shape. A validation package would
  add domain constraints (ranges, lengths, patterns, cross-field rules) and
  aggregate path-aware issues. This is feasible without new syntax and belongs
  first in an optional first-party package, not the prelude; its API, package
  name, coercion policy and relationship to `codec.Problems` remain undecided.

The low-level path stays available in every package: raw SQL, frames, streams,
process handles or native sessions. High-level convenience should encode a
specific useful policy rather than hide resource/correctness costs.

## 4. Agent-friendly without agent-first language design

Research follow-up, 2026-10-22. The user asks whether Veles should accommodate
coding agents while remaining an ordinary ambitious developer-friendly language.
Recommendation: yes to excellent machine-accessible developer tools; no to an
AI-specific source language, weakened checks or model-dependent semantics.
This section proposes investigations, not approved commands or APIs.

### 4.1 Need, rationale and limits

A new language has less representation in model training, and its evolving APIs
may be confused with superficially similar Kotlin, Go, Rust or TypeScript APIs.
The amount of Veles in any particular model's training data is unknown. Provide
retrievable current facts rather than relying on memorization or promising that
a familiar keyword will make a model reliable.

The useful loop is: discover the installed version and real APIs; inspect the
relevant contract; make a small change; check; test; inspect the diff. Humans,
editors, CI and migration tools benefit from the same infrastructure. Models
can still misunderstand requirements, security, performance and business rules
after compilation succeeds. Agent support must improve evidence, not confidence
without evidence. It must remain optional and work offline without a model.

### 4.2 Existing footholds, checked locally

- [CLI](main.go) exposes check, formatter/test controls and checker dumps including
  types and effects. This inspection does not establish a stable machine schema
  or complete JSON event output for those commands.
- [LSP](lsp/server.go) supports document/workspace symbols and quick fixes.
  Diagnostics already carry range, severity, source, message, family code and
  documentation link; some carry fix edits. Guessed typo fixes are distinguished
  from preferred fixes. Code actions currently return URI-keyed text changes;
  snapshot/version protection should be designed before treating them as an
  unattended edit transaction.
- [Docs harness](docs/docs_test.go) and the existing docs policy provide a route
  to compiler-checked examples. An agent introduction must derive from current
  examples, not a second hand-maintained description of the language.

These are source observations, not fresh runtime tests or a complete tooling
audit. Reuse compiler/LSP facts rather than build a second approximate checker.

### 4.3 Concrete capabilities to investigate

| ID   | Capability                   | Example agent need and required result                                                                                                                                                                              |
| ---- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| AG01 | Version-aware API lookup     | Ask how to read rows with this installed DB package; receive exact signatures, imports, effects/errors, package/toolchain version and a tested example. Never silently substitute a newer API.                      |
| AG02 | Structured check/test events | Receive a versioned schema for diagnostics and test pass/fail/skip/timeout; distinguish no tests, cancelled runs, truncated output and infrastructure failure from success. Preserve useful human output.           |
| AG03 | Semantic queries             | Query a symbol at a source position for its resolved declaration, inferred type/error/suspension, implementations and references; report unsupported or incomplete analysis explicitly.                             |
| AG04 | Explanations with provenance | Ask why a function suspends or a value escapes; receive an originating call/location chain rather than only a top-level conclusion. Allocation predictions must say whether measured, proven or estimated.          |
| AG05 | Guarded semantic edits       | Preview a rename/fix with affected symbols, source-version/hash preconditions and applicability/guess classification; reject stale or conflicting edits and recheck after applying.                                 |
| AG06 | Compact learning material    | Offer an installed-version quick reference, task recipes and explicit differences from nearby languages; link to full contracts and compiled examples. A documentation index is useful, not proof an agent uses it. |
| AG07 | Public contract snapshots    | Export resolved public types, errors/effects, defaults and package identities so reviews can identify an API change even where source signatures rely on inference.                                                 |
| AG08 | Focused reproducible checks  | Discover relevant tests, run a chosen case with machine results and retain failing seeds/fixtures; report the selection so an agent cannot mistake a subset for full verification.                                  |
| AG09 | Optional agent adapter       | Expose selected existing queries/checks through LSP integration or MCP, using schemas and bounded results; no mandatory vendor, cloud account or model runtime.                                                     |
| AG10 | Real agent evaluation        | Measure completed task correctness, wrong-API invention, repair rounds, total time/cost, diff scope and safety regressions across several models and nontrivial tasks.                                              |

All schemas need explicit versioning, source coordinate encoding, package/target
identity, snapshot freshness and completeness. Bounded results need pagination
or retrieval pointers, not silently omitted evidence. Provide signatures first
and expand bodies/call paths on demand instead of dumping whole compiler IR
into every prompt. A CLI interface should remain useful without an MCP host.

### 4.4 Language design implications

Preserve regular syntax, orthogonal features, consistent APIs, searchable names,
predictable scoping and actionable errors because they are good language design.
Typed errors, exhaustiveness, null safety and resource/concurrency checks can
reject some incorrect generated programs as well as handwritten ones; they do
not establish business correctness.

Inference should remain ergonomic in source, but resolved public contracts and
provenance must be inspectable. Do not add mandatory annotations solely to help
an agent reconstruct facts the compiler already knows. Conversely, do not hide
semantically important behavior behind framework magic. Keep generated adapters
inspectable and distinguish generated declarations from source declarations.

Avoid speculative syntax changes justified only by token count, a model's
current preferences or training popularity. Do not add alternate AI syntax,
natural-language execution, fuzzy name resolution, missing-member invention,
dynamic fallback or disabled safety checks. No automatic conversion from a
failing test to a weakened assertion. Public language changes still follow the
ordinary decision process; agent performance is one input, not a veto on design.

### 4.5 Security boundary

Read-only semantic analysis and running project code are different permissions.
Tests, native build steps, generators and package acquisition can have side
effects. An agent adapter should default to queries, with explicit execution
authorization and host-enforced filesystem/network/process limits where needed.
Typed suspension/errors are not a sandbox, and tool annotations are not access
control. No compiler guarantee should be advertised for isolation it does not
enforce.

Treat repository comments, package docs, diagnostics and tool results as data,
not authority to override the user's instructions. Prompt injection is not
solved by a type system. Do not expose environment secrets or unrestricted shell
execution as convenience tools. Redact sensitive traces, bound work and retain
an audit trail. Check snapshot preconditions because humans and multiple agents
may edit concurrently. A thin adapter should not silently download dependencies
or apply migrations while answering an API query.

### 4.6 Measurement and first investment

Hypothesis: version-matched retrieval plus semantic/check/test tools will reduce
invented API usage and repair cycles relative to docs-only access. This is not
measured yet. A cheap first experiment uses existing CLI/LSP features plus a
curated version-matched recipe bundle, without implementing new syntax or a
large agent server.

Use a held-out set of roughly 20-30 tasks: add a paginated endpoint; repair a
resource escape; cancel an obsolete search; fix a nested nullable bug; migrate
a renamed DB column; preserve a low-level parser's allocation budget. Include
multi-file repairs, unfamiliar APIs and cases where correct behavior is refusal
or asking for missing requirements. Test on several models with equal budgets
and fixed compiler/docs snapshots; repeat trials rather than report one success.

Compare source/docs-only, existing tools, and proposed semantic retrieval when
available. Keep independent hidden tests and human review; record unsafe escapes,
disabled checks, dependencies added, changed tests and unrelated diff churn.
Compilation and visible-test pass rates alone are insufficient. Include human
developer trials so a model-specific improvement does not degrade ordinary use.
Training/fine-tuning can be considered later with licensed, versioned material;
it cannot replace authoritative live tools or guarantee correctness.

First investigate AG01/AG06 using generated existing docs, AG02 over the current
checker/test runner, and AG05 stale-edit protection. Then add deeper explanations
and an optional adapter based on measured gaps. These are research priorities,
not a replacement for the current compiler/runtime build order.

### 4.7 Protocol references

- [LSP 3.17](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)
  is an existing editor/tool integration boundary; reuse it where applicable.
- [MCP tools, 2025-06-18 version](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
  defines input/output schemas, structured results and tool errors, and warns
  against trusting annotations from untrusted servers. It is a transport option,
  not evidence that a language or generated program is correct.

## 5. Existing records

- [Spec](veles-spec.md): authoritative decisions through D197 at this review.
- [Checklist](veles-checklist.md#11-known-limitations-to-revisit): actual recorded limits.
- [Plan](veles-plan.md): existing commitments; this research does not reorder them.
- [Idioms and APIs](veles-idioms-and-apis.md): earlier I1-I14 and A1-A83 menu;
  some entries have since been decided or built.
- [Generator research](veles-research-generators.md).
- [View research](veles-research-views.md): option B became D196.
- [Streams specification draft](veles-streams-spec.md): selected directions and
  unresolved contracts, not implemented.
- [Streams research](veles-research-streams.md).
