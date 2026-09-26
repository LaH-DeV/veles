# Veles — Language Specification

**Working draft v0.34** — language design complete; D58 adds the derivation story D51 deferred, D59 the standard library's cryptography. Remaining work is not language design: C ABI FFI, and the v0.1 build plan.

Decision IDs are stable. They are never renumbered; superseded decisions are struck through and replaced by a new ID.

---

## 1. Identity

| | |
|---|---|
| Name | Veles |
| Source extension | `.vs` (module file), `.vss` (script: a one-file program) |
| Classification | Compiled, statically typed, garbage collected |
| Primary target | Native machine code |
| Secondary target | WebAssembly (deferred — not day-one) |
| Influences | Kotlin (ergonomics), Rust (abstraction), Go (memory & runtime), TypeScript (modules) |

**Vocabulary.** *Package* = a distributable project. *Module* = a directory. *File* = a file. These three words have exactly these meanings throughout.

---

## 2. Semantics — settled

### D1 — Memory management: tracing garbage collector

Not ARC, not ownership/borrowing. The language does not expose lifetimes and does not require the programmer to reason about deallocation.

### D2 — Concurrency: coloring-free surface, compiler-inferred suspension

Users write task launches, channels, and blocking-looking I/O. **No `async` on a function declaration, ever** — that token is reserved as a prefix on a *call expression* meaning "launch as a task". `await` suspends on anything suspendable: a channel, a task handle, a timer.

The compiler performs call-graph analysis to determine which functions can suspend and CPS-transforms only those into state machines. Execution is **stackless coroutines** — no stack switching, no growable-stack scheduler.

*Rationale:* Go's ergonomics with an implementation that ports to WebAssembly later without re-engineering the concurrency layer or the GC's root scanning.

*Amended by D40:* inference covers direct calls only. Trait methods and function types declare their effects explicitly, because dynamic dispatch makes the reachable implement set open.

*Known costs:* Stack traces and profiler output are worse than green threads would give. Least battle-tested of the available approaches.

Channel syntax is settled in D16.

### D3 — Structured concurrency

Scoped task lifetimes, cancellation trees, no orphan tasks. Orthogonal to D2.

### D4 — Errors: `Result<T, E>` with `throws` as sugar

`Result<T, E>` is canonical. A `throws` function compiles to **exactly the same ABI** — error in a register, no unwinding tables, no stack ripping. `try` is propagate-or-unwrap sugar. Inside a `throws` function `throw e` is the failing return — sugar for `return Err(e)` — and a plain `return v` is the success case; `Ok`/`Err` are only spelled when a `Result` is handled explicitly (`when (r) { is Ok(v) => ...; is Err(e) => ... }`) or built as a value.

Consequence: `throws` and explicit `Result` interconvert freely; libraries cannot split into two incompatible error camps; the happy path is zero-cost.

**A `Result` must be consumed.** A `throws` call as a bare statement, or bound to a name that is never read, is an error — regardless of whether the caller is itself `throws`, since only `try` propagates. `val _ = f()` is the explicit discard. (Non-`Result` bindings that are never read are a warning.)

**Errors are declared, and every error implements `Error`.** The prelude declares `trait Error { fun message(): string = "$self" }`. An error type is declared with a contextual keyword: `error ParseError { text: string; fun message(): string = ... }` is sugar for `struct ParseError { text: string }` plus `implement Error for ParseError { override fun message() ... }`. Inside an `error` body `message` needs no `override` (there is exactly one trait in play; writing it is allowed), and no other method may carry one — they are inherent. A field `message: string` with no `message()` written is used as the message, so `error Failed { message: string }` is thrown as `Failed(message: "...")`; `Panic` is exactly that. **Only errors can be in error position** (a `throws` clause, a `throw` operand, the `E` of a `Result`): throwing a plain struct is an error naming the fix (`error Dog { ... }`), non-structs are rejected, and a generic `throws X` needs the bound `X: Error`. `implement Error for T {}` on an existing struct is the escape hatch for types one does not own. Because every member of an error union implements `Error`, **the union exposes `Error`'s methods directly**: `e.message()` on a `ParseError | RangeError` compiles to a tag switch, no `when` required. An error escaping `main() throws` is reported through `message()`. This is Go's one-method `error` interface with structured payloads kept as fields, and the declaration form makes "is an error" a property of the type rather than of its uses. `error` is a keyword only at declaration position (`error Name`); elsewhere it is an ordinary identifier, so `is Err(error) => ...` is unaffected. Rejected: treating every struct as an implicit `Error` (it made `throw Dog()` legal); `struct X : Error` (in D12 that syntax means variant membership); and making `message` a mandatory field (it cannot be computed from the other fields, which is the common case).

**`is Ok` / `is Err` smart-cast to the payload.** After `if (r is Ok)` the subject `r` *is* the `T`; in the `else` branch (or after `!is Ok`) it is the `E`. `r.ok` and `r.err` are the same tests spelled as properties (v0.23): `if (r.ok)` narrows exactly like `if (r is Ok)`, and the teaching form. In a `when`, `is Ok` / `is Err` arms narrow the subject the same way, and `when (val r = expr)` names an expression subject for the arms. This is D5's `T?` → `T` rule applied to `Result` (and `Option`'s `Some`): the wrapper variants are handled, never held as values. A general two-variant sealed type gets the complementary narrowing in the failed branch too, but only `Result`/`Option` read through to a payload.

Error type is **typed but inferrable** for ordinary functions. *Amended by D40:* trait methods must declare it. *Extended by D45:* inference across callees with different error types produces a union.

### D5 — Nullability: `T?` with flow-sensitive smart casts, nesting

Kotlin-style syntax and narrowing, but `T?` is **genuine sugar over `Option<T>`** and therefore nests: `String??` is a distinct type from `String?`. Swift is the working precedent.

Rationale: Kotlin's collapsing `T?` makes generic lookups ambiguous — `map[key]` returning `V?` cannot distinguish "absent" from "present but null" when `V` is itself nullable. Kotlin's own workaround (`containsKey`) is two lookups and racy. And collapsing is not simpler: Kotlin had to add the `T & Any` intersection type to recover the expressiveness it destroyed.

- `null` is sugar for the `None` case
- Implicit promotion: assigning `String` where `String?` is expected wraps automatically, so the constructor is almost never written by hand
- The explicit spellings are `Some(x)` and `null` (a leading-dot case syntax was tried and dropped: a line starting with `.` continues a method chain, so `.none =>` could never begin a `when` arm)
- `String??` arises only from generic instantiation, essentially never from hand-written source

**Representation is dual:** nullable pointer → niche-optimized, zero cost. Nullable value struct → tagged union with a discriminant. Nesting costs a discriminant only at the nested level. Does not disturb D8: nullable and non-nullable pointers are the same GC shape.

**Precedence with pointers:** `?` binds tighter than `*`, so `*User?` is a pointer to a nullable `User`. A nullable pointer is written `(*User)?`.

This follows the conventional rule that postfix binds tighter than prefix, but it hands the terse form to the rarer case — optional references are common, pointers to nullable values are not — and it reads against the representation note above, where the nullable pointer is the free one. Expect `(*T)?` to be frequent.

**What a smart cast is about: a stable place.** A fact (`x != null`, `x is Circle`, `r is Ok`) attaches to a *place*: a local variable, or a chain of direct struct fields starting from one (`config.cause`, `self.address` in a non-`mut` method). Reads of the place are narrowed until something could change it: an assignment to the place or to a prefix of it (`u.address = ...`, `u = ...`), a `mut` method call on the root, or `&root` being taken — after which no new facts form on the root's fields either. Nothing reached through a pointer, `?.` or an index is a place: with `p: *User`, `p.address` may change through another pointer between the test and the use, so it is never narrowed (this is Kotlin's "stable value" rule with aliasing made explicit; Veles has pointers, Kotlin does not). Inside a `mut` method `self` is a pointer, so `self.field` is bound to a `val` first. Facts on the variable itself survive a `mut` call or `&` (they cannot change which variant a value is); only the field-path facts are dropped.

**Smart casts and `?.` reach places, not copies (v0.24, revised v0.27).** A narrowed place is writable: after `if (p != null)` both `p.n = 5` and `p.bump()` act on the payload inside `p`, and after `is Circle` on the variant inside the sealed value — the same storage a read sees. `?.` has the same property, on calls and on assignment: `x?.f = v` and `x?.f op= v` write only when `x` is present and do nothing otherwise (Kotlin's and Swift's rule), and `x?.m()` calls a `mut fun` on the value where it lives. The receiver may be a nullable variable or field, or a nullable pointer — which is how a collection element is reached: `xs.ref(i)?.n += 1`, `m.ref(k)?.bump()` (D25). A write through `?.` into a temporary value (`make()?.n = 1`, and `xs.at(i)?.n = 1`, since `at` returns a copy) is an error rather than a silent no-op, and `val` bindings stay closed to it.

### D6 — Polymorphism: Rust-style nominal traits

Nominal, not structural. Implementable on foreign types after the fact. Coherence/orphan rules TBD.

### D7 — Data representation: values + explicit pointers

Go's model. `struct` values live inline; pointers are explicit.

### D8 — Generics: GC-shape stenciling

One compiled body per GC shape, not per type. All pointer-shaped instantiations share code; distinct value layouts get their own.

*Cost:* trait method calls inside generic code dispatch through a dictionary and **cannot be inlined**. Generic code will frequently be slower than the hand-written equivalent. In direct tension with D6, which creates an expectation of zero-cost abstraction. See §6.2.

### D9 — Trait objects: implicit boxing

A value struct coerced to a trait object is boxed automatically, Go-style. Convenient; hides allocations inside ordinary-looking assignments.

**Sealed and open trait values are spelled identically but are not the same thing.** `s: Shape` where `Shape` is sealed is an inline tagged union — a value, no allocation (D12). `d: Display` where `Display` is open is a boxed trait object. Nothing in the type distinguishes them.

Distinct syntax for boxed trait objects was considered and rejected in favour of keeping the surface uniform. The consequence must therefore be documented prominently: whether a trait-typed binding allocates depends on whether that trait is sealed, and it is invisible at the use site.

### D10 — Pointers: `&local` with escape analysis, interior pointers permitted

`&x` on a local is legal; escape analysis promotes it to the heap when it outlives the frame. Pointers into struct fields are permitted.

---

## 3. Modules and packages — settled

### M1 — Package identity lives in a manifest

`veles.toml` at the package root holds name, version, dependencies, license, exports, build config. **No source file declares package identity.** The resolver builds a dependency graph by reading manifests, never by parsing Veles source.

A monorepo contains multiple manifests, one per package.

**The package is the unit that runs.** A package is a program when its root module declares `fun main()`, and a library otherwise; a module is never a program on its own. The tools locate the package from any path inside it (the nearest `veles.toml` above), so `veles run` on a sub-module directory runs the package; a directory with no manifest above it is its own single-module package. `check` accepts modules and libraries.

