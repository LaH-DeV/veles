package sema

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D113: compile-time evaluation. A `const`'s initializer is checked as any
// expression is, then evaluated here, over the checked HIR, to a ConstVal.
// Every use of the constant is then that value written in as a literal
// (constExpr); a table (List, Map, Set) is laid out once, read-only, in the
// binary (ConstTable).
//
// The evaluator gives what the program would compute at run time, except
// that what would be a run-time failure — an overflow in any profile, a
// division by zero, a shift past the width, an index out of range, a panic
// — is a compile error at the constant, since a constant is something
// someone wrote. What it cannot evaluate (a function call, a `val`, I/O)
// is an error naming it: the constant is then not a constant.

// ConstVal is a value computed at compile time.
type ConstVal interface{ Type() types.Type }

type (
	CInt struct { // every integer type and enums (D57: a member is its integer)
		T types.Type
		V *big.Int
	}
	CFloat struct { // f32 values are kept rounded to f32
		T types.Type
		V float64
	}
	CBool   struct{ V bool }
	CString struct{ V string }
	CUnit   struct{}
	CNull   struct{ T *types.Nullable }
	CSome   struct { // a value of a nullable type that is not null
		T *types.Nullable
		V ConstVal
	}
	CTuple struct {
		T     *types.Tuple
		Elems []ConstVal
	}
	CStruct struct {
		T      *types.Struct
		Fields []ConstVal
	}
	CList struct {
		T     *types.List
		Elems []ConstVal
	}
	CArray struct { // an `Array<T, N>` (D121): inline, a value
		T     *types.Array
		Elems []ConstVal
	}
	CMap struct { // insertion order (D25)
		T    *types.Map
		Keys []ConstVal
		Vals []ConstVal
	}
	CSet struct {
		T     *types.Set
		Elems []ConstVal
	}
)

func (v *CInt) Type() types.Type    { return v.T }
func (v *CFloat) Type() types.Type  { return v.T }
func (v *CBool) Type() types.Type   { return types.TBool }
func (v *CString) Type() types.Type { return types.TString }
func (v *CUnit) Type() types.Type   { return types.TUnit }
func (v *CNull) Type() types.Type   { return v.T }
func (v *CSome) Type() types.Type   { return v.T }
func (v *CTuple) Type() types.Type  { return v.T }
func (v *CStruct) Type() types.Type { return v.T }
func (v *CList) Type() types.Type   { return v.T }
func (v *CArray) Type() types.Type  { return v.T }
func (v *CMap) Type() types.Type    { return v.T }
func (v *CSet) Type() types.Type    { return v.T }

// constFailed unwinds the evaluator after its error was reported.
type constFailed struct{}

type constEval struct {
	c     *Checker
	env   map[*Var]ConstVal
	at    source.Span // the initializer: where a node without a span of its own reports
	quiet bool        // tryConst: not being a constant is no error
}

// evalConst evaluates a checked constant expression; nil after an error,
// which it has reported.
func (c *Checker) evalConst(e Expr, at source.Span) ConstVal {
	return c.runConst(&constEval{c: c, env: map[*Var]ConstVal{}, at: at}, e)
}

// tryConst is e's value when e is a constant expression, else nil; it
// reports nothing (an index into a constant table, D113).
func (c *Checker) tryConst(e Expr) ConstVal {
	return c.runConst(&constEval{c: c, env: map[*Var]ConstVal{}, quiet: true}, e)
}

func (c *Checker) runConst(ev *constEval, e Expr) (v ConstVal) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(constFailed); !ok {
				panic(r)
			}
			v = nil
		}
	}()
	return ev.expr(e)
}

func (ev *constEval) errorf(span source.Span, format string, args ...any) {
	if ev.quiet {
		panic(constFailed{})
	}
	if span.File == nil {
		span = ev.at
	}
	ev.c.errorf(span, format, args...)
	panic(constFailed{})
}

