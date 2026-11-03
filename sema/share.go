package sema

import (
	"github.com/LaH-DeV/veles/types"
)

// The share pass (D231). Lists, maps and sets are values, held as a pointer
// to a header. A value read out of a place that stays alive — a variable, a
// field, an element, a dereference — and then stored, passed, returned or
// captured is a second holder of that header: the pass wraps such a read in
// a Share, which marks the header shared, so that the first change through
// either holder copies it first (codegen/llvm/cow.go).
//
// No Share is needed where the value is only looked at (a comparison, a
// condition, the receiver of a built-in), where it is fresh (a call's
// result, a literal), where a local is returned at its end (a move), where a
// lent temporary holds it (newLentTemp), or where it is lent to a parameter
// the callee neither changes nor keeps (Var.ReadOnly) and nothing else can
// reach it during the call (a local no closure captures and whose address
// is not taken). The callee marks it itself if it keeps it.
//
// It runs after the receiver pass, which settles ReadOnly, captures and
// address-taken variables.

// use is what the value of an expression is used for.
type use int

const (
	useLook   use = iota // read for the moment: no second holder
	useKeep              // stored, bound, passed or returned: a second holder
	useReturn            // the function's result: a move for a local at its end
)

type sharer struct {
	c  *Checker
	fn *Func
	// ownedOnce: parameters whose one keep is a move (ownedOnceParams)
	ownedOnce map[*Var]bool
	// partsMoved: tuples whose parts are each taken once (partsMoved)
	partsMoved map[*Var]bool
}

func (c *Checker) sharePass(prog *Program) {
	for _, fn := range prog.Funcs {
		c.shareBody(fn)
	}
	for _, g := range prog.Globals {
		if g.Init != nil {
			s := &sharer{c: c}
			g.Init = s.expr(g.Init, useKeep)
		}
	}
}

// shareBody runs the pass over one function, once. (The compiler's evaluator
// needs none of it: it copies a collection wherever it binds one, copyVal.)
func (c *Checker) shareBody(fn *Func) {
	if fn.Body == nil || fn.shared {
		return
	}
	fn.shared = true
	result := useReturn
	if fn.Sig != nil && types.IsUnit(fn.Sig.Ret) {
		result = useLook
	}
	(&sharer{c: c, fn: fn, ownedOnce: ownedOnceParams(fn), partsMoved: partsMoved(fn)}).block(fn.Body, result)
}

// holdsCollection reports whether a value of t holds a list, map or set by
// value (codegen's needsShare, without its knowledge of boxed variants).
func holdsCollection(t types.Type, seen map[types.Type]bool) bool {
	switch t := types.Underlying(t).(type) {
	case *types.List, *types.Map, *types.Set:
		return true
	case *types.Nullable:
		return holdsCollection(t.Elem, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if holdsCollection(e, seen) {
				return true
			}
		}
	case *types.Array:
		return holdsCollection(t.Elem, seen)
	case *types.Struct:
		if t.Union || seen[t] {
			return false
		}
		seen[t] = true
		for _, f := range t.Fields {
			if holdsCollection(f.Type, seen) {
				return true
			}
		}
	case *types.Sealed:
		if seen[t] {
			return false
		}
		seen[t] = true
		for _, v := range t.Variants {
			if holdsCollection(v, seen) {
				return true
			}
		}
	}
	return false
}

// held reports whether e's value is also held where it was read from: a
// place, an element, or a part of one.
func held(e Expr) bool {
	switch e := e.(type) {
	case *VarRef, *Deref, *Downcast, *LazyRead:
		return true
	case *FieldGet:
		return held(e.X)
	case *TupleGet:
		return held(e.X)
	case *Unwrap:
		return held(e.X)
	case *VariantCast:
		return held(e.X)
	case *ResultValue:
		return held(e.X)
	case *UnionCast:
		return held(e.X)
	case *Builtin:
		switch e.Op {
		case "list.get", "list.ref", "list.refUnchecked", "map.get", "deref":
			return true
		}
	}
	return false
}

