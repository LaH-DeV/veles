package sema

import (
	"math"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

func bad() Expr { return &UnitConst{exprBase{types.TInvalid}} }

func isResultType(t types.Type) bool {
	s, ok := t.(*types.Sealed)
	return ok && s.Template != nil && s.Template.Name == "Result" && s.Template.Module == "<prelude>"
}

// payloadField returns the single field of a builtin `Ok`, `Err` or `Some`
// variant. A smart cast onto one of these reads through to the payload:
// `if (r is Ok)` makes `r` a `T`, exactly as `x != null` makes a `T?` a
// `T` — the wrapper is handled, never held (D4/D5).
func payloadField(t types.Type) *types.Field {
	st, ok := t.(*types.Struct)
	if !ok || st.Sealed == nil || len(st.Fields) != 1 {
		return nil
	}
	s := st.Sealed
	if s.Template != nil {
		s = s.Template
	}
	if s.Module != "<prelude>" || (s.Name != "Result" && s.Name != "Option") {
		return nil
	}
	return st.Fields[0]
}

// checkExprTo checks e and converts it to want, reporting a mismatch.
func (f *fnCtx) checkExprTo(e ast.Expr, want types.Type) Expr {
	x := f.checkExpr(e, want)
	return f.coerce(x, want, e.Span())
}

// coerce converts x to want using the implicit conversions the spec allows:
// Never to anything, T to T? (D5), a variant to its sealed type (D12), a
// member to an error union (D45).
func (f *fnCtx) coerce(x Expr, want types.Type, span source.Span) Expr {
	have := x.Type()
	if want == nil || types.IsInvalid(want) || types.IsInvalid(have) {
		return x
	}
	if types.Identical(have, want) || types.IsNever(have) {
		return x
	}
	if conv := f.convertAt(x, want, span); conv != nil {
		return conv
	}
	if types.IsUnit(want) {
		return x // statement position: any value may be discarded
	}
	if n, ok := have.(*types.Nullable); ok && types.Identical(n.Elem, want) {
		f.errorf(span, "type mismatch: expected '%s', found '%s'; the value may be null — supply a fallback with '?:' or check for null first (D5)", want, have)
		return x
	}
	if hf, ok := have.(*types.Func); ok {
		if wf, ok := want.(*types.Func); ok && len(hf.Params) == len(wf.Params) && (hf.Effects != wf.Effects) {
			// same shape, different effects: a function value keeps its own
			// calling convention, a lambda written in place adapts (D40)
			f.errorf(span, "type mismatch: expected '%s', found '%s'; a function value does not take on 'suspends' or 'throws' — pass a lambda that calls it, '(x) => f(x)'", want, have)
			return x
		}
	}
	if types.IsString(want) && (types.IsNumeric(have) || types.IsBool(have)) {
		f.errorf(span, "type mismatch: expected '%s', found '%s'; build strings with interpolation: \"...${expr}...\"", want, have)
		return x
	}
	f.errorf(span, "type mismatch: expected '%s', found '%s'", want, have)
	return x
}

func (f *fnCtx) convert(x Expr, want types.Type) Expr {
	return f.convertAt(x, want, source.Span{})
}

func (f *fnCtx) convertAt(x Expr, want types.Type, span source.Span) Expr {
	have := x.Type()
	switch w := want.(type) {
	case *types.Nullable:
		if types.Identical(have, w.Elem) {
			return &SomeWrap{exprBase{want}, x}
		}
		if inner := f.convertAt(x, w.Elem, span); inner != nil {
			return &SomeWrap{exprBase{want}, inner}
		}
	case *types.Sealed:
		if st, ok := have.(*types.Struct); ok && st.Sealed == w {
			return &MakeVariant{exprBase{want}, w, st, x}
		}
	case *types.List, *types.Map, *types.Set:
		// D25: a mutable collection is usable as its immutable form (the
		// same handle; the callee just cannot change it through this type)
		if views := receiverViews(have); len(views) > 1 && types.Identical(views[1], want) {
			return &Cast{exprBase{want}, x}
		}
	case *types.Trait:
		// implicit boxing into a trait object (D9)
		if _, isTrait := have.(*types.Trait); !isTrait && f.findImpl(have, w) != nil {
			return f.boxValue(x, w, span)
		}
	case *types.Pointer:
		// Inside unsafe a GC pointer may be handed to C as a raw pointer: the
		// collector is non-moving (§5), so the address is stable.
		if hp, ok := have.(*types.Pointer); ok && w.Raw && !hp.Raw && types.Identical(hp.Elem, w.Elem) && f.unsafe > 0 {
			return &Cast{exprBase{want}, x}
		}
	case *types.Func:
		// a sendable function is also an ordinary one (D35): same value,
		// viewed through the plain type
		if h, ok := have.(*types.Func); ok && h.Sendable && !w.Sendable {
			plain := *h
			plain.Sendable = false
			if types.Identical(&plain, w) {
				return &Cast{exprBase{want}, x}
			}
		}
	case *types.ErrorUnion:
		if types.UnionIndex(w, have) >= 0 {
			return &ErrorConvert{exprBase{want}, x, have}
		}
		if hu, ok := have.(*types.ErrorUnion); ok {
			for _, m := range hu.Members {
				if types.UnionIndex(w, m) < 0 {
					return nil
				}
			}
			return &ErrorConvert{exprBase{want}, x, have}
		}
	}
	return nil
}

// assignableTo reports whether a value of type have could be coerced to want.
func (f *fnCtx) assignableTo(have, want types.Type) bool {
	if types.Identical(have, want) || types.IsNever(have) || types.IsInvalid(have) || types.IsInvalid(want) {
		return true
	}
	switch w := want.(type) {
	case *types.Nullable:
		return f.assignableTo(have, w.Elem)
	case *types.Sealed:
		st, ok := have.(*types.Struct)
		return ok && st.Sealed == w
	case *types.List, *types.Map, *types.Set:
		views := receiverViews(have)
		return len(views) > 1 && types.Identical(views[1], want)
	case *types.Trait:
		if _, isTrait := have.(*types.Trait); isTrait {
			return false
		}
		return f.findImpl(have, w) != nil
	case *types.ErrorUnion:
		for _, m := range types.UnionMembers(have) {
			if types.UnionIndex(w, m) < 0 {
				return false
			}
		}
		return true
	case *types.Func:
		// a sendable function is also an ordinary one (D35); the reverse
		// would let a closure over mutable state cross a task boundary
		if h, ok := have.(*types.Func); ok && h.Sendable && !w.Sendable {
			plain := *h
			plain.Sendable = false
			return types.Identical(&plain, w)
		}
	}
	return false
}

// numericHint strips nullable layers from an expected type to find the
// primitive a literal should take.
func numericHint(want types.Type) types.Type {
	for {
		n, ok := want.(*types.Nullable)
		if !ok {
			return want
		}
		want = n.Elem
	}
}

// checkExpr checks an expression. want is a hint (may be nil); the caller
// performs the final coercion.
func (f *fnCtx) checkExpr(e ast.Expr, want types.Type) Expr {
	if p, ok := f.boundPlace[e]; ok {
		return p // the receiver of a `?.` write (check_safe.go)
	}
	switch e := e.(type) {
	case *ast.IntLit:
		return f.intLit(e, want, false)
	case *ast.FloatLit:
		t := numericHint(want)
		if !types.IsFloat(t) {
			t = types.TF64
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(e.Text, "_", ""), 64)
		if err != nil {
			f.errorf(e.Pos, "invalid float literal")
		}
		return &FloatConst{exprBase{t}, v}
	case *ast.BoolLit:
		return &BoolConst{exprBase{types.TBool}, e.Value}
	case *ast.StringLit:
		return f.stringLit(e)
	case *ast.CharLit:
		// `'"'` is a byte (D18 addendum, v0.24): the u8 of one ASCII character,
		// for code that walks a string with `byteAt`. Not a character type —
		// a multi-byte character has no single byte to be.
		if len(e.Value) != 1 {
			f.errorf(e.Pos, "a byte literal holds one ASCII character; '%s' is %d bytes of UTF-8 — use a one-character string (D18)", e.Value, len(e.Value))
			return bad()
		}
		t := numericHint(want)
		if !types.IsInteger(t) {
			t = types.TU8
		}
		return &IntConst{exprBase{t}, uint64(e.Value[0]), false}
	case *ast.NullLit:
		if n, ok := want.(*types.Nullable); ok {
			return &NullConst{exprBase{n}}
		}
		if want != nil && !types.IsInvalid(want) && !types.IsUnit(want) {
			f.errorf(e.Pos, "'null' is not a value of type '%s'; the type must be nullable ('%s?')", want, want)
			return bad()
		}
		f.errorf(e.Pos, "cannot infer the type of 'null' here; annotate the binding, e.g. 'val x: T? = null'")
		return bad()
	case *ast.SelfExpr:
		self := f.selfRef()
		if self == nil {
			if f.fn != nil && f.fn.tmpl != nil && f.fn.tmpl.Decl.Static {
				f.errorf(e.Pos, "'self' is not available in a static function; it has no receiver (D23)")
				return bad()
			}
			f.errorf(e.Pos, "'self' outside of a method")
			return bad()
		}
		// the receiver pointer, read as the value it points at (D22), with
		// the smart casts on `self` in force
		x := &Deref{exprBase{self.Type.(*types.Pointer).Elem}, &VarRef{exprBase{self.Type}, self}}
		return f.narrowPlace(x, pv(self))
	case *ast.NameExpr:
		return f.nameExpr(e, want)
	case *ast.MemberExpr:
		return f.memberExpr(e, want)
	case *ast.IndexExpr:
		return f.indexExpr(e)
	case *ast.CallExpr:
		return f.callExpr(e, want)
	case *ast.UnaryExpr:
		return f.unaryExpr(e, want)
	case *ast.BinaryExpr:
		return f.binaryExpr(e, want)
	case *ast.OrFailExpr:
		return f.orFailExpr(e)
	case *ast.WithExpr:
		return f.withExpr(e, want)
	case *ast.ElvisExpr:
		return f.elvisExpr(e, want)
	case *ast.RangeExpr:
		return f.rangeExpr(e, want)
	case *ast.TupleExpr:
		if len(e.Elems) == 0 {
			return &UnitConst{exprBase{types.TUnit}}
		}
		var wants []types.Type
		if tt, ok := want.(*types.Tuple); ok && len(tt.Elems) == len(e.Elems) {
			wants = tt.Elems
		}
		lit := &TupleLit{}
		tt := &types.Tuple{}
		for i, el := range e.Elems {
			var w types.Type
			if wants != nil {
				w = wants[i]
			}
			var x Expr
			if w != nil {
				x = f.checkExprTo(el, w)
			} else {
				x = f.checkExpr(el, nil)
			}
			lit.Elems = append(lit.Elems, x)
			tt.Elems = append(tt.Elems, x.Type())
		}
		lit.T = tt
		return lit
	case *ast.ListLit:
		return f.listLit(e, want)
	case *ast.MapLit:
		return f.mapLit(e, want)
	case *ast.IfExpr:
		return f.ifExpr(e, want)
	case *ast.WhenExpr:
		return f.whenExpr(e, want)
	case *ast.BlockExpr:
		b := f.checkBlock(e.Block, want, true)
		return &BlockExpr{exprBase{b.Type}, b}
	case *ast.TryExpr:
		return f.tryExpr(e)
	case *ast.IsExpr:
		return f.isExpr(e)
	case *ast.CastExpr:
		return f.castExpr(e)
	case *ast.UnsafeExpr:
		f.unsafe++
		b := f.checkBlock(e.Body, want, true)
		f.unsafe--
		return &BlockExpr{exprBase{b.Type}, b}
	case *ast.LambdaExpr:
		return f.lambdaExpr(e, want)
	case *ast.AwaitExpr:
		return f.awaitExpr(e)
	case *ast.GatherExpr:
		return f.gatherExpr(e)
	case *ast.RaceExpr:
		return f.raceExpr(e, want)
	case *ast.ControlExpr:
		stmts, _ := f.checkStmt(e.Stmt)
		return &BlockExpr{exprBase{types.TNever}, &Block{Stmts: stmts, Type: types.TNever}}
	case *ast.BadExpr:
		return bad()
	}
	f.errorf(e.Span(), "unsupported expression")
	return bad()
}

func (f *fnCtx) intLit(e *ast.IntLit, want types.Type, neg bool) Expr {
	t := numericHint(want)
	if !types.IsNumeric(t) {
		t = types.TI64
	}
	text := strings.ReplaceAll(e.Text, "_", "")
	v, err := strconv.ParseUint(text, 0, 64)
	if err != nil {
		f.errorf(e.Pos, "integer literal is too large")
		return &IntConst{exprBase{t}, 0, false}
	}
	if types.IsFloat(t) {
		fv := float64(v)
		if neg {
			fv = -fv
		}
		return &FloatConst{exprBase{t}, fv}
	}
	bits := types.BitSize(t)
	var max uint64
	if types.IsSigned(t) {
		max = uint64(1)<<(bits-1) - 1
		if neg {
			max++
		}
	} else {
		if neg && v != 0 {
			f.errorf(e.Pos, "negative literal for unsigned type '%s'", t)
		}
		if bits == 64 {
			max = math.MaxUint64
		} else {
			max = uint64(1)<<bits - 1
		}
	}
	if v > max {
		f.errorf(e.Pos, "literal %s does not fit in '%s'", e.Text, t)
	}
	return &IntConst{exprBase{t}, v, neg}
}

func (f *fnCtx) stringLit(e *ast.StringLit) Expr {
	if len(e.Parts) == 1 && e.Parts[0].Expr == nil {
		return &StringConst{exprBase{types.TString}, e.Parts[0].Text}
	}
	if len(e.Parts) == 0 {
		return &StringConst{exprBase{types.TString}, ""}
	}
	cat := &StringConcat{exprBase{types.TString}, nil}
	for _, p := range e.Parts {
		if p.Expr == nil {
			cat.Parts = append(cat.Parts, &StringConst{exprBase{types.TString}, p.Text})
			continue
		}
		x := f.checkExpr(p.Expr, nil)
		cat.Parts = append(cat.Parts, f.toString(x, p.Expr.Span()))
	}
	return cat
}

// toString converts any showable value to a string (interpolation).
func (f *fnCtx) toString(x Expr, span source.Span) Expr {
	t := x.Type()
	if types.IsString(t) {
		return x
	}
	if types.IsInvalid(t) {
		return x
	}
	if types.IsUnit(t) || types.IsNever(t) {
		f.errorf(span, "cannot interpolate a value of type '%s'", t)
		return x
	}
	return &ToString{exprBase{types.TString}, x}
}

// ---------------------------------------------------------------------------
// names and members

func (f *fnCtx) lookup(name string) *Symbol {
	return f.scope.Lookup(name)
}

func (f *fnCtx) nameExpr(e *ast.NameExpr, want types.Type) Expr {
	sym := f.lookup(e.Name)
	if sym == nil {
		f.errorf(e.Pos, "unknown name '%s'%s", e.Name, f.c.suggestUnknownName(f.module, e.Name))
		return bad()
	}
	if sym.Kind == SymLocal {
		v := f.localVar(sym.Var)
		markUsed(v)
		x := f.narrowedRef(v)
		f.c.refVarAs(e.Pos, v, x.Type())
		return x
	}
	f.c.refSym(e.Pos, sym)
	switch sym.Kind {
	case SymGlobal:
		v := f.globalVar(sym.Global)
		return &VarRef{exprBase{v.Type}, v}
	case SymFunc:
		return f.funcValue(sym.Func, e.Pos)
	case SymType:
		if st, ok := sym.Type.(*types.Struct); ok && len(st.Fields) == 0 && st.Sealed != nil {
			// a field-less variant used as a value: `None`-style
			return f.variantValue(st, want, e.Pos)
		}
		f.errorf(e.Pos, "'%s' is a type, not a value", e.Name)
		return bad()
	case SymModule:
		f.errorf(e.Pos, "'%s' is a module, not a value", e.Name)
		return bad()
	case SymVariantCtor:
		return f.variantCtor(e.Name, nil, want, e.Pos)
	}
	f.errorf(e.Pos, "'%s' cannot be used as a value", e.Name)
	return bad()
}

// narrowedRef reads a variable applying its current smart cast (D5).
func (f *fnCtx) narrowedRef(v *Var) Expr {
	return f.narrowPlace(&VarRef{exprBase{v.Type}, v}, pv(v))
}

// narrowExpr converts a value of a wider type to a narrower one already
// established by a test: T? to T, Sealed to Variant, nested.
func narrowExpr(x Expr, to types.Type) Expr {
	for !types.Identical(x.Type(), to) {
		switch t := x.Type().(type) {
		case *types.Pointer:
			if t.Raw {
				return x
			}
			x = &Deref{exprBase{t.Elem}, x}
		case *types.Nullable:
			x = &Unwrap{exprBase{t.Elem}, x}
		case *types.Sealed:
			if st, ok := to.(*types.Struct); ok && st.Sealed == t {
				return &VariantCast{exprBase{st}, x, st}
			}
			// A fact established through a payload (`is Err` then
			// `is ParseError`) reads through the variant that can reach it.
			var via *types.Struct
			for _, v := range t.Variants {
				if fld := payloadField(v); fld != nil && narrowReaches(fld.Type, to) {
					via = v
					break
				}
			}
			if via == nil {
				return x
			}
			fld := via.Fields[0]
			x = &FieldGet{exprBase{fld.Type}, &VariantCast{exprBase{via}, x, via}, fld.Index, fld.Name}
		case *types.ErrorUnion:
			if types.UnionIndex(t, to) < 0 {
				return x
			}
			return &UnionCast{exprBase{to}, x, to}
		default:
			return x
		}
	}
	return x
}

// narrowReaches reports whether narrowExpr can take a value of type from
// to the type to.
func narrowReaches(from, to types.Type) bool {
	for !types.Identical(from, to) {
		switch t := from.(type) {
		case *types.Pointer:
			if t.Raw {
				return false
			}
			from = t.Elem
		case *types.Nullable:
			from = t.Elem
		case *types.Sealed:
			if st, ok := to.(*types.Struct); ok && st.Sealed == t {
				return true
			}
			for _, v := range t.Variants {
				if fld := payloadField(v); fld != nil && narrowReaches(fld.Type, to) {
					return true
				}
			}
			return false
		case *types.ErrorUnion:
			return types.UnionIndex(t, to) >= 0
		default:
			return false
		}
	}
	return true
}

func (f *fnCtx) memberExpr(e *ast.MemberExpr, want types.Type) Expr {
	// module member or sealed variant?
	if n, ok := e.X.(*ast.NameExpr); ok {
		if sym := f.lookup(n.Name); sym != nil {
			switch sym.Kind {
			case SymModule:
				member := sym.Mod.Scope.LookupLocal(e.Name.Name)
				if member == nil {
					f.errorf(e.Name.Pos, "module '%s' has no declaration '%s'", n.Name, e.Name.Name)
					return bad()
				}
				if !member.Pub {
					f.errorf(e.Name.Pos, "'%s' is private to module '%s' (M5)", e.Name.Name, n.Name)
					return bad()
				}
				return f.symbolValue(member, e.Name.Pos, want)
			case SymType:
				if s, ok := sym.Type.(*types.Sealed); ok {
					v := s.VariantByName(e.Name.Name)
					if v == nil {
						f.errorf(e.Name.Pos, "'%s' has no variant '%s'", s.Name, e.Name.Name)
						return bad()
					}
					if len(v.Fields) == 0 {
						return f.variantValue(v, want, e.Pos)
					}
					f.errorf(e.Pos, "variant '%s.%s' needs its fields: '%s.%s(...)'", s.Name, v.Name, s.Name, v.Name)
					return bad()
				}
				if st, ok := f.c.symType(sym).(*types.Struct); ok {
					// `Status.ok`: a static val, or a static fun used as a value
					f.c.refSym(n.Pos, sym)
					return f.staticValue(st, e, want)
				}
			}
		}
	}
	if rt := f.moduleTypeNamed(e.X); rt != nil {
		// `http.Status.ok`
		if st, ok := rt.(*types.Struct); ok {
			return f.staticValue(st, e, want)
		}
		if !types.IsInvalid(rt) {
			f.errorf(e.Name.Pos, "'%s' has no static '%s'", rt, e.Name.Name)
		}
		return bad()
	}
	x := f.checkExpr(e.X, nil)
	r := f.fieldAccess(x, e, want)
	// a field path the flow has narrowed (`when (config.cause) { is E => config.cause.x }`)
	if p, ok := f.placeOf(e); ok {
		if n := f.narrowPlace(r, p); n != r {
			f.c.renarrowRef(e.Name.Pos, n.Type(), r.Type())
			r = n
		}
	}
	return r
}

// staticValue checks `Type.name` in value position: a `static val` of the
// struct (a module global), or a static function as a value.
func (f *fnCtx) staticValue(st *types.Struct, e *ast.MemberExpr, want types.Type) Expr {
	name := e.Name.Name
	tmpl := st
	if st.Template != nil {
		tmpl = st.Template
	}
	if sym, ok := f.c.staticVals[tmpl][name]; ok {
		if !sym.Pub && sym.Module != f.module {
			f.errorf(e.Name.Pos, "'%s.%s' is private to module '%s' (M5)", st.Name, name, sym.Module.Name())
			return bad()
		}
		f.c.refSym(e.Name.Pos, sym)
		return f.symbolValue(sym, e.Name.Pos, want)
	}
	if t, _, _ := f.findMethod(st, name); t != nil && t.Decl.Static {
		return f.funcValue(t, e.Name.Pos)
	}
	f.errorf(e.Name.Pos, "'%s' has no static '%s'; a 'static val' or 'static fun' in its body declares one (D23)", st.Name, name)
	return bad()
}

func (f *fnCtx) symbolValue(sym *Symbol, span source.Span, want types.Type) Expr {
	switch sym.Kind {
	case SymLocal:
		v := f.localVar(sym.Var)
		markUsed(v)
		return f.narrowedRef(v)
	case SymGlobal:
		v := f.globalVar(sym.Global)
		return &VarRef{exprBase{v.Type}, v}
	case SymFunc:
		return f.funcValue(sym.Func, span)
	case SymType:
		f.errorf(span, "'%s' is a type, not a value", sym.Name)
	default:
		f.errorf(span, "'%s' is not a value", sym.Name)
	}
	return bad()
}

// fieldAccess handles `x.name` and `x?.name` on values; auto-derefs (D39).
func (f *fnCtx) fieldAccess(x Expr, e *ast.MemberExpr, want types.Type) Expr {
	if e.Safe {
		x = f.flattenNullable(x)
		nt, ok := x.Type().(*types.Nullable)
		if !ok {
			f.errorf(e.Pos, "'?.' on a non-nullable value of type '%s'; use '.'", x.Type())
			return bad()
		}
		// let tmp = x; if tmp == null then null else Some(tmp!.name)
		tmp := f.newTemp(nt)
		inner := f.fieldOf(&Unwrap{exprBase{nt.Elem}, &VarRef{exprBase{nt}, tmp}}, e.Name, e.Pos)
		if types.IsInvalid(inner.Type()) {
			return inner
		}
		rt := types.Type(&types.Nullable{Elem: inner.Type()})
		if _, alreadyNullable := inner.Type().(*types.Nullable); alreadyNullable {
			// D30: chain type is T?; a nullable field stays single-nullable
			rt = inner.Type()
		} else {
			inner = &SomeWrap{exprBase{rt}, inner}
		}
		body := &If{exprBase{rt}, &IsNull{exprBase{types.TBool}, &VarRef{exprBase{nt}, tmp}},
			&Block{Value: &NullConst{exprBase{rt}}, Type: rt},
			&Block{Value: inner, Type: rt}}
		return &Let{exprBase{rt}, tmp, x, body}
	}
	return f.fieldOf(x, e.Name, e.Pos)
}

// flattenNullable turns a value nullable more than once (`T??`, as
// `xs.at(i)` yields on a `List<T?>`) into a `T?` for a `?.` chain: null at
// any level is null (D30: the chain's type is `T?`).
func (f *fnCtx) flattenNullable(x Expr) Expr {
	for {
		nt, ok := x.Type().(*types.Nullable)
		if !ok {
			return x
		}
		inner, nested := nt.Elem.(*types.Nullable)
		if !nested {
			return x
		}
		tmp := f.newTemp(nt)
		body := &If{exprBase{inner}, &IsNull{exprBase{types.TBool}, &VarRef{exprBase{nt}, tmp}},
			&Block{Value: &NullConst{exprBase{inner}}, Type: inner},
			&Block{Value: &Unwrap{exprBase{inner}, &VarRef{exprBase{nt}, tmp}}, Type: inner}}
		x = &Let{exprBase{inner}, tmp, x, body}
	}
}

func (f *fnCtx) fieldOf(x Expr, name ast.Ident, span source.Span) Expr {
	t := x.Type()
	if p, ok := t.(*types.Pointer); ok {
		if p.Raw && f.unsafe == 0 {
			f.errorf(span, "dereferencing a raw pointer requires an 'unsafe' block (D44)")
		}
		x = &Deref{exprBase{p.Elem}, x}
		t = p.Elem
	}
	switch tt := t.(type) {
	case *types.Struct:
		fld := f.lookupField(tt, name.Name, name.Pos)
		if fld == nil {
			return bad()
		}
		return &FieldGet{exprBase{fld.Type}, x, fld.Index, fld.Name}
	case *types.Tuple:
		idx, err := strconv.Atoi(name.Name)
		if err != nil || idx < 0 || idx >= len(tt.Elems) {
			f.errorf(name.Pos, "tuple of %d elements has no element '%s'", len(tt.Elems), name.Name)
			return bad()
		}
		return &TupleGet{exprBase{tt.Elems[idx]}, x, idx}
	case *types.Range:
		switch name.Name {
		case "lo":
			return &FieldGet{exprBase{tt.Elem}, x, 0, "lo"}
		case "hi":
			return &FieldGet{exprBase{tt.Elem}, x, 1, "hi"}
		case "inclusive":
			return &FieldGet{exprBase{types.TBool}, x, 2, "inclusive"}
		}
	case *types.Nullable:
		f.errorf(span, "value of type '%s' may be null; use '?.', '?:' or check for null first (D5)", tt)
		return bad()
	case *types.Sealed:
		if v := resultTest(tt, name.Name); v != nil {
			// `r.ok` / `r.err`: the tag test, spelled as a property; it smart-casts
			// like `r is Ok` (condFacts reads it the same way)
			return &VariantTest{exprBase{types.TBool}, x, v}
		}
		f.errorf(span, "'%s' is a sealed trait; match on its variants with 'when' before accessing '%s' (D13)", tt, name.Name)
		return bad()
	}
	if types.IsInvalid(t) {
		return bad()
	}
	f.errorf(span, "type '%s' has no field '%s'", t, name.Name)
	return bad()
}

// resultTest returns the variant `r.ok` or `r.err` tests on a Result, or
// nil when t is not a Result or name is neither.
func resultTest(t types.Type, name string) *types.Struct {
	s, ok := t.(*types.Sealed)
	if !ok || !isResultType(s) {
		return nil
	}
	switch name {
	case "ok":
		return s.VariantByName("Ok")
	case "err":
		return s.VariantByName("Err")
	}
	return nil
}

func (f *fnCtx) indexExpr(e *ast.IndexExpr) Expr {
	// Bracket indexing is no longer in the language (lint_index.go); the
	// old forms are still typed so that one diagnostic, with its fix, is
	// all the author sees.
	x := f.checkExpr(e.X, nil)
	switch t := x.Type().(type) {
	case *types.List:
		f.indexRead(e, false)
		idx := f.indexValue(e.Index)
		return &Builtin{exprBase{t.Elem}, "list.get", []Expr{x, idx}, e.Pos}
	case *types.Map:
		f.indexRead(e, true)
		k := f.checkExprTo(e.Index, t.Key)
		return &Builtin{exprBase{&types.Nullable{Elem: t.Value}}, "map.get", []Expr{x, k}, e.Pos}
	case *types.Basic:
		if t.Kind == types.String {
			f.errorf(e.Pos, "strings are not indexable; use 's.byteAt(i)' for a byte or 's.chars()' for the code points (D18)")
			return bad()
		}
	}
	if !types.IsInvalid(x.Type()) {
		f.errorf(e.Pos, "'[...]' after a value of type '%s' is not indexing; brackets are for collection literals only (D25)", x.Type())
	}
	return bad()
}

// ---------------------------------------------------------------------------
// variant constructors: Some / None / Ok / Err and `.name`

func (f *fnCtx) variantValue(v *types.Struct, want types.Type, span source.Span) Expr {
	if len(v.TypeParams) > 0 && v.TypeArgs == nil {
		// generic field-less variant: needs the expected type
		if s, ok := numericHint(want).(*types.Sealed); ok && sealedTemplate(s) == v.Sealed {
			v = s.Variants[v.Tag]
		} else {
			f.errorf(span, "cannot infer the type arguments of '%s' here", v.Name)
			return bad()
		}
	}
	lit := &StructLit{exprBase{v}, v, nil}
	return &MakeVariant{exprBase{v.Sealed}, v.Sealed, v, lit}
}

// variantCtor handles the prelude constructors with expected-type-driven
// inference. In a throwing function `Err(e)` outside a Result context
// throws (D4: throws is sugar over Result).
func (f *fnCtx) variantCtor(name string, args []ast.Arg, want types.Type, span source.Span) Expr {
	hint := want
	switch name {
	case "None":
		if len(args) != 0 {
			f.errorf(span, "'None' takes no arguments")
		}
		if n, ok := hint.(*types.Nullable); ok {
			return &NullConst{exprBase{n}}
		}
		f.errorf(span, "cannot infer the type of 'None' here; annotate the binding")
		return bad()
	case "Some":
		if len(args) != 1 {
			f.errorf(span, "'Some' takes exactly one argument")
			return bad()
		}
		var inner Expr
		if n, ok := hint.(*types.Nullable); ok {
			inner = f.checkExprTo(args[0].Value, n.Elem)
		} else {
			inner = f.checkExpr(args[0].Value, nil)
		}
		if types.IsInvalid(inner.Type()) {
			return bad()
		}
		return &SomeWrap{exprBase{&types.Nullable{Elem: inner.Type()}}, inner}
	case "Ok", "Err":
		if len(args) != 1 {
			f.errorf(span, "'%s' takes exactly one argument", name)
			return bad()
		}
		if rs, ok := hint.(*types.Sealed); ok && isResultType(rs) {
			v := rs.Variants[0]
			if name == "Err" {
				v = rs.Variants[1]
			}
			inner := f.checkExprTo(args[0].Value, v.Fields[0].Type)
			lit := &StructLit{exprBase{v}, v, []Expr{inner}}
			return &MakeVariant{exprBase{rs}, rs, v, lit}
		}
		if f.throws {
			if name == "Ok" {
				return f.checkExprTo(args[0].Value, f.retType)
			}
			errv := f.checkExpr(args[0].Value, nil)
			return f.throwExpr(errv, span)
		}
		if name == "Err" {
			f.errorf(span, "'Err(...)' here would fail the function, but it is not declared 'throws' (D4); or bind it to a 'Result<T, E>' typed value")
		} else {
			f.errorf(span, "cannot infer the Result type of 'Ok(...)' here; annotate the binding as 'Result<T, E>'")
		}
		return bad()
	}
	return bad()
}

// throwExpr returns Err(errv) from the enclosing throwing function.
func (f *fnCtx) throwExpr(errv Expr, span source.Span) Expr {
	et := errv.Type()
	if types.IsInvalid(et) {
		return bad()
	}
	if u, ok := et.(*types.ErrorUnion); ok {
		for _, m := range u.Members {
			f.recordError(m, span)
		}
	} else {
		f.recordError(et, span)
	}
	f.c.checkErrorType(et, span)
	return &Throw{exprBase{types.TNever}, errv, et, f.currentErrType()}
}

// recordError adds an error type to the function's inferred set or checks
// it against the declared set.
func (f *fnCtx) recordError(t types.Type, span source.Span) {
	if f.isGlobal {
		f.errorf(span, "errors cannot be raised in a global initializer")
		return
	}
	if f.errType != nil {
		if types.UnionIndex(f.errType, t) < 0 {
			f.errorf(span, "error type '%s' is not in the declared 'throws %s' of this function (D45: declare it or widen the union)", t, f.errType)
		}
		return
	}
	for _, m := range f.fn.inferredErrors {
		if types.Identical(m, t) {
			return
		}
	}
	f.fn.inferredErrors = append(f.fn.inferredErrors, t)
}

func (f *fnCtx) currentErrType() types.Type {
	if f.errType != nil {
		return f.errType
	}
	if f.fn != nil {
		return f.fn.Sig.Effects.Error
	}
	return nil
}

// ---------------------------------------------------------------------------
// operators

func (f *fnCtx) unaryExpr(e *ast.UnaryExpr, want types.Type) Expr {
	switch e.Op {
	case lexer.Minus:
		if lit, ok := e.X.(*ast.IntLit); ok {
			return f.intLit(lit, want, true)
		}
		x := f.checkExpr(e.X, want)
		if types.IsInvalid(x.Type()) {
			return x
		}
		if !types.IsNumeric(x.Type()) || types.IsUnsigned(x.Type()) {
			f.errorf(e.Pos, "cannot negate a value of type '%s'", x.Type())
			return bad()
		}
		return &Unary{exprBase{x.Type()}, OpNeg, x, e.Pos}
	case lexer.Bang:
		x := f.checkExprTo(e.X, types.TBool)
		return &Unary{exprBase{types.TBool}, OpNot, x, e.Pos}
	case lexer.Tilde:
		x := f.checkExpr(e.X, want)
		if !types.IsInteger(x.Type()) {
			if !types.IsInvalid(x.Type()) {
				f.errorf(e.Pos, "'~' is only defined for integers, not '%s'", x.Type())
			}
			return bad()
		}
		return &Unary{exprBase{x.Type()}, OpBitNot, x, e.Pos}
	case lexer.Amp:
		// D10: address of a local (heap-promoted); D50: always a GC pointer.
		// The address of a temporary boxes the value.
		if !isPlaceSyntax(e.X) {
			if repl, ok := elemReadCall(e.X); ok {
				// `&xs.atOrPanic(i)` would box a copy of the element; the
				// pointer to the element itself is `refOrPanic` (D25, v0.27)
				f.c.errorFix(e.Pos, fixReplace("Replace with '"+repl+"'", e.Pos, repl),
					"'&%s' is the address of a copy of the element, not of the element; use '%s' (D25)", srcText(e.X), repl)
				f.checkExpr(e.X, nil)
				return bad()
			}
			x := f.checkExpr(e.X, nil)
			if types.IsInvalid(x.Type()) {
				return bad()
			}
			return &AddrOf{exprBase{&types.Pointer{Elem: x.Type()}}, x}
		}
		lv, root := f.checkLValue(e.X, false)
		if lv == nil {
			return bad()
		}
		if root != nil {
			root.AddrTaken = true
			f.invalidatePaths(root) // fields may now change through the pointer
		}
		return &AddrOf{exprBase{&types.Pointer{Elem: lv.Type()}}, lv}
	case lexer.Star:
		x := f.checkExpr(e.X, nil)
		p, ok := x.Type().(*types.Pointer)
		if !ok {
			if !types.IsInvalid(x.Type()) {
				f.errorf(e.Pos, "cannot dereference a value of type '%s'", x.Type())
			}
			return bad()
		}
		if p.Raw && f.unsafe == 0 {
			f.errorf(e.Pos, "dereferencing a raw pointer requires an 'unsafe' block (D44)")
		}
		return &Deref{exprBase{p.Elem}, x}
	}
	f.errorf(e.Pos, "unsupported unary operator")
	return bad()
}

func isLiteralExpr(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.IntLit, *ast.FloatLit:
		return true
	case *ast.UnaryExpr:
		return e.Op == lexer.Minus && isLiteralExpr(e.X)
	}
	return false
}

