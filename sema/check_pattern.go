package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// whenExpr lowers `when` to a Match (D13).
func (f *fnCtx) whenExpr(e *ast.WhenExpr, want types.Type) Expr {
	asValue := want != nil && !types.IsUnit(want)
	inferValue := want == nil
	m := &Match{Span: e.Pos}

	var subj Expr
	var subjPlace place
	var subjOK bool
	var subjType types.Type
	if e.Subject != nil {
		subj = f.checkExpr(e.Subject, nil)
		subjType = subj.Type()
		if types.IsInvalid(subjType) {
			return bad()
		}
		subjPlace, subjOK = f.placeOf(e.Subject)
		// D39: a pointer scrutinee is dereferenced without ceremony.
		if p, ok := subjType.(*types.Pointer); ok && !p.Raw {
			subj = &Deref{exprBase{p.Elem}, subj}
			subjType = p.Elem
		}
		if e.Bind != nil {
			// `when (val r = subject)`: the subject is a named local of the arms,
			// and smart casts apply to it
			f.pushScope()
			defer f.popScope()
			v := f.newVar(e.Bind.Name, subjType, false, e.Bind.Pos)
			f.declareLocal(e.Bind.Name, v, e.Bind.Pos)
			m.Subject = v
			subjPlace, subjOK = pv(v), true
		} else {
			m.Subject = f.newTemp(subjType)
		}
		m.Init = subj
	}

	cov := newCoverage()
	hasElse := false
	var resultType types.Type
	var nullArms []*NullConst // `=> null` arms, typed from the others at the end
	var valueArms []*MatchArm
	saved := f.saveNarrow()
	for _, arm := range e.Arms {
		f.pushScope()
		f.restoreNarrow(saved)
		ha := &MatchArm{}
		switch {
		case arm.Else:
			// not recorded in cov: the sealed-else lint below asks whether the
			// other arms cover everything on their own
			hasElse = true
		case e.Subject == nil:
			cond := f.checkExprTo(arm.Cond, types.TBool)
			ha.Test = cond
			tf, _ := f.condFacts(arm.Cond, cond)
			f.applyFacts(tf)
		default:
			var tests []Expr
			subjRef := &VarRef{exprBase{subjType}, m.Subject}
			for i, pat := range arm.Patterns {
				test, binds, _ := f.compilePattern(pat, subjRef, subjType, pat.Span())
				if len(binds) > 0 && len(arm.Patterns) > 1 {
					f.errorf(pat.Span(), "a pattern that binds names cannot be combined with other patterns in one arm; give it an arm of its own")
				}
				if test != nil {
					tests = append(tests, test)
				} else if len(arm.Patterns) > 1 {
					f.errorf(pat.Span(), "pattern matches everything; the other alternatives in this arm are redundant")
				}
				if i == 0 {
					ha.Binds = binds
				}
				if arm.Guard == nil {
					f.cover(cov, pat, subjType)
				}
				// smart cast of the subject variable inside the arm
				if subjOK && len(arm.Patterns) == 1 {
					if tp, ok := pat.(*ast.TypePat); ok {
						if target := f.resolvePatternType(subjType, tp.Type); target != nil && !types.Identical(target, subjType) {
							f.narrow[subjPlace] = target
						}
					}
				}
			}
			if len(tests) > 0 {
				ha.Test = tests[0]
				for _, t := range tests[1:] {
					ha.Test = &Binary{exprBase{types.TBool}, OpOr, ha.Test, t, arm.Pos}
				}
			}
			// bindings are in scope for guard and body
			for _, b := range ha.Binds {
				if vd, ok := b.(*VarDecl); ok {
					f.declareLocal(vd.Var.Name, vd.Var, vd.Var.Span)
				}
			}
			for _, pat := range arm.Patterns {
				f.declareUnbound(pat)
			}
			if arm.Guard != nil {
				ha.Guard = f.checkExprTo(arm.Guard, types.TBool)
			}
		}
		// body
		var body *Block
		switch {
		case asValue:
			x := f.checkExprTo(arm.Body, want)
			body = f.valueBlock(x)
		case inferValue:
			if _, isNull := arm.Body.(*ast.NullLit); isNull {
				// typed from the other arms once they are known, as in
				// ifExpr: `is Circle(r) => r  else => null` is an `f64?`
				nc := &NullConst{exprBase{&types.Nullable{Elem: types.TNever}}}
				nullArms = append(nullArms, nc)
				body = &Block{Value: nc, Type: nc.T}
				break
			}
			x := f.checkExpr(arm.Body, nil)
			body = f.valueBlock(x)
		default:
			x := f.checkExpr(arm.Body, types.TUnit)
			body = &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: types.TUnit}
			if types.IsNever(x.Type()) {
				body.Type = types.TNever
			}
		}
		ha.Body = body
		if body.Value != nil {
			valueArms = append(valueArms, ha)
			if _, isNull := body.Value.(*NullConst); resultType == nil && !(isNull && inferValue) {
				resultType = body.Type
			}
		}
		m.Arms = append(m.Arms, ha)
		f.popScope()
	}
	f.restoreNarrow(saved)

	// exhaustiveness
	exhaustive := hasElse || f.isExhaustive(cov, subjType, e.Subject == nil)
	m.Exhaustive = exhaustive
	if !exhaustive {
		if asValue || inferValue || f.requiresExhaustive(subjType) {
			f.errorf(e.Pos, "'when' is not exhaustive: %s (D13; add the missing arms or 'else')", f.missingArms(cov, subjType))
		}
	} else if hasElse && subjType != nil {
		if st, sealed := subjType.(*types.Sealed); sealed {
			// D13 lint, scoped (2026-09-18): an `else` that stands for one
			// missing variant is an enumeration that a new variant would
			// silently fall into — warn. One-variant extraction
			// (`is JStr(v) => v  else => null`) is the honest spelling and
			// stays silent. An `else` with every variant covered is dead: the
			// fix removes it; otherwise the author has to write the missing
			// arm, which no fix can invent.
			covered := 0
			for _, v := range st.Variants {
				if cov.variants[v] {
					covered++
				}
			}
			switch {
			case f.isExhaustive(cov, subjType, false):
				var fix *source.Fix
				for _, arm := range e.Arms {
					if arm.Else {
						fix = fixDeleteLine("Remove the unreachable 'else' arm", arm.Pos)
					}
				}
				f.warnFix(e.Pos, fix, "'else' is unreachable: every variant of '%s' has an arm (D13 lint)", st.Name)
			case len(st.Variants) > 1 && covered == len(st.Variants)-1:
				f.warnf(e.Pos, "'else' stands for the one remaining variant, %s, and would silently take any variant added to '%s' later; name it instead (D13 lint)", f.missingArms(cov, subjType), st.Name)
			}
		}
		if en, isEnum := subjType.(*types.Enum); isEnum {
			// the same lint over an enum's members (D57)
			covered := 0
			for _, m := range en.Members {
				if cov.enumMembers[m] {
					covered++
				}
			}
			switch {
			case f.isExhaustive(cov, subjType, false):
				var fix *source.Fix
				for _, arm := range e.Arms {
					if arm.Else {
						fix = fixDeleteLine("Remove the unreachable 'else' arm", arm.Pos)
					}
				}
				f.warnFix(e.Pos, fix, "'else' is unreachable: every member of '%s' has an arm (D13 lint)", en.Name)
			case len(en.Members) > 1 && covered == len(en.Members)-1:
				f.warnf(e.Pos, "'else' stands for the one remaining member, %s, and would silently take any member added to '%s' later; name it instead (D13 lint)", f.missingArms(cov, subjType), en.Name)
			}
		}
	}

	// result type
	if asValue {
		m.T = want
	} else if inferValue && len(nullArms) > 0 && nullableOf(resultType) == nil {
		// every value arm is `null`, or the others give nothing to go on
		f.errorf(e.Pos, "cannot infer the type of 'null' in this 'when'; annotate the binding, e.g. 'val x: T? = when ...'")
		m.T = types.TInvalid
	} else if inferValue && resultType != nil {
		// unify arms; a `null` arm makes the result nullable
		rt := resultType
		if len(nullArms) > 0 {
			rt = nullableOf(rt)
			for _, nc := range nullArms {
				nc.T = rt
			}
			for _, ha := range valueArms {
				if _, ok := ha.Body.Value.(*NullConst); ok {
					ha.Body.Type = rt
				}
			}
		}
		incompatible := false
		for _, ha := range valueArms {
			if !types.Identical(ha.Body.Type, rt) {
				if f.assignableTo(ha.Body.Type, rt) {
					ha.Body.Value = f.coerce(ha.Body.Value, rt, e.Pos)
					ha.Body.Type = rt
				} else if f.assignableTo(rt, ha.Body.Type) {
					rt = ha.Body.Type
				} else {
					f.errorf(e.Pos, "'when' arms have incompatible types '%s' and '%s'; make every arm give the same type, or annotate the result ('val x: T = when ...')", rt, ha.Body.Type)
					incompatible = true
				}
			}
		}
		if incompatible {
			rt = types.TInvalid // reported once; coercing each arm would report it again
		} else {
			for _, ha := range valueArms {
				if !types.Identical(ha.Body.Type, rt) {
					ha.Body.Value = f.coerce(ha.Body.Value, rt, e.Pos)
					ha.Body.Type = rt
				}
			}
		}
		m.T = rt
		if !exhaustive {
			m.T = types.TUnit
		}
	} else {
		m.T = types.TUnit
		allNever := len(m.Arms) > 0 && exhaustive
		for _, ha := range m.Arms {
			if ha.Body.Value != nil {
				ha.Body.Stmts = append(ha.Body.Stmts, &ExprStmt{X: ha.Body.Value})
				ha.Body.Value = nil
				ha.Body.Type = types.TUnit
			}
			if !types.IsNever(ha.Body.Type) {
				allNever = false
			}
		}
		if allNever {
			m.T = types.TNever
		}
	}
	if (asValue || inferValue) && m.T != nil && !types.IsUnit(m.T) {
		for _, ha := range m.Arms {
			if ha.Body.Value == nil && !types.IsNever(ha.Body.Type) {
				f.errorf(e.Pos, "every arm of a 'when' used as a value must produce a value")
			}
		}
	}
	return m
}

