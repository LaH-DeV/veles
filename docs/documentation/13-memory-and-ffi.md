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
  var total = 0
  var cur = list
  loop (cur != null) {
    total += cur.value
    cur = cur.next
  }
  total
}

fun main() {
  var grand = 0
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
method, `fun close()` — can be bound with `with`. The statement
`with x = e` opens `x` for the rest of the block it is written in:
`close()` runs when that block ends, and on **every** way out of it
before then — `return`, `break`, `continue`, a failed `try`, a thrown
error, a panic, or cancellation of the task (D43/D47/D100). Several
close in reverse order of opening:

```veles
use io

struct Res {
  name: string
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun open(name: string): Res {
  io.println("open $name")
  Res(name)
}

error Oops { }

fun early(flag: bool): i64 {
  with a = open("a")
  with b = open("b")
  if (flag) return 1
  io.println("body ${a.name} ${b.name}")
  2
}

fun failing(): i64 throws Oops {
  with r = open("r")
  throw Oops()
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

When the block ends in a value — `2` above — the value is computed
first and the resources close before it is used. In a loop body the
block ends with every iteration, so a resource opened there is closed
before the next one opens.

`with x = e` needs statements after it in a braced block: as the last
statement it would close at once (a warning), and it is refused where
there is no "rest of the block" — at module level, as an expression
body, inside an expression, or as a body without braces.

To close a resource *before* its block ends, give it a block of its
own: `with (x = e) { ... }` closes `x` at that `}`. The block form also
opens several at once, `with (a = …, b = …) { }`, and is an expression:
its value is the body's, and the resources are closed before that value
is used — the shape of every "open, read, close" function:

```veles
use io

struct Handle {
  name: string
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
  fun contents(): string => "<${this.name}>"
}

fun read(name: string): string => with (h = Handle(name)) { h.contents() }

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

A value the block holds but never uses needs no name (D109): `with
sem.acquire()` holds a permit to the end of the block, and in the block
form the items mix, `with (f = try fs.open(p), sem.acquire()) { … }`.
`with conn`, naming a value you already have, hands its closing to the
`with`; from there on `conn` is a resource like any other. `with _ = e`
says the same as `with e`.

A resource is closed when its block ends, so it must not outlive the
block (D100): returning it, making it the block's value, storing it in a
variable or field declared outside the block, putting it in a collection
or sending it on a channel are errors — and so is returning or storing a
lambda that captures it. Return what you read from it instead. Lending
it is fine: passing the resource, or a lambda that uses it, as an
argument compiles, because the call ends before the block does. The
compiler does not follow the value into the function you pass it to, so
a function that *keeps* what it was lent is not caught — do not keep a
resource you were passed. Nor is a `with` value closed by hand: `with`
closes it already, so `r.close()` on it is an error (D136) — use the block
form to close it earlier.

```veles
// fragment
fun leak(path: string): fs.File throws IoError {
  with f = try fs.open(path)
  return f                                      // error: 'f' cannot be returned: it is closed when its 'with' block ends
}

fun firstLine(path: string): string throws IoError {
  with f = try fs.open(path)
  val text = try withTimeout(Duration.seconds(1), () => try f.read(4096))   // lent: fine
  text.decodeUtf8() ?: ""
}
```

The other way round, a resource held with `val` that is never closed is a
warning (D115): `'f' is never closed`, with a quick fix that turns the
`val` into `with`. Closing it by hand, returning it, storing it, passing
it to a function (`serve(listener)` hands it on) or capturing it in a
lambda all count as dealing with it, so the warning never fires on a
hand-off — and it cannot see a function that drops what it was handed.
A call whose `Closeable` result is thrown away warns too; write
`val _ = open(p)` when that is what you mean.

```veles
// fragment
fun size(path: string): i64 throws IoError {
  val f = try fs.open(path)        // warning: 'f' is never closed — fix: with f = try fs.open(path)
  try f.size()
}
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
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun deep(i: i64): i64 {
  await sleep(Duration.zero)          // a suspension point: the panic below unwinds through it
  [10, 20].at(i) ?: panic("deep: index $i is out of range")  // the panic continues in the caller
}

fun risky(i: i64): i64 => with (a = Res(name: "a"), b = Res(name: "b")) {
  deep(i)
}

fun main() {
  loop (i in [1, 5]) {
    when (gather { async risky(i) }) {
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
panicked: deep: index 5 is out of range
```

