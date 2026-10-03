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
	// Wire maps the name of a declaration a derived coding implement writes
	// out — a field, an enum member, a sealed variant — to what writes it
	// (`Codable for Note`), so a rename can refuse to change a wire format.
	Wire map[source.Span]string
	// Inferred maps a function's name (its declaration) to what the
	// compiler inferred and the source leaves unwritten, for inlay hints.
	Inferred map[source.Span]Inferred
	// Impls maps a trait's name (its declaration) to the implements of it
	// the author wrote — a sealed trait's also to its variants — and a
	// trait method's name to each method implementing it, for
	// go-to-implementation.
	Impls map[source.Span][]source.Span
	// Held marks the value of each `with` that receives a value holding
	// tasks (D111): its brace stops those tasks before it closes the value.
	Held map[source.Span]bool
}

// TypeDecl is where the type of a value is declared, for
// go-to-type-definition: through a nullable, a pointer or a task to what
// it holds, and from a built-in container (`List<Tag>`, `Map<K, Tag>`) to
// the first element type that has a declaration. Invalid when nothing in
// it was declared in source (`i64`, `List<string>`).
func TypeDecl(t types.Type) source.Span {
	name := func(d any) source.Span {
		switch d := d.(type) {
		case *ast.StructDecl:
			return d.Name.Pos
		case *ast.TraitDecl:
			return d.Name.Pos
		case *ast.EnumDecl:
			return d.Name.Pos
		}
		return source.Span{}
	}
	var args []types.Type
	switch t := t.(type) {
	case *types.Struct:
		if sp := name(t.Decl); sp.IsValid() {
			return sp
		}
		args = t.TypeArgs
	case *types.Sealed:
		if sp := name(t.Decl); sp.IsValid() {
			return sp
		}
		args = t.TypeArgs
	case *types.Trait:
		return name(t.Decl)
	case *types.Enum:
		return name(t.Decl)
	case *types.Nullable:
		return TypeDecl(t.Elem)
	case *types.Pointer:
		return TypeDecl(t.Elem)
	case *types.List:
		return TypeDecl(t.Elem)
	case *types.Set:
		return TypeDecl(t.Elem)
	case *types.Channel:
		return TypeDecl(t.Elem)
	case *types.Task:
		return TypeDecl(t.Result)
	case *types.Map:
		args = []types.Type{t.Key, t.Value}
	case *types.Tuple:
		args = t.Elems
	}
	for _, a := range args {
		if sp := TypeDecl(a); sp.IsValid() {
			return sp
		}
	}
	return source.Span{}
}

// indexImpls fills Index.Impls from the implements collection found. It
// runs whether or not the package checked, so it works mid-edit.
func (c *Checker) indexImpls() {
	if c.index == nil {
		return
	}
	c.index.Impls = map[source.Span][]source.Span{}
	add := func(key, at source.Span) {
		if !key.IsValid() || !at.IsValid() {
			return
		}
		for _, s := range c.index.Impls[key] {
			if s == at {
				return
			}
		}
		c.index.Impls[key] = append(c.index.Impls[key], at)
	}
	methods := func(td *ast.TraitDecl, fns []*ast.FunDecl) {
		for _, m := range td.Methods {
			for _, fn := range fns {
				if fn.Name.Name == m.Name.Name {
					add(m.Name.Pos, fn.Name.Pos)
				}
			}
		}
	}
	for trait, list := range c.impls {
		td, ok := trait.Decl.(*ast.TraitDecl)
		if !ok {
			continue
		}
		for _, impl := range list {
			// a compiler-made implement (a supertrait's, a variant's) is
			// not where anything was written
			if impl.Decl == nil || impl.Decl.Derived {
				continue
			}
			add(td.Name.Pos, impl.Decl.Pos)
			methods(td, impl.Decl.Methods)
		}
	}
	for s := range c.sealedDecl {
		td, ok := s.Decl.(*ast.TraitDecl)
		if !ok {
			continue
		}
		for _, v := range s.Variants {
			if vd, ok := v.Decl.(*ast.StructDecl); ok {
				add(td.Name.Pos, vd.Name.Pos)
				methods(td, vd.Methods)
			}
		}
	}
	for _, spans := range c.index.Impls {
		sort.Slice(spans, func(i, j int) bool {
			a, b := spans[i], spans[j]
			if a.File.Path != b.File.Path {
				return a.File.Path < b.File.Path
			}
			return a.Start < b.Start
		})
	}
}

// Inferred is the unwritten part of a function's signature.
type Inferred struct {
	Ret      string // the return type of `fun f() = expr`; "" when written
	Suspends bool   // suspends, and `suspends` is not written (D2)
	// SuspendsIf: it suspends only when what these parameters are given
	// does (D116)
	SuspendsIf []string
	Throws     string // the error set of a bare `throws` (D45)
}

