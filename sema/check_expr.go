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
	if conv := f.convert(x, want); conv != nil {
		return conv
	}
	if types.IsUnit(want) {
		return x // statement position: any value may be discarded
	}
	f.errorf(span, "type mismatch: expected '%s', found '%s'", want, have)
	return x
}

func (f *fnCtx) convert(x Expr, want types.Type) Expr {
	have := x.Type()
	switch w := want.(type) {
	case *types.Nullable:
		if types.Identical(have, w.Elem) {
			return &SomeWrap{exprBase{want}, x}
		}
		if inner := f.convert(x, w.Elem); inner != nil {
			return &SomeWrap{exprBase{want}, inner}
		}
	case *types.Sealed:
		if st, ok := have.(*types.Struct); ok && st.Sealed == w {
			return &MakeVariant{exprBase{want}, w, st, x}
		}
	case *types.Pointer:
		// Inside unsafe a GC pointer may be handed to C as a raw pointer: the
		// collector is non-moving (§5), so the address is stable.
		if hp, ok := have.(*types.Pointer); ok && w.Raw && !hp.Raw && types.Identical(hp.Elem, w.Elem) && f.unsafe > 0 {
			return &Cast{exprBase{want}, x}
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
	case *types.ErrorUnion:
		for _, m := range types.UnionMembers(have) {
			if types.UnionIndex(w, m) < 0 {
				return false
			}
		}
		return true
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
		f.errorf(e.Pos, "character literals are not part of the language; strings are byte-indexed (D18) — use a one-character string")
		return bad()
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
		if f.selfVar == nil {
			f.errorf(e.Pos, "'self' outside of a method")
			return bad()
		}
		if f.selfMut {
			return &Deref{exprBase{f.selfVar.Type.(*types.Pointer).Elem}, &VarRef{exprBase{f.selfVar.Type}, f.selfVar}}
		}
		return f.narrowedRef(f.selfVar)
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
		f.errorf(e.Pos, "Map is not implemented yet in the bootstrap compiler")
		return bad()
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
		f.errorf(e.Pos, "lambdas and closures are not implemented yet in the bootstrap compiler (D32)")
		return bad()
	case *ast.AwaitExpr:
		f.errorf(e.Pos, "'await' and the concurrency runtime are not implemented yet in the bootstrap compiler (build plan stage 4)")
		return bad()
	case *ast.GatherExpr:
		f.errorf(e.Pos, "'gather' is not implemented yet in the bootstrap compiler (build plan stage 4)")
		return bad()
	case *ast.RaceExpr:
		f.errorf(e.Pos, "'race' is not implemented yet in the bootstrap compiler (build plan stage 4)")
		return bad()
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
		t = types.TI32
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
	if _, ok := t.(*types.Trait); ok {
		f.errorf(span, "cannot interpolate a trait object")
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
		f.errorf(e.Pos, "unknown name '%s'", e.Name)
		return bad()
	}
	switch sym.Kind {
	case SymLocal:
		return f.narrowedRef(sym.Var)
	case SymGlobal:
		v := f.globalVar(sym.Global)
		return &VarRef{exprBase{v.Type}, v}
	case SymFunc:
		f.errorf(e.Pos, "function '%s' used as a value; function values are not supported yet in the bootstrap compiler", e.Name)
		return bad()
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
	ref := Expr(&VarRef{exprBase{v.Type}, v})
	to, ok := f.narrow[v]
	if !ok {
		return ref
	}
	return narrowExpr(ref, to)
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
			st, ok := to.(*types.Struct)
			if !ok || st.Sealed != t {
				return x
			}
			return &VariantCast{exprBase{st}, x, st}
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

func (f *fnCtx) memberExpr(e *ast.MemberExpr, want types.Type) Expr {
	if e.X == nil {
		// `.some(x)`, `.none` — resolved against the expected type
		return f.leadingDot(e, nil, want)
	}
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
			}
		}
	}
	x := f.checkExpr(e.X, nil)
	return f.fieldAccess(x, e, want)
}

func (f *fnCtx) symbolValue(sym *Symbol, span source.Span, want types.Type) Expr {
	switch sym.Kind {
	case SymGlobal:
		v := f.globalVar(sym.Global)
		return &VarRef{exprBase{v.Type}, v}
	case SymFunc:
		f.errorf(span, "function values are not supported yet in the bootstrap compiler")
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
	case *types.Nullable:
		f.errorf(span, "value of type '%s' may be null; use '?.', '?:' or check for null first (D5)", tt)
		return bad()
	case *types.Sealed:
		f.errorf(span, "'%s' is a sealed trait; match on its variants with 'when' before accessing '%s' (D13)", tt, name.Name)
		return bad()
	}
	if types.IsInvalid(t) {
		return bad()
	}
	f.errorf(span, "type '%s' has no field '%s'", t, name.Name)
	return bad()
}

func (f *fnCtx) indexExpr(e *ast.IndexExpr) Expr {
	x := f.checkExpr(e.X, nil)
	switch t := x.Type().(type) {
	case *types.List:
		idx := f.indexValue(e.Index)
		return &Builtin{exprBase{t.Elem}, "list.get", []Expr{x, idx}, e.Pos}
	case *types.Basic:
		if t.Kind == types.String {
			f.errorf(e.Pos, "strings are not indexable with '[]'; use 's.bytes[i]' (§4b) — not yet implemented")
			return bad()
		}
	}
	if !types.IsInvalid(x.Type()) {
		f.errorf(e.Pos, "cannot index a value of type '%s'", x.Type())
	}
	return bad()
}

// ---------------------------------------------------------------------------
// variant constructors: Some / None / Ok / Err and `.name`

func (f *fnCtx) leadingDot(e *ast.MemberExpr, args []ast.Arg, want types.Type) Expr {
	switch e.Name.Name {
	case "some", "none", "ok", "err":
		return f.variantCtor(strings.ToUpper(e.Name.Name[:1])+e.Name.Name[1:], args, want, e.Pos)
	}
	s, ok := want.(*types.Sealed)
	if !ok {
		if n, ok := want.(*types.Nullable); ok {
			if s2, ok := n.Elem.(*types.Sealed); ok {
				s = s2
			}
		}
	}
	if s == nil {
		f.errorf(e.Pos, "cannot resolve '.%s': no sealed type is expected here", e.Name.Name)
		return bad()
	}
	v := s.VariantByName(e.Name.Name)
	if v == nil {
		f.errorf(e.Name.Pos, "'%s' has no variant '%s'", s.Name, e.Name.Name)
		return bad()
	}
	if args == nil && len(v.Fields) == 0 {
		return f.variantValue(v, want, e.Pos)
	}
	return f.constructStruct(v, args, e.Pos)
}

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
		if args != nil && len(args) != 0 {
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
	case lexer.Amp:
		// D10: address of a local (heap-promoted); D50: always a GC pointer.
		// The address of a temporary boxes the value.
		if !isPlaceSyntax(e.X) {
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
	case OpLt, OpLe, OpGt, OpGe:
		if !types.IsNumeric(t) && !types.IsString(t) {
			f.errorf(span, "operator '%s' is not defined for '%s'", op, t)
			return bad()
		}
		return &Binary{exprBase{types.TBool}, op, l, r, span}
	}
	f.errorf(span, "unsupported operator")
	return bad()
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
			if v := varOf(other, f); v != nil {
				if _, declaredNullable := v.Type.(*types.Nullable); declaredNullable {
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
		l = f.checkExprTo(e.L, r.Type())
	} else {
		l = f.checkExpr(e.L, nil)
		r = f.checkExprTo(e.R, l.Type())
	}
	t := l.Type()
	if types.IsInvalid(t) || types.IsInvalid(r.Type()) {
		return bad()
	}
	if !f.comparable(t) {
		f.errorf(e.Pos, "values of type '%s' cannot be compared with '%s'", t, op)
		return bad()
	}
	return &Binary{exprBase{types.TBool}, op, l, r, e.Pos}
}

func (f *fnCtx) comparable(t types.Type) bool {
	switch t := t.(type) {
	case *types.Basic:
		return t.Kind != types.Unit && t.Kind != types.Never && t.Kind != types.Invalid
	case *types.Pointer:
		return true
	case *types.Nullable:
		return f.comparable(t.Elem)
	case *types.Struct:
		for _, fld := range t.Fields {
			if !f.comparable(fld.Type) {
				return false
			}
		}
		return true
	case *types.Sealed:
		for _, v := range t.Variants {
			if !f.comparable(v) {
				return false
			}
		}
		return true
	case *types.Tuple:
		for _, e := range t.Elems {
			if !f.comparable(e) {
				return false
			}
		}
		return true
	}
	return false
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
	if lt, ok := numericHint(want).(*types.List); ok {
		elem = lt.Elem
		mutable = lt.Mutable
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
		f.errorf(e.Pos, "cannot infer the element type of an empty list; annotate it, e.g. 'val xs: List<i32> = []' (D25)")
		return bad()
	}
	lit.T = &types.List{Elem: elem, Mutable: mutable}
	return lit
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
	x := f.checkExpr(e.X, nil)
	rs, ok := x.Type().(*types.Sealed)
	if !ok || !isResultType(rs) {
		if !types.IsInvalid(x.Type()) {
			f.errorf(e.Pos, "'try' needs a Result (a call to a 'throws' function), found '%s'", x.Type())
		}
		return bad()
	}
	okT, errT := rs.TypeArgs[0], rs.TypeArgs[1]
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

type facts map[*Var]types.Type

func (f *fnCtx) saveNarrow() facts {
	out := facts{}
	for k, v := range f.narrow {
		out[k] = v
	}
	return out
}

func (f *fnCtx) restoreNarrow(saved facts) {
	f.narrow = map[*Var]types.Type{}
	for k, v := range saved {
		f.narrow[k] = v
	}
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
	return sym.Var
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
			v := varOf(other, f)
			if v == nil {
				return
			}
			cur := f.currentType(v)
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
	case *ast.IsExpr:
		v := varOf(c.X, f)
		if v == nil {
			return
		}
		target := f.patternTargetType(f.currentType(v), c.Pat)
		if target == nil {
			return
		}
		if c.Not {
			whenFalse[v] = target
		} else {
			whenTrue[v] = target
		}
	}
	return
}

func (f *fnCtx) currentType(v *Var) types.Type {
	if t, ok := f.narrow[v]; ok {
		return t
	}
	return v.Type
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
		return e.X != nil && !e.Safe && isPlaceSyntax(e.X)
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
