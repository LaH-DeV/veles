package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D210: `lazy val name = value` at module level, or `static lazy val` in a
// type, is built on its first read, once. It is D146's `Lazy` with plain
// reads: the checker gives the name no storage of its own but a hidden
// global `Lazy(init: () => value)`, initialised with the other globals (a
// cheap handle, nothing of the value), and every read of the name is that
// global's `get()`. So the rules are `Lazy`'s: concurrent first readers wait
// for the one building it, a panic in the initializer is raised again by
// every later read, and the initializer is a `sendable` function that may
// not suspend or throw. The hidden global is ordered by dependency like any
// other, reaching into the initializer, so the values it reads are ready
// before the first read anywhere and a cycle through it is an error (D113).

// lazyGuard registers the hidden global of a `lazy val` declaration, and
// reports whether there is one; a refused `lazy` is registered as written,
// without it. owner is the type of a `static lazy val`.
func (c *Checker) lazyGuard(g *Global, d *ast.ValDecl, owner *types.Struct) bool {
	if !d.Lazy {
		return false
	}
	switch d.Kind {
	case ast.BindVar:
		c.errorf(d.Name.Pos, "a lazy value is built once and never changes: it is a 'lazy val', not a 'lazy var'; a value that changes is a 'var' or a 'Mutex' (D210)")
		return false
	case ast.BindConst:
		c.errorf(d.Name.Pos, "a 'const' is computed while compiling; 'lazy' is for a value built at run time, on its first read: write 'lazy val' or 'const' (D210)")
		return false
	}
	if d.Value == nil {
		return false // checkGlobal says it needs an initializer
	}
	at := d.Value.Span()
	build := &ast.LambdaExpr{Ret: d.Type, Body: d.Value, Pos: at}
	call := &ast.CallExpr{Fun: &ast.PreludeName{Name: "Lazy", Pos: d.Name.Pos},
		Args: []ast.Arg{{Name: &ast.Ident{Name: "init", Pos: at}, Value: build}}, Pos: at}
	hidden := &ast.ValDecl{Kind: ast.BindVal, Name: ast.Ident{Name: d.Name.Name + "$lazy", Pos: d.Name.Pos}, Value: call, Pos: d.Pos}
	guard := &Global{Name: g.Name + "$lazy", Display: g.Display, Span: g.Span}
	c.globals[guard] = hidden
	c.globalMod[guard] = c.globalMod[g]
	c.globalFile[guard] = c.globalFile[g]
	if owner != nil {
		c.staticOwner[guard] = owner // the initializer is inside its type
	}
	g.Guard = guard
	if c.lazyOf == nil {
		c.lazyOf, c.lazyInit, c.lazyDecl = map[*Global]*Global{}, map[source.Span]*Global{}, map[*Global]*ast.ValDecl{}
	}
	c.lazyOf[guard], c.lazyInit[at], c.lazyDecl[g] = g, g, d
	return true
}

// lazyRead is a read of the lazy value g at span: its guard's get(), whose
// caller location is the read, so a panic there names it.
func (f *fnCtx) lazyRead(g *Global, span source.Span) Expr {
	if f.constInit || f.fn != nil && f.fn.Const {
		f.errorf(span, "'%s' is a lazy value, built at run time on its first read; a constant cannot read it: make it a 'const' if the compiler can compute it (D210, D113)", g.Display)
		return bad()
	}
	v := f.globalVar(g.Guard)
	if types.IsInvalid(v.Type) {
		return bad()
	}
	get := &ast.CallExpr{Fun: &ast.MemberExpr{X: &ast.CheckedExpr{X: &VarRef{exprBase{v.Type}, v}, Pos: span},
		Name: ast.Ident{Name: "get"}, Pos: span}, Pos: span} // the member has no span: the read hovers as the value, not as Lazy.get
	build := f.checkExpr(get, nil)
	if types.IsInvalid(build.Type()) {
		return build
	}
	if g.Type == nil {
		g.Type = build.Type()
	}
	return &LazyRead{exprBase{g.Type}, f.globalVar(g), build}
}
