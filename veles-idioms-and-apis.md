# Idiomatic Veles, what to improve, and high-level APIs to add

2026-10-14, second pass. The user asked: what does idiomatic Veles look like
(with many examples) and where can it improve; and which high-level APIs
(like Bluetooth) could be added, for different purposes — a menu to choose
from.

This builds on `veles-direction-2026-10-14.md` (same day), whose questions
became **D157–D167** and are decided but not built yet. Nothing here
re-asks them; where code below uses them it says so.

How the code is marked:

- **today** — compiled and run with the compiler as it is (scratchpad probes
  `g1`–`g3`, `p1`–`p9`; outputs quoted where they matter);
- **decided** — spelled as D157–D167 say; does not compile until built;
- **idea** — nothing decided. Ideas are numbered **I1…** (language and
  tooling) and **A1…** (APIs) so you can answer "yes to I3, A12, A30".

---

## Part 1 — What idiomatic Veles is

### 1.1 The rules, each in one line of code

| # | Rule | In code |
|---|---|---|
| 1 | Absence is a type and leaves in one token | `val boss = u.manager ?: return "ops@example.org"` |
| 2 | Failure is a type and leaves in one token | `val n = try line.toInt() ?! BadLine(line: i)` |
| 3 | Prove an access instead of panicking | `val [cmd, ..rest] = args else return usage()` |
| 4 | Closed sets are `sealed` or `enum`; `when` is exhaustive | `when (p) { is Pending(a) => …; is Settled(a, r) => … }` |
| 5 | Values by default; sharing is a visible handle | `struct` copies; `MutableList`, `*T`, `Mutex<T>` are handles |
| 6 | A resource closes with its block | `with f = try fs.open(p)` |
| 7 | Concurrency has no colour; tasks never outlive their block | `val (a, b) = awaitAll(async f(), async g())` |
| 8 | Types instead of bare numbers where mixing is possible | `Duration.millis(20)`, `Timestamp`, `Digest`, `Secret<T>` |
| 9 | Compute at compile time what does not change | `const CRC: List<u32> = crcTable()` + `static assert` |
| 10 | The low level is the same language, fenced | `unsafe { strlen(c.ptr()) }` under a `// SAFETY:` line |
| 11 | Names are words; one obvious spelling | `indexOfFirst`, `readLine(max:)`, never `idx`, `rl` |
| 12 | A footgun is a compile error, not a doc line | interpolating a `Result` is refused (D157) |

### 1.2 Gallery — today

#### Data with a closed set of states, errors as types (`g1`, runs)

```veles
sealed trait Payment {
  fun describe(): string => when (this) {
    is Pending(amount)            => "waiting for ${money(amount)}"
    is Settled(amount, reference) => "paid ${money(amount)} ($reference)"
    is Refused(reason)            => "refused: $reason"
  }
}

struct Pending : Payment {
  amount: i64
}

struct Settled : Payment {
  amount:    i64
  reference: string
}

struct Refused : Payment {
  reason: string
}

error CardDeclined {
  code: string
}

error Unreachable {
  host: string
}

error ChargeErrors = CardDeclined | Unreachable

fun charge(card: string, cents: i64): string throws ChargeErrors {
  if (card.startsWith("0000")) throw CardDeclined(code: "insufficient_funds")
  if (card.isEmpty()) throw Unreachable(host: "psp.example")
  "ref-${card.len()}-$cents"
}

/// Every outcome of a charge becomes a state; the compiler checks all are handled.
fun settle(card: string, cents: i64): Payment => when (val r = charge(card, cents)) {
  is Ok  => Settled(amount: cents, reference: r)
  is Err => when (r) {
    is CardDeclined => Refused(reason: r.code)
    is Unreachable  => Pending(amount: cents)
  }
}
```

Prints `paid 19.99 EUR (ref-4-1999)`, `refused: insufficient_funds`,
`waiting for 7.00 EUR`. No `Ok(…)`/`Err(…)` is spelled; `r` is the payload
in each arm; adding a fourth error to `ChargeErrors` breaks `settle` at
compile time.

#### Nullables and collections (`g1`, runs)

```veles
struct User {
  name:    string
  manager: (*User)? = null
  email:   string? = null
}

fun escalationAddress(u: User): string {
  val boss = u.manager ?: return "ops@example.org"
  boss.email ?: "${boss.name.toLower()}@example.org"
}

fun topRegions(sales: List<Sale>, n: i64): List<(string, i64)> {
  val totals: MutableMap<string, i64> = [:]
  loop (s in sales) totals.set(s.region, totals.getOrDefault(s.region, 0) + s.cents)
  totals.entries().sortedBy(e => (-e.1, e.0)).take(n)       // biggest first, ties by name
}

loop ((region, cents) in topRegions(sales, 2)) println("$region ${money(cents)}")
val (big, small) = sales.partition(s => s.cents >= 500)
if (val cheapest = sales.minBy(s => s.cents)) println("cheapest: ${cheapest.item}")
```

#### A value type with operators; a generic container (`g2`, runs)

```veles
struct Money {
  cents: i64

  implement Addable {
    fun plus(other: Money): Money => Money(cents: this.cents + other.cents)
  }

  implement Comparable {
    fun compareTo(other: Money): Ordering => this.cents.compareTo(other.cents)
  }

  implement Display {
    fun toString(): string => "${this.cents / 100}.${"${this.cents % 100}".padStart(2, "0")}"
  }
}

val total = basket.fold(Money(cents: 0), (a, b) => a + b)    // "total 22.50, dearest 19.99"
println("total $total, dearest ${basket.max()}")

/// A bounded cache that forgets the least recently used entry.
struct Lru<K, V> {
  capacity: i64
  private values: MutableMap<K, V> = [:]
  private order:  Deque<K> = Deque<K>()

  fun get(key: K): V? {
    val v = this.values.get(key) ?: return null
    this.touch(key)
    v
  }

  fun put(key: K, value: V) {
    if (!this.values.containsKey(key) && this.values.len() >= this.capacity) {
      if (val oldest = this.order.removeFirst()) this.values.remove(oldest)
    }
    this.values.set(key, value)
    this.touch(key)
  }
  …
}
```

