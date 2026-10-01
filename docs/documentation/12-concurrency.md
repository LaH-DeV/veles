# 12. Concurrency

New to threads, coroutines and tasks? [Concurrency, explained from
scratch](concurrency-explained.md) builds the picture first.

Veles concurrency rests on two decisions. First, there is no `async`
keyword on function declarations: whether a function *suspends* (may
pause waiting for something) is **inferred** from its body, the same way
its error type is (D2). The editor shows what was inferred: hovering a
function, at its declaration or at any call, reads `fun wait() suspends`,
and an inlay hint marks the declaration. Second, tasks are always **structured**: a task
is started inside a `scope`, and the scope does not finish until every
task in it has (D3). Nothing leaks, nothing is fire-and-forget.

## Starting tasks

```veles
use io

fun work(id: i64, d: Duration): i64 {
  await sleep(d)             // suspends here; the executor runs other tasks
  io.println("task $id done")
  id * 10
}

fun main() {
  scope {
    val a = async work(1, Duration.millis(100))
    val b = async work(2, Duration.millis(10))
    io.println("both started")
    io.println("results ${await a} ${await b}")
  }
  io.println("scope finished")
}
```

Output:
```text
both started
task 2 done
task 1 done
results 10 20
scope finished
```

- `async f(...)` starts `f` as a task in the enclosing `scope` and gives
  back a handle. `await handle` waits for its result.
- `sleep(d)` takes a `Duration` and always suspends, so it is awaited
  explicitly. Calling `work` makes `main` a suspending function too —
  you did not have to say so anywhere.
- `await` is only written on the primitives that are known to suspend
  (`sleep`, `recv`, task handles); a call to an ordinary function that
  happens to suspend needs nothing (D16).

Tasks run in parallel, on one thread per core (D66): the executor
spreads them over a pool of worker threads that share one heap, and a
task that suspends may resume on another thread. `VELES_THREADS=n` in the
environment sets the pool's size — `VELES_THREADS=1` runs everything
on one thread, which is handy when debugging. What the compiler checks
in this chapter — only Sendable values cross into a task, shared state
sits in a `Mutex` or an `Atomic` — is what makes that safe: two tasks
never change the same memory at once without a lock between them.

## Fail fast

If a task throws or panics, its scope **cancels its siblings** and
re-raises the failure once they have stopped (D34/D52):

```veles
use io

error Boom { n: i64 }

fun mayFail(n: i64): i64 throws Boom {
  await sleep(Duration.millis(10))
  if (n == 2) throw Boom(n)
  n * 10
}

fun slow() {
  loop (i in 0..<100) {
    io.println("slow tick $i")
    await sleep(Duration.millis(500))
  }
  io.println("never printed")
}

fun main() throws {
  scope {
    async mayFail(1)
    async mayFail(2)
    async slow()
  }
  io.println("not reached either")
}
```

Output:
```text
slow tick 0
```

The program ends with `error: main failed with Boom(n: 2)` on standard
error. Cancellation is checked at every suspension point, so `slow()`
stops at its next `await sleep`. A task that never suspends cannot be
cancelled — and does not need to be, since it also cannot block anyone.

## Background tasks: `with t = async f()`

A `scope` waits for its tasks, which is right for work you want
finished. A task that should run only *while* something else happens —
a server under test, a ticker, a reader draining a socket — is started
with `with` instead (D100). It runs in the background until the block it
is written in ends, and is then **cancelled, then joined**; a task that
has already finished has nothing left to stop:

```veles
use io

fun ticker(ticks: Channel<i64>) {
  var n = 0
  loop {
    await sleep(Duration.millis(5))
    n += 1
    ticks.send(n)
  }
}

fun work(): i64 {
  await sleep(Duration.millis(30))
  42
}

fun main() {
  val ticks = Channel<i64>(capacity: 100)
  with t = async ticker(ticks)
  with w = async work()
  io.println("work gave ${await w}")
  io.println("ticked meanwhile: ${ticks.len() > 0}")
}   // ticker cancelled here, then joined
```

Output:
```text
work gave 42
ticked meanwhile: true
```

- It is fail-fast like a `scope` child: if the task throws or panics,
  the rest of the block is cancelled at its next suspension point and the
  failure comes out of the block, its error joining the function's
  `throws`.
- `await t` waits for it and gives its value; the end of the block then
  has nothing to do.
- With other `with`s in the same block it forms one stack: the last
  opened is closed — or cancelled — first. In the program above, `w` is
  dealt with before `t`.
