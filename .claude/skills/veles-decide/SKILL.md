---
name: veles-decide
description: Prepare and ask a Veles language, syntax, semantics or public std API design decision, then record the answer in the spec, checklist and plan. Use whenever a change would alter what a program means, how it is spelled, or a public API — before building it.
---

# Design decisions

Veles is the user's language. Agents never settle a design question alone;
they make it cheap to answer well. Design goal the user set: allow what
Go/Rust/C allow, with the best developer experience possible.

## Is it a decision?

Yes: new or changed syntax, keywords, semantics, defaults, error vs warning,
panic vs throw, public std names and signatures, removing anything.
No: internal structure, test layout, performance work that keeps behaviour,
bug fixes that make code match the spec. When unsure, it is one.

Check first that it is not already decided: search `veles-spec.md`, the
checklist §10 log, `notes_to_change.txt` and (for older discussions)
`archive/notes-history.txt`. Several ideas were considered
and rejected (e.g. `!!`, a panic `catch`, `readonly`, classes, name
imports) — do not re-propose a rejected option without new evidence, and
say what is new.

## Prepare

Do all decision-free work first. Then, per decision:

1. **The problem** in two or three sentences, with the program that
   motivates it (from an example, a doc, a bug) — real code, not `foo`.
2. **Options**, usually three, always including "leave it as is" when that
   is viable. For each: the same motivating program written with it, the
   edge cases (generics, nullables, suspension, errors, threads, the
   formatter, hover), what Go/Rust/Kotlin/Swift/TS do, and pros/cons.
3. **Recommendation** and the one reason that decides it.
4. **Cost**: what it touches (compiler layers, std, docs, migration).
5. **Both levels and self-hosting** (user's principles, 2026-09-28): how
   the high-level spelling reads, what the low-level escape is, and — when
   it applies — what the option means for writing the compiler in Veles.

Language facts in the options must be checked against the compiler — run
the snippets that should compile today.

## Ask

Present all pending decisions in one message with the options written out
in full, then call `AskUserQuestion` (recommended option first, marked).
Batch related decisions; do not drip them one per turn. If the user says
"idk yet", record it as open and move on to other work.

## Record the answer

1. `veles-spec.md`: a new `### Dn — Title (vX.YZ)` entry (next free number)
   or a dated addendum to the entry it amends; bump the working-draft
   version in the header. Include the example, the rule, the edge cases,
   and "Rejected: …" with why. Superseded text is struck through, never
   deleted or renumbered.
2. `veles-checklist.md`: remove the §9 item (its `Qn` label is not reused),
   add a §10 row: date, decision, **result** (user, recommended of
   N — or "over the recommended X"), rejected options. A note it settles
   leaves `notes_to_change.txt` (the full text stays in git history).
3. `veles-plan.md`: the task it unblocks, with acceptance criteria.
4. Then build it (`veles-develop`), unless the user said to wait.

Quote the user's own words when they gave a reason; they are the
rationale future agents need.
