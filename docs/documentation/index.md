# Learning Veles

These tutorials are written for three kinds of readers. Pick the track that
matches you; each chapter is self-contained and every program in it
compiles and runs with the bootstrap compiler (the test suite checks that).

## Track 1 — New to programming, or new to typed languages

Start here if Python or JavaScript is what you know, or if this is your
first language.

1. [Getting started](01-getting-started.md) — install, run your first program, use the editor.
2. [Values, types and strings](02-values-and-strings.md) — `val`/`var`, numbers, text, `bool`.
3. [Functions and control flow](03-functions-and-control-flow.md) — `fun`, `if`, `loop`, ranges, `when`.
4. [Collections](04-collections.md) — lists, maps, sets and the operations on them.
5. [Structs and methods](05-structs-and-methods.md) — your own types, static functions.
6. [Nothing, maybe: nullable types](06-nullable-types.md) — `T?`, `?.`, `?:` and smart casts.

## Track 2 — Coming from Go, Kotlin, Swift, TypeScript or Java

Skim Track 1 (the syntax is close to Kotlin's), then:

7. [Errors: `throws`, `throw`, `try`](07-errors.md) — errors are values, without the ceremony.
8. [Traits and generics](08-traits-and-generics.md) — interfaces without inheritance, stenciled generics, trait objects, the operator traits.
9. [Sealed types, enums and `when`](09-sealed-types.md) — algebraic data types and closed sets of values, with exhaustive matching.
10. [Closures and iterators](10-closures-and-iterators.md) — lambdas, captures, lazy pipelines, `Iterable`.
11. [Modules, packages and tests](11-modules-and-packages.md) — directories as modules, `veles.toml`, `@test`.
15. [Files, paths and processes](15-files-and-processes.md) — `fs`, `path`, `os`, `StringBuilder`, `IoError`, `time`, `random`.

## Track 3 — Systems and concurrency

For readers who want to know what the compiler actually does.

12. [Concurrency](12-concurrency.md) — inferred suspension, `async`/`await`, `scope`, `gather`, `race`, channels, cancellation, `withTimeout`.
13. [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md) — the collector, value vs pointer, cleanup, raw pointers, `extern "C"`.
14. [Attributes and the test runner](14-attributes-and-testing.md) — `@test`, `@deprecated`, `@mustUse`, `@inline`.
16. [Networking](16-networking.md) — TCP with `net`: one task per connection, `readLine`/`write`, timeouts.
17. [An HTTP server](17-http.md) — `http`: router, handlers and what their errors mean, static files, keep-alive.

## Reference

- [Cheat sheet](reference/cheatsheet.md) — the whole syntax on one page.
- [Standard library](reference/stdlib.md) — what the bootstrap prelude and `io` provide today.
- [Common error messages](reference/errors.md) — what they mean and how to fix them.
- The language design itself is in [`veles-spec.md`](../veles-spec.md); tutorials cite its decisions as `D<n>`.

## Conventions

Code blocks are complete programs unless they begin with `// fragment`.
Where a program prints something, the output follows the block. Run any
program by saving it as `main.vs` in an empty directory and typing
`veles run <that directory>`.
