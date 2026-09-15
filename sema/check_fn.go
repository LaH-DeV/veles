package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// fnCtx is the per-function checking state.
type fnCtx struct {
	c      *Checker
	fn     *Func
	module *Module
	file   *ast.File
	env    *typeEnv
	subst  map[*types.TypeParam]types.Type
	scope  *Scope
	loops  []*loopFrame
	narrow map[*Var]types.Type

	retType  types.Type // declared return type (never the Result wrapper)
	throws   bool
	errType  types.Type // declared error union, or nil when inferred
	unsafe   int
	selfVar  *Var
	selfMut  bool
	isGlobal bool // checking a global initializer
}

type loopFrame struct {
	hir   *Loop
	label string
}

func (c *Checker) newFnCtx(fn *Func, module *Module, file *ast.File, env *typeEnv, subst map[*types.TypeParam]types.Type) *fnCtx {
	scope := NewScope(module.Imports[file])
	if scope.parent == nil {
		scope = NewScope(module.Scope)
	}
	return &fnCtx{c: c, fn: fn, module: module, file: file, env: env, subst: subst, scope: scope, narrow: map[*Var]types.Type{}}
}

func (f *fnCtx) errorf(span source.Span, format string, args ...any) {
	f.c.errorf(span, format, args...)
}

// resolve turns a syntactic type into a concrete type in this function.
func (f *fnCtx) resolve(t ast.Type) types.Type {
	rt := f.c.resolveType(f.env, t)
	return types.Subst(rt, f.subst)
}

func (f *fnCtx) newVar(name string, t types.Type, mutable bool, span source.Span) *Var {
	f.c.nextVar++
	v := &Var{Name: name, Type: t, Mutable: mutable, ID: f.c.nextVar, Span: span}
	if f.fn != nil {
		f.fn.Locals = append(f.fn.Locals, v)
	}
	return v
}

func (f *fnCtx) newTemp(t types.Type) *Var {
	f.c.nextTmp++
	return f.newVar("$tmp"+itoa(f.c.nextTmp), t, true, source.Span{})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func (f *fnCtx) pushScope() { f.scope = NewScope(f.scope) }
func (f *fnCtx) popScope()  { f.scope = f.scope.parent }

func (f *fnCtx) declareLocal(name string, v *Var, span source.Span) {
	if name == "_" {
		return
	}
	if old := f.scope.LookupLocal(name); old != nil {
		f.errorf(span, "'%s' is already declared in this scope", name)
		return
	}
	f.scope.Insert(&Symbol{Name: name, Kind: SymLocal, Var: v, Span: span})
}

// ---------------------------------------------------------------------------
// function bodies

func (c *Checker) checkBody(fn *Func) {
	t := fn.tmpl
	env := &typeEnv{module: t.Module, file: t.File, tps: map[string]*types.TypeParam{}}
	for _, tp := range t.TypeParams {
		env.tps[tp.Name] = tp
	}
	var owner types.Type
	if t.Owner != nil {
		for k, v := range c.structDecl[t.Owner].tps {
			env.tps[k] = v
		}
		owner = t.Owner
		if len(t.Owner.TypeParams) > 0 {
			var args []types.Type
			for _, tp := range t.Owner.TypeParams {
				args = append(args, fn.subst[tp])
			}
			owner = c.instantiateStruct(t.Owner, args, fn.Span)
		}
	}
	if t.Impl != nil {
		for _, tp := range t.Impl.TypeParams {
			env.tps[tp.Name] = tp
		}
		owner = types.Subst(t.Impl.Target, fn.subst)
	}
	if t.Trait != nil {
		for k, v := range c.traitDecl[t.Trait].tps {
			env.tps[k] = v
		}
		owner = fn.subst[selfParamOf(t.Trait)]
	}
	env.self = owner
	f := c.newFnCtx(fn, t.Module, t.File, env, fn.subst)
	f.retType = fn.Sig.Ret
	f.throws = fn.Sig.Effects.Throws
	if f.throws && t.Decl.Effects.Error != nil {
		f.errType = fn.Sig.Effects.Error
	}
	if t.Decl.Unsafe {
		f.unsafe = 1
	}
	if owner != nil {
		selfType := owner
		if t.Decl.Mut {
			selfType = &types.Pointer{Elem: owner}
		}
		fn.Receiver = f.newVar("self", selfType, t.Decl.Mut, t.Decl.Name.Pos)
		f.selfVar = fn.Receiver
		f.selfMut = t.Decl.Mut
	}
	for i, p := range t.Decl.Params {
		v := f.newVar(p.Name.Name, fn.Sig.Params[i].Type, false, p.Name.Pos)
		fn.Params = append(fn.Params, v)
		f.declareLocal(p.Name.Name, v, p.Name.Pos)
	}
	if t.Decl.ExprBody != nil {
		f.pushScope()
		var body *Block
		if f.retType == nil {
			// `fun f(...) = expr` with no declared type: infer it (§4).
			x := f.checkExpr(t.Decl.ExprBody, nil)
			rt := x.Type()
			if types.IsNever(rt) {
				rt = types.TUnit
			}
			fn.Sig.Ret = rt
			f.retType = rt
			if types.IsUnit(rt) {
				body = &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: types.TUnit}
			} else {
				body = &Block{Stmts: []Stmt{&Return{Value: x}}, Type: types.TNever}
			}
			f.popScope()
			fn.Body = body
			return
		}
		if types.IsUnit(f.retType) {
			x := f.checkExpr(t.Decl.ExprBody, types.TUnit)
			body = &Block{Type: types.TUnit}
			if !types.IsUnit(x.Type()) && !types.IsNever(x.Type()) {
				f.errorf(t.Decl.ExprBody.Span(), "function '%s' returns nothing but its body has type '%s'; add a return type", t.Name, x.Type())
			}
			body.Stmts = append(body.Stmts, &ExprStmt{X: x})
		} else {
			x := f.checkExprTo(t.Decl.ExprBody, f.retType)
			body = &Block{Stmts: []Stmt{&Return{Value: x}}, Type: types.TNever}
		}
		f.popScope()
		fn.Body = body
	} else if t.Decl.Body != nil {
		body := f.checkBlock(t.Decl.Body, f.retType, !types.IsUnit(f.retType))
		if body.Value != nil {
			body.Stmts = append(body.Stmts, &Return{Value: body.Value})
			body.Value = nil
		} else if !types.IsUnit(f.retType) && !types.IsNever(body.Type) {
			f.errorf(t.Decl.Body.Pos, "missing return: function '%s' must return a value of type '%s'", t.Name, f.retType)
		}
		fn.Body = body
	}
}

