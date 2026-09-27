# Review fixes — 2026-09-25

Engineering findings from a review pass over the Go bootstrap compiler. These
are harness and robustness issues, not language gaps: the language work is
tracked in `veles-checklist.md` and this file deliberately stays off that
ground. Ordered by what buys the most.

State at the time of review: `go build ./...` clean, `go vet ./...` silent,
`go test ./...` green, one TODO in the whole tree.

## 1. The end-to-end suites can pass against a stale binary

**Where:** `docs/docs_test.go:29`, `examples/examples_test.go:28`

Both suites build the compiler only when the binary is missing:

```go
if _, err := os.Stat(veles); err != nil { /* go build */ }
```

Go's test cache keys on the *test* package's inputs, not on the compiler's Go
sources. So editing the compiler and running `go test ./...` re-reports both
suites from cache. Verified: appending a line to `codegen/llvm/types.go` and
running `go test ./examples/ ./docs/` printed `ok (cached)` for both.

This is the most important item on the page, because of §2 — these two suites
are the *only* thing covering code generation.

- [x] Remove the `os.Stat` shortcut in both tests; always build.
- [x] Build into `t.TempDir()` so the binary is per-run and cannot go stale,
      and so a stray `veles.exe` in the tree is never what gets tested.
- [x] Make the build failure `t.Fatal` with the combined output (already the
      case — keep it when restructuring).
- [x] Optionally defeat the cache deliberately: have the test read the built
      binary's path through `os.Open` so the cache tracks it, or accept that a
      fresh `t.TempDir()` build makes the question moot.

Rough shape:

```go
veles := filepath.Join(t.TempDir(), "veles")
if runtime.GOOS == "windows" { veles += ".exe" }
build := exec.Command("go", "build", "-o", veles, "..")
if out, err := build.CombinedOutput(); err != nil {
	t.Fatalf("building compiler: %v\n%s", err, out)
}
```

## 2. `codegen/llvm` has no direct coverage

**Where:** `codegen/llvm/` — 5,076 lines, 0.0% of statements

Per-package coverage at review time:

| package        | coverage |
| -------------- | -------- |
| format         | 96.1%    |
| parser         | 86.4%    |
| lsp            | 76.7%    |
| sema           | 76.3%    |
| driver         | 32.4%    |
| lexer          | 20.5%    |
| codegen/llvm   | 0.0%     |

No test imports `codegen` at all. Its only exercise is black-box, through the
compiled binary, in the two suites from §1 — so the most intricate package in
the tree (`expr.go` is 1,654 lines; `builtin` alone is a 379-line switch) is
guarded by the one harness that can silently go stale.

- [x] Fix §1 first — that alone restores the guarantee that codegen results
      are real.
- [x] Add a golden-IR test in `codegen/llvm`: lower a handful of small
      modules through `sema` and compare the emitted `.ll` against a checked-in
      fixture, with `-update` to regenerate. Cheap, and it makes the diff of a
      codegen change readable.
- [ ] Cover the shapes the examples do not reach: each `Builtin` op, each cast
      pair in `expr.go:730`, map operations in `maps.go`, and the suspension
      paths in `coro.go`.
- [ ] Raise `lexer` (20.5%) and `driver` (32.4%) opportunistically — lower
      stakes, since `parser` and `format` exercise the lexer indirectly.

## 3. A codegen panic reaches the user as a Go stack trace

**Where:** `codegen/llvm/expr.go:371`, `expr.go:619`, `expr.go:730`,
`expr.go:1394`, `gen.go:649`, `types.go:84`, `maps.go:36`, `sema/enum.go:332`

Twelve `panic` sites encode "this construct is unsupported". That is a
defensible strategy for can't-happen states, but nothing recovers them on the
compile path. `lsp/server.go:178` wraps requests in `recover()` so a crash does
not take down the editor session; the driver has no equivalent.

- [x] Add a `recover()` at the top of the compile driver that prints
      `internal compiler error: <msg>` plus the source position being lowered,
      a request to report it, and exits non-zero — no Go stack trace by
      default.
- [x] Gate the full stack behind an env var or `-debug` flag so the trace is
      still one flag away when reporting a bug.
