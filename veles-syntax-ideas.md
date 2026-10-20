# Veles — syntax and ergonomics ideas (2026-10-20)

Written on the user's request ("let's think about more places that syntax
could be better just for developing in this language … a lot of them … think
them through and explain and ask. no code yet"). Nothing here is decided or
built; an answer becomes a spec entry through `veles-decide`.

**Evidence.** Counts are over the Veles source of the repository that is
written by hand: `std/`, `selfhost/`, `examples/`, `bench/` (252 files, test
files excluded), measured 2026-10-20 with small scripts. "S" ideas are new
here; ideas that already sit on `veles-idioms-and-apis.md` keep their `I`
number and only get new evidence.

**Already in the language** (probed, so nothing below re-proposes it):
multi-pattern `when` arms (`1, 2, 3 =>`), a subject-less `when { c => … }`,
tuple swap `(a, b) = (b, a)`, numeric literals with separators (`1_000`, `0b101`),
`loop ((i, x) in xs.enumerate())`, `(0..10).step(3)`, `(0..<5).reversed()`,
`val x = e else return`, `if (val x = e && c)`, `%=`, subject `in` inside
`when` arms and `loop`. Rejected earlier and not re-proposed: `it`, trailing
lambdas, `!!`, `|>` (D173, "for now"), brackets for indexing, `::`.

---

## A. Loops

### S1 — Counter loops (the user's question: `loop (i < chars.len()) with i++`)

