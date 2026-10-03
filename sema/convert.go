package sema

import (
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// D86: numeric conversions are methods, split by what they can lose.
//
//	b.toI64()     total where the whole range fits (u8 → i64, i32 → f64), else
//	              a `T?`: null when the value is out of range (i64 → u8,
//	              f64 → i32; a float truncates toward zero, NaN is null)
//	x.wrapU8()    integer to integer, always total: keeps the low bits
//
// `as` only renames (D85); `p.cast<*raw U>()` reinterprets a raw pointer.

// convTarget reads `toI64` / `wrapU8` into the kind and the target type.
func convTarget(name string) (kind string, to *types.Basic, ok bool) {
	for _, k := range []string{"to", "wrap"} {
		rest, found := strings.CutPrefix(name, k)
		if !found || rest == "" {
			continue
		}
		b, isPrim := types.Primitives[strings.ToLower(rest[:1])+rest[1:]]
		if isPrim && types.IsNumeric(b) {
			return k, b, true
		}
	}
	return "", nil, false
}

// losslessConv reports whether every value of `from` is a value of `to`.
func losslessConv(from, to types.Type) bool {
	switch {
	case types.IsInteger(from) && types.IsInteger(to):
		fb, tb := types.BitSize(from), types.BitSize(to)
		if types.IsSigned(from) == types.IsSigned(to) {
			return tb >= fb
		}
		return !types.IsSigned(from) && tb > fb
	case types.IsFloat(from) && types.IsFloat(to):
		return true // f64 → f32 rounds; it never has no answer
	case types.IsInteger(from) && types.IsFloat(to):
		return true // rounds to nearest
	}
	return false
}

// numericConversion lowers `x.toT()` / `x.wrapT()`; nil when `name` is not one.
func (f *fnCtx) numericConversion(recv Expr, from types.Type, name string, e *ast.CallExpr) Expr {
	kind, to, ok := convTarget(name)
	if !ok || !types.IsNumeric(from) {
		return nil
	}
	if len(e.Args) != 0 {
		f.arityError(e.Pos, from, name, 0)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if kind == "wrap" {
		if !types.IsInteger(from) || !types.IsInteger(to) {
			f.errorf(e.Pos, "'%s' keeps the low bits of an integer, so it needs integer types on both sides (D86); write '.to%s()' for '%s', which is null when the value does not fit", name, name[len("wrap"):], from)
			return bad()
		}
		return &Cast{exprBase{to}, recv}
	}
	if losslessConv(from, to) {
		return &Cast{exprBase{to}, recv}
	}
	if c, isConst := recv.(*IntConst); isConst && !intConstFits(c, to) {
		if sel, ok := e.Fun.(*ast.MemberExpr); ok && !isLiteralExpr(sel.X) {
			// a constant written in (D113): its value is known, so is the answer
			v := strconv.FormatUint(c.Value, 10)
			if c.Neg {
				v = "-" + v
			}
			f.errorf(e.Pos, "'%s' is the constant %s, which does not fit '%s', so this is always null (D86, D113); use '.wrap%s()' to keep the low bits", srcText(sel.X), v, to, name[len("to"):])
			return bad()
		}
		f.errorf(e.Pos, "the literal does not fit '%s' (D86); use '.wrap%s()' to keep the low bits", to, name[len("to"):])
		return bad()
	}
	return &Builtin{exprBase{&types.Nullable{Elem: to}}, "num.toChecked", []Expr{recv}, e.Pos}
}

// intConstFits reports whether a literal is inside the range of `to`.
func intConstFits(c *IntConst, to *types.Basic) bool {
	n := types.BitSize(to)
	if !types.IsInteger(to) || n <= 0 || n > 64 {
		return true
	}
	if types.IsSigned(to) {
		if c.Neg {
			return c.Value <= uint64(1)<<(n-1)
		}
		return c.Value <= uint64(1)<<(n-1)-1
	}
	if c.Neg {
		return c.Value == 0
	}
	if n == 64 {
		return true
	}
	return c.Value <= uint64(1)<<n-1
}

// rawPointerCast lowers `p.cast<*raw U>()`; nil when `name` is not `cast`.
func (f *fnCtx) rawPointerCast(recv Expr, name string, typeArgs []types.Type, e *ast.CallExpr) Expr {
	if name != "cast" {
		return nil
	}
	rp, _ := recv.Type().(*types.Pointer)
	if len(e.Args) != 0 || len(typeArgs) != 1 || !isRawPointer(typeArgs[0]) {
		f.errorf(e.Pos, "'cast' takes the raw pointer type to reinterpret as and no arguments: 'p.cast<*raw U>()' (D86)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if rp.Raw && f.unsafe == 0 {
		f.errorf(e.Pos, "casting a raw pointer requires an 'unsafe' block (D44)")
	}
	if !rp.Raw && f.unsafe == 0 {
		f.errorf(e.Pos, "handing out a raw address requires an 'unsafe' block (D44)")
	}
	return &Cast{exprBase{typeArgs[0]}, recv}
}

// convMethodFor names the method that replaces `x as T` for a conversion
// between the two types, or "" when no numeric method does.
func convMethodFor(from, to types.Type) string {
	fb, ok1 := types.Underlying(from).(*types.Basic)
	tb, ok2 := types.Underlying(to).(*types.Basic)
	if !ok1 || !ok2 || !types.IsNumeric(fb) || !types.IsNumeric(tb) {
		return ""
	}
	suffix := strings.ToUpper(tb.Name[:1]) + tb.Name[1:]
	if types.IsInteger(fb) && types.IsInteger(tb) && !losslessConv(fb, tb) {
		return "wrap" + suffix // `as` truncated: the fix keeps that meaning, spelled out
	}
	return "to" + suffix
}

// removedAs reports `x as T` (D86: `as` only renames), with the method that
// replaces it as a fix. The caller goes on to lower the cast as before.
func (f *fnCtx) removedAs(e *ast.CastExpr, from, to types.Type) {
	recv := srcText(e.X)
	if recv != "" && !isSimpleOperand(recv) {
		recv = "(" + recv + ")"
	}
	target := srcTextSpan(e.Type.Span(), to.String())
	var repl, what string
	switch {
	case types.Identical(from, to):
		repl, what = srcText(e.X), "drop the cast, the value already is '"+to.String()+"'"
	case convMethodFor(from, to) != "":
		m := convMethodFor(from, to)
		repl, what = recv+"."+m+"()", "write '."+m+"()'"
		if strings.HasPrefix(m, "to") && !losslessConv(types.Underlying(from), types.Underlying(to)) {
			what += " (a '" + to.String() + "?': null when it does not fit)"
		}
	case isRawPointer(to):
		repl, what = recv+".cast<"+target+">()", "write '.cast<"+target+">()'"
	}
	if repl == "" || recv == "" {
		f.errorf(e.Pos, "'as' no longer converts (D86; it only renames): numeric conversions are methods ('.toI64()', '.wrapU8()') and a raw pointer uses '.cast<*raw T>()'")
		return
	}
	f.c.errorFix(e.Pos, fixReplace("Replace with the conversion method", e.Pos, repl),
		"'as' no longer converts (D86; it only renames): %s", what)
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// The conversion methods are catalogued for hover and completion: one entry
// per target on each family (the receiver's own width decides whether `toT`
// is total or a `T?`, which the doc says).
func init() {
	builtinDocs = append(builtinDocs, BuiltinDoc{Recv: "int", Name: "wrapTo", Sig: "<T>(): T",
		Doc: "`wrapT()` with the target as a type argument, for generic code: `wide.wrapTo<T>()` keeps the low bits of an integer `T` (D86)."})
	for _, name := range []string{"i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64"} {
		up := upperFirst(name)
		for _, family := range []string{"int", "float"} {
			doc := "Converts to `" + name + "`. Where every value of this type is a `" + name + "` it returns a `" + name + "`; where a value could be lost it returns a `" + name + "?`, null when it does not fit" +
				map[string]string{"int": " (D86).", "float": ": a float truncates toward zero, NaN is null (D86)."}[family]
			builtinDocs = append(builtinDocs, BuiltinDoc{Recv: family, Name: "to" + up, Sig: "(): " + name + "?", Doc: doc})
		}
		if !strings.HasPrefix(name, "f") {
			builtinDocs = append(builtinDocs, BuiltinDoc{Recv: "int", Name: "wrap" + up, Sig: "(): " + name,
				Doc: "Converts to `" + name + "` keeping the low bits (two's complement): the conversion `+%` is to `+`. Always succeeds (D86)."})
		}
	}
}

// wrapTo lowers `x.wrapTo<T>()` (D86 addendum, Q19): `wrapT()` for a target
// written as a type argument, which generic code can leave to its own type
// parameter. Checked once the parameter is known, so `wrapTo<string>()` is
// refused where it is instantiated.
func (f *fnCtx) wrapTo(recv Expr, from types.Type, typeArgs []types.Type, e *ast.CallExpr) Expr {
	if len(e.Args) != 0 || len(typeArgs) != 1 {
		f.errorf(e.Pos, "'wrapTo' takes the integer type to wrap into and no arguments: 'x.wrapTo<u8>()' (D86)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	to := typeArgs[0]
	if !types.IsInteger(from) || !types.IsInteger(to) {
		f.errorf(e.Pos, "'wrapTo' keeps the low bits of an integer, so it needs integer types on both sides, found '%s' to '%s' (D86)", from, to)
		return bad()
	}
	return &Cast{exprBase{to}, recv}
}
