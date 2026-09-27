package sema

import (
	"sort"
	"strconv"
	"strings"

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
	narrow map[place]types.Type
	// provenReads are the `at`/`first`/`last` calls a bounds fact made total
	// (D62), so a `?:` after one is a warning, not an error.
	provenReads map[*ast.CallExpr]bool
	// lenAliases: `val n = xs.len()` makes n stand for xs.len() while the
	// fact place{n, "$lenof:<xs>"} holds (D62).
	lenAliases map[*Var]*Var
	// bodyAST is the body being checked (a Block or an expression), for the
	// whole-function look D63's move check takes.
	bodyAST any
	// loopIters is the checked type of each `loop (x in iter)` head, for lints
	// that look at the loop after it was checked (lint_count.go).
	loopIters map[*ast.LoopStmt]types.Type

	retType     types.Type // declared return type (never the Result wrapper)
	throws      bool
	errType     types.Type // declared error union, or nil when inferred
	unsafe      int
	launching   *ast.CallExpr  // the call `async` is launching: it must stay a plain Call
	selfVar   *Var           // `this`: a pointer to the receiver's place (D22 v0.30)
	initOwned   map[string]int // checking an `init { }` block: the fields it must assign, by index (D28)
	selfAsRecv  bool           // the next `this` is a method receiver, not a value (init blocks)
	isGlobal    bool           // checking a global initializer
	staticOwner *types.Struct  // the struct whose `static val` this global initializer is, if any

	// lambda support
	parent      *fnCtx
	vars        map[*Var]bool // variables declared in this function
	captures    map[*Var]*Var // outer variable -> inner stand-in
	captureList []*Var        // outer variables in environment order
	paramInit   []Stmt        // lambda: statements destructuring tuple-pattern parameters
	isLambda    bool
	pending     []Stmt            // statements hoisted by adapter lowering
	boundPlace  map[ast.Expr]Expr // receiver of a `?.` assignment, already lowered to its place (check_safe.go)
	adapter     *adapterState     // the eager collection operation being lowered (lower_try.go)
	scopes      []*ScopeBlock
	awaitNext   bool
	inRaceArm   bool
	inferThrows bool
	inferredRet types.Type
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
	return &fnCtx{c: c, fn: fn, module: module, file: file, env: env, subst: subst, scope: scope, narrow: map[place]types.Type{}, vars: map[*Var]bool{}, captures: map[*Var]*Var{}}
}

func (f *fnCtx) errorf(span source.Span, format string, args ...any) {
	f.c.errorf(span, format, args...)
}

// resolve turns a syntactic type into a concrete type in this function.
func (f *fnCtx) resolve(t ast.Type) types.Type {
	rt := f.c.resolveType(f.env, t)
	return f.c.hooks.Subst(rt, f.subst)
}

