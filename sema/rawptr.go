package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Raw pointer arithmetic (D50). Inside `unsafe`, `p + n` and `p - n` step
// by whole elements as in C, `p - q` counts the elements between two
// pointers of one type, and `<`, `<=`, `>`, `>=` order addresses. Nothing
// is checked — a raw pointer knows no bounds, so there is nothing to check
// against — and nothing wraps or traps: the address is what C would compute.

// rawPointer is t as a (non-nullable) `*raw T`, or nil.
func rawPointer(t types.Type) *types.Pointer {
	if p, ok := t.(*types.Pointer); ok && p.Raw {
		return p
	}
	return nil
}

// isRawPointerOp: the operators D50 gives a raw pointer on its left.
func isRawPointerOp(op BinOp) bool {
	switch op {
	case OpAdd, OpSub, OpLt, OpLe, OpGt, OpGe:
		return true
	}
	return false
}

// rawPointerBinary checks `l op rhs` where l is a `*raw T`.
func (f *fnCtx) rawPointerBinary(op BinOp, l Expr, rhs ast.Expr, span source.Span) Expr {
	p := rawPointer(l.Type())
	if !isRawPointerOp(op) {
		f.checkExpr(rhs, nil)
		f.errorf(span, "operator '%s' is not defined for '%s'; a raw pointer takes '+ n', '- n', '- q' and '<' '<=' '>' '>=' (D50)", op, l.Type())
		return bad()
	}
	if f.unsafe == 0 {
		f.errorf(span, "pointer arithmetic requires an 'unsafe' block (D50)")
	}
	r := f.checkExpr(rhs, nil)
	if types.IsInvalid(r.Type()) {
		return bad()
	}
	q := rawPointer(r.Type())
	switch {
	case op == OpAdd || (op == OpSub && q == nil):
		if !types.IsInteger(r.Type()) {
			f.errorf(rhs.Span(), "a raw pointer moves by a whole number of elements, not by a '%s' (D50)", r.Type())
			return bad()
		}
		if !f.sizedPointee(p, span) {
			return bad()
		}
		if !types.Identical(r.Type(), types.TI64) {
			r = &Cast{exprBase{types.TI64}, r}
		}
		name := "rawptr.add"
		if op == OpSub {
			name = "rawptr.sub"
		}
		return &Builtin{exprBase{p}, name, []Expr{l, r}, span}
	case q == nil || !types.Identical(p, q):
		if op == OpSub {
			f.errorf(span, "cannot subtract a '%s' from a '%s': 'p - q' counts elements between two pointers of one type (D50)", r.Type(), l.Type())
		} else {
			f.errorf(span, "cannot compare a '%s' with a '%s': both sides must be pointers of one type (D50)", l.Type(), r.Type())
		}
		return bad()
	case op == OpSub:
		if !f.sizedPointee(p, span) {
			return bad()
		}
		return &Builtin{exprBase{types.TI64}, "rawptr.diff", []Expr{l, r}, span}
	}
	return &Builtin{exprBase{types.TBool}, "rawptr." + rawPointerCompare[op], []Expr{l, r}, span}
}

var rawPointerCompare = map[BinOp]string{OpLt: "lt", OpLe: "le", OpGt: "gt", OpGe: "ge"}

// sizedPointee: arithmetic steps by the element's size, which a `*raw ()`
// (C's `void *`) does not have.
func (f *fnCtx) sizedPointee(p *types.Pointer, span source.Span) bool {
	if types.IsUnit(p.Elem) || types.IsNever(p.Elem) {
		f.errorf(span, "'%s' points at nothing with a size, so it cannot step; cast it to '*raw u8' to move by bytes (D50)", p)
		return false
	}
	return true
}