// indexInferred fills Index.Inferred once effects are known. A generic
// function suspends when any instance does.
func (c *Checker) indexInferred(prog *Program) {
	if c.index == nil {
		return
	}
	c.index.Inferred = map[source.Span]Inferred{}
	for _, fn := range prog.Funcs {
		t := fn.tmpl
		if fn.IsClosure || t == nil || t.Decl == nil || t.Sig == nil || !t.Decl.Name.Pos.IsValid() {
			continue
		}
		d := t.Decl
		inf := c.index.Inferred[d.Name.Pos]
		if d.Ret == nil && d.ExprBody != nil && fn.Sig.Ret != nil && !types.IsUnit(fn.Sig.Ret) && !types.IsInvalid(fn.Sig.Ret) && monomorphic(t) {
			inf.Ret = fn.Sig.Ret.String()
		}
		if fn.Suspends && !d.Effects.Suspends {
			if fn.Conditional {
				for _, p := range fn.Params {
					if isSuspendParam(p) && !containsString(inf.SuspendsIf, p.Name) {
						inf.SuspendsIf = append(inf.SuspendsIf, p.Name)
					}
				}
			} else {
				inf.Suspends = true // any instance that always suspends decides
			}
		}
		if d.Effects.Throws && d.Effects.Error == nil && fn.Sig.Effects.Error != nil && monomorphic(t) {
			inf.Throws = fn.Sig.Effects.Error.String()
		}
		if inf.Ret != "" || inf.Suspends || inf.Throws != "" || len(inf.SuspendsIf) > 0 {
			c.index.Inferred[d.Name.Pos] = inf
		}
	}
	c.showInferredSuspends()
}

// showInferredSuspends makes every hover of a function that suspends
// without saying so (D2) spell `suspends`, at its declaration and at each
// call: hovers were rendered from the written signature, before the
// suspension pass knew.
func (c *Checker) showInferredSuspends() {
	for i := range c.index.Refs {
		r := &c.index.Refs[i]
		if r.Kind != "fun" || !r.Def.IsValid() {
			continue
		}
		inf := c.index.Inferred[r.Def]
		line, rest, _ := strings.Cut(r.Detail, "\n")
		params := sigParamsEnd(line)
		switch {
		case inf.Suspends && !strings.Contains(line[params:], " suspends"):
			// after the parameters: a parameter's own type may say `suspends`
			if at := strings.Index(line[params:], " throws"); at >= 0 {
				line = line[:params+at] + " suspends" + line[params+at:]
			} else {
				line += " suspends"
			}
		case len(inf.SuspendsIf) > 0:
			// D116: the call suspends when what it is given does
			line += "\n// suspends if `" + strings.Join(inf.SuspendsIf, "` or `") + "` does"
		default:
			continue
		}
		if rest != "" {
			line += "\n" + rest
		}
		r.Detail = line
	}
}

// sigParamsEnd is where a rendered signature's parameter list ends: after
// the parenthesis that closes the first one opened, so a parameter's
// function type is not read as the function's effects.
func sigParamsEnd(line string) int {
	depth := 0
	for i, r := range line {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return 0
}

type Ref struct {
	Span   source.Span // the occurrence
	Def    source.Span // the declaration; invalid for builtins
	Kind   string      // "val", "var", "fun", "struct", "trait", "sealed", "enum", "field", "module", "type"
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
	// Derived, on the trait name of an implement whose body the compiler
	// wrote (D58), says what it wrote: the header with any inferred bounds,
	// the synthesized signatures, and the wire shape of a Codable derive.
	Derived string
	// Where names the declaration a member belongs to (`struct Notes` for a
	// field), shown above Detail in the hover.
	Where string
	// Family, on a method that implements a trait's method, is the trait
	// method's declaration: the trait's method and every implementation of
	// it are one name to find-references and rename.
	Family source.Span
	// Pun marks a field reference written as a D28 pun (`Hashed(file)` for
	// `file: file`): the span is also a use of the variable, so renaming one
	// of the two spells the pair out instead of renaming both.
	Pun bool
}

// Key is the declaration a reference groups under for find-references and
// rename: the trait method for an implementation of one, else Def.
func (r *Ref) Key() source.Span {
	if r.Family.IsValid() {
		return r.Family
	}
	return r.Def
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
		// the innermost span; on a tie the one recorded last, which is the
		// better informed (a body check refines what collection recorded)
		if best == nil || r.Span.End-r.Span.Start <= best.Span.End-best.Span.Start {
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
		c.refGlobal(span, sym.Name, sym.Global)
	case SymFunc:
		c.refFunc(span, sym.Func)
	case SymType:
		if sym.TypeAlias != nil {
			if t := c.symType(sym); t != nil {
				c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: sym.Span, Kind: "type", Name: sym.Name, Type: t, Detail: aliasDetail(sym, t), Doc: sym.TypeAlias.decl.Doc, Shape: c.shapeFrom(t, c.viewFrom(span))})
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
		head, _ := moduleHead(sym.Mod)
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: "module", Name: sym.Name, Detail: head, Module: sym.Mod, Doc: sym.Mod.Doc(), Shape: c.moduleShape(sym.Mod)})
	case SymVariantCtor:
		v := variantCtorDocs[sym.Name]
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Kind: "fun", Name: sym.Name, Detail: v[0], Where: v[1], Doc: v[2]})
	}
}