func (f *fnCtx) binaryExpr(e *ast.BinaryExpr, want types.Type) Expr {
	op := BinOpFromToken(e.Op)
	switch op {
	case OpAnd, OpOr:
		l := f.checkExprTo(e.L, types.TBool)
		// narrowing flows into the right operand
		saved := f.saveNarrow()
		tf, ff := f.condFacts(e.L, l)
		if op == OpAnd {
			f.applyFacts(tf)
		} else {
			f.applyFacts(ff)
		}
		r := f.checkExprTo(e.R, types.TBool)
		f.restoreNarrow(saved)
		return &Binary{exprBase{types.TBool}, op, l, r, e.Pos}
	case OpEq, OpNe:
		return f.equality(e, op)
	case OpShl, OpShr:
		// the count keeps its own integer type
		l := f.checkExpr(e.L, want)
		r := f.checkExpr(e.R, nil)
		return f.makeBinary(op, l, r, e.Pos)
	}
	// Arithmetic and ordering: operands must share a type. Literals adapt
	// to the other side.
	var l, r Expr
	if isLiteralExpr(e.L) && !isLiteralExpr(e.R) {
		r = f.checkExpr(e.R, want)
		l = f.checkExprTo(e.L, r.Type())
	} else {
		l = f.checkExpr(e.L, want)
		r = f.checkExprTo(e.R, l.Type())
	}
	return f.makeBinary(op, l, r, e.Pos)
}

