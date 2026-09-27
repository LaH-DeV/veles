package sema

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Tests (D78). A test is `test "sentence" { body }`: no signature, so it
// cannot be called, takes nothing, and may throw and suspend without
// saying so — it is checked as a function with a bare `throws`, whose error
// set is inferred (D45) and whose suspension always is (D2). Test code — a
// test, a `test fun` helper, anything in a `*.test.vs` file — may use the
// test vocabulary (`expect`, `require`, ...) and call test helpers; other
// code may not.

// isTestFile reports whether a file holds only test code: `name.test.vs`.
func isTestFile(f *ast.File) bool {
	return f != nil && f.Source != nil && strings.HasSuffix(f.Source.Path, ".test.vs")
}

// dropTestFiles leaves `*.test.vs` files out of a package being built:
// they hold tests and their helpers, which a program never runs.
func dropTestFiles(pkg *Package) {
	for _, m := range pkg.Modules {
		kept := m.Files[:0]
		for _, f := range m.Files {
			if !isTestFile(f) {
				kept = append(kept, f)
			}
		}
		m.Files = kept
	}
}

// fileSuite is the suite a file's tests are in: `parser.test.vs` is the
// suite "parser"; tests in an ordinary file are in none.
func fileSuite(f *ast.File) []string {
	if !isTestFile(f) {
		return nil
	}
	return []string{strings.TrimSuffix(filepath.Base(f.Source.Path), ".test.vs")}
}

// suiteName is a test's or suite's qualified name: "router / auth / rejects
// a bad token". It is what `--filter` matches and the summary names.
func suiteName(path []string, name string) string {
	return strings.Join(append(append([]string(nil), path...), name), " / ")
}

// claimTestName reserves a qualified name in its module: two tests (or a
// test and a suite) under one name would make a report line ambiguous.
func (c *Checker) claimTestName(m *Module, qualified string, at source.Span, what string) {
	if c.testNames == nil {
		c.testNames = map[*Module]map[string]source.Span{}
	}
	names := c.testNames[m]
	if names == nil {
		names = map[string]source.Span{}
		c.testNames[m] = names
	}
	if prev, dup := names[qualified]; dup {
		c.errorf(at, "another test or suite here is named %q (at %s); a report line must say which %s it is", qualified, prev, what)
	}
	names[qualified] = at
}

// declareSuite declares a suite's tests under its name and its helpers in
// a scope of its own, between the file's and the tests' (D78).
func (c *Checker) declareSuite(m *Module, f *ast.File, d *ast.SuiteDecl, path []string, outer *Scope) {
	c.claimTestName(m, suiteName(path, d.Name), d.At, "suite")
	if outer == nil {
		outer = m.Imports[f]
		if outer == nil {
			outer = m.Scope
		}
	}
	scope := NewScope(outer)
	inner := append(append([]string(nil), path...), d.Name)
	c.suites++
	for _, x := range d.Decls {
		switch x := x.(type) {
		case *ast.TestDecl:
			c.declareTest(m, f, x, inner, scope)
		case *ast.SuiteDecl:
			c.declareSuite(m, f, x, inner, scope)
		case *ast.FunDecl:
			t := c.newTemplate(m, f, x, nil, nil)
			t.Mangled = m.prefix() + ".$suite" + itoa(c.suites) + "." + x.Name.Name
			t.SuiteScope = scope
			if c.suiteHelpers == nil {
				c.suiteHelpers = map[string]string{}
			}
			c.suiteHelpers[x.Name.Name] = suiteName(path, d.Name)
			if old := scope.Insert(&Symbol{Name: x.Name.Name, Kind: SymFunc, Module: m, Span: x.Name.Pos, Func: t}); old != nil {
				c.errorf(x.Name.Pos, "'%s' is already declared in this suite (at %s)", x.Name.Name, old.Span)
			}
		}
	}
}

