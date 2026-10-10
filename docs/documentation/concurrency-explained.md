# Concurrency, explained from scratch

This page is for you if words like *thread*, *coroutine* or *green
thread* make your eyes glaze over. No background is assumed beyond
having written some JavaScript (or Python) — if you have ever typed
`await fetch(...)`, you already know half of it. The full reference is
[chapter 12](12-concurrency.md); this page is the picture behind it.

## The problem: most programs spend their time waiting

Imagine a program that downloads three files. Each download is mostly
*waiting* — for the network, for the other computer to answer. The
simple way does one after another:

```veles
use io

fun download(name: string): string {
  await sleep(Duration.millis(200))   // pretend the network is slow
  "contents of $name"
}

fun main() {
  val a = download("a.txt")
  val b = download("b.txt")
  val c = download("c.txt")
  io.println("$a | $b | $c")
}
```

Output:
```text
contents of a.txt | contents of b.txt | contents of c.txt
```

That takes about 600 ms, and for almost all of it the computer does
nothing. The three waits could overlap — start all three downloads,
then collect the answers — and the whole thing would take about 200 ms.
Doing several things so their waiting overlaps is **concurrency**.
Here is the same program, concurrent:

```veles
use io

fun download(name: string): string {
  await sleep(Duration.millis(200))
  "contents of $name"
}

fun main() {
  scope {
    val a = async download("a.txt")
    val b = async download("b.txt")
    val c = async download("c.txt")
    io.println("${await a} | ${await b} | ${await c}")
  }
}
```

Output:
```text
contents of a.txt | contents of b.txt | contents of c.txt
```

Same answer, a third of the time. Three new words did it: `scope`,
`async` and `await`. The rest of this page is what they mean and what
happens underneath.

## The words, in plain language

| Word | What it is | Do you write it? |
|---|---|---|
| **Thread** | A worker the *operating system* gives your program. It can run code on its own CPU core, at the same moment as other threads. Threads are expensive: each one reserves memory for its stack and takes the OS a while to create, so a program has a handful, not thousands. | Never. Veles creates them for you. |
| **Task** | One job your program is doing: "download a.txt". You start one with `async`. Tasks are cheap — a few hundred bytes, well under a microsecond to start — so 100 000 of them is fine. | Yes: `async f(x)`. |
| **Suspending** | A task saying "I am waiting now; give my thread to someone else". It happens at an `await`. | Yes: `await ...`. |
| **Coroutine** | A function that can stop in the middle and continue later, right where it stopped, with its local variables intact. Every Veles function that may suspend is compiled into one. It is the *bookmark* that lets a task pause. | No — the compiler decides. |
| **Green thread** | A "thread" faked by a language's runtime instead of the OS: cheap, so you can have millions (Go's goroutines, Java's virtual threads). Veles tasks play exactly this role. | No. |
| **Executor** (scheduler) | The manager inside your program that hands tasks to threads, notices when a waiting task can continue, and puts it back in line. | No — it starts with your program. |

The one sentence to remember: **you write tasks; the executor runs them
on threads; `await` is where a task lets go of its thread.**

## A kitchen

Think of a restaurant kitchen.

- The **cooks** are threads. There are only a few, one per stove (a
  stove is a CPU core).