**What the code does today.** 105 loops are hand-written counters
(`var i = …; loop (i < n) { …; i += 1 }`). In 84 the increment is the last
statement; in 21 it is somewhere else (a lexer advancing by a token's length,
deflate's run lengths). 70 step by 1, 35 by something else. 6 contain a
`continue`. Another 42 loops are already `loop (i in 0..<xs.len())` and 38
use `enumerate()`.

**How many could be range loops already.** A strict check (the `var` right
before the loop, the counter assigned only by the final `+= 1`, not read
after the loop) finds 24 candidates and 16 of them are exactly
`loop (i in a..<b)` — for example the lexer's "find the byte" loop:

```veles
// today
var i = from
loop (i < this.src.len()) {
  if (this.src.byteAt(i) == b) return i
  i += 1
}
-1
// already possible, nothing new
loop (i in from..<this.src.len()) {
  if (this.src.byteAt(i) == b) return i
}
-1
```

**What no range loop can express** (the rest): the counter advances by a
variable amount (`i += size`), is changed inside the body, or is needed after
the loop (a scan that ends where a token ends).

**The bug a header step would prevent.** `continue` jumps over a manual
increment and the loop never ends:

```veles
var i = 0
loop (i < n) {
  if (skip(i)) continue     // forgets i += 1: runs forever
  work(i)
  i += 1
}
```

The HIR already has this shape (`Loop.Post`: "runs at the end of every
iteration and on `continue`"; range loops use it), so a step clause is a
parser-and-spelling question, not a new mechanism.

**On the spelling you proposed.** `with i++` has two obstacles: `with` is the
resource keyword (D100) and the parser answers `loop (i < 3) with i += 1 { }`
with "'with x = e' closes x when its block ends"; and there is no `++`
(368 `+= 1` and 56 `-= 1` in the code; `i++` is a parse error).

Options:

- **A. No new syntax; two lints and the existing range loop (recommended).**
  `loop_is_range`: `var i = a; loop (i < b) { …; i += 1 }` with the counter
  untouched elsewhere and unused afterwards → "write `loop (i in a..<b)`",
  with a fix. `continue_skips_step`: a `continue` that can be reached before
  the counter's increment → warning naming the line. Zero cost, catches the
  bug class, nudges the 16 convertible loops. Leaves the variable-step loops
  as they are.
- **B. A step clause after the condition**: `loop (i < n) step i += 1 { … }`
  (any short keyword; `step` is free, `then` reads worse). The counter stays
  declared outside (so it survives the loop for scans), `continue` runs the
  step, and `break` does not. Fixes the `continue` bug by construction.
  Cost: parser, AST, formatter, LSP, the self-hosted parser, docs, a
  conformance case per rule.
- **C. C-style header**: `loop (var i = 0; i < n; i += 1) { … }`. Familiar
  from C, Go and Java; scopes `i` to the loop (the scan loops that need it
  afterwards cannot use it); three clauses in one pair of parentheses beside
  `loop (x in xs)`. Same cost as B plus a scoping rule.
- **D. `loop (i < n; i += 1)`**: B without a keyword, `;` as the separator.
  The terse middle ground; the semicolon appears nowhere else in the
  language.

My pick is A now and B only if you want the form for readability. In every
option the body may not reassign a counter that the header steps (otherwise
the "only counts up" fact that lets `xs.at(i)` be proven, D62, is lost).

### S2 — A loop that binds: `loop (val x = e) { … }` (while-let)

71 places write `val x = f() ?: break` at the top of a loop, and 136 loops are
bare `loop {`. D95 already allows `if (val x = e && cond)`; `loop` can take the
same head:

```veles
// today
loop {
  val tok = this.next() ?: break
  handle(tok)
}
// idea
loop (val tok = this.next()) {
  handle(tok)
}
```

The loop ends when the value is null; `&&` continues the condition. Pros:
the same form as `if`, no `?: break` line, the narrowing is scoped to the
body. Cons: one more place for `val`; `loop (val x = e)` and `loop (x in c)`
look alike (the `=` vs `in` is the whole difference). Small (the parser has
`LetCond`). Recommended.

### S3 — Scan helpers instead of one-line loops

24 loops are `loop (i < n && pred(src.byteAt(i))) i += 1` (skip whitespace,
identifier characters, digits). A library function says it once:
`val end = src.skipWhile(from: i, isIdentChar)` (the index of the first byte
that fails) and `src.indexWhere(from: i, pred)`. Pros: no counter, no off-by-one,
inlined like `map`. Cons: an API to name (decision batch), lambdas hide cost
unless inlined. Library only; no syntax.

---

## B. Conditions and operators

### S4 — `in` and `!in` as an operator (I3, with evidence)

```veles
if (b in " \t\r\n") …          // today: b == ' ' || b == '\t' || b == '\r' || b == '\n'
if (c in '0'..'9') …           // today: c >= '0' && c <= '9'
if (word !in STOP_WORDS) …     // today: !STOP_WORDS.contains(word)
```

Evidence: 81 lines `x == a || x == b`, 39 with three or more alternatives, 64
`byteAt(i) == '…'` comparisons, 38 range tests `x >= a && x <= b`, 45
"outside a range" tests. `in` already exists in `when` arms and `loop`; outside
them it is a parse error. Meaning: `contains` on anything that has it (list,
set, map keys, range, string for a byte or a substring). Pros: reads as the
condition it is; the range form also feeds the bounds facts. Cons: one more
operator; `b in " \t"` for a byte in a string needs the rule "a `u8` in a
`string`" written down. Recommended.

### S5 — Chained comparison (`0 <= i < n`): not recommended

Python allows it; `0 < a < 3` is an error here (type mismatch bool/i64), which
is a good error. `i in 0..<n` (S4) is the same condition with one meaning.

### S6 — The missing compound assignments

`|= &= ^= <<= >>=` and `&&= ||=` are parse errors ("expected an expression,
found '='"); `%=` works. 31 hand-written `x = x | y`-style updates exist, all
in bit-twiddling code (hashes, bit readers). Pure consistency; the lexer
already has the compound-token machinery. Recommended; no design content.

### S7 — `++`, `--`, `?:=`: not recommended

`i++` would be a second spelling of `i += 1` (368 uses). `x ?:= v` ("assign
if null") has 2 candidate sites. A statement-only `++` is possible and
removes nothing; the cost is a second way to do it.

### S8 — `else if` ladders

131 `} else if` lines. The subject-less `when { a => …; b => …; else => … }`
exists (34 uses). Tooling instead of syntax: a lint `else_if_ladder` for three
or more branches with a fix to `when` (this is F10 in the review notes).
Recommended as a lint.

---

## C. Strings and numbers

### S9 — `substring` is nullable and 122 of 130 calls say `?: …`

`this.src.substring(i, i + size) ?: "\u{FFFD}"`: the answer is `null` when a
bound is out of range or not on a UTF-8 character boundary, and almost every
caller either knows it is not (ASCII scanning) or wants a fallback. Options:

- **A. Keep**; callers keep the `?:`.
- **B. Add `slice(from, to): string`** that clamps both ends to the string and
  moves them to the nearest character boundary, never null (the sibling of
  `List.slice`, which clamps). The few callers that must tell "invalid" keep
  `substring`. Recommended.
- **C. A `Span`/view type** for byte ranges of one string: the lexer's
  natural currency, bigger, a separate design.

### S10 — Numeric conversion noise (reopens D86)

`wrapU32()` and friends: 196 uses; `toI64()` 100; the other `toT()`s 57. They
are the price of D86 ("conversions are methods split by risk"): spelled out
by design. If you want to revisit: (a) a constant that fits its context needs
no conversion (I have not checked how far that already goes), (b) a lossless
widening is implicit in *arithmetic only* (`u8 + i64`),
(c) a shorter name for the wrap family (`x.u32()` / `x.wrap<u32>()`).
Not recommended without a reason beyond count: each option gives up something
D86 chose on purpose.

---

## D. Lambdas and noise

### S11 — Member-reference shorthand

57 lambdas have the shape `p => p.message` or `s => s.len()`. Options: keep
(D32: "every lambda names its parameters", `it` rejected); `xs.map(Item.name)`
meaning `x => x.name` (a type-qualified member as a function value — today
`Type.member` means a static call or a constant, so it needs a rule); a
leading-dot form `xs.map(.name)` (no name for the parameter at all, which is
the thing D32 refused). Not recommended: the lambdas are short and read well.

### S12 — `val _ = f()` where nothing needs discarding

73 statements `val _ = …` (`val _ = seen.add(x)`, `val _ = this.take()`). The
house rule in the idioms notes is "`val _ = f()` for a value that is not a
`Result` → just `f()`", and the compiler accepts a bare statement call (probed:
`s.add(1)` and `xs.pop()` as statements are clean). At least 16 are
`add`/`remove`/`take`/`pop`/`insert` style. A lint `redundant_discard` with a
fix drops them everywhere; the 57 left are `Result`s and `await`s that must
be named. Recommended (and I would then apply it repo-wide as for
`redundant_copy`).

### S13 — Properties instead of tiny methods: not recommended

25 `fun f(): T => this.x` getters. A `val x: T get` syntax is a second way
to write a method; the `.len()` / `.isEmpty()` style is consistent with the
language. No.

---

## E. Declarations (already on the menu, with new numbers)

- **I10 — variants declared inside the sealed trait.** `selfhost/ast/ast.vs`
  has about 90 data-only variants, each a `struct … : Token { … }` plus a
  separate declaration line. The short form halves that file and makes the
  closed set visible at the trait. Highest value for the compiler port.
- **I9 — methods on enums.** Today `extend Level` is refused (D57). A
  `label(level)` free function works but reads backwards from
  `level.label()`.
- **I1 — generators (`yield`).** `fs.walk`, a lexer handing tokens on, paged
  APIs: all want a lazy producer; coroutine frames already exist. Larger.
- **I12 — `requires` clauses.** 88 `at(i) ?: panic("…")` sites say by sentence
  what a clause would state by rule. Research-sized.

### S14 — `defer { … }`

One mention in the code. `with` + `Closeable` (D43) covers resources. A `defer`
would cover ad-hoc cleanup (restore a flag, unlock without a guard) and run on
panic and cancellation like `with` does. Value is low until a second use
appears; not recommended now.

### S15 — `when (val t = f()) { … }`

22 places write `val t = f()` and then `when (t)`. The binding could live in
the head, scoping `t` to the `when`. Cosmetic; not recommended.

---

## What I would do, in order

1. S12 `redundant_discard` lint and repo-wide cleanup (mechanical, like `redundant_copy`).
2. S6 the missing compound assignments (pure consistency).
3. S4 `in` / `!in` (largest count of any pattern).
4. S2 `loop (val x = e)`.
5. S1 option A (two lints); B only if you want it.
6. S9 `slice`.
7. S8 as a lint.
8. I10, then I1, from the existing menu.
