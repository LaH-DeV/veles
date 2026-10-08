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
use http { Request, Response }
use io { eprintln as warn, println, readLine }

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

The order of imports means nothing, so `veles fmt` decides it: a run of
`use` lines is sorted — standard library modules first, then everything
else — with one `use` per module. `use fs, io, os` is the same three
imports written on one line, and a package that prefers that form sets
`imports = "merged"` in `[format]` (below): the formatter then writes one
comma-separated `use` per origin.

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

public fun distance(a: Point, b: Point): f64 => root((a.x - b.x) * (a.x - b.x) + (a.y - b.y) * (a.y - b.y))
public fun origin(): Point => Point(x: 0.0, y: 0.0)

fun root(x: f64): f64 => ...   // private helper
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
imports = "lines"             # one `use` per module; "merged" writes `use fs, io, os`

[lint]
implicit_return = "lambda"    # "full" (default) | "lambda" | "expr": where a body may end in its value

[native]
static-libs = ["z"]           # C libraries the package's extern blocks need (chapter 13)
```

- What other packages may import is written in source, not here: see
  [What a package shows](#what-a-package-shows) below (D89).
- `[lint]` turns on checks a team may want and the language does not
  require. Each is an error in this package's own code, with a fix that
  `veles check --fix` applies; dependencies and the standard library follow
  their own manifests. `implicit_return` says where a body's last
  expression may be its value without `return`: `"full"` (the default)
  anywhere; `"lambda"` in every lambda and in `fun f() => expr`, while a
  function's or method's `{ }` body ends in `return value`; `"expr"` only in
  a body without braces (`fun f() => expr`, `x => expr`), so every `{ }` body
  says `return`. The blocks of an `if` or a `when` arm are never affected:
  their value is the branch's, not the function's.
- A dependency is imported by its manifest name: `use utils`,
  `use utils.text`. A dependency is a path (`name = "path"` or
  `name = { path = "..." }`), a git repository or a registry package; see
  [The manifest in full](#the-manifest-in-full) and
  [Dependencies from git and the registry](#dependencies-from-git-and-the-registry).
- **A package is loaded once, whatever it is called.** If `a` and `b` both
  depend on `util`, or your manifest names the same directory twice, there is
  one `util`: one set of modules, types and module-level state. The name in
  `[dependencies]` is only your local alias. Two different packages that
  share a name (two versions of a library) are two packages and may be used
  side by side under two aliases.
- Only a package's own `[dependencies]` can be imported: if `a` depends on
  `util`, you still add `util` to your manifest to `use` it. A dependency
  name that is also a standard module (`io`) or a directory of your package is
  an error, so `use io` never means two things.
- A package cannot depend on itself, directly or through its dependencies.
- **A package is the program; a module is not.** `fun main()` lives in the
  root module. `veles run` or `veles build` pointed anywhere inside the
  package finds the manifest above it and runs the package — you cannot
  run `geometry/` on its own, and a package whose root has no `main` is a
  library. `veles check` accepts any module or library. Without a
  `veles.toml`, a directory is taken as its own single-module package.

### The manifest in full

`veles.toml` is TOML, the small part of it a manifest needs: tables,
strings, numbers, `true`/`false`, lists (over several lines, with a
trailing comma) and one-line `{ inline = "tables" }`. Dotted keys,
`[[arrays of tables]]`, floats and dates are refused with a message, and so
is a key or table the compiler does not know (a typo is never a setting
that does nothing).

```toml
[package]
name = "app"                  # required, except in a workspace root
version = "0.1.0"             # major.minor.patch; a release is a tag of this
description = "A notes app"
license = "MIT"
veles = "0.70"                # the oldest compiler that builds it

[dependencies]                # the key is the name `use` says
mathlib = "../mathlib"                                           # a path
httputil = { registry = "acme/httputil", version = "1.4.2" }     # the registry
pg = { git = "https://github.com/veles-db/pg", version = "2.1.0" }   # a git tag
fast = { git = "https://example.com/fast", commit = "3f2a9c1" }     # a commit

[dev-dependencies]            # only for `veles test` and *.test.vs
fakeclock = { registry = "lah/fakeclock", version = "0.3.0" }

[workspace]                   # a monorepo root: its members build together
members = ["app", "libs/mathlib"]

