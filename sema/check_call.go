package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

func (f *fnCtx) callExpr(e *ast.CallExpr, want types.Type) Expr {
	if e.Async {
		return f.launch(e, want)
	}
	var typeArgs []types.Type
	for _, ta := range e.TypeArgs {
		typeArgs = append(typeArgs, f.resolve(ta))
	}
	switch callee := e.Fun.(type) {
	case *ast.NameExpr:
		if isCollectionCtor(callee.Name) && f.lookup(callee.Name) == nil {
			return f.collectionCtor(callee.Name, typeArgs, e, want)
		}
		if callee.Name == "Channel" && f.lookup(callee.Name) == nil {
			return f.channelCtor(typeArgs, e, want)
		}
		if callee.Name == "sleep" && f.lookup(callee.Name) == nil {
			return f.sleepCall(e)
		}
		sym := f.lookup(callee.Name)
		if sym == nil {
			f.errorf(callee.Pos, "unknown function '%s'", callee.Name)
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		return f.callSymbol(sym, callee.Name, typeArgs, e, want)
	case *ast.MemberExpr:
		if callee.X == nil {
			return f.leadingDot(callee, e.Args, want)
		}
		if n, ok := callee.X.(*ast.NameExpr); ok {
			if sym := f.lookup(n.Name); sym != nil {
				switch sym.Kind {
				case SymModule:
					member := sym.Mod.Scope.LookupLocal(callee.Name.Name)
					if member == nil {
						f.errorf(callee.Name.Pos, "module '%s' has no declaration '%s'", n.Name, callee.Name.Name)
						f.checkArgsLoosely(e.Args)
						return bad()
					}
					if !member.Pub {
						f.errorf(callee.Name.Pos, "'%s' is private to module '%s' (M5)", callee.Name.Name, n.Name)
						f.checkArgsLoosely(e.Args)
						return bad()
					}
					return f.callSymbol(member, callee.Name.Name, typeArgs, e, want)
				case SymType:
					if s, ok := sym.Type.(*types.Sealed); ok {
						v := s.VariantByName(callee.Name.Name)
						if v == nil {
							f.errorf(callee.Name.Pos, "'%s' has no variant '%s'", s.Name, callee.Name.Name)
							f.checkArgsLoosely(e.Args)
							return bad()
						}
						if len(typeArgs) > 0 {
							v = f.c.instantiateStruct(v, typeArgs, e.Pos)
						}
						return f.constructStruct(v, e.Args, e.Pos)
					}
				}
			}
		}
		return f.methodCall(callee, typeArgs, e, want)
	}
	fnv := f.checkExpr(e.Fun, nil)
	if ft, ok := fnv.Type().(*types.Func); ok {
		return f.callValue(fnv, ft, e.Args, e.Pos)
	}
	if !types.IsInvalid(fnv.Type()) {
		f.errorf(e.Fun.Span(), "value of type '%s' is not callable", fnv.Type())
	}
	f.checkArgsLoosely(e.Args)
	return bad()
}

func (f *fnCtx) checkArgsLoosely(args []ast.Arg) {
	for _, a := range args {
		f.checkExpr(a.Value, nil)
	}
}

func (f *fnCtx) callSymbol(sym *Symbol, name string, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	switch sym.Kind {
	case SymFunc:
		return f.callTemplate(sym.Func, nil, typeArgs, e.Args, e.Pos, want)
	case SymType:
		switch t := sym.Type.(type) {
		case *types.Struct:
			st := t
			if len(t.TypeParams) > 0 {
				if len(typeArgs) == 0 {
					st = f.inferStructArgs(t, e.Args, want, e.Pos)
					if st == nil {
						return bad()
					}
				} else {
					if len(typeArgs) != len(t.TypeParams) {
						f.errorf(e.Pos, "'%s' expects %d type arguments, got %d", t.Name, len(t.TypeParams), len(typeArgs))
						return bad()
					}
					st = f.c.instantiateStruct(t, typeArgs, e.Pos)
				}
			} else if len(typeArgs) > 0 {
				f.errorf(e.Pos, "'%s' is not generic", t.Name)
			}
			return f.constructStruct(st, e.Args, e.Pos)
		case *types.Sealed:
			f.errorf(e.Pos, "'%s' is a sealed trait; construct one of its variants, e.g. '%s.%s(...)'", t.Name, t.Name, firstVariantName(t))
		case *types.Basic:
			f.errorf(e.Pos, "'%s' is not callable; convert with 'as'", t.Name)
		default:
			f.errorf(e.Pos, "'%s' is not callable", name)
		}
		f.checkArgsLoosely(e.Args)
		return bad()
	case SymVariantCtor:
		return f.variantCtor(name, e.Args, want, e.Pos)
	case SymLocal, SymGlobal:
		v := f.symbolValue(sym, e.Pos, want)
		if ft, ok := v.Type().(*types.Func); ok {
			return f.callValue(v, ft, e.Args, e.Pos)
		}
		if !types.IsInvalid(v.Type()) {
			f.errorf(e.Pos, "'%s' has type '%s' and is not callable", name, v.Type())
		}
	default:
		f.errorf(e.Pos, "'%s' is not callable", name)
	}
	f.checkArgsLoosely(e.Args)
	return bad()
}

func firstVariantName(s *types.Sealed) string {
	if len(s.Variants) > 0 {
		return s.Variants[0].Name
	}
	return "Variant"
}

// argBinding pairs each parameter with the argument expression supplied
// for it (D28: positional then named, each parameter at most once).
func (f *fnCtx) bindArgs(params []types.Param, args []ast.Arg, what string, span source.Span) ([]ast.Expr, bool) {
	bound := make([]ast.Expr, len(params))
	positional := true
	ok := true
	for i, a := range args {
		if a.Name == nil {
			if !positional {
				f.errorf(a.Value.Span(), "positional argument after a named argument")
				ok = false
				continue
			}
			if i >= len(params) {
				f.errorf(a.Value.Span(), "too many arguments to %s: expected %d", what, len(params))
				ok = false
				break
			}
			bound[i] = a.Value
			continue
		}
		positional = false
		idx := -1
		for j, p := range params {
			if p.Name == a.Name.Name {
				idx = j
			}
		}
		if idx < 0 {
			f.errorf(a.Name.Pos, "%s has no parameter named '%s'", what, a.Name.Name)
			ok = false
			continue
		}
		if bound[idx] != nil {
			f.errorf(a.Name.Pos, "argument '%s' given more than once", a.Name.Name)
			ok = false
			continue
		}
		bound[idx] = a.Value
	}
	return bound, ok
}

// callTemplate checks a call to a function template, inferring any type
// arguments from the arguments (D8 stenciling — one instance per type set).
func (f *fnCtx) callTemplate(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, typeArgs []types.Type, args []ast.Arg, span source.Span, want types.Type) Expr {
	return f.callTemplateRecv(t, ownerSubst, typeArgs, nil, args, span, want)
}

func (f *fnCtx) callTemplateRecv(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, typeArgs []types.Type, recv Expr, args []ast.Arg, span source.Span, want types.Type) Expr {
	f.c.resolveSignature(t)
	f.noteUse(t, span)
	if t.Extern && f.unsafe == 0 {
		f.errorf(span, "calling extern \"C\" function '%s' requires an 'unsafe' block (D44)", t.Name)
	}
	if t.Decl.Unsafe && f.unsafe == 0 {
		f.errorf(span, "calling 'unsafe fun %s' requires an 'unsafe' block (D44)", t.Name)
	}
	m := map[*types.TypeParam]types.Type{}
	for k, v := range ownerSubst {
		m[k] = v
	}
	if len(typeArgs) > 0 {
		if len(typeArgs) != len(t.TypeParams) {
			f.errorf(span, "'%s' expects %d type arguments, got %d", t.Name, len(t.TypeParams), len(typeArgs))
			f.checkArgsLoosely(args)
			return bad()
		}
		for i, tp := range t.TypeParams {
			m[tp] = typeArgs[i]
		}
	}
	what := fmt.Sprintf("'%s'", t.Name)
	bound, ok := f.bindArgs(t.Sig.Params, args, what, span)
	if !ok {
		f.checkArgsLoosely(args)
		return bad()
	}
	// Check arguments; infer type parameters from those whose parameter
	// type mentions one.
	exprs := make([]Expr, len(bound))
	// Lambdas go last so that their parameter types can come from type
	// parameters bound by the other arguments.
	var order []int
	for i := range t.Sig.Params {
		if bound[i] != nil {
			if _, isLambda := bound[i].(*ast.LambdaExpr); !isLambda {
				order = append(order, i)
			}
		}
	}
	for i := range t.Sig.Params {
		if bound[i] != nil {
			if _, isLambda := bound[i].(*ast.LambdaExpr); isLambda {
				order = append(order, i)
			}
		}
	}
	for _, i := range order {
		p := t.Sig.Params[i]
		pt := types.Subst(p.Type, m)
		if types.ContainsTypeParam(pt) {
			var x Expr
			if _, isLambda := bound[i].(*ast.LambdaExpr); isLambda {
				x = f.checkExpr(bound[i], pt)
			} else {
				x = f.checkExpr(bound[i], nil)
			}
			exprs[i] = x
			if types.IsInvalid(x.Type()) {
				continue
			}
			if !unify(pt, x.Type(), m) {
				if n, isN := pt.(*types.Nullable); isN && unify(n.Elem, x.Type(), m) {
					continue
				}
				f.errorf(bound[i].Span(), "cannot infer type parameters: argument of type '%s' does not match parameter type '%s'", x.Type(), pt)
			}
		} else {
			exprs[i] = f.checkExprTo(bound[i], pt)
		}
	}
	// Unbound type parameters may still come from the expected return type.
	if want != nil && types.ContainsTypeParam(types.Subst(t.Sig.Ret, m)) {
		unify(types.Subst(t.Sig.Ret, m), want, m)
	}
	var finalArgs []types.Type
	for _, tp := range t.TypeParams {
		bt, ok := m[tp]
		if !ok {
			f.errorf(span, "cannot infer type parameter '%s' of '%s'; supply it explicitly: '%s<...>(...)'", tp.Name, t.Name, t.Name)
			return bad()
		}
		for _, bound := range tp.Bounds {
			if !f.implements(bt, bound) {
				f.errorf(span, "type '%s' does not implement trait '%s' required by parameter '%s' of '%s'", bt, bound.Name, tp.Name, t.Name)
			}
		}
		finalArgs = append(finalArgs, bt)
	}
	fn := f.c.instantiate(t, ownerSubst, finalArgs, span)
	// Second pass: coerce generic arguments now that types are known, and
	// supply defaults.
	var callArgs []Expr
	if recv != nil {
		callArgs = append(callArgs, recv)
	}
	for i, p := range fn.Sig.Params {
		if bound[i] == nil {
			if !p.HasDefault {
				f.errorf(span, "missing argument '%s' in call to %s", p.Name, what)
				return bad()
			}
			callArgs = append(callArgs, f.defaultArg(t, i, p.Type, fn.subst))
			continue
		}
		x := exprs[i]
		if types.ContainsTypeParam(t.Sig.Params[i].Type) {
			x = f.coerce(x, p.Type, bound[i].Span())
		}
		callArgs = append(callArgs, x)
	}
	rt := fn.Sig.Ret
	if fn.Sig.Effects.Throws {
		rt = f.c.ResultType(fn.Sig.Ret, fn.Sig.Effects.Error)
	}
	return &Call{exprBase{rt}, fn, callArgs}
}

// defaultArg evaluates a parameter's default expression in the callee's
// module context.
func (f *fnCtx) defaultArg(t *FuncTemplate, i int, pt types.Type, subst map[*types.TypeParam]types.Type) Expr {
	env := &typeEnv{module: t.Module, file: t.File, tps: map[string]*types.TypeParam{}}
	g := f.c.newFnCtx(f.fn, t.Module, t.File, env, subst)
	g.unsafe = f.unsafe
	g.throws, g.errType, g.retType = f.throws, f.errType, f.retType
	return g.checkExprTo(t.Decl.Params[i].Default, pt)
}

// implements reports whether a concrete type has an impl for trait.
func (f *fnCtx) implements(t types.Type, trait *types.Trait) bool {
	return f.findImpl(t, trait) != nil
}

func (f *fnCtx) findImpl(t types.Type, trait *types.Trait) *Impl {
	for _, impl := range f.c.impls[trait] {
		m := map[*types.TypeParam]types.Type{}
		if unify(impl.Target, t, m) {
			return impl
		}
	}
	return nil
}

// inferStructArgs infers a generic struct's type arguments from its
// constructor arguments or the expected type.
func (f *fnCtx) inferStructArgs(t *types.Struct, args []ast.Arg, want types.Type, span source.Span) *types.Struct {
	f.c.resolveStruct(t)
	m := map[*types.TypeParam]types.Type{}
	if ws, ok := numericHint(want).(*types.Struct); ok && templateOf(ws) == t {
		return ws
	}
	params := make([]types.Param, len(t.Fields))
	for i, fld := range t.Fields {
		params[i] = types.Param{Name: fld.Name, Type: fld.Type, HasDefault: fld.HasDefault}
	}
	bound, ok := f.bindArgs(params, args, "'"+t.Name+"'", span)
	if !ok {
		return nil
	}
	for i, fld := range t.Fields {
		if bound[i] == nil || !types.ContainsTypeParam(fld.Type) {
			continue
		}
		x := f.checkExpr(bound[i], nil)
		if !types.IsInvalid(x.Type()) {
			unify(fld.Type, x.Type(), m)
		}
	}
	var targs []types.Type
	for _, tp := range t.TypeParams {
		bt, ok := m[tp]
		if !ok {
			f.errorf(span, "cannot infer type parameter '%s' of '%s'; write '%s<...>(...)' or annotate the binding", tp.Name, t.Name, t.Name)
			return nil
		}
		targs = append(targs, bt)
	}
	return f.c.instantiateStruct(t, targs, span)
}

// constructStruct implements the implicit constructor (D28).
func (f *fnCtx) constructStruct(st *types.Struct, args []ast.Arg, span source.Span) Expr {
	f.c.resolveStruct(templateOf(st))
	if len(st.TypeParams) > 0 && st.TypeArgs == nil {
		f.errorf(span, "'%s' is generic; supply type arguments", st.Name)
		return bad()
	}
	if templateOf(st).Module != f.module.prefix() {
		for _, fld := range st.Fields {
			if !fld.Pub {
				f.errorf(span, "cannot construct '%s' here: field '%s' is private, so the implicit constructor is only callable inside module '%s' (D28)", st.Name, fld.Name, st.Module)
				f.checkArgsLoosely(args)
				return bad()
			}
		}
	}
	params := make([]types.Param, len(st.Fields))
	for i, fld := range st.Fields {
		params[i] = types.Param{Name: fld.Name, Type: fld.Type, HasDefault: fld.HasDefault}
	}
	bound, ok := f.bindArgs(params, args, "'"+st.Name+"'", span)
	if !ok {
		f.checkArgsLoosely(args)
		return bad()
	}
	lit := &StructLit{exprBase{st}, st, make([]Expr, len(st.Fields))}
	for i, fld := range st.Fields {
		if bound[i] != nil {
			lit.Fields[i] = f.checkExprTo(bound[i], fld.Type)
			continue
		}
		if !fld.HasDefault {
			f.errorf(span, "missing field '%s' in constructor of '%s'", fld.Name, st.Name)
			lit.Fields[i] = bad()
			continue
		}
		lit.Fields[i] = f.fieldDefault(st, i)
	}
	if st.Sealed != nil {
		return &MakeVariant{exprBase{st.Sealed}, st.Sealed, st, lit}
	}
	return lit
}

func (f *fnCtx) fieldDefault(st *types.Struct, i int) Expr {
	tmpl := templateOf(st)
	ctx := f.c.structDecl[tmpl]
	if ctx == nil {
		return bad()
	}
	d := ctx.decl.(*ast.StructDecl)
	env := f.c.envFor(ctx, st)
	g := f.c.newFnCtx(f.fn, ctx.module, ctx.file, env, substOf(st))
	return g.checkExprTo(d.Fields[i].Default, st.Fields[i].Type)
}

// ---------------------------------------------------------------------------
// method calls

func (f *fnCtx) methodCall(callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	recv := f.checkExpr(callee.X, nil)
	if types.IsInvalid(recv.Type()) {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if callee.Safe {
		nt, ok := recv.Type().(*types.Nullable)
		if !ok {
			f.errorf(callee.Pos, "'?.' on a non-nullable value of type '%s'; use '.'", recv.Type())
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		tmp := f.newTemp(nt)
		inner := f.dispatchMethod(&Unwrap{exprBase{nt.Elem}, &VarRef{exprBase{nt}, tmp}}, callee, typeArgs, e, want)
		if types.IsInvalid(inner.Type()) {
			return inner
		}
		rt := types.Type(&types.Nullable{Elem: inner.Type()})
		if _, isN := inner.Type().(*types.Nullable); isN {
			rt = inner.Type()
		} else if types.IsUnit(inner.Type()) {
			rt = types.TUnit
		} else {
			inner = &SomeWrap{exprBase{rt}, inner}
		}
		var elseBlock *Block
		if types.IsUnit(rt) {
			elseBlock = &Block{Type: rt}
		} else {
			elseBlock = &Block{Value: &NullConst{exprBase{rt}}, Type: rt}
		}
		thenBlock := &Block{Value: inner, Type: rt}
		if types.IsUnit(rt) {
			thenBlock = &Block{Stmts: []Stmt{&ExprStmt{X: inner}}, Type: rt}
		}
		body := &If{exprBase{rt}, &IsNull{exprBase{types.TBool}, &VarRef{exprBase{nt}, tmp}}, elseBlock, thenBlock}
		return &Let{exprBase{rt}, tmp, recv, body}
	}
	return f.dispatchMethod(recv, callee, typeArgs, e, want)
}

func (f *fnCtx) dispatchMethod(recv Expr, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	name := callee.Name.Name
	rt := recv.Type()
	// auto-deref (D39)
	viaPointer := false
	if p, ok := rt.(*types.Pointer); ok {
		if p.Raw {
			f.errorf(callee.Pos, "raw pointers have no methods; dereference inside 'unsafe' first (D50)")
			return bad()
		}
		viaPointer = true
		recv = &Deref{exprBase{p.Elem}, recv}
		rt = p.Elem
	}
	if b := f.builtinMethod(recv, rt, name, e); b != nil {
		return b
	}
	switch ct := rt.(type) {
	case *types.Sealed:
		if tr := sealedTemplate(ct).Trait; tr != nil {
			if _, has := tr.Methods[name]; has {
				return f.sealedDispatch(recv, ct, callee, typeArgs, e, want)
			}
		}
	case *types.Trait:
		return f.virtualCall(recv, ct, callee, e)
	case *types.Channel:
		return f.channelMethod(recv, ct, name, e)
	case *types.Map:
		return f.mapMethod(recv, ct, name, e)
	case *types.Set:
		return f.setMethod(recv, ct, name, e)
	}
	// inherent methods
	if st, ok := rt.(*types.Struct); ok {
		tmpl := templateOf(st)
		if t, ok := f.c.methods[tmpl][name]; ok {
			return f.callMethod(t, substOf(st), typeArgs, recv, viaPointer, callee, e, want)
		}
	}
	// trait impls (D26: any trait method is callable; ambiguity is an error)
	var found []*FuncTemplate
	var foundSubst []map[*types.TypeParam]types.Type
	for trait, impls := range f.c.impls {
		if _, has := trait.Methods[name]; !has {
			continue
		}
		for _, impl := range impls {
			m := map[*types.TypeParam]types.Type{}
			if !unify(impl.Target, rt, m) {
				continue
			}
			if t, ok := impl.Methods[name]; ok {
				found = append(found, t)
				foundSubst = append(foundSubst, m)
			} else if dt := f.c.traitDefault(trait, name); dt != nil {
				m[selfParamOf(trait)] = rt
				found = append(found, dt)
				foundSubst = append(foundSubst, m)
			}
		}
	}
	if len(found) > 1 {
		var names []string
		for _, t := range found {
			tr := t.Trait
			if t.Impl != nil {
				tr = t.Impl.Trait
			}
			names = append(names, tr.Name)
		}
		f.errorf(callee.Name.Pos, "ambiguous method '%s' on '%s': provided by traits %s (D26)", name, rt, strings.Join(names, " and "))
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if len(found) == 1 {
		return f.callMethod(found[0], foundSubst[0], typeArgs, recv, viaPointer, callee, e, want)
	}
	// a field holding a function value: `self.f(x)`
	if st, ok := rt.(*types.Struct); ok {
		for _, fld := range st.Fields {
			if fld.Name == name {
				if ft, isFn := fld.Type.(*types.Func); isFn {
					return f.callValue(&FieldGet{exprBase{fld.Type}, recv, fld.Index, fld.Name}, ft, e.Args, e.Pos)
				}
			}
		}
	}
	switch tt := rt.(type) {
	case *types.Nullable:
		f.errorf(callee.Pos, "value of type '%s' may be null; use '?.' or check for null first (D5)", tt)
	case *types.Sealed:
		f.errorf(callee.Name.Pos, "no method '%s' on sealed trait '%s'; match on its variants with 'when' (D13)", name, tt)
	case *types.Trait:
		f.errorf(callee.Pos, "trait objects (boxed '%s') are not supported yet in the bootstrap compiler; use a generic bound instead (D9)", tt)
	default:
		f.errorf(callee.Name.Pos, "no method '%s' on type '%s'", name, rt)
	}
	f.checkArgsLoosely(e.Args)
	return bad()
}

func (f *fnCtx) callMethod(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, typeArgs []types.Type, recv Expr, viaPointer bool, callee *ast.MemberExpr, e *ast.CallExpr, want types.Type) Expr {
	f.c.resolveSignature(t)
	if t.Impl != nil && !t.Pub && t.Impl.Module != f.module {
		// impl methods follow the trait's visibility; nothing extra
	}
	if t.Owner != nil && !t.Pub && t.Module != f.module {
		f.errorf(callee.Name.Pos, "method '%s' is private to module '%s' (M5)", t.Name, t.Module.Path)
	}
	var recvArg Expr = recv
	if t.Decl.Mut {
		// D22: a mut method needs a mutable place.
		if viaPointer {
			recvArg = recv.(*Deref).X
		} else {
			if !isPlaceSyntax(callee.X) {
				// a temporary: mutate a fresh copy (iterator chains rely on this)
				tmp := f.newTemp(recv.Type())
				recvArg = &AddrOf{exprBase{&types.Pointer{Elem: recv.Type()}}, ref(tmp)}
				call := f.callTemplateRecv(t, ownerSubst, typeArgs, recvArg, e.Args, e.Pos, want)
				return &Let{exprBase{call.Type()}, tmp, recv, call}
			}
			lv, root := f.checkLValue(callee.X, true)
			if lv == nil {
				return bad()
			}
			if root != nil && !root.Mutable && root != f.selfVar {
				// already reported by checkLValue
			}
			recvArg = &AddrOf{exprBase{&types.Pointer{Elem: lv.Type()}}, lv}
		}
	}
	return f.callTemplateRecv(t, ownerSubst, typeArgs, recvArg, e.Args, e.Pos, want)
}

// builtinMethod resolves methods on string, List and MutableList.
func (f *fnCtx) builtinMethod(recv Expr, rt types.Type, name string, e *ast.CallExpr) Expr {
	nargs := func(n int) bool {
		if len(e.Args) != n {
			f.errorf(e.Pos, "'%s' takes %d argument(s)", name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	switch t := rt.(type) {
	case *types.Basic:
		if t.Kind == types.String {
			switch name {
			case "len":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{types.TI64}, "string.len", []Expr{recv}, e.Pos}
			case "isEmpty":
				if !nargs(0) {
					return bad()
				}
				return &Binary{exprBase{types.TBool}, OpEq, &Builtin{exprBase{types.TI64}, "string.len", []Expr{recv}, e.Pos}, &IntConst{exprBase{types.TI64}, 0, false}, e.Pos}
			case "startsWith", "endsWith", "contains":
				if !nargs(1) {
					return bad()
				}
				arg := f.checkExprTo(e.Args[0].Value, types.TString)
				return &Builtin{exprBase{types.TBool}, "string." + name, []Expr{recv, arg}, e.Pos}
			case "substring":
				if !nargs(2) {
					return bad()
				}
				lo := f.checkExprTo(e.Args[0].Value, types.TI64)
				hi := f.checkExprTo(e.Args[1].Value, types.TI64)
				// D19: a slice that splits a code point yields string?
				return &Builtin{exprBase{&types.Nullable{Elem: types.TString}}, "string.substring", []Expr{recv, lo, hi}, e.Pos}
			case "toInt":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{&types.Nullable{Elem: types.TI64}}, "string.toInt", []Expr{recv}, e.Pos}
			}
		}
	case *types.List:
		switch name {
		case "len":
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{types.TI64}, "list.len", []Expr{recv}, e.Pos}
		case "isEmpty":
			if !nargs(0) {
				return bad()
			}
			return &Binary{exprBase{types.TBool}, OpEq, &Builtin{exprBase{types.TI64}, "list.len", []Expr{recv}, e.Pos}, &IntConst{exprBase{types.TI64}, 0, false}, e.Pos}
		case "toList":
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{&types.List{Elem: t.Elem}}, "list.copy", []Expr{recv}, e.Pos}
		case "toMutable":
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{&types.List{Elem: t.Elem, Mutable: true}}, "list.copy", []Expr{recv}, e.Pos}
		case "push":
			if !t.Mutable {
				f.errorf(e.Pos, "cannot push into an immutable List; use MutableList (D25)")
				f.checkArgsLoosely(e.Args)
				return bad()
			}
			if !nargs(1) {
				return bad()
			}
			x := f.checkExprTo(e.Args[0].Value, t.Elem)
			return &Builtin{exprBase{types.TUnit}, "list.push", []Expr{recv, x}, e.Pos}
		case "pop":
			if !t.Mutable {
				f.errorf(e.Pos, "cannot pop from an immutable List; use MutableList (D25)")
				return bad()
			}
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{&types.Nullable{Elem: t.Elem}}, "list.pop", []Expr{recv}, e.Pos}
		case "clear":
			if !t.Mutable {
				f.errorf(e.Pos, "cannot clear an immutable List; use MutableList (D25)")
				return bad()
			}
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{types.TUnit}, "list.clear", []Expr{recv}, e.Pos}
		default:
			return f.listAdapter(recv, t, name, e)
		}
	}
	return nil
}