func (f *fnCtx) valueBlock(x Expr) *Block {
	if be, ok := x.(*BlockExpr); ok {
		return be.Block
	}
	if types.IsNever(x.Type()) {
		return &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: types.TNever}
	}
	return &Block{Value: x, Type: x.Type()}
}

func (f *fnCtx) requiresExhaustive(t types.Type) bool {
	switch tt := t.(type) {
	case *types.Sealed, *types.Nullable, *types.Enum:
		return true
	case *types.Basic:
		return tt.Kind == types.Bool
	}
	return false
}

// ---------------------------------------------------------------------------
// pattern compilation

// compilePattern turns a pattern over subj (of type t) into a boolean test
// (nil when irrefutable) and the bindings it introduces.
func (f *fnCtx) compilePattern(pat ast.Pattern, subj Expr, t types.Type, span source.Span) (test Expr, binds []Stmt, irrefutable bool) {
	and := func(a, b Expr) Expr {
		if a == nil {
			return b
		}
		if b == nil {
			return a
		}
		return &Binary{exprBase{types.TBool}, OpAnd, a, b, span}
	}
	switch p := pat.(type) {
	case *ast.WildcardPat:
		return nil, nil, true
	case *ast.BindPat:
		v := f.newVar(p.Name.Name, t, false, p.Name.Pos)
		v.checkUse = true
		return nil, []Stmt{&VarDecl{Var: v, Init: subj}}, true
	case *ast.LiteralPat:
		if tp := f.variantNamePattern(p); tp != nil {
			return f.compileTypePattern(tp, subj, t, span)
		}
		if _, isNull := p.Value.(*ast.NullLit); isNull {
			if _, ok := t.(*types.Nullable); !ok {
				f.errorf(p.Span(), "'null' pattern on a non-nullable subject of type '%s': it never matches; remove the arm", t)
				return &BoolConst{exprBase{types.TBool}, false}, nil, false
			}
			return &IsNull{exprBase{types.TBool}, subj}, nil, false
		}
		if nt, ok := t.(*types.Nullable); ok {
			inner, b, _ := f.compilePattern(pat, &Unwrap{exprBase{nt.Elem}, subj}, nt.Elem, span)
			notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, subj}, span}
			return and(notNull, inner), b, false
		}
		// refused before the constant is checked against the subject's
		// type: a constant can never be a value of it, and saying so as a
		// type mismatch too would report one mistake twice
		if !f.comparable(t) {
			f.errorf(p.Span(), "values of type '%s' cannot be matched against a constant; match its fields, or test it with a guard", t)
			return &BoolConst{exprBase{types.TBool}, false}, nil, false
		}
		v := f.checkExprTo(p.Value, t)
		if types.IsInvalid(v.Type()) {
			return &BoolConst{exprBase{types.TBool}, false}, nil, false
		}
		return &Binary{exprBase{types.TBool}, OpEq, subj, v, span}, nil, false
	case *ast.RangePat:
		if nt, ok := t.(*types.Nullable); ok {
			inner, b, _ := f.compilePattern(pat, &Unwrap{exprBase{nt.Elem}, subj}, nt.Elem, span)
			notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, subj}, span}
			return and(notNull, inner), b, false
		}
		if !types.IsInteger(t) {
			f.errorf(p.Pos, "range patterns need an integer subject, found '%s'", t)
			return &BoolConst{exprBase{types.TBool}, false}, nil, false
		}
		lo := f.checkExprTo(p.Range.Lo, t)
		hi := f.checkExprTo(p.Range.Hi, t)
		hiOp := OpLe
		if !p.Range.Inclusive {
			hiOp = OpLt
		}
		test := and(&Binary{exprBase{types.TBool}, OpGe, subj, lo, span}, &Binary{exprBase{types.TBool}, hiOp, subj, hi, span})
		return test, nil, false
	case *ast.TuplePat:
		tt, ok := t.(*types.Tuple)
		if !ok {
			f.errorf(p.Pos, "tuple pattern on a non-tuple subject of type '%s'", t)
			return &BoolConst{exprBase{types.TBool}, false}, nil, false
		}
		if len(tt.Elems) != len(p.Elems) {
			f.errorf(p.Pos, "tuple has %d elements but the pattern has %d", len(tt.Elems), len(p.Elems))
			return &BoolConst{exprBase{types.TBool}, false}, nil, false
		}
		irref := true
		for i, el := range p.Elems {
			et, b, ir := f.compilePattern(el, &TupleGet{exprBase{tt.Elems[i]}, subj, i}, tt.Elems[i], span)
			test = and(test, et)
			binds = append(binds, b...)
			irref = irref && ir
		}
		return test, binds, irref
	case *ast.TypePat:
		return f.compileTypePattern(p, subj, t, span)
	case *ast.ListPat:
		return f.compileListPattern(p, subj, t, span)
	case *ast.RestPat:
		f.errorf(p.Pos, "'..' only stands inside a list pattern: '[first, ..rest]'")
		return &BoolConst{exprBase{types.TBool}, false}, nil, false
	}
	f.errorf(pat.Span(), "unsupported pattern")
	return &BoolConst{exprBase{types.TBool}, false}, nil, false
}