func (ev *constEval) expr(e Expr) ConstVal {
	switch e := e.(type) {
	case *IntConst:
		v := new(big.Int).SetUint64(e.Value)
		if e.Neg {
			v.Neg(v)
		}
		if types.IsFloat(e.Type()) { // a literal typed by its context
			f, _ := new(big.Float).SetInt(v).Float64()
			return mkFloat(e.Type(), f)
		}
		return &CInt{e.Type(), v}
	case *FloatConst:
		return mkFloat(e.Type(), e.Value)
	case *BoolConst:
		return &CBool{e.Value}
	case *StringConst:
		return &CString{e.Value}
	case *ConstTable:
		return e.Value
	case *UnitConst:
		return &CUnit{}
	case *NullConst:
		return &CNull{e.Type().(*types.Nullable)}
	case *VarRef:
		return ev.varRef(e)
	case *Binary:
		return ev.binary(e)
	case *Unary:
		return ev.unary(e)
	case *Cast:
		return ev.cast(e)
	case *ToString:
		return &CString{ev.text(ev.expr(e.X), e.X.Type())}
	case *StringConcat:
		var b strings.Builder
		for _, p := range e.Parts {
			b.WriteString(ev.text(ev.expr(p), p.Type()))
		}
		return &CString{b.String()}
	case *TupleLit:
		t := e.Type().(*types.Tuple)
		out := &CTuple{T: t}
		for _, x := range e.Elems {
			out.Elems = append(out.Elems, ev.expr(x))
		}
		return out
	case *StructLit:
		out := &CStruct{T: e.Struct}
		for _, x := range e.Fields {
			out.Fields = append(out.Fields, ev.expr(x))
		}
		return out
	case *FieldGet:
		if s, ok := ev.expr(e.X).(*CStruct); ok && e.Index < len(s.Fields) {
			return s.Fields[e.Index]
		}
	case *TupleGet:
		if t, ok := ev.expr(e.X).(*CTuple); ok && e.Index < len(t.Elems) {
			return t.Elems[e.Index]
		}
	case *ListLit:
		if at, isArray := e.Type().(*types.Array); isArray {
			out := &CArray{T: at}
			for _, x := range e.Elems {
				out.Elems = append(out.Elems, ev.expr(x))
			}
			return out
		}
		out := &CList{T: e.Type().(*types.List)}
		for _, x := range e.Elems {
			out.Elems = append(out.Elems, ev.expr(x))
		}
		return out
	case *MapLit:
		out := &CMap{T: e.Type().(*types.Map)}
		for _, en := range e.Entries {
			mapPut(out, ev.expr(en[0]), ev.expr(en[1]))
		}
		return out
	case *SomeWrap:
		return &CSome{e.Type().(*types.Nullable), ev.expr(e.X)}
	case *IsNull:
		_, null := ev.expr(e.X).(*CNull)
		return &CBool{null}
	case *Unwrap:
		switch v := ev.expr(e.X).(type) {
		case *CSome:
			return v.V
		}
	case *Elvis:
		switch v := ev.expr(e.L).(type) {
		case *CSome:
			return v.V
		case *CNull:
			return ev.expr(e.R)
		}
	case *If:
		if ev.truth(e.Cond) {
			return ev.block(e.Then)
		}
		if e.Else == nil {
			return &CUnit{}
		}
		return ev.block(e.Else)
	case *BlockExpr:
		return ev.block(e.Block)
	case *Let:
		ev.env[e.Var] = ev.expr(e.Init)
		return ev.expr(e.Body)
	case *Match:
		return ev.match(e)
	case *Builtin:
		return ev.builtin(e)
	case *Call:
		ev.errorf(e.Span, "a constant cannot call '%s': only literals, other constants and operators on them are evaluated at compile time (D113); compute it at run time with 'val'", e.Fn.Display)
	}
	ev.errorf(spanOf(e), "%s is not a constant expression (D113); compute it at run time with 'val'", describeNode(e))
	return nil
}

