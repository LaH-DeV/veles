package lsp

import (
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// Hover on a keyword: what the word does, the shape it is written in, and
// where the documentation explains it. A contextual word (`inline`,
// `annotation`, `on`, …) is explained only where it is that word, never on
// a name spelled the same.

const docsBase = "https://github.com/LaH-DeV/veles/blob/main/docs/documentation/"

type keywordDoc struct {
	what    string // a sentence or two
	example string // Veles, one or a few lines
	doc     string // chapter file and anchor under docsBase
}

var keywordDocs = map[string]keywordDoc{
	"fun":       {"Declares a function. A one-expression body is written `=> expr`; a block's last expression is its value.", "fun area(w: f64, h: f64): f64 => w * h", "03-functions-and-control-flow.md#declaring-functions"},
	"val":       {"Binds a name that is never reassigned. `val x = e else { … }` takes the value or leaves (let-else).", "val n = 5\nval [first, ..] = xs else return", "02-values-and-strings.md"},
	"var":       {"Binds a name that may be reassigned; on a field, the field may be assigned after construction.", "var total = 0\ntotal += 1", "02-values-and-strings.md"},
	"const":     {"A value computed while compiling (D113); `const fun` is a function the compiler can run for one.", "const LIMIT: i64 = 4 * 1024\nconst fun kib(n: i64): i64 => n * 1024", "02-values-and-strings.md"},
	"if":        {"A condition, and an expression: both branches give its value. `if (val x = e)` binds `x` when `e` is not null (D95).", "val sign = if (n < 0) -1 else 1\nif (val user = find(id)) greet(user)", "03-functions-and-control-flow.md#if-is-an-expression"},
	"else":      {"The other branch of an `if`, the fallback arm of a `when`, or what a let-else does when there is no value.", "val sign = if (n < 0) \"-\" else \"+\"\nval user = find(id) else return", "03-functions-and-control-flow.md#if-is-an-expression"},
	"loop":      {"The one loop keyword: `loop (x in xs)`, `loop (cond)`, or `loop { }` forever. `break` and `continue` work in all three.", "loop (i in 0..<n) total += i\nloop (queue.isNotEmpty()) step()", "03-functions-and-control-flow.md#one-loop-keyword"},
	"break":     {"Leaves the innermost loop, or the labelled one (`break outer`).", "loop (line in lines) { if (line.isEmpty()) break }", "03-functions-and-control-flow.md#one-loop-keyword"},
	"continue":  {"Starts the next round of the innermost loop, or of the labelled one.", "loop (x in xs) { if (x < 0) continue\n  use(x) }", "03-functions-and-control-flow.md#one-loop-keyword"},
	"return":    {"Leaves the function with a value; at the end of a body the last expression is returned without it.", "if (xs.isEmpty()) return 0", "03-functions-and-control-flow.md#the-last-expression-is-the-result"},
	"struct":    {"Declares a value type with fields; methods and `implement` blocks go in its body.", "struct Point {\n  x: f64\n  y: f64\n  fun len(): f64 => (x * x + y * y).sqrt()\n}", "05-structs-and-methods.md"},
	"trait":     {"Declares a set of methods types implement; a default body is shared by every implementing type.", "trait Shape { fun area(): f64 }", "08-traits-and-generics.md"},
	"implement": {"Makes a type implement a trait, inside its body (`implement Display { … }`) or apart (`implement Display for Point { … }`). An empty `implement Codable` derives the body (D58).", "implement Display for Point {\n  fun toString(): string => \"($x, $y)\"\n}", "08-traits-and-generics.md"},
	"sealed":    {"A `sealed trait` has a closed set of variants, declared in its body: `when` over it must cover them all (D12, D193).", "sealed trait Shape {\n  struct Circle { r: f64 }\n  struct Square { side: f64 }\n}", "09-sealed-types.md"},
	"public":    {"Visible to other packages (D142). Without a modifier a declaration is visible in its package.", "public fun parse(text: string): Doc", "11-modules-and-packages.md"},
	"internal":  {"Visible only in its own module (M5): another module of the same package cannot use it.", "internal fun helper(): i64 => 1", "11-modules-and-packages.md"},
	"private":   {"A member visible only inside its own type's declarations.", "private fun recount() { … }", "05-structs-and-methods.md#visibility"},
	"protected": {"`protected var`: everyone who sees the field reads it, only the type's own code changes it; on a mutable collection, others may look but not take (D87).", "public protected var count: i64 = 0", "05-structs-and-methods.md#protected-var-everyone-reads-the-type-writes"},
	"use":       {"Imports a module: `use io`, then `io.println`. `use m { A, b }` also binds names; `use m as x` renames it.", "use http { Request, Response }", "11-modules-and-packages.md"},
	"when":      {"Matches a value against patterns, one arm per line; over a sealed type or an enum it must cover every case. Without a subject it is an if-else ladder.", "when (shape) {\n  is Circle => shape.r * shape.r * 3.14\n  is Square => shape.side * shape.side\n}", "03-functions-and-control-flow.md#when-matching-values"},
	"is":        {"Tests a value's variant (and narrows it in the branch), or at run time whether a trait object implements a trait (D117).", "if (shape is Circle) io.println(\"${shape.r}\")", "09-sealed-types.md"},
	"as":        {"Renames an import (`use m as x`, `use m { A as B }`), or names a `race` arm's value (`ch.recv() as msg => …`).", "use json as js", "11-modules-and-packages.md"},
	"in":        {"`x in xs` tests membership (`!in` its negation); in a loop head it binds each element.", "if (c in 'a'..'z') …\nloop (x in xs) …", "03-functions-and-control-flow.md#one-loop-keyword"},
	"throws":    {"Declares the errors a function may throw; callers handle them with `try`, `catch` or `??` (D4).", "fun parse(s: string): Config throws ParseError", "07-errors.md"},
	"throw":     {"Fails with an error value; the function must declare it with `throws`.", "if (n < 0) throw RangeError(n)", "07-errors.md"},
	"try":       {"Propagates an error to the caller: `try f()` is the value of `f()` or returns its error. One `try` covers a whole call chain (D134).", "val config = try load(path)", "07-errors.md"},
	"catch":     {"Handles an error where it happens: `f() catch (e) { … }` gives a value or leaves; `do { … } catch (e) { … }` covers several calls (D98).", "val n = parse(s) catch (e) { 0 }", "07-errors.md"},
	"do":        {"`do { … } catch (e) { … }`: several failing calls sharing one handler (D98).", "do {\n  val a = try first()\n  try second(a)\n} catch (e) { log(e) }", "07-errors.md"},
	"suspends":  {"Marks a function type (or a function) that may wait for I/O, a timer or a task. Usually inferred (D116); written on function types.", "fun retry(body: fun(): i64 suspends): i64", "12-concurrency.md#suspends-from-the-callers-side"},
	"async":     {"Starts a task: `val t = async f()` runs `f` alongside and must be awaited; `async { … }` makes a task from a block.", "val a = async fetch(\"eu\")\nval b = async fetch(\"us\")\nawait a + await b", "12-concurrency.md#one-task-alongside-val-t--async-f"},
	"await":     {"Waits for a task's value, or for a primitive that always waits (`await sleep(d)`); ordinary suspending calls are not marked (D16).", "val page = await task", "12-concurrency.md#one-task-alongside-val-t--async-f"},
	"scope":     {"A block whose tasks all end before it does: a failing one cancels the others (structured concurrency). `scope(workers: n)` limits them.", "scope {\n  loop (url in urls) async fetch(url)\n}", "12-concurrency.md#a-pool-of-your-own-scopeworkers-n-and-a-scopes-value"},
	"gather":    {"Like `scope`, but collects every task's outcome instead of failing fast.", "val results = gather { loop (u in urls) async fetch(u) }", "12-concurrency.md#collecting-every-outcome-gather"},
	"race":      {"Runs alternatives and takes the first to finish; the others are cancelled. One arm per line.", "race {\n  inbox.recv() as msg => handle(msg)\n  sleep(tick)         => flush()\n}", "12-concurrency.md#first-one-wins-race"},
	"with":      {"Opens a resource that is closed when the block (or the enclosing one) ends, on every path (D43).", "with file = try fs.open(path)\nval text = try file.readAll()", "15-files-and-processes.md"},
	"unsafe":    {"A block (or `unsafe fun`) where raw pointers and C calls are allowed; a `// SAFETY:` comment above says why it is sound.", "// SAFETY: the buffer outlives the call\nunsafe { c_fill(buf, n) }", "13-memory-and-ffi.md"},
	"extern":    {"Declares functions or types that live in C (D67): `extern \"C\" { fun strlen(s: *raw u8): usize }`.", "extern \"C\" {\n  fun getpid(): i32\n}", "13-memory-and-ffi.md"},
	"override":  {"Replaces a trait's default method in an implementation.", "override fun toString(): string => name", "08-traits-and-generics.md"},
	"static":    {"A function or value of the type itself, called on the type: `Point.origin()`.", "static fun origin(): Point => Point(x: 0, y: 0)", "05-structs-and-methods.md#static-functions"},
	"this":      {"The receiver inside a method; a member can also be written bare (D158).", "fun moved(dx: f64): Point => Point(x: this.x + dx, y: y)", "05-structs-and-methods.md"},
	"This":      {"The implementing type, in a trait or a type's body (D175).", "trait Parsable { static fun parse(s: string): This? }", "08-traits-and-generics.md"},
	"raw":       {"`*raw T`: a pointer the compiler does not track, for C (D69); used only in `unsafe` code.", "fun c_write(fd: i32, buf: *raw u8, n: usize): isize", "13-memory-and-ffi.md"},
	"type":      {"Names a type (`type Celsius = f64`, an alias), or in a trait declares an associated type each implementation says.", "type Grid = List<List<i64>>", "05-structs-and-methods.md#naming-a-type-type"},
	"enum":      {"A closed set of named values of one integer type (D57); `E.values()`, `E.parse(name)`, `E.fromValue(n)`.", "enum Level { Low, High }", "05-structs-and-methods.md"},
	"for":       {"In `implement Trait for Type`, the type implemented; in `annotation … for Trait`, the trait that reads the annotation (D174).", "implement Display for Point { … }\nannotation Min(value: i64) on field for Validate", "08-traits-and-generics.md"},
	"null":      {"No value: the only value a `T?` has besides a `T`. `?:`, `?.` and `if (val x = …)` handle it (D5).", "val name = user?.name ?: \"anonymous\"", "06-nullable-types.md"},
}

// contextual words: a keyword only where the predicate says so (the token
// after it, the one before it, the line it is on)
type contextualDoc struct {
	keywordDoc
	applies func(toks []lexer.Token, i int) bool
}

func nextIs(toks []lexer.Token, i int, kind lexer.TokenKind) bool {
	return i+1 < len(toks) && toks[i+1].Kind == kind
}

// inAnnotationDecl: the token is on a line that declares an annotation.
func inAnnotationDecl(toks []lexer.Token, i int) bool {
	for j := i - 1; j >= 0 && toks[j].Kind != lexer.Semi; j-- {
		if toks[j].Kind == lexer.Ident && toks[j].Text == "annotation" {
			return true
		}
	}
	return false
}

var contextualDocs = map[string][]contextualDoc{
	"inline": {
		{keywordDoc{"`inline loop (f in T.fields) { … }` is unrolled while compiling: the body is checked and compiled once per field (or member), each copy with that field's own type in `f.Type`. There is no `break` or `continue`: there are copies, not rounds (D174).", "inline loop (f in This.fields) {\n  if (val min = f.attr<Min>()) {\n    if (f.get(this) < min.value) out.push(f.name)\n  }\n}", "14-attributes-and-testing.md#reading-them"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.KwLoop) }},
		{keywordDoc{"`inline struct`: a variant of a sealed tree kept inside the value instead of boxed (D198).", "sealed trait Expr {\n  inline struct Hole { }\n  struct Add { left: *Expr; right: *Expr }\n}", "09-sealed-types.md"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.KwStruct) }},
	},
	"indirect": {
		{keywordDoc{"`indirect struct`: a variant stored behind a pointer, so a large one does not make every value of the family as big (D198).", "sealed trait Event {\n  indirect struct Upload { name: string; bytes: List<u8> }\n}", "09-sealed-types.md"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.KwStruct) }},
	},
	"annotation": {
		{keywordDoc{"Declares an annotation: typed data written `@Name(…)` on a declaration, read while compiling by the trait named after `for` (D174). `on` lists where it may stand; `= wrapper` makes one that wraps functions (D176).", "annotation Min(value: i64) on field for Validate\n\n@Min(18)\nage: i64", "14-attributes-and-testing.md#annotations-of-your-own"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.Ident) }},
	},
	"on": {
		{keywordDoc{"Where an annotation may stand: `field`, `struct`, `fun`, `trait`, `enum`, `enumMember`, `variant`, `test`, `suite` (D174). Anywhere else it is an error naming these.", "annotation Positional on field for Args", "14-attributes-and-testing.md#annotations-of-your-own"},
			inAnnotationDecl},
	},
	"extend": {
		{keywordDoc{"Adds functions to a type the package declares, or (as extension functions, D173) to any type for the modules that import them.", "extend string with fun shout(): string => this.toUpper()", "05-structs-and-methods.md#methods-outside-the-body-extend"},
			func(toks []lexer.Token, i int) bool {
				return i+1 < len(toks) && toks[i+1].Kind != lexer.Dot && toks[i+1].Kind != lexer.LParen
			}},
	},
	"select": {
		{keywordDoc{"Waits on several channels or tasks and runs the arm of the first that is ready, one arm per line.", "select {\n  val v = ch.recv() => use(v)\n  sleep(d)           => timeout()\n}", "12-concurrency.md#waiting-on-several-things-select"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.LBrace) }},
	},
	"test": {
		{keywordDoc{"`test fun` is a helper for tests: only tests and other helpers call it, and a build leaves it out (D78). Its failures point at the test that called it.", "test fun sample(n: i64): List<i64> => (0..<n).toList()\n\ntest \"sum of a sample\" {\n  expect(sample(3).sum() == 3)\n}", "14-attributes-and-testing.md#helpers-and-test-files"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.KwFun) }},
		{keywordDoc{"`test \"what it checks\" { … }` is a test, run by `veles test`; it may carry a label, `test emptyInput \"…\"` (D78, D170).", "test \"parse reads a number\" {\n  expect(parse(\"5\") == 5)\n}", "14-attributes-and-testing.md#writing-tests"},
			func(toks []lexer.Token, i int) bool {
				return nextIs(toks, i, lexer.String) || nextIs(toks, i, lexer.Ident)
			}},
	},
	"implements": {
		{keywordDoc{"`T implements Trait` asks, while compiling, whether a type parameter's type implements a trait: an `if` on it keeps only the branch that applies, which may then use the trait's methods (D117). At run time, on a trait object, it is `x is Trait`.", "fun show<T>(x: T): string {\n  if (T implements Display) return \"$x\"\n  \"<not printable>\"\n}", "08-traits-and-generics.md#asking-about-t-t-implements-trait"},
			func(toks []lexer.Token, i int) bool {
				return i > 0 && toks[i-1].Kind == lexer.Ident && nextIs(toks, i, lexer.Ident)
			}},
	},
	"suite": {
		{keywordDoc{"Groups tests under a label or a sentence (D78, D170); `veles test` reports them together.", "suite Lexer \"tokenising\" {\n  test emptyInput { … }\n}", "14-attributes-and-testing.md#writing-tests"},
			func(toks []lexer.Token, i int) bool {
				return nextIs(toks, i, lexer.String) || nextIs(toks, i, lexer.Ident)
			}},
	},
	"error": {
		{keywordDoc{"Declares an error type: a struct a function can `throw` and declare in `throws` (D4).", "error NotFound { path: string }", "07-errors.md"},
			func(toks []lexer.Token, i int) bool { return nextIs(toks, i, lexer.Ident) }},
	},
}

// keywordHover is the hover for the keyword (or contextual word) at off,
// or nil when the token there is not one.
func keywordHover(f *source.File, off int) any {
	toks := lexer.Tokenize(f, &source.Diagnostics{})
	for i, t := range toks {
		if off < t.Span.Start || off >= t.Span.End {
			continue
		}
		var doc *keywordDoc
		if d, ok := keywordDocs[t.Text]; ok && t.Kind != lexer.Ident && t.Kind != lexer.String {
			doc = &d
		} else if t.Kind == lexer.Ident {
			for _, c := range contextualDocs[t.Text] {
				if c.applies(toks, i) {
					d := c.keywordDoc
					doc = &d
					break
				}
			}
		}
		if doc == nil {
			return nil
		}
		value := "**`" + t.Text + "`** — " + doc.what + "\n\n```veles\n" + doc.example + "\n```\n\n[Documentation](" + docsBase + doc.doc + ")"
		return map[string]any{
			"contents": map[string]any{"kind": "markdown", "value": value},
			"range":    spanToRange(t.Span),
		}
	}
	return nil
}
