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
	case *ast.TypeExpr:
		// a resolved struct as the constructor: synthesized code only (D58)
		if st, ok := f.resolve(callee.Type).(*types.Struct); ok {
			return f.constructStruct(st, e.Args, e.Pos)
		}
		f.errorf(callee.Pos, "'%s' is not a struct", f.resolve(callee.Type))
		f.checkArgsLoosely(e.Args)
		return bad()
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
		if callee.Name == "ioWait" && f.module.Std && f.lookup(callee.Name) == nil {
			return f.ioWaitCall(e)
		}
		if callee.Name == "listRawData" && f.module.Std && f.lookup(callee.Name) == nil && len(e.Args) == 1 {
			// std-only (D69): the storage of a list, for `withRaw` to lend C
			// for the length of a closure — never a pointer user code holds
			xs := f.checkExpr(e.Args[0].Value, nil)
			lt, ok := xs.Type().(*types.List)
			if !ok {
				f.errorf(e.Pos, "listRawData takes a list")
				return bad()
			}
			return &Builtin{exprBase{&types.Pointer{Elem: lt.Elem, Raw: true}}, "list.rawData", []Expr{xs}, e.Pos}
		}
		if callee.Name == "listTouched" && f.module.Std && f.lookup(callee.Name) == nil && len(e.Args) == 1 {
			// std-only: an order change made element by element (sort, swap)
			// counts as a change of the list for a loop walking it (D102)
			xs := f.checkExpr(e.Args[0].Value, nil)
			if lt, ok := xs.Type().(*types.List); !ok || !lt.Mutable {
				f.errorf(e.Pos, "listTouched takes a MutableList")
				return bad()
			}
			return &Builtin{exprBase{types.TUnit}, "list.touch", []Expr{xs}, e.Pos}
		}
		if callee.Name == "listAppendText" && f.module.Std && f.lookup(callee.Name) == nil && len(e.Args) == 2 {
			// std-only: a string's bytes onto a MutableList<u8> in one copy
			// (StringBuilder.append), not a list of them pushed one by one
			xs := f.checkExpr(e.Args[0].Value, nil)
			s := f.checkExprTo(e.Args[1].Value, types.TString)
			if lt, ok := xs.Type().(*types.List); !ok || !lt.Mutable || !types.Identical(lt.Elem, types.TU8) {
				f.errorf(e.Pos, "listAppendText takes a MutableList<u8> and a string")
				return bad()
			}
			return &Builtin{exprBase{types.TUnit}, "list.appendText", []Expr{xs, s}, e.Pos}
		}
		if callee.Name == "stringSplit" && f.module.Std && f.lookup(callee.Name) == nil && len(e.Args) == 2 {
			// std-only: String.split's loop in the runtime (a non-empty sep;
			// the parts share the text's bytes)
			s := f.checkExprTo(e.Args[0].Value, types.TString)
			sep := f.checkExprTo(e.Args[1].Value, types.TString)
			return &Builtin{exprBase{&types.List{Elem: types.TString}}, "string.split", []Expr{s, sep}, e.Pos}
		}
		if callee.Name == "listDecodeUtf8Range" && f.module.Std && f.lookup(callee.Name) == nil && len(e.Args) == 3 {
			// std-only: xs[from:to] as text in one copy (a parser's string
			// out of its input), not a slice and then a second copy
			xs := f.checkExpr(e.Args[0].Value, nil)
			lo := f.checkExprTo(e.Args[1].Value, types.TI64)
			hi := f.checkExprTo(e.Args[2].Value, types.TI64)
			if lt, ok := xs.Type().(*types.List); !ok || !types.Identical(lt.Elem, types.TU8) {
				f.errorf(e.Pos, "listDecodeUtf8Range takes a List<u8> and two positions")
				return bad()
			}
			return &Builtin{exprBase{&types.Nullable{Elem: types.TString}}, "list.decodeUtf8Range", []Expr{xs, lo, hi}, e.Pos}
		}
		if _, ok := atomicBuiltins[callee.Name]; ok && f.module.Std && f.lookup(callee.Name) == nil {
			return f.atomicCall(callee.Name, e)
		}
		if callee.Name == "panic" && f.lookup(callee.Name) == nil {
			return f.panicCall(e)
		}
		if (testWords[callee.Name] || callee.Name == "assert") && f.lookup(callee.Name) == nil {
			return f.testCall(callee.Name, typeArgs, e, want)
		}
		if callee.Name == "$testFail" && len(e.Args) == 1 {
			return f.testFailCall(e) // built by the test vocabulary
		}
		sym := f.lookup(callee.Name)
		if sym == nil {
			if suite, ok := f.c.suiteHelpers[callee.Name]; ok {
				f.errorf(callee.Pos, "'%s' is a helper of suite %q, visible only inside it; move the test into the suite, or the helper out of it (D78)", callee.Name, suite)
			} else if callee.Name == "check" {
				// Kotlin's check(cond) — and this compiler's own, briefly
				f.errorf(callee.Pos, "unknown function 'check'; an invariant is 'assert(cond, \"why it must hold\")' (D78)")
			} else if !f.removedFactory(callee.Name, e) {
				hint, fix := f.unknownNameHint(callee.Pos, callee.Name, false)
				f.c.errorFix(callee.Pos, fix, "unknown function '%s'%s", callee.Name, hint)
			}
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		f.c.refSym(callee.Pos, sym)
		if sym.Kind == SymFunc && sym.Func.TestCode && !f.inTest() {
			f.errorf(callee.Pos, "'%s' is test code (a 'test fun', or declared in a *.test.vs file); only tests can call it, and a build leaves it out (D78)", callee.Name)
		}
		return f.traceHelperCall(sym, f.callSymbol(sym, callee.Name, typeArgs, e, want), e)
	case *ast.PreludeName:
		// synthesized code (D58): the prelude's function, whatever the
		// module declares under the same name
		if callee.Name == "panic" {
			return f.panicCall(e)
		}
		sym := f.c.preludeSym(callee.Name)
		if sym == nil || sym.Kind != SymFunc {
			panic("synthesized call to a missing prelude function " + callee.Name)
		}
		return f.callSymbol(sym, callee.Name, typeArgs, e, want)
	case *ast.MemberExpr:
		if n, ok := callee.X.(*ast.NameExpr); ok {
			if sym := f.lookup(n.Name); sym != nil {
				switch sym.Kind {
				case SymModule:
					member := sym.Mod.Scope.LookupLocal(callee.Name.Name)
					if member == nil {
						if !f.removedFactory(n.Name+"."+callee.Name.Name, e) {
							f.c.noMember(n.Pos, callee.Name.Pos, n.Name, callee.Name.Name, sym.Mod)
						}
						f.checkArgsLoosely(e.Args)
						return bad()
					}
					if !member.Pub {
						f.errorf(callee.Name.Pos, "'%s' is private to module '%s'%s (M5)", callee.Name.Name, n.Name, privateHint(sym.Mod.Path))
						f.checkArgsLoosely(e.Args)
						return bad()
					}
					f.c.refSym(n.Pos, sym)
					f.c.refSym(callee.Name.Pos, member)
					if member.Kind == SymFunc && member.Func.TestCode && !f.inTest() {
						f.errorf(callee.Name.Pos, "'%s' is test code (a 'test fun', or declared in a *.test.vs file); only tests can call it, and a build leaves it out (D78)", callee.Name.Name)
					}
					return f.traceHelperCall(member, f.callSymbol(member, callee.Name.Name, typeArgs, e, want), e)
				case SymType:
					if s, ok := sym.Type.(*types.Sealed); ok {
						v := s.VariantByName(callee.Name.Name)
						if v == nil {
							// `Shape.decode(from)`: a static function of a trait
							// implemented for the family (D58)
							if t, _, _ := f.findMethod(s, callee.Name.Name); t != nil && t.Decl.Static {
								f.c.refSym(n.Pos, sym)
								return f.staticCall(s, callee, typeArgs, e, want)
							}
							f.errorf(callee.Name.Pos, "'%s' has no variant '%s'", s.Name, callee.Name.Name)
							f.checkArgsLoosely(e.Args)
							return bad()
						}
						f.c.refSym(n.Pos, sym)
						f.c.refType(callee.Name.Pos, v.Name, v, variantDefSpan(v))
						if len(typeArgs) > 0 {
							v = f.c.instantiateStruct(v, typeArgs, e.Pos)
						}
						return f.constructStruct(v, e.Args, e.Pos)
					}
				}
			}
			if rt := f.typeNamed(n, callee.Name.Name); rt != nil {
				// `Type.f(args)`: a static function (D23)
				if len(typeArgs) == 0 {
					if st, ok := rt.(*types.Struct); ok && len(st.TypeParams) > 0 && st.TypeArgs == nil {
						if !f.staticNamed(st, callee.Name.Name) {
							// the name, not the type arguments, is what is wrong
							f.errorf(callee.Name.Pos, "no static function '%s' on type '%s'%s", callee.Name.Name, st.Name, f.staticHint(st, callee.Name.Name))
							f.checkArgsLoosely(e.Args)
							return bad()
						}
						f.errorf(n.Pos, "'%s' is generic; write the type arguments, e.g. '%s<T>.%s(...)'", st.Name, st.Name, callee.Name.Name)
						f.checkArgsLoosely(e.Args)
						return bad()
					}
				}
				return f.staticCall(rt, callee, typeArgs, e, want)
			}
		}
		if rt := f.moduleTypeNamed(callee.X); rt != nil {
			// `module.Type.f(args)`
			if st, ok := rt.(*types.Struct); ok && len(st.TypeParams) > 0 && st.TypeArgs == nil {
				f.errorf(callee.X.Span(), "'%s' is generic; write the type arguments, e.g. '%s<T>.%s(...)'", st.Name, st.Name, callee.Name.Name)
				f.checkArgsLoosely(e.Args)
				return bad()
			}
			return f.staticCall(rt, callee, typeArgs, e, want)
		}
		if te, ok := callee.X.(*ast.TypeExpr); ok {
			// a resolved type as the receiver: synthesized code only (D58)
			return f.staticCall(f.resolve(te.Type), callee, typeArgs, e, want)
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
	f.looseArgs++
	for _, a := range args {
		f.checkExpr(a.Value, nil)
	}
	f.looseArgs--
}

// inLooseArgs: an argument of a call that already failed is being checked;
// a lambda there cannot know its parameter types, and saying so is noise.
func (f *fnCtx) inLooseArgs() bool {
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.looseArgs > 0 {
			return true
		}
	}
	return false
}