// selfParamOf returns a synthetic type parameter standing for Self in a
// trait default method template.
var selfParams = map[*types.Trait]*types.TypeParam{}

func selfParamOf(t *types.Trait) *types.TypeParam {
	if p, ok := selfParams[t]; ok {
		return p
	}
	p := &types.TypeParam{Name: "Self", Owner: t.Name}
	selfParams[t] = p
	return p
}

// ---------------------------------------------------------------------------
// globals

func (c *Checker) checkGlobal(g *Global) {
	d := c.globals[g]
	m := c.globalMod[g]
	file := c.globalFile[g]
	env := &typeEnv{module: m, file: file, tps: map[string]*types.TypeParam{}}
	f := c.newFnCtx(nil, m, file, env, nil)
	f.isGlobal = true
	var declared types.Type
	if d.Type != nil {
		declared = f.resolve(d.Type)
	}
	if d.Value == nil {
		c.errorf(d.Pos, "module-level '%s' needs an initializer", d.Kind)
		return
	}
	var init Expr
	if declared != nil {
		init = f.checkExprTo(d.Value, declared)
	} else {
		init = f.checkExpr(d.Value, nil)
		declared = init.Type()
	}
	if d.Kind == ast.BindConst && !isConstExpr(init) {
		c.errorf(d.Value.Span(), "'const' requires a compile-time constant (§4: 'const' is reserved for compile-time constants); use 'val' for runtime values")
	}
	if types.IsNever(declared) || types.IsUnit(declared) {
		c.errorf(d.Name.Pos, "a global cannot have type '%s'", declared)
	}
	g.Type = declared
	g.Init = init
}

func isConstExpr(e Expr) bool {
	switch e := e.(type) {
	case *IntConst, *FloatConst, *BoolConst, *StringConst:
		return true
	case *Unary:
		return isConstExpr(e.X)
	case *Binary:
		return isConstExpr(e.L) && isConstExpr(e.R)
	}
	return false
}

// ---------------------------------------------------------------------------
// blocks and statements