// listShape is a list pattern's length requirement: exactly n elements, or
// at least n when it has a `..`. rest is the RestPat's index in Elems, -1
// without one; ok is false (and reported) for a second `..`.
func (f *fnCtx) listShape(p *ast.ListPat, report bool) (n int, rest int, ok bool) {
	rest, ok = -1, true
	for i, el := range p.Elems {
		if _, isRest := el.(*ast.RestPat); isRest {
			if rest >= 0 {
				if report && ok {
					f.errorf(el.Span(), "a list pattern has at most one '..'")
				}
				ok = false
				continue
			}
			rest = i
			continue
		}
		n++
	}
	return n, rest, ok
}

// compileListPattern matches a list by length and element (D62): `[a, b]`
// tests `len == 2`, `[a, ..rest, z]` tests `len >= 2`, then each element
// pattern against its element — counted from the front before the `..` and
// from the back after it. The reads cannot fail, since the length test
// guards them (`&&` short-circuits, and the bindings run only on a match).
// `..rest` binds a new List of what lies between.
func (f *fnCtx) compileListPattern(p *ast.ListPat, subj Expr, t types.Type, span source.Span) (Expr, []Stmt, bool) {
	fail := func() (Expr, []Stmt, bool) { return &BoolConst{exprBase{types.TBool}, false}, nil, false }
	and := func(a, b Expr) Expr {
		if a == nil {
			return b
		}
		if b == nil {
			return a
		}
		return &Binary{exprBase{types.TBool}, OpAnd, a, b, span}
	}
	if ptr, ok := t.(*types.Pointer); ok && !ptr.Raw {
		subj = &Deref{exprBase{ptr.Elem}, subj}
		t = ptr.Elem
	}
	if nt, ok := t.(*types.Nullable); ok {
		// `[a, b]` on a `List<T>?` also means "not null"
		inner, b, _ := f.compileListPattern(p, &Unwrap{exprBase{nt.Elem}, subj}, nt.Elem, span)
		notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, subj}, span}
		return and(notNull, inner), b, false
	}
	lt, ok := t.(*types.List)
	if !ok {
		f.errorf(p.Pos, "list pattern on a subject of type '%s', which is not a List", t)
		return fail()
	}
	// a second `..` is reported and the arm never matches, but its names are
	// still declared so the body does not cascade into "unknown name"
	n, rest, shapeOK := f.listShape(p, true)
	length := func() Expr { return &Builtin{exprBase{types.TI64}, "list.len", []Expr{subj}, span} }
	var test Expr
	if rest < 0 {
		test = &Binary{exprBase{types.TBool}, OpEq, length(), i64c(uint64(n)), span}
	} else if n > 0 {
		test = &Binary{exprBase{types.TBool}, OpGe, length(), i64c(uint64(n)), span}
	}
	irref := test == nil
	var binds []Stmt
	after := 0 // elements after the `..`
	if rest >= 0 {
		after = len(p.Elems) - rest - 1
	}
	for i, el := range p.Elems {
		if _, isRest := el.(*ast.RestPat); isRest && i != rest {
			continue
		}
		if i == rest {
			rp := el.(*ast.RestPat)
			if rp.Name == nil {
				continue
			}
			rt := &types.List{Elem: lt.Elem}
			to := &Binary{exprBase{types.TI64}, OpWrapSub, length(), i64c(uint64(after)), span}
			v := f.newVar(rp.Name.Name, rt, false, rp.Name.Pos)
			v.checkUse = true
			binds = append(binds, &VarDecl{Var: v, Init: &Builtin{exprBase{rt}, "list.slice", []Expr{subj, i64c(uint64(rest)), to}, span}})
			continue
		}
		var idx Expr = i64c(uint64(i))
		if rest >= 0 && i > rest {
			idx = &Binary{exprBase{types.TI64}, OpWrapSub, length(), i64c(uint64(len(p.Elems) - i)), span}
		}
		get := &Builtin{exprBase{lt.Elem}, "list.get", []Expr{subj, idx}, span}
		et, b, ir := f.compilePattern(el, get, lt.Elem, span)
		test = and(test, et)
		binds = append(binds, b...)
		irref = irref && ir
	}
	if !shapeOK {
		return &BoolConst{exprBase{types.TBool}, false}, binds, false
	}
	return test, binds, irref
}

