package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Lint, chosen per package (`[lint] implicit_return` in its manifest): where
// a body's last expression may be its value without `return`.
//
//   - "full", the default: anywhere.
//   - "lambda": in a lambda, braced or not, and in `fun f() => expr`; a
//     function's or method's `{ }` body ends in `return value`.
//   - "expr": only in a body without braces, `fun f() => expr` or `x => expr`;
//     every `{ }` body ends in `return value`.
//
// The blocks of an `if` or a `when` are never touched: their value is the
// branch's, and a `return` there would leave the function. Unit and Never
// values return nothing. The fix writes the `return`. Dependencies and the
// standard library follow their own manifests.

// lintImplicitReturn reports the last statement of a `{ }` body when its
// value is what the body returns and the package's setting asks for
// `return` there; lambda says whose body it is.
func (f *fnCtx) lintImplicitReturn(body *ast.Block, value Expr, lambda bool) {
	level := f.module.implicitReturn()
	if level == ImplicitFull || (lambda && level == ImplicitLambda) || len(body.Stmts) == 0 || value == nil {
		return
	}
	if t := value.Type(); types.IsUnit(t) || types.IsNever(t) || types.IsInvalid(t) {
		return
	}
	last, ok := body.Stmts[len(body.Stmts)-1].(*ast.ExprStmt)
	if !ok {
		return
	}
	at := last.X.Span()
	insert := source.Span{File: at.File, Start: at.Start, End: at.Start}
	written := "return " + srcText(last.X)
	if strings.Contains(written, "\n") || len(written) > 60 {
		written = "return …" // the value itself is under the caret
	}
	setting := "lambda"
	if level == ImplicitExpr {
		setting = "expr"
	}
	f.c.errorFix(at, fixReplace("Add 'return'", insert, "return "),
		"this package returns a '{ }' body's value with 'return' ([lint] implicit_return = %q): write '%s'", setting, written)
}

// implicitReturn is the implicit_return setting of the module's package;
// "full" for the standard library and a package without a manifest.
func (m *Module) implicitReturn() ImplicitReturn {
	if m == nil || m.Std || m.Pkg == nil || m.Pkg.Manifest == nil {
		return ImplicitFull
	}
	return m.Pkg.Manifest.Lint.ImplicitReturn
}