// checkBlock checks a block. When wantValue is set and the last statement
// is an expression, it becomes the block's value checked against expected.
func (f *fnCtx) checkBlock(b *ast.Block, expected types.Type, wantValue bool) *Block {
	f.pushScope()
	defer f.popScope()
	out := &Block{Type: types.TUnit}
	terminated := false
	for i, s := range b.Stmts {
		if terminated {
			f.errorf(s.Span(), "unreachable code")
			break
		}
		last := i == len(b.Stmts)-1
		if last && wantValue {
			if es, ok := s.(*ast.ExprStmt); ok {
				var x Expr
				if expected != nil {
					x = f.checkExprTo(es.X, expected)
				} else {
					x = f.checkExpr(es.X, nil)
				}
				if types.IsNever(x.Type()) {
					out.Stmts = append(out.Stmts, &ExprStmt{X: x})
					out.Type = types.TNever
				} else {
					out.Value = x
					out.Type = x.Type()
				}
				continue
			}
		}
		stmts, term := f.checkStmt(s)
		out.Stmts = append(out.Stmts, stmts...)
		if term {
			terminated = true
			out.Type = types.TNever
		}
	}
	return out
}

// checkStmt lowers one statement; term reports whether control cannot
// continue past it.
func (f *fnCtx) checkStmt(s ast.Stmt) (stmts []Stmt, term bool) {
	switch s := s.(type) {
	case *ast.ValStmt:
		return f.checkValStmt(s), false
	case *ast.ExprStmt:
		x := f.checkExpr(s.X, types.TUnit)
		if isResultType(x.Type()) {
			f.errorf(s.X.Span(), "unused Result: the call may fail; use 'try' to propagate the error or 'when' to handle it (D4)")
		}
		return []Stmt{&ExprStmt{X: x}}, types.IsNever(x.Type())
	case *ast.AssignStmt:
		return f.checkAssign(s), false
	case *ast.ReturnStmt:
		return f.checkReturn(s), true
	case *ast.BreakStmt:
		lp := f.findLoop(s.Label, s.Pos, "break")
		if lp == nil {
			return nil, true
		}
		return []Stmt{&Break{Loop: lp}}, true
	case *ast.ContinueStmt:
		lp := f.findLoop(s.Label, s.Pos, "continue")
		if lp == nil {
			return nil, true
		}
		return []Stmt{&Continue{Loop: lp}}, true
	case *ast.LoopStmt:
		return f.checkLoop(s), false
	case *ast.Block:
		b := f.checkBlock(s, nil, false)
		return []Stmt{b}, types.IsNever(b.Type)
	case *ast.WithStmt:
		f.errorf(s.Pos, "'with' resource blocks are not implemented yet in the bootstrap compiler (D43)")
		return nil, false
	case *ast.ScopeStmt:
		f.errorf(s.Pos, "structured concurrency ('scope') is not implemented yet in the bootstrap compiler (build plan stage 4)")
		return nil, false
	case *ast.FunStmt:
		f.errorf(s.Fun.Pos, "local functions are not supported; use a lambda or a module-level function")
		return nil, false
	case *ast.BadStmt:
		return nil, false
	}
	f.errorf(s.Span(), "unsupported statement")
	return nil, false
}

func (f *fnCtx) findLoop(label *ast.Ident, span source.Span, what string) *Loop {
	if len(f.loops) == 0 {
		f.errorf(span, "'%s' outside of a loop", what)
		return nil
	}
	if label == nil {
		return f.loops[len(f.loops)-1].hir
	}
	for i := len(f.loops) - 1; i >= 0; i-- {
		if f.loops[i].label == label.Name {
			return f.loops[i].hir
		}
	}
	f.errorf(label.Pos, "no enclosing loop labelled '%s'", label.Name)
	return nil
}