func (f *fnCtx) newVar(name string, t types.Type, mutable bool, span source.Span) *Var {
	f.c.nextVar++
	v := &Var{Name: name, Type: t, Mutable: mutable, ID: f.c.nextVar, Span: span}
	if name != "this" && !strings.HasPrefix(name, "$") && t != nil {
		f.c.refVar(span, v)
	}
	f.vars[v] = true
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

// declareChecked declares a binding that must be read somewhere: a `val`
// or `var`, a destructured element, a loop or pattern variable. Parameters
// and `with` resources are exempt (a resource may be held only for its
// close), and `_` is the explicit discard.
func (f *fnCtx) declareChecked(name string, v *Var, span source.Span) {
	v.checkUse = name != "_"
	f.declareLocal(name, v, span)
}

// markUsed records a read of v. A read inside a lambda counts for the
// captured variable in every enclosing function.
func markUsed(v *Var) {
	for ; v != nil; v = v.Outer {
		v.used = true
	}
}

// reportUnused runs after a body is checked. A binding that is never read
// is a warning, except that an unread Result silently swallows a failure
// the same way a discarded call would, so it is the same error (D4).
// Assignments do not count as reads.
func (f *fnCtx) reportUnused() {
	for _, v := range f.fn.Locals {
		if !v.checkUse || v.used || v.Outer != nil || f.c.syntheticSpans[v.Span] {
			continue // a name in derived code (D58) is no one's to rename
		}
		if isResultType(v.Type) {
			f.errorf(v.Span, "unused Result '%s': the call may fail; use 'try' to propagate the error or 'when' to handle it (D4)", v.Name)
		} else if !types.IsInvalid(v.Type) {
			f.warnFix(v.Span, fixReplace("Rename to '_' (discard)", v.Span, "_"), "'%s' is never used", v.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// function bodies

func (c *Checker) checkBody(fn *Func) {
	source.SetWhere("checking", fn.Display, fn.Span)
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
		owner = c.hooks.Subst(t.Impl.Target, fn.subst)
	}
	if t.Trait != nil {
		for k, v := range c.traitDecl[t.Trait].tps {
			env.tps[k] = v
		}
		env.trait = t.Trait
		owner = fn.subst[selfParamOf(t.Trait)]
	}
	if t.Impl != nil {
		env.implAssoc = t.Impl.AssocTypes
	}
	env.self = owner
	f := c.newFnCtx(fn, t.Module, t.File, env, fn.subst)
	if t.SuiteScope != nil {
		f.scope = NewScope(t.SuiteScope) // the suite's helpers are in reach
	}
	if t.Decl != nil {
		if t.Decl.Body != nil {
			f.bodyAST = t.Decl.Body
		} else if t.Decl.ExprBody != nil {
			f.bodyAST = t.Decl.ExprBody
		}
	}
	c.refFunc(t.Decl.Name.Pos, t)
	f.retType = fn.Sig.Ret
	f.throws = fn.Sig.Effects.Throws
	if f.throws && t.Decl.Effects.Error != nil {
		f.errType = fn.Sig.Effects.Error
		c.checkErrorType(f.errType, t.Decl.Effects.Error.Span())
	}
	if t.Decl.Unsafe {
		f.unsafe = 1
	}
	if owner != nil && !t.Decl.Static {
		// D22 (v0.30): every method receives a pointer to the place it was
		// called on, so it may assign the receiver's `var` fields; `this`
		// still reads as the value (checkLValue/selfRef dereference it).
		fn.Receiver = f.newVar("this", &types.Pointer{Elem: owner}, true, t.Decl.Name.Pos)
		fn.Receiver.IsSelf = true
		f.selfVar = fn.Receiver
		if t.Name == "$init" {
			// the `init { }` block: the fields no caller supplies are its to
			// assign, on every path, before anything reads them (D28)
			f.initOwned = map[string]int{}
			if st, ok := owner.(*types.Struct); ok {
				for _, fld := range st.Fields {
					if fld.Init {
						f.initOwned[fld.Name] = fld.Index
					}
				}
			}
		}
	}
	for i, p := range t.Decl.Params {
		v := f.newVar(p.Name.Name, fn.Sig.Params[i].Type, false, p.Name.Pos)
		v.IsParam = true
		if ft, ok := t.Sig.Params[i].Type.(*types.Func); ok && ft.Effects.Throws && ft.Effects.Error != nil && types.ContainsTypeParam(ft.Effects.Error) {
			v.ErrPoly = true // `try f(x)` stays valid in the instance where E is Never
		}
		fn.Params = append(fn.Params, v)
		f.declareLocal(p.Name.Name, v, p.Name.Pos)
	}
	defer f.reportUnused()
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
		if f.initOwned != nil && !types.IsNever(body.Type) {
			// every field the block owns must be assigned on every path
			for _, name := range f.initMissing() {
				f.errorf(t.Decl.Name.Pos, "'init' does not assign '%s' on every path; it has no default, so 'init' must give it a value before the block ends (D28)", name)
			}
		}
		fn.Body = body
	}
}

// initFact is the flow fact "the init block has assigned field name":
// kept in the narrowing state, so it joins and dies with control flow
// like any smart cast (definite assignment for free).
func (f *fnCtx) initFact(name string) place {
	return place{v: f.selfVar, path: "$init." + name}
}

// initScope is the init-block context enclosing f (lambdas inside see it).
func (f *fnCtx) initScope() *fnCtx {
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.initOwned != nil {
			return ctx
		}
	}
	return nil
}

// initMissing lists the owned fields not yet definitely assigned here, in
// declaration order.
func (f *fnCtx) initMissing() []string {
	var out []string
	for name := range f.initOwned {
		if _, ok := f.narrow[f.initFact(name)]; !ok {
			out = append(out, name)
		}
	}
	sort.Slice(out, func(i, j int) bool { return f.initOwned[out[i]] < f.initOwned[out[j]] })
	return out
}

// selfParamOf returns the synthetic type parameter standing for Self in a
// trait default method template, kept on the trait itself.
func selfParamOf(t *types.Trait) *types.TypeParam {
	if t.SelfParam == nil {
		t.SelfParam = &types.TypeParam{Name: "Self", Owner: t.Name}
	}
	return t.SelfParam
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
	f.staticOwner = c.staticOwner[g] // a `static val` initializer is inside its type
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
	if d.Kind == ast.BindVar && !isSynchronized(declared) {
		// D66: every task sees the same globals, from whichever thread runs it
		example := "val " + d.Name.Name + " = Mutex(value: ...)"
		if _, ok := declared.(*types.Basic); ok {
			example = "val " + d.Name.Name + " = Atomic(value: ...)"
		}
		c.errorf(d.Name.Pos, "module-level 'var %s' is shared by every task, and tasks run on several threads at once (D66); keep the state behind a lock: '%s'", d.Name.Name, example)
	}
	g.Type = declared
	g.Init = init
}

// isSynchronized: a Mutex or an Atomic, whose operations take their lock
func isSynchronized(t types.Type) bool {
	st, ok := t.(*types.Struct)
	return ok && st.Module == "std.prelude" && (st.Name == "Mutex" || st.Name == "Atomic")
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
	f.lintStaleRefs(b.Stmts)
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
		if i > 0 {
			f.lintCountingLoop(b.Stmts, i-1) // needs the loop at i checked
		}
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
		if call, ok := x.(*Call); ok && call.Fn.tmpl != nil {
			if _, must := call.Fn.tmpl.Attrs["mustUse"]; must && !types.IsUnit(call.Type()) {
				f.errorf(s.X.Span(), "result of '%s' must be used (@mustUse)", call.Fn.Display)
			}
		}
		if isResultType(x.Type()) {
			f.errorf(s.X.Span(), "unused Result: the call may fail; use 'try' to propagate the error or 'when' to handle it (D4)")
		}
		return []Stmt{&ExprStmt{X: x}}, types.IsNever(x.Type())
	case *ast.AssignStmt:
		return f.checkAssign(s), false
	case *ast.ReturnStmt:
		return f.checkReturn(s), true
	case *ast.ThrowStmt:
		return f.checkThrow(s), true
	case *ast.BreakStmt:
		lp := f.findLoop(s.Label, s.Pos, "break")
		if lp == nil {
			return nil, true
		}
		lp.hasBreak = true
		return []Stmt{&Break{Loop: lp}}, true
	case *ast.ContinueStmt:
		lp := f.findLoop(s.Label, s.Pos, "continue")
		if lp == nil {
			return nil, true
		}
		lp.hasContinue = true
		return []Stmt{&Continue{Loop: lp}}, true
	case *ast.LoopStmt:
		stmts := f.checkLoop(s)
		// an infinite loop that never breaks does not fall through
		if s.Cond == nil && s.Var == nil && len(stmts) > 0 {
			if lp, ok := stmts[len(stmts)-1].(*Loop); ok && !lp.hasBreak {
				return stmts, true
			}
		}
		return stmts, false
	case *ast.Block:
		b := f.checkBlock(s, nil, false)
		return []Stmt{b}, types.IsNever(b.Type)
	case *ast.ScopeStmt:
		stmts := f.scopeStmt(s)
		// a body that always returns or throws leaves through the scope's
		// cleanup (children cancelled and joined); nothing follows
		if len(stmts) == 1 {
			if sb, ok := stmts[0].(*ScopeBlock); ok && sb.Body != nil && types.IsNever(sb.Body.Type) {
				return stmts, true
			}
		}
		return stmts, false
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
	if s.Else != nil {
		return f.letElse(s)
	}
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
		if s.Value != nil {
			v.InitText = srcText(s.Value) // for the hover
			f.c.refVar(b.Name.Pos, v)     // re-record the declaration with it
		}
		f.declareChecked(b.Name.Name, v, b.Name.Pos)
		if s.Value != nil {
			f.declFacts(v, s.Value)
		}
		return []Stmt{&VarDecl{Var: v, Init: init}}
	}
	// tuple destructuring (D37)
	init := f.checkExpr(s.Value, nil)
	tt, ok := init.Type().(*types.Tuple)
	if !ok {
		if !types.IsInvalid(init.Type()) {
			f.errorf(s.Value.Span(), "cannot destructure a value of type '%s'; only tuples destructure positionally (D37)", init.Type())
		}
		f.bindPattern(&b, types.TInvalid, mutable) // the names exist: one error, not one per use
		return nil
	}
	if len(tt.Elems) != len(b.Tuple) {
		f.errorf(b.Pos, "tuple has %d elements but %d names are bound", len(tt.Elems), len(b.Tuple))
		f.bindPattern(&b, types.TInvalid, mutable)
		return nil
	}
	tmp, parts := f.bindPattern(&b, tt, mutable)
	return append([]Stmt{&VarDecl{Var: tmp, Init: init}}, parts...)
}

func (f *fnCtx) checkReturn(s *ast.ReturnStmt) []Stmt {
	if f.isGlobal {
		f.errorf(s.Pos, "'return' outside of a function")
		return nil
	}
	if f.retType == nil {
		// lambda with an inferred return type: the first return decides
		if s.Value == nil {
			f.retType = types.TUnit
		} else {
			x := f.checkExpr(s.Value, nil)
			f.retType = x.Type()
			f.inferredRet = x.Type()
			if types.IsNever(x.Type()) {
				return []Stmt{&ExprStmt{X: x}}
			}
			return []Stmt{&Return{Value: x}}
		}
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
// checkAssign checks an assignment; counting a non-negative index up keeps
// it non-negative (D62).
func (f *fnCtx) checkAssign(s *ast.AssignStmt) []Stmt {
	keep := f.countsUp(s)
	out := f.checkAssignInner(s)
	if keep != nil {
		f.narrow[nonNegKey(keep)] = types.TUnit
	}
	return out
}

func (f *fnCtx) checkAssignInner(s *ast.AssignStmt) []Stmt {
	if safe := safeMemberOf(s.Target); safe != nil {
		return f.safeAssign(s, safe)
	}
	f.lintSelfAddress(s)
	if ix, ok := s.Target.(*ast.IndexExpr); ok {
		// removed form (lint_index.go): reported once here with its fix, then
		// typed as before so nothing else cascades
		m := f.checkExpr(ix.X, nil)
		if mt, isMap := m.Type().(*types.Map); isMap {
			f.indexWrite(s, ix, true)
			if s.Op != lexer.Assign {
				f.checkExprTo(s.Value, mt.Value)
				return nil
			}
			return f.mapIndexAssign(ix, m, mt, s.Value, s.Pos)
		}
		if lt, isList := m.Type().(*types.List); isList {
			f.indexWrite(s, ix, false)
			target := f.listElemPlace(m, lt, ix.Index, ix.Pos, true)
			var value Expr
			if s.Op == lexer.Assign {
				value = f.coerce(f.checkExpr(s.Value, lt.Elem), lt.Elem, s.Value.Span())
			} else {
				value = f.makeBinary(BinOpFromToken(s.Op), target, f.checkExprTo(s.Value, lt.Elem), s.Pos)
			}
			return []Stmt{&Assign{Target: target, Value: value}}
		}
	}
	if tup, ok := s.Target.(*ast.TupleExpr); ok {
		return f.tupleAssign(s, tup)
	}
	if n, isName := s.Target.(*ast.NameExpr); isName {
		// `n += 1` on a `val n: *i64` (a `loop (&n in nums)` variable or a
		// `&` binding): the pointer cannot be rebound; the pointee can
		if sym := f.scope.Lookup(n.Name); sym != nil && sym.Kind == SymLocal {
			v := f.localVar(sym.Var)
			if pt, isPtr := v.Type.(*types.Pointer); isPtr && !v.Mutable && !pt.Raw {
				repl := "*" + n.Name
				f.c.errorFix(n.Pos, fixReplace("Replace with '"+repl+"'", n.Pos, repl),
					"'%s' is a pointer to a '%s' and cannot be reassigned; write '%s' to change the value it points to", n.Name, pt.Elem, repl)
				f.checkExpr(s.Value, nil)
				return nil
			}
		}
	}
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
		var pre []Stmt
		target, pre = f.hoistPlace(target)
		if operandTakesOperator(op, target.Type()) {
			// t += d is t = t.plus(d) (D71); the result must fit the place
			value = f.coerce(f.operatorCall(op, target, s.Value, s.Pos), target.Type(), s.Pos)
			return append(pre, f.assignPlace(s.Target, target, root, value, rawType))
		}
		rhs := f.checkExprTo(s.Value, target.Type())
		value = f.makeBinary(op, target, rhs, s.Pos)
		return append(pre, f.assignPlace(s.Target, target, root, value, rawType))
	}
	return []Stmt{f.assignPlace(s.Target, target, root, value, rawType)}
}

// tupleAssign is `(a, b) = expr` (D37 destructuring as an assignment): the
// right side is evaluated once into a temporary and then each place is
// assigned in order, so `(a, b) = (b, a)` swaps without a named temp.
func (f *fnCtx) tupleAssign(s *ast.AssignStmt, tup *ast.TupleExpr) []Stmt {
	if s.Op != lexer.Assign {
		f.errorf(s.Pos, "compound assignment cannot target a tuple; assign each place separately")
		f.checkExpr(s.Value, nil)
		return nil
	}
	var targets []Expr
	var roots []*Var
	var elemTypes []types.Type
	ok := true
	for _, el := range tup.Elems {
		t, root := f.checkLValue(el, true)
		if t == nil {
			ok = false
			continue
		}
		targets = append(targets, t)
		roots = append(roots, root)
		elemTypes = append(elemTypes, t.Type())
	}
	if !ok {
		f.checkExpr(s.Value, nil)
		return nil
	}
	want := &types.Tuple{Elems: elemTypes}
	var raw Expr
	rawTypes := elemTypes // what each place is narrowed to after the store
	if lit, ok := s.Value.(*ast.TupleExpr); ok && len(lit.Elems) == len(targets) {
		// a literal on the right (`(a, b) = (b, a)`): check each element
		// against its own place, exactly as a plain assignment would
		tl := &TupleLit{}
		rawTypes = nil
		for i, el := range lit.Elems {
			x := f.checkExpr(el, elemTypes[i])
			rawTypes = append(rawTypes, x.Type())
			tl.Elems = append(tl.Elems, f.coerce(x, elemTypes[i], el.Span()))
		}
		tl.T = want
		raw = tl
	} else {
		raw = f.checkExpr(s.Value, want)
	}
	tt, isTuple := raw.Type().(*types.Tuple)
	if !isTuple {
		if !types.IsInvalid(raw.Type()) {
			f.errorf(s.Value.Span(), "cannot destructure a value of type '%s' into %d places; only tuples destructure positionally (D37)", raw.Type(), len(targets))
		}
		return nil
	}
	if len(tt.Elems) != len(targets) {
		f.errorf(s.Value.Span(), "tuple has %d elements but %d places are assigned", len(tt.Elems), len(targets))
		return nil
	}
	tmp := f.newTemp(tt)
	stmts := []Stmt{&VarDecl{Var: tmp, Init: raw}}
	for i, target := range targets {
		part := Expr(&TupleGet{exprBase{tt.Elems[i]}, &VarRef{exprBase{tt}, tmp}, i})
		value := f.coerce(part, target.Type(), tup.Elems[i].Span())
		stmts = append(stmts, f.assignPlace(tup.Elems[i], target, roots[i], value, rawTypes[i]))
	}
	return stmts
}

// assignPlace builds the store of value into target and updates the smart
// casts for the assigned place.
func (f *fnCtx) assignPlace(targetAst ast.Expr, target Expr, root *Var, value Expr, rawType types.Type) Stmt {
	// Assignment re-narrows the variable to the assigned value's type (a
	// non-null value smart-casts a nullable var), or clears the narrowing.
	if root != nil {
		if _, ok := target.(*VarRef); ok {
			f.invalidatePlace(pv(root))
			if rawType != nil && !types.Identical(rawType, root.Type) && !types.IsNever(rawType) && !types.IsInvalid(rawType) && f.assignableTo(rawType, root.Type) {
				f.narrow[pv(root)] = rawType
			}
		} else if p, ok := f.placeOf(targetAst); ok {
			// a field write drops the facts about that field and below
			f.invalidatePlace(p)
		} else {
			f.invalidatePaths(root)
		}
	}
	if m, ok := targetAst.(*ast.MemberExpr); ok && f.initOwned != nil {
		if _, isSelf := m.X.(*ast.SelfExpr); isSelf {
			if _, owned := f.initOwned[m.Name.Name]; owned {
				f.narrow[f.initFact(m.Name.Name)] = types.TUnit // assigned from here on
			}
		}
	}
	return &Assign{Target: target, Value: value}
}

// checkLValue checks an assignable expression. It returns the lvalue and,
// when the place is rooted in a local or global variable (not through a
// pointer), that variable. With mutate=true it enforces D11/D22 mutability.
func (f *fnCtx) checkLValue(e ast.Expr, mutate bool) (Expr, *Var) {
	if p, ok := f.boundPlace[e]; ok {
		return p, nil // the receiver of a `?.` write (check_safe.go)
	}
	switch e := e.(type) {
	case *ast.NameExpr:
		sym := f.scope.Lookup(e.Name)
		if sym == nil {
			f.errorf(e.Pos, "unknown name '%s'", e.Name)
			return nil, nil
		}
		switch sym.Kind {
		case SymLocal:
			v := f.localVar(sym.Var)
			if mutate && !v.Mutable {
				f.errorf(e.Pos, "cannot assign to '%s': it is a 'val'; declare it with 'var' (D11)", e.Name)
			}
			if !mutate {
				// a field write or address-of reads the binding; a plain
				// assignment to the name itself does not
				markUsed(v)
			}
			f.c.refVar(e.Pos, v) // the name at an assignment hovers like a read
			return &VarRef{exprBase{v.Type}, v}, v
		case SymGlobal:
			g := sym.Global
			if mutate && !g.Mutable {
				f.errorf(e.Pos, "cannot assign to '%s': it is not a 'var'", e.Name)
			}
			v := f.globalVar(g)
			f.c.refGlobal(e.Pos, e.Name, g)
			return &VarRef{exprBase{v.Type}, v}, v
		}
		f.errorf(e.Pos, "'%s' is not assignable", e.Name)
		return nil, nil
	case *ast.SelfExpr:
		self := f.selfRef()
		if self == nil {
			f.errorf(e.Pos, "'this' outside of a method")
			return nil, nil
		}
		if mutate {
			f.errorf(e.Pos, "cannot assign to 'this': assign its 'var' fields, or return the new value (D22)")
		}
		return &Deref{exprBase{self.Type.(*types.Pointer).Elem}, &VarRef{exprBase{self.Type}, self}}, self
	case *ast.MemberExpr:
		if e.Safe {
			f.errorf(e.Pos, "cannot assign through '?.'")
			return nil, nil
		}
		// Through a pointer, mutability is not gated by the binding (D11).
		var base Expr
		var root *Var
		temporary := false
		if !isPlaceSyntax(e.X) {
			// an rvalue such as a call result: a field behind a pointer it
			// returns is writable (`ptrOf(h).n = 5`); a value is a temporary
			// — `xs.at(i)` included, which reads a copy (D25, v0.27)
			base = f.checkExpr(e.X, nil)
			if types.IsInvalid(base.Type()) {
				return nil, nil
			}
			temporary = true
		} else if base, root = f.checkLValue(e.X, false); base == nil {
			return nil, nil
		} else {
			base = f.narrowLValue(base, e.X)
		}
		bt := base.Type()
		if p, ok := bt.(*types.Pointer); ok {
			if p.Raw && f.unsafe == 0 {
				f.errorf(e.Pos, "dereferencing a raw pointer requires an 'unsafe' block (D44)")
			}
			base = &Deref{exprBase{p.Elem}, base}
			bt = p.Elem
			root = nil
		} else if mutate && root != nil && root.IsGlobal && !root.Mutable {
			// a global `val` is shared by every task (D35): nothing in it changes
			f.errorf(e.Pos, "cannot assign to field '%s' of '%s': a global 'val' is a constant; declare it 'var' (D11/D35)", e.Name.Name, root.Name)
		} else if mutate && (temporary || root == nil && !isPlaceExpr(base)) {
			f.copyMutationHint(e.X, e.Pos, "cannot assign to a field of a temporary value")
			return nil, nil
		}
		if tt, isTuple := bt.(*types.Tuple); isTuple {
			// a tuple element is a place too (`pair.1.bump()`); elements
			// are not assignable one at a time
			idx, err := strconv.Atoi(e.Name.Name)
			if err != nil || idx < 0 || idx >= len(tt.Elems) {
				f.errorf(e.Name.Pos, "tuple of %d elements has no element '%s'", len(tt.Elems), e.Name.Name)
				return nil, nil
			}
			if mutate {
				f.errorf(e.Pos, "cannot assign to a tuple element; assign the whole tuple (D48)")
			}
			return &TupleGet{exprBase{tt.Elems[idx]}, base, idx}, root
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
		if mutate {
			// D22 (v0.30): mutability is declared on the field, whoever holds
			// the struct: a bare field is set once, by the constructor call
			// — or by the `init { }` block through `this`, the one other place
			// (D28); a `protected var` is assigned only by the type's own
			// declarations
			if !fld.Var && !(root != nil && root.IsSelf && f.inInit(st)) {
				f.c.errorFix(e.Name.Pos, f.fixVarField(st, fld), "cannot assign to '%s.%s': the field is immutable; declare it 'var %s: %s' to allow assignment, or build a new '%s' (D22)", st.Name, fld.Name, fld.Name, fld.Type, st.Name)
			} else if fld.Protected && !f.insideType(st) {
				f.errorf(e.Name.Pos, "cannot assign to '%s.%s' here: the field is 'protected var', assigned only by '%s' itself — its methods, impl and extend blocks; call a method of '%s' (D22)", st.Name, fld.Name, st.Name, st.Name)
			}
		}
		return &FieldGet{exprBase{fld.Type}, base, fld.Index, fld.Name}, root
	case *ast.IndexExpr:
		// removed form (lint_index.go); `xs.set(i, v)` and `xs.ref(i)` reach the place now
		x := f.checkExpr(e.X, nil)
		lt, ok := x.Type().(*types.List)
		if !ok {
			f.errorf(e.Pos, "'[...]' after a value of type '%s' is not indexing; brackets are for collection literals only (D25)", x.Type())
			return nil, nil
		}
		f.indexRead(e, false)
		return f.listElemPlace(x, lt, e.Index, e.Pos, mutate), nil
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
	if repl, ok := elemReadCall(e); ok && !strings.Contains(repl, "?.") {
		// `xs.at(i) += 1`: the read is a copy; the element's storage is
		// behind `xs.ref(i)`, a nullable pointer (D25 v0.27, D62)
		f.checkExpr(e, nil) // so the names in it count as used
		repl = "*(" + repl + " ?: panic(\"" + panicReasonTODO + "\"))"
		f.c.errorFix(e.Span(), fixReplace("Replace with '"+repl+"'", e.Span(), repl),
			"'%s' is a copy of the element, not the element; assign through '%s', or use 'set' (D25)", srcText(e), repl)
		return nil, nil
	}
	f.errorf(e.Span(), "expression is not assignable")
	return nil, nil
}

// inInit reports whether the code being checked is st's `init { }` block
// (its hidden `$init` method), where bare fields may be assigned once.
func (f *fnCtx) inInit(st *types.Struct) bool {
	tmpl := templateOf(st)
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.fn != nil && ctx.fn.tmpl != nil && ctx.fn.tmpl.Name == "$init" && ctx.fn.tmpl.Owner != nil && templateOf(ctx.fn.tmpl.Owner) == tmpl {
			return true
		}
	}
	return false
}

// insideType reports whether this code belongs to the type: one of its
// methods, an `impl`/`extend` block for it in its own module, or the
// initializer of one of its `static val`s — the places that may use its
// `private` members. Lambdas inside such code count.
func (f *fnCtx) insideType(st *types.Struct) bool {
	tmpl := templateOf(st)
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.staticOwner != nil && templateOf(ctx.staticOwner) == tmpl {
			return true
		}

		if ctx.fn == nil || ctx.fn.tmpl == nil {
			continue
		}
		t := ctx.fn.tmpl
		if t.Owner != nil && templateOf(t.Owner) == tmpl {
			return true
		}
		if t.Impl != nil && t.Impl.Module == f.module {
			if hs, ok := t.Impl.Target.(*types.Struct); ok && templateOf(hs) == tmpl {
				return true
			}
		}
	}
	return false
}

func (f *fnCtx) lookupField(st *types.Struct, name string, span source.Span) *types.Field {
	f.c.resolveStruct(templateOf(st))
	for _, fld := range st.Fields {
		if fld.Name == name {
			if fld.Private && !f.insideType(st) {
				f.errorf(span, "field '%s' is private to '%s': only its own methods, impl and extend blocks may use it", name, st.Name)
			} else if !fld.Pub && st.Module != f.module.prefix() {
				f.errorf(span, "field '%s' of '%s' is private to module '%s' (M5)", name, st.Name, st.Module)
			}
			f.c.refField(span, st, fld)
			return fld
		}
	}
	f.errorf(span, "'%s' has no field '%s'", st, name)
	return nil
}

// globalVar returns the Var standing for a module-level binding.
func (f *fnCtx) globalVar(g *Global) *Var {
	if v, ok := f.c.globalVars[g]; ok {
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
	f.c.globalVars[g] = v
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
	// The body runs again after its last statement, so a call anywhere in it
	// ends what was known about MutableList lengths everywhere in it (D62).
	if astMayShrink(s.Body) || s.Cond != nil && astMayShrink(s.Cond) {
		f.killMutableBounds()
	}
	if s.Cond != nil {
		f.lintLoopTrue(s)
	}

	var pre []Stmt
	f.pushScope()
	defer f.popScope()
	switch {
	case s.Var != nil:
		iter := f.checkExpr(s.Iter, nil)
		if f.loopIters == nil {
			f.loopIters = map[*ast.LoopStmt]types.Type{}
		}
		f.loopIters[s] = iter.Type()
		switch it := iter.Type().(type) {
		case *types.Range:
			// var i = lo; val hi = hi; loop (i < hi) { val x = i; body; post: i += 1 }
			if hasRefBinding(s.Var) {
				f.errorf(s.Var.Pos, "'&' binds an element of a MutableList or a value of a MutableMap in place; a range yields values (D42)")
			}
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
			v, parts := f.bindLoopVar(s.Var, it.Elem)
			if s.Var.Name != nil {
				f.rangeLoopFacts(s, v)
			}
			f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
			body := f.checkBlock(s.Body, nil, false)
			f.loops = f.loops[:len(f.loops)-1]
			body.Stmts = append(append([]Stmt{&VarDecl{Var: v, Init: &VarRef{exprBase{it.Elem}, idx}}}, parts...), body.Stmts...)
			lp.Body = body
		case *types.List:
			byRef := s.Var.Name != nil && s.Var.Ref
			if !byRef && hasRefBinding(s.Var) {
				f.errorf(s.Var.Pos, "'&' binds a list element as a whole: 'loop (&x in xs)' (D42)")
			}
			if byRef && !it.Mutable {
				f.errorf(s.Var.Pos, "'loop (&x in xs)' needs a MutableList: it changes the elements in place, and a List is read-only (D25, D42)")
			}
			listTmp := f.newTemp(it)
			pre = append(pre, &VarDecl{Var: listTmp, Init: iter})
			idx := f.newTemp(types.TI64)
			pre = append(pre, &VarDecl{Var: idx, Init: &IntConst{exprBase{types.TI64}, 0, false}})
			lenExpr := &Builtin{exprBase{types.TI64}, "list.len", []Expr{&VarRef{exprBase{it}, listTmp}}, s.Pos}
			lp.Cond = &Binary{exprBase{types.TBool}, OpLt, &VarRef{exprBase{types.TI64}, idx}, lenExpr, s.Pos}
			lp.Post = []Stmt{&Assign{Target: &VarRef{exprBase{types.TI64}, idx}, Value: &Binary{exprBase{types.TI64}, OpWrapAdd, &VarRef{exprBase{types.TI64}, idx}, &IntConst{exprBase{types.TI64}, 1, false}, s.Pos}}}
			var elemT types.Type = it.Elem
			if byRef {
				elemT = &types.Pointer{Elem: it.Elem}
			}
			v, parts := f.bindLoopVar(s.Var, elemT)
			f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
			body := f.checkBlock(s.Body, nil, false)
			f.loops = f.loops[:len(f.loops)-1]
			var get Expr = &Builtin{exprBase{it.Elem}, "list.get", []Expr{&VarRef{exprBase{it}, listTmp}, &VarRef{exprBase{types.TI64}, idx}}, s.Pos}
			if byRef {
				// `loop (&x in xs)`: x is a pointer to the element's storage (D42)
				get = &AddrOf{exprBase{elemT}, &Builtin{exprBase{it.Elem}, "list.ref", []Expr{&VarRef{exprBase{it}, listTmp}, &VarRef{exprBase{types.TI64}, idx}}, s.Pos}}
			}
			body.Stmts = append(append([]Stmt{&VarDecl{Var: v, Init: get}}, parts...), body.Stmts...)
			lp.Body = body
		case *types.Map, *types.Set:
			if mt, isMap := it.(*types.Map); isMap && s.Var.Name == nil && len(s.Var.Tuple) == 2 && s.Var.Tuple[1].Ref && !s.Var.Tuple[0].Ref {
				return f.mapRefLoop(s, iter, mt, lp, label)
			}
			if hasRefBinding(s.Var) {
				if _, isMap := it.(*types.Map); isMap {
					f.errorf(s.Var.Pos, "a map key cannot be changed in place; '&' goes on the value: 'loop ((k, &v) in m)' (D42)")
				} else {
					f.errorf(s.Var.Pos, "'&' cannot bind a set element in place: sets hold their elements by value (D42)")
				}
			}
			op, elem := "map.entries", types.Type(nil)
			if mt, isMap := it.(*types.Map); isMap {
				elem = &types.Tuple{Elems: []types.Type{mt.Key, mt.Value}}
			} else {
				op, elem = "map.keys", it.(*types.Set).Elem
			}
			listT := &types.List{Elem: elem}
			snapshot := &Builtin{exprBase{listT}, op, []Expr{iter}, s.Pos}
			copy := *s
			if hasRefBinding(s.Var) {
				copy.Var = withoutRefs(s.Var) // reported above; the snapshot loop is by value
			}
			copy.Iter = &ast.NameExpr{Name: "", Pos: s.Iter.Span()}
			sv, decl := f.hidden("snapshot", snapshot, false)
			copy.Iter = nameOf(sv, s.Iter.Span())
			inner := f.checkLoop(&copy)
			return append([]Stmt{decl}, inner...)
		default:
			if hasRefBinding(s.Var) && !types.IsInvalid(iter.Type()) {
				f.errorf(s.Var.Pos, "'&' binds an element of a MutableList or a value of a MutableMap in place; an iterator yields values (D42)")
			}
			if stmts, handled := f.iteratorLoop(s, iter, lp, label); handled {
				return stmts
			}
			if !types.IsInvalid(iter.Type()) {
				f.errorf(s.Iter.Span(), "cannot iterate over '%s': it implements neither Iterable nor Iterator (D42)", iter.Type())
			}
			// the body is still checked, with the names bound (to nothing
			// known), so a bad iterable is one error and not one per use
			f.bindLoopVar(s.Var, types.TInvalid)
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
	if lp.Body != nil && types.IsNever(lp.Body.Type) && !lp.hasContinue {
		// every path through the body breaks, returns or throws: the loop
		// runs its body once and the `loop` is doing nothing
		kw := source.Span{File: s.Pos.File, Start: s.Pos.Start, End: s.Pos.Start + len("loop")}
		f.warnf(kw, "this loop never repeats: every path through its body leaves it (break, return or throw); drop the 'loop' or add a condition")
	}
	return append(pre, lp)
}

// withoutRefs is a copy of a binding with every `&` dropped.
func withoutRefs(b *ast.Binding) *ast.Binding {
	c := *b
	c.Ref = false
	c.Tuple = nil
	for i := range b.Tuple {
		c.Tuple = append(c.Tuple, *withoutRefs(&b.Tuple[i]))
	}
	return &c
}

// hasRefBinding reports whether a loop binding, or any part of a tuple
// binding, is written `&name`.
func hasRefBinding(b *ast.Binding) bool {
	if b.Ref {
		return true
	}
	for i := range b.Tuple {
		if hasRefBinding(&b.Tuple[i]) {
			return true
		}
	}
	return false
}

// mapRefLoop lowers `loop ((k, &v) in m)`: the keys are snapshotted, and
// for each one still present `v` is a pointer to the stored value, so the
// body changes the entry where it lives (D42). An entry removed by the
// body is skipped; one added is not visited.
func (f *fnCtx) mapRefLoop(s *ast.LoopStmt, m Expr, mt *types.Map, lp *Loop, label string) []Stmt {
	if !mt.Mutable {
		f.errorf(s.Var.Pos, "'loop ((k, &v) in m)' needs a MutableMap: it changes the values in place, and a Map is read-only (D25, D42)")
	}
	mapTmp := f.newTemp(mt)
	keysT := &types.List{Elem: mt.Key}
	keys := f.newTemp(keysT)
	idx := f.newTemp(types.TI64)
	pre := []Stmt{
		&VarDecl{Var: mapTmp, Init: m},
		&VarDecl{Var: keys, Init: &Builtin{exprBase{keysT}, "map.keys", []Expr{ref(mapTmp)}, s.Pos}},
		&VarDecl{Var: idx, Init: i64c(0)},
	}
	lp.Cond = &Binary{exprBase{types.TBool}, OpLt, ref(idx), &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(keys)}, s.Pos}, s.Pos}
	lp.Post = []Stmt{&Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), i64c(1), s.Pos}}}
	ptrT := &types.Pointer{Elem: mt.Value}
	k, kParts := f.bindLoopVar(&s.Var.Tuple[0], mt.Key)
	v, _ := f.bindLoopVar(&s.Var.Tuple[1], ptrT)
	f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
	body := f.checkBlock(s.Body, nil, false)
	f.loops = f.loops[:len(f.loops)-1]
	slot := f.newTemp(&types.Nullable{Elem: ptrT})
	lp.hasContinue = true
	prefix := []Stmt{&VarDecl{Var: k, Init: &Builtin{exprBase{mt.Key}, "list.get", []Expr{ref(keys), ref(idx)}, s.Pos}}}
	prefix = append(prefix, kParts...)
	prefix = append(prefix,
		&VarDecl{Var: slot, Init: &Builtin{exprBase{slot.Type}, "map.ref", []Expr{ref(mapTmp), ref(k)}, s.Pos}},
		&ExprStmt{X: &If{exprBase{types.TUnit}, &IsNull{exprBase{types.TBool}, ref(slot)},
			&Block{Stmts: []Stmt{&Continue{Loop: lp}}, Type: types.TNever}, nil}},
		&VarDecl{Var: v, Init: &Unwrap{exprBase{ptrT}, ref(slot)}})
	body.Stmts = append(prefix, body.Stmts...)
	lp.Body = body
	return append(pre, lp)
}

