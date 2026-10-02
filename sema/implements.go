package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// `T implements Trait` (D117, compile time). A generic body is checked once
// per instance, with its type arguments substituted, so the condition is a
// constant in each: true where the argument implements the trait. An `if`
// whose condition that constant decides checks and compiles only the
// branch the instance takes — the other may call what this argument does
// not have (`x.toString()` where T is not Display). The untaken branch is
// not checked for that instance; another instance, or none, takes it.

// implementsCond checks `T implements Trait` and gives its answer.
func (f *fnCtx) implementsCond(e *ast.ImplementsExpr) Expr {
	answer, ok := f.implementsAnswer(e, true)
	if !ok {
		return &BoolConst{exprBase{types.TBool}, false}
	}
	return &BoolConst{exprBase{types.TBool}, answer}
}

// implementsAnswer is the value of `T implements Trait` in this instance;
// ok is false when the condition is not well formed (reported when report).
func (f *fnCtx) implementsAnswer(e *ast.ImplementsExpr, report bool) (answer, ok bool) {
	tp := f.typeParamNamed(e.Type)
	if tp == nil {
		if report {
			name := ""
			if nt, isNamed := e.Type.(*ast.NamedType); isNamed && len(nt.Path) > 0 {
				name = nt.Path[len(nt.Path)-1].Name
			}
			f.errorf(e.Type.Span(), "'implements' asks about a type parameter of the function or type it is written in, and '%s' is not one; for a value of a trait-object type write 'x is Trait' (D117)", name)
		}
		return false, false
	}
	rt := f.resolve(e.Trait)
	tr, isTrait := rt.(*types.Trait)
	if !isTrait {
		if report && !types.IsInvalid(rt) {
			f.errorf(e.Trait.Span(), "'%s' is not a trait; 'implements' asks whether a type implements a trait (D117)", rt)
		}
		return false, false
	}
	arg := f.resolve(e.Type)
	if _, still := arg.(*types.TypeParam); still || types.IsInvalid(arg) {
		return false, false
	}
	return f.implements(arg, tr), true
}

// staticCond is what an `if` condition is known to be in this instance,
// when `implements` decides it: `T implements X`, combined with `!`, `&&`
// and `||` (a part that is not known keeps the whole unknown unless the
// other part decides it). Any other condition, `false` included, is
// unknown: both branches are checked as always.
func (f *fnCtx) staticCond(cond ast.Expr) (value, known bool) {
	switch c := cond.(type) {
	case *ast.ImplementsExpr:
		return f.implementsAnswer(c, false)
	case *ast.UnaryExpr:
		if c.Op == lexer.Bang {
			v, k := f.staticCond(c.X)
			return !v, k
		}
	case *ast.BinaryExpr:
		lv, lk := f.staticCond(c.L)
		rv, rk := f.staticCond(c.R)
		switch c.Op {
		case lexer.AndAnd:
			if lk && !lv || rk && !rv {
				return false, true
			}
			return true, lk && rk
		case lexer.OrOr:
			if lk && lv || rk && rv {
				return true, true
			}
			return false, lk && rk
		}
	}
	return false, false
}

// staticIf is an `if` whose condition `implements` decides: the branch
// taken, checked and compiled alone.
func (f *fnCtx) staticIf(e *ast.IfExpr, taken bool, want types.Type, asValue bool) Expr {
	live, dead := e.Then, e.Else
	if !taken {
		live, dead = e.Else, e.Then
	}
	f.skipBranch(dead)
	empty := func() *Block { return &Block{Type: types.TUnit} }
	if live == nil {
		return &BlockExpr{exprBase{types.TUnit}, empty()}
	}
	b := f.checkBlock(live, want, asValue || want == nil)
	if asValue && e.Else != nil {
		return &BlockExpr{exprBase{b.Type}, b}
	}
	// a statement: its value is discarded, as in any `if`, and it stays an
	// `if` over the constant — a branch that leaves (`return x.toString()`)
	// leaves in this instance only, and what follows is still reached in
	// the instances that do not take it
	if b.Value != nil {
		b.Stmts = append(b.Stmts, &ExprStmt{X: b.Value})
		b.Value = nil
	}
	b.Type = types.TUnit
	cond := &BoolConst{exprBase{types.TBool}, taken}
	if taken {
		return &If{exprBase{types.TUnit}, cond, b, empty()}
	}
	return &If{exprBase{types.TUnit}, cond, empty(), b}
}

// skipBranch leaves a branch this instance does not take unchecked; the
// locals it names count as used, as they are in the instances that take
// it.
func (f *fnCtx) skipBranch(b *ast.Block) {
	if b == nil {
		return
	}
	walkAST(b, func(n any) bool {
		if ne, ok := n.(*ast.NameExpr); ok {
			if sym := f.lookup(ne.Name); sym != nil && sym.Kind == SymLocal {
				markUsed(sym.Var)
			}
		}
		return true
	})
}
