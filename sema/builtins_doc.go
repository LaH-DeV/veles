package sema

import (
	"sort"
	"strings"
	"sync"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The compiler's built-in operations have no Veles source: `xs.len()`,
// `s.toInt()`, `x.sqrt()` are lowered directly by the checker and the code
// generator. This catalogue documents them so the editor can show a
// signature and a description on hover, and BuiltinStub renders it as a
// declarations-only file (in the spirit of a `.d.ts`) that go-to-definition
// lands in.

// BuiltinDoc describes one built-in method.
type BuiltinDoc struct {
	Recv string // receiver family: "string", "List", "MutableList", "Map", ... "float", "int", "enum"
	Name string
	Sig  string // parameter list and result, e.g. "(i: i64): T?"
	Doc  string
}

// builtinStatics lists the catalogued operations called on the type rather
// than on a value, as "family.name".
var builtinStatics = map[string]bool{"enum.values": true, "enum.fromValue": true, "enum.parse": true, "Array.make": true}

// Static reports whether the operation is called on the type.
func (d BuiltinDoc) Static() bool { return builtinStatics[d.Recv+"."+d.Name] }

// builtinFamilies orders the receiver families in the stub, with the
// declaration header each is rendered under.
var builtinFamilies = []struct{ family, header, doc string }{
	{"string", "struct string", "Immutable UTF-8 text. Concatenate with `+`, build with interpolation `\"...$x...\"`."},
	{"List", "struct List<T>", "An immutable list. `[a, b]` literals; `xs.at(i)` is a `T?`, or a `T` where the index is known to be in range (D62)."},
	{"MutableList", "struct MutableList<T>", "A growable list: every `List` method, plus mutation. `mut [a, b]` literals (D25); `.toList()` copies it into a `List` (D63)."},
	{"Array", "struct Array<T, const N: i64>", "`N` elements stored inline — in a local, a field, another array, a C struct — with no allocation, and a value: assignment and passing copy it (D121). The read-only `List` methods work on it, over a copy."},
	{"Map", "struct Map<K, V>", "An immutable hash map. `[k: v]` literals; `m.get(k)` is a `V?`."},
	{"MutableMap", "struct MutableMap<K, V> : Map<K, V>", "A hash map that can be changed in place."},
	{"Set", "struct Set<T>", "An immutable hash set."},
	{"MutableSet", "struct MutableSet<T> : Set<T>", "A hash set that can be changed in place."},
	{"Channel", "struct Channel<T>", "A bounded channel between tasks (D16). `Channel<T>(capacity: n)` creates one."},
	{"Task", "struct Task<T>", "A handle to a running child of a `scope`, from `async f()` — or of a block, from `with t = async f()`, which cancels it when the block ends (D100); `await t` is its result, `t.cancel()` stops it."},
	{"Range", "struct Range<T>", "`lo..hi` (inclusive) or `lo..<hi` (exclusive); iterate with `loop (i in r)`."},
	{"float", "struct f64", "Floating-point numbers (`f64`, `f32`). Every method is a single machine instruction."},
	{"int", "struct i64", "Integers (`i8`..`i64`, `u8`..`u64`). Arithmetic panics on overflow in debug builds."},
	{"enum", "struct Enum", "Every `enum E : i64 { A = 1, B }` (D57): a closed set of named values of one integer type. `E.A` names a member, `x.value` reads its number."},
}

var builtinDocs = []BuiltinDoc{
	{"string", "len", "(): i64", "Length in bytes."},
	{"string", "isEmpty", "(): bool", "True when the length is zero."},
	{"string", "startsWith", "(prefix: string): bool", "True when the text begins with `prefix`."},
	{"string", "endsWith", "(suffix: string): bool", "True when the text ends with `suffix`."},
	{"string", "contains", "(part: string): bool", "True when `part` occurs anywhere in the text."},
	{"string", "charCount", "(): i64", "Number of Unicode code points (not bytes)."},
	{"string", "chars", "(): List<string>", "The code points, each as a one-character string."},
	{"string", "substring", "(from: i64, to: i64): string?", "The bytes in `from..<to`, or `null` when the bounds fall outside the text or cut through a code point (D19)."},
	{"string", "toInt", "(): i64?", "Parses a decimal integer, or `null` when the text is not one."},
	{"string", "byteAt", "(i: i64): u8", "The byte at index `i`; panics when out of range (D18: strings are byte-indexed)."},
	{"string", "byteAtUnchecked", "(i: i64): u8", "The byte at index `i` without the bounds check in a release build; only inside `unsafe` (D114). A debug build still checks and panics \"unchecked index out of bounds\"; in release an out-of-range `i` is undefined behaviour."},
	{"string", "bytes", "(): List<u8>", "A copy of the UTF-8 bytes; `List<u8>.decodeUtf8()` goes back."},

	{"List", "len", "(): i64", "Number of elements."},
	{"List", "isEmpty", "(): bool", "True when there are no elements."},
	{"List", "at", "(i: i64): T?", "The element at `i`, or `null` when `i` is out of range; a negative `i` counts from the end, so `xs.at(-1)` is the last element."},
	{"List", "atUnchecked", "(i: i64): T", "The element at `i` (`0 <= i < len`, no counting from the end) without the bounds check in a release build; only inside `unsafe` (D114). A debug build still checks and panics \"unchecked index out of bounds\"; in release an out-of-range `i` is undefined behaviour."},
	{"List", "first", "(): T?", "The first element, or `null` when empty."},
	{"List", "last", "(): T?", "The last element, or `null` when empty."},
	{"List", "contains", "(x: T): bool", "True when some element equals `x`."},
	{"List", "indexOf", "(x: T): i64", "Index of the first element equal to `x`, or -1."},
	{"List", "filterIs", "<V>(): List<V>", "On a list of a sealed type: the elements that are the variant `V`, typed as `V` — `shapes.filterIs<Circle>()`. What `filter(s => s is Circle)` cannot promise, the type argument does."},
	{"List", "sorted", "(): List<T>", "A sorted copy, stable, O(n log n) (elements must be numbers, strings, tuples of those, or `Comparable`)."},
	{"List", "sortedBy", "<K>(key: fun(T): K): List<T>", "A copy sorted by `key(x)`; a tuple key sorts on several fields: `sortedBy(e => (-e.size, e.name))`."},
	{"List", "reversed", "(): List<T>", "A reversed copy."},
	{"List", "join", "(sep: string): string", "The elements rendered and joined with `sep`."},
	{"List", "iter", "(): Iterator<T>", "A lazy iterator over the elements (D46)."},
	{"List", "slice", "(from: i64, to: i64): List<T>", "The elements in `from..<to`, both ends clamped to the list; empty when `from >= to`. One bulk copy."},
	{"List", "toList", "(): List<T>", "An immutable copy."},
	{"List", "toMutable", "(): MutableList<T>", "A mutable copy."},
	{"List", "decodeUtf8", "(): string?", "On a `List<u8>` only: the bytes as text, or `null` when they are not valid UTF-8 (D18)."},
	{"Array", "len", "(): i64", "The length `N`, a constant."},
	{"Array", "isEmpty", "(): bool", "True when `N` is zero."},
	{"Array", "at", "(i: i64): T?", "The element at `i`, or `null` when `i` is out of range; a negative `i` counts from the end. Where `i` is a constant in range — or the bounds facts put it there — it is a plain `T` (D121)."},
	{"Array", "atUnchecked", "(i: i64): T", "The element at `i` (`0 <= i < N`) without the bounds check in a release build; only inside `unsafe` (D114). A debug build still checks and panics."},
	{"Array", "first", "(): T", "The first element (an array of `N > 0` is never empty); `null` for `N == 0`."},
	{"Array", "last", "(): T", "The last element (an array of `N > 0` is never empty); `null` for `N == 0`."},
	{"Array", "set", "(i: i64, x: T)", "Replaces the element at `i` in the variable the method is called on, which must be a `var`; panics when `i` is out of range. A negative `i` counts from the end."},
	{"Array", "setUnchecked", "(i: i64, x: T)", "Replaces the element at `i` (`0 <= i < N`) without the bounds check in a release build; only inside `unsafe` (D114)."},
	{"Array", "indices", "(): Range<i64>", "The valid indexes, `0..<N`. In `loop (i in a.indices())` the checker knows each `i` is in range, so `a.at(i)` is a `T` (D62)."},
	{"Array", "toList", "(): List<T>", "An immutable copy as a list."},
	{"Array", "toMutable", "(): MutableList<T>", "A mutable copy as a list."},
	{"Array", "make", "(value: T)", "Called on the type: `Array<i64, 8>.make(0)` is eight copies of `value`; `Array.make(0)` where the expected type names the array."},
	{"MutableList", "push", "(x: T)", "Appends `x`."},
	{"MutableList", "addAll", "(xs: List<T>)", "Appends every element of `xs` (a bulk copy); `xs` may be the list itself."},
	{"MutableList", "reserve", "(n: i64)", "Makes room for at least `n` elements in all, so pushing up to `n` does not grow the list again. Never shrinks it, never changes the elements."},
	{"MutableList", "set", "(i: i64, x: T)", "Replaces the element at `i`; panics when `i` is out of range. A negative `i` counts from the end."},
	{"MutableList", "setUnchecked", "(i: i64, x: T)", "Replaces the element at `i` (`0 <= i < len`) without the bounds check in a release build; only inside `unsafe` (D114). A debug build still checks and panics."},
	{"MutableList", "ref", "(i: i64): (*T)?", "A pointer to the element at `i`, or `null` when `i` is out of range: `xs.ref(i)?.bump()` and `xs.ref(i)?.n = 0` change the element in place. A negative `i` counts from the end."},
	{"MutableList", "pop", "(): T?", "Removes and returns the last element, or `null` when empty."},
	{"MutableList", "clear", "()", "Removes every element."},

	{"Map", "len", "(): i64", "Number of entries."},
	{"Map", "isEmpty", "(): bool", "True when there are no entries."},
	{"Map", "get", "(key: K): V?", "The value for `key`, or `null` when there is no entry. `m.get(k)?.f = v` and `m.get(k)?.m()` reach the stored value when it is there."},
	{"Map", "getOrDefault", "(key: K, d: V): V", "The value for `key`, or `d` when there is no entry; `m.get(key) ?: d`."},
	{"Map", "containsKey", "(key: K): bool", "True when `key` has an entry."},
	{"Map", "keys", "(): List<K>", "The keys."},
	{"Map", "values", "(): List<V>", "The values."},
	{"Map", "entries", "(): List<(K, V)>", "The entries as `(key, value)` tuples."},
	{"Map", "toMap", "(): Map<K, V>", "An immutable copy."},
	{"Map", "toMutable", "(): MutableMap<K, V>", "A mutable copy."},
	{"MutableMap", "set", "(key: K, value: V)", "Inserts or replaces the entry for `key`."},
	{"MutableMap", "reserve", "(n: i64)", "Makes room for at least `n` entries in all, so inserting up to `n` neither grows the map nor rehashes it. Never shrinks it, never changes the entries (D105)."},
	{"MutableMap", "ref", "(key: K): (*V)?", "A pointer to the value stored for `key`, or `null` when there is no entry: `m.ref(k)?.bump()`, `m.ref(k)?.n += 1` change the entry in place."},
	{"MutableMap", "remove", "(key: K): V?", "Removes the entry for `key`, returning its value, or `null`."},
	{"MutableMap", "clear", "()", "Removes every entry."},

	{"Set", "len", "(): i64", "Number of elements."},
	{"Set", "isEmpty", "(): bool", "True when there are no elements."},
	{"Set", "contains", "(x: T): bool", "True when `x` is a member."},
	{"Set", "toList", "(): List<T>", "The members as a list."},
	{"Set", "toSet", "(): Set<T>", "An immutable copy."},
	{"Set", "toMutable", "(): MutableSet<T>", "A mutable copy."},
	{"Set", "union", "(other: Set<T>): Set<T>", "The members of either set."},
	{"Set", "intersect", "(other: Set<T>): Set<T>", "The members of both sets."},
	{"Set", "difference", "(other: Set<T>): Set<T>", "The members of this set that are not in `other`."},
	{"Set", "isSubsetOf", "(other: Set<T>): bool", "True when every member of this set is in `other`."},
	{"MutableSet", "add", "(x: T): bool", "Adds `x`; true when it was not already a member."},
	{"MutableSet", "reserve", "(n: i64)", "Makes room for at least `n` members in all, so adding up to `n` neither grows the set nor rehashes it. Never shrinks it, never changes the members (D105)."},
	{"MutableSet", "remove", "(x: T): bool", "Removes `x`; true when it was a member."},
	{"MutableSet", "clear", "()", "Removes every member."},

	{"Channel", "send", "(x: T) suspends", "Sends `x`, suspending while the channel is full. As a `race` arm, `ch.send(x) => …`, `x` is sent only if that arm wins (D108)."},
	{"Channel", "recv", "(): T? suspends", "Receives the next value, suspending while empty; `null` once closed and drained."},
	{"Channel", "trySend", "(x: T): bool", "Sends `x` if there is room, without waiting; `false` when the channel is full. Panics on a closed channel, as `send` does."},
	{"Channel", "tryRecv", "(): T?", "Takes the next buffered value without waiting; `null` when there is none — empty now, or closed and drained. `len()` or `recv` tells the two apart."},
	{"Channel", "close", "()", "Closes the channel: receivers drain what is buffered, then get `null`."},
	{"Channel", "closeAfter", "(n: i64)", "Closes the channel by itself once `n` more values have been sent — how several producers end a channel without coordinating. A further send panics, as on any closed channel."},
	{"Channel", "len", "(): i64", "Number of buffered values."},

	{"Task", "cancel", "()", "Asks the task to stop: it unwinds at its next suspension point, running its `with` cleanups (D43), and its scope still waits for it. Awaiting a cancelled task panics; `withTimeout` is the usual way to use this."},

	{"Range", "iter", "(): Iterator<T>", "An iterator from `lo` to `hi`."},

	{"float", "sqrt", "(): f64", "Square root. A negative input gives `NaN`, as IEEE 754 defines; test `x < 0.0` first when that is an error for you."},
	{"float", "abs", "(): f64", "Absolute value."},
	{"float", "floor", "(): f64", "Largest integral value not greater than this."},
	{"float", "ceil", "(): f64", "Smallest integral value not less than this."},
	{"float", "round", "(): f64", "Nearest integral value, halves away from zero."},
	{"float", "trunc", "(): f64", "Integral part, toward zero."},
	{"float", "pow", "(y: f64): f64", "This raised to `y`."},
	{"float", "min", "(y: f64): f64", "The smaller of this and `y`."},
	{"float", "max", "(y: f64): f64", "The larger of this and `y`."},
	{"float", "mod", "(y: f64): f64", "Euclidean modulo: the remainder in `0.0..<|y|`, so `(-7.5).mod(2.0)` is `0.5`."},
	{"float", "clamp", "(lo: f64, hi: f64): f64", "This limited to `lo..hi`."},
	{"float", "sign", "(): f64", "`-1.0`, `0.0` or `1.0` (a NaN stays NaN)."},
	{"float", "log", "(): f64", "Natural logarithm."},
	{"float", "log2", "(): f64", "Base-2 logarithm."},
	{"float", "log10", "(): f64", "Base-10 logarithm."},
	{"float", "exp", "(): f64", "`e` raised to this."},
	{"float", "sin", "(): f64", "Sine (radians)."},
	{"float", "cos", "(): f64", "Cosine (radians)."},
	{"float", "tan", "(): f64", "Tangent (radians)."},
	{"float", "atan2", "(x: f64): f64", "The angle of the point `(x, this)`: `y.atan2(x)`, in radians."},
	{"float", "hypot", "(y: f64): f64", "`sqrt(this² + y²)` without intermediate overflow."},
	{"float", "isNaN", "(): bool", "True for a NaN. NaN is an ordinary `f64` value (IEEE 754): it comes from `0.0 / 0.0`, `inf - inf`, a negative `sqrt`, and it propagates through arithmetic; `==` on it is always false, so test with this."},
	{"float", "isFinite", "(): bool", "True when neither infinite nor NaN."},
	{"float", "isSignNegative", "(): bool", "True when the sign bit is set: for negative numbers, `-0.0` and a negative NaN, which `< 0.0` cannot tell (D104)."},
	{"float", "copySign", "(sign: f64): f64", "This number's magnitude with the sign of `sign`, `-0.0` and NaNs included (D104)."},
	{"float", "isInfinite", "(): bool", "True for positive or negative infinity (not for NaN)."},
	{"int", "abs", "(): i64", "Absolute value (the type's own width; unsigned types return themselves). The minimum has no positive counterpart: `MIN.abs()` is an overflow, a panic in a debug build and `MIN` in a release one, like `-MIN` (D21)."},
	{"int", "min", "(y: i64): i64", "The smaller of this and `y`."},
	{"int", "max", "(y: i64): i64", "The larger of this and `y`."},
	{"int", "mod", "(y: i64): i64", "Euclidean modulo: the remainder in `0..<|y|`, never negative, so `(-7).mod(3)` is `2` where `-7 % 3` is `-1`. Panics when `y` is 0."},
	{"int", "clamp", "(lo: i64, hi: i64): i64", "This limited to `lo..hi`."},
	{"int", "sign", "(): i64", "`-1`, `0` or `1`."},
	{"int", "pow", "(n: i64): i64", "This raised to `n`; panics on overflow or a negative `n` (D21)."},
	{"int", "wrappingAdd", "(y: i64): i64", "Addition that wraps around on overflow instead of panicking (the `+%` operator)."},
	{"int", "wrappingSub", "(y: i64): i64", "Subtraction that wraps around on overflow."},
	{"int", "wrappingMul", "(y: i64): i64", "Multiplication that wraps around on overflow."},
	{"int", "saturatingAdd", "(y: i64): i64", "Addition that stops at the type's minimum or maximum instead of overflowing."},
	{"int", "saturatingSub", "(y: i64): i64", "Subtraction that stops at the type's minimum or maximum."},
	{"int", "saturatingMul", "(y: i64): i64", "Multiplication that stops at the type's minimum or maximum instead of overflowing."},
	{"int", "checkedAdd", "(y: i64): i64?", "The sum, or `null` when it would overflow."},
	{"int", "checkedSub", "(y: i64): i64?", "The difference, or `null` when it would overflow."},
	{"int", "checkedMul", "(y: i64): i64?", "The product, or `null` when it would overflow."},
	{"int", "countOnes", "(): i64", "Number of one bits."},
	{"int", "leadingZeros", "(): i64", "Number of zero bits above the highest one bit (the width for 0)."},
	{"int", "trailingZeros", "(): i64", "Number of zero bits below the lowest one bit (the width for 0)."},
	{"int", "rotateLeft", "(n: i64): i64", "The bits rotated left by `n` modulo the width — what leaves the top comes back at the bottom; a negative `n` rotates right (D104)."},
	{"int", "rotateRight", "(n: i64): i64", "The bits rotated right by `n` modulo the width; a negative `n` rotates left (D104)."},
	{"int", "swapBytes", "(): i64", "The bytes in reverse order — between big- and little-endian; the identity on 8-bit types (D104)."},
	{"int", "reverseBits", "(): i64", "The bits in reverse order: the lowest becomes the highest (D104)."},

	{"enum", "toString", "(): string", "The member's name as written in the declaration: `Ordering.Less.toString() == \"Less\"`; interpolation uses it too."},
	{"enum", "compareTo", "(other: Self): Ordering", "The order of the two values' numbers; `<` and `sorted()` use it."},
	{"enum", "values", "(): List<Self>", "Every member in declaration order."},
	{"enum", "fromValue", "(value: i64): Self?", "The member holding `value`, or `null` when none does; the parameter has the enum's base type."},
	{"enum", "parse", "(s: string): Self?", "The member named `s`, or `null` when none is: the inverse of `toString`."},
}

// BuiltinStubPath is the path the stub file is known by, next to the
// standard library's embedded sources.
const BuiltinStubPath = "std/builtins.vs"

var (
	stubOnce  sync.Once
	stubFile  *source.File
	stubSpans map[string]source.Span // "family.name" -> name span in the stub
)

// BuiltinStub renders the catalogue as a declarations-only Veles file and
// remembers where each method is declared in it.
func BuiltinStub() *source.File {
	stubOnce.Do(buildStub)
	return stubFile
}

func buildStub() {
	stubSpans = map[string]source.Span{}
	var sb strings.Builder
	type nameAt struct {
		key   string
		start int
		end   int
	}
	var names []nameAt
	sb.WriteString("/// The compiler's built-in operations.\n///\n/// This file is documentation only: none of these has Veles source, each is\n/// lowered directly by the compiler (a runtime call or a single instruction).\n/// Types are written for the common case; `f64` methods exist on `f32` too,\n/// and `i64` methods on every integer width.\n\n")
	for _, fam := range builtinFamilies {
		sb.WriteString("/// ")
		sb.WriteString(fam.doc)
		sb.WriteString("\n")
		sb.WriteString(fam.header)
		sb.WriteString(" {\n")
		for _, d := range builtinDocs {
			if d.Recv != fam.family {
				continue
			}
			sb.WriteString("  /// ")
			sb.WriteString(d.Doc)
			if d.Static() {
				sb.WriteString("\n  static fun ")
			} else {
				sb.WriteString("\n  fun ")
			}
			start := sb.Len()
			sb.WriteString(d.Name)
			names = append(names, nameAt{fam.family + "." + d.Name, start, sb.Len()})
			sb.WriteString(d.Sig)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}
	stubFile = source.NewFile(BuiltinStubPath, sb.String())
	for _, n := range names {
		stubSpans[n.key] = source.Span{File: stubFile, Start: n.start, End: n.end}
	}
}

// builtinFamily names the catalogue family of a receiver type.
func builtinFamily(t types.Type) string {
	switch tt := t.(type) {
	case *types.Basic:
		switch {
		case tt.Kind == types.String:
			return "string"
		case types.IsFloat(tt):
			return "float"
		case types.IsInteger(tt):
			return "int"
		}
	case *types.List:
		if tt.Mutable {
			return "MutableList"
		}
		return "List"
	case *types.Map:
		if tt.Mutable {
			return "MutableMap"
		}
		return "Map"
	case *types.Set:
		if tt.Mutable {
			return "MutableSet"
		}
		return "Set"
	case *types.Array:
		return "Array"
	case *types.Channel:
		return "Channel"
	case *types.Task:
		return "Task"
	case *types.Range:
		return "Range"
	case *types.Enum:
		return "enum"
	}
	return ""
}

// lookupBuiltinDoc finds the entry for a method on a receiver family; a
// mutable collection inherits its immutable family's methods.
func lookupBuiltinDoc(family, name string) *BuiltinDoc {
	for i := range builtinDocs {
		if builtinDocs[i].Recv == family && builtinDocs[i].Name == name {
			return &builtinDocs[i]
		}
	}
	if base := strings.TrimPrefix(family, "Mutable"); base != family {
		return lookupBuiltinDoc(base, name)
	}
	return nil
}

// refBuiltin records a reference to a built-in method for the editor: the
// hover shows the catalogue signature and description, and go-to-definition
// lands on the method's line in the stub file.
func (c *Checker) refBuiltin(span source.Span, recv types.Type, name string) {
	if c.index == nil || !span.IsValid() {
		return
	}
	family := builtinFamily(recv)
	d := lookupBuiltinDoc(family, name)
	if d == nil {
		return
	}
	BuiltinStub()
	def := stubSpans[d.Recv+"."+name]
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: "fun", Name: name,
		Detail: "fun " + recv.String() + "." + name + receiverSig(d.Sig, recv) + "  (built in)", Doc: d.Doc})
}

