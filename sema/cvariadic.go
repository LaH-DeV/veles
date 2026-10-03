package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// C's variadic arguments (D123): `fun printf(format: *raw u8, ...): i32` in
// an `extern "C"` block. The arguments after the named ones get C's default
// promotions — bool, i8, i16, u8, u16 widen to i32 and f32 to f64 — and
// must then be a type C reads from `...`: a 32- or 64-bit integer, f64, a
// raw pointer (nullable or not) or a C function pointer. A literal is typed
// as C types it: an integer literal is an `i32` (C's `int`), a float one an
// `f64`, so `printf("%d", 5)` passes what `%d` reads.

// splitCVariadic takes the arguments past the named parameters off a call
// of a C-variadic function: they are positional, never named.
func (f *fnCtx) splitCVariadic(sig *types.Func, args []ast.Arg) (named, extra []ast.Arg, ok bool) {
	n := len(sig.Params)
	if len(args) <= n {
		return args, nil, true
	}
	ok = true
	for _, a := range args[n:] {
		if a.Name != nil || a.Spread {
			f.errorf(a.Value.Span(), "C's variadic arguments are positional, after every named parameter: no name, no '...' (D123)")
			ok = false
		}
	}
	return args[:n], args[n:], ok
}

// cVariadicArgs checks the extra arguments and applies the promotions.
func (f *fnCtx) cVariadicArgs(extra []ast.Arg) []Expr {
	var out []Expr
	for _, a := range extra {
		var x Expr
		lit := a.Value
		if u, ok := lit.(*ast.UnaryExpr); ok && u.Op == lexer.Minus {
			lit = u.X // `-1` is typed as `1` is
		}
		switch lit.(type) {
		case *ast.IntLit:
			x = f.checkExprTo(a.Value, types.TI32)
		case *ast.FloatLit:
			x = f.checkExprTo(a.Value, types.TF64)
		default:
			x = f.checkExpr(a.Value, nil)
		}
		for {
			// a @transparent wrapper passes as its field (D120)
			st, ok := x.Type().(*types.Struct)
			if !ok || !st.Transparent || len(st.Fields) != 1 {
				break
			}
			x = &FieldGet{exprBase{st.Fields[0].Type}, x, 0, st.Fields[0].Name}
		}
		t := types.Underlying(x.Type())
		if types.IsInvalid(t) {
			out = append(out, x)
			continue
		}
		switch {
		case types.IsBool(t), types.IsInteger(t) && types.BitSize(t) < 32:
			x = &Cast{exprBase{types.TI32}, x}
		case types.IsFloat(t) && types.BitSize(t) == 32:
			x = &Cast{exprBase{types.TF64}, x}
		case types.IsNumeric(t), cVariadicPointer(t):
		default:
			f.errorf(a.Value.Span(), "a '%s' cannot be passed to C's variadic arguments: pass a number, a raw pointer or a C function pointer (D123)", x.Type())
			x = bad()
		}
		out = append(out, x)
	}
	return out
}

func cVariadicPointer(t types.Type) bool {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	switch t := t.(type) {
	case *types.Pointer:
		return t.Raw
	case *types.Func:
		return t.C
	}
	return false
}

func paramTypesOf(sig *types.Func) []types.Type {
	out := make([]types.Type, len(sig.Params))
	for i, p := range sig.Params {
		out[i] = p.Type
	}
	return out
}