- `with (t = async f()) { ... }` stops the task at that `}` instead.
- `async` is allowed in exactly these places: inside `scope` or `gather`,
  and as the whole value of a `with`. Inside a `scope`, a `with`-task is
  cancelled at the end of its own block, before the scope waits for its
  other children.

## Collecting every outcome: `gather`

When you want all results, failures included, use `gather`. It waits
for every task and returns a tuple of `Result`s, one per task, in order.
A panic inside a task becomes `Err(Panic(...))` rather than taking the
program down (D36/D52):

```veles
use io

error Boom {
  n: i64
}

fun mayFail(n: i64): i64 throws Boom {
  await sleep(Duration.millis(1))
  if (n == 2) throw Boom(n)
  n * 10
}

fun crashes(): i64 {
  val xs = [1]
  xs.at(5) ?: panic("crashes: index 5 is out of range")
}

fun main() {
  val (a, b, c) = gather {
    async mayFail(1)
    async mayFail(2)
    async crashes()
  }
  io.println("$a")
  io.println("$b")
  when (c) {
    is Ok(v)            => io.println("ok $v")
    is Err(e)           => when (e) {
      is Panic(message) => io.println("panic: $message")
    }
  }
}
```

Output:
```text
Ok(value: 10)
Err(error: Boom(n: 2))
panic: crashes: index 5 is out of range
```

## Channels

A `Channel<T>` moves values between tasks. `send` suspends when the
channel is full, `recv` when it is empty, and `recv` returns `null`
once the channel is closed and drained (D16):

```veles
use io

struct Job { id: i64 }

fun worker(name: string, jobs: Channel<Job>, results: Channel<string>) {
  loop {
    val job = await jobs.recv()
    if (job == null) break
    await sleep(Duration.millis(1))
    results.send("$name did ${job.id}")
  }
}

fun main() {
  val jobs = Channel<Job>(capacity: 8)
  val results = Channel<string>(capacity: 8)
  scope {
    async worker("a", jobs, results)
    async worker("b", jobs, results)
    results.closeAfter(4)   // closes itself after the 4th send, whoever sends it
    loop (i in 1..4) { jobs.send(Job(id: i)) }
    jobs.close()
    val done: MutableList<string> = []
    loop {
      done.push(await results.recv() ?: break)
    }
    // which worker took which job is up to the scheduler
    val jobsDone = done.map(line => line.splitOnce(" did ")?.1 ?: "?")
    io.println("${done.len()} results, jobs ${jobsDone.sorted()}")
  }
}
```

Output:
```text
4 results, jobs [1, 2, 3, 4]
```

That is the worker pool, spelled out: a bounded number of tasks pulling
from one channel and reporting on another, inside a `scope` that joins
them. Several producers cannot tell which of them sent the last value,
so `results.closeAfter(4)` says it once and the consumer loops until
`recv()` returns `null`. A channel of `Result<T, E>` carries failures as
values when one bad item should not stop the others.

A channel made without a capacity, `Channel<T>()`, holds nothing: it is
a **rendezvous**. `send` finishes only when a receiver has taken the
value, so the two tasks meet — the sender knows its value has been
picked up, not just queued:

```veles
use io

fun hand(ch: Channel<string>, log: Channel<string>) {
  ch.send("baton")
  log.send("handed over")   // only once a receiver took the baton
}

fun main() {
  val ch = Channel<string>()
  val log = Channel<string>(capacity: 4)
  var baton = ""
  scope {
    async hand(ch, log)
    await sleep(Duration.millis(50))   // the sender waits all this time
    log.send("receiver ready")
    baton = await ch.recv() ?: ""
  }
  log.close()
  loop {
    io.println(await log.recv() ?: break)
  }
  io.println("got the $baton")
}
```

Output:
```text
receiver ready
handed over
got the baton
```

With a capacity, `send` finishes as soon as the value is buffered and
waits only when the buffer is full; blocked senders are served in the
order they arrived.

`send` and `recv` wait. When waiting is wrong — a best-effort
notification that should be dropped rather than hold up the sender, or a
loop that has other work to do — `trySend(v)` and `tryRecv()` ask and
move on:

```veles
use io

fun main() {
  val events = Channel<string>(capacity: 2)
  loop (e in ["start", "tick", "tick", "stop"]) {
    if (!events.trySend(e)) io.println("dropped $e")   // full: do not wait
  }
  loop {
    val e = events.tryRecv() ?: break                  // nothing buffered
    io.println("got $e")
  }
}
```