(`struct Lru<K: Hashable, V>` — what most people write first — is refused
for `string` keys; see F2.)

#### Compile time (`g2`, runs)

```veles
const fun crcTable(): List<u32> {
  val table: MutableList<u32> = []
  loop (n in 0..<256) {
    var c = n.toU32() ?: 0
    loop (_ in 0..<8) c = if (c & 1 != 0) 0xEDB88320 ^ (c >> 1) else c >> 1
    table.push(c)
  }
  table.toList()
}

const CRC: List<u32> = crcTable()                         // built by the compiler, read-only in the binary
static assert(CRC.len() == 256, "one entry per byte value")

fun crc32(bytes: List<u8>): u32 {
  var c: u32 = 0xFFFFFFFF
  loop (b in bytes) c = (CRC.at(((c ^ b.toU32()) & 0xFF).toI64()) ?: 0) ^ (c >> 8)
  c ^ 0xFFFFFFFF
}
```

The same function runs at compile time and at run time; `crc32` of the
fox sentence prints `1095738169` (0x414FA339, the published value).

#### Concurrency (`g3`, runs)

```veles
/// Every price at once, four in flight, in input order; a missing item is null.
fun prices(items: List<string>): List<i64?> =>
  items.mapConcurrent(item => lookupPrice(item).getOrNull(), workers: 4)

/// The first mirror to answer wins; the other is cancelled (D169 — before
/// D168/D169 this took a scope, two handles and a manual `cancel()` in each
/// arm, or it silently waited for the slow mirror).
fun fastest(): string => race {
  async mirror("eu", Duration.millis(30))
  async mirror("us", Duration.millis(10))
}

/// A price against a deadline: the select's own task is cancelled if the timer wins.
fun priceOrZero(item: string): i64 throws Unavailable => select {
  val price = async lookupPrice(item) => price
  sleep(Duration.millis(50))          => 0
}

/// A producer and a consumer joined by a bounded channel; the producer is a
/// background task stopped when the function returns.
fun pipeline(): i64 {
  val lines = Channel<string>(capacity: 4)
  with producer = async produce(lines)
  var bytes = 0
  loop {
    val line = await lines.recv() ?: break
    bytes += line.len()
  }
  bytes
}

fun record(stats: Mutex<Stats>, i: i64) {
  with s = stats.lock()                         // held to the end of the block; nothing in it may suspend
  if (i % 3 == 0) s.misses += 1 else s.hits += 1
}

val (a, b) = awaitAll(async lookupPrice("tea"), async lookupPrice("cake"))
val slow = withTimeout(Duration.millis(20), () => mirror("slow", Duration.millis(200))) ?? "gave up"
scope {
  loop (i in 0..<100) async record(stats, i)
}
```

`lookupPrice` sleeps and throws, yet nothing marks it `async`; its callers
are ordinary calls. Prints `[300, null, 400]`, `fastest mirror: us`,
`piped 61 bytes`, `both: 300 400`, `gave up`, `hits 66, misses 34`.

#### A web service (from `examples/httpd`, runs)

```veles
fun app(dir: string): http.Handler {
  val notes = Mutex(value: Notes())
  val router = http.Router()
  router.wrap(http.requestId())
  router.wrap(http.timeout(Duration.seconds(1)))
  router.get("/static/*", http.files(dir))

  router.post("/api/notes", req => {
    val text = try req.text().trim()
    if (text.isEmpty()) throw http.badRequest("a note needs some text")
    val note = notes.withLock(n => n.add(text))
    http.Response.json(try json.encode(note), status: http.Status.created)
      .withHeader(http.Header.location, "/api/notes/${note.id}")
  })

  router.get("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val note = try notes.withLock(n => n.find(id)) ?! http.notFound("no note $id")
    http.Response.json(try json.encode(note))
  })
  router.handler()
}
```

`?!` turns "absent" into the right HTTP status in one token; the JSON is
derived (`implement Codable` on `Note`).

#### Tests (D78; `g4`, passes)

```veles
suite "money" {
  test "adds in cents" {
    expect(Money(cents: 250) + Money(cents: 1) == Money(cents: 251))
  }
  test "a declined card is refused, not pending" {
    val p = settle("0000-1", 500)
    val Refused(reason) = p else fail("expected Refused, got $p")
    expect(reason == "insufficient_funds")
  }
}
```

The low-level gallery (extern, `@packed`, `Array<u8, N>`, atomics, a named
thread pool) is in `veles-direction-2026-10-14.md` §3 and still current.

### 1.3 The same code once D158 and D163–D167 are built (decided)

```veles
// today                                              // decided (D158, D163, D164, D165, D166)
fun remove(id: i64): bool {                           fun remove(id: i64): bool {
  val at = this.items.indexOfFirst(x => x.id == id)     val at = items.indexOfFirst(x => x.id == id) ?: return false
  if (at >= 0) this.items.removeAt(at)                  items.removeAt(at)
  at >= 0                                               true
}                                                     }

throw UsageError(message: "unknown option '$arg'")    throw UsageError("unknown option '$arg'")
val notes = Mutex(value: Notes())                     val notes = Mutex(Notes())
val moved = Point(x: 5, y: p.y, label: p.label)       val moved = p.copy(x: 5)
.sortedBy(e => (-e.1, e.0))                           .sortedBy(e => (-e.value, e.key))
fun divide(a: i64, b: i64): (i64, i64)                fun divide(a: i64, b: i64): (quotient: i64, remainder: i64)

struct Options { var port: i64 = 8080 … 30-line parse }   struct Options { port: i64 = 8080 … implement cli.Args }
                                                      val opts = try cli.parse<Options>(os.args())   // D167
```

