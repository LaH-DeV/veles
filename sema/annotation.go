package sema

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// Annotations (D174): typed data on a declaration, declared with
// `annotation Name(params) on targets for Readers`. An annotation is a
// struct whose fields are the parameters, so a use `@Name(args)` is a
// construction whose arguments are constants; what the compiler adds is the
// list of places it may stand (`on`) and the traits whose reflection reads
// it (`for`). Nothing here changes how code is compiled: the uses are kept
// by declaration for the reflection (`T.fields`) to hand to libraries.
// The lowercase names (`@test`, `@inline`, …) belong to the compiler (D51).

// annotationTargets are the words after `on`: the kinds of declaration an
// annotation can stand on.
var annotationTargets = map[string]bool{
	"field": true, "struct": true, "fun": true, "trait": true, "enum": true, "enumMember": true, "variant": true,
}

// AnnotationUse is one resolved `@Name(args)` on a declaration.
type AnnotationUse struct {
	Type  *types.Struct // the annotation
	Value ConstVal      // its arguments as a constant of Type; nil after an error
	Attr  *ast.Attribute
}

// isUserAnnotation reports whether an attribute is written the way a user
// annotation is: through a module, or named like a type. A lowercase name
// that is not the compiler's is a mistake attrsOf reports.
func isUserAnnotation(a *ast.Attribute) bool {
	if knownAttrs[a.Name.Name] && a.Qual.Name == "" {
		return false
	}
	if a.Qual.Name != "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(a.Name.Name)
	return unicode.IsUpper(r)
}

// annotationOf is the declaration info of a struct that is an annotation,
// or nil.
func annotationOf(t types.Type) *ast.AnnotationInfo {
	st, ok := t.(*types.Struct)
	if !ok {
		return nil
	}
	if sd, ok := st.Decl.(*ast.StructDecl); ok {
		return sd.Annotation
	}
	return nil
}

// checkAnnotationDecl checks what an `annotation` declaration says by
// itself: a type-like name, known targets.
func (c *Checker) checkAnnotationDecl(d *ast.StructDecl) {
	info := d.Annotation
	if r, _ := utf8.DecodeRuneInString(d.Name.Name); !unicode.IsUpper(r) {
		c.errorf(d.Name.Pos, "an annotation is named like a type, '%s': lowercase names belong to the compiler (D174)", strings.ToUpper(d.Name.Name[:1])+d.Name.Name[1:])
	}
	seen := map[string]bool{}
	for _, t := range info.On {
		switch {
		case !annotationTargets[t.Name]:
			c.errorf(t.Pos, "'%s' is not a place an annotation can stand; the places are field, struct, fun, trait, enum, enumMember and variant (D174)", t.Name)
		case seen[t.Name]:
			c.errorf(t.Pos, "'%s' is listed twice after 'on' (D174)", t.Name)
		}
		seen[t.Name] = true
	}
}

// resolveAnnotations resolves every use of a user annotation in the
// package, once all types and impls exist: the type, the place it stands
// on, its arguments as a constant, and a warning when nothing it is read
// by can see it.
func (c *Checker) resolveAnnotations(mods []*Module) {
	if c.globalState == nil {
		c.globalState = map[*Global]int8{} // an argument may read a `static val` (`Duration.zero`)
	}
	for _, s := range c.structs {
		if annotationOf(s) != nil {
			c.annotationReaders(s) // reports a `for` that names no trait, used or not
			c.annotationWrapper(s) // and a wrapper that is not a function
		}
	}
	for _, m := range mods {
		for _, f := range m.Files {
			for _, d := range f.Decls {
				c.annotateDecl(m, f, d)
			}
		}
	}
}

func (c *Checker) annotateDecl(m *Module, f *ast.File, d ast.Decl) {
	switch d := d.(type) {
	case *ast.FunDecl:
		c.useAnnotations(m, f, d, d.Attrs, "fun", nil)
	case *ast.StructDecl:
		owner := c.moduleType(m, d.Name.Name)
		what := "struct"
		if d.Variant != nil {
			what = "variant"
		}
		c.useAnnotations(m, f, d, d.Attrs, what, owner)
		for _, fld := range d.Fields {
			c.useAnnotations(m, f, fld, fld.Attrs, "field", owner)
		}
		for _, fn := range d.Methods {
			c.useAnnotations(m, f, fn, fn.Attrs, "fun", owner)
		}
		for _, impl := range d.Impls {
			for _, fn := range impl.Methods {
				c.useAnnotations(m, f, fn, fn.Attrs, "fun", owner)
			}
		}
	case *ast.TraitDecl:
		c.useAnnotations(m, f, d, d.Attrs, "trait", nil)
		for _, fn := range d.Methods {
			c.useAnnotations(m, f, fn, fn.Attrs, "fun", nil)
		}
	case *ast.EnumDecl:
		owner := c.moduleType(m, d.Name.Name)
		c.useAnnotations(m, f, d, d.Attrs, "enum", owner)
		for _, mem := range d.Members {
			c.useAnnotations(m, f, mem, mem.Attrs, "enumMember", owner)
		}
	case *ast.ImplDecl:
		for _, fn := range d.Methods {
			c.useAnnotations(m, f, fn, fn.Attrs, "fun", nil)
		}
	case *ast.ValDecl:
		c.useAnnotations(m, f, d, d.Attrs, "value", nil)
	case *ast.TypeAliasDecl:
		c.useAnnotations(m, f, d, d.Attrs, "type", nil)
	case *ast.ErrorAliasDecl:
		c.useAnnotations(m, f, d, d.Attrs, "error", nil)
	}
}

