# Concurrency: what happens, exactly

This page states what the runtime does in every situation a concurrent
program can be in: when a task starts and stops, what `await` waits for,
where a task can be interrupted, what happens on an error, a panic, a
cancellation, a timeout or a deadlock, and what everything costs. The
tutorial is [chapter 12](../12-concurrency.md); the picture behind it,
for readers new to the subject, is
[Concurrency, explained from scratch](../concurrency-explained.md). Every
program here runs as shown; the rules cite the spec's decisions (`D<n>`).

## The model in six sentences

1. A **task** is a function call running on its own; `async f(x)` starts
   one, and it belongs to the `scope` (or `gather`, or `with`) it was
   started in (D3, D34).
2. Tasks run on a pool of threads, one per core by default, and a task
   may continue on a different thread each time it resumes (D66).
3. A task gives up its thread only where it actually **waits** — an
   `await`, a full channel, a socket — never in the middle of other code
   (cooperative scheduling).
4. When control passes a scope's closing `}`, every task started in it has
   finished, or has been cancelled and has run its cleanups.
5. A task that fails fails its scope: the other tasks are cancelled, and
   the error leaves the scope once they have stopped (D35).
6. Nothing a task can reach may be changed by another task at the same
   time, unless it is behind a `Mutex` or an `Atomic` — checked at compile
   time (D35).

## Starting a task: `async`

**It starts now, and the code after it does not wait.** `async f(x)`
evaluates `f` and its arguments in the current task, creates the new task
and queues it; then the current task goes on. The new task runs as soon as
a thread takes it: possibly at once on another thread, possibly only
after the current task next waits. **There is no guaranteed order**
between the code after `async` and the start of the task; a program that
needs one uses a channel or `await`.

**What it is given is fixed when it starts.** The arguments are values
computed before the task exists; a method's receiver is **copied** into
the task, so `async c.bump()` changes a copy, not `c`:

```veles
use io

struct Counter {
  var n: i64

  fun bump(): i64 {
    this.n += 1
    this.n
  }
}

fun main() {
  var c = Counter(n: 0)
  scope {
    val t = async c.bump()
    io.println("the task's copy is at ${await t}")
  }
  io.println("c is still at ${c.n}")
}
```

Output:
```text
the task's copy is at 1
c is still at 0
```

Everything passed or captured must be Sendable (D35) — a `MutableList`
is refused, a `List` is shared. To give a task state it changes and you
read back, pass a `Mutex` or an `Atomic`, or have it return a value.

**Where `async` may be written**: directly inside a `scope { }` or
`gather { }` body, as the value of a `with` (`with t = async f()`, D100),
or as a field of a value that holds a task (D111). Anywhere else it is a
compile error — there is no way to start a task that belongs to nothing.

**A function value** runs the same way if its type is `sendable fun(…)`
(D103); a plain `fun(…)` is refused, since its captures could be shared.

## Waiting: `await`

`await` is written on the things that always wait: a task handle,
`sleep(d)`, `ch.recv()`. A call of a function that happens to wait is
written as a plain call — the compiler works out that it suspends (D2,
D16).

**Awaiting one task does not stop the others.** `await t` pauses only the
task that wrote it. Every other task keeps running — so a loop that awaits
handles one by one still runs them all concurrently: it collects the
results in its own order, and a handle whose task has already finished
gives its value at once. The total time is that of the slowest task, not
the sum:

```veles
use io

fun job(id: i64, ms: i64, finished: Channel<i64>): i64 {
  await sleep(Duration.millis(ms))
  finished.send(id)
  id * 10
}

fun main() {
  val finished = Channel<i64>(capacity: 3)
  var sum: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    tasks.push(async job(1, 60, finished))
    tasks.push(async job(2, 40, finished))
    tasks.push(async job(3, 20, finished))
    loop (t in tasks) sum += await t   // waits about 60 ms in all, not 120
  }
  finished.close()
  io.println("sum $sum; they finished in the order ${finished.toList()}")
}
```

