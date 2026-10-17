package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D142: `public` is what other packages may use. A program (its root module
// declares `main`) has no other package importing it, so a `public` there
// guards nothing and is noise; the warning carries a fix that removes the
// word, and `veles check --fix` migrates a whole program.

// lintProgramPublic warns on every `public` declaration, field, method and
// static of the entry package when that package is a program.
func (c *Checker) lintProgramPublic() {
	entry := c.pkg.Entry
	if entry == nil || c.testMode {
		return
	}
	if sym := entry.Scope.LookupLocal("main"); sym == nil || sym.Kind != SymFunc {
		return
	}
	for _, m := range c.pkg.Modules {
		if m.Pkg != c.pkg || m.Std {
			continue
		}
		for _, f := range m.Files {
			for _, d := range f.Decls {
				c.publicInDecl(d)
			}
		}
	}
}

func (c *Checker) publicInDecl(d ast.Decl) {
	switch d := d.(type) {
	case *ast.FunDecl:
		c.noPublic(d.Pub, d.Name)
	case *ast.ValDecl:
		c.noPublic(d.Pub, d.Name)
	case *ast.EnumDecl:
		c.noPublic(d.Pub, d.Name)
	case *ast.ErrorAliasDecl:
		c.noPublic(d.Pub, d.Name)
	case *ast.TypeAliasDecl:
		c.noPublic(d.Pub, d.Name)
	case *ast.StructDecl:
		c.noPublic(d.Pub, d.Name)
		for _, fld := range d.Fields {
			c.noPublic(fld.Pub, fld.Name)
		}
		c.publicInFuns(d.Methods)
		for _, sv := range d.Statics {
			c.noPublic(sv.Pub, sv.Name)
		}
		for _, impl := range d.Impls {
			c.publicInFuns(impl.Methods)
		}
	case *ast.TraitDecl:
		c.noPublic(d.Pub, d.Name)
		c.publicInFuns(d.Methods)
	case *ast.ImplDecl:
		c.publicInFuns(d.Methods)
	case *ast.ExecutorDecl:
		c.noPublic(d.Pub, d.Name)
		for _, sv := range d.State {
			c.noPublic(sv.Pub, sv.Name)
		}
		c.publicInFuns(d.Funs)
	}
}

func (c *Checker) publicInFuns(funs []*ast.FunDecl) {
	for _, fn := range funs {
		c.noPublic(fn.Pub, fn.Name)
	}
}

// noPublic reports one `public` on the declaration named by name. The word
// is found on the name's own line, before the name; a declaration that did
// not write it (a synthesized one) has no such text and is skipped.
func (c *Checker) noPublic(pub bool, name ast.Ident) {
	if !pub || name.Pos.File == nil {
		return
	}
	src := name.Pos.File.Content
	lineStart := strings.LastIndexByte(src[:name.Pos.Start], '\n') + 1
	head := src[lineStart:name.Pos.Start]
	at := strings.LastIndex(head, "public ")
	if at < 0 {
		return
	}
	cut := source.Span{File: name.Pos.File, Start: lineStart + at, End: lineStart + at + len("public ")}
	c.warnFix(name.Pos, fixReplace("Remove 'public'", cut, ""),
		"'public' on '%s' does nothing in a program: no other package imports it; leave it off (D142)", name.Name)
}

// namedTypesIn calls visit for every declared type (struct, sealed trait,
// enum, trait) that t mentions, through type arguments and wrappers.
func namedTypesIn(t types.Type, visit func(types.Type)) {
	switch t := t.(type) {
	case *types.Pointer:
		namedTypesIn(t.Elem, visit)
	case *types.Nullable:
		namedTypesIn(t.Elem, visit)
	case *types.Tuple:
		for _, e := range t.Elems {
			namedTypesIn(e, visit)
		}
	case *types.List:
		namedTypesIn(t.Elem, visit)
	case *types.Array:
		namedTypesIn(t.Elem, visit)
	case *types.Map:
		namedTypesIn(t.Key, visit)
		namedTypesIn(t.Value, visit)
	case *types.Set:
		namedTypesIn(t.Elem, visit)
	case *types.Channel:
		namedTypesIn(t.Elem, visit)
	case *types.Task:
		namedTypesIn(t.Result, visit)
	case *types.Func:
		for _, p := range t.Params {
			namedTypesIn(p.Type, visit)
		}
		namedTypesIn(t.Ret, visit)
	case *types.Struct:
		visit(t)
		for _, a := range t.TypeArgs {
			namedTypesIn(a, visit)
		}
	case *types.Sealed:
		visit(t)
		for _, a := range t.TypeArgs {
			namedTypesIn(a, visit)
		}
	case *types.Enum, *types.Trait:
		visit(t)
	}
}

// typeLevel is what decides whether a declared type is visible outside its
// package: its own `public`, and the module it lives in.
func typeLevel(t types.Type) (name, module string, pub, internal bool) {
	switch t := t.(type) {
	case *types.Struct:
		t = templateOf(t)
		return t.Name, t.Module, t.Pub, t.Internal
	case *types.Sealed:
		t = sealedTemplate(t)
		return t.Name, t.Module, t.Pub, t.Internal
	case *types.Enum:
		return t.Name, t.Module, t.Pub, t.Internal
	case *types.Trait:
		return t.Name, t.Module, t.Pub, t.Internal
	}
	return "", "", true, false
}

// lintPublicSignatures reports a `public` function whose parameter or result
// type, or a `public` field whose type, names a type that other packages
// cannot see (D142, Q23): it is neither `public` nor exported by a facade.
func (c *Checker) lintPublicSignatures() {
	exported := map[string]bool{} // "module.Name" of a type a facade exports
	wholeModule := map[string]bool{}
	for _, m := range c.pkg.Modules {
		if m.Scope == nil {
			continue
		}
		for _, sym := range m.Scope.symbols {
			switch {
			case sym.Exported && sym.Kind == SymType:
				if name, module, _, _ := typeLevel(c.symType(sym)); name != "" {
					exported[module+"."+name] = true
				}
			case sym.Facade && sym.Kind == SymModule && sym.Mod != nil:
				wholeModule[sym.Mod.prefix()] = true
			}
		}
	}
	hidden := func(t types.Type, from string) (string, bool) {
		var bad string
		namedTypesIn(t, func(n types.Type) {
			name, module, pub, internal := typeLevel(n)
			if bad != "" || pub || name == "" || !samePackage(module, from) {
				return
			}
			if !internal && (exported[module+"."+name] || wholeModule[module]) {
				return
			}
			bad = name
		})
		return bad, bad != ""
	}
	for _, t := range c.templates {
		d := t.Decl
		if d == nil || !t.Pub || t.Sig == nil || t.Extern || t.Impl != nil && t.Impl.Trait != nil || t.Module == nil || t.Module.Pkg != c.pkg && !t.Module.Std {
			continue
		}
		if t.Owner != nil && !t.Owner.Pub {
			continue // the type is not reachable, so neither is its method
		}
		from := t.Module.prefix()
		for _, p := range t.Sig.Params {
			if bad, ok := hidden(p.Type, from); ok {
				c.publicSignature(t, "parameter '"+p.Name+"'", bad)
			}
		}
		if bad, ok := hidden(t.Sig.Ret, from); ok {
			c.publicSignature(t, "result", bad)
		}
	}
	for _, s := range c.structs {
		if !s.Pub || s.Module == "" {
			continue
		}
		for _, fld := range s.Fields {
			if !fld.Pub {
				continue
			}
			if bad, ok := hidden(fld.Type, s.Module); ok {
				if d, isDecl := s.Decl.(*ast.StructDecl); isDecl {
					for _, f := range d.Fields {
						if f.Name.Name == fld.Name {
							c.errorf(f.Name.Pos, "field '%s' is public but its type '%s' is not visible outside the package: make '%s' public, or drop 'public' from the field (D142)", fld.Name, bad, bad)
						}
					}
				}
			}
		}
	}
}

func (c *Checker) publicSignature(t *FuncTemplate, what, bad string) {
	name := t.Decl.Name
	var fix *source.Fix
	if f := name.Pos.File; f != nil {
		src := f.Content
		lineStart := strings.LastIndexByte(src[:name.Pos.Start], '\n') + 1
		if at := strings.LastIndex(src[lineStart:name.Pos.Start], "public "); at >= 0 {
			cut := source.Span{File: f, Start: lineStart + at, End: lineStart + at + len("public ")}
			fix = fixReplace("Drop 'public' from '"+name.Name+"'", cut, "")
		}
	}
	msg := "'%s' is public but its %s type '%s' is not visible outside the package: make '%s' public, or drop 'public' from '%s' (D142)"
	if strings.HasPrefix(what, "parameter") {
		msg = "'%s' is public but its %s's type '%s' is not visible outside the package: make '%s' public, or drop 'public' from '%s' (D142)"
		what = strings.TrimSuffix(what, "")
	}
	if fix != nil {
		c.errorFix(name.Pos, fix, msg, name.Name, what, bad, bad, name.Name)
		return
	}
	c.errorf(name.Pos, msg, name.Name, what, bad, bad, name.Name)
}
