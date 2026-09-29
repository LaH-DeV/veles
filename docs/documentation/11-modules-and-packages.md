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
are reached through that name: `geometry.Point`, `geometry.norm()`. The
prefix says where a thing comes from, and two modules may both have a
`Point` without anyone renaming anything. When the prefix is long, rename
the module with `as`. One `use` lists any number of imports, separated by
commas:

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

### Names you use all the time

A name written on every other line does not need its prefix. List it in
braces after the module (D85):

```veles
use io { println, eprintln as warn, readLine }, http { Request, Response }

fun main() {
  println("hello")           // bare
  warn("to standard error")  // renamed with `as`, like a module
  io.println("still works")  // the module keeps its name too
}
```

Functions, types (structs, enums, sealed types, traits) and `static val`s
can be named. A name is the module's own symbol, so `Request` means
exactly what `http.Request` means. The rules are short:

- `use m { … }` never removes `m.`: braces add bare names on top.
- There is no `{ * }`: name what you use, so a reader sees where each
  name comes from.
- Two imported names may not be the same, nor a name this module
  declares; the message says to rename one with `as`.
- A parameter or local of the same name shadows an imported one, as it
  does a module.
- A name nothing uses is a warning (`veles check --fix` removes it).
- `veles fmt` sorts the names inside the braces.
- Completion inside the braces offers the module's names; the old
  `use m.{ … }` spelling is an error whose fix removes the dot.

The editor does the bookkeeping. Typing the start of a name a module
offers — `readL` — completes to `readLine` and adds it to the braces of
your `use io { … }` (or writes the line) in the same edit. A renamed name
is a declaration of its own at the alias: hover says `alias of
io.eprintln`, and renaming `warn` changes the alias and its uses and
nothing of the module's.

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
changes it". What leaves the *package* is what its root module re-exports
with `public use`, below.

## Packages and `veles.toml`

A **package** is the directory tree under a `veles.toml`. The manifest
holds the package's identity and dependencies; source files never do
(M1):

```toml
[package]
name = "mathlib"
version = "0.1.0"

[dependencies]
utils = "../utils"            # a path dependency

[format]
indent = 2                    # spaces per level, or "tab"; the default is 2
max_blank_lines = 1           # consecutive blank lines kept by `veles fmt`

[native]
static-libs = ["z"]           # C libraries the package's extern blocks need (chapter 13)
```

- What other packages may import is written in source, not here: see
  [What a package shows](#what-a-package-shows) below (D89).
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

### What a package shows

`public` makes something visible to the *rest of the package*. What other
packages may reach is what the package's **root module** (`lib.vs`, or any
file of the root directory) re-exports with `public use` (D89):

```veles
// fragment: mathlib/lib.vs
public use geometry                    // `use mathlib.geometry` works
public use support { root as sqrt }    // `mathlib.sqrt(x)`; `support` stays hidden

public fun twice(x: i64): i64 = x * 2
```

- `public use geometry` re-exports the module under its name; `as` gives it
  another (`public use geometry as geo` is `use mathlib.geo`).
- `public use support { root as sqrt }` puts those items into the module
  itself, flattened: `use mathlib { sqrt }` or `mathlib.sqrt(x)`, while the
  module `support` is not reachable — a facade that hides the layout.
- A module that is not re-exported cannot be imported from another package:
  `module 'hidden' of package 'mathlib' is not re-exported; add 'public
  use hidden' to its root module`.
- `public use` in any module works the same way for that module's importers,
  so a module can pass on what it re-exports: `use mathlib.geometry.deep`
  is fine when `geometry` says `public use deep`.
- Only the package's own modules can be re-exported, not a dependency's or
  a standard module's. The old manifest `exports` list is gone; a manifest
  that still has one gets an error naming this form.

`examples/packages` in the repository is a two-package project you can
run: `veles run examples/packages/app`.

### API documentation

`///` comments are documentation: on a declaration they describe it, at
the top of a file they describe the module. `veles doc` renders a
package's public surface as Markdown — each module a reader outside the
package can reach (the root and what it re-exports), each public
declaration spelled as the editor's hover spells it, its comment, and the
comments of its documented members:

```bash
veles doc                 # every module, on standard output
veles doc . -o api        # api/<module>.md, one file per module
```

Private members and modules the root does not re-export are left out;
a package with errors is refused, since its API is not settled.

## The standard library

`use io` is the only import you need for the tutorials. The **prelude**
— `Iterator`, `Iterable`, `Closeable`, `Mutex`, `Atomic`, `Panic`,
`Some`/`None`/`Ok`/`Err` — is in scope without any `use`. The
[standard library reference](reference/stdlib.md) lists everything.

## Tests

A test is `test "what it checks" { ... }` in the module it tests, so it
can call private functions. `veles test <dir>` compiles the module with
its tests and runs each as its own task; `veles run` and `veles build`
leave them out. Inside a test, `expect(cond)` records a failure and goes
on, and `require(x)` unwraps a `T?` or a `Result` or ends the test:

```veles
use io

fun add(a: i64, b: i64) = a + b

test "adds small numbers" {
  expect(add(2, 2) == 4)
  expect(add(-1, 1) == 0)
}

fun main() {
  io.println("ordinary run; tests are skipped")
}
```

Output:
```text
ordinary run; tests are skipped
```

A module with many tests keeps them in `*.test.vs` files beside its
source — part of the module, never part of a build.
[Chapter 14](14-attributes-and-testing.md#writing-tests) has the whole
vocabulary, the report, helpers and the runner's flags.

Next: [Concurrency](12-concurrency.md), or [Files, paths and processes](15-files-and-processes.md) for the modules a tool needs.
