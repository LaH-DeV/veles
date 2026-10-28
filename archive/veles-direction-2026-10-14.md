# Veles — direction review, 2026-10-14

Asked by the user: are the spec, plan, checklist and the other files current;
what the next steps to self-hosting are; what idiomatic Veles looks like and
where it could be better; and low-level platform APIs (Bluetooth and the
like) with a high-level developer experience. macOS is out of scope for now.

Everything under "today" below was compiled and run with the compiler as it
stands (scratchpad probes `gallery`, `gallery2`, `idiom1`–`idiom4`). Code
under "proposed" does **not** compile; it is what a decision would make
possible. The decisions are Q32–Q43 in `veles-checklist.md` §9.

---

## 0. Summary

1. **The documents were mostly current, with drift in status markers** — the
   plan still listed done work as open (C5/C6, B14, tracks C/E), the spec had
   twelve examples in the removed `fun f() = expr` form, the README described
   v0.42, and the cheat sheet showed a module-level `var` (an error since
   D66). All fixed in this change (§1). The real next build step was hidden:
   it is **B17, D142 visibility** — decided 2026-10-08, never built.
2. **Self-hosting:** the lexer and parser in Veles agree with Go byte for byte
   (G1–G4). What stands between that and a compiler in Veles is about 50 000
   lines of `sema`/`codegen`/`driver`, and seven concrete gaps: no debug info,
   no oracle dumps for `sema`, no interface files, a front end 2.4× slower
   than Go's, no per-target code, no argument parsing, and the ceremony
   counted in §3 (2 235 `this.` in 5 900 lines of parser). New plan track
   S6–S12 (§2).
3. **Idiomatic Veles is already good** at errors, nullables, sealed data,
   concurrency and C interop (§3). Seven places cost readers the most; each
   is a decision with options (§4). One is a **footgun found by this
   review**: `"${f()}"` where `f` throws compiles and prints
   `Err(error: …)` — the error vanishes into a string.
4. **Platform APIs:** Veles has no UDP, no Unix sockets, no file watching, no
   terminal control, no serial/HID/USB/Bluetooth, and no way to write
   per-OS code in Veles (all of it lives in the C runtime today). §5 proposes
   a layering — OS bindings in Veles under `std/sys/*`, portable modules on
   top, a simulated backend for every device module so tests run in WSL and
   CI — and a Bluetooth LE API written out in full.

---

## 1. Are the documents current?

| File | Found | Done in this change |
|---|---|---|
| `veles-plan.md` | Build order step 11 (C5/C6) not struck though done 2026-10-04; B14 row said "part 4 open"; track C items 2–8 and track E items 1, 2, 7 carried no status; track D described the 2026-09-30 batch order; nothing said where to continue | statuses added; track D says all batches are decided and points at Q32–Q43; a "where to continue" paragraph (B17 → F19 migration → S6–S8 → S9–S12); E4/E5 status; new rows **S6–S12**; new **track P** (platform APIs, waits for Q39–Q43) |
| `veles-spec.md` | 12 examples and 4 prose spans used `fun f() = expr` (removed by D140); §4b read "`implement`, not `implement`" (a mechanical rename hit the old word) | all migrated to `=>`; §4b says "`implement`, not `impl`" |
| `veles-checklist.md` | OTel (built 2026-10-05) unticked "not built"; `substring` copy check unticked (verified for the self-host plan); `std/fs` marked open (done in C5/C6); GC said "stop-the-world today" (F18); §8b said four decisions still had to be made; §11 listed `std/db` as deferred | ticked / rewritten with dates; Q32–Q43 added to §9; §11 entries for the gaps this review found |
| `README.md` | "v0.42", "self-hosting is future work", "fifteen chapters" (25), "32 programs" (63), "M7 registry not implemented" (E7 done), "timers share one runtime lock" (F9), "by-value C ABI not implemented" (A8) | rewritten to the current state; the known gaps now name macOS, Linux GC marking and debug info |
| `veles-selfhost-frontend-plan.md` | no status line; P2–P5 sections read as open | status block at the top; P2–P5 marked done |
| `docs/…/cheatsheet.md` | `var counter = 0` at module level (an error since D66); integer row read `i8 i16 i64 i64` | `val counter = Atomic(value: 0)` with the reason; `i32` restored |
| `examples/httpd/main.vs` | a comment explained why a method "needs no `mut`" (removed in v0.30) | comment rewritten (no behaviour or expected output changed) |
| `notes_to_change.txt` | #14 says its work was built as part of B15 (2026-10-02) | left for the user: it is your text; by the file's own rule it can leave |