// spanOf is the span a node carries, if any.
func spanOf(e Expr) source.Span {
	switch e := e.(type) {
	case *Binary:
		return e.Span
	case *Unary:
		return e.Span
	case *Builtin:
		return e.Span
	case *Call:
		return e.Span
	case *Match:
		return e.Span
	}
	return source.Span{}
}

// describeNode names what the reader wrote, for the refusal.
func describeNode(e Expr) string {
	switch e := e.(type) {
	case *Closure:
		return "a lambda"
	case *CallIndirect, *CallVirtual:
		return "a call"
	case *AddrOf:
		return "taking an address"
	case *MakeVariant:
		return "a sealed variant"
	case *Builtin:
		return "'" + e.Op[strings.LastIndexByte(e.Op, '.')+1:] + "'"
	case *Launch, *AwaitTask, *ScopeBlock, *Race:
		return "a task"
	case *Try, *Throw, *MakeResult:
		return "an error"
	}
	return "this"
}

func (ev *constEval) varRef(e *VarRef) ConstVal {
	v := e.Var
	if val, ok := ev.env[v]; ok {
		return val
	}
	if v.IsGlobal && v.Global != nil {
		g := v.Global
		d := ev.c.globals[g]
		if d == nil || d.Kind != ast.BindConst {
			ev.errorf(source.Span{}, "'%s' is a '%s', computed at run time, not a constant; declare it 'const' if its initializer is one (D113)", g.Display, d.Kind)
		}
		if g.Const != nil {
			return g.Const
		}
		if ev.c.globalState[g] == globalChecking {
			ev.errorf(source.Span{}, "the constant '%s' is defined in terms of itself", g.Display)
		}
		panic(constFailed{}) // its own error is reported at it
	}
	ev.errorf(source.Span{}, "'%s' is not a constant (D113)", v.Name)
	return nil
}

func (ev *constEval) truth(e Expr) bool {
	b, ok := ev.expr(e).(*CBool)
	if !ok {
		ev.errorf(spanOf(e), "%s is not a constant expression (D113); compute it at run time with 'val'", describeNode(e))
	}
	return b.V
}

func (ev *constEval) block(b *Block) ConstVal {
	for _, s := range b.Stmts {
		ev.stmt(s)
	}
	if b.Value == nil {
		return &CUnit{}
	}
	return ev.expr(b.Value)
}

func (ev *constEval) stmt(s Stmt) {
	switch s := s.(type) {
	case *VarDecl:
		if s.Init == nil {
			ev.errorf(source.Span{}, "this statement is not a constant expression (D113)")
		}
		ev.env[s.Var] = ev.expr(s.Init)
	case *ExprStmt:
		ev.expr(s.X)
	case *Assign:
		r, ok := s.Target.(*VarRef)
		if !ok || ev.env[r.Var] == nil {
			ev.errorf(source.Span{}, "this statement is not a constant expression (D113)")
		}
		ev.env[r.Var] = ev.expr(s.Value)
	case *Block:
		ev.block(s)
	default:
		ev.errorf(source.Span{}, "this statement is not a constant expression (D113)")
	}
}

func (ev *constEval) match(m *Match) ConstVal {
	if m.Subject != nil {
		ev.env[m.Subject] = ev.expr(m.Init)
	}
	for _, arm := range m.Arms {
		if arm.Test != nil && !ev.truth(arm.Test) {
			continue
		}
		for _, b := range arm.Binds {
			ev.stmt(b)
		}
		if arm.Guard != nil && !ev.truth(arm.Guard) {
			continue
		}
		return ev.block(arm.Body)
	}
	if m.Exhaustive {
		ev.errorf(m.Span, "no arm of the 'when' matches the constant")
	}
	return &CUnit{}
}

// ---------------------------------------------------------------------------
// numbers

func mkFloat(t types.Type, f float64) *CFloat {
	if types.BitSize(t) == 32 {
		f = float64(float32(f))
	}
	return &CFloat{t, f}
}

