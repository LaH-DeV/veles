package sema

import (
	"sort"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// `static assert(cond, "why")` (D113): cond is a constant expression,
// checked when the program is compiled — at module level once, in a body
// once per instance of the function (so `T implements Trait`, D117, can be
// asserted of a generic's type argument). A failure quotes the reason and
// the values of the constants the condition names.

type moduleAssert struct {
	m *Module
	f *ast.File
	d *ast.StaticAssert
}

func (c *Checker) checkModuleAssert(a moduleAssert) {
	env := &typeEnv{module: a.m, file: a.f, tps: map[string]*types.TypeParam{}}
	f := c.newFnCtx(nil, a.m, a.f, env, nil)
	f.isGlobal = true
	f.staticAssert(a.d)
}

func (f *fnCtx) staticAssert(d *ast.StaticAssert) {
	cond := f.checkExprTo(d.Cond, types.TBool)
	if d.Reason == nil {
		return // the parser said a reason is needed
	}
	reason := f.checkExprTo(d.Reason, types.TString)
	if types.IsInvalid(cond.Type()) || types.IsInvalid(reason.Type()) {
		return
	}
	cv := f.c.evalConst(cond, d.Cond.Span())
	rv := f.c.evalConst(reason, d.Reason.Span())
	if cv == nil || rv == nil || cv.(*CBool).V {
		return
	}
	f.errorf(d.Pos, "static assert failed: %s%s", rv.(*CString).V, f.namedConstants(d.Cond))
}

// namedConstants lists the constants an expression names with their
// values, for a failed assertion: " (KB = 1024, LIMIT = 1000)".
func (f *fnCtx) namedConstants(e ast.Expr) string {
	seen := map[string]bool{}
	var parts []string
	var walk func(e ast.Expr)
	add := func(text string, sym *Symbol) {
		if sym == nil || sym.Kind != SymGlobal || sym.Global.Const == nil || seen[text] {
			return
		}
		seen[text] = true
		parts = append(parts, text+" = "+constString(sym.Global.Const))
	}
	walk = func(e ast.Expr) {
		switch e := e.(type) {
		case *ast.NameExpr:
			add(e.Name, f.lookup(e.Name))
		case *ast.MemberExpr:
			if n, ok := e.X.(*ast.NameExpr); ok {
				if mod := f.lookup(n.Name); mod != nil && mod.Kind == SymModule && mod.Mod != nil {
					add(n.Name+"."+e.Name.Name, mod.Mod.Scope.LookupLocal(e.Name.Name))
					return
				}
			}
			walk(e.X)
		case *ast.BinaryExpr:
			walk(e.L)
			walk(e.R)
		case *ast.UnaryExpr:
			walk(e.X)
		case *ast.CallExpr:
			walk(e.Fun)
			for _, a := range e.Args {
				walk(a.Value)
			}
		case *ast.ElvisExpr:
			walk(e.L)
			walk(e.R)
		case *ast.TupleExpr:
			for _, x := range e.Elems {
				walk(x)
			}
		}
	}
	walk(e)
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return " (" + strings.Join(parts, ", ") + ")"
}

// constString shows a constant as the reader would write it.
func constString(v ConstVal) string {
	join := func(xs []ConstVal) string {
		parts := make([]string, len(xs))
		for i, x := range xs {
			parts[i] = constString(x)
		}
		return strings.Join(parts, ", ")
	}
	switch v := v.(type) {
	case *CInt:
		if en, ok := v.T.(*types.Enum); ok {
			for _, m := range en.Members {
				if (m.Neg == (v.V.Sign() < 0)) && v.V.IsInt64() && absU64(v.V.Int64()) == m.Value {
					return en.Name + "." + m.Name
				}
			}
		}
		return v.V.String()
	case *CFloat:
		return formatFloat(v.V, types.BitSize(v.T) == 32)
	case *CBool:
		return strconv.FormatBool(v.V)
	case *CString:
		return strconv.Quote(v.V)
	case *CUnit:
		return "()"
	case *CNull:
		return "null"
	case *CSome:
		return constString(v.V)
	case *CTuple:
		return "(" + join(v.Elems) + ")"
	case *CStruct:
		parts := make([]string, len(v.Fields))
		for i, x := range v.Fields {
			parts[i] = v.T.Fields[i].Name + ": " + constString(x)
		}
		return v.T.Name + "(" + strings.Join(parts, ", ") + ")"
	case *CList:
		return "[" + join(v.Elems) + "]"
	case *CArray:
		return "[" + join(v.Elems) + "]"
	case *CSet:
		return "[" + join(v.Elems) + "]"
	case *CMap:
		parts := make([]string, len(v.Keys))
		for i, k := range v.Keys {
			parts[i] = constString(k) + ": " + constString(v.Vals[i])
		}
		if len(parts) == 0 {
			return "[:]"
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return "?"
}

func absU64(n int64) uint64 {
	if n < 0 {
		return uint64(-n)
	}
	return uint64(n)
}