func (f *fnCtx) makeBinary(op BinOp, l, r Expr, span source.Span) Expr {
	t := l.Type()
	if types.IsInvalid(t) || types.IsInvalid(r.Type()) {
		return bad()
	}
	switch op {
	case OpAdd:
		if types.IsString(t) {
			return &StringConcat{exprBase{types.TString}, []Expr{l, r}}
		}
		fallthrough
	case OpSub, OpMul, OpDiv, OpRem:
		if !types.IsNumeric(t) {
			f.errorf(span, "operator '%s' is not defined for '%s'", op, t)
			return bad()
		}
		if op == OpRem && types.IsFloat(t) {
			f.errorf(span, "'%%' is not defined for floats")
			return bad()
		}
		return &Binary{exprBase{t}, op, l, r, span}
	case OpWrapAdd, OpWrapSub, OpWrapMul:
		if !types.IsInteger(t) {
			f.errorf(span, "wrapping operator '%s' is only defined for integers (D21)", op)
			return bad()
		}
		return &Binary{exprBase{t}, op, l, r, span}
	case OpBitAnd, OpBitOr, OpBitXor:
		if !types.IsInteger(t) {
			f.errorf(span, "operator '%s' is only defined for integers, not '%s'", op, t)
			return bad()
		}
		return &Binary{exprBase{t}, op, l, r, span}
	case OpShl, OpShr:
		// the count may be any integer type; a count at or beyond the width
		// shifts everything out (Go's rule, no undefined behaviour)
		if !types.IsInteger(t) {
			f.errorf(span, "operator '%s' is only defined for integers, not '%s'", op, t)
			return bad()
		}
		if !types.IsInteger(r.Type()) {
			f.errorf(span, "a shift count must be an integer, not '%s'", r.Type())
			return bad()
		}
		return &Binary{exprBase{t}, op, l, r, span}
	case OpLt, OpLe, OpGt, OpGe:
		if types.IsNumeric(t) || types.IsString(t) {
			return &Binary{exprBase{types.TBool}, op, l, r, span}
		}
		if cmp := f.compareOp(op, l, r, span); cmp != nil {
			return cmp
		}
		f.errorf(span, "operator '%s' is not defined for '%s'; implement 'Comparable' to order it", op, t)
		return bad()
	}
	f.errorf(span, "unsupported operator")
	return bad()
}