[policy]                      # what dependencies may do (see "What a dependency can do")
deny = ["unsafe"]
```

- A dependency says where it comes from with exactly one of `path`,
  `registry` or `git`. A registry or git dependency is pinned by an exact
  `version` (a tag); a git dependency may instead be pinned by `commit`
  (a hex prefix of 7 to 40 digits), never both. A bare string is always a
  path, so `utils = "1.4.2"` is an error that shows the table to write.
- A registry name is `owner/name`, lowercase letters, digits, `-` and `_`.
- A version is `major.minor.patch`, optionally `-pre-release`; no leading
  `v` in the manifest (a repository's tag may have one) and no `+build`.
  What makes two versions the same package is the **major**, by Cargo's
  rule: the first non-zero part, so `1.4.2` is major 1, `0.3.1` is major 0.3
  and `0.0.2` is major 0.0.2.
- `[workspace] members` lists directories below the root manifest, each with
  a `veles.toml` of its own; patterns such as `libs/*` are not supported
  yet. A root that only lists members needs no `[package]`.
- Every error names the file and the line: `veles.toml:7: dependency 'pg':
  a git dependency is pinned by exactly one of version (a tag) or commit`.

### Dependencies from git and the registry

```toml
[dependencies]
pg = { git = "https://github.com/veles-db/pg", version = "2.1.0" }
```

A build resolves these on its own; `veles fetch` does only that (use it in
CI to download once and build offline). [Chapter 25](25-packages-and-the-registry.md)
explains what a build checks and how to read a refusal.

- **A version is a tag.** `version = "2.1.0"` fetches the tag `v2.1.0` (or
  `2.1.0`) of the repository. Publishing is pushing a tag. A pre-release such
  as `2.2.0-beta.1` is used only when a manifest writes exactly that.
- **Minimal version selection.** What you write is a minimum. If your
  `[dependencies]` ask for `util` 1.0.0 and a package you use asks for 1.3.0,
  everyone gets 1.3.0, the highest any manifest asks for — never a newer
  release that nobody asked for, so a build does not change because a library
  published something. Moving up is explicit: edit the version.
- **One package per (source, major).** Two majors of one repository
  (`1.x` and `2.x`) are two packages that may be used together under two
  names; the same major is always one. The major is the first non-zero part
  of the version: `1.4.2` is 1, `0.3.1` is 0.3. A repository's source is its
  URL, so another spelling of it is another package.
- **A commit pin** (`commit = "3f2a9c1"`) names an untagged revision. A
  package that others depend on and pins a commit gets a warning: its users
  cannot run version selection over a commit. A tag and a commit pin of one
  (repository, major), or two commits of one repository, are an error that
  names both.
- **Fetched packages hold plain files.** A fetched package may depend on
  other fetched packages, never on a path; a symbolic link in a package is
  refused.
- **The module cache** is `~/.veles/pkg` (`VELES_HOME` moves it), one
  directory per version, never changed after it is written. Git is needed for
  git dependencies; the files are exactly the committed bytes, whatever
  `.gitattributes` or `core.autocrlf` say.
- **`veles.sum`** is committed with the source, beside the root manifest (the
  workspace root's, in a workspace): one line per package version in the
  build,

  ```text
  git https://github.com/veles-db/pg 2.1.0 h1:3b6f…
  git https://example.com/fast commit:3f2a9c1d… h1:9a01…
  registry acme/httputil 1.4.2 h1:77ce…
  ```

  The hash covers the exact bytes of every file, sorted by path. The first
  fetch records a line; afterwards a package that hashes differently is a hard
  error, and so is a cache entry edited after it was fetched. Nothing is
  built from either. If a tag was moved on purpose and you trust the new
  contents, delete its line.
- **Registry packages** (`{ registry = "owner/name", version = "1.4.2" }`)
  come from the proxy named by `VELES_PROXY`: `GET <proxy>/<owner>/<name>/@v/<version>.zip`,
  an archive with the package's files at its root. The registry service
  itself is not live yet, so without a proxy such a dependency says so; a git
  dependency always works.
- **The language server never downloads.** It reads the cache and says
  "not in the module cache; run `veles fetch`" otherwise.
- Nothing runs at build time: a package has no build script, and fetching
  only copies files.

### Managing dependencies

The commands edit `veles.toml` as text — one line changes; comments, order
and line endings stay — then resolve the result. If that fails (a version
that does not exist, a repository that cannot be reached) the manifest is put
back as it was.

```text
veles add github.com/veles-db/pg            # the newest release: pg = { git = "…", version = "2.1.0" }
veles add github.com/veles-db/pg@2.1.0      # a version
veles add acme/httputil --dev               # a registry package, for tests only
veles add ../libs/mathlib --as math         # a directory, under another name
veles update pg                              # the newest release of pg's major
veles update pg@3.0.0                        # that version (a newer major is never taken by itself)
veles update --all
veles remove pg
veles deps                                   # the build as a tree
veles deps --why util                        # every way util gets into the build
veles vendor                                 # copy the build into vendor/
```

- `add` takes a path (`../x`, `./x`), a git URL (`https://…`, `git@host:…`,
  or `github.com/owner/repo`) or a registry name (`owner/name`); `@version`
  after it pins a release, otherwise the newest stable tag is used — never a
  pre-release, which is added only when you name it. The local name is the
  last part of the source (`pg`), or the package's own name for a directory;
  `--as` chooses another. A repository on your own disk is a `file://` URL,
  since a bare path means a directory.
- `update` moves each dependency to the newest release of its **own major**
  and mentions a newer major when there is one; taking it is explicit
  (`update name@version`). A path or a commit pin has no version to update.
  `update` and `remove` also delete the lines of `veles.sum` that nothing uses
  any more (not in a workspace, where members share the file).
- `deps` prints each package once, `(*)` marking a second sight and
  `1.0.0 -> 1.3.0` where version selection raised what a manifest asked for.
  `--why` prints the chains from your package to the one you ask about.
- `vendor` copies every package version the resolution mentions into
  `vendor/` beside `veles.sum`. While `vendor/` exists, builds read packages
  from it and nowhere else — no network, no module cache — and still check
  each against `veles.sum`. A package missing from it says to run
  `veles vendor` again; delete `vendor/` to go back to the cache. `veles fmt`
  skips `vendor/`.

### What a dependency can do: capabilities and policy

Nothing runs when a package is fetched or built — there are no build
scripts — but a dependency's *code* runs inside your program, so you should
know what it can reach. The compiler works that out from the source; the
author declares nothing, so nothing can be understated:

| capability | the package's source… |
|---|---|
| `unsafe` | has an `unsafe` block or an `unsafe fun` |
| `extern` | has an `extern` block, or an `extern "C" fun` C can call |
| `native` | has a `[native]` table: it links C libraries |
| `net` | imports `net`, `tls`, `http`, `db` or `otel` |
| `fs` | imports `fs` or `config` |
| `os` | imports `os` (processes, the environment) or `config` |
| `ffi` | imports `ffi` |
| `unlisted` | is not in the registry's listed tier (any git package; not source-based, see below) |

Only modules whose purpose is the capability count: `http` reads the clock
and the environment inside, which does not make every program with a socket
"use os". Every directory below the root counts, even one named `vendor` or
holding a `veles.toml` of its own, because the compiler reads any directory
with sources as a module; test files and hidden directories are left out. A
directory with no sources is not a module, so it cannot hide `use fs`.

```text
$ veles add github.com/acme/httputil
added httputil = https://github.com/acme/httputil 1.4.2 to [dependencies]
httputil can use: net, fs (with its dependencies); `veles audit --detail` shows where

$ veles audit --detail
https://github.com/acme/httputil 1.4.2  git      net fs
    net    client.vs:1 use http
    fs     static.vs:2 use fs
https://github.com/veles-db/pg 2.1.0    git      net unsafe extern native
    ...
no [policy] in veles.toml: nothing is denied
```

`audit` lists every package of the build with its **own** capabilities, and
`(dependencies add: …)` for those it only gets through what it requires.
A **policy** makes a build refuse what you do not want:

```toml
[policy]
deny = ["net", "unsafe"]                              # no dependency may use these…
allow = { net = ["acme/httputil"], unsafe = ["pg"] }  # …except the packages named
```

- A package is named by its source (`acme/httputil`, or the git URL as it is
  written in the dependency) or by its own package name. `allow net` does
  not allow anything else.
- The policy covers registry and git packages. A path dependency is your own
  code, and so is the package itself.
- Every command that resolves dependencies enforces it — a build, `fetch`,
  `add` (a refused addition leaves `veles.toml` as it was), `update`. The
  error names the package, the first place it uses the capability and the
  `allow` line that would permit it. `veles audit` reports the same without
  refusing, and exits 1 when something is denied.
- In a workspace a member without a `[policy]` follows the root's.

### The registry: publishing, tiers and reviews

Besides git, packages come from a **registry**: `{ registry =
"acme/httputil", version = "1.4.2" }`, served from the address in
`[registry]`. The registry mirrors every version immutably (a deleted
repository cannot break a build) and tells the client more than a git
repository can. The protocol is in
[the registry protocol](reference/registry-protocol.md); how the whole
system works, what it protects you from and what it does not, and what each
refusal means are in [chapter 25](25-packages-and-the-registry.md).

```toml
[registry]
url = "https://registry.example.com"
key = "ed25519:…"          # the registry signs its metadata; pin its key
```

- **Pin the key.** The registry signs each version's metadata (its hash, tier
  and yank state). With the key in `veles.toml` a hijacked host cannot make up
  hashes; without it the first fetch is trusted and `veles.sum` guards the
  rest. The key changes by editing the manifest.
- **Tiers.** A version is *listed* (the registry's automated gates passed:
  1.0.0 or later, builds on Windows and Linux, its tests pass, no unexplained
  API break) or *unreviewed*. A git package is always unreviewed. `veles
  audit` shows the tier, and `deny = ["unlisted"]` in `[policy]` refuses
  anything that is not listed (`allow = { unlisted = ["acme/x"] }` exempts
  one).
- **Yank.** The author can yank a version (`veles yank acme/lib@1.0.0 --reason
  "…"`, `--undo`). It stays downloadable, but a resolution that has no
  `veles.sum` line for it — `add`, `update`, a fresh clone without a sum — is
  refused with the reason, and `update` never picks it. A project whose
  `veles.sum` already holds it keeps building and is warned.
- **Publishing.** Put `version` and `registry = "owner/name"` in `[package]`,
  and a token in `VELES_TOKEN` (or `~/.veles/token`), then `veles publish`: it
  checks the package, archives its files (not `.git`, `vendor/`, `bin/`,
  `veles.sum`, hidden files) and uploads them. A version cannot be replaced;
  publish the next one. A registry service may also publish a git tag by
  itself; that is the service's business, not the compiler's.
- **Reviews.** Anyone can sign a statement that they reviewed a package
  version, and whoever depends on it decides whose statements count:

  ```text
  veles attest keygen alice                  # ~/.veles/keys/alice.key; prints the public key
  veles attest sign httputil --key alice     # signs the exact bytes in your build
  veles attest sign httputil --key alice --push   # …and sends it to the registry
  veles attest verify                        # every statement the project holds, against the build
  veles attest import reviews.attest         # add statements from a file or URL, verified
  ```

  ```toml
  [policy]
  trust = { alice = "ed25519:…", bob = "ed25519:…" }   # whose reviews count
  require = { reviewed = 2 }                           # how many of them each dependency needs
  allow = { reviewed = ["acme/old"] }                  # exempt packages
  ```

  Statements live in `attestations/*.attest` next to `veles.sum` (commit
  them) and, for a registry package, at the registry. A review is about
  exact bytes: after the dependency changes, old reviews no longer count. The
  requirement covers registry and git packages, not path dependencies; a
  build that falls short says which package has how many reviews.

### Workspaces: several packages in one repository

A monorepo's root manifest lists its packages:

```toml
# veles.toml at the repository root
[workspace]
members = ["app", "libs/mathlib", "tools"]
```

Each member is a package with a `veles.toml` of its own, and members depend
on each other by path (`mathlib = "../libs/mathlib"`). Pointed at the root,
`veles check`, `veles test` and `veles build` (and `veles check --fix`) run
over every member in order, printing `==> libs/mathlib (mathlib)` before
each:

```text
veles test
==> app (app)
==> libs/mathlib (mathlib)
...
veles: 3 workspace members ok
```

- A failing member does not stop the rest, so one run shows every broken
  member; the exit status is 1 and the last line names them
  (`veles: 2 of 3 workspace members failed: mathlib, tools`).
- `veles build` builds each member that has a `main` and skips libraries;
  the executables go to `bin/<package name>` under the workspace root.
  `-o` is refused at the root (a workspace builds one executable per member),
  and so is `veles run`, which names the members to choose from.
- Pointed at anything inside a member, the tools behave as always: that
  member alone.
- A root that has a `[package]` too is the first member (`==> . (root)`).
  Member names are unique, and a member cannot itself be a workspace.
- Publishing members of a workspace is not part of this release.

### What a package shows

`public` makes something visible to the *rest of the package*. What other
packages may reach is what the package's **root module** (`lib.vs`, or any
file of the root directory) re-exports with `public use` (D89):

```veles
// fragment: mathlib/lib.vs
public use geometry                    // `use mathlib.geometry` works
public use support { root as sqrt }    // `mathlib.sqrt(x)`; `support` stays hidden

public fun twice(x: i64): i64 => x * 2
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

`examples/packages` in the repository is a small project you can run:
`veles run examples/packages/app` uses a path dependency (`mathlib`) and a
registry package (`greeter`) that is vendored in `vendor/` with its hash in
`veles.sum`, so it builds with no network.

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

fun add(a: i64, b: i64) => a + b

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
