---
name: veles-code
description: Write idiomatic Veles source — std modules, examples, benchmarks, doc programs, probe programs — and design std APIs the way this project does. Use before writing or editing any .vs/.vss file.
---

# Writing Veles

The spec is `veles-spec.md`; the fastest syntax refresher is
`docs/documentation/reference/cheatsheet.md`. When unsure whether
something compiles, write it to the scratchpad and run
`veles check <dir>` — do not guess from Kotlin or Rust.

## Current spellings (older forms are errors with fixes)

- Receiver is `this`; methods live in the struct body; traits via
  `implement Trait for T` (or inline in the body); `extend T { }` only for
  types the package declares.
- Generic functions: `fun name<T: Bound>(…)`. Visibility: `public`,
  `private`, nothing (module). Fields: bare = never assigned, `var` =
  assignable.
- No `!!`, no `…OrPanic`. Prove the access instead: list patterns
  (`val [a, b] = xs else return …`, `when (xs) { [x, ..rest] => … }`),
  index loops the checker understands, or `xs.at(i) ?: panic("why this
  cannot fail")` with a real reason.
- Guards: `val x = r else { e => … }` (let-else), `?:` for nullables,
  `??` for a Result. Errors are `error Name { }`; functions `throws`,
  callers `try`.
- Element access is methods only: `at`/`set` on lists, `get`/`set` on
  maps; `ref` for a write through a pointer. Brackets are literals; `[:]`
  is the empty map.
- Time is typed: `Duration.seconds(5)`, `await sleep(d)`,
  `withTimeout(d, f)`. `await` only goes on primitives that always suspend
  (D16); ordinary suspending calls are not marked.
- `when` arms are separated by newlines, not commas.

## Style the user expects

- Use the language's own idioms: `loop cur = x ?: return`,
  `(a..b).step(n)`, `.reversed()`, `zip`, list patterns, eager
  `map`/`filter` on lists.
- Names are words: no `l`, `r`, `rr`, `cs`, `sb` (`n`, `i`, `j` are fine).
- `val` for a reference-typed collection mutated only through methods.
- Examples do real work — a small tool, a real algorithm, a real protocol —
  not `foo`/`bar` demonstrations.
- Nothing may depend on task scheduling order; synchronise explicitly.
- Run `veles fmt` on what you write; the formatter keeps your line breaks
  in lists and chains, so break them where a reader would.

## Designing std APIs

- Names keep their JS/Kotlin meaning (`fill` overwrites in place);
  short verbs (`repeat`, `make`) over participles; Kotlin-flavoured, not
  Rust (`compareTo`, `toString`, `Comparable`). `use` is the import
  keyword, so a method cannot be called `use`.
- **Panic vs throw**: a caller bug (bad size, index proven impossible)
  panics with a message naming the function and the value; anything the
  environment or outside input can cause (I/O, parsing, network, a clock)
  throws a typed error (`IoError`, a module error). Untrusted input never
  reaches a panic or an overflow.
- Bounded by default: a read from outside takes a required limit
  (`readLine(max:)`); recursion over untrusted data uses the prelude
  `Depth`/`maxRecursionDepth`.
- Generics that duplicate a value across tasks are bounded `T: Sendable`.
- Types over raw numbers where confusion is possible (`Duration`,
  `Timestamp`, `Digest` with constant-time `==`).
- A footgun is refused by the type checker, not documented.
- Every public declaration gets a `///` comment; std tests use published
  vectors (RFCs, FIPS) where they exist.
- Any new public name or signature is a decision (`veles-decide`) unless
  the plan item already fixes it.

## Where things go

- std: `std/<module>/`, embedded — rebuild the compiler to test a change.
  `veles check <repo>/std/<module>` loads that module from disk. Generic
  prelude bodies are checked only when instantiated, so exercise them from
  a program.
- examples: `examples/<name>/main.vs` + `expected.txt` (see `veles-test`).
- probes and dogfood programs: the scratchpad, never the repo.
