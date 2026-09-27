package lsp

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Completion. Three situations:
//
//   - `module.` — the module's public declarations;
//   - `value.`  — the members of the value's type, found through the last
//     successful analysis (the buffer usually does not parse at this
//     moment, so the receiver is looked up by name, not by span);
//   - a bare name — declarations in scope, keywords, builtin types.

type completionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

const (
	ciMethod     = 2
	ciFunction   = 3
	ciField      = 5
	ciVariable   = 6
	ciClass      = 7
	ciInterface  = 8
	ciModule     = 9
	ciKeyword    = 14
	ciEnum       = 13
	ciEnumMember = 20
	ciStruct     = 22
)

var keywordCompletions = []string{
	"fun", "val", "var", "const", "if", "else", "loop", "break", "continue", "return", "throw",
	"struct", "enum", "error", "trait", "implement", "extend", "sealed", "public", "private", "internal", "protected", "use", "when", "is", "as", "in",
	"throws", "suspends", "try", "async", "await", "scope", "gather", "race", "with",
	"unsafe", "extern", "mut", "override", "true", "false", "null", "this", "Self", "type",
}

var builtinTypeCompletions = []string{
	"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64", "bool", "string",
	"List", "MutableList", "Map", "MutableMap", "Set", "MutableSet", "Channel", "Range", "Option", "Result",
}

type adder func(label string, kind int, detail string)