func (f *fnCtx) bindLoopVar(b *ast.Binding, t types.Type) (*Var, []Stmt) {
	return f.bindPattern(b, t, false)
}

// bindPattern declares the names of a binding pattern for a value of type
// t: a plain name is one variable; a tuple pattern binds a hidden variable
// for the whole and, nested as deep as the pattern goes, one for each part
// (D37). The returned statements initialise the parts from the hidden
// variable, which the caller initialises.
func (f *fnCtx) bindPattern(b *ast.Binding, t types.Type, mutable bool) (*Var, []Stmt) {
	if b.Name == nil {
		tt, ok := t.(*types.Tuple)
		if !ok || len(tt.Elems) != len(b.Tuple) {
			if !types.IsInvalid(t) {
				f.errorf(b.Pos, "cannot destructure a '%s' into %d names", t, len(b.Tuple))
			}
			// the names still exist, typed as unknown, so each use is not
			// one more error
			for i := range b.Tuple {
				f.bindPattern(&b.Tuple[i], types.TInvalid, mutable)
			}
			return f.newTemp(t), nil
		}
		tmp := f.newTemp(t)
		var extra []Stmt
		for i, el := range b.Tuple {
			part, more := f.bindPattern(&el, tt.Elems[i], mutable)
			extra = append(extra, &VarDecl{Var: part, Init: &TupleGet{exprBase{tt.Elems[i]}, ref(tmp), i}})
			extra = append(extra, more...)
		}
		return tmp, extra
	}
	if b.Type != nil {
		want := f.resolve(b.Type)
		if !types.Identical(want, t) {
			f.errorf(b.Pos, "'%s' is bound to a value of type '%s', not '%s'", b.Name.Name, t, want)
		}
	}
	v := f.newVar(b.Name.Name, t, mutable, b.Name.Pos)
	f.declareChecked(b.Name.Name, v, b.Name.Pos)
	return v, nil
}