A `close()` that itself panics during that unwinding does not stop it:
the remaining resources still close and the first panic is the one
reported.

### A close that waits

Some resources end with a conversation: a transaction sends `ROLLBACK`
and waits for the answer, a telemetry pipeline sends what it still holds.
Their `close()` says so with `suspends` (D147):

```veles
use io

struct Session {
  name: string
  implement Closeable {
    fun close() suspends {
      await sleep(Duration.millis(5))     // say goodbye to the peer
      io.println("closed ${this.name}")
    }
  }
}

struct Log {
  implement Closeable {
    fun close() {
      io.println("closed the log")
    }
  }
}

fun work(): i64 {
  with log = Log()
  with s = Session(name: "session")
  io.println("working")
  42
}

fun main() {
  io.println("got ${work()}")
}
```

Output:
```text
working
closed session
closed the log
got 42
```

Suspension follows the type: a `with` on such a value waits at its end,
so the function holding it suspends, as a call of a suspending function
would make it; a `with` on any other resource is unchanged. Every way out
of the block waits for it — the end, `return`, `throw`, a cancellation,
and a panic, whose close still runs before the resources outside it, so
everything closes innermost first. The close is *shielded*: a
cancellation that arrives while it waits lets it finish and is seen
afterwards. That cuts both ways: a close that hangs keeps its block from
ending, so a close that talks to a peer bounds its own wait (the
transaction's uses its timeout).

Generic code follows the type argument: `fun closeAll<T: Closeable>(x: T)`
waits in the instance for a `Session` and not in the one for a `Log`.
Three places refuse such a value, because nothing there may wait: a lock
region (D107), a lambda that cannot suspend, and a `Closeable` trait
object — a method table holds one plain `close` — so keep the concrete
type or a type parameter. A `close()` that waits without saying
`suspends` is an error that asks for it.

## Atomics and memory orders

An `Atomic` of a number, a `bool`, an enum or a pointer — nullable or not
— is one machine word changed by one processor instruction, with no lock
([chapter 12](12-concurrency.md) lists its operations). That is enough to
build lock-free structures. A stack whose `push` and `pop` are each one
compare-and-set, retried when another task got in between, needs only an
`Atomic` of its top node; the collector keeps a popped node alive while any
task still holds it, so a node's address is never reused under a task that
is about to compare against it:

```veles
use io

struct Node {
  value: i64
  next:  (*Node)?
}

struct Stack {
  head: Atomic<(*Node)?> = Atomic(value: null)

  fun push(value: i64) {
    var seen = this.head.load(order: MemoryOrder.Relaxed)
    loop {
      val node = &Node(value, next: seen)
      // publishes the node: what was written to it is visible to the
      // task that acquires it in pop
      val found = this.head.compareExchange(seen, node, order: MemoryOrder.Release, failure: MemoryOrder.Relaxed)
      if (found == seen) return
      seen = found                     // another push got in: try again on top of it
    }
  }

  fun pop(): i64? {
    loop {
      val top = this.head.load(order: MemoryOrder.Acquire) ?: return null
      if (this.head.compareAndSet(top, top.next, order: MemoryOrder.Acquire)) return top.value
    }
  }
}

fun fill(s: Stack, from: i64) {
  loop (i in from..<from + 1000) s.push(i)
}

fun main() {
  val s = Stack()
  scope {
    loop (k in 0..<4) async fill(s, k * 1000)
  }
  var sum: i64 = 0
  var count: i64 = 0
  loop {
    val v = s.pop() ?: break
    sum += v
    count += 1
  }
  io.println("$count values, sum $sum")
}
```

Output:
```text
4000 values, sum 7998000
```

Every operation takes an optional `order:`, a `MemoryOrder`. Without one
it is `SeqCst`: all tasks see all atomic operations in a single order,
which is the order a program reads them in. The weaker orders let the
processor and the compiler move other memory accesses past the atomic
one, which is cheaper on some machines (ARM most of all) and correct only
when paired as below:

| Order | Means | Takes it |
|---|---|---|
| `Relaxed` | the operation is atomic and nothing more: a statistics counter read at the end | every operation |
| `Release` | what this task wrote before the store is visible to a task that loads the value with `Acquire` | a store; a read-modify-write |
| `Acquire` | the other half: after this load, what the releasing task wrote is visible | a load; a read-modify-write |
| `AcqRel` | both, on an operation that reads and writes | a read-modify-write |
| `SeqCst` | `AcqRel`, plus the single order every task agrees on | every operation |

An order an operation cannot take is a compile error: a load does not
release, a store does not acquire.

```veles
// fragment
val ready = Atomic(value: false)
val n = ready.load(order: MemoryOrder.Release)
// error: MemoryOrder.Release is not an order for a load; use Relaxed, Acquire or SeqCst (D144)
```

`compareAndSet` and `compareExchange` take a second order, `failure:`, for
the load a failed attempt makes: `Relaxed`, `Acquire` or `SeqCst`, no
stronger than `order:`. Left out, it is the strongest load `order:` allows
(`Acquire` for `AcqRel`, `Relaxed` for `Release`). An order chosen while
the program runs (a variable rather than a `MemoryOrder.` case) is not
checked; one the operation cannot take then acts as `SeqCst`. An `Atomic`
guarded by a lock word accepts `order:` and is always `SeqCst`.

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

Arithmetic on a `*raw T` is C's: `p + n` and `p - n` step by whole
elements (`*raw u8` steps by bytes), `p - q` counts the elements between
two pointers of one type, and `<` `<=` `>` `>=` order addresses. Nothing
is checked — that is what `unsafe` means here:

```veles
use ffi
use io

fun main() {
  val n = 5
  // SAFETY: every step stays inside the n i64 of `block`, freed once after its last use
  unsafe {
    val block = ffi.alloc(n * 8).cast<*raw i64>()
    val end = block + n
    var p = block
    loop (p < end) {
      *p = (p - block) * 2
      p += 1
    }
    io.println("${*(end - 1)} ${end - block}")
    ffi.free(block.cast<*raw u8>())
  }
}
```

Output:
```text
8 5
```

An `unsafe` block is a promise the compiler cannot check, so the reason
it holds is written down next to it: a `// SAFETY:` comment on the line
above the block (or as the first line inside it) saying why it is sound.
A block without one is a warning whose quick fix inserts the comment for
you to finish — an empty reason still warns. The alternative is to
declare the function `unsafe fun`: the obligation then moves to its
callers, who need an `unsafe` block (and a reason) of their own.

```veles
// fragment
fun length(s: ffi.CString): u64 {
  // SAFETY: an open CString is a live, NUL-terminated copy; strlen only reads it
  unsafe { strlen(s.ptr()) }
}
```

Every `unsafe` block in the standard library carries one; that is how its
uses of C were audited.

### Indexing without the bounds check

Every index is checked, in every build profile; there is no switch that
turns the checks off for a whole program, and the compiler already drops
the checks it can prove (chapter 4). For the hot loop where it cannot —
a lexer walking its input byte by byte — three accesses skip the check
in a release build (D114):

| Access | On | Checked spelling |
|---|---|---|
| `xs.atUnchecked(i): T` | `List`, `MutableList` | `xs.at(i)` |
| `xs.setUnchecked(i, v)` | `MutableList` | `xs.set(i, v)` |
| `s.byteAtUnchecked(i): u8` | `string` | `s.byteAt(i)` |

They are allowed only inside `unsafe`, and `i` is the plain offset
(`0 <= i < len`, no counting from the end). A debug build still checks
them and panics "unchecked index … out of bounds" at the call, so tests
catch a wrong index; in a release build an index out of range is
undefined behaviour.

```veles
use io

fun digits(s: string): i64 {
  var n = 0
  var i = 0
  loop (i < s.len()) {
    // SAFETY: i < s.len() by the loop condition
    val b = unsafe { s.byteAtUnchecked(i) }
    if (b >= '0' && b <= '9') n += 1
    i += 1
  }
  n
}

fun main() {
  io.println("${digits("veles 0.68, 2026")}")
}
```

Output:
```text
7
```

## Calling C

Declare the C signature in an `extern "C"` block and call it inside
`unsafe`. The compiler links against the C library by default, so libc
functions need no extra setup: (Square root, `abs` and the like are
built into the language — `x.sqrt()` — so they never go through C.)

```veles
use io

extern "C" {
  fun ldexp(x: f64, exp: i32): f64
  fun toupper(c: i32): i32
}

// x times 2 to the power `exp`, exact
fun scaled(x: f64, exp: i32): f64 => unsafe {
  // SAFETY: ldexp takes two numbers and returns one
  ldexp(x, exp)
}

fun main() {
  // SAFETY: toupper takes a character code and returns one
  io.println("${scaled(3.0, 4)} ${unsafe { toupper(97) }}")
}
```

Output:
```text
48.0 65
```

Numeric types map to their C counterparts (mind that C's `long` is 32
bits on Windows and 64 on Linux and macOS), `bool` to a byte, `string` to
the runtime's string struct, `*raw T` to `T*`, an `extern struct` to the
C struct with the same fields. A GC-managed `*T` cannot be passed: the
collector would not know C holds it (D44/D50). This is how the standard
`io` module is written — it is a dozen lines over three C functions.

An `extern struct` can be passed and returned by value, in both
directions — to a C function, through an `extern fun` pointer, and from C
into an `extern "C" fun`. The compiler follows each platform's C calling
convention for it (Windows x64, System V on Linux and Intel macOS, AAPCS64
on ARM), so a struct travels in the registers or memory a C compiler would
use:

```veles
use io

extern struct LDiv {
  quot: i64
  rem: i64
}

extern "C" {
  fun lldiv(n: i64, d: i64): LDiv   // long long on every platform
}

fun main() {
  // SAFETY: lldiv takes two numbers and returns a struct of two
  val r = unsafe { lldiv(70000000001, 7) }
  io.println("${r.quot} remainder ${r.rem}")
}
```

Output:
```text
10000000000 remainder 1
```

C headers lay some data out by hand, and four attributes say so (D120):

```veles
use io

extern union EpollData {        // every field at offset 0; size of the largest
  ptr: *raw ()
  var fd: i32
  var u64: u64
}

@packed                         // no padding: data starts at byte 4
extern struct EpollEvent {
  var events: u32
  var data: EpollData
}

extern struct Header {
  kind: u8
  @align(16)
  payload: i32  // a field at the next multiple of 16
}

@transparent struct Fd { handle: i32 }       // crosses into C as the i32 itself

@align(64) struct Counter { var hits: i64 }  // a cache line of its own

fun main() {
  var ev = EpollEvent(events: 1, data: EpollData(fd: 3))   // a union starts as one field
  ev.events = 5
  // SAFETY: data was built from fd, so fd is the live field
  val fd = unsafe { ev.data.fd }
  io.println("${ev.events} $fd ${Header(kind: 1, payload: 2).payload} ${Fd(handle: 4).handle}")
}
```

Output:
```text
5 3 2 4
```

- **`extern union`** holds one of its fields at a time, all at offset 0.
  It is built from exactly one field. Reading or writing a field happens
  inside `unsafe`, since only C knows which field is live. It has no `==`,
  no text and nothing derived, and it is not a constant.
- **`@packed`** on an extern struct or union removes the padding. The
  compiler reads and writes a misaligned field through the whole struct,
  so it is never reached through a pointer of its own type: `&ev.events` is
  refused, and so is passing a packed struct to C by value — pass
  `*raw EpollEvent`.
- **`@align(n)`** raises an alignment: on a field of an extern struct
  (C's `_Alignas`), or on any struct. `n` is a power of two from 1 to 4096,
  never below the natural alignment. An over-aligned struct is aligned
  wherever it lives — locals, globals, list storage, the heap — which is
  what keeps two hot counters on different cache lines.
- **`@transparent`** on a struct with one field gives a binding its own
  handle types at no cost: C sees the field, in registers and all.

### Arrays in a C struct

An `Array<T, N>` field of an extern struct is C's `T x[N]` (D121): the same
size, the same offsets, the same stride in an array of the struct. The
struct crosses by value as C passes it — an eight-byte `Array<u8, 8>` in a
register, a larger one in memory — and `Array<Array<u8, 3>, 2>` is C's
two-dimensional `uint8_t rows[2][3]`:

```veles
use io

extern struct SockAddr {
  family: u16
  port: u16
  addr: u32
  zero: Array<u8, 8>
}

extern "C" {
  fun memset(p: *raw u8, c: i32, n: i64): *raw u8
}

fun main() {
  var buf: Array<u8, 6> = [1, 2, 3, 4, 5, 6]
  // SAFETY: six bytes of storage, written before C returns
  buf.withRaw(p => unsafe { memset(p, 7, 3) })
  io.println("$buf ${SockAddr(family: 2, port: 80, addr: 1, zero: Array.make(0)).zero.len()}")
}
```

Output:
```text
[7, 7, 7, 4, 5, 6] 8
```

`buf.withRaw(f)` lends C a pointer to the array's own storage for the length
of the closure, as `List.withRaw` does for a list's: C may read it, and write
it when the array is a `var`. A struct holding an array may be larger than
C's registers — there is no limit, and the compiler follows the platform's
rule for each size.

### Large values live in memory

A value of 128 bytes or more that holds an array is never an SSA value: it
is copied with `memmove`, handed to a function as the address of a copy, and
returned through a pointer the caller gives (the memory class, D121). It
applies in the same way when the value is an argument to a C function, which
is how a 160-byte extern struct reaches C as C expects it, and it is why a
megabyte array on the stack costs a `memcpy` per copy and no more.

A C function may block — sleep, wait on a lock, read a file. That is
fine: while the call runs, a collection does not wait for it, and if it
lasts longer than about a millisecond while other tasks are waiting to
run, its thread's work moves to a spare thread (D66). Other tasks never
stall behind a slow C call.

### Linking a C library

A library other than the C runtime is named in the package's
`veles.toml`, never in source (D67):

```toml
[native]
libs = ["pq"]                 # -lpq: the shared library (or its import library)
static-libs = ["z"]           # the archive itself: nothing needed at run time
lib-paths = ["vendor/lib"]    # searched first; relative to veles.toml
pkg-config = ["libpq"]        # flags from `pkg-config --libs`
```

```veles
// fragment
extern "C" {
  fun compressBound(sourceLen: u64): u64
}

fun main() {
  // SAFETY: compressBound takes a size and returns one
  io.println("${unsafe { compressBound(1000) }}")   // 1013, with static-libs = ["z"]
}
```

An entry that names a file — `"vendor/libfoo.a"`, `"shim.o"` — is linked
as that file. A dependency's `[native]` table links into every program
that uses the dependency, so a package that binds a C library carries
its link instructions with it.

### Strings, buffers and callbacks

The collector's memory never becomes a pointer C may keep (D69). Memory
crosses one of three ways:

- **Copied for C to keep** — `std/ffi`: `ffi.CString.of(s)` is a
  NUL-terminated copy in C's own memory, freed by `with` (a string holding
  a NUL byte is refused with `ffi.NulByte`, since C would read it as
  ending there); `ffi.alloc(n)`/`ffi.free(p)` for raw buffers. Coming back,
  `ffi.readString(p)` and `ffi.readBytes(p, n)` copy out of C memory; they
  trust the pointer, so they are `unsafe fun`s.
- **Lent for the length of a call** — `xs.withRaw(p => ...)` hands C a
  pointer to the list's own elements, no copy. The elements must be
  `CLayout`: numbers, `bool`, raw pointers and `extern struct`s — what C
  reads as it lies in memory. The pointer is good only inside the lambda.
- **A value C hands back** — `ffi.handle(value)` gives C an opaque
  `void *` for a callback's `userdata`; the callback gets the value back
  with `ffi.Handle<T>.from(p)`. The value stays alive while the handle is
  open.

A Veles function C can call is declared `extern "C" fun` with a body.
`&name` is its address, of type `extern fun(...)` — a C function pointer:

```veles
use ffi
use io

extern "C" {
  fun qsort(base: *raw u8, count: u64, size: u64, compare: extern fun(*raw u8, *raw u8): i32)
}

extern "C" fun ascending(a: *raw u8, b: *raw u8): i32 {
  // SAFETY: qsort calls this only with pointers into the list of i64 it sorts
  val x = unsafe {
    *(a.cast<*raw i64>())
  }
  // SAFETY: as for `a`
  val y = unsafe {
    *(b.cast<*raw i64>())
  }
  if (x < y) -1 else if (x > y) 1 else 0
}

fun main() throws ffi.NulByte {
  val xs: MutableList<i64> = [42, 7, 19, 3]
  // SAFETY: qsort sorts xs.len() elements of 8 bytes inside the lent storage
  xs.withRaw(p => unsafe {
    qsort(p.cast<*raw u8>(), xs.len().wrapU64(), 8, &ascending)
  })
  with s = try ffi.CString.of("hello")
  // SAFETY: `s` is open here; readString copies up to its NUL
  io.println("$xs ${unsafe { ffi.readString(s.ptr()) }}")
}
```

Output:
```text
[3, 7, 19, 42] hello
```

Its parameters and result must be `CLayout` (or an `extern fun`); it may
not throw — C cannot receive the error, so return a status code — and
may not suspend. A panic inside it ends the process with its location:
it cannot unwind through C's frames. `p.cast<*raw T>()` reinterprets a raw
pointer (C's `void *`), and `(&x).cast<*raw u8>()` gives C the address of a
Veles value for the length of a call; both need `unsafe`, as does calling
through an `extern fun` value. `examples/ffi` puts all of it together.

### Variadic C functions

`...` ends the parameter list of a C function taking variadic arguments —
`printf`, `snprintf`, and POSIX's `open` and `fcntl` (D123):

```veles
use ffi
use io

extern "C" {
  fun snprintf(buf: *raw u8, size: u64, format: *raw u8, ...): i32
}

fun main() throws ffi.NulByte {
  with format = try ffi.CString.of("%d items, %.1f%% full, %s")
  with name = try ffi.CString.of("queue")
  val fill: f32 = 37.5
  val buf: MutableList<u8> = MutableList.repeat(0, 64)
  // SAFETY: the format names exactly the three arguments; snprintf writes at most 64 bytes
  buf.withRaw(p => unsafe { snprintf(p, 64, format.ptr(), 12, fill, name.ptr()) })
  // SAFETY: snprintf ended the text with a NUL inside the buffer
  io.println(buf.withRaw(p => unsafe { ffi.readString(p) }))
}
```

Output:
```text
12 items, 37.5% full, queue
```

The extra arguments get C's default promotions: `bool`, `i8`, `i16`, `u8`
and `u16` widen to `i32`, `f32` to `f64`, and a literal is typed as C types
it — `12` is an `i32` (C's `int`), `2.5` an `f64`. What may be passed is a
32- or 64-bit integer, an `f64`, a raw pointer or a C function pointer; a
string, a struct or any Veles value is refused (convert it, or pass a
pointer). Only a function in an `extern "C"` block can say `...`: a Veles
function takes `name: T...` instead, and C cannot call a variadic Veles
function. The format string is C's business — a mismatch between it and the
arguments is undefined behaviour the compiler cannot see, which is why the
call is `unsafe`.

### When C goes wrong: `--sanitize`

A mistake on the C side of the boundary — a buffer one byte short, a
pointer used after `free` — corrupts memory silently, and the crash, if
any, comes later somewhere else. `veles build --sanitize` (also on `run`
and `test`) builds the runtime under AddressSanitizer and
UndefinedBehaviorSanitizer and links their runtimes: the first bad access
stops the program with a report naming the access and the call chain,
Veles functions included:

```text
==685==ERROR: AddressSanitizer: heap-buffer-overflow on address 0x502000000014
WRITE of size 5 at 0x502000000014 thread T0
    #0 in memset
    #1 in v_main.main
```

Libc functions (`memset`, `strcpy`, ...) are checked as they are; compile
your own C code with `-fsanitize=address,undefined` to have it checked
too. The Veles code itself needs no instrumentation — it is bounds-checked
already — and the program runs a few times slower, so this is a debugging
build. On Windows it needs clang's sanitizer runtimes installed (MSYS2:
`pacman -S mingw-w64-ucrt-x86_64-compiler-rt`).

Next: [Attributes and the test runner](14-attributes-and-testing.md).