func (s *Server) completion(params json.RawMessage) any {
	var p positionParams
	json.Unmarshal(params, &p)
	items := []completionItem{}
	seen := map[string]bool{}
	var add adder = func(label string, kind int, detail string) {
		if seen[label] {
			return
		}
		seen[label] = true
		items = append(items, completionItem{label, kind, detail})
	}
	finish := func() any {
		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
		return map[string]any{"isIncomplete": false, "items": items}
	}

	d := s.docs[p.TextDocument.URI]
	a, _ := s.analysisFor(p.TextDocument.URI)
	receiver, afterDot, off, dot := "", false, 0, 0
	if d != nil {
		f := source.NewFile(d.path, d.text)
		off = positionToOffset(f, p.Position)
		i := off
		for i > 0 && isIdentByte(d.text[i-1]) {
			i--
		}
		if i > 0 && d.text[i-1] == '.' {
			afterDot = true
			dot = i - 1
			j := dot
			for j > 0 && isIdentByte(d.text[j-1]) {
				j--
			}
			receiver = d.text[j:dot]
		}
	}

	if !afterDot {
		if a != nil {
			// the names in scope unqualified (M5): this module's own
			// declarations and the prelude's public ones; every other
			// module is reached through its name
			cur, _ := s.moduleOf(a, d)
			for _, m := range a.pkg.Modules {
				prelude := m.Std && m.Path == "std/prelude"
				if m != cur && !prelude {
					continue
				}
				for _, f := range m.Files {
					for _, decl := range f.Decls {
						if !prelude || isPub(decl) && sema.PreludeHome(declNameOf(decl)) == "" {
							s.addDecl(add, decl)
						}
					}
				}
			}
			if _, f := s.moduleOf(a, d); f != nil {
				for _, decl := range f.Decls {
					if ud, ok := decl.(*ast.UseDecl); ok {
						for _, u := range ud.Specs {
							add(importedName(u), ciModule, "module "+pathString(u.Path))
						}
					}
				}
			}
		}
		for _, k := range keywordCompletions {
			add(k, ciKeyword, "keyword")
		}
		for _, t := range builtinTypeCompletions {
			add(t, ciClass, "builtin type")
		}
		return finish()
	}

	if a != nil && receiver != "" {
		// `module.`
		if m, f := s.moduleOf(a, d); m != nil {
			for _, decl := range f.Decls {
				ud, ok := decl.(*ast.UseDecl)
				if !ok {
					continue
				}
				for _, u := range ud.Specs {
					if importedName(u) == receiver {
						if target := m.Uses[u]; target != nil {
							s.addModuleDecls(add, target)
							return finish()
						}
					}
				}
			}
		}
		// `value.` — a receiver whose type is known gets exactly its
		// members, even when there are none (a Result has no methods)
		idx := a.index
		if idx == nil {
			// mid-edit: the dangling `.` breaks the parse, so a binding
			// introduced in this same edit has no reference yet; check the
			// buffer with the `.` and the partial name after it removed
			idx = s.indexWithout(d, dot, off)
		}
		if idx == nil {
			idx = a.lastGood
		}
		if receiver == "this" {
			// `this.`: the type whose body, impl or extend block holds the cursor
			sc := s.scopeAt(a, d, off)
			if owner := sc.enclosingType(); owner != "" {
				if ref := refNamedIn(idx, sema.OverlayKey(d.path), owner, len(d.text)); ref != nil && ref.Type != nil {
					s.addMembers(add, a, ref.Type, sc)
					return finish()
				}
			}
		}
		if ref := refNamedIn(idx, sema.OverlayKey(d.path), receiver, off); ref != nil {
			if ref.Kind == "module" && ref.Module != nil {
				s.addModuleDecls(add, ref.Module)
				return finish()
			}
			if ref.Type != nil {
				sc := s.scopeAt(a, d, off)
				if ref.Kind == "struct" || ref.Kind == "type" || ref.Kind == "sealed" || ref.Kind == "trait" || ref.Kind == "enum" {
					// `Type.`: the type's namespace — statics and, for a
					// sealed trait, its variants — never instance members
					s.addStatics(add, a, ref.Type, sc)
					return finish()
				}
				s.addMembers(add, a, ref.Type, sc)
				return finish()
			}
		}
	}

	// unknown receiver: every reachable member name in the package plus
	// the builtins
	if a != nil {
		sc := s.scopeAt(a, d, off)
		for _, m := range a.pkg.Modules {
			for _, f := range m.Files {
				for _, decl := range f.Decls {
					switch dd := decl.(type) {
					case *ast.StructDecl:
						for _, fld := range dd.Fields {
							if sc.allows(dd.Name.Name, m, fld.Pub, fld.Private) {
								add(fld.Name.Name, ciField, dd.Name.Name+"."+fld.Name.Name+": "+ast.TypeString(fld.Type))
							}
						}
						for _, mth := range dd.Methods {
							if sc.allows(dd.Name.Name, m, mth.Pub, mth.Private) {
								add(mth.Name.Name, ciMethod, dd.Name.Name+"."+mth.Name.Name+funSignature(mth))
							}
						}
					case *ast.TraitDecl:
						if m == sc.mod || dd.Pub {
							for _, mth := range dd.Methods {
								add(mth.Name.Name, ciMethod, dd.Name.Name+"."+mth.Name.Name+funSignature(mth))
							}
						}
					case *ast.ImplDecl:
						owner := ast.TypeString(dd.Target)
						if !dd.Extend {
							owner = ast.TypeString(dd.Trait)
						}
						for _, mth := range dd.Methods {
							if sc.allows(typeHeadName(dd.Target), m, mth.Pub || !dd.Extend, mth.Private) {
								add(mth.Name.Name, ciMethod, owner+"."+mth.Name.Name+funSignature(mth))
							}
						}
					}
				}
			}
		}
	}
	for _, t := range []types.Type{&types.List{Elem: types.TI64}, &types.Map{Key: types.TString, Value: types.TI64}, types.TString} {
		family := sema.BuiltinFamily(t)
		for _, d := range sema.BuiltinMethods(t) {
			add(d.Name, ciMethod, family+"."+d.Name+d.Sig)
		}
	}
	return finish()
}

func (s *Server) addModuleDecls(add adder, m *sema.Module) {
	for _, f := range m.Files {
		for _, decl := range f.Decls {
			if isPub(decl) {
				s.addDecl(add, decl)
			}
		}
	}
	// a module whose names are written in the prelude (D75: `codec.Value`)
	if m.Scope == nil {
		return
	}
	home := m.Path[strings.LastIndexByte(m.Path, '/')+1:]
	for _, sym := range m.Scope.Symbols() {
		if sym.Module == nil || sym.Module == m || sym.Module.Path != "std/prelude" {
			continue
		}
		for _, f := range sym.Module.Files {
			for _, decl := range f.Decls {
				if isPub(decl) && declNameOf(decl) == sym.Name && sema.PreludeHome(sym.Name) == home {
					s.addDecl(add, decl)
				}
			}
		}
	}
}

