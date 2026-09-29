# Error messages

The compiler's messages try to say what to do, not only what is wrong.
Every one of them belongs to a **family**, and after the errors the
compiler names the families that came up:

```text
main.vs:5:15: error: 'helper' is private to module 'geo'; declare it 'public' there to use it from here (M5)
    val _ = geo.helper()
                ^^^^^^
see: veles explain private-to-module
```

`veles explain private-to-module` prints that family's section of this
page, from the copy the compiler carries — offline, and for the compiler
you have. An editor shows the same family as a link. The `D<n>` / `M<n>`
tags point into `veles-spec.md`, where each rule is argued.

## At compile time

### syntax

**The text does not parse.** `expected ')', found newline`, `unexpected
'}'`, `unterminated string literal`, `unknown escape sequence '\q'`.

The message names what the parser wanted at that point and what it found
instead. When the place it points at looks right, the mistake is usually
just before it: an unclosed `(`, `[` or `"` on an earlier line, or two
statements on one line without a newline between them. Statements end at
the end of a line; a line that ends in an operator, a `,` or an open
bracket continues on the next one.

### removed-syntax

**A spelling that Veles used to have, or that another language has.**
`'impl' is spelled 'implement'`, `there is no 'for'; every loop is spelled
'loop'`, `'xs[i]' is not indexing; brackets are for collection literals
only — use 'xs.at(i)'`, `'=' assigns; use '==' to compare`.

The old form still parses, so the rest of the file checks, and the error
carries the fix: the editor's quick fix, or `veles check --fix` for a
whole package, rewrites it.

#### `the receiver is spelled 'this'`

