# Veles — syntax decisions to settle (prepared 2026-10-20)

Prepared with `veles-decide` for the seven items the user picked from
`veles-syntax-ideas.md` (S4, S2, S6, S9, I10, I9, I1) plus the explanation of
S3. Language facts below were checked by running snippets on the compiler of
2026-10-20 (the probes are quoted where they matter). Counts are over the
hand-written Veles source (`std/`, `selfhost/`, `examples/`, `bench/`).
Nothing here is built; an answer becomes a spec entry and a plan task.

---

## 1. `x in c` and `x !in c` as expressions (I3)

**Problem.** Membership is written as a chain of equalities or a pair of
comparisons. Counts: 81 lines of `x == a || x == b`, 39 of them with three or
more alternatives; 64 `byteAt(i) == '…'` comparisons; 38 range tests
`x >= a && x <= b`; 45 "outside a range" tests. The lexer's whitespace test:

```veles
if (b == ' ' || b == '\t' || b == '\r' || b == '\n') …
```

**What exists.** `in` is the loop keyword and a `when` pattern
(`when (c) { in 65..90 => … }` works). Elsewhere it is a parse error. The
library has `Range.contains` (`('A'..'Z').contains(c)` works for a `u8`),
`List.contains`, `Set.contains`, `Map.containsKey`, `string.contains(string)`.
Two gaps found by probing: `" \t".contains(b)` with `b: u8` is a type error
(only a string fits), and `[65, 66].contains(c)` with `c: u8` is an error
because the literal list is `List<i64>` (D179 means no overloading, so
`contains(u8)` cannot be added beside `contains(string)`).

**Options.**

- **A. Leave it.** Write `('A'..'Z').contains(c)` and equality chains.
- **B. The operator means `contains`** (recommended). `x in c` is
  `c.contains(x)` for a list, set, range or string; for a `Map` it is
  `containsKey` (Kotlin and Python read it as the keys). `x !in c` is the
  negation, spelled like the existing `!is`. The right operand of a literal
  set is compiled with no allocation: `b in [' ', '\t', '\r', '\n']`, `b in 'a'..'z'`,
  `b in "xyz"` (a string literal as a set of bytes) become comparisons,
  expected type of the elements taken from the left operand — which also makes
  the `List<i64>` literal problem disappear.
- **C. As B, but also a `u8` in any string** (`b in someString` = the byte
  occurs). A second meaning for `in`, because `contains(u8)` cannot exist as a
  method; the literal-string form is already covered by B.

```veles
// today                                      // with B
if (c >= '0' && c <= '9') …                   if (c in '0'..'9') …
if (b == ' ' || b == '\t' || b == '\n') …     if (b in [' ', '\t', '\n']) …
if (!STOP.contains(word)) continue            if (word !in STOP) continue
```

**Edge cases.** A nullable left operand is a type error (as `==` of a
non-nullable with `T?` is a warning today). Precedence: with comparisons, so
`a in xs && b` groups as `(a in xs) && b`, and `a in xs == true` is rejected
as ambiguous. Inside `loop (x in xs)` and a `when` arm `in` keeps its
meaning; the parser reads the loop head first, so no clash. Bounds facts
(D62): `i in 0..<xs.len()` establishes `0 <= i < len`, so `xs.at(i)` below it
is a `T` — the same fact a `loop (i in …)` gives. A user type with a
`contains` method gets the operator for free (resolved at the use, no new
trait). Suspension: none (a method call). Threads: none. Formatter and hover:
the hover shows the `contains` it resolved to.

**Elsewhere.** Python `in`/`not in` (any container). Kotlin `in`/`!in` →
`contains`. Swift and Rust: none (Rust has `matches!` and `contains`). Go:
none. TypeScript `in` is a different thing (keys of an object).

**Cost.** Parser (a binary operator at comparison level, `!in` as a token
pair); sema (lower to the resolved `contains`, plus the literal-set
compilation); formatter; LSP tokens; the self-hosted parser; docs and a
conformance case per rule. Small to medium.

**Both levels / self-hosting.** High level: reads as the condition it is.
Low level: the literal forms compile to the compares written by hand today,
so nothing is slower. The lexer and the parser of the compiler are full of
these tests.

**Recommendation: B.** (C only if you want `b in text` for a non-literal
string; I would not — two meanings for one operator.)

---

## 2. A loop head that binds: `loop (val x = e) { … }`

**Problem.** 71 places write `val x = f() ?: break` as the first line of a
loop, and 136 loops are a bare `loop {` (most of them have that shape or a
leading `if (…) break`).

```veles
// today
loop {
  val tok = this.next() ?: break
  handle(tok)
}
```

