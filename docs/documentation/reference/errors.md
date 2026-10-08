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

#### `module 'x' of package 'lib' is not re-exported`

A package's root module decides what leaves the package: `public use x`
there (D89). Import a module it re-exports, or add that line to the
dependency's root module. The message names the line to add.

#### `'public use m' re-exports a module of another package`

Only a package's own modules can be re-exported; a dependency's or a
standard module's stays where it is — import it where it is used.

#### `'x' is already declared in this module` (a re-export)

A re-exported name lives in the module's namespace like a declaration.
Re-export it under another name: `public use m as other`, or
`public use m { x as other }`.

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

#### `cannot take 'T.f' out of 'T': the field is 'protected var'`, `cannot call 'push' on 'T.f'`

On a `protected var` field whose type is a mutable collection, the
contents belong to the type too (D87). From outside, read it through a
method that does not change it (`t.items.len()`, `.first()`, `.contains(x)`),
loop over it, or interpolate it. Binding, passing or returning the field
would hand out the handle, and a mutating method (`push`, `set`, `clear`,
`swap`, `ref`, ...) changes it: give the type a method that does the change,
or copy with `t.items.toList()`.

### type-mismatch

**A value of one type where another is needed.** `type mismatch: expected
'i32', found 'i64'`, `branches have incompatible types 'string' and 'i64'`.

There are no implicit conversions. When a conversion is the usual fix the
message names it: `.toI64()` / `.wrapU8()` between numbers, `.value` for an enum's number,
interpolation for building a string, `?:` for a nullable.

#### `type mismatch: expected 'i32', found 'i64'`

There are no implicit numeric conversions (D21). Convert with a method (D86):
`n.toI32()` is an `i32?` (null when it does not fit), `n.wrapI32()` keeps the low bits. Integer literals, lengths, indices and `toInt()` are all
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
none to infer from, write them after the name: `parse<Config>(text)`. A
static function of a generic type infers them the same way (D137):
`Box.of("x")` is a `Box<string>`; `Box.empty()` with nothing to go on
needs `Box<string>.empty()` or a typed binding.

#### `a Secret holds text or bytes: 'Secret<string>' or 'Secret<List<u8>>', not 'Secret<i64>'`

`Secret<T>` (D112) keeps its bytes in storage the collector wipes, so it
holds text or bytes only. Keep a number or a struct secret by storing its
text form, or keep the struct and mark the sensitive fields `Secret`.

### nullable

**A value that may be absent, used as if it were there.** `value of type
'string?' may be null; use '?.', '?:' or check for null first`.

A nullable value is not a value (D5). `x ?: fallback`, `x?.member`, or
`if (x == null) return ...` and continue with `x` narrowed. The reverse
mistakes — `?.` on a value that cannot be null, comparing one with
`null` — are reported too: the check can never do anything.

#### `'val x = ...' in a condition needs a value that can be null`

`if (val n = 5)` binds the value of a nullable when there is one, so the
value has to be a `T?`; for a plain value `val n = 5` is the binding. (D95)

#### `'val x = ...' is only allowed in the condition of an 'if', joined to the rest by '&&'`

The binding lives from where it is written to the end of the `then`
branch, which only makes sense along an `&&` chain: under `||` or `!` the
name might not have a value, and outside an `if` there is no branch to
scope it to. Bind with a `val` statement first, or use `val x = e else ...`
to leave when it is missing. (D95)

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

#### `'f' declares 'throws', but nothing in its body can throw`

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

Inside `error Name { }`, `fun message(): string => ...` replaces the
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

#### `a 'do' block needs a 'catch (e) { ... }' after it`

`do { ... }` exists to give a failing `try` or `throw` inside it somewhere
to go in the same function; without the `catch` it has nowhere. Write
`do { ... } catch (e) { ... }`. There is no `do`-`while`: for a loop that
runs once before it tests, `loop { ...; if (!cond) break }`. (D98)

#### `'catch' follows an expression that can fail or a 'do' block`

`catch (e) { ... }` is the failure branch of what comes before it — a call
to a `throws` function, `try chain`, or a `do { ... }` block — and does not
stand alone. (D98)

#### `'catch' handles the error of a Result, and a nullable has none`

`f() catch (e) { ... }` needs a `Result`, since a nullable fails with nothing
to bind. Its fallback is `f() ?: fallback`, and to leave when it is null,
`val x = f() else return`. (D98)

#### `'catch' needs a Result before it ... which cannot fail`

Whatever stands before `catch` has no failure to handle: it is not a call to
a `throws` function, not a `try` chain and not a `do` block. Drop the `catch`.
(D98)

#### `a handler that sees the error is 'catch (e) { ... }' now`

