# Concurrency review — 2026-10-09

> **Status (kept current):** B1 fixed (F1); B4 largely done (F2: task 448 → 248 bytes,
> 657 → 400 bytes per parked task) and a 17+-arm `race` crash found and fixed with it;
> B3 done as D145 (F4: loops are cancellation points; suspending loops yield; a panic now waits
> for the children of the scopes it leaves). B2 done (F3: a suspending call that does not wait 70 → 4.4 ns; `httphello` ~15 % faster).
> Part 3's atomics gap done as D144 (F5: compare-and-set, integer add and bit operations, memory
> orders, lock-free pointers); the missing synchronisation types done as D146 (F6: `RwLock`,
> `Event`, `Lazy`, `Broadcast`, `Watch`, their waits usable in `race`); a `close()` that suspends done as
> D147 (F7: Tx rollback and otel flush at close; TLS close_notify at close as Go does);
> control of OS threads done as D143 (F8: `Executor.pool`/`Executor.thread`, `scope(on:)`,
> `run`, `blocking`, `Thread`, names, priority and CPUs; CPU work on a pool keeps the default
> pool's request latency near idle); B5/B6 and the Linux start latency done as F9 (child start
> ≤ 136 µs on Linux; nanosecond timers; race waits, sleeps and scope cancellation off the
> runtime lock; `httphello` 32 threads 0.95 → 0.57 s, still slower than 8 as Go's own is);
> F10 (2026-10-10) then took the allocation, the collector and the last runtime-lock uses out of
> a server's path: `httphello` as fast at 32 threads as at 8, a parked task 345 bytes (`bench/idle`);
> the open decisions were decided the same day as D143–D147 (Q24–Q28, all the
> recommended options). The order of the remaining work is track F in `veles-plan.md`.
> Found on the way: on Linux a child task can take up to ~7 ms to start at 8 threads
> (Windows ≤ 0.3 ms) — part of F9; a doc example that depended on it was made exact.

Requested by the user: "review all of the concurrency in Veles so we create a
powerful (performant) and usable language … very little memory and cpu when
programming asynchronously, and the possibility to precisely manage the
native OS threads." Written by us. Scope: spec D2, D3, D16, D20, D34–D38,
D47, D52, D54, D56, D66, D72, D100, D103, D107–D111, D116, D141; runtime
`veles_task.c`, `veles_sync.c`, `veles_poll.c`; codegen `coro.go`; prelude
`sync.vs`, `concurrent.vs`; chapter 12.

Machine: Windows 11, 32 logical cores, `--release` builds. Probe programs are
inlined below; each number was measured, not estimated.

## Verdict

The **surface design is strong** and should be kept: inferred suspension with
no coloured functions (D2/D116), structured concurrency with no orphans
(D3/D34/D141), `scope`/`gather`/`race` as typed constructs, compile-time
data-race freedom through `Sendable` (D35/D54), `await` refused under a lock
(D107), scoped task-locals (D72), cleanup shielded from cancellation (D47),
task-holding values received with `with` (D111). Few languages have all of
these at once.

What stands between it and "very little memory and CPU" is the
**implementation of suspension**, two **runtime pathologies**, and the
**absence of any low-level thread control**: today the only knob is the
`VELES_THREADS` environment variable.

| | Today | Go (same machine) | Target |
|---|---|---|---|
| Memory per parked task | 560–670 B | ~8.9 KB | ≤ 250 B |
| A suspending call that does not wait | ~70 ns, 4 GC allocations + a `setjmp` | ~1 ns | ≤ 5 ns |
| CPU while 100k tasks park on a channel (32 cores) | **all 32 cores at 100 % for seconds** | 0 | 0 |
| `withTimeout(200 ms)` around CPU work | returns after 1 278 ms | n/a (Go cannot either) | ≈ 200 ms |
| OS-thread control from code | none | `runtime.LockOSThread`, `GOMAXPROCS` | pools, dedicated threads, affinity |

---

## Part 1 — Bugs (reproduced)

### B1 (critical) Parking on a channel is O(n²) under a pure spinlock: every core spins

```veles
fun waiter(ch: Channel<i64>): i64 {
  val v = await ch.recv() ?: return 0
  v
}
fun main() {
  val ch = Channel<i64>()
  scope {
    loop (_ in 0..<100000) async waiter(ch)
    await sleep(Duration.millis(4000))
    ch.close()
  }
}
```

CPU used during 2 s of the "idle" period, by number of parked tasks:

| tasks | threads | CPU in 2 s |
|---|---|---|
| 100 | 32 | 47 ms |
| 10 000 | 32 | 188 ms |
| 100 000 | 2 | 3 984 ms |
| 100 000 | 8 | 15 969 ms |
| 100 000 / 1 000 000 | 32 | **62 800 ms** (every core at 100 %) |

Cause: `push_waiter` (`runtime/c/veles_task.c:1392`) walks the whole waiter
list to append — n parks cost n²/2 pointer chases, each a cache miss into a
different 430-byte task — and does it under the channel's `spin_lock`
(`:380`), which never yields or parks, so every other worker that touches the
channel spins on it. `remove_waiter` (`:1398`) is O(n) per call too (a
cancelled or raced waiter). Any server with many clients waiting on one
channel (a broadcast, a job queue, a `Semaphore`, which is a channel of
permits) hits this.

Fix (decision-free): a tail pointer and a doubly linked node (O(1) push and
unlink); the spinlock becomes spin-a-little, then `sched_yield`/`SwitchToThread`,
then park (or reuse the `Mutex` word's spin-then-park). Pin with a runtime
test that parks 100k receivers and asserts bounded CPU time, at 1/2/8 threads.

### B2 (high) Every suspending call is a whole task

```veles
fun leaf(i: i64): i64 {
  if (i < 0) await sleep(Duration.millis(1))   // never taken
  i % 7
}
fun mid(i: i64): i64 => leaf(i) + 1
fun top(i: i64): i64 => mid(i) + 1
// 10M calls of top: 2 112 ms. The same three functions without the sleep: 8 ms.
```

`callSuspending` (`codegen/llvm/coro.go:270`) runs each call of a suspending
function as a child task: `veles_task_new` (a ~430-byte GC object), an
argument cell, the coroutine frame, a result cell, and `run_entry`'s
`setjmp` and `jmp_buf` copy — about **70 ns per call that never waits**,
≈250× a plain call.

This matters more in Veles than anywhere else, *because* of D2: suspension
spreads by inference, so in a server nearly every helper between a handler
and a socket becomes suspending without the author seeing it. The language
promises that colour is free; the implementation charges for it on every call.
The spec already calls this the bootstrap's shortcut (D35 v0.29: "each
suspending call is a task of its own in the bootstrap").

Fix (decision-free, the largest performance item in this review):

1. **One task, a chain of frames.** A suspending call creates the callee's
   coroutine frame and resumes it inline; if it finishes without suspending,
   the caller reads the result from the frame and continues — no task, no
   scheduler, no `setjmp`. If it suspends, the caller suspends too and the
   task records the innermost frame to resume (C++20/Rust shape). The
   cancellation order D35 requires (innermost first) is the frame chain
   walked downwards, which is what the task chain does today.
2. **Frame memory:** frames within a task are strictly LIFO, so they can come
   from a per-task bump arena (segments of e.g. 1 KB, scanned by the GC as a
   unit) instead of one GC object each; LLVM's `coro.elide` (HALO) removes
   the allocation entirely where the callee's frame cannot outlive the call.
3. **Panics:** one `setjmp` per task resume, not per call (the cleanup stack
   already makes the unwinding correct).

Target: ≤ 5 ns for a suspending call that does not wait. Add
`bench/suscall` from the probe above so it cannot regress.

### B3 (high) A CPU-bound task cannot be preempted or cancelled

```veles
fun spin(n: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<n) x = (x * 6364136223846793005 +% i) % 1000003
  x
}
scope { async ticker(sw); async spin(400000000) }   // ticker wants a tick per 100 ms
val r = withTimeout(Duration.millis(200), () => spin(400000000))
```

With `VELES_THREADS=1`, the first tick arrives at 1 324 ms instead of 100 ms.
With default threads ticks are on time (another worker steals), but
`withTimeout(200 ms)` returns `Timeout` after **1 278 ms** at any thread
count: the computation never reaches a suspension point, so the cancellation
is only seen when it finishes. §11 records the timeout half; the starvation
half is not recorded.

Fix, in two parts:

- **Runtime, decision-free:** the monitor that already hands off the run
  queue of a thread blocked in C (D66 addendum) treats a task that has run
  for more than ~10 ms without suspending the same way: its queued tasks move
  to a spare thread. Nothing changes in meaning; latency stops depending on
  every task being polite.
- **Language, a decision (Q-C below):** let cancellation also be observed at
  loop back-edges, where the GC safepoint poll already sits. The check is the
  same load the poll does; the unwinding is the cleanup stack panics already
  use.

### B4 (medium) A task object is ~430 bytes, most of it idle

`veles_task` (`veles_task.c:88`) has ~44 words: 15 `int64_t` flags
(`state`, `failed`, `unwrap`, `listed`, `yielded`, `panicked`, `io_write`,
`io_waiting`, `io_ready`, `cancel_requested`, `unwinding`, `sched`,
`chan_wait_send`, `chan_done`, `test_task`), the panic report (9 words) and
the debug call chain (3 words) in every task, and separate fields for each
kind of wait (channel node 5 words, socket 6, timer 3, race, await) although a
task waits on one thing at a time. Each finish also allocates a result cell.

Fix (decision-free): flags into one or two words; panic data in a side
object allocated on panic; the shadow chain only in debug builds; the wait
fields in a union keyed by wait kind; small results (≤ 16 bytes) inline in
the task. Expected: ~140 bytes per task, ~250 bytes per parked task with its
frame — about 35× less than Go here. Add the idle probe above as a memory
benchmark (`bench/idle`: peak working set / tasks).

### B5 (known, §11) The runtime lock on hot paths

`race_wait` (24 times per HTTP request through `withTimeout`), timers,
`scope_cancel` and socket waits still take the one runtime lock; idle
workers sleep on one condition variable; `bench/httphello` is ~2× slower at
32 threads than at 8. Recorded already; listed here because it is the next
scaling limit after B1/B2. Direction: a timer heap (or wheel) per worker,
a lock-free race claim (cancellation design noted in §11), one parking slot
per worker.

### B6 (known, §11) `sleep` and race timeouts round up to whole milliseconds

Fine for servers, wrong for low-level work (audio, game loops, rate limiters
at high frequency). Nanosecond deadlines in the timer heap; the reactor's
wait takes the nearest deadline at the platform's resolution.

---

## Part 2 — Gaps: managing native OS threads (the user's explicit ask)

Today: `VELES_THREADS` at start, one worker pool for everything, automatic
hand-off of a thread blocked in C after 1 ms. A task may resume on any
thread after any suspension, and nothing in the language can say otherwise.
What cannot be written at all:

1. **Thread-affine native code.** GUI toolkits (the main thread on macOS and
   Windows), OpenGL/Vulkan contexts, COM apartments, C libraries with
   thread-local state, a Python interpreter's GIL thread. Each needs "every
   call from this one OS thread".
2. **Isolation of work classes.** CPU-heavy work (compression, the
   self-hosted compiler's passes) sharing workers with latency-sensitive I/O;
   no way to give each its own threads.
3. **Real-time or latency-critical threads.** Priority, CPU affinity,
   pinning, a name a profiler or debugger shows.
4. **Configuration in code or manifest**, not only the environment: a
   library or a test cannot ask for one thread; a server cannot cap itself.
5. **An explicit blocking offload.** The automatic hand-off is good devex,
   but a program that knows a call is long (a 2 s `fsync`, a DNS lookup in C)
   cannot say so and pays the 1 ms detection plus a spare-thread spin-up each
   time, unbounded in number.
6. **Single-threaded state without `Sendable`.** A program whose tasks all
   live on one thread could share non-Sendable state safely (Swift's
   `@MainActor`, Kotlin's `Dispatchers.Main`), with no locks at all —
   the cheapest possible async.

A shape that answers all six while keeping the structured rules (to be asked
as a decision, Q-A below; spellings are placeholders):

```veles
// fragment — proposal, not built
// 1. The pool the program runs on, from the manifest (env still overrides)
//    veles.toml:  [runtime] threads = "auto" | 4
// 2. Executors are values; children of a scope can be placed on one
val cpu = Executor.pool(threads: 4, name: "cpu")         // Closeable: joined on close
val ui  = Executor.thread(name: "ui")                     // one OS thread, an event loop
scope(on: cpu) { loop (f in files) async compress(f) }   // same scope rules, other threads
val h = await ui.run(() => window.redraw())               // runs on that thread, result back
// 3. Explicit blocking work on a bounded blocking pool
val data = blocking(() => legacyC.readAll(path))
// 4. A raw thread for full control; structured: joined when its `with` ends
with t = Thread.start(name: "audio", priority: .realtime, cpu: 3, stackSize: 256.kb) {
  audioLoop()                                              // plain code; may run its own executor
}
```

Points to decide with it: whether a task in `Executor.thread` may capture
non-Sendable values (it never leaves the thread); whether `scope(on:)` or a
per-`async` placement; what a `Thread` body may do (suspend? only on its own
executor); how affinity/priority degrade on platforms that refuse them.

---

## Part 3 — Gaps: low-level primitives

- **Atomics are seq_cst only and have no read-modify-write instructions.**
  `Atomic<T>` offers `load`, `store`, `swap`, `update(f)`. A counter is a CAS
  loop through a lambda (`update(n => n + 1)`), not one `lock xadd`; there is
  no `compareExchange` (the primitive every lock-free structure starts from),
  no `fetchAdd/Sub/And/Or/Xor`, no memory orderings (relaxed counters,
  acquire/release publication), no atomic reference (`Atomic<*T>`/`Atomic<T?>`
  of a pointer). Note the name clash to resolve: `Ordering` is already the
  comparison enum (D48), so a memory order needs another name. (Q-B)
- **Missing synchronisation types:** `RwLock<T>`; a notify/condition primitive
  (`Event`/`Notify`: wait until signalled, without a dummy channel); `Once` /
  lazy module-level values (today a module-level `var` must be `Mutex`/
  `Atomic`, so a lazily built table is a `Mutex<T?>` checked on every read);
  `Latch`/`Barrier`; broadcast and watch channels (config reload, shutdown
  signal to N tasks). (Q-D)
- **A named yield and cancellation check.** The documented idiom for a long
  computation is `await sleep(Duration.zero)`. A `yieldNow()` (and a
  `checkCancelled()` that does not yield) says what it means.

## Part 4 — Gaps: high-level devex

- **`Closeable.close()` cannot suspend**, against D47, which describes a
  suspending close shielded from cancellation. §11 records three casualties
  already: TLS `close_notify`, a database `Tx` that cannot send `ROLLBACK`,
  and `otel.start` that cannot flush. A suspending close (shielded, bounded
  by D47's hazard note) is a decision. (Q-E)
- **Data-parallel loops.** `mapConcurrent(workers: 4)` is the I/O worker
  pool; for CPU work the default of 4 is arbitrary (this machine has 32
  threads) and one task per element costs ~400 ns of scheduling. A
  `parallelMap`/`parallelForEach` that splits into chunks per thread is what
  the self-hosted compiler's passes and every batch tool want. Belongs with
  P11's still-open `awaitAll`, `firstConcurrent`, `filterConcurrent`.
- **D52's open point:** a supervisor cannot tell a cancelled child from a
  panicked one in a `gather` result.
- **`await` on `send`.** D16 says send and receive stay symmetric, but
  `await ch.recv()` is required and `ch.send(x)` is written bare, though
  both wait. Either is defensible; the asymmetry is the footgun to settle.
- **Observability of tasks:** a dump of every task, where it waits and
  since when (Go's SIGQUIT dump), on a hang or a signal; task/thread counts
  as runtime metrics (D126 lists them, not built).

---

## Proposed order of work

| # | Item | Kind | Size |
|---|---|---|---|
| 1 | B1: O(1) waiter lists; spin-then-park locks; runtime test | bug | small |
| 2 | B4: slim the task; result inline; `bench/idle` | perf | medium |
| 3 | B2: frame-chained suspending calls, per-task frame arena, `coro.elide`; `bench/suscall` | perf | large |
| 4 | B3 runtime half: hand off the queue of a long-running task | perf | small |
| 5 | Decisions Q-A … Q-E (one `veles-decide` batch) | design | — |
| 6 | Build the decided thread control and primitives | feature | large |
| 7 | B5/B6: per-worker timers, lock-free race, ns deadlines | perf | medium |

Items 1–4 change no program's meaning and can start now. Every step is
measured with `veles-bench` before and after, and on Linux (WSL).

## Open decisions raised (also in checklist §9)

- **Q-A** OS-thread control: executors as values, `scope(on:)`, a
  single-thread executor (non-Sendable state allowed?), `blocking(f)`, a
  structured `Thread`, `[runtime]` in the manifest.
- **Q-B** Atomics: `compareExchange`, `fetchAdd` family, memory orderings
  (and their name), atomic references.
- **Q-C** Cancellation observed at loop back-edges in non-suspending code.
- **Q-D** Synchronisation types: `RwLock`, `Event`/`Notify`, `Once`/lazy
  module values, `Latch`/`Barrier`, broadcast/watch channels, `yieldNow`.
- **Q-E** A suspending, shielded `close()`.