// refGlobal records a use of a module-level or static value, rendered as
// its declaration with every implicit word spelled out:
// `val limit: i64 = 10`, `public static val ok: Status = ...`.
func (c *Checker) refGlobal(span source.Span, name string, g *Global) {
	if c.index == nil || g == nil || !span.IsValid() {
		return
	}
	ref := Ref{Span: span, Def: g.Span, Kind: "val", Name: name, Type: g.Type, Doc: c.globalDoc(g), Shape: c.shapeFrom(g.Type, c.viewFrom(span))}
	if g.Mutable {
		ref.Kind = "var"
	}
	ref.Detail = c.globalDecl(name, g)
	if owner := c.staticOwner[g]; owner != nil {
		ref.Where = structHead(owner)
	}
	c.index.Refs = append(c.index.Refs, ref)
}

// globalDecl renders a global or static value as declared.
func (c *Checker) globalDecl(name string, g *Global) string {
	d := c.globals[g]
	if d == nil {
		kind := "val"
		if g.Mutable {
			kind = "var"
		}
		return kind + " " + name + typeSuffix(g.Type)
	}
	var sb strings.Builder
	sb.WriteString(visPrefix(d.Pub, false))
	if c.staticOwner[g] != nil {
		sb.WriteString("static ")
	}
	sb.WriteString(d.Kind.String() + " " + name + typeSuffix(g.Type))
	if d.Value != nil {
		if t := srcText(d.Value); t != "" && len(t) <= 60 && !strings.Contains(t, "\n") {
			sb.WriteString(" = " + t)
		} else {
			sb.WriteString(" = ...")
		}
	}
	if g.Const != nil && d.Value != nil {
		// what the compiler computed (D113), when it is not what is written
		if v := constString(g.Const); len(v) <= 60 && v != srcText(d.Value) {
			sb.WriteString("  // " + v)
		}
	}
	return sb.String()
}

// moduleHead names a module the way `use` spells it (`module io`,
// `module mathlib.geometry`) and says where it comes from: "std",
// "package mathlib" or "this package".
func moduleHead(m *Module) (head, from string) {
	path := m.Path
	from = "this package"
	switch {
	case strings.HasPrefix(path, "std/"):
		path = strings.TrimPrefix(path, "std/")
		from = "std"
	case strings.HasPrefix(path, "dep/"):
		path = strings.TrimPrefix(path, "dep/")
		if i := strings.Index(path, "/"); i >= 0 {
			from = "package " + path[:i]
		} else {
			from = "package " + path
		}
	case path == "":
		path = "main"
	}
	return "module " + strings.ReplaceAll(path, "/", "."), from
}

// moduleShape lists a module's public surface — what `use` brings in — as
// it would be declared: types, functions and values in source order.
func (c *Checker) moduleShape(m *Module) string {
	if m == nil || m.Scope == nil {
		return ""
	}
	var syms []*Symbol
	for _, sym := range m.Scope.symbols {
		if sym.Pub && sym.Kind != SymModule {
			syms = append(syms, sym)
		}
	}
	sort.Slice(syms, func(i, j int) bool {
		a, b := syms[i].Span, syms[j].Span
		if a.File != b.File {
			pa, pb := "", ""
			if a.File != nil {
				pa = a.File.Path
			}
			if b.File != nil {
				pb = b.File.Path
			}
			return pa < pb
		}
		return a.Start < b.Start
	})
	var sb strings.Builder
	head, from := moduleHead(m)
	sb.WriteString(head + " {  // " + from + "\n")
	const limit = 40
	for i, sym := range syms {
		if i == limit {
			sb.WriteString(fmt.Sprintf("  // ... and %d more\n", len(syms)-limit))
			break
		}
		line := ""
		switch sym.Kind {
		case SymType:
			switch {
			case sym.TypeAlias != nil:
				if t := c.symType(sym); t != nil {
					line = "public " + aliasDetail(sym, t)
				}
			case sym.Alias != nil:
				c.resolveAlias(sym)
				if u, ok := sym.Type.(*types.ErrorUnion); ok {
					line = "public error " + sym.Name + " = " + types.Unaliased(u, false).String()
				}
			default:
				switch t := sym.Type.(type) {
				case *types.Struct:
					line = structHead(t)
				case *types.Trait:
					line = traitHead(t)
				case *types.Sealed:
					line = sealedHead(t)
				}
			}
		case SymFunc:
			c.resolveSignature(sym.Func)
			line = funDecl(sym.Func)
		case SymGlobal:
			line = c.globalDecl(sym.Name, sym.Global)
		}
		if line != "" {
			sb.WriteString("  " + line + "\n")
		}
	}
	sb.WriteString("}")
	return sb.String()
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
	} else if v.InitText != "" && len(v.InitText) <= 60 && !strings.Contains(v.InitText, "\n") {
		detail += " = " + v.InitText
	} else if v.InitText != "" {
		detail += " = ..."
	}
	if v.IsParam {
		detail += "  (parameter)"
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: v.Span, Kind: kind, Name: v.Name, Type: t, Detail: detail, Shape: c.shapeFrom(t, c.viewFrom(span))})
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
			r.Shape = c.shapeFrom(as, c.viewFrom(span))
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
	// the function as declared with nothing left implicit, under the
	// declaration it belongs to: `struct Notes`,
	// `impl Display for Point`, `extend Point`, `public trait Shape`
	ref := Ref{Span: span, Def: def, Kind: "fun", Name: t.Name, Type: t.Sig, Detail: funDecl(t), Where: funWhere(t)}
	if t.Decl != nil {
		ref.Doc = t.Decl.Doc
	}
	if t.Sig != nil {
		if hv := c.heldValue(t.Sig.Ret); hv != nil {
			ref.Doc = c.withHeldNote(ref.Doc, hv)
		}
	}
	if t.Impl != nil && t.Impl.Trait != nil {
		if d, ok := t.Impl.Trait.Decl.(*ast.TraitDecl); ok {
			for _, m := range d.Methods {
				if m.Name.Name == t.Name {
					ref.Family = m.Name.Pos
					if ref.Doc == "" {
						// an undocumented impl method inherits the trait's description of it
						ref.Doc = m.Doc
					}
				}
			}
		}
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

// refTraitMethod records a use of a trait's method that is not a call of
// one implementation: its declaration in the trait, or a call through a
// trait object. Declared in a supertrait, it is found there.
func (c *Checker) refTraitMethod(span source.Span, tr *types.Trait, name string) {
	if c.index == nil || tr == nil || !span.IsValid() {
		return
	}
	d, _ := tr.Decl.(*ast.TraitDecl)
	if d == nil {
		return
	}
	for _, m := range d.Methods {
		if m.Name.Name != name {
			continue
		}
		detail := "fun " + name
		if sig := tr.Methods[name]; sig != nil {
			detail += funSigString(sig)
		}
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: m.Name.Pos, Kind: "fun", Name: name, Type: tr.Methods[name],
			Detail: detail, Where: traitHead(tr), Doc: m.Doc})
		return
	}
	for _, sup := range tr.Supers {
		if _, ok := sup.Methods[name]; ok {
			c.refTraitMethod(span, sup, name)
			return
		}
	}
}

