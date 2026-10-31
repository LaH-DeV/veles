# 12. Concurrency

New to threads, coroutines and tasks? [Concurrency, explained from
scratch](concurrency-explained.md) builds the picture first. Every rule
of the runtime — what happens on each failure, cancellation, timeout or
deadlock, and the mistakes the compiler cannot catch — is collected in
[Concurrency: what happens, exactly](reference/concurrency.md).

Veles concurrency rests on two decisions. First, there is no `async`
keyword on function declarations: whether a function _suspends_ (may
pause waiting for something) is **inferred** from its body, the same way
its error type is (D2). The editor shows what was inferred: hovering a
function, at its declaration or at any call, reads `fun wait() suspends`,
and an inlay hint marks the declaration. Second, tasks are always **structured**: a task
belongs to the block that starts it — a `scope`, or the block of a
`val`/`with` that holds its handle — and that block does not finish until
the task has (D3). Nothing leaks, nothing is fire-and-forget.

Because a suspending function is called like any other, calling one and
waiting for its result needs no keyword at all: `io.println(load(path))`.
`async` means only one thing — _run this alongside what follows_.

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
- `async` also starts a function _value_ whose type is `sendable fun(...)`
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

## One task alongside: `val t = async f()`

For one or two tasks you need no `scope` (D156). A handle bound with
`val` runs alongside the rest of its block, and the compiler makes sure
the block awaits it:

```veles
use io

fun fetchRate(currency: string): i64 {
  await sleep(Duration.millis(20))   // a slow lookup
  if (currency == "EUR") 108 else 100
}

fun price(amount: i64, currency: string): i64 {
  val rate = async fetchRate(currency)   // starts now
  val fee = amount / 50                  // meanwhile
  amount * (await rate) / 100 + fee
}

fun main() {
  io.println("${price(500, "EUR")}")
}
```

Output:

```text
550
```

- **Every normal way out of the block awaits it** — falling off its end,
  `return`, and a `break` or `continue` that leaves it. A path that does
  not is a compile error; the check is the one that makes a `val` assigned
  on every path.
- **A failure on the way out stops it.** If a `try` fails, a `throw` or a
  panic leaves the block, or the task is cancelled, the handle's task is
  cancelled and joined before the error goes on, so nothing it opened is
  still open.
- **It is fail-fast**, like a scope's child: if the task throws, the rest
  of the block is cancelled at its next suspension point and the error
  leaves the block, joining the function's `throws`.
- **It is only awaited.** Passing the handle to a function, storing it,
  capturing it in a lambda or cancelling it is an error: the compiler could
  no longer see that it is awaited. A task you may stop is a `with` task
  (below); one you hand around belongs in a `scope`.

```veles
// fragment
fun quote(c: bool): i64 {
  val rate = async fetchRate("EUR")
  if (c) return 0        // error: 'rate' is not awaited on this path
  await rate
}
```

The compiler also warns when nothing runs alongside a task — `val t =
async f()` followed straight by `await t` is `f()` with extra steps:
"'f' runs with nothing alongside it; call 'f()' directly". (In a
`scope(on: e)` it does not warn: there the task is what moves `f` onto
`e`.)

## Several values: `awaitAll`

`awaitAll` waits for several tasks and gives their values as a tuple, in
argument order (D156). An argument is a handle or an `async` call written
in place:

::: code-group

```veles [Veles]
use io

error NotFound { id: i64 }

fun user(id: i64): string throws NotFound {
  await sleep(Duration.millis(10))
  if (id < 0) throw NotFound(id)
  "user$id"
}

fun orders(id: i64): i64 {
  await sleep(Duration.millis(5))
  id * 3
}

fun summary(id: i64): string throws NotFound {
  val (name, count) = awaitAll(async user(id), async orders(id))
  "$name has $count orders"
}

fun main() {
  io.println(summary(7) catch (e) { "no such user" })
  io.println(summary(-1) catch (e) { "no such user ${e.id}" })
}
```