// callValue calls a function value (D28: positional arguments only).
func (f *fnCtx) callValue(fnv Expr, ft *types.Func, args []ast.Arg, span source.Span) Expr {
	if len(args) != len(ft.Params) {
		f.errorf(span, "function value takes %d argument(s), got %d", len(ft.Params), len(args))
		f.checkArgsLoosely(args)
		return bad()
	}
	var vals []Expr
	for i, a := range args {
		if a.Name != nil {
			f.errorf(a.Name.Pos, "named arguments apply only to direct calls of named functions, not function values (D28)")
		}
		vals = append(vals, f.checkExprTo(a.Value, ft.Params[i].Type))
	}
	rt := ft.Ret
	if ft.Effects.Throws {
		rt = f.c.ResultType(ft.Ret, ft.Effects.Error)
	}
	return &CallIndirect{exprBase{rt}, fnv, vals}
}

// funcValue turns a named, non-generic function into a value.
func (f *fnCtx) funcValue(t *FuncTemplate, span source.Span) Expr {
	f.c.resolveSignature(t)
	if len(t.TypeParams) > 0 {
		f.errorf(span, "generic function '%s' cannot be used as a value without instantiation", t.Name)
		return bad()
	}
	if t.Extern || t.Decl.Unsafe {
		f.errorf(span, "unsafe or extern function '%s' cannot be used as a value (D44)", t.Name)
		return bad()
	}
	if t.Owner != nil || t.Impl != nil || t.Trait != nil {
		f.errorf(span, "methods cannot be used as values; wrap the call in a lambda")
		return bad()
	}
	fn := f.c.instantiate(t, nil, nil, span)
	return &FuncRef{exprBase{fn.Sig}, fn}
}

