# Review of all Veles code written so far

Started 2026-10-19, at the user's request: "a review of every Veles code already
written to maybe improve it in terms of code quality / architecture /
performance / developer experience". Trigger: the first self-hosted TOML and
manifest port was a line-by-line copy of the Go code, with the same helper
written in six modules — the standard library was not being used as the
language's own proving ground. How the review runs was chosen the same day:
**an inventory and a findings log first, then fix batches** (the user, the
recommended of three).

This file is the plan and, below, the findings log. It is the source of truth
for the review; `veles-plan.md` has the row.

## 1. Scope

Every `.vs` file in the repository, about 45 000 lines.

| Area | Lines (code / tests) | Notes |
|---|---:|---|
| `std/http` | 7 000 / 2 600 | server, client, bodies, TLS glue |
| `std/prelude` | 6 000 / 270 | the language's own library: lists, strings, codecs, tasks, sync |
| `std/regex` | 4 700 / 360 | engine; `unicode.vs` is 2 100 generated lines |
| `std/otel`, `std/db` | 2 300 / 890, 2 300 / 580 | |
| `std/compress`, `std/crypto`, `std/lint` | 1 900, 1 700, 1 500 | `std/lint` mirrors the compiler's AST |
| `std/toml`, `std/fs`, `std/time`, `std/config` | 1 100, 940, 900, 890 | `std/toml` redone 2026-10-19 |
| the other 15 std modules | 4 000 | json, tls, net, jwt, log, utf8, os, base64, ffi, fmt, path, io, random, hex, codec |
| `selfhost/` | 10 000 | lexer, parser, ast, source, text, pkg — gated against the bootstrap compiler |
| `examples/`, `bench/` | 6 500 | |

Not in scope: the Go compiler (`lexer/`, `parser/`, `sema/`, `codegen/`, …) and
the C runtime. They are what the Veles code is checked against.

## 2. What each module is checked for

1. **Duplication and std gaps.** A helper written in more than one module, a
   loop that is a std method, a type that exists twice. Each is either deleted
   in favour of what std has, or becomes a std addition (a *decision*: a public
   name — `veles-decide`).
2. **Architecture.** One concern per file; layers that point one way; types
   that make the wrong state unrepresentable (sealed families and enums over
   flags, strings and tagged structs); no boolean threaded through the middle
   of code to switch a dialect; tables of data over `when` chains on strings
   (the settings registry); mutation confined and visible.
3. **Idiom.** The `veles-code` list: `?:`/`else` guards over null-check pairs,
   `if (val x = …)`, list patterns over index arithmetic, expression bodies,
   `loop` forms, names that are words, `///` on every public declaration.
4. **Performance.** Measured, never guessed (`veles-bench`): allocation in hot
   loops, `+` string building where `StringBuilder` is meant, repeated scans,
   `List` copies, locking. A finding carries its number.
5. **Developer experience.** Error messages that say what to do; the API read
   cold from its doc comment; the example in the docs compiles; hover and
   completion have something to show; tests name the behaviour.
6. **Tests.** The behaviour is pinned; a std module without a `.test.vs` is a
   finding; published vectors where they exist.

## 3. How it runs

- **Wave 1 — inventory (read-only).** For each module: size, public surface,
  imports, the five checks above, a list of findings. Nothing is changed. The
  order follows what depends on what: `prelude` → `utf8`/`path`/`fmt`/`io`/`os`
  → `regex` → `json`/`codec`/`config`/`toml` → `time`/`fs`/`net`/`tls`/`http` →
  `db`/`otel`/`crypto`/`compress`/`jwt`/`lint` → `selfhost` → `examples`.
- **Wave 2 — decision batches.** Findings that need a public name, a changed
  signature, or a changed behaviour go to the user through `veles-decide`, a
  batch per theme (not one per finding).
- **Wave 3 — fix batches.** A batch per module or theme: the change, the tests
  that pin it, the numbers where it was performance. Every batch ends with
  `go build ./... && go vet ./... && go test ./...` green and, for std, the
  Linux run.

**Rules.** No behaviour change without a decision. The selfhost gates
(`TestP*`, `TestR1*`) and the examples' `expected.txt` stay byte-identical. A
finding is written down even when it is not fixed now (checklist §11).

## 4. The findings log

