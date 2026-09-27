package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The test vocabulary (D78). Each word is compiler-known so its failure
// can say what was written, what each side was, and where — which a
// library function cannot. They are in scope only in test code; `assert`,
// the invariant form, is in scope everywhere. A declaration of the same
// name wins, as with `panic`.
var testWords = map[string]bool{"expect": true, "require": true, "expectThrows": true, "expectPanics": true, "fail": true}

// inTest reports whether the code being checked is test code: a test, a
// `test fun`, a function in a `*.test.vs` file, or a lambda inside one.
func (f *fnCtx) inTest() bool {
	for g := f; g != nil; g = g.parent {
		if g.fn != nil && g.fn.tmpl != nil && g.fn.tmpl.TestCode {
			return true
		}
	}
	return isTestFile(f.file)
}

// failIndent starts each detail line under a failure's first line: the
// runner prints "  file:line:col: " before the first.
const failIndent = "\n      "

// testCall checks one word of the vocabulary.
func (f *fnCtx) testCall(name string, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	if name != "assert" && !f.inTest() {
		f.errorf(e.Fun.Span(), "'%s' is for tests: use it inside 'test \"...\" { }', a 'test fun' or a *.test.vs file; outside a test, 'assert(cond, \"why\")' panics when an invariant does not hold (D78)", name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if f.c.index != nil {
		if n, ok := e.Fun.(*ast.NameExpr); ok {
			f.c.index.Refs = append(f.c.index.Refs, Ref{Span: n.Pos, Kind: "fun", Name: name, Detail: testWordSigs[name] + "  (built in)", Doc: testWordDocs[name]})
		}
	}
	if name == "assert" {
		// the reason is not optional: an invariant that breaks in
		// production must say what was meant to hold
		if len(e.Args) != 2 || e.Args[0].Name != nil || e.Args[1].Name != nil {
			f.errorf(e.Pos, "'assert' takes a condition and the reason it must hold, like 'panic' takes a message: assert(xs.len() > 0, \"a report has at least one row\")")
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		why := f.checkExprTo(e.Args[1].Value, types.TString)
		cond := e.Args[0].Value
		return f.expectCall(name, "assert("+srcText(cond)+")", cond, why, e.Pos)
	}
	if len(e.Args) != 1 || e.Args[0].Name != nil {
		f.errorf(e.Pos, "'%s' takes one argument: %s", name, testWordSigs[name])
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	arg := e.Args[0].Value
	head := srcText(e) // the call as written, type arguments and all
	switch name {
	case "expect":
		return f.expectCall(name, head, arg, nil, e.Pos)
	case "require":
		return f.requireCall(head, arg, e.Pos, want)
	case "expectThrows":
		if len(typeArgs) > 1 {
			f.errorf(e.Pos, "'expectThrows' takes one error type: expectThrows<RangeError>(() => ...)")
		}
		var errType types.Type
		if len(typeArgs) == 1 {
			errType = typeArgs[0]
		}
		return f.expectThrowsCall(head, arg, errType, e.Pos)
	case "expectPanics":
		return f.expectPanicsCall(head, arg, e.Pos)
	case "fail":
		msg := f.checkExprTo(arg, types.TString)
		return &Builtin{exprBase{types.TNever}, "test.stop", []Expr{msg}, e.Pos}
	}
	return bad()
}

var testWordSigs = map[string]string{
	"expect":       "fun expect(condition: bool)",
	"require":      "fun require<T>(value: T? or Result<T, E>): T",
	"expectThrows": "fun expectThrows<E>(body: fun() throws E)",
	"expectPanics": "fun expectPanics(body: fun())",
	"fail":         "fun fail(why: string): Never",
	"assert":       "fun assert(condition: bool, message: string)",
}

var testWordDocs = map[string]string{
	"expect": "Records a failure when `condition` is false, and the test goes on — one run reports every broken expectation. " +
		"The report shows the expression as written and, for a comparison, the value of each side:\n\n" +
		"```text\n  main.vs:4:3: expect(parsePort(\"80\") == 80)\n      left:  81\n      right: 80\n```\n\nIn test code only (D78).",
	"require": "The value of a `T?` or a `Result`; when there is none, records the failure (with the error) and ends the test: " +
		"`val cfg = require(load(\"app.toml\"))`. In test code only (D78).",
	"expectThrows": "Calls `body` and records a failure unless it throws — an `E`, when one is named: " +
		"`expectThrows<RangeError>(() => parsePort(\"70000\"))`. In test code only (D78).",
	"expectPanics": "Calls `body` in a task of its own and records a failure unless it panics: " +
		"`expectPanics(() => { val _ = [1].at(5) ?: panic(\"out of range\") })`. In test code only (D78).",
	"fail": "Records `why` and ends the test: `val x = maybe ?: fail(\"no x\")`. In test code only (D78).",
	"assert": "Panics with `message` when `condition` is false — an invariant that must hold, anywhere in a program. " +
		"The panic also shows the condition and, for a comparison, both sides' values; `message` is evaluated only then (D78).",
}

// isLiteral reports whether x is written as its own value, so a failure
// can show its text instead of its value: `80`, `"abc"`, `null`, `[]`.
func isLiteral(x ast.Expr) bool {
	switch x := x.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.NullLit, *ast.CharLit:
		return true
	case *ast.StringLit:
		for _, p := range x.Parts {
			if p.Expr != nil {
				return false
			}
		}
		return true
	case *ast.UnaryExpr:
		return x.Op == lexer.Minus && isLiteral(x.X)
	case *ast.ListLit:
		for _, el := range x.Elems {
			if !isLiteral(el) {
				return false
			}
		}
		return true
	}
	return false
}

func isComparison(op lexer.TokenKind) bool {
	switch op {
	case lexer.Eq, lexer.NotEq, lexer.Lt, lexer.LtEq, lexer.Gt, lexer.GtEq:
		return true
	}
	return false
}

// hiddenName declares a checked value under a name source cannot spell, for
// the syntax built around it; the caller binds it with a Let.
func (f *fnCtx) hiddenName(prefix string, x Expr) (*Var, ast.Expr) {
	f.c.nextTmp++
	name := "$" + prefix + itoa(f.c.nextTmp)
	v := f.newVar(name, x.Type(), false, source.Span{})
	f.declareLocal(name, v, source.Span{})
	return v, &ast.NameExpr{Name: name}
}

// shown is how a failure prints a value: a string quoted, so "" and " "
// are visible; anything else as interpolation prints it.
func (f *fnCtx) shown(x Expr, span source.Span) Expr {
	if types.IsString(x.Type()) {
		return concat(strConst("\""), x, strConst("\""))
	}
	return f.toString(x, span)
}

func strConst(s string) Expr { return &StringConst{exprBase{types.TString}, s} }

func concat(parts ...Expr) Expr { return &StringConcat{exprBase{types.TString}, parts} }

// failWith records a soft failure, or for `assert` panics with it.
func (f *fnCtx) failWith(word string, msg Expr, span source.Span) Expr {
	if word == "assert" {
		return &Builtin{exprBase{types.TNever}, "panic", []Expr{msg}, span}
	}
	return &Builtin{exprBase{types.TUnit}, "test.fail", []Expr{msg}, span}
}

// unless is `if (!cond) body`.
func (f *fnCtx) unless(cond Expr, body Expr, span source.Span) Expr {
	not := &Unary{exprBase{types.TBool}, OpNot, cond, span}
	return &If{exprBase{types.TUnit}, not, f.valueBlock(body), nil}
}

// expectCall is `expect(cond)` and `assert(cond, why)`. A comparison has
// each side evaluated once, in source order, so the report can show its
// value; a literal side is shown as written. An assert's reason leads its
// panic — "panic: a report has rows", then the condition — and is only
// evaluated when the condition is false.
func (f *fnCtx) expectCall(word, head string, cond ast.Expr, why Expr, span source.Span) Expr {
	lead := []Expr{strConst(head)}
	if why != nil {
		lead = []Expr{why, strConst(failIndent + head)}
	}
	b, isCmp := cond.(*ast.BinaryExpr)
	if !isCmp || !isComparison(b.Op) {
		c := f.checkExprTo(cond, types.TBool)
		if types.IsInvalid(c.Type()) {
			return bad()
		}
		return f.unless(c, f.failWith(word, concat(lead...), span), span)
	}
	f.pushScope()
	defer f.popScope()
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
		return bad()
	}
	lv, ln := f.hiddenName("left", l)
	rv, rn := f.hiddenName("right", r)
	c := f.checkExprTo(&ast.BinaryExpr{Op: b.Op, L: ln, R: rn, Pos: b.Pos}, types.TBool)
	if types.IsInvalid(c.Type()) {
		return bad()
	}
	side := func(v *Var, x ast.Expr) Expr {
		if isLiteral(x) {
			return strConst(srcText(x))
		}
		return f.shown(&VarRef{exprBase{v.Type}, v}, span)
	}
	msg := concat(append(lead, strConst(failIndent+"left:  "), side(lv, b.L), strConst(failIndent+"right: "), side(rv, b.R))...)
	test := f.unless(c, f.failWith(word, msg, span), span)
	return &Let{exprBase{types.TUnit}, lv, l, &Let{exprBase{types.TUnit}, rv, r, test}}
}

// requireCall is `require(x)`: the value of a `T?` or a `Result`, or the
// end of the test.
func (f *fnCtx) requireCall(head string, arg ast.Expr, span source.Span, want types.Type) Expr {
	var hint types.Type
	if want != nil && !types.IsInvalid(want) {
		hint = &types.Nullable{Elem: want}
	}
	x := f.checkExpr(arg, hint)
	xt := x.Type()
	if types.IsInvalid(xt) {
		return bad()
	}
	tmp := f.newTemp(xt)
	ref := &VarRef{exprBase{xt}, tmp}
	stop := func(msg Expr) *Block {
		return f.valueBlock(&Builtin{exprBase{types.TNever}, "test.stop", []Expr{msg}, span})
	}
	switch t := xt.(type) {
	case *types.Nullable:
		missing := &IsNull{exprBase{types.TBool}, ref}
		value := &Unwrap{exprBase{t.Elem}, ref}
		pick := &If{exprBase{t.Elem}, missing, stop(strConst(head + failIndent + "was null")), &Block{Value: value, Type: t.Elem}}
		return &Let{exprBase{t.Elem}, tmp, x, pick}
	case *types.Sealed:
		if isResultType(t) {
			okT := t.TypeArgs[0]
			okV, errV := t.Variants[0], t.Variants[1]
			failed := &VariantTest{exprBase{types.TBool}, ref, errV}
			payload := &FieldGet{exprBase{okT}, &VariantCast{exprBase{okV}, ref, okV}, 0, okV.Fields[0].Name}
			errValue := &FieldGet{exprBase{errV.Fields[0].Type}, &VariantCast{exprBase{errV}, ref, errV}, 0, errV.Fields[0].Name}
			msg := concat(strConst(head+failIndent+"threw: "), f.toString(errValue, span))
			pick := &If{exprBase{okT}, failed, stop(msg), &Block{Value: payload, Type: okT}}
			return &Let{exprBase{okT}, tmp, x, pick}
		}
	}
	f.errorf(arg.Span(), "'require' unwraps a nullable or a Result (a call to a 'throws' function, without 'try'), found '%s'; for a condition, write 'expect(...)'", xt)
	return bad()
}

// body checks the function a word calls — a lambda or any function value
// taking nothing — and binds it to a hidden name.
func (f *fnCtx) body(word string, arg ast.Expr) (Expr, *Var, ast.Expr, *types.Func) {
	fx := f.checkExpr(arg, nil)
	ft, ok := fx.Type().(*types.Func)
	if !ok {
		if !types.IsInvalid(fx.Type()) {
			f.errorf(arg.Span(), "'%s' takes a function to call, like '() => parse(text)'; found '%s'", word, fx.Type())
		}
		return nil, nil, nil, nil
	}
	if len(ft.Params) > 0 {
		f.errorf(arg.Span(), "'%s' calls its function with no arguments; wrap the call: '() => ...'", word)
		return nil, nil, nil, nil
	}
	v, n := f.hiddenName("body", fx)
	return fx, v, n, ft
}

// expectThrowsCall is `expectThrows<E>(body)`: a failure unless calling
// body throws (an E, when named). A body that cannot throw is refused:
// the expectation could never hold.
func (f *fnCtx) expectThrowsCall(head string, arg ast.Expr, errType types.Type, span source.Span) Expr {
	f.pushScope()
	defer f.popScope()
	fx, v, n, ft := f.body("expectThrows", arg)
	if fx == nil {
		return bad()
	}
	// `() => parse(x)` returns the call's Result; `() => try parse(x)` throws
	// — either way, calling it yields a Result
	rs, returnsResult := ft.Ret.(*types.Sealed)
	throws := ft.Effects.Throws && ft.Effects.Error != nil && !types.IsNever(ft.Effects.Error)
	if !throws && (!returnsResult || !isResultType(rs)) {
		f.errorf(arg.Span(), "nothing in this function can throw, so 'expectThrows' would always fail")
		return bad()
	}
	thrown := ft.Effects.Error
	if !throws {
		thrown = rs.TypeArgs[1]
	}
	if errType != nil && !types.IsInvalid(errType) && !canThrow(thrown, errType) {
		f.errorf(arg.Span(), "this function throws '%s', never '%s', so 'expectThrows<%s>' could not hold", thrown, errType, errType)
		return bad()
	}
	b := &synth{sp: span}
	errName := "$error" + itoa(f.c.nextTmp)
	var onErr []ast.Stmt
	if errType != nil {
		wrong := &ast.IsExpr{X: b.name(errName), Pat: &ast.TypePat{Type: b.typ(errType), Pos: span}, Not: true, Pos: span}
		onErr = []ast.Stmt{b.ifStmt(wrong, []ast.Stmt{b.stmt(b.call(b.name("$testFail"), b.interp(head+failIndent+"expected "+errType.String()+", threw: ", b.name(errName), "")))}, nil)}
	}
	bind := ""
	if errType != nil {
		bind = errName
	}
	w := &ast.WhenExpr{Subject: b.call(n), Pos: span, Arms: []*ast.WhenArm{
		{Patterns: []ast.Pattern{b.resultPat("Ok", "")}, Body: b.call(b.name("$testFail"), b.str(head+failIndent+"nothing was thrown")), Pos: span},
		{Patterns: []ast.Pattern{b.resultPat("Err", bind)}, Body: b.blockExpr(onErr), Pos: span},
	}}
	return &Let{exprBase{types.TUnit}, v, fx, f.checkExprTo(w, types.TUnit)}
}

// canThrow reports whether an error of set thrown can be a want.
func canThrow(thrown, want types.Type) bool {
	if types.Identical(thrown, want) {
		return true
	}
	if u, ok := thrown.(*types.ErrorUnion); ok {
		return types.UnionIndex(u, want) >= 0
	}
	return false
}

// expectPanicsCall is `expectPanics(body)`: body runs in a task of its
// own, behind the `gather` boundary where a panic is a value (D52), and a
// failure is recorded unless it panicked.
func (f *fnCtx) expectPanicsCall(head string, arg ast.Expr, span source.Span) Expr {
	f.pushScope()
	defer f.popScope()
	b := &synth{sp: span}
	f.c.nextTmp++
	errName := "$error" + itoa(f.c.nextTmp)
	// the body is checked as the argument of the prelude's runTestBody, so
	// a lambda is checked as the sendable function a task needs
	launch := &ast.CallExpr{Fun: &ast.PreludeName{Name: "runTestBody", Pos: span}, Args: []ast.Arg{{Value: arg}}, Async: true, Pos: span}
	outcome := b.member(&ast.GatherExpr{Body: b.block([]ast.Stmt{b.stmt(launch)}), Pos: span}, "0")
	notPanic := &ast.IsExpr{X: b.name(errName), Pat: &ast.TypePat{Type: b.typ(f.c.panicType()), Pos: span}, Not: true, Pos: span}
	w := &ast.WhenExpr{Subject: outcome, Pos: span, Arms: []*ast.WhenArm{
		{Patterns: []ast.Pattern{b.resultPat("Ok", "")}, Body: b.call(b.name("$testFail"), b.str(head+failIndent+"it returned without panicking")), Pos: span},
		{Patterns: []ast.Pattern{b.resultPat("Err", errName)}, Body: b.blockExpr([]ast.Stmt{
			b.ifStmt(notPanic, []ast.Stmt{b.stmt(b.call(b.name("$testFail"), b.interp(head+failIndent+"it threw instead: ", b.name(errName), "")))}, nil),
		}), Pos: span},
	}}
	return f.checkExprTo(w, types.TUnit)
}

// testFailCall is the `$testFail(message)` the words above build: a soft
// failure recorded at the call's location. Source cannot spell the name.
func (f *fnCtx) testFailCall(e *ast.CallExpr) Expr {
	msg := f.checkExprTo(e.Args[0].Value, types.TString)
	return &Builtin{exprBase{types.TUnit}, "test.fail", []Expr{msg}, e.Pos}
}
