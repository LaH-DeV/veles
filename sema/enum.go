package sema

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Enums (D57): `enum Ordering : i64 { Less = -1, Equal, Greater }` is a
// closed set of named values of one integer type. A value is its base
// integer at run time and nothing else; the names are compile-time facts
// the checker spells out as synthesized functions — `toString`, `values`,
// `fromValue`, `parse`, `compareTo` — the way tuple ordering is
// synthesized (tuple_order.go). An enum has no methods, fields or impls of
// its own: it is a set of values, and behaviour goes in functions that
// take one. It compares with its base type (`o < 0`, `o == -1`) but never
// converts to or from it implicitly: `x.value` reads the number,
// `E.fromValue(n)` looks a member up.

// declareEnum is declare() for an enum declaration.
func (c *Checker) declareEnum(m *Module, f *ast.File, d *ast.EnumDecl) {
	c.attrsOf(d.Attrs, "enum")
	e := &types.Enum{Name: d.Name.Name, Module: m.prefix(), Pub: d.Pub, Internal: d.Internal, Base: types.TI64, Decl: d}
	c.enumDecl[e] = &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
	c.enums = append(c.enums, e)
	c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Internal: d.Internal, Module: m, Span: d.Name.Pos, Type: e})
}

// resolveEnum fixes the base type and the members' values: an explicit
// value is an integer literal (optionally negative); an implicit one is the
// previous member's plus one, and the first member's is 0. Every value
// must fit the base type, and no two members may share one.
func (c *Checker) resolveEnum(e *types.Enum) {
	ctx := c.enumDecl[e]
	if ctx.resolved {
		return
	}
	ctx.resolved = true
	d := ctx.decl.(*ast.EnumDecl)
	if d.Base != nil {
		env := &typeEnv{module: ctx.module, file: ctx.file, tps: ctx.tps}
		bt := c.resolveType(env, d.Base)
		if b, ok := bt.(*types.Basic); ok && types.IsInteger(b) {
			e.Base = b
		} else if !types.IsInvalid(bt) {
			c.errorf(d.Base.Span(), "an enum counts in an integer type (i8..i64, u8..u64), not '%s' (D57)", bt)
		}
	}
	if len(d.Members) == 0 {
		c.errorf(d.Name.Pos, "enum '%s' has no members; a closed set needs at least one value (D57)", e.Name)
	}
	lo, hi := intRange(e.Base)
	next := big.NewInt(0)
	byValue := map[string]*ast.EnumMember{}
	byName := map[string]bool{}
	byFold := map[string]string{} // D205: a forgiving parse must never be ambiguous
	for i, md := range d.Members {
		if byName[md.Name.Name] {
			c.errorf(md.Name.Pos, "duplicate member '%s' in enum '%s'", md.Name.Name, e.Name)
			continue
		}
		if other, ok := byFold[caseFolded(md.Name.Name)]; ok {
			c.errorf(md.Name.Pos, "members '%s' and '%s' of enum '%s' are the same name in another spelling: 'parse(text, case: Case.Any)' could not tell them apart; rename one (D205)", other, md.Name.Name, e.Name)
		}
		byFold[caseFolded(md.Name.Name)] = md.Name.Name
		c.attrsOf(md.Attrs, "member")
		byName[md.Name.Name] = true
		v := next
		if md.Value != nil {
			if cv, ok := constInt(md.Value); ok {
				v = cv
			} else {
				c.errorf(md.Value.Span(), "an enum member's value is an integer literal (D57)")
			}
		}
		if v.Cmp(lo) < 0 || v.Cmp(hi) > 0 {
			c.errorf(md.Name.Pos, "value %s of '%s.%s' does not fit in '%s'", v, e.Name, md.Name.Name, e.Base)
		} else if prev, dup := byValue[v.String()]; dup {
			c.errorf(md.Name.Pos, "'%s.%s' and '%s.%s' share the value %s; every member of an enum is a distinct value (D57)", e.Name, prev.Name.Name, e.Name, md.Name.Name, v)
		} else {
			byValue[v.String()] = md
		}
		m := &types.EnumMember{Name: md.Name.Name, Doc: md.Doc, Index: i}
		m.Neg = v.Sign() < 0
		m.Value = new(big.Int).Abs(v).Uint64()
		e.Members = append(e.Members, m)
		next = new(big.Int).Add(v, big.NewInt(1))
	}
	if c.index != nil {
		c.refType(d.Name.Pos, e.Name, e, d.Name.Pos)
		for i, m := range e.Members {
			c.refEnumMember(d.Members[i].Name.Pos, e, m)
		}
	}
}

