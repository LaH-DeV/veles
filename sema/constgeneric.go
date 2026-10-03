package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// newTypeParam declares a generic parameter: a type, or a constant when
// written `const N: i64` (D121).
func newTypeParam(tp ast.TypeParam, i int, owner string) *types.TypeParam {
	return &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: owner, Const: tp.Const}
}

// Constant arguments (D121). A `<const N: i64>` parameter takes, where a
// type argument goes, a number or a constant expression (`Array<u8, 64 * 4>`),
// the name of a constant, or another constant parameter in scope.

// maxArrayBytes bounds one inline array: a megabyte-scale buffer is
// ordinary, a gigabyte-scale one is a mistake the compiler can see.
const maxArrayBytes = 1 << 30

// resolveTypeArg resolves one type argument; isConst says the parameter it
// fills is a constant.
func (c *Checker) resolveTypeArg(env *typeEnv, a ast.Type, isConst bool) types.Type {
	if !isConst {
		if ct, ok := a.(*ast.ConstType); ok {
			c.errorf(ct.Pos, "expected a type here, found a number; a constant goes where the declaration says '<const N: i64>' (D121)")
			return types.TInvalid
		}
		return c.resolveType(env, a)
	}
	switch a := a.(type) {
	case *ast.ConstType:
		return c.constArg(env, a.X, a.Pos)
	case *ast.NamedType:
		if len(a.Args) == 0 {
			if len(a.Path) == 1 {
				if tp, ok := env.tps[a.Path[0].Name]; ok && tp.Const {
					return tp
				}
			}
			if sym, _ := c.lookupTypeName(env, a.Path); sym != nil && sym.Kind == SymType {
				c.errorf(a.Pos, "expected a constant here, found the type '%s'; this argument is a '<const N: i64>' (D121)", pathString(a.Path))
				return types.TInvalid
			}
			return c.constArg(env, pathExpr(a.Path), a.Pos)
		}
	}
	c.errorf(a.Span(), "expected a constant here: a number, the name of a constant, or an expression of constants (D121)")
	return types.TInvalid
}

// pathExpr is the expression a (possibly qualified) name stands for.
func pathExpr(path []ast.Ident) ast.Expr {
	var x ast.Expr = &ast.NameExpr{Name: path[0].Name, Pos: path[0].Pos}
	for _, seg := range path[1:] {
		x = &ast.MemberExpr{X: x, Name: seg, Pos: path[0].Pos.To(seg.Pos)}
	}
	return x
}

// constArg evaluates a constant written as a type argument.
func (c *Checker) constArg(env *typeEnv, x ast.Expr, span source.Span) types.Type {
	var param string
	walkAST(x, func(n any) bool {
		if name, ok := n.(*ast.NameExpr); ok && param == "" {
			if tp, isTP := env.tps[name.Name]; isTP && tp.Const {
				param = name.Name
			}
		}
		return param == ""
	})
	if param != "" {
		c.errorf(span, "a type argument may name the constant parameter '%s' or be built from module constants, not calculated from it: 'Array<u8, %s + 1>' needs its own parameter, 'Array<u8, M>' with 'M' given by the caller (D121)", param, param)
		return types.TInvalid
	}
	f := c.newFnCtx(nil, env.module, env.file, env, nil)
	f.isGlobal = true
	e := f.checkExprTo(x, types.TI64)
	if types.IsInvalid(e.Type()) {
		return types.TInvalid
	}
	v, ok := c.evalConst(e, span).(*CInt)
	if !ok {
		return types.TInvalid
	}
	if !v.V.IsInt64() || v.V.Sign() < 0 || v.V.Int64() > maxArrayBytes {
		c.errorf(span, "a constant argument is a length from 0 to %d, not %s (D121)", maxArrayBytes, v.V)
		return types.TInvalid
	}
	return &types.Const{V: v.V.Int64()}
}