// ordered reports whether values of t can be compared with `<`: numbers,
// strings, and types implementing the prelude's Comparable.
func (f *fnCtx) ordered(t types.Type) bool {
	return f.c.orderedType(t)
}

// compareOp lowers an ordering operator on a type with a custom
// Comparable impl to `l.compareTo(r) <op> 0`; nil when t has none.
func (f *fnCtx) compareOp(op BinOp, l, r Expr, span source.Span) Expr {
	ops := f.c.customOps(l.Type())
	if ops == nil || ops.Compare == nil {
		return nil
	}
	cmp := &Call{exprBase: exprBase{types.TI64}, Fn: ops.Compare, Args: []Expr{recvArg(ops.Compare, l), r}}
	return &Binary{exprBase{types.TBool}, op, cmp, i64c(0), span}
}

// equality handles ==/!= including null comparisons and sealed/struct
// structural equality.
func (f *fnCtx) equality(e *ast.BinaryExpr, op BinOp) Expr {
	_, lNull := e.L.(*ast.NullLit)
	_, rNull := e.R.(*ast.NullLit)
	if lNull || rNull {
		other := e.R
		if rNull {
			other = e.L
		}
		x := f.checkExpr(other, nil)
		if _, ok := x.Type().(*types.Nullable); !ok {
			if p, isPlace := f.placeOf(other); isPlace {
				if _, declaredNullable := f.declaredTypeOf(p).(*types.Nullable); declaredNullable {
					// smart-cast to non-null already; the test is decided
					return &BoolConst{exprBase{types.TBool}, op == OpNe}
				}
			}
			if !types.IsInvalid(x.Type()) {
				f.errorf(e.Pos, "comparing a non-nullable '%s' with null is always %v", x.Type(), op == OpNe)
			}
			return &BoolConst{exprBase{types.TBool}, op == OpNe}
		}
		test := Expr(&IsNull{exprBase{types.TBool}, x})
		if op == OpNe {
			test = &Unary{exprBase{types.TBool}, OpNot, test, e.Pos}
		}
		return test
	}
	var l, r Expr
	if isLiteralExpr(e.L) && !isLiteralExpr(e.R) {
		r = f.checkExpr(e.R, nil)
		r = f.immutableView(r)
		l = f.checkExprTo(e.L, r.Type())
	} else {
		l = f.checkExpr(e.L, nil)
		l = f.immutableView(l)
		r = f.checkExprTo(e.R, l.Type())
	}
	t := l.Type()
	if types.IsInvalid(t) || types.IsInvalid(r.Type()) {
		return bad()
	}
	if !f.comparable(t) {
		f.errorf(e.Pos, "values of type '%s' cannot be compared with '%s'; implement 'Equatable' to define it", t, op)
		return bad()
	}
	return &Binary{exprBase{types.TBool}, op, l, r, e.Pos}
}

