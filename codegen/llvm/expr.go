package llvm

import (
	"fmt"
	"math"
	"runtime"
	"strconv"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

func targetTriple() string {
	switch runtime.GOOS {
	case "windows":
		return "x86_64-w64-windows-gnu"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "arm64-apple-macosx"
		}
		return "x86_64-apple-macosx"
	default:
		if runtime.GOARCH == "arm64" {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-pc-linux-gnu"
	}
}

// isPlace reports whether an expression denotes memory we can point at.
func isPlace(e sema.Expr) bool {
	switch e := e.(type) {
	case *sema.VarRef, *sema.Deref:
		return true
	case *sema.FieldGet:
		return isPlace(e.X)
	case *sema.TupleGet:
		return isPlace(e.X)
	case *sema.Builtin:
		return e.Op == "list.ref"
	}
	return false
}

// place returns a pointer to the storage an lvalue expression denotes.
func (g *gen) place(e sema.Expr) string {
	switch e := e.(type) {
	case *sema.VarRef:
		return g.varPtr(e.Var)
	case *sema.Deref:
		return g.expr(e.X)
	case *sema.FieldGet:
		base := g.place(e.X)
		p := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, g.llType(e.X.Type()), base, e.Index)
		return p
	case *sema.TupleGet:
		base := g.place(e.X)
		p := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, g.llType(e.X.Type()), base, e.Index)
		return p
	case *sema.Builtin:
		if e.Op == "list.ref" {
			list := g.expr(e.Args[0])
			idx := g.expr(e.Args[1])
			p := g.newTmp()
			g.emit("%s = call ptr @veles_list_ref(ptr %s, i64 %s)", p, list, idx)
			return p
		}
	}
	// Not a place: materialise into a temporary.
	v := g.expr(e)
	llt := g.llType(e.Type())
	tmp := g.alloca(llt)
	g.emit("store %s %s, ptr %s", llt, v, tmp)
	return tmp
}

