# The Veles programming language

Bootstrap compiler for Veles, written in Go, following `veles-spec.md`
(v0.21) and `veles-build-plan.md`. It is at **build-plan Stage 1 with most
of Stage 2**: the whole surface syntax parses, a substantial core of the
language type-checks and compiles to native code through LLVM IR and clang.

```bash
go build -o veles.exe .
./veles.exe run examples/hello          # compile and run a module directory
./veles.exe build examples/shapes -o shapes.exe
./veles.exe check examples/errors       # type-check only
./veles.exe parse examples/syntax_tour.vs   # dump the syntax tree
./veles.exe build examples/hello --emit-llvm   # write hello.ll instead of linking
```

Requirements: Go 1.23 and `clang` (LLVM 17+) on `PATH`, or set `VELES_CLANG`.
`--release` disables overflow checks (D21); the default is a checked debug build.

## Layout

| Package | Role |
|---|---|
| `source/` | files, spans, diagnostics with source excerpts |
| `lexer/` | hand-written scanner; Go-style automatic semicolon insertion (§4b), string interpolation |
| `ast/`, `parser/` | the full spec grammar: structs, traits, impls, sealed variants, `when` patterns, lambdas, effects, `scope`/`gather`/`race`/`with`/`unsafe`, `extern "C"`, attributes, tuples, ranges |
| `types/` | semantic types (primitives, `*T`, `*raw T`, `T?`, tuples, `List`, sealed, structs, traits, error unions) |
| `sema/` | module loading (a module is a directory, M2/M3), name resolution, type checking, effect inference, lowering to a typed HIR |
| `codegen/llvm/` | textual LLVM IR emission (I1/I2) |
| `runtime/c/veles_rt.c` | C runtime linked into every program (no GC yet — leak everything, per Stage 1) |
| `std/` | standard library sources embedded in the compiler (`io`) |
| `examples/` | programs with expected output; `go test ./...` compiles and runs them |

A package root is the nearest ancestor directory containing `veles.toml`, or
the entry directory itself. `use math.geometry` resolves to the directory
`math/geometry` under the root; names not found locally resolve to the
standard library.

## Implemented

- Primitives with explicit widths, `bool`, `string` (immutable UTF-8, byte `len()`, D18/D19), `val`/`var`/`const`, `if`/`else` as expressions, `loop` in all three forms with labels, `break`/`continue`/`return` as expressions
- Functions with named arguments and defaults (D28), expression bodies with inferred return types, generics by per-instantiation stenciling (D8/D15), trait bounds
- Structs with defaults and the implicit constructor, inherent methods, `mut fun` and the `val` rule (D22), `&x` with heap promotion (D10), `.` auto-deref (D39)
- `T?` as genuine `Option<T>` that nests (`i32??`), implicit promotion, flow-sensitive smart casts including assignment and branch merging, `?.`, `?:` (D5/D30)
- Sealed traits laid out as inline tagged unions, variant construction, `when` with type/destructuring/literal/range/tuple patterns, guards, exhaustiveness (variant-set coverage; guarded arms never count), the D13 `else`-on-sealed lint
- `throws` as sugar over `Result<T, E>` with error-in-register ABI, `try`, `Err(...)` to fail, inferred error unions across call chains to a fixpoint, width subtyping, `when` over unions, `Result` values matched with `Ok`/`Err` (D4/D45)
- Open traits with default bodies and `override` (D53), `impl Trait for Type` including foreign types, global coherence (D17), parameter-name inheritance (D28), ambiguity as an error (D26), declared effects on trait methods (D40)
- Modules as directories, per-file `use` with aliases and `{ }` item lists, `pub` (M5), import cycle detection (M4)
- `List<T>` / `MutableList<T>` with the immutable/mutable split (D25), literals, index, `push`/`pop`/`len`/`toList`/`toMutable`; ranges `..`/`..<` (D29); tuples with destructuring (D37)
- `+%`/`-%`/`*%` wrapping operators; `+` checked in debug builds, wrapping in release (D21); bounds and division panics (D20)
- `unsafe` blocks, `unsafe fun`, `extern "C"` declarations, `*raw T` (D44/D50); string interpolation with generated formatting for every type

## Not yet implemented (reported as errors, not silently accepted)

- Concurrency: `async`, `await`, `scope`, `gather`, `race`, `suspends` (build-plan Stage 4)
- Lambdas/closures (D32) and function values; `with` blocks (D43)
- `Map`/`Set`; `Iterable`/`Iterator`-driven `loop` (D42); associated types; trait objects (D9); supertraits
- Garbage collection (Stage 3), the package manager and `veles.sum` (Stage 5)

## Deviations to be aware of

- Inside `unsafe`, a `*T` converts implicitly to `*raw T` so the runtime can be
  bound from Veles; the collector is non-moving so the address is stable.
- Extern functions pass `string` as a `(ptr, len)` pair and may not return
  aggregates; use out-pointers.
- Exhaustiveness is a variant-set check, not full pattern-usefulness
  analysis, so nested refutable sub-patterns are only tracked for nullables.