// immutableView coerces a mutable collection to its read-only view, so
// that `==` compares a MutableList with a List (they share one layout and
// one element-wise equality).
func (f *fnCtx) immutableView(x Expr) Expr {
	views := receiverViews(x.Type())
	if len(views) == 1 {
		return x
	}
	return f.coerce(x, views[1], source.Span{})
}

// comparable reports whether `==` is defined for t: structural equality
// over the fields, or a custom Equatable impl.
func (f *fnCtx) comparable(t types.Type) bool {
	return f.comparableIn(t, map[types.Type]bool{})
}

// comparableIn is comparable with the structs and sealed types already on
// the path in seen, so a type that contains itself by value (a D31 error)
// does not recurse forever.
func (f *fnCtx) comparableIn(t types.Type, seen map[types.Type]bool) bool {
	switch t := t.(type) {
	case *types.Basic:
		return t.Kind != types.Unit && t.Kind != types.Never && t.Kind != types.Invalid
	case *types.Pointer:
		return true
	case *types.Nullable:
		return f.comparableIn(t.Elem, seen)
	case *types.Struct:
		if seen[t] {
			return true
		}
		seen[t] = true
		if f.c.implementsPrelude(t, "Equatable") {
			return true
		}
		for _, fld := range t.Fields {
			if !f.comparableIn(fld.Type, seen) {
				return false
			}
		}
		return true
	case *types.Sealed:
		if seen[t] {
			return true
		}
		seen[t] = true
		if f.c.implementsPrelude(t, "Equatable") {
			return true
		}
		for _, v := range t.Variants {
			if !f.comparableIn(v, seen) {
				return false
			}
		}
		return true
	case *types.Tuple:
		for _, e := range t.Elems {
			if !f.comparableIn(e, seen) {
				return false
			}
		}
		return true
	case *types.List:
		// collections compare element-wise (D25, v0.26)
		return f.comparableIn(t.Elem, seen)
	case *types.Set:
		return f.comparableIn(t.Elem, seen)
	case *types.Map:
		return f.comparableIn(t.Key, seen) && f.comparableIn(t.Value, seen)
	}
	return false
}