func declNameOf(decl ast.Decl) string {
	switch dd := decl.(type) {
	case *ast.FunDecl:
		return dd.Name.Name
	case *ast.StructDecl:
		return dd.Name.Name
	case *ast.TraitDecl:
		return dd.Name.Name
	case *ast.EnumDecl:
		return dd.Name.Name
	case *ast.ValDecl:
		return dd.Name.Name
	case *ast.ErrorAliasDecl:
		return dd.Name.Name
	case *ast.TypeAliasDecl:
		return dd.Name.Name
	}
	return ""
}

// addMembers offers the fields and methods of a value of type t: the
// compiler's built-ins from the catalogue, the struct's own fields and
// methods, and every `extend` and `impl` block whose target names the type.
func (s *Server) addMembers(add adder, a *analysis, t types.Type, sc *scope) {
	switch tt := t.(type) {
	case *types.Pointer:
		s.addMembers(add, a, tt.Elem, sc)
		return
	case *types.Nullable:
		s.addMembers(add, a, tt.Elem, sc) // reached through `?.`
		return
	case *types.Tuple:
		for i, e := range tt.Elems {
			add(strconv.Itoa(i), ciField, e.String())
		}
		return
	case *types.Trait:
		s.addTraitMethods(add, a, tt.Name)
		return
	case *types.Sealed:
		s.addTraitMethods(add, a, tt.Name)
		return
	case *types.ErrorUnion:
		// every member implements Error (D4); its methods dispatch on the union
		s.addTraitMethods(add, a, "Error")
		return
	case *types.Range:
		add("lo", ciField, "Range.lo")
		add("hi", ciField, "Range.hi")
		add("inclusive", ciField, "Range.inclusive")
	case *types.Enum:
		// a value has its number and the catalogued methods (D57)
		add("value", ciField, tt.Name+".value: "+tt.Base.Name)
		for _, d := range sema.BuiltinMethods(t) {
			if !d.Static() {
				add(d.Name, ciMethod, "enum."+d.Name+d.Sig)
			}
		}
		return
	case *types.Struct:
		base := tt
		if tt.Template != nil {
			base = tt.Template
		}
		if d, ok := base.Decl.(*ast.StructDecl); ok {
			// only what the cursor may name (M5): private members inside
			// the type, unmarked ones inside its module, public ones anywhere
			ownerMod := sc.moduleOfDecl(d)
			for _, fld := range d.Fields {
				if sc.allows(d.Name.Name, ownerMod, fld.Pub, fld.Private) {
					add(fld.Name.Name, ciField, tt.String()+"."+fld.Name.Name+": "+ast.TypeString(fld.Type))
				}
			}
			for _, mth := range d.Methods {
				if !mth.Static && sc.allows(d.Name.Name, ownerMod, mth.Pub, mth.Private) {
					add(mth.Name.Name, ciMethod, tt.Name+"."+mth.Name.Name+funSignature(mth))
				}
			}
			if d.Variant != nil {
				s.addTraitMethods(add, a, typeHeadName(d.Variant))
			}
		}
	}
	family := sema.BuiltinFamily(t)
	for _, d := range sema.BuiltinMethods(t) {
		add(d.Name, ciMethod, family+"."+d.Name+d.Sig)
	}
	// the names an impl or extend target may spell this type with: a
	// mutable collection also has its immutable form's blocks (D25)
	heads := map[string]bool{}
	if st, ok := t.(*types.Struct); ok {
		heads[st.Name] = true
	} else if family != "" {
		heads[family] = true
		if base := strings.TrimPrefix(family, "Mutable"); base != family {
			heads[base] = true
		}
		if b, ok := t.(*types.Basic); ok {
			heads[b.Name] = true // `i64`, `f64`: the family is "int"/"float"
		}
	}
	if len(heads) == 0 {
		return
	}
	for _, m := range a.pkg.Modules {
		for _, f := range m.Files {
			for _, decl := range f.Decls {
				impl, ok := decl.(*ast.ImplDecl)
				if !ok || !heads[typeHeadName(impl.Target)] {
					continue
				}
				owner := ast.TypeString(impl.Target)
				if !impl.Extend {
					owner = ast.TypeString(impl.Trait)
				}
				for _, mth := range impl.Methods {
					// an impl's methods follow the trait's visibility; an
					// extend's are members like the struct's own (M5)
					if sc.allows(typeHeadName(impl.Target), m, mth.Pub || !impl.Extend, mth.Private) {
						add(mth.Name.Name, ciMethod, owner+"."+mth.Name.Name+funSignature(mth))
					}
				}
				if !impl.Extend {
					s.addTraitMethods(add, a, typeHeadName(impl.Trait))
				}
			}
		}
	}
}

