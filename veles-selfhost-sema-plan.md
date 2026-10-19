# Veles — rewriting `sema` in Veles

Plan track S11 (2026-10-19). Written after `veles check --dump` (S10, first
cut), the bootstrap harness (S14, `internal/bootstrap`) and the lexer/parser
port (E4, G1–G4 green); it is the sequencing document for the next, much bigger
slice, in the manner of `veles-selfhost-frontend-plan.md`. It is a plan with
gates, not a schedule, and it decides nothing about the language: the places
where it needs a decision are listed in §8 and go through `veles-decide`.

---

## 0. The cut

**In scope.** `types/` and `sema/` — the checker: loading a package, scopes,
declarations, type resolution, bodies, generics, traits, effects, the
compile-time evaluator, derived code, lowering to HIR — and the diagnostics it
reports, word for word.

**Out of scope for this plan.** `codegen/llvm` (the next slice, with the codegen
goldens as its oracle), `driver`, `lsp`, `format` (slice S9), and `runtime/c`,
which stays C. The package index that the language server reads
(`sema/index.go`, `suggest.go`, `explain.go`, 2 600 lines) follows `lsp`.

**Not a replacement, as with the front end.** At the end of every phase the Go
`sema` is still what `veles build` uses. The Veles one is a second
implementation that provably agrees with it, phase by phase, until the
bootstrap harness (S14) can put the whole compiler through its fixpoint.

---

## 1. What `sema` is, by the numbers (2026-10-19)

`sema/` is 98 source files and 46 900 lines of Go, plus `types/` (1 450):

| Component | Files (main ones) | Lines | What it does |
|---|---|---:|---|
| Package loading | `loader.go`, `manifest.go`, `toml.go` | 2 100 | modules, imports, `veles.toml`, dependencies, per-target files |
| Checker core | `checker.go`, `scope.go`, universe | 3 600 | the passes (below), scopes, symbols, diagnostics with dedup |
| Expressions and bodies | `check_expr.go`, `check_call.go`, `check_pattern.go`, `check_fn.go`, `check_lambda.go`, `check_map.go`, `check_guard.go` | 9 500 | types of every expression, calls and generics inference, patterns, narrowing, lowering to HIR |
| Tasks and effects | `check_task.go`, `suspend.go`, `resources.go`, `taskheld.go`, `receivers.go`, `move.go`, `awaited*.go` | 3 400 | `async`/`await`/`race`/`select`, suspension inference, `with` resources, held regions, receiver passes |
| Derived code and reflection | `derive*.go`, `annotation.go`, `reflect.go`, `attrs.go`, `template.go` | 3 200 | `implement Codable`, annotations and decorators, `This`, template literals, regex literals |
| Compile-time evaluator | `consteval.go`, `constfun.go`, `constnum.go`, `constgeneric.go`, `staticassert.go` | 2 800 | `const fun`, constants, `static assert`, lint rules written in Veles |
| Lints and fixes | `lint*.go`, `lints.go`, `lintrules.go`, `fixes.go` | 3 300 | the lint registry, levels, each lint, automatic corrections |
| HIR | `hir.go` | 900 | the output: ~75 node types |
| Index, hover, suggestions | `index.go`, `suggest.go`, `explain.go` | 2 650 | what the language server reads |
| `types/` | `types.go`, `layout.go` | 1 450 | the type representation, keys, C layout |

The passes, in the order `checkCollect` runs them:

1. **load** (`LoadPackage`): files, modules, imports, the manifest.
2. **universe + collect**: the built-in types, every declaration of every
   module put into its scope; nothing is resolved yet.
3. **rounds** (up to six): every body checked, generics instantiated on demand,
   inferred error sets and suspension grown until nothing changes (D4/D45) —
   the effect fixpoint.
4. **whole-program passes**: global initialisation order, receivers, suspension,
   held regions, inferred signatures for the index.
5. **lints and levels**, then the HIR (`Program`) goes to `codegen`.

The sizes that matter for the schedule: expressions and bodies (9 500) are a
quarter of it, and the checker core, tasks/effects and derive/reflection
another 10 000 that are entangled with it. The compile-time evaluator alone
(2 800) is the whole of an interpreter over the HIR.

---

## 2. What the oracles are

The front end had a free oracle (`veles parse`). `sema` has several, each
exact, none free; the plan is to use all of them.

