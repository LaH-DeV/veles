package sema

import (
	"math"
	"math/big"
	"math/bits"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// The numeric built-ins a `const fun` may use (D113). Integers are exact
// (big.Int, checked against the type); the floats are the ones whose result
// is defined by IEEE 754 and so is the same in the compiler as in the
// program: `sqrt`, rounding, `abs`, `min`/`max`, `mod`, `clamp`, `sign`,
// `copySign` and the tests. `sin`, `exp`, `log`, `pow` and the others are the
// C library's, and two libraries differ in the last bit; the compiler does not
// guess which one the program will link, so they are not evaluated.
func init() {
	for _, op := range []string{
		"int.abs", "int.sign", "int.min", "int.max", "int.mod", "int.clamp", "int.pow",
		"int.wrappingAdd", "int.wrappingSub", "int.wrappingMul",
		"int.saturatingAdd", "int.saturatingSub", "int.saturatingMul",
		"int.checkedAdd", "int.checkedSub", "int.checkedMul",
		"int.countOnes", "int.leadingZeros", "int.trailingZeros", "int.swapBytes", "int.reverseBits",
		"int.rotateLeft", "int.rotateRight",
		"float.sqrt", "float.abs", "float.floor", "float.ceil", "float.round", "float.trunc",
		"float.sign", "float.min", "float.max", "float.mod", "float.clamp", "float.copySign",
		"float.isSignNegative", "float.isNaN", "float.isFinite", "float.isInfinite",
	} {
		constBuiltins[op] = true
	}
}

func (ev *constEval) intVal(e *Builtin, i int) *big.Int {
	v, ok := ev.expr(e.Args[i]).(*CInt)
	if !ok {
		ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	}
	return v.V
}

func (ev *constEval) floatVal(e *Builtin, i int) float64 {
	v, ok := ev.expr(e.Args[i]).(*CFloat)
	if !ok {
		ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	}
	return v.V
}

// builtinNum runs the numeric operations; the second result says the
// operation is not one of them.
func (ev *constEval) builtinNum(e *Builtin) (ConstVal, bool) {
	switch {
	case strings.HasPrefix(e.Op, "int."):
		return ev.intBuiltin(e)
	case strings.HasPrefix(e.Op, "float."):
		return ev.floatBuiltin(e)
	}
	return nil, false
}

func (ev *constEval) intBuiltin(e *Builtin) (ConstVal, bool) {
	// the operand's type: the result's, except for the checked family (a T?)
	t := e.Type()
	if nt, ok := t.(*types.Nullable); ok {
		t = nt.Elem
	}
	lo, hi := intBounds(t)
	width := types.BitSize(types.Underlying(t))
	signed := types.IsSigned(types.Underlying(t))
	mk := func(v *big.Int) ConstVal { return &CInt{t, v} }
	check := func(v *big.Int, what string) ConstVal {
		if !fitsInt(v, t) {
			ev.errorf(e.Span, "constant overflow: %s is %s, outside '%s' (%s to %s) (D113)", what, v, t, lo, hi)
		}
		return mk(v)
	}
	// the bit pattern of v as an unsigned number of the type's width
	pattern := func(v *big.Int) uint64 {
		return new(big.Int).Mod(v, new(big.Int).Lsh(big.NewInt(1), uint(width))).Uint64()
	}
	switch e.Op {
	case "int.abs":
		x := ev.intVal(e, 0)
		if !signed || x.Sign() >= 0 {
			return mk(x), true
		}
		return check(new(big.Int).Neg(x), "abs of "+x.String()), true
	case "int.sign":
		x := ev.intVal(e, 0)
		return mk(big.NewInt(int64(x.Sign()))), true
	case "int.min", "int.max":
		x, y := ev.intVal(e, 0), ev.intVal(e, 1)
		if (x.Cmp(y) < 0) == (e.Op == "int.min") {
			return mk(x), true
		}
		return mk(y), true
	case "int.clamp":
		x, l, h := ev.intVal(e, 0), ev.intVal(e, 1), ev.intVal(e, 2)
		if x.Cmp(l) < 0 {
			x = l
		}
		if x.Cmp(h) > 0 {
			x = h
		}
		return mk(x), true
	case "int.mod":
		x, y := ev.intVal(e, 0), ev.intVal(e, 1)
		if y.Sign() == 0 {
			ev.errorf(e.Span, "division by zero in a constant: %s mod 0", x)
		}
		if signed && y.Cmp(big.NewInt(-1)) == 0 && x.Cmp(lo) == 0 {
			ev.errorf(e.Span, "constant overflow: %s mod -1 (D113)", x)
		}
		r := new(big.Int).Rem(x, y)
		if r.Sign() < 0 {
			r.Add(r, new(big.Int).Abs(y))
		}
		return mk(r), true
	case "int.pow":
		x, n := ev.intVal(e, 0), ev.intVal(e, 1)
		if n.Sign() < 0 {
			ev.errorf(e.Span, "a negative exponent in a constant: %s.pow(%s) (D113)", x, n)
		}
		// beyond |x| <= 1 the result outgrows every type within 64 steps
		if x.CmpAbs(big.NewInt(1)) > 0 && n.Cmp(big.NewInt(64)) > 0 {
			ev.errorf(e.Span, "constant overflow: %s.pow(%s) does not fit '%s' (D113)", x, n, t)
		}
		z := new(big.Int).Exp(x, n, nil)
		return check(z, x.String()+".pow("+n.String()+")"), true
	case "int.wrappingAdd", "int.wrappingSub", "int.wrappingMul":
		x, y := ev.intVal(e, 0), ev.intVal(e, 1)
		z := new(big.Int)
		switch e.Op {
		case "int.wrappingAdd":
			z.Add(x, y)
		case "int.wrappingSub":
			z.Sub(x, y)
		default:
			z.Mul(x, y)
		}
		return mk(wrapInt(z, t)), true
	case "int.saturatingAdd", "int.saturatingSub", "int.saturatingMul":
		x, y := ev.intVal(e, 0), ev.intVal(e, 1)
		z := new(big.Int)
		switch e.Op {
		case "int.saturatingAdd":
			z.Add(x, y)
		case "int.saturatingSub":
			z.Sub(x, y)
		default:
			z.Mul(x, y)
		}
		if z.Cmp(lo) < 0 {
			z = lo
		} else if z.Cmp(hi) > 0 {
			z = hi
		}
		return mk(z), true
	case "int.checkedAdd", "int.checkedSub", "int.checkedMul":
		x, y := ev.intVal(e, 0), ev.intVal(e, 1)
		z := new(big.Int)
		switch e.Op {
		case "int.checkedAdd":
			z.Add(x, y)
		case "int.checkedSub":
			z.Sub(x, y)
		default:
			z.Mul(x, y)
		}
		nt := e.Type().(*types.Nullable)
		if !fitsInt(z, t) {
			return &CNull{nt}, true
		}
		return &CSome{nt, mk(z)}, true
	case "int.countOnes":
		return mk(big.NewInt(int64(bits.OnesCount64(pattern(ev.intVal(e, 0)))))), true
	case "int.leadingZeros":
		return mk(big.NewInt(int64(width - bits.Len64(pattern(ev.intVal(e, 0)))))), true
	case "int.trailingZeros":
		p := pattern(ev.intVal(e, 0))
		if p == 0 {
			return mk(big.NewInt(int64(width))), true
		}
		return mk(big.NewInt(int64(bits.TrailingZeros64(p)))), true
	case "int.swapBytes":
		p := pattern(ev.intVal(e, 0))
		return mk(wrapInt(new(big.Int).SetUint64(bits.ReverseBytes64(p)>>(64-uint(width))), t)), true
	case "int.reverseBits":
		p := pattern(ev.intVal(e, 0))
		return mk(wrapInt(new(big.Int).SetUint64(bits.Reverse64(p)>>(64-uint(width))), t)), true
	case "int.rotateLeft", "int.rotateRight":
		p := pattern(ev.intVal(e, 0))
		n := int(new(big.Int).Mod(ev.intVal(e, 1), big.NewInt(int64(width))).Int64())
		if e.Op == "int.rotateRight" {
			n = (width - n) % width
		}
		mask := uint64(math.MaxUint64)
		if width < 64 {
			mask = 1<<uint(width) - 1
		}
		r := (p<<uint(n) | p>>uint(width-n)) & mask
		if n == 0 {
			r = p
		}
		return mk(wrapInt(new(big.Int).SetUint64(r), t)), true
	}
	return nil, false
}

func (ev *constEval) floatBuiltin(e *Builtin) (ConstVal, bool) {
	t := e.Type()
	f := func(v float64) ConstVal { return mkFloat(t, v) }
	// minnum/maxnum: a NaN operand gives the other
	maxnum := func(a, b float64) float64 {
		switch {
		case math.IsNaN(a):
			return b
		case math.IsNaN(b):
			return a
		}
		return math.Max(a, b)
	}
	minnum := func(a, b float64) float64 {
		switch {
		case math.IsNaN(a):
			return b
		case math.IsNaN(b):
			return a
		}
		return math.Min(a, b)
	}
	switch e.Op {
	case "float.sqrt":
		return f(math.Sqrt(ev.floatVal(e, 0))), true
	case "float.abs":
		return f(math.Abs(ev.floatVal(e, 0))), true
	case "float.floor":
		return f(math.Floor(ev.floatVal(e, 0))), true
	case "float.ceil":
		return f(math.Ceil(ev.floatVal(e, 0))), true
	case "float.round":
		return f(math.Round(ev.floatVal(e, 0))), true // half away from zero, as llvm.round
	case "float.trunc":
		return f(math.Trunc(ev.floatVal(e, 0))), true
	case "float.sign":
		x := ev.floatVal(e, 0)
		switch {
		case x < 0:
			return f(-1), true
		case x > 0:
			return f(1), true
		}
		return f(x), true // NaN and zeros stay themselves
	case "float.min":
		return f(minnum(ev.floatVal(e, 0), ev.floatVal(e, 1))), true
	case "float.max":
		return f(maxnum(ev.floatVal(e, 0), ev.floatVal(e, 1))), true
	case "float.mod":
		x, y := ev.floatVal(e, 0), ev.floatVal(e, 1)
		r := math.Mod(x, y)
		if r < 0 {
			r += math.Abs(y)
		}
		return f(r), true
	case "float.clamp":
		return f(minnum(maxnum(ev.floatVal(e, 0), ev.floatVal(e, 1)), ev.floatVal(e, 2))), true
	case "float.copySign":
		return f(math.Copysign(ev.floatVal(e, 0), ev.floatVal(e, 1))), true
	case "float.isSignNegative":
		return &CBool{math.Signbit(ev.floatVal(e, 0))}, true
	case "float.isNaN":
		return &CBool{math.IsNaN(ev.floatVal(e, 0))}, true
	case "float.isFinite":
		x := ev.floatVal(e, 0)
		return &CBool{!math.IsNaN(x) && !math.IsInf(x, 0)}, true
	case "float.isInfinite":
		return &CBool{math.IsInf(ev.floatVal(e, 0), 0)}, true
	}
	return nil, false
}
