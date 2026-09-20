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
		{"impl Foo { }", "'for'"},
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
  fun norm(): i64 = self.x
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

// `impl Trait { }` inside a struct body is an impl for that struct with
// its type parameters (D23, v0.23); it joins the file's declarations.
func TestInlineImpl(t *testing.T) {
	src := `
struct Box<T: Show> {
  item: T
  impl Display {
    fun toString(): string = "box"
  }
  impl Iterator {
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
		"struct P { x: i64\n impl<T> Display { fun toString(): string = \"\" } }": "uses the struct's type parameters",
		"struct P { x: i64\n impl Display for P { fun toString(): string = \"\" } }": "drop 'for'",
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