// declareTest turns `test "name" { }` into a function template the runner
// calls. Its name is the qualified sentence, which the summary prints and
// `--filter` matches; its symbol is none, so nothing can call it. scope is
// its suite's, for the suite's helpers.
func (c *Checker) declareTest(m *Module, f *ast.File, d *ast.TestDecl, path []string, scope *Scope) {
	if d.Body == nil {
		return // the parser reported it
	}
	c.claimTestName(m, suiteName(path, d.Name), d.At, "test")
	fd := &ast.FunDecl{
		Doc:     d.Doc,
		Test:    true,
		Name:    ast.Ident{Name: fmt.Sprintf("$test%d", len(c.tests)), Pos: d.At},
		Effects: ast.Effects{Throws: true},
		Body:    d.Body,
		Pos:     d.Pos,
	}
	t := c.newTemplate(m, f, fd, nil, nil)
	t.Name = suiteName(path, d.Name)
	t.Suite = path
	t.SuiteScope = scope
	c.tests = append(c.tests, t)
}

// oldTestSpelling reports `@test fun name()` (before D78) with a fix that
// writes `test "name" { ... }`, the name spelled as words: `@test fun
// parsesDates() throws E { ... }` becomes `test "parses dates" { ... }`.
// A test's errors are inferred, so the `throws` goes with the signature.
func (c *Checker) oldTestSpelling(a *ast.Attribute, d *ast.FunDecl) {
	msg := fmt.Sprintf("a test is written 'test \"%s\" { ... }' (D78)", words(d.Name.Name))
	if d.Body == nil || len(d.Params) > 0 || d.Ret != nil || a.Pos.File == nil {
		c.errorf(a.Pos, "%s", msg)
		return
	}
	cut := source.Span{File: a.Pos.File, Start: a.Pos.Start, End: d.Body.Pos.Start}
	c.errorFix(a.Pos, fixReplace("Rewrite as 'test \""+words(d.Name.Name)+"\"'", cut, fmt.Sprintf("test %q ", words(d.Name.Name))), "%s", msg)
}

// words spells an identifier as the sentence a test name reads as:
// `parsesDates` → "parses dates", `rejects_empty` → "rejects empty",
// `parsesURLQuickly` → "parses URL quickly" (an acronym stays one word).
func words(name string) string {
	var parts []string
	rs := []rune(name)
	start := 0
	cut := func(end int) {
		if w := strings.Trim(string(rs[start:end]), "_"); w != "" {
			parts = append(parts, w)
		}
		start = end
	}
	for i := 1; i < len(rs); i++ {
		prev, r := rs[i-1], rs[i]
		switch {
		case r == '_':
			cut(i)
		case unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev)):
			cut(i) // parses|Dates
		case unicode.IsUpper(r) && unicode.IsUpper(prev) && i+1 < len(rs) && unicode.IsLower(rs[i+1]):
			cut(i) // URL|Quickly
		}
	}
	cut(len(rs))
	for i, w := range parts {
		if strings.ToUpper(w) != w || len([]rune(w)) == 1 {
			parts[i] = strings.ToLower(w) // an acronym keeps its capitals
		}
	}
	return strings.Join(parts, " ")
}

// traceHelperCall wraps a call from test code to a test helper (a `test
// fun`, or a function of a `*.test.vs` file) so the runtime knows the call
// site while it runs: a failure recorded inside the helper then says which
// line of the test called it, not only the helper's own line (D78).
//
//	test.enter at the call; the call; test.leave — the value passes through
func (f *fnCtx) traceHelperCall(sym *Symbol, call Expr, e *ast.CallExpr) Expr {
	if sym == nil || sym.Kind != SymFunc || !sym.Func.TestCode || !f.inTest() || e == f.launching {
		return call
	}
	span := e.Pos
	t := call.Type()
	if types.IsInvalid(t) || t == types.TNever {
		return call // a helper that never returns ends the test anyway
	}
	prev := f.newTemp(types.TI64)
	enter := &Builtin{exprBase{types.TI64}, "test.enter", nil, span}
	leave := &ExprStmt{&Builtin{exprBase{types.TUnit}, "test.leave", []Expr{ref(prev)}, span}}
	var body Expr
	if types.Identical(t, types.TUnit) {
		body = &BlockExpr{exprBase{types.TUnit}, &Block{Stmts: []Stmt{&ExprStmt{call}, leave}, Type: types.TUnit}}
	} else {
		r := f.newTemp(t)
		body = &Let{exprBase{t}, r, call, &BlockExpr{exprBase{t}, &Block{Stmts: []Stmt{leave}, Value: ref(r), Type: t}}}
	}
	return &Let{exprBase{t}, prev, enter, body}
}
