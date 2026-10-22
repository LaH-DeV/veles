# Veles streaming: draft specification and open decisions

Date: 2026-10-22. Status: **partially settled design; not implemented**.

This document consolidates what the user selected and what remains unknown
about collector flows, asynchronous iteration, observable state, shared
events and their relationship to channels and generators. It is a working
specification, not a declaration that the current compiler supports these
APIs. No implementation is authorized by this document alone.

The earlier [research brief](veles-research-streams.md) preserves the initial
comparisons and questionnaire. This document is the consolidated discussion
snapshot: it incorporates the answers instead of presenting every earlier
recommendation as if it were still equally pending.

## Contents

1. [Status and interpretation](#1-status-and-interpretation)
2. [Existing language and library contracts](#2-existing-language-and-library-contracts)
3. [Selected design directions](#3-selected-design-directions)
4. [Concepts and boundaries](#4-concepts-and-boundaries)
5. [Worked behavior under the selected directions](#5-worked-behavior-under-the-selected-directions)
6. [Open decisions with explanations and examples](#6-open-decisions-with-explanations-and-examples)
7. [Implementation boundaries and acceptance](#7-implementation-boundaries-and-acceptance)
8. [Question and dependency index](#8-question-and-dependency-index)
9. [Evidence and verification](#9-evidence-and-verification)

## 1. Status and interpretation

Four labels are used throughout:

- **Existing:** already specified or present in the implementation. Where
  specification and implementation differ, the difference is stated.
- **Selected:** the user selected this direction in the S1-S9 questionnaire.
  It constrains this draft, but does not settle every signature or edge case.
- **Open:** a choice the user has not settled, including details necessary
  to turn a selected direction into an implementable public contract.
- **Recommended:** the assistant's proposal, not the user's decision.

Every code block below is explicitly `text`: **proposed pseudocode**, not
compiler-verified Veles. Names such as `Flow`, `State`, `Events`, `collect`,
`asFlow`, `share`, `finish` and `mapLatest` are discussion placeholders.
Helper functions and domain variants stand for application code. A snippet
demonstrates a behavior or alternative; its inclusion does not approve its
syntax, constructor, generic parameters or error annotation.

Do not infer a new `async` function modifier, universal call-site `await`,
generator syntax, general union value type, first-class scope value or
special exception-catching mechanism from the pseudocode.

The main [language specification](veles-spec.md) remains authoritative.
No new D-number is allocated here. Open labels O1-O15 are local to this
document, not checklist Q-labels or permanent language decision IDs.

## 2. Existing language and library contracts

### 2.1 Constraints inherited from the language

| Existing decision                      | Constraint on this design                                                                                                                          |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| D5/D42: nested optionals and iteration | A nullable item is distinguishable from end-of-sequence; one loop spelling covers sync/async iteration                                             |
| D42: iteration through traits          | Async generators must interoperate with the established iteration model, not introduce an unrelated mandatory loop syntax                          |
| D45: typed error unions                | Failure composition uses the error system; this design does not add unrestricted value unions                                                      |
| D46: eager/lazy operations             | Preserve established collection/iterator meanings; its restriction on throwing lazy callbacks needs an explicit amendment if next becomes fallible |
| D47 and D147: cleanup                  | Suspending cleanup and its cancellation shielding are inherited, not reinvented by flow operators                                                  |
| D116: conditional suspension           | Ordinary functions infer suspension; sequential callbacks need not automatically become sendable concurrent work                                   |
| D146: notification primitives          | Existing Watch/Broadcast semantics cannot silently change to match Kotlin names                                                                    |

Changing one of these is possible only through an explicit design decision
and migration where relevant. A new library surface is not permission to
change the underlying language contract accidentally.

### 2.2 Mechanisms actually present

See [notification source](std/prelude/notify.vs) and
[iteration source](std/prelude/iter.vs).

| Mechanism            | Current contract                                                                                                                                          | Important distinction                                                     |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `Watch<T>`           | Holds current value/version; each copy tracks its seen version; set/get/changed; equal sets still advance version                                         | Not StateFlow equality suppression; no explicit terminal API              |
| `Broadcast<T>`       | Bounded ring; send does not suspend for slow readers; new subscriptions begin with future values; slow readers receive Lagged; close wakes/drains readers | Retention for existing readers is not replay for new readers              |
| `Subscription<T>`    | Nullable receive with typed Lagged; subscription can be closed                                                                                            | Not a complete sharing/restart abstraction                                |
| `Event`              | Waitable flag with set/reset                                                                                                                              | Not a stream of event payloads                                            |
| `AsyncIterator`      | `next(): Item? suspends`                                                                                                                                  | No declared typed terminal error on next in the current trait             |
| Lazy `mapConcurrent` | Worker-bounded admission, input-ordered output and owned child tasks                                                                                      | Owner-task failure is not a general local fallible-next/recovery protocol |

D46 intends lazy adapters to work with async iteration. The current
AsyncIterator declaration does not expose the full adapter family. Keep
that implementation gap separate from genuinely new design decisions.

The existing byte-oriented I/O Stream abstraction is not a reactive flow.
Any eventual use of the public name Stream must account for that overlap.

## 3. Selected design directions

These are the user's questionnaire selections, not inferred preferences:

| Original question | Selected direction                                                                                      | Still not settled by that selection                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| S3 errors         | Typed terminal errors plus optional per-item outcomes; upstream-only recovery/retry                     | Collector error-effect representation, fallible loop integration, close-error reporting |
| S4 state          | Preserve Watch; add state-specific semantics                                                            | Public names, equality customization, update callback contract                          |
| S5 events         | Require an explicit delivery policy; no-reader replay-only retention, offline queue separate            | Capacity accounting, reports for each loss policy, publication/close races              |
| S6 sharing        | Scope-owned eager/lazy/while-subscribed policies; minimal while-subscribed stop/restart behavior        | Ownership spelling, completed-session behavior, grace/stale replay options              |
| S7 termination    | Owner-bounded observation; domain failures as values; preserve closable Broadcast                       | Per-type finish API, final-state delivery and post-finish operations                    |
| S8 execution      | Sequential by default; explicit bounded concurrency and placement; preserve consumer executor           | Exact controls, queue accounting and cursor/collector bridge implementation             |
| S9 scope          | Staged coherent std core, followed by advanced operators; latest work cancels/joins obsolete work first | Stage boundaries, detailed time/combine/merge contracts                                 |

For S1 the user wrote:

> I would say something like collector sounds good. we would have the
> channels, collector (flow) and with generators and async generators would
> looked like async iteration, no? everything would be covered with a
> reason... WHAT DO YOU THINK ABOUT THIS STATEMENT, AM I MISSING SOMETHING ?

This is a collector-oriented preference and a request for analysis, **not**
approval of the earlier iterator-first recommendation or final protocol.

For S2 the user wrote:

> not sure, need more examples, pros and cons and why do and why not

Repeated collection, source/cursor identity and their public representation
therefore remain open. O1-O3 address them without assuming an answer.

### 3.1 Requirements following from the selected directions

- Ordinary sequential processing must not require a task/channel per operator.
- Explicit concurrent stages must bound retained and admitted work, including
  completed results waiting for ordered consumption.
- Cancellation and panic must not be consumed by ordinary error recovery.
- Recovering a source must not accidentally recover a consumer's failure.
- Early exit must release owned upstream resources and join owned work.
- State and events must remain distinct: skipping intermediate state is not
  permission to silently drop events under a lossless event policy.
- Sharing must have an owner; leaving a subscription must not intrinsically
  cancel other subscribers while they still have a valid owner.
- Existing Watch/Broadcast users keep their established behavior unless an
  explicit, separately approved migration says otherwise.

These are draft design requirements, not claims that tests or code exist.

## 4. Concepts and boundaries

### 4.1 The mechanisms answer different questions

| Concept          | The question it answers                                                | Example                                       |
| ---------------- | ---------------------------------------------------------------------- | --------------------------------------------- |
| Producer recipe  | What computation can start when consumed?                              | Fetch pages for a query                       |
| Cursor/traversal | Which position and resources belong to one consumption?                | Page 4 of one request session                 |
| Collector        | What should happen when the producer emits a value?                    | Render or save that page                      |
| Generator        | How can incremental production be written without manual cursor state? | Nested loops producing matching records       |
| Channel          | How do separately scheduled activities hand values to one another?     | Image jobs distributed among workers          |
| State            | What is the latest observable value?                                   | Current connection status                     |
| Event broadcast  | Which subscribers see occurrences?                                     | Each active subscriber receives a message     |
| Sharing          | Which consumers observe one producer, under whose lifetime?            | Two views observe one network query           |
| Cache/log        | Which past values remain retrievable?                                  | Export the exact results previously displayed |

The user's proposed split is viable. What it does not automatically supply
is a common contract for adapters, errors, early termination, hot-source
observation and ownership. Those are part of this specification, not reasons
to reject having channels, flows and generators together.

### 4.2 Independent axes

Cold/hot, push/pull, sequential/concurrent, buffered/unbuffered and
one-shot/restartable are separate axes.

A cold collector flow can run sequentially in one task: emit calls downstream
and waits for it. An async iterator can wrap a hot event subscription. A
channel-backed cold producer may create concurrent work explicitly. A
restartable recipe can still produce different values each time.

In particular:

- Cold does not mean pure, repeatable results or no side effects.
- Hot does not imply broadcast or retained history.
- Pull does not prove upstream memory is bounded.
- Suspending emit does not prove the external source can slow down.
- Send completion does not mean a subscriber finished processing a value.
- Read-only observation does not mean a frozen snapshot.
- Generators do not inherently require a background task.

## 5. Worked behavior under the selected directions

These examples explain the selected principles. Details explicitly marked
open in section 6 are not resolved by these sketches.

### 5.1 A sequential collector pipeline

```text
val results = flow(emit => {
  loop (page in 1..3) {
    val response = try fetchPage(page)
    emit(response)
  }
})

try results
  .map(response => try decode(response))
  .filter(page => page.hasOpenIssues())
  .collect(page => try save(page))
```

Without an explicit concurrency boundary, the intended trace is:

```text
fetch page 1 -> decode -> filter -> save -> return to producer
fetch page 2 -> decode -> filter -> save -> return to producer
fetch page 3 -> decode -> filter -> save -> complete
```

A suspending fetch/save suspends this task; it does not imply separate
producer and consumer tasks. Saving a page can throttle subsequent fetching.
Whether a second collect restarts this body is O2, not assumed here.

### 5.2 Terminal error versus a bad item

```text
rowsFromFile(path)
  .map(row => validateAsOutcome(row))
  .collect(outcome => recordValidation(outcome))
```

An unreadable file or broken decoder can terminate iteration through a typed
error. An invalid record can be an explicit outcome value and need not stop
later records. The caller chooses the meaning; the library must not convert
every failure into an apparently successful skipped item.

A nullable record remains data. Its null value cannot be reused as the
terminal signal for a nullable-item cursor; preserve nested optional meaning.

### 5.3 Upstream recovery and retry

```text
remotePages()
  .map(page => try decode(page))
  .recover(error => cachedPagesFor(error))
  .collect(page => try save(page))
```

Recovery at this position may handle remote/decode errors, not save errors.
A fallback error propagates with its own type. Cancellation is not a signal
to start using cached data, and a panic is not an ordinary network failure.

Retry restarts the retry operator's upstream region, not the whole application.
It does not undo earlier saves. If attempt one emits A/B and fails, attempt
two may emit A/B/C; consumers may observe A/B/A/B/C. A retryable source may
need deduplication, resume tokens or idempotent writes. Automatic infinite
retry is not implied by choosing a retry operator.

### 5.4 State is not an event history

```text
val writer = State(initial: Connecting())
val reader = writer.readOnly()
writer.set(Online())
writer.set(Online())
writer.set(Reconnecting())
```

State observers receive an initial/current value and subsequent relevant
changes. Equal values are suppressed under the chosen state semantics; a
slow observer can skip Online and see Reconnecting as its next value.
That is correct for current status, but wrong for recording every connection
transition. Use events/logging when each occurrence matters.

The reader can observe but cannot call the writer's update API. Sharing the
view must not expose mutable payload references that bypass notification or
existing Sendable/confinement requirements.

### 5.5 Event policies expose the tradeoff

```text
val messages = Events<Message>(capacity: 64, replay: 0,
                               overflow: Suspend)
val metrics = Events<Metric>(capacity: 64, replay: 0,
                            overflow: DropOldestWithLag)
```

With active subscribers, messages can make a producer wait rather than
overwrite unread values. A slow subscriber can therefore delay all readers.
Metrics can sacrifice old samples but report the loss as part of the policy.
These are proposed policy names, not existing constructors.

With no subscribers, the selected direction retains only replay. Therefore
replay zero retains nothing; emit does not wait for a reader to appear. A
job that must be handled later belongs in a queue/log, not this event policy.

### 5.6 Sharing and latest work have structured lifetimes

```text
scope {
  with shared = queryResults.share(start: WhileSubscribed, replay: 1)
  async displayTable(shared)
  async displaySummary(shared)
}
```

There is one owned producer rather than one per view. The owner's exit
stops and joins owned work. Closing the table's subscription alone does not
intrinsically end the summary's subscription; uncaught parent-scope failures
still follow the existing fail-fast rules.

For latest-work processing, obsolete owned work is cancelled and joined
before replacement begins. This bounds overlap but can delay replacement
while cleanup runs. It cannot undo an external request already accepted.

## 6. Open decisions with explanations and examples

Each entry separates the motivation, alternatives, pros/cons, recommendation,
cost and specific question. Recommendations are not adopted rules. Related
choices should be answered together after their consequences are understood.

### O1. Collector-oriented flows versus an iteration-oriented common core

**Problem.** The user wants channels, collector flows and generators to each
have a reason. All can exist. The unresolved choice is which protocol owns
the primary lazy pipeline surface and how the others interoperate.

Use the same paginated-query program for comparison.

**A. Collector-oriented flow with generator/iterator adapters.**

```text
val pages = flow(emit => {
  loop (number in 1..10) emit(try fetchPage(number))
})
try pages.map(page => try decode(page)).collect(page => render(page))
try generatedPages().asFlow().collect(page => render(page))
```

The producer invokes downstream processing through emit. This can remain
one task, with downstream suspension providing sequential backpressure.

Pros: convenient imperative producers before generators exist; familiar
Kotlin-style pipeline surface; natural event/callback adapters when their
ownership is explicit. Cons: downstream failures travel through emit; early
stop and callback lifetime need enforcement; pull conversion is not free.

**B. Iteration-oriented core with a collector-friendly facade.**

```text
val pages = generatedPages().map(page => try decode(page))
with cursor = try pages.open()
loop {
  val page = try cursor.next() ?: break
  render(page)
}
```

Consumers request values; collector methods drive the cursor internally.
Exact fallible loop syntax is deliberately not invented here.

Pros: directly builds on D42/D46; demand, break and cursor state are clear;
sequential adapters can avoid task/channel machinery. Cons: imperative
producer authoring needs a generator/builder or manual state; callback and
push sources need a bridge; generator design can become a dependency.

**C. Independent flow and iterator families, with explicit bridges.**

```text
val decodedFlow = pageFlow().map(page => try decode(page))
val decodedCursor = pageCursor().map(page => try decode(page))
```

Pros: each protocol can use its best native implementation; boundaries are
honest. Cons: duplicated operators and documentation; error/cleanup meanings
can diverge; converting in the middle of a pipeline can change memory and
task behavior. Existing eager/lazy duplication should not multiply casually.

Keeping today's primitives without a common higher-level pipeline remains
possible, but leaves the motivating composition/cleanup problem to users.

**Recommendation:** investigate A seriously in line with the user's preference,
while requiring coherent operator/error/lifetime contracts across adapters.
Do not choose a common internal representation solely for visual API symmetry.
Kotlin shows A can be efficient; C#/Rust-style async iteration illustrates B.

**Cost:** public protocol and operator ownership, generic effects, bridge
runtime needs, loops and tooling. The self-hosted compiler must retain cheap
ordinary iteration rather than acquire reactive/task overhead by default.

**Question O1:** Prefer A collector-oriented pipelines, B iteration core with
a collector facade, or C two explicitly distinct native families? Must loop
consumption remain available even when converting a collector source costs
an explicit bounded execution boundary?

### O2. What exactly does collecting twice mean?

**Problem.** A pipeline value may be a computation recipe, an already-running
session or a traversal position. If that is ambiguous, a second consumer
can repeat side effects, miss values or steal another reader's work.

```text
val query = pageFlow("open issues")
try query.collect(page => showInTable(page))
try query.collect(page => exportPage(page))
```

**A. Cold recipe: execute independently on every collect.** Each consumer
fetches from page one with independent state. Pros: reusable pipeline value;
one consumer's early stop does not exhaust another. Cons: duplicate network
work/effects; the export can see newer data than the table. A restartable
recipe is not a promise of deterministic results or idempotent operations.

**B. One-shot session: consume once, then reject reuse.** Pros: direct resource
ownership and no accidental repeat. Cons: reusable application pipelines need
a factory; the language must decide how reuse is rejected, not assume linear
types or a new ownership checker already exists.

**C. Shared producer: attach to one running session.** Pros: overlapping
consumers avoid duplicate requests. Cons: a later export sees future values
and configured replay, not necessarily all pages. The session may already
have finished. Implicit sharing also hides owner/start policy choices.

For a socket the distinction is unavoidable:

```text
with socket = try connect(address)
val incoming = socket.messages()
try incoming.collect(message => updateModel(message))
```

An existing socket cannot rerun its history. A separate connect-and-read
recipe can create a new connection per collection, but that is a different
operation with new effects. It must not masquerade as replay of the old one.

For exact table/export identity, use a retained snapshot or bounded/durable
replayable store. Neither re-fetch nor share-with-replay-one is a cache.

**Recommendation:** ordinary cold builders are recipes; resource-backed
one-shot consumption is explicit; sharing/cache remain explicit. This is
still a recommendation because the user left S2 unresolved.

**Cost:** acquisition timing, repeated/concurrent collection tests, cursor
ownership, reuse diagnostics and documentation. No generator syntax is
needed to settle this behavior.

**Question O2:** Should normal cold builders rerun per collect, while an
already-open resource can explicitly expose one-shot consumption? Or should
the initial API be cursor-only, with a factory required for every repeat?

### O3. How should recipe versus traversal appear in the public API?

**Problem.** Agreeing that two lifetimes exist does not settle whether users
need two named protocols, a factory or a mode on one universal type.

**A. Distinct recipe and traversal types.**

```text
val query = pagesFor("open issues")
with first = try query.open()
with second = try query.open()
```

Pros: signatures/hover can explain ownership and repeatability; adapters
preserve meaningful types. Cons: more concepts/generic types, and open versus
next versus collect needs careful teaching. Type separation alone does not
prove a particular recipe is pure or repeatable after external changes.

**B. Factory plus scoped one-shot handle.**

```text
val makePages = () => openPages("open issues")
with first = try makePages()
with second = try makePages()
```

Pros: reuses function values and scope ownership; small initial public
surface. Cons: operators over factories need a convention; tooling must still
explain one-shot handles; an unscoped captured handle can still be misused.

**C. One universal flow type with runtime modes.**

```text
val source = pages(mode: OneShot)
```

Pros: fewer type names. Cons: the same signature hides whether collect
restarts, attaches or rejects; more runtime surprises; generic operators
must preserve mode/lifetime information anyway. A hot/cold marker is not
by itself a complete ownership contract.

**Recommendation:** compare A and B with real application signatures; B is
a viable smaller surface, not a reason to erase the semantic distinction.
Avoid C if callers cannot discover the difference before consuming.

**Cost:** API inventory, formatter-readable signatures, hover/completion,
escaping captures, generics and reuse diagnostics. No new ownership feature
should be assumed without its own language decision.

**Question O3:** Separate recipe/traversal protocols or factory/scoped-handle
convention first? How should invalid repeated/concurrent consumption be
rejected when it cannot be proved statically?

### O4. Bridge behavior and callback-driven sources

**Problem.** Supporting loops and collectors is only useful if conversion
does not hide unbounded queues, retained callbacks or unexpected executors.

**A. Direct cursor-to-collector bridge.**

```text
with cursor = try source.open()
try cursor.asFlow().collect(value => process(value))
```

It can repeatedly request next, call downstream and close on exit. Pros:
potentially same-task and direct demand. Cons: the adapter must preserve
typed source versus consumer failures and cleanup ownership.

**B. Resumable collector-to-cursor bridge.** A producer pauses at emit until
the next pull. Pros: can preserve tight demand without a separate producer
task. Cons: requires suitable continuation/generator lowering; arbitrary
suspending callbacks and unwinding must be supported, not just a small demo.

**C. Explicit task plus bounded handoff.**

```text
with cursor = flow.asCursor(buffer: 1, execution: OwnedTask)
```

Pros: general implementation path using existing task mechanisms; boundary
is inspectable. Cons: task/synchronization overhead, possible prefetch,
Sendable constraints and cancellation races. Buffer one still allows a
producer to be holding an additional in-flight value; accounting must say so.

For an external callback API, lifecycle is an additional requirement:

```text
callbackFlow(registration => {
  val token = external.onMessage(value => registration.tryEmit(value))
  registration.onClose(() => external.remove(token))
})
```

This sketch does not approve a close-callback syntax. It means registration
must be undone even on early stop, callback-after-close must be safe, and an
unsuspendable external thread cannot call a secretly suspending emit. A
full buffer needs an explicit loss/rejection/disconnect policy.

**Recommendation:** direct bridges where proven; costly bridges explicit;
callback adapters scoped and bounded. Do not promise a universal zero-cost
bidirectional conversion before implementation/performance evidence.

**Question O4:** Is an explicit owned-task collector-to-cursor bridge an
acceptable first implementation, or must that bridge wait for resumable
lowering? Should callback adaptation be a separately named bounded facility?

### O5. Typed errors across collect, next and loop consumption

**Problem.** Typed terminal errors were selected, but the current async next
trait is not fallible. Collector execution also contains errors originating
on both sides of emit. The signatures must preserve that distinction.

```text
pageFlow()
  .map(page => try decode(page))
  .recover(sourceError => cachedPages(sourceError))
  .collect(page => try save(page))
```

Recover sees fetch/decode failures, not save failures. The outer consuming
function still needs to handle all failures that escape, including fallback
and save failures. No general value-union type is introduced to express this.

**A. Declared source/operator error effect plus consumer effect.** A collect
operation composes them for its caller while retaining error origin for
recovery. Pros: fits typed effects and useful hover. Cons: public generic
signatures can become large; implementation needs exception transparency,
not just broad catch around emit.

**B. Fallible cursor as the boundary.** Source/adapter errors arise from next;
consumer errors arise after next in the loop body. Pros: clear operational
separation. Cons: changes D46's lazy-error restriction and needs typed-error
integration in loop lowering and any collector bridge.

**C. First support typed errors in collect only.** Pros: smaller first slice
if the collector model is chosen. Cons: loops over the same source cannot
silently erase failure; a restricted/deferred fallible-loop adapter must be
documented rather than made to fail an unrelated owner scope.

These can combine at different layers. None is permission to invent a new
`try loop` spelling here or silently repurpose cancellation as EOF.

**Recommendation:** preserve normal Veles try/error inference and make
source versus consumer origin explicit in implementation and diagnostics.
Choose public signatures only after O1/O4 establish the protocol boundaries.

**Question O5:** Must the first coherent release support both fallible collect
and loop consumption, or may fallible loop integration be explicitly staged?
How should declared source errors and downstream callback errors appear in
public signatures and hover without losing their origin?

### O6. Early stopping and emit authority

**Problem.** A collector pipeline needs the equivalent of breaking a loop.
Stopping after five results is success, not a source failure or cancellation
of the caller's whole scope. The producer must still release its connection.

```text
try pages().take(5).collect(page => render(page))
```

**A. Structured internal stop control.** take asks upstream to stop and
unwinds owned production; callers observe normal completion. Pros: clean
surface, no boolean on every emit. Cons: stop must not be catchable as an
ordinary error or swallowed by user recovery; implementation must identify
which collection it belongs to.

**B. Explicit continue/stop result from downstream.**

```text
emit(page) returns Continue or Stop
```

Pros: visible protocol without nonlocal internal control. Cons: every
producer must honor Stop; ignoring it risks continued work or resource use;
may require a restricted builder rather than an unrestricted callback.

**C. Cancel a producer task.** Pros: natural for a task-backed bridge. Cons:
not suitable as universal same-task collection control; cancelling the wrong
owner can cancel siblings; cancellation must not be reported as success in
unrelated code.

Emit authority has a related boundary: can the producer store emit, use it
after return, call it concurrently or invoke it on another executor? Allowing
all of these turns a simple cold builder into a concurrent callback hub.

**Recommendation:** plain builders emit sequentially only within collection;
concurrent/callback builders are explicit. Use a structured collection-local
stop contract, with exact lowering chosen after the protocol decision.

**Question O6:** Prefer hidden structured stop, explicit Continue/Stop, or
task cancellation only for task-backed adapters? Should escaping/concurrent
emit be rejected in the ordinary builder and require a distinct facility?

### O7. Cleanup failures and cancellation

**Problem.** A source can fail while its close also fails, or an observer can
leave while a producer is suspended. Existing cleanup precedence/shielding
rules must remain recognizable through every adapter.

```text
with source = try openRemotePages()
try source.collect(page => try save(page))
```

Consider four exits: successful completion; source error; consumer error;
owner cancellation. In each case, owned resources close once and child work
is joined. A cancellation must not become a successful empty stream.

**A. Close follows existing with rules; its effect is reflected at the owning
boundary.** Pros: predictable language-wide behavior and retained failure
context. Cons: signatures/tooling must show where close can fail or suspend.

**B. Resource close itself cannot throw; fallible finish is explicit.** Pros:
simpler cleanup effects. Cons: callers must not forget finish; cannot silently
redefine existing general cleanup behavior to fit one library.

**C. Close failures are logged/dropped.** Pros: simpler-looking callers.
Cons: lost flush/commit errors and misleading success; unsuitable as a
universal default without an explicit best-effort contract.

**Recommendation:** A, inheriting existing error precedence rather than
inventing it. Cleanup must be bounded because cancellation shielding does
not make stuck external cleanup interruptible.

**Question O7:** Which operations expose fallible close/finish, and how does
the pipeline preserve the primary error plus cleanup context across bridges?
No question here reopens the already specified general cleanup precedence.

### O8. Atomic state update and equality

**Problem.** State should avoid redundant updates and lost increments, but
"atomic update" does not state whether its callback can run repeatedly.

```text
count.update(value => value + 1)
```

**A. Serialize a short non-suspending update under a lock.** Pros: callback
can be evaluated once per call that reaches it; straightforward atomicity.
Cons: contention, reentrant operations can deadlock, long computations stall
other updates. Existing held-lock restrictions must reject suspension.

**B. Read/compute/CAS retry.** Pros: avoids holding a mutex across computation
and can work well under some contention patterns. Cons: callback may run
several times, so logging, ID allocation and other effects inside it can repeat.

```text
count.update(value => {
  writeAuditRecord(value)
  value + 1
})
```

That example is unsafe as an exactly-once audit under retry semantics. A
successful state update is not a transaction with unrelated external effects.

**C. Set/get only; callers coordinate transformations.** Pros: tiny API.
Cons: get-plus-set loses concurrent increments unless callers add locking;
does not meet the selected atomic-update direction without another safe API.

Equality introduces another choice: default value equality, a custom
comparator per state, or explicit identity/versioned state variants. Custom
comparators can be slow, non-transitive or reentrant; mutating payloads
outside set/update can defeat any equality policy.

**Recommendation:** expose atomic update and equality semantics explicitly;
do not promise callback-once and lock-free retry together. Keep Watch's
version notifications for its existing role. Benchmark the eventual strategy.

**Question O8:** Must update callbacks be evaluated at most once per call,
or may they retry under a documented no-side-effects contract? Is default
equality sufficient initially, or is custom distinctness required at launch?

### O9. Event capacity, replay and loss reporting

**Problem.** Capacity can mean total retained values or extra lag room beyond
replay. If the API leaves this vague, users cannot predict memory or blocking.

```text
val events = Events(capacity: 4, replay: 2, overflow: Suspend)
```

**A. Capacity is the total retained-item bound, replay is a subset.** At most
four retained items, of which at most the last two are replay candidates.
Pros: simple headline memory bound. Cons: replay competes with lag retention;
the constraints when replay exceeds capacity must be explicit.

**B. Replay plus extra lag capacity.** With replay two and extra capacity
four, storage can retain up to six items. Pros: independently tune late-reader
history and active-reader lag, similar to Kotlin's explicit extra capacity.
Cons: a parameter called just capacity would mislead; users must add both.

**C. Separate replay storage and per-reader queues.** Pros: flexible reader
policies. Cons: memory grows with reader count, payload references persist
in multiple places, and a single capacity no longer bounds the whole hub.

Loss reporting must match the chosen policy. Dropping old unread events can
advance a lagging reader and report a missed count. Dropping a newly emitted
event requires an acceptance/loss report or metric; it does not automatically
produce the current Broadcast Lagged behavior. Do not return unconditional
success while promising observable loss reporting.

Also define bounds on blocked emitters and in-flight payloads. Four retained
items plus thousands of tasks blocked with large values is not four items of
total application memory. Backpressure must pair with bounded admission.

**Recommendation:** A for an easily explained total bound, or B with explicit
`extra` naming if independent lag room is important; do not silently choose C.

**Question O9:** Total bound with replay inside it, or replay plus explicitly
named extra capacity? Which loss policies are initially supported, and what
does emit/tryEmit report for rejection, dropping and lag?

### O10. No-reader retention and durable delivery boundaries

**Selected:** no active readers retain only replay; emit does not wait for an
imaginary subscriber. The open question is how applications needing stronger
delivery discover and select a different mechanism.

```text
val events = Events(replay: 0, overflow: Suspend, capacity: 8)
events.emit(PaymentAccepted(id))
```

With no subscribers this event is not retained under the selected direction.
Suspend applies to full active-subscriber buffering, not offline persistence.
That behavior must be visible in docs and examples, not buried as a caveat.

**A. In-memory work queue for one handler per job.** Pros: bounded handoff,
jobs can wait for workers. Cons: consumers compete rather than all receive;
process failure loses memory-only jobs; shutdown ownership still matters.

**B. Explicit bounded offline event queue.** Pros: retain events until
subscribers appear under a defined limit. Cons: overflow/wait policy and who
owns old history must be specified; still not durable across process failure.

**C. Durable log/outbox with acknowledgement.** Pros: survives process restarts
and supports redelivery/replay. Cons: storage/transaction/recovery complexity;
not a promise that a general in-memory flow library should pretend to satisfy.

Exactly-once external effects are not supplied by any in-memory emit return.
The application may require idempotent processing and durable acknowledgement.

**Recommendation:** make the distinction explicit; keep durable delivery out
of the core flow promise. Existing channels remain the low-level work queue.

**Question O10:** Is documentation plus channel adapters enough initially,
or is a bounded offline queue a required companion API? Durable logs/outboxes
remain a separate design rather than silently expanding event-hub semantics.

### O11. Publication racing finish, and the final state

**Problem.** Owner-bounded observation was selected, but a producer can race
normal finish. State readers may also need a final value after publication ends.

```text
events.emit(lastMessage)
events.finish()
events.emit(tooLateMessage)
```

**A. Typed Closed outcome for post-finish publication.** Pros: normal concurrent
shutdown is recoverable and composable. Cons: every publication signature may
need a closed/error result, including calls that are structurally safe.

**B. Failed immediate acceptance result.** Pros: useful for non-suspending
tryEmit paths; closed versus full can be explicit. Cons: callers must inspect
it, and a suspending emit still needs a defined outcome on close.

**C. Programmer-error panic.** Pros: strong misuse signal, like current
Broadcast send-after-close behavior. Cons: harsh for ordinary shutdown races;
callers may need synchronization just to avoid a panic.

Whatever surface is chosen, each racing publication must be ordered either
before finish and accepted according to its policy, or after finish and
rejected. It must not partly publish to an unexplained subset of subscribers.
Define what a waiting emitter observes when finish releases it.

For state, alternatives are retain-readable-last-value with final observation,
retain-readable-last-value but no new subscription, or make the whole handle
invalid after finish. Those choices affect UI teardown and late diagnostics.
Owner cancellation must still remain cancellation, not synthetic normal finish.

**Recommendation:** preserve existing Broadcast unchanged; favor an explicit
closed outcome for new concurrently closed surfaces. Retaining the last
readable state is useful, but final subscription behavior needs approval.

**Question O11:** For new APIs, is racing/post-finish emit a typed error,
an acceptance result or panic? Can a late state observer read/receive the
last value and then complete normally after explicit finish?

### O12. Sharing: restart, replay and failure ownership

**Selected baseline:** scope-owned eager/lazy/while-subscribed sharing;
minimal while-subscribed stops immediately when the last reader leaves and
starts a fresh attempt with cleared replay when a reader returns.

**Problem.** That does not settle what happens after upstream completes while
readers remain, or how producer failure is made visible without inventing
implicit supervision.

```text
scope {
  with shared = loadPages().share(start: WhileSubscribed, replay: 1)
  async table(shared)
  async summary(shared)
}
```

If loadPages finishes, should a later subscriber see retained completion,
start another request, or require an explicit restart? If loadPages fails,
does the owning scope fail unless the failure is handled, or is it published
as domain state through an explicitly configured conversion?

**A. One completed session per sharing handle; explicit restart/new handle.**
Pros: stable finite semantics and no surprise repeated effects. Cons: a UI
refresh needs another operation; while-subscribed restart after inactivity
must be distinguished from restart after a completed session.

**B. Restart whenever a new subscriber needs a completed source.** Pros:
convenient refresh. Cons: two arrivals can duplicate work without coordination;
late subscription can trigger unanticipated external effects.

**C. Never expose upstream completion; keep hot observation alive.** Pros:
ongoing UI surface. Cons: readers cannot infer a finite job completed unless
completion is data; not appropriate as the sole finite-event abstraction.

Failure policy should remain explicit: upstream recovery before sharing,
typed failure at the owner boundary, or deliberate conversion to status/data.
Closing one subscription is not automatic supervision of all failures in its
parent task. Preserve existing fail-fast structured concurrency.

**Recommendation:** separate inactivity restart from completed-session restart;
prefer explicit refresh for finite completed work. Grace timeout, replay
expiration and stale-while-revalidate are later named policies, not hidden defaults.

**Question O12:** After normal upstream completion, does sharing remain
completed, restart on a new reader or keep a non-completing domain view?
Where must failures be handled/converted before they can avoid failing the owner?

### O13. Bounded concurrency and latest-work details

**Selected:** sequential default, explicit bounds/placement, input-order
behavior preserved for the existing lazy concurrent map, and latest-work
replacement begins after obsolete owned work is cancelled and joined.

**Problem.** A worker limit alone does not bound reorder buffers. And latest
processing can receive many inputs while waiting for old cleanup.

```text
pages().mapConcurrent(page => try enrich(page), workers: 4,
                      maxOutstanding: 8, order: Input)
```

If the first item is slow while later items finish, continuing to admit
unlimited work accumulates finished results. The limit must count accepted
but not yet consumed work, not just currently executing callbacks. Specify
whether the source can hold one extra fetched-but-not-admitted item.

**A. One outstanding bound covers active, queued and completed results.**
Pros: understandable memory/backpressure contract. Cons: throughput can
decline behind a slow ordered item; payload byte size still varies.

**B. Independent worker, input-buffer and result-buffer bounds.** Pros:
precise tuning. Cons: more knobs and total accounting; invalid combinations
can deadlock unless the implementation reserves progress paths correctly.

For latest processing:

```text
queries.debounce(250ms).mapLatest(query => try search(query))
```

The duration is conceptual, not a new duration-literal proposal. While
cancellation cleanup runs, retaining just the newest pending query is bounded
and matches latest intent. Queueing all obsolete queries is not latest-work.
Generation identity prevents late results from an old request being rendered;
it does not cancel effects already committed by an external service.

**Recommendation:** A initially, preserving explicit input/completion ordering
where offered; retain at most the latest pending input during joined cleanup.
Execution placement must not silently move consumer callbacks to the producer pool.

**Question O13:** One total outstanding bound or multiple explicit queue
bounds? Is newest-only pending input the latest-work contract during cleanup,
and how are non-cooperative external operations surfaced rather than hidden?

### O14. Time and combination operator contracts

**Selected:** build a coherent core first, then advanced operators. Their
names alone do not settle observable behavior.

```text
queryChanges.debounce(Duration.millis(250)).collect(query => search(query))
```

For input A at time 0 and B at time 100ms, debounce normally emits B after
the quiet window. If upstream completes at 150ms, alternatives are flush B
immediately, wait until its deadline, or discard it. Each is observable;
completion and cancellation need not use the same rule.

For combine-latest, wait until every source has an initial value. Then new
input uses the latest values, not an atomic cross-source transaction:

```text
left:  A1 ---- A2
right: --- B1 ---- B2
output: (A1,B1), (A2,B1), (A2,B2)
```

Zip instead pairs next-with-next and commonly ends when the shortest source
ends; a longer owned source must then be stopped/closed. Merge interleaves
arrivals but needs bounded fan-in and a defined failure policy. Sample needs
a clock and must decide whether repeated unchanged values are emitted.

**A. Specify each operator's completion/cancellation/buffer contract before
adding it.** Pros: predictable composition and testable semantics. Cons:
slower growth of the operator catalogue.

**B. Adopt named Kotlin-like semantics as a package.** Pros: familiar and
well documented externally. Cons: typed errors and owner-bounded hot types
mean differences still need explicit adaptation; names cannot replace a spec.

**Recommendation:** A, borrowing established contracts where appropriate;
do not claim the staged first release already implements all of these.

**Question O14:** Should debounce flush its pending value on normal completion,
but discard on cancellation? Which of merge/combine/zip/time/latest is the
first advanced acceptance target after the core, and why that application?

### O15. Public surface, migration and release boundaries

**Problem.** Placeholder names are useful for discussion but can accidentally
become a large public API before the protocols are settled.

```text
val changes = state.changes()
try changes.collect(value => render(value))
```

Does changes include current value immediately, or only future changes? A
name alone does not answer that. Is the writer called State or MutableState?
Does the reader own a subscription or open one on each collect? Do free
functions or protocol methods own operators in Veles' declaration model?

**A. Agree a compact public catalogue after O1-O3.** Document each constructor,
method, effect, ownership requirement and completion meaning; keep advanced
operators explicitly staged. Pros: coherent docs/hover/migration. Cons:
requires application examples before coding rather than inventing signatures
as implementation proceeds.

**B. Mirror Kotlin names closely.** Pros: recognition and migration familiarity.
Cons: this design deliberately differs in typed errors, owner-bounded
termination and existing primitives; identical names can imply false parity.

**C. Expose only adapters around existing primitives initially.** Pros:
smallest new surface, no need to rename Watch/Broadcast. Cons: selected state,
sharing and composition ergonomics can remain incomplete if adapters are
treated as the final design rather than a deliberate implementation slice.

**Recommendation:** A; use existing runtime mechanisms where suitable without
changing their public meanings. Retain Watch/Broadcast. Any eventual removal
or rename needs explicit approval, diagnostics and whole-tree migration.

The high-level target is concise pipelines and loops; the low-level escape
is direct cursor/channel/primitive ownership. The self-hosted compiler should
be able to use whichever is appropriate without mandatory reactive machinery.

**Question O15:** Once the foundation is chosen, prefer a Veles-native compact
catalogue or close Kotlin naming with explicit differences? What minimum
end-to-end application must work before calling the first core stage complete?

## 7. Implementation boundaries and acceptance

This is a pending plan inside the draft, not a completed checklist entry.
Do not start it merely because this document exists.

### 7.1 Suggested stages

| Stage                | Deliverable                                                                                       | Gate                                                                               |
| -------------------- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Design closure       | O1-O3 and core error/stop contracts; public catalogue and remaining explicitly deferred decisions | User approves a concrete spec, not just placeholder names                          |
| Protocol core        | Collection/iteration ownership, typed failures, early stop, acquisition and cleanup               | Sequential examples, errors/nullables, cancellation and no task-per-map regression |
| State/events         | Read-only state, atomic update, policy-driven subscriptions/replay/termination                    | No lost initial-update race; bounded memory; finish/unsubscribe wakeups            |
| Sharing              | Owned producer and selected start policies                                                        | Independent subscriber exit, restart/session behavior and parent failure policy    |
| Advanced composition | Explicit bounded concurrency, selected time/combine/latest operators                              | Deterministic tests for ordering, limits, cleanup and obsolete results             |
| Integration          | Docs, diagnostics, formatter/LSP, compiler/selfhost boundaries and migration                      | Repository gates and supported-platform validation recorded honestly               |

### 7.2 Adversarial behavior that must be pinned

- Construct an unused cold recipe: it must follow the chosen acquisition
  timing, without silently opening resources just to describe a pipeline.
- Collect twice and concurrently: independent recipes, one-shot rejection
  and shared subscriptions follow their distinct chosen contracts.
- Empty sources, nullable items and normal terminal state remain distinct
  from source failure, per-item failure and cancellation.
- Fetch, transform, consumer, fallback and close fail independently and in
  combination. Recovery boundaries retain error origin and primary context.
- take/break/timeout/unsubscribe and owner exit release resources once and
  join all owned children; no producer survives through an escaped emitter.
- State initial subscription races an update; get must not consume another
  observer's notification; equal/unequal values and updates under contention
  follow the selected callback/equality contract.
- Full buffers, replay establishment, slow/no readers, dropped publications,
  finish while blocked and unsubscribe while emitting never lose wakeups or
  exceed their documented retention/admission bounds.
- Sharing restarts and completed sessions do not mix stale/new results
  without a selected replay policy. A reader leaving differs from failure
  of its parent task and cancellation of the sharing owner.
- Concurrent map input/completion order and total admitted work remain
  bounded behind a slow first item; all queue/in-flight accounting is tested.
- Latest processing joins old cleanup, bounds pending input and rejects
  stale generation results; external non-cooperation remains visible.
- Executor affinity, confinement/Sendable, GC roots across suspension and
  cancellation-safe cleanup hold through adapters, not just native sources.
- Documentation, hover and diagnostics explain coldness, ownership, errors,
  loss, completion and costly bridges without contradicting runtime behavior.

### 7.3 Measurement and platform gates

No source shape proves a performance win. Before changing runtime/codegen or
hot std paths, follow the benchmark workflow: compare direct loops/channels,
native collectors/cursors and any bridges; report per-item cost, allocations,
retained memory, fan-out contention, slow-reader latency and cancellation time.

Equality on large snapshots, replay retention and many blocked producers
need separate measurement. Test relevant thread counts, GC pressure and
sanitizers according to repository practice; verify Windows, Linux and macOS
where affected. Full build/vet/test and self-host/tooling parity remain
implementation gates, not checks this documentation change claims to pass.

## 8. Question and dependency index

Read the explanation before answering its question. "Open; need another
example" remains a valid response. Choosing a selected direction does not
silently approve every recommendation inside its open details.

| Local question | Actual unresolved choice                                                | Dependency                                 |
| -------------- | ----------------------------------------------------------------------- | ------------------------------------------ |
| O1             | Collector-oriented core, iteration core or explicitly separate families | First architectural decision               |
| O2             | Repeated collection, one-shot resources and explicit sharing            | Independent of collector versus iterator   |
| O3             | Public recipe/traversal types versus factory/scoped handles             | O2; affects signatures                     |
| O4             | Direct/resumable/task-backed bridges and callback registration          | O1, O2 and ownership                       |
| O5             | Typed-error protocol signatures, origins and loop integration           | O1/O4; D46 amendment where needed          |
| O6             | Early stop and sequential/escaping/concurrent emit authority            | O1/O4/O5                                   |
| O7             | Fallible close/finish effects and error context                         | Existing cleanup rules plus O4/O5          |
| O8             | State update callback evaluation and custom equality                    | Selected state direction                   |
| O9             | Capacity/replay accounting and loss reports                             | Selected explicit delivery policy          |
| O10            | Offline queue companion API versus documentation/adapters               | No-reader policy already selected          |
| O11            | Finish/publication races and final-state observation                    | Selected owner-bounded termination         |
| O12            | Completed sharing sessions, restarts and failure ownership              | Selected scope/start policies              |
| O13            | Outstanding work bounds and latest pending inputs                       | Selected explicit concurrency/latest rules |
| O14            | Time/combine/merge completion and first advanced target                 | Core semantics before advanced operators   |
| O15            | Public names/catalogue, migration and first release application         | O1-O3 plus essential core contracts        |

The first focused discussion should settle O1-O3 with actual programs, not
ask the user to approve fifteen unrelated exact APIs at once. Other entries
ensure edge contracts are not forgotten while those foundations are debated.

## 9. Evidence and verification

Source documents:

- [Research, original alternatives and selected questionnaire directions](veles-research-streams.md).
- [Language decisions and amendments](veles-spec.md).
- [Open user notes and standing directions](notes_to_change.txt).
- [Current notification implementation](std/prelude/notify.vs).
- [Current iteration implementation](std/prelude/iter.vs).
- [Generator research, still separate and unsettled](veles-research-generators.md).

External primary references, consulted during the 2026-10-21 research:

- [Kotlin Flow: collection, sequential execution and context preservation](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-flow/).
- [Kotlin catch: upstream-only exception transparency](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/catch.html).
- [Kotlin StateFlow: equality-conflated current state](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-state-flow/).
- [Kotlin SharedFlow: replay, overflow and no-subscriber behavior](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-shared-flow/).
- [Tokio watch: latest value and receiver observation](https://docs.rs/tokio/latest/tokio/sync/watch/index.html).
- [Tokio broadcast: bounded retention, lag and closure](https://docs.rs/tokio/latest/tokio/sync/broadcast/index.html).
- [C# asynchronous streams: pull consumption and async disposal](https://learn.microsoft.com/en-us/dotnet/csharp/asynchronous-programming/generate-consume-asynchronous-stream).
- [ReactiveX Observable: subscription and terminal notifications](https://reactivex.io/documentation/observable.html).

The research brief contains a baseline program compiler-verified on macOS
on 2026-10-21. This draft adds no new claim that proposed APIs compile.
Its examples describe alternatives, not executable tests. No new streaming
runtime, compiler change, benchmark result or cross-platform validation is
claimed. Open work is retained in sections 6-8 rather than marked done.
