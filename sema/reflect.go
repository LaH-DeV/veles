package sema

import (
	"fmt"
	"math/big"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Reflection (D174), at compile time only. `T.fields` is the constant list
// of a struct's fields; a `const fun` reads it as data and an
// `inline loop (f in T.fields) { … }` unrolls once per field with `f` bound
// to a handle that is known while the body is checked, so each copy has the
// field's own type: `f.name`, `f.Type`, `f.get(x)`, `f.has<A>()`,
// `f.attr<A>()`, and `T.build(f => …)` to make a value field by field.

// fieldHandle is what the name of an `inline loop` (or of the lambda given
// to `T.build`) stands for in one unrolled copy: one field of one struct.
type fieldHandle struct {
	st    *types.Struct
	index int
	fld   *types.Field
	decl  *ast.Field // nil when the declaration is not at hand
	// member is set instead of fld for `inline loop (m in T.members)`: an
	// enum's member or a sealed family's variant (D203)
	member *memberRef
	// hover is what the editor shows on the loop's name, the same in
	// every copy of the body
	hover *Ref
}

// handleHover describes an inline loop's name for the editor: what it walks
// and the fields (or members) it stands for, one copy each.
func handleHover(name string, rt types.Type, what string, hs []*fieldHandle) *Ref {
	var items []string
	for _, h := range hs {
		if h.member != nil {
			items = append(items, h.member.name)
		} else {
			items = append(items, h.fld.Name+": "+h.fld.Type.String())
		}
	}
	owner := "?"
	if rt != nil {
		owner = rt.String()
	}
	kind := "field"
	if what == "members" {
		kind = "member"
	}
	detail := fmt.Sprintf("inline loop (%s in %s.%s)", name, owner, what)
	doc := fmt.Sprintf("`%s` stands for one %s of `%s` in each copy of the body — the loop is unrolled while compiling, so each copy has its own `%s.Type` (D174):\n\n", name, kind, owner, name)
	for _, it := range items {
		doc += "- `" + it + "`\n"
	}
	doc += "\nIt has `name`, `doc`, `index`" + map[bool]string{true: ", `Type` (a variant's), `has<A>()`, `attr<A>()`", false: ", `typeName`, `hasDefault`, `isOptional`, `isList`, `Type`, `Inner`, `get(x)`, `default()`, `has<A>()`, `attr<A>()`"}[kind == "member"] + "."
	return &Ref{Kind: "var", Name: name, Detail: detail, Doc: doc}
}

// refHandle records the inline loop's name at span (its binding or a use)
// once, whatever the number of copies.
func (c *Checker) refHandle(span source.Span, r *Ref) {
	if c.index == nil || r == nil || !span.IsValid() {
		return
	}
	for i := len(c.index.Refs) - 1; i >= 0 && i >= len(c.index.Refs)-64; i-- {
		if c.index.Refs[i].Span == span {
			return // an earlier copy recorded it
		}
	}
	ref := *r
	ref.Span = span
	c.index.Refs = append(c.index.Refs, ref)
}

// memberRef is one member of an enum or variant of a sealed family.
type memberRef struct {
	name, doc string
	variant   *types.Struct // the variant's type (`m.Type`); nil for an enum member
	node      any           // the declaration its annotations are kept by
}

// handleType is the type `f.Type` names: the field's, or the variant's; nil
// for an enum member.
func (h *fieldHandle) handleType() types.Type {
	if h.member != nil {
		if h.member.variant == nil {
			return nil
		}
		return h.member.variant
	}
	return h.fld.Type
}

// handleName is the field's or member's name.
func (h *fieldHandle) handleName() string {
	if h.member != nil {
		return h.member.name
	}
	return h.fld.Name
}

// memberHandles makes one handle per member of an enum or variant of a
// sealed family; ok is false when rt is neither (reported), and abstract
// when the type is not known yet.
func (f *fnCtx) memberHandles(rt types.Type, span source.Span) (hs []*fieldHandle, abstract, ok bool) {
	switch t := rt.(type) {
	case *types.TypeParam:
		return nil, true, true
	case *types.Enum:
		f.c.resolveEnum(t)
		d, _ := t.Decl.(*ast.EnumDecl)
		for i, m := range t.Members {
			ref := &memberRef{name: m.Name}
			if d != nil && i < len(d.Members) {
				ref.doc, ref.node = d.Members[i].Doc, d.Members[i]
			}
			hs = append(hs, &fieldHandle{index: i, member: ref})
		}
		return hs, false, true
	case *types.Sealed:
		for i, v := range t.Variants {
			ref := &memberRef{name: v.Name, variant: v, node: templateOf(v).Decl}
			if sd, isDecl := templateOf(v).Decl.(*ast.StructDecl); isDecl {
				ref.doc = sd.Doc
			}
			hs = append(hs, &fieldHandle{index: i, member: ref})
		}
		return hs, false, true
	}
	if !types.IsInvalid(rt) {
		f.errorf(span, "'%s' has no members to read: 'members' lists an enum's members or a sealed family's variants; a struct has 'fields' (D203)", rt)
	}
	return nil, false, false
}

// membersList is `T.members`: a constant List<MemberInfo>.
func (f *fnCtx) membersList(rt types.Type, e *ast.MemberExpr) Expr {
	mi, _ := f.c.preludeType("MemberInfo").(*types.Struct)
	list := &CList{T: &types.List{Elem: mi}}
	hs, abstract, ok := f.memberHandles(rt, e.Pos)
	if !ok {
		return bad()
	}
	if abstract || mi == nil {
		return constExpr(list)
	}
	owner := f.c.reflectMemberOwner(rt)
	for _, h := range hs {
		list.Elems = append(list.Elems, &CStruct{T: mi, Fields: []ConstVal{&CString{h.member.name}, &CString{h.member.doc}, i64Const(h.index), i64Const(owner)}})
	}
	return constExpr(list)
}

// reflectMemberOwner numbers an enum or sealed family, so a MemberInfo can
// say whose member it is (for `m.has<A>()` in a `const fun`).
func (c *Checker) reflectMemberOwner(t types.Type) int {
	for i, o := range c.reflMembers {
		if types.Identical(o, t) {
			return i
		}
	}
	c.reflMembers = append(c.reflMembers, t)
	return len(c.reflMembers) - 1
}

// memberRefAt is member index of a reflected enum or sealed family.
func (c *Checker) memberRefAt(t types.Type, index int) *memberRef {
	switch t := t.(type) {
	case *types.Enum:
		if d, ok := t.Decl.(*ast.EnumDecl); ok && index < len(d.Members) {
			return &memberRef{name: d.Members[index].Name.Name, node: d.Members[index]}
		}
	case *types.Sealed:
		if index < len(t.Variants) {
			v := t.Variants[index]
			return &memberRef{name: v.Name, variant: v, node: templateOf(v).Decl}
		}
	}
	return nil
}

// typeDoc is `T.doc`: the doc comment above a struct, enum or sealed trait,
// a constant (D203, for a command's help). false for any other type, or
// when the type has a static `doc` of its own.
func (f *fnCtx) typeDoc(rt types.Type) (Expr, bool) {
	doc := ""
	switch t := rt.(type) {
	case *types.TypeParam:
		// the generic body is checked again for each type it is used with
	case *types.Struct:
		if _, own := f.c.staticVals[templateOf(t)]["doc"]; own {
			return nil, false
		}
		if _, own := f.c.methods[templateOf(t)]["doc"]; own {
			return nil, false
		}
		if sd, ok := templateOf(t).Decl.(*ast.StructDecl); ok {
			doc = sd.Doc
		}
	case *types.Enum:
		if d, ok := t.Decl.(*ast.EnumDecl); ok {
			doc = d.Doc
		}
	case *types.Sealed:
		if d, ok := sealedTemplate(t).Decl.(*ast.TraitDecl); ok {
			doc = d.Doc
		}
	default:
		return nil, false
	}
	return &StringConst{exprBase{types.TString}, doc}, true
}

// memberAnnotation is the use of annotation a on a member, or nil.
func (c *Checker) memberAnnotation(m *memberRef, a *types.Struct) *AnnotationUse {
	if m.node == nil {
		return nil
	}
	for _, u := range c.annotations[m.node] {
		if u.Type == a {
			return u
		}
	}
	return nil
}

// handleAnnotation is the use of annotation a on what h stands for.
func (c *Checker) handleAnnotation(h *fieldHandle, a *types.Struct) *AnnotationUse {
	if h.member != nil {
		return c.memberAnnotation(h.member, a)
	}
	return c.annotationOnField(h.st, h.index, a)
}

// fieldDecls are the declared fields of a struct, in the order of its
// types.Field list.
func (c *Checker) fieldDecls(st *types.Struct) []*ast.Field {
	if sd, ok := templateOf(st).Decl.(*ast.StructDecl); ok && len(sd.Fields) == len(st.Fields) {
		return sd.Fields
	}
	return nil
}

func (c *Checker) fieldInfoStruct() *types.Struct {
	st, _ := c.preludeType("FieldInfo").(*types.Struct)
	return st
}

// reflectOwner numbers a struct, so a FieldInfo can say whose field it is.
func (c *Checker) reflectOwner(st *types.Struct) int {
	if i, ok := c.reflIndex[st]; ok {
		return i
	}
	c.reflOwners = append(c.reflOwners, st)
	c.reflIndex[st] = len(c.reflOwners) - 1
	return len(c.reflOwners) - 1
}

func i64Const(n int) *CInt { return &CInt{types.TI64, big.NewInt(int64(n))} }

// fieldsList is `T.fields` for a struct: a constant List<FieldInfo>.
func (c *Checker) fieldsList(st *types.Struct) *CList {
	fi := c.fieldInfoStruct()
	list := &CList{T: &types.List{Elem: fi}}
	if fi == nil {
		return list
	}
	owner := c.reflectOwner(st)
	decls := c.fieldDecls(st)
	for i, fld := range st.Fields {
		doc, hasDefault := "", fld.HasDefault
		if decls != nil {
			doc = decls[i].Doc
		}
		list.Elems = append(list.Elems, &CStruct{T: fi, Fields: []ConstVal{
			&CString{fld.Name}, &CString{doc}, &CString{fld.Type.String()}, &CBool{hasDefault},
			i64Const(i), i64Const(owner),
		}})
	}
	return list
}

// reflectTarget is the type a `T.fields` / `T.build(…)` receiver names: a
// type, a type parameter (its instance), or nil when it names neither.
func (f *fnCtx) reflectTarget(x ast.Expr) types.Type {
	switch x := x.(type) {
	case *ast.NameExpr:
		if f.lookup(x.Name) != nil && f.lookup(x.Name).Kind != SymType {
			return nil
		}
		return f.typeNamed(x, "fields")
	case *ast.MemberExpr:
		return f.moduleTypeNamed(x)
	case *ast.TypeExpr:
		return f.resolve(x.Type)
	}
	return nil
}

// fieldsExpr checks `T.fields`.
func (f *fnCtx) fieldsExpr(rt types.Type, e *ast.MemberExpr) Expr {
	fi := f.c.fieldInfoStruct()
	listT := &types.List{Elem: fi}
	switch t := rt.(type) {
	case *types.TypeParam:
		// the generic body is checked once without a type (and again for each
		// type it is used with, which is where the fields are read)
		return constExpr(&CList{T: listT})
	case *types.Struct:
		f.c.resolveStruct(templateOf(t))
		return constExpr(f.c.fieldsList(t))
	case *types.Sealed:
		f.errorf(e.Pos, "'%s' is a sealed trait: reflection reads the fields of a struct, and a family's variants are structs of their own (D174)", t.Name)
		return bad()
	}
	if !types.IsInvalid(rt) {
		f.errorf(e.Pos, "'%s' has no fields to read: reflection reads the fields of a struct (D174)", rt)
	}
	return bad()
}

// reflectMember is `T.fields` where T is a type; false when e is not that.
func (f *fnCtx) reflectMember(e *ast.MemberExpr) (Expr, bool) {
	if e.Name.Name != "fields" && e.Name.Name != "members" && e.Name.Name != "doc" || e.Safe {
		return nil, false
	}
	rt := f.reflectTarget(e.X)
	if rt == nil {
		return nil, false
	}
	if e.Name.Name == "doc" {
		return f.typeDoc(rt)
	}
	if e.Name.Name == "members" {
		if _, isEnum := rt.(*types.Enum); !isEnum {
			if _, isSealed := rt.(*types.Sealed); !isSealed {
				if _, isParam := rt.(*types.TypeParam); !isParam {
					return nil, false // a struct's own `members`, or none
				}
			}
		}
		return f.membersList(rt, e), true // D203
	}
	if st, ok := rt.(*types.Struct); ok {
		// a `static val fields` of the type itself still wins
		if _, own := f.c.staticVals[templateOf(st)]["fields"]; own {
			return nil, false
		}
	}
	return f.fieldsExpr(rt, e), true
}

// ---------------------------------------------------------------------------
// handles

// withHandle checks body with `name` standing for the handle h: a symbol for
// the members, and `name.Type` for the field's type in a type position.
func (f *fnCtx) withHandle(name string, h *fieldHandle, body func()) {
	f.pushScope()
	f.scope.Insert(&Symbol{Name: name, Kind: SymFieldHandle, Handle: h})
	saved := f.env
	env := *f.env
	env.handles = map[string]types.Type{}
	for k, v := range saved.handles {
		env.handles[k] = v
	}
	if t := h.handleType(); t != nil {
		env.handles[name] = t
	}
	f.env = &env
	body()
	f.env = saved
	f.popScope()
}

// handleMember is `f.name`, `f.doc`, … on a handle: constants.
func (f *fnCtx) handleMember(h *fieldHandle, e *ast.MemberExpr) Expr {
	if n, ok := e.X.(*ast.NameExpr); ok {
		f.c.refHandle(n.Pos, h.hover) // hover on the loop's name where it is used
	}
	if m := h.member; m != nil {
		switch e.Name.Name {
		case "name":
			return &StringConst{exprBase{types.TString}, m.name}
		case "doc":
			return &StringConst{exprBase{types.TString}, m.doc}
		case "index":
			return &IntConst{exprBase{types.TI64}, uint64(h.index), false}
		}
		f.errorf(e.Name.Pos, "a member handle has name, doc, index and, for a variant, Type, and the methods has and attr; '%s' is none of them (D203)", e.Name.Name)
		return bad()
	}
	switch e.Name.Name {
	case "name":
		return &StringConst{exprBase{types.TString}, h.fld.Name}
	case "doc":
		doc := ""
		if h.decl != nil {
			doc = h.decl.Doc
		}
		return &StringConst{exprBase{types.TString}, doc}
	case "typeName":
		return &StringConst{exprBase{types.TString}, h.fld.Type.String()}
	case "hasDefault":
		return &BoolConst{exprBase{types.TBool}, h.fld.HasDefault}
	case "index":
		return &IntConst{exprBase{types.TI64}, uint64(h.index), false}
	case "isOptional", "isList":
		return &BoolConst{exprBase{types.TBool}, handleFlag(h, e.Name.Name)}
	}
	f.errorf(e.Name.Pos, "a field handle has name, doc, typeName, hasDefault, index, isOptional, isList, Type and Inner, and the methods get, default, has and attr; '%s' is none of them (D174, D203)", e.Name.Name)
	return bad()
}

// handleFlag is `f.isOptional` (the field is a `T?`) or `f.isList` (a
// `List<T>` or `MutableList<T>`), D203.
func handleFlag(h *fieldHandle, name string) bool {
	if h.fld == nil {
		return false
	}
	switch types.Unaliased(h.fld.Type, false).(type) {
	case *types.Nullable:
		return name == "isOptional"
	case *types.List:
		return name == "isList"
	}
	return false
}

// handleInner is `f.Inner`: the `T` of a `T?` or `List<T>` field, or nil
// for any other (D203).
func handleInner(t types.Type) types.Type {
	switch tt := types.Unaliased(t, false).(type) {
	case *types.Nullable:
		return tt.Elem
	case *types.List:
		return tt.Elem
	}
	return nil
}

// handleConst is a condition made of a handle's constants — `f.isOptional`,
// `f.isList`, `f.hasDefault`, `f.has<A>()`, `f.name == "x"` — for an `if`
// that keeps only the branch it decides (D203); not known for anything else.
func (f *fnCtx) handleConst(cond ast.Expr) (value, known bool) {
	handle := func(x ast.Expr) *fieldHandle {
		n, ok := x.(*ast.NameExpr)
		if !ok {
			return nil
		}
		sym := f.lookup(n.Name)
		if sym == nil || sym.Kind != SymFieldHandle {
			return nil
		}
		return sym.Handle
	}
	switch c := cond.(type) {
	case *ast.CallExpr:
		return f.handleHas(c)
	case *ast.MemberExpr:
		h := handle(c.X)
		if h == nil || c.Safe {
			return false, false
		}
		switch c.Name.Name {
		case "isOptional", "isList":
			return handleFlag(h, c.Name.Name), true
		case "hasDefault":
			return h.fld != nil && h.fld.HasDefault, true
		}
	case *ast.BinaryExpr:
		if c.Op != lexer.Eq && c.Op != lexer.NotEq {
			return false, false
		}
		m, ok := c.L.(*ast.MemberExpr)
		lit, isLit := c.R.(*ast.StringLit)
		if !ok || !isLit || m.Name.Name != "name" || len(lit.Parts) != 1 || lit.Parts[0].Expr != nil {
			return false, false
		}
		h := handle(m.X)
		if h == nil {
			return false, false
		}
		return (h.handleName() == lit.Parts[0].Text) == (c.Op == lexer.Eq), true
	}
	return false, false
}

// annotationArg resolves the annotation type argument of `has<A>()` and
// `attr<A>()`.
func (f *fnCtx) annotationArg(typeArgs []types.Type, e *ast.CallExpr, what string) *types.Struct {
	if len(typeArgs) != 1 {
		f.errorf(e.Pos, "'%s' takes the annotation as its type argument: %s<Min>() (D174)", what, what)
		return nil
	}
	st, ok := typeArgs[0].(*types.Struct)
	if !ok || annotationOf(st) == nil {
		if !types.IsInvalid(typeArgs[0]) {
			f.errorf(e.Pos, "'%s' is not an annotation (D174)", typeArgs[0])
		}
		return nil
	}
	return st
}

// annotationOnField is the use of annotation a on a field, or nil.
func (c *Checker) annotationOnField(st *types.Struct, index int, a *types.Struct) *AnnotationUse {
	decls := c.fieldDecls(st)
	if decls == nil {
		return nil
	}
	for _, u := range c.annotations[decls[index]] {
		if u.Type == a {
			return u
		}
	}
	return nil
}

// handleHas is the compile-time answer of `f.has<A>()` where f is a field
// handle, for an `if` that keeps only the branch it decides; not known for
// any other call.
func (f *fnCtx) handleHas(c *ast.CallExpr) (value, known bool) {
	m, ok := c.Fun.(*ast.MemberExpr)
	if !ok || m.Name.Name != "has" || len(c.TypeArgs) != 1 || len(c.Args) != 0 {
		return false, false
	}
	n, ok := m.X.(*ast.NameExpr)
	if !ok {
		return false, false
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymFieldHandle {
		return false, false
	}
	a, _ := f.resolve(c.TypeArgs[0]).(*types.Struct)
	if a == nil || annotationOf(a) == nil {
		return false, false // reported when the call itself is checked
	}
	return f.c.handleAnnotation(sym.Handle, a) != nil, true
}

// handleCall is `f.get(x)`, `f.has<A>()` and `f.attr<A>()` on a handle.
func (f *fnCtx) handleCall(h *fieldHandle, callee *ast.MemberExpr, typeArgs []ast.Type, e *ast.CallExpr, want types.Type) Expr {
	if n, ok := callee.X.(*ast.NameExpr); ok {
		f.c.refHandle(n.Pos, h.hover)
	}
	var targs []types.Type
	for _, ta := range typeArgs {
		targs = append(targs, f.resolve(ta))
	}
	switch callee.Name.Name {
	case "has":
		a := f.annotationArg(targs, e, "has")
		if a == nil {
			return bad()
		}
		return &BoolConst{exprBase{types.TBool}, f.c.handleAnnotation(h, a) != nil}
	case "attr":
		a := f.annotationArg(targs, e, "attr")
		if a == nil {
			return bad()
		}
		nt := &types.Nullable{Elem: a}
		if u := f.c.handleAnnotation(h, a); u != nil && u.Value != nil {
			return constExpr(&CSome{T: nt, V: u.Value})
		}
		return constExpr(&CNull{T: nt})
	case "default", "get":
		if h.member != nil {
			f.errorf(callee.Name.Pos, "'%s' reads a field; a member handle has the methods has and attr (D203)", callee.Name.Name)
			f.checkArgsLoosely(e.Args)
			return bad()
		}
	}
	switch callee.Name.Name {
	case "default":
		// the declared default, as the constructor gives it (D203)
		if len(e.Args) != 0 {
			f.errorf(e.Pos, "'default' takes no arguments: f.default() is the field's declared default (D203)")
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		if !h.fld.HasDefault {
			f.errorf(callee.Name.Pos, "'%s' has no default: test 'f.hasDefault' before 'f.default()' (D203)", h.fld.Name)
			return bad()
		}
		return f.fieldDefault(h.st, h.index)
	case "get":
		if len(e.Args) != 1 || e.Args[0].Name != nil {
			f.errorf(e.Pos, "'get' takes the value to read the field from: f.get(x) (D174)")
			return bad()
		}
		// the field read as if written `x.name`. In a trait default body the
		// implementing type has opted in (it wrote `implement Trait`), so its
		// fields are readable whatever their visibility; anywhere else the
		// usual rules hold, and a generic function cannot read what a type
		// keeps private.
		if f.env != nil && f.env.self != nil {
			if self, ok := f.c.hooks.Subst(f.env.self, f.subst).(*types.Struct); ok && templateOf(self) == templateOf(h.st) {
				saved := f.reflectSelf
				f.reflectSelf = self
				defer func() { f.reflectSelf = saved }()
			}
		}
		return f.checkExpr(&ast.MemberExpr{X: e.Args[0].Value, Name: ast.Ident{Name: h.fld.Name, Pos: callee.Name.Pos}, Pos: e.Pos}, want)
	}
	f.errorf(callee.Name.Pos, "a field handle has the methods get, default, has and attr; '%s' is none of them (D174, D203)", callee.Name.Name)
	f.checkArgsLoosely(e.Args)
	return bad()
}

// ---------------------------------------------------------------------------
// the unrolled loop and `T.build`

// handlesOf makes one handle per field of the struct rt names, or nil with
// ok when the type is not known yet (a generic body checked without a type).
func (f *fnCtx) handlesOf(rt types.Type, span source.Span) (hs []*fieldHandle, abstract bool) {
	switch t := rt.(type) {
	case *types.TypeParam:
		return nil, true
	case *types.Struct:
		f.c.resolveStruct(templateOf(t))
		decls := f.c.fieldDecls(t)
		for i, fld := range t.Fields {
			h := &fieldHandle{st: t, index: i, fld: fld}
			if decls != nil {
				h.decl = decls[i]
			}
			hs = append(hs, h)
		}
		return hs, false
	}
	if !types.IsInvalid(rt) {
		f.errorf(span, "'%s' has no fields to read: reflection reads the fields of a struct (D174)", rt)
	}
	return nil, true
}

// inlineLoop checks `inline loop (f in T.fields) body`: the body once per
// field, in order, with f bound to that field.
func (f *fnCtx) inlineLoop(s *ast.LoopStmt) []Stmt {
	if s.Var != nil && s.Label == nil && len(s.Var.Tuple) == 2 && s.Var.Tuple[0].Name != nil && s.Var.Tuple[1].Name != nil {
		// `inline loop ((f, a) in T.annotated<A>())`
		if rt, a, ok := f.annotatedTarget(s.Iter); ok {
			return f.annotatedLoop(s, rt, a)
		}
	}
	switch {
	case s.Var == nil || s.Var.Name == nil || s.Var.Ref || len(s.Var.Tuple) > 0:
		f.errorf(s.Pos, "an inline loop names the field: 'inline loop (f in T.fields) { … }' (D174)")
		return nil
	case s.Label != nil:
		f.errorf(s.Label.Pos, "an inline loop is unrolled while compiling: it has no label, 'break' or 'continue' (D174)")
		return nil
	}
	m, ok := s.Iter.(*ast.MemberExpr)
	if !ok || m.Name.Name != "fields" && m.Name.Name != "members" || f.reflectTarget(m.X) == nil {
		f.errorf(s.Iter.Span(), "an inline loop walks the fields of a type or the members of an enum or sealed family: 'inline loop (f in T.fields)', 'inline loop (m in T.members)' (D174, D203)")
		return nil
	}
	var hs []*fieldHandle
	var abstract bool
	if m.Name.Name == "members" {
		var known bool
		hs, abstract, known = f.memberHandles(f.reflectTarget(m.X), m.Pos)
		if !known {
			return nil
		}
	} else {
		hs, abstract = f.handlesOf(f.reflectTarget(m.X), m.Pos)
	}
	if abstract {
		return nil // each type the generic body is used with is checked in its own copy
	}
	f.inlineBase = append(f.inlineBase, len(f.loops))
	defer func() { f.inlineBase = f.inlineBase[:len(f.inlineBase)-1] }()
	hover := handleHover(s.Var.Name.Name, f.reflectTarget(m.X), m.Name.Name, hs)
	f.c.refHandle(s.Var.Name.Pos, hover)
	var out []Stmt
	for _, h := range hs {
		h.hover = hover
		f.withHandle(s.Var.Name.Name, h, func() {
			out = append(out, f.checkBlock(s.Body, nil, false))
		})
	}
	return out
}

// inlineJump reports a `break`/`continue` that would leave an inline loop:
// the loop has no iterations to leave, only copies.
func (f *fnCtx) inlineJump(span source.Span, what string) bool {
	if n := len(f.inlineBase); n > 0 && len(f.loops) <= f.inlineBase[n-1] {
		f.errorf(span, "'%s' cannot leave an inline loop: it is unrolled while compiling, one copy of the body per field (D174)", what)
		return true
	}
	return false
}

// buildCall is `T.build(f => value)`: the struct made field by field, the
// body checked once per field with f bound to it and its value taken as
// that field's. false when the call is not one (a `static fun build` of the
// type, say).
func (f *fnCtx) buildCall(callee *ast.MemberExpr, e *ast.CallExpr, want types.Type) (Expr, bool) {
	if callee.Name.Name != "build" || len(e.Args) != 1 || e.Args[0].Name != nil {
		return nil, false
	}
	lam, ok := e.Args[0].Value.(*ast.LambdaExpr)
	if !ok || len(lam.Params) != 1 {
		return nil, false
	}
	rt := f.reflectTarget(callee.X)
	if rt == nil {
		return nil, false
	}
	if st, isStruct := rt.(*types.Struct); isStruct && f.staticNamed(st, "build") {
		return nil, false
	}
	hs, abstract := f.handlesOf(rt, callee.Pos)
	if abstract {
		return bad(), true
	}
	st := rt.(*types.Struct)
	var args []ast.Arg
	for _, h := range hs {
		if h.fld.Init {
			continue // the struct's `init` gives it a value
		}
		args = append(args, ast.Arg{
			Name:  &ast.Ident{Name: h.fld.Name, Pos: lam.Params[0].Pos},
			Value: &ast.BoundExpr{Param: lam.Params[0].Name.Name, Handle: h, Body: lam.Body, Pos: lam.Pos},
		})
	}
	// in a trait default body the implementing type has opted in (it wrote
	// `implement Trait`): its fields are set whatever their visibility, as
	// `f.get(x)` reads them, so a library builds a struct of another package
	if f.env != nil && f.env.self != nil {
		if self, ok := f.c.hooks.Subst(f.env.self, f.subst).(*types.Struct); ok && templateOf(self) == templateOf(st) {
			saved := f.reflectSelf
			f.reflectSelf = self
			defer func() { f.reflectSelf = saved }()
		}
	}
	return f.constructStruct(st, args, e.Pos, want), true
}

// boundExpr checks the body of a `T.build` lambda for one field.
func (f *fnCtx) boundExpr(e *ast.BoundExpr, want types.Type) Expr {
	h := e.Handle.(*fieldHandle)
	var out Expr
	f.withHandle(e.Param, h, func() {
		out = f.checkExprTo(e.Body, h.fld.Type)
	})
	return out
}

// ---------------------------------------------------------------------------
// reading annotations from a FieldInfo value, in a `const fun`

// inConstFun: this code runs in a function the compiler may run.
func (f *fnCtx) inConstFun() bool {
	for ctx := f; ctx != nil; ctx = ctx.parent {
		if ctx.fn != nil && ctx.fn.Const {
			return true
		}
	}
	return false
}

// fieldInfoCall is `f.has<A>()` / `f.attr<A>()` on a FieldInfo value (a
// local, such as the variable of `loop (f in T.fields)` in a `const fun`).
func (f *fnCtx) fieldInfoCall(callee *ast.MemberExpr, typeArgs []ast.Type, e *ast.CallExpr, want types.Type) (Expr, bool) {
	if callee.Name.Name != "has" && callee.Name.Name != "attr" {
		return nil, false
	}
	n, ok := callee.X.(*ast.NameExpr)
	if !ok {
		return nil, false
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymLocal {
		return nil, false
	}
	fi := f.c.fieldInfoStruct()
	mi, _ := f.c.preludeType("MemberInfo").(*types.Struct)
	if st, ok := types.Unaliased(sym.Var.Type, false).(*types.Struct); !ok || st != fi && st != mi {
		return nil, false
	}
	var targs []types.Type
	for _, ta := range typeArgs {
		targs = append(targs, f.resolve(ta))
	}
	a := f.annotationArg(targs, e, callee.Name.Name)
	if a == nil {
		return bad(), true
	}
	if !f.inConstFun() {
		f.errorf(callee.Name.Pos, "'%s' reads annotations, which exist while the program is compiled: use it in a 'const fun', or in 'inline loop (f in T.fields)' (D174)", callee.Name.Name)
		return bad(), true
	}
	recv := f.checkExpr(callee.X, nil)
	nt := &types.Nullable{Elem: a}
	if callee.Name.Name == "has" {
		return &Builtin{exprBase{types.TBool}, "reflect.has", []Expr{recv, &NullConst{exprBase{nt}}}, e.Pos}, true
	}
	return &Builtin{exprBase{nt}, "reflect.attr", []Expr{recv}, e.Pos}, true
}

// reflectAttr is the evaluator's `f.attr<A>()`: the annotation A on the
// field FieldInfo f describes, as an A?.
func (ev *constEval) reflectAttr(fieldInfo ConstVal, nt *types.Nullable) ConstVal {
	cs := fieldInfo.(*CStruct)
	if mi, _ := ev.c.preludeType("MemberInfo").(*types.Struct); mi != nil && cs.T == mi {
		// a MemberInfo (D203): name, doc, index, owner
		index := int(cs.Fields[2].(*CInt).V.Int64())
		owner := int(cs.Fields[3].(*CInt).V.Int64())
		a, _ := nt.Elem.(*types.Struct)
		if owner >= 0 && owner < len(ev.c.reflMembers) && a != nil {
			if ref := ev.c.memberRefAt(ev.c.reflMembers[owner], index); ref != nil {
				if u := ev.c.memberAnnotation(ref, a); u != nil && u.Value != nil {
					return &CSome{nt, u.Value}
				}
			}
		}
		return &CNull{nt}
	}
	owner := int(cs.Fields[5].(*CInt).V.Int64())
	index := int(cs.Fields[4].(*CInt).V.Int64())
	a, _ := nt.Elem.(*types.Struct)
	if owner >= 0 && owner < len(ev.c.reflOwners) && a != nil {
		if u := ev.c.annotationOnField(ev.c.reflOwners[owner], index, a); u != nil && u.Value != nil {
			return &CSome{nt, u.Value}
		}
	}
	return &CNull{nt}
}

// readsAnnotations: fn (a `const fun`) reads annotations, directly or
// through another function. Such a function has an answer only while the
// compiler runs it.
func (c *Checker) readsAnnotations(fn *Func, seen map[*Func]bool) bool {
	if v, ok := c.readsAnn[fn]; ok {
		return v
	}
	if seen[fn] || fn.Body == nil {
		return false
	}
	seen[fn] = true
	found := false
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Builtin:
			if n.Op == "reflect.attr" || n.Op == "reflect.has" {
				found = true
			}
		case *Call:
			if !found && c.readsAnnotations(n.Fn, seen) {
				found = true
			}
		}
	})
	c.readsAnn[fn] = found
	return found
}

// checkReflectUse refuses a call, from code that runs when the program
// does, of a function that reads annotations: they are gone by then.
func (c *Checker) checkReflectUse() {
	report := func(n any) {
		if call, ok := n.(*Call); ok && call.Fn.Const && c.readsAnnotations(call.Fn, map[*Func]bool{}) {
			c.errorf(call.Span, "'%s' reads annotations, which exist only while the program is compiled: call it in a constant, 'const X = %s(…)', and use that (D174)", call.Fn.Display, call.Fn.Display)
		}
	}
	for _, fn := range c.funcs {
		if !fn.Const && fn.Body != nil {
			walkBlock(fn.Body, report)
		}
	}
	for _, g := range c.prog.Globals {
		if g.Init != nil {
			walkExpr(g.Init, report)
		}
	}
}

// attrLetIf is `if (val a = f.attr<A>()) { … } else { … }` where f is a
// field handle: whether the field carries A is known while compiling, so
// only the branch that applies is checked (the other, written for a field of
// another type, may not type-check here), and `a` is the constant itself.
func (f *fnCtx) attrLetIf(e *ast.IfExpr, want types.Type, asValue bool) (Expr, bool) {
	lc, ok := e.Cond.(*ast.LetCond)
	if !ok {
		return nil, false
	}
	call, ok := lc.Value.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 || len(call.TypeArgs) != 1 {
		return nil, false
	}
	m, ok := call.Fun.(*ast.MemberExpr)
	if !ok || m.Name.Name != "attr" {
		return nil, false
	}
	n, ok := m.X.(*ast.NameExpr)
	if !ok {
		return nil, false
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymFieldHandle {
		return nil, false
	}
	a, _ := f.resolve(call.TypeArgs[0]).(*types.Struct)
	if a == nil || annotationOf(a) == nil {
		return nil, false // reported when the call is checked
	}
	use := f.c.handleAnnotation(sym.Handle, a)
	if use == nil || use.Value == nil {
		return f.staticIf(e, false, want, asValue), true
	}
	f.pushScope()
	defer f.popScope()
	v := f.newVar(lc.Name.Name, a, false, lc.Name.Pos)
	f.declareLocal(lc.Name.Name, v, lc.Name.Pos)
	markUsed(v)
	decl := &VarDecl{Var: v, Init: constExpr(use.Value)}
	r := f.staticIf(e, true, want, asValue)
	switch r := r.(type) {
	case *If:
		r.Then.Stmts = append([]Stmt{decl}, r.Then.Stmts...)
	case *BlockExpr:
		r.Block.Stmts = append([]Stmt{decl}, r.Block.Stmts...)
	}
	return r, true
}

// ---------------------------------------------------------------------------
// T.annotated<A>()

// annotatedTarget recognises `T.annotated<A>()`: the type T names and the
// annotation A, or nil when x is not that call.
func (f *fnCtx) annotatedTarget(x ast.Expr) (rt types.Type, a *types.Struct, ok bool) {
	call, isCall := x.(*ast.CallExpr)
	if !isCall || len(call.Args) != 0 || len(call.TypeArgs) != 1 {
		return nil, nil, false
	}
	m, isMember := call.Fun.(*ast.MemberExpr)
	if !isMember || m.Name.Name != "annotated" || m.Safe {
		return nil, nil, false
	}
	rt = f.reflectTarget(m.X)
	if rt == nil {
		return nil, nil, false
	}
	a, _ = f.resolve(call.TypeArgs[0]).(*types.Struct)
	if a == nil || annotationOf(a) == nil {
		f.errorf(call.TypeArgs[0].Span(), "'%s' is not an annotation: 'annotated<A>()' lists the fields that carry A (D174)", f.resolve(call.TypeArgs[0]))
		return rt, nil, true
	}
	return rt, a, true
}

// annotatedExpr is `T.annotated<A>()` as a value: a constant
// List<(FieldInfo, A)> of the fields that carry A, with the annotation's
// arguments, in declaration order — what a `const fun` wants to build a table.
func (f *fnCtx) annotatedExpr(rt types.Type, a *types.Struct, span source.Span) Expr {
	fi := f.c.fieldInfoStruct()
	if a == nil {
		return bad() // reported
	}
	pair := &types.Tuple{Elems: []types.Type{fi, a}}
	list := &CList{T: &types.List{Elem: pair}}
	if st, ok := rt.(*types.Struct); ok {
		f.c.resolveStruct(templateOf(st))
		infos := f.c.fieldsList(st)
		for i := range st.Fields {
			if u := f.c.annotationOnField(st, i, a); u != nil && u.Value != nil {
				list.Elems = append(list.Elems, &CTuple{T: pair, Elems: []ConstVal{infos.Elems[i], u.Value}})
			}
		}
	} else if _, abstract := rt.(*types.TypeParam); !abstract && !types.IsInvalid(rt) {
		f.errorf(span, "'%s' has no fields to read: reflection reads the fields of a struct (D174)", rt)
	}
	return constExpr(list)
}

// annotatedLoop is `inline loop ((f, a) in T.annotated<A>()) { … }`: the body
// once for each field that carries A, with f the field and a the annotation's
// value — the `has`/`attr` question asked and answered while compiling.
func (f *fnCtx) annotatedLoop(s *ast.LoopStmt, rt types.Type, a *types.Struct) []Stmt {
	if a == nil {
		return nil // reported
	}
	hs, abstract := f.handlesOf(rt, s.Iter.Span())
	if abstract {
		return nil
	}
	fn, an := s.Var.Tuple[0].Name, s.Var.Tuple[1].Name
	f.inlineBase = append(f.inlineBase, len(f.loops))
	defer func() { f.inlineBase = f.inlineBase[:len(f.inlineBase)-1] }()
	var out []Stmt
	for _, h := range hs {
		u := f.c.annotationOnField(h.st, h.index, a)
		if u == nil || u.Value == nil {
			continue
		}
		f.withHandle(fn.Name, h, func() {
			f.pushScope()
			v := f.newVar(an.Name, a, false, an.Pos)
			f.declareLocal(an.Name, v, an.Pos)
			markUsed(v)
			b := f.checkBlock(s.Body, nil, false)
			f.popScope()
			b.Stmts = append([]Stmt{&VarDecl{Var: v, Init: constExpr(u.Value)}}, b.Stmts...)
			out = append(out, b)
		})
	}
	return out
}

// annotatedCall checks `T.annotated<A>()` used as a value.
func (f *fnCtx) annotatedCall(e *ast.CallExpr) (Expr, bool) {
	rt, a, ok := f.annotatedTarget(e)
	if !ok {
		return nil, false
	}
	return f.annotatedExpr(rt, a, e.Pos), true
}
