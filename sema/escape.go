package sema

import (
	"reflect"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D188: which arguments a function keeps. D63 moves a mutable collection into
// its immutable form only while nobody else can see the storage; passing it
// to a function used to count as "somebody else" always. A function that only
// reads and fills its parameter — calls the collection's own methods on it,
// loops over it, or hands it on to a function that does the same — does not
// keep it, so a collection passed to it is still unshared afterwards. This is
// a summary per parameter of a free function, computed on the syntax tree
// (so it does not depend on the order functions are checked in), memoized,
// and pessimistic: anything it cannot resolve, a cycle included, keeps.

type paramKey struct {
	decl *ast.FunDecl
	idx  int
}

const (
	paramUnknown int8 = iota
	paramComputing
	paramStays  // never kept
	paramLeaves // may be kept
)

// fileScope is the scope names at the top of a function in file resolve in.
func (c *Checker) fileScope(m *Module, file *ast.File) *Scope {
	scope := NewScope(m.Imports[file])
	if scope.parent == nil {
		scope = NewScope(m.Scope)
	}
	return scope
}

// identNames is every identifier written inside node: a binding, a
// parameter, a member name. A function name that is also in this set may be
// shadowed there, so a call by that name is not resolved.
func identNames(node any) map[string]bool {
	names := map[string]bool{}
	identType := reflect.TypeOf(ast.Ident{})
	spanType := reflect.TypeOf(source.Span{})
	astPkg := identType.PkgPath()
	seen := map[uintptr]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem())
			}
		case reflect.Ptr:
			// syntax nodes only: a ResolvedType reaches the (cyclic) type graph
			if v.IsNil() || v.Type().Elem().PkgPath() != astPkg || seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
			walk(v.Elem())
		case reflect.Struct:
			if v.Type().PkgPath() != astPkg && v.Type() != spanType {
				return
			}
			if v.Type() == identType {
				names[v.Field(0).String()] = true
				return
			}
			if v.Type() == spanType {
				return
			}
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node))
	return names
}

// calleeOf resolves the free function a call names: `f(x)` in scope, or
// `mod.f(x)` through a module. Anything else — a method, a closure, a
// constructor, a name the body also declares — is nil.
func (c *Checker) calleeOf(call *ast.CallExpr, scope *Scope, declared map[string]bool) *FuncTemplate {
	switch fn := call.Fun.(type) {
	case *ast.NameExpr:
		if declared[fn.Name] {
			return nil
		}
		if sym := scope.Lookup(fn.Name); sym != nil && sym.Kind == SymFunc {
			return sym.Func
		}
	case *ast.MemberExpr:
		ne, ok := fn.X.(*ast.NameExpr)
		if !ok || fn.Safe || declared[ne.Name] {
			return nil
		}
		if sym := scope.Lookup(ne.Name); sym != nil && sym.Kind == SymModule {
			if member := sym.Mod.Scope.LookupLocal(fn.Name.Name); member != nil && member.Kind == SymFunc {
				return member.Func
			}
		}
	}
	return nil
}

// argStays reports whether argument i of call is not kept by the function
// the call names: it resolves the callee, finds the parameter the argument
// binds, and asks its summary.
func (c *Checker) argStays(call *ast.CallExpr, i int, scope *Scope, declared map[string]bool) bool {
	if call.Async || call.Block {
		return false // a task keeps its arguments
	}
	a := call.Args[i]
	if a.Spread {
		return false
	}
	t := c.calleeOf(call, scope, declared)
	if t == nil {
		return false
	}
	d := t.Decl
	if d == nil {
		return false
	}
	idx := -1
	if a.Name != nil {
		for j := range d.Params {
			if d.Params[j].Name.Name == a.Name.Name {
				idx = j
			}
		}
	} else {
		idx = i
	}
	if idx < 0 || idx >= len(d.Params) {
		return false
	}
	return c.paramStays(t, idx)
}

// paramStays reports whether the function never keeps parameter idx.
func (c *Checker) paramStays(t *FuncTemplate, idx int) bool {
	d := t.Decl
	if d == nil || idx >= len(d.Params) {
		return false
	}
	key := paramKey{d, idx}
	switch c.paramState[key] {
	case paramStays:
		return true
	case paramLeaves, paramComputing:
		return false
	}
	if c.paramState == nil {
		c.paramState = map[paramKey]int8{}
	}
	c.paramState[key] = paramComputing
	ok := c.computeParamStays(t, idx)
	if ok {
		c.paramState[key] = paramStays
	} else {
		c.paramState[key] = paramLeaves
	}
	return ok
}

