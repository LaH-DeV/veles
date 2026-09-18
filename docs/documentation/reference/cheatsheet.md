# Veles cheat sheet

Everything on one page. `D<n>` refers to a decision in `veles-spec.md`.

## Program structure

```veles
// fragment
use io                          // import a module; call as io.println
use geometry.{ Point as P, norm } // import names directly, with renames

const limit = 10                // module-level constant
val banner = "hi"               // module-level value
var counter = 0                 // module-level variable

fun main() { }                  // entry point
```

A module is a directory; every `.vs` in it shares a namespace (M2).
`pub` exports from the module (M5). `veles.toml` names the package and
its `exports` and `[dependencies]` (M1).

## Bindings and types

```veles
// fragment
val x = 1                       // immutable, inferred i64
var y: i64 = 2                  // mutable, explicit type
val (a, b) = (1, "one")         // tuple destructuring; (a, b) = (b, a) assigns (swap)
```

| Kind | Types |
|---|---|
| integers | `i8 i16 i64 i64 u8 u16 u32 u64` |
| floats | `f32 f64` |
| other scalars | `bool string` |
| nullable | `T?` (`(*T)?` for a nullable pointer) |
| pointer | `*T` (GC-managed), `*raw T` (unsafe, D50) |
| tuple | `(A, B)`, unit is `()` |
| function | `fun(A, B): R`, with `suspends` / `throws E` |
| collections | `List<T> MutableList<T> Map<K,V> MutableMap<K,V> Set<T> MutableSet<T>` |
| ranges | `Range<T>` from `a..b` (inclusive) or `a..<b` |
| concurrency | `Channel<T>`, `Task<T>`, `Mutex<T>`, `Atomic<T>` |
| results | `Result<T, E>`; `T?` is `Option<T>` |

## Operators

| | |
|---|---|
| arithmetic | `+ - * / %` (checked, D21); wrapping `+% -% *%` |
| bitwise | `& \| ^ << >> ~` on integers; `&` and shifts bind like `*`, `\|` `^` like `+` |
| comparison | `== != < <= > >=` |
| logic | `&& \|\| !` |
| conversion | `x as T` (numeric only) |
| nullable | `x ?: fallback`, `x?.member`, `x?.method()` |
| type test | `x is T`, `x !is T` |
| address | `&x` → `*T`, `*p` reads through |
| range | `a..b`, `a..<b` |
| assignment | `= += -= *= /= %=` |
| strings | `"a" + "b"`, `s.len()` (bytes), `s.trim()`, `s.split(",")`, `s.replace(a, b)`, `s.indexOf(p)`, `s.toUpper()`, `s.padStart(n)`, `s.toInt()`, `s.chars()`, `s.byteAt(i)` (a `u8`; `'"'` is a byte literal), `"$name ${expr}"`, escapes `\n \t \\ \" \$ \u{..}` |

## Control flow

```veles
// fragment
if (c) a else b                       // expression
loop { ... break }                    // forever
loop (cond) { ... }                   // while
loop (x in xs) { ... }                // any Iterable, Range, Map ((k, v) in m); (a..b).step(n), (a..b).reversed()
loop :outer (x in xs) { continue outer; break outer }
when (v) { 1 => "one"; 2, 3 => "few"; else => "many" }
when { x < 0 => "neg"; else => "pos" }
when (shape) { is Circle(r) => ...; is Rect(w, h) if w > h => ...; is Point => ... }
when (opt) { Some(x) => ...; null => ... }
when (val r = res) { is Ok => r; is Err => r.message() }   // r is the payload in each arm; Ok(v) / Err(e) also work
return v; break; continue; throw e; panic("msg")   // panic never returns (D20)
```

## Functions and lambdas

```veles
// fragment
fun add(a: i64, b: i64): i64 = a + b                // expression body
fun greet(name: string, punct: string = "!") { }    // default; call greet("x", punct: "?")
fun sum(xs: i64...): i64 = xs.fold(0, (a, b) => a + b)  // variadic: sum(1, 2), sum(list...)
fun <T: Show> show(x: T): string = x.show()         // generic with bound
fun fetch(url: string): string suspends throws E    // effects (inferred for free functions)
val f = x => x * 2                                  // lambda; (a, b) => ..., (x: i64) => ...
val g = () => { var n = 0; n }                      // block body
val h = (a: i64) => (b: i64) => a + b               // curried
```

Lambdas capture by reference (D37). Last expression of a block is its
value.

## Structs, traits, sealed types

```veles
// fragment
struct Point {
  x: i64
  y: i64 = 0                    // default
  fun len(): i64 = self.x + self.y
  mut fun move(dx: i64) { self.x += dx }   // D22: only on a var
  static fun origin(): Point = Point(x: 0)  // no self; Point.origin()
}
val p = Point(x: 1)             // named construction; p == q, "$p" work
val n = i64.parse("42")         // i64?; Parsable — T.parse(s) in generic code

trait Shape {
  fun area(): f64
  fun describe(): string = "area ${self.area()}"   // default
}
struct Sq { s: f64; impl Shape { fun area(): f64 = self.s * self.s; override fun describe(): string = "sq" } }   // your own type: impl in the body
impl Shape for i64 { fun area(): f64 = 0.0 }   // a foreign type: top-level impl (for your own type it is a lint with a quick fix)
val s: Shape = Sq(s: 2.0)       // trait object (D9)

extend Point {                  // more inherent methods, outside the body (D23)
  fun norm(): i64 = self.x.abs() + self.y.abs()
}
extend<T: Shape> Box<T> { ... } // only for types you declare; bounds allowed

sealed trait Expr {             // fixed set of variants (D12)
  fun eval(): i64 = when (self) { is Num(v) => v; is Neg(e) => -e.eval() }
}
struct Num : Expr { v: i64 }
struct Neg : Expr { e: *Expr }

trait Iterator { type Item; mut fun next(): Item? }   // associated type
impl Iterator for Countdown { type Item = i64; mut fun next(): i64? { ... } }

impl Comparable for Point { fun compareTo(other: Point): i64 = self.x - other.x }  // <, sorted, min
impl Display for Point { fun toString(): string = "(${self.x})" }              // "$p"
// also Equatable (==) and Hashable (map keys); structural by default
```

