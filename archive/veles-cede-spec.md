# Cede: explicit cooperative scheduling

Research and specification, 2026-10-21. **D197: built 2026-10-27** (plan
F20; see "Implementation status" at the end). The user approved a reserved,
statement-only keyword and a scheduling opportunity rather than an ordering
barrier. `yieldNow()` is gone; its uses are errors with fixes.

Sections 1-7 are normative under [D197](veles-spec.md); the research above
them and implementation guidance below them are explanatory. Earlier
decisions continue to govern anything D197 does not change.

## Scope

Replace the prelude scheduler-yield function `yieldNow()` with an explicit
control-flow keyword. Do not implement generators, change ordinary-call
suspension spelling, add `noSuspend`, or redesign scheduling in this change.

The user's rationale: keywords are cleaner when expressing "tell the
compiler something"; `cede` should be uncommon, "just a possibility".

## Verified implementation baseline

- [The current helper](std/prelude/concurrent.vs) calls
  `await sleep(Duration.zero)`.
- [Suspension checking](sema/check_task.go) lowers that operation to the
  `task.sleep` builtin and propagates the suspension effect.
- [Coroutine lowering](codegen/llvm/coro.go) calls `veles_task_sleep`, parks
  the current frame, and checks cancellation and fail-fast scopes when
  execution resumes. Cleanup waits are shielded from cancellation.
- [The runtime](runtime/c/veles_task.c), `veles_task_sleep_impl`, enqueues a
  zero-duration sleeper once using `TB_YIELDED`, then completes the operation
  on its next entry. It does not arm a new sleep timer for that operation.
- [Lock-region checking](sema/lockguard.go) rejects suspending operations and
  named calls whose inferred effects suspend. This is distinct from cleanup
  shielding: a suspending `close()` can wait under D147.

The old helper's comment promises every other runnable task a turn. That
must not become a portable language guarantee: executor-local queues,
work stealing, concurrent wakes and operating-system scheduling do not
establish a global task order. A yield is not synchronization.

## Primary-source comparison

Sources consulted on 2026-10-21; summaries describe the documented contract,
not experiments performed in those runtimes.