func (f *fnCtx) checkValStmt(s *ast.ValStmt) []Stmt {
	mutable := s.Kind == ast.BindVar
	if s.Kind == ast.BindConst {
		f.errorf(s.Pos, "'const' is only allowed at module level; use 'val' for a local immutable binding")
	}
	b := s.Binding
	if b.Name != nil {
		var declared types.Type
		if b.Type != nil {
			declared = f.resolve(b.Type)
		}
		var init Expr
		if s.Value != nil {
			if declared != nil {
				init = f.checkExprTo(s.Value, declared)
			} else {
				init = f.checkExpr(s.Value, nil)
				declared = init.Type()
				if types.IsUnit(declared) {
					f.errorf(s.Value.Span(), "cannot bind a value of type '()'")
				}
				if types.IsNever(declared) {
					f.errorf(s.Value.Span(), "initializer never produces a value")
				}
			}
		}
		if declared == nil {
			declared = types.TInvalid
		}
		v := f.newVar(b.Name.Name, declared, mutable, b.Name.Pos)
		f.declareLocal(b.Name.Name, v, b.Name.Pos)
		return []Stmt{&VarDecl{Var: v, Init: init}}
	}
	// tuple destructuring (D37)
	init := f.checkExpr(s.Value, nil)
	tt, ok := init.Type().(*types.Tuple)
	if !ok {
		if !types.IsInvalid(init.Type()) {
			f.errorf(s.Value.Span(), "cannot destructure a value of type '%s'; only tuples destructure positionally (D37)", init.Type())
		}
		return nil
	}
	if len(tt.Elems) != len(b.Tuple) {
		f.errorf(b.Pos, "tuple has %d elements but %d names are bound", len(tt.Elems), len(b.Tuple))
		return nil
	}
	tmp := f.newTemp(tt)
	stmts := []Stmt{&VarDecl{Var: tmp, Init: init}}
	for i, el := range b.Tuple {
		if el.Name == nil {
			f.errorf(el.Pos, "nested tuple destructuring is not supported yet")
			continue
		}
		et := tt.Elems[i]
		if el.Type != nil {
			want := f.resolve(el.Type)
			if !types.Identical(want, et) {
				f.errorf(el.Pos, "element %d has type '%s', not '%s'", i, et, want)
			}
		}
		v := f.newVar(el.Name.Name, et, mutable, el.Name.Pos)
		f.declareLocal(el.Name.Name, v, el.Name.Pos)
		stmts = append(stmts, &VarDecl{Var: v, Init: &TupleGet{exprBase{et}, &VarRef{exprBase{tt}, tmp}, i}})
	}
	return stmts
}

func (f *fnCtx) checkReturn(s *ast.ReturnStmt) []Stmt {
	if f.isGlobal {
		f.errorf(s.Pos, "'return' outside of a function")
		return nil
	}
	if s.Value == nil {
		if !types.IsUnit(f.retType) {
			f.errorf(s.Pos, "missing return value of type '%s'", f.retType)
		}
		return []Stmt{&Return{}}
	}
	if types.IsUnit(f.retType) {
		x := f.checkExpr(s.Value, types.TUnit)
		if types.IsNever(x.Type()) {
			return []Stmt{&ExprStmt{X: x}}
		}
		if !types.IsUnit(x.Type()) {
			f.errorf(s.Value.Span(), "function returns nothing but a value of type '%s' is returned", x.Type())
		}
		return []Stmt{&ExprStmt{X: x}, &Return{}}
	}
	x := f.checkExprTo(s.Value, f.retType)
	if types.IsNever(x.Type()) {
		return []Stmt{&ExprStmt{X: x}}
	}
	return []Stmt{&Return{Value: x}}
}

// checkAssign handles `target = value` and compound assignment.
func (f *fnCtx) checkAssign(s *ast.AssignStmt) []Stmt {
	target, root := f.checkLValue(s.Target, true)
	if target == nil {
		f.checkExpr(s.Value, nil)
		return nil
	}
	var value Expr
	var rawType types.Type
	if s.Op == lexer.Assign {
		raw := f.checkExpr(s.Value, target.Type())
		rawType = raw.Type()
		value = f.coerce(raw, target.Type(), s.Value.Span())
	} else {
		op := BinOpFromToken(s.Op)
		rhs := f.checkExprTo(s.Value, target.Type())
		value = f.makeBinary(op, target, rhs, s.Pos)
	}
	// Assignment re-narrows the variable to the assigned value's type (a
	// non-null value smart-casts a nullable var), or clears the narrowing.
	if root != nil {
		if _, ok := target.(*VarRef); ok {
			delete(f.narrow, root)
			if rawType != nil && !types.Identical(rawType, root.Type) && !types.IsNever(rawType) && !types.IsInvalid(rawType) && f.assignableTo(rawType, root.Type) {
				f.narrow[root] = rawType
			}
		}
	}
	return []Stmt{&Assign{Target: target, Value: value}}
}

