package sema

import (
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
	Recv string // receiver family: "string", "List", "MutableList", "Map", ... "float", "int"
	Name string
	Sig  string // parameter list and result, e.g. "(i: i64): T?"
	Doc  string
}

// builtinFamilies orders the receiver families in the stub, with the
// declaration header each is rendered under.
var builtinFamilies = []struct{ family, header, doc string }{
	{"string", "struct string", "Immutable UTF-8 text. Concatenate with `+`, build with interpolation `\"...$x...\"`."},
	{"List", "struct List<T>", "An immutable list. `[a, b]` literals; `xs.at(i)` is a `T?`, `xs.atOrPanic(i)` a `T`."},
	{"MutableList", "struct MutableList<T> : List<T>", "A growable list: everything a `List` has, plus mutation. `mut [a, b]` literals (D25)."},
	{"Map", "struct Map<K, V>", "An immutable hash map. `[k: v]` literals; `m.get(k)` is a `V?`."},
	{"MutableMap", "struct MutableMap<K, V> : Map<K, V>", "A hash map that can be changed in place."},
	{"Set", "struct Set<T>", "An immutable hash set."},
	{"MutableSet", "struct MutableSet<T> : Set<T>", "A hash set that can be changed in place."},
	{"Channel", "struct Channel<T>", "A bounded channel between tasks (D16). `Channel<T>(capacity: n)` creates one."},
	{"Range", "struct Range<T>", "`lo..hi` (inclusive) or `lo..<hi` (exclusive); iterate with `loop (i in r)`."},
	{"float", "struct f64", "Floating-point numbers (`f64`, `f32`). Every method is a single machine instruction."},
	{"int", "struct i64", "Integers (`i8`..`i64`, `u8`..`u64`). Arithmetic panics on overflow in debug builds."},
}

