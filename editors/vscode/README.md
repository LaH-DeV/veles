# Veles for Visual Studio Code

Syntax highlighting for `.vs` files plus a language client for `veles lsp`:
diagnostics as you type, hover with types and signatures, go to definition,
find all references, highlighting of the name under the cursor, rename,
inlay hints for inferred types and effects, signature help inside a call,
document outline, workspace symbols (Ctrl+T) and completion.

## Setup

1. Build the compiler: `go build -o veles.exe .` in the repository root.
2. Either put the binary on `PATH`, or set `veles.serverPath` to its full
   path. When the workspace root contains `veles.exe` (as the compiler
   repository does) it is used automatically.
3. Install the extension:

   ```bash
   cd editors/vscode
   npm install
   npm run compile
   npx vsce package          # produces veles-0.1.0.vsix
   code --install-extension veles-0.1.0.vsix
   ```

   or press F5 in this folder to launch an Extension Development Host.

## Commands

- **Veles: Run Current Package** — `veles run <dir of the active file>` in a terminal.
- **Veles: Test Current Package** — `veles test <dir>`.
- **Veles: Restart Language Server**.

Formatting (**Format Document**, or `editor.formatOnSave`) is served by the
same server — `veles fmt` inside the editor, using the `[format]` table of
the package's `veles.toml` when there is one.

Lints come with quick fixes: a warning that carries a correction shows the
light bulb, and **Quick Fix** (Ctrl+.) applies the edit. Today: a redundant
`mut` on a literal (removed), an unused binding (renamed to `_`), an
unreachable `else` on a sealed subject (removed), and a top-level implement that
belongs in the struct body (moved, across files when the struct lives in
another file of the module). `veles check
<dir> --fix` applies every such correction from the command line.

## Find references and rename

**Find All References** (Shift+F12) and **Rename Symbol** (F2) work on
anything you declared: functions, methods, fields, parameters, locals,
types, enum members. They follow a name everywhere the compiler resolved it
— across the package's modules, into `"$name"` interpolations and into
named arguments (`scale(factor: 2)`, `Point(x: 1)`). A trait's method and
every implementation of it are one name.

A rename is checked before it is applied: the server re-checks the package
with the new name and refuses — saying where — if any name would then
refer to something else (the new name shadows or is captured by another
declaration) or if it would add an error. A field written as a pun,
`Point(x)`, is spelled out (`Point(col: x)`) so the variable keeps its
name. It also refuses to rename a field or a sealed variant whose name a
derived `Codable` writes on the wire, since the program would compile and
read none of its old data back: put `@key("old")` on it first. Modules
(named by their directory), `this`, built-ins and the standard library are
not renamed.

## Inlay hints

The editor shows, in grey, what the compiler inferred and the source does
not say: the type of a binding written without one (`val n: i64 = ...`,
lambda parameters too), the return type of `fun f(x: i64) = x * 2`,
`suspends` on a function that suspends without declaring it, and the error
set of a bare `throws`. A binding whose type is already on the line —
`val p = Point(...)` — gets no hint, and neither does anything in a
generic body whose instances disagree. Toggle them with VS Code's
`editor.inlayHints.enabled`.

## How it works

The server is the compiler itself (`lsp/` in the repository). Every edit
re-parses and re-checks the package with the unsaved buffer overlaid on
disk, publishes diagnostics for every file of the package, and keeps a
reference index for hover and definition. Standard-library modules are
embedded in the binary, so definitions inside them are not navigable.
