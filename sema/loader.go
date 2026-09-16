package sema

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	Root      string
	Modules   map[string]*Module // by slash path; std modules keyed "std/<name>", dependencies "dep/<name>/<path>"
	Entry     *Module            // the root module: where a program's `main` lives; nil when the root has no sources
	Given     *Module            // the module of the path the tool was pointed at (may be Entry)
	GivenDir  string
	Manifest  *Manifest
	// NeedMain is set by the driver for build/run: a package is a program
	// only if its root module declares `fun main()`.
	NeedMain bool
	Deps      map[string]*Package
	KeyPrefix string // "" for the entry package, "dep/<name>/" for dependencies
	diags     *source.Diagnostics
	// overlay maps OverlayKey(path) to unsaved editor contents (LSP).
	overlay map[string]string
}

// readSource reads a source file, preferring an editor overlay.
func (p *Package) readSource(full string) ([]byte, error) {
	if p.overlay != nil {
		if text, ok := p.overlay[OverlayKey(full)]; ok {
			return []byte(text), nil
		}
	}
	return os.ReadFile(full)
}

// OverlayKey normalises a path for overlay lookup.
func OverlayKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		abs = strings.ToLower(abs)
	}
	return abs
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
	if modPath == "." {
		dir = p.Root
		modPath = ""
	}
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
	m := &Module{Path: p.KeyPrefix + modPath, Dir: dir, Pkg: p}
	if p.KeyPrefix != "" && modPath == "" {
		m.Path = strings.TrimSuffix(p.KeyPrefix, "/")
	}
	for _, name := range names {
		full := filepath.Join(dir, name)
		data, err := p.readSource(full)
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
	m := &Module{Path: "std/" + modPath, Dir: "<std>/" + modPath, Std: true, Pkg: p}
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
	if from != nil && from.Pkg != nil && from.Pkg != p {
		return from.Pkg.Resolve(path, span, from)
	}
	// a dependency package (M6: logical paths resolved through the manifest)
	if p.Manifest != nil {
		if depPath, ok := p.Manifest.Deps[path[0]]; ok {
			return p.resolveDep(path[0], depPath, path[1:], span)
		}
	}
	key := p.KeyPrefix + strings.Join(path, "/")
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
	m, ok := p.loadLocal(strings.Join(path, "/"))
	if !ok {
		m, ok = p.loadStd(strings.Join(path, "/"))
		if !ok {
			p.diags.Errorf(span, "unknown module '%s': no directory '%s' in the package, no dependency of that name and no such standard module", strings.Join(path, "/"), strings.Join(path, "/"))
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
			dep := m.Pkg.Resolve(path, u.Pos, m)
			if dep != nil {
				m.Deps = append(m.Deps, dep)
				if m.Uses == nil {
					m.Uses = map[*ast.UseDecl]*Module{}
				}
				m.Uses[u] = dep
			}
		}
	}
	m.state = 2
}

// LoadPackage loads the entry module and, transitively, everything it uses.
func LoadPackage(entry string, diags *source.Diagnostics) (*Package, error) {
	return LoadPackageOverlay(entry, diags, nil)
}

// LoadPackageOverlay is LoadPackage with unsaved editor buffers, keyed by
// OverlayKey, standing in for files on disk.
func LoadPackageOverlay(entry string, diags *source.Diagnostics, overlay map[string]string) (*Package, error) {
	root, entryDir, err := FindRoot(entry)
	if err != nil {
		return nil, err
	}
	p := &Package{Root: root, Modules: map[string]*Module{}, Deps: map[string]*Package{}, diags: diags, overlay: overlay}
	man, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	p.Manifest = man
	// The package's root module is the program (its `main`); the module the
	// tool was pointed at is loaded too, so a module deep in the tree can be
	// checked or edited even when nothing imports it yet.
	p.GivenDir = entryDir
	modPath := p.modulePathOf(entryDir)
	given, ok := p.loadLocal(modPath)
	if !ok {
		diags.Errorf(source.Span{}, "no .vs files in %s", entryDir)
		return p, nil
	}
	p.Modules[modPath] = given
	p.Given = given
	if modPath == "" {
		p.Entry = given
	} else if rootMod, ok := p.loadLocal(""); ok {
		p.Modules[""] = rootMod
		p.Entry = rootMod
	}
	if prelude, ok := p.loadStd("prelude"); ok {
		p.Modules[prelude.Path] = prelude
		p.loadImports(prelude)
	}
	p.loadImports(given)
	if p.Entry != nil && p.Entry != given {
		p.loadImports(p.Entry)
	}
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

// resolveDep loads a module of a path dependency, honouring its exports.
func (p *Package) resolveDep(name, depPath string, rest []string, span source.Span) *Module {
	dep, ok := p.Deps[name]
	if !ok {
		root := depPath
		if !filepath.IsAbs(root) {
			root = filepath.Join(p.Manifest.Dir, root)
		}
		root, _ = filepath.Abs(root)
		man, err := readManifest(root)
		if err != nil {
			p.diags.Errorf(span, "dependency '%s': %v", name, err)
			return nil
		}
		if man == nil {
			p.diags.Errorf(span, "dependency '%s' at %s has no veles.toml (M1)", name, root)
			return nil
		}
		dep = &Package{Root: root, Modules: p.Modules, Manifest: man, Deps: map[string]*Package{}, KeyPrefix: "dep/" + name + "/", diags: p.diags, overlay: p.overlay}
		p.Deps[name] = dep
	}
	modPath := strings.Join(rest, "/")
	key := dep.KeyPrefix + modPath
	if modPath == "" {
		key = strings.TrimSuffix(dep.KeyPrefix, "/")
	}
	if m, ok := p.Modules[key]; ok {
		if m.state == 1 {
			p.diags.Errorf(span, "import cycle through dependency '%s' (M4)", name)
			return nil
		}
		return m
	}
	if !dep.Manifest.exported(modPath) {
		p.diags.Errorf(span, "module '%s' of package '%s' is not in its exports (M5: nothing escapes a package except through the manifest's exports)", modPath, name)
		return nil
	}
	local := modPath
	if local == "" {
		local = "."
	}
	m, ok := dep.loadLocal(local)
	if !ok {
		p.diags.Errorf(span, "package '%s' has no module '%s'", name, modPath)
		return nil
	}
	p.Modules[key] = m
	dep.loadImports(m)
	return m
}

// stdModuleNames lists the standard modules embedded in the compiler.
func (p *Package) stdModuleNames() []string {
	entries, err := fs.ReadDir(std.FS, ".")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