**What exists.** D95: `if (val x = e && cond)` binds the non-null value for
the rest of the condition and the branch (probed: `if (val v = x && v > 2) …`
works). A `loop (val x = e)` is today an error ("only allowed in the condition
of an 'if'").

**Options.**

- **A. Leave `?: break`.**
- **B. The same head as `if`** (recommended): `loop (val tok = this.next()) { … }`
  runs while the value is not null; `&& cond` continues the condition;
  `continue` evaluates the head again; `break` leaves.
- **C. A new keyword form** (`while let`-style). Rejected up front: a second
  loop keyword for what the head already says.

```veles
loop (val tok = this.next()) {            // B
  handle(tok)
}
loop (val line = try reader.readLine() && !line.isEmpty()) {   // the head may try / await
  out.push(line)
}
```

**Edge cases.** `val x = e` in a loop head and `x in c` differ by one token,
which is the same distance as `if (val …)` from `if (cond)`. A `Result` head
is not unwrapped (as in D95, only nullable); `val x = e else …` (let-else)
stays a statement. Shadowing follows `if`. A suspending head makes the loop
suspend as its body would. Labels work as for any loop. The bounds facts
treat the binding like an `if` binding. D102 (a collection changed while a
loop walks it) concerns `loop (x in c)` heads only.

**Elsewhere.** Rust `while let Some(x) = e`; Swift `while let x = e`;
Kotlin has none (`while (true) { val x = … ?: break }`); Go: `for` with an
`if … break`.

**Cost.** Parser (the loop head accepts `LetCond`, which the `if` head already
parses), sema (lower to `loop { val t = e; if (t == null) break; … }`, which
the `if` lowering already does for its condition), formatter, the
self-hosted parser, docs. Small.

**Recommendation: B.**

---

## 3. The missing compound assignments

**Problem.** `+= -= *= /= %=` exist. `|= &= ^= <<= >>=` and the wrapping
forms `+%= -%= *%=` are parse errors ("expected an expression, found '='",
probed). 31 hand-written updates of the shape `x = x | y` exist in the
hashing, bit-reader and compression code, and `+%` appears throughout the
crypto code.

**Options.**

- **A. Leave.**
- **B. Add the five bit operators** (`|= &= ^= <<= >>=`).
- **C. B plus the three wrapping ones** (`+%= -%= *%=`) (recommended). Same
  rule as `+=`: `x op= y` is `x = x op y` with the place of `x` evaluated
  once, on integers only.
- **D. C plus `&&=` and `||=`.** Not recommended: `x &&= f()` hides a
  short-circuit that a reader must remember, and the code has no use for it.

```veles
h ^= b                       // today: h = h ^ b
bits |= (b.wrapU32() << shift)
acc +%= word                 // today: acc = acc +% word
```

**Edge cases.** The shift count's type follows the existing `<<`. A compound
form on a `bool` (`ok &= check()`) is not offered (`&&` is the operator for
booleans). Overflow behaves as the operator it abbreviates (D21: checked
`+=`, wrapping `+%=`). The formatter and `veles explain` need the tokens; the
removed-syntax machinery is not involved (nothing is removed).

**Elsewhere.** All of C, Go, Rust, Java, Kotlin (not the wrapping family, a
Veles feature) and Swift have the bit forms.

**Cost.** Lexer tokens, parser precedence entry, `checkAssign`, formatter, the
self-hosted lexer and parser, docs. Small; pure consistency.

**Recommendation: C.**

---

## 4. A string slice that is never null

**Problem.** `substring(from, to): string?` is null when a bound falls outside
the text or cuts through a UTF-8 code point (D19). 122 of the 130 calls in
the code say `?: fallback`, and the fallbacks are mostly not decisions:
`?: ""` 47 times (the largest group), `?: "0"` 5, `?: return` 6, `?: text` 5,
the rest panics with a reason.

```veles
this.src.substring(i, i + size) ?: "\u{FFFD}"
text.substring(0, at) ?: ""
```

**Options.**

- **A. Keep** (callers keep the `?:`).
- **B. Add `slice(from, to): string`** (recommended), the sibling of
  `List.slice`: both ends are clamped to the text, a bound inside a code
  point moves *outward* to the character boundary, `from >= to` is empty.
  `substring` stays for callers that must tell "invalid" from "empty".
- **C. A view type for byte ranges of one string.** Not needed for text: a
  `string` is already a pointer and a length, and `substring` shares the bytes
  of the original (runtime `veles_string_substring`: `out->data = s + lo`; no
  allocation, the collector follows the interior pointer). The open question
  is a read-only window over a `List<T>` (bytes), see `veles-research-views.md`.

```veles
val word = src.slice(start, i)           // B: "" when start >= i, no ?:
```

**Edge cases.** Moving a cut bound outward (not inward) keeps every byte the
caller asked for; the alternative "inward" can lose the byte a lexer is
looking at. A negative bound clamps to 0. `slice`, like `substring`, shares
the original's bytes: no copy and no allocation (the whole original stays
alive while a piece of it does, as with Go substrings).

