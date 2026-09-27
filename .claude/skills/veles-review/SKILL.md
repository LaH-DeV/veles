---
name: veles-review
description: Review Veles changes or an area of the compiler/std/runtime for real bugs — adversarial probe programs, spec conformance, stress and fuzzing — and report findings by severity. Use when asked to review a diff, audit a feature after it lands, or hunt bugs in an area.
---

# Reviewing Veles

The goal is bugs a user would hit, proven with a program — not style notes.
Past reviews that paid off found overflow on untrusted input, a formatter
eating comments, a GC root lost in a register, and a race-arm deadlock.
Each came from running something, not from reading alone.

## Scope

- A diff: `git diff` (and `git diff --cached`) against HEAD; read the whole
  touched functions, not only the hunks, and every caller of a changed
  signature.
- An area: the spec entries for it (`veles-spec.md`), its checklist
  section, its example and doc chapter, then the code.

## Recipe

1. **Spec conformance.** For each rule the `D` entry states, is there a
   check and a test? A rule with no test is a finding when the code is
   wrong, and a test gap otherwise.
2. **Probe programs** in the scratchpad, one per feature edge: empty,
   one, max/min integer, i64 overflow in intermediate math, invalid UTF-8,
   huge inputs from "outside" (parsers, network, dates), nested nullables,
   generics instantiated with a struct and with a trait object, suspension
   inside the construct, panics and cancellation mid-way, `with` cleanups
   on every exit path.
3. **Both builds**: default (checked) and `--release`.
4. **Stress**: `VELES_THREADS=1/2/4/8` repeated, `VELES_GC_THRESHOLD=256`,
   `VELES_GC_POISON=1` (see `veles-test`).
5. **Fuzz** the touched front-end package for a few minutes
   (`FuzzParse`, `FuzzFormat`, `FuzzCheckMutated`).
6. **Round trips**: `veles fmt --check` on new code; format → parse gives
   the same AST; `parse(x.toString()) == x` for std types that claim it.
7. **Diagnostics**: does the error say what to do? Does a fix exist where
   the edit is mechanical, and does applying it produce code that checks?
8. **Tooling**: hover/completion/go-to-definition on the new syntax.

## What counts

- **Bug**: wrong output, a crash, a panic reachable from valid input,
  unbounded resource use from outside input, a hang, a race, a miscompile,
  a spec violation. Always with the reproducing program and its output.
- **Footgun**: valid code that silently does the surprising thing. The
  project's rule is to refuse it at compile time; propose how.
- **Test gap**: behaviour that could regress unseen.
- Not findings: taste in names or syntax (that is `veles-decide`), gofmt
  noise on CRLF files, anything you could not reproduce — say "suspected"
  or drop it.

## Report

Most severe first. Each finding: one-line claim, the reproducer (file or
inline), actual vs expected, the cause if found, and the fix you propose.
If asked to fix: one regression test per bug, fail-before/pass-after, then
the fix (`veles-develop`). Record fixed bugs in the plan's progress log;
design questions a review raised go to the checklist §9, not into code.
