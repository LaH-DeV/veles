package sema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Index records what every resolved name in the package refers to. It is
// built for editor tooling (the language server) and is off during normal
// compilation. A reference remembers where it is, where its definition is,
// and a one-line description used for hover text.
type Index struct {
	Refs []Ref
}

type Ref struct {
	Span   source.Span // the occurrence
	Def    source.Span // the declaration; invalid for builtins
	Kind   string      // "val", "var", "fun", "struct", "trait", "sealed", "field", "module", "type"
	Name   string
	Type   types.Type // value/field type or function signature; nil for modules and traits
	Detail string     // rendered hover line
	Module *Module    // for Kind "module": the module referred to
	// Unfold, when set, is a named error set involved in the reference (the
	// set itself, or the one a function's `throws` names), spelled out with
	// its members so a hover can show and link each of them.
	Unfold *Unfold
	// Doc is the declaration's documentation comment, markdown.
	Doc string
	// Shape spells out a struct/error/sealed/trait type involved in the
	// reference — its fields, variants or methods — for the hover.
	Shape string
}

// Unfold is a named error set and its members, each with its declaration.
type Unfold struct {
	Name    string
	Def     source.Span
	Members []RefLink
}

// RefLink names a declaration a hover may link to.
type RefLink struct {
	Name  string
	Def   source.Span // invalid when there is nothing to jump to
	Shape string      // the member spelled out (see Ref.Shape), one line
}

// unfoldOf builds the hover unfolding of an error set symbol.
func (c *Checker) unfoldOf(sym *Symbol) *Unfold {
	u, ok := sym.Type.(*types.ErrorUnion)
	if !ok {
		return nil
	}
	out := &Unfold{Name: sym.Name, Def: sym.Span}
	for _, m := range u.Members {
		link := RefLink{Name: m.String()}
		if st, isStruct := m.(*types.Struct); isStruct {
			if d, ok := templateOf(st).Decl.(*ast.StructDecl); ok {
				link.Def = d.Name.Pos
			}
			link.Shape = c.compactStructShape(st)
		}
		out.Members = append(out.Members, link)
	}
	return out
}

// CheckIndex type-checks the package with reference recording on. Unlike
// Check it always returns the index, even when there are errors, so an
// editor keeps hover and go-to-definition while the user is mid-edit.
func CheckIndex(pkg *Package, diags *source.Diagnostics) *Index {
	idx := &Index{}
	checkWith(pkg, diags, false, false, idx)
	return idx
}

// RefAt returns the innermost reference covering a byte offset in a file.
func (ix *Index) RefAt(file *source.File, offset int) *Ref {
	var best *Ref
	for i := range ix.Refs {
		r := &ix.Refs[i]
		if r.Span.File != file || offset < r.Span.Start || offset > r.Span.End {
			continue
		}
		if best == nil || r.Span.End-r.Span.Start < best.Span.End-best.Span.Start {
			best = r
		}
	}
	return best
}

func (c *Checker) refSym(span source.Span, sym *Symbol) {
	if c.index == nil || sym == nil || !span.IsValid() {
		return
	}
	switch sym.Kind {
	case SymLocal:
		c.refVar(span, sym.Var)
	case SymGlobal:
		g := sym.Global
		kind := "val"
		if g.Mutable {
			kind = "var"
		}
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: sym.Span, Kind: kind, Name: sym.Name, Type: g.Type,
			Detail: kind + " " + sym.Name + typeSuffix(g.Type), Doc: c.globalDoc(g), Shape: c.shapeOf(g.Type)})
	case SymFunc:
		c.refFunc(span, sym.Func)
	case SymType:
		if sym.TypeAlias != nil {
			if t := c.symType(sym); t != nil {
				c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: sym.Span, Kind: "type", Name: sym.Name, Type: t, Detail: aliasDetail(sym, t), Doc: sym.TypeAlias.decl.Doc, Shape: c.shapeOf(t)})
			}
			return
		}
		c.refType(span, sym.Name, sym.Type, sym.Span)
	case SymModule:
		// the definition of a module is the top of its first file
		def := sym.Span
		if len(sym.Mod.Files) > 0 {
			def = source.Span{File: sym.Mod.Files[0].Source, Start: 0, End: 0}
		}
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: "module", Name: sym.Name, Detail: "module " + sym.Mod.Path, Module: sym.Mod, Doc: sym.Mod.Doc()})
	case SymVariantCtor:
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Kind: "fun", Name: sym.Name, Detail: "prelude " + sym.Name})
	}
}

