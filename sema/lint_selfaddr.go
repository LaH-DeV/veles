package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

// Lint: `x = Node(child: &x)` takes the address of the very variable being
// assigned, so the new value ends up containing itself (`&x` names the
// variable, not a copy of its current value — D10). Almost always the
// author meant to box the old value: `val old = x; x = Node(child: &old)`.
// Found by the expression-calculator program (notes M2).

// selfAddress returns the `&name` expression inside value that names the
// assigned variable, or nil.
func selfAddress(name string, value ast.Expr) *ast.UnaryExpr {
	var found *ast.UnaryExpr
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		if e == nil || found != nil {
			return
		}
		switch e := e.(type) {
		case *ast.UnaryExpr:
			if e.Op == lexer.Amp {
				if n, ok := e.X.(*ast.NameExpr); ok && n.Name == name {
					found = e
					return
				}
			}
			walk(e.X)
		case *ast.BinaryExpr:
			walk(e.L)
			walk(e.R)
		case *ast.ElvisExpr:
			walk(e.L)
			walk(e.R)
		case *ast.CallExpr:
			walk(e.Fun)
			for _, a := range e.Args {
				walk(a.Value)
			}
		case *ast.MemberExpr:
			walk(e.X)
		case *ast.TupleExpr:
			for _, x := range e.Elems {
				walk(x)
			}
		case *ast.ListLit:
			for _, x := range e.Elems {
				walk(x)
			}
		case *ast.MapLit:
			for _, en := range e.Entries {
				walk(en.Key)
				walk(en.Value)
			}
		case *ast.TryExpr:
			walk(e.X)
		case *ast.IfExpr:
			walk(e.Cond)
		}
	}
	walk(value)
	return found
}

// lintSelfAddress warns on `x = ... &x ...` for a plain variable target.
func (f *fnCtx) lintSelfAddress(s *ast.AssignStmt) {
	n, ok := s.Target.(*ast.NameExpr)
	if !ok || s.Op != lexer.Assign {
		return
	}
	if addr := selfAddress(n.Name, s.Value); addr != nil {
		f.c.warnf(addr.Pos, "'&%s' is the address of the variable being assigned, so the new value would contain itself; box the old value first: 'val old = %s' and use '&old' (D10)", n.Name, n.Name)
	}
}
