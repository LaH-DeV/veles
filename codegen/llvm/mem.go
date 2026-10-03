package llvm

import (
	"fmt"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// The memory class (D121). A value of an aggregate type that is at least
// memThreshold bytes and holds an array — a big `Array`, a struct or tuple
// with one in it, a nullable or sealed value around one — is never an LLVM
// SSA value: LLVM scalarises a first-class aggregate element by element, so
// a copy of a kilobyte array would be a kilobyte of instructions and a
// megabyte one would not compile. Such a value lives in memory:
//
//   - a register that stands for it holds its address (vt is "ptr");
//   - a copy is a memmove, a store of it a memmove into the destination,
//     a read of it the address itself — `loadVal` returns the place, so a
//     read costs nothing and the user of the value copies where one is due;
//   - an argument is the address of a copy the caller made for the call, a
//     result is written through a hidden `sret` pointer;
//   - an expression that builds a value (a literal, a call) builds it in
//     storage of its own, and one that names a place (a variable, a field)
//     yields the place: whoever keeps the value copies it (exprOwned).
//
// Everything smaller keeps its SSA form and its IR, byte for byte.

const memThreshold = 128

// isMem reports whether values of t are in the memory class.
func (g *gen) isMem(t types.Type) bool {
	switch t.(type) {
	case *types.Array, *types.Struct, *types.Tuple, *types.Nullable, *types.Sealed, *types.ErrorUnion:
	default:
		return false
	}
	if g.memo == nil {
		g.memo = map[string]bool{}
	}
	key := types.Key(t)
	if v, ok := g.memo[key]; ok {
		return v
	}
	g.memo[key] = false // a type that contains itself by value was refused earlier
	size, _ := g.layout(t)
	v := size > memThreshold && g.holdsArray(t, map[string]bool{})
	g.memo[key] = v
	return v
}

// holdsArray: an array is, or is somewhere inside, a value of t.
func (g *gen) holdsArray(t types.Type, seen map[string]bool) bool {
	switch t := t.(type) {
	case *types.Array:
		return true
	case *types.Nullable:
		return g.holdsArray(t.Elem, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if g.holdsArray(e, seen) {
				return true
			}
		}
	case *types.Struct:
		key := types.Key(t)
		if seen[key] {
			return false
		}
		seen[key] = true
		for _, f := range t.Fields {
			if g.holdsArray(f.Type, seen) {
				return true
			}
		}
	case *types.Sealed:
		key := types.Key(t)
		if seen[key] {
			return false
		}
		seen[key] = true
		for _, v := range t.Variants {
			if g.holdsArray(v, seen) {
				return true
			}
		}
	case *types.ErrorUnion:
		for _, m := range t.Members {
			if g.holdsArray(m, seen) {
				return true
			}
		}
	}
	return false
}

// vt is the type of a register holding a value of t: the value's own type,
// or for the memory class the address of its storage.
func (g *gen) vt(t types.Type) string {
	if g.isMem(t) {
		return "ptr"
	}
	return g.llType(t)
}

// memSize is the byte size of a value of t.
func (g *gen) memSize(t types.Type) int {
	s, _ := g.layout(t)
	return s
}

// copyMem copies size bytes from src to dst. memmove, since the source of
// an assignment may overlap its destination (`a = a`, `x.a = x.a`).
func (g *gen) copyMem(src, dst string, size int) {
	if src == dst || size == 0 {
		return
	}
	g.emit("call void @llvm.memmove.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", dst, src, size)
}

// storeVal puts the value v of type t into the storage at p.
func (g *gen) storeVal(t types.Type, v, p string) {
	if g.isMem(t) {
		g.copyMem(v, p, g.memSize(t))
		return
	}
	g.emit("store %s %s, ptr %s", g.llType(t), v, p)
}

// loadVal is the value of type t stored at p: loaded, or for the memory
// class p itself.
func (g *gen) loadVal(t types.Type, p string) string {
	if g.isMem(t) {
		return p
	}
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, g.llType(t), p)
	return v
}

// newMem is fresh, uninitialised storage for a value of t.
func (g *gen) newMem(t types.Type) string {
	return g.alloca(g.llType(t))
}

