package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D62 C: `atOrPanic`, `getOrPanic` and `refOrPanic` are gone. A read that
// can only fail through a bug says why it cannot, where it is written:
// `xs.at(i) ?: panic("the header always has three parts")` — or it is
// written so the checker can prove it (a list pattern, an index loop).
// `panic(` is then the one spelling of every panic in Veles code.

// panicReasonTODO is what the fix writes in place of the reason, for the
// author to replace.
const panicReasonTODO = "TODO: say why this cannot fail"

// orPanicReplacement maps a removed method to the checked one it becomes.
var orPanicReplacement = map[string]string{"atOrPanic": "at", "getOrPanic": "get", "refOrPanic": "ref"}

// removedOrPanic reports a call of a removed `…OrPanic` method, with a
// fix to `(x.at(i) ?: panic("…"))`. The caller goes on to lower the call
// as before, so nothing else cascades from it.
func (f *fnCtx) removedOrPanic(e *ast.CallExpr, name string) {
	checked := orPanicReplacement[name]
	m, _ := e.Fun.(*ast.MemberExpr)
	recv, arg := "", ""
	if m != nil && len(e.Args) == 1 {
		recv, arg = srcText(m.X), srcText(e.Args[0].Value)
	}
	if recv == "" || arg == "" {
		f.errorf(e.Pos, "'%s' was removed (D62): write '.%s(…) ?: panic(\"why this cannot fail\")'", name, checked)
		return
	}
	repl := "(" + recv + "." + checked + "(" + arg + ") ?: panic(\"" + panicReasonTODO + "\"))"
	f.c.errorFix(e.Pos, fixReplace("Replace with '."+checked+"(…) ?: panic(…)'", e.Pos, repl),
		"'%s' was removed (D62): say why this cannot fail — '%s.%s(%s) ?: panic(\"…\")' — or write it so the checker can see it (a list pattern, 'loop (i in %s.indices())')",
		name, recv, checked, arg, recv)
}

// D205: `codec.KeyStyle` became the prelude's `Case`, for keys and enum text
// alike; its members were renamed.
var keyStyleMembers = map[string]string{"AsWritten": "AsWritten", "SnakeCase": "Snake", "CamelCase": "Camel", "UpperSnake": "ScreamingSnake"}

// isKeyStyle reports `KeyStyle` or `codec.KeyStyle` where neither names
// anything else.
func (f *fnCtx) isKeyStyle(x ast.Expr) bool {
	switch x := x.(type) {
	case *ast.NameExpr:
		return x.Name == "KeyStyle" && f.lookup("KeyStyle") == nil
	case *ast.MemberExpr:
		n, ok := x.X.(*ast.NameExpr)
		return ok && x.Name.Name == "KeyStyle" && n.Name == "codec"
	}
	return false
}

// removedKeyStyle reports `KeyStyle.SnakeCase` (or `codec.KeyStyle…`) with a
// fix to `Case.Snake`, and checks the replacement so nothing cascades.
func (f *fnCtx) removedKeyStyle(e *ast.MemberExpr) (Expr, bool) {
	if !f.isKeyStyle(e.X) {
		return nil, false
	}
	member, known := keyStyleMembers[e.Name.Name]
	if !known {
		f.errorf(e.Pos, "'KeyStyle' was replaced by 'Case' (D205): its members are Case.AsWritten, Case.Snake, Case.Camel, Case.ScreamingSnake and more")
		return bad(), true
	}
	repl := "Case." + member
	f.c.errorFix(e.Pos, fixReplace("Replace with '"+repl+"'", e.Pos, repl), "'KeyStyle' was replaced by 'Case' (D205): write '%s'", repl)
	if ct := f.c.preludeType("Case"); ct != nil {
		return f.checkExpr(&ast.MemberExpr{X: &ast.TypeExpr{Type: &ast.ResolvedType{T: ct, Pos: e.Pos}, Pos: e.Pos}, Name: ast.Ident{Name: member, Pos: e.Name.Pos}, Pos: e.Pos}, nil), true
	}
	return bad(), true
}

