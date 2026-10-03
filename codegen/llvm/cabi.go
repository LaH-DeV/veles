package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// The C calling convention for an `extern struct` passed or returned by
// value (plan A8). LLVM does not implement it: a first-class aggregate in
// a `call` is split into registers by LLVM's own rules, which match no C
// compiler's, so clang rewrites every such signature in its front end. This
// file does what clang does, per target, checked against `clang -S
// -emit-llvm` of the same C declarations (driver TestCStructsByValue):
//
//   - Windows x64: a struct of 1, 2, 4 or 8 bytes travels as an integer of
//     that size; any other is passed as a pointer to a copy the caller
//     makes, and returned through a hidden pointer (`sret`).
//   - SysV x86-64 (Linux, Intel macOS): up to 16 bytes, each eightbyte is
//     classed INTEGER (any non-float in it) or SSE (floats only) and
//     travels in the matching registers — when enough of them are left,
//     else the whole struct goes on the stack (`byval`); larger structs
//     are `byval` and returned through `sret`.
//   - AAPCS64 (Linux and macOS on ARM): one to four floats or doubles of a
//     kind (a homogeneous float aggregate) travel in vector registers;
//     other structs up to 16 bytes as one or two 64-bit integers; larger
//     ones as a pointer to a copy, returned through `sret`.
//
// Scalars, strings (std's pointer-and-length pair) and lists keep their
// existing lowering; only an `extern struct` is rewritten.

type cABI int

const (
	abiWin64 cABI = iota
	abiSysV
	abiAAPCS64
)

// currentABI is the convention code is generated for; a test classifies the
// same shapes for every target.
var currentABI = targetABI

func targetABI() cABI {
	t := targetTriple()
	switch {
	case strings.HasPrefix(t, "aarch64"), strings.HasPrefix(t, "arm64"):
		return abiAAPCS64
	case strings.Contains(t, "windows"):
		return abiWin64
	}
	return abiSysV
}

// cPass is how one value crosses.
type cPass int

const (
	cDirect   cPass = iota // as its own LLVM type
	cCoerce                // stored to memory and reloaded as another type
	cIndirect              // a pointer to a copy (an argument), or `sret` (a result)
	cByval                 // SysV memory class: a pointer marked `byval`
)

type cValue struct {
	pass  cPass
	ty    string // the LLVM type that crosses for cCoerce
	llt   string // the value's own LLVM type
	align int
	mem   bool // the value is in the memory class (mem.go): a register holds its address
	size  int
}

type cSig struct {
	params []cValue
	ret    cValue // cIndirect: returned through a leading `sret` pointer
	void   bool
}

// isCStruct reports a value the C ABI must rewrite.
// isCStruct: a struct C sees as an aggregate — an extern struct or union —
// or as its one field, a @transparent struct (D120); either crosses through
// cClassify.
func isCStruct(t types.Type) bool {
	s, ok := t.(*types.Struct)
	return ok && (s.Extern || s.Transparent)
}

// cSignature classifies a C function's parameters and result.
func (g *gen) cSignature(params []types.Type, ret types.Type) cSig {
	abi := currentABI()
	var sig cSig
	gpr, sse := 6, 8 // SysV argument registers still free
	sig.void = ret == nil || types.IsUnit(ret) || types.IsNever(ret)
	if !sig.void {
		sig.ret = g.cClassify(abi, ret, true, &gpr, &sse)
		if sig.ret.pass == cIndirect {
			gpr-- // the sret pointer takes the first integer register
		}
	}
	for _, p := range params {
		sig.params = append(sig.params, g.cClassify(abi, p, false, &gpr, &sse))
	}
	return sig
}