// intBounds is the range of an integer (or enum) type.
func intBounds(t types.Type) (lo, hi *big.Int) {
	t = types.Underlying(t)
	n := uint(types.BitSize(t))
	one := big.NewInt(1)
	if types.IsSigned(t) {
		hi = new(big.Int).Sub(new(big.Int).Lsh(one, n-1), one)
		lo = new(big.Int).Neg(new(big.Int).Lsh(one, n-1))
		return lo, hi
	}
	return big.NewInt(0), new(big.Int).Sub(new(big.Int).Lsh(one, n), one)
}

func fitsInt(v *big.Int, t types.Type) bool {
	lo, hi := intBounds(t)
	return v.Cmp(lo) >= 0 && v.Cmp(hi) <= 0
}

// wrapInt keeps the low bits of v as a value of t (two's complement).
func wrapInt(v *big.Int, t types.Type) *big.Int {
	n := uint(types.BitSize(types.Underlying(t)))
	mod := new(big.Int).Lsh(big.NewInt(1), n)
	r := new(big.Int).Mod(v, mod) // 0 <= r < 2^n
	if types.IsSigned(types.Underlying(t)) && r.Bit(int(n-1)) == 1 {
		r.Sub(r, mod)
	}
	return r
}

var binOpText = map[BinOp]string{
	OpAdd: "+", OpSub: "-", OpMul: "*", OpDiv: "/", OpRem: "%",
	OpWrapAdd: "+%", OpWrapSub: "-%", OpWrapMul: "*%",
	OpShl: "<<", OpShr: ">>",
}

func (ev *constEval) binary(e *Binary) ConstVal {
	switch e.Op {
	case OpAnd:
		return &CBool{ev.truth(e.L) && ev.truth(e.R)}
	case OpOr:
		return &CBool{ev.truth(e.L) || ev.truth(e.R)}
	}
	l, r := ev.expr(e.L), ev.expr(e.R)
	if e.Op == OpEq || e.Op == OpNe {
		if why := ev.c.notConstType(e.L.Type(), true, map[types.Type]bool{}); why != "" {
			ev.errorf(e.Span, "'==' here is not a constant expression: %s (D113)", why)
		}
	}
	switch e.Op {
	case OpEq:
		return &CBool{constEqual(l, r)}
	case OpNe:
		return &CBool{!constEqual(l, r)}
	case OpLt, OpLe, OpGt, OpGe:
		if a, ok := l.(*CFloat); ok {
			b := r.(*CFloat)
			switch e.Op { // NaN: every ordering is false, as at run time
			case OpLt:
				return &CBool{a.V < b.V}
			case OpLe:
				return &CBool{a.V <= b.V}
			case OpGt:
				return &CBool{a.V > b.V}
			}
			return &CBool{a.V >= b.V}
		}
		c, ok := constCompare(l, r)
		if !ok {
			break
		}
		switch e.Op {
		case OpLt:
			return &CBool{c < 0}
		case OpLe:
			return &CBool{c <= 0}
		case OpGt:
			return &CBool{c > 0}
		}
		return &CBool{c >= 0}
	}
	switch a := l.(type) {
	case *CInt:
		b, ok := r.(*CInt)
		if ok {
			return ev.intOp(e, a, b)
		}
	case *CFloat:
		if b, ok := r.(*CFloat); ok {
			var f float64
			switch e.Op {
			case OpAdd:
				f = a.V + b.V
			case OpSub:
				f = a.V - b.V
			case OpMul:
				f = a.V * b.V
			case OpDiv:
				f = a.V / b.V
			case OpRem:
				f = math.Mod(a.V, b.V)
			default:
				ev.errorf(e.Span, "this operator is not a constant expression (D113)")
			}
			return mkFloat(e.Type(), f)
		}
	case *CBool:
		if b, ok := r.(*CBool); ok {
			switch e.Op {
			case OpBitAnd:
				return &CBool{a.V && b.V}
			case OpBitOr:
				return &CBool{a.V || b.V}
			case OpBitXor:
				return &CBool{a.V != b.V}
			}
		}
	case *CString:
		if b, ok := r.(*CString); ok && e.Op == OpAdd {
			return &CString{a.V + b.V}
		}
	}
	ev.errorf(e.Span, "this operator is not a constant expression (D113)")
	return nil
}