Output:
```text
sum 60; they finished in the order [3, 2, 1]
```

While the loop waited for task 1, tasks 3 and 2 finished; their `await`s
returned at once. To bound how many run at a time — a thousand requests,
not a thousand cheap computations — use `xs.mapConcurrent(f, workers: n)`.

**A handle can be awaited more than once, and from another task.** The
second `await` gives the same value; a handle passed to a sibling task
(it is Sendable) can be awaited there:

```veles
use io

fun work(): i64 {
  await sleep(Duration.millis(5))
  7
}

fun plusOne(t: Task<i64>): i64 => await t + 1

fun main() {
  scope {
    val t = async work()
    val u = async plusOne(t)
    io.println("${await t} ${await t} ${await u}")
  }
}
```

Output:
```text
7 7 8
```

**What `await` gives** (D141): in a `scope`, and for `with t = async f()`,
the task's value — `Task<i64>` for a function that `throws`, and no `try`
on the `await`, because the error never reaches it: it fails the scope
instead (below). In `gather` the handle is `Task<Result<T, E>>`. A task
that failed is never awaited with a value: the code awaiting it is part
of the failing scope, and is abandoned or cancelled at that `await`.

**A handle stays in its scope** (D141). Returning it, storing it in a
variable or collection declared before the scope, or capturing it in a
lambda stored there is a compile error: after an early exit its task
would be gone. Await inside, keep the value.

## Where a task can be interrupted

A task is interrupted — by the scheduler, by a cancellation, by a failed
sibling — only at a point where it **actually waits**:

| Waits (a suspension point) | Does not wait |
|---|---|
| `await t` on a task not yet finished | `await t` on a finished task |
| `await sleep(d)`, any `d` (zero yields: back of the queue) | a computation, a loop, a call of a non-suspending function |
| `await ch.recv()` on an empty channel | `ch.send(v)` with room in the buffer, `trySend`, `tryRecv` |
| `ch.send(v)` on a full (or rendezvous) channel | `println` (it writes and returns; see Output) |
| a socket or TLS read or write that has to wait | `Mutex.withLock`, `with m.lock()` (a thread lock — below) |
| `race`, the end of a `scope`, `gather` | `Atomic` operations |

Code between two suspension points runs to the next one without
interruption. That is why a cancelled task that never waits is not
stopped — it runs to its end. Here the task is cancelled once before it
started and once while sending into a channel with room; the first never
runs, the second sends everything, because a send with room is not a wait:

```veles
use io

fun fill(ch: Channel<i64>, sent: Atomic<i64>) {
  loop (i in 0..<100000) {
    ch.send(i)
    sent.store(i + 1)
  }
}

fun main() {
  val early = Atomic(value: 0)
  scope {
    val t = async fill(Channel<i64>(capacity: 100000), early)
    t.cancel()   // before it got a thread: it never starts
  }
  val late = Atomic(value: 0)
  scope {
    val t = async fill(Channel<i64>(capacity: 100000), late)
    await sleep(Duration.millis(1))
    t.cancel()   // running and never waiting: it runs to the end
  }
  io.println("${early.load()} then ${late.load()}")
}
```

Output:
```text
0 then 100000
```

To make long work stoppable — and fair to other tasks on a busy machine
or under `VELES_THREADS=1` — put `await sleep(Duration.zero)` in it now
and then: it does not wait, it goes to the back of the queue, and it is
where a cancellation is seen.

`sleep(d)` sleeps **at least** `d`, rounded up to a whole millisecond
(D60); a zero or negative duration only yields.

## The end of a scope

When the body of a `scope` ends — normally or not — the scope **waits for
every task started in it**. Tasks you never awaited are waited for just
the same; their values are dropped. A failure of any of them, awaited or
not, fails the scope. So:

- `scope { async a(); async b() }` runs both and continues when both are
  done.
- Leaving the body early — `return`, `break`, a `throw` or failed `try`,
  a cancellation from further out — **cancels** the tasks still running,
  waits for them to unwind (each runs its `with` cleanups), and only then
  completes the `return` or `throw`.