// heldIn is the variable a value read from a place or an element is in, or
// nil (behind a pointer, a call's result).
func heldIn(e Expr) *Var {
	for {
		switch x := e.(type) {
		case *VarRef:
			return x.Var
		case *Deref:
			if v, ok := x.X.(*VarRef); ok && v.Var.IsSelf {
				return v.Var
			}
			return nil
		case *FieldGet:
			e = x.X
		case *TupleGet:
			e = x.X
		case *Unwrap:
			e = x.X
		case *VariantCast:
			e = x.X
		case *Builtin:
			switch x.Op {
			case "list.get", "list.ref", "list.refUnchecked", "map.get":
				e = x.Args[0]
			default:
				return nil
			}
		default:
			return nil
		}
	}
}

// lendable: the value of e can be lent to a ReadOnly parameter — nothing but
// this function can reach where it lives while the call runs: a local no
// closure captures and whose address is not taken. A value read through a
// pointer (`*p`, `p.items`, `this.f`) is not: the callee may reach the same
// object through a global, a struct or closure it is handed, or its own
// receiver, and change it under its parameter (found 2026-11-03; those were
// lent while no other argument was a pointer).
func lendable(e Expr) bool {
	v := heldIn(e)
	return v != nil && !v.IsGlobal && !v.Captured && !v.AddrTaken && !v.IsSelf
}

// derefRoot is the local pointer variable a place is read through, or nil.
func derefRoot(e Expr) *Var {
	for {
		switch x := e.(type) {
		case *Deref:
			if r, ok := x.X.(*VarRef); ok && !r.Var.IsSelf {
				return r.Var
			}
			return nil
		case *FieldGet:
			e = x.X
		case *TupleGet:
			e = x.X
		case *Unwrap:
			e = x.X
		case *VariantCast:
			e = x.X
		case *Builtin:
			switch x.Op {
			case "list.get", "list.ref", "list.refUnchecked", "map.get":
				e = x.Args[0]
			default:
				return nil
			}
		default:
			return nil
		}
	}
}

// loopChangesSubject: tmp is the subject of a loop whose body may change the
// collection init reads (a write rooted at the same variable, `this`
// included): the loop then takes a copy, so it walks the collection as it
// was when it began (D231 addendum). A change the body makes only through a
// function it calls is not seen here; the loop's guard reports that one
// (D102).
func loopChangesSubject(tmp *Var, init Expr) bool {
	lp := tmp.LentFor
	if lp == nil || lp.Body == nil {
		return false
	}
	v := heldIn(init)
	if v == nil {
		v = derefRoot(init)
	}
	if v == nil {
		return false
	}
	rooted := func(x Expr) bool { return collRoot(x) == v || derefRoot(x) == v }
	found := false
	walkBlock(lp.Body, func(n any) {
		switch n := n.(type) {
		case *Assign:
			found = found || rooted(n.Target)
		case *AddrOf:
			found = found || rooted(n.X)
		case *Builtin:
			found = found || (collectionWrites[n.Op] && rooted(n.Args[0]))
		}
	})
	return found
}

// movable: e is a local this function returns at its end, so the value
// moves to the caller rather than being held twice.
func movable(e Expr) bool {
	ref, ok := e.(*VarRef)
	if !ok {
		return false
	}
	v := ref.Var
	return !v.IsParam && !v.IsGlobal && !v.Captured && !v.AddrTaken && !v.IsSelf && !v.Lent
}

// leaf wraps e in a Share when its use keeps a value also held elsewhere.
func (s *sharer) leaf(e Expr, u use) Expr {
	if u == useLook || !held(e) || !holdsCollection(e.Type(), map[types.Type]bool{}) {
		return e
	}
	if u == useReturn && movable(e) {
		return e
	}
	if r, ok := e.(*VarRef); ok && s.ownedOnce[r.Var] {
		return e // the function's own copy, kept once: a move
	}
	if tg, ok := e.(*TupleGet); ok {
		if r, ok := tg.X.(*VarRef); ok && s.partsMoved[r.Var] {
			return e // a tuple's part, taken once: a move
		}
	}
	return &Share{exprBase{e.Type()}, e}
}

