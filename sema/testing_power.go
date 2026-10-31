package sema

import (
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// What a failed `expect` or `assert` shows under its first line (D78): the
// values its condition was made of, so `expect(path.within(base, file))`
// says what `base` and `file` were. Each part is evaluated once, where the
// condition evaluates it, into a hidden variable; the report reads those.
//
//	main.vs:7:5: expect(path.within(base, file))
//	    base: "/srv"
//	    file: "/srv/../etc"
//
// `&&` and `||` report the side that decided: the first false operand of an
// `&&`, both operands of an `||`. `!` reports its operand. A comparison shows
// both sides, `x in c` the value and the container, a call its receiver and
// arguments. Anything else is reported by its text and the value it had.

// part is one line of a failure: a label and the value shown after it.
type part struct {
	label string
	value Expr
	// block, for a part with no label, is its text given the column the
	// values start at: a diff, whose caret points into the line above
	block func(column int) Expr
}

// binding is a hidden variable the condition evaluates a part into.
type binding struct {
	v    *Var
	init Expr
}

// logic kinds of a `&&` / `||` for the outcome it must have: in a chain
// each operand must hold (`a && b` must be true), or one is enough.
const (
	logicNone = iota
	logicEach
	logicAny
)

// logicKind is the kind of a, peeling `!`, when it must evaluate to want.
func logicKind(a ast.Expr, want bool) int {
	for {
		u, ok := a.(*ast.UnaryExpr)
		if !ok || u.Op != lexer.Bang {
			break
		}
		a, want = u.X, !want
	}
	b, ok := a.(*ast.BinaryExpr)
	if !ok || (b.Op != lexer.AndAnd && b.Op != lexer.OrOr) {
		return logicNone
	}
	if (b.Op == lexer.AndAnd) == want {
		return logicEach
	}
	return logicAny
}

// failing checks a condition that must evaluate to want and returns the code
// that runs miss when it does not and pass when it does (nil: nothing). nil
// when the condition does not check.
//
// The operands of a `&&`/`||` are followed only within a chain of one kind:
// otherwise a continuation that checks the rest of the chain would be called
// from more than one place, and the rest checked twice. An operand of the
// other kind is reported as a whole.
func (f *fnCtx) failing(a ast.Expr, want, top bool, kind int, miss func([]part) Expr, pass func() Expr) Expr {
	if u, ok := a.(*ast.UnaryExpr); ok && u.Op == lexer.Bang {
		return f.failing(u.X, !want, top, kind, miss, pass)
	}
	k := logicKind(a, want)
	if k == logicNone || (kind != logicNone && k != kind) {
		return f.leaf(a, want, top, miss, pass)
	}
	b := a.(*ast.BinaryExpr)
	// the right operand is checked where it runs: after the left one, with
	// what the left one proved (`x != null && x.len() > 0`), as `&&` does
	right := func(miss func([]part) Expr) Expr {
		saved := f.saveNarrow()
		whenTrue, whenFalse := f.condFacts(b.L, nil)
		if b.Op == lexer.AndAnd {
			f.applyFacts(whenTrue)
		} else {
			f.applyFacts(whenFalse)
		}
		x := f.failing(b.R, want, false, k, miss, pass)
		f.restoreNarrow(saved)
		return x
	}
	if k == logicEach {
		// the first operand that does not hold fails it
		return f.failing(b.L, want, false, k, miss, func() Expr { return right(miss) })
	}
	// it fails when no operand holds: both are shown
	return f.failing(b.L, want, false, k, func(left []part) Expr {
		return right(func(parts []part) Expr {
			return miss(append(append([]part{}, left...), parts...))
		})
	}, pass)
}

// leaf checks a condition that is not followed further.
func (f *fnCtx) leaf(a ast.Expr, want, top bool, miss func([]part) Expr, pass func() Expr) Expr {
	f.pushScope()
	defer f.popScope()
	cond, lets, parts := f.operand(a, top)
	if cond == nil || types.IsInvalid(cond.Type()) {
		// the rest of the condition is still checked, for its errors
		if pass != nil {
			pass()
		}
		miss(nil)
		return nil
	}
	if len(parts) == 0 && !top {
		value := "false"
		if !want {
			value = "true"
		}
		if label := partLabel(a); label != "" {
			parts = []part{{label: label, value: strConst(value)}}
		}
	}
	if !want {
		cond = &Unary{exprBase{types.TBool}, OpNot, cond, a.Span()}
	}
	// either continuation may check the rest of the condition, which may
	// not check
	onMiss := miss(parts)
	if onMiss == nil {
		return nil
	}
	var x Expr
	if pass == nil {
		x = f.unless(cond, onMiss, a.Span())
	} else {
		onPass := pass()
		if onPass == nil {
			return nil
		}
		x = &If{exprBase{types.TUnit}, cond, f.valueBlock(onPass), f.valueBlock(onMiss)}
	}
	for i := len(lets) - 1; i >= 0; i-- {
		x = &Let{exprBase{types.TUnit}, lets[i].v, lets[i].init, x}
	}
	return x
}

// operand checks one condition and says what a failure shows of it.
func (f *fnCtx) operand(a ast.Expr, top bool) (Expr, []binding, []part) {
	switch e := a.(type) {
	case *ast.BinaryExpr:
		if isComparison(e.Op) {
			var names []string
			if top {
				names = []string{"left", "right"}
			}
			return f.comparisonParts(e, names)
		}
	case *ast.InExpr:
		return f.inParts(e)
	case *ast.CallExpr:
		c := f.checkExprTo(a, types.TBool)
		if types.IsInvalid(c.Type()) {
			return c, nil, nil
		}
		lets, parts := f.callParts(e, c)
		return c, lets, parts
	}
	return f.checkExprTo(a, types.TBool), nil, nil
}

// comparisonParts is `l OP r` with each side evaluated once, in source order.
// The whole condition's sides are labelled by names (`left` and `right`;
// `expected` and `actual` for expectEqual), with a literal shown as written;
// inside `&&`/`||` a side is labelled by its text and a literal, which says
// nothing new, is left out.
func (f *fnCtx) comparisonParts(b *ast.BinaryExpr, names []string) (Expr, []binding, []part) {
	// a literal side takes its type from the other, as it does in the
	// comparison itself: `expect(port == 80)` with a u16 port
	var l, r Expr
	if isLiteral(b.L) && !isLiteral(b.R) {
		r = f.checkExpr(b.R, nil)
		l = f.checkExpr(b.L, r.Type())
	} else {
		l = f.checkExpr(b.L, nil)
		r = f.checkExpr(b.R, l.Type())
	}
	if types.IsInvalid(l.Type()) || types.IsInvalid(r.Type()) {
		return bad(), nil, nil
	}
	lv, ln := f.hiddenName("left", l, b.L.Span())
	rv, rn := f.hiddenName("right", r, b.R.Span())
	c := f.checkExprTo(&ast.BinaryExpr{Op: b.Op, L: ln, R: rn, Pos: b.Pos}, types.TBool)
	if types.IsInvalid(c.Type()) {
		return c, nil, nil
	}
	// a side that is a call shows its arguments too, after both sides:
	// `expect(parsePort(input) > 0)` says what `input` was
	var lets []binding
	var args []part
	call := func(x ast.Expr, checked Expr) {
		if e, ok := x.(*ast.CallExpr); ok {
			moved, shown := f.callParts(e, checked)
			lets = append(lets, moved...)
			args = append(args, shown...)
		}
	}
	call(b.L, l)
	lets = append(lets, binding{lv, l})
	call(b.R, r)
	lets = append(lets, binding{rv, r})
	var parts []part
	top := names != nil
	side := func(label string, v *Var, x ast.Expr) {
		switch {
		case isLiteral(x):
			if top {
				parts = append(parts, part{label: label, value: strConst(srcText(x))})
			}
		case top:
			parts = append(parts, part{label: label, value: f.shown(&VarRef{exprBase{v.Type}, v}, x.Span())})
		default:
			if l := partLabel(x); l != "" && f.showable(v.Type) {
				parts = append(parts, part{label: l, value: f.shown(&VarRef{exprBase{v.Type}, v}, x.Span())})
			}
		}
	}
	if top {
		side(names[0], lv, b.L)
		side(names[1], rv, b.R)
	} else {
		side("", lv, b.L)
		side("", rv, b.R)
	}
	if top && b.Op == lexer.Eq {
		if d := f.diffPart(lv, rv, b.Pos, [2]string{names[0], names[1]}); d != nil {
			parts = append(parts, *d)
		}
	}
	// what a side already shows is not shown again: `expect(f(x) == x)`
	for _, a := range args {
		if a.label != partLabel(b.L) && a.label != partLabel(b.R) {
			parts = append(parts, a)
		}
	}
	return c, lets, parts
}

// inParts is `x in c` with x — and c, unless `in` compiles it to comparisons
// — evaluated once, in the order `in` evaluates them. A literal is left where
// it is: `in` gives it its type from the other operand.
func (f *fnCtx) inParts(e *ast.InExpr) (Expr, []binding, []part) {
	var lets []binding
	var parts []part
	bind := func(name string, at ast.Expr) (ast.Expr, bool) {
		x := f.checkExpr(at, nil)
		if types.IsInvalid(x.Type()) {
			return nil, false
		}
		v, n := f.hiddenName(name, x, at.Span())
		lets = append(lets, binding{v, x})
		if l := partLabel(at); l != "" && f.showable(v.Type) {
			parts = append(parts, part{label: l, value: f.shown(&VarRef{exprBase{v.Type}, v}, at.Span())})
		}
		return n, true
	}
	x, c := e.X, e.C
	ok := true
	if !isLiteral(x) {
		x, ok = bind("value", x)
	}
	if ok && !isLiteral(c) && !literalSet(c) {
		c, ok = bind("set", c)
	}
	if !ok {
		return bad(), nil, nil
	}
	return f.checkExprTo(&ast.InExpr{X: x, C: c, Not: e.Not, Pos: e.Pos}, types.TBool), lets, parts
}

// literalSet reports whether `in` compiles c to comparisons (inexpr.go).
func literalSet(c ast.Expr) bool {
	switch c := c.(type) {
	case *ast.ListLit:
		return !c.Mut && len(c.Elems) > 0
	case *ast.RangeExpr:
		return true
	case *ast.StringLit:
		text, plain := plainText(c)
		return plain && len(text) > 0 && len(text) <= maxByteSet
	}
	return false
}

// callParts shows a call's receiver and arguments. The call is checked as
// written; the arguments that are not plain reads are then moved into hidden
// variables evaluated in the same order, before the call, so the report can
// show them without calling anything twice. A receiver or argument that is a
// plain read (a variable, a field of one) is read again for the report.
func (f *fnCtx) callParts(e *ast.CallExpr, c Expr) ([]binding, []part) {
	for _, a := range e.Args {
		if a.Name != nil || a.Spread {
			return nil, f.namedParts(e) // reordered or packed: the arguments no longer line up
		}
	}
	var args []Expr
	direct := false // a call of a function, whose arguments may be moved out
	switch x := c.(type) {
	case *Call:
		if x.Fn == nil || x.Fn.CallerLoc || x.Fn.Sig == nil {
			return nil, f.namedParts(e)
		}
		for _, p := range x.Fn.Sig.Params {
			if p.Variadic || p.Lazy {
				return nil, f.namedParts(e)
			}
		}
		n := len(x.Fn.Params)
		if x.Fn.Receiver != nil {
			n++
		}
		if n != len(x.Args) {
			return nil, f.namedParts(e)
		}
		args, direct = x.Args, true
	case *Builtin:
		args = x.Args
	default:
		return nil, f.namedParts(e)
	}
	// the arguments as written line up with the last ones; one more in front
	// is the receiver, written before the dot
	lead := len(args) - len(e.Args)
	var recv ast.Expr
	switch lead {
	case 0:
		if _, ok := e.Fun.(*ast.NameExpr); !ok {
			if m, ok := e.Fun.(*ast.MemberExpr); !ok || !f.isModuleName(m.X) {
				return nil, f.namedParts(e)
			}
		}
	case 1:
		m, ok := e.Fun.(*ast.MemberExpr)
		if !ok || f.isModuleName(m.X) {
			return nil, f.namedParts(e)
		}
		recv = m.X
	default:
		return nil, f.namedParts(e)
	}
	labels := make([]ast.Expr, len(args))
	if recv != nil {
		labels[0] = recv
	}
	for i, a := range e.Args {
		labels[lead+i] = a.Value
	}
	// moving arguments out keeps their order only when nothing before them
	// is evaluated in the call: the receiver must be a plain read
	move := direct && (lead == 0 || pureRead(args[0]))
	for i := lead; move && i < len(args); i++ {
		if _, isFn := args[i].Type().(*types.Func); isFn {
			move = false // a lambda stays where its call can see it
		}
	}
	var lets []binding
	var parts []part
	for i, arg := range args {
		label := labels[i]
		shownArg := arg
		if a, ok := arg.(*AddrOf); ok && i < lead {
			shownArg = a.X // a receiver passed by its place
		}
		show := label != nil && !isLiteral(label) && !isLiteralExpr(label) && partLabel(label) != "" && f.showable(shownArg.Type())
		switch {
		case pureRead(shownArg):
			if show {
				parts = append(parts, part{label: partLabel(label), value: f.shown(cloneRead(shownArg), label.Span())})
			}
		case move && i >= lead:
			v := f.newTemp(arg.Type())
			lets = append(lets, binding{v, arg})
			args[i] = &VarRef{exprBase{v.Type}, v}
			if show {
				parts = append(parts, part{label: partLabel(label), value: f.shown(&VarRef{exprBase{v.Type}, v}, label.Span())})
			}
		}
	}
	return lets, parts
}

// namedParts, for a call whose checked form does not line up with what is
// written (a method the checker expands in place, defaults filled in), shows
// the receiver and arguments that name a local variable, read again.
func (f *fnCtx) namedParts(e *ast.CallExpr) []part {
	var named []ast.Expr
	if m, ok := e.Fun.(*ast.MemberExpr); ok && !f.isModuleName(m.X) {
		named = append(named, m.X)
	}
	for _, a := range e.Args {
		named = append(named, a.Value)
	}
	var parts []part
	for _, x := range named {
		n, ok := x.(*ast.NameExpr)
		if !ok {
			continue
		}
		sym := f.lookup(n.Name)
		if sym == nil || sym.Kind != SymLocal || !f.showable(sym.Var.Type) {
			continue
		}
		parts = append(parts, part{label: n.Name, value: f.shown(f.narrowedRef(f.localVar(sym.Var)), n.Pos)})
	}
	return parts
}

// isModuleName reports whether x names a module: `path` in `path.join(a)`.
func (f *fnCtx) isModuleName(x ast.Expr) bool {
	n, ok := x.(*ast.NameExpr)
	if !ok {
		return false
	}
	sym := f.lookup(n.Name)
	return sym != nil && sym.Kind == SymModule
}

// pureRead reports whether reading x again gives the same value without
// doing anything: a constant, a variable, a field or element of one.
func pureRead(x Expr) bool {
	switch x := x.(type) {
	case *IntConst, *FloatConst, *BoolConst, *StringConst, *NullConst, *VarRef:
		return true
	case *FieldGet:
		return pureRead(x.X)
	case *TupleGet:
		return pureRead(x.X)
	case *Unwrap:
		return pureRead(x.X)
	case *VariantCast:
		return pureRead(x.X)
	case *SomeWrap:
		return pureRead(x.X)
	case *Cast:
		return pureRead(x.X)
	case *AddrOf:
		return pureRead(x.X)
	}
	return false
}

// cloneRead copies a pureRead expression, so the report's reading of it is
// a tree of its own.
func cloneRead(x Expr) Expr {
	switch x := x.(type) {
	case *VarRef:
		return &VarRef{x.exprBase, x.Var}
	case *FieldGet:
		return &FieldGet{x.exprBase, cloneRead(x.X), x.Index, x.Name}
	case *TupleGet:
		return &TupleGet{x.exprBase, cloneRead(x.X), x.Index}
	case *Unwrap:
		return &Unwrap{x.exprBase, cloneRead(x.X)}
	case *VariantCast:
		return &VariantCast{x.exprBase, cloneRead(x.X), x.Variant}
	case *SomeWrap:
		return &SomeWrap{x.exprBase, cloneRead(x.X)}
	case *Cast:
		return &Cast{x.exprBase, cloneRead(x.X)}
	case *AddrOf:
		return &AddrOf{x.exprBase, cloneRead(x.X)}
	}
	return x // a constant: never changed by codegen
}

// showable reports whether a failure can print a value of type t.
func (f *fnCtx) showable(t types.Type) bool {
	if t == nil || types.IsInvalid(t) || types.IsUnit(t) || types.IsNever(t) || isResultType(t) {
		return false
	}
	switch t.(type) {
	case *types.Func, *types.Pointer, *types.Task, *types.Channel:
		return false
	}
	return f.unionInside(t, map[types.Type]bool{}) == nil
}

// partLabel is how a part is labelled: its text on one line, "" when it is
// too long to be a label.
func partLabel(x ast.Expr) string {
	text := strings.Join(strings.Fields(srcText(x)), " ")
	if len(text) > 60 {
		return ""
	}
	return text
}

// partLines lays the parts out one per line, the values aligned. A value
// printed on several lines (a text, a list of texts) has its later lines
// indented to where the value starts, so the report keeps its shape.
func (f *fnCtx) partLines(parts []part, span source.Span) []Expr {
	width := 0
	for _, p := range parts {
		width = max(width, len(p.label))
	}
	column := len(failIndent) - 1 + width + 2 // under "label: "
	var out []Expr
	for _, p := range parts {
		if p.block != nil { // a block of its own, a diff
			out = append(out, p.block(column))
			continue
		}
		value := p.value
		if _, constant := value.(*StringConst); !constant {
			value = f.indented(value, column, span)
		}
		out = append(out, strConst(failIndent+p.label+":"+strings.Repeat(" ", width-len(p.label)+1)), value)
	}
	return out
}

// indented is the prelude's testIndent(text, column).
func (f *fnCtx) indented(text Expr, column int, span source.Span) Expr {
	call := &ast.CallExpr{
		Fun: &ast.PreludeName{Name: "testIndent", Pos: span},
		Args: []ast.Arg{
			{Value: &ast.CheckedExpr{X: text, Pos: span}},
			{Value: &ast.IntLit{Text: strconv.Itoa(column), Pos: span}},
		},
		Pos: span,
	}
	x := f.checkExprTo(call, types.TString)
	if types.IsInvalid(x.Type()) {
		return text
	}
	return x
}

// diffPart is, for `expect(a == b)` and `expectEqual` on two texts or two
// lists, a diff of the two as printed, when they are long enough that the
// reader would have to look for the difference: the prelude's `testDiff`
// decides. names are the two sides' labels. nil for other types.
func (f *fnCtx) diffPart(l, r *Var, span source.Span, names [2]string) *part {
	if !diffable(l.Type) || !diffable(r.Type) {
		return nil
	}
	// text is compared as it is, without the quotes `left:` shows it in
	text := func(v *Var) Expr {
		if types.IsString(v.Type) {
			return &VarRef{exprBase{v.Type}, v}
		}
		return f.shown(&VarRef{exprBase{v.Type}, v}, span)
	}
	arg := func(x ast.Expr) ast.Arg { return ast.Arg{Value: x} }
	block := func(column int) Expr {
		call := &ast.CallExpr{
			Fun: &ast.PreludeName{Name: "testDiff", Pos: span},
			Args: []ast.Arg{
				arg(&ast.CheckedExpr{X: text(l), Pos: span}),
				arg(&ast.CheckedExpr{X: text(r), Pos: span}),
				arg(&ast.BoolLit{Value: types.IsString(l.Type), Pos: span}),
				arg(&ast.IntLit{Text: strconv.Itoa(column), Pos: span}),
				arg(&ast.CheckedExpr{X: strConst(names[0]), Pos: span}),
				arg(&ast.CheckedExpr{X: strConst(names[1]), Pos: span}),
			},
			Pos: span,
		}
		d := f.checkExprTo(call, types.TString)
		if types.IsInvalid(d.Type()) {
			return strConst("")
		}
		return d
	}
	return &part{block: block}
}

// diffable: the types whose printed form can be long and differ in a detail.
func diffable(t types.Type) bool {
	switch types.Underlying(t).(type) {
	case *types.List, *types.Map, *types.Set, *types.Struct, *types.Array, *types.Tuple:
		return true
	}
	return types.IsString(t)
}

// expectEqualCall is `expectEqual(expected, actual)` (D229): `expected ==
// actual`, each evaluated once, expected first, reported as `expected:` and
// `actual:`.
func (f *fnCtx) expectEqualCall(head string, expected, actual ast.Expr, span source.Span) Expr {
	f.pushScope()
	defer f.popScope()
	eq := &ast.BinaryExpr{Op: lexer.Eq, L: expected, R: actual, Pos: span}
	cond, lets, parts := f.comparisonParts(eq, []string{"expected", "actual"})
	if cond == nil || types.IsInvalid(cond.Type()) {
		return bad()
	}
	msg := concat(append([]Expr{strConst(head)}, f.partLines(parts, span)...)...)
	return wrapLets(lets, f.unless(cond, f.failWith("expectEqual", msg, span), span))
}

// expectNearCall is `expectNear(expected, actual, within: tolerance)` (D230):
// the two floats are equal, or `|actual - expected| <= within`; a NaN never
// passes, since neither holds for it.
func (f *fnCtx) expectNearCall(head string, expected, actual, within ast.Expr, span source.Span) Expr {
	f.pushScope()
	defer f.popScope()
	var e, a Expr
	if isLiteral(expected) && !isLiteral(actual) {
		a = f.checkExpr(actual, nil)
		e = f.checkExpr(expected, a.Type())
	} else {
		e = f.checkExpr(expected, nil)
		a = f.checkExpr(actual, e.Type())
	}
	w := f.checkExpr(within, e.Type())
	if types.IsInvalid(e.Type()) || types.IsInvalid(a.Type()) || types.IsInvalid(w.Type()) {
		return bad()
	}
	if !types.IsFloat(e.Type()) || !types.Identical(e.Type(), a.Type()) || !types.Identical(e.Type(), w.Type()) {
		f.errorf(span, "'expectNear' compares two floats of one type within a tolerance of that type, found '%s', '%s' and '%s'; for other values, expectEqual(expected, actual)", e.Type(), a.Type(), w.Type())
		return bad()
	}
	ev, en := f.hiddenName("expected", e, expected.Span())
	av, an := f.hiddenName("actual", a, actual.Span())
	wv, wn := f.hiddenName("within", w, within.Span())
	offBy := func() ast.Expr {
		diff := &ast.BinaryExpr{Op: lexer.Minus, L: an, R: en, Pos: span}
		return &ast.CallExpr{Fun: &ast.MemberExpr{X: diff, Name: ast.Ident{Name: "abs", Pos: span}, Pos: span}, Pos: span}
	}
	near := &ast.BinaryExpr{Op: lexer.OrOr, Pos: span,
		L: &ast.BinaryExpr{Op: lexer.Eq, L: en, R: an, Pos: span},
		R: &ast.BinaryExpr{Op: lexer.LtEq, L: offBy(), R: wn, Pos: span},
	}
	cond := f.checkExprTo(near, types.TBool)
	off := f.checkExprTo(offBy(), e.Type())
	if types.IsInvalid(cond.Type()) || types.IsInvalid(off.Type()) {
		return bad()
	}
	shown := func(label string, v *Var, x ast.Expr) part {
		if isLiteral(x) {
			return part{label: label, value: strConst(srcText(x))}
		}
		return part{label: label, value: f.shown(&VarRef{exprBase{v.Type}, v}, x.Span())}
	}
	parts := []part{shown("expected", ev, expected), shown("actual", av, actual), {label: "off by", value: f.shown(off, span)}}
	msg := concat(append([]Expr{strConst(head)}, f.partLines(parts, span)...)...)
	lets := []binding{{ev, e}, {av, a}, {wv, w}}
	return wrapLets(lets, f.unless(cond, f.failWith("expectNear", msg, span), span))
}

// wrapLets binds each hidden variable around x, the first outermost.
func wrapLets(lets []binding, x Expr) Expr {
	for i := len(lets) - 1; i >= 0; i-- {
		x = &Let{exprBase{types.TUnit}, lets[i].v, lets[i].init, x}
	}
	return x
}