- A `with t = async f()` task is the same, except that it is cancelled
  when its block ends normally too (D100): it is background work for the
  rest of the block. `await t` first if you want its result.

## When a task fails

A task fails when its function throws, or panics.

**In a `scope`: fail fast.** The first task to fail decides the error.
Then, all at once: its siblings are cancelled (each stops at its next
suspension point, running its cleanups), and the scope's body is
abandoned at its next suspension point — including a suspending call it
is inside, which is cancelled and waited for too. When all have stopped,
the error leaves the scope as if the scope had thrown it. A second task
that fails meanwhile is not reported:

```veles
use io

error Boom { n: i64 }

fun fail(n: i64, ms: i64): i64 throws Boom {
  await sleep(Duration.millis(ms))
  throw Boom(n)
}

fun run(): i64 throws Boom {
  scope {
    async fail(1, 40)
    async fail(2, 5)
  }
  0
}

fun main() {
  io.println("the scope threw Boom ${run() catch (e) { e.n }}")
}
```

Output:
```text
the scope threw Boom 2
```

Code in the body that does not wait is not interrupted: a long
computation in the body finishes first, and the failure is seen at the
body's next suspension point or at the scope's end.

**The body's own error.** If the body throws, its tasks are cancelled and
waited for, and then the body's error leaves the scope:

```veles
use io

error Boom { n: i64 }

struct Res {
  name: string
  implement Closeable {
    fun close() { io.println("closed ${this.name}") }
  }
}

fun child(opened: Channel<bool>) {
  with r = Res(name: "child's resource")
  opened.send(true)
  await sleep(Duration.seconds(1))
  io.println("never printed")
}

fun run(): i64 throws Boom {
  val opened = Channel<bool>(capacity: 1)
  scope {
    async child(opened)
    val _ = await opened.recv()   // the child holds its resource now
    throw Boom(n: 9)
  }
}

fun main() {
  io.println("run threw Boom ${run() catch (e) { e.n }}")
}
```

Output:
```text
closed child's resource
run threw Boom 9
```

A child that has not started yet when its scope is left never starts, so
it has nothing to close; that is why the body above waits for the child to
say its resource is open, rather than sleeping and hoping it has started.

**A panic** in a task is handled the same way — siblings cancelled and
cleaned up — and then continues as a panic in the task that owns the
scope, and so on outwards. Unless a `gather` turns it into a value, the
program ends with the panic's message on standard error and exit code
101.

**In `gather`: nothing is cancelled.** Every task runs to its end; each
outcome, panics included, is a `Result` in the gathered tuple (D36, D52):

```veles
use io

error Boom { n: i64 }

fun fail(): i64 throws Boom {
  await sleep(Duration.millis(1))
  throw Boom(n: 1)
}

fun slow(): i64 {
  await sleep(Duration.millis(30))
  io.println("slow finished")
  2
}

fun main() {
  val (a, b) = gather {
    async fail()
    async slow()
  }
  io.println("$a $b")
}
```

Output:
```text
slow finished
Err(error: Boom(n: 1)) Ok(value: 2)
```

To handle one task's error yourself inside a `scope`, launch a function
that *returns* a `Result` (without `throws`): its `Err` is a value, and
`await` gives the `Result`.

## Cancellation

A task is cancelled when a sibling in its `scope` fails, when its scope's
body is left early, when the block of its `with t = async …` ends, or
when `t.cancel()` is called (D20, D34, D100). Cancellation is a
**request**:

- A task cancelled before it got a thread **never starts**.
- A running task continues to its next suspension point, and **unwinds**
  there: each `with` it is inside closes, innermost first; a suspending
  call it is inside is unwound from the innermost call outwards.
- Its scope waits for the unwinding to finish.
- Cleanup is never cancelled (D47): a `close()` that is running finishes,
  even if it waits.
- Several tasks cancelled together unwind at the same time, on any
  threads, so their cleanups run in no fixed order between them.