func (f *fnCtx) callSymbol(sym *Symbol, name string, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	switch sym.Kind {
	case SymFunc:
		return f.callTemplate(sym.Func, nil, typeArgs, e.Args, e.Pos, want)
	case SymType:
		symT := sym.Type
		if sym.TypeAlias != nil {
			// `P(x: 1)` with `type P = geo.Point`: the alias names the struct;
			// a generic alias takes its arguments here (`Pair<i64>(...)`)
			if len(typeArgs) > 0 {
				symT = f.c.resolveTypeAlias(sym, typeArgs, e.Pos)
				typeArgs = nil
			} else {
				symT = f.c.symType(sym)
				if symT == nil {
					f.errorf(e.Pos, "'%s' is generic; write the type arguments, e.g. '%s<T>(...)'", sym.Name, sym.Name)
					f.checkArgsLoosely(e.Args)
					return bad()
				}
			}
			if types.IsInvalid(symT) {
				f.checkArgsLoosely(e.Args)
				return bad()
			}
		}
		switch t := symT.(type) {
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
				f.errorf(e.Pos, "'%s' is not generic: write it without '<...>'", t.Name)
			}
			return f.constructStruct(st, e.Args, e.Pos)
		case *types.Sealed:
			f.errorf(e.Pos, "'%s' is a sealed trait; construct one of its variants, e.g. '%s.%s(...)'", t.Name, t.Name, firstVariantName(t))
		case *types.Basic:
			f.errorf(e.Pos, "'%s' is not callable; convert with a method: 'x.to%s()' (returns '%s?' when it can lose) or 'x.wrap%s()' (D86)", t.Name, upperFirst(t.Name), t.Name, upperFirst(t.Name))
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
	// A variadic last parameter takes the trailing positional arguments as
	// a list literal, or one `xs...` argument as the list itself.
	if n := len(params); n > 0 && params[n-1].Variadic {
		v := n - 1
		var trailing []ast.Expr
		rest := args
		for i, a := range args {
			if a.Name != nil {
				break
			}
			if i < v {
				continue
			}
			if a.Spread {
				if len(trailing) > 0 || i+1 < len(args) && args[i+1].Name == nil {
					f.errorf(a.Value.Span(), "'...' must be the only argument for '%s'", params[v].Name)
					ok = false
				}
				bound[v] = a.Value
				trailing = nil
				rest = append(append([]ast.Arg{}, args[:v]...), args[i+1:]...)
				break
			}
			trailing = append(trailing, a.Value)
			rest = append(append([]ast.Arg{}, args[:v]...), args[i+1:]...)
		}
		if bound[v] == nil && len(trailing) > 0 {
			bound[v] = &ast.ListLit{Elems: trailing, Pos: trailing[0].Span().To(trailing[len(trailing)-1].Span())}
		}
		args = rest
		defer func() {
			if bound[v] == nil {
				bound[v] = &ast.ListLit{Pos: span} // no argument: the empty list
			}
		}()
	}
	for _, a := range args {
		if a.Spread {
			f.errorf(a.Value.Span(), "'...' spreads a list into a variadic parameter; %s has none here", what)
			ok = false
		}
	}
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
		if idx < 0 && a.Name.Name == "mergeStderr" {
			// D82: `os.run`'s flag became `stderr: os.Stderr`; `true` merged,
			// and `false` — the old default — let standard error through
			repl := "stderr: os.Stderr.Merge"
			if b, isBool := a.Value.(*ast.BoolLit); isBool && !b.Value {
				repl = "stderr: os.Stderr.Inherit"
			}
			span := source.Span{File: a.Name.Pos.File, Start: a.Name.Pos.Start, End: a.Value.Span().End}
			f.c.errorFix(a.Name.Pos, fixReplace("Write '"+repl+"'", span, repl), "'mergeStderr' was removed: write '%s' (D82)", repl)
			ok = false
			continue
		}
		if idx < 0 {
			if strings.HasPrefix(what, "struct ") {
				f.errorf(a.Name.Pos, "%s has no field named '%s'", what, a.Name.Name)
			} else {
				f.errorf(a.Name.Pos, "%s has no parameter named '%s'", what, a.Name.Name)
			}
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
	for i, p := range params {
		if p.Lazy && bound[i] != nil {
			bound[i] = lazyArgument(bound[i]) // D90
		}
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
	f.c.refArgLabels(args, t)
	bound, ok := f.bindArgs(t.Sig.Params, args, what, span)
	if !ok {
		f.checkArgsLoosely(args)
		return bad()
	}
	// Check arguments; infer type parameters from those whose parameter
	// type mentions one.
	exprs := make([]Expr, len(bound))
	// Lambdas and empty collection literals go last so that their types can
	// come from type parameters bound by the other arguments.
	var order []int
	for i := range t.Sig.Params {
		if bound[i] != nil && !deferredArg(bound[i]) {
			order = append(order, i)
		}
	}
	for i := range t.Sig.Params {
		if bound[i] != nil && deferredArg(bound[i]) {
			order = append(order, i)
		}
	}
	// a literal argument takes its type from the expected result when that
	// binds the parameter: `val small: i8 = id(12)` is id<i8>, where the
	// literal's default would make it id<i64> and then a mismatch
	seeded := map[*types.TypeParam]types.Type{}
	if want != nil && len(typeArgs) == 0 && types.ContainsTypeParam(f.c.hooks.Subst(t.Sig.Ret, m)) {
		for k, v := range m {
			seeded[k] = v
		}
		if !unify(f.c.hooks.Subst(t.Sig.Ret, m), want, seeded) {
			seeded = map[*types.TypeParam]types.Type{}
		}
	}
	argFailed := false  // an argument already reported: inference has nothing to say
	mismatched := false // an argument cannot fit its parameter: the call is reported, nothing to coerce
	for _, i := range order {
		p := t.Sig.Params[i]
		pt := f.c.hooks.Subst(p.Type, m)
		if types.ContainsTypeParam(pt) {
			var x Expr
			if deferredArg(bound[i]) && argFailed {
				// the type parameters it needed were to come from the argument
				// that failed: saying its lambda cannot be typed would report
				// that one mistake a second time
				f.checkArgsLoosely([]ast.Arg{{Value: bound[i]}})
				exprs[i] = bad()
				continue
			}
			if deferredArg(bound[i]) {
				x = f.checkExpr(bound[i], pt)
			} else if hint := f.c.hooks.Subst(p.Type, seeded); len(seeded) > 0 && isLiteral(bound[i]) && !types.ContainsTypeParam(hint) {
				x = f.checkExpr(bound[i], hint)
			} else {
				x = f.checkExpr(bound[i], literalHint(bound[i], pt))
			}
			exprs[i] = x
			if types.IsInvalid(x.Type()) {
				argFailed = true
				continue
			}
			if !unify(pt, x.Type(), m) {
				if n, isN := pt.(*types.Nullable); isN && unify(n.Elem, x.Type(), m) {
					continue
				}
				// D25: a mutable collection is its immutable form too, so
				// `MutableList<string>` binds `List<T>` with T = string
				if views := receiverViews(x.Type()); len(views) > 1 && unify(pt, views[1], m) {
					continue
				}
				f.errorf(bound[i].Span(), "cannot infer type parameters: argument of type '%s' does not match parameter type '%s'", x.Type(), pt)
				mismatched = true
			}
		} else {
			exprs[i] = f.checkExprTo(bound[i], pt)
		}
	}
	if mismatched {
		return bad()
	}
	// Unbound type parameters may still come from the expected return type.
	if want != nil && types.ContainsTypeParam(f.c.hooks.Subst(t.Sig.Ret, m)) {
		unify(f.c.hooks.Subst(t.Sig.Ret, m), want, m)
	}
	var finalArgs []types.Type
	for _, tp := range t.TypeParams {
		bt, ok := m[tp]
		if !ok {
			if !argFailed {
				f.errorf(span, "cannot infer type parameter '%s' of '%s'; supply it explicitly: '%s<...>(...)'", tp.Name, t.Name, t.Name)
			}
			return bad()
		}
		for _, bound := range tp.Bounds {
			if !f.implements(bt, bound) {
				if inner, innerTrait, why := f.unmetThrough(bt, bound); inner != nil {
					// `List<Tag>` is Encodable when Tag is: say so, and fix Tag
					f.c.errorFix(span, f.c.implementFix(inner, innerTrait), "type '%s' does not implement trait '%s' required by parameter '%s' of '%s'; %s%s", bt, bound.Name, tp.Name, t.Name, why, implementHint(inner, innerTrait))
					return bad()
				}
				f.c.errorFix(span, f.c.implementFix(bt, bound), "type '%s' does not implement trait '%s' required by parameter '%s' of '%s'%s", bt, bound.Name, tp.Name, t.Name, implementHint(bt, bound))
				return bad() // instantiating anyway would report the same inside the callee
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
	return &Call{exprBase: exprBase{rt}, Fn: fn, Args: callArgs, Span: span}
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
	if isSendableTrait(trait) {
		// the auto-derived marker (D35): answered from the type's shape, never
		// from an impl
		return sendable(t)
	}
	if isCLayoutTrait(trait) {
		return cLayout(t) // the same, for memory C can read as it is (D69)
	}
	if tt, ok := t.(*types.Tuple); ok && trait.Name == "Comparable" && trait.Module == "std.prelude" {
		return f.c.tupleCompare(tt) != nil // lexicographic order (tuple_order.go)
	}
	if _, ok := t.(*types.Enum); ok && trait.Module == "std.prelude" {
		return f.c.implementsPrelude(t, trait.Name) // Comparable, Display, Equatable, Hashable (enum.go)
	}
	if f.findImpl(t, trait) != nil {
		return true
	}
	if isCombination(trait) {
		// a trait that is only its supers is implemented by implementing them
		for _, s := range trait.Supers {
			if !f.implements(t, s) {
				return false
			}
		}
		return true
	}
	return false
}

// isSendableTrait recognises the prelude's `Sendable` marker trait.
func isSendableTrait(trait *types.Trait) bool {
	return trait != nil && trait.Name == "Sendable" && trait.Module == "std.prelude"
}

// findImpl finds the impl of trait that serves t: its target unifies and
// its own bounds hold (`impl<T: Encodable> Encodable for List<T>` serves
// `List<H>` only when `H` is Encodable).
func (f *fnCtx) findImpl(t types.Type, trait *types.Trait) *Impl {
	for _, impl := range f.c.impls[trait] {
		m := map[*types.TypeParam]types.Type{}
		if unify(impl.Target, t, m) && f.unmetImplBound(impl, m, "", t) == "" {
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
		params[i] = types.Param{Name: fld.Name, Type: fld.Type, HasDefault: true}
	}
	if initT := f.c.methods[t]["$init"]; initT != nil && initT.Sig != nil {
		params = append(params, initT.Sig.Params...) // D73: `Boxed(value: 1)` infers T too
	}
	// a pun names its parameter (D28); constructStruct reports the rest
	named := make([]ast.Arg, len(args))
	for i, a := range args {
		named[i] = a
		if n, isName := a.Value.(*ast.NameExpr); a.Name == nil && isName && len(n.TypeArgs) == 0 && (hasField(t, n.Name) || f.c.hasInitParam(t, n.Name)) {
			named[i].Name = &ast.Ident{Name: n.Name, Pos: n.Pos}
		}
	}
	bound, ok := f.bindArgs(params, named, "struct '"+t.Name+"'", span)
	if !ok {
		return nil
	}
	for i, p := range params {
		if bound[i] == nil || !types.ContainsTypeParam(p.Type) {
			continue
		}
		x := f.checkExpr(bound[i], literalHint(bound[i], p.Type))
		if !types.IsInvalid(x.Type()) {
			unify(p.Type, x.Type(), m)
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
			// a `private` field is settled below (M5 v0.30): left to its
			// default, or given by the call; an unmarked one belongs to the
			// module, and the module alone may construct
			if !fld.Pub && !fld.Private {
				f.errorf(span, "cannot construct '%s' here: field '%s' belongs to module '%s', so the implicit constructor is only callable there (D28); mark the field 'private' to hide it and let other modules construct '%s'", st.Name, fld.Name, st.Module, st.Name)
				f.checkArgsLoosely(args)
				return bad()
			}
		}
	}
	params := make([]types.Param, len(st.Fields))
	for i, fld := range st.Fields {
		// a field the `init` block assigns is not the caller's to give
		params[i] = types.Param{Name: fld.Name, Type: fld.Type, HasDefault: fld.HasDefault || fld.Init}
	}
	// `init(value: T)`: its parameters follow the fields (D73)
	initT := f.c.methods[templateOf(st)]["$init"]
	var initFn *Func
	if initT != nil {
		initFn = f.c.instantiate(initT, substOf(st), nil, span)
		for _, p := range initFn.Sig.Params {
			params = append(params, types.Param{Name: p.Name, Type: p.Type, HasDefault: true})
		}
	}
	written := args
	args, ok := f.punFields(st, args)
	if !ok {
		f.checkArgsLoosely(args)
		return bad()
	}
	if f.c.index != nil {
		punned := make([]bool, len(args))
		for i := range args {
			punned[i] = written[i].Name == nil && args[i].Name != nil
		}
		f.c.refFieldLabels(st, args, punned)
	}
	bound, ok := f.bindArgs(params, args, "struct '"+st.Name+"'", span)
	if !ok {
		f.checkArgsLoosely(args)
		return bad()
	}
	inside := f.insideType(st)
	lit := &StructLit{exprBase{st}, st, make([]Expr, len(st.Fields))}
	for i, fld := range st.Fields {
		var v Expr
		switch {
		case fld.Private && !inside && fld.HasDefault && bound[i] != nil:
			// a private field with a default is the type's own state: from
			// outside it takes its default (M5). One without a default is
			// the initial state the constructor call must supply — nobody
			// else could — so it may be given from anywhere the type is
			// visible, and is private from then on.
			f.errorf(bound[i].Span(), "field '%s' is private to '%s' and has a default; it cannot be set from outside — drop it, or construct through a static function of '%s'", fld.Name, st.Name, st.Name)
			v = bad()
		case fld.Init && bound[i] != nil:
			f.errorf(bound[i].Span(), "field '%s' is assigned by the 'init' block of '%s'; the constructor call does not take it (D28)", fld.Name, st.Name)
			f.checkArgsLoosely([]ast.Arg{{Value: bound[i]}})
			v = &Zero{exprBase{fld.Type}}
		case fld.Init:
			v = &Zero{exprBase{fld.Type}} // bound by `init`, never read before
		case bound[i] != nil:
			v = f.checkExprTo(bound[i], fld.Type)
		case !fld.HasDefault:
			f.errorf(span, "missing field '%s' in constructor of '%s'; add '%s: ...' to the call, or give the field a default", fld.Name, st.Name, fld.Name)
			v = bad()
		default:
			v = f.fieldDefault(st, i)
		}
		lit.Fields[i] = v
	}
	var result Expr = lit
	if initT != nil {
		// `init { }`: the block runs on the freshly built value, then the
		// value is the result (D28); its parameters are the arguments
		// after the fields (D73)
		built := f.newTemp(st)
		args := []Expr{&AddrOf{exprBase{&types.Pointer{Elem: st}}, ref(built)}}
		for j, p := range initFn.Sig.Params {
			k := len(st.Fields) + j
			switch {
			case bound[k] != nil:
				args = append(args, f.checkExprTo(bound[k], p.Type))
			case initT.Decl.Params[j].Default != nil:
				args = append(args, f.defaultArg(initT, j, p.Type, initFn.subst))
			default:
				f.errorf(span, "missing argument '%s' in constructor of '%s': its 'init' takes it (D73)", p.Name, st.Name)
				args = append(args, bad())
			}
		}
		call := &Call{exprBase: exprBase{types.TUnit}, Fn: initFn, Args: args}
		call.Recv, call.RecvRoot, call.RecvSpan, call.RecvType = RecvPlace, built, span, st
		body := &Block{Stmts: []Stmt{&ExprStmt{X: call}}, Value: ref(built), Type: st}
		result = &Let{exprBase{st}, built, lit, &BlockExpr{exprBase{st}, body}}
	}
	if st.Sealed != nil {
		result = &MakeVariant{exprBase{st.Sealed}, st.Sealed, st, result}
	}
	return result
}

// punFields applies D28's construction rule to the arguments: every field
// is named, and a bare argument is allowed only as a pun — a variable
// named like the field, `Hashed(file, size: n)` for `file: file`. Any other
// bare argument is an error carrying the fix that names it after the field
// at its position; `file: file` is a warning with the fix that puns it.
func (f *fnCtx) punFields(st *types.Struct, args []ast.Arg) ([]ast.Arg, bool) {
	out := make([]ast.Arg, len(args))
	ok := true
	for i, a := range args {
		out[i] = a
		if a.Name != nil {
			if n, isName := a.Value.(*ast.NameExpr); isName && n.Name == a.Name.Name && len(n.TypeArgs) == 0 {
				fix := fixReplace("Write '"+n.Name+"' once", source.Span{File: a.Name.Pos.File, Start: a.Name.Pos.Start, End: n.Pos.Start}, "")
				f.warnFix(a.Name.Pos, fix, "'%s: %s' can be written '%s' (D28)", n.Name, n.Name, n.Name)
			}
			continue
		}
		if n, isName := a.Value.(*ast.NameExpr); isName && len(n.TypeArgs) == 0 && (hasField(st, n.Name) || f.c.hasInitParam(st, n.Name)) {
			ident := ast.Ident{Name: n.Name, Pos: n.Pos}
			out[i].Name = &ident
			continue
		}
		ok = false
		if i < len(st.Fields) {
			fld := st.Fields[i].Name
			fix := fixReplace("Name the field", source.Span{File: a.Value.Span().File, Start: a.Value.Span().Start, End: a.Value.Span().Start}, fld+": ")
			f.c.errorFix(a.Value.Span(), fix, "construct '%s' by field name: '%s: %s'; a bare name is accepted only for a variable named like the field (D28)", st.Name, fld, srcText(a.Value))
		} else {
			f.errorf(a.Value.Span(), "construct '%s' by field name (D28)", st.Name)
		}
	}
	return out, ok
}

func hasField(st *types.Struct, name string) bool {
	for _, fld := range st.Fields {
		if fld.Name == name {
			return true
		}
	}
	return false
}

// fieldDefault checks a field's default expression in the struct's own
// module, for a constructor call that did not supply the field. A default
// is a constant: it may not read `this` — a field derived from the others
// is assigned in `init { }`, which sees the whole value (D28 v0.30).
func (f *fnCtx) fieldDefault(st *types.Struct, i int) Expr {
	tmpl := templateOf(st)
	ctx := f.c.structDecl[tmpl]
	if ctx == nil {
		return bad()
	}
	d := ctx.decl.(*ast.StructDecl)
	if sp, ok := mentionsSelf(d.Fields[i].Default); ok {
		f.c.errorf(sp, "a field default cannot read 'this': the value does not exist yet. Derive '%s' in the 'init' block instead — declare it without a default and write 'init { this.%s = ... }' (D28)", d.Fields[i].Name.Name, d.Fields[i].Name.Name)
		return bad()
	}
	env := f.c.envFor(ctx, st)
	g := f.c.newFnCtx(f.fn, ctx.module, ctx.file, env, substOf(st))
	return g.checkExprTo(d.Fields[i].Default, st.Fields[i].Type)
}

// mentionsSelf finds the first `this` in an expression.
func mentionsSelf(e ast.Expr) (source.Span, bool) {
	var at source.Span
	found := false
	walkAST(e, func(n any) bool {
		if s, isSelf := n.(*ast.SelfExpr); isSelf && !found {
			at, found = s.Pos, true
		}
		return !found
	})
	return at, found
}

// ---------------------------------------------------------------------------
// method calls

// methodCall checks `recv.name(args)`. Inside an `init` block a call on
// `this` while owned fields are still unassigned is recorded on the Call
// (InitMissing), for the receiver pass to check against what the method
// reads (D28).
func (f *fnCtx) methodCall(callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	f.refuseHandClose(callee, e)
	var initMissing []int
	if _, isSelf := callee.X.(*ast.SelfExpr); isSelf {
		if in := f.initScope(); in != nil && in == f {
			f.selfAsRecv = true
			for _, name := range in.initMissing() {
				initMissing = append(initMissing, in.initOwned[name])
			}
		}
	}
	x := f.methodCallOn(callee, typeArgs, e, want)
	f.selfAsRecv = false
	if c, ok := x.(*Call); ok && len(initMissing) > 0 {
		c.InitMissing = initMissing
	}
	return x
}

func (f *fnCtx) methodCallOn(callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	f.lookOnly(callee.X) // D87: a protected collection may be a receiver...
	recv := f.checkExpr(callee.X, nil)
	f.pendingLook = f.takeLooked() // ...of a method that does not change it (dispatchMethod)
	if types.IsInvalid(recv.Type()) {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if callee.Safe {
		recv = f.flattenNullable(recv)
		nt, ok := recv.Type().(*types.Nullable)
		if !ok {
			f.errorf(callee.Pos, "'?.' on a non-nullable value of type '%s'; use '.'", recv.Type())
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		if isPlaceExpr(recv) {
			// a nullable variable or field: test the place and call on its
			// payload where it lives, so the method's changes land (D22/D25)
			inner := f.dispatchMethod(&Unwrap{exprBase{nt.Elem}, recv}, callee, typeArgs, e, want)
			if types.IsInvalid(inner.Type()) {
				return inner
			}
			notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, recv}, callee.Pos}
			return f.safeCallBranch(inner, notNull)
		}
		tmp := f.newTemp(nt)
		inner := f.dispatchMethod(&Unwrap{exprBase{nt.Elem}, &VarRef{exprBase{nt}, tmp}}, callee, typeArgs, e, want)
		if types.IsInvalid(inner.Type()) {
			return inner
		}
		notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, &VarRef{exprBase{nt}, tmp}}, callee.Pos}
		body := f.safeCallBranch(inner, notNull)
		return &Let{exprBase{body.Type()}, tmp, recv, body}
	}
	return f.dispatchMethod(recv, callee, typeArgs, e, want)
}

func (f *fnCtx) dispatchMethod(recv Expr, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	name := callee.Name.Name
	look := f.pendingLook
	f.pendingLook = nil
	rt := recv.Type()
	if _, ok := rt.(*types.Pointer); ok && name == "cast" {
		return f.rawPointerCast(recv, name, typeArgs, e) // D86
	}
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
	if lt, isList := rt.(*types.List); isList && name == "filterIs" {
		f.c.refBuiltin(callee.Name.Pos, rt, name)
		return f.listFilterIs(recv, lt, typeArgs, e)
	}
	if bt, ok := rt.(*types.Basic); ok && name == "wrapTo" && types.IsNumeric(bt) {
		f.c.refBuiltin(callee.Name.Pos, rt, name)
		return f.wrapTo(recv, rt, typeArgs, e) // D86
	}
	if look != nil && mutatesCollection(rt, name) {
		f.refuseContentsChange(look, name, callee.Name.Pos)
	}
	if isMutexLock(rt, name) {
		f.lockUse(callee, e)
	}
	if b := f.builtinMethod(recv, rt, name, e); b != nil {
		f.c.refBuiltin(callee.Name.Pos, rt, name)
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
	case *types.ErrorUnion:
		if len(ct.Members) > 0 {
			return f.unionDispatch(recv, ct, callee, typeArgs, e, want)
		}
	case *types.Channel:
		// the built-in operations; the rest (forEach, toList, D110) are the
		// prelude's extend block, below
		if x := f.channelMethod(recv, ct, name, e); x != nil {
			f.c.refBuiltin(callee.Name.Pos, rt, name)
			return x
		}
	case *types.Task:
		if name == "cancel" {
			f.c.refBuiltin(callee.Name.Pos, rt, name)
			if len(e.Args) != 0 {
				f.errorf(e.Pos, "'cancel' takes no arguments")
				f.checkArgsLoosely(e.Args)
				return bad()
			}
			return &Builtin{exprBase{types.TUnit}, "task.cancel", []Expr{recv}, e.Pos}
		}
	case *types.Map:
		if x := f.mapMethod(recv, ct, name, e); x != nil {
			f.c.refBuiltin(callee.Name.Pos, rt, name)
			return x
		}
	case *types.Set:
		if x := f.setMethod(recv, ct, name, e); x != nil {
			f.c.refBuiltin(callee.Name.Pos, rt, name)
			return x
		}
	case *types.Enum:
		if x := f.enumMethodCall(recv, ct, callee, e); x != nil {
			return x
		}
		// anything else comes from a derived impl (encode, D58) or is an error below
	}
	// inherent methods
	if st, ok := rt.(*types.Struct); ok {
		tmpl := templateOf(st)
		if t, ok := f.c.methods[tmpl][name]; ok {
			return f.callMethod(t, substOf(st), typeArgs, recv, viaPointer, callee, e, want)
		}
	}
	// extend blocks: inherent too; the first block whose target matches wins
	// (checkCoherence rejects overlapping ones). A mutable collection also
	// has its immutable form's methods (D25: MutableList<T> is a List<T>),
	// looked up after any block naming the mutable type itself.
	for vi, view := range receiverViews(rt) {
		for _, ext := range f.c.extends {
			t, ok := ext.Methods[name]
			if !ok {
				continue
			}
			if look != nil && vi == 0 && isMutableCollection(rt) {
				f.refuseContentsChange(look, name, callee.Name.Pos) // a block naming the mutable type itself
			}
			m := map[*types.TypeParam]types.Type{}
			if !unify(ext.Target, view, m) {
				continue
			}
			for _, tp := range ext.TypeParams {
				for _, bound := range tp.Bounds {
					if bt, ok := m[tp]; ok && !f.implements(bt, bound) {
						f.errorf(callee.Name.Pos, "'%s' on '%s' requires '%s' to implement '%s' (extend<%s: %s> %s)%s", name, rt, bt, bound.Name, tp.Name, bound.Name, ext.Target, sendableHint(bound))
						f.checkArgsLoosely(e.Args)
						return bad()
					}
				}
			}
			return f.callMethod(t, m, typeArgs, recv, viaPointer, callee, e, want)
		}
	}
	// trait impls (D26: any trait method is callable; ambiguity is an error)
	var found []*FuncTemplate
	var foundSubst []map[*types.TypeParam]types.Type
	unmet := ""
	for trait, impls := range f.c.impls {
		if _, has := trait.Methods[name]; !has {
			continue
		}
		for _, impl := range impls {
			m := map[*types.TypeParam]types.Type{}
			if !unify(impl.Target, rt, m) {
				continue
			}
			if why := f.unmetImplBound(impl, m, name, rt); why != "" {
				unmet = why
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
	// a sealed trait's default method applies to every variant, with or
	// without an `impl` block for that variant
	if len(found) == 0 {
		if st, ok := rt.(*types.Struct); ok && st.Sealed != nil {
			if trait := sealedTemplate(st.Sealed).Trait; trait != nil {
				if dt := f.c.traitDefault(trait, name); dt != nil {
					// the default body sees `this` as the whole sealed type, so
					// `when (self)` over the variants works inside it
					return f.callSealedDefault(dt, st.Sealed, &MakeVariant{exprBase{st.Sealed}, st.Sealed, st, recv}, callee, typeArgs, e, want)
				}
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
	if unmet != "" {
		f.errorf(callee.Name.Pos, "%s", unmet)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	// a field holding a function value: `this.f(x)`
	if st, ok := rt.(*types.Struct); ok {
		for _, fld := range st.Fields {
			if fld.Name == name {
				if ft, isFn := fld.Type.(*types.Func); isFn {
					return f.callValue(&FieldGet{exprBase{fld.Type}, recv, fld.Index, fld.Name}, ft, e.Args, e.Pos)
				}
			}
		}
	}
	if tt, ok := rt.(*types.Tuple); ok && name == "compareTo" && len(e.Args) == 1 {
		if cmp := f.c.tupleCompare(tt); cmp != nil {
			other := f.checkExprTo(e.Args[0].Value, tt)
			return &Call{exprBase: exprBase{cmp.Sig.Ret}, Fn: cmp, Args: []Expr{recvArg(cmp, recv), other}}
		}
	}
	switch tt := rt.(type) {
	case *types.Nullable:
		f.errorf(callee.Pos, "value of type '%s' may be null; use '?.' or check for null first (D5)", tt)
	case *types.Sealed:
		if isResultType(tt) {
			// `try f().m()` applies `try` to the whole chain, so `m` is
			// looked up on the Result; the unwrap has to happen first
			f.errorf(callee.Name.Pos, "no method '%s' on '%s'; 'try' covers the whole chain — write '(try %s).%s(...)' to unwrap first, or match with 'when' (D13)", name, tt, srcText(callee.X), name)
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		f.errorf(callee.Name.Pos, "no method '%s' on sealed trait '%s'; match on its variants with 'when' (D13)", name, tt)
	case *types.Enum:
		f.errorf(callee.Name.Pos, "no method '%s' on enum '%s'; an enum value has 'toString()', 'compareTo(other)', 'encode(to)' and '.value' (D57)", name, tt.Name)
	case *types.Trait:
		f.errorf(callee.Pos, "trait objects (boxed '%s') are not supported yet in the bootstrap compiler; use a generic bound instead (D9)", tt)
	default:
		hint, hit := f.noMethodHint(rt, name)
		f.c.errorFix(callee.Name.Pos, typoFix(callee.Name.Pos, hit), "no method '%s' on type '%s'%s", name, rt, hint)
	}
	f.checkArgsLoosely(e.Args)
	return bad()
}

// isReferenceType reports whether values of t are handles to shared
// storage (D25 collections, channels), so that copying one does not copy
// what it refers to.
func isReferenceType(t types.Type) bool {
	switch t.(type) {
	case *types.List, *types.Map, *types.Set, *types.Channel:
		return true
	}
	return false
}

// receiverViews lists the types an extend target may match for a receiver:
// the type itself, then — for a mutable collection — its immutable form.
func receiverViews(rt types.Type) []types.Type {
	switch t := rt.(type) {
	case *types.List:
		if t.Mutable {
			return []types.Type{rt, &types.List{Elem: t.Elem}}
		}
	case *types.Map:
		if t.Mutable {
			return []types.Type{rt, &types.Map{Key: t.Key, Value: t.Value}}
		}
	case *types.Set:
		if t.Mutable {
			return []types.Type{rt, &types.Set{Elem: t.Elem}}
		}
	}
	return []types.Type{rt}
}

func (f *fnCtx) callMethod(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, typeArgs []types.Type, recv Expr, viaPointer bool, callee *ast.MemberExpr, e *ast.CallExpr, want types.Type) Expr {
	f.checkStoreEscape(recv.Type(), e)
	if t.Decl.Static {
		f.errorf(callee.Name.Pos, "'%s' is a static function; call it on the type: '%s.%s(...)'", t.Name, recv.Type(), t.Name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	f.c.resolveSignature(t)
	f.c.refFunc(callee.Name.Pos, t)
	// inherent methods (struct body or extend block) follow M5; trait impl
	// methods follow the trait's visibility
	inherent := t.Owner != nil || (t.Impl != nil && t.Impl.Trait == nil)
	if inherent && t.Decl.Private {
		owner := t.Owner
		if owner == nil {
			owner, _ = t.Impl.Target.(*types.Struct)
		}
		if owner != nil && !f.insideType(owner) {
			f.errorf(callee.Name.Pos, "method '%s' is private to '%s': only its own methods, impl and extend blocks may call it", t.Name, owner.Name)
		}
	} else if inherent && !t.Pub && t.Module != f.module {
		f.errorf(callee.Name.Pos, "method '%s' is private to module '%s'%s (M5)", t.Name, t.Module.Path, privateHint(t.Module.Path))
	}
	// D22 (v0.30): every method takes a pointer to its receiver's place.
	var recvArg Expr
	if viaPointer {
		recvArg = recv.(*Deref).X
	} else if !isPlaceExpr(recv) || isReferenceType(recv.Type()) {
		// a temporary: the method runs on a fresh copy (iterator chains rely
		// on this). A collection is a reference (D25), so a copy of the
		// handle reaches the same elements, as with the built-in `push`.
		// Whether the method changes the copy — and so loses the change — is
		// known once every body is checked: the receiver pass reports it.
		tmp := f.newTemp(recv.Type())
		recvArg = &AddrOf{exprBase{&types.Pointer{Elem: recv.Type()}}, ref(tmp)}
		call := f.callTemplateRecv(t, ownerSubst, typeArgs, recvArg, e.Args, e.Pos, want)
		if c, ok := call.(*Call); ok {
			c.Recv = RecvTemp
			if isReferenceType(recv.Type()) {
				c.Recv = RecvHandle
			}
			c.RecvSpan = callee.Name.Pos
			c.RecvType = recv.Type()
			c.RecvExpr = callee.X
		}
		return &Let{exprBase{call.Type()}, tmp, recv, call}
	} else {
		// the receiver as checked is a place (a variable, a field, a
		// dereference, a narrowed payload of one): the method works on it
		root := rootVar(recv)
		kind := RecvPlace
		if root != nil && strings.HasPrefix(root.Name, "$") {
			// a temporary the lowering bound (the receiver of `?.`): a copy
			kind, root = RecvTemp, nil
		}
		if root != nil {
			f.invalidateVarPaths(root) // the call may assign its `var` fields
		}
		recvArg = &AddrOf{exprBase{&types.Pointer{Elem: recv.Type()}}, recv}
		call := f.callTemplateRecv(t, ownerSubst, typeArgs, recvArg, e.Args, e.Pos, want)
		if c, ok := call.(*Call); ok {
			c.Recv = kind
			c.RecvRoot = root
			c.RecvSpan = callee.Name.Pos
			c.RecvType = recv.Type()
			c.RecvExpr = callee.X
		}
		return call
	}
	return f.callTemplateRecv(t, ownerSubst, typeArgs, recvArg, e.Args, e.Pos, want)
}

// builtinMethod resolves methods on string, List and MutableList.
func (f *fnCtx) builtinMethod(recv Expr, rt types.Type, name string, e *ast.CallExpr) Expr {
	f.checkStoreEscape(rt, e)
	nargs := func(n int) bool {
		if len(e.Args) != n {
			f.arityError(e.Pos, rt, name, n)
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
			case "charCount":
				// D18: len() is bytes; this counts Unicode scalar values
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{types.TI64}, "string.charCount", []Expr{recv}, e.Pos}
			case "chars":
				// each code point as a one-character string (no char type, D18)
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{&types.List{Elem: types.TString}}, "string.chars", []Expr{recv}, e.Pos}
			case "byteAt":
				if !nargs(1) {
					return bad()
				}
				i := f.checkExprTo(e.Args[0].Value, types.TI64)
				return &Builtin{exprBase{types.TU8}, "string.byteAt", []Expr{recv, i}, e.Pos}
			case "bytes":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{&types.List{Elem: types.TU8}}, "string.bytes", []Expr{recv}, e.Pos}
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
		if conv := f.numericConversion(recv, t, name, e); conv != nil {
			return conv // D86
		}
		// math on numbers: single LLVM instructions/intrinsics, no runtime call
		if types.IsFloat(t) {
			switch name {
			case "sqrt", "abs", "floor", "ceil", "round", "trunc", "log", "log2", "log10", "exp", "sin", "cos", "tan":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "float." + name, []Expr{recv}, e.Pos}
			case "sign":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "float.sign", []Expr{recv}, e.Pos}
			case "pow", "min", "max", "mod", "atan2", "hypot":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "float." + name, []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "clamp":
				if !nargs(2) {
					return bad()
				}
				return &Builtin{exprBase{t}, "float.clamp", []Expr{recv, f.checkExprTo(e.Args[0].Value, t), f.checkExprTo(e.Args[1].Value, t)}, e.Pos}
			case "copySign":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "float.copySign", []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "isSignNegative":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{types.TBool}, "float.isSignNegative", []Expr{recv}, e.Pos}
			case "isNaN", "isFinite", "isInfinite":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{types.TBool}, "float." + name, []Expr{recv}, e.Pos}
			}
		}
		if types.IsInteger(t) {
			switch name {
			case "abs":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int.abs", []Expr{recv}, e.Pos}
			case "sign":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int.sign", []Expr{recv}, e.Pos}
			case "min", "max", "mod":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int." + name, []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "clamp":
				if !nargs(2) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int.clamp", []Expr{recv, f.checkExprTo(e.Args[0].Value, t), f.checkExprTo(e.Args[1].Value, t)}, e.Pos}
			case "pow":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int.pow", []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "wrappingAdd", "wrappingSub", "wrappingMul", "saturatingAdd", "saturatingSub", "saturatingMul":
				// the overflow family: `+` panics on overflow in debug builds (D21);
				// these spell the other policies
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int." + name, []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "checkedAdd", "checkedSub", "checkedMul":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{&types.Nullable{Elem: t}}, "int." + name, []Expr{recv, f.checkExprTo(e.Args[0].Value, t)}, e.Pos}
			case "countOnes", "leadingZeros", "trailingZeros":
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int." + name, []Expr{recv}, e.Pos}
			case "swapBytes", "reverseBits":
				// D104: on the bit pattern, so a signed value cannot overflow
				if !nargs(0) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int." + name, []Expr{recv}, e.Pos}
			case "rotateLeft", "rotateRight":
				if !nargs(1) {
					return bad()
				}
				return &Builtin{exprBase{t}, "int." + name, []Expr{recv, f.checkExprTo(e.Args[0].Value, types.TI64)}, e.Pos}
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
		case "decodeUtf8":
			if !types.Identical(t.Elem, types.TU8) {
				break // only List<u8>
			}
			if !nargs(0) {
				return bad()
			}
			return &Builtin{exprBase{&types.Nullable{Elem: types.TString}}, "list.decodeUtf8", []Expr{recv}, e.Pos}
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
		case "reserve":
			if !t.Mutable {
				f.errorf(e.Pos, "cannot reserve room in an immutable List; use MutableList (D25)")
				f.checkArgsLoosely(e.Args)
				return bad()
			}
			if !nargs(1) {
				return bad()
			}
			n := f.checkExprTo(e.Args[0].Value, types.TI64)
			return &Builtin{exprBase{types.TUnit}, "list.reserve", []Expr{recv, n}, e.Pos}
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
	if ft.C && f.unsafe == 0 {
		f.errorf(span, "calling a C function pointer requires an 'unsafe' block (D44/D69)")
	}
	if len(args) != len(ft.Params) {
		f.errorf(span, "function value takes %d %s, got %d", len(ft.Params), plural(len(ft.Params), "argument"), len(args))
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
	if ft.Effects.Suspends {
		f.refuseHeldSuspension(span, "this call")
	}
	return &CallIndirect{exprBase{rt}, fnv, vals}
}

// funcValue turns a named, non-generic function into a value; it captures
// nothing, so it is sendable (D35).
func (f *fnCtx) funcValue(t *FuncTemplate, span source.Span) Expr {
	f.c.resolveSignature(t)
	if len(t.TypeParams) > 0 {
		f.errorf(span, "generic function '%s' cannot be used as a value; wrap the call in a lambda, whose parameter types pick the instance: '(x) => %s(x)'", t.Name, t.Name)
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
	t.ValueUsed = true
	fn := f.c.instantiate(t, nil, nil, span)
	sig := *fn.Sig
	sig.Sendable = true
	return &FuncRef{exprBase{&sig}, fn}
}

// objectSafe reports whether a trait can be used as a trait object: no
// associated types, no generic methods, no Self in signatures — in the
// trait and in every supertrait, whose methods the object answers to too
// (D58: the table composes the supers').
func (f *fnCtx) objectSafe(trait *types.Trait) (string, bool) {
	slots, clash := objectSlots(trait)
	if clash != "" {
		return clash, false
	}
	for _, tr := range objectOrder(trait) {
		if len(tr.AssocTypes) == 0 {
			continue
		}
		where := ""
		if tr != trait {
			where = " (from '" + tr.Name + "')"
		}
		if tr.ImplicitError && len(tr.AssocTypes) == 1 {
			return "a method is declared with a bare 'throws' (an impl-defined error)" + where + "; declare the error type, e.g. 'throws E', to use the trait as an object", false
		}
		return "it has associated types" + where, false
	}
	for _, s := range slots {
		where := ""
		if s.Owner != trait {
			where = " (from '" + s.Owner.Name + "')"
		}
		key := s.Owner.Name + "." + s.Name
		if f.c.traitStatic[key] {
			return "function '" + s.Name + "' is static" + where, false
		}
		if len(f.c.traitMethodTPs[key]) > 0 {
			return "method '" + s.Name + "' is generic" + where, false
		}
		self := selfParamOf(s.Owner)
		for _, p := range s.Sig.Params {
			if mentions(p.Type, self) {
				return "method '" + s.Name + "' takes Self" + where, false
			}
		}
		if mentions(s.Sig.Ret, self) {
			return "method '" + s.Name + "' returns Self" + where, false
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
	// a combination trait (only its supers) is boxed from their impls, so
	// ask the predicate rather than for an impl of the trait itself
	if !f.implements(t, trait) {
		f.errorf(span, "'%s' does not implement '%s'; add 'implement %s { ... }' to its declaration", t, trait.Name, trait.Name)
		return bad()
	}
	slots, _ := objectSlots(trait)
	box := &Box{exprBase{trait}, x, trait, nil}
	for _, s := range slots {
		// a supertrait's methods live in that super's own impl, written or
		// derived (D58); each is instantiated for the concrete type
		impl := f.findImpl(t, s.Owner)
		if impl == nil {
			f.errorf(span, "'%s' does not implement '%s', required by '%s'", t, s.Owner.Name, trait.Name)
			return bad()
		}
		subst := map[*types.TypeParam]types.Type{}
		unify(impl.Target, t, subst)
		tmpl, ok := impl.Methods[s.Name]
		if !ok {
			tmpl = f.c.traitDefault(s.Owner, s.Name)
			subst[selfParamOf(s.Owner)] = t
		}
		if tmpl == nil {
			f.errorf(span, "'%s' does not implement '%s.%s'", t, s.Owner.Name, s.Name)
			return bad()
		}
		fn := f.c.instantiate(tmpl, subst, nil, span)
		box.Methods = append(box.Methods, fn)
	}
	return box
}

// virtualCall checks a method call on a trait object.
func (f *fnCtx) virtualCall(recv Expr, trait *types.Trait, callee *ast.MemberExpr, e *ast.CallExpr) Expr {
	name := callee.Name.Name
	slots, clash := objectSlots(trait)
	slot, idx := findSlot(slots, name)
	if idx < 0 {
		if clash != "" {
			f.errorf(callee.Name.Pos, "'%s' cannot be a trait object: %s (D9)", trait.Name, clash)
		} else {
			names := make([]string, 0, len(slots))
			for _, s := range slots {
				names = append(names, s.Name)
			}
			hint := ""
			if hit := didYouMean(name, names); hit != "" {
				hint = "; did you mean '" + hit + "'?"
			}
			f.errorf(callee.Name.Pos, "trait '%s' has no method '%s'%s", trait.Name, name, hint)
		}
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	sig := slot.Sig
	f.c.refTraitMethod(callee.Name.Pos, trait, name)
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
	return &CallVirtual{exprBase{rt}, recv, trait, idx, sig, args}
}

// sealedDispatch lowers a method call on a sealed value to a tag switch
// over its variants, each calling that variant's impl (D12/D23).
func (f *fnCtx) sealedDispatch(recv Expr, s *types.Sealed, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	trait := sealedTemplate(s).Trait
	name := callee.Name.Name
	// A default body that no variant overrides runs directly on the sealed
	// value; `this` inside it is the sealed type.
	if dt := f.c.traitDefault(trait, name); dt != nil {
		overridden := false
		for _, v := range s.Variants {
			if impl := f.findImpl(v, trait); impl != nil {
				if _, has := impl.Methods[name]; has {
					overridden = true
				}
			}
		}
		if !overridden {
			return f.callSealedDefault(dt, s, recv, callee, typeArgs, e, want)
		}
	}
	tmp := f.newTemp(s)
	m := &Match{Subject: tmp, Init: recv, Exhaustive: true, Span: e.Pos}
	// the arms work on the value itself when it is a place, so that a
	// method assigning the variant's `var` fields reaches it (D22)
	var subject Expr = ref(tmp)
	if isPlaceExpr(recv) {
		subject = recv
	}
	var rt types.Type
	for _, v := range s.Variants {
		if f.findImpl(v, trait) == nil && f.c.traitDefault(trait, name) == nil {
			f.errorf(e.Pos, "variant '%s.%s' does not implement '%s'", s.Name, v.Name, name)
			return bad()
		}
		payload := &VariantCast{exprBase{v}, subject, v}
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

// unionDispatch lowers a method call on an error union to a tag switch
// over its members (D4): every member implements Error, so `e.message()`
// needs no `when`. Any method every member provides dispatches this way.
func (f *fnCtx) unionDispatch(recv Expr, u *types.ErrorUnion, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	name := callee.Name.Name
	tmp := f.newTemp(u)
	m := &Match{Subject: tmp, Init: recv, Exhaustive: true, Span: e.Pos}
	var subject Expr = ref(tmp)
	if isPlaceExpr(recv) {
		subject = recv
	}
	var rt types.Type
	for _, mem := range u.Members {
		payload := &UnionCast{exprBase{mem}, subject, mem}
		call := f.dispatchMethod(payload, callee, typeArgs, e, want)
		if types.IsInvalid(call.Type()) {
			return bad()
		}
		if rt == nil {
			rt = call.Type()
		} else if !types.Identical(rt, call.Type()) {
			f.errorf(e.Pos, "members of '%s' disagree on the type of '%s'", u, name)
			return bad()
		}
		body := &Block{Value: call, Type: rt}
		if types.IsUnit(rt) {
			body = &Block{Stmts: []Stmt{&ExprStmt{X: call}}, Type: types.TUnit}
		}
		m.Arms = append(m.Arms, &MatchArm{Test: &UnionTest{exprBase{types.TBool}, ref(tmp), mem}, Body: body})
	}
	m.T = rt
	return m
}

// deferredArg reports an argument whose type is best checked after the
// other arguments have bound the callee's type parameters: a lambda (its
// parameter types come from the signature) or an empty collection literal
// (its element type has no other source).
func deferredArg(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.LambdaExpr:
		return true
	case *ast.ListLit:
		return len(e.Elems) == 0
	case *ast.MapLit:
		return len(e.Entries) == 0
	}
	return false
}

// callSealedDefault invokes a sealed trait's default method with `Self`
// bound to the sealed type itself.
func (f *fnCtx) callSealedDefault(dt *FuncTemplate, s *types.Sealed, recv Expr, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	trait := sealedTemplate(s).Trait
	m := map[*types.TypeParam]types.Type{selfParamOf(trait): s}
	for i, tp := range trait.TypeParams {
		if i < len(s.TypeArgs) {
			m[tp] = s.TypeArgs[i]
		}
	}
	return f.callMethod(dt, m, typeArgs, recv, false, callee, e, want)
}

// literalHint is the expected type to check an inference argument with:
// a collection literal gets the (still generic) parameter type, from which
// it takes only the mutability (D25/#8); anything else is checked bare.
func literalHint(arg ast.Expr, pt types.Type) types.Type {
	switch arg.(type) {
	case *ast.ListLit, *ast.MapLit:
		return pt
	}
	return nil
}

// safeCallBranch wraps a `?.` call: `if (cond) inner else null`, with the
// result type lifted to nullable unless the call already returns one or
// nothing.
func (f *fnCtx) safeCallBranch(inner Expr, cond Expr) Expr {
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
	return &If{exprBase{rt}, cond, thenBlock, elseBlock}
}

// panicCall is `panic(message)` (D20): it never returns; it unwinds to the
// task scope.
func (f *fnCtx) panicCall(e *ast.CallExpr) Expr {
	if len(e.Args) != 1 {
		f.errorf(e.Pos, "'panic' takes one argument: the message")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if n, ok := e.Fun.(*ast.NameExpr); ok && f.c.index != nil {
		f.c.index.Refs = append(f.c.index.Refs, Ref{Span: n.Pos, Kind: "fun", Name: "panic", Detail: "fun panic(message: string): Never  (built in)", Doc: panicDoc})
	}
	msg := f.checkExprTo(e.Args[0].Value, types.TString)
	return &Builtin{exprBase{types.TNever}, "panic", []Expr{msg}, e.Pos}
}

// isCLayoutTrait recognises the prelude's `CLayout` marker (D69).
func isCLayoutTrait(trait *types.Trait) bool {
	return trait != nil && trait.Name == "CLayout" && trait.Module == "std.prelude"
}

// cLayout: values C reads as they lie in memory — numbers, bool, raw
// pointers, and `extern struct`s — so a list of them can be lent to C
// element for element (`withRaw`, D69).
func cLayout(t types.Type) bool {
	if types.IsNumeric(t) || types.IsBool(t) {
		return true
	}
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
		if p, ok := t.(*types.Pointer); ok {
			return p.Raw // a nullable raw pointer is a C pointer that may be NULL
		}
		return false
	}
	switch t := t.(type) {
	case *types.Pointer:
		return t.Raw
	case *types.Struct:
		return t.Extern
	case *types.Func:
		return t.C
	}
	return false
}

// cFunctionPointer is `&name` where name is an `extern "C" fun` (D69): the
// address C calls, typed `extern fun(...)`. Nil when e is anything else,
// which leaves `&` its ordinary meaning.
func (f *fnCtx) cFunctionPointer(e *ast.UnaryExpr) Expr {
	n, ok := e.X.(*ast.NameExpr)
	if !ok {
		return nil
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymFunc || sym.Func == nil || sym.Func.Decl == nil || !sym.Func.Decl.ExportC {
		return nil
	}
	t := sym.Func
	f.c.refSym(n.Pos, sym)
	f.c.resolveSignature(t)
	if len(t.TypeParams) > 0 || t.Sig == nil {
		return bad() // reported at the declaration
	}
	fn := f.c.instantiate(t, nil, nil, n.Pos)
	sig := *fn.Sig
	sig.C = true
	sig.Sendable = false
	return &FuncRef{exprBase{&sig}, fn}
}

// initParams are the parameters `init(...)` adds to a struct's constructor
// (D73); nil when its init takes none or it has no init.
func (c *Checker) initParams(st *types.Struct) []ast.Param {
	if d := c.initDecl[templateOf(st)]; d != nil {
		return d.Params
	}
	return nil
}

func (c *Checker) hasInitParam(st *types.Struct, name string) bool {
	for _, p := range c.initParams(st) {
		if p.Name.Name == name {
			return true
		}
	}
	return false
}

// unmetThrough explains why t lacks trait when an impl would serve it but
// for one of its own bounds — `implement<T: Encodable> Encodable for
// List<T>` and a `List<Tag>` — by naming the innermost type that fails:
// ('Tag', Encodable, "'List<T>' implements 'Encodable' when 'T' does, and
// 'Tag' does not"). nil when no impl comes that close.
func (f *fnCtx) unmetThrough(t types.Type, trait *types.Trait) (types.Type, *types.Trait, string) {
	for _, impl := range f.c.impls[trait] {
		m := map[*types.TypeParam]types.Type{}
		if !unify(impl.Target, t, m) {
			continue
		}
		for _, tp := range impl.TypeParams {
			bt, ok := m[tp]
			if !ok || types.ContainsTypeParam(bt) {
				continue
			}
			for _, b := range tp.Bounds {
				if f.implements(bt, b) {
					continue
				}
				why := fmt.Sprintf("'%s' implements '%s' when '%s' does, and '%s' does not", impl.Target, trait.Name, tp.Name, bt)
				if deeper, dt, _ := f.unmetThrough(bt, b); deeper != nil {
					return deeper, dt, why
				}
				return bt, b, why
			}
		}
	}
	return nil, nil, ""
}
