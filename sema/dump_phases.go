package sema

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Phase dumps (plan S10b): text renderings of what the checker holds between
// its passes, the oracles a checker written in Veles is gated against phase by
// phase (veles-selfhost-sema-plan.md §2.1). Like the program dumps they are
// deterministic: sorted by name, no pointers.
//
//	modules  the module graph: files, imports and what each resolved to (after loading)
//	scopes   the declarations of every module and the names each file imports,
//	         before any body is checked (after collecting)
//	decls    the resolved declarations: fields, members, signatures, impls (after checking)
//	effects  what each function throws and whether it suspends, once inference
//	         has settled, and how many rounds that took (after checking)
const (
	DumpModules = "modules"
	DumpScopes  = "scopes"
	DumpDecls   = "decls"
	DumpEffects = "effects"
)

// PhaseKinds lists the dump names that need a look inside the checker.
var PhaseKinds = []string{DumpModules, DumpScopes, DumpDecls, DumpEffects}

// IsPhaseDump reports whether kind is one of PhaseKinds.
func IsPhaseDump(kind string) bool {
	for _, k := range PhaseKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// DumpOptions are what every dump takes.
type DumpOptions struct {
	// Std includes the standard library's symbols.
	Std bool
	// Spans adds where each thing is written: `@file:line:col-line:col` after a
	// declaration, `at:` fields in a body. Paths are relative to the package root.
	Spans bool
}

// PhaseDump is a dump of one kind in progress: handed to CheckWithDump, which
// writes it when the checker reaches the pass the kind belongs to.
type PhaseDump struct {
	w    io.Writer
	kind string
	opts DumpOptions
	root string
}

// NewPhaseDump prepares a dump of kind (one of PhaseKinds) to w.
func NewPhaseDump(w io.Writer, kind string, opts DumpOptions) (*PhaseDump, error) {
	if !IsPhaseDump(kind) {
		return nil, errors.New("unknown dump " + strconv.Quote(kind) + " (one of " + strings.Join(append(append([]string{}, PhaseKinds...), DumpKinds...), ", ") + ")")
	}
	return &PhaseDump{w: w, kind: kind, opts: opts}, nil
}

// CheckWithDump is Check (or CheckTests), writing d on the way.
func CheckWithDump(pkg *Package, diags *source.Diagnostics, release, testMode bool, d *PhaseDump) *Program {
	if testMode {
		dropForeignTestFiles(pkg)
	} else if pkg.NeedMain {
		dropTestFiles(pkg)
	}
	prog, _ := checkCollectObserved(pkg, diags, release, testMode, nil, d)
	return prog
}

func (d *PhaseDump) line(depth int, format string, args ...any) {
	fmt.Fprint(d.w, strings.Repeat("  ", depth))
	fmt.Fprintf(d.w, format, args...)
	fmt.Fprintln(d.w)
}

// at is where a thing is written, when the dump was asked for spans.
func (d *PhaseDump) at(sp source.Span) string {
	if !d.opts.Spans || sp.File == nil {
		return ""
	}
	return "  @" + relPath(d.root, sp.File.Path) + ":" + spanRange(sp)
}

// spanRange renders a span as `line:col-line:col`.
func spanRange(sp source.Span) string {
	l1, c1 := sp.File.Position(sp.Start)
	l2, c2 := sp.File.Position(sp.End)
	return fmt.Sprintf("%d:%d-%d:%d", l1, c1, l2, c2)
}

// relPath is path relative to the package root, with forward slashes; a path
// that is not under it (the embedded standard library) is kept as it is.
func relPath(root, path string) string {
	if root == "" || strings.HasPrefix(path, "<") {
		return filepath.ToSlash(path)
	}
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func level(pub, internal bool) string {
	switch {
	case pub:
		return "public"
	case internal:
		return "internal"
	}
	return "package"
}

func sortedModules(pkg *Package, withStd bool) []*Module {
	var mods []*Module
	for _, m := range pkg.Modules {
		if m.Std && !withStd {
			continue
		}
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
	return mods
}

func moduleLabel(m *Module) string {
	if m.Path == "" {
		return "main"
	}
	return m.Path
}

// ---------------------------------------------------------------------------
// the hooks the checker calls

func (d *PhaseDump) afterLoad(pkg *Package) {
	d.root = pkg.Root
	if d.kind == DumpModules {
		d.modules(pkg)
	}
}

func (d *PhaseDump) afterCollect(c *Checker) {
	if d.kind == DumpScopes {
		d.scopes(c.pkg)
	}
}

func (d *PhaseDump) afterCheck(c *Checker, prog *Program) {
	if prog == nil {
		return
	}
	switch d.kind {
	case DumpDecls:
		d.decls(c)
	case DumpEffects:
		d.effects(c, prog)
	}
}

// ---------------------------------------------------------------------------
// modules

func (d *PhaseDump) modules(pkg *Package) {
	for _, m := range sortedModules(pkg, d.opts.Std) {
		kind := "module"
		if m.Std {
			kind = "std module"
		}
		d.line(0, "%s %s", kind, moduleLabel(m))
		for _, f := range m.Files {
			d.line(1, "file %s", relPath(pkg.Root, f.Source.Path))
			for _, decl := range f.Decls {
				u, ok := decl.(*ast.UseDecl)
				if !ok {
					continue
				}
				for _, spec := range u.Specs {
					d.line(2, "%s", d.useText(m, u, spec))
				}
			}
		}
		var deps []string
		for _, dep := range m.Deps {
			deps = append(deps, moduleLabel(dep))
		}
		sort.Strings(deps)
		if len(deps) > 0 {
			d.line(1, "needs %s", strings.Join(deps, ", "))
		}
	}
}

// useText renders one import as written and what it resolved to.
func (d *PhaseDump) useText(m *Module, u *ast.UseDecl, spec *ast.UseSpec) string {
	var path []string
	for _, p := range spec.Path {
		path = append(path, p.Name)
	}
	var b strings.Builder
	if u.Pub {
		b.WriteString("public ")
	}
	b.WriteString("use " + strings.Join(path, "."))
	if spec.Alias != nil {
		b.WriteString(" as " + spec.Alias.Name)
	}
	if len(spec.Names) > 0 {
		var names []string
		for _, n := range spec.Names {
			s := n.Name.Name
			if n.Alias != nil {
				s += " as " + n.Alias.Name
			}
			names = append(names, s)
		}
		b.WriteString(" { " + strings.Join(names, ", ") + " }")
	}
	if target, ok := m.Uses[spec]; ok && target != nil {
		b.WriteString(" -> " + moduleLabel(target))
	} else {
		b.WriteString(" -> unresolved")
	}
	b.WriteString(d.at(spec.Pos))
	return b.String()
}

// ---------------------------------------------------------------------------
// scopes

func symbolKind(k SymbolKind) string {
	switch k {
	case SymType:
		return "type"
	case SymFunc:
		return "fun"
	case SymGlobal:
		return "global"
	case SymLocal:
		return "local"
	case SymModule:
		return "module"
	case SymVariantCtor:
		return "constructor"
	case SymFieldHandle:
		return "field"
	}
	return "?"
}

func symbolNames(s *Scope) []string {
	var names []string
	for name := range s.symbols {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (d *PhaseDump) scopes(pkg *Package) {
	for _, m := range sortedModules(pkg, d.opts.Std) {
		d.line(0, "module %s", moduleLabel(m))
		if m.Scope != nil {
			for _, name := range symbolNames(m.Scope) {
				sym := m.Scope.symbols[name]
				d.line(1, "%s %s  %s%s", symbolKind(sym.Kind), name, level(sym.Pub, sym.Internal), d.at(sym.Span))
			}
		}
		for _, f := range m.Files {
			imports := m.Imports[f]
			if imports == nil || len(imports.symbols) == 0 {
				continue
			}
			d.line(1, "file %s imports", relPath(pkg.Root, f.Source.Path))
			for _, name := range symbolNames(imports) {
				sym := imports.symbols[name]
				target := ""
				if sym.Kind == SymModule && sym.Mod != nil {
					target = " -> " + moduleLabel(sym.Mod)
				} else if sym.Module != nil {
					target = " <- " + moduleLabel(sym.Module)
				}
				d.line(2, "%s %s%s", symbolKind(sym.Kind), name, target)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// decls

func typeParamsText(tps []*types.TypeParam) string {
	if len(tps) == 0 {
		return ""
	}
	var parts []string
	for _, tp := range tps {
		parts = append(parts, tp.String())
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

func sigText(sig *types.Func) string {
	if sig == nil {
		return "?"
	}
	return sig.String()
}

func (d *PhaseDump) decls(c *Checker) {
	for _, m := range sortedModules(c.pkg, d.opts.Std) {
		d.line(0, "module %s", moduleLabel(m))
		if m.Scope != nil {
			for _, name := range symbolNames(m.Scope) {
				d.symbolDecl(c, m, m.Scope.symbols[name])
			}
		}
		d.implDecls(c, m)
	}
}

func (d *PhaseDump) symbolDecl(c *Checker, m *Module, sym *Symbol) {
	lv := level(sym.Pub, sym.Internal)
	switch sym.Kind {
	case SymFunc:
		t := sym.Func
		d.line(1, "fun %s%s: %s  %s%s%s", sym.Name, typeParamsText(t.TypeParams), sigText(t.Sig), lv, inferred(t), d.at(sym.Span))
	case SymGlobal:
		typ := "?"
		if sym.Global != nil && sym.Global.Type != nil {
			typ = sym.Global.Type.String()
		}
		d.line(1, "global %s: %s  %s%s", sym.Name, typ, lv, d.at(sym.Span))
	case SymType:
		d.typeDecl(c, m, sym, lv)
	}
}

func (d *PhaseDump) typeDecl(c *Checker, m *Module, sym *Symbol, lv string) {
	switch t := sym.Type.(type) {
	case *types.Struct:
		kind := "struct"
		if t.Sealed != nil {
			kind = "variant of " + t.Sealed.Name
		}
		if t.Extern {
			kind = "extern " + kind
		}
		d.line(1, "%s %s%s  %s%s", kind, t.Name, typeParamsText(t.TypeParams), lv, d.at(sym.Span))
		for _, f := range t.Fields {
			flags := fieldFlags(f)
			d.line(2, "field %s: %s%s", f.Name, f.Type, flags)
		}
		methods := c.methods[t]
		var names []string
		for name := range methods {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			tmpl := methods[name]
			d.line(2, "fun %s%s: %s  %s%s", name, typeParamsText(tmpl.TypeParams), sigText(tmpl.Sig), level(tmpl.Pub, tmpl.Internal), inferred(tmpl))
		}
	case *types.Sealed:
		d.line(1, "sealed %s%s  %s%s", t.Name, typeParamsText(t.TypeParams), lv, d.at(sym.Span))
		for _, v := range t.Variants {
			d.line(2, "variant %s", v.Name)
		}
	case *types.Enum:
		d.line(1, "enum %s: %s  %s%s", t.Name, t.Base, lv, d.at(sym.Span))
		for _, mem := range t.Members {
			sign := ""
			if mem.Neg {
				sign = "-"
			}
			d.line(2, "member %s = %s%d", mem.Name, sign, mem.Value)
		}
	case *types.Trait:
		supers := ""
		if len(t.Supers) > 0 {
			var names []string
			for _, s := range t.Supers {
				names = append(names, s.Name)
			}
			supers = " : " + strings.Join(names, ", ")
		}
		d.line(1, "trait %s%s%s  %s%s", t.Name, typeParamsText(t.TypeParams), supers, lv, d.at(sym.Span))
		for _, a := range t.AssocTypes {
			d.line(2, "type %s", a)
		}
		for _, name := range t.MethodList {
			d.line(2, "fun %s: %s", name, sigText(t.Methods[name]))
		}
	case nil:
		if sym.TypeAlias != nil {
			d.line(1, "type %s  %s%s", sym.Name, lv, d.at(sym.Span))
		}
	}
}

func fieldFlags(f *types.Field) string {
	var flags []string
	if f.Var {
		flags = append(flags, "var")
	}
	if f.Protected {
		flags = append(flags, "protected")
	}
	if f.Private {
		flags = append(flags, "private")
	} else if f.Pub {
		flags = append(flags, "public")
	} else if f.Internal {
		flags = append(flags, "internal")
	}
	if f.HasDefault {
		flags = append(flags, "default")
	}
	if f.Init {
		flags = append(flags, "init")
	}
	if len(flags) == 0 {
		return ""
	}
	return "  (" + strings.Join(flags, ", ") + ")"
}

// implDecls writes the implements and extends declared in module m.
func (d *PhaseDump) implDecls(c *Checker, m *Module) {
	type entry struct {
		title string
		impl  *Impl
	}
	var entries []entry
	for _, impls := range c.impls {
		for _, im := range impls {
			if im.Module == m {
				entries = append(entries, entry{"implement " + im.Trait.Name + " for " + im.Target.String(), im})
			}
		}
	}
	for _, im := range c.extends {
		if im.Module == m {
			entries = append(entries, entry{"extend " + im.Target.String(), im})
		}
	}
	for _, im := range c.foreignExts {
		if im.Module == m {
			entries = append(entries, entry{"extend " + im.Target.String() + " (foreign)", im})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].title < entries[j].title })
	for _, e := range entries {
		d.line(1, "%s%s", e.title, typeParamsText(e.impl.TypeParams))
		var names []string
		for name := range e.impl.Methods {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			tmpl := e.impl.Methods[name]
			derived := ""
			if e.impl.Derived[name] {
				derived = "  (derived)"
			}
			d.line(2, "fun %s%s: %s%s", name, typeParamsText(tmpl.TypeParams), sigText(tmpl.Sig), derived)
		}
		var assoc []string
		for name := range e.impl.AssocTypes {
			assoc = append(assoc, name)
		}
		sort.Strings(assoc)
		for _, name := range assoc {
			d.line(2, "type %s = %s", name, e.impl.AssocTypes[name])
		}
	}
}

// ---------------------------------------------------------------------------
// effects

func (d *PhaseDump) effects(c *Checker, prog *Program) {
	d.line(0, "rounds %d", c.rounds)
	for _, f := range sortedFuncs(prog) {
		if !d.opts.Std && fromStd(f.Span) {
			continue
		}
		var parts []string
		if f.Sig != nil && f.Sig.Effects.Throws {
			if e := f.Sig.Effects.Error; e != nil {
				parts = append(parts, "throws "+e.String())
			} else {
				parts = append(parts, "throws ?")
			}
		}
		if f.Suspends {
			parts = append(parts, "suspends")
		}
		if f.Conditional {
			parts = append(parts, "conditional")
		}
		effects := "pure"
		if len(parts) > 0 {
			effects = strings.Join(parts, " ")
		}
		d.line(0, "fun %s: %s%s", f.Name, effects, d.at(f.Span))
	}
}

// inferred is what a declaration with a bare `throws` came to throw (D45), for
// the line that shows its signature, which can only say `throws`.
func inferred(t *FuncTemplate) string {
	if t.InferError == nil {
		return ""
	}
	return "  [throws " + t.InferError.String() + "]"
}