func (s *sharer) block(b *Block, valueUse use) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		s.stmt(st)
	}
	if b.Value != nil {
		b.Value = s.expr(b.Value, valueUse)
	}
}

func (s *sharer) stmt(st Stmt) {
	switch st := st.(type) {
	case *Block:
		s.block(st, useLook)
	case *VarDecl:
		if st.Init != nil {
			if s.lentAlias(st.Var, st.Init) {
				st.Var.Lent = true // only read, from a place nothing changes meanwhile
			}
			u := useKeep
			if st.Var.Lent && !loopChangesSubject(st.Var, st.Init) {
				u = useLook
			}
			st.Init = s.expr(st.Init, u)
		}
	case *Assign:
		st.Target = s.place(st.Target)
		st.Value = s.expr(st.Value, useKeep)
	case *ExprStmt:
		st.X = s.expr(st.X, useLook)
	case *Return:
		if st.Value != nil {
			st.Value = s.expr(st.Value, useReturn)
		}
	case *Loop:
		if st.Cond != nil {
			st.Cond = s.expr(st.Cond, useLook)
		}
		s.block(st.Body, useLook)
		for _, p := range st.Post {
			s.stmt(p)
		}
	case *With:
		st.Init = s.expr(st.Init, useKeep)
		if st.Close != nil {
			st.Close = s.expr(st.Close, useLook)
		}
		s.block(st.Body, useLook)
	case *ScopeBlock:
		s.scope(st, useLook)
	}
}

// place walks an assignment target: its own storage is written, not read,
// so only the values inside it (an index) are expressions to walk.
func (s *sharer) place(e Expr) Expr {
	switch x := e.(type) {
	case *FieldGet:
		x.X = s.place(x.X)
	case *TupleGet:
		x.X = s.place(x.X)
	case *Unwrap:
		x.X = s.place(x.X)
	case *VariantCast:
		x.X = s.place(x.X)
	case *Deref:
		x.X = s.expr(x.X, useLook)
	case *Builtin:
		for i, a := range x.Args {
			if i == 0 {
				x.Args[0] = s.place(a)
			} else {
				x.Args[i] = s.expr(a, useLook)
			}
		}
	}
	return e
}

func (s *sharer) exprs(es []Expr, u use) {
	for i, e := range es {
		es[i] = s.expr(e, u)
	}
}

