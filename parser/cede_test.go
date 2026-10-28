package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// D197: `cede` is a statement wherever one goes, ended by a newline, `;`,
// `}`, `else` or the end of the file; a newline after it is never a
// continuation, CRLF and comments included.
func TestCedeStatement(t *testing.T) {
	cases := map[string]string{
		"fun f() {\n  cede\n}":                               "(cede)",
		"fun f() {\n  cede;\n  cede; cede\n}":                "(cede)\n    (cede)\n    (cede)",
		"fun f() {\n  cede\r\n  g()\r\n}":                    "(cede)\n    (call g)",
		"fun f() {\n  cede // why\n  g()\n}":                 "(cede)\n    (call g)",
		"fun f() {\n  cede /* why */\n  g()\n}":              "(cede)\n    (call g)",
		"fun f() { if (x) cede else g() }":                   "(if x\n      (block\n        (cede))\n      else (block\n        (call g)))",
		"fun f() { loop (x) cede }":                          "(loop (x)\n      (block\n        (cede)))",
		"fun f() { val t = async { cede } }":                 "(async-block\n      (block\n        (cede)))",
		"fun f() { val g: fun() suspends = () => { cede } }": "(lambda () (block\n      (cede)))",
	}
	for src, want := range cases {
		f, diags := parse(t, src)
		if diags.HasErrors() {
			t.Errorf("%q: %s", src, diags.Render())
			continue
		}
		if got := ast.Dump(f); !strings.Contains(got, want) {
			t.Errorf("%q:\n got %s\nwant it to contain %s", src, got, want)
		}
	}
	// the file may end right after it
	if errs := errorsOf(mustParse(t, "fun f() {\n  cede")); len(errs) != 1 || errs[0].Message != "expected '}', found end of file" {
		t.Errorf("a block left open at EOF: %v", errs)
	}
	if errs := errorsOf(mustParse(t, "fun f() {\n  cede\n}\ncede")); len(errs) != 1 || !strings.Contains(errs[0].Message, "expected a declaration") {
		t.Errorf("cede at the top level: %v", errs)
	}
	// the next line's `.x()` is not continued from `cede`
	f, diags := parse(t, "fun f() {\n  cede\n    .x()\n  g()\n}")
	if !strings.Contains(diags.Render(), "expected an expression, found '.'") || !strings.Contains(ast.Dump(f), "(call g)") {
		t.Errorf("a chain after cede: %s\n%s", diags.Render(), ast.Dump(f))
	}
}

// D197 1, 6: every form that is not the bare statement is an error that
// says what to write, one per mistake, and the statements after it parse.
func TestCedeErrors(t *testing.T) {
	cases := []struct{ src, want, fix, fixed string }{
		{"cede()", "is a statement, not a function: write 'cede' without parentheses", "Remove the parentheses", "cede"},
		{"cede(1)", "takes no arguments: write 'cede' alone", "", ""},
		{"cede x", "takes no operand, label, arguments or type arguments", "", ""},
		{"cede@outer", "takes no operand, label, arguments or type arguments", "", ""},
		{"cede<T>()", "takes no operand, label, arguments or type arguments", "", ""},
		{"cede.member", "has no value: '.' cannot follow it", "", ""},
		{"cede + 1", "has no value: '+' cannot follow it", "", ""},
		{"cede = 1", "has no value: '=' cannot follow it", "", ""},
		{"await cede", "remove 'await'", "Remove 'await'", "cede"},
		{"try cede", "cannot fail: remove 'try'", "Remove 'try'", "cede"},
		{"val t = async cede", "write 'async { cede }'", "Write 'async { cede }'", "val t = async { cede }"},
		{"val pause = cede", "has no value: write it where a statement goes", "", ""},
		{"return cede", "has no value: write it where a statement goes", "", ""},
		{"consume(cede)", "has no value: write it where a statement goes", "", ""},
		{"val f = () => cede", "write a block, '=> { cede }'", "Write a block", "val f = () => { cede }"},
		{"when (x) { 1 => cede\n else => { } }", "write a block, '=> { cede }'", "Write a block", "when (x) { 1 => { cede }\n else => { } }"},
	}
	for _, tc := range cases {
		src := "fun f() {\n  " + tc.src + "\n  after()\n}\n"
		f, diags := parse(t, src)
		errs := errorsOf(diags)
		if len(errs) != 1 || !strings.Contains(errs[0].Message, tc.want) {
			t.Errorf("%q: want one error containing %q, got:\n%s", tc.src, tc.want, diags.Render())
			continue
		}
		if !strings.Contains(ast.Dump(f), "(call after)") {
			t.Errorf("%q: the next statement was lost:\n%s", tc.src, ast.Dump(f))
		}
		checkFix(t, src, errs[0], tc.fix, "fun f() {\n  "+tc.fixed+"\n  after()\n}\n")
	}
	// an expression body
	src := "fun pause() => cede\n"
	f, diags := parse(t, src)
	errs := errorsOf(diags)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "write 'fun pause(…) { cede }'") {
		t.Fatalf("expression body: %s", diags.Render())
	}
	if !strings.Contains(ast.Dump(f), "(block\n    (cede))") {
		t.Errorf("expression body not read as the block: %s", ast.Dump(f))
	}
	checkFix(t, src, errs[0], "Write the body as a block", "fun pause() { cede }\n")
	// a name that became the keyword
	for _, src := range []string{"fun cede() {}\n", "fun f(cede: i64) {}\n", "val cede = 1\n", "fun f() {\n  loop :cede { }\n}\n"} {
		if _, diags := parse(t, src); !strings.Contains(diags.Render(), "'cede' is a statement now (D197); rename this and every use of it") {
			t.Errorf("%q: %s", src, diags.Render())
		}
	}
	// a longer name is a name
	if _, diags := parse(t, "fun ceded() { val cedeCount = 1 }\n"); diags.HasErrors() {
		t.Errorf("ceded: %s", diags.Render())
	}
}

func mustParse(t *testing.T, src string) *source.Diagnostics {
	t.Helper()
	_, diags := parse(t, src)
	return diags
}

func errorsOf(diags *source.Diagnostics) []source.Diagnostic {
	var out []source.Diagnostic
	for _, d := range diags.Items {
		if d.Severity == source.Error {
			out = append(out, d)
		}
	}
	return out
}

// checkFix applies d's fix to src and compares; title "" means no fix.
func checkFix(t *testing.T, src string, d source.Diagnostic, title, want string) {
	t.Helper()
	if title == "" {
		if d.Fix != nil {
			t.Errorf("%q: unexpected fix %q", src, d.Fix.Title)
		}
		return
	}
	if d.Fix == nil || d.Fix.Title != title {
		t.Errorf("%q: want the fix %q, got %+v", src, title, d.Fix)
		return
	}
	got := src
	for i := len(d.Fix.Edits) - 1; i >= 0; i-- {
		e := d.Fix.Edits[i]
		got = got[:e.Span.Start] + e.NewText + got[e.Span.End:]
	}
	if got != want {
		t.Errorf("%q: fixed to\n%s\nwant\n%s", src, got, want)
	}
	if _, diags := parse(t, got); diags.HasErrors() {
		t.Errorf("%q: the fixed source does not parse: %s", src, diags.Render())
	}
}
