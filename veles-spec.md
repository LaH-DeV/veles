# Veles — Language Specification

**Working draft v0.21** — language design complete. Every open question in the language itself is closed. Remaining work is not language design: C ABI FFI, and the v0.1 build plan.

Decision IDs are stable. They are never renumbered; superseded decisions are struck through and replaced by a new ID.

---

## 1. Identity

| | |
|---|---|
| Name | Veles |
| Source extension | `.vs` |
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

*Amended by D40:* inference covers direct calls only. Trait methods and function types declare their effects explicitly, because dynamic dispatch makes the reachable impl set open.

*Known costs:* Stack traces and profiler output are worse than green threads would give. Least battle-tested of the available approaches.

Channel syntax is settled in D16.

### D3 — Structured concurrency

Scoped task lifetimes, cancellation trees, no orphan tasks. Orthogonal to D2.

### D4 — Errors: `Result<T, E>` with `throws` as sugar

`Result<T, E>` is canonical. A `throws` function compiles to **exactly the same ABI** — error in a register, no unwinding tables, no stack ripping. `try` is propagate-or-unwrap sugar.

Consequence: `throws` and explicit `Result` interconvert freely; libraries cannot split into two incompatible error camps; the happy path is zero-cost.

Error type is **typed but inferrable** for ordinary functions. *Amended by D40:* trait methods must declare it. *Extended by D45:* inference across callees with different error types produces a union.

### D5 — Nullability: `T?` with flow-sensitive smart casts, nesting

Kotlin-style syntax and narrowing, but `T?` is **genuine sugar over `Option<T>`** and therefore nests: `String??` is a distinct type from `String?`. Swift is the working precedent.

Rationale: Kotlin's collapsing `T?` makes generic lookups ambiguous — `map[key]` returning `V?` cannot distinguish "absent" from "present but null" when `V` is itself nullable. Kotlin's own workaround (`containsKey`) is two lookups and racy. And collapsing is not simpler: Kotlin had to add the `T & Any` intersection type to recover the expressiveness it destroyed.

- `null` is sugar for the `None` case
- Implicit promotion: assigning `String` where `String?` is expected wraps automatically, so the constructor is almost never written by hand
- Leading-dot case syntax (`.some(x)`, `.none`) for the rare explicit case; this generalizes to all sum types
- `String??` arises only from generic instantiation, essentially never from hand-written source

**Representation is dual:** nullable pointer → niche-optimized, zero cost. Nullable value struct → tagged union with a discriminant. Nesting costs a discriminant only at the nested level. Does not disturb D8: nullable and non-nullable pointers are the same GC shape.

**Precedence with pointers:** `?` binds tighter than `*`, so `*User?` is a pointer to a nullable `User`. A nullable pointer is written `(*User)?`.

This follows the conventional rule that postfix binds tighter than prefix, but it hands the terse form to the rarer case — optional references are common, pointers to nullable values are not — and it reads against the representation note above, where the nullable pointer is the free one. Expect `(*T)?` to be frequent.

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

### M2 — A module is a directory

All `.vs` files in a directory share one namespace and one import path. **Files in the same module do not import each other** — mutual recursion between them is free and always legal.

### M3 — Module identity derives from directory path

A module's name is its directory path relative to the package root. **No overrides.** Directory names that are not legal identifiers are handled by a normalization rule, not by a declaration. The resolver stays manifest-only and never parses Veles source to build the module graph.

### M4 — Import cycles between modules are forbidden

The build graph is a DAG. If two modules need mutual recursion, that is evidence they are one module — and since modules are directories, merging them is moving files.

This is not a style rule. Both of Veles' inferred effects (D2 suspension, D4 error type) are inferred per-module; a DAG makes inference a single topological pass, while cycles would require fixpoint iteration across module boundaries.

### M5 — `pub` grants package-wide visibility

Default visibility is module-private. `pub` makes a declaration visible to the rest of the package. Nothing escapes the package except through the manifest's `exports` field.