// zeroMem is fresh storage holding t's all-zero value.
func (g *gen) zeroMem(t types.Type) string {
	p := g.newMem(t)
	g.emit("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", p, g.memSize(t))
	return p
}

// ownCopy is storage of its own holding a copy of the value v of type t.
func (g *gen) ownCopy(t types.Type, v string) string {
	p := g.newMem(t)
	g.copyMem(v, p, g.memSize(t))
	return p
}

// freshMem: evaluating e builds its value in storage nothing else refers
// to, so a holder can keep that storage instead of copying it.
func freshMem(e sema.Expr) bool {
	switch e := e.(type) {
	case *sema.ListLit, *sema.StructLit, *sema.TupleLit, *sema.Call, *sema.CallIndirect, *sema.CallVirtual,
		*sema.SomeWrap, *sema.MakeVariant, *sema.NullConst, *sema.Zero:
		return true
	case *sema.Builtin:
		return len(e.Op) > 6 && e.Op[:6] == "array." || e.Op == "list.toArray"
	}
	return false
}

// exprOwned evaluates e to a value in storage that is the caller's own: a
// fresh result as it is, a place (a variable, a field, an element) copied.
// An argument, a stored value and a returned one are made this way.
func (g *gen) exprOwned(e sema.Expr) string {
	v := g.expr(e)
	if !g.isMem(e.Type()) || freshMem(e) {
		return v
	}
	return g.ownCopy(e.Type(), v)
}

// fieldPtr is the address of element idx of the aggregate stored at base.
func (g *gen) fieldPtr(aggLL, base string, idx int) string {
	p := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, aggLL, base, idx)
	return p
}

// part is element idx of the aggregate value v of type t, of type et: an
// extractvalue for an SSA aggregate, the address (or a load) of the element
// for one in memory.
func (g *gen) part(t types.Type, v string, idx int, et types.Type) string {
	if g.isMem(t) {
		return g.loadVal(et, g.fieldPtr(g.llType(t), v, idx))
	}
	x := g.newTmp()
	g.emit("%s = extractvalue %s %s, %d", x, g.llType(t), v, idx)
	return x
}

// spill is the address of the value v of type t: v itself for the memory
// class, else a temporary holding it (for code that needs an address).
func (g *gen) spill(t types.Type, v string) string {
	if g.isMem(t) {
		return v
	}
	p := g.newMem(t)
	g.emit("store %s %s, ptr %s", g.llType(t), v, p)
	return p
}

// sretParam is the leading parameter of a function returning a memory-class
// value: where it writes the result.
func (g *gen) sretParam(t types.Type) string {
	_, align := g.layout(t)
	if a := g.llAlign(t); a > align {
		align = a
	}
	return fmt.Sprintf("ptr noalias sret(%s) align %d", g.llType(t), max(align, 1))
}

// fnRet is the logical result type of fn as its body returns it: the Result
// of a throwing function, nil for none.
func (g *gen) fnRet(fn *sema.Func) types.Type {
	switch {
	case fn.Sig.Effects.Throws:
		return g.resultOf(fn)
	case types.IsUnit(fn.Sig.Ret) || types.IsNever(fn.Sig.Ret):
		return nil
	}
	return fn.Sig.Ret
}

// sigRet is fnRet for a function type: what a call of a value of it yields.
func (g *gen) sigRet(sig *types.Func) types.Type {
	switch {
	case sig.Effects.Throws:
		return g.prog.ResultType(sig.Ret, sig.Effects.Error)
	case types.IsUnit(sig.Ret) || types.IsNever(sig.Ret):
		return nil
	}
	return sig.Ret
}

// emitRet returns the value v of type t (nil: nothing) from the function
// being emitted: through the `sret` pointer in the memory class, as the
// coroutine's result in a task.
func (g *gen) emitRet(t types.Type, v string, isResult bool) {
	if g.coro != nil {
		g.coroReturn(t, v, isResult)
		return
	}
	switch {
	case t == nil:
		g.emitTerm("ret void")
	case g.isMem(t):
		g.storeVal(t, v, "%sret")
		g.emitTerm("ret void")
	default:
		g.emitTerm("ret %s %s", g.llType(t), v)
	}
}

