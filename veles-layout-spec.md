# Value layout: research and proposed rules

Research and proposal, 2026-10-26. **Q61 and Q62 were decided the same day
as [D214](veles-spec.md) (definite assignment for local `var`s) and
[D215](veles-spec.md) (declaration order guaranteed, `compact struct`);**
the rest changes how values are represented, not what programs mean, and is
plan B32, built step by step. Until a step lands the spec (D5, D7, D12,
D120, D121, D198) governs.

## 1. What must not break

- **Meaning.** No program prints, compares, hashes, serializes or fails
  differently, in debug or release, on any platform. The only allowed
  observable change is speed and memory use.
- **C layouts.** `extern struct`, `extern union`, `@packed`, `@align`,
  `@transparent`, `Array<T: CLayout, N>` and the C ABI classifier
  (`codegen/llvm/cabi.go`) stay exactly as D120–D123 and
  `TestCLayoutAgreesWithC` / `TestCStructsByValue` pin them.
- **One layout truth.** Every producer of a value's bytes (codegen, GC
  descriptors, constants, debug info, runtime offsets) derives from
  `types.Layout`, so the checker, codegen and runtime cannot disagree (the
  rule D120 set; §4 lists where it is broken today).
- **Self-hosting.** Each rule must be expressible by a compiler written in
  Veles: plain tables and a recursive walk, nothing the Go front end alone
  can do.

## 2. How it works today (verified)

| Value                  | Representation                                                                                                                | Source                                                                                 |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| plain struct           | fields in declaration order, C alignment rules; `[n x i8]` inserted where `@align` needs more; `fieldMap` remaps LLVM indices | [types/layout.go](types/layout.go), [codegen/llvm/types.go](codegen/llvm/types.go)     |
| `string`               | `{ ptr, i64 }`; `""` literal points at a constant; `{ null, 0 }` also occurs (zeroed local, empty `takeText`)                 | [gen.go](codegen/llvm/gen.go), [veles_rt.c](runtime/c/veles_rt.c)                      |
| `T?`, `T` pointer-like | `ptr`, null = absent                                                                                                          | `isPtrLike`                                                                            |
| `T?`, otherwise        | `{ i1, T }`; all-zero = `null` (`NullConst` is `zeroinitializer`)                                                             | `makeNullable`, `nullFlag`, `nullVal` and ~a dozen direct sites in `expr.go`/`mem.go`  |
| sealed, error union    | `{ i32 tag, [N x i64] }`, payload at byte 8; recursive variants boxed (D198)                                                  | `tagOf`, `makeTaggedT`, `extractTaggedT`, plus `errorConvert`, `heldTaskAt`, `placeAs` |
| closure / trait object | `{ ptr code, ptr env }` / `{ ptr data, ptr vtable }`                                                                          | `llType`                                                                               |
| memory class (D121)    | aggregates **> 128 B that hold an `Array`**: address, `memmove`, caller-made copy as argument, `sret`                         | [mem.go](codegen/llvm/mem.go)                                                          |
| everything else        | first-class LLVM aggregate, passed and returned by value                                                                      |                                                                                        |

Generic instances are per type set (D8 as built), so a layout that depends on
the concrete type is safe in generic code. The runtime copies values as
opaque bytes of a compiler-given size (lists, maps, channels, task results);
it reads no nullable or tag, except that the fail-fast `await` assumes the
`Result` payload is at byte 8 (`veles_task_value`, runtime/c/veles_task.c).

## 3. Measurements

All on macOS arm64 (M-series), release builds, three runs each.

**Padding from field order** (scratch analyser over the IR of `selfhost/`,
437 structs): sorting fields by alignment would shrink 9 structs by 8 bytes
each — 16,601 → 16,529 bytes, **0.4 %**. Real code is mostly 8-byte fields
(`string`, pointers, the default `i64`). The adversarial
`{ u8, i64, u8, i32, bool }` halves (32 → 16), but nothing in the tree looks
like it.

**Sealed tag** (source scan of std, selfhost, examples, bench): of the
sealed traits found, **none** has only variants of ≤ 4-byte fields, so a
smaller tag or a payload at byte 4 would save nothing in this corpus.

**Nullable of a value type** in `selfhost`: 25 struct fields are
`{ i1, Struct }` (`ast.Ident?` 48 B, `ast.Block?` 40 B, `ast.Binding` holds
one), 8 are `i64?`, 2 `string?`, 2 sealed; 47 function parameters are
nullable values. Each struct case would lose its 8-byte flag word under §5.2.

**Big values passed by value** (`@noinline` callee reading two fields,
50 M calls):