Before v0.40 a method named its receiver `self`. The old word still reads
as the receiver, so the rest of the file checks, but every use is an error
whose fix writes `this` — `veles check --fix` (or the editor's quick fix)
migrates a whole package in one run, `"$self"` and `"${self.x}"` included.
`Self`, the type, is unchanged. (D65)

### unknown-name

**A name that nothing in scope declares.** `unknown name 'cout'; did you
mean 'count'?`, `unknown function 'lenght'`, `unknown type 'Strng'`.

When a declared name is close — a typo, a different case, the name
another language uses (`size` for `len`, `append` for `push`) — the
message suggests it and the editor offers it as a quick fix. It is a
guess, so `veles check --fix` never applies it. Otherwise the name needs
a declaration, or a `use` of the module that has it.

### members

**A field, method or static that the type does not have.** `no method
'size' on 'List<i64>'; did you mean 'len'?`, `type 'Point' has no field
'z'`, `'Point' has no static 'origin'`.

A method called as a field (`xs.len`) or a field called as a method
(`p.x()`) is told which it is. Hover shows everything a type has, and
completion after the `.` lists it.

### modules

**`use`, and what a module has.** `unknown module 'jsn'`, `module 'io'
has no declaration 'prinln'; did you mean 'io.println'?`, `import cycle:
module 'a' is already being loaded`.

A module is a directory of the package, a dependency named in
`veles.toml`, or a standard module. Its members are reached through its
name — `io.println` — or, when a braced import lists it, bare
(`use io { println }`, D85). There is no `{ * }`; a bare name may not equal
another import or a declaration of the module, and a listed name nothing
uses is a warning.

#### `import cycle: module 'a' is already being loaded`

Modules must form a DAG (M4). Move the shared code into a third module
both can import, or merge the two.

### private-to-module

**A name another module keeps to itself.** `'helper' is private to module
'geometry'; declare it 'public' there to use it from here`.

Mark it `public`, or if it is meant to stay internal, add a `public`
function in that module that does what you need (M5). The same holds for
a field, a method or a static of a type from another module. For a
standard module the message says instead that the name is not part of
the standard library's API: it is an internal helper, and the public
functions around it are the way in.

#### `module 'x' of package 'lib' is not in its exports`

The dependency's `veles.toml` decides what leaves the package. Import an
exported module, or add `x` to that package's `exports` (M5).

### private-to-type

**A member only the type's own code may touch.** `field 'items' is
private to 'Acc': only its own methods, impl and extend blocks may use
it`, `cannot assign to 'T.f' here: the field is 'protected var'`.

`private` hides a field or method from everything outside the type's own
declarations — its methods, `implement` and `extend` blocks. Reach the
value through a method of the type instead.

#### `cannot assign to 'T.f' here: the field is 'protected var'`

A `protected var` field is assigned only inside the type's own
declarations — its methods, `implement` and `extend` blocks; everyone else
reads it. Call a method of the type, or make the field a plain `var` if
outside writes are intended (D22).

### type-mismatch

**A value of one type where another is needed.** `type mismatch: expected
'i32', found 'i64'`, `branches have incompatible types 'string' and 'i64'`.

There are no implicit conversions. When a conversion is the usual fix the
message names it: `as` between numbers, `.value` for an enum's number,
interpolation for building a string, `?:` for a nullable.

#### `type mismatch: expected 'i32', found 'i64'`

There are no implicit numeric conversions (D21). Convert with `as`:
`n as i32`. Integer literals, lengths, indices and `toInt()` are all
`i64`, so the mismatch usually means a signature was written with `i32`
for no particular reason — use `i64` unless the width matters.

#### `a function value does not take on 'suspends' or 'throws'`

A named function passed where a `suspends` or `throws` function type is
wanted: pass a lambda that calls it, `(x) => f(x)` (D40).

### inference

**Nothing says what the type is.** `cannot infer the element type of an
empty list; annotate it`, `cannot infer the type of 'null' here`.

Write the type where the value is bound — `val xs: List<i64> = []`,
`val x: string? = null` — or pass the value where a type is already
expected: a parameter, a field, a return.

#### `cannot infer the element type of an empty list; annotate it`

`[]` and `[:]` have no elements to infer from. Write the type on the
binding — `var xs: MutableList<i64> = []` — or pass the literal
where the expected type is known (a parameter, a field, a return). (D25)

### generics

**Type arguments: too many, too few, or none that can be worked out.**
`'Box' expects 1 type arguments, got 2`, `'Plain' is not generic: write it
without '<...>'`, `cannot infer type parameter 'T' of 'first'; write
'first<...>(...)'`.

Type arguments are usually inferred from the arguments. When there are
none to infer from, write them after the name: `parse<Config>(text)`.

### nullable

**A value that may be absent, used as if it were there.** `value of type
'string?' may be null; use '?.', '?:' or check for null first`.

A nullable value is not a value (D5). `x ?: fallback`, `x?.member`, or
`if (x == null) return ...` and continue with `x` narrowed. The reverse
mistakes — `?.` on a value that cannot be null, comparing one with
`null` — are reported too: the check can never do anything.

#### `'?:' is for a nullable; a Result's value-or-fallback is '??'` (and the reverse)

`?:` unwraps a `T?`, `??` unwraps a `Result` — one idea, split by what is
on the left, so the operator tells the reader which kind of "maybe" it is.
The quick fix (and `veles check --fix`) swaps the operator. (D61)

### results-and-errors

**`throws`, `try`, and failures nobody handled.** Errors are values of a
function's result (D4): a call to a `throws` function gives a `Result`,
and it has to be propagated with `try` or handled with `when`, `??` or
`?!`.

#### `unused Result: the call may fail; use 'try' to propagate the error or 'when' to handle it`

You called a `throws` function as a statement, or bound its result to a
name (`unused Result 'r'`) and never read it. Decide: `try f()` if your
function is `throws`, or `when (val r = f()) { is Ok => ...; is Err => ... }`.
Ignoring a failure is not an option; `val _ = f()` is the deliberate
discard. (D4)

#### `'try' propagates an error, but the enclosing function is not declared 'throws'`

Add `throws` to the function (the error type is inferred), or handle the
`Result` with `when`.

#### `'throw' fails the function, but it is not declared 'throws'`

Same fix: add `throws`, or return a `T?` if "nothing" is the right
answer rather than "failure".

#### `'f' is declared 'throws' but nothing in its body can throw`

A warning: no `throw`, and no `try` of something that can fail, anywhere
in the body — so the clause only makes every caller write a `try` that
can never propagate anything. Remove it; `veles check --fix` does, and on
the next pass also removes the callers' now-pointless `try` (which is the
error `'try' needs a Result`, carrying its own fix). A `public` function,
a trait method and its implementations, and a function passed as a value
are left alone: there the clause is a contract, not a claim about one
body. (D45)

#### `'Dog' is not an error: declare it with 'error Dog { ... }' instead of 'struct'`

Only types declared with `error` (or given an explicit `implement Error for`)
can be thrown or named in `throws`. Change `struct` to `error`, or wrap a
non-struct value: `error Failed { why: string }`. A generic `throws X`
needs `X: Error`.