Still true and not changed: `veles-spec.md` §5's "module interface files" is
a plan, not a fact (now S12); §7 keeps D41 (literals wired to fixed types) as
undecided; `archive/` is untouched. I did not re-verify every line of
checklist §11 (≈390 lines); the entries above are the ones a search for
done-but-open wording found.

---

## 2. Self-hosting: where we are and what is needed

### Where we are

| Piece | Go lines | In Veles | Oracle |
|---|---:|---|---|
| `source`, `lexer`, `ast`, `parser` | 7 200 | **done** (`selfhost/`, 5 900 lines + tables) | `veles parse`/`tokens` byte for byte, G1–G4 green |
| `format` | 2 400 | — | `veles fmt --stdout`: free, exact |
| `types` + `sema` | 36 400 | — | none yet (needs dumps, S10) |
| `codegen/llvm` | 9 000 | — | the codegen goldens + the IR of every example |
| `driver` | 2 200 | — | the CLI's behaviour |
| `lsp`, `fetch`, `registry` | 6 500 | — | their tests |
| `runtime/c` | 15 000 | stays C | — |

Measured: the Veles parser reads 3 MB of std in ~1.3 s, Go's in ~0.55 s.

### The steps (plan track S, decision-free unless marked)

| # | Step | Why it is next |
|---|---|---|
| — | **B17: D142 visibility** | decided, unbuilt, touches every declaration in std/selfhost/examples — build it before more Veles code is written in the old shape |
| S6 | Finish E4: `Layout`, a dump with spans, byte input | G2 does not compare spans today; non-UTF-8 files are outside the gate |
| S7 | Front-end speed as a finding | 2.4× Go; fix the compiler/runtime, not the port |
| S8 | Debug info (DWARF / CodeView line tables + locals) | nobody writes 50 000 lines without a debugger; today a crash gives a panic trace only |
| S9 | The formatter in Veles (gate G5) | the next slice whose oracle is free and exact |
| S10 | `veles check --dump=names\|types\|hir` | the oracle a Veles `sema` is gated against, phase by phase |
| S11 | `veles-selfhost-sema-plan.md` | the cut and the gates of the big port, measured from S6–S9 |
| S12 | Module interface files | a 50 000-line program is rechecked whole on every build today |

After S12 the shape of the bootstrap, for when it is time (not decided now):
**stage 1** — the Go compiler builds `velesc` from `selfhost/`; **stage 2** —
`velesc` builds itself; **stage 3** — stage 2 builds itself, and stage 2 and
stage 3 emit byte-identical IR. Until stage 3 holds and the whole test suite
passes with `velesc`, the Go compiler stays the reference.

### What the language must offer first (decisions)

Each was checked against the parser port, the largest Veles program there is:

| Need | Evidence | Decision |
|---|---|---|
| Members without `this.` | 2 235 `this.` in 5 900 lines of `selfhost/` (one line in three); 2 495 in std | **Q33** |
| Named results / labelled tuples | the parser returns pairs (`(tokens, diags)`), maps give `(K, V)`; 40 `.0`/`.1` reads in the tree | **Q35** |
| Updating an immutable value | AST rewrites (`left = Binary(left: &left, …)`) rebuild every field | **Q36** |
| Per-target code in Veles | the driver picks clang flags, paths and linkers per OS; today only C can branch on the OS | **Q39** |
| Argument parsing | three hand-written `Options.parse` loops in examples; the driver has ~40 flags | **Q38** |
| Mapping a file into memory | a compiler reads every source of a package | track P (`fs.map`) |

---

## 3. Idiomatic Veles today

### A small tool, high level (compiles and runs today)

A log analyser — errors as types, list patterns with let-else, `?!`, a
sealed family with an exhaustive `when`, a derived wire form, a
compile-time table, and an awaited task without a `scope` (D156):

