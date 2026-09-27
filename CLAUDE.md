# Veles

Bootstrap compiler for the Veles language, in Go: `lexer` → `parser` → `sema`
(checking, effect inference, lowering to HIR) → `codegen/llvm` (textual IR,
linked by clang) + a C runtime (`runtime/c`) + a standard library written in
Veles (`std/`, embedded in the binary). README.md has the package map.

## Where the truth lives

| File | Holds | Rule |
|---|---|---|
| `veles-spec.md` | language decisions, `D`/`M`-numbered | IDs never renumbered; a superseded entry is struck through |
| `veles-checklist.md` | production-readiness work; §9 open decisions, §10 decision log, §11 known limits | tick only what is built, tested and documented |
| `veles-plan.md` | order of work + acceptance; progress log at the bottom | continue from the first unfinished item |
| `notes_to_change.txt` | the user's own design notes | read at session start; never rewrite the user's text |
| `veles-selfhost-frontend-plan.md` | the Veles rewrite of lexer/parser | gates on byte-identical output vs the Go front end |

## Rules

- **Never `git commit` or `git push`** — the user commits. Leave changes in
  the working tree and report them.
- **Never `git stash`, `git checkout -- <path>`, `git reset --hard`,
  `git clean`.** The tree usually holds the user's uncommitted work, and
  `stash pop` rewrites LF files as CRLF. To see whether a failure predates
  you: `git worktree add <scratchpad>/head HEAD` and run it there.
- **Language and std design is the user's.** Any choice that changes what a
  program means, how it is spelled, or a public std API goes through the
  `veles-decide` skill. Do the decision-free work first.
- Removed syntax keeps parsing, with an error that names the new form and a
  fix (see `sema/removed.go`, `lint_index.go`); migrate the whole tree
  (std, examples, docs) in the same change.
- Refuse a footgun at compile time rather than documenting it.

## Windows hazards (all have bitten before)

- Many files are CRLF (`core.autocrlf=true`). `sed -i` strips CRLF;
  `perl -0pi` with `\n` patterns silently does nothing on them. Use the Edit
  tool. For bulk edits write a small Go program in the scratchpad (own
  `go.mod`, build it, run the exe) — there is no Python.
- perl `s{..}{..}` with an unbalanced brace in the replacement corrupts the
  file. The Bash tool eats one level of `\` in heredocs; the Edit tool
  decodes `\uXXXX`. Files with literal `\n` in Go strings
  (`lsp/server_test.go`) are patched with a Go program.
- Write new `.vs` files with the Write tool, not heredocs.
- Probe programs go in the scratchpad, never `%TEMP%` directly: stray `.vs`
  files next to a program load as sibling modules.
- Never hand-edit `examples/*/expected.txt` (CRLF lines + a bare-LF
  `exit=N`); regenerate with `-update` (see `veles-test`).

## Done means

`go build ./... && go vet ./... && go test ./...` green; the behaviour
pinned by a test; docs updated; the checklist ticked and a line in the plan's
progress log when a tracked item closes. Then report what changed and what
was not verified.

## Skills

`veles-develop` (implementing), `veles-code` (writing Veles source),
`veles-test`, `veles-debug`, `veles-review`, `veles-bench`,
`veles-decide`, `veles-docs`.
