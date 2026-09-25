# Veles — rewriting the lexer and parser in Veles

A plan for the first slice of self-hosting. Against `veles-spec.md` v0.34.
This is a sequencing document with gates, not a schedule.

`veles-build-plan.md` puts self-hosting at Stage 5 in one line. This file
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
| `lexer/` | 891 | 106 token kinds, automatic semicolons, interpolated strings, doc comments |
| `ast/` | 1822 | 92 node structs in five disjoint families, plus the S-expression printer |
| `parser/` | 2576 | recursive descent, Pratt expressions, error recovery, the formatter's `Layout` |
| `source/` | 183 | files, byte spans, line/column, diagnostics |
| | **5472** | |

Expect 1.2–1.5× that in Veles — call it 6500–8000 lines — most of it
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

87 `.vs` and `.vss` files in the repository today (`std/`, `examples/`,
`docs/`), plus:

- the ```veles blocks the docs test already extracts,
- `parser/testdata/fuzz/` and `sema/testdata/fuzz/` — the interesting
  ones, because they are malformed and exercise error recovery,
- deliberately broken files written for the purpose: unterminated strings,
  stray brackets, a `when` with no arms, tabs and CRLF, a file of 10 000
  lines.

The corpus must include bad input. A front end that agrees on valid files
and diverges on invalid ones is worse than useless, because the divergence
shows up as a confusing error message on the day a user makes a typo.

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
search — the prelude has no `binarySearch`, so write one here and consider
promoting it later (`notes_to_change`).

Deliverable: a program that reads a file and prints
`path:line:col` for a byte offset given on the command line, matching
Go's `Position` on every offset in every corpus file. A silly program that
pins the one function everything else's spans depend on.

**Gate P0:** offsets agree, including on CRLF files and on the offset one
past the end.

### P1 — tokens (≈900 lines)

`selfhost/lexer/`: `TokenKind` as an `enum` (D57 — 106 members, and
`toString()` comes free, which is exactly what G1 needs to print),
`Token`, `Comment`, `StringPart`, the scanner, the keyword map, automatic
semicolon insertion, `--tokens`.

The order to write it in, so each step is testable: punctuation and
operators (longest match first) → identifiers and keywords → numbers →
strings without interpolation → comments and docs → automatic semicolons →
interpolation → character literals.

**Gate P1 = G1** over the whole corpus, including the fuzz findings.

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
  file.

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
- Module-level `var` is a real mutable global, if any interning table
  wants one.
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
