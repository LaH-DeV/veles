# Veles — Language Specification

**Working draft v0.69** — decisions D1–D137. Open design questions: none (`veles-checklist.md` §9); the order of building them is the "Build order" list in `veles-plan.md`.

Decided 2026-09-30/10-01 with the user, from the review in `archive/veles-spec-prep.md` (each entry is written to be built without further questions):

| Entries | Subject |
|---|---|
| D100 | `with x = e` as a statement; `with t = async f()` |
| D101–D106 | heads and `val`, a collection changed while looped, one-task `gather`, `async` on a value, bit operations, `reserve`, empty literals |
| D107–D111 | `with p = m.lock()`, `race` send arms, `with expr`, concurrency helpers, values that hold a task |
| D112–D115 | `Secret<T>`, compile-time evaluation and `const fun`, unchecked access, never-closed warning |
| D116–D119 | suspension follows the argument, `is Trait`, derivation direction, `Default` |
| D120–D123 | C layout attributes, `Array<T, N>` and const generics, no `.d.vs`, variadic C calls |
| D124–D130, D133 | compression, config, OpenTelemetry, the HTTP client, `io.Stream` + TLS, template literals + `std/db`, small std additions, health endpoints |
| D131–D132 | the manifest is `package.vs`; decentralized packages |
| D134–D135 | (consistency pass) one `try` over a chain; `is T` downcast on a trait object |
| D136 | `close()` by hand on a `with` value is refused |
| D137 | a static of a generic type infers its type arguments |

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

**Errors are declared, and every error implements `Error`.** The prelude declares `trait Error { fun message(): string = "$this" }`. An error type is declared with a contextual keyword: `error ParseError { text: string; fun message(): string = ... }` is sugar for `struct ParseError { text: string }` plus `implement Error for ParseError { override fun message() ... }`. Inside an `error` body `message` needs no `override` (there is exactly one trait in play; writing it is allowed), and no other method may carry one — they are inherent. A field `message: string` with no `message()` written is used as the message, so `error Failed { message: string }` is thrown as `Failed(message: "...")`; `Panic` is exactly that. **Only errors can be in error position** (a `throws` clause, a `throw` operand, the `E` of a `Result`): throwing a plain struct is an error naming the fix (`error Dog { ... }`), non-structs are rejected, and a generic `throws X` needs the bound `X: Error`. `implement Error for T {}` on an existing struct is the escape hatch for types one does not own. Because every member of an error union implements `Error`, **the union exposes `Error`'s methods directly**: `e.message()` on a `ParseError | RangeError` compiles to a tag switch, no `when` required. An error escaping `main() throws` is reported through `message()`. This is Go's one-method `error` interface with structured payloads kept as fields, and the declaration form makes "is an error" a property of the type rather than of its uses. `error` is a keyword only at declaration position (`error Name`); elsewhere it is an ordinary identifier, so `is Err(error) => ...` is unaffected. Rejected: treating every struct as an implicit `Error` (it made `throw Dog()` legal); `struct X : Error` (in D12 that syntax means variant membership); and making `message` a mandatory field (it cannot be computed from the other fields, which is the common case).

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

**What a smart cast is about: a stable place.** A fact (`x != null`, `x is Circle`, `r is Ok`) attaches to a *place*: a local variable, or a chain of direct struct fields starting from one (`config.cause`, `this.address` in a non-`mut` method). Reads of the place are narrowed until something could change it: an assignment to the place or to a prefix of it (`u.address = ...`, `u = ...`), a `mut` method call on the root, or `&root` being taken — after which no new facts form on the root's fields either. Nothing reached through a pointer, `?.` or an index is a place: with `p: *User`, `p.address` may change through another pointer between the test and the use, so it is never narrowed (this is Kotlin's "stable value" rule with aliasing made explicit; Veles has pointers, Kotlin does not). Inside a `mut` method `this` is a pointer, so `this.field` is bound to a `val` first. Facts on the variable itself survive a `mut` call or `&` (they cannot change which variant a value is); only the field-path facts are dropped.

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

*(v0.66, D131: the manifest is `package.vs`, one typed constant evaluated alone; read `veles.toml` below as that file, and "never by parsing Veles source" as "parsing only that constant". No `version` field — versions are tags, D132.)* `veles.toml` at the package root holds name, version, dependencies, license, ~~exports,~~ build config (D89: a package's surface is what its root module re-exports). **No source file declares package identity.** The resolver builds a dependency graph by reading manifests, never by parsing Veles source.

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

Default visibility is module-private. `public` makes a declaration visible to the rest of the package. ~~Nothing escapes the package except through the manifest's `exports` field.~~ *(D89, v0.48)* Nothing escapes the package except what its root module re-exports with `public use`; the manifest has no `exports` field.

Consequence: library authors get a deliberately curated public surface, consumers cannot reach into internals, and Veles never needs Go's magic `internal/` directories.

*Amended (v0.29) — spelling and a third level.* The keyword is `public`, not `pub`: it now sits next to `private`, and the pair every Java, C#, Kotlin, Swift and TypeScript reader knows is `public`/`private`; the three extra characters on each exported declaration were weighed against that and lost. `export` was rejected for it — exporting is what the *package* does, in the manifest, module by module, and a keyword by that name on a struct field would claim a boundary the field does not cross. (Whether the package boundary itself should one day be spelled in source with `export` rather than in the manifest is an open question the user has reserved; nothing here prejudges it.) The levels are therefore: **`private`** — visible only inside the type's own declarations: its methods, its `implement` and `extend` blocks in the same module, and its `static val` initializers (Swift's rule for extensions in the same file); **nothing** — the module, as before, which keeps small programs light; **`public`** — the package. `private` exists for fields and methods only; a module-level declaration is already module-private. A private field cannot be read, assigned or bound in a pattern from outside. In a constructor call the rule turns on the default (v0.30): a private field *with* a default is the type's own state and outsiders leave it to the default; one *without* a default is the initial state only the constructor call can supply, so it is given from anywhere the type is visible and is private from then on — Kotlin's `class Parser(private val toks: ...)`, without the syntax. Swift's rule (a private stored property makes the memberwise initializer private) was the v0.29 behaviour and was dropped: it forced a `static fun` on every type that merely wanted to hide what it was given. That is the whole encapsulation story, with no getters and no `friend`. Motivated by a one-file program whose request handlers reached into a store's counter: modules were the only boundary, and a directory per invariant is too heavy.

*Addendum (v0.30) — the unwritten level has a name: `internal`.* "Nothing written means the module" was judged not self-explanatory, so the level is called *internal* (Kotlin's and Swift's word for it; `protected` was considered for this level and given instead to write-protection of fields, D22 addendum, where its "the owner only" sense fits) and may be written: `internal fun helper()`, `internal x: i64`, `internal struct Note`, on the same positions as `public`. It changes nothing and is never required; the formatter keeps it where the author wrote it; `public internal` is a contradiction and an error. The example package's `internal` module was renamed `support`, since the word is a keyword now.

### M6 — Import paths are logical

Resolved by the package manager against the manifest. No filesystem-relative string paths.

*Addendum (v0.28) — modules only, one statement, many imports.* *(names since D85: `use m { f }` adds bare names; the rest below stands.)* A `use` imports modules and nothing smaller: members are always qualified (`geometry.Point`, `io.println()`), and `use geometry as geo` renames the module when the prefix is long. The braced name-import form (`use geometry.{ Point as P, norm }`) is gone — Go's reasoning: a qualified name says at the use site where a thing comes from, two modules may both declare a `Point` with no renaming, and there is one way to write a call. A parameter or local named like a module shadows it (`fun mkdir(path: string)` cannot call `path.dir`); the answer is an alias at the import, not a name-import escape hatch. Type aliases (`type P = geo.Point`) are the remaining way to shorten a type name and are not in the language yet; add them if the prefix on types proves to hurt. `use` takes a comma-separated list: `use fs, io, os`, `use geometry as geo, shapes`; a comma at the end of a line continues the list. Import order and grouping carry no meaning, so the formatter owns them: consecutive `use` lines become one sorted list per origin — the standard library first, then everything else — one `use` statement each. Go groups its imports the same way (goimports), by convention rather than syntax: a separate spelling for standard-library imports would turn every move of a module between the library and a package into a source edit, and the compiler already knows which is which.

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

*(v0.66, D132: the major version is part of the version, not the path — identity is (repository, major).)* Requires: a hard "no breaking changes within a major version" culture, since MVS trusts that promise completely; ~~the major-version-in-import-path convention for v2 and later;~~ and an explicit upgrade command, because MVS selects the *minimum* satisfying version and will not pick up security patches on its own.

### D22 — Method receivers: implicit `this` (spelled `self` before v0.40, D65); mutability is declared on the field (`var`)

**REVISED (v0.30).** The original rule — `mut fun` marks a method that assigns its receiver's fields, and only a `var` binding may call one — is withdrawn. It was honest for structs of scalars and silent for the common shape of a domain type, a struct holding a collection: `val s = Stack()` did not stop `s.items.push(x)` in a plain `fun`, so the signature said "does not mutate" and the reader was misled. Making `mut` transitive through reference fields was considered again and rejected again: it needs to know which structs are values and which are handles, which is a `class`/`struct` split (Swift's) that the language does not want, or an ownership system.

**The rule.** Whether a field can change is written on the field:

```vs
struct Counter {
  label: string          // bare: set by the constructor call, never assigned again
  var n:  i32 = 0        // var: assignable

  fun get(): i32 = this.n
  fun bump() { this.n += 1 }     // no marker: a method may assign the receiver's var fields
}
```

- A **bare field** is never assigned after construction — not through `this`, not through a `var` binding, not through a pointer, not through `loop (&x in xs)`. The compiler reports it with a fix that inserts `var`.
- A **`var` field** is assignable through any place: `this`, a binding whether `val` or `var`, a pointer, an element reference. `val`/`var` on a binding govern rebinding only (D11, amended) — as they always did for a `MutableList`.
- `var` governs *that slot*. `o.inner.x = 1` needs `x` to be `var`, not `inner`: what is inside a field is governed by its own type, and a method call on `o.inner` could always reach it. The guarantee is therefore stated transitively: **a type with no `var` fields and no mutable collections reachable by value cannot change, whoever holds it.** Fields hold the truth about mutability where a reader looks for it, and `private var` next to a bare handle (`private var next: i64; private items: MutableList<Note>`) says exactly which two things in a type ever move.
- `this` is not assignable as a whole (`this = ...`): assign its fields, or return the new value.
- There is no method marker. Kotlin's `val`/`var` properties are the model; Swift's `mutating` and Rust's `&mut self` were the alternatives, and both exist to gate the *call* on the binding, which is the thing this design gives up. A struct is still a value: a method called on a temporary — `xs.at(i)?.bump()`, `make().bump()` — changes a copy that is discarded, which is an error when the method changes its receiver and returns nothing (the rule that guarded `mut fun` on a copy, D25 v0.27, kept). A struct passed to a function is a copy the callee may change without the caller seeing it; that is Go's value receiver and it is one rule rather than two.

**Calling convention.** Every method receives a *pointer to the place it was called on*, and `this` reads as the value; a temporary is spilled first. The representation is not declared and not observable, as before. Two facts about each method are computed once every body is checked (the receiver pass), to a fixpoint over the call graph: whether it *may write* its receiver (assigns a field of `this`, takes a pointer into it, calls a method that does) — this is what the discarded-copy error uses — and whether the receiver *may escape* (a closure captures `this`, `&this` or a pointer into `this` is taken, a callee does either). A receiver that may escape cannot stay on the caller's stack: its variable is heap-allocated, the same promotion `&x` performs (D10). This closes a hole the `mut fun` design had: a closure over `this` in a `mut fun` held a pointer to the caller's stack frame.

**Interaction with narrowing (D5).** `this` is a place like a local and its field paths narrow; a method call on a variable or on `this` drops the facts about paths through a `var` field and keeps those through bare fields, which no call can assign.

**Interaction with tasks (D35).** A sendable closure shares its captures by reference, so it may capture only `val`s of Sendable types *without `var` fields* (transitively, by value; collections and `Mutex` excluded as before) — another task could otherwise watch them change; the error names the capture. `this` is judged by the receiver's type. `async recv.m()` copies the receiver into the task, as it always copied arguments. Elements of an immutable `List` are read as copies, so `var` fields inside them are unreachable and do not count.

Rejected — Go's declared receiver (`(c Circle)` vs `(c *Circle)`): it declares representation, which produces two well-known failures. A value-receiver method that mutates silently modifies a copy with no diagnostic, and method sets diverge so that `Circle` does not satisfy an interface that `*Circle` does. Neither can arise here, because no representation is ever declared.

Rejected — Rust's `&self` / `&mut self`: enforced by a borrow checker Veles does not have, so the distinction would be decoration.

Rejected — private-by-default fields alongside `var`: with bare fields immutable, an exposed field is readable and nothing else, which is what a record is for; three visibility levels need one unmarked level and it must be the same for fields, methods and top-level declarations (M5).

*Addendum (v0.30) — `protected var`, and `val` written out.* Between "never assigned" and "assigned by anyone" sits the most common shape of managed state: a value everyone may read that only its type updates — C#'s `{ get; private set; }`, Swift's `private(set) var`, Kotlin's `var x` with a `private set`. Veles spells it `protected var`: `public protected var count: i64 = 0` is read wherever it is visible and assigned only inside the type's own declarations (methods, `implement` and `extend` blocks, the same set `private` uses). It replaces a `private var` plus a getter. The two modifiers answer two questions in order — the first word says who *sees* the field (`private` / nothing or `internal` / `public`), the rest who *assigns* it (bare or `val`: nobody; `protected var`: the type; `var`: anyone who sees it). `protected` always qualifies `var` (a bare field has nothing to protect; the compiler says so), and `private protected var` is refused as redundant — nobody outside can see the field, so `private var` already says it. It never restricts reading. For narrowing and for Sendable a `protected var` counts as `var` (the type can change it). The bare field may be written `val` for emphasis; the formatter keeps the author's word and never requires it.

On the word: `protected` means "the class and its subclasses" in Java, C#, C++ and Kotlin. Veles has no inheritance and will not get it — shared behaviour is a trait (D6/D9), closed families are sealed traits (D12) — so the subclass reading has nothing to attach to, and the word is free to mean what it says: protected from writes by anyone but the owner. The alternatives were weighed: `readonly` (TypeScript's and C#'s word for what Veles's *bare* field already is — it would invert the term for those readers, and it sits confusingly next to `val`), `var(private)` (not Veles syntax), Swift's `private(set)` (same). Making the protected level the *default* was proposed and rejected: it would take back the at-a-glance guarantee (a struct of bare fields cannot change) and make most `val` structs uncapturable by sendable closures.

*Addendum (2026-09-25) — a changed parameter is a lost change.* A struct parameter is the caller's value copied (D7), and `val`/`var` govern rebinding only, so a function may assign a `var` field of a parameter, or call a method that writes `this` on it — and the caller never sees it: `fun step(f: Fuzzer) { f.rng.next() }` advances a copy of the generator. This is a **warning**, on the parameter, naming the change and the fixes (`f: *Fuzzer`, or return the value). Silent when the copy is the point (the function returns the parameter's type, or uses the parameter whole after changing it), when the parameter's address is taken, and for writes through a reference field (seen by the caller). Rejected: Swift's immutable parameters, which would change the rule above for one case.

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
  fun depth(): i64 = this.items.len()
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

*Addendum (v0.30) — a default is a constant; derived fields are `init`'s.* A field default cannot read `this` (error, with the `init` form suggested). Defaults derived from earlier fields — `positions: List<i64> = this.toks.map(t => t.1)`, evaluated in declaration order on a partially built value, with a per-method field-use check for calls on `this` — were implemented and withdrawn the same day in favour of the `init` block below: two mechanisms for one job, and the declaration-order rule was a subtlety the block does not need (definite assignment orders things for you). What survives from that work is the receiver pass's field-use analysis, which `init` relies on.