// intRange is the closed range of values an integer type holds.
func intRange(t *types.Basic) (lo, hi *big.Int) {
	bits := uint(types.BitSize(t))
	if types.IsSigned(t) {
		hi = new(big.Int).Lsh(big.NewInt(1), bits-1)
		lo = new(big.Int).Neg(hi)
		hi.Sub(hi, big.NewInt(1))
		return lo, hi
	}
	hi = new(big.Int).Lsh(big.NewInt(1), bits)
	hi.Sub(hi, big.NewInt(1))
	return big.NewInt(0), hi
}

// constInt evaluates an enum member's value: an integer literal, possibly
// negated.
func constInt(e ast.Expr) (*big.Int, bool) {
	switch x := e.(type) {
	case *ast.IntLit:
		text := x.Text
		for i := 0; i < len(text); i++ {
			if text[i] == '_' {
				text = text[:i] + text[i+1:]
				i--
			}
		}
		v, err := strconv.ParseUint(text, 0, 64)
		if err != nil {
			return nil, false
		}
		return new(big.Int).SetUint64(v), true
	case *ast.UnaryExpr:
		if x.Op != lexer.Minus {
			return nil, false
		}
		v, ok := constInt(x.X)
		if !ok {
			return nil, false
		}
		return v.Neg(v), true
	}
	return nil, false
}

// enumMemberSpan is where a member is declared.
func enumMemberSpan(e *types.Enum, m *types.EnumMember) source.Span {
	if d, ok := e.Decl.(*ast.EnumDecl); ok && m.Index < len(d.Members) {
		return d.Members[m.Index].Name.Pos
	}
	return source.Span{}
}

// memberText spells a member's value.
func memberText(m *types.EnumMember) string { return EnumMemberText(m) }

// EnumMemberText spells an enum member's value, for the editor.
func EnumMemberText(m *types.EnumMember) string {
	s := strconv.FormatUint(m.Value, 10)
	if m.Neg {
		s = "-" + s
	}
	return s
}

// refEnumMember records a use of `E.Member` for the editor.
func (c *Checker) refEnumMember(span source.Span, e *types.Enum, m *types.EnumMember) {
	if c.index == nil || !span.IsValid() {
		return
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: enumMemberSpan(e, m), Kind: "val", Name: m.Name, Type: e,
		Detail: e.Name + "." + m.Name + " = " + memberText(m), Doc: m.Doc, Where: enumHead(e)})
}

// enumHead is the declaration line of an enum with every implicit word
// spelled out.
func enumHead(e *types.Enum) string {
	head := visPrefix(e.Pub, e.Internal, false) + "enum " + e.Name
	return head + " : " + e.Base.Name
}

// enumShape spells an enum out for a hover: its head and members.
func enumShape(e *types.Enum) string {
	s := enumHead(e) + " {\n"
	for _, m := range e.Members {
		s += "  " + m.Name + " = " + memberText(m) + "\n"
	}
	return s + "}"
}

// enumConst is the HIR constant for a member: the value, typed as the enum.
func enumConst(e *types.Enum, m *types.EnumMember) Expr {
	return &IntConst{exprBase{e}, m.Value, m.Neg}
}

// enumMember checks `E.Name` in value position.
func (f *fnCtx) enumMember(e *types.Enum, name ast.Ident) Expr {
	m := e.MemberByName(name.Name)
	if m == nil {
		switch name.Name {
		case "values", "fromValue", "parse":
			f.errorf(name.Pos, "'%s.%s' is a function; call it: '%s.%s(...)'", e.Name, name.Name, e.Name, name.Name)
		default:
			f.errorf(name.Pos, "enum '%s' has no member '%s'%s", e.Name, name.Name, enumSuggest(e, name.Name))
		}
		return bad()
	}
	f.c.refEnumMember(name.Pos, e, m)
	return enumConst(e, m)
}

// enumSuggest lists the members for an unknown-member message.
func enumSuggest(e *types.Enum, name string) string {
	if len(e.Members) == 0 {
		return ""
	}
	s := "; its members are "
	for i, m := range e.Members {
		if i > 0 {
			s += ", "
		}
		s += m.Name
	}
	return s
}

