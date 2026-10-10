package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// std-only builtins behind the prelude's Atomic (D66, D144): one processor
// instruction on a cell whose value fits a machine word, so an Atomic<int>
// never takes a lock. Bodies are checked per instantiation, so the cell's
// element type is concrete here; atomicLockFree says whether it qualifies,
// and the others are only reached when it does (codegen emits `unreachable`
// for them otherwise). Each takes its memory orders last (D144).
var atomicBuiltins = map[string]struct{ values, orders int }{
	"atomicLockFree":        {0, 0},
	"atomicLoad":            {0, 1},
	"atomicStore":           {1, 1},
	"atomicSwap":            {1, 1},
	"atomicCompareAndSwap":  {2, 2}, // expected, new; success, failure
	"atomicCompareExchange": {2, 2},
	// the integer read-modify-writes: each returns the old value
	"atomicAdd": {1, 1},
	"atomicSub": {1, 1},
	"atomicAnd": {1, 1},
	"atomicOr":  {1, 1},
	"atomicXor": {1, 1},
}

// AtomicWord reports whether values of t are read and written with one
// atomic instruction: the integers, bool, the floats, and a pointer or a
// nullable pointer (one machine word, D144); an enum is its integer (D57).
func AtomicWord(t types.Type) bool {
	t = types.Underlying(t)
	if types.IsInteger(t) || types.IsFloat(t) {
		return true
	}
	switch t := t.(type) {
	case *types.Basic:
		return t.Kind == types.Bool
	case *types.Pointer:
		return true
	case *types.Nullable:
		_, ok := t.Elem.(*types.Pointer)
		return ok
	}
	return false
}

func (f *fnCtx) atomicCall(name string, e *ast.CallExpr) Expr {
	shape := atomicBuiltins[name]
	if len(e.Args) != 1+shape.values+shape.orders {
		f.errorf(e.Pos, "%s takes %d arguments", name, 1+shape.values+shape.orders)
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
	for _, a := range e.Args[1 : 1+shape.values] {
		args = append(args, f.checkExprTo(a.Value, p.Elem))
	}
	order := f.c.preludeType("MemoryOrder")
	for _, a := range e.Args[1+shape.values:] {
		args = append(args, f.checkExprTo(a.Value, order))
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

// The memory orders, by their MemoryOrder case (D144).
var memoryOrders = []string{"Relaxed", "Acquire", "Release", "AcqRel", "SeqCst"}

// atomicOrders says which orders each Atomic method takes in `order:`; a
// method missing here takes all five.
var atomicOrders = map[string][]string{
	"load":  {"Relaxed", "Acquire", "SeqCst"},
	"store": {"Relaxed", "Release", "SeqCst"},
}

// checkMemoryOrders refuses, at a call of an Atomic method, an order the
// operation cannot take when it is written as a MemoryOrder case (D144): a
// load does not release and a store does not acquire, and a failed
// compare-and-set is a load no stronger than its `order:`. An order known
// only when the program runs is not checked; one invalid for the operation
// then acts as SeqCst.
func (f *fnCtx) checkMemoryOrders(t *FuncTemplate, e *ast.CallExpr) {
	if t.Module == nil || t.Module.Path != "std/prelude" || !f.atomicMethod(t) {
		return
	}
	success := "SeqCst"
	for _, a := range e.Args {
		if a.Name == nil || (a.Name.Name != "order" && a.Name.Name != "failure") {
			continue
		}
		o := f.orderCase(a.Value)
		if o == "" {
			continue
		}
		if a.Name.Name == "order" {
			success = o
			if valid, ok := atomicOrders[t.Name]; ok && !containsString(valid, o) {
				f.errorf(a.Value.Span(), "MemoryOrder.%s is not an order for %s; use %s (D144)", o, article(t.Name), orList(valid))
			}
			continue
		}
		valid := []string{"Relaxed", "Acquire", "SeqCst"}
		if !containsString(valid, o) {
			f.errorf(a.Value.Span(), "MemoryOrder.%s is not an order for the load of a failed %s; use %s (D144)", o, t.Name, orList(valid))
		}
	}
	for _, a := range e.Args {
		if a.Name == nil || a.Name.Name != "failure" {
			continue
		}
		if o := f.orderCase(a.Value); o != "" && strength(o) > strength(success) {
			f.errorf(a.Value.Span(), "the failure order MemoryOrder.%s is stronger than the order MemoryOrder.%s; a failed %s is a load at most as strong as the operation (D144)", o, success, t.Name)
		}
	}
}

// atomicMethod reports whether t is a method of the prelude's Atomic, in its
// body or in an extend block of one of its instances.
func (f *fnCtx) atomicMethod(t *FuncTemplate) bool {
	if t.Owner != nil {
		return t.Owner.Name == "Atomic"
	}
	if t.Impl != nil && t.Impl.Trait == nil {
		st, ok := t.Impl.Target.(*types.Struct)
		return ok && st.Name == "Atomic"
	}
	return false
}

// orderCase is the MemoryOrder case an argument names, written
// `MemoryOrder.Acquire`, or "" when it is any other expression.
func (f *fnCtx) orderCase(x ast.Expr) string {
	m, ok := x.(*ast.MemberExpr)
	if !ok {
		return ""
	}
	id, ok := m.X.(*ast.NameExpr)
	if !ok || id.Name != "MemoryOrder" {
		return ""
	}
	if sym := f.lookup(id.Name); sym == nil || sym.Type != f.c.preludeType("MemoryOrder") && f.c.symType(sym) != f.c.preludeType("MemoryOrder") {
		return ""
	}
	if !containsString(memoryOrders, m.Name.Name) {
		return ""
	}
	return m.Name.Name
}

// strength orders the loads a failed compare-and-set may make: Release adds
// nothing to a load, AcqRel acquires.
func strength(o string) int {
	switch o {
	case "Acquire", "AcqRel":
		return 1
	case "SeqCst":
		return 2
	}
	return 0
}

func article(method string) string {
	switch method {
	case "load":
		return "a load"
	case "store":
		return "a store"
	}
	return method
}

func orList(xs []string) string {
	return strings.Join(xs[:len(xs)-1], ", ") + " or " + xs[len(xs)-1]
}