| Argument                    | By value (today) | Address of the caller's value | Memory class today (copy + address)          |
| --------------------------- | ---------------- | ----------------------------- | -------------------------------------------- |
| 40 B (`string` + 3 × `i64`) | 46–48 ms         | 46–50 ms                      | —                                            |
| 64 B                        | 38–64 ms         | 34–38 ms                      | —                                            |
| 96 B                        | 45–47 ms         | 35 ms                         | —                                            |
| 144 B                       | 68–94 ms         | 35–36 ms                      | 95–99 ms (a 144 B struct holding an `Array`) |

The win comes from **not copying**, not from passing an address: the
existing memory class, which copies and then passes the address, is slower
than plain by-value. `selfhost` has 280 functions taking a struct by value;
`ast.FunDecl` is 280 B and `parser.Parser` 152 B.

**Locals without an initializer** (`var x: T` with no value, inside a
function) in std, selfhost, examples, bench and docs: **0**.

## 4. Bugs found while researching

Each reproduced in a scratch program (outside the repo).

1. **An uninitialized local breaks non-null types.** `var x: T` with no
   value is accepted and zero-filled ([sema/check_fn.go](sema/check_fn.go)
   `checkValStmt`; codegen zeroes it), with no definite-assignment check.
   - `var p: *Named` then `fun wrap(p: *Named): (*Named)? => p` →
     `wrap(p) == null` is **true**: a non-nullable pointer is null, and the
     pointer niche D5 promises reads it as absent.
   - `var xs: List<i64>` then `xs.len()` → the program exits with status
     255 and **no message**.
   - Same for closures, trait objects, `Map`, `Set`, `Channel`, `Task`,
     and structs or sealed values holding any of them.
2. **A constant table of an over-aligned struct does not compile.**
   `@align(64) struct Counter { hits: i64 }`,
   `const TABLE = [Counter(hits: 3), Counter(hits: 4)]`, then
   `loop (c in TABLE)` → clang: `initializer with struct type has wrong #
elements` (`%S.main.Counter { i64 3 }` for `{ i64, [56 x i8] }`).
   `constFields` ([consts.go](codegen/llvm/consts.go)) writes fields in
   source order without the plan's padding fields.
3. **Spec and code disagree on the memory-class threshold.** D121 says
   "128 bytes or more"; `isMem` tests `size > 128`.
4. **A runtime parameter is ignored.** `veles_task_set_unwrap(t, off)`
   discards `off` and `veles_task_value` hard-codes 8. Correct today (the
   payload is at 8); it silently breaks if the payload offset ever moves.

## 5. Proposed rules

### 5.1 The validity invariant (needs Q61)

> Every value a program can observe is valid for its type: a non-nullable
> pointer, `List`/`Map`/`Set`/`Channel`/`Task` handle, closure code pointer
> and trait-object vtable are never null; a sealed value's tag names one of
> its variants; a string's length is ≥ 0 and its pointer is non-null when
> the length is not 0. Code in `unsafe` that produces an invalid value has
> undefined behaviour. A value typed by C (`*raw T`, `extern` fields,
> `CLayout`) carries C's validity only: raw pointers may be null.

The compiler enforces it by refusing to read a local before it is assigned
on every path (Q61, recommended), which the checker already does for
fields in `init` blocks (flow facts in the narrowing state). That fixes
bug 1. The niche in §5.2 depends on this invariant.

### 5.2 Nullables without a flag where a value has a spare bit pattern

_Built 2026-10-26 (plan B32 step 4) as written below, packed structs
excluded as well; `types.Layout.Niche`, D5 addendum._

Rule: **the all-zero bit pattern is `null` for every `T?`** (unchanged:
`NullConst`, zeroed slots, fresh GC memory and `map.get`'s zeroed result
keep working). `T?` has no flag word when some field of `T`, at a fixed
offset, can never be all-zero under §5.1:

| `T` contains, at a fixed offset (through struct and tuple fields, element 0 of an `Array<_, N ≥ 1>`)        | `T?`                                       |
| ----------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| a non-nullable `*T`, `List`, `Map`, `Set`, `Channel`, `Task`                                                | same size as `T`; null when that word is 0 |
| a closure (code pointer), a trait object (vtable pointer)                                                   | same size as `T`                           |
| none of these (numbers, `bool`, `string`, raw pointers, C function pointers, extern structs, sealed values) | `{ i1, T }` as today                       |

- `T??` is `{ i1, T? }`: the zero pattern is used by the inner level, so the
  outer level pays a discriminant — exactly D5's "nesting costs a
  discriminant only at the nested level".