// refArgLabels records the names of a call's named arguments as uses of
// the parameters (or, for a constructor, the fields) they bind, so
// find-references and rename reach `send(to: x)` and `Point(x: 1)`.
func (c *Checker) refArgLabels(args []ast.Arg, t *FuncTemplate) {
	if c.index == nil || t == nil || t.Decl == nil {
		return
	}
	for _, a := range args {
		if a.Name == nil {
			continue
		}
		for j, p := range t.Decl.Params {
			if p.Name.Name == a.Name.Name && t.Sig != nil && j < len(t.Sig.Params) {
				pt := t.Sig.Params[j].Type
				c.index.Refs = append(c.index.Refs, Ref{Span: a.Name.Pos, Def: p.Name.Pos, Kind: "val", Name: p.Name.Name, Type: pt,
					Detail: "val " + p.Name.Name + typeSuffix(pt) + "  (parameter of " + t.Name + ")"})
			}
		}
	}
}

// refFieldLabels is refArgLabels for a constructor call; `punned` are the
// arguments written as a bare variable named like the field.
func (c *Checker) refFieldLabels(st *types.Struct, args []ast.Arg, punned []bool) {
	if c.index == nil {
		return
	}
	for i, a := range args {
		if a.Name == nil {
			continue
		}
		for _, fld := range st.Fields {
			if fld.Name == a.Name.Name {
				n := len(c.index.Refs)
				c.refField(a.Name.Pos, st, fld)
				if len(c.index.Refs) > n {
					c.index.Refs[n].Pun = punned[i]
				}
			}
		}
	}
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
	// the field as declared, every implicit word spelled out (`val`), so
	// the hover says who sees it and who may assign it
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: "field", Name: fld.Name, Type: fld.Type,
		Detail: fieldDecl(st, fld), Where: structHead(st), Doc: doc, Shape: c.shapeFrom(fld.Type, c.viewFrom(span))})
}

// visibilityWord spells the M5 level of a member or declaration. The
// unwritten level (module-internal) has no word: hover shows what was
// written, and `internal` on every line only made it harder to read.
func visibilityWord(pub, private bool) string {
	switch {
	case private:
		return "private"
	case pub:
		return "public"
	}
	return ""
}

// visPrefix is visibilityWord ready to prepend: the word and a space, or
// nothing at all.
func visPrefix(pub, private bool) string {
	if w := visibilityWord(pub, private); w != "" {
		return w + " "
	}
	return ""
}

// fieldDecl renders a field the way its declaration reads with nothing
// left implicit: `public protected var count: i64 = ...`. Types, not
// values: a default is shown as `= ...` (it exists, and the constructor
// call may omit the field); a field the `init` block assigns says so.
func fieldDecl(st *types.Struct, fld *types.Field) string {
	var sb strings.Builder
	sb.WriteString(visPrefix(fld.Pub, fld.Private))
	switch {
	case fld.Protected:
		sb.WriteString("protected var ")
	case fld.Var:
		sb.WriteString("var ")
	default:
		sb.WriteString("val ")
	}
	sb.WriteString(fld.Name + ": " + fld.Type.String())
	switch {
	case fld.HasDefault:
		sb.WriteString(" = ...")
	case fld.Init:
		sb.WriteString("  // assigned by init")
	}
	return sb.String()
}