// checkLValue checks an assignable expression. It returns the lvalue and,
// when the place is rooted in a local or global variable (not through a
// pointer), that variable. With mutate=true it enforces D11/D22 mutability.
func (f *fnCtx) checkLValue(e ast.Expr, mutate bool) (Expr, *Var) {
	switch e := e.(type) {
	case *ast.NameExpr:
		sym := f.scope.Lookup(e.Name)
		if sym == nil {
			f.errorf(e.Pos, "unknown name '%s'", e.Name)
			return nil, nil
		}
		switch sym.Kind {
		case SymLocal:
			if mutate && !sym.Var.Mutable {
				f.errorf(e.Pos, "cannot assign to '%s': it is a 'val'; declare it with 'var' (D11)", e.Name)
			}
			return &VarRef{exprBase{sym.Var.Type}, sym.Var}, sym.Var
		case SymGlobal:
			g := sym.Global
			if mutate && !g.Mutable {
				f.errorf(e.Pos, "cannot assign to '%s': it is not a 'var'", e.Name)
			}
			v := f.globalVar(g)
			return &VarRef{exprBase{v.Type}, v}, v
		}
		f.errorf(e.Pos, "'%s' is not assignable", e.Name)
		return nil, nil
	case *ast.SelfExpr:
		if f.selfVar == nil {
			f.errorf(e.Pos, "'self' outside of a method")
			return nil, nil
		}
		if f.selfMut {
			return &Deref{exprBase{f.selfVar.Type.(*types.Pointer).Elem}, &VarRef{exprBase{f.selfVar.Type}, f.selfVar}}, nil
		}
		if mutate {
			f.errorf(e.Pos, "cannot mutate 'self' in a non-'mut' method; declare the method 'mut fun' (D22)")
		}
		return &VarRef{exprBase{f.selfVar.Type}, f.selfVar}, f.selfVar
	case *ast.MemberExpr:
		if e.X == nil {
			break
		}
		if e.Safe {
			f.errorf(e.Pos, "cannot assign through '?.'")
			return nil, nil
		}
		// Through a pointer, mutability is not gated by the binding (D11).
		base, root := f.checkLValue(e.X, false)
		if base == nil {
			// maybe an rvalue like a call result
			x := f.checkExpr(e.X, nil)
			base = x
			root = nil
		}
		bt := base.Type()
		if p, ok := bt.(*types.Pointer); ok {
			if p.Raw && f.unsafe == 0 {
				f.errorf(e.Pos, "dereferencing a raw pointer requires an 'unsafe' block (D44)")
			}
			base = &Deref{exprBase{p.Elem}, base}
			bt = p.Elem
			root = nil
		} else if mutate && root != nil && root == f.selfVar && !f.selfMut {
			f.errorf(e.Pos, "cannot mutate 'self.%s' in a non-'mut' method; declare the method 'mut fun' (D22)", e.Name.Name)
		} else if mutate && root != nil && !root.Mutable {
			f.errorf(e.Pos, "cannot assign to field '%s' of '%s': it is a 'val' (D11/D22)", e.Name.Name, root.Name)
		} else if mutate && root == nil {
			if _, isDeref := base.(*Deref); !isDeref {
				if _, isField := base.(*FieldGet); !isField {
					f.errorf(e.Pos, "cannot assign to a field of a temporary value")
				}
			}
		}
		st, ok := bt.(*types.Struct)
		if !ok {
			f.errorf(e.Pos, "type '%s' has no field '%s'", bt, e.Name.Name)
			return nil, nil
		}
		fld := f.lookupField(st, e.Name.Name, e.Name.Pos)
		if fld == nil {
			return nil, nil
		}
		return &FieldGet{exprBase{fld.Type}, base, fld.Index, fld.Name}, root
	case *ast.IndexExpr:
		x := f.checkExpr(e.X, nil)
		lt, ok := x.Type().(*types.List)
		if !ok {
			f.errorf(e.Pos, "cannot index-assign into '%s'", x.Type())
			return nil, nil
		}
		if !lt.Mutable {
			f.errorf(e.Pos, "cannot assign into an immutable List; use MutableList (D25)")
		}
		idx := f.indexValue(e.Index)
		return &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{x, idx}, e.Pos}, nil
	case *ast.UnaryExpr:
		if e.Op == lexer.Star {
			p := f.checkExpr(e.X, nil)
			pt, ok := p.Type().(*types.Pointer)
			if !ok {
				f.errorf(e.Pos, "cannot dereference '%s'", p.Type())
				return nil, nil
			}
			if pt.Raw && f.unsafe == 0 {
				f.errorf(e.Pos, "dereferencing a raw pointer requires an 'unsafe' block (D44)")
			}
			return &Deref{exprBase{pt.Elem}, p}, nil
		}
	}
	f.errorf(e.Span(), "expression is not assignable")
	return nil, nil
}

