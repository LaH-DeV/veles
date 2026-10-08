# Veles — rewriting the lexer and parser in Veles

A plan for the first slice of self-hosting. Written against `veles-spec.md`
v0.34; **refreshed against v0.49 on 2026-09-29 (plan S1)** — what changed is
in §8, and the capability audit for the *whole* compiler is §9 (plan S3).
This is a sequencing document with gates, not a schedule.

`archive/veles-build-plan.md` puts self-hosting at Stage 5 in one line. This file
is that line expanded for the front end only — and argues that the front
end is worth doing *before* the rest of Stage 5, because it is the one part
that can be proved correct against the existing implementation instead of
against a test suite someone has to write.

---

## 0. The cut

**In scope.** `lexer/`, `ast/` and `parser/`, plus the part of `source/`
they need (files, positions, spans, diagnostics). Rewritten in Veles, in
the repository, as a package that builds with the Go-hosted compiler.

**Out of scope.** `sema/`, `types/`, `codegen/`, `driver/`, `format/`,
`lsp/`. Nothing in the Go compiler changes except a new test that runs the
Veles front end and compares its output.

**Not a replacement.** At the end of this plan the Go front end is still
what `veles build` uses. The Veles one is a second implementation that
provably agrees with it. Replacing it means porting `sema`, which is
five times the work and a different decision.

### What the Go front end is, by the numbers

| Package | Non-test lines | What it is |
|---|---|---|
| `lexer/` | 1036 | ~107 token kinds, automatic semicolons, interpolated strings, doc comments, UAX #31 identifiers |
| `ast/` | 2017 | ~100 node structs in five disjoint families, plus the S-expression printer |
| `parser/` | 2945 | recursive descent, Pratt expressions, error recovery, the formatter's `Layout` |
| `source/` | 355 | files, byte spans, line/column, diagnostics |
| | **6353** | |

*(2026-09-29: was 5472 lines at v0.34; `wc -l` of the non-test files.)*

Expect 1.2–1.5× that in Veles — call it 7500–9500 lines — most of it
mechanical. The interesting part is not the volume.

---

## 1. Why the front end first

**It is the only phase with a free oracle.** A lexer is a function from
text to tokens and a parser is a function from text to a tree. Both are
deterministic, both already have a working implementation, and both already
have a *printed* form: `lexer` tokens are dumped by `veles parse` and
`ast.Dump` renders a file as an indented S-expression. So equivalence is
a string comparison over a corpus, not a judgement call. No other phase has
that: `sema`'s output is a `Program` graph, and "the same types were
inferred" has no printed form to diff.

**It has almost no runtime surface.** Files in, text out. No sockets, no
tasks, no FFI beyond `fs.readFile`. If the Veles front end is slow or
leaks, the reason is in the language, not in a binding.

**It is the biggest single dogfooding program available.** The examples are
hundreds of lines; this is thousands, written by someone who wants the
language to be pleasant, on a problem — walking a tagged tree — that Veles
was designed for (sealed traits, `when`, exhaustiveness). Every rough edge
it finds is one a user would have found.

**It exercises the parts of the design that are least proven.** 92 sealed
variants across five families, 75 one-line `span()` methods, thousands of
short-lived heap nodes, and deep recursion — a stop-the-world mark-sweep
collector and a recursive descent parser meeting for the first time.

---

## 2. The equivalence harness

This is the load-bearing idea; everything else is ordinary porting.

### The corpus

173 `.vs` and `.vss` files in the repository today (`std/`, `examples/`,
`docs/`, `bench/`; 87 when this was written), plus:

- the ```veles blocks the docs test already extracts,
- `parser/testdata/fuzz/` and the conformance suite's error files
  (`sema/testdata/conform/*.vs`, deliberately wrong programs) — the
  interesting ones, because they are malformed and exercise error recovery,
- deliberately broken files written for the purpose: unterminated strings,
  stray brackets, a `when` with no arms, tabs and CRLF, a file of 10 000
  lines.

The corpus must include bad input. A front end that agrees on valid files
and diverges on invalid ones is worse than useless, because the divergence
shows up as a confusing error message on the day a user makes a typo.

### The current language only (user, 2026-10-08)

The Go front end also reads the spellings Veles has removed — `self`,
`impl`, `fun <T> f`, `mut fun`, `use m.{ }`, `T::Item`, the `{ e => }`
handlers — and reports each with a fix, so that `veles check --fix`
migrates an old program. The self-hosted front end does not: there `self`
is an identifier and the others are ordinary syntax errors. A corpus file
in which the Go parser reports a removed spelling (the list is
`removedSpellings` in `selfhost_test.go`) is therefore outside G1–G3;
`TestRemovedSpellings` checks instead that the Veles parser reports an error
in it, and that `edge/p-removed` makes the Go parser report every entry of
the list, so the list cannot fall behind. Four files are out today: two
conformance cases written in the old spellings, one fuzz finding and
`p-removed`.

Newcomers' mistakes are not removed spellings and stay in both: `for`,
`->`, `=` in a condition, `do … while`, `fun (x) { }`.

### The three comparisons

| Gate | Veles side prints | Go side prints | Why this one |
|---|---|---|---|
| **G1 tokens** | kind, byte span, text, auto-semi flag, one per line | the same, from `lexer.TokenizeAll` | the lexer alone, before any tree exists |
| **G2 tree** | `ast.Dump` equivalent | `ast.Dump(parser.ParseFile(...))` | the parser, including every span |
| **G3 diagnostics** | every diagnostic: span, severity, text | `source.Diagnostics.Render()` | error recovery and message wording |

Byte-identical output on all three, over the whole corpus, is the
definition of done. Not "equivalent trees" — identical text, because
`ast.Dump` already prints every field and every span, and a span that is
off by one is a formatter bug waiting to happen.

### How it runs

A Go test, `selfhost/selfhost_test.go`, in the same shape as
`examples/examples_test.go`:

1. build `selfhost/` with `veles.exe build` once per run,
2. for each corpus file, run the Veles binary with `--tokens`, `--tree`,
   `--diags`,
3. compute the Go answer in-process,
4. diff, and on a mismatch print the first differing line with context.

`-update` writes nothing: there is no golden file to update, the Go
compiler *is* the golden file. That is the whole point.

### The final gate

**G4: the Veles front end parses itself, and agrees.** Its own 7000 lines
go into the corpus. This is the first genuine bootstrap milestone in the
project, and it is reachable without a type checker.

---

## 3. Phases

Each phase ends in something runnable. No phase is "port the rest".

### P0 — `source` and the harness skeleton (≈250 lines)

`selfhost/source/`: `File` (path, content, line table), `Span`,
`Position(offset)`, `Line(n)`, `Diagnostic`, `Diagnostics` with `Errorf`,
`Warnf`, `Render`.

The line table is built once in the constructor and searched with a binary
search — the prelude's `partitionPoint` (§4: `binarySearch` and its family
exist since 2026-09-23).

Deliverable: a program that reads a file and prints
`path:line:col` for a byte offset given on the command line, matching
Go's `Position` on every offset in every corpus file. A silly program that
pins the one function everything else's spans depend on.

**Gate P0:** offsets agree, including on CRLF files and on the offset one
past the end.

*Done 2026-10-08.* `selfhost/source/source.vs` (`File.of`, `position`,
`line`, `lineCount`, `Span` with `to` and `"$span"`, `Severity`,
`Diagnostic`, `Diagnostics` with `errorAt`/`warnAt`/`render`),
`selfhost/main.vs` (`--positions`, `--lines`, `--render`) and the harness
`selfhost/selfhost_test.go`: every `.vs`/`.vss` in the repository (331
files with the edge files: empty, CRLF, lone `\r`, multibyte characters at
line ends, ten thousand lines) agrees with Go's `source` on every offset,
every line and a rendered diagnostic every 37 bytes. The line table is
searched with the prelude's `partitionPoint`. Two things P1 must settle:
the files that are not valid UTF-8 are left out of the corpus today
(`fs.readFile` refuses them; the lexer will read bytes), and `render` does
not yet print the `see: veles explain` lines (the families come with the
messages).

### P1 — tokens (≈900 lines)

`selfhost/lexer/`: `TokenKind` as an `enum` (D57 — ~107 members, and
`toString()` comes free, which is exactly what G1 needs to print),
`Token`, `Comment`, `StringPart`, the scanner, the keyword map, automatic
semicolon insertion, `--tokens`.

The order to write it in, so each step is testable: punctuation and
operators (longest match first) → identifiers and keywords → numbers →
strings without interpolation → comments and docs → automatic semicolons →
interpolation → character literals.

**Gate P1 = G1** over the whole corpus, including the fuzz findings.

*Done 2026-10-08.* `selfhost/lexer/` — `token.vs` (`Kind` in the Go
order, `KEYWORDS` and the kind names as constant tables, `Token`,
`StringPart`, `Comment`), `lexer.vs` (the scanner, automatic semicolons,
interpolation, documentation comments, the module doc), `ident.vs` (UAX #31
and how a diagnostic names a character) and `tables.vs`, the identifier and
"invisible" character classes generated from the Go lexer's own decisions by
`selfhost/tables_test.go` (which fails when they go stale) — about 900 lines
of Veles plus the table. `selfhost --tokens` prints every token (kind number
and name, span, inserted or not, text, doc, string parts), every comment,
the module doc and every diagnostic; G1 agrees on all 357 corpus files,
including hand-written broken ones (unterminated strings, comments and
interpolations, bad escapes, numeric suffixes, invisible and bidirectional
characters, a BOM, every doc-comment shape, chain continuations, unbalanced
brackets). The one fuzz finding today is not valid UTF-8 and is left out,
with every such file, until the lexer reads bytes (`fs.readFile` refuses
them). Three Go front-end bugs found by porting, each fixed in both
lexers: `TokenKind.String()` ranged over the keyword map, so a message
about `this` said `'this'` or `'self'` at random; `"\é"` reported
`unknown escape sequence '\Ã'` and then a bogus "invalid UTF-8" (the escape
took one byte of the character); and the caret under a diagnostic counted
bytes, so after any non-ASCII character or a tab it stood to the right of
the span and was too wide.

### P2 — the AST (≈1500 lines, no logic)

`selfhost/ast/`: five sealed traits (`Type`, `Decl`, `Stmt`, `Expr`,
`Pattern`) and 92 variant structs, each with `fun span(): Span`.

The Go families are disjoint — no node implements two markers — which is
what makes this a mechanical translation rather than a redesign. Two
details need a decision, both small:

- Go's `Node` interface (`Span()` over all five families) has no sealed
  equivalent. Where the printer needs it, take the family; where it needs
  genuinely any node, it is a five-arm `when`. Count the sites first: if it
  is fewer than five, do nothing.
- `ast.CallExpr.T any` and `FieldDefaultExpr.Struct any` hold a
  `types.Type` so that `ast` need not import `types`. The Veles port drops
  both fields: they are filled by `sema`, which is out of scope, and
  `ast.Dump` prints `Struct` only for a synthesized node no parser
  produces.

**Gate P2:** the package type-checks and a hand-built tree of every
variant round-trips through the printer. No parser yet.

### P3 — the printer (≈900 lines)

`selfhost/ast/print.vs`: `Dump`, `DumpDecl`. This is ported *before* the
parser, so that P4 has its oracle from the first declaration it can parse.

Three Go formatting behaviours have to be reproduced exactly:

- `%q` on a string — Go's `strconv.Quote`: the escapes, `\xNN` for
  non-printables, `\u` above ASCII. Used for the `extern` ABI and string
  parts. Needs a faithful `quote()` helper; write it here and consider a
  `std/strconv`-shaped home for it later.
- `%v` on a `bool` — `true`/`false`, which Veles interpolation already
  gives.
- indentation and the exact parenthesisation of the S-expressions.

**Gate P3:** `DumpDecl` of the hand-built trees from P2 matches strings
captured from the Go printer.

### P4 — declarations and types (≈700 lines)

`parseFile`, `parseDecl` and `parseType`: `use`, `fun`, `struct`, `trait`,
`implement`, `enum`, `val`, `error`, `extern`, `extend`, plus the nine type
forms. Bodies parse as `BadStmt` placeholders at first.

**Gate P4:** G2 over a corpus subset of signature-only files (write them:
the std modules with their bodies removed are a good source).

### P5 — statements and expressions (≈1400 lines)

The Pratt table (`infixBp`, `parseUnary`, `parsePostfix`), all 35
expression forms, all 13 statement forms, patterns, `when` arms, lambdas
and the two places the parser needs a speculative retry — type arguments
before a call (`tryTypeArgsBeforeCall`) and the arm-head-is-not-a-lambda
rule (`noLambda`, `parseArmHead`).

These two are where a reimplementation is most likely to diverge, because
both are "try, and rewind on failure" with a diagnostics buffer that must
be discarded. Port them with their comments intact and give each its own
corpus file.

**Gate P5 = G2 and G3** over the whole corpus. The front end is now
complete for valid input and for every malformed input in the corpus.

### P6 — `Layout`, then self-parse (≈300 lines)

`ParseFileLayout`'s comment list and parenthesised-expression spans — the
formatter's input. Nothing consumes it on the Veles side yet, so it is
gated by G2-style comparison of the two `Layout` values printed.

Then add `selfhost/**/*.vs` to the corpus.

**Gate P6 = G4.**

---

## 4. What the language and library are missing

Everything below was checked against the compiler as it stands at D59.
None of it is a blocker; four of them are decisions worth making before P1
rather than during it.

> **Settled, 2026-09-23.** All four decisions below were taken as
> recommended and the work is in the tree, so P0 starts against a library
> that has what it needs. What changed:
>
> - **`std/utf8`** exists (decision 1a): `decode`, `decodeLast`,
>   `decodeBytes`, `encodeTo`, `encode`, `char`, `size`, `isScalar`,
>   `isSurrogate`, `isContinuation`, `isStart`, `combineSurrogates`,
>   `isValid`, `count`, and `utf8.Rune { code, size }`. `std/json` decodes
>   its `\u` escapes through it now, so it has a caller from day one, and
>   `examples/utf8` pins it against the Unicode Table 3-7 and Kuhn stress
>   vectors. The lexer's three decode sites are `utf8.decode(s, i)`.
> - **`unicode.IsPrint` turned out not to be a gap at all** (decision 2,
>   and *neither* of its options). The call is unreachable for anything
>   above ASCII: `isIdentStart` accepts every byte ≥ 0x80 (D18), so
>   `operator()` — the only caller — is only ever reached with a one-byte
>   rune. The Go lexer now says `showAsItself(r)`, which is `r >= 0x20 &&
>   r != 0x7F`, and `lexer/lexer_test.go` pins both that and the fact that
>   an invisible character like U+00A0 or U+202E is lexed as an *identifier*
>   rather than reported. The Veles port needs no Unicode data and has no
>   documented divergence. (That a bidirectional override is a legal
>   identifier byte is a real question, but it is D18's, not this plan's —
>   `notes_to_change` #21.)
> - **Superseded 2026-09-25 (D18 addendum):** identifiers are now UAX #31,
>   so the paragraph above no longer holds. The Go lexer asks Go's
>   `unicode` tables for ID_Start/ID_Continue and names invisible characters
>   in its errors (`lexer/ident.go`); the Veles lexer needs the same
>   answers, so P1 gains a generated table — the ID_Start/ID_Continue ranges
>   above ASCII, written out by a small Go program from the same `unicode`
>   package, so the two lexers cannot disagree about which Unicode version
>   they follow. `describeRune`'s wording is part of the token-dump gate.
> - **One recursion limit** exists (decision 3a), in the prelude:
>   `maxRecursionDepth` (1000, a stack budget — see its doc comment),
>   `tooDeepMessage(limit)` so every caller reports the same sentence, and
>   `Depth.of(n)` with `enter`/`leave`/`deepest` for a walk that keeps no
>   stack of its own, which is exactly the parser's shape —
>   `examples/recursion` is that parser in miniature, and is the shape P5
>   should copy. Writing it closed a hole in `std/json` on the way past: the
>   *encoder* was unbounded, so a tree too deep to decode could still be
>   walked on the way out.
> - **`selfhost/`** is the package location (decision 4).
>
> Also done, from "gaps that are just work": **`binarySearch`** and its
> family are in `std/prelude/list.vs` — `binarySearch`, `binarySearchWith`,
> `binarySearchBy`, `lowerBound`, `upperBound`, `partitionPoint`. P0's line
> table is `starts.partitionPoint(s => s <= offset) - 1`; nothing needs to
> be written locally and promoted later.
>
> Still open, deliberately: the **Go-compatible `%q`** of P3. It is left
> where the plan put it, because getting `strconv.Quote` right to the
> character is P3's job and choosing its permanent home (a `std/strconv`?)
> is a decision that wants the caller to exist first.

### Gaps that need a decision

**1. Stepping one UTF-8 code point.** The lexer decodes a rune in three
places: copying one character out of a string literal, reading a character
literal, and reporting an unexpected character. Veles has `byteAt`,
`bytes()`, `substring` and `charCount`, and no rune API at all.

- *(a)* A `std/utf8` module in Veles over `byteAt`: `decode(s, at): (code:
  i64, size: i64)?`, `encode(code): List<u8>`. No compiler change, useful
  to every user who touches text, and the lexer needs perhaps 40 lines of
  it. **Recommended.**
- *(b)* Builtins `string.codeAt(i): i64?` and `string.charSize(i): i64`.
  Faster, but adds surface to the catalogue for a rare need.
- *(c)* Probe with `substring(i, i+1)` … `substring(i, i+4)` and take the
  first that is not null. Works today with no new code, and is
  embarrassing.

**2. `unicode.IsPrint`.** One call, in one error message, on one path:
"unexpected character". Getting it right needs Unicode category tables;
getting it wrong is a divergence on input no corpus file contains.

- *(a)* Reproduce Go's behaviour for ASCII and treat every non-ASCII rune
  as printable. One-line divergence, documented, and the corpus can pin
  the ASCII half. **Recommended** — and worth *changing the Go lexer to
  match*, so the two agree by construction rather than by luck.
- *(b)* A `std/unicode` with the `IsPrint` ranges (~200 lines of tables).
  Real, and nothing else needs it yet.

**3. Recursion depth.** `parseBinary` → `parseUnary` → `parsePostfix` →
`parsePrimary` → `parseExpr` is unbounded, and a file of 50 000 `(`
overflows the stack in *either* implementation. Go's stacks grow; Veles
runs on the C stack.

- *(a)* A depth counter in the parser, `maxDepth` in the low thousands,
  reported as an ordinary diagnostic. Cheap, and §2 of the checklist
  already wants exactly this for the JSON decoder and the router — so do
  it once, the same way, in both. **Recommended.**
- *(b)* Nothing, and a crash on hostile input.

**4. Where the package lives.** `selfhost/` with its own `veles.toml` and
modules `source`, `lexer`, `ast`, `parser` is the obvious shape and needs
no new compiler feature (a local directory is a module, `use lexer`
resolves it). The names collide with the Go package names, which is
confusing in conversation but not in the build. Alternative: `front/` with
modules `vsource`, `vlexer`, … which is ugly but unambiguous.
**Recommended: `selfhost/`, and say "the Veles lexer" when it matters.**

### Gaps that are just work

- ~~**`binarySearch`** for the line table (P0) — 15 lines. Worth promoting to
  `std/prelude/list.vs` afterwards.~~ Done, in the prelude.
- **Go-compatible `%q`** (P3) — 40 lines. Worth a home in std later.
- **A keyword map** — `Map<string, TokenKind>` works; a module-level `val`
  is initialised once at start-up (D59's note), so it costs nothing per
  file. Measured 2026-09-29: `bench/lexer` (byte scan, keyword map, token
  structs) runs at 0.6–1.5× Go.

### Things that turned out not to be gaps

Worth writing down, because each was a plausible blocker:

- `substring` **does not copy** — the runtime aliases the original bytes
  (`veles_string_substring`), so a lexer that takes a substring per token
  is O(n), not O(n²).
- Identifiers are classified **by byte**: `_`, ASCII letters, or any byte
  ≥ 0x80. No Unicode tables needed for the common path.
- The five AST families are **disjoint**, so sealed traits fit without
  redesign, and `when` over them is exhaustively checked — strictly better
  than Go's type switch with a `default: panic`.
- Speculative parsing needs only an index to save and restore, and a
  scratch `Diagnostics` to throw away. Both are ordinary values.
- ~~Module-level `var` is a real mutable global, if any interning table
  wants one.~~ **Struck 2026-09-29 (D66, built 2026-09-27):** a module-level
  `var` whose type is not `Mutex`/`Atomic` is a compile error, because tasks
  run on one thread per core. An interning table is a value the parser or
  lexer *owns* (a struct field, as `bench/intern` does with `Interner`), or a
  module-level `val` of a `Mutex<...>`; the keyword table is a plain
  module-level `val`.
- Enums give `toString()` for free, which is what G1 prints.

---

## 5. Risks, in the order they are likely to bite

**Allocation rate, not correctness.** A parser makes one heap object per
node and throws away most of a token list. The collector is stop-the-world
mark-sweep. Parsing 7000 lines might allocate ~100 000 objects.

*Mitigation:* measure at the end of P1, when a token list is the only
output and the number is easy to attribute. Print peak heap and total
bytes allocated with `VELES_GC_THRESHOLD` varied. If the front end is more
than ~10× the Go one, that is a finding about the collector — exactly the
kind §3.1 of the checklist is waiting for — and it belongs in the
benchmark set (§3.3) rather than being worked around here.

*The trap to avoid:* rewriting the port to allocate less. The value of
this exercise is that it is written the way a user would write it. Arena
tricks would hide the finding.

**Span arithmetic.** Every `Span` in Go carries a `*source.File`. In Veles
that is a struct reference; passing it by value through 92 node
constructors is a copy per node unless `File` is held behind a reference.
G2 catches a wrong span immediately, but a *slow* span is invisible.

*Mitigation:* decide the representation in P0 and write it down. A `Span`
of `(start, end)` with the file implied by the parser is smaller and is
what most compilers do; it changes `Span`'s API, so it is a P0 decision,
not a P5 discovery.

**Divergence in messages, not structure.** G3 compares diagnostic text.
Every one of the ~150 error messages must match to the character, including
the ones that interpolate a token's text.

*Mitigation:* accept that some will differ, and fix them **in the Go
lexer** where the Veles wording is better. The point of the exercise is two
implementations that agree, and the Go one is not sacred.

**Scope creep into `sema`.** The moment the Veles parser works, the
temptation is to resolve names with it.

*Mitigation:* the plan ends at G4. `sema` is a separate decision with a
separate oracle problem, and it is where the cut-over risk the build plan
warns about actually lives.

---

## 6. What this buys, and what it does not

**Buys:**

- the first bootstrap milestone, reachable without a type checker;
- a second implementation of the grammar, which is how ambiguities get
  found (the leading-dot pattern conflict in `notes_to_change` #18 is
  exactly the kind of thing a second reading exposes);
- the largest dogfood program the language has had, on a problem it was
  designed for;
- a real workload for the collector and the generic machinery, with a
  baseline to compare against;
- a `std/utf8`, a `binarySearch` and a Go-compatible `quote()` that
  everybody else gets to use.

**Does not buy:**

- a self-hosted compiler — `sema`, `types` and `codegen` are ~20 000 more
  lines and have no free oracle;
- a faster front end — expect slower, and that is information, not
  failure;
- anything a user of the language can see, until the cut-over.

---

## 7. Suggested order against the rest of the checklist

The front end needs nothing from §1.2 (FFI), §1.3 (threads) or §5.3–5.8,
so it can run alongside them. It wanted two things from the checklist
first, and both are done (2026-09-23), so P0 is unblocked:

1. ~~**§2 stack depth** — one recursion limit, shared by the decoder, the
   router and the parser (decision 3 above).~~ Done; §2 is ticked. The
   router turned out not to recurse, so the sharers are the decoders, the
   encoders and, from P5, the parser.
2. ~~**`std/utf8`** — not on the checklist yet; add it under §5.10, because
   the lexer is not the only program that will want it.~~ Done; §5.10 is
   ticked, and `std/json` is already a caller.

And it *produces* one: `bench/parse` belongs in §3.3 from P1 onward.

---

## 8. What changed since v0.34 (S1, 2026-09-29)

Checked against D60–D91 and the compiler as it stands. Nothing here moves a
gate; each line is something P0–P6 would otherwise have tripped over.

| Change | What it means for the port |
|---|---|
| **D66 — a module-level `var` must be `Mutex`/`Atomic`** | struck the "real mutable global" claim (§4). State lives in a struct the lexer or parser owns; the keyword table is a `val`. |
| **D65 — `this` replaces `self`; the `init { }` block (v0.30)** | structs with derived fields (line tables, keyword sets) use `init` as-is; a parser that keeps token positions next to its tokens needs no `static fun` for the derivation. |
| **D78 — `test "sentence" { }` with `expect`/`require`** | unit tests of the Veles lexer and parser sit beside the code in `*.test.vs` and run with `veles test`; the equivalence harness stays a Go test, because it compares against the Go front end. |
| **D86 — no `as`; conversions are methods** | a byte-to-digit step is `(b - '0').toI64()`-shaped, truncation is `wrapU8()`; every `as` in the plan's sketches reads as a method. |
| **D85 — named imports** | `use lexer { Token, TokenKind }` instead of qualifying every use; the Go package names (`lexer`, `ast`) collide with nothing. |
| **D89 — `public use`** | `selfhost/` can present one facade module (`public use lexer`, `public use parser`) without a flat namespace. |
| **D18 addendum — UAX #31 identifiers** | already in §4: a generated ID_Start/ID_Continue table (P1). |
| **D64, D81, D88 — panics carry locations, call chains and the caller's line** | an invariant broken in the port reports `at file:line:col`, and in a debug build the chain, which makes a G1–G3 mismatch easier to chase. |
| **D90, D91 — `lazy`, `std/log`** | `log.debug("token $t")` costs a closure while off (~30 ns, checklist §5.7): fine for diagnostics, not for the scanner's per-byte path, where the port logs nothing. |

Still true as written: `std/utf8`, `partitionPoint`, `recursion.Depth`, the
disjoint AST families, `substring` not copying, `selfhost/` as the location.
Not yet true: a Go-compatible `quote()` (P3's job).

Measured 2026-09-29 (`bench/lexer`, `bench/ast`): scanning with a keyword map
and token structs runs at 0.6–1.5× Go, and a sealed-family tree built and
walked with `when` at 0.4–0.6× Go. The §5 "allocation rate" risk is therefore
not expected to bite; the check at the end of P1 stays as written.

---

## 9. Capability audit for the whole compiler (S3, 2026-09-29)

The front end is the cheap slice. `sema`, `types`, `codegen` and `driver` are
a different program: large graphs, interning, maps of structured keys,
reproducible ordering, a subprocess, and recursion over user input. Each row
was probed with a small program, not assumed. **has** means it works today
and is measured or tested; **partial** and **gap** are listed below the
table.

| Need | Verdict | Evidence |
|---|---|---|
| Large pointer graphs, with cycles, under the GC | **has** | 800 000 nodes with parent↔child cycles (`var parent: (*Node)?`, `kids: MutableList<*Node>`) build and walk; `bench/trees` 0.2–0.5× Go |
| A sealed family walked with `when`, exhaustively | **has** | `bench/ast` 0.4–0.6× Go; a missing arm is a compile error |
| Interning: a string-keyed map handing out dense ids | **has** | `bench/intern` 0.6–2.0× Go |
| Maps keyed by a struct, and by a sealed-family value | **has** | `MutableMap<Key, i64>` with `Key { a: i64, b: string }`, and `MutableMap<Type, i64>` over a sealed `Type`: equal values find the same entry |
| Deterministic iteration order | **has** | maps and sets are insertion-ordered by guarantee (D25), so output is reproducible without sorting keys |
| Sorting: stable, by key, by comparator | **has** | `sorted`, `sortedBy`, `sortedWith`, `sortWith`; stable (chapter 8) |
| Building large text | **has** | `bench/emit`: 250 000 lines of IR through one `StringBuilder`, 0.4–1.2× Go |
| Tokenising a large source | **has** | `bench/lexer` 0.6–1.5× Go |
| A subprocess with captured output and its exit code | **has** | a C file compiled and run through `os.run("clang", ...)`; `Output.code`, `ok()`, stdin and stderr capture |
| File I/O, directory walk, paths, args, env, exit codes | **has** | `std/fs`, `std/path`, `std/os` |
| Parallel work (per-function codegen) | **has** | tasks on one thread per core (D66); `mapConcurrent` |
| Emitting a float constant in LLVM's hex form | **has, since 2026-09-29 (D93)** | `x.toBits()` / `f64.fromBits(bits)`; was **partial**, see below |
| Deep recursion | **gap, closed 2026-09-29 (D92)** | see below |

**Partial — float bits (closed the same day, D93: `toBits()` / `fromBits()` in std, `examples/floatbits`).** As found: there was no `f64` ↔ `u64` bit reinterpretation in
std. It works with an `unsafe` cast (`p.cast<*raw u64>()`: 1.5 gives
0x3FF8000000000000), but a code generator should not need `unsafe` for it. A
safe `toBits()` / `fromBits()` is a std decision; it goes through
`veles-decide` when the code generator is on the horizon (checklist §5.10).

**Gap — deep recursion (closed the same day, D92: 256 MB stacks on every thread, the program started on one, and a `stack overflow` panic that names the function; a million frames run).** As found: a task runs on a worker thread's native stack, at
the OS default. A recursion that cannot be turned into a loop overflows at
roughly 30 000–50 000 frames on Windows (1 MB) and 200 000–1 000 000 on Linux
(8 MB), and the failure is **silent**: exit code 127 on Windows, a bare
segmentation fault on Linux, no message and no function name.
`recursion.Depth` stays the answer for walks over input (a limit of 1000 with
one sentence for the error). What is missing is (a) a diagnostic when the
stack is exhausted anyway and (b) a stack size the program can rely on (a
larger main and worker stack, reserved and not committed). Checklist §2.

What this means for a self-hosted `sema`: nothing found blocks it. The two
findings become checklist items (§2 and §5.10). Not probed, because it has no
bearing until the rewrite starts: the compile time of a 30 000-line Veles
program by the Veles compiler itself, and the peak memory of a whole-program
HIR under the collector; both are E4 measurements.

## 10. The compile-time evaluator is part of the port (2026-10-08)

`const fun` (D113 part 3) makes the compiler run Veles code while it
compiles: `sema/consteval.go`, `constfun.go` and `constnum.go` interpret the
checked HIR — about 2 300 lines of Go. A Veles-written `sema` must carry the
same evaluator, with the same results: exact integers that fail where the
program would, `f32` rounded after every operation, `"${x}"` printed as the
run time prints it, the same step budget and call-depth limit, and the same
whitelist of built-ins (`constBuiltins`) and std functions marked `const`.
Two consequences for the rewrite: the evaluator is a port cost in its own
right (an interpreter over a sealed HIR is the shape `bench/ast` measures),
and its results are part of the byte-identical gate — a constant table the
Go compiler lays out must come out the same from the Veles one, so
`driver/constfun_test.go`'s differential cases move to the harness with it.

## 11. The port is Veles, not Go in Veles (user, 2026-10-08)

The first port followed the Go code line by line and read like it: `(Token,
bool)` results and `val (_, ok) = …` at every call, `val _ = …` in front of
each call whose result went unused, a `box<T>` helper for every pointer, a
type annotation to steer each construction, two forty-line functions that
patched `doc` and `internal` into every declaration kind afterwards, and the
removed spellings carried over. The rule since the review: the
behaviour is Go's, byte for byte, and the code is written as Veles is
written —

- a check that may fail returns `bool` (`expect`) or `T?` (`expectIdent`),
  read with `?:`, `else` and `if (val x = …)` (D95); a result that does not
  matter is not bound;
- a node is boxed with `&value` (the expected pointee type now reaches the
  operand, D10, so `&ast.ImplDecl(…)` stays an `ImplDecl`); a variable that
  is rewrapped in a loop (`left = Binary(left: &left, …)` would point at
  itself) goes through a function whose parameter is a fresh binding — the
  Pratt loop's `parseInfix`, `parsePostfixOp`, `typePostfix`;
- what is written before a declaration (`DeclHead`: attributes, doc,
  visibility, start) is parsed once and handed to the declaration's parser,
  which builds the node whole;
- names are words and follow the current language: `Kind.KwThis`,
  `KwSelfType`, `KwImplement`, `KwPublic`, `ast.ThisExpr`, `left`/`right`;
- `veles fmt` is applied;
- the Veles sources do not mention Go (user, 2026-10-08): a comment says
  what the code does, not which Go function it copies (`quote`, not
  `goQuote` after `strconv.Quote`); only the harness, which is Go and
  compares against Go, names the bootstrap compiler.
