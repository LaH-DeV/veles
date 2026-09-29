package lsp

import (
	"io/fs"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

// Auto-import (D85): completing a bare name a module offers and the file
// has not imported adds the import with the item — the name goes into the
// module's braces, or a new `use m { name }` line is written.

// importCandidate is a public name of a module the file could import.
type importCandidate struct {
	module string // the path written after `use`
	name   string
	kind   int
	detail string
}

// importCandidates lists the public names of every module the file might
// import: the ones the package loaded, and the standard library's, which
// are read from the embedded sources once.
func (s *Server) importCandidates(a *analysis, cur *sema.Module) []importCandidate {
	var out []importCandidate
	collect := func(module string, decls []ast.Decl) {
		for _, decl := range decls {
			if !isPub(decl) {
				continue
			}
			s.addDecl(func(label string, kind int, detail string) {
				out = append(out, importCandidate{module, label, kind, detail})
			}, decl)
		}
	}
	loaded := map[string]bool{}
	for _, m := range a.pkg.Modules {
		if m == cur || m.Path == "std/prelude" || strings.HasPrefix(m.Path, "std/prelude/") {
			continue
		}
		path := strings.ReplaceAll(strings.TrimPrefix(m.Path, "std/"), "/", ".")
		if path == "" {
			continue // the root module: nothing imports it
		}
		loaded[path] = true
		for _, f := range m.Files {
			collect(path, f.Decls)
		}
	}
	for _, sm := range s.stdCandidates() {
		if !loaded[sm.module] && (cur == nil || strings.TrimPrefix(cur.Path, "std/") != sm.module) {
			out = append(out, sm)
		}
	}
	return out
}

// stdCandidates reads the embedded standard modules' public names, once.
func (s *Server) stdCandidates() []importCandidate {
	s.stdOnce.Do(func() {
		entries, err := fs.ReadDir(std.FS, ".")
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "prelude" {
				continue
			}
			files, _ := fs.ReadDir(std.FS, e.Name())
			for _, fe := range files {
				if fe.IsDir() || !strings.HasSuffix(fe.Name(), ".vs") || strings.HasSuffix(fe.Name(), ".test.vs") {
					continue
				}
				data, err := fs.ReadFile(std.FS, e.Name()+"/"+fe.Name())
				if err != nil {
					continue
				}
				file := parser.ParseFile(source.NewFile(e.Name()+"/"+fe.Name(), string(data)), &source.Diagnostics{})
				for _, decl := range file.Decls {
					if !isPub(decl) {
						continue
					}
					mod := e.Name()
					s.addDecl(func(label string, kind int, detail string) {
						s.stdNames = append(s.stdNames, importCandidate{mod, label, kind, detail})
					}, decl)
				}
			}
		}
	})
	return s.stdNames
}

// autoImports are the completion items for names the file could import,
// each carrying the edit that imports it. Only for a typed prefix, so a
// bare cursor is not answered with every name of every module.
func (s *Server) autoImports(a *analysis, d *document, off int, seen map[string]bool) []completionItem {
	if a == nil || d == nil {
		return nil
	}
	start := off
	for start > 0 && isIdentByte(d.text[start-1]) {
		start--
	}
	prefix := strings.ToLower(d.text[start:off])
	if prefix == "" {
		return nil
	}
	cur, _ := s.moduleOf(a, d)
	file := source.NewFile(d.path, d.text)
	parsed := parser.ParseFile(file, &source.Diagnostics{})
	var items []completionItem
	taken := map[string]bool{}
	for _, c := range s.importCandidates(a, cur) {
		lower := strings.ToLower(c.name)
		if seen[c.name] || taken[c.name] || !strings.HasPrefix(lower, prefix) {
			continue
		}
		if sema.PreludeHome(c.name) != "" && sema.PreludeHome(c.name) != c.module {
			continue
		}
		taken[c.name] = true
		edit := importEdit(file, parsed, c.module, c.name)
		if edit == nil {
			continue
		}
		items = append(items, completionItem{
			Label:               c.name,
			Kind:                c.kind,
			Detail:              c.detail + "  — imports " + c.module,
			SortText:            "~" + c.name,
			AdditionalTextEdits: []lspTextEdit{*edit},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// importEdit is the edit that brings name of module into the file: into the
// braces of a `use module { … }` there, after a bare `use module`, or as a
// line of its own after the last `use`.
func importEdit(file *source.File, parsed *ast.File, module, name string) *lspTextEdit {
	insert := func(off int, text string) *lspTextEdit {
		p := offsetToPosition(file, off)
		return &lspTextEdit{Range: lspRange{Start: p, End: p}, NewText: text}
	}
	lastUseEnd := -1
	for _, decl := range parsed.Decls {
		ud, ok := decl.(*ast.UseDecl)
		if !ok {
			continue
		}
		lastUseEnd = ud.Pos.End
		for _, u := range ud.Specs {
			if pathString(u.Path) != module {
				continue
			}
			for _, n := range u.Names {
				if n.Name.Name == name {
					return nil // already imported (under another name, perhaps)
				}
			}
			if len(u.Names) > 0 {
				return insert(u.Names[len(u.Names)-1].Pos.End, ", "+name)
			}
			end := u.Path[len(u.Path)-1].Pos.End
			if u.Alias != nil {
				end = u.Alias.Pos.End
			}
			return insert(end, " { "+name+" }")
		}
	}
	if lastUseEnd >= 0 {
		return insert(lastUseEnd, "\nuse "+module+" { "+name+" }")
	}
	return insert(0, "use "+module+" { "+name+" }\n\n")
}
