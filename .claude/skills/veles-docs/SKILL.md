---
name: veles-docs
description: Write or update Veles documentation — tutorial chapters, the cheat sheet, the stdlib and error-message references, spec entries, doc comments, and the plan/checklist records. Use whenever a change is user-visible or a tracked item lands.
---

# Documenting Veles

Every ```` ```veles ```` block under `docs/documentation/` is compiled by
`go test ./docs`: checked, run if it has `fun main(`, and compared with a
following `Output:` ```` ```text ```` block. Docs that do not compile fail
the build, so examples in docs are real programs.

## What a change needs

| Change | Update |
|---|---|
| new/changed syntax | the tutorial chapter that teaches it, `reference/cheatsheet.md` |
| new/changed std API | the chapter for that module, `reference/stdlib.md` |
| new diagnostic users will meet | `reference/errors.md` — message, why, how to fix, `(Dn)` |
| a new area (module, big feature) | a new numbered chapter + a line in `index.md` under the right track |
| a decided rule | `veles-spec.md` (see `veles-decide`) |
| a tracked item done | tick `veles-checklist.md`; dated entry in the `veles-plan.md` progress log |
| built-in method | catalogue entry in `sema/builtins_doc.go` (hover text) |
| public std declaration | a `///` doc comment |

Readers are in three tracks (`index.md`): new to programming; coming from
Go/Kotlin/Swift/TS; systems and concurrency. Write for the chapter's track.

## Code blocks

- A complete program: `use` lines, `fun main()`, and an `Output:` block
  with the exact stdout. Prefer this — it proves the claim.
- `// fragment` as the first line: shown, never compiled. Use only for
  signatures and partial code; never to hide a program that does not work.
- A block that must fail to compile does not exist in the harness — show
  the error text in prose or a ```` ```text ```` block.
- Output must not depend on time, randomness, the machine, or task order.
- Idioms as in `veles-code`; code in docs is copied by readers.

## Prose

Match the existing chapters (e.g. `20-time.md`): say what a thing is for
and why it is shaped that way before how; name the mistake it prevents;
short concrete examples; compare with languages the reader knows where it
helps. No marketing words, no "simply", no filler introductions or
summaries, no emoji. A limitation is stated plainly, with the workaround.
Tutorials cite spec decisions as `D<n>`.

Records (`veles-plan.md` log, checklist rows, spec entries) are written for
the next agent: what landed, the rule, the numbers measured, the bugs
found and their cause, what is still open. Dates are absolute
(`2026-09-27`), never "today".

## Check

`go test ./docs` (about two minutes; run one chapter with
`-run 'TestDocs/20-time'`). Links between chapters resolve. `veles fmt
--check` style holds for code in docs as well — the formatter tests
round-trip docs blocks.
