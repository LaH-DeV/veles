package source

import (
	"regexp"
	"strings"
)

// A diagnostic belongs to a family (D79): a readable name that is also the
// heading of its section in docs/documentation/reference/errors.md, where
// `veles explain <family>` reads it from and an editor links to. The
// family is found from the message itself, so a message passed through
// from elsewhere is classified the same as one written in place.
//
// Families are tried in order and the first match wins, so a narrow family
// comes before a broad one that would also match (`type-mismatch` before
// `nullable`, whose words a mismatch can mention). A test in sema checks
// that every message the compiler can report belongs to a family, and one
// in docs that every family has its section.

// ErrorsDocURL is the page the families are sections of, as an editor
// opens it.
const ErrorsDocURL = "https://github.com/LaH-DeV/veles/blob/main/docs/documentation/reference/errors.md"

type family struct {
	name string
	re   *regexp.Regexp
}

// fam compiles a family's patterns. They are written against the messages'
// formats, so a `%s` in one stands for any text and a `%d` for a number.
func fam(name string, patterns ...string) family {
	re := strings.Join(patterns, "|")
	re = strings.NewReplacer("%s", ".*", "%d", `\d+`).Replace(re)
	return family{name, regexp.MustCompile(re)}
}

var families = []family{
	// compiler-internal checks a program cannot reach, and the std intrinsics
	fam("internal", `^unsupported (expression|statement|pattern|operator|unary operator|type syntax)$`,
		`^list(RawData|DecodeUtf8Range|AppendText) takes`, `^%s takes a pointer$`, `^cannot read `),
	fam("removed-syntax", `was removed`, `no longer exists`, `is spelled`, `there is no 'for'`,
		`is written 'loop`, `are written as lambdas`, `type parameters follow the function name`,
		`are not imported one by one`, `brackets are for collection literals only`, `^'=' assigns`,
		`reads better as`, `can be written '`),
	fam("tests", `\(D78\)`, `says what it checks`, `^a (test|suite)\b`, `^another test`, `\bsuites?\b`,
		`'test`, `test code`, `the test's body`, `expectThrows`, `'require'`, `'assert'`),
	fam("attributes", `^@`, `@(key|tag|skip|required)`, `attribute`, `is deprecated`, `cannot derive`,
		`\(D58\)`, `\(D51\)`),
	fam("unsafe-and-ffi", `unsafe`, `SAFETY`, `extern`, `C ABI`, `into C`, `handed to C`, `C function pointer`,
		`raw pointer`, `pointer arithmetic`, `\(D50\)`, `\(D69\)`, `^cannot dereference`, `CLayout`,
		`^unsupported ABI`, `cannot step`),
	fam("private-to-module", `is private to module`, `is not in its exports`, `is private \(M5\)`,
		`belongs to module`),
	fam("private-to-type", `is private to '`, `the field is 'protected var'`),
	fam("manifest", `^%s:%d: `, `^%s: \[`, `\[(package|format|native)\]`),
	fam("modules", `^unknown module`, `import cycle`, `^dependency`, `has no module`, `^'use' `,
		`is already imported`, `^'%s' is not a module`, `is a module, not a value`, `^no \.vs files`,
		`^module '%s' has no declaration`),
	fam("unknown-name", `^unknown (name|function|type)`, `^cannot resolve`),
	fam("type-mismatch", `^type mismatch`, `incompatible types`, `^expected a function`, `^lambda `,
		`needs a function that returns`, `cannot bind a value of type '\(\)'`, `is bound to a value of type`,
		`is declared '%s' but`, `^a throwing function cannot be passed`, `^'%s' needs a set of`),
	fam("tasks", `\basync\b`, `\bawait\b`, `suspend`, `Sendable`, `task boundary`, `^'scope'`, `'gather'`,
		`'race'`, `race arm`, `'recv\(\)'`, `^'sleep'`, `'sleep\(\)'`, `ioWait`, `^Channel`, `the channel`,
		`^Task takes`, `^'cancel'`, `shared by every task`, `cannot cross`, `\(D35\)`),
	fam("control-flow", `unreachable code`, `^'return' outside`, `^missing return`, `returns nothing`,
		`^'if' used as a value`, `never produces a value`, `outside of a loop`, `no enclosing loop`,
		`this loop`, `^cannot iterate over`, `^'next\(\)' must return`),
	fam("results-and-errors", `\btry\b`, `throw`, `unused Result`, `'\?\?'`, `'\?!'`, `error set`,
		`is not an error`, `cannot be an error`, `^'Err\(`, `^error type`, `'message' can be overridden`,
		`only a Result has`, `error union`, `'Ok\(\.\.\.\)'`, `\(D4\)`),
	fam("val-else", `'val \.\.\. else'`, `needs 'else \{`, `the 'else' can never run`),
	fam("nullable", `may be null`, `^'null'`, `'\?\.'`, `non-nullable`, `'\?:'`, `never null`, `\(D5\)`),
	fam("enums", `\benum\b`, `\(D57\)`),
	fam("generics", `not generic`, `is generic`, `type arguments?`, `type parameters?`, `^cannot infer type parameters`,
		`generic function`, `generic traits`, `associated type`, `^bound '`),
	fam("inference", `^cannot infer`),
	fam("operators", `^operator `, `cannot compare`, `cannot be compared`, `^cannot order by`, `^cannot negate`,
		`only defined for integers`, `^'~'`, `^'%'`, `shift count`, `wrapping operator`, `^cannot cast`),
	fam("numbers", `^(integer|negative|invalid float) literal`, `^literal `, `does not fit`,
		`ranges are over integers`, `range patterns need an integer`, `^index must be an integer`,
		`^cannot iterate a range`, `numeric literal`),
	fam("text", `string literal`, `character literal`, `a comment contains`, `byte literal`, `not indexable`,
		`interpolat`, `U\+`),
	fam("references", `copy of the element`, `reach the element`, `the address of`, `^'&' `, `^a map key cannot`,
		`'loop \(&x in xs\)' needs`, `'loop \(\(k, &v\) in m\)' needs`, `a pointer into an immutable`,
		`copy of the caller's`, `while 'loop`, `move or reorder`, `is a pointer to a`),
	fam("mutability", `^cannot assign`, `not assignable`, `immutable List|immutable Map|immutable Set`,
		`\bMutable(List|Map|Set)\b`, `'mut'`, `^compound assignment`, `global 'val' is a constant`,
		`never changes: copy it`, `'getOrPut'`),
	fam("traits", `does not implement`, `not a trait`, `supertrait`, `^trait '`, `conflicting impl`,
		`^implement of`, `in trait '`, `trait '%s' declares`, `'override'`, `ambiguous`, `extend<`, `trait object`,
		`already implements`, `sealed trait '%s' declares no methods`, `cannot be implemented by hand`,
		`'implement'`, `is a (static function|method) in trait`, `requires itself`, `cannot require itself`,
		`map key or set element`, `Closeable`),
	fam("when-and-patterns", `pattern`, `destructur`, `^tuple has`, `exhaustive`, `'else' is unreachable`,
		`'else' stands for`, `can never be`, `^'(Some|None)' has`, `cannot be matched`, `^'\.\.' only`,
		`bind a single name`, `^tuple destructuring`, `^every arm of a 'when'`),
	fam("sealed-types", `sealed`, `variant`, `^members of '%s' disagree`),
	fam("constructors", `^construct `, `^cannot construct`, `^missing field`, `constructor`, `'init'`,
		`^a field default cannot`, `'this' is used before`),
	fam("this", `'this'`, `'Self'`),
	fam("calls", `\btakes\b`, `argument`, `not callable`, `^'\.\.\.'`, `variadic`, `cannot be used as a value`,
		`as values`, `is a function; call it`, `is a value, not a function`, `call it on`, `^'%s' calls its function`),
	fam("members", `^no (method|static function)`, `has no (field|static|variant|member|function|parameter|element)`,
		`has only %d field`, `^a number has no`, `'%s\.%s' is a function`, `^'%s' is not (a struct|a value|a type)`,
		`is a type, not a value`, `^a type is not a value`, `is not a module or sealed trait`),
	fam("declarations", `duplicate`, `already (declared|a static|provided|defined)`, `cannot be redeclared`,
		`only allowed at module level`, `needs an initializer`, `needs a (type|body)`, `^local functions`,
		`^'static'`, `static (member|value)`, `static val`, `visibility`, `^'public'`, `^'private'`,
		`^'protected'`, `^'private protected'`, `^a field is`, `extend`, `impl`, `^'main' must`, `infinite size`, `type alias`,
		`^'const'`, `^a global cannot`, `^'%s' binding needs`, `^'suspends' must`),
	fam("unused", `never used`, `must be used`),
	fam("syntax", `escape`, `^expected`, `^unexpected`, `^unterminated`, `^empty type`),
}

// FamilyOf names the family of a diagnostic message, or "" when none
// claims it.
func FamilyOf(message string) string {
	for _, f := range families {
		if f.re.MatchString(message) {
			return f.name
		}
	}
	return ""
}

// Families lists every family name, in the order they are tried.
func Families() []string {
	out := make([]string, len(families))
	for i, f := range families {
		out[i] = f.name
	}
	return out
}
