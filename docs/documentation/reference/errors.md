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

### `'?:' is for a nullable; a Result's value-or-fallback is '??'` (and the reverse)

`?:` unwraps a `T?`, `??` unwraps a `Result` — one idea, split by what is
on the left, so the operator tells the reader which kind of "maybe" it is.
The quick fix (and `veles check --fix`) swaps the operator. (D61)

### `the 'else' of a 'val ... else' must leave`

`val x = r else { ... }` binds `x` only when `r` has a value; on the other
path `x` does not exist, so the block cannot fall through to code that
uses it. End it with `return`, `break`, `continue`, `throw` or `panic`. To
carry on with a default instead, use `r ?? fallback` (a Result) or
`n ?: fallback` (a nullable). (D61)

### `nothing here can fail: 'val ... else' needs a nullable, a Result or a pattern`

The value always exists, so the `else` could never run. Drop it. (D61)

### `'f' is declared 'throws' but nothing in its body can throw`

A warning: no `throw`, and no `try` of something that can fail, anywhere
in the body — so the clause only makes every caller write a `try` that
can never propagate anything. Remove it; `veles check --fix` does, and on
the next pass also removes the callers' now-pointless `try` (which is the
error `'try' needs a Result`, carrying its own fix). A `public` function,
a trait method and its implementations, and a function passed as a value
are left alone: there the clause is a contract, not a claim about one
body. (D45)

### `'Dog' is not an error: declare it with 'error Dog { ... }' instead of 'struct'`

Only types declared with `error` (or given an explicit `implement Error for`)
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

### `unexpected character U+00A0, a no-break space; use an ordinary space`

Names may use letters from any script — `café`, `日本語`, `µs` — and
combining marks and digits after the first letter (Unicode's identifier
rules, UAX #31). A character that is none of those is reported by name,
because it is usually invisible: a no-break space pasted from a web page,
a zero-width space, a byte-order mark in the middle of a file. A
*bidirectional control* (U+202A–U+202E, U+2066–U+2069) is refused in code
and in comments: it makes a file display in a different order from how it
compiles (the "Trojan source" attack). Inside a string it is a warning —
write it as `\u{202E}` so it can be seen. (D18)

### `the receiver is spelled 'this'`

Before v0.40 a method named its receiver `self`. The old word still reads
as the receiver, so the rest of the file checks, but every use is an error
whose fix writes `this` — `veles check --fix` (or the editor's quick fix)
migrates a whole package in one run, `"$self"` and `"${self.x}"` included.
`Self`, the type, is unchanged. (D65)

### `'f' is a copy of the caller's 'Fuzzer': this function calls 'range', which changes 'f.rng', and the caller never sees the change`

A warning. A struct is a value (D7), so a parameter is a copy of what the
caller passed; changing its `var` fields changes the copy. Take a pointer
— `f: *Fuzzer`, called with `&fuzzer` — to change the caller's value, or
return the changed one. Fields that are references (a `MutableList`, a
`*State`) are shared, so pushing into them is seen and is not reported.

### `interpolating 'this' inside its own 'toString' calls the same 'toString' again, forever`

Interpolation prints a value through its `Display` implementation, so
`"$this"` inside that very `toString` is the function calling itself with
the same argument — a stack overflow at run time. Interpolate the fields
(`"(${this.x}, ${this.y})"`). Printing a *different* value of the type is
allowed, since that recursion can end. (P7)

### `'when' is not exhaustive: Rect`

A `when` over a sealed type must cover every variant or end with `else`
(D13). Prefer listing the variant so the next one you add is caught too.

### `cannot assign to 'x': it is a 'val'; declare it with 'var'`

A binding declared `val` cannot be rebound (D11). It says nothing about
the value's insides: a `val` struct's `var` fields and a `val`
collection's contents can still change.

### `cannot assign to 'T.f': the field is immutable; declare it 'var f: ...'`

A bare field is set by the constructor call and never assigned again,
whoever holds the struct — through a binding, a pointer or `this`. Mark
the field `var` if it is meant to change, `protected var` if only the
type's own code should change it, or build a new value (D22). A
method call on a global `val` is refused the same way when the method
changes its receiver: a global is shared by every task (D35).

### `cannot assign to 'T.f' here: the field is 'protected var'`

A `protected var` field is assigned only inside the type's own
declarations — its methods, `implement` and `extend` blocks; everyone else
reads it. Call a method of the type, or make the field a plain `var` if
outside writes are intended (D22).

### `'protected' qualifies 'var'`

`protected` restricts *who may assign* a field, so it goes with `var`:
`protected var count: i64`. A bare field is never assigned by anyone, and
so has nothing to protect (D22).

### `cannot push into an immutable List; use MutableList`

`List` is immutable. Use a `MutableList` (`mut [...]` or the
`MutableList<T>` annotation), or build a new list with `map`/`filter`.
(D25)

### `argument of type 'MutableList<i64>' is not Sendable and cannot cross a task boundary`

Tasks may only share immutable data (D35). Pass a `List` (`.toList()`),
send items over a `Channel`, or guard shared state with `Mutex(value: ...)`.

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

Mark it `public`, or if it is meant to stay internal, add a `public`
function in that module that does what you need (M5).

### `module 'x' of package 'lib' is not in its exports`

The dependency's `veles.toml` decides what leaves the package. Import an
exported module, or add `x` to that package's `exports` (M5).

### `positional argument after a named argument`

Once you name one argument, name the rest: `f(1, b: 2, c: 3)` is fine,
`f(a: 1, 2)` is not (D28).

### `a byte literal holds one ASCII character`

`'x'` is a `u8` — the byte of one ASCII character, for code that walks a
string with `byteAt` (`b == '"'`, `b >= '0' && b <= '9'`). A character
outside ASCII has no single byte to be; use a one-character string, `"é"`
(D18).

### `panic: index 3 out of bounds for list of length 1`

A runtime panic, not a compile error: `xs.set(i, v)` or `xs.swap(i, j)`
with a bad index. A read is `xs.at(i)`, a `T?` — or a `T` where the
compiler can see the index is in range (D62); where it cannot, say why
the read cannot fail: `xs.at(i) ?: panic("…")`.
The line under the message, `at main.vs:7:31`, is where it happened (D64).
Panics end the current task; a `gather` reports them as `Err(Panic)`,
a `scope` re-raises them (D52).

### `panic: integer overflow`

Checked arithmetic (D21). Use a wider type, or the wrapping operators
`+% -% *%` if wrapping is intended. `--release` builds drop the check.