*Amended (v0.31) — scripts.* A **script** is a file named `*.vss` and is a package by itself: its root module is that one file. It ignores the directory it sits in — sibling `.vs` and `.vss` files are not part of it, and it is not part of any directory module (the directory's module still reads only `.vs` files), so several scripts and a directory package may share a folder without seeing each other's `main`. A script has no manifest and no modules of its own; it imports the standard library, and will import dependencies once there is a way to name them without a manifest (open). It exists for the program that is one file — a tutorial step, a tool, an experiment — where a directory per program is ceremony. The rule stays M3's: the resolver decides membership from the file name, never by reading the source.

**Documentation comments.** `/// ...` lines and `/** ... */` blocks directly above a declaration, field or method are that item's documentation, carried by the compiler (markdown, shown on hover). A doc comment at the top of a file, separated from the first declaration by a blank line, documents the module; a module's documentation is its files' top comments in file order. Ordinary `//` and `/* */` comments are discarded.

### M2 — A module is a directory

All `.vs` files in a directory share one namespace and one import path. **Files in the same module do not import each other** — mutual recursion between them is free and always legal.

### M3 — Module identity derives from directory path

A module's name is its directory path relative to the package root. **No overrides.** Directory names that are not legal identifiers are handled by a normalization rule, not by a declaration. The resolver stays manifest-only and never parses Veles source to build the module graph.

### M4 — Import cycles between modules are forbidden

The build graph is a DAG. If two modules need mutual recursion, that is evidence they are one module — and since modules are directories, merging them is moving files.

This is not a style rule. Both of Veles' inferred effects (D2 suspension, D4 error type) are inferred per-module; a DAG makes inference a single topological pass, while cycles would require fixpoint iteration across module boundaries.

### M5 — `public` grants package-wide visibility

Default visibility is module-private. `public` makes a declaration visible to the rest of the package. Nothing escapes the package except through the manifest's `exports` field.

Consequence: library authors get a deliberately curated public surface, consumers cannot reach into internals, and Veles never needs Go's magic `internal/` directories.

*Amended (v0.29) — spelling and a third level.* The keyword is `public`, not `pub`: it now sits next to `private`, and the pair every Java, C#, Kotlin, Swift and TypeScript reader knows is `public`/`private`; the three extra characters on each exported declaration were weighed against that and lost. `export` was rejected for it — exporting is what the *package* does, in the manifest, module by module, and a keyword by that name on a struct field would claim a boundary the field does not cross. (Whether the package boundary itself should one day be spelled in source with `export` rather than in the manifest is an open question the user has reserved; nothing here prejudges it.) The levels are therefore: **`private`** — visible only inside the type's own declarations: its methods, its `implement` and `extend` blocks in the same module, and its `static val` initializers (Swift's rule for extensions in the same file); **nothing** — the module, as before, which keeps small programs light; **`public`** — the package. `private` exists for fields and methods only; a module-level declaration is already module-private. A private field cannot be read, assigned or bound in a pattern from outside. In a constructor call the rule turns on the default (v0.30): a private field *with* a default is the type's own state and outsiders leave it to the default; one *without* a default is the initial state only the constructor call can supply, so it is given from anywhere the type is visible and is private from then on — Kotlin's `class Parser(private val toks: ...)`, without the syntax. Swift's rule (a private stored property makes the memberwise initializer private) was the v0.29 behaviour and was dropped: it forced a `static fun` on every type that merely wanted to hide what it was given. That is the whole encapsulation story, with no getters and no `friend`. Motivated by a one-file program whose request handlers reached into a store's counter: modules were the only boundary, and a directory per invariant is too heavy.

*Addendum (v0.30) — the unwritten level has a name: `internal`.* "Nothing written means the module" was judged not self-explanatory, so the level is called *internal* (Kotlin's and Swift's word for it; `protected` was considered for this level and given instead to write-protection of fields, D22 addendum, where its "the owner only" sense fits) and may be written: `internal fun helper()`, `internal x: i64`, `internal struct Note`, on the same positions as `public`. It changes nothing and is never required; the formatter keeps it where the author wrote it; `public internal` is a contradiction and an error. The example package's `internal` module was renamed `support`, since the word is a keyword now.

### M6 — Import paths are logical

Resolved by the package manager against the manifest. No filesystem-relative string paths.

*Addendum (v0.28) — modules only, one statement, many imports.* A `use` imports modules and nothing smaller: members are always qualified (`geometry.Point`, `io.println()`), and `use geometry as geo` renames the module when the prefix is long. The braced name-import form (`use geometry.{ Point as P, norm }`) is gone — Go's reasoning: a qualified name says at the use site where a thing comes from, two modules may both declare a `Point` with no renaming, and there is one way to write a call. A parameter or local named like a module shadows it (`fun mkdir(path: string)` cannot call `path.dir`); the answer is an alias at the import, not a name-import escape hatch. Type aliases (`type P = geo.Point`) are the remaining way to shorten a type name and are not in the language yet; add them if the prefix on types proves to hurt. `use` takes a comma-separated list: `use fs, io, os`, `use geometry as geo, shapes`; a comma at the end of a line continues the list. Import order and grouping carry no meaning, so the formatter owns them: consecutive `use` lines become one sorted list per origin — the standard library first, then everything else — one `use` statement each. Go groups its imports the same way (goimports), by convention rather than syntax: a separate spelling for standard-library imports would turn every move of a module between the library and a package into a source edit, and the compiler already knows which is which.

---

## 4. Syntax — provisional

Nothing here is locked. Current working sketch:

```vs
use io, otherModule

fun main() throws {
  val sum = add(5, 7)
  printSum(sum)
}

fun add(a: i32, b: i32) = a + b

fun printSum(sum: i32) {
  io.println("The sum is $sum")
  otherModule.someFunc()
}

// otherModule/funcs.vs
public fun someFunc() {
  //
}
```

### Resolved syntax points

- **No `: null` return type.** `null` is a *value* — an inhabitant of `T?`. Using it as the name of the unit type conflates absence-of-value with "returns nothing" and collapses under generics (`Result<null, E>` is meaningless). Use a real unit type with exactly one inhabitant; omit it entirely when there is no return.
- **String interpolation, not `printf`.** Given `"$sum"`, format specifiers are redundant.
- **One import form.** Logical paths only.
- **Explicit integer widths.** Go's platform-dependent `int` is a portability hazard and would bite hard on wasm32. `int` may exist as an alias at most.

- **`val` / `var`**, not `const`, for runtime-immutable bindings. `const` is reserved for compile-time constants.
- **`=`**, not `=>`, for expression bodies, keeping the arrow free for lambdas.
- **`throws` follows the return type**: `fun f(): T throws` and `fun f(): T throws IoError`.

### Still provisional

- Semicolon rule — recommend copying Go's automatic insertion wholesale
- Lambda syntax
- Trait declaration and implement syntax

### D11 — Mutability: `val` / `var` on bindings only

Kotlin's model. Immutability is **shallow and non-transitive** — a `val p: *Point` still permits mutation through the pointer. Veles cannot express "immutable view of mutable data" in the type system. `const` is reserved for genuine compile-time constants.

*Amended (v0.30) — the binding says nothing about the value.* With mutability declared on fields (D22), `val`/`var` on a binding govern rebinding only, for every type alike: a `val c: Counter` can have its `var` fields assigned and its methods called, as a `val xs: MutableList` could always be pushed to. What cannot change is decided by the type — a bare field is never assigned — and D22 states the guarantee that replaces "a `val` struct is immutable". A global `val` is the one exception: it is a constant, shared by every task (D35), so neither its fields nor a method that changes its receiver may touch it.

### D12 — Sum types: sealed traits, no `enum` sugar

```vs
sealed trait Shape
struct Circle : Shape { radius: f64 }
struct Rect   : Shape { w: f64, h: f64 }
```

- **Sealing is a layout feature, not just an exhaustiveness feature.** Under D9 a trait object boxes implicitly; a plain open trait would heap-allocate every sum value, including every `Option<T>`. A closed variant set lets the compiler compute a maximum size and lay the value out inline as a tagged union.
- Variants must be declared in the **same module** as the trait. If a `public` sealed trait could be extended elsewhere in the package (reachable under M5), exhaustiveness checking would break across modules.
- Variant names are **scoped and importable**: `Shape.Circle` by default, bare `Circle` after explicit import. Top-level variants were rejected — `Loading`, `Error`, `Empty`, `Pending` collide across any two sealed traits in one module.
- **All variant fields must be named.** No positional `Circle(f64)`. This was originally forced by the absence of destructuring; D13 has since added it, but named fields are retained on their own merits and destructuring binds by field name.

*Addendum (v0.32).* "No `enum` sugar" means no second spelling of a sum type. A closed set of plain *values* is a different thing and has its own keyword since D57 (`enum Phase { Red, Amber, Green }`): every member is a number, `when` over it is exhaustive by member, and it carries no data — the moment a member needs a field, it is a sealed trait.

### D13 — Pattern matching: `when`, as an expression

Type-test with smart casts, **and** destructuring patterns (revised — destructuring was originally excluded). A subject may be bound in the head — `when (val r = parse(s)) { is Ok => r ... }` — so type tests can narrow an expression that has no name of its own (v0.23).

```vs
val area = when (shape) {
  is Shape.Circle -> PI * shape.radius * shape.radius
  is Shape.Rect   -> shape.w * shape.h
}
```

Narrowing inside a branch falls out of D5's flow typing. Destructuring binds **by field name**, since D12 requires named fields, with a shorthand when the binding name matches the field:

```vs
when (shape) {
  is Shape.Circle(radius) => PI * radius * radius
  is Shape.Rect(w, h)     => w * h
}
```

The bare `is Shape.Circle =>` form remains available; destructuring is additive. Nested patterns such as `is Some(Some(x))` become expressible, which matters because D5's `T?` nests. Tuples (D37) destructure positionally.

**Cost:** exhaustiveness checking becomes pattern-coverage analysis rather than variant-set membership — Maranget's usefulness algorithm, not a set check. Budget for it.

**Subjectless form.** `when { cond => ... }` with no scrutinee acts as a condition chain, as in Kotlin.

**Pattern guards** are supported: `is Shape.Circle(radius) if radius > 10 =>`. A guarded arm **never counts toward exhaustiveness**, since the compiler cannot prove a guard is ever satisfied — Rust's and Scala's rule. This catches people out regularly and needs a clear diagnostic.

Exhaustiveness is required; `else` opts out. **Lint against `else` when the scrutinee is sealed**, since it silences exactly the error you want when a variant is added later. *Scoped (v0.24):* the lint fires when the `else` stands for exactly one missing variant (an enumeration that a later variant would fall into) or when it is unreachable (every variant has an arm; the fix removes it). One-variant extraction — `is JStr(v) => v  else => null` with several variants left — is the honest spelling and stays silent; two mid-size programs hit it eight times.

*Addendum (v0.28) — narrowing a list.* A predicate cannot promise a type (`filter(s => s is Circle)` yields `List<Shape>`; nothing a lambda learns crosses a call boundary), and Veles deliberately has no user-written type predicates (TypeScript's `x is T`), which the compiler cannot verify. `xs.filterIs<Circle>()` states the variant as a type argument and yields `List<Circle>`; `filterNotNull()` and `oks()`/`errors()` do the same for `T?` and `Result`.

### D14 — Variance: deferred, generics are invariant

`None<T>` keeps a zero-sized phantom parameter, resolved by inference at the construction site; `null` sugar hides it entirely. This is how Rust's `None` behaves.

Rejected: invariance combined with a `Nothing` bottom type, which does not compose — invariance means `Option<Nothing>` is not assignable to `Option<String>`, so the bottom type buys nothing without covariance.

**Deferring is safe.** Turning an invariant parameter covariant later only widens what is accepted, so existing code keeps compiling. Go ships no variance at all; Rust exposes none to users. The eventual cost is auditing the stdlib for widenable parameters when `out`/`in` are added.

### D15 — Specialization is semantically invisible

Whether a trait call was specialized or dispatched through a dictionary must never be observable. Therefore, as hard rules:

- No per-instantiation static state
- No Rust-style `specialization` where a more specific implement overrides a generic one

With those in place, specialization is purely an optimization policy. Veles can ship dictionary dispatch in v1 and add auto-specialization later with no language change. The §6.2 question becomes permanently deferrable.

### D16 — Channels are methods, not operators

`await ch.recv()` and `ch.send(x)`. Send and receive stay syntactically symmetric.

**Rule required by D2:** `await` is mandatory only on primitives that *always* suspend — channels, timers, task handles. Ordinary function calls that transitively suspend carry no marker. This marks the genuine yield points without reintroducing viral coloring, but it must be a stated rule rather than an accident of the primitive set.

### D17 — Trait coherence: global, enforced at link time

No orphan rule. Any module may implement any trait for any type, but **at most one implement may exist per (trait, type) pair across the entire program.**

Module-scoped resolution was considered and rejected. Under D9 a trait object's vtable is built at the boxing site, and under D8 the dictionary is chosen at the instantiation site, so scoped impls would make "this type implements this trait" stop being a fact about the type. The concrete failure: a `HashMap<Foo, V>` populated in one module and read in another silently fails to find present keys, with no error anywhere.

Two consequences to design for:

- **Conflicts should be caught at dependency resolution, not link time.** Impls are visible in the materialized module interfaces (§5), so the package manager can detect a collision while resolving and report both module paths, instead of emitting a linker error with no useful context.
- **Adding an implement is a breaking change.** A library shipping a new `implement Display for Foo` in v1.1 can collide with an implement a downstream user already wrote. Rust's orphan rule eliminates this class; Veles has traded it for flexibility. The semver policy must state that new impls require a major version.

### D18 — Strings: validated UTF-8, immutable, byte-indexed

*Addendum:* `len()` is the byte length; the scalar view is available on demand as `charCount()` and `chars()` (a `List<string>` of one-code-point strings), so character-level work never goes through byte indices.

- Storage is UTF-8 and **validity is guaranteed**, unlike Go. Required at FFI boundaries: bytes entering from C must be validated.
- A single immutable `string` type plus a builder for construction. No owned/borrowed split — with a GC and no borrow checker it would buy nothing.
- `len` counts bytes. Byte-level indexing is available.
- Iteration is explicit: a bytes view and a scalar-value view. **Grapheme segmentation lives in an opt-in stdlib module**, so Unicode segmentation tables don't land in every binary.
- Rejected: UTF-16 (Java/C#/JS), where anything outside the BMP occupies two slots and indexing can split a surrogate pair.

*Resolved (§4b):* byte access is the explicit `s.byteAt(i)` / `s.bytes()` view; there is no subscript on text (and, since v0.24, no subscript anywhere — D25).

*Addendum (2026-09-25) — identifiers are UAX #31, and source text is checked for Trojan source.* Until now every byte ≥ 0x80 was an identifier byte: no Unicode tables to scan a name, and `val café = 1` worked — but so did a no-break space, a zero-width space and U+202E RIGHT-TO-LEFT OVERRIDE inside a name, the "Trojan source" family (CVE-2021-42574) where a file reads one way and compiles another. Decided (user, over the smaller "refuse a fixed list" option): an identifier is `_` or an **ID_Start** character followed by **ID_Continue** characters, the Unicode Standard Annex #31 definitions (`L + Nl + Other_ID_Start`, then `+ Mn + Mc + Nd + Pc + Other_ID_Continue`, minus `Pattern_Syntax` and `Pattern_White_Space`). Anything else above ASCII outside a string or comment is an error that *names* the character — "U+00A0, a no-break space; use an ordinary space", "U+202E, a bidirectional control character, which makes text display in a different order from how it compiles" — because the reader cannot see it. Beyond identifiers, as Rust does: a bidirectional control in a **comment** is an error (that is where the attack hides code); in a **string** it is a warning suggesting the `\u{...}` escape (it may be legitimate text, but should be visible). A byte-order mark is skipped at the very start of a file and is an error anywhere else. Identifiers are not NFC-normalized (two spellings of `é` are two names; Go's standard library has no normalizer). Cost: Unicode property tables in the lexer — Go's `unicode` package here, and a generated table in the self-hosted lexer (veles-selfhost-frontend-plan.md §4.2, amended).

### D19 — A slice that splits a code point returns `string?`

Not a panic. Boundary violation is recoverable and flows through the smart-cast machinery of D5.

### D20 — Panics: unwind to the enclosing task scope

Index-out-of-bounds, null dereference, and division by zero are not `Result` — every array access would return one. They raise a panic that unwinds to the enclosing task scope (D3), cancelling that scope's children.

**Implementation interaction with D2.** A suspended task's frames are heap-allocated state machine objects, not native stack frames, so unwinding must traverse both representations, possibly interleaved. LLVM's `invoke`/landing-pad machinery covers only the native half.

Because the GC means there is no drop glue to run, a cheaper option is available: compile panics as a hidden error-return path — the same mechanism as D4 — which works identically in both frame kinds at the cost of a predictable branch on return and some code size. It cannot catch panics originating in FFI or from signals.

### D21 — Integer overflow: checked in debug, wrapping in release, `+%` always wraps

*Amended (v0.23) — bitwise operators.* Integers have `&`, `|`, `^`, `<<`, `>>` and unary `~`, with Go's grouping (`&`, `<<`, `>>` at the level of `*`; `|`, `^` at the level of `+`). A shift count may be any integer type; a count at or beyond the width yields 0 (the sign fill for a signed `>>`), never undefined behaviour. Literals may be hexadecimal (`0xFF`) or binary (`0b1010`).


Rust's model, plus an explicit operator. `+` is checked in debug builds and wraps in release. `+%` means "I intend wrapping, in every build" — Rust's `wrapping_add`, promoted to an operator.

*Known wart:* overflow behaves differently between builds, so a bug that panics loudly in tests can wrap silently in production.

*Resolves the D20 interaction:* release builds have no overflow checks, so no landing pads on arithmetic.

*Default integer type:* an integer literal with no expected type is `i64`, and so are lengths, indices and counts. There are no implicit widenings (a `T` is never silently an `i64`), so having one default everywhere is what keeps ordinary code cast-free; the narrower types are for layout, C interop and bit work, and a literal adapts to them wherever one is expected.

### M7 — Version resolution: minimal version selection

Go's MVS. Reproducible without lockfiles and far simpler to implement than SAT solving.

Requires: a hard "no breaking changes within a major version" culture, since MVS trusts that promise completely; the major-version-in-import-path convention for v2 and later; and an explicit upgrade command, because MVS selects the *minimum* satisfying version and will not pick up security patches on its own.

### D22 — Method receivers: implicit `self`; mutability is declared on the field (`var`)

**REVISED (v0.30).** The original rule — `mut fun` marks a method that assigns its receiver's fields, and only a `var` binding may call one — is withdrawn. It was honest for structs of scalars and silent for the common shape of a domain type, a struct holding a collection: `val s = Stack()` did not stop `s.items.push(x)` in a plain `fun`, so the signature said "does not mutate" and the reader was misled. Making `mut` transitive through reference fields was considered again and rejected again: it needs to know which structs are values and which are handles, which is a `class`/`struct` split (Swift's) that the language does not want, or an ownership system.

**The rule.** Whether a field can change is written on the field:

```vs
struct Counter {
  label: string          // bare: set by the constructor call, never assigned again
  var n:  i32 = 0        // var: assignable

  fun get(): i32 = self.n
  fun bump() { self.n += 1 }     // no marker: a method may assign the receiver's var fields
}
```

- A **bare field** is never assigned after construction — not through `self`, not through a `var` binding, not through a pointer, not through `loop (&x in xs)`. The compiler reports it with a fix that inserts `var`.
- A **`var` field** is assignable through any place: `self`, a binding whether `val` or `var`, a pointer, an element reference. `val`/`var` on a binding govern rebinding only (D11, amended) — as they always did for a `MutableList`.
- `var` governs *that slot*. `o.inner.x = 1` needs `x` to be `var`, not `inner`: what is inside a field is governed by its own type, and a method call on `o.inner` could always reach it. The guarantee is therefore stated transitively: **a type with no `var` fields and no mutable collections reachable by value cannot change, whoever holds it.** Fields hold the truth about mutability where a reader looks for it, and `private var` next to a bare handle (`private var next: i64; private items: MutableList<Note>`) says exactly which two things in a type ever move.
- `self` is not assignable as a whole (`self = ...`): assign its fields, or return the new value.
- There is no method marker. Kotlin's `val`/`var` properties are the model; Swift's `mutating` and Rust's `&mut self` were the alternatives, and both exist to gate the *call* on the binding, which is the thing this design gives up. A struct is still a value: a method called on a temporary — `xs.at(i)?.bump()`, `make().bump()` — changes a copy that is discarded, which is an error when the method changes its receiver and returns nothing (the rule that guarded `mut fun` on a copy, D25 v0.27, kept). A struct passed to a function is a copy the callee may change without the caller seeing it; that is Go's value receiver and it is one rule rather than two.

**Calling convention.** Every method receives a *pointer to the place it was called on*, and `self` reads as the value; a temporary is spilled first. The representation is not declared and not observable, as before. Two facts about each method are computed once every body is checked (the receiver pass), to a fixpoint over the call graph: whether it *may write* its receiver (assigns a field of `self`, takes a pointer into it, calls a method that does) — this is what the discarded-copy error uses — and whether the receiver *may escape* (a closure captures `self`, `&self` or a pointer into `self` is taken, a callee does either). A receiver that may escape cannot stay on the caller's stack: its variable is heap-allocated, the same promotion `&x` performs (D10). This closes a hole the `mut fun` design had: a closure over `self` in a `mut fun` held a pointer to the caller's stack frame.

**Interaction with narrowing (D5).** `self` is a place like a local and its field paths narrow; a method call on a variable or on `self` drops the facts about paths through a `var` field and keeps those through bare fields, which no call can assign.

**Interaction with tasks (D35).** A sendable closure shares its captures by reference, so it may capture only `val`s of Sendable types *without `var` fields* (transitively, by value; collections and `Mutex` excluded as before) — another task could otherwise watch them change; the error names the capture. `self` is judged by the receiver's type. `async recv.m()` copies the receiver into the task, as it always copied arguments. Elements of an immutable `List` are read as copies, so `var` fields inside them are unreachable and do not count.

Rejected — Go's declared receiver (`(c Circle)` vs `(c *Circle)`): it declares representation, which produces two well-known failures. A value-receiver method that mutates silently modifies a copy with no diagnostic, and method sets diverge so that `Circle` does not satisfy an interface that `*Circle` does. Neither can arise here, because no representation is ever declared.

Rejected — Rust's `&self` / `&mut self`: enforced by a borrow checker Veles does not have, so the distinction would be decoration.

Rejected — private-by-default fields alongside `var`: with bare fields immutable, an exposed field is readable and nothing else, which is what a record is for; three visibility levels need one unmarked level and it must be the same for fields, methods and top-level declarations (M5).

*Addendum (v0.30) — `protected var`, and `val` written out.* Between "never assigned" and "assigned by anyone" sits the most common shape of managed state: a value everyone may read that only its type updates — C#'s `{ get; private set; }`, Swift's `private(set) var`, Kotlin's `var x` with a `private set`. Veles spells it `protected var`: `public protected var count: i64 = 0` is read wherever it is visible and assigned only inside the type's own declarations (methods, `implement` and `extend` blocks, the same set `private` uses). It replaces a `private var` plus a getter. The two modifiers answer two questions in order — the first word says who *sees* the field (`private` / nothing or `internal` / `public`), the rest who *assigns* it (bare or `val`: nobody; `protected var`: the type; `var`: anyone who sees it). `protected` always qualifies `var` (a bare field has nothing to protect; the compiler says so), and `private protected var` is refused as redundant — nobody outside can see the field, so `private var` already says it. It never restricts reading. For narrowing and for Sendable a `protected var` counts as `var` (the type can change it). The bare field may be written `val` for emphasis; the formatter keeps the author's word and never requires it.

On the word: `protected` means "the class and its subclasses" in Java, C#, C++ and Kotlin. Veles has no inheritance and will not get it — shared behaviour is a trait (D6/D9), closed families are sealed traits (D12) — so the subclass reading has nothing to attach to, and the word is free to mean what it says: protected from writes by anyone but the owner. The alternatives were weighed: `readonly` (TypeScript's and C#'s word for what Veles's *bare* field already is — it would invert the term for those readers, and it sits confusingly next to `val`), `var(private)` (not Veles syntax), Swift's `private(set)` (same). Making the protected level the *default* was proposed and rejected: it would take back the at-a-glance guarantee (a struct of bare fields cannot change) and make most `val` structs uncapturable by sendable closures.

*Addendum (2026-09-25) — a changed parameter is a lost change.* A struct parameter is the caller's value copied (D7), and `val`/`var` govern rebinding only, so a function may assign a `var` field of a parameter, or call a method that writes `self` on it — and the caller never sees it: `fun step(f: Fuzzer) { f.rng.next() }` advances a copy of the generator. This is a **warning**, on the parameter, naming the change and the fixes (`f: *Fuzzer`, or return the value). Silent when the copy is the point (the function returns the parameter's type, or uses the parameter whole after changing it), when the parameter's address is taken, and for writes through a reference field (seen by the caller). Rejected: Swift's immutable parameters, which would change the rule above for one case.

### D23 — Methods in the struct body; `implement` blocks for traits; no extension functions

Inherent methods are declared inside the struct. Trait implementations go in `implement Trait for Type` blocks, which are required anyway since D6 permits implementing traits for foreign types.

**`struct Circle : Shape` is not a conformance declaration** — it is variant membership in a sealed set, required by D12. The header says the type is one of the trait's variants; the `implement` block supplies the methods. A non-sealed trait has no header clause at all.

Putting trait implementations in the struct body was considered and rejected. It would mean a trait implement could only be written by whoever controls the type's source, which makes `implement Display for SomeForeignType` impossible — removing the capability D6 grants, that D17's global coherence was built around, and that D23 cited as the reason extension functions were unnecessary. It would also force every trait a type ever implements to be named in its header.

*Amended (v0.23):* the top-level block stays the general form, but for a type you declare an implement may also be written inside the struct body as `implement Trait { ... }` — sugar for `implement<Ps> Trait for Name<Ps>` with the struct's own type parameters. Nothing above changes: foreign types still take the top-level form, coherence is unchanged, and the header still names only sealed membership. An implement that needs bounds the struct lacks (`implement<T: Display> Display for Pair<T>`) is written at top level. A top-level implement for a struct of the same module that the inline form could express is a lint: a warning with an automatic fix that moves it into the body (editor quick fix, `veles check --fix`).

*Amended (v0.23) — static functions.* A function in a struct body, trait, implement or extend block may be declared `static fun`: it has no receiver and is called on the type — `Point.origin()`, `i64.parse(s)`, `Stack<i64>.of(x)` — and, for a trait, on a type parameter bounded by it: `T.parse(s)` resolves to the implement for the concrete `T` of each stencil. An `extend` block in the prelude may add statics to a built-in generic type, called with the type arguments written: `MutableList<bool>.repeat(false, n)`, `MutableList<Row>.make(n, i => ...)` (v0.25). An implement declares `static` exactly where the trait does. A trait with a static function is not object-safe (D9), and a sealed trait cannot declare one (its methods dispatch on a variant). The prelude declares `Parsable { static fun parse(s: string): Self? }` for the numbers, `bool` and `string`.

*Addendum (v0.29) — static values.* A struct body may declare `static val name[: T] = expr`: a constant in the type's namespace, read as `Type.name` (`Status.notFound`, `http.Status.ok` from outside the module), `public` to export and module-private otherwise, initialised with the module's globals in source order. It is always a `val` — a mutable global belongs at module level, where it is visibly one — and a generic struct cannot have one (a value per instantiation would be a different feature; a `static fun` serves). Motivated by the HTTP module: status codes, methods and content types are *open* sets with well-known members, which is a value type plus named constants, not an enumeration; closed sets are `enum` (D57).

Consequence: inherent methods cannot be added to a type you do not own. Extension functions were declined, so the route is declaring a trait and implementing it. This is narrower than it sounds — Veles owns `string` and the collections, so stdlib types stay method-rich; the ceremony only appears when extending a third-party type. Rust lives this way.

**Addendum (v0.22) — `extend` blocks.** A package may add inherent methods to a type it declares, outside the struct body:

```vs
extend<T> Stack<T> {
  fun depth(): i64 = self.items.len()
}
extend<T: Show> Stack<T> {          // bounded: only for showable elements
  fun render(): string = ...
}
```

The ownership rule is unchanged — `extend` names only a type declared in the same package, and the built-in types (`string`, the numbers, `List`, `Map`, `Set`, `Range`, `Channel`) are declared by the standard library. This is not an extension-function mechanism: what it provides is a home for the methods of the built-in types, which have no struct body to hold them, so that `trim`, `split`, `take`, `chunked` and the rest are written in Veles in the prelude rather than in the compiler. A `MutableList<T>` also has every `extend<T> List<T>` method (D25). Method names must not collide with the struct body, another `extend` in the package, or a compiler built-in; coherence is checked program-wide like D17. `public` on the methods follows M5.

### D24 — Full prelude

`Result`, `Option`, primitives, `List`, `Map`, and the common traits are in scope in every file with no import.

**Version the prelude against language editions from day one.** Adding a name to the prelude later can shadow a user's existing declaration; Rust ties its prelude to editions for exactly this reason, and retrofitting that is far harder than building it in.

### D25 — Collections: `List` / `MutableList` split, reference semantics

**Reference types, not values.** A `List<T>` is a header pointing at a heap buffer; as a value type, copying would share the buffer, which is precisely Go's slice-aliasing footgun where appending to a copy sometimes writes through and sometimes does not. Swift avoids this with copy-on-write, which needs cheap uniqueness checks that ARC provides and D1's tracing GC does not.

**Cost, stated plainly:** because collections are references, `val list` does not prevent mutation. The `List` / `MutableList` split is the mitigation. *(v0.30: `val` no longer claims to prevent mutation of a struct either; what cannot change is declared on the type — D22, D11 amended — and a `MutableList` field is simply the mutable part of it, visible in the declaration.)*

**REVISED by D35 — `List` is genuinely immutable, not a read-only view.** The original design followed Kotlin, where a `List<T>` reference may point at a `MutableList<T>` and a holder of the mutable handle can change it underneath you. That is incompatible with D35's isolation rules: a view over someone else's mutable buffer can never be `Sendable`, which would make `List` useless for the case it most needs to serve.

Therefore `List<T>` owns its buffer, and `mutableList.toList()` **copies**. This costs O(n) where Kotlin gives O(1), and it is the price of the guarantee. `MutableList<T>` is no longer a subtype of `List<T>` — conversion is explicit and copying.

D14 remains deferred; nothing here requires variance.

**Maps are insertion-ordered.** Python's and JavaScript's behaviour, not Go's. Iteration is faster than open addressing; memory is roughly 1.3x because of the separate index array; deletions leave tombstones requiring periodic compaction. Ordering is a guarantee that can never be withdrawn — Go randomizes iteration specifically to prevent dependence on it.

~~**Index assignment.** `map[key] = value` is supported on mutable collections via an assignable-index operator.~~

**Element access is methods only (v0.24); reads are values, writes go through references (v0.27).** Brackets are collection-literal syntax and nothing else; there is no index operator, reading or writing. Lists read with `xs.at(i): T?` (null out of range, a negative index counting from the end), `xs.atOrPanic(i): T` (panics out of range — programmer error, D20, never a thrown error) and `xs.atOrDefault(i, d)`; maps with `m.get(k): V?`, `m.getOrPanic(k)` and `m.getOrDefault(k, d)`. Every read is a *value*: a struct element comes out as a copy, exactly as binding it to a name would copy it, so `xs.atOrPanic(i).n += 1` is the same mistake as `val t = xs.atOrPanic(i); t.n += 1` and both are errors ("a copy of the element"). The mutable kinds write with `xs.set(i, v)` and `m.set(k, v)`, or through a *reference* to the element: `xs.refOrPanic(i): *T` and `m.refOrPanic(k): *V` (panic when absent), `xs.ref(i): (*T)?` and `m.ref(k): (*V)?` (null when absent), and `loop (&x in xs)` / `loop ((k, &v) in m)` (D42). A reference is an ordinary pointer, so everything follows D10/D11 with no special case: `xs.refOrPanic(i).bump()`, `*counts.refOrPanic(x) += 1`, `m.ref(k)?.n += 1`, `val p = xs.refOrPanic(i)` (a `val` holding a pointer is still written through). `ref`/`refOrPanic` exist only on `MutableList`/`MutableMap` — a pointer into an immutable collection could change it. Sets are not addressable: an element is its own hash key. A pointer into a list or map stays valid as memory (D10, the collector keeps the old buffer alive) but points at stale storage once the collection has grown. Rationale: one spelling per operation with the failure mode in its name, and a read that can never be mistaken for a write — the earlier rule that made `xs.atOrPanic(i)` a place in some syntactic positions (v0.24–v0.26) meant two identical-looking expressions with different semantics. `&xs.atOrPanic(i)` is refused with a fix, since it would box a copy. The old forms are reported with fixes (`veles check --fix`).

*Amended (v0.37, D62).* `atOrPanic`, `getOrPanic` and `refOrPanic` are removed. A read is `xs.at(i)` / `m.get(k)`, typed `T` where the checker can see the index is in range (D62 B); a read that can fail only through a bug says why where it is written, `xs.at(i) ?: panic("…")`. A pointer is `xs.ref(i)` / `m.ref(k)`, `*T` under the same proof. The paragraph above keeps the v0.27 wording for the record.

**Literals are bracket-delimited** (Swift's form): `[1, 2, 3]` for lists, `["a": 1]` for maps, `[:]` for an empty map. `{}` was rejected because it already means block, lambda, and struct literal; a fourth meaning would make `{ port: port }` ambiguous between a map and a struct.

A bare `[1, 2, 3]` with no expected type is a `List`. A growable collection therefore needs an annotation — `var xs: MutableList<i32> = []`, not `var xs = []` — or the `mut` prefix on the literal: `var xs = mut [1, 2, 3]` is a `MutableList<i32>` and `var m = mut ["k": 1]` a `MutableMap<string, i32>`. The empty forms `mut []` and `mut [:]` still need an annotation for the element types.

*Amended (v0.25) — Deque and PriorityQueue.* The prelude adds `Deque<T>` (a ring buffer: O(1) at both ends, the FIFO queue and the sliding window) and `PriorityQueue<T>` (a binary heap over `Comparable` or a comparator), constructed by `deque<T>()`, `priorityQueue<T>()` and `priorityQueueBy(compare)`. Both are plain Veles (`std/prelude/collections.vs`) and have the reference semantics of this section, obtained by holding their state behind a GC pointer (D31): a `val` binding can grow them and a callee shares the caller's. `MutableList` serves as the stack. `MutableList<T>.fill(n, x)` and `.make(n, make)` build a list of known size; `swap(i, j)` exchanges two elements.

*Amended (v0.26) — collections compare by content.* `==` on `List`, `Map` and `Set` (and their mutable views, which compare with each other) is element-wise: two lists are equal when they have the same elements in the same order, two maps when they hold the same keys with equal values, two sets when they hold the same elements — insertion order is not part of a map or set. It is defined whenever the elements are (structurally or through `Equatable`), so a struct with a `List` field keeps its structural equality. Hashing follows: an immutable `List`, `Map` or `Set` of hashable elements can be a map key or set element and is found by content; the mutable views cannot be keys, since they can change after they are stored. Handles are still references (assigning shares, `==` compares) — the split between identity and equality that Kotlin and Python make.

### D26 — Trait methods are always callable; ambiguity is an error

No import required to call a trait method. Under D17 there is only ever one implement per (trait, type) pair, so scoping is unnecessary for correctness — it would only serve to disambiguate two traits sharing a method name, which is reported as an error instead.

**Consequence for semver:** adding a method to a trait can introduce an ambiguity in code that compiles today. Together with D17's "new impls are breaking," these are the two breaking-change classes that M7's minimal version selection trusts library authors to handle correctly.

### D27 — Traits have both associated types and generic parameters

Rust's design. The distinction, which should be documented early because reversing it is the standard beginner mistake:

- **Generic parameter** when a type may implement the trait several ways — `From<i32>` and `From<string>` on the same type.
- **Associated type** when there is exactly one natural choice per type — `Iterator.Item`.

**`Iterator` must use an associated type.** Under a generic `Iterator<T>`, a single type could implement `Iterator<i32>` and `Iterator<string>` — legal under D17, since those are distinct (trait, type) pairs — and `for (x in thing)` would have no way to infer `x`. The associated form permits at most one implement per type, so inference always succeeds.

### D28 — Construction: call syntax, named arguments, implicit constructor

```vs
val c = Config(port: 8080, host: "localhost")
val s = Stack<i32>()
```

Every struct gets an implicit constructor from its fields. Fields with declared defaults may be omitted.

*Addendum (v0.28) — construction is by name only, with puns.* A constructor argument is always `field: value`; the one shorthand is a bare identifier that names both a field and a variable in scope, `Hashed(file, size: n)` for `file: file` (Rust's field-init shorthand, JS's `{ file }`), and `file: file` written out is a warning with a fix. Any other bare argument — `Point(3, -4)` — is an error with the fix that names the fields, so a reordered or renamed field can never silently change what a call means. Named arguments to *functions* keep D28's positional-then-named rule; the pun exists only where positional arguments do not.

*Amended (v0.23) — variadic parameters.* The last parameter of a function may be `name: T...`; the call supplies any number of trailing positional arguments (`join("/", "a", "b")`, `sum()`), collected into a `List<T>`, or one list spread with `join("/", parts...)`. Inside the function the parameter is a plain `List<T>`. A variadic parameter has no default, an `extern` function cannot declare one, and an implement declares it exactly as its trait does.

- **Declaring an explicit constructor suppresses the implicit one.** Otherwise invariants could always be bypassed by calling the generated version.
- **The implicit constructor is callable only where every field is visible.** Otherwise a `public` struct with private fields would leak construction. *(v0.30: refined for `private` fields by M5's amendment — a private field without a default is supplied by the call, one with a default is not.)*

*Addendum (v0.30) — a default is a constant; derived fields are `init`'s.* A field default cannot read `self` (error, with the `init` form suggested). Defaults derived from earlier fields — `positions: List<i64> = self.toks.map(t => t.1)`, evaluated in declaration order on a partially built value, with a per-method field-use check for calls on `self` — were implemented and withdrawn the same day in favour of the `init` block below: two mechanisms for one job, and the declaration-order rule was a subtlety the block does not need (definite assignment orders things for you). What survives from that work is the receiver pass's field-use analysis, which `init` relies on.

*Addendum (v0.30) — the `init` block.* A struct body may contain one `init { }` block (Kotlin's `init`): it runs after every construction — the fields bound from the call and the defaults, in order — on the freshly built value, before anyone sees it. It is compiled as a hidden method the constructor calls (named `$init`, so no user code can name, call, hover or complete it). A field *without a default* that the block assigns (`self.f = ...`, found syntactically) is the block's: the constructor call refuses it, and the block must assign it on every path before it ends — definite assignment is tracked as flow facts in the narrowing state, so it joins at `if`/`when`, dies in loops and is discharged by a diverging branch exactly like a smart cast. Until such a field is assigned, reading it, using `self` as a whole, or calling a method whose field-use set includes it is an error (the receiver pass computes, for every method, the receiver fields it uses — directly, through the methods it calls on `self`, through closures over `self`; using `self` as a whole counts as every field); afterwards the block is ordinary method code. `init` cannot suspend, and has no `throws` (a failing construction is a `static fun ... throws`); it has no visibility, one per struct, runs for a sealed variant before the tag is applied. Spelled `init` rather than the proposed `constructor` because it is not the constructor: it cannot take parameters or replace the implicit call by fields, it only runs after it. `init` is contextual (`init { ` at member position), so a field or function named `init` stays legal. Deriving from a `var` field is a warning, since the stored copy would not follow later changes. The spelling is `self.f` rather than a bare `f` so that a default cannot be misread as a global. It composes with M5's constructor rule: a private field with a derived default cannot be overridden from outside, so the derivation is an invariant the type can rely on. Motivated by a parser holding token positions next to its tokens, which otherwise needed a `static fun` for nothing but the derivation.

**Named arguments are language-wide, not a constructor feature.** Consequences:

- **Parameter names become public API.** Renaming one breaks callers. This is the third breaking-change class, alongside D17 (new impls) and D26 (new trait methods).
- **Trait impls must inherit the trait's parameter names**, or a caller would see different names depending on whether the call went through the trait.
- **Named arguments apply only at direct call sites of named functions**, never through function values. If function types carried parameter names, names would become part of the type and complicate assignment.

A parameter has **one name**, Kotlin-style, not a Swift external label plus internal name. Renaming it is therefore always a breaking change.

### D29 — Ranges: `1..100` inclusive, `1..<100` exclusive

Kotlin's current form, not the deprecated `until` infix — Kotlin introduced `..<` specifically to replace it. The two forms pair visually, and no keyword is spent.

Noted tension: index iteration (`0..<arr.len()`) is the more common case and gets the longer form.

### D30 — Elvis operator `?:`

Supplies a default for a `T?`: `counts.get(word) ?: 0`. Pairs with D5's smart casts. Its counterpart for a `Result` is `??`, and `val x = e else ...` binds or leaves (D61).

The safe-call `?.` is included: `a?.b?.c` short-circuits to `null` and the chain's type is `T?`. A receiver that is nullable more than once — `xs.at(i)` on a `List<T?>` is a `T??`, since a missing element and a stored `null` are different answers — is flattened by `?.`: `xs.at(i)?.f` is null when the index is out of range or the element is null, and `T??` values are otherwise kept apart (v0.26).

*Addendum (v0.29) — the or-fail operator `?!`.* `x ?! e` turns absence or failure into failure with `e`: a `T?` becomes a `Result<T, E>`, a `Result<T, E1>` a `Result<T, E2>` with the original error dropped; `e` must be an error type and is evaluated only on that path. It rewrites the failure and nothing more — `try` remains the one place propagation happens, so `try users.get(id) ?! NotFound(id)` and `try parse(s) ?! BadRequest(...)` read "or fail with", and the parser takes `try x ?! e` as `try (x ?! e)` (the other grouping is written with its parentheses). `Result.mapError(f)` in the prelude is the form that keeps the old error in hand. Spelled `?!` rather than `??` because `??` is null-coalescing in C#, JS and Swift and would read as a second `?:`. Motivated by request handlers, where every lookup and parse must become a status code without a `when` per call.

### D31 — Recursive data structures require indirection

Because D12 lays a sealed trait out as an inline tagged union, a variant containing the trait by value has infinite size. Recursive types must go through `*`:

```vs
struct Node<T> : Tree<T> {
  value: T
  left:  *Tree<T>
  right: *Tree<T>
}
```

This is a hard constraint, not a style preference, and it is the first wall every new user will hit when writing a linked list. **The compiler must detect the infinite-size case and emit a message naming the fix**, not a size-computation error.

### D32 — Lambdas: TypeScript arrows

```vs
val doubled = nums.map(x => x * 2)
val total   = nums.fold(0, (acc, x) => acc + x)
val typed   = nums.map((x: i32) => x * 2)
```

Parentheses are optional for a single untyped parameter and required otherwise. There is **no `it` shorthand** — every lambda names its parameters. The discard `_` is a parameter name (v0.25): `make(n, _ => [])` and `(_, x) => x` take an argument they do not use, as `loop (_ in xs)` does.

No trailing-lambda sugar. `=` was chosen over `=>` for expression bodies (§4) specifically to keep `=>` free for this.

### D33 — `=>` is the only arrow

Used for both `when` branches and lambdas, on the grounds that both mean "maps to" and two similar arrows invite confusion.

```vs
val area = when (s) {
  is Shape.Circle => PI * s.radius * s.radius
  is Shape.Rect   => s.w * s.h
}
```

*Known hazard:* a `when` branch returning a lambda reads `cond => x => x * 2`. It parses unambiguously — `=>` is right-associative — but it is hard to read. Accepted for now; a lint is the likely eventual answer.

### D34 — `scope` is a language construct, not a function

```vs
scope {
  async worker(1, jobs, results)
  async worker(2, jobs, results)
}
```

It cannot be an ordinary function taking a callback. D20 specifies that panics unwind to the enclosing task scope, so the compiler must know where scope boundaries are in order to place the unwind target — and D32 removed trailing-lambda sugar, so a callback form would read `scope(() => { ... })`.

Naming was reconsidered and `scope` retained. The argument against, recorded because it may resurface: "scope" already means lexical scope in every language Veles borrows from, so the word carries two meanings. Alternatives weighed were `nursery` (the term of art from Trio's structured-concurrency work, no competing meaning), `taskGroup` (Swift's `withTaskGroup`), and `concurrent`.

### D35 — Multi-threaded executor with Swift-style `Sendable` and isolation

Tasks may run on multiple cores. Data-race freedom is a **compile-time guarantee**, not a runtime hope.

**Why this was necessary rather than optional.** D7 gives multi-word value structs, so a racing read can observe a 16-byte `Point` half-written and see a value that never existed. Go has this exact hole — an interface value is two words, and a racy write can pair the type pointer of one value with the data pointer of another, producing a pointer the runtime then dereferences. Slices and strings have the same shape. Java escapes it only because every field is word-sized and it has no value structs. Accepting races would have meant Veles is not memory-safe under concurrency.

**Why not Rust's `Send`/`Sync`.** Those work because ownership and borrowing tell the compiler when a value is moved or aliased. `Send` is an ownership claim — "the sender no longer touches this" — and Veles has no ownership to establish it with. Swift 6 solved the same problem with ARC and no borrow checker, which is the closest precedent to Veles' situation.

**The rules:**

- `Sendable` is an auto-derived marker: a type is `Sendable` when all its fields are. *(v0.25)* It is a nameable prelude trait usable as a bound (`extend<T: Sendable>`, `fun f<T: Sendable>`) and answered from the type's shape; an `implement Sendable for X` written by hand is an error.
- Anything captured by an `async` call must be `Sendable`, immutable, or an explicitly synchronized wrapper (`Mutex<T>`, `Atomic<T>`).
- A captured `var` cannot be mutated from more than one task; plain mutable references do not cross task boundaries.
- *(v0.28)* **Structured cancellation reaches the body.** When a child of a fail-fast `scope` fails, the scope's own body is abandoned at its next suspension point (a `recv` on a channel the failed child was meant to feed would otherwise wait forever); the scope then joins the surviving children and re-raises. `Channel.closeAfter(n)` closes a channel after `n` further sends, so several producers can end a channel none of them owns. The prelude's `mapConcurrent`/`forEachConcurrent` (`std/prelude/concurrent.vs`) are the worker pool written once over these primitives.
- *(v0.29)* **Cancellation, fully structured.** A task is cancelled when a sibling in its fail-fast scope fails, when the body of its scope leaves early (`return`, `throw`, a failed `try`, a cancellation from further out) while it is still running, or when `task.cancel()` is called on its handle. Cancellation is a *request*: the task runs to its next suspension point and unwinds there, running the `close()` of every `with` it is inside (D43 — this was specified before and is implemented now), and only then counts as finished, so its scope waits for the unwinding. A task blocked in a suspending call is unwound from the innermost call outwards (each suspending call is a task of its own in the bootstrap; the request follows the await chain down and the finishes come back up), so the deepest resource closes first. Leaving a `scope` or `gather` body early therefore cancels and joins its children as a cleanup, like a `with` close, and a body that always returns or throws is `Never`-typed. Cleanup code is shielded (D47): a suspension point inside it — the join of an abandoned scope — takes no cancellation. `withTimeout(ms, f)` in the prelude is `scope` + `async` + `race` over this: it throws `Timeout` after cancelling and joining `f`'s task, so nothing `f` opened is still open when the caller sees the error.
- *(v0.28)* **Sendable functions.** A function type may be marked `sendable fun(A): R ...`; a value of it may cross a task boundary. A named function is sendable; a lambda is sendable exactly when every capture is a `val` whose type is Sendable (nothing it reaches can change under another task — Swift's `@Sendable` closure rule) — and, since v0.30, whose type has no `var` fields (D22): a captured `val` struct with a `var` field is shared with the lambda and could change under it. The property is part of the closure's type, inferred from its captures at the lambda, so `val g = x => x + k` is a `sendable fun`; a sendable function is assignable to the plain function type, never the reverse, and a lambda checked against a `sendable fun` reports the offending capture. This is what lets the prelude's `mapConcurrent` take an ordinary-looking lambda and run it in pool tasks. *Rationale for a type flag rather than a call-site check only:* a handler stored in a struct and invoked from worker tasks (an HTTP router) must carry the promise in its type.

**What structured concurrency buys here.** Because `scope` (D34) guarantees children cannot outlive their parent frame, immutable values captured from the enclosing frame are provably safe to share by reference with no copy. Go's unstructured `go` statement cannot establish this; Veles gets it free from D3.

**Cost, stated plainly.** `T: Sendable` becomes viral across every generic signature that crosses a task boundary, in the same way Rust's `Send` is. Swift 6's rollout of exactly this was notoriously painful because it retroactively invalidated large amounts of ordinary-looking code. Veles has the advantage of doing it from day one.

**Shared mutable state is `Mutex<T>`, not actors — for now.** Two of the three classic lock hazards are already dead here: forgetting to lock doesn't compile under isolation, and **the compiler statically rejects `await` inside a lock scope**, because stackless coroutines make every suspension point known at compile time. That last one is the nastiest coroutine-plus-lock bug — suspend holding a lock, the releasing task can't run, the executor deadlocks — and Go and Rust cannot cleanly prevent it.

Actors are deferred rather than rejected. Unlike `Sendable`, an actor is a new *kind* of type, so adding one later breaks nothing. The trigger to revisit: repeatedly wrapping the same struct in a mutex where every method is lock-work-unlock is an actor asking to exist. Note that under D2 actors would be more ergonomic in Veles than in Swift, where every cross-actor call needs a visible `await`. The cost is a third type kind plus a re-entrancy policy — Swift's actors are re-entrant across suspension points, so state can change mid-method, which is a documented and regularly-cursed footgun.

### D36 — `scope` and `gather` are separate constructs

`scope` is a statement with fail-fast semantics: the first child error cancels its siblings and propagates out. This is Trio's, Kotlin's and Swift's behaviour, and it matches `Promise.all`.

`gather` is an **expression** yielding every child's outcome, matching `Promise.allSettled`:

```vs
val results = gather {
  async fetch(a)
  async fetch(b)
}   // List<Result<Page, FetchError>>
```

Two constructs rather than one construct with a policy flag, because the two have different *types* — fail-fast produces nothing, collect-all must yield results — and a flag that changes a construct's type is hard to infer and harder to read.

`gather` is **heterogeneous and tuple-typed**: children may return different types, and the result is a tuple of `Result`s in launch order.

This would normally require variadic generics — a feature Rust still hasn't shipped and Swift only got in 5.9. It doesn't here, because `gather` is a language construct like `scope`, so the compiler special-cases its typing rather than exposing general variadic generics. Tuples themselves (D37) are a real addition.

### D37 — Tuple types with destructuring bindings

```vs
val (a, b) = gather { ... }
val pair: (i32, string) = (1, "x")
```

Required by D36's heterogeneous `gather`. Tuples destructure positionally in both bindings and `when` patterns (D13).

*Parsing note:* `(a, b)` as a tuple literal and `(a, b) => ...` as a lambda parameter list need lookahead to distinguish. TypeScript has the same situation and handles it.

**Tuple parameters auto-adapt.** A two-parameter lambda is accepted where a one-parameter lambda over a tuple is expected, so `entries().sortedByDescending((k, v) => v)` works without doubled parentheses. This is Scala's tupling conversion. Cost: a special case in the checker, and it can mask genuine arity errors, since a two-argument lambda passed where one argument is expected now silently succeeds.

*Addendum (v0.28) — patterns everywhere a tuple is bound.* Destructuring nests to any depth in `val`/`var`, in loop heads and in lambda parameters: `groups.map(((size, hash), files) => ...)`, `sortedWith((((sa, _), _), ((sb, _), _)) => sa - sb)`. A lambda parameter written as a tuple pattern binds the whole element to a hidden variable and destructures it as the body's first statements; it composes with the tupling conversion above.

*Addendum (v0.25) — destructuring assignment.* `(a, b) = expr` assigns to existing places (variables, fields, `*p` for a `p = xs.ref(i)` element) positionally. The right side is evaluated in full before the first store, so `(a, b) = (b, a)` swaps and `(a, b) = (b, a + b)` steps without a named temporary — Python's and Go's rule. Each place keeps its own mutability check (D11) and its own smart cast after the store (D5). Only plain `=` destructures; a compound `(a, b) += ...` is an error, since element-wise arithmetic on tuples is not defined. Nested tuples do not destructure in assignment, as they do not yet in bindings.

### D38 — `race { }` for the first-ready construct

Completes the concurrency family:

- `scope { }` — all children, fail fast on the first error
- `gather { }` — all children, collect every outcome
- `race { }` — first arm to become ready wins

```vs
race {
  val job = jobs.recv() => process(job)
  val cmd = ctrl.recv() => handle(cmd)
  timer(1.second)       => giveUp()
}
```

`race` is itself the suspension point, so arms carry no `await`.

*Addendum (v0.29).* Exactly one arm runs, so a smart cast made inside an arm (an assignment to a `var`) survives the race only when every arm makes it — the `if`/`else` join rule (D5). A race whose arms all return or throw is `Never`-typed, which is what lets `withTimeout` be written as `race { val r = await t => return try r; sleep(ms) => throw Timeout() }` inside a scope whose early exit cancels `t`.

Named `race` rather than Go's `select` because the three constructs then share a vocabulary — each name says what it does with the set of children — instead of borrowing the third from an unrelated tradition. Recorded against it: `select` is what anyone arriving from Go will look for, and Go's `select` is typically driven in a loop to service channels repeatedly, which "race" slightly undersells.

### D39 — `.` auto-dereferences pointers

`p.field` works whether `p` is a `T` or a `*T`; no explicit dereference operator is needed for field access or method calls. Go's behaviour.

This applies inside `when` as well, so `is Tree.Node(left, right)` destructures a `*Tree<T>` scrutinee without ceremony.

### D40 — Effects are declared wherever dispatch is dynamic

**The problem.** D2 infers suspension by call-graph analysis and D4 infers error types the same way. Both work for direct calls, where the callee's body or materialized interface is visible. Neither works through **trait dispatch**: under D8 a trait call goes through a dictionary, and under D17 any module may add an implement with coherence checked only at link time, so the set of impls a call may reach is not closed at compile time.

Conservatism does not rescue this. Assuming every trait method may suspend means CPS-transforming every caller of every trait method — and with a full prelude (D24) plus `Iterator`, `Ord` and `Display`, that reaches almost everything. `loop (x in items)` calls `Iterator.next`, so every loop in the language would become a suspension point and nearly the whole program would be state-machined. That is a collapse, not a tuning problem.

**The rule.** Trait methods and function types declare their effects; ordinary functions continue to infer them. Declaration follows the return type, matching `throws`:

```vs
trait Fetcher {
  fun fetch(url: string): Bytes suspends throws
}

// function types carry effects too
val handler: fun(Request): Response suspends throws HttpError
```

**Error types on trait methods (revised v0.24).** A trait method says *that* it can fail; which error is either fixed by the trait or left to each implement:

- `throws E` — every implement throws `E` (or a subset); callers see `E` however they reach the method.
- a bare `throws` — the error is **implement-defined**: the trait carries an implicit associated type `Error` (D27), and each implement defines it through its methods: what they declare, or, when they too say only `throws`, what their bodies throw (D45 inference); an implement that cannot fail defines it as `Never`. Generic code sees it as `F.Error` (`fun load<F: Fetcher>(f: F): Bytes throws F.Error`) and, once stenciled, as the implement's exact union. Two throwing methods of one trait share the one `Error`. A trait may still write `type Error` and `throws Self.Error` explicitly — it is the same thing spelled out — and an implement may bind `type Error = E` to pin it.
- A trait with an implement-defined error is not object-safe for now: through a trait object the error would have to be erased to `Error` (a boxed error), which needs a dyn-error type in unions and adapter thunks in the vtable. Declare `throws E` on the trait to use it as an object. Open item.

Associated-type projections are written with a dot, `Self.Error`, `I.Item` (`::` was dropped in v0.24 with the other Rust spellings).

**Defaults.** A trait method neither suspends nor throws unless it says so.

**Sealed traits are exempt.** D12 requires variants in the same module as the trait, so the implement set genuinely is closed and inference works normally. `Option`, `Result` and every user-defined sealed type remain fully inferred.

**What this costs.** This is a partial retreat from D2's no-coloring goal, confined to trait declarations and function types — ordinary function definitions never carry an effect marker, so libraries still don't split into sync and async ecosystems. But a suspending iterator cannot share a trait with a non-suspending one, so Veles will need the `Iterator` / `AsyncIterator` split that Rust has as `Iterator` / `Stream` and Kotlin has as `Iterator` / `Flow`. That is the state of the art, not a Veles-specific failure, but it should be recorded as the price of D17's open impls.

### D41 — Collections are concrete types; abstraction goes through traits

`List<T>`, `MutableList<T>`, `Map<K, V>` and `MutableMap<K, V>` are **concrete types**, not traits.

This is forced by D9. An open trait used as a variable type is a boxed trait object, so declaring `val xs: List<i32>` against a `List` *trait* would allocate and dispatch dynamically on every operation. Kotlin pays exactly this cost because its `List` is an interface.

Abstraction moves to traits taken as generic bounds — `fun summarize<I: Iterable>(xs: I)` — which D8 stencils with no boxing and no dynamic dispatch.

**Consequence:** a user-defined collection cannot be passed where a `List<T>` is expected. It implements `Iterable` instead, and functions that want to accept it must be generic rather than taking `List<T>` concretely. Stdlib signatures should therefore prefer `Iterable` bounds over concrete `List` wherever they only need to iterate.

`List<T>` and `MutableList<T>` are unrelated types (D25 removed the subtyping), converted explicitly by `toList()` and `toMutable()`, both of which copy. `List<T>` is `Sendable` when `T` is; `MutableList<T>` never is (D35).

*Open:* whether literal syntax (`[1, 2]`, `["a": 1]`) is wired to fixed stdlib types or to an opt-in trait, Swift's `ExpressibleByArrayLiteral` style, so user types can use it too.

### D42 — `loop (x in c)` desugars through `Iterable`

```vs
trait Iterable {
  type Iter: Iterator;
  fun iterator(): Iter;
}

trait Iterator {
  type Item;
  mut fun next(): Item?;
}
```

`loop (x in c) { body }` desugars to:

```vs
var __it = c.iterator()
loop {
  val __v = __it.next()
  if (__v == null) break
  val x = __v
  body
}
```

`__it` is `var` because `next` is a `mut fun` (D22). The loop variable's type is the projection `Iter.Item`, using the associated-type projection D40 introduced with `Self.Error`.

**Termination depends on D5 nesting.** If `Item` is itself `string?`, `next()` returns `string??` and the outer `null` means end-of-sequence. Under Kotlin's collapsing rule this would be ambiguous with a sequence containing nulls — a concrete case where D5's nesting decision is load-bearing rather than theoretical.

**One loop syntax covers sync and async.** An `AsyncIterable` whose `next` is declared `suspends` (D40) drives the identical `loop (x in c)`; the compiler selects by which trait the receiver implements, and the enclosing function is inferred suspending by D2. Because ordinary functions carry no effect markers, nothing at the call site changes. Rust requires `for await` or `while let`, and Kotlin requires `.collect { }`; Veles requires neither. Implementing both traits on one type is an ambiguity error.

*Open:* loop labels for `break` and `continue` targeting an outer loop.

*Addendum (v0.27) — iterating by reference.* `loop (&x in xs)` binds `x` to a pointer to each element of a `MutableList` (`x: *T`), and `loop ((k, &v) in m)` binds `v` to a pointer to each value of a `MutableMap` (`v: *V`), so the body changes the elements where they live: `loop (&c in counters) c.bump()`, `loop (&n in nums) *n += 1`. The plain forms copy (a value struct element is a copy, D25), which is why they cannot update anything — a `loop (x in xs) { x.n += 1 }` is an error, the variable being a `val`. `&` is refused on a read-only `List`/`Map`, on a set (elements are their own hash keys), on a map key, on a range and on an iterator. The list loop reads the element by index each time; the map loop snapshots the keys and looks each one up, so an entry the body removes is skipped and one it adds is not visited. This is D25.s `ref` spelled as a loop head: the same `&` in both places means the same thing.

### D43 — `with` blocks for scoped resource cleanup

Veles has a GC and therefore no destructors, and no `defer`. Before this, nothing in the language released a file descriptor, a socket, a lock, or a C allocation.

```vs
with (f = File.open(path)) {
  process(f)
}   // f.close() runs on exit, on panic, and on cancellation
```

Named `with` rather than `use`, because `use` is already the import keyword and the two are unrelated.

- Resources must implement `Closeable`.
- Multiple bindings close in reverse order: `with (a = ..., b = ...) { }`.
- Cleanup runs on **every** exit path. Under D20 cancellation is a panic delivered at a suspension point, so a `with` block containing a suspension point still releases when its task is cancelled.
- If `close()` fails and the body also failed, the body's error wins and the close failure is attached rather than replacing it — Kotlin's suppression rule.
- *(v0.29)* `with` is an expression: its value is the body's, and the resources close before the value is used (`val text = with (f = open(p)) { f.readAll() }`), so an "open, use, close" function has no `return` in its middle. In statement position the body is a plain block, as before.

**This raises the stakes on §6.1.** The panic mechanism is still undecided, and `with` cleanup has to run during unwinding: under a hidden error-return path it is an ordinary code path, under DWARF it needs landing pads. Either works, but the choice is no longer purely internal.

`Mutex.withLock` predates this and takes a lambda instead. Whether it is rewritten as a `with` resource is an open stdlib question.

### D44 — `unsafe` blocks, and two kinds of pointer

D35 promises compile-time data-race freedom. FFI, raw pointers, and pointer arithmetic all sit outside that promise, and until now nothing distinguished them — which made the guarantee untrue in any program touching C, with no way to identify which programs those were.

**The marker only works if pointers split.** A `*T` obtained from `&x` is GC-managed: D10 guarantees the address is stable and I3's scanning keeps the target reachable. A raw C pointer has neither property. Spelling both `*T` would leave `unsafe` guarding nothing.

- **`*T`** — GC-managed, safe to dereference anywhere. Unchanged from D10.
- **Raw pointers** — unmanaged, dereferenceable only inside `unsafe`. Rust's `*const T` / `*mut T` to Veles' `&T`.

```vs
unsafe {
  val n = strlen(ptr)
}
```

- `unsafe fun` marks a function whose contract the compiler cannot check; callable only from an `unsafe` context.
- All `extern "C"` calls are implicitly unsafe.
- D35's claim is restated precisely: **Veles is data-race free in safe code.**

Raw pointer syntax and semantics are settled in D50.

### D45 — Inferred error types are unions, confined to error position

D4's inference silently assumed every callee throws the same type. It doesn't survive the most ordinary function in any program:

```vs
fun loadConfig(path: string): Config throws {
  val text = try fs.readText(path)     // IoError
  val port = try parsePort(text)       // ParseError
  ...
}
```

**The compiler infers the union.** `loadConfig` throws `IoError | ParseError`. This is Zig's error-set inference, and it fits D4's "typed but inferrable" phrasing exactly — inference produces a type rather than giving up.

**Unions exist in error position only.** Not a general type former. Otherwise Veles would have two mechanisms for "one of several types" — nominal, closed, declared sealed traits (D12) versus structural, anonymous, inferred unions — competing for the same job. Zig confines error sets for the same reason.

Rules:

- Unions flatten: `(A | B) | C` is `A | B | C`.
- Width subtyping applies in error position: `IoError` is usable where `IoError | ParseError` is expected. This does **not** reopen D14 — it is not generic variance.
- `when` matches over union members with normal exhaustiveness checking.
- An associated error type on a trait method (D40) may be a union, but it is declared rather than inferred.
- **Named error sets.** `error PortErrors = ParseError | RangeError` names a union (Zig's named error sets). The name is transparent — it flattens into any union it appears in, `when` sees the members — and it is not a nominal type: it cannot be constructed, bound as a value type, or tested with `is`. It may appear only where a union may: after `throws`, inside another error set, or as the type of a field of an `error`. Cycles between sets are an error. Sets are resolved once, after collection, so an error set declared in another file or module works and its members are checked for being errors at that point.
- **A cause field.** The one place a union appears outside `throws` is a field of an `error` (`error ConfigError { key: string, cause: PortErrors }`). This is how context wraps a cause (Go's `%w`, Rust's `source()`); `self.cause.message()` dispatches on the union like any other. Fields of plain structs may not hold unions.

Rejected: Rust-style `From` conversion at `try`, which requires the caller to declare its error type and abandons inference; and boxing into an `Error` trait object, which discards the typing D4 exists to preserve and allocates on every error under D9.

**Cost — this is the fourth breaking-change class.** Adding a failure mode to a function changes its inferred union and breaks downstream exhaustiveness. It joins D17 (new impls), D26 (new trait methods) and D28 (parameter renames) as ways a library breaks consumers without anyone editing a signature. Since M7's minimal version selection trusts authors to honour semver, all four must be documented prominently.

**Mitigation:** declare `throws E` explicitly on public API boundaries to pin the set. This should be recommended practice from day one, not discovered later — deep call chains otherwise accumulate large unions, which is the complaint Zig users raise.

### D46 — Collection operations come in eager and lazy forms

Eager methods on collections (`list.map(...)` returns a `List`), and lazy adapters on iterators (`list.iter().map(...)` materializes only at `toList()`). Kotlin's split between collections and sequences.

**Lazy adapters are `Iterator` methods, so "sequence" is not a third concept.** `AsyncIterator` (D42) inherits every adapter for free — a lazy `map` over an async source needs nothing extra.

Cost: every operation exists twice, so the collection area of the stdlib roughly doubles, and there is a performance cliff users have to learn. D23's absence of extension functions means both sets must be declared up front by whoever owns the types; neither can be added later from outside.

*Amended (v0.24) — a throwing function in an eager operation.* `xs.map(x => try parse(x))` is `Result<List<U>, E>`: the inlined loop stops at the first `Err` and the operation yields it, dropping the partial result; otherwise `Ok(list)`. The same holds for `filter`, `fold`, `forEach`, `any`/`all`, `find`, `count`, and the map operations `forEach`/`mapValues`/`filter`. Callers write `try xs.map(...)` like any fallible call; nothing panics. `sortedBy` and `getOrPut` run their function where no loop can be left and refuse a throwing one. Lazy adapters do not take throwing functions (`next()` would have to become fallible).

*Amended (v0.28) — the same rule for functions written in Veles.* A higher-order function declares `f: fun(T): R throws E` with `E` one of its own type parameters and itself `throws E`; a lambda argument then has its error type *inferred* into `E` — a throwing lambda binds `E` to what it throws, a non-throwing one binds `E` to nothing, and an instance whose `E` is nothing is an ordinary non-throwing function (`throws Never` is erased at instantiation, and a `try` in the generic body on such a call is the identity). So `xs.mapConcurrent(x => try parse(x))` throws `E` exactly as the built-in `map` does, without the compiler knowing anything about `mapConcurrent`. Rust spells this `F: Fn(T) -> Result<R, E>`; Veles keeps the one error channel.

*Addendum (v0.29) — a type parameter in an error union.* Such a function may add errors of its own: `withTimeout(ms, f: fun(): R throws E): R throws E | Timeout`. On instantiation the union is normalised — a member bound to `Never` vanishes, so a non-throwing lambda gives `throws Timeout` and a throwing one `throws IoError | Timeout` — and the erasure reaches one step further than a call: in an instance where `E` is `Never`, a value the template typed as `Result<R, E>` (an awaited `Task<Result<R, E>>`) is already the payload, and a `try` on it is the identity.

### D47 — Cleanup is implicitly non-cancellable

D43 says `with` releases on cancellation; D20 says cancellation is a panic delivered at the next suspension point. Those combine badly: a `close()` that suspends — flushing a socket, say — would be cancelled immediately and the resource would leak. Kotlin hit this and added `NonCancellable`.

**Cleanup runs shielded from cancellation.** No explicit construct; `close()` bodies are implicitly protected, so a suspending cleanup completes.

*Hazard, to be documented:* a cleanup that hangs cannot be interrupted, and the enclosing scope hangs with it. Cleanup must be bounded. Kotlin's `NonCancellable` carries the identical warning.

### D48 — Alternative orderings use comparator lambdas

D17 permits one `Ord` implement per type, so descending and case-insensitive sorts need another route. That route is comparator and key-extractor lambdas — `sort(by: ...)`, `sortBy(...)` — not a second `Ord` implement.

This **formally drops named-impls-as-values**, which had been recorded as still-useful since D17. Nothing else in the design now needs it.

*Addendum (v0.28) — tuples are ordered.* A tuple whose elements are all ordered (numbers, strings, `Comparable` types, such tuples) is `Comparable`, element by element: `(1, "b") < (2, "a")`. The comparison is a function the compiler synthesizes per tuple type and reaches through the same path as a hand-written `compareTo` (`<`, `sorted`, `min`, a `T: Comparable` bound, `a.compareTo(b)`). Consequence: a multi-key sort is `sortedBy(e => (-e.size, e.name))` — Python's key tuple — with no comparator-combinator API; a comparator lambda remains for the cases a key cannot express (a descending string). Equality and hashing of tuples were already structural.

*Amended (v0.32).* Every comparison — `compareTo`, the synthesized tuple one, a comparator lambda — returns the prelude enum `Ordering` (`Less = -1`, `Equal`, `Greater`; D57), not an `i64`. A comparator that computed `a - b` is written `a.compareTo(b)`.

### D49 — Panics unwind via DWARF tables, with a separate coroutine-frame chain

Native frames use LLVM `invoke` and landing pads. Suspended coroutine frames (D2) are heap objects the DWARF unwinder cannot see, so they carry an explicit parent chain that is walked separately.

**The two meet at the resume boundary.** A task resumes through a native call from the executor, so a panic raised inside it walks coroutine frames up to the resume point and then converts into a native unwind into the executor. Both directions need the handoff.

**What this buys:** genuine zero cost on the happy path. No branch at every call site, no panic-freedom analysis, simpler codegen for the common case — the reason C++ and Rust use table-based unwinding.

**What it costs:**

- Two unwinders to build and keep consistent.
- Two platform implementations: `.eh_frame` on Unix, SEH on Windows.
- `.eh_frame` is often larger than the branches it replaces.
- Slow when a panic actually fires — table lookup, microseconds. Acceptable only because panics are rare, but note that **cancellation is a panic** (D20), so cancellation is not cheap.
- **No wasm story.** WebAssembly has no DWARF unwinding; its exception-handling proposal is separate and newer. The native half must be replaced wholesale when the wasm target arrives, which is the one place this choice works against the rationale that motivated D2.

After the GC, this is the largest single piece of runtime engineering in the compiler.

Rejected: a hidden error-return path, which works identically in native and coroutine frames because it is just code, needs no platform-specific machinery, and ports to wasm unchanged — at the cost of a predictable branch per call site and roughly 5–15% code size, much of it recoverable by panic-freedom analysis on leaf functions.

*Addendum (v0.29) — what the bootstrap does today.* The bootstrap has no unwinder yet: a panic is a `longjmp` back into the executor, so nothing between the panic site and the resume point runs. What D43 promised — `with` closes on panic — is kept another way: every `with` entry (and every `scope` body) registers its cleanup with the task on a dynamic stack and unregisters it on each exit the compiler emits; `veles_task_panic` runs whatever is still registered, innermost first, on the live stack before the jump, so a plain function's resource (whose frame the jump would destroy) is closed while it still exists. A cleanup that panics does not stop the unwinding, and the first message is the one reported. A panicking scope body cancels its children (they unwind on their own; the owner is gone). This is the D20 "cheap option" restricted to cleanups: a push/pop pair per `with`, no landing pads, and it stays correct when the DWARF unwinder replaces the jump. Not covered, as before: a panic in a `close()` that suspends — cleanups do not suspend today.

### D50 — Raw pointers: `*raw T`

```vs
extern "C" {
  fun strlen(s: *raw u8): usize;
}
```

Pairs visually with `*T`, costs one new keyword, and is unmistakable at a glance.

**Nullability follows D5's precedence.** `?` binds tighter than `*`, so a nullable raw pointer is `(*raw T)?` and bare `*raw T?` is a raw pointer to a nullable `T`. Same rule as GC pointers, no special case.

**No mutability variants.** D11 made mutability a property of bindings rather than types, and reintroducing it only for raw pointers would be inconsistent. The cost is that a C header binder discards `const` information — which is advisory in C anyway.

**Arithmetic uses operators**, inside `unsafe` only:

- `p + n` and `p - n` advance by `sizeof(T)`, **element-scaled as in C**. Byte-stepping requires `*raw u8`.
- `p - q` yields an element count.
- Comparison operators are defined.
- **No overflow check.** D21 makes `+` trap for integers, which cannot apply here — a bounds-checked raw pointer is a contradiction in terms.

**Provenance rule.** Raw pointers arise only from C or from explicit unmanaged allocation. **`&x` always yields a GC-managed `*T`**, never a raw pointer. If `&x` could produce one, a program could hold an unmanaged reference to a GC object with no registered handle — exactly the situation the FFI handle mechanism exists to prevent.

### D51 — Attributes

```vs
@test
fun userQueryReturnsRows() { ... }

@deprecated("use parseConfig instead")
fun loadConfig(path: string): Config throws { ... }
```

Placed on the line above a declaration; may take typed arguments.

**Attributes can only express what the compiler implements.** There is no derivation mechanism (Rust's proc macros, Kotlin's annotation processors) and no runtime reflection, so `@serialize(name: "user_id")` would do nothing — nothing could read it. **User-definable attributes are deferred** until there is a derivation story, which is a substantial feature in its own right. This does not solve serialization and should not be described as though it does.

Initial compiler-known set:

- `@test` — marks a test function (see `veles-testing.md`)
- `@deprecated(reason)` — warning at use sites
- `@inline` / `@noinline` — hints
- `@mustUse` — unused-result lint; this is what the `Channel.send()` returning `Result` decision needed
- `@specialize` — makes §6's deferred specialization question expressible when wanted (D15)

*Amended (v0.33, D58).* The derivation story exists, and with it the first attributes that carry data the compiler reads on the author's behalf — all of them serve the synthesized `Codable` impls and nothing else: `@key(...)`, `@skip`, `@required` on a field, `@key` on an enum member or sealed variant, `@tag(...)` on a sealed trait. They are still compiler-known; "user-definable attributes are deferred" stands.

`extern struct` (FFI §3) is **retained as a keyword** rather than folded into a layout attribute.

*Noted pressure point:* `extern struct` does not extend. Packed layout, explicit alignment, and transparent single-field wrappers all come up in FFI and wire-format work, and there is currently nowhere to express them without inventing further keywords or revisiting this.

### D52 — A panic surfaces as a `Result` error at a `gather` boundary

D20 said a panic unwinds to the enclosing task scope. It never said anyone could observe it, which left a test harness — or any supervisor — with no way to report a failure rather than die with it.

**`Result<T, E | Panic>`.** `Panic` is simply another member of D45's inferred error union; no new outcome type is needed.

**`scope` re-raises; `gather` captures.** A panic propagating out of a fail-fast `scope` remains a panic in the parent. Only `gather` converts it to a value.

This keeps D4's line intact: panics remain non-recoverable *within* a task, and become observable only at a boundary where they were already terminating that task. It is not `catch` in arbitrary positions, and no `catch_unwind` equivalent is needed.

*Open:* whether a cancellation — which is a panic under D20 — is distinguishable from a genuine panic in the captured value. A supervisor almost certainly wants to tell them apart.

### D53 — Traits may carry default method bodies

A trait method may supply a body; impls inherit it and may override.

**This was silently required by D46.** Lazy adapters — `map`, `filter`, `take` — live on the `Iterator` trait. Without default bodies, every `Iterator` implementation would have to write its own `map`, which is absurd. The same applies to `Iterable` conveniences and `Ord`'s derived comparisons. Go's interfaces lack defaults, which is why Go uses free functions instead; that route is closed here because D23 declined extension functions.

**Effects on a default body are declared on the trait (D40).** A default `map` declared non-suspending is fixed as non-suspending, and an override cannot quietly begin to suspend. This falls out of D40 rather than needing a separate rule.

Default bodies may call the trait's other methods — that is the point, since `map` is written in terms of `next`.

**`override` is required when overriding a default body**, and only then. A method with no default needs no keyword.

*Consequence — this is the fifth breaking-change class.* Adding a default body to an existing trait method breaks every implement that already implements it, since all of them would suddenly require `override`. It joins D17 (new impls), D26 (new trait methods), D28 (parameter renames) and D45 (widened error unions). Kotlin avoids this by requiring `override` on every interface method implementation; Veles trades that for less ceremony and one more thing library authors must watch when bumping a major version.

### D54 — Values crossing a task boundary must be `Sendable`, including error types

D52 has a child's `Result<T, E | Panic>` cross into `gather`, and D35 requires anything crossing a task boundary to be `Sendable`. **Both `T` and `E` must therefore be `Sendable`**, and so must `Panic`.

This was never stated and it constrains error design directly: an error type carrying a `MutableList` payload (D25 — mutable collections are never `Sendable`) silently becomes unusable in any concurrent code. Because D45 infers error *unions*, a single non-`Sendable` component poisons the whole union for every caller above it.

Consequences:

- Error payloads should be immutable data — strings, numbers, `List`, value structs. Not mutable collections, not open handles.
- **Every stdlib error type must be `Sendable`.** This is a design constraint on the stdlib, not a guideline.
- Applies to `scope`, `gather`, `race` and `Channel<T>` uniformly.

---

### D55 — Type aliases: a name for a type, never a new type (v0.28)

```vs
type Index = i64
type Key = (Index, u64)
type Handler = fun(Request): Response throws HttpError
type StrMap<V> = Map<string, V>
public type Point = geo.Point
```

`type Name<T> = Type` at module level declares another name for a type. The alias is **transparent**: `Key` and `(Index, u64)` are the same type everywhere — assignable both ways, one instantiation of every generic, no conversion. A distinct type with the same representation is a one-field struct, as before; `type` never provides safety, only a name. What the alias does own is its *spelling*: diagnostics and hover print `Key` where the source said `Key`, its definition as written, and the full expansion when that differs (structural types carry the display name; a named type — struct, sealed, trait — keeps its own name, so `type Point = geo.Point` reads `Point`). The same mechanism names `error Set = A | B` in messages.

Rules. Module level only; `public` exports it, and a `public` alias of a private type is allowed — it *is* the facade (Go, TS). Parameters take no bounds (state them where the alias is used). No unions: `error` names an error set, `sealed trait` a closed family of types, and `type` never spells `A | B`. Not recursive: `type Json = Map<string, Json>` is an error; a recursive type is a sealed trait or a struct (which also gives its cases names). Everything else sees through the alias: `implement`/`extend` on an alias follow the underlying type's ownership rule (D23), an alias of a struct constructs (`Point(x: 1.0, y: 2.0)`), calls statics (`Point.origin()`) and matches (`is Point`); a generic alias in value position takes its arguments (`Pair<i64>(...)`).

Rejected: aliases in std for numbers (`int = i64`) — two spellings for one type is the import problem again; TS-style type-level computation (`keyof`, mapped and conditional types) — the Veles answer to "compute a type from a type" is an associated type on a trait (`Iterator.Item`).

### D56 — Networking: non-blocking sockets under the task executor (v0.29)

```vs
with (listener = try net.listen(host: "", port: 8080)) {
  scope {
    loop {
      val conn = try listener.accept()
      async handle(conn)
    }
  }
}
```

`std/net` is TCP: `listen`/`connect`, a `Listener` with `accept()` and `port()`, a `Conn` with `read`, `readExact`, `readLine`, `write`, `writeText`, `shutdownWrite`, `peer()`; both are `Closeable` and `Sendable`, and every failing call throws `IoError` with the address in `path`. Every call that has to wait is a `suspends` function, and a function that calls one becomes one — the waiting is in the types, not in a callback or a colour of its own.

**How.** Sockets are non-blocking. A call that would block parks the task on the descriptor (`await ioWait(fd, write)`, a compiler intrinsic that exists only inside the standard library) and returns to the executor, which polls every parked descriptor together with its timers — `poll` on POSIX, `WSAPoll` on Windows — and resumes the task when the socket is ready; the call is then retried. Nothing blocks a thread, so one thread holds any number of idle connections, and a task blocked on a socket is cancelled like one blocked on a timer (D34 v0.29: it unwinds, its `with` closes the connection). The design is Go's netpoller and Node's event loop with the suspension explicit; it stays valid when the executor gains threads (D35), since the parked task is just a wait-list entry.

**The `Conn`.** A connection is a handle plus a read-ahead buffer (`readLine` needs one). The buffer sits behind a `Mutex` so that `Conn` is Sendable — handing a connection to `async handle(conn)` or to `withTimeout(ms, () => try conn.readLine())` is the first thing a server does — with every access a short non-suspending section; the socket calls happen outside it. Two tasks reading one connection interleave bytes, as they do everywhere; the type system rules out the data race, not the protocol error.

**Why not threads and blocking calls.** A blocking `accept`/`recv` under the single-threaded executor would stall every other task; a thread per connection is the model D2/D35 exist to avoid. The intrinsic is deliberately not user-facing: a program wanting another kind of descriptor wait (a pipe, a signal) asks for a standard-library binding, which keeps every wait the executor knows about in one place.

Not in this decision: TLS (a binding to a system library, later), UDP, name resolution beyond `getaddrinfo` at connect time.

**The HTTP layer (v0.29, `std/http`)** is written in Veles on `net`, and its one design question was what a handler's error means. A handler stored in a router needs a closed type, Veles has no "any error" (D45), and forcing every handler to convert every `IoError` into a status by hand is the Go shape the language exists to avoid. The answer: the *stored* type is `Handler = sendable fun(Request): Response suspends` (no throws), and registration is error-polymorphic — `app.get<E>(pattern, h: fun(Request): Response suspends throws E | Fail)` — with one rule applied at that point: `Fail(status, text)` answers with its status, anything else answers 500 and is logged. Erasure happens in exactly one documented place; `try` stays free inside handlers; `?!` says which status a failure deserves. This needed a type parameter *inside* an error union to unify (`E` binds to what the lambda throws beyond `Fail`, `Never` when nothing; a lambda checked against `throws E | Fail` may throw `Fail` regardless). Panics in handlers are isolated per request by a `gather` around the call — no new construct, the panic is a value at that boundary (D52) — and the `with` cleanups a handler held have run by then (D49 addendum). Rejected: a `supervise` scope construct (Erlang/Kotlin `SupervisorScope`), noted as a possibility if a second program needs "children fail independently"; an open `Error` type, again.

### D57 — Enums: a closed set of named values of one integer type (v0.32)

```vs
enum Ordering { Less = -1, Equal, Greater }      // i64 unless a base is written
enum Phase : u8 { Red = 1, Amber, Green = 10 }   // Amber is 2

when (guess.compareTo(secret)) {
  Ordering.Less    => io.println("Too small!")
  Ordering.Greater => io.println("Too big!")
  Ordering.Equal   => io.println("You win!")
}
```

D12 rejected `enum` as *sugar for sum types*, and that stands: alternatives that carry data are a sealed trait. What D12 did not cover is the closed set of *values* — a phase, a direction, the result of a comparison — which the language had been spelling as bare integers (the guessing game matched `-1`, `0`, `1`) or as a struct plus `static val` constants (D23 addendum, an *open* set with well-known members). An `enum` is the closed one: every value has a name, the compiler knows the whole list, and `when` over it is exhaustive by member (D13's rule, with the same `else` lints).

**Representation.** A member is a number of one integer type: `i64` unless the declaration says `: u8` and the like (integers only — a value must be able to count). An explicit value is an integer literal; an implicit one is the previous member's plus one, and the first counts from 0. Two members cannot share a value, and a value must fit the type — both errors at the declaration. At run time an enum value *is* its number and nothing more: no allocation, no tag, structural equality and hashing are the integer's. The names live in the compiler, which spells them out where they are needed.

**What every enum has, with nothing to write.** `E.Member` names a value; `x.value` reads its number; `x.toString()` (and interpolation) is the member's name as declared; `E.values()` is every member in declaration order; `E.fromValue(n)` and `E.parse(s)` are `E?`, the inverses of `.value` and `toString`. An enum is `Comparable` by its numbers (`<`, `sorted()`, `min()`, a `T: Comparable` bound, `x.compareTo(y)`) and `Hashable` (a map key, a set element). It is `Sendable`.

**Comparison with the base type, no conversion.** `phase == 2` and `ordering < 0` compare the number — a comparison operator accepts an enum on one side and a value of its base type on the other. Nothing converts: an integer is not an `E` (the error names `E.fromValue(n)`), an `E` is not an integer (the error names `.value`), and `as` does not bridge them. Two different enums do not compare at all. A `when` arm over an enum names a member, never a number.

**What an enum is not.** It has no methods, fields, `implement` or `extend` blocks (the compiler refuses them: it already compares, hashes, orders and prints by itself) and cannot be generic. Behaviour that needs one is a function that takes it; a set of alternatives that grows data is a sealed trait, which is the point at which an enum is outgrown. Rejected: string-valued enums (the base must count; a string-keyed closed set is a map or a sealed trait), methods on enums (Kotlin's `enum class` — a type with behaviour is a struct, and the `static val` form already covers named constants of a struct), and implicit conversion to the integer (C's rule; it is what makes `switch` on a C enum silently accept a stray `int`).

**Consequence: `compareTo` returns `Ordering`.** The prelude declares `enum Ordering { Less = -1, Equal, Greater }` and `Comparable.compareTo(other: Self): Ordering` (it was `i64`), so the result can be matched by name; because the values are -1, 0 and 1 and an enum compares with its base type, `a.compareTo(b) < 0` still reads and still holds. Comparator lambdas (`sortedWith`, `minWith`, `maxWith`, `priorityQueueBy`, D48) return `Ordering` for the same reason; a comparator that computed `a - b` becomes `a.compareTo(b)`. The synthesized tuple comparison (D48 addendum) returns `Ordering` too.

### D58 — Derivation: an empty `implement` asks the compiler to write the body (v0.33)

```vs
struct User {
  @key("user_id") id: i64
  name:  string
  email: string?                 // null on the wire, or absent: both read as null
  role:  Role = Role.Member      // absent → the default
  @skip passwordHash: string = ""
  implement Codable                   // encode and decode, synthesized from the fields
}

sealed trait Shape
struct Circle : Shape { r: f64 }
struct Rect   : Shape { w: f64; h: f64 }
implement Codable for Shape           // {"type": "Circle", "r": 1.0}

val text = json.encode(user)
val back = try json.decode<User>(text)         // DecodeError lists every problem, with paths
```

**The problem.** A server maps its types to the outside world constantly — JSON in and out, rows from a database, settings from the environment — and D51 had closed the door on it: no reflection, no derivation, so `@serialize` "would do nothing". Writing `toJson` and `fromJson` by hand for every type is the thing no one will do, and the language already synthesizes from shape wherever it can (structural `==` and hashing, `Sendable`, tuple ordering, everything an enum has). This decision is the derivation story D51 asked for, scoped to what the compiler implements.

**Mechanism: an empty `implement` opts in.** `implement Codable for User { }` — or, in a struct body, just `implement Codable`, the braces being optional when the body is empty — asks the compiler to synthesize the body from the type's fields. A method the author writes is kept and the rest is synthesized, so one direction can be hand-written; a foreign type is covered the same way at top level (`implement Codable for geo.Point`), one implement per pair program-wide as D17 already requires. On a generic struct the empty implement is read with the bounds the derive needs (`implement Codable` in `Page<T>` is `implement<T: Codable> Codable for Page<T>`; a parameter that appears only in a `@skip` field gets no bound), and hover shows the header and the body that were written. The formatter drops empty braces.

*Why declared, when `==` is not.* The line is between a **property of the value** — equality, hashing, sendability, ordering of tuples, which hold because of what the value is — and a **contract with the outside world** — a wire format, which others depend on and which a renamed field silently breaks. The first kind is implicit; the second is declared, in one line, where a reader can see that the type is a DTO. Rejected: implicit-from-shape (every type a wire format by default: a `passwordHash` ships unless someone remembers `@skip`, a rename is a silent protocol change); Go's exported-field rule (couples API visibility to wire shape — "why is my field missing" is that language's most-asked serialization question); a `@derive(Codable)` attribute (a second spelling for "give me an implement", no partial override, no foreign types); user-definable derivation (compile-time code execution, hygiene, tooling — a later phase, to be built on the shape description this decision makes the compiler compute).

**Target: one derive, every format.** The synthesized traits are format-agnostic — Swift's `Codable`, serde's `Serialize`/`Deserialize`:

```vs
trait Encodable { fun encode(to: Encoder) throws EncodeError }
trait Decodable { static fun decode(from: Decoder): Self throws DecodeError }
trait Codable : Encodable + Decodable { }   // a supertrait pair (spelled like a bound): the one line a DTO writes

trait Encoder {
  fun format(): string                        // "json", "db", "env": what @key(json: ...) selects on
  fun beginObject() throws EncodeError;  fun key(name: string) throws EncodeError;  fun endObject() throws EncodeError
  fun beginList()   throws EncodeError;  fun endList() throws EncodeError
  fun i64(v: i64) throws EncodeError          // and u64, f64, bool, string, bytes, null
}
trait Decoder {
  fun format(): string
  fun beginObject() throws DecodeError;  fun nextKey(): string? throws DecodeError;  fun endObject() throws DecodeError
  fun beginList()   throws DecodeError;  fun hasNext(): bool throws DecodeError;    fun endList() throws DecodeError
  fun i64(): i64 throws DecodeError           // and the other primitives
  fun isNull(): bool throws DecodeError;  fun skip() throws DecodeError;  fun path(): string
}
```

A flat event stream: the derive writes `key("id")` and then the value encodes *itself* (`self.id.encode(to)`), so the traits carry no generic method (a boxed trait object could not, D9) and no value is boxed on the way out; a format is one implementation of each trait, and `std/json`, a row decoder and an environment decoder all drive the same synthesized code. The encoder is a **trait object**, one virtual call per primitive — `Encodable` stays object-safe and the derive stays simple; a program that links one JSON encoder devirtualizes later. The error types are fixed, as D40 requires of a trait used as an object. Rejected: a JSON-specific `toJson()`/`fromJson()` (a derive per format, a tree allocation per encode); Swift's container objects (`field<T>` is a generic method, impossible on a trait object, and its non-generic form boxes every value); a format-neutral `Value` tree as the intermediate (an allocation per field); a generic `fun encode<E: Encoder>(to: E)` (direct calls, but `Encodable` is no longer an object and every format stencils the world).

**`Codable` is a supertrait, and supertraits now exist.** `trait Codable : Encodable + Decodable { }` is an ordinary prelude trait, its supers spelled like a bound with no members of its own. A bound `T: Codable` expands transitively, so `T` has `encode` and `decode`; `implement Codable for X { ... }` requires that `X` implements the supers — and an *empty* implement of a trait with supertraits synthesizes every super implement the type lacks, which is how one line derives both directions. A trait with supertraits **is** a trait object: its method table is the supers' tables laid end to end, most-derived declaration winning a name they share, so an object answers to every inherited method and a default body from a super is in the table like any other. Two supers that neither requires the other declaring the same name is an ambiguity and the trait is not object safe; every other object-safety rule (no associated types, no generic methods, no static functions, no `Self` in a signature) is now asked of the supers too — which is why `Codable` is still not an object, `Decodable.decode` being static and returning `Self`. A trait object is never converted to another trait's object: that needs the concrete type back (§9 item 14). Rejected: `Codable` as compiler-known sugar (a prelude name that is not a trait — hover, docs and errors would all special-case it); `implement Encodable + Decodable for X` (new implement syntax, two names on every DTO).

**What the derive knows.** The prelude implements the traits for the primitives, `List<T>`, `Map<string, T>`, `T?`, tuples and the `Json` tree itself. A struct becomes an object of its non-skipped fields in declaration order. A field's key is its name, or `@key("user_id")` for every format, or `@key(json: "userId", db: "user_id")` per format by the string the encoder reports (a format written in a library picks its own string, no compiler change); a casing policy for a whole API is an *encoder option* (`json.Options(keys: KeyStyle.SnakeCase)`), never an attribute. `@skip` leaves a field out in both directions (it needs a default, or the decoder could not construct the value); `@skip(json)` for one format. Two fields on one key are a compile error; a field of a type that cannot be derived (a function, a `Mutex`, a raw pointer, a handle) is an error at the implement naming the field, as the `Equatable`-key check already does.

**Absent, null, required.** A field with no default and a non-nullable type is required: absent or `null` is a problem. A field default is the value when the key is absent — there is no `@default` attribute because the language already has one. A `T?` field reads `null` and *absent* both as `null` (Swift's and serde's rule; Kotlin's stricter one makes the most common API shape the most verbose); `@required` marks a nullable field whose key must be present. When absent and null must be told apart — a PATCH body — the field is `T??` (D5 nests) or `json.Patch<T>`. Encoding writes `null` for a null field; `json.Options(omitNulls: true)` drops them.

**Sealed traits are internally tagged.** `implement Codable for Shape` covers the trait and every variant (the set is closed; a variant may still write its own implement): `{"type": "Circle", "r": 1.0}`, the tag being the variant's declared name (renamed by `@key` on the variant) under the key `"type"` (renamed by `@tag("kind")` on the trait; `@tag("type", content: "value")` opts into the adjacent layout `{"type": "Circle", "value": {...}}`). A variant with a field named like the tag is an error. Decoding takes the fast path when the tag is the first key — what this encoder writes — and otherwise buffers the object through the `Codable` `Json` tree and decodes the variant from that. A format that has no notion of variants (a row) reports a `DecodeError` at the path. Rejected: externally tagged `{"Circle": {...}}` (rare in APIs), untagged (try each variant: fragile, slow, and the errors are useless).

**Enums need nothing.** An enum is compiler-owned (D57: no `implement` blocks), so like a primitive it is `Codable` without a declaration: its name on the wire, matched exactly on the way in (an unknown name is a problem listing the valid ones), `@key("active")` renaming a member. Whether an enum travels as its name or its number is a property of *where it is going*, not of the enum — JSON wants names, a `smallint` column wants numbers — so it is the format's policy: `json.Options(enums: EnumStyle.Number)`; `json` defaults to `Name`, a row codec to `Number`; decoding is strict per style. Rejected: `@key(number)` on the enum (a second meaning for `@key`); a new attribute for a rare case.

**Decoding reports every problem.** `DecodeError { problems: List<Problem> }`, a `Problem` being a path and a message (`user.address[2].zip`: "expected a number, found a string"; `Problem.pointer()` is the RFC 6901 spelling, and a key containing a dot is quoted). The synthesized `decode` reads each field into a temporary, records a problem instead of throwing, skips what it cannot read, and throws once at the end; a nested struct's problems come back with the path prefixed; the list is capped so a hostile document cannot grow it. A malformed document is one problem — there is nothing to continue from. Validation, when it comes, appends to the same list, so a client fixes a request in one round trip. Rejected: fail-fast (one field per round trip); fail-fast for types with collected validation (two responses for one bad request).

**Also synthesized: `Comparable`.** `implement Comparable for Version` orders lexicographically by field in declaration order — the tuple rule of D48 applied to a struct — and fails at the implement on a field that is not ordered. `==`, hashing and printing were already structural and stay implicit.

*Implementation notes (v0.33).* A `Decoder` records a value of the wrong type as a problem, consumes it and returns a zero, so the derived code never has to recover a stream; `problem`/`problemAt`/`problems` are part of the trait. A nested value that cannot be built throws after its object is fully read, and its parent catches that, finishes its own checks, and fails once. Sealed families decode through the `Value` tree today; the in-stream fast path for a leading tag is still to come. Derived bodies are synthesized as syntax (with three nodes no source spells: a resolved type, a resolved type as a receiver, a field's default) and checked like anything written, so every rule of the language applies to them.

**Consequences.** D51 gains its first data-carrying attributes, all in service of this decision (`@key`, `@skip`, `@required`, `@tag`); the sentence "there is no derivation mechanism" is replaced by "derivation is what the compiler implements, requested by an empty `implement`". The compiler's synthesized-function machinery (tuple comparison, enum functions) becomes the general path for a derived body, so a later user-definable derivation has one description of a type's shape to expose. `std/json` is the first format; rows and environment follow.

---

### D59 — Cryptography and the encodings that carry it (v0.34)

A server cannot be trusted with a session, a webhook or an API key without
a hash, a keyed hash, unpredictable bytes and a way to write bytes as text.
Four modules, and the shape of each is chosen so that the safe call is the
short one.

**Four modules, not one.** `crypto` holds the digests (SHA-256, SHA-384,
SHA-512, SHA-1), HMAC, constant-time comparison, the system CSPRNG and
UUIDs; `hex` and `base64` are their own modules; `jwt` sits on top of all
three. base64 is an *encoding*, not encryption, and a name that suggests
otherwise is how `base64.encode(password)` gets written — so it does not
live behind `crypto.`. Rejected: one `encoding` module with the codecs as
statics (`encoding.Base64.encodeUrl(...)` is three hops for the commonest
call in a web server); everything under `crypto` (misfiles two modules that
protect nothing).

**A hash returns a `Digest`, not `List<u8>`.** `Digest` implements
`Display` (lower-case hex), `Hashable`, and `Equatable` with a
**constant-time** `equals` — so `mac == expected` is the natural spelling
*and* the safe one. `bytes()` is there for sending the digest, and its
result compares with the structural, short-circuiting `==`; the asymmetry
is deliberate, and `Digest.of(bytes)` wraps a signature that arrived from
outside so it can be compared the safe way. `crypto.equalBytes(a, b)` is
the same guarantee for bytes that are not digests. Rejected: returning
`List<u8>` (`if (mac == expected)` then compiles into a timing oracle, and
the only defence is a doc comment — the shape this language has refused
before, D45's reasoning applied to a leak instead of an error).

**A digest is a `Hasher`.** `start()`/`update`/`finish()` with three static
members (`algorithm`, `blockSize`, `digestSize`); the one-shot
`crypto.sha256(data)` is `digest<Sha256>(data)`. HMAC is therefore written
once, generic over `H: Hasher` — `Hmac<Sha256>` — rather than three times.
A hasher finishes once: `update` after `finish` panics, `finish` twice
returns the same digest. Streaming is not a nicety; an `ETag` for a file
and a MAC over a large body both need it.

**Decoding is strict and canonical.** Both base64 decoders take padded or
unpadded input, and refuse a character from the other alphabet (naming
which function to call instead), an `=` before the end, any whitespace, a
length that cannot spell whole bytes, and a final character with bits set
that the byte count does not use. That last one is the security case: a
non-canonical encoding means two different texts decode to one signature,
so a replay filter keyed on the text never sees the repeat. Every refusal
carries the byte offset. `hex.decode` is the same: either case, an even
number of digits, nothing else — no `0x`, no separators, no whitespace.

**`randomBytes` panics; it does not throw.** The operating system's
generator (`BCryptGenRandom`, `getrandom(2)`, `arc4random_buf`) failing is
not a condition a caller can answer: there is no weaker source worth
falling back to, and nothing to decide. Threading `throws IoError` through
every call site — including `uuidV4()`, which would then throw — buys
nothing, and a panic is still caught at a request boundary (D56). This is
the one place in the standard library where an *external* failure is a
panic rather than an error, and the reason is that it is not recoverable
rather than that it is rare. Go 1.24 moved `crypto/rand.Read` the same way.
Rejected: `throws IoError` (every session-token line grows a `try`, and the
handler can only crash anyway); a silent fallback to `std/random` (a
predictable token is worse than no token).

**UUID v7 is strictly increasing.** The twelve bits RFC 9562 calls `rand_a`
hold a counter instead (its §6.2 "fixed-length dedicated counter"),
starting each millisecond at a random point in the lower half of its range;
a millisecond that exhausts the counter borrows the next one, and a clock
that jumps backwards is ignored rather than obeyed. Monotonicity is the
whole reason to prefer v7 over v4 — as a primary key it appends to the
index instead of scattering writes — and milliseconds alone do not give it,
because a server makes many ids per millisecond. The cost is module-level
mutable state, which becomes a lock when the executor gets threads (D35).

**A JWT library is a list of refusals.** `jwt` implements the HMAC family
(`HS256`/`HS384`/`HS512`) and: takes the expected algorithm from the
*caller*, never from the token's `alg` (the header is unsigned input — this
is `alg: none` and RS256-verified-as-HS256); requires `exp` unless asked
not to; rejects `crit`; compares the signature through `Digest`; and
refuses a key shorter than the digest (RFC 7518 §3.2) with a panic, because
a short HMAC key is a configuration mistake and serving requests with it is
worse than stopping. `Claims` spells the registered claims out (`issuer`,
`expiresAt`, …) in Unix seconds and keeps the rest as `Value`; `Invalid`
carries a `Reason` enum, because `Expired` means "refresh" and everything
else means "sign in again". `readHeader` exists for `kid` lookup and says
in its doc that nothing it returns is trustworthy. RSA and ECDSA are out of
scope until there is a bignum or a binding.

*Implementation notes (v0.34).* All of it is Veles except one runtime call,
`veles_random_bytes` in `veles_os.c` (and `-lbcrypt` on Windows). The
digests keep a per-hasher scratch buffer so a long message allocates
nothing per block; the round-constant tables are module-level `val`s, which
are globals initialised once at start-up. Two compiler changes fell out of
writing it: a module-qualified generic type may now be a static call target
(`crypto.Hmac<Sha256>.start(key)` — the parser only allowed a bare name
before, `ast.MemberExpr.TypeArgs`), and the formatter keeps the author's
grouping inside a list it breaks, so a table written as a grid stays a grid
instead of becoming one constant per line.

---

### D60 — Time: a length, a wall clock and a monotonic clock are three types (v0.35)

Before this entry every time in Veles was an `i64` whose unit lived in a
parameter name. `withTimeout(5000, ...)` was milliseconds, `Stopwatch.elapsedNanos()`
was nanoseconds, `time.now()` was milliseconds since the epoch, and all
three could be added to each other. `withTimeout(5, ...)` — five seconds,
surely — compiled and gave up after five thousandths of one. This is the
D59 `Digest` argument again: the mistake that matters is the one that
compiles.

**`Duration` is a type, in the prelude.** A one-field struct over an `i64`
of **nanoseconds** (±292 years), with `Duration.seconds(5)` and its
siblings, `Comparable`, `Display` (`0s`, `250ms`, `1.5s`, `2m30s`, `1d1h`)
and `Parsable`, which reads back exactly what `Display` writes — the
fraction is carried in whole nanoseconds, never through a float, so
`Duration.parse("$d") == d` for every `d`. It lives in the prelude, not in
`std/time`, because the things that take one are there: `sleep`,
`withTimeout`, `Timeout`. Nanoseconds rather than milliseconds so that
`Stopwatch.elapsed()` is the same type as a cache lifetime.

Arithmetic is methods — `a.plus(b)`, not `a + b` — because the language has
only five operator traits (`Comparable`, `Equatable`, `Hashable`,
`Display`, `Parsable`) and no arithmetic ones. That cost was accepted
rather than paid for with a language change: adding an `Arithmetic` trait
is a decision about operators, not about time, and it can be taken later
without a second spelling appearing here, because `plus`, `minus`, `times`
and `dividedBy` are named to be adopted by it. What made the cost small is
that the two sums anyone actually writes — "now plus a timeout" and "how
much is left" — are not arithmetic at all once `Deadline` exists.

**Two clocks, two types, and no raw monotonic reading.** `Timestamp` is a
point on the wall clock; `Deadline` and `Stopwatch` are the only things
built from the monotonic one, and `time.monotonic()` is gone. A monotonic
reading therefore cannot be compared with a wall-clock one, serialized into
a log, or mistaken by a reader for a time of day — the confusion is not
documented, it is unrepresentable. Rejected: Go's `time.Time`, which
carries both readings in one value and whose subtraction changes meaning
silently once the value has been serialized.

**`Timestamp` counts microseconds.** Milliseconds would lose the six
fractional digits an RFC 3339 producer and a PostgreSQL `timestamptz` both
write, on a round trip through a server whose job is to hand them back;
nanoseconds would buy precision the OS clock does not have and cost the
range, as Go's year-2262 ceiling shows. Microseconds span ±292,000 years
and order two events inside one millisecond. `toSeconds`/`toMillis` round
**down**, not toward zero — they name the second *containing* the instant,
which is what the calendar needs and what keeps working before 1970 —
while `Duration`'s conversions truncate toward zero, because a length has a
sign and a point does not.

**Zones are a fixed `Offset`, and the calendar is pure Veles.** `Offset` is
minutes east of UTC (±18:00); `DateTime` carries one, so it prints full
RFC 3339, and `Offset.local(at:)` asks the host for its offset *at that
instant*, which is the only form the question has. The conversion itself is
Howard Hinnant's `days_from_civil` and its inverse, in Veles: deterministic,
no C round trip, and no `mktime` ambiguity. The C side answers exactly two
questions — what time is it, and what is this host's offset. The IANA
database, and with it "09:30 local on the morning the clocks go forward",
is deliberately out of scope.

A `DateTime`'s fields are data, not an invariant: `month: 13` is January of
the next year and `day: 71` is forty days after the 31st, so date
arithmetic needs no second API. Text is the strict half — `parseRfc3339`
refuses month 13.

**One RFC 3339 parser, with three named leniencies**: lower-case `t`/`z`
(the RFC's own NOTE), a space where the `T` goes (§5.6, and what PostgreSQL
prints), and ISO 8601's expanded year, so that `parse(t.toString())` is
total rather than total only between the years 0000 and 9999 — bar the last
second at each end of the microsecond range, which prints and does not read
back, a pair of instants not worth unreadable arithmetic. It refuses a
missing offset, `24:00:00`, a bare date and any field out of range; `-00:00`
("offset unknown") reads as UTC and is never written. A fraction longer
than six digits is truncated, not rounded, so the order of two texts stays
the order of their instants. `23:59:60` is legal RFC 3339 and has no POSIX
instant: it is accepted as the last microsecond of that minute, the instant
a POSIX clock reports while it happens. Go rejects it; a conforming
producer's timestamp failing to parse is the worse answer.

**A date from outside is refused, never overflowed.** The year in
`+999999-01-01T00:00:00Z` is 3.2e19 microseconds, and the year in
`Sun, 06 Nov 99999999999999 08:49:37 GMT` is whatever the client typed; both
reached `days * 86400 * 1000000` and panicked on the overflow D21 checks for
(or, in a release build, wrapped into a plausible wrong instant, which is
worse). Every conversion from calendar fields now goes through one checked
place, `instantOf`, which bounds the year before `daysFromCivil` can
overflow and the seconds before the multiplication can — so a parser answers
`null` and only a hand-built `DateTime` can still reach the panic, which is
D21's business. Formatting was made total the same way: applying an offset
to the microsecond count, and `floorMod`'s `a - floorDiv(a, b) * b`, both
overflowed within a day of the ends of the i64 range, so `at` now shifts the
day and the microsecond within it separately and `floorMod` is
`((a % b) + b) % b`. What is left is one honest gap: the last second at each
end prints and does not parse back, because the parser stops where the
product would leave the i64, and the arithmetic to recover those two seconds
is not arithmetic a reader could check.

**The host is asked what it can answer.** `Offset.local(at:)` reads the zone
through the C library, and the Microsoft CRT refuses a negative `time_t` and
anything past the year 3000 where glibc is happy — so a date of birth read
back as local time silently became `Z`, right only in the UK. The C function
now reports "cannot answer" distinguishably from "UTC" (zero is a real
offset), and `std/time` asks again for the same month, day and time of day
in a year the host can convert, keeping the leap-year parity. That is the
assumption a fixed-offset model makes anyway — that the rules did not
change — and it gets the daylight-saving half right, which a fallback to
UTC does not.
**HTTP-date lives in `std/time`, not `std/http`.** `formatHttp` writes
IMF-fixdate only; `parseHttp` reads it and the two obsolete forms RFC 9110
§5.6.7 says a recipient must accept, because RFC 850 and asctime are
exactly what an old client puts in `If-Modified-Since`. `Last-Modified` and
`If-Modified-Since` are two ends of one conversation and belong to one
module; `http.httpDate` is a one-line name for the same function.

**The migration is the point, so it was done in full.** `sleep` is a
compiler builtin and now takes a `Duration`, resolved through
`preludeType("Duration")` as `Ordering` and `Encoder` already are, and
lowered to `(ns + 999999) / 1000000` — rounded **up**, so a sleep is never
shorter than it was asked for and a sub-millisecond one still yields.
`sleep(500)` is an error that names the fix. `withTimeout` and `Timeout`,
`http.Limits`' three clocks, `http.timeout`, `uuidV7`, `jwt.now` and the
`random` seed all moved with it; `Timestamp` also implements `Codable` by
hand, as RFC 3339 text, so a timestamp field in a derived struct is never a
number of microseconds on the wire. Rejected: keeping the types and
migrating later (two spellings, which is the objection `notes_to_change`
keeps making), and leaving `sleep` on `i64` (the one place in the language
where a time would still be an unlabelled number).

**Addendum (2026-09-25): a `Duration` on the wire.** `Duration` implements
`Codable` by hand, and *how* it is written is the format's policy, not the
type's — the `EnumStyle` precedent: `Encoder`/`Decoder` gained
`durations(): DurationStyle` (default `Seconds`), `json.Options` a
`durations` field, `ValueEncoder`/`ValueDecoder.of` a `durations`
parameter. The styles: **`Seconds`** `"90.5s"` (the default: protobuf's
JSON mapping for `google.protobuf.Duration` and Go's `time.ParseDuration`
read it, and it is exact), `Iso8601` `"PT1M30.5S"` (hours at most on the
way out, as Java writes it; days read as 24 hours on the way in; years,
months and weeks refused with a message saying why — a month is not a
length), `Text` (the `Display` form), `Nanos` (an exact integer) and
`Millis` (an integer when whole, a fraction otherwise). Every text style
reuses the exact integer parser behind `Duration.parse`. Decoding is strict
per style. Rejected: one fixed form (the user's requirement was to be able
to convert to the others); ISO 8601 as the default (Go and Python's
standard libraries do not read it); a number of seconds as the default
(precision past 2^53 ns).

### D61 — Falling back and bailing out: `??` and let-else (v0.36)

`try` hands a failure up; `?:` (D30) replaces a missing value. What was
missing was the same for a `Result`, and a way to *bind* a value or leave,
without burying the happy path in a `when` whose only job is to bail out
(notes I1; the write-up with the alternatives is `veles-guard-design.md`).

**`r ?? fallback`** — the value of a `Result<T, E>`, or `fallback` when it
is an `Err`. `fallback` is a `T`, or `Never` (`?? return`, `?? continue`,
`?? throw X`), or a **handler** `{ e => ... }` whose block sees the error
and yields a `T` or leaves (`{ _ => ... }` ignores it). Right-associative,
at `?:`'s precedence; a line may start with `??` and continue the previous
one, as with `?:` and `?!`. In a type, `T??` is still two `?`.

**One operator per kind of "maybe"** (user decision, 2026-09-25). `?:` is
for a nullable, `??` for a `Result`; each on the other kind is an *error*
whose fix swaps the operator, so a reader always knows from the operator
which kind is being unwrapped. Every other language with `??` (Swift, C#,
JavaScript, Dart, PHP) means by it what Veles means by `?:`; the error and
its fix are what teach the difference.

**Let-else** — `val <binding> = <expr> else <handler>`:

- on a **`Result`**, binds the value; `else { e => ... }` sees the error;
- on a **nullable**, binds the value (`else { e => }` is refused: there is
  no error to bind);
- with a **variant pattern**, `val Circle(r) = shape else ...`, binds the
  pattern's names — the one form of `val` that takes a pattern, since a
  pattern can fail and only let-else says what happens then.

The `else` **must diverge** (checked: its type is `Never`), because the
names do not exist on that path. It may be a braced block or, like the
body of `if (c) stmt`, one statement: `val v = parse(s) else continue`.
The formatter keeps a one-statement `else` on the line; a braced one
breaks, as every braced body does. `var` works the same; tuple bindings
destructure the value. `else` may start the next line. Errors: `else` that
does not leave; `else` on something that cannot fail ("drop the 'else'");
a pattern that always matches; a pattern `val` without `else`.

Known edge: `val x = if (c) a` followed by `else ...` on the next line is
an `if`/`else`, not a let-else — the `if` claims its `else` first, as it
always has. Parenthesise the `if`.

Rejected: `?:` widened to `Result` (one operator for both — what was
recommended, and turned down so the operator shows the kind); a
`Fallible` trait letting user types take part in `?:`/`??`/`try`/let-else
(a separate decision, not foreclosed by this one); Swift's `guard let`
spelling (a second keyword for what `val` already says).

### D62 — Proving an index instead of panicking on it (v0.37)

`atOrPanic` made the unchecked read the short one, so code checked a
length and then read with `atOrPanic` anyway, because nothing tied the
check to the reads (`if (parts.len() != 3) throw ...` followed by
`parts.atOrPanic(0)` in std/jwt). A survey of std's ~140 panicking reads
found five shapes: (1) a length checked, then constant indexes; (2) a loop
over the list's own indices; (3) index arithmetic (`i + 1`, `mid`);
(4) a structural invariant (a parser's stack is never empty); (5) real
misuse (`chunked(0)`). Only 3–5 need a panic. D62 gives 1 and 2 a
spelling that cannot fail, and makes the rest say why they cannot.

**A — List patterns**, in `when` arms and let-else (D61), beside the
variant and tuple patterns (D13, D37):

```veles
val [header, payload, sig] = token.split(".") else throw Malformed()
when (args) {
  []            => usage()
  [cmd]         => run(cmd, [])
  [cmd, ..rest] => run(cmd, rest)
}
val [.., last] = xs else return null
```

`[p1, …, pn]` matches exactly `n` elements; one `..` (at most one, in
any position) matches any number, and `..name` binds them as a new
`List<T>` (a copy, like every read — D25). Elements are sub-patterns:
names, `_`, literals, and nested tuple/variant/list patterns. The names
are copies taken at the match, so a later change to a `MutableList`
does not reach them. Exhaustiveness is by length: `[]`, `[x]` and
`[x, ..]` together cover every list; a literal element never makes an arm
cover. A list pattern `val` without `else` is an error, as for any
pattern that can fail.

**B — Bounds facts.** `xs.at(i)` (and `xs.ref(i)`) has type `T` (`*T`),
not `T?`, where the checker knows `-xs.len() <= i < xs.len()` — D5's
smart casts, applied to an index. What establishes a fact:

- `loop (i in 0..<xs.len())` and `loop (i in xs.indices())`, for the loop body;
- a guard `i < xs.len()` (with `i >= 0` also known, or `i` a loop
  variable counting up from `0`) — `if`, `&&`, `while`, and the negated
  early exit (`if (i >= xs.len()) return`), as for null checks;
- a constant index `k` after `xs.len() == n` / `>= n` / `> n` with `k < n`,
  or `-k` counted from the end.

What ends it: assigning `xs` or `i`; on a `MutableList`, any call of a
length-changing method on it (`push`, `pop`, `insert`, `removeAt`,
`clear`, …) and **any call that could reach it** — a `MutableList` is a
reference, so an unknown call may shrink it through an alias. An
immutable `List`'s facts survive calls. Arithmetic (`i + 1`) is not
reasoned about; such reads stay `T?`. Losing a fact after a refactor is a
compile error (a `T?` where a `T` was wanted), never a wrong program.

*As built (2026-09-26).* The facts ride in the smart-cast map (D5), so
branches, `&&`/`||`/`!`, the early exit and the merge after an `if` treat
them exactly as they treat a null check. `xs.first()` and `xs.last()` are
`T` after a fact that the list is non-empty (`!xs.isEmpty()`,
`xs.len() > 0`). A "call that could reach it" is, conservatively, any
call except the list's own `at`/`len`/`set`/`ref`/`isEmpty`/`push`/`get`
(and `byteAt`); a loop body is scanned before it is checked, because a
call late in the body runs before the next iteration's read, and so is the
right side of `&&`/`||`. A lambda body starts with no bounds facts. The
proven read still goes through the runtime's checked get, so a case the
rules miss is a panic, not an out-of-bounds read. A `while` over
`var i = 0 … i += 1` is not covered (the assignment ends the fact); use
`loop (i in xs.indices())`.

*Gaining a fact must not break code.* When a read that used to be `T?`
becomes `T`, a `?:` after it is a **warning** with a fix that drops the
fallback, not the usual "needs a nullable left operand" error: adding a
length check above working code should never make that code an error.

**C — No `…OrPanic`.** `atOrPanic`, `getOrPanic` and `refOrPanic` are
removed. A read that can only fail through a bug says so where it is
written: `xs.at(i) ?: panic("json: the stack is never empty while
parsing")`. `panic(message)` is the one way to panic from Veles code, so
`panic(` finds every site. The removed names are errors carrying a fix to
the `?: panic("...")` form with the reason left for the author. *(Built v0.37: the fix writes `(x.at(i) ?: panic("TODO: say why this cannot fail"))`; `panic(` never returns, so it does not end bounds facts.)*

**D — Shapes instead of indexes, in std.** `xs.indices(): Range`,
`s.splitOnce(sep): (string, string)?`, and `enumerate()`/`zip()` on
`List` directly, so the common loops need no index at all. *(Built v0.37: `enumerate()` is eager, like the other `List` adapters; `xs.iter().enumerate()` stays the lazy form.)*

Rejected: keeping `atOrPanic` behind a lint (the short spelling stays the
unchecked one); an `.expect("why")` method (a second spelling of
`?: panic("why")`); fixed-size arrays for constant indexes (a larger
feature, its own decision); reasoning about index arithmetic (a solver,
and errors nobody can predict).

### D63 — A MutableList becomes a List by moving or by copying (v0.38)

D35 made `List` genuinely immutable and `mutableList.toList()` a copy, but
the checker still let a `MutableList` stand where a `List` was expected,
passing the *same* handle — so a "List" could change during a call (found
by D62's bounds facts: a `List` parameter shrank through a global alias),
and could be shared with a task while its owner kept writing. The same
holds for `MutableMap`/`Map` and `MutableSet`/`Set`.

**The rule.** A mutable collection is accepted where its immutable form is
expected only when it is **moved**: a local that this function built
fresh (a literal, or a `toMutable()` copy) and that has not escaped — used
only as the receiver of the collection's own methods (not `ref`, `iter`)
and in `loop (x in c)`, never passed, stored, aliased, captured, assigned
or bound by `&` — at its **last use**: a `return`, the function's result,
or textually the last mention outside any loop the local was not
declared in. Nothing is copied; nobody else can see the storage any more.
Anywhere else it is an error whose fix appends `.toList()` / `.toMap()` /
`.toSet()`, which copies.

```veles
fun encode(bytes: List<u8>): List<u8> {
  val out: MutableList<u8> = []
  loop (b in bytes) out.push(b ^ 0x5A)
  out                                  // moved: no copy
}
fun keep(buffer: MutableList<u8>) {
  digest(buffer)                       // error: a parameter may be shared — 'buffer.toList()'
}
```

Rejected: implicit copy (hidden O(n) cost); explicit `.toList()` always
(a copy for every builder); keeping the view (D35's `Sendable List`
would be false). The analysis is conservative: a case it cannot prove is
a `.toList()` away, never a wrong program.

### D64 — No `!!`; every panic says where it happened (v0.39)

Asked after D62: a postfix `x!!` (Kotlin) that panics when `x` is null,
reporting the line. **Not added.** It would bring back D62's cheatcode at
two characters, for every `T?` rather than only index reads —
`s.toInt()!!` makes a panic the easy answer to bad input, which is a
thrown error's job — and it records no reason, where
`?: panic("two counters")` states the assumption that failed. It also
reads badly beside prefix `!` (`x!!` against `!!flag`).

What `!!` offered that was missing: the location. `panic` printed only
its message. Now every panic the compiler emits carries the site's
`file:line:col`, relative to the package root with `/` separators (std
files keep `std/...`), so a binary does not embed the build machine's
directories:

```text
panic: index 3 is past the end of the list
  at main.vs:7:31
```

It covers `panic(...)`, overflow, division by zero, `pow`, a
non-exhaustive match and a list index out of range (`xs.set(9, v)`); a
panic raised inside the runtime itself (a closed channel, a deadlock)
has no line. A task's panic keeps its location through `scope`'s
re-raise, `gather` exposes it as `Panic.location` (`""` when there is
none; a field with a default, so `Panic(message: m)` still builds), and
`veles test` prints it under `FAILED: panic:`.

Rejected: `!!` everywhere; `!!` with a warning and a `--fix` to
`?: panic("TODO…")`; `!!` only in tests.

*Known gap:* a panic written in std for a caller's misuse
(`xs.swap(0, 7)`, `chunked(0)`) reports the std line, not the caller's —
Rust's `#[track_caller]` is the model if it proves to matter.

---

## 4b. Settled minor decisions

- **Semicolons** — Go-style automatic insertion.
- **Loop labels** — `loop :outer { ... break outer }`; the label follows the keyword so the statement still starts with `loop`.
- **Byte access** — `s.byteAt(i)` and `s.bytes()`, explicit views; `List<u8>.decodeUtf8()` validates on the way back. `'"'` is a byte literal: the `u8` of one ASCII character (v0.24), so the code that walks bytes can name what it compares against; a multi-byte character is an error, never a char type. There is no subscript on text (D25, v0.24: no subscript anywhere).
- **`panic(message)`** — a built-in that never returns (D20); a `T?` fallback like `xs.at(i) ?: panic("...")` types as `T`.
- **`Set`** — follows D25's immutable/mutable split and is insertion-ordered, matching `Map`.
- **`gc.retain` handles** — `Closeable`, acquired through `with` (D43).
- **`implement`, not `implement`** *(v0.33)* — the keyword is the word: `implement Display for Point { }`, `implement Codable` in a struct body. `implement` was the Rust abbreviation; the parser still reads it and reports the spelling with the fix. The AST node keeps its name.
- **A function's type parameters follow its name** *(v0.33)* — `fun encode<T: Encodable>(value: T)`, as on a struct (`struct Page<T>`) and at the call (`decode<User>(text)`); Swift, TypeScript, Go and Rust agree. Kotlin's `fun <T> encode(...)` was accepted alongside it since v0.1 and is now an error with the fix: one spelling.
- **Module interface files** — build cache, regenerated. Packages distribute as source under the decentralized registry model (manifest §7), so there is nothing to ship them in.
- **`veles.sum`** — mandatory. Supply-chain integrity is not opt-in.
- **Variadic C functions** — excluded. The varargs calling convention differs per platform and is the nastiest corner of the C ABI; bind a fixed-arity wrapper instead.
- **`testing`** — stdlib, with a `test` build mode. Go's version of this is a genuine strength.
- **Discarded `async` handles** — no different from bound ones. `scope` joins either way, so there is nothing to warn about.

---

## 4a. Implementation plan

**I1 — Backend: LLVM.**

**I2 — Emit textual LLVM IR, not cgo bindings.** Go↔LLVM through cgo means version-matched C++ builds, slow FFI, and historically under-maintained bindings. Writing `.ll` and invoking `llc` costs some compile speed and nothing else. The C API remains available later if process overhead ever matters.

**I3 — The GC needs no stack maps.** LLVM's `gc.statepoint` support is weak and designed for moving collectors. Veles has none: D10 forced non-moving, and D2's stackless coroutines mean stacks never grow or relocate, so no stack pointer ever needs updating. Scan stacks **conservatively**, keep the heap precise via type info in object headers — the standard mostly-precise design. Go needed precise stack maps only because it moves growable stacks.

**I4 — Bootstrap in Go, self-host once stable.** Go will be tedious for this: Veles has sealed traits and exhaustive matching, and its first compiler is written in a language with neither, so AST and IR handling becomes interfaces plus type switches with no exhaustiveness checking.

Self-hosting sets stdlib priority order: the first thing Veles must be able to write is its own compiler, so files, maps, strings, and error handling come before everything else.

---

## 5. Consequences already locked in

**The GC algorithm is determined by D10.** Interior pointers rule out a moving collector — compaction would have to resolve every arbitrary address back to its containing object during pointer fixup. Veles therefore requires:

- Non-moving, mark-sweep collection
- Size-segregated spans with per-span metadata, so any address resolves to its base object
- Non-generational (generational collection is substantially harder without moving; Go does not have one)
- Fragmentation is a permanent operational concern, not a solvable one

**Two inferred effects require materialized module interfaces.** Suspension (D2) and error type (D4) are both inferred, and both leak into a module's public interface without appearing in its source. The fix for both: infer within a module, then write the results into a generated interface file — the model used by Rust's crate metadata and Swift's `.swiftinterface`. Downstream compilation reads materialized facts and never re-infers.

Combined with M2, the **module is the inference unit**, and one interface file is generated per module. This format is load-bearing far beyond error handling; it is the backbone of the module system and the package manager, and should be designed early rather than treated as a compiler implementation detail.

---

## 6. Open questions requiring a decision

**6.1 — Whether to lint the `cond => x => ...` case.** See D33. Accepted as-is for now.

**6.2 — Manifest schema.** Drafted in `veles-manifest.md`.

## 7. Not yet designed

- Trait declaration syntax (generic bound syntax settled v0.33: `fun name<T: Ord>(...)`, the parameters after the name as on a struct)
- Closure representation and capture semantics (syntax is settled in D32)
- Whether `Mutex.withLock` becomes a `with` resource (D43)
- Whether collection literals are wired to fixed types or an opt-in trait (D41)
- `Mutex<T>`, `Atomic<T>`, and the rest of the synchronization surface (D35)
- `for` loop syntax and its desugaring onto `Iterator`
- Task API surface: spawning, joining, cancellation propagation (D3 states the discipline, not the API)
- C ABI FFI design
- Standard library scope beyond the self-hosting minimum
- Build tooling, formatter, LSP


---

## 8. Interop — scoping note

Direct import of Go, Kotlin, or TypeScript libraries means hosting each of those runtimes (the Go scheduler, a JVM, V8) inside the Veles process. This is not a feature; it is three separate projects, each larger than Veles itself.

The tractable version:

- **C ABI FFI** for native targets — reaches C, C++, Rust, Zig, and anything else speaking the C ABI
- **JS import/export** for the WebAssembly target when it arrives
- **A declaration-file system**, in the spirit of `.d.ts`, so foreign libraries can be bound with Veles-side type safety

This covers the large majority of the practical value at a small fraction of the cost.