// resolvePatternType resolves the type named in `is T` relative to the
// subject: bare variant names resolve within the subject's sealed type and
// Some/None/Ok/Err within nullable and Result subjects.
func (f *fnCtx) resolvePatternType(subjType types.Type, t ast.Type) types.Type {
	if p, ok := subjType.(*types.Pointer); ok && !p.Raw {
		subjType = p.Elem
	}
	nt, ok := t.(*ast.NamedType)
	if ok && len(nt.Path) == 1 && len(nt.Args) == 0 {
		name := nt.Path[0].Name
		base := subjType
		if n, isN := subjType.(*types.Nullable); isN {
			switch name {
			case "Some":
				return n.Elem
			case "None":
				return nil
			}
			base = n.Elem
		} else if sym := f.lookup(name); (name == "Some" || name == "None") && (sym == nil || sym.Kind == SymVariantCtor) {
			return nil // on a non-nullable subject: compileTypePattern says so
		}
		if s, isSealed := base.(*types.Sealed); isSealed {
			if v := s.VariantByName(name); v != nil {
				return v
			}
		}
	}
	rt := f.resolve(t)
	if types.IsInvalid(rt) {
		return nil
	}
	// A variant template named directly (e.g. imported `Circle`) resolves
	// to the instance belonging to the subject's sealed type.
	if st, ok := rt.(*types.Struct); ok && st.Sealed != nil {
		base := subjType
		if n, isN := subjType.(*types.Nullable); isN {
			base = n.Elem
		}
		if s, isSealed := base.(*types.Sealed); isSealed && sealedTemplate(s) == sealedTemplate(st.Sealed) {
			return s.Variants[st.Tag]
		}
	}
	return rt
}

