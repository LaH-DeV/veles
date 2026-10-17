package sema

import (
	"reflect"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D158: in a method a bare name reaches the receiver's members — locals and
// parameters first, then the members of `this`, then the module and the
// prelude. The checker settles it before it reads a body: every bare name
// that is a member of the receiver, and that the body binds nowhere as a
// local, becomes `this.name` in the tree, so every later pass (narrowing,
// moves, resources, the index) sees the one spelling the language always had.
//
// A name the body both binds somewhere and reads as a member elsewhere is not
// rewritten at all: where the local is in scope it is the local, and where it
// is not the usual "unknown name" arrives with the fix that writes `this.`.
// That is the one case where a reader cannot tell which is meant (the spec's
// shadowing warning); asking for `this.` there is the same answer.

var (
	exprIface = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	typeIface = reflect.TypeOf((*ast.Type)(nil)).Elem()
	identType = reflect.TypeOf(ast.Ident{})
	spanType  = reflect.TypeOf(source.Span{})
)

// resolveBareMembers rewrites the bare member names of one method's body.
func (f *fnCtx) resolveBareMembers(d *ast.FunDecl) {
	if d == nil || f.selfVar == nil && !d.Static {
		return
	}
	if f.c.bareDone == nil {
		f.c.bareDone = map[*ast.FunDecl]bool{}
	}
	if f.c.bareDone[d] {
		return
	}
	f.c.bareDone[d] = true
	if d.Body == nil && d.ExprBody == nil {
		return
	}
	members := map[string]bool{}
	if f.selfVar != nil {
		fields, methods := f.memberNames(f.selfVar.Type)
		for _, m := range append(fields, methods...) {
			members[m] = true
		}
	}
	// the type's static members are reachable bare in every function of it
	statics := map[string]string{}
	if st, ok := f.env.self.(*types.Struct); ok {
		tmpl := templateOf(st)
		for name := range f.c.staticVals[tmpl] {
			statics[name] = tmpl.Name
		}
		for name, t := range f.c.methods[tmpl] {
			if t.Decl != nil && t.Decl.Static {
				statics[name] = tmpl.Name
			}
		}
	}
	if len(members) == 0 && len(statics) == 0 {
		return
	}
	// every name the body (and the parameters) could bind
	bound := map[string]bool{}
	for _, p := range d.Params {
		bound[p.Name.Name] = true
	}
	w := &bareWalker{members: members, statics: statics, bound: bound, seen: map[uintptr]bool{}}
	if d.Body != nil {
		w.walk(reflect.ValueOf(d.Body), true)
	}
	if d.ExprBody != nil {
		w.walk(reflect.ValueOf(&d.ExprBody).Elem(), true)
	}
	w.rewrite, w.seen = true, map[uintptr]bool{}
	w.shadows = func(ne *ast.NameExpr, target string) {
		if sym := f.scope.Lookup(ne.Name); sym != nil && sym.Kind != SymLocal {
			f.c.warnf(ne.Pos, "'%s' means %s.%s here (D158): a member wins over the module's own '%s'; rename one of them to say which you mean", ne.Name, target, ne.Name, ne.Name)
		}
	}
	if d.Body != nil {
		w.walk(reflect.ValueOf(d.Body), false)
	}
	if d.ExprBody != nil {
		w.walk(reflect.ValueOf(&d.ExprBody).Elem(), false)
	}
}

type bareWalker struct {
	members map[string]bool
	statics map[string]string // static member -> the type's name
	bound   map[string]bool
	rewrite bool
	seen    map[uintptr]bool
	// shadows is told of each bare member name before it is rewritten
	shadows func(ne *ast.NameExpr, target string)
}

// walk visits the tree under v. The first pass (collect) gathers every
// identifier that is not a member access, a label or part of a type; the
// second rewrites the bare names that are members and bound nowhere.
func (w *bareWalker) walk(v reflect.Value, collect bool) {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		if v.Type() == typeIface {
			return
		}
		if !collect && v.Type() == exprIface && v.CanSet() {
			if ne, ok := v.Interface().(*ast.NameExpr); ok && len(ne.TypeArgs) == 0 && w.statics[ne.Name] != "" && !w.members[ne.Name] && !w.bound[ne.Name] {
				if w.shadows != nil {
					w.shadows(ne, w.statics[ne.Name])
				}
				v.Set(reflect.ValueOf(&ast.MemberExpr{
					X:    &ast.NameExpr{Name: w.statics[ne.Name], Pos: ne.Pos},
					Name: ast.Ident{Name: ne.Name, Pos: ne.Pos},
					Pos:  ne.Pos,
				}))
				return
			}
			if ne, ok := v.Interface().(*ast.NameExpr); ok && len(ne.TypeArgs) == 0 && w.members[ne.Name] && !w.bound[ne.Name] {
				if w.shadows != nil {
					w.shadows(ne, "this")
				}
				v.Set(reflect.ValueOf(&ast.MemberExpr{
					X:    &ast.SelfExpr{Pos: ne.Pos},
					Name: ast.Ident{Name: ne.Name, Pos: ne.Pos},
					Pos:  ne.Pos,
				}))
				return
			}
		}
		w.walk(v.Elem(), collect)
	case reflect.Ptr:
		if v.IsNil() || w.seen[v.Pointer()] {
			return
		}
		w.seen[v.Pointer()] = true
		w.walk(v.Elem(), collect)
	case reflect.Struct:
		t := v.Type()
		if t == spanType {
			return
		}
		if t == identType {
			if collect {
				w.bound[v.Field(0).String()] = true
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			ft := t.Field(i)
			if !ft.IsExported() {
				continue
			}
			// a member access's name and a call argument's label name no
			// binding; types are not expressions
			if (t == reflect.TypeOf(ast.MemberExpr{}) && ft.Name == "Name") || (t == reflect.TypeOf(ast.Arg{}) && ft.Name == "Name") {
				continue
			}
			w.walk(v.Field(i), collect)
		}
	case reflect.Slice:
		if v.Type().Elem() == typeIface {
			return
		}
		for i := 0; i < v.Len(); i++ {
			w.walk(v.Index(i), collect)
		}
	}
}