// orFailExpr checks `x ?! err`: absence or failure becomes failure with
// `err`. A `T?` gives a `Result<T, E>`, a `Result<T, E1>` a `Result<T, E>`
// with the original error dropped (mapError keeps it); the right operand
// is evaluated only on that path. `try` then propagates as usual:
// `val user = try users.get(id) ?! notFound(id)`.
func (f *fnCtx) orFailExpr(e *ast.OrFailExpr) Expr {
	l := f.checkExpr(e.L, nil)
	lt := l.Type()
	if types.IsInvalid(lt) {
		f.checkExpr(e.R, nil)
		return bad()
	}
	var okT types.Type
	var isNullable bool
	switch t := lt.(type) {
	case *types.Nullable:
		okT, isNullable = t.Elem, true
	case *types.Sealed:
		if isResultType(t) {
			okT = t.TypeArgs[0]
		}
	}
	if okT == nil {
		f.errorf(e.Pos, "'?!' needs a nullable or a Result on its left, found '%s'", lt)
		f.checkExpr(e.R, nil)
		return bad()
	}
	r := f.checkExpr(e.R, nil)
	et := r.Type()
	if types.IsInvalid(et) {
		return bad()
	}
	if u, ok := et.(*types.ErrorUnion); ok {
		for _, m := range u.Members {
			f.c.checkErrorType(m, e.R.Span())
		}
	} else if _, isStruct := et.(*types.Struct); isStruct || types.IsNever(et) {
		f.c.checkErrorType(et, e.R.Span())
	} else {
		f.errorf(e.R.Span(), "the right operand of '?!' must be an error (a type declared with 'error'), found '%s'", et)
		return bad()
	}
	rs := f.c.ResultType(okT, et)
	okV, errV := rs.Variants[0], rs.Variants[1]
	tmp := f.newTemp(lt)
	var failed Expr
	var payload Expr
	if isNullable {
		failed = &IsNull{exprBase{types.TBool}, &VarRef{exprBase{lt}, tmp}}
		payload = &Unwrap{exprBase{okT}, &VarRef{exprBase{lt}, tmp}}
	} else {
		src := lt.(*types.Sealed)
		srcOk, srcErr := src.Variants[0], src.Variants[1]
		failed = &VariantTest{exprBase{types.TBool}, &VarRef{exprBase{lt}, tmp}, srcErr}
		payload = &FieldGet{exprBase{okT}, &VariantCast{exprBase{srcOk}, &VarRef{exprBase{lt}, tmp}, srcOk}, 0, srcOk.Fields[0].Name}
	}
	errValue := &MakeVariant{exprBase{rs}, rs, errV, &StructLit{exprBase{errV}, errV, []Expr{r}}}
	okValue := &MakeVariant{exprBase{rs}, rs, okV, &StructLit{exprBase{okV}, okV, []Expr{payload}}}
	pick := &If{exprBase{rs}, failed, &Block{Value: errValue, Type: rs}, &Block{Value: okValue, Type: rs}}
	return &Let{exprBase{rs}, tmp, l, pick}
}

func (f *fnCtx) elvisExpr(e *ast.ElvisExpr, want types.Type) Expr {
	l := f.checkExpr(e.L, nil)
	nt, ok := l.Type().(*types.Nullable)
	if !ok {
		if !types.IsInvalid(l.Type()) {
			f.errorf(e.Pos, "'?:' needs a nullable left operand, found '%s'", l.Type())
		}
		return bad()
	}
	r := f.checkExprTo(e.R, nt.Elem)
	return &Elvis{exprBase{nt.Elem}, l, r}
}

func (f *fnCtx) rangeExpr(e *ast.RangeExpr, want types.Type) Expr {
	var elemWant types.Type
	if r, ok := want.(*types.Range); ok {
		elemWant = r.Elem
	}
	var lo, hi Expr
	if isLiteralExpr(e.Lo) && !isLiteralExpr(e.Hi) {
		hi = f.checkExpr(e.Hi, elemWant)
		lo = f.checkExprTo(e.Lo, hi.Type())
	} else {
		lo = f.checkExpr(e.Lo, elemWant)
		hi = f.checkExprTo(e.Hi, lo.Type())
	}
	if !types.IsInteger(lo.Type()) {
		if !types.IsInvalid(lo.Type()) {
			f.errorf(e.Pos, "ranges are over integers, found '%s'", lo.Type())
		}
		return bad()
	}
	return &RangeLit{exprBase{&types.Range{Elem: lo.Type()}}, lo, hi, e.Inclusive}
}

func (f *fnCtx) listLit(e *ast.ListLit, want types.Type) Expr {
	var elem types.Type
	mutable := false
	// D25/#8: the expected type decides mutability; `mut` is for literals
	// with nothing to infer it from.
	if st, ok := numericHint(want).(*types.Set); ok {
		return f.setLit(e, st)
	}
	if lt, ok := numericHint(want).(*types.List); ok {
		// a still-generic expected type (`items: MutableList<T>` during
		// inference) settles the mutability; the element type is inferred
		if !types.ContainsTypeParam(lt.Elem) {
			elem = lt.Elem
		}
		mutable = lt.Mutable
		if e.Mut && mutable {
			f.warnFix(e.Pos, fixDropMut(e.Pos), "redundant 'mut': the expected type '%s' already makes the literal mutable", lt)
		}
	}
	if e.Mut {
		mutable = true
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
		if e.Mut {
			f.errorf(e.Pos, "cannot infer the element type of an empty list; annotate it, e.g. 'var xs: MutableList<i32> = []' (D25)")
		} else {
			f.errorf(e.Pos, "cannot infer the element type of an empty list; annotate it, e.g. 'val xs: List<i32> = []' (D25)")
		}
		return bad()
	}
	lit.T = &types.List{Elem: elem, Mutable: mutable}
	return lit
}

// setLit builds a set from a list literal where a Set is expected:
// `val s: Set<i64> = [1, 2]` becomes `{ val tmp = MutableSet(); tmp.add(1); ...; tmp }`.
func (f *fnCtx) setLit(e *ast.ListLit, st *types.Set) Expr {
	if e.Mut && st.Mutable {
		f.warnFix(e.Pos, fixDropMut(e.Pos), "redundant 'mut': the expected type '%s' already makes the literal mutable", st)
	}
	if !f.c.checkHashable(st.Elem, e.Pos) {
		return bad()
	}
	outT := &types.Set{Elem: st.Elem, Mutable: true}
	out := f.newTemp(outT)
	stmts := []Stmt{&VarDecl{Var: out, Init: &Builtin{exprBase{outT}, "set.new", nil, e.Pos}}}
	for _, el := range e.Elems {
		x := f.checkExprTo(el, st.Elem)
		stmts = append(stmts, &ExprStmt{X: &Builtin{exprBase{types.TBool}, "set.add", []Expr{ref(out), x}, e.Pos}})
	}
	resultT := &types.Set{Elem: st.Elem, Mutable: st.Mutable || e.Mut}
	return &BlockExpr{exprBase{resultT}, &Block{Stmts: stmts, Value: &Cast{exprBase{resultT}, ref(out)}, Type: resultT}}
}

func (f *fnCtx) castExpr(e *ast.CastExpr) Expr {
	to := f.resolve(e.Type)
	var x Expr
	if isLiteralExpr(e.X) {
		x = f.checkExpr(e.X, nil) // `300 as u8` converts an i32, it does not retype the literal
	} else {
		x = f.checkExpr(e.X, to)
	}
	from := x.Type()
	if types.IsInvalid(from) || types.IsInvalid(to) {
		return bad()
	}
	if types.Identical(from, to) {
		return x
	}
	if types.IsNumeric(from) && types.IsNumeric(to) {
		return &Cast{exprBase{to}, x}
	}
	if conv := f.convert(x, to); conv != nil {
		return conv
	}
	f.errorf(e.Pos, "cannot cast '%s' to '%s'; 'as' converts between numeric types only", from, to)
	return bad()
}

func (f *fnCtx) tryExpr(e *ast.TryExpr) Expr {
	if x, done := f.tryChain(e); done {
		return x
	}
	x := f.checkExpr(e.X, nil)
	return f.tryOn(x, e.Pos)
}

// tryChain handles `try f().m(...)` where `f()` is a Result and `m` is not
// a method of Result: `try` covers the whole chain by grammar, so `m`
// would be looked up on the Result and fail. What was meant is
// `(try f()).m(...)`, and that is what this checks — with a warning and
// the fix that writes the parentheses, so the source says what it does.
// Only a call or member chain qualifies as the receiver (a bare name may
// be a type, and `try r.m()` on a Result variable stays as it reads).
func (f *fnCtx) tryChain(e *ast.TryExpr) (Expr, bool) {
	call, ok := e.X.(*ast.CallExpr)
	if !ok || call.Async {
		return nil, false
	}
	mem, ok := call.Fun.(*ast.MemberExpr)
	if !ok || mem.Safe {
		return nil, false
	}
	switch mem.X.(type) {
	case *ast.CallExpr, *ast.MemberExpr:
	default:
		return nil, false
	}
	if mx, isMember := mem.X.(*ast.MemberExpr); isMember && f.moduleTypeNamed(mx) != nil {
		return nil, false
	}
	recv := f.checkExpr(mem.X, nil)
	var typeArgs []types.Type
	for _, ta := range call.TypeArgs {
		typeArgs = append(typeArgs, f.resolve(ta))
	}
	rs, isSealed := recv.Type().(*types.Sealed)
	if !isSealed || !isResultType(rs) || f.hasMethod(rs, mem.Name.Name) {
		// the ordinary reading: the method applies to what f() returned
		return f.tryOn(f.dispatchMethod(recv, mem, typeArgs, call, nil), e.Pos), true
	}
	span := source.Span{File: e.Pos.File, Start: e.Pos.Start, End: mem.X.Span().End}
	f.warnFix(e.Pos, fixReplace("Write '(try ...)' around the call", span, "(try "+srcText(mem.X)+")"),
		"'try' covers the whole chain, but '%s' is not a method of '%s'; read as '(try %s).%s(...)' — write the parentheses", mem.Name.Name, rs, srcText(mem.X), mem.Name.Name)
	unwrapped := f.tryOn(recv, e.Pos)
	if types.IsInvalid(unwrapped.Type()) {
		f.checkArgsLoosely(call.Args)
		return bad(), true
	}
	return f.dispatchMethod(unwrapped, mem, typeArgs, call, nil), true
}