func (f *fnCtx) lookupField(st *types.Struct, name string, span source.Span) *types.Field {
	f.c.resolveStruct(templateOf(st))
	for _, fld := range st.Fields {
		if fld.Name == name {
			if !fld.Pub && st.Module != f.module.prefix() {
				f.errorf(span, "field '%s' of '%s' is private to module '%s' (M5)", name, st.Name, st.Module)
			}
			return fld
		}
	}
	f.errorf(span, "'%s' has no field '%s'", st, name)
	return nil
}

// globalVar returns the Var standing for a module-level binding.
var globalVars = map[*Global]*Var{}

func (f *fnCtx) globalVar(g *Global) *Var {
	if v, ok := globalVars[g]; ok {
		if v.Type == nil || types.IsInvalid(v.Type) {
			v.Type = g.Type
		}
		return v
	}
	if g.Type == nil {
		// referenced before its own initializer was checked (e.g. from another global)
		f.c.checkGlobal(g)
		if g.Type == nil {
			g.Type = types.TInvalid
		}
	}
	v := &Var{Name: g.Display, Type: g.Type, Mutable: g.Mutable, IsGlobal: true, Global: g, Span: g.Span}
	globalVars[g] = v
	return v
}

// ---------------------------------------------------------------------------
// loops (D42 desugaring for ranges and lists)