## Errors (D4)

```veles
// fragment
error NotFound { key: string }                          // a struct that is an Error; only errors can be thrown
error Invalid { why: string; fun message(): string = self.why }
error Failed { message: string }                        // a `message` field is the message
error GetErrors = NotFound | Invalid | Failed             // a named error set (D45)
error Wrapped { cause: GetErrors }                      // an error's field may hold a set
fun get(k: string): string throws NotFound = if (k == "a") "A" else throw NotFound(key: k)
fun getAll(): string throws = try get("a") + try get("b")   // error type inferred: NotFound
when (val r = get("z")) { is Ok => r; is Err => r.key }    // caller sees Result; r is the payload per arm
val r: Result<i64, NotFound> = Ok(1)
```

Unused `Result` is an error. `try` needs an enclosing `throws`. `if (r.ok)` (or `r is Ok`)
smart-casts `r` to the payload; `r.message()` works on any error or error union.

## Collections

```veles
// fragment
val xs = [1, 2, 3]              // List<i64>;   mut [1, 2] is MutableList (untyped only)
val m = ["a": 1]                // Map<string, i64>;  mut ["a": 1]; typed: var m: MutableMap<string, i64> = [:]
var s = MutableSet<i64>()
xs.at(9) ?: -1; xs.atOrPanic(0); xs.atOrDefault(9, -1); xs.at(-1); xs.len(); xs.contains(2); xs.first(); xs.last(); xs.indexOf(2)
xs.map(f); xs.filter(p); xs.fold(0, f); xs.any(p); xs.all(p); xs.find(p); xs.forEach(f); xs.count(p)
xs.take(2); xs.drop(2); xs.slice(1, 3); xs.zip(ys); xs.flatMap(f); xs.distinct(); xs.chunked(2); xs.windowed(2)
xs.sorted(); xs.sortedBy(key); xs.sortedDescending(); xs.reversed(); xs.min(); xs.max(); xs.sum(); xs.join(", "); xs.iter()
ml.push(x); ml.pop(); ml.set(i, x); ml.insert(i, x); ml.removeAt(i); ml.addAll(ys); ml.sort(); ml.clear(); ml.toList(); xs.toMutable()
m.get(k) ?: d; m.getOrPanic(k); m.getOrDefault(k, d); m.containsKey(k); m.keys(); m.values(); m.entries(); mm.set(k, v); mm.remove(k)
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
val (a, b) = gather { async f(); async g() }   // (Result<A, E|Panic>, Result<B, ...>)
val winner = race { val m = ch.recv() => ...; sleep(100) => "timeout"; val v = t => ... }
val ch = Channel<i64>(capacity: 8); ch.send(1); await ch.recv(); ch.close()
await sleep(ms)
val m = mutex(state); m.withLock(s => s.n += 1); m.get(); m.set(v)
val a = atomic(0); a.load(); a.store(1); a.swap(2)
```

Suspension is inferred; `await` only on `sleep`, `recv`, task handles.
Data passed to `async` must be Sendable (D35): no `Mutable*`.

## Resources and unsafe

```veles
// fragment
with (f = open("a"), g = open("b")) { ... }   // close() on every exit (D43)
impl Closeable for File { mut fun close() { } }
extern "C" { fun strlen(s: *raw u8): i64 }
val n = unsafe { strlen(p) }                  // C calls and raw pointers need unsafe (D44)
```

## Files, paths, processes

```veles
// fragment
use fs; use path; use os
val text = try fs.readFile(p); try fs.writeFile(p, text); try fs.appendFile(p, "x")
fs.exists(p); fs.isFile(p); fs.isDir(p); try fs.listDir(d); try fs.mkdir(d); try fs.remove(p); try fs.rename(a, b)
path.join(a, b, c); path.join(parts...); path.dir(p); path.base(p); path.stem(p); path.ext(p); path.isAbsolute(p)
os.args(); os.env("HOME"); os.exit(1); val r = try os.run("clang", ["--version"]); r.code; r.stdout; r.ok()
val sb = stringBuilder(); sb.append("a"); sb.appendLine("b"); sb.toString()   // linear-time building
```

Everything that can fail throws `IoError { path, code, detail }`.

## Attributes (D51)

`@test`, `@deprecated("msg")`, `@mustUse`, `@inline`, `@noinline`.

## Tooling

```text
veles run <dir>        veles build <dir> -o app [--release]
veles check <dir>      veles test <dir>
veles parse <file>     veles lsp     (editor server)
veles fmt <paths>      [--check | --stdout]   format in place; [format] in veles.toml: indent = 2 | "tab", max_blank_lines = 1
VELES_CLANG=<path>     VELES_GC_TRACE=1     VELES_GC_THRESHOLD=<bytes>
```