func (c *Checker) refVar(span source.Span, v *Var) {
	c.refVarAs(span, v, nil)
}

// refVarAs records a use of v where a smart cast (D5) has narrowed it to
// `as`; the hover shows the narrowed type and where it came from.
func (c *Checker) refVarAs(span source.Span, v *Var, as types.Type) {
	if c.index == nil || v == nil || !span.IsValid() {
		return
	}
	for v.Outer != nil {
		v = v.Outer
	}
	kind := "val"
	if v.Mutable {
		kind = "var"
	}
	t := v.Type
	detail := kind + " " + v.Name + typeSuffix(t)
	if as != nil && !types.Identical(as, v.Type) {
		t = as
		detail = kind + " " + v.Name + typeSuffix(as) + "  (smart cast from " + v.Type.String() + ")"
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: v.Span, Kind: kind, Name: v.Name, Type: t, Detail: detail, Shape: c.shapeOf(t)})
}

// renarrowRef updates the ref just recorded at span (a field read) to show
// the smart-cast type instead of the declared one.
func (c *Checker) renarrowRef(span source.Span, as, declared types.Type) {
	if c.index == nil || types.Identical(as, declared) {
		return
	}
	for i := len(c.index.Refs) - 1; i >= 0; i-- {
		r := &c.index.Refs[i]
		if r.Span == span {
			r.Type = as
			r.Detail = strings.Replace(r.Detail, typeSuffix(declared), typeSuffix(as), 1) + "  (smart cast from " + declared.String() + ")"
			r.Shape = c.shapeOf(as)
			return
		}
	}
}

func (c *Checker) refFunc(span source.Span, t *FuncTemplate) {
	if c.index == nil || t == nil || !span.IsValid() {
		return
	}
	c.resolveSignature(t)
	def := span
	if t.Decl != nil {
		def = t.Decl.Name.Pos
	}
	ref := Ref{Span: span, Def: def, Kind: "fun", Name: t.Name, Type: t.Sig, Detail: funDetail(t)}
	if t.Decl != nil {
		ref.Doc = t.Decl.Doc
	}
	// `throws PortErrors`: show the set's name in the signature and unfold it
	if sym := c.errorSetOf(t); sym != nil {
		ref.Detail = strings.Replace(ref.Detail, " throws "+sym.Type.String(), " throws "+sym.Name, 1)
		ref.Unfold = c.unfoldOf(sym)
	}
	c.index.Refs = append(c.index.Refs, ref)
}

// errorSetOf returns the named error set a function's `throws` clause
// names directly, or nil.
func (c *Checker) errorSetOf(t *FuncTemplate) *Symbol {
	if t.Decl == nil || t.Decl.Effects.Error == nil || t.Sig == nil {
		return nil
	}
	nt, ok := t.Decl.Effects.Error.(*ast.NamedType)
	if !ok {
		return nil
	}
	env := &typeEnv{module: t.Module, file: t.File}
	sym, _ := c.lookupTypeName(env, nt.Path)
	if sym == nil || sym.Alias == nil || sym.Type == nil {
		return nil
	}
	return sym
}

