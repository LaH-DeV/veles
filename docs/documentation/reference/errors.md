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

A tuple labels every element or none, each label once (D166): `(x: 1, 2)`
and `(x: 1, x: 2)` are syntax errors.

#### `the text is not valid UTF-8 here (byte 0xE9): save the file as UTF-8`

A source file is UTF-8, as in Go (D199). A byte that does not decode —
usually a file saved as Latin-1 or Windows-1252, where `é` is the one byte
`0xE9` — is an error wherever it stands, in a comment or a string too, and
each such byte is reported; nothing else in the file is checked until it is
saved as UTF-8 (every editor can: "Save with encoding" or "Reopen with
encoding", then UTF-8). A byte-order mark at the very start is allowed.

### removed-syntax

**A spelling that Veles used to have, or that another language has.**
`'impl' is spelled 'implement'`, `there is no 'for'; every loop is spelled
'loop'`, `'xs[i]' is not indexing; brackets are for collection literals
only — use 'xs.at(i)'`, `'=' assigns; use '==' to compare`.

The old form still parses, so the rest of the file checks, and the error
carries the fix: the editor's quick fix, or `veles check --fix` for a
whole package, rewrites it.

`'indexOf' answers null when nothing is found, never -1 (D165)`: the
index-of family (`indexOf`, `indexOfFirst`, `lastIndexOf`, `binarySearch`,
`binarySearchBy`, `binarySearchWith`) answers `i64?`, so a test of the old
`-1` — `>= 0`, `< 0`, `== -1`, `!= -1` — is refused with the fix that tests
for null. Better still: `val i = xs.indexOf(v) ?: return`, after which
`xs.at(i)` is a `T`, or `contains` when the index is not needed.

#### `the receiver is spelled 'this'`

A method's receiver is `this`, as in most languages that have one. `self`
(Rust's and Swift's word) still reads as the receiver, so the rest of the
file checks, but every use is an error whose fix writes `this` —
`veles check --fix` (or the editor's quick fix) rewrites a whole package in
one run, `"$self"` and `"${self.x}"` included. (D65)

#### `the implementing type is spelled 'This'`

Inside a struct, trait or `implement`, the type being written is `This`, the
pair of the receiver `this`: `fun compareTo(other: This): Ordering`,
`This.fields`, `This.default()`. `Self` (Rust's and Swift's word) still reads
as that type, so the rest of the file checks, but every use is an error whose
fix writes `This`. (D175)

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

A method that exists only as an extension function in a module this file
does not import says so: `'shout' is an extension function in module
'strx': import it with 'use strx extend { slug }'` (D173, D185). Two imported extensions of one name, or an
extension and a trait method, are `ambiguous method 'twice' on 'string'`:
drop one of the imports or rename one.

A method called as a field (`xs.len`) or a field called as a method
(`p.x()`) is told which it is. Hover shows everything a type has, and
completion after the `.` lists it.

A tuple is read by position (`t.0`) or, when its type labels the elements,
by label: `tuple '(quotient: i64, remainder: i64)' has no element 'rest';
its labels are 'quotient', 'remainder'` (D166). A generic type keeps no
labels in its type arguments, so `box.value.lo` on a `Box<(lo: i64, hi:
i64)>` is refused where `Box` is written; read `box.value.0`, or make the
pair a struct.

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

The four levels (D142): `private` (the type), `internal` (the module),
nothing (the package), `public` (other packages). Reaching an `internal`
name from another module of the package: leave the word off to share it
with the package. Reaching an unmarked name from another _package_: mark
it `public` there, or have its facade export it. The same holds for
a field, a method or a static of a type. For a standard module or a
dependency the message says instead that the name is not part of that
package's API: it is an internal helper, and the public functions around
it are the way in.

#### `'public' on 'x' does nothing in a program`

A program (its root module declares `main`) has no other package importing
it, so `public` guards nothing. `veles check --fix` removes the word.

#### `'f' is public but its result type 'T' is not visible outside the package`

A `public` function, or a `public` field, names a type that other packages
cannot see (it is not `public` and no `public use` exports it). Make the type
`public`, or drop `public` from the function.

#### `package 'lib' has no module 'x'`

A module of another package is reached by its path (`use lib.geometry`) —
a directory of the package, or an alias its root module's `public use`
gives — and what it shows is what it declares `public`, or what a facade
exports (D142, D89).

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

A named function or a function variable passed where its effects do not
fit: one that throws where the parameter does not, for instance. Pass a
lambda that calls it, `(x) => f(x)`; a lambda written in place takes the
effects the parameter asks for (D40). A plain function where the
parameter only _may_ suspend is no error (`xs.map(double)`, D217).

#### `a function that does not suspend cannot stand here`

A plain function value accepted where a parameter may suspend (D217) is
refused when the call would run it as a suspending function after all:
the function it is passed to keeps it (stores or returns it), another
argument makes the call suspend, it is passed through a function value or
a trait object, or it is not an argument at all
(`val g: fun(i64): i64 suspends = double`). The fix is the lambda that
calls it, `(x) => f(x)`, which is compiled as a suspending function.

#### `type mismatch: expected '(y: i64, x: i64)', found '(x: i64, y: i64)'; the labels differ`

A labelled tuple where a tuple with other labels is expected (D166).
Unlabelled tuples convert to labelled ones and back, but between two label
lists the elements would silently change meaning — here `x` would be read
as `y`. When the positions are what you meant, drop the value's labels:
the fix deletes them from a literal (`(x: 1, y: 2)` → `(1, 2)`) or
rebuilds a variable unlabelled (`(p.0, p.1)`). When the names are what you
meant, reorder the elements. The same mismatch inside a list or a map
(`List<(a: i64, b: i64)>` as a `List<(b: i64, a: i64)>`) has no fix:
convert the elements with `map`.

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

#### `cannot interpolate 'expr' of type 'Result<T, E>': its error would be lost`

Printing an unhandled result would turn a failure into ordinary text.
Use `"${try expr}"` in a throwing function, or `"${expr ?? fallback}"`
to handle it locally. `check --fix` and the editor's preferred quick fix
add `try` when the enclosing function may throw. Explicit
`expr.toString()` prints the wrapper on purpose. Template values and
formatted interpolation follow the same rule. (D157)

#### `cannot interpolate task handle 'expr' of type 'Task<T>'`

A handle is not its task's value. Await it first, for example
`"${await task}"`. The task's errors belong to its owning block. (D157)

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

#### `this child can fail with '…', which fails the 'scope' … but the function is not declared 'throws'`

A child whose function `throws` fails its `scope` when it throws, and the
scope leaves with that error: the function containing it must be able to.
Declare it `throws`, or use `gather`, whose tuple keeps each child's
outcome as a value. (D4, D141)

#### `a task this value holds can fail with '…' … but the function is not declared 'throws'`

A value holding a task whose field is `Task<Result<T, E>>` hands the
task's error to the block that receives it with `with` (D111), and from
there to the function: declare it `throws`.

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

`x as i64` is how many languages convert a number; in Veles `as` only
renames an import. Conversions are methods, each naming what it can lose (`n.toI64()`, `n.toU8()`,
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

A `lazy val` (D210) is built on its first read instead, so its errors are
about that read: `the initializer of 'lazy val thrown' cannot throw or
suspend` (the read that builds it neither handles an error nor waits —
handle the error inside, or build the value in `main`), `'fine' is a lazy
value, built at run time on its first read; a constant cannot read it`,
and `lazy` on a `var` or a `const`, which is a contradiction: a lazy value
is built once and never changes, and a `const` is computed while
compiling.

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

#### `format spec '.2' does not fit a value of type 'string'`

A format spec (`"${price:>10.2}"`, D171) is checked against the type of the
value when the program is compiled. An integer takes `d x X b o`, a float
`f` or `%` and a precision, text and everything that prints takes a fill, an
alignment and a width. The message says which part does not fit. A spec that
is not well formed (`"${n:q}"`) is reported with the grammar:
`[[fill]align][sign][#][0][width][,][.precision][type]`. Not available yet:
the type `e` and a computed width (`${x:>${w}}`).

#### `a byte literal holds one ASCII character`

`'x'` is a `u8` — the byte of one ASCII character, for code that walks a
string with `byteAt` (`b == '"'`, `b >= '0' && b <= '9'`). A character
outside ASCII has no single byte to be; use a one-character string, `"é"`
(D18).

#### `a text block's text starts on the line after the opening """`

A text block (D172) is `"""`, a line break, the lines of text, and a closing
`"""` on a line of its own. The closing delimiter's indentation is the margin:
it is removed from every line, so each line of the text must be indented at
least that far (a blank line may be shorter). Put the text on the lines
between, and the closing `"""` alone on its line.

A raw string between backticks has no such rules, and no escapes or
interpolation; it is closed by the next backtick.

#### `unexpected character U+00A0, a no-break space; use an ordinary space`

Names may use letters from any script — `café`, `日本語`, `µs` — and
combining marks and digits after the first letter (Unicode's identifier
rules, UAX #31). A character that is none of those is reported by name,
because it is usually invisible: a no-break space pasted from a web page,
a zero-width space, a byte-order mark in the middle of a file. A
_bidirectional control_ (U+202A–U+202E, U+2066–U+2069) is refused in code
and in comments: it makes a file display in a different order from how it
compiles (the "Trojan source" attack). Inside a string it is a warning —
write it as `\u{202E}` so it can be seen. (D18)

#### `interpolating 'this' inside its own 'toString' calls the same 'toString' again, forever`

Interpolation prints a value through its `Display` implementation, so
`"$this"` inside that very `toString` is the function calling itself with
the same argument — a stack overflow at run time. Interpolate the fields
(`"(${this.x}, ${this.y})"`). Printing a _different_ value of the type is
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

#### `'items' is a 'MutableList<i64>'; a const struct holds only immutable data: use 'List<i64>'`

A `const struct` promises that nothing reached through its values
changes (D212), so each field must be deeply immutable. The message names
the field and what to use: the immutable collection, `string` for a
`StringBuilder`, `const struct` on a plain struct you declared (the
promise is written at every level), a `const` variant for a sealed type.
A `var` field is refused the same way; build a changed value with
`copy`. Pointers, functions, trait objects, channels, task handles and
`Mutex`/`Atomic` are state, not data: keep them beside the value. For a
generic one the error is where the type arguments are written:
`'Pair' is a const struct, so its field 'second' is a 'MutableList<i64>'`.

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
a default may be left out. A constructor that takes one parameter also
takes it positionally, `Meters(5)` (D163); when the type grows a second
one, the call is this error with a fix that names the field. An `init`
block runs after the fields are set and must give every field without a
default a value before it ends (D28, D73).

`'copy' takes the fields it changes by name`, `field 'area' is assigned by
the 'init' block of 'Rect', which a copy runs again`, `'Worker' holds a
running task, which a copy would duplicate`: the generated `copy` (D164)
names what changes, cannot name what `init` assigns or what the call site
cannot see, and is refused for task-holding types, `with`-bound resources
and types whose `init` takes parameters.

### this

**The receiver.** `'this' is only inside a method; a function outside a
type takes the value as a parameter`, `'this' is not available in a static
function`.

A method's receiver is `this` and its type is `This`; a static function
and a top-level function have neither (D23, D65).

#### `'name' means this.name here (D158)`

Inside a method a bare name that is a member of the type means
`this.name`; a module-level function or value of the same name is hidden by
it (a warning, because the code compiled before the member existed and
reads the same). Rename one of the two. For a `static` member the warning
reads `'name' means Type.name here`.

#### `'name' is a member of 'T', but this function is static`

A `static fun` has no receiver, so an instance field or method is not
reachable bare. Take the value as a parameter, or make the function a method.

#### `a block '{ … }' alone is a function where one is expected (D207)`

A block written alone, `{ … }`, is a function of no parameters, and it takes
its meaning from where it goes: where a `fun on T()` is expected it is a
_receiver block_, whose `this` is the T (`server.apply { port = 8080 }`);
where a `fun()` is expected it is a plain function (`measure { work() }`).
With nothing expected — `val f = { … }` — say which with a type:
`val f: fun() = { … }`.

#### `this block takes no parameters, but a 'fun(i64)' is expected (D207)`

A block never has parameters. A function that takes some is written
`(a) => { … }` — also where a `fun on T(A)` is expected, with `this` still
the T.

#### `'apply' changes 'x' in place, and 'x' is a 'val' (D207)`

`apply` runs its block on the value where it is held, so the place must be
the caller's to change: a `var`, a `var` field, what a pointer points at, or
a temporary. On a `val` use `edit`, which gives a changed copy:
`val changed = x.edit { … }`. The fix rewrites the call.

#### `'x' is a global 'val' … a function on it may change it (D207)`

A `fun on T` gets a pointer to its receiver, so it can change it; a global
`val` is a constant every task shares. Copy it into a `var` first.

### declarations

**How a declaration is written, and where.** `duplicate field 'x'`,
`'x' is already declared in this scope`, `'static' belongs to a function in
a struct, trait, impl or extend body`, `struct 'Node' has infinite size`.

#### `extension 'trim' on 'string' is never called`

`extend T { }` on a type your package does not own declares _extension
functions_ (D173), and an inherent method of the type always wins: the
struct body's, a compiler built-in's, or a method of the same name in an
`extend` of the type's own package. An extension with such a name is never
reached; rename it. Also refused there: a `static fun` (write a free
function in the module), `private` (leave it off, or write `internal`), and
extending a `Result` or an `Option`. See
[chapter 5](../05-structs-and-methods.md#extension-functions-extend-on-a-type-you-did-not-write).

#### `'protected' qualifies 'var'`

`protected` restricts _who may assign_ a field, so it goes with `var`:
`protected var count: i64`. A bare field is never assigned by anyone, and
so has nothing to protect (D22).

#### `'xs' is used before it is assigned on every path to here`

A local `var` may be declared without a value (`var name: string`) when the
branches below give it one, but nothing may read it, take `&` of it,
capture it in a lambda, call a method on it or write one of its fields until
every path to that point has assigned it (D214). An assignment inside a
loop body does not count after the loop, since the body may run no times;
one inside a lambda counts only inside it. Give the variable a value where
it is declared, or assign it on each branch above the use.

#### `an extern struct is laid out as C lays it out … drop 'compact'`

`compact struct` lets the compiler place a struct's fields for the smallest
size (D215). It is refused where the layout is not the compiler's to choose:
an `extern struct` (C's order; `@packed` applies only there) and a
`@transparent` one (it is its one field). On a struct with fewer than two
fields it is a warning: there is nothing to place. See
[chapter 13](../13-memory-and-ffi.md#field-order-and-compact-structs).

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

#### `variant 'Upload' is 9 words, the others at most 1 word: mark it 'indirect'` (a warning)

A sealed value is sized for its largest variant, so a type that is mostly
small values with one large variant carries that room in every value — in
every list and map of them too. Mark the large one `indirect struct` in the
trait's body (the fix does it): it moves to the heap and the value shrinks
to a tag and a pointer for it (D198). A tree — a sealed type that holds
itself — is boxed already and never warned. Silence it in `veles.toml`
with `[lint] large_variant = "allow"` when the size is what you want.

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

#### `'async' must be lexically inside a 'scope' or 'gather' block, or be the whole value of a 'val' or 'with'`

A task cannot outlive the block that started it (D3/D34). Start it as
`val t = async f()` when the block needs its value (it must then await
it, D156), inside `scope { }` when the block should wait for many (a pool
of workers), or as `with t = async f()` when it runs in the background
until the block ends, which cancels it then (D100) — a server in a test,
a ticker.

#### `module-level 'val cache' has type 'MutableMap<i64, i64>', which is not Sendable`

Every task, on any thread, reaches a module-level `val` or a `static
val`. A mutable collection, a `StringBuilder`, or a struct holding one
would be changed through its methods by several threads at once (D66).
Make the value immutable (`List`, `Map`, `Set`; `.toList()`), keep it
behind a lock (`val cache = Mutex(value: MutableMap<i64, i64>())`), or
keep it in an executor's state when one thread owns it (D150). For a
trait object, declare the trait `Sendable` (`trait Clock : Sendable`) so
every implementor is.

#### `module-level 'var total' is shared by every task`

No module global is a `var` (D66). For a plain value, keep the state
behind a lock: `val hits = Atomic(value: 0)`. For a `Mutex`, an `Atomic`
or another synchronized type, the lock is already inside the value: the
global itself must not be replaced, because a task still holding the old
lock would run beside a task taking the new one. Declare it `val` (the fix
does this), and change what it holds: `total.set(0)`, `hits.store(0)`,
`ready.reset()`. An executor's state is the one place a global `var` lives
(D150).

#### `'t' is not awaited on this path` / `'t' is not awaited on every path to the end of its block`

A task bound with `val` runs alongside the rest of its block, and every
normal way out of the block — its end, a `return`, a `break` or `continue`
that leaves it — must have awaited it (D156): otherwise its value would be
lost and nobody would know whether it failed. Await it on that path, or,
when leaving it behind is what you mean, start it with `with t = async
f()`: a `with` task is cancelled when its block ends. A failed `try`, a
`throw` or a panic needs no await — the task is cancelled and joined on
the way out.

#### `'t' must be awaited by this block, so it cannot be passed on, stored, captured or cancelled`

A `val` task is checked to be awaited on every path, which only works
while every use of it is an `await` in its own block (D156). Pass the
_value_ (`show(await t)`), start a task you want to cancel with `with`,
and keep handles you hand around in a `scope`.

#### `a task is bound with 'val' (it must then be awaited) or 'with' (it is cancelled when the block ends), not 'var'`

A handle names one task; it is never reassigned (D156).

#### `'f' runs with nothing alongside it; call 'f(…)' directly` (warning)

The first thing after `val t = async f()` is `await t`, so the block
waits at once and nothing runs at the same time: the task only costs a
task (D156). Call `f()` — a suspending function needs no keyword (D2).
Start the task earlier, or start two, when you want them to overlap.

#### `'work(1)' finishes before 'awaitAll' starts, so these run one after the other`

`awaitAll` waits for tasks (D156). A plain call among its arguments runs
to its end before `awaitAll` is even called; write `async work(1)` (the
quick fix) so the calls run at once. `'awaitAll' waits; remove 'await'`
says the opposite: `awaitAll` is itself where the task waits, so no
`await` goes before it.

#### `'workers' must be at least 1` / `this body runs in the calling task` / `'scope(workers: …)' … starts none`

`scope(workers: n)` runs at most `n` of the scope's children at once
(D156), so `n` is at least 1. A `scope(on: e)` or `scope(workers: n)`
whose body starts no task is refused: the body itself always runs in the
calling task, so `on:` would move nothing and `workers:` would limit
nothing. To run code on an executor, write `e.run(() => …)`.

#### `a 'scope' used as a value ends in an expression`

A `scope` used as a value gives its body's last expression once every
task it started has finished (D156); a body that ends in a loop or an
assignment has none to give.

#### `this task cannot be stored in 'tasks', which outlives the block: a task belongs to the 'scope' that started it`

When control passes a `scope`'s `}`, each of its tasks has finished — or,
if the scope was left early, has been cancelled (D141). A handle kept in
something declared before the scope (an outer `var` or list, a field), or
returned, would outlive the task it names. Declare the list inside the
scope, await there, and keep the _values_:

```veles
// fragment
var total: i64 = 0
scope {
  val tasks: MutableList<Task<i64>> = []
  loop (u in urls) tasks.push(async fetchSize(u))
  loop (t in tasks) total += await t
}
```

Tasks you never await are fine: the scope waits for them anyway.

#### `a task of this 'scope' is 'Task<i64>', not 'Task<Result<i64, Boom>>'` / `'await' gives the task's value`

In a `scope`, and for `with t = async f()`, a child's error fails the
block (D34) — it never reaches an `await`. So the handle of a throwing
`f` holds `f`'s value, and `await t` needs no `try` (D141; the quick fix
removes it). To handle a child's error yourself, use `gather` (its
handles and results keep the `Result`), or launch a function that
_returns_ a `Result` instead of throwing.

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

#### `'wait()' always suspends and must be awaited: 'await ready.wait()'`

`Event.wait()`, `Watch.changed()` and `Subscription.recv()` are waits, as
a channel's `recv()` is, so they are written with `await` (D16, D146):
the `await` marks where the task may stop. In a `select` arm they are
written bare, like `ch.recv()`: `ready.wait() => …`.

#### `a select arm on 'wait()' needs its receiver in a variable`

A select arm on one of those waits looks at its receiver twice — once to
wait, once to take what it waited for when the arm wins — so the receiver
must be a variable or a field, not a call's result: `val e = events()`
before the `select`, then `e.wait() => …` (D146).

#### `'scope(on: …)' takes an Executor, not 'i64'` / `'scope(…)' takes 'on:', the executor its tasks run on (D143), and 'workers:'`

`scope` and `gather` take two named arguments, each at most once: `on:`,
the executor their children run on (D143), and `workers:`, how many of
them run at once (D156): `scope(on: cpu, workers: 4) { … }`. Make an
executor with `try Executor.pool(threads: n, name: "cpu")` or
`try Executor.thread(name: "gl")`, and hold it in a `with`.

#### `MemoryOrder.Release is not an order for a load; use Relaxed, Acquire or SeqCst`

An `Atomic` operation's `order:` must be one the operation can take
(D144): a load cannot release (nothing it writes to publish), a store
cannot acquire (nothing it reads). A failed `compareAndSet` or
`compareExchange` is a load, so its `failure:` is `Relaxed`, `Acquire` or
`SeqCst`, and no stronger than `order:`. Leave `order:` out for `SeqCst`,
which is always right; the weaker orders are explained in
[chapter 13](../13-memory-and-ffi.md#atomics-and-memory-orders).

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

#### `… has no value to name`

A send, `sleep`, `wait()` or `changed()` arm waits for nothing it could
give (D108, D169): when a send arm wins, `v` is in the channel. Drop the
`val x =`.

#### `a 'race' with arms is a 'select' now` / `a select arm names its value 'val x = …'` / `a select arm says what happens when it wins` / `a select arm does not take 'await'`

The spellings D169 replaced. A `race` with arms is a `select`: `race {
val m = ch.recv() => … }` is `select { val m = ch.recv() => … }`, and
`race { }` now runs tasks and gives the first to finish. An arm names its
value before the source — `ch.recv() as msg => …` (D168) is
`val msg = ch.recv() => …` — and every arm has a `=> body`: a bare arm
`ch.recv()` is `val v = ch.recv() => v`. An arm waits by itself, so
`val r = await t => …` is `val r = t => …`. Each error carries the fix;
`veles check --fix` rewrites a whole program.

#### `'awaitFirst' is now a 'race'` / `this 'race' starts no task` / `the tasks of a 'race' give one type`

`awaitFirst(async a(), async b())` (D168) is written `race { async a();
async b() }` (D169); the fix writes it. A `race` races the tasks its body
starts, so its body starts at least one (`async f()` or `async { … }`),
and they all give the same type — the race's value is one of theirs.

#### `this lambda cannot cross a task boundary: it captures 'n', a 'var'` (in `async { }`)

An `async { }` block runs in another task, so it may use only `val`s of
Sendable types from outside (D35). Share changing state through an
`Atomic` or a `Mutex`, or have the block return the result and assign it
after `await`.

### resources

**`with` and what it closes.** `'NoClose' is not Closeable; 'with'
resources must implement Closeable`, `'conn' cannot be returned: it is
closed when its 'with' block ends`.

`with conn = open(…)` closes `conn` when its block ends — the rest of the
block is its body — and on every way out before then: `return`,
`break`/`continue`, a failed `try`, `throw`, a panic, cancellation
(D43/D100). `with (conn = open(…)) { … }` is the same with a block of its
own, for closing before the enclosing block ends.

#### `this 'close()' suspends, so it says so: 'fun close() suspends'` / `'Conn' cannot be a 'Closeable' object: its 'close()' suspends`

A `close()` may wait — send a goodbye, a `ROLLBACK`, the last telemetry —
when it is declared `fun close() suspends` (D147); then every `with` on the
type waits at its end. The declaration is required so that the cost is
visible where the type is defined. Such a type cannot be boxed as a
`Closeable` (or a trait that includes it): a method table holds one plain
`close`. Keep the concrete type, or take a type parameter `T: Closeable`,
whose instances follow the type.

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

#### `unknown annotation '@Min'`, `'@Min' stands on field, and cannot stand on …` (D174)

`@Name` with a capital is an annotation a library or your program declares
with `annotation Name(params) on field`; the lowercase names are the
compiler's. The error says what is missing: the declaration or an import
(`use validate { Min }`), or the places the annotation was declared `on` —
`field`, `struct`, `fun`, `trait`, `enum`, `enumMember`, `variant`. Its
arguments are the constructor's, and constants. A second `@Min` on one
declaration is an error.

#### `nothing reads '@Min' here` (D174)

An annotation declared `for Validate` is read by that trait's reflection; the
type it stands on does not implement `Validate`, so nothing will look at it.
Add `implement Validate` to the type, or remove the annotation.

#### `'f' stands for a field of the type being walked`, `'break' cannot leave an inline loop`, `'has' reads annotations …` (D174)

`inline loop (f in T.fields)` is unrolled while the program is compiled, one
copy of the body per field; `f` is not a value but a handle with members
(`f.name`, `f.Type`, `f.get(x)`, `f.has<A>()`, `f.attr<A>()`), and the loop has
no `break`, `continue` or label. `has` and `attr` on a `FieldInfo` value work
in a `const fun` only, and a function that reads annotations cannot be called
from code that runs when the program does: put the call in a constant,
`const RULES = Signup.rules()`.

#### `the wrapper of '@Retry' …`, `a decorator wraps a body …` (D176)

`annotation Retry(…) on fun = retryCall` makes a decorator: `retryCall` must be a
function in scope where the annotation is declared, taking the annotation and
the body — `(a: Retry, body: fun(): R)` — or the annotation, the arguments and
the body — `(a: Retry, args: A, body: fun(A): R)`. A decorator stands on
functions only, on one that has a body (not a test, an extern function or a
trait method without a default), and the second shape cannot wrap a function
with a variadic parameter.

#### `'@caller_location' is reserved for the standard library for now`

`@caller_location` makes a function's misuse panics point at its caller
(D88). Only std uses it so far; write the check in your own function
with a `panic` that names the argument, and the chain in a debug build
shows the caller. It cannot mark a trait implementation or an extern
function.

### tests

**Tests and the test-only vocabulary.** `'expect' is for tests`, `a test
takes no parameters and returns nothing`.

`expect`, `require` and `expectThrows` exist only inside `test "…" { }`,
a `test fun` and `*.test.vs` files; outside a test, `assert(cond, "why")`
states an invariant (D78).

**Skipping.** `'skip' takes the reason`, `'skipIf' takes a condition and the
reason`, `a test takes annotations (@Skip, @SkipIf)`, `'@Skip' stands on test,
suite`.

`skip(why)` and `skipIf(cond, why)` end a test as skipped (D178); in a `suite`
body they skip every test below. `@Skip("why")` and `@SkipIf(cond, "why")`
stand on a test or a suite and nothing else; a test or suite takes no other
attribute. The `skipIf` condition is a bool, and the annotation's a constant.

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

### regular-expressions

**A `/…/` literal the compiler refuses.** `this regular expression cannot be compiled: missing ')' (D182)`, `unterminated regular expression; a /…/ literal is on one line and ends with a slash`, `unknown regular expression flag 'q'`, `a regular expression literal needs the 'regex' module: add 'use regex'`.

`/(?<year>\d{4})-(?<month>\d\d)/i` is a regular expression (D182,
[the stdlib reference](stdlib.md)). The compiler runs the pattern through the
same parser `Regex.compile` uses, so a bad pattern is an error at the
literal, with the parser's own words, instead of a failure when the program
runs. A literal stays on one line, takes no `${…}`, and ends at the first
slash outside a `[…]` class that no backslash protects (`\/`); the flags after
it are `i`, `m`, `s`, `x` and `u`. `\1` and `(?=…)` cannot be matched in bounded
time and are refused in a literal; `Regex.compile(text, backtracking: true)` takes them,
at the price of a search that can run for ever. The file
needs `use regex`; a pattern built from text at run time is
`Regex.compile`, which throws a `RegexError` with the same message.

### lint-rules

**A lint rule written in Veles that cannot run.** `[lint] other.rule: 'other' is not a dependency of this package`, `package 'acme_lints' has no rule 'no_sleeps'; its rules are no_sleep (did you mean 'no_sleep'?)`, `package 'acme_lints' has no 'public const fun rules(): List<lint.Rule>'`, `a lint rule was shown the syntax 'X', which std/lint cannot show yet`.

A rule comes from a dependency (D177): the key in `[lint]` is the dependency's name in `[dependencies]`, a dot and the rule's `name`. The dependency lists its rules in `public const fun rules()`. A rule is a `const fun` like any other, so a panic or a run past `--const-steps` in it is reported with the chain of calls. The last message means the checker rewrote the tree into a node `std/lint` has no type for; it is a gap to report.

### manifest

**`veles.toml`.** `veles.toml:3: unknown [format] key "indnt"`.

The message names the file, the line and the keys that section accepts.
An unknown lint in `[lint]` suggests the closest name
(`unknown [lint] key "unused_imports"; did you mean "unused_import"?`); a
level other than `"allow"`, `"warn"` or `"deny"` names the three.
A dependency that names no source, two sources, a version that is not
`major.minor.patch`, a bare `"1.4.2"` where a path is expected, or a
`[workspace]` member that leaves the workspace each get a message saying
what to write instead (see _The manifest in full_ in chapter 11).

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