// expr evaluates an expression and returns its SSA value. Unit-typed
// expressions return "zeroinitializer" (a `{}` constant).
func (g *gen) expr(e sema.Expr) string {
	switch e := e.(type) {
	case *sema.IntConst:
		if types.IsFloat(e.Type()) {
			return floatConst(float64(e.Value))
		}
		if e.Neg {
			return "-" + strconv.FormatUint(e.Value, 10)
		}
		return strconv.FormatUint(e.Value, 10)
	case *sema.FloatConst:
		return floatConst(e.Value)
	case *sema.BoolConst:
		if e.Value {
			return "true"
		}
		return "false"
	case *sema.StringConst:
		return g.stringConst(e.Value)
	case *sema.UnitConst:
		return "zeroinitializer"
	case *sema.NullConst:
		if isPtrLike(e.Type().(*types.Nullable).Elem) {
			return "null"
		}
		return "zeroinitializer"
	case *sema.VarRef:
		p := g.varPtr(e.Var)
		v := g.newTmp()
		g.emit("%s = load %s, ptr %s", v, g.llType(e.Type()), p)
		return v
	case *sema.Call:
		return g.call(e)
	case *sema.Binary:
		return g.binary(e)
	case *sema.Unary:
		x := g.expr(e.X)
		v := g.newTmp()
		if e.Op == sema.OpNot {
			g.emit("%s = xor i1 %s, true", v, x)
			return v
		}
		if types.IsFloat(e.Type()) {
			g.emit("%s = fneg %s %s", v, g.llType(e.Type()), x)
			return v
		}
		if g.prog.Release {
			g.emit("%s = sub %s 0, %s", v, g.llType(e.Type()), x)
			return v
		}
		return g.checkedArith("ssub", g.llType(e.Type()), "0", x, e.Span.String())
	case *sema.Cast:
		return g.cast(e)
	case *sema.ToString:
		x := g.expr(e.X)
		return g.show(e.X.Type(), x)
	case *sema.StringConcat:
		acc := g.expr(e.Parts[0])
		for _, part := range e.Parts[1:] {
			r := g.expr(part)
			ap, al := g.strPtrLen(acc)
			bp, bl := g.strPtrLen(r)
			out := g.alloca(strType)
			g.emit("call void @veles_string_concat(ptr %s, ptr %s, i64 %s, ptr %s, i64 %s)", out, ap, al, bp, bl)
			acc = g.newTmp()
			g.emit("%s = load %s, ptr %s", acc, strType, out)
		}
		return acc
	case *sema.FieldGet:
		if isPlace(e.X) {
			p := g.place(e)
			v := g.newTmp()
			g.emit("%s = load %s, ptr %s", v, g.llType(e.Type()), p)
			return v
		}
		x := g.expr(e.X)
		v := g.newTmp()
		g.emit("%s = extractvalue %s %s, %d", v, g.llType(e.X.Type()), x, e.Index)
		return v
	case *sema.TupleGet:
		x := g.expr(e.X)
		v := g.newTmp()
		g.emit("%s = extractvalue %s %s, %d", v, g.llType(e.X.Type()), x, e.Index)
		return v
	case *sema.StructLit:
		vals := make([]string, len(e.Fields))
		for i, f := range e.Fields {
			vals[i] = g.expr(f)
		}
		return g.buildStruct(e.Struct, vals)
	case *sema.TupleLit:
		llt := g.llType(e.Type())
		acc := "undef"
		for i, el := range e.Elems {
			v := g.expr(el)
			n := g.newTmp()
			g.emit("%s = insertvalue %s %s, %s %s, %d", n, llt, acc, g.llType(el.Type()), v, i)
			acc = n
		}
		return acc
	case *sema.AddrOf:
		if isPlace(e.X) {
			return g.place(e.X)
		}
		// box a temporary on the heap
		v := g.expr(e.X)
		size, _ := g.layout(e.X.Type())
		cell := g.newTmp()
		g.emit("%s = call ptr @veles_alloc(i64 %d)", cell, size)
		g.emit("store %s %s, ptr %s", g.llType(e.X.Type()), v, cell)
		return cell
	case *sema.Deref:
		p := g.expr(e.X)
		v := g.newTmp()
		g.emit("%s = load %s, ptr %s", v, g.llType(e.Type()), p)
		return v
	case *sema.SomeWrap:
		x := g.expr(e.X)
		nt := e.Type().(*types.Nullable)
		if isPtrLike(nt.Elem) {
			return x
		}
		llt := g.llType(nt)
		a := g.newTmp()
		g.emit("%s = insertvalue %s undef, i1 true, 0", a, llt)
		b := g.newTmp()
		g.emit("%s = insertvalue %s %s, %s %s, 1", b, llt, a, g.llType(nt.Elem), x)
		return b
	case *sema.IsNull:
		x := g.expr(e.X)
		nt := e.X.Type().(*types.Nullable)
		v := g.newTmp()
		if isPtrLike(nt.Elem) {
			g.emit("%s = icmp eq ptr %s, null", v, x)
			return v
		}
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(nt), x)
		g.emit("%s = xor i1 %s, true", v, tag)
		return v
	case *sema.Unwrap:
		x := g.expr(e.X)
		nt := e.X.Type().(*types.Nullable)
		if isPtrLike(nt.Elem) {
			return x
		}
		v := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", v, g.llType(nt), x)
		return v
	case *sema.MakeVariant:
		payload := g.expr(e.Value)
		return g.makeTagged(g.llType(e.Sealed), e.Variant.Tag, g.llType(e.Variant), payload)
	case *sema.VariantTest:
		x := g.expr(e.X)
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(e.X.Type()), x)
		v := g.newTmp()
		g.emit("%s = icmp eq i32 %s, %d", v, tag, e.Variant.Tag)
		return v
	case *sema.VariantCast:
		x := g.expr(e.X)
		return g.extractTagged(g.llType(e.X.Type()), x, g.llType(e.Variant))
	case *sema.UnionTest:
		x := g.expr(e.X)
		u := e.X.Type()
		idx := types.UnionIndex(u, e.Member)
		if _, isUnion := u.(*types.ErrorUnion); !isUnion {
			return "true"
		}
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(u), x)
		v := g.newTmp()
		g.emit("%s = icmp eq i32 %s, %d", v, tag, idx)
		return v
	case *sema.UnionCast:
		x := g.expr(e.X)
		u := e.X.Type()
		if _, isUnion := u.(*types.ErrorUnion); !isUnion {
			return x
		}
		return g.extractTagged(g.llType(u), x, g.llType(e.Member))
	case *sema.ErrorConvert:
		return g.errorConvert(e)
	case *sema.If:
		return g.ifExpr(e)
	case *sema.BlockExpr:
		return g.block(e.Block)
	case *sema.Match:
		return g.match(e)
	case *sema.Try:
		return g.try(e)
	case *sema.Throw:
		v := g.expr(e.Value)
		g.throwValue(v, e.From)
		return "undef"
	case *sema.Elvis:
		return g.elvis(e)
	case *sema.Let:
		v := g.expr(e.Init)
		st := g.declareVar(e.Var)
		g.emit("store %s %s, ptr %s", g.llType(e.Var.Type), v, st)
		return g.expr(e.Body)
	case *sema.ListLit:
		return g.listLit(e)
	case *sema.Builtin:
		return g.builtin(e)
	case *sema.RangeLit:
		lo := g.expr(e.Lo)
		hi := g.expr(e.Hi)
		llt := g.llType(e.Type())
		et := g.llType(e.Type().(*types.Range).Elem)
		a := g.newTmp()
		g.emit("%s = insertvalue %s undef, %s %s, 0", a, llt, et, lo)
		b := g.newTmp()
		g.emit("%s = insertvalue %s %s, %s %s, 1", b, llt, a, et, hi)
		c := g.newTmp()
		incl := "false"
		if e.Inclusive {
			incl = "true"
		}
		g.emit("%s = insertvalue %s %s, i1 %s, 2", c, llt, b, incl)
		return c
	case *sema.Panic:
		g.panicMsg(e.Message)
		return "undef"
	case *constExpr:
		return e.v
	case *sema.FuncRef:
		panic("codegen: function values are not supported")
	}
	panic(fmt.Sprintf("codegen: unsupported expression %T", e))
}

