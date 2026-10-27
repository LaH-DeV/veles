# The Veles programming language

Bootstrap compiler for Veles, written in Go, implementing `veles-spec.md`
(v0.90, D1–D176): the whole surface syntax, the type system with inferred
effects, compile-time evaluation (`const fun`), a non-moving collector with
concurrent marking on Windows, a multi-threaded executor for stackless
coroutines with declared executors and OS-thread control, packages (path,
git and a registry, with `veles.sum` and capability policies), C interop
with the C ABI for structs by value, and a test runner.
This-hosting has begun: the lexer and parser are rewritten in Veles
(`selfhost/`) and agree with the Go front end byte for byte; the order of
what follows is track S of `veles-plan.md`.

```bash
go build -o veles.exe .
./veles.exe run examples/hello                      # compile and run a module directory
./veles.exe build examples/calc -o calc              # produce an executable
./veles.exe test examples/features/testing           # run the tests (D78; --filter text, --timeout 30s)
./veles.exe check examples/features/errors           # type-check only (--fix applies lint corrections)
./veles.exe parse examples/features/syntax_tour.vs   # dump the syntax tree
./veles.exe build examples/hello --emit-llvm         # write the LLVM IR instead of linking
./veles.exe fmt examples std                         # format sources in place (--check lists, --stdout prints)
./veles.exe lsp                                      # language server over stdio (see editors/vscode)
```

New to the language? Start with [docs/documentation/index.md](docs/documentation/index.md): twenty-five
tutorial chapters for three levels of reader plus a cheat sheet, a
standard-library reference and a guide to error messages. Every program in
them is compiled and run by `go test ./docs/`.

Requirements: Go 1.27.1 and `clang` (LLVM 17+) on `PATH`, or `VELES_CLANG`.
The Go version is pinned for every machine: `go.mod` asks for 1.27.1, so an
older Go (1.21 or newer) downloads and uses it, and the test scripts set
`GOTOOLCHAIN=go1.27.1`. The generated Unicode tables (`selfhost/lexer`,
`std/regex`) come from that Go's Unicode data (17.0.0), and their freshness
tests fail under a Go with other data; move the pin and regenerate them
together (`go test ./selfhost -run UnicodeTables -update`,
`go test ./std -run RegexUnicodeTables -update-unicode`).
`--release` disables overflow checks (D21) and optimises; the default is a
checked debug build. `run` and `test` build into a temporary directory; only
`build` leaves an executable behind. The C runtime is compiled once per
compiler/clang/flag combination and cached under the user cache directory
(`%LOCALAPPDATA%\veles\rt` on Windows). `VELES_GC_TRACE=1` reports collections;
`VELES_GC_THRESHOLD=<bytes>` lowers the collection trigger for testing,
`VELES_GC_MARKERS=<n>` sets how many threads mark and sweep, and
`VELES_GC_CONCURRENT=0|1|all` chooses whether marking runs alongside the
program (docs chapter 13).

### macOS

Apple Silicon is supported and tested with Go 1.27.1 and Apple clang 21.
Install Xcode Command Line Tools (`xcode-select --install`) and Go. TLS
uses Homebrew OpenSSL 3 (`brew install openssl@3`); the runtime loads it
from `/opt/homebrew/lib` or `/usr/local/lib`, without extra linker flags.

```bash
GOTOOLCHAIN=go1.27.1 go build -o veles .
./veles run examples/hello
bash internal/macos-test.sh
```

The script runs build, vet and every test with Go 1.27.1 by default, or a
supplied command, such as `bash internal/macos-test.sh go test ./examples -sanitize`.
`GOTOOLCHAIN` may override that version, but Go releases with different
Unicode data fail the generated-table freshness checks until the tables are
updated together. The runtime cache is under `~/Library/Caches/veles/rt`.
Sockets use kqueue; Intel Macs are not yet validated. Thread names work;
non-default priorities and CPU affinity are refused rather than ignored.

