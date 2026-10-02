# Veles cheat sheet

Everything on one page. `D<n>` refers to a decision in `veles-spec.md`.

## Program structure

```veles
// fragment
use fs, io, os                  // import modules; call as io.println (fmt sorts: std first, then the rest)
use geometry as geo             // rename a module: geo.Point
use io { println, eprintln as warn }  // bare names on top of io.: println("x"); rename with `as`

const limit = 10                // module-level constant
val banner = "hi"               // module-level value
var counter = 0                 // module-level variable

fun main() { }                  // entry point
```

A module is a directory; every `.vs` in it shares a namespace (M2).
`public` exports from the module (M5); `public use m` in the root module
shows a module to other packages (D89). `veles.toml` names the package and
its `[dependencies]` (M1).

## Bindings and types

```veles
// fragment
val x = 1                       // immutable, inferred i64
var y: i64 = 2                  // mutable, explicit type
val (a, b) = (1, "one")         // tuple destructuring, nests: val ((x, y), z) = ...; also in loop heads and lambda params ((x, y), z) => ...; (a, b) = (b, a) swaps
```

| Kind | Types |
|---|---|
| integers | `i8 i16 i64 i64 u8 u16 u32 u64` |
| floats | `f32 f64` |
| other scalars | `bool string` |
| nullable | `T?` (`(*T)?` for a nullable pointer) |
| pointer | `*T` (GC-managed), `*raw T` (unsafe, D50) |
| tuple | `(A, B)`, unit is `()` |
| function | `fun(A, B): R`, with `suspends` / `throws E`; `sendable fun(...)` may cross tasks |
| collections | `List<T> MutableList<T> Map<K,V> MutableMap<K,V> Set<T> MutableSet<T>` |
| ranges | `Range<T>` from `a..b` (inclusive) or `a..<b` |
| concurrency | `Channel<T>`, `Task<T>`, `Mutex<T>`, `Atomic<T>` |
| results | `Result<T, E>`; `T?` is `Option<T>` |
| alias | `type Key = (i64, u64)`, `type StrMap<V> = Map<string, V>`, `public type Point = geo.Point` — a name, never a new type (D55) |

## Operators

| | |
|---|---|
| arithmetic | `+ - * / %` (checked, D21); wrapping `+% -% *%` |
| bitwise | `& \| ^ << >> ~` on integers; `&` and shifts bind like `*`, `\|` `^` like `+` |
| comparison | `== != < <= > >=` |
| logic | `&& \|\| !` |
| conversion | `x.toI64()` (`T?` when it can lose), `x.wrapU8()`, `p.cast<*raw T>()` |
| operators on your types | `implement Addable { fun plus(other: T): Out }` → `a + b`, `+=`; also Subtractable, Multipliable, Divisible, Negatable (D71) |
| nullable | `x ?: fallback`, `x?.member`, `x?.method()`; `x?.a.b()` skips the whole rest of the chain on null (D70) |
| result | `r ?? fallback` (`?:` for a Result; the fallback may leave), `r catch (e) { ... }` sees the error, `val v = r else return` binds or leaves |
| several `try`s, one handler | `do { val a = try f(); try g(a) } catch (e) { fallback }`, and `try f().g() catch (e) { fallback }` for one call and a chain — a failed `try` or `throw` goes to the handler; `e` is the union of the errors; `return`/`break`/`continue` leave the function or loop (D98) |
| a chain | `try client.fetch(url).json<User>()` — one `try` covers every failing call in the chain; the error type is their union; arguments are not covered (D134) |
| or fail | `x ?! error` — a `T?` or `Result` becomes a `Result` failing with `error`; `try x ?! e` propagates it |
| type test | `x is T`, `x !is T` |
| address | `&x` → `*T`, `*p` reads through |
| range | `a..b`, `a..<b` |
| assignment | `= += -= *= /= %=` |
| strings | `"a" + "b"`, `s.len()` (bytes), `s.trim()`, `s.split(",")`, `s.splitOnce("=")` (a `(string, string)?`), `s.replace(a, b)`, `s.indexOf(p)`, `s.toUpper()`, `s.padStart(n)`, `s.toInt()`, `s.chars()`, `s.byteAt(i)` (a `u8`; `'"'` is a byte literal), `"$name ${expr}"`, escapes `\n \t \\ \" \$ \u{..}`; one code point at a time is `use utf8` — `utf8.decode(s, i)`, `utf8.encodeTo(out, code)` |