func floatConst(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return fmt.Sprintf("0x%016X", math.Float64bits(v))
	}
	// LLVM accepts the hexadecimal IEEE form for exactness.
	return fmt.Sprintf("0x%016X", math.Float64bits(v))
}

// buildStruct assembles a struct value from field values.
func (g *gen) buildStruct(st *types.Struct, vals []string) string {
	llt := g.llType(st)
	if len(vals) == 0 {
		return "zeroinitializer"
	}
	acc := "undef"
	for i, v := range vals {
		n := g.newTmp()
		g.emit("%s = insertvalue %s %s, %s %s, %d", n, llt, acc, g.llType(st.Fields[i].Type), v, i)
		acc = n
	}
	return acc
}

// makeTagged builds a `{ i32, [N x i64] }` value holding payload under tag.
func (g *gen) makeTagged(taggedLL string, tag int, payloadLL string, payload string) string {
	tmp := g.alloca(taggedLL)
	g.emit("store %s zeroinitializer, ptr %s", taggedLL, tmp)
	tagP := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 0", tagP, taggedLL, tmp)
	g.emit("store i32 %d, ptr %s", tag, tagP)
	if payloadLL != "{}" {
		payP := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", payP, taggedLL, tmp)
		g.emit("store %s %s, ptr %s", payloadLL, payload, payP)
	}
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, taggedLL, tmp)
	return v
}

// extractTagged reads the payload of a tagged value as payloadLL.
func (g *gen) extractTagged(taggedLL string, value string, payloadLL string) string {
	if payloadLL == "{}" {
		return "zeroinitializer"
	}
	tmp := g.alloca(taggedLL)
	g.emit("store %s %s, ptr %s", taggedLL, value, tmp)
	payP := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", payP, taggedLL, tmp)
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, payloadLL, payP)
	return v
}

// ---------------------------------------------------------------------------
// calls

func (g *gen) call(e *sema.Call) string {
	fn := e.Fn
	var args []string
	for i, a := range e.Args {
		v := g.expr(a)
		if fn.Extern && types.IsString(a.Type()) {
			p, l := g.strPtrLen(v)
			args = append(args, "ptr "+p, "i64 "+l)
			continue
		}
		_ = i
		args = append(args, g.llType(a.Type())+" "+v)
	}
	ret := g.retLL(fn)
	if fn.Extern && types.IsUnit(fn.Sig.Ret) {
		ret = "void"
	}
	if ret == "void" {
		g.emit("call void @%s(%s)", fn.Name, joinArgs(args))
		if types.IsNever(fn.Sig.Ret) {
			g.emitTerm("unreachable")
		}
		return "zeroinitializer"
	}
	v := g.newTmp()
	g.emit("%s = call %s @%s(%s)", v, ret, fn.Name, joinArgs(args))
	return v
}

func joinArgs(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += ", "
		}
		s += a
	}
	return s
}

// ---------------------------------------------------------------------------
// operators

