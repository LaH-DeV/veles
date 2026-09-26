package sema

import (
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// Bounds facts (D62 B): `xs.at(i)` is a `T`, not a `T?`, where the checker
// knows the index is in range — D5's smart casts, applied to an index.
//
// The facts ride in f.narrow beside the smart casts, under paths no field
// can have, so the flow machinery that already exists — branches, `&&`,
// `||`, `!`, the early exit, the merge after an `if`, a loop head dropping
// what its body assigns — carries them without a second copy of it:
//
//	place{i,  "$in:<xs>"}  i < xs.len()   (value: xs's list type)
//	place{i,  "$nonneg"}   i >= 0         (value: unit)
//	place{xs, "$len>=n"}   xs.len() >= n  (value: xs's list type)
//
// Only locals take part — a list or an index held in a variable whose
// address is not taken — so an assignment is the one way to change them,
// and an assignment drops the facts (invalidatePlace). A MutableList can
// also shrink through an alias, so its facts end at any call that could
// reach it (killMutableBounds). The read the fact proves still goes
// through the runtime's checked get: a missed case is a panic, never a
// read out of bounds.
const (
	pathIn     = "$in:"
	pathLen    = "$len>="
	pathNonNeg = "$nonneg"
	pathLenOf  = "$lenof:"
)

func inKey(idx, list *Var) place { return place{v: idx, path: pathIn + strconv.Itoa(list.ID)} }
func nonNegKey(idx *Var) place   { return place{v: idx, path: pathNonNeg} }
func lenKey(list *Var, n int64) place {
	return place{v: list, path: pathLen + strconv.FormatInt(n, 10)}
}

func isBoundsPath(p string) bool {
	return strings.HasPrefix(p, pathIn) || strings.HasPrefix(p, pathLen) || strings.HasPrefix(p, pathLenOf) || p == pathNonNeg
}

// boundsList is the local list variable e names, if facts can be kept
// about it.
func (f *fnCtx) boundsList(e ast.Expr) (*Var, *types.List) {
	v := varOf(e, f)
	if _, isSelf := e.(*ast.SelfExpr); isSelf {
		v = f.selfRef() // `this.len()` in an `extend List`
	}
	if v == nil || v.AddrTaken || v.Captured {
		return nil, nil
	}
	lt, ok := f.currentTypeOf(pv(v)).(*types.List)
	if !ok {
		return nil, nil
	}
	if v.IsSelf && !lt.Mutable {
		// an `extend List` method also runs on a MutableList (D25), so its
		// receiver may change during a call like any MutableList
		lt = &types.List{Elem: lt.Elem, Mutable: true}
	}
	return v, lt
}

// boundsIndex is the local integer variable e names, if facts can be kept
// about it.
func (f *fnCtx) boundsIndex(e ast.Expr) *Var {
	v := varOf(e, f)
	if v == nil || v.AddrTaken || v.Captured || !types.IsInteger(f.currentTypeOf(pv(v))) {
		return nil
	}
	return v
}

// lenCallOf recognises `xs.len()` on a local list.
func (f *fnCtx) lenCallOf(e ast.Expr) (*Var, *types.List) {
	c, ok := e.(*ast.CallExpr)
	if !ok || len(c.Args) != 0 {
		return nil, nil
	}
	m, ok := c.Fun.(*ast.MemberExpr)
	if !ok || m.Safe || m.Name.Name != "len" {
		return nil, nil
	}
	return f.boundsList(m.X)
}

// intConst is the value of an integer literal, negated or not.
func intConst(e ast.Expr) (int64, bool) {
	neg := false
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == lexer.Minus {
		neg, e = true, u.X
	}
	lit, ok := e.(*ast.IntLit)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.ReplaceAll(lit.Text, "_", ""), 0, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		v = -v
	}
	return v, true
}

// flipCmp is the operator with its operands swapped: `a < b` is `b > a`.
func flipCmp(op lexer.TokenKind) lexer.TokenKind {
	switch op {
	case lexer.Lt:
		return lexer.Gt
	case lexer.Gt:
		return lexer.Lt
	case lexer.LtEq:
		return lexer.GtEq
	case lexer.GtEq:
		return lexer.LtEq
	}
	return op
}