The spellings `r ?? { e => ... }` and `val x = r else { e => ... }` are gone;
one form gives a handler the error: `r catch (e) { ... }`. A fallback that
does not need the error is still `r ?? value`, and a let-else that does not
is still `val x = r else return`. (D98)

#### `the error is named in a head: 'catch (e) { ... }', not 'catch { e => ... }'`

The name goes in parentheses after `catch`, as `when (v)` and `loop (x in xs)`
take theirs; the block after it is an ordinary one. (D98)

#### `nothing in this 'do' block can fail`

No `try` of something that can fail and no `throw` in the block reaches the
`catch`, so it could never run. Drop the `do` and the `catch`, or put the
calls that can fail inside. (D98)

#### `a child that can fail cannot be launched in a 'scope' inside a 'do' block`

A `scope` passes a failing child's error to the function that contains it,
not to a handler around it. Put the `scope` in a function of its own and
`try` that function inside the `do`. (D98)

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
convert a float with `.toI64()` (null when it does not fit).

#### `'as' no longer converts (D86; it only renames)`

`x as i64` was the old spelling of every numeric conversion. Conversions are
methods now, each naming what it can lose (`n.toI64()`, `n.toU8()`,
`n.wrapU8()`, `p.cast<*raw T>()` — see chapter 2). The message names the one
that replaces the cast and the editor offers it as a fix; `veles check --fix`
applies it across a project. The fix for a narrowing between integers is
`wrapT()` (what `as` did); for a float to an integer it is `toT()`, a `T?`
that is null when the value does not fit, so the surrounding code decides
what that case means.

#### `the literal does not fit 'u8' (D86)`

`300.toU8()` can never succeed, so it is refused. Write `300.wrapU8()` for
the low bits, or a value that fits.

#### `'wrapI64' keeps the low bits of an integer`

`wrapT()` is integer to integer. A float becomes an integer with `toT()`
(null when it does not fit); an integer becomes a float with `toF64()`.

### constants

**What a `const` may hold, and what fails when it is computed.**
`constant overflow: 9223372036854775807 + 1 is 9223372036854775808, outside
'i64'`, `a constant cannot call 'three'`, `static assert failed: the wire
header is 16 bytes (HEADER_SIZE = 16)`, `initialization cycle: 'a' reads 'b',
which reads 'a'`.

A `const` is computed when the program is compiled (D113): literals, other
constants, operators, interpolation, `len()`, `toT()`/`wrapT()`, `if` and
`when`, tuples, enum members, structs built without an `init`, and the
read-only `List`, `Map` and `Set` — laid out once in the binary, never built
at start-up. What would fail at run time fails the build instead, in every
profile: an overflow (write `+%` to wrap on purpose), a division by zero, a
shift past the width, `TABLE.at(i)` out of range. A call of a function that is
not a `const fun`, a `val`, anything mutable or made at run time is refused:
compute it with `val`.

**`const fun`** is a function the compiler runs for a constant (D113). What it
may contain is checked where it is declared: `in 'const fun f': it calls 'g',
which is not a 'const fun'`, `it reads the module-level 'counter'`, `a lambda
is not supported in a 'const fun' yet`, `a 'const fun' cannot throw yet`, `a
'const fun' cannot use 'unsafe'`. What fails while one runs fails the build at
the constant, with the message and the chain of calls: `panic in a constant:
reached zero`, `evaluating the constant 'C' took more than 10000000 steps`
(`--const-steps n` raises the budget), `'f' recursed more than 4096 calls deep`.

`static assert(cond, "why")` checks a constant condition at compile time, at
module level or in a body; in a generic body it is checked per instance,
so `static assert(T implements Comparable, "…")` states what a type argument
must provide.

Module-level values (`val` too) are computed before `main` in the order they
need each other — through the functions their initializers call as well. A
value that needs itself, directly or round a cycle, has no first value:
compute it in a function, or pass it as a parameter.

### operators

**An operator the type does not have.** `operator '-' is not defined for
'Point'; implement 'Subtractable' (fun minus(other: R): Out) to give it one`, `'%' is
not defined for floats; 'x.mod(y)' is the remainder`.

Arithmetic comes with the numbers; a type of your own gets an operator by
implementing its trait (D71). `==` needs `Equatable` and `<` needs
`Comparable` — both derived for most types. `.toI64()` and `.wrapU8()` convert
between numbers.

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

#### `'xs.push' changes 'xs' while the loop at line 12 walks it`

A loop walks the collection as it is, so a call in its body that changes
its length or order — `push`, `pop`, `insert`, `removeAt`, `clear`,
`sort`, a map's `remove` or `set` of a new key, a set's `add` — would
make it visit elements twice or skip them (D102). Loop over a copy
(`loop (x in xs.toList())`, the quick fix), or collect the changes and
apply them after the loop. Replacing an element in place (`xs.set(i, v)`,
`*x = v` in `loop (&x in xs)`, `m.set(k, v)` for the key being visited)
is allowed, and so is a loop over `xs.indices()` or a range, which walks
no collection — the way to grow a worklist while walking it. A change
made where the compiler cannot see it (a function handed the same list)
panics instead, when the loop next steps: `'xs' changed while a loop
walked it`.

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