**Do not await a task you cancelled.** `await t` after `t.cancel()`
panics with `awaited task was cancelled` — unless the task had already
finished, when it gives the value. Since which one happens depends on
timing, cancel only tasks whose result you no longer want.

## Time limits

`withTimeout(limit, f)` runs `f` in a task and races it against a timer.
When the time is up it **cancels the task and waits for it to unwind**,
then throws `Timeout` — so whatever `f` opened is closed by the time you
see the error. The same rule as above applies: a function that never
waits cannot be stopped. It runs to its end, its result is dropped, and
then `Timeout` is thrown — later than the limit:

```veles
use io

fun count(n: i64): i64 {
  var x: i64 = 0
  loop (i in 0..<n) x = x ^ (i * 31)
  x
}

fun main() {
  when (val r = withTimeout(Duration.millis(1), () => count(100000000))) {
    is Ok => io.println("finished in time")
    is Err => io.println("${r.message()}, but only once the count had finished")
  }
}
```

Output:
```text
timed out after 1ms, but only once the count had finished
```

## `race`

- The arms are tried in the order written; the first one **ready** wins.
  An arm that is always ready shadows the ones after it. If none is ready
  the race waits, and the first to become ready wins (D38).
- The losing arms simply stop waiting. A losing `recv` arm took nothing
  from its channel; a losing `send` arm sent nothing (D108); a task
  awaited by a losing arm is **not cancelled** — it keeps running, and its
  scope waits for it:

```veles
use io

fun slow(): string {
  await sleep(Duration.millis(30))
  io.println("the slow task finished anyway")
  "slow"
}

fun main() {
  scope {
    val t = async slow()
    val r = race {
      val v = await t           => v
      sleep(Duration.millis(1)) => "the timer"
    }
    io.println("won: $r")
  }
  io.println("scope done")
}
```

Output:
```text
won: the timer
the slow task finished anyway
scope done
```

- An arm awaiting a task that failed never wins; the failure is its
  scope's.

## Channels

- `Channel<T>(capacity: n)` buffers `n` values: `send` returns as soon as
  the value is buffered and waits only when the buffer is full.
  `Channel<T>()` buffers nothing: `send` returns only once a receiver has
  taken the value.
- Waiting senders and waiting receivers are each served in the order they
  arrived. Each value goes to exactly one receiver.
- `close()`: values already buffered are still received; after them,
  `recv()` returns `null` at once, for every receiver. Closing twice does
  nothing. **`send` on a closed channel panics** (`send on a closed
  channel`) — close a channel from the side that sends, after the last
  send, or use `closeAfter(n)` when several tasks send.
- `trySend` and `tryRecv` never wait: `false` when full, `null` when
  nothing is buffered (also when closed).

## Deadlock

When every task is waiting and nothing can wake any of them — no timer
pending, no socket being waited on, no thread busy — the program stops
with a panic instead of hanging:

```veles
use io

fun relay(from: Channel<i64>, to: Channel<i64>) {
  val v = await from.recv() ?: 0
  to.send(v)
}

fun main() {
  val a = Channel<i64>()
  val b = Channel<i64>()
  io.println("each task waits for the other")
  scope {
    async relay(a, b)
    async relay(b, a)
  }
}
```

Output:
```text
each task waits for the other
```