### 1.4 Smells and their idioms

| Seen in code | Write instead | Why |
|---|---|---|
| `val x = f(); if (x != null) { … }` | `if (val x = f()) { … }` | D95: binds the non-null value for the branch |
| `if (r.ok) r else return` / `when` just to unwrap | `val v = r else return` | let-else |
| `val v = f() catch (e) { throw Other(…) }` | `val v = try f() ?! Other(…)` (or `.mapError(…)` to keep the cause) | one token |
| three `try`s each with the same `catch` | `do { … try … } catch (e) { … }` | one handler (D98) |
| `xs.at(i) ?: panic("unreachable")` | a list pattern, an index loop the checker proves, or a panic that says *why* | D62/D64 |
| `idx >= 0` after `indexOf` | `?: return` on the `i64?` (after D165) | the absent case is a type |
| `.0` / `.1` on a returned pair | destructure `val (q, r) = …`, or labels (D166) | names over positions |
| a manual `var out = mut []; loop … push` | `map` / `filter` / `mapNotNull` / `partition` | eager and as fast (inlined) |
| `val _ = f()` for a value that is not a `Result` | just `f()` | noise |
| a `scope { val t = async f(); await t }` | `f()` | the call already waits (D2) |
| `scope { val a = async f(); val b = async g(); (await a, await b) }` | `awaitAll(async f(), async g())` | D156 |
| a `Mutex` held across a suspending call | refused; copy out with `withLock`, then call | D107 |
| `Thread` + a hand-rolled queue for CPU work | `with cpu = try Executor.pool(threads: n)` + `scope(on: cpu)` | D143 |
| an `i64` of milliseconds | `Duration` | D60 |
| a `string` holding a password or token | `Secret<string>` | D112: never printed, wiped |
| a hand-written `toJson()` | `implement Codable` | D58 |
| `if (os == "windows")` at run time | `build.target.os` constant branch or a `_windows.vs` file (D160) | dead branch not compiled |
| a three-line `unsafe { }` around one call with no `SAFETY:` | one line, with the `// SAFETY:` reason above | D44 |
| abbreviations (`l`, `sb`, `cs`, `rr`) | words (`line`, `out`, `chars`, `result`) | house style |
| a `foo/bar` example | a tool someone would run | notes R5 |

---

## Part 2 — Where it could be better

### 2.1 Defects found by this pass (decision-free; recorded in checklist §11)

| # | Found | Today | Should be |
|---|---|---|---|
| F1 | Destructuring a plain struct in a `val` demands `else` | `val Point(x, y) = p` → "a pattern in a 'val' can fail to match, so it needs 'else'" | a struct pattern on a value of exactly that struct type (not a sealed variant) cannot fail: accepted without `else` |
| F2 | `Equatable`/`Hashable` as bounds refuse the types they describe | `fun f<T: Hashable>` rejects `i64`, `string` and any struct ("does not implement trait 'Hashable'"), while `T: Comparable` accepts `i64`/`string`, and `==`/map keys work structurally without a bound | either every structurally comparable type satisfies `Equatable`/`Hashable` (the traits stay the hook for custom behaviour), or the error says "no bound is needed: `==` and map keys work on every type" — the first changes what compiles, so it is idea **I6** |
| F3 | Wrong fix in a nullable-pointer error | `manager: *User? = null` → "the type must be nullable ('*User??')" | "`*User?` is a pointer to a nullable `User`; a nullable pointer is `(*User)?`" |
| F4 | Poor error for `count()` without a predicate on a `List` | `s.chars().count()` → "cannot infer type parameter 'E' of 'count'" | "`List.count` takes a predicate; the number of elements is `len()`" |
| F5 | Cheat sheet showed a race arm on a bare task | `val v = t => …` (an error: "a race arm waits on … 'await task'") | `val v = await t => …` — **fixed in this change** |

### 2.2 Language and tooling ideas

Each: the problem with evidence, the proposal, what it touches, size
(S/M/L), and whether it helps the compiler port (self-hosting).

#### I1 — Generators: `yield` in a function returning an `Iterator`

```veles
// today: an iterator is a struct with state and next(); a tree walk needs an explicit stack
struct Walk : Iterator {
  type Item = *Node
  private stack: MutableList<*Node>
  fun next(): (*Node)? { … 15 lines … }
}

// idea
fun walk(root: *Node): Iterator<*Node> {
  yield root
  loop (child in root.children) loop (n in walk(child)) yield n
}

loop (n in walk(tree).filter(n => n.isLeaf()).take(10)) …     // lazy: stops after 10
```

`yield` is an error today (probe `p2`). Lazy sequences are how a lexer
hands tokens to a parser, how `fs.walk` could stream instead of returning a
`List`, how a paged HTTP API becomes a loop. The compiler already turns
suspending functions into state machines (coroutine frames); a generator is
the same transformation with `yield` as the suspension point, so the cost
is mostly checking rules: a generator may suspend (an async generator =
`loop (row in db.query(…))` streaming rows) and throw (the error surfaces at
the `loop`). **M–L. Self-host: yes** (tree walks are everywhere in `sema`).

#### I2 — Multi-line and raw strings

