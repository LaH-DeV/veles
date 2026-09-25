package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// Lint: counting by hand what the collection already knows (notes T4).
//
//	var n = 0
//	loop (x in xs) { n += 1 }
//
// is `val n = xs.len()`. Only the exact shape is matched — a `var`
// starting at 0, the loop right after it, a body that is nothing but
// `n += 1` — over a List, Set, Map or Range, whose `len()` is the same
// number without the walk; and only when `n` is never assigned again, so
// `val` is right. The loop variable is necessarily unused (the body is one
// statement that does not mention it). Anything with a condition inside
// is a `count(pred)` in disguise, but the rewrite then depends on the
// body, and this lint stays with the case it can fix exactly.
func (f *fnCtx) lintCountingLoop(stmts []ast.Stmt, i int) {
	decl, ok := stmts[i].(*ast.ValStmt)
	if !ok || decl.Kind != ast.BindVar || decl.Binding.Name == nil || decl.Binding.Type != nil || i+1 >= len(stmts) {
		return
	}
	if lit, ok := decl.Value.(*ast.IntLit); !ok || lit.Text != "0" {
		return
	}
	name := decl.Binding.Name.Name
	loop, ok := stmts[i+1].(*ast.LoopStmt)
	if !ok || loop.Var == nil || loop.Var.Ref || loop.Label != nil || loop.Body == nil || len(loop.Body.Stmts) != 1 {
		return
	}
	inc, ok := loop.Body.Stmts[0].(*ast.AssignStmt)
	if !ok || inc.Op != lexer.PlusEq {
		return
	}
	if target, ok := inc.Target.(*ast.NameExpr); !ok || target.Name != name {
		return
	}
	if one, ok := inc.Value.(*ast.IntLit); !ok || one.Text != "1" {
		return
	}
	switch types.Underlying(f.loopIters[loop]).(type) {
	case *types.List, *types.Set, *types.Map, *types.Range:
	default:
		return
	}
	for _, s := range stmts[i+2:] {
		if assignsName(s, name) {
			return
		}
	}
	iter := spanText(loop.Iter.Span())
	switch loop.Iter.(type) {
	case *ast.NameExpr, *ast.CallExpr, *ast.MemberExpr:
	default:
		iter = "(" + iter + ")"
	}
	span := decl.Pos.To(loop.Pos)
	f.warnFix(span, fixReplace("Write 'val "+name+" = "+iter+".len()'", span, "val "+name+" = "+iter+".len()"),
		"this loop counts the elements one by one; the count is 'val %s = %s.len()'", name, iter)
}

// assignsName reports whether s assigns name (`name = `, `name += `, a
// field or element under it) or takes its address, anywhere inside it —
// lambdas included, since a closure over a `var` can write it.
func assignsName(s ast.Stmt, name string) bool {
	found := false
	walkAST(s, func(n any) bool {
		if found {
			return false
		}
		switch n := n.(type) {
		case *ast.AssignStmt:
			target := n.Target
			for {
				m, ok := target.(*ast.MemberExpr)
				if !ok {
					break
				}
				target = m.X
			}
			if x, ok := target.(*ast.NameExpr); ok && x.Name == name {
				found = true
			}
			if t, ok := n.Target.(*ast.TupleExpr); ok && mentionsName(t, name) {
				found = true
			}
		case *ast.UnaryExpr:
			if x, ok := n.X.(*ast.NameExpr); ok && n.Op == lexer.Amp && x.Name == name {
				found = true
			}
		}
		return !found
	})
	return found
}
