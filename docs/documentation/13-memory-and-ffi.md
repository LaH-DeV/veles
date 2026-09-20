# 13. Memory, `with`, `unsafe` and C

## The collector

Veles is garbage collected (D1). You allocate by writing `&x`, building a
list or map, capturing a variable in a closure, or starting a task; you
never free. The bootstrap collector is a non-moving mark-sweep: the
compiler emits a layout descriptor for every type so the heap is scanned
precisely, and the native stack is scanned conservatively.

What this buys you is the absence of a whole category of decisions.
There are no lifetimes, no ownership annotations and no reference
counting to reason about; a pointer is valid for as long as anyone holds
it.

```veles
use io

struct Node { value: i64, next: (*Node)? }

fun build(n: i64): (*Node)? {
  var head: (*Node)? = null
  loop (i in 0..<n) { head = &Node(value: i, next: head) }
  head
}

fun sum(list: (*Node)?): i64 {
  var total = 0 as i64
  var cur = list
  loop (cur != null) {
    total += cur.value
    cur = cur.next
  }
  total
}

fun main() {
  var grand = 0 as i64
  loop (_ in 0..<50) {
    grand += sum(build(1000))    // each list becomes garbage after this line
  }
  io.println("$grand")
}
```

Output:
```text
24975000
```

Run it with `VELES_GC_TRACE=1` in the environment to see the collections
happen. `VELES_GC_THRESHOLD=<bytes>` lowers the trigger, which is how
the collector's own tests stress it.

## Values versus pointers

A struct is a value: assigning copies it, passing it copies it (D7).
`&x` gives a pointer, `*T`, to a heap copy of `x` — or to `x` itself when
`x` is a local whose address is taken, in which case the compiler quietly
moves that local to the heap so the pointer stays valid after the
function returns (D10).

```veles
use io

struct Box { var n: i64 }

fun makeCounter(): *Box {
  var b = Box(n: 0)     // lives on the heap because &b escapes
  &b
}

fun main() {
  val p = makeCounter()
  p.n += 5
  val q = p             // copies the pointer; q and p share the Box
  q.n += 1
  var v = *p            // copies the value out
  v.n += 100
  io.println("${p.n} ${q.n} ${v.n}")
}
```

Output:
```text
6 6 106
```

Rule of thumb: values for small data and anything you want to copy
freely; pointers for identity (two places must see the same object) and
for recursive structures.

## Cleanup with `with`

Files, sockets and locks are not memory; the collector will not close
them for you at a useful time. A type that implements `Closeable` — one
method, `fun close()` — can be bound in a `with` statement, and
`close()` runs on **every** way out of the block: normal completion,
`return`, `break`, `continue`, a thrown error, or cancellation of the
task (D43/D47):

```veles
use io

struct Res {
  name: string
  impl Closeable {
    fun close() { io.println("close ${self.name}") }
  }
}

fun open(name: string): Res {
  io.println("open $name")
  Res(name)
}

error Oops { }

fun early(flag: bool): i64 {
  with (a = open("a"), b = open("b")) {
    if (flag) return 1
    io.println("body ${a.name} ${b.name}")
  }
  2
}

fun failing(): i64 throws Oops {
  with (r = open("r")) {
    throw Oops()
  }
}

fun main() {
  io.println("result ${early(true)}")
  io.println("result ${early(false)}")
  when (failing()) {
    is Ok(v) => io.println("ok $v")
    is Err(e) => io.println("failed $e")
  }
}
```

Output:
```text
open a
open b
close b
close a
result 1
open a
open b
body a b
close b
close a
result 2
open r
close r
failed Oops
```

Resources close in reverse order of acquisition.

`with` is an expression: its value is the body's, and the resources are
closed before that value is used — the shape of every "open, read,
close" function:

```veles
use io

struct Handle {
  name: string
  impl Closeable {
    fun close() { io.println("close ${self.name}") }
  }
  fun contents(): string = "<${self.name}>"
}

fun read(name: string): string = with (h = Handle(name)) { h.contents() }

fun main() {
  val both = with (a = Handle(name: "a"), b = Handle(name: "b")) {
    a.contents() + b.contents()
  }
  io.println("${read("r")} $both")
}
```

Output:
```text
close b
close a
close r
<r> <a><b>
```

A panic unwinds through `with` as well. The task that panicked is lost
(D20 — there is no catching it inside the task), but everything it held
is released on the way out, innermost first, so a bug in one request
handler cannot leak a socket or a file; a `gather` boundary is where the
panic becomes a value ([chapter 12](12-concurrency.md)):

```veles
use io

struct Res {
  name: string
  impl Closeable {
    fun close() { io.println("close ${self.name}") }
  }
}

fun deep(i: i64): i64 {
  await sleep(0)          // a suspending call: a task of its own underneath
  [10, 20].atOrPanic(i)   // its panic continues in the caller
}

fun risky(i: i64): i64 = with (a = Res(name: "a"), b = Res(name: "b")) {
  deep(i)
}

fun main() {
  loop (i in [1, 5]) {
    when ((gather { async risky(i) }).0) {
      is Ok(v)  => io.println("value $v")
      is Err(p) => io.println("panicked: ${p.message()}")
    }
  }
}
```

Output:
```text
close b
close a
value 20
close b
close a
panicked: index 5 out of bounds for list of length 2
```

A `close()` that itself panics during that unwinding does not stop it:
the remaining resources still close and the first panic is the one
reported.

## `unsafe`

Some things the type system cannot vouch for: calling C, dereferencing a
raw pointer, pointer arithmetic. They are allowed only inside an
`unsafe { }` block, so a search for `unsafe` finds every place the
guarantees of chapter 12 might not hold (D44).

There are two pointer types, and the difference is the whole point:

| | `*T` | `*raw T` |
|---|---|---|
| comes from | `&x` | C, or unmanaged allocation |
| kept alive by the GC | yes | no |
| dereference | anywhere | `unsafe` only |
| arithmetic | no | `unsafe` only, C-style (D50) |

## Calling C

Declare the C signature in an `extern "C"` block and call it inside
`unsafe`. The compiler links against the C library by default, so libc
functions need no extra setup: (Square root, `abs` and the like are
built into the language — `x.sqrt()` — so they never go through C.)

```veles
use io

extern "C" {
  fun cbrt(x: f64): f64
  fun toupper(c: i32): i32
}

fun cubeRoot(x: f64): f64 = unsafe { cbrt(x) }

fun main() {
  io.println("${cubeRoot(27.0)} ${unsafe { toupper(97) }}")
}
```

Output:
```text
3.0 65
```

Numeric types map to their C counterparts, `bool` to a byte, `string` to
the runtime's string struct, `*raw T` to `T*`. Passing a `*T` to C is
allowed but the C side must not keep it beyond the call, since the
collector does not know about that reference. This is how the standard
`io` module is written — it is a dozen lines over three C functions.

Next: [Attributes and the test runner](14-attributes-and-testing.md).