// orderingType is the prelude's Ordering enum, the result of every
// `compareTo`; i64 when the prelude is not loaded (tests over bare files).
func (c *Checker) orderingType() types.Type {
	if c.ordering != nil {
		return c.ordering
	}
	if sym := c.universe.LookupLocal("Ordering"); sym != nil && sym.Kind == SymType {
		if e, ok := sym.Type.(*types.Enum); ok {
			c.resolveEnum(e)
			c.ordering = e
			return e
		}
	}
	return types.TI64
}

// orderingConst is Less (-1), Equal (0) or Greater (1) as a constant of
// the Ordering type (or of i64 without the prelude).
func (c *Checker) orderingConst(sign int) Expr {
	t := c.orderingType()
	if sign < 0 {
		return &IntConst{exprBase{t}, 1, true}
	}
	return &IntConst{exprBase{t}, uint64(sign), false}
}

// enumFunc returns a synthesized function of an enum, building it on first
// use in the round: "toString", "compareTo", "values", "fromValue" or
// "parse".
func (c *Checker) enumFunc(e *types.Enum, op string) *Func {
	c.resolveEnum(e)
	if c.enumFns == nil {
		c.enumFns = map[string]*Func{}
	}
	key := types.Key(e) + "." + op
	if fn, ok := c.enumFns[key]; ok {
		return fn
	}
	newVar := func(name string, t types.Type) *Var {
		c.nextVar++
		return &Var{Name: name, Type: t, ID: c.nextVar, IsParam: true}
	}
	fn := &Func{Name: mangleName("veles.enum." + op + "<" + types.Key(e) + ">"), Display: e.Name + "." + op}
	eq := func(l, r Expr) Expr { return &Binary{exprBase{types.TBool}, OpEq, l, r, source.Span{}} }
	value := func(t types.Type, x Expr) *Block { return &Block{Value: x, Type: t} }
	// chain builds `if tests[0] then values[0] else if ... else last` as
	// one expression of type t
	chain := func(t types.Type, tests []Expr, values []Expr, last Expr) Expr {
		out := last
		for i := len(tests) - 1; i >= 0; i-- {
			out = &If{exprBase{t}, tests[i], value(t, values[i]), value(t, out)}
		}
		return out
	}
	switch op {
	case "toString":
		x := newVar("$x", e)
		fn.Sig = &types.Func{Params: []types.Param{{Name: "x", Type: e}}, Ret: types.TString}
		fn.Params = []*Var{x}
		var tests, names []Expr
		var last Expr = &StringConst{exprBase{types.TString}, ""}
		for i, m := range e.Members {
			name := &StringConst{exprBase{types.TString}, m.Name}
			if i == len(e.Members)-1 {
				last = name
				break
			}
			tests = append(tests, eq(ref(x), enumConst(e, m)))
			names = append(names, name)
		}
		fn.Body = &Block{Stmts: []Stmt{&Return{Value: chain(types.TString, tests, names, last)}}, Type: types.TNever}
	case "compareTo":
		ord := c.orderingType()
		a, b := newVar("$a", e), newVar("$b", e)
		fn.Sig = &types.Func{Params: []types.Param{{Name: "a", Type: e}, {Name: "b", Type: e}}, Ret: ord}
		fn.Params = []*Var{a, b}
		lt := &Binary{exprBase{types.TBool}, OpLt, ref(a), ref(b), source.Span{}}
		gt := &Binary{exprBase{types.TBool}, OpGt, ref(a), ref(b), source.Span{}}
		body := chain(ord, []Expr{lt, gt}, []Expr{c.orderingConst(-1), c.orderingConst(1)}, c.orderingConst(0))
		fn.Body = &Block{Stmts: []Stmt{&Return{Value: body}}, Type: types.TNever}
	case "values":
		lt := &types.List{Elem: e}
		fn.Sig = &types.Func{Ret: lt}
		lit := &ListLit{exprBase{lt}, nil}
		for _, m := range e.Members {
			lit.Elems = append(lit.Elems, enumConst(e, m))
		}
		fn.Body = &Block{Stmts: []Stmt{&Return{Value: lit}}, Type: types.TNever}
	case "fromValue":
		nt := &types.Nullable{Elem: e}
		n := newVar("$n", e.Base)
		fn.Sig = &types.Func{Params: []types.Param{{Name: "value", Type: e.Base}}, Ret: nt}
		fn.Params = []*Var{n}
		var tests, values []Expr
		for _, m := range e.Members {
			tests = append(tests, eq(ref(n), &IntConst{exprBase{e.Base}, m.Value, m.Neg}))
			values = append(values, &SomeWrap{exprBase{nt}, enumConst(e, m)})
		}
		fn.Body = &Block{Stmts: []Stmt{&Return{Value: chain(nt, tests, values, &NullConst{exprBase{nt}})}}, Type: types.TNever}
	case "parse":
		nt := &types.Nullable{Elem: e}
		s := newVar("$s", types.TString)
		fn.Sig = &types.Func{Params: []types.Param{{Name: "s", Type: types.TString}}, Ret: nt}
		fn.Params = []*Var{s}
		var tests, values []Expr
		for _, m := range e.Members {
			tests = append(tests, eq(ref(s), &StringConst{exprBase{types.TString}, m.Name}))
			values = append(values, &SomeWrap{exprBase{nt}, enumConst(e, m)})
		}
		fn.Body = &Block{Stmts: []Stmt{&Return{Value: chain(nt, tests, values, &NullConst{exprBase{nt}})}}, Type: types.TNever}
	default:
		panic("enumFunc: unknown operation " + op)
	}
	fn.checked = true
	fn.Const = true // plain HIR over constants and comparisons: the compiler may run it (D113)
	c.funcs = append(c.funcs, fn)
	c.enumFns[key] = fn
	return fn
}