// hasMethod reports whether a method of that name applies to the type
// through an extend block or a trait impl (a Result has no built-ins).
func (f *fnCtx) hasMethod(rt types.Type, name string) bool {
	for _, view := range receiverViews(rt) {
		for _, ext := range f.c.extends {
			if _, ok := ext.Methods[name]; ok && unify(ext.Target, view, map[*types.TypeParam]types.Type{}) {
				return true
			}
		}
	}
	for trait, impls := range f.c.impls {
		if _, has := trait.Methods[name]; !has {
			continue
		}
		for _, impl := range impls {
			if unify(impl.Target, rt, map[*types.TypeParam]types.Type{}) {
				return true
			}
		}
	}
	return false
}

// tryOn applies `try` to a checked operand.
func (f *fnCtx) tryOn(x Expr, pos source.Span) Expr {
	e := struct{ Pos source.Span }{pos}
	if errPolyCall(x) {
		// generic code calling something declared `throws E`: in this
		// instance E is Never, the call is plain (its value may itself be a
		// Result the caller asked for), and `try` is the identity
		return x
	}
	rs, ok := x.Type().(*types.Sealed)
	if !ok || !isResultType(rs) {
		if f.neverInstance() && !types.IsInvalid(x.Type()) {
			// the same erasure one step removed: the operand was a
			// `Result<R, E>` in the template — an awaited `Task<Result<R, E>>`,
			// a stored call result — and E is Never in this instance, so the
			// value is already the payload
			return x
		}
		if !types.IsInvalid(x.Type()) {
			f.errorf(e.Pos, "'try' needs a Result (a call to a 'throws' function), found '%s'", x.Type())
		}
		return bad()
	}
	okT, errT := rs.TypeArgs[0], rs.TypeArgs[1]
	if types.IsNever(errT) {
		// nothing can be propagated: `try` just unwraps (a `throws E` callee
		// whose E is Never in this instance still returns a Result)
		tmp := f.newTemp(rs)
		okV := rs.Variants[0]
		var payload Expr = &FieldGet{exprBase{okT}, &VariantCast{exprBase{okV}, &VarRef{exprBase{rs}, tmp}, okV}, 0, okV.Fields[0].Name}
		return &Let{exprBase{okT}, tmp, x, payload}
	}
	if !f.throws {
		f.errorf(e.Pos, "'try' propagates an error, but the enclosing function is not declared 'throws'; add 'throws' or handle the Result with 'when' (D4)")
		return &ResultValue{exprBase{okT}, x, false}
	}
	if !types.IsNever(errT) {
		for _, m := range types.UnionMembers(errT) {
			f.recordError(m, e.Pos)
		}
	}
	return &Try{exprBase{okT}, x, errT, f.currentErrType()}
}

// ---------------------------------------------------------------------------
// if / is and flow-sensitive narrowing (D5, D13)

// place is what a smart cast is about: a local variable, or a chain of
// direct struct fields from one (`config.cause`). Only fields of value
// structs count — nothing reached through a pointer, `?.` or an index —
// so the place cannot change behind the test's back except by an
// assignment to it, which invalidates the fact (D5).
type place struct {
	v    *Var
	path string // "" for the variable itself; "cause" / "cause.inner" for fields
}

func pv(v *Var) place { return place{v: v} }

type facts map[place]types.Type

func (f *fnCtx) saveNarrow() facts {
	out := facts{}
	for k, v := range f.narrow {
		out[k] = v
	}
	return out
}

func (f *fnCtx) restoreNarrow(saved facts) {
	f.narrow = map[place]types.Type{}
	for k, v := range saved {
		f.narrow[k] = v
	}
}

// invalidatePlace drops the facts an assignment to p can break: p itself
// and every field path under it. An assignment to the whole variable
// drops everything rooted at it.
func (f *fnCtx) invalidatePlace(p place) {
	for k := range f.narrow {
		if k.v != p.v {
			continue
		}
		if p.path == "" || k.path == p.path || strings.HasPrefix(k.path, p.path+".") {
			delete(f.narrow, k)
		}
	}
}

// invalidatePaths drops the field-path facts rooted at v but keeps the
// fact about v itself: `&v` may rewrite v's fields
// but cannot change which variant v is.
func (f *fnCtx) invalidatePaths(v *Var) {
	for k := range f.narrow {
		if k.v == v && k.path != "" {
			delete(f.narrow, k)
		}
	}
}

// invalidateVarPaths drops the field-path facts rooted at v that a method
// call on v (or on one of its fields) can break: those with a `var` field
// somewhere on the path. A bare field is never assigned, so a fact about
// it survives any call (D22 v0.30).
func (f *fnCtx) invalidateVarPaths(v *Var) {
	for k := range f.narrow {
		if k.v != v || k.path == "" {
			continue
		}
		t := f.declaredTypeOf(pv(v))
		for _, name := range strings.Split(k.path, ".") {
			st, ok := t.(*types.Struct)
			if !ok {
				break
			}
			var fld *types.Field
			for _, cand := range st.Fields {
				if cand.Name == name {
					fld = cand
					break
				}
			}
			if fld == nil {
				break
			}
			if fld.Var {
				delete(f.narrow, k)
				break
			}
			t = fld.Type
		}
	}
}

// placeOf returns the place an expression denotes: a local variable or a
// chain of direct struct fields from one.
func (f *fnCtx) placeOf(e ast.Expr) (place, bool) {
	switch e := e.(type) {
	case *ast.NameExpr:
		if v := varOf(e, f); v != nil {
			return pv(v), true
		}
	case *ast.SelfExpr:
		// the receiver is a pointer to its place (D22); a method call on
		// `self` drops the facts about its `var` fields (invalidateVarPaths)
		if self := f.selfRef(); self != nil {
			return pv(self), true
		}
	case *ast.MemberExpr:
		if e.Safe {
			return place{}, false
		}
		base, ok := f.placeOf(e.X)
		if !ok || base.v.AddrTaken {
			// with a pointer to the variable around, its fields can change
			// behind a test's back
			return place{}, false
		}
		st, isStruct := f.currentTypeOf(base).(*types.Struct)
		if !isStruct {
			return place{}, false
		}
		for _, fld := range st.Fields {
			if fld.Name == e.Name.Name {
				p := place{v: base.v, path: e.Name.Name}
				if base.path != "" {
					p.path = base.path + "." + e.Name.Name
				}
				return p, true
			}
		}
	}
	return place{}, false
}

// currentTypeOf is the flow-sensitive type of a place: its narrowed type
// if a fact holds, else the field's declared type on the (narrowed)
// prefix, else the variable's type.
func (f *fnCtx) currentTypeOf(p place) types.Type {
	if t, ok := f.narrow[p]; ok {
		if fld := payloadField(t); fld != nil {
			return fld.Type
		}
		return t
	}
	return f.declaredTypeOf(p)
}

// declaredTypeOf is the type of a place before any fact about the place
// itself (facts about its prefix still apply).
func (f *fnCtx) declaredTypeOf(p place) types.Type {
	if p.path == "" {
		if p.v.IsSelf {
			return p.v.Type.(*types.Pointer).Elem
		}
		return p.v.Type
	}
	prefix, last := place{v: p.v}, p.path
	if i := strings.LastIndex(p.path, "."); i >= 0 {
		prefix.path, last = p.path[:i], p.path[i+1:]
	}
	if st, ok := f.currentTypeOf(prefix).(*types.Struct); ok {
		for _, fld := range st.Fields {
			if fld.Name == last {
				return fld.Type
			}
		}
	}
	return types.TInvalid
}

// narrowPlace applies the fact about p, if any, to the expression that
// reads it (D5): T? to T, sealed to variant, Result to payload.
func (f *fnCtx) narrowPlace(x Expr, p place) Expr {
	to, ok := f.narrow[p]
	if !ok {
		return x
	}
	x = narrowExpr(x, to)
	if fld := payloadField(to); fld != nil && types.Identical(x.Type(), to) {
		x = &FieldGet{exprBase{fld.Type}, x, fld.Index, fld.Name}
	}
	return x
}

func (f *fnCtx) applyFacts(fs facts) {
	for k, v := range fs {
		if v == nil {
			delete(f.narrow, k)
		} else {
			f.narrow[k] = v
		}
	}
}

