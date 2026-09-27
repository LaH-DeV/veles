---
name: veles-develop
description: Implement a feature, fix or plan item in the Veles compiler, runtime or std end to end — picking the work, finding every layer a change touches, and finishing it to the repo's definition of done. Use for any change to lexer/parser/sema/codegen/runtime/std/format/lsp.
---

# Developing Veles

## 1. Pick and frame the work

- Next work = the first unfinished item in `veles-plan.md`, else an open
  `[ ]` in `veles-checklist.md` the user pointed at. Read the item's
  acceptance list; that is the target, not your own idea of it.
- If the item needs a design choice not already in the spec, stop and use
  `veles-decide`. Read the relevant `D` entry in `veles-spec.md` before
  touching semantics — the spec wins over the current code.
- Performance work starts with a measurement (`veles-bench`), not a theory.

## 2. Find every layer the change touches

A language feature is rarely one package. Walk this list and say for each
whether it applies:

| Layer | Where | Notes |
|---|---|---|
| tokens | `lexer/` | new keyword → also `editors/vscode/syntaxes/veles.tmLanguage.json` |
| syntax | `ast/`, `parser/` | `ast/print.go` `Dump` must print new nodes (the self-host oracle and `veles parse` depend on it) |
| formatter | `format/format.go` | every new node needs a printer case; `format_test.go` round-trips all sources |
| checking | `sema/` | diagnostics say what to do; add a fix (`c.errorFix`/`c.warnFix`) when the edit is mechanical |
| builtins | `sema/builtins_doc.go` | every built-in method gets a catalogue entry (hover, completion, `std/builtins.vs` stub) |
| codegen | `codegen/llvm/` | add or extend a golden fixture when IR shape changes |
| runtime | `runtime/c/` | C ABI small ints carry `zeroext`/`signext`; threads: every shared field is atomic or under a lock |
| std | `std/` | embedded — rebuild the compiler to see a std change; `veles-code` for conventions |
| tooling | `lsp/` (hover in `sema/index.go`, `lsp/completion.go`) | hover shows the declaration as the reader would write it |
| docs | `docs/documentation/`, `reference/` | `veles-docs` |

Grep for a sibling feature that already exists (e.g. how `static val` or
`enum` threads through) and follow its path; consistency beats invention.

## 3. Build it

- Smallest change that satisfies the acceptance list. No speculative
  options, no dead parameters, no TODOs left behind.
- Match the surrounding code: comment density, naming, the "why" comments
  above non-obvious rules (see `sema/removed.go` for the house style).
- Changing a spelling: old form still parses and errors with a fix; migrate
  std, examples and docs in the same change (`veles check <dir> --fix`
  applies fixes; it never writes embedded std — edit std by hand).
- Pin the behaviour first where you can: a failing `sema_test.go` case, a
  golden, or an example — then make it pass (`veles-test`).

## 4. Finish

1. `go build ./... && go vet ./... && staticcheck ./...` clean
   (`gofmt -l` is noisy on CRLF files — not a signal).
2. `go test ./...` green. If something else is red, check it at HEAD in a
   worktree before calling it pre-existing.
3. Runtime or codegen touched → run the relevant benchmark and say the
   number; concurrency touched → the thread stress in `veles-test`.
4. Docs, spec addendum (only for decided semantics), checklist tick,
   progress-log entry in `veles-plan.md` — dated, what landed, what was
   measured, bugs found on the way.
5. Report: files changed, tests added, what was not verified. Do not commit.