func (f *fnCtx) compileTypePattern(p *ast.TypePat, subj Expr, t types.Type, span source.Span) (Expr, []Stmt, bool) {
	if ptr, ok := t.(*types.Pointer); ok && !ptr.Raw {
		subj = &Deref{exprBase{ptr.Elem}, subj}
		t = ptr.Elem
	}
	fail := func() (Expr, []Stmt, bool) { return &BoolConst{exprBase{types.TBool}, false}, nil, false }
	and := func(a, b Expr) Expr {
		if a == nil {
			return b
		}
		if b == nil {
			return a
		}
		return &Binary{exprBase{types.TBool}, OpAnd, a, b, span}
	}
	name := ""
	if nt, ok := p.Type.(*ast.NamedType); ok && len(nt.Path) == 1 {
		name = nt.Path[0].Name
	}
	// Nullable subjects
	if nt, ok := t.(*types.Nullable); ok {
		notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, subj}, span}
		switch name {
		case "None":
			if p.HasArg {
				f.errorf(p.Pos, "'None' has no fields")
			}
			return &IsNull{exprBase{types.TBool}, subj}, nil, false
		case "Some":
			inner := &Unwrap{exprBase{nt.Elem}, subj}
			if !p.HasArg {
				return notNull, nil, false
			}
			if len(p.Fields) != 1 {
				f.errorf(p.Pos, "'Some' has exactly one field")
				return fail()
			}
			sub := p.Fields[0].Pat
			if sub == nil {
				sub = &ast.BindPat{Name: p.Fields[0].Name}
			}
			it, b, _ := f.compilePattern(sub, inner, nt.Elem, span)
			return and(notNull, it), b, false
		}
		target := f.resolvePatternType(t, p.Type)
		if target == nil {
			return fail()
		}
		if types.Identical(target, nt.Elem) && !p.HasArg {
			return notNull, nil, false
		}
		// `is Variant(...)` through the nullable
		inner := &Unwrap{exprBase{nt.Elem}, subj}
		it, b, _ := f.compileTypePattern(p, inner, nt.Elem, span)
		return and(notNull, it), b, false
	}
	if sym := f.lookup(name); (name == "None" || name == "Some") && (sym == nil || sym.Kind == SymVariantCtor) {
		// before resolving: they are not types, and "'Some' is not a type"
		// would not say what is wrong
		f.errorf(p.Pos, "'%s' pattern on a non-nullable subject of type '%s': the value is always there; match it directly", name, t)
		return fail()
	}
	target := f.resolvePatternType(t, p.Type)
	if target == nil {
		return fail()
	}
	if u, ok := t.(*types.ErrorUnion); ok {
		if types.UnionIndex(u, target) < 0 {
			f.errorf(p.Pos, "'%s' is not a member of the error union '%s'", target, u)
			return fail()
		}
		test := Expr(&UnionTest{exprBase{types.TBool}, subj, target})
		payload := &UnionCast{exprBase{target}, subj, target}
		if v, isStruct := target.(*types.Struct); isStruct {
			it, b, _ := f.compileFields(p, payload, v, span)
			return and(test, it), b, false
		}
		if p.HasArg {
			f.errorf(p.Pos, "'%s' has no fields to destructure", target)
		}
		return test, nil, false
	}
	switch st := t.(type) {
	case *types.Sealed:
		v, ok := target.(*types.Struct)
		if !ok || v.Sealed != st {
			f.errorf(p.Pos, "'%s' is not a variant of '%s'", target, st)
			return fail()
		}
		test := Expr(&VariantTest{exprBase{types.TBool}, subj, v})
		payload := &VariantCast{exprBase{v}, subj, v}
		it, b, _ := f.compileFields(p, payload, v, span)
		return and(test, it), b, false
	case *types.Struct:
		v, ok := target.(*types.Struct)
		if !ok || v != st {
			f.errorf(p.Pos, "subject has type '%s', which can never be '%s', so this never matches; remove it", t, target)
			return fail()
		}
		return f.compileFields(p, subj, v, span)
	case *types.Basic:
		if types.Identical(target, t) {
			if p.HasArg {
				f.errorf(p.Pos, "'%s' has no fields to destructure", t)
			}
			return nil, nil, true
		}
		f.errorf(p.Pos, "subject has type '%s', which can never be '%s', so this never matches; remove it", t, target)
		return fail()
	}
	if types.Identical(target, t) {
		return nil, nil, true
	}
	f.errorf(p.Pos, "subject has type '%s', which can never be '%s', so this never matches; remove it", t, target)
	return fail()
}