## Control flow

```veles
// fragment
if (c) a else b                       // expression
loop { ... break }                    // forever
loop (cond) { ... }                   // while
loop (x in xs) { ... }                // any Iterable, Range, Map ((k, v) in m); (a..b).step(n), (a..b).reversed(), (a..b).reversed().step(n); the body may not push/remove on xs itself (D102)
loop (&x in xs) x.bump()             // by reference: x is *T (MutableList only); loop ((k, &v) in m) for map values
loop :outer (x in xs) { continue outer; break outer }
when (v) { 1 => "one"; 2, 3 => "few"; else => "many" }
when { x < 0 => "neg"; else => "pos" }
when (shape) { is Circle(r) => ...; is Rect(w, h) if w > h => ...; is Point => ... }
when (opt) { Some(x) => ...; null => ... }
when (val r = res) { is Ok => r; is Err => r.message() }   // r is the payload in each arm; Ok(v) / Err(e) also work
return v; break; continue; throw e; panic("msg")   // panic and os.exit are Never: nothing after them runs; fun f(): Never is allowed
```

## Functions and lambdas

```veles
// fragment
fun add(a: i64, b: i64): i64 = a + b                // expression body
fun greet(name: string, punct: string = "!") { }    // default; call greet("x", punct: "?")
fun sum(xs: i64...): i64 = xs.fold(0, (a, b) => a + b)  // variadic: sum(1, 2), sum(list...)
fun show<T: Show>(x: T): string = x.show()         // generic with bound
fun fetch(url: string): string suspends throws E    // effects (inferred for free functions)
val f = x => x * 2                                  // lambda; (a, b) => ..., (x: i64) => ..., _ => ... ignores its argument
val g = () => { var n = 0; n }                      // block body
val h = (a: i64) => (b: i64) => a + b               // curried
```

Lambdas capture by reference (D37). Last expression of a block is its
value.

## Structs, traits, sealed types, enums