func (g *gen) cClassify(abi cABI, t types.Type, ret bool, gpr, sse *int) cValue {
	if st, ok := t.(*types.Struct); ok && st.Transparent && len(st.Fields) == 1 {
		// passed exactly as its field (D120): the field's class, the struct's bytes
		v := g.cClassify(abi, st.Fields[0].Type, ret, gpr, sse)
		if v.pass == cDirect {
			v.pass, v.ty = cCoerce, v.llt
		}
		v.llt = g.llType(t)
		return v
	}
	llt := g.llType(t)
	if !isCStruct(t) {
		// a scalar spends one register of its class (SysV counting only)
		switch {
		case types.IsString(t):
			*gpr -= 2
		case isFloat(t):
			*sse--
		default:
			*gpr--
		}
		return cValue{pass: cDirect, llt: llt}
	}
	size, align := g.layout(t)
	v := cValue{llt: llt, align: align, mem: g.isMem(t), size: size}
	switch abi {
	case abiWin64:
		switch size {
		case 1, 2, 4, 8:
			v.pass, v.ty = cCoerce, fmt.Sprintf("i%d", size*8)
		default:
			v.pass = cIndirect
		}
	case abiAAPCS64:
		if kind, n := g.cHomogeneousFloats(t); n > 0 {
			v.pass = cCoerce
			if ret {
				// returned as a flat struct of the floats, as clang does
				v.ty = "{ " + strings.TrimSuffix(strings.Repeat(kind+", ", n), ", ") + " }"
			} else {
				v.ty = fmt.Sprintf("[%d x %s]", n, kind)
			}
			return v
		}
		switch {
		case size > 16:
			v.pass = cIndirect
		case size > 8:
			v.pass, v.ty = cCoerce, "[2 x i64]"
		case ret:
			v.pass, v.ty = cCoerce, fmt.Sprintf("i%d", size*8)
		default:
			v.pass, v.ty = cCoerce, "i64"
		}
	case abiSysV:
		if size > 16 {
			if ret {
				v.pass = cIndirect
			} else {
				v.pass = cByval
			}
			return v
		}
		ints, floats, parts := g.cEightbytes(t, size)
		if !ret {
			if ints > *gpr || floats > *sse {
				v.pass = cByval // all in registers or none
				return v
			}
			*gpr -= ints
			*sse -= floats
		}
		v.pass = cCoerce
		if len(parts) == 1 {
			v.ty = parts[0]
		} else {
			v.ty = "{ " + strings.Join(parts, ", ") + " }"
		}
	}
	return v
}

func isFloat(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && (b.Kind == types.F32 || b.Kind == types.F64)
}

// cScalar is one leaf of an extern struct, at its byte offset.
type cScalar struct {
	off, size int
	float     bool
}

func (g *gen) cScalars(t types.Type, base int, out []cScalar) []cScalar {
	if s, ok := t.(*types.Struct); ok && (s.Extern || s.Transparent) {
		for i, off := range g.layouter().StructPlan(s).Offsets {
			out = g.cScalars(s.Fields[i].Type, base+off, out)
		}
		return out
	}
	if a, ok := t.(*types.Array); ok {
		// C's `T x[N]`: N leaves; past a few there is no register class
		// for them to qualify for, so the bytes count as one integer (D121)
		es, _ := g.layout(a.Elem)
		if n := arrayLen(a); n <= 16 {
			for i := 0; i < int(n); i++ {
				out = g.cScalars(a.Elem, base+i*es, out)
			}
			return out
		}
		size, _ := g.layout(t)
		return append(out, cScalar{off: base, size: size})
	}
	size, _ := g.layout(t)
	return append(out, cScalar{off: base, size: size, float: isFloat(t)})
}

// cEightbytes classes each eightbyte of a struct of at most 16 bytes (SysV)
// and names the type each one travels as: an integer covering the bytes
// it holds, or `double`, `float` or `<2 x float>`.
func (g *gen) cEightbytes(t types.Type, size int) (ints, floats int, parts []string) {
	scalars := g.cScalars(t, 0, nil)
	for start := 0; start < size; start += 8 {
		end := min(start+8, size)
		var in []cScalar
		allFloat := true
		for _, s := range scalars {
			if s.off >= start && s.off < end {
				in = append(in, s)
				if !s.float {
					allFloat = false
				}
			}
		}
		if !allFloat || len(in) == 0 {
			ints++
			parts = append(parts, fmt.Sprintf("i%d", (end-start)*8))
			continue
		}
		floats++
		switch {
		case len(in) == 1 && in[0].size == 8:
			parts = append(parts, "double")
		case len(in) == 1:
			parts = append(parts, "float")
		default:
			parts = append(parts, "<2 x float>")
		}
	}
	return ints, floats, parts
}