```veles
use fs
use io { println }
use json
use os

error BadLine {
  line: i64
  text: string
  fun message(): string => "line ${this.line}: cannot read '${this.text}'"
}

enum Level : u8 {
  Debug
  Info
  Warn
  Error
}

struct Entry {
  at:      i64
  level:   Level
  service: string
  text:    string
  implement Codable
}

sealed trait Finding {
  fun describe(): string => when (this) {
    is Spike(service, count) => "$service logged $count errors"
    is Silent(service)       => "$service said nothing"
  }
}

struct Spike : Finding {
  service: string
  count:   i64
}

struct Silent : Finding {
  service: string
}

const LEVEL_NAMES: List<string> = ["DEBUG", "INFO", "WARN", "ERROR"]

const fun levelOf(word: string): Level? {
  loop (i in LEVEL_NAMES.indices()) {
    if (LEVEL_NAMES.at(i) == word) return Level.fromValue(i.toU8() ?: return null)
  }
  null
}

fun parseLine(number: i64, text: string): Entry throws BadLine {
  val bad = BadLine(line: number, text)
  val [stamp, word, service, ..rest] = text.split(" ") else throw bad
  val at = try stamp.toInt() ?! bad
  val level = try levelOf(word) ?! bad
  Entry(at, level, service, text: rest.join(" "))
}

fun parseAll(text: string): (List<Entry>, List<BadLine>) {
  val entries: MutableList<Entry> = []
  val problems: MutableList<BadLine> = []
  loop ((i, line) in text.lines().enumerate()) {
    if (line.trim().isEmpty()) continue
    when (val r = parseLine(i + 1, line)) {
      is Ok  => entries.push(r)
      is Err => problems.push(r)
    }
  }
  (entries.toList(), problems.toList())
}

fun summarize(path: string): string throws IoError {
  val text = try fs.readFile(path)
  val parsed = async parseAll(text)          // runs alongside; must be awaited on every path
  val size = text.len()
  val (entries, problems) = await parsed
  loop (p in problems) println("skipped ${p.message()}")
  val worst = entries
    .filter(e => e.level >= Level.Warn)
    .sortedBy(e => (-e.level.value.toI64(), e.at))
    .take(3)
  "${entries.len()} entries, $size bytes, worst first: ${worst.map(e => e.text).join(", ")}"
}

fun main() {
  val [path] = os.args() else {
    println("usage: logstat <file>")
    os.exit(2)
  }
  val summary = summarize(path) catch (e) {
    println("logstat: ${e.message()}")
    os.exit(1)
  }
  println(summary)
}
```

What reads well: no exceptions or `Option` wrappers in sight; failure paths
are one token (`try`, `?!`, `else`, `catch`); the `when` over `Finding` is
checked for exhaustiveness; `Entry` gets JSON for one line; `levelOf` runs at
compile time when its argument is a constant; the concurrency is two words.

### The low level (compiles and runs today)

```veles
use ffi
use io { println }

extern "C" {
  fun strlen(s: *raw u8): i64
}

@packed
extern struct Header {
  magic:   u32
  version: u16
  flags:   u8
}

/// FNV-1a over a fixed block: no allocation, `Array` is a value.
fun checksum<const N: i64>(block: Array<u8, N>): u32 {
  var h: u32 = 2166136261
  loop (b in block) h = (h ^ b.toU32()) *% 16777619
  h
}

fun cLength(text: string): i64 throws ffi.NulByte {
  with c = try ffi.CString.of(text)          // freed when the block ends
  // SAFETY: c.ptr() is a live NUL-terminated copy until the block ends
  unsafe { strlen(c.ptr()) }
}

fun main() throws ffi.NulByte | ThreadError {
  val hits = Atomic(value: 0)
  with cpu = try Executor.pool(threads: 2, name: "cpu")   // real OS threads, named
  scope(on: cpu) {
    loop (i in 0..<8) async hits.add(i, order: MemoryOrder.Relaxed)
  }
  println("hits ${hits.load()}, strlen ${try cLength("veles")}")
}
```

Both levels are one language: the C call is fenced by `unsafe` and a
`SAFETY:` comment, the C memory is closed by `with`, the thread pool is a
value that closes, and the memory order is a named argument with a safe
default.

### What the compiler's own code looks like (`selfhost/parser/parser.vs`)

```veles
fun expect(kind: Kind): bool {
  if (this.accept(kind)) return true
  this.errorExpected("'${kindText(kind)}'")
  false
}

fun expectIdent(): ast.Ident? {
  if (!this.at(Kind.Ident)) {
    this.errorExpected("identifier")
    return null
  }
  val t = this.next()
  ast.Ident(name: t.text, pos: t.span)
}

fun cur(): Token => this.toks.at(this.pos) ?: panic("parser: the cursor is always on a token")
```

