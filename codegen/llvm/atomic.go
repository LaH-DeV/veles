package llvm

import (
	"fmt"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// atomic lowers the prelude's Atomic builtins (D66): a value that fits a
// machine word is read, written, swapped and compared-and-swapped with one
// sequentially consistent instruction on its integer bits — a bool as a
// byte, a float as the integer of its width, so a compare is bitwise.
func (g *gen) atomic(e *sema.Builtin) string {
	elem := e.Args[0].Type().(*types.Pointer).Elem
	if e.Op == "atomicLockFree" {
		if sema.AtomicWord(elem) {
			return "true"
		}
		return "false"
	}
	if !sema.AtomicWord(elem) {
		// behind `if (atomicLockFree(cell))`: never reached
		for _, a := range e.Args {
			g.expr(a)
		}
		g.emitTerm("unreachable")
		return "zeroinitializer"
	}
	vt := g.llType(elem)
	it, size := atomicInt(elem)
	cell := g.expr(e.Args[0])
	bits := func(v string) string { // the value as the integer stored
		switch {
		case vt == it:
			return v
		case vt == "i1":
			t := g.newTmp()
			g.emit("%s = zext i1 %s to i8", t, v)
			return t
		default:
			t := g.newTmp()
			g.emit("%s = bitcast %s %s to %s", t, vt, v, it)
			return t
		}
	}
	value := func(b string) string { // the integer loaded as the value
		switch {
		case vt == it:
			return b
		case vt == "i1":
			t := g.newTmp()
			g.emit("%s = trunc i8 %s to i1", t, b)
			return t
		default:
			t := g.newTmp()
			g.emit("%s = bitcast %s %s to %s", t, it, b, vt)
			return t
		}
	}
	align := fmt.Sprintf("align %d", size)
	switch e.Op {
	case "atomicLoad":
		b := g.newTmp()
		g.emit("%s = load atomic %s, ptr %s seq_cst, %s", b, it, cell, align)
		return value(b)
	case "atomicStore":
		v := bits(g.expr(e.Args[1]))
		g.emit("store atomic %s %s, ptr %s seq_cst, %s", it, v, cell, align)
		return "zeroinitializer"
	case "atomicSwap":
		v := bits(g.expr(e.Args[1]))
		b := g.newTmp()
		g.emit("%s = atomicrmw xchg ptr %s, %s %s seq_cst, %s", b, cell, it, v, align)
		return value(b)
	default: // atomicCompareAndSwap
		want := bits(g.expr(e.Args[1]))
		next := bits(g.expr(e.Args[2]))
		pair := g.newTmp()
		g.emit("%s = cmpxchg ptr %s, %s %s, %s %s seq_cst seq_cst, %s", pair, cell, it, want, it, next, align)
		ok := g.newTmp()
		g.emit("%s = extractvalue { %s, i1 } %s, 1", ok, it, pair)
		return ok
	}
}

// atomicInt is the integer type an atomic value is stored as, and its size.
func atomicInt(t types.Type) (string, int) {
	switch t.(*types.Basic).Kind {
	case types.Bool, types.I8, types.U8:
		return "i8", 1
	case types.I16, types.U16:
		return "i16", 2
	case types.I32, types.U32, types.F32:
		return "i32", 4
	default:
		return "i64", 8
	}
}
