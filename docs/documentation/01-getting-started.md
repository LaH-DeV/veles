# 1. Getting started

## What Veles is

Veles is a compiled, garbage-collected language with a static type system
that stays out of your way: types are inferred inside functions, errors
and concurrency are inferred across them, and there is exactly one way to
do most things. Programs compile through LLVM to native executables.

If you know Kotlin or Swift the syntax will feel familiar. If you know
Go, the module system will. If you know Rust, you will notice what was
left out on purpose: no lifetimes, no borrow checker, no `async` keyword
to remember.

## Installing the toolchain

You need Go 1.23+ and `clang` (LLVM 17 or newer) on your `PATH`, or the
environment variable `VELES_CLANG` pointing at a clang executable.

```bash
git clone <this repository> veles
cd veles
go build -o veles .
```

Put the resulting `veles` (or `veles.exe`) on your `PATH`.

## Hello

The quickest start is `veles new`, which makes a package that runs and
tests on the first try — a manifest, a `main.vs` with a test, and a
`.gitignore`:

```bash
veles new hello
cd hello
veles run      # Hello, world!
veles test     # test greets by name ... ok
```

By hand it is just as short. Create a directory with one file, `main.vs`:

```veles
use io

fun main() {
  io.println("Hello, Veles!")
}
```

Output:
```text
Hello, Veles!
```

Run it:

```bash
veles run hello/
```

`veles run` compiles the directory and executes the result. There is no
file to name: a **module is a directory**, every `.vs` file inside it
shares one namespace, and `fun main()` is the entry point.

For a program that fits in one file, save it as `hello.vss` instead — a
**script** — and run the file:

```bash
veles run hello.vss
```

A script is a package of its own: it does not see the other files in its
directory, so a folder of scripts can hold a `main` in each. It imports
the standard library like any other file (`use io`), but nothing else.

The other commands you will use:

| Command | What it does |
|---|---|
| `veles new <dir>` | create a package: manifest, `main.vs` with a test, `.gitignore`; `--template server` makes an HTTP service (health check, logging, graceful stop, tests of its handlers) |
| `veles doc [dir]` | the package's public API as Markdown (`-o dir` for one file per module) |
| `veles fetch [dir]` | download the git and registry dependencies into the module cache and record them in `veles.sum` (a build does this too; for CI, and for the editor, which never downloads) |
| `veles add <spec>[@version]`, `update <name>` or `update --all`, `remove <name>`, `deps [--why name]`, `vendor` | manage dependencies: edit `veles.toml` in place, show the build as a tree, copy it into `vendor/` (chapter 11) |
| `veles audit [dir] [--detail]` | what each dependency can do (`unsafe`, `extern`, `native`, `net`, `fs`, `os`, `ffi`), computed from its source, and whether the `[policy]` in `veles.toml` allows it; exit 1 if not (chapter 11) |
| `veles publish [dir]`, `veles yank <owner/name>@<version>`, `veles attest keygen|sign|verify|import` | publish a package to a registry, withdraw a version, and sign / check / import reviews of dependencies (chapter 11) |
| `veles build <dir> -o app` | produce an executable (named after the package without `-o`) |
| `veles check <dir>` | type-check without compiling; `--fix` applies lint corrections |
| `veles explain <family>` | what a kind of error means and how to fix it (the `see:` line) |
| `veles test <dir>` | run every `test "..." { }` (chapter 14), several at once (`--jobs 1`: one at a time) |
| `veles build <dir> --release` | optimise, and drop integer overflow checks |
| `veles build <dir> --sanitize` | the C runtime and the link under AddressSanitizer and UBSan (chapter 13) |
| `veles build <dir> --timings` | how long each phase took, and the modules that cost the most to check — for a build that feels slow |

Leave out `<dir>` inside a package and these take the current directory:
`veles run`, `veles test --filter parse`. Flags go before or after the
path.

## Reading the first program

- `use io` imports the `io` module of the standard library. Its
  functions are called with a prefix: `io.println`.
- `fun main()` declares a function with no parameters and no return
  value. Bodies are blocks in braces.
- Statements end at the end of the line. Semicolons exist but you will
  almost never write one.
- Comments are `//` to end of line or `/* ... */`. A comment written as
  `/// ...` lines or a `/** ... */` block directly above a declaration, a
  field or a method is its **documentation**: the editor shows it on hover
  (markdown), so write it the way you would a JSDoc or KDoc comment. A doc
  comment at the very top of a file, set off from the first declaration by
  a blank line, documents the **module** (shown when hovering its name).

## Setting up an editor

The repository ships a Visual Studio Code extension in `editors/vscode`
with syntax highlighting, live diagnostics, hover, go-to-definition,
find-references, a rename that refuses to change what the program means, and
completion. Its `README.md` has three-line install instructions. The
language server is the compiler itself (`veles lsp`), so what the editor
says and what the compiler says never disagree.

The compiler is also the formatter: `veles fmt <files or directories>`
rewrites sources in the canonical style (`--check` only lists what would
change, `--stdout` prints one file), and the extension formats on save
when VS Code's `editor.formatOnSave` is on. The style is fixed like
prettier's — spacing, indentation and alignment are the formatter's job,
and a `{ }` block always breaks onto its own lines — but the line breaks
you choose inside argument lists, literals, method chains and the
patterns of a `when` arm, and blank lines between statements, are kept.

## When something goes wrong

The compiler reports errors with the source line and a caret, and most
messages name the design decision they enforce (`D25`, `M5`, …) so you can
look it up in `veles-spec.md`:

```text
main.vs:4:21: error: type mismatch: expected 'i64', found 'string'
  val n: i64 = "one"
               ^^^^^
see: veles explain type-mismatch
```

Fix the first error and re-run; later errors are often consequences of
it. Every message belongs to a family, and the `see:` line under the
errors names each family that came up: `veles explain type-mismatch`
prints what that kind of error means and how it is usually fixed, from
[the error reference](reference/errors.md) the compiler carries, so it
works offline. In the editor the family is a link to the same page.

A name the compiler does not know comes with its best guess — a typo, or
what another language calls the same thing:

```text
main.vs:5:17: error: no method 'size' on type 'List<i64>'; did you mean 'len'?
main.vs:6:6: error: module 'io' has no declaration 'printn'; did you mean 'io.println'?
```

In the editor each guess is a quick fix. Fixes the compiler is sure of —
a removed spelling, an unused binding — are applied by
`veles check --fix`; a guess never is, since only you know what you
meant.

Next: [Values, types and strings](02-values-and-strings.md).
