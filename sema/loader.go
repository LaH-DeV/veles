package sema

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

// Package is a set of modules rooted at one directory (M1). The bootstrap
// has no manifest resolver yet: the root is the nearest ancestor directory
// containing `veles.toml`, or the entry file's own directory.
type Package struct {
	Root    string
	Modules map[string]*Module // by slash path; std modules keyed "std/<name>"
	Entry   *Module
	diags   *source.Diagnostics
}

// FindRoot locates the package root for an entry path.
func FindRoot(entry string) (root string, entryDir string, err error) {
	abs, err := filepath.Abs(entry)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		entryDir = abs
	} else {
		entryDir = filepath.Dir(abs)
	}
	for dir := entryDir; ; {
		if _, err := os.Stat(filepath.Join(dir, "veles.toml")); err == nil {
			return dir, entryDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return entryDir, entryDir, nil
}

func (p *Package) modulePathOf(dir string) string {
	rel, err := filepath.Rel(p.Root, dir)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

// loadModule parses every `.vs` file in a directory. Import cycles are
// detected by the caller through Module.state.
func (p *Package) loadLocal(modPath string) (*Module, bool) {
	dir := filepath.Join(p.Root, filepath.FromSlash(modPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".vs") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return nil, false
	}
	sort.Strings(names)
	m := &Module{Path: modPath, Dir: dir}
	for _, name := range names {
		full := filepath.Join(dir, name)
		data, err := os.ReadFile(full)
		if err != nil {
			p.diags.Errorf(source.Span{}, "cannot read %s: %v", full, err)
			continue
		}
		f := parser.ParseFile(source.NewFile(full, string(data)), p.diags)
		m.Files = append(m.Files, f)
	}
	return m, true
}

func (p *Package) loadStd(modPath string) (*Module, bool) {
	entries, err := fs.ReadDir(std.FS, modPath)
	if err != nil {
		return nil, false
	}
	m := &Module{Path: "std/" + modPath, Dir: "<std>/" + modPath, Std: true}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".vs") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		data, _ := fs.ReadFile(std.FS, modPath+"/"+name)
		f := parser.ParseFile(source.NewFile("std/"+modPath+"/"+name, string(data)), p.diags)
		m.Files = append(m.Files, f)
	}
	return m, len(names) > 0
}

// Resolve finds or loads the module for a `use` path (M3/M6). Local
// directories shadow standard modules of the same name.
func (p *Package) Resolve(path []string, span source.Span, from *Module) *Module {
	key := strings.Join(path, "/")
	if m, ok := p.Modules[key]; ok {
		if m.state == 1 {
			p.diags.Errorf(span, "import cycle: module '%s' is already being loaded (M4: the module graph must be a DAG)", key)
			return nil
		}
		return m
	}
	if m, ok := p.Modules["std/"+key]; ok {
		return m
	}
	if from != nil && from.Std {
		// std modules may only import std modules
		m, ok := p.loadStd(key)
		if !ok {
			p.diags.Errorf(span, "unknown module '%s'", key)
			return nil
		}
		p.Modules[m.Path] = m
		p.loadImports(m)
		return m
	}
	m, ok := p.loadLocal(key)
	if !ok {
		m, ok = p.loadStd(key)
		if !ok {
			p.diags.Errorf(span, "unknown module '%s': no directory '%s' in the package and no such standard module", key, key)
			return nil
		}
	}
	p.Modules[m.Path] = m
	p.loadImports(m)
	return m
}

// loadImports resolves every `use` in a module, loading dependencies.
func (p *Package) loadImports(m *Module) {
	m.state = 1
	for _, f := range m.Files {
		for _, d := range f.Decls {
			u, ok := d.(*ast.UseDecl)
			if !ok {
				continue
			}
			var path []string
			for _, seg := range u.Path {
				path = append(path, seg.Name)
			}
			dep := p.Resolve(path, u.Pos, m)
			if dep != nil {
				m.Deps = append(m.Deps, dep)
			}
		}
	}
	m.state = 2
}

// LoadPackage loads the entry module and, transitively, everything it uses.
func LoadPackage(entry string, diags *source.Diagnostics) (*Package, error) {
	root, entryDir, err := FindRoot(entry)
	if err != nil {
		return nil, err
	}
	p := &Package{Root: root, Modules: map[string]*Module{}, diags: diags}
	modPath := p.modulePathOf(entryDir)
	m, ok := p.loadLocal(modPath)
	if !ok {
		diags.Errorf(source.Span{}, "no .vs files in %s", entryDir)
		return p, nil
	}
	p.Modules[modPath] = m
	p.Entry = m
	p.loadImports(m)
	return p, nil
}

// SortedModules returns modules in dependency order (dependencies first).
func (p *Package) SortedModules() []*Module {
	var out []*Module
	seen := map[*Module]bool{}
	var visit func(m *Module)
	visit = func(m *Module) {
		if seen[m] {
			return
		}
		seen[m] = true
		for _, d := range m.Deps {
			visit(d)
		}
		out = append(out, m)
	}
	keys := make([]string, 0, len(p.Modules))
	for k := range p.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		visit(p.Modules[k])
	}
	return out
}