// constructorLine spells how st is built from outside the type — the
// implicit constructor's parameters (D28): the fields the call must give,
// then, as `= ...`, the ones it may give. Private fields with a default
// and fields assigned by `init` are not the caller's; they are left out.
func constructorLine(st *types.Struct, v viewpoint) string {
	var parts []string
	for _, fld := range st.Fields {
		switch {
		case fld.Init, fld.HasDefault && !v.sees(st, fld.Pub, fld.Private):
			// not the caller's to give from here
			continue
		case fld.HasDefault:
			parts = append(parts, fld.Name+": "+fld.Type.String()+" = ...")
		default:
			parts = append(parts, fld.Name+": "+fld.Type.String())
		}
	}
	parts = append(parts, InitParamLabels(st, "...")...)
	return st.Name + "(" + strings.Join(parts, ", ") + ")"
}

// InitParamLabels spells the parameters `init(...)` adds to st's
// constructor (D73) as written, a default shown as `= ` + dflt.
func InitParamLabels(st *types.Struct, dflt string) []string {
	d, _ := templateOf(st).Decl.(*ast.StructDecl)
	if d == nil {
		return nil
	}
	var out []string
	for _, p := range d.InitParams {
		label := p.Name.Name
		if p.Type != nil {
			label += ": " + ast.TypeString(p.Type)
		}
		if p.Default != nil {
			label += " = " + dflt
		}
		out = append(out, label)
	}
	return out
}

// structHead is a struct's declaration line with its visibility spelled
// out: `struct Notes`, `public error NotFound`, `struct Circle : Shape`.
func structHead(st *types.Struct) string {
	tmpl := templateOf(st)
	d, _ := tmpl.Decl.(*ast.StructDecl)
	head := "struct "
	pub := st.Pub
	if d != nil {
		if d.Error {
			head = "error "
		}
		pub = d.Pub
	}
	head = visPrefix(pub, false) + head
	if len(st.TypeParams) > 0 && len(st.TypeArgs) == 0 {
		head += st.Name + typeParamList(st.TypeParams)
	} else {
		head += st.String()
	}
	if st.Sealed != nil {
		head += " : " + st.Sealed.Name
	}
	return head
}

// methodDecl renders an inherent method for a struct's shape: visibility,
// `static`, then the signature without the owner prefix.
func methodDecl(tmpl *types.Struct, t *FuncTemplate) string {
	return funDecl(t)
}

// funDecl renders a function or method the way its declaration reads with
// every implicit word spelled out: `public static fun of(n: i64): Notes`,
// `override fun toString(): string`, `fun describe(): string = ...` for a
// trait's default body. The owner is not repeated (see funWhere).
func funDecl(t *FuncTemplate) string {
	if t.Name == "$init" {
		return "init" + initParamList(t.Decl.Params) + " { ... }  // runs after every construction; not callable"
	}
	var sb strings.Builder
	d := t.Decl
	implMethod := t.Impl != nil && t.Impl.Trait != nil
	traitMethod := t.Trait != nil
	if d != nil && !implMethod && !traitMethod {
		// an impl's or a trait's method has the trait's visibility
		sb.WriteString(visPrefix(d.Pub, d.Private))
	}
	if d != nil {
		if d.Override {
			sb.WriteString("override ")
		}
		if d.Extern {
			sb.WriteString("extern ")
		}
		if d.Unsafe {
			sb.WriteString("unsafe ")
		}
		if d.Static {
			sb.WriteString("static ")
		}
	}
	sig := funDetail(t)
	for _, prefix := range ownerPrefixes(t) {
		sig = strings.Replace(sig, "fun "+prefix+".", "fun ", 1)
	}
	sb.WriteString(sig)
	if traitMethod && d != nil && (d.Body != nil || d.ExprBody != nil) {
		sb.WriteString(" = ...")
	}
	return sb.String()
}

// ownerPrefixes lists the `Owner.` spellings funDetail may have used.
func ownerPrefixes(t *FuncTemplate) []string {
	var out []string
	if t.Owner != nil {
		out = append(out, t.Owner.Name)
	}
	if t.Impl != nil && t.Impl.Trait != nil {
		out = append(out, t.Impl.Trait.Name)
	} else if t.Impl != nil {
		out = append(out, t.Impl.Target.String())
	}
	if t.Trait != nil {
		out = append(out, t.Trait.Name)
	}
	return out
}

// funWhere names the declaration a method belongs to, spelled out:
// `struct Notes`, `implement Display for Point`, `extend<T> Box<T>`,
// `public trait Shape`. Empty for a free function.
func funWhere(t *FuncTemplate) string {
	switch {
	case t.Owner != nil:
		return structHead(t.Owner)
	case t.Impl != nil && t.Impl.Trait != nil:
		return "implement " + t.Impl.Trait.Name + " for " + t.Impl.Target.String()
	case t.Impl != nil:
		return "extend " + t.Impl.Target.String()
	case t.Trait != nil:
		return traitHead(t.Trait)
	}
	return ""
}

// traitHead is a trait's declaration line with its visibility spelled
// out and its supertraits: `public trait Shape : Display`.
func traitHead(tr *types.Trait) string {
	head := visPrefix(tr.Pub, false)
	d, _ := tr.Decl.(*ast.TraitDecl)
	if d != nil && d.Sealed {
		head += "sealed "
	}
	head += "trait " + tr.Name + typeParamList(tr.TypeParams)
	if d != nil && len(d.Supers) > 0 {
		var supers []string
		for _, s := range d.Supers {
			supers = append(supers, ast.TypeString(s))
		}
		head += " : " + strings.Join(supers, " + ")
	}
	return head
}