```go [Go]
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type NotFound struct{ ID int64 }

func (e *NotFound) Error() string { return fmt.Sprint("no user ", e.ID) }

func user(ctx context.Context, id int64) (string, error) {
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if id < 0 {
		return "", &NotFound{ID: id}
	}
	return fmt.Sprint("user", id), nil
}

func orders(ctx context.Context, id int64) int64 {
	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
	}
	return id * 3
}

// both at once; the first failure cancels the other through the context
func summary(id int64) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type named struct {
		name string
		err  error
	}
	nameCh, countCh := make(chan named, 1), make(chan int64, 1)
	go func() {
		name, err := user(ctx, id)
		if err != nil {
			cancel()
		}
		nameCh <- named{name, err}
	}()
	go func() { countCh <- orders(ctx, id) }()
	n, count := <-nameCh, <-countCh
	if n.err != nil {
		return "", n.err
	}
	return fmt.Sprintf("%s has %d orders", n.name, count), nil
}

func main() {
	for _, id := range []int64{7, -1} {
		s, err := summary(id)
		var nf *NotFound
		if errors.As(err, &nf) {
			s = fmt.Sprint("no such user ", nf.ID)
		}
		fmt.Println(s)
	}
}
```

```ts [TypeScript]
class NotFound extends Error {
  id: number;
  constructor(id: number) {
    super(`no user ${id}`);
    this.id = id;
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

async function user(id: number): Promise<string> {
  await sleep(10);
  if (id < 0) throw new NotFound(id);
  return `user${id}`;
}

async function orders(id: number): Promise<number> {
  await sleep(5);
  return id * 3;
}

// Promise.all rejects on the first failure, but the other promise keeps running
async function summary(id: number): Promise<string> {
  const [name, count] = await Promise.all([user(id), orders(id)]);
  return `${name} has ${count} orders`;
}

for (const id of [7, -1]) {
  try {
    console.log(await summary(id));
  } catch (e) {
    if (e instanceof NotFound) console.log(`no such user ${e.id}`);
    else throw e;
  }
}
```

```swift [Swift]
struct NotFound: Error { let id: Int }

func user(_ id: Int) async throws(NotFound) -> String {
    try? await Task.sleep(for: .milliseconds(10))
    if id < 0 { throw NotFound(id: id) }
    return "user\(id)"
}

func orders(_ id: Int) async -> Int {
    try? await Task.sleep(for: .milliseconds(5))
    return id * 3
}

// `async let` starts both; leaving the scope by a throw cancels the other
func summary(_ id: Int) async throws -> String {
    async let name = user(id)
    async let count = orders(id)
    return "\(try await name) has \(await count) orders"
}

for id in [7, -1] {
    do {
        print(try await summary(id))
    } catch let e as NotFound {
        print("no such user \(e.id)")
    }
}
```

:::

Output:

```text
user7 has 21 orders
no such user -1
```

- The calls run at once; `awaitAll` returns when every one has finished.
- The first failure **cancels the others** and leaves from `awaitAll` —
  unlike JavaScript's `Promise.all`, which leaves them running.
- `awaitAll(tasks)` over a `List` or `MutableList` of handles gives a
  `List` of their values, in order.
- There is no `await` in front: `awaitAll` is itself the point where the
  task waits, as `select` is.

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
  failure fails the scope. `await` is how the body _uses_ a task's value
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
finished. A task that should run only _while_ something else happens —
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
  as the whole value of a `val` (awaited, above) or a `with`, as an
  argument of `awaitAll`, and as a field of a value that holds a task
  (below). Inside a `scope`, a `with`-task is cancelled at the end of its
  own block, before the scope waits for its other children.
- `val` or `with`? A `val` task is one the block needs: every path awaits
  it. A `with` task is one the block may leave behind: it is stopped when
  the block ends.

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
  io.println("${a.toString()}")
  io.println("${b.toString()}")
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

A task's error is the `Err` only when its function `throws`. A function
that _returns_ a `Result` finished either way, so its `Result` is the
task's value: `async check(n)` with `check(n: i64): Result<i64, Boom>`
gives a `Result<Result<i64, Boom>, Panic>`, where `Ok(Err(…))` is "check
said no" and `Err(Panic(…))` is "check crashed" (D155) — as in a `scope`,
where awaiting it gives the `Result`.

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
    val jobsDone = done.map(line => line.splitOnce(" did ")?.after ?: "?")
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

### One result at a time: the lazy `mapConcurrent`