var builtinDocs = []BuiltinDoc{
	{"string", "len", "(): i64", "Length in bytes."},
	{"string", "isEmpty", "(): bool", "True when the length is zero."},
	{"string", "startsWith", "(prefix: string): bool", "True when the text begins with `prefix`."},
	{"string", "endsWith", "(suffix: string): bool", "True when the text ends with `suffix`."},
	{"string", "contains", "(part: string): bool", "True when `part` occurs anywhere in the text."},
	{"string", "charCount", "(): i64", "Number of Unicode code points (not bytes)."},
	{"string", "chars", "(): List<string>", "The code points, each as a one-character string."},
	{"string", "substring", "(from: i64, to: i64): string?", "The bytes in `from..<to`, or `null` when the bounds are not valid."},
	{"string", "toInt", "(): i64?", "Parses a decimal integer, or `null` when the text is not one."},
	{"string", "byteAt", "(i: i64): u8", "The byte at index `i`; panics when out of range (D18: strings are byte-indexed)."},
	{"string", "bytes", "(): List<u8>", "A copy of the UTF-8 bytes; `List<u8>.decodeUtf8()` goes back."},

	{"List", "len", "(): i64", "Number of elements."},
	{"List", "isEmpty", "(): bool", "True when there are no elements."},
	{"List", "at", "(i: i64): T?", "The element at `i`, or `null` when `i` is out of range; a negative `i` counts from the end, so `xs.at(-1)` is the last element."},
	{"List", "atOrPanic", "(i: i64): T", "The element at `i`; panics when `i` is out of range. Names the element in place, so `xs.atOrPanic(i).bump()` mutates it where it lives (as does `xs.at(i)?.bump()`). A negative `i` counts from the end."},
	{"List", "first", "(): T?", "The first element, or `null` when empty."},
	{"List", "last", "(): T?", "The last element, or `null` when empty."},
	{"List", "contains", "(x: T): bool", "True when some element equals `x`."},
	{"List", "indexOf", "(x: T): i64", "Index of the first element equal to `x`, or -1."},
	{"List", "find", "(pred: fun(T): bool): T?", "The first element `pred` accepts, or `null`."},
	{"List", "any", "(pred: fun(T): bool): bool", "True when `pred` accepts some element."},
	{"List", "all", "(pred: fun(T): bool): bool", "True when `pred` accepts every element (true for an empty list)."},
	{"List", "map", "<U>(f: fun(T): U): List<U>", "A new list of `f` applied to each element (eager; `iter().map` is lazy, D46)."},
	{"List", "filter", "(pred: fun(T): bool): List<T>", "A new list of the elements `pred` accepts."},
	{"List", "fold", "<A>(init: A, f: fun(A, T): A): A", "Folds left: `f(f(f(init, x0), x1), x2)`."},
	{"List", "forEach", "(f: fun(T))", "Calls `f` on each element in order."},
	{"List", "sorted", "(): List<T>", "A sorted copy (elements must be `Ord`)."},
	{"List", "sortedBy", "<K>(key: fun(T): K): List<T>", "A copy sorted by `key(x)`."},
	{"List", "reversed", "(): List<T>", "A reversed copy."},
	{"List", "join", "(sep: string): string", "The elements rendered and joined with `sep`."},
	{"List", "iter", "(): Iterator<T>", "A lazy iterator over the elements (D46)."},
	{"List", "toList", "(): List<T>", "An immutable copy."},
	{"List", "toMutable", "(): MutableList<T>", "A mutable copy."},
	{"List", "decodeUtf8", "(): string?", "On a `List<u8>` only: the bytes as text, or `null` when they are not valid UTF-8 (D18)."},
	{"MutableList", "push", "(x: T)", "Appends `x`."},
	{"MutableList", "set", "(i: i64, x: T)", "Replaces the element at `i`; panics when `i` is out of range. A negative `i` counts from the end."},
	{"MutableList", "pop", "(): T?", "Removes and returns the last element, or `null` when empty."},
	{"MutableList", "clear", "()", "Removes every element."},

	{"Map", "len", "(): i64", "Number of entries."},
	{"Map", "isEmpty", "(): bool", "True when there are no entries."},
	{"Map", "get", "(key: K): V?", "The value for `key`, or `null` when there is no entry. `m.get(k)?.f = v` and `m.get(k)?.m()` reach the stored value when it is there."},
	{"Map", "getOrPanic", "(key: K): V", "The value for `key`; panics when there is no entry. Names the entry in place, so `m.getOrPanic(k).bump()` mutates the stored value (as does `m.get(k)?.bump()`)."},
	{"Map", "getOrDefault", "(key: K, d: V): V", "The value for `key`, or `d` when there is no entry; `m.get(key) ?: d`."},
	{"Map", "containsKey", "(key: K): bool", "True when `key` has an entry."},
	{"Map", "keys", "(): List<K>", "The keys."},
	{"Map", "values", "(): List<V>", "The values."},
	{"Map", "entries", "(): List<(K, V)>", "The entries as `(key, value)` tuples."},
	{"Map", "toMap", "(): Map<K, V>", "An immutable copy."},
	{"Map", "toMutable", "(): MutableMap<K, V>", "A mutable copy."},
	{"MutableMap", "set", "(key: K, value: V)", "Inserts or replaces the entry for `key`."},
	{"MutableMap", "remove", "(key: K): V?", "Removes the entry for `key`, returning its value, or `null`."},
	{"Map", "forEach", "(f: fun(K, V))", "Calls `f` on each entry."},
	{"Map", "mapValues", "<U>(f: fun(V): U): Map<K, U>", "A new map with the same keys and `f` applied to each value."},
	{"Map", "filter", "(pred: fun(K, V): bool): Map<K, V>", "A new map of the entries `pred` accepts."},
	{"MutableMap", "clear", "()", "Removes every entry."},
	{"MutableMap", "getOrPut", "(key: K, make: fun(): V): V", "The value for `key`; when absent, `make()` is stored under `key` and returned."},

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
	{"MutableSet", "remove", "(x: T): bool", "Removes `x`; true when it was a member."},
	{"MutableSet", "clear", "()", "Removes every member."},

	{"Channel", "send", "(x: T) suspends", "Sends `x`, suspending while the channel is full."},
	{"Channel", "recv", "(): T? suspends", "Receives the next value, suspending while empty; `null` once closed and drained."},
	{"Channel", "close", "()", "Closes the channel: receivers drain what is buffered, then get `null`."},
	{"Channel", "len", "(): i64", "Number of buffered values."},

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
	{"float", "isInfinite", "(): bool", "True for positive or negative infinity (not for NaN)."},
	{"int", "abs", "(): i64", "Absolute value (the type's own width; unsigned types return themselves)."},
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
	{"int", "checkedAdd", "(y: i64): i64?", "The sum, or `null` when it would overflow."},
	{"int", "checkedSub", "(y: i64): i64?", "The difference, or `null` when it would overflow."},
	{"int", "checkedMul", "(y: i64): i64?", "The product, or `null` when it would overflow."},
	{"int", "countOnes", "(): i64", "Number of one bits."},
	{"int", "leadingZeros", "(): i64", "Number of zero bits above the highest one bit (the width for 0)."},
	{"int", "trailingZeros", "(): i64", "Number of zero bits below the lowest one bit (the width for 0)."},
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
			sb.WriteString("\n  fun ")
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
	case *types.Channel:
		return "Channel"
	case *types.Range:
		return "Range"
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
		Detail: "fun " + recv.String() + "." + name + d.Sig + "  (built in)", Doc: d.Doc})
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
