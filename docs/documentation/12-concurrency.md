# 12. Concurrency

New to threads, coroutines and tasks? [Concurrency, explained from
scratch](concurrency-explained.md) builds the picture first. Every rule
of the runtime — what happens on each failure, cancellation, timeout or
deadlock, and the mistakes the compiler cannot catch — is collected in
[Concurrency: what happens, exactly](reference/concurrency.md).

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
- `async` also starts a function *value* whose type is `sendable fun(...)`
  — a parameter, a field, a local: `async handler(req)`. The function and
  its arguments are evaluated here, and the task's errors are the ones its
  type declares (D103). A plain `fun(...)` value is refused: what it
  captures may not cross into another task.

Tasks run in parallel, on one thread per core (D66): the executor
spreads them over a pool of worker threads that share one heap, and a
task that suspends may resume on another thread. `[runtime] threads = n`
in the program's `veles.toml` sets the pool's size, and `VELES_THREADS=n`
in the environment overrides it — `VELES_THREADS=1` runs everything on
one thread, which is handy when debugging. Work that needs threads of its
own — CPU-heavy jobs, a library tied to one thread — goes on an
executor (see "Choosing the threads", below). What the compiler checks
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

## Handles belong to their scope

What `scope` gives you is a promise about its `}`: once control passes it,
every task started inside has finished — or, if the scope was left early,
has been cancelled and has run its cleanups. So:

- **A task you never await is fine.** The scope waits for it, and its
  failure fails the scope. `await` is how the body *uses* a task's value
  before the end, never what keeps the task alive.
- **A handle stays inside its scope** (D141). Storing it in something
  declared before the scope, or returning it, is a compile error: after an
  early exit its task would be gone. Await inside the scope, keep the value.
- **`await` gives the value, even when the function throws.** A child's
  error fails the scope before any `await` can see it, so the handle of a
  `throws Boom` function is `Task<i64>` and `await t` needs no `try`.

```veles
use io

error Boom { n: i64 }

fun size(n: i64): i64 throws Boom {
  await sleep(Duration.millis(5 - n))
  if (n < 0) throw Boom(n)
  n * 100
}

fun total(ns: List<i64>): i64 throws Boom {
  var sum: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []   // declared inside: the handles cannot outlive the scope
    loop (n in ns) tasks.push(async size(n))
    loop (t in tasks) sum += await t         // the value; a Boom fails the scope instead
  }
  sum
}

fun main() {
  io.println("total ${total([1, 2, 3]) catch (e) { -1 }}")
  io.println("total ${total([1, -2, 3]) catch (e) { e.n }}")
}
```

Output:
```text
total 600
total -2
```

To handle each child's error yourself, use `gather` (below): its results
keep the `Result`.

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
  as the whole value of a `with`, and as a field of a value that holds a
  task (below). Inside a `scope`, a `with`-task is cancelled at the end of
  its own block, before the scope waits for its other children.

### A value that holds a task

A helper that starts something and hands back a value to use it — Go's
`httptest.NewServer` — is a struct with a `Task` field (D111). Such a
value is **received with `with`** where it is made, or returned straight
to the caller, where the same rule applies; its tasks then belong to that
block, exactly as a `with`-task does:

```veles
use io

struct Ticker {
  ticks:   Channel<i64>
  running: Task<()>

  implement Closeable {
    fun close() { io.println("closed after the ticker stopped") }
  }
}

fun count(ticks: Channel<i64>) {
  var n = 0
  loop {
    await sleep(Duration.millis(2))
    n += 1
    ticks.send(n)
  }
}

fun ticking(): Ticker {
  val ticks = Channel<i64>(capacity: 100)
  Ticker(ticks, running: async count(ticks))
}

fun main() {
  with t = ticking()
  io.println("first tick: ${await t.ticks.recv() ?: 0}")
}   // the ticker is cancelled and joined here, then t.close() runs
```

Output:
```text
first tick: 1
closed after the ticker stopped
```

- Anything else is an error with the fix: `val t = ticking()`, passing it
  as an argument, putting it in a list or a field of a value that holds no
  task, or dropping it — a task with no block to own it would outlive
  everything.
- `async f()` is allowed as a field argument of such a constructor whose
  value is received or returned. Its arguments are evaluated where they
  are written, but the task starts only once every field has been, so a
  later field whose `try` fails starts nothing.
- At the end of the block the tasks are cancelled and joined, **then** the
  value's `close()` runs (a value holding a task need not be `Closeable`).
  A task that panics or ends with an `Err` fails the block, its error
  joining the function's `throws`.