// enumOps are the custom operations of an enum for the backend: its text
// and its order. Equality and hashing are those of the base integer.
func (c *Checker) enumOps(e *types.Enum) *CustomOps {
	return &CustomOps{ToString: c.enumFunc(e, "toString"), Compare: c.enumFunc(e, "compareTo")}
}

// enumStaticCall checks `E.values()`, `E.fromValue(n)` and `E.parse(s)`.
func (f *fnCtx) enumStaticCall(e *types.Enum, callee *ast.MemberExpr, call *ast.CallExpr) Expr {
	name := callee.Name.Name
	switch name {
	case "values", "fromValue", "parse":
	default:
		if m := e.MemberByName(name); m != nil {
			f.errorf(callee.Name.Pos, "'%s.%s' is a value, not a function", e.Name, name)
		} else {
			f.errorf(callee.Name.Pos, "enum '%s' has no function '%s'; an enum has 'values()', 'fromValue(n)', 'parse(s)' and 'decode(from)' (D57)", e.Name, name)
		}
		f.checkArgsLoosely(call.Args)
		return bad()
	}
	f.c.refBuiltin(callee.Name.Pos, e, name)
	if name == "parse" {
		if x := f.enumParse(e, callee, call); x != nil {
			return x
		}
	}
	fn := f.c.enumFunc(e, name)
	bound, ok := f.bindArgs(fn.Sig.Params, call.Args, "'"+e.Name+"."+name+"'", call.Pos)
	if !ok {
		return bad()
	}
	var args []Expr
	for i, p := range fn.Sig.Params {
		args = append(args, f.checkExprTo(bound[i], p.Type))
	}
	return &Call{exprBase: exprBase{fn.Sig.Ret}, Fn: fn, Args: args}
}

// enumMethodCall checks `x.toString()` and `x.compareTo(y)` on an enum
// value; nil when name is neither.
func (f *fnCtx) enumMethodCall(recv Expr, e *types.Enum, callee *ast.MemberExpr, call *ast.CallExpr) Expr {
	name := callee.Name.Name
	if m := enumPredicate(e, name); m != nil {
		// `os.isWindows()`: true for that member alone (D206)
		if len(call.Args) != 0 {
			f.errorf(call.Pos, "'%s' takes no arguments: it says whether the value is %s.%s (D206)", name, e.Name, m.Name)
			f.checkArgsLoosely(call.Args)
			return bad()
		}
		f.c.refBuiltin(callee.Name.Pos, e, name)
		return &Binary{exprBase{types.TBool}, OpEq, recv, enumConst(e, m), call.Pos}
	}
	if name == "toString" && len(call.Args) == 1 {
		return f.enumToStringCase(recv, e, callee, call) // D205
	}
	switch name {
	case "toString", "compareTo":
	default:
		return nil
	}
	f.c.refBuiltin(callee.Name.Pos, e, name)
	fn := f.c.enumFunc(e, name)
	bound, ok := f.bindArgs(fn.Sig.Params[1:], call.Args, "'"+name+"'", call.Pos)
	if !ok {
		return bad()
	}
	args := []Expr{recv}
	for i, p := range fn.Sig.Params[1:] {
		args = append(args, f.checkExprTo(bound[i], p.Type))
	}
	return &Call{exprBase: exprBase{fn.Sig.Ret}, Fn: fn, Args: args}
}