// moduleType is the type a module-level declaration of that name made.
func (c *Checker) moduleType(m *Module, name string) types.Type {
	if sym := m.Scope.LookupLocal(name); sym != nil {
		return sym.Type
	}
	return nil
}

// useAnnotations resolves the user annotations among attrs and records
// them under key, the declaration node they stand on.
func (c *Checker) useAnnotations(m *Module, f *ast.File, key any, attrs []*ast.Attribute, what string, owner types.Type) {
	var uses []*AnnotationUse
	seen := map[*types.Struct]bool{}
	for _, a := range attrs {
		if !isUserAnnotation(a) {
			continue
		}
		u := c.resolveAnnotationUse(m, f, a, what, owner)
		if u == nil {
			continue
		}
		if seen[u.Type] {
			c.errorf(a.Pos, "duplicate annotation '@%s': an annotation stands once on a declaration (D174)", u.Type.Name)
			continue
		}
		seen[u.Type] = true
		uses = append(uses, u)
	}
	if len(uses) > 0 {
		c.annotations[key] = uses
		if fd, ok := key.(*ast.FunDecl); ok {
			c.applyDecorators(fd, uses)
		}
	}
}

func (c *Checker) resolveAnnotationUse(m *Module, f *ast.File, a *ast.Attribute, what string, owner types.Type) *AnnotationUse {
	path := []ast.Ident{a.Name}
	var fun ast.Expr = &ast.NameExpr{Name: a.Name.Name, Pos: a.Name.Pos}
	if a.Qual.Name != "" {
		path = []ast.Ident{a.Qual, a.Name}
		fun = &ast.MemberExpr{X: &ast.NameExpr{Name: a.Qual.Name, Pos: a.Qual.Pos}, Name: a.Name, Pos: a.Pos}
	}
	fc := c.newFnCtx(nil, m, f, &typeEnv{module: m, file: f, tps: map[string]*types.TypeParam{}}, nil)
	fc.isGlobal = true
	if a.Qual.Name == "" && fc.scope.Lookup(a.Name.Name) == nil {
		c.errorf(a.Pos, "unknown annotation '@%s': declare it with 'annotation %s(…) on field', or import it with 'use module { %s }' (D174)", a.Name.Name, a.Name.Name, a.Name.Name)
		return nil
	}
	t := fc.resolve(&ast.NamedType{Path: path, Pos: a.Pos})
	if types.IsInvalid(t) {
		return nil
	}
	info := annotationOf(t)
	if info == nil {
		c.errorf(a.Pos, "'@%s' is not an annotation: an annotation is declared with 'annotation %s(…) on field' (D174)", a.Name.Name, a.Name.Name)
		return nil
	}
	st := t.(*types.Struct)
	if sd, ok := st.Decl.(*ast.StructDecl); ok {
		c.refType(a.Name.Pos, st.Name, st, sd.Name.Pos) // hover and go-to-definition on `@Min`
	}
	if what == "value" || what == "type" || what == "error" || !hasTarget(info, what) {
		c.errorf(a.Pos, "'@%s' stands on %s, and cannot stand on %s (D174)", st.Name, listTargets(info), describeTarget(what))
		return nil
	}
	args := a.Args
	if len(args) == 1 && args[0].Name == nil && !args[0].Spread && len(st.Fields) == 1 {
		// the one parameter may be given bare: @Min(18)
		args = []ast.Arg{{Name: &ast.Ident{Name: st.Fields[0].Name, Pos: args[0].Value.Span()}, Value: args[0].Value}}
	}
	call := &ast.CallExpr{Fun: fun, Args: args, Pos: a.Pos}
	init := fc.checkExprTo(call, st)
	u := &AnnotationUse{Type: st, Attr: a}
	if !types.IsInvalid(init.Type()) && info.Wrapper == nil {
		// a decorator's arguments are built on each call (D176), so any value
		// will do; data is read while compiling, so it must be a constant
		u.Value = c.constValue(init, st, a.Pos, "@"+st.Name)
	}
	c.warnUnread(m, f, a, st, info, what, owner)
	return u
}