## Layout

| Package           | Role                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `source/`         | files, spans, diagnostics with source excerpts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `lexer/`          | hand-written scanner; Go-style semicolon insertion (§4b), string interpolation                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `ast/`, `parser/` | the full spec grammar                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `types/`          | semantic types                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `sema/`           | modules and manifests (M1–M6), name resolution, type checking, effect inference (D2/D4/D45), lowering to a typed HIR                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `codegen/llvm/`   | textual LLVM IR emission (I1/I2); coroutines via `llvm.coro.*`; GC type descriptors; vtables                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `runtime/c/`      | `veles_rt.c` (strings, lists, maps), `veles_gc.c` (collector), `veles_task.c` (executor), `veles_poll.c` (the socket reactor: epoll, AFD/IOCP, kqueue, `poll()`), `veles_os.c` (files, bytes, processes, environment, clocks)                                                                                                                                                                                                                                                                                                                                                                                                |
| `std/`            | standard library in Veles, embedded in the compiler: `prelude` (D24: `Iterator`/`Iterable` and adapters, `extend` blocks for `string`, `List`, `Range`, the operator traits `Comparable`/`Equatable`/`Hashable`/`Display`, `Closeable`, `Mutex`/`Atomic`/`RwLock`, executors, `Panic`, `IoError`, `StringBuilder`, `Secret`, `Duration`), `io`, `os`, `fs`, `path`, `time`, `random`, `net`, `tls`, `http` (server and client), `json`, `codec`, `config`, `log`, `otel`, `db` (PostgreSQL), `compress`, `crypto`, `hex`, `base64`, `jwt`, `utf8`, `ffi`, `recursion`, `lint` (the API of lint rules written in Veles, D177) |
| `format/`         | the formatter (`veles fmt`, and `textDocument/formatting` in the LSP): prettier-style — blocks always break, columns align, comments and the author's list/chain line breaks are kept                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `fetch/`          | the package system's network side (D138, D139): resolves registry and git dependencies by minimal version selection, the module cache and `veles.sum`, vendoring, capabilities computed from source and the `[policy]` that refuses them, signed reviews, publishing; the only code that runs git or touches the network (the compiler core reads directories through `sema.ResolveRemote`)                                                                                                                                                                                                                                  |
| `registry/`       | a reference implementation of the registry protocol, in memory, for tests and small private registries                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `lsp/`            | language server: diagnostics with quick fixes, hover, definition, type definition, implementations, folding, references, rename (re-checked before it applies), highlights, inlay hints, signature help, symbols, completion, formatting over the compiler front end                                                                                                                                                                                                                                                                                                                                                         |
| `editors/vscode/` | VS Code extension: TextMate grammar and client for `veles lsp`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `docs/`           | tutorials and reference; `go test ./docs/` runs every code block                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `selfhost/`       | the front end rewritten in Veles (veles-selfhost-frontend-plan.md), phase by phase; `go test ./selfhost` builds it and checks its output against the Go front end's, byte for byte, on every Veles file in the repository                                                                                                                                                                                                                                                                                                                                                                                                    |
| `examples/`       | programs with expected output, real tools at the top and one short program per language feature in `features/`; `go test ./...` compiles and runs them (a `commands.txt` scripts a command-line tool, `-update` rewrites `expected.txt`; a `name.vss` script is a test when `name.expected.txt` exists, fed `name.stdin.txt` when present)                                                                                                                                                                                                                                                                                                                                                                 |

`go test ./...` is the regression suite. The parser, formatter and checker also
have native fuzz targets — `go test -fuzz=FuzzParse ./parser`, `-fuzz=FuzzFormat
./format`, `-fuzz=FuzzCheck ./sema` — seeded with every program in the
repository; inputs that once failed live under each package's `testdata/fuzz`
and run with the ordinary tests.

## Spec coverage