- A returned value's tasks may not be given what the function's own
  `with`s close: they outlive it.
- The task-holding types are read from declared fields: a `List<Task<T>>`
  or a generic `Box<Task<T>>` does not count.

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

With a single task, `gather` gives that task's `Result` itself rather than
a one-element tuple: `when (gather { async risky() }) { ... }` is how a
program runs code and finds out whether it panicked (D103).

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

fun check(n: i64): i64 throws Rejected => if (n == 4) throw Rejected(n) else n

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

## Draining, limiting, retrying

Three more shapes come up often enough to be in the prelude (D110).
`ch.toList()` and `ch.forEach(f)` receive until the channel is closed and
drained. A `Semaphore(permits: n)` lets at most `n` holders in at once:
`acquire()` waits for a permit and returns a `Permit` that gives it back
when closed, so it is held with `with`; `tryAcquire()` does not wait. And
`retry(times, f, delay:)` calls `f` again after a thrown error, waiting
`delay` between calls, and throws the last error when the calls run out —
a panic is not retried, and a cancelled task stops at the wait:

```veles
use io

error Busy { }

fun produce(ch: Channel<i64>) {
  loop (i in 1..4) {
    ch.send(i)
  }
  ch.close()
}

fun query(db: Semaphore, inside: Atomic<i64>, most: Atomic<i64>) {
  with db.acquire()                  // at most two at a time
  val now = inside.add(1)
  most.update(m => m.max(now))       // update may rerun its lambda: no side effects in it
  await sleep(Duration.millis(1))
  inside.sub(1)
}

fun flaky(calls: Atomic<i64>): string throws Busy {
  if (calls.add(1) < 3) throw Busy()
  "answered on call ${calls.load()}"
}

fun main() {
  val ch = Channel<i64>(capacity: 2)
  scope {
    async produce(ch)
    io.println("${ch.toList()}")
  }
  val db = Semaphore(permits: 2)
  val inside = Atomic(value: 0)
  val most = Atomic(value: 0)
  scope {
    loop (_ in 0..<10) {
      async query(db, inside, most)
    }
  }
  io.println("at most ${most.load()} at once")
  val calls = Atomic(value: 0)
  when (val r = retry(5, () => try flaky(calls), delay: Duration.millis(1))) {
    is Ok  => io.println(r)
    is Err => io.println("gave up")
  }
}
```

Output:
```text
[1, 2, 3, 4]
at most 2 at once
answered on call 3
```

A ticker — a channel that receives the time every period — is
`time.ticker` ([chapter 20](20-time.md)).

## First one wins: `race`

`race` waits for whichever arm is ready first and stops waiting on the
rest — a task awaited by a losing arm is not cancelled; it runs on, and
its scope waits for it. Arms can receive from channels, sleep (a timeout),
or await tasks (D38):

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

An arm can also **send**: `ch.send(v) => …` is ready when the channel can
take `v` — room in its buffer, or a receiver waiting (D108). When it wins,
`v` is in the channel; when another arm wins, `v` was **not** sent. That
is the bounded send with a deadline, without a task to do the sending:

```veles
use io

fun offer(queue: Channel<string>, line: string): bool {
  race {
    queue.send(line)           => true    // there was room: sent
    sleep(Duration.millis(20)) => false   // still full: not sent
  }
}

fun main() {
  val queue = Channel<string>(capacity: 1)
  io.println("${offer(queue, "first")} ${offer(queue, "second")}")
  io.println("${await queue.recv()} then ${queue.tryRecv() ?: "nothing"}")
}
```

Output:
```text
true false
first then nothing
```

The channel and the value are evaluated once, in arm order, when the race
starts, not again while it waits. A send arm binds nothing. A send arm on
a channel that is closed when it would be chosen panics, as `send` does.

**Which arm wins.** When the race starts, the arms are tried in the order
they are written, and the first one ready wins — so an arm that is always
ready (a `sleep(Duration.zero)`, a channel with a value waiting) shadows
the arms after it. If none is ready, the race waits, and the first arm to
become ready wins. The rule is deterministic and favours earlier arms; it
is not fair in Go's sense (Go picks among ready cases at random).

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
unwinding completes even if the task is cancelled again meanwhile.
Cancellation is seen at a suspension point and at the **end of every loop
iteration** (D145), so a long computation stops too: the loop ends its
iteration and the task unwinds from there, closing its `with` blocks. It
is not seen inside a lock region (`withLock`, `with m.lock()`) or a
`close()`, which always run to their end. `checkCancelled()` adds a check
anywhere else.

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