- [x] Where a panic is genuinely reachable from valid-but-unimplemented Veles
      source, convert it to a proper diagnostic with a span instead.

## 4. `veles fmt` rewrites source files non-atomically

**Where:** `driver/fmt.go:78`

```go
if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
```

A crash, a full disk, or a kill mid-write leaves the user's source truncated.
A formatter is the one tool that runs over a whole tree of files someone cares
about, so the failure mode is bad out of proportion to its likelihood.

- [x] Write to a temp file in the *same directory*, then `os.Rename` over the
      original — same-directory keeps the rename atomic on one filesystem.
- [x] Preserve the original file's mode rather than hardcoding `0o644`.
- [x] Clean up the temp file when the rename fails.

## 5. `filepath.WalkDir`'s error is discarded

**Where:** `driver/fmt.go:37`

The walk's return value is ignored, and the callback swallows per-entry errors
with `return nil`. A walk that fails at the root formats nothing and still
exits 0 — `veles fmt --check` would report success on a tree it never read.

- [x] Capture the `WalkDir` error, report it on stderr, and set `code = 1`.
- [x] Decide explicitly whether a per-entry error should be reported rather
      than skipped; if skipping is intended, say so in a comment.

## 6. Hygiene

- [x] Delete the stray `veles.exe~` (8 MB) from the working tree. It is
      gitignored, but it is exactly the kind of stale artifact §1 feeds on.
- [ ] `notes_to_change.txt` is 108 KB at the root alongside four planning
      documents. Consider folding the live parts into `veles-checklist.md` and
      moving the rest under `docs/` or out of the tree.

## Not issues

Noted so a later pass does not re-litigate them:

- The `_ = ` sites in `parser/` (`p.expectIdent()` and friends) are the
  deliberate expect-and-recover idiom — the callee records the diagnostic.
  Correct as written.
- Fuzz corpora under `*/testdata/fuzz/` are force-added past the
  `**/testdata/*` ignore. That is the right call and easy to get wrong.
- The long `switch` functions in `codegen/llvm` and `sema` are flat dispatch,
  not deep nesting. Normal for a compiler; no restructuring warranted.

## Resolution — 2026-09-25

- §1: `internal/buildtest` builds the compiler into `t.TempDir()` for both
  suites and first *reads* every compiler source (Go files, the embedded
  `std/*.vs` and `runtime/c/*`), so Go's test cache keys on them. Verified:
  an unchanged tree reports `(cached)`; appending a line to
  `codegen/llvm/types.go` reran examples (39 s) and docs (103 s); appending
  one to `std/path/path.vs` reran examples.
- §2: `codegen/llvm/golden_test.go` + `golden/<name>/{main.vs,main.ll,output.txt}`
  — eight fixtures (arith with every cast, control, sealed, collections,
  closures, traits, tasks, strings). The golden keeps only the fixture module's own
  IR (`ModuleIR`), with string constants renumbered in first-use order so a
  prelude edit cannot shift them. Each fixture is also built with clang and
  run, so the IR is valid and computes the right thing. Coverage of
  `codegen/llvm`: 0% → 56.8%. `-update` regenerates.
- §3: `driver.Guard` wraps every subcommand (`main.go`); a panic prints
  `internal compiler error: …`, the function being checked or generated
  (`source.SetWhere`, one store per function) and the VELES_DEBUG=1 hint,
  and exits 3. With VELES_DEBUG set the trace follows and the panic
  continues. Panic sites audited: sema admits only numeric casts and
  conversions, which codegen covers pair for pair; the others guard HIR
  shapes sema never produces. None is reachable from a checked program, so
  none became a diagnostic. (A Go runtime *fatal* error — stack exhaustion —
  cannot be recovered and still prints Go's own report.)
- §4/§5: `writeAtomic` (temp in the same directory, `Sync`, original mode,
  rename, cleanup on failure). Walk errors are printed and make the exit
  code 1; the walk continues past an unreadable entry so the rest of the
  tree is still reported. Tests in `driver/fmt_test.go`.
- §6: `veles.exe~` deleted. `notes_to_change.txt` left for the user
  (`veles-plan.md` Phase 3, item 8).
