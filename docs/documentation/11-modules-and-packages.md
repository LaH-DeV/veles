# 11. Modules, packages and tests

## A module is a directory

Every `.vs` file in a directory belongs to one module and shares one
namespace (M2). Files in the same directory never import each other;
split a module across files however you like. There is no `package` or
`module` line at the top of a file — the directory *is* the declaration
(M3).

```text
myapp/
  veles.toml          # package manifest (optional for single-module programs)
  main.vs             # module ""  — the root
  helpers.vs          # also module ""
  geometry/
    point.vs          # module "geometry"
    shapes.vs         # module "geometry"
  storage/
    db.vs             # module "storage"
```

## Scripts

A `.vss` file is a **script**: a program that is one file. It is its own
package — the file is the root module — and it ignores its neighbours:
other `.vs` and `.vss` files in the same directory are not part of it, and
a directory module never reads `.vss` files. Scripts import the standard
library and nothing else (dependencies: later). Use one for a tutorial
step, a throwaway tool or an experiment; when it grows a second file,
move it into a directory and rename it `.vs`.

```text
scratch/
  fizzbuzz.vss        # `veles run scratch/fizzbuzz.vss`
  primes.vss          # a separate program with its own main
```

## Importing

`use geometry` brings the module in under its own name, and its members
are always reached through that name: `geometry.Point`, `geometry.norm()`.
There is no way to import a single name — the prefix is the point (M6):
at the use site it says where a thing comes from, and two modules may
both have a `Point` without anyone renaming anything. When the prefix is
long, rename the module with `as`. One `use` lists any number of
imports, separated by commas:

```veles
// fragment — main.vs, next to a geometry/ directory
use io
use geometry as geo

fun main() {
  val p = geo.Point(x: 3.0, y: 4.0)
  io.println("${geo.distance(p, geo.origin())}")
}
```

A parameter or local named like a module shadows it inside its scope
(`fun mkdir(path: string)` cannot call `path.dir(path)`); rename the
module at the import (`use path as paths`) when that happens.

Import paths are logical, resolved by the compiler against the package
(M6): never a file path, never a URL. Nested directories use dots:
`use net.http`.

The order of imports means nothing, so `veles fmt` decides it: consecutive
`use` lines are merged into one sorted list per origin — standard library
modules on one line, everything else on the next — as in the example above.

Imports must form a DAG. If `a` uses `b` and `b` uses `a`, the compiler
reports the cycle (M4) — merge them or move the shared part into a third
module. The reason is not aesthetic: the compiler infers errors and
suspension one module at a time in dependency order, and a cycle would
make that inference circular.

## Visibility

Everything is private to its module unless marked `public` (M5): functions,
structs, fields, methods, module-level values.

```veles
// fragment — geometry/point.vs
public struct Point {
  public x: f64
  public y: f64
}

public fun distance(a: Point, b: Point): f64 = root((a.x - b.x) * (a.x - b.x) + (a.y - b.y) * (a.y - b.y))
public fun origin(): Point = Point(x: 0.0, y: 0.0)

fun root(x: f64): f64 = ...   // private helper
```

A struct whose fields are module-private cannot be constructed outside
its module with the `Point(x: .., y: ..)` syntax; give it a `public fun`
constructor instead. That is how you keep invariants at the module
boundary; for an invariant that even the rest of the module must not
touch, a member can be `private` to its type
([chapter 5](05-structs-and-methods.md#visibility)).

The three levels, then: `private` — the type's own methods, implement and
extend blocks; `internal` — the module (the directory), which is what
nothing written means, so the word is optional; `public` — the whole
package. These say who can *see* a name. Who can *assign* a field is a
separate question with its own words — bare, `protected var`, `var`
([chapter 5](05-structs-and-methods.md#protected-var-everyone-reads-the-type-writes)) —
so `public protected var count` reads "everyone sees it, only the type
changes it". What leaves the *package* is not a keyword but the manifest's
`exports` list, below.

## Packages and `veles.toml`

A **package** is the directory tree under a `veles.toml`. The manifest
holds the package's identity and dependencies; source files never do
(M1):

```toml
[package]
name = "mathlib"
version = "0.1.0"
exports = ["geometry"]        # modules other packages may import

[dependencies]
utils = "../utils"            # a path dependency

[format]
indent = 2                    # spaces per level, or "tab"; the default is 2
max_blank_lines = 1           # consecutive blank lines kept by `veles fmt`
```

- `exports` is the package's public surface. `public` makes something
  visible to the *rest of the package*; only modules listed in `exports`
  (plus the root module) can be imported from *outside* it (M5).
- A dependency is imported by its manifest name: `use utils`,
  `use utils.text`. In the bootstrap compiler dependencies are
  path-based (`name = "path"` or `name = { path = "..." }`); a registry
  and version selection are planned (M7).
- **A package is the program; a module is not.** `fun main()` lives in the
  root module. `veles run` or `veles build` pointed anywhere inside the
  package finds the manifest above it and runs the package — you cannot
  run `geometry/` on its own, and a package whose root has no `main` is a
  library. `veles check` accepts any module or library. Without a
  `veles.toml`, a directory is taken as its own single-module package.

`examples/packages` in the repository is a two-package project you can
run: `veles run examples/packages/app`.

## The standard library

`use io` is the only import you need for the tutorials. The **prelude**
— `Iterator`, `Iterable`, `Closeable`, `Mutex`, `Atomic`, `Panic`,
`Some`/`None`/`Ok`/`Err` — is in scope without any `use`. The
[standard library reference](reference/stdlib.md) lists everything.

## Tests

A function marked `@test` is a test. `veles test <dir>` compiles the
module with tests included and runs each one as its own task; a test
passes when it returns, and fails when it throws or panics.

```veles
use io

error Mismatch {
  expected: i64
  actual:   i64
}

fun add(a: i64, b: i64) = a + b

fun expectEq(expected: i64, actual: i64) throws Mismatch {
  if (expected != actual) throw Mismatch(expected, actual)
}

@test
fun additionWorks() throws Mismatch {
  try expectEq(4, add(2, 2))
}

@test
fun thisOneFails() throws Mismatch {
  try expectEq(1, add(1, 1))
}

@test
fun panicsAreReported() {
  val xs = [1]
  io.println("${xs.atOrPanic(3)}")
}

fun main() {
  io.println("ordinary run; tests are skipped")
}
```

Output:
```text
ordinary run; tests are skipped
```

`veles test` on that module prints:

```text
test additionWorks ... ok
test thisOneFails ... FAILED: Mismatch(expected: 1, actual: 2)
test panicsAreReported ... FAILED: panic: index 3 out of bounds for list of length 1
```

and exits non-zero. Tests live next to the code they test, in the same
module, so they can call private functions. `veles run` and `veles
build` compile them out.

There is no assertion library in the bootstrap; a small `expectEq`
throwing a struct, as above, is the idiom. The error value is what gets
printed, so make it descriptive.

Next: [Concurrency](12-concurrency.md), or [Files, paths and processes](15-files-and-processes.md) for the modules a tool needs.