// objectSafe reports whether a trait can be used as a trait object: no
// associated types, no generic methods, no Self in signatures.
func (f *fnCtx) objectSafe(trait *types.Trait) (string, bool) {
	if len(trait.AssocTypes) > 0 {
		return "it has associated types", false
	}
	self := selfParamOf(trait)
	for _, name := range trait.MethodList {
		if len(f.c.traitMethodTPs[trait.Name+"."+name]) > 0 {
			return "method '" + name + "' is generic", false
		}
		sig := trait.Methods[name]
		for _, p := range sig.Params {
			if mentions(p.Type, self) {
				return "method '" + name + "' takes Self", false
			}
		}
		if mentions(sig.Ret, self) {
			return "method '" + name + "' returns Self", false
		}
	}
	return "", true
}

func mentions(t types.Type, p *types.TypeParam) bool {
	switch t := t.(type) {
	case *types.TypeParam:
		return t == p
	case *types.Pointer:
		return mentions(t.Elem, p)
	case *types.Nullable:
		return mentions(t.Elem, p)
	case *types.List:
		return mentions(t.Elem, p)
	case *types.Tuple:
		for _, e := range t.Elems {
			if mentions(e, p) {
				return true
			}
		}
	case *types.Func:
		for _, q := range t.Params {
			if mentions(q.Type, p) {
				return true
			}
		}
		return mentions(t.Ret, p)
	case *types.Assoc:
		return mentions(t.Base, p)
	}
	return false
}