Output:
```text
dropped tick
dropped stop
got start
got tick
```

`tryRecv()` returns `null` both when the channel is empty for now and when
it is closed and drained; `len()` or a waiting `recv()` tells the two
apart. A loop that polls should `await sleep(Duration.zero)` between
tries: that yields, so the tasks it is waiting for get to run.

## The same job on every element

Most pools do one thing: apply a function to every element with a limit
on how many run at once. The prelude has that written once:

```veles
use io

error Rejected { n: i64 }

fun fetch(n: i64): i64 {
  await sleep(Duration.millis(1))
  n * n
}

fun check(n: i64): i64 throws Rejected = if (n == 4) throw Rejected(n) else n

fun main() {
  val ids = [1, 2, 3, 4, 5]
  val offset = 100
  io.println("${ids.mapConcurrent(n => fetch(n) + offset, workers: 2)}")
  val handled = Channel<i64>(capacity: 5)
  ids.forEachConcurrent(n => handled.send(n), workers: 3)
  var total = 0
  loop {
    val n = handled.tryRecv() ?: break
    total += n
  }
  io.println("handled all, total $total")
  when (val r = ids.mapConcurrent(n => try check(n))) {
    is Ok  => io.println("all $r")
    is Err => io.println("rejected ${r.n}")
  }
}
```

Output:
```text
[101, 104, 109, 116, 125]
handled all, total 15
rejected 4
```

`mapConcurrent` keeps the input order and runs at most `workers` calls at
a time; `forEachConcurrent` is the same for effects. The calls themselves
run in parallel on the executor's threads, in no fixed order — here each
reports on a channel rather than printing, so the output does not depend
on which finished first. The function may
suspend, and it may throw: then the whole call throws, and the first
error cancels the work still queued — the rule the eager adapters follow
([chapter 10](10-closures-and-iterators.md)). Go-to-definition opens
`std/prelude/concurrent.vs`, which is the channel machine above with
names on it; write the machine yourself when the shape is different — a
stream you do not want to collect first, a pipeline, per-worker state.

## First one wins: `race`

`race` waits for whichever arm is ready first and cancels the rest.
Arms can receive from channels, sleep (a timeout), or await tasks (D38):

```veles
use io

fun producer(ch: Channel<string>) {
  await sleep(Duration.millis(5))
  ch.send("hello")
}

fun main() {
  val ch = Channel<string>(capacity: 1)
  scope {
    async producer(ch)
    val first = race {
      val msg = ch.recv() => "got ${msg ?: "closed"}"
      sleep(Duration.seconds(1)) => "timeout"
    }
    val second = race {
      val msg = ch.recv() => "got ${msg ?: "closed"}"
      sleep(Duration.millis(2)) => "timeout"
    }
    io.println("$first / $second")
  }
}
```

Output:
```text
got hello / timeout
```

## Cancellation

A task is cancelled in four situations: a sibling in a fail-fast `scope`
failed; the body of its `scope` left early — a `return`, a `throw` or a
`try` that failed, a cancellation coming from further out — while it was
still running; the block of a `with t = async …` ended while it was
still running; or someone called `cancel()` on its handle. In each case
the same thing happens (D20/D34/D43):

- the task keeps running until its **next suspension point** (an `await`,
  a `sleep`, a channel operation, a socket read); a task that never
  suspends finishes on its own, and one cancelled before it got a thread
  never starts — which is why the example below waits until each worker
  holds its resource;
- there it **unwinds**: every `with` it is inside runs its `close()`,
  innermost first — a cancelled task calling a suspending function
  unwinds from the innermost call outwards, so a connection opened three
  calls deep is closed before the caller's own cleanup runs;
- the scope that owns it **waits** for that to finish. Leaving a scope
  body early therefore does not leak children: they are cancelled and
  joined before the `return` or `throw` completes.

```veles
use io

struct Res {
  name: string
  implement Closeable {
    fun close() { io.println("closed ${this.name}") }
  }
}

fun worker(name: string, opened: Channel<string>) {
  with r = Res(name)
  opened.send(name)
  await sleep(Duration.seconds(1))
  io.println("never printed")
}

fun firstReady(): string {
  val opened = Channel<string>(capacity: 1)
  scope {
    async worker("a", opened)
    await opened.recv()   // "a" holds its resource now
    return "gave up"
  }
}

fun main() {
  io.println(firstReady())
  val opened = Channel<string>(capacity: 1)
  with (t = async worker("c", opened)) {
    await opened.recv()   // the block ends: "c" is cancelled, then joined
  }
  io.println("done")
}
```