// varOf returns the local variable an expression denotes, if it is a plain
// (possibly narrowed) variable reference.
func varOf(e ast.Expr, f *fnCtx) *Var {
	n, ok := e.(*ast.NameExpr)
	if !ok {
		return nil
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymLocal {
		return nil
	}
	return f.localVar(sym.Var)
}

// condFacts computes what is known about variables when the condition is
// true and when it is false.
func (f *fnCtx) condFacts(cond ast.Expr, checked Expr) (whenTrue, whenFalse facts) {
	whenTrue, whenFalse = facts{}, facts{}
	switch c := cond.(type) {
	case *ast.BinaryExpr:
		switch c.Op {
		case lexer.Eq, lexer.NotEq:
			_, lNull := c.L.(*ast.NullLit)
			_, rNull := c.R.(*ast.NullLit)
			if lNull == rNull {
				return
			}
			other := c.R
			if rNull {
				other = c.L
			}
			v, isPlace := f.placeOf(other)
			if !isPlace {
				return
			}
			cur := f.currentTypeOf(v)
			nt, ok := cur.(*types.Nullable)
			if !ok {
				return
			}
			if c.Op == lexer.Eq {
				whenFalse[v] = nt.Elem
			} else {
				whenTrue[v] = nt.Elem
			}
		case lexer.AndAnd:
			t1, _ := f.condFacts(c.L, nil)
			t2, _ := f.condFacts(c.R, nil)
			for k, v := range t1 {
				whenTrue[k] = v
			}
			for k, v := range t2 {
				whenTrue[k] = v
			}
		case lexer.OrOr:
			_, f1 := f.condFacts(c.L, nil)
			_, f2 := f.condFacts(c.R, nil)
			for k, v := range f1 {
				whenFalse[k] = v
			}
			for k, v := range f2 {
				whenFalse[k] = v
			}
		}
	case *ast.UnaryExpr:
		if c.Op == lexer.Bang {
			t, fl := f.condFacts(c.X, nil)
			return fl, t
		}
	case *ast.MemberExpr:
		// `r.ok` / `r.err` narrow like `r is Ok` / `r is Err`
		if c.Safe {
			return
		}
		v, ok := f.placeOf(c.X)
		if !ok {
			return
		}
		from := f.currentTypeOf(v)
		target := resultTest(from, c.Name.Name)
		if target == nil {
			return
		}
		whenTrue[v] = target
		whenFalse[v] = from.(*types.Sealed).Variants[1-target.Tag]
	case *ast.IsExpr:
		v, ok := f.placeOf(c.X)
		if !ok {
			return
		}
		from := f.currentTypeOf(v)
		target := f.patternTargetType(from, c.Pat)
		if target == nil {
			return
		}
		// On a two-variant sealed the failed test pins the other variant,
		// so `if (r is Ok) ... else ...` sees an Err in the else branch.
		var other types.Type
		if s, ok := from.(*types.Sealed); ok && len(s.Variants) == 2 {
			if st, ok := target.(*types.Struct); ok && st.Sealed == s {
				other = s.Variants[1-st.Tag]
			}
		}
		if c.Not {
			whenFalse[v] = target
			if other != nil {
				whenTrue[v] = other
			}
		} else {
			whenTrue[v] = target
			if other != nil {
				whenFalse[v] = other
			}
		}
	}
	return
}

// patternTargetType returns the narrowed type a successful `is` test on a
// value of type from establishes, or nil.
func (f *fnCtx) patternTargetType(from types.Type, pat *ast.TypePat) types.Type {
	target := f.resolvePatternType(from, pat.Type)
	if target == nil {
		return nil
	}
	return target
}

func (f *fnCtx) ifExpr(e *ast.IfExpr, want types.Type) Expr {
	cond := f.checkExprTo(e.Cond, types.TBool)
	whenTrue, whenFalse := f.condFacts(e.Cond, cond)
	saved := f.saveNarrow()

	asValue := want != nil && !types.IsUnit(want)
	if e.Else == nil && asValue {
		f.errorf(e.Pos, "'if' used as a value needs an 'else' branch")
	}
	f.applyFacts(whenTrue)
	then := f.checkBlock(e.Then, want, asValue || want == nil)
	var thenState facts
	if !types.IsNever(then.Type) {
		thenState = f.saveNarrow()
	}
	f.restoreNarrow(saved)

	var els *Block
	f.applyFacts(whenFalse)
	var elseState facts
	if e.Else != nil {
		els = f.checkBlock(e.Else, want, asValue || want == nil)
	}
	if els == nil || !types.IsNever(els.Type) {
		elseState = f.saveNarrow()
	}
	f.restoreNarrow(saved)
	// Facts that hold on every path that continues survive the merge.
	f.narrow = mergeFacts(thenState, elseState)

	// Result type.
	var rt types.Type = types.TUnit
	if els == nil {
		if then.Value != nil {
			// value discarded
			then.Stmts = append(then.Stmts, &ExprStmt{X: then.Value})
			then.Value = nil
			then.Type = types.TUnit
		}
		return &If{exprBase{rt}, cond, then, nil}
	}
	tt, et := then.Type, els.Type
	switch {
	case types.IsNever(tt) && types.IsNever(et):
		rt = types.TNever
	case types.IsNever(tt):
		rt = et
	case types.IsNever(et):
		rt = tt
	case then.Value != nil && els.Value != nil:
		rt = f.unifyBranches(then, els, e.Pos)
	default:
		// statement form
		if then.Value != nil {
			then.Stmts = append(then.Stmts, &ExprStmt{X: then.Value})
			then.Value = nil
		}
		if els.Value != nil {
			els.Stmts = append(els.Stmts, &ExprStmt{X: els.Value})
			els.Value = nil
		}
		rt = types.TUnit
	}
	if types.IsUnit(rt) {
		then.Type, els.Type = types.TUnit, types.TUnit
	}
	return &If{exprBase{rt}, cond, then, els}
}

// unifyBranches reconciles two value blocks' types (T and T? unify to T?,
// a variant and its sealed type to the sealed type).
func (f *fnCtx) unifyBranches(a, b *Block, span source.Span) types.Type {
	ta, tb := a.Type, b.Type
	if types.Identical(ta, tb) {
		return ta
	}
	if f.assignableTo(ta, tb) {
		a.Value = f.coerce(a.Value, tb, span)
		a.Type = tb
		return tb
	}
	if f.assignableTo(tb, ta) {
		b.Value = f.coerce(b.Value, ta, span)
		b.Type = ta
		return ta
	}
	f.errorf(span, "branches have incompatible types '%s' and '%s'", ta, tb)
	return ta
}

func (f *fnCtx) isExpr(e *ast.IsExpr) Expr {
	x := f.checkExpr(e.X, nil)
	if types.IsInvalid(x.Type()) {
		return bad()
	}
	if e.Pat.HasArg {
		f.errorf(e.Pos, "destructuring patterns are only allowed in 'when' arms; use 'when' to bind fields")
	}
	test, _, _ := f.compilePattern(e.Pat, x, x.Type(), e.Pos)
	if test == nil {
		test = &BoolConst{exprBase{types.TBool}, true}
	}
	if e.Not {
		return &Unary{exprBase{types.TBool}, OpNot, test, e.Pos}
	}
	return test
}

// isPlaceSyntax reports whether an expression syntactically denotes a
// place (variable, field, index, dereference) rather than a temporary.
func isPlaceSyntax(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.NameExpr, *ast.SelfExpr:
		return true
	case *ast.MemberExpr:
		return !e.Safe && isPlaceSyntax(e.X)
	case *ast.IndexExpr:
		return true
	case *ast.UnaryExpr:
		return e.Op == lexer.Star
	}
	return false
}

// mergeFacts intersects the narrowing state of two joining paths; a nil
// state is a path that never continues (Never-typed) and imposes nothing.
func mergeFacts(a, b facts) facts {
	if a == nil && b == nil {
		return facts{}
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := facts{}
	for k, v := range a {
		if w, ok := b[k]; ok && types.Identical(v, w) {
			out[k] = v
		}
	}
	return out
}

// indexValue checks a list index of any integer width and widens it to i64.
func (f *fnCtx) indexValue(e ast.Expr) Expr {
	x := f.checkExpr(e, types.TI64)
	t := x.Type()
	if types.IsInvalid(t) {
		return x
	}
	if !types.IsInteger(t) {
		f.errorf(e.Span(), "index must be an integer, found '%s'", t)
		return bad()
	}
	if types.Identical(t, types.TI64) {
		return x
	}
	return &Cast{exprBase{types.TI64}, x}
}

// neverInstance reports whether this code is an instance of a generic
// function (or a lambda inside one) with a type parameter bound to Never:
// the instance in which `throws E` and `Result<R, E>` have been erased to
// plain values (D46), so a template-level `try` may find no Result.
func (f *fnCtx) neverInstance() bool {
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.fn == nil {
			continue
		}
		for _, t := range ctx.fn.subst {
			if types.IsNever(t) {
				return true
			}
		}
	}
	return false
}

// errPolyCall reports whether x calls something whose declaration says
// `throws E` for a type parameter E, in an instance where that E is Never
// and the call therefore does not throw: a `fun(..) throws E` parameter
// (Var.ErrPoly), or a generic function with such a signature.
func errPolyCall(x Expr) bool {
	switch c := x.(type) {
	case *CallIndirect:
		v, ok := c.Fn.(*VarRef)
		ft, isFn := v.Var.Type.(*types.Func)
		return ok && v.Var.ErrPoly && isFn && !ft.Effects.Throws
	case *Call:
		if c.Fn.Sig.Effects.Throws || c.Fn.tmpl == nil || c.Fn.tmpl.Sig == nil {
			return false
		}
		decl := c.Fn.tmpl.Sig.Effects
		return decl.Throws && decl.Error != nil && types.ContainsTypeParam(decl.Error)
	}
	return false
}