| Runtime                                                                                                                  | Documented operation                                                                                          | Design lesson                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| [Kotlin coroutines](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines/yield.html) | `yield()` offers other coroutines execution; cancellation is checked even when no actual suspension is needed | Check cancellation even on a possible future fast path; do not poll for readiness with repeated yields                  |
| [Tokio](https://docs.rs/tokio/latest/tokio/task/fn.yield_now.html)                                                       | Awaited `yield_now()` requeues work; immediate repoll and other scheduling orders remain possible             | Specify non-guarantees; do not import assumptions about combinator polling into Veles                                   |
| [Go](https://pkg.go.dev/runtime#Gosched)                                                                                 | `runtime.Gosched()` yields execution and resumes automatically                                                | Distinguish scheduler yielding from waiting for an external event; Go's terminology is not Veles' inferred-effect model |
| [Python 3.13 asyncio](https://raw.githubusercontent.com/python/cpython/3.13/Doc/library/asyncio-task.rst)                | `sleep(0)` always suspends the task, with an optimized zero-delay path                                        | A scheduler boundary need not arm a timer; a dedicated spelling makes intent clearer than a magic duration              |
| [Rust OS threads](https://doc.rust-lang.org/std/thread/fn.yield_now.html)                                                | `std::thread::yield_now()` yields to the OS; repeated yielding can busy-wait                                  | Do not confuse a task scheduler boundary with an OS-thread yield; prefer event-based waits                              |

These are not identical contracts. Veles adopts an explicit task scheduling
boundary with inferred suspension and existing structured cancellation;
it does not adopt Go's goroutine type model, Python's explicit async
declarations, Tokio's poll-combinator behavior, or Rust's OS-thread primitive.
Swift's documentation page could not be extracted in this session; no
Swift-specific guarantee is used as evidence.

## Alternatives and rationale

The motivating case is bounded CPU work sharing an executor with other
tasks. Existing source uses `yieldNow()` between chunks. All snippets with
`cede` below describe the target language, not currently compiling programs.

| Option                     | Chunk boundary                           | Benefit                                                   | Cost or reason rejected                                                            |
| -------------------------- | ---------------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Keep the helper            | `yieldNow()`                             | No grammar change; first-class function already available | Hides control flow behind a prelude name; the user requested its deletion          |
| Reuse sleep                | `await sleep(Duration.zero)`             | Works already; preserves a low-level equivalent           | Duration spelling obscures scheduling intent                                       |
| Scheduler `yield`          | `yield`                                  | Familiar coroutine vocabulary                             | Couples scheduling vocabulary to the separate generator investigation              |
| Expression keyword         | `cede` in arbitrary expression positions | Concise expression bodies and Unit arguments              | Creates unnecessary value-position rules for an explicit control-flow boundary     |
| Contextual keyword         | statement-position `cede`                | Preserves existing identifiers                            | Ambiguous calls/references and less direct wrong-form diagnostics                  |
| Reserved statement, chosen | `cede`                                   | Explicit, unambiguous; separate from generator output     | A new keyword; all frontend/tooling layers and existing identifiers need migration |

High level: ordinary I/O, channels, structured concurrency and existing
automatic yielding remain the normal tools. Low level: `cede` offers an
explicit boundary in a CPU-work helper without spawning a task or pretending
to sleep. Self-hosting: the same statement must be represented and printed
by both frontends; compiler code uses it only where a suspending caller is
appropriate, never to bypass a plain callback or held-lock restriction.

## 1. Lexical and grammar rules

`cede` is a case-sensitive reserved keyword, not an identifier, prelude
declaration, callable value, modifier, or generator operation. The normal
keyword rules apply to declaration names, parameters, member names, import
names and labels. Identifiers such as `ceded` remain identifiers. Existing
identifiers exactly named `cede` must be renamed, including their references;
no new escaped-identifier mechanism is introduced.

```text
cede-statement := 'cede'
```

It is legal anywhere an ordinary executable statement is legal and the
suspension effect is permitted, not only in a loop. Normal braced bodies and
single-statement bodies apply. It takes no arguments, value, label, executor
or duration. Explicit semicolons are optional under the existing statement
termination rules. `cede` itself can terminate a statement at newline,
closing brace or EOF; EOF does not legalize an otherwise unclosed block.

Automatic semicolon insertion must recognize the keyword in both lexers.
A newline after `cede` cannot be suppressed to continue a member/operator
chain: the keyword has no operand to continue. Comments preserve the normal
newline rules. On one line, an unexpected operand must be diagnosed rather
than interpreted as part of the statement or silently discarded.

It is not an expression and has no value or type of its own. A surrounding
block or function retains its ordinary Unit/result rules. A Unit-returning
block lambda can contain it; a one-expression function body cannot consist
of the statement alone.

```veles
fun schedulerBoundary() {
  cede
}

fun main() {
  schedulerBoundary()
  cede;
  io.println("resumed")
}
```

This target program prints `resumed` once. It makes no assertion about
which other task runs first. Representative target forms:

| Form                                                 | Result                                            |
| ---------------------------------------------------- | ------------------------------------------------- |
| `if (needBoundary) cede`                             | Legal in a suspension-capable context             |
| `loop (needWork) cede`                               | Legal; condition must have its normal bool type   |
| `() => { cede }`                                     | Legal Unit-returning suspending lambda            |
| `async { cede }`                                     | Legal where an async task block is normally legal |
| `cede()` / `cede(1)` / `cede value` / `cede@label`   | Error: no arguments, operand or label             |
| `await cede` / `try cede` / `async cede`             | Error: statement, not an expression               |
| `val pause = cede` / `return cede` / `consume(cede)` | Error: statement has no value                     |
| `fun pause() => cede`                                | Error: use `fun pause() { cede }`                 |
| `cede.member` / `cede + 1`                           | Error: not a value; no chaining or arithmetic     |

An unbraced statement body containing a single `cede` is not the same as
an expression body introduced by `=>`. D197 does not change the rules for
`when` arms, race/select arms, or value-producing blocks: use a block where
those constructs allow one, and meet its normal result and effect contract.
`cede` is not a selectable readiness operation.

## 2. Effect and context rules

Executing `cede` is a direct suspension-effect seed, just like the current
helper's body. It propagates through ordinary calls, recursive call graphs,
function values and conditional higher-order inference (D116). Callers do
not gain an `async` declaration modifier or an `await` requirement. The
effect cannot disappear merely because a runtime currently has one task.

- A normal function or lambda can infer suspension from `cede`.
- Assignment/passing to a non-suspending function type is refused; a
  suspension-permitting callback accepts it under the existing type rules.
- A callback containing a transitive call to a ceding helper is equally
  suspending. Generic/conditional code gets its existing plain and suspending
  instances, not a new special case that turns every instance into a coroutine.
- Held-lock regions refuse it, directly or transitively, under D107.
  Lexically placing it in a helper or a lambda does not bypass this rule.
- Constant evaluation, immediate module/global initialization, native or
  plain-thread callback contracts, and other contexts that forbid suspension
  refuse it under their existing rules. A `runBlocking` bridge remains
  governed by D152; D197 does not create an implicit bridge.
- A lambda declaration that is not invoked during initialization is not
  itself execution of `cede`; ordinary function-value rules still apply.
- A suspending `close()` may contain it under D147, unless another enclosing
  rule forbids suspension (for example, a held lock). D47 cleanup shielding
  still applies. `cede` does not make cleanup cancellable.

There is no new `noSuspend` construct, no change to trait method effect
contracts, and no new exception to Sendable checking.

## 3. Scheduling semantics

On reaching `cede`, execution of the current task offers a scheduling
opportunity to its current executor. Unless cancellation or fail-fast
unwinding wins, the continuation becomes runnable on that same executor,
and normal execution resumes at the following statement when selected.

The operation must relinquish the currently executing coroutine chain to
the executor's scheduling machinery, without waiting for a timer, an I/O
event, another task's completion or an external wake. It must not merely
poll a cancellation flag and continue inline while bypassing dispatch.
The executor may select this task again immediately, even when other work
exists. With no other work, the task must remain eligible to resume: a
single `cede` cannot itself leave it stranded waiting for a wakeup.

Implementations may optimize the scheduler boundary only while preserving
these semantics and cancellation/fail-fast observation; they may not erase,
hoist or merge it as an effect-free Unit computation. No new task, child
scope, timer registration or timed delay is created by the operation.
Coroutine-frame setup or scheduler bookkeeping is not claimed allocation-free.

There is deliberately no guarantee of:

- FIFO order, round-robin order, a different next task, or a turn for every
  currently runnable task;
- a particular number of tasks running, timer/reactor polling between two
  statements, or all CPU cores being used;
- elapsed delay, bounded scheduling latency, starvation freedom against
  arbitrary other tasks, or an OS-thread context switch;
- another task observing writes, a memory fence, atomicity, a happens-before
  edge, or any other synchronization beyond existing language rules.

`cede` does not change the selected executor, priority, task identity,
task-local state, parent scope, deadline, or cancellation target. A pool may
resume the continuation on a different worker, as for other suspension
points. `Executor.thread` and declared single-thread executor affinity are
preserved. One-worker pool configuration is not OS-thread affinity.
Normal non-Sendable confinement rules are unchanged.

Repeated `cede` in a readiness-polling loop can still waste CPU. Use a
channel receive, task join, event or an appropriate I/O wait for readiness.
Do not use ceding to make an unsynchronized program or a timing test correct.
For timed delay use `sleep`; for cancellation checking without deliberately
ceding use `checkCancelled()`; for a blocking foreign API use the existing
blocking/bridge mechanisms. Existing automatic yielding of suspending loops
and back-edge cancellation are unchanged. A long unchunked synchronous call
does not become preemptible just because a `cede` follows it.

## 4. Cancellation, failure and cleanup

`cede` is a cancellation and enclosing fail-fast observation point under
the ordinary suspension rules (D34, D35, D47, D145, D151). A cancellation
already pending when the operation is reached must be observed no later than
its completion, before the following user statement executes, unless
delivery is shielded. It may begin unwinding without first requeueing.
Cancellation arriving while the continuation is queued is checked on resume.
A request racing after the final check is delivered at the next applicable
point; D197 does not promise an impossible atomic boundary with future requests.

Unwinding closes `with` resources innermost first and cancels/joins abandoned
children according to the existing structured-lifetime rules. A failed child
in an enclosing fail-fast scope is observed before continuing past the
boundary, under the existing failure/cancellation precedence. Cede does not
return a Result, throw a new `Cancelled` error, swallow a child failure, or
decide Q45's operation-token completion contract.

During shielded cleanup, ceding still offers scheduling but must not deliver
task cancellation or abort the close because of an enclosing fail-fast
request. Pending requests remain pending for the ordinary post-cleanup
delivery path. A close that loops forever ceding can delay cancellation
forever; this is the existing non-cancellable cleanup contract, not a new
timeout guarantee. Nested calls made by cleanup preserve shielding across
suspension, and must never resume as an unshielded ordinary call by mistake.

## 5. Deletion and migration of yieldNow

The implementing change removes the public prelude declaration, its export,
completion/signature entries and teaching recommendations. There is no
callable compatibility alias, deprecation-only interval, or prelude value
named `cede`. The compiler still recognizes removed standard forms to issue
an actionable removal diagnostic, following the project's removed-syntax
policy. Recognizing an old form must never make it executable successfully.

The diagnostic must use normal name resolution and shadowing precedence:
a user's independent local function, method, imported declaration or variable
named `yieldNow` is not the removed prelude symbol. After the standard
declaration disappears, a removal fallback applies only to uses that would
otherwise resolve to that removed standard name, not every matching token.
Old qualified/imported forms need diagnostics only where those forms could
actually denote the old declaration; do not invent previously legal syntax.

| Old use of the standard helper                                     | Migration                                                                                                            |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| Standalone `yieldNow()`                                            | Replace the whole call with `cede`                                                                                   |
| `if (ready) yieldNow()`                                            | Replace the statement call, retaining the unbraced body                                                              |
| `fun pause() => yieldNow()`                                        | Rewrite the function body as `{ cede }`, retaining its signature                                                     |
| Value/reference `yieldNow`                                         | Use a Unit-returning suspending wrapper, e.g. `() => { cede }`, where the expected type permits suspension           |
| Call used inside a value expression                                | Restructure the enclosing expression/block so ceding remains at the same evaluation point; no blind text replacement |
| Old helper with arguments, type arguments or invalid qualification | Diagnose the invalid/removed form; never discard argument evaluation through a fix                                   |

An old `async yieldNow()` becomes `async { cede }`, preserving the existing
task-handle and await obligations. Never replace just its inner call with a
statement in an expression position.

Automatic edits are offered only when the enclosing AST and effect contract
prove the replacement sound. Preserve comments, source order and evaluation
order. If a safe edit is not available, explain the required wrapper or body
rewrite without applying a guessed fix. Renaming an existing identifier
`cede` likewise requires resolution-aware changes, not global substitution.

`await sleep(Duration.zero)` remains valid and retains the same scheduling
and cancellation contract; there is no new lint forbidding it. Existing
negative-duration sleep policy is unchanged. `checkCancelled()` stays a
prelude function. `yield` remains reserved and generator semantics remain
open under I1; no generator receives or emits a value through `cede`.

## 6. Diagnostics and tooling contract

Diagnostics must name the invalid form, explain the mistake and supply the
new spelling where possible. Exact wording can follow repository diagnostic
conventions, but these distinctions are required:

| Invalid use                       | Required explanation/fix                                                                               |
| --------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `cede()`                          | `cede` is a statement, not a function; remove parentheses                                              |
| `await cede`                      | `cede` already marks the scheduling boundary; remove `await`                                           |
| Operand/label/type arguments      | `cede` takes no operand, label, arguments or type arguments                                            |
| Value or expression-body position | No value; use a block containing the statement, respecting the enclosing result contract               |
| Identifier named `cede`           | Reserved keyword; rename the declaration and references                                                |
| Forbidden suspension context      | Name the callback/lock/constant/plain context that forbids suspension; do not suggest bypassing safety |
| Removed standard `yieldNow`       | Removed in D197; use `cede` for a statement or a suspending block wrapper for a function value         |

Keyword hover: **"Yields execution to the current executor's scheduler.
May resume immediately. Cancellation is checked unless cleanup is shielded."**
Completion inserts `cede`, never `cede()` or `await cede`. The editor grammar
and semantic tokens classify it as a control-flow keyword. Inferred
declaration effects and call hover must reflect ceding helpers. This decision
does not silently add a general call-site inlay-hint feature: existing effect
hints remain, and any future call-site hint should say "may suspend", not
"allows suspension". The keyword itself already exposes the boundary.

The formatter prints `cede` without parentheses or operands, using existing
indentation/semicolon conventions and preserving attached comments. AST
printers, visitors, source spans, source maps and selfhost serialization must
not omit the statement or change its position. The Go and self-hosted
frontends must remain byte-identical at their existing parity gates.

## 7. Required acceptance matrix

The implementation is not complete until every applicable row is pinned by
tests, documented and verified. Test ordering through channels/events/joins
or runtime instrumentation, never by assuming one `cede` gives a sibling a
turn. Test timeouts are harness failure bounds, not language latency promises.

| Area                | Cases and assertions                                                                                                                                                                                                           |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Lexer/parser        | Bare and semicolon forms; adjacent statements; nested blocks; unbraced if/loop; newline/CRLF; line/block comments; EOF; a following chain operator cannot absorb a newline; malformed tokens recover to later statements       |
| Keyword reservation | Declarations, members, parameters, labels/import names follow existing keyword policy; longer identifiers unaffected; both lexers agree                                                                                        |
| Invalid grammar     | Every rejected form in sections 1 and 6, including expression bodies, value arguments and labelled forms; no panic or misleading unknown-name cascade                                                                          |
| Effects             | Direct/indirect helpers, recursion, lambda references, plain vs suspending callbacks, generic conditional instances, trait contracts and forbidden initializer/const/native/Thread contexts; plain instances remain plain      |
| Lock safety         | Direct and transitive ceding under Mutex/RwLock and withLock rejected under their existing held-lock rules; no escape through callbacks                                                                                        |
| Continuation        | Single task completes; repeated ceding completes; deep suspending call chains resume once at the next statement; live locals survive under GC pressure; no duplicate enqueue/resume or lost wake                               |
| Scheduling          | Instrument a real dispatch boundary and same-executor requeue; no new task, timer or external-wake dependency; peers can progress under ordinary scheduler operation without asserting global FIFO                             |
| Cancellation        | Pending before boundary, cancellation while queued, cancellation racing resume, nested deadlines/timeouts; post-boundary marker does not run after delivery; no request or queued-task state lost                              |
| Failure/lifetime    | Failed sibling observed; abandoned children joined; with resources close exactly once innermost first; outer cancellation is not converted into the wrong inner timeout                                                        |
| Shielded cleanup    | Direct and nested helper ceding during suspending close; cancellation/fail-fast cannot abort the close; pending request is delivered after cleanup; no continuation queued twice                                               |
| Executors           | Default/pool and on-executor nested scopes retain executor; fixed-thread id unchanged across many cedes, including cleanup and cancellation; task locals and declared-executor confinement preserved                           |
| Sleep coexistence   | Zero/negative/positive sleeps retain their contracts; cede after completed or cancelled waits cannot consume a stale timer/yield flag; automatic loop yielding unchanged                                                       |
| Removal             | Prelude direct calls/references/body forms get actionable errors; safe fixes compile and preserve comments/order; invalid arguments never silently vanish; custom/shadowing yieldNow remains valid; no callable standard alias |
| Tooling             | Formatter idempotence and comments, AST printer/visitors, selfhost lexer/parse gates, hover/completion/semantic tokens, diagnostics/fix spans and diagnostic-family coverage                                                   |
| Platforms/stress    | Debug/release, VELES_THREADS=1/2/4/8, low GC threshold and GC poisoning where supported; Linux sanitizer; Windows/Linux/macOS backend and affinity gates or explicit pending platform records                                  |

## Implementation guidance (not new language rules)

Use a dedicated AST statement and a distinct semantic operation (for example
`task.cede`) so source intent, effects, diagnostics and debug spans survive
lowering. Reusing the zero-sleep scheduler primitive is acceptable; there is
no requirement for a new exported C ABI. Avoid routing through a prelude
function or manufacturing a Duration. Preserve existing coroutine park,
resume, cancellation, cleanup-depth and fail-fast handling. A timer or yield
flag left over from an abandoned operation must be retired correctly.

Implement in small, validated slices:

1. Both token sets/lexers and semicolon rules; Go AST/parser/printer and the
   selfhost AST/parser serialization; parser/parity and formatter tests.
2. Semantic effect seed, context diagnostics, HIR and lowering; conformance
   and codegen golden tests; ordinary and shielded continuation regressions.
3. Removed-name resolution/fixes and actual prelude deletion; update callers,
   docs and editor/LSP support in the same landing change, not a release later.
4. Stress/platform/performance gates and the final whole-tree audit. Do not
   mark D197 built until these gates and the migration are complete.

Ownership/migration inventory from the research:

- [Tokens](lexer/token.go), [lexing](lexer/lexer.go), [statements](parser/stmt.go),
  [AST](ast/ast.go), [AST printing](ast/print.go), [formatter](format/format.go),
  [selfhost frontend](selfhost), [semantic checking](sema),
  [LLVM lowering](codegen/llvm) and [runtime](runtime/c).
- [Removed forms](sema/removed.go), [lint/diagnostic indexing](sema/lint_index.go),
  [LSP](lsp) and [VS Code support](editors/vscode).
- [Prelude helper](std/prelude/concurrent.vs), [idle benchmark](bench/idle/main.vs)
  and [runtime regressions](driver/runtime_test.go).
- [Concurrency chapter](docs/documentation/12-concurrency.md),
  [networking chapter](docs/documentation/16-networking.md),
  [concurrency explanation](docs/documentation/concurrency-explained.md),
  [concurrency reference](docs/documentation/reference/concurrency.md),
  [stdlib reference](docs/documentation/reference/stdlib.md), plus the
  cheatsheet and new diagnostic explanations when implementation lands.

Audit all active source, docs, examples, benchmarks, generated API entries
and executable snippets for the old standard call, including any that appear
after this inventory. Regenerate outputs/goldens with their repository tools.
Archive and dated implementation history remain history; do not mechanically
rewrite them. Negative migration tests and historical references may retain
`yieldNow`; the audit must classify these rather than demand zero matches.

Before and after lowering/runtime changes, use `veles-bench`: compare repeated
cede against the old helper/zero sleep, and check `suscall`, `spawn`, `channels`,
`idle` and `httphello` at 1 and 8 workers where supported. Separate cold/frame
setup from steady-state ceding; record throughput, allocation and peer/timer
latency, not just mean yield cost. No numerical improvement or latency bound
is claimed by this specification. Existing build/vet/test, docs, selfhost,
sanitizer and platform gates still apply at implementation time.

## Verification status and next phase

Source inspection supports the baseline and the five primary-source
comparisons. With Go 1.23.2, the existing driver tests
`TestLoopsAreCancellationPoints`, `TestPutOffCancelEndsTheNextWait` and
`TestWithTimeoutInCallersTask` passed on macOS on 2026-10-21. They check
existing cancellation behavior, not unimplemented `cede` syntax or all of
the acceptance matrix. `TestAtomicOperations` also passed, exercising current
`yieldNow()` callers. Document checks cover local links, code fences, unique
D197/version and F20 records, decision-table placement and pending status.
No full build/vet/test run, cross-platform run or performance result for
`cede` exists from this documentation-only task. No claim of absence of all
specification or implementation bugs is made.

After this specification, return to [generator research](veles-research-generators.md)
and I1 planning: compare ownership of iterator state, early-exit cleanup,
errors, suspension, borrowing/retention and implementation costs across
languages before choosing generator syntax or lowering. D197 frees the
scheduler vocabulary from that choice; it does not settle it.

## Implementation status (2026-10-27)

Built in one change with the migration (plan F20). Sections 1-7 are met
as follows; what was not run is listed, not claimed.

- Lexer/parser, keyword reservation, invalid grammar, tooling: both front
  ends, gated byte for byte by the self-host corpus file `p-cede` (every
  form of section 1 and 6, CRLF, comments, EOF, a chain after the keyword,
  `ceded`); `parser/cede_test.go` (trees, one error per mistake, the next
  statement still parsed, every fix applied and re-parsed); formatter;
  `veles parse`; hover (section 6 text), completion, the VS Code grammar.
- Effects and lock safety: `sema/cede_test.go` and conformance
  `D197-cede.vs` (helpers, recursion, D116 instances that stay plain, trait
  contracts, D147 closes; refused under `withLock`, a lock guard directly
  and through a helper, `RwLock`, a global initializer, `const fun`,
  `lazy val`, `Thread.start`, an unannotated lambda as D40 requires).
- Removal: `sema/cede_test.go` and `D197-yieldnow-removed.vs` (each fix
  applied and re-checked; no fix for a call in a value, with arguments, or
  a value where the wanted type does not suspend; a program's own
  function, local or parameter named `yieldNow` untouched).
- Continuation, scheduling, cancellation, failure, shielded cleanup,
  executors, sleep coexistence: `driver/cede_test.go`, debug and release,
  `VELES_THREADS=1/2/4/8`, `VELES_GC_THRESHOLD=4096`, `VELES_GC_POISON=1`.
  A peer is shown to run by a flag only it sets, waited for by recursion
  (no loop back edge can stand in for the boundary); affinity by the OS
  thread id; cancellation by deadlines that pass inside a lock region, so
  the `cede` after it is the first point that can see them. Running that
  case twice found a runtime bug: a yield abandoned by a cancellation left
  `TB_YIELDED` set, and the next `cede` returned at once without a turn or
  a cancellation check (fixed in `veles_task_leave_waits_impl` and
  `veles_task_cancelled`).
- Codegen: golden `codegen/llvm/golden/cede`.
- Platforms: Windows and Linux (WSL). Not run: macOS, and the Linux
  sanitizer build (checklist section 11).
- Performance: `bench/results.md`, 2026-10-27 (yield cost before and
  after, regression benchmarks, and plan F21).
