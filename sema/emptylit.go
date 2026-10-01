package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D106: `var xs = []` and `val m = [:]` stay errors — the type is not
// visible at the declaration otherwise — but the message names the right
// kind (`MutableList` when the binding is changed later, `List` otherwise)
// and, when a later use in the same function decides the element type, the
// error carries the fix that writes the annotation.

// mutatingMethods change a collection in place; a binding that receives
// one of these calls wants the mutable kind.
var mutatingMethods = map[string]bool{
	"push": true, "pop": true, "insert": true, "removeAt": true, "addAll": true, "clear": true,
	"set": true, "remove": true, "add": true, "getOrPut": true, "sort": true, "sortWith": true,
	"swap": true, "fill": true, "reserve": true,
}

// emptyLiteralBinding reports `val/var name = []` (or `[:]`) written
// without a type; it returns false when s is not such a binding.
func (f *fnCtx) emptyLiteralBinding(s *ast.ValStmt) bool {
	b := s.Binding
	if b.Name == nil || b.Type != nil || s.Value == nil {
		return false
	}
	isMap, mut := false, false
	switch v := s.Value.(type) {
	case *ast.ListLit:
		if len(v.Elems) > 0 {
			return false
		}
		mut = v.Mut
	case *ast.MapLit:
		if len(v.Entries) > 0 {
			return false
		}
		mut = v.Mut
		isMap = true
	default:
		return false
	}
	name := b.Name.Name
	mutable, decided := f.laterUses(name, s.Pos.End, isMap)
	kind := "List"
	if isMap {
		kind = "Map"
	}
	if mutable || mut {
		kind = "Mutable" + kind
	}
	args := decided
	if args == "" {
		args = "i64"
		if isMap {
			args = "string, i64"
		}
	}
	written := kind + "<" + args + ">"
	what := "the element type of an empty list"
	if isMap {
		what = "the types of an empty map"
	}
	if decided == "" {
		f.errorf(s.Value.Span(), "cannot infer %s; annotate it, e.g. '%s %s: %s = %s' (D106)", what, s.Kind, name, written, srcText(s.Value))
	} else {
		at := source.Span{File: b.Name.Pos.File, Start: b.Name.Pos.End, End: b.Name.Pos.End}
		f.c.errorFix(s.Value.Span(), fixReplace("Write the type: "+written, at, ": "+written),
			"cannot infer %s here, though a later use decides it; write it: '%s %s: %s = %s' (D106)", what, s.Kind, name, written, srcText(s.Value))
	}
	// declared, so the uses report nothing more
	v := f.newVar(name, types.TInvalid, s.Kind == ast.BindVar, b.Name.Pos)
	f.declareChecked(name, v, b.Name.Pos)
	return true
}

// laterUses looks at what the function does with name after offset: whether
// it changes it in place, and the type arguments the first deciding use
// gives — a push, add, insert or set of a value whose type is plain to see,
// or passing it to a named function with a declared parameter type.
func (f *fnCtx) laterUses(name string, after int, isMap bool) (mutable bool, decided string) {
	if f.bodyAST == nil {
		return false, ""
	}
	walkAST(f.bodyAST, func(n any) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || call.Pos.Start < after {
			return true
		}
		if m, isMember := call.Fun.(*ast.MemberExpr); isMember {
			if x, isName := m.X.(*ast.NameExpr); isName && x.Name == name {
				if mutatingMethods[m.Name.Name] {
					mutable = true
				}
				if decided == "" {
					decided = f.decidingCall(m.Name.Name, call.Args, isMap)
				}
			}
			return true
		}
		if fn, isName := call.Fun.(*ast.NameExpr); isName && decided == "" {
			sym := f.scope.Lookup(fn.Name)
			if sym == nil || sym.Kind != SymFunc || len(sym.Func.TypeParams) > 0 {
				return true
			}
			f.c.resolveSignature(sym.Func)
			for i, a := range call.Args {
				if x, ok := a.Value.(*ast.NameExpr); ok && x.Name == name && i < len(sym.Func.Sig.Params) {
					switch pt := sym.Func.Sig.Params[i].Type.(type) {
					case *types.List:
						if !isMap {
							decided = pt.Elem.String()
							mutable = mutable || pt.Mutable
						}
					case *types.Map:
						if isMap {
							decided = pt.Key.String() + ", " + pt.Value.String()
							mutable = mutable || pt.Mutable
						}
					}
				}
			}
		}
		return true
	})
	return mutable, decided
}

// decidingCall is the type arguments `xs.push(1)`, `m.set("a", 2.5)` give.
func (f *fnCtx) decidingCall(method string, args []ast.Arg, isMap bool) string {
	switch {
	case isMap && method == "set" && len(args) == 2:
		k, v := f.plainType(args[0].Value), f.plainType(args[1].Value)
		if k != "" && v != "" {
			return k + ", " + v
		}
	case !isMap && (method == "push" || method == "add") && len(args) == 1:
		return f.plainType(args[0].Value)
	case !isMap && method == "insert" && len(args) == 2:
		return f.plainType(args[1].Value)
	}
	return ""
}

// plainType is the type of an expression that shows it without checking:
// a literal (with the literal defaults) or a local declared before.
func (f *fnCtx) plainType(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.IntLit:
		return "i64"
	case *ast.FloatLit:
		return "f64"
	case *ast.StringLit:
		return "string"
	case *ast.BoolLit:
		return "bool"
	case *ast.NameExpr:
		if sym := f.scope.Lookup(e.Name); sym != nil && sym.Kind == SymLocal && sym.Var.Type != nil && !types.IsInvalid(sym.Var.Type) {
			return sym.Var.Type.String()
		}
	}
	return ""
}