// typeArgList spells the trait's type arguments as an impl wrote them
// (`impl From<string>`), or nothing for a plain trait.
func typeArgList(impl *Impl, trait *types.Trait) string {
	if impl.Decl == nil || impl.Decl.Trait == nil || len(trait.TypeParams) == 0 {
		return ""
	}
	if s := ast.TypeString(impl.Decl.Trait); strings.HasPrefix(s, trait.Name) {
		return strings.TrimPrefix(s, trait.Name)
	}
	return ""
}

// sealedHead is a sealed trait's declaration line with its visibility
// spelled out: `public sealed trait Shape`.
func sealedHead(s *types.Sealed) string {
	tmpl := sealedTemplate(s)
	name := s.String()
	if len(tmpl.TypeParams) > 0 && len(s.TypeArgs) == 0 {
		name = s.Name + typeParamList(tmpl.TypeParams)
	}
	return visPrefix(tmpl.Pub, false) + "sealed trait " + name
}

// traitBody lists a trait's associated types and methods as declared:
// `type Item`, `fun next(): Item?`, `fun map<U>(...) = ...` for a default.
func (c *Checker) traitBody(tr *types.Trait) string {
	var sb strings.Builder
	d, _ := tr.Decl.(*ast.TraitDecl)
	if d != nil {
		for _, at := range d.AssocTypes {
			sb.WriteString("  type " + at.Name.Name)
			if len(at.Bounds) > 0 {
				var bounds []string
				for _, b := range at.Bounds {
					bounds = append(bounds, ast.TypeString(b))
				}
				sb.WriteString(": " + strings.Join(bounds, " + "))
			}
			sb.WriteString("\n")
		}
	}
	if tr.ImplicitError {
		sb.WriteString("  type Error  // each impl's `throws`\n")
	}
	for _, name := range tr.MethodList {
		sb.WriteString("  ")
		if c.traitStatic[tr.Name+"."+name] {
			sb.WriteString("static ")
		}
		sb.WriteString("fun " + name + typeParamList(c.traitMethodTPs[tr.Name+"."+name]) + funSigString(tr.Methods[name]))
		if c.traitDefault(tr, name) != nil {
			sb.WriteString(" = ...")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (c *Checker) refType(span source.Span, name string, t types.Type, def source.Span) {
	if c.index == nil || t == nil || !span.IsValid() {
		return
	}
	kind, detail := "type", "type "+name
	switch tt := t.(type) {
	case *types.Struct:
		kind = "struct"
		detail = structHead(tt)
	case *types.ErrorUnion:
		kind = "error"
		detail = "error " + name + " = " + types.Unaliased(tt, false).String()
	case *types.Sealed:
		kind = "sealed"
		detail = sealedHead(tt)
	case *types.Trait:
		kind = "trait"
		detail = traitHead(tt)
	case *types.Enum:
		kind = "enum"
		detail = enumHead(tt)
	case *types.Basic:
		detail = "builtin type " + tt.Name
		c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: kind, Name: name, Type: t, Detail: detail,
			Doc: basicTypeDocs[tt.Kind], Shape: c.basicShape(tt)})
		return
	}
	c.index.Refs = append(c.index.Refs, Ref{Span: span, Def: def, Kind: kind, Name: name, Type: t, Detail: detail,
		Doc: c.withHeldNote(docOfType(t), t), Shape: c.shapeFrom(t, c.viewFrom(span))})
}

// heldNote is what a hover adds for a type that holds a task (D111).
const heldNote = "Holds a running task: receive it with `with` where it is made, or return it — its tasks belong to the receiving block (D111)."

// withHeldNote appends heldNote to doc when values of t hold a task.
func (c *Checker) withHeldNote(doc string, t types.Type) string {
	if t == nil || !c.taskHolding(t) {
		return doc
	}
	if doc == "" {
		return heldNote
	}
	return doc + "\n\n" + heldNote
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
	case *types.Enum:
		if d, ok := tt.Decl.(*ast.EnumDecl); ok {
			return d.Doc
		}
	}
	return ""
}

// viewpoint is where a hover is read from: the module of the reference
// and, for a struct, whether the reference sits inside the struct's own
// declarations. It decides which members the shape lists (M5): private
// ones only inside the type, unmarked ones only inside the module, public
// ones anywhere — what the reader could actually name from there.
type viewpoint struct {
	c    *Checker
	file *source.File
	off  int
	mod  *Module
}

func (c *Checker) viewFrom(span source.Span) viewpoint {
	v := viewpoint{c: c, file: span.File, off: span.Start}
	if span.File != nil && c.pkg != nil {
		for _, m := range c.pkg.Modules {
			for _, f := range m.Files {
				if f.Source == span.File {
					v.mod = m
				}
			}
		}
	}
	return v
}