// arrayType builds `Array<T, N>` from the resolved arguments.
func (c *Checker) arrayType(elem, n types.Type, span source.Span) types.Type {
	if types.IsInvalid(elem) || types.IsInvalid(n) {
		return types.TInvalid
	}
	a := &types.Array{Elem: elem, Len: n}
	if _, known := n.(*types.Const); known && !types.ContainsTypeParam(elem) {
		c.arrayUses = append(c.arrayUses, arrayUse{a, span})
	}
	return a
}

type arrayUse struct {
	t    *types.Array
	span source.Span
}

// checkArraySizes runs once every struct is resolved: an inline array of
// more than a gigabyte is refused.
func (c *Checker) checkArraySizes() {
	lay := &types.Layout{}
	for _, u := range c.arrayUses {
		if size, _ := lay.Of(u.t); size > maxArrayBytes || size < 0 {
			c.errorf(u.span, "'%s' would be %d bytes; an inline array is at most %d (1 GiB). Use a List for a buffer this large (D121)", u.t, size, maxArrayBytes)
		}
	}
	c.arrayUses = nil
}

// resolveCallArg resolves an explicit type argument of a call, before the
// callee is known: a number, or a name that is a constant (a constant
// parameter in scope, or a module-level constant) and no type, is a
// constant argument (`zeros<16>()`, `zeros<N>()`); anything else a type.
func (f *fnCtx) resolveCallArg(ta ast.Type) types.Type {
	switch a := ta.(type) {
	case *ast.ConstType:
		return f.c.hooks.Subst(f.c.constArg(f.env, a.X, a.Pos), f.subst)
	case *ast.NamedType:
		if len(a.Args) == 0 && len(a.Path) == 1 {
			name := a.Path[0].Name
			if tp, ok := f.env.tps[name]; ok && tp.Const {
				return f.c.hooks.Subst(tp, f.subst)
			}
			if sym, _ := f.c.lookupTypeName(f.env, a.Path); sym == nil {
				if g := f.lookup(name); g != nil && g.Kind == SymGlobal {
					return f.c.constArg(f.env, pathExpr(a.Path), a.Pos)
				}
			}
		}
	}
	return f.resolve(ta)
}

// constParam reads a constant parameter in a body as the constant it was
// instantiated with (D121): inside `fun zeros<const N: i64>()` the name
// `N` is, in each instance, the number. A local of that name shadows it.
func (f *fnCtx) constParam(e *ast.NameExpr, sym *Symbol) Expr {
	if sym != nil && sym.Kind == SymLocal {
		return nil
	}
	tp, ok := f.env.tps[e.Name]
	if !ok || !tp.Const {
		return nil
	}
	c, ok := f.c.hooks.Subst(tp, f.subst).(*types.Const)
	if !ok {
		return nil // checked as a template, with no instance to read it from
	}
	return &IntConst{exprBase{types.TI64}, uint64(c.V), false}
}

// arrayLit checks a `[…]` literal where an `Array<T, N>` is expected
// (D121): the length must be N, and an N still being inferred is taken
// from the literal.
func (f *fnCtx) arrayLit(e *ast.ListLit, at *types.Array) Expr {
	var elem types.Type
	if !types.ContainsTypeParam(at.Elem) {
		elem = at.Elem
	}
	if e.Mut {
		f.errorf(e.Pos, "'mut' makes a MutableList; an Array is changed through a 'var' binding, so the literal needs no 'mut' (D121)")
	}
	lit := &ListLit{}
	for _, el := range e.Elems {
		var x Expr
		if elem != nil {
			x = f.checkExprTo(el, elem)
		} else {
			x = f.checkExpr(el, nil)
			elem = x.Type()
		}
		lit.Elems = append(lit.Elems, x)
	}
	if elem == nil {
		f.errorf(e.Pos, "cannot infer the element type of an empty array literal; annotate it, e.g. 'val a: Array<i64, 0> = []' (D121)")
		return bad()
	}
	n := int64(len(e.Elems))
	if want, ok := at.Len.(*types.Const); ok && want.V != n {
		f.errorf(e.Pos, "'%s' holds %d elements, this literal has %d (D121)", at, want.V, n)
		return bad()
	}
	lit.T = &types.Array{Elem: elem, Len: &types.Const{V: n}}
	return lit
}