// boxValue builds the trait object for a concrete value.
func (f *fnCtx) boxValue(x Expr, trait *types.Trait, span source.Span) Expr {
	t := x.Type()
	if reason, ok := f.objectSafe(trait); !ok {
		f.errorf(span, "'%s' cannot be a trait object: %s (D9)", trait.Name, reason)
		return bad()
	}
	impl := f.findImpl(t, trait)
	if impl == nil {
		f.errorf(span, "'%s' does not implement '%s'", t, trait.Name)
		return bad()
	}
	m := map[*types.TypeParam]types.Type{}
	unify(impl.Target, t, m)
	box := &Box{exprBase{trait}, x, trait, nil, nil}
	for _, name := range trait.MethodList {
		var tmpl *FuncTemplate
		subst := map[*types.TypeParam]types.Type{}
		for k, v := range m {
			subst[k] = v
		}
		if mt, ok := impl.Methods[name]; ok {
			tmpl = mt
		} else {
			tmpl = f.c.traitDefault(trait, name)
			subst[selfParamOf(trait)] = t
		}
		fn := f.c.instantiate(tmpl, subst, nil, span)
		box.Methods = append(box.Methods, fn)
		box.Mut = append(box.Mut, tmpl.Decl.Mut)
	}
	return box
}

// virtualCall checks a method call on a trait object.
func (f *fnCtx) virtualCall(recv Expr, trait *types.Trait, callee *ast.MemberExpr, e *ast.CallExpr) Expr {
	name := callee.Name.Name
	idx := -1
	for i, n := range trait.MethodList {
		if n == name {
			idx = i
		}
	}
	if idx < 0 {
		f.errorf(callee.Name.Pos, "trait '%s' has no method '%s'", trait.Name, name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	sig := trait.Methods[name]
	bound, ok := f.bindArgs(sig.Params, e.Args, "'"+name+"'", e.Pos)
	if !ok {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	var args []Expr
	for i, p := range sig.Params {
		if bound[i] == nil {
			f.errorf(e.Pos, "missing argument '%s' in call to '%s'", p.Name, name)
			return bad()
		}
		args = append(args, f.checkExprTo(bound[i], p.Type))
	}
	rt := sig.Ret
	if sig.Effects.Throws {
		rt = f.c.ResultType(sig.Ret, sig.Effects.Error)
	}
	if sig.Effects.Suspends {
		f.errorf(e.Pos, "suspending trait methods are not supported yet")
	}
	return &CallVirtual{exprBase{rt}, recv, trait, idx, args}
}

// sealedDispatch lowers a method call on a sealed value to a tag switch
// over its variants, each calling that variant's impl (D12/D23).
func (f *fnCtx) sealedDispatch(recv Expr, s *types.Sealed, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	trait := sealedTemplate(s).Trait
	name := callee.Name.Name
	tmp := f.newTemp(s)
	m := &Match{Subject: tmp, Init: recv, Exhaustive: true, Span: e.Pos}
	var rt types.Type
	for _, v := range s.Variants {
		if f.findImpl(v, trait) == nil && f.c.traitDefault(trait, name) == nil {
			f.errorf(e.Pos, "variant '%s.%s' does not implement '%s'", s.Name, v.Name, name)
			return bad()
		}
		payload := &VariantCast{exprBase{v}, ref(tmp), v}
		call := f.dispatchMethod(payload, callee, typeArgs, e, want)
		if types.IsInvalid(call.Type()) {
			return bad()
		}
		if rt == nil {
			rt = call.Type()
		} else if !types.Identical(rt, call.Type()) {
			f.errorf(e.Pos, "variants of '%s' disagree on the type of '%s'", s.Name, name)
			return bad()
		}
		body := &Block{Value: call, Type: rt}
		if types.IsUnit(rt) {
			body = &Block{Stmts: []Stmt{&ExprStmt{X: call}}, Type: types.TUnit}
		}
		m.Arms = append(m.Arms, &MatchArm{Test: &VariantTest{exprBase{types.TBool}, ref(tmp), v}, Body: body})
	}
	m.T = rt
	return m
}