| Oracle | Compares | Exists |
|---|---|---|
| `veles check --dump names\|types\|hir` (S10) | the symbols, signatures with effects, and every lowered body with its types, as stable text | first cut 2026-10-19 (after lowering only) |
| the conformance suite | every diagnostic's exact text and place: 110 cases in `sema/testdata/conform`, grouped by decision, `// error:` expectations | A4 |
| `sema` unit tests | 178 test functions: rules, fixes, lints, hover | yes |
| the codegen goldens | the IR of the fixture programs | yes |
| the examples | the output of 72 programs, and their IR | yes |
| `driver/constfun_test.go` | the compile-time evaluator against run time, debug and release | yes |
| the bootstrap harness (S14) | the IR a stage emits for the compiler's own source | harness built, gate waits on this plan |

**Diagnostics are part of the contract.** The checker has 904 reporting sites.
Their text, their order and which one wins after an earlier error are what a
user reads, and what the conformance suite pins. A Veles `sema` reports the
same set in the same order — including the suppression of a second error that
is a consequence of the first. This is the most expensive kind of equivalence
here, and the reason the port goes by phases with per-phase dumps.

### 2.1 What the dumps need before the port starts

S10's first cut dumps the program **after** lowering. A phase gate needs the
state between phases. Before R2 starts, extend it (decision-free, S10b):

- `--dump=modules`: the module graph — each module's path, its files, its
  imports and what each import resolved to, the per-target files chosen;
- `--dump=scopes`: for every module, the declarations of its scope with kind,
  visibility level and where declared, **before any body is checked**;
- `--dump=decls`: the resolved declarations — struct fields and layouts, enum
  members, trait members, impls and extensions, generic parameters and
  bounds — without bodies;
- `--dump=effects`: the inferred `throws` set and suspension of each function
  after the fixpoint, with the number of rounds it took;
- spans in every dump (a flag), once the Veles checker can be compared on
  positions.

Each is a text rendering of data the Go checker already holds; none changes
what it does.

---

## 3. The phases

Each phase ends with a gate. A gate is a command that runs the Veles checker
over a corpus and compares its output with the Go checker's, byte for byte.
The corpus is `std/`, `examples/`, `selfhost/` itself, the conformance cases and
the fixtures of the `sema` and `driver` tests. A phase does not start until the
one before is green. The order follows what depends on what.

| Phase | Port | Lines of Go | Gate |
|---|---|---:|---|
| **R0** | the harness: `selfhost/sema` module, a `--check` mode in `selfhost/main.vs`, the comparer in `selfhost_test.go`, the corpus lists | — | trivial files agree |
| **R1** | package loading: `toml`, `manifest`, `loader` (modules, imports, per-target files, dependencies) | 2 100 | `--dump=modules` equal on the corpus and on the `driver` package fixtures; every manifest and import error text in the conformance suite |
| **R2** | `types/` and the type representation; the universe; scopes; **collect** | 1 450 + ~2 000 | `--dump=scopes` equal; visibility errors (`private-to-module` family) equal |
| **R3** | declarations: struct/enum/trait/sealed/impl/extend resolution, generic parameters and bounds, instantiation, C layout, associated types, type aliases | ~4 000 | `--dump=decls` equal; the `declarations`, `traits`, `sealed-types`, `generics` families |
| **R4** | bodies: expressions, calls and inference, patterns, narrowing and bounds facts, lambdas, strings and formats, `when` exhaustiveness, lowering each construct to HIR | ~12 000 | `--dump=hir` equal on every body of the corpus; the conformance suite's diagnostics for all families but tasks, constants and tests |
| **R5** | effects: errors and suspension to a fixpoint, tasks and `race`/`select`, `with` resources, held regions, receivers, move/ownership, global order | ~4 000 | `--dump=effects` equal, rounds included; the `tasks`, `resources`, `results-and-errors`, `unused` families |
| **R6** | derived code, annotations and decorators, reflection and `inline loop`, templates, executors, test words | ~3 200 | `veles explain --derive` equal on every `implement`; `--dump=hir` of the generated bodies |
| **R7** | the compile-time evaluator, `static assert`, `const fun`, regex literal validation, lint rules written in Veles | ~2 800 | `driver/constfun_test.go` and the const-function list (`std/testdata/const-functions.txt`) through the Veles evaluator; the `constants` family |
| **R8** | lints and fixes | ~3 300 | `veles check --fix` output equal; the lint registry listing equal |
| **R9** | the package index for the language server | ~2 650 | together with `lsp` |