One row per finding: `F<n>`, module, check (1–6), what and where, the fix
proposed, severity (**high**: wrong or a trap; **medium**: costs a reader or a
machine; **low**: tidiness), status (`open`, `decided`, `done`).

### Seeds (found while building R1, 2026-10-19)

| # | Module | Check | Finding | Proposed fix | Sev | Status |
|---|---|---|---|---|---|---|
| F1 | std (many) | 1 | `isDigit` written in seven modules, `hexValue` in four, and about thirty ranges (`b >= 'a' && b <= 'z'`) written out inline in base64, hex, json, uuid, otel, db, http, config, codec, regex, the selfhost lexer and pkg | `u8.isAscii…()`, `toAsciiLower/Upper()`, `digitValue` (D184) | high | **done** except `std/regex/parse.vs`, which works on code points (see F2). A grep for the range patterns found what a grep for the helper names could not: the first pass missed them |
| F2 | `std/regex/parse.vs` | 1 | works on code points (`i64`) and keeps its own `isDigit`, `isLetter`, `hexValue` | the shared Unicode/code-point module (F3), then delete | medium | open |
| F3 | `std/regex/unicode.vs`, `selfhost/lexer/tables.vs` | 1, 2 | two generated copies of Unicode tables (2 100 and 2 200 lines); the selfhost `text` module reaches into the lexer for them | one `std/unicode` module generated once: categories, scripts, ID_Start/Continue, printable; `utf8.isLetter(code)` … | high | open, needs a decision |
| F4 | `std/toml`, `selfhost/text` | 1 | `quote` (Go-style quoting for messages) written twice; `std/log` has a third (`json.encode`) | a public `fmt.quote` | medium | open, needs a decision |
| F5 | `std/lint/ast.vs` | 1 | 1 000 lines mirroring `selfhost/ast/ast.vs` | one definition, if the compiler's AST can be the lint API's | high | open, needs a decision (D177) |
| F6 | `selfhost/pkg` | 2 | `veles.toml` tables were `when (entry.key)` chains, and the package system knew the formatter's and linter's options | settings registry (`selfhost/settings`: `Schema`/`Setting`); each part declares its own schema (`formatter`, `linter` with the lint registry, `pkg`) — **done** internally; the public extension point for lint/format packages is to be designed | medium | internal done, public open |
| F7 | `std/http` | 2 | 7 000 lines in a handful of files (`http.vs`, `client.vs` ≈ 1 100 each) | split by concern: wire format, server, client, bodies, cookies | medium | open |
| F8 | `std/prelude` | 2, 6 | 6 000 lines, 270 of tests; `codec.vs` is 1 000 lines in one file | split codec by concern; tests for lists/strings/maps | medium | open |
| F9 | std (no tests) | 6 | `json`, `tls`, `jwt`, `utf8`, `os`, `base64`, `ffi`, `path`, `io`, `random`, `hex` have no `.test.vs` | tests with published vectors | high | open |
| F10 | selfhost, std/db, std/regex, std/http, std/config | 3 | `else if` chains of three or more branches testing one thing (lexer 28, db conn 12, regex parse 8, http client 6, config 6, print 6) read better as `when` | rewrite as `when`, one file at a time, gates green | medium | open (`regex expand` done) |
| F11 | selfhost/formatter, linter | 2 | enum ↔ word tables written as `when` (`ImportStyle`, `CommaStyle`, `LineEndings`) | `Setting.choice` over an enum's members, words from the member names | low | open |
| F12 | std | 1 | no `Version` type: `selfhost/pkg` has one; every package tool will need it | `std/version`: semver with constraints (decided by the user as `std/version`, scope "semver with constraints"); `pkg` then uses it | high | **done** (D186) |
| F13 | selfhost/ast/print.vs | 2 | one 1 000-line `Printer` struct | split by family with `extend Printer` blocks; experiment: derive the regular nodes' dump by reflection (D174) and keep only the exceptions by hand, if the output stays byte-identical | medium | open |

### Findings from the waves

*(filled in as each module is reviewed)*

## 5. Done means

Every module reviewed and its row in the log closed or in checklist §11; the
duplicated helpers gone; the std additions decided and built; the numbers of
every performance fix in `veles-checklist.md`; the docs and examples compile
against the final API.