*Addendum (v0.30) — the `init` block.* A struct body may contain one `init { }` block (Kotlin's `init`): it runs after every construction — the fields bound from the call and the defaults, in order — on the freshly built value, before anyone sees it. It is compiled as a hidden method the constructor calls (named `$init`, so no user code can name, call, hover or complete it). A field *without a default* that the block assigns (`this.f = ...`, found syntactically) is the block's: the constructor call refuses it, and the block must assign it on every path before it ends — definite assignment is tracked as flow facts in the narrowing state, so it joins at `if`/`when`, dies in loops and is discharged by a diverging branch exactly like a smart cast. Until such a field is assigned, reading it, using `this` as a whole, or calling a method whose field-use set includes it is an error (the receiver pass computes, for every method, the receiver fields it uses — directly, through the methods it calls on `this`, through closures over `this`; using `this` as a whole counts as every field); afterwards the block is ordinary method code. `init` cannot suspend, and has no `throws` (a failing construction is a `static fun ... throws`); it has no visibility, one per struct, runs for a sealed variant before the tag is applied. Spelled `init` rather than the proposed `constructor` because it is not the constructor: it cannot take parameters or replace the implicit call by fields, it only runs after it. `init` is contextual (`init { ` at member position), so a field or function named `init` stays legal. Deriving from a `var` field is a warning, since the stored copy would not follow later changes. The spelling is `this.f` rather than a bare `f` so that a default cannot be misread as a global. It composes with M5's constructor rule: a private field with a derived default cannot be overridden from outside, so the derivation is an invariant the type can rely on. Motivated by a parser holding token positions next to its tokens, which otherwise needed a `static fun` for nothing but the derivation.

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

*Known hazard:* a `when` branch returning a lambda reads `cond => x => x * 2`. It parses unambiguously — `=>` is right-associative — but it is hard to read. Accepted for now; a lint is the likely eventual answer. *(v0.60, D106: no lint — `veles fmt` prints it `cond => (x => x * 2)`.)*

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

`gather` is **heterogeneous and tuple-typed**: children may return different types, and the result is a tuple of `Result`s in launch order. *(v0.60, D103: with exactly one launch the result is that task's `Result` itself.)*

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

`race` is itself the suspension point, so arms carry no `await`. *(v0.61, D108: an arm may also be `ch.send(v)`.)*

*Addendum (v0.29).* Exactly one arm runs, so a smart cast made inside an arm (an assignment to a `var`) survives the race only when every arm makes it — the `if`/`else` join rule (D5). A race whose arms all return or throw is `Never`-typed, which is what lets `withTimeout` be written as `race { val r = await t => return try r; sleep(ms) => throw Timeout() }` inside a scope whose early exit cancels `t`.

*Addendum (2026-10-02, written down while building D108; not a change).* **Which arm wins.** When the race starts, the arms are tried in written order and the first one ready wins, so an always-ready arm shadows the ones after it. If none is ready the race waits, and the first arm to become ready wins. The rule is deterministic and biased towards earlier arms; it is not fair in Go's sense (Go picks among ready cases at random). Changing it would be a decision.

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

Veles has a GC and therefore no destructors, and no `defer` (rejected again in D100; the word stays reserved). *(v0.59)* D100 adds the statement form `with x = e`, whose body is the rest of the enclosing block, and `with t = async f()`. Before this, nothing in the language released a file descriptor, a socket, a lock, or a C allocation.

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

~~**This raises the stakes on §6.1.** The panic mechanism is still undecided, and~~ *(v0.59: stale — the mechanism is D49 and its v0.29 addendum; kept for history.)* `with` cleanup has to run during unwinding: under a hidden error-return path it is an ordinary code path, under DWARF it needs landing pads. Either works, but the choice is no longer purely internal.

`Mutex.withLock` predates this and takes a lambda instead. Whether it is rewritten as a `with` resource is an open stdlib question. *(v0.61: answered by D107 — `with p = m.lock()` beside `withLock`.)*

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

*Addendum (user decision 2026-09-18, built 2026-09-27) — the reason is written down.* An `unsafe { }` block in a function that is not `unsafe fun` carries a `// SAFETY:` comment saying why it is sound: on the lines directly above the line the block starts on, or as the first line inside the block (the place for a declaration's body, `fun pid(): i64 = unsafe {`, whose line above is its doc comment). Without one it is a **warning** (a lint, not a rule); the fix inserts `// SAFETY: ` and a comment with nothing after the marker still warns, so the fix cannot silence it. Blocks inside `unsafe fun` or inside another `unsafe` block are exempt — the obligation is already stated. Every block in the standard library carries a reason.

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
- **A cause field.** The one place a union appears outside `throws` is a field of an `error` (`error ConfigError { key: string, cause: PortErrors }`). This is how context wraps a cause (Go's `%w`, Rust's `source()`); `this.cause.message()` dispatches on the union like any other. Fields of plain structs may not hold unions.

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

*Noted pressure point (answered in v0.64 by D120):* `extern struct` does not extend. Packed layout, explicit alignment, and transparent single-field wrappers all come up in FFI and wire-format work, and there is currently nowhere to express them without inventing further keywords or revisiting this.

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

A flat event stream: the derive writes `key("id")` and then the value encodes *itself* (`this.id.encode(to)`), so the traits carry no generic method (a boxed trait object could not, D9) and no value is boxed on the way out; a format is one implementation of each trait, and `std/json`, a row decoder and an environment decoder all drive the same synthesized code. The encoder is a **trait object**, one virtual call per primitive — `Encodable` stays object-safe and the derive stays simple; a program that links one JSON encoder devirtualizes later. The error types are fixed, as D40 requires of a trait used as an object. Rejected: a JSON-specific `toJson()`/`fromJson()` (a derive per format, a tree allocation per encode); Swift's container objects (`field<T>` is a generic method, impossible on a trait object, and its non-generic form boxes every value); a format-neutral `Value` tree as the intermediate (an allocation per field); a generic `fun encode<E: Encoder>(to: E)` (direct calls, but `Encodable` is no longer an object and every format stencils the world).

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
(notes I1; the write-up with the alternatives is `archive/veles-guard-design.md`).

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

*Amended by D98:* the handler forms that bound the error, `r ?? { e => ... }` and
`val x = r else { e => ... }`, are removed; `r catch (e) { ... }` is the one
spelling that sees the error. `r ?? fallback`, `r ?? return`, and let-else
without a binding stay as written above.

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

### D65 — The receiver is `this` (v0.40)

A method refers to its receiver as `this`, as in Kotlin, TypeScript, Java
and C# (Rust, Swift and Python say `self`; the user chose Kotlin's word):

```veles
struct Counter {
  var n: i64 = 0
  fun bump() { this.n += 1 }
  implement Display { fun toString(): string = "Counter(${this.n})" }
}
```

Nothing else about the receiver changes — D22's calling convention, the
receiver pass, `init` blocks, trait default bodies and `extend` blocks
read `this` exactly as they read `self`. `Self`, the type, keeps its
name: it names a type, not a value, and has no counterpart in Kotlin to
borrow.

`this` was reserved; `self` is not free for identifiers either. The old
spelling still parses as the receiver so a file keeps checking, with an
error per use whose fix writes `this` (`veles check --fix`, the LSP's
quick fix), including inside `"$self"` and `"${self.x}"`. The VS Code
grammar highlights `this` and marks `self` as illegal.

### D66 — The executor: work-stealing M:N over one shared heap (v0.41)

D35 promised that tasks run on several cores; this fixes how. **Tasks are
scheduled M:N onto a pool of worker threads (one per core by default,
`VELES_THREADS` to override) and a suspended task may resume on any of
them** — Go's and Tokio's model. There is one garbage-collected heap;
collection stops every worker at a *safepoint* (an allocation or a
suspension point, both of which the compiler already knows) and marks
from all their roots. Each worker allocates from a buffer of its own so
the common allocation takes no lock. `Mutex<T>` becomes a real lock and
`Atomic<T>` real atomics; channels, timers and the I/O poller become
thread-safe inside the runtime.

Built in two stages so each is measurable in `bench/`: first N workers
on one global run queue, then per-worker queues with stealing.

What becomes an error: a module-level `var` whose type is not
synchronized (`Mutex`/`Atomic`) — two workers could write it at once,
and D35 rules races out at compile time. std's own (the UUID v7 counter)
move behind a `Mutex`. Nothing else in the language changes: `Sendable`
and the `await`-under-lock ban were designed for this.

Rejected: thread-per-core with pinned tasks (one hot connection starves
a core, and it still needs safepoints and cross-thread channels, so it
saves about a third of the work); staying single-threaded with an
offload pool and scaling by processes (breaks D35's promise, no in-process
parallel computation).

*Stage 1 built (v0.42).* What the decision left to the implementation:
a call to a foreign function (an extern outside std, or through an
`extern fun` pointer) runs in a *safe region* — the collector does not
wait for a thread blocked in C, and a callback from C leaves the region
for its Veles code. `Mutex` re-locked inside its own `withLock` panics
rather than hanging, and a panicking `withLock` function releases the
lock. `Atomic<T>` gains `update(f)`, the read-modify-write a
`load`/`store` pair cannot do without losing a concurrent store.
*Stage 2 built (v0.42).* Each worker has a run queue of its own that
idle workers steal half of, plus a shared queue; a task's scheduling
state is one word changed by compare-and-swap, so taking, running and
requeueing a task, spawning one, and a successful finish take no global
lock. Channels, races and timers stay one runtime structure under one
lock.
*Addendum (v0.42): word atomics and blocking calls.* An `Atomic<T>` whose
`T` is an integer, a float or `bool` takes no lock: `load`, `store`
and `swap` are single sequentially consistent instructions, and
`update(f)` loops on compare-and-swap — so under contention `f` may run
more than once, each time with the newer value, and must only compute
(Java's `updateAndGet` contract). A float compares by its bits. Other
types keep the lock word. The API does not change; the choice is made
per instantiation by std-only builtins (`atomicLockFree`, `atomicLoad`,
…), never visible to user code. A call that blocks its thread — a foreign
function, a terminal read, a child process, a contended `Mutex` — marks
the worker's run queue blocked; a monitor thread that finds the same call
still running a millisecond later, with work waiting (tasks queued, a
timer due, sockets to poll) and no worker idle, hands the queue to a
spare thread (Go's sysmon/P handoff). The thread returning from the call
learns by one compare-and-swap that it lost the queue, finishes its
task's step down to the next suspension (the frame is on its stack), and
waits as a spare. At most `VELES_THREADS` threads run Veles code at once;
threads blocked in calls are extra. The monitor sleeps once nothing has
blocked for 100 ms and is woken by the next blocking call.
*Addendum (v0.42): channels.* `Channel<T>()` — no capacity — is a
rendezvous: `send` completes when a receiver has taken the value (it had
behaved as capacity 1). A blocked sender or receiver waits with its value
slot in its node and the side arriving second copies across (Go's
sudog), so blocked senders of a full channel are served in order and
`tryRecv` takes from a blocked sender. A send or receive completed this
way has happened: a cancellation that lands after it is seen at the
task's next suspension point, so a value is never lost to it. Each
channel has a lock of its own; a `race` claims its winner by
compare-and-swap, so a value, a close, a timer and a finished task on
different threads cannot complete it twice. The runtime lock is left for
timers and socket waits.

### D67 — FFI: `extern` blocks anywhere, native libraries in the manifest (v0.41)

`extern "C" { fun ... }` may appear in any package (the checker never
restricted it to std; only the std used it). Every foreign function is
`unsafe`: a call needs an `unsafe` block, and an extern function cannot
be used as a value (so it cannot escape into a safe call later).
`extern struct` gives C layout; C function pointers are a type for
callbacks.

Native libraries are declared in `veles.toml`, not in source, and a
dependency's table links into every program that uses it:

```toml
[native]
libs = ["pq"]                 # -lpq: the shared library (or its import library)
static-libs = ["z"]           # the archive itself, so nothing is needed at run time
lib-paths = ["vendor/lib"]    # searched first; relative to veles.toml
pkg-config = ["libpq"]        # flags from `pkg-config --libs`
```

An entry with a directory part or a library extension (`"vendor/x.a"`,
`"shim.o"`) is linked as that file. `static-libs` resolves the archive
by name through `lib-paths` and clang's own library directories and
passes its path, which is what makes a link static on every platform
(a linker shown only `-lz` prefers the shared library).

*(2026-10-01, plan A8 — a bug fixed, no change of meaning.)* An `extern
struct` passed or returned by value follows the platform's C calling
convention — Windows x64, System V x86-64, AAPCS64 — in calls to C, calls
through `extern fun` pointers and exported `extern "C" fun`s
(`codegen/llvm/cabi.go`). Before, the struct was handed to LLVM as an
aggregate, which no C compiler expects: `lldiv` crashed on Windows.

Bindings are written by hand for now. A `veles bindgen header.h` that
writes the same `extern` blocks from clang's AST may come later; it
would generate this form, so nothing written by hand is thrown away.
Rejected: importing C headers into source at build time (Zig's
`@cImport`): clang work on every build, macros that do not translate,
and a compiler that depends on a C toolchain's AST format.

### D68 — Shutdown: a signal is something you wait for (v0.41)

```veles
// fragment
with (listener = try net.listen(port: 8080)) {
  http.serve(listener, app.handler(), stop: () => os.shutdownSignal(), grace: Duration.seconds(10))
}
```

`os.shutdownSignal(): os.Signal` suspends until the process is asked to
stop — SIGINT or SIGTERM on POSIX; Ctrl+C, Ctrl+Break, closing the
console, log-off or system shutdown on Windows — and says which
(`Signal.Interrupt`, `Signal.Terminate`). Nothing is intercepted until a
program first asks, so a program that never calls it keeps the default
(the signal ends it). After the first signal is taken, a second one has
the default effect again: a stuck shutdown can still be killed with a
second Ctrl+C. A signal that arrives while nobody waits is kept for the
next call.

`http.serve(..., stop:, grace:)`: when `stop` returns, `serve` stops
accepting, closes idle kept-alive connections, lets requests in flight
finish (their responses carry `connection: close`) for up to `grace`,
then cancels what is left — whose `with` blocks run (D43) — and returns.
Without `stop` it runs until its task is cancelled, as before. A stop
condition is ordinary code, so a test stops a server with a channel and
a program with two servers waits once and stops both.

Rejected: `os.onSignal(sig, callback)` (an unstructured entry point that
must reach shared state through `sendable` captures); `serve` handling
signals itself by default (hidden global behaviour that any second
server or other cleanup has to undo).

*Not taken with it (asked 2026-09-26):* making a std panic caused by the
caller (`xs.swap(0, 7)`) report the caller's line. It keeps reporting the
std line until stack traces exist (D64's known gap).

### D69 — FFI: memory crosses explicitly; C calls back through `extern "C" fun` (v0.42)

**Strings and buffers.** GC memory never becomes a `*raw` pointer that
outlives a call (D50's provenance rule stands). What crosses is either
copied into unmanaged memory or lent for the length of a closure:

```veles
// fragment
use ffi

with (name = try ffi.CString.of(path)) {      // malloc'd, NUL-terminated; `with` frees it
  unsafe { sqlite3_open(name.ptr(), &db) }      // `&db`: *T becomes *raw T inside unsafe
}
val version = unsafe { ffi.readString(zlibVersion()) }   // copies up to the NUL
val data = unsafe { ffi.readBytes(p, n) }                 // copies n bytes
bytes.withRaw(p => unsafe { write(fd, p, bytes.len()) })  // lent: no copy
```

`CString.of` refuses a string holding a NUL byte (`ffi.NulByte`): C would
read it as ending there, which is how a checked name becomes a different
one. `readString`/`readBytes` trust their pointer, so they are `unsafe
fun`s. `withRaw` hands C the list's own storage while the closure runs —
sound because the collector does not move objects and the list is alive
for the call; the pointer must not be kept past the closure, which the
type cannot express, so the closure's calls are `unsafe` by contract. Its
elements must be `CLayout` — a prelude marker answered from the shape,
like `Sendable`: numbers, `bool`, raw pointers, `extern struct`s and
`extern fun`s, what C reads as it lies in memory. `ffi.alloc(n)` /
`ffi.free(p)` are the raw allocator. `p as *raw T` reinterprets a raw
pointer (C's `void *`), and `&x as *raw U` gives C a Veles address for
the length of a call; both only inside `unsafe`.

**A value C hands back.** `ffi.handle(value)` lends a Veles value to C as
an opaque `void *` — an index into a table the collector scans, never a
GC address — and `unsafe ffi.Handle<T>.from(p)` gets it back in the
callback. It is `Closeable`; the value stays alive while it is open.

**Callbacks.** A function with a body in an `extern "C"` position is a
Veles function with the C calling convention:

```veles
// fragment
extern "C" fun byLength(a: *raw u8, b: *raw u8): i32 { ... }
unsafe { qsort(base, n, 8, &byLength) }
```

`&byLength` is a C function pointer of type `extern fun(*raw u8, *raw u8):
i32`. The body may not suspend or throw; a panic in it ends the process
with its location, since it must not unwind through C frames. Context
travels through the C API's `void *userdata` as a `*raw` value.

Rejected: implicit bridging of `string` to `char*` (a hidden allocation,
and a pointer C keeps dangles silently); an attribute spelling
(`@cabi`) — one word, `extern "C"`, marks the boundary both ways.

### D70 — `?.` short-circuits the rest of the chain (v0.42)

After `?.`, a null skips everything to the end of the postfix chain, so
`xs.ref(i)?.tags.push(x)` and `user?.address.city` work; the chain's type
is the last step's, made nullable (Swift's optional chaining). R8 had kept
Kotlin's rule — `?.` covers one step, `a?.b?.c` spelled out — and left the
question open; the chain is what such code means, and the extra `?.`s
said nothing a reader needed. The chain ends at the end of the postfix
expression: `(a?.b).c` is a separate, non-chained access to a nullable.

### D71 — Arithmetic operators on user types (v0.42)

`a + b`, `a - b`, `a * b`, `a / b` and `-a` on a non-numeric type call
`plus`, `minus`, `times`, `dividedBy` and `negate` — the names D60 chose
for `Duration` so that this could adopt them. They are prelude traits
(`Addable`, `Subtractable`, `Multipliable`, `Divisible`, `Negatable`)
whose right operand and result are associated types (`type Rhs`,
`type Out`), so the operand types may differ (`Timestamp + Duration →
Timestamp`), a generic can require them, and an implement is only the
method: an associated type an implement does not bind is read off the
method's own parameter or result type (a general rule, not one for these
traits). One implement per trait and type means an operator means one
thing per type — the difference of two `Timestamp`s stays
`t.since(earlier)`. `a += b` is `a = a + b`. What overflow does is the implementation's
business, as it is today for the methods. Comparison stays `Comparable`
(D57/P7). Rejected: only `+`/`-`; method calls only.

### D72 — Task-local values: scoped bindings that follow `async` (v0.42)

```veles
// fragment
val requestId = taskLocal("-")                       // T: Sendable
requestId.withValue(req.id, () => process(req))      // bound for f and every task started inside
fun log(msg: string) = io.println("[${requestId.get()}] $msg")
```

A request id, a trace context or a logger follows the work without being
a parameter of every function on the way. **A binding is scoped and
immutable** (Swift's `@TaskLocal`, Java's `ScopedValue`): `withValue(v, f)`
binds for the duration of `f` — ending however `f` ends: return, throw,
panic, cancellation — and a nested `withValue` shadows it until it ends.
**A task keeps what was bound where it started**: the bindings are a
list nothing changes in place, a new task (by `async`, or the task a
suspending call runs as) starts from its creator's head, and a later
rebinding in the creator does not reach it. `get()` outside every binding
returns the fallback given to `taskLocal`. The value is read on other
threads, so `T: Sendable`; a `TaskLocal` of a Sendable value is Sendable.
`withValue` takes a `suspends` function, so it is refused where nothing
may suspend (a `withLock`, a lambda for `List.map`).

Cost: a pointer copied when a task starts; `get` walks the bindings in
effect (a handful). Rejected (user, recommended of three): a mutable
task-local (`set`/`get`, ThreadLocal-style — a helper that sets it changes
what its caller sees, and a value nobody resets carries over); nothing in
the language, an explicit context parameter (Go's `context.Context` —
every function on the path carries it).

### D73 — `init` takes parameters: every value is `Type(...)` (v0.43)

```veles
// fragment
public struct Mutex<T> {
  private cell: *T                    // assigned by init: not a parameter
  private word: LockWord = newWord()
  init(value: T) { this.cell = &value }
}
val counter = Mutex(value: 0)
val jobs = PriorityQueue<Job>(compare: (a, b) => a.due.compareTo(b.due))
```

The user's note: a factory function "should not be the required way" of
making a value. D28's implicit constructor covers a struct whose state is
its fields; a type that does work at construction — a lock word to
allocate, a value to put behind a pointer, a comparator to choose — needed
a function beside it (`mutex(0)`, `priorityQueueBy(cmp)`), and a reader had
to know it existed. **An `init` block may declare parameters**; they are
parameters of the implicit constructor, after the field parameters, named
at the call like fields (D28: by name, a bare argument only as a pun), with
defaults allowed. Inside the block they are ordinary `val`s. Everything
else about `init` holds (D28 v0.30 addendum): fields it assigns are not
parameters and must be assigned on every path, it cannot suspend or throw.
A parameter may not share a name with a field the call can pass (the call
would be ambiguous). A type whose natural construction needs a bound the
constructor cannot state (`T: Comparable` for a natural-order queue) keeps
a `static fun` for that one form: `PriorityQueue<i64>.natural()`.

Factories removed with it: `mutex(v)` → `Mutex(value: v)`, `atomic(v)` →
`Atomic(value: v)`, `taskLocal(f)` → `TaskLocal(fallback: f)`,
`priorityQueueBy(c)` → `PriorityQueue<T>(compare: c)`, `priorityQueue<T>()`
→ `PriorityQueue<T>.natural()`; like `stringBuilder()`, `deque<T>()`,
`http.router()` and `Depth.of(n)` before them (constructors since M5 v0.30
was implemented across modules, 2026-09-27), each old call is an error
naming the new one, with a fix where the arguments carry over.

Rejected (user, recommended of three): a `static fun of` convention
(`Mutex.of(0)` — still a function to know about); keeping the lowercase
factories.

### D74 — HTTP statuses, methods and headers are named values (v0.43)

```veles
// fragment
throw http.Fail(status: http.Status.notFound, "no such note")
if (req.method == http.Method.post) return Response.json(body, status: http.Status.created)
resp.withHeader(http.Header.cacheControl, "no-store")
val teapot = http.Status(code: 418)
```

The user's note: "the enums / mappings / consts should be a norm in std,
like statuses, methods etc in http". A status, a method and a header name
are **open** sets with well-known members — a client must carry a status
it has never heard of, a server may answer WebDAV's `PROPFIND` — so they
are value types with `static val` constants (D23 v0.29), not enums (D57 is
for closed sets). `Status { code: i64 }` has `reason()`,
`isInformational/isSuccess/isRedirect/isClientError/isServerError()`,
prints as `404 Not Found`, compares and hashes by code. `Method { name:
string }` constants for the nine RFC 9110 methods plus `PATCH`; it prints
as its name. `Header` is a namespace of lower-case name constants
(`Header.contentType == "content-type"`), since headers are looked up by
string. Every public `status: i64` and `method: string` in `std/http`
becomes `Status` / `Method`; `http.call(handler, Method.get, "/")`.
Constants are lowerCamel like every `static val` (`Status.notFound`).

Rejected (user, recommended of three): closed enums (an unknown code is
unrepresentable, and an enum has no methods); integer/string constants
only (no type to stop `status: 44`).

### D75 — The prelude holds what almost every program uses (v0.43)

The user's note: "carefully think about what should be a global-available
built-in and what should be in its own module". Global: the primitives
and their methods; `List`/`Map`/`Set`/`Deque`/`PriorityQueue`;
`Option`/`Result`; `Error`, `IoError`, `Panic`; the core traits
(`Comparable`, `Equatable`, `Hashable`, `Display`, `Parsable`, the
arithmetic traits, `Iterator`/`Iterable`, `Closeable`, `Sendable`);
`StringBuilder`, `Duration`, `Mutex`, `Atomic`, `TaskLocal`,
`withTimeout`, `Timeout`; `Codable`/`Encodable`/`Decodable` with
`EncodeError`/`DecodeError`. **Moved to modules**: the codec machinery
(`Value` and its variants, `Encoder`/`Decoder`, `ValueEncoder`/
`ValueDecoder`, the style enums, `Problems`/`Problem`, the path helpers) to
`use codec`; `CLayout` to `use ffi`; `Depth`, `maxRecursionDepth`,
`tooDeepMessage` to `use recursion`. Derived code reaches the codec
helpers without an import (`ast.PreludeName` resolves into the module), so
`implement Codable` still needs none. Rejected (user, recommended of
three): a Go-minimal prelude (every program would start with `use sync`,
`use time`); leaving about 70 global names, a third of them plumbing.

### D76 — `IoError.kind`: which failure, portably (v0.44)

```veles
val text = fs.readFile(path) else { e =>
  if (e.kind == IoKind.NotFound) return "{}"
  throw e
}
```

`IoError` gains `public kind: IoKind`, an enum in the prelude beside it:
`NotFound`, `PermissionDenied`, `AlreadyExists`, `NotADirectory`,
`IsADirectory`, `DirectoryNotEmpty`, `ConnectionRefused`,
`ConnectionReset`, `ConnectionAborted`, `TimedOut`, `AddressInUse`,
`AddressNotAvailable`, `BrokenPipe`, `Interrupted`, `InvalidInput`,
`InvalidData` (text that is not UTF-8), `Other`. The runtime maps the platform's number (errno; a Winsock code on
Windows) to the kind; `code` stays, for logs and for what the enum does
not name (`Other`). One error type, so no `throws` set changes. User's
note #5: enums, not magic numbers, are the norm in std. Rejected (user,
recommended of four): separate error types per failure (every `throws
IoError` widens); predicates (`e.isNotFound()`, not exhaustive); as is.

### D77 — `DateTime.weekday()` is a `Weekday` (v0.44)

`public enum Weekday { Monday = 1 … Sunday = 7 }` in `std/time`, ISO
numbering (was `i64`, 0 = Sunday). `when (d.weekday())` is exhaustive;
`.value` gives the ISO number. `month` stays an `i64`: constructors and
arithmetic (`month + 1`) read better with a number. Rejected (user,
recommended of three): a `Month` enum as well; as is.

### D78 — Tests are `test "sentence" { }`, with a test-only vocabulary (v0.44)

The user: assertions "should only be usable inside test function"; "do we
need '@test' … maybe we could add "test" keyword". Investigation in
`archive/veles-testing-design.md`; the user took its recommended combination.

```veles
test "parses a port" {
  expect(parsePort("80") == 80)                 // soft: recorded, the test goes on
  val cfg = require(load("app.toml"))           // hard: ends the test; unwraps T? / Result
  expectThrows<RangeError>(() => parsePort("70000"))
  expectPanics(() => [1].at(5) ?: panic("no"))
}
```

- **Declaring.** `test "name" { body }` at top level; `test` is contextual
  (only `test "` starts one, so a function or value named `test` stays
  legal). No signature: it cannot be called or take parameters; it may
  throw and suspend without saying so (inferred like a lambda). Two tests
  with one name in a module are an error; the name is a plain string (no
  interpolation). `veles test --filter` matches the name. `@test fun`
  keeps parsing, with an error and a fix to the new form.
- **Vocabulary**, compiler-known and in scope only in test code (a test
  block, a `test fun`, a `*.test.vs` file); elsewhere the names are
  unknown with a hint. `expect(cond)` captures the expression: its source
  text and, for a comparison, both sides' values; a failure is recorded
  and the test goes on. `require(x)` unwraps a `T?` or a `Result` and
  ends the test when there is nothing. `expectThrows<E>(f)`,
  `expectPanics(f)`, `fail(why)`. Every failure carries the call's
  location.
- **Helpers.** `test fun name(...)` is callable only from test code, may
  use the vocabulary, and is left out of `build`/`run`. Every declaration
  in a `*.test.vs` file is test code: it sees the module's private names
  and is never loaded by `build`/`run`.
- **Invariants.** ~~`check(cond)`~~ `assert(cond, "why")`, anywhere (amended
  2026-09-27, the user: "the everywhere available 'check' should be called
  'assert' (and it should check condition and require string explanation
  like panic)"). The reason is required and evaluated only on failure; the
  panic is the reason, then the condition and, for a comparison, both
  sides. A bare `check(...)` with no declaration of that name is an error
  naming `assert`.

*Built 2026-09-27.* A test lowers to a function with a bare `throws`
(nobody wrote the clause, so D45's "needless throws" lint skips it); its
name is the sentence, its symbol none. `expect(a OP b)` binds each side
once, in source order (a literal side takes the other's type, as in the
comparison), and a failure prints `  file:line:col: expect(...)` then
`left:`/`right:` (strings quoted; a literal side as written). The runtime
collects failures per test (`veles_test_fail`/`veles_test_take`, locked:
tasks on any thread may record); `require`/`fail` record and then panic,
and the runner shows their line instead of the panic. `expectThrows`
refuses at compile time a body that cannot throw, or cannot throw the
named `E`. `expectPanics` launches the body through a private prelude
`runTestBody` inside `gather` (D52), so the body must be sendable.
`assert`'s panic is its reason, then `assert(cond)` and the detail lines.
`*.test.vs` files are dropped before checking in `build`/`run`; calling
test code (a `test fun` or anything in a test file) from other code is an
error, by name or `module.name`.

*Amended 2026-09-27 — suites.* The user: "I think both is the answer" —
`suite "name" { ... }` holds tests, nested suites and `test fun` helpers
(visible only inside the suite, so two suites may each have their own),
and a `*.test.vs` file is a suite named after the file. A test's full name
is its suites' names and its own joined with ` / `; the summary lists it,
`--filter` matches it, and names must differ only within one suite. The
report is grouped (the user's choice over one qualified line per test): a
heading per suite, its tests indented under it, nested suites further in,
tests outside any suite first. A suite holds nothing else: setup that
runs before and after each test was proposed as the suite's own `val`s
and `with`s and rejected — "the setup for before and after isn't good...
(no new keyword for them either)" — and is open (checklist §9 Q2).

Rejected (user, recommended combination): `test fun name()` as the test
form; a `testing` module anyone can import (leaks into programs, needs
`try`, loses the location); a built-in `assert` everywhere.

### D79 — A diagnostic belongs to a named family; `veles explain <family>` (v0.45)

Every diagnostic the compiler reports belongs to a **family** with a
readable name — `private-to-module`, `missing-return`, `not-sendable` —
which is also the heading anchor of its section in
`docs/documentation/reference/errors.md`. No numbers.

```
main.vs:5:15: error: 'helper' is private to module 'geo'; declare it 'public' there to use it from here (M5)
    val _ = geo.helper()
                ^^^^^^
see: veles explain private-to-module
```

- The command line prints the `see:` lines after the diagnostics, one per
  family that occurred, not one per diagnostic.
- `veles explain <family>` prints that section from the copy of
  `errors.md` compiled into the binary: it works offline and matches the
  compiler in hand. An unknown name is answered with the nearest family
  names ("did you mean").
- The language server sends the family as the diagnostic's `code` and a
  link to the same anchor of `errors.md` on GitHub as `codeDescription`,
  so an editor shows a clickable link.
- Every diagnostic format belongs to a family, which a test enforces the
  way the conformance suite enforces a case per diagnostic.

Rejected (user, recommended of 4): a GitHub URL printed with every error
(needs a network, follows `main` rather than the compiler in hand);
Rust-style numbered codes `error[V0105]` (opaque, a numbering to keep
forever); no link.

### D80 — `veles test` runs tests in parallel by default (v0.45)

Tests run as tasks across the runtime's threads; `veles test --jobs N`
bounds how many run at once, and `--jobs 1` runs them one at a time, in
declaration order, as before. The report does not change with it: each
test's lines — its result, its recorded failures, its captured output —
are printed together, in declaration order, whatever order the tests
finished in.

Parallel is a safe default here rather than a gamble because the language
already rules out data races between tests: a module-level `var` is
behind a lock (D66), and only Sendable values cross between tasks (D35).
What remains is logical interference — two tests resetting one counter,
binding one port — and `--jobs 1` is the way out; the watchdog's message
for a test that outlives its bound says so.

Rejected (user, recommended of 4): sequential by default with `--jobs N`
to opt in (Go's model; nobody gets the speed without asking); a
`@serial` attribute per test or suite (more vocabulary, for a need
`--jobs 1` meets); leaving tests sequential.

### D81 — A panic prints its call chain in debug builds, from a shadow stack (v0.45)

```
panic: index 7 out of bounds for list of length 3
  at main.vs:12:5 in parse
  called from main.vs:30:9 in load
  called from main.vs:41:3 in main
```

In a debug build every call of a Veles function records, in a small stack
belonging to the current task, the callee and the line it is called from,
and removes it on return; a panic prints that stack under its `at` line.
The stack is the task's, not the machine's, so the chain is complete
through suspension and across threads, and it is the same on every
platform (WebAssembly included). A release build keeps no stack and
prints the panic's own line, with a note that a debug build shows the
chain.

Rejected (user, recommended of 3, after the two were explained): DWARF
unwinding and line tables at the panic (D49's route) — zero cost in
normal execution, but a decoder for three binary formats in the runtime,
a second walker for suspended coroutine frames (most server code
suspends), debug info in release binaries, and nothing for WebAssembly;
it stays possible later for release traces. A shadow stack in release
builds too (a cost on every call, in the build that matters for speed).

### D82 — `os.run` takes input and captures standard error (v0.45)

```veles
val r = try os.run("git", ["apply", "-"], input: patch)
if (!r.ok()) io.eprintln(r.stderr)
```

`os.run(program, args, input: string = "", stderr: os.Stderr = os.Stderr.Capture)`.
`input` is written to the program's standard input, which is then closed.
`os.Stderr` is `Capture` (into `Output.stderr`), `Inherit` (to this
process's standard error, as before) or `merge` (into `Output.stdout`, in
order). `Output` gains `stderr: string`. `mergeStderr:` is removed; it
still parses, with an error whose fix writes `stderr: os.Stderr.Merge`.

Rejected (user, recommended of 3): two more booleans beside `mergeStderr`
(flags that interact); a `Command` builder (more API for one call).

### D83 — `MutableList.reserve(n)` (v0.45)

`xs.reserve(n)` makes room for at least `n` elements in total, so pushing
up to `n` does not grow the list again. It never shrinks and never
changes the elements.

Rejected (user, recommended of 4): `MutableList<T>.withCapacity(n)`;
both; neither.

### D84 — An associated type two bounds declare is named through its trait (v0.45)

```veles
fun f<T: HasItem + AlsoItem>(x: T.AlsoItem.Item) { }
```

When more than one bound of a type parameter declares an associated type
of the name, `T.Item` is an error that lists the traits and the
qualified form; `T.Trait.Item` names the one meant, and is allowed
whenever `Trait` is a bound of `T` (with one bound it is merely longer).
Before, `T.Item` silently meant the first bound's.

Rejected (user, recommended of 3): refusing without a qualified form (the
traits would have to avoid each other's names); keeping the first bound.

### D85 — Named imports: `use m { f, T as U }` (v0.46)

```veles
use io { println, eprintln as warn, readLine }, http { Request, Response }

fun main() {
  println("hello")            // bare
  warn("to standard error")   // renamed with `as`
  io.println("qualified")     // the module keeps its name
}
```

A `use` entry may end in a brace group naming members of the module: a
function, a type (struct, enum, sealed type, trait) or a `static val`. Each
name is bound bare in the file, as the module's own symbol; `as` renames it
the way it renames a module. `m.` stays available — braces add names, they
never replace the qualifier. The old `use m.{ … }` (M6, removed by R9) is an
error whose fix drops the dot.

Rules: no `{ * }` (an error saying to name what is used); a name may not be
imported twice in a file nor equal a declaration of the importing module
(the message says to rename with `as`); a parameter or local of the same name
shadows an import, as it shadows a module; an explicit import wins over a
prelude name; a private member or test code is refused as when qualified; a
name nothing uses is a warning whose fix removes it; the formatter sorts
names inside the braces; completion inside the braces offers the module's
names, bare completion offers the imported ones, and typing the start of a
name a module offers but the file has not imported completes it together
with the edit that imports it. A renamed name is a declaration of its own at
the alias (hover: `alias of m.f`; rename changes the alias and its uses, not
the member), an unrenamed bare use hovers with its module. `as` inside the braces
does not depend on Q16 (what `as` means for conversions).

Rejected (user, recommended of 4): `from m import { … }` (a second import
syntax, two new keywords), `::` and `:` for renaming (a new operator; a colon
already means "type of" and "name of argument"), `use m.{ … }` (the removed
spelling); braces that replace the qualifier (`use io, io { println }` to have
both).

### D86 — Conversions are methods split by risk; `as` only renames (v0.47)

```veles
val i = b.toI64()             // u8 → i64 cannot lose: an i64
val small = n.toU8()          // i64 → u8 may lose: a u8?, null when out of range
val low = (acc >> bits).wrapU8()   // keep the low 8 bits — truncation, spelled out
val x = count.toF64()
val p = raw.cast<*raw Header>()    // raw pointer to raw pointer, unsafe only
```

`x as T` no longer converts. Every numeric conversion is a method named
`to<T>` or `wrap<T>` on the numeric types (`i8 i16 i32 i64 u8 u16 u32 u64 f32
f64`); `as` is left to renaming (`use m as x`, D85).

- `toT()` — on an integer receiver whose whole range fits `T` (or float
  widening `f32.toF64()`, and integer to float) it is total and returns `T`;
  where a value could be lost it returns `T?`, null when out of range. Float
  to integer truncates toward zero and gives null for NaN or an out-of-range
  value. Integer to float rounds to nearest (`i64`/`u64` above 2^53 may
  round; that is the meaning of a float, not a lost range). `f64.toF32()`
  is total and rounds.
- `wrapT()` — integer to integer only, always total, keeps the low bits
  (two's complement), the conversion `+%` is to `+`. Available from every
  integer type, including where `toT()` is total.
- A literal that does not fit is a compile-time error (`300.toU8()`);
  the constant forms `toT()` of a literal are folded.
- `p.cast<*raw U>()`: reinterpret a raw pointer; requires `unsafe` (D44).
  `&x` to `*raw U` is `(&x).cast<*raw U>()` inside `unsafe`.
- `x.wrapTo<T>()` (addendum, Q19): `wrapT()` with the target as a type
  argument, so generic code can convert to its own type parameter
  (`(wide - r).wrapTo<T>()`); both sides must be integers once `T` is known,
  checked where the function is instantiated. Rejected (user, recommended of
  4): an `Integer` trait with `T.wrap(from:)` (later, if bounds are wanted),
  keeping a std-only `as` exemption, rewriting `Range.reversed()` without a
  conversion.
- An enum is still not its number (D57): `.value`, `E.fromValue(n)`.
- The old `x as T` keeps parsing with an error naming the method to write
  (`sema/removed.go`), with a fix.

Both levels: `toI64()` reads as the safe default and never surprises;
truncation and pointer reinterpretation are visible words in the source.
Self-hosting: a lexer/hash/emitter is byte↔int code; the lossless forms are
free and every wrap is greppable.

Rejected (user, recommended of 5): keeping `as` for conversions (A), `as`
for lossless only (E), call-style `i64(x)` (B), an operator `to`/`cast` (C).
Naming (user, recommended of 3): `toU8()` checked + `wrapU8()`, over
`truncateU8()` and `u8OrNull()`.

### D87 — `protected` on a mutable-collection field: look, don't take (v0.48)

```veles
struct Bag {
  public protected var items: MutableList<i64> = []
  public fun count(): i64 = this.items.len()   // the type's own code: unrestricted
}

fun main() {
  val b = Bag()
  b.items.len()          // fine: an immutable-form method
  loop (x in b.items) { }   // fine: a loop head
  "${b.items}"           // fine: interpolation
  b.items.push(9)        // error: 'items' is protected: only Bag changes its contents
  val taken = b.items    // error: cannot take 'items'; copy it with 'b.items.toList()'
}
```

`protected var` already meant "assigned only by its type" (D22). For a field
whose type is a mutable collection (`MutableList`, `MutableMap`, `MutableSet`,
`Deque`) it also means the collection's *contents* are the type's:
outside the type's own declarations the field may only be the receiver of the
immutable form's methods, a loop head, or interpolated; a mutating method,
binding it, passing it and returning it are errors, keeping D25's no-views
rule. `.toList()` / `.toMap()` copy and are unrestricted. A bare or `val`
field of a mutable collection is unchanged (the documented reference
semantics of D25/D41).

Rejected (user, recommended of 3): leaving it (accessors on a `private`
field still work), read-only view types (D25/D35).

### D88 — `@caller_location`: a misuse panic points at the caller (v0.48)

A function marked `@caller_location` receives its call site as a hidden
argument, and a `panic(...)` written directly in its body reports that
location as its `at` instead of the panic's own line (the panic's own line
stays the innermost frame of the call chain in debug builds). Both profiles:
release prints the caller's `file:line:col` where it used to print
`std/prelude/list.vs:408`. Std only for now (the attribute is refused in
user code with a message saying it is reserved; making it public is a later
decision). Marked: the prelude's misuse panics — `swap`, `set`, `insert`,
`removeAt`, `chunked`, `windowed`, `step`, `pow` and the like.

Rejected (user, recommended of 3): leaving it (debug's `called from` only),
automatic caller reporting for every std panic (needs the frame chain in
release).

### D89 — Package surface in source: `public use` (v0.48)

```veles
// mathlib/lib.vs — the root module
public use geometry                      // outsiders: use mathlib.geometry
public use shapes { area, Circle as Round }   // flattened: use mathlib → mathlib.area
```

`public use` in any module adds the named module, or the named items, to that
module's public surface; the root module's public surface is the package's,
and the manifest's `exports` list goes away (the bootstrap has no users to
keep compatible; a manifest that still has one gets an error naming the fix).
Forms mirror `use` (D85): `public use m` re-exports the module under its
name, `public use m { a, T as U }` flattens those items into this module
(`use mathlib` then gives `mathlib.a`), `as` renames. Only the package's own
modules and items may be re-exported; re-exporting a dependency is an error
for now (revisit with M7). `public use` is a `use` edge for import-cycle
detection; a flattened name collides with a declaration of the module as a
duplicate does. The dependency graph still comes from manifests; the loader
reads a dependency's root module to learn its surface, and the error for a
module that is not re-exported names the `public use` to add. Hover and
go-to-definition follow a re-export to the original.

Rejected (user, recommended of 3): keeping manifest `exports`, whole modules
only.

### D90 — `lazy` parameters (v0.49, std-only for now)

```veles
public fun debug(lazy msg: fun(): string, fields: Field...) {
  if (!enabled(Level.Debug)) return
  emit(Level.Debug, msg(), fields)
}

log.debug("loaded $n rows from $path")   // the string is built only if Debug is on
log.debug(() => expensiveDump())         // a lambda is passed as it is
```

A parameter marked `lazy` must have a zero-argument function type,
`fun(): T`, that does not suspend or throw. At a call, an argument that is
not itself a lambda is wrapped in one — `f(expr)` is `f(() => expr)` — so it
runs only if, and each time, the function calls `msg()`. The wrapped
expression cannot suspend or throw (it runs inside the callee). The
modifier is refused outside the standard library for now ("reserved for the
standard library"); lifting that, and a sugar that lets the body read `msg`
without the call, are open. Swift's `@autoclosure` is the model: the
parameter stays visibly a function.

Rejected (user, recommended of 3): an `@lazy` attribute spelling; a compiler
special case for `std/log` alone.

### D91 — `std/log` (v0.49)

```veles
use log { field }

log.info("served", field("path", req.path), field("ms", 3))
log.debug("cache miss for $key")                 // `lazy`: built only when Debug is on
log.setLevel(Level.Warn)                          // default Info; VELES_LOG=debug|info|warn|error|off
log.withFields([field("id", id)], () => handle(req))   // task-local: children inherit
```

Levels are `enum Level { Debug, Info, Warn, Error }` (no Fatal: a panic
already stops the task, D56), with functions `debug info warn error`;
`Level.Off` is a fifth, threshold-only member (`VELES_LOG=off`), and
`log.enabled(level)` says whether a message would be logged — the guard for
calls whose `field(...)` arguments are expensive, since those are evaluated
at the call (only the message is `lazy`).
`field<T: Encodable>(key, value): Field` keeps the value's JSON type.
Output goes to standard error, one line per call written whole under a lock
(threads do not interleave): on a terminal, text —
`2026-09-29T10:15:03.123Z INFO  served path=/x ms=3 id=abc` (the level padded to five) — otherwise one
JSON object per line —
`{"time":"…","level":"info","msg":"served","path":"/x","ms":3,"id":"abc"}`.
`withFields` binds fields in a task-local (D72) for the task and the tasks
it starts; `http.logging()` and `requestId()` use it, so a handler's lines
carry the request id. The level comes from `VELES_LOG` at startup or
`setLevel`.

Rejected (user, recommended of 3, 3 and 2): the lambda-only call, a manual
`enabled` guard; a map of fields, a derived struct per message; text only,
a pluggable sink trait (later, if wanted); an explicit `Logger` value
threaded through calls.

---

### D92 — Stack size and stack overflow (v0.50)

```veles
fun depth(n: i64): i64 {
  if (n == 0) return 1
  val r = depth(n - 1)
  (r ^ (r << 1)) + n % 7
}
// depth(1_000_000) runs; depth(1_000_000_000) prints
//   panic: stack overflow
//     the stack is 256 MB; VELES_STACK=<megabytes> sets it
//     in depth
//     called from main.vs:5:11 in depth
//     ...
// and exits with the panic exit code, 101
```

Found by the self-host audit (plan S3, `veles-selfhost-frontend-plan.md`
§9): a recursion that is not a loop killed the process silently — exit code
127 on Windows at 30–50k frames (1 MB stack), a bare SIGSEGV on Linux at
200k–1M (8 MB) — with no message, and tasks ran on worker threads' stacks
of the same OS default.

**Size.** Every thread that runs Veles code has a stack of 256 MB, reserved
and not committed (pages appear as the recursion reaches them; address space
only — eight workers reserve 2 GB), or `VELES_STACK` megabytes (1 to 4096;
anything else is reported once and ignored). A refused reservation is
retried at half, down to 8 MB. Worker threads get it from
`veles_thread_spawn`; the runtime's own helper threads (the timer monitor, the
mutex watchdog) keep the OS default.

**The main thread.** The process's `main` belongs to the runtime
(`veles_stack.c`); the generated one is `veles_main`. It starts the program
on a thread with the big stack and waits: the main thread's own stack is
whatever the OS gave the process and cannot be sized from inside, and a
program that behaved differently on the thread that happened to run `main`
would be worse than one that behaves the same everywhere.

**The panic.** A fault handler recognises the guard region (a SIGSEGV or
SIGBUS on a per-thread alternate stack on POSIX; a vectored exception
handler for `EXCEPTION_STACK_OVERFLOW`, with 64 KB reserved for it, on
Windows) and prints in the style of D64/D81, using no allocation and no
formatted output, since it runs on the exhausted stack: `panic: stack
overflow`, the stack size, and in a debug build the innermost calls of the
D81 chain (`in f`, `called from site in caller`, three of them, and how deep
the recursion was — the chain keeps the first 4096 calls); a release build
says a debug build shows the chain. Exit code 101 as any panic that ends the
process. It is not catchable and does not unwind — there is no stack to
unwind on — so `recursion.Depth` (D75) stays the way a walk over input
turns depth into an *error*.

Not covered: a single frame larger than the guard region (a huge local
array) can jump the guard and fault elsewhere — clang's stack probes would
close it; a fault that is not near the guard is an ordinary crash, as before;
macOS and other threads not created by the runtime (a C library's callbacks
on its own thread) are untested or outside it.

Rejected (user, recommended of 3, 3 and 3): compiler-inserted stack probes
(a check on every call for a failure a guard page already reports); leaving
the failure silent with `recursion.Depth` as the only answer; 64 MB (a
million frames in a small function, against the same address-space bill);
the OS defaults with only the diagnostic; raising `RLIMIT_STACK` at startup
(unverified against the kernel's mmap layout); leaving the main thread at 8
MB (recursion depth would depend on which thread the task landed on).

---

### D93 — Float bit patterns: `toBits` / `fromBits` (v0.51)

```veles
val bits = (1.5).toBits()            // u64: 0x3FF8000000000000
val x = f64.fromBits(bits)           // 1.5
val single: f32 = 1.5
single.toBits()                      // u32: 0x3FC00000
f32.fromBits(0x40490FDB)             // 3.1415927
```

Found by the self-host audit (plan S3): a code generator writes an LLVM
`double` constant as its exact hexadecimal bit pattern, the way the Go
compiler does with `math.Float64bits`, and until now the only spelling was an
`unsafe` cast of a local's address. `x.toBits()` on `f64` and `f32` gives the
IEEE 754 pattern as `u64` and `u32`; the static `f64.fromBits(bits)` and
`f32.fromBits(bits)` give the number back. Nothing is rounded or
canonicalised: a NaN keeps its payload (a signalling NaN stays signalling as
far as the hardware allows), `-0.0` and `0.0` differ, and the pair round-trips
every pattern. Every pattern is a number, so `fromBits` cannot fail.

Implemented in `std/prelude/number.vs`, in Veles, over the same `unsafe`
reinterpretation the user would have written, with its SAFETY comments; no
compiler change. At `-O2` LLVM turns the local into a `bitcast`; a debug
build pays a store and a load. A builtin lowered to `bitcast` can replace it
later without changing what a program means. The spelling is Rust's
(`to_bits`/`from_bits`), close to Kotlin's `toRawBits` and Go's
`Float64bits`.

Rejected (user, recommended of 3, 2 and 3): instance methods both ways in the
D86 conversion style (`x.toBitsU64()`, `bits.bitsToF64()` — float-only methods
on every integer); leaving it to `unsafe`; a compiler builtin for now.
Deliberately not included, and left for a later decision (checklist Q20): the
float sign and neighbour helpers (`copySign`, `signBit`, `nextUp`,
`nextDown`) and the integer bit operations that do not exist yet
(`rotateLeft`, `rotateRight`, `byteSwap`, `reverseBits`; `countOnes`,
`leadingZeros` and `trailingZeros` already do).

---

### D94 — `std/http`: cookies and forms (v0.52)

```veles
struct Signup { name: string, age: i64, newsletter: bool = false, tags: List<string> = [] }

app.post("/signup", req => {
  val s = try req.form<Signup>()                    // 400 naming every bad field; 415 for another content type
  http.Response.redirect("/welcome", status: http.Status.seeOther)
    .withCookie(http.Cookie(name: "sid", value: newSession(s), maxAge: Duration.days(30)))
})
app.get("/me", req => http.Response.text(req.cookie("sid") ?: "nobody"))
app.get("/find", req => { val q = try req.query<Search>(); ... })   // /find?q=veles&tags=a&tags=b
```

The first of the C2 tasks (plan C, checklist §5.2), taken in the order the
user chose: cookies and forms, then static-file caching, then the body model
(chunked, streaming, multipart, `Expect: 100-continue`), then limits and
middleware.

**Cookies.** `Response` gains `cookies: List<Cookie>`, written as one
`Set-Cookie` line each — a header map holds a name once and a response may set
many. `resp.withCookie(c)` and `resp.withoutCookie(name, path:, domain:,
secure:)` (a `Max-Age=0` cookie) add to it. `Cookie` has `name`, `value`,
`path = "/"`, `domain = null`, `maxAge: Duration? = null` (a session cookie),
`secure = false`, `httpOnly = true`, `sameSite: SameSite? = Lax`, with `enum
SameSite { Lax, Strict, None }`. The defaults are the safe ones; `secure`
stays off so a program served over plain http while it is written works.
The value is percent-encoded on the way out (every byte but `A-Za-z0-9-._~`
as `%XX`) and decoded on the way in, so any string round-trips and none can
end the header early. What cannot be encoded is checked, and a bad one
panics at the caller's line (`@caller_location`, D88) — a handler's panic is
already a 500 with a log line: a name that is not a token, a path that does
not start with `/` or holds `;` or a control character, a domain that is not
a host name, a negative `maxAge` (use `withoutCookie`), `SameSite.None`
without `secure` (browsers drop it), a `__Host-` name that is not secure, at
`/` and without a domain, a `__Secure-` name that is not secure.
`req.cookies(): Map<string, string>` and `req.cookie(name): string?` read the
`Cookie` header (quoted values unquoted, percent sequences decoded, the first
of a repeated name winning). The server keeps one header per name, so a
client that sent two `Cookie` lines has the last one read — HTTP/1.1 clients
send one.

**Forms and query strings.** `Request.rawQuery` keeps the query text as it
came. `Fields` (`Fields.parse(text)`, `get(name)`, `all(name)`, `names()`) is
the ordered list of pairs with repeats kept; `req.queryFields()` and
`req.formFields()` (an `application/x-www-form-urlencoded` body; another type
is a 415, a body that is not UTF-8 a 400) return one, and `req.formValue(name)`
and `req.formValues(name)` are the low-level shortcuts. `req.form<T>()` and
`req.query<T>()` read a `T: Decodable` through the codec layer (D58) with a
`FormDecoder` — a `codec.Decoder` whose format name is `form`, so
`@key(form: "...")` selects on it and `keys:` applies a `KeyStyle`: the
fields are one flat object; text is parsed into what the struct asks for
(integers, numbers, booleans — `true`/`false`, `on`/`off`, `yes`/`no`,
`1`/`0`, which is what a checkbox sends —, strings, enums by name); a `List`
field takes every value of its name, and a single value reads as a list of
one; an empty value is `null` for an optional field and an empty list for a
list; a bad value or a missing required field is recorded at the field's name
and every problem is reported together as a 400, `invalid form:` (or
`invalid query:`) and one line per problem. Nested structs and lists of lists
are not forms and are problems. `req.query` (the field) still holds the last
value of each name; `req.query<T>()` (the method) is the typed reader — a
struct may have both.

Not included, left for later decisions: signed and encrypted cookies (session
storage on `std/crypto`), multipart forms (they need streaming bodies, the
body-model task), a multi-valued `headers` type (decided with the body model).

Rejected (user, recommended of 2, 2, 3 and 3): the body model first;
multi-map headers now (breaks every `headers:` literal); raw, everything-off
cookies as in Go, and strict cookies with no encoding; untyped-only forms,
and untyped multi-valued maps.

---

### D95 — `if (val x = e && ...)`: binding a nullable's value in a condition (v0.53)

```veles
if (val age = c.maxAge) out.append("; Max-Age=${age.toSeconds()}")
if (val d = c.domain) out.append("; Domain=$d") else out.append("; no domain")
if (val n = header(name)?.toInt() && n > 40) return "big $n"        // chain: n is an i64 in the rest of it
if (ready && val v = expensive() && val w = v.next()) use(v, w)      // expensive() only runs when ready is true
val label = if (val n = header("n")) "n=$n" else "none"              // an expression, too
```

Found writing `std/http`'s `setCookieLine`, where "do this if the field has a
value" took either a temporary `val` and a null test, or a `when` with a
`null` arm and `?.` in the other. Smart casts (D5) cover a variable or a
field path (`if (c.maxAge != null) ... c.maxAge.toSeconds()`), not a call
result or a `?.` chain, and `val x = e else ...` (D61) leaves when the value is
missing but does not run a short block when it is present.

**The form.** `val name = expr` is an operand of the `&&` chain that forms an
`if` condition. It is true when `expr` — any expression of type `T?` — is not
null, and `name` is then the `T`. Bindings chain left to right and each name
is in scope for the operands after it and for the `then` branch; not for the
`else` (`else if (val ...)` binds again) and not after the `if`. Evaluation is
the `&&` chain's: a failed operand stops it, so a later value is not
computed. The expression binds tighter than `&&`, so `val a = x || y` needs
parentheses, and a value with a lower-precedence operator is parenthesised by
the formatter as the author wrote it. `val` is immutable and the name is a
plain local — hover, go-to-definition, rename and the unused-name warning
treat it like any other.

**Refused.** A value that cannot be null (`if (val n = 5)`: a plain `val`
binds it); a binding anywhere but the `&&` chain of an `if` condition — under
`||` or `!`, in a call, in a `when` arm, in a `loop` head. Both messages carry
`(D95)` and belong to the `nullable` family.

**Lowering.** The checker turns each binding into a `Let` (evaluate into a
variable, test it against null) and a fact that narrows the variable to its
non-null type, the machinery of smart casts; code generation, the formatter's
layout and the effect analysis see nothing new. The parser accepts
`val name = expr` at an operand position and the checker decides where it may
stand, so the error names the rule instead of reporting a syntax error.

**Both levels and self-hosting.** It is a block, not a lambda: `return`,
`break`, `continue`, `throw`, `try` and calls that suspend work inside it, which
a parser or code generator written in Veles needs ("if the next token is a
comma, take it and go on"). Kotlin's `x?.let { }` cannot leave the enclosing
function from inside; Swift's `if let` and Rust's `if let` (with let chains) are
the model.

Rejected (user, recommended of 3, 2 and 2): Kotlin's `x?.let(v => ...)` (a
lambda: no `return`/`break`/`continue` out of it, and a result of `R?`);
nothing new (smart cast plus a `val` for unstable places); a single binding
per condition (the chain avoids the nesting); the same form on `loop`
(`loop (val line = readLine())` — `loop { val line = readLine() ?: break }`
already says it; the user answered "No").

### D96 — `std/http`: static files, and `fs.stat` (v0.54)

```veles
app.get("/assets/*", http.files("./public", maxAge: Duration.days(365), immutable: true))
app.get("/docs/*", http.files("./site", redirect: false))
val st = try fs.stat("report.pdf")           // Stat { size, modified: Timestamp, isDir, isFile() }
```

C2's second task. `http.files` sent a file and a content type; a browser, a
CDN or a video player expects validators, ranges and a cache policy, and a
`.env` in the served directory was served like any other file.

**`fs.stat(path): Stat throws IoError`** (public std API): the size, the write
time as a `time.Timestamp` — microseconds, from the 100 ns NTFS keeps or the
nanoseconds POSIX does — and whether it is a directory; links are followed; a
missing entry is `IoError` (`NotFound`). It is the base of every decision
below, so a `304` or a `412` never reads the file.

**Validators.** `last-modified` and a weak `etag` `W/"<size>-<write µs>"`. The
conditions run in RFC 9110 §13.2.2's order: `If-Match` (strong comparison, so
only `*` passes a weak tag) and `If-Unmodified-Since` → 412;
`If-None-Match` (weak comparison, a list, `*`) and, only when it is absent,
`If-Modified-Since` → 304, which carries `etag`, `last-modified` and
`cache-control` and no body. A date that does not parse is ignored.
`etag: false` / `lastModified: false` drop a header with its conditions.

**One range.** `accept-ranges: bytes`; `bytes=a-b`, `a-`, `-n` → 206 with
`content-range`; a `b` past the end is cut, a suffix longer than the file is
the file; a start past the end, or `-0`, → 416 with `content-range: bytes
*/size`. Several ranges, another unit, `b < a` and unparsable text are ignored
and the file is sent whole (RFC 9110 §14.2 permits). `If-Range`: a date is
honoured only if it is the file's to the second; an entity tag needs a strong
match, which a weak tag never gives, so it means "everything". The body is
still read whole and sliced until the streaming body model (C2 next task)
turns that into a seek.

**`Cache-Control`.** `no-cache` by default — safe, and cheap through the 304 —
or `max-age=N` from `maxAge:`, plus `immutable` from `immutable: true`. No
`public`/`private`: the handler cannot know whether the route is behind
authorization. `immutable` without `maxAge`, or a negative `maxAge`, panics at
the `files` call (`@caller_location`, as D94's cookies).

**Directories and hidden files** (the user: "the code must be configurable
like web frameworks, so probably no slash pathing should be also an option").
A directory serves the first of `index: List<string> = ["index.html"]` it
holds, else 404, with no listing. `/docs` with no slash is a 308 to `/docs/`
so relative links resolve, keeping the query; `redirect: false` serves the
index at `/docs` instead. The redirect target is rebuilt from the decoded
segments, each percent-encoded, so `//docs` or `/\docs` can never become a
protocol-relative `//host`. A path segment starting with `.` is a 404 — not a
403, which would confirm the file — unless `dotfiles: true`. Only GET and HEAD
are served, anything else is a 405 with `allow`.

`contentTypeOf` grew `csv xml webp avif woff woff2 mp3 wav ogg mp4 webm zip`
(range requests are what media needs).

Rejected (user, recommended of 4, 3, 3 and 3): a content-hash strong ETag
(reads and hashes the file on every request, and a 304 would save bandwidth
but not IO); Last-Modified only; `multipart/byteranges` (no client asks for
it); no ranges; a general `CacheControl` value on any `Response` (can be added
on top of `maxAge`/`immutable` without breaking them); no header (clients
guess a lifetime from `Last-Modified`); only an `index` parameter, or nothing
new (the `.git`/`.env` exposure and the broken relative links stay).

**Both levels and self-hosting.** The high-level spelling is one call with
named options; the low-level piece is `fs.stat`, which any tool that asks "is
this out of date" (a build system, the compiler's own incremental cache) needs.

*Addendum (D97):* the body is no longer read whole — `http.files` streams from
disk with a declared length, so a range is a seek and `HEAD` never opens the
file. `contentTypeOf` is gone: `MediaType.ofExtension` (D97) is its one spelling.

### D97 — `std/http`: the body model, streaming, multipart, `fs.File`, `MediaType` (v0.55)

```veles
val text = try req.text()                                   // whole body, at most Limits.bodyBytes
val body = req.stream(max: 100 * 1024 * 1024)               // as it arrives; `max` has no default
val form = try req.multipart(max: 50 * 1024 * 1024)         // part by part
http.Response.stream(http.MediaType.eventStream, out => try out.writeText("data: tick\n\n"))
with (f = try fs.open(path, fs.FileMode.Write)) { try f.write(chunk) }
```

C2's third task. A request's body was read whole before the handler ran, into a
`List<u8>`, only under `Content-Length` (chunked was a 501); a response was a
`List<u8>` with a computed length. That could not take an upload larger than
memory, multipart, server-sent events, or a download that is not held whole,
and `Expect: 100-continue` was ignored. The user chose the recommended option
of each of four, and asked that the media-type table sit with the named values
("shouldn't things like `".html", ".htm" => "text/html; charset=utf-8"` sit in
the 'values' file in the http?").

**A lazy request body.** `Request.body` is gone; the body is read when asked:
`req.bytes(max:)`, `req.text(max:)` and the form readers collect it — at most
`Limits.bodyBytes` unless told, 413 over it, before the first byte when the
`Content-Length` already says so — and keep what they read, so they can be
called again in any mix (without that, `formValue` then `formValues` would see
an empty body the second time: found by the existing form test). `req.stream(
max:)` returns a `Body` with `read(max)`, `readAll()` and `length()` (null when
chunked); `max` has no default, as `Conn.readLine`'s, since the sender decides
how much comes. Reads that fail, or are refused over the ceiling, mark the
connection unusable. `Request` holds shared reader state (a `Mutex`, like
`Conn`'s buffer), so copies made by `withHeader` see one body. `http.call`
builds an in-memory body, so tests do not change. Migration: `req.body` →
`try req.bytes()`; `formFields`/`formValue(s)`/`form<T>`/`text` now
`suspends throws Fail | IoError`.

**Chunked and framing.** `Transfer-Encoding: chunked` is decoded (extensions
ignored, chunk-size lines and trailers capped, trailers dropped, the ceiling on
the decoded size, a chunk that cannot fit refused before its data). Both
`Transfer-Encoding` and `Content-Length`, or any coding in an HTTP/1.0 request,
is 400 (RFC 9112 §6.1); a coding other than `chunked` is 501; an unknown
`Expect` is 417.

**`Expect: 100-continue`** is answered on the first read, not at the head: a
handler that refuses without reading never invites the upload (and the
connection closes after, the body having never been sent). **What the handler
left unread** is read and dropped after the response, up to 64 KiB, so a
kept-alive connection stays in step; more, or a failed read, closes it.
**`bodyTimeout`** is now the longest silence between two reads, not the time
the whole body may take, so a large upload that keeps arriving is not cut off
(nginx's `client_body_timeout` is the same). A 413 for `Content-Length` over
the limit now reaches the handler's middleware (it carries a request id); it
used to be refused in `readRequest` before any handler existed.

**`Response.stream(type, producer, length:)`.** The producer, a sendable
`fun(BodyWriter) suspends throws IoError`, runs after the head is sent; each
`write` goes out at once as a chunk (`transfer-encoding: chunked`), or as bare
bytes with `content-length` when `length:` is given (a download needs one for
progress and resume; writing more or fewer than declared fails), or as bare
bytes ended by the connection for an HTTP/1.0 client. An empty write sends
nothing (an empty chunk would end the body). A write error means the client
left; a panic in the producer is caught behind the D56 boundary, and either
closes the connection without the final chunk — the head has gone, so no 500 —
so a truncated body never looks complete. `HEAD` gets the head and no call. The
producer runs after the handler returns, so `http.timeout` does not bound it.
`http.call` collects a stream whole (a never-ending one never returns).

**`http.files` streams**: `sendFile` reads the file 64 KiB at a time with
`readAt`, with `length:` from the stat.

**`fs.open(path, mode): File`** (`FileMode { Read, Write, Append }`),
`Closeable`, with `read(max)` (from where the last read ended), `readAt(offset,
max)`, `write`, `size`. `read` and `readAt` are positional (`pread`,
`ReadFile` with an offset), the place `read` continues from is the Veles
side's, so on Windows and POSIX alike a `readAt` does not disturb it. A file
opened to read refuses writes and the other way round; a closed one refuses.
Reads and writes run synchronously (no worker-thread offload yet).

**`req.multipart(max:, maxParts: 100)`** returns a `Multipart` whose
`next(): Part?` yields parts in order; `Part` has `name`, `filename`,
`contentType`, `headers` and `read(max)`, `bytes(max:)`, `text(max:)`,
`saveTo(path, max:)` (the part `max`es have no default; over one is 413 and
`saveTo` removes its file). A part left unread is skipped by `next`; only the
full boundary line ends a part, so data that resembles the start of one is
data; a preamble, an epilogue and transport padding are tolerated. Malformed
framing, a missing boundary/name/Content-Disposition, a part that is not
`form-data`, oversized part headers (8192) or a body ending inside a part are
400; another content type is 415. `filename` is the client's word and never a
path: `saveTo` takes the caller's. Typed `req.form<T>()` for multipart, with a
file-field type, is not part of this decision (open).

**`MediaType`** joins `Status`, `Method` and `Header` in `values.vs` (D74): a
value type with constants (`text html css javascript csv eventStream json xml
form pdf zip wasm octetStream svg png jpeg gif webp avif icon woff woff2 mp3
wav ogg mp4 webm`), `MediaType.ofExtension(".png" | "png")` (case-insensitive,
`octetStream` when unknown), `essence()` (type/subtype without parameters,
lower-cased) and `Display`. `Response.bytes` and `Response.stream` take `T:
AsMediaType` — a `MediaType`, or a plain string through `implement AsMediaType
for string` — so both spellings work. `http.contentTypeOf` is removed.

Rejected (user, recommended of 3, 3, 3 and 3): buffered bodies with chunked
support only (no upload larger than memory, multipart only buffered); a
per-route stream mode (the server must choose before routing; two paths; a
streaming route that reads `body` silently sees nothing); a `Response.stream`
without files, or none (`Range` would keep slicing memory); buffered multipart
(`List<Part>` inside `bodyBytes`) and deferring it; moving `contentTypeOf` to
`values.vs` unchanged, and leaving it in `files.vs`.

**Not built (open):** request-body decompression; a bound on a producer from
`http.timeout`; typed multipart forms; an SSE helper (`out.event(...)`); a
zero-copy `sendfile`; blocking-thread offload for `fs.File`.

**Both levels and self-hosting.** High level: `req.text()`, `req.form<T>()`,
`http.files`, a `Response.stream` producer that reads like the loop it is; low
level: `req.stream`, `Body.read`, `BodyWriter`, `fs.File` with `readAt` — the
same pieces the std is built from. A compiler in Veles needs `fs.File` and
`readAt` to read sources incrementally and to write output as it is produced
without holding a whole file.

### D98 — `do { ... } catch (e) { ... }`: one handler for several `try`s (v0.56)

```veles
val text = do {
  val a = try req.text()
  val n = try parse(a)
  "got $n"
} catch (e) {
  "cannot read: ${e.message()}"
}

loop (line in lines) {
  do {
    sum += try check(try parse(line))
    if (sum > 100) break            // the loop's own break
  } catch (e) {
    log.warn("skipping: ${e.message()}")
    continue                        // the loop's own continue
  }
}
```

The user, while D97's tests were being written: "we probably need to add some
kind of 'catch', so we could spam the 'try' like we do, but catch AND deal with
the one that failed". Veles had handlers for one call (`r ?? { e => }`, D61's
`val x = r else { e => }`, `when`), and the immediately called closure
`(() => { ... })()` with a `??` after it worked for several — but a closure is a
lambda, so `continue`/`break` in its body is refused ("outside of a loop") and
`return` returns from the closure. The rejection of a `catch` for *panics*
(2026-09-26, D52) is a different thing and stands: a panic is the bug channel;
`do/catch` handles typed errors (`Result`/`throws`), which are ordinary values
(D4).

**The handler's spelling, revised the same day (user).** The first form was
`catch { e => ... }`. The user: "I think the catch syntax is still not the best...
We should make `catch (e) {` like we have `when (v) {` ... (the same with "else"
with errors and other syntaxes like that)". Every other head in Veles binds in
parentheses and then takes an ordinary block — `if (c)`, `loop (x in xs)`,
`when (v)`, `with (d = x)` — and `{ e => }` was the odd one out, a block that
looks like a lambda body and is not. It is now one spelling everywhere an error
is bound:

```veles
val n = parse(s) catch (e) { return -1 }             // one call, was `else { e => }`
val m = parse(s) catch (e) { -e.line.len() }         // was `?? { e => }`
val k = parse(s) catch { 0 }                         // the binding is optional
val t = try parse(s).len() catch (e) { -1 }          // a chain: try marks the call, catch covers the chain
val u = do { val a = try f(); try g(a) } catch (e) { "failed: ${e.message()}" }
```

- **`catch (e) { ... }` is a postfix** on a `Result`-valued expression, on a
  `try` expression, and on a `do` block. It binds tighter than any operator
  (the user chose the tight postfix): `a + parse(s) catch (e) { 0 }` is
  `a + (parse(s) catch ...)`. It may start the next line.
- **Removed, not deprecated** (the user: nobody else uses the old spelling
  yet): `r ?? { e => ... }` and `val x = r else { e => ... }` — the parser
  refuses them with one line naming `catch (e) { ... }`, and so it does
  `catch { e => ... }`. What stays: `r ?? fallback`, `r ?? return`, `x ?: y`,
  `val x = r else return` and the pattern and nullable let-else, none of which
  see an error. So "give me the error" has exactly one spelling, and D61's
  toolbox has one form fewer.
- **`try chain catch (e) { }`** (the user's question: "`(parse(s) catch (e) {
  0 }).len()` is not looking good... what about `parse(s)?.len() catch (e) {}`
  working as well?"). `try` already covers a whole method chain by grammar, so
  `try parse(s).len() catch (e) { -1 }` is `do { try parse(s).len() } catch (e)
  { -1 }`: the `catch` after a `try` belongs to the whole `try` expression, the
  handler yields the chain's type, and the "write the parentheses" warning of
  `try f().m()` stays quiet because a `catch` follows. (A `catch` after a bare
  Result still attaches to that Result.) A chain with two failing calls still
  wants `do { }` or parentheses *(v0.67: no longer — D134, one `try` covers
  every failing call in its chain)*: a postfix marker per call (`?.` for a Result,
  or `!.`) was not chosen — `?.` means nullable chaining (D70, and D61's "one
  operator per kind of maybe"), and `!.` reads as the rejected `!!` (D64); it
  stays open as a separate decision if such chains turn out to be common.
- **Refused:** `catch` after a nullable ("a nullable has none; its fallback is
  `?:`"), after a value that cannot fail, alone, and `catch (e)` written as
  `catch { e => }`; `try f() catch` is *not* refused (unlike `try f() ?? 0`,
  where `try` would propagate the very error `??` handles, a `catch` intercepts
  it).
- **Patterns** (the user: "one way of addressing it would be the patterns but
  I'm not sure"): none now. `when (e) { is A => ... is B => ... }` inside the
  handler is exhaustive-checked over the union today; typed clauses
  (`catch (e: A) { } catch (e: B) { }`, exhaustive, explicit `throw e` to
  propagate) would be a superset of this head and can be added without
  changing anything written now.

**The form.** `do { block } catch (e) { handler }` is an expression. Its value
is the block's last expression, or what the handler yields; the handler is the
handler block of D61's `??` — `(e) { ... }` or `{ ... }` — yielding
the block's type or leaving (`return`, `break`, `continue`, `throw`, `panic`).
A failed `try` or a `throw` anywhere in the block (not inside a lambda in it,
which has its own rules) jumps to the handler. `try` stays: a call that can fail
and is not marked is still D4's "unused Result" error, so every place a failure
can start stays visible (the user chose this over an implicit-propagation
variant). `e` is the union of the error types the block can raise (D45), in the
order they appear, so `e.message()` works and `when (e) { is A => ... }` tells
them apart; `catch (_) { }` and `catch { ... }` ignore it. The `catch` may
start the next line, as an `else` may (D61).

**Block, not lambda.** `return` leaves the function, `break`/`continue` the loop
(or labelled loop) around the `do`, and a call that suspends needs nothing
declared, exactly as in D95's `if (val ...)`. The handler is *outside* the
block: a `try` or `throw` in it belongs to the function or to a `do` around it,
so `catch (e) { throw Wrapped(cause: e) }` makes the function `throws Wrapped`.
Errors the handler catches are not the function's: a function whose only
`try`s are in a `do` need not be `throws` (they are not recorded, so D45's
"declared throws but nothing throws" lint is right about it too).
What the block proved (smart-cast facts) does not hold after it, since the
handler may run from anywhere in it; nor do facts about what it assigned.

**Diagnostics** (family `results-and-errors`, all end `(D98)`): a `do` without
`catch`, and `do { } while (c)` (there is no do-while; the message says
`loop { ...; if (!cond) break }`); a `catch` with no `do`; a block in which
nothing can fail ("the `catch` could never run"); and — refused rather than
mis-compiled — a fail-fast `scope` inside the block that launches a child which
can fail, because a scope passes its error to the *function*, past the handler:
put the scope in its own function and `try` that.

**`do` and `catch` were already reserved words** (the lexer refused them as
identifiers, next to `while` and `finally`); they are now keywords, so no program
that compiled before changes meaning. The user asked whether `do` would be a
contextual keyword — it is not needed: with no do-while and no trailing lambdas,
`do {` at the start of an expression can only mean this. Completion offers both;
the TextMate grammar colours them as control flow.

**Lowering, no code generation change.** The checker turns the block into a
one-shot loop that the block's failing `try`s leave, the technique the eager
collection adapters already use for a throwing function argument (lower_try.go):
`var err: E? = null; var value: T? = null; loop { ...; value = Some(last); break }`
then `if (err == null) value! else handler`. A `try` becomes `let r = x; if (r is
Err) { err = Some(convert(e)); break } else payload`; the union `E` is only known
when the block is done, so the nodes built on the way are patched at the end. The
loop is only in the lowering — the checker's loop stack does not contain it — so
`break`/`continue` written in the block name the program's own loops. A `with`
left by a failing `try` closes (tested). Hover, go-to-definition and rename on
`e` work as for any binding; `e` is typed as the union.

Rejected (user, recommended of 4 and 3, then revisited): the spelling
`catch { block } else { e => }` — chosen first, then withdrawn: "other languages
have 'success story' in a try, and catch block is where the error happens" —
and `attempt { } else { e => }` (a new word), `try { } catch (e) { }` (`try` is
already the propagate prefix: `try` inside `try`), `on { e => }`; implicit
propagation with no `try` in the block (the failure points stop being visible;
a helper that becomes `throws` changes behaviour silently); a function-level
`fun f() { } catch (e) { }` (coarse: no help inside a loop iteration); leaving
the closure idiom. `throw` inside the block goes to the handler (Swift's rule),
not out of the function — otherwise `try` and `throw` two lines apart would
disagree.

**Open:** patterns in the handler (`catch { is Bad => ... }`); a `Fallible`
trait (D61's open item); a fail-fast `scope` inside a block.

**Both levels and self-hosting.** A loop that skips bad input reads as prose; the
`Result` ABI underneath is unchanged. A compiler in Veles needs exactly this
shape — parse a statement and, when it fails, record the error, resynchronise and
carry on with the next one.

### D99 — `std/http`: connection limit, CORS, `guard`, and `std/compress` (v0.58)

```veles
http.serve(listener, app.handler(), limits: http.Limits(connections: 2000))
app.wrap(http.cors(origins: ["https://app.example.com", "https://*.example.org"], credentials: true))
app.wrap(http.bearer(token => crypto.equalBytes(token.bytes(), secret.bytes())))
app.wrap(http.guard(req => if (req.path.startsWith("/admin") && !isAdmin(req)) http.Response.text("no", status: Status.forbidden) else null))
```

C2's last task. The user chose the recommended option of each of four.

- **Connections.** `Limits.connections: i64 = 10_000`, `0` = unlimited.
  Backpressure at `accept`: a permit is taken before `accept()` and given back
  when the connection task ends, so a full server stops accepting and the kernel
  backlog absorbs the burst. No refused connection costs a task or a write.
  A graceful stop wakes an accept that waits for a permit. Rejected: accept then
  503 (each refusal still costs a task and a write under a flood); unbounded.
- **CORS.** `http.cors(origins:, methods:, headers:, expose:, credentials:,
  maxAge:)`. The default allows no origin. `origins` is exact strings, `["*"]`,
  or `https://*.example.com` (matched by the library on a dot boundary, so
  `evilexample.com` never matches). Preflight (`OPTIONS` with
  `Access-Control-Request-Method`) is answered by the middleware before routing;
  `Vary: Origin` is added whenever the answer depends on the origin; a request
  with no `Origin` passes through untouched; `["*"]` with `credentials: true`
  panics at the caller's line (the cookie rule, D94). Rejected: a predicate
  `allow: fun(origin): bool` (invites `endsWith("example.com")`); it can be added
  later without breaking this.
- **Auth.** `http.guard(check: sendable fun(Request): Response?)` — `null` lets the
  request on, a `Response` answers it instead, and a `throw Fail` is the answer
  too; `http.basicAuth(realm, verify: (user, pass) => bool)` and
  `http.bearer(verify: token => bool)` are built on it and send
  `WWW-Authenticate`. No new std function for the secret comparison: the brief
  said std had no constant-time compare, which was wrong — `crypto.equalBytes`
  is one, and the docs example uses it. A guard wraps the whole router;
  per-route guards wait for route groups.
- **Compression.** `std/compress` is written in Veles (inflate and deflate, gzip
  framing), then `http.compress()` (gzip only) follows. Rejected: vendoring
  miniz into the runtime (kept as the fallback if the measured speed is poor);
  linking system zlib (a build dependency on every platform).

### D100 — `with` as a statement, and `with t = async f()` (v0.59)

```veles
test "at the connection limit a new connection waits until one closes" {
  with listener = try net.listen()
  with server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
  val port = listener.port()
  with first = try net.connect("127.0.0.1", port)
  try first.writeText(limitedGet(false))
  with second = try net.connect("127.0.0.1", port)
  try second.writeText(limitedGet(true))
  expect(withTimeout(Duration.millis(300), () => try limitedRead(second)) is Err)
  try first.shutdownWrite()
  expect((try limitedRead(second)).startsWith("HTTP/1.1 200"))
}   // closes second, first; cancels and joins server; closes listener
```

User, 2026-09-30: "are we able to make Veles safe, performant, intuitive, but
less indented by default? … I do not want to have to create 'towers of
terrors' when programming safely and 'properly'." The same test was five
levels deep: a `with` block per resource opened after other statements, a
`scope` to be allowed to write `async`, and a `server.cancel()` whose omission
hung the test for ever. The user chose the recommended option of N1, N2 and N4
(`archive/veles-spec-prep.md` §4); on migration: "just migrate all of our codebase, not
needed support for later".

**1. The statement form.** In a braced block, the statement `with x = e`
followed by statements `S…` means exactly `with (x = e) { S… }` (D43): the rest
of the enclosing block is its body. Every D43 rule holds unchanged — `e` must
be `Closeable` (a `T?` is refused; unwrap first); `close()` runs on every exit
(the end of the block, `return`, `break`/`continue`, a failed `try`, `throw`,
a panic, cancellation); several close in reverse order; the body's error wins
over a close error, which is attached (Kotlin's suppression rule); the close
is shielded from cancellation (D47). One binding per statement (the block form
keeps `with (a = …, b = …)`). If the block ends in an expression that is its
value, the value is computed first and the resources close before it is used
(D43 v0.29).

- *Where.* Any braced statement block: function, test, `test fun`, lambda,
  loop body (it closes at the end of every iteration), `if`/`else`/`when`-arm
  block, `do` block, `scope`/`gather` body. Refused, each with an error naming
  the block form: at module level; as an expression-bodied function
  (`fun f() = …`); as a braceless `if`/`loop`/arm body (nothing would follow
  it); as the operand of an expression.
- *Order.* Statement-form `with`s, block-form `with`s and the with-tasks of
  part 2 in one block form one stack: the last registered is closed first.
- *A `with` that is the last statement* has an empty body: the value is closed
  at once. That is a warning ("closed as soon as it is opened") with no fix.
- *Nothing changes for tooling that reads the tree*: the formatter prints the
  statement as written and never converts between forms; the parser keeps it as
  its own node (`ast.WithStmt`, or a flag on the existing `With` node with the
  body span running to the enclosing `}`), so hover, rename and go-to work on
  `x`. Hover on `with` says where it closes ("closed at the end of this block,
  line 42"); an LSP inlay hint after the enclosing `}` lists what closes there
  in order (`closes second, first; cancels server; closes listener`).
- *Code generation*: none new. The checker lowers it to the block form, so the
  IR is identical to the nested program.

**2. A task bound by `with`.** `with t = async f(args)` (and the block form
`with (t = async f(args)) { }`) starts `f` as a *background child* of the rest
of the block:

- It is fail-fast like a `scope` child (D34/D35 v0.28): if it throws or panics,
  the rest of the block is cancelled at its next suspension point and the error
  propagates from the block — its error types join the enclosing function's
  inferred `throws` exactly as a scope child's do.
- At the end of the block (every exit, in the stack order of part 1) it is
  **cancelled, then joined** — closing a task stops it — unless it has already
  finished. `await t` waits for it and gives its value as for any task handle,
  after which the close has nothing to do. The `Cancelled` its own close causes
  is not an error; a failure raised while it unwinds follows D43's suppression
  rule; the join is shielded cleanup (D47).
- `async` is therefore allowed in exactly two places: lexically inside
  `scope`/`gather`, and as the whole value of a `with` binding. *(v0.62,
  D111: and as a field of a task-holding value that is returned or
  `with`-bound.)* The error for
  `async` elsewhere names both. The value must be the `async` call itself — a
  `Task` obtained another way is not a with-resource (`Task` does not implement
  `Closeable`).
- Captures follow the `async` rules unchanged (Sendable, D35). Inside a `scope`
  body a with-task is cancelled at the end of its own block, before the scope
  joins its other children; inside `gather` it is not one of the gathered
  results.
- Lowering: `with t = async f(); S…` is a scope whose body is `S…` and whose
  child `t` is flagged *cancel at end*; the runtime already cancels children
  when a scope body leaves early, so the new path is the same request made on
  the normal exit before the join.

`scope { }` stays for tasks that are to be *waited for* (a worker pool).

**3. A resource must not outlive its block** (new, both forms). It is an error
(a new family `resources` in `source/family.go` and `reference/errors.md`,
which also takes D43's existing "not Closeable" message; "'x' is closed when its
block ends") to `return` the bound
value or yield it as the block's value, to assign it to anything declared
outside the block (a field, an outer `var`, a global), to store it in a
collection, or to return or store a lambda that captures it. Passing it — or a
lambda capturing it — as an argument is allowed (`withTimeout(d, () => try
read(second))` must keep compiling); a callee that keeps it is not caught, which
is stated in the docs. Checked today: none of these is refused yet.

**4. Migration** (user: "just migrate all of our codebase … as is"). In the same
change, every statement-position `with (…) { }` that is the last statement of
its enclosing block becomes the statement form, and every
`scope { … t.cancel() }` whose only reason was a background task becomes
`with t = async …`, in std, examples, bench, templates, docs and the Go tests'
embedded programs. A `with` block followed by more statements stays a block
(flattening it would delay the close), as does an expression-valued one. No
warning or fix is kept for the old shape — the block form remains legal and is
the way to close before the block ends.

**Low level / self-hosting.** `close()` can still be called by hand on a plain
`val`; the block form is the explicit short scope. The compiler opens few
resources, so self-hosting is neutral; the rule removes one reason for helper
functions.

Rejected: `defer f.close()` (Go/Zig/Swift — two statements where one does,
forgetting it compiles, a failing close in a `defer` has no good answer, and it
is a second way to do D43's job; `defer` stays reserved); leaving blocks only; a
bare `scope` statement (keeps the manual cancel and its hang); implicit scopes in
every block, Swift `async let` style (any `}` or `return` may wait invisibly);
`with val f = …` and `val f = with …` spellings; a warning plus fix for the old
shape.

*(2026-10-01, built — plan B10; decision-free details recorded here.)*
`ast.WithStmt` holds one binding and stays flat in its block; the checker
reads the statements after it as the body of D43's block form, so codegen
sees the nested program. A with-task lowers to a `ScopeBlock` flagged
`Cancel`, whose only child is the launch: an `async` written in the body
still needs a `scope`/`gather` of its own and joins *that* scope. Codegen
cancels the flagged scope's children when its body ends, before the join.
On the fail-fast abort path of a suspension point (an enclosing scope's
child failed), the innermost scope's children are now cancelled before its
join — before, an inner `scope`'s children were waited for, and a
with-task there would have been waited for for ever. Part 3 is a local
check (`sema/resources.go`): a value counts as the resource when it is its
name, a `val` alias of it, a tuple/list/map/struct literal around it, or a
lambda naming it; "outside the block" is a global, a parameter, `this`, a
place through a pointer, or a local declared before the resource; the
storing calls are those of mutable collections, `Deque`, `PriorityQueue`
and channels. A task obtained elsewhere is refused as a resource with a
message naming `with t = async f()`. The hover on `with` names the line of
the closing brace; the inlay hint is syntactic. Migration: the
statement-position sites (38 in std and examples, 14 in docs samples, 4 in
std doc comments), and the `scope { … t.cancel() }` shapes
in `std/http` (tests, the acceptor of `acceptAndServe`), `examples/httpd`,
`session`, `fuzz` and docs chapters 12 and 17; the cancels in
`examples/cancel` and in `serve`'s grace race are what those programs are
about, and stay.

### D101 — Which heads write `val` (v0.60)

**Rule:** *a head that holds an expression writes `val` to bind a name in it;
a head that can only bind does not.* `if (val x = e && …)` (D95) and
`when (val n = e)` take any expression, so `val` marks the binding; `with (f =
e)`, `with f = e` (D100), `loop (x in c)` and `catch (e)` can only bind, so
they do not. This states what already holds; nothing changes in meaning.

The one change is the error: `when (m = parse(s))` and `if (m = parse(s))`
today give two parser errors (`expected ')', found '='` and a stray `)`). The
parser now reads `IDENT =` at the start of an `if`/`when` head as a missing
`val`: one error, family `syntax`, "a binding in a condition is written 'val m
= …'", with a fix inserting `val `, and the head parsed as the binding so no
error follows it. When `m` is an existing `var` the message is the same (an
assignment is a statement, never a condition).

User, 2026-09-30, recommended of 3 (notes I2). Rejected: dropping `val` from
`if`/`when` (reads as assignment; C's `if (x = e)` bug when `x` exists);
`val` in every head (`loop (val x in xs)`, `catch (val e)` — noise, and
contradicts D100's spelling).

### D102 — A collection cannot change while a loop walks it (v0.60)

```veles
loop (x in xs) {
  if (x < 3) xs.push(x + 10)   // error: 'xs.push' changes 'xs' while this loop walks it
}
loop ((k, n) in counts) counts.set(k, n + 1)   // fine: replaces the value of the key being visited
```

Before this (probed 2026-09-30) the meaning was unspecified: a value loop over
a `MutableList` read by index against the live length, so appends were visited
(`xs.push(x)` in every step never ended) and removing an earlier element
skipped one; by-reference loops only warned (`sema/lint_staleref.go`).

**Compile time.** In the body of `loop (… in c)` over a `MutableList`,
`MutableMap`, `MutableSet` or `Deque` named by a local, parameter or `this.f`
path, a call on the same path that changes the collection's **length or
order** is an error, family `mutability`, naming the call and the loop, with
two fixes: loop over a copy (`xs.toList()` / `.toMap()` / `.toSet()`), or
"collect the changes and apply them after the loop". Such calls: on a list
`push`, `pop`, `clear`, `insert`, `removeAt`, `addAll`, `sort`, `sortWith`,
`swap`; on a map `remove`, `clear`, `getOrPut`, and `set` unless its key
argument is the loop's own key variable; on a set `add`, `remove`, `clear`; on
a deque every push and remove — and any method of an `extend` block on the
mutable type (it may do any of these). Allowed: replacing an element in place
(`xs.set(i, v)`, `*x = v` in `loop (&x in xs)`, `m.set(k, v)` for the visited
key, `loop ((k, &v) in m)`), `reserve` (it changes neither), reads. The same
rule covers the by-reference loops, so `lint_staleref`'s warning becomes this
error. It applies inside lambdas in the body only when they are called
there (a lambda passed as an argument counts as called).

**Run time.** A change through another path (a function that reaches the same
collection, an alias) is caught when the loop next steps: the collection's
header carries a modification count, incremented by every length- or
order-changing operation (not by element replacement or `reserve`); the loop
reads it at the start and compares it on each step; a difference panics
"'xs' changed while a loop walked it" at the loop's line (D64 location, D81
chain). Both profiles. It also covers the iterators D42 hands out
(`xs.iter()`) for the four types. Cost: one 8-byte field per collection, an
increment per structural change, a load and compare per step — measured with
`veles-bench` (sort, json, sha256, the list benchmarks) before it lands; LLVM
hoists the compare out of loops that make no calls.

Edge cases: a loop over a `List` (immutable) needs nothing; a loop over
`xs.indices()` or a range, or a condition loop (`loop (i < xs.len())`), walks
no collection — they are the way to grow a worklist while walking it
(self-hosting: the compiler's worklists use them or a `Deque`); `break` right
after the change does not make it legal (keep it simple and uniform); nested
loops over the same collection: the inner change is refused for both.

User, 2026-09-30, recommended of 4 (notes I3). Rejected: compile-time only
(indirect changes stay unspecified); a snapshot (a copy per loop, or
copy-on-write in the runtime); documenting the live-index behaviour.

### D103 — A one-task `gather` is its `Result`; `async` on a `sendable fun` value (v0.60)

```veles
when (val r = gather { async work(9) }) {
  is Ok  => println("done: $r")
  is Err => println("failed: ${r.message()}")
}
val t = async handler(req)        // handler: sendable fun(Request): Response
```

**One task.** A `gather` whose body launches exactly one task yields that
task's `Result<T, E>` itself, not a 1-tuple (D36): the 1-tuple exists nowhere
else, so tuples start at two elements everywhere. With two or more launches the
result is the tuple as before. A one-task `gather` is how a program runs code
and observes whether it panicked (D52); the error type is the one the tuple
element had. Migration: every `.0` read of a one-task gather in std, examples,
docs and tests (Q17a).

**Function values.** `async f(args)` where `f` is a local, parameter, field or
any expression of type `sendable fun(…)` launches it as a task: `f` and the
arguments are evaluated in the parent (as for a method's receiver), the task's
effects come from the function type (D40: its `throws` joins the scope's
errors; a function type that suspends may be launched — it is a task), and
sendability is already guaranteed by the type (D35 v0.28). A value of a plain
`fun(…)` type is refused, family `tasks`: "'f' is not a sendable function; declare
its type 'sendable fun(…)'", with the fix on the declaration when it is in the
same file. The direct-call error that stays ("'async' launches a call") names
what to write. Works everywhere `async` does, including `with t = async f()`
(D100). Migration: the prelude's trampolines (`withTimeout`,
`mapConcurrent`/`forEachConcurrent`, any `fun call(f) = f()`) launch the value
directly (Q17b).

User, 2026-09-30, recommended of 3 and 3. Rejected: leaving the 1-tuple; an
`attempt(f)` prelude function; any function value (a non-sendable closure's
captures would cross tasks); keeping the refusal.

### D104 — Bit operations: rotate, swap, reverse; float sign and neighbours (v0.60)

Rust's names, camel-cased, as the existing `countOnes`, `leadingZeros`,
`trailingZeros` (Q20). Compiler built-ins in the catalogue (hover, completion,
docs), on every integer type (`i8 i16 i32 i64 isize u8 u16 u32 u64 usize`):

| Method | Result | Meaning | LLVM |
|---|---|---|---|
| `x.rotateLeft(n: i64)` | same type | bits rotated left by `n` mod width; a negative `n` rotates right (Go) | `fshl` |
| `x.rotateRight(n: i64)` | same type | the reverse | `fshr` |
| `x.swapBytes()` | same type | byte order reversed (identity on 8-bit types) | `bswap` |
| `x.reverseBits()` | same type | bit order reversed | `bitreverse` |

Signed types operate on the bit pattern (no overflow is possible). On `f32`/`f64`:
`x.copySign(y)` (magnitude of `x`, sign of `y`; `llvm.copysign`),
`x.isSignNegative()` (the sign bit: true for `-0.0` and a negative NaN — next
to `isNaN`/`isFinite`), `x.nextUp()` / `x.nextDown()` (IEEE 754-2008: the
nearest representable value above/below; NaN stays NaN, `+∞.nextUp()` is `+∞`,
`0.0.nextUp()` and `-0.0.nextUp()` are the smallest positive subnormal) —
written in the prelude over `toBits`/`fromBits` (D93). Tests: an example with
the edges of every width, debug and release identical.

Out of scope here: reading and writing integers as big/little endian bytes
(decided later the same day, D130).

User, 2026-09-30, recommended of 3. Rejected: a Go-style `bits` module of free
functions (breaks with the existing methods); Java's names.

### D105 — `reserve(n)` on every growable container (v0.60)

D83's `reserve(n)` — room for at least `n` in total, never shrinks, never
changes the contents — on `StringBuilder` (bytes), `MutableMap` and
`MutableSet` (entries: no rehash while inserting up to `n`) and `Deque`
(elements), besides `MutableList`. A negative `n` is a no-op as on
`MutableList`. Allowed on a local that is later moved (D63's list), like
`MutableList.reserve`; not a structural change for D102. Q12.

User, 2026-09-30, recommended of 3. Rejected: `StringBuilder` only; leaving it.

### D106 — Three small ones: `var xs = []`, `Range.isEmpty`, `cond => (x => …)` (v0.60)

- **An empty literal still needs its type, and the tool writes it.** `var xs =
  []` / `val m = [:]` stays an error, but its message names the right kind
  (`MutableList` for a `var` that is changed, `List` otherwise; `i64`/`f64` as
  the literal defaults) and, when a later use in the same function fixes the
  element type (`xs.push(1)`, an argument, an assignment), carries a fix that
  writes the annotation (`var xs: MutableList<i64> = []`), applied by `veles
  check --fix` and the editor (preferred). No fix when no use decides it.
  Rejected: inference from later use (Rust — larger checker change, type not
  visible at the declaration); the message only.
- **`Range.isEmpty()`**: public on every range, what the prelude called
  `holdsNothing` (renamed), completion and hover (Q17c).
- **A lambda returned by a `when` arm is printed in parentheses**: `veles fmt`
  writes `cond => x => x * 2` as `cond => (x => x * 2)`; no warning (closes spec
  §6.1).

User, 2026-09-30, recommended of 3, and both small ones accepted.

*(2026-10-01, D101–D106 built — plan B11; decision-free details.)* **D101**:
the parser reads `IDENT =` opening an `if`/`when` head as the binding, so the
head goes on (`&&` included) without a second error. **D102**: the
compile-time check matches the loop's collection by its source text (a
local, a parameter or a `this.f` path) and refuses, besides the listed
calls, any method of an `extend` block whose target is the mutable type;
`fill` (an `extend` method) is therefore refused too. The run-time count is
a trailing `mods` field of the list and map headers (codegen's header type
gained an `i64`), bumped by `push` (inline and in the runtime), `pop`,
`clear`, the byte appends, a map's new entry, `remove` and `clear`, and —
through the std-only `listTouched` — by `sort`, `sortWith` and `swap`; not
by `reserve`, `set` or a map's compaction. Map and set loops already walked
a snapshot; they are checked all the same, so the rule is one. Iterators
(`iter()`, `iterator()`) of the mutable types already walk a copy, so a
change cannot disturb one and they carry no check. A `Deque` loop walks a
copy too: the compile-time rule covers it, the run-time count does not.
Measured: no change beyond noise on any benchmark (`bench/results.md`,
2026-10-01). **D103**: `async f(args)` on a function value starts a
synthesized function per function type that calls the value; the
trampolines `invoke` (prelude, `http`) are gone. **D104**: `nextUp`/`nextDown`
are prelude methods over `toBits`; the rest are LLVM intrinsics.
**D105**: a map's `reserve` compacts dead entries first, then sizes the
entry arrays to `n` and the index so that no insertion up to `n` rehashes.
**D106**: the fix is offered when a later `push`/`add`/`insert`/`set` takes
a literal or a local declared earlier, or the binding is passed to a
non-generic named function; the kind is mutable when a call changes it in
place or the literal says `mut`.

### D107 — `with p = m.lock()`: a `Mutex` held to the end of a block (v0.61)

```veles
with n = notes.lock()          // n: *Notes; the lock is held to the end of the block
n.add(text)
n.save()

val all = notes.withLock(n => n.all())   // one expression: unchanged
```

`Mutex<T>.lock()` is usable only as the value of a `with` (both forms; else an
error with the fix) and binds a `*T` to the value inside. The **held region** is
the `with`'s body — the rest of the enclosing block for the statement form, the
braces for the block form. Inside it:

- **a suspension is a compile error** (family `tasks`): any suspending call,
  `await`, `race`, `scope`, `gather`, `async` — "the lock on 'notes' is held
  until the end of this block; release it first with the block form `with (n =
  notes.lock()) { … }`". This is D35's rule, which `withLock` gets from its
  lambda type (`fun(*T): R` cannot suspend); the guard gets it from the region;
- the pointer cannot leave the region (D100 part 3);
- locking the same `Mutex` again panics, as inside `withLock`; holding two
  different ones is allowed (lock order is the program's, as today).

The unlock is the close: on every exit, shielded (D43/D47). Compiler-known (the
prelude implements it over the existing `Held.take`); lending functions, which
would have made it ordinary library code, were not adopted (D111). `with notes.lock()` with no name
(D109) is a bare critical section. `withLock` stays for one-expression uses.

User, 2026-10-01, recommended of 3 (spec §7, open since D43). Rejected: the
guard replacing `withLock` (longer one-liners); `withLock` only.

*Built 2026-10-02.* `lock()` returns the prelude's private `Locked<T>` (the
`Held` and the `*T`); the `with` holds it under a hidden name and binds `n` to
the pointer. Suspension points the checker sees (`await`, `race`, `scope`,
`gather`, `async`, `send`, `recv`, `sleep`, a call of a suspending function
value) are refused as checked; a call of a named function that suspends is
refused after suspension inference, so that error appears only once the rest
of the program has none. The escape message for `n` is "points into a Mutex
that is unlocked when its 'with' block ends"; `n.close()` closes the value,
not the lock, and is not refused (D136). The re-lock panic now reads "a Mutex
was locked again while this task holds it". The editor's hint after the block
says `unlocks notes`.

### D108 — Send arms in `race` (v0.61)

```veles
race {
  queue.send(line)            => { }             // there was room: sent
  sleep(Duration.millis(50))  => dropped += 1    // still full: the line was not sent
}
```

A `race` arm may be `ch.send(v)`: it is ready when the channel can take `v`
(buffer room, or a receiver waiting). When it wins, `v` is in the channel;
**when another arm wins, `v` was not sent** — the commit and the win are one
step. The channel and `v` are evaluated once, in arm order, when the race starts
(Go's `select`), not again while it waits. A send arm on a channel that is
closed when it would be chosen panics, as `send` does (2026-09-25 decision on
`trySend`). The arm binds nothing (`ch.send(v) => body`). When several arms are
ready at once the existing `race` rule applies; that rule is unwritten today, so
the implementer documents it in chapter 12 and D38 (and, if it is not
deterministic or not fair, asks before changing it). Runtime: a send waiter on
the channel that claims the race before writing, the same claim receive arms
use. No task per send; `trySend` and `withTimeout(d, () => ch.send(v))` keep
working.

User, 2026-10-01, recommended of 3 (§9 Q1, asked 2026-09-27). Rejected: leaving
it (a task per bounded send); keeping it open.

*Built 2026-10-02.* The which-arm rule is now written in D38 (first ready in
written order, else first to become ready; not random). A send arm's node waits
in the channel's senders; whoever takes its value claims its race first, and a
send arm that meets another race's receive arm on the same channel takes both
claims or neither (the race's task marks its own race busy for those few
instructions; other claimers wait instead of failing). `val x = ch.send(v) =>`
is an error: a send arm binds nothing.

### D109 — `with expr` without a name (v0.61)

`with sem.acquire()` / `with (sem.acquire()) { … }` hold and close a value the
block does not use. In the block form, an item without `name =` is an
expression, and items mix: `with (f = try open(p), sem.acquire()) { }`. A bare
existing name is allowed (`with conn` takes over closing `conn`; Python's `with
f:`). `with _ = expr` stays legal. Every D43/D100 rule holds; D100's "closed as
soon as it is opened" warning applies when the statement is last.

User, 2026-10-01, recommended of 2. Rejected: requiring `with _ = expr`.

*Built 2026-10-02.* `with _ = expr` did not parse before; it does now, and
means `with expr`. In the statement form, `with` followed by anything but `(`
is the statement; `with (` is the block form. `with conn` holds a copy of
`conn` and closes it; `conn` is a resource from there on (D100 part 3, D136).

### D110 — Concurrency helpers: `retry`, `Semaphore`, channel drains, `ticker` (v0.61)

Pre-approved so they are added when a program needs them, without another
question (notes P11). Prelude unless noted; each `@caller_location` for its
misuse panics (D88).

- **`retry<R, E>(times: i64, f: fun(): R suspends throws E, delay: Duration =
  Duration.zero): R throws E`** — calls `f` until it returns; after a thrown
  error, waits `delay` and tries again, up to `times` calls in all; then throws
  the last error. A panic is not retried. Cancellation stops it at the wait.
  `times < 1` panics.
- **`Semaphore(permits: n)`** (Sendable): `acquire(): Permit` suspends until a
  permit is free (cancellable); `tryAcquire(): Permit?` does not wait;
  `available(): i64`. `Permit` is `Closeable` — closing returns the permit, a
  second close does nothing. Used as `with sem.acquire()` (D109). `n < 1`
  panics.
- **`ch.forEach(f: fun(T) suspends throws E) throws E`** and **`ch.toList():
  List<T>`** on `Channel<T>`: receive until the channel is closed and drained.
- **`time.ticker(every: Duration): time.Ticker`** — in `std/time`, not the
  prelude, because its ticks are `Timestamp`s (consistency pass 2026-10-01).
  `Ticker` is `Closeable` and has
  `ticks: Channel<Timestamp>` (capacity 1): a runtime timer, no task, does a
  `trySend` each period, so a slow reader misses ticks rather than queueing
  them (Go's ticker); the first tick after one period; `close()` stops the timer
  and closes `ticks`. A race arm is `val at = clock.ticks.recv() => …`.
  `every <= 0` panics.

Not approved yet, asked when a program needs them: `filterConcurrent`,
`firstConcurrent`, channel-to-channel stages, `awaitAll`.

User, 2026-10-01: all four ticked.

*Built 2026-10-02.* `retry` is not `@caller_location`: it suspends, and D88
forbids that, so its `times < 1` panic reports the std line (checklist §11).
`Semaphore` is a channel of permits; `Permit` closes once through an
`Atomic<bool>`. `ch.forEach`/`ch.toList` are a prelude `extend<T> Channel<T>`
(channel methods the compiler does not know now fall through to extend
blocks). `Ticker` is fed by a runtime ticker list beside the task timers; std
alone may pass a `Channel` across the C ABI for it. Closing a ticker closes
`ticks`, and a tick already in it can still be received, as from any closed
channel. Building `retry` needed generic `do { } catch` to work in an
instance where `E` is `Never`: the handler is kept (nothing reaches it) and
a `throw` of a `Never` value there is unreachable, instead of "nothing in this
'do' block can fail" and "'throw' fails the function" errors.

### D111 — A value that holds a task is received with `with` (v0.62)

```veles
struct TestServer {
  listener: net.Listener
  server: Task<()>
  fun port(): i64 = this.listener.port()
  implement Closeable { fun close() { this.listener.close() } }
}

fun serving(limits: Limits): TestServer throws IoError {
  val listener = try net.listen()
  TestServer(listener, server: async serve(listener, limitedOk(), limits: limits, log: false))
}

test "at the connection limit a new connection waits until one closes" {
  with srv = try serving(Limits(connections: 1))
  with first = try net.connect("127.0.0.1", srv.port())
  ...
}   // srv's task is cancelled and joined, then srv.close() closes the listener
```

Go's `httptest.NewServer` shape — a helper returns a value with a running task
inside — without unstructured tasks. The user, after lending functions were
proposed (§9 Q2): "I'm not convinced for lend functions, maybe returned
reference to something should work like in golang? and that would work with
`with` no?" It answers Q2 (shared test setup) with no test-specific feature: a
fixture is an ordinary function returning an ordinary value.

- **Task-holding types.** A struct or sealed variant with a field of type
  `Task<…>`, or of a task-holding type, is *task-holding* (from its declared
  fields; a generic instantiated with `Task` is not, so a `Task` cannot reach
  one). Hover on the type and on a call returning one: "holds a task: receive
  it with `with`".
- **Received with `with`.** A task-holding value produced by a call or
  constructor must be the value of a `with` (either form, D100) where it is
  produced, or be returned straight to the caller (the rule then applies
  there); anything else — `val`, an argument, a field of a non-task-holding
  value, a collection, discarding it — is an error, family `tasks`, with the
  fix `with x = …`. A `with`-bound one cannot leave its block (D100 part 3).
- **Its tasks belong to that block.** They are fail-fast background children
  of the `with`'s block exactly as in D100 part 2: a task that panics, or ends
  with an `Err` (an async call of a throwing function is `Task<Result<R, E>>`,
  so `E` is in the field type), cancels the rest of the block and the error
  propagates from it (joining the enclosing function's inferred `throws`); at
  the end of the block every held task is cancelled and joined, in reverse
  field order, **before** the value's own `close()` (so the listener closes
  after the server stops). A task-holding value need not be `Closeable`.
- **Where `async` may appear.** Besides `scope`/`gather` bodies and a `with`
  value (D100), `async f()` may be a field argument of a constructor of a
  task-holding type whose value is returned from the function or bound by
  `with`. If the function fails between that `async` and its return (a later
  argument's `try`), the task is cancelled and joined before the error leaves.
  Storing the `Task` in a local first is not allowed (it would need flow
  tracking; revisit if a program needs it).
- **std** may then offer `with srv = try http.testServer(handler)` (asked with
  the std APIs of batch 7 — decided: D130).

User, 2026-10-01, recommended of 4 (§9 Q2, open since 2026-09-27). Rejected:
lending functions (`with fun … yield v` — "not convinced"); suite-level setup
lines (the 2026-09-27 rejection stands); leaving it open.

*Built 2026-10-02 (B13).* How the tasks reach the block, the user's choice
when adopting them after the return proved racy (a finishing task against
the move; a cancelled caller orphaning them): while a `with` computes a
value that holds tasks, the running task carries that `with`'s scope as its
*receiving scope* (a task-local binding, so a suspending callee inherits it),
and an `async` field argument launches straight into it. `return helper()`
passes it through untouched. An `async` field argument's own arguments are
evaluated in place, but the task starts only after every field of the value
(nested task-holding constructors included), so "fails between the `async`
and its return" cannot leave a started task. Also refused: `async f()` where
`f` returns a value holding a task (no block could receive it), and handing a
returned value's task what the function's own `with`s close. A variant of a
sealed type that holds no task is free where it is made (`val s: Slot =
Free()`). Hover on such a type or a function returning one, and an inlay
hint `stops tasks of, then closes`.

### D112 — `Secret<T>`: redacted, not encodable, wiped (v0.62)

```veles
struct Config {
  port: i64 = 8080
  databaseUrl: Secret<string>
  implement Decodable
}
log.info("starting", log.field("db", cfg.databaseUrl))   // error: Secret is not Encodable
println("db: ${cfg.databaseUrl}")                      // prints "db: [redacted]"
val conn = try pg.connect(cfg.databaseUrl.expose())
```

A prelude type (it has to be the easy spelling in every config struct, as
`Duration` is), for `T` = `string` or `List<u8>` only (other types are an
error: "a Secret holds text or bytes"); it owns a private copy of the bytes.

- `Secret.of(v)` makes one; **`s.expose(): T` is the only way to the value**
  (a fresh copy, grep-able); `toString()`, interpolation, `expect` capture
  and the debugger print `[redacted]`.
- **Not `Encodable`**: passing it where an `Encodable` is needed is a compile
  error, and deriving `Codable`/`Encodable` on a struct with a `Secret` field
  is an error naming `@skip` or a hand-written `encode`. **`Decodable`**: a
  config file or the environment fills it.
- `==` is constant-time (as `Digest`, D59); not `Hashable`, not `Comparable`;
  `Sendable`.
- **Wiped twice.** `Closeable`: `close()` zeroes its bytes, after which
  `expose()` panics at the caller (D88). And the collector zeroes them when it
  frees them: the bytes live in an object whose header carries a *wipe* flag,
  and the sweep `memset`s flagged objects before reusing them — no finalizer,
  no ordering. Honest limit, stated in the docs: what `expose()` returned is
  ordinary memory.

User, 2026-10-01: "also zero when collected" (over the recommended
redact-and-zero-on-close only). Rejected: leaving it.

*Built 2026-10-02 (B13).* Implementation choices, the user's (recommended
of 3 each): the wipe flag is a bit of the type descriptor's `kind`
(`DESC_WIPE`, objects do not grow); std's keys move to `Secret`:
`jwt.sign`/`verify` and `crypto.hmac…()` take `Secret<List<u8>>`, while
`Hmac.start` keeps raw bytes as the low level. `Secret.of(v)` compiles
through D137. A Secret is a `Closeable`, so D115 warns on a local one never
closed. `s.len()` gives the length, which is not secret. Not built: a
debugger's view (Veles has no debugger integration yet).

### D113 — Compile-time evaluation: constant expressions, constant tables, `const fun`, `static assert` (v0.62)

```veles
const KB: i64 = 1024
const MB: i64 = KB * 1024
const KEYWORDS: Map<string, Kind> = ["fun": Kind.Fun, "val": Kind.Val, "when": Kind.When]
const fun crc32Table(): List<u32> { … }          // runs in the compiler for a const, at run time otherwise
const CRC: List<u32> = crc32Table()
static assert(HEADER_SIZE == 16, "the wire header is 16 bytes")
```

Today (probed): only literals and operators over literals are constant —
`KB * 1024` is refused. The user chose all three levels: "I think expressions
and tables and comptime functions"; marked `const fun` (recommended of 3).

**1. Constant expressions.** Literals; other `const`s (any module, as visible;
a cycle is an error); arithmetic, bitwise, shifts, comparison and logic on
numbers and `bool` — an overflow, a division by zero or a shift past the width
is a **compile error** in every profile (it is a constant someone wrote);
string `+` and interpolation of constants; `len()`; `toT()`/`wrapT()` (D86);
`if`/`when` expressions; tuples, enum members, `null`, and constructors of
structs without an `init` block whose fields are constants; `at(i)` on a
constant table (an index out of range is a compile error); calls of `const
fun`s.

**2. Constant tables.** A `const` of type `List<T>`, `Map<K, V>`, `Set<T>`
(the read-only types) or `Array<T, N>` (D121) built from constant elements is
laid out read-only in the binary: no start-up cost, one storage however often it is used. The
collector treats pointers into it as outside the heap. Usable wherever a value
of its type is; elements usable in `when` patterns where a constant pattern is.

**3. `const fun`.** A function declared `const fun` may be called in a constant
expression; it is an ordinary function at run time. Its body is checked
against the rules **at its declaration** (so an edit cannot silently break a
caller's constant in another package): locals, `val`/`var`, loops,
`if`/`when`, recursion, constructors, building `MutableList`/`MutableMap`/
`StringBuilder` locally and returning their read-only form, calls of other
`const fun`s and of std functions std marks `const fun` (string, number and
collection operations, as they are needed). Refused, each an error naming the
rule: I/O, suspension, `async`, `unsafe`/FFI, reading a module-level `val`,
`Mutex`/`Atomic`, task-locals. Generic `const fun`s are allowed. At compile
time: a `panic` (or a thrown error) is a compile error at the `const` with the
message and the call chain; evaluation has a step budget (default 10 million
steps, `--const-steps n`) and the recursion limit, and exceeding either is a
compile error naming the constant. Results equal run-time results: integers
are checked as in part 1, floats follow IEEE 754 binary64/binary32 exactly
(`f32` rounded after every operation), string and collection operations are
the std implementations evaluated, not reimplemented. Implementation: an
evaluator over checked HIR (`sema/consteval`), a constant then lowered as a
table or literal. Self-hosting: a Veles-written compiler must carry the same
evaluator — recorded as a cost; it is also what lets the self-hosted lexer
build its UAX #31 and keyword tables as constants.

**4. `static assert(cond, "why")`** at module level or in a body, `cond` a
constant expression, checked at compile time; the reason is required (as
D78's `assert`); failing is a compile error quoting it and the values of the
constants in `cond`.

Rejected: expressions only; expressions and tables without functions;
`comptime fun` (a second word beside `const`); no marker (Zig-style inference
— an edit inside a function breaks constants in other packages).

*Built 2026-10-03 (B14 parts 1, 2 and 4 of this entry — `const fun`, part 3
above, is still open).* Implementation choices (user, recommended of each):
a constant `Map`/`Set` is laid out whole by the compiler — entries,
metadata and hash index, hashed by a Go copy of the runtime's structural
hash (a differential test checks every key is found at run time) — so a
key type with its own `hash`/`equals` is refused until `const fun`;
scalars, structs and tuples are written in as literals at every use, a
table is one `constant` global (rodata: a write faults) named after its
`const`; a `const` is no run-time global at all. Details settled while
building: the evaluator is `sema/consteval.go` (not a separate package,
which would import the HIR it reads); integers are evaluated exactly and
checked against the type, so `+%`/`-%`/`*%` and `wrapT()` wrap and
everything else overflowing is an error; float division by zero gives the
IEEE result (an infinity), as at run time; `CONST.at(i)` with a constant
`i` is read by the compiler — the element, typed `T` not `T?`, and out of
range an error, in any code, not only in constants; `KB.toU8()` where the
constant does not fit is refused as a literal is (D86). `static assert`
in a generic body is checked per instance. Interpolation formats a float
as the runtime does (shortest round-trip text); an enum or struct is not
interpolated in a constant. Hover on a constant shows the computed value
when it differs from the text.

Found with it: module-level values were initialized in source order, so
`val a = f()` with `f` reading a later `val` saw that value's zero bits
(a crash for a list), and a cycle between untyped globals overflowed the
compiler's stack. Now they are initialized in dependency order — reads
through called functions and lambdas count — and a cycle is an error,
family `constants` (as Go does).

### D114 — No global switch for bounds checks; an unchecked access in `unsafe` (v0.62)

Bounds checks stay on in every profile; there is no `--release-unchecked`. The
low-level escape is local: `xs.atUnchecked(i): T` on `List`/`MutableList`,
`xs.setUnchecked(i, v)` on `MutableList`, `s.byteAtUnchecked(i): u8` on
`string` (a lexer's hot loop) — callable only inside `unsafe` (D44, with the
`// SAFETY:` lint). In a debug build they still check and panic "unchecked
index out of bounds" at the caller (Rust's debug assertion in
`get_unchecked`); in a release build an out-of-range index is undefined
behaviour. The compiler keeps removing the checks it can prove (D62 B).
D21's overflow policy is not reopened.

*Built 2026-10-02 (B13).* `i` is the plain offset (`0 <= i < len`), never
counted from the end; the debug panic reads "unchecked index 3 out of
bounds for list of length 3" at the caller.

User, 2026-10-01, recommended (§9 Q11). Rejected: a loud global profile
(every index in every dependency at once); leaving no unchecked access.

### D115 — A `Closeable` never closed is a warning (v0.62)

A local holding a `Closeable` that is never `with`-bound, `close()`d,
returned, stored (a field, a collection, an outer variable), passed as an
argument or captured warns "'f' is never closed" (family `resources`), with
the fix turning `val f =` into `with f =`; a call whose `Closeable` result is
discarded warns too. Passing counts as handing it off (`serve(listener)`), so
the analysis is local and has no false positive on a hand-off; it can miss a
callee that drops it. Measured over std and examples before it lands, each
hit fixed. Task-holding values (D111) need no warning — not receiving them
with `with` is an error.

*Built 2026-10-02 (B13).* The discard that silences the second warning is
Veles's existing `val _ = …` (user: the explicit discard silences it, over
warning anyway); the quick fix writes `with _ = …`. A local counts as made
here only when its initializer is a call or a constructor (under `try`), so
`val l = srv.listener` borrows and never warns; a method whose receiver
escapes (`SelfEscapes`) counts as a hand-off. Measured over std, examples,
bench and std's test files: no hits (B10's migration already put every
resource under `with`).

User, 2026-10-01, recommended. Rejected: an error (refuses hand-offs it cannot
see); nothing.

### D116 — Suspension follows the argument (v0.63)

```veles
fun mapS<T, U, E>(xs: List<T>, f: fun(T): U suspends throws E): List<U> throws E { … }

m.withLock(p => mapS(xs, x => x + *p))     // fine: nothing passed suspends, so the call does not
val pages = urls.map(u => try fetch(u))     // fine: map suspends here, because the lambda does
```

Probed 2026-10-01: `throws E` already follows the argument (`E` is empty for a
pure lambda), but a `suspends` parameter made every call suspend — refused under
a lock, spreading to every caller of `log.withFields` and `TaskLocal.withValue`
— and the built-in adapters refused a suspending lambda outright. That, not
speed, is what blocked moving them into the prelude (plan B8).

**Rule.** A parameter whose type is a function type marked `suspends` (also
`(… suspends)?`) means *may suspend*. A function's suspension then has two
sources: *unconditional* (an `await`, a call that suspends unconditionally,
`scope`/`race`/`async`) and *conditional* (calling such a parameter, or passing
it on to another function's such parameter). A function whose only suspension
is conditional is **conditionally suspending**; a call of it suspends exactly
when an argument bound to one of those parameters suspends — a lambda whose
body suspends, or a function value whose type says `suspends`. Inference is the
existing D2 pass with one more state; hover says "suspends if `f` does".

- **Instances.** Such a function is compiled at most twice — a plain instance
  and a coroutine instance — and each call site picks one (D8's stenciling key
  gains one bit). A plain call is an ordinary call: allowed in a lock region
  (D107), under `withLock`, in `init`, in a `const fun` (D113) when the rest
  allows it.
- **Nothing loses a suspension point it relied on**: a call that suspended
  before stops suspending only when nothing it passes can suspend, so there is
  no cancellation point inside it either.
- **Not for dynamic dispatch (for now).** A trait method keeps the effects it
  declares (D40): a method table holds one instance.
- **Storing the parameter** (in a field, an escaping lambda) is allowed; a later
  call through the stored value suspends by that value's type, as today.
- **`throws E`** is unchanged — it already works this way.

**Consequences, in the same change** (plan B8, unblocked): the eager adapters
(`map`, `filter`, `fold`, `flatMap`, `any`, `all`, `count`, `find`, `sumOf`…)
move from the Go lowering (`sema/lower_list.go`, `check_map.go`) into the
prelude as ordinary Veles generic over `throws E` and conditional suspension,
with `reserve` (D83/D105); `xs.map(x => slow(x))` works (sequentially;
`mapConcurrent` is the concurrent form); `withFields`/`withValue`/`withLock`
callers stop suspending when their lambda does not. Measured before and after
with `veles-bench` (B8 found the prelude form 0.53 s against 0.56 s at
`--release`) and the IR size of `examples/httpd` (B2).

Self-hosting: a compiler's passes are higher-order functions over trees; they
are written once and stay plain where nothing suspends.

*Built 2026-10-02 (B15, with B8).* Sema: the suspension pass computes two
views per function, its coroutine instance's and (when it has
suspend-parameters) its plain instance's; a call of a conditional function
suspends when an argument bound to one of them does — a lambda whose body
suspends under the caller's view, or a value typed `suspends` — and the
lock-region, `init`, global and lambda-type rules ask the call, not the
callee. Decisions made building it (user, 2026-10-02: clone after
inference): codegen emits a plain instance (`.plain`) on demand as a copy of
the function's header over the same body, with its view attached; a lambda
is emitted when a value of it is made, under the view of the instance making
it, plain when the call it is passed to is; a conditional function's
coroutine copy is kept only if something refers to it. Limits, by
construction: only first-order parameters count (one whose own parameters
are not suspending functions); a function that *keeps* such a parameter
(stores, returns, or hands it to a function that is not conditional) is not
conditional, since a plain instance would hand out a plain function where a
coroutine is called later. Found on the way: a call through a suspending
function value evaluated its arguments twice (fixed). B8: `map`, `filter`,
`forEach`, `fold`, `any`, `all`, `find` and Map's `forEach`, `mapValues`,
`filter`, `getOrPut` are prelude Veles (`std/prelude/adapters.vs`); `count`,
`flatMap`, `indexOfFirst`, `mapNotNull`, `partition` gained `suspends throws
E`; the Go lowering and its `Result` machinery are deleted. Measured:
`bench/adapters` 64–73 ms lowered → 43–46 ms prelude (`reserve` up front);
`examples/httpd` IR 4 607 016 → 4 594 632 bytes. Driver
`TestConditionalSuspension`, conformance `D116-conditional-suspension`, LSP
`TestHoverShowsConditionalSuspension`, chapter 12.

User, 2026-10-01, recommended of 3. Rejected: an explicit marker on the
parameter (`suspends?` — more to write on every higher-order function);
leaving it (B8 blocked, the lowered adapters kept).

### D117 — `is Trait`: at run time on trait objects, at compile time in generics (v0.63)

```veles
fun finish(w: Writer) {
  if (w is Flusher) w.flush()          // run time: w's concrete type implements Flusher
}

fun show<T>(x: T): string {
  if (T implements Display) return "$x" // compile time: decided per instantiation
  "<not printable>"
}
```

Q5, open since 2026-09-21. The user chose both forms (over the recommended run-
time one alone). Probed: `x is Display` was refused with a wrong message
("'i64' can never be 'Display'").

**Run time.** `x is Trait` (and `!is`, and `is Trait` arms in `when`) where `x`
is a trait-object value tests whether its concrete type implements `Trait`; in
the true branch `x` is narrowed to `Trait` (a trait object over the same data
with `Trait`'s method table) — the conversion `sema/supers.go` refuses
elsewhere. The program is linked whole, so the compiler emits, for each trait
named in an `is`, a table from type id to method table over every type that
implements it; the type id comes from the box's method table. A lookup per
check, no per-box cost. Limits: the trait must be object-safe and fully applied
(`is Into<string>`, not `is Into`); a trait with supertraits follows the
existing trait-object limit (D58 log: supertraits cannot yet be a trait
object). On a concrete static type the answer is known: a warning "always
true/false" with the fix. On a type parameter it is an error naming the
compile-time form.

**Compile time.** `T implements Trait` is a condition, where `T` is a type
parameter in scope: usable in `if` (with `&&`, `||`, `!`), `static assert`
(D113) and `const` contexts. In the true branch `T` is treated as bounded by
`Trait` (its methods can be called on values of `T`); the branch not taken is
removed per instance. Both branches are type-checked once, generically. With
D8's shared instances, the condition and the method table come from the
instance's dictionary (one entry per `implements` in the body).

Rejected: run time only, compile time only, neither.

*Built 2026-10-02 (B15).* Run time: with D135 (its build note). A trait whose
supertraits make it an object is allowed (supertrait objects were built with
D58 since this was written). The "always true/false" warning also covers a
trait the object's own trait already requires; on an `is` expression its fix
writes the answer. Compile time: a body is checked per instance (D8 as built
stencils every type set; there are no shared instances), so "checked once,
generically" becomes: each instance checks and compiles only the branch it
takes, and the other is not checked for it (locals named there count as used).
A statement-form `if` stays an `if` over the constant, so a branch that
leaves does not make the code after it unreachable for other instances.
`static assert` and `const` contexts come with B14. `implements` is a word
only after a name in an expression; the built-in types print, compare and hash
without the traits, so `i64 implements Display` is false. Conformance
`D117-implements`, `D135-is-on-trait-objects`; `examples/generics`,
`traitobjects`; chapter 8.

### D118 — User-defined derivation: later, through compile-time reflection (v0.63)

Q6. Not built now; std's own needs are covered by D58's compiler-known set
(`Codable`, `Comparable`). The direction is fixed so no one starts a macro
system: once D113's `const fun` evaluator exists, a trait's author writes the
derivation once, as ordinary Veles that walks a type's fields at compile time
(Zig's `@typeInfo` + `inline for`), and an empty `implement` of that trait runs
it. Designed when a library asks; the design goes through `veles-decide`.

User, 2026-10-01, recommended of 3. Rejected: macros (a second language for
authors, slow builds, unreadable errors); designing it now.

### D119 — `Default` (v0.63)

```veles
struct Stats {
  count: i64
  names: List<string>
  implement Default
}
fun fill<T: Default>(n: i64): List<T> = MutableList.repeat(T.default(), n).toList()
```

A prelude trait `Default { static fun default(): Self }`, implemented for every
number (`0`), `bool` (`false`), `string` (`""`), `List`/`Map`/`Set` and their
mutable forms (empty), `T?` (`null`), tuples of `Default` types, and
`Duration` (`zero`). A struct opts in with an empty `implement Default` (D58's
rule): each field takes its declared default, or else its type's `default()`;
a field with neither is an error naming it. A generic struct's empty implement
infers the bounds (D58). A sealed trait, an enum, `Secret` (D112) and a
task-holding type (D111) are not derivable (an error saying to write
`default()` by hand, which is allowed). D58 deferred it ("field defaults cover
it"); generic code is what field defaults do not reach.

User, 2026-10-01, recommended of 3. Rejected: automatic for every struct whose
fields all have defaults (a contract nobody declared); leaving it.

*Built 2026-10-02 (B15):* `std/prelude/default.vs`; tuples up to eight
elements (prelude implements, as for any arity a program writes by hand). An
enum cannot implement any trait (D57), so it cannot write `default()` by hand
either — the D57 error stands. A struct whose `init` takes a parameter with no
default is not derived. The example above said `make(n, …)`, which takes a
function of the index; `repeat(value, n)` is the value form. Conformance
`D119-default`, `examples/generics`, chapter 8, the stdlib reference.

### D120 — C layout: `@packed`, `@align(n)`, `extern union`, `@transparent` (v0.64)

```veles
@packed
extern struct EpollEvent {
  events: u32
  data: EpollData
}
extern union EpollData {
  ptr: *raw ()
  fd: i32
  u64: u64
}
@transparent struct Fd { raw: i32 }      // passed to C exactly as an i32
@align(64) struct Counter { hits: Atomic<i64> }   // a cache line of its own
```

Answers D51's "noted pressure point" (Q7) with compiler-known attributes, as
D51 intends attributes to be.

- **`@packed`** on an `extern struct` or `extern union`: no padding, alignment
  1. A field read copies with an unaligned load; `&s.field` on a packed field
  is an error (the pointer would be misaligned — Rust's rule). Refused on a
  plain struct (its layout is the compiler's).
- **`@align(n)`** on any struct, or on a field of an `extern struct`: `n` a
  power of two not below the natural alignment (else an error naming it);
  the size rounds up to it. On a plain struct it is the low-level tool for
  keeping hot `Atomic`s apart (false sharing).
- **`extern union U { … }`**: fields of `CLayout` types (D69); size the
  largest field, alignment the strictest; built with exactly one field
  (`EpollData(fd: 3)`); **reading or writing a field only inside `unsafe`**
  (which field is live is C's business); `CLayout` and `Sendable`; no `==`, no
  `Display`, not derivable.
- **`@transparent`** on a struct with exactly one field: its layout and C ABI
  are the field's, so it may appear in `extern` signatures and in `extern
  struct`s wherever the field's type may; otherwise an ordinary struct
  (methods, implements). A binding gives its handles their own types at no
  cost.

User, 2026-10-01, recommended of 3. Rejected: a layout clause
`extern(packed, align: 16) struct` (a second syntax for what attributes
express); leaving it.

### D121 — `Array<T, N>`: fixed-size inline arrays, and const generic parameters (v0.64)

```veles
extern struct SockaddrIn {
  family: u16
  port: u16
  addr: u32
  zero: Array<u8, 8>
}
struct Sha256 {
  var state: Array<u32, 8> = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                              0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19]
}
fun sum<const N: i64>(a: Array<i64, N>): i64 = a.fold(0, (s, x) => s + x)
```

A prelude value type: `N` elements of `T` stored inline — in a local, a field,
another array, an `extern struct` (C's `T x[N]`) — with no heap allocation.

- **`N`** is a constant expression (D113) of type `i64`, `0 ≤ N`, the whole
  value at most 1 GiB (else a compile error).
- **Value semantics**, like a value struct (D7): assignment and passing copy.
  Mutable through a `var` binding or field: `a.set(i, v)`, `loop (&x in a)`.
- **Reads** as a list's (D25/D62): `a.at(i): T?`, `T` where the index is
  proven — a constant index is always proven, so it compiles to a plain load;
  `atUnchecked` in `unsafe` (D114). `len()` is the constant `N`. `loop (x in
  a)`, the read-only collection methods (`map`, `fold`, `contains`, …, D46),
  `a.toList()`, `xs.toArray<N>(): Array<T, N>?` (null on a length mismatch).
- **Construction**: a literal `[…]` where the type is expected (the length
  must match: a compile error otherwise); `Array<T, N>.make(value)` fills;
  `Default` when `T` is (D119).
- **Structural** `==`, hashing and `Display` as a list's; `Codable` as a list
  (decoding requires exactly `N` elements); `Sendable` when `T` is; `CLayout`
  when `T` is (D69), and `withRaw` on an `Array<u8, N>` like on a list.
- **Const generic parameters**: `<const N: i64>` on functions, structs and
  `extend` blocks; `N` is inferred from argument types or written
  (`zeros<16>()`); inside, `N` is a constant. A type position takes a const
  parameter or a constant expression without parameters — `Array<u8, N + 1>`
  is refused for now (Rust's `generic_const_exprs` lesson). Instances are per
  `N` (the size is part of D8's shape). The GC descriptor of an inline array of
  pointers repeats its element's.

Self-hosting: inline tables and buffers without a heap object each; SHA and
the lexer's small buffers. Brackets stay literal syntax (D25).

User, 2026-10-01, recommended of 4. Rejected: `[T; N]` (a second meaning for
brackets); arrays only inside `extern struct`; leaving it.

### D122 — No `.d.vs` declaration files (v0.64)

A binding is an ordinary package (D67 allows `extern` blocks anywhere): by
convention a raw package of `extern` declarations and a safe wrapper package
on top of it (Rust's `-sys` crates, Go's cgo packages). A future `veles
bindgen` writes ordinary `.vs`. The generated `builtins.vs` stub stays a
reference document (notes #16 closed). Q7.

User, 2026-10-01, recommended of 3. Rejected: a declaration-only file kind;
deciding later.

### D123 — Calling variadic C functions (v0.64)

```veles
extern "C" {
  fun fcntl(fd: i32, cmd: i32, ...): i32
  fun printf(format: *raw u8, ...): i32
}
```

`...` may end the parameter list of a function in an `extern "C"` block.
A call (inside `unsafe`, as every extern call) passes the extra arguments with
C's default promotions: `bool`, `i8`, `i16`, `u8`, `u16` widen to `i32`; `f32`
to `f64`; `i32`, `u32`, `i64`, `u64`, `isize`, `usize`, `f64`, raw pointers,
`(*raw T)?`, `extern fun` pointers and `@transparent` wrappers of those pass as
they are; anything else (a struct, a string, a list, a Veles value) is an error.
LLVM implements each platform's variadic call convention (`call i32 (i32, i32,
...) @fcntl(…)`), given the target triple. A Veles function cannot be variadic,
and `extern "C" fun` (a callback) cannot either. Supersedes the §4b entry.
New evidence over §4b: POSIX `open` and `fcntl` are variadic, and calling is
the easy half.

User, 2026-10-01, recommended of 2 (Q7). Rejected: keeping them excluded (a C
wrapper file per variadic function).

### D124 — `std/compress` and `http.compress()` (v0.65)

Written in Veles (D99), measured against Go's `compress/flate` with
`veles-bench`; miniz in the runtime stays the fallback if it is far slower.

- `compress.gzip(bytes, level: i64 = 6): List<u8>`, `compress.gunzip(bytes,
  max: i64 = 64 * 1024 * 1024): List<u8> throws CompressError`; raw
  `deflate`/`inflate` the same; streaming `compress.GzipWriter(to: io.Stream,
  level:)` and `compress.GzipReader(from: io.Stream, max:)` (D128).
  `CompressError.kind`: `Corrupt`, `Truncated`, `TooLarge` (the output passed
  `max`). `level` 0–9, else a panic at the caller (D88).
- **The decompression ceiling is a default, 64 MiB**, changeable with `max:`
  (user's choice over the recommended required `max:`).
- `http.compress(minBytes: 1024)` middleware: gzip when `Accept-Encoding`
  allows it, the body is at least `minBytes`, the type is textual (`text/*`,
  JSON, JavaScript, SVG, XML) and the handler set no `Content-Encoding`; adds
  `Vary: Accept-Encoding`; a streamed body (D97) is compressed as it streams.
  `http.files` serves `name.gz` when it exists beside `name` and the client
  accepts gzip. Compressed *request* bodies are not decoded unless asked
  (`http.decompressRequests(max:)`).

User, 2026-10-01: default 64 MiB (over the recommended required `max:`).
Rejected: a required `max:`; no limit.

### D125 — `std/config`: a struct read from the environment (v0.65)

```veles
struct Config {
  port: i64 = 8080
  databaseUrl: Secret<string>          // DATABASE_URL
  @key("LOG_LEVEL")
  level: log.Level = log.Level.Info
  db: DbConfig                         // DB_POOL_SIZE, DB_TIMEOUT, ...
  implement Decodable
}
val cfg = try config.load<Config>(files: [".env"])
```

- `config.load<T: Decodable>(files: List<string> = [], prefix: string = ""):
  T throws config.Error` decodes `T` through a `codec.Decoder` (format name
  `env`, so `@key(env: "…")` works) over the variables.
- **Names**: a field `databaseUrl` is `DATABASE_URL`; a nested struct
  prefixes (`db.poolSize` → `DB_POOL_SIZE`); `@key` replaces the name; `prefix:
  "APP_"` goes in front of every name.
- **Values by type**: numbers, `bool` (`true`/`false` only), `string`,
  `Secret` (D112), `Duration` (`"30s"`, D60), `Timestamp` (RFC 3339), enums
  by member name (exact, D58), `List<T>` comma-separated, `T?` (absent →
  `null`); a field default is used when the variable is absent; other types
  (maps) are an error at the type.
- **Sources, strongest first**: the real environment, then `files` in reverse
  order, then field defaults. A file is dotenv format (`KEY=value`, `#`
  comments, quotes, an `export ` prefix ignored) or JSON by extension (keys by
  field name, nested objects for nested structs). A missing file is skipped (the
  dotenv convention); one that cannot be parsed is a problem.
- **Every problem at once**: `config.Error { problems: List<Problem> }` (the
  `DecodeError` shape), each naming the variable, the expected type and the
  source — and never printing the value of a `Secret` field.
- `config.describe<T>(): List<config.Variable>` lists the variables a config
  reads with their types and defaults (for `--help` and docs).

User, 2026-10-01, recommended of 3. Rejected: environment only; getters.

### D126 — OpenTelemetry: metrics, traces and logs over OTLP/protobuf (v0.65)

```veles
fun main() throws IoError | config.Error {
  val cfg = try config.load<Config>()
  with try otel.start(service: "notes", endpoint: cfg.otlpEndpoint)   // D109: no name needed
  val created = otel.meter("notes").counter("notes.created", unit: "1")
  ...
}

fun load(id: i64): Note throws db.Error {
  with otel.span("load note", attrs: [otel.attr("note.id", id)])
  ...
}
```

The user chose all three signals and the protobuf encoding (over the
recommended metrics + traces with OTLP/JSON). Module `std/otel`.

- **Start**: `otel.start(service:, endpoint:, headers: Map<string, Secret<string>>,
  interval: Duration = 10s, resource: attrs)` returns a task-holding value
  (D111) that batches and exports in the background; received with `with` in
  `main`, its close flushes. Before `start` (and in tests) every instrument is
  a no-op costing one load and branch.
- **Metrics**: `otel.meter(name)` → `counter`, `upDownCounter`, `histogram`
  (explicit buckets, `record(f64)` or `record(Duration)` in seconds),
  `gauge` (`set`); `add(n, attrs: List<Attr>)`; delta-free cumulative
  aggregation per attribute set; runtime metrics (GC pauses and heap from E5,
  tasks, threads, open connections) registered automatically.
- **Traces**: `with span = otel.span(name, attrs:, kind:)` (D109 allows `with
  otel.span(…)` without a name); the current span is a task-local (D72), so
  child tasks continue it; `span.set(attr)`, `span.event(name)`,
  `span.fail(error)` (status error + exception event); a span that ends by a
  panic or a thrown error records it. `http.serve` opens a server span per
  request from an incoming W3C `traceparent` (or a new trace), `http.fetch`
  (D127) a client span and sends `traceparent`, `std/db` (D129) a span per
  query; attribute names follow the OpenTelemetry semantic conventions.
  Sampling: parent-based, ratio from `start(sampleRatio: 1.0)`.
- **Logs**: once started, every `std/log` line (D91) is also an OTLP log record
  with the trace and span ids of its task; text/JSON output is unchanged and
  carries `trace_id` too.
- **Export**: OTLP over HTTP, `application/x-protobuf`, to `/v1/metrics`,
  `/v1/traces`, `/v1/logs`, gzipped (D124), retried with backoff (D110),
  dropping (and counting) data when the collector is down rather than
  growing without bound. The protobuf encoding is a small hand-written encoder
  for the OTLP messages inside `std/otel` (no code generator, no general
  protobuf module). Needs the HTTP client (D127) and TLS (E1) for `https`.
- Not in this decision: a Prometheus pull endpoint (an extra exporter, asked
  when wanted).

User, 2026-10-01: "All signals + protobuf" (over the recommended metrics +
traces with OTLP/JSON and a Prometheus exporter). Rejected: Prometheus-shaped
metrics as the model; metrics first; counters and gauges only.

### D127 — The HTTP client: `fetch` and the Go spellings (v0.65)

```veles
val res = try http.fetch("https://api.example.com/notes", method: Method.post, json: note)
val created = try res.json<Note>()
val page = try http.get("https://example.com").text()
with client = http.Client(timeout: Duration.seconds(5), headers: ["user-agent": "notes/1.0"])
val users = try client.get("https://api.example.com/users").json<List<User>>()
```

The user chose both spellings.

- `http.fetch(url, method: Method = Method.get, headers:, body: List<u8>? =
  null, text: string? = null, json: T? = null, form: …, timeout: Duration =
  30s, redirect: bool = true, retry: i64 = 0): http.ClientResponse throws
  http.FetchError`; `http.get/post/put/patch/delete(url, …)` are the same with
  the method fixed; a `http.Client(timeout:, headers:, maxRedirects: 10,
  maxIdlePerHost: 8, proxy:)` has the same methods (`client.fetch`,
  `client.get`, …) and is `Closeable` (closes idle connections); the
  top-level functions use a shared default client.
- `ClientResponse`: `status`, `headers`, `ok` (2xx), `ensureSuccess()` (throws
  `http.StatusError` for a non-2xx — a non-2xx is not an error by itself,
  as in `fetch` and Go), `text(max:)`, `bytes(max:)`, `json<T>(max:)` (whole
  reads, `max` defaulting to 64 MiB as D124), `stream()` (the D97 body).
- Connection pooling per host (keep-alive; idle connections dropped lazily on
  use, no background task), a total timeout by default and per-phase limits,
  redirects for GET/HEAD only (others returned as is), `HTTP_PROXY` /
  `HTTPS_PROXY` / `NO_PROXY`, retries only when `retry:` asks and only for
  idempotent methods (D110's `retry` with backoff), TLS verification on
  (D128), a client span and `traceparent` (D126).

User, 2026-10-01: "Both spellings" (over the recommended `fetch` only).
Rejected: one spelling only; a separate request type with `send(req)` (the
named parameters already describe a request).

### D128 — `io.Stream`, and the TLS API (v0.65)

`trait io.Stream : Closeable` with `read(max: i64 = 65536): List<u8>`
(empty at the end), `readExact(n)`, `readLine(max)` (the 2026-09-22 bound;
its `net.TooLong` moves to `io.TooLong`, every use migrated in the same
change),
`write(bytes)`, `writeText(s)`, `shutdownWrite()`; its effects are declared
(D40: `suspends`, `throws IoError | TooLong`). `net.Conn`, `tls.Conn` and
`fs.File` (where `shutdownWrite` does nothing) implement it; `http`'s server
and client take a `Stream`, so HTTPS is the same code as HTTP; D124's streaming
compressors wrap one.

TLS (`std/tls`, plan E1; backends SChannel on Windows, OpenSSL elsewhere):
`tls.connect(host, port, options: tls.Options = tls.Options()): tls.Conn`
(SNI from `host`); `tls.listen(host, port, cert:)` → `tls.Listener`, or
`tls.wrap(listener, cert)`; `tls.Certificate.load(certPath, keyPath)` (the key
read into a `Secret`, D112); `tls.CertificateReloader(certPath, keyPath, every:
Duration)` — a task-holding value (D111) that swaps the certificate for new
connections; TLS 1.2 minimum, 1.3 preferred; `Options(roots:, serverName:,
alpn:, dangerouslyAcceptAnyCertificate: false)` — verification against the
system roots is on, and the only opt-out says what it is.

User, 2026-10-01, recommended of 2. Rejected: a separate `tls.Conn` with its
own HTTP path.

### D129 — Template literals, and `std/db` with `sql"…"` (v0.65)

```veles
val users = try pool.query<User>(sql"select id, name from users where name = ${name} and age > ${minAge}")
val order = db.ident(column) ?: throw http.badRequest("unknown column")
val page = try pool.query<User>(sql"select * from users order by ${order} desc limit ${n}")
```

The user asked how a tagged literal would look and whether it is safe; after
the example (the database receives the text with `$1`, `$2` and the values in
separate protocol fields, and a plain string is refused), chose it
(recommended of 3).

**1. Template literals (language).** An identifier — or a qualified name
(`db.sql`) — written directly before a string literal, with no space, is a
*template literal*: `sql"…"`. The name must resolve to a function marked
`@template` whose parameters are `(parts: List<string>, values: List<V>)`; the
compiler calls it with the literal's text pieces (escapes processed; always
written in the source, never input — one more piece than values) and the
interpolated values (`$x` and `${expr}` as in any string), each converted to
`V` — usually a trait object, so each value's type must implement it (an error
at the `${…}` otherwise). Its result type is the literal's type; it is not a
`string` and converts to none. `@template` is allowed in any package. Formatter,
hover (the template function), highlighting (the tag coloured; embedded SQL
highlighting may come later). Later uses: `html"…"` that escapes, `regex"…"`.

**2. `std/db`** (plan E2, libpq first):
- `db.Sql` is made only by `@template fun sql(parts, values: List<db.Param>)`;
  `db.Param` is implemented for the numbers, `bool`, `string`, `List<u8>`,
  `Timestamp`, `Duration`, `Uuid`, `T?` of those, `Secret` (sent, never
  logged) — and `Sql` itself, which is spliced as SQL with its own parameters
  renumbered (checked with `is Sql`, D135). **Every query method takes `Sql`**;
  there is no `string` overload. `db.ident(name): Sql?` quotes a plain
  identifier (letters, digits, `_`, dotted parts) and is `null` for anything
  else; `db.Sql.dangerouslyRaw(text)` exists for migrations and DDL files, named
  like D128's opt-out.
- `with pool = try db.open(url: Secret<string>, size: 10)` — a task-holding
  value (D111: its health checks run in the background); `pool.query<T:
  Decodable>(sql): List<T>`, `queryOne<T>(sql): T?`, `exec(sql): i64` (rows
  affected), `pool.ping()`; rows decode by the derive with column names as keys
  (`@key(db: "user_id")`, D58; `Number` enums, D58 part 2); statement and
  connection timeouts (`timeout:` per call, defaults on the pool).
- **Transactions**: `with tx = try pool.begin()` — `tx.query/exec` as on the
  pool, `try tx.commit()` explicit; leaving the block without a commit (an
  error, a panic, cancellation, or simply forgetting) rolls back.
- A span per query (D126) with the statement text, never the values.

Self-hosting: neutral (the compiler emits IR through `StringBuilder`).

Rejected: constant SQL plus a list of arguments (as safe, placeholders counted
by hand); a plain string plus arguments.

### D130 — Small std additions (v0.65)

Pre-approved together (user, all four ticked):
- **`with srv = try http.testServer(handler)`** (D111): a real listener on a
  free loopback port running `serve`; `srv.url`, `srv.port()`.
- **Endian bytes** (D104's naming, D121's arrays): `x.toBeBytes()` /
  `x.toLeBytes(): Array<u8, N>` and `u32.fromBeBytes(a)` / `fromLeBytes(a)` on
  every integer type; on `List<u8>`, `readU16Be(offset): u16?` … `readI64Le`
  for every width, signedness and order (null when out of range), and
  `MutableList<u8>.pushU32Be(x)`-style writers.
- **`std/fs`**: `fs.writeAtomic(path, bytes)` (temp file in the same
  directory, fsync, rename, fsync of the directory on POSIX), `with lock = try
  file.lock()` / `file.tryLock(): Lock?` (exclusive advisory: `flock` /
  `LockFileEx`), `file.sync()`, `file.seek(offset)`, `fs.lines(path, max:)`
  (a lazy iterator, each line bounded), `fs.copy(from, to)`.
- **Password hashing**: `crypto.hashPassword(Secret<string>): string` (PHC
  string format, argon2id with OWASP's parameters) and
  `crypto.verifyPassword(Secret<string>, hash): bool`, through a binding
  (libargon2) declared in std's `[native]`.

### D131 — The manifest is `package.vs`, one typed constant (v0.66)

```veles
// package.vs
const package = Package(
  name: "notes",
  license: "MIT",
  require: [
    path("../mathlib"),
    github("acme/httputil", "1.4.2"),
    github("veles-db/pg", "2.1.0"),
  ],
  testRequire: [codeberg("lah/fakeclock", "0.3.0")],
  native: if (target.os == Os.Windows) Native(libs: ["libpq"]) else Native(pkgConfig: ["libpq"]),
  format: Format(indent: 2),
)
```

Q14 and notes #3/#17. The user was shown the same manifest as flattened TOML,
a `veles.mod` line format and this, and reasoned: "package.vs might introduce
noise, but having 'Package' struct, might tell us, that in .vss files we could
have similar thing… most of the language features shouldn't be possible (but
`const` should already do it for us)… there could be a reason for 'if's".

- **The file.** `package.vs` at the package root replaces `veles.toml` and
  marks the root (the nearest `package.vs` above a path, M1). It is not part of
  the root module. It holds `const package = Package(…)` and may hold other
  `const`s and `const fun`s it uses; nothing else (no `use`, no `fun`, no
  types) — each an error.
- **Evaluated alone, as a constant (D113).** The resolver parses and checks
  `package.vs` by itself before any module exists: only its own declarations
  and std's `build` module are in scope — the latter unqualified (`Package`,
  `Native`, `Format`, `path`, `github`, `gitlab`, `codeberg`, `sourcehut`,
  `git`, `commit`, `target`, `Os`, `Arch`, …). D113's rules are what keep
  logic out: no I/O, no loops outside a `const fun`, no reading anything but
  constants. M1 is amended accordingly: the resolver reads Veles source, but
  only this one constant.
- **Conditions on the target only**: the constant `target` — `target.os`
  (`Os.Windows`, `Os.Linux`, `Os.MacOS`), `target.arch` (`Arch.X64`,
  `Arch.Arm64`), `target.release` — describes the target, so a cross-compile
  sees the target. (Spelled `build.os` while being asked; renamed in the
  consistency pass, since the module's names are in scope unqualified.) **No environment variables**
  (user, recommended): a checkout builds the same in every shell, which
  reproducibility, `veles.sum` and caching rely on.
- **`Package`**: `name` (required in a package), `description`, `license`,
  `require: List<Dependency>`, `testRequire` (only for `veles test` and
  `*.test.vs`), `native: Native` (D67's table: `libs`, `staticLibs`,
  `libPaths`, `pkgConfig`), `format: Format` (`indent`, `maxBlankLines`). No
  `version`: versions are tags (D132).
- **Dependencies**: `path(dir)`, `github(repo, version)`, `gitlab`,
  `codeberg`, `sourcehut`, `git(url, version)`, each with an optional `as:`
  local name; the local name is otherwise the dependency's own package name
  (`use httputil`). A version is a string checked at compile time (`"1.4.2"`)
  or `commit("3f2a9c1")` (D132).
- **Scripts**: a `.vss` may declare the same top-level `const package =
  Package(require: […])` (name defaulting to the file's); the `build` names
  are in scope in that initializer; checksums go in `<script>.vss.sum` beside
  it. This closes M1's "scripts will import dependencies… (open)".
- **Tools.** `veles add/remove/update` (D132) edit the `require` list
  through the formatter when it is a literal list, and ask for a hand edit
  when it is computed; the LSP gives completion, hover and errors as in any
  file; `veles fmt` reads `format`; `veles new` writes `package.vs`.
- **Migration.** Every `veles.toml` in the tree (examples, templates, test
  fixtures, docs chapters 11 and 13) becomes `package.vs`; a `veles.toml`
  found by the tools is an error naming `package.vs` and printing the
  equivalent constant. `sema/manifest.go` reads the evaluated constant.
- Self-hosting: no TOML parser to write; the manifest is read by the
  compiler's own front end and D113's evaluator.

Rejected: a `veles.mod` line format (the recommendation until the user's point
about scripts and conditions — least noise, but new syntax in scripts and no
conditions); TOML, flattened (a TOML parser in Veles for self-hosting);
environment variables in conditions; no conditions.

### D132 — Where packages come from: decentralized, without Go's look (v0.66)

The user: "decentralized but I would like to see more ideas, because go works
fine but look awful"; then chose all four refinements below. M7 (minimal
version selection) stands; its major-version-in-the-path convention is
replaced.

- **Identity is (repository, major version).** `github:` / `gitlab:` /
  `codeberg:` / `sourcehut:` shorthands in `Package` (`github("acme/httputil",
  …)`) expand to the https git URL; any other host is written in full with
  `git("git.example.com/team/lib", …)`. Paths appear in `package.vs` only —
  code says `use httputil` (M6).
- **The major is in the version, not the path.** `github("veles-db/pg",
  "2.1.0")`, never `…/pg/v2`. Two majors of one repository coexist only under
  two local names (`as: "pg1"`). MVS selects per (repository, major).
- **Versions are tags**: `v1.4.2` or `1.4.2` in the repository, written
  `"1.4.2"`; a pre-release is selected only when written exactly. **No
  pseudo-versions**: an untagged revision is an explicit `commit("3f2a9c1")`
  pin; a library (no `main`) that depends on a commit pin gets a warning
  (its users cannot run MVS over it); a tag and a commit pin of the same
  (repository, major) in one build list is an error naming both.
- **Fetching**: git into a module cache (`~/.veles/pkg/<host>/<repo>@<version>`),
  or through `VELES_PROXY` — a plain HTTP protocol (`/<repo>/@v/list`,
  `/<repo>/@v/<version>.zip`, Go's GOPROXY shape) — when set.
- **`veles.sum` is mandatory** (§4b): one line per module version in the build
  list, `<repo> <version> h1:<sha256 of the file tree>`; verified on every
  fetch, a mismatch is a hard error; committed with the source.
- **No lockfile** (user, recommended): MVS makes the `require` lists the exact
  build list, `veles.sum` makes it byte-reproducible.
- **Commands**: `veles add <spec>[@version]` (the latest tag of the newest
  major when none), `veles update [name | --all]` (the explicit upgrade M7
  requires), `veles remove <name>`, `veles deps [--why <name>]`, `veles vendor`
  (copies the build list into `vendor/`, used when present).
- **Publishing is pushing a tag.** A search index of names → repositories can
  come later and changes no manifest.

Rejected: a central registry (a service to build, run and secure); Go's
`/v2` paths and pseudo-versions; a lockfile.

### D133 — Health endpoints: `http.Health` (v0.67)

```veles
val health = http.Health()
health.check("db", () => try pool.ping())
health.check("cache", () => try cache.ping(), timeout: Duration.millis(500))
app.wrap(health.endpoints())
```

Left out of D126 (they came with the Prometheus option the user did not
take); the user confirmed them 2026-10-01.

- `http.Health()` is a builder like `Router`; `check<E>(name, f: sendable
  fun() suspends throws E, timeout: Duration = Duration.seconds(2))` is
  error-polymorphic per call (the router's erasure rule, D-std/http v0.29).
- `endpoints(live: "/healthz", ready: "/readyz")` is middleware that answers
  those two paths before routing:
  - **`/healthz`** (liveness): `200 ok` whenever the process can answer — it
    runs no checks, so a slow database never gets the process restarted;
  - **`/readyz`** (readiness): runs every check at once, each under its
    timeout; `200` with `{"status": "ok", "checks": {"db": "ok", …}}` when all
    pass, `503` with the failing names otherwise; and **`503` as soon as a
    graceful stop has begun** (D68's `stop:`), so a load balancer drains the
    instance before it closes. Failure details go to the log (D91), not into
    the response (they may describe infrastructure).
- Health requests are skipped by `logging()` and get no spans (D126) by
  default — a probe every few seconds is noise.
- `veles new --template server` (B7) uses it instead of its hand-written
  `/healthz`.

User, 2026-10-01: "yes, add health endpoints".

### D134 — One `try` covers every failing call in its chain (v0.67)

```veles
val users = try http.get(url).json<List<User>>()        // was: try (try http.get(url)).json<List<User>>()
val page = try client.fetch(url, timeout: d).text()      // each failing link unwrapped
```

D98 left this open "if such chains turn out to be common"; the consistency
pass found that D127 (`fetch(…).json()`) and D129 make them the normal shape.
Swift's rule: one `try` marks an expression and covers every throwing call in
it. The user chose it (recommended of 2).

- **Rule.** In `try E` where `E` is a postfix chain (`a.b(…).c(…)…`), each
  link whose value is a `Result` and is followed by a member that is not a
  method of `Result` is unwrapped there (its error propagates), and the
  chain's last value is unwrapped by the `try` as before. The error type is
  the union of every unwrapped link's errors (D45). A link followed by a
  `Result` method (`try f().ok`) keeps today's reading. `?.` links compose as
  D70.
- **The receiver chain only**: an argument is not covered — `try f(g()).h()`
  passes `g()`'s `Result` as a value, as it does today (a `Result` is a value
  you may pass on purpose).
- `try chain catch (e) { }` (D98) handles the union; `do { }` unchanged.
- The "write the parentheses" warning and its fix are removed; `try (try
  f()).g()` stays legal but warns that the inner `try` is redundant, with a
  fix; std, examples and docs migrated in the same change.

Rejected: keeping one `try` per failing call (parentheses or two lines).

*(2026-10-01, built — plan B11.)* A field link (`try f().body`) is a link
too. When the chain's last value is not a `Result` but an inner link was
unwrapped, the `try` is satisfied by those links. A `(try x).m()` followed
by an operator (`== `, `?:`) stays parenthesized: one `try` would take the
whole operator expression.

### D135 — `x is T` on a trait object: a downcast (v0.67)

```veles
fun param(v: db.Param, out: *SqlBuilder) {
  if (v is db.Sql) out.splice(v) else out.bind(v)
}
```

D117 tests whether a trait object's concrete type implements a *trait*; this
tests whether it *is* a concrete type — Go's type assertion, Kotlin's `is`.
Found in the consistency pass: D129 splices a `Sql` given as a `${…}` value by
exactly this test, which was refused with a wrong message ("can never be").
The user chose it (recommended of 2, over a `asSql(): Sql?` method on
`db.Param`).

- `x is T` / `!is T` / `is T` arms in `when`, where `x` is an open trait
  object and `T` a concrete type implementing that trait: a comparison of the
  box's type id with `T`'s — no table; in the true branch `x` is `T`.
  ~~A value struct is read out by value (D7).~~ **Amended 2026-10-02 (user,
  recommended of 3, found building it):** the narrowed `x` is the value
  *inside* the box, not a copy — a field write or a method that changes
  `this` reaches what the object's own methods see next (Kotlin's smart
  cast). Read by value, a write would have changed a copy silently; refusing
  writes was the other option. A `T` that does not implement the trait is
  the "never matches" error, now correct; a type parameter `T` is refused
  (D117's `T implements X` is the compile-time form).
- Sealed traits already narrow by variant (D12); this is for open traits.
- *Built 2026-10-02 (B15), with D117's run-time half:* slot 0 of every method
  table points to the type's info global (a dense id); `is T` compares that
  pointer, `is Trait` loads Trait's table at the id (null: not implemented)
  and narrows to an object over the same data. The tables cover every type
  boxed anywhere in the program and are built after checking, to a fixpoint
  with instantiation. Conformance `D135-is-on-trait-objects`, the
  `traitobjects` example, chapter 8.

### D136 — `close()` by hand on a `with` value is refused (v0.68)

```veles
fun copy(src: string, dst: string) throws IoError {
  with out = try fs.open(dst, fs.FileMode.Write)
  try out.write(try fs.readBytes(src))
  out.close()        // error: 'out' is closed when its 'with' block ends; closing it here would close it twice
}
```

Found building B10: `with` calls `close()` on every way out of its block, so a
hand-written `close()` on the same value closes it twice — the editor's close
hint said so (user, 2026-10-01). The user chose the recommended of 3.

- **Rule.** Calling `close()` on a value bound by `with` (either form), or on
  a `val` alias of one (D100 part 3's alias tracking), is an error, family
  `resources`, with the fix that removes the call. To close earlier, give the
  resource a block of its own: `with (x = e) { … }` closes at that `}`.
- `t.cancel()` on a with-task stays allowed: it asks the task to stop, and
  the block's end then joins it; a second cancel is a no-op, and stopping a
  background task early is a use the form has (D100 part 2).
- Not caught: a function the resource is lent to that closes it (as for D100
  part 3, stated in the docs).

Rejected: `with` noticing a hand-written `close()` and skipping its own (a
hidden flag per binding, and only within one function); leaving it, with
every `Closeable` required to tolerate a second close (Go's `io.Closer`
advice — a footgun the compiler can refuse).

### D137 — A static of a generic type infers its type arguments (v0.69)

```veles
struct Box<T> {
  value: T
  public static fun of(v: T): Box<T> = Box(value: v)
  public static fun none(): Box<T>? = null
}
val b = Box.of("x")                          // Box<string>
val n: Box<i64>? = Box.none()                // from the expected type
val flags = MutableList.repeat(false, 3)     // MutableList<bool>
val rows = MutableList.make(4, i => i * i)   // MutableList<i64>
val e = Box.none()                           // error: cannot infer type parameter 'T' of 'Box' from this call; write the type arguments, e.g. 'Box<T>.none(...)', or annotate the binding
```

Found building D112, whose `Secret.of(v)` did not compile: a static function
called on a generic struct needed its type arguments written
(`Secret<string>.of(v)`). User, 2026-10-02, recommended of 3.

- **Rule.** `Type.f(args)` on a generic struct, or on a built-in collection
  whose static the prelude adds (`MutableList.repeat`, `MutableList.make`),
  written without type arguments, infers them as a constructor does (D28):
  from the expected type when the result names the type (`Box<T>?` against
  `Box<i64>?`, or against `Box<i64>` when the result is not nullable), else
  from the arguments whose parameter types mention them — other arguments
  first, then lambdas and empty literals against what is bound so far.
- What nothing pins is an error asking for the type arguments; an argument
  that fails on its own reports only its own error. Written type arguments
  (`Box<string>.of(…)`) stay valid and win.

Rejected: inferring for `Secret.of` alone (a rule for one type); changing
D112's spelling to `Secret<string>.of(v)`.

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
- **Module interface files** — build cache, regenerated. Packages distribute as source under the decentralized registry model (manifest §7), so there is nothing to ship them in. *(v0.59: not built — the bootstrap checks a package whole; see §5 and plan E8.)*
- **`veles.sum`** — mandatory. Supply-chain integrity is not opt-in.
- ~~**Variadic C functions** — excluded. The varargs calling convention differs per platform and is the nastiest corner of the C ABI; bind a fixed-arity wrapper instead.~~ *(v0.64: superseded by D123 — calling is allowed, defining is not.)*
- ~~**`testing`** — stdlib, with a `test` build mode. Go's version of this is a genuine strength.~~ Superseded by D78 (`test "…" { }`, a test-only vocabulary, no importable `testing` module).
- **Discarded `async` handles** — no different from bound ones. `scope` joins either way, so there is nothing to warn about. *(v0.59: a task bound by `with` (D100) is the exception — it is cancelled, not waited for, when its block ends.)*

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

*(v0.59, state)* Not built: the bootstrap loads and checks a whole package, dependencies included, from source on every build, and infers across modules in one pass. Interface files arrive with per-module caching (plan E8) and the registry (E7).

---

## 6. Open questions requiring a decision

~~**6.1 — Whether to lint the `cond => x => ...` case.** See D33. Accepted as-is for now.~~ Decided in D106: the formatter parenthesizes the lambda, no lint.

**6.2 — Manifest schema.** ~~Drafted in `veles-manifest.md`.~~ *(v0.59: that file does not exist; `veles.toml` as built is described in the docs, chapter 11, and its open syntax is checklist §9 Q14.)* *(v0.66: decided — D131, `package.vs`.)*

*(v0.59)* The open questions now live in `veles-checklist.md` §9; this section keeps 6.1 and 6.2 for their history.

## 7. Not yet designed

*(v0.59, reviewed.)* Most of this list has since been designed and built; the struck items name where.

- ~~Trait declaration syntax~~ — built (D23, D40, D84; bounds after the name, v0.33)
- ~~Closure representation and capture semantics~~ — built (D32, D35 v0.28 sendable functions)
- ~~Whether `Mutex.withLock` becomes a `with` resource (D43)~~ — decided: D107 adds `with n = notes.lock()`, `withLock` stays
- Whether collection literals are wired to fixed types or an opt-in trait (D41) — still open
- ~~`Mutex<T>`, `Atomic<T>`, and the rest of the synchronization surface~~ — built (D66, D73)
- ~~`for` loop syntax and its desugaring onto `Iterator`~~ — built (`loop (x in c)`, D42)
- ~~Task API surface~~ — built (D34–D36, D72, D100)
- ~~C ABI FFI design~~ — built (D67, D69); the rest decided (D120–D123)
- Standard library scope beyond the self-hosting minimum — decided API by API (plan track C)
- ~~Build tooling, formatter, LSP~~ — built


---

## 8. Interop — scoping note

Direct import of Go, Kotlin, or TypeScript libraries means hosting each of those runtimes (the Go scheduler, a JVM, V8) inside the Veles process. This is not a feature; it is three separate projects, each larger than Veles itself.

The tractable version:

- **C ABI FFI** for native targets — reaches C, C++, Rust, Zig, and anything else speaking the C ABI
- **JS import/export** for the WebAssembly target when it arrives
- ~~**A declaration-file system**, in the spirit of `.d.ts`, so foreign libraries can be bound with Veles-side type safety~~ *(v0.64, D122: bindings are ordinary packages of `extern` blocks; no separate file kind.)*

This covers the large majority of the practical value at a small fraction of the cost.