```veles
// fragment
struct Point {
  var x: i64                    // var: assignable (through this, a binding, a pointer)
  y: i64 = 0                    // bare (or `val`): set by the constructor, never assigned (D22); default
  sum: i64                      // no default + assigned in init = init's, not the caller's (a default is a constant: no `this` in it)
  init { this.sum = this.x + this.y }  // runs after every construction; must assign sum on every path; not callable (D28)
  public protected var hits: i64 = 0  // protected var: read wherever visible, assigned only by Point's own code
  fun len(): i64 = this.x + this.y
  fun move(dx: i64) { this.x += dx }   // no marker: a method may assign the var fields
  static fun origin(): Point = Point(x: 0)  // no this; Point.origin()
  static val unit = Point(x: 1)             // a constant in the type's namespace: Point.unit (public to export; never var)
  private count: i64 = 0                    // private: only Point's own methods/implement/extend blocks; no marker (or `internal`) = the module; public = the package
}
struct Shared<T> {
  private cell: *T
  init(value: T) { this.cell = &value }     // init parameters join the constructor (D73): Shared(value: 1)
}
val p = Point(x: 1)             // named construction (Point(x, y) puns variables named like fields); p == q, "$p" work
val n = i64.parse("42")         // i64?; Parsable — T.parse(s) in generic code

trait Shape {
  fun area(): f64
  fun describe(): string = "area ${this.area()}"   // default
}
struct Sq { s: f64; implement Shape { fun area(): f64 = this.s * this.s; override fun describe(): string = "sq" } }   // your own type: implement in the body
implement Shape for i64 { fun area(): f64 = 0.0 }   // a foreign type: top-level implement (for your own type it is a lint with a quick fix)
val s: Shape = Sq(s: 2.0)       // trait object (D9)

extend Point {                  // more inherent methods, outside the body (D23)
  fun norm(): i64 = this.x.abs() + this.y.abs()
}
extend<T: Shape> Box<T> { ... } // only for types you declare; bounds allowed

sealed trait Expr {             // fixed set of variants (D12)
  fun eval(): i64 = when (this) { is Num(v) => v; is Neg(e) => -e.eval() }
}
struct Num : Expr { v: i64 }
struct Neg : Expr { e: *Expr }

enum Phase : u8 { Red = 1, Amber, Green = 10 }   // closed set of VALUES of one integer type (D57); i64 when unsaid; +1 from the previous
val ph = Phase.Amber            // ph.value == 2, "$ph" == "Amber", ph.toString(); ph == 2, ph < Phase.Green (compares with the base type, never converts)
Phase.values(); Phase.fromValue(10); Phase.parse("Red")   // List<Phase>; Phase?; Phase?  — no methods, fields or impls of its own; Comparable + Hashable for free
when (ph) { Phase.Red => ...; Phase.Amber => ...; Phase.Green => ... }   // exhaustive by member (D13)

trait Iterator { type Item; fun next(): Item? }   // associated type
implement Iterator for Countdown { type Item = i64; fun next(): i64? { ... } }

implement Comparable for Point { fun compareTo(other: Point): Ordering = this.x.compareTo(other.x) }  // <, sorted, min; Ordering = enum { Less = -1, Equal, Greater }
implement Display for Point { fun toString(): string = "(${this.x})" }              // "$p"
// also Equatable (==) and Hashable (map keys); structural by default
```

## Errors (D4)

```veles
// fragment
error NotFound { key: string }                          // a struct that is an Error; only errors can be thrown
error Invalid { why: string; fun message(): string = this.why }
error Failed { message: string }                        // a `message` field is the message
error GetErrors = NotFound | Invalid | Failed             // a named error set (D45)
error Wrapped { cause: GetErrors }                      // an error's field may hold a set
fun get(k: string): string throws NotFound = if (k == "a") "A" else throw NotFound(key: k)
fun getAll(): string throws = try get("a") + try get("b")   // error type inferred: NotFound
when (val r = get("z")) { is Ok => r; is Err => r.key }    // caller sees Result; r is the payload per arm
val r: Result<i64, NotFound> = Ok(1); r.getOrNull(); r.getOrDefault(0); r.errorOrNull(); results.oks(); results.errors()
val user = try users.get(id) ?! NotFound(key: id)        // `x ?! e`: absence (T?) or failure (Result) becomes failure with e; try propagates
val n = try parse(s) ?! Invalid(why: "not a number")     // the old error is dropped; keep it with mapError:
val m = try parse(s).mapError(e => Invalid(why: e.message()))
val n = parse(s) ?? 0; parse(s) catch (e) { fallback(e) }; parse(s) ?? return   // ?? is ?: for a Result: the value, or the fallback; catch sees the error
val v = parse(s) else continue; val w = parse(s) catch (e) { log(e); return }  // let-else: the else must leave (also T? and patterns: val Circle(r) = s else return 0.0)
if (val age = c.maxAge) out.append(age.toSeconds()); if (val a = f() && val b = g(a) && b > 1) ...   // binds a T? non-null for the rest of the && chain and the then-branch (D95); not in the else
```

Unused `Result` is an error. `try` needs an enclosing `throws`. `if (r.ok)` (or `r is Ok`)
smart-casts `r` to the payload; `r.message()` works on any error or error union.