// enumField checks `x.value` on an enum: the base integer.
func (f *fnCtx) enumField(x Expr, e *types.Enum, name ast.Ident) Expr {
	if name.Name != "value" {
		switch name.Name {
		case "toString", "compareTo":
			f.errorf(name.Pos, "'%s' is a function; call it: '.%s(...)'", name.Name, name.Name)
		default:
			f.errorf(name.Pos, "enum '%s' has no field '%s'; an enum value has '.value' (its %s) and 'toString()' (D57)", e.Name, name.Name, e.Base)
		}
		return bad()
	}
	if f.c.index != nil {
		f.c.index.Refs = append(f.c.index.Refs, Ref{Span: name.Pos, Kind: "field", Name: "value", Type: e.Base,
			Detail: e.Name + ".value: " + e.Base.Name, Doc: "The member's number, as declared or counted (D57).", Where: enumHead(e)})
	}
	return &Cast{exprBase{e.Base}, x}
}

// enumCompare lowers a comparison with an enum on one or both sides: two
// values of the enum compare as their base integers, and an enum compares
// with a value of its base type (`o < 0`, `o == -1`). nil when the pair is
// not that shape (the caller reports).
func (f *fnCtx) enumCompare(op BinOp, l, r Expr, span source.Span) Expr {
	le, lok := l.Type().(*types.Enum)
	re, rok := r.Type().(*types.Enum)
	switch {
	case lok && rok:
		if le != re {
			return nil
		}
	case lok:
		if !types.Identical(r.Type(), le.Base) {
			return nil
		}
		l = &Cast{exprBase{le.Base}, l}
	case rok:
		if !types.Identical(l.Type(), re.Base) {
			return nil
		}
		r = &Cast{exprBase{re.Base}, r}
	default:
		return nil
	}
	return &Binary{exprBase{types.TBool}, op, l, r, span}
}

// enumExample names a member for a message: the first one.
func enumExample(e *types.Enum) string {
	if len(e.Members) == 0 {
		return e.Name + ".<member>"
	}
	return e.Name + "." + e.Members[0].Name
}

// isEnumeratedTrait recognises the prelude's `Enumerated` (D205), the one
// trait an enum may implement.
func isEnumeratedTrait(trait *types.Trait) bool {
	return trait != nil && trait.Name == "Enumerated" && trait.Module == "std.prelude"
}

// enumOverride is the method an `implement Enumerated for E` writes in place
// of the compiler's (`parse`, `toString`), with the substitution that calls
// it; nil when the enum writes none.
func (f *fnCtx) enumOverride(e *types.Enum, name string) (*FuncTemplate, map[*types.TypeParam]types.Type) {
	trait := f.c.traitNamed("Enumerated")
	if trait == nil {
		return nil, nil
	}
	for _, impl := range f.c.impls[trait] {
		if !types.Identical(impl.Target, e) {
			continue
		}
		if t, ok := impl.Methods[name]; ok && t.Decl != nil {
			return t, map[*types.TypeParam]types.Type{selfParamOf(trait): e}
		}
	}
	return nil, nil
}

// enumPredicate is the member a predicate name asks about: `isWindows` on an
// enum with a member `Windows` (D206); nil for any other name.
func enumPredicate(e *types.Enum, name string) *types.EnumMember {
	if len(name) <= 2 || !strings.HasPrefix(name, "is") {
		return nil
	}
	return e.MemberByName(name[2:])
}