// tagOf is the tag of the sealed or error-union value v of type t.
func (g *gen) tagOf(t types.Type, v string) string {
	r := g.newTmp()
	if g.isMem(t) {
		p := g.fieldPtr(g.llType(t), v, 0)
		g.emit("%s = load i32, ptr %s", r, p)
		return r
	}
	g.emit("%s = extractvalue %s %s, 0", r, g.llType(t), v)
	return r
}

// makeTaggedT builds the value of the sealed type or error union t that
// holds payload (a value of payloadT) under tag.
func (g *gen) makeTaggedT(t types.Type, tag int, payloadT types.Type, payload string) string {
	if !g.isMem(t) {
		return g.makeTagged(g.llType(t), tag, g.llType(payloadT), payload)
	}
	tmp := g.newMem(t)
	g.emit("store i32 %d, ptr %s", tag, g.fieldPtr(g.llType(t), tmp, 0))
	if g.llType(payloadT) != "{}" {
		g.storeVal(payloadT, payload, g.fieldPtr(g.llType(t), tmp, 1))
	}
	return tmp
}

// extractTaggedT reads the payload of the tagged value v of type t as a
// value of payloadT.
func (g *gen) extractTaggedT(t types.Type, v string, payloadT types.Type) string {
	if !g.isMem(t) {
		return g.extractTagged(g.llType(t), v, g.llType(payloadT))
	}
	if g.llType(payloadT) == "{}" {
		return "zeroinitializer"
	}
	return g.loadVal(payloadT, g.fieldPtr(g.llType(t), v, 1))
}

// nullFlag is the presence flag of the nullable value v (not pointer-like).
func (g *gen) nullFlag(nt *types.Nullable, v string) string {
	r := g.newTmp()
	if g.isMem(nt) {
		g.emit("%s = load i1, ptr %s", r, g.fieldPtr(g.llType(nt), v, 0))
		return r
	}
	g.emit("%s = extractvalue %s %s, 0", r, g.llType(nt), v)
	return r
}

// nullVal is the value inside the nullable value v (not pointer-like).
func (g *gen) nullVal(nt *types.Nullable, v string) string {
	if g.isMem(nt) {
		return g.loadVal(nt.Elem, g.fieldPtr(g.llType(nt), v, 1))
	}
	r := g.newTmp()
	g.emit("%s = extractvalue %s %s, 1", r, g.llType(nt), v)
	return r
}

// callRet emits a call of callee and returns the value of retT it yields:
// "zeroinitializer" for none; a memory-class result is written through a
// leading sret argument into storage that is then the value. callee is
// @name or a register holding the code pointer; args are rendered
// "type value".
func (g *gen) callRet(retT types.Type, callee string, args []string) string {
	switch {
	case retT == nil:
		g.emit("call void %s(%s)", callee, joinArgs(args))
		return "zeroinitializer"
	case g.isMem(retT):
		tmp := g.newMem(retT)
		g.emit("call void %s(%s)", callee, joinArgs(append([]string{g.sretParam(retT) + " " + tmp}, args...)))
		return tmp
	}
	v := g.newTmp()
	g.emit("%s = call %s %s(%s)", v, g.llType(retT), callee, joinArgs(args))
	return v
}

// zeroSlot is fresh storage for a t, zeroed: where a runtime call that
// fills it in (a channel receive) starts from.
func (g *gen) zeroSlot(t types.Type) string {
	if g.isMem(t) {
		return g.zeroMem(t)
	}
	slot := g.alloca(g.llType(t))
	g.emit("store %s zeroinitializer, ptr %s", g.llType(t), slot)
	return slot
}

// slotOf is the address of a value of t for a runtime call that reads it:
// the value itself in the memory class, else a temporary holding it.
func (g *gen) slotOf(t types.Type, v string) string {
	if g.isMem(t) {
		return v
	}
	slot := g.alloca(g.llType(t))
	g.emit("store %s %s, ptr %s", g.llType(t), v, slot)
	return slot
}