// expr walks e, whose value is used as u, and returns it, wrapped in a
// Share where needed.
func (s *sharer) expr(e Expr, u use) Expr {
	switch x := e.(type) {
	case nil:
		return nil
	case *VarRef, *IntConst, *FloatConst, *BoolConst, *StringConst, *UnitConst, *Zero,
		*NullConst, *FuncRef, *ConstTable, *Panic:
	case *Share:
		// `return out.toList()`: the copy of a local that ends here cannot
		// be seen — it is the move
		if u == useReturn && movable(x.X) {
			return x.X
		}
		x.X = s.expr(x.X, useLook)
		return x
	case *Call:
		s.call(x)
	case *CallIndirect:
		x.Fn = s.expr(x.Fn, useLook)
		s.exprs(x.Args, useKeep)
	case *CallVirtual:
		x.Obj = s.expr(x.Obj, useLook)
		s.exprs(x.Args, useKeep)
	case *Builtin:
		s.builtin(x)
	case *Binary:
		x.L = s.expr(x.L, useLook)
		x.R = s.expr(x.R, useLook)
	case *Unary:
		x.X = s.expr(x.X, useLook)
	case *ToString:
		x.X = s.expr(x.X, useLook)
	case *StringConcat:
		s.exprs(x.Parts, useLook)
	case *FieldGet:
		x.X = s.expr(x.X, useLook)
	case *TupleGet:
		x.X = s.expr(x.X, useLook)
	case *StructLit:
		s.exprs(x.Fields, useKeep)
	case *TupleLit:
		s.exprs(x.Elems, useKeep)
	case *ListLit:
		s.exprs(x.Elems, useKeep)
	case *MapLit:
		for _, en := range x.Entries {
			en[0] = s.expr(en[0], useKeep)
			en[1] = s.expr(en[1], useKeep)
		}
	case *AddrOf:
		if isPlaceExpr(x.X) {
			x.X = s.place(x.X) // a pointer to the place itself: nothing copied
		} else {
			x.X = s.expr(x.X, useKeep) // `&e` boxes a copy of e
		}
	case *Deref:
		x.X = s.expr(x.X, useLook)
	case *Downcast:
		x.X = s.expr(x.X, useLook)
	case *Cast, *SomeWrap, *ErrorConvert, *MakeResult:
		// the value goes on inside the result: used as the result is
		s.wrapped(x, u)
		return x
	case *IsNull:
		x.X = s.expr(x.X, useLook)
	case *Unwrap:
		x.X = s.expr(x.X, useLook)
	case *MakeVariant:
		x.Value = s.expr(x.Value, useKeep)
	case *VariantTest:
		x.X = s.expr(x.X, useLook)
	case *VariantCast:
		x.X = s.expr(x.X, useLook)
	case *TypeTest:
		x.X = s.expr(x.X, useLook)
	case *TraitTest:
		x.X = s.expr(x.X, useLook)
	case *TraitCast:
		x.X = s.expr(x.X, useLook)
	case *If:
		x.Cond = s.expr(x.Cond, useLook)
		s.block(x.Then, u)
		s.block(x.Else, u)
		return x
	case *BlockExpr:
		s.block(x.Block, u)
		return x
	case *ResultIsErr:
		x.X = s.expr(x.X, useLook)
	case *ResultValue:
		x.X = s.expr(x.X, useLook)
	case *UnionTest:
		x.X = s.expr(x.X, useLook)
	case *UnionCast:
		x.X = s.expr(x.X, useLook)
	case *RangeLit:
		x.Lo = s.expr(x.Lo, useLook)
		x.Hi = s.expr(x.Hi, useLook)
	case *Elvis:
		x.L = s.expr(x.L, u)
		x.R = s.expr(x.R, u)
		return x
	case *Let:
		iu := useKeep
		if x.Var.Lent {
			iu = useLook
		}
		x.Init = s.expr(x.Init, iu)
		x.Body = s.expr(x.Body, u)
		return x
	case *Match:
		// the subject is only looked at: the arms that bind a part of it
		// keep that part, and are walked as bindings
		x.Init = s.expr(x.Init, useLook)
		for _, arm := range x.Arms {
			if arm.Test != nil {
				arm.Test = s.expr(arm.Test, useLook)
			}
			for _, b := range arm.Binds {
				s.stmt(b)
			}
			if arm.Guard != nil {
				arm.Guard = s.expr(arm.Guard, useLook)
			}
			s.block(arm.Body, u)
		}
		return x
	case *Try:
		x.X = s.expr(x.X, u)
		return x
	case *Throw:
		x.Value = s.expr(x.Value, useKeep)
	case *Box:
		x.X = s.expr(x.X, useKeep)
	case *Closure:
		return s.closure(x)
	case *LazyRead:
		x.Build = s.expr(x.Build, useKeep)
	case *Launch:
		// a task runs alongside its starter: every argument is a second
		// holder, whatever the callee does with it
		if x.Call.Fn.Receiver != nil && len(x.Call.Args) > 0 {
			x.Call.Args[0] = s.expr(x.Call.Args[0], useLook)
			s.exprs(x.Call.Args[1:], useKeep)
		} else {
			s.exprs(x.Call.Args, useKeep)
		}
	case *AwaitTask:
		x.X = s.expr(x.X, useLook)
	case *ScopeBlock:
		s.scope(x, u)
		return x
	case *RaceFirst:
		x.Tasks = s.expr(x.Tasks, useLook)
	case *Race:
		for _, arm := range x.Arms {
			arm.Source = s.expr(arm.Source, useLook)
			if arm.Value != nil {
				arm.Value = s.expr(arm.Value, useKeep)
			}
			s.block(arm.Body, u)
		}
		return x
	}
	return s.leaf(e, u)
}