func (c *Checker) refField(span source.Span, st *types.Struct, fld *types.Field) {
	if c.index == nil || fld == nil || !span.IsValid() {
		return
	}
	def := source.Span{}
	if d, ok := templateOf(st).Decl.(*ast.StructDecl); ok {
		for _, f := range d.Fields {
			if f.Name.Name == fld.Name {
				def = f.Name.Pos
			}
		}
	}
	doc := ""
	if d, ok := templateOf(st).Decl.(*ast.StructDecl); ok {
		for _, f := range d.Fields {
			if f.Name.Name == fld.Name {
				doc = f.Doc
			}
		}
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: "field", Name: fld.Name, Type: fld.Type,
		Detail: st.String() + "." + fld.Name + typeSuffix(fld.Type), Doc: doc, Shape: c.shapeOf(fld.Type)})
}

func (c *Checker) refType(span source.Span, name string, t types.Type, def source.Span) {
	if c.index == nil || t == nil || !span.IsValid() {
		return
	}
	kind, detail := "type", "type "+name
	switch tt := t.(type) {
	case *types.Struct:
		kind = "struct"
		if tt.Sealed != nil {
			detail = "struct " + tt.Name + " : " + tt.Sealed.Name
		} else {
			detail = "struct " + tt.String()
		}
		if len(tt.TypeParams) > 0 && len(tt.TypeArgs) == 0 {
			detail = "struct " + tt.Name + typeParamList(tt.TypeParams)
		}
		if d, ok := templateOf(tt).Decl.(*ast.StructDecl); ok && d.Error {
			detail = "error" + strings.TrimPrefix(detail, "struct")
		}
	case *types.ErrorUnion:
		kind = "error"
		detail = "error " + name + " = " + types.Unaliased(tt, false).String()
	case *types.Sealed:
		kind = "sealed"
		detail = "sealed trait " + tt.Name + typeParamList(tt.TypeParams)
	case *types.Trait:
		kind = "trait"
		detail = "trait " + tt.Name + typeParamList(tt.TypeParams)
	case *types.Basic:
		detail = "builtin type " + tt.Name
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: kind, Name: name, Type: t, Detail: detail,
		Doc: docOfType(t), Shape: c.shapeOf(t)})
}

// docOfType is the documentation comment on a type's declaration.
func docOfType(t types.Type) string {
	switch tt := t.(type) {
	case *types.Struct:
		if d, ok := templateOf(tt).Decl.(*ast.StructDecl); ok {
			return d.Doc
		}
	case *types.Sealed:
		if d, ok := sealedTemplate(tt).Decl.(*ast.TraitDecl); ok {
			return d.Doc
		}
	case *types.Trait:
		if d, ok := tt.Decl.(*ast.TraitDecl); ok {
			return d.Doc
		}
	}
	return ""
}