// addStatics offers what lives in a type's namespace, `Type.`: its
// `static val`s and `static fun`s (in the body, in extend blocks), and a
// sealed trait's variants — filtered by what the cursor may name (M5).
func (s *Server) addStatics(add adder, a *analysis, t types.Type, sc *scope) {
	switch tt := t.(type) {
	case *types.Struct:
		base := tt
		if tt.Template != nil {
			base = tt.Template
		}
		d, ok := base.Decl.(*ast.StructDecl)
		if !ok {
			return
		}
		ownerMod := sc.moduleOfDecl(d)
		for _, sv := range d.Statics {
			if !sc.allows(d.Name.Name, ownerMod, sv.Pub, false) {
				continue
			}
			detail := tt.Name + "." + sv.Name.Name
			if sv.Type != nil {
				detail += ": " + ast.TypeString(sv.Type)
			}
			add(sv.Name.Name, ciField, "static val "+detail)
		}
		for _, mth := range d.Methods {
			if mth.Static && sc.allows(d.Name.Name, ownerMod, mth.Pub, mth.Private) {
				add(mth.Name.Name, ciMethod, "static fun "+tt.Name+"."+mth.Name.Name+funSignature(mth))
			}
		}
	case *types.Sealed:
		for _, v := range tt.Variants {
			add(v.Name, ciStruct, "struct "+v.Name+" : "+tt.Name)
		}
	case *types.Enum:
		// the members, then values() / fromValue(n) / parse(s) (D57)
		for _, m := range tt.Members {
			add(m.Name, ciEnumMember, tt.Name+"."+m.Name+" = "+sema.EnumMemberText(m))
		}
		for _, d := range sema.BuiltinMethods(t) {
			if d.Static() {
				add(d.Name, ciMethod, "static fun "+tt.Name+"."+d.Name+d.Sig)
			}
		}
		return
	case *types.Trait:
		return // `Trait.` names nothing callable; the methods are on values
	}
	// static functions declared in extend and impl blocks for the type —
	// a struct's, or a builtin's (`MutableList.make`, `i64.parse`)
	heads := typeHeads(t)
	if len(heads) == 0 {
		return
	}
	for _, m := range a.pkg.Modules {
		for _, f := range m.Files {
			for _, decl := range f.Decls {
				impl, ok := decl.(*ast.ImplDecl)
				if !ok || !heads[typeHeadName(impl.Target)] {
					continue
				}
				name := ast.TypeString(impl.Target)
				for _, mth := range impl.Methods {
					if mth.Static && sc.allows(typeHeadName(impl.Target), m, mth.Pub || !impl.Extend, mth.Private) {
						add(mth.Name.Name, ciMethod, "static fun "+name+"."+mth.Name.Name+funSignature(mth))
					}
				}
			}
		}
	}
}

