package sema

import (
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