func (f *fnCtx) checkLoop(s *ast.LoopStmt) []Stmt {
	f.c.nextLoop++
	lp := &Loop{ID: f.c.nextLoop}
	label := ""
	if s.Label != nil {
		label = s.Label.Name
	}
	lp.Label = label
	// Narrowing established outside the loop may be invalidated by
	// assignments in the body; drop it for assigned variables.
	f.invalidateAssigned(s.Body)

	var pre []Stmt
	f.pushScope()
	defer f.popScope()
	switch {
	case s.Var != nil:
		iter := f.checkExpr(s.Iter, nil)
		switch it := iter.Type().(type) {
		case *types.Range:
			// var i = lo; val hi = hi; loop (i < hi) { val x = i; body; post: i += 1 }
			if !types.IsInteger(it.Elem) {
				f.errorf(s.Iter.Span(), "cannot iterate a range of '%s'", it.Elem)
			}
			rangeTmp := f.newTemp(it)
			pre = append(pre, &VarDecl{Var: rangeTmp, Init: iter})
			idx := f.newTemp(it.Elem)
			pre = append(pre, &VarDecl{Var: idx, Init: &FieldGet{exprBase{it.Elem}, &VarRef{exprBase{it}, rangeTmp}, 0, "lo"}})
			hi := f.newTemp(it.Elem)
			pre = append(pre, &VarDecl{Var: hi, Init: &FieldGet{exprBase{it.Elem}, &VarRef{exprBase{it}, rangeTmp}, 1, "hi"}})
			// condition: inclusive ? i <= hi : i < hi — decided at runtime
			// through the range's flag so both forms share one lowering.
			incl := &FieldGet{exprBase{types.TBool}, &VarRef{exprBase{it}, rangeTmp}, 2, "inclusive"}
			lt := &Binary{exprBase{types.TBool}, OpLt, &VarRef{exprBase{it.Elem}, idx}, &VarRef{exprBase{it.Elem}, hi}, s.Pos}
			le := &Binary{exprBase{types.TBool}, OpLe, &VarRef{exprBase{it.Elem}, idx}, &VarRef{exprBase{it.Elem}, hi}, s.Pos}
			lp.Cond = &If{exprBase{types.TBool}, incl, &Block{Value: le, Type: types.TBool}, &Block{Value: lt, Type: types.TBool}}
			lp.Post = []Stmt{&Assign{Target: &VarRef{exprBase{it.Elem}, idx}, Value: &Binary{exprBase{it.Elem}, OpWrapAdd, &VarRef{exprBase{it.Elem}, idx}, &IntConst{exprBase{it.Elem}, 1, false}, s.Pos}}}
			v := f.bindLoopVar(s.Var, it.Elem)
			f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
			body := f.checkBlock(s.Body, nil, false)
			f.loops = f.loops[:len(f.loops)-1]
			body.Stmts = append([]Stmt{&VarDecl{Var: v, Init: &VarRef{exprBase{it.Elem}, idx}}}, body.Stmts...)
			lp.Body = body
		case *types.List:
			listTmp := f.newTemp(it)
			pre = append(pre, &VarDecl{Var: listTmp, Init: iter})
			idx := f.newTemp(types.TI64)
			pre = append(pre, &VarDecl{Var: idx, Init: &IntConst{exprBase{types.TI64}, 0, false}})
			lenExpr := &Builtin{exprBase{types.TI64}, "list.len", []Expr{&VarRef{exprBase{it}, listTmp}}, s.Pos}
			lp.Cond = &Binary{exprBase{types.TBool}, OpLt, &VarRef{exprBase{types.TI64}, idx}, lenExpr, s.Pos}
			lp.Post = []Stmt{&Assign{Target: &VarRef{exprBase{types.TI64}, idx}, Value: &Binary{exprBase{types.TI64}, OpWrapAdd, &VarRef{exprBase{types.TI64}, idx}, &IntConst{exprBase{types.TI64}, 1, false}, s.Pos}}}
			v := f.bindLoopVar(s.Var, it.Elem)
			f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
			body := f.checkBlock(s.Body, nil, false)
			f.loops = f.loops[:len(f.loops)-1]
			get := &Builtin{exprBase{it.Elem}, "list.get", []Expr{&VarRef{exprBase{it}, listTmp}, &VarRef{exprBase{types.TI64}, idx}}, s.Pos}
			body.Stmts = append([]Stmt{&VarDecl{Var: v, Init: get}}, body.Stmts...)
			lp.Body = body
		default:
			if !types.IsInvalid(iter.Type()) {
				f.errorf(s.Iter.Span(), "cannot iterate over '%s': only ranges and lists are iterable in the bootstrap compiler (Iterable/Iterator traits, D42, come later)", iter.Type())
			}
			f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
			lp.Body = f.checkBlock(s.Body, nil, false)
			f.loops = f.loops[:len(f.loops)-1]
			return nil
		}
	case s.Cond != nil:
		cond := f.checkExprTo(s.Cond, types.TBool)
		lp.Cond = cond
		saved := f.saveNarrow()
		tf, _ := f.condFacts(s.Cond, cond)
		f.applyFacts(tf)
		f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
		lp.Body = f.checkBlock(s.Body, nil, false)
		f.loops = f.loops[:len(f.loops)-1]
		f.restoreNarrow(saved)
	default:
		f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
		lp.Body = f.checkBlock(s.Body, nil, false)
		f.loops = f.loops[:len(f.loops)-1]
	}
	return append(pre, lp)
}

func (f *fnCtx) bindLoopVar(b *ast.Binding, t types.Type) *Var {
	if b.Name == nil {
		f.errorf(b.Pos, "tuple destructuring in loops is not supported yet")
		return f.newTemp(t)
	}
	if b.Type != nil {
		want := f.resolve(b.Type)
		if !types.Identical(want, t) {
			f.errorf(b.Pos, "loop variable has type '%s', not '%s'", t, want)
		}
	}
	v := f.newVar(b.Name.Name, t, false, b.Name.Pos)
	f.declareLocal(b.Name.Name, v, b.Name.Pos)
	return v
}

// invalidateAssigned drops narrowing for variables assigned inside a block.
func (f *fnCtx) invalidateAssigned(b *ast.Block) {
	var walk func(s ast.Stmt)
	walk = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.AssignStmt:
			if n, ok := s.Target.(*ast.NameExpr); ok {
				if sym := f.scope.Lookup(n.Name); sym != nil && sym.Kind == SymLocal {
					delete(f.narrow, sym.Var)
				}
			}
		case *ast.Block:
			for _, x := range s.Stmts {
				walk(x)
			}
		case *ast.LoopStmt:
			walk(s.Body)
		case *ast.ExprStmt:
			if ie, ok := s.X.(*ast.IfExpr); ok {
				walk(ie.Then)
				if ie.Else != nil {
					walk(ie.Else)
				}
			}
		}
	}
	walk(b)
}

func (f *fnCtx) warnf(span source.Span, format string, args ...any) {
	f.c.warnf(span, format, args...)
}