#### `only 'message' can be overridden in an error`

Inside `error Name { }`, `fun message(): string = ...` replaces the
default text with no `override` needed; any other method is an ordinary
method and cannot be marked `override`.

#### `'PortErrors' names an error set; it can only appear after 'throws' ...`

`error PortErrors = A | B` is a name for a union, and unions exist only in
error position (D45): after `throws`, inside another error set, or as the
type of a field of an `error` (a cause). It is not a value type — use
`when` on the members, or one of the members itself.

#### `error set 'S' refers to itself`

Two error sets include each other. Sets flatten, so a cycle adds nothing;
remove one direction.

### val-else

**`val ... else`: binding a value that may not be there.**

#### `the 'else' of a 'val ... else' must leave`

`val x = r else { ... }` binds `x` only when `r` has a value; on the other
path `x` does not exist, so the block cannot fall through to code that
uses it. End it with `return`, `break`, `continue`, `throw` or `panic`. To
carry on with a default instead, use `r ?? fallback` (a Result) or
`n ?: fallback` (a nullable). (D61)

#### `nothing here can fail: 'val ... else' needs a nullable, a Result or a pattern`

The value always exists, so the `else` could never run. Drop it. (D61)

### numbers

**Literals that do not fit, and integers where they are required.**
`literal 300 does not fit in 'u8', which holds 0..255`, `integer literal
does not fit in 64 bits`, `negative literal for unsigned type 'u8'`.

Pick a type that holds the value, or write a float (`1e20`) when an
approximation will do. Ranges, indices and shift counts are integers;
convert a float with `as`.

### operators

**An operator the type does not have.** `operator '-' is not defined for
'Point'; implement 'Subtractable' (fun minus(other: R): Out) to give it one`, `'%' is
not defined for floats; 'x.mod(y)' is the remainder`.

Arithmetic comes with the numbers; a type of your own gets an operator by
implementing its trait (D71). `==` needs `Equatable` and `<` needs
`Comparable` — both derived for most types. `as` converts between numbers
only.

### text

**Strings, bytes and characters.** `strings are not indexable; use
's.byteAt(i)' for a byte or 's.chars()' for the code points`, `a byte
literal holds one ASCII character`, `cannot interpolate a value of type
'()'`.

#### `a byte literal holds one ASCII character`

`'x'` is a `u8` — the byte of one ASCII character, for code that walks a
string with `byteAt` (`b == '"'`, `b >= '0' && b <= '9'`). A character
outside ASCII has no single byte to be; use a one-character string, `"é"`
(D18).

#### `unexpected character U+00A0, a no-break space; use an ordinary space`