Consequence: library authors get a deliberately curated public surface, consumers cannot reach into internals, and Veles never needs Go's magic `internal/` directories.

### M6 — Import paths are logical

Resolved by the package manager against the manifest. No filesystem-relative string paths.

---

## 4. Syntax — provisional

Nothing here is locked. Current working sketch:

```vs
use io
use otherModule.{ someFunc }

fun main() throws {
  val sum = add(5, 7)
  printSum(sum)
}

fun add(a: i32, b: i32) = a + b

fun printSum(sum: i32) {
  io.println("The sum is $sum")
  someFunc()
}

// otherModule/funcs.vs
pub fun someFunc() {
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
- Trait declaration and impl syntax

### D11 — Mutability: `val` / `var` on bindings only

Kotlin's model. Immutability is **shallow and non-transitive** — a `val p: *Point` still permits mutation through the pointer. Veles cannot express "immutable view of mutable data" in the type system. `const` is reserved for genuine compile-time constants.

### D12 — Sum types: sealed traits, no `enum` sugar

```vs
sealed trait Shape
struct Circle : Shape { radius: f64 }
struct Rect   : Shape { w: f64, h: f64 }
```

- **Sealing is a layout feature, not just an exhaustiveness feature.** Under D9 a trait object boxes implicitly; a plain open trait would heap-allocate every sum value, including every `Option<T>`. A closed variant set lets the compiler compute a maximum size and lay the value out inline as a tagged union.
- Variants must be declared in the **same module** as the trait. If a `pub` sealed trait could be extended elsewhere in the package (reachable under M5), exhaustiveness checking would break across modules.
- Variant names are **scoped and importable**: `Shape.Circle` by default, bare `Circle` after explicit import. Top-level variants were rejected — `Loading`, `Error`, `Empty`, `Pending` collide across any two sealed traits in one module.
- **All variant fields must be named.** No positional `Circle(f64)`. This was originally forced by the absence of destructuring; D13 has since added it, but named fields are retained on their own merits and destructuring binds by field name.

### D13 — Pattern matching: `when`, as an expression

Type-test with smart casts, **and** destructuring patterns (revised — destructuring was originally excluded).

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

Exhaustiveness is required; `else` opts out. **Lint against `else` when the scrutinee is sealed**, since it silences exactly the error you want when a variant is added later.

### D14 — Variance: deferred, generics are invariant

`None<T>` keeps a zero-sized phantom parameter, resolved by inference at the construction site; `null` sugar hides it entirely. This is how Rust's `None` behaves.

Rejected: invariance combined with a `Nothing` bottom type, which does not compose — invariance means `Option<Nothing>` is not assignable to `Option<String>`, so the bottom type buys nothing without covariance.

**Deferring is safe.** Turning an invariant parameter covariant later only widens what is accepted, so existing code keeps compiling. Go ships no variance at all; Rust exposes none to users. The eventual cost is auditing the stdlib for widenable parameters when `out`/`in` are added.

### D15 — Specialization is semantically invisible

Whether a trait call was specialized or dispatched through a dictionary must never be observable. Therefore, as hard rules:

- No per-instantiation static state
- No Rust-style `specialization` where a more specific impl overrides a generic one

With those in place, specialization is purely an optimization policy. Veles can ship dictionary dispatch in v1 and add auto-specialization later with no language change. The §6.2 question becomes permanently deferrable.

### D16 — Channels are methods, not operators

`await ch.recv()` and `ch.send(x)`. Send and receive stay syntactically symmetric.

**Rule required by D2:** `await` is mandatory only on primitives that *always* suspend — channels, timers, task handles. Ordinary function calls that transitively suspend carry no marker. This marks the genuine yield points without reintroducing viral coloring, but it must be a stated rule rather than an accident of the primitive set.

### D17 — Trait coherence: global, enforced at link time

No orphan rule. Any module may implement any trait for any type, but **at most one impl may exist per (trait, type) pair across the entire program.**

Module-scoped resolution was considered and rejected. Under D9 a trait object's vtable is built at the boxing site, and under D8 the dictionary is chosen at the instantiation site, so scoped impls would make "this type implements this trait" stop being a fact about the type. The concrete failure: a `HashMap<Foo, V>` populated in one module and read in another silently fails to find present keys, with no error anywhere.

Two consequences to design for:

- **Conflicts should be caught at dependency resolution, not link time.** Impls are visible in the materialized module interfaces (§5), so the package manager can detect a collision while resolving and report both module paths, instead of emitting a linker error with no useful context.
- **Adding an impl is a breaking change.** A library shipping a new `impl Display for Foo` in v1.1 can collide with an impl a downstream user already wrote. Rust's orphan rule eliminates this class; Veles has traded it for flexibility. The semver policy must state that new impls require a major version.

### D18 — Strings: validated UTF-8, immutable, byte-indexed

- Storage is UTF-8 and **validity is guaranteed**, unlike Go. Required at FFI boundaries: bytes entering from C must be validated.
- A single immutable `string` type plus a builder for construction. No owned/borrowed split — with a GC and no borrow checker it would buy nothing.
- `len` counts bytes. Byte-level indexing is available.
- Iteration is explicit: a bytes view and a scalar-value view. **Grapheme segmentation lives in an opt-in stdlib module**, so Unicode segmentation tables don't land in every binary.
- Rejected: UTF-16 (Java/C#/JS), where anything outside the BMP occupies two slots and indexing can split a surrogate pair.

*Open:* whether byte access is `s[i]` or an explicit `s.bytes[i]` view. The latter avoids the surprise of a `u8` coming out of a subscript on validated text, and leaves `s[i]` unspent.

### D19 — A slice that splits a code point returns `string?`

Not a panic. Boundary violation is recoverable and flows through the smart-cast machinery of D5.

### D20 — Panics: unwind to the enclosing task scope

Index-out-of-bounds, null dereference, and division by zero are not `Result` — every array access would return one. They raise a panic that unwinds to the enclosing task scope (D3), cancelling that scope's children.

**Implementation interaction with D2.** A suspended task's frames are heap-allocated state machine objects, not native stack frames, so unwinding must traverse both representations, possibly interleaved. LLVM's `invoke`/landing-pad machinery covers only the native half.

Because the GC means there is no drop glue to run, a cheaper option is available: compile panics as a hidden error-return path — the same mechanism as D4 — which works identically in both frame kinds at the cost of a predictable branch on return and some code size. It cannot catch panics originating in FFI or from signals.

### D21 — Integer overflow: checked in debug, wrapping in release, `+%` always wraps

Rust's model, plus an explicit operator. `+` is checked in debug builds and wraps in release. `+%` means "I intend wrapping, in every build" — Rust's `wrapping_add`, promoted to an operator.

*Known wart:* overflow behaves differently between builds, so a bug that panics loudly in tests can wrap silently in production.

*Resolves the D20 interaction:* release builds have no overflow checks, so no landing pads on arithmetic.

### M7 — Version resolution: minimal version selection

Go's MVS. Reproducible without lockfiles and far simpler to implement than SAT solving.

Requires: a hard "no breaking changes within a major version" culture, since MVS trusts that promise completely; the major-version-in-import-path convention for v2 and later; and an explicit upgrade command, because MVS selects the *minimum* satisfying version and will not pick up security patches on its own.

### D22 — Method receivers: implicit `self`, `mut fun` to mutate

Declare **intent, not representation**. The compiler chooses the calling convention — small non-mutating receivers by value, mutating ones by pointer — and the choice is unobservable.

```vs
struct Counter {
  n: i32

