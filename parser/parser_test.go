package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

func parse(t *testing.T, src string) (*ast.File, *source.Diagnostics) {
	t.Helper()
	diags := &source.Diagnostics{}
	f := ParseFile(source.NewFile("test.vs", src), diags)
	return f, diags
}

func TestSyntaxTourParsesCleanly(t *testing.T) {
	data, err := os.ReadFile("../examples/syntax_tour.vs")
	if err != nil {
		t.Skip("syntax tour missing")
	}
	_, diags := parse(t, string(data))
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", diags.Render())
	}
}

func TestExpressions(t *testing.T) {
	cases := map[string]string{
		"1 + 2 * 3":            "(+ 1 (* 2 3))",
		"a ?: b ?: c":          "(?: a (?: b c))",
		"x ?: 0 > 5":           "(> (?: x 0) 5)",
		"a || b && c":          "(|| a (&& b c))",
		"-x * y":               "(* (- x) y)",
		"!a == b":              "(== (! a) b)",
		"1..<n":                "(..< 1 n)",
		"a + 1..b":             "(.. (+ a 1) b)",
		"x as i64 * 2":         "(* (as x i64) 2)",
		"x is Foo && y":        "(&& (is x Foo) y)",
		"try f(x)":             "(try (call f x))",
		"async work(1, 2)":     "(async-call work 1, 2)",
		"a?.b?.c":              "(?. (?. a b) c)",
		"a.b.c(1)(2)":          "(call (call (. (. a b) c) 1) 2)",
		"xs[0][1]":             "(index (index xs 0) 1)",
		"Stack<i32>()":         "(call Stack<i32>)",
		"a < b":                "(< a b)",
		`sql"a ${x} b"`:        `(template sql (str "a " ${x} " b"))`,
		`db.sql"x"`:            `(template (. db sql) (str "x"))`,
		`try db.sql"x ${n}"`:   `(try (template (. db sql) (str "x " ${n})))`,
		`f(sql"a", 1)`:         `(call f (template sql (str "a")), 1)`,
		"a < b > c":            "(> (< a b) c)",
		"f(a < b, c > d)":      "(call f (< a b), (> c d))",
		"Map<string, i32>()":   "(call Map<string, i32>)",
		"x => x * 2":           "(lambda (x) (* x 2))",
		"(a, b) => a + b":      "(lambda (a, b) (+ a b))",
		"(x: i32): i32 => x":   "(lambda (x: i32): i32 x)",
		"() => 1":              "(lambda () 1)",
		"cond => x => x * 2":   "(lambda (cond) (lambda (x) (* x 2)))",
		"(1, 2)":               "(tuple 1 2)",
		"(1)":                  "1",
		"()":                   "(tuple)",
		"[]":                   "(list)",
		"[:]":                  "(map)",
		"[1, 2, 3]":            "(list 1 2 3)",
		`["a": 1]`:             `(map [(str "a"): 1])`,
		"mut [1]":              "(mut-list 1)",
		"mut [:]":              "(mut-map)",
		"throw e":              "(throw e)",
		"a +% b":               "(+% a b)",
		"&x":                   "(& x)",
		"*p":                   "(* p)",
		`"a$b${c + 1}"`:        `(str "a" ${b} ${(+ c 1)})`,
		`"\n\t\u{41}"`:         `(str "\n\tA")`,
		"if (a) b else c":      "(if a\n  (block\n    b)\n  else (block\n    c))",
		"await ch.recv()":      "(await (call (. ch recv)))",
		"pair.0":               "(. pair 0)",
		"f(name: 1, other: x)": "(call f name: 1, other: x)",
	}
	for src, want := range cases {
		diags := &source.Diagnostics{}
		e := ParseExprString(src, diags)
		if diags.HasErrors() {
			t.Errorf("%s: %s", src, diags.Render())
			continue
		}
		if got := ast.DumpExpr(e); got != want {
			t.Errorf("%s:\n  got  %s\n  want %s", src, got, want)
		}
	}
}