func (ev *constEval) intOp(e *Binary, a, b *CInt) ConstVal {
	t := e.Type()
	z := new(big.Int)
	checked := true
	switch e.Op {
	case OpAdd, OpWrapAdd:
		z.Add(a.V, b.V)
		checked = e.Op == OpAdd
	case OpSub, OpWrapSub:
		z.Sub(a.V, b.V)
		checked = e.Op == OpSub
	case OpMul, OpWrapMul:
		z.Mul(a.V, b.V)
		checked = e.Op == OpMul
	case OpDiv, OpRem:
		if b.V.Sign() == 0 {
			ev.errorf(e.Span, "division by zero in a constant: %s %s 0", a.V, binOpText[e.Op])
		}
		if e.Op == OpDiv {
			z.Quo(a.V, b.V) // truncated, as the machine divides
		} else {
			z.Rem(a.V, b.V)
		}
	case OpBitAnd:
		return &CInt{t, wrapInt(z.And(a.V, b.V), t)}
	case OpBitOr:
		return &CInt{t, wrapInt(z.Or(a.V, b.V), t)}
	case OpBitXor:
		return &CInt{t, wrapInt(z.Xor(a.V, b.V), t)}
	case OpShl, OpShr:
		width := types.BitSize(types.Underlying(t))
		if b.V.Sign() < 0 || b.V.Cmp(big.NewInt(int64(width))) >= 0 {
			ev.errorf(e.Span, "a shift by %s in a constant is outside 0..%d, the width of '%s'", b.V, width-1, t)
		}
		n := uint(b.V.Uint64())
		if e.Op == OpShl {
			return &CInt{t, wrapInt(z.Lsh(a.V, n), t)}
		}
		return &CInt{t, z.Rsh(a.V, n)} // arithmetic for a negative value, as the signed shift is
	default:
		ev.errorf(e.Span, "this operator is not a constant expression (D113)")
	}
	if !checked {
		return &CInt{t, wrapInt(z, t)}
	}
	if !fitsInt(z, t) {
		lo, hi := intBounds(t)
		ev.errorf(e.Span, "constant overflow: %s %s %s is %s, outside '%s' (%s to %s); a constant is checked in every profile — use '%s%%' to wrap on purpose", a.V, binOpText[e.Op], b.V, z, t, lo, hi, binOpText[e.Op])
	}
	return &CInt{t, z}
}

func (ev *constEval) unary(e *Unary) ConstVal {
	x := ev.expr(e.X)
	switch e.Op {
	case OpNot:
		if b, ok := x.(*CBool); ok {
			return &CBool{!b.V}
		}
	case OpNeg:
		switch v := x.(type) {
		case *CInt:
			z := new(big.Int).Neg(v.V)
			if !fitsInt(z, e.Type()) {
				ev.errorf(e.Span, "constant overflow: -(%s) does not fit '%s'", v.V, e.Type())
			}
			return &CInt{e.Type(), z}
		case *CFloat:
			return mkFloat(e.Type(), -v.V)
		}
	case OpBitNot:
		if v, ok := x.(*CInt); ok {
			return &CInt{e.Type(), wrapInt(new(big.Int).Not(v.V), e.Type())}
		}
	}
	ev.errorf(e.Span, "this operator is not a constant expression (D113)")
	return nil
}