A trait can promise it for all its implementors: `trait Stream : Closeable + Sendable`.
Its objects — an `io.Stream` holding a socket or a file — cross task
boundaries, every `implement` of it must be Sendable, and the compiler refuses
to turn a value that is not into one.

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
fun run(f: sendable fun(i64): i64, x: i64): i64 => f(x)
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
bites is a lambda passed to a non-suspending parameter, such as
`withLock`'s: its function runs with a lock held and cannot pause. The
list adapters take suspending functions: `xs.map(x => fetch(x))` fetches
one after another, and `xs.mapConcurrent(x => fetch(x))` all at once. And
a suspending function is compiled
as a state machine (see below), which is why a named non-suspending
function does not fit a `suspends` parameter as a value: pass
`x => f(x)` and the lambda is compiled the suspending way.

On a parameter, `suspends` means *may* suspend (D116). A function whose
only waiting is calling such a parameter — or passing it on to another
function's — suspends exactly when what it is given does. Given a lambda
that does not suspend, the call is an ordinary call, allowed wherever
waiting is not: under a lock, in `init`, in a global's initializer.

```veles
use io

fun eachTwice(xs: List<i64>, f: fun(i64): () suspends) {
  loop (x in xs) {
    f(x)
    f(x)
  }
}

fun main() {
  val total = Mutex(value: 0)
  total.withLock(t => eachTwice([1, 2, 3], x => *t += x))   // nothing passed suspends: a plain call
  io.println("total ${total.get()}")
  eachTwice([1], x => {
    await sleep(Duration.millis(1))                       // this one suspends, and so does the call
    io.println("waited for $x")
  })
}
```

Output:
```text
total 12
waited for 1
waited for 1
```

The compiler makes two copies of such a function, an ordinary one and a
suspending one, and each call picks the one it needs; the hover says
"suspends if `f` does". A function that *keeps* the parameter — stores it,
returns it, hands it to a function that keeps it — cannot know how it will
be called later, so it suspends whenever it is called. A trait method
keeps the effects its trait declares.

For state that genuinely must be shared and mutated, wrap it. A
`Mutex<T>` runs a function on the value with its lock held; an `Atomic<T>`
changes the whole value in one step. The tasks below run on different
threads at once, so both really exclude each other: a `Mutex` is a lock,
cheap when nobody else holds it (one atomic instruction to take, one to
give back), and an `Atomic` of a number, a `bool`, an enum or a pointer
takes no lock at all — its operations are single processor instructions.
What an `Atomic` offers:

| | |
|---|---|
| `load()`, `store(v)`, `swap(v)` | read, replace, replace and return the old value |
| `add(n)`, `sub(n)` | an integer's counter step; returns the new value, wraps on overflow |
| `fetchAnd(m)`, `fetchOr(m)`, `fetchXor(m)` | change an integer's bits; return the old value |
| `compareAndSet(expected, new)` | store `new` only if the value is still `expected`; says whether it did |
| `compareExchange(expected, new)` | the same, returning the value it found |
| `update(f)` | replace the value with `f(value)`, retrying until no other task got in between |