Good shape (bool / `T?` results instead of Go's `(T, bool)`), but every
other token is `this.`, and an index that is always in range still needs a
`?: panic(…)` with a sentence (84 such sites in the tree).

---

## 4. Where it could be better

Each item: the problem as it is in the tree today, the evidence, and the
proposal. Options and edge cases are in the checklist §9 entry; the
recommendation is first.

### Q32 — Interpolating a `Result` swallows the error (footgun, found here)

```veles
fun parse(s: string): i64 throws IoError => s.toInt() ?: throw IoError(path: s, code: 0, detail: "nan")
println("got ${parse("x")}")      // compiles; prints: got Err(error: IoError(path: x, …))
```

The same happened writing the gallery: `"${json.encode(worst)}"` printed
`Ok(value: [...])`. An unused `Result` is an error; a `Result` turned into
text is as unused as one dropped. **Proposed:** interpolating a `Result`
(and a `Task`) is an error with two fixes, `${try f()}` and `${f() ?? x}`;
`"$r"` stays possible through `r.toString()` written out.

### Q33 — Members without `this.`

```veles
// today                                         // proposed
fun next(): Token {                              fun next(): Token {
  val t = this.cur()                               val t = cur()
  if (this.pos < this.toks.len() - 1)              if (pos < toks.len() - 1) pos += 1
    this.pos += 1                                  t
  t                                              }
}
```

2 235 `this.` in the parser port, 2 495 in std. Kotlin, Swift, Java, C#
and C++ resolve a bare member name to the receiver; Rust, Python and TS do
not. **Proposed:** a name not found among locals and parameters resolves to
a member of `this` before module scope; a local or parameter shadows a
member (`init(x) { this.x = x }` keeps working), and shadowing a member by a
local is a warning only where the member is also used in the same body.
`this.` stays valid; `veles fmt` keeps what was written; hover says "field
of `this`". Lambdas see members as captures of `this` (D35's Sendable rule
applies as today).

### Q34 — One-field types constructed positionally

```veles
throw UsageError(message: "unknown option '$arg'")   // today, 78 times in the tree
throw UsageError("unknown option '$arg'")            // proposed
val notes = Mutex(Notes())                           // proposed (today Mutex(value: Notes()))
```

D28 made construction by name so a reordered field cannot change a call's
meaning. A type with **one** field (counting `init` parameters) has no order
to change, and adding a second field turns every positional call into a
compile error with the fix — the safety is kept. Evidence: 78
`throw X(message: …)`; `Mutex`, `Atomic`, `Secret`-style wrappers.

### Q35 — Labelled tuples (written out in full at the user's request)

The user, 2026-10-14: "not sure how would it look like in all of the use
cases and pros and cons". Below: what tuples are today (probed), every place
a label would show, the options, the rules each needs, and the trade-offs.