func hasTarget(info *ast.AnnotationInfo, what string) bool {
	for _, t := range info.On {
		if t.Name == what {
			return true
		}
	}
	return false
}

func listTargets(info *ast.AnnotationInfo) string {
	var names []string
	for _, t := range info.On {
		names = append(names, t.Name)
	}
	if len(names) == 0 {
		return "nothing"
	}
	return strings.Join(names, ", ")
}

func describeTarget(what string) string {
	switch what {
	case "fun":
		return "a function"
	case "enumMember":
		return "an enum member"
	case "value":
		return "a value"
	case "type":
		return "a type alias"
	case "error":
		return "an error set"
	}
	return "a " + what
}

// warnUnread says that nothing reads an annotation on this declaration: it
// names the traits whose reflection reads it (`for`), and the type it
// belongs to implements none of them (D51's rule, kept: nothing is
// silently ignored).
func (c *Checker) warnUnread(m *Module, f *ast.File, a *ast.Attribute, st *types.Struct, info *ast.AnnotationInfo, what string, owner types.Type) {
	if len(info.For) == 0 || owner == nil {
		return
	}
	switch what {
	case "field", "enumMember", "variant", "struct", "enum":
	default:
		return
	}
	switch o := owner.(type) {
	case *types.Struct:
		if len(o.TypeParams) > 0 {
			return
		}
	case *types.Sealed:
		return
	}
	readers := c.annotationReaders(st)
	if len(readers) == 0 {
		return
	}
	fc := c.newFnCtx(nil, m, f, &typeEnv{module: m, file: f, tps: map[string]*types.TypeParam{}}, nil)
	names := make([]string, 0, len(readers))
	for _, tr := range readers {
		if fc.implements(owner, tr) {
			return
		}
		names = append(names, tr.Name)
	}
	c.warnf(a.Pos, "nothing reads '@%s' here: it is read by %s, and '%s' does not implement %s (D174)", st.Name, strings.Join(names, " or "), owner, pick(len(names) == 1, "it", "one of them"))
}

func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// annotationReaders resolves the `for` clause of an annotation to traits,
// once; a name that is not a trait is reported there.
func (c *Checker) annotationReaders(st *types.Struct) []*types.Trait {
	if rs, ok := c.annReaders[st]; ok {
		return rs
	}
	var out []*types.Trait
	ctx := c.structDecl[st]
	sd, _ := st.Decl.(*ast.StructDecl)
	if ctx != nil && sd != nil && sd.Annotation != nil {
		fc := c.newFnCtx(nil, ctx.module, ctx.file, &typeEnv{module: ctx.module, file: ctx.file, tps: map[string]*types.TypeParam{}}, nil)
		fc.isGlobal = true
		for _, rt := range sd.Annotation.For {
			t := fc.resolve(rt)
			if types.IsInvalid(t) {
				continue
			}
			tr, ok := t.(*types.Trait)
			if !ok {
				c.errorf(rt.Span(), "'%s' is not a trait; after 'for' an annotation names the traits whose reflection reads it (D174)", t)
				continue
			}
			out = append(out, tr)
		}
	}
	c.annReaders[st] = out
	return out
}

// ---------------------------------------------------------------------------
// decorators (D176)

// annWrapper is the function a decorator annotation names (`= retryCall`):
// arity 2 takes (annotation, body), arity 3 (annotation, args, body).
type annWrapper struct {
	sym   *Symbol
	arity int
}