`update`'s function may run more than once, so it should only compute the
new value. Every operation is sequentially consistent — all tasks see the
atomic operations happen in one order — unless it is given a weaker
`order:`, which is for lock-free code
([chapter 13](13-memory-and-ffi.md#atomics-and-memory-orders)). An
`Atomic` of any other type is guarded by a lock:

```veles
use io

struct Counter { var hits: i64 }

fun bump(m: Mutex<Counter>, total: Atomic<i64>, times: i64) {
  loop (_ in 0..<times) {
    m.withLock(c => c.hits += 1)
    total.add(1)
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

When the work under the lock is more than one expression, hold the lock
with `with` instead (D107). `with n = notes.lock()` binds `n` to a
pointer to the value and keeps the lock until the block ends — on every
way out of it, as for any `with` ([chapter 13](13-memory-and-ffi.md)):

```veles
use io

struct Notes {
  var lines: MutableList<string> = []
  var saved: i64 = 0

  fun add(text: string) {
    this.lines.push(text)
  }
}

fun record(notes: Mutex<Notes>, text: string) {
  with n = notes.lock()     // held to the end of the block
  n.add(text)
  n.saved = n.lines.len()
}

fun main() {
  val notes = Mutex(value: Notes())
  scope {
    loop (i in 1..3) {
      async record(notes, "note $i")
    }
  }
  io.println("saved ${notes.withLock(n => n.saved)}")
}
```

Output:
```text
saved 3
```

The rules are `withLock`'s, applied to the block. Nothing in it may
suspend — `await`, `sleep`, a channel `send` or `recv`, a call of a
function that suspends — and the error points at the suspension and says
how to end the lock sooner, with the block form `with (n = notes.lock())
{ … }`. The pointer cannot leave the block: returning it, storing it or
capturing it in an escaping lambda is the same error as for any `with`
value. `lock()` is only a `with` value — `val n = notes.lock()` is an
error whose fix writes `with`. Locking the same `Mutex` again before the
block ends panics, as inside `withLock`; holding two different ones is
allowed, and the order is the program's. `with notes.lock()`, with no
name, is a bare critical section.

A module-level `var` is an error for the same reason a captured one is:
every task sees the module's bindings, from whichever thread it runs on.
Module state that changes is a `val` holding a lock:

```veles
// fragment
var served = 0              // error: module-level 'var served' is shared by every task …
val served = Atomic(value: 0)      // ok: served.add(1)
val cache = Mutex(value: MutableMap<string, string>())
val table = Lazy(init: () => buildTable())   // ok: built on first use
```

## More shared state: `RwLock`, `Event`, `Lazy`, `Broadcast`, `Watch`

Five more types cover the other shapes shared state takes (D146). Each is
`Sendable` when what it holds is, and copies of one are the same lock,
event or feed.

| Type | For | Operations |
|---|---|---|
| `RwLock<T>` | read often, changed rarely: many readers at once, one writer | `with c = l.read()`, `with w = l.write()`, `withRead(f)`, `withWrite(f)`, `get()`, `set(v)` |
| `Event` | "it happened": tasks wait until one sets it | `set()`, `reset()`, `isSet()`, `await e.wait()` |
| `Lazy<T>` | a value built once, on first use, by whoever asks first | `Lazy(init: f)`, `get()`, `isReady()` |
| `Broadcast<T>` | every receiver gets every value | `send(v)`, `subscribe()`, `close()`; `try await sub.recv()` |
| `Watch<T>` | the latest value of something that changes | `set(v)`, `get()`, `await w.changed()` |

`RwLock`'s `read()` and `write()` follow `Mutex.lock()`'s rules: usable
only as a `with` value, nothing in the region may suspend, and taking the
same `RwLock` again inside it panics. A writer that waits keeps new
readers out, so it is never starved. `Lazy`'s `init` runs under its lock:
tasks that ask meanwhile wait, and a panic in it is the panic of every
`get` from then on.

The waiting operations — `wait()`, `changed()`, `recv()` — always suspend,
so they are awaited, like a channel's `recv`; a cancelled task stops
waiting, and each can be a `race` arm. A `Broadcast` keeps its last
`capacity` values; a subscriber that falls further behind loses the
oldest and is told, once, by `Lagged`. Each copy of a `Watch` remembers
the version it last read, so a task given its own copy waits only for
changes it has not seen:

```veles
use io

struct Config { var level: i64 }

fun worker(config: RwLock<Config>, ready: Event, news: Subscription<string>, out: Channel<string>) {
  await ready.wait()
  val level = config.withRead(c => c.level)   // a read region cannot wait: read, then leave it
  loop {
    val item = (try await news.recv()) catch (e) { "lost ${e.missed}" } ?: break
    out.send("$item at level $level")
  }
}

fun follow(level: Watch<i64>, seen: Channel<i64>) {
  loop {
    await level.changed()
    seen.send(level.get())
    if (level.get() >= 2) return
  }
}

fun main() {
  val config = RwLock(value: Config(level: 1))
  val ready = Event()
  val news = Broadcast<string>(capacity: 8)
  val out = Channel<string>(capacity: 8)
  with sub = news.subscribe()
  scope {
    async worker(config, ready, sub, out)
    with (w = config.write()) {
      w.level = 2
    }
    news.send("a")
    news.send("b")
    news.close()
    ready.set()
  }
  out.close()
  io.println("${out.toList()}")

  val level = Watch(value: 0)
  val seen = Channel<i64>(capacity: 4)
  scope {
    async follow(level, seen)
    level.set(1)
    level.set(2)
  }
  seen.close()
  io.println("last seen: ${seen.toList().at(-1)}")
}
```

Output:
```text
[a at level 2, b at level 2]
last seen: 2
```

The follower may wake once for both changes or once for each — a
`Watch` promises the latest value, not every value; a `Broadcast` is the
one that delivers each.

## Choosing the threads: executors

Every task runs on the default pool unless told otherwise: one thread per
core, or the number `[runtime] threads` gives in the program's
`veles.toml` (`VELES_THREADS` in the environment overrides both). Most
programs never need more. Four tools place work elsewhere (D143):

| Tool | For |
|---|---|
| `try Executor.pool(threads: n, name: s)` | CPU-heavy work that must not take the threads a server answers on |
| `try Executor.thread(name: s)` | a library that must always be called from the same OS thread (a GL context, COM, a GUI loop) |
| `blocking(f)` | one call known to block its thread, such as a synchronous C read |
| `try Thread.start(name: s, f: f)` | plain code on an OS thread of its own: an audio loop, a pinned worker |

`scope(on: e) { … }` and `gather(on: e) { … }` start every child of the
block on `e`; everything else about the scope is as before. A task
**stays on its executor across every suspension**, and a child it
launches runs there too, unless its own scope says `on:`. `e.run(f)` is
the one-call form: it runs `f` on `e` and waits for the result.

```veles
use io

fun checksum(n: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<n) x = (x * 31 + i) % 1000003
  x
}

fun render(frame: i64): string => "frame $frame drawn"

fun main() throws ThreadError {
  with cpu = try Executor.pool(threads: 2, name: "cpu")
  val (a, b) = gather(on: cpu) {
    async checksum(1000)
    async checksum(2000)
  }
  io.println("${a} ${b}")

  with ui = try Executor.thread(name: "ui")
  io.println(ui.run(() => render(1)))

  io.println("${blocking(() => checksum(10))}")

  val mixed = Atomic(value: 0)
  with (audio = try Thread.start(name: "audio", priority: Priority.Normal, f: () => mixed.store(checksum(500)))) {
    io.println("mixing on its own thread")
  }
  io.println("mixed ${mixed.load()}")
}
```

Output:
```text
Ok(value: 737307) Ok(value: 418372)
frame 1 drawn
467877
mixing on its own thread
mixed 718330
```

- **Closing.** An `Executor` and a `Thread` are `Closeable`, so they are
  held by a `with`. Closing an executor stops its threads; the tasks
  placed on it have already finished, because the scopes that placed
  them joined them. Closing a `Thread` waits for its function to return
  — the closing thread blocks, as `join` does in Rust — and a panic in
  the function is raised again there.
- **Priority and CPUs.** `Executor.pool`, `Executor.thread` and
  `Thread.start` take `priority:` (`Priority.Low`, `Normal`, `High`,
  `Realtime`) and `cpus:` (the logical CPUs the threads may run on,
  numbered from 0). What the OS refuses is a `ThreadError`, never a
  setting quietly dropped: an unknown CPU, or on Linux a raised priority
  without `CAP_SYS_NICE`. Threads carry their names (`cpu-0`, `cpu-1`,
  …) into debuggers and profilers.
- **What crosses.** An `Executor` is `Sendable`. What a task on it
  receives follows the same rules as between any two tasks.
- **`blocking`** runs `f` on a pool that starts threads as calls need
  them, up to 128 (a call beyond that waits for one), and ends a thread
  idle for 10 seconds. `f` does not suspend. A blocking call made
  without it is still noticed after about a millisecond, and the
  default pool's other tasks moved to another thread; `blocking` says
  so up front.
- **Timers and sockets** are served by the default pool for every
  executor: a task on a pool can sleep, race and read a socket like any
  other — also while every default thread is busy in a long loop (the
  monitor steps in).

## How it compiles

A suspending function becomes a state machine: its locals move into a
heap-allocated frame and each `await` is a resume point (this is why
the design calls them *stackless coroutines*). A function that does not
suspend is a plain function with a plain stack frame; deciding which is
which is the inference pass. Trait methods declare `suspends` explicitly
(D40) so that callers through the trait know which calling convention to
use.

A task gives up its thread at a suspension point, and a loop in a
suspending function also gives it up by itself once it has run for about
10 ms while other tasks wait (D145). A function that does not suspend
keeps its thread until it returns (the other threads go on); `yieldNow()`
in its loop gives the thread up, and makes the function suspend. A call that blocks the thread itself — a C
function, a read from the terminal, waiting for a child process — is
noticed by a monitor within about a millisecond, and the thread's queue
of tasks goes to a spare thread until the call returns (D66).

Next: [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md).
