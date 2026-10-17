package sema

import (
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// D171: "${x:spec}". The spec is
//
//	[[fill]align][sign][#][0][width][,][.precision][type]
//
// with align `<` `>` `^`, sign `+` or a space, and type `d x X b o f %` (or
// `s`). It is read and checked here, against the type of `x`, and lowered to
// a call of one of the prelude helpers in std/prelude/format.vs with the
// parts as constants — no format string exists at run time.

type fmtSpec struct {
	fill      string
	align     rune // 0, '<', '>', '^'
	sign      rune // 0, '+', ' '
	alt       bool
	zero      bool
	width     int
	group     bool
	precision int // -1: none
	typ       rune
}

func isAlign(r rune) bool { return r == '<' || r == '>' || r == '^' }

// parseFmtSpec reads a spec; the message is empty when it is well formed.
func parseFmtSpec(s string) (sp fmtSpec, msg string) {
	sp = fmtSpec{fill: " ", precision: -1}
	rs := []rune(s)
	i := 0
	if len(rs) >= 2 && isAlign(rs[1]) {
		sp.fill, sp.align, i = string(rs[0]), rs[1], 2
	} else if len(rs) >= 1 && isAlign(rs[0]) {
		sp.align, i = rs[0], 1
	}
	if i < len(rs) && (rs[i] == '+' || rs[i] == ' ') {
		sp.sign = rs[i]
		i++
	}
	if i < len(rs) && rs[i] == '#' {
		sp.alt = true
		i++
	}
	if i < len(rs) && rs[i] == '0' {
		sp.zero = true
		i++
	}
	digits := func() (int, bool) {
		start := i
		for i < len(rs) && rs[i] >= '0' && rs[i] <= '9' {
			i++
		}
		if start == i {
			return 0, false
		}
		n, err := strconv.Atoi(string(rs[start:i]))
		if err != nil || n > 10000 {
			return 0, false
		}
		return n, true
	}
	if n, ok := digits(); ok {
		sp.width = n
	}
	if i < len(rs) && rs[i] == ',' {
		sp.group = true
		i++
	}
	if i < len(rs) && rs[i] == '.' {
		i++
		n, ok := digits()
		if !ok {
			return sp, "'.' needs the number of digits after it"
		}
		sp.precision = n
	}
	if i < len(rs) {
		switch rs[i] {
		case 'd', 'x', 'X', 'b', 'o', 'f', '%', 's':
			sp.typ = rs[i]
			i++
		case 'e':
			return sp, "the type 'e' is not available yet"
		}
	}
	if i < len(rs) {
		return sp, "unexpected '" + string(rs[i]) + "'"
	}
	return sp, ""
}

// formatted lowers one "${x:spec}" part; x is the checked expression.
func (f *fnCtx) formatted(p ast.StringPart, x Expr) Expr {
	at := p.SpecPos
	if !at.IsValid() {
		at = p.Expr.Span()
	}
	t := x.Type()
	if types.IsInvalid(t) {
		return x
	}
	if p.Spec == "" {
		return f.toString(x, p.Expr.Span())
	}
	if strings.Contains(p.Spec, "${") {
		f.errorf(at, "a format spec is fixed text for now: '${…}' inside it (a computed width) is not available yet (D171)")
		return f.toString(x, p.Expr.Span())
	}
	sp, msg := parseFmtSpec(p.Spec)
	if msg != "" {
		f.errorf(at, "format spec '%s' is not valid: %s; a spec is [[fill]align][sign][#][0][width][,][.precision][type] (D171)", p.Spec, msg)
		return f.toString(x, p.Expr.Span())
	}
	b := &synth{sp: p.Expr.Span()}
	checked := func() ast.Expr { return &ast.CheckedExpr{X: x, Pos: b.sp} }
	intLit := func(n int) ast.Expr { return &ast.IntLit{Text: strconv.Itoa(n), Pos: b.sp} }
	alignCode := func(def int) int {
		switch sp.align {
		case '<':
			return 1
		case '>':
			return 2
		case '^':
			return 3
		}
		return def
	}
	bad := func(why string) Expr {
		f.errorf(at, "format spec '%s' does not fit a value of type '%s': %s (D171)", p.Spec, t, why)
		return f.toString(x, p.Expr.Span())
	}
	call := func(fn string, args ...ast.Expr) Expr {
		return f.checkExpr(b.call(b.prelude(fn), args...), types.TString)
	}
	switch {
	case types.IsInteger(t):
		radix, prefix := 10, ""
		switch sp.typ {
		case 0, 'd':
		case 'x', 'X':
			radix, prefix = 16, "0x"
		case 'b':
			radix, prefix = 2, "0b"
		case 'o':
			radix, prefix = 8, "0o"
		default:
			return bad("an integer takes d, x, X, b or o")
		}
		if sp.precision >= 0 {
			return bad("a precision belongs to a float")
		}
		if sp.alt && radix == 10 {
			return bad("'#' adds a prefix to x, b and o")
		}
		if !sp.alt {
			prefix = ""
		}
		digits := b.callArgs(b.member(checked(), "toString"), b.namedArg("radix", intLit(radix)))
		return call("fmtNum", digits, b.str(sp.fill), intLit(alignCode(0)), intLit(signCode(sp.sign)), b.str(prefix),
			b.boolLit(sp.zero), b.boolLit(sp.group), b.boolLit(sp.typ == 'X'), b.str(""), intLit(sp.width))
	case types.IsFloat(t):
		suffix := ""
		var digits ast.Expr
		switch sp.typ {
		case 0, 'f', '%':
		default:
			return bad("a float takes f or %")
		}
		if sp.alt {
			return bad("'#' adds a prefix to x, b and o")
		}
		switch {
		case sp.typ == '%':
			prec := sp.precision
			if prec < 0 {
				prec = 6
			}
			suffix = "%"
			scaled := b.bin(lexer.Star, checked(), &ast.FloatLit{Text: "100.0", Pos: b.sp})
			digits = b.mcall(scaled, "toFixed", intLit(prec))
		case sp.typ == 'f' || sp.precision >= 0:
			prec := sp.precision
			if prec < 0 {
				prec = 6
			}
			digits = b.mcall(checked(), "toFixed", intLit(prec))
		default:
			digits = b.interp("", checked(), "")
		}
		return call("fmtNum", digits, b.str(sp.fill), intLit(alignCode(0)), intLit(signCode(sp.sign)), b.str(""),
			b.boolLit(sp.zero), b.boolLit(sp.group), b.boolLit(false), b.str(suffix), intLit(sp.width))
	}
	// text, and every type that prints: width, fill and alignment of what it prints
	if sp.typ != 0 && sp.typ != 's' {
		return bad("'" + string(sp.typ) + "' is for numbers; text takes s, or nothing")
	}
	if sp.precision >= 0 || sp.sign != 0 || sp.alt || sp.zero || sp.group {
		return bad("only a fill, an alignment and a width apply to text")
	}
	return call("fmtText", &ast.CheckedExpr{X: f.toString(x, p.Expr.Span()), Pos: b.sp}, b.str(sp.fill), intLit(alignCode(1)), intLit(sp.width))
}

func signCode(r rune) int {
	switch r {
	case '+':
		return 1
	case ' ':
		return 2
	}
	return 0
}