// invalidateAssigned drops narrowing for variables assigned inside a block.
func (f *fnCtx) invalidateAssigned(b *ast.Block) {
	var walk func(s ast.Stmt)
	walk = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.AssignStmt:
			// any write rooted at a local drops every fact about it
			target := s.Target
			for {
				m, ok := target.(*ast.MemberExpr)
				if !ok {
					break
				}
				target = m.X
			}
			if n, ok := target.(*ast.NameExpr); ok {
				if sym := f.scope.Lookup(n.Name); sym != nil && sym.Kind == SymLocal {
					keep := f.countsUp(s)
					f.invalidatePlace(pv(sym.Var))
					if keep != nil {
						f.narrow[nonNegKey(keep)] = types.TUnit
					}
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

// hidden binds an expression to a compiler-generated local that source
// code cannot name, so that synthesized syntax can refer to it.
func (f *fnCtx) hidden(prefix string, init Expr, mutable bool) (*Var, Stmt) {
	f.c.nextTmp++
	name := "$" + prefix + itoa(f.c.nextTmp)
	v := f.newVar(name, init.Type(), mutable, source.Span{})
	f.scope.Insert(&Symbol{Name: name, Kind: SymLocal, Var: v})
	return v, &VarDecl{Var: v, Init: init}
}

func nameOf(v *Var, span source.Span) *ast.NameExpr {
	return &ast.NameExpr{Name: v.Name, Pos: span}
}

// traitNamed finds a prelude trait by name.
func (c *Checker) traitNamed(name string) *types.Trait {
	if sym := c.preludeSym(name); sym != nil && sym.Kind == SymType {
		if t, ok := sym.Type.(*types.Trait); ok {
			return t
		}
	}
	return nil
}

// iteratorLoop lowers `loop (x in c)` through Iterable / Iterator (D42):
//
//	var it = c.iterator()      // or c itself when it is already an Iterator
//	loop { val v = it.next(); if (v == null) break; val x = v; body }
func (f *fnCtx) iteratorLoop(s *ast.LoopStmt, iter Expr, lp *Loop, label string) ([]Stmt, bool) {
	iterable, iterator := f.c.traitNamed("Iterable"), f.c.traitNamed("Iterator")
	if iterable == nil || iterator == nil {
		return nil, false
	}
	t := iter.Type()
	var pre []Stmt
	var itVar *Var
	switch {
	case f.findImpl(t, iterable) != nil:
		src, decl := f.hidden("src", iter, false)
		pre = append(pre, decl)
		call := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(src, s.Iter.Span()), Name: ast.Ident{Name: "iterator", Pos: s.Iter.Span()}, Pos: s.Iter.Span()}, Pos: s.Iter.Span()}
		itExpr := f.checkExpr(call, nil)
		if types.IsInvalid(itExpr.Type()) {
			return nil, true
		}
		var d Stmt
		itVar, d = f.hidden("it", itExpr, true)
		pre = append(pre, d)
	case f.findImpl(t, iterator) != nil:
		var d Stmt
		itVar, d = f.hidden("it", iter, true)
		pre = append(pre, d)
	default:
		return nil, false
	}
	next := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(itVar, s.Iter.Span()), Name: ast.Ident{Name: "next", Pos: s.Iter.Span()}, Pos: s.Iter.Span()}, Pos: s.Iter.Span()}
	nx := f.checkExpr(next, nil)
	nt, ok := nx.Type().(*types.Nullable)
	if !ok {
		if !types.IsInvalid(nx.Type()) {
			f.errorf(s.Iter.Span(), "'next()' must return 'Item?', found '%s'", nx.Type())
		}
		return nil, true
	}
	v := f.newTemp(nt)
	x, parts := f.bindLoopVar(s.Var, nt.Elem)
	f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
	body := f.checkBlock(s.Body, nil, false)
	f.loops = f.loops[:len(f.loops)-1]
	stop := &Block{Stmts: []Stmt{&Break{Loop: lp}}, Type: types.TNever}
	head := []Stmt{
		&VarDecl{Var: v, Init: nx},
		&ExprStmt{X: &If{exprBase{types.TUnit}, &IsNull{exprBase{types.TBool}, ref(v)}, stop, nil}},
		&VarDecl{Var: x, Init: &Unwrap{exprBase{nt.Elem}, ref(v)}},
	}
	head = append(head, parts...)
	body.Stmts = append(head, body.Stmts...)
	lp.Body = body
	return append(pre, lp), true
}