// typeHeads lists the names an impl or extend target may spell a type
// with: a struct's name; a builtin's family and, for a mutable collection,
// its immutable form's (D25); a numeric type's own name.
func typeHeads(t types.Type) map[string]bool {
	heads := map[string]bool{}
	if st, ok := t.(*types.Struct); ok {
		heads[st.Name] = true
		return heads
	}
	family := sema.BuiltinFamily(t)
	if family != "" {
		heads[family] = true
		if base := strings.TrimPrefix(family, "Mutable"); base != family {
			heads[base] = true
		}
		if b, ok := t.(*types.Basic); ok {
			heads[b.Name] = true
		}
	}
	return heads
}

func (s *Server) addTraitMethods(add adder, a *analysis, traitName string) {
	for _, m := range a.pkg.Modules {
		for _, f := range m.Files {
			for _, decl := range f.Decls {
				if td, ok := decl.(*ast.TraitDecl); ok && td.Name.Name == traitName {
					for _, mth := range td.Methods {
						add(mth.Name.Name, ciMethod, td.Name.Name+"."+mth.Name.Name+funSignature(mth))
					}
				}
			}
		}
	}
}

func (s *Server) addDecl(add adder, decl ast.Decl) {
	switch dd := decl.(type) {
	case *ast.FunDecl:
		add(dd.Name.Name, ciFunction, "fun "+dd.Name.Name+funSignature(dd))
	case *ast.StructDecl:
		if dd.Error {
			add(dd.Name.Name, ciStruct, "error "+dd.Name.Name)
		} else {
			add(dd.Name.Name, ciStruct, "struct "+dd.Name.Name)
		}
	case *ast.ErrorAliasDecl:
		add(dd.Name.Name, ciStruct, "error "+dd.Name.Name+" = "+ast.TypeString(dd.Members))
	case *ast.TypeAliasDecl:
		add(dd.Name.Name, ciClass, "type "+dd.Name.Name+" = "+ast.TypeString(dd.Type))
	case *ast.EnumDecl:
		add(dd.Name.Name, ciEnum, "enum "+dd.Name.Name)
	case *ast.TraitDecl:
		if dd.Sealed {
			add(dd.Name.Name, ciStruct, "sealed trait "+dd.Name.Name)
		} else {
			add(dd.Name.Name, ciInterface, "trait "+dd.Name.Name)
		}
	case *ast.ValDecl:
		add(dd.Name.Name, ciVariable, dd.Kind.String()+" "+dd.Name.Name)
	}
}

func isPub(decl ast.Decl) bool {
	switch dd := decl.(type) {
	case *ast.FunDecl:
		return dd.Pub
	case *ast.StructDecl:
		return dd.Pub
	case *ast.ErrorAliasDecl:
		return dd.Pub
	case *ast.TypeAliasDecl:
		return dd.Pub
	case *ast.EnumDecl:
		return dd.Pub
	case *ast.TraitDecl:
		return dd.Pub
	case *ast.ValDecl:
		return dd.Pub
	}
	return false
}

// moduleOf finds the module and parsed file of an open document.
func (s *Server) moduleOf(a *analysis, d *document) (*sema.Module, *ast.File) {
	if d == nil {
		return nil, nil
	}
	key := sema.OverlayKey(d.path)
	for _, m := range a.pkg.Modules {
		for _, f := range m.Files {
			if sema.OverlayKey(f.Source.Path) == key {
				return m, f
			}
		}
	}
	return nil, nil
}

func importedName(u *ast.UseSpec) string {
	if u.Alias != nil {
		return u.Alias.Name
	}
	return u.Path[len(u.Path)-1].Name
}

func pathString(path []ast.Ident) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = p.Name
	}
	return strings.Join(parts, ".")
}

func typeHeadName(t ast.Type) string {
	switch tt := t.(type) {
	case *ast.NamedType:
		return tt.Path[len(tt.Path)-1].Name
	case *ast.PointerType:
		return typeHeadName(tt.Elem)
	}
	return ""
}

