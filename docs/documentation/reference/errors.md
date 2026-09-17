# Common error messages

The compiler's messages try to say what to do, not only what is wrong.
This page collects the ones newcomers meet most, with the reasoning
behind each. The `D<n>` / `M<n>` tags point into `veles-spec.md`.

### `cannot infer the element type of an empty list; annotate it`

`[]` and `[:]` have no elements to infer from. Write the type on the
binding — `var xs: MutableList<i64> = []` — or pass the literal
where the expected type is known (a parameter, a field, a return). (D25)

### `type mismatch: expected 'i32', found 'i64'`

There are no implicit numeric conversions (D21). Convert with `as`:
`n as i32`. Integer literals, lengths, indices and `toInt()` are all
`i64`, so the mismatch usually means a signature was written with `i32`
for no particular reason — use `i64` unless the width matters.

### `unused Result: the call may fail; use 'try' to propagate the error or 'when' to handle it`

You called a `throws` function as a statement, or bound its result to a
name (`unused Result 'r'`) and never read it. Decide: `try f()` if your
function is `throws`, or `when (val r = f()) { is Ok => ...; is Err => ... }`.
Ignoring a failure is not an option; `val _ = f()` is the deliberate
discard. (D4)

### `'x' is never used`

A warning: the `val`/`var`, loop, destructuring or pattern binding is never
read (assigning to it does not count). Use it, drop it, or name it `_`.

### `'try' propagates an error, but the enclosing function is not declared 'throws'`

Add `throws` to the function (the error type is inferred), or handle the
`Result` with `when`.

### `'Dog' is not an error: declare it with 'error Dog { ... }' instead of 'struct'`

Only types declared with `error` (or given an explicit `impl Error for`)
can be thrown or named in `throws`. Change `struct` to `error`, or wrap a
non-struct value: `error Failed { why: string }`. A generic `throws X`
needs `X: Error`.

### `only 'message' can be overridden in an error`

Inside `error Name { }`, `fun message(): string = ...` replaces the
default text with no `override` needed; any other method is an ordinary
method and cannot be marked `override`.

### `'PortErrors' names an error set; it can only appear after 'throws' ...`

`error PortErrors = A | B` is a name for a union, and unions exist only in
error position (D45): after `throws`, inside another error set, or as the
type of a field of an `error` (a cause). It is not a value type — use
`when` on the members, or one of the members itself.

### `error set 'S' refers to itself`

Two error sets include each other. Sets flatten, so a cycle adds nothing;
remove one direction.

### `'throw' fails the function, but it is not declared 'throws'`

Same fix: add `throws`, or return a `T?` if "nothing" is the right
answer rather than "failure".

### `value of type 'string?' may be null; use '?.', '?:' or check for null first`

A nullable value is not a value (D5). `x ?: fallback`, `x?.member`, or
`if (x == null) return ...` and continue with `x` narrowed.

### `'when' is not exhaustive: Rect`

A `when` over a sealed type must cover every variant or end with `else`
(D13). Prefer listing the variant so the next one you add is caught too.

### `cannot assign to 'x': it is a 'val'; declare it with 'var'`

Also reported for `a.bump()` when `bump` is a `mut fun` and `a` is a
`val`. Only `var` bindings can be mutated or have `mut fun` methods
called on them (D11/D22).

### `cannot push into an immutable List; use MutableList`

`List` is immutable. Use a `MutableList` (`mut [...]` or the
`MutableList<T>` annotation), or build a new list with `map`/`filter`.
(D25)

### `argument of type 'MutableList<i64>' is not Sendable and cannot cross a task boundary`

Tasks may only share immutable data (D35). Pass a `List` (`.toList()`),
send items over a `Channel`, or guard shared state with `mutex(...)`.

### `'await' applies to channels, timers and task handles` / `'recv()' always suspends and must be awaited`

Suspension is inferred (D2/D16). Call ordinary functions normally, even
those that suspend; write `await` only on `sleep(...)`, `ch.recv()` and
task handles from `async`.

### `calling extern "C" function 'strlen' requires an 'unsafe' block`

Wrap the call: `unsafe { strlen(p) }`. The block marks the places where
the compiler cannot vouch for memory safety (D44).

### `import cycle: module 'a' is already being loaded`

Modules must form a DAG (M4). Move the shared code into a third module
both can import, or merge the two.

### `'helper' is private to module 'geometry'`

Mark it `pub`, or if it is meant to stay internal, add a `pub`
function in that module that does what you need (M5).

### `module 'x' of package 'lib' is not in its exports`

The dependency's `veles.toml` decides what leaves the package. Import an
exported module, or add `x` to that package's `exports` (M5).

### `positional argument after a named argument`

Once you name one argument, name the rest: `f(1, b: 2, c: 3)` is fine,
`f(a: 1, 2)` is not (D28).

### `character literals are not part of the language`

Use a one-character string, `"x"`. Strings are byte-indexed UTF-8 and
there is no separate char type in the bootstrap (D18).

### `panic: index 3 out of bounds for list of length 1`

A runtime panic, not a compile error: `xs[i]` with a bad index. Use
`xs.at(i)`, which returns `T?`, when the index is not known to be valid.
Panics end the current task; a `gather` reports them as `Err(Panic)`,
a `scope` re-raises them (D52).

### `panic: integer overflow`

Checked arithmetic (D21). Use a wider type, or the wrapping operators
`+% -% *%` if wrapping is intended. `--release` builds drop the check.
