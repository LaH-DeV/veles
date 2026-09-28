package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Arithmetic operators on types that are not numbers (D71): `a + b`,
// `a - b`, `a * b`, `a / b` and `-a` call `plus`, `minus`, `times`,
// `dividedBy` and `negate` of the prelude traits Addable, Subtractable,
// Multipliable, Divisible and Negatable. The right operand's type and the
// result's are the trait's associated types, so they may differ from the
// left (`Timestamp + Duration`), and a generic reaches them through a bound.

// operatorTraits maps a binary operator to its trait and method.
var operatorTraits = map[BinOp][2]string{
	OpAdd: {"Addable", "plus"},
	OpSub: {"Subtractable", "minus"},
	OpMul: {"Multipliable", "times"},
	OpDiv: {"Divisible", "dividedBy"},
}

// operandTakesOperator reports whether a binary operator on a left operand
// of type t goes through its trait: arithmetic on anything but a number
// (and `+` on anything but a string).
func operandTakesOperator(op BinOp, t types.Type) bool {
	if _, ok := operatorTraits[op]; !ok {
		return false
	}
	if types.IsInvalid(t) || types.IsNumeric(t) || types.IsEnum(t) || rawPointer(t) != nil {
		return false
	}
	return !(op == OpAdd && types.IsString(t))
}

// preludeTrait finds a prelude trait by name.
func (c *Checker) preludeTrait(name string) *types.Trait {
	if tr, ok := c.preludeType(name).(*types.Trait); ok {
		return tr
	}
	return nil
}

// operatorCall lowers `lhs <op> rhs` to `lhs.<method>(rhs)` when lhs's type
// implements the operator's trait. The checked left operand stands in for
// a placeholder receiver, so the call takes the ordinary method path:
// dispatch through a bound, the right operand typed from the parameter.
func (f *fnCtx) operatorCall(op BinOp, lhs Expr, rhs ast.Expr, span source.Span) Expr {
	names := operatorTraits[op]
	trait := f.c.preludeTrait(names[0])
	if trait == nil || !f.implements(lhs.Type(), trait) {
		if declaredType(lhs.Type()) {
			f.errorf(span, "operator '%s' is not defined for '%s'; implement '%s' (fun %s(other: R): Out) to give it one (D71)", op, lhs.Type(), names[0], names[1])
		} else {
			// bool, a tuple, a list: nothing a program can implement for
			f.errorf(span, "operator '%s' is not defined for '%s'", op, lhs.Type())
		}
		f.checkExpr(rhs, nil)
		return bad()
	}
	return f.callOnChecked(lhs, names[1], []ast.Arg{{Value: rhs}}, span)
}

// negateCall lowers `-x` on a type implementing Negatable to `x.negate()`.
func (f *fnCtx) negateCall(x Expr, span source.Span) Expr {
	trait := f.c.preludeTrait("Negatable")
	if trait == nil || !f.implements(x.Type(), trait) {
		if declaredType(x.Type()) {
			f.errorf(span, "cannot negate a value of type '%s'; implement 'Negatable' (fun negate(): Out) to give it '-' (D71)", x.Type())
		} else {
			f.errorf(span, "cannot negate a value of type '%s'", x.Type())
		}
		return bad()
	}
	return f.callOnChecked(x, "negate", nil, span)
}

// declaredType reports whether a type is a struct or sealed trait, the
// kinds a program declares and so can give an operator (D71).
func declaredType(t types.Type) bool {
	switch t.(type) {
	case *types.Struct, *types.Sealed:
		return true
	}
	return false
}

// callOnChecked checks `recv.name(args)` where recv is already checked.
func (f *fnCtx) callOnChecked(recv Expr, name string, args []ast.Arg, span source.Span) Expr {
	holder := &ast.NameExpr{Name: "$operand", Pos: span}
	if f.boundPlace == nil {
		f.boundPlace = map[ast.Expr]Expr{}
	}
	f.boundPlace[holder] = recv
	defer delete(f.boundPlace, holder)
	call := &ast.CallExpr{Fun: &ast.MemberExpr{X: holder, Name: ast.Ident{Name: name, Pos: span}, Pos: span}, Args: args, Pos: span}
	return f.checkExpr(call, nil)
}