R1–R3 are the foundation and can each be gated alone. **R4 is the port.** It
is the only phase whose size is a risk in itself; it is split by construct
(literals and operators → names and members → calls and generics → patterns →
control flow → lambdas) and each split is gated by the HIR dump of the bodies
that use only what is done, which the corpus has plenty of.

The order differs from the Go checker's file layout in one place on purpose:
the compile-time evaluator (R7) comes after the bodies, although a body can
need a constant. Until R7 a Veles checker reports "needs the evaluator" for a
construct it cannot fold, and the gate excludes those bodies; the number of
excluded bodies is printed, and it must reach zero at R7.

### 3.1 The first slice, concretely (R0 + R1)

1. S10b: `--dump=modules` in the Go checker, golden-tested like the others.
2. `selfhost/sema/toml.vs`, `manifest.vs`: the TOML subset and the manifest
   schema (D131/D138) — small, self-contained, and the first code in the
   compiler that reads a user's configuration.
3. `selfhost/sema/loader.vs`: module discovery, `use` resolution, the
   per-target file rule (D160), the standard library from the embedded tree.
4. `selfhost/main.vs --modules DIR`, and the Go test that compares it with
   `veles check --dump=modules` over every directory in the corpus.

That is about 2 100 lines of Veles, a small piece, and it already
exercises file I/O, maps, error reporting with spans and the diagnostics
renderer — everything the later phases lean on.

---

## 4. What the port will stress

From the capability audit (`veles-selfhost-frontend-plan.md` §9), the front
end and the benchmarks, the language and the runtime already have what `sema`
needs. What is new at this size, and must be watched, not assumed:

- **Type identity.** Types are compared and used as map keys all over the
  checker (`types.Key`, instances keyed by their arguments). A Veles `sema`
  needs a type family that is a sealed hierarchy with structural equality and
  hashing, as `Type` is in Go, and interned instances so that two
  instantiations of `List<i64>` are the same value. `bench/intern` and the map
  keys of the audit say this works; R2 measures it.
- **Mutual recursion and shared mutable state.** `Checker` is one struct with
  about forty maps and a few hundred methods across 70 files. In Veles a type's
  methods live in its body or in `extend` blocks of its own package (private
  fields stay reachable from both), so one large struct with its methods spread
  over many files is possible today; the alternative is a set of smaller state
  types passed around. **This is a design question the port should settle in
  R2**, before R4 multiplies it (§8, Q-S2).
- **Callbacks.** `types.Hooks` (instantiators, associated-type resolution) are
  function values the checker hands to the type package; the port keeps the
  shape or removes the cycle by putting both in one package.
- **Compile time of the compiler.** A 50 000-line Veles program is checked by
  the Go compiler today; the measured cost (`--timings`, S12 interface files)
  decides how comfortable the port is to work on. `bench/parse` says the Veles
  parser is 2.4× Go's over std (S7); `sema` at that ratio would check std in
  seconds, not milliseconds. Each phase's gate prints the time next to Go's.
- **Memory of a whole-program HIR under the collector.** The audit left this
  as an E4 measurement; R4 is where it is first real. Pause times are already
  measured to be low on Windows (F18); Linux marks in the pause (§11 of the
  checklist), which a 2-million-node HIR will feel.
- **Recursion depth.** Nested expressions, patterns and types recurse in the
  checker as in the parser; the runtime's stack is 256 MB (D92) and
  `recursion.Depth` is the answer for hostile input, as in the front end.
- **Diagnostics text.** 904 messages built with `fmt.Sprintf` verbs. Veles has
  string interpolation and `std/fmt`; the port writes the messages as
  interpolated strings. A table of the messages (the family patterns of
  `source/family.go` already enumerate them) is generated for the port to
  check against, so a message cannot be forgotten.
- **Stable iteration.** The Go checker iterates maps in sorted order wherever
  the output depends on it (the dumps rely on it). Veles maps are insertion
  ordered by guarantee (D25), which is the same property for free — except
  where Go sorts by *name* and the Veles insertion order is the declaration
  order. Each place is a gate failure to look at, not a surprise.

---

## 5. What the port is not allowed to do

The rules of the front-end port (its §11) stand, and sharpen:

- **The behaviour is Go's, byte for byte; the code is Veles.** `bool` and `T?`
  results with `?:`, `else` and `if (val x = …)`, no `Result` where a nullable
  is the question, sealed families walked by `when`, `&value` boxing, current
  spellings, `veles fmt` applied, names are words.