func (g *gen) binary(e *sema.Binary) string {
	switch e.Op {
	case sema.OpAnd, sema.OpOr:
		return g.shortCircuit(e)
	}
	l := g.expr(e.L)
	r := g.expr(e.R)
	t := e.L.Type()
	llt := g.llType(t)
	v := g.newTmp()
	if types.IsString(t) {
		lp, ll := g.strPtrLen(l)
		rp, rl := g.strPtrLen(r)
		switch e.Op {
		case sema.OpEq:
			g.emit("%s = call i1 @veles_string_eq(ptr %s, i64 %s, ptr %s, i64 %s)", v, lp, ll, rp, rl)
			return v
		case sema.OpNe:
			eq := g.newTmp()
			g.emit("%s = call i1 @veles_string_eq(ptr %s, i64 %s, ptr %s, i64 %s)", eq, lp, ll, rp, rl)
			g.emit("%s = xor i1 %s, true", v, eq)
			return v
		default:
			c := g.newTmp()
			g.emit("%s = call i32 @veles_string_cmp(ptr %s, i64 %s, ptr %s, i64 %s)", c, lp, ll, rp, rl)
			pred := map[sema.BinOp]string{sema.OpLt: "slt", sema.OpLe: "sle", sema.OpGt: "sgt", sema.OpGe: "sge"}[e.Op]
			g.emit("%s = icmp %s i32 %s, 0", v, pred, c)
			return v
		}
	}
	if types.IsFloat(t) {
		switch e.Op {
		case sema.OpAdd:
			g.emit("%s = fadd %s %s, %s", v, llt, l, r)
		case sema.OpSub:
			g.emit("%s = fsub %s %s, %s", v, llt, l, r)
		case sema.OpMul:
			g.emit("%s = fmul %s %s, %s", v, llt, l, r)
		case sema.OpDiv:
			g.emit("%s = fdiv %s %s, %s", v, llt, l, r)
		default:
			pred := map[sema.BinOp]string{sema.OpEq: "oeq", sema.OpNe: "une", sema.OpLt: "olt", sema.OpLe: "ole", sema.OpGt: "ogt", sema.OpGe: "oge"}[e.Op]
			g.emit("%s = fcmp %s %s %s, %s", v, pred, llt, l, r)
		}
		return v
	}
	if types.IsInteger(t) || types.IsBool(t) {
		signed := types.IsSigned(t)
		switch e.Op {
		case sema.OpAdd, sema.OpSub, sema.OpMul:
			if g.prog.Release {
				op := map[sema.BinOp]string{sema.OpAdd: "add", sema.OpSub: "sub", sema.OpMul: "mul"}[e.Op]
				g.emit("%s = %s %s %s, %s", v, op, llt, l, r)
				return v
			}
			intr := map[sema.BinOp]string{sema.OpAdd: "add", sema.OpSub: "sub", sema.OpMul: "mul"}[e.Op]
			if signed {
				intr = "s" + intr
			} else {
				intr = "u" + intr
			}
			return g.checkedArith(intr, llt, l, r, e.Span.String())
		case sema.OpWrapAdd:
			g.emit("%s = add %s %s, %s", v, llt, l, r)
		case sema.OpWrapSub:
			g.emit("%s = sub %s %s, %s", v, llt, l, r)
		case sema.OpWrapMul:
			g.emit("%s = mul %s %s, %s", v, llt, l, r)
		case sema.OpDiv, sema.OpRem:
			g.divCheck(llt, l, r, signed, e.Span.String())
			op := "udiv"
			if e.Op == sema.OpRem {
				op = "urem"
			}
			if signed {
				op = "s" + op[1:]
			}
			g.emit("%s = %s %s %s, %s", v, op, llt, l, r)
		default:
			var pred string
			switch e.Op {
			case sema.OpEq:
				pred = "eq"
			case sema.OpNe:
				pred = "ne"
			case sema.OpLt:
				pred = "ult"
			case sema.OpLe:
				pred = "ule"
			case sema.OpGt:
				pred = "ugt"
			case sema.OpGe:
				pred = "uge"
			}
			if signed && len(pred) == 3 {
				pred = "s" + pred[1:]
			}
			g.emit("%s = icmp %s %s %s, %s", v, pred, llt, l, r)
		}
		return v
	}
	// aggregates and pointers: structural / identity equality
	switch e.Op {
	case sema.OpEq, sema.OpNe:
		eq := g.equal(t, l, r)
		if e.Op == sema.OpEq {
			return eq
		}
		g.emit("%s = xor i1 %s, true", v, eq)
		return v
	}
	panic("codegen: unsupported binary operator on " + t.String())
}