func TestSemicolonInsertion(t *testing.T) {
	src := `
fun f(): List<i32> {
  val xs = foo(
    1,
    2,
  )
  val y = xs
    .map(x => x + 1)
    .len()
  val z = a
    ?: b
  if (y > 0) {
    bar()
  }
  else {
    baz()
  }
  return xs
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	fn := f.Decls[0].(*ast.FunDecl)
	if len(fn.Body.Stmts) != 5 {
		t.Fatalf("expected 5 statements, got %d:\n%s", len(fn.Body.Stmts), ast.Dump(f))
	}
	if _, ok := fn.Body.Stmts[3].(*ast.ExprStmt).X.(*ast.IfExpr); !ok {
		t.Fatalf("expected if statement, got %T", fn.Body.Stmts[3])
	}
	if fn.Body.Stmts[3].(*ast.ExprStmt).X.(*ast.IfExpr).Else == nil {
		t.Fatalf("else on the next line was not attached")
	}
}

func TestErrorRecovery(t *testing.T) {
	src := `
fun ok1() { }
fun broken( { }
fun ok2() {
  val x =
  val y = 2
}
struct S { a: i32 b: i32 }
fun ok3() = 1
`
	f, diags := parse(t, src)
	if !diags.HasErrors() {
		t.Fatal("expected errors")
	}
	names := []string{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FunDecl); ok {
			names = append(names, fn.Name.Name)
		}
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"ok1", "ok2", "ok3"} {
		if !strings.Contains(joined, want) {
			t.Errorf("recovery lost function %s; parsed %s\n%s", want, joined, diags.Render())
		}
	}
	// Each mistake should produce a small number of messages, not a cascade.
	if diags.ErrorCount() > 6 {
		t.Errorf("too many cascaded errors (%d):\n%s", diags.ErrorCount(), diags.Render())
	}
}

func TestSpecRejections(t *testing.T) {
	cases := []struct{ src, msg string }{
		{"fun f() { val x = fun() {} }", "lambdas"},
		{"fun f() { scope { } + 1 }", "statement"},
		{"implement Foo { }", "'for'"},
		{"fun f() { async 1 }", "'async' must prefix a call"},
		{"fun f() { val x = a -> b }", "'=>'"},
	}
	for _, c := range cases {
		_, diags := parse(t, c.src)
		found := false
		for _, d := range diags.Items {
			if strings.Contains(d.Message, c.msg) {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: expected an error mentioning %q, got:\n%s", c.src, c.msg, diags.Render())
		}
	}
}

func TestDocComments(t *testing.T) {
	f, diags := parse(t, `/// Adds two numbers.
/// Second line.
fun add(a: i64, b: i64): i64 = a + b

/**
 * A point.
 */
struct Point {
  /// horizontal
  x: i64
  // not a doc comment
  y: i64
  /// distance to origin
  @inline
  fun norm(): i64 = this.x
}

/// orphaned by a blank line

fun plain() { }

//// four slashes is an ordinary comment
fun other() { }

/// A named set.
error Errs = Point
`)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	add := f.Decls[0].(*ast.FunDecl)
	if add.Doc != "Adds two numbers.\nSecond line." {
		t.Errorf("fun doc = %q", add.Doc)
	}
	pt := f.Decls[1].(*ast.StructDecl)
	if pt.Doc != "A point." {
		t.Errorf("struct doc = %q", pt.Doc)
	}
	if pt.Fields[0].Doc != "horizontal" || pt.Fields[1].Doc != "" {
		t.Errorf("field docs = %q, %q", pt.Fields[0].Doc, pt.Fields[1].Doc)
	}
	if pt.Methods[0].Doc != "distance to origin" {
		t.Errorf("method doc (through an attribute) = %q", pt.Methods[0].Doc)
	}
	if d := f.Decls[2].(*ast.FunDecl).Doc; d != "" {
		t.Errorf("doc separated by a blank line attached: %q", d)
	}
	if d := f.Decls[3].(*ast.FunDecl).Doc; d != "" {
		t.Errorf("//// attached as doc: %q", d)
	}
	if d := f.Decls[4].(*ast.ErrorAliasDecl).Doc; d != "A named set." {
		t.Errorf("error set doc = %q", d)
	}
}

func TestModuleDoc(t *testing.T) {
	f, _ := parse(t, "/// The geometry module.\n/// Points and shapes.\n\n/// Doubles n.\nfun twice(n: i64): i64 = n * 2\n")
	if f.Doc != "The geometry module.\nPoints and shapes." {
		t.Errorf("module doc = %q", f.Doc)
	}
	if d := f.Decls[0].(*ast.FunDecl).Doc; d != "Doubles n." {
		t.Errorf("fun doc after a module doc = %q", d)
	}
	// no blank line: the comment documents the declaration, not the module
	f, _ = parse(t, "/// Doubles n.\nfun twice(n: i64): i64 = n * 2\n")
	if f.Doc != "" || f.Decls[0].(*ast.FunDecl).Doc != "Doubles n." {
		t.Errorf("doc without a blank line: module=%q fun=%q", f.Doc, f.Decls[0].(*ast.FunDecl).Doc)
	}
}

// TestArmHeadsAreNotLambdas: the `=>` after a guard, a subjectless
// condition or a race source belongs to the arm, even when the head ends
// in a name; a lambda inside brackets in the head still parses.
func TestArmHeadsAreNotLambdas(t *testing.T) {
	src := `
fun f(v: i64, limit: i64, xs: List<i64>): string = when (v) {
  is i64 if v > limit => "big"
  in 0..9 if xs.any(x => x == v) => "listed"
  Nothing => "none"
  else => "small"
}
fun g(n: i64, limit: i64): string = when {
  n > limit => "over"
  else => "under"
}
fun h() {
  val w = race {
    val v = t => v
    timer => 0
  }
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(when v", "(lambda (x)", "Nothing => ", "(race"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
	if strings.Contains(dump, "(lambda (limit)") || strings.Contains(dump, "(lambda (t)") || strings.Contains(dump, "(lambda (Nothing)") {
		t.Errorf("an arm head was read as a lambda:\n%s", dump)
	}
}

// `implement Trait { }` inside a struct body is an impl for that struct with
// its type parameters (D23, v0.23); it joins the file's declarations.
func TestInlineImpl(t *testing.T) {
	src := `
struct Box<T: Show> {
  item: T
  implement Display {
    fun toString(): string = "box"
  }
  implement Iterator {
    type Item = T
    fun next(): T? = null
  }
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	var impls []*ast.ImplDecl
	for _, d := range f.Decls {
		if impl, ok := d.(*ast.ImplDecl); ok {
			impls = append(impls, impl)
		}
	}
	if len(impls) != 2 || !impls[0].Inline || len(impls[0].TypeParams) != 1 {
		t.Fatalf("expected 2 inline impls with the struct's type parameter, got:\n%s", ast.Dump(f))
	}
	if sd := f.Decls[0].(*ast.StructDecl); len(sd.Impls) != 2 || sd.Impls[0] != impls[0] {
		t.Errorf("the struct should own the same impl nodes")
	}
	if got := ast.Dump(f); !strings.Contains(got, "Box<T>") || !strings.Contains(got, "type Item = T") {
		t.Errorf("target should be Box<T>:\n%s", got)
	}
	for src, want := range map[string]string{
		"struct P { x: i64\n implement<T> Display { fun toString(): string = \"\" } }":    "uses the struct's type parameters",
		"struct P { x: i64\n implement Display for P { fun toString(): string = \"\" } }": "drop 'for'",
	} {
		_, diags := parse(t, src)
		found := false
		for _, d := range diags.Items {
			if strings.Contains(d.Message, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %q for %q, got:\n%s", want, src, diags.Render())
		}
	}
}

// An empty trait impl may omit its braces — `implement Codable` in a struct
// body, `implement Codable for geo.Point` at top level — and a field may carry
// attributes (D58).
func TestBracelessImplAndFieldAttributes(t *testing.T) {
	src := `
struct User {
  @key("user_id") id: i64
  @skip passwordHash: string = ""
  implement Codable
  implement Comparable
  name: string
}
implement Codable for geo.Point
implement Display for User { fun toString(): string = "u" }
extend User { fun hello(): string = "hi" }
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	sd := f.Decls[0].(*ast.StructDecl)
	if len(sd.Fields) != 3 || len(sd.Fields[0].Attrs) != 1 || sd.Fields[0].Attrs[0].Name.Name != "key" || len(sd.Fields[1].Attrs) != 1 || len(sd.Fields[2].Attrs) != 0 {
		t.Fatalf("field attributes not kept:\n%s", ast.Dump(f))
	}
	var braceless, braced int
	for _, d := range f.Decls {
		if impl, ok := d.(*ast.ImplDecl); ok {
			if impl.Braceless {
				braceless++
				if len(impl.Methods) != 0 {
					t.Errorf("a braceless impl has no methods")
				}
			} else {
				braced++
			}
		}
	}
	if braceless != 3 || braced != 2 {
		t.Fatalf("expected 3 braceless and 2 braced impls, got %d and %d:\n%s", braceless, braced, ast.Dump(f))
	}
	if _, diags := parse(t, "extend User\nfun f() { }"); !diags.HasErrors() {
		t.Errorf("an extend block always has braces")
	}
}

// `static fun` is a modifier in struct, trait, impl and extend bodies;
// `Name<T>.f(...)` names a generic type as a static call target.
func TestStaticFun(t *testing.T) {
	src := `
struct S<T> {
  static fun of(x: T): S<T> = S<T>()
  public static fun z(): i64 = 0
}
trait P { static fun parse(s: string): Self? }
extend S<i64> { static fun one(): S<i64> = S<i64>.of(1) }
fun main() {
  val a = S<i64>.of(1)
  val b = a < c
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(fun static of", "(fun public static z", "(fun static parse", "(call (. S<i64> of) 1)", "(< a c)"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}

// A line ending in `>>` closes two generic lists at once (`List<List<T>>`)
// and must terminate the statement like a single `>` does.
func TestSemicolonAfterDoubleGt(t *testing.T) {
	src := `
struct G {
  n:   i64
  adj: MutableList<MutableList<i64>>

  static fun mk(n: i64): G = G(n: n, adj: [])
}
fun f() {
  var t: MutableList<MutableList<i64>>
  t = []
}
`
	_, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
}

// `_ => ...` and `(_, x) => ...`: the discard as a lambda parameter, as in
// `loop (_ in xs)`.
func TestLambdaDiscardParam(t *testing.T) {
	src := `
fun f() {
  val a = MutableList<i64>.make(3, _ => 0)
  val b = xs.map((_, x) => x)
  val c = ys.fold(0, (_: i64, x) => x)
}
`
	file, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	fn := file.Decls[0].(*ast.FunDecl)
	for i, want := range [][]string{{"_"}, {"_", "x"}, {"_", "x"}} {
		decl := fn.Body.Stmts[i].(*ast.ValStmt)
		call := decl.Value.(*ast.CallExpr)
		lam := call.Args[len(call.Args)-1].Value.(*ast.LambdaExpr)
		var got []string
		for _, p := range lam.Params {
			got = append(got, p.Name.Name)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("stmt %d: params %v, want %v", i, got, want)
		}
	}
}

// An `if` without braces as a `when` arm body must not take the `else =>`
// arm on the next line as its own `else`.
func TestElseArmAfterBracelessIf(t *testing.T) {
	src := `
fun f(x: i64): bool {
  when (x) {
    0 => if (x > 0) return false
    else => return true
  }
  false
}
`
	file, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	fn := file.Decls[0].(*ast.FunDecl)
	w := fn.Body.Stmts[0].(*ast.ExprStmt).X.(*ast.WhenExpr)
	if len(w.Arms) != 2 {
		t.Fatalf("want 2 arms, got %d", len(w.Arms))
	}
	ifx := w.Arms[0].Body.(*ast.IfExpr)
	if ifx.Else != nil {
		t.Fatalf("the if in the arm body took the when's else arm")
	}
}

// A braceless body may start on the line after `if (c)`, `else` or
// `loop (c)` (Kotlin's layout); the line break does not end the statement.
func TestBodyOnNextLine(t *testing.T) {
	src := `
fun sign(x: i64): string =
  if (x > 0)
    "positive"
  else if (x < 0)
    "negative"
  else
    "zero"
fun f() {
  var n = 0
  loop (n < 3)
    n += 1
}
`
	file, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	sign := file.Decls[0].(*ast.FunDecl)
	ifx := sign.ExprBody.(*ast.IfExpr)
	if ifx.Else == nil || len(ifx.Then.Stmts) != 1 {
		t.Fatalf("if on the next line parsed wrong:\n%s", ast.Dump(file))
	}
	inner := ifx.Else.Stmts[0].(*ast.ExprStmt).X.(*ast.IfExpr)
	if inner.Else == nil {
		t.Fatalf("else-if chain lost its final else")
	}
}

// `pair.0.1` is two tuple indexes: after a member dot a number is an index,
// never a float literal.
func TestNestedTupleIndex(t *testing.T) {
	f, diags := parse(t, "fun f(p: ((i64, i64), i64)): i64 = p.0.1 + p.1\nfun g(): f64 = 0.1 + 1.5e3")
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	if got := ast.Dump(f); !strings.Contains(got, "(. (. p 0) 1)") {
		t.Errorf("p.0.1 did not parse as two indexes:\n%s", got)
	}
}

// `enum Name : Base { A = 1, B }` (D57): members separated like fields,
// each with an optional constant; a doc comment on the enum and on a member.
func TestEnumDecl(t *testing.T) {
	src := "/// Phases.\npublic enum Phase : u8 {\n  /// Stop.\n  Red = 1\n  Amber, Green = -3\n}\nenum Plain { A }\nfun f(p: Phase): bool = p == Phase.Red\n"
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(enum public Phase : u8", "(member Red = 1)", "(member Amber)", "(member Green = (- 3))", "(enum Plain", "(member A)", "(== p (. Phase Red))"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
	d := f.Decls[0].(*ast.EnumDecl)
	if d.Doc != "Phases." || d.Members[0].Doc != "Stop." {
		t.Errorf("doc comments lost: %q %q", d.Doc, d.Members[0].Doc)
	}
	_, diags = parse(t, "enum G<T> { A }\n")
	if !strings.Contains(diags.Render(), "an enum is not generic") {
		t.Errorf("generic enum accepted:\n%s", diags.Render())
	}
}

// v0.33 spellings: `implement` replaces `impl`, and a function's type
// parameters follow its name. The old forms still parse, with an error
// that names the new one, so a file is fixed in one pass.
func TestV033Spellings(t *testing.T) {
	src := "struct P { x: i64\n  impl Display { fun toString(): string = \"p\" } }\nimpl Show for P { }\nfun <T: Display> show(x: T): string = \"$x\"\nfun ok<T>(x: T): T = x\n"
	f, diags := parse(t, src)
	var msgs []string
	for _, d := range diags.Items {
		msgs = append(msgs, d.Message)
	}
	joined := strings.Join(msgs, "\n")
	if strings.Count(joined, "'impl' is spelled 'implement'") != 2 || !strings.Contains(joined, "write 'fun show<...>(...)'") {
		t.Fatalf("expected the two spelling errors, got:\n%s", diags.Render())
	}
	impls := 0
	for _, d := range f.Decls {
		if _, ok := d.(*ast.ImplDecl); ok {
			impls++
		}
	}
	if impls != 2 {
		t.Errorf("both impls should still be in the tree, got %d", impls)
	}
	if fn := f.Decls[len(f.Decls)-2].(*ast.FunDecl); fn.Name.Name != "show" || len(fn.TypeParams) != 1 {
		t.Errorf("the old generic form should still parse: %s", ast.Dump(f))
	}
}

// v0.40 (D65): the receiver is `this`. The old `self` still parses as the
// receiver — in code and inside an interpolation — with an error whose fix
// writes `this` over exactly the four bytes.
func TestV040ThisSpelling(t *testing.T) {
	src := "struct P { x: i64\n  fun a(): i64 = self.x\n  fun b(): string = \"${self.x} $self\"\n  fun c(): i64 = this.x\n}\n"
	f, diags := parse(t, src)
	var fixes int
	for _, d := range diags.Items {
		if !strings.Contains(d.Message, "the receiver is spelled 'this'") {
			t.Errorf("unexpected diagnostic: %s", d.Message)
			continue
		}
		if d.Fix == nil || len(d.Fix.Edits) != 1 || d.Fix.Edits[0].NewText != "this" {
			t.Errorf("no fix on %s", d.Message)
			continue
		}
		e := d.Fix.Edits[0].Span
		if got := src[e.Start:e.End]; got != "self" {
			t.Errorf("fix covers %q, want \"self\"", got)
		}
		fixes++
	}
	if fixes != 3 {
		t.Errorf("want 3 fixes, got %d:\n%s", fixes, diags.Render())
	}
	if !strings.Contains(ast.Dump(f), "this") {
		t.Errorf("the receiver should dump as 'this':\n%s", ast.Dump(f))
	}
}

// D73: `init` may take parameters; a field named `init` still parses.
func TestInitParams(t *testing.T) {
	src := "struct M<T> {\n  init: i64 = 0\n  private cell: *T\n  init(value: T, n: i64 = 1) {\n    this.cell = &value\n  }\n}\n"
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	d := f.Decls[0].(*ast.StructDecl)
	if len(d.InitParams) != 2 || d.InitParams[0].Name.Name != "value" || len(d.Fields) != 2 {
		t.Fatalf("init params %v, fields %d", d.InitParams, len(d.Fields))
	}
	if dump := ast.Dump(f); !strings.Contains(dump, "(init (value: T, n: i64 = 1) ") {
		t.Errorf("dump lacks the parameters:\n%s", dump)
	}
	if _, diags = parse(t, "struct M {\n  init(value) { }\n}\n"); !strings.Contains(diags.Render(), "an 'init' parameter needs a type") {
		t.Errorf("untyped init parameter accepted:\n%s", diags.Render())
	}
}

// Kotlin's `->` in a `when` or `race` arm is one error with a fix per
// arrow, and the arms after it parse normally.
func TestThinArrowInArms(t *testing.T) {
	src := "fun f(x: i64): string = when (x) {\n  1 -> \"one\"\n  2 => \"two\"\n  else -> \"many\"\n}\n"
	_, diags := parse(t, src)
	if len(diags.Items) != 2 {
		t.Fatalf("want one error per '->':\n%s", diags.Render())
	}
	fixed := src
	for i := len(diags.Items) - 1; i >= 0; i-- {
		d := diags.Items[i]
		if d.Fix == nil || len(d.Fix.Edits) != 1 || d.Fix.Edits[0].NewText != "=>" {
			t.Fatalf("no fix: %+v", d)
		}
		e := d.Fix.Edits[0]
		fixed = fixed[:e.Span.Start] + e.NewText + fixed[e.Span.End:]
	}
	if _, again := parse(t, fixed); again.HasErrors() {
		t.Errorf("fixed source still fails:\n%s\n%s", fixed, again.Render())
	}
}

// `test "name" { }` and `test fun` (D78). `test` is contextual: a function,
// a value or a call named `test` still parses as before.
func TestTestDecl(t *testing.T) {
	src := "/// doc\ntest \"adds \\\"two\\\" numbers\" {\n  expect(1 + 1 == 2)\n}\ntest fun helper(x: i64) { }\nfun test(x: i64) { }\nval test2 = 1\nfun main() {\n  test(1)\n  val test = 2\n}\n"
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	d, ok := f.Decls[0].(*ast.TestDecl)
	if !ok || d.Name != "adds \"two\" numbers" || d.Doc != "doc" || d.Body == nil {
		t.Fatalf("test decl: %#v", f.Decls[0])
	}
	if fn := f.Decls[1].(*ast.FunDecl); !fn.Test || fn.Name.Name != "helper" {
		t.Errorf("test fun: %#v", fn)
	}
	if fn := f.Decls[2].(*ast.FunDecl); fn.Test || fn.Name.Name != "test" {
		t.Errorf("fun test: %#v", fn)
	}
	dump := ast.Dump(f)
	if !strings.Contains(dump, `(test "adds \"two\" numbers"`) || !strings.Contains(dump, "(fun test helper(x: i64)") {
		t.Errorf("dump:\n%s", dump)
	}
	_, diags = parse(t, "test \"n is ${n}\" { }\n")
	if !strings.Contains(diags.Render(), "a test's name is fixed text") {
		t.Errorf("interpolated name: %s", diags.Render())
	}
	_, diags = parse(t, "test \"\" { }\n")
	if !strings.Contains(diags.Render(), "a test needs a name") {
		t.Errorf("empty name: %s", diags.Render())
	}
}

// `suite "name" { }` (D78) holds tests, suites and `test fun` helpers.
func TestSuiteDecl(t *testing.T) {
	f, diags := parse(t, "suite \"router\" {\n  test fun h() { }\n  test \"routes\" { }\n  suite \"auth\" {\n    test \"rejects\" { }\n  }\n}\nfun suite() { }\n")
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	s, ok := f.Decls[0].(*ast.SuiteDecl)
	if !ok || s.Name != "router" || len(s.Decls) != 3 {
		t.Fatalf("suite: %#v", f.Decls[0])
	}
	if dump := ast.Dump(f); !strings.Contains(dump, `(suite "router"`) || !strings.Contains(dump, `(suite "auth"`) {
		t.Errorf("dump:\n%s", dump)
	}
	_, diags = parse(t, "suite \"a\" {\n  fun f() { }\n  val v = 1\n}\n")
	if out := diags.Render(); !strings.Contains(out, "a helper is 'test fun f'") || !strings.Contains(out, "a suite holds tests") {
		t.Errorf("members: %s", out)
	}
}

// D100: `with x = e` is a statement of its own, kept flat in its block — the
// checker reads the statements after it as its body — while `with (` is
// the block form.
func TestWithStatement(t *testing.T) {
	src := `
fun f() {
  with conn = try net.connect(host, port)
  with t = async serve(conn)
  with (g = open()) { g.read() }
  conn.write(x)
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	stmts := f.Decls[0].(*ast.FunDecl).Body.Stmts
	if len(stmts) != 4 {
		t.Fatalf("expected 4 statements, got %d:\n%s", len(stmts), ast.Dump(f))
	}
	if w, ok := stmts[0].(*ast.WithStmt); !ok || w.Binding.Name.Name != "conn" {
		t.Errorf("statement 0 is %T, want a WithStmt binding conn", stmts[0])
	}
	if w, ok := stmts[1].(*ast.WithStmt); !ok || !w.Binding.Value.(*ast.CallExpr).Async {
		t.Errorf("statement 1 is %T, want a WithStmt over an async call", stmts[1])
	}
	if _, ok := stmts[2].(*ast.ExprStmt).X.(*ast.WithExpr); !ok {
		t.Errorf("statement 2 is %T, want the block form", stmts[2])
	}
	if dump := ast.Dump(f); !strings.Contains(dump, "(with-stmt conn (try") {
		t.Errorf("dump lacks the statement:\n%s", dump)
	}
}

// D109: an item without `name =` is an expression, in both forms; `_ =`
// is a name.
func TestWithUnnamed(t *testing.T) {
	src := `
fun f() {
  with sem.acquire()
  with conn
  with _ = open()
  with (g = open(), sem.acquire()) { g.read() }
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(with-stmt (call (. sem acquire)))", "(with-stmt conn)", "(with-stmt _ (call open))", "(with (g = (call open)) ((call (. sem acquire)))"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %s:\n%s", want, dump)
		}
	}
}

// D121: a number or an expression of constants goes where a type argument
// does, and `>>` after one still closes two argument lists; `<const N: i64>`
// declares a constant parameter.
func TestConstTypeArguments(t *testing.T) {
	src := `
struct Buf<const N: i64> {
  data: Array<u8, N>
}
struct S {
  a: Array<u8, 64>
  b: Array<u8, 4 * 16>
  c: Array<u8, WIDTH>
  d: Array<u8, WIDTH * 2>
  e: MutableMap<string, Array<u8, 200>>
  f: Array<Array<i64, 3>, 2>
  g: Buf<8>
  h: Array<u8, (1 << 4)>
}
fun zeros<T, const N: i64>(): Array<T, N> = Array.make(0)
fun main() {
  val z = zeros<i64, 16>()
  val q: Array<i64, 4> = Array<i64, 4>.make(1)
}
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	sd := f.Decls[1].(*ast.StructDecl)
	if sd.Fields[1].Type.(*ast.NamedType).Args[1].(*ast.ConstType) == nil {
		t.Fatal("4 * 16 is a constant argument")
	}
	if nt := sd.Fields[2].Type.(*ast.NamedType); nt.Args[1].(*ast.NamedType).Path[0].Name != "WIDTH" {
		t.Fatal("a lone name stays a name for the checker to place")
	}
	if _, ok := sd.Fields[3].Type.(*ast.NamedType).Args[1].(*ast.ConstType); !ok {
		t.Fatal("WIDTH * 2 is a constant argument")
	}
	bd := f.Decls[0].(*ast.StructDecl)
	if tp := bd.TypeParams[0]; !tp.Const || tp.Name.Name != "N" {
		t.Fatalf("const N: %+v", tp)
	}
	if _, bad := parse(t, "fun f<const N: f64>() {}"); !bad.HasErrors() {
		t.Fatal("a constant parameter is an i64")
	}
}

// D129: a string right after a name is a template literal only with nothing
// between them; with a space it is an error that says so.
func TestTemplateLiteralNeedsNoSpace(t *testing.T) {
	_, diags := parse(t, "fun main() {\n  val q = sql \"x\"\n}\n")
	if !strings.Contains(diags.Render(), "a template literal has nothing between the name and the string: write 'sql\"…\"'") {
		t.Errorf("a spaced template literal was not explained:\n%s", diags.Render())
	}
	_, diags = parse(t, "fun main() {\n  val q = sql\"x ${1 + 2}\"\n}\n")
	if diags.HasErrors() {
		t.Errorf("a template literal with an expression failed:\n%s", diags.Render())
	}
}

// `const fun` (D113): a free function, a method, a static function and an
// extend method may be marked; `const` alone still declares a constant.
func TestConstFun(t *testing.T) {
	src := `
const KB: i64 = 1024
const fun square(n: i64): i64 = n * n
public const fun cube(n: i64): i64 = n * n * n
struct Counter {
  var n: i64
  const fun bump(by: i64) { this.n += by }
  public static const fun zero(): Counter = Counter(n: 0)
}
extend Counter { const fun twice(): i64 = this.n * 2 }
`
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatalf("parse errors:\n%s", diags.Render())
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(fun const square", "(fun public const cube", "(fun const bump", "(fun public static const zero", "(fun const twice"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}
