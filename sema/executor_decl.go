package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Declared executors (D150): `executor gl(priority: …) { state; functions }`.
//
// The executor is a module global holding an Executor, started with the
// module's globals by the prelude's startExecutor. Its state is globals of
// its own, reached by their short names only from its code — its
// functions, the initializers of its state, and the lambdas in them that
// cannot leave it — and initialised on its thread (Executor.Init). Its
// functions are templates whose bodies see that state; a call from its
// own code is an ordinary call, and one from anywhere else runs the
// function on the executor: `gl.texture(5)` is checked as
// `gl.run(() => gl.texture($arg))`, the arguments evaluated first, here.

// execInfo is one `executor` declaration.
type execInfo struct {
	decl   *ast.ExecutorDecl
	module *Module
	file   *ast.File
	single bool    // one thread: it may hold state, and its children stay on it
	global *Global // the executor value
	scope  *Scope  // its state and functions by their short names
	state  []*Global
	byName map[string]*Symbol
}

func (c *Checker) declareExecutor(m *Module, file *ast.File, d *ast.ExecutorDecl) {
	info := &execInfo{decl: d, module: m, file: file, single: true, byName: map[string]*Symbol{}}
	parent := m.Imports[file]
	if parent == nil {
		parent = m.Scope
	}
	info.scope = NewScope(parent)
	pos := d.Name.Pos
	args := []ast.Arg{
		{Name: &ast.Ident{Name: "name", Pos: pos}, Value: &ast.StringLit{Parts: []ast.StringPart{{Text: d.Name.Name}}, Pos: pos}},
	}
	single := ast.Arg{Name: &ast.Ident{Name: "single", Pos: pos}}
	args = append(args, single)
	for _, a := range d.Args {
		if a.Name == nil {
			c.errorf(a.Value.Span(), "an executor's arguments are named: 'threads:', 'priority:' or 'cpus:' (D150)")
			continue
		}
		switch a.Name.Name {
		case "threads":
			info.single = false
		case "priority", "cpus":
		case "name":
			c.errorf(a.Name.Pos, "an executor is named by its declaration: its threads are named '%s' (D150)", d.Name.Name)
			continue
		default:
			c.errorf(a.Name.Pos, "an executor takes 'threads:', 'priority:' and 'cpus:', not '%s:' (D150)", a.Name.Name)
			continue
		}
		args = append(args, a)
	}
	args[1].Value = &ast.BoolLit{Value: info.single, Pos: pos}
	start := &ast.CallExpr{Fun: &ast.PreludeName{Name: "startExecutor", Pos: pos}, Args: args, Pos: d.Pos}
	vd := &ast.ValDecl{Doc: d.Doc, Pub: d.Pub, Internal: d.Internal, Kind: ast.BindVal, Name: d.Name, Value: start, Pos: d.Pos}
	g := &Global{Name: m.prefix() + "." + d.Name.Name, Display: d.Name.Name, Span: d.Name.Pos}
	c.globals[g] = vd
	c.globalMod[g] = m
	c.globalFile[g] = file
	c.execOfGlobal[g] = info
	info.global = g
	c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymGlobal, Pub: d.Pub, Internal: d.Internal, Module: m, Span: d.Name.Pos, Global: g})
	add := func(sym *Symbol) bool {
		if old, dup := info.byName[sym.Name]; dup {
			c.errorf(sym.Span, "'%s' is already declared in executor '%s' (at %s)", sym.Name, d.Name.Name, old.Span)
			return false
		}
		info.byName[sym.Name] = sym
		info.scope.Insert(sym)
		return true
	}
	for _, s := range d.State {
		if !info.single {
			c.errorf(s.Name.Pos, "a pool runs its functions on several threads at once, so it holds no state of its own: keep shared state behind a 'Mutex' at module level, or declare an executor of one thread (D150)")
			continue
		}
		c.attrsOf(s.Attrs, "value")
		gs := &Global{Name: m.prefix() + "." + d.Name.Name + "." + s.Name.Name, Display: d.Name.Name + "." + s.Name.Name, Mutable: s.Kind == ast.BindVar, Span: s.Name.Pos}
		if !add(&Symbol{Name: s.Name.Name, Kind: SymGlobal, Pub: s.Pub, Internal: s.Internal, Module: m, Span: s.Name.Pos, Global: gs}) {
			continue
		}
		c.globals[gs] = s
		c.globalMod[gs] = m
		c.globalFile[gs] = file
		c.stateOwner[gs] = info
		info.state = append(info.state, gs)
	}
	for _, fn := range d.Funs {
		t := c.newTemplate(m, file, fn, nil, nil)
		t.Mangled = m.prefix() + "." + d.Name.Name + "." + fn.Name.Name
		t.Exec = info
		add(&Symbol{Name: fn.Name.Name, Kind: SymFunc, Pub: fn.Pub, Internal: fn.Internal, Module: m, Span: fn.Name.Pos, Func: t})
	}
	c.execs = append(c.execs, info)
}

