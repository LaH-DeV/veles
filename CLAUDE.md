# Veles

Bootstrap compiler for the Veles language, in Go: `lexer` → `parser` → `sema`
(checking, effect inference, lowering to HIR) → `codegen/llvm` (textual IR,
linked by clang) + a C runtime (`runtime/c`) + a standard library written in
Veles (`std/`, embedded in the binary). README.md has the package map.

## Direction (user, 2026-09-28)

Every feature complete (all layers, all platforms), performant (measured),
manageable at the low level with the best developer experience at the high
level — and the language kept on course for self-hosting: ask "could the
compiler be written with this?" (veles-plan.md, Principles and track S).

## Where the truth lives

| File                              | Holds                                                                                                               | Rule                                                                                                                    |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `veles-spec.md`                   | language decisions, `D`/`M`-numbered                                                                                | IDs never renumbered; a superseded entry is struck through                                                              |
| `veles-checklist.md`              | production-readiness work; §9 open decisions (`Q`-labels, removed when decided), §10 decision log, §11 known limits | tick only what is built, tested and documented                                                                          |
| `veles-plan.md`                   | order of work (tracks A–F, S, P) + acceptance; progress log at the bottom                                           | after what the user asks, continue from the first unchecked line of "Where to continue"                                 |
| `notes_to_change.txt`             | the user's open design notes and standing directions                                                                | read at session start; never rewrite the user's text; a done item leaves the file (status lines of ours start with `→`) |
| `veles-selfhost-frontend-plan.md` | the Veles rewrite of lexer/parser                                                                                   | gates on byte-identical output vs the Go front end                                                                      |
| `veles-code-review-plan.md`       | the review of all Veles code (std, selfhost, examples): scope, checks, waves, findings log                          | no behaviour change without a decision; findings not fixed go to checklist §11                                          |
| `archive/`                        | finished plans, decided design notes, the old logs and notes                                                        | read-only history; nothing there is open                                                                                |

## Rules

- **Never `git commit` or `git push`** — the user commits. Leave changes in
  the working tree and report them.
- **Never `git stash`, `git checkout -- <path>`, `git reset --hard`,
  `git clean`.** The tree usually holds the user's uncommitted work, and
  `stash pop` rewrites LF files as CRLF on Windows. To see whether a failure
  predates you: `git worktree add <scratchpad>/head HEAD` and run it there;
  where that is not allowed, rerun the test alone several times and report
  it as unexplained rather than guessing whose it is.
- **Language and std design is the user's.** Any choice that changes what a
  program means, how it is spelled, or a public std API goes through the
  `veles-decide` skill. Do the decision-free work first.
- Removed syntax keeps parsing, with an error that names the new form and a
  fix (see `sema/removed.go`, `lint_index.go`); migrate the whole tree
  (std, examples, docs) in the same change.
- Refuse a footgun at compile time rather than documenting it.

## Machines

The user works on three: **Windows** (Git Bash, MSYS2 clang), **Linux**
(WSL on that Windows machine; CI) and **macOS** (Apple Silicon, Apple clang
from the Xcode Command Line Tools). Find out which one you are on before
reaching for a tool; a change to the runtime, codegen, std or a test's
expected output is verified on more than one when it can be.

| | Windows | Linux (WSL) | macOS |
|---|---|---|---|
| line endings | many files CRLF (`core.autocrlf=true`) | LF | LF (no autocrlf) |
| clang | MSYS2 UCRT64 (`C:\msys64\ucrt64\bin`), or `VELES_CLANG` | clang 18 from apt | Apple clang |
| bulk edits | a small Go program (no Python) | Go program or Python 3 | Go program or `python3` |
| `sed -i` | strips CRLF — do not use | GNU: `sed -i 's/a/b/'` | BSD: `sed -i '' 's/a/b/'` |
| debugger | `gdb` (MSYS2) | `gdb` | `lldb` |
| `--sanitize` | needs MSYS2's compiler-rt; the tests skip it | yes | yes |
| full gate | `go test ./...` from Git Bash | `internal/wsl-test.sh` (below) | `internal/macos-test.sh` |
| scratchpad | the session's own directory, never `%TEMP%` itself | session directory | session directory |

**Everywhere:**

- Prefer the Edit tool for existing files and the Write/create tool for new
  ones (new `.vs` files especially): shell heredocs eat one level of `\`, and
  the Edit tool decodes `\uXXXX`. Files with literal `\n` in Go strings
  (`lsp/server_test.go`, `driver/*_test.go`) are safest to patch with a
  program that reads and writes the file whole.
- perl `s{..}{..}` with an unbalanced brace in the replacement corrupts the
  file; `perl -0pi` with `\n` patterns silently does nothing on CRLF files.
- Probe programs go in a directory of their own: stray `.vs` files next to
  a program load as sibling modules.
- Never hand-edit `examples/*/expected.txt` (compared byte for byte,
  `-text` in `.gitattributes`); regenerate with `-update` (see `veles-test`).
- `gofmt -l` lists files that were unformatted before you (and every CRLF
  file on Windows): judge only what you touched.
- Do not run two `go test ./driver` at once on one machine.

**Linux from Windows.**
`wsl -d Ubuntu -- bash /mnt/c/Users/lahpo/Code/veles/internal/wsl-test.sh [cmd]`
runs `cmd` (default: build, vet, every test) against this working tree in a
clone at `~/veles` inside WSL, CRLF stripped; from Git Bash set
`MSYS_NO_PATHCONV=1` first, and write a `$` meant for `cmd` as `\$` (wsl.exe
re-parses its command line). Go is in `~/.local/go` there.

**macOS.** `internal/macos-test.sh [cmd]` runs `cmd` (default: build, vet,
every test, with the JSON test events and stderr kept in a directory it
prints; `VELES_TEST_LOG_DIR` picks its parent) with Go 1.27.1.

**CI.** `.github/workflows/ci.yml` runs build, vet and every test on all
three for a push; `nightly.yml` fuzzes and runs the sanitizer, GC-pressure
and thread-stress suites (both written 2026-10-30, not yet seen running —
checklist §11). They check Windows out with `core.autocrlf=false`, so a
CRLF-only failure shows on a developer's Windows checkout, not in CI.

**Go 1.27.1 is pinned** (`go.mod`; the scripts set `GOTOOLCHAIN=go1.27.1`):
the generated Unicode tables follow that Go's Unicode data, so move the pin
only together with regenerating them (README). A newer local Go (`go
version` says 1.27.2 on the Mac) is used by a plain `go test`; set
`GOTOOLCHAIN=go1.27.1` or use the script when the exact version matters, and
say which one a reported run used.

## Done means

`go build ./... && go vet ./... && go test ./...` green (`go test -short
./...` is the quick check while working, not the gate); the behaviour pinned
by a test; docs updated; the checklist ticked and a line in the plan's
progress log when a tracked item closes. Then report what changed, which
machine and Go the runs were on, and what was not verified (the other
platforms included).

Whatever is not done, missing or cut short goes into `veles-checklist.md` §11
(or the plan) in the same change, so it is built later rather than lost
(user, 2026-10-04).

## Skills

`veles-develop` (implementing), `veles-code` (writing Veles source),
`veles-test`, `veles-debug`, `veles-review`, `veles-bench`,
`veles-decide`, `veles-docs`.
