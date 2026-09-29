package main

// Go references for the compiler-shaped benchmarks (checklist S2): the same
// work as bench/{lexer,ast,intern,emit}/main.vs, the same checksums. Building
// the input is bracketed with setup() so the runner leaves it out, as the
// Veles side does by starting its clock afterwards.

import (
	"strconv"
	"strings"
	"time"
)

func setup(start time.Time) { goSetup += time.Since(start) }

func init() {
	references["lexer"] = refLexer
	references["ast"] = refAST
	references["intern"] = refIntern
	references["emit"] = refEmit
}

var lexKeywords = map[string]int64{
	"fun": 1, "val": 2, "var": 3, "if": 4, "else": 5, "return": 6,
	"loop": 7, "when": 8, "struct": 9, "enum": 10, "use": 11, "throws": 12,
}

type lexToken struct {
	kind       int64 // Ident 0, Keyword 1, Number 2, Str 3, Comment 4, Punct 5
	start, end int64
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b >= 128
}
func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func tokenise(src string) []lexToken {
	var out []lexToken
	n := int64(len(src))
	var i int64
	for i < n {
		b := src[i]
		start := i
		switch {
		case b == ' ' || b == '\n' || b == '\t':
			i++
		case isIdentStart(b):
			for i < n && (isIdentStart(src[i]) || isDigit(src[i])) {
				i++
			}
			kind := int64(0)
			if _, ok := lexKeywords[src[start:i]]; ok {
				kind = 1
			}
			out = append(out, lexToken{kind, start, i})
		case isDigit(b):
			for i < n && (isDigit(src[i]) || src[i] == '.') {
				i++
			}
			out = append(out, lexToken{2, start, i})
		case b == '"':
			i++
			for i < n && src[i] != '"' {
				i++
			}
			i++
			out = append(out, lexToken{3, start, i})
		case b == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
			out = append(out, lexToken{4, start, i})
		default:
			i++
			out = append(out, lexToken{5, start, i})
		}
	}
	return out
}

func refLexer() string {
	t0 := time.Now()
	var sb strings.Builder
	for i := 0; i < 20000; i++ {
		n := strconv.Itoa(i)
		sb.WriteString("fun add" + n + "(a: i64, b: i64): i64 {\n")
		sb.WriteString("  val x = a + b * " + n + " // scale\n")
		sb.WriteString("  if (x > 10) return x else return \"s" + n + "\"\n")
		sb.WriteString("}\n")
	}
	src := sb.String()
	setup(t0)

	var check, tokens int64
	for r := 0; r < 3; r++ {
		toks := tokenise(src)
		tokens += int64(len(toks))
		for _, t := range toks {
			check += (t.end - t.start) * (t.kind + 1)
		}
	}
	return strconv.FormatInt(check, 10)
}

type astExpr interface{}
type astNum struct{ value int64 }
type astNeg struct{ operand astExpr }
type astAdd struct{ left, right astExpr }
type astMul struct{ left, right astExpr }
type astCond struct{ test, yes, no astExpr }

func astBuild(depth, seed int64) astExpr {
	if depth == 0 {
		return &astNum{seed%7 + 1}
	}
	s := seed*1103515245 + 12345
	pick := (s >> 8) % 5
	a := astBuild(depth-1, s%1000003)
	b := astBuild(depth-1, (s>>3)%1000003)
	switch pick {
	case 0:
		return &astNeg{a}
	case 1:
		return &astAdd{a, b}
	case 2:
		return &astMul{a, b}
	case 3:
		return &astCond{a, b, a}
	}
	return &astAdd{b, a}
}

func astEval(e astExpr) int64 {
	switch e := e.(type) {
	case *astNum:
		return e.value
	case *astNeg:
		return 0 - astEval(e.operand)
	case *astAdd:
		return (astEval(e.left) + astEval(e.right)) % 1000003
	case *astMul:
		return (astEval(e.left) * astEval(e.right)) % 1000003
	case *astCond:
		if astEval(e.test)%2 == 0 {
			return astEval(e.yes)
		}
		return astEval(e.no)
	}
	panic("unreachable")
}

func astCount(e astExpr) int64 {
	switch e := e.(type) {
	case *astNum:
		return 1
	case *astNeg:
		return 1 + astCount(e.operand)
	case *astAdd:
		return 1 + astCount(e.left) + astCount(e.right)
	case *astMul:
		return 1 + astCount(e.left) + astCount(e.right)
	case *astCond:
		return 1 + astCount(e.test) + astCount(e.yes) + astCount(e.no)
	}
	panic("unreachable")
}

func refAST() string {
	var check, nodes int64
	for r := int64(0); r < 6; r++ {
		tree := astBuild(17, r+1)
		nodes += astCount(tree)
		for i := 0; i < 4; i++ {
			check += astEval(tree)
		}
	}
	return strconv.FormatInt(check, 10)
}

func refIntern() string {
	t0 := time.Now()
	pool := make([]string, 0, 50000)
	for i := int64(0); i < 50000; i++ {
		pool = append(pool, "sym_"+strconv.FormatInt(i*7919%100003, 10)+"_x")
	}
	stream := make([]string, 0, 1000000)
	x := int64(88172645463325252)
	for i := 0; i < 1000000; i++ {
		x = x ^ (x << 13)
		x = x ^ (x >> 7)
		x = x ^ (x << 17)
		stream = append(stream, pool[(x&0x7fffffff)%50000])
	}
	setup(t0)

	var check int64
	ids := map[string]int64{}
	var names []string
	for _, s := range stream {
		id, ok := ids[s]
		if !ok {
			id = int64(len(names))
			ids[s] = id
			names = append(names, s)
		}
		check += id
	}
	check += int64(len(names))
	return strconv.FormatInt(check, 10)
}

func emitFunction(sb *strings.Builder, index int64) {
	line := func(s string) { sb.WriteString(s); sb.WriteString("\n") }
	idx := strconv.FormatInt(index, 10)
	line("define i64 @f" + idx + "(i64 %a, i64 %b) {")
	line("entry:")
	for t := int64(0); t < 40; t++ {
		prev := "%a"
		if t != 0 {
			prev = "%t" + strconv.FormatInt(t-1, 10)
		}
		line("  %t" + strconv.FormatInt(t, 10) + " = add i64 " + prev + ", " + strconv.FormatInt(t*3+index%11, 10))
	}
	line("  %cmp = icmp slt i64 %t39, %b")
	line("  br i1 %cmp, label %yes, label %no")
	line("yes:")
	line("  ret i64 %t39")
	line("no:")
	line("  ret i64 %b")
	line("}")
}

func refEmit() string {
	var check int64
	for r := 0; r < 4; r++ {
		var sb strings.Builder
		for f := int64(0); f < 5000; f++ {
			emitFunction(&sb, f)
		}
		check += int64(len(sb.String()))
	}
	return strconv.FormatInt(check, 10)
}