**Semantics.** D1 tracing GC (non-moving mark-sweep, size-class spans,
conservative stack and precise heap scanning — I3/§5). D2 coloring-free
suspension inferred over the call graph and compiled to stackless
coroutines; `await` only on primitives that always suspend (D16). D3/D34/D36/D38
`scope` (fail-fast, cancellation at suspension points), `gather` (tuple of
`Result<T, E | Panic>`), `race`, channels, timers. D4/D45 `throws` as
`Result` sugar, `throw e`, inferred error unions, `error Name { }` declarations
with `message()` dispatching on unions, `r.ok`/`r.err` and `is Ok` smart casts to the payload. D5 nesting `T?` with smart casts,
`?.`, `?:`. D6/D9/D17/D26/D53 traits, trait objects with vtables, global
coherence, default bodies, the prelude operator traits (`Comparable`, `Equatable`,
`Hashable`, `Display` replace structural ordering, equality, hashing and text; `Parsable` for parsing). D7/D10/D39 value structs, `&x` with heap
promotion, auto-deref. D8/D15 generics by stenciling. D11/D22 `val`/`var`
bindings, `var` fields. D12/D13 sealed traits as inline tagged unions, `when` with
destructuring, guards, exhaustiveness, methods on sealed traits dispatched
by tag. D57 enums (closed sets of named integer values: `enum Phase : u8 { Red = 1, Amber }`, exhaustive `when`, `toString`/`values`/`fromValue`/`parse` for free; `compareTo` returns the prelude `Ordering`). D18/D19 byte-indexed UTF-8 strings. D20/D52 panics unwind to the
task scope; `panic(msg)`. D21 checked/wrapping arithmetic, bitwise operators. D23 methods in struct bodies,
`implement` blocks (also inline in the body of your own types), `static fun`, `extend` blocks (the prelude adds the string, list and range methods in Veles). D25/D41 `List`/`Map`/`Set` with the immutable/mutable split,
insertion-ordered maps, literals typed by context (`mut [...]` only for untyped ones); element access is methods only (`at`/`set`, `get`/`set`; `at` is a `T` where the index is proven, D62), brackets are literals. D27/D42/D46 associated types, `loop (x in c)`
through `Iterable`, lazy adapters. D28 named arguments, defaults and variadic parameters. D29
ranges. D31 infinite-size diagnostic. D32/D33/D37 lambdas, `=>`, tuples
with tupling conversion. D35/D54 `Sendable` derivation, `Mutex`/`Atomic`.
D40 declared effects on trait methods and function types. D43/D47 `with`
cleanup on every exit path. D44/D50 `unsafe`, `*raw T`, `extern "C"`. D48
comparator/key lambdas. D51 `@deprecated`, `@mustUse`, `@inline`,
`@noinline`; D78 `test "..." { }` with `expect`/`require`/`check`.

**Modules and packages.** M1 `veles.toml` identity, M2/M3 directory
modules, M4 cycle detection, M5 visibility (D142: `private`, `internal`, package-wide unmarked, `public` for other packages) plus the root module's `public use` re-exports (D89), M6 logical
paths; dependencies by path, git (tags or a commit) and a registry, minimal version selection per
major, workspaces, `veles.sum`, vendoring, capabilities and a `[policy]` (D138, D139; chapter 25).

## Known gaps and deviations

- The collector marks concurrently on Windows only; on Linux and macOS marking is in
  the pause (checklist §11).
- No debug info yet: a debugger cannot step Veles source (plan S8).
- Panics unwind via `setjmp`/`longjmp` to the executor rather than D49's
  DWARF tables; the task's active `with` cleanups run on the way.
- Exhaustiveness is a variant-set check; nested refutable sub-patterns are
  only tracked for nullables.
- Trait objects require object-safe traits (no associated types, generic
  methods, `This` in signatures, or suspending methods).
- Inside `unsafe`, `*T` converts implicitly to `*raw T` (needed to bind the
  runtime from Veles).