## Collections

```veles
// fragment
val xs = [1, 2, 3]              // List<i64>;   mut [1, 2] is MutableList (untyped only)
val m = ["a": 1]                // Map<string, i64>;  mut ["a": 1]; typed: var m: MutableMap<string, i64> = [:]
var s = MutableSet<i64>()
xs.at(9) ?: -1; xs.at(0) ?: panic("why it is there"); xs.atOrDefault(9, -1); xs.at(-1); xs.len(); xs.contains(2); xs.first(); xs.last(); xs.indexOf(2)
// reads are copies; writes go through a reference: ml.ref(i)?.bump(); ml.ref(i)?.n = 0; ml.set(i, x)
// a T, not a T?, where the index is known: loop (i in xs.indices()) xs.at(i); if (xs.len() > 2) xs.at(2)
val [first, ..rest] = xs else return; when (args) { [] => 0; [cmd, ..] => 1 }   // list patterns (D62)
xs.map(f); xs.filter(p); xs.fold(0, f); xs.any(p); xs.all(p); xs.find(p); xs.forEach(f); xs.count(p); xs.mapNotNull(f); xs.partition(p)
shapes.filterIs<Circle>(); maybes.filterNotNull()   // narrowed lists: List<Circle>, List<T>
xs.take(2); xs.drop(2); xs.slice(1, 3); xs.zip(ys); xs.enumerate(); xs.flatMap(f); xs.distinct(); xs.chunked(2); xs.windowed(2)
xs.sorted(); xs.sortedBy(key); xs.sortedBy(e => (-e.size, e.name)); xs.sortedDescending(); xs.sortedWith((a, b) => ...); xs.reversed(); xs.min(); xs.max(); xs.sum(); xs.join(", "); xs.iter()  // tuples order element by element
xs.minBy(key); xs.maxBy(key); xs.minWith(cmp); xs.maxWith(cmp); xs.distinctBy(key); xs == ys  // lists, maps, sets compare by content
sorted.binarySearch(x); sorted.binarySearchBy(key, target); sorted.binarySearchWith(e => e.compareTo(x))  // first match or -1; the list must already be in order
sorted.lowerBound(x); sorted.upperBound(x); sorted.partitionPoint(e => e < x)   // insertion points; upperBound - lowerBound is how many times x occurs
ml.push(x); ml.pop(); ml.set(i, x); ml.insert(i, x); ml.removeAt(i); ml.addAll(ys); ml.sort(); ml.clear(); ml.toList(); xs.toMutable()
ml.swap(i, j); ml.fill(x); MutableList.repeat(false, n) /* MutableList<bool>: a generic static infers its type, D137 */; MutableList<MutableList<i64>>.make(n, _ => [])
val q = Deque<i64>(); q.addLast(x); q.addFirst(x); q.removeFirst(); q.removeLast(); q.first(); q.last(); q.at(-1); q.len()
val pq = PriorityQueue<i64>.natural(); pq.push(x); pq.pop(); pq.peek(); PriorityQueue<i64>(compare: (a, b) => b.compareTo(a))
m.get(k) ?: d; m.get(k) ?: panic("why"); m.getOrDefault(k, d); m.containsKey(k); m.keys(); m.values(); m.entries(); mm.set(k, v); mm.remove(k)
mm.ref(k)?.bump(); mm.ref(k)?.n += 1   // a pointer to the stored value; get returns a copy
s.add(x); s.contains(x); s.remove(x); s.toList()
xs.iter().filter(p).map(f).take(n).skip(n).enumerate().zip(ys.iter()).toList()
it.count(); it.fold(z, f); it.any(p); it.all(p); it.find(p); it.last(); it.forEach(f)
```

## Concurrency (D2/D3)