// cast is a conversion the checker already proved total (D86) or a
// retyping (an enum and its base, a collection made read-only).
func (ev *constEval) cast(e *Cast) ConstVal {
	x := ev.expr(e.X)
	to := e.Type()
	switch v := x.(type) {
	case *CInt:
		switch {
		case types.IsInteger(types.Underlying(to)):
			return &CInt{to, wrapInt(v.V, to)}
		case types.IsFloat(to):
			bf := new(big.Float).SetInt(v.V)
			if types.BitSize(to) == 32 {
				f, _ := bf.Float32()
				return &CFloat{to, float64(f)}
			}
			f, _ := bf.Float64()
			return &CFloat{to, f}
		}
	case *CFloat:
		if types.IsFloat(to) {
			return mkFloat(to, v.V)
		}
	case *CList:
		if lt, ok := to.(*types.List); ok {
			return &CList{lt, v.Elems}
		}
	case *CArray:
		if at, ok := to.(*types.Array); ok {
			return &CArray{at, v.Elems}
		}
	case *CMap:
		if mt, ok := to.(*types.Map); ok {
			return &CMap{mt, v.Keys, v.Vals}
		}
	case *CSet:
		if st, ok := to.(*types.Set); ok {
			return &CSet{st, v.Elems}
		}
	default:
		if types.Identical(x.Type(), to) {
			return x
		}
	}
	ev.errorf(source.Span{}, "this conversion is not a constant expression (D113)")
	return nil
}

func (ev *constEval) builtin(e *Builtin) ConstVal {
	switch e.Op {
	case "string.len":
		s := ev.expr(e.Args[0]).(*CString)
		return &CInt{types.TI64, big.NewInt(int64(len(s.V)))}
	case "list.len":
		if a, ok := ev.expr(e.Args[0]).(*CArray); ok {
			return &CInt{types.TI64, big.NewInt(int64(len(a.Elems)))}
		}
		l := ev.expr(e.Args[0]).(*CList)
		return &CInt{types.TI64, big.NewInt(int64(len(l.Elems)))}
	case "map.len":
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			return &CInt{types.TI64, big.NewInt(int64(len(m.Keys)))}
		case *CSet:
			return &CInt{types.TI64, big.NewInt(int64(len(m.Elems)))}
		}
	case "list.get":
		var elems []ConstVal
		switch l := ev.expr(e.Args[0]).(type) {
		case *CArray:
			elems = l.Elems
		case *CList:
			elems = l.Elems
		}
		i := ev.expr(e.Args[1]).(*CInt)
		if i.V.Sign() < 0 || i.V.Cmp(big.NewInt(int64(len(elems)))) >= 0 {
			ev.errorf(e.Span, "index %s is out of range for a constant list of length %d (D113)", i.V, len(elems))
		}
		return elems[i.V.Int64()]
	case "map.get":
		m := ev.expr(e.Args[0]).(*CMap)
		k := ev.expr(e.Args[1])
		nt := e.Type().(*types.Nullable)
		for i, key := range m.Keys {
			if constEqual(key, k) {
				return &CSome{nt, m.Vals[i]}
			}
		}
		return &CNull{nt}
	case "map.contains":
		k := ev.expr(e.Args[1])
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			return &CBool{constIndex(m.Keys, k) >= 0}
		case *CSet:
			return &CBool{constIndex(m.Elems, k) >= 0}
		}
	case "set.new":
		return &CSet{T: e.Type().(*types.Set)}
	case "set.add":
		s := ev.expr(e.Args[0]).(*CSet)
		k := ev.expr(e.Args[1])
		if constIndex(s.Elems, k) >= 0 {
			return &CBool{false}
		}
		s.Elems = append(s.Elems, k)
		return &CBool{true}
	case "num.toChecked":
		nt := e.Type().(*types.Nullable)
		to := nt.Elem
		switch v := ev.expr(e.Args[0]).(type) {
		case *CInt:
			if fitsInt(v.V, to) {
				return &CSome{nt, &CInt{to, v.V}}
			}
			return &CNull{nt}
		case *CFloat:
			if math.IsNaN(v.V) || math.IsInf(v.V, 0) {
				return &CNull{nt}
			}
			z, _ := big.NewFloat(math.Trunc(v.V)).Int(nil)
			if fitsInt(z, to) {
				return &CSome{nt, &CInt{to, z}}
			}
			return &CNull{nt}
		}
	}
	ev.errorf(e.Span, "%s is not a constant expression (D113); compute it at run time with 'val'", describeNode(e))
	return nil
}