// compileFields handles destructuring `V(a, b: pat)` against a struct
// value (D12: binds by field name; a lone positional pattern binds the
// single field).
func (f *fnCtx) compileFields(p *ast.TypePat, value Expr, st *types.Struct, span source.Span) (Expr, []Stmt, bool) {
	if !p.HasArg {
		return nil, nil, true
	}
	var test Expr
	var binds []Stmt
	irref := true
	f.c.resolveStruct(templateOf(st))
	for i, fp := range p.Fields {
		var fld *types.Field
		if fp.Name.Name != "" {
			for _, cand := range st.Fields {
				if cand.Name == fp.Name.Name {
					fld = cand
				}
			}
			if fld == nil && fp.Pat == nil && len(st.Fields) == 1 && len(p.Fields) == 1 {
				fld = st.Fields[0] // `Some(x)`-style positional shorthand
			}
			if fld == nil {
				f.errorf(fp.Name.Pos, "'%s' has no field '%s' (patterns bind by field name, D12/D13)", st.Name, fp.Name.Name)
				continue
			}
		} else {
			if i >= len(st.Fields) {
				f.errorf(fp.Pat.Span(), "'%s' has only %d field(s)", st.Name, len(st.Fields))
				continue
			}
			fld = st.Fields[i]
		}
		if fld.Private && !f.insideType(st) {
			f.errorf(fp.Name.Pos, "field '%s' is private to '%s' and cannot be matched here", fld.Name, st.Name)
			continue
		}
		if !fld.Pub && st.Module != f.module.prefix() {
			f.errorf(p.Pos, "field '%s' of '%s' is private (M5)", fld.Name, st.Name)
		}
		sub := fp.Pat
		if sub == nil {
			sub = &ast.BindPat{Name: fp.Name}
		}
		get := &FieldGet{exprBase{fld.Type}, value, fld.Index, fld.Name}
		it, b, ir := f.compilePattern(sub, get, fld.Type, span)
		if it != nil {
			if test == nil {
				test = it
			} else {
				test = &Binary{exprBase{types.TBool}, OpAnd, test, it, span}
			}
		}
		binds = append(binds, b...)
		irref = irref && ir
	}
	return test, binds, irref
}

// ---------------------------------------------------------------------------
// exhaustiveness (variant-set coverage; guarded arms never count)

type coverage struct {
	all         bool
	null        bool
	some        bool
	someInner   *coverage // coverage of the payload when Some(pat) is refutable
	tru, fals   bool
	variants    map[*types.Struct]bool
	members     map[string]bool            // error-union members (D45)
	enumMembers map[*types.EnumMember]bool // enum members named by value patterns (D57)
	listLens    map[int]bool               // list lengths covered exactly by `[a, b]` (D62)
	listMin     int                        // lengths >= listMin covered by `[a, ..]`; -1 when none
}

func newCoverage() *coverage {
	return &coverage{variants: map[*types.Struct]bool{}, members: map[string]bool{}, enumMembers: map[*types.EnumMember]bool{}, listLens: map[int]bool{}, listMin: -1}
}