// shapeOf spells out what a value of type t has inside, as it would be
// declared: a struct's or error's fields and inherent methods, a sealed
// trait's variants, a trait's methods. Empty for every other type.
func (c *Checker) shapeOf(t types.Type) string {
	var sb strings.Builder
	switch tt := t.(type) {
	case *types.Pointer:
		return c.shapeOf(tt.Elem)
	case *types.Nullable:
		return c.shapeOf(tt.Elem)
	case *types.Struct:
		tmpl := templateOf(tt)
		d, _ := tmpl.Decl.(*ast.StructDecl)
		if d == nil {
			return ""
		}
		c.resolveStruct(tmpl)
		kw := "struct "
		if d.Error {
			kw = "error "
		}
		sb.WriteString(kw + tt.String())
		if tt.Sealed != nil {
			sb.WriteString(" : " + tt.Sealed.Name)
		}
		sb.WriteString(" {\n")
		for _, fld := range tt.Fields {
			sb.WriteString("  ")
			if fld.Pub {
				sb.WriteString("pub ")
			}
			sb.WriteString(fld.Name + ": " + fld.Type.String())
			if fld.HasDefault {
				sb.WriteString(" = ...")
			}
			sb.WriteString("\n")
		}
		for _, name := range sortedMethodNames(c.methods[tmpl]) {
			sb.WriteString("  " + strings.Replace(funDetail(c.methods[tmpl][name]), tmpl.Name+".", "", 1) + "\n")
		}
		if d.Error {
			sb.WriteString("  fun message(): string\n")
		}
		sb.WriteString("}")
	case *types.Sealed:
		sb.WriteString("sealed trait " + tt.String() + " {\n")
		for _, v := range tt.Variants {
			sb.WriteString("  " + v.Name)
			if len(v.Fields) > 0 {
				sb.WriteString(" { ")
				for i, fld := range v.Fields {
					if i > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(fld.Name + ": " + fld.Type.String())
				}
				sb.WriteString(" }")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("}")
	case *types.Trait:
		sb.WriteString("trait " + tt.Name + " {\n")
		for _, name := range tt.MethodList {
			sb.WriteString("  fun " + name + funSigString(tt.Methods[name]) + "\n")
		}
		sb.WriteString("}")
	default:
		return ""
	}
	return sb.String()
}

func sortedMethodNames(m map[string]*FuncTemplate) []string {
	var names []string
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// funSigString renders a resolved signature's parameter list and effects.
func funSigString(sig *types.Func) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i, p := range sig.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name + ": " + p.Type.String())
	}
	sb.WriteString(")")
	if sig.Ret != nil && !types.IsUnit(sig.Ret) {
		sb.WriteString(": " + sig.Ret.String())
	}
	if sig.Effects.Suspends {
		sb.WriteString(" suspends")
	}
	if sig.Effects.Throws {
		sb.WriteString(" throws")
		if sig.Effects.Error != nil && !types.IsNever(sig.Effects.Error) {
			sb.WriteString(" " + sig.Effects.Error.String())
		}
	}
	return sb.String()
}

func typeSuffix(t types.Type) string {
	if t == nil || types.IsInvalid(t) {
		return ""
	}
	return ": " + t.String()
}

func typeParamList(tps []*types.TypeParam) string {
	if len(tps) == 0 {
		return ""
	}
	names := make([]string, len(tps))
	for i, tp := range tps {
		names[i] = tp.Name
	}
	return "<" + strings.Join(names, ", ") + ">"
}

// funDetail renders a template's signature the way it was declared.
func funDetail(t *FuncTemplate) string {
	var sb strings.Builder
	if t.Decl != nil && t.Decl.Mut {
		sb.WriteString("mut ")
	}
	sb.WriteString("fun ")
	if t.Owner != nil {
		sb.WriteString(t.Owner.Name + ".")
	} else if t.Impl != nil && t.Impl.Trait != nil {
		sb.WriteString(t.Impl.Trait.Name + ".")
	} else if t.Impl != nil {
		sb.WriteString(t.Impl.Target.String() + ".")
	} else if t.Trait != nil {
		sb.WriteString(t.Trait.Name + ".")
	}
	sb.WriteString(t.Name)
	sb.WriteString(typeParamList(t.TypeParams))
	sb.WriteString("(")
	if t.Sig != nil {
		for i, p := range t.Sig.Params {
			if i > 0 {
				sb.WriteString(", ")
			}
			if p.Name != "" {
				sb.WriteString(p.Name + ": ")
			}
			if lt, ok := p.Type.(*types.List); ok && p.Variadic {
				sb.WriteString(lt.Elem.String() + "...")
			} else {
				sb.WriteString(p.Type.String())
			}
			if p.HasDefault {
				sb.WriteString(" = ...")
			}
		}
	}
	sb.WriteString(")")
	if t.Sig != nil {
		if t.Sig.Ret != nil && !types.IsUnit(t.Sig.Ret) {
			sb.WriteString(": " + t.Sig.Ret.String())
		}
		if t.Sig.Effects.Suspends {
			sb.WriteString(" suspends")
		}
		if t.Sig.Effects.Throws {
			sb.WriteString(" throws")
			errT := t.Sig.Effects.Error
			if errT == nil {
				// inferred (D45): stable by the final round, when refs are recorded
				errT = t.InferError
			}
			if errT != nil && !types.IsNever(errT) {
				sb.WriteString(" " + errT.String())
			}
		}
	}
	return sb.String()
}