- The **orders** are tasks. There can be hundreds, pinned on a rail.
- A **recipe** is your function; the order remembers how far along it
  is (that is the coroutine's bookmark).
- When a dish goes into the oven for 20 minutes, the cook does *not*
  stand and stare at the oven. They pin the order back on the rail with
  a note "oven, 20 min" and pick up another order. That is `await`.
- When the oven timer rings, the order goes back in line, and **whichever
  cook is free** finishes it — not necessarily the one who started it.
  A Veles task can continue on a different thread than the one it began
  on, and you never notice.

```text
cook 1:  [soup: chop]──oven──►            [toast: butter]   [soup: serve]
cook 2:        [toast: slice]──toaster──►  [salad: all of it]
rail:    soup waits for the oven, toast for the toaster, salad for a cook
```

Here is that kitchen as a program. Each dish takes its own time; the
cooks never wait for one dish while another could be made:

```veles
use io

fun cook(dish: string, minutes: i64) {
  await sleep(Duration.millis(minutes * 50))   // the oven: nobody stands here
  io.println("$dish is ready")
}

fun main() {
  scope {
    async cook("soup", 3)
    async cook("toast", 1)
    async cook("pie", 5)
    io.println("orders placed")
  }
  io.println("kitchen closed")
}
```

Output:
```text
orders placed
toast is ready
soup is ready
pie is ready
kitchen closed
```

Notice the order: the toast was ordered second but is ready first,
because it spent the least time in the oven. And `orders placed` comes
before any dish, because placing an order (`async`) does not wait for
it. `kitchen closed` comes last because a `scope` does not end until
every task started in it has — the kitchen does not close with food in
the oven.

## If you know JavaScript

JavaScript has **one cook**. Everything runs on a single thread with an
event loop: `await` hands the thread back to the loop, which runs other
callbacks until the awaited thing is ready. JavaScript's `async`
functions are coroutines too — so the idea is the same. Three things
differ:

**1. Several cooks.** Veles runs tasks on one thread *per CPU core* at
the same time. A heavy computation in one task does not freeze the
others, the way a long loop freezes a web page. (JavaScript's answer to
that is Web Workers, which cannot share memory; Veles tasks share one
heap, safely — see "sharing" below.)

**2. No `async` on functions.** In JavaScript, once one function is
`async`, every function that calls it must become `async` and `await`
it — the "what colour is your function" problem. In Veles the compiler
works out on its own which functions may suspend. `download` above has
no marker; it suspends because it contains an `await`, and `main`
suspends because it calls `download`. You only write `await` in front of
the few things that *always* wait: `sleep(...)`, `channel.recv()`, and a
task handle.

| JavaScript | Veles |
|---|---|
| `async function f() { ... }` | `fun f() { ... }` — the compiler infers it |
| `const p = f()` (starts running) | `val t = async f()` |
| `await p` | `await t` |
| `await Promise.all([a, b])` | `await a`, `await b` inside one `scope` — or `gather` |
| `Promise.race([...])` | `race { ... }` |
| `setTimeout(fn, 100)` | `await sleep(Duration.millis(100))` in a task |
| an unawaited promise that fails silently | impossible: every task belongs to a `scope` |

**3. Nothing floats away.** In JavaScript you can start a promise and
forget it; if it fails, nobody may notice. In Veles a task can only be
started inside a `scope`, and the scope waits for all of its tasks. If
one fails, the scope cancels the others and passes the error up. There
are no orphan tasks — this is called **structured concurrency**.

## What happens when — the whole life of a task

Here is exactly which machinery each line uses:

| You write | What happens underneath |
|---|---|
| ordinary code (`x + 1`, a loop, a call) | Runs on whatever thread is running your task, like in any language. |
| `async f(x)` | A task is created (a small record plus the coroutine's frame, holding `f`'s local variables) and put in a queue. It does **not** run yet; a free thread picks it up in a moment. |
| `await sleep(d)` | The task's coroutine saves its place and **parks**; the thread immediately takes another task. A timer puts the task back in the queue when `d` has passed. |
| `await channel.recv()` | Same: parks until another task sends a value. |
| `await t` (a task handle) | Parks until task `t` finishes, then gives you its result. |
| reading a socket (`net`, `http`) | The task parks; the executor asks the OS to tell it when data arrives. **No thread is used while waiting** — which is why one server can hold thousands of connections. |
| a long computation, no `await` | The task keeps its thread until it is done. The other threads carry on with other tasks. |
| calling C code that blocks, reading the terminal, waiting for a child process | The thread is really stuck inside the call. Veles notices within about a millisecond and gives the rest of that thread's work to a spare thread, so other tasks keep running. |
| end of a `scope` | The task that owns the scope parks until every child has finished. |

Where does the program *start*? `main` itself runs as the first task.
The program ends when `main` returns.

### Why "coroutine" and not "green thread"?

Go's goroutines are green threads with their **own stack** each: a
goroutine can pause anywhere, deep inside any function, because its
whole stack is kept. Veles uses **stackless coroutines** instead: only
the functions that may suspend are compiled into coroutines, and only
the variables they still need after an `await` are kept, in a small
object on the heap. Two consequences, both good:

- A task that is waiting costs a few hundred bytes, not a stack.
- A function that never suspends is a plain, fast function — nothing
  is added to it. The compiler knows which is which, because it
  inferred it.

The price is that pausing only happens at an `await`. Which leads to
the next point.

### When a task gives up its thread

A task gives its thread up when it reaches an `await` (or finishes) — and
a loop in a function that can suspend also gives it up by itself, at the
end of an iteration, once it has run for about 10 ms while other tasks
wait. A function that never suspends is a plain function with no frame
to come back to, so its loop keeps the thread until it is done. With one
thread per core this rarely matters — a long computation keeps one core
busy while the others run everything else. It matters with a single
thread (`VELES_THREADS=1`), or when many tasks compute for a long time at
once. Such a loop can offer its thread now and then:

```veles
// fragment
loop (i in 0..<10000000) {
  total += work(i)
  if (i % 100000 == 0) yieldNow()   // let the others have a turn
}
```

`yieldNow()` does not wait at all; it just goes to the back of the line.
And a cancellation stops a loop at the end of an iteration, whichever
kind of function it is in, so a long computation can always be stopped.

## How many threads?

By default, one per CPU core: on an 8-core laptop, eight tasks really
run at the same instant. You can change that with an environment
variable, which is handy when debugging:

```text
VELES_THREADS=1 veles run .     # everything on one thread, like JavaScript
VELES_THREADS=4 veles run .     # four threads
```

Behind the scenes there are also a couple of helpers: a small monitor
thread that watches for threads stuck in blocking calls, and spare
threads it hands work to. At any moment, at most `VELES_THREADS` threads
run your Veles code.

## Sharing: the one real danger

Several cooks in one kitchen can collide. If two tasks on two threads
change the same variable at the same instant, one change can be lost —
each reads `5`, each writes `6`, and one increment vanishes. This bug is
called a **data race**, and in most languages it compiles fine and
fails once in a thousand runs.

Veles refuses to compile it. The rule: what you hand to a task must be
**Sendable** — something no one can change behind anyone's back:
numbers, strings, immutable lists (`List`, not `MutableList`), structs
made of those. A function that would change a `var` from inside tasks
is refused (`forEachConcurrent` runs its function as tasks, several at
once):

```veles
// fragment
var count = 0
words.forEachConcurrent(w => { count += 1 })
// error: this lambda cannot cross a task boundary: it captures 'count', a 'var'
```

When tasks really need to share something that changes, there are
three tools, from simplest to most flexible:

1. **Return a result.** Let each task compute its own value and collect
   them with `await` — like the downloads above. No sharing at all.
2. **Send messages.** A `Channel` is a queue between tasks: one task
   `send`s, another `recv`s. Like `postMessage`, but typed.
3. **Lock it.** An `Atomic` holds a number (or any value) that tasks
   update one at a time; a `Mutex` holds anything and lets one task at
   a time work on it.

```veles
use io

fun countWords(text: string, total: Atomic<i64>, longest: Mutex<string>) {
  loop (word in text.split(" ")) {
    total.add(1)
    longest.withLock(best => {
      if (word.len() > best.len()) *best = word
    })
  }
}

fun main() {
  val total = Atomic(value: 0)
  val longest = Mutex(value: "")
  scope {
    async countWords("the quick brown fox", total, longest)
    async countWords("jumps over the lazy dog", total, longest)
    async countWords("concurrency without the fear", total, longest)
  }
  io.println("${total.load()} words, longest: ${longest.get()}")
}
```

Output:
```text
13 words, longest: concurrency
```

The three tasks run at the same time on different threads, and the
totals still come out right, every time. An `Atomic` of a number does
not even take a lock: the processor has instructions that update a
number in one indivisible step, and Veles uses them.

One more rule protects you from the classic lock bug: you cannot
`await` inside `withLock`. A task that paused while holding a lock
could make every other task wait for it forever; the compiler does not
let that be written.

## Things that surprise people

**"My output comes out in a different order each run."** Tasks run at
the same time, so which one prints first is up to the timing. Do not
rely on it: collect results and print them in one place, or sort them.
(The examples on this page only print in an order that timing makes
certain.)

**"Do I need to mark my function `async`?"** No. There is no such
marker on functions. `async` is only used to *start* a task.

**"Can I start a task and let it run in the background forever?"** Not
outside a `scope`. Put the scope where the background work should end
— often around the body of `main`. That way a program never exits while
a task is half-done, and a failing task is never silently ignored.

**"Is this like Go?"** Close. Both run many cheap tasks on a few threads
and move work between threads when one runs out. Go's goroutines have
their own stacks and can be interrupted anywhere; Veles tasks are
stackless coroutines that pause at `await`, and Veles checks at compile
time that tasks do not race on shared data.

**"How do I get a request id to every log line without passing it
everywhere?"** A task-local value: `val requestId = TaskLocal(fallback: "-")`,
bind it with `requestId.withValue(id, () => handle(req))`, read it with
`requestId.get()` anywhere inside — tasks started there see it too. See
"Values that follow a task" in [chapter 12](12-concurrency.md).

**"What if a task crashes?"** Its scope cancels its sibling tasks (each
stops at its next `await`), waits for them to stop, and then passes the
failure on to the code around the scope. See "Fail fast" in
[chapter 12](12-concurrency.md).

## Summary

- **Task** = a job you start with `async`. Cheap. Think in tasks.
- **Thread** = a worker from the OS. Expensive. Veles owns them: one per
  core by default (`VELES_THREADS`).
- **Coroutine** = how a task pauses: the compiler turns every function
  that may wait into one. You never write it.
- **`await`** = "I am waiting; give my thread to someone else." The only
  place a task lets go of its thread.
- **`scope`** = the task's home. Nothing started in it outlives it.
- **Sharing** = hand tasks unchangeable values; for changing state use
  a result, a `Channel`, an `Atomic` or a `Mutex`. The compiler refuses
  the rest.

Next: [chapter 12, Concurrency](12-concurrency.md) — the full reference,
with `gather`, `race`, channels, cancellation and timeouts — and
[Concurrency: what happens, exactly](reference/concurrency.md), every rule
and every gotcha in one place.