// BuiltinMethods lists the catalogued methods of a receiver type: those of
// its family plus, for a mutable collection, its immutable family's.
func BuiltinMethods(t types.Type) []BuiltinDoc {
	family := builtinFamily(t)
	if family == "" {
		return nil
	}
	var out []BuiltinDoc
	for _, d := range builtinDocs {
		if d.Recv == family || (strings.HasPrefix(family, "Mutable") && d.Recv == strings.TrimPrefix(family, "Mutable")) {
			out = append(out, d)
		}
	}
	return out
}

// BuiltinFamily names the catalogue family of a receiver type ("string",
// "List", "MutableMap", "int", ...), or "" for types with no built-ins.
func BuiltinFamily(t types.Type) string { return builtinFamily(t) }

// basicTypeDocs describes the primitive types for hover.
var basicTypeDocs = map[types.BasicKind]string{
	types.Bool:   "`true` or `false`. `&&` and `||` short-circuit; there is no truthiness — a condition is a `bool`.",
	types.I8:     "A signed 8-bit integer, -128 through 127.",
	types.I16:    "A signed 16-bit integer, -32768 through 32767.",
	types.I32:    "A signed 32-bit integer, -2147483648 through 2147483647.",
	types.I64:    "A signed 64-bit integer, -9223372036854775808 through 9223372036854775807. The type of an integer literal with nothing else to go by.",
	types.ISize:  "A signed integer the width of a pointer.",
	types.U8:     "An unsigned 8-bit integer (a byte), 0 through 255. `'a'` is a `u8` literal.",
	types.U16:    "An unsigned 16-bit integer, 0 through 65535.",
	types.U32:    "An unsigned 32-bit integer, 0 through 4294967295.",
	types.U64:    "An unsigned 64-bit integer, 0 through 18446744073709551615.",
	types.USize:  "An unsigned integer the width of a pointer.",
	types.F32:    "A 32-bit IEEE 754 floating-point number.",
	types.F64:    "A 64-bit IEEE 754 floating-point number. The type of a literal with a `.` or an exponent.",
	types.String: "Immutable UTF-8 text, indexed by byte (D18). Build it with interpolation `\"...$x...\"` or a `StringBuilder`; `s.chars()` walks code points.",
	types.Unit:   "The type of an expression with no value; a function without `: T` returns it.",
	types.Never:  "The type of an expression that does not finish: `return`, `break`, `throw`, `panic(...)`. It converts to every type.",
}