#### `subject has type 'Shape', which can never be 'Plain'`

On a trait object, `is T` asks for a concrete type the object may hold
(D135). A type that does not implement the object's trait can never be
inside it; check the name, or implement the trait for the type. The
warnings `… so this test is always true/false` mean the answer is known
where the test is written: the value's type is concrete, or the object's
trait already requires the one tested. Remove the test (an `is`
expression gets a fix that writes the answer).

### control-flow

**Where execution goes.** `missing return: function 'f' must return a
value of type 'i64'`, `unreachable code: the statement before it always
leaves`, `'break' outside of a loop`.

A function with a result type ends in a value on every path — the last
expression of its body, or a `return`. Code after `return`, `break`,
`throw` or `panic` never runs.

A package that sets `[lint] implicit_return = "lambda"` or `"expr"`
(chapter 11) asks for the `return` to be written in some `{ }` bodies:
`this package returns a '{ }' body's value with 'return'` points at the
last expression of such a body, and its fix writes `return` in front of it.

### unused

**Something written and never used.** Warnings: `'x' is never used`,
`result of 'f' must be used`.

#### `'x' is never used`

The `val`/`var`, loop, destructuring or pattern binding is never read
(assigning to it does not count). Use it, drop it, or name it `_`.

### tasks

**Tasks, suspension and what may cross between them.** `argument of type
'MutableList<i64>' is not Sendable and cannot cross a task boundary`,
`'async' must be lexically inside a 'scope' or 'gather' block, or be the
value of a 'with'`.

#### `'async' must be lexically inside a 'scope' or 'gather' block, or be the value of a 'with'`

A task cannot outlive the block that started it (D3/D34). Start it
inside `scope { }` when the block should wait for it (a pool of workers),
or as `with t = async f()` when it runs in the background until the block
ends, which cancels it then (D100) — a server in a test, a ticker.

#### `'TestServer' holds a running task, so it must be received with 'with' where it is made`

A struct with a `Task` field — or a field that holds one — keeps a task
running, and a task must belong to a block (D111). Receive the value with
`with srv = try serving()` (the quick fix turns `val` into `with`), or
return it straight to your caller, who then receives it. At the end of
the block its tasks are cancelled and joined, then it is closed. Storing
it, passing it, putting it in a collection or dropping it would leave
its task with no block to stop it.

#### `'async' would start a task whose result, a 'Worker', holds a task of its own`

A task's result reaches whoever awaits it, not a `with`, so a value
holding a task cannot come out of one (D111). Call the function directly
and receive what it returns with `with`.

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

#### `'recv' suspends, and the lock on 'notes' is held until the end of this block`

`with n = notes.lock()` holds the lock to the end of its block (D107), and
a task that waits while holding a lock can stall every other task that
needs it — the rule `withLock` gets from its lambda type. Do the waiting
before the lock or after it: give the lock a block of its own,
`with (n = notes.lock()) { … }`, and suspend after that `}`. Named
functions that suspend are refused the same way (`'fetch' suspends, …`),
and so is a call of a suspending function value (`this call suspends`).

#### `a send arm binds nothing`

`ch.send(v) => …` in a `race` has no value to bind (D108): when the arm
wins, `v` is in the channel. Drop the `val x =`.

### resources

**`with` and what it closes.** `'NoClose' is not Closeable; 'with'
resources must implement Closeable`, `'conn' cannot be returned: it is
closed when its 'with' block ends`.

`with conn = open(…)` closes `conn` when its block ends — the rest of the
block is its body — and on every way out before then: `return`,
`break`/`continue`, a failed `try`, `throw`, a panic, cancellation
(D43/D100). `with (conn = open(…)) { … }` is the same with a block of its
own, for closing before the enclosing block ends.

#### `'Res' is not Closeable; 'with' resources must implement Closeable`

Only a type that implements `Closeable` (one method, `close()`) can be a
`with` resource. A nullable is refused: unwrap it first
(`val f = open(p) ?: return`, then `with`). A task is a resource only
where `with` starts it: `with t = async f()`.

#### `'conn' cannot be returned: it is closed when its 'with' block ends`

A resource does not outlive its block (D100): it cannot be returned, be
the block's value, be stored in a variable or field declared outside the
block, be put in a collection or sent on a channel — nor can a lambda
that captures it. Return or store what you read from it instead. Passing
it, or a lambda capturing it, as an argument is allowed
(`withTimeout(d, () => try read(conn))`); a function that keeps what it is
passed is not caught, so do not keep a resource you are lent.