// insideType reports whether the viewpoint is within st's body, or an
// impl or extend block for st in its module.
func (v viewpoint) insideType(st *types.Struct) bool {
	if v.mod == nil || v.file == nil {
		return false
	}
	name := templateOf(st).Name
	within := func(sp source.Span) bool {
		return sp.File == v.file && v.off >= sp.Start && v.off <= sp.End
	}
	for _, f := range v.mod.Files {
		if f.Source != v.file {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.StructDecl:
				if d.Name.Name == name && within(d.Pos) {
					return true
				}
			case *ast.ImplDecl:
				if t, ok := d.Target.(*ast.NamedType); ok && t.Path[len(t.Path)-1].Name == name && within(d.Pos) {
					return true
				}
			}
		}
	}
	return false
}

// sees reports whether a member with the given visibility of a type
// declared in module modPrefix can be named from the viewpoint.
func (v viewpoint) sees(st *types.Struct, pub, private bool) bool {
	switch {
	case private:
		return v.insideType(st)
	case pub:
		return true
	}
	return v.mod != nil && v.mod.prefix() == templateOf(st).Module
}

// shapeOf spells out what a value of type t has inside, as it would be
// declared: a struct's or error's fields and inherent methods, a sealed
// trait's variants, a trait's methods. Empty for every other type.
func (c *Checker) shapeOf(t types.Type) string { return c.shapeFrom(t, viewpoint{c: c}) }

// shapeFrom is shapeOf as seen from v: members the reader could not name
// from there are left out, and a count says so.
func (c *Checker) shapeFrom(t types.Type, v viewpoint) string {
	var sb strings.Builder
	switch tt := t.(type) {
	case *types.Pointer:
		return c.shapeFrom(tt.Elem, v)
	case *types.Nullable:
		return c.shapeFrom(tt.Elem, v)
	case *types.Enum:
		return enumShape(tt)
	case *types.Struct:
		tmpl := templateOf(tt)
		d, _ := tmpl.Decl.(*ast.StructDecl)
		if d == nil {
			return ""
		}
		c.resolveStruct(tmpl)
		// the declaration with every implicit word spelled out: the mutability
		// a bare field has (`val`) — what a reader needs to know and cannot
		// see at the use
		sb.WriteString(structHead(tt))
		sb.WriteString(" {\n")
		hidden := 0
		if !d.Extern {
			// what a caller must (and may) give to build one, from here
			sb.WriteString("  // " + constructorLine(tt, v) + "\n")
		}
		for _, fld := range tt.Fields {
			if !v.sees(tt, fld.Pub, fld.Private) {
				hidden++
				continue
			}
			sb.WriteString("  " + fieldDecl(tt, fld) + "\n")
		}
		if statics := c.staticVals[tmpl]; len(statics) > 0 {
			var names []string
			for name := range statics {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				sym := statics[name]
				if !v.sees(tt, sym.Pub, false) {
					hidden++
					continue
				}
				sb.WriteString("  " + visPrefix(sym.Pub, false) + "static val " + name)
				if sym.Global != nil && sym.Global.Type != nil {
					sb.WriteString(": " + sym.Global.Type.String())
				}
				sb.WriteString(" = ...\n")
			}
		}
		for _, name := range sortedMethodNames(c.methods[tmpl]) {
			if strings.HasPrefix(name, "$") {
				continue // the hidden `$init` method: shown as the block below
			}
			m := c.methods[tmpl][name]
			if m.Decl != nil && !v.sees(tt, m.Decl.Pub, m.Decl.Private) {
				hidden++
				continue
			}
			sb.WriteString("  " + methodDecl(tmpl, m) + "\n")
		}
		if d.Init != nil && v.insideType(tt) {
			sb.WriteString("  init" + initParamList(d.InitParams) + " { ... }\n")
		}
		if hidden > 0 {
			// the reader cannot name these from here; the count says the
			// type has more than it shows
			word := "members"
			if hidden == 1 {
				word = "member"
			}
			sb.WriteString(fmt.Sprintf("  // ... and %d %s not visible from here\n", hidden, word))
		}
		if d.Error {
			sb.WriteString("  public fun message(): string\n")
		}
		// the traits the type implements, wherever the impl was written or
		// derived (D58). A trait one of them requires is not listed again:
		// `implement Codable` stands for its Encodable and Decodable.
		var own []*Impl
		for _, list := range c.impls {
			for _, impl := range list {
				if target, ok := impl.Target.(*types.Struct); ok && templateOf(target) == tmpl {
					own = append(own, impl)
				}
			}
		}
		var impls []string
		for _, impl := range own {
			covered := false
			for _, other := range own {
				if other != impl && containsTrait(allSupers(other.Trait), impl.Trait) {
					covered = true
				}
			}
			if covered {
				continue
			}
			line := "  implement " + impl.Trait.Name + typeArgList(impl, impl.Trait)
			if impl.Decl.Derived {
				line += "  // derived"
			}
			impls = append(impls, line)
		}
		sort.Strings(impls)
		for _, line := range impls {
			sb.WriteString(line + "\n")
		}
		sb.WriteString("}")
	case *types.Sealed:
		sb.WriteString(sealedHead(tt) + " {\n")
		if tt.Trait != nil {
			sb.WriteString(c.traitBody(tt.Trait))
		}
		// the variants, as they would read at a use
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
		sb.WriteString(traitHead(tt) + " {\n")
		sb.WriteString(c.traitBody(tt))
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
		sb.WriteString(lazyPrefix(p) + p.Name + ": " + p.Type.String())
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
		if tp.Const {
			names[i] = "const " + tp.Name + ": i64" // a constant, not a type (D121)
		}
	}
	return "<" + strings.Join(names, ", ") + ">"
}