// Factory functions that stood in for a constructor are gone: a `private`
// field with a default no longer keeps the implicit constructor inside its
// module (M5 v0.30, now across modules too), so `StringBuilder()` is how a
// builder is made, like any other struct. Keyed by how the old call reads
// (`module.name` or `Type.name`, the prelude's without a qualifier); the
// value is the constructor to write, the label its one argument now takes
// (D73: `mutex(0)` is `Mutex(value: 0)`), and, when the call changes in a
// way no edit can make, how it reads instead.
var removedFactories = map[string]struct {
	ctor  string
	label string
	call  string
}{
	"stringBuilder":   {"StringBuilder", "", ""},
	"http.router":     {"http.Router", "", ""},
	"Depth.of":        {"Depth", "", "Depth(limit: n)"},
	"JsonEncoder.of":  {"JsonEncoder", "", ""},
	"deque":           {"Deque", "", ""},
	"mutex":           {"Mutex", "value", ""},
	"atomic":          {"Atomic", "value", ""},
	"taskLocal":       {"TaskLocal", "fallback", ""},
	"priorityQueueBy": {"PriorityQueue", "compare", ""},
	"priorityQueue":   {"PriorityQueue", "", "PriorityQueue<T>.natural()"},
	"http.reasonOf":   {"http.Status", "", "http.Status(code: n).reason()"},
}

// removedFactory reports a call of a removed factory function, with a fix
// to the constructor when the arguments carry over; false when key names
// nothing that was removed.
func (f *fnCtx) removedFactory(key string, e *ast.CallExpr) bool {
	r, ok := removedFactories[key]
	if !ok {
		return false
	}
	if r.call != "" {
		f.errorf(e.Fun.Span(), "'%s' was removed: write '%s' (D73, D74)", key, r.call)
		return true
	}
	span := e.Fun.Span()
	if n, ok := e.Fun.(*ast.NameExpr); ok {
		span = n.Pos // `deque<i64>()`: the type arguments stay
	}
	shape := r.ctor + "(…)"
	fix := fixReplace("Replace with '"+shape+"'", span, r.ctor)
	if r.label != "" {
		shape = r.ctor + "(" + r.label + ": …)"
		fix.Title = "Replace with '" + shape + "'"
		if len(e.Args) == 1 && e.Args[0].Name == nil {
			at := e.Args[0].Value.Span()
			fix.Edits = append(fix.Edits, source.TextEdit{Span: source.Span{File: at.File, Start: at.Start, End: at.Start}, NewText: r.label + ": "})
		} else {
			fix = nil // named or several arguments: nothing to carry over mechanically
		}
	}
	f.c.errorFix(span, fix, "'%s' was removed: construct it — '%s' — every value is made by its type's constructor (D73)", key, shape)
	return true
}

// D197: the prelude's `yieldNow()` is gone; the statement `cede` says it.
// Only a name that would have meant the prelude's function is reported — a
// program's own `yieldNow` is its own. A fix is offered where the rewrite
// keeps the meaning: the call standing as a statement, as a body, or after
// `async`. A call inside a value, or one given arguments, is explained and
// left to the author: `cede` has no value, and what an argument computes
// must not vanish with it.

// yieldNowCall is x when it is `yieldNow()` naming the removed function.
func (f *fnCtx) yieldNowCall(x ast.Expr) *ast.CallExpr {
	call, ok := x.(*ast.CallExpr)
	if !ok || call.Async || len(call.Args) > 0 || len(call.TypeArgs) > 0 {
		return nil
	}
	if n, ok := call.Fun.(*ast.NameExpr); !ok || n.Name != "yieldNow" || f.lookup(n.Name) != nil {
		return nil
	}
	return call
}