// checkedArith performs an overflow-checked operation (D21: checked in
// debug builds).
func (g *gen) checkedArith(intr, llt, l, r, where string) string {
	pair := g.newTmp()
	g.emit("%s = call { %s, i1 } @llvm.%s.with.overflow.%s(%s %s, %s %s)", pair, llt, intr, llt, llt, l, llt, r)
	v := g.newTmp()
	g.emit("%s = extractvalue { %s, i1 } %s, 0", v, llt, pair)
	of := g.newTmp()
	g.emit("%s = extractvalue { %s, i1 } %s, 1", of, llt, pair)
	bad, ok := g.newLabel("overflow"), g.newLabel("arith.ok")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", of, bad, ok)
	g.placeLabel(bad)
	g.panicMsg("integer overflow at " + where)
	g.placeLabel(ok)
	return v
}

func (g *gen) divCheck(llt, l, r string, signed bool, where string) {
	isZero := g.newTmp()
	g.emit("%s = icmp eq %s %s, 0", isZero, llt, r)
	bad, ok := g.newLabel("divzero"), g.newLabel("div.ok")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isZero, bad, ok)
	g.placeLabel(bad)
	g.panicMsg("division by zero at " + where)
	g.placeLabel(ok)
	if signed && !g.prog.Release {
		// MIN / -1 overflows
		bits := map[string]int{"i8": 8, "i16": 16, "i32": 32, "i64": 64}[llt]
		min := fmt.Sprintf("-%d", uint64(1)<<(bits-1))
		a := g.newTmp()
		g.emit("%s = icmp eq %s %s, %s", a, llt, l, min)
		b := g.newTmp()
		g.emit("%s = icmp eq %s %s, -1", b, llt, r)
		both := g.newTmp()
		g.emit("%s = and i1 %s, %s", both, a, b)
		bad2, ok2 := g.newLabel("divof"), g.newLabel("div.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", both, bad2, ok2)
		g.placeLabel(bad2)
		g.panicMsg("integer overflow at " + where)
		g.placeLabel(ok2)
	}
}

func (g *gen) shortCircuit(e *sema.Binary) string {
	res := g.alloca("i1")
	l := g.expr(e.L)
	g.emit("store i1 %s, ptr %s", l, res)
	rhs, end := g.newLabel("sc.rhs"), g.newLabel("sc.end")
	if e.Op == sema.OpAnd {
		g.emitTerm("br i1 %s, label %%%s, label %%%s", l, rhs, end)
	} else {
		g.emitTerm("br i1 %s, label %%%s, label %%%s", l, end, rhs)
	}
	g.placeLabel(rhs)
	r := g.expr(e.R)
	g.emit("store i1 %s, ptr %s", r, res)
	g.placeLabel(end)
	v := g.newTmp()
	g.emit("%s = load i1, ptr %s", v, res)
	return v
}

func (g *gen) cast(e *sema.Cast) string {
	x := g.expr(e.X)
	from, to := e.X.Type(), e.Type()
	if _, ok := from.(*types.Pointer); ok {
		return x // *T to *raw T
	}
	fl, tl := g.llType(from), g.llType(to)
	if fl == tl {
		return x
	}
	v := g.newTmp()
	fb, tb := types.BitSize(from), types.BitSize(to)
	switch {
	case types.IsInteger(from) && types.IsInteger(to):
		switch {
		case tb < fb:
			g.emit("%s = trunc %s %s to %s", v, fl, x, tl)
		case types.IsSigned(from):
			g.emit("%s = sext %s %s to %s", v, fl, x, tl)
		default:
			g.emit("%s = zext %s %s to %s", v, fl, x, tl)
		}
	case types.IsInteger(from) && types.IsFloat(to):
		if types.IsSigned(from) {
			g.emit("%s = sitofp %s %s to %s", v, fl, x, tl)
		} else {
			g.emit("%s = uitofp %s %s to %s", v, fl, x, tl)
		}
	case types.IsFloat(from) && types.IsInteger(to):
		if types.IsSigned(to) {
			g.emit("%s = fptosi %s %s to %s", v, fl, x, tl)
		} else {
			g.emit("%s = fptoui %s %s to %s", v, fl, x, tl)
		}
	case types.IsFloat(from) && types.IsFloat(to):
		if tb > fb {
			g.emit("%s = fpext %s %s to %s", v, fl, x, tl)
		} else {
			g.emit("%s = fptrunc %s %s to %s", v, fl, x, tl)
		}
	default:
		panic("codegen: unsupported cast " + from.String() + " to " + to.String())
	}
	return v
}

// ---------------------------------------------------------------------------
// control flow expressions