// funDetail renders a template's signature the way it was declared.
func funDetail(t *FuncTemplate) string {
	var sb strings.Builder
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
				sb.WriteString(lazyPrefix(p) + p.Name + ": ")
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
		if t.Sig.CVariadic {
			if len(t.Sig.Params) > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("...") // C's variadic arguments (D123)
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
	hit := c.unknownHit(m, name)
	if hit == "" {
		return ""
	}
	modName := hit[:strings.Index(hit, ".")]
	for _, dep := range m.Deps {
		if dep.Name() == modName {
			return fmt.Sprintf("; did you mean '%s'?", hit)
		}
	}
	return fmt.Sprintf("; did you mean '%s'? (add 'use %s' at the top of the file)", hit, modName)
}

// unknownHit is the qualified name (`codec.Value`, `io.println`) an
// unresolved name most likely meant, or "".
func (c *Checker) unknownHit(m *Module, name string) string {
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
	if home := PreludeHome(name); home != "" {
		hits = append(hits, home+"."+name) // written in the prelude, reached through its module (D75)
	}
	// modules already loaded (imported somewhere in the package)
	for _, mod := range c.pkg.Modules {
		if len(hits) > 0 {
			break
		}
		consider(mod)
	}
	// standard modules that nothing has imported yet
	for _, std := range c.pkg.stdModuleNames() {
		if len(hits) > 0 {
			break
		}
		if _, loaded := c.pkg.Modules["std/"+std]; !loaded {
			if mod, ok := c.pkg.loadStd(std); ok {
				consider(mod)
			}
		}
	}
	if len(hits) == 0 {
		return ""
	}
	return hits[0]
}

// unknownFix writes the qualified name in place of the unresolved one and,
// when the file does not import its module yet, the `use` too — one quick
// fix for "did you mean 'codec.Value'? (add 'use codec' ...)".
//
// Only for a name the prelude writes for a module (D75), where the meaning
// is certain: `veles check --fix` applies every fix, and qualifying any
// other unknown name would be a guess at what a typo meant.
func (c *Checker) unknownFix(file *ast.File, m *Module, span source.Span, name string) *source.Fix {
	if PreludeHome(name) == "" || file == nil || span.File == nil {
		return nil
	}
	hit := c.unknownHit(m, name)
	if hit == "" {
		return nil
	}
	modName := hit[:strings.Index(hit, ".")]
	fix := fixReplace("Write '"+hit+"'", span, hit)
	for _, d := range file.Decls {
		if u, ok := d.(*ast.UseDecl); ok {
			for _, s := range u.Specs {
				if len(s.Path) > 0 && (s.Alias != nil && s.Alias.Name == modName || s.Alias == nil && s.Path[len(s.Path)-1].Name == modName) {
					return fix
				}
			}
		}
	}
	at, beforeUse := useInsertAt(file, span.File.Content)
	line := "use " + modName + "\n"
	if !beforeUse {
		line += "\n" // above a declaration: set apart like any use block
	}
	fix.Title = "Write '" + hit + "' and add 'use " + modName + "'"
	fix.Edits = append(fix.Edits, source.TextEdit{Span: source.Span{File: span.File, Start: at, End: at}, NewText: line})
	return fix
}

// useInsertAt is where a new `use` line goes: at the file's first `use`
// (the formatter merges them afterwards), else above the first
// declaration and the doc comment attached to it — never between the two.
func useInsertAt(file *ast.File, src string) (int, bool) {
	first := -1
	for _, d := range file.Decls {
		start := d.Span().Start
		if _, ok := d.(*ast.UseDecl); ok {
			return lineStart(src, start), true
		}
		if first < 0 || start < first {
			first = start
		}
	}
	if first < 0 {
		return len(src), false
	}
	at := lineStart(src, first)
	for at > 0 {
		prev := lineStart(src, at-1)
		if !strings.HasPrefix(strings.TrimSpace(src[prev:at]), "///") && !strings.HasPrefix(strings.TrimSpace(src[prev:at]), "@") {
			break
		}
		at = prev
	}
	return at, false
}

func lineStart(src string, off int) int {
	if off > len(src) {
		off = len(src)
	}
	return strings.LastIndexByte(src[:off], '\n') + 1
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
	case *ast.EnumDecl:
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
	case *ast.EnumDecl:
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

// monomorphic reports whether a function has one meaning for its types:
// no type parameters of its own or of the type it belongs to, and not a
// trait's default body (checked once per implementing type).
func monomorphic(t *FuncTemplate) bool {
	return len(t.TypeParams) == 0 && t.Trait == nil &&
		(t.Owner == nil || len(t.Owner.TypeParams) == 0) &&
		(t.Impl == nil || len(t.Impl.TypeParams) == 0)
}

// initParamList is `init`'s parameter list as written, or nothing.
func initParamList(params []ast.Param) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Name.Name
		if p.Type != nil {
			parts[i] += ": " + ast.TypeString(p.Type)
		}
		if p.Default != nil {
			parts[i] += " = ..."
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