```veles
// idea: the closing delimiter's column is the margin (Swift / Java text blocks)
val page = """
  <h1>${title}</h1>
  <p>${body}</p>
  """
val query = sql"""
  SELECT id, name
  FROM users
  WHERE team = ${team}
  """
val pattern = #"C:\Users\(\w+)\Documents"#         // raw: no escapes, no interpolation; ##"…"## to hold "#
```

Both are errors today (`p3`). Every SQL template (D129), HTML, regex, test
fixture and generated C/LLVM text is written with `\n` and `+` now; the
code generator in Veles will emit textual IR all day. **S–M. Self-host:
yes** (codegen, test goldens).

#### I3 — Membership with `in`

```veles
if (word in STOP_WORDS) continue          // idea; today: STOP_WORDS.contains(word)
if (c !in 'a'..'z') return false
```

`in` already works in `when` arms (`in 90..100 => "A"`, probe `p1`), and is
the loop keyword; outside them it is a parse error. Sugar for `contains` on
anything with a `contains` method (lists, sets, map keys, ranges, strings).
**S.**

#### I4 — Format specs in interpolation

```veles
println("${name:<20} ${price:>10.2} ${share:.1%} ${id:08x}")   // idea
// today
println("${name.padEnd(20)} ${price.toFixed(2).padStart(10)} ${(share * 100.0).toFixed(1)}% ${id.toHex().padStart(8, "0")}")
```

Reports, tables and CLI output are the most common output of a tool. The
spec after `:` is checked at compile time against the value's type (`.2`
on a string is an error), so there is no run-time format string to get
wrong. Alternative with no new syntax: a small set of methods
(`x.format(width:, precision:, align:)`). **S–M.**

#### I5 — `async` on a block (decided and built as D169, 2026-10-16)

```veles
scope {
  async {                                   // idea; today: a named function, then `async name(args)`
    loop (i in 1..10) lines.send("line $i")
    lines.close()
  }
  …
}
```