**Elsewhere.** Go `s[a:b]` panics out of range and cuts bytes; Rust `&s[a..b]`
panics on a non-boundary; Python clamps (by characters); Kotlin
`substring` throws, `take`/`drop` clamp; Swift uses indices that cannot be
invalid.

**Cost.** One library function with a test per rule; docs; replacing the
`?: ""` sites is optional (a lint could suggest it). Small.

**Recommendation: B.**

---

## 5. Variants declared inside the sealed trait (I10)

**Problem.** A sealed family is declared as the trait plus one top-level
`struct` per variant, each naming the trait again. The self-hosted AST has
88 variants in 1 028 lines (`Expr` 41, `Decl` 16, `Stmt` 14, `Type` 9,
`Pattern` 8). Nesting a struct in the trait is refused today: "expected 'type'
or 'fun' in trait body, found keyword 'struct'" (probed).

```veles
sealed trait Expr

struct IntLit : Expr {
  var text: string = ""
  var pos:  Span = Span.none()
}
```

**Options.**

- **A. Leave.**
- **B. Nested structs** (recommended): the trait body may contain `struct`
  declarations, each an ordinary struct with every struct feature (fields with
  defaults and `var`, methods, `implement`, attributes), implicitly a variant
  of the enclosing trait. The variant names stay top-level, as today
  (`IntLit`, not `Expr.IntLit`).
- **C. A data-only short form**: `Num(value: i64)`, `Plus` inside the trait.
  Shorter for tiny variants, but needs its own rules for `var` fields,
  defaults, doc comments and attributes — the AST uses all of them.

```veles
sealed trait Expr {                       // B
  struct IntLit {
    var text: string = ""
    var pos:  Span = Span.none()
  }
  struct FloatLit { var text: string = ""; var pos: Span = Span.none() }
}
```

**Edge cases.** Generic sealed traits (`sealed trait Result<T, E>`) nest the
same way, with the variants' type parameters declared as today. The old form
keeps working (no removal), so the migration of a big file is optional and can
be done by a `--fix` quick fix ("move into the trait"). Visibility: a nested
struct is as visible as the trait unless marked. `when` exhaustiveness is
unchanged. Hover and go-to-definition for a variant name must point into the
trait.

**Elsewhere.** Kotlin nests subclasses in a `sealed class`/`interface`
(allowed beside or inside); Rust's `enum` variants are inside the enum (with
the enum's name as prefix); Swift enum cases; TypeScript unions are separate
declarations as in Veles today.

**Cost.** Parser (struct declarations in a trait body), AST (a trait lists its
nested variants), the loader (declare them as variants), formatter, LSP
outline and rename, the self-hosted parser, docs. Medium. **Self-hosting
value: the highest of the list** — it shortens the file the compiler port
reads most, and makes the closed set visible at the trait.

**Recommendation: B.**

---

## 6. Methods on enums (I9, reopens D57)