// executorsHIR records the executors of this round for codegen: each one's
// state, initialised by a plain function the runtime runs on its thread.
func (c *Checker) executorsHIR() {
	for _, info := range c.execs {
		x := &Executor{Name: info.global.Name, Value: info.global, Single: info.single, State: info.state}
		if len(info.state) > 0 {
			env := &typeEnv{module: info.module, file: info.file, tps: map[string]*types.TypeParam{}}
			f := c.newFnCtx(nil, info.module, info.file, env, nil)
			body := &Block{Type: types.TUnit}
			for _, g := range info.state {
				g.Exec = x
				if g.Init == nil || g.Type == nil || types.IsInvalid(g.Type) {
					continue
				}
				v := f.globalVar(g)
				body.Stmts = append(body.Stmts, &Assign{Target: &VarRef{exprBase{v.Type}, v}, Value: g.Init})
			}
			body.Stmts = append(body.Stmts, &Return{})
			body.Type = types.TNever
			x.Init = &Func{Name: info.global.Name + ".$init", Display: info.decl.Name.Name + " (its state)", Body: body, checked: true,
				Sig: &types.Func{Ret: types.TUnit}}
			c.funcs = append(c.funcs, x.Init)
		}
		c.prog.Executors = append(c.prog.Executors, x)
	}
}

// execRef: x names a declared executor — `gl`, or `mod.gl` from another
// module — or nil.
func (f *fnCtx) execRef(x ast.Expr) *execInfo {
	switch x := x.(type) {
	case *ast.NameExpr:
		if sym := f.lookup(x.Name); sym != nil && sym.Kind == SymGlobal {
			return f.c.execOfGlobal[sym.Global]
		}
	case *ast.MemberExpr:
		n, ok := x.X.(*ast.NameExpr)
		if !ok {
			return nil
		}
		if sym := f.lookup(n.Name); sym != nil && sym.Kind == SymModule {
			if member := sym.Mod.Scope.LookupLocal(x.Name.Name); member != nil && member.Kind == SymGlobal && memberVisible(sym, member, f.module) {
				return f.c.execOfGlobal[member.Global]
			}
		}
	}
	return nil
}

// checkStateUse refuses an executor's state named outside its code: a
// lambda that may run on any thread, inside one of its functions (D150).
// (Anywhere else the state is not in scope at all.)
func (f *fnCtx) checkStateUse(g *Global, span source.Span) {
	info := f.c.stateOwner[g]
	if info == nil || f.exec == info {
		return
	}
	name := info.decl.Name.Name
	f.errorf(span, "'%s' is the state of executor '%s', and only %s's own code names it: this function may run on another thread (a 'sendable' one may run on any); call a function of %s for it (D150)", g.Display, name, name, name)
}