func (g *gen) ifExpr(e *sema.If) string {
	hasValue := !types.IsUnit(e.Type()) && !types.IsNever(e.Type())
	var res string
	llt := g.llType(e.Type())
	if hasValue {
		res = g.alloca(llt)
	}
	c := g.expr(e.Cond)
	thenL, endL := g.newLabel("if.then"), g.newLabel("if.end")
	elseL := endL
	if e.Else != nil {
		elseL = g.newLabel("if.else")
	}
	g.emitTerm("br i1 %s, label %%%s, label %%%s", c, thenL, elseL)
	g.placeLabel(thenL)
	v := g.block(e.Then)
	if hasValue && !g.term && e.Then.Value != nil {
		g.emit("store %s %s, ptr %s", llt, v, res)
	}
	if !g.term {
		g.emitTerm("br label %%%s", endL)
	}
	if e.Else != nil {
		g.placeLabel(elseL)
		v := g.block(e.Else)
		if hasValue && !g.term && e.Else.Value != nil {
			g.emit("store %s %s, ptr %s", llt, v, res)
		}
		if !g.term {
			g.emitTerm("br label %%%s", endL)
		}
	}
	g.placeLabel(endL)
	if types.IsNever(e.Type()) {
		g.emitTerm("unreachable")
		return "undef"
	}
	if !hasValue {
		return "zeroinitializer"
	}
	out := g.newTmp()
	g.emit("%s = load %s, ptr %s", out, llt, res)
	return out
}

func (g *gen) match(m *sema.Match) string {
	hasValue := !types.IsUnit(m.Type()) && !types.IsNever(m.Type())
	llt := g.llType(m.Type())
	var res string
	if hasValue {
		res = g.alloca(llt)
	}
	if m.Subject != nil {
		v := g.expr(m.Init)
		st := g.declareVar(m.Subject)
		g.emit("store %s %s, ptr %s", g.llType(m.Subject.Type), v, st)
	}
	endL := g.newLabel("when.end")
	next := g.newLabel("when.arm")
	g.emitTerm("br label %%%s", next)
	for _, arm := range m.Arms {
		g.placeLabel(next)
		next = g.newLabel("when.arm")
		bodyL := g.newLabel("when.body")
		if arm.Test != nil {
			t := g.expr(arm.Test)
			bindL := g.newLabel("when.bind")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", t, bindL, next)
			g.placeLabel(bindL)
		}
		for _, b := range arm.Binds {
			g.stmt(b)
		}
		if arm.Guard != nil {
			gv := g.expr(arm.Guard)
			g.emitTerm("br i1 %s, label %%%s, label %%%s", gv, bodyL, next)
		} else {
			g.emitTerm("br label %%%s", bodyL)
		}
		g.placeLabel(bodyL)
		v := g.block(arm.Body)
		if hasValue && !g.term && arm.Body.Value != nil {
			g.emit("store %s %s, ptr %s", llt, v, res)
		}
		if !g.term {
			g.emitTerm("br label %%%s", endL)
		}
	}
	g.placeLabel(next)
	if m.Exhaustive {
		g.panicMsg("unreachable: non-exhaustive match at " + m.Span.String())
	} else {
		g.emitTerm("br label %%%s", endL)
	}
	g.placeLabel(endL)
	if types.IsNever(m.Type()) {
		g.emitTerm("unreachable")
		return "undef"
	}
	if !hasValue {
		return "zeroinitializer"
	}
	out := g.newTmp()
	g.emit("%s = load %s, ptr %s", out, llt, res)
	return out
}

func (g *gen) elvis(e *sema.Elvis) string {
	nt := e.L.Type().(*types.Nullable)
	l := g.expr(e.L)
	llt := g.llType(e.Type())
	res := g.alloca(llt)
	isNull := g.newTmp()
	if isPtrLike(nt.Elem) {
		g.emit("%s = icmp eq ptr %s, null", isNull, l)
	} else {
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(nt), l)
		g.emit("%s = xor i1 %s, true", isNull, tag)
	}
	defL, someL, endL := g.newLabel("elvis.default"), g.newLabel("elvis.some"), g.newLabel("elvis.end")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isNull, defL, someL)
	g.placeLabel(someL)
	if isPtrLike(nt.Elem) {
		g.emit("store %s %s, ptr %s", llt, l, res)
	} else {
		inner := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", inner, g.llType(nt), l)
		g.emit("store %s %s, ptr %s", llt, inner, res)
	}
	g.emitTerm("br label %%%s", endL)
	g.placeLabel(defL)
	r := g.expr(e.R)
	if !g.term {
		g.emit("store %s %s, ptr %s", llt, r, res)
		g.emitTerm("br label %%%s", endL)
	}
	g.placeLabel(endL)
	out := g.newTmp()
	g.emit("%s = load %s, ptr %s", out, llt, res)
	return out
}