func (c *Checker) computeParamStays(t *FuncTemplate, idx int) bool {
	d := t.Decl
	p := d.Params[idx]
	// only a plain free function with a body of its own: a method sees its
	// receiver, an executor function runs elsewhere, a decorated function
	// runs inside a wrapper, an extern or trait function has no body here
	if d.Extern || d.ExportC || t.Owner != nil || t.Impl != nil || t.Trait != nil || t.Exec != nil {
		return false
	}
	if len(c.annotations[d]) > 0 || p.Variadic || p.Lazy || p.Pattern != nil {
		return false
	}
	var body any
	switch {
	case d.Body != nil:
		body = d.Body
	case d.ExprBody != nil:
		body = d.ExprBody
	default:
		return false
	}
	scope := c.fileScope(t.Module, t.File)
	declared := identNames(d)
	return c.nameStays(p.Name.Name, body, scope, declared)
}

// nameStays reports whether every mention of name in body is one that keeps
// nothing: the receiver of one of the collection's own methods, the subject
// of a `loop (x in name)` that binds no pointer, or an argument to a function
// that does not keep it. Reassigning it, a mention inside a lambda, returning
// it, storing it and every other use keep.
func (c *Checker) nameStays(name string, body any, scope *Scope, declared map[string]bool) bool {
	allowed := map[*ast.NameExpr]bool{}
	var uses []*ast.NameExpr
	kept := false
	walkAST(body, func(n any) bool {
		if kept {
			return false
		}
		switch n := n.(type) {
		case *ast.LambdaExpr:
			walkAST(n.Body, func(m any) bool {
				if ne, ok := m.(*ast.NameExpr); ok && ne.Name == name {
					kept = true
				}
				return !kept
			})
			return false
		case *ast.LoopStmt:
			if ne, ok := n.Iter.(*ast.NameExpr); ok && ne.Name == name {
				if n.Var != nil && hasRefBinding(n.Var) {
					kept = true
					return false
				}
				allowed[ne] = true
			}
		case *ast.CallExpr:
			if m, ok := n.Fun.(*ast.MemberExpr); ok && !m.Safe {
				if ne, ok := m.X.(*ast.NameExpr); ok && ne.Name == name && moveMethods[m.Name.Name] {
					allowed[ne] = true
				}
			}
			for i, a := range n.Args {
				if ne, ok := a.Value.(*ast.NameExpr); ok && ne.Name == name && c.argStays(n, i, scope, declared) {
					allowed[ne] = true
				}
			}
		case *ast.AssignStmt:
			if ne, ok := n.Target.(*ast.NameExpr); ok && ne.Name == name {
				kept = true
				return false
			}
		case *ast.NameExpr:
			if n.Name == name {
				uses = append(uses, n)
			}
		}
		return true
	})
	if kept {
		return false
	}
	for _, u := range uses {
		if !allowed[u] {
			return false
		}
	}
	return true
}

// freshCollection reports whether e builds a collection nobody else holds:
// a literal, a `toMutable()` copy, a constructor (`MutableSet<string>()`) or
// a static factory of a mutable collection type (`MutableList<bool>.repeat(…)`).
func freshCollection(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ListLit, *ast.MapLit:
		return true
	case *ast.CallExpr:
		switch fn := e.Fun.(type) {
		case *ast.NameExpr:
			return isMutableCollectionName(fn.Name)
		case *ast.MemberExpr:
			if fn.Safe {
				return false
			}
			if fn.Name.Name == "toMutable" {
				return true
			}
			ne, ok := fn.X.(*ast.NameExpr)
			return ok && isMutableCollectionName(ne.Name)
		}
	}
	return false
}

func isMutableCollectionName(name string) bool {
	return name == "MutableList" || name == "MutableMap" || name == "MutableSet"
}

