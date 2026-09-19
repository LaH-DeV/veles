# 12. Concurrency

Veles concurrency rests on two decisions. First, there is no `async`
keyword on function declarations: whether a function *suspends* (may
pause waiting for something) is **inferred** from its body, the same way
its error type is (D2). Second, tasks are always **structured**: a task
is started inside a `scope`, and the scope does not finish until every
task in it has (D3). Nothing leaks, nothing is fire-and-forget.

## Starting tasks

```veles
use io

fun work(id: i64, ms: i64): i64 {
  await sleep(ms)            // suspends here; the executor runs other tasks
  io.println("task $id done")
  id * 10
}

fun main() {
  scope {
    val a = async work(1, 20)
    val b = async work(2, 10)
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
- `sleep(ms)` is a primitive that always suspends, so it is awaited
  explicitly. Calling `work` makes `main` a suspending function too —
  you did not have to say so anywhere.
- `await` is only written on the primitives that are known to suspend
  (`sleep`, `recv`, task handles); a call to an ordinary function that
  happens to suspend needs nothing (D16).

The bootstrap executor is single-threaded: tasks interleave at
suspension points, in order. Everything in this chapter is written so
that it stays correct when the executor becomes parallel.

## Fail fast

If a task throws or panics, its scope **cancels its siblings** and
re-raises the failure once they have stopped (D34/D52):

```veles
use io

error Boom { n: i64 }

fun mayFail(n: i64): i64 throws Boom {
  await sleep(1)
  if (n == 2) throw Boom(n)
  n * 10
}

fun slow() {
  loop (i in 0..<100) {
    io.println("slow tick $i")
    await sleep(1)
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
slow tick 1
```

The program ends with `error: main failed with Boom(n: 2)` on standard
error. Cancellation is checked at every suspension point, so `slow()`
stops at its next `await sleep`. A task that never suspends cannot be
cancelled — and does not need to be, since it also cannot block anyone.

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
  await sleep(1)
  if (n == 2) throw Boom(n)
  n * 10
}

fun crashes(): i64 {
  val xs = [1]
  xs.atOrPanic(5)
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
panic: index 5 out of bounds for list of length 1
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
    await sleep(1)
    results.send("$name did ${job.id}")
  }
}

fun main() {
  val jobs = Channel<Job>(capacity: 8)
  val results = Channel<string>(capacity: 8)
  scope {
    async worker("a", jobs, results)
    async worker("b", jobs, results)
    loop (i in 1..4) { jobs.send(Job(id: i)) }
    jobs.close()
    var done: MutableList<string> = []
    loop (_ in 1..4) {
      val r = await results.recv()
      if (r != null) done.push(r)
    }
    io.println("${done.len()} results, first ${done.sorted().first() ?: ""}")
  }
}
```

Output:
```text
4 results, first a did 1
```

## First one wins: `race`

`race` waits for whichever arm is ready first and cancels the rest.
Arms can receive from channels, sleep (a timeout), or await tasks (D38):

```veles
use io

fun producer(ch: Channel<string>) {
  await sleep(5)
  ch.send("hello")
}

fun main() {
  val ch = Channel<string>(capacity: 1)
  scope {
    async producer(ch)
    val first = race {
      val msg = ch.recv() => "got ${msg ?: "closed"}"
      sleep(1000) => "timeout"
    }
    val second = race {
      val msg = ch.recv() => "got ${msg ?: "closed"}"
      sleep(2) => "timeout"
    }
    io.println("$first / $second")
  }
}
```

Output:
```text
got hello / timeout
```

## What may cross a task boundary

Data handed to `async` must be **Sendable** (D35): numbers, strings,
immutable collections, structs whose fields are Sendable, channels,
`Mutex` and `Atomic`. A `MutableList` is not — two tasks could then
mutate it — and the compiler refuses the `async`. This is the concrete
reason `List` and `MutableList` are separate types.

For state that genuinely must be shared and mutated, wrap it:

```veles
use io

struct Counter { hits: i64 }

fun bump(m: Mutex<Counter>, times: i64) {
  loop (_ in 0..<times) {
    m.withLock(c => c.hits += 1)
    await sleep(0)
  }
}

fun main() {
  val shared = mutex(Counter(hits: 0))
  val total = atomic(0)
  scope {
    async bump(shared, 5)
    async bump(shared, 7)
  }
  total.store(shared.get().hits)
  io.println("hits ${shared.get().hits} atomic ${total.load()}")
}
```

Output:
```text
hits 12 atomic 12
```

## How it compiles

A suspending function becomes a state machine: its locals move into a
heap-allocated frame and each `await` is a resume point (this is why
the design calls them *stackless coroutines*). A function that does not
suspend is a plain function with a plain stack frame; deciding which is
which is the inference pass. Trait methods declare `suspends` explicitly
(D40) so that callers through the trait know which calling convention to
use.

Next: [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md).