// annotationWrapper resolves `= wrapper` of an annotation declaration, once;
// nil for a data annotation or after an error.
func (c *Checker) annotationWrapper(st *types.Struct) *annWrapper {
	if w, ok := c.annWrappers[st]; ok {
		return w
	}
	c.annWrappers[st] = nil
	sd, _ := st.Decl.(*ast.StructDecl)
	ctx := c.structDecl[st]
	if sd == nil || sd.Annotation == nil || sd.Annotation.Wrapper == nil || ctx == nil {
		return nil
	}
	info := sd.Annotation
	if len(info.On) != 1 || info.On[0].Name != "fun" {
		c.errorf(info.WrapperPos, "an annotation with a wrapper stands on functions and nothing else: 'annotation %s(…) on fun = wrapper' (D176)", st.Name)
		return nil
	}
	fc := c.newFnCtx(nil, ctx.module, ctx.file, &typeEnv{module: ctx.module, file: ctx.file, tps: map[string]*types.TypeParam{}}, nil)
	var sym *Symbol
	switch w := info.Wrapper.(type) {
	case *ast.NameExpr:
		sym = fc.lookup(w.Name)
	case *ast.MemberExpr:
		if n, ok := w.X.(*ast.NameExpr); ok {
			if ms := fc.lookup(n.Name); ms != nil && ms.Kind == SymModule {
				sym = ms.Mod.Scope.LookupLocal(w.Name.Name)
			}
		}
	}
	span := info.Wrapper.Span()
	if sym == nil {
		c.errorf(span, "the wrapper of '@%s' is not a function in scope: write the name of an ordinary function, 'annotation %s(…) on fun = wrapper' (D176)", st.Name, st.Name)
		return nil
	}
	if sym.Kind != SymFunc || sym.Func == nil || sym.Func.Decl == nil {
		c.errorf(span, "the wrapper of '@%s' must be a function (D176)", st.Name)
		return nil
	}
	ps := sym.Func.Decl.Params
	if len(ps) != 2 && len(ps) != 3 {
		c.errorf(span, "the wrapper '%s' of '@%s' takes the annotation and the body — '(a: %s, body: fun(): R)' — or the annotation, the arguments and the body — '(a: %s, args: A, body: fun(A): R)' (D176)", sym.Name, st.Name, st.Name, st.Name)
		return nil
	}
	if nt, ok := ps[0].Type.(*ast.NamedType); !ok || nt.Path[len(nt.Path)-1].Name != st.Name {
		c.errorf(span, "the first parameter of the wrapper '%s' must be the annotation '%s' (D176)", sym.Name, st.Name)
		return nil
	}
	w := &annWrapper{sym: sym, arity: len(ps)}
	c.annWrappers[st] = w
	return w
}

// applyDecorators rewrites a function that carries decorator annotations:
// its body becomes a lambda handed to the wrapper, and the wrapper's result
// is the function's. Several nest, the first written outermost.
func (c *Checker) applyDecorators(fd *ast.FunDecl, uses []*AnnotationUse) {
	var decs []*AnnotationUse
	for _, u := range uses {
		if c.annotationWrapper(u.Type) != nil {
			decs = append(decs, u)
		}
	}
	if len(decs) == 0 {
		return
	}
	first := decs[0].Attr
	switch {
	case fd.Body == nil && fd.ExprBody == nil:
		c.errorf(first.Pos, "a decorator wraps a body, and '%s' has none (D176)", fd.Name.Name)
		return
	case fd.Test || fd.Extern:
		c.errorf(first.Pos, "a decorator cannot wrap a test or an extern function (D176)")
		return
	}
	var inner ast.Expr
	if fd.Body != nil {
		inner = &ast.BlockExpr{Block: fd.Body}
	} else {
		inner = fd.ExprBody
	}
	for i := len(decs) - 1; i >= 0; i-- {
		u := decs[i]
		w := c.annotationWrapper(u.Type)
		pos := u.Attr.Pos
		args := u.Attr.Args
		if len(args) == 1 && args[0].Name == nil && !args[0].Spread && len(u.Type.Fields) == 1 {
			args = []ast.Arg{{Name: &ast.Ident{Name: u.Type.Fields[0].Name, Pos: pos}, Value: args[0].Value}}
		}
		value := &ast.CallExpr{Fun: &ast.TypeExpr{Type: &ast.ResolvedType{T: u.Type, Pos: pos}, Pos: pos}, Args: args, Pos: pos}
		lambda := &ast.LambdaExpr{Body: inner, Pos: pos}
		callArgs := []ast.Arg{{Value: value}}
		if w.arity == 3 {
			var names []ast.Expr
			for _, p := range fd.Params {
				if p.Variadic {
					c.errorf(pos, "a decorator that takes the arguments cannot wrap a function with a variadic parameter ('%s'); use a wrapper of the annotation and the body only (D176)", p.Name.Name)
					return
				}
				lambda.Params = append(lambda.Params, ast.Param{Name: p.Name, Type: p.Type, Pos: p.Pos})
				names = append(names, &ast.NameExpr{Name: p.Name.Name, Pos: p.Name.Pos})
			}
			var tuple ast.Expr
			if len(names) == 1 {
				tuple = names[0]
			} else {
				tuple = &ast.TupleExpr{Elems: names, Pos: pos}
			}
			callArgs = append(callArgs, ast.Arg{Value: tuple})
		}
		callArgs = append(callArgs, ast.Arg{Value: lambda})
		inner = &ast.CallExpr{Fun: &ast.SymbolRef{Sym: w.sym, Name: w.sym.Name, Pos: pos}, Args: callArgs, Pos: pos}
		if w.sym.Func.Decl.Effects.Throws {
			// the wrapper's errors are the function's (`try`), whatever E turns out to be
			inner = &ast.TryExpr{X: inner, Pos: pos}
		}
	}
	if fd.Body != nil {
		fd.Body = &ast.Block{Stmts: []ast.Stmt{&ast.ExprStmt{X: inner}}, Pos: fd.Body.Pos}
	} else {
		fd.ExprBody = inner
	}
}