`list.mapConcurrent` collects every result before it returns. On an
iterator, `mapConcurrent` is lazy (D46's split): it keeps up to `workers`
calls under way and gives the results **in order, as the loop asks for
them**, so the loop handles one while the next ones are computed (D156):

```veles
use io

fun download(part: i64): string {
  await sleep(Duration.millis(10 * (4 - part)))   // later parts are quicker
  "part $part"
}

fun main() {
  val parts = [1, 2, 3]
  loop (data in parts.iter().mapConcurrent(p => download(p), workers: 2)) {
    io.println("saved $data")
  }
}
```

Output:

```text
saved part 1
saved part 2
saved part 3
```

- The elements are taken from the iterator in the task that loops, so
  the iterator need not cross a task; each element must be Sendable.
- The value holds the tasks running the calls (D111), and **the loop owns
  them**: when it ends — exhausted, `break`, `return`, a failure — the calls
  still under way are cancelled and joined. Anywhere else it is received
  with `with`, as any value holding a task is.
- The first error `f` throws fails the block that owns the loop.
- Its type is an `AsyncIterator`: an iterator whose `next()` may wait
  (D42). Any type implementing that trait is looped over the same way, and
  the loop's function suspends.

## A pool of your own: `scope(workers: n)`, and a scope's value

`scope(workers: n)` runs at most `n` of its children at once: when `n`
are running, the next `async` waits for one of them to finish, so a
scope can be handed a million elements without holding a million tasks
(D156). It combines with `on:` — an executor's threads limit how much
runs on the CPU, `workers` how many tasks are under way. And a `scope`
may be used as a value: its body's last expression, once every task it
started has finished.

```veles
use io

fun lines(name: string): i64 {
  await sleep(Duration.millis(2))
  name.len() * 10
}

fun main() {
  val files = ["a.txt", "notes.md", "main.vs", "readme"]
  val total = scope(workers: 2) {
    val tasks: MutableList<Task<i64>> = []
    loop (f in files) tasks.push(async lines(f))
    awaitAll(tasks).sum()
  }
  io.println("$total lines")
}
```

Output:

```text
260 lines
```

- `workers` is evaluated once, before the scope starts; below 1 it
  panics (a constant below 1 is a compile error).
- It counts the scope's own children; a `with` task or a nested `scope`
  in the body is a child of its own block.
- `gather(workers: n)` limits a gather the same way.
- A `scope(on: e)` or `scope(workers: n)` whose body starts no task is
  an error: the body itself runs in the calling task, so `on:` would move
  nothing — to run code on an executor, write `e.run(() => …)`.

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

`race { }` is written like `gather`: its body starts tasks, and they race.
The first to finish gives the race its value; the others are cancelled,
and waited for until their cleanups have run, before the race returns
(D169). Use it for two mirrors of one file, a cache and the database
behind it, a request and its hedged copy:

```veles
use io

fun mirror(name: string, delay: Duration): string {
  await sleep(delay)
  name
}

fun main() {
  val name = race {
    async mirror("eu", Duration.millis(30))
    async mirror("us", Duration.millis(5))
  }
  io.println("fastest: $name")

  // a hedged request: the backup starts only if the primary is slow
  val page = race {
    async mirror("primary", Duration.millis(200))
    async {
      await sleep(Duration.millis(20))
      mirror("backup", Duration.millis(5))
    }
  }
  io.println("served by $page")

  // the first of a list
  val first = race {
    loop (n in ["a", "b", "c"]) async mirror(n, Duration.millis(if (n == "b") 1 else 40))
  }
  io.println("first of three: $first")
}
```

Output:

```text
fastest: us
served by backup
first of three: b
```

- Every task the body starts is a contestant — `async f()` or
  `async { … }`, in a loop or an `if` — except one inside a nested
  `scope`, `gather`, `race` or `with`. They give one type, which is the
  race's.
- The first to finish wins **even if it failed**: then the race throws
  its error, after cancelling the rest, as a `scope` does. The error
  leaves the function, so a race that can fail goes in a function you
  `try` or `catch`.
- A timeout is one more contestant — `async { await sleep(d); throw
Timeout(limit: d) }` — or `withTimeout` around the race.
- A body that starts no task is a compile error, or a panic when a loop
  started none; a race of one task is a warning (call it directly).

## Waiting on several things: `select`

`select` waits on several sources at once — channels, timers, tasks,
events — and runs the **arm** of the first that is ready (D38, D169). Each
arm is `source => body`; `val name = source => body` names what the
source gave. No arm takes `await`: the select is where the task waits.

```veles
use io

fun producer(ch: Channel<string>) {
  await sleep(Duration.millis(5))
  ch.send("hello")
}

fun lookup(key: string): string {
  await sleep(Duration.millis(50))
  "value of $key"
}

fun main() {
  val ch = Channel<string>(capacity: 1)
  scope {
    async producer(ch)
    val first = select {
      val msg = ch.recv()        => "got ${msg ?: "closed"}"
      sleep(Duration.seconds(1)) => "timeout"
    }
    val second = select {
      val msg = ch.recv()       => "got ${msg ?: "closed"}"
      sleep(Duration.millis(2)) => "timeout"
    }
    io.println("$first / $second")
  }
  val answer = select {
    val v = async lookup("k")  => v
    sleep(Duration.millis(10)) => "too slow"
  }
  io.println(answer)
}
```

Output:

```text
got hello / timeout
too slow
```

**Whose task.** A task an arm starts (`async lookup("k")`) belongs to the
select: when another arm wins it is cancelled, and the select waits for it
to stop before the winning arm's body runs; if it fails first, the select
throws its error. A task started elsewhere (`val t = async f()` before the
select, a handle in a `scope` body) belongs to the block that started it:
an arm `val v = t => …` waits for it, but if it loses it is not cancelled
and runs on. A task belongs to the place its `async` is written.

An arm can also **send**: `ch.send(v) => …` is ready when the channel can
take `v` — room in its buffer, or a receiver waiting (D108). When it wins,
`v` is in the channel; when another arm wins, `v` was **not** sent. That
is the bounded send with a deadline, without a task to do the sending:

```veles
use io

fun offer(queue: Channel<string>, line: string): bool {
  select {
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

The channel and the value are evaluated once, in arm order, when the
select starts, not again while it waits. A send, sleep, `wait()` or
`changed()` arm has no value to name. A send arm on a channel that is
closed when it would be chosen panics, as `send` does.

**Which arm wins.** When the select starts, the arms are tried in the
order they are written, and the first one ready wins — so an arm that is
always ready (a `sleep(Duration.zero)`, a channel with a value waiting)
shadows the arms after it. If none is ready, the select waits, and the
first arm to become ready wins. The rule is deterministic and favours
earlier arms; it is not fair in Go's sense (Go picks among ready cases at
random).

## A task from a block: `async { }`

`async { … }` starts the block as a task, like `async f()` for a function
written in place (D169). It stands wherever `async f()` may — in a `scope`
or `race` body, as `val t = async { … }` that the block awaits, as
`with t = async { … }`, as a `select` arm — and awaiting it gives the
block's value. `return v` ends the block with `v`; it cannot leave the
enclosing function, which may have moved on.

```veles
use io

fun main() {
  val total = async {
    var sum = 0
    loop (i in 1..100) sum += i
    if (sum < 0) return -1
    sum
  }
  io.println("adding in the background")
  io.println("total ${await total}")
}
```

Output:

```text
adding in the background
total 5050
```

The block crosses into another task, so what it uses from outside follows
the rule of every task (D35): `val`s of Sendable types. A `var` it would
change is refused — share it through an `Atomic` or a `Mutex`, or return
the result and assign it after `await`.

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

A tight counted loop — over a range or a list, its body calling no
function — makes that check once per chunk of up to 1024 rounds instead
(D218), about a microsecond of work: the check would otherwise keep the
compiler from unrolling and vectorizing it, and nothing can tell the two
apart but a clock.

### Giving the executor a turn: `cede`

A task keeps its thread until it reaches a suspension point, and a loop in
a suspending function gives it up by itself once it has run for about
10 ms while others wait (D145). The statement `cede` is a suspension point
you place: the task offers its executor a turn and goes on at the next
statement when it is picked again — which may be at once (D197). It suits
a long computation that shares an executor with tasks that should not wait
for all of it, given a turn between its chunks:

```veles
use io

/// FNV-1a over the data, offering the executor a turn after each 64 KiB
fun fnv1a(data: List<u8>): u64 {
  var hash: u64 = 0xcbf29ce484222325
  loop (chunk in data.chunked(65536)) {
    loop (b in chunk) hash = (hash ^ b.toU64()) *% 0x100000001b3
    cede
  }
  hash
}

fun main() {
  val data = MutableList<u8>.make(1 << 20, i => (i % 251).wrapU8())
  val digest = async fnv1a(data)
  io.println("hashing alongside")
  io.println("${await digest:x}")
}
```

Output:

```text
hashing alongside
4c568eccaeaf6c44
```

- `cede` is a statement, alone on its line or before `;`, `}` or `else`:
  it has no value and takes nothing. `cede()`, `await cede`,
  `val x = cede` are errors that say so, and an expression body is
  written as a block: `fun pause() { cede }`, `() => { cede }`,
  `async { cede }`.
- A function that cedes suspends, and so does every caller, as with any
  suspension point; it is refused where suspending is — in a lock region,
  a global initializer, a `const fun`, a callback whose type does not
  suspend.
- It is where a cancellation is seen and where a scope body notices a
  failed child. In a `close()` that suspends it still gives the turn, but
  the cancellation waits until the close is done (D47).
- It promises nothing about who runs next — not the oldest waiting task,
  not every other task first — and no delay. It is not synchronisation:
  to wait for something, wait on it (a channel, an `await`, an `Event`);
  a loop that cedes until a flag changes only burns the thread.
- The task stays where it is: on an `Executor.thread` it goes on on that
  thread. `await sleep(Duration.zero)` does the same as `cede`;
  `checkCancelled()` checks for a cancellation without giving a turn.

### Time limits: `withTimeout`

The prelude's `withTimeout(limit, f)` calls `f` and gives up once `limit`
has passed by throwing `Timeout`; `f`'s own errors are rethrown, so the
call throws `E | Timeout`. `f` runs in the calling task, as an ordinary
call (D151): it may use anything the caller can, a `MutableList` included,
and starts no task:

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

When the time is up, `f` is stopped where it waits, or at the end of a
loop iteration — also in a function that never waits, on any number of
threads — and unwinds as a cancelled task does: the scopes inside it
cancel and join their tasks, its `with`s close, innermost first, and only
then does `withTimeout` throw. So whatever `f` had open is closed by the
time you see the `Timeout`. Limits nest, each stopping its own `f`; a
cancellation of the whole task passes through as one. A limit that passes
while `f` is inside a lock region (where nothing is interrupted) still
ends in `Timeout`, once `f` is done.

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

On a parameter, `suspends` means _may_ suspend (D116). A function whose
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
"suspends if `f` does". A function that _keeps_ the parameter — stores it,
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

|                                            |                                                                                |
| ------------------------------------------ | ------------------------------------------------------------------------------ |
| `load()`, `store(v)`, `swap(v)`            | read, replace, replace and return the old value                                |
| `add(n)`, `sub(n)`                         | an integer's counter step; returns the new value, wraps on overflow            |
| `fetchAnd(m)`, `fetchOr(m)`, `fetchXor(m)` | change an integer's bits; return the old value                                 |
| `compareAndSet(expected, new)`             | store `new` only if the value is still `expected`; says whether it did         |
| `compareExchange(expected, new)`           | the same, returning the value it found                                         |
| `update(f)`                                | replace the value with `f(value)`, retrying until no other task got in between |

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
For the same reason a module-level `val`, and a `static val`, must have a
`Sendable` type. A `MutableMap` in a `val` is never reassigned, but its
methods change it. Module state that changes is a `val` holding a lock:

```veles
// fragment
var served = 0              // error: module-level 'var served' is shared by every task …
var lock = Mutex(value: 0)  // error: … the lock is inside the value, so declare it 'val' and change what it holds: 'lock.set(...)'
val seen = MutableMap<string, i64>()   // error: … 'MutableMap<string, i64>', which is not Sendable …
val served = Atomic(value: 0)      // ok: served.add(1)
val cache = Mutex(value: MutableMap<string, string>())
lazy val table = buildTable()                // ok: built on its first read (below)
val ports = ["http": 80, "https": 443]       // ok: an immutable Map is Sendable
```

No module global is a `var`, not even one holding a lock: replacing the
`Mutex` itself would let a task holding the old one run beside a task
taking the new one. The value changes through its own methods
(`served.store(0)`, `cache.set(…)`). A global of a trait type, `val clock:
Clock = SystemClock()`, needs a
trait that requires `Sendable` of every implementor (`trait Clock :
Sendable`). State that only one thread uses can be kept in an executor
without a lock (D150). A resource — a database pool, a client, a file —
is not a module value at all: it is made in `main` and handed down
(chapter 26).

### Built on first read: `lazy val`

A module `val` is built before `main`, in the order the source gives,
whether or not the program needs it. One that is expensive — a table, a
parsed embedded file, a compiled set of patterns — can be `lazy` instead:
built on its first read, once, by whichever task gets there first, while
the others wait for it (D210). A read is a plain name.

```veles
use io

