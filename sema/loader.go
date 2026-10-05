package sema

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

// Package is a set of modules rooted at one directory (M1). The bootstrap
// has no manifest resolver yet: the root is the nearest ancestor directory
// containing `veles.toml`, or the entry file's own directory.
type Package struct {
	Root     string
	Modules  map[string]*Module // by slash path; std modules keyed "std/<name>", dependencies "dep/<name>/<path>"
	Entry    *Module            // the root module: where a program's `main` lives; nil when the root has no sources
	Given    *Module            // the module of the path the tool was pointed at (may be Entry)
	GivenDir string
	Manifest *Manifest
	// NeedMain is set by the driver for build/run: a package is a program
	// only if its root module declares `fun main()`.
	NeedMain bool
	// ConstSteps is the step budget of one constant's evaluation (D113,
	// `--const-steps`); 0 is the default.
	ConstSteps int64
	// Script is the absolute path of a script (`.vss`): a one-file package
	// whose root module is that file alone. Sibling files are never read, no
	// manifest applies, and only standard modules can be imported.
	Script    string
	Deps      map[string]*Package
	KeyPrefix string // "" for the entry package, "dep/<name>/" for dependencies
	diags     *source.Diagnostics
	// overlay maps OverlayKey(path) to unsaved editor contents (LSP).
	overlay map[string]string
	// timings records per-module time for `--timings`; nil otherwise
	timings *Timings
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

// ScriptExt is the extension of a script: one file that is a whole program
// by itself.
const ScriptExt = ".vss"

// IsScript reports whether a path names a script file.
func IsScript(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ScriptExt)
}

// Key identifies the package for caching: its root directory, or the file
// itself for a script (several scripts may share a directory).
func (p *Package) Key() string {
	if p.Script != "" {
		return p.Script
	}
	return p.Root
}