#### `'f' is never closed: bind it with 'with f = …' so it is closed at the end of the block`

A warning (D115). A `Closeable` made here — by a call or a constructor —
is neither closed nor handed on. The quick fix turns `val f =` into
`with f =`. Anything that hands it on silences it: `f.close()`, returning
it, storing it in a field, a collection or an outer variable, passing it
as an argument, capturing it in a lambda. The check stays inside the
function, so it never reports a hand-off — and cannot see a callee that
drops what it was given.

#### `the 'File' this returns is never closed: bind it with 'with', or discard it on purpose with 'val _ = …'`

A warning (D115): a statement throws away the `Closeable` a call just
returned, so nothing can close it. The quick fix writes `with _ = …`,
which closes it at the end of the block; `val _ = …` says the drop is
deliberate.

#### `'out' is closed when its 'with' block ends; closing it here would close it twice`

`with` calls `close()` on every way out of its block, so calling it by hand
as well closes the resource twice (D136). Remove the call (the quick fix);
to close earlier than the end of the enclosing block, give the resource a
block of its own: `with (out = …) { … }` closes at that `}`. A with-task's
`t.cancel()` is fine: it stops the task early, and the block's end joins it.

#### `'lock()' holds the lock to the end of a 'with' block, so it is usable only as a 'with' value`

`notes.lock()` returns the taken lock and a pointer to the value, and only
`with` gives the lock back on every way out (D107). Write
`with n = notes.lock()` (the quick fix turns `val` into `with`), or, for
one expression, `notes.withLock(n => …)`. The pointer `n` is refused
where a resource is: it `points into a Mutex that is unlocked when its
'with' block ends`.

#### `'r' is closed as soon as it is opened`

A warning: `with r = …` is the last statement of its block, so nothing
can use `r` before the block's end closes it. Use it in the statements
after the `with`, or drop the `with`.

#### `'with x = e' is a statement` / `a module has no end` / `a body without braces has nothing after it`

The statement form closes its resource where the enclosing block ends, so
it needs a braced block with statements after it: not at module level,
not as an expression body (`fun f() => …`), an operand, a `when` arm or a
braceless `if`/`loop` body. Write the block form `with (x = e) { ... }`
there, or add braces. One binding per statement: `with a = …` and
`with b = …` on two lines (they close in reverse order).

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

#### `'atUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block`

`atUnchecked`, `setUnchecked` and `byteAtUnchecked` (D114) are undefined
behaviour in a release build when the index is out of range, so they are
written inside `unsafe { }` with a `// SAFETY:` comment saying why the
index is in range. Most code wants the checked form the message names
(`at(i)`, `set(i, v)`, `byteAt(i)`); the compiler already drops a check
it can prove.

### attributes

**`@` attributes and derived code.** `@key needs the name: @key("user_id")`,
`cannot derive 'Codable' for 'Conn': ...`, `'old' is deprecated: use 'new'`.

Attributes shape what the compiler derives (D58): `@key` renames a field
on the wire, `@skip` leaves it out, `@tag` names a sealed variant.
`veles explain <path> --derive` prints the code a derive wrote.

#### `'@caller_location' is reserved for the standard library for now`

`@caller_location` makes a function's misuse panics point at its caller
(D88). Only std uses it so far; write the check in your own function
with a `panic` that names the argument, and the chain in a debug build
shows the caller. It cannot mark a trait implementation, an extern
function, or a function that suspends.

### tests

**Tests and the test-only vocabulary.** `'expect' is for tests`, `a test
takes no parameters and returns nothing`.

`expect`, `require` and `expectThrows` exist only inside `test "…" { }`,
a `test fun` and `*.test.vs` files; outside a test, `assert(cond, "why")`
states an invariant (D78).

### template-literals

**Template literals and `@template` functions.** `'plain' is not a template`,
`a template literal has nothing between the name and the string`, `the second
parameter of a @template is the values`.

`sql"select … ${id}"` calls a function marked `@template` with the text pieces
and the values (D129, [chapter 2](../02-values-and-strings.md)). The function must
be `@template` and take `(parts: List<string>, values: List<V>)`; the name must
be written right against the string (a space is an error), and it is a
function — or `module.function` — not a value. A value whose type does not fit
`V` is the usual type error at its `${…}`.

### manifest

**`veles.toml`.** `veles.toml:3: unknown [format] key "indnt"`.

The message names the file, the line and the keys that section accepts.
A dependency that names no source, two sources, a version that is not
`major.minor.patch`, a bare `"1.4.2"` where a path is expected, or a
`[workspace]` member that leaves the workspace each get a message saying
what to write instead (see *The manifest in full* in chapter 11).

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