// movableField: D63's move for `acc.items`, a field of a struct local that
// this function built and kept to itself. The struct may not be passed,
// assigned or returned; a method may be called on it when its summary says it
// keeps neither the struct nor the field's storage, and it does not use the
// field after the move (D201, move_method.go). The field may only be used the
// way a local collection may, and be built fresh — by the constructor's
// argument or the declared default — so no other name reaches the storage.
func (f *fnCtx) movableField(x Expr, span source.Span) bool {
	fg, ok := x.(*FieldGet)
	if !ok {
		return false
	}
	base, ok := fg.X.(*VarRef)
	if !ok {
		return false
	}
	v := base.Var
	if v.IsParam || v.IsGlobal || v.Captured || v.IsSelf || !f.vars[v] || f.bodyAST == nil {
		return false
	}
	st, ok := v.Type.(*types.Struct)
	if !ok {
		return false
	}
	field := fg.Name
	scope := f.c.fileScope(f.module, f.file)
	declared := identNames(f.bodyAST)

	var decl *ast.ValStmt
	var declLoops []*ast.LoopStmt
	var conv *ast.MemberExpr
	var convLoops []*ast.LoopStmt
	var loops []*ast.LoopStmt
	handled := map[*ast.NameExpr]bool{}
	allowed := map[*ast.MemberExpr]bool{}
	var mentions []*ast.MemberExpr
	methodCalls := map[*ast.MemberExpr]bool{}
	var calls []*ast.MemberExpr
	escaped := false

	var visit func(n any) bool
	visit = func(n any) bool {
		if escaped {
			return false
		}
		switch n := n.(type) {
		case *ast.LambdaExpr:
			walkAST(n.Body, func(m any) bool {
				if ne, ok := m.(*ast.NameExpr); ok && ne.Name == v.Name {
					escaped = true
				}
				return !escaped
			})
			return false
		case *ast.ValStmt:
			if n.Binding.Name != nil && n.Binding.Name.Pos == v.Span {
				decl = n
				declLoops = append([]*ast.LoopStmt(nil), loops...)
			}
			if me, ok := n.Value.(*ast.MemberExpr); ok && isFieldOf(me, v.Name, field) && copiesElements(n.Pattern) {
				allowed[me] = true
			}
		case *ast.LoopStmt:
			if me, ok := n.Iter.(*ast.MemberExpr); ok && isFieldOf(me, v.Name, field) {
				if n.Var != nil && hasRefBinding(n.Var) {
					escaped = true
					return false
				}
				allowed[me] = true
			}
			walkAST(n.Iter, visit)
			if n.Cond != nil {
				walkAST(n.Cond, visit)
			}
			loops = append(loops, n)
			walkAST(n.Body, visit)
			loops = loops[:len(loops)-1]
			return false
		case *ast.CallExpr:
			if m, ok := n.Fun.(*ast.MemberExpr); ok && !m.Safe {
				if me, ok := m.X.(*ast.MemberExpr); ok && isFieldOf(me, v.Name, field) && moveMethods[m.Name.Name] {
					allowed[me] = true
				}
				// a method of the struct, judged by its summary (D201)
				if ne, ok := m.X.(*ast.NameExpr); ok && ne.Name == v.Name && !structHasField(st, m.Name.Name) && !n.Async && !n.Block {
					methodCalls[m] = true
				}
			}
			for i, a := range n.Args {
				if me, ok := a.Value.(*ast.MemberExpr); ok && isFieldOf(me, v.Name, field) && f.c.argStays(n, i, scope, declared) {
					allowed[me] = true
				}
			}
		case *ast.AssignStmt:
			if me, ok := n.Target.(*ast.MemberExpr); ok && isFieldOf(me, v.Name, field) {
				escaped = true // reassigned: which collection is it now?
				return false
			}
		case *ast.MemberExpr:
			ne, ok := n.X.(*ast.NameExpr)
			if !ok || ne.Name != v.Name || ne.Pos.Start < v.Span.End {
				return true
			}
			handled[ne] = true
			switch {
			case n.Name.Name == field:
				if n.Pos == span {
					conv = n
					convLoops = append([]*ast.LoopStmt(nil), loops...)
				} else {
					mentions = append(mentions, n)
				}
			case structHasField(st, n.Name.Name):
				// another field: nothing to do with this one
			case methodCalls[n]:
				calls = append(calls, n)
			default:
				escaped = true // a method taken as a value holds the struct
			}
		case *ast.NameExpr:
			if n.Name == v.Name && n.Pos.Start >= v.Span.End && !handled[n] {
				escaped = true // the struct itself is passed, stored, aliased or returned
			}
		}
		return true
	}
	walkAST(f.bodyAST, visit)
	if escaped || decl == nil || conv == nil {
		return false
	}
	if !f.structBuiltFresh(decl, st, field) {
		return false
	}
	// each method called on the struct keeps neither it nor the field's
	// storage, and none after the move reads the field (D201)
	for _, call := range calls {
		facts := f.c.receiverSummary(st, call.Name.Name)
		if facts == nil || facts.aliased[field] {
			return false
		}
		if facts.used[field] && call.Pos.Start > conv.Pos.Start {
			return false
		}
	}
	for _, m := range mentions {
		if !allowed[m] && !f.inReturnPositionExpr(m) {
			return false
		}
	}
	if f.inReturnPositionExpr(conv) {
		return true
	}
	for _, m := range mentions {
		if m.Pos.Start > conv.Pos.Start {
			return false
		}
	}
	return len(convLoops) == len(declLoops)
}

