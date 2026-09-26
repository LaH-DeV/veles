package sema

import "github.com/LaH-DeV/veles/ast"

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