// checkExecFunc refuses one of an executor's functions called by its
// short name from a lambda in its code that may run elsewhere: the call
// would not run on the executor (D150).
func (f *fnCtx) checkExecFunc(t *FuncTemplate, span source.Span) {
	if t.Exec == nil || f.exec == t.Exec {
		return
	}
	name := t.Exec.decl.Name.Name
	f.errorf(span, "'%s' runs on executor '%s', and this function may run on another thread: call it as '%s.%s(…)', which goes to %s (D150)", t.Name, name, name, t.Name, name)
}

// execCall checks `gl.member(args)` on a declared executor; false leaves
// the call to the ordinary method lookup (`run`, `runBlocking`, …).
func (f *fnCtx) execCall(info *execInfo, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) (Expr, bool) {
	name := callee.Name.Name
	sym := info.byName[name]
	exec := info.decl.Name.Name
	if sym == nil {
		switch name {
		case "close":
			f.errorf(callee.Name.Pos, "executor '%s' is declared: it runs for the whole program and is not closed by code (D150)", exec)
			f.checkArgsLoosely(e.Args)
			return bad(), true
		case "run", "runBlocking":
			// the lambda handed straight to it runs as the executor's code
			if len(e.Args) == 1 {
				if l, ok := e.Args[0].Value.(*ast.LambdaExpr); ok {
					savedL, savedE := f.runOnLambda, f.runOnExec
					f.runOnLambda, f.runOnExec = l, info
					x := f.methodCall(callee, typeArgs, e, want)
					f.runOnLambda, f.runOnExec = savedL, savedE
					return x, true
				}
			}
		}
		return nil, false
	}
	f.c.refSym(callee.Name.Pos, sym)
	if !declVisible(sym.Pub, sym.Internal, sym.Exported, info.module.Path, f.module.Path) {
		f.errorf(callee.Name.Pos, "'%s.%s' is private to module '%s'%s (M5)", exec, name, info.module.Path, privateHint(info.module.Path))
		f.checkArgsLoosely(e.Args)
		return bad(), true
	}
	if sym.Kind == SymGlobal {
		if f.exec != info {
			f.errorf(callee.Name.Pos, "'%s' is the state of executor '%s': only %s's own code names it; add a function to %s for what you need (D150)", name, exec, exec, exec)
		} else {
			f.errorf(callee.Name.Pos, "'%s.%s' is state, not a function", exec, name)
		}
		f.checkArgsLoosely(e.Args)
		return bad(), true
	}
	if f.exec == info {
		return f.callSymbol(sym, name, typeArgs, e, want), true // its own code: a plain call
	}
	return f.execHop(info, sym, callee, e, want), true
}