// isFieldOf reports whether me is `<name>.<field>`.
func isFieldOf(me *ast.MemberExpr, name, field string) bool {
	ne, ok := me.X.(*ast.NameExpr)
	return ok && !me.Safe && ne.Name == name && me.Name.Name == field
}

func structHasField(st *types.Struct, name string) bool {
	for _, fl := range st.Fields {
		if fl.Name == name {
			return true
		}
	}
	return false
}

// structBuiltFresh: the `val acc = Acc(...)` that declares the struct builds
// `field` as a collection of its own — from an argument that is a fresh
// literal, or from the declared default — and the type has no `init { }`
// block that could hand the field to someone.
func (f *fnCtx) structBuiltFresh(decl *ast.ValStmt, st *types.Struct, field string) bool {
	call, ok := decl.Value.(*ast.CallExpr)
	if !ok || call.Async {
		return false
	}
	ne, ok := call.Fun.(*ast.NameExpr)
	if !ok {
		return false
	}
	sym := f.lookup(ne.Name)
	if sym == nil || sym.Kind != SymType {
		return false
	}
	tmpl := st
	if st.Template != nil {
		tmpl = st.Template
	}
	sd, ok := tmpl.Decl.(*ast.StructDecl)
	if !ok || sd.Init != nil || len(sd.InitParams) > 0 {
		return false
	}
	var fd *ast.Field
	fieldIdx := -1
	for i, d := range sd.Fields {
		if d.Name.Name == field {
			fd, fieldIdx = d, i
		}
	}
	if fd == nil {
		return false
	}
	for i, a := range call.Args {
		switch {
		case a.Spread:
			return false
		case a.Name != nil && a.Name.Name == field:
			return freshCollection(a.Value)
		case a.Name == nil && i == fieldIdx:
			return freshCollection(a.Value)
		}
	}
	return fd.Default != nil && freshCollection(fd.Default)
}

// inReturnPositionExpr is inReturnPosition for a member expression.
func (f *fnCtx) inReturnPositionExpr(me *ast.MemberExpr) bool {
	found := false
	walkAST(f.bodyAST, func(m any) bool {
		if found {
			return false
		}
		switch m := m.(type) {
		case *ast.LambdaExpr:
			return false
		case *ast.ReturnStmt:
			if m.Value == ast.Expr(me) {
				found = true
			}
		}
		return true
	})
	if found {
		return true
	}
	body := f.bodyAST
	if be, ok := body.(*ast.BlockExpr); ok {
		body = be.Block
	}
	switch b := body.(type) {
	case *ast.Block:
		if len(b.Stmts) > 0 {
			if es, ok := b.Stmts[len(b.Stmts)-1].(*ast.ExprStmt); ok && es.X == ast.Expr(me) {
				return true
			}
		}
	case ast.Expr:
		return b == ast.Expr(me)
	}
	return false
}

// lintRedundantCopy: `out.toList()` (or `toMap`, `toSet`) where a plain `out`
// would be accepted — the collection is not used again, so it is moved
// anyway (D63, D188) and the copy only costs. Only where the expected type is
// the immutable collection: a bare `val xs = out.toList()` would otherwise
// change xs's type if the call were dropped.
func (f *fnCtx) lintRedundantCopy(e *ast.CallExpr, recv Expr, name string) {
	switch w := numericHint(f.callWant).(type) {
	case *types.List:
		if w.Mutable {
			return
		}
	case *types.Map:
		if w.Mutable {
			return
		}
	case *types.Set:
		if w.Mutable {
			return
		}
	default:
		return
	}
	m, ok := e.Fun.(*ast.MemberExpr)
	if !ok || m.Safe {
		return
	}
	span := m.X.Span()
	if span.File == nil || f.skipStd(span) || !f.movable(recv, span) {
		return
	}
	text := spanText(span)
	var fix *source.Fix
	if text != "" {
		fix = fixReplace("Drop '."+name+"()': the collection is moved", e.Pos, text)
	}
	f.warnLint("redundant_copy", e.Pos, fix, "'.%s()' copies '%s', which is not used after this: without it the collection is moved immutable (D188)", name, text)
}
