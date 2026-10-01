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
method, `fun close()` — can be bound in a `with` statement, and
`close()` runs on **every** way out of the block: normal completion,
`return`, `break`, `continue`, a thrown error, or cancellation of the
task (D43/D47):

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
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
  fun contents(): string = "<${this.name}>"
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
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun deep(i: i64): i64 {
  await sleep(Duration.zero)          // a suspending call: a task of its own underneath
  [10, 20].at(i) ?: panic("deep: index $i is out of range")  // the panic continues in the caller
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
panicked: deep: index 5 is out of range
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

Arithmetic on a `*raw T` is C's: `p + n` and `p - n` step by whole
elements (`*raw u8` steps by bytes), `p - q` counts the elements between
two pointers of one type, and `<` `<=` `>` `>=` order addresses. Nothing
is checked — that is what `unsafe` means here:

```veles
use ffi, io

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
fun scaled(x: f64, exp: i32): f64 = unsafe {
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
use ffi, io

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
  with (s = try ffi.CString.of("hello")) {
    // SAFETY: `s` is open here; readString copies up to its NUL
    io.println("$xs ${unsafe { ffi.readString(s.ptr()) }}")
  }
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
