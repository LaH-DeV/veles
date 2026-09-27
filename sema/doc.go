package sema

import (
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// ModuleDoc is one module's API documentation, as Markdown.
type ModuleDoc struct {
	Name string // the module as `use` spells it: "geometry", "main"
	Text string
}

// DocPackage renders the public surface of a package's own modules (not
// the standard library's, not dependencies') for `veles doc`: each
// module's documentation, then its public declarations in source order,
// each spelled as its hover spells it for a reader in another package —
// the same shapes, the same visibility rule — followed by its `///`
// comment and the comments of its documented members. Nil when the
// package does not check.
func DocPackage(pkg *Package, diags *source.Diagnostics) []ModuleDoc {
	_, c := checkCollect(pkg, diags, false, false, nil)
	if diags.HasErrors() || c == nil {
		return nil
	}
	outside := viewpoint{c: c}
	var out []ModuleDoc
	var paths []string
	exported := map[string]bool{}
	if pkg.Manifest != nil {
		for _, e := range pkg.Manifest.Exports {
			exported[e] = true
		}
	}
	for p, m := range pkg.Modules {
		if m.Std || strings.HasPrefix(p, "dep/") {
			continue
		}
		// a library's surface is its root and what the manifest exports;
		// a module it keeps to itself is not API
		if len(exported) > 0 && p != "" && !exported[p] {
			continue
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		m := pkg.Modules[p]
		head, _ := moduleHead(m)
		name := strings.TrimPrefix(head, "module ")
		if p == "" && pkg.Manifest != nil && pkg.Manifest.Name != "" {
			name = pkg.Manifest.Name // the root module is the package itself
		}
		var sb strings.Builder
		sb.WriteString("# Module `" + name + "`\n\n")
		if doc := m.Doc(); doc != "" {
			sb.WriteString(doc + "\n\n")
		}
		n := 0
		for _, f := range m.Files {
			for _, d := range f.Decls {
				if !declIsPub(d) && !isPubAlias(d) {
					continue
				}
				if entry := c.docEntry(m, d, outside); entry != "" {
					sb.WriteString(entry)
					n++
				}
			}
		}
		if n == 0 {
			sb.WriteString("_Nothing public._\n")
		}
		out = append(out, ModuleDoc{Name: name, Text: strings.TrimRight(sb.String(), "\n") + "\n"})
	}
	return out
}

func isPubAlias(d ast.Decl) bool {
	switch d := d.(type) {
	case *ast.ErrorAliasDecl:
		return d.Pub
	case *ast.TypeAliasDecl:
		return d.Pub
	}
	return false
}

// docEntry is one declaration's section: a heading, the declaration in a
// code block, its documentation, and its documented public members.
func (c *Checker) docEntry(m *Module, d ast.Decl, v viewpoint) string {
	var title, code, doc string
	var members [][2]string
	switch d := d.(type) {
	case *ast.StructDecl:
		sym := m.Scope.LookupLocal(d.Name.Name)
		if sym == nil || sym.Type == nil {
			return ""
		}
		kind := "struct"
		if d.Error {
			kind = "error"
		}
		title, doc = kind+" "+d.Name.Name, d.Doc
		code = shownShape(c.shapeFrom(sym.Type, v))
		for _, f := range d.Fields {
			if f.Pub && f.Doc != "" {
				members = append(members, [2]string{f.Name.Name, f.Doc})
			}
		}
		for _, fn := range d.Methods {
			if fn.Pub && fn.Doc != "" {
				members = append(members, [2]string{fn.Name.Name + "()", fn.Doc})
			}
		}
	case *ast.TraitDecl:
		sym := m.Scope.LookupLocal(d.Name.Name)
		if sym == nil || sym.Type == nil {
			return ""
		}
		title, doc = "trait "+d.Name.Name, d.Doc
		if d.Sealed {
			title = "sealed trait " + d.Name.Name
		}
		code = shownShape(c.shapeFrom(sym.Type, v))
		for _, fn := range d.Methods {
			if fn.Doc != "" {
				members = append(members, [2]string{fn.Name.Name + "()", fn.Doc})
			}
		}
	case *ast.EnumDecl:
		sym := m.Scope.LookupLocal(d.Name.Name)
		if sym == nil || sym.Type == nil {
			return ""
		}
		title, doc = "enum "+d.Name.Name, d.Doc
		code = shownShape(c.shapeFrom(sym.Type, v))
		for _, mem := range d.Members {
			if mem.Doc != "" {
				members = append(members, [2]string{mem.Name.Name, mem.Doc})
			}
		}
	case *ast.FunDecl:
		sym := m.Scope.LookupLocal(d.Name.Name)
		if sym == nil || sym.Func == nil {
			return ""
		}
		title, doc, code = "fun "+d.Name.Name, d.Doc, funDecl(sym.Func)
	case *ast.ValDecl:
		sym := m.Scope.LookupLocal(d.Name.Name)
		if sym == nil || sym.Global == nil {
			return ""
		}
		title, doc, code = d.Kind.String()+" "+d.Name.Name, d.Doc, c.globalDecl(d.Name.Name, sym.Global)
	case *ast.ErrorAliasDecl:
		title, doc, code = "error "+d.Name.Name, d.Doc, "public error "+d.Name.Name+" = "+ast.TypeString(d.Members)
	case *ast.TypeAliasDecl:
		title, doc, code = "type "+d.Name.Name, d.Doc, "public type "+d.Name.Name+" = "+ast.TypeString(d.Type)
	default:
		return ""
	}
	if code == "" {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## `" + title + "`\n\n```veles\n" + code + "\n```\n\n")
	if doc != "" {
		sb.WriteString(doc + "\n\n")
	}
	for _, mem := range members {
		sb.WriteString("- `" + mem[0] + "` — " + strings.ReplaceAll(mem[1], "\n", " ") + "\n")
	}
	if len(members) > 0 {
		sb.WriteString("\n")
	}
	return sb.String()
}

// shownShape drops the hover's "not visible from here" count: the reader
// of API documentation is outside by definition.
func shownShape(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if !strings.HasPrefix(strings.TrimSpace(l), "// ... and ") {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}