Standard error then reads `panic: deadlock: every task is blocked`, and
the exit code is 101. Only a *whole-program* deadlock is detected: if some
tasks are stuck while another still runs, sleeps on a timer or waits on a
socket (a server's accept loop, a ticker), the stuck ones wait forever.
Give such waits a limit (`race` with a `sleep` arm, `withTimeout`).

## Locks and atomics

- A `Mutex` is a real lock taken by the thread: while one task holds it,
  another task that wants it **blocks its thread** until it is free.
  Because nothing may wait while holding a lock — `withLock`'s function
  and a `with m.lock()` block cannot suspend, which the compiler checks
  (D35, D107) — a lock is held only for a short computation, never across
  a wait. If a thread stays blocked (a lock held for long, foreign code),
  the runtime notices within about a millisecond and moves that thread's
  queued tasks to a spare thread.
- Taking a `Mutex` again while the same task holds it panics (`a Mutex
  was locked again while this task holds it`) rather than hanging.
  Holding two different ones is allowed; the order is yours.
- A lock is released when its function or block ends, also by a panic.
- An `Atomic` of a number or `bool` takes no lock. `update(f)` may call
  `f` more than once (when another task changed the value in between),
  so `f` must only compute the new value.

## Threads and scheduling

- `VELES_THREADS=n` sets the number of threads that run Veles code; the
  default is the number of cores. `VELES_THREADS=1` runs every task on one
  thread — useful when debugging, and the way to see whether a program
  relies on parallelism for progress.
- Each thread has its own queue. A task woken by the running one — a
  message it was waiting for, a child that finished, a new task — runs
  next on the same thread; an idle thread takes half of a busy thread's
  queue. A yield (`sleep(Duration.zero)`) goes to the back of a shared
  queue, behind everything already waiting.
- A task may resume on a different thread after each wait. Task-local
  values (`TaskLocal`, D72) follow the task, not the thread; a task keeps
  the values bound where it was started.
- A call that blocks its thread — foreign C code, reading the terminal,
  waiting for a child process, a contended `Mutex` — is noticed by a
  monitor thread within about a millisecond, and the thread's queued
  tasks move to a spare thread until the call returns.

## Output

`io.println` writes the whole line at once, under a lock: lines from
different tasks never mix within a line, but their order between tasks is
whatever the timing makes it. `println` does not suspend. A program whose
output must be ordered collects results and prints them in one place.

## The program's end

`main` runs as the first task. The program ends when `main` returns; since
every task belongs to a scope inside `main`, none is still running then.
An error that leaves `main` is printed as `error: main failed with …` and
the exit code is 1; a panic prints `panic: …` with where it happened, and
the exit code is 101.

## What it costs

- Starting and joining a task costs about as much as a goroutine in Go:
  `bench/spawn` starts and joins 100 000 small tasks in about 40 ms, the
  same as its Go version (`bench/results.md`). A waiting task is a few hundred
  bytes — its frame, holding only the variables still needed — not a
  stack.
- A function that suspends is compiled into a state machine whose frame
  lives in its task's frame arena; a function that does not suspend is a
  plain function. A call of a suspending function runs in the caller's
  task: one that finishes without waiting costs a few nanoseconds more
  than a plain call (`bench/suscall`), and one that waits parks the task
  in its own frame and returns to the caller when it is done.
- `await` on a finished task, a channel operation that does not have to
  wait, and an uncontended `Mutex` take no lock of the runtime.

## Mistakes the compiler stops, and those it cannot

| Mistake | What happens |
|---|---|
| Starting a task that belongs to no scope | compile error (D3) |
| Keeping a handle after its scope | compile error (D141) |
| Sharing a `MutableList` or a `var` with a task | compile error (D35) |
| Waiting while holding a lock | compile error (D35, D107) |
| `try` on an `await` in a scope | compile error with a fix (D141) |
| Ignoring a task's failure | impossible: it fails the scope |
| A long loop that never waits | not stopped by cancellation or `withTimeout`; add `await sleep(Duration.zero)` |
| `await` on a task you cancelled | panic (`awaited task was cancelled`) unless it had finished |
| `send` on a closed channel | panic (`send on a closed channel`) |
| Locking a `Mutex` you already hold | panic, not a hang |
| Every task waiting for another | panic (`deadlock: every task is blocked`) |
| Some tasks waiting for each other while others run | waits forever; put a limit on the wait |
| Relying on which task prints or runs first | no order is guaranteed; collect, then print |
| Expecting `async obj.method()` to change `obj` | the task works on a copy |
| A `withTimeout` around code that never waits | the limit is noticed only when the code finishes |
| A race arm's task losing | it keeps running; the scope waits for it |