  fun get(): i32 = self.n
  mut fun bump() { self.n += 1 }
}
```

Rejected — Go's declared receiver (`(c Circle)` vs `(c *Circle)`): it declares representation, which produces two well-known failures. A value-receiver method that mutates silently modifies a copy with no diagnostic, and method sets diverge so that `Circle` does not satisfy an interface that `*Circle` does. Neither can arise here, because no representation is ever declared.

Rejected — Rust's `&self` / `&mut self`: enforced by a borrow checker Veles does not have, so the distinction would be decoration.

**Consequence: `val` gains teeth, but only over the struct's own storage.** Calling a `mut fun` on a `val`-bound struct is a compile error, because with value semantics that *is* mutation of the binding.

**Scope of the guarantee, restated precisely.** `mut` means "mutates my own fields," not "mutates anything reachable." A non-`mut` method may freely mutate through a reference-typed field:

```vs
struct Stack<T> {
  items: MutableList<T> = []
  fun sneak(x: T) { self.items.push(x) }   // legal: no `mut` required
}
```

`items` is a reference (D25), so the struct's own storage never changes. `val s = Stack<i32>()` therefore does **not** prevent the stack's contents from changing.

The alternative — making `mut` transitive through reference fields — was considered and rejected. It would require tracking reachable mutability across the whole object graph, which is a substantial analysis and edges toward the ownership system Veles deliberately does not have.

So `val` is stronger than Kotlin's for plain value structs and no stronger for anything holding a collection. Documentation must say this plainly rather than claiming immutability.

### D23 — Methods in the struct body; `impl` blocks for traits; no extension functions

Inherent methods are declared inside the struct. Trait implementations go in `impl Trait for Type` blocks, which are required anyway since D6 permits implementing traits for foreign types.

**`struct Circle : Shape` is not a conformance declaration** — it is variant membership in a sealed set, required by D12. The header says the type is one of the trait's variants; the `impl` block supplies the methods. A non-sealed trait has no header clause at all.

Putting trait implementations in the struct body was considered and rejected. It would mean a trait impl could only be written by whoever controls the type's source, which makes `impl Display for SomeForeignType` impossible — removing the capability D6 grants, that D17's global coherence was built around, and that D23 cited as the reason extension functions were unnecessary. It would also force every trait a type ever implements to be named in its header.

Consequence: inherent methods cannot be added to a type you do not own. Extension functions were declined, so the route is declaring a trait and implementing it. This is narrower than it sounds — Veles owns `string` and the collections, so stdlib types stay method-rich; the ceremony only appears when extending a third-party type. Rust lives this way.

### D24 — Full prelude

`Result`, `Option`, primitives, `List`, `Map`, and the common traits are in scope in every file with no import.

**Version the prelude against language editions from day one.** Adding a name to the prelude later can shadow a user's existing declaration; Rust ties its prelude to editions for exactly this reason, and retrofitting that is far harder than building it in.

### D25 — Collections: `List` / `MutableList` split, reference semantics

**Reference types, not values.** A `List<T>` is a header pointing at a heap buffer; as a value type, copying would share the buffer, which is precisely Go's slice-aliasing footgun where appending to a copy sometimes writes through and sometimes does not. Swift avoids this with copy-on-write, which needs cheap uniqueness checks that ARC provides and D1's tracing GC does not.

**Cost, stated plainly:** because collections are references, `val list` does not prevent mutation. D22's immutability guarantee covers plain structs but stops here. The `List` / `MutableList` split is the mitigation.

**REVISED by D35 — `List` is genuinely immutable, not a read-only view.** The original design followed Kotlin, where a `List<T>` reference may point at a `MutableList<T>` and a holder of the mutable handle can change it underneath you. That is incompatible with D35's isolation rules: a view over someone else's mutable buffer can never be `Sendable`, which would make `List` useless for the case it most needs to serve.

Therefore `List<T>` owns its buffer, and `mutableList.toList()` **copies**. This costs O(n) where Kotlin gives O(1), and it is the price of the guarantee. `MutableList<T>` is no longer a subtype of `List<T>` — conversion is explicit and copying.

D14 remains deferred; nothing here requires variance.

**Maps are insertion-ordered.** Python's and JavaScript's behaviour, not Go's. Iteration is faster than open addressing; memory is roughly 1.3x because of the separate index array; deletions leave tombstones requiring periodic compaction. Ordering is a guarantee that can never be withdrawn — Go randomizes iteration specifically to prevent dependence on it.

**Index assignment.** `map[key] = value` is supported on mutable collections via an assignable-index operator.

**Literals are bracket-delimited** (Swift's form): `[1, 2, 3]` for lists, `["a": 1]` for maps, `[:]` for an empty map. `{}` was rejected because it already means block, lambda, and struct literal; a fourth meaning would make `{ port: port }` ambiguous between a map and a struct.

A bare `[1, 2, 3]` with no expected type is a `List`. A growable collection therefore needs an annotation — `var xs: MutableList<i32> = []`, not `var xs = []`.

### D26 — Trait methods are always callable; ambiguity is an error

No import required to call a trait method. Under D17 there is only ever one impl per (trait, type) pair, so scoping is unnecessary for correctness — it would only serve to disambiguate two traits sharing a method name, which is reported as an error instead.

**Consequence for semver:** adding a method to a trait can introduce an ambiguity in code that compiles today. Together with D17's "new impls are breaking," these are the two breaking-change classes that M7's minimal version selection trusts library authors to handle correctly.

### D27 — Traits have both associated types and generic parameters

Rust's design. The distinction, which should be documented early because reversing it is the standard beginner mistake:

- **Generic parameter** when a type may implement the trait several ways — `From<i32>` and `From<string>` on the same type.
- **Associated type** when there is exactly one natural choice per type — `Iterator::Item`.

**`Iterator` must use an associated type.** Under a generic `Iterator<T>`, a single type could implement `Iterator<i32>` and `Iterator<string>` — legal under D17, since those are distinct (trait, type) pairs — and `for (x in thing)` would have no way to infer `x`. The associated form permits at most one impl per type, so inference always succeeds.

### D28 — Construction: call syntax, named arguments, implicit constructor

```vs
val c = Config(port: 8080, host: "localhost")
val s = Stack<i32>()
```

Every struct gets an implicit constructor from its fields. Fields with declared defaults may be omitted.

- **Declaring an explicit constructor suppresses the implicit one.** Otherwise invariants could always be bypassed by calling the generated version.
- **The implicit constructor is callable only where every field is visible.** Otherwise a `pub` struct with private fields would leak construction.

**Named arguments are language-wide, not a constructor feature.** Consequences:

- **Parameter names become public API.** Renaming one breaks callers. This is the third breaking-change class, alongside D17 (new impls) and D26 (new trait methods).
- **Trait impls must inherit the trait's parameter names**, or a caller would see different names depending on whether the call went through the trait.
- **Named arguments apply only at direct call sites of named functions**, never through function values. If function types carried parameter names, names would become part of the type and complicate assignment.

A parameter has **one name**, Kotlin-style, not a Swift external label plus internal name. Renaming it is therefore always a breaking change.

### D29 — Ranges: `1..100` inclusive, `1..<100` exclusive

Kotlin's current form, not the deprecated `until` infix — Kotlin introduced `..<` specifically to replace it. The two forms pair visually, and no keyword is spent.

Noted tension: index iteration (`0..<arr.len()`) is the more common case and gets the longer form.

### D30 — Elvis operator `?:`

Supplies a default for a `T?`: `counts[word] ?: 0`. Pairs with D5's smart casts.

The safe-call `?.` is included: `a?.b?.c` short-circuits to `null` and the chain's type is `T?`.

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

Parentheses are optional for a single untyped parameter and required otherwise. There is **no `it` shorthand** — every lambda names its parameters.

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

- `Sendable` is an auto-derived marker: a type is `Sendable` when all its fields are.
- Anything captured by an `async` call must be `Sendable`, immutable, or an explicitly synchronized wrapper (`Mutex<T>`, `Atomic<T>`).
- A captured `var` cannot be mutated from more than one task; plain mutable references do not cross task boundaries.

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

Named `race` rather than Go's `select` because the three constructs then share a vocabulary — each name says what it does with the set of children — instead of borrowing the third from an unrelated tradition. Recorded against it: `select` is what anyone arriving from Go will look for, and Go's `select` is typically driven in a loop to service channels repeatedly, which "race" slightly undersells.

### D39 — `.` auto-dereferences pointers

`p.field` works whether `p` is a `T` or a `*T`; no explicit dereference operator is needed for field access or method calls. Go's behaviour.

This applies inside `when` as well, so `is Tree.Node(left, right)` destructures a `*Tree<T>` scrutinee without ceremony.

### D40 — Effects are declared wherever dispatch is dynamic

**The problem.** D2 infers suspension by call-graph analysis and D4 infers error types the same way. Both work for direct calls, where the callee's body or materialized interface is visible. Neither works through **trait dispatch**: under D8 a trait call goes through a dictionary, and under D17 any module may add an impl with coherence checked only at link time, so the set of impls a call may reach is not closed at compile time.

Conservatism does not rescue this. Assuming every trait method may suspend means CPS-transforming every caller of every trait method — and with a full prelude (D24) plus `Iterator`, `Ord` and `Display`, that reaches almost everything. `loop (x in items)` calls `Iterator::next`, so every loop in the language would become a suspension point and nearly the whole program would be state-machined. That is a collapse, not a tuning problem.

**The rule.** Trait methods and function types declare their effects; ordinary functions continue to infer them. Declaration follows the return type, matching `throws`:

```vs
trait Fetcher {
  type Error;
  fun fetch(url: string): Bytes suspends throws Self::Error;
}

