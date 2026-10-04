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

When you are ready for tasks: [Concurrency, explained from scratch](concurrency-explained.md) — threads, tasks, coroutines and `await` in plain words, compared with JavaScript.

## Track 2 — Coming from Go, Kotlin, Swift, TypeScript or Java

Skim Track 1 (the syntax is close to Kotlin's), then:

7. [Errors: `throws`, `throw`, `try`](07-errors.md) — errors are values, without the ceremony.
8. [Traits and generics](08-traits-and-generics.md) — interfaces without inheritance, stenciled generics, trait objects, the operator traits.
9. [Sealed types, enums and `when`](09-sealed-types.md) — algebraic data types and closed sets of values, with exhaustive matching.
10. [Closures and iterators](10-closures-and-iterators.md) — lambdas, captures, lazy pipelines, `Iterable`.
11. [Modules, packages and tests](11-modules-and-packages.md) — directories as modules, `veles.toml`, tests.
15. [Files, paths and processes](15-files-and-processes.md) — `fs`, `path`, `os`, `StringBuilder`, `IoError`, `time`, `random`.

## Track 3 — Systems and concurrency

For readers who want to know what the compiler actually does.

12. [Concurrency](12-concurrency.md) — inferred suspension, `async`/`await`, `scope`, `gather`, `race`, channels, cancellation, `withTimeout`, task-local values.
13. [Memory, `with`, `unsafe` and C](13-memory-and-ffi.md) — the collector, value vs pointer, cleanup, raw pointers, `extern "C"`.
14. [Attributes and tests](14-attributes-and-testing.md) — `@deprecated`, `@mustUse`, `@inline`; `test "..." { }`, `expect`, `require`, `check`.
16. [Networking](16-networking.md) — TCP with `net`: one task per connection, `readLine`/`write`, timeouts.
17. [An HTTP server](17-http.md) — `http`: router, handlers and what their errors mean, static files, keep-alive.
18. [Codable and JSON](18-codable-and-json.md) — `implement Codable` derives the wire code; `json` reads and writes it, reporting every problem with its path.
19. [Secrets, hashes and tokens](19-secrets-and-crypto.md) — `crypto`, `hex`, `base64` and `jwt`: hashing, MACs, random bytes, UUIDs and signed tokens, and what each of them refuses.
20. [Time](20-time.md) — `Duration` in the prelude; `time`: the wall clock and the monotonic one as different types, the calendar, RFC 3339 and HTTP dates.
21. [Logging](21-logging.md) — `log`: four levels, named fields, text on a terminal and JSON elsewhere, messages that cost nothing while off, fields for a whole request.
22. [Configuration](22-configuration.md) — `config`: settings read into a struct from the environment, `.env` and JSON files, every problem at once, secrets never echoed, `describe` for `--help`.
23. [Observability](23-observability.md) — `otel`: traces, metrics and logs in OpenTelemetry's format; spans that follow tasks and cross services by `traceparent`; OTLP export.

## Reference

- [Cheat sheet](reference/cheatsheet.md) — the whole syntax on one page.
- [Standard library](reference/stdlib.md) — what the bootstrap prelude and `io` provide today.
- [Error messages](reference/errors.md) — every family of error the compiler reports, what it means and how to fix it (`veles explain <family>`).
- The language design itself is in [`veles-spec.md`](../../veles-spec.md); tutorials cite its decisions as `D<n>`.

## Conventions

Code blocks are complete programs unless they begin with `// fragment`.
Where a program prints something, the output follows the block. Run any
program by saving it as `main.vs` in an empty directory and typing
`veles run <that directory>`.