// checkWith lowers `with (a = x, b = y) { body }` into nested With
// statements so that resources close in reverse order.
// withExpr checks `with (r = init) { body }` (D43). It is an expression:
// in statement position (want is unit) the body is a plain block; where a
// value is wanted the body's value is the result, carried out through a
// temporary declared before the resources are opened, so the close calls
// run between the body and the use of the value.
func (f *fnCtx) withExpr(e *ast.WithExpr, want types.Type) Expr {
	closeable := f.c.traitNamed("Closeable")
	f.pushScope()
	defer f.popScope()
	asValue := want == nil || !types.IsUnit(want)
	var result *Var
	stmts, bodyT := f.withBindings(e, 0, closeable, want, asValue, &result)
	out := &Block{Type: types.TUnit}
	if result != nil {
		out.Stmts = append(out.Stmts, &VarDecl{Var: result})
	}
	out.Stmts = append(out.Stmts, stmts...)
	switch {
	case types.IsNever(bodyT):
		out.Type = types.TNever
	case result != nil:
		out.Value = &VarRef{exprBase{result.Type}, result}
		out.Type = result.Type
	}
	return &BlockExpr{exprBase{out.Type}, out}
}

func (f *fnCtx) withBindings(s *ast.WithExpr, i int, closeable *types.Trait, want types.Type, asValue bool, result **Var) ([]Stmt, types.Type) {
	if i == len(s.Bindings) {
		if !asValue {
			b := f.checkBlock(s.Body, nil, false)
			return []Stmt{b}, b.Type
		}
		b := f.checkBlock(s.Body, want, true)
		if b.Value != nil && !types.IsUnit(b.Type) && !types.IsNever(b.Type) {
			*result = f.newTemp(b.Type)
			b.Stmts = append(b.Stmts, &Assign{Target: &VarRef{exprBase{b.Type}, *result}, Value: b.Value})
			b.Value = nil
			bt := b.Type
			b.Type = types.TUnit
			return []Stmt{b}, bt
		}
		if b.Value != nil {
			b.Stmts = append(b.Stmts, &ExprStmt{X: b.Value})
			b.Value = nil
		}
		return []Stmt{b}, b.Type
	}
	b := s.Bindings[i]
	init := f.checkExpr(b.Value, nil)
	if types.IsInvalid(init.Type()) {
		return nil, types.TInvalid
	}
	v := f.newVar(b.Name.Name, init.Type(), false, b.Name.Pos)
	f.declareLocal(b.Name.Name, v, b.Name.Pos)
	if closeable == nil || f.findImpl(init.Type(), closeable) == nil {
		f.errorf(b.Value.Span(), "'%s' is not Closeable; 'with' resources must implement Closeable (D43)", init.Type())
		return nil, types.TInvalid
	}
	// synthesize `name.close()` (a method call works on any binding, D22)
	call := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(v, b.Name.Pos), Name: ast.Ident{Name: "close", Pos: b.Name.Pos}, Pos: b.Name.Pos}, Pos: b.Name.Pos}
	closeCall := f.checkExpr(call, types.TUnit)
	if isResultType(closeCall.Type()) {
		f.errorf(b.Name.Pos, "close() of '%s' throws; throwing cleanup is not supported yet", init.Type())
	}
	inner, bodyT := f.withBindings(s, i+1, closeable, want, asValue, result)
	body := &Block{Stmts: inner, Type: types.TUnit}
	if types.IsNever(bodyT) {
		body.Type = types.TNever
	}
	return []Stmt{&With{Var: v, Init: init, Close: closeCall, Body: body}}, bodyT
}

// checkThrow is `throw e`: `return Err(e)` in a throwing function (D4).
func (f *fnCtx) checkThrow(s *ast.ThrowStmt) []Stmt {
	if s.Value == nil {
		return nil
	}
	if f.isGlobal {
		f.errorf(s.Pos, "'throw' outside of a function")
		return nil
	}
	if !f.throws {
		f.errorf(s.Pos, "'throw' fails the function, but it is not declared 'throws' (D4)")
		f.checkExpr(s.Value, nil)
		return nil
	}
	errv := f.checkExpr(s.Value, nil)
	x := f.throwExpr(errv, s.Pos)
	if types.IsInvalid(x.Type()) {
		return nil
	}
	return []Stmt{&ExprStmt{X: x}}
}