// enumParse checks `E.parse(text, case: c)` (D205): with a case it reads the
// member `text` names in that spelling; with none it is nil, and the
// compiler's exact `parse` is called. An `implement Enumerated` that writes
// `parse` is called instead, with or without a case.
func (f *fnCtx) enumParse(e *types.Enum, callee *ast.MemberExpr, call *ast.CallExpr) Expr {
	caseT := f.c.preludeType("Case")
	if caseT == nil {
		return nil
	}
	if t, subst := f.enumOverride(e, "parse"); t != nil {
		f.c.refFunc(callee.Name.Pos, t)
		args := call.Args
		if len(args) == 1 && len(t.Decl.Params) == 2 && t.Decl.Params[1].Default == nil {
			// the trait's default, `case: Case.AsWritten`, where the
			// override does not repeat it
			b := &synth{sp: call.Pos}
			asWritten := &ast.MemberExpr{X: &ast.TypeExpr{Type: &ast.ResolvedType{T: caseT, Pos: b.sp}, Pos: b.sp}, Name: ast.Ident{Name: "AsWritten", Pos: b.sp}, Pos: b.sp}
			args = append(append([]ast.Arg(nil), args...), ast.Arg{Value: asWritten})
		}
		return f.callTemplateRecv(t, subst, nil, nil, args, call.Pos, nil)
	}
	params := []types.Param{{Name: "text", Type: types.TString}, {Name: "case", Type: caseT, HasDefault: true}}
	bound, ok := f.bindArgs(params, call.Args, "'"+e.Name+".parse'", call.Pos)
	if !ok {
		return bad()
	}
	if bound[1] == nil {
		return nil // exact: the compiler's own parse
	}
	b := &synth{sp: call.Pos}
	var names []ast.Expr
	for _, m := range e.Members {
		names = append(names, b.str(m.Name))
	}
	values := &ast.CallExpr{Fun: &ast.MemberExpr{X: &ast.TypeExpr{Type: &ast.ResolvedType{T: e, Pos: b.sp}, Pos: b.sp}, Name: ast.Ident{Name: "values", Pos: b.sp}, Pos: b.sp}, Pos: b.sp}
	pick := b.call(b.prelude("enumPick"), values, &ast.ListLit{Elems: names, Pos: b.sp}, bound[0], bound[1])
	return f.checkExpr(pick, &types.Nullable{Elem: e})
}

// enumToStringCase checks `x.toString(case: c)` (D205): the member's name
// spelled in `c`, or what an `implement Enumerated` writes for it.
func (f *fnCtx) enumToStringCase(recv Expr, e *types.Enum, callee *ast.MemberExpr, call *ast.CallExpr) Expr {
	if t, subst := f.enumOverride(e, "toString"); t != nil {
		f.c.refFunc(callee.Name.Pos, t)
		// a method: the receiver is passed as a method's is (by address)
		return f.callMethod(t, subst, nil, recv, false, callee, call, nil)
	}
	caseT := f.c.preludeType("Case")
	if caseT == nil {
		return nil
	}
	f.c.refBuiltin(callee.Name.Pos, e, "toString")
	bound, ok := f.bindArgs([]types.Param{{Name: "case", Type: caseT}}, call.Args, "'toString'", call.Pos)
	if !ok {
		return bad()
	}
	fn := f.c.enumFunc(e, "toString")
	name := &Call{exprBase: exprBase{types.TString}, Fn: fn, Args: []Expr{recv}}
	b := &synth{sp: call.Pos}
	return f.checkExpr(b.call(b.prelude("caseSpelled"), &ast.CheckedExpr{X: name, Pos: b.sp}, bound[0]), types.TString)
}

// caseWords splits a name into lower-case words as the prelude's `Case`
// does (D205): at a lower case letter or digit followed by a capital, before
// the last capital of a run followed by a lower case letter, and at `-`,
// `_` and spaces.
func caseWords(name string) []string {
	var words []string
	start := -1
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '-' || b == '_' || b == ' ' {
			if start >= 0 {
				words = append(words, strings.ToLower(name[start:i]))
			}
			start = -1
			continue
		}
		if start >= 0 && b >= 'A' && b <= 'Z' {
			before := name[i-1]
			var after byte
			if i+1 < len(name) {
				after = name[i+1]
			}
			lower := func(c byte) bool { return c >= 'a' && c <= 'z' }
			if lower(before) || before >= '0' && before <= '9' || before >= 'A' && before <= 'Z' && lower(after) {
				words = append(words, strings.ToLower(name[start:i]))
				start = i
			}
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, strings.ToLower(name[start:]))
	}
	return words
}

// caseFolded is the form two spellings of one name share under `Case.Any`.
func caseFolded(name string) string { return strings.Join(caseWords(name), "") }