// basicShape spells a primitive type out the way a struct's hover does:
// every method a value of it has — the compiler's own, then those the
// prelude's `extend` blocks add — and the traits it implements.
func (c *Checker) basicShape(t *types.Basic) string {
	lines := map[string]string{}
	for _, d := range BuiltinMethods(t) {
		kw := "fun "
		if d.Static() {
			kw = "static fun "
		}
		lines[d.Name] = kw + d.Name + receiverSig(d.Sig, t)
	}
	for _, ext := range c.extends {
		if !unify(ext.Target, t, map[*types.TypeParam]types.Type{}) {
			continue
		}
		for name, m := range ext.Methods {
			if _, dup := lines[name]; dup || m.Decl == nil || !m.Decl.Pub || strings.HasPrefix(name, "$") {
				continue
			}
			lines[name] = funDecl(m)
		}
	}
	names := make([]string, 0, len(lines))
	for name := range lines {
		names = append(names, name)
	}
	sort.Strings(names)
	var sb strings.Builder
	sb.WriteString("builtin type " + t.Name + " {\n")
	for _, name := range names {
		sb.WriteString("  " + strings.TrimPrefix(lines[name], "public ") + "\n")
	}
	var impls []string
	for trait, list := range c.impls {
		for _, impl := range list {
			if types.Identical(impl.Target, t) {
				impls = append(impls, "  implement "+trait.Name+typeArgList(impl, trait))
			}
		}
	}
	sort.Strings(impls)
	for _, line := range impls {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}

// receiverSig writes a catalogue signature for a concrete receiver: the
// element type where the catalogue says `T` (`List<i64>.sorted(): List<i64>`),
// the key and value types for a map's `K` and `V`, the type itself for
// `Self`.
func receiverSig(sig string, recv types.Type) string {
	sub := map[string]string{"Self": recv.String()}
	switch r := recv.(type) {
	case *types.List:
		sub["T"] = r.Elem.String()
	case *types.Array:
		sub["T"] = r.Elem.String()
	case *types.Set:
		sub["T"] = r.Elem.String()
	case *types.Channel:
		sub["T"] = r.Elem.String()
	case *types.Range:
		sub["T"] = r.Elem.String()
	case *types.Task:
		sub["T"] = r.Result.String()
	case *types.Map:
		sub["K"] = r.Key.String()
		sub["V"] = r.Value.String()
	}
	var sb strings.Builder
	for i := 0; i < len(sig); {
		j := i
		for j < len(sig) && (sig[j] == '_' || sig[j] >= 'a' && sig[j] <= 'z' || sig[j] >= 'A' && sig[j] <= 'Z' || sig[j] >= '0' && sig[j] <= '9') {
			j++
		}
		if j == i {
			sb.WriteByte(sig[i])
			i++
			continue
		}
		word := sig[i:j]
		if s, ok := sub[word]; ok {
			word = s
		}
		sb.WriteString(word)
		i = j
	}
	return sb.String()
}

// variantCtorDocs gives the hover of the built-in variant constructors:
// the declaration, the type it belongs to, and when to write it at all.
var variantCtorDocs = map[string][3]string{
	"Ok":   {"Ok(value: T)", "sealed trait Result<T, E>", "A success. Rarely written: a `throws` function's value is its `Ok` (D4); `r is Ok` / `r.ok` smart-cast `r` to the value."},
	"Err":  {"Err(error: E)", "sealed trait Result<T, E>", "A failure. Inside a `throws` function write `throw e`; `r is Err` / `r.err` smart-cast `r` to the error."},
	"Some": {"Some(value: T)", "sealed trait Option<T>  // written T?", "A present value of a `T?`. Rarely written: a `T` converts to `T?` where one is expected (D5)."},
	"None": {"None", "sealed trait Option<T>  // written T?", "The absent value of a `T?`; spelled `null` (D5)."},
}

// panicDoc is the hover of the built-in `panic`.
const panicDoc = "Stops the task with `message` and the call's `file:line:col` (D64). For a bug — a state the program's own logic rules out — not for a failure the caller could handle: that is `throws` (D4)."
