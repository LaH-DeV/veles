package sema

import (
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Tuples of ordered elements are ordered lexicographically (D48, v0.28):
// `(1, "b") < (2, "a")`, `xs.sortedBy(e => (-e.size, e.name))`. The
// comparison is a synthesized `compareTo` per tuple type, which every
// ordering path — `<`, `sorted`, `min`, a `T: Comparable` bound, an
// explicit `a.compareTo(b)` — reaches through customOps.

// tupleCompare returns the comparison function of a tuple type, or nil when
// an element cannot be ordered.
func (c *Checker) tupleCompare(tt *types.Tuple) *Func {
	if types.ContainsTypeParam(tt) {
		return nil
	}
	if c.tupleCmp == nil {
		c.tupleCmp = map[string]*Func{}
	}
	key := types.Key(tt)
	if fn, done := c.tupleCmp[key]; done {
		return fn
	}
	c.tupleCmp[key] = nil // recursion guard
	for _, e := range tt.Elems {
		if !c.orderedType(e) {
			return nil
		}
	}
	ord := c.orderingType()
	fn := &Func{Name: mangleName("veles.tuple.compare<" + key + ">"), Display: "compareTo", Span: source.Span{},
		Sig: &types.Func{Params: []types.Param{{Name: "a", Type: tt}, {Name: "b", Type: tt}}, Ret: ord}}
	newVar := func(name string, t types.Type) *Var {
		c.nextVar++
		return &Var{Name: name, Type: t, ID: c.nextVar}
	}
	a := newVar("$a", tt)
	b := newVar("$b", tt)
	fn.Params = []*Var{a, b}
	body := &Block{Type: types.TNever}
	for i, et := range tt.Elems {
		l := &TupleGet{exprBase{et}, &VarRef{exprBase{tt}, a}, i}
		r := &TupleGet{exprBase{et}, &VarRef{exprBase{tt}, b}, i}
		cmp := newVar("$cmp", ord)
		cmp.Mutable = true
		body.Stmts = append(body.Stmts, &VarDecl{Var: cmp, Init: c.elemCompare(et, l, r)})
		nonZero := &Binary{exprBase{types.TBool}, OpNe, &VarRef{exprBase{ord}, cmp}, c.orderingConst(0), source.Span{}}
		early := &If{exprBase{types.TUnit}, nonZero, &Block{Stmts: []Stmt{&Return{Value: &VarRef{exprBase{ord}, cmp}}}, Type: types.TNever}, nil}
		body.Stmts = append(body.Stmts, &ExprStmt{X: early})
	}
	body.Stmts = append(body.Stmts, &Return{Value: c.orderingConst(0)})
	fn.Body = body
	fn.checked = true
	c.funcs = append(c.funcs, fn)
	c.tupleCmp[key] = fn
	return fn
}

// elemCompare is the three-way comparison of two values of an ordered type
// as an expression of type Ordering: Less, Equal or Greater for numbers,
// strings and enums, the custom or synthesized `compareTo` otherwise.
func (c *Checker) elemCompare(t types.Type, l, r Expr) Expr {
	ord := c.orderingType()
	if types.IsNumeric(t) || types.IsString(t) || types.IsEnum(t) {
		lt := &Binary{exprBase{types.TBool}, OpLt, l, r, source.Span{}}
		gt := &Binary{exprBase{types.TBool}, OpGt, l, r, source.Span{}}
		inner := &If{exprBase{ord}, gt, &Block{Value: c.orderingConst(1), Type: ord}, &Block{Value: c.orderingConst(0), Type: ord}}
		return &If{exprBase{ord}, lt, &Block{Value: c.orderingConst(-1), Type: ord}, &Block{Value: inner, Type: ord}}
	}
	if ops := c.customOps(t); ops != nil && ops.Compare != nil {
		return &Call{exprBase: exprBase{ord}, Fn: ops.Compare, Args: []Expr{recvArg(ops.Compare, l), r}}
	}
	return c.orderingConst(0)
}

// orderedType is the checker-level `ordered`: numbers, strings, types with a
// Comparable impl, and tuples of ordered elements.
func (c *Checker) orderedType(t types.Type) bool {
	if types.IsNumeric(t) || types.IsString(t) || types.IsEnum(t) {
		return true
	}
	if tt, ok := t.(*types.Tuple); ok {
		return c.tupleCompare(tt) != nil
	}
	return c.implementsPrelude(t, "Comparable")
}
