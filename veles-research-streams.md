# Asynchronous sequences, observable state and shared events

Decision brief, 2026-10-21. **Research and proposals, not approved APIs or
implementation.** Prepared at the user's request: explain the alternatives
with snippets, pros and cons before asking specific questions. This precedes
generator planning; it neither decides generators nor starts implementation.

Discussion update, 2026-10-21: the user selected the recommended directions
for S3-S9. These are recorded preferences, not complete public signatures or
implementation authorization. S1 remains open: the user favors collector
flows alongside channels, generators and async generators, and asks whether
anything is missing. S2 remains open pending more examples and tradeoffs.
See section 15. No normative spec entry is recorded while the core protocol
and collection-identity contracts remain unsettled.

The examples use provisional names to expose behavior. Apart from the
explicitly tested baseline below, snippets are Veles-like pseudocode, not
programs the current compiler accepts. In particular, `pages`, `open`,
`State`, `Events`, `share`, `recover`, `debounce` and `mapLatest` below are
not assertions about existing public APIs. The helper functions stand for
application code. Choosing a behavior does not approve these exact names.

## 1. Existing decisions and implementation

- D42 already chooses one `loop (item in source)` spelling for synchronous
  and suspending iteration. Do not introduce mandatory `await foreach`
  merely because C# uses it.
- D46 already chooses eager collection operations and lazy iterator
  adapters, not a mandatory third "sequence" concept. It also explicitly
  refuses throwing lazy callbacks while `next()` is non-fallible. Adding
  fallible lazy iteration would amend that restriction, not fill a wholly
  undecided blank.
- D116 makes suspension follow callback effects. Ordinary calls do not
  acquire an `async` function modifier or require `await` everywhere.
- D146 provides `Watch<T>`, `Broadcast<T>`, `Subscription<T>`, `Event` and
  `Lagged`. They are useful mechanisms, not Kotlin Flow under other names.
- D147 and structured-lifetime decisions already govern suspending cleanup,
  cancellation and joining owned tasks. A streaming API must cooperate
  with them rather than launch unowned background workers.

Current code: [notifications](std/prelude/notify.vs),
[iteration](std/prelude/iter.vs), [language decisions](veles-spec.md).

| Existing mechanism   | Actual current behavior                                                                                                                           | Not yet the higher-level contract                                                                                        |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `Watch<T>`           | Current value, per-copy seen version, `set/get/changed`; every set advances version, including equal values; intermediate versions may be skipped | Not StateFlow equality suppression, a read-only state view, atomic transform API, or a full pipeline                     |
| `Broadcast<T>`       | Bounded retained events, non-waiting send, future-only new subscriptions, explicit lag error, close and drain                                     | Not configurable replay or suspending backpressure for slow subscribers                                                  |
| `AsyncIterator`      | `next(): Item? suspends`; end indicated by outer null                                                                                             | No declared typed failure on next; no adapter family in this declaration                                                 |
| Lazy `mapConcurrent` | Bounded admitted work, ordered results, owned tasks                                                                                               | Not a general hot/cold streaming and recovery API; task-owner failure is not the same as a recoverable next-item failure |

D46 says async iteration should receive lazy adapters. The current trait
declaration does not expose that family. That is an intent/implementation
gap; it is not evidence that the language decided against async adapters.

### Tested current-language baseline

This complete program ran with Go 1.23.2 and the current Veles compiler on
macOS on 2026-10-21. It uses no proposed stream API.

```veles
use io

fun main() throws Lagged {
  val settings = Watch(value: 1)
  io.println("${settings.get()}")
  settings.set(2)
  await settings.changed()
  io.println("${settings.get()}")

  val news = Broadcast<string>(capacity: 2)
  with subscription = news.subscribe()
  news.send("ready")
  news.close()
  io.println((try await subscription.recv()) ?: "closed")
  val ended = (try await subscription.recv()) == null
  io.println("$ended")

  loop (value in [1, 2, 3].iter().mapConcurrent(value => value * 2, workers: 2)) {
    io.println("$value")
  }
}
```

Output:

```text
1
2
ready
true
2
4
6
```

This verifies these particular operations, not absence of races in every
notification path or completeness of streaming semantics.

## 2. Four jobs that should not be confused

| Job                                   | Example                                    | What a slow reader is allowed to miss                                                   |
| ------------------------------------- | ------------------------------------------ | --------------------------------------------------------------------------------------- |
| Cold computation producing a sequence | Fetch every page of a query                | Nothing unless an explicit lossy operator is requested                                  |
| Current observable state              | Configuration or current connection status | Intermediate states; latest state matters                                               |
| Shared events                         | Chat messages or cache invalidations       | Depends on an explicit delivery policy; state-style conflation is not generally correct |
| Work queue                            | Distribute image resizing between workers  | Each accepted job goes to a worker, not every worker                                    |