// ---------------------------------------------------------------------------
// errors (D4/D45)

// errorConvert re-tags an error value into a wider union.
func (g *gen) errorConvert(e *sema.ErrorConvert) string {
	x := g.expr(e.X)
	to := e.Type()
	from := e.From
	toU, toIsUnion := to.(*types.ErrorUnion)
	if !toIsUnion {
		return x // same single type
	}
	toLL := g.llType(toU)
	if _, fromIsUnion := from.(*types.ErrorUnion); !fromIsUnion {
		idx := types.UnionIndex(toU, from)
		return g.makeTagged(toLL, idx, g.llType(from), x)
	}
	// union to union: map tags, copy payload words
	fromU := from.(*types.ErrorUnion)
	fromLL := g.llType(fromU)
	oldTag := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", oldTag, fromLL, x)
	newTag := "0"
	for i, m := range fromU.Members {
		j := types.UnionIndex(toU, m)
		is := g.newTmp()
		g.emit("%s = icmp eq i32 %s, %d", is, oldTag, i)
		sel := g.newTmp()
		g.emit("%s = select i1 %s, i32 %d, i32 %s", sel, is, j, newTag)
		newTag = sel
	}
	src := g.alloca(fromLL)
	g.emit("store %s %s, ptr %s", fromLL, x, src)
	dst := g.alloca(toLL)
	g.emit("store %s zeroinitializer, ptr %s", toLL, dst)
	tagP := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 0", tagP, toLL, dst)
	g.emit("store i32 %s, ptr %s", newTag, tagP)
	sp := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", sp, fromLL, src)
	dp := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", dp, toLL, dst)
	words := g.unionWords(fromU)
	if w := g.unionWords(toU); w < words {
		words = w
	}
	g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", dp, sp, 8*words)
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, toLL, dst)
	return v
}

// throwValue returns Err(errValue) from the current throwing function.
func (g *gen) throwValue(errValue string, from types.Type) {
	rs := g.fnResult.(*types.Sealed)
	errVariant := rs.Variants[1]
	to := errVariant.Fields[0].Type
	conv := g.convertError(errValue, from, to)
	payload := g.buildStruct(errVariant, []string{conv})
	r := g.makeTagged(g.llType(rs), 1, g.llType(errVariant), payload)
	g.emitTerm("ret %s %s", g.llType(rs), r)
}

// constExpr wraps an already-evaluated SSA value as an expression.
type constExpr struct {
	v string
	t types.Type
}

func (c *constExpr) Type() types.Type { return c.t }

// convertError converts an evaluated error value between error types.
func (g *gen) convertError(v string, from, to types.Type) string {
	if types.Identical(from, to) {
		return v
	}
	if types.IsNever(to) {
		return "zeroinitializer"
	}
	e := &sema.ErrorConvert{X: &constExpr{v: v, t: from}, From: from}
	e.T = to
	return g.errorConvert(e)
}

func (g *gen) try(e *sema.Try) string {
	rs := e.X.Type().(*types.Sealed)
	rsLL := g.llType(rs)
	x := g.expr(e.X)
	tag := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", tag, rsLL, x)
	isErr := g.newTmp()
	g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
	errL, okL := g.newLabel("try.err"), g.newLabel("try.ok")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
	g.placeLabel(errL)
	errVariant := rs.Variants[1]
	payload := g.extractTagged(rsLL, x, g.llType(errVariant))
	errVal := "zeroinitializer"
	if g.llType(errVariant) != "{}" {
		errVal = g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", errVal, g.llType(errVariant), payload)
	}
	g.throwValue(errVal, errVariant.Fields[0].Type)
	g.placeLabel(okL)
	okVariant := rs.Variants[0]
	okPayload := g.extractTagged(rsLL, x, g.llType(okVariant))
	if g.llType(okVariant) == "{}" {
		return "zeroinitializer"
	}
	v := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", v, g.llType(okVariant), okPayload)
	return v
}

// ---------------------------------------------------------------------------
// builtins

