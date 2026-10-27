# Veles — Execution Plan (reset 2026-09-28)

The order of work from here. Everything before this date — Phases 0–5 of
the previous plan (harness, codegen goldens, decision-free debt, three
batches of decisions, D65–D78, the threaded executor, FFI, the test
framework, the 2026-09-27 security pass) and its progress log — is done
and kept verbatim in `archive/progress-log-2026-09.md`.

How the files relate:

| File                              | Holds                                                                                                   |
| --------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `veles-spec.md`                   | language decisions (D/M numbers) — what a program means                                                 |
| `veles-checklist.md`              | production readiness per area; §9 the open decisions (Q-labels), §10 the decision log, §11 known limits |
| this file                         | the **order** of work, with acceptance; a progress log at the bottom                                    |
| `notes_to_change.txt`             | the user's open design notes and standing directions                                                    |
| `veles-selfhost-frontend-plan.md` | the Veles rewrite of the lexer/parser (track E4)                                                        |
| `archive/`                        | finished plans, decided design notes, old logs — read-only                                              |

A task is done when its acceptance holds, `go build ./... && go vet ./...
&& go test ./...` is green, docs say what changed, and the checklist is
ticked. A task that changes what a program means or a public std API waits
for its §9 decision (`veles-decide`).

## Principles (user, 2026-09-28)

1. **Every feature is complete**: every layer it touches (syntax, checker,
   codegen, runtime, std, formatter, LSP, docs, tests) and every platform,
   or it is not done.
2. **Performant**: measured against the Go reference when it touches the
   runtime, codegen or hot std code.
3. **Two levels**: manageable at the lower level (raw pointers, capacity,
   explicit resources, `unsafe` where it must be), while the high-level
   path is the shortest, safest spelling with the best developer experience.
4. **Towards self-hosting, slowly but surely**: not writing the Veles
   compiler yet, but every language and std decision is checked against
   "could the compiler be written with this?" (track S).

Order rule: **A** (the compiler is right on every platform) before **B**
(what every user touches daily) before **C** (std breadth, each API asked
first) and **E** (large items). **D** is asked in batches whenever a batch
is ready, so the user's answers are never the bottleneck.

### Build order for the 2026-09-30/10-01 decisions (D100–D135)

Every decision is recorded (checklist §9 is empty), so an agent can work down
this list without asking; each step is green on Windows and WSL before the
next. A step names what it needs; the reason for the order is in brackets.

1. ~~**A8**~~ **Done 2026-10-01.** C ABI for structs by value — refuse first, then the classifier
   [a crash today; B16 builds on it].
2. ~~**B10**~~ **Done 2026-10-01.** D100 `with` statement, `with t = async`, no escape [B12, B13 extend it].
3. ~~**B11**~~ **Done 2026-10-01.** D101–D106, D134 small consistencies [independent; touches everyday
   code; D134 before the client and `std/db` are written].
4. ~~**B12**~~ **Done 2026-10-02.** D107–D110 lock guard, `race` send arms, `with expr`, helpers
   (`time.ticker` lives in `std/time`) [needs B10].
5. ~~**B13**~~ **Done 2026-10-02.** D111, D112, D114, D115 task-holding values, `Secret`, unchecked
   access, never-closed warning [needs B10].
6. ~~**B15**~~ **Done 2026-10-02.** D116, D117, D119, D135 suspension follows the argument + B8, `is
Trait`, `Default`, the downcast [independent; shrinks the IR before the big
   std work; D135 before E2].
7. ~~**B14**~~ **Done 2026-10-03.** D113 parts 1–3 (constant expressions, tables, `static assert`)
   [B16's `Array<T, N>` needs constant lengths; E7 needs part 1–2].
8. ~~**B16**~~ **Done 2026-10-03.** D120–D123 C layout, `Array<T, N>`, variadic calls [needs A8, B14].
9. ~~**C2**~~ **Done 2026-10-04** (`tls.Conn` joins `io.Stream` with E1). `io.Stream` (D128: trait, `net.Conn`, `fs.File`, `http` over it) →
   `std/compress` + `http.compress()` (D124) → `http.testServer` (D130, needs
   B13) [compress streams over `Stream`].
10. ~~**C3**~~ **Done 2026-10-04.** `std/config` (D125) [needs B13's `Secret`].
11. ~~**C5/C6**~~ **Done 2026-10-04.** `std/fs` additions, endian bytes (D130), helpers (D110 if not in B12).
12. ~~**C8**~~ **Done 2026-10-04.** template literals (D129 part 1) [before E2].
13. ~~**C7**~~ **Done 2026-10-04.** the HTTP client over TCP (D127) [needs C2's `Stream`].
14. **C4** ~~health endpoints (D133)~~ **done 2026-10-04**, ~~`std/otel` (D126)~~ **done 2026-10-05** [OTLP export needs
    C7 and C2's gzip].
15. ~~**E1**~~ **Done 2026-10-05**, `http.serve(tls:)` included. TLS (D128) → HTTPS for the client and OTLP.
16. ~~**E2**~~ **Done 2026-10-05.** `std/db` (D129 part 2) + password hashing (D130) [needs C8, B13].
17. ~~**B14 part 4**~~ **Done 2026-10-05.** `const fun` [large; E7 needs it only if a manifest helper
    is a `const fun`].
18. ~~**E7**~~ **Done 2026-10-07.** packages (D131, D132, D138, D139): stages a–h; `package.vs` stays decided and unbuilt.

Then the rest of track E in its own order (reactor, self-hosted front end, GC,
release engineering, compiler speed). A7 (macOS arm64) is done 2026-10-21.

## Where to continue (2026-10-27)

Kept current: the first unchecked line is the next piece of work. Each is
decision-free unless it names a Q.

1. [x] **B32 step 6** — exact pointer maps for sealed payloads and
   over-aligned values inside tuples, nullables and sealed payloads
   (`veles-layout-spec.md` §5.4); then `noalias` on borrowed parameters
   after the stress runs pass with it (§5.3). _Done 2026-10-27._
2. [ ] **B30** D212 `const struct`, **B31** D213 allocation hints, **B29
   part 2** D210 `lazy val` — decided 2026-10-26, nothing built.
3. [ ] **F20** D197 `cede` (`veles-cede-spec.md`; `yieldNow()` migrated in
   the same change) and **F21** (measure D145's automatic yield).
4. [ ] **Self-hosting, front end → checker** (the way, in order):
   - the front end is done: lexer, parser, AST, spans and `Layout` agree
     with Go byte for byte (gates G1–G4, P6, `go test ./selfhost`);
   - **S9** the formatter in Veles (`selfhost/formatter` holds only its
     options): its oracle is free — `veles fmt` over the corpus;
   - **S13** one syntax tree for the compiler and lint rules (today three
     copies: `ast/`, `selfhost/ast`, generated `std/lint/ast.vs`);
   - **R1 steps 3–4** (`veles-selfhost-sema-plan.md` §3.1): the module
     loader in Veles and the `--modules` gate against `veles check
     --dump=modules` (TOML and the manifest, steps 1–2, are `std/toml` and
     `selfhost/pkg`);
   - **S12** module interface files, so building the compiler is not a
     full recheck; then **R2–R9**, each gated on its `--dump`.
5. [ ] The open findings of `veles-code-review-plan.md` (F2–F7: shared
   Unicode tables, `fmt.quote`, the lint AST, splitting `std/http`).
6. [ ] **Track P** from P2 (P1's `build.target` landed with C9; per-target
   files and the LSP per OS are what is left of P1).
7. Decisions waiting for the user (checklist §9): Q45, Q57, Q59, Q60, Q63, Q64.

**Where to continue (2026-10-14 audit, history):** every step above is done, and so are tracks C
and F. B17 (D142 visibility) and B20 (D158 bare members) were built 2026-10-17. Next, in
order: **B26** (D174 annotations, decided 2026-10-18, before C9), **B19** (D157, small, a correctness fix), **B21** (D163–D165),
**B22** (D166), the F19 migration left in checklist §11, then **S6–S8** (finish the front
end, its speed, debug info), **C9** (`std/cli`, before the driver is ported), then
**S9–S12**. Track P (D159–D162) runs beside S from P1. Checklist §9 is empty again.
_(2026-10-23: B26, B19, B21, B22, the F19 migration and S6–S8 are done but for D199's part of S6; D198 and D199 were decided the same day, and S16 (D198) is built. 2026-10-24: D200–D202 decided and built (S18–S20; bench/parse 0.7–0.9× Go), and S17 (D199) built; C9 built the same day (D167 names, D203, D204, `std/build`, `std/cli`) — next are **S9–S12**, with track P from P1.)_

---

## A — Correct everywhere (next)

| #   | Task                                                                                                                                                                                                      | Acceptance                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A1  | **Done 2026-09-28.** **Linux build and test** (notes R3), in WSL Ubuntu 24.04 (`wsl -d Ubuntu`; Go 1.23.2 in `~/.local/go`, clang 18 from apt); rerun with `internal/wsl-test.sh`                         | `go test ./...` green on Linux x86-64: the runtime compiles (incl. the never-compiled `posix_spawnp` branch of `os.run`), every example and doc block passes, the thread stress (`VELES_THREADS=1/2/4/8`) passes; differences fixed, not skipped. A `bench/results.md` run recorded from that machine                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| A2  | **Done 2026-09-28.** **Sanitizers over the examples** (`--sanitize`; `go test ./examples -sanitize`, `go test ./driver -sanitize` on Linux)                                                               | the runtime built with `-fsanitize=address,undefined` and a tiny `VELES_GC_THRESHOLD`, run over every example (a test flag or a script under `internal/`); clean, or each finding fixed with a test                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| A3  | **Done 2026-09-28.** **Verify D21 ships**: `+` checked in debug, wrapping in release, `+%` always; `as` between widths truncates only where written                                                       | a codegen golden per profile; checklist §1.4/§2 items ticked or turned into bugs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| A4  | **Done 2026-09-28** (every diagnostic has a case or the reason no program reaches it). **Conformance suite** (the old P6 item)                                                                            | `sema/testdata/conform/*.vs` with `// error: text` / `// warning: text` expectations, grouped by D-number, one negative case per diagnostic the checker emits; a test runs them all and lists diagnostics without a case                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| A5  | **Done 2026-09-28.** **Socket close vs a read in flight** (§11)                                                                                                                                           | a per-socket in-use count (Go's fdMutex idea) so a close on one thread waits for, or fails, I/O in progress on another; a threaded test that closes during reads                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| A6  | **Done 2026-09-28.** **Fuzz the HTTP request parser** (checklist §2)                                                                                                                                      | an in-process fuzz target over the request parsing `http.call` shares with `serve`; corpus kept; no panic, limits hold                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| A8  | **Done 2026-10-01** (the classifier landed in the same change, so the interim refusal was not needed). **C ABI for structs by value** (bug found 2026-10-01: `lldiv` segfaults on Windows; decision-free) | first, at once: a by-value `extern struct` in an `extern` signature, an `extern fun` type or an exported `extern "C" fun` is an error naming the pointer form (a footgun refused until it works); then a classifier per target — Win64 (≤ 8 bytes of size 1/2/4/8 in a register, else by hidden pointer / `sret`), SysV x86-64 (INTEGER/SSE eightbyte classes, ≤ 16 bytes in registers, else memory), AAPCS64 (HFA/HVA, ≤ 16 bytes in registers, else indirect) — applied to calls, `extern fun` pointer calls and exported functions, the error lifted; a C file in a driver test with functions taking and returning structs of each class (1–4 `i32`, two `i64`, `{u8, i64, u16}`, two `f64`, `{f32, f32, f32}`, 24 bytes) called both ways, on Windows and WSL (aarch64 cross-compiled only until A7) |
| A7  | **Done 2026-10-21.** **macOS arm64**, Apple clang 21 and Go 1.23.2; rerun with `internal/macos-test.sh`                                                                                                   | build, vet and every test green, C structs and varargs in debug/release, GC register capture and stack overflow exercised natively; kqueue reactor and poll fallback tested, 1/2/4/8-worker reactor runs, GC/sync examples each 20 runs per worker count with poison and a 256-byte threshold, every example and reactor/C ABI tests under ASan/UBSan; benchmarks recorded. Staticcheck, Intel Macs and Windows/Linux reruns remain unverified (checklist §11)                                                                                                                                                                                                                                                                                                                                            |

## B — Daily developer experience (decision-free unless marked)

| #   | Task                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Acceptance                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| B1  | **Done 2026-09-28.** **Every diagnostic names its fix and links its docs** (checklist §6; D79 named families + `veles explain`)                                                                                                                                                                                                                                                                                                                                   | audit of every `errorf`/`warnf` site; each message says what to write; an anchor per error family in `reference/errors.md`, printed as `see: <link>` and sent as LSP `codeDescription`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| B2  | **Done 2026-09-28.** `veles build --timings` (checklist §3.2)                                                                                                                                                                                                                                                                                                                                                                                                     | wall time per phase and per module; docs 01                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| B3  | **Done 2026-09-28** (D80). `veles test`: parallel tests (`--jobs`); helper tracing through `async helper()` (§11)                                                                                                                                                                                                                                                                                                                                                 | tests run as tasks across threads, reports stay in order; a test pins the order                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| B4  | **Done 2026-09-28** (D81). **Panic stack traces with symbol names** (checklist §4, D49)                                                                                                                                                                                                                                                                                                                                                                           | a panic in a task prints the Veles call chain, `at file:line` per frame; the crash-report format documented                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| B5  | **Done 2026-09-28** (D82). `os.run` with stdin and a separately captured stderr (checklist §5.10) — **public API, ask first**                                                                                                                                                                                                                                                                                                                                     | both platforms; docs 15                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| B6  | **Done 2026-09-28.** LSP leftovers                                                                                                                                                                                                                                                                                                                                                                                                                                | code action "implement missing trait members"; whatever the B1 audit shows missing                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| B7  | **Done 2026-09-28.** `veles new --template server` (checklist §6)                                                                                                                                                                                                                                                                                                                                                                                                 | logging, `/healthz`, graceful shutdown wired; runs and tests first try                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| B8  | **Done 2026-10-02 (with B15).** _(2026-10-01: unblocked by D116 — built as part of B15.)_ **Blocked 2026-09-28: not by speed (with `reserve` the prelude form is as fast at -O2) but by effects — the lowered adapters let a lambda throw or suspend (`xs.map(a => try eval(a))` is a `Result`); a prelude function would need to be generic over its argument's effects. See the progress log.** Move the Go-lowered eager adapters into the prelude (notes #14) | benchmark first (`veles-bench`); move only what is not slower at -O2                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| B17 | **Done 2026-10-17** (progress log: what differs from the acceptance text). **D142: visibility — unmarked is the package, `internal` the module, `public` other packages** (decided 2026-10-08; whole tree migrated in the same change)                                                                                                                                                                                                                            | sema: the four levels on top-level declarations, fields and methods; an import from another package resolves any module's `public` items by path (`use dep.mod`), `public use` facades work and may export unmarked items (only under the facade's name; `internal`/`private` refused); a `public` signature naming a type not visible outside the package is an error with both fixes; a field or method not visible from another package cannot be read, assigned or bound there, and the implicit constructor supplies/defaults it as M5 v0.30 does for `private`; errors: top-level `private` (fix `internal`), `public internal`, an item not `public` imported from another package (fix names `public`); warning + `--fix`: `public` in a program (root declares `main`); the loader stops treating the root as the only surface; LSP hover/completion follow the levels. Migration program (scratchpad, from the checker's view): unmarked → `internal`, `public` → unmarked unless reachable from another package; run over std, examples, bench, selfhost, docs programs, templates; `veles fmt` clean. Tests: `sema/testdata/conform/D142-visibility.vs`, a driver test with a two-package build (path export, facade, construction with an unseen field with and without default, the program warning), `examples/features/packages` updated. Docs: chapter on modules/packages, cheat sheet, errors reference. Green on Windows and WSL            |
| B18 | **Small devex fixes found by the 2026-10-14 review** (decision-free)                                                                                                                                                                                                                                                                                                                                                                                              | `veles fmt` keeps a one-expression `unsafe { f() }` on one line when it fits (format case; std reformatted, 73 sites); the 84 `at(...) ?: panic(...)` sites in the tree classified, and D62's bounds facts extended where one pattern repeats (a field kept in range by every assignment is out of scope unless measured to matter); conformance cases for each new fact                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| B19 | **D157: a `Result` or task handle cannot be interpolated** (decided 2026-10-14)                                                                                                                                                                                                                                                                                                                                                                                   | sema: the error (family `results-and-errors`) in `"$x"`, `"${e}"` and template-literal values when the type is a `Result` (a throwing call without `try` included) or `Task`, with the `try` fix (applied by `--fix` where the function may throw) and the `??` fix; conformance `D157-interpolation.vs` (a call, a `Result` local, a `Task`, inside a template literal, `r.toString()` and `"${r.ok}"` accepted); the tree fixed where it trips (any hit is a bug found); docs chapter 7, the errors reference, cheat sheet. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| B20 | **Done 2026-10-17** (progress log). **D158: bare names reach the receiver's members** (decided 2026-10-14)                                                                                                                                                                                                                                                                                                                                                        | sema: the lookup order of the entry (locals, members of `this`, module, prelude) in methods, `init`, `implement`/`extend` bodies and trait defaults, with D5 narrowing and D22's `var` rule unchanged; statics reach static members only, with the error naming `static`; the shadowing warning (family `this`); lambdas capture `this`; conformance `D158-*` (each rule, a parameter shadowing a field, narrowing through a bare field, a sendable lambda naming a member, `executor` state unchanged); LSP hover, rename and references over both spellings; formatter round trip; docs chapter 5, cheat sheet, errors reference. Whether a lint prefers one spelling and whether std/`selfhost/` migrate are asked when it lands (checklist §11). Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| B21 | **Done 2026-10-23** (progress log; `copy` with `init` parameters and on an `extern union` decided the same day — refused, D164 addendum; the `throw X(message: …)` migration left to the user in checklist §11 "B21 left open"). **D163–D165: positional one-parameter constructors, `copy`, `i64?` from the index-of family** (decided 2026-10-14)                                                                                                               | **D163**: sema accepts one positional argument where the constructor has exactly one parameter (fields with defaults and `init` parameters counted), the "add a name" error and fix when a type grows; the tree migrated where it reads better (`throw X(message: …)` sites by `--fix`, as the user chooses when built); conformance `D163-*`. **D164**: `copy` generated per struct/error/variant (generic too), named arguments defaulting to the receiver's fields, `init` run, visibility as construction, refusals (D111, `with`-bound), a hand-written `copy` wins, a field named `copy` warned; `const fun` evaluable; conformance `D164-*`, a run test, hover/completion signature. **D165**: the seven functions return `i64?`; the D62 fact; the migration error and fix for `>= 0`/`< 0`/`== -1`/`!= -1`; std, examples, selfhost, bench, docs migrated; `veles-bench` on `sort`/`strings` no slower. Docs chapters 4–5, cheat sheet, references. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| B22 | **Done 2026-10-23** (progress log; labels in the type arguments of a generic struct or sealed type are refused, not kept — checklist §11 "B22 left open"). **D166: labelled tuples** (decided 2026-10-14)                                                                                                                                                                                                                                                         | parser/AST/formatter: labels in tuple types and literals (all or none; mixed a syntax error); checker: `.label` access, unlabelled ↔ labelled conversion, the different-labels error with its fix, labels in hover/errors; codegen unchanged (same layout and instances — a golden shows it); `"$t"` prints labels; std: `Map.entries()`, `enumerate()`, `splitOnce` labelled, `.0`/`.1` sites in std/examples/selfhost moved to labels where clearer; the self-hosted parser reads the new syntax (G-gates over a corpus file using it); conformance `D166-*`; docs chapters 3–4 with the tuple-vs-struct guideline, cheat sheet. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| B23 | **Done 2026-10-16.** **D168: racing computations** (decided 2026-10-15)                                                                                                                                                                                                                                                                                                                                                                                           | parser (Go and `selfhost/`, G-gates): `async` arms, bare arms, `src as name =>`; the removed `val x = src =>` / `val r = await h =>` / `val _ = src =>` keep parsing with an error naming the fix (`sema/removed.go`); checker: arm kinds, bare-arm and `as` rules (none on sleep/send/wait/changed), race type as the join, `async` placement extended to race arms, effects (arms' errors join the function's `throws`), `awaitFirst` built-in (n ≥ 2, inline `async` only, the errors of the entry); lowering: async arms launched in arm order into a race-owned group, losers cancelled and joined before the winning body; formatter aligns `as`/`=>`; hover on async arms; std (`http`, prelude `iter`), examples, docs (chapter 12, cheat sheet, `veles-idioms-and-apis.md`) migrated; tests: winner/loser timing (no wait for the loser), loser `with` cleanup ran, first-finished error thrown, handle arms not cancelled, under 1/2/4/8 threads; conformance `D168-*`. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| B24 | **Done 2026-10-16.** **D169: `race` as a block, `select` with arms, `async { }`** (decided 2026-10-16)                                                                                                                                                                                                                                                                                                                                                            | parser (Go and `selfhost/`, G-gates): `select {` contextual, `race { body }`, `async { body }`; the old `race` with arms, `as` arms, bare arms and `awaitFirst` error with fixes; checker: race contestants (launches into the race's group, one type, none → error/panic, one → warning), select = the D168 arm checker without bare/`as`, async block = a started sendable lambda (captures, `return`, break/continue refused); lowering: race over a run-time list of tasks (`veles_race_new(n)`), losers cancelled and joined; formatter, dump, hover; std, examples, docs, tests migrated; conformance `D169-*`; driver test on 1/2/4/8 threads. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| B26 | **Built 2026-10-18 except what checklist §11 lists** (progress log; `T.annotated<A>()` and the pair form of `inline loop` the same day). **D174: user-defined annotations** (decided 2026-10-18; large — the mechanism first, before C9)                                                                                                                                                                                                                          | (1) parser/AST/formatter (Go and `selfhost/`, G-gates): `annotation Name(params) on targets [for Traits]`, `@Name(args)` resolved as a type (imports, qualified, D142 visibility); sema: target check, constant arguments, the reader-clause warning, lowercase names reserved, "attribute" renamed "annotation" in the diagnostics, `ast` and docs; (2) reflection: `T.fields` / `Self.fields` as a constant description (name, type, doc comment, default, annotations; enum members and variants too), `f.attr<A>()`, `f has A`, readable from `const fun` (results stored read-only) and `inline loop (f in Self.fields)` in trait default bodies with typed `get`/`set` by field; errors inside an unrolled body name the field and its type; (3) hover shows values, `veles doc` lists annotations, the LSP completes them; (4) conformance `D174-*`, a driver test of a small library (a `Validate` trait and an `Args`-like table) used from another package; docs chapter 14 (written for readers who dislike annotations: data only), cheat sheet, errors reference. Then C9 on top; the Go `Codable` derive and its `@key`/`@skip`/`@required`/`@tag` move to `std/codec` later (checklist §11). Windows and WSL green                                                                                                                                                                                                                      |
| B27 | **D177: lint levels, lint rules in Veles, formatter options** (decided 2026-10-18; parts 1–2 built 2026-10-18, part 3 built 2026-10-19 except `[lint.forbid]`, type-aware rules and exit callbacks)                                                                                                                                                                                                                                                               | (1) levels: a lint registry in `sema` (name, default, doc), `[lint] name = allow\|warn\|deny`, `[lint.forbid]`, errors for unknown names with a suggestion, `veles lint --list`; (2) formatter: `trailing_commas`, `line_endings`, the `long_line` lint with `max_width` 120; tests, `driver` manifest tests, docs (chapter 11, 25, cheat sheet, errors reference); (3) rule API: first measure the D113 evaluator running a visitor over the std tree (budget: no visible `veles check` slowdown), then `std/lint` (`Rule`, `Visitor`, `ctx`), the reflected AST, rules from dependencies by `package.name`, fixes through `veles check --fix`, a sample rules package with tests; if the evaluator is too slow, record the numbers and ask before changing the run model                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| B28 | **D178: skipping tests** (built 2026-10-18; left over: checklist §11)                                                                                                                                                                                                                                                                                                                                                                                             | (1) vocabulary `skip`/`skipIf` in test code (sema, as `require`; works in helpers and lambdas); (2) codegen/runtime: a third outcome, the `skipped: why` line, the summary `N passed, M failed, K skipped`, exit 0, `veles test --fail-on-skip`; (3) suite bodies accept `skip`/`skipIf` statements (parser Go and `selfhost/` with the G-gates, formatter, AST), evaluated once before the suite's first test; (4) std annotations `Skip`/`SkipIf` and the targets `test` and `suite` (D174), constant conditions, a skipped test is not run; (5) the Go driver tests' `t.Skip` conditions mirrored in `selfhost`/std tests where they exist; LSP outline/hover, docs chapter 14, cheat sheet, errors reference, conformance `D178-*`, driver tests                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| B29 | **Part 1 done 2026-10-26** (progress log). **D210: `lazy val` globals; D211's docs; the F1 race fix** (decided 2026-10-26, `veles-patterns.md`)                                                                                                                                                                                                                                                                                                                   | (1) **F1 first** (decision-free, checklist §11): a module `val` and a `static val` must have a `Sendable` type, error with a fix (`Mutex(value: …)`, an immutable collection, an executor), std/examples migrated; pinned by a sema rule test and the four-task `MutableMap` program refused. (2) **D210**: parser (Go and `selfhost/`, G-gates) `lazy` before a module or `static val`; sema (refused on `var`/locals/fields/`const`, initializer `sendable`, no suspend/throw, `Sendable` type, cycles counted); codegen a once-cell with an acquire fast path, hoisted out of loops; formatter modifier order; hover "lazy: built on first read"; tests: built once under 8 threads, never built when unread, panic raised again, debug and release, Windows and Linux; a bench showing the read costs no more than a plain global after the first. (3) **D211**: a docs chapter on application structure (resources made in `main`, passed down or bound to task-locals, fakes in tests).                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| B30 | **D212: `const struct`** (decided 2026-10-26)                                                                                                                                                                                                                                                                                                                                                                                                                     | parser (Go and `selfhost/`, G-gates): `const` before `struct`, also for variants in a sealed trait; AST, `Dump`, formatter modifier order; sema: the deep-immutability check per field with an error naming the field and the immutable alternative; per-instance check for generic const structs; const structs Sendable by construction; hover and API diff; tests: accepted and refused field kinds, generics, nested and sealed cases, a const struct as a module `val`, a map key and a `const`; docs chapter on structs and the cheat sheet. The pass-by-address optimisation only with a benchmark.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| B31 | **D213: allocation hints** (decided 2026-10-26)                                                                                                                                                                                                                                                                                                                                                                                                                   | sema: record each allocating site (the eight kinds in D213) in the index with what, why and how to avoid it; LSP: opt-in inlay hints `heap` with that hover (a client setting); `veles explain --alloc <file\|fun>`; a test comparing hinted sites with the allocation calls in the IR over a corpus (examples, bench); docs chapter on performance and the editor chapter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| B32 | **Done: steps 1–5 2026-10-26, step 6 and `noalias` 2026-10-27** (Windows and Linux) (progress log). **Value layout: D214, D215 and the decision-free rest of `veles-layout-spec.md`** (decided 2026-10-26; user: "all but step by step" — each step green before the next)                                                                                                                                                                                                                                              | (1) **One layout truth** (§5.4, checklist §11 bugs 2–4): `constFields` from `StructPlan`/`fidx`, `isMem` at ≥ 128 as D121 says, `veles_task_set_unwrap` keeps `off`, the sealed payload offset from one `Layout` function; the nullable shape known only to `makeNullableT`/`nullFlag`/`nullVal` (IR byte-identical, goldens unchanged); regression test for the constant table of an `@align` struct. (2) **D214**: definite assignment for local `var`s in sema (read, `&`, capture, method call), error + docs (chapter 2, errors reference), conformance `D214-*`, the two §4.1 programs refused. (3) **Borrowed big arguments** (§5.3): `bench/bigargs` with a Go reference first; caller/callee rules, `readonly nocapture`, D121 text amended; conformance for every aliasing case in §6.3; `-record` before and after; `noalias` only after the stress runs. (4) **Nullable niche** (§5.2, needs 2): `Layout.Niche`, `T?` sized as `T` where it applies, GC descriptors, constants, D5 text amended; selfhost peak heap before/after. (5) **D215**: `compact struct` in the parser (Go and `selfhost/`, G-gates), AST/`Dump`, formatter modifier order, sema refusals and the one-field warning, `StructPlan` placement, hover size/padding/"`compact` would make it N"; tests and docs chapter 13. (6) Exact sealed pointer maps; over-aligned values in tuples/nullables/sealed payloads. Each step: the §6 checks, Windows/WSL/macOS green. |
| B25 | **Done 2026-10-17** (progress log). **D173: extension functions on types you do not own** (decided 2026-10-17)                                                                                                                                                                                                                                                                                                                                                    | parser (Go and `selfhost/`, G-gates): `extend T with fun …` as a one-function form beside `extend T { }`; sema: `extend` on a foreign type becomes an extension (inherent wins + "never called" warning, public API only, not satisfying bounds or `is`, importer-scoped through `use`, ambiguity error naming the fix, effects inferred, `public` per M5); the own-type `extend` unchanged; formatter keeps the one-line form; hover and `veles doc` name the source module; std gets a first user (a helper that was a free function or a one-method trait); tests per rule, conformance `D173-*`, docs (traits chapter, cheat sheet, errors reference); the open details in checklist §11 settled and removed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |

| B10 | **Done 2026-10-01.** **D100: `with x = e` as a statement, `with t = async f()`, resources cannot outlive their block; whole tree migrated** (decided 2026-09-30) | parser: `with name = expr` in a braced statement list (not module level, expression bodies, braceless bodies or operands — each an error naming the block form), one binding per statement; checker: lowers to D43's block form over the rest of the block, one cleanup stack per block with block-form `with`s (last registered closes first); a statement-form `with` as the last statement warns "closed as soon as it is opened"; `async` allowed as a `with` value — a fail-fast child of the rest of the block, cancelled then joined at every exit unless finished, `await t` gives its value, errors join the inferred `throws`, captures under D35; the "`async` must be inside scope/gather" error names the `with` form; new family `resources` with the D43 not-Closeable message and D100 part 3's escape error (return, block value, stored outside, captured by a returned/stored lambda; passing as an argument allowed); formatter round trip; LSP hover on `with` says where it closes, inlay hint after the enclosing `}` lists the closes in order, completion; conformance `D100-*` (every error and warning, the ordering, loop iterations); driver/run tests that each exit path (end, return, break, continue, failed try, throw, panic, cancellation) closes in order and that a with-task is cancelled at the end, fails fast, and is not waited for when awaited; `examples/features/with` extended; migration of every statement-position `with` block that ends its block and every `scope { … t.cancel() }` background-task shape in std, examples, bench, templates, docs and Go test programs, expected outputs unchanged; docs chapters 7 and 12, cheat sheet, errors reference; Windows and WSL green |
| B16 | **Done 2026-10-03.** **D120–D123: C layout attributes, `extern union`, `Array<T, N>` and const generics, variadic C calls** (decided 2026-10-01; after A8, which they rely on for by-value layout) | **D120**: `@packed` (unaligned field loads, `&` on a packed field refused), `@align(n)` on structs and extern fields (checked power of two ≥ natural), `extern union` (construction with one field, access in `unsafe`, `CLayout`, no `==`/`Display`), `@transparent` (ABI of its field in extern signatures); a driver test against a C file checking `sizeof`/`offsetof` of each shape agrees with Veles (`epoll_event`-like packed struct with a union, an aligned struct, a transparent handle). **D121**: `Array<T, N>` type (sema/types, layout, GC descriptor repeating the element's), literals with length check, `at`/`set`/`len`/loops/read-only adapters/`toList`/`toArray<N>`/`make`, `==`/hash/`Display`/`Codable`/`Default`/`Sendable`/`CLayout`/`withRaw`; const generic parameters on functions, structs and `extend` (inference from argument types, explicit `<16>`, `N + 1` refused), per-`N` instances; constant indexes compile to plain loads (golden); std's SHA-256 state and similar fixed buffers moved to it with `veles-bench` before/after; conformance, an example, formatter, hover. **D122**: notes #16 closed, docs say how a binding package is laid out. **D123**: `...` in extern declarations, promotions, refusals; `printf` and `fcntl`/`open` (POSIX) called in a driver test on both platforms. Docs chapter 13, the cheat sheet, the errors reference. Windows and WSL green |
| B15 | **Done 2026-10-02.** **D116, D117, D119, D135: suspension follows the argument (and B8), `is Trait` both ways, `Default`, `is T` downcast** (decided 2026-10-01) | **D135** (with D117's run-time part): `x is T` on an open trait object by type-id comparison, narrowing, `when` arms, the "never matches" error only when `T` does not implement the trait, a type parameter refused; conformance and a run case. **D116**: the suspension pass gains _conditional_ suspension (calls of, or passing on, a `suspends` function parameter); call sites suspend only when a bound argument does; codegen emits a plain and a coroutine instance on demand (stenciling key bit); hover "suspends if `f` does"; trait methods unchanged; conformance and run tests (a pure lambda through a `suspends` parameter inside `withLock` and a D107 region; a suspending one suspends; cancellation inside it); then **B8**: the eager adapters move from `sema/lower_list.go`/`check_map.go` into the prelude over `throws E` + conditional suspension, `xs.map(x => slow(x))` allowed, `veles-bench` before/after (no regression over noise) and the `--timings` IR size of `examples/httpd` recorded, the Go lowering deleted. **D117**: the decision-free message fix first; run-time `is Trait` on trait objects with narrowing, `!is`, `when` arms, the per-trait type-id table, the limits (object-safe, fully applied, supertrait limit) and the static warning; compile-time `T implements Trait` in `if`/`static assert`, the bound added in the true branch, dead branch dropped per instance, dictionary entry for shared instances; conformance, a run example (a `Flusher` upgrade, a generic `show`). **D119**: prelude `Default` with the listed implements, derive by empty `implement` (field default, else the type's `default()`, else an error naming the field; generic bounds inferred; refusals for sealed/enum/`Secret`/task-holding), `T.default()`; conformance and an example. Docs: chapters 8, 10, 12, the cheat sheet, stdlib and errors references. Windows and WSL green |
| B13 | **Done 2026-10-02.** **D111, D112, D114, D115: task-holding values, `Secret<T>`, unchecked access, never-closed warning** (decided 2026-10-01; D111 after B10) | **D111**: task-holding types computed from declared fields; such a value must be a `with` value or returned straight up (error family `tasks` with the fix elsewhere); `async` accepted as a field argument of such a constructor that is returned or `with`-bound; held tasks become fail-fast background children of the receiving block (D100 part 2 machinery), cancelled and joined in reverse field order before `close()`; a failure between the `async` and the return cancels and joins first; hover; conformance cases; `std/http/limits.test.vs` rewritten with a `TestServer` helper and every other server test migrated where it repeats setup; a run test that a held task's `Err` fails the receiving block and that close order is tasks then `close()`. **D112**: prelude `Secret<T>` for `string`/`List<u8>` (other `T` refused), redaction everywhere text is produced (interpolation, `expect` capture), not `Encodable` (derive error names `@skip`), `Decodable`, constant-time `==`, not `Hashable`/`Comparable`, `close()` wipes and later `expose()` panics at the caller; runtime: a wipe flag in the object header and a `memset` at sweep (test with `VELES_GC_THRESHOLD` that the freed bytes are zero — a runtime test reading the span); docs (chapter 7 or a security page), stdlib reference; std's keys (HMAC, JWT) moved to it. **D114**: the three unchecked built-ins, refused outside `unsafe`, checked with a panic in debug, unchecked in release (codegen golden in both profiles). **D115**: the local hand-off analysis, the warning and fix, measured over std/examples/bench (each hit fixed or argued), conformance cases including no warning on `serve(listener)`. Windows and WSL green |
| B14 | **Done: parts 1–3 2026-10-03, part 4 (`const fun`) 2026-10-05; the review list is notes_to_change C1 and checklist §11.** **D113: compile-time evaluation** (decided 2026-10-01; large — the parts land in order, each green) | (1) constant expressions over consts with cycles refused, compile-time overflow/division/shift errors, string `+`/interpolation, `len`, conversions, `if`/`when`, tuples/enums/structs without `init`, `at(i)` on tables; (2) constant `List`/`Map`/`Set` emitted read-only in the binary (the collector skips them; a test that a table is not rebuilt at start-up and is shared), elements in `when` patterns; (3) `static assert(cond, "why")` at module level and in bodies, the failure quoting the reason and the constants' values; (4) `const fun` — declaration-time checking of the body rules (each refusal an error with its rule), the HIR evaluator `sema/consteval` with a step budget (`--const-steps`, default 10 000 000) and the recursion limit, panics and throws as compile errors with the call chain, IEEE-exact floats (`f32` rounded each step), std functions marked `const fun` as programs need them (strings, numbers, list building first); a differential test: every `const fun` in the test corpus evaluated at compile time equals its run-time result; migrate std's start-up tables (SHA-256 constants, base64 alphabets, keyword tables) to constants and measure start-up and binary size; docs chapter 2/3 and the cheat sheet; self-host plan notes the evaluator as a port cost. Windows and WSL green |
| B12 | **Done 2026-10-02.** **D107–D110: `Mutex` guard, `race` send arms, `with expr`, concurrency helpers** (decided 2026-10-01; build after B10, which they extend) | **D107**: `lock()` accepted only as a `with` value (error with fix elsewhere), binds `*T`; the held region (the `with`'s body) refuses every suspension with the message in the spec; the pointer under D100 part 3; re-locking panics; conformance cases (suspension in statement and block form, escape, outside `with`), a threaded run test (8 tasks × 10 000 increments through `with n = counter.lock()` equal the total), hover. **D108**: parser/checker accept `ch.send(v) =>` arms (operands evaluated once in arm order at the start); the runtime claim makes win and commit one step; tests: a losing send arm never delivered (a receiver after the race sees nothing), a winning one delivered, a closed channel panics, under `VELES_THREADS` 1/2/4/8 ×5 and `--sanitize` on Linux; the which-arm-wins rule documented (checklist §1.3). **D109**: `with expr` and mixed block items; conformance and format cases. **D110**: the four helpers with doc comments, catalogue/hover entries, `std/…test.vs` unit tests (retry counts and delay, cancellation during the wait; Semaphore never exceeds n under 8 threads, `tryAcquire`; drains end at close; ticker drops ticks for a slow reader and stops on close), docs chapter 12 and the stdlib reference. Windows and WSL green |
| B11 | **Done 2026-10-01.** **D101–D106 and D134, the small consistencies** (decided 2026-09-30/10-01), plus three decision-free fixes found preparing them | **D134**: `tryChain` (`sema/check_expr.go`) unwraps every failing link of the receiver chain instead of warning, the error type the union, arguments not covered, a `Result` method link read as today; the old warning becomes "the inner `try` is redundant" with a fix; conformance cases (two and three failing links, a `Result` method in the chain, an argument Result, with `catch`); `(try f()).g()` sites in std/examples/docs migrated. **D101**: the parser reads `IDENT =` at the start of an `if`/`when` head as a missing `val` — one `syntax` error with a fix, no cascade; conformance case. **D102**: the compile-time refusal (the call list in the spec, map `set` allowed for the loop's own key, `extend` methods on the mutable type count, lambdas called in the body count), replacing `lint_staleref`'s warning; the runtime modification count on list/map/set/deque headers, bumped by structural operations only, checked each step of `loop` and of `iter()`, panicking at the loop's line in both profiles; `veles-bench` before/after (sort, json, sha256, list workloads) recorded, with no regression over noise or a stated cost; migrate any std/example loop the rule refuses; conformance and a run test for the indirect panic. **D103**: one-launch `gather` typed as the `Result` (checker, lowering, hover), `.0` sites migrated; `async` on a `sendable fun` value (local, parameter, field, expression) with effects from the type, plain `fun` refused with a fix; prelude trampolines removed; conformance, a run example, `with t = async f()` covered once B10 lands. **D104**: eight built-ins with catalogue entries (LLVM `fshl`/`fshr`/`bswap`/`bitreverse`/`copysign`; `nextUp`/`nextDown` in the prelude over `toBits`), an example with the edges of every width, identical in debug and release, codegen golden. **D105**: `reserve` on `StringBuilder`, `MutableMap`, `MutableSet`, `Deque`, allowed before a move (D63), catalogue entries, one test each that inserting up to `n` does not grow. **D106**: empty-literal message names `MutableList`/`List` and the default element type, fix from the first deciding use (preferred, applied by `check --fix`); `Range.isEmpty()` public (`holdsNothing` renamed); formatter parenthesizes a lambda that is a `when` arm's value (format case, corpus unchanged or reformatted). **Decision-free**: the `async` direct-call error names what to write; the needless-`throws` warning's family (checklist §6); a type argument inferred from the expected type (`val small: i8 = id(12)`, §11 limit removed). Docs: cheat sheet, chapters 4 (collections, loops), 6 (`when`), 12 (tasks), the stdlib and errors references. Windows and WSL green |
| B9 | **Done 2026-09-29.** **D86: numeric conversions are methods; `as` only renames** (decided 2026-09-29) | `toT()` / `wrapT()` on all ten numeric types and `p.cast<*raw U>()`; `x as T` an error with a fix and a lint migrating std, examples, docs and tests; literal overflow is a compile error; formatter, LSP hover, docs (tutorial, cheat sheet, errors), a codegen golden and sema conformance cases; Windows and WSL green |

## C — Standard library breadth (each public API asked first)

In this order, because each unblocks the next real program:

1. **Done 2026-09-29.** **`std/log`** — levels, structured fields, request-scoped through
   task-locals, no cost when a level is off (checklist §5.7).
   _(2026-10-01: every public API below is decided — batch 7 of
   `archive/veles-spec-prep.md`, spec D124–D130 — so this track needs no further
   questions; each item's acceptance is its spec entry built in full, tests in
   `std/<module>/_.test.vs`, docs (a chapter section and the stdlib reference),
   Windows and WSL green.)\*

2. **Done 2026-10-04** (compress and `testServer` with C2). **`std/http` server completeness** — cookies, forms (urlencoded,
   multipart to disk), static-file caching (`ETag`, `Range`), max
   connections with backpressure, chunked encoding, streaming bodies, CORS
   (checklist §5.2). Remaining: **`std/compress` + `http.compress()` (D124)**,
   benchmarked against Go's `compress/flate`; `http.testServer` (D130, after
   B13's D111).
3. **Done 2026-10-04.** **`std/config`** (D125) — `config.load<T>`, env + dotenv/JSON files, every
   problem at once, `config.describe<T>()`; `examples/` service template
   (`veles new --template server`) moved to it.
4. **Done 2026-10-05** (runtime metrics deferred, checklist §11). **Observability** (D126) — `std/otel`: metrics, traces, logs; OTLP/protobuf
   export (needs C7 and, for `https`, E1); spans in `http.serve`, the client and
   `std/db`; runtime metrics (GC counters from E5). Health endpoints
   (`http.Health`, D133) first — they need neither the client nor TLS — and
   the server template moved to them.
5. **Done 2026-10-04.** **`std/fs`** (D130) — `writeAtomic`, `lock`/`tryLock`, `sync`, `seek`,
   `lines`, `copy`; streaming already through `fs.File`, and `io.Stream`
   (D128).
6. **Done** (D110 with B12 2026-10-02, endian bytes 2026-10-04). The concurrency helpers (D110 — after B12) and endian bytes (D130).
7. **Done 2026-10-04** (HTTPS with E1). **The HTTP client over plain TCP** (D127) — `http.fetch`, `get`/`post`/…,
   `http.Client`; `io.Stream` (D128) first, so TLS (E1) plugs in.
8. **Done 2026-10-04.** **Template literals** (D129 part 1) — the language feature (`@template`,
   `tag"…"`), before E2 needs it.
9. **Built 2026-10-24** (progress log; `examples/todo` stays on its own parser, checklist §11). **`std/cli` (D167, decided 2026-10-14; after B26 — its `@positional`, `@short`, `@name`, `@env`,
   `@hidden` become annotations declared in `std/cli` on D174's mechanism, not compiler code)** — the derive, `cli.parse<T>`/`tryParse<T>`,
   help from doc comments, subcommands as a sealed trait, typo suggestions, every problem at
   once; tests in `std/cli/*.test.vs`; `examples/dedup`, `httpd`, `todo` moved to it (their
   `commands.txt` outputs updated with `-update` where help/usage text changes); docs chapter
   15 and the reference. Names confirmed 2026-10-24 (D167 addendum): `cli.parse<T>` throws,
   `cli.args<T>()` prints and exits; `@Flag(short:, name:, env:, hidden:)` and `@Positional`;
   `cli.Args`, `cli.Error` (`kind` Help/Version/Usage), exit 0 / 2.
   **Built first, in this order (decided 2026-10-24):** (a) the `build` module (D160:
   `build.target`, `Os`, `Arch`; D204: `build.program`), constants filled by the compiler from
   the target and the manifest, with tests and the D160 constant-branch rule; (b) D203's
   reflection — `f.default()`, `f.isOptional`/`f.isList`/`f.Inner`, pruning of `if`s on handle
   constants, `T.members` (enum members, sealed variants: name, doc, index, annotations,
   `Type`), every enum `Parsable` — each with conformance tests; (c) `std/cli` on them.

## D — Decisions to ask (checklist §9)

_(2026-10-27)_ Open: Q45 (a cancelled operation's outcome), Q57 (a marker for
functions that panic), Q59 (serial executor), Q60 (replaceable globals), Q63
(definite assignment through `break`). The menus with unasked items:
`veles-patterns.md` (P5, R1–R8), `veles-idioms-and-apis.md`,
`veles-syntax-ideas.md`, `veles-streams-spec.md` (S1, S2).

_(2026-10-14, history)_ Every batch below was asked and decided by 2026-10-13 (D100–D156);
the batch prepared in `archive/veles-direction-2026-10-14.md` (Q32–Q43: idioms, self-hosting
needs, platform APIs) was decided the same day as D157–D167. Checklist §9 is empty.

_(2026-09-30, history)_ The order of asking is now the batch table of
`archive/veles-spec-prep.md` §3: batches 1 (nesting, D100) and 2 (small
consistencies, D101–D106) are decided; next are 3 tests and concurrency (Q2,
Q1, P11, a `Mutex` guard), 4 safety and build (Q9, Q11, Q10, unused
`Closeable`), 5 types and effects (effect-generic functions — B8's blocker —
Q5, Q6, `Default`), 6 FFI (Q7), 7 the std APIs ahead of track C/E, 8 packages
(Q14).

~~First, raised by the user: Q15 … Later: Q5, Q6, Q9–Q12, Q14.~~ (Q3, Q4, Q8,
Q15, Q16 decided 2026-09-29.)

## S — Towards self-hosting (ongoing; no compiler rewrite yet)

The front-end rewrite itself is E4. This track keeps the language and std
on course for it, a little at a time, alongside A–C.

| #   | Task                                                                                   | Acceptance                                                                                                                                                                                                                                                                                 |
| --- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| S1  | **Done 2026-09-29.** **Refresh `veles-selfhost-frontend-plan.md`** against D60–D78     | its §4 "not gaps" list re-checked (a module-level `var` is now an error unless `Mutex`/`Atomic` — the keyword table and any interning move to a `val` or into a struct; `this`; `init(params)`; D78 tests for the harness; the UAX #31 table generator); stale claims struck with the date |
| S2  | **Done 2026-09-29.** **Compiler-shaped benchmarks** in `bench/`                        | Veles vs Go on: tokenising a large file, building a sealed-tree AST and walking it with `when`, a string-interning map, emitting text through `StringBuilder`; recorded, and each gap over 3× Go gets a checklist item                                                                     |
| S3  | **Done 2026-09-29.** **Capability audit for a whole compiler**, not only the front end | a section in the self-host plan: what sema/codegen in Veles would lean on (large pointer graphs under the GC, maps keyed by structs, deterministic iteration order, sorting, `os.run` of clang, file I/O, deep recursion) — each marked has / gap, gaps become checklist items             |
| S4  | **Readiness in every decision**                                                        | `veles-decide` prepares a "what this means for writing the compiler in Veles" line when it applies (Q15/Q16 first)                                                                                                                                                                         |
| S5  | **Done 2026-10-08.** Start E4's P0 (`source` + harness)                                | once A1 is green and S1 is done                                                                                                                                                                                                                                                            |

_(2026-10-14, from the audit in `archive/veles-direction-2026-10-14.md` §2.)_ The front end agrees
with Go's (G1–G4). What stands between it and a compiler written in Veles, decision-free
steps first; the sizes are today's Go: `sema` 35 000 lines, `codegen/llvm` 9 000, `types`
1 400, `driver` 2 200, `format` 2 400, `lsp` 3 200 (the C runtime stays C).

| #    | Task                                                                                                                                                                                                                                                                                                               | Acceptance                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| S6   | **Built 2026-10-23; non-UTF-8 input 2026-10-24 (S17, D199)** (progress log). **Finish E4**: P6 `Layout`, a dump that prints every span, files that are not valid UTF-8                                                                                                                                             | G2 compares spans; the `Layout` of every corpus file agrees with Go's; the lexer reads bytes (`fs.readBytes`), so the fuzz findings and the non-UTF-8 files join the corpus; checklist §8b ticked                                                                                                                                                                                                                                                                                                                                                                             |
| S7   | **Done 2026-10-23 with S16: `bench/parse` 3.0× → 1.0–1.1× Go on Linux, 1.1× on Windows** (progress log; the target became Go's speed, D198). **Front-end speed, as a finding about the language** (3 MB of std: Veles 1.3 s, Go 0.55 s)                                                                            | profile (`bench/parse` added: the self-hosted parser over std), fix what is the compiler's or the runtime's (not the port's), record in `bench/results.md`; target ≤ 1.3× Go. The port is not rewritten to allocate less (self-host plan §5)                                                                                                                                                                                                                                                                                                                                  |
| S8   | **Built 2026-10-23** (progress log; DWARF on Windows too — the target is `windows-gnu`; what is left: checklist §11). **Debug info**: LLVM `!dbg` line tables and locals in debug builds (DWARF on Linux, CodeView on Windows)                                                                                     | `gdb`/`lldb` and the VS Code C++ debugger stop at a Veles `file:line`, step, and show locals of scalar and struct types in a probe; a codegen golden; docs chapter 13. A compiler written in Veles needs a debugger                                                                                                                                                                                                                                                                                                                                                           |
| S9   | **The formatter in Veles** (`selfhost/format`), the next slice with a free oracle                                                                                                                                                                                                                                  | needs S6's `Layout`; `veles fmt --stdout` byte-identical with the Veles formatter over the whole corpus (gate G5); the Veles sources stay idiomatic (self-host plan §11)                                                                                                                                                                                                                                                                                                                                                                                                      |
| S10  | **First cut built 2026-10-19** (progress log; what is left: checklist §11). **Oracles for `sema`**: stable text dumps of each phase                                                                                                                                                                                | `veles check --dump=names                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | types  | hir`(resolution, every expression's type, the lowered HIR) deterministic across runs and platforms, golden-tested; this is what a Veles`sema`is gated against, phase by phase, as the parser was against`veles parse` |
| S11  | **Written 2026-10-19: `veles-selfhost-sema-plan.md`** (phases R0–R9, gates, costs, four questions Q-S1–Q-S4 for `veles-decide`). **`veles-selfhost-sema-plan.md`** — the cut and the phases of the checker port                                                                                                    | the order (resolver → types → declarations → bodies → effects → lowering → `const fun` evaluator), each phase's gate over S10's dumps, what the language must offer first (the Q32–Q43 answers that apply), a size and time estimate measured from S6–S9                                                                                                                                                                                                                                                                                                                      |
| S10b | **Built 2026-10-19** (progress log; what is left: checklist §11). **Per-phase dumps** for the sema port's gates (decision-free; from `veles-selfhost-sema-plan.md` §2.1)                                                                                                                                           | `veles check --dump=modules                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | scopes | decls                                                                                                                                                                                                                 | effects` (the module graph, the scopes before bodies, the resolved declarations, the inferred effects with their rounds), a spans flag on every dump, golden-tested; the first gates of R1–R3 and R5 |
| S12  | **Module interface files** (spec §5; E8 per-module caching)                                                                                                                                                                                                                                                        | inferred effects and signatures written per module and read back, so a 50 000-line compiler in Veles is not rechecked whole on every build; `--timings` before/after on `selfhost/`                                                                                                                                                                                                                                                                                                                                                                                           |
| S13  | **One syntax tree for the compiler and for lint rules** (from the std/lint duplication, 2026-10-19). Today the tree exists as `ast/ast.go`, `selfhost/ast` and a generated copy in `std/lint` (`go test ./selfhost -run LintAPI -update`), plus a Go-to-Veles reflector (`sema/lintrules.go`, `TestRuleASTFields`) | (1) once the Veles front end is the compiler (S6, S9–S12), rules get its tree directly: delete the reflector and `ruleFieldAlias`; (2) **decide** (`veles-decide`) whether `ast`, `source` and `Kind` become one public std module that `selfhost` imports, removing the generator, or stay generated; (3) **decide** the rule API's stability: version `std/lint` apart from the compiler tree, or accept breaking rules until 1.0, and say so in chapter 11; (4) until then the generator and `TestLintAPI`/`TestRuleASTFields` are what keep the three trees from drifting |

_(2026-10-19, from the gap review "what do we need to be self-hosted and feature rich".)_
Decision-free items added to the track; the language-shaping questions (SIMD, a wasm target)
go through `veles-decide` first. (Operator overloading already exists: D71 `Addable` etc.)

| #   | Task                                                                                                                                                                                                                                                                                         | Acceptance                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| S14 | **Harness built 2026-10-19** (`internal/bootstrap`, progress log); the gate itself waits for S11's `selfhost/compiler`. **Bootstrap fixpoint test**: stage0 (Go) builds stage1 (Veles compiler), stage1 builds stage2                                                                        | `veles build` of the Veles compiler by stage1 and by stage2 produce byte-identical IR/binaries; a driver test and a CI line; the proof that the compiler is self-hosted. Needs S11's port; the harness (a stage runner, an IR diff) can be written earlier against the front end                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| S15 | **Backend decision for a self-hosted compiler**: keep textual LLVM IR + clang, or add a small second backend                                                                                                                                                                                 | a short write-up (via `veles-decide`): what a Veles-written compiler depends on (clang on PATH), what a second backend would cost, what the wasm target (G1 below) needs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| S16 | **Built 2026-10-23** (progress log; `bench/parse` at Go's speed on Linux, 1.1× on Windows; what is left: checklist §11). **D198: trees are boxed** (decided 2026-10-23; from S7) — recursive sealed types box their variants, `indirect struct` / `inline struct`, the lopsided-variant lint | sema: recursion found through pointers, collections, nullables and other types' fields; a boxed variant's layout (tag + one word) in `types/layout.go`; codegen: construction allocates the variant, `when`/`is`/field reads load through the pointer, a write through a narrowed `var` copies the box first, equality/hash/order/show/`Codable` and the constant evaluator unchanged in result; GC descriptors; parser, formatter, self-hosted parser, editor grammar and hover for `indirect`/`inline` (contextual words before `struct` in a sealed trait's body); the lint with its fix (family performance); conformance `D198-*` and a driver test of value semantics under boxing (debug and release, 1–8 threads); a codegen golden; `bench/parse` against Go's speed (the target the user set: Go, not 1.3×), `trees`, `ast` and the whole suite no slower; `selfhost/ast` marks nothing it does not need; docs chapter 9 and chapter 13 (layout), the cheat sheet. Windows and WSL green          |
| S17 | **Built 2026-10-24** (progress log). **D199: a source file must be valid UTF-8** (decided 2026-10-23; finishes S6)                                                                                                                                                                           | Go lexer: every undecodable sequence an error at its position, comments included (message in D199), family syntax; the self-hosted front end reads bytes (`fs.readBytes`), reports the same errors and lexes nothing more; the non-UTF-8 fuzz finding and a hand-written file (a stray byte in a comment, a string and code) join the corpus under G1–G3; conformance file; errors reference; LSP shows the diagnostic. Windows and WSL green                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| S18 | **Built 2026-10-24** (progress log). **D202: `string.startsWith(prefix, startIndex)`** (decided 2026-10-24)                                                                                                                                                                                  | builtin signature + check (named and positional), runtime compare with the out-of-range rule, hover/builtin docs, cheat sheet; a test of the edges (negative, past the end, empty prefix); the self-hosted lexer's `holdsAt` replaced; bench/parse re-measured                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| S19 | **Built 2026-10-24** (progress log). **D200: `[runtime] gc.minHeap` / `gc.goal`** (decided 2026-10-24)                                                                                                                                                                                       | manifest reader (Go and self-hosted test) with the errors in D200; start-up code passes them to the collector; `VELES_GC_MIN_HEAP` / `VELES_GC_GOAL` (and `VELES_GC_THRESHOLD`) override; a driver test runs one program under each; manifest docs; bench/parse sets 32 MiB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| S20 | **Built 2026-10-24** (progress log). **D201: a field moves out of a local whose methods were called** (decided 2026-10-24)                                                                                                                                                                   | per-method summaries (keeps `this`, fields used) in sema; the move rule of D201 with each refusal reason pinned by a conformance test; hover and `redundant_copy`; the self-hosted lexer's `Scan(…)` moves its lists and the parser stops copying them back (`>>` split without `insert`); instruction counts re-measured on Linux                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| G1  | **Gap list from the 2026-10-19 review**, each asked before built                                                                                                                                                                                                                             | **SIMD, D181** (`Vector128/256/512<T>`, `Simd<T, bits>`, lane aliases; codegen to LLVM vectors, sema rules for `T` and width, std type, docs, a `bench/` dot product and byte scan against scalar); **a WebAssembly target, D180**; **`std/regex`, D182 — built 2026-10-19, what is left is in checklist §11** (the `/…/flags` literal in the Go and `selfhost/` lexers, parser, formatter, LSP and editor grammar; the engine in Veles; compile-time check of literals through the const evaluator; string extensions; tests, docs, a log-analyser example, a `bench/` run against Go's `regexp`); `BigInt`/`Decimal`/`Vec`/`Mat` on D71's operator traits (A44, A60); `std/regex` (the `regex"…"` tag); `std/bigint`/`std/decimal`; Unicode normalization and grapheme segmentation; property-based and snapshot tests in `veles test`; runnable doc-comment examples; `veles test --coverage` and a profiler command; `veles apidiff` on module interfaces (S12); CSV/TOML/YAML/URL/MIME/archive modules |

## P — Platform APIs: the low level reached from Veles, a high level on top (decided 2026-10-14: D159–D162)

User, 2026-10-14: "low-level apis like bluetooth etc with high-level dev-ex approach added
to the language". Prepared in `archive/veles-direction-2026-10-14.md` §5; the layer, the order and
the rules are D161, Bluetooth D159; each module's public names are asked when its step
comes (`veles-decide`). Each module: Windows and Linux (macOS with A7), a simulated backend so
its tests run in WSL and CI, typed errors with a `kind`, `with` for every open device,
`io.Stream`/`Subscription` for data, cancellation and `withTimeout` honoured, a package
capability, docs chapter and stdlib reference, an example that does real work (R5).

| #   | Task                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Needs                  |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| P1  | **D160** per-target code: the loader picks `_windows`/`_linux`/`_macos` files for the target; std `build` with `build.target` (`Target`, `Os`, `Arch`) as a constant; constant `if`/`when` compile the taken branch; `veles check` and the LSP check each OS's file set (errors name the OS); conformance `D160-*`, a driver test that builds one module for Windows and Linux (cross-compiled IR) and a check that reports a Linux-only mistake on Windows; docs chapter 11 and 13, cheat sheet. Windows and WSL green | —                      |
| P2  | `net`: UDP, multicast, Unix-domain sockets, `lookup`                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | P1                     |
| P3  | `std/term`: raw mode, size, keys, colour, a progress line                                                                                                                                                                                                                                                                                                                                                                                                                                                               | P1                     |
| P4  | `fs.watch`, `fs.map`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | P1                     |
| P5  | `std/sys/windows/com` (COM/WinRT calls) and `std/sys/linux/dbus` (a D-Bus client)                                                                                                                                                                                                                                                                                                                                                                                                                                       | P1, P2 (Unix sockets)  |
| P6  | `std/serial`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | P1                     |
| P7  | `std/bluetooth`, LE central — **decided 2026-10-14 (D159)**: the surface, errors, `bluetooth.Simulated`, the capability, WinRT/BlueZ; acceptance: an example (a heart-rate reader) tested against the simulator in CI and WSL, std's own tests on real hardware on Windows behind `VELES_HW_TESTS=1`, docs chapter + reference                                                                                                                                                                                          | P1, P5                 |
| P8  | `std/hid`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | P1                     |
| P9  | `std/usb`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | P1, P8                 |
| P10 | **D162** `veles bindgen` over clang's JSON AST; layouts checked by a generated C probe; a driver test over a header with each C shape (struct, packed, union, array field, enum, `#define`, variadic, function pointer); docs chapter 13                                                                                                                                                                                                                                                                                | any time; P5–P9 use it |

## E — Large items, in order

1. ~~**TLS**~~ **Done 2026-10-05** (`examples/apiclient` still open, checklist §8). **TLS** through a system binding (SChannel / OpenSSL) → `std/tls` (D128)
   → the HTTP client over it (§5.3, §5.4); then `examples/apiclient`.
2. ~~**PostgreSQL**~~ **Done 2026-10-05** (the wire protocol in Veles, not libpq; `examples/pgnotes` still open). **PostgreSQL**: libpq binding, `std/db` as D129 (`sql"…"`, pool as a
   task-holding value, transactions as `with`, rows through the derive) →
   `examples/pgnotes` (§5.8). Password hashing (D130) with it.
3. ~~**I/O reactor**~~ **Done 2026-10-08**, kqueue added with A7 (2026-10-21); writev/readv left (checklist §11
   "Deferred from E3"): `poll` → epoll/kqueue/IOCP, writev/readv; an HTTP
   hello load test added to `bench/` first, to measure it (§3.1, §3.3).
4. **Self-hosted front end**, P0 → P6 (`veles-selfhost-frontend-plan.md`). **P0–P5 done 2026-10-08
   (G1–G4 green); P6 `Layout`, span-checking dump and byte input open** — the next steps
   towards a self-hosted compiler are track S below (S6–S12, 2026-10-14).
5. **GC**: _(2026-10-13: the pause is out of the way on Windows, F18; Linux still marks in
   the pause; the metrics and the heap ceiling are still open.)_ pause/heap/allocation metrics, then generational or incremental
   marking once a server-shaped heap is measured (§3.1).
6. **Release engineering**: an `-O2` + LTO profile, static binaries,
   Windows → Linux cross-compile, a Docker image, the "deploying Veles"
   page (§3.2, §4, §7).
7. ~~**Packages**~~ **Done 2026-10-07** (stages a–h; leftovers in checklist §11). **Packages** (D131, D132, revised by D138 on 2026-10-05). In order, each green:
   (a) manifest: `veles.toml` extended — dependency tables (registry version, `git` + version or `commit`, `path`), per-target
   `[native]`, `veles` minimum version, `[workspace]`; the schema written down so `package.vs` can map onto it later; TOML subset
   parser kept small (self-hosting); (b) the loader keys dependency modules by (source, major) with a local name per importer,
   identity-based mangling, only direct requires nameable, name clashes an error — pinned by a diamond and a two-majors test;
   (c) workspaces: members, `veles build|test|check` from the root, path dependencies between members; (d) fetching: git and a
   proxy over the built-in HTTPS client, module cache, MVS per (source, major) with Cargo's major rule, `veles.sum` (hash of the
   tagged bytes, CRLF-proof) written and verified (a tampered cache is a hard error — test), commit pins and the library warning;
   (e) `veles add/update/remove/deps/vendor`, editing the manifest; (f) capabilities: computed per package from source (`unsafe`,
   `extern`, `[native]`, std modules), shown by `veles add` and `veles audit`, a policy table in the manifest that refuses some;
   (g) the registry protocol (documented HTTP: index, versions, immutable archives, checksum log) with a client and a local
   reference server for tests, tiers unreviewed/listed, signed attestations a policy can require; the hosted service itself and
   publishing members of a workspace come after; (h) a driver test against local bare git repositories and the reference
   server (tags, two majors, a commit pin, a sum mismatch) on Windows and WSL; `examples/features/packages` uses one published dependency.
   `package.vs` (D131) stays decided and unbuilt; its `package` keyword clash is to be resolved first.
8. **Compiler speed and code quality**: per-module IR caching,
   devirtualisation, escape analysis, panic-freedom analysis (§3.2).

## F — Concurrency: little memory and CPU, and control of OS threads

From the concurrency review (`archive/veles-concurrency-review.md`, 2026-10-09; user:
"very little memory and cpu when programming asynchronously … precisely
manage the native OS threads"). In order; each step green on Windows and
WSL, measured with `veles-bench` before and after, and run under the
thread stress (`VELES_THREADS=1/2/4/8`, `VELES_GC_THRESHOLD`, `--sanitize`
on Linux).

1. ~~**F1 B1**~~ **Done 2026-10-09.** O(1) channel waiter lists; spinlocks yield.
2. ~~**F2**~~ **Done 2026-10-09, short of its target** (task 448 → 248 bytes, in the 256 class; a parked
   task 657 → 400 bytes, not ≤ 250 — the rest is its frame and the argument block, which F3 and an
   arena for `async` frames take further; `bench/idle` not added yet). **F2 the task object (B4).** Flags packed, panic data in a side object,
   the debug call chain in debug builds only, one union for the wait kinds,
   small results inline. Acceptance: `bench/idle` added (peak working set per
   parked task) and ≤ 250 bytes per parked task (560–670 today); `spawn`,
   `channels`, `pipes`, `httphello` no slower.
3. ~~**F3**~~ **Done 2026-10-09** (4.4 ns per call that does not wait; `coro.elide` not needed).
   **F3 suspending calls in one task (B2).** A suspending call creates the
   callee's frame and resumes it inline; a callee that finishes without
   suspending costs no task, no scheduler, no `setjmp`; one that suspends
   suspends the chain, and the task resumes the innermost frame. Frames come
   from a per-task arena; `coro.elide` where a frame cannot escape.
   Cancellation still unwinds innermost first (D35 v0.29); panic traces and
   the debug call chain (D81) unchanged. Acceptance: `bench/suscall` (the
   review's probe) ≤ 5 ns per call that does not wait (70 ns today);
   `httphello` measured; the codegen golden `tasks` updated.
4. ~~**F4**~~ **Done 2026-10-09** (`withTimeout(100 ms)` around a plain loop: 100–400 ms at ≥ 2 threads,
   was the whole loop; a suspending loop gives one thread up within ~10 ms).
   **F4 D145** back-edge cancellation and yielding, `yieldNow()`,
   `checkCancelled()`. Acceptance: `withTimeout(200 ms)` around a plain
   loop returns within ~210 ms; the review's ticker probe on one thread
   ticks on time when the spinning task is a suspending function; no check
   in a lock region or `close()` (tests); `sha256`/`sort`/`json` benches
   within noise.
5. ~~**F5**~~ **Done 2026-10-09** (contended counter: `add` 130–153 ms vs `update` 266–376 ms;
   `hits.add(1)` is one `lock incq` in release).
   **F5 D144 atomics.** The operations, `MemoryOrder`, the compile errors for
   invalid orders, lock-free pointers; hover/completion entries; chapter 12
   and the low-level chapter. Acceptance: a lock-free stack test under 8
   threads; counter bench `add` vs `update`.
6. ~~**F6**~~ **Done 2026-10-09** (`TestSyncTypes` debug/release at 1/2/8 threads; docs chapter 12, references).
   **F6 D146 sync types** (`RwLock`, `Event`, `Lazy`, `Broadcast`/`Watch`), each
   with `race` support where it suspends. Acceptance: std tests, stress
   tests under threads, docs chapter 12 and reference/stdlib.
7. ~~**F7**~~ **Done 2026-10-10** (TLS sends close_notify at close as Go does — see the D147 addendum).
   **F7 D147 suspending `close()`**, then std: TLS `close_notify`, `Tx`
   `ROLLBACK` at close, `otel.start` flush at close; the three §11
   deviations removed. Acceptance: sema tests for the refusals (lock region,
   trait object), the two-instance generic case, driver tests against the
   real peers (Go TLS, PostgreSQL, the OTLP capture).
8. ~~**F8**~~ **Done 2026-10-10** (pools, single threads, `blocking`, `Thread`; latency kept on the default pool — see the log).
   **F8 D143 executors and threads** — the largest: the runtime's queues,
   timers, reactor and GC stop per executor; `[runtime]` in the manifest;
   `Executor`, `Executor.thread`, `scope(on:)`, `run`, `blocking`, `Thread`;
   priority/affinity on Windows, Linux (macOS with A7). Acceptance: a task
   on `Executor.thread` observes one OS thread id across 1 000 suspensions;
   CPU work on a pool does not move `httphello`'s latency on the default
   pool; `ThreadError` for a refused affinity; docs.
9. ~~**F9**~~ **Done 2026-10-10** (child start ≤ 136 µs on Linux; ns timers; race/sleep off the runtime lock; 32-thread `httphello` still slower than 8, as Go's is — see the log).
   **F9 the runtime lock (B5) and timers (B6).** Per-worker timer heaps,
   nanosecond deadlines, a lock-free race claim, one parking slot per
   worker; the Linux wake-up path (a launched child waits up to ~7 ms to start at 8
   threads on WSL, ≤ 0.3 ms on Windows; the review's `startlat` probe). Acceptance:
   `httphello` at 32 threads no slower than at 8; child start worst case ≤ 0.5 ms on Linux.
10. ~~**F10**~~ **Done 2026-10-10** (`httphello` as fast at 32 threads as at 8; CPU at 32 threads a quarter of before — see the log).
    **F10 memory, the collector and the last locks** (after F9, the user: "the concurrency
    topic must be done perfectly"): measure where the CPU and the allocation of a server go, and
    take out what is not the work. Acceptance: F9's `httphello` acceptance; `bench/idle` reports
    bytes per parked task; no runtime-lock contention left on a server's paths.
11. ~~**F11**~~ **Done 2026-10-10** (idle threads park each on its own; Windows timers to ~0.5 ms; CPU lists past 63 on Windows — see the log).
    **F11 parking and the last §11 items of F9** (one condition variable per executor, Windows'
    1 ms timers, processor groups). Acceptance: no runtime lock on idle or wake; sub-millisecond
    sleeps on Windows; `httphello` no slower.
12. ~~**F12**~~ **Done 2026-10-11** (249 bytes per parked task, was 345: the task is 152 bytes, the frame stayed
    apart — the diet alone reached the target — see the log).
    **F12 the task diet** (review B4, decision-free). The task object and its root frame in one
    allocation, the rarely used fields behind one pointer. Acceptance: `bench/idle` ≤ 250 bytes per
    parked task; `spawn` and `httphello` no slower; every concurrency test green on both systems.
13. ~~**F13**~~ **Done 2026-10-11** (and a smart cast set in a loop no longer outlives it — see the log).
    **F13 `await` after `cancel()` (D154).** Acceptance: a sema test with the dominating cancel
    (plain `await`, a `race` arm) refused with the line of the cancel, the cancel on one path
    accepted; docs (reference/concurrency's "Do not await a task you cancelled") rewritten.
14. ~~**F14**~~ **Done 2026-10-11** (and every `Executor.pool` was one thread on Windows: fixed — see the log).
    **F14 a pool hands off (D153).** Acceptance: a driver test where every thread of a 2-thread
    pool blocks in a C call while its other tasks keep running on spare threads, named like the
    pool; an `Executor.thread` blocked in a call keeps its tasks; Linux and Windows.
15. ~~**F15**~~ **Done 2026-10-11.**
    **F15 `runBlocking` and `Executor.default` (D152).** Acceptance: a `Thread`'s function and a
    `blocking` thread send to a channel through it; a compile error in a suspending function; the
    panic on an executor's thread (default pool, pool, one thread); docs.
16. ~~**F16**~~ **Done 2026-10-11** (116 ns a call, was ~450; a plain loop stopped on one thread).
    **F16 `withTimeout` in the caller's task (D151).** Acceptance: no task per call (`bench`
    measures one call: time and allocation); a plain loop under `VELES_THREADS=1` stopped within
    the limit plus ~10 ms; nested limits each report their own; an outer cancellation passes
    through; scopes inside `f` joined and `with` values closed before `Timeout`; a non-Sendable
    capture compiles; `httphello` no slower. (`net.Conn` keeps its buffer `Mutex`: an accept loop
    hands each connection to a task, so it must be `Sendable` anyway.)
17. ~~**F17**~~ **Done 2026-10-11.**
    **F17 declared executors (D150).** Acceptance: the D150 example compiles and runs (state on
    gl's thread — its thread id checked —, children capturing a `MutableList`, a call from main
    hopping to gl); errors for state named outside gl, a pool with state, a plain caller, a
    sendable lambda naming gl's state, `gl.close()`, a refused priority panicking before main;
    formatter round trip, hover "runs on gl", docs (chapter 12, reference/concurrency).
18. ~~**F18**~~ **Done 2026-10-13 on Windows; Linux marks in the pause (checklist §11).**
    **F18 the collector out of the pause** (user, 2026-10-13: "do 1, then 2"; decision-free —
    the runtime only, no language or codegen change). Measured first (scratchpad gcpause: 134 MB
    live in 4M pointer-linked nodes, small lists made and dropped, 8 markers, Windows): every
    pause 19 ms — mark 13, sweep 5.7; Go's under 1 ms. A store barrier does not fit: the
    coroutine frames are heap objects LLVM writes its spills into, and the runtime's C writes
    pointers in hundreds of places — one missed barrier is a use after free. The OS sees every
    write instead (Boehm's mostly-concurrent marking). Steps, each measured and soaked:
    (a) **lazy sweeping** — a span is swept when it is claimed, and helper threads sweep the rest
    while the program runs; the pause keeps the mark. (b) **spans from tracked regions** — large
    reservations the spans are carved from (Windows `MEM_WRITE_WATCH`), a page table readers can
    use while it grows. (c) **mostly concurrent marking** — a short pause takes the roots, the
    helpers mark while the program runs (objects made meanwhile are born marked), rounds rescan
    the marked objects on pages written meanwhile, and a short final pause rescans the roots and
    the last written pages. The written pages: Windows' write watch; Linux's soft-dirty bits when
    a self-test at start-up shows they work (stock Ubuntu/Debian kernels have them; WSL2's does
    not); elsewhere marking stays in the pause. A test mode that takes every page as written runs
    the protocol where the OS cannot tell. (d) **pacing** — a thread allocating fast while the
    helpers mark helps them instead of growing the heap without bound. Acceptance: gcpause's
    longest pause under 2 ms on Windows with the same live heap, throughput within 10 % of now;
    every test, the sanitized examples and a soak under `VELES_GC_THRESHOLD`/`VELES_GC_POISON`
    clean on Windows and Linux; `veles_gc_collect()` stays a full collection (the tests and
    bench/idle read exact counts after it).
19. ~~**F19**~~ **Built 2026-10-13** (all five parts); the migration of std/examples/docs and
    the `awaitall`/`awaitscope` benches done 2026-10-23 (progress log).
    **F19 concurrency with less ceremony (D156; decided 2026-10-13).** In this order,
    each its own green step: (a) `scope` as an expression and the error for a placed scope that
    starts nothing; (b) `val h = async f()` in any block with the must-await check (sema: a
    definite-await pass beside definite assignment; lowering: the with-task path) and the warning
    for an immediate await; (c) `awaitAll` (tuple and list forms; inline launches in a scope of
    their own); (d) `scope`/`gather(workers: n)` (runtime: a count and a wait queue on the scope;
    the launch is a suspension point); (e) lazy `iter.mapConcurrent` in the prelude and loops
    receiving a task-holding iterable. Then migrate std, examples, docs and benchmarks where a
    `scope` existed only to launch and await (D156's shapes), and the self-hosted parser.
    Acceptance: a conformance file per part with every error of the spec entry (a `return`/
    `break`/`continue` skipping the await, a branch, a loop, a race arm, passing/storing/cancelling
    the handle, a non-task `awaitAll` argument, `await awaitAll`, a no-launch `scope(on:)`, a
    scope value ending in a statement); driver tests under `VELES_THREADS=1/2/4/8`, debug and
    release: an abnormal exit cancels and joins an un-awaited handle before the error leaves
    (its `with`s closed), `awaitAll` cancels its inline siblings on the first failure, never more
    than `workers` children running (counted by the children) with `on:` and without, the lazy
    `mapConcurrent` keeps order and runs ahead (timed), a `break` out of its loop cancels and joins
    the calls in flight; formatter round trip; hover texts; `veles-bench`: `spawn`, `suscall`,
    `httphello` no slower, and a new bench of `awaitAll` over two tasks against the `scope`
    form; docs chapter 12, reference/concurrency, reference/errors, cheatsheet.
20. **F20 D197 `cede` and removal of `yieldNow()` (specified, not built).**
    Implement the reserved, statement-only scheduler operation per
    `veles-cede-spec.md` sections 1-7. Both frontends, AST/printers, sema
    effect propagation and context checks, HIR/codegen/runtime lifecycle,
    removed-form diagnostics/safe fixes, formatter/LSP/editor grammars and
    live std/examples/bench/docs migration land together. Keep zero sleep,
    automatic loop yields and `checkCancelled()`; no compatibility alias.
    Acceptance: the specification's syntax/effect/cancellation/fail-fast/
    shielded-cleanup/affinity/name-resolution matrix; selfhost parity;
    debug/release under 1/2/4/8 workers and GC stress, supported Linux
    sanitizer and Windows/Linux/macOS runs; before/after yield cost and
    allocations at 1/8 workers plus idle/spawn/channels/suscall/httphello
    regressions; full build/vet/tests and tested docs. No ordering-dependent
    expected output. Record unavailable gates in checklist section 11.
    User requested specification first: do not start this implementation
    instead of the next generator-research/planning discussion (I1).

21. **F21 Is D145's loop yield right? (measure; decided 2026-10-26 with
    `suspend loop` rejected).** The user: "we'd need check if the current
    approach is good". Measure, with `veles-bench`, on Windows and Linux
    at 1 and 8 workers:
    - (1) the latency a ticking task sees beside a tight suspending loop
      (the D145 review's 1.3 s case), against the 10 ms slice;
    - (2) a one-thread executor (D150) running a long loop;
    - (3) a plain, non-suspending loop, which cannot yield;
    - (4) the throughput cost of the back-edge check in a tight loop;
    - (5) once F20 lands, `cede` per iteration against the automatic yield.

    Acceptance: numbers in `bench/results.md`; a verdict on the slice (keep,
    change, or make it per executor, which would be a `veles-decide`
    question); any gap goes into checklist §11.

---

## Progress log

Older entries: `archive/progress-log-2026-09.md`.

### 2026-10-27 — B32 step 6 and `noalias`: value layout complete

- **Exact GC maps for tagged values**: a sealed value's descriptor lists
  the union of its variants' pointer words (a boxed variant: its box), an
  error union its members', instead of every payload word (golden
  `gcmaps`: `Token` scans 2 words of 3, `Failure` 1 of 2).
- **`@align(n)` everywhere**: `Layout.Placed` puts tuple elements and a
  nullable's value at their guaranteed alignment, and the code generator
  pads those LLVM types as it pads structs (`placedType`, through `fidx`
  and `overAlign`); a sealed or error payload starts at
  `Layout.PayloadOffset` (8, or an over-aligned variant's alignment), the
  tag then lying in an `[n x i8]` read through memory, so the payload stays
  at index 1. The runtime keeps a non-8 unwrap offset in the task's rare
  block, so the task stays 152 bytes. Where nothing is over-aligned the IR
  is byte-identical (every other golden unchanged).
- **`noalias nocapture readonly`** on memory-class parameters; the
  borrowed-argument probes (closure, pointer, holder, list, `this`
  writing the original mid-call) agree in debug, release and
  `VELES_THREADS=8 VELES_GC_THRESHOLD=256 VELES_GC_POISON=1`.
- **Bug fixed (older than this branch)**: a suspending function's frame
  came 8-aligned from the collector or arena, while LLVM lays it out for
  its most aligned field and assumes the storage is that aligned (D120's
  note said it would realign; LLVM 17 does not). A release build of a
  128-byte `@align(64)` local kept across a suspension crashed at HEAD;
  frames now take `llvm.coro.align` (`veles_alloc_words_aligned`, arena
  padding folded away when the frame needs only a word).
- Benchmarks (Windows, A/B against HEAD, three pairs): `suscall` 119 ms
  both after the padding fold, `trees` 146 → 143 ms, `awaitscope` and
  `bigargs` within noise.
- Golden fixtures `gcmaps` and `overalign` (the latter also run in
  `--release`); checklist §11 entry closed, the parked string/sealed
  niche kept there.

### 2026-10-27 — Review of the `struct-layout` branch before its merge

The branch (B32 steps 1–5, Go 1.27.1) was built and tested on macOS only.
Now on Windows and Linux (WSL): build, vet and every package green on
Linux; on Windows two failures, neither from the branch (below). Probe
programs, debug and `--release`, `VELES_GC_THRESHOLD=256` +
`VELES_GC_POISON=1`, `VELES_THREADS=1/8`: niche nullables (`S?`, `S??`,
`Some(null)`, narrowed writes, maps, channels, tasks, `throws`, sealed
payloads, closures, empty-trait objects, `Array<S, 1>?`, constant tables);
borrowed big arguments against every aliasing route (a receiver that
writes beside a borrowed argument, `v = f(v)` through `sret`, an argument
that assigns `v`, a lambda that writes it, generics, virtual and indirect
calls, receiver blocks, tasks); `compact struct` (generic, `@align`,
constants, equality, sets, 2 000 values under GC stress); D214 holes
(loops, `break`, `catch`, `when`, lambdas). All correct. Fuzzed 90 s each:
`FuzzParse` and `FuzzCheckMutated` clean, `FuzzFormat` found one.

Fixed (checklist §11): a plain string spanning lines kept `\r` on a CRLF
checkout (`examples/todo` printed it on Windows; both lexers), the
formatter copied that `\r` into LF output (`TestCorpus`), and `veles fmt`
was not idempotent on a one-line receiver block holding a `when`. Raised:
Q63 (definite assignment through `break`; the search loop is refused).
Flaky: `std/http`'s "closed while idle is replaced" failed once in a full
WSL run (38 passes alone and under load after): the server's 100 ms idle
wait also bounds a new connection's first request; margins widened, the
server question in §11. After the fixes: Windows build, vet and every
package green; WSL green but for that one test, which passes on rerun.
Recorded: B32 step 6 and `noalias` as open (§11). Benchmarks on Windows,
A/B against `spec-driven`: no regression (`channels` 4.9 ms both, after
three reruns); `bigargs` 0.3× Go; recorded in `bench/results.md`.

Docs: decided design notes archived (`veles-concurrency-review.md`,
`veles-direction-2026-10-14.md`, `veles-syntax-decisions.md`, references
updated); two NUL bytes in this file (which made git and grep treat it as
binary) replaced by `\0`; a stray `test.vs.bak` removed; "Where to
continue" rewritten as a checklist at the top of this file.

Examples (standing direction R5: examples do real work): the 17 real
programs (tools, servers, parsers) stay at the top of `examples/`, beside
`hello` and `tutorial`; the 50 one-feature programs and `syntax_tour.vs`
moved to `examples/features/` (the old `features` program is now
`features/tour`). Same tests, same expected output; references updated.

### 2026-10-26 — Value layout researched; D214, D215 decided (plan B32)

`veles-layout-spec.md`: how values are laid out today, measurements and
proposed rules. Measured on macOS arm64: reordering fields would save 0.4 %
of struct bytes in `selfhost`; no sealed type in the tree would gain from a
smaller tag; passing a 144-byte struct costs 2× the address of the caller's
value (96 B: 1.3×, 40 B: none), and today's memory class (copy, then
address) is slower than plain by-value. Bugs found and put in checklist
§11: an uninitialized local `var` makes null non-nullable pointers and
handles (now D214), a constant table of an `@align` struct fails in clang,
the memory-class threshold differs from D121, a runtime offset parameter is
ignored. Decided: D214 (definite assignment for locals), D215 (declaration
order guaranteed, `compact struct`).

**Step 1 built the same day.** `structSlots` (`codegen/llvm/types.go`) is
the one list of a struct's LLVM members; the type and constants are built
from it, so a constant table of an `@align`, aligned-field or packed struct
compiles (`TestConstantTablesOfPaddedStructs`, debug and release). `isMem`
starts at 128 bytes, as D121 says. `types.PayloadOffset` feeds the GC
descriptors and the fail-fast launch; `veles_task_set_unwrap` stops with an
internal error on any other offset (the task struct has no room to store
one without moving to the next size class). The nullable shape is known
only to `makeNullable(T)`, `nullFlag`, `nullVal`, `nullPayloadPtr`,
`storeNullFlag` (and `constInit`, which step 4 changes). The goldens are
unchanged and `selfhost`'s IR is byte-identical before and after. `go test
./...` is green but for three failures that are already at HEAD on this
Mac: `TestStdModuleTests/compress` (exit 255, checklist §11) and the two
Unicode-table tests (the local Go has Unicode 17). Not run: WSL, Windows,
the GC/thread stress.

**Step 2 (D214) built the same day.** A local `var` declared without a
value carries `noInit`; an assignment to the whole variable sets the flow
fact `$assigned` (kept by `invalidatePlace`/`invalidatePaths`, so a method
call or `&` does not drop it); a read, `&`, a field write, a method call,
`x += 1` or a capture before the fact holds on every path is an error
(family `declarations`). Inside a lambda an assignment before the lambda
counts, one in the lambda counts only there. Pinned by
`sema/testdata/conform/D214-definite-assignment.vs`; docs chapter 2, the
errors reference and the cheat sheet. Nothing in std, examples, selfhost or
the docs tripped it. A second full run on this Mac also failed
`TestHashFlooding`, `TestReactor*`, `TestTLSServer`, `TestStdHttpUnitTests`
and `TestStdRegexUnitTests` (segfaults, a "list was frozen" panic); the
same tests fail in a worktree at HEAD, so they predate B32 (checklist §11).

**Those crashes fixed the same day (macOS only).** Cause: `os_decommit`
(`runtime/c/veles_gc.c`) gives a dead large object's pages back with
`MADV_DONTNEED`, and a span later carved from that place is taken to read as
zeros; on Darwin the call keeps the bytes, so new objects began with stale
words (a list header's frozen bit, old pointers). Found with `veles-debug`:
`VELES_GC_OFF=1` made `std/compress` pass, a scratch test reproduced it
under `VELES_THREADS=1`, and the frozen header came straight out of
`veles_list_new`, which relies on zeroed memory. On macOS the range now gets
a fresh mapping (`mmap` `MAP_FIXED`), with a `memset` if that fails.
`TestReleasedMemoryReadsAsZeros` frees large lists full of set bits, then
fills their place with small lists (fails without the fix). `go test ./...`
is green on this Mac but for the two Unicode-table tests.

**Go pinned to 1.27.1 on every machine (user, the same day).** `go.mod` says
`go 1.27.1`, so an older Go switches to it; `internal/macos-test.sh` and
`internal/wsl-test.sh` set `GOTOOLCHAIN=go1.27.1`. The generated tables
(`selfhost/lexer/tables.vs`, `std/regex/unicode.vs`) were regenerated from
its Unicode 17.0.0 data (they were Unicode 15.0.0, from Go 1.23.2). README,
docs chapter 1 and CLAUDE.md say so. `go vet ./...` is clean under the new
language version and `go test ./...` passes on macOS arm64. Not run: WSL and
Windows, which now fetch Go 1.27.1 on their first build.

**Step 3 (large arguments without a copy) built the same day.** Measured
first: `bench/bigargs` (a 144-byte struct and a struct holding an
`Array<i64, 16>`, each passed to a `@noinline` reader 20 M times; Go
reference alike) ran 88 ms against Go's 90. (a) The receiver pass sets
`Var.ReadOnly` on a parameter whose place the body never writes or lends
(`selfUse`, which now also sees an `Array` element written or lent to C
through `withRaw`); a memory-class parameter is read in the caller's
storage when read-only and copied on entry otherwise. The caller passes its
own variable (or a field of it) when nothing can change it during the call
— not global, not captured, not address-taken, and no other argument of the
call writes it (`sema.MayWrite`) — for direct, indirect and virtual calls;
task launches still copy. Codegen panics if a borrowed parameter is ever
written. 88 → 53 ms. (b) Every aggregate over 64 bytes joins the memory
class (`wideThreshold`): 53 → 46 ms (0.5× Go); the other benchmarks
unchanged within run-to-run spread (A/B against HEAD, three runs each);
`selfhost` builds 9 % faster in release and its binary is 6 % smaller. The
`arrays` and `layout` goldens change as intended (two 264-byte copies gone;
`Holder` in memory). `TestLargeArgumentsKeepValueSemantics` pins value
semantics through every aliasing case of `veles-layout-spec.md` §6.3
(debug and release, same output as before the change). `go test ./...`
green; examples under `VELES_GC_THRESHOLD=256`, `VELES_GC_POISON=1` and
`VELES_THREADS=1/8` green; results recorded. `noalias` on the parameter is
not added yet. Not run: WSL, Windows.

**Step 4 (nullables laid out as their value) built the same day.**
`types.Layout.Niche` finds the first word of a type that D214's validity
makes never zero — a non-raw pointer, a collection, channel or task
handle, a closure's code, a trait object's table — through struct fields,
tuple elements and an array's first element (extern, packed and union
structs, strings, numbers, nullables and sealed values have none). A `T?`
of such a type is `T`, all zeros for null: `Of`, `llType`, GC descriptors,
constants and the five nullable helpers (`nullFlag` compares that word
with null; `makeNullable` selects the value or zeros) — no other code knew
the shape (step 1). Audited first: every way an all-zero value of a type
can be observed (`init` placeholders are refused before assignment, D214
for locals, scratch slots only read when written, the two std raw reads
only of stored values); `Atomic` takes only pointer nullables. Pinned by
`types/layout_test.go` (14 shapes, offsets and sizes) and
`TestNullablesLaidOutAsTheirValue` (structs, closures, trait objects,
tuples, arrays, memory-class values, `T??`, narrowing writes, `?.`, `?:`,
`==`, sets, maps, pop, generics, sealed payloads, constants, Deque; debug
and release under `VELES_GC_POISON`); a second probe (channels, tasks,
`Mutex`, JSON round trip, constant tables, `copy`, `withTimeout`) gave the
same output as before under 1 and 8 threads and GC stress. In a probe
program the `{ i1, … }` wrappers fell from 417 to 99. Benchmarks unchanged
(A/B against HEAD); `selfhost`'s peak memory parsing std unchanged (its heap
is held at `gc.minHeap`). `go test ./...` green; examples under GC and
thread stress green; `selfhost` under `VELES_GC_THRESHOLD=4096` green. Not
run: WSL, Windows.

Step 5 (D215, `compact struct`): the contextual keyword parses before
`struct` at top level and in sealed bodies (after `indirect`/`inline`), in
the Go and the Veles front ends (G-gates green, `std/lint/ast.vs`
regenerated); formatter and AST dump print it. `types.Struct.Compact`;
`layout.go` places a compact struct's fields by alignment, descending
(stable), codegen's `structSlots` follows offset order and remaps field
indices. Refused: `compact` with `extern` (so with `@packed`) and with
`@transparent`; a warning under two fields. Hover shows size, alignment and
padding, and what `compact` would save. Tests: `D215-compact.vs`,
`TestStructLayoutLine`, `TestCompactStructsKeepDeclaredOrder` (debug and
release, GC stress). Docs: chapter 13 section, errors.md, cheat sheet, LSP
keyword doc, VS Code grammar. `go test ./...` green. Not run: WSL, Windows.

### 2026-10-24 — C12: tests that can run in the compiler run there too (D208) — built

`sema/compile_tests.go`: `compilerCanRun` decides a test or helper (the
`const fun` rules, quietly, memoized; a test is declared throwing, so its
body decides); after the last round of `veles test`'s check the checker runs
each such test in the evaluator, whose `test.fail`/`stop`/`skip` record
instead of erroring and whose panics and stops become failure lines like the
runtime's; over the step budget a test runs in the program only. The runner
prints `ok (also in the compiler)`, or fails a test the compiler failed and
the program passed ("FAILED in the compiler only …"), or adds "in the
compiler it passed" under a program failure. `@SkipIf(helper())` evaluates
a runnable helper; one that is not says why. `VELES_COMPILER_TESTS=why`
explains a refusal. 24 of the prelude's tests now run both ways and agree.
Tests: `TestTestsRunInTheCompiler` (both ways, program only, a helper in
`@SkipIf`, a disagreement through `sema.SkewReplaceForTest`),
`TestConstantCallsTestHelper`. Docs: chapter 14, cheat sheet.

Acceptance: the checker decides per test (and per `test fun`) whether the
compiler can run it — the `const fun` rules, inferred, no keyword;
`veles test` evaluates those in the interpreter and runs them in the
program, one report: `… ok (also in the compiler)`, a one-sided failure
naming the side and the first `expect` that differs; `@SkipIf` accepts a
helper the compiler can run (the D113/D78 error stays for one it cannot,
saying why); the step limit as for constants; tests (a test that diverges on
purpose through an evaluator hook, a test that cannot run in the compiler,
a helper in `@SkipIf`); std's const helpers exercised both ways by their
existing tests; docs (chapter 14, errors reference).

### 2026-10-24 — C11: receiver blocks (D207) — built

`fun on T(A): R` in both parsers, the formatter, types (`types.Func.On`:
the first parameter is a pointer to the receiver's place); `{ … }` alone
and trailing after a call or a name (`ast.LambdaExpr.Bare`,
`ast.CallExpr.Trailing`), the formatter writing the trailing form and
keeping a one-line block on one line; a receiver block checked with `this`
its first parameter and bare members resolved against it
(`resolveBlockMembers`, `SelfExpr.Implicit/Outer` for an enclosing
method's members); `sema/apply_edit.go`: `apply`/`edit` on every type,
the literal block inlined (`fnCtx.recvStack`), `return`/leaving jumps
refused, the heap decision deferred to the receiver pass
(`settleOnCalls`); `StringBuilder.trim/toLower/toUpper/replace/insert` on
new runtime functions (`veles_bytes_*`) with evaluator twins; the edited
string taken from the builder's bytes without a copy. Measured
(`--release`, 20M calls): `p.apply { x += i; y -= 1 }` 0.33 ns per call,
the same as the two assignments (a closure was ~25 ns);
`text.edit { trim(); toLower(); replace(" ", "-") }` ~400 ns, as the
chain `trim().toLower().replace(…)` on a 32-byte string. Tests:
`D207-receiver-blocks.vs`, `std/prelude/builder.test.vs` (9 tests, all
also in the compiler), `parser/receiver_test.go`, a formatter case, the
self-hosted front end byte-identical on all of it. Docs: chapter 10
"Receiver blocks", cheat sheet, errors reference, hover for `on`,
`apply`, `edit`; the editor grammar colours `fun on`. Left: checklist §11.

### 2026-10-24 — C11: blocks on a receiver (D207) — decided, after C10

Amended the same day (D207 addendum): receiver blocks are passed trailing,
`x.apply { … }`, `configure(&s) { … }`, `measure { … }` (a same-line `{`
after a call or method name, filling a last `fun on T()` or `fun()`), in
both parsers; the formatter writes the trailing form; docs say "receiver
block" / "receiver function".

Acceptance: the type `fun on T(...): R` in both parsers, the formatter,
hover; a `{ … }` lambda (or `(a) => { … }`) checked with `this` the
receiver, bare members reaching it (D158); std `apply` (in place; a `val`
refused with a fix to `edit`) and `edit` for string (a mutable text with
in-place `trim`/`toLower`/`toUpper`/`replace`/`insert`), List, Map, Set
and structs, one copy each (measured); conformance and runtime tests; docs
(closures chapter, reference, cheat sheet); the self-hosted front end.

### 2026-10-24 — C10: enums as text and predicates (D205, D206) — built

Built as accepted below: `std/prelude/case.vs` (`Case`, `Enumerated`, the
casing helpers, in Veles), `sema/enum.go` (`parse` with a case through
`enumPick`, `toString(case:)`, `isM()`, overrides through `implement
Enumerated for E` — its receiver passed as a method's, which a first try got
wrong and crashed — collisions under `Case.Any`), `KeyStyle` → `Case` by a
scratchpad tool over std, examples and docs (`KeyStyle.SnakeCase`,
`codec.KeyStyle` and the type are errors with fixes, `sema/removed.go`),
`std/build` reading the compiler's words with `Case.Any`, `std/cli` reading
enum flags in any spelling and listing their choices. Tests: two conformance
files, `std/prelude/case.test.vs`, the KeyStyle conformance file, a cli test.
The same day, decision-free: `toLower`/`toUpper`/`replace` copy once (or not
at all), hover on built-in generics and an inline loop's name, keyword hover
for `implements` and `test fun`, annotation targets highlighted.
`std/time/stopwatch.test.vs` was updated to the user's staged change making
`measure` a free function.

Acceptance: prelude `Case` and `Enumerated`; `E.parse(text, case:)` (exact
default, `Any`, one form), `e.toString(case:)`, `e.isM()` folding to a
constant; collisions under `Any` refused; `implement Enumerated for E`
overriding `parse`/`toString(case:)`; `KeyStyle` → `Case` across std, docs,
examples, with the old name an error and a fix; `std/build` without
`osNamed`/`archNamed`, its tests with `isWindows()`; `std/cli` reading enum
flags with `Case.Any` and listing their choices in kebab case; conformance
and runtime tests; docs (enums chapter, reference, cheat sheet).

### 2026-10-24 — C9: `std/cli`, on `std/build` and D203's reflection

The user confirmed D167's names (one `@Flag(short:, name:, env:, hidden:)`
over five annotations; `cli.parse<T>` throws, `cli.args<T>()` prints and
exits 0/2) and, when starting showed D174 could not serve it, decided D203
and D204. Built in that order:

- **`std/build`** (D160's module, D204): `build.target` (`Os`, `Arch`,
  `release`) and `build.program` (`name`, `version` from the manifest),
  constants from two std-only compiler calls (`sema/buildinfo.go`).
  `@SkipIf(build.target.os == …)` works. Found on the way: an annotation
  argument that calls a `const fun` crashed the compiler (the instance table
  did not exist yet while declarations were collected) — fixed.
- **D203** (`sema/reflect.go`): `f.default()`, `f.isOptional`, `f.isList`,
  `f.Inner`; `if` and subject-less `when { }` keep only the branch handle
  constants decide (`f.name == "x"` too); `T.members` for enums and sealed
  families (`MemberInfo`, and `m.Type`/annotations in an `inline loop`);
  every enum `Parsable`; `T.doc` for a type's doc comment. `T.build` in a
  trait default body now sets the implementing type's non-visible fields,
  as `f.get` reads them (a library could not build a user's struct). The
  parser kept no doc comment on a variant nested in a sealed trait (Go and
  `selfhost/`, so hover lost it too) — fixed in both.
- **`std/cli`** in Veles: flags in kebab case, switches with `--no-`, `T?`,
  `List<T>`, positionals (a list takes the rest), short groups and attached
  values, `--`, environment fallback, every problem at once with the
  nearest flag or command, help from doc comments with defaults, `--version`
  from `build.program`, subcommands as a sealed family. Tests:
  `std/cli/cli.test.vs` (10), `TestCliProgram` (another package, exit codes,
  stderr, env), `TestReflectionForLibraries`, `TestBuildModule`, the D203
  conformance file. `examples/dedup` and `httpd` moved to it (their
  transcripts gained `--help` and a typo; the example runner now shows
  stderr after stdout); `todo` did not fit (checklist §11). Chapter 15 has a
  section, the stdlib reference two modules.
- **Std tests in the suite**: ten std modules' `*.test.vs` (prelude,
  crypto, toml, config, …) were run by nobody; `TestStdModuleTests` now runs
  every module's that has no test of its own.

### 2026-10-24 — A flaky driver test under load: `TestLessCeremony`

Found by running `./driver` beside the heavy packages on Linux (one failure
in four runs; the full suite had failed once): in `stopped()` the release
build on 8 threads printed `every started call closed: false`. The runtime
was right — the test asserted that at least two calls had started when the
loop broke, which its own comment says depends on scheduling: the first
call ends at once, and on a loaded machine the loop broke before any other
call ran (a call cancelled before it starts never runs). The loop now waits
until a second call has started before it breaks, so the break always
cancels a call inside its `with` — what the line checks. Eight loaded runs
of the test (each debug and release on 1, 2, 4 and 8 threads) pass.

### 2026-10-24 — S17 / D199: a source file must be UTF-8

- **Both lexers** check the text before lexing it: every byte that does not
  decode (in the steps of Go's `utf8.DecodeRune` — overlong forms,
  surrogates, past U+10FFFF and a truncated sequence included) is an error
  `the text is not valid UTF-8 here (byte 0xE9): save the file as UTF-8`
  (family `syntax`), and nothing more is lexed, so the two agree byte for
  byte. The self-hosted front end reads bytes: `source.readSource` keeps
  each bad byte's offset and value on `File` (`notUtf8`) and a `?` in its
  place in `content`, so every offset is the disk's.
- **Gates**: the corpus no longer drops such files and fuzz findings; four
  hand-written ones (a comment, a string, code with overlong / surrogate /
  past-the-end / truncated sequences, after a BOM) pass every mode but the
  three that print text. `TestNotUTF8` pins the Go lexer's offsets.
- **Users**: the excerpt under the error shows U+FFFD for each bad byte (the
  caret stays aligned); the language server analyses the disk's bytes while
  the buffer is that file as the editor decoded it (`TestNotUTF8OnDisk`);
  `veles fmt` leaves such a file alone. Errors reference and chapter 1.
- **Found on the way and fixed**: a format spec on any integer but `i64` and
  `u64` failed to compile (`"${b:02X}"` with `b: u8`: "no method
  'toString'"); a narrower integer now widens first
  (`std/prelude/format.test.vs`).

### 2026-10-24 — S18–S20 / D200–D202: the three choices left by the parse work

Decided by the user the same day (recommended for D200 and D201; Kotlin's
`startIndex` over the recommended `at:` for D202), then built:

- **D202** `string.startsWith(prefix, startIndex)`: checked in sema (named
  or positional; a wrong name or count is an error naming the form), folded
  in constants, and emitted **inline** — a bounds test, a first-byte
  compare, `memcmp` for the rest. A runtime call per try was tried first and
  made the parse _slower_ (616M → 679M instructions: the lexer tries 59
  operators at a position, and the hand loop it replaced rejected on the
  first byte inline). The self-hosted lexer's `holdsAt` is gone.
- **D200** `[runtime] gc.minHeap` / `gc.goal` (also as `[runtime.gc]`):
  both manifest readers with the same refusals (gate R1 has the cases),
  passed to `veles_runtime_gc` at start-up, `VELES_GC_MIN_HEAP` /
  `VELES_GC_GOAL` override (`VELES_GC_THRESHOLD` kept as the old name), the
  goal now a percent in `set_goal`. `TestRuntimeGCManifest` counts
  collections (default 9, 1 GiB none, goal 1000 vs 10: 4 vs 25).
  `selfhost/veles.toml` sets `gc.minHeap = "32MiB"`.
- **D201** a field moves out of a local whose methods were called:
  `sema/move_method.go` summarises each method from its source (keeps
  `this`? which fields it uses, which it may hand out), the union over the
  methods it reaches; `movableField` consults it. A list pattern that binds
  no rest now counts as reading elements (for locals too). Accepted and
  refused cases are in `TestMutableToImmutableMove`. The self-hosted lexer
  hands the parser its token list (`tokenizeForParser`, a labelled tuple)
  and both copies per file are gone; `redundant_copy` found two more.

**Measured.** Self-hosted parse of std 616M → **507M** instructions (Go's
front end: 1,024M). `bench/parse` **0.7–0.9× Go on Windows** (four runs,
loaded machine; was 1.1×), **0.7× on Linux**. Windows and WSL suites green.
macOS not run. Left: checklist §11 (D200–D202).

### 2026-10-23 — The Windows gap on `bench/parse`, measured by instructions

The machine was loaded (a game) while this was done, so wall-clock numbers
were compared only interleaved (a scratchpad tool runs binaries in turn
and reports min and median) and the work was guided by instruction counts:
valgrind's callgrind, unpacked into the WSL home like perf, is the same
however loaded the machine is.

**What the counts said.** The self-hosted parse of std is 616M
instructions; the Go front end doing the same work is 1,024M (its operator
scan alone, with `memeq`, ~230M against our 118M). Veles is not slower by
code — it is ~40% leaner. Where its time goes: the lexer's operator scan
19%, `memcpy` 12% (two whole-list copies of the tokens per file — the
port's `toList()`/`toMutable()` — 7%, and list growth 5%), allocation 13%,
marking 10%.

**Changed (compiler and runtime only).**

- A list element's address uses the element type's size, known at compile
  time, instead of loading the stride from the header and multiplying.
- The sweep reads a dead object only when its span may hold a `Secret`'s
  bytes or poisoning is on (`span.wipes`), and clears the marks with one
  `memset`.
- `veles_gc_alloc` reads the thread's block once, not four times; a reused
  slot of a small class is cleared with a constant-size `memset` (a few
  stores). `veles_gc_alloc`'s own instructions 71M → 40M.
- Tried and reverted: a word-at-a-time search for a free slot (more
  instructions — the runs are short); `memory(read, inaccessiblemem:
readwrite)` on the back-edge poll's call (LLVM hoisted nothing and the
  scan grew).

Interleaved on Windows: 3–4% faster (min 114 → 110 ms under that load).
The Windows ratio to Go could not be measured cleanly under the load (the
runner showed 0.6–0.9×, Go suffering more). What remains, by experiment:
with a 32 MiB collection threshold, or none, the parse is ~5% faster —
Go marks on other cores, a Veles collection of a heap this small marks in
a pause. Left for the user (checklist §11): a larger minimum heap (memory
for speed in every program), D188's move extended to a field of a dead
local (removes the two token-list copies, 7%), and a std `startsWith(at:)`
the lexer could use.

### 2026-10-23 — S16 / D198: trees are boxed

**Syntax.** `indirect struct` and `inline struct` in a sealed trait's body,
`indirect sealed trait` before a declaration — contextual words, in the Go
and self-hosted parsers (`StructDecl.Indirect/Inline`, `TraitDecl.Indirect`),
`veles parse`, the formatter, the editor grammar; `std/lint/ast.vs` and the
span walker regenerated.

**Checking.** `markRecursive` (`sema/boxed.go`) sets `Sealed.Recursive` on a
sealed type one of whose variants reaches it again through a pointer, a
collection, a nullable, a tuple or array, another struct's fields or
another sealed type's variants (`Stmt` → `Block` → `List<Stmt>` included).
The rule for which variants are boxed lives in `types.Layout`
(`VariantBoxed`, `SealedPayloadWords`), shared by the checker and codegen:
`indirect`, or larger than a word in a tree unless `inline`. Lint
`large_variant` (registry, both sides) warns of a non-tree, non-generic
sealed type whose largest variant is over four words and over twice the
next, with a fix that writes `indirect`; it fires nowhere in std,
examples or selfhost. Hover says `indirect`/`inline`, or "boxed: 'Expr' is
a tree".

**Codegen** (`codegen/llvm/boxed.go`). Every sealed value is built and read
through `makeTaggedT`/`extractTaggedT`, so a boxed variant is allocated
there (its own descriptor and size) and loaded through the pointer; the
payload words are conservatively traced already. A place reached through
a narrowed boxed variant (`placeAs` of a `VariantCast`) loads the pointer;
for a write or a pointer that outlives the access it copies the box first
and stores the copy in the value — unless the variant has no `var` field,
when nothing can be written. A probe's output is identical with the HEAD
compiler (markers removed).

**Speed.** Every `selfhost` AST union is now `{ i32, [1 x i64] }` (16
bytes; `Expr` was 180, `Decl` 348). The parse alone did not move at first
(≈85 ms on Windows): the parser's functions dropped out of the profile
(`parsePostfixOp` 7% → 1%), and what was left was the lexer and the
collector. On Linux that is parity: 63–70 ms against Go's 64–65. On
Windows it was 78–80 ms against 61: the write-watch concurrent marking,
which started at 4 MiB live, cost a fast-allocating program ~10 ms (with
`VELES_GC_CONCURRENT=0`, 69 ms). `CONCURRENT_FROM` is 16 MiB now — a pause
marks that much in about a millisecond — and the Windows parse is 69–74 ms,
1.1×. `gcpause`'s longest pause is unchanged (1.39–1.42 ms at 4 MiB,
1.32–1.45 ms at 16 MiB, same session); `ast` and `json`, A/B against HEAD
alternately, within noise.

Tests: conformance `D198-boxed-variants` (the lint, `indirect`, `inline`,
`indirect sealed trait`), `D198-boxed-syntax`; `driver/boxed_test.go`
(value semantics — a write through a narrowed copy, equality, a set,
printing, an `indirect` variant written through a `var` — and deep trees
built in tasks under `VELES_GC_THRESHOLD=65536`, debug and release, 1 and
4 threads); `parser/boxed_test.go`; formatter round trip; `lsp/boxed_test.go`;
codegen golden `boxed` (the box allocated, read through, copied before a
write). Docs: chapter 9 "Trees: where a variant is stored", cheat sheet,
errors reference.

### 2026-10-23 — S8: debug information

A debug build now carries DWARF (`codegen/llvm/debug.go`), so gdb and an
editor's debugger work on Veles source. The checker gives every lowered
statement the position it was written at (`Pos` on the HIR statements,
`ValuePos` for a block's last expression, set once where an AST statement
is lowered and on the returns made from expression bodies); codegen opens a
`DISubprogram` per function, attaches the current statement's
`DILocation` to every instruction it emits (`emit` appends it), and
declares locals and parameters with `llvm.dbg.declare` and a type: base
types for numbers and `bool`, a `string` as `{data, len}` (gdb prints the
text), a struct with its fields at `StructPlan`'s offsets, a pointer to its
target; other types by name and size. Helpers and the entry point carry
none. The driver asks for it when the build is not `--release`
(`Program.DebugInfo`); the goldens leave it out except
`codegen/llvm/golden/debuginfo`, whose metadata `ModuleIR` renumbers by
first use with the directory hidden.

Windows is `x86_64-w64-windows-gnu`, whose gdb (MSYS2) and VS Code's
C/C++ extension in gdb mode read DWARF, so CodeView was not needed.

**Found on the way.** The "Debug Info Version" flag makes clang verify the
module, which the driver otherwise skips (`-disable-llvm-verifier`); every
async program failed: `llvm.coro.end` was declared with the token argument
clang 18 added, which clang 17 rejects. It is the two-argument form now,
which clang 18 upgrades (checked on both); goldens regenerated. Every debug
build is now verified by LLVM as a side effect.

Tests: `driver/debuginfo_test.go` runs gdb in batch mode over a program —
a breakpoint by line and by function name, arguments, `next` onto a block's
last expression, locals (a struct, a string's text, numbers), and a
suspending function after its `sleep` on another thread; it skips without
gdb (`VELES_GDB` names one off PATH). Green on Windows (MSYS2 gdb 15) and
Linux (gdb 15.1, unpacked without `sudo`). Docs: chapter 13 "Stepping
through a program: a debugger".

Left (checklist §11): a local that lives across a suspension loses its
description when LLVM moves it into the coroutine frame; contents of
lists, maps, sets, nullables, sealed values and closures; std's embedded
files cannot be shown; lldb on macOS not run; no line information in
`--release` builds.

### 2026-10-23 — S7: the self-hosted front end's speed, first pass

**Measured first.** `bench/parse` times the self-hosted lexer and parser
over std's 155 files against the Go front end (`selfhost --bench-parse
std`; the runner's `programs` table builds `selfhost/` for it). Start: 164
ms against 55 ms, 3.0×. Lexing alone was 88 ms (Go 40), parsing 76 (Go 15).
Profiled on Linux with `perf` (no `sudo` needed: the packages unpacked into
the home directory) and LLVM's IR-level entry counts through a `VELES_CLANG`
wrapper.

**Fixed, all the compiler's or the runtime's.**

- `s.byteAt(i)` was a call into the C runtime — 16 M calls over std. It is
  now a compare and a load, the runtime called only to panic: 150 ms;
  `bench/lexer` 85 → 53 ms.
- Every allocation cleared its slot, though a span comes from the OS
  zeroed and a large one goes back when its object dies: growing a big list
  touched each page twice. A span now knows up to where its slots have
  never held an object (`virgin`) and clears only below: 130 ms; most of
  the suite gains a little (table in `bench/results.md`).
- `MutableList.insert` and `removeAt` shifted element by element through
  `at`/`set`; they are one `memmove` now (`listMove`, a std-only builtin
  the constant evaluator runs too). No visible change here — the parser
  inserts rarely — but no longer O(n) calls of a generic `set`.

**Fixed in the port, as its plan said to.** The self-host plan's §5 named
"span arithmetic" as a risk: a `Span` holding its `File` by value is a copy
per node. It did: a span was 64 bytes and an `ast.Expr` 380. `Span.file` is
now `(*File)?` (24 bytes, the null niche), which took the parse to 84–91 ms,
1.5–1.7× Go. `std/lint` keeps the by-value file — rules run in the
constant evaluator, which builds spans that way (the generator says so).

**Left, and why.** An `ast.Expr` is still a 180-byte inline union and a
`Decl` 348 — D12 lays a sealed value out inline, sized for its largest
variant, so every node returned or boxed moves that much; Go's interface is
16 bytes. Changing that changes D12's promise of no allocation, so it is
the user's (checklist §11). The target of ≤ 1.3× Go is not reached.

### 2026-10-23 — S6: the span gate and P6 `Layout`

**Every span compared.** G2 compares `ast.Dump`'s text, which carries no
span. `selfhost/spans_test.go` adds a gate over the whole corpus (970
files): each side lists `Node.field start end` for every span-typed field
of every node, sorted, each once (the Go parser shares some nodes), and the
lists must be equal. The Go side walks the tree by reflection; the Veles
side by `selfhost/ast/spans.vs`, generated from `selfhost/ast/ast.vs`
(`go test ./selfhost -run SpanWalker -update`; the test fails when it is
stale), run by `selfhost --spans`. Two nodes are named apart (`File` is
`SourceFile`, `SelfExpr` `ThisExpr`) and one field (`AnnotationInfo.For` is
`readers`). Found by it: the Veles `ExecutorDecl` kept no span of its
arguments (`argsPos`, which the Go formatter reads) — added, and
`std/lint/ast.vs` regenerated.

**P6 `Layout`.** `parseFileLayout` returns the tree, the comments and the
spans of parenthesised expressions, as `parser.ParseFileLayout` does — the
input a formatter written in Veles needs (plan S9). `selfhost --layout`
prints them; `TestP6Layout` compares with Go over the corpus. Green.

**Not done: files that are not valid UTF-8.** None is in the repository;
one fuzz finding of ten is. A Veles `string` is always UTF-8 and
`fs.readFile` refuses such a file, while the Go lexer reads the bytes and
reports a character or a string literal it cannot decode — and accepts an
invalid byte in a comment. Matching it means either a byte-level `File` in
the self-hosted front end, or both compilers refusing a non-UTF-8 file up
front; the second changes what the compiler accepts, so it is the user's
(checklist §11).

### 2026-10-23 — F19's migration: scopes that only launched and awaited

The `scope` blocks whose only job was to start named tasks and await them
now use D156's shapes: `std/net`'s stream test, `std/otel`'s child-span test
and `examples/streams` hold a `val server = async …` in the function's own
block; `http.Health`'s `runAll` is `scope { …; awaitAll(tasks) }` as the
function's value; `examples/httpclient` fetches its three pages with one
`awaitAll`. Left as they are on purpose: scopes of unnamed children (the
scope is what joins them), scopes that cancel a handle (`serve`'s
`serving.cancel()`, `examples/features/cancel`), fan-outs over a loop, the
`TaskLocal` test (its `await async` is the point), and chapter 12's first
example, which introduces `scope` before the sections on the lighter forms.
`concurrency-explained.md` no longer says a task can only start in a
`scope`: its first program has no scope, the JavaScript table maps
`Promise.all` to `awaitAll`, and "nothing floats away" names both owners.

New benches `awaitall` (two inline launches joined by `awaitAll`) and
`awaitscope` (the same pairs in a `scope` with two handles), one Go
reference (two goroutines and a `WaitGroup` per pair). Windows, default
threads, four runs: 40.6–44.8 ms against 41.0–41.7 ms — the same cost; Go
63 ms. Recorded in `bench/results.md`.

### 2026-10-23 — B22: labelled tuples (D166)

**Syntax.** A tuple type or literal labels every element or none
(`(quotient: i64, remainder: i64)`, `(key: k, value: v)`); mixed labels and
a repeated label are syntax errors. `ast.TupleType`/`TupleExpr` carry
`Labels` (nil when unlabelled; `ast.Clone` regenerated); `veles parse`
prints them; the formatter lays a labelled literal out like named arguments
(a broken one keeps its breaks); a labelled one-element tuple needs no
trailing comma. The self-hosted AST, parser and printer do the same
(G-gates green; `std/lint/ast.vs` regenerated).

**Checking.** `types.Tuple.Labels` is for reading and display: `Identical`
and `Key` ignore it, so layout, `==`, hashing, ordering and the generic
instance are the unlabelled tuple's. `t.label` reads an element (`t.0`
still does); an unlabelled literal takes the expected type's labels.
`types.LabelsDiffer` finds two different label lists in the same position —
at the top or inside lists, maps, sets, nullables, tasks, channels and
function types — and `coerce`/`assignableTo` refuse it (family
type-mismatch, both types named); at the top the fix drops the value's
labels (deletes them from a literal, rebuilds a name as `(p.0, p.1)`). A
function instance holds no labels (`types.StripLabels` on its type
arguments, so what it shows cannot depend on which caller came first); the
call gives its caller the labels back by substituting the template's result
type with the caller's bindings, so `entries().sortedBy(…).first()` is a
labelled entry. The `Result` instance of a throwing call is shared too, so
the HIR `Call` keeps the labelled success type (`Ok`), which `try`, `??`,
`catch` and `?!` read (`okType`). Generic struct and sealed instances hold
none either; writing labels in a generic type's arguments
(`Box<(lo: i64, hi: i64)>`) is refused with the unlabelled spelling, rather
than dropped where `box.value.lo` would then fail (checklist §11).

**Text and constants.** `"$t"` prints `(quotient: 3, remainder: 1)`:
codegen's display helpers are cached by `types.LabelKey`. The constant
evaluator prints a tuple, an optional and a list by the static type, as run
time does (a case in `driver/constfun_test.go`, debug and release).

**std.** `Map.entries()` gives `(key: K, value: V)`, `List.enumerate()` and
`Iterator.enumerate()` `(index: i64, value: T)`, `splitOnce`
`(before: string, after: string)?`. Moved to labels: `examples/dedup`,
`todo` (its rows are `(number: i64, task: Task)`), `json`, `regex`, `fuzz`,
`session`, `std/http/cookie.test.vs`, docs chapter 12. Regenerated:
`examples/features/maps/expected.txt` (entries print their labels),
`std/testdata/const-functions.txt`.

**Tooling.** Hover shows the labels (they are in the type's text);
completion after `t.` offers each label beside its position.

**Found on the way, fixed.** (1) A tuple key in a map literal,
`[(1, 2): "a"]`, was read as a lambda's parameters (`(…):` was taken for a
return type): `looksLikeLambda` now requires a type and `=>` after the
colon, in both parsers. (2) `implement Codable` on a struct with a tuple
field said "no method 'encode' on type '(string, i64)'"; the derive now
names the field and says a tuple has no wire form (a list, set or map of
tuples too).

Tests: conformance `D166-labelled-tuples`, `-syntax`,
`-generic-type-labels`, `-tuples-not-codable`; `sema/labelled_tuple_test.go`
(both fixes compile; one instance serves three labellings);
`parser/tuple_test.go` (labels, their errors, tuple keys vs lambdas);
`format` `TestLabelledTupleRoundTrips`; `lsp/tuple_test.go`; codegen golden
`labels` (the labelled and unlabelled `divmod` compile to the same body, one
`swap` instance, labels only in the text helper). Docs: chapter 3 "Several
results", chapter 4 "Tuples" with the tuple-vs-struct guideline, cheat
sheet, stdlib and errors references. `go build`, `go vet` clean; staticcheck
reports only its three earlier unused symbols. `go test ./...` green on
Windows and on Linux (WSL) — the first full runs since B19 and B21, which
were therefore verified on both here too.

### 2026-10-23 — B21: one-parameter constructors, `copy`, `i64?` index-of

**D163.** A constructor that takes exactly one parameter from the call site
— fields `init` does not assign and the caller may see, plus `init`'s
parameters — takes it positionally (`soleParam`, `sema/check_call.go`), in
construction and in generic inference (`Mutex(0)` is a `Mutex<i64>`); not
for `extern struct`s. The count follows visibility: `Mutex`'s private
defaulted `word` does not count from outside, does inside. A positional
argument records no field reference in the index, so a rename leaves the
call alone. When a type grows a second parameter the existing D28 error
fires, its text now naming both rules, with the fix that writes the name.

**D164.** `v.copy(field: e, …)` lowers to a construction from a temporary of
v (`copyStruct`): named fields from the arguments, the rest read from v,
`init` run again through the same helper construction now uses
(`runInit`). Refused: positional arguments (puns allowed), unknown or
repeated fields, fields `init` assigns, fields the caller cannot see, type
arguments, task-holding types (D111), `with`-bound resources (D100),
`extern union`s, and types whose `init` takes parameters — the last two
gaps in D164, decided the same day by the user as built (refused; D164
addendum v0.105). A hand-written `copy` or a field named `copy`
(warned at the field) means no generated one. Hover shows the signature for
the call site (`fun Rect.copy(w: i64 = this.w, h: i64 = this.h): Rect
(generated)`), completion offers it. Found on the way: the constant
evaluator had no case for `Zero`, so no type with an `init` block could be
built in a constant; it now can, and `copy` with it.

**D165.** `List.indexOf` (built in), `indexOfFirst`, `string.indexOf`,
`lastIndexOf`, `binarySearch`, `binarySearchBy`, `binarySearchWith` answer
`i64?`; `contains` tests for null. `val i = xs.indexOf(v) ?: <leaves>` puts
i in range of xs (D62, `foundIndexOf` in `sema/bounds.go`); `?: 0` does not.
`… >= 0`, `< 0`, `== -1`, `!= -1` on a call of the family is an error
(family removed-syntax) with the fix to `!= null`/`== null` — `== -1` would
otherwise compile and always be false. Migrated: std (prelude string,
format, duration, codec; path, http, db, regex, toml, version), examples
(httpd, codable, algorithms), selfhost (lexer, parser, linter, pkg, text),
docs chapters 2 and 4, references, cheat sheet; most `>= 0` tests became
`contains`, most "find then check -1" became `?: return`. Regenerated:
`examples/algorithms/expected.txt` (`-1` → `null`), the `strings` codegen
golden (IR only), `std/testdata/const-functions.txt`. `veles-bench`, three
runs each against a HEAD worktree on Windows: `sort` 35.6–36.9 ms vs
37.3–39.6 ms, `strings` 57.9–60.0 ms vs 56.0–61.8 ms (medians 58.7 vs
58.1) — no slower.

Tests: conformance `D163-positional-constructor`, `D164-copy`,
`D165-index-of-nullable`; `sema/copy_test.go` (visibility across modules
for both), `sema/index_family_test.go` (both fixes compile), a `copy` case
in `driver/constfun_test.go` (compile time = run time, debug and release),
LSP hover (`lsp/copy_test.go`) and completion.

### 2026-10-23 — Embedded std parsed once per process

Checklist §11 "Test-suite wall time", point (2). A profile of
`TestCheckEveryPrefix/httpd` (27 s wall, 69 s CPU) put 13 s in loading —
12 s of it parsing std — 11 s in `Check` and the rest in GC, which parsing
fed most. The checker writes into the tree it is given: D158 replaces bare
member names in every method body by `this.name` through reflection, D176
replaces a decorated function's body, so a parsed tree cannot be shared
between checks (a data race under the LSP's concurrent loads, a D158
shadowing warning lost on the second check, and a rewrite that depends on
the program's own impls on std types). `sema/stdparse.go` parses each
embedded std file once and hands every load a deep copy; `ast.Clone` is
generated from `ast/ast.go` (`go generate ./ast`, `ast/internal/clonegen`;
a test fails when it is stale) and keeps the sharing the parser makes (a
struct's impls are also in the file's declarations; desugared declarations
share a `NamedType`) — `ast/clone_test.go` checks over every std and
repository source that the copy shares nothing with the original and shares
within itself exactly what the original does. Prelude: parse 13.7 ms /
15.8 MB, copy 2.3 ms / 2.3 MB. A first reflective copy took 7.5 ms and cost
more than it saved under load.

Measured on Windows: `go test ./sema` 384 s → 168 s;
`TestCheckEveryPrefix/httpd` 27.5 s → 17.3 s (CPU 69 → 31 s); a language
server edit of `examples/httpd/main.vs` loads in 9 ms instead of 45 ms, but
checking it (`CheckIndex`) takes ~590 ms either way — recorded in §11 as
"Checking the embedded std once", with what caching checked std would take.
`go test ./lsp` unchanged (~10 s; it is not load-bound). `go build`, `go vet`
clean; `go test ./...` green on Windows (8 min wall) except two failures
that come from a CRLF checkout (`core.autocrlf=true`), not from this change:
`format` `TestCorpus` and `examples/todo` fail on CRLF copies of
`examples/regex/main.vs` and `examples/todo/main.vs` and pass on LF ones
(checklist §11). Linux (WSL): build, vet and `go test ./...` green; `go
test -race` clean on `ast` and on `sema`'s concurrent-load test and httpd
prefixes.

### 2026-10-22 - B19 / D157: reject unhandled values in interpolation

**Implemented; cross-platform completion remains pending in checklist 11.**
Strings, format specs and template values reject a Result or task handle.
Result diagnostics name propagation and fallback; throwing functions receive
a mechanical `try` edit, including expansion of `$name` to `${try (name)}`.
CLI repair is idempotent and the LSP offers the same preferred edit.
Explicit `Result.toString()` is catalogued for tooling and lowers to the
existing display HIR; try-chain resolution recognizes it as a Result method.
Task diagnostics direct the caller to `await`. No parser, codegen, runtime or
std API change was needed. The D157 generic declared-parameter exception and
test capture remain accepted.

Regression tests were observed failing before implementation. Conformance,
accepted forms, edit round trips, CLI debug/release execution and LSP quick
fixes pass on macOS arm64 with Go 1.23.2. Existing checker fixtures, two task
examples, an executor test and executable documentation now inspect Results
explicitly, with their expected output unchanged. Final `go build ./...`,
`go vet ./...` and `go test ./...` all passed on macOS arm64 with Go 1.23.2.
The new fixture was formatted and its line-sensitive expectations rechecked.
Staticcheck is not installed; Windows/Linux are not available for reruns.
The initial runtime timing failure,
not reproduced in an isolated HEAD test or the next full driver run, is tracked
in checklist 11 rather than treated as fixed.

### 2026-10-21 — Cede specification, implementation pending

**D197 specified, not built.** `veles-cede-spec.md` records primary-source
research (Kotlin, Tokio, Go, Python asyncio, Rust threads), the reserved
statement-only `cede`, cancellation/cleanup/executor guarantees, explicit
scheduling non-guarantees, removal of the prelude `yieldNow()` with safe
migration, all-layer ownership and the acceptance matrix. Existing driver
cancellation/deferred-cancellation/caller-task-timeout regressions passed
on macOS with Go 1.23.2, as did `TestAtomicOperations` (existing `yieldNow`
callers). New syntax was not tested or implemented; no full build/vet/test,
runtime benchmark or Windows/Linux run was performed. F20 remains pending.
Next discussion: resume generator research/planning (I1), not generator
implementation; D197 does not decide a generator design.

### 2026-09-28

**Records cleaned up.** Finished plans and decided design notes moved to
`archive/` (`veles-build-plan.md`, `veles-review-fixes.md`,
`veles-guard-design.md`, `veles-testing-design.md`); the previous plan
with its log, and `notes_to_change.txt` as it stood, kept verbatim there.
The notes file now holds only open items (the user's text unchanged;
status lines marked `→`). Checklist §9 rebuilt as the open decisions
Q1–Q14 (the old list appended to the archived log); stale items ticked
(orphan rules — none beyond D17; `Deque`/`PriorityQueue`/`Set` operations)
or marked `[?]` with their Q; the duplicated panic-freedom item merged;
§11 cleared of the limits fixed on 2026-09-27 and given the two found then.

**User directions added (2026-09-28).** Four principles now lead this
file (complete, performant, two levels, towards self-hosting) and are in
CLAUDE.md and the `veles-decide` preparation list. New track S
(self-hosting readiness, S1–S5). Checklist §9 gained Q15 named imports
(reopens R9) and Q16 `as` no longer meaning conversion, first in track D.
A1 runs in WSL Ubuntu 24.04 (present; Go and clang still to install); A7
is the user's M4, later.

**A1 done: Linux x86-64.** Go 1.23.2 installed in WSL user space
(`~/.local/go`, no sudo; clang 18 was already there);
`internal/wsl-test.sh` syncs the Windows working tree into a clone at
`~/veles` and runs the suite there. Green: `go build/vet/test ./...`,
every example under `VELES_THREADS=1/2/4/8` ×5, `VELES_GC_THRESHOLD=256`,
`VELES_GC_POISON`, the driver ×3; the `posix_spawnp` branch of `os.run`
compiled and its test passes. Found and fixed: (1) the runtime did not
compile — glibc hides `pthread_getattr_np` without `_GNU_SOURCE`, now
defined at the top of every runtime file on Linux; (2) **Windows wrote
stdin/stdout/stderr in C text mode**: every `\n` printed became `\r\n`
and stdin ended silently at the first 0x1A byte — binary mode now, so a
program's output is the same bytes on every platform (driver
`TestStandardStreamsAreBytes`, red before); every `expected.txt` is now
plain LF; (3) two doc samples were platform-dependent: glibc's `cbrt(27)`
is 3.0000000000000004 (the FFI sample now calls `ldexp`, exact by
definition), and the cancellation sample raced a 1 ms sleep against a
child's start (it now waits on a channel; docs 12 says a task cancelled
before it starts never runs). Bench recorded from Linux: single-threaded
work matches Windows; `spawn` is 3.9× Go against 1.1× on Windows —
checklist §3.1 item.

**Typo suggestions (B1, first part).** `sema/suggest.go`: every unknown
name, function, method, field and module member says what was probably
meant — edit distance with transpositions, ignoring case, long prefixes
(`toUppercase`→`toUpper`), ties to the longest shared subsequence
(`printn`→`println`), and other languages' names (`size`→`len`,
`append`→`push`, `get` on a List→`at`, `has`→`containsKey`,
`reduce`→`fold`, `lenght`→`len`) offered only when the type has the
target; private members and statics are never suggested. A field called
as a method, or a method read as a field, is told which it is. The guess
is a `source.Fix` with `Guess` set: an LSP quick fix that is not
`isPreferred`, never applied by `check --fix`. Two old annoyances fixed
alongside: a local named only through a typo was "never used", and
`check --fix` renamed it to `_`; a failed call's lambda arguments added
"cannot infer the type of lambda parameter" errors. Tests:
`sema/suggest_test.go`, driver `TestCheckFixLeavesGuesses`, lsp
`TestCodeActionTypoGuess`; docs 01.

**A2 done: sanitizers.** `veles build/run/test --sanitize` compiles the
C runtime with `-fsanitize=address,undefined -fno-sanitize-recover=undefined`
at -O1 (its own object cache) and links the sanitizer runtimes; the Veles
code is not instrumented (bounds-checked already; its heap is the
collector's). Two runtime changes make it sound: the collector's
`scan_range` is `no_sanitize("address")` (it reads redzones on purpose),
and `__asan_default_options` turns off use-after-return fake frames, which
would keep C locals off the stack the collector scans. Test flags:
`go test ./examples -sanitize` (collector every 4 KiB; a sanitizer report
fails the example) and `go test ./driver -sanitize` (the runtime stress and
`veles test` programs). On Linux: every example at threads 1/2/8 ×2–3 and
the driver ×3 clean; one finding, fixed — `buf_append` in the test
runner's capture did `NULL + 0` on an empty print (UB; red under
`go test ./driver -sanitize` without the fix). A deliberate `memset` past a
`malloc` block is reported with the Veles frame in the stack (driver
`TestSanitizeCatchesOverflow`, skipped where clang lacks compiler-rt, as in
this MSYS2). Docs 01 and 13 (a "When C goes wrong" section). Found on the
way: D50's raw-pointer arithmetic (`p + n`, `p - q`, ordering) is decided
and documented but not built — next.

**D50 raw pointer arithmetic built.** Decided and documented (docs 13's
table said "`unsafe` only, C-style") but `p + 4` was "operator '+' is not
defined" — found while writing the A2 probe. Now `p + n` / `p - n` /
`p += n` step by elements (a plain `getelementptr`, not `inbounds`: D50
promises no checks and no assumptions), `p - q` counts elements (byte
difference over the element size), and the orderings compare addresses
(`icmp ult`…), all inside `unsafe`. Errors name the rule: outside
`unsafe`, `n + p`, a non-integer step, `*raw ()` (no size — cast to
`*raw u8`), mixed pointer types. sema/rawptr.go; tests
`sema/rawptr_test.go`, golden `codegen/llvm/golden/rawptr` (IR and run);
docs 13 has a runnable sample; `examples/features/ffi` now turns `bsearch`'s
result pointer into an index with `p - q`.

**A3 done: D21 verified, four edge bugs fixed.** The golden test now runs
a fixture in a second profile when it has `main.release.ll`: golden
`overflow` is one source that prints "integer overflow" (a panic caught by
`gather`) for `127 + 1` (i8), `0 - 1` (u8), `MIN * -1`, `-MIN`,
`MIN / -1`, `MIN.abs()` and `x += 1` in debug, and the wrapped values in
release; `+%`, `as` truncation and the range cases print the same in
both. Bugs found probing type edges: (1) **an inclusive range ending at
its type's maximum looped for ever** in every profile (`loop (i in
250..255)` over u8: the index wrapped to 0 and `i <= hi` held) — the loop
lowering and the prelude's `RangeIter` now stop on a flag set at the last
value, Rust's `RangeInclusive` approach; bench before/after (3 runs each,
all workloads) shows no change, LLVM folds it; (2) `Range.len()`,
`step()` and `RangeStepIter.reversed()` measured spans in `T`:
`(-128..127).len()` panicked, `(-128..127).step(100)` silently yielded
just `[-128]`, and an exclusive range ending at MIN counted as nearly
full — spans are wrapping distances now (negative only past half the
type, where no step is too long); (3) `MIN.abs()` returned MIN silently in
debug while `-MIN` panicked — now an overflow like it (catalogue updated).
Three ergonomic questions went to §9 Q17 (a one-task `gather` is a
1-tuple; `async f()` on a function value; `Range.isEmpty`), and two gaps
to the checklist (generic prelude bodies unchecked until used; no
inference of a type argument from the expected type).

**A4 done: the conformance suite, and the cascades it exposed.**
`sema/conform_test.go`: case files under `sema/testdata/conform/` (20,
by D-number) end each offending line with `// error: text` or
`// warning: text` (several per line allowed); a case must provoke
exactly what it lists. Coverage is by diagnostic format: the test parses
the checker's source for every `errorf`/`warnf`/`errorFix`/`warnFix`
format (string constants resolved), records what a full `go test ./sema`
reported (a hook in the four Checker functions, set only by the test's
`TestMain`), and fails for a format nothing provoked that
`uncovered.txt` does not list — so a new diagnostic comes with a case,
and a covered one must leave the list (`-conform-update` rewrites it,
keeping `# why` notes). 267 uncovered at the start → 65: 33 with a
reason (parser-guarded, std-internal intrinsics, two-module M5 rules,
defensive AST cases), 32 still to write. `.gitignore` ignored every
`testdata/*`; the conform directory is excepted (it would have been left
out of the commit). Writing the cases found **13 checker defects**, all
fixed and pinned by the cases: cascading second errors after a generic
method's type parameters differ from the trait's, after a value-`if`
without `else`, after a generic argument that cannot fit, after `when`
arms of incompatible types, after a lambda parameter declared with the
wrong type, after every `race` arm was refused; names bound by a refused
`when` pattern, `is` pattern or let-else destructuring reported as
unknown or never used; `Some`/`None` on a non-nullable subject reported
as "'Some' is not a type"; a function or lambda body ending in a call
with no value reported "expected 'i64', found '()'" instead of "missing
return"; `try`/`throw` in a global initializer blamed an "enclosing
function"; the needless-`throws` lint firing on a body that failed to
check; and `sealed trait X : Named` without methods silently dropping its
supertraits.

**A5 done: closing a socket while another task uses it.** `std/net`
wraps the number in a `Socket`: every `accept`/`read`/`write`/
`shutdownWrite`/`port` holds it for the call (`with (held =
fd.using())`, a count in one atomic with a closing bit); `close()` sets
the bit, and the runtime's new closing set (`veles_io_closing` /
`veles_io_closed` in `veles_task.c`) wakes every task parked on that fd
and refuses new waits, so they fail with an `IoError` at once. The
number is released by whichever of `close` and the last in-flight call
finishes second — a call can no longer reach a number the system handed
to a newer socket. Red before: driver `TestSocketCloseDuringRead` hung
(a read parked on a closed socket never woke, verified with HEAD's
`net.vs`); green after, 200 rounds at `VELES_THREADS` 1 and 8, on Windows
and Linux, and ×3 under `--sanitize`; `httpd`/`shutdown`/`cancel` ×5 at
2 and 8 threads. §11 limit removed, chapter 16 says what `close()` from
another task does.

**A6 done: the HTTP request parser, fuzzed over a real connection.**
Not through `http.call` (it never parses bytes) but through `serve`
itself: `examples/fuzz` runs `http.serve` in the same process on a
loopback listener with small `Limits`, and each run writes one to three
requests on a fresh connection and closes its side. The last request
carries one of 21 faults, each with the status RFC 9112 asks for (414,
431 ×3, 413, 400 for framing and syntax, 501, 505), or the whole write is
mutated; the client parses everything that comes back and fails on a
malformed response, a 500, a refusal that keeps the connection, a wrong
echo of method, decoded path or body, or a read that waits (5 s — the
client has closed, so the server has no reason to). Against HEAD's server
the first 500 runs failed 225 times. Found and fixed in `std/http`:
**response splitting** — a path decoding to `\r\n`, echoed into a header
by the handler, let the client write headers into the response; values
are now written with CR/LF/NUL as spaces, names that are not tokens are
dropped, and `content-length`/`transfer-encoding`/`connection` belong to
the server (a handler's `connection: close` is still obeyed). A request
line or header that is not UTF-8 closed the connection without an answer
(now 400). HTTP/1.0 was kept alive by default. And the head was read
leniently where a proxy could read it differently: space before a colon,
folded lines, non-token names and methods, CR or NUL inside a value,
`Content-Length: +5`, two differing lengths, a missing or doubled `Host`,
`HTTP/1.1x` — all 400 now. The ten inputs are a corpus checked every run
(all ten fail at HEAD). 30 seeds × 3000 runs at 1/2/4/8 threads on Linux:
0 failures. On Windows, a long sweep runs out of ephemeral ports (closed
client ports sit in TIME_WAIT for 120 s) — the default 500 runs are well
inside that. Chapter 17 says what the server refuses and why.

**B1 done: every diagnostic names its fix and its family (D79).**
The user chose named families (recommended of 4) for the docs link.
`source/family.go` sorts every message into one of 34 families by its
wording; the CLI prints `see: veles explain <family>` once per family
under the errors, `veles explain <family>` prints that section of
`reference/errors.md` from the copy compiled in (`docs/errors.go`;
offline, version-matched, "did you mean" on a wrong name), and the LSP
sends the family as `code` with the GitHub anchor as `codeDescription`.
`errors.md` is rewritten as one section per family, the old entries
under theirs, and the panics in their own part. Two tests hold it: sema
`TestEveryDiagnosticHasAFamily` fills each of the 612 formats of
sema/parser/lexer with sample arguments and needs a family for it (it
caught the first table matching formats but not real messages — a `%s`
in a pattern now stands for any text); docs `TestEveryFamilyIsExplained`
needs a section per family and no stray one. The audit, rule: a message
names what to write, unless its fix is simply undoing the one fact it
states, or it is compiler-internal. About 45 messages rewritten — a
private name says to declare it `public` there (or that a std name is
not API); a built-in's arity quotes its signature (`'at' takes 1
argument: at(i: i64): string?`); a literal that does not fit names its
type's range; `%` on floats points at `mod`; a bare member inside a
method says `this.twice` (a certain fix, not a guess); `Type.method`
says it is a method; a method implementing a trait method it does not
have gets "did you mean"; a global initializer that suspends or opens a
scope says to compute it in `main`; missing fields, impossible patterns,
unreachable code, missing returns, a value from a unit function,
`break` outside a loop, labels, Sendable, generic functions as values —
each says what to write. Found on the way: `true - false` told the user
to implement `Subtractable` for `bool`, which no program can do — the
"implement it" hint now appears only for the program's own structs and
sealed traits; `atOrPanic()` and a non-comparable constant pattern each
reported one mistake twice. Families in chapter 1, the command table,
the index, and the veles-test / veles-docs skills.

**A4 closed: every diagnostic has a case or a reason.** The 32 left
without a case got one or a written reason: 10 new case files (sealed
variants, trait objects, `when` values, constructors and members, `init`
reads, removed reads and indexes, error unions, the old `@test`, tasks
in globals, operators) and two package tests for the two-module rules
(`TestPrivateAcrossModulesSaysHow`, `TestExternCSymbolIsUnique`); 46
listed, every one with the reason no program reaches it. The test now
also fails for a listed entry without a reason. Writing the cases found
a **missing check**: a variant of a sealed trait with no `implement` of
the trait's methods was reported only when a call dispatched to it (and
then naming whichever variant came first, even one whose method sat one
level too far out) — it is now an error at the variant's declaration,
with the method to add, or "move it inside 'implement Shape { }'".
Hit a wall: `T.Item` with two bounds declaring `Item` silently takes the
first bound's; refusing it needs a way to name the other, which is
language design — checklist §9 Q18.

**B2 done: `--timings`.** `veles build|run|test|check --timings` prints
each phase (load with files/bytes/modules, check, codegen with the IR's
size, the runtime objects, clang with its flags) and the ten modules
that cost the most to parse + check (`sema.LoadPackageTimed`; the loader
times each parse, `drainQueue` each body, charged to the module it is
declared in; nil when not asked). Driver tests pin the report and the
duration format. First reading, `examples/httpd` debug: 939ms total —
clang 853ms on 2.7 MB of IR, the whole front end 27ms. So build time is
IR size: 630 std functions make 2.3 MB, and one body can be large
(`http.readRequest`, ~100 lines, is 109 KB of IR) because lowering
writes adapters, checks and interpolation inline — which makes B8 (the
eager adapters as prelude functions) a build-time item as well as a
runtime one (checklist §3.2).

**B3 done: tests run at once (D80), and helpers are traced through
`async` and panics.** The user chose parallel by default (recommended of
4). The runtime keeps one record per test (failures, captured output)
bound as a task-local value of the test's task, so the tasks a test
starts record into it; the runner queues every test (`veles_test_queue`:
up to `--jobs`, default one per runtime thread; a test that ends starts
the next), then waits for them in declaration order with `veles_run`
and prints each report whole — the output is what it was, test by test.
The watchdog watches every running test; its report says how many others
had not finished and that `--jobs 1` runs them one at a time. Found on
the way: `veles_run` reset the executor's clock base on every call (it
was only ever called once per test, sequentially); the queue and the
records were reachable only from C globals the collector does not scan —
`VELES_GC_POISON` crashed `--jobs 1` until they became roots. Helper call
sites: a test helper launched with `async helper()` is traced (the site
is bound around the launch, so the task inherits it), and a panic inside
a helper lists the test's calling lines under its `at` line — both §11
limits gone. Pinned by `TestTestsRunAtOnceReportInOrder` (four sleeping
tests overlap, `--jobs 1` does not, reports in order with each test's
own output), `TestTestHelperPanicCallSites`, the async case in
`TestTestHelperCallSites`. Windows and Linux; Linux also under
`--sanitize`, `VELES_THREADS` 1/2/4/8 ×3 and GC poison.

**B6 done: "Add the missing methods".** An implement that leaves out
methods of its trait carries, on each of its errors, one fix that adds
them all — each as `fun name(...): T = panic("'name' is not written
yet")`, indented into the block whether it was `{ }` on one line or a
body with methods in it; a sealed variant with no `implement` of its
trait (the check added under A4) gets the whole block the same way.
Scaffolding, so it is a guess: the editor offers it, not preferred, and
`check --fix` never applies it. `TestAddMissingMethodsFix` applies each
fix and checks the result compiles. The B1 audit left nothing else for
the language server. Docs: chapter 8, the VS Code README.

**B7 done: `veles new --template server`, and live stdout.** The
template is an HTTP service that builds, runs and passes its tests as
created: a Router with `/healthz` and a JSON endpoint, `requestId` and a
5-second `timeout` wrapped around every route, request logging (serve's
own), HOST/PORT from the environment, `http.serve(..., stop:
os.shutdownSignal())`, and three tests of the handlers through
`http.call` — no port. `veles new` names the templates (`app`, `server`)
and refuses an unknown one; the old hint named a test (`greetsByName`)
that had been renamed. Driver `TestNewServer` checks, tests and builds
it, starts it with PORT=0, reads the address from its first line and
asks `/healthz`. That first line found a **real bug**: on Windows the
line never arrived — C's runtime ignores `_IOLBF` there and fully
buffered stdout, so a Veles service under Docker or a supervisor would
log nothing until it stopped. Standard output is now written at each
line end unless it is a regular file (measured: a million lines to a
file stay at 0.18s; flushing each would have been 2.1s), on every
platform; Linux had been line-buffering even files. `TestNew` now runs
the app template's test too, not only a check.

**B8 measured, and blocked.** The prelude form of an eager adapter —
`fun mapV<T, U>(xs: List<T>, f: fun(T): U): List<U>` with a push loop,
and the same for `filter` — against the Go-lowered `xs.map(...).filter(...)`,
5M elements ×10, `--release`: 1.15s against 0.78s, **45% slower**, stable
over three runs. The lambda stays an indirect call nothing inlines, and the
result grows by pushing where the lowering allocates it once. So B8's own
rule says: do not move them. What would make the move free is a way to
specialise a generic function for the function value it is called with —
a compiler pass (per-call-site specialisation of higher-order generics,
which LLVM then inlines), or a language form like Kotlin's `inline fun`,
which is a decision. Until one exists the adapters stay lowered (and are
most of the IR size measured under B2). Also found writing the benchmark,
and recorded for their own decisions: `MutableList` has no `reserve(n)` (public API: asked separately). Two
diagnostics were wrong and are fixed, with cases: `List.generate(n, ...)`
was told to write `List<T>.generate`, which does not exist either — it now
says "no static function 'generate' on type 'List'; did you mean
'MutableList<T>.make(...)'?" (the nearest static of the type or of its
mutable counterpart, other languages' builder names leading to `make`);
and a lambda passed after an argument that had failed was reported as
"cannot infer the type of lambda parameter" — the same mistake twice, now
checked loosely.

**B8, measured again with `reserve` (D83): speed is not the blocker.**
With `out.reserve(xs.len())` in the prelude form, `mapV`+`filterV` over
5M ×10 at `--release` ran 0.53s against the lowered adapters' 0.56s (three
runs each) — the 45% earlier was regrowth, and LLVM does inline the lambda
once `mapV` is inlined. The real reason the adapters are lowered in Go is
effects: a lambda given to `map`/`filter`/`fold`... may throw
(`args.map(a => try eval(a))` types as `Result<List<U>, E>`, lower_try.go)
or suspend, and the lowering carries that out of the loop. A prelude
function would have to be generic over its argument's effects —
`fun map<U, E>(f: fun(T): U throws E): List<U> throws E`, with `E = Never`
meaning "does not throw", and the same for `suspends`. Next step when B8's
turn comes back: find whether the checker already instantiates `throws E`
with `Never` to a non-throwing function, and what suspension-generic would
mean; if either is a language change, it goes through `veles-decide`.

**Decisions D81–D84 (user, all recommended), and three of them built.**
Asked together: panic call chains (D81, a shadow stack in debug builds —
after the user asked for DWARF to be explained next to it), `os.run`
input and stderr (D82), list capacity (D83), and Q18 (D84).

- **D83 `xs.reserve(n)`**: a built-in (`veles_list_reserve`), in the
  catalogue and hover, allowed on a local that is later moved (D63's
  method list), refused on a `List`; chapter 4's `build(n)` uses it.
- **D84**: `T.Item` with two bounds declaring `Item` is an error naming
  both and the qualified form; `T.Trait.Item` resolves through that bound
  (a run returns each implement's value). Conformance
  `D84-associated-types.vs`; chapter 8.
- **D82 / B5 done**: `os.run(program, args, input: "", stderr:
os.Stderr.Capture)` with `Capture | Inherit | Merge` (capitalised like
  every enum's members — the question had them in lower case) and
  `Output.stderr`. Both platforms feed the input and read both outputs at
  once — Windows with a writer and a reader thread, POSIX with one `poll`
  loop (and SIGPIPE ignored, so a child that stops reading is an EPIPE) —
  so a child that writes a megabyte of errors before reading its input
  cannot deadlock (driver `TestRunInputAndStderr`, with a 60s guard; also
  under `--sanitize` on Linux). No input is an empty input now, not this
  process's. `mergeStderr:` still parses, with an error whose fix writes
  `os.Stderr.Merge`, or `Inherit` for `false` (the old default).
  Chapter 15, the stdlib reference, the cheat sheet.

- **D81 / B4 done**: in a debug build every call the programmer wrote is
  bracketed by `veles_call_push("site\0callee\0")` / `veles_call_pop`
  (`codegen/llvm/expr.go`); the stack is the task's own (`veles_shadow`, in
  collected memory), one per thread outside tasks, and a suspending call's
  child task links to its caller (`veles_call_link`) so the chain runs
  through suspension. A panic copies the chain as text (`panic_trace`), a
  scope's re-raise carries the child's chain up, and the test report shows
  it under `at` (a helper call site the chain already has is not printed
  twice; helper lines now indent like the rest). Release builds emit
  nothing and print `(a debug build shows the call chain)`. Deeper than
  4096 frames the chain is dropped rather than shown wrongly. Tests: driver
  `TestPanicPrintsCallChain` (three calls, through a suspension, release
  note), updated test-runner reports; the goldens carry the push/pop.
  Chapter 7, the errors and stdlib references.

- **2026-09-29, D85 — named imports (checklist Q15 decided, built)**:
  `use io { println, eprintln as warn }`, spelled with `as`, braces add bare
  names on top of `io.` (user's choice among `from … import`, `::`, `:`,
  `as`). `ast.UseSpec.Names`; the parser reads the braces, refuses `{ * }` and
  `{ }`, and reads the removed `use m.{ }` with a fix that drops the dot;
  `declareUseNames` binds the module's own symbols into the file's import
  scope (a copy under the alias), refuses private, test-code, missing (with a
  typo guess), prelude-global and prelude-home names, and collisions with
  another import or a module declaration; `lintUnusedNames` warns with a
  fix (the scope records which names a lookup found). Formatter sorts the
  names; completion offers the module's names inside braces and the bound
  names elsewhere; TextMate colours them. Tests: conform `D85-*` (three
  files), `TestNamedImports` (across modules: type, enum, function, alias,
  shadowing, private, test code, unused), format and LSP cases; example
  `examples/features/named`. Docs: chapter 11, cheat sheet, errors reference. Not
  done: auto-import on completion of a name the file has not imported;
  rename of an aliased name follows the member, not the alias. `as` for
  conversions (Q16) is untouched.
- **2026-09-29, D85 follow-ups** (the two "not done" items above are done,
  and the tree is migrated): auto-import on completion — the standard
  modules' public names are read once from the embedded sources, the loaded
  modules' from the package; the edit goes into the braces, after a bare
  `use m`, or on a new line, and only with a typed prefix. A renamed import
  is a declaration of its own (`finishImportRefs`): uses of the alias
  reference the alias, hover says `alias of io.eprintln`, an unrenamed bare
  use hovers `module io`, and renaming the alias leaves the member alone.
  Migration: `io.println/print/eprintln/eprint/readLine` became bare in std,
  examples, bench and the `veles new` templates (63 files; a small Go
  program did the text change, `veles fmt` the layout; every expected output
  unchanged). Docs and the compiler's own Go tests keep the qualified form.
  Tests: `TestAutoImportOnCompletion`, `TestRenamedImportHoverAndRename`.

- **2026-09-29, B9 / D86: numeric conversions are methods, `as` only
  renames.** `x.toT()` on all twelve numeric types (`i8…usize`, `f32`, `f64`):
  total where nothing can be lost, a `T?` (null when out of range) otherwise;
  a float truncates toward zero and NaN is null. `x.wrapT()` between integers
  keeps the low bits; a literal that cannot fit (`300.toU8()`) is a compile
  error. `p.cast<*raw U>()` replaces `as *raw U` (unsafe). Codegen:
  `num.toChecked` (range test by converting back and comparing, with the sign
  flip check; floats by bounds, NaN false) beside the existing cast. `x as T`
  keeps parsing with an error naming the method and a fix — `veles check --fix`
  migrated std, examples, bench and the goldens (~140 sites; nested casts took
  two passes); docs (chapters 2, 3, 6, 7, 13, 19, cheat sheet, errors
  reference: three new anchors under `numbers`) and the Go tests' embedded
  programs by hand. Two float→int sites became explicit decisions
  (`Duration.ofSeconds` panics when the count does not fit, `jwt` refuses an
  out-of-range NumericDate) and `Value.asI64` reads a float only when
  integral. Hover/completion entries for every method. Tests: conform
  `D86-conversions`, `examples/features/conversions` (edges of every family), golden
  `arith` regenerated. Q19 (decided the same day): `x.wrapTo<T>()` is
  `wrapT()` with the target as a type argument, for generic code (the range
  iterators use it); the embedded-std `as` exemption is gone. Also: `internal/wsl-test.sh`
  cleaned the clone after, not before, checking out HEAD, so a file the
  commit began to track aborted the checkout and the run used a stale base;
  order fixed.

- **2026-09-29, hover shows an inferred `suspends`** (user's ordering: this,
  then Q3/Q4/Q8, `std/log`). Hovers were rendered from the written signature
  before the suspension pass ran, so only the inlay hint knew. Now
  `showInferredSuspends` (`sema/index.go`) adds ` suspends` to the hover of
  every reference to a function whose effect was inferred, at its declaration
  and at each call, methods included (before ` throws`). Test
  `TestHoverShowsInferredSuspends` (declaration, transitive, call site,
  method, and a pure function that must not say it); docs chapter 12.

- **2026-09-29, Q4 / D87: `protected` on a mutable-collection field is "look,
  don't take".** `sema/protected_contents.go`: a read of such a field from
  outside the type is refused unless it is a method receiver, a value-loop
  head or an interpolated value (`lookOnly` marks the expression checked
  next); a receiver is then refused when the method mutates — a built-in of
  the mutable family (the catalogue lists those apart), or an `extend` block
  naming the mutable type itself. `loop (&x in …)`, binding, passing and
  returning are refused; `.toList()` copies. Nothing in std or examples used
  a protected collection, so no migration. Tests: conform
  `D87-protected-collections` (every refusal and every allowed look), docs
  chapter 5 (a run block), errors reference. Notes R18(b) and R32 leave
  the notes file.

- **2026-09-29, Q8 / D88: `@caller_location`.** A marked function takes the
  site it is called from as a hidden last parameter (`Func.CallerLoc`; the
  checker refuses the attribute outside std, on trait implementations,
  externs, and on anything that suspends); a call passes its own span, or —
  from inside another marked function — the site it was itself given, so
  `n.toString(radix: 1)` through `checkRadix` reports the user's line; a
  `panic(...)` in the body reports that site (its own line when the site is
  empty: a compiler-made call, a function value's thunk). Both profiles. The
  debug chain says the site once: the runtime finds the chain line whose site
  is the panic's, prints `at SITE in CALLER` and drops the std frames above
  it. Marked: `swap insert removeAt chunked windowed step` (both), `toString
(radix:)` + `checkRadix`, `randomBytes`, `Uuid.of`, `random.range`. Not
  marked: trait implementations (`Hmac.update`), the JWT key check, and
  `mapConcurrent` (suspends). Tests: driver `TestCallerLocationPanics`
  (debug + release, through two marked frames), conform `D88-*`, goldens
  regenerated (an extra argument on every marked call); docs chapter 7 and the
  errors reference.

- **2026-09-29, Q3 / D89: `public use`.** Parser and formatter: a `public use`
  is a declaration of its own (`UseDecl.Pub`; the formatter neither merges it
  into the run of plain imports nor sorts it). Checker (`sema/reexport.go`):
  `public use m` puts the module symbol into the module's scope under its name
  or alias, `public use m { a, T as U }` puts those items there (the module's
  own symbol, so hover and go-to-definition land on the declaration);
  collisions with declarations are errors; std and dependencies cannot be
  re-exported; a re-exported name is never "unused". Loader (`resolveDep`): a
  dependency's surface is read from its root module's `public use` — one per
  path segment, so a module passes on what it re-exports — replacing the
  manifest list; the error names the line to add. The manifest's `exports` is
  an error naming the replacement (`Manifest.Exports` and `exported` are
  gone); `veles doc` lists the root plus what it re-exports. Migrated:
  `examples/features/packages` (root re-exports `geometry` and flattens
  `support { root as sqrt }`, a module that stays hidden). Tests:
  `TestPublicUse` (facade, alias, hidden module, flattened-only module, chain,
  std/dependency refused, both collisions, manifest error), conform
  `D89-public-use`, format `TestPublicUseIsNotMerged`, LSP
  `TestDefinitionFollowsReexport`, docs chapter 11 ("What a package shows"),
  errors reference, cheat sheet. Notes #19 leaves the notes file.

- **2026-09-29, C1 / D90 + D91: `lazy` parameters and `std/log`.**
  `lazy` (parser, `ast.Param.Lazy`, `types.Param.Lazy`, `sema/lazy.go`): a
  parameter of type `fun(): T` (no arguments or effects, std-only) takes a
  plain expression at the call, wrapped in a lambda by `bindArgs`;
  formatter, hover and signature help spell the modifier. `std/log`: `Level`
  (Debug, Info, Warn, Error, Off), `debug/info/warn/error(lazy msg, Field...)`,
  `field<T: Encodable>`, `setLevel`, `enabled`, `withFields` (a `TaskLocal`,
  D72), `VELES_LOG`; text on a terminal (a runtime `veles_stderr_is_terminal`,
  both platforms), JSON otherwise; one `eprintln` per line, which the runtime
  already writes whole under the stream lock. `http`: `serve`, `logging()`
  and the handler-failure lines go through `log`; `requestId()` binds `id`
  for the request, so `logging()` no longer reads the header itself (it needs
  `requestId()` outside it, as documented). Tests: `std/log/log.test.vs`
  (both line formats, quoting, level names, thresholds — run by
  `TestStdLogUnitTests`), `TestLogInAProgram` (typed JSON, laziness, scope
  through a child task, `VELES_LOG`, 400 lines from 8 tasks all whole),
  conform `D90-lazy`, format round trip; docs chapter 21, the stdlib
  reference, chapter 17. Measured: a disabled `log.debug` is ~30 ns (a
  closure per call), the guarded form ~0.3 ns — checklist §5.7 keeps the
  hoisting as an open item.

- **2026-09-29, S1–S3: self-host readiness.** S1: `veles-selfhost-frontend-plan.md`
  refreshed against v0.49 — line counts (6353 non-test lines, was 5472), corpus
  size (173 files), the "module-level `var` is a real mutable global" claim
  struck (D66), stale `binarySearch` and `partitionPoint` references fixed,
  and a new §8 listing what D60–D91 change for the port (`this`/`init`, D78
  tests, D86 conversion methods, D85 named imports, D89 `public use`, panic
  chains, `lazy`/`std/log`). S2: four compiler-shaped benchmarks, each with a
  Go reference (`bench/compiler_refs.go`; the runner now subtracts the setup
  a reference does before Veles's clock starts, `goSetup`): `lexer` (byte
  scan, keyword map, token structs), `ast` (sealed family built and walked
  with `when`), `intern` (string-keyed symbol table), `emit` (250 000 lines of
  IR through a `StringBuilder`). Ratios to Go 0.4–2.0×, none over 3×, so no
  checklist item; the Go side swung 2–3× between runs on a loaded machine
  (caveat in `bench/results.md`). S3: §9 of the self-host plan audits what a
  whole compiler would lean on, each row probed with a program: large
  cyclic pointer graphs, struct- and sealed-keyed maps, insertion-ordered
  iteration, stable sorting, `os.run` of clang, file I/O, parallel tasks all
  **have** it. Two findings became checklist items: stack exhaustion is
  silent (exit 127 / SIGSEGV at ~30–50k frames on Windows, ~200k–1M on
  Linux) and the stack size is the OS default (§2), and there is no safe
  `f64`↔`u64` bit reinterpretation (§5.10, needs a decision). Not done: the
  compile time and peak memory of a 30 000-line program by a Veles-written
  compiler, which belong to E4.

- **2026-09-29, stack overflow (S3 finding): D92.** `runtime/c/veles_stack.c`
  is new: the process's `main` (the generated one is now `veles_main`) starts
  the program on a thread with a 256 MB reserved stack (`VELES_STACK=<MB>`,
  1–4096; a refused reservation is retried at half down to 8 MB), and
  `veles_thread_spawn` gives every worker the same, while the timer monitor
  and the mutex watchdog use `veles_thread_spawn_small` (OS default). A fault
  handler — SIGSEGV/SIGBUS on a per-thread `sigaltstack` on POSIX, a vectored
  exception handler with `SetThreadStackGuarantee(64 KB)` on Windows — prints
  `panic: stack overflow`, the size and, in a debug build, the innermost
  calls of the D81 chain (`veles_shadow_peek` in `veles_task.c`), using no
  allocation and no formatted output, then exits 101. Found on the way: below
  the 8 MB floor the size loop never tried a thread (fixed), and `_exit` needed
  more stack than a 1 MB stack leaves on Windows (`TerminateProcess`).
  Test: driver `TestStackOverflowIsReported` (a million frames in main and in
  a task; overflow in both profiles; `VELES_STACK=1`; a bad value), green on
  Windows and WSL, as is the full suite; `spawn`/`parallel`/`channels`/`pipes`
  benchmarks unchanged. Docs: chapter 7, chapter 3, the stdlib reference,
  `veles-debug`. Open: a frame bigger than the guard region (clang stack
  probes), macOS (A7).

- **2026-09-29, float bits (S3 finding): D93.** `x.toBits()` on `f64`/`f32`
  (`u64`/`u32`) and the statics `f64.fromBits(bits)` / `f32.fromBits(bits)`,
  in `std/prelude/number.vs` over the `unsafe` reinterpretation the caller
  would otherwise write (with SAFETY comments; no compiler change).
  `examples/features/floatbits` pins it (LLVM-style 16-digit hex constants for 1.5,
  0.1, -0.0, pi, infinity; exact round trips; a signalling-NaN payload kept;
  `-0.0 == 0.0` yet different bits; the `f32` pair; sign, exponent and
  fraction taken apart), identical in debug and release. Docs: chapter 2, the
  stdlib reference. The float sign helpers and the missing integer bit
  operations were left out on purpose and recorded as checklist Q20.

- **2026-09-29, editor support for `lazy` (D90).** The word had no hover and no
  highlighting: it is contextual, so neither the lexer's keyword table nor the
  TextMate grammar knew it. `ast.Param.LazyPos` (the parser records the word's
  span); the checker records a hover reference for it (`refLazyWord`, kind
  "keyword") that says what a `lazy` parameter is; the shared grammar
  (`editors/vscode/syntaxes/veles.tmLanguage.json`, which the docs site's
  Shiki also loads) colours `lazy` as a storage modifier only when a parameter
  name and `:` follow, so `val lazy = 1` and a field called `lazy` stay plain.
  Test: lsp `TestHoverOnLazyWord` (the word, the function signature, an
  ordinary parameter, and that a rename on the word is refused). Not done:
  completion offers no `lazy` (std-only for now); the packaged
  `editors/vscode/veles-0.1.0.vsix` is rebuilt by the user.

- **2026-09-29, C2 (first task): cookies and forms, D94.** New files
  `std/http/cookie.vs` (`SameSite`, `Cookie`, the checks, `Set-Cookie` text,
  the `Cookie` header parser) and `std/http/form.vs` (`Fields`, `FormDecoder`
  — a `codec.Decoder` over the fields, format name `form` — and
  `decodeFields`). `http.vs`: `Response.cookies` written one `Set-Cookie` line
  each (and kept through `withHeader` and a head-only copy), `withCookie` /
  `withoutCookie` (`@caller_location` panics for a bad cookie);
  `Request.rawQuery`, `cookie`/`cookies`, `queryFields`, `formFields`,
  `formValue(s)`, `form<T>`, `query<T>` (a field and a method may share the
  name `query`). Found on the way: a decoder's `path()` between members must be
  the object's own path (a derived decoder names a missing field under it);
  constructors are by field name (D28), so the API is `Cookie(name:, value:)`.
  Tests: `std/http/cookie.test.vs` and `form.test.vs` (35, run by driver
  `TestStdHttpUnitTests`), `examples/session` over a real socket (both
  `Set-Cookie` lines, the encoded value, 400/415, the refused cookies);
  docs chapter 17 (two sections) and the stdlib reference. Next in C2, in the
  order chosen: static-file caching (`ETag`, `Last-Modified`, `Range`,
  `Cache-Control`), then the body model, then limits and middleware.

- **2026-09-29, `if (val x = e && ...)`: D95.** The user found `setCookieLine`
  clumsy ("something like `?.let` in Kotlin ... execute some code without
  letting nullable") and chose the block form over a lambda. `ast.LetCond`
  (`val name = value` as an operand; the parser reads the value above `&&` so
  the chain goes on), the formatter and the AST dump print it; the checker
  (`sema/letcond.go`) lowers each binding to a `Let` (evaluate, test against
  null) plus a fact narrowing the variable (`condFacts`), opens a scope in
  `ifExpr` that closes before the `else`, and refuses a non-nullable value or a
  binding outside the `&&` chain of an `if` (family `nullable`, `(D95)`). No
  code generation change. `std/http/cookie.vs` uses it. Tests: conformance
  `D95-if-val` (scopes, both errors, the unused warning), format
  `TestIfValRoundTrips`, lsp `TestHoverOnIfValBinding`, `examples/features/nullbind`
  (evaluation order, chains, else-if, early return, as an expression); docs
  chapter 6, the errors reference and the cheat sheet.

- **2026-09-29, C2 static files: D96.** The user chose every recommended
  option (validators, one range, `no-cache` default, redirect + dotfiles) and
  asked that directory handling be configurable, so `redirect: false` serves the
  index without the slash. `fs.stat` (`veles_fs_info` in `veles_os.c`: a handle
  and `GetFileInformationByHandle` on Windows for the 100 ns write time,
  `st_mtim`/`st_mtimespec` elsewhere) returns `Stat { size, modified, isDir }`.
  `http.files` moved to `std/http/files.vs` as a `FileServer` struct behind the
  same `sendable fun` and takes `maxAge`, `immutable`, `index`, `dotfiles`,
  `redirect`, `etag`, `lastModified`; conditions in RFC 9110 order, 304/412
  decided from the stat, one range with `If-Range`, the slash redirect rebuilt
  from encoded segments so `//host` cannot appear, only GET/HEAD. Found by the
  tests: the parallel tests truncated each other's fixture files, so the tree is
  built once at start-up; `bytes=10-` on ten bytes was read as invalid
  (fixed, pinned). Tests: `std/http/files.test.vs` (15 tests, run by driver
  `TestStdHttpUnitTests`); docs chapters 15 and 17 (a run example), the stdlib
  reference and the cheat sheet. Next in C2: the body model (chunked,
  streaming, multipart, `Expect: 100-continue`), then limits/middleware.

- **2026-09-29, C2 body model: D97.** The user chose every recommended option
  (lazy request body, `Response.stream` + `fs.File`, streamed multipart,
  `MediaType`), the last after asking why the extension table was not with the
  named values. `std/http/body.vs` (`Body`: Content-Length, chunked and in-memory
  framing behind a `Mutex`; `100 Continue` on the first read; drain of an
  unread body up to 64 KiB; the longest-silence timeout), `stream.vs`
  (`BodyWriter`, `produce` behind the D56 boundary), `multipart.vs`
  (`Multipart`/`Part` over the body reader), `files.vs` streaming with
  `readAt`, `values.vs` `MediaType`; `std/fs`: `open` → `File`, backed by
  `veles_fs_open/file_read_at/file_write/file_size/file_close` in
  `veles_os.c` (a HANDLE and `ReadFile` with an offset on Windows, `pread`
  elsewhere). `Response.stream` gained a `length:` (not in the brief) because
  a download without `content-length` loses progress and resume; a producer that
  writes more or fewer bytes than declared fails. The fuzz example now sends
  valid chunked requests and four chunk faults (found: an IoError from a
  non-UTF-8 chunk line was a 500 — it is a 400 like the head's; a chunked body
  with HTTP/1.0 is refused by the server, so the generator no longer
  produces it); the form readers keep what they read so `formValue` then
  `formValues` still works. Tests: `std/http/body.test.vs`, `stream.test.vs`,
  `multipart.test.vs` (real sockets for chunked, 100-continue, draining, the
  stall, truncated streams), `std/fs/fs.test.vs` (`TestStdFsUnitTests`),
  `examples/fuzz` and `httpd` regenerated (a 413 now carries a request id); docs
  15 and 17 with five new run examples, the stdlib reference and the cheat
  sheet. Next in C2: limits and middleware (max connections, CORS, auth hook,
  compression).

- **2026-09-29, `do { } catch { e => }`: D98.** Raised while testing D97: several
  `try`s that should fail into one local handler in a function that carries on;
  the closure-called-at-once idiom cannot `continue`/`break`/`return` past
  itself. The user chose a block form with `try` kept (over implicit
  propagation, a function-level suffix, or leaving it), then — "other languages
  have 'success story' in a try, and catch block is where the error happens" —
  changed the spelling from `catch { } else { e => }` to `do { } catch { e => }`.
  `do` and `catch` were reserved words; `lexer` makes them keywords, `ast.CatchExpr`
  holds the block and the existing `Handler`, the parser reads it (with clean
  errors for a missing `catch`, `do`-`while` and a stray `catch`, each a
  well-formed node so the statement parser does not resynchronise and cascade),
  the formatter prints it. `sema/catch.go` lowers the block to a one-shot HIR loop
  with failure slots, the adapters' technique: `try`/`throw` inside route to the
  slot through `f.catching` (a `catchFrame` on `fnCtx`), and the nodes built
  before the error union is known are patched at the end — no code generation
  change. A fail-fast `scope` child inside the block is refused, since the scope
  passes its error to the function. Tests: conformance `D98-do-catch` and
  `D98-do-syntax`, lsp `TestHoverOnCatchBinding`, `examples/features/catchblock` (`with`
  closing on a failing `try`, a suspending call, `break outer`, a lambda inside,
  nested blocks, a rethrow); docs chapter 7 (a run example), the errors
  reference, the cheat sheet; completion and the TextMate grammar know both
  words. Found on the way: the D97 commit changed 500 bodies to the reason
  phrase but not the two examples that print them (`session`, `httpd`), so their
  expected output was regenerated.

- **2026-09-29, D98 revised: `catch (e) { }`.** Two more rounds with the user:
  the spelling `catch { e => }` "is still not the best" (every other head binds
  in parentheses — `when (v)`, `loop (x in xs)`), and the same for the handlers
  of `??` and let-else. Result: one spelling, `catch (e) { ... }`, a tight postfix
  on a `Result`, on `try chain` and on a `do` block; `?? { e => }`, `else { e => }`
  and `catch { e => }` are refused with one line naming the new form (no
  deprecation period, per the user); no patterns yet (`when (e)` inside is
  exhaustive over the union). `try f().g() catch (e) { }` covers the whole chain
  (the user disliked `(parse(s) catch (e) { 0 }).len()`); a per-call postfix
  marker for Results stays a separate, open decision. Parser: `parseCatchHandler`
  and `parseCatchPostfix`, a `catch` may start the next line, `inTry` keeps a
  `catch` after a `try` operand from binding to its last call; sema: a postfix
  `catch` on a Result reuses the `??` lowering (`coalesceOf`, with `catch` in the
  messages), on a `try` it is checked as `do { try ... } catch`, and the dead
  "only a Result has an error to bind" branch is gone. Migrated: 8 `??` and about
  6 `else` handlers plus the 22 `catch { e => }` in std, examples, docs, Go test
  sources and the veles-code skill. Tests: conformance (`D98-do-catch` with the
  postfix, `try` chain, optional binding and next-line cases and their refusals;
  `D98-do-syntax` with the three old spellings), format cases for postfix and
  `try ... catch`, lsp hover; `examples/features/catchblock` output unchanged.

- **2026-09-30, C2 limits and middleware: D99.** The user chose every recommended
  option. `Limits.connections` (default 10000, 0 = none) bounds the connections
  served at once with backpressure at `accept` (a permit channel taken before
  `accept()` and returned when the connection task ends; a stop cancels the wait).
  `http.cors` (`std/http/cors.vs`): explicit origins, `["*"]`, or `https://*.example.com`
  matched on a dot; preflight answered before the router; `Vary: Origin`; `*` with
  credentials and malformed patterns panic at the caller. `http.guard`, `basicAuth`,
  `bearer` (`std/http/auth.vs`): a check that answers or lets on; Basic puts the
  user in `x-remote-user` (`Header.remoteUser`) over a forged one. The brief said
  std had no constant-time compare; `crypto.equalBytes` is one, so nothing was
  added (corrected in D99). Tests: `limits.test.vs` (real sockets), `cors.test.vs`
  (12), `auth.test.vs` (10). Docs chapter 17 ("Other origins, and who may call",
  connections, the `Limits` table), stdlib reference. `Expect: 100-continue` ticked
  (it was built in D97). Next in C2: `std/compress` in Veles (inflate, deflate, gzip;
  measured with veles-bench, miniz as the fallback), then `http.compress()` and
  precompressed `.gz` siblings for `files`; then C3 `std/config`.

### 2026-09-30

- **Spec review and preparation (`archive/veles-spec-prep.md`).** The user asked for a
  review of the language's state and for the spec to be prepared, together, so
  that another agent can build the rest without close supervision. The review
  (in that file): state per track; seven stale or contradictory spec passages,
  fixed in the spec the same day (the header, §4b varargs vs Q7 and `testing`
  vs D78, §5 and §4b interface files marked not built, §6.2's missing manifest
  file, §7's "not designed" list, D43's §6.1 remark); one bug (the
  needless-`throws` warning's family, checklist §6); the open questions ordered
  into eight batches. **Batch 1 decided: D100** (`with x = e` as a statement,
  `with t = async f()`, no escape of a with-bound value, the tree migrated) —
  plan B10, not built.
- **Batch 2 decided: D101–D106**, every recommended option (Q12, Q13, Q17,
  Q20, spec §6.1 and empty literals leave the open lists). Probed first: a
  list loop saw its own appends and skipped after a removal (now refused,
  D102); a one-task `gather` was the only 1-tuple in the language (D103).
  Plan B11, not built. Notes I2 and I3 leave `notes_to_change.txt`.

### 2026-10-01

- **Batch 3 (tests and concurrency): D107–D110 decided, Q2 still open.**
  `with p = m.lock()` beside `withLock` (D107), send arms in `race` (D108, Q1
  closed), `with expr` without a name (D109), and `retry` / `Semaphore` /
  channel drains / `ticker` pre-approved (D110, notes P11) — plan B12, not
  built. Q2 (shared setup) was explained twice — suite-level `with` lines run
  around each test, and lending functions (`with fun … yield v`) for fixtures
  that start a task — and the user is not sure yet; it stays in §9 with both
  shapes written down. Found: which `race` arm wins when several are ready is
  unwritten (checklist §1.3).
- **Q2 decided from the user's own idea (D111), and batch 4 (safety and
  build): D112–D115.** The user was not convinced by lending functions and
  asked for Go's shape — a helper returning a value; a value holding a `Task`
  is now received with `with`, its tasks living until that block ends (D111),
  which answers shared test setup without a test feature. `Secret<T>` wiped on
  close and by the collector (D112, over the recommendation), compile-time
  evaluation with constant tables and `const fun` (D113, beyond the
  recommendation), unchecked access only inside `unsafe` (D114), a warning for
  a `Closeable` never closed (D115). Plan B13 and B14, not built. Q2, Q9, Q10,
  Q11 leave §9.
- **Batch 5 (types and effects): D116–D119.** Probed first: `throws E`
  already followed the argument, suspension did not (a `suspends` parameter
  made every call suspend, and the adapters refused suspending lambdas) — D116
  makes it follow the argument, which unblocks B8. `is Trait` both at run time
  and at compile time (D117, over the recommendation), derivation by
  compile-time reflection later (D118), `Default` (D119). Plan B15. Q5, Q6
  leave §9. Found: `x is Display` gives a wrong message (checklist §1.4).
- **Batch 6 (FFI, Q7): D120–D123.** Probed first and found a **bug**: structs
  passed or returned by value across the C boundary had no C ABI lowering
  (`lldiv` segfaulted on Windows) — plan A8, refused until fixed. Decided:
  `@packed`/`@align`/`@transparent` and `extern union` (D120), `Array<T, N>`
  with const generic parameters as a general inline value type (D121), no
  `.d.vs` (D122, notes #16 closed), calls of variadic C functions (D123,
  superseding §4b — POSIX `open`/`fcntl` are the new evidence). Plan B16. Q7
  leaves §9; only Q14 (after M7) remains there.
- **Batch 7 (the std APIs): D124–D130.** Compression with a 64 MiB default
  ceiling (user over the recommended required `max:`), config from a struct
  over env + files, **OpenTelemetry with all three signals and OTLP/protobuf**
  (user over the recommended Prometheus-shaped metrics, then over metrics +
  traces with OTLP/JSON), the HTTP client in both `fetch` and Go spellings
  (user), `io.Stream` shared by TCP, TLS and files, **template literals** and
  `std/db` built on `sql"…"` (asked twice; the second time with how it sends
  values apart from the text), and four small additions. Track C needs no more
  questions. **Not asked, assumed — to confirm**: the `/healthz`/`/readyz`
  helpers were bundled with the Prometheus option the user did not take; they
  are independent of telemetry and stay in checklist §5.9 as open.
- **Batch 8 (packages): D131, D132 — checklist §9 is empty.** The user asked
  to see each manifest format in full with pros and cons, then reasoned that a
  `Package` struct could also serve scripts and that `const` already forbids
  logic; the recommendation moved from a `veles.mod` line format to
  `package.vs` (D131: one typed constant, conditions on target facts only).
  Packages stay decentralized but drop Go's look: host shorthands, the major in
  the version, no pseudo-versions, a search index later; no lockfile (D132).
  Plan E7 rewritten with acceptance. Notes #3 and #17 leave the notes file.
- **Consistency pass over D100–D133, and D133–D135.** Health endpoints added
  (D133, user confirmed). Probed two assumptions the new entries made and both
  were false: one `try` did not cover a chain's second failing call (D134,
  Swift's rule, closing what D98 left open), and `is Sql` on a trait object was
  refused (D135, the downcast D129 needs). Fixed in the spec: `ticker` moves to
  `std/time` (its ticks are `Timestamp`s), constant `Array` tables (D113/D121),
  the manifest's `build.os` renamed `target.os` (the module's names are in scope
  unqualified), `net.TooLong` → `io.TooLong` under `io.Stream`, an attribute on
  its own line (D51), nameless `with` in the OpenTelemetry example, `log.field`,
  three stale "batch"/"§9 Q7" references, and the header rewritten as an index of
  D100–D135. The plan gained a **Build order** for all of it (dependencies in
  brackets), and CLAUDE.md points agents at it.

- **A8 done: structs by value across the C boundary.** The bug found preparing
  batch 6 (`lldiv`, a 16-byte return, crashed on Windows; `div` was right only
  because an 8-byte `{i32, i32}` happens to come back in RAX) was that an
  `extern struct` was handed to LLVM as a first-class aggregate, which LLVM
  splits by rules no C compiler uses. `codegen/llvm/cabi.go` now does what
  clang's front end does, per target: Win64 (1/2/4/8 bytes as an integer,
  anything else by pointer to a caller copy and `sret`), SysV x86-64
  (eightbytes classed INTEGER/SSE and coerced — `i24`, `{ i64, i32 }`,
  `<2 x float>`, `{ double, i64 }` — `byval` beyond 16 bytes _or when the
  registers left cannot hold the whole struct_, `sret` for large results),
  AAPCS64 (homogeneous float aggregates as `[n x float|double]`, other
  structs up to 16 bytes as `i64`/`[2 x i64]`, larger by pointer, `sret`).
  One classification serves the four places a struct crosses: extern
  declarations, direct extern calls, calls through `extern fun` pointers, and
  the wrapper of an exported `extern "C" fun`. Scalars, std's string pairs and
  lists are unchanged (no golden moved). Tests: driver `TestCStructsByValue`
  generates a C file and a Veles program over 17 shapes (every class above,
  a nested struct), each called C-from-Veles, Veles-from-C by name and through
  an `extern fun` pointer, plus two register-exhaustion calls — debug and
  release; red at HEAD (access violation in a HEAD worktree), green on
  Windows and Linux. Codegen `TestCStructClassification` checks the
  signatures for all three conventions against what `clang -S -emit-llvm`
  writes for the same C, so ARM64 is covered before A7 can run it. The
  planned interim refusal was skipped: the classifier landed in the same
  change. Docs chapter 13 has a runnable `lldiv` sample; spec D67 addendum.
  Not done: `@packed` structs and unions (D120, plan B16) will need the
  classifier's MEMORY rule for unaligned fields when they land.

- **B10 done: D100, `with x = e`, `with t = async f()`, no escape.** The
  parser keeps `with x = e` as a flat `ast.WithStmt`; the checker reads the
  rest of its block as the body of D43's block form, so the IR is the nested
  program's. Refused with the block form named: module level, expression
  bodies, operands and arms, braceless bodies, two bindings in one statement;
  a `with` as the last statement warns. A with-task is a `ScopeBlock` flagged
  `Cancel` holding only its launch; codegen cancels its children when the
  body ends, then joins. Found on the way: the fail-fast abort path jumped to
  the innermost scope's join without cancelling that scope's children, so an
  inner scope waited out its children after an outer one had failed — now
  cancelled (the `tasks` golden gained the call). Part 3 is
  `sema/resources.go` (family `resources`, which also took the
  not-Closeable message): returns, block values, assignments to places that
  outlive the block, stores into collections and channels, and lambdas
  capturing the resource, through `val` aliases and literals. Hover on `with`
  says where it closes; an inlay hint after the `}` lists the closes in
  order. Migrated with a parser-driven rewrite: 38 sites in std and examples,
  14 in docs samples, 4 std doc comments by hand, and the
  `scope { … cancel() }` shapes in `std/http`, `examples/httpd`, `session`,
  `fuzz`, docs 12 and 17; `std/http/limits.test.vs` is now D100's own
  example; expected outputs unchanged. Docs: chapter 13 teaches the statement
  form first (the plan said chapter 7; `with` lives in 13), chapter 12 has
  "Background tasks", cheat sheet, stdlib and errors references. Tests:
  conformance `D100-with-statement`/`-syntax` (+ a D16 global case), driver
  `TestWithStatementExits` (end, return, break, continue, failed `try`,
  throw, panic, cancellation; with-task cancelled at the end, fail-fast,
  awaited — debug and release), lsp `TestWithInTheEditor`, parser
  `TestWithStatement`, a format case, `examples/features/with` extended. Windows and
  WSL green.

- **B11 done: D101–D106, D134 and three decision-free fixes.** D134: one
  `try` unwraps every failing link of its receiver chain (calls and fields),
  the error type their union; `try (try f()).g()` warns as redundant with a
  fix; the `(try f()).g()` sites that end an expression rewritten. D101:
  `if (m = …)`/`when (m = …)` is one error with the `val` fix, the head read
  as the binding. D102: a call in a loop's body that changes the walked
  collection's length or order is refused (`sema/loopchange.go`, family
  `mutability`, fix "loop over a copy"), replacing `lint_staleref`'s
  by-reference warning; lists and maps carry a modification count checked at
  each step of a loop over a mutable list or map, panicking at the loop's
  line in both profiles. Benchmarks against HEAD (alternate runs): no change
  beyond noise. D103: a one-task `gather` is its `Result` (five `.0` sites and
  `expectPanics` migrated); `async` on a `sendable fun` value through a
  synthesized per-type function; the `invoke` trampolines removed. D104: eight
  bit/float methods, `examples/features/bits` (debug = release), golden `bits`. D105:
  `reserve` on `StringBuilder`, `MutableMap`, `MutableSet`, `Deque` (runtime
  `veles_map_reserve`; checked once with temporary instrumentation that
  inserting up to `n` neither grows nor rehashes), `examples/features/reserve`. D106:
  empty-literal message with the kind and the fix from the first deciding use,
  `Range.isEmpty`, lambda arm values parenthesized by `veles fmt`.
  Decision-free: the `async` placement error names what to write; the
  needless-`throws` warning reworded into `results-and-errors`; a literal
  argument takes the type the expected result binds (`val small: i8 =
id(12)`, §11 limit removed). Also fixed after B10: the close-order inlay hint
  no longer lists a with-task that is awaited on every path. Not done: the
  run-time count is not checked by iterators, which already walk copies.

- **D136 decided and built (2026-10-01).** From the B10 close-order hint: a
  hand-written `close()` on a `with` value ran twice. The user chose to refuse
  it (recommended of 3): `r.close()` on a `with`-bound value or a `val` alias
  of one is an error, family `resources`, whose fix removes the line; a
  with-task's `cancel()` stays allowed. Conformance cases in
  `D100-with-statement` (direct, through an alias inside a lambda, a plain
  `val` allowed, `t.cancel()` allowed); the fix checked with `veles check
--fix`. Docs chapter 13 and the errors reference; spec D136 (v0.68).

- **B12 done (2026-10-02): D107–D110.** D107: `with n = m.lock()` (prelude
  `Mutex.lock()` returning a private `Locked<T>`; `sema/lockguard.go`): `lock()`
  outside a `with` is an error whose fix turns `val` into `with`; suspension
  in the held region is refused as checked, and calls of suspending named
  functions after inference; `n` cannot escape; hover and an `unlocks notes`
  inlay hint. TestLockGuard: 8 tasks × 10 000 increments = 80 000 on 1/2/4/8
  threads, unlock on return, throw and panic, the re-lock panic. D108: `race`
  send arms, runtime send waiters, and a busy state so a send arm and another
  race's receive arm pair safely; TestRaceSendArms (4 producers racing sends
  against timers, a plain and a racing consumer: every value counted as sent
  arrives once, none counted as dropped arrives; closed channel panics) on
  1/2/4/8 threads ×5, plus 50 runs at 8 threads. The which-arm rule written in
  D38 and chapter 12. D109: `with expr` and mixed block items, `with _ = e`
  (which did not parse before), `with conn`; parser, formatter, conformance.
  D110: `retry`, `Semaphore`/`Permit`, `ch.forEach`/`ch.toList` (channel
  methods now fall through to extend blocks), `time.ticker` (a runtime ticker
  list, no task); tests in TestConcurrencyHelpers and
  `std/time/ticker.test.vs`. Bugs found on the way: generic `do { } catch`
  failed to compile in an instance where `E` is `Never` (fixed, conformance
  D98-generic-catch); two of my ticker tests were timing-dependent and failed
  on WSL (fixed). Not benchmarked: plain channel receive and close each
  gained one branch (a sender that is a race). The `--sanitize` runs came
  after the fix below.

- **Sanitized programs end cleanly again (2026-10-02).** Every program built
  with `--sanitize` on Linux aborted as its first thread ended ("AddressSanitizer
  failed to deallocate … error code 22"), at HEAD too, so `go test ./examples
-sanitize` and `./driver -sanitize` had stopped meaning anything. Cause: each
  runtime thread installs a malloc'd alternate signal stack over the one ASan
  gave it, and ASan unmaps whatever is installed when the thread ends; the
  Veles main thread also never called `veles_stack_thread_done`. Now
  `veles_stack_thread_init` keeps the previous alternate stack and
  `veles_stack_thread_done` reinstalls it, and the main thread calls it.
  TestSanitizedProgramEndsCleanly (Linux, always on; fails at HEAD with that
  abort); TestLockGuard and TestRaceSendArms honour `-sanitize`. Then clean:
  `go test ./examples -sanitize`, `go test ./driver -sanitize`, and the D107,
  D108 and D110 stress programs under `--sanitize` on 1/2/4/8 threads ×5.

- **B13 done (2026-10-02): D114, D115, D112, D111, and D137.** Built smallest
  first, the implementation choices asked first (all recommended options).
  D114: `atUnchecked`/`setUnchecked`/`byteAtUnchecked` only in `unsafe`
  (the error names the checked spelling), a debug check panicking "unchecked
  index … out of bounds" at the caller, a plain load or store in release;
  golden `unchecked` (both profiles), TestUncheckedAccess, chapter 13.
  D115: `sema/lint_closeable.go` — a `val` bound to a Closeable made here
  (a call or constructor) that is never closed nor handed on warns with the
  `val` → `with` fix; a discarded one warns with `with _ =`; `val _ =`
  discards. Measured over std, examples, bench and std's test files: no hits
  (B10 had already put every resource under `with`). D112: prelude `Secret`
  over a wiped `List<u8>` (runtime `DESC_WIPE` descriptor bit, zeroed by the
  sweep and by `close()`), redacted, Decodable not Encodable, constant-time
  `==`, refusals with Secret-specific messages; jwt and `crypto.hmac…()` take
  `Secret<List<u8>>` (examples/features/crypto migrated, output unchanged); the runtime
  probe `veles_gc_test_wipe` proves a freed Secret's slot reads zero and an
  ordinary one is untouched. `Secret.of(v)` did not compile — a static of a
  generic struct needed its type arguments — so **D137** (user, recommended
  of 3): such statics, and the prelude's collection statics, infer them as
  constructors do (`MutableList.repeat(false, n)` now compiles). D111: the
  adopt-after-return hand-off the user first chose was found racy reading
  the runtime (a finishing task against the move; a cancelled caller
  orphaning staged tasks); the user then chose the recommended _receiving
  scope_: a task-local binding (key -3) carrying the `with`'s scope while the
  value is computed, so held tasks launch straight into it and suspending
  callees inherit it. A held `with` is a `ScopeBlock` with `Held`
  (`codegen/llvm/held.go`): tasks are cancelled and joined before `close()`
  on every exit, a failed task's error is rethrown by matching it against the
  value's task fields (structs, sealed variants, `Task?`), and a held task's
  spawn waits until every field is evaluated, so a failing later field starts
  nothing. TestHeldTasks (debug and release, 1/2/4/8 threads), conform
  `D111-task-holding`, lsp TestHeldInTheEditor (hover, inlay hint
  `stops tasks of, then closes`). std/http's tests replaced the lending
  function `withServer(h, limits, port => …)` with a `TestServer` value
  (21 sites in four files). Found on the way: chapter 12's Semaphore sample
  incremented a counter inside `Atomic.update`'s lambda, which reruns under
  contention, so it saw "3 at once" under the parallel doc tests — the sample
  is fixed (the Semaphore was right); `update` with a side-effecting lambda is
  only documented, not refused. Also found, not fixed: `veles test` of a
  directory runs the embedded std test files of the modules it imports
  (`fs`, `time`); a mismatched `expect` argument can be reported at
  `<builtin>` with no location. Windows and WSL green (the full suite on
  both; one D100 case the held path had swallowed was fixed and rerun).
- **2026-10-02 — B15 (D135, D117, D119, D116) and B8 done**, after two bugs
  from B13: `veles test <dir>` no longer runs the embedded std tests of
  imported modules (`TestTestSkipsImportedStdTests`), and a mismatched
  comparison in `expect` is reported at its side, not `<builtin>`. Choices
  (user): order bugs → D135+D117 run time → D119 → D117 compile time → D116 →
  B8; a type-info slot in every method table with per-trait tables; plain
  instances cloned after inference; B8 moves everything and improves.
  D135/D117 run time: slot 0 of a method table points to the type's info
  global (a dense id), `is T` compares it, `is Trait` loads Trait's table at
  the id; tables cover every boxed type, built after checking to a fixpoint.
  Found building it, decided by the user: after `is Clicks` the name is the
  value inside the box, not a copy (writes reach it; D135 amended). D119:
  prelude `Default`, derived by an empty implement. D117 compile time: `T
implements X` is a per-instance constant; only the taken branch is checked
  and compiled. D116: two views per function in the suspension pass, a call
  asks whether what it passes suspends; codegen emits `.plain` copies on
  demand, lambdas under the creating instance's view, and keeps a
  conditional function's coroutine copy only when referenced; a parameter
  the function keeps makes it unconditional. Found on the way: a call
  through a suspending function value evaluated its arguments twice (fixed,
  pinned). B8: the list and map adapters are `std/prelude/adapters.vs`, the
  Go lowering and its Result machinery deleted; the old adapter messages
  survive as generic-inference messages for every higher-order function.
  Measured: `bench/adapters` (new) 64–73 ms lowered → 43–46 ms; other rows
  within noise of HEAD; `examples/httpd` IR 4 607 016 → 4 594 632 bytes.
  Windows and WSL green (full suite; examples and the D111/D116 driver tests
  on 1/2/4/8 threads on both).
- **2026-10-03 — B14 parts 1–3 done (D113: constant expressions, constant
  tables, `static assert`)**, after a bug found probing: module-level
  values were initialized in source order, so `val a = f()` with `f`
  reading a later `val` saw its zero bits, and a cycle between untyped
  globals overflowed the compiler's stack. Now `sema/globalorder.go` orders
  initializers by what they read, through called functions and lambdas, and
  a cycle is an error (conform `D11-init-cycle*`). Choices (user,
  recommended each): the compiler lays a constant Map/Set out whole with a
  Go copy of the runtime's hash; scalars are written in at each use, tables
  are one `constant` global; tables live in rodata. Built: the evaluator
  `sema/consteval.go` over checked HIR (exact integers checked against the
  type, f32 rounding, float text as the runtime prints it); `Global.Const`
  and `ConstTable`; `codegen/llvm/consts.go` (list/map/set headers,
  arrays, metadata and index as LLVM constants named `@const.<module>.<name>`);
  `CONST.at(i)` with a constant index read by the compiler (typed `T`, out
  of range an error) in any code; `static assert(cond, "why")` (ast node,
  parser, `veles parse` dump, formatter, checker at module level and per
  instance in bodies, the failure quoting the reason and the constants'
  values); hover shows a constant's computed value; new error family
  `constants`. Tests: conform `D113-constants`, `D113-static-assert`;
  TestConstantsMatchRunTime (debug and release: float text equals the run
  time's, every key of 13 constant tables of every key shape found by the
  runtime's lookup, a copy leaves the table alone, init order); golden
  `consts`; TestConstantsInTheEditor; TestStaticAssertRoundTrips. std's
  tables moved to `const` (SHA-1/256/512 round constants and IVs, multipart
  bytes, day/month names, `maxDays` now `maxSeconds / 86400`, numeric
  limits, `utf8.maxCode`…, `maxRecursionDepth`): none is built at start-up
  any more. Measured: `sha256` 77–85 ms after vs 78–79 ms before (noise —
  the cost is the rounds, not the table). Docs: chapter 2 (constants), the
  cheat sheet, the errors reference (`constants`). Not done here: part 4
  (`const fun`), so a constant cannot call a function yet. Windows and WSL
  green (full suite on both).
- **2026-10-03 — B16 started: D123 (variadic C calls) done.** Choice asked
  for the rest of B16 (user, recommended): `Array<T, N>` and any value over
  a size threshold get a _memory class_ in codegen (memcpy copies, passed
  by pointer to a caller copy, returned through `sret`), not SSA. D123:
  `...` ends the parameters of a function in an `extern "C"` block
  (`FunDecl.CVariadic`, `types.Func.CVariadic`); the extra arguments are
  promoted as C does (bool/i8/i16/u8/u16 → i32, f32 → f64, literals typed
  as C types them) and refused unless numbers, raw pointers or C function
  pointers (`sema/cvariadic.go`); the call names the function with its type
  written out (`i32 (ptr, ptr, ...) @snprintf`) so LLVM applies the
  platform's convention. Formatter, `veles parse`, hover. Tests: conform
  `D123-c-variadic*`, TestCVariadicCalls (snprintf with every promotion,
  POSIX `open` + `fcntl`, Windows `_open`; debug and release),
  TestCVariadicRoundTrips; chapter 13 section, cheat sheet. **Bug found on
  the way:** D100's escape check refused any argument of any `MutableList`
  / map / set / channel method when a `with` value was captured —
  `xs.withRaw(p => r.use(p))`, `xs.sortWith(…)` — though only storing
  methods keep their argument; now it is the storing methods only
  (`push`, `set`, `insert`, `add`, `getOrPut`, `send`, …), conform cases
  in `D100-with-statement`.
- **2026-10-03 — B16: D120 (C layout) done.** One layout calculation for
  the checker and the code generator (`types/layout.go`: sizes, the
  alignment a program is guaranteed and the one LLVM gives, struct plans).
  `@packed` → LLVM packed structs, its fields read and written through the
  whole struct (`&` refused, methods get copies); `extern union` → an
  integer of its alignment plus bytes, fields through memory, built from
  one field, touched in `unsafe`, no `==`/text/derive/const; `@align(n)` →
  explicit `[k x i8]` padding in LLVM struct types with a field-index map
  (`fidx`, used by every field access, hash, `==`, print, held-task and GC
  descriptor site), aligned allocas and globals, and collector descriptors
  carrying the alignment so `veles_gc_alloc` returns aligned bodies (lists
  of over-aligned elements included); `@transparent` → classified as its
  field for every C crossing, extension attributes included. Tests:
  TestCLayoutAgreesWithC (C fills and reads an epoll-style packed struct
  with a union, an `_Alignas` field struct and arrays of both, passes one
  by value and a transparent handle; sizeof/offsetof/strides agree; a
  `@align(64)` value is 64-aligned on the stack and in a list), conform
  `D120-layout*`, golden `layout`, TestCLayoutRoundTrips, chapter 13.
  **Bug found:** coroutine frames told LLVM they were 16-aligned while the
  collector gives 8; now `llvm.coro.id(i32 8)` (goldens `tasks`/`overflow`
  differ by that line only; `spawn` 43.9 ms vs 42.4 ms recorded, `channels`
  4.7 ms vs 9.3 ms — noise). Windows and WSL green.
- **2026-10-03 — B16: D121 (`Array<T, N>`, const generics) done — B16 complete.**
  _Types and syntax:_ `types.Array` (length a `types.Const`, a const type
  parameter, or open in a template), `<const N: i64>` on functions, structs
  and impls, constants where a type argument goes (`Array<u8, 4 * 16>`,
  `Array<u8, WIDTH>`; `>>` still closes two lists), `N` an `i64` constant in
  each instance, per-`N` instances, inference through `unify`, `N + 1` and
  non-`i64` parameters refused; formatter, `ast.Dump`, hover (`<const N: i64>`),
  completion, catalogue, tmLanguage. _Memory class_ (the user's choice): a
  value of 128 bytes or more that holds an array is an address in a register,
  copied by `memmove`, passed as the address of the caller's copy, returned
  through `sret` — `codegen/llvm/mem.go` (isMem, storeVal/loadVal, part,
  makeTaggedT, callRet), threaded through every site a value flows through
  (locals, fields, literals, nullables, tagged values, `Result`, closures,
  thunks, vtables, tasks and their argument blocks, channels, maps, trait
  objects, C by value); everything smaller emits the same IR as before (all
  goldens unchanged). A 1 MiB local compiles in a second. _Arrays:_ literals
  with the length checked, `at` (T where the index is a constant in range or
  proven: `loop (i in a.indices())`, `0..<K.len()`, `i < 4`), `set`, `first`,
  `last`, `indices`, `toList`/`toMutable`, `Array.make`, `xs.toArray<N>()`,
  `loop (x in a)` over a copy and `loop (&x in a)` in place, `==`/hash/text,
  const tables (`[N x T]` rodata, read in place, constant index folded),
  prelude adapters for `map`/`filter`/`forEach`/`fold`/`any`/`all`/`find`,
  every other read-only `List` method on a copy, `Default`, `Codable` (exactly
  `N`), `Sendable`, `CLayout` + `withRaw`, C arrays in extern structs.
  _Tests:_ TestArraysAgreeWithC (sockaddr, mixed, float and 80/160-byte
  structs by value, 2-D arrays; Windows ABI here, SysV on WSL), TestArrays
  (12 tests: values, reads, loops, const generics, list methods, hash/text/
  default, codable, then the memory class through nullables, tuples, branches,
  lists, maps, errors, closures, trait objects, tasks, channels, a megabyte),
  conform `D121-*` (4 files), golden `arrays`, parser `TestConstTypeArguments`,
  format `TestConstGenericsRoundTrip`, lsp `TestArraysInTheEditor`, example
  `arrays`. Docs: chapters 4 (Arrays), 8 (constants as generic arguments),
  13 (arrays in a C struct, large values), cheat sheet, stdlib reference.
  _Std:_ SHA-256 on arrays — `sha256` 113–135 ms → 86–91 ms (four runs
  each; `bench/results.md`). **Also (user note):** field and enum-member
  annotations now sit on the line above, as on declarations, in the
  formatter (`@key("x")` / `id: i64`), in docs, examples and tests.
  **Bugs found on the way:** `receiverViews` would have made an array
  assignable to a `List` had it listed the list view (kept local to method
  lookup); `Channel<Array<u8, 200>>(…)` — a constant before `>>` — parsed
  as a shift (now a type-argument context stops at `>>`); a List method that
  lends storage (`withRaw`) found through the array→List view would have
  written to a copy (the array has its own). Windows and WSL green.
- **2026-10-04 — C2 part 1: `io.Stream` (D128).** _Language:_ suspending trait
  methods work through trait objects (the checker refused them: "suspending
  trait methods are not supported yet") — a slot declared `suspends` holds the
  coroutine form of the impl's method (a ramp for a plain impl, so `fs.File`
  and `net.Conn` fit one trait), `callVirtual` starts it as a task and awaits it
  (`sema.ObjectSigs`, `vtableThunk`); default arguments declared in the trait now
  apply to calls through an object (`traitParamDefault`). _Std:_ `io.Stream : Closeable`
  (read, readExact, readLine, write, writeText, shutdownWrite), `io.TooLong`
  (moved from `net`, every use migrated), implemented by `net.Conn` (its methods
  moved into the implement) and `fs.File` (new `readExact`/`readLine`/`writeText`/`shutdownWrite`;
  `readLine` is stateless over `readAt`). _Tests:_ `std/fs/fs.test.vs` (a file through the object and a
  generic bound, lines, ceilings), `std/net/net.test.vs` (a suspending call on a
  real connection; a std test file may import only what its module already
  does — `io.test.vs` importing `fs`/`net` made every program using `io` load
  them, which the golden tests showed as shifted lambda numbers), example `streams`. Docs: stdlib reference, chapter 16, spec D40 addendum.
  Not yet: `http` over `Stream` (the server and client still name `net.Conn`), the
  compressors. Windows and WSL green.
  **Bug found on the way:** a cycle of constants with declared types
  (`const A: i64 = B + 1` / `const B: i64 = A`) was accepted without a
  diagnostic when the checker met `A` first — the evaluator treated an
  unreached constant as "reported at its own check", which found `A` already
  done. It now checks the constant it meets (`sema/consteval.go`,
  `TestConstCycleIsAlwaysReported`); seen through `use io` once `io` gained a
  test file, as the conformance harness keeps std test files.
- **2026-10-04 — C2 part 2: `http` over `io.Stream`.** The server's
  connection loop, request reading, body framing and `BodyWriter` take an
  `io.Stream` (the peer's address is passed next to it: a stream has none);
  the accept loop still produces `net.Conn`. Needed a language decision
  (user, recommended of 3, spec D35 addendum): a trait may require
  `Sendable`, so `trait Stream : Closeable + Sendable` objects cross into
  `withTimeout`/`async` — implementors checked at their `implement`, boxing
  at the conversion; plus `with s = <object>` accepts an object whose trait
  requires `Closeable`. Conformance `D35-sendable-supertrait*`; `veles test
std/http` 105 passed. `tls.Conn` joins with E1.
- **2026-10-04 — C2 part 3: `std/compress` (D124) and `http.testServer` (D130) — C2 done.**
  _Std:_ `std/compress` in Veles: a resumable inflater (the bit buffer in locals,
  48 bits at a time, restart at the last whole symbol), a deflater (hash chains, lazy
  matching from level 4, per block the cheapest of stored/fixed/dynamic, length-limited
  Huffman by the two-queue method with Kraft repair, a sliding window so a stream
  compresses in constant memory), CRC-32 by slicing-by-eight, `gzip`/`gunzip`/`deflate`/`inflate`
  (`max:` is a ceiling on the output, 64 MiB), `GzipWriter`/`GzipReader` over `io.Stream`,
  `GzipEncoder` (no stream; what the HTTP body writer holds under a lock).
  _http:_ `http.compress(minBytes:, level:)`, `http.decompressRequests(max:)`, `.gz` siblings in
  `http.files`, `http.testServer` (the tests' own helper, migrated: 25 call sites).
  _Measured_ (`bench/gzip`, `bench/gunzip`, Go's `compress/gzip`): 1.1× / 1.5× on 2 MiB of word
  text; on 0.8 MB of Markdown prose 2.3× / ~3× (it began at 2.3× / 4.6× — the CRC went from
  byte-wise to slicing-by-eight, and `slice`/`addAll` from a push per element to one `memcpy`,
  8× on bytes); output 0.7–1.7% smaller than Go's. The other benchmarks did not move
  (json 1.0×, sort 0.8×). `examples/httpd` now compiles 5.6 MB of IR (was 4.4) and links a
  2.7 MB binary (was 2.45) because every function of an imported module is emitted — the
  case for dead-function elimination in codegen. _Tests:_ `std/compress/compress.test.vs` (15:
  Go-made vectors, every level, damaged/cut/oversized input, a reader fed one byte at a time,
  the writer with flushes), `std/http/compress.test.vs` (13: which responses are touched, a
  stream, over a socket, `.gz` siblings, request bodies and their ceiling),
  `testserver.test.vs`, examples `gzipstream` and `traitobjects` (extended), conformance
  D28 (`addAll` on a `List`); every output also decoded by Go's `compress/gzip` (80 + 32 files,
  scratch program). **Language/compiler bugs found on the way:** an impl method that throws
  less than its trait declares returned a mis-tagged `Result` through a trait object (a crash
  — now every impl carries the trait's `throws`, spec D40 addendum); `try` on a call of a
  function-typed field was an internal compiler error (a nil `*VarRef` in `errPolyCall`); the
  `is Trait` tables vanished when the checker needed a second round (`c.tt` was not reset with
  the Program — codegen then referenced an undefined `@it.…`); a const cycle slipped through
  (see C2 part 2). **Not done:** `inflate` could write into a sized buffer instead of pushing
  (the safepoint poll in each inner loop and the oversized-shift selects show in the IR);
  levels 7–9 on large inputs are slow (no zlib-style give-up on a long chain). Windows and WSL
  green (see below).

- 2026-10-04 — **unused-import lint and generics.** `use http { field }` warned "imported but never used" in programs importing `std/http` because `field` is only used inside the generic `screen<E>` in `auth.vs`, whose body is checked only when instantiated. `lintUnusedNames` now skips a file holding a generic that was never instantiated (sema/checker.go `hasUncheckedGeneric`); conformance D85-named-imports-generic. Limit: a truly unused import in such a file stays unreported until the generic is used somewhere.

- 2026-10-04 — **C3 `std/config` (D125) done.** Decided with the user first (all recommended): the
  derive writes `Decodable.schema(format, keys)` beside a derived `decode`; an empty variable is a
  value; dotenv has no interpolation. Built: `std/config/{config,dotenv}.vs` — `load<T>(files:,
prefix:)` (environment, files last to first, field defaults), `describe<T>(prefix:)`,
  `config.Error` (one `Problem` per variable, named by the variable with its source, a `Secret`
  never echoed), dotenv reader (quotes, escapes, multi-line, `export`, ` #` comments, no
  interpolation), JSON files by field name; the schema types `Schema`/`SchemaField`/`SchemaKind`
  and `KeyStyle.UpperSnake` in the prelude (homed in `codec`), `sema/derive_schema.go`, and a
  `schema()` for every built-in decodable, `Duration`, `Timestamp` and `Secret`. Tests:
  `std/config/config.test.vs` (19), `dotenv.test.vs` (10), the derive printer test, conformance
  D04; example `examples/config`; docs chapter 22 + stdlib reference + chapter 18 note;
  `veles new --template server` reads `HOST`/`PORT`/`.env` through `config`. **Compiler bugs found
  on the way:** a module could not declare a type named `Error` (the `error` sugar resolved the
  trait by scope and found itself — `ast.ImplDecl.ErrorSugar` now names the prelude's); the
  derive printer had no case for a list literal. **Not done:** a `Map` or list-of-structs field
  has no variable form (a panic naming the field on the first call, not a compile error); no
  `os.setEnv`, so tests give a lookup to the private `loadWith`; `.env` search upward is not
  offered. Next in the build order: C5/C6 (`std/fs` additions, endian bytes).

- 2026-10-04 — **C5/C6 done: endian bytes and the `std/fs` additions (D130).** `fs.lines` decided
  with the user first (Result items + `Closeable`, `max:` required). Built: `std/prelude/endian.vs`
  (generated once by a scratchpad Go program, committed as source): `toBeBytes`/`toLeBytes`/
  `fromBeBytes`/`fromLeBytes` on all ten integer types over `Array<u8, N>`, twelve readers on
  `List<u8>` (null when out of range), twelve writers on `MutableList<u8>`; `fs.writeAtomic` (C:
  temp file, fsync, rename, directory fsync; `MoveFileEx` with write-through on Windows),
  `fs.copy`, `fs.lines`, `File.seek`/`sync`/`lock`/`tryLock`, `fs.Lock` (runtime `veles_os.c`).
  Tests: `std/prelude/endian.test.vs` (9), `std/fs/fs.test.vs` (+9), example `examples/binfile`,
  docs chapter 15 and the reference. **Found on Windows:** `LockFileEx` is mandatory against reads
  of the locked range (a second handle got "Permission denied" reading the file) — the lock is now
  one byte at offset 2^63−2, advisory like `flock`; a file opened `FILE_APPEND_DATA` alone cannot be
  locked — append now opens with write access and writes at the end by overlapped offset; a file
  open elsewhere cannot be renamed over (documented, the example closes its handles). **Not done:**
  `http.testServer` was C2; password hashing (argon2id, needs a native binding) is open;
  byte-range and shared locks are not offered. Next in the Build order: step 12 C8 template
  literals (D129 part 1).

- 2026-10-04 — **Test files (D78 amended) and C8 template literals (D129 part 1).** _Test files_ (the
  user): a top-level `fun` in a `*.test.vs` file is an error with a fix that writes `test fun`; a type,
  value or helper declared there is an error when a non-test file names it (`declaredInTestFile`); the whole
  tree migrated (12 test files, one Go test's embedded source); tests `TestTestFile…` in
  `sema/testfile_rules_test.go`. **The user's "later"** — exporting test code so people can write reusable test
  libraries — is open (checklist §11). _Standing rule_ (the user): everything not done goes into checklist
  §11 in the same change (CLAUDE.md, memory); the deferred items of C2, C3, C5/C6 were written there.
  _C8:_ `tag"…"` parsed (adjacent string after a name or `module.name`), lowered by the checker to the call
  `tag(pieces, values)`, `@template` validated, formatter/`Dump`/grammar, example `examples/features/templates`, docs 2
  and 14; no design question was open (D129 fixes the form). **Not done:** `std/db`, std tags,
  semantic tokens, tagged multi-line/raw literals (checklist §11). Next in the Build order: step 13 C7, the
  HTTP client over TCP (D127), then C4 health endpoints (D133) and `std/otel` (D126).

**Progress 2026-10-04 (C7, the HTTP client, D127).** `std/http/client.vs`: `http.fetch`, `get/head/post/put/patch/delete`,
`http.Client` (pool of keep-alive connections keyed by `scheme://host:port`, lazy idle expiry, one retry on a stale
pooled connection for repeatable methods), `Payload`, `ClientResponse` / `ClientBody`, `FetchError` / `FetchKind` /
`StatusError`. One total deadline covers the request and its body; redirects for GET/HEAD with credentials dropped
across hosts; bodies bounded where read (`max:`, 64 MiB default); the response head parsed as strictly as the
server's request (two lengths, folded or malformed headers, a body cut short are errors); `retry:` with 100 ms
doubling backoff for idempotent methods. **One design question** (asked and answered): `json: T? = null` cannot be
inferred, so the body is a `Payload` value (D127 addendum). Pinned by `std/http/client.test.vs` (42 tests: a real
`testServer` for the behaviour, canned bytes on a raw socket for the hostile answers), `examples/httpclient`, docs
chapter 17 "Calling other servers", reference/stdlib. The example caught a bug the in-module tests could not: the
`pool` field was not `private`, so no other module could construct a `Client` (D28). Four test helpers in other
`*.test.vs` files (`head`, `post`, `fetch`) collided with the new public names and were renamed. **Not done**
(checklist §11): https/TLS (E1) and proxies, transparent gzip, per-phase timeouts, streaming uploads, cookie jar,
`traceparent` (with `std/otel`). Next in the Build order: C4 health endpoints (D133), then `std/otel` (D126), E1.

**Progress 2026-10-04 (C4 health endpoints, D133).** `std/http/health.vs`: `http.Health` (`check<E>(name, f, timeout: 2s)`, `endpoints(live:, ready:)` middleware, `stopping()`), `http.serve(health:)`. `/healthz` runs no check; `/readyz` runs all at once, one task each behind a panic boundary, `200`/`503` with `{"status", "checks"}`, details to the log only. `Response.quiet` (private, carried through every copy) keeps probes out of `logging()` and `serve`'s line. The server template (`veles new --template server`) uses it. **One design question** (asked and answered): `serve(health:)` flips readiness at the stop (D133 addendum). Pinned by `std/http/health.test.vs` (14 tests), the driver template tests, docs chapter 17 "Health checks", reference/stdlib. **Not done** (checklist §11): the readiness 503 window during a stop is nearly empty today (serve stops reading at once; a `preStop`-style delay is the user's choice), result caching, `degraded`, the OTel span exclusion. Next in the Build order: `std/otel` (D126), then E1 TLS.

**Progress 2026-10-05 (`std/otel`, D126).** `std/otel` (data, attr, proto, trace, metrics, otel): spans with the current span as a task-local (`TaskLocal.bind`, now public, and `LocalBinding`), `otel.inSpan`, W3C `traceparent` parse/write, parent-based trace-id-ratio sampling; counters, up-down counters, gauges and histograms aggregated per attribute set with a series cap; an OTLP/protobuf encoder written by hand (checked against the protobuf documentation's own byte examples); a pipeline with bounded queues, retry with backoff, drop counting, `flush`/`shutdown`; `http.otlp` (gzip, `Secret` headers), server spans and client spans with `traceparent` in `http`, the `std/log` bridge (OTLP records, `trace_id` in lines). **Two design questions** (asked and answered): the otel↔http cycle (explicit `exporter:`) and binding a span (`bind` + `inSpan`) — D126 addendum. **Deviation, forced by the language:** `with otel.start(...)` cannot flush at close (`Closeable.close()` cannot suspend), so `shutdown()` does it; recorded for the user. Pinned by 44 tests in `std/otel`, 12 in `std/http/otel.test.vs` (real collector, server and client spans, probes left out), the log bridge tests, `examples/telemetry`, docs chapter 23, reference/stdlib. **A runtime bug found and fixed on the way:** `veles_list_append_list` (`addAll`/`concat`) sized the grown buffer in elements, not bytes — a heap overflow for any list wider than bytes, seen as a collector crash; regression test in `std/prelude/list.test.vs`. Four debugging sessions' worth: the crash was reproduced single-threaded, narrowed from the collector's corrupt header to the structs written over it, then to a 15-line program with no otel. **Not done** (checklist §11): runtime metrics (E5), `std/db` spans (E2), `https` export (E1), the flush-at-close language question, exponential histograms/exemplars/links/baggage, attribute limits. Next in the Build order: E1 TLS (D128), then E2 `std/db` (D129 part 2) + password hashing.

**Progress 2026-10-05 (E1 TLS, D128).** `std/tls`: `connect`, `client` (over any `io.Stream`), `Options` (`roots`, `serverName`, `alpn`, `dangerouslyAcceptAnyCertificate`), `Conn` (`io.Stream`, ALPN, strict about a missing `close_notify`), the server side (`Certificate.load/fromPem/replace`, `listen`, `wrap`, handshake on first use, `reloading` keeping a certificate current). The engine is `runtime/c/veles_tlsio.c`, transport-free: SChannel (Windows; server keys stored under a per-process name, swept after a crash) and OpenSSL via `dlopen` (Linux; macOS untested). `http` clients speak `https://` (`Client(tlsOptions:)`, `http.otlp(tlsOptions:)`; the client was moved from `net.Conn` to `io.Stream`). Pinned by `driver/tls_test.go` (Go's `crypto/tls` serving: good certificate, wrong name, unknown authority, system roots, the escape hatch, TLS 1.2 and 1.1, ALPN, truncation, a 1 MiB echo with a concurrent reader, an HTTPS client reusing a pooled connection) and `driver/tls_server_test.go` (a Veles server against Go clients: RSA/ECDSA keys in four encodings, ALPN, 20 000 lines each way, the protocol floor, an idle client, certificate reload and a refused renewal, a mismatched key), green on Windows (SChannel) and WSL (OpenSSL 3), and a real request to example.com. Docs: chapter 16 "Encrypted connections", 17, 23, reference/stdlib. **Bug found on the way (decision-free, fixed):** the `std/http` otel tests were flaky at HEAD (`health probes are not traced` failed in most runs): the other tests' requests ran while the process-wide pipeline was on and their spans landed in the capture; the tests now look only at the spans of their own server's port. `http.serve(tls: cert)` (the user's choice of three; `TestHTTPSServe` serves HTTPS from Veles to Go's client: keep-alive, a 96 KB echo, a cleartext client dropped). **Not done** (checklist §11): macOS, mutual TLS, revocation, resumption. Next in the Build order: E2 `std/db` (D129 part 2) + password hashing.

**Progress 2026-10-05 (E2 `std/db` and password hashing, D129 part 2, D130).** `std/db` (sql, url, wire, scram, conn, row, pool, errors): `sql"…"` → `Sql` (parts + args, splicing without renumbering, `db.ident`, `dangerouslyRaw`), `db.Param`/`Arg` for the numbers, `bool`, `string`, `List<u8>`, `Timestamp`, `Duration`, `Uuid`, `T?`, `Secret`; the PostgreSQL wire protocol in Veles over `net`/`tls` (startup, SSLRequest, SCRAM-SHA-256 with the server proved, cleartext only over TLS or loopback, MD5 refused, extended queries, every result as text); `db.open` → `Pool` (a task checks idle connections; `size` permits; connections kept and reused), `query<T>`/`queryOne<T>`/`exec`/`ping`, `begin` → `Tx` (`commit`/`rollback`; abandoned = connection dropped), per-call timeouts (a timed-out or cancelled statement drops its connection and the pool carries on), a span per statement while `otel` runs; rows through a `codec.Decoder` (`RowDecoder`: struct = row in snake_case, `@key(db:)`, scalars, NULL, bytea and arrays as lists, timestamps as RFC 3339, `Uuid`). Password hashing: BLAKE2b and Argon2 (d, i, id, any lanes) in `std/crypto`, `hashPassword`/`verifyPassword` in the PHC format — **60 ms per hash at -O2**, checked against RFC 9106's three vectors, RFC 7693's BLAKE2b vectors and a hash made by argon2-cffi (m=65536, t=3, p=4). Pinned by 34 tests in `std/db` (the ones that need a server return at once unless `VELES_TEST_PG_URL` is set), 8 in `std/crypto`, and `driver/db_test.go`, which starts a throwaway cluster (trust login, a SCRAM role, TLS with its own certificate) from the binaries in `VELES_PG_BIN` and runs them — green on Windows (PostgreSQL 16.4) and WSL (16.4); the SCRAM exchange, TLS verification and the escape hatch, constraint codes, a refused statement leaving the pool usable, transactions, fifty tasks on four connections, timeouts and cancellation are all tested against the real server. **A compiler diagnostic caught a real bug on the way:** `Cursor` methods mutated a copy; the checker's "'c' is a copy of the caller's" warning pointed at it before a single test ran. **Bugs found and fixed:** the SCRAM state was copied between messages (the server's proof was checked against an empty signature — caught by the first run against a real server); the Windows reset after a refused login is recorded in §11. Docs: chapter 24, chapter 19 (passwords), reference/stdlib. **Deviations** (spec D129/D130 addenda): the driver is Veles, not libpq; argon2id is Veles, not libargon2; `Tx` close drops the connection. **Not done** (checklist §11): prepared-statement cache, streaming results, `COPY`, `LISTEN`, other databases, `interval`/`json` into typed fields, `examples/pgnotes`. Next in the Build order: step 17 B14 part 4 `const fun`, then E7 packages.

**Progress 2026-10-05 (B14 part 4, `const fun`, D113 part 3).** `const` is a modifier of `fun` (free functions, methods, statics, `extend`/`implement` methods). A constant that calls one runs the callee's checked HIR in the part 1–2 evaluator (`sema/consteval.go`, new `constfun.go`, `constnum.go`): variables are cells (so `&x`, receivers and list element places are pointers), structs/tuples/arrays are copied where stored, loops/`return`/`break`/`continue`/labels, generics, methods, a struct's `init` block, `Display` impls that are `const fun`. Rules are checked at the declaration (`validateConstFun`: only `const fun` callees, no `unsafe` outside std, no module-level `val`/`var`, no tasks/`with`/function values/`throws`, built-ins from a whitelist); failures (panic, overflow, index, division, `pow`) are errors at the constant with the call chain; budget 10 M steps (`--const-steps n`) and 4096 calls. Integer built-ins exact, IEEE-fixed float ones (`sqrt`, rounding, `min`/`max`, `mod`…) evaluated; libm functions (`sin`, `exp`, `log`, `pow`…) deliberately not (libraries differ in the last bit) — a deviation from the spec text, in the D113 addendum. std marks: `StringBuilder`, the Veles-written string functions, number `compareTo`/`toString(radix)`, an enum's synthesized functions. Pinned by `driver/constfun_test.go` (20 tests, each function run at compile time and at run time, debug and release), `sema/testdata/conform/D113-const-fun.vs`, parser/format round trips, `driver/conststeps_test.go`, `examples/features/constfun`; green on Windows and WSL. Docs: chapter 2, reference (errors, cheatsheet, stdlib). **Not done** (checklist §11 "Deferred from `const fun`"): `throws`, function values (so no `map`/`filter`), derived `equals`/`hash`/`toString`, most prelude collection helpers and other std modules, libm floats, evaluator speed (~6 M steps/s). Next in the Build order: E7 packages (D131, D132).

**Progress 2026-10-05 (E7 stage a, the manifest, D138).** `veles.toml` is read by a real TOML subset reader (`sema/toml.go`: tables, strings, numbers, booleans, multi-line lists, one-line inline tables; dotted keys, `[[…]]`, floats and dates refused with a message) and `sema/manifest.go` interprets it strictly — an unknown table or key is an error naming what exists, every message carries `file:line`. New: `[dependencies]` entries as a path (the short string form stays a path; `"1.4.2"` bare is an error showing the table to write), `{ registry = "owner/name", version }`, `{ git, version | commit }`; `[dev-dependencies]`; `[workspace] members`; `[package]` `description`, `license`, `veles`. `sema/semver.go`: `Version`, `ParseVersion` (`v` prefix, pre-release, no build metadata), ordering, and `MajorID` by Cargo's rule (`1.4.2`→1, `0.3.1`→0.3, `0.0.2`→0.0.2). The loader still resolves path dependencies only; importing a registry or git one now says it "cannot be fetched yet". Pinned by `sema/toml_test.go`, `manifest_deps_test.go` (33 error cases), `package_remote_test.go`; docs: chapter 11 "The manifest in full", reference/errors. **Not done** (checklist §11): minimum-compiler enforcement, workspace patterns, dev-dependency loading, the policy table. Next: stage (b), the loader keyed by (source, major).

**Progress 2026-10-05 (E7 stage b, package identity, D138).** The loader keeps one instance per identity in a table shared by the whole program (`depTable`): the same directory reached by two importers, or under two local names, is one `Package` with one set of modules, symbols and module-level state; a dependency that leads back to the entry package is an error ("is this package itself"). The module-key prefix is `dep/<the package's own manifest name>/` (not the importer's alias, so symbols do not depend on who imports first); two different packages with one name get `dep/<name>-<hash>/` (a `~` broke LLVM symbols — found by the first run). A dependency whose local name is a std module or a directory of the importer is an error naming the manifest line; a workspace root without `[package]` cannot be depended on. Pinned by `sema/package_identity_test.go` (diamond, same name twice, only-direct-requires, name clashes, self cycle) and `driver/packages_identity_test.go` (builds and runs a diamond with a shared `Atomic` counter and two `pg` packages side by side). Docs: chapter 11. Identity is the resolved directory for path dependencies; (source, major) for registry/git arrives with stage (d), where `prefixFor` takes the major. **Not done:** `use` of a transitive dependency stays "unknown module" (correct, but the message does not say which manifest to add it to). Next: stage (c), workspaces.

**Progress 2026-10-05 (E7 stage c, workspaces, D138).** `[workspace] members` at a directory makes `veles check`, `check --fix`, `test` and `build` run over every member in order (`driver/workspace.go`), each as the package it is (path dependencies between members already worked); pointed inside a member the tools are unchanged. A failing member does not stop the others and the summary names them; `build` skips library members and writes each program to `bin/<name>` under the root (a `./app` would collide with the member directory `app`, found on Linux); `run` and `-o` at the root are refused with the member list; a root with `[package]` is the first member; duplicate names, a missing manifest and a nested workspace are errors. Pinned by `driver/workspace_test.go` (every member checked/tested/built and the built app run; every failure reported; edges). Docs: chapter 11 "Workspaces". **Not done** (checklist §11): `veles doc`, the language server at a workspace root, `veles new --template workspace`. Next: stage (d), fetching.

**Progress 2026-10-05 (E7 stage d, fetching, D138).** New package `fetch` (the compiler core stays free of network and git: `sema.ResolveRemote` is the hook the loader calls, the driver and the language server link `fetch` in). `Resolve` walks the graph (path dependencies in place, git/registry ones fetched to read their own requirements), selects per (source, major) the highest version anyone asked for — minimal version selection with Cargo's major rule — and hands the loader a directory per (importer, local name); the loader's one-instance-per-identity table (stage b) takes it from there. Git: tag `v<ver>` or `<ver>`, or a commit pin, fetched into a temporary directory with `info/attributes` turning every conversion off, `.git` removed, so the files are the committed bytes on every platform. Proxy (`VELES_PROXY`): `GET <base>/<owner>/<name>/@v/<version>.zip`, unpacked with path/link/size/duplicate refusals. Module cache `~/.veles/pkg` (`VELES_HOME`), a marker with each entry's hash; every use re-hashes it (an edited entry is a hard error). `veles.sum` (`git|registry <source> <version|commit:<full>> h1:<sha256 over sorted path + file-hash>`): recorded on first fetch, a mismatch is a hard error that names the line and why, nothing is cached from a failed check; placed at the workspace root for members. Commit pins: tag-and-commit or two-commits of one (repository, major) are errors naming both; a fetched package that pins a commit warns. A fetched package cannot have path dependencies. `veles fetch [dir]` resolves on its own; the language server is offline and says "run `veles fetch`". Pinned by `fetch/fetch_test.go` (14 tests: tag, cache reuse offline, MVS with a shared util and a second major, moved tag, tampered cache, wrong sum line, byte-exact checkout under `eol=crlf`, commit pins, library warning, path dep refused, missing tag/offline, no-remote touches nothing, workspace sum, proxy, proxy errors, unsafe archives, sum format, tree hash) and `driver/fetch_test.go` (builds and runs an app on git dependencies from local repositories, then checks from the cache alone). Found by the first run: the cleanup after a failed fetch read a result variable blanked by the failing `return`, and Windows cannot delete git's read-only files. Docs: chapter 11 "Dependencies from git and the registry", getting-started, cheatsheet. **Not done** (checklist §11 "Deferred from E7 stage (d)"): add/update/remove/vendor, dev-dependencies, the rest of the registry protocol, subdirectory packages, cache cleaning. Next: stage (e), `veles add/update/remove/deps/vendor`.

**Progress 2026-10-07 (E7 stage e, the dependency commands, D138).** `veles add <spec>[@version] [--as n] [--dev] [--dir d]`, `update <name>[@version]... | --all`, `remove <name>...`, `deps [dir] [--why name]`, `vendor [dir]` (`driver/pkgcmd.go`; the logic in `fetch/edit.go`, `versions.go`, `deps.go`). The three that change things edit `veles.toml` as text — one line, comments/order/CRLF kept (`fetch.SetDependency`/`RemoveDependency`; an empty table is filled below its opening comments; `[dependencies.x]` tables are refused) — then resolve, and write the old file back if resolution fails. `add` reads a path, a git URL (`https://`, `git@host:`, `github.com/o/r`, `file://`) or `owner/name`; the newest stable tag when none is given (`git ls-remote --tags`, or the proxy's new `@v/list`); a pre-release only when named. `update` stays in the major, mentions a newer one, takes it as `name@version`; a path or commit pin is skipped under `--all`. `remove` and `update` prune unused `veles.sum` lines (`PruneSum`, `PruneAllSum`; not in a workspace). `deps` prints the selected build as a tree (`(*)` for a second sight, `a -> b` where selection raised it) and `--why` lists the chains. `vendor` copies every mentioned version to `vendor/` beside the sum; while it exists builds read only from it (still hash-checked; a missing package says `veles vendor`); `fmt` skips `vendor/`. Spec: D138 addendum records the defaults taken in a–e. Pinned by `fetch/edit_test.go`, `fetch/deps_test.go` (versions, proxy list, tree and why, vendor offline and tampered, prune), `driver/pkgcmd_test.go` (the whole life cycle against local repositories: add, add twice, add a missing version leaves the manifest byte-identical, update with a newer major, deps, vendor then check with the cache and repositories gone, remove, workspace root refused, path dependency), `TestFormatSkipsVendor`. Docs: chapter 11 "Managing dependencies", getting-started, cheatsheet. **Not done** (checklist §11 "Deferred from E7 stage (e)"): name-clash check at `add`, `update --major`/`--check`, whole-workspace fetch/deps/vendor. Next: stage (f), capabilities computed from source, `veles audit` and the policy table.

**Progress 2026-10-07 (E7 stage f, capabilities and policy, D138).** `fetch/caps.go`: `PackageCaps` parses a package's own `.vs` files (not tests, nested packages, `vendor/`, hidden directories) and reports seven capabilities with `file:line` evidence — `unsafe` (a block or an `unsafe fun`), `extern` (a block or an `extern "C" fun`), `native` (a `[native]` table), and `net`/`fs`/`os`/`ffi` from the std modules whose purpose they are (`net tls http db otel`→net, `fs`, `config`→fs+os, `os`, `ffi`); a local module that shares a std name is not std. An AST walk by reflection (the tree has no visitor). `[policy] deny = [...]` / `allow = { cap = [package…] }` is a manifest table (`sema.ManifestPolicy`), enforced inside every resolution on registry and git packages (path dependencies are the project's own code); a refused build, `fetch` or `add` names the package, the first use and the `allow` line, writes nothing to `veles.sum`, and `add` leaves the manifest as it was; a workspace member without a policy follows the root's. `veles audit [dir] [--detail]` lists every package with its own capabilities (and what its dependencies add), reports the policy without refusing, exit 1 when something is denied; `veles add` says what the new package can do. A first design that followed std imports transitively reported `os` for every program with a socket (`net` imports `os` inside) — dropped for the explicit purpose table; found by the first test run. Pinned by `fetch/caps_test.go`, `fetch/policy_test.go` (refusal, allow by source and by name, offender named at depth, path dependencies spared, report totals, SkipPolicy, workspace inheritance), `sema/manifest_policy_test.go`, `driver/pkgcmd_test.go` `TestCapabilitiesAuditAndPolicy`. Docs: chapter 11 "What a dependency can do", getting-started, cheatsheet; spec D138 addendum. **Not done** (checklist §11 "Deferred from E7 stage (f)"): `veles deps` does not show capabilities; licences; the listed tier's gates and signed attestations (stage g); a capability for time/randomness is deliberately not included. Next: stage (g), the registry protocol, tiers and attestations.

**Progress 2026-10-07 (E7 stage g, the registry, D139).** Decided with the user first (D139): publishing both ways, `unlisted` as a policy capability, portable ed25519-signed reviews, a signed `.info` with a pinned key. Built: **protocol** (`docs/documentation/reference/registry-protocol.md`): `list`, signed `.info` (`{version,time,hash,tier,yanked,yankReason,key,sig}`, signature over one line of text so there is no JSON canonicalisation), `.zip`, `.attest`; writes `PUT .zip`, `POST .attest`, `PUT/DELETE .yank` with a bearer token that belongs to an owner. **Client** (`fetch/info.go`, `cache.go`): `.info` is verified against `[registry] key`, the archive must hash to `info.hash`, which must agree with `veles.sum` (a lying registry is a hard error, nothing cached); the tier, yank state and hash go into the cache marker; a yanked version is refused for a resolution with no `veles.sum` line and warned for one with it; a new resolution, `fetch` and `audit` ask the registry again about cached packages (`Fetcher.Refresh`); `add`/`update` skip yanked versions (`LatestUsable`). **Policy** (`sema/manifest.go`): `unlisted` joins the capabilities (a git package is always unlisted), `[policy] trust = { name = "ed25519:…" }`, `require = { reviewed = N }`, `allow` takes capabilities and claims; `[registry] url, key`; `[package] registry = "owner/name"`; a workspace member inherits the root's registry and policy. **Reviews** (`fetch/attest.go`): `attest v1 …` lines verify on their own; `LoadAttestations` (the project's `attestations/*.attest`, beside `veles.sum`), the registry's `.attest`, `TrustedReviewers` counts distinct trusted keys for exactly these bytes; keys in `~/.veles/keys/<name>.key` (0600, never overwritten). **Commands** (`driver/registrycmd.go`): `veles publish` (checks the package first, zips without `.git`/`vendor`/`bin`/`veles.sum`/hidden files), `yank [--reason] [--undo]`, `attest keygen|sign [--claim] --key [--push]|verify|import`; `audit` gains a tier column and the reviewers; `add` prints the tier. **Reference server** (`registry` package, in memory, token-per-owner, a replaceable listing gate whose default is "1.0.0 or later, not a pre-release"). Pinned by `registry/server_test.go` (publish and its refusals, tier, pinned key wrong, archive that does not match its metadata, unpinned first use, yank both ways, reviews: untrusted key / other bytes / other claim / tampered line do not count, project-held and registry-held reviews do, an `allow` exempts, key and statement round trips, every field of `.info` covered by the signature), `driver/registrycmd_test.go` (the commands end to end), `sema/manifest_policy_test.go`. Docs: chapter 11 "The registry: publishing, tiers and reviews", the protocol reference, getting-started, cheatsheet. **Not done** (checklist §11 "Deferred from E7 stage (g)"): the hosted service and its gates, a transparency log, replay protection, persistence and a `registry serve` command. Next: stage (h), the driver test against bare git repositories and the reference server on Windows and WSL, `examples/features/packages` with one published dependency, then tick E7 in the checklist.

**Progress 2026-10-07 (E7 stage h, the end-to-end test; E7 done).** `driver/packages_e2e_test.go`: one program built from every kind of dependency at once — a path, git tags with minimal version selection (util 1.0.0/1.3.0/2.0.0, mid needing 1.3.0), two majors of one repository under two names, a commit pin, and a package published to the reference registry through `fetch.Publish` — run (`7 130 130 200 1 12`), `veles.sum` checked for all six kinds of line, `deps` and `audit` seen to know them, and `check` repeated with the repositories and the registry deleted (the module cache alone). `TestMovedTagStopsTheBuild`: a tag force-moved after `veles.sum` recorded it, on a machine with an empty cache, is `checksum mismatch … Nothing was built`, exit 1, no executable. `examples/features/packages/app` now also uses a registry package, `veles/greeter 1.0.0`, from a committed `vendor/` and `veles.sum` (made by publishing to the reference server and `veles vendor`; `.gitattributes` marks them `-text` so no checkout rewrites a byte the hash covers), so it builds offline in `TestExamples`. Green on Windows; the new and affected tests also on WSL. E7 is closed: nothing in it is silently missing — everything not done is in checklist §11 under "Deferred from E7" (the hosted service and its gates, a transparency log, `package.vs`, publishing workspace members, `update --major`, and more). Next in the plan: the first unfinished item of the Build order after E7, then track A, then B.

**Progress 2026-10-07 (E7 review: documentation, bugs, .gitignore).** Documentation: a new chapter, `25-packages-and-the-registry.md` — the pieces and what a build does step by step, identity/majors/selection with a worked example, every file and whether to commit it, vendoring, the **trust model as a threat table with what is not protected**, the registry from the inside (owners, tiers, signed metadata, yanks, reviews, the two ways in, the honest status of the reference server), walk-throughs (add, audit, policy, review, publish, CI), reference tables (manifest keys, capabilities, environment, commands) and "when something is refused" (message → meaning → fix); linked from chapter 11, the index and the protocol reference. Bugs found by reading the new code adversarially and fixed, each with a test: git addresses (option/helper/path injection; a fetched package's manifest held to network schemes; `GIT_ALLOW_PROTOCOL`), archive names (`:`, `\`, `.git`, case twins), the **module cache keyed without the registry's address** (dependency confusion) and its **unauthenticated tier/yank** (now believed only if verified with the pinned key), **capability evasion** (an empty `fs/` directory, a nested `veles.toml`, `vendor/` all hid `use` lines), tokens over plain http, `publish` uploading files git ignores, statements with whitespace, `pkg-config` option names, key-file creation race, an import with no timeout, a dependency without a manifest read as the enclosing package (`ReadManifestIn`), a resolution failure aborting the whole load (now one diagnostic, the editor keeps working), a slow upload holding the reference server's lock, and `veles yank` needing a package. `.gitignore`: `.veles/`, `bin/`, `*.key`, `.env*` (not `.env.example`) and editor/OS litter; `veles new` writes the same for a new package; `veles.sum`, `vendor/` and `attestations/` are deliberately not ignored. Spec: D139 addendum. **Not done:** nothing new beyond checklist §11 "Deferred from E7".

**Progress 2026-10-08 (E3, the I/O reactor, and what measuring it found).** `bench/httphello` came first: 64 keep-alive clients × 250 requests against an in-process server, the Go reference the same with `net/http`. The first reading was **20.6× Go** on Windows (9.9× on Linux), and more threads made it slower (32 threads: 64 s of CPU for 3 s of wall time). A sampling profiler written for the purpose (suspend each busy thread, read RIP and the return addresses on its stack, symbolize the exe's COFF table and the DLLs' exports) found, in order: (1) **`veles_net_recv` allocated 64 KiB of GC memory per call**, would-block calls included — now it receives on the stack and allocates what arrived (5.3 s → 1.6 s single-threaded); (2) the socket wait rebuilt a `pollfd` array of every parked socket under the runtime lock — now **`runtime/c/veles_poll.c`**: epoll (one-shot, eventfd wake) on Linux, AFD poll requests on an I/O completion port on Windows (the readiness interface under `WSAPoll`, as libuv/mio/wepoll use it; base-handle fallbacks for layered providers), `poll()` with a self-pipe elsewhere; parked tasks are kept by descriptor (readers and writers apart, the descriptor armed for their union), one idle worker blocks in the reactor and `wake_worker` interrupts it (a Dekker-ordered flag, Go's netpollBreak), a close forgets the descriptor first; (3) **timers were an unsorted list** scanned whole by every worker's every pass and removed in O(n) — now a min-heap with each task's slot (a server's per-connection timeouts made this quadratic); (4) **`veles_lock_enter` gave a contended lock back and retried**, two hand-overs per entry — the runtime lock now keeps it (`veles_lock_enter_kept`; the heap lock, which the collector takes, keeps the old protocol — the first attempt deadlocked a collection at 4 threads, caught at once); (5) ~160 runtime-lock acquisitions per request: **`await`, a task's finish and a scope's wait and failure check are lock-free now** (a Dekker handshake on `waiter`/`state` — a cancelled target included, which the first version missed — and lock-free wakes, as channels already did); (6) idle workers **spin briefly** before sleeping (at most half the busy ones, Go's nmspinning) and `wake_worker` wakes no sleeper while one spins; (7) on POSIX the runtime mutex tries 64 times before sleeping in the kernel. Result on Windows: 1 thread 1.08 s, 4 threads 0.55 s, 8 threads **0.53 s vs Go's 0.43 s (~1.2×)**, 32 threads 1.1 s (was 3.3); Linux: 1 thread 0.87 s, 4 threads 0.66 s, 8 threads 0.80 s (Go 0.58 s). **A bug the new timer test found:** the executor's clock was `GetTickCount64` (15.6 ms steps) truncated to milliseconds, so on Windows **891 of 2000 sleeps returned early**; deadlines are monotonic nanoseconds now (`veles_time_monotonic_ns`, its frequency cached, exact integer scaling) and waits round up. Pinned by `driver/reactor_test.go`: 300 connections × 40 round trips plus two full-duplex sockets (2 MiB each way, read and write armed on one socket at once) at 1/2/8 threads, the same through the `poll()` backend on Linux (`runtimeTestFlags`), and 3000 sleepers with every third cancelled out of the middle of the heap, none early. Green on Windows and WSL. Spec D-entry addendum on the reactor; README. **Not done** (checklist §11 "Deferred from E3"): kqueue (A7), writev/readv, the runtime lock in `race_wait`/scope cancel/timers/socket waits (32 threads is still about twice 8), per-thread parking.

**Also 2026-10-08 (decision-free fixes found on the way).** A union of error types could not be written as a type argument — `Result<i64, IoError | io.TooLong>`, the type the checker itself prints — because `Name |` in a type argument was read as a constant expression (D121); names joined by `|` there are now an error union, and a constant parameter reads them back as `N | M` (`sema/testdata/conform/D45-union-type-argument.vs`). `throws A, B` in a declaration said "function needs a body"; it is now "error types are joined with '|', not ','" with a fix, and the declaration still means the union (`D45-throws-comma.vs`).

**`const fun`, the review list (notes C1/C2), 2026-10-08.** C2 (documentation) done: chapter 2 has how the evaluator works, a table of what a `const fun` may call (each row checked by a probe at compile time against run time), a worked failure with the real output, and when to use `const fun` vs `const` vs `val`; the self-host plan §10 says the evaluator is part of the port; the veles-develop skill has the contributor checklist. C1.a: **lambdas, closures (captures share their cells, as at run time), named `const fun`s as values and std's function-taking helpers** (`map`, `filter`, `fold`, `sortedBy`, `binarySearch`, `Map.mapValues`…) now run in the compiler; a lambda is held to the rules with the function it is written in; trait objects remain refused. C1.c: the function-free `List`/`Range` helpers, `Duration`, the `hex`/`base64` encoders and `std/utf8` are `const fun`, and `std/testdata/const-functions.txt` lists every std `const fun` so a dropped mark fails `go test ./std`. **Bugs fixed on the way:** an instance of a generic `const fun` was refused at std's declaration whenever its type argument kept it from running in the compiler — so `xs.max()` at _run time_ on a type with an ordinary `compareTo` failed to compile — it is now refused only if a constant runs it, with the reason; a `const fun` whose body did not check added a cascade at its constant ("this (\*sema.UnitConst) is not a constant expression"), and the evaluator's fallback message no longer prints Go type names. Pinned by `driver/constfun_test.go` (25 differential tests, debug and release), `D113-const-fun-cascade.vs`, `D113-const-fun-generic-instance.vs` and the updated `D113-const-fun.vs`.

**Progress 2026-10-08 (S5 / E4 P0: the self-hosted front end begins).** `selfhost/`: a Veles package (`veles.toml`, `main.vs`, `source/source.vs`) that ports Go's `source` — files, the line table, positions, lines, spans, diagnostics and their rendering — and `selfhost/selfhost_test.go`, the equivalence harness of the self-host plan §2: it builds `selfhost/` with a freshly built compiler and compares its output with the Go `source` package's, byte for byte, on every Veles file in the repository plus edge files (331 files, ~8 s). Gate P0 holds; a deliberately broken `line` (keeping a `\r`) fails it on the first CRLF file. Next: P1, the lexer (G1, tokens), which must also read files that are not valid UTF-8.

**Progress 2026-10-08 (E4 P1: the lexer in Veles, gate G1).** `selfhost/lexer/` ports the Go lexer: the token kinds in the Go order (constant tables for the keywords and the kind names), the scanner with automatic semicolons, interpolation and documentation comments, UAX #31 identifiers through tables generated from the Go lexer's own character classes (`selfhost/tables_test.go` regenerates them and fails when they are stale). `go test ./selfhost` now also runs gate G1: tokens, string parts, docs, comments, the module doc and every diagnostic agree byte for byte with `lexer.TokenizeAll` on all 357 corpus files, broken ones included. **Go bugs found by the port, fixed in both front ends:** a message naming `this` said `'this'` or `'self'` at random (map iteration); `"\é"` reported `'\Ã'` and a bogus invalid-UTF-8 error; the caret under every diagnostic counted bytes, so it stood to the right of the span after any non-ASCII character or tab (`source/render_test.go`). Next: P2/P3, the AST and its printer.

**Progress 2026-10-08 (E4 P2–P5: the AST, the printer and the parser in Veles; G2, G3 and G4).** `selfhost/ast/` (five sealed families, the S-expression printer with a faithful Go `%q`), `selfhost/parser/` (parser, decl, types, stmt, expr, pattern — every message Go's word for word). `go test ./selfhost` checks: the tree as `veles parse` prints it (G2) and every diagnostic (G3), byte for byte with the Go parser, on all corpus files plus four hand-written broken files covering each recovery path (79 diagnostics); the self-host sources are in the corpus, so the Veles front end parses itself and agrees (G4). 3 MB of std parses in ~1.3 s vs Go's ~0.55 s. **Language defect found and fixed (decision-free, no accepted program changes meaning):** a sealed-trait variant could be named as a type but never constructed as one — `val c: Circle = Circle(...)` was a type mismatch; a construction whose expected type is the variant now stays the variant (`sema/check_call.go` `variantWanted`, `D12-variant-values.vs`, docs chapter 9). **Not done:** P6 Layout, a span-printing dump (ast.Dump prints no spans, so G2 does not check spans), non-UTF-8 corpus files; checklist §8b.

**Progress 2026-10-08 (E4: the self-hosted front end reads the current language, rewritten as Veles).** The user's review: the port carried the removed spellings (`self`, `impl`, `fun <T> f`, `mut fun`, `use m.{ }`, `T::Item`, `{ e => }` handlers) and read like Go written in Veles. Now the Veles lexer has no `self` keyword and the Veles parser reports each removed spelling as an ordinary syntax error. The Go front end keeps them, to migrate old programs with `--fix`. The harness leaves a file where Go reports a removed spelling outside G1–G3 (`removedSpellings`; four files today), and `TestRemovedSpellings` checks that the Veles parser rejects each one and that `edge/p-removed` covers the whole list. The parser was rewritten idiomatically (self-host plan §11): `expect` returns `bool` and `expectIdent` returns `Ident?` instead of `(T, bool)`; `val _ =` is gone; `&value` replaces the `box<T>` helper; the Pratt loop is split into `parseInfix`/`parsePostfixOp`; a `DeclHead` replaces two 45-line doc/internal patch-up functions; `if (val x = …)` replaces 40 temp-and-null-check pairs; the token kinds, nodes and fields follow the language (`KwThis`, `KwSelfType`, `KwImplement`, `KwPublic`, `ThisExpr`, `left`/`right`); `veles fmt` is applied. G1–G4 hold over 367 files. **Compiler gap found and fixed (decision-free; only programs that were errors change):** `&e` on a temporary checked `e` with no expected type, so `&Leaf(…)` where a `*Leaf` was wanted was a `*Node` and `p: *i32 = &5` a `*i64`. The expected pointee now reaches the operand (`sema/check_expr.go` `pointee`, `D10-address-of-temporary.vs`, spec D10 addendum). The D12 addendum (a variant is a type of its own) is written. **The "type mismatch: expected 'TryExpr', found 'Expr'" the user saw** came from the stale `veles.exe` in the repo root, which the editor's language server runs; it predated the D12 fix and is rebuilt. **Formatter gap fixed:** `veles fmt` joined a `when` arm's patterns onto one line however the author broke them (the lexer's `endsStatement` arm became 230 characters); a break after a comma between patterns is now kept with a continuation indent, like a break after `=>` (`format_test.go` "an arm's patterns broken after a comma stay broken", chapter 1).

**Progress 2026-10-08 (formatter imports, opt-in explicit_return lint; user requests).** `veles fmt` writes one `use` per import by default — sorted, the standard library first — instead of one comma list per origin; the comma form stays valid and is what `[format] imports = "merged"` writes (`format.Options.Imports`, `sema` manifest key, `driver.StyleFor`, so `veles fmt`, `--fix` and the language server agree). std, examples, bench, selfhost and 53 documentation programs were reformatted (only `use` lines changed; no expected output moved). The manifest gains `[lint]`, opt-in lints of the package's own code (dependencies and std follow their own manifests): `explicit_return = true` reports the implicit return of a `{ }` function, method or block-lambda body as an error with a fix that writes `return` (`sema/lint_return.go`); `= expr` bodies, lambdas without braces, `if`/`when` blocks, unit and `Never` values are untouched. Tests: `format_test.go` TestImportStyles and the updated `use` cases, `sema/lint_return_test.go` (the lint and its fix on a package with and without the setting; manifest keys and their errors). Docs: chapters 11 and 25, the cheatsheet, `reference/errors.md`; spec addendum to the v0.28 `use` addendum.

**Progress 2026-10-08 (implicit_return levels; user).** The on/off `[lint] explicit_return` became `implicit_return = "full" | "lambda" | "expr"`: "full" (the default) allows a body's last expression as its value anywhere; "lambda" keeps it in every lambda and in `fun f() = expr` but asks a function's or method's `{ }` body for `return`; "expr" allows it only in bodies without braces. `sema/lint_return.go`, `sema/manifest.go`; `lint_return_test.go` checks the lines each level reports and the fix. Docs: chapters 11 and 25, `reference/errors.md`, the spec addendum.

**Progress 2026-10-08 (D140: `fun f() => expr`; user, recommended of 3).** An expression body is written after `=>`, so `=` only binds and `=>` only yields. The Go parser reads `=> expr`; the removed `= expr` is an error with a fix that writes `=>` (family removed-syntax), and `=> { … }` an error with a fix that removes the arrow. The formatter prints ` =>` and parenthesizes a returned lambda (`=> (x => x + n)`, as D106 for arms); the trait-method stubs a fix writes and the hover of a trait's default method use `=>`. The self-hosted parser reads `=> expr` only; `= expr` joined `removedSpellings` and `edge/p-removed`, `fun q() => { 1 }` the `p-signatures` edge. Migration: the parser's own fix applied by a scratch tool to 278 files — std, examples, bench, selfhost, the conformance cases, the codegen golden sources, the docs' and skills' programs, and the Veles programs inside Go test strings (1,956 bodies), plus 13 lines of `driver/constfun_test.go` and one `Sprintf` template by hand. Codegen goldens re-accepted (panic-site columns +1), two hand-written expected call sites moved one column, the vendored `examples/features/packages/app` greeter's hash updated, `std/const_test.go` cuts a signature at `=>`. Tests: `parser_test.go` TestD140ArrowBodies, a formatter style case. `go test ./...` green on Windows.

**Progress 2026-10-08 (D141: task handles stay in their scope; `await` gives the value; user, recommended of 3 each).** The user's question on `bench/httphello` (handles pushed into a list declared before the `scope`, awaited after it) found: a handle could leave its scope, and awaiting it after an early exit panicked at run time; in a fail-fast block a throwing child's handle was `Task<Result<R, E>>` whose `Err` was never observed. Probing found two bugs besides, fixed in the same change: a scope abandoned by a failed child left the suspending call its body was inside running after the scope had rethrown (an orphan, against D3), and `async` of a function returning a `Result` without `throws`, awaited in a scope, was an internal compiler error. Sema: each `scope`/`gather` has a marker that its `async` handles alias, so D100 part 3's local escape checks (`resources.go`) refuse returning, assigning outward, storing in an outer collection and outward-stored lambdas (a collection of the block's own may hold them); `Launch` gains `Fails` (the callee throws) and `Unwrap` (a fail-fast launch: handle `Task<R>`); a `Task<Result<…>>` written where the handle goes, and `try await t`, get D141 messages with a fix. Runtime: a failing child marks its scope failed and cancels its siblings before it publishes `T_DONE`; `veles_task_set_unwrap`/`veles_task_value` give `await` the Ok payload; an await (or race arm) on a failed unwrap child never yields it — the awaiter yields until abandoned or cancelled, and panics when the scope is already over; `veles_scope_abandon` cancels the body's call in progress and makes it join the scope. Migrated: `withTimeout`, `http.requestLine`, `std/net` test, `examples/httpclient`, `examples/streams`, `bench/spawn`, `bench/pipes`, `bench/httphello`, two driver test programs, the D45 conformance case. Tests: `sema/testdata/conform/D141-task-handles.vs`, `driver/taskhandle_test.go` (debug and release, 1–8 threads). Docs: chapter 12 "Handles belong to their scope", errors reference (`tasks`), cheat sheet, stdlib table. Limits in checklist §11.

**Progress 2026-10-08 (docs: concurrency, exactly; user: "people using this language have to know every little and big detail about what veles is doing in any circumstances").** New `docs/documentation/reference/concurrency.md`: the model, `async` (no start-order guarantee, arguments fixed at launch, the receiver copied), `await` (awaiting in order still runs everything concurrently; twice and from a sibling), the table of what is and is not a suspension point, scope ends, failures (first failure wins, body abandoned at its next wait, the body's own error, panics, `gather` cancels nothing), cancellation (never-started tasks never start; a task that never waits runs to its end), `withTimeout` on code that never waits, `race` losers keep running, channels (FIFO both ways, close, send-on-closed panic), whole-program deadlock detection, locks, scheduling (per-thread queues, runnext, stealing, yields, the blocking monitor), output, program end and exit codes, costs (each suspending call is a task today), and a table of the mistakes caught and not caught. Every claim was probed against the compiler first; the 10 programs run in `go test ./docs` (stable on 1, 2 and 8 threads). Chapter 12's "`race` … cancels the rest" corrected (a losing task arm is not cancelled). Candidates for compile-time refusal recorded in checklist §11.

**Decided 2026-10-08 (D142, visibility).** Unmarked is the package, `internal` the module, `public` other packages (exported at its module path, `public use` as a facade), `private` the type; construction across packages follows M5's private-field rule. Recorded in the spec (M5 and D89 struck through where superseded), checklist §10 and §11; Q22 (a facade may export unmarked items) and Q23 (a `public` signature naming an unexported type is an error) decided the same day into D142. To build as B17; nothing built yet.

**Progress 2026-10-09 (concurrency review; F1 B1 fixed; D143–D147 decided).** The user asked for a review of all concurrency for "very little memory and cpu" and "precisely manage the native OS threads": `archive/veles-concurrency-review.md` (measured on 32 cores; Go baseline alongside). **Bug fixed (F1):** parking on a channel was O(n²) — `push_waiter` walked to the tail and `remove_waiter` scanned, under a `spin_lock` that never yielded — so 100k parked receivers kept all 32 cores at 100 % (62.8 s of CPU in 2 s); now doubly linked waiter lists with a tail (O(1), a node knows the list it is on) and a spinlock that yields its time slice after 128 spins: 47 ms of CPU in 2 s. Pinned by `driver/TestManyWaitersOnOneChannel` (50k receivers + 50k racing ones that time out out of the middle of the list, then 50k sends; 1/8 threads): times out (20 s) at HEAD, 1.7 s after. `bench/spawn` unchanged. **Decided** (user, the recommended option each): D143 executors and threads, D144 atomics, D145 loop back-edges cancel and suspending loops yield, D146 `RwLock`/`Event`/`Lazy`/`Broadcast`/`Watch`, D147 a `close()` may suspend; track F orders the building. Not yet verified on WSL at the time of this entry (see below if added). Next: F2.

**Progress 2026-10-09 (F2, the task object; a race crash).** `veles_task` 448 → 248 bytes (the GC's 256-byte class instead of 512): the panic record (message, location, trace) is a side object made when a task panics, the debug call chain and its parent link one made by a debug build's first call, the flags `int32_t` (the ones changed by compare-and-swap) or bytes. Peak memory per parked task 657 → 400 bytes (1M tasks: 626 → 381 MB). **Bug found and fixed:** a `race` held a fixed array of 16 arms and nothing limited the count — a 20-arm race segfaulted; a race is now allocated for its arms (`veles_race_new(task, n)`), which also makes the two-arm race of every `withTimeout` 1 704 → ~250 bytes (`TestRaceWithManyArms`, debug and release). Benchmarks against HEAD, three rounds each: `spawn` 54 → 47 ms, `pipes` 15–19 → 7–12 ms, `httphello` 1.10–1.24 → 0.99–1.11 s, `channels`/`parallel` within noise. **Found on Linux (not caused here):** `TestDocs/concurrency/6` fails at HEAD on WSL: the example threw 5 ms after `async child()` and expected the child's `with` to have opened; on Linux a just-launched child takes up to ~7 ms to start at 8 threads (mean 20–60 µs; Windows worst 0.3 ms). The example now waits for the child to say its resource is open (the doc explains why); the start latency is recorded under F9. Windows suite green; WSL suite green, and the channel and race tests clean under `-sanitize` there.
**Progress 2026-10-09 (D148, indexed list increments).** Added `MutableList<i64>.incrementAt(i)` as a `const fun` that increments in place, returns unit, counts negative indexes from the end and panics at the caller on invalid indexes. Migrated counting sort and the eight exact `+1` frequency updates in DEFLATE/INFLATE; counting sort keeps its domain-specific error. Tests cover normal and negative indexes, invalid indexes, and compile-time/runtime parity. Windows verification: `go test ./std`, the targeted const-fun driver, algorithms example, collections docs, and Veles tests for `std/prelude` and `std/compress`.
**Verification 2026-10-09 (D148).** `go build ./...`, `go vet ./...`, and the uncached serial Windows suite (`go test -count=1 -p 1 ./...`) passed; `internal/wsl-test.sh` passed on WSL. The first parallel Windows run hit an HTTP idle-connection flake and a sema prefix-test timeout under concurrent Windows/WSL load; the serial rerun passed.
**Progress 2026-10-09 (D149, `MutableList.updateAt`).** Added the generic in-place transform `updateAt(i, transform): T`: synchronous/non-throwing callback called once, stores and returns its replacement, same bounds behavior as `at`/`set`, and a documented warning not to mutate the receiver from the callback. Kept `incrementAt` for the counter case. Regression tests cover a side-effect counter proving one call, `i64` and `string` elements, negative/out-of-range indexes, and const-vs-runtime behavior. Updated collection/reference/const-function docs.
**Verification 2026-10-09 (D149).** Windows `go build ./...`, `go vet ./...`, and uncached serial `go test -count=1 -p 1 ./...` passed. WSL targeted const-fun, `std/prelude`, and `std/compress` tests passed. The full WSL script did not finish: the driver process hit its 10-minute timeout during `TestStdHttpUnitTests` in the HTTP limits group; this change's targeted WSL tests are green, but the full WSL suite remains unverified.

**Progress 2026-10-09 (F3: a suspending call runs in its caller's task).** Before, every call of a suspending function was a task of its own — a task object, an argument block, a frame, a result cell and a `setjmp` — about 70 ns even when nothing waited. Now a call passes the callee a _link_ in the caller's frame (the caller's handle, the callee's depth, a done flag, the result): a callee that finishes without waiting writes its result and returns, and the caller goes on; one that waits parks the task in its own frame (`veles_frame_park`) and, when it ends, hands the task back to its caller (`popped`; the executor's `resume` runs the caller next). A task's own frame has the root link and still finishes the task. Frames of calls come from a per-task bump arena (dropped when the task parks in its own frame, so an idle task carries none); the common paths of allocating, freeing and returning are inline IR (`veles.frame.*`, field offsets asserted in `veles_task.c`). Cancellation unwinds frame by frame, innermost first; a failed child abandoning a scope body inside calls marks the task's `abandon_depth`, and every deeper frame sees a cancellation, leaves its waits and unwinds back to the body (so the body's own `with` now closes after the calls' — before, it closed first). A panic unwinds the whole task. The task lost `awaiting` (the old callee-task chain) and kept 248 bytes (`result` shares storage with `entry_args`). **Measured** against HEAD, three rounds: `bench/suscall` (new: 10M three-deep calls that never wait) 2.45–2.61 s → 128–148 ms (~70 → ~4.4 ns a call); `httphello` 1.11–1.30 → 0.93–0.98 s; `spawn` 55–73 → 41–51 ms; `channels`, `pipes`, `parallel`, `json` unchanged or better. **Bug found and fixed on the way:** a method-table slot declared `suspends` over a method that does not suspend forwarded no arguments to its ramp (it took them from the ramp, which lists none) and worked only because they stayed in their registers; the extra link argument broke it (`std/fs`'s stream test ran out of memory). Pinned by `TestSuspendingCallsInOneTask` (abandonment three calls deep, timeout three calls deep, values, big values and errors across suspensions, 20 000-deep recursion, a panic two calls deep; debug and release, 1/2/8 threads, GC pressure), golden IR updated; docs reference/concurrency "What it costs", chapter 13; spec D35 addendum. Windows and WSL suites green, examples at 1/2/8 threads on WSL, sanitizer clean on the concurrency tests. Next: F4 (D145).

**Progress 2026-10-09 (F4: D145 — loops are cancellation points, suspending loops yield).** A loop's back edge now polls `veles_attention` (the collector raises it as it raised `veles_stop_requested`; so do a cancellation or a scope abandonment of a _running_ task, and the monitor's request to yield), alone on a 64-byte line. The slow path asks the runtime: a suspending function takes its ordinary cancellation path, yields through `sleep(0)`, or runs the scope body's fail-fast check; a plain function unwinds the task from where it is, as a panic does. **New: a task unwound without its frame (a panic, or that cancellation) ends only once the children of the scopes it left have finished** — it waits as `T_ENDING`; a panic used to end at once and leave them running (D3). Shielding: a per-thread count raised by a held `Mutex` and around each `with` close (`runClose`). The monitor (5 ms looks while tasks run) asks a task running ~10 ms with work waiting to yield, and at its first look moves that worker's `runnext` (a child the running task launched) to the shared queue — found while testing: a child launched by a body that then looped never started until the loop ended (`runnext` is now changed by exchange everywhere). `yieldNow()`, `checkCancelled()` in the prelude. **Measured**: `withTimeout(100 ms)` around a 4-billion-step plain loop returns in ~100 ms at ≥ 2 threads (it ran to the end before); with one thread a suspending loop lets a 50 ms ticker run on time (was after the loop, 1.3 s); benchmarks against HEAD equal or better (`sort`, `json`, `strings`, `channels`; `spawn` 56 → 36 ms, `httphello` 1.02–1.06 → 0.90 s). **A regression caught and fixed:** `sha256` was 8–10 % slower until the polled word got a cache line of its own (59–66 ms against HEAD's 66–79 after). Pinned by `TestLoopsAreCancellationPoints` (a plain loop under `withTimeout`, a suspending loop's fairness on one thread, a lock region and a `close()` shielded, `checkCancelled`, the wait for a scope's children after a cancellation and after a panic, a body loop abandoned by a failed child — debug and release, 2/4/8 threads under GC pressure, and one thread); docs reference/concurrency (the interruption rule, both examples rewritten to show it, the mistakes table), chapters 12, 14, 16, concurrency-explained, stdlib; spec D145 "Built". **Not done** (checklist): under `VELES_THREADS=1` a plain computation keeps the only thread. Next: F5 (D144 atomics).

**Progress 2026-10-09 (F5: D144 — atomics).** `Atomic<T>` gains `compareAndSet` and `compareExchange` (with `failure:`), and on the ten integer types `add`/`sub` (new value, wrapping) and `fetchAnd`/`fetchOr`/`fetchXor` (old value), one `extend Atomic<T>` block per type. Every operation takes `order: MemoryOrder` (a new prelude enum: `Relaxed`, `Acquire`, `Release`, `AcqRel`, `SeqCst`, the default). An order written as a case and invalid for the operation is a compile error at the call (`sema/atomic.go` `checkMemoryOrders`): a load that releases, a store that acquires, a failure order that stores or is stronger than `order:`. An order known only at run time goes to the instruction as a value: a switch with one block per LLVM ordering, folded when inlined (release `hits.add(1)` = one `lock incq`), and an invalid one acts as `SeqCst`. An `Atomic` of a pointer, a nullable pointer or an enum is lock-free now. `add` on a non-integer `Atomic` says where it lives. std's counters use `add`/`sub` instead of `update(n => n ± k)`. Tests: `sema TestAtomicMemoryOrders`, `driver TestAtomicOperations` (a Treiber stack on `Atomic<(*Node)?>`, relaxed adds, bit flags, a release/acquire hand-off, wrapping; debug/release, 1/2/8 threads, GC pressure). Docs: chapter 12 (the operation table), chapter 13 "Atomics and memory orders" (the stack, the order table), reference/concurrency, stdlib, cheatsheet. Measured: 8 tasks × 2 M increments on 8 threads, `add` 130–153 ms, `update` 266–376 ms.

**Progress 2026-10-09 (F6: D146 — RwLock, Event, Lazy, Broadcast, Watch).** All prelude. `RwLock<T>`: a two-word runtime lock (`veles_rw_read/write_lock/unlock` in `veles_sync.c`: writer bit, parked bit, readers from bit 2, plus a waiting-writers count that keeps new readers out), parked on the Mutex stripes; the thread keeps a list of the RwLocks it holds (TLS `rw_held`), so a re-lock panics; `read()`/`write()` go through the D107 machinery (`isMutexLock`, `HeldLock.Method`), with `withRead`/`withWrite` and `get`/`set`. `Event`, `Watch`, `Broadcast`/`Subscription` (`std/prelude/notify.vs`) keep state behind a Mutex and wake waiters by closing a signal channel. Their waits are _awaited_ methods: `sema/awaited.go` requires `await` on `wait`/`changed`/`recv` (D16) and lowers a race arm `x.m()` to a channel arm on `x.mSignal()` whose win runs `x.mTake()`. `Lazy<T>` builds under its lock, publishes with a release store and reads with an acquire load; a panic in `init` is recorded by a close (`veles_panic_current`) and re-raised by every later `get`. All Sendable when `T` is (`syncWrapper`). Tests: sema `TestSyncTypes` (awaits, the race receiver rule, read() outside `with`, a wait in a write region, Sendable); driver `TestSyncTypes` (debug/release, 1/2/8 threads, GC pressure; the three re-lock panics). Docs: chapter 12 section and example, reference/concurrency, stdlib, cheatsheet, errors. Settled details in the D146 "Built" addendum (Watch's last-seen per copy, the race receiver rule, send after close panics, get/set/isReady added for parity). Found on the way: `sema TestCheckEveryPrefix` (the "load flake" of F4) failed reliably now — the hint for an unknown name re-parsed every standard module not yet imported, once per unknown name (a 128-line prefix of `examples/algorithms` has 67): parsed modules are kept per package now (`Package.peeked`), `veles check` of that prefix 5.3 → 2.4 s, the test 184 → 157 s. And `examples/features/cancel` (F4's other "load flake") failed on an idle Linux run: it gave its children 20 ms to start, and a child there can take longer (F9); it now waits on a channel the children signal when they hold their resources and when the gather's child has closed — output unchanged, 30/30 on Linux.

**Progress 2026-10-10 (F7: D147 — a close() that suspends).** sema: `fun close() suspends` is allowed in `implement Closeable` and required when the close suspends (`suspend.go`); a Closeable object of such a type is refused (`boxValue`, supers included); lock regions and non-suspending lambdas refuse it through the existing checks. Codegen/runtime: `runClose` shields the task (`veles_task_shield_enter/leave`, a `shield` byte in `veles_task`, still 248 bytes; `cancel_task` neither wakes a shielded parked task nor does a failed child abandon its body); a panic's close thunk hands a suspending close to a _closer_ task (`veles_cleanup_continue`): the remaining cleanups are copied to the heap through each record's new `move` function — the frames and the stack they point into are being left (the first try kept the records in place and crashed on one living on the C stack) — and the closer runs them after the close (`veles_task_finish`); the unwinding task waits for it as for a draining scope. std: `Tx.close()` sends ROLLBACK within the transaction's timeout and keeps the connection; `Telemetry.close()` flushes; TLS: `io.Stream : Closeable` and TLS conns are `io.Stream` objects, so the close stays plain and sends close_notify as Go's `Close` does, with one non-blocking write (`net.Conn.tryWrite`, new) — the user: "do it like go/rust". Tests: sema `TestSuspendingClose`; driver `TestSuspendingClose` (normal end, a panic across two suspending closes, cancellation parked and in a plain loop, a cancellation during the close, generics; debug/release, 1/2/8 threads, GC pressure); `TestTLSClient` (a Go server reads the alert from the socket; fails without it); `std/db` against PostgreSQL 18 (the abandoned transaction's connection is the next statement's; fails with the old close); `std/otel` (the end of the `with` exports). Docs: chapter 13 "A close that waits", 16, 23, 24, reference concurrency/stdlib/errors; spec D147 "Built" addendum with the TLS deviation.

**Progress 2026-10-10 (F8: D143 — executors and threads).** Decided while building (user; D143 addendum): an executor is made by static functions that throw, `try Executor.pool(threads:, name:, priority:, cpus:)` and `try Executor.thread(name:, …)` (D73: an `init` cannot throw); `Thread.start` takes its function as `f:`; closing a `Thread` blocks (Rust's `join`). The user rewrote `Executor.run`'s body as `return await async f()`. Runtime (`veles_task.c`): the default pool's globals became a `veles_exec` record — shared queue, run queues, idle/waking/spinning counts — and every pool, `Executor.thread` and the blocking pool has one; a task runs on its scope's executor (`veles_scope.exec`: `on:`, or the owner's), so `place` queues every wake there; `veles_task` unchanged (248 bytes). Other executors' threads only run tasks: timers and sockets stay with the default pool, which a pool thread wakes when it arms an earlier timer or parks on a socket; no hand-off around blocking calls outside the default pool; the monitor looks at every executor's runs (the list under the runtime lock); the deadlock check counts every executor's queues and running threads. Threads set their name, priority and CPUs as they start (`veles_thread_configure` in `veles_sync.c`: Windows priorities and affinity masks, Linux nice/SCHED_FIFO and `sched_setaffinity`, macOS a name) and report before the next starts, so a refusal is a `ThreadError` with nothing left running; a thread of a closed executor detaches from the collector (`veles_thread_detach`) and frees its block. The blocking pool grows a thread at a time up to 128, threads end after 10 s idle. `Thread`: the function runs as a task outside every executor (a panic unwinds and is raised again at the join; task-locals follow). Language: `scope(on: e)`, `gather(on: e)` in both front ends (Go and `selfhost`), the formatter and the AST printer; sema checks the type and lowers to the handle; `[runtime] threads` in the manifest reaches the runtime before anything runs (`veles_runtime_threads`). Prelude `std/prelude/executor.vs`: `Executor` (Sendable), `Priority`, `ThreadError`, `blocking`, `Thread`. **Measured** (Windows, 8-thread default pool, an HTTP server and 16 clients): p50 230–265 µs idle, 450–466 µs with 6 spinning loops on the default pool, 279–289 µs with them on their own pool. Tests: driver `TestExecutors` (debug/release, 1/2/8 threads, GC pressure — at most a pool's threads used, one OS thread across 1 000 suspensions, `run`/`blocking`/`gather(on:)`, a timer and a socket on a pool's task, a Thread's join and panic, refused CPUs, a closed executor), `TestRuntimeThreadsManifest`; sema `TestExecutorPlacement`, `TestManifestRuntime`; format `TestScopeOnRoundTrips`; codegen goldens re-accepted (two declarations). Docs: chapter 12 "Choosing the threads: executors" (tested), chapter 11 `[runtime]`, concurrency-explained, reference concurrency/stdlib/cheatsheet/errors. Not done (checklist §11): CPUs past 63 on Windows, macOS settings, a blocking `run` for a Thread's plain function, the latency tail with a busy pool (Linux: median kept, p99 1.6–2.5 ms). Verified on WSL (full suite green; priority High/Realtime refused without CAP_SYS_NICE, with the reason). Open for the user: Q29 (non-Sendable state on an `Executor.thread`). Next: F9.

**Progress 2026-10-10 (F9: the runtime lock, timers, the wake-up path).** Measured first with a start-latency probe and lock statistics (per caller, how often the runtime lock was taken and found taken). **Found:** the root task's ramp ran inline in `veles_task_start`, before `veles_run` had started a worker or the monitor, so a child launched before `main` first waited never started while `main` computed (18 s in the probe); now the root is queued and `veles_run` runs it. A child in a computing parent's `runnext` waited for the monitor (5–10 ms; about 7 ms on Linux): `place` now wakes an idle worker for it, and a stealer takes a running worker's `runnext` when its next look (a spin round later) finds the same task and the same run — Go's runnext steal, without its sleep. `sleep` and race deadlines are nanoseconds end to end (sema passes `Duration.ns`; `veles_task_sleep`, `veles_race_sleep`, `veles_ticker_start` take ns); idle waits use them at the system's resolution: condition variables on the monotonic clock with ns timeouts, `epoll_pwait2` (ENOSYS → `epoll_wait`), and on Windows the 1 ms timer period (`timeBeginPeriod` loaded from winmm). The lock statistics put 99 % of runtime-lock entries at 32 threads in `race_wait` (384k, 94 % contended), `scope_cancel` (192k) and socket waits (64k): the timer heap moved under a spinlock of its own (allocation and wakes outside it), `cancel_task` now takes apart only a _parked_ task's waits (claimed S_IDLE → S_QUEUED first) and a running one leaves its own when it sees the request (`veles_task_cancelled`), so `race_wait` and `sleep` take no runtime lock; `scope_cancel` with no child left takes none. **Measured** (Windows): child start p99 5–10 µs, worst 16–106 µs (Linux worst 136 µs at 32 threads; target ≤ 0.5 ms); `sleep(200 µs)` 14–16 ms → 0.2–2 ms (Linux about 0.3 ms); `httphello` 8 threads 0.42 → 0.33 s, 32 threads 0.95 → 0.57 s — still slower than at 8, but Go's reference goes 175 → 665 ms on this machine (Linux 183 → 580 ms; Veles 0.45 → 0.63 s), so the acceptance "no slower at 32" is not met by either and is recorded in §11 with what is left (socket waits on the runtime lock, one condition variable per executor). Benchmarks otherwise equal or better (`spawn` 32 ms; Linux `spawn` 3.9× → 2.2× Go, `parallel` 1.2× → 0.7×). Test: driver `TestChildStartAndFineTimers` (a child starting before main's first wait and while its parent computes, sub-ms sleep and race timeouts never early and not a timer slice late; debug/release, 2/8 threads; fails at HEAD). A reference example that cancelled a just-launched task expecting it never to start became timing-dependent and now shows the running case only; the rule's text says why. **Found while verifying (there at HEAD too):** `TestTLSClient`'s 1 MB echo stalled for the Go server's 2-minute deadline about once in six runs under load — a deadlock in `std/tls`: a read that found records queued (often the writer's, not yet taken) waited for `sendLock`, held by the writer stalled on a full socket, while the echoing peer waited for this side to read. A reader now never waits for the lock: it leaves the records to the holder (`sendWanted`), which sends them before and just after it lets go; 8/8 under the same load. Docs chapter 20, reference/concurrency, stdlib. Track F is done; the review's remaining items are in §11.

**Progress 2026-10-10 (F10: memory, the collector and the last locks).** The user: "the concurrency topic must be done perfectly". Measured before changing anything — CPU time by category in the default pool's loop (rdtsc), lock statistics per call site, the collector's phases, allocation per call site, frame sizes and arena use at each park: at 32 threads `bench/httphello` burned 7–10 s of user CPU in a 0.65 s run (Go 1.6–3.2 s), and the time was inside the tasks, not in spinning. **The collector was half the run:** ~100 stop-the-world collections in 0.4 s, marking and sweeping on one thread while the others waited. **Allocation:** two thirds of it (612 MB of ~900 MB) were 2 KB frame-arena chunks — a task dropped its chunk at every park in its own frame and at its end, and took a new one at its next call; a frame chain shrinking back dropped the chunks it had grown into. Chunks now go back to a cache in the thread block (16), and are sized to the collector's 2048-byte class (2072 bytes had taken a 4096 slot). **Coroutine frames** were 0.8–5.6 KB in `std/http` (`readRequest` 5.6 KB: 169 allocas, 56 of them its own result type, one per return and throw): without lifetime markers the coroutine split counts every alloca live from the function's entry, so each temporary written after a suspension point took a frame slot of its own. A temporary built and taken apart in place (the codegen's `scratch`: tagged values, union fields and conversions, if/when/race results outside the memory class, the result handed to `veles_frame_return`) now gets `llvm.lifetime.start` where it is made, and stays on the stack: those frames are under 256 bytes now, and a task parked ten calls deep in a read holds ~4 KB. A first attempt marked every alloca and broke two tests: a race's send arm and an integer's text buffer are kept by a call past a suspension the IR does not show — the split then put them on a discarded stack; the comment on `scratch` says so. **The collector:** size classes at most a quarter apart from 64 bytes on (36, found in one look for sizes up to 1 KB); marking shared with helper threads (one per core up to eight, once 1 MB stays live; `VELES_GC_MARKERS`) that sleep until a marker shares a batch of its stack — half of it, oldest first, from eight entries on (helpers spinning for work slowed the collecting thread down; sharing only past 512 entries never happened on a tree's narrow stack); sweeping in chunks of spans claimed by the markers; a heap-bounds check before the page table. **Locks:** socket waits in 64 shards of their own (the runtime lock had been found taken in 78 % of them at 32 threads), the timers in 64 heaps chosen by the task's address (the one timer heap's spinlock 73 %), the default pool's housekeeping between batches (timers, a quick look at the sockets) off the runtime lock, tickers under a spinlock of their own, and the Windows reactor's socket table in 32 shards. The monitor now registers with the collector (it moves tasks between queues) and fires timers and looks at sockets that no default thread came back for — a pool task's `sleep` waited for every default thread's plain loop to end. **Found and fixed:** a suspending call cancelled at a loop's back edge before it ever waited returned to its caller with `done` set, and the caller went on and read the result the call never wrote (`TestLoopsAreCancellationPoints` case 9 printed the caller's line at HEAD): `veles_frame_unwound` marks the link 2, and the caller then unwinds or abandons its scope as after a suspension point. **Measured:** `httphello` on Windows 310 → 160–180 ms at 8 threads, 650 → 180–230 ms at 32 (Go 150 / 400 ms; CPU at 32 threads 7–10 s → 2–2.5 s); Linux 0.45 → 0.22–0.26 s at 8, 0.63 → 0.26–0.33 s at 32 (Go 0.19–0.22 / 0.55–0.60 s, with twice the CPU); collections 102 → 65 at 8 threads, mark 58 → 17 ms and sweep 34 → 8 ms in all; `bench/idle` (new, with a memory table in the runner): 345 bytes per parked task, Go 8.7 KB per goroutine, the review's 250 not reached (§11 says what it needs). Other benchmarks equal or better. Tests: driver `TestParallelCollector` (1/2/8 markers, 2/8 threads, debug/release, poisoned sweeps), `TestExecutors` case 11 (a pool's timer and socket while every default thread runs a plain loop; fails without the monitor's part), `TestLoopsAreCancellationPoints` case 9. Docs chapters 12 and 13, concurrency-explained, reference/concurrency, cheatsheet, README. Left in §11: `withTimeout`'s task per call, the task's size, the stop-the-world pause, idle threads on one condition variable per executor, Windows' 1 ms timers.

**Progress 2026-10-10 (F11: parking).** Idle threads waited on one condition variable per executor under the runtime lock: every wake took the lock, the system chose the waiter, and on Windows every timeout was whole milliseconds. Now each thread parks on its own (`veles_park` in veles_sync.c: a futex on Linux; an auto-reset event and a high-resolution waitable timer on Windows 10 1803+; a mutex and condition variable elsewhere), an executor keeps a list of its idle threads under a spinlock, and a wake takes one off and wakes it by name; a thread lists itself before its last look at the queues, so a waker that queued first and a thread that listed first always see each other. One idle thread of the default pool keeps time until the nearest timer (the one in the reactor, if any, does instead); a timer armed sooner wakes it. The reactor's wait is claimed by compare-and-swap, so no idle path takes the runtime lock; it is left for the queues handed off around blocking calls and the deadlock check. Windows CPU lists may name any CPUs of one processor group (`SetThreadGroupAffinity`; across groups a `ThreadError`, since CPU sets are only a preference). **Found:** a task waiting on a channel that a `Thread`'s body fills was reported as a deadlock (a running Thread did not count as active; `TestExecutors` case 12 panics at HEAD); and the first version's `wake_all` popped idle threads until the list was empty, which never ended while the woken ones, finding nothing, listed themselves again — on Linux at 32 threads `veles test --jobs 1`'s main thread looped there with the next test in its own runnext (`TestTestRunner`; found with an in-memory event ring, since tracing to stderr hid it); it takes the list whole now, and idle nodes are recycled, never freed, when a thread ends. **Measured:** `sleep(200 µs)` on Windows median ~0.55 ms (was 0.2–2 ms), Linux unchanged; `httphello` 8 threads 176 → 168 ms, 32 threads 205 → 206 ms (six runs each, alternated); other benchmarks unchanged. What stays in §11: on Windows the thread waiting in the reactor keeps time to the millisecond.

**Progress 2026-10-11 (F12–F17: the task diet, and D150–D154).** The decisions F10/F11 raised were taken as one batch (D150–D154; Q29 in a second round, where the user turned `scope(local)` down for isolation without annotations: "the annotation attributes for threading gives me an ick, but 'higher level' feels nice"). **F12:** a parked task 345 → 249 bytes (`bench/idle`, the review's 250): the task is 152 bytes, the 160-byte class with the collector's header — its channel and socket waits in one union (a plain channel node finds its task and channel from where it sits), `entry`/`race`/`result` and `entry_args`/`cleanups` sharing words by phase (a spawned task not started is marked `W_UNSTARTED`; a closer's inherited cleanups wait in the rare part), panic, debug chain, timer and test mark behind one pointer, fourteen flags in eight bits; timers are nodes of their own in the heaps, so a deadline can have one. **F13 (D154):** `await t` after `t.cancel()` on every path is refused; on the way, a smart cast a loop body set outlived the loop (`x: i64?` taken as `i64` after a loop over an empty list — it compiled and printed 1). **F14 (D153):** a pool's blocked thread hands its queue to a spare named after it; on the way, `Executor.pool` passed a `bool` to a C `int64_t`, whose upper bits Windows leaves undefined — every pool had one thread there. **F15 (D152):** `runBlocking` and `Executor.default`. **F16 (D151):** `withTimeout` runs `f` in the caller's task under a deadline kept in its own frame (116 ns a call, was ~450; no task, no scope, no race); a fired deadline cuts the frames deeper than `withTimeout`'s as a cancellation does, and a plain loop runs `f`'s cleanups and jumps back — into the call of `f` while it is on the stack, else through the executor — after the scopes it cancelled have drained; a 1 ms limit around a plain loop under `VELES_THREADS=1` fired after 16 s before, 4 ms now. **F17 (D150):** `executor gl { state; functions }` — sema scopes the state to gl's code, a call from outside is `hop(gl, () => gl.f($args))` with the arguments evaluated in the caller, children in gl's code skip the Sendable check, state is made on gl's thread by `veles_block_on` before `main`; formatter, hover, `veles doc`, and the self-hosted parser (its parity corpus has the new conformance cases). Tests: driver `TestPoolBlockingCallHandsOff`, `TestWithTimeoutInCallersTask`, `TestExecutorDeclarations`, `TestExecutors` 13–14; sema `TestAwaitAfterCancel`, `TestLoopFactsDoNotLeak`, `TestRunBlockingOnlyInPlainCode`, `TestExecutorDeclarationRules`, `TestExecutorAcrossModules`; format, lsp. Docs chapters 12 and 16, reference/concurrency, stdlib, cheatsheet. Left in §11: the root frame outside the task, `withTimeout`'s closure allocation, `runBlocking`'s extra hop, declared executors not closed at exit, `async gl.f(call())` refused.

**Progress 2026-10-11 (after F12–F17: four of its §11 leftovers).** `async gl.f(call())` launches: an executor hop's arguments, in a launch as in a call, are evaluated first into hidden locals of the launching task (the Launch is wrapped with them; a held launch, D111, keeps them before it), so a call among them runs in the caller and a `var` crosses as its value at the launch. A function of an executor takes variadic parameters: the hop types each trailing argument as an element (`gl.total("a", 1, 2)` with `bytes: u8...`). `runBlocking` places its task straight on the executor (`veles_task_place_on`), not on the default pool to hop from there. `withTimeout` 116 → 76 ns a call: `runWithin` calls `f` itself and its Result lands in the link in `withTimeout`'s frame (null when the limit came first; an `f` throwing nothing gets its `Ok` made there), so the `var` cell, the closure around it and `attempt` are gone. On the way: `async cpu.run(f)` was refused — a generic method's instance gets its receiver when its body is checked, after the launch, which then took the receiver for a pointer argument; the call's own receiver kind now says it is a method. Tests: driver `TestExecutorDeclarations` (arguments made in the caller, variadics), sema `TestExecutorDeclarationRules`, `TestExecutorPlacement`, conformance D150. Still in §11: the root frame outside the task, an executor's function as a value, declared executors not closed at exit.

**Progress 2026-10-11 (checklist §11, the concurrency entries).** `@caller_location` (D88) takes suspending functions — the site is one more argument of the call in the task and of a launch; a plain marked function started with `async` had left it out of its ramp (an IR mismatch, unreached until now) — and `retry`, `mapConcurrent`, `forEachConcurrent` are marked (driver `TestCallerLocationInSuspendingCalls`). The D111 error for `withTimeout(d, () => serving())` now points at the call, its body keeping no variable since `runWithin` returns `f`'s Result. Looked at and left, with the reason in §11: a woken receiver's retry under the channel's lock (a lock-free retry measured no change on `bench/idle` at 32 threads, and was taken out); the monitor's pass over the pools under the runtime lock (microseconds every 5 ms, not the tail's milliseconds); the root frame inside the task (at most 8 bytes). `TestStdHttpUnitTests` passed 60/60 on Linux under load — the loop: `go build -o /tmp/vt .`, `go test ./sema -count=1` in a loop in the background, and `timeout 300 /tmp/vt test std/http` 60 times keeping the output of each failure. Raised for a decision: a `gather` child that returns a `Result` without `throws` has its `Err` flattened into the gather's error, where `scope` (D141) and a D111 field keep it as the value.

**Progress 2026-10-11 (D155, Q31).** Decided with the user: Q30 → D155, a `gather` child that returns a `Result` without `throws` gives it as its value (`Result<Result<T, E>, Panic>`; it was flattened into the gather's error, unlike `scope` and D111 fields); Q31 → an executor's function as a value outside its code stays an error (D150 addendum). Built D155 in sema (the gather element type splits a child's Result only when the launch `Fails`) and codegen (`gatherResults` wraps any other value in `Ok`). Found on the way: a function not declared `throws` with a failing child in a `scope`, or receiving a value whose task field is `Task<Result<T, E>>`, crashed codegen (`throwValue` with no Result to throw into) — now errors with the fix, as `try` there is. Tests: driver `TestTaskHandles` (gather and a held field, debug/release, 1–8 threads), conformance D141 and D111. Docs: chapter 12, reference/concurrency, reference/errors.

**Progress 2026-10-11 (the hang hunt).** Two unexplained hangs (a 48 s `TestLoopsAreCancellationPoints`, a 10-minute driver timeout) were chased to two scheduler bugs, both fixed. (1) A cancellation, a fired `withTimeout` deadline or a failed child told a running task's loops to look only when its state read `S_RUNNING`; a task woken while it ran (`S_WOKEN`) was missed, and a plain loop in it ran to its end — 30 s against a 100 ms limit in the reproducer (`runs_now`; driver `TestWokenTaskStillLooks`). (2) A request reaching a queued task can be put off past its next suspension point (a completed send returns first: `chan_done`; a suspending close finishes first), and the check runs only when a suspension point resumes — so the next wait parked the task unseen and its scope waited forever: std/http's accept loop, cancelled at a server's stop just after a connection gave its place back. A frame that parks with a request pending now queues its task again (`veles_frame_park`; cleanup joins park with `veles_frame_park_held`); driver `TestPutOffCancelEndsTheNextWait` (deterministic on one thread). Found with a new tool: `VELES_WATCHDOG=<s>` makes a program still running after that long print every live task (state, wait, scope, a debug build's calls) and every queue, then exit 124 — the live tasks are a set kept only then (open addressing over a rooted collected array; the collector's new timed stop gives a consistent snapshot). The soak that caught (2): the driver test binary 15 times in a loop beside the sema tests on Windows, and 15 times on Linux plus `go test ./examples -sanitize`, with `VELES_WATCHDOG=45` — after the fixes 15/15 and 15/15 clean, sanitized examples clean. Benchmarks unchanged (suscall 120 ms, spawn 35 ms, idle 106 ms / 249 B). Docs: reference/concurrency, cheatsheet, the veles-debug skill.

**Progress 2026-10-13 (F18: the collector out of the pause).** Measured first with a probe that became `bench/gcpause` (134 MB live in 4M nodes, small lists made and dropped): every pause 19 ms — mark 13, sweep 5.7. (a) Lazy sweeping: a collection lists its spans, which are swept when an allocator claims them, two at a time by allocations on the slow path, and by the helpers in the background; live bytes come from the marking; `veles_gc_collect()` still sweeps fully. Sweep in the pause 5.7 → 0.1 ms. (b) Spans from 64 MiB arenas (Windows: reserved with `MEM_WRITE_WATCH`), found through a two-level table a marker can read while it grows; a dead large object's memory goes back at once. (c) Mostly concurrent marking on Windows: the coordinator thread finishes the last sweep, resets write watch, stops the world only to take the roots, marks with the helpers while the program runs, and stops it again to rescan the pages written meanwhile, the globals and the stacks. What made it fast was measurement: born-marked allocation and resetting every page cost a third of the program's speed (a reset page faults at its next write, and an allocator writes every page it fills) — new objects are unmarked, only spans with no allocator since the last collection are reset, the rounds reset only pages in spans holding something marked, and a rescan skips spans with nothing marked; and a claim walked every full span of its class while marking ran (new spans now go to the tail, the cursor past full ones). Result: pauses under 1 ms (the loop's longest stall ~1 ms; Go's 4–35 ms in the same loop, its assists), 753–765 ms against 798 before (Go 627–660). (d) Pacing: a concurrent collection is triggered early by what the last one allocated while it marked; a thread past half again the goal waits. Linux keeps marking in the pause by default: soft-dirty bits reset the whole process (offered with `VELES_GC_CONCURRENT=1` after a self-test, untested here — WSL2 has none). Tests: driver `TestConcurrentMarkingKeepsWhatIsLive` (fails on a build that skips the rescan), the examples and the driver suite under `VELES_GC_CONCURRENT=1`/`all` with poisoning and the sanitizer, soaks of the driver tests with the watchdog on — 8/8 Windows (=1), 8/8 Linux (=all). Bugs met on the way: LeakSanitizer does not scan our mappings (arenas registered as its root regions); the coordinator scanning from its own stack pointer to the main thread's stack base. Left in §11: Linux (noscan spans + mprotect on quiet spans), a structure built while marking runs, the first collection, assists instead of waits.

**Progress 2026-10-13 (D156 decided).** The user asked whether `scope` is needed to run one task and await it ("maybe we could just require to await the async handle by the compiler?"). Probed first: a `val h = async f()` outside a scope is refused today, `scope` is refused as a value, `xs.iter().mapConcurrent` does not exist, and `scope(on: cpu) { r = decode(4) }` compiles and runs `decode` on the caller's thread. Decided as one batch (all recommended; `awaitAll` over `all` by the user's choice): must-await `val` handles in any block with a warning for an immediate await, `awaitAll`, `scope`/`gather(workers:)` combinable with `on:`, a lazy ordered `iter.mapConcurrent` with loops owning a task-holding iterable, `scope` as an expression, and an error for a placed scope that starts nothing. Spec D156 (v0.81), checklist §10 and §11, notes P11; built as F19.

**Progress 2026-10-13 (F19 built: D156).** Tests first: conformance files D156-awaited-handles, -await-all, -workers, -scope-value, -lazy-map; driver `TestLessCeremony` (an abandoned `val` task cancelled and joined before the error leaves, a failing child failing the block, `awaitAll` cancelling its inline siblings, peaks under `workers:` with and without `on:` and in `gather`, the lazy form's order and run-ahead, a `break` joining every call it started, a scope's value; debug and release, 1–8 threads); a new example `examples/verify` (a `sha256sum -c`: manifest and listing read with `awaitAll`, files hashed by the lazy `mapConcurrent`) and a D156 section in `examples/features/tasks`. Built: (a) `scope` as an expression (`ast.ScopeExpr`; a scope ending a block that has a value is that value) and `workers:` in the parser, formatter and self-hosted parser; (b) `val h = async f()` lowered as a with-task whose handle is checked awaited on every normal exit by a flow fact (`sema/mustawait.go`), every other use refused; the immediate-await warning; (c) `awaitAll` (`sema/awaitall.go`; the list form is the prelude's `awaitEach`); (d) `workers:` in the runtime — `veles_scope_slot` parks the owner at the limit and a leaving child wakes it (`slot_wait`, Dekker-ordered with `live`); (e) D42's `AsyncIterator`, `ConcurrentMap` with a runner task, and loops receiving a task-holding value (`loopHeld`). Found on the way and fixed: `when` dropped every flow fact its arms established (now joined, as `if` and `race` do), and a `when` value whose arms all leave was not `Never`; `Task<Result<R, Never>>` did not match the task of a non-throwing instance (Subst now makes it `Task<R>`); D111 refused `tasks.at(0)` on a scope's own list of handles. Measured against HEAD (spawn, suscall, channels, idle): within run-to-run noise. Linux: the D156 tests, the two examples ×5 on 1/2/4/8 threads and under the sanitizer, clean. Not done: migrating std/examples away from scopes that only launch and await (none trips the new warning), a bench of `awaitAll` against the scope form, and the rest listed in checklist §11.

**Review 2026-10-14 (direction: documents, self-hosting, idioms, platform APIs; `archive/veles-direction-2026-10-14.md`).** The user asked whether the spec, plan and checklist are current, what self-hosting needs next, what idiomatic Veles looks like and where it could improve, and for low-level platform APIs (Bluetooth etc.) with a high-level developer experience. _Documents:_ status drift fixed — Build order step 11, B14, track C items 2–8, track E items 1/2/4/5/7, track D's batch text; the spec's twelve examples and four prose spans still in the removed `fun f() = expr` form migrated to `=>`, §4b's "`implement`, not `implement`" restored to `impl`; checklist ticks for OTel, the `substring` check, `std/fs`, the GC line (F18), §8b's intro, a stale §11 entry; README rewritten to v0.81 (registry, reactor, C ABI, 25 chapters, 63 examples); the self-host plan has a status block; the cheat sheet's module-level `var` (an error since D66) and `i8 i16 i64 i64` fixed; a stale `mut` comment in `examples/httpd`. _Found:_ B17 (D142) is the real next build step; interpolating a `Result` compiles and loses the error (Q32); no per-target Veles code, no debug info, no UDP/Unix sockets/file watching/terminal/device modules. _Added:_ track S rows S6–S12, B18, track P (P1–P10), checklist §9 Q32–Q43 and a §11 entry. Every "today" program in the review was compiled and run (scratchpad `gallery`, `gallery2`, `doc1`, `doc2`). Not done: nothing was built; the full `go test ./...` was not run — only documentation and one example comment changed: `go build ./...` and `go vet ./...` green, `go test ./docs/` and the `httpd` example green after the edits.

**Decided 2026-10-14 (D157–D167, the review's batch).** Asked in four rounds the same day; every recommended option except where noted. D157 a `Result`/task cannot be interpolated (Q32); D158 bare names reach the receiver's members (Q33); D159 `std/bluetooth`, LE central, high level, simulated backend (Q42); D160 per-target code — file suffixes plus `build.target` from an imported std `build` module (Q39: the user turned down a free `target`, offered `package.target.os`, then chose "'build' is better"; `package.vs` stays held back); D161 `std/sys/<os>` in Veles with portable modules on top, all four module groups (Q40, Q41); D162 `veles bindgen` (Q43); D163 one-parameter constructors positional (Q34); D164 `copy` (Q36); D165 the index-of family answers `i64?` (Q37); D166 labelled tuples (Q35, after the full write-up of every use case the user asked for); D167 `std/cli` by derive (Q38). Spec v0.82; checklist §9 empty, §10 rows, §11 "decided, not built"; plan B19–B22, C9, P1/P7/P10 acceptance, the continue-order updated. Nothing built yet.

**Decided 2026-10-15, built 2026-10-16 (B23: D168, racing computations).** From the idioms review (`veles-idioms-and-apis.md`): its `fastest()` — a `scope`, two handles, `await` in each arm — waited 300 ms for the losing mirror, because a race does not cancel a handle that loses; the correct version cancelled each loser by hand. The user: "this piece of code looks awful and we should change it in the language". Asked in two rounds (Q44). Built: `async f()` race arms owned by the race (a cancelling scope of the race's own; the race records the winner, the losers are cancelled and joined, then the winning body runs — `sema/check_task.go` `raceGroup`); bare arms (the value is the race's; refused on sleep/send/wait/changed); `src as name =>` (parser `raceHead`, Go and `selfhost/`); no `await` in arms (a handle arm is checked as the `await` D156 counts); `awaitFirst(async a(), async b())` as that race (`sema/awaitfirst.go`); the removed `val x = src =>`, `val r = await h =>`, `val _ = src =>` keep parsing and error with a fix (`veles check --fix` migrated `examples/features/tasks`). Migrated: std `http` (`requestLine` lost its scope and handle), prelude `iter`, `time` docs, examples, chapters 12 and 20, the concurrency and stdlib references, the cheat sheet, errors reference, conformance files; new `D168-racing.vs`, driver `TestRaceTaskArms` (loser cleanup before the body, first error wins, a handle arm not cancelled, 200 concurrent pairs; 1/2/4/8 threads). Measured: `fastest()` 10 ms instead of 300. Found on the way: a loser cancelled before it first ran opened no `with`, so "every loser cleaned" holds for the losers that started (the test counts those). Not built (checklist §11): `async` arms placed by an enclosing `scope(on:)`, completion for `awaitFirst`, a first-success race, `awaitFirst` over a list.

**Decided and built 2026-10-16 (B24: D169, `race` as a block, `select`, `async { }`).** The user, after B23: "naming 'as' in race arms was a mistake… 'race' without arms, just calls like gather that races and losers are canceled and another `select` that has arms always", plus `async { }` blocks (an `async () => { }` lambda form was considered and dropped by the user: "would create it but not call it"). Four questions, each as recommended: `awaitFirst` removed, a select arm names its value `val x = src =>`, `async` arms in `select` owned and cancelled, `return` in `async { }` gives the task's value (Q45). Built: parser (Go and `selfhost/`, G-gates over new corpus lines) — `select {` contextual, `race { body }` (a top-level `=>` in the body still reads as the removed arm form, for the error), `async { }` as an async call of a no-parameter lambda flagged `Block` (so all 25 checker sites that know `async f()` apply: placement, D156 awaits, D35 captures, `with`); checker: `race` contestants pushed onto a hidden `MutableList<Task<T>>` as they launch (`sema/race.go`), one type, none → error or (from a loop) a run-time panic, one → warning; lowering: a cancelling scope whose body ends in `RaceFirst` over the list — new runtime `veles_race_list` registers every handle with the existing race machinery, so no change to the scheduler — then `await` of the winner; `select` is the D168 arm checker renamed; the D168 spellings (`race` with arms, `as`, bare arms, `awaitFirst`) error with fixes. Migrated std (`http`: `requestLine` is a `select` with an `async` arm; prelude `iter`; `time` docs), examples, chapters 12 (new sections `race`, `select`, `async { }`, each tested) and 20, the references, cheat sheet, errors reference, the editor grammar (`select` contextual), conformance (new `D169-race-select-async.vs`, D168's file retired), Go tests. Found on the way: the test of a failing race counted on the loser having started; it now gives the loser time to start (a task cancelled before it ran opened no `with`).

**2026-10-17 — labels on tests and suites (D170), B17 (D142 visibility) and B20 (D158 bare members) built.** The user, asking for small DX steps: `clamp` already existed (`v.clamp(0, 255)` on every integer and float type — only unknown to them); labels, package-level visibility and implicit `this` were not built. **D170:** `test listMethods "sentence" { }` / `suite Lexer "sentence" { }` — an optional identifier before an optional sentence; reports, `--filter` and the editor outline use `label: sentence`; a label is unique within its suite (parser, `ast.Title`, formatter, LSP symbols, `sema/testing.go`, the self-hosted parser with a P5 corpus line; `TestTestLabels`, `TestTestSuites`). **B17:** `Pub`/`Internal`/`Exported`/`Facade` on symbols, fields, types; one rule `declVisible` (`sema/visibility.go`): own module, else `internal` stops, else the same package (std is one package), else `public`/exported; a module of another package is reachable by its directory path (`resolveDep`), a `public use m { x }` facade copy exports even an unmarked item (under the facade's name only), `public use m` of a whole module shows its unmarked items (`Symbol.Facade`); a field a reader cannot see behaves like `private` for the implicit constructor; hover/completion/shape follow the levels (`internal` is spelled when written); the warning `'public' … does nothing in a program` (fix removes it) and the Q23 error for `public` functions/fields naming a type other packages cannot see (`sema/lint_public.go`; std needed only `Locked`/`RwLocked` made `public`); tests `visibility_test.go`, `package_test.go`, `reexport_test.go`, conformance `D142-visibility.vs`. **Migration as built:** `public` was removed from every program (examples, selfhost, conformance files) with `veles check --fix`; libraries (std, `examples/features/packages`) keep `public`; the unwritten level widened from the module to the package, so nothing written before stopped compiling and **no `internal` was inserted** — the spec's "unmarked → `internal`" step was not done (it would put a keyword on every private helper, which is the painting D142 exists to remove); an author who wants a module-only helper writes `internal`. **B20:** before a method body is checked, `resolveBareMembers` (`sema/implicit_this.go`) rewrites every bare name that is a field/method of the receiver — and that the body binds nowhere as a local — to `this.name`, and every bare static member of the type to `Type.name`, so narrowing, moves, resources and the index all see the one spelling; a member that hides a module-level name warns (`'x' means this.x here`), a bare instance member in a `static fun` errors with `this function is static`; trait defaults, `extend` bodies, `init` and lambdas follow; one std conflict (`Router.handler`/`http.handler`) got a helper; conformance `D158-bare-members.vs`, `D28` updated. Found on the way: `veles check <file> --fix` loads and fixes the sibling `.vs` files of its directory (do not run it on a path inside a populated test-data directory). Not verified: WSL/Linux run. Left in checklist §11.

**2026-10-17 (later) — D171 and D172 decided, not built.** After the above, the user chose format specs after `:` with a higher-level `std/fmt` (locales noted as a todo) and `"""` text blocks plus backtick raw strings. Build order when picked up: lexer (`"""` margin, backticks, the `:spec` inside `${}`) → parser/AST → formatter → sema spec check → `selfhost/` lexer/parser with corpus lines → `std/fmt` → docs. Q47/Q48 are closed in checklist §9/§10.

**2026-10-17 (later still) — D172 and D171 built.** _D172:_ the lexer reads `"""` text blocks (text starts on the line after the opening delimiter; the closing delimiter's indentation is the margin; escapes and `${}` work; a less-indented line is an error; CRLF folds to LF; a tag may precede it) and backtick raw strings (no escapes, no interpolation, over lines); a failed literal still yields a string token so the parser does not cascade. _D171:_ the lexer splits `${expr:spec}` at the first top-level colon (outside brackets, strings and character literals, not `?:`) into `StringPart.Spec`; `ast.StringPart` carries it (and `ast.Dump` prints `:"spec"`); sema parses the spec (`sema/formatspec.go`), checks it against the value's type and lowers it to the prelude helpers `fmtNum`/`fmtText` (`std/prelude/format.vs`, not public) through a synthesized call on the already-checked value (`ast.CheckedExpr`), so nothing is parsed at run time; widths count characters. `selfhost/` lexer, parser, AST and printer mirror both, with gate corpus lines (`text-blocks`, `raw-strings`, `format-specs`) byte-identical to the Go front end. `std/fmt` (first set: `thousands`, `bytes`, `percent`, `ordinal`, `duration`, `table`; `std/fmt/fmt.test.vs` with labelled tests). Docs: chapter 2 (text blocks, raw strings, format specs), stdlib reference, errors reference, cheat sheet; conformance `D171-format-specs.vs`; VS Code grammar for `"""` and backticks. Limits in checklist §11.

**2026-10-17 (later still) — pipelines: D173 decided (extension functions), not built.** The user asked how to serve people who like pipelining (`|>` / `->` floated). Findings: method chains already are the pipeline (`.` continues a line, one `try` covers the chain, D134); the gap is free functions and foreign types, because D23 declined extension functions (the trait + `implement` route works and chains, but is global under D17 and takes two declarations). The user chose extension functions over a pipe or UFCS, visible by importing the module, with their own spelling `extend T with fun f() …` beside `extend T { }`; recorded as D173, §10 Q49, B25; `|>` not added (re-open only if real code shows a gap). Nothing built; open details in checklist §11.

**2026-10-17 (later still) — B25 built: D173 extension functions.** _Parser (Go and `selfhost/`, G-gates with a new `extensions` corpus line):_ `extend T with fun f() …` (`ImplDecl.With`), `public extend` / `internal extend` (`ImplDecl.Pub/Internal`; the old "'extend' cannot be public" error is gone), the error `expected 'fun' after 'extend … with'`; dump (`(extend T public with`), formatter keep the one-line form. _Sema (`sema/extension.go`, `declareExtend`, `check_call.go`):_ `extend` of a type the package does not own (`ownsType` false) declares an extension — `Impl.Foreign`, kept in `Checker.foreignExts` apart from `extends`, so no existing lookup (hover shape of built-ins, loop-change checks, suggestions) sees it. Resolution is the last step of `dispatchMethod`, after inherent methods and trait methods: in scope when the file `use`s the module (a braced `use m { x }` counts; the module's own files always), `callMethod` enforces `public`/`internal`/unmarked per D142; more than one candidate (two imports, or a trait method) is `ambiguous method … drop one of the imports or rename one of them (D173)`; a bound that does not hold gives the `extend<T: B>` message of own blocks. Declaration rules: no `static fun`, no `private`, no `Result`/`Option`, an inherent method of the same name (struct body, built-in, own-package extend) gives a warning "never called" (post-pass in `checkCoherence`). A call from a file that does not import the module says `'f' is an extension function in module 'm': add 'use m'` (per-file: a sibling file's import does not help). Hover: `extend T — extension from module 'm'`, and a `public extend` block's visibility shows on the function. _Tests:_ `sema/extension_test.go` (11 cases: scope per file, braced import, shadow warning, two-module ambiguity, trait ambiguity, bounds, public-API-only through `Duration.ns`, not a trait bound, statics/private/Result), `TestExtendBlocks` cases flipped, conformance `D173-extension-functions.vs`, `TestExtensionRoundTrips`, parser rejections, `TestHoverExtensionFunction`, example `examples/features/extensions` (module `strx` + a chain). Docs: chapter 5 (new section), chapter 8 table, cheat sheet, errors reference (declarations, members). Limits in checklist §11.

**2026-10-18 — D173 extensions shown on the import (user: "the imported module's hover (or maybe even inline hint) should inform us of the extensions").** `Checker.indexExtensions` (`sema/extension.go`) adds, to every reference to a module, the extension functions the referring file may call (`declVisible` from that file's module): the module hover gets a `// extension functions` block of `public extend T with fun …` lines, and `Ref.Extends` lists the types. `lsp/inlay.go` `extensionHints` ends a `use` line with `extends string, List<T>`. Test `TestExtensionsShownOnTheImport` (an `internal` extension of another module stays out); docs chapter 5.

**2026-10-18 — documentation and examples review (user: Veles is not public, so no old syntax is kept in the docs).** Mechanical sweeps over `docs/documentation`, `examples`, `bench`, `selfhost`, `std`: `veles check` on every program (warnings fixed: `shapes` named its last variant, `tasks` unused loop variable, the table generator in `selfhost/tables_test.go` no longer writes `public`), `veles fmt --check` (three files formatted), a link and anchor check over the Markdown (clean), and a grep for provisional or historical wording. Docs: chapter 1 (`async` claim), chapter 2 (formatting numbers points to format specs; the `padStart` example uses `${id:06}`), chapter 5 (`enum` exists), chapter 8 (`${cents % 100:02}`), chapter 11 (the `use m.{…}` and `exports` remarks), chapter 13/stdlib reference ("bootstrap" wording, a "Not yet" list that named things that exist), chapter 14 (the `@test` remark), chapter 17 (task-locals exist), errors reference (`self`, `as`), `index.md`, `README.md` (spec version, B17 built, example count). Examples: `padEnd`/`padStart` with a literal width became format specs in `config`, `crypto`, `recursion`, `time`, `todo`, `utf8`; `syntax_tour.vs` no longer casts with `as`.

**2026-10-18 — B26, D174 user-defined annotations built.** Parser (Go and `selfhost/`, G-gates green): `annotation Name(params) on targets [for Traits]` as a `StructDecl` with `Annotation` info (a declaration ending in a keyword such as `on fun` ends at the line), `@mod.Name(args)`, `inline loop`, and `Self` as a receiver before a dot; formatter, dump, TextMate grammar. Sema: `sema/annotation.go` (declaration checks, use resolution after the types exist, target/duplicate/unknown/not-an-annotation errors, the unread warning, hover and go-to-definition on `@Name`) and `sema/reflect.go` (prelude `FieldInfo`, `T.fields`, `inline loop` with field handles, `f.Type`, `f.get`, `f.has`, `f.attr`, pruning of `if (f.has<A>())` and `if (val a = f.attr<A>())`, `T.build`, the evaluator builtins, and `checkReflectUse` for calls from code that runs). Tests: `sema/testdata/conform/D174-annotations.vs` and `D174-reflection.vs`, `parser/annotation_test.go`, `driver/annotations_test.go` (a two-package build, debug and release, and the misuse error), `examples/features/annotations`. Docs: chapter 14 (new section), cheat sheet, errors reference, stdlib reference (`FieldInfo`). Found on the way: `println("${…}")` of a `List` is how an example prints one; the formatter moves a trailing comment from an attribute line to the field line, so conformance files that expect a diagnostic on an attribute are not formatted. Not done, in checklist §11.

**2026-10-18 — D175: the implementing type is `This`; D174 reading helpers.** The user asked why `Self` and not `This`; the type was never named on its own (D65 renamed the receiver only), so it was decided: `This` is the keyword (type position, and a receiver before a dot), `Self` parses as the same type with an error whose fix writes `This` (`parser.oldSelfTypeSpelling`; test `TestOldSelfTypeSpelling`). Migrated in the same change: lexer, Go and `selfhost/` parsers and printers, formatter, completion, error family, the built-in catalogue, `std/` (13 uses), `selfhost/`, `examples/`, docs, the conformance files and the Go tests that embed programs, README; the spec, plan and checklist keep their history. The user also asked for reading helpers (done: `T.annotated<A>()`, a constant `List<(FieldInfo, A)>`, and `inline loop ((f, a) in T.annotated<A>())`; the validation example in chapter 14 lost its `has`/`attr` questions) and for typed decorators with a simpler spelling than `on fun wraps retryCall` (open, checklist §9 Q52).

**2026-10-18 — D176 typed decorators built.** `annotation Name(…) on fun = wrapper` parses (Go and `selfhost/`; dump, formatter) into `AnnotationInfo.Wrapper`. `sema/annotation.go`: `annotationWrapper` resolves the wrapper once (a function in scope where the annotation is declared; two or three parameters; the first is the annotation), `applyDecorators` rewrites the body of every function that carries decorator annotations when the package is checked — the body becomes a lambda, the call to the wrapper (an `ast.SymbolRef`, resolved already, so the user's imports do not matter) wraps it in `try` when the wrapper throws, the first annotation written is outermost — and the annotation's arguments are built on each call, not constants. Tests: `sema/testdata/conform/D176-decorators.vs` (the declaration's mistakes and the uses), `examples/features/decorators` (retry, logging with arguments, stacking, a method; debug and release). Docs: chapter 14 (Decorators), cheat sheet, errors reference. `@Retry(3)` is still `@Retry(times: 3)` when the annotation has a defaulted second parameter (D163's rule, not built yet). Not done, in checklist §11.

- 2026-10-18 — B27 parts 1–2 (D177): `sema/lints.go` registry (17 lints named, `Diagnostic.Lint`, levels `allow|warn|deny` in `[lint]`, `max_width`, `long_line` off by default, `veles lint --list`); `[format] trailing_commas` (`multiline`|`never`, one-tuple comma kept) and `line_endings` (`lf`|`crlf`|`auto`). Tests: `sema/lints_test.go`, `format` (`TestTrailingCommas`, `TestLineEndings`), `driver` (`TestFormatManifestCommasAndEndings`). Docs: chapters 11 and 25, cheat sheet, errors reference. Part 3 and what else is left: checklist §11.

- 2026-10-19 — B27 part 3 (D177), run model: evaluator (user's choice after the 2026-10-18 probe: 294k nodes ~2.8 us/node/pass, sealed variants not constants). Built: `CVariant` (sealed variants and `*T` inside a `const fun`), `std/lint` + generated AST from `selfhost/ast`, tree reflector and rule runner (`sema/lintrules.go`), `[lint] dep.rule`, bare dotted TOML keys, `examples/features/lintrules`, tests (`TestLibraryLints`, `TestRuleASTFields`, `TestLibraryLintFix`, `std/lint/lint.test.vs`, a sealed case in `TestConstFunMatchesRunTime`). Measured with real rules: 8,600 lines, three rules 0.22 s -> 0.29 s. Left (`[lint.forbid]`, type-aware rules, exit callbacks, evaluator tuning, ...): checklist §11.

- 2026-10-18 — B28 (D178): skipping tests. Sema: `skip(why): Never` and `skipIf(cond, why)` in the test vocabulary (`sema/testing_words.go`, `test.skip` builtin, a stop like `require`); std annotations `Skip`/`SkipIf` (`std/prelude/testing.vs`, targets `test`/`suite` added to D174, constant conditions, positional arguments for these two); a suite body accepts `skip`/`skipIf` statements (`ast.SkipDecl`, Go parser + `selfhost/` + formatter + `Dump`), compiled into one gate function per suite. Runner (`codegen/llvm/coro.go`, `veles_test_skip`/`veles_test_skipped`): gates run first, once each, as tasks of their own; annotation-skipped tests are never queued; a gated test is queued only when no gate skipped; `test x ... skipped: why`, summary `N passed, M failed, K skipped: names` (the count only when K > 0), exit 0 unless `--fail-on-skip`. Tests: `driver.TestTestSkipping`, conformance `D178-skipping-tests.vs`/`D178-skipping-places.vs`, `parser.TestSkipSyntax`, a `format` case. Docs: chapter 14 (Skipping tests), cheat sheet, errors reference. `build.target` (plan P1, D160) is not built, so the examples use a `const`.

- 2026-10-19 — S10 built (first cut): `veles check --dump names|types|hir` (`--dump-std` adds the standard library's symbols), `sema/dump.go`. `names` lists the package's functions, closures, externs, globals, structs, sealed types and tests; `types` their signatures with inferred effects (`suspends`, `conditional`, `self-escapes`, `writes-self`), parameter and local types, struct fields and sealed variants; `hir` writes every function's lowered body by reflection over the HIR, so a new node is dumped without being listed, each expression with its type, a call's callee, the operators, the captured variables. Deterministic: sorted by mangled name, map keys sorted, variables numbered per function (`a#0`), no spans, no pointers. Golden fixtures `sema/testdata/dump/{basics,sealed,effects}` (`go test ./sema -run TestDumps -update-dump`), determinism and `--dump-std` tests. What the first cut leaves is in checklist §11. Linux not run.
- 2026-10-19 — Gap review ("what do we need to be self-hosted and feature rich"): S14 (bootstrap fixpoint test), S15 (backend decision), G1 (the gap list) added to track S; D180 (wasm32-wasi, later) and D181 (`Vector128/256/512<T>`, `Simd<T, bits>`, lane aliases) decided. Operator overloading already exists (D71).
- 2026-10-19 — S14 harness built: `internal/bootstrap` (`Compiler.Build`, `Compiler.EmitIR`, `Fixpoint`, `FirstDifference`). Stage 0 (Go) builds the compiler source into stage 1, stage 1 builds it into stage 2, and each emits the IR (`--release --emit-llvm`) of that source; the compiler is self-hosted when stage 1's and stage 2's IR are identical (the fixpoint) and stage 0's and stage 1's are too. `TestFixpointMechanics` proves the plumbing with a compiler that is its own successor, `TestBuildProducesARunnableStage` the build step, `TestSelfHostedCompiler` is the gate and skips until `selfhost/compiler` exists (S11). Not verified on Linux.
- 2026-10-19 — D182 decided (`std/regex`: `/…/flags` literal, linear-time engine by default, opt-in backtracking, ASCII classes with a `u` flag, string extensions, typed groups later). Plan G1 carries the build; spec v0.96. Nothing built.
- 2026-10-19 — D182 built: `std/regex` and the `/…/flags` literal. `std/regex` (`parse.vs`, `compile.vs`, `vm.vs`, `regex.vs`, `regex.test.vs`, 20 tests): an RE2-syntax parser, an instruction compiler, a Pike VM and a bounded backtracker (a visited set of (instruction, position), results identical — a differential test over 24 patterns × 14 texts compares them), `Regex`, `Match`, five string extensions, `validate` and `literal`. The parser and the builder are `const fun`s that record the first error instead of throwing, so the compiler runs them on every literal (`static assert`s in the test file prove the evaluator can). Lexer (Go and `selfhost/`, G1–G4 green): a `Regex` token where a value may start, JavaScript's rule on the previous token, flags `imsxu`, errors for an unterminated literal and a bad or repeated flag. Parser/AST/dump (`(regex "pattern" "flags")`), formatter (verbatim), checker (`sema/regexlit.go`: needs `use regex`, runs `validate` through the evaluator, lowers to `regex.literal`), family `regular-expressions` (errors.md), conformance cases `D182-*`, `examples/regex` (a log analyser), docs chapter 2, cheat sheet, stdlib reference, VS Code grammar, `std/lint` regenerated. Measured against Go's `regexp`: 7× on a capturing line pattern, 3× on `\w+` (was 47× and 12×); what is not built or left is in checklist §11. Linux not run.
- 2026-10-19 — D182, second part: the rest of what the first part left out. `\p{…}` / `\pL` / `\P{…}` / `\p{^…}` for every Unicode 15 category and script, and the `u` flag (`\d \w \s \b` and the case folding of `i` in Unicode terms), on tables generated from Go's `unicode` package into `std/regex/unicode.vs` (`go test ./std -run RegexUnicodeTables -update-unicode`; the test fails when the file or the Go toolchain's Unicode moves). `Regex.compile(…, unicode:, backtracking:)`: with `backtracking: true`, backreferences (`\1`, `\k<name>`) and lookaround (`(?=` `(?!` `(?<=` `(?<!`) on an exhaustive depth-first engine (`runFrom`; loops that match nothing are cut by a progress check; a lookbehind tries each earlier start). The refusal in a literal or a plain compile names that call. The parser and builder stay `const fun`, so the compiler validates `\p`, `u` and `iu` literals (the folding runs in the evaluator: `static assert`s in the test file). 24 tests (properties, the `u` flag, backtracking, its errors). VS Code grammar: `const fun` is a function (the binding rule no longer takes `fun` for a name), `test label "…"` colours the label, and a regex literal is coloured inside (escapes, classes, groups, quantifiers, flags). Not done: typed groups, full case folding, a lazy DFA — checklist §11. Linux not run.
- 2026-10-19 — std/regex: the Linux (WSL) run of the whole suite is green (build, vet, every package). The backtracker follows a path through straight-line instructions without touching its stack and pushes only at a split: 398 → 234 ms on the 100 000-line capturing benchmark (Go's regexp 52 ms) and 519 → 351 ms on `\w+`; results unchanged (the engine-agreement test).
- 2026-10-19 — std/regex speed, second pass (profiled first: setup per call 0.27 µs, the visited set 0.35 µs, the rest is per instruction, ~10 ns each). Done: `MutableList.make`/`repeat` reserve their capacity (prelude, any program gains: 350 → 200 ns for 155 elements) and the engine uses `repeat` instead of a closure per slot; the backtracker marks its visited set only at instructions two paths can reach (`Program.heads`), tests ASCII sets by a shift and a mask (`Program.ascii`), and fuses a loop over one character (`.*`, `\w+`, `[A-Z]*?`; `Program.runs`) into a tight run with one stack entry that gives characters back (greedy) or takes one more (lazy) — same order of attempts. 100 000 log lines: capturing pattern 398 → 109 ms (Go's regexp 52 ms, now 2.1×), `\w+` 519 → 265 ms (Go 165 ms, 1.6×). The engine-agreement test grew (loops, multibyte text) and compares the backtracker with the threads on all of it. Windows only for this pass; the Linux run of the pass before it was green.
- 2026-10-19 — std/regex speed, third pass: the visited set is sized by the jump targets (`Program.headIndex`/`headCount`: about a quarter of the instructions) so more jobs fit the backtracker and each search allocates less, and the thread engine's lists are made only when a job is too big for the backtracker (a search that needs only the backtracker allocated 6 more lists). Empty-text setup 27 → 16 ms per 100 000 calls; the capturing benchmark 109 → 101 ms (Go's regexp 52 ms, 1.9×), `\w+` 265 → 255 ms (1.5×). Engine-agreement test and the whole regex suite green.
- 2026-10-19 — S11 written: `veles-selfhost-sema-plan.md` — the cut (`types/` + `sema/`, 46 900 lines of Go in 98 files, sized by component), the oracles (the dumps, 110 conformance cases, 178 sema tests, the goldens, the examples, the const-function differential, the S14 harness), phases R0–R9 each with a gate, the first slice (R0 + R1: TOML, manifest, loader, `--dump=modules`), what the port will stress, a cost estimate (45 000–55 000 lines of Veles), what must come first (S10b, S6, S7, S8, S12) and four decisions to ask: a readable HIR as a bridge to the Go back end (Q-S1), the shape of the checker's state (Q-S2), tree or arena HIR (Q-S3), error-recovery policy (Q-S4). New plan row S10b (per-phase dumps). Nothing built.
- 2026-10-19 — S10b built: `veles check --dump modules|scopes|decls|effects` and `--dump-spans` (`sema/dump_phases.go`). `modules`: the module graph after loading — each module's files, each import as written and the module it resolved to, the modules it needs. `scopes`: written when collecting ends, before any body is checked — every declaration of every module with its kind and visibility level, and the names each file imports (so it is there even for a program with a type error in a body). `decls`: the resolved declarations after checking — struct fields with their flags, sealed variants, enum members, trait members, signatures with the error a bare `throws` came to, implements and extends with their derived methods. `effects`: `rounds N`, then each function's inferred `throws` and `suspends`/`conditional`. `--dump-spans` puts `@file:l:c-l:c` after declarations and `Span:`/`Pos:` fields in the `hir` dump (paths relative to the package root). The checker calls the dump through `checkCollectObserved` (a nil observer is the old behaviour). Fixtures `sema/testdata/dump/{basics,sealed,effects,imports}` — `imports` has two modules and an effect fixpoint of three rounds — goldens for every dump and, with spans, for the dumps that carry them; determinism and std-filter tests; the scopes dump surviving a body error is pinned. Windows; Linux not run.
- 2026-10-19 — Q-S1..Q-S4 decided (sema plan §8): readable HIR bridge (to be cleaned up later, checklist §11), state as a family of types, flat HIR arena kept readable, diagnostics promise the set of errors in a stable order per file. R0+R1 (TOML, manifest, loader in Veles) started.
- 2026-10-19 — R1 step 1 built: `selfhost/sema/toml.vs` (the TOML reader of `veles.toml`, port of `sema/toml.go`, ~330 lines of Veles) and `selfhost/sema/toml_dump.vs`; `selfhost --toml FILE` prints the tree or the one error, `sema.TOMLDump` (Go) prints the same, and `TestR1Toml` (`selfhost/sema_test.go`) compares them on every `veles.toml` of the repository and 58 cases (every accepted form, every refusal, BOM, CRLF, line numbers, integer limits, `\u` escapes): 68 subtests equal. Windows; Linux not run. Next in R1: `manifest.vs` (the schema, 941 Go lines), then `loader.vs` and `--dump=modules`.
- 2026-10-19 — `std/toml` (D183, the user: "shouldn't the language itself have a toml parser that sema reuses?!"): the private `selfhost/sema/toml.vs` is gone; `std/toml/toml.vs` reads the whole of TOML 1.0 (array-of-tables, floats, dates, hex/octal/binary, multi-line strings, inline dotted keys) and `parse(text, strict: true)` reads exactly what `veles.toml` accepted, so `TestR1Toml` (68 subtests against the Go reader, now through `std/toml`) still passes. 12 std tests (`veles test std/toml`). Windows; Linux not run.
- 2026-10-19 — R1 step 2 built: the manifest reader in Veles. `selfhost/sema/manifest.vs` (the schema of every `veles.toml` table, ~560 lines, on top of `std/toml` strict), `semver.vs` (`parseVersion`), `suggest.vs` (`didYouMean`, the name-matching half of `sema/suggest.go`: foreign names, edit distance with transpositions, longest common subsequence), `manifest_dump.vs`; `selfhost --manifest FILE LINT...` prints every field or the error; `sema.ManifestDump`/`LintNames` (Go) print the same; `TestR1Manifest` compares them on every `veles.toml` of the repository and 117 cases (each table, each key, every refusal, lint-name typos, `Sscanf` quirks) — all equal. Two findings fixed on the Go side (Q-S4): a `[workspace]` member that is a root (`/x`, `\x`, `C:/x`) or climbs out (`..\x`) was judged by the host OS's `filepath` — now the same on every platform (`memberIsAbsolute`, `path.Clean`); and the `[policy] allow` check walked a map, so two bad claims gave a different message per run — now sorted. Windows only so far. Next in R1: `loader.vs` (module discovery, `use` resolution, per-target files, the embedded std tree) and `--dump=modules`.
- 2026-10-19 — R1 redone properly (the user: "very poor architecture… the selfhost should be at the highest possible level"). The first TOML/manifest port was a line-by-line copy; replaced with a designed model, same gates. **`std/toml`** is now six concerns in as many files: `value.vs` (a sealed `Data` family — `Text`, `Integer`, `Real`, `Flag`, `Moment`, `Items`, `Nested` — in a `Value` with its line; `Table` with its `Origin`), `cursor.vs` (`Cursor` lexical moves, `Rules.full()`/`Rules.manifest()` as named capabilities instead of a `strict` flag in the middle of code), `strings.vs`, `numbers.vs`, `moments.vs`, `tree.vs` (who may define what), `reader.vs` (the grammar); 12 tests. **`selfhost/pkg`** is the package system, a typed model: `Source` is a sealed family (`PathSource`, `RegistrySource`, `GitSource` with a `TagPin`/`CommitPin`) so a dependency naming two sources cannot exist; one reader function per table into an immutable value (`Identity`, `FormatOptions` with enums and `Indentation`, `LintSettings`, `NativeLibraries`, `Policy` with a `Capability` enum, `RegistrySettings`, `threads: i64?`); `Section` carries the file so every refusal names it. **`selfhost/text`** holds `quote` (moved out of `ast`, which `sema` had no business importing) and `didYouMean`. The harness dumps (`dump_toml.vs`, `dump_manifest.vs`) are test code in the root module. `TestR1Toml` (68), `TestR1Manifest` (~130), `TestP3Quote`, `TestP5Parse` all equal.
- 2026-10-19 — D184 (the user: "the alphanumeric and other checks are already in std no? shouldn't we use our language to write code?"): `std/prelude/ascii.vs` — `u8.isAsciiDigit()/isAsciiLetter()/isAsciiLower()/isAsciiUpper()/isAsciiAlphanumeric()/isAsciiHexDigit()/isAsciiSpace()/digitValue(radix)` as const extension methods; 3 tests. Seven private copies of `isDigit` and four of `hexValue` (prelude string, time, http, db url/row, toml, the selfhost lexer, `selfhost/pkg`) are gone; the const-function golden updated; a dead variable in the selfhost lexer removed. Left: `std/regex/parse.vs` works on code points (`i64`) and keeps its own until the shared Unicode module (review finding). Also decided with the user: the `veles.toml` tables move onto a settings registry (internal first, a public extension point for lint/format packages designed in the code review), and the code review runs as inventory + findings log first, then fix batches.
- 2026-10-19 — Settings registry (the user: "the lint or format could be a registry instead of `when (entry.key)`… because we allowed custom formatting and linting"): `selfhost/pkg/schema.vs` — `Setting<T>` (key, doc, read) with builders `text`, `choice`, `names`, and `Schema<T>` (settings in declared order, removed keys with their way out, a fallback for `[lint]`'s by-name lints); every table of `veles.toml` is a schema, and the file itself is a schema whose settings are the tables. The refusal of an unknown key lists the keys from the same declarations; `settings` carry a doc line for references, completion and hover. All 130 manifest cases still equal; 4 Veles tests pin the registry (docs present, no duplicate keys, the unknown-key message, removed keys, lint suggestions). The public extension point (a lint or format package declaring its own settings, checked and completed like the built-in ones) is left to the code review, with real code in hand (F6).
- 2026-10-19 — Code review planned: `veles-code-review-plan.md` — scope (every `.vs`, ~45 000 lines), the six checks (duplication and std gaps, architecture, idiom, performance, developer experience, tests), three waves (inventory and findings log → decision batches → fix batches), rules, and nine seed findings from R1 (F1 done; F3 two copies of the Unicode tables, F4 `quote` three times, F5 `std/lint/ast.vs` mirroring the compiler's AST, F9 eleven std modules without tests, … ). Nothing in wave 1 started.
- 2026-10-19 — Modules by ownership (the user: "shouldn't things like formatter or linter get its own module in selfhost so we can split registry from the implementation and have an easy way of adding new rules/settings?"): `selfhost/settings` (the registry machinery: `Setting`, `Schema`, `Section`, `SettingsError`), `selfhost/formatter` (`FormatOptions` and its schema — the formatter itself will land here), `selfhost/linter` (the lint registry — 20 lints with default levels and documentation, ported and pinned against the bootstrap compiler by `TestR1Lints` — and the `[lint]` schema), `selfhost/pkg` (identity, dependencies, workspace, native, policy, registry, runtime, and the manifest that routes each table to its owner). Adding a lint is one `LintInfo`; adding a format option is a field and a `Setting`; neither touches `pkg`. The harness no longer passes the lint names: the Veles side has the registry.
- 2026-10-19 — Sweep for hand-written ASCII ranges (the user found `(byte >= 'a' && byte <= 'z') || …` in `selfhost/pkg` after the helper migration: "I assume you didn't fix more things" — right: the first pass searched for the helper names, not for the pattern). About thirty sites in base64, hex, json, uuid, otel, db sql, http client/cookie/cors, config dotenv, prelude codec/string/duration, regex, the selfhost lexer, formatter and pkg now use `isAsciiLetter/Digit/Alphanumeric/Lower/Upper`, `digitValue` and the new `toAsciiLower/toAsciiUpper` (added to D184: three places each lower- or upper-cased by hand). The duplicate `isAsciiDigit` function in `prelude/duration.vs` is gone. `std/regex/parse.vs` (code points, `i64`) is the one place left. Lint system: `selfhost/linter/registry.vs` is now the one place that declares everything `[lint]` can set — the lints, and the options (`max_width`, `implicit_return`) next to them with `LintSettings`; `table.vs` only reads the table. Gates, 4 registry tests, std tests of every touched module green.
- 2026-10-19 — D185, the `extend` clause of a `use` (the user's questions during the review: opt-out of auto-included extensions → importing only some → their spelling `use m { A } extend { f }`, with "no extend => no extensions, bare extend => all, empty braces an error"). Built: ast, Go parser, dump, formatter, sema (`extensionInScope` by clause, `checkExtendClauses`, the missing-import hint names the clause), the selfhost ast/parser/printer (lint API regenerated), conformance cases, parser/format/sema/lsp tests, docs (chapters 2 and 5, errors, stdlib, cheatsheet), the examples. **Bug fixed with it**: std modules' extensions on built-in types (`std/regex`'s string methods) were inherent and global once any file loaded the module; only the prelude owns the built-in types now, so D173 rule 4 holds for std too. Findings logged for the review: F10 (`else if` chains of three or more branches in lexer, db conn, regex parse, http client, config, print), F11 (enum word tables: `Setting.choice` over an enum's members), F12 (`std/version`: a semver and constraints module decided as `std/version`, not yet built), F13 (the AST printer: one 1 000-line struct; split by family, try deriving the regular nodes by reflection).
- 2026-10-19 — D186, `std/version` (the user: "semver with constraints but called std/version"): `Version` (parse, precedence, bumps, `majorId`), `Constraint` (sets, `^ ~ > >= < <= =`, partials and `x` ranges, pre-release rule, `matches`, `newest`, `exactly`) in `std/version` (six files, 18 tests: the semver.org ordering, ~60 constraint cases). `selfhost/pkg` uses it; its own copy is gone. The bootstrap compiler's `ParseVersion` was tightened to the same pre-release rules (empty identifier, leading zero) — three new gate cases. Also: the editor grammar colours `extend` in a `use` like `use` (D185).
- 2026-10-20 — The inlay hint at the end of a `use` line ("extends string, List<T>") follows the `extend` clause (D185): none for a use without one, only the types of the named functions with `extend { a, b }`; `lsp/extend_hint_test.go` pins four forms.
- 2026-10-20 — D187 (an empty value owns no storage) built: mutable lists/maps/sets and `StringBuilder` allocate on the first write, immutable `[]`/`[:]`/`Set<T>()` are shared constants, `.toList()` of an empty list too, `a + b` with an empty side shares the other string unless a piece is a formatted number, a map of up to 8 entries has no hash index, an unbuffered channel no buffer; `bench/emptycoll` 205 → ~70 ms (Go 19 ms with escape analysis, ~105 ms heap-allocating: the gap is escape analysis, a session proposed). D188 decided, not built: D63's move proof grows with the escape analysis, then an explicit `freeze()` (Q56 open: a pointer taken before the freeze).
- 2026-10-20 — D188 built (the compile-time half): a per-parameter "does the function keep it" summary (`sema/escape.go`) widens D63's move to arguments handed to functions that only read or fill them and to fields of a struct built in the function; the `redundant_copy` lint (and ~55 `x.toList()` sites in `std/` fixed by `--fix`); and stack headers for mutable lists/maps/sets that never leave their function (`sema/escape_hir.go`, codegen `stackLit`; `VELES_NO_STACK_COLLECTIONS=1` turns it off): `bench/emptycoll` 70 → ~27 ms. `TestStackCollections` is a differential test (heap vs stack headers) under a tiny GC threshold. `freeze()` waits for Q56.
- 2026-10-20 — `freeze()` built (D188, Q56): a flags word (exposed, frozen) ends every list and map header; `xs.freeze()` freezes in place, copies the storage first if a pointer into it was ever made; writes through old names panic (`TestFreeze`, 17 panic cases in debug and release); write-guard cost within noise on sort/gzip/gunzip/intern. The move inlay hint (`moved, not copied`) is in the editor. Q57 (a marker for panicking functions) stays open: the user leans to none.
- 2026-10-20 — `isFrozen()` on MutableList/MutableMap/MutableSet (D188; the user's question, recommended option): the frozen bit as one inline load; not a mutating method for `protected var` fields; `TestFreeze` and `sema/testdata/conform/D188-freeze.vs` cover it.
- 2026-10-20 — Style lints from the syntax review (`veles-syntax-ideas.md`): `redundant_discard`, `loop_is_range`, `continue_skips_step`, `else_if_ladder` built with fixes and applied to the whole repository (about 100 + 23 + 59 sites, `TestRedundantDiscardLint`, `TestElseIfLadderLint`, `TestCounterLoopLints`); the `redundant_copy` lint now covers maps and sets, and D63's "built here" rule accepts constructors and static factories (`MutableSet<T>()`, `MutableList<bool>.repeat(…)`). Found on the way: the lint once rewrote the compiler's own derived `implement Decodable` into a `when` (spans of derived code are the user's); derived code is skipped now.
- 2026-10-21 — A7, macOS arm64: native kqueue with one-shot read/write filters, EVFILT_USER wake coalescing, close-on-exec and nanosecond waits; portable poll remains testable on macOS. `TestReactorKqueue` pins backend selection, filter replacement, one-shot readiness, EOF, forgetting and invalid descriptors; the connection stress now includes 4 workers, and the sanitizer smoke test runs on macOS. Fixed the C varargs fixture's Linux-only O_CREAT value (Darwin uses 0x200); snprintf/varargs and C structs already used the correct Apple ABI. Go 1.23.2 + Apple clang 21: build/vet/all tests green; all examples under poisoned GC at threshold 256 and ASan/UBSan, reactor/C ABI sanitizer tests, and GC/sync examples each 20 repetitions at 1/2/4/8 workers green. Sustained connection stress hit Darwin's ephemeral port limit, not a lost wake. All 22 benchmark checksums match Go, recorded in `bench/results.md`: httphello 386 ms default, 790 ms at 1 worker, 384 ms at 8; longest gcpause 15.96 ms (marking remains in the pause). Same-toolchain comparison: poll 369 ms before, kqueue 388–394 ms over three runs after, within the 10% noise threshold; no speedup claimed. `internal/macos-test.sh` pins the validation toolchain by default: Go 1.27's newer Unicode fails generated-table freshness, so no language data was regenerated. TLS uses tested Homebrew OpenSSL 3; D143's refused priority/affinity settings remain unchanged. Staticcheck was skipped by the user; Intel Macs and Windows/Linux reruns remain in checklist §11.
- 2026-10-23 — Test-suite wall time, point (3): codegen emits only what is reachable (`codegen/llvm/prune.go`). A last pass over the finished module keeps the definitions and private constants reachable from what is named from outside it (`veles_main`, `veles_init_globals`, `extern "C"` exports, non-private globals) through every `@name`: calls, function values and their thunks, closures, coroutine frames and method tables. Order and bodies are unchanged, so the codegen goldens did not move; the new `golden/reach` fixture pins it (an unused function and method gone; methods reached only through a trait object's table, a function passed by name and a lambda kept). Hello world 540 → 4 `define`s (1.3 MB → 28 KB IR), clang ~450 → ~245 ms; `examples/httpd` 2156 → 822 (9.3 → 4.0 MB), clang 2.9 → 1.4 s; `go test ./docs ./examples` 59 → 32 s; `driver` 586 → 325 s; benchmarks unchanged (`bench/results.md`). Linux (WSL) all green. Windows is green except three tests that also fail without this change in a fresh checkout: `examples/regex` and `examples/todo` sources check out with CRLF there (checklist §11).
- 2026-10-25 — Security review (user question: log4j-style holes, "prevent all known insecurities"). The classes were checked against the tree; two holes fixed. (1) Hash flooding: maps hashed strings with unkeyed FNV-1a and probed from `hash & mask` (an integer is its own hash), so 200 000 keys `i * 2^20` took 24.6 s to insert. The runtime now draws a key from the OS at start-up; strings use Go's keyed memhash fallback, the combine is keyed, and every hash is scrambled once at the map's entry points with the scrambled value kept in the metadata; constant maps are hashed at start-up by `veles_init_const_maps` (D113 addendum). The same insert takes 0.28 s. Driver `TestHashFlooding`. `bench/maps` 58 → 74 ms (Go 87): its keys spread almost perfectly under the old identity slot. A one-multiply scramble measured the same. `intern`/`json` within noise. (2) Text log lines escape C0/DEL/C1 controls in message, keys and values (forged lines, terminal escapes); std/log tests, docs chapter 21. Left open, in checklist §11 "Security review": D21 release overflow (being prepared as a decision), a user `hash()` not keyed, `html"…"`, an SSRF guard, rules for XML/archives/parsers, DEL/C1 in JSON.
- 2026-10-25 — D209 (Q58, from the security review; user, recommended of 3): integer overflow is checked in every build, D21's release wrapping removed. Prepared with a prototype switch in a HEAD worktree measuring the suite checked vs. wrapping (typically +5%; one +31% on `adapters` did not reproduce). Built: five branches in `codegen/llvm/expr.go` (`+ - *`, unary `-`, `abs`, the `MIN / -1` check, the release `sdiv` work-around gone); goldens `overflow` (release output = debug output) and `unchecked`; docs chapters 1–2, `Duration`/`time` comments. Full suite on the built change within noise (`adapters` 44 ms, `lexer` 51, `parse` 53, `json` 57).
- 2026-10-26 — Patterns menu (`veles-patterns.md`; user: "how we'll support 'singleton' pattern … and different other common patterns"). Decided: D210 `lazy val` globals, D211 resources made in `main`; open: Q59 serial executors, Q60 `context val`. Found by its probes: a module `val` of a `MutableMap` compiles and races (hang/segfault with 8 threads), checklist §11 F1. Next: plan B29.
- 2026-10-26 — B29 part 1 (F1): a module or `static val` must have a `Sendable` type (`sema/check_fn.go` `checkGlobal`; executor state exempt), with a fix hint per kind. The four-task `MutableMap` program that hung or segfaulted now fails to compile. Pinned by `TestConcurrencyRules`/`TestConcurrencyAccepted`; no std/example/selfhost/docs code hit it (one sema fixture did). Docs: chapters 12 and 14, error reference. Green on Windows except `examples/todo` and the `regex`/`todo` format corpus, which fail identically at HEAD; sema and docs green in WSL. Found: a module `var` of a synchronized type can be reassigned unsynchronized (checklist §11, needs a decision).
- 2026-10-26 — D66 addendum (v0.114; user, recommended of 4): no module-level `var`. A `var` of a synchronized type, accepted before, is now an error with a `val` fix and a hint naming `set`/`store`/`reset` (`sema/check_fn.go` `checkGlobal`, `syncWriteHint`, `globalWriteHint`, `keywordBefore`). The errors for assigning a global `val`, its field, or calling a mutating method on it (`sema/receivers.go`) no longer advise `var`; executor state still does. Evidence: `var total = Mutex(…)` reassigned while two tasks locked it printed 200 002 of 200 000 increments. Pinned by `TestConcurrencyRules` and `TestModuleVarOfLockFix`; docs chapter 12 and the error reference. Nothing in the tree used it.
- 2026-10-26 — The user's `to_figure_out.md` reasoned through: D212 `const struct` (plan B30), D213 opt-in allocation hints (plan B31), `suspend loop` not added with D145's yield to be measured (plan F21). Probed first: structs already print, compare, `copy` and are `const`, but are shallowly immutable (`t.names.push` through a `val`).