```veles
// fragment
scope {                          // every task started inside finishes here
  val t = async work(1)          // Task<i64>
  val v = await t
}
val (a, b) = gather { async f(); async g() }   // (Result<A, E|Panic>, Result<B, ...>); one task: its Result — when (gather { async f() }) { ... } (D103)
val winner = race { val m = ch.recv() => ...; sleep(Duration.millis(100)) => "timeout"; val v = t => ... }
race { queue.send(line) => { }; sleep(d) => dropped += 1 }   // a send arm: sent only if it wins (D108); arms tried in order, first ready wins
val ch = Channel<i64>(capacity: 8); ch.send(1); await ch.recv(); ch.close(); ch.closeAfter(n)  // closes itself after n sends
ch.trySend(1); ch.tryRecv()      // never wait: false when full, null when nothing is buffered
xs.mapConcurrent(f, workers: 4); xs.forEachConcurrent(f, workers: 4)   // the worker pool, results in order; f may suspend/throw
fun run(f: sendable fun(i64): i64)   // a function that may cross a task boundary: named, or a lambda over vals of Sendable types
await sleep(d)                   // a Duration: Duration.millis(100), Duration.seconds(5); zero yields
with server = async serve(l)     // a background task until the block ends: then cancelled, then joined; fail-fast; `await server` for its value (D100)
t.cancel()                       // stop a task at its next suspension point; its `with` cleanups run, the scope still joins it
val v = try withTimeout(Duration.seconds(1), () => try fetch())   // R throws E | Timeout; the task is cancelled and unwound before Timeout is thrown
// leaving a scope body early (return / throw / cancellation) cancels and joins its children
val m = Mutex(value: state); m.withLock(s => s.n += 1); m.get(); m.set(v)
with s = m.lock()                // s: *State, locked to the end of the block; nothing in it may suspend (D107)
val a = Atomic(value: 0); a.load(); a.store(1); a.swap(2); a.update(n => n + 1)
val sem = Semaphore(permits: 8); with sem.acquire(); sem.tryAcquire(); sem.available()   // (D110)
try retry(3, () => try fetch(), delay: Duration.millis(200))   // again after an error; the last error after the last call
ch.toList(); ch.forEach(v => ...)   // until closed and drained
with clock = time.ticker(Duration.seconds(1)); await clock.ticks.recv()   // a slow reader misses ticks
```

Suspension is inferred; `await` only on `sleep`, `recv`, task handles.
Data passed to `async` must be Sendable (D35): no `Mutable*`.

## Resources and unsafe

```veles
// fragment
with f = open("a")                            // closed when this block ends, and on every way out before (D43/D100)
with g = open("b")                            // the last opened closes first; must not be returned or stored outside the block
val h = open("c")                             // warning unless closed or handed on (returned, stored, passed, captured) (D115); val _ = open(p) drops it on purpose
with (f = open("a"), g = open("b")) { ... }   // block form: closes at its `}`; an expression: val text = with (f = open(p)) { f.readAll() }
with sem.acquire()                            // held and closed, never named (D109); `with conn` hands closing conn to the with
with srv = try serving()                      // a value with a Task field: received with `with` or returned; its tasks stop, then it closes (D111)
implement Closeable for File { fun close() { } }
extern "C" { fun strlen(s: *raw u8): i64 }
// SAFETY: p is a live NUL-terminated buffer  ← why the block is sound (a warning without it)
val n = unsafe { strlen(p) }                  // C calls and raw pointers need unsafe (D44)
unsafe { s.byteAtUnchecked(i) }               // also xs.atUnchecked(i), xs.setUnchecked(i, v): checked in debug only (D114)
with key = Secret.of(crypto.randomBytes(32))  // Secret<List<u8>>: prints [redacted], not Encodable, == constant time, wiped on close and when freed (D112)
val url = cfg.databaseUrl.expose()            // the only way to the value: a fresh copy
```

## Files, paths, processes