// cHomogeneousFloats reports a struct made of one to four floats of one
// kind (AAPCS64's HFA): the LLVM type of the element and the count, or 0.
func (g *gen) cHomogeneousFloats(t types.Type) (string, int) {
	scalars := g.cScalars(t, 0, nil)
	if len(scalars) == 0 || len(scalars) > 4 {
		return "", 0
	}
	for _, s := range scalars {
		if !s.float || s.size != scalars[0].size {
			return "", 0
		}
	}
	if scalars[0].size == 8 {
		return "double", len(scalars)
	}
	return "float", len(scalars)
}

// sretDecl is the leading parameter of a signature returning through `sret`.
func (s cSig) sretDecl() string {
	return fmt.Sprintf("ptr sret(%s) align %d", s.ret.llt, s.ret.align)
}

func (p cValue) decl(ext string) string {
	switch p.pass {
	case cCoerce:
		return p.ty + ext // ext: a @transparent small integer's extension (D120)
	case cIndirect:
		return "ptr"
	case cByval:
		return fmt.Sprintf("ptr byval(%s) align %d", p.llt, p.align)
	}
	return p.llt + ext
}

// retDecl is the LLVM result type of a C signature.
func (s cSig) retDecl(llt, ext string) string {
	switch {
	case s.void || s.ret.pass == cIndirect:
		return "void"
	case s.ret.pass == cCoerce:
		return ext + s.ret.ty
	}
	return ext + llt
}

// cArg lowers one argument value for a call: what the call passes.
func (g *gen) cArg(p cValue, v, ext string) string {
	switch p.pass {
	case cCoerce:
		slot := g.alloca(p.ty)
		g.emit("store %s %s, ptr %s", p.llt, v, slot)
		c := g.newTmp()
		g.emit("%s = load %s, ptr %s", c, p.ty, slot)
		return p.ty + ext + " " + c
	case cIndirect, cByval:
		slot := g.alloca(p.llt)
		if p.mem {
			g.copyMem(v, slot, p.size) // the callee may change its copy
		} else {
			g.emit("store %s %s, ptr %s", p.llt, v, slot)
		}
		if p.pass == cByval {
			return fmt.Sprintf("ptr byval(%s) align %d %s", p.llt, p.align, slot)
		}
		return "ptr " + slot
	}
	return p.llt + ext + " " + v
}

// cCall emits a call through a C signature (`callee` is `@name` or a
// pointer value) and returns the result as a value of its own type.
func (g *gen) cCall(sig cSig, callee, retLL, retExt string, args []string) string {
	var sret string
	if !sig.void && sig.ret.pass == cIndirect {
		sret = g.alloca(sig.ret.llt)
		args = append([]string{fmt.Sprintf("ptr sret(%s) align %d %s", sig.ret.llt, sig.ret.align, sret)}, args...)
	}
	rd := sig.retDecl(retLL, retExt)
	if rd == "void" {
		g.emit("call void %s(%s)", callee, joinArgs(args))
		if sret == "" {
			return "zeroinitializer"
		}
		if sig.ret.mem {
			return sret // the result is in memory already
		}
		v := g.newTmp()
		g.emit("%s = load %s, ptr %s", v, sig.ret.llt, sret)
		return v
	}
	r := g.newTmp()
	g.emit("%s = call %s %s(%s)", r, rd, callee, joinArgs(args))
	if sig.ret.pass != cCoerce {
		return r
	}
	slot := g.alloca(sig.ret.ty)
	g.emit("store %s %s, ptr %s", sig.ret.ty, r, slot)
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, sig.ret.llt, slot)
	return v
}