func mapPut(m *CMap, k, v ConstVal) {
	if i := constIndex(m.Keys, k); i >= 0 {
		m.Vals[i] = v // a repeated key keeps its place and takes the later value
		return
	}
	m.Keys = append(m.Keys, k)
	m.Vals = append(m.Vals, v)
}

func constIndex(xs []ConstVal, k ConstVal) int {
	for i, x := range xs {
		if constEqual(x, k) {
			return i
		}
	}
	return -1
}

// constEqual is `==` on constants: structural, NaN unequal to itself.
func constEqual(a, b ConstVal) bool {
	switch a := a.(type) {
	case *CInt:
		b, ok := b.(*CInt)
		return ok && a.V.Cmp(b.V) == 0
	case *CFloat:
		b, ok := b.(*CFloat)
		return ok && a.V == b.V
	case *CBool:
		b, ok := b.(*CBool)
		return ok && a.V == b.V
	case *CString:
		b, ok := b.(*CString)
		return ok && a.V == b.V
	case *CUnit:
		_, ok := b.(*CUnit)
		return ok
	case *CNull:
		_, ok := b.(*CNull)
		return ok
	case *CSome:
		b, ok := b.(*CSome)
		return ok && constEqual(a.V, b.V)
	case *CTuple:
		b, ok := b.(*CTuple)
		return ok && constAllEqual(a.Elems, b.Elems)
	case *CStruct:
		b, ok := b.(*CStruct)
		return ok && constAllEqual(a.Fields, b.Fields)
	case *CList:
		b, ok := b.(*CList)
		return ok && constAllEqual(a.Elems, b.Elems)
	case *CArray:
		b, ok := b.(*CArray)
		return ok && constAllEqual(a.Elems, b.Elems)
	case *CSet:
		b, ok := b.(*CSet)
		if !ok || len(a.Elems) != len(b.Elems) {
			return false
		}
		for _, x := range a.Elems {
			if constIndex(b.Elems, x) < 0 {
				return false
			}
		}
		return true
	case *CMap:
		b, ok := b.(*CMap)
		if !ok || len(a.Keys) != len(b.Keys) {
			return false
		}
		for i, k := range a.Keys {
			j := constIndex(b.Keys, k)
			if j < 0 || !constEqual(a.Vals[i], b.Vals[j]) {
				return false
			}
		}
		return true
	}
	return false
}

func constAllEqual(a, b []ConstVal) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !constEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// constCompare orders integers and strings (bytes, as at run time).
func constCompare(a, b ConstVal) (int, bool) {
	switch a := a.(type) {
	case *CInt:
		if b, ok := b.(*CInt); ok {
			return a.V.Cmp(b.V), true
		}
	case *CString:
		if b, ok := b.(*CString); ok {
			return strings.Compare(a.V, b.V), true
		}
	}
	return 0, false
}

// text is a constant as string interpolation prints it.
func (ev *constEval) text(v ConstVal, t types.Type) string {
	switch v := v.(type) {
	case *CString:
		return v.V
	case *CBool:
		return strconv.FormatBool(v.V)
	case *CInt:
		if !types.IsEnum(t) {
			return v.V.String()
		}
	case *CFloat:
		return formatFloat(v.V, types.BitSize(v.T) == 32)
	}
	ev.errorf(source.Span{}, "interpolating a '%s' is not a constant expression (D113); interpolate numbers, 'bool's and strings, or compute it at run time with 'val'", t)
	return ""
}

// formatFloat prints a float as the runtime's float_to_string does: the
// shortest text that reads back as the value, plain digits for exponents
// in -4..14, an exponent without padding otherwise, and always a point.
func formatFloat(v float64, f32 bool) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	bits := 64
	if f32 {
		bits = 32
	}
	s := strconv.FormatFloat(v, 'g', -1, bits)
	if i := strings.IndexByte(s, 'e'); i >= 0 {
		if exp, _ := strconv.Atoi(s[i+1:]); exp >= -4 && exp < 15 {
			s = strconv.FormatFloat(v, 'f', -1, bits)
		} else if len(s) > i+3 && s[i+2] == '0' {
			s = s[:i+2] + s[i+3:] // 1e-07 → 1e-7
		}
	}
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