// execHop: a function of executor info called from code that is not its
// own runs on the executor, and the caller suspends until it returns. The
// arguments are evaluated here, in order, and cross to it (Sendable).
func (f *fnCtx) execHop(info *execInfo, sym *Symbol, callee *ast.MemberExpr, e *ast.CallExpr, want types.Type) Expr {
	t := sym.Func
	f.c.resolveSignature(t)
	exec := info.decl.Name.Name
	var params []types.Param
	if t.Sig != nil && len(t.TypeParams) == 0 {
		params = t.Sig.Params
	}
	var pre []Stmt
	args := make([]ast.Arg, len(e.Args))
	for i, a := range e.Args {
		args[i] = a
		var want types.Type
		switch {
		case a.Name != nil:
			for _, p := range params {
				if p.Name == a.Name.Name {
					want = p.Type
				}
			}
		case len(params) > 0 && params[len(params)-1].Variadic && i >= len(params)-1:
			// the trailing arguments of a variadic parameter: its elements,
			// or `xs...`, the list itself (D123)
			want = params[len(params)-1].Type
			if lt, ok := want.(*types.List); ok && !a.Spread {
				want = lt.Elem
			}
		case i < len(params) && !a.Spread:
			want = params[i].Type
		}
		var x Expr
		if want != nil {
			x = f.checkExprTo(a.Value, want)
		} else {
			x = f.checkExpr(a.Value, nil)
		}
		if types.IsInvalid(x.Type()) {
			continue
		}
		if !sendable(x.Type()) {
			f.errorf(a.Value.Span(), "argument of type '%s' is not Sendable, and this call runs '%s' on executor '%s', another thread; pass immutable data (a List, not a MutableList: '.toList()') or share state behind a 'Mutex' (D35, D150)", x.Type(), t.Name, exec)
			continue
		}
		v, decl := f.hidden("arg", x, false)
		pre = append(pre, decl)
		args[i].Value = nameOf(v, a.Value.Span())
	}
	inner := &ast.CallExpr{Fun: callee, TypeArgs: e.TypeArgs, Args: args, Pos: e.Pos}
	var body ast.Expr = inner
	if t.Decl.Effects.Throws {
		body = &ast.TryExpr{X: inner, Pos: e.Pos}
	}
	lambda := &ast.LambdaExpr{Body: body, Pos: e.Pos}
	// the prelude's hop(gl, f): gl.run's body, as a free function, so that
	// `async gl.f(x)` launches it as any call; the lambda is gl's code
	hop := &ast.CallExpr{Fun: &ast.PreludeName{Name: "hop", Pos: callee.Name.Pos}, Args: []ast.Arg{{Value: callee.X}, {Value: lambda}}, Pos: e.Pos}
	savedL, savedE, savedLaunch := f.runOnLambda, f.runOnExec, f.launching
	f.runOnLambda, f.runOnExec = lambda, info
	if f.launching == e {
		f.launching = hop // the launch's call is now this one
	}
	x := f.checkExpr(hop, want)
	f.runOnLambda, f.runOnExec, f.launching = savedL, savedE, savedLaunch
	if len(pre) == 0 {
		return x
	}
	if f.launching == e {
		// the launch must stay a plain call: its arguments are evaluated
		// before it, in this task (launch wraps them around the Launch)
		f.launchPre = append(f.launchPre, pre...)
		return x
	}
	return &BlockExpr{exprBase{x.Type()}, &Block{Stmts: pre, Value: x, Type: x.Type()}}
}

// execMember checks `gl.member` used as a value: an executor's state and
// functions are not reached from outside its code that way (D150); nil
// leaves the rest to the ordinary field lookup.
func (f *fnCtx) execMember(info *execInfo, e *ast.MemberExpr) Expr {
	sym := info.byName[e.Name.Name]
	if sym == nil {
		return nil
	}
	exec := info.decl.Name.Name
	f.c.refSym(e.Name.Pos, sym)
	if f.exec == info {
		if sym.Kind == SymGlobal {
			return f.globalValue(sym.Global)
		}
		return f.funcValue(sym.Func, e.Name.Pos)
	}
	if sym.Kind == SymGlobal {
		f.errorf(e.Name.Pos, "'%s' is the state of executor '%s': only %s's own code names it; add a function to %s for what you need (D150)", e.Name.Name, exec, exec, exec)
	} else {
		f.errorf(e.Name.Pos, "'%s.%s' runs on executor '%s': outside its code it is called, not taken as a value — write the lambda, '(…) => %s.%s(…)' (D150)", exec, e.Name.Name, exec, exec, e.Name.Name)
	}
	return bad()
}

// childStaysOnThread: a child launched here runs on the one thread of the
// executor this code runs on — its scope names no other executor — so
// what it is given need not be Sendable (D150).
func (f *fnCtx) childStaysOnThread() bool {
	if f.exec == nil || !f.exec.single {
		return false
	}
	if n := len(f.scopes); n > 0 {
		if on := f.scopes[n-1].On; on != nil {
			ref, ok := on.(*VarRef)
			return ok && ref.Var.Global == f.exec.global
		}
	}
	return true
}