// loadScript reads a script as the root module of its own package.
func (p *Package) loadScript(path string) (*Module, bool) {
	data, err := p.readSource(path)
	if err != nil {
		p.diags.Errorf(source.Span{}, "cannot read %s: %v", path, err)
		return nil, false
	}
	m := &Module{Path: "", Dir: filepath.Dir(path), Pkg: p}
	m.Files = append(m.Files, p.parse(m, source.NewFile(path, string(data))))
	return m, true
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
		f := p.parse(m, source.NewFile(full, string(data)))
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
		sf := source.NewFile("std/"+modPath+"/"+name, string(data))
		sf.Embedded = true
		f := p.parse(m, sf)
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
	if p.Script != "" {
		// a script has no modules of its own (dependencies: later)
		m, ok := p.loadStd(key)
		if !ok {
			p.diags.Errorf(span, "unknown module '%s': a script imports only standard modules, and there is no such standard module", key)
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
			for _, s := range u.Specs {
				var path []string
				for _, seg := range s.Path {
					path = append(path, seg.Name)
				}
				dep := m.Pkg.Resolve(path, s.Pos, m)
				if dep != nil {
					m.Deps = append(m.Deps, dep)
					if m.Uses == nil {
						m.Uses = map[*ast.UseSpec]*Module{}
					}
					m.Uses[s] = dep
				}
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
	return loadPackage(entry, diags, overlay, nil)
}

func loadPackage(entry string, diags *source.Diagnostics, overlay map[string]string, timings *Timings) (*Package, error) {
	if IsScript(entry) {
		abs, err := filepath.Abs(entry)
		if err != nil {
			return nil, err
		}
		dir := filepath.Dir(abs)
		p := &Package{Root: dir, Script: abs, GivenDir: dir, Modules: map[string]*Module{}, Deps: map[string]*Package{}, diags: diags, overlay: overlay, timings: timings}
		given, ok := p.loadScript(abs)
		if !ok {
			return p, nil
		}
		p.Modules[""] = given
		p.Given, p.Entry = given, given
		if prelude, ok := p.loadStd("prelude"); ok {
			p.Modules[prelude.Path] = prelude
			p.loadImports(prelude)
		}
		p.loadImports(given)
		return p, nil
	}
	root, entryDir, err := FindRoot(entry)
	if err != nil {
		return nil, err
	}
	p := &Package{Root: root, Modules: map[string]*Module{}, Deps: map[string]*Package{}, diags: diags, overlay: overlay, timings: timings}
	man, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	p.Manifest = man
	// The package's root module is the program (its `main`); the module the
	// tool was pointed at is loaded too, so a module deep in the tree can be
	// checked or edited even when nothing imports it yet.
	p.GivenDir = entryDir
	// The compiler's own standard library sources (`<repo>/std/<module>`)
	// are checked as the std module they are, replacing the embedded copy,
	// so that editing the prelude gives real diagnostics against the edit.
	if name, ok := stdSourceDir(entryDir); ok {
		p.Root = filepath.Dir(entryDir)
		given, ok := p.loadStdFromDisk(entryDir, name)
		if !ok {
			diags.Errorf(source.Span{}, "no .vs files in %s", entryDir)
			return p, nil
		}
		p.Modules[given.Path] = given
		p.Given = given
		if name != "prelude" {
			if prelude, ok := p.loadStd("prelude"); ok {
				p.Modules[prelude.Path] = prelude
				p.loadImports(prelude)
			}
		}
		p.loadImports(given)
		return p, nil
	}
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

// resolveDep loads a module of a path dependency, honouring what it re-exports.
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
		dep = &Package{Root: root, Modules: p.Modules, Manifest: man, Deps: map[string]*Package{}, KeyPrefix: "dep/" + name + "/", diags: p.diags, overlay: p.overlay, timings: p.timings}
		p.Deps[name] = dep
	}
	// the module the path names: the dependency's root module, then one
	// `public use` per segment (D89: nothing escapes a package except what
	// its modules re-export)
	cur := p.depModule(dep, name, "", span)
	for i, seg := range rest {
		if cur == nil {
			return nil
		}
		spec := publicUseOf(cur, seg)
		if spec == nil {
			where := "its root module"
			if i > 0 {
				where = "module '" + strings.Join(rest[:i], ".") + "'"
			}
			p.diags.Errorf(span, "module '%s' of package '%s' is not re-exported; add 'public use %s' to %s (D89: nothing escapes a package except what its modules re-export)", strings.Join(rest[:i+1], "."), name, seg, where)
			return nil
		}
		var target []string
		for _, s := range spec.Path {
			target = append(target, s.Name)
		}
		cur = p.depModule(dep, name, strings.Join(target, "/"), span)
	}
	return cur
}

// depModule loads a module of a dependency package by its path in that
// package ("" is its root module).
func (p *Package) depModule(dep *Package, name, modPath string, span source.Span) *Module {
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

// publicUseOf finds the `public use` of a whole module that a module
// exposes under `name` (its alias, else the last segment of its path).
func publicUseOf(m *Module, name string) *ast.UseSpec {
	for _, f := range m.Files {
		for _, d := range f.Decls {
			u, ok := d.(*ast.UseDecl)
			if !ok || !u.Pub {
				continue
			}
			for _, s := range u.Specs {
				if len(s.Names) > 0 || len(s.Path) == 0 {
					continue
				}
				exposed := s.Path[len(s.Path)-1].Name
				if s.Alias != nil {
					exposed = s.Alias.Name
				}
				if exposed == name {
					return s
				}
			}
		}
	}
	return nil
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

// stdSourceDir reports whether dir is one of the compiler's own standard
// library source directories — a directory named after an embedded std
// module, inside a directory named `std` — and which module it is.
func stdSourceDir(dir string) (string, bool) {
	name := filepath.Base(dir)
	if filepath.Base(filepath.Dir(dir)) != "std" {
		return "", false
	}
	if _, err := fs.ReadDir(std.FS, name); err != nil {
		return "", false
	}
	return name, true
}

// loadStdFromDisk loads a std module from a source directory instead of
// the embedded copy (see LoadPackageOverlay).
func (p *Package) loadStdFromDisk(dir, name string) (*Module, bool) {
	m, ok := p.loadLocal(name)
	if !ok {
		return nil, false
	}
	m.Path = "std/" + name
	m.Std = true
	return m, true
}

// LoadAll loads every module of the package — each directory under the
// root holding `.vs` files — whether anything imports it or not: what
// `veles doc` documents. A directory with a manifest of its own is another
// package and is skipped, as are hidden directories.
func (p *Package) LoadAll() {
	if p.Script != "" {
		return
	}
	filepath.WalkDir(p.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path != p.Root {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "veles.toml")); err == nil {
				return filepath.SkipDir
			}
		}
		rel := p.modulePathOf(path)
		if rel == "" {
			return nil // the root module is loaded already
		}
		if _, loaded := p.Modules[p.KeyPrefix+rel]; loaded {
			return nil
		}
		if m, ok := p.loadLocal(rel); ok {
			p.Modules[m.Path] = m
			p.loadImports(m)
		}
		return nil
	})
}
