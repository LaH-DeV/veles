---
name: veles-debug
description: Diagnose a Veles compiler or runtime failure — internal compiler errors, wrong diagnostics, miscompiles, crashes, hangs, GC corruption, and failures that appear only under threads. Use when a Veles program or test misbehaves and the cause is not yet known.
---

# Debugging Veles

Find the cause before changing code. A fix without a reproducer that fails
before and passes after is a guess.

## 1. Reproduce small

Copy the failing program into the scratchpad (never `%TEMP%` — sibling
`.vs` files become modules) and cut it down until every remaining line is
needed. The minimal program usually becomes the regression test.

## 2. Localise the layer

Walk the pipeline and stop at the first stage whose output is wrong:

| Stage | Look with |
|---|---|
| tokens | `veles tokens file.vs` |
| syntax | `veles parse file.vs` (dumps the AST) |
| checking | `veles check dir` — diagnostics; `VELES_DEBUG_DERIVE=1` or `veles explain dir --derive [Type]` for synthesized implements |
| IR | `veles build dir --emit-llvm` (or `--keep` for every intermediate) |
| optimisation | same program with and without `--release`; a difference points at UB in emitted IR or the runtime |
| runtime | the C sources under `runtime/c/`; debug with `gdb` (`C:\msys64\ucrt64\bin`) |

An internal compiler error prints the phase and position and exits 3;
`VELES_DEBUG=1` re-panics with the Go stack.

## 3. Runtime knobs

| Variable | Effect |
|---|---|
| `VELES_THREADS=n` | worker count; `1` separates logic bugs from races |
| `VELES_GC_THRESHOLD=bytes` | collect early and often — shakes out missing roots |
| `VELES_GC_POISON=1` | fill swept objects with 0xCD — a use-after-sweep crashes deterministically |
| `VELES_GC_OFF=1` | if the bug disappears, it is GC-related (roots, descriptors, write ordering) |
| `VELES_GC_TRACE=1` | report each collection |

## 4. Known shapes of past bugs

- **Missing GC root**: an object held only in a register or in a C frame
  across a collection (the setjmp-in-helper bug; see `veles_tls.h`).
  Symptom: rare crash or garbage under `VELES_GC_THRESHOLD` + threads.
  Check `codegen/llvm/gcdesc.go` descriptors for new heap shapes too.
- **C ABI**: a small integer/bool passed to C without `zeroext`/`signext`
  has undefined upper bits — works at -O0, breaks at -O2.
- **Races / lost wakeups**: a state read before registering as a waiter
  without re-checking after; a wake delivered to a task that already
  finished. Lock order is runtime → channel.
- **Schedule-order assumptions**: output that depends on which task ran
  first. The program is wrong, not the runtime — unless the spec promises
  the order.
- **Formatter**: comments flushed to the wrong owner span (`flushComments`
  bounds).
- **Parser**: an arm/lambda ambiguity (`noLambda`, `parseArmHead`) or a
  line-continuation rule.

## 5. Hangs

Run under a timeout. With threads, attach `gdb -p <pid>` and
`thread apply all bt`. The deadlock detector in `veles_task.c` panics with
"deadlock: every task is blocked"; if it fires wrongly, check which queues
and runnext slots it looked at.

## 6. Close the loop

Write the regression test (`veles-test`), confirm it fails without the fix,
then fix the cause rather than the symptom. Record a bug found this way in
the plan's progress log — one sentence on the cause, one on how it was found.