**Today** (probe `tuples`): `(A, B)` types, `.0`/`.1`, positional
destructuring in `val`, loop heads and lambda parameters (`(name, n) =>`,
D37's tupling), structural `==`/hash/order, printed as `(apple, 3)`; tuples
are **not** `Codable` (deriving on a struct with a tuple field fails, and the
error points at `implement Codable` without naming the field — a diagnostic
to fix either way, checklist §11). In the tree: 40 `.0`/`.1` reads.

**The use cases, today and with labels (option A):**

| Use | Today | Option A |
|---|---|---|
| A map entry passed on | `entries().filter(e => e.1 > 1).sortedBy(e => -e.1)` | `entries().filter(e => e.value > 1).sortedBy(e => -e.value)` |
| A map loop | `loop ((word, n) in counts)` | unchanged (destructuring is positional and already reads well) |
| Two results from one function | `fun divide(a, b): (i64, i64)` — the reader must know which is which | `fun divide(a, b): (quotient: i64, remainder: i64)`; `val r = divide(7, 2); r.remainder` or `val (q, r) = divide(7, 2)` |
| A parser step that yields a value and a position | `(ast.Expr, i64)`, read as `.0`/`.1` | `(expr: ast.Expr, next: i64)` |
| `enumerate()` | `(i64, T)`; `e.0` | `(index: i64, value: T)` |
| `splitOnce("=")` | `(string, string)?`; `.0`/`.1` | `(before: string, after: string)?` |
| `zip` | `(A, B)` | unchanged (no meaningful names: `first`/`second` say nothing) |
| A list of small records for sorting or printing | declare a struct, or `List<(string, i64)>` and `.1` | `List<(name: string, age: i64)>`, `sortedBy(p => p.age)` |
| `gather` results, `awaitAll` | `val (a, b) = …` | unchanged (destructured at once) |

**Options.**

- **A — labels as part of the tuple type (Swift's model; recommended).**
  `(key: K, value: V)` in types and literals; `.key` and `.0` both read it;
  - *Conversion:* an unlabelled tuple converts to a labelled one and back
    freely (`val r: (quotient: i64, remainder: i64) = (3, 1)`); between two
    **different** label sets it is an error — `(x: 1, y: 2)` where
    `(y: i64, x: i64)` is expected would otherwise silently swap meanings
    (Swift refuses it; C# only warns).
  - *Identity:* labels do not change layout, `==`, hashing, ordering or the
    generic instance (`List<(key: K, value: V)>` and `List<(K, V)>` share
    code); hover and errors print the labels.
  - *Destructuring:* positional as today (`val (k, v) = e`); no by-label
    destructuring (it would be a second pattern syntax).
  - *Text:* `"$e"` prints `(key: apple, value: 3)`.
  - *Wire:* tuples stay non-`Codable` (a wire shape deserves a struct and
    its `@key`s) — unchanged by labels.
  - *std:* `entries()` → `(key, value)`, `enumerate()` → `(index, value)`,
    `splitOnce` → `(before, after)`; existing `.0`/`.1` keep compiling, so no
    migration is forced.
  - *Cost:* parser and formatter (labels in tuple types and literals), the
    checker (label-aware assignability), hover; small.
- **B — a struct per use, no language change.** `Map.entries()` returns
  `MapEntry<K, V> { key, value }`, `enumerate()` an `Indexed<T> { index,
  value }`, `splitOnce` a `Split { before, after }`; user code declares a
  struct for every multi-value return. Struct patterns destructure them
  (`is MapEntry(k, v)`), but `val (k, v) = entry` and the `(k, v) =>`
  lambda form would no longer apply to entries unless structs gain tuple
  destructuring (a separate rule).
- **C — leave it.** Positional `.0`/`.1`; a struct where names matter.
- *(Considered, not offered: anonymous records `{ key: K, value: V }` as in
  TypeScript — a whole new kind of type with its own identity, equality and
  wire rules; much larger than the problem.)*

**Pros and cons.**

| | A: labelled tuples | B: structs per use | C: leave |
|---|---|---|---|
| Reading `e.value` instead of `e.1` | yes | yes (std) / per struct (user code) | no |
| Multi-value returns documented in the signature | yes, at no cost | one declaration per function | no |
| Positional destructuring and `(k, v) =>` keep working | yes | not for the new structs without another rule | yes |
| Swapped-meaning bug (`(x, y)` into `(y, x)`) caught | yes (label mismatch is an error) | yes (different types) | no |
| Two ways to model a record (tuple vs struct) | yes — guideline needed: a labelled tuple for a transient grouping (a return, an entry), a struct for anything with behaviour, invariants, docs or a wire form | no | no |
| New language surface | small (types, literals, one conversion rule) | none | none |
| Migration | none forced | `entries()` users change `.0`/`.1` | none |
| Elsewhere | Swift (same rules), C# (names erased, mismatch a warning), TS (labels as documentation only, index access) | Kotlin (`Pair`/data classes), Rust (structs), Python (`NamedTuple`) | Go (named results, not first-class) |
| Compiler in Veles | many two-value helpers (`(expr, next)`, `(type, effects)`) read without structs | a struct per helper result | `.0`/`.1` throughout |

**Recommended: A**, because it removes the `.0`/`.1` reading cost and
documents multi-value returns without adding a nominal type per use, and
its one new rule (different labels do not convert) catches a bug B and C
cannot (or B only by declaring a type for each).

### Q36 — A copy with some fields changed

```veles
val moved = Point(x: 5, y: q.y, label: q.label, …)   // today: every field again
val moved = q.copy(x: 5)                              // proposed (Kotlin's data-class copy)
```

Bare fields are never assigned (D22), so changing one means rebuilding the
value — the compiler port does this to AST and type nodes constantly.
`copy` is generated for every struct, takes each field as an optional named
argument, runs `init`, and may only name fields visible at the call (a
`private` field cannot be set from outside — the same rule as construction).

### Q37 — `indexOf` and friends answer `-1`

```veles
val at = this.items.indexOfFirst(x => x.id == id)   // today
if (at >= 0) this.items.removeAt(at)
at >= 0

val at = items.indexOfFirst(x => x.id == id) ?: return false   // proposed: i64?
items.removeAt(at)
true
```

A `-1` used as an index is a bug no checker sees; `i64?` makes the absent
case a compile-time fact, as `at()` and `find()` already are. Affects
`indexOf`, `indexOfFirst`, `lastIndexOf`, `string.indexOf`, `binarySearch*`
(not `lowerBound`/`upperBound`/`partitionPoint`, which always answer).

### Q38 — `std/cli`: arguments by derive

```veles
// today: a 30-line loop in every tool (dedup, httpd, todo) and soon the compiler's driver

/// Find duplicate files under a directory.
struct Options {
  /// The directory to scan.
  @positional
  dir:     string
  /// How many files to hash at once.
  workers: i64 = 4
  @name("min-size")
  minSize: i64 = 1
  @short("v")
  verbose: bool = false
  implement cli.Args
}

fun main() throws cli.Error {
  val opts = try cli.parse<Options>(os.args())   // --help, usage errors with exit code 2, typos suggested
  …
}
```

Same machinery as `config.load<T>` (D125) and the derive (D58): flags from
field names in kebab case, doc comments as help, defaults shown, enums as
choices, `List<T>` repeatable, subcommands as a sealed trait. The driver of
a self-hosted compiler is its first large user.

### Also proposed, decision-free (plan B18)

- `veles fmt` keeps a one-expression `unsafe { strlen(p) }` on one line (73
  three-line `unsafe` blocks in std hold one call each).
- The checker proves more indexes (`this.toks.at(this.pos)` when `pos` is
  kept in range by every assignment is beyond D62's facts today); measure
  the 84 `at(...) ?: panic` sites and extend the facts where a pattern
  repeats.

Not re-proposed, because they were decided against: `it` in lambdas (spec
D32 "no `it` shorthand"), `!!`, a panic `catch`, extension functions.

---

## 5. Low-level platform APIs with a high-level developer experience

### What exists, what is missing

Exists: `extern "C"` with structs by value, C layout, unions, variadics,
callbacks into Veles, foreign threads registered on their first callback, a
blocking call handing its queue on, `ffi.CString`/`Handle`, `[native]`
linking, capabilities per package (`unsafe`, `extern`, `native`, `net`, `fs`,
`os`, `ffi`) with a policy that can refuse them.

Missing: UDP and multicast, Unix-domain sockets, DNS lookup as an API, file
watching, memory-mapped files, terminal control (raw mode, size, keys,
colour), serial ports, HID, USB, Bluetooth — and the means to write any of
them **in Veles**: there is no per-target code (Q39), so every OS
difference lives in `runtime/c` today.

### Layering (Q40)

```
your program            bluetooth.adapter().scan(…)       serial.open("COM3", baud: 115200)
                                  │                                 │
portable std            std/bluetooth  std/serial  std/hid  std/usb  std/term  fs.watch  net.udp/unix
(one API, typed errors,           │
 a simulated backend)   ──────────┼──────────────────────────────────────────────
per-OS bindings, Veles  std/sys/windows (Win32, COM/WinRT)     std/sys/linux (syscalls, ioctl, netlink, D-Bus)
                                  │
                        extern "C" — what exists today
```

- **`std/sys/<os>`**: thin, typed, `unsafe` inside and safe at its surface —
  handles are `Closeable` values, errors are `sys.OsError { code, call }`.
  Usable directly for what the portable layer does not cover (the low
  level), and the place a contributor adds a missing call.
- **Portable modules**: the shortest safe spelling (the high level). Every
  device module has `raw()` to reach its `sys` handle — the escape is part
  of the design, not an `unsafe` cast.
- **Rules every module follows** (the existing house style, applied):
  discovery returns values; an open device is a `with` value; a byte channel
  is an `io.Stream`; events are a `Subscription` (`recv` suspends, works in
  `race`, closes with `with`); every wait is cancellable and honours
  `withTimeout`; blocking OS calls run on `blocking`; errors are typed with
  a `kind` enum (`Unavailable`, `Denied`, `NotFound`, `Disconnected`,
  `Busy`), and `Denied` says what to do (the Linux group, the Windows
  setting); bounded reads (`max:`) where data comes from outside.
- **A simulated backend for every device module** (`bluetooth.Simulated`,
  `serial.loopback()`), so a program's tests run in CI and in WSL, which has
  no Bluetooth or USB hardware. Real-hardware tests run on Windows behind an
  environment flag.
- **Capabilities**: `bluetooth`, `usb`, `serial`, `hid` join the package
  capability list (D138 f), so `veles audit` shows which dependency talks to
  hardware and a `[policy]` can refuse it.

### Bluetooth LE — the API written out (Q42; proposed)

```veles
use bluetooth { Uuid }
use io { println }

const HEART_RATE = Uuid.short(0x180D)
const MEASUREMENT = Uuid.short(0x2A37)

/// The heart-rate characteristic: a flags byte, then the rate as u8 or u16.
fun bpm(value: List<u8>): i64? {
  val flags = value.at(0) ?: return null
  if (flags & 1 == 0) value.at(1)?.toI64() else value.readU16Le(1)?.toI64()
}

fun main() throws bluetooth.Error | Timeout {
  val adapter = try bluetooth.adapter()               // kind: Unavailable, PoweredOff, Denied
  val found = try withTimeout(Duration.seconds(10), () => {
    with scan = try adapter.scan(services: [HEART_RATE])    // scanning stops when the block ends
    await scan.next()                                 // an Advertisement: name, rssi, services, data
  })
  with link = try found.device.connect()               // disconnected on every way out
  val rate = try link.characteristic(HEART_RATE, MEASUREMENT)
  with readings = try rate.subscribe()                 // notifications: a Subscription<List<u8>>
  loop (value in readings) {
    if (val n = bpm(value)) println("$n bpm")
  }
}
```

The same program's test, with no radio:

```veles
test "prints the rate a monitor notifies" {
  val sim = bluetooth.Simulated()
  val monitor = sim.device(name: "HRM", services: [HEART_RATE])
  monitor.characteristic(HEART_RATE, MEASUREMENT).notifyAfterSubscribe([[0, 72], [1, 0x2c, 0x01]])
  expect(collectRates(sim.adapter()) == [72, 300])
}
```

Surface, first version (central role): `adapter()`, `adapters()`,
`Adapter.scan(services:, duplicates:)`, `Advertisement`, `Device.connect()`,
`Link.services()`, `Link.characteristic(service, id)`, `Characteristic.read()`,
`write(bytes, response: true)`, `subscribe()`, `Link.mtu()`,
`Link.disconnected()` (an `Event`), `Uuid` (`short`, `parse`, constant
`==`). Later: the peripheral role (advertise, a GATT server), pairing, and
classic RFCOMM as an `io.Stream`.

Implementation per platform:

| | Windows 10 1803+ | Linux | macOS (later, A7) |
|---|---|---|---|
| API | WinRT `Windows.Devices.Bluetooth*` (the Win32 GATT functions need prior pairing and cannot scan) | BlueZ over D-Bus (`org.bluez.Adapter1`, `Device1`, `GattCharacteristic1`) | CoreBluetooth |
| needs first | COM/WinRT calls from Veles: vtables as `extern struct`s of `extern fun` pointers, `HSTRING`, async completions arriving on pool threads (already supported as foreign-thread callbacks) → `std/sys/windows/com` | Unix sockets (`net`), then a D-Bus client in Veles (`std/sys/linux/dbus`, SASL EXTERNAL) | the Objective-C runtime |
| reused by | audio (WASAPI), clipboard, notifications, HID | NetworkManager, systemd, notifications | — |

A COM layer and a D-Bus client are each worth more than Bluetooth: they
open most of what each OS offers.

### The other modules, proposed order (Q41)

1. **`net`: UDP, multicast, Unix sockets, `net.lookup`** — small, and a
   prerequisite for D-Bus (Linux Bluetooth) and mDNS.
2. **`std/term`** — raw mode, size, key events as a `Subscription`, colour
   honouring `NO_COLOR`, a progress line; also the compiler's coloured
   diagnostics and a future REPL.
3. **`fs.watch(dir)`** → a `Subscription` of changes (inotify /
   `ReadDirectoryChangesW`); `veles test --watch` and the LSP use it.
4. **`fs.map(path)`** — a read-only memory-mapped file as bytes; the compiler
   reads every source this way.
5. **`std/serial`** — ports listed, opened as an `io.Stream` with baud,
   parity, flow control.
6. **`std/bluetooth`** — LE central, as above.
7. **`std/hid`** — devices listed, reports read and written (hidraw / the
   Windows HID API).
8. **`std/usb`** — devices, interfaces, control/bulk/interrupt transfers
   (usbfs / WinUSB); the largest.

### `veles bindgen` (Q43)

Every binding above starts as hundreds of `extern` declarations. clang is
already required, and `clang -Xclang -ast-dump=json` parses any C header
exactly; `veles bindgen <header> [--filter prefix]` would write the `extern`
blocks, structs (with D120 layout attributes), unions, enums as constants
and function-pointer types, checked by compiling a `sizeof`/`offsetof` probe
as `TestCLayoutAgreesWithC` does. Windows metadata (`.winmd`) for WinRT is a
second input later.

### What a platform module needs from the language (Q39)

Per-target code, written in Veles:

```veles
// proposed: a file per target, chosen by its suffix (Go's rule)
std/serial/serial.vs            // the portable surface
std/serial/serial_windows.vs    // fun openPort(...) over CreateFileW / SetCommState
std/serial/serial_linux.vs      // fun openPort(...) over open / termios

// and, for a small difference, a compile-time branch (D113 constants, D117's dead-branch rule)
use build { Os }
val sep = if (build.target.os == Os.Windows) "\\" else "/"     // decided as D160
```

Both branches are type-checked on every platform (a typo on Linux is found
on Windows); only the taken one is compiled and linked.

---

## 6. Order of work

1. **B17** (D142 visibility) — decision-free now, largest migration first.
2. The F19 migration left in checklist §11 (std/examples/docs where a
   `scope` only launched and awaited) and its bench.
3. **S6, S7, S8** — finish the front end, measure its speed, debug info.
4. **B18** — the formatter's one-line `unsafe`, more index facts.
5. With the answers to Q32–Q38: build them in order of reach (Q32 first —
   it is a correctness fix; Q33, Q35, Q36 before S9 so the formatter port is
   written in the new shape).
6. **S9–S12**.
7. With the answers to Q39–Q43: **track P** — P1 per-target code, P2 `net`
   additions, P3 `term`, P4 `fs.watch`/`fs.map`, P5 `sys/windows/com` and
   `sys/linux/dbus`, P6 `serial`, P7 `bluetooth`, P8 `hid`, P9 `usb`,
   P10 `bindgen` (can start any time after P1).

---

## 7. The decisions (checklist §9)

Answered the same day: Q32 → **D157**, Q33 → **D158**, Q42 → **D159** (as
recommended); Q39 → **D160** with the target spelled `build.target.os` (the
user's choice over a free `target`); Q40 + Q41 → **D161** (all four module
groups); Q43 → **D162**; Q34 → **D163**, Q36 → **D164**, Q37 → **D165**
(as recommended); Q35 → **D166** (after the full write-up in §4), Q38 →
**D167**. Nothing is open; the build order is in `veles-plan.md` ("Where to
continue").

| Q | Question | Recommended |
|---|---|---|
| Q32 | Interpolating a `Result`/`Task` | an error with `try`/`??` fixes — **D157** |
| Q33 | Members without `this.` | bare names resolve to `this`'s members after locals — **D158** |
| Q34 | Positional construction | one-field types (counting `init` params) take one positional argument |
| Q35 | Labelled tuples | `(key: K, value: V)`; names readable, structure unchanged; `entries()` uses them |
| Q36 | Copy with changes | generated `copy(field: v, …)` on every struct, visibility as construction |
| Q37 | `indexOf` answering `-1` | `i64?` across the family, migrated |
| Q38 | Argument parsing | `std/cli` by derive (`implement cli.Args`, `cli.parse<T>`) |
| Q39 | Per-target code | file suffixes `_windows`/`_linux`/`_macos` + `target.os` constant branches, both checked |
| Q40 | Where OS bindings live | `std/sys/<os>` in Veles; portable modules on top; C runtime only for what must be C |
| Q41 | Which platform modules, in what order | §5's list, UDP/Unix first, USB last |
| Q42 | Bluetooth API | LE central as written in §5, simulated backend, `bluetooth` capability |
| Q43 | Binding generator | `veles bindgen` over clang's JSON AST |
