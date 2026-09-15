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
		".none":                "(. <nil> none)",
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