Output:
```text
closed a
gave up
closed c
done
```

Several children cancelled together unwind at the same time, on
whichever threads are free, so their cleanups run in no fixed order
between them; each task's own `with` blocks still close innermost
first, and all of them before the scope's `return` completes.

Cleanup itself is never cancelled (D47): a `close()` that runs during
unwinding completes even if the task is cancelled again meanwhile. Note
what cancellation is *not*: it is not a signal that interrupts running
code. A loop that computes without ever suspending will not notice it
until it reaches one. To make a long computation stoppable, give it a
suspension point now and then: `await sleep(Duration.zero)` does not
wait, offers the thread to other tasks, and is where a cancellation is
seen.

### Time limits: `withTimeout`

The prelude's `withTimeout(ms, f)` runs `f` in a task of its own and
gives up after `ms` milliseconds by throwing `Timeout`; `f`'s own errors
are rethrown, so the call throws `E | Timeout`. `f` must be sendable,
like anything handed to a task:

```veles
use io

fun slow(): i64 {
  await sleep(Duration.millis(500))
  42
}

fun quick(): i64 {
  await sleep(Duration.millis(1))
  7
}

fun main() {
  when (withTimeout(Duration.millis(20), () => slow())) {
    is Ok(v) => io.println("got $v")
    is Err(e) => io.println("failed: ${e.message()}")
  }
  when (withTimeout(Duration.millis(500), () => quick())) {
    is Ok(v) => io.println("got $v")
    is Err(e) => io.println("failed: ${e.message()}")
  }
}
```

Output:
```text
failed: timed out after 20ms
got 7
```

On timeout the task running `f` is cancelled and `withTimeout` waits for
it to unwind before throwing — the rule above — so whatever `f` had open
is closed by the time you see the `Timeout`. It is `scope` + `async` +
`race` written once; go-to-definition shows the seven lines.

## Values that follow a task: `TaskLocal`

A request id, a trace context or a logger is needed deep inside the
work, by functions that have no other reason to take it as a parameter.
A **task-local value** carries it instead (D72): bind it around a piece
of work with `withValue`, and everything that runs inside — including
tasks started there — reads it with `get`:

```veles
use io

val requestId = TaskLocal(fallback: "-")

fun log(msg: string) {
  io.println("[${requestId.get()}] $msg")
}

fun lookup(user: string): string {
  await sleep(Duration.millis(1))
  log("looked up $user")        // a task started inside the binding sees it
  user
}

fun handle(id: string, user: string) {
  requestId.withValue(id, () => {
    log("start")
    scope {
      val name = async lookup(user)
      log("found ${await name}")
    }
  })
}

fun main() {
  log("booting")
  handle("req-1", "ann")
  handle("req-2", "bob")
  log("done")
}
```

Output:
```text
[-] booting
[req-1] start
[req-1] looked up ann
[req-1] found ann
[req-2] start
[req-2] looked up bob
[req-2] found bob
[-] done
```

- `TaskLocal(fallback: v)` is declared once, usually at module level; `get`
  returns the fallback outside every binding.
- A binding cannot be changed while it is in effect, only shadowed: a
  nested `withValue` wins until it ends, and then the outer value is
  back. It ends with `withValue`'s function, also when that throws,
  panics or is cancelled.
- A task keeps the values bound where it was started, for as long as it
  runs — rebinding in the parent afterwards does not reach it.
- The value is read from other tasks, on other threads, so it must be
  Sendable (a `MutableList` is refused); a `TaskLocal` of a Sendable
  value is itself Sendable and can be captured by a sendable lambda.
- `withValue` may suspend (its function may), so it is not available
  inside a function that cannot — a lambda given to `List.map`, a
  `withLock`.

## What may cross a task boundary

Data handed to `async` must be **Sendable** (D35): numbers, strings,
immutable collections, structs whose fields are Sendable, channels,
`Mutex` and `Atomic`. A `MutableList` is not — two tasks could then
mutate it — and the compiler refuses the `async`. This is the concrete
reason `List` and `MutableList` are separate types.

### Functions that cross

A function value is a special case worth its own rule, because a closure
carries its captures with it. Handing `n => n * k` to another task hands
over `k` as well; handing over `n => { total += n; n }` would hand over
`total`, and two tasks could then write it at once — exactly the race D35
exists to forbid.