// boundsCondFacts adds what a comparison says about indexes and lengths:
// `i < xs.len()`, `i >= 0`, `xs.len() == 3`, `xs.len() >= n` and their
// mirror images and negations (the false side of `i >= xs.len()` is
// `i < xs.len()`, so `if (i >= xs.len()) return` leaves the fact behind).
func (f *fnCtx) boundsCondFacts(c *ast.BinaryExpr, whenTrue, whenFalse facts) {
	op, l, r := c.Op, c.L, c.R
	switch op {
	case lexer.Lt, lexer.LtEq, lexer.Gt, lexer.GtEq, lexer.Eq, lexer.NotEq:
	default:
		return
	}
	// put the length, or the index, on the left
	if lv, _ := f.lenOf(r); lv != nil {
		op, l, r = flipCmp(op), r, l
	} else if _, isConst := intConst(l); isConst {
		op, l, r = flipCmp(op), r, l
	}

	if list, lt := f.lenOf(l); list != nil {
		// xs.len() <op> i: only `>` and `<=` say something about i
		if idx := f.boundsIndex(r); idx != nil {
			switch op {
			case lexer.Gt:
				whenTrue[inKey(idx, list)] = lt
			case lexer.LtEq:
				whenFalse[inKey(idx, list)] = lt
			}
			return
		}
		// xs.len() <op> n
		n, ok := intConst(r)
		if !ok {
			return
		}
		atLeast := func(fs facts, k int64) {
			if k > 0 {
				fs[lenKey(list, k)] = lt
			}
		}
		switch op {
		case lexer.Eq, lexer.GtEq:
			atLeast(whenTrue, n)
			if op == lexer.Eq && n == 0 {
				atLeast(whenFalse, 1) // a length is never negative
			}
		case lexer.NotEq:
			atLeast(whenFalse, n)
			if n == 0 {
				atLeast(whenTrue, 1)
			}
		case lexer.Gt:
			atLeast(whenTrue, n+1)
		case lexer.Lt:
			atLeast(whenFalse, n)
		case lexer.LtEq:
			atLeast(whenFalse, n+1)
		}
		return
	}

	idx := f.boundsIndex(l)
	if idx == nil {
		return
	}
	// i <op> xs.len() was flipped above to xs.len() <op'> i, so what is
	// left is i <op> n
	n, ok := intConst(r)
	if !ok {
		return
	}
	switch {
	case op == lexer.GtEq && n >= 0, op == lexer.Gt && n >= -1:
		whenTrue[nonNegKey(idx)] = types.TUnit
	case op == lexer.Lt && n <= 0, op == lexer.LtEq && n < 0:
		whenFalse[nonNegKey(idx)] = types.TUnit
	}
}

// isEmptyFacts: `xs.isEmpty()` is false when the list has an element.
func (f *fnCtx) isEmptyFacts(c *ast.CallExpr, whenFalse facts) {
	m, ok := c.Fun.(*ast.MemberExpr)
	if !ok || m.Safe || m.Name.Name != "isEmpty" || len(c.Args) != 0 {
		return
	}
	if list, lt := f.boundsList(m.X); list != nil {
		whenFalse[lenKey(list, 1)] = lt
	}
}

// dropMutableBounds removes the facts about MutableLists from fs: what a
// condition learned before a call later in the same condition.
func dropMutableBounds(fs facts) {
	for k, t := range fs {
		if lt, ok := t.(*types.List); ok && lt.Mutable && isBoundsPath(k.path) {
			delete(fs, k)
		}
	}
}

// killMutableBounds ends every fact about a MutableList: a call may reach
// the list through an alias and shrink it (D62).
func (f *fnCtx) killMutableBounds() {
	dropMutableBounds(f.narrow)
}

func (f *fnCtx) hasMutableBounds() bool {
	for k, t := range f.narrow {
		if lt, ok := t.(*types.List); ok && lt.Mutable && isBoundsPath(k.path) {
			return true
		}
	}
	return false
}

