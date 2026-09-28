---
name: veles-test
description: Run, choose, write and update Veles tests — sema rule tests, examples with expected output, tested docs, codegen goldens, formatter/LSP tests, runtime stress under threads and GC pressure, and fuzzing. Use when verifying a change, adding a regression test, or a test is failing.
---

# Testing Veles

The end-to-end suites (`examples/`, `docs/`) build a fresh compiler into a
temp dir every run (`internal/buildtest`) and key the Go test cache on
every compiler source, so `(cached)` is trustworthy and a stale
`veles.exe` in the tree is never what is tested.

## Pick the cheapest test that pins the behaviour

| Behaviour | Test | Where |
|---|---|---|
| a checker rule / diagnostic text | `expectError` / `expectClean` / `expectWarning` table case | `sema/sema_test.go` (sources start with `prelude`) |
| a new diagnostic (any `errorf`/`warnf`/`errorFix`/`warnFix`) | a line ending `// error: text` or `// warning: text` in a case file, strict both ways; declaration-level and body-level mistakes in separate files (the first stops the second) | `sema/testdata/conform/D<nn>-topic.vs`. A full `go test ./sema` fails for a diagnostic no test provokes unless `uncovered.txt` lists it (with `# why` when it cannot be reached); `-conform-update` rewrites the list. Its wording must also fall in a family (`source/family.go`, D79) whose `### name` section is in `reference/errors.md` — `TestEveryDiagnosticHasAFamily` (sema) and `TestEveryFamilyIsExplained` (docs) |
| an IR change per build profile | `main.release.ll` + `output.release.txt` next to the golden | `codegen/llvm/golden/<name>/` |
| a program's output | an example dir with `main.vs` + `expected.txt` | `examples/<name>/` |
| a CLI program | `commands.txt` (one invocation per line, `NAME=v` env prefixes, `fixtures/` copied in) | `examples/<name>/` |
| a one-file script | `name.vss` + `name.expected.txt` (+ `name.stdin.txt`) | `examples/<dir>/` |
| IR shape | golden fixture `main.vs` → `main.ll` + `output.txt` | `codegen/llvm/golden/<name>/` |
| formatting | `TestStyle` case | `format/format_test.go` |
| hover/completion/fixes | in-memory client (0-based line/char — count the fixture's lines) | `lsp/server_test.go` |
| runtime under threads | `driver/runtime_test.go` pattern: build once, run with `VELES_THREADS=…` | `driver/` |
| a tutorial snippet | the doc block itself | `docs/` (see `veles-docs`) |

A test must fail without the fix. Check it: revert the fix (in a worktree
or by hand), see red, restore.

## Commands

```bash
go test ./...                                          # everything, ~2 min (run in background)
go test ./sema -run TestSpecRules                      # one table
go test ./examples -run TestExamples/json              # one example
go test ./examples -run TestExamples/json -update      # rewrite its expected.txt
go test ./codegen/llvm -update                         # rewrite goldens
go test ./docs                                         # every doc block
```

After `-update`, read the diff of the rewritten file — an update that
blesses a wrong output is the easiest way to hide a bug.

## Stress: concurrency and GC

Programs must not depend on which task runs first. For any change to the
runtime, codegen of tasks/channels, or a concurrent example:

- `VELES_THREADS=1`, `2`, `4`, `8`, each repeated (20+ runs): build every
  affected program once, loop the binary from a scratchpad script.
- `VELES_GC_THRESHOLD=256 go test ./examples` — collects constantly.
- `VELES_GC_POISON=1` — swept objects are filled with 0xCD, so a use after
  free crashes instead of passing.
- `go test ./examples -sanitize` — every example built `--sanitize` (the C
  runtime under ASan + UBSan) with the collector running every 4 KiB; a
  sanitizer report fails it. Linux only for now (MSYS2 lacks compiler-rt).
- All of the above on Linux too, through `internal/wsl-test.sh` (CLAUDE.md).

## Fuzzing (by hand; there is no CI)

```bash
go test -fuzz=FuzzParse -fuzztime=2m ./parser
go test -fuzz=FuzzFormat -fuzztime=2m ./format
go test -fuzz=FuzzCheckMutated -fuzztime=4m -parallel=8 ./sema
```

`FuzzCheckMutated` finds far more than `FuzzCheck`. A crasher lands in
`<pkg>/testdata/fuzz/` and then runs with the ordinary tests — minimise it,
fix it, keep it.

## When a test fails

- Is it yours? Run it at HEAD in `git worktree add <scratchpad>/head HEAD`.
  Never stash.
- Flaky only under threads = a real race or an ordering assumption; do not
  retry until green. See `veles-debug`.
- A pre-existing failure is reported to the user, not silently fixed or
  "updated" away.
