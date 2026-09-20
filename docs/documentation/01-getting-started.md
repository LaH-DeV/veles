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

Create a directory with one file, `main.vs`:

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
| `veles build <dir> -o app` | produce an executable |
| `veles check <dir>` | type-check without compiling; `--fix` applies lint corrections |
| `veles test <dir>` | run every `@test` function |
| `veles build <dir> --release` | optimise, and drop integer overflow checks |

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
with syntax highlighting, live diagnostics, hover, go-to-definition and
completion. Its `README.md` has three-line install instructions. The
language server is the compiler itself (`veles lsp`), so what the editor
says and what the compiler says never disagree.

The compiler is also the formatter: `veles fmt <files or directories>`
rewrites sources in the canonical style (`--check` only lists what would
change, `--stdout` prints one file), and the extension formats on save
when VS Code's `editor.formatOnSave` is on. The style is fixed like
prettier's — spacing, indentation and alignment are the formatter's job,
and a `{ }` block always breaks onto its own lines — but the line breaks
you choose inside argument lists, literals and method chains, and blank
lines between statements, are kept.

## When something goes wrong

The compiler reports errors with the source line and a caret, and most
messages name the design decision they enforce (`D25`, `M5`, …) so you can
look it up in `veles-spec.md`:

```text
main.vs:4:21: error: type mismatch: expected 'i64', found 'string'
  val n: i64 = "one"
               ^^^^^
```

Fix the first error and re-run; later errors are often consequences of
it.

Next: [Values, types and strings](02-values-and-strings.md).