lazy val keywords = buildKeywords()

fun buildKeywords(): Set<string> {
  io.println("building")
  ["fun", "val", "var", "loop"].toSet()
}

fun isKeyword(word: string): bool => keywords.contains(word)

struct Zone {
  name: string

  static lazy val utc = Zone(name: "UTC")
}

fun main() {
  io.println("start")
  io.println("${isKeyword("val")} ${isKeyword("x")} ${Zone.utc.name}")
}
```

Output:
```text
start
building
true false UTC
```

The rules are `Lazy`'s, below: a panic in the initializer is raised again
by every later read, and the initializer may not suspend or throw — it
runs inside whichever read comes first, which neither waits nor handles
an error — so a value that needs either is built in `main` and passed
down. Its type must be `Sendable`, like any module value's, and it takes
part in the order of the module's values (a cycle is still an error). A
`const` cannot read one. Once built, a read is one load of a flag, a
branch and the value. `lazy` goes only on a module `val` or a `static
val`: on a field it would put a flag in every value, so a field, a local or
a value passed around holds a `Lazy<T>` and calls `get()`.

## More shared state: `RwLock`, `Event`, `Lazy`, `Broadcast`, `Watch`

Five more types cover the other shapes shared state takes (D146). Each is
`Sendable` when what it holds is, and copies of one are the same lock,
event or feed.

| Type           | For                                                          | Operations                                                                                  |
| -------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------- |
| `RwLock<T>`    | read often, changed rarely: many readers at once, one writer | `with c = l.read()`, `with w = l.write()`, `withRead(f)`, `withWrite(f)`, `get()`, `set(v)` |
| `Event`        | "it happened": tasks wait until one sets it                  | `set()`, `reset()`, `isSet()`, `await e.wait()`                                             |
| `Lazy<T>`      | a value built once, on first use, by whoever asks first      | `Lazy(init: f)`, `get()`, `isReady()`                                                       |
| `Broadcast<T>` | every receiver gets every value                              | `send(v)`, `subscribe()`, `close()`; `try await sub.recv()`                                 |
| `Watch<T>`     | the latest value of something that changes                   | `set(v)`, `get()`, `await w.changed()`                                                      |

`RwLock`'s `read()` and `write()` follow `Mutex.lock()`'s rules: usable
only as a `with` value, nothing in the region may suspend, and taking the
same `RwLock` again inside it panics. A writer that waits keeps new
readers out, so it is never starved. `Lazy`'s `init` runs under its lock:
tasks that ask meanwhile wait, and a panic in it is the panic of every
`get` from then on.

The waiting operations — `wait()`, `changed()`, `recv()` — always suspend,
so they are awaited, like a channel's `recv`; a cancelled task stops
waiting, and each can be a `select` arm. A `Broadcast` keeps its last
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

| Tool                                     | For                                                                                          |
| ---------------------------------------- | -------------------------------------------------------------------------------------------- |
| `try Executor.pool(threads: n, name: s)` | CPU-heavy work that must not take the threads a server answers on                            |
| `try Executor.thread(name: s)`           | a library that must always be called from the same OS thread (a GL context, COM, a GUI loop) |
| `blocking(f)`                            | one call known to block its thread, such as a synchronous C read                             |
| `try Thread.start(name: s, f: f)`        | plain code on an OS thread of its own: an audio loop, a pinned worker                        |

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
  io.println("${a.toString()} ${b.toString()}")

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
- **Plain code waiting for tasks.** `e.runBlocking(f)` runs `f` as a task
  on `e` and blocks the calling OS thread until it returns (D152) — how a
  `Thread`'s function, a `blocking` call or a C library's thread hands
  work to tasks and waits for the answer:
  `Executor.default.runBlocking(() => events.send(i))` (`Executor.default`
  is the default pool). A task waits with `run` instead: in a function
  that suspends, `runBlocking` is a compile error, and on a thread of an
  executor it panics, since it would hold a thread the tasks need, or wait
  for itself.
- **`blocking`** runs `f` on a pool that starts threads as calls need
  them, up to 128 (a call beyond that waits for one), and ends a thread
  idle for 10 seconds. `f` does not suspend. A blocking call made
  without it is still noticed after about a millisecond, and the
  thread's other tasks moved to a spare thread — in the default pool and
  in a pool alike (a pool's spare has its name, priority and CPUs), but
  never on an `Executor.thread`, whose tasks stay on its one thread;
  `blocking` says so up front.
- **Timers and sockets** are served by the default pool for every
  executor: a task on a pool can sleep, select and read a socket like any
  other — also while every default thread is busy in a long loop (the
  monitor steps in).

### An executor's own state: `executor`

A GL context, a GUI or a C library bound to its thread keeps state for
the life of the program, used by many callers. Declared at module level,
an **executor** holds that state and the functions that use it (D150):

```veles
use io