// function types carry effects too
val handler: fun(Request): Response suspends throws HttpError
```

Error types on trait methods use an **associated type** (D27), since the concrete error varies per impl.

**Defaults.** A trait method neither suspends nor throws unless it says so.

**Sealed traits are exempt.** D12 requires variants in the same module as the trait, so the impl set genuinely is closed and inference works normally. `Option`, `Result` and every user-defined sealed type remain fully inferred.

**What this costs.** This is a partial retreat from D2's no-coloring goal, confined to trait declarations and function types — ordinary function definitions never carry an effect marker, so libraries still don't split into sync and async ecosystems. But a suspending iterator cannot share a trait with a non-suspending one, so Veles will need the `Iterator` / `AsyncIterator` split that Rust has as `Iterator` / `Stream` and Kotlin has as `Iterator` / `Flow`. That is the state of the art, not a Veles-specific failure, but it should be recorded as the price of D17's open impls.

### D41 — Collections are concrete types; abstraction goes through traits

`List<T>`, `MutableList<T>`, `Map<K, V>` and `MutableMap<K, V>` are **concrete types**, not traits.

This is forced by D9. An open trait used as a variable type is a boxed trait object, so declaring `val xs: List<i32>` against a `List` *trait* would allocate and dispatch dynamically on every operation. Kotlin pays exactly this cost because its `List` is an interface.

Abstraction moves to traits taken as generic bounds — `fun <I: Iterable> summarize(xs: I)` — which D8 stencils with no boxing and no dynamic dispatch.

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

`__it` is `var` because `next` is a `mut fun` (D22). The loop variable's type is the projection `Iter::Item`, using the associated-type projection D40 introduced with `Self::Error`.

**Termination depends on D5 nesting.** If `Item` is itself `string?`, `next()` returns `string??` and the outer `null` means end-of-sequence. Under Kotlin's collapsing rule this would be ambiguous with a sequence containing nulls — a concrete case where D5's nesting decision is load-bearing rather than theoretical.

**One loop syntax covers sync and async.** An `AsyncIterable` whose `next` is declared `suspends` (D40) drives the identical `loop (x in c)`; the compiler selects by which trait the receiver implements, and the enclosing function is inferred suspending by D2. Because ordinary functions carry no effect markers, nothing at the call site changes. Rust requires `for await` or `while let`, and Kotlin requires `.collect { }`; Veles requires neither. Implementing both traits on one type is an ambiguity error.

*Open:* loop labels for `break` and `continue` targeting an outer loop.

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

Rejected: Rust-style `From` conversion at `try`, which requires the caller to declare its error type and abandons inference; and boxing into an `Error` trait object, which discards the typing D4 exists to preserve and allocates on every error under D9.

**Cost — this is the fourth breaking-change class.** Adding a failure mode to a function changes its inferred union and breaks downstream exhaustiveness. It joins D17 (new impls), D26 (new trait methods) and D28 (parameter renames) as ways a library breaks consumers without anyone editing a signature. Since M7's minimal version selection trusts authors to honour semver, all four must be documented prominently.

**Mitigation:** declare `throws E` explicitly on public API boundaries to pin the set. This should be recommended practice from day one, not discovered later — deep call chains otherwise accumulate large unions, which is the complaint Zig users raise.

### D46 — Collection operations come in eager and lazy forms

Eager methods on collections (`list.map(...)` returns a `List`), and lazy adapters on iterators (`list.iter().map(...)` materializes only at `toList()`). Kotlin's split between collections and sequences.

**Lazy adapters are `Iterator` methods, so "sequence" is not a third concept.** `AsyncIterator` (D42) inherits every adapter for free — a lazy `map` over an async source needs nothing extra.

Cost: every operation exists twice, so the collection area of the stdlib roughly doubles, and there is a performance cliff users have to learn. D23's absence of extension functions means both sets must be declared up front by whoever owns the types; neither can be added later from outside.

### D47 — Cleanup is implicitly non-cancellable

D43 says `with` releases on cancellation; D20 says cancellation is a panic delivered at the next suspension point. Those combine badly: a `close()` that suspends — flushing a socket, say — would be cancelled immediately and the resource would leak. Kotlin hit this and added `NonCancellable`.

**Cleanup runs shielded from cancellation.** No explicit construct; `close()` bodies are implicitly protected, so a suspending cleanup completes.

*Hazard, to be documented:* a cleanup that hangs cannot be interrupted, and the enclosing scope hangs with it. Cleanup must be bounded. Kotlin's `NonCancellable` carries the identical warning.

### D48 — Alternative orderings use comparator lambdas

D17 permits one `Ord` impl per type, so descending and case-insensitive sorts need another route. That route is comparator and key-extractor lambdas — `sort(by: ...)`, `sortBy(...)` — not a second `Ord` impl.

This **formally drops named-impls-as-values**, which had been recorded as still-useful since D17. Nothing else in the design now needs it.

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

*Consequence — this is the fifth breaking-change class.* Adding a default body to an existing trait method breaks every impl that already implements it, since all of them would suddenly require `override`. It joins D17 (new impls), D26 (new trait methods), D28 (parameter renames) and D45 (widened error unions). Kotlin avoids this by requiring `override` on every interface method implementation; Veles trades that for less ceremony and one more thing library authors must watch when bumping a major version.

### D54 — Values crossing a task boundary must be `Sendable`, including error types

D52 has a child's `Result<T, E | Panic>` cross into `gather`, and D35 requires anything crossing a task boundary to be `Sendable`. **Both `T` and `E` must therefore be `Sendable`**, and so must `Panic`.

This was never stated and it constrains error design directly: an error type carrying a `MutableList` payload (D25 — mutable collections are never `Sendable`) silently becomes unusable in any concurrent code. Because D45 infers error *unions*, a single non-`Sendable` component poisons the whole union for every caller above it.

Consequences:

- Error payloads should be immutable data — strings, numbers, `List`, value structs. Not mutable collections, not open handles.
- **Every stdlib error type must be `Sendable`.** This is a design constraint on the stdlib, not a guideline.
- Applies to `scope`, `gather`, `race` and `Channel<T>` uniformly.

---

## 4b. Settled minor decisions

- **Semicolons** — Go-style automatic insertion.
- **Loop labels** — `outer: loop { ... break outer }`.
- **Byte access** — `s.bytes[i]`, an explicit view. `s[i]` is left unspent rather than producing a `u8` from validated text.
- **`Set`** — follows D25's immutable/mutable split and is insertion-ordered, matching `Map`.
- **`gc.retain` handles** — `Closeable`, acquired through `with` (D43).
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

- Trait declaration syntax; generic bound syntax (`fun <T: Ord>` is provisional)
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