func (f *fnCtx) cover(cov *coverage, pat ast.Pattern, t types.Type) {
	switch p := pat.(type) {
	case *ast.WildcardPat, *ast.BindPat:
		cov.all = true
	case *ast.LiteralPat:
		if tp := f.variantNamePattern(p); tp != nil {
			f.cover(cov, tp, t)
			return
		}
		if en, ok := t.(*types.Enum); ok {
			// `E.Member =>`: compilePattern has already checked that the
			// value is a member of this enum, so the name is enough
			if m := enumMemberPattern(en, p); m != nil {
				cov.enumMembers[m] = true
			}
			return
		}
		switch v := p.Value.(type) {
		case *ast.NullLit:
			cov.null = true
		case *ast.BoolLit:
			if v.Value {
				cov.tru = true
			} else {
				cov.fals = true
			}
		}
	case *ast.TuplePat:
		all := true
		for _, el := range p.Elems {
			if !f.irrefutablePat(el) {
				all = false
			}
		}
		if all {
			cov.all = true
		}
	case *ast.ListPat:
		if nt, ok := t.(*types.Nullable); ok {
			if cov.someInner == nil {
				cov.someInner = newCoverage()
			}
			f.cover(cov.someInner, p, nt.Elem)
			return
		}
		// only a pattern whose elements all match anything covers a length
		for _, el := range p.Elems {
			if !f.irrefutablePat(el) {
				return
			}
		}
		n, rest, ok := f.listShape(p, false)
		switch {
		case !ok:
		case rest < 0:
			cov.listLens[n] = true
		case n == 0:
			cov.all = true
		case cov.listMin < 0 || n < cov.listMin:
			cov.listMin = n
		}
	case *ast.TypePat:
		name := ""
		if nt, ok := p.Type.(*ast.NamedType); ok && len(nt.Path) == 1 {
			name = nt.Path[0].Name
		}
		if nt, ok := t.(*types.Nullable); ok {
			switch name {
			case "None":
				cov.null = true
				return
			case "Some":
				if !p.HasArg || (len(p.Fields) == 1 && f.irrefutableField(p.Fields[0])) {
					cov.some = true
				} else if len(p.Fields) == 1 && p.Fields[0].Pat != nil {
					if cov.someInner == nil {
						cov.someInner = newCoverage()
					}
					f.cover(cov.someInner, p.Fields[0].Pat, nt.Elem)
				}
				return
			}
			target := f.resolvePatternType(t, p.Type)
			if target != nil && types.Identical(target, nt.Elem) {
				allFields := true
				for _, fp := range p.Fields {
					if !f.irrefutableField(fp) {
						allFields = false
					}
				}
				if allFields {
					cov.some = true
				}
			}
			return
		}
		target := f.resolvePatternType(t, p.Type)
		if target == nil {
			return
		}
		allFields := true
		for _, fp := range p.Fields {
			if !f.irrefutableField(fp) {
				allFields = false
			}
		}
		if !allFields {
			return
		}
		if types.Identical(target, t) {
			cov.all = true
			return
		}
		if u, ok := t.(*types.ErrorUnion); ok && types.UnionIndex(u, target) >= 0 {
			cov.members[types.Key(target)] = true
			return
		}
		if v, ok := target.(*types.Struct); ok && v.Sealed != nil {
			cov.variants[v] = true
		}
	}
}

func (f *fnCtx) irrefutableField(fp ast.FieldPat) bool {
	return fp.Pat == nil || f.irrefutablePat(fp.Pat)
}

func (f *fnCtx) irrefutablePat(p ast.Pattern) bool {
	switch p := p.(type) {
	case *ast.WildcardPat, *ast.BindPat, *ast.RestPat:
		return true
	case *ast.TuplePat:
		for _, el := range p.Elems {
			if !f.irrefutablePat(el) {
				return false
			}
		}
		return true
	case *ast.ListPat:
		// only `[..]` / `[..rest]` matches every list
		n, rest, ok := f.listShape(p, false)
		return ok && rest >= 0 && n == 0
	}
	return false
}

func (f *fnCtx) isExhaustive(cov *coverage, t types.Type, subjectless bool) bool {
	if cov.all {
		return true
	}
	if subjectless {
		return false
	}
	switch tt := t.(type) {
	case *types.Nullable:
		some := cov.some || (cov.someInner != nil && f.isExhaustive(cov.someInner, tt.Elem, false))
		return cov.null && some
	case *types.ErrorUnion:
		for _, m := range tt.Members {
			if !cov.members[types.Key(m)] {
				return false
			}
		}
		return true
	case *types.Sealed:
		for _, v := range tt.Variants {
			if !cov.variants[v] {
				return false
			}
		}
		return len(tt.Variants) > 0
	case *types.Enum:
		for _, m := range tt.Members {
			if !cov.enumMembers[m] {
				return false
			}
		}
		return len(tt.Members) > 0
	case *types.List:
		// every length below the shortest `[a, ..]` needs its own arm
		if cov.listMin < 0 {
			return false
		}
		for k := 0; k < cov.listMin; k++ {
			if !cov.listLens[k] {
				return false
			}
		}
		return true
	case *types.Basic:
		if tt.Kind == types.Bool {
			return cov.tru && cov.fals
		}
	}
	return false
}

// listArm spells a list pattern of n wildcards, open-ended with `..`:
// `[]`, `[_, _]`, `[_, ..]`.
func listArm(n int, open bool) string {
	parts := make([]string, 0, n+1)
	for i := 0; i < n; i++ {
		parts = append(parts, "_")
	}
	if open {
		parts = append(parts, "..")
	}
	return "'[" + strings.Join(parts, ", ") + "]'"
}