// constExpr writes a constant in as HIR: a literal for a scalar, a struct
// or tuple of literals, and the one read-only table for a collection.
func constExpr(v ConstVal) Expr {
	switch v := v.(type) {
	case *CInt:
		mag := new(big.Int).Abs(v.V)
		return &IntConst{exprBase{v.T}, mag.Uint64(), v.V.Sign() < 0}
	case *CFloat:
		return &FloatConst{exprBase{v.T}, v.V}
	case *CBool:
		return &BoolConst{exprBase{types.TBool}, v.V}
	case *CString:
		return &StringConst{exprBase{types.TString}, v.V}
	case *CUnit:
		return &UnitConst{exprBase{types.TUnit}}
	case *CNull:
		return &NullConst{exprBase{v.T}}
	case *CSome:
		return &SomeWrap{exprBase{v.T}, constExpr(v.V)}
	case *CTuple:
		out := &TupleLit{exprBase: exprBase{v.T}}
		for _, x := range v.Elems {
			out.Elems = append(out.Elems, constExpr(x))
		}
		return out
	case *CStruct:
		out := &StructLit{exprBase{v.T}, v.T, nil}
		for _, x := range v.Fields {
			out.Fields = append(out.Fields, constExpr(x))
		}
		return out
	}
	return &ConstTable{exprBase{v.Type()}, v, ""}
}

// constValue evaluates a `const`'s initializer once its type is known to be
// one a constant can have; nil after an error.
func (c *Checker) constValue(init Expr, t types.Type, span source.Span) ConstVal {
	if why := c.notConstType(t, false, map[types.Type]bool{}); why != "" {
		c.errorf(span, "a constant cannot have type '%s': %s (D113); use 'val' for a value computed at run time", t, why)
		return nil
	}
	return c.evalConst(init, span)
}

// notConstType says why a value of t cannot be a constant, or "". A key
// (of a Map, an element of a Set) must also hash and compare the way the
// compiler does, so a type with its own `hash` or `equals` is refused there.
func (c *Checker) notConstType(t types.Type, key bool, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	switch t := t.(type) {
	case *types.Basic:
		if types.IsNever(t) || types.IsInvalid(t) {
			return "it has no values"
		}
		return ""
	case *types.Enum:
		return ""
	case *types.Nullable:
		return c.notConstType(t.Elem, key, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if why := c.notConstType(e, key, seen); why != "" {
				return why
			}
		}
		return ""
	case *types.Struct:
		if t.Sealed != nil {
			return "a sealed variant is not a constant value yet"
		}
		if t.Union {
			return "an extern union is C's data, not a constant"
		}
		if key {
			if ops := c.customOps(t); ops != nil && (ops.Hash != nil || ops.Equals != nil) {
				return "'" + t.Name + "' hashes or compares with its own 'hash'/'equals', which the compiler does not run"
			}
		}
		for _, f := range t.Fields {
			if why := c.notConstType(f.Type, key, seen); why != "" {
				return why
			}
		}
		return ""
	case *types.Array:
		return c.notConstType(t.Elem, key, seen)
	case *types.List:
		if t.Mutable {
			return "a 'MutableList' can change and a constant cannot; use 'List'"
		}
		return c.notConstType(t.Elem, key, seen)
	case *types.Set:
		if t.Mutable {
			return "a 'MutableSet' can change and a constant cannot; use 'Set'"
		}
		return c.notConstType(t.Elem, true, seen)
	case *types.Map:
		if t.Mutable {
			return "a 'MutableMap' can change and a constant cannot; use 'Map'"
		}
		if why := c.notConstType(t.Key, true, seen); why != "" {
			return why
		}
		return c.notConstType(t.Value, key, seen)
	case *types.Pointer:
		return "a pointer is an address, known only at run time"
	case *types.Func:
		return "a function value is not a constant"
	case *types.Trait:
		return "a trait object is not a constant"
	}
	return "its values are made at run time"
}
