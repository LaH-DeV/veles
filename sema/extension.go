package sema

import (
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Extension functions (D173): `extend T with fun f() …` or `extend T { … }`
// on a type the package does not own. The call `x.f()` is resolved statically
// to a plain function taking x; the function is in scope only in files that
// import its module, it never wins over an inherent method, and it is no part
// of the type's interface (no trait bound, no `is`, no vtable).

// extCandidate is one extension function a method call may mean.
type extCandidate struct {
	t     *FuncTemplate
	subst map[*types.TypeParam]types.Type
}

// extensionInScope reports whether the file being checked sees ext: its own
// module always does, any other through a `use` of that module.
func (f *fnCtx) extensionInScope(ext *Impl) bool {
	if ext.Module == f.module {
		return true
	}
	if f.file == nil {
		return false
	}
	for _, d := range f.file.Decls {
		u, ok := d.(*ast.UseDecl)
		if !ok {
			continue
		}
		for _, s := range u.Specs {
			if f.module.Uses[s] == ext.Module {
				return true
			}
		}
	}
	return false
}

// extensionsFor lists the visible extension functions called name that apply
// to a receiver of type rt. unmet is the complaint when one matched the
// receiver but its type parameter bounds did not hold.
func (f *fnCtx) extensionsFor(rt types.Type, name string) (found []extCandidate, unmet string) {
	for _, view := range receiverViews(rt) {
		for _, ext := range f.c.foreignExts {
			t, ok := ext.Methods[name]
			if !ok || !f.extensionInScope(ext) {
				continue
			}
			m := map[*types.TypeParam]types.Type{}
			if !unify(ext.Target, view, m) {
				continue
			}
			bounded := true
			for _, tp := range ext.TypeParams {
				for _, bound := range tp.Bounds {
					if bt, ok := m[tp]; ok && !f.implements(bt, bound) {
						unmet = "'" + name + "' on '" + rt.String() + "' requires '" + bt.String() + "' to implement '" + bound.Name + "' (extend<" + tp.Name + ": " + bound.Name + "> " + ext.Target.String() + ")" + sendableHint(bound)
						bounded = false
					}
				}
			}
			if bounded {
				found = append(found, extCandidate{t, m})
			}
		}
	}
	return found, unmet
}

// unimportedExtensionHint completes "no method 'name'" when an extension
// function of that name exists for the receiver in a module this file does
// not import.
func (f *fnCtx) unimportedExtensionHint(rt types.Type, name string) string {
	for _, view := range receiverViews(rt) {
		for _, ext := range f.c.foreignExts {
			if _, ok := ext.Methods[name]; !ok || f.extensionInScope(ext) {
				continue
			}
			if unify(ext.Target, view, map[*types.TypeParam]types.Type{}) {
				return "; '" + name + "' is an extension function in module '" + ext.Module.Name() + "': add 'use " + ext.Module.Name() + "' (D173)"
			}
		}
	}
	return ""
}

// indexExtensions tells every reference to a module what its extension
// functions are (D173), for the module's hover and the hint on its `use`
// line: a file that imports a module gets methods nothing in the module's
// own list of declarations shows. Only the ones the referring file may call
// are named.
func (c *Checker) indexExtensions() {
	if c.index == nil || len(c.foreignExts) == 0 {
		return
	}
	fileModule := map[*source.File]*Module{}
	for _, m := range c.pkg.Modules {
		for _, f := range m.Files {
			fileModule[f.Source] = m
		}
	}
	byModule := map[*Module][]*FuncTemplate{}
	for _, ext := range c.foreignExts {
		names := make([]string, 0, len(ext.Methods))
		for name := range ext.Methods {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			byModule[ext.Module] = append(byModule[ext.Module], ext.Methods[name])
		}
	}
	for i := range c.index.Refs {
		r := &c.index.Refs[i]
		from := fileModule[r.Span.File]
		if r.Kind != "module" || r.Module == nil || from == nil || from == r.Module {
			continue
		}
		var lines []string
		seen := map[string]bool{}
		for _, t := range byModule[r.Module] {
			if !declVisible(t.Pub, t.Internal, false, t.Module.Path, from.Path) {
				continue
			}
			c.resolveSignature(t)
			// as it is written: the visibility leads, `public extend T with fun f()`
			decl, vis := funDecl(t), ""
			for _, w := range []string{"public ", "internal "} {
				if strings.HasPrefix(decl, w) {
					decl, vis = strings.TrimPrefix(decl, w), w
				}
			}
			lines = append(lines, "  "+vis+"extend "+t.Impl.Target.String()+" with "+decl)
			if target := t.Impl.Target.String(); !seen[target] {
				seen[target] = true
				r.Extends = append(r.Extends, target)
			}
		}
		if len(lines) > 0 && strings.HasSuffix(r.Shape, "}") {
			r.Shape = strings.TrimSuffix(r.Shape, "}") + "  // extension functions (D173): methods in the files that import this module\n" + strings.Join(lines, "\n") + "\n}"
		}
	}
}
