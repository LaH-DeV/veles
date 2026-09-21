# Veles for Visual Studio Code

Syntax highlighting for `.vs` files plus a language client for `veles lsp`:
diagnostics as you type, hover with types and signatures, go to definition,
document outline and completion.

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

## How it works

The server is the compiler itself (`lsp/` in the repository). Every edit
re-parses and re-checks the package with the unsaved buffer overlaid on
disk, publishes diagnostics for every file of the package, and keeps a
reference index for hover and definition. Standard-library modules are
embedded in the binary, so definitions inside them are not navigable.