```veles
// fragment
use fs; use path; use os
val text = try fs.readFile(p); try fs.writeFile(p, text); try fs.appendFile(p, "x")
fs.exists(p); fs.isFile(p); fs.isDir(p); try fs.stat(p) /* size, modified, isDir */; with (f = try fs.open(p)) { try f.read(4096); try f.readAt(off, n) }; try fs.listDir(d); try fs.walk(d); try fs.mkdir(d); try fs.remove(p); try fs.rename(a, b)
path.join(a, b, c); path.join(parts...); path.dir(p); path.base(p); path.stem(p); path.ext(p); path.isAbsolute(p); path.clean(p); path.within(root, p)  // within: the check before opening a file named from outside
os.args(); os.env("HOME"); os.pid(); try os.hostname(); os.tempDir(); os.exit(1); val r = try os.run("clang", ["--version"]); r.code; r.stdout; r.stderr; r.ok(); try os.run("git", ["apply", "-"], input: patch)
val sb = StringBuilder(); sb.append("a"); sb.appendLine("b"); sb.toString()   // linear-time building
```

Everything that can fail throws `IoError { path, code, detail }`.

## Networking

```veles
// fragment
use net
with listener = try net.listen(host: "", port: 8080)   // defaults: loopback, any free port (listener.port())
scope { loop { val conn = try listener.accept(); async handle(conn) } }   // Conn is Sendable
with conn = try net.connect("example.org", 80)
try conn.writeText("ping\n"); val line = try conn.readLine() ?: "closed"   // readLine: string?, null at end of stream
val body = try conn.readExact(n); val chunk = try conn.read(); try conn.write(bytes); try conn.shutdownWrite()
val line = try withTimeout(Duration.seconds(5), () => try conn.readLine())   // throws IoError | Timeout
```

Every waiting call suspends the task; failures throw `IoError` with the address in `path`.

```veles
// fragment
use http
val app = http.Router()
app.get("/users/{id}", req => http.Response.json(try find(req.param("id")) ?! http.notFound()))   // Fail → its status, other errors → 500
app.get("/static/*", http.files("./public"))                                          // 304/206/416, no-cache; maxAge:, immutable:, index:, dotfiles:, redirect: (D96)
with listener = try net.listen(host: "", port: 8080)
http.serve(listener, app.handler())
```

## Attributes (D51)

`@deprecated("msg")`, `@mustUse`, `@inline`, `@noinline`; for the wire: `@key`, `@skip`, `@required`, `@tag`.

## Tests (D78)

```text
test "parses a port" {                       // a sentence names it; no signature
  expect(parsePort("80") == 80)               // soft: records (with both sides) and goes on
  val cfg = require(load("app.toml"))         // T? or Result: the value, or the test ends
  expectThrows<RangeError>(() => parsePort("70000"))
  expectPanics(() => { val _ = [1].at(5) ?: panic("no") })
  fail("not written yet")                    // ends the test
}
test fun expectSorted(xs: List<i64>) { ... }  // helper: test code only, left out of builds
suite "parser" { test "..." { } suite "..." { } test fun h() { } }  // a group; its helpers are its own
parser.test.vs                                // a file of tests, the suite "parser"; never built into the program
veles test --filter "parser / rejects"        // a test's full name: its suites and its own, joined with " / "
assert(n > 0, "why it must hold")            // anywhere: an invariant; panics with the reason and both sides
```

## Tooling

```text
veles new <dir>        veles run [dir]        veles build [dir] -o app [--release]
veles doc [dir] [-o out]   (the public API as Markdown)
veles check <dir>      veles test <dir> [--filter text] [--timeout 10m]
veles parse <file>     veles lsp     (editor server)
veles fmt <paths>      [--check | --stdout]   format in place; [format] in veles.toml: indent = 2 | "tab", max_blank_lines = 1
VELES_CLANG=<path>     VELES_GC_TRACE=1     VELES_GC_THRESHOLD=<bytes>
```