// variantDefSpan finds the declaration of a sealed variant reached through
// `Parent.Variant`, which has no module-scope symbol of its own.
func variantDefSpan(v *types.Struct) source.Span {
	if d, ok := templateOf(v).Decl.(*ast.StructDecl); ok {
		return d.Name.Pos
	}
	return source.Span{}
}

// suggestUnknown returns a hint for an unresolved name: a public
// declaration of the same name in a module of the package or the standard
// library ("did you mean 'io.println'? add 'use io'"), or nothing.
func (c *Checker) suggestUnknown(m *Module, name string) string {
	seen := map[string]bool{}
	var hits []string
	consider := func(mod *Module) {
		if mod == nil || seen[mod.Path] || mod.Path == "std/prelude" {
			return
		}
		seen[mod.Path] = true
		for _, f := range mod.Files {
			for _, d := range f.Decls {
				if declName(d) == name && declIsPub(d) {
					hits = append(hits, mod.Name()+"."+name)
					return
				}
			}
		}
	}
	// modules already loaded (imported somewhere in the package)
	for _, mod := range c.pkg.Modules {
		consider(mod)
	}
	// standard modules that nothing has imported yet
	for _, std := range c.pkg.stdModuleNames() {
		if _, loaded := c.pkg.Modules["std/"+std]; !loaded {
			if mod, ok := c.pkg.loadStd(std); ok {
				consider(mod)
			}
		}
	}
	if len(hits) == 0 {
		return ""
	}
	modName := hits[0][:strings.Index(hits[0], ".")]
	imported := false
	for _, dep := range m.Deps {
		if dep.Name() == modName {
			imported = true
		}
	}
	if imported {
		return fmt.Sprintf("; did you mean '%s'?", hits[0])
	}
	return fmt.Sprintf("; did you mean '%s'? (add 'use %s' at the top of the file)", hits[0], modName)
}

func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.FunDecl:
		return d.Name.Name
	case *ast.StructDecl:
		return d.Name.Name
	case *ast.TraitDecl:
		return d.Name.Name
	case *ast.ValDecl:
		return d.Name.Name
	}
	return ""
}

func declIsPub(d ast.Decl) bool {
	switch d := d.(type) {
	case *ast.FunDecl:
		return d.Pub
	case *ast.StructDecl:
		return d.Pub
	case *ast.TraitDecl:
		return d.Pub
	case *ast.ValDecl:
		return d.Pub
	}
	return false
}

// suggestUnknownName is suggestUnknown for a bare name, which may also be
// an un-imported module used as a prefix (`io.println` without `use io`).
func (c *Checker) suggestUnknownName(m *Module, name string) string {
	for _, std := range c.pkg.stdModuleNames() {
		if std == name {
			return fmt.Sprintf("; '%s' is a standard module: add 'use %s' at the top of the file", name, name)
		}
	}
	return c.suggestUnknown(m, name)
}

// globalDoc is the documentation comment on a module-level binding.
func (c *Checker) globalDoc(g *Global) string {
	if d := c.globals[g]; d != nil {
		return d.Doc
	}
	return ""
}

// compactStructShape is a one-line `error Name { a: T, b: U }` for lists.
func (c *Checker) compactStructShape(st *types.Struct) string {
	tmpl := templateOf(st)
	c.resolveStruct(tmpl)
	kw := "struct "
	if d, ok := tmpl.Decl.(*ast.StructDecl); ok && d.Error {
		kw = "error "
	}
	if len(st.Fields) == 0 {
		return kw + st.String()
	}
	parts := make([]string, len(st.Fields))
	for i, fld := range st.Fields {
		parts[i] = fld.Name + ": " + fld.Type.String()
	}
	return kw + st.String() + " { " + strings.Join(parts, ", ") + " }"
}