Names may use letters from any script — `café`, `日本語`, `µs` — and
combining marks and digits after the first letter (Unicode's identifier
rules, UAX #31). A character that is none of those is reported by name,
because it is usually invisible: a no-break space pasted from a web page,
a zero-width space, a byte-order mark in the middle of a file. A
*bidirectional control* (U+202A–U+202E, U+2066–U+2069) is refused in code
and in comments: it makes a file display in a different order from how it
compiles (the "Trojan source" attack). Inside a string it is a warning —
write it as `\u{202E}` so it can be seen. (D18)

#### `interpolating 'this' inside its own 'toString' calls the same 'toString' again, forever`

Interpolation prints a value through its `Display` implementation, so
`"$this"` inside that very `toString` is the function calling itself with
the same argument — a stack overflow at run time. Interpolate the fields
(`"(${this.x}, ${this.y})"`). Printing a *different* value of the type is
allowed, since that recursion can end. (P7)

### mutability

**Changing what cannot change.** `cannot assign to 'x': it is a 'val';
declare it with 'var'`, `cannot push into an immutable List; use
MutableList`.

A `val` is never rebound, a bare field is never reassigned, and a `List`,
`Map` or `Set` never changes; the `var`, `var` field and `Mutable…`
versions do (D11, D22, D25).

#### `cannot assign to 'x': it is a 'val'; declare it with 'var'`

A binding declared `val` cannot be rebound (D11). It says nothing about
the value's insides: a `val` struct's `var` fields and a `val`
collection's contents can still change.

#### `cannot assign to 'T.f': the field is immutable; declare it 'var f: ...'`

A bare field is set by the constructor call and never assigned again,
whoever holds the struct — through a binding, a pointer or `this`. Mark
the field `var` if it is meant to change, `protected var` if only the
type's own code should change it, or build a new value (D22). A
method call on a global `val` is refused the same way when the method
changes its receiver: a global is shared by every task (D35).

#### `cannot push into an immutable List; use MutableList`

`List` is immutable. Use a `MutableList` (`mut [...]` or the
`MutableList<T>` annotation), or build a new list with `map`/`filter`.
(D25)

### references

**Changing a copy when the original was meant.** `'p' is a copy of the
element, not the element; assign through 'xs', or use 'set'`, `'&x' is the
address of a copy of the element`.

A struct is a value (D7): reading it out of a list, a map or a parameter
copies it. To change the element itself, reach it in place —
`loop (&p in points)`, `xs.set(i, p)` — or take a pointer, `*T`.

#### `'f' is a copy of the caller's 'Fuzzer': this function calls 'range', which changes 'f.rng', and the caller never sees the change`

A warning. A struct is a value (D7), so a parameter is a copy of what the
caller passed; changing its `var` fields changes the copy. Take a pointer
— `f: *Fuzzer`, called with `&fuzzer` — to change the caller's value, or
return the changed one. Fields that are references (a `MutableList`, a
`*State`) are shared, so pushing into them is seen and is not reported.

### calls

**The arguments do not fit the function.** `'at' takes 1 argument:
at(i: i64): string?`, `missing argument 'port' in call to 'connect'`,
`positional argument after a named argument`.

The message shows the function's parameters; hover and signature help in
the editor show them as you type.

#### `positional argument after a named argument`

Once you name one argument, name the rest: `f(1, b: 2, c: 3)` is fine,
`f(a: 1, 2)` is not (D28).

### constructors

**Building a struct.** `construct 'Point' by field name`, `missing field
'y' in constructor of 'Point'`, `'init' does not assign 'id' on every
path`.

A struct is built by naming its fields, `Point(x: 1, y: 2)`; a field with
a default may be left out. An `init` block runs after the fields are set
and must give every field without a default a value before it ends (D28,
D73).

### this

**The receiver.** `'this' is only inside a method; a function outside a
type takes the value as a parameter`, `'this' is not available in a static
function`.

A method's receiver is `this` and its type is `Self`; a static function
and a top-level function have neither (D23, D65).

### declarations

**How a declaration is written, and where.** `duplicate field 'x'`,
`'x' is already declared in this scope`, `'static' belongs to a function in
a struct, trait, impl or extend body`, `struct 'Node' has infinite size`.

#### `'protected' qualifies 'var'`

`protected` restricts *who may assign* a field, so it goes with `var`:
`protected var count: i64`. A bare field is never assigned by anyone, and
so has nothing to protect (D22).

### traits

**What a type must provide to be used as a trait.** `'Point' does not
implement 'Display'; add 'implement Display { ... }' to its declaration`,
`method 'area' returns 'f64' but trait 'Shape' declares 'i64'`.

A type implements a trait with an `implement` block — in its body, or at
top level as `implement Trait for Type` — whose methods match the trait's
signatures. Many standard traits (`Equatable`, `Hashable`, `Display`,
`Codable`, …) are derived instead: the compiler writes them (D58).

### sealed-types

**Sealed traits and their variants.** `'Shape' is a sealed trait;
construct one of its variants, e.g. 'Shape.Circle(...)'`, `'Square' is not a
variant of 'Shape'`.

A sealed trait is a closed set of variants declared in its module (D12).
Build a variant, and match on the variants with `when` to reach what only
some of them have.

#### `variant 'Rect' of 'Shape' does not implement 'area'`

A method the sealed trait declares without a body is written by every
variant, inside `implement Shape { }` in the variant's body — a call on a
`Shape` runs whichever variant it holds. When the method is already in
the variant's body but outside that block, the message says to move it
in.

### enums

**Enums: a closed set of integer values.** `an enum is not its number —
read it with '.value'`, `enum 'Color' has no member 'Purple'`.

An enum member converts to its number with `.value` and back with
`Color.fromValue(n)`; an enum has no methods of its own, and compares,
hashes, orders, prints and encodes by itself (D57).

### when-and-patterns

**`when`, `is` and destructuring.** `'when' is not exhaustive: Rect`,
`tuple has 3 elements but the pattern has 2`, `subject has type 'i64',
which can never be 'string'`.

#### `'when' is not exhaustive: Rect`

A `when` over a sealed type must cover every variant or end with `else`
(D13). Prefer listing the variant so the next one you add is caught too.

### control-flow

**Where execution goes.** `missing return: function 'f' must return a
value of type 'i64'`, `unreachable code: the statement before it always
leaves`, `'break' outside of a loop`.

A function with a result type ends in a value on every path — the last
expression of its body, or a `return`. Code after `return`, `break`,
`throw` or `panic` never runs.

### unused

**Something written and never used.** Warnings: `'x' is never used`,
`result of 'f' must be used`.

#### `'x' is never used`

The `val`/`var`, loop, destructuring or pattern binding is never read
(assigning to it does not count). Use it, drop it, or name it `_`.

### tasks

**Tasks, suspension and what may cross between them.** `argument of type
'MutableList<i64>' is not Sendable and cannot cross a task boundary`,
`'async' must be lexically inside a 'scope' or 'gather' block`.

#### `argument of type 'MutableList<i64>' is not Sendable and cannot cross a task boundary`

Tasks may only share immutable data (D35). Pass a `List` (`.toList()`),
send items over a `Channel`, or guard shared state with `Mutex(value: ...)`.

#### `'await' applies to channels, timers and task handles` / `'recv()' always suspends and must be awaited`

Suspension is inferred (D2/D16). Call ordinary functions normally, even
those that suspend; write `await` only on `sleep(...)`, `ch.recv()` and
task handles from `async`.

#### `'await' is not written on a call: a function that suspends is called like any other`

`await net.connect(host, port)` is written `net.connect(host, port)`: the
call suspends by itself (D2), and `try` goes straight on it
(`try net.connect(...)`). The quick fix removes `await`.

### unsafe-and-ffi

**`unsafe`, raw pointers and C.** `calling extern "C" function 'strlen'
requires an 'unsafe' block`, `pointer arithmetic requires an 'unsafe'
block`.

#### `calling extern "C" function 'strlen' requires an 'unsafe' block`

Wrap the call: `unsafe { strlen(p) }`. The block marks the places where
the compiler cannot vouch for memory safety (D44).

#### `an 'unsafe' block needs a '// SAFETY:' comment on the line above saying why it is sound`

A warning. Write why the block is sound in a `// SAFETY:` comment on the
line above it (or as the first line inside it) — what the C function
reads and keeps, why the pointer is live. The quick fix inserts the
comment; until the reason after `SAFETY:` is filled in, the warning says
it "gives no reason". Or declare the function `unsafe fun`, so its
callers take on the obligation.

### attributes

**`@` attributes and derived code.** `@key needs the name: @key("user_id")`,
`cannot derive 'Codable' for 'Conn': ...`, `'old' is deprecated: use 'new'`.

Attributes shape what the compiler derives (D58): `@key` renames a field
on the wire, `@skip` leaves it out, `@tag` names a sealed variant.
`veles explain <path> --derive` prints the code a derive wrote.

### tests

**Tests and the test-only vocabulary.** `'expect' is for tests`, `a test
takes no parameters and returns nothing`.

`expect`, `require` and `expectThrows` exist only inside `test "…" { }`,
a `test fun` and `*.test.vs` files; outside a test, `assert(cond, "why")`
states an invariant (D78).

### manifest

**`veles.toml`.** `veles.toml:3: unknown [format] key "indnt"`.

The message names the file, the line and the keys that section accepts.

### internal

**The compiler's own consistency checks.** `unsupported expression`.

A program should never reach one of these: it is a compiler bug. Please
report it with the smallest program that shows it.

## At run time

A panic is not a compile error; it ends the task that raised it and
prints where (D64).

### `panic: index 3 out of bounds for list of length 1`

`xs.set(i, v)` or `xs.swap(i, j)` with a bad index. A read is `xs.at(i)`,
a `T?` — or a `T` where the compiler can see the index is in range (D62);
where it cannot, say why the read cannot fail: `xs.at(i) ?: panic("…")`.
The line under the message, `at main.vs:7:31`, is where it happened (D64). A debug build adds `in <function>` and a
`called from file:line:col in <function>` line per call up to the task's
first function (D81); a release build prints the location and
`(a debug build shows the call chain)`.
Panics end the current task; a `gather` reports them as `Err(Panic)`,
a `scope` re-raises them (D52).

### `panic: integer overflow`

Checked arithmetic (D21). Use a wider type, or the wrapping operators
`+% -% *%` if wrapping is intended. `--release` builds drop the check.