// indexWithout checks the package as if the buffer were d.text with the
// bytes [from, to) removed, and returns the resulting index, or nil when
// even that does not parse. Used by completion on a non-parsing buffer.
func (s *Server) indexWithout(d *document, from, to int) *sema.Index {
	if from < 0 || to > len(d.text) || from >= to {
		return nil
	}
	overlay := make(map[string]string, len(s.overlay))
	for k, v := range s.overlay {
		overlay[k] = v
	}
	overlay[sema.OverlayKey(d.path)] = d.text[:from] + d.text[to:]
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackageOverlay(d.path, diags, overlay)
	if err != nil || diags.HasErrors() {
		return nil
	}
	return sema.CheckIndex(pkg, diags)
}

// refNamedIn finds the reference to `name` in a file closest before `off`.
func refNamedIn(idx *sema.Index, fileKey, name string, off int) *sema.Ref {
	if idx == nil {
		return nil
	}
	var best *sema.Ref
	for i := range idx.Refs {
		r := &idx.Refs[i]
		if r.Name != name || r.Span.File == nil || sema.OverlayKey(r.Span.File.Path) != fileKey {
			continue
		}
		if r.Span.Start <= off && (best == nil || r.Span.Start > best.Span.Start) {
			best = r
		}
	}
	if best == nil {
		for i := range idx.Refs {
			if idx.Refs[i].Name == name {
				return &idx.Refs[i]
			}
		}
	}
	return best
}

func isIdentByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// scope is where a completion happens: the module of the buffer and the
// offset in it. It decides which members are reachable (M5): private
// ones only inside the type's own declarations, unmarked ones only inside
// the type's module, public ones everywhere.
type scope struct {
	s    *Server
	a    *analysis
	mod  *sema.Module
	file *ast.File
	off  int
}

func (s *Server) scopeAt(a *analysis, d *document, off int) *scope {
	sc := &scope{s: s, a: a, off: off}
	if a != nil && d != nil {
		sc.mod, sc.file = s.moduleOf(a, d)
	}
	return sc
}

// insideType reports whether the cursor is inside a declaration owned by
// the struct named owner in the buffer's module: its body, or an impl or
// extend block whose target names it.
func (sc *scope) insideType(owner string) bool {
	if sc.mod == nil {
		return false
	}
	within := func(sp source.Span, f *ast.File) bool {
		return sp.File != nil && f != nil && sp.File == f.Source && sc.off >= sp.Start && sc.off <= sp.End
	}
	for _, f := range sc.mod.Files {
		for _, decl := range f.Decls {
			switch dd := decl.(type) {
			case *ast.StructDecl:
				if dd.Name.Name == owner && within(dd.Pos, f) {
					return true
				}
			case *ast.ImplDecl:
				if typeHeadName(dd.Target) == owner && within(dd.Pos, f) {
					return true
				}
			}
		}
	}
	return false
}

// enclosingType names the struct whose body, impl or extend block holds
// the cursor, or "" outside any.
func (sc *scope) enclosingType() string {
	if sc.mod == nil {
		return ""
	}
	within := func(sp source.Span, f *ast.File) bool {
		return sp.File != nil && f != nil && sp.File == f.Source && sc.off >= sp.Start && sc.off <= sp.End
	}
	for _, f := range sc.mod.Files {
		for _, decl := range f.Decls {
			switch dd := decl.(type) {
			case *ast.StructDecl:
				if within(dd.Pos, f) {
					return dd.Name.Name
				}
			case *ast.ImplDecl:
				if within(dd.Pos, f) {
					return typeHeadName(dd.Target)
				}
			}
		}
	}
	return ""
}

// moduleOfDecl finds the module a declaration was parsed in.
func (sc *scope) moduleOfDecl(decl ast.Decl) *sema.Module {
	if sc.a == nil {
		return nil
	}
	for _, m := range sc.a.pkg.Modules {
		for _, f := range m.Files {
			for _, d := range f.Decls {
				if d == decl {
					return m
				}
			}
		}
	}
	return nil
}

// allows reports whether a member of owner (declared in ownerMod) with
// the given visibility can be named from the cursor.
func (sc *scope) allows(owner string, ownerMod *sema.Module, pub, private bool) bool {
	if private {
		return sc.insideType(owner)
	}
	if ownerMod != nil && sc.mod != nil && ownerMod != sc.mod {
		return pub
	}
	return true
}
