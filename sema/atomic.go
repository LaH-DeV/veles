package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// std-only builtins behind the prelude's Atomic (D66): one processor
// instruction on a cell whose value fits a machine word, so an Atomic<int>
// never takes a lock. Bodies are checked per instantiation, so the cell's
// element type is concrete here; atomicLockFree says whether it qualifies,
// and the other four are only reached when it does (codegen emits
// `unreachable` for them otherwise).
var atomicBuiltins = map[string]int{ // name → argument count
	"atomicLockFree":       1,
	"atomicLoad":           1,
	"atomicStore":          2,
	"atomicSwap":           2,
	"atomicCompareAndSwap": 3,
}

// AtomicWord reports whether values of t are read and written with one
// atomic instruction: the integers, bool and the floats.
func AtomicWord(t types.Type) bool {
	if types.IsInteger(t) || types.IsFloat(t) {
		return true
	}
	b, ok := t.(*types.Basic)
	return ok && b.Kind == types.Bool
}

func (f *fnCtx) atomicCall(name string, e *ast.CallExpr) Expr {
	if len(e.Args) != atomicBuiltins[name] {
		f.errorf(e.Pos, "%s takes %d arguments", name, atomicBuiltins[name])
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	cell := f.checkExpr(e.Args[0].Value, nil)
	p, ok := cell.Type().(*types.Pointer)
	if !ok {
		f.errorf(e.Pos, "%s takes a pointer", name)
		f.checkArgsLoosely(e.Args[1:])
		return bad()
	}
	args := []Expr{cell}
	for _, a := range e.Args[1:] {
		args = append(args, f.checkExprTo(a.Value, p.Elem))
	}
	var t types.Type
	switch name {
	case "atomicLockFree", "atomicCompareAndSwap":
		t = types.TBool
	case "atomicStore":
		t = types.TUnit
	default:
		t = p.Elem
	}
	return &Builtin{exprBase{t}, name, args, e.Pos}
}