Today `async` must prefix a call (probe `g3`: "'async' must prefix a call
expression"), so a three-line producer
becomes a top-level function with its captured values as parameters. The
block is a `sendable` lambda launched at once, under exactly D35/D156's
rules (it can capture only Sendable values; same placement rules as
`async f()`). **S.**

#### I6 — Structural traits satisfy their bounds (fixes F2)

`T: Equatable`, `T: Hashable`, `T: Display` hold for every type whose
`==`/hash/text the compiler already provides (all but function values and
types that opt out like `Secret`); implementing the trait overrides it. Then
`struct Lru<K: Hashable, V>` reads as intended and documents the
requirement. **S. Changes what compiles** (only accepts more).

#### I7 — The missing collection vocabulary

Kotlin-named, eager on lists like the rest (all absent today, probe `p4`):

| Name | Does | Common use |
|---|---|---|
| `groupBy(key)` → `Map<K, List<T>>` | group | reports |
| `associateBy(key)` / `associate(f)` → `Map` | index a list | lookups by id |
| `sumOf(f)`, `maxOf(f)`, `minOf(f)` | aggregate a projection | totals |
| `countBy(key)` → `Map<K, i64>` | frequencies | word counts, histograms |
| `flatten()` | `List<List<T>>` → `List<T>` | |
| `zipWithNext()` | adjacent pairs | deltas, slopes |
| `runningFold(z, f)` | prefix sums | cumulative series |
| `single()` → `T?` | the only element | "exactly one match" |
| `firstNotNull(f)` | first non-null projection | parsers, resolution |
| `MutableMap.merge(k, v, f)` / `increment(k)` | update in place | counters |
| `Map.toList()`, `List<(K, V)>.toMap()` | conversions | |

**S each; one decision batch for the names.**

#### I8 — Collections a compiler and a server want

`SortedMap<K, V>` / `SortedSet<T>` (B-tree, range queries:
`m.range(from, to)`), `BitSet` (dataflow, liveness), `SmallList<T, N>`
(inline storage for the common 0–4 elements, D121 arrays underneath), an
`Interner` (`Symbol` ids for names: O(1) compare), and an `Arena` for
allocation-heavy phases (`with arena = Arena()`; everything allocated in
the block freed at once — AST, per-request scratch). **M. Self-host: yes**
(the Go compiler leans on maps of interned names and sorted output).

#### I9 — Methods on enums (reopens D57)

```veles
// today: "cannot extend enum 'Level': … write functions that take one (D57)"
fun label(level: Level): string => when (level) { … }
println(label(entry.level))

// idea: an `extend` with methods only — no fields, no state, no implement
extend Level {
  fun label(): string => when (this) { Level.Debug => "dbg"; … }
  fun isSevere(): bool => this >= Level.Warn
}
println(entry.level.label())
```

D57 kept enums compiler-owned; methods with no state do not touch what it
protected (layout, `Codable`, compare/hash). Asked only if you want to
reopen it. **S.**

#### I10 — Variants declared inside the sealed trait

```veles
// today: one struct declaration per variant, each naming the trait
sealed trait Token { }
struct Num : Token { value: i64 }
struct Name : Token { text: string }
struct Plus : Token { }

// idea: a short form for data-only variants (methods still go on the trait or an extend)
sealed trait Token {
  Num(value: i64)
  Name(text: string)
  Plus
}
```

`p6` shows nested `struct`s in a trait are refused. The self-hosted AST
declares about 90 variants (`selfhost/ast/ast.vs`), most of them data only. Variant
names stay top-level (`Num`, not `Token.Num`) as today. **M. Self-host:
yes.**

#### I11 — Tests: properties, snapshots, tables, virtual time

```veles
test "parse inverts print" for all (e: Expr) {        // generated, shrunk on failure; Expr via a derive
  expect(parse(print(e)) == e)
}

test "formats a report" {
  expectSnapshot(report(sample))                     // stored beside the test; `veles test --update` accepts
}

test "rounds" for ((input, want) in [(1.25, "1.3"), (1.35, "1.4")]) {   // one case per row in the report
  expect(input.toFixed(1) == want)
}

test "a slow server times out" {
  with clock = time.Simulated()                      // sleeps, tickers, withTimeout use it: instant, deterministic
  val r = withTimeout(Duration.seconds(5), () => slowCall())
  clock.advance(Duration.seconds(6))
  expect(r is Err)
}
```

The simulated clock is the same idea as D159's simulated Bluetooth
adapter; with it, every timeout/retry/ticker path is testable without
waiting. **M (each S–M).**

#### I12 — Contracts that feed the bounds facts

```veles
fun tokenAt(i: i64): Token requires i in 0..<toks.len() => toks.at(i)   // idea: no `?: panic` inside
```

84 `at(…) ?: panic("…")` sites in the tree prove by sentence what the
checker could prove by rule. A `requires` clause is checked at each call
(statically where the facts suffice, else a debug-build assertion naming
the caller) and is a fact inside the body. Research-sized; worth it for
both safety and the compiler port's speed (no checks in hot loops).
**L.**

#### I13 — Compile-time reflection, sooner (D118's direction)

D118 fixed the direction (user derives through compile-time reflection on
`const fun`, no macros) but not the time. Many of the APIs below want a
derive: `cli.Args` (D167, decided), a binary layout (Bluetooth/USB/packets),
`sql` row mapping, property-test generators (I11), builders, `Default` with
custom rules. Building D118 once turns each of those from compiler work
into std work. **L; the multiplier for Part 3.**

#### I14 — Editor and tool experience

| Idea | What it gives |
|---|---|
| Inlay hints for inferred effects: `suspends`, `throws E` after a call | "no colour" is only good if you can still see colour on demand |
| `veles watch [run\|test]` | rebuilds on save (uses `fs.watch`, D161) |
| `veles eval '<expr>'` and a REPL | try a std function without a project; teaching |
| Quick fixes from every diagnostic that names a fix | B1 made messages name the fix; the LSP applies it |
| `veles fix --to v0.82` | runs the migrations a release made (D157–D166 change spellings) |
| `veles doc --serve` with search | the std reference offline |
| Coverage (`veles test --cover`) | LLVM's instrumentation is already in clang |
| `veles bench` + `bench "…" { }` blocks next to tests | the `veles-bench` skill's harness, available to users |

---

## Part 3 — High-level APIs to add, by purpose

### 3.0 What is already decided (not on this menu)

UDP/multicast/Unix sockets/`net.lookup`, `std/term`, `fs.watch`, `fs.map`,
`std/serial`, `std/bluetooth` (LE central), `std/hid`, `std/usb`,
`std/sys/{windows,linux}`, `veles bindgen`, `std/cli` — D159–D162, D167,
plan track P and C9.

Every module below follows D161's rules: a portable high-level surface,
`raw()` down to its `std/sys` handle, devices as `with` values, events as a
`Subscription`, every wait cancellable and honouring `withTimeout`, typed
errors with a `kind`, bounded reads, a **simulated backend** for tests, and
a **capability** where it touches the outside world.

Columns: **Size** S (days) / M (weeks) / L (a month or more) /
XL (a project). **Needs** = what must exist first (P-numbers are plan
track P). **Self-host** marks what the compiler port would use.

### 3.1 Hardware and devices

| # | Module | The high level | Under it (Windows / Linux) | Needs | Size |
|---|---|---|---|---|---|
| A1 | `std/gamepad` | `loop (e in pads.events()) when (e) { is Pressed(Button.A) => jump() … }`; rumble | XInput + GameInput / evdev | `hid` (P8) | M |
| A2 | `std/midi` | `with out = try midi.outputs().first()?.open(); out.noteOn(60, velocity: 100)`; inputs as a Subscription | WinMM / ALSA sequencer | P1 | M |
| A3 | `std/audio` | `with player = try audio.output(); player.play(try audio.decode(bytes))`; `with mic = try audio.input(rate: 48000)` → frames | WASAPI / PipeWire–ALSA | COM (P5) | L |
| A4 | `std/camera` | `with cam = try camera.open(); val frame = try cam.capture()` → an `Image` | Media Foundation / V4L2 | COM, `image` (A41) | L |
| A5 | `std/gpio` (+ `i2c`, `spi`, `pwm`) | `with led = try gpio.output(chip: 0, line: 17); led.set(true)`; `with bus = try i2c.open(1); try bus.read(0x48, register: 0, n: 2)` | — / gpiochip cdev, i2c-dev, spidev | P1 | M |
| A6 | `std/can` | `with bus = try can.open("can0"); loop (f in bus.frames()) …` | (vendor DLLs) / SocketCAN | P2 | S–M |
| A7 | `std/smartcard` | `with card = try smartcard.readers().first()?.connect(); try card.transmit(apdu)` — YubiKeys, ID cards, NFC readers | WinSCard / pcsc-lite | P1 | M |
| A8 | `std/power` | `power.onBattery()`, `power.level()`, `with power.keepAwake(reason: "export")`, a Subscription of sleep/resume | PowerSetRequest / logind over D-Bus | P5 | S–M |
| A9 | `std/display` | monitors, their size, scale and refresh rate; a Subscription of changes | DXGI/`EnumDisplayMonitors` / DRM or Wayland output info | P5 | S |
| A10 | `std/sensors` | thermal zones, fan speeds, accelerometers where present | WMI / hwmon, IIO | P1 | M |
| A11 | `std/printer` | `try printer.default().print(pdf)` | XPS/winspool / CUPS (IPP over HTTP) | — | M |

### 3.2 OS integration (desktop and system)

| # | Module | The high level | Under it | Needs | Size |
|---|---|---|---|---|---|
| A12 | `std/notify` | `try notify.show("Build finished", body: "12 tests passed", actions: ["Open"])` → which action was clicked | toast (WinRT) / `org.freedesktop.Notifications` | P5 | S |
| A13 | `std/clipboard` | `try clipboard.text()`, `try clipboard.setText(s)`, images | Win32 clipboard / Wayland data-control or X11 | P1 | S–M |
| A14 | `std/tray` | an icon with a menu; clicks as a Subscription | Shell_NotifyIcon / StatusNotifierItem (D-Bus) | P5 | M |
| A15 | `std/dialog` | `try dialog.openFile(filters: ["*.csv"])`, `dialog.confirm("Delete?")` | IFileDialog / xdg-desktop-portal | P5 | M |
| A16 | `std/keychain` | `try keychain.set("myapp", "token", Secret.of(t))`, `try keychain.get(…)` → `Secret<string>?` | Credential Manager / Secret Service (libsecret protocol over D-Bus) | P5 | S–M |
| A17 | `std/dirs` | `dirs.config("myapp")`, `dirs.cache(…)`, `dirs.data(…)`, `dirs.downloads()` — created on first use | Known Folders / XDG base dirs | P1 | S |
| A18 | `os.open(target)` | open a URL or file with the user's default app | ShellExecuteW / `xdg-open` portal | P1 | S |
| A19 | `std/service` | `service.run(name: "veles-agent", serve)` — the same program is a console app, a Windows service, or a systemd unit with `Type=notify` and socket activation; stop requests arrive as the shutdown signal tasks already honour | SCM / sd_notify, `LISTEN_FDS` | P1 | M |
| A20 | `std/sandbox` | `try sandbox.restrict(read: [dataDir], write: [cacheDir], net: false)` — the process gives up what it will never need; the package capability list (D138) applied at run time too | AppContainer / Job objects / Landlock + seccomp | P1 | M–L |
| A21 | `std/instance` | `with lock = try instance.single("myapp") ?: { try instance.sendToFirst(args); os.exit(0) }` — one running copy, later launches forward their arguments | named mutex + pipe / abstract Unix socket | P2 | S |
| A22 | `std/autostart` | `try autostart.enable(args: ["--tray"])` | Run key / `~/.config/autostart` | A17 | S |
| A23 | `std/process` (beyond `os.run`) | streaming child processes: `with child = try process.spawn("ffmpeg", args); loop (line in child.stderr.lines(max: 4096)) …`; pipes between children; a process group killed when the block ends | CreateProcessW + job / posix_spawn + pidfd | — | M |
| A24 | `std/ipc` | named pipes / Unix sockets as `io.Stream`, a typed request/response channel between two Veles processes (`ipc.serve<Request, Response>(name, handler)`, Codable on the wire) | named pipes / Unix sockets | P2 | M |
| A25 | `std/locale` + time zones | `locale.current()`, number/date formatting per locale, `Timestamp.in(zone("Europe/Prague"))` with the IANA database | ICU-free tables generated at compile time (`const fun`) | — | M |

### 3.3 Networking and protocols

| # | Module | The high level | Needs | Size |
|---|---|---|---|---|
| A26 | `std/ws` (WebSocket) | server: `router.ws("/chat", sock => loop (msg in sock.messages()) sock.send(msg))`; client: `with sock = try ws.connect(url)` | `http` | M |
| A27 | `http` SSE | `router.sse("/events", out => loop (e in news) out.send(e))` | `http` | S |
| A28 | `std/discover` (mDNS/DNS-SD) | `try discover.announce("_myapp._tcp", port: 8080)`; `loop (svc in discover.browse("_ipp._tcp")) …` — printers, peers, IoT devices with no config | UDP multicast (P2) | M |
| A29 | `http` with automatic TLS (ACME) | `http.serve(listener, app, tls: acme(domains: ["example.org"], cache: dirs.data("app")))` — a certificate obtained and renewed by itself | `tls` server, `crypto` ECDSA | M |
| A30 | HTTP/2 (then HTTP/3/QUIC) | invisible: `http.serve` negotiates h2; the client too | `tls` ALPN; QUIC needs UDP | L / XL |
| A31 | `std/mqtt` | `with broker = try mqtt.connect(url); loop (m in try broker.subscribe("sensors/#")) …` — IoT | `net`, `tls` | M |
| A32 | `std/grpc` + protobuf derive | `implement Protobuf` on a struct; a service is a trait; `grpc.serve(impl)` | HTTP/2 (A30), I13 for the derive | L |
| A33 | `std/ssh` | `with s = try ssh.connect(host, auth: ssh.agent()); val out = try s.run("uptime")`; SFTP as `fs`-shaped calls | `crypto` (ed25519, X25519, AES-GCM/ChaCha) | L |
| A34 | `std/smtp` / mail | `try mail.send(Message(to: [...], subject, text, attachments), via: smtp(url))` | `tls`, `codec` (MIME) | S–M |
| A35 | `std/dns` (resolver, records) | `try dns.query("example.org", Record.TXT)`; DoH | UDP (P2) | S–M |
| A36 | `std/rpc` (Veles-to-Veles) | `rpc.serve<Api>(impl)` / `val api = try rpc.connect<Api>(url)`: a trait as the contract, Codable over HTTP or IPC | I13 | M |
| A37 | `std/p2p` (NAT traversal, QUIC peers) | `with peer = try p2p.dial(ticket)` | QUIC, STUN | XL |

### 3.4 Data, text and storage

| # | Module | The high level | Needs | Size |
|---|---|---|---|---|
| A38 | `std/regex` + `regex"…"` | `val date = regex"(?<y>\d{4})-(?<m>\d\d)"` checked at compile time (a bad pattern is a compile error; group names become typed fields: `m.y`); linear time (RE2-style, no catastrophic backtracking — untrusted input safe) | D129 tags, I2 raw strings | M |
| A39 | `std/sqlite` (vendored) | the `std/db` API unchanged: `with db = try sqlite.open(dirs.data("app") + "/app.db")`; `db.query<User>(sql"…")`; migrations: `try db.migrate(dir: "migrations")` | `[native]` vendoring | M |
| A40 | `std/csv` | `loop (row in try csv.read<Sale>(path, max: …)) …` via `Codable`; writing too | `codec` | S |
| A41 | `std/image` | `val img = try image.decode(bytes)` (PNG, JPEG, WebP, GIF), `img.resize(width: 256)`, `try image.encode(img, Format.Png)`; pixels as an `Array`-backed buffer | `compress` (inflate exists) | M–L |
| A42 | `std/archive` | `with z = try zip.open(path); loop (e in z.entries()) …`; tar(.gz) read/write | `compress` | S–M |
| A43 | Codable formats | TOML, YAML (safe subset), MessagePack, CBOR, Protobuf on the existing `codec` events | `codec` | S each |
| A44 | `std/decimal` + `std/bigint` | `Decimal.parse("19.99")` exact money arithmetic with operators (D71); `BigInt` for crypto and IDs | — | M |
| A45 | `std/unicode` | normalization (NFC/NFKC), grapheme clusters (`s.graphemes()`), width for terminals, case folding, collation | tables via `const fun` | M |
| A46 | `std/i18n` | `tr"You have ${n} new messages"` resolved from a catalogue with plural rules (ICU MessageFormat), the catalogue checked at compile time for missing keys | D129 tags, A25 | M |
| A47 | `std/kv` (embedded store) | `with store = try kv.open(path); try store.put(key, value)`; transactions; ordered scans | `fs.map`, I8 | L |
| A48 | `std/template` / `html"…"` | `html"<p>${user.name}</p>"` escapes by context; layouts and partials | D129 | S–M |
| A49 | `std/markdown` | CommonMark → HTML (`veles doc` uses one already in Go) | — | M |
| A50 | `std/diff` | `diff.lines(a, b)` → hunks; unified output (test snapshots, tools) | — | S |
| A51 | `embed(…)` at compile time | `const ASSETS = embed("public/")` — files inside the binary, served by `http.files(ASSETS)`; one-file deployment | D113 const | S |

### 3.5 UI, graphics and media

| # | Module | The high level | Under it | Needs | Size |
|---|---|---|---|---|---|
| A52 | `std/tui` (on `std/term`) | widgets — list, table, input, progress, spinner, layout — redrawn from state: `tui.run(state, view, update)`; prompts: `try tui.select("Pick one", items)`, `try tui.confirm(…)` | term (P3) | M |
| A53 | `std/webview` app shell | `with app = try webview.open(title: "Notes", html: embed("ui/"))`; JS calls Veles functions by name (`app.bind("save", save)`, JSON over the bridge); the Veles side is ordinary async code | WebView2 / WebKitGTK | COM (P5) | M |
| A54 | `std/window` + `std/gpu` | `with win = try window.open(…)`, events as a Subscription; a WebGPU-shaped API (`gpu.device()`, buffers, pipelines, WGSL shaders) | Win32 + D3D12 / Wayland+X11 + Vulkan | XL |
| A55 | `std/canvas` (2D) | `canvas.drawText`, paths, images → PNG/SVG or a window | A41; A54 for on-screen | L |
| A56 | `std/ui` (native widgets, declarative) | a retained, declarative UI (`Column { Text(…); Button("Save", onClick) }`) | XL (needs a language story for builders) | XL |
| A57 | `std/video` | decode/encode via the OS (H.264/AV1) | Media Foundation / VA-API | L |

My read: **A53 (webview)** gives desktop apps for the least work — the UI
is HTML, the backend is Veles' strongest part (async, typed errors, JSON).
A54/A56 are projects of their own and pull the language toward builder
syntax; a later decision.

### 3.6 Compute and performance

| # | Module | The high level | Needs | Size |
|---|---|---|---|---|
| A58 | `Simd<T, N>` | `val v = Simd<f32, 8>.load(xs, at: i); acc += v * k` — portable vectors over LLVM's `<8 x float>`; `reduceSum()`, masks, shuffles; falls back to scalar where the target lacks them | codegen, D121 | M |
| A59 | `xs.parallelMap(f)` / `parallelSort` | CPU parallelism over `Executor.pool` with chunking chosen by size: the data-parallel twin of `mapConcurrent` | F8 | S |
| A60 | `std/math` | `Vec2/3/4`, `Mat3/4`, quaternions with operators (D71); `stats` (mean, median, percentile, stddev, histogram) | — | S–M |
| A61 | `std/random` additions | distributions (normal, exponential), `shuffle`, `sample(k)`, seeded and reproducible for tests | — | S |
| A62 | `std/hash` (non-crypto) | xxHash3 / wyhash for fast hashing of data; consistent hashing (`Ring`) | — | S |
| A63 | `compress` additions | zstd and brotli (decode first: HTTP `br`, packages) | — | M each |
| A64 | `std/profile` | `with profile.cpu(to: "cpu.pprof")`, `profile.heap()`, in-process sampling; `http.debug()` mounts `/debug/tasks`, `/debug/gc`, `/debug/pprof` | runtime counters (E5) | M |
| A65 | `std/ml` (small) | tensors on `Simd`, ONNX inference via the OS (DirectML / ONNX Runtime) | A58 | L |

### 3.7 Security and identity

| # | Module | The high level | Needs | Size |
|---|---|---|---|---|
| A66 | `crypto` signatures and key exchange | `ed25519`, `ecdsa` (P-256), `x25519`, AES-GCM, ChaCha20-Poly1305 — `Secret` keys, constant-time compare | — | M |
| A67 | `std/x509` | parse/verify certificates, make a CSR or a self-signed cert for dev (`x509.selfSigned("localhost")`) | A66 | M |
| A68 | `std/oauth` | `val token = try oauth.deviceFlow(provider: github, scopes: [...])` for CLIs; authorization-code + PKCE for servers; OIDC id-token checks | `http`, `jwt` | M |
| A69 | `std/webauthn` (passkeys) | `router.post("/login/finish", req => try passkeys.finish(req))` — registration/assertion verification | A66, `codec` CBOR (A43) | M |
| A70 | `std/totp` | `totp.verify(secret, code, at: time.now())`, provisioning URI | `crypto` HMAC (exists) | S |
| A71 | `std/age` | file encryption to recipients (`age.encrypt(bytes, to: [recipient])`) | A66 | S–M |
| A72 | `http` hardening helpers | `http.rateLimit(per: Duration.minutes(1), max: 60, key: req => req.ip)`, CSRF, security headers in one `http.secure()` | `http` | S |

### 3.8 Services and operations

| # | Module | The high level | Needs | Size |
|---|---|---|---|---|
| A73 | `std/jobs` | `jobs.every(Duration.minutes(5), refresh)`, `jobs.cron("0 3 * * *", backup)`; a durable queue on `std/db`/`std/sqlite` with retries | `time`, A39 | M |
| A74 | Resilience | `breaker.call(f)` (circuit breaker), `bulkhead(n)`, `hedge(after: d, f)`; `retry` exists | — | S |
| A75 | Feature flags | `flags.enabled("new-checkout", user: id)` from config, percentages, reloaded on change | `config`, `fs.watch` | S |
| A76 | `std/update` (self-update) | `if (val next = try update.check(feed)) try next.apply()` — signed with the registry's ed25519 statements | `http`, A66 | M |
| A77 | Build info | `build.version`, `build.commit`, `build.time` constants, printed by `--version` (via `cli`) | D160 | S |
| A78 | Crash reports | a panic writes a report (trace, build info, task dump) to `dirs.cache`; `crash.upload(url)` on next start | B4, A17 | S–M |

### 3.9 Testing and tools as libraries

| # | Module | The high level | Size |
|---|---|---|---|
| A79 | `net.Simulated()` / `http.test` | an in-memory network: `val client = http.test(app)` sends requests to a handler with no socket; latency and loss injected | S–M |
| A80 | `fs.memory()` | an in-memory file system behind the `fs` API for tests | M |
| A81 | `time.Simulated()` | virtual clock (I11) | M |
| A82 | Fakes from traits | `val repo = fake<Repo>()` — every method records calls and returns defaults; `repo.on("find", …)` | needs I13 |
| A83 | `std/lsp` as a library | the protocol types and a server loop, so tools written in Veles can be language servers | M |

### 3.10 Foundations worth building once

Most rows above reduce to a handful of shared pieces. Building these first
makes each module S or M:

| Foundation | Unlocks |
|---|---|
| COM/WinRT from Veles (P5) | A3, A4, A8, A9, A12, A14, A15, A53, Bluetooth on Windows |
| D-Bus client (P5) | A8, A12, A14, A15, A16, Bluetooth on Linux, NetworkManager, systemd |
| A **binary layout derive** (`implement Binary`, `@be`/`@le`, `@bits(n)`, length-prefixed fields) — the binary twin of `Codable`, mentioned as later in D159 | Bluetooth payloads, USB/HID descriptors, CAN frames, file formats (PNG chunks, zip headers), DNS/mDNS packets, the compiler's object/debug-info writers |
| Compile-time reflection (I13) | `cli`, Protobuf, gRPC, RPC, fakes, property tests, SQL mapping |
| Simulated backends + virtual time (I11) | testable timeouts, devices and networks in CI and WSL |
| Raw/multi-line strings + `regex"…"`/`html"…"` tags (I2, A38, A48) | every text-heavy module |
| `Simd` (A58) | image, audio, crypto, hashing, ML, the compiler's lexer |

### 3.11 A shortlist, if you want my order

By value for effort, keeping self-hosting in view:

1. **Defects F1–F4** (decision-free) and **I6**.
2. **I2** raw/multi-line strings, **I3** `in`, **I5** `async { }`, **I7**
   collection names — small, felt in every program.
3. **I1 generators** — the largest expressiveness gain left; the compiler
   port and every lazy API want it.
4. **A38 regex**, **A51 embed**, **A39 sqlite**, **A40 csv**, **A17 dirs**,
   **A77 build info** — what real tools reach for first.
5. **The binary derive** and **I11 virtual time** — before the device
   modules (P6–P9), so they are built on them.
6. **A26 WebSocket + A27 SSE**, **A12 notify**, **A16 keychain**,
   **A19 service**, **A20 sandbox** (it extends D138's capabilities to run
   time — a feature few languages have).
7. **A53 webview** — desktop apps.
8. Then by demand: A58 `Simd`, A66 crypto, A29 ACME, A52 TUI, A3 audio,
   A5 GPIO, A28 discovery.

Nothing here is decided. A picked item goes through `veles-decide`
(names, signatures, edge cases) before it is built, and lands in the plan
as a track-P or track-C row.