So a function value is **sendable** when calling it from another task can
reach no shared mutable state: a named function always is (it captures
nothing), and a lambda is when everything it captures is a `val` of a
Sendable type — immutable, so shared by reference safely. The compiler
works this out from the captures and records it in the lambda's type,
`sendable fun(i64): i64`. A parameter that will hand the function to a
task is declared with that type — that is what `mapConcurrent`'s `f` looks
like — and a plain `fun(i64): i64` parameter, like `List.map`'s, accepts
either. The reverse is refused with the capture named:

```veles
// fragment
fun run(f: sendable fun(i64): i64, x: i64): i64 = f(x)
val k = 10
var total = 0
run(n => n * k, 1)          // ok: k is a val of a Sendable type
run(n => { total += n; n }, 1)  // error: captures 'total', a 'var'
```

The rule composes with the rest: a struct holding a `sendable fun` field
is itself Sendable (a router with its handlers can be given to worker
tasks), and a lambda that needs mutable state shared across tasks
captures a `Mutex` or an `Atomic`, which are Sendable by design.

### `suspends`, from the caller's side

`suspends` on a function type means calling it may pause the current
task. Two consequences. It can only be called from a function that may
itself suspend — which is inferred, so in practice the only place this
bites is a lambda passed to a non-suspending parameter: `xs.map(x =>
fetch(x))` is refused, because `map` runs its function in a plain loop
and cannot pause; `xs.mapConcurrent(x => fetch(x))` is fine, because its
parameter is declared `suspends`. And a suspending function is compiled
as a state machine (see below), which is why a named non-suspending
function does not fit a `suspends` parameter as a value: pass
`x => f(x)` and the lambda is compiled the suspending way.

For state that genuinely must be shared and mutated, wrap it. A
`Mutex<T>` runs a function on the value with its lock held; an `Atomic<T>`
reads, replaces or `update`s the whole value in one step. The tasks
below run on different threads at once, so both really exclude each
other: a `Mutex` is a lock, cheap when nobody else holds it (one atomic
instruction to take, one to give back), and an `Atomic` of a number or a
`bool` takes no lock at all — its operations are single processor
instructions, and `update` retries a compare-and-swap until no other
task got in between. Its function may therefore run more than once, so
it should only compute the new value. An `Atomic` of any other type is
guarded by a lock:

```veles
use io

struct Counter { var hits: i64 }

fun bump(m: Mutex<Counter>, total: Atomic<i64>, times: i64) {
  loop (_ in 0..<times) {
    m.withLock(c => c.hits += 1)
    val _ = total.update(n => n + 1)
  }
}

fun main() {
  val shared = Mutex(value: Counter(hits: 0))
  val total = Atomic(value: 0)
  scope {
    loop (_ in 0..<4) {
      async bump(shared, total, 1000)
    }
  }
  io.println("hits ${shared.get().hits}, total ${total.load()}")
}
```

Output:
```text
hits 4000, total 4000
```

Three rules keep this simple. `withLock`'s function cannot suspend, so
no task ever waits at an `await` while holding a lock — the classic way
to deadlock a coroutine runtime is ruled out by the compiler. Locking a
`Mutex` again inside its own `withLock` panics with a message instead
of hanging. And a lock is given back even when the function panics.

A module-level `var` is an error for the same reason a captured one is:
every task sees the module's bindings, from whichever thread it runs on.
Module state that changes is a `val` holding a lock:

```veles
// fragment
var served = 0              // error: module-level 'var served' is shared by every task …
val served = Atomic(value: 0)      // ok: served.update(n => n + 1)
val cache = Mutex(value: MutableMap<string, string>())
```

## How it compiles

A suspending function becomes a state machine: its locals move into a
heap-allocated frame and each `await` is a resume point (this is why
the design calls them *stackless coroutines*). A function that does not
suspend is a plain function with a plain stack frame; deciding which is
which is the inference pass. Trait methods declare `suspends` explicitly
(D40) so that callers through the trait know which calling convention to
use.

A task gives up its thread only at a suspension point: the executor is
cooperative, so a long computation keeps its thread (the other threads
go on) until it reaches an `await` — `await sleep(Duration.zero)` offers
the thread without waiting. A call that blocks the thread itself — a C
function, a read from the terminal, waiting for a child process — is
noticed by a monitor within about a millisecond, and the thread's queue
of tasks goes to a spare thread until the call returns (D66).

Next: [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md).