// removedYieldNowStmt reports `yieldNow()` written as a statement, with the
// fix to `cede`, and checks it as `cede`, so nothing cascades.
func (f *fnCtx) removedYieldNowStmt(call *ast.CallExpr) Stmt {
	f.c.errorFix(call.Pos, fixReplace("Replace with 'cede'", call.Pos, "cede"), "'yieldNow()' was removed (D197): write the statement 'cede'")
	return f.cede(call.Pos)
}

// removedYieldNowBody reports `=> yieldNow()`, the whole body of a function
// or a lambda, with the fix rewriting replace to `{ cede }`; the body is
// checked as that block.
func (f *fnCtx) removedYieldNowBody(call *ast.CallExpr, replace source.Span) *Block {
	f.c.errorFix(call.Pos, fixReplace("Replace with '{ cede }'", replace, "{ cede }"), "'yieldNow()' was removed (D197): 'cede' is a statement, so the body is the block '{ cede }'")
	return &Block{Stmts: []Stmt{f.cede(call.Pos)}, Type: types.TUnit}
}

// removedYieldNowCall reports any other call of the removed function (the
// name resolved to nothing): `async yieldNow()` with the fix to
// `async { cede }`, and one in a value or with arguments with how to
// restructure it. False when the call is not of `yieldNow`.
func (f *fnCtx) removedYieldNowCall(e *ast.CallExpr) bool {
	if n, ok := e.Fun.(*ast.NameExpr); !ok || n.Name != "yieldNow" {
		return false
	}
	switch {
	case len(e.Args) > 0 || len(e.TypeArgs) > 0:
		f.errorf(e.Pos, "'yieldNow()' was removed (D197), and it took no arguments: write the statement 'cede' on a line of its own, keeping what the arguments compute")
	case e.Async || f.launching == e: // launch checks a copy without the flag
		f.c.errorFix(e.Pos, fixReplace("Replace with 'async { cede }'", e.Pos, "async { cede }"), "'yieldNow()' was removed (D197): a task that cedes is 'async { cede }'")
	default:
		f.errorf(e.Pos, "'yieldNow()' was removed (D197): 'cede' is a statement and has no value — write it on a line of its own, where this expression evaluated the call")
	}
	return true
}

// removedYieldNowRef reports the removed function used as a value. Where a
// `fun()` that may suspend is wanted, the fix passes a lambda that cedes.
func (f *fnCtx) removedYieldNowRef(e *ast.NameExpr, want types.Type) Expr {
	if ft, ok := want.(*types.Func); ok && len(ft.Params) == 0 && types.IsUnit(ft.Ret) && ft.Effects.Suspends {
		f.c.errorFix(e.Pos, fixReplace("Replace with '() => { cede }'", e.Pos, "() => { cede }"), "'yieldNow' was removed (D197): 'cede' is a statement, not a function — pass '() => { cede }'")
	} else {
		f.errorf(e.Pos, "'yieldNow' was removed (D197): 'cede' is a statement, not a function — where a function that may suspend is wanted, pass '() => { cede }'")
	}
	return bad()
}

// arrowThrough is the span from the `=>` before body to body's end: what a
// fix replaces to turn an expression body into a block. Just body's span
// when no arrow is found there.
func arrowThrough(body ast.Expr) source.Span {
	sp := body.Span()
	if sp.File == nil {
		return sp
	}
	i := sp.Start
	for i > 0 && strings.ContainsRune(" \t\r\n", rune(sp.File.Content[i-1])) {
		i--
	}
	if i >= 2 && sp.File.Content[i-2:i] == "=>" {
		sp.Start = i - 2
	}
	return sp
}

// typeHead is a type's bare name, without type arguments or module.
func typeHead(t types.Type) string {
	switch t := t.(type) {
	case *types.Struct:
		return t.Name
	case *types.Sealed:
		return t.Name
	case *types.Enum:
		return t.Name
	}
	return t.String()
}