- `Some(x)` is `x` itself; `x == null` is one word compared with 0; a write
  through a narrowed place writes a valid `T`, which is never zero there, so
  the value stays present; `p = null` zeroes the whole value.
- Excluded on purpose: **`string`** (`{ null, 0 }` is a valid empty string
  today, and strings come from C through `*raw string`), **raw and C
  pointers** (C's NULL), **anything inside an `extern` struct** (C writes
  it), **sealed values** (tag 0 is a variant; making tags 1-based would
  touch every tag site for 2 uses in `selfhost` — parked).
- One function decides it: `Layout.Niche(t) (offset int, ok bool)`, used by
  `Of` (size of `T?`), `pointerOffsets` (a niche `T?` scans as `T`),
  `constInit` (`CSome` is the payload; `CNull` stays `zeroinitializer`),
  and by the nullable helpers, which become the **only** code that knows
  the shape: the direct sites in `placeAs`, the `SomeWrap` case of `expr`
  and `checked*` arithmetic move into `makeNullableT`/`nullFlag`/`nullVal`
  first, as a separate change with no change to the IR.

D5 amendment text, when built: "A `T?` is as big as `T` when `T` holds a
value that is never zero (a pointer, a collection handle, a closure, a
trait object); otherwise it carries a discriminant. Nesting adds a
discriminant at each further level."

### 5.3 Big arguments by address, without a copy

_Built 2026-10-26 (plan B32 step 3), through the existing memory class rather
than a second calling convention: every aggregate over 64 bytes joined it,
so all call paths (closures, vtables, tasks, C) already agreed on an
address; the copy at the call is skipped by the rule below, and the callee
copies by `Var.ReadOnly`. `bench/bigargs`: 88 ms → 46 ms (Go 90 ms)._

Rule: an argument of a Veles function whose type is an aggregate **larger
than 64 bytes** (measured, §3; tuned per target by the bench) is passed as
the address of a value that **no one can change during the call**:

- **Caller.** It passes the address of the place itself when the argument
  is a temporary (a literal, a call result) or a local that is neither
  address-taken (`&x`, a method that writes it), nor captured by a
  closure, nor a global; otherwise it copies into a fresh slot first, as
  D121 does today. A place reached through a pointer, a list element or
  `this` is always copied: something else may write it during the call.
- **Callee.** The parameter is read in place. If the body writes it (a
  field assignment, a method that writes its receiver — `sema/receivers.go`
  knows which — or `&param`), the callee copies it into its own slot on
  entry, as it already does for address-taken memory-class parameters.
  Coroutine callees already copy their parameters into the frame.
- **Unchanged.** `async f(big)` stores the argument in the task block;
  channel sends, list and map storage copy; extern "C" calls and
  `export` wrappers keep the C classifier.
- **LLVM.** The parameter is `ptr readonly nocapture align N`. `noalias` is
  added only after the stress runs below pass with it, since a mistake in
  the caller rule would then miscompile instead of being slow.
- **Returns** are left alone: LLVM already returns large aggregates
  through memory on arm64 and x86-64; whether an explicit `sret` without
  a copy is faster is unmeasured, so it is not proposed.

This applies to the existing memory class too (its argument copy becomes
conditional), which closes the §11 note on copying large arrays. D121's
text "an argument is the address of a copy the caller made" becomes "an
argument is the address of the caller's value, or of a copy when the
caller's value could change during the call". Bug 3 is settled in the same
change by making the code match the spec (≥ 128).

### 5.4 Smaller items

- **One layout truth** (bug 2, bug 4): `constFields` builds struct
  constants from `StructPlan` and `fidx` (padding fields included);
  `veles_task_set_unwrap` stores `off` and `veles_task_value` uses it; the
  sealed payload offset in `gcdesc.go`, `errorConvert` and `heldTaskAt`
  comes from one `Layout.PayloadOffset`.
- **Exact pointer maps for sealed payloads**: list the union of the
  variants' pointer offsets (shifted by the payload offset) instead of
  every payload word. Less scanning, and an `i64` that happens to look
  like a heap address no longer keeps garbage alive.
- **Over-aligned values inside tuples, nullables and sealed payloads**:
  D120 accepts that they are not over-aligned there. Lay those out with
  explicit padding too, so `@align(64)` holds everywhere. Low priority.

### 5.5 Parked, with the evidence

- **Reordering every struct automatically**: 0.4 % on `selfhost` (§3);
  D215 makes it opt-in (`compact struct`) instead.
- **A smaller sealed tag / payload at byte 4**: no candidate in the tree.
- **A niche in `string`** (needs every empty string to have a non-null
  pointer, including C-produced ones) and **in sealed values** (1-based
  tags): small measured use; revisit with numbers.