- **No transliteration of Go's idioms:** no `(T, bool)` pairs, no
  `interface{}`-shaped any-typed maps — a node kind is a sealed variant — no
  `reflect`. The Go checker uses Go's `reflect` in two places (`dump.go`
  and the host that shows syntax to lint rules, `lintrules.go`); the Veles one
  uses the language's own annotation and reflection mechanism (D174) where it
  needs the same, and writes the dump by hand or by a derived trait.
- **No comment names Go.** The comparers (`selfhost_test.go`, Go) are the only
  place that mentions the bootstrap compiler.
- **Every phase leaves the tree green:** `go build ./... && go vet ./... &&
  go test ./...` and the Linux run; the new gate is one more test.
- **Whatever a phase cuts is recorded** in checklist §11 in the same change.

---

## 6. Cost, honestly

| Phase group | Veles lines (1.2–1.5× the Go) | Notes |
|---|---:|---|
| R1–R3 | ~10 000 | foundation; small enough to finish before the schedule is believed |
| R4 | ~15 000 | the port; needs the checker-state design settled |
| R5–R6 | ~9 000 | the entangled passes |
| R7–R8 | ~8 000 | evaluator and lints |
| R9 | ~3 500 | with `lsp` |
| **sema in total** | **~45 000–55 000** | for 46 900 lines of Go |

The front end was about a seventh of this (6 350 lines of Go) and ended up 2.4× slower
than Go's; the same ratio over `sema` is acceptable for a bootstrap compiler
and not for a daily driver, which is why S7 (front-end speed as a language
finding) and S12 (interface files, so the compiler is not rechecked whole)
come first.

---

## 7. What has to be done before R1 starts

All decision-free, in this order:

1. **S10b**: the per-phase dumps of §2.1 (`modules`, `scopes`, `decls`,
   `effects`, spans).
2. **S6** finished: `Layout`, spans in the parse dump, non-UTF-8 files — so the
   front end the checker sits on is complete.
3. **S7**: the parser's speed, as a finding about the language and runtime.
4. **S8**: debug info, because nobody writes 45 000 lines they cannot step
   through.
5. **S12**: module interface files, so a build of the compiler is not a full
   recheck.

None of them blocks writing the TOML and manifest code (R1's first two steps),
which can start in parallel with 2–4.

---

## 8. Decisions this plan needs (to ask, not to assume)

- **Q-S1 — a bridge between a Veles checker and the Go back end.** Until
  `codegen` is ported, a Veles `sema` ends in a dump, not in a program. A
  *readable* HIR format (the `hir` dump made complete: spans, every type in a
  parseable form, constants) would let the Go `codegen` read a Veles checker's
  output and so let the whole pipeline run with `sema` in Veles long before
  `codegen` is. The cost is a reader in Go and a stable, versioned HIR text;
  the gain is that every example and every driver test exercises the Veles
  checker as a real compiler stage. The alternative is to compare dumps only
  and keep `codegen` waiting. *Recommendation: build the bridge; it turns the
  gate from "equal text" into "the programs still run".*
- **Q-S2 — the shape of the checker's state in Veles.** One large `Checker`
  struct with its methods in `extend` blocks across the files (it works today
  and mirrors the Go code, at the price of one type that every file can reach
  into everything of), or a family of state types (`Scopes`, `Instances`,
  `Effects`, `Globals`) passed explicitly (more code to thread, much clearer
  about what a pass may touch, and the better fit for tasks if checking ever
  runs in parallel). Not a language decision unless R2 shows a missing
  feature. *To be prepared with real code from R2 before asking.*
- **Q-S3 — whether the HIR stays a tree of `struct`s or a flat arena.** The Go
  HIR is ~75 pointer-linked node types. A flat arena with indices would be
  faster under the collector and easier to serialise (Q-S1), at the price of a
  style that is less readable. *Measure with R4's first constructs.*
- **Q-S4 — error recovery policy.** The Go checker keeps checking after an
  error to report more than one, with a few explicit "do not report a
  consequence" rules. The port must copy them exactly; if any of them turns
  out to depend on the Go checker's map order or pointer identity, that is a
  finding, and it may need a decision about what the language's diagnostics
  promise.

---

## 9. Where this goes next

When S10b, S6 and S7 are green: R0 and R1 (§3.1), then R2 with the answer to
Q-S2 prepared from what the first 10 000 lines of Veles `sema` looked like
— not guessed beforehand. After R3 the plan is revisited with the measured
cost of the port so far, and Q-S1 is decided with a prototype in hand.