**Problem.** D57 kept enums to named values with nothing to write: no methods,
`implement` or `extend` ("the compiler refuses them: it already compares,
hashes, orders and prints by itself"). A function over the enum reads
backwards:

```veles
fun label(level: Level): string => when (level) { … }
println(label(entry.level))          // today
println(entry.level.label())         // wanted
```

Probed: `extend Level with fun label(): string => …` and the block form
`extend Level { … }` are both refused with the D57 message — D173's
extension functions stop at enums too.

**Options.**

- **A. Leave** (functions that take the enum).
- **B. Extension functions only** (recommended): `extend Level with fun label()`
  and `extend Level { … }` are allowed. No fields, no `implement`, so the
  representation, comparison, hashing, printing and `Codable` derivation stay
  the compiler's; the methods are static dispatch and cannot shadow `value`,
  `toString`, `values`, `fromValue`, `parse`.
- **C. B plus `implement Trait for Level`**, so an enum can satisfy a user
  trait and be a trait object or a generic bound (`T: Describe`). Larger: the
  impl tables, trait objects over an integer-backed value, and the question of
  overriding `toString` through a trait.

```veles
enum Level { Debug, Info, Warn, Error }

extend Level {                           // B
  fun label(): string => when (this) {
    Level.Debug => "dbg"
    Level.Info  => "inf"
    Level.Warn  => "wrn"
    Level.Error => "err"
  }
  fun isSevere(): bool => this >= Level.Warn
}
```

**Edge cases.** D185: an extension is in scope where the module's `use … extend`
clause brings it, exactly as for other extensions; inside the module that
declares the enum it is in scope. A `static fun` in the block (`Level.default()`)
would be the one question: allowed, if `extend` blocks allow statics elsewhere
(D173 says which). Name clashes with the generated members are errors with the
name of the member. `const fun` is allowed. Enums stay non-generic.

**Elsewhere.** Rust and Swift enums have methods; Kotlin enum classes have
methods and can implement interfaces; Go has methods on any named type; Java
enums are classes.

**Cost.** Remove the refusal for extension functions, make sure the receiver
reads as the enum value, docs (D57 gets an addendum, not a strike-through),
a conformance case. Small for B; C is medium-large.

**Recommendation: B**, C later if a real need appears.

---

## 7. Generators: `yield` (I1)

**Problem.** A lazy producer is a struct with state and a `next()` that is
written as a state machine by hand. A tree walk needs an explicit stack:

```veles
struct Walk : Iterator {
  type Item = *Node
  private stack: MutableList<*Node>
  fun next(): (*Node)? { … 15 lines … }
}
```

`yield` is a parse error today (probed: "expected an expression, found
keyword 'yield'"). The compiler already turns suspending functions into
heap-allocated state machines (coroutine frames), so a generator is the same
transformation with `yield` as the suspension point.

**Options.**

- **A. Leave.** Iterators stay structs; the std keeps its adapters.
- **B. `yield` in a function returning `Iterator<T>`** (recommended):

  ```veles
  fun walk(root: *Node): Iterator<*Node> {
    yield root
    loop (child in root.children) loop (n in walk(child)) yield n
  }
  loop (n in walk(tree).filter(n => n.isLeaf()).take(10)) …    // lazy
  ```
  The body runs when the iterator is pulled; `return` ends the sequence.
  A generator may `throw` (the error surfaces at the `loop` that pulled the
  element) and may suspend (an async generator is an `AsyncIterator`, D42).
- **C. A builder function instead of a keyword**: `iterator { emit -> … emit(x) … }`
  (Kotlin's `sequence { yield(x) }`), the body a suspending lambda. No new
  keyword; the cost moves into closures with their own frames.

**Edge cases.** *Early exit*: `loop (n in walk(t)) { if (…) break }` abandons
the generator; resources held inside it (a `with`) must be closed, so the
`loop` head must treat a generator as something it closes on break — the same
protocol `Closeable` iterators need. *Recursion*: `loop (n in walk(child)) yield n`
costs depth per element; a `yield all walk(child)` form (Kotlin's
`yieldAll`, Python's `yield from`) removes it and can be a second step.
*Sendability*: a generator frame is not `Sendable` unless everything it
captures is. *Generics*: a generator in a generic function is instantiated per
type like any function. *`const fun`*: not evaluable at compile time at first.
*Debugging*: a panic inside shows the call chain through the pulling loop.

**Elsewhere.** Python and JavaScript `yield`; C# `yield return`; Kotlin
`sequence { yield }` (option C); Rust has unstable generators; Go: channels
or the new `iter.Seq` push functions; Swift: `AsyncStream`.

**Cost.** Large: a `yield` statement, the effect rules (a generator is its
own kind of function), the coroutine transform for a pull model, the
`loop`-closes-the-iterator protocol, formatter, LSP, the self-hosted parser
and checker. **Self-hosting value: real** (tree walks are everywhere in
`sema`; `fs.walk` could stream instead of returning a `List`). Do it after
items 1–6.

**Recommendation: B, last in the order.**

---

## S3 explained — `skipWhile` and `indexWhere`

The 24 one-line scan loops in the code (`loop (i < n && isDigit(src.byteAt(i))) i += 1`)
have the same two shapes. As library functions:

```veles
// today (selfhost lexer)
var i = start
loop (i < n && isIdentChar(src.byteAt(i))) i += 1
val word = src.substring(start, i) ?: ""

// with the helpers
val end  = src.skipWhile(from: start, isIdentChar)  // index of the first byte after
                                                    // `start` where the predicate is false
                                                    // (the length when it never is)
val word = src.slice(start, end)                    // decision 4
val dot  = src.indexWhere(from: start, b => b == '.')   // i64?: the first index where it is true
```

`skipWhile` is the scanner's "advance over"; `indexWhere` is `indexOf` with a
predicate. The same pair for `List<T>` (`xs.indexWhere(from: 0, x => x.done)`).
They are `const fun`s taking a function, so the call is inlined like `map` and
the generated loop is the one written by hand. The names are the only open
question (Kotlin: `indexOfFirst`, `dropWhile`; Rust: `position`, `skip_while`;
Swift: `firstIndex(where:)`; Go: `strings.IndexFunc`). Not asked yet; if you
want them, the names go through one decision batch.