## 6. How we make sure nothing breaks

Per item, before the change lands:

1. **A layout agreement test** (new, Go): for a corpus of types (every
   struct, nullable, sealed and union in std, selfhost and the examples),
   check that `types.Layout`, the LLVM type, the GC descriptor and the
   constant initializer agree — sizes and offsets compared with clang's
   `DataLayout` through a generated IR probe, as `TestCLayoutAgreesWithC`
   does for C.
2. **Regression tests for the bugs**: §4.1 and §4.2 as `sema_test.go` /
   driver cases (fail before, pass after).
3. **Conformance**: a `sema/testdata/conform` file per rule (nested
   nullables `S??` with a niche `S`, narrowed writes, `p = null`,
   `Some(null)`, generics instantiated with a niche and a non-niche struct,
   a borrowed argument mutated by the callee, `&x` passed beside `x`, a
   closure that writes a captured `x` while `x` is borrowed, `async f(big)`
   then a write to `big`).
4. **Whole-tree differential run**: every example and `selfhost` output
   byte-identical, debug and `--release`; the self-host gates (G1–G5).
5. **GC and threads**: `VELES_GC_THRESHOLD=256`, `VELES_GC_POISON=1`,
   `VELES_THREADS=1/8` over the examples and the runtime stress.
6. **Goldens**: IR changes are expected only in the fixtures that contain
   the affected shapes; each diff is read, not blindly `-update`d.
7. **Benchmarks** (`veles-bench`): a new `bench/bigargs` (the §3 program
   with a Go reference) and `go run ./bench -record` before and after;
   `selfhost` parse time and peak heap as the real-program check.
8. **Linux and macOS**: the WSL script and `internal/macos-test.sh`.

## 7. Order and cost

| Step                                                 | Needs  | Touches                                                                        | Risk                                    |
| ---------------------------------------------------- | ------ | ------------------------------------------------------------------------------ | --------------------------------------- |
| 1. Bugs 2–4, nullable helpers centralized            | —      | `consts.go`, `veles_task.c`, `coro.go`, `expr.go`, `mem.go`, `isMem`           | low; IR byte-identical except bug fixes |
| 2. Definite assignment for locals (§5.1)             | Q61    | `sema` flow facts, error + fix, docs ch. 2, error reference                    | low; 0 uses in the tree                 |
| 3. Borrowed big arguments (§5.3)                     | —      | `expr.go` `call`, `gen.go` param binding, thunks, `mem.go`, D121 text          | medium; aliasing rule must be exact     |
| 4. Nullable niche (§5.2)                             | step 2 | `types/layout.go` `Niche`, nullable helpers, `gcdesc.go`, `consts.go`, D5 text | medium; contained once step 1 is done   |
| 5. Exact sealed pointer maps, over-aligned placement | —      | `gcdesc.go`, `types.go`                                                        | low                                     |
| 6. `compact struct` (D215)                           | —      | parser (Go and `selfhost/`), formatter, sema, `StructPlan`, hover              | low; placement only, one function       |

Self-hosting: the rules are a recursive walk over the type (`Niche`) and a
per-call check of the argument expression (temporary or clean local) — both
plain Veles. A compiler written in Veles gets them for free: `ast.Ident?`
fields shrink, and its large AST structs stop being copied on every call.

## 8. Decisions (asked 2026-10-26)

**Q61 → D214 (A).** **Q62 → D215**: declaration order guaranteed, plus the
user's addition of an opt-in modifier, spelled `compact struct`.

**Q61 — a local `var x: T` with no value.** Today it is zero-filled, which
gives null non-nullable pointers and handles (bug 1). Options:
(A, recommended) refuse reading it before it is assigned on every path
(Kotlin, Swift, Rust; the checker already does this for `init` blocks);
(B) allow it only for types whose zero is a valid value (numbers, `bool`,
`string`, `T?`, structs of those — Go's zero values) and refuse the rest;
(C) leave it — the bug stays and §5.2 cannot be built. No code in the tree
uses the form.

**Q62 — what the layout of a plain struct promises.** D120 says it is "the
compiler's" but nothing says more. Options:
(A, recommended) **declaration order with C's alignment rules is
guaranteed** for structs and tuples; the encodings of `T?`, sealed values,
error unions, closures and trait objects are unspecified and may change
between versions (the measured gain from reordering is 0.4 %; a
predictable layout is what the low-level user and the debugger need);
(B) everything unspecified, the compiler may reorder fields (Rust's
default; a `@packed`-like attribute would then be needed to pin order);
(C) leave it unstated.