executor gl {
  var uploads: i64 = 0                            // made on gl's own thread
  val textures: MutableMap<string, i64> = [:]     // not Sendable, and needs no lock

  public fun texture(name: string): i64 {
    textures.get(name) ?: upload(name)            // a call between gl's functions: direct
  }

  fun upload(name: string): i64 {
    uploads += 1
    textures.set(name, uploads)
    uploads
  }

  public fun preload(names: List<string>): i64 {
    val done: MutableList<string> = []
    scope {
      loop (n in names) async mark(done, n)       // children stay on gl: no Sendable check
    }
    done.len()
  }

  public fun count(): i64 => uploads
}

fun mark(done: MutableList<string>, name: string) {
  await sleep(Duration.millis(1))
  done.push(name)
}

executor cpu(threads: 2)                          // a named pool

fun main() {
  io.println("${gl.texture("grass")} ${gl.texture("stone")} ${gl.texture("grass")}")
  io.println("preloaded ${gl.preload(["a", "b", "c"])}, ${gl.count()} uploads")
  scope(on: cpu) {
    async io.println("on cpu")
  }
}
```

Output:

```text
1 2 1
preloaded 3, 2 uploads
on cpu
```

- **Its code** is its functions, the initializers of its state, and the
  lambdas in them that cannot leave it (a `sendable` one may run on any
  thread, so it is not). Only its code names its state; a module-level
  `var` stays an error (D66), an executor's is not, since one thread runs
  everything that names it.
- **From its code** a call to another of its functions is an ordinary
  call, and a child launched in a `scope` without `on:` stays on it, so
  what it is given need not be `Sendable`.
- **From anywhere else** `gl.texture("grass")` runs the function on gl
  and suspends the caller until it returns: the arguments are evaluated
  first, in the caller, and must be `Sendable`, as must the result; the
  function's errors are the caller's to handle. A plain function cannot
  make the call (it would have to block its thread: `gl.runBlocking`).
- `executor name(threads: n, priority:, cpus:)` takes `Executor.pool`'s
  arguments, `name:` being the declaration's: with `threads:` it is a
  pool, which holds functions but no state. Executors start with the
  module's globals; a one-thread executor's state is made on its own
  thread, before `main`; a priority or CPU the OS refuses ends the
  program there, there being no caller to throw to. A declared executor
  runs as long as the program and is not closed by code.
- `gl` is an `Executor` value too: `scope(on: gl)`, `gl.run(f)` — a lambda
  handed straight to its `run` is its code.

## How it compiles

A suspending function becomes a state machine: its locals move into a
heap-allocated frame and each `await` is a resume point (this is why
the design calls them _stackless coroutines_). A function that does not
suspend is a plain function with a plain stack frame; deciding which is
which is the inference pass. Trait methods declare `suspends` explicitly
(D40) so that callers through the trait know which calling convention to
use.

A task gives up its thread at a suspension point, and a loop in a
suspending function also gives it up by itself once it has run for about
10 ms while other tasks wait (D145). A function that does not suspend
keeps its thread until it returns (the other threads go on); `cede` in
its loop gives the thread up, and makes the function suspend. A call that
blocks the thread itself — a C
function, a read from the terminal, waiting for a child process — is
noticed by a monitor within about a millisecond, and the thread's queue
of tasks goes to a spare thread until the call returns (D66).

Next: [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md).
