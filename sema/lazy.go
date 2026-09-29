package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// D90: a `lazy` parameter is a `fun(): T` that takes a plain expression at
// the call, wrapped in a lambda, so the function decides whether it runs.
// Std only for now.

// checkLazyParam validates a `lazy` parameter of d.
func (c *Checker) checkLazyParam(env *typeEnv, p ast.Param, pt types.Type, d *ast.FunDecl) {
	if env.module == nil || !env.module.Std {
		c.errorf(p.Name.Pos, "'lazy' is reserved for the standard library for now (D90)")
		return
	}
	ft, ok := pt.(*types.Func)
	if types.IsInvalid(pt) {
		return
	}
	if !ok || len(ft.Params) != 0 || ft.Effects.Suspends || ft.Effects.Throws || p.Variadic || p.Default != nil {
		c.errorf(p.Name.Pos, "a 'lazy' parameter must be a plain 'fun(): T' — no arguments, effects, default or '...' (D90)")
	}
}

// lazyPrefix is "lazy " for a lazy parameter, as hover spells it.
func lazyPrefix(p types.Param) string {
	if p.Lazy {
		return "lazy "
	}
	return ""
}

// LazyWord is lazyPrefix for the editor's signature help.
func LazyWord(p types.Param) string { return lazyPrefix(p) }

// lazyArgument wraps an argument for a lazy parameter in a lambda; a lambda
// is passed as it is.
func lazyArgument(e ast.Expr) ast.Expr {
	if _, isLambda := e.(*ast.LambdaExpr); isLambda {
		return e
	}
	return &ast.LambdaExpr{Body: e, Pos: e.Span()}
}