Ten readers of the same channel do not automatically create ten copies of
every event. Conversely, turning a work queue into broadcast duplicates
work. State equality suppression would lose intentionally repeated events
such as two presses of the same button.

"Hot" means the producer/state exists independently of one collection.
"Cold" means a computation is started for a particular collection. It does
not mean slow/fast, synchronous/asynchronous, or buffered/unbuffered.

## 3. Cross-language evidence

Primary sources consulted on 2026-10-21:

| Approach                                                                                                                                              | Contract and useful lesson                                                                                                                                   | Cost or mismatch for Veles                                                                                                       |
| ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| [Kotlin Flow](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-flow/)                                   | A normal flow builder is cold and sequential; a suspending collector gives backpressure; operators compose; Flow's interface also admits hot implementations | Collector exception transparency and context preservation need rigorous enforcement; Flow does not itself prove a source is cold |
| [Kotlin StateFlow](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-state-flow/)                        | Initial/current value, equality-based suppression, latest-value conflation, read-only/mutable views and atomic updates                                       | Equality has a cost; mutable payloads undermine meaningful snapshots; never-completing state is not a finite result stream       |
| [Kotlin SharedFlow](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/-shared-flow/)                      | Broadcast, configurable replay and overflow; a scope can share a cold computation                                                                            | It never completes normally or intrinsically fails; with no subscribers, emit does not wait and only replay is retained          |
| [Tokio watch](https://docs.rs/tokio/latest/tokio/sync/watch/index.html) and [broadcast](https://docs.rs/tokio/latest/tokio/sync/broadcast/index.html) | Separate latest-value state and bounded events; explicit per-reader cursors and lag                                                                          | More explicit mechanisms and adapters rather than one high-level surface; close/loss behavior differs from Kotlin                |
| [C# async streams](https://learn.microsoft.com/en-us/dotnet/csharp/asynchronous-programming/generate-consume-asynchronous-stream)                     | Pull-based async iteration, natural loops, cleanup through async disposal on loop exit                                                                       | Explicit async syntax and exception/cancellation details should not be copied over Veles' existing rules                         |
| [ReactiveX / RxJS](https://reactivex.io/documentation/observable.html)                                                                                | Subscription-based values, terminal error/completion and rich combine/switch/time operators                                                                  | Subscription lifetime and backpressure need care; an Observable is not automatically bounded or backpressured                    |

Do not confuse Kotlin's collector model with a mandatory background task:
sequential emission and collection normally execute in the same coroutine.
Likewise a pull interface alone does not guarantee bounded upstream memory
when a source secretly prefetches or wraps an unbounded external queue.

## 4. S1: Consumer protocol and producer ergonomics

Motivation: display pages while a query is fetched, and stop after enough
results without leaking the connection.

### A. Async iteration as the common consumer contract (recommended)

Proposed pseudocode:

```text
val query = pages("open issues")
with cursor = try query.open()
loop {
  val page = try cursor.next() ?: break
  render(page)
  if (enoughResults()) break
}
```

The high-level target is the existing loop spelling, with typed-error loop
integration designed separately, not guessed as `try loop` here.

Pros: builds on D42/D46; natural early exit; a cursor exposes demand;
adapter state can be kept in the cursor without a task/channel per operator.
Cons: arbitrary producers need explicit state machines until a builder or
generator makes them easier; branching and callback-driven sources need bridges.

### B. Collector protocol as the common consumer contract

```text
val query = pageFlow("open issues", emit => {
  loop (pageNumber in 1..10) {
    emit(try fetchPage(pageNumber))
  }
})
try query.collect(page => render(page))
```

`emit` may suspend while the downstream consumer processes the page. It
must not be stored and invoked after collection ends or concurrently unless
the builder explicitly permits it. Early termination needs a structured
stop mechanism rather than swallowing errors from `emit`.

Pros: convenient producers without first designing generators; sequential
operator fusion is possible in one task, as in Kotlin.
Cons: early exit, downstream-error transparency and callback lifetime are
harder contracts; the error channel contains both producer and consumer
failure unless the implementation separates their origins correctly.

### C. Keep manual channels and existing iterators

```text
scope {
  val output = Channel<Page>(capacity: 4)
  async fetchPagesInto(output)
  consumePagesFrom(output)
}
```

This is schematic: the application must handle producer failure, close the
channel correctly and cancel/join the producer when consumption stops.
Pros: no new core abstraction; excellent low-level escape.
Cons: users repeatedly implement termination, failure and cleanup; merely
adding map/filter helpers does not solve ownership.

Recommendation: A as the common consumption contract; allow a collector-like
builder later if it interoperates without a second operator ecosystem.
This does not require declaring generator syntax or choosing its lowering.

**Question S1:** Common consumer protocol: A, B, or C? Is an interoperable
collector-style builder desirable later without making it the core protocol?

## 5. S2: Restartable sources versus one-shot cursors

Two consumers of a cold query should not accidentally steal each other's
pages. Conversely, an already-open socket cannot honestly promise replay.

```text
val query = pages("open issues")
with first = try query.open()
with second = try query.open()
// Independent requests/state; each starts from the beginning.

with input = try socketMessages.open()
// This source may be explicitly one-shot; a second open must not silently
// compete for the same cursor or claim it restarts the socket history.
```

A. Distinguish source/factory from cursor (recommended). A restartable source
opens independent cursors; a one-shot source advertises that restriction.
Sharing is explicit. Pros: honest lifetime and replay semantics. Cons: more
concepts/types; independent collection can duplicate expensive requests.

B. Make every stream a one-shot cursor. Pros: minimal protocol, direct fit
for resources. Cons: re-collection needs an explicit factory, less convenient
for derived UI/query pipelines. This is a viable minimal first release.

C. Reuse one producer automatically for every consumer. Pros: avoids duplicate
work. Cons: secretly hot, changes side effects/lifetimes, demands buffering,
replay and ownership choices even for a small sequential query.

Independent cold collection repeats side effects; restarting a computation
does not guarantee identical data, idempotence or a transaction snapshot.
Calling `next` concurrently on the same cursor is not part of the recommended
contract. Internal scope ownership need not mean a new public "Sequence" type.

**Question S2:** A explicit source/cursor distinction, B cursor-only first,
or C automatic sharing? Exact public names remain a later API decision.

## 6. S3: Terminal failure, item outcomes and recovery

One query may fail completely; one bad record need not invalidate a whole
import. These are different meanings, not interchangeable spellings.

### A. Typed terminal failure, optional outcomes as data (recommended)

Hypothetical protocol shape, not a declaration accepted by the current API:

```text
trait FallibleAsyncIterator {
  type Item
  type Error
  fun next(): Item? suspends throws Error
}
```

End-of-sequence is the outer absence, not a failed item. A nullable Item
still needs Veles' nested optional distinction (D5/D42); do not collapse
null data into end-of-stream. Exact associated-type names are provisional.

```text
with cursor = try importedRows.open()
loop {
  val row = try cursor.next() ?: break   // File/decoder failure stops iteration.
  recordOutcome(validate(row))          // An invalid row is data, if requested.
}
```

Pros: normal Veles typed-error handling; successful items stay unwrapped;
explicit per-item outcomes can continue after recoverable validation errors.
Cons: amends D46's non-fallible lazy restriction; loop lowering, generic
error effects, adapters, cleanup and diagnostics all need complete integration.

B. All failures are values such as `Result<Item, E>`. Pros: explicit and
convenient for independent batch outcomes. Cons: every consumer must inspect
values; terminal failure versus continue-after-error still needs a contract;
discarding an error value must not silently report success.

C. Keep next non-fallible and fail the owner task/scope. Pros: matches some
existing task-backed iteration. Cons: local stream recovery and retry cannot
generally recover an error that has already abandoned their owning scope.

#### Recovery boundaries

```text
pages("open issues")
  .map(page => try decode(page))
  .recover(error => cachedPages(error))
  .collect(page => try save(page))
```

Recommended: recovery handles query/decoding failure upstream of its position,
not `save` failure downstream, not cancellation, and not panic. A fallback
may fail too; that failure must remain typed. Error unions belong to Veles'
error system (D45), not a newly invented general value-union type.

Retry opens a new upstream attempt; values already delivered are not rolled
back. A retried query can emit duplicates, and repeated side effects need
application idempotence or a resumable cursor. Prefer explicit bounded
attempts/backoff, cancellation-aware waiting and an explicit error predicate.
Do not automatically retry every error or create an unbounded task tree.

Kotlin's [catch contract](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines.flow/catch.html)
is useful here: upstream-only, transparent to downstream failure/cancellation.

**Question S3:** A typed terminal failures plus optional item outcomes,
B all outcomes as values, or C existing scope-failure mechanism only?
Should recovery/retry preserve the upstream-only boundary above?

## 7. S4: Observable state, equality and mutation authority

Motivation: expose connection status to a UI without giving the UI permission
to modify it, and increment counters without a read/modify/write race.

```text
val mutableStatus = State(initial: "connecting")
val status = mutableStatus.readOnly()
mutableStatus.set("online")
mutableStatus.set("online")
// Equality-conflated state: the second set produces no additional change.
// Versioned Watch: both sets advance its version.

val count = State(initial: 0)
count.update(value => value + 1)
// Not count.set(count.get() + 1), which can lose concurrent increments.
```

A. Keep Watch's low-level contract; add a state-facing API (recommended).
Always initial/current value, equality suppression by default, latest-value
conflation, atomic update and a read-only view; reuse suitable machinery.
Pros: familiar state semantics without silently breaking Watch users.
Cons: another public concept and equality cost; it must not just duplicate
Watch with a different name and the same ambiguities.

B. Every set notifies; callers add explicit distinct filtering. Pros:
preserves version-notification meaning, no mandatory equality cost.
Cons: redundant state work by default; consumers must consistently request
the same distinctness policy. Latest-value storage still cannot promise
delivery of every intermediate update.

C. Replace/change Watch into StateFlow semantics. Pros: fewer public names.
Cons: breaking behavior and migration for callers using version notifications;
needs an explicit amendment to D146 and a whole-tree review.

Under A, a new subscription gets the current value without a snapshot/subscribe
gap. Reading current state should not implicitly change another subscription's
seen cursor. A read-only view restricts mutation authority, not visibility of
future changes; it is not a frozen snapshot.

Payloads crossing tasks must obey Sendable/confinement rules. A mutable object
must not change behind the state's back without an update notification. Prefer
immutable snapshots; expose references only under a separately safe contract.
Equality on a huge list can be expensive, so any custom-comparator/explicit
distinctness option needs its own precise API and performance evaluation.

For `update`, a conservative implementation serializes a short non-suspending
transform under a lock, once per successful call; it must reject waiting or
reentrant mutation that would deadlock. A CAS/retry implementation can be
faster under some loads but may call the transform repeatedly, so side effects
would be unsafe. Do not promise both exactly-once callback evaluation and
arbitrary lock-free retry. Benchmark before choosing the internal strategy.

**Question S4:** A preserve Watch and add state semantics, B every-set
notification plus explicit filtering, or C change Watch? Are initial/current
value, a read-only view and atomic update requirements for the chosen design?

## 8. S5: Shared events, overflow, replay and offline readers

Motivation: distribute messages to several active readers without unlimited
memory, while deciding whether a slow reader can lose messages.

```text
val messages = Events<Message>(capacity: 64, replay: 0,
                               overflow: Overflow.Suspend)
with reader = messages.subscribe()
messages.emit(message)
```

All names are proposed. There are three honest bounded delivery choices:

A. Require a policy for each event hub (recommended): choose suspending
backpressure for lossless active-subscriber delivery, or explicit loss policy.
Pros: intent visible; no accidentally lossy default. Cons: constructor/config
ceremony and more combinations to test. Capacity/replay need clear total
retention accounting, not independent hidden buffers.

B. Default to suspending emit when active readers fill the buffer. Pros:
no silent loss under active subscriptions. Cons: one slow subscriber can
stall a producer and every other subscriber; unsubscribe/cancellation must
wake blocked emitters. This is not a durable delivery guarantee.

C. Keep Broadcast-style non-waiting send and reported lag. Pros: producer
latency and retained items stay bounded; existing Veles behavior. Cons:
readers can miss events and must recover. Drop-oldest can report a cursor's
missed events; drop-newest needs an explicit delivery report/metric, not an
assumption that existing Lagged semantics automatically cover it.

Replay is separate: `replay: 2` gives a newly subscribed reader up to the last
two events. It is not additional unbounded history, an acknowledgement of
handling, or a request to rerun the producer. Subscription/replay establishment
must be atomic relative to publication so an event is not duplicated or lost
between "read replay" and "start live delivery".

Recommended zero-subscriber rule: retain only the configured replay, do not
wait for a nonexistent reader. This means replay zero loses such events.
Alternatives: wait for a subscriber (can hang startup), or explicitly use a
bounded offline queue/log with a separate ownership/delivery contract.
Kotlin SharedFlow uses replay-only retention without subscribers, even when
the configured overflow policy suspends with active subscribers.

No choice provides exactly-once effects. Emit completing means acceptance
under the delivery contract, not completion of every handler. Finite memory,
unbounded production, no loss and indefinitely stalled readers cannot all be
promised together. A bounded hub also does not bound the payloads retained by
an unlimited number of tasks blocked in emit: bound producer admission too.
External sources that cannot slow down need loss, coalescing, disconnection
or durable storage, not a misleading suspension promise.

**Question S5:** A require an explicit delivery policy, B lossless-active
backpressure by default, or C current reported-lag behavior only? Should
zero-subscriber retention be replay-only, wait-for-reader, or a separate
offline-queue facility rather than changing every event hub?

## 9. S6: Sharing owner and subscription lifetime

Two screens observing one connection should not open two connections, but
one screen closing should not accidentally stop the other's connection.

```text
scope {
  with shared = pages("open issues").share(owner: currentScope,
                                           start: Start.WhileSubscribed,
                                           replay: 1)
  async showTable(shared)
  async showSummary(shared)
}
// Owned producer is stopped, resources closed and children joined here.
```

`currentScope` is a conceptual owner reference, not a decided public value
or required syntax. Existing scope ownership might supply it implicitly;
it must not become an unbounded global subscription registry.

A. Explicitly scope-owned sharing with start policy (recommended).
Eager starts at sharing creation; lazy starts on first reader and stays
until the owner ends; while-subscribed stops when the last reader leaves.
Pros: predictable cleanup, independent subscriber lifetime, no hidden global
tasks. Cons: restart/replay rules and optional stop grace need careful design.

B. Automatically start on first subscription and keep forever. Pros:
convenient long-lived singleton sources. Cons: leaks work/resource lifetime
unless "forever" really means a named application owner.

C. No sharing abstraction; users explicitly manage a producer and Broadcast.
Pros: transparent control, viable low-level escape. Cons: repeated lifecycle
code, easy cleanup mistakes, no standardized restart policy.

For a minimal while-subscribed policy, recommend immediate stop, clear replay
on restart, and a fresh cold attempt for the next reader. Retaining stale
replay or adding a stop timeout can be valuable, but must be separately
configured and tested; do not quietly mix old results into a fresh attempt.
Application-level resume/transaction identity is not inferred by sharing.

Losing one subscriber cancels/closes its subscription, not the shared producer
while others remain. An uncaught error in a fail-fast parent can still cancel
siblings under existing scope rules: sharing is not an implicit supervisor.
The owner must decide whether a producer failure fails its scope or is handled
and published as state/data. Unsubscribing a blocked reader must detach/wake
it and release retention; merely setting an unread flag is insufficient for
a new robust subscription API. No existing close bug is claimed from this
design requirement alone.

**Question S6:** A owned sharing with eager/lazy/while-subscribed policies,
B implicit application-lifetime sharing, or C manual producer ownership?
Is immediate stop with fresh restart/cleared replay an acceptable first
while-subscribed contract, with grace/stale-replay options designed later?

## 10. S7: Completion and failure of hot state/events

Kotlin StateFlow and SharedFlow do not intrinsically complete or fail.
Veles Broadcast explicitly closes and drains. Do not silently change it
to a forever-open type in pursuit of Kotlin naming compatibility.

```text
val status = State(initial: Connecting())
status.set(Failed(connectionError))
status.set(Online())
// Recoverable domain status: subscribers continue observing.

events.finish()
// Finite stream end: subscribers drain accepted events, then complete.
```

A. Keep low-level Broadcast completion; make higher-level subscriptions
owner-bounded, with domain failures represented as data (recommended).
Pros: structured shutdown; state can recover without destroying subscribers;
finite events remain usable. Cons: not an exact Kotlin contract, and each
adapter must distinguish normal completion, owner cancellation and failure.

B. Make new state/events never complete intrinsically, like Kotlin; readers
stop only through cancellation/truncation. Pros: faithful Kotlin model,
simple ongoing-domain-state meaning. Cons: finite sources need a completion
value/envelope; collectors such as toList never finish without truncation.

C. Put terminal typed failure/completion directly on all hot types. Pros:
finite event processing has one familiar error channel. Cons: one source
failure can permanently kill a state observation; reconnecting requires
replacing/restarting its subscription, not setting a new status.

Under A, recommend preserving the last state as readable after normal finish;
a state subscription can deliver that final state and then end. Cancellation
of its observer/owner still follows cancellation, not fabricated normal EOF.
Whether a publication racing normal finish reports a typed Closed outcome or
is a programmer error needs a separate public-method decision; existing
Broadcast's send-after-close panic is not changed by this proposal.

`Failed(error)` is an application-declared state variant, not a newly universal
Error value type; respect D45's restrictions on unions outside error position.
State equality can suppress identical repeated failures, so use an event
channel when every failure occurrence must be recorded.

**Question S7:** A owner-bounded subscriptions and domain failures as values,
B Kotlin-style non-completing hot types, or C terminal errors on every hot
type? Must existing closable Broadcast remain available unchanged?

## 11. S8: Concurrency, execution context and buffering

The simple case should not launch a task or allocate a queue for every map.

```text
pages("open issues")
  .map(page => try decode(page))
  .filter(page => page.isRelevant())
  .collect(page => render(page))
```

A. Sequential and unbuffered by default; explicit concurrency (recommended).
Pros: bounded demand and easy local ordering, potential cursor/operator fusion,
no mandatory scheduler hop. Cons: source and consumer do not overlap unless
requested; a slow consumer slows the chain.

B. Buffered/background stages by default. Pros: more automatic overlap.
Cons: allocations/tasks even for cheap operations, hidden prefetch and latency,
more cancellation/error races; buffer limits become user-visible semantics.

C. Require users to build all overlap with channels/executors. Pros: precise
low-level control. Cons: too much ceremony for common streaming pipelines.

```text
pages("open issues")
  .mapConcurrent(page => try enrich(page), workers: 4,
                 maxOutstanding: 8, order: Order.Input)
  .collect(page => render(page))
```

The last three options are illustrative and extend earlier proposals; only
the existing lazy mapConcurrent workers parameter is implemented. An eventual
completion-order form should keep the same bound/accounting contract. Count
all admitted-but-not-consumed items, including work and reorder buffers, not
just actively executing callbacks. Item bounds do not imply byte bounds.

Execution-placement options must distinguish upstream execution from consumer
callbacks. A producer on a CPU pool must not silently move a UI/executor-local
consumer there. Concurrency requires existing Sendable checks; sequential
single-task processing must not acquire unnecessary Sendable bounds. Fixed
executor affinity and cancellation-safe cleanup remain existing obligations.

**Question S8:** A sequential default with explicit bounded buffer/concurrency,
B automatic staged execution, or C manual channels for all overlap? Should
execution placement be explicit and preserve the consumer's executor?

## 12. S9: Operator coverage and latest-work cancellation

The point of a pipeline API is composition, not a new type with only map.
Common operators differ in ways that must be explained before they are used:

| Operator concept    | Behavior                                                         | Main cost/risk                                                              |
| ------------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------- |
| map/filter/take     | Transform, select or truncate sequentially                       | Early take must close owned upstream work                                   |
| merge               | Interleave sources as values arrive                              | Bounded fan-in queues; failure cancels owned sibling work                   |
| zip                 | Pair the next element from each source                           | Slow input stalls pairing; define shortest-source termination               |
| combine latest      | Emit using latest value from every source after all have a value | Not an atomic multi-source snapshot; intermediate combinations are possible |
| debounce            | Emit after a quiet interval                                      | Latest value retained, timer/cancellation and completion-flush policy       |
| sample              | Observe latest value at intervals                                | Intentionally skips values; sampling clock/lifetime matters                 |
| latest-work mapping | Cancel obsolete in-flight transformation for a newer input       | Cancellation must join/clean up; external effects cannot be undone          |
| recover/retry       | Handle failure or restart upstream                               | Error-origin boundaries, duplicate results, bounded retry policy            |

Motivating live search, proposed pseudocode:

```text
queryState.changes()
  .debounce(Duration.millis(250))
  .mapLatest(query => try search(query))
  .collect(results => render(results))
```

This does not approve names or a builder. Suggested latest-work rule: request
cancellation of the previous owned operation, join its cleanup, then start the
next, retaining at most the latest pending input while cleanup runs. Ignore
obsolete results using generation identity. This gives bounded active work,
but slow/uninterruptible cleanup delays the next query. An alternative starts
new work before old cleanup finishes, improving latency but requiring a
separate bound and an explicit overlap contract. Neither can undo an HTTP
request or database side effect already accepted by an external service.

A. Stage a coherent core first (recommended): acquisition/iteration, typed
failure and cleanup, state/events adapters, map/filter/take and sharing;
then bounded combine/merge, time and latest-work operators. Write their
semantics and acceptance criteria now; do not imply they already exist.

B. Require the full reactive operator family before first release. Pros:
immediately useful rich composition. Cons: much larger surface and debugging
cost; time/lifetime semantics can delay validating the foundation.

C. Only primitive adapters; leave advanced operators to libraries. Pros:
smaller std. Cons: interoperability and safety contracts can fragment; the
public protocol must still let libraries express these operators correctly.

**Question S9:** A staged coherent std core with planned advanced operators,
B full reactive family first, or C primitive/std core plus third-party
operators? For latest-work, prefer cancel-and-join before starting the next,
or explicitly bounded overlap for lower latency?

## 13. Cost, self-hosting and validation

| Work                              | Compiler/tooling impact                                                                                                          | Library/runtime impact                                                                             |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Fallible async iteration          | Trait/error effects, loop lowering and spans, HIR/codegen, conformance, hover and both frontend/checker contracts where relevant | Cursor state, cleanup and adapters; no new scheduler assumed                                       |
| Cold-source/builder API           | Mostly existing syntax; a new builder syntax would be a separate decision                                                        | Reentrancy, resource acquisition, independent collection, early stop                               |
| Observable state/events           | Public API docs/completion and effect signatures; Sendable/confinement checks                                                    | Atomic subscription/update, bounded retention, cancellation-safe wakeups, publication/finish races |
| Sharing/concurrent/time operators | Existing scope/effect/executor integration; diagnostics need useful error origins                                                | Owned tasks, bounded queues/timers, cancellation and fail-fast coordination                        |

The self-hosted compiler benefits from streamed input/results and watchable
configuration, but plain lexer/parser loops must not become suspending or
task-spawning just because streaming adapters exist. Both low-level cursor
control and ergonomic loops must remain available. An async sequence is not
a requirement to implement the compiler as a reactive application.

No source shape proves performance. Measure allocations/retained memory,
per-item cost, fan-out cost, contention, fairness and latency under slow
consumers; compare direct loops/channels and suitable Go references before
claiming a win. Equality, copying large immutable snapshots and replay
retention need measurement too. Optimize shared implementation where safe,
not by erasing observable overflow/error semantics.

Required adversarial checks before any eventual implementation is complete:

- Independent cold cursors, explicit one-shot rejection, nullable items and
  completion; no acquisition just from constructing an unused recipe.
- Source, transform, consumer, fallback and cleanup failures remain distinct;
  cancellation/panic not swallowed; retries bounded with duplicate delivery
  understood; retained live values survive GC across suspension.
- Early break, take, unsubscribe, timeout and owner cancellation close resources
  exactly once and join owned children; shutdown/failure never looks like an
  accidentally empty successful result.
- Initial state versus concurrent update has no lost-subscription gap;
  equal/unequal updates, cursor independence, concurrent atomic updates and
  read-only capability boundaries follow the selected contract.
- Buffer full, replay versus subscribe race, no readers, very slow readers,
  finish while emitters/readers wait, unsubscribe while emitting, lag reporting
  and bounded producer admission are tested without timing-order assumptions.
- Reconnect/sharing restarts do not accidentally mix sessions; one observer's
  departure does not kill remaining subscribers; enclosing fail-fast behavior
  remains intentional rather than suppressed by an implicit supervisor.
- Latest-work cleanup/late result races, explicit active/outstanding limits,
  executor affinity, Sendable rules and synchronous fast paths are covered.
- Go/selfhost/compiler/tooling/docs integration and supported platform,
  thread/GC/sanitizer/performance gates are recorded honestly when built.

This is a design brief, not a claim those tests have been implemented or run.
Only the baseline program above was compiler-verified for this brief. No
runtime/compiler behavior, language spec decision or API was changed here.

## 14. Consolidated questions, after the explanations

These labels are local to this research brief, not new language decision IDs.
"Open / need more examples" is a valid answer to every question. Answers
choose a direction; exact signatures, migration, and remaining edge contracts
must be reviewed before a final specification or implementation.

| Question               | Options                                                                                           | Recommendation                                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| S1 consumer protocol   | A async iteration, B collector-first, C existing manual primitives                                | A; possible interoperable collector builder later                |
| S2 collection identity | A explicit source/cursor, B cursor-only first, C automatic sharing                                | A                                                                |
| S3 errors              | A typed terminal failure plus optional item outcomes, B all errors as items, C owner failure only | A; upstream-only recovery/retry                                  |
| S4 observable state    | A preserve Watch plus state-facing API, B every-set with explicit distinct, C change Watch        | A; initial value, read-only view, atomic update                  |
| S5 event delivery      | A explicit policy, B suspend-active default, C reported-lag only                                  | A; no readers retain replay only, offline queue separate         |
| S6 sharing lifetime    | A scope-owned policies, B implicit application lifetime, C manual                                 | A; minimal while-subscribed stops immediately and restarts fresh |
| S7 hot termination     | A owner-bounded/domain failures as data, B never completing, C terminal errors everywhere         | A; preserve existing closable Broadcast                          |
| S8 scheduling          | A sequential default/explicit bounded concurrency, B automatic stages, C all overlap manual       | A; preserve consumer executor                                    |
| S9 scope/latest-work   | A staged coherent std, B whole reactive family first, C advanced operators external               | A; cancel-and-join previous work first                           |

Names, `update` callback re-evaluation/reentrancy, error-bearing loop spelling,
finish/publication race outcomes, exact capacity/replay units, time-operator
completion behavior, custom equality and compatibility migration are deliberately
not smuggled in as decided details. Record any unresolved choice explicitly
in the eventual spec/checklist before coding. Do not allocate D198 or amend
D42/D46/D146 merely because this brief recommends a direction.

## 15. Follow-up: collector flows plus generators, and source identity

### 15.1 The user's proposed split is coherent

The user asks whether channels + collector flows + generators/async generators
cover the territory, each for a reason. Yes, as complementary mechanisms:

| Mechanism                      | Primary role                                                  | It does not imply                                         |
| ------------------------------ | ------------------------------------------------------------- | --------------------------------------------------------- |
| Channel                        | Transfer/queue values between separately scheduled activities | Broadcast, replay or automatic cleanup of every producer  |
| Collector flow                 | Compose a pipeline by passing emitted values downstream       | A channel/task per operator, or automatic sharing         |
| Generator                      | Write incremental production in ordinary-looking control flow | Re-collectability or a background task                    |
| Async generator                | Same producer ergonomics with suspending production           | Hotness, multicast or automatic parallelism               |
| Observable state/shared events | Expose current state or fan out events                        | A cold generator whose execution restarts for each reader |

Generators describe how a producer is written. Async iteration is a protocol
through which a consumer drives it. Collector flow is a different interaction
protocol: the producer invokes a downstream callback. One system can support
both, but "each has a reason" does not by itself define their interoperability.

An example of a coherent collector-oriented surface, all still pseudocode:

```text
val pages = flow(emit => {
  loop (number in 1..10) emit(try fetchPage(number))
})
try pages.map(page => decode(page)).collect(page => render(page))

with cursor = generatedPages().asAsyncCursor()
try cursor.asFlow().map(page => decode(page)).collect(page => render(page))
```

The problem is not that collector flow is wrong. It is whether `.map` has
one semantic contract and reusable implementation, or there are unrelated
generator, iterator, flow, channel and hot-source operator families. D46
already has eager/lazy duplication; avoid multiplying it without benefit.

Pull-to-collector adaptation can drive `next` in a loop, calling the collector
for each item and closing the cursor on every exit. General suspending
collector-to-pull adaptation is not a free symmetric wrapper: it needs to
pause an arbitrary producer at emit, possibly using generator/continuation
machinery or an explicitly owned task and bounded handoff. Do not claim
every bridge is allocation-free or one-task without a proven implementation.

A collector-first architecture is defensible when producer ergonomics and
pipeline use dominate. My revised recommendation is to investigate it as a
serious candidate, not reject it because async iteration already exists.
Keep generators as producer facilities, keep existing loops useful, standardize
error/cleanup/operator meanings across adapters, and make expensive bridges
visible. Do not approve two independent fully overlapping ecosystems first
and hope the bridge becomes free later.

Collector error transparency is a concrete additional obligation:

```text
flow(emit => {
  val page = try fetchPage(1)
  emit(page)
}).recover(error => cachedPages(error))
  .collect(page => try save(page))
```

The recovery stage may handle fetch failure, not save failure propagated back
through emit. Broadly catching everything around emit and calling it a source
failure would violate the selected S3 direction. Typed error effects must
distinguish producer/operator errors and downstream callback errors without
turning cancellation/panic into stream values.

### 15.2 S2 is independent of collectors versus iteration

A collector API still has to decide what a second collect means. A generator
API still has to decide whether an object is a recipe or one traversal. The
same questions exist without naming any new public type.

#### Example A: a paginated query displayed twice

```text
val query = pageFlow("open issues")
try query.collect(page => showInTable(page))
try query.collect(page => exportPage(page))
```

Three different honest contracts:

1. Cold recipe: first collect fetches from page one; second collect starts a
   fresh independent request sequence. Good: simple repeatable pipeline
   description, independent consumers. Bad: duplicate work/side effects;
   underlying data may change, so exports need not match the earlier table.
2. One-shot traversal: first collect consumes the session; second collection
   is refused or needs an explicitly documented continuation contract. Good:
   no false replay promise, direct ownership of a connection. Bad: storing a
   pipeline does not automatically let multiple consumers reuse it; clear
   diagnostics are essential, preferably before runtime where feasible.
3. Explicit shared source: collection attaches to one owned running producer.
   Good: no duplicate request execution for overlapping consumers. Bad: late
   export sees only future values/configured replay, not necessarily all pages;
   after the table finishes there may be no active producer or retained history.

Share is not cache. To export exactly the displayed results, materialize a
bounded snapshot or use a replayable/durable store with its own limits.
Neither cold recollection nor replay-one sharing guarantees identical history.

#### Example B: a socket's incoming messages

```text
with socket = try connect(address)
val incoming = socket.messages()
try incoming.collect(message => updateModel(message))
try incoming.collect(message => writeAudit(message))
```

An existing connection is not a recipe for all messages since connection
startup. If messages are consumed destructively, two readers must not silently
split traffic while the API claims broadcast. A one-shot incoming traversal is
honest. To update the model and audit every event, attach both readers to an
explicit shared producer before publication, or perform both actions in one
collector when their coupled latency/error behavior is acceptable.

A separate `connectAndRead(address)` recipe could open a new socket for each
collection. That is legitimate cold behavior, but it duplicates connection
side effects and does not recreate history from the old connection. The API
must not confuse the two merely because both emit Message values.

#### Example C: a generator object versus its factory

```text
val makeNumbers = () => generatedNumbers()
with first = makeNumbers()
with second = makeNumbers()
```

Calling a factory twice can create independent traversals, while an individual
generator object can be one-shot. No separate public Source class is required
to make that distinction real. The generator's exact syntax and contracts
remain undecided; this is a lifetime example, not an approved implementation.

### 15.3 The actual remaining S1/S2 questions

- S1: Prefer a collector-oriented high-level pipeline with generator/iterator
  adapters, while requiring consistent operator/errors/cleanup contracts and
  explicit costly bridges? Or prefer iteration as the common core with a
  collector-friendly producer facade? Either can be coherent; producer
  ergonomics versus bridge/control-flow cost is the real tradeoff.
- S2a: Should an ordinary cold flow builder rerun its body for every collect,
  including repeated side effects? Recommended yes, with sharing/cache explicit.
- S2b: Must resource-backed sources be allowed to declare themselves one-shot,
  instead of silently restarting or sharing? Recommended yes.
- S2c: Should that distinction require separate public source/cursor types,
  or initially be expressed by recipe/factory versus scoped one-shot handle?
  The latter is a viable smaller surface. It still needs enforced/documented
  lifetime behavior, useful hover and diagnostics; one ambiguous "stream"
  object that secretly does either is not a solution.

No need to decide generator syntax to answer these. A later implementation
probe can compare collector-to-cursor bridge costs and error propagation
before committing to a common internal representation.