func (f *fnCtx) missingArms(cov *coverage, t types.Type) string {
	switch tt := t.(type) {
	case *types.Nullable:
		var missing []string
		if !cov.null {
			missing = append(missing, "'null'")
		}
		if !cov.some {
			missing = append(missing, "the non-null case ('is "+tt.Elem.String()+"' or 'Some(x)')")
		}
		return "missing " + strings.Join(missing, " and ")
	case *types.Sealed:
		var missing []string
		for _, v := range tt.Variants {
			if !cov.variants[v] {
				missing = append(missing, "'is "+tt.Name+"."+v.Name+"'")
			}
		}
		return "missing " + strings.Join(missing, ", ")
	case *types.ErrorUnion:
		var missing []string
		for _, m := range tt.Members {
			if !cov.members[types.Key(m)] {
				missing = append(missing, "'is "+m.String()+"'")
			}
		}
		return "missing " + strings.Join(missing, ", ")
	case *types.Enum:
		var missing []string
		for _, m := range tt.Members {
			if !cov.enumMembers[m] {
				missing = append(missing, "'"+tt.Name+"."+m.Name+"'")
			}
		}
		return "missing " + strings.Join(missing, ", ")
	case *types.List:
		limit := cov.listMin
		if limit < 0 {
			for k := range cov.listLens {
				if k+1 > limit {
					limit = k + 1
				}
			}
			if limit < 0 {
				limit = 0
			}
		}
		var missing []string
		for k := 0; k < limit; k++ {
			if !cov.listLens[k] {
				missing = append(missing, listArm(k, false))
			}
		}
		if cov.listMin < 0 {
			missing = append(missing, listArm(limit, true))
		}
		return "missing " + strings.Join(missing, ", ")
	case *types.Basic:
		if tt.Kind == types.Bool {
			if !cov.tru {
				return "missing 'true'"
			}
			return "missing 'false'"
		}
	}
	if t == nil {
		return "a condition chain needs an 'else' arm"
	}
	return "subjects of type '" + t.String() + "' need an 'else' arm"
}

// variantNamePattern recognises a bare name pattern that denotes a
// field-less variant or a prelude constructor (`None`, `Point`,
// `Shape.Point`) and rewrites it as a type pattern.
func (f *fnCtx) variantNamePattern(p *ast.LiteralPat) *ast.TypePat {
	var path []ast.Ident
	switch v := p.Value.(type) {
	case *ast.NameExpr:
		sym := f.lookup(v.Name)
		if sym == nil {
			return nil
		}
		switch sym.Kind {
		case SymVariantCtor:
		case SymType:
			st, ok := sym.Type.(*types.Struct)
			if !ok || st.Sealed == nil {
				return nil
			}
		default:
			return nil
		}
		path = []ast.Ident{{Name: v.Name, Pos: v.Pos}}
	case *ast.MemberExpr:
		n, ok := v.X.(*ast.NameExpr)
		if !ok {
			return nil
		}
		sym := f.lookup(n.Name)
		if sym == nil || sym.Kind != SymType {
			return nil
		}
		if _, ok := sym.Type.(*types.Sealed); !ok {
			return nil
		}
		path = []ast.Ident{{Name: n.Name, Pos: n.Pos}, v.Name}
	default:
		return nil
	}
	return &ast.TypePat{Type: &ast.NamedType{Path: path, Pos: p.Value.Span()}, Pos: p.Value.Span()}
}

// enumMemberPattern is the member a value pattern over an enum subject
// names: `E.Member` or `mod.E.Member`. nil for any other expression (a
// constant of the base type, which covers nothing).
func enumMemberPattern(en *types.Enum, p *ast.LiteralPat) *types.EnumMember {
	m, ok := p.Value.(*ast.MemberExpr)
	if !ok {
		return nil
	}
	switch x := m.X.(type) {
	case *ast.NameExpr:
	case *ast.TypeExpr: // synthesized code (D58)
	case *ast.MemberExpr:
		if _, ok := x.X.(*ast.NameExpr); !ok {
			return nil
		}
	default:
		return nil
	}
	return en.MemberByName(m.Name.Name)
}

// declareUnbound gives each name a pattern would bind, but that a rejected
// pattern did not, a binding of no type: the arm's guard and body then
// report their own mistakes rather than an "unknown name" for every one.
func (f *fnCtx) declareUnbound(pat ast.Pattern) {
	declare := f.declareSilently
	var walk func(p ast.Pattern)
	walk = func(p ast.Pattern) {
		switch p := p.(type) {
		case *ast.BindPat:
			declare(p.Name)
		case *ast.TypePat:
			for _, fp := range p.Fields {
				if fp.Pat == nil {
					declare(fp.Name)
				} else {
					walk(fp.Pat)
				}
			}
		case *ast.TuplePat:
			for _, e := range p.Elems {
				walk(e)
			}
		case *ast.ListPat:
			for _, e := range p.Elems {
				walk(e)
			}
		case *ast.RestPat:
			if p.Name != nil {
				declare(*p.Name)
			}
		}
	}
	walk(pat)
}

// declareSilently declares name with no type unless it is declared already;
// what failed to bind it was reported, so neither a use of it nor its
// never being used is an error of its own.
func (f *fnCtx) declareSilently(name ast.Ident) {
	if name.Name == "" || name.Name == "_" || f.scope.LookupLocal(name.Name) != nil {
		return
	}
	v := f.newVar(name.Name, types.TInvalid, false, name.Pos)
	markUsed(v)
	f.declareLocal(name.Name, v, name.Pos)
}

// declareBindingSilently is declareSilently for every name of a
// destructuring binding whose value could not be destructured.
func (f *fnCtx) declareBindingSilently(b ast.Binding) {
	if b.Name != nil {
		f.declareSilently(*b.Name)
	}
	for _, sub := range b.Tuple {
		f.declareBindingSilently(sub)
	}
}