// afterExpr ends the MutableList facts when the checked expression x can
// run code that might shrink one: any call, or a pop or clear.
func (f *fnCtx) afterExpr(x Expr) {
	if x == nil || !f.hasMutableBounds() {
		return
	}
	if hirMayShrink(x) {
		f.killMutableBounds()
	}
}

func hirMayShrink(x Expr) bool {
	found := false
	walkExpr(x, func(n any) {
		switch n := n.(type) {
		case *Call, *CallIndirect, *CallVirtual, *AwaitTask:
			found = true
		case *Builtin:
			if n.Op == "list.pop" || n.Op == "list.clear" {
				found = true
			}
		}
	})
	return found
}

// pureListMethods are the methods that cannot run user code or shrink a
// list; every other call is assumed to be able to.
var pureListMethods = map[string]bool{
	"at": true, "len": true, "set": true, "ref": true, "isEmpty": true, "push": true,
	"byteAt": true, "get": true,
}

// astMayShrink reports whether running node could shrink a MutableList:
// it contains a call other than a pure list method. Used before the code
// is checked — a loop body runs again after its last statement, so a call
// anywhere in it ends the facts everywhere in it.
func astMayShrink(node any) bool {
	found := false
	walkAST(node, func(n any) bool {
		if found {
			return false
		}
		switch n := n.(type) {
		case *ast.LambdaExpr:
			return false // runs only when called, and a call is a call
		case *ast.CallExpr:
			if name, isName := n.Fun.(*ast.NameExpr); isName && name.Name == "panic" {
				return false // never returns: nothing after it sees the list
			}
			m, ok := n.Fun.(*ast.MemberExpr)
			if !ok || !pureListMethods[m.Name.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// astChangesVar reports whether node assigns the local named name or takes
// its address.
func astChangesVar(node any, name string) bool {
	found := false
	root := func(e ast.Expr) string {
		for {
			switch x := e.(type) {
			case *ast.MemberExpr:
				e = x.X
				continue
			case *ast.NameExpr:
				return x.Name
			}
			return ""
		}
	}
	walkAST(node, func(n any) bool {
		if found {
			return false
		}
		switch n := n.(type) {
		case *ast.AssignStmt:
			if root(n.Target) == name {
				found = true
			}
		case *ast.UnaryExpr:
			if n.Op == lexer.Amp && root(n.X) == name {
				found = true
			}
		}
		return !found
	})
	return found
}

// rangeLoopFacts are the facts `loop (i in lo..<hi)` gives its body: i is
// never negative when lo is a non-negative constant, and i < xs.len() when
// hi is `xs.len()` — unless the body can change xs or shrink it, since the
// range is evaluated once and the body runs many times.
func (f *fnCtx) rangeLoopFacts(s *ast.LoopStmt, idx *Var) {
	if idx == nil || idx.Mutable {
		return
	}
	var list *Var
	var lt *types.List
	switch it := s.Iter.(type) {
	case *ast.RangeExpr:
		lo, ok := intConst(it.Lo)
		if !ok || lo < 0 {
			return
		}
		f.narrow[nonNegKey(idx)] = types.TUnit
		if it.Inclusive {
			return
		}
		list, lt = f.lenOf(it.Hi)
	case *ast.CallExpr:
		// `xs.indices()`
		m, ok := it.Fun.(*ast.MemberExpr)
		if !ok || m.Safe || m.Name.Name != "indices" || len(it.Args) != 0 {
			return
		}
		list, lt = f.boundsList(m.X)
		if list == nil {
			return
		}
		f.narrow[nonNegKey(idx)] = types.TUnit
	default:
		return
	}
	if list == nil || astChangesVar(s.Body, list.Name) {
		return
	}
	if lt.Mutable && astMayShrink(s.Body) {
		return
	}
	f.narrow[inKey(idx, list)] = lt
}

// indexProven reports whether the facts put `recv.at(arg)` in range: a
// constant below a known length (or a negative one within it, counted from
// the end), or an index variable known to be `0 <= i < recv.len()`.
func (f *fnCtx) indexProven(recv, arg ast.Expr) bool {
	lv, _ := f.boundsList(recv)
	if lv == nil {
		return false
	}
	if k, ok := intConst(arg); ok {
		known := int64(0)
		for key := range f.narrow {
			if key.v == lv && strings.HasPrefix(key.path, pathLen) {
				if n, err := strconv.ParseInt(key.path[len(pathLen):], 10, 64); err == nil && n > known {
					known = n
				}
			}
		}
		return (k >= 0 && k < known) || (k < 0 && -k <= known)
	}
	idx := f.boundsIndex(arg)
	if idx == nil {
		return false
	}
	_, in := f.narrow[inKey(idx, lv)]
	_, nonNeg := f.narrow[nonNegKey(idx)]
	return in && nonNeg
}

// receiverIndexProven is indexProven for the receiver of a method call.
func (f *fnCtx) receiverIndexProven(e *ast.CallExpr, arg ast.Expr) bool {
	m, ok := e.Fun.(*ast.MemberExpr)
	return ok && !m.Safe && f.indexProven(m.X, arg)
}

// markProven records a read a bounds fact made total.
func (f *fnCtx) markProven(e *ast.CallExpr) {
	if f.provenReads == nil {
		f.provenReads = map[*ast.CallExpr]bool{}
	}
	f.provenReads[e] = true
}

// lenOf is the list whose length e stands for: `xs.len()`, or a `val`
// bound to it while the list is unchanged.
func (f *fnCtx) lenOf(e ast.Expr) (*Var, *types.List) {
	if list, lt := f.lenCallOf(e); list != nil {
		return list, lt
	}
	n := varOf(e, f)
	if n == nil {
		return nil, nil
	}
	list := f.lenAliases[n]
	if list == nil {
		return nil, nil
	}
	t, ok := f.narrow[place{v: n, path: pathLenOf + strconv.Itoa(list.ID)}]
	if !ok {
		return nil, nil
	}
	return list, t.(*types.List)
}

// declFacts are what a new binding's initializer says: `var i = 0` is not
// negative, and `val n = xs.len()` stands for the length of xs.
func (f *fnCtx) declFacts(v *Var, init ast.Expr) {
	if !types.IsInteger(v.Type) {
		return
	}
	if k, ok := intConst(init); ok && k >= 0 {
		f.narrow[nonNegKey(v)] = types.TUnit
		return
	}
	list, lt := f.lenCallOf(init)
	if list == nil {
		return
	}
	f.narrow[nonNegKey(v)] = types.TUnit
	if v.Mutable {
		return
	}
	if f.lenAliases == nil {
		f.lenAliases = map[*Var]*Var{}
	}
	f.lenAliases[v] = list
	f.narrow[place{v: v, path: pathLenOf + strconv.Itoa(list.ID)}] = lt
}

// countsUp is the variable an assignment counts up — `i += k` or
// `i = i + k` with k a non-negative constant — when it is known not to be
// negative, so that it stays so. (A counter that wraps past the largest
// i64 would break this; the runtime's checked read still catches it.)
func (f *fnCtx) countsUp(s *ast.AssignStmt) *Var {
	n, ok := s.Target.(*ast.NameExpr)
	if !ok {
		return nil
	}
	v := varOf(n, f)
	if v == nil {
		return nil
	}
	if _, known := f.narrow[nonNegKey(v)]; !known {
		return nil
	}
	nonNegConst := func(e ast.Expr) bool {
		k, ok := intConst(e)
		return ok && k >= 0
	}
	switch s.Op {
	case lexer.PlusEq:
		if nonNegConst(s.Value) {
			return v
		}
	case lexer.Assign:
		b, ok := s.Value.(*ast.BinaryExpr)
		if !ok || b.Op != lexer.Plus {
			return nil
		}
		same := func(e ast.Expr) bool {
			m, ok := e.(*ast.NameExpr)
			return ok && m.Name == n.Name
		}
		if same(b.L) && nonNegConst(b.R) || same(b.R) && nonNegConst(b.L) {
			return v
		}
	}
	return nil
}