func (g *gen) listLit(e *sema.ListLit) string {
	lt := e.Type().(*types.List)
	size, _ := g.layout(lt.Elem)
	list := g.newTmp()
	g.emit("%s = call ptr @veles_list_new(i64 %d, i64 %d)", list, size, len(e.Elems))
	if len(e.Elems) > 0 {
		et := g.llType(lt.Elem)
		tmp := g.alloca(et)
		for _, el := range e.Elems {
			v := g.expr(el)
			g.emit("store %s %s, ptr %s", et, v, tmp)
			g.emit("call void @veles_list_push(ptr %s, ptr %s)", list, tmp)
		}
	}
	return list
}

func (g *gen) builtin(e *sema.Builtin) string {
	switch e.Op {
	case "list.len":
		l := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call i64 @veles_list_len(ptr %s)", v, l)
		return v
	case "list.get":
		l := g.expr(e.Args[0])
		i := g.expr(e.Args[1])
		p := g.newTmp()
		g.emit("%s = call ptr @veles_list_ref(ptr %s, i64 %s)", p, l, i)
		v := g.newTmp()
		g.emit("%s = load %s, ptr %s", v, g.llType(e.Type()), p)
		return v
	case "list.ref":
		return g.place(e)
	case "list.push":
		l := g.expr(e.Args[0])
		x := g.expr(e.Args[1])
		et := g.llType(e.Args[1].Type())
		tmp := g.alloca(et)
		g.emit("store %s %s, ptr %s", et, x, tmp)
		g.emit("call void @veles_list_push(ptr %s, ptr %s)", l, tmp)
		return "zeroinitializer"
	case "list.pop":
		l := g.expr(e.Args[0])
		nt := e.Type().(*types.Nullable)
		et := g.llType(nt.Elem)
		tmp := g.alloca(et)
		g.emit("store %s zeroinitializer, ptr %s", et, tmp)
		ok := g.newTmp()
		g.emit("%s = call i1 @veles_list_pop(ptr %s, ptr %s)", ok, l, tmp)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, et, tmp)
		return g.makeNullable(nt, ok, val)
	case "list.copy":
		l := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call ptr @veles_list_copy(ptr %s)", v, l)
		return v
	case "list.clear":
		l := g.expr(e.Args[0])
		g.emit("call void @veles_list_clear(ptr %s)", l)
		return "zeroinitializer"
	case "string.len":
		s := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", v, strType, s)
		return v
	case "string.startsWith", "string.endsWith", "string.contains":
		a := g.expr(e.Args[0])
		b := g.expr(e.Args[1])
		ap, al := g.strPtrLen(a)
		bp, bl := g.strPtrLen(b)
		fn := map[string]string{"string.startsWith": "veles_string_starts_with", "string.endsWith": "veles_string_ends_with", "string.contains": "veles_string_contains"}[e.Op]
		v := g.newTmp()
		g.emit("%s = call i1 @%s(ptr %s, i64 %s, ptr %s, i64 %s)", v, fn, ap, al, bp, bl)
		return v
	case "string.substring":
		s := g.expr(e.Args[0])
		lo := g.expr(e.Args[1])
		hi := g.expr(e.Args[2])
		sp, sl := g.strPtrLen(s)
		out := g.alloca(strType)
		g.emit("store %s zeroinitializer, ptr %s", strType, out)
		ok := g.newTmp()
		g.emit("%s = call i1 @veles_string_substring(ptr %s, ptr %s, i64 %s, i64 %s, i64 %s)", ok, out, sp, sl, lo, hi)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, strType, out)
		return g.makeNullable(e.Type().(*types.Nullable), ok, val)
	case "string.toInt":
		s := g.expr(e.Args[0])
		sp, sl := g.strPtrLen(s)
		out := g.alloca("i64")
		g.emit("store i64 0, ptr %s", out)
		ok := g.newTmp()
		g.emit("%s = call i1 @veles_string_to_int(ptr %s, i64 %s, ptr %s)", ok, sp, sl, out)
		val := g.newTmp()
		g.emit("%s = load i64, ptr %s", val, out)
		return g.makeNullable(e.Type().(*types.Nullable), ok, val)
	}
	panic("codegen: unknown builtin " + e.Op)
}

// makeNullable builds a T? from a presence flag and a value.
func (g *gen) makeNullable(nt *types.Nullable, present, val string) string {
	if isPtrLike(nt.Elem) {
		v := g.newTmp()
		g.emit("%s = select i1 %s, ptr %s, ptr null", v, present, val)
		return v
	}
	llt := g.llType(nt)
	a := g.newTmp()
	g.emit("%s = insertvalue %s undef, i1 %s, 0", a, llt, present)
	b := g.newTmp()
	g.emit("%s = insertvalue %s %s, %s %s, 1", b, llt, a, g.llType(nt.Elem), val)
	return b
}