// wrapped walks the operand of a node that carries a value on unchanged
// (an implicit `Some`, a conversion, an Ok), with the use of the whole.
func (s *sharer) wrapped(e Expr, u use) {
	switch x := e.(type) {
	case *Cast:
		x.X = s.expr(x.X, u)
	case *SomeWrap:
		x.X = s.expr(x.X, u)
	case *ErrorConvert:
		x.X = s.expr(x.X, u)
	case *MakeResult:
		x.Value = s.expr(x.Value, u)
	}
}

func (s *sharer) scope(x *ScopeBlock, u use) {
	if x.On != nil {
		x.On = s.expr(x.On, useLook)
	}
	if x.Workers != nil {
		x.Workers = s.expr(x.Workers, useLook)
	}
	if x.Held != nil {
		x.Held.Init = s.expr(x.Held.Init, useKeep)
	}
	s.block(x.Body, u)
	if x.Held != nil && x.Held.Close != nil {
		x.Held.Close = s.expr(x.Held.Close, useLook)
	}
}

// call walks a direct call: a method's receiver is a pointer to its place,
// nothing copied; an argument for a ReadOnly parameter is lent when nothing
// else can reach it during the call.
func (s *sharer) call(x *Call) {
	params, args := x.Fn.Params, x.Args
	if x.Fn.Receiver != nil && len(args) > 0 {
		args[0] = s.expr(args[0], useLook)
		args = args[1:]
	}
	for i, a := range args {
		u := useKeep
		if i < len(params) && params[i].ReadOnly && lendable(a) {
			u = useLook
		}
		args[i] = s.expr(a, u)
	}
}

// lookArgs are the built-ins whose arguments after the receiver are only
// looked at (a key that is hashed and compared); any other built-in may
// keep what it is given.
var lookArgs = map[string]bool{
	"map.get": true, "map.contains": true, "map.remove": true, "map.ref": true, "map.refOrPanic": true,
	"list.checkMods": true, "list.unlend": true,
}

// builtin walks a built-in: its receiver is read, or changed in its place;
// what it is given otherwise it may keep.
func (s *sharer) builtin(x *Builtin) {
	for i, a := range x.Args {
		switch {
		case i == 0 && collectionWrites[x.Op]:
			x.Args[0] = s.place(a)
		case i == 0 || lookArgs[x.Op]:
			x.Args[i] = s.expr(a, useLook)
		default:
			x.Args[i] = s.expr(a, useKeep)
		}
	}
}

// closure: a lambda shares the variables it captures with its maker, so a
// capture copies nothing — except into a task: a sendable lambda's captured
// collections are marked shared when it is made, so that its maker's next
// change copies them rather than writing under it (D35, D231).
func (s *sharer) closure(x *Closure) Expr {
	ft, _ := x.Type().(*types.Func)
	if ft == nil || !ft.Sendable {
		return x
	}
	var marks []Stmt
	for _, v := range x.Captures {
		if holdsCollection(v.Type, map[types.Type]bool{}) {
			r := ref(v)
			marks = append(marks, &Assign{Target: ref(v), Value: &Share{exprBase{v.Type}, r}})
		}
	}
	if len(marks) == 0 {
		return x
	}
	return &BlockExpr{exprBase{x.Type()}, &Block{Stmts: marks, Value: x, Type: x.Type()}}
}

// changesCaptured reports whether the lambda fn changes a collection in
// the outer variable v it captured — a built-in change, or an element set:
// such a lambda is not sendable (check_lambda.go, D35, D231).
func changesCaptured(fn *Func, v *Var) bool {
	if fn.Body == nil || !holdsCollection(v.Type, map[types.Type]bool{}) {
		return false
	}
	inner := map[*Var]bool{v: true}
	for _, cv := range fn.CapVars {
		if cv.Outer == v {
			inner[cv] = true
		}
	}
	found := false
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Builtin:
			found = found || (collectionWrites[n.Op] && inner[collRoot(n.Args[0])])
		case *Assign:
			_, isElem := n.Target.(*Builtin)
			found = found || (isElem && inner[collRoot(n.Target)])
		}
	})
	return found
}
